package autoflags

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unsafe"

	"github.com/thediveo/enumflag/v2"
	"go.uber.org/zap/zapcore"
)

// DefineHookFunc defines how to create a flag for a custom type
type DefineHookFunc func(runCtx *defineContext, field reflect.StructField, name, short, descr string, fieldValue reflect.Value)

// Registry for predefined flag definition functions
var defineHookRegistry = map[string]DefineHookFunc{
	"zapcore.Level": DefineZapcoreLevelHookFunc(),
}

// DefineZapcoreLevelHookFunc creates a flag definition function for zapcore.Level
func DefineZapcoreLevelHookFunc() DefineHookFunc {
	return func(runCtx *defineContext, field reflect.StructField, name, short, descr string, fieldValue reflect.Value) {
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
		enumFlag := enumflag.New(fieldPtr, field.Type.String(), logLevels, enumflag.EnumCaseInsensitive)
		runCtx.targetF.VarP(enumFlag, name, short, descr+addendum)
	}
}

// defineHookFromRegistry checks if there's already a flag definition function for the given type
func (ctx *defineContext) defineHookFromRegistry(typename string, field reflect.StructField, name, short, descr string, fieldValue reflect.Value) bool {
	if defineFunc, ok := defineHookRegistry[typename]; ok {
		defineFunc(ctx, field, name, short, descr, fieldValue)

		return true
	}

	return false
}
