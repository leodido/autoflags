package internalhooks

import (
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

const (
	FlagDecodeHookAnnotation = "___leodido_structcli_flagdecodehooks"
)

type DecodeHookFunc func(input any) (any, error)

type decodingAnnotation struct {
	ann string
	fx  mapstructure.DecodeHookFunc
}

var DecodeHookRegistry = map[string]decodingAnnotation{
	"time.Duration": {
		"StringToTimeDurationHookFunc",
		mapstructure.StringToTimeDurationHookFunc(),
	},
	"zapcore.Level": {
		"StringToZapcoreLevelHookFunc",
		StringToZapcoreLevelHookFunc(),
	},
	"slog.Level": {
		"StringToSlogLevelHookFunc",
		StringToSlogLevelHookFunc(),
	},
	"[]string": {
		"StringToSliceHookFunc",
		mapstructure.StringToSliceHookFunc(","),
	},
	"[]int": {
		"StringToIntSliceHookFunc",
		StringToIntSliceHookFunc(","),
	},
}

// AnnotationToDecodeHookRegistry maps annotation names to decode hook functions
var AnnotationToDecodeHookRegistry map[string]mapstructure.DecodeHookFunc

func init() {
	// Map annotations to decoding hook
	AnnotationToDecodeHookRegistry = make(map[string]mapstructure.DecodeHookFunc)
	for typename, data := range DecodeHookRegistry {
		if _, exists := AnnotationToDecodeHookRegistry[data.ann]; exists {
			panic(fmt.Sprintf("duplicate annotation name '%s' found in decode hook registry (type: %s)", data.ann, typename))
		}

		AnnotationToDecodeHookRegistry[data.ann] = data.fx
	}
}

func InferDecodeHooks(c *cobra.Command, name, typename string) bool {
	if data, ok := DecodeHookRegistry[typename]; ok {
		_ = c.Flags().SetAnnotation(name, FlagDecodeHookAnnotation, []string{data.ann})

		return true
	}

	return false
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

// StringToSlogLevelHookFunc creates a decode hook that converts string values
// to slog.Level types during configuration unmarshaling.
func StringToSlogLevelHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(slog.LevelInfo) {
			return data, nil
		}

		var level slog.Level
		err := level.UnmarshalText([]byte(data.(string)))
		if err != nil {
			return nil, fmt.Errorf("invalid string for slog.Level '%s': %w", data.(string), err)
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

func StoreDecodeHookFunc(c *cobra.Command, flagname string, decodeM reflect.Value, target reflect.Type) error {
	s := internalscope.Get(c)

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
	s.SetCustomDecodeHook(k, hookFunc)

	return c.Flags().SetAnnotation(flagname, FlagDecodeHookAnnotation, []string{k})
}
