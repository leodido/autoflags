package errors

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvalidBooleanTagError_ErrorMessage(t *testing.T) {
	err := &InvalidBooleanTagError{
		FieldName: "InvalidCustom",
		TagName:   "flagcustom",
		TagValue:  "invalid",
	}

	expected := "field 'InvalidCustom': tag 'flagcustom=invalid': invalid boolean value"
	assert.Equal(t, expected, err.Error())
}

func TestInvalidBooleanTagError_ContainsExpectedStrings(t *testing.T) {
	err := &InvalidBooleanTagError{
		FieldName: "SomeField",
		TagName:   "flagcustom",
		TagValue:  "bad_value",
	}

	errorMsg := err.Error()

	// These are the strings our flagcustom test expects to find
	assert.Contains(t, errorMsg, "SomeField")
	assert.Contains(t, errorMsg, "flagcustom")
	assert.Contains(t, errorMsg, "bad_value")
}

func TestInvalidBooleanTagError_FieldInterface(t *testing.T) {
	err := &InvalidBooleanTagError{
		FieldName: "TestField",
		TagName:   "flagenv",
		TagValue:  "maybe",
	}

	// Test that it implements DefinitionError interface
	var fieldErr DefinitionError = err
	assert.Equal(t, "TestField", fieldErr.Field())
}

func TestInvalidBooleanTagError_ErrorsIs(t *testing.T) {
	err := &InvalidBooleanTagError{
		FieldName: "TestField",
		TagName:   "flagenv",
		TagValue:  "invalid",
	}

	// Test errors.Is() functionality
	assert.True(t, errors.Is(err, ErrInvalidBooleanTag))
	assert.False(t, errors.Is(err, ErrInvalidShorthand))
}

func TestInvalidBooleanTagError_ErrorsAs(t *testing.T) {
	err := NewInvalidBooleanTagError("TestField", "flagcustom", "maybe")

	// Test errors.As() functionality
	var boolErr *InvalidBooleanTagError
	require.True(t, errors.As(err, &boolErr))
	assert.Equal(t, "TestField", boolErr.FieldName)
	assert.Equal(t, "flagcustom", boolErr.TagName)
	assert.Equal(t, "maybe", boolErr.TagValue)

	// Test DefinitionError interface extraction
	var fieldErr DefinitionError
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "TestField", fieldErr.Field())
}

func TestInvalidShorthandError_ErrorMessage(t *testing.T) {
	err := &InvalidShorthandError{
		FieldName: "VerboseFlag",
		Shorthand: "verb",
	}

	expected := "field 'VerboseFlag': shorthand flag 'verb' must be a single character"
	assert.Equal(t, expected, err.Error())
}

func TestInvalidShorthandError_ContainsExpectedStrings(t *testing.T) {
	err := &InvalidShorthandError{
		FieldName: "SomeFlag",
		Shorthand: "abc",
	}

	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "SomeFlag")
	assert.Contains(t, errorMsg, "abc")
	assert.Contains(t, errorMsg, "single character")
}

func TestInvalidShorthandError_ErrorsIs(t *testing.T) {
	err := &InvalidShorthandError{
		FieldName: "TestField",
		Shorthand: "too-long",
	}

	assert.True(t, errors.Is(err, ErrInvalidShorthand))
	assert.False(t, errors.Is(err, ErrInvalidBooleanTag))
}

func TestMissingDefineHookError_ErrorMessage(t *testing.T) {
	err := &MissingDefineHookError{
		FieldName:    "ServerMode",
		ExpectedHook: "DefineServerMode",
	}

	expected := "field 'ServerMode': flagcustom='true' but missing define hook 'DefineServerMode'"
	assert.Equal(t, expected, err.Error())
}

func TestMissingDefineHookError_ErrorsIs(t *testing.T) {
	err := &MissingDefineHookError{
		FieldName:    "TestField",
		ExpectedHook: "DefineTestField",
	}

	assert.True(t, errors.Is(err, ErrMissingDefineHook))
	assert.False(t, errors.Is(err, ErrInvalidBooleanTag))
}

func TestMissingDecodeHookError_ErrorMessage(t *testing.T) {
	err := &MissingDecodeHookError{
		FieldName:    "ServerMode",
		ExpectedHook: "DecodeServerMode",
	}

	expected := "field 'ServerMode': flagcustom='true' but missing decode hook 'DecodeServerMode'"
	assert.Equal(t, expected, err.Error())
}

func TestMissingDecodeHookError_ErrorsIs(t *testing.T) {
	err := &MissingDecodeHookError{
		FieldName:    "TestField",
		ExpectedHook: "DecodeTestField",
	}

	assert.True(t, errors.Is(err, ErrMissingDecodeHook))
	assert.False(t, errors.Is(err, ErrInvalidBooleanTag))
}

