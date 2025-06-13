package autoflags

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

	autoflagserrors "github.com/leodido/autoflags/errors"
	"github.com/spf13/cobra"
)

// DefineOption configures the behavior of the Define function.
type DefineOption func(*defineContext)

// defineContext holds context for the definition of the options
type defineContext struct {
	exclusions map[string]string
	comm       *cobra.Command
}

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
	if err := validateStruct(o); err != nil {
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
	bindEnv(c)
	// Generate the usage message
	SetupUsage(c)

	return nil
}

func define(c *cobra.Command, o any, startingGroup string, structPath string, exclusions map[string]string, defineEnv bool, mandatory bool) error {
	// Assuming validation already caught untyped nils...
	val := getValue(o)
	if !val.IsValid() {
		val = getValue(getValuePtr(o).Interface())
	}

	for i := range val.NumField() {
		field := val.Field(i)
		// Ignore private fields
		if !field.CanInterface() {
			continue
		}

		if !field.CanAddr() {
			continue
		}

		f := val.Type().Field(i)
		path := ""
		if structPath == "" {
			path = strings.ToLower(f.Name)
		} else {
			path = fmt.Sprintf("%s.%s", strings.ToLower(structPath), strings.ToLower(f.Name))
		}

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
		name := getName(path, alias)

		// Determine whether to represent hierarchy with the command name
		// We assume that options that are not context options are subcommand-specific options
		cName := ""
		if _, isContextOptions := o.(ContextOptions); !isContextOptions {
			cName = c.Name()
		}

		envs, defineEnv := getEnv(f, defineEnv, path, alias, cName)
		mandatory := isMandatory(f) || mandatory

		kind := f.Type.Kind()

		// Flags with `flagcustom:"true"` tag (validation already done)
		custom, _ := strconv.ParseBool(f.Tag.Get("flagcustom"))
		if custom && kind != reflect.Struct {
			defineHookName := fmt.Sprintf("Define%s", f.Name)
			decodeHookName := fmt.Sprintf("Decode%s", f.Name)

			if structPtr := getValuePtr(o); structPtr.IsValid() {
				defineHookFunc := structPtr.MethodByName(defineHookName)
				decodeHookFunc := structPtr.MethodByName(decodeHookName)

				if defineHookFunc.IsValid() {
					// Call user's define hook
					defineHookFunc.Call([]reflect.Value{
						reflect.ValueOf(c),
						reflect.ValueOf(name),
						reflect.ValueOf(short),
						reflect.ValueOf(descr),
						reflect.ValueOf(f),
						reflect.ValueOf(field),
					})

					// FIXME: here we can verify the DefineX hook actually created a flag
					// FIXME: it's probably better to change the signature of define hooks by requiring return types to use here to create the flag with c.Flags().VarP()

					// Register user's decode hook (`Unmarshal` will call it)
					if err := storeDecodeHookFunc(c, name, decodeHookFunc, f.Type); err != nil {
						return fmt.Errorf("couldn't register decode hook %s: %w", decodeHookName, err)
					}

					goto definition_done
				}
				// The users set `flagcustom:"true"` but they didn't define a custom define hook
				// We fallback to look up the hooks registries to avoid erroring out
				if inferDefineHooks(c, name, short, descr, f, field) {
					inferDecodeHooks(c, name, f.Type.String())

					goto definition_done
				}

				// This should never happen since validation would have caught missing hooks
				return fmt.Errorf("internal error: custom flag %s passed validation but no hooks found", f.Name)
			}
		}

		// Check registry for known custom types
		if inferDefineHooks(c, name, short, descr, f, field) {
			if !inferDecodeHooks(c, name, f.Type.String()) {
				return fmt.Errorf("internal error: missing decode hook for built-in type %s", f.Type.String())
			}

			goto definition_done
		}

		// Skip custom types that aren't in registry
		if !isStandardType(f.Type) && kind != reflect.Struct && kind != reflect.Slice {
			continue
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
			inferDecodeHooks(c, name, f.Type.String()) // FIXME: handle error?

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
		}

		if alias != "" && path != alias {
			// Make the field name (path) an alias for the flag name (alias)
			// Allows mapstructure to find values provided via the flag tag name in the config files
			GetViper(c).RegisterAlias(path, alias)

		}

		if len(envs) > 0 {
			_ = c.Flags().SetAnnotation(name, flagEnvsAnnotation, envs)
		}

		// Set the group annotation on the current flag
		if group != "" {
			_ = c.Flags().SetAnnotation(name, flagGroupAnnotation, []string{group})
		}
	}

	return nil
}

