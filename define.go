package autoflags

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// DefineOption configures the Define function behavior
type DefineOption func(*defineContext)

// defineContext holds configuration for the Define function
type defineContext struct {
	validation         bool
	rawExclusions      []string
	usePersistentFlags bool

	targetC     *cobra.Command
	targetF     *pflag.FlagSet
	targetV     *viper.Viper
	isGlobalV   bool
	scope       *scope
	ignoreFlagC map[string]string
}

// WithValidation enables strict validation of struct tags
func WithValidation() DefineOption {
	return func(cfg *defineContext) {
		cfg.validation = true
	}
}

// WithExclusions sets flags to exclude from definition
func WithExclusions(exclusions ...string) DefineOption {
	return func(cfg *defineContext) {
		if cfg.rawExclusions == nil {
			cfg.rawExclusions = []string{}
		}
		cfg.rawExclusions = append(cfg.rawExclusions, exclusions...)
	}
}

// WithPersistentFlags instructs Define to register flags as persistent flags on the command they are defined for.
func WithPersistentFlags() DefineOption {
	return func(cfg *defineContext) {
		cfg.usePersistentFlags = true
	}
}

// Define creates flags from struct tags
func Define(c *cobra.Command, o Options, defineOpts ...DefineOption) error {
	runCtx := &defineContext{
		targetC: c,
	}
	// Apply user options
	for _, opt := range defineOpts {
		opt(runCtx)
	}

	// Map flags to exclude for the current command
	if len(runCtx.rawExclusions) > 0 {
		runCtx.ignoreFlagC = make(map[string]string)
		for _, flagStr := range runCtx.rawExclusions {
			runCtx.ignoreFlagC[strings.ToLower(flagStr)] = runCtx.targetC.Name()
		}
	}

	// Possibly run validation
	if runCtx.validation {
		if err := validateStructTags(o); err != nil {
			return err
		}
	}

	// Determine the target flag set
	if runCtx.usePersistentFlags {
		runCtx.targetF = c.PersistentFlags()
	} else {
		runCtx.targetF = c.Flags()
	}

	// Determine the target viper instance
	isRootC := c.Parent() == nil
	if runCtx.usePersistentFlags && isRootC {
		runCtx.targetV = viper.GetViper() // Viper global singleton
		runCtx.isGlobalV = true
	} else {
		runCtx.targetV = GetViper(c) // Viper specific for the target command
		runCtx.isGlobalV = false
	}

	// Obtain scope for the target command
	runCtx.scope = getScope(c)

	// Define the flags from struct
	runCtx.process(o, "", "", false, false)
	runCtx.bind()

	// Generate the usage message
	setUsage(c)

	return nil
}

func (ctx *defineContext) bind() {
	// Bind flag values to struct field values
	ctx.targetV.BindPFlags(ctx.targetF)
	// Bind environment
	ctx.bindEnvironmentVariables()
}

