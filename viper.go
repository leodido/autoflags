package autoflags

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	autoflagserrors "github.com/leodido/autoflags/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// GetViper returns the viper instance associated with the given command.
//
// Each command has its own isolated viper instance for configuration management.
func GetViper(c *cobra.Command) *viper.Viper {
	s := getScope(c)

	return s.viper()
}

// createConfigC creates a configuration map for a specific command by merging
// top-level settings with command-specific settings from the global configuration.
func createConfigC(globalSettings map[string]any, commandName string) map[string]any {
	configToMerge := make(map[string]any)

	// First, add all top-level settings (for root command and shared config)
	for key, value := range globalSettings {
		// Skip command-specific sections to avoid conflicts
		if _, isMap := value.(map[string]any); !isMap {
			configToMerge[key] = value
		}
	}

	// Then, if there's a command-specific section, promote its contents to top level
	if commandSettings, exists := globalSettings[commandName]; exists {
		if commandMap, ok := commandSettings.(map[string]any); ok {
			for key, value := range commandMap {
				configToMerge[key] = value
			}
		}
	}

	return configToMerge
}

// Unmarshal populates the options struct with values from flags, environment variables,
// and configuration files.
//
// It automatically handles decode hooks, validation, transformation, and context updates based on the options type.
// NOTE: See https://github.com/spf13/viper/pull/1715
func Unmarshal(c *cobra.Command, opts Options, hooks ...mapstructure.DecodeHookFunc) error {
	scope := getScope(c)
	vip := scope.viper()

	// Merging the config map (if any) from the global viper singleton instance
	configToMerge := createConfigC(viper.AllSettings(), c.Name())
	vip.MergeConfigMap(configToMerge)

	// Look for decode hook annotation appending them to the list of hooks to use for unmarshalling
	c.Flags().VisitAll(func(f *pflag.Flag) {
		if decodeHooks, defineDecodeHooks := f.Annotations[flagDecodeHookAnnotation]; defineDecodeHooks {
			for _, decodeHook := range decodeHooks {
				// Custom decode hook have precedence
				if customDecodeHook, customDecodeHookExists := scope.getCustomDecodeHook(decodeHook); customDecodeHookExists {
					hooks = append(hooks, customDecodeHook)

					continue
				}

				// Check the registry for built-in decode hooks
				if decodeHookFunc, ok := annotationToDecodeHookRegistry[decodeHook]; ok {
					hooks = append(hooks, decodeHookFunc)
				}
			}
		}
	})

	custonNameHook := viper.DecoderConfigOption(func(c *mapstructure.DecoderConfig) {
		// The destination struct.
		c.Result = opts

		// This enables conversion of strings to bool, int, etc.
		c.WeaklyTypedInput = true

		// This is the custom matching logic that solves the problem.
		c.MatchName = getNameMatcher()
	})

	decodeHook := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		hooks...,
	))

	if err := vip.Unmarshal(opts, custonNameHook, decodeHook); err != nil {
		return fmt.Errorf("couldn't unmarshal config to options: %w", err)
	}

	// Automatically set common options into the context of the cobra command
	if o, ok := opts.(ContextOptions); ok {
		c.SetContext(o.Context(c.Context()))
	}

	// Automatically transform options if feasible
	if o, ok := opts.(TransformableOptions); ok {
		if transformErr := o.Transform(c.Context()); transformErr != nil {
			return fmt.Errorf("couldn't transform options: %w", transformErr)
		}
	}

	// Automatically run options validation if feasible
	if o, ok := opts.(ValidatableOptions); ok {
		if validationErrors := o.Validate(c.Context()); validationErrors != nil {
			return &autoflagserrors.ValidationError{
				ContextName: c.Name(),
				Errors:      validationErrors,
			}
		}
	}

	syncMandatoryFlags(c, reflect.TypeOf(opts), vip, "")

	// Automatic debug output if debug is on
	UseDebug(c, c.OutOrStdout())

	return nil
}

func getNameMatcher() func(mapKey, fieldName string) bool {
	// Build the mapping of field names to their `flag` tag aliases.
	fieldMappings := make(map[string]string)
	globalFieldMappingsCache.Range(func(key, value interface{}) bool {
		fieldMappings[key.(string)] = value.(string)
		return true
	})

	return func(mapKey, fieldName string) bool {
		// First, check for a direct case-insensitive match (default behavior).
		if strings.EqualFold(mapKey, fieldName) {
			return true
		}
		// If that fails, check if the mapKey matches the field's `flag` tag alias.
		if alias, ok := fieldMappings[strings.ToLower(fieldName)]; ok {
			if strings.EqualFold(mapKey, alias) {
				return true
			}
		}
		return false
	}
}

// syncMandatoryFlags tells cobra that a required flag is present when its value is provided by a source other than the command line (e.g., config file).
func syncMandatoryFlags(c *cobra.Command, T reflect.Type, vip *viper.Viper, structPath string) {
	if T.Kind() == reflect.Ptr {
		T = T.Elem()
	}
	if T.Kind() != reflect.Struct {
		return
	}

	for i := range T.NumField() {
		structField := T.Field(i)

		// Path calculation logic
		path := getFieldPath(structPath, structField)

		// Recurse into nested structs first
		if structField.Type.Kind() == reflect.Struct {
			syncMandatoryFlags(c, structField.Type, vip, path)
		}

		// Go on only for mandatory fields
		if !isMandatory(structField) {
			continue
		}

		// Determine the flag name (which is the viper key)
		alias := structField.Tag.Get("flag")
		name := getName(path, alias)

		// If viper has a value for this key, find the corresponding
		// cobra flag and mark its Changed property as true.
		if vip.IsSet(name) {
			if f := c.Flags().Lookup(name); f != nil {
				f.Changed = true
			}
		}
	}
}
