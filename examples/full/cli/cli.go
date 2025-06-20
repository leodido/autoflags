package full_example_cli

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/validator/v10"
	"github.com/leodido/autoflags"
	"github.com/leodido/autoflags/config"
	"github.com/leodido/autoflags/debug"
	"github.com/leodido/autoflags/values"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
)

type Environment string

const (
	EnvDevelopment Environment = "dev"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "prod"
)

type EvenDeeper struct {
	Setting   string `flag:"deeper-setting" default:"default-deeper-setting"`
	NoDefault string
}

type Deeply struct {
	Setting string `flag:"deep-setting" default:"default-deep-setting"`
	Deeper  EvenDeeper
}

type ServerOptions struct {
	// Basic flags
	Host string `flag:"host" flagdescr:"Server host" default:"localhost"`
	Port int    `flagshort:"p" flagdescr:"Server port" flagrequired:"true" flagenv:"true"`

	// Environment variable binding
	APIKey string `flagenv:"true" flagdescr:"API authentication key"`

	// Flag grouping for organized help
	LogLevel zapcore.Level `flag:"log-level" flaggroup:"Logging" flagdescr:"Set log level"`
	LogFile  string        `flag:"log-file" flaggroup:"Logging" flagdescr:"Log file path" flagenv:"true"`

	// Nested structs for organization
	Database DatabaseConfig `flaggroup:"Database"`

	// Custom type
	TargetEnv Environment `flagcustom:"true" flag:"target-env" flagdescr:"Set the target environment"`

	Deep Deeply
}

type DatabaseConfig struct {
	URL      string `flag:"db-url" flagdescr:"Database connection URL"`
	MaxConns int    `flagdescr:"Max database connections" default:"10" flagenv:"true"`
}

// DefineTargetEnv defines the custom flag for Environment with autocompletion
func (o *ServerOptions) DefineTargetEnv(name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
	enhancedDesc := descr + " {dev,staging,prod}"
	fieldPtr := fieldValue.Addr().Interface().(*Environment)
	*fieldPtr = EnvDevelopment

	// Since Environment is a string type, we cast its pointer to *string and use our string value helper.
	return values.NewString((*string)(fieldPtr)), enhancedDesc
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
func (o *ServerOptions) Attach(c *cobra.Command) error {
	if err := autoflags.Define(c, o); err != nil {
		return err
	}

	c.RegisterFlagCompletionFunc("target-env", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"dev\tDevelopment environment",
			"staging\tStaging environment",
			"prod\tProduction environment",
		}, cobra.ShellCompDirectiveNoFileComp
	})

	return nil
}

func makeSrvC() *cobra.Command {
	commonOpts := &UtilityFlags{}
	opts := &ServerOptions{}

	srvC := &cobra.Command{
		Use:   "srv",
		Short: "Start the server",
		Long:  "Start the server with the specified configuration",
		PreRunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintln(c.OutOrStdout(), "|--srvC.PreRunE")
			if err := autoflags.Unmarshal(c, opts); err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), pretty(opts))

			return nil
		},
		Run: func(c *cobra.Command, args []string) {
			fmt.Fprintln(c.OutOrStdout(), "|--srvC.RunE")
		},
	}
	opts.Attach(srvC)

	versionC := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintln(c.OutOrStdout(), "|---versionC.RunE")
			if err := commonOpts.FromContext(c.Context()); err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), pretty(commonOpts))

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
func (o *UserConfig) Attach(c *cobra.Command) error {
	return autoflags.Define(c, o)
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
			fmt.Fprintln(c.OutOrStdout(), "|---add.PreRunE")
			if err := commonOpts.FromContext(c.Context()); err != nil {
				return err
			}
			fmt.Fprintln(c.OutOrStdout(), pretty(commonOpts))

			return autoflags.Unmarshal(c, opts)
		},
		RunE: func(c *cobra.Command, args []string) error {
			fmt.Fprintln(c.OutOrStdout(), "|---add.RunE")
			fmt.Fprintln(c.OutOrStdout(), pretty(opts))

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
	DryRun  bool `flag:"dry" flaggroup:"Utility" flagenv:"true"`
}

type utilityFlagsKey struct{}

func (f *UtilityFlags) Attach(c *cobra.Command) error {
	return autoflags.Define(c, f)
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

func NewRootC(exitOnDebug bool) (*cobra.Command, error) {
	// Options implementing CommonOptions propagate automatically via commands context
	commonOpts := &UtilityFlags{}

	rootC := &cobra.Command{
		Use:               "full",
		Short:             "A beautiful CLI application",
		Long:              "A demonstration of the autoflags library with beautiful CLI features",
		SilenceUsage:      true,
		DisableAutoGenTag: true,
		// Parse its own flags first, then continue traversing down to find subcommands
		// Useful for allowing context options not being attached to all the subcommands
		// Eg, `go run main.go --dry-run usr add` would fail otherwise
		TraverseChildren: true,
		// Because we handle errors ourselves in this example
		SilenceErrors: true,
	}

	// Global persistent pre-run for config file support
	rootC.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		fmt.Fprintln(c.OutOrStdout(), "|-rootC.PersistentPreRunE")

		// Load config file if found
		_, configMessage, configErr := autoflags.UseConfigSimple(c)
		if configErr != nil {
			return configErr
		}
		if configMessage != "" {
			c.Println(configMessage)
		}

		if err := autoflags.Unmarshal(c, commonOpts); err != nil {
			return err
		}

		return nil
	}
	rootC.RunE = func(c *cobra.Command, args []string) error {
		fmt.Fprintln(c.OutOrStdout(), "|-rootC.RunE")

		return nil
	}

	commonOpts.Attach(rootC)
	rootC.AddCommand(makeSrvC())
	rootC.AddCommand(makeUsrC())

	// This single line enables the configuration file support
	if err := autoflags.SetupConfig(rootC, config.Options{AppName: "full"}); err != nil {
		return nil, err
	}
	// This single line enables the debugging global flag
	if err := autoflags.SetupDebug(rootC, debug.Options{Exit: exitOnDebug}); err != nil {
		return nil, err
	}

	return rootC, nil
}

func pretty(opts any) string {
	prettyOpts, err := json.MarshalIndent(opts, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("Error marshalling options: %s", err.Error()))
	}

	return string(prettyOpts)
}