func (ctx *defineContext) process(
	currentOptions interface{},
	currentStartingGroup string,
	currentStructPath string,
	shouldDefineEnv bool,
	mandatory bool,
) {
	val := getValue(currentOptions)
	if !val.IsValid() {
		val = getValue(getValuePtr(currentOptions).Interface())
	}

	for i := range val.NumField() {
		field := val.Field(i)
		// Ignore private fields
		if !field.CanInterface() {
			continue
		}

		f := val.Type().Field(i)
		pathSegment := strings.ToLower(f.Name)
		fieldFullPath := ""
		if currentStructPath == "" {
			fieldFullPath = pathSegment
		} else {
			fieldFullPath = fmt.Sprintf("%s.%s", currentStructPath, pathSegment)
		}

		if cname, ok := ctx.ignoreFlagC[strings.TrimPrefix(strings.TrimPrefix(fieldFullPath, "-"), "-")]; ok && ctx.targetC.Name() == cname {
			continue
		}

		alias := f.Tag.Get("flag")
		if cname, ok := ctx.ignoreFlagC[strings.ToLower(alias)]; ok && ctx.targetC.Name() == cname {
			continue
		}

		ignore, _ := strconv.ParseBool(f.Tag.Get("flagignore"))
		if ignore {
			continue
		}

		short := f.Tag.Get("flagshort")
		defval := f.Tag.Get("default")
		descr := f.Tag.Get("flagdescr")
		group := f.Tag.Get("flaggroup")
		if currentStartingGroup != "" {
			group = currentStartingGroup
		}
		name := getName(fieldFullPath, alias)

		// Determine whether to represent hierarchy with the command name
		// We assume that options that are not common options are subcommand-specific options
		cName := ""
		if _, isCommonOptions := currentOptions.(CommonOptions); !isCommonOptions && !ctx.isGlobalV {
			cName = ctx.targetC.Name()
		}

		envs, shouldDefineEnv := getEnv(f, shouldDefineEnv, fieldFullPath, alias, cName)
		mandatory := isMandatory(f) || mandatory

		// Flags with custom definition hooks
		custom, _ := strconv.ParseBool(f.Tag.Get("flagcustom"))
		if custom && f.Type.Kind() != reflect.Struct {
			hookName := fmt.Sprintf("Define%s", f.Name)
			if structPtr := getValuePtr(currentOptions); structPtr.IsValid() {
				hookFunc := structPtr.MethodByName(hookName)
				if hookFunc.IsValid() {
					hookFunc.Call([]reflect.Value{
						getValuePtr(ctx.targetC),
						getValue(f.Type.String()),
						getValue(name),
						getValue(short),
						getValue(descr),
					})
					ctx.decodeHookFromRegistry(name, f.Type.String())

					goto definition_done
				} else {
					// Fallback to define hooks registry
					if ctx.defineHookFromRegistry(f.Type.String(), f, name, short, descr, field) {
						ctx.decodeHookFromRegistry(name, f.Type.String())

						goto definition_done
					}

					// Neither user method nor registry can handle this: skip it
					continue
				}
			}
		}

		if !field.CanAddr() {
			continue
		}

		// TODO: complete type switch with missing types
		switch f.Type.Kind() {
		case reflect.Struct:
			// NOTE > field.Interface() doesn't work because it actually returns a copy of the object wrapping the interface
			ctx.process(field.Addr().Interface(), group, fieldFullPath, shouldDefineEnv, mandatory)

			continue

		case reflect.Bool:
			val := field.Interface().(bool)
			ref := (*bool)(unsafe.Pointer(field.UnsafeAddr()))
			ctx.targetF.BoolVarP(ref, name, short, val, descr)

		case reflect.String:
			val := field.Interface().(string)
			ref := (*string)(unsafe.Pointer(field.UnsafeAddr()))
			ctx.targetF.StringVarP(ref, name, short, val, descr)

		case reflect.Int:
			val := field.Interface().(int)
			ref := (*int)(unsafe.Pointer(field.UnsafeAddr()))
			if f.Tag.Get("flagtype") == "count" {
				ctx.targetF.CountVarP(ref, name, short, descr)

				continue
			}
			ctx.targetF.IntVarP(ref, name, short, val, descr)

		case reflect.Uint:
			val := field.Interface().(uint)
			ref := (*uint)(unsafe.Pointer(field.UnsafeAddr()))
			ctx.targetF.UintVarP(ref, name, short, val, descr)

		case reflect.Uint8:
			val := field.Interface().(uint8)
			ref := (*uint8)(unsafe.Pointer(field.UnsafeAddr()))
			ctx.targetF.Uint8VarP(ref, name, short, val, descr)

		case reflect.Uint16:
			val := field.Interface().(uint16)
			ref := (*uint16)(unsafe.Pointer(field.UnsafeAddr()))
			ctx.targetF.Uint16VarP(ref, name, short, val, descr)

		case reflect.Uint32:
			val := field.Interface().(uint32)
			ref := (*uint32)(unsafe.Pointer(field.UnsafeAddr()))
			ctx.targetF.Uint32VarP(ref, name, short, val, descr)

		case reflect.Uint64:
			val := field.Interface().(uint64)
			ref := (*uint64)(unsafe.Pointer(field.UnsafeAddr()))
			ctx.targetF.Uint64VarP(ref, name, short, val, descr)

		case reflect.Slice:
			switch f.Type.Elem().Kind() {
			case reflect.String:
				val := field.Interface().([]string)
				ref := (*[]string)(unsafe.Pointer(field.UnsafeAddr()))
				ctx.targetF.StringSliceVarP(ref, name, short, val, descr)
			case reflect.Int:
				val := field.Interface().([]int)
				ref := (*[]int)(unsafe.Pointer(field.UnsafeAddr()))
				ctx.targetF.IntSliceVarP(ref, name, short, val, descr)
			}
			ctx.decodeHookFromRegistry(name, f.Type.String())

		case reflect.Int64:
			switch f.Type.String() {
			case "int64":
				val := field.Interface().(int64)
				ref := (*int64)(unsafe.Pointer(field.UnsafeAddr()))
				ctx.targetF.Int64VarP(ref, name, short, val, descr)

			case "time.Duration":
				val := field.Interface().(time.Duration)
				ref := (*time.Duration)(unsafe.Pointer(field.UnsafeAddr()))
				ctx.targetF.DurationVarP(ref, name, short, val, descr)
				ctx.decodeHookFromRegistry(name, f.Type.String())

			default:
				continue
			}

		case reflect.Int8:
			val := field.Interface().(int8)
			ref := (*int8)(unsafe.Pointer(field.UnsafeAddr()))
			ctx.targetF.Int8VarP(ref, name, short, val, descr)

		case reflect.Int16:
			val := field.Interface().(int16)
			ref := (*int16)(unsafe.Pointer(field.UnsafeAddr()))
			ctx.targetF.Int16VarP(ref, name, short, val, descr)

		case reflect.Int32:
			val := field.Interface().(int32)
			ref := (*int32)(unsafe.Pointer(field.UnsafeAddr()))
			ctx.targetF.Int32VarP(ref, name, short, val, descr)

		default:
			continue
		}

	definition_done:

		// Marking the flag
		if mandatory {
			cobra.MarkFlagRequired(ctx.targetF, name)
		}

		// Set the defaults
		if defval != "" && f.Tag.Get("flagtype") != "count" {
			ctx.targetV.SetDefault(name, defval)
			// This is needed for the usage help messages
			ctx.targetF.Lookup(name).DefValue = defval
		}

		if alias != "" && name == alias && fieldFullPath != alias {
			// Alias the actual path to the flag name (ie., the alias when not empty)
			ctx.targetV.RegisterAlias(fieldFullPath, alias)
		}

		// Annotate
		pFlag := ctx.targetF.Lookup(name)
		if pFlag != nil {
			if len(envs) > 0 {
				_ = ctx.targetF.SetAnnotation(name, FlagEnvsAnnotation, envs)
			}

			if group != "" {
				_ = ctx.targetF.SetAnnotation(name, FlagGroupAnnotation, []string{group})
			}
		}
	}
}