func TestInvalidTagUsageError_ErrorMessage(t *testing.T) {
	err := &InvalidTagUsageError{
		FieldName: "TestField",
		TagName:   "flagignore",
		Message:   "cannot ignore a required field",
	}

	expected := "field 'TestField': invalid usage of tag 'flagignore': cannot ignore a required field"
	assert.Equal(t, expected, err.Error())
}

func TestInvalidTagUsageError_ErrorsIs(t *testing.T) {
	err := &InvalidTagUsageError{
		FieldName: "TestField",
		TagName:   "tag1",
		Message:   "message",
	}

	assert.True(t, errors.Is(err, ErrInvalidTagUsage))
	assert.False(t, errors.Is(err, ErrUnsupportedType))
}

func TestUnsupportedTypeError_ErrorMessage(t *testing.T) {
	err := &UnsupportedTypeError{
		FieldName: "ComplexField",
		FieldType: "complex128",
		Message:   "complex numbers are not supported as flags",
	}

	expected := "field 'ComplexField': unsupported type 'complex128': complex numbers are not supported as flags"
	assert.Equal(t, expected, err.Error())
}

func TestUnsupportedTypeError_ErrorsIs(t *testing.T) {
	err := &UnsupportedTypeError{
		FieldName: "TestField",
		FieldType: "TestType",
		Message:   "not supported",
	}

	assert.True(t, errors.Is(err, ErrUnsupportedType))
	assert.False(t, errors.Is(err, ErrInvalidShorthand))
}

func TestDuplicateFlagError_ErrorMessage(t *testing.T) {
	err := &DuplicateFlagError{
		FlagName:          "port",
		NewFieldPath:      "Server.Port",
		ExistingFieldPath: "Database.Port",
	}

	expected := "field 'Server.Port': flag name 'port' is already in use by field 'Database.Port'"
	assert.Equal(t, expected, err.Error())
}

func TestDuplicateFlagError_ContainsExpectedStrings(t *testing.T) {
	err := &DuplicateFlagError{
		FlagName:          "port",
		NewFieldPath:      "Server.Port",
		ExistingFieldPath: "Database.Port",
	}

	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "port")
	assert.Contains(t, errorMsg, "Server.Port")
	assert.Contains(t, errorMsg, "Database.Port")
}

func TestDuplicateFlagError_FieldInterface(t *testing.T) {
	err := &DuplicateFlagError{
		FlagName:     "port",
		NewFieldPath: "Server.Port",
	}

	// Test that it implements DefinitionError interface
	var fieldErr DefinitionError = err
	assert.Equal(t, "Server.Port", fieldErr.Field())
}

func TestDuplicateFlagError_ErrorsIs(t *testing.T) {
	err := &DuplicateFlagError{
		FlagName: "port",
	}

	assert.True(t, errors.Is(err, ErrDuplicateFlag))
	assert.False(t, errors.Is(err, ErrInvalidShorthand))
}

func TestDuplicateFlagError_ErrorsAs(t *testing.T) {
	err := NewDuplicateFlagError("port", "Server.Port", "Database.Port")

	// Test errors.As() functionality
	var dupErr *DuplicateFlagError
	require.True(t, errors.As(err, &dupErr))
	assert.Equal(t, "port", dupErr.FlagName)
	assert.Equal(t, "Server.Port", dupErr.NewFieldPath)
	assert.Equal(t, "Database.Port", dupErr.ExistingFieldPath)

	// Test DefinitionError interface extraction
	var fieldErr DefinitionError
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "Server.Port", fieldErr.Field())
}

func TestInvalidFlagNameError_ErrorMessage(t *testing.T) {
	err := &InvalidFlagNameError{
		FieldName: "MyField",
		FlagName:  "invalid flag",
	}

	expected := "field 'MyField': generated flag name 'invalid flag' is invalid. Use only alphanumeric characters, dashes, and dots."
	assert.Equal(t, expected, err.Error())
}

func TestInvalidFlagNameError_FieldInterface(t *testing.T) {
	err := &InvalidFlagNameError{
		FieldName: "TestField",
		FlagName:  "test-flag",
	}

	// Test that it implements DefinitionError interface
	var fieldErr DefinitionError = err
	assert.Equal(t, "TestField", fieldErr.Field())
}

