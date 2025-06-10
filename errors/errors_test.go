package errors

import (
	"errors"
	"fmt"
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

	// Test that it implements FieldError interface
	var fieldErr FieldError = err
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

	// Test FieldError interface extraction
	var fieldErr FieldError
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

func TestMissingCustomHookError_ErrorMessage(t *testing.T) {
	err := &MissingCustomHookError{
		FieldName:    "ServerMode",
		ExpectedHook: "DefineServerMode",
		TypeName:     "main.ServerMode",
	}

	expected := "field 'ServerMode': flagcustom='true' but hook 'DefineServerMode' not found for type 'main.ServerMode'"
	assert.Equal(t, expected, err.Error())
}

func TestMissingCustomHookError_ErrorsIs(t *testing.T) {
	err := &MissingCustomHookError{
		FieldName:    "TestField",
		ExpectedHook: "DefineTestField",
		TypeName:     "TestType",
	}

	assert.True(t, errors.Is(err, ErrMissingCustomHook))
	assert.False(t, errors.Is(err, ErrInvalidBooleanTag))
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

func TestNewMissingCustomHookError_Constructor(t *testing.T) {
	err := NewMissingCustomHookError("ServerMode", "DefineServerMode", "main.ServerMode")

	var hookErr *MissingCustomHookError
	require.True(t, errors.As(err, &hookErr))
	assert.Equal(t, "ServerMode", hookErr.FieldName)
	assert.Equal(t, "DefineServerMode", hookErr.ExpectedHook)
	assert.Equal(t, "main.ServerMode", hookErr.TypeName)
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

func TestNewUnsupportedTypeError_Constructor(t *testing.T) {
	err := NewUnsupportedTypeError("ComplexField", "complex128", "not supported")

	var typeErr *UnsupportedTypeError
	require.True(t, errors.As(err, &typeErr))
	assert.Equal(t, "ComplexField", typeErr.FieldName)
	assert.Equal(t, "complex128", typeErr.FieldType)
	assert.Equal(t, "not supported", typeErr.Message)
}

func TestFieldError_Interface_MultipleTypes(t *testing.T) {
	tests := []struct {
		name  string
		err   FieldError
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
			name: "MissingCustomHookError",
			err: &MissingCustomHookError{
				FieldName:    "CustomField",
				ExpectedHook: "DefineCustomField",
				TypeName:     "CustomType",
			},
			field: "CustomField",
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
			name: "UnsupportedTypeError",
			err: &UnsupportedTypeError{
				FieldName: "UnsupportedField",
				FieldType: "UnsupportedType",
				Message:   "not supported",
			},
			field: "UnsupportedField",
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
