package autoflags

import (
	"fmt"
	"strings"
)

// FieldError represents an error that occurred while processing a struct field's tags.
type FieldError struct {
	FieldName string
	TagName   string
	TagValue  string
	Message   string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("field '%s': tag '%s=%s': %s", e.FieldName, e.TagName, e.TagValue, e.Message)
}

// ValidationError wraps multiple validation errors that occurred during option validation.
type ValidationError struct {
	ContextName string
	Errors      []error
}

func (e *ValidationError) Error() string {
	var sb strings.Builder
	if e.ContextName != "" {
		sb.WriteString(fmt.Sprintf("invalid options for %s:", e.ContextName))
	} else {
		sb.WriteString("invalid options:")
	}

	for _, err := range e.Errors {
		sb.WriteString("\n       ")
		sb.WriteString(err.Error())
	}
	return sb.String()
}

// UnderlyingErrors returns the slice of individual validation errors.
func (e *ValidationError) UnderlyingErrors() []error {
	return e.Errors
}