func TestInvalidFlagNameError_ErrorsIs(t *testing.T) {
	err := &InvalidFlagNameError{
		FieldName: "TestField",
		FlagName:  "invalid flag",
	}

	// Test errors.Is() functionality
	assert.True(t, errors.Is(err, ErrInvalidFlagName))
	assert.False(t, errors.Is(err, ErrInvalidShorthand))
}

func TestInvalidFlagNameError_ErrorsAs(t *testing.T) {
	err := NewInvalidFlagNameError("TestField", "invalid-flag ")

	// Test errors.As() functionality
	var flagErr *InvalidFlagNameError
	require.True(t, errors.As(err, &flagErr))
	assert.Equal(t, "TestField", flagErr.FieldName)
	assert.Equal(t, "invalid-flag ", flagErr.FlagName)

	// Test DefinitionError interface extraction
	var fieldErr DefinitionError
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "TestField", fieldErr.Field())
}

func TestNewDuplicateFlagError_Constructor(t *testing.T) {
	err := NewDuplicateFlagError("port", "Server.Port", "Database.Port")

	var dupErr *DuplicateFlagError
	require.True(t, errors.As(err, &dupErr))
	assert.Equal(t, "port", dupErr.FlagName)
	assert.Equal(t, "Server.Port", dupErr.NewFieldPath)
	assert.Equal(t, "Database.Port", dupErr.ExistingFieldPath)
}

func TestNewInvalidBooleanTagError_Constructor(t *testing.T) {
	err := NewInvalidBooleanTagError("TestField", "flagenv", "maybe")

	var boolErr *InvalidBooleanTagError
	require.True(t, errors.As(err, &boolErr))
	assert.Equal(t, "TestField", boolErr.FieldName)
	assert.Equal(t, "flagenv", boolErr.TagName)
	assert.Equal(t, "maybe", boolErr.TagValue)
}

func TestNewInvalidShorthandError_Constructor(t *testing.T) {
	err := NewInvalidShorthandError("VerboseFlag", "verb")

	var shortErr *InvalidShorthandError
	require.True(t, errors.As(err, &shortErr))
	assert.Equal(t, "VerboseFlag", shortErr.FieldName)
	assert.Equal(t, "verb", shortErr.Shorthand)
}

func TestNewMissingDefineHookError_Constructor(t *testing.T) {
	err := NewMissingDefineHookError("ServerMode", "DefineServerMode")

	var hookErr *MissingDefineHookError
	require.True(t, errors.As(err, &hookErr))
	assert.Equal(t, "ServerMode", hookErr.FieldName)
	assert.Equal(t, "DefineServerMode", hookErr.ExpectedHook)
}

func TestNewMissingDecodeHookError_Constructor(t *testing.T) {
	err := NewMissingDecodeHookError("ServerMode", "DecodeServerMode")

	var hookErr *MissingDecodeHookError
	require.True(t, errors.As(err, &hookErr))
	assert.Equal(t, "ServerMode", hookErr.FieldName)
	assert.Equal(t, "DecodeServerMode", hookErr.ExpectedHook)
}

func TestNewInvalidTagUsageError_Constructor(t *testing.T) {
	err := NewInvalidTagUsageError("TestField", "flagrequired", "cannot ignore required field")

	var tagErr *InvalidTagUsageError
	require.True(t, errors.As(err, &tagErr))
	assert.Equal(t, "TestField", tagErr.FieldName)
	assert.Equal(t, "flagrequired", tagErr.TagName)
	assert.Equal(t, "cannot ignore required field", tagErr.Message)
}

func TestNewUnsupportedTypeError_Constructor(t *testing.T) {
	err := NewUnsupportedTypeError("ComplexField", "complex128", "not supported")

	var typeErr *UnsupportedTypeError
	require.True(t, errors.As(err, &typeErr))
	assert.Equal(t, "ComplexField", typeErr.FieldName)
	assert.Equal(t, "complex128", typeErr.FieldType)
	assert.Equal(t, "not supported", typeErr.Message)
}

func TestNewInvalidFlagNameError_Constructor(t *testing.T) {
	err := NewInvalidFlagNameError("MyField", "bad flag")

	var flagErr *InvalidFlagNameError
	require.True(t, errors.As(err, &flagErr))
	assert.Equal(t, "MyField", flagErr.FieldName)
	assert.Equal(t, "bad flag", flagErr.FlagName)
}

