package autoflags

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/leodido/autoflags/options"
	"github.com/spf13/cobra"
)

func Define(c *cobra.Command, o options.Options, exclusions ...string) {
	v := GetViper(c.Name())

	// Map flags to exclude to the current command
	ignores := map[string]string{}
	for _, flag := range exclusions {
		ignores[strings.ToLower(flag)] = c.Name()
	}

	// Define the flags from struct
	define(c, o, "", "", ignores, false, false)
	// Bind flag values to struct field values
	v.BindPFlags(c.Flags())
	// Bind environment
	bindEnv(v, c)
	// Generate the usage message
	setUsage(c)
}

func define(c *cobra.Command, o interface{}, startingGroup string, structPath string, exclusions map[string]string, defineEnv bool, mandatory bool) {
	val := getValue(o)
	// TODO: double-check this one
	// if !val.IsValid() {
	// 	val = getValue(getValuePtr(o))
	// }

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		// Ignore private fields
		if !field.CanInterface() {
			continue
		}

		f := val.Type().Field(i)
		path := ""
		if structPath == "" {
			path = strings.ToLower(f.Name)
		} else {
			path = fmt.Sprintf("%s.%s", strings.ToLower(structPath), strings.ToLower(f.Name))
		}

		if cname, ok := exclusions[strings.TrimPrefix(strings.TrimPrefix(path, "-"), "-")]; ok && c.Name() == cname {
			continue
		}

		ignore, _ := strconv.ParseBool(f.Tag.Get("flagignore"))
		if ignore {
			continue
		}

		short := f.Tag.Get("flagshort")
		alias := f.Tag.Get("flag")
		if cname, ok := exclusions[alias]; ok && c.Name() == cname {
			continue
		}
		defval := f.Tag.Get("default")
		descr := f.Tag.Get("flagdescr")
		group := f.Tag.Get("flaggroup")
		if startingGroup != "" {
			group = startingGroup
		}
		name := getName(path, alias)
		envs, defineEnv := getEnv(f, defineEnv, path, alias) // FIXME: pass down hierarchy?
		mandatory := isMandatory(f) || mandatory

		// Flags with custom definition hooks
		custom, _ := strconv.ParseBool(f.Tag.Get("flagcustom"))
		if custom && f.Type.Kind() != reflect.Struct {
			hookName := fmt.Sprintf("Define%s", f.Name)
			if structPtr := getValuePtr(o); structPtr.IsValid() {
				hookFunc := structPtr.MethodByName(hookName)
				if !hookFunc.IsValid() {
					continue
				}
				hookFunc.Call([]reflect.Value{
					getValuePtr(c),
					getValue(f.Type.String()),
					getValue(name),
					getValue(short),
					getValue(descr),
				})
				inferDecodeHooks(c, name, f.Type.String())

				goto definition_done
			}
		}

		// TODO: complete type switch with missing types
		switch f.Type.Kind() {
		case reflect.Struct:
			// NOTE > field.Interface() doesn't work because it actually returns a copy of the object wrapping the interface
			define(c, field.Addr().Interface(), group, path, exclusions, defineEnv, mandatory)

			continue

		case reflect.Bool:
			val := field.Interface().(bool)
			ref := (*bool)(unsafe.Pointer(field.UnsafeAddr()))
			c.Flags().BoolVarP(ref, name, short, val, descr)

		case reflect.String:
			val := field.Interface().(string)
			ref := (*string)(unsafe.Pointer(field.UnsafeAddr()))
			c.Flags().StringVarP(ref, name, short, val, descr)

		case reflect.Int:
			val := field.Interface().(int)
			ref := (*int)(unsafe.Pointer(field.UnsafeAddr()))
			if f.Tag.Get("type") == "count" {
				c.Flags().CountVarP(ref, name, short, descr)

				continue
			}
			c.Flags().IntVarP(ref, name, short, val, descr)

		case reflect.Uint:
			val := field.Interface().(uint)
			ref := (*uint)(unsafe.Pointer(field.UnsafeAddr()))
			c.Flags().UintVarP(ref, name, short, val, descr)

		case reflect.Uint8:
			val := field.Interface().(uint8)
			ref := (*uint8)(unsafe.Pointer(field.UnsafeAddr()))
			c.Flags().Uint8VarP(ref, name, short, val, descr)

		case reflect.Slice:
			if f.Type.Elem().Kind() == reflect.String {
				val := field.Interface().([]string)
				ref := (*[]string)(unsafe.Pointer(field.UnsafeAddr()))
				c.Flags().StringSliceVarP(ref, name, short, val, descr)
				inferDecodeHooks(c, name, f.Type.String())
			}

		case reflect.Int64:
			switch f.Type.String() {
			case "int64":
				val := field.Interface().(int64)
				ref := (*int64)(unsafe.Pointer(field.UnsafeAddr()))
				c.Flags().Int64VarP(ref, name, short, val, descr)

			case "time.Duration":
				val := field.Interface().(time.Duration)
				ref := (*time.Duration)(unsafe.Pointer(field.UnsafeAddr()))
				c.Flags().DurationVarP(ref, name, short, val, descr)

			default:
				continue
			}

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
			vipers[c.Name()].SetDefault(name, defval)
			// This is needed for the usage help messages
			c.Flags().Lookup(name).DefValue = defval
		}

		if alias != "" && path != alias {
			// Alias the actual path to the flag name (ie., the alias when not empty)
			vipers[c.Name()].RegisterAlias(path, alias)
		}

		if len(envs) > 0 {
			_ = c.Flags().SetAnnotation(name, FlagEnvsAnnotation, envs)
		}

		// Set the group annotation on the current flag
		if group != "" {
			_ = c.Flags().SetAnnotation(name, FlagGroupAnnotation, []string{group})
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

func getValuePtr(o interface{}) reflect.Value {
	val := reflect.ValueOf(o)
	if val.Type().Kind() == reflect.Ptr {
		return val
	}

	return reflect.New(reflect.TypeOf(o))
}
