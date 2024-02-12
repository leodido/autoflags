package autoflags

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

const (
	FlagDecodeHookAnnotation = "___flagdecodehooks"
)

var decodeHookRegistry = map[string]mapstructure.DecodeHookFunc{
	"StringToZapcoreLevelHookFunc": StringToZapcoreLevelHookFunc(),
}

func inferDecodeHooks(c *cobra.Command, name, typename string) {
	switch typename {
	case "zapcore.Level":
		_ = c.Flags().SetAnnotation(name, FlagDecodeHookAnnotation, []string{"StringToZapcoreLevelHookFunc"})
	}
}

type DecodeHookFuncType func(reflect.Type, reflect.Type, interface{}) (interface{}, error)

func StringToZapcoreLevelHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(zapcore.DebugLevel) {
			return data, nil
		}

		return zapcore.ParseLevel(data.(string))
	}
}