func TestDefinitionError_Interface_MultipleTypes(t *testing.T) {
	tests := []struct {
		name  string
		err   DefinitionError
		field string
	}{
		{
			name: "InvalidBooleanTagError",
			err: &InvalidBooleanTagError{
				FieldName: "BoolField",
				TagName:   "flagenv",
				TagValue:  "invalid",
			},
			field: "BoolField",
		},
		{
			name: "InvalidShorthandError",
			err: &InvalidShorthandError{
				FieldName: "ShortField",
				Shorthand: "too-long",
			},
			field: "ShortField",
		},
		{
			name: "MissingDefineHookError",
			err: &MissingDefineHookError{
				FieldName:    "CustomField",
				ExpectedHook: "DefineCustomField",
			},
			field: "CustomField",
		},
		{
			name: "MissingDecodeHookError",
			err: &MissingDecodeHookError{
				FieldName:    "CustomField",
				ExpectedHook: "DecodeCustomField",
			},
			field: "CustomField",
		},
		{
			name: "InvalidTagUsage",
			err: &InvalidTagUsageError{
				FieldName: "InvalidTagField",
				TagName:   "tag2",
				Message:   "invalid_tag",
			},
			field: "InvalidTagField",
		},
		{
			name: "ConflictingTagsError",
			err: &ConflictingTagsError{
				FieldName:       "ConflictField",
				ConflictingTags: []string{"tag1", "tag2"},
				Message:         "conflict",
			},
			field: "ConflictField",
		},
		{
			name: "ConflictingTypeError",
			err: &ConflictingTypeError{
				Type:     reflect.TypeOf(""),
				TypeName: "string",
				Fields:   []string{"ConflictField1", "ConflictField2"},
				Message:  "conflict",
			},
			field: "ConflictField1, ConflictField2",
		},
		{
			name: "UnsupportedTypeError",
			err: &UnsupportedTypeError{
				FieldName: "UnsupportedField",
				FieldType: "UnsupportedType",
				Message:   "not supported",
			},
			field: "UnsupportedField",
		},
		{
			name: "DuplicateFlagError",
			err: &DuplicateFlagError{
				FlagName:          "port",
				NewFieldPath:      "New.Path.Port",
				ExistingFieldPath: "Old.Path.Port",
			},
			field: "New.Path.Port",
		},
		{
			name: "InvalidFlagNameError",
			err: &InvalidFlagNameError{
				FieldName: "FlagField",
				FlagName:  "invalid name",
			},
			field: "FlagField",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.field, tt.err.Field())
		})
	}
}

func TestErrorChaining_WithWrapping(t *testing.T) {
	originalErr := NewInvalidBooleanTagError("TestField", "flagcustom", "invalid")

	// Test wrapping with additional context
	wrappedErr := fmt.Errorf("failed to process field: %w", originalErr)

	// Should still work with errors.Is through the wrap
	assert.True(t, errors.Is(wrappedErr, ErrInvalidBooleanTag))

	// Should still work with errors.As through the wrap
	var boolErr *InvalidBooleanTagError
	assert.True(t, errors.As(wrappedErr, &boolErr))
	assert.Equal(t, "TestField", boolErr.FieldName)
}

func TestValidationError_ErrorMessage_WithContextName(t *testing.T) {
	err1 := fmt.Errorf("a")
	err2 := fmt.Errorf("b")

	validationErr := &ValidationError{
		ContextName: "server",
		Errors:      []error{err1, err2},
	}

	expected := "invalid options for server:\n" +
		"       a\n" +
		"       b"

	assert.Equal(t, expected, validationErr.Error())
}

func TestValidationError_ErrorMessage_WithoutContextName(t *testing.T) {
	err1 := errors.New("first error")
	err2 := errors.New("some other error")

	validationErr := &ValidationError{
		ContextName: "",
		Errors:      []error{err1, err2},
	}

	expected := "invalid options:\n" +
		"       first error\n" +
		"       some other error"

	assert.Equal(t, expected, validationErr.Error())
}

func TestValidationError_ErrorMessage_WithContextName_NoErrors(t *testing.T) {
	validationErr := &ValidationError{
		ContextName: "testcmd",
		Errors:      []error{},
	}

	expected := "invalid options for testcmd"
	assert.Equal(t, expected, validationErr.Error())
}

func TestValidationError_ErrorMessage_WithoutContextName_NoErrors(t *testing.T) {
	validationErr := &ValidationError{
		Errors: []error{},
	}

	expected := "invalid options"
	assert.Equal(t, expected, validationErr.Error())
}

func TestValidationError_ErrorMessage_WithContextName_NilErrors(t *testing.T) {
	validationErr := &ValidationError{
		ContextName: "testcmd",
		Errors:      nil,
	}

	expected := "invalid options for testcmd"
	assert.Equal(t, expected, validationErr.Error())
}

func TestValidationError_ErrorMessage_WithoutContextName_NilErrors(t *testing.T) {
	validationErr := &ValidationError{
		Errors: nil,
	}

	expected := "invalid options"
	assert.Equal(t, expected, validationErr.Error())
}

