package autoflags

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
	"unsafe"

	autoflagsvalues "github.com/leodido/autoflags/values"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/thediveo/enumflag/v2"
	"go.uber.org/zap/zapcore"
)

// FIXME: remove short from the signature?

// DefineHookFunc defines how to create a flag for a custom type.
//
// It receives flag metadata and struct field information and must return a pflag.Value
// that knows how to set the underlying field's value, along with an optional enhanced
// description for the flag's usage message.
type DefineHookFunc func(name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) (pflag.Value, string)

// Registry for predefined flag definition functions
var defineHookRegistry = map[string]DefineHookFunc{
	"zapcore.Level": DefineZapcoreLevelHookFunc(),
	"time.Duration": DefineTimeDurationHookFunc(),
}

func DefineTimeDurationHookFunc() DefineHookFunc {
	return func(name, short, descr string, _ reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
		val := fieldValue.Interface().(time.Duration)
		ref := (*time.Duration)(unsafe.Pointer(fieldValue.UnsafeAddr()))

		return autoflagsvalues.NewDuration(val, ref), descr
	}
}

// DefineZapcoreLevelHookFunc creates a flag definition function for zapcore.Level.
//
// It returns an enum flag that implements pflag.Value.
func DefineZapcoreLevelHookFunc() DefineHookFunc {
	return func(name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
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
		enhancedDescr := descr + addendum

		// Get pointer to the field for the enum flag
		fieldPtr := (*zapcore.Level)(unsafe.Pointer(fieldValue.UnsafeAddr()))
		enumFlag := enumflag.New(fieldPtr, structField.Type.String(), logLevels, enumflag.EnumCaseInsensitive)

		return enumFlag, enhancedDescr
	}
}

// inferDefineHooks checks if there's a predefined flag definition function for the given type
func inferDefineHooks(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) bool {
	if defineFunc, ok := defineHookRegistry[structField.Type.String()]; ok {
		value, usage := defineFunc(name, short, descr, structField, fieldValue)
		c.Flags().VarP(value, name, short, usage)

		return true
	}

	return false
}
