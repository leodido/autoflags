package structcli

import (
	"fmt"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	structclierrors "github.com/leodido/structcli/errors"
	internalconfig "github.com/leodido/structcli/internal/config"
	internalhooks "github.com/leodido/structcli/internal/hooks"
	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// GetViper returns the viper instance associated with the given command.
//
// Each command has its own isolated viper instance for configuration management.
func GetViper(c *cobra.Command) *viper.Viper {
	s := internalscope.Get(c)

	return s.Viper()
}

// Unmarshal populates the options struct with values from flags, environment variables,
// and configuration files.
//
// It automatically handles decode hooks, validation, transformation, and context updates based on the options type.
func Unmarshal(c *cobra.Command, opts Options, hooks ...mapstructure.DecodeHookFunc) error {
	scope := internalscope.Get(c)
	vip := scope.Viper()

	// Merging the config map (if any) from the global viper singleton instance
	configToMerge := internalconfig.Merge(viper.AllSettings(), c)
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
	hooks = append([]mapstructure.DecodeHookFunc{internalconfig.KeyRemappingHook(aliasToPathMap, defaultsMap)}, hooks...)

	// Look for decode hook annotation appending them to the list of hooks to use for unmarshalling
	c.Flags().VisitAll(func(f *pflag.Flag) {
		if decodeHooks, defineDecodeHooks := f.Annotations[internalhooks.FlagDecodeHookAnnotation]; defineDecodeHooks {
			for _, decodeHook := range decodeHooks {
				// Custom decode hook have precedence
				if customDecodeHook, customDecodeHookExists := scope.GetCustomDecodeHook(decodeHook); customDecodeHookExists {
					hooks = append(hooks, customDecodeHook)

					continue
				}

				// Check the registry for built-in decode hooks
				if decodeHookFunc, ok := internalhooks.AnnotationToDecodeHookRegistry[decodeHook]; ok {
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
			return &structclierrors.ValidationError{
				ContextName: c.Name(),
				Errors:      validationErrors,
			}
		}
	}

	internalconfig.SyncMandatoryFlags(c, reflect.TypeOf(opts), vip, "")

	// Automatic debug output if debug is on
	UseDebug(c, c.OutOrStdout())

	return nil
}