func TestValidationError_UnderlyingErrors_ReturnsCorrectSlice(t *testing.T) {
	err1 := NewInvalidBooleanTagError("Field1", "flagcustom", "invalid")
	err2 := fmt.Errorf("errorf")
	err3 := errors.New("custom error")

	originalErrors := []error{err1, err2, err3}
	validationErr := &ValidationError{
		ContextName: "server",
		Errors:      originalErrors,
	}

	underlyingErrors := validationErr.UnderlyingErrors()

	require.Len(t, underlyingErrors, 3)
	require.Equal(t, originalErrors, underlyingErrors)
}

func TestValidationError_UnderlyingErrors_EmptySlice(t *testing.T) {
	validationErr := &ValidationError{
		ContextName: "testcmd",
		Errors:      []error{},
	}

	underlyingErrors := validationErr.UnderlyingErrors()

	require.NotNil(t, underlyingErrors)
	require.Len(t, underlyingErrors, 0)
}

func TestValidationError_UnderlyingErrors_NilSlice(t *testing.T) {
	validationErr := &ValidationError{
		ContextName: "testcmd",
		Errors:      nil,
	}

	underlyingErrors := validationErr.UnderlyingErrors()

	require.Nil(t, underlyingErrors)
}

func TestValidationError_UnderlyingErrors_Immutability(t *testing.T) {
	err1 := errors.New("ciao")
	err2 := errors.New("hello")

	originalErrors := []error{err1, err2}
	validationErr := &ValidationError{
		ContextName: "server",
		Errors:      originalErrors,
	}

	// Get the underlying errors
	underlyingErrors := validationErr.UnderlyingErrors()

	// Modify the returned slice
	underlyingErrors[0] = errors.New("modified error")

	require.NotEqual(t, "modified error", validationErr.Errors[0].Error())
	require.Equal(t, err1, validationErr.Errors[0])
}

func TestInvalidDecodeHookSignatureError_ErrorMessage(t *testing.T) {
	err := &InvalidDecodeHookSignatureError{
		FieldName: "ServerMode",
		HookName:  "DecodeServerMode",
		Message:   "decode hook must have signature: func(input interface{}) (interface{}, error)",
	}

	expected := "field 'ServerMode': invalid 'DecodeServerMode' decode hook: decode hook must have signature: func(input interface{}) (interface{}, error)"
	assert.Equal(t, expected, err.Error())
}

func TestInvalidDecodeHookSignatureError_ContainsExpectedStrings(t *testing.T) {
	err := &InvalidDecodeHookSignatureError{
		FieldName: "CustomField",
		HookName:  "DecodeCustomField",
		Message:   "wrong signature",
	}

	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "CustomField")
	assert.Contains(t, errorMsg, "DecodeCustomField")
	assert.Contains(t, errorMsg, "wrong signature")
	assert.Contains(t, errorMsg, "decode hook")
}

func TestInvalidDecodeHookSignatureError_FieldInterface(t *testing.T) {
	err := &InvalidDecodeHookSignatureError{
		FieldName: "TestField",
		HookName:  "DecodeTestField",
		Message:   "invalid signature",
	}

	// Test that it implements DefinitionError interface
	var fieldErr DefinitionError = err
	assert.Equal(t, "TestField", fieldErr.Field())
}

func TestInvalidDecodeHookSignatureError_ErrorsIs(t *testing.T) {
	err := &InvalidDecodeHookSignatureError{
		FieldName: "TestField",
		HookName:  "DecodeTestField",
		Message:   "invalid signature",
	}

	// Test errors.Is() functionality
	assert.True(t, errors.Is(err, ErrInvalidDecodeHookSignature))
	assert.False(t, errors.Is(err, ErrInvalidDefineHookSignature))
	assert.False(t, errors.Is(err, ErrInvalidBooleanTag))
}

func TestInvalidDecodeHookSignatureError_ErrorsAs(t *testing.T) {
	originalErr := errors.New("parameter 0 has wrong type")
	err := NewInvalidDecodeHookSignatureError("TestField", "DecodeTestField", originalErr)

	// Test errors.As() functionality
	var decodeErr *InvalidDecodeHookSignatureError
	require.True(t, errors.As(err, &decodeErr))
	assert.Equal(t, "TestField", decodeErr.FieldName)
	assert.Equal(t, "DecodeTestField", decodeErr.HookName)
	assert.Equal(t, "parameter 0 has wrong type", decodeErr.Message)

	// Test DefinitionError interface extraction
	var fieldErr DefinitionError
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "TestField", fieldErr.Field())
}

