package errors

import (
	"errors"
	"fmt"
	"strings"
)

// ValidationError wraps multiple validation errors that occurred during ValidatableOptions unmarshalling.
type ValidationError struct {
	ContextName string
	Errors      []error
}

func (e *ValidationError) Error() string {
	var sb strings.Builder
	if e.ContextName != "" {
		sb.WriteString(fmt.Sprintf("invalid options for %s", e.ContextName))
	} else {
		sb.WriteString("invalid options")
	}
	if len(e.Errors) >= 1 {
		sb.WriteString(":")
	}

	for _, err := range e.Errors {
		sb.WriteString("\n       ")
		sb.WriteString(err.Error())
	}

	return sb.String()
}

// UnderlyingErrors returns the slice of individual validation errors (immutable).
func (e *ValidationError) UnderlyingErrors() []error {
	if e.Errors == nil {
		return nil
	}

	// Return a copy to prevent mutations
	result := make([]error, len(e.Errors))
	copy(result, e.Errors)

	return result
}

// These are all DefinitionError
var (
	ErrInvalidBooleanTag          = errors.New("invalid boolean tag value")
	ErrInvalidShorthand           = errors.New("invalid shorthand flag")
	ErrMissingDefineHook          = errors.New("missing custom flag definition hook")
	ErrMissingDecodeHook          = errors.New("missing custom flag decoding hook")
	ErrInvalidDefineHookSignature = errors.New("invalid define hook signature")
	ErrInvalidDecodeHookSignature = errors.New("invalid decode hook signature")
	ErrInvalidFlagName            = errors.New("invalid flag name")
	ErrInvalidTagUsage            = errors.New("invalid tag usage")
	ErrConflictingTags            = errors.New("conflicting struct tags")
	ErrUnsupportedType            = errors.New("unsupported field type")
)

// DefinitionError represents an error that occurred while processing a struct field's tags at definition time.
type DefinitionError interface {
	error
	Field() string
}

// InvalidBooleanTagError represents an invalid boolean value in struct tags
type InvalidBooleanTagError struct {
	FieldName string
	TagName   string
	TagValue  string
}

func (e *InvalidBooleanTagError) Error() string {
	return fmt.Sprintf("field '%s': tag '%s=%s': invalid boolean value", e.FieldName, e.TagName, e.TagValue)
}

func (e *InvalidBooleanTagError) Field() string {
	return e.FieldName
}

func (e *InvalidBooleanTagError) Unwrap() error {
	return ErrInvalidBooleanTag
}

// InvalidShorthandError represents an invalid shorthand flag specification
type InvalidShorthandError struct {
	FieldName string
	Shorthand string
}

func (e *InvalidShorthandError) Error() string {
	return fmt.Sprintf("field '%s': shorthand flag '%s' must be a single character", e.FieldName, e.Shorthand)
}

func (e *InvalidShorthandError) Field() string {
	return e.FieldName
}

func (e *InvalidShorthandError) Unwrap() error {
	return ErrInvalidShorthand
}

// MissingDefineHookError represents a missing custom flag definition hook
type MissingDefineHookError struct {
	FieldName    string
	ExpectedHook string
}

func (e *MissingDefineHookError) Error() string {
	return fmt.Sprintf("field '%s': flagcustom='true' but missing define hook '%s'", e.FieldName, e.ExpectedHook)
}

func (e *MissingDefineHookError) Field() string {
	return e.FieldName
}

func (e *MissingDefineHookError) Unwrap() error {
	return ErrMissingDefineHook
}

// MissingDecodeHookError represents a missing custom flag decoding hook
type MissingDecodeHookError struct {
	FieldName    string
	ExpectedHook string
}

func (e *MissingDecodeHookError) Error() string {
	return fmt.Sprintf("field '%s': flagcustom='true' but missing decode hook '%s'", e.FieldName, e.ExpectedHook)
}

func (e *MissingDecodeHookError) Field() string {
	return e.FieldName
}

func (e *MissingDecodeHookError) Unwrap() error {
	return ErrMissingDecodeHook
}

// InvalidDecodeHookSignatureError represents an invalid custom flag definition hook
type InvalidDecodeHookSignatureError struct {
	FieldName string
	HookName  string
	Message   string
}

func (e *InvalidDecodeHookSignatureError) Error() string {
	return fmt.Sprintf("field '%s': invalid '%s' decode hook: %s",
		e.FieldName, e.HookName, e.Message)
}

func (e *InvalidDecodeHookSignatureError) Field() string {
	return e.FieldName
}

func (e *InvalidDecodeHookSignatureError) Unwrap() error {
	return ErrInvalidDecodeHookSignature
}