func getName(name, alias string) string {
	res := name
	if alias != "" {
		res = alias
	}

	return res
}

func getValue(o interface{}) reflect.Value {
	var ptr reflect.Value
	var val reflect.Value

	val = reflect.ValueOf(o)
	// When we get a pointer, we want to get the value pointed to.
	// Otherwise, we need to get a pointer to the value we got.
	if val.Type().Kind() == reflect.Ptr {
		ptr = val
		val = ptr.Elem()
	} else {
		ptr = reflect.New(reflect.TypeOf(o))
		temp := ptr.Elem()
		temp.Set(val)
		val = temp
	}

	return val
}

func getValuePtr(o any) reflect.Value {
	val := reflect.ValueOf(o)
	if val.Type().Kind() == reflect.Ptr {
		// Create a new zero-valued instance of the pointed-to type
		if val.IsNil() {
			return reflect.New(val.Type().Elem())
		}

		return val
	}

	return reflect.New(reflect.TypeOf(o))
}

// validateBooleanTag validates that a struct tag contains a valid boolean value
func validateBooleanTag(fieldName, tagName, tagValue string) error {
	if tagValue != "" {
		if _, err := strconv.ParseBool(tagValue); err != nil {
			return &FieldError{
				FieldName: fieldName,
				TagName:   tagName,
				TagValue:  tagValue,
				Message:   "invalid boolean value",
			}
		}
	}
	return nil
}

// validateStructTags checks for invalid boolean values in struct tags
func validateStructTags(o interface{}) error {
	val := getValue(o)
	if !val.IsValid() {
		val = getValue(getValuePtr(o).Interface())
	}

	return validateFieldTags(val, "")
}

// validateFieldTags recursively validates tags in struct fields
func validateFieldTags(val reflect.Value, prefix string) error {
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := val.Type().Field(i)

		// Skip private fields
		if !field.CanInterface() {
			continue
		}

		fieldName := fieldType.Name
		if prefix != "" {
			fieldName = prefix + "." + fieldName
		}

		// Validate flagcustom tag
		if err := validateBooleanTag(fieldName, "flagcustom", fieldType.Tag.Get("flagcustom")); err != nil {
			return err
		}

		// Validate flagenv tag
		if err := validateBooleanTag(fieldName, "flagenv", fieldType.Tag.Get("flagenv")); err != nil {
			return err
		}

		// Validate flagignore tag
		if err := validateBooleanTag(fieldName, "flagignore", fieldType.Tag.Get("flagignore")); err != nil {
			return err
		}

		// Validate flagrequired tag
		if err := validateBooleanTag(fieldName, "flagrequired", fieldType.Tag.Get("flagrequired")); err != nil {
			return err
		}

		// TODO: check is an integer type when "flagtype" is "count"

		// Recursively validate children structs
		if fieldType.Type.Kind() == reflect.Struct {
			if err := validateFieldTags(field, fieldName); err != nil {
				return err
			}
		}
	}

	return nil
}