func TestInvalidDefineHookSignatureError_ErrorMessage(t *testing.T) {
	err := &InvalidDefineHookSignatureError{
		FieldName: "ServerMode",
		HookName:  "DefineServerMode",
		Message:   "define hook must have signature: func(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value)",
	}

	expected := "field 'ServerMode': invalid 'DefineServerMode' define hook: define hook must have signature: func(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value)"
	assert.Equal(t, expected, err.Error())
}

func TestInvalidDefineHookSignatureError_ContainsExpectedStrings(t *testing.T) {
	err := &InvalidDefineHookSignatureError{
		FieldName: "CustomField",
		HookName:  "DefineCustomField",
		Message:   "wrong number of parameters",
	}

	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "CustomField")
	assert.Contains(t, errorMsg, "DefineCustomField")
	assert.Contains(t, errorMsg, "wrong number of parameters")
	assert.Contains(t, errorMsg, "define hook")
}

func TestInvalidDefineHookSignatureError_FieldInterface(t *testing.T) {
	err := &InvalidDefineHookSignatureError{
		FieldName: "TestField",
		HookName:  "DefineTestField",
		Message:   "invalid signature",
	}

	// Test that it implements DefinitionError interface
	var fieldErr DefinitionError = err
	assert.Equal(t, "TestField", fieldErr.Field())
}

func TestInvalidDefineHookSignatureError_ErrorsIs(t *testing.T) {
	err := &InvalidDefineHookSignatureError{
		FieldName: "TestField",
		HookName:  "DefineTestField",
		Message:   "invalid signature",
	}

	// Test errors.Is() functionality
	assert.True(t, errors.Is(err, ErrInvalidDefineHookSignature))
	assert.False(t, errors.Is(err, ErrInvalidDecodeHookSignature))
	assert.False(t, errors.Is(err, ErrInvalidBooleanTag))
}

func TestInvalidDefineHookSignatureError_ErrorsAs(t *testing.T) {
	originalErr := errors.New("parameter 1 has wrong type")
	err := NewInvalidDefineHookSignatureError("TestField", "DefineTestField", originalErr)

	// Test errors.As() functionality
	var defineErr *InvalidDefineHookSignatureError
	require.True(t, errors.As(err, &defineErr))
	assert.Equal(t, "TestField", defineErr.FieldName)
	assert.Equal(t, "DefineTestField", defineErr.HookName)
	assert.Equal(t, "parameter 1 has wrong type", defineErr.Message)

	// Test DefinitionError interface extraction
	var fieldErr DefinitionError
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "TestField", fieldErr.Field())
}

func TestConflictingTagsError_ErrorMessage(t *testing.T) {
	err := &ConflictingTagsError{
		FieldName:       "TestField",
		ConflictingTags: []string{"flagignore", "flagrequired"},
		Message:         "cannot ignore a required field",
	}
	expected := "field 'TestField': conflicting tags [flagignore, flagrequired]: cannot ignore a required field"
	assert.Equal(t, expected, err.Error())
}

func TestConflictingTagsError_ErrorsIs(t *testing.T) {
	err := &ConflictingTagsError{
		FieldName:       "TestField",
		ConflictingTags: []string{"tag1", "tag2"},
		Message:         "conflict message",
	}
	assert.True(t, errors.Is(err, ErrConflictingTags))
	assert.False(t, errors.Is(err, ErrUnsupportedType))
}

func TestNewConflictingTagsError_Constructor(t *testing.T) {
	tags := []string{"flagignore", "flagrequired"}
	err := NewConflictingTagsError("TestField", tags, "cannot ignore required field")
	var conflictErr *ConflictingTagsError
	require.True(t, errors.As(err, &conflictErr))
	assert.Equal(t, "TestField", conflictErr.FieldName)
	assert.Equal(t, tags, conflictErr.ConflictingTags)
	assert.Equal(t, "cannot ignore required field", conflictErr.Message)
}

func TestConflictingTypeError_ErrorMessage(t *testing.T) {
	// Create a test type for the error
	var testValue int
	testType := reflect.TypeOf(testValue)

	err := &ConflictingTypeError{
		Type:     testType,
		TypeName: testType.String(),
		Fields:   []string{"Field1", "Field2", "Field3"},
		Message:  "create distinct custom types for each field",
	}

	expected := "fields [Field1, Field2, Field3]: conflicting type [int]: create distinct custom types for each field"
	assert.Equal(t, expected, err.Error())
}