// InvalidDefineHookSignatureError represents an invalid custom flag definition hook
type InvalidDefineHookSignatureError struct {
	FieldName string
	HookName  string
	Message   string
}

func (e *InvalidDefineHookSignatureError) Error() string {
	return fmt.Sprintf("field '%s': invalid '%s' define hook: %s",
		e.FieldName, e.HookName, e.Message)
}

func (e *InvalidDefineHookSignatureError) Field() string {
	return e.FieldName
}

func (e *InvalidDefineHookSignatureError) Unwrap() error {
	return ErrInvalidDefineHookSignature
}

// InvalidTagUsageError represents invalid tag usages
type InvalidTagUsageError struct {
	FieldName string
	TagName   string
	Message   string
}

func (e *InvalidTagUsageError) Error() string {
	return fmt.Sprintf("field '%s': invalid usage of tag '%s': %s", e.FieldName, e.TagName, e.Message)
}

func (e *InvalidTagUsageError) Field() string {
	return e.FieldName
}

func (e *InvalidTagUsageError) Unwrap() error {
	return ErrInvalidTagUsage
}

// ConflictingTagsError represents conflicting struct tag values
type ConflictingTagsError struct {
	FieldName       string
	Message         string
	ConflictingTags []string
}

func (e *ConflictingTagsError) Error() string {
	return fmt.Sprintf(
		"field '%s': conflicting tags [%s]: %s",
		e.FieldName,
		strings.Join(e.ConflictingTags, ", "),
		e.Message,
	)
}

func (e *ConflictingTagsError) Field() string {
	return e.FieldName
}
func (e *ConflictingTagsError) Unwrap() error {
	return ErrConflictingTags
}

// UnsupportedTypeError represents an unsupported field type
type UnsupportedTypeError struct {
	FieldName string
	FieldType string
	Message   string
}

func (e *UnsupportedTypeError) Error() string {
	return fmt.Sprintf("field '%s': unsupported type '%s': %s", e.FieldName, e.FieldType, e.Message)
}

func (e *UnsupportedTypeError) Field() string {
	return e.FieldName
}

func (e *UnsupportedTypeError) Unwrap() error {
	return ErrUnsupportedType
}

func NewInvalidBooleanTagError(fieldName, tagName, tagValue string) error {
	return &InvalidBooleanTagError{
		FieldName: fieldName,
		TagName:   tagName,
		TagValue:  tagValue,
	}
}

func NewInvalidShorthandError(fieldName, shorthand string) error {
	return &InvalidShorthandError{
		FieldName: fieldName,
		Shorthand: shorthand,
	}
}

func NewMissingDefineHookError(fieldName, hookName string) error {
	return &MissingDefineHookError{
		FieldName:    fieldName,
		ExpectedHook: hookName,
	}
}

func NewMissingDecodeHookError(fieldName, hookName string) error {
	return &MissingDecodeHookError{
		FieldName:    fieldName,
		ExpectedHook: hookName,
	}
}

func NewInvalidDecodeHookSignatureError(fieldName, hookName string, err error) error {
	return &InvalidDecodeHookSignatureError{
		FieldName: fieldName,
		HookName:  hookName,
		Message:   err.Error(),
	}
}

func NewInvalidDefineHookSignatureError(fieldName, hookName string, err error) error {
	return &InvalidDefineHookSignatureError{
		FieldName: fieldName,
		HookName:  hookName,
		Message:   err.Error(),
	}
}

func NewInvalidTagUsageError(fieldName, tagName, message string) error {
	return &InvalidTagUsageError{
		FieldName: fieldName,
		TagName:   tagName,
		Message:   message,
	}
}

func NewConflictingTagsError(fieldName string, tags []string, message string) error {
	return &ConflictingTagsError{
		FieldName:       fieldName,
		ConflictingTags: tags,
		Message:         message,
	}
}

func NewUnsupportedTypeError(fieldName, fieldType, message string) error {
	return &UnsupportedTypeError{
		FieldName: fieldName,
		FieldType: fieldType,
		Message:   message,
	}
}

var ErrInputValue = errors.New("invalid input value")

// InputError represents an invalid input value for flag definition
type InputError struct {
	InputType string
	Message   string
}

func (e *InputError) Error() string {
	return fmt.Sprintf("invalid input value of type '%s': %s", e.InputType, e.Message)
}

func (e *InputError) Unwrap() error {
	return ErrInputValue
}

// Add this constructor function after the existing constructor functions
func NewInputError(inputType, message string) error {
	return &InputError{
		InputType: inputType,
		Message:   message,
	}
}
