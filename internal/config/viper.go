package internalconfig

import (
	"reflect"
	"strings"

	"maps"

	"github.com/go-viper/mapstructure/v2"
	internalpath "github.com/leodido/autoflags/internal/path"
	internaltag "github.com/leodido/autoflags/internal/tag"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Merge creates a configuration map for a specific command.
//
// It merges the top-level settings with command-specific settings (which can be nested) from the global configuration.
func Merge(globalSettings map[string]any, c *cobra.Command) map[string]any {
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

	maps.Copy(configToMerge, finalSettings)

	return configToMerge
}

// SyncMandatoryFlags tells cobra that a required flag is present when its value is provided by a source other than the command line (e.g., config file).
func SyncMandatoryFlags(c *cobra.Command, T reflect.Type, vip *viper.Viper, structPath string) {
	if T.Kind() == reflect.Ptr {
		T = T.Elem()
	}
	if T.Kind() != reflect.Struct {
		return
	}

	for i := range T.NumField() {
		structField := T.Field(i)

		// Path calculation logic
		path := internalpath.GetFieldPath(structPath, structField)

		// Recurse into nested structs first
		if structField.Type.Kind() == reflect.Struct {
			SyncMandatoryFlags(c, structField.Type, vip, path)
		}

		// Go on only for mandatory fields
		if !internaltag.IsMandatory(structField) {
			continue
		}

		// Determine the flag name (which is the viper key)
		alias := structField.Tag.Get("flag")
		name := internalpath.GetName(path, alias)

		// If viper has a value for this key, find the corresponding
		// cobra flag and mark its Changed property as true.
		if vip.IsSet(name) {
			if f := c.Flags().Lookup(name); f != nil {
				f.Changed = true
			}
		}
	}
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