func TestConflictingTypeError_ContainsExpectedStrings(t *testing.T) {
	// Create a test type for the error
	type CustomStruct struct {
		Name string
	}
	testType := reflect.TypeOf(CustomStruct{})

	err := &ConflictingTypeError{
		Type:     testType,
		TypeName: testType.String(),
		Fields:   []string{"FieldA", "FieldB"},
		Message:  "multiple fields cannot use the same custom type",
	}

	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "FieldA")
	assert.Contains(t, errorMsg, "FieldB")
	assert.Contains(t, errorMsg, testType.String())
	assert.Contains(t, errorMsg, "multiple fields cannot use the same custom type")
	assert.Contains(t, errorMsg, "conflicting type")
}

func TestConflictingTypeError_FieldInterface(t *testing.T) {
	var testValue string
	testType := reflect.TypeOf(testValue)

	err := &ConflictingTypeError{
		Type:     testType,
		TypeName: testType.String(),
		Fields:   []string{"FirstField", "SecondField"},
		Message:  "test message",
	}

	// Test that it implements DefinitionError interface
	var fieldErr DefinitionError = err
	assert.Equal(t, "FirstField, SecondField", fieldErr.Field())
}

func TestConflictingTypeError_ErrorsIs(t *testing.T) {
	var testValue bool
	testType := reflect.TypeOf(testValue)

	err := &ConflictingTypeError{
		Type:     testType,
		TypeName: testType.String(),
		Fields:   []string{"BoolField1", "BoolField2"},
		Message:  "conflicting boolean fields",
	}

	// Test errors.Is() functionality
	assert.True(t, errors.Is(err, ErrConflictingType))
	assert.False(t, errors.Is(err, ErrConflictingTags))
	assert.False(t, errors.Is(err, ErrUnsupportedType))
	assert.False(t, errors.Is(err, ErrInvalidBooleanTag))
}

func TestConflictingTypeError_ErrorsAs(t *testing.T) {
	type TestType struct {
		Value int
	}
	testType := reflect.TypeOf(TestType{})
	fields := []string{"TestField1", "TestField2"}

	err := NewConflictingTypeError(testType, fields, "test conflict message")

	// Test errors.As() functionality
	var conflictErr *ConflictingTypeError
	require.True(t, errors.As(err, &conflictErr))
	assert.Equal(t, testType, conflictErr.Type)
	assert.Equal(t, testType.String(), conflictErr.TypeName)
	assert.Equal(t, fields, conflictErr.Fields)
	assert.Equal(t, "test conflict message", conflictErr.Message)

	// Test DefinitionError interface extraction
	var fieldErr DefinitionError
	require.True(t, errors.As(err, &fieldErr))
	assert.Equal(t, "TestField1, TestField2", fieldErr.Field())
}

func TestNewConflictingTypeError_Constructor(t *testing.T) {
	type CustomType struct {
		Data string
	}
	testType := reflect.TypeOf(CustomType{})
	fields := []string{"CustomField1", "CustomField2", "CustomField3"}
	message := "create distinct custom types for each field"

	err := NewConflictingTypeError(testType, fields, message)

	var conflictErr *ConflictingTypeError
	require.True(t, errors.As(err, &conflictErr))
	assert.Equal(t, testType, conflictErr.Type)
	assert.Equal(t, testType.String(), conflictErr.TypeName)
	assert.Equal(t, fields, conflictErr.Fields)
	assert.Equal(t, message, conflictErr.Message)
}

func TestInputError_ErrorMessage(t *testing.T) {
	err := &InputError{
		InputType: "nil",
		Message:   "cannot define flags from nil value",
	}

	expected := "invalid input value of type 'nil': cannot define flags from nil value"
	assert.Equal(t, expected, err.Error())
}

func TestInputError_ContainsExpectedStrings(t *testing.T) {
	err := &InputError{
		InputType: "*main.Options",
		Message:   "cannot obtain valid reflection value",
	}

	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "*main.Options")
	assert.Contains(t, errorMsg, "cannot obtain valid reflection value")
	assert.Contains(t, errorMsg, "invalid input value")
}

func TestInputError_ErrorsIs(t *testing.T) {
	err := &InputError{
		InputType: "nil",
		Message:   "cannot define flags from nil value",
	}

	// Test errors.Is() functionality
	require.True(t, errors.Is(err, ErrInputValue))
	assert.False(t, errors.Is(err, ErrInvalidBooleanTag))
	assert.False(t, errors.Is(err, ErrInvalidShorthand))
	assert.False(t, errors.Is(err, ErrMissingDefineHook))
}

