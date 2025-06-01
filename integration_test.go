package autoflags_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-playground/mold/v4"
	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/validator/v10"
	"github.com/leodido/autoflags"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Package-level variables to hold the initialized instances
var (
	testMolder    *mold.Transformer
	testValidator *validator.Validate
)

// TestMain sets up the molder and validator instances once for all tests in this package.
func TestMain(m *testing.M) {
	// Setup
	testMolder = modifiers.New()
	testValidator = validator.New()

	// Register custom validation functions or tags here
	// testValidator.RegisterValidation("my_custom_tag", myCustomValidationFunc)

	// Run all tests in the package
	exitCode := m.Run()

	// Teardown (if necessary, though not typically for molder/validator instances)
	os.Exit(exitCode)
}

type unmarshalIntegrationOptions struct {
	Name                 string `flag:"name" mod:"trim"`
	Email                string `flag:"email" mod:"trim,lcase" validate:"required,email"`
	Age                  int    `flag:"age" validate:"min=18,max=120"`
	Status               string `flag:"status" mod:"default=active" validate:"required,oneof=active inactive pending"`
	Justification        string `flag:"justification" validate:"required_if=Status pending"`
	SimulatePreMoldError bool
}

// Attach (definition remains the same)
func (o *unmarshalIntegrationOptions) Attach(c *cobra.Command) {
	c.Flags().StringVar(&o.Name, "name", "", "User's name")
	c.Flags().StringVar(&o.Email, "email", "", "User's email address")
	c.Flags().IntVar(&o.Age, "age", 0, "User's age")
	c.Flags().StringVar(&o.Status, "status", "", "User's status (active, inactive, pending)")
	c.Flags().StringVar(&o.Justification, "justification", "", "Justification if status is pending")
}

func (o *unmarshalIntegrationOptions) Transform(ctx context.Context) error {
	if o.SimulatePreMoldError {
		return errors.New("simulated pre-mold transformation error")
	}
	err := testMolder.Struct(ctx, o)
	if err != nil {
		return fmt.Errorf("mold transformation failed: %w", err)
	}
	return nil
}

