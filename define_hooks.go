package autoflags

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
	"unsafe"

	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag/v2"
	"go.uber.org/zap/zapcore"
)

// DefineHookFunc defines how to create a flag for a custom type.
//
// It receives the command, flag metadata, struct field information, and the field value to create specialized flag definitions beyond the standard types.
type DefineHookFunc func(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value)

// Registry for predefined flag definition functions
var defineHookRegistry = map[string]DefineHookFunc{
	"zapcore.Level": DefineZapcoreLevelHookFunc(),
	"time.Duration": DefineTimeDurationHookFunc(),
}

func DefineTimeDurationHookFunc() DefineHookFunc {
	return func(c *cobra.Command, name, short, descr string, _ reflect.StructField, fieldValue reflect.Value) {
		if !fieldValue.CanAddr() {
			return
		}

		val := fieldValue.Interface().(time.Duration)
		ref := (*time.Duration)(unsafe.Pointer(fieldValue.UnsafeAddr()))
		c.Flags().DurationVarP(ref, name, short, val, descr)
	}
}

// DefineZapcoreLevelHookFunc creates a flag definition function for zapcore.Level.
//
// It generates an enum flag with all valid log levels and proper shell completion.
func DefineZapcoreLevelHookFunc() DefineHookFunc {
	return func(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
		if !fieldValue.CanAddr() {
			return
		}

		logLevels := map[zapcore.Level][]string{
			zapcore.DebugLevel:  {"debug"},
			zapcore.InfoLevel:   {"info"},
			zapcore.WarnLevel:   {"warn"},
			zapcore.ErrorLevel:  {"error"},
			zapcore.DPanicLevel: {"dpanic"},
			zapcore.PanicLevel:  {"panic"},
			zapcore.FatalLevel:  {"fatal"},
		}

		keys := []int{}
		for k := range logLevels {
			keys = append(keys, int(k))
		}
		sort.Ints(keys)
		values := []string{}
		for _, k := range keys {
			values = append(values, logLevels[zapcore.Level(k)][0])
		}
		addendum := fmt.Sprintf(" {%s}", strings.Join(values, ","))

		// Get pointer to the field for the enum flag
		fieldPtr := (*zapcore.Level)(unsafe.Pointer(fieldValue.UnsafeAddr()))
		enumFlag := enumflag.New(fieldPtr, structField.Type.String(), logLevels, enumflag.EnumCaseInsensitive)
		c.Flags().VarP(enumFlag, name, short, descr+addendum)
	}
}

// inferDefineHooks checks if there's a predefined flag definition function for the given type
func inferDefineHooks(c *cobra.Command, typename string, structField reflect.StructField, name, short, descr string, fieldValue reflect.Value) bool {
	if defineFunc, ok := defineHookRegistry[typename]; ok {
		defineFunc(c, name, short, descr, structField, fieldValue)

		return true
	}

	return false
}
