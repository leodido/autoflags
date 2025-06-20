package autoflags

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	internalenv "github.com/leodido/autoflags/internal/env"
	internalhooks "github.com/leodido/autoflags/internal/hooks"
	internalpath "github.com/leodido/autoflags/internal/path"
	internalreflect "github.com/leodido/autoflags/internal/reflect"
	internaltag "github.com/leodido/autoflags/internal/tag"
	internalusage "github.com/leodido/autoflags/internal/usage"
	internalvalidation "github.com/leodido/autoflags/internal/validation"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// DefineOption configures the behavior of the Define function.
type DefineOption func(*defineContext)

// defineContext holds context for the definition of the options
type defineContext struct {
	exclusions map[string]string
	comm       *cobra.Command
}

// globalAliasCache stores the mapping of a struct field's path to its `flag` tag alias.
var globalAliasCache = &sync.Map{}

// globalDefaultsCache stores the mapping of a default to its `flag` tag alias.
var globalDefaultsCache = &sync.Map{}

// WithExclusions sets flags to exclude from definition based on flag names or paths.
//
// Exclusions are case-insensitive and apply only to the specific command.
func WithExclusions(exclusions ...string) DefineOption {
	return func(cfg *defineContext) {
		if cfg.exclusions == nil {
			cfg.exclusions = make(map[string]string)
		}
		// Map exclusions to the command name
		for _, flag := range exclusions {
			cfg.exclusions[strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(flag), "-"), "-")] = cfg.comm.Name()
		}
	}
}

// Define creates flags from struct field tags and binds them to the command.
//
// It processes struct tags to generate appropriate cobra flags, handles environment
// variable binding, sets up flag groups, and configures the usage template.
func Define(c *cobra.Command, o Options, defineOpts ...DefineOption) error {
	ctx := &defineContext{
		comm: c,
	}

	// Apply configuration options
	for _, opt := range defineOpts {
		opt(ctx)
	}

	// Run input validation (on by default)
	if err := internalvalidation.Struct(c, o); err != nil {
		return err
	}

	v := GetViper(c)

	// Define the flags from struct
	if err := define(c, o, "", "", ctx.exclusions, false, false); err != nil {
		return err
	}
	// Bind flag values to struct field values
	v.BindPFlags(c.Flags())
	// Bind environment
	internalenv.BindEnv(c)
	// Generate the usage message
	SetupUsage(c)

	return nil
}