func (o *unmarshalIntegrationOptions) Validate() []error {
	var errs []error
	err := testValidator.Struct(o)
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

func TestUnmarshal_Integration_WithLibraries(t *testing.T) {
	setupTest := func() {
		viper.Reset()
	}

	t.Run("PreMoldTransformationFails", func(t *testing.T) {
		setupTest()
		cmd := &cobra.Command{Use: "testcmd-premoldfail"}
		opts := &unmarshalIntegrationOptions{
			SimulatePreMoldError: true,
		}
		errDefine := autoflags.Define(cmd, opts)
		require.NoError(t, errDefine)

		err := autoflags.Unmarshal(cmd, opts)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "couldn't transform options:")
		assert.Contains(t, err.Error(), "simulated pre-mold transformation error")
	})

	t.Run("ValidationFails_InvalidAge", func(t *testing.T) {
		setupTest()
		cmd := &cobra.Command{Use: "testcmd-agefail"}
		opts := &unmarshalIntegrationOptions{}

		errDefine := autoflags.Define(cmd, opts)
		require.NoError(t, errDefine)

		viper.Set("email", "valid@example.com")
		viper.Set("age", 5) // Invalid age

		err := autoflags.Unmarshal(cmd, opts)

		require.Error(t, err, "Unmarshal should return an error for invalid age")
		var valErr *autoflags.ValidationError
		require.True(t, errors.As(err, &valErr), "Error should be of type *autoflags.ValidationError")

		assert.Equal(t, cmd.Name(), valErr.ContextName)

		foundAgeError := false
		for _, specificErr := range valErr.UnderlyingErrors() {
			var fieldErr validator.FieldError
			if errors.As(specificErr, &fieldErr) {
				if fieldErr.Field() == "Age" && fieldErr.Tag() == "min" {
					foundAgeError = true
				}
			}
		}
		assert.True(t, foundAgeError, "Expected validator.FieldError for Age with 'min' tag")

		assert.Contains(t, err.Error(), "invalid options for "+cmd.Name()+":")
		assert.Contains(t, err.Error(), "Error:Field validation for 'Age' failed on the 'min' tag")
	})

	t.Run("ValidationFails_RequiredIf_Justification", func(t *testing.T) {
		setupTest()
		cmd := &cobra.Command{Use: "testcmd-reqif-fail"}
		opts := &unmarshalIntegrationOptions{}

		errDefine := autoflags.Define(cmd, opts)
		require.NoError(t, errDefine)

		viper.Set("email", "valid@example.com")
		viper.Set("age", 30)
		viper.Set("status", "pending")
		viper.Set("justification", "")

		err := autoflags.Unmarshal(cmd, opts)

		assert.Error(t, err, "Unmarshal should return an error if Justification is missing when Status is pending")
		assert.Contains(t, err.Error(), "invalid options")
		assert.Contains(t, err.Error(), "Error:Field validation for 'Justification' failed on the 'required_if' tag")
	})

	t.Run("ValidationFails_InvalidEmail_AfterMold", func(t *testing.T) {
		setupTest()
		cmd := &cobra.Command{Use: "testcmd-emailfail"}
		opts := &unmarshalIntegrationOptions{}

		errDefine := autoflags.Define(cmd, opts)
		require.NoError(t, errDefine)

		viper.Set("email", "  NOTANEMAIL@domain  ")
		viper.Set("age", 25)

		err := autoflags.Unmarshal(cmd, opts)

		var valErr *autoflags.ValidationError
		require.Error(t, err, "Unmarshal should return an error for invalid email format")
		require.True(t, errors.As(err, &valErr), "Error should be of type *autoflags.ValidationError")

		assert.Equal(t, cmd.Name(), valErr.ContextName, "ValidationError ContextName should match command name")

		foundEmailError := false
		for _, specificErr := range valErr.UnderlyingErrors() {
			var fieldErr validator.FieldError
			require.True(t, errors.As(specificErr, &fieldErr), "Underlying error should be of type validator.FieldError")
			if errors.As(specificErr, &fieldErr) {
				if fieldErr.Field() == "Email" && fieldErr.Tag() == "email" {
					foundEmailError = true
				}
			}
		}
		assert.True(t, foundEmailError, "Expected a validator.FieldError for 'Email' field with 'email' tag")

		assert.Contains(t, err.Error(), "invalid options for "+cmd.Name()+":")
		assert.Contains(t, err.Error(), "Error:Field validation for 'Email' failed on the 'email' tag")

		assert.Equal(t, "notanemail@domain", opts.Email)
		assert.Equal(t, "active", opts.Status)
	})

	t.Run("Success_WithMoldAndValidator", func(t *testing.T) {
		setupTest()
		cmd := &cobra.Command{Use: "testcmd-success-libs"}
		opts := &unmarshalIntegrationOptions{}

		errDefine := autoflags.Define(cmd, opts)
		require.NoError(t, errDefine)

		viper.Set("name", "  Test User  ")
		viper.Set("email", "  USER.TEST@Example.COM  ")
		viper.Set("age", 42)
		viper.Set("status", "inactive")

		err := autoflags.Unmarshal(cmd, opts)

		assert.NoError(t, err, "Unmarshal should succeed")
		assert.Equal(t, "Test User", opts.Name)
		assert.Equal(t, "user.test@example.com", opts.Email)
		assert.Equal(t, 42, opts.Age)
		assert.Equal(t, "inactive", opts.Status)
	})
}

type TestDefineConfigFlags struct {
	LogLevel string `default:"info" flag:"log-level" flagdescr:"set the logging level" flaggroup:"Config"`
	Timeout  int    `flagdescr:"set the timeout, in seconds"`
	Endpoint string `flagdescr:"the endpoint emitting the verdicts" flaggroup:"Config" flagrequired:"true"`
}

