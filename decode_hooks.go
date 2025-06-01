package autoflags

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"go.uber.org/zap/zapcore"
)

const (
	FlagDecodeHookAnnotation = "___flagdecodehooks"
)

var decodeHookRegistry = map[string]mapstructure.DecodeHookFunc{
	"StringToZapcoreLevelHookFunc": StringToZapcoreLevelHookFunc(),
	"StringToSliceHookFunc":        mapstructure.StringToSliceHookFunc(","),
	"StringToTimeDurationHookFunc": mapstructure.StringToTimeDurationHookFunc(),
	"StringToIntSliceHookFunc":     StringToIntSliceHookFunc(","),
}

func (ctx *defineContext) decodeHookFromRegistry(name, typename string) {
	switch typename {
	case "time.Duration":
		_ = ctx.targetF.SetAnnotation(name, FlagDecodeHookAnnotation, []string{"StringToTimeDurationHookFunc"})
	case "zapcore.Level":
		_ = ctx.targetF.SetAnnotation(name, FlagDecodeHookAnnotation, []string{"StringToZapcoreLevelHookFunc"})
	case "[]string":
		_ = ctx.targetF.SetAnnotation(name, FlagDecodeHookAnnotation, []string{"StringToSliceHookFunc"})
	case "[]int":
		_ = ctx.targetF.SetAnnotation(name, FlagDecodeHookAnnotation, []string{"StringToIntSliceHookFunc"})
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

		level, err := zapcore.ParseLevel(data.(string))
		if err != nil {
			return nil, fmt.Errorf("invalid string for zapcore.Level '%s': %w", data.(string), err)
		}

		return level, nil
	}
}

func StringToIntSliceHookFunc(sep string) mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.SliceOf(reflect.TypeOf(int(0))) {
			return data, nil
		}

		raw := data.(string)
		if raw == "" {
			return []int{}, nil
		}

		parts := strings.Split(raw, sep)
		result := make([]int, len(parts))

		for i, part := range parts {
			trimmed := strings.TrimSpace(part)
			num, err := strconv.Atoi(trimmed)
			if err != nil {
				return nil, fmt.Errorf("invalid integer '%s' at position %d: %w", trimmed, i, err)
			}
			result[i] = num
		}

		return result, nil
	}
}
