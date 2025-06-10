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
	if len(e.Errors) > 1 {
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

var (
	ErrInvalidBooleanTag = errors.New("invalid boolean tag value")
	ErrInvalidShorthand  = errors.New("invalid shorthand flag")
	ErrMissingCustomHook = errors.New("missing custom flag definition hook")
	ErrInvalidFlagName   = errors.New("invalid flag name")
	ErrConflictingTags   = errors.New("conflicting struct tags")
	ErrUnsupportedType   = errors.New("unsupported field type")
)

// FieldError represents an error that occurred while processing a struct field's tags at definition time.
type FieldError interface {
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

// MissingCustomHookError represents a missing custom flag definition hook
type MissingCustomHookError struct {
	FieldName    string
	ExpectedHook string
	TypeName     string
}

func (e *MissingCustomHookError) Error() string {
	return fmt.Sprintf("field '%s': flagcustom='true' but hook '%s' not found for type '%s'",
		e.FieldName, e.ExpectedHook, e.TypeName)
}

func (e *MissingCustomHookError) Field() string {
	return e.FieldName
}

func (e *MissingCustomHookError) Unwrap() error {
	return ErrMissingCustomHook
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

func NewMissingCustomHookError(fieldName, hookName, typeName string) error {
	return &MissingCustomHookError{
		FieldName:    fieldName,
		ExpectedHook: hookName,
		TypeName:     typeName,
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
