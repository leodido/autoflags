package internalreflect

import (
	"fmt"
	"reflect"
	"strings"

	autoflagserrors "github.com/leodido/autoflags/errors"
)

func GetValue(o any) reflect.Value {
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

func GetValuePtr(o any) reflect.Value {
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

// GetStructPtr is a helper that gets a pointer to a struct value.
//
// Similar to getValuePtr but works with reflect.Value directly.
func GetStructPtr(structValue reflect.Value) reflect.Value {
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

// GetValidValue attempts to get a valid reflect.Value from the input object.
//
// It handles untyped nil as an error (no type information available).
// For typed nil pointers, it uses a fallback approach to create zero values.
//
// Returns an error if no valid Value can be obtained.
func GetValidValue(o any) (reflect.Value, error) {
	// Handle untyped nil
	if o == nil {
		return reflect.Value{}, autoflagserrors.NewInputError("nil", "cannot define flags from nil value")
	}

	val := GetValue(o)
	if !val.IsValid() {
		// Try the fallback approach for cases like typed nil pointers
		// This allows us to create zero values from type information
		valPtr := GetValuePtr(o)
		if !valPtr.IsValid() {
			// This should not happen for valid typed inputs
			inputType := fmt.Sprintf("%T", o)

			return reflect.Value{}, autoflagserrors.NewInputError(inputType, "cannot obtain valid reflection value")
		}

		// Only call Interface() if we have a valid value
		val = GetValue(valPtr.Interface())
		if !val.IsValid() {
			// This should also not happen for valid inputs
			inputType := fmt.Sprintf("%T", o)

			return reflect.Value{}, autoflagserrors.NewInputError(inputType, "fallback reflection approach failed")
		}
	}

	return val, nil
}

func Signature(f any) string {
	t := reflect.TypeOf(f)
	if t.Kind() != reflect.Func {
		return "<not a function>"
	}

	buf := strings.Builder{}
	buf.WriteString("func(")

	// Input parameters
	inParams := []string{}
	for i := range t.NumIn() {
		inParams = append(inParams, t.In(i).String())
	}
	buf.WriteString(strings.Join(inParams, ", "))
	buf.WriteString(")")

	// Output parameters
	if numOut := t.NumOut(); numOut > 0 {
		buf.WriteString(" (")
		outParams := []string{}
		for i := range t.NumOut() {
			outParams = append(outParams, t.Out(i).String())
		}
		buf.WriteString(strings.Join(outParams, ", "))
		buf.WriteString(")")
	}

	return buf.String()
}