type TestDefineDeepFlags struct {
	Deep time.Duration `default:"deepdown" flagdescr:"deep flag" flag:"deep" flagshort:"d" flaggroup:"Deep"`
}

type TestDefineJSONFlags struct {
	JSON bool                `flagdescr:"output the verdicts (if any) in JSON form"`
	JQ   string              `flagshort:"q" flagdescr:"filter the output using a jq expression"`
	Deep TestDefineDeepFlags `flagrequired:"true"`
}

type TestDefineOptions struct {
	TestDefineConfigFlags `flaggroup:"Configuration"`
	Nest                  TestDefineJSONFlags
}

func (o TestDefineOptions) Attach(c *cobra.Command)             {}
func (o TestDefineOptions) Transform(ctx context.Context) error { return nil }
func (o TestDefineOptions) Validate() []error                   { return nil }

func TestDefine_Integration(t *testing.T) {
	setupTest := func() {
		viper.Reset()
	}

	cases := []struct {
		desc  string
		input autoflags.Options
	}{
		{
			"flags definition from struct reference",
			&TestDefineOptions{},
		},
		{
			"flags definition from struct",
			TestDefineOptions{},
		},
	}

	confAnnotation := []string{"Configuration"}
	requiredAnnotation := []string{"true"}
	deepAnnotation := []string{"Deep"}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			setupTest()
			c := &cobra.Command{}
			autoflags.Define(c, tc.input)
			f := c.Flags()
			vip := autoflags.GetViper(c)

			// LogLevel
			logLevelFlag := f.Lookup("log-level")
			require.NotNil(t, logLevelFlag, "Pflag 'log-level' should be defined")
			require.Equal(t, "info", vip.Get("log-level"), "Viper default for 'log-level' should be 'info'")
			require.Equal(t, vip.Get("testdefineconfigflags.loglevel"), vip.Get("log-level"), "Viper should resolve path 'testdefineconfigflags.loglevel' same as 'log-level'")
			require.NotNil(t, logLevelFlag.Annotations, "'log-level' flag annotations should exist")
			require.Equal(t, confAnnotation, logLevelFlag.Annotations[autoflags.FlagGroupAnnotation], "Group annotation for 'log-level' should be 'Configuration' (override)")
			require.Equal(t, "set the logging level", logLevelFlag.Usage, "Usage string for 'log-level'")

			// Endpoint
			endpointFlag := f.Lookup("testdefineconfigflags.endpoint")
			require.NotNil(t, endpointFlag, "Pflag 'testdefineconfigflags.endpoint' should be defined")
			require.NotNil(t, endpointFlag.Annotations, "'testdefineconfigflags.endpoint' flag annotations should exist")
			require.Equal(t, confAnnotation, endpointFlag.Annotations[autoflags.FlagGroupAnnotation], "Group annotation for 'testdefineconfigflags.endpoint' should be 'Configuration' (override)")
			require.NotNil(t, endpointFlag.Annotations[cobra.BashCompOneRequiredFlag], "'testdefineconfigflags.endpoint' should have required annotation")
			require.Equal(t, requiredAnnotation, endpointFlag.Annotations[cobra.BashCompOneRequiredFlag], "Required annotation for 'testdefineconfigflags.endpoint'")
			require.Equal(t, "the endpoint emitting the verdicts", endpointFlag.Usage, "Usage string for 'testdefineconfigflags.endpoint'")

			// Timeout
			timeoutFlag := f.Lookup("testdefineconfigflags.timeout")
			require.NotNil(t, timeoutFlag, "Pflag 'testdefineconfigflags.timeout' should be defined")
			require.NotNil(t, timeoutFlag.Annotations, "'testdefineconfigflags.timeout' flag annotations should exist (or be nil if no annotations are expected)")
			require.Equal(t, confAnnotation, timeoutFlag.Annotations[autoflags.FlagGroupAnnotation], "Group annotation for 'testdefineconfigflags.timeout' should be 'Configuration'")
			require.Equal(t, "set the timeout, in seconds", timeoutFlag.Usage, "Usage string for 'testdefineconfigflags.timeout'")

			// Nest.JSON
			nestJSONFlag := f.Lookup("nest.json")
			require.NotNil(t, nestJSONFlag, "Pflag 'nest.json' should be defined")
			require.Nil(t, nestJSONFlag.Annotations[autoflags.FlagGroupAnnotation], "'nest.json' should have no group annotation unless specified")
			require.Equal(t, "output the verdicts (if any) in JSON form", nestJSONFlag.Usage, "Usage string for 'nest.json'")

			// Nest.JQ (flag name "nest.jq", shorthand "q")
			nestJQFlag := f.Lookup("nest.jq")
			require.NotNil(t, nestJQFlag, "Pflag 'nest.jq' should be defined")
			require.Nil(t, nestJQFlag.Annotations[autoflags.FlagGroupAnnotation], "'nest.jq' should have no group annotation unless specified")
			require.NotNil(t, f.ShorthandLookup("q"), "Shorthand 'q' for 'nest.jq' should exist")
			require.Equal(t, "filter the output using a jq expression", nestJQFlag.Usage, "Usage string for 'nest.jq'")

			// Nest.Deep.Deep (flag name "deep", shorthand "d")
			deepFlag := f.Lookup("deep")
			require.NotNil(t, deepFlag, "Pflag 'deep' should be defined")
			require.NotNil(t, f.ShorthandLookup("d"), "Shorthand 'd' for 'deep' should exist")
			require.Equal(t, "deepdown", vip.Get("nest.deep.deep"), "Viper default for path 'nest.deep.deep'")                             // Path
			require.Equal(t, vip.Get("nest.deep.deep"), vip.Get("deep"), "Viper should resolve path 'nest.deep.deep' same as flag 'deep'") // Path vs Alias
			require.NotNil(t, deepFlag.Annotations, "'deep' flag annotations should exist")
			require.Equal(t, deepAnnotation, deepFlag.Annotations[autoflags.FlagGroupAnnotation], "Group annotation for 'deep'")
			require.NotNil(t, deepFlag.Annotations[cobra.BashCompOneRequiredFlag], "'deep' flag should have required annotation")
			require.Equal(t, requiredAnnotation, deepFlag.Annotations[cobra.BashCompOneRequiredFlag], "Required annotation for 'deep'")
			require.Equal(t, "deep flag", deepFlag.Usage, "Usage string for 'deep'")
		})
	}
}

