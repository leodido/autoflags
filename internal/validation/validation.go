package internalvalidation

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	autoflagserrors "github.com/leodido/autoflags/errors"
	internalhooks "github.com/leodido/autoflags/internal/hooks"
	internalpath "github.com/leodido/autoflags/internal/path"
	internalreflect "github.com/leodido/autoflags/internal/reflect"
	internalscope "github.com/leodido/autoflags/internal/scope"
	internaltag "github.com/leodido/autoflags/internal/tag"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// IsValidBoolTag validates that a struct tag contains a valid boolean value
func IsValidBoolTag(fieldName, tagName, tagValue string) (*bool, error) {
	if tagValue == "" {
		return nil, nil
	}
	val, err := strconv.ParseBool(tagValue)
	if err != nil {
		return nil, autoflagserrors.NewInvalidBooleanTagError(fieldName, tagName, tagValue)
	}

	return &val, nil
}

// Struct checks the coherence of definitions in the given struct
func Struct(c *cobra.Command, o any) error {
	val, err := internalreflect.GetValidValue(o)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	s := internalscope.Get(c)

	typeToFields := make(map[reflect.Type][]string)
	typeName := val.Type().Name()
	if err := Fields(val, typeName, typeToFields, s); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	for fieldType, fieldNames := range typeToFields {
		if len(fieldNames) > 1 {
			return autoflagserrors.NewConflictingTypeError(fieldType, fieldNames, "create distinct custom types for each field")
		}
	}

	return nil
}

// Fields recursively validates the struct fields
func Fields(val reflect.Value, prefix string, typeToFields map[reflect.Type][]string, s *internalscope.Scope) error {
	for i := range val.NumField() {
		field := val.Field(i)
		structF := val.Type().Field(i)

		// Skip private fields
		if !field.CanInterface() {
			continue
		}

		fieldName := internalpath.GetFieldName(prefix, structF)
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
		flagCustomValue, flagCustomErr := IsValidBoolTag(fieldName, "flagcustom", structF.Tag.Get("flagcustom"))
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
			if !internaltag.IsStandardType(structF.Type) {
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
		if _, err := IsValidBoolTag(fieldName, "flagenv", structF.Tag.Get("flagenv")); err != nil {
			return err
		}

		// Validate flagignore tag
		flagIgnoreValue, flagIgnoreErr := IsValidBoolTag(fieldName, "flagignore", structF.Tag.Get("flagignore"))
		if flagIgnoreErr != nil {
			return flagIgnoreErr
		}

		// Ensure that flagignore is given to non-struct types
		if flagIgnoreValue != nil && *flagIgnoreValue && isStructKind {
			return autoflagserrors.NewInvalidTagUsageError(fieldName, "flagignore", "flagignore cannot be used on struct types")
		}

		// Validate flagrequired tag
		flagRequiredValue, flagRequiredErr := IsValidBoolTag(fieldName, "flagrequired", structF.Tag.Get("flagrequired"))
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

		// Check for duplicate flags
		if !isStructKind {
			// Skip ignored fields from duplicate check
			if flagIgnoreValue != nil && *flagIgnoreValue {
				continue
			}

			alias := structF.Tag.Get("flag")
			var flagName string
			if alias != "" {
				flagName = alias
			} else {
				flagName = strings.ToLower(structF.Name)
			}

			if !internaltag.IsValidFlagName(flagName) {
				return autoflagserrors.NewInvalidFlagNameError(fieldName, flagName)
			}

			if err := s.AddDefinedFlag(flagName, fieldName); err != nil {
				return err
			}
		}

		// Recursively validate children structs
		if isStructKind {
			if err := Fields(field, fieldName, typeToFields, s); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateDefineHookSignature(m reflect.Value) error {
	expectedType := reflect.TypeOf((*internalhooks.DefineHookFunc)(nil)).Elem()
	actualType := m.Type()

	if actualType.NumIn() != expectedType.NumIn() || actualType.NumOut() != expectedType.NumOut() {
		var fx internalhooks.DefineHookFunc

		return fmt.Errorf("define hook must have signature: %s", internalreflect.Signature(fx))
	}

	// Check input types
	for i := range actualType.NumIn() {
		if actualType.In(i) != expectedType.In(i) {
			return fmt.Errorf("define hook parameter %d has wrong type: expected %v, got %v", i, expectedType.In(i), actualType.In(i))
		}
	}

	// Check return types
	pflagValueType := reflect.TypeOf((*pflag.Value)(nil)).Elem()
	if !actualType.Out(0).Implements(pflagValueType) {
		return fmt.Errorf("define hook first return value must be a pflag.Value")
	}
	if actualType.Out(1).Kind() != reflect.String {
		return fmt.Errorf("define hook second return value must be a string")
	}

	return nil
}

func validateDecodeHookSignature(m reflect.Value) error {
	expectedType := reflect.TypeOf((*internalhooks.DecodeHookFunc)(nil)).Elem()
	actualType := m.Type()

	if actualType.NumIn() != expectedType.NumIn() || actualType.NumOut() != expectedType.NumOut() {
		var fx internalhooks.DecodeHookFunc

		return fmt.Errorf("decode hook must have signature: %s", internalreflect.Signature(fx))
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
	structPtr := internalreflect.GetStructPtr(structValue)
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
	_, inDefineRegistry := internalhooks.DefineHookRegistry[fieldType]
	_, inDecodeRegistry := internalhooks.DecodeHookRegistry[fieldType]

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
