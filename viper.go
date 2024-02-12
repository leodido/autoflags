package autoflags

import (
	"fmt"

	"github.com/leodido/autoflags/options"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	vipers map[*cobra.Command]*viper.Viper = map[*cobra.Command]*viper.Viper{}
)

func Viper(c *cobra.Command) (*viper.Viper, error) {
	res, ok := vipers[c]
	if !ok {
		return nil, fmt.Errorf("couldn't find a viper instance for %s", c.Name())
	}

	return res, nil
}

// NOTE: See https://github.com/spf13/viper/pull/1715
func Unmarshal(c *cobra.Command, opts options.Options, hooks ...mapstructure.DecodeHookFunc) error {
	res, err := Viper(c)
	if err != nil {
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

			return fmt.Errorf(ret)
		}
	}

	// Automatically transform options if feasible
	if o, ok := opts.(options.TransformableOptions); ok {
		if transformErr := o.Transform(c.Context()); transformErr != nil {
			return transformErr
		}
	}

	return nil
}