type RootGlobalOptions struct {
	Config     string `flag:"config" mod:"trim" validate:"omitempty,filepath"`
	LogLevel   string `flag:"log-level" mod:"default=info,lcase" validate:"oneof=debug info warn error fatal panic"`
	GlobalOnly string `flag:"global-only" mod:"default=global_default"`
}

func (opts *RootGlobalOptions) Attach(c *cobra.Command) {}
func (opts *RootGlobalOptions) Transform(ctx context.Context) error {
	return testMolder.Struct(ctx, opts)
}
func (opts *RootGlobalOptions) Validate() []error {
	err := testValidator.Struct(opts)
	if err == nil {
		return nil
	}
	if vErrs, ok := err.(validator.ValidationErrors); ok {
		var errs []error
		for _, fe := range vErrs {
			errs = append(errs, fe)
		}
		return errs
	}
	return []error{fmt.Errorf("non-validator error during RootGlobalOptions validation: %w", err)}
}

type ChildCommandOptions struct {
	LocalSetting string `flag:"local-setting" validate:"required"`
	// This is gonna shadow the global one
	LogLevel         string `flag:"log-level" mod:"default=child_info,lcase" validate:"oneof=debug info warn error fatal panic"`
	TransformedLocal string // To check local transformation
}

func (opts *ChildCommandOptions) Attach(c *cobra.Command) {}
func (opts *ChildCommandOptions) Transform(ctx context.Context) error {
	if err := testMolder.Struct(ctx, opts); err != nil {
		return err
	}
	opts.TransformedLocal = strings.ToUpper(opts.LocalSetting)
	return nil
}
func (opts *ChildCommandOptions) Validate() []error {
	err := testValidator.Struct(opts)
	if err == nil {
		return nil
	}
	if vErrs, ok := err.(validator.ValidationErrors); ok {
		var errs []error
		for _, fe := range vErrs {
			errs = append(errs, fe)
		}
		return errs
	}
	return []error{fmt.Errorf("non-validator error during ChildCommandOptions validation: %w", err)}
}

