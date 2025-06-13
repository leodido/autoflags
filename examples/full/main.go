package main

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/validator/v10"
	"github.com/leodido/autoflags"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

type Environment string

const (
	EnvDevelopment Environment = "dev"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "prod"
)

type ServerOptions struct {
	// Basic flags
	Host string `flag:"host" flagdescr:"Server host" default:"localhost"`
	Port int    `flagshort:"p" flagdescr:"Server port" flagrequired:"true"`

	// Environment variable binding
	APIKey string `flagenv:"true" flagdescr:"API authentication key"`

	// Flag grouping for organized help
	LogLevel zapcore.Level `flag:"log-level" flaggroup:"Logging" flagdescr:"Set log level"`
	LogFile  string        `flag:"log-file" flaggroup:"Logging" flagdescr:"Log file path"`

	// Nested structs for organization
	Database DatabaseConfig `flaggroup:"Database"`

	// Custom type
	TargetEnv Environment `flagcustom:"true" flag:"target-env" flagdescr:"Set the target environment"`
}

type DatabaseConfig struct {
	URL      string `flag:"db-url" flagdescr:"Database connection URL"`
	MaxConns int    `flagdescr:"Max database connections" default:"10" flagenv:"true"`
}

// DefineTargetEnv defines the custom flag for Environment with autocompletion
func (o *ServerOptions) DefineTargetEnv(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	enhancedDesc := descr + " {dev,staging,prod}"
	c.Flags().StringP(name, short, "dev", enhancedDesc)

	c.RegisterFlagCompletionFunc(name, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"dev\tDevelopment environment",
			"staging\tStaging environment",
			"prod\tProduction environment",
		}, cobra.ShellCompDirectiveNoFileComp
	})
}

// DecodeTargetEnv converts string input to Environment type with validation
func (o *ServerOptions) DecodeTargetEnv(input any) (any, error) {
	var strValue string

	switch v := input.(type) {
	case string:
		strValue = v
	case *string:
		if v != nil {
			strValue = *v
		}
	default:
		return nil, fmt.Errorf("expected string input for environment, got %T", input)
	}

	switch strings.ToLower(strings.TrimSpace(strValue)) {
	case "dev", "development":
		return EnvDevelopment, nil
	case "staging", "stage":
		return EnvStaging, nil
	case "prod", "production":
		return EnvProduction, nil
	default:
		return nil, fmt.Errorf("invalid environment: %s (one of: dev, staging, prod)", strValue)
	}
}

// Attach makes ServerOptions implement the Options interface
func (o *ServerOptions) Attach(c *cobra.Command) {
	autoflags.Define(c, o)
}

func makeSrvC() *cobra.Command {
	commonOpts := &UtilityFlags{}
	opts := &ServerOptions{}

	srvC := &cobra.Command{
		Use:   "srv",
		Short: "Start the server",
		Long:  "Start the server with the specified configuration",
		PreRunE: func(c *cobra.Command, args []string) error {
			fmt.Println("|--srvC.PreRunE")
			if err := autoflags.Unmarshal(c, opts); err != nil {
				return err
			}
			spew.Dump(opts)

			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Println("|--srvC.RunE")

			return nil
		},
	}
	opts.Attach(srvC)

	versionC := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Println("|---versionC.PersistentPreRunE")
			if err := commonOpts.FromContext(c.Context()); err != nil {
				return err
			}
			spew.Dump(commonOpts)

			return nil
		},
	}

	commonOpts.Attach(versionC)
	srvC.AddCommand(versionC)

	return srvC
}

var _ autoflags.ValidatableOptions = (*UserConfig)(nil)
var _ autoflags.TransformableOptions = (*UserConfig)(nil)

type UserConfig struct {
	Email string `flag:"email" flagdescr:"User email" validate:"email"`
	Age   int    `flag:"age" flagdescr:"User age" validate:"min=18,max=120"`
	Name  string `flag:"name" flagdescr:"User name" mod:"trim,title"`
}