func define(c *cobra.Command, o any, startingGroup string, structPath string, exclusions map[string]string, defineEnv bool, mandatory bool) error {
	// Assuming validation already caught untyped nils...
	val := internalreflect.GetValue(o)
	if !val.IsValid() {
		val = internalreflect.GetValue(internalreflect.GetValuePtr(o).Interface())
	}

	for i := range val.NumField() {
		field := val.Field(i)
		// Ignore private fields
		if !field.CanInterface() {
			continue
		}

		// Only if the field is addressable
		if !field.CanAddr() {
			continue
		}

		f := val.Type().Field(i)
		path := internalpath.GetFieldPath(structPath, f)

		// Check exclusions for struct path with command name validation (case-insensitive)
		if cname, ok := exclusions[path]; ok && c.Name() == cname {
			continue
		}

		// Check exclusions for alias with command name validation (case-insensitive)
		alias := f.Tag.Get("flag")
		if alias != "" {
			if cname, ok := exclusions[strings.ToLower(alias)]; ok && c.Name() == cname {
				continue
			}
		}

		ignore, _ := strconv.ParseBool(f.Tag.Get("flagignore"))
		if ignore {
			continue
		}

		short := f.Tag.Get("flagshort")
		defval := f.Tag.Get("default")
		descr := f.Tag.Get("flagdescr")
		group := f.Tag.Get("flaggroup")
		if startingGroup != "" {
			group = startingGroup
		}
		name := internalpath.GetName(path, alias)

		// Determine whether to represent hierarchy with the command name
		// We assume that options that are not context options are subcommand-specific options
		cName := ""
		if _, isContextOptions := o.(ContextOptions); !isContextOptions {
			cName = c.Name()
		}

		envs, defineEnv := internalenv.GetEnv(f, defineEnv, path, alias, cName)
		mandatory := internaltag.IsMandatory(f) || mandatory

		kind := f.Type.Kind()

		// Flags with `flagcustom:"true"` tag (validation already done)
		custom, _ := strconv.ParseBool(f.Tag.Get("flagcustom"))
		if custom && kind != reflect.Struct {
			defineHookName := fmt.Sprintf("Define%s", f.Name)
			decodeHookName := fmt.Sprintf("Decode%s", f.Name)

			if structPtr := internalreflect.GetValuePtr(o); structPtr.IsValid() {
				defineHookFunc := structPtr.MethodByName(defineHookName)
				decodeHookFunc := structPtr.MethodByName(decodeHookName)

				if defineHookFunc.IsValid() {
					// Call user's define hook
					results := defineHookFunc.Call([]reflect.Value{
						reflect.ValueOf(name),
						reflect.ValueOf(short),
						reflect.ValueOf(descr),
						reflect.ValueOf(f),
						reflect.ValueOf(field),
					})

					returnedValue := results[0].Interface().(pflag.Value)
					returnedUsage := results[1].Interface().(string)
					c.Flags().VarP(returnedValue, name, short, returnedUsage)

					// Register user's decode hook (`Unmarshal` will call it)
					if err := internalhooks.StoreDecodeHookFunc(c, name, decodeHookFunc, f.Type); err != nil {
						return fmt.Errorf("couldn't register decode hook %s: %w", decodeHookName, err)
					}

					goto definition_done
				}
				// The users set `flagcustom:"true"` but they didn't define a custom define hook
				// We fallback to look up the hooks registries to avoid erroring out
				if internalhooks.InferDefineHooks(c, name, short, descr, f, field) {
					internalhooks.InferDecodeHooks(c, name, f.Type.String())

					goto definition_done
				}

				// This should never happen since validation would have caught missing hooks
				return fmt.Errorf("internal error: custom flag %s passed validation but no hooks found", f.Name)
			}
		}

		// Check registry for known custom types
		if internalhooks.InferDefineHooks(c, name, short, descr, f, field) {
			if !internalhooks.InferDecodeHooks(c, name, f.Type.String()) {
				return fmt.Errorf("internal error: missing decode hook for built-in type %s", f.Type.String())
			}

			goto definition_done
		}

		// Skip custom types that aren't in registry
		if !internaltag.IsStandardType(f.Type) && kind != reflect.Struct && kind != reflect.Slice {
			continue
		}

		if c.Flags().Lookup(name) != nil {
			goto definition_done
		}

		// TODO: complete type switch with missing types for:
		// c.Flags().StringArrayVarP()
		// c.Flags().IPSliceVarP()
		// c.Flags().DurationSliceVarP()
		// c.Flags().BoolSliceVarP()
		// c.Flags().UintSliceVarP()
		// c.Flags().BytesBase64VarP()
		// c.Flags().BytesHexVarP()
		// c.Flags().IPMaskVarP()
		// c.Flags().IPNetVarP()
		// c.Flags().IPVarP()
		// c.Flags().StringToStringVarP()
		// c.Flags().StringToInt64VarP()
		// c.Flags().StringToIntVarP()
		switch kind {
		case reflect.Struct:
			// NOTE > field.Interface() doesn't work because it actually returns a copy of the object wrapping the interface
			if err := define(c, field.Addr().Interface(), group, path, exclusions, defineEnv, mandatory); err != nil {
				return err
			}

			continue

		case reflect.Bool:
			val := field.Interface().(bool)
			ref := (*bool)(unsafe.Pointer(field.UnsafeAddr()))
			c.Flags().BoolVarP(ref, name, short, val, descr)

		case reflect.String:
			val := field.Interface().(string)
			ref := (*string)(unsafe.Pointer(field.UnsafeAddr()))
			c.Flags().StringVarP(ref, name, short, val, descr)

		case reflect.Uint:
			val := field.Interface().(uint)
			ref := (*uint)(unsafe.Pointer(field.UnsafeAddr()))
			c.Flags().UintVarP(ref, name, short, val, descr)

		case reflect.Uint8:
			val := field.Interface().(uint8)
			ref := (*uint8)(unsafe.Pointer(field.UnsafeAddr()))
			c.Flags().Uint8VarP(ref, name, short, val, descr)

		case reflect.Uint16:
			val := field.Interface().(uint16)
			ref := (*uint16)(unsafe.Pointer(field.UnsafeAddr()))
			c.Flags().Uint16VarP(ref, name, short, val, descr)

		case reflect.Uint32:
			val := field.Interface().(uint32)
			ref := (*uint32)(unsafe.Pointer(field.UnsafeAddr()))
			c.Flags().Uint32VarP(ref, name, short, val, descr)

		case reflect.Uint64:
			val := field.Interface().(uint64)
			ref := (*uint64)(unsafe.Pointer(field.UnsafeAddr()))
			c.Flags().Uint64VarP(ref, name, short, val, descr)

		case reflect.Int:
			val := field.Interface().(int)
			ref := (*int)(unsafe.Pointer(field.UnsafeAddr()))
			if f.Tag.Get("flagtype") == "count" {
				c.Flags().CountVarP(ref, name, short, descr)

				goto definition_done
			}
			c.Flags().IntVarP(ref, name, short, val, descr)

		case reflect.Int8:
			val := field.Interface().(int8)
			ref := (*int8)(unsafe.Pointer(field.UnsafeAddr()))
			c.Flags().Int8VarP(ref, name, short, val, descr)

		case reflect.Int16:
			val := field.Interface().(int16)
			ref := (*int16)(unsafe.Pointer(field.UnsafeAddr()))
			c.Flags().Int16VarP(ref, name, short, val, descr)

		case reflect.Int32:
			val := field.Interface().(int32)
			ref := (*int32)(unsafe.Pointer(field.UnsafeAddr()))
			c.Flags().Int32VarP(ref, name, short, val, descr)

		case reflect.Int64:
			val := field.Interface().(int64)
			ref := (*int64)(unsafe.Pointer(field.UnsafeAddr()))
			c.Flags().Int64VarP(ref, name, short, val, descr)

		case reflect.Slice:
			switch f.Type.Elem().Kind() {
			case reflect.String:
				val := field.Interface().([]string)
				ref := (*[]string)(unsafe.Pointer(field.UnsafeAddr()))
				c.Flags().StringSliceVarP(ref, name, short, val, descr)
			case reflect.Int:
				val := field.Interface().([]int)
				ref := (*[]int)(unsafe.Pointer(field.UnsafeAddr()))
				c.Flags().IntSliceVarP(ref, name, short, val, descr)
			}
			internalhooks.InferDecodeHooks(c, name, f.Type.String()) // FIXME: handle error?

		default:
			continue
		}

	definition_done:

		// Marking the flag
		if mandatory {
			c.MarkFlagRequired(name)
		}

		// Set the defaults
		if defval != "" {
			GetViper(c).SetDefault(name, defval)
			GetViper(c).SetDefault(path, defval)
			// This is needed for the usage help messages
			c.Flags().Lookup(name).DefValue = defval
			globalDefaultsCache.Store(name, defval)
		}

		if alias != "" && path != alias {
			globalAliasCache.Store(alias, path)
		}

		if len(envs) > 0 {
			_ = c.Flags().SetAnnotation(name, internalenv.FlagAnnotation, envs)
		}

		// Set the group annotation on the current flag
		if group != "" {
			_ = c.Flags().SetAnnotation(name, internalusage.FlagGroupAnnotation, []string{group})
		}
	}

	return nil
}

func ResetGlobals() {
	globalAliasCache = &sync.Map{}
	globalDefaultsCache = &sync.Map{}
}
