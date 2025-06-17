package values

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringValue(t *testing.T) {
	t.Run("Set_Valid", func(t *testing.T) {
		var s string
		sv := NewString(&s)

		err := sv.Set("hello world")
		require.NoError(t, err)

		assert.Equal(t, "hello world", s, "underlying string should be updated")
		assert.Equal(t, "hello world", sv.String(), "String() should return the new value")
	})

	t.Run("Type", func(t *testing.T) {
		sv := NewString(new(string))
		assert.Equal(t, "string", sv.Type())
	})
}

func TestIntValue(t *testing.T) {
	t.Run("Set_Valid", func(t *testing.T) {
		var i int = 0
		iv := NewInt(&i)

		err := iv.Set("12345")
		require.NoError(t, err)

		assert.Equal(t, 12345, i, "underlying int should be updated")
		assert.Equal(t, "12345", iv.String(), "String() should return the new value")
	})

	t.Run("Set_Invalid", func(t *testing.T) {
		var i int = 42 // Start with a known value
		iv := NewInt(&i)

		err := iv.Set("not-a-number")
		require.Error(t, err, "Set should return an error for invalid input")

		assert.Equal(t, 42, i, "underlying int should not be changed on error")
	})

	t.Run("Type", func(t *testing.T) {
		iv := NewInt(new(int))
		assert.Equal(t, "int", iv.Type())
	})
}

func TestDurationValue(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		var d time.Duration
		defaultValue := 5 * time.Second
		dv := NewDuration(defaultValue, &d)

		assert.Equal(t, defaultValue, d, "NewDuration should set the initial value of the pointer")
		assert.Equal(t, "5s", dv.String(), "String() should return the initial value")
	})

	t.Run("Set_Valid", func(t *testing.T) {
		var d time.Duration
		dv := NewDuration(0, &d) // Initialize with zero

		err := dv.Set("1m30s")
		require.NoError(t, err)

		expected := 90 * time.Second
		assert.Equal(t, expected, d, "underlying duration should be updated")
		assert.Equal(t, "1m30s", dv.String(), "String() should return the new value")
	})

	t.Run("Set_Invalid", func(t *testing.T) {
		var d time.Duration
		initialValue := 10 * time.Second
		dv := NewDuration(initialValue, &d)

		err := dv.Set("invalid-duration")
		require.Error(t, err, "Set should return an error for invalid input")

		assert.Equal(t, initialValue, d, "underlying duration should not be changed on error")
	})

	t.Run("Type", func(t *testing.T) {
		dv := NewDuration(0, new(time.Duration))
		assert.Equal(t, "duration", dv.Type())
	})
}
