package autoflags

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

const (
	flagDecodeHookAnnotation = "___leodido_autoflags_flagdecodehooks"
)

type DecodeHookFunc func(input any) (any, error)

var decodeHookRegistry = map[string]mapstructure.DecodeHookFunc{
	"StringToZapcoreLevelHookFunc": StringToZapcoreLevelHookFunc(),
	"StringToTimeDurationHookFunc": mapstructure.StringToTimeDurationHookFunc(),
	"StringToSliceHookFunc":        mapstructure.StringToSliceHookFunc(","),
	"StringToIntSliceHookFunc":     StringToIntSliceHookFunc(","),
}

func inferDecodeHooks(c *cobra.Command, name, typename string) bool {
	switch typename {
	case "time.Duration":
		_ = c.Flags().SetAnnotation(name, flagDecodeHookAnnotation, []string{"StringToTimeDurationHookFunc"})

		return true
	case "zapcore.Level":
		_ = c.Flags().SetAnnotation(name, flagDecodeHookAnnotation, []string{"StringToZapcoreLevelHookFunc"})

		return true
	case "[]string":
		_ = c.Flags().SetAnnotation(name, flagDecodeHookAnnotation, []string{"StringToSliceHookFunc"})

		return true
	case "[]int":
		_ = c.Flags().SetAnnotation(name, flagDecodeHookAnnotation, []string{"StringToIntSliceHookFunc"})

		return true
	default:
		return false
	}
}

// StringToZapcoreLevelHookFunc creates a decode hook that converts string values
// to zapcore.Level types during configuration unmarshaling.
func StringToZapcoreLevelHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
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

// StringToIntSliceHookFunc creates a decode hook that converts comma-separated
// string values to []int slices during configuration unmarshaling.
func StringToIntSliceHookFunc(sep string) mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data any,
	) (any, error) {
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

func storeDecodeHookFunc(c *cobra.Command, flagname string, decodeM reflect.Value, target reflect.Type) error {
	s := getScope(c)

	// Wrap that adapts user method to mapstructure.DecodeHookFuncType signature
	hookFunc := func(from reflect.Type, to reflect.Type, data any) (any, error) {
		// Only apply this hook to the specific target type
		if to != target {
			return data, nil
		}

		// Only convert from string env var and config file values
		// They always come as strings
		if from.Kind() != reflect.String {
			return data, nil
		}

		// Call user's decode hook: DecodeX(input interface{}) (target, error)
		results := decodeM.Call([]reflect.Value{reflect.ValueOf(data)})

		if len(results) != 2 {
			return nil, fmt.Errorf("user decode method must return (value, error)")
		}

		// Check if error is not nil
		if !results[1].IsNil() {
			return nil, results[1].Interface().(error)
		}

		return results[0].Interface(), nil
	}

	k := fmt.Sprintf("customDecodeHook_%s_%s", c.Name(), flagname)
	s.setCustomDecodeHook(k, hookFunc)

	return c.Flags().SetAnnotation(flagname, flagDecodeHookAnnotation, []string{k})
}