// Transform makes UserConfig implement the ValidatableOptions interface
//
// The UserConfig options (flags/envs/configs) will be validated at unmarshalling time.
func (o *UserConfig) Validate(ctx context.Context) []error {
	var errs []error
	err := validator.New().Struct(o)
	if err != nil {
		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			for _, fieldErr := range validationErrs {
				errs = append(errs, fieldErr)
			}
		} else {
			errs = append(errs, fmt.Errorf("validator.Struct() failed unexpectedly: %w", err))
		}
	}
	if len(errs) == 0 {
		return nil
	}

	return errs
}

// Transform makes UserConfig implement the TransformableOptions interface
//
// The UserConfig options (flags/envs/configs) will be at molded at unmarshalling time (before validation).
func (o *UserConfig) Transform(ctx context.Context) error {
	return modifiers.New().Struct(ctx, o)
}

// Attach makes UserConfig implement the Options interface
func (o *UserConfig) Attach(c *cobra.Command) {
	autoflags.Define(c, o)
}

func makeUsrC() *cobra.Command {
	// Options implementing CommonOptions propagate automatically via commands context
	commonOpts := &UtilityFlags{}
	opts := &UserConfig{}

	usrC := &cobra.Command{
		Use:   "usr",
		Short: "User management",
		Long:  "Commands for managing users in the server",
	}

	addC := &cobra.Command{
		Use:   "add",
		Short: "Add a new user",
		Long:  "Add a new user to the system with the specified details",
		PreRunE: func(c *cobra.Command, args []string) error {
			fmt.Println("|---add.PreRunE")
			if err := commonOpts.FromContext(c.Context()); err != nil {
				return err
			}
			spew.Dump(commonOpts)

			return autoflags.Unmarshal(c, opts)
		},
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Println("|---add.RunE")
			spew.Dump(opts)

			return nil
		},
	}

	opts.Attach(addC)
	commonOpts.Attach(addC)
	usrC.AddCommand(addC)
	// Setup of the usage text happens at autoflags.Define
	// For the `usr` command we do it explicitly since it has no local flags
	autoflags.SetupUsage(usrC)

	return usrC
}

var _ autoflags.ContextOptions = (*UtilityFlags)(nil)

type UtilityFlags struct {
	Verbose int  `flagtype:"count" flagshort:"v" flaggroup:"Utility"`
	DryRun  bool `flag:"dry-run" flaggroup:"Utility"`
}

type utilityFlagsKey struct{}

func (f *UtilityFlags) Attach(c *cobra.Command) {
	autoflags.Define(c, f)
}

// Context implements the CommonOptions interface
func (f *UtilityFlags) Context(ctx context.Context) context.Context {
	return context.WithValue(ctx, utilityFlagsKey{}, f)
}

func (f *UtilityFlags) FromContext(ctx context.Context) error {
	value, ok := ctx.Value(utilityFlagsKey{}).(*UtilityFlags)
	if !ok {
		return fmt.Errorf("couldn't obtain from context")
	}
	*f = *value

	return nil
}

func main() {
	// Options implementing CommonOptions propagate automatically via commands context
	commonOpts := &UtilityFlags{}

	rootC := &cobra.Command{
		Use:               "full",
		Short:             "A beautiful CLI application",
		Long:              "A demonstration of the autoflags library with beautiful CLI features",
		SilenceUsage:      true,
		DisableAutoGenTag: true,
	}

	// Global persistent pre-run for config file support
	rootC.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		fmt.Println("|-rootC.PersistentPreRunE")

		// Load config file if found
		if _, _, err := autoflags.UseConfigSimple(c); err != nil {
			return err
		}

		if err := autoflags.Unmarshal(c, commonOpts); err != nil {
			return err
		}

		return nil
	}
	rootC.RunE = func(c *cobra.Command, args []string) error {
		fmt.Println("|-rootC.RunE")

		return nil
	}

	commonOpts.Attach(rootC)
	rootC.AddCommand(makeSrvC())
	rootC.AddCommand(makeUsrC())

	// This single line enables the configuration file support
	autoflags.SetupConfig(rootC, autoflags.ConfigOptions{AppName: "full"})
	// This single line enables the debugging global flag
	autoflags.SetupDebug(rootC, autoflags.DebugOptions{})

	if err := rootC.Execute(); err != nil {
		os.Exit(1)
	}
}
