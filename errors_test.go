package autoflags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldError_ErrorMessage(t *testing.T) {
	err := &FieldError{
		FieldName: "InvalidCustom",
		TagName:   "flagcustom",
		TagValue:  "invalid",
		Message:   "invalid boolean value",
	}

	expected := "field 'InvalidCustom': tag 'flagcustom=invalid': invalid boolean value"
	assert.Equal(t, expected, err.Error())
}

func TestFieldError_ContainsExpectedStrings(t *testing.T) {
	err := &FieldError{
		FieldName: "SomeField",
		TagName:   "flagcustom",
		TagValue:  "bad_value",
		Message:   "parsing error",
	}

	errorMsg := err.Error()

	// These are the strings our flagcustom test expects to find
	assert.Contains(t, errorMsg, "SomeField")
	assert.Contains(t, errorMsg, "flagcustom")
	assert.Contains(t, errorMsg, "bad_value")
}