func getName(name, alias string) string {
	res := name
	if alias != "" {
		res = alias
	}

	return res
}

func getValue(o any) reflect.Value {
	var ptr reflect.Value
	var val reflect.Value

	val = reflect.ValueOf(o)
	// Check if the value is valid before trying to access its type, otherwise let the caller handle it
	if !val.IsValid() {
		return val
	}
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
	// Check if the value is valid before trying to access its type, otherwise let the caller handle it
	if !val.IsValid() {
		return val
	}
	if val.Type().Kind() == reflect.Ptr {
		// Create a new zero-valued instance of the pointed-to type
		if val.IsNil() {
			return reflect.New(val.Type().Elem())
		}

		return val
	}

	return reflect.New(reflect.TypeOf(o))
}

// getStructPtr is a helper that gets a pointer to a struct value.
//
// Similar to getValuePtr but works with reflect.Value directly.
func getStructPtr(structValue reflect.Value) reflect.Value {
	if !structValue.IsValid() {
		return reflect.Value{}
	}

	// If it's already a pointer, handle appropriately
	if structValue.Type().Kind() == reflect.Ptr {
		if structValue.IsNil() {
			// Create new instance of the pointed-to type
			return reflect.New(structValue.Type().Elem())
		}

		return structValue
	}

	// For non-pointer values, try to get address if possible
	if structValue.CanAddr() {
		return structValue.Addr()
	}

	// Create a pointer to a copy if we can't get address
	newPtr := reflect.New(structValue.Type())
	newPtr.Elem().Set(structValue)

	return newPtr
}

// getValidValue attempts to get a valid reflect.Value from the input object.
//
// It handles untyped nil as an error (no type information available).
// For typed nil pointers, it uses a fallback approach to create zero values.
//
// Returns an error if no valid Value can be obtained.
func getValidValue(o any) (reflect.Value, error) {
	// Handle untyped nil
	if o == nil {
		return reflect.Value{}, autoflagserrors.NewInputError("nil", "cannot define flags from nil value")
	}

	val := getValue(o)
	if !val.IsValid() {
		// Try the fallback approach for cases like typed nil pointers
		// This allows us to create zero values from type information
		valPtr := getValuePtr(o)
		if !valPtr.IsValid() {
			// This should not happen for valid typed inputs
			inputType := fmt.Sprintf("%T", o)

			return reflect.Value{}, autoflagserrors.NewInputError(inputType, "cannot obtain valid reflection value")
		}

		// Only call Interface() if we have a valid value
		val = getValue(valPtr.Interface())
		if !val.IsValid() {
			// This should also not happen for valid inputs
			inputType := fmt.Sprintf("%T", o)

			return reflect.Value{}, autoflagserrors.NewInputError(inputType, "fallback reflection approach failed")
		}
	}

	return val, nil
}

func getFieldName(prefix string, structField reflect.StructField) string {
	if prefix == "" {
		return structField.Name
	}
	return prefix + "." + structField.Name
}

// validateStruct checks the coherence of definitions in the given struct
func validateStruct(o any) error {
	val, err := getValidValue(o)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	typeToFields := make(map[reflect.Type][]string)
	if err := validateFields(val, "", typeToFields); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	for fieldType, fieldNames := range typeToFields {
		if len(fieldNames) > 1 {
			return autoflagserrors.NewConflictingTypeError(fieldType, fieldNames, "create distinct custom types for each field")
		}
	}

	return nil
}

