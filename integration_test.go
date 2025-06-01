package autoflags_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

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

	t.Run("ValidationFails_InvalidEmail_AfterMold", func(t *testing.T) {
		setupTest()
		cmd := &cobra.Command{Use: "testcmd-emailfail"}
		opts := &unmarshalIntegrationOptions{}

		errDefine := autoflags.Define(cmd, opts)
		require.NoError(t, errDefine)

		viper.Set("email", "  NOTANEMAIL@domain  ")
		viper.Set("age", 25)

		err := autoflags.Unmarshal(cmd, opts)

		assert.Error(t, err, "Unmarshal should return an error for invalid email format")
		assert.Contains(t, err.Error(), "invalid options")
		assert.Contains(t, err.Error(), "Error:Field validation for 'Email' failed on the 'email' tag")
		assert.Equal(t, "notanemail@domain", opts.Email)
		assert.Equal(t, "active", opts.Status)
	})

	t.Run("ValidationFails_InvalidAge", func(t *testing.T) {
		setupTest()
		cmd := &cobra.Command{Use: "testcmd-agefail"}
		opts := &unmarshalIntegrationOptions{}

		errDefine := autoflags.Define(cmd, opts)
		require.NoError(t, errDefine)

		viper.Set("email", "valid@example.com")
		viper.Set("age", 5)

		err := autoflags.Unmarshal(cmd, opts)

		assert.Error(t, err, "Unmarshal should return an error for invalid age")
		assert.Contains(t, err.Error(), "invalid options")
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
