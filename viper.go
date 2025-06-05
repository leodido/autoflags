package autoflags

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

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

// NOTE: See https://github.com/spf13/viper/pull/1715
func Unmarshal(c *cobra.Command, opts Options, hooks ...mapstructure.DecodeHookFunc) error {
	res := GetViper(c)

	// Merging the config map (if any) from the global viper singleton instance
	configToMerge := createConfigC(viper.AllSettings(), c.Name())
	res.MergeConfigMap(configToMerge)

	// Look for decode hook annotation appending them to the list of hooks to use for unmarshalling
	c.Flags().VisitAll(func(f *pflag.Flag) {
		if decodeHooks, defineDecodeHooks := f.Annotations[flagDecodeHookAnnotation]; defineDecodeHooks {
			for _, decodeHook := range decodeHooks {
				if decodeHookFunc, ok := decodeHookRegistry[decodeHook]; ok {
					hooks = append(hooks, decodeHookFunc)
				}
			}
		}
	})

	decodeHook := viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		hooks...,
	))
	if err := res.Unmarshal(opts, decodeHook); err != nil {
		return fmt.Errorf("couldn't unmarshal config to options: %w", err)
	}

	// Automatically set common options into the context of the cobra command
	if o, ok := opts.(CommonOptions); ok {
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
		if validationErrors := o.Validate(); validationErrors != nil {
			return &ValidationError{
				ContextName: c.Name(),
				Errors:      validationErrors,
			}
		}
	}

	// Automatic debug output if debug is on
	UseDebug(c, c.OutOrStdout())

	return nil
}