// validateFields recursively validates the struct fields
func validateFields(val reflect.Value, prefix string, typeToFields map[reflect.Type][]string) error {
	for i := range val.NumField() {
		field := val.Field(i)
		structF := val.Type().Field(i)

		// Skip private fields
		if !field.CanInterface() {
			continue
		}

		fieldName := getFieldName(prefix, structF)
		isStructKind := structF.Type.Kind() == reflect.Struct

		// Validate flagshort tag
		short := structF.Tag.Get("flagshort")
		if short != "" && len(short) > 1 {
			return autoflagserrors.NewInvalidShorthandError(fieldName, short)
		}

		// Ensure that flagshort is given to non-struct types
		if short != "" && isStructKind {
			return autoflagserrors.NewInvalidTagUsageError(fieldName, "flagshort", "flagshort cannot be used on struct types")
		}

		// Validate flagcustom tag
		flagCustomValue, flagCustomErr := validateBooleanTag(fieldName, "flagcustom", structF.Tag.Get("flagcustom"))
		if flagCustomErr != nil {
			return flagCustomErr
		}

		// Ensure that flagcustom is given to non-struct types
		if flagCustomValue != nil && *flagCustomValue && isStructKind {
			return autoflagserrors.NewInvalidTagUsageError(fieldName, "flagcustom", "flagcustom cannot be used on struct types")
		}

		// Validate the define and decode hooks when flagcustom is true
		if flagCustomValue != nil && *flagCustomValue && !isStructKind {
			// Map current field name to its custom type
			if !isStandardType(structF.Type) {
				typeToFields[structF.Type] = append(typeToFields[structF.Type], fieldName)
			}
			// Extract the field name (without prefix) for hook lookup
			parts := strings.Split(fieldName, ".")
			methodFieldName := parts[len(parts)-1]

			if err := validateCustomFlag(val, methodFieldName, structF.Type.String()); err != nil {
				return err
			}
		}

		// Validate flagenv tag (can be on struct fields for inheritance)
		if _, err := validateBooleanTag(fieldName, "flagenv", structF.Tag.Get("flagenv")); err != nil {
			return err
		}

		// Validate flagignore tag
		flagIgnoreValue, flagIgnoreErr := validateBooleanTag(fieldName, "flagignore", structF.Tag.Get("flagignore"))
		if flagIgnoreErr != nil {
			return flagIgnoreErr
		}

		// Ensure that flagignore is given to non-struct types
		if flagIgnoreValue != nil && *flagIgnoreValue && isStructKind {
			return autoflagserrors.NewInvalidTagUsageError(fieldName, "flagignore", "flagignore cannot be used on struct types")
		}

		// Validate flagrequired tag
		flagRequiredValue, flagRequiredErr := validateBooleanTag(fieldName, "flagrequired", structF.Tag.Get("flagrequired"))
		if flagRequiredErr != nil {
			return flagRequiredErr
		}

		// Ensure that flagrequired is given to non-struct types
		if flagRequiredValue != nil && *flagRequiredValue && isStructKind {
			return autoflagserrors.NewInvalidTagUsageError(fieldName, "flagrequired", "flagrequired cannot be used on struct types")
		}

		if flagRequiredValue != nil && flagIgnoreValue != nil && *flagRequiredValue && *flagIgnoreValue {
			return autoflagserrors.NewConflictingTagsError(fieldName, []string{"flagignore", "flagrequired"}, "mutually exclusive tags")
		}

		// Recursively validate children structs
		if isStructKind {
			if err := validateFields(field, fieldName, typeToFields); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateBooleanTag validates that a struct tag contains a valid boolean value
func validateBooleanTag(fieldName, tagName, tagValue string) (*bool, error) {
	if tagValue == "" {
		return nil, nil
	}
	val, err := strconv.ParseBool(tagValue)
	if err != nil {
		return nil, autoflagserrors.NewInvalidBooleanTagError(fieldName, tagName, tagValue)
	}

	return &val, nil
}

func validateDefineHookSignature(m reflect.Value) error {
	expectedType := reflect.TypeOf((*DefineHookFunc)(nil)).Elem()
	actualType := m.Type()

	if actualType.NumIn() != expectedType.NumIn() || actualType.NumOut() != expectedType.NumOut() {
		var fx DefineHookFunc

		return fmt.Errorf("define hook must have signature: %s", signature(fx))
	}

	for i := range actualType.NumIn() {
		if actualType.In(i) != expectedType.In(i) {
			return fmt.Errorf("define hook parameter %d has wrong type: expected %v, got %v", i, expectedType.In(i), actualType.In(i))
		}
	}

	return nil
}

func validateDecodeHookSignature(m reflect.Value) error {
	expectedType := reflect.TypeOf((*DecodeHookFunc)(nil)).Elem()
	actualType := m.Type()

	if actualType.NumIn() != expectedType.NumIn() || actualType.NumOut() != expectedType.NumOut() {
		var fx DecodeHookFunc

		return fmt.Errorf("decode hook must have signature: %s", signature(fx))
	}

	if actualType.In(0) != expectedType.In(0) {
		return fmt.Errorf("decode hook input parameter has wrong type: expected %v, got %v", expectedType.In(0), actualType.In(0))
	}

	if actualType.Out(0) != expectedType.Out(0) ||
		actualType.Out(1) != expectedType.Out(1) {
		return fmt.Errorf("decode hook must return (any, error)")
	}

	return nil
}

// validateCustomFlag validates that a custom flag has proper define and decode mechanisms
func validateCustomFlag(structValue reflect.Value, fieldName, fieldType string) error {
	// Get pointer to struct to access methods
	structPtr := getStructPtr(structValue)
	if !structPtr.IsValid() {
		return fmt.Errorf("cannot get pointer to struct for field '%s'", fieldName)
	}

	// Check if struct has Define<FieldName> method
	defineMethodName := fmt.Sprintf("Define%s", fieldName)
	defineHookFunc := structPtr.MethodByName(defineMethodName)

	// Check if struct has Decode<FieldName> method
	decodeMethodName := fmt.Sprintf("Decode%s", fieldName)
	decodeHookFunc := structPtr.MethodByName(decodeMethodName)

	// Case 1: User has defined custom methods
	if defineHookFunc.IsValid() {
		// Must have corresponding decode method
		if !decodeHookFunc.IsValid() {
			return autoflagserrors.NewMissingDecodeHookError(fieldName, decodeMethodName)
		}

		// Validate signatures
		if err := validateDefineHookSignature(defineHookFunc); err != nil {
			return autoflagserrors.NewInvalidDefineHookSignatureError(fieldName, defineMethodName, err)
		}
		if err := validateDecodeHookSignature(decodeHookFunc); err != nil {
			return autoflagserrors.NewInvalidDecodeHookSignatureError(fieldName, decodeMethodName, err)
		}

		return nil
	}

	// Check registries
	_, inDefineRegistry := defineHookRegistry[fieldType]
	_, inDecodeRegistry := decodeHookRegistry[fieldType]

	// Case 2: Check registry
	if inDefineRegistry {
		if !inDecodeRegistry {
			return fmt.Errorf("internal error: missing decode hook for built-in type %s", fieldType)
		}

		return nil
	}

	// Case 3: No define mechanism found
	return autoflagserrors.NewMissingDefineHookError(fieldName, defineMethodName)
}

var standardTypes = func() map[reflect.Kind]reflect.Type {
	types := make(map[reflect.Kind]reflect.Type)
	for _, v := range []any{
		"", int(0), bool(false), int8(0), int16(0), int32(0), int64(0),
		uint(0), uint8(0), uint16(0), uint32(0), uint64(0),
	} {
		t := reflect.TypeOf(v)
		types[t.Kind()] = t
	}
	return types
}()

func isStandardType(t reflect.Type) bool {
	expected, exists := standardTypes[t.Kind()]

	return exists && t == expected
}

func signature(f any) string {
	t := reflect.TypeOf(f)
	if t.Kind() != reflect.Func {
		return "<not a function>"
	}

	buf := strings.Builder{}
	buf.WriteString("func (")
	for i := 0; i < t.NumIn(); i++ {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(t.In(i).String())
	}
	buf.WriteString(")")
	if numOut := t.NumOut(); numOut > 0 {
		if numOut > 1 {
			buf.WriteString(" (")
		} else {
			buf.WriteString(" ")
		}
		for i := 0; i < t.NumOut(); i++ {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(t.Out(i).String())
		}
		if numOut > 1 {
			buf.WriteString(")")
		}
	}

	return buf.String()
}
