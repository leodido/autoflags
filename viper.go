package autoflags

import (
	"fmt"
	"os"

	"github.com/go-viper/mapstructure/v2"
	"github.com/leodido/autoflags/options"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	vipers map[string]*viper.Viper = map[string]*viper.Viper{}
)

func GetViper(path string) *viper.Viper {
	reuse, ok := vipers[path]
	if !ok {
		vipers[path] = viper.New()

		return vipers[path]
	}

	return reuse
}

func Debug(c *cobra.Command, opts options.DebuggableOptions) error {
	if !opts.Debuggable() {
		return nil
	}

	res, ok := vipers[c.Name()]
	if !ok {
		return fmt.Errorf("couldn't find a viper instance for %s", c.Name())
	}
	res.Debug()
	fmt.Fprintf(os.Stdout, "Values:\n%#v\n", res.AllSettings())

	return nil
}

func Viper(c *cobra.Command) (*viper.Viper, error) {
	res, ok := vipers[c.Name()]
	if !ok {
		return nil, fmt.Errorf("couldn't find a viper instance for %s", c.Name())
	}

	return res, nil
}

// createConfigC creates a configuration map for a specific command by merging
// top-level settings with command-specific settings from the global configuration.
func createConfigC(globalSettings map[string]any, commandName string) map[string]any {
	configToMerge := make(map[string]any)

	// First, add all top-level settings (for root command and shared config)
	for key, value := range globalSettings {
		// Skip command-specific sections to avoid conflicts
		if _, isMap := value.(map[string]any); !isMap || key != commandName {
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
func Unmarshal(c *cobra.Command, opts options.Options, hooks ...mapstructure.DecodeHookFunc) error {
	res, err := Viper(c)
	if err != nil {
		return err
	}

	// Merging the config map (if any) from the global viper singleton instance
	configToMerge := createConfigC(viper.AllSettings(), c.Name())
	if err := res.MergeConfigMap(configToMerge); err != nil {
		return err
	}

	// Look for decode hook annotation appending them to the list of hooks to use for unmarshalling
	c.Flags().VisitAll(func(f *pflag.Flag) {
		if decodeHooks, defineDecodeHooks := f.Annotations[FlagDecodeHookAnnotation]; defineDecodeHooks {
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
		return err
	}

	// Automatically set common options into the context of the cobra command
	if o, ok := opts.(options.CommonOptions); ok {
		c.SetContext(o.Context(c.Context()))
	}

	// Automatically run options validation if feasible
	if o, ok := opts.(options.ValidatableOptions); ok {
		if validationErrors := o.Validate(); validationErrors != nil {
			ret := "invalid options" // FIXME: get name of the options
			for _, e := range validationErrors {
				ret += "\n       "
				ret += e.Error()
			}

			return fmt.Errorf("%s", ret)
		}
	}

	// FIXME: transform before validation?
	// Automatically transform options if feasible
	if o, ok := opts.(options.TransformableOptions); ok {
		if transformErr := o.Transform(c.Context()); transformErr != nil {
			return transformErr
		}
	}

	return nil
}