func TestComplexHierarchy_GlobalAndLocalFlags(t *testing.T) {
	setupCommandsAndOptions := func(t *testing.T) (rootCmd *cobra.Command, childCmd *cobra.Command, globalOpts *RootGlobalOptions, childOpts *ChildCommandOptions, rootCmdRan *bool, childCmdRan *bool) {
		viper.Reset()

		markRootCmdRan := false
		markChildCmdRan := false
		rootCmdRan = &markRootCmdRan
		childCmdRan = &markChildCmdRan

		globalOpts = &RootGlobalOptions{}
		childOpts = &ChildCommandOptions{}

		rootCmd = &cobra.Command{
			Use: "root",
			PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
				t.Logf("RootCmd/PersistentPreRunE: unmarshalling globalOptions for command %q", cmd.Name())

				return autoflags.Unmarshal(cmd, globalOpts)
			},
			RunE: func(cmd *cobra.Command, args []string) error {
				*rootCmdRan = true
				t.Logf("RootCmd/RunE exec: globalOpts: %+v", globalOpts)

				return nil
			},
		}

		childCmd = &cobra.Command{
			Use: "child",
			RunE: func(cmd *cobra.Command, args []string) error {
				*childCmdRan = true
				if err := autoflags.Unmarshal(cmd, childOpts); err != nil {
					return fmt.Errorf("childCmd failed to unmarshal its local options: %w", err)
				}
				t.Logf("ChildCmd/RunE executed. GlobalOpts (via test variable): %+v, ChildOpts: %+v", globalOpts, childOpts)

				return nil
			},
		}
		rootCmd.AddCommand(childCmd)

		// Definisci le flag QUI, DOPO viper.Reset() e la creazione dei comandi per questo scenario
		errDefineGlobal := autoflags.Define(rootCmd, globalOpts, autoflags.WithPersistentFlags(), autoflags.WithValidation())
		require.NoError(t, errDefineGlobal, "Define for global options should succeed")

		errDefineLocal := autoflags.Define(childCmd, childOpts, autoflags.WithValidation())
		require.NoError(t, errDefineLocal, "Define for child options should succeed")

		return // Restituisce i comandi e le opzioni configurati
	}

	t.Run("ValuesFromCLI", func(t *testing.T) {
		rootCmd, childCmd, globalOpts, childOpts, rootCmdRan, childCmdRan := setupCommandsAndOptions(t)

		spew.Dump(rootCmdRan)

		rootCmd.SetArgs([]string{
			"child",
			"--config=cli_config.yaml",
			"--log-level=DEBUG",
			"--global-only=cli_global",
			"--local-setting=cli_local",
		})

		t.Log("Executing rootCmd for CLI test...")
		errExecute := rootCmd.Execute()
		require.NoError(t, errExecute, "rootCmd.Execute() should succeed for CLI test")
		require.True(t, *childCmdRan, "childCmd.RunE should have been executed")

		assert.Equal(t, "cli_config.yaml", globalOpts.Config, "Global 'config' should be from CLI")
		assert.Equal(t, "debug", globalOpts.LogLevel, "Global 'log-level' should be 'debug' from CLI (after lcase transform)")
		assert.Equal(t, "cli_global", globalOpts.GlobalOnly, "Global 'global-only' should be from CLI")

		assert.Equal(t, "cli_local", childOpts.LocalSetting, "Child 'local-setting' should be from CLI")
		assert.Equal(t, "CLI_LOCAL", childOpts.TransformedLocal, "Child 'TransformedLocal' should be uppercased CLI input")
		assert.Equal(t, "debug", childOpts.LogLevel, "Child 'log-level' (shadowing global) should be 'debug' from CLI (after lcase transform)")

		childCmdViper := autoflags.GetViper(childCmd)
		assert.Equal(t, "cli_global", childCmdViper.GetString("global-only"), "Child's viper should have 'global-only' from merged global settings after CLI parse")
		assert.Equal(t, "debug", childCmdViper.GetString("log-level"), "Child's viper 'log-level' should be 'debug' from CLI")
	})

	t.Run("ValuesFromConfigFileWithCLIShadowing", func(t *testing.T) {
		rootCmd, childCmd, globalOpts, childOpts, rootCmdRan, childCmdRan := setupCommandsAndOptions(t)

		spew.Dump(rootCmdRan)

		configFileContent := `
loglevel: "warn"
global-only: "config_global"
child:
  local-setting: "config_local_from_section"
  log-level: "error"
`
		tmpFile, err := os.CreateTemp("", "autoflags_*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		_, err = tmpFile.WriteString(configFileContent)
		require.NoError(t, err)
		err = tmpFile.Close()
		require.NoError(t, err)

		// Qui non usiamo viper.SetConfigFile() direttamente nel test,
		// ma passiamo --config al comando, che dovrebbe essere gestito da RootGlobalOptions.Config
		// e poi usato da Viper nel PersistentPreRunE o globalmente.
		// Per far sì che Viper legga il file:
		viper.SetConfigFile(tmpFile.Name()) // Imposta il file di config per il Viper globale
		errRead := viper.ReadInConfig()     // Leggi nel Viper globale
		require.NoError(t, errRead, "Error reading viper config file in test setup")

		rootCmd.SetArgs([]string{
			"child",
			// Non passiamo --config qui, ci aspettiamo che Viper globale sia già configurato
			"--log-level=debug", // CLI per global log-level (influenza RootGlobalOptions.LogLevel e la flag locale child.LogLevel)
		})

		t.Log("Executing rootCmd for ConfigFile test...")
		errExecute := rootCmd.Execute()
		require.NoError(t, errExecute)
		require.True(t, *childCmdRan)

		// Asserzioni su globalOpts
		// Config non è impostato da CLI, quindi dovrebbe essere il valore zero o default se presente
		assert.Equal(t, "", globalOpts.Config) // A meno che non abbia un default o venga letto dall'ambiente
		// Precedenza: CLI ("debug") > Config ("warn") per la flag globale
		assert.Equal(t, "debug", globalOpts.LogLevel)           // Trasformato da lcase
		assert.Equal(t, "config_global", globalOpts.GlobalOnly) // Da file di config (Viper globale)

		// Asserzioni su childOpts
		assert.Equal(t, "config_local_from_section", childOpts.LocalSetting) // Da sezione 'child' del config
		assert.Equal(t, "CONFIG_LOCAL_FROM_SECTION", childOpts.TransformedLocal)
		// Per childOpts.LogLevel:
		// Flag locale "--log-level" su childCmd.
		// Valore CLI per "--log-level" (persistente) è "debug". Questo vince su tutto.
		// La flag locale del figlio prenderà questo valore.
		assert.Equal(t, "debug", childOpts.LogLevel) // Trasformato da lcase

		// Verifica childViper
		childViper := autoflags.GetViper(childCmd)
		assert.Equal(t, "config_global", childViper.GetString("global-only"))               // Da config, non sovrascritto da CLI in questo scenario
		assert.Equal(t, "debug", childViper.GetString("log-level"))                         // Valore CLI della flag (sia essa globale o locale)
		assert.Equal(t, "config_local_from_section", childViper.GetString("local-setting")) // Da sezione config per il figlio
	})
}
