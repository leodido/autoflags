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

// createConfigC creates a configuration map for a specific command.
//
// It merges the top-level settings with command-specific settings (which can be nested) from the global configuration.
func createConfigC(globalSettings map[string]any, c *cobra.Command) map[string]any {
	configToMerge := make(map[string]any)

	// First, add all top-level settings
	// This is for root command
	// It also serves as defaults that can be overridden by more specific command sections
	for key, value := range globalSettings {
		// Skip command-specific sections to avoid conflicts
		if _, isMap := value.(map[string]any); !isMap {
			configToMerge[key] = value
		}
	}

	var finalSettings map[string]any
	subpathC := strings.Split(c.CommandPath(), " ")[1:]
	currentLevel := globalSettings

	for _, part := range subpathC {
		if settings, ok := currentLevel[part]; ok {
			if settingsMap, isMap := settings.(map[string]any); isMap {
				// Move one level deeper for the next iteration
				currentLevel = settingsMap
				// Store the current level's settings as the most specific found so far
				finalSettings = settingsMap
			} else {
				// The path is broken by a non-map value, so we stop.
				finalSettings = nil // Invalidate to avoid merging partial path

				break
			}
		} else {
			// The path does not exist in the config, so we stop.
			finalSettings = nil // Invalidate

			break
		}
	}

	for key, value := range finalSettings {
		configToMerge[key] = value
	}

	return configToMerge
}

// Unmarshal populates the options struct with values from flags, environment variables,
// and configuration files.
//
// It automatically handles decode hooks, validation, transformation, and context updates based on the options type.
func Unmarshal(c *cobra.Command, opts Options, hooks ...mapstructure.DecodeHookFunc) error {
	scope := getScope(c)
	vip := scope.viper()

	// Merging the config map (if any) from the global viper singleton instance
	configToMerge := createConfigC(viper.AllSettings(), c)
	vip.MergeConfigMap(configToMerge)

	// Create the full alias-to-path map from its global cache
	aliasToPathMap := make(map[string]string)
	globalAliasCache.Range(func(k, v any) bool {
		aliasToPathMap[k.(string)] = v.(string)

		return true
	})

	// Create the defaults map from its global cache
	defaultsMap := make(map[string]string)
	globalDefaultsCache.Range(func(k, v any) bool {
		defaultsMap[k.(string)] = v.(string)

		return true
	})

	// Use `KeyRemappingHook` for smart config keys
	hooks = append([]mapstructure.DecodeHookFunc{KeyRemappingHook(aliasToPathMap, defaultsMap)}, hooks...)

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

	decodeHook := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		hooks...,
	))

	if err := vip.Unmarshal(opts /*custonNameHook,*/, decodeHook); err != nil {
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

// KeyRemappingHook allows config keys to match either a field's name or its `flag` tag.
//
// It correctly handles flattened keys that point to nested struct fields.
func KeyRemappingHook(aliasToPathMap map[string]string, defaultsMap map[string]string) mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data any) (any, error) {
		// Only when decoding a map into a struct...
		if f.Kind() != reflect.Map || t.Kind() != reflect.Struct {
			return data, nil
		}

		configMap, ok := data.(map[string]any)
		if !ok {
			return data, nil
		}

		// Hande flattened keys for nested structs
		for alias, path := range aliasToPathMap {
			// Find nested paths like "database.url"
			if strings.Contains(path, ".") {
				// Check if the flattened alias key exists at this level
				if aliasValue, ok := configMap[alias]; ok && aliasValue != "" {
					// Do not override the user-provided value with the default value
					if aliasDefaultValue, ok := defaultsMap[alias]; ok && aliasValue == aliasDefaultValue {
						continue
					}
					pathParts := strings.Split(path, ".")

					// Start at the top of the map
					currentMap := configMap

					// Walk the path down, creating nested maps as needed
					for i := range len(pathParts) - 1 {
						part := pathParts[i]

						var nextMap map[string]any
						if val, ok := currentMap[part]; ok {
							nextMap, _ = val.(map[string]any)
						}
						if nextMap == nil {
							nextMap = make(map[string]any)
							currentMap[part] = nextMap
						}
						// Move one level deeper
						currentMap = nextMap
					}

					// At the deepest level, set the final key and value
					finalKey := pathParts[len(pathParts)-1]
					currentMap[finalKey] = aliasValue

					// Delete the original flattened key as it has been moved
					delete(configMap, alias)
				}
			}
		}

		// For every field in the destination struct...
		for i := range t.NumField() {
			field := t.Field(i)
			fieldNameKey := strings.ToLower(field.Name)
			alias := field.Tag.Get("flag")

			// When the alias exists and the config map has a value for that alias...
			if alias != "" && alias != fieldNameKey {
				if aliasValue, ok := configMap[alias]; ok && aliasValue != "" {
					// Then, make the alias value available under the field name key.
					configMap[fieldNameKey] = aliasValue

					continue
				}
				if fieldNameVal, ok := configMap[fieldNameKey]; ok && fieldNameKey != "" {
					// Or, make the field value available under the alias.
					configMap[alias] = fieldNameVal

					continue
				}
				// This ensures the decoder can find the value for this field name key or the alias.
			}
		}

		return configMap, nil
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
