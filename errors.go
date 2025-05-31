package autoflags

import "fmt"

// FieldError represents an error that occurred while processing a struct field
type FieldError struct {
	FieldName string
	TagName   string
	TagValue  string
	Message   string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("field '%s': tag '%s=%s': %s",
		e.FieldName, e.TagName, e.TagValue, e.Message)
}