func TestInputError_ErrorsAs(t *testing.T) {
	err := NewInputError("nil", "cannot define flags from nil value")

	// Test errors.As() functionality
	var inputErr *InputError
	require.True(t, errors.As(err, &inputErr))
	assert.Equal(t, "nil", inputErr.InputType)
	assert.Equal(t, "cannot define flags from nil value", inputErr.Message)
}

func TestInputError_DifferentInputTypes(t *testing.T) {
	testCases := []struct {
		name      string
		inputType string
		message   string
		expected  string
	}{
		{
			name:      "nil_input",
			inputType: "nil",
			message:   "cannot define flags from nil value",
			expected:  "invalid input value of type 'nil': cannot define flags from nil value",
		},
		{
			name:      "invalid_pointer",
			inputType: "*main.InvalidStruct",
			message:   "cannot obtain valid reflection value",
			expected:  "invalid input value of type '*main.InvalidStruct': cannot obtain valid reflection value",
		},
		{
			name:      "fallback_failed",
			inputType: "interface{}",
			message:   "fallback reflection approach failed",
			expected:  "invalid input value of type 'interface{}': fallback reflection approach failed",
		},
		{
			name:      "complex_type",
			inputType: "map[string]interface{}",
			message:   "unsupported input type for flag definition",
			expected:  "invalid input value of type 'map[string]interface{}': unsupported input type for flag definition",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := &InputError{
				InputType: tc.inputType,
				Message:   tc.message,
			}

			assert.Equal(t, tc.expected, err.Error())
			assert.Equal(t, tc.inputType, err.InputType)
			assert.Equal(t, tc.message, err.Message)
		})
	}
}

func TestNewInputError_Constructor(t *testing.T) {
	err := NewInputError("nil", "cannot define flags from nil value")

	var inputErr *InputError
	require.True(t, errors.As(err, &inputErr))
	assert.Equal(t, "nil", inputErr.InputType)
	assert.Equal(t, "cannot define flags from nil value", inputErr.Message)
}

func TestNewInputError_ConstructorVariations(t *testing.T) {
	testCases := []struct {
		name      string
		inputType string
		message   string
	}{
		{
			name:      "empty_strings",
			inputType: "",
			message:   "",
		},
		{
			name:      "whitespace_strings",
			inputType: " \t\n",
			message:   " \t\n",
		},
		{
			name:      "unicode_strings",
			inputType: "🚀Type",
			message:   "message with unicode: 你好",
		},
		{
			name:      "long_strings",
			inputType: "very.long.package.name.with.many.parts.VeryLongTypeName",
			message:   "this is a very long error message that describes exactly what went wrong during the flag definition process and provides detailed context",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := NewInputError(tc.inputType, tc.message)

			var inputErr *InputError
			require.True(t, errors.As(err, &inputErr))
			assert.Equal(t, tc.inputType, inputErr.InputType)
			assert.Equal(t, tc.message, inputErr.Message)

			// Verify the error message format
			expectedMsg := fmt.Sprintf("invalid input value of type '%s': %s", tc.inputType, tc.message)
			assert.Equal(t, expectedMsg, err.Error())
		})
	}
}

func TestInputError_ErrorChaining(t *testing.T) {
	originalErr := NewInputError("nil", "cannot define flags from nil value")

	// Test wrapping with additional context
	wrappedErr := fmt.Errorf("failed to process input: %w", originalErr)

	// Should still work with errors.Is through the wrap
	assert.True(t, errors.Is(wrappedErr, ErrInputValue))

	// Should still work with errors.As through the wrap
	var inputErr *InputError
	assert.True(t, errors.As(wrappedErr, &inputErr))
	assert.Equal(t, "nil", inputErr.InputType)
	assert.Equal(t, "cannot define flags from nil value", inputErr.Message)
}

func TestInputError_Vs_DefinitionError_Distinction(t *testing.T) {
	// Create an InputError
	inputErr := NewInputError("nil", "cannot define flags from nil value")

	// Create a DefinitionError
	fieldErr := NewInvalidBooleanTagError("TestField", "flagcustom", "invalid")

	// InputError should NOT implement DefinitionError interface
	var defErr DefinitionError
	assert.False(t, errors.As(inputErr, &defErr), "InputError should not implement DefinitionError")

	// DefinitionError should NOT be an InputError
	var inErr *InputError
	assert.False(t, errors.As(fieldErr, &inErr), "DefinitionError should not be an InputError")

	// They should have different error variable types
	assert.True(t, errors.Is(inputErr, ErrInputValue))
	assert.False(t, errors.Is(inputErr, ErrInvalidBooleanTag))

	assert.True(t, errors.Is(fieldErr, ErrInvalidBooleanTag))
	assert.False(t, errors.Is(fieldErr, ErrInputValue))
}