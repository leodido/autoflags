package autoflags_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/mold/v4"
	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/validator/v10"
	"github.com/leodido/autoflags"
	"github.com/spf13/afero"
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

type testContextKey string

type commonOptionsForContextTest struct {
	DummyField string `flag:"dummy"`
}

func (o *commonOptionsForContextTest) Attach(c *cobra.Command) {}

func (o *commonOptionsForContextTest) Context(ctx context.Context) context.Context {
	return context.WithValue(ctx, testContextKey("test-key"), o)
}

func TestUnmarshal_SetsContext_WhenCommonOptions(t *testing.T) {
	cmd := &cobra.Command{Use: "test-context"}
	opts := &commonOptionsForContextTest{}

	autoflags.Define(cmd, opts)

	err := autoflags.Unmarshal(cmd, opts)
	require.NoError(t, err)

	finalCtx := cmd.Context()
	require.NotNil(t, finalCtx, "The command context should not be nil")

	val := finalCtx.Value(testContextKey("test-key"))
	assert.Equal(t, opts, val, "The context should contain the value set from the 'Context()' implementation")
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

	requiredAnnotation := []string{"true"}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			setupTest()
			c := &cobra.Command{
				Use: "testcmd",
				Run: func(cmd *cobra.Command, args []string) {},
			}
			c.SetErr(io.Discard)
			c.SetOut(io.Discard)
			autoflags.Define(c, tc.input)
			f := c.Flags()
			vip := autoflags.GetViper(c)
			u := c.UsageString()

			// Usage + Grouping
			require.NotEmpty(t, u)
			assert.Contains(t, u, "Configuration Flags:", "The help output should containt the 'Configuration' group")
			assert.Contains(t, u, "Deep Flags:", "The help output should containt the 'Deep' group")

			// LogLevel
			logLevelFlag := f.Lookup("log-level")
			require.NotNil(t, logLevelFlag, "Pflag 'log-level' should be defined")
			require.Equal(t, "info", vip.Get("log-level"), "Viper default for 'log-level' should be 'info'")
			require.Equal(t, vip.Get("testdefineconfigflags.loglevel"), vip.Get("log-level"), "Viper should resolve path 'testdefineconfigflags.loglevel' same as 'log-level'")
			require.NotNil(t, logLevelFlag.Annotations, "'log-level' flag annotations should exist")
			assertFlagInGroup(t, u, "Configuration", "--log-level")
			require.Equal(t, "set the logging level", logLevelFlag.Usage, "Usage string for 'log-level'")
			require.Contains(t, u, "--log-level string", "Flag from LogLevel field")

			// Endpoint
			endpointFlag := f.Lookup("testdefineconfigflags.endpoint")
			require.NotNil(t, endpointFlag, "Pflag 'testdefineconfigflags.endpoint' should be defined")
			require.NotNil(t, endpointFlag.Annotations, "'testdefineconfigflags.endpoint' flag annotations should exist")
			assertFlagInGroup(t, u, "Configuration", "testdefineconfigflags.endpoint")
			require.NotNil(t, endpointFlag.Annotations[cobra.BashCompOneRequiredFlag], "'testdefineconfigflags.endpoint' should have required annotation")
			require.Equal(t, requiredAnnotation, endpointFlag.Annotations[cobra.BashCompOneRequiredFlag], "Required annotation for 'testdefineconfigflags.endpoint'")
			require.Equal(t, "the endpoint emitting the verdicts", endpointFlag.Usage, "Usage string for 'testdefineconfigflags.endpoint'")

			// Timeout
			timeoutFlag := f.Lookup("testdefineconfigflags.timeout")
			require.NotNil(t, timeoutFlag, "Pflag 'testdefineconfigflags.timeout' should be defined")
			require.NotNil(t, timeoutFlag.Annotations, "'testdefineconfigflags.timeout' flag annotations should exist (or be nil if no annotations are expected)")
			assertFlagInGroup(t, u, "Configuration", "--testdefineconfigflags.timeout")
			require.Equal(t, "set the timeout, in seconds", timeoutFlag.Usage, "Usage string for 'testdefineconfigflags.timeout'")

			// Nest.JSON
			nestJSONFlag := f.Lookup("nest.json")
			require.NotNil(t, nestJSONFlag, "Pflag 'nest.json' should be defined")
			assertFlagInDefaultGroup(t, u, "nest.json")
			require.Equal(t, "output the verdicts (if any) in JSON form", nestJSONFlag.Usage, "Usage string for 'nest.json'")

			// Nest.JQ (flag name "nest.jq", shorthand "q")
			nestJQFlag := f.Lookup("nest.jq")
			require.NotNil(t, nestJQFlag, "Pflag 'nest.jq' should be defined")
			require.NotNil(t, f.ShorthandLookup("q"), "Shorthand 'q' for 'nest.jq' should exist")
			assertFlagInDefaultGroup(t, u, "nest.jq")
			require.Equal(t, "filter the output using a jq expression", nestJQFlag.Usage, "Usage string for 'nest.jq'")

			// Nest.Deep.Deep (flag name "deep", shorthand "d")
			deepFlag := f.Lookup("deep")
			require.NotNil(t, deepFlag, "Pflag 'deep' should be defined")
			require.NotNil(t, f.ShorthandLookup("d"), "Shorthand 'd' for 'deep' should exist")
			require.Equal(t, "deepdown", vip.Get("nest.deep.deep"), "Viper default for path 'nest.deep.deep'")                             // Path
			require.Equal(t, vip.Get("nest.deep.deep"), vip.Get("deep"), "Viper should resolve path 'nest.deep.deep' same as flag 'deep'") // Path vs Alias
			require.NotNil(t, deepFlag.Annotations, "'deep' flag annotations should exist")
			assertFlagInGroup(t, u, "Deep", "--deep")
			require.NotNil(t, deepFlag.Annotations[cobra.BashCompOneRequiredFlag], "'deep' flag should have required annotation")
			require.Equal(t, requiredAnnotation, deepFlag.Annotations[cobra.BashCompOneRequiredFlag], "Required annotation for 'deep'")
			require.Equal(t, "deep flag", deepFlag.Usage, "Usage string for 'deep'")

			// Required flag enforcement
			c.SetArgs([]string{})
			err := c.Execute()

			require.Error(t, err, "Execute() should fail when required flag are missing")
			assert.Contains(t, err.Error(), `required flag(s)`)
			assert.Contains(t, err.Error(), `"testdefineconfigflags.endpoint"`)
			assert.Contains(t, err.Error(), `"deep"`)

			c.SetArgs([]string{"--testdefineconfigflags.endpoint=http://test.com", "--deep=1s"})
			notErr := c.Execute()
			require.NoError(t, notErr, "Execute() should work when mandatory flags are provided")
		})
	}
}

func assertFlagInGroup(t *testing.T, usageString, groupTitle, flagName string) {
	t.Helper()

	fullGroupTitle := groupTitle + " Flags:"
	sections := strings.Split(usageString, "\n\n")

	var targetSection string
	var foundGroup bool

	// Look for the section starting with fullGroupTitle
	for _, section := range sections {
		if strings.HasPrefix(section, fullGroupTitle) {
			targetSection = section
			foundGroup = true
			break
		}
	}
	require.True(t, foundGroup, "Couldn't find the section for group '%s'", fullGroupTitle)
	assert.Contains(t, targetSection, flagName, "The flag '%s' should be in group '%s'", flagName, fullGroupTitle)
}

func assertFlagInDefaultGroup(t *testing.T, usageString, flagName string) {
	t.Helper()

	const defaultGroupTitle = "Flags:"
	sections := strings.Split(usageString, "\n\n")

	var defaultSection string
	var foundGroup bool

	for _, section := range sections {
		// La sezione di default è quella la cui prima riga è esattamente "Flags:"
		lines := strings.SplitN(section, "\n", 2)
		if len(lines) > 0 && lines[0] == defaultGroupTitle {
			defaultSection = section
			foundGroup = true
			break
		}
	}

	require.True(t, foundGroup, "Couldn't find the default section 'Flags:'")
	assert.Contains(t, defaultSection, flagName, "The flag '%s' should be in the default 'Flags:' section", flagName)
}

func TestSetupConfig_Integration(t *testing.T) {
	// Setup and cleanup for each test
	setupTest := func() {
		viper.Reset()
		// Clear any environment variables that might interfere
		os.Unsetenv("MYAPP_CONFIG")
		os.Unsetenv("CUSTOM_CONFIG_VAR")
		os.Unsetenv("MY_CLI_TOOL_CONFIG")
		os.Unsetenv("TESTCMD_CONFIG")
		os.Unsetenv("MYAPP_SETTINGS")
		// Reset the global prefix
		autoflags.SetEnvPrefix("")
	}

	teardownTest := func() {
		viper.Reset()
	}

	t.Run("RootCommandValidation_Success", func(t *testing.T) {
		setupTest()
		defer teardownTest()

		rootCmd := &cobra.Command{Use: "testapp"}
		opts := autoflags.ConfigOptions{}

		err := autoflags.SetupConfig(rootCmd, opts)
		assert.NoError(t, err, "SetupConfig should succeed on root command")
	})

	t.Run("DefaultApplication_AllDefaults", func(t *testing.T) {
		setupTest()
		defer teardownTest()

		rootCmd := &cobra.Command{Use: "myapp"}
		opts := autoflags.ConfigOptions{} // All defaults

		err := autoflags.SetupConfig(rootCmd, opts)
		require.NoError(t, err)

		// Verify flag was created with defaults
		configFlag := rootCmd.PersistentFlags().Lookup("config")
		require.NotNil(t, configFlag, "config flag should be created")
		assert.Equal(t, "", configFlag.DefValue, "default value should be empty")
		assert.Contains(t, configFlag.Usage, "config file", "usage should mention config file")
		assert.Contains(t, configFlag.Usage, "/etc/myapp", "usage should contain /etc/<root_cmd_name>")
		assert.Contains(t, configFlag.Usage, "{executable_dir}/.myapp", "usage should contain {executable_dir}/.<root_cmd_name>")
		assert.Contains(t, configFlag.Usage, "$HOME/.myapp", "usage should contain $HOME/.<root_cmd_name>")
		assert.Contains(t, configFlag.Usage, "config.", "usage should use config.<ext> as config file name")
	})

	t.Run("DefaultApplication_PartialDefaults", func(t *testing.T) {
		setupTest()
		defer teardownTest()

		rootCmd := &cobra.Command{Use: "testapp"}
		opts := autoflags.ConfigOptions{
			FlagName:   "settings",
			ConfigName: "app-config",
			EnvVar:     "CUSTOM_CONFIG_VAR",
		}

		err := autoflags.SetupConfig(rootCmd, opts)
		require.NoError(t, err)

		// Verify custom flag name
		configFlag := rootCmd.PersistentFlags().Lookup("settings")
		require.NotNil(t, configFlag, "custom flag name should be used")

		// Verify no flag with default name
		defaultFlag := rootCmd.PersistentFlags().Lookup("config")
		require.Nil(t, defaultFlag, "default config flag should not exist")

		// Verify description includes custom config name
		assert.Contains(t, configFlag.Usage, "app-config", "usage should contain custom config name")
		assert.Contains(t, configFlag.Usage, ".testapp", "usage should contain dot directory using root command name")
	})

	t.Run("DefaultApplication_AppNameFromRootCommand", func(t *testing.T) {
		setupTest()
		defer teardownTest()

		rootCmd := &cobra.Command{Use: "my-cli-tool"}
		opts := autoflags.ConfigOptions{} // AppName should default to root command name

		err := autoflags.SetupConfig(rootCmd, opts)
		require.NoError(t, err)

		configFlag := rootCmd.PersistentFlags().Lookup("config")
		require.NotNil(t, configFlag)

		// Should use root command name in paths
		assert.Contains(t, configFlag.Usage, "my-cli-tool", "should use root command name in paths")
		assert.Contains(t, configFlag.Usage, "$HOME/.my-cli-tool", "should default to $HOME dot directory")
	})

	t.Run("FlagCreation_PersistentFlagProperties", func(t *testing.T) {
		setupTest()
		defer teardownTest()

		rootCmd := &cobra.Command{Use: "testapp"}
		childCmd := &cobra.Command{Use: "subcmd"}
		rootCmd.AddCommand(childCmd)
		opts := autoflags.ConfigOptions{}

		err := autoflags.SetupConfig(rootCmd, opts)
		require.NoError(t, err)

		configFlag := rootCmd.PersistentFlags().Lookup("config")
		require.NotNil(t, configFlag)

		// Verify it's a string flag
		assert.Equal(t, "string", configFlag.Value.Type(), "should be string flag")

		// Verify it's persistent (should be available on child commands through inherited flags)
		childConfigFlag := childCmd.InheritedFlags().Lookup("config")
		assert.NotNil(t, childConfigFlag, "config flag should be inherited by child commands")

		// This simulates actual usage where child commands can access parent persistent flags
		childCmd.SetArgs([]string{"--config", "test.yaml"})
		err = childCmd.ParseFlags([]string{"--config", "test.yaml"})
		assert.NoError(t, err, "child command should be able to parse parent's persistent flag")
	})

	t.Run("CompleteWorkflow_WithCustomPaths", func(t *testing.T) {
		setupTest()
		defer teardownTest()

		rootCmd := &cobra.Command{Use: "myapp"}
		opts := autoflags.ConfigOptions{
			AppName:     "myapp",
			FlagName:    "config",
			ConfigName:  "settings",
			EnvVar:      "MYAPP_SETTINGS",
			SearchPaths: []autoflags.SearchPathType{autoflags.SearchPathHomeHidden, autoflags.SearchPathWorkingDirHidden, autoflags.SearchPathCustom},
			CustomPaths: []string{"/opt/myapp"},
		}

		err := autoflags.SetupConfig(rootCmd, opts)
		require.NoError(t, err)

		// Verify all components are set up correctly
		configFlag := rootCmd.PersistentFlags().Lookup("config")
		require.NotNil(t, configFlag)

		// Check description includes custom config name and paths
		assert.Contains(t, configFlag.Usage, "settings.", "should mention custom config name")
		assert.Contains(t, configFlag.Usage, "config file", "should be identified as config file")
		assert.Contains(t, configFlag.Usage, "$HOME", "should mask $HOME actual path")
		assert.Contains(t, configFlag.Usage, "/opt/myapp", "should mentuon custom config path")
	})

	t.Run("SearchPaths_DefaultPaths", func(t *testing.T) {
		setupTest()
		defer teardownTest()

		rootCmd := &cobra.Command{Use: "myapp"}
		opts := autoflags.ConfigOptions{
			AppName: "myapp",
		}

		err := autoflags.SetupConfig(rootCmd, opts)
		require.NoError(t, err)

		configFlag := rootCmd.PersistentFlags().Lookup("config")
		require.NotNil(t, configFlag)

		// Description should mention the default search paths
		usage := configFlag.Usage
		assert.Contains(t, usage, "config file", "should mention config file")

		// Should contain examples of search paths
		assert.Contains(t, usage, "myapp", "should contain app name in paths")
		assert.Contains(t, usage, "{executable_dir}", "should mask the executable directory in paths")
		assert.Contains(t, usage, "$HOME", "should mask the home directory in paths")
	})

	t.Run("SearchPaths_CustomSearchPaths", func(t *testing.T) {
		setupTest()
		defer teardownTest()

		rootCmd := &cobra.Command{Use: "myapp"}
		opts := autoflags.ConfigOptions{
			AppName:     "myapp",
			SearchPaths: []autoflags.SearchPathType{autoflags.SearchPathCustom, autoflags.SearchPathHomeHidden},
			CustomPaths: []string{"/custom/{APP}/path1", "$PWD/path2"},
		}

		err := autoflags.SetupConfig(rootCmd, opts)
		require.NoError(t, err)

		configFlag := rootCmd.PersistentFlags().Lookup("config")
		require.NotNil(t, configFlag)

		// Description should reflect the custom search behavior
		usage := configFlag.Usage
		require.Contains(t, usage, "config file", "should mention config file")
		require.Contains(t, usage, "/custom/myapp/path1", "should mention fallback to custom path")
		require.Contains(t, usage, "$PWD/path2", "should mention $PWD custom path without resolving $PWD")
		require.Contains(t, usage, ".myapp", "should mention fallback to home dot directory")
		require.Contains(t, usage, "$HOME", "should mention $HOME directory")
	})
}

func TestConfigFlow_FileDiscovery(t *testing.T) {
	setupTest := func() {
		viper.Reset()
		autoflags.SetEnvPrefix("")
	}

	setupMockEnvironment := func(t *testing.T) (fs afero.Fs, cleanup func()) {
		// Create mock filesystem
		fs = afero.NewMemMapFs()

		// Store original environment values
		originalHome := os.Getenv("HOME")
		originalPwd := os.Getenv("PWD")

		// Set mock environment values
		mockHome := "/home/testuser"
		mockPwd := "/current/dir"

		os.Setenv("HOME", mockHome)
		os.Setenv("PWD", mockPwd)

		// Create mock directories in filesystem
		err := fs.MkdirAll(mockHome, 0755)
		require.NoError(t, err)
		err = fs.MkdirAll(mockPwd, 0755)
		require.NoError(t, err)
		err = fs.MkdirAll("/etc", 0755)
		require.NoError(t, err)

		// Configure viper to use our mock filesystem
		viper.SetFs(fs)

		return fs, func() {
			os.Setenv("HOME", originalHome)
			os.Setenv("PWD", originalPwd)
			viper.Reset()
		}
	}

	createConfigContent := func(configType string) string {
		switch configType {
		case "yaml":
			return `
loglevel: debug
jsonlogging: true
dns:
  freeze: true
  cgroups:
    - test-group1
    - test-group2
tty:
  ignore-comms:
    - bash
    - zsh
`
		case "json":
			return `{
  "loglevel": "debug",
  "jsonlogging": true,
  "dns": {
    "freeze": true,
    "cgroups": ["test-group1", "test-group2"]
  },
  "tty": {
    "ignore-comms": ["bash", "zsh"]
  }
}`
		default:
			return ""
		}
	}

	t.Run("ConfigFromExplicitFlag", func(t *testing.T) {
		for _, format := range []string{"yaml", "json"} {
			t.Run(format, func(t *testing.T) {
				setupTest()
				fs, cleanup := setupMockEnvironment(t)
				defer cleanup()

				// Create explicit config file
				explicitConfigPath := "/custom/path/myconfig." + format
				err := fs.MkdirAll(filepath.Dir(explicitConfigPath), 0755)
				require.NoError(t, err)

				err = afero.WriteFile(fs, explicitConfigPath, []byte(createConfigContent(format)), 0644)
				require.NoError(t, err)

				// Create a buffer to capture command output
				var buf bytes.Buffer

				// Set up command with a proper run function
				rootCmd := &cobra.Command{
					Use: "testapp",
					Run: func(cmd *cobra.Command, args []string) {
						// Test config discovery inside the command execution
						inUse, message, err := autoflags.UseConfig(func() bool { return true })
						require.NoError(t, err)

						// Write results to buffer so we can check them
						if inUse {
							buf.WriteString("CONFIG_LOADED:")
							buf.WriteString(message)
							buf.WriteString(":LOGLEVEL:")
							buf.WriteString(viper.GetString("loglevel"))
							buf.WriteString(":JSONLOGGING:")
							if viper.GetBool("jsonlogging") {
								buf.WriteString("true")
							} else {
								buf.WriteString("false")
							}
						} else {
							buf.WriteString("NO_CONFIG:")
							buf.WriteString(message)
						}
					},
				}

				// Redirect output to our buffer
				rootCmd.SetOut(&buf)
				rootCmd.SetErr(&buf)

				configOpts := autoflags.ConfigOptions{
					AppName: "testapp",
				}

				err = autoflags.SetupConfig(rootCmd, configOpts)
				require.NoError(t, err)

				// Execute the command with the --config flag
				rootCmd.SetArgs([]string{"--config", explicitConfigPath})
				err = rootCmd.Execute()
				require.NoError(t, err)

				// Verify the results from the command execution
				output := buf.String()
				assert.Contains(t, output, "CONFIG_LOADED:", "Config should be loaded")
				assert.Contains(t, output, explicitConfigPath, "Output should contain the config file path")
				assert.Contains(t, output, "Using config file:", "Output should indicate config file is being used")
				assert.Contains(t, output, ":LOGLEVEL:debug", "Config loglevel should be loaded")
				assert.Contains(t, output, ":JSONLOGGING:true", "Config jsonlogging should be loaded")
			})
		}
	})

	t.Run("ConfigFromSearchPaths", func(t *testing.T) {
		setupTest()
		fs, cleanup := setupMockEnvironment(t)
		defer cleanup()

		// Create config file in one of the default search paths ($HOME/.testapp/)
		homeConfigPath := "/home/testuser/.testapp/config.yaml"
		err := fs.MkdirAll(filepath.Dir(homeConfigPath), 0755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, homeConfigPath, []byte(createConfigContent("yaml")), 0644)
		require.NoError(t, err)

		// Create a buffer to capture command output
		var buf bytes.Buffer

		// Set up command with a proper run function
		rootCmd := &cobra.Command{
			Use: "testapp",
			Run: func(cmd *cobra.Command, args []string) {
				// Test config discovery inside the command execution
				inUse, message, err := autoflags.UseConfig(func() bool { return true })
				require.NoError(t, err)

				// Write results to buffer so we can check them
				if inUse {
					buf.WriteString("CONFIG_LOADED:")
					buf.WriteString(message)
					buf.WriteString(":LOGLEVEL:")
					buf.WriteString(viper.GetString("loglevel"))
					buf.WriteString(":JSONLOGGING:")
					if viper.GetBool("jsonlogging") {
						buf.WriteString("true")
					} else {
						buf.WriteString("false")
					}
				} else {
					buf.WriteString("NO_CONFIG:")
					buf.WriteString(message)
				}
			},
		}

		// Redirect output to our buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)

		configOpts := autoflags.ConfigOptions{
			AppName: "testapp",
		}

		err = autoflags.SetupConfig(rootCmd, configOpts)
		require.NoError(t, err)

		// Execute the command WITHOUT --config flag and WITHOUT env var (should discover from search paths)
		rootCmd.SetArgs([]string{})
		err = rootCmd.Execute()
		require.NoError(t, err)

		// Verify the results from the command execution
		output := buf.String()
		assert.Contains(t, output, "CONFIG_LOADED:", "Config should be loaded from search paths")
		assert.Contains(t, output, homeConfigPath, "Output should contain the search path config file")
		assert.Contains(t, output, "Using config file:", "Output should indicate config file is being used")
		assert.Contains(t, output, ":LOGLEVEL:debug", "Config loglevel should be loaded from search path")
		assert.Contains(t, output, ":JSONLOGGING:true", "Config jsonlogging should be loaded from search path")
	})

	t.Run("ConfigPrecedenceOrder", func(t *testing.T) {
		setupTest()
		fs, cleanup := setupMockEnvironment(t)
		defer cleanup()

		// Create config files in multiple locations with different values
		explicitConfigPath := "/explicit/config.yaml"
		envConfigPath := "/env/config.yaml"
		homeConfigPath := "/home/testuser/.testapp/config.yaml"

		err := fs.MkdirAll(filepath.Dir(explicitConfigPath), 0755)
		require.NoError(t, err)
		err = fs.MkdirAll(filepath.Dir(envConfigPath), 0755)
		require.NoError(t, err)
		err = fs.MkdirAll(filepath.Dir(homeConfigPath), 0755)
		require.NoError(t, err)

		// Create configs with different loglevel values to test precedence
		explicitConfig := `loglevel: error
jsonlogging: false`
		envConfig := `loglevel: warn
jsonlogging: false`
		homeConfig := `loglevel: debug
jsonlogging: true`

		err = afero.WriteFile(fs, explicitConfigPath, []byte(explicitConfig), 0644)
		require.NoError(t, err)
		err = afero.WriteFile(fs, envConfigPath, []byte(envConfig), 0644)
		require.NoError(t, err)
		err = afero.WriteFile(fs, homeConfigPath, []byte(homeConfig), 0644)
		require.NoError(t, err)

		// Set environment variable
		originalConfigEnv := os.Getenv("TESTAPP_CONFIG")
		os.Setenv("TESTAPP_CONFIG", envConfigPath)
		defer func() {
			if originalConfigEnv != "" {
				os.Setenv("TESTAPP_CONFIG", originalConfigEnv)
			} else {
				os.Unsetenv("TESTAPP_CONFIG")
			}
		}()

		// Create a buffer to capture command output
		var buf bytes.Buffer

		// Set up command with a proper run function
		rootCmd := &cobra.Command{
			Use: "testapp",
			Run: func(cmd *cobra.Command, args []string) {
				// Test config discovery inside the command execution
				inUse, message, err := autoflags.UseConfig(func() bool { return true })
				require.NoError(t, err)

				// Write results to buffer so we can check them
				if inUse {
					buf.WriteString("CONFIG_LOADED:")
					buf.WriteString(message)
					buf.WriteString(":LOGLEVEL:")
					buf.WriteString(viper.GetString("loglevel"))
					buf.WriteString(":JSONLOGGING:")
					if viper.GetBool("jsonlogging") {
						buf.WriteString("true")
					} else {
						buf.WriteString("false")
					}
				} else {
					buf.WriteString("NO_CONFIG:")
					buf.WriteString(message)
				}
			},
		}

		// Redirect output to our buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)

		configOpts := autoflags.ConfigOptions{
			AppName: "testapp",
		}

		err = autoflags.SetupConfig(rootCmd, configOpts)
		require.NoError(t, err)

		// Execute with explicit --config flag (should take precedence over env var and search paths)
		rootCmd.SetArgs([]string{"--config", explicitConfigPath})
		err = rootCmd.Execute()
		require.NoError(t, err)

		// Verify explicit config takes precedence
		output := buf.String()
		assert.Contains(t, output, "CONFIG_LOADED:", "Config should be loaded")
		assert.Contains(t, output, explicitConfigPath, "Should use explicit config file")
		assert.Contains(t, output, ":LOGLEVEL:error", "Should use explicit config loglevel (error)")
		assert.Contains(t, output, ":JSONLOGGING:false", "Should use explicit config jsonlogging (false)")
	})

	t.Run("ConfigFileNotFound", func(t *testing.T) {
		setupTest()
		_, cleanup := setupMockEnvironment(t)
		defer cleanup()

		// Don't create any config files - test when none are found

		// Create a buffer to capture command output
		var buf bytes.Buffer

		// Set up command with a proper run function
		rootCmd := &cobra.Command{
			Use: "testapp",
			Run: func(cmd *cobra.Command, args []string) {
				// Test config discovery inside the command execution
				inUse, message, err := autoflags.UseConfig(func() bool { return true })
				require.NoError(t, err)

				// Write results to buffer so we can check them
				if inUse {
					buf.WriteString("CONFIG_LOADED:")
					buf.WriteString(message)
				} else {
					buf.WriteString("NO_CONFIG:")
					buf.WriteString(message)
				}
			},
		}

		// Redirect output to our buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)

		configOpts := autoflags.ConfigOptions{
			AppName: "testapp",
		}

		err := autoflags.SetupConfig(rootCmd, configOpts)
		require.NoError(t, err)

		// Execute the command (should not find any config)
		rootCmd.SetArgs([]string{})
		err = rootCmd.Execute()
		require.NoError(t, err)

		// Verify no config was found
		output := buf.String()
		assert.Contains(t, output, "NO_CONFIG:", "No config should be found")
		assert.Contains(t, output, "Running without a configuration file", "Should indicate no config file found")
	})

	t.Run("CustomSearchPaths", func(t *testing.T) {
		setupTest()
		fs, cleanup := setupMockEnvironment(t)
		defer cleanup()

		// Create config file in a custom search path location
		customConfigPath := "/custom/search/path/config.yaml"
		err := fs.MkdirAll(filepath.Dir(customConfigPath), 0755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, customConfigPath, []byte(createConfigContent("yaml")), 0644)
		require.NoError(t, err)

		// Create a buffer to capture command output
		var buf bytes.Buffer

		// Set up command with a proper run function
		rootCmd := &cobra.Command{
			Use: "testapp",
			Run: func(cmd *cobra.Command, args []string) {
				// Test config discovery inside the command execution
				inUse, message, err := autoflags.UseConfig(func() bool { return true })
				require.NoError(t, err)

				// Write results to buffer so we can check them
				if inUse {
					buf.WriteString("CONFIG_LOADED:")
					buf.WriteString(message)
					buf.WriteString(":LOGLEVEL:")
					buf.WriteString(viper.GetString("loglevel"))
					buf.WriteString(":JSONLOGGING:")
					if viper.GetBool("jsonlogging") {
						buf.WriteString("true")
					} else {
						buf.WriteString("false")
					}
				} else {
					buf.WriteString("NO_CONFIG:")
					buf.WriteString(message)
				}
			},
		}

		// Redirect output to our buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)

		configOpts := autoflags.ConfigOptions{
			AppName:     "testapp",
			SearchPaths: []autoflags.SearchPathType{autoflags.SearchPathCustom},
			CustomPaths: []string{"/custom/search/path"},
		}

		err = autoflags.SetupConfig(rootCmd, configOpts)
		require.NoError(t, err)

		// Execute the command (should find config in custom search path)
		rootCmd.SetArgs([]string{})
		err = rootCmd.Execute()
		require.NoError(t, err)

		// Verify config was found in custom search path
		output := buf.String()
		assert.Contains(t, output, "CONFIG_LOADED:", "Config should be loaded from custom search path")
		assert.Contains(t, output, customConfigPath, "Output should contain custom config file path")
		assert.Contains(t, output, "Using config file:", "Output should indicate config file is being used")
		assert.Contains(t, output, ":LOGLEVEL:debug", "Config loglevel should be loaded")
		assert.Contains(t, output, ":JSONLOGGING:true", "Config jsonlogging should be loaded")
	})

	t.Run("CustomSearchPathsContainingAppPlaceholder", func(t *testing.T) {
		setupTest()
		fs, cleanup := setupMockEnvironment(t)
		defer cleanup()

		// Create config file in a custom search path location
		customConfigPath := "/custom/testapp/path/config.yaml"
		err := fs.MkdirAll(filepath.Dir(customConfigPath), 0755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, customConfigPath, []byte(createConfigContent("yaml")), 0644)
		require.NoError(t, err)

		// Create a buffer to capture command output
		var buf bytes.Buffer

		// Set up command with a proper run function
		rootCmd := &cobra.Command{
			Use: "testapp",
			Run: func(cmd *cobra.Command, args []string) {
				// Test config discovery inside the command execution
				inUse, message, err := autoflags.UseConfig(func() bool { return true })
				require.NoError(t, err)

				// Write results to buffer so we can check them
				if inUse {
					buf.WriteString("CONFIG_LOADED:")
					buf.WriteString(message)
					buf.WriteString(":LOGLEVEL:")
					buf.WriteString(viper.GetString("loglevel"))
					buf.WriteString(":JSONLOGGING:")
					if viper.GetBool("jsonlogging") {
						buf.WriteString("true")
					} else {
						buf.WriteString("false")
					}
				} else {
					buf.WriteString("NO_CONFIG:")
					buf.WriteString(message)
				}
			},
		}

		// Redirect output to our buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)

		configOpts := autoflags.ConfigOptions{
			AppName:     "testapp",
			SearchPaths: []autoflags.SearchPathType{autoflags.SearchPathCustom},
			CustomPaths: []string{"/custom/{APP}/path"},
		}

		err = autoflags.SetupConfig(rootCmd, configOpts)
		require.NoError(t, err)

		// Execute the command (should find config in custom search path)
		rootCmd.SetArgs([]string{})
		err = rootCmd.Execute()
		require.NoError(t, err)

		// Verify config was found in custom search path
		output := buf.String()
		assert.Contains(t, output, "CONFIG_LOADED:", "Config should be loaded from custom search path")
		assert.Contains(t, output, customConfigPath, "Output should contain custom config file path")
		assert.Contains(t, output, "Using config file:", "Output should indicate config file is being used")
		assert.Contains(t, output, ":LOGLEVEL:debug", "Config loglevel should be loaded")
		assert.Contains(t, output, ":JSONLOGGING:true", "Config jsonlogging should be loaded")
	})

	t.Run("CustomSearchPathsContainingHomeVar", func(t *testing.T) {
		setupTest()
		fs, cleanup := setupMockEnvironment(t)
		defer cleanup()

		// Create config file in a custom search path location
		customConfigPath := fmt.Sprintf("%s/custom/path/config.yaml", os.Getenv("HOME"))
		err := fs.MkdirAll(filepath.Dir(customConfigPath), 0755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, customConfigPath, []byte(createConfigContent("yaml")), 0644)
		require.NoError(t, err)

		// Create a buffer to capture command output
		var buf bytes.Buffer

		// Set up command with a proper run function
		rootCmd := &cobra.Command{
			Use: "testapp",
			Run: func(cmd *cobra.Command, args []string) {
				// Test config discovery inside the command execution
				inUse, message, err := autoflags.UseConfig(func() bool { return true })
				require.NoError(t, err)

				// Write results to buffer so we can check them
				if inUse {
					buf.WriteString("CONFIG_LOADED:")
					buf.WriteString(message)
					buf.WriteString(":LOGLEVEL:")
					buf.WriteString(viper.GetString("loglevel"))
					buf.WriteString(":JSONLOGGING:")
					if viper.GetBool("jsonlogging") {
						buf.WriteString("true")
					} else {
						buf.WriteString("false")
					}
				} else {
					buf.WriteString("NO_CONFIG:")
					buf.WriteString(message)
				}
			},
		}

		// Redirect output to our buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)

		configOpts := autoflags.ConfigOptions{
			AppName:     "testapp",
			SearchPaths: []autoflags.SearchPathType{autoflags.SearchPathCustom},
			CustomPaths: []string{"$HOME/custom/path"},
		}

		err = autoflags.SetupConfig(rootCmd, configOpts)
		require.NoError(t, err)

		// Execute the command (should find config in custom search path)
		rootCmd.SetArgs([]string{})
		err = rootCmd.Execute()
		require.NoError(t, err)

		// Verify config was found in custom search path
		output := buf.String()
		assert.Contains(t, output, "CONFIG_LOADED:", "Config should be loaded from custom search path")
		assert.Contains(t, output, customConfigPath, "Output should contain custom config file path")
		assert.Contains(t, output, "Using config file:", "Output should indicate config file is being used")
		assert.Contains(t, output, ":LOGLEVEL:debug", "Config loglevel should be loaded")
		assert.Contains(t, output, ":JSONLOGGING:true", "Config jsonlogging should be loaded")
	})

	t.Run("CustomSearchPathsContainingPwdVar", func(t *testing.T) {
		setupTest()
		fs, cleanup := setupMockEnvironment(t)
		defer cleanup()

		// Create config file in a custom search path location
		customConfigPath := fmt.Sprintf("%s/path/config.yaml", os.Getenv("PWD"))
		err := fs.MkdirAll(filepath.Dir(customConfigPath), 0755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, customConfigPath, []byte(createConfigContent("yaml")), 0644)
		require.NoError(t, err)

		// Create a buffer to capture command output
		var buf bytes.Buffer

		// Set up command with a proper run function
		rootCmd := &cobra.Command{
			Use: "testapp",
			Run: func(cmd *cobra.Command, args []string) {
				// Test config discovery inside the command execution
				inUse, message, err := autoflags.UseConfig(func() bool { return true })
				require.NoError(t, err)

				// Write results to buffer so we can check them
				if inUse {
					buf.WriteString("CONFIG_LOADED:")
					buf.WriteString(message)
					buf.WriteString(":LOGLEVEL:")
					buf.WriteString(viper.GetString("loglevel"))
					buf.WriteString(":JSONLOGGING:")
					if viper.GetBool("jsonlogging") {
						buf.WriteString("true")
					} else {
						buf.WriteString("false")
					}
				} else {
					buf.WriteString("NO_CONFIG:")
					buf.WriteString(message)
				}
			},
		}

		// Redirect output to our buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)

		configOpts := autoflags.ConfigOptions{
			AppName:     "testapp",
			SearchPaths: []autoflags.SearchPathType{autoflags.SearchPathCustom},
			CustomPaths: []string{"$PWD/path"},
		}

		err = autoflags.SetupConfig(rootCmd, configOpts)
		require.NoError(t, err)

		// Execute the command (should find config in custom search path)
		rootCmd.SetArgs([]string{})
		err = rootCmd.Execute()
		require.NoError(t, err)

		// Verify config was found in custom search path
		output := buf.String()
		assert.Contains(t, output, "CONFIG_LOADED:", "Config should be loaded from custom search path")
		assert.Contains(t, output, customConfigPath, "Output should contain custom config file path")
		assert.Contains(t, output, "Using config file:", "Output should indicate config file is being used")
		assert.Contains(t, output, ":LOGLEVEL:debug", "Config loglevel should be loaded")
		assert.Contains(t, output, ":JSONLOGGING:true", "Config jsonlogging should be loaded")
	})

	t.Run("FlagSetupWithDefaults", func(t *testing.T) {
		setupTest()
		fs, cleanup := setupMockEnvironment(t)
		defer cleanup()

		// Create config file in default location
		defaultConfigPath := "/home/testuser/.testapp/config.yaml"
		err := fs.MkdirAll(filepath.Dir(defaultConfigPath), 0755)
		require.NoError(t, err)

		defaultConfigContent := `loglevel: info
jsonlogging: false
timeout: 30`
		err = afero.WriteFile(fs, defaultConfigPath, []byte(defaultConfigContent), 0644)
		require.NoError(t, err)

		// Create a buffer to capture command output
		var buf bytes.Buffer

		// Set up command with a proper run function
		rootCmd := &cobra.Command{
			Use: "testapp",
			Run: func(cmd *cobra.Command, args []string) {
				// Test config discovery inside the command execution
				inUse, message, err := autoflags.UseConfig(func() bool { return true })
				require.NoError(t, err)

				// Write results to buffer so we can check them
				if inUse {
					buf.WriteString("CONFIG_LOADED:")
					buf.WriteString(message)
					buf.WriteString(":LOGLEVEL:")
					buf.WriteString(viper.GetString("loglevel"))
					buf.WriteString(":TIMEOUT:")
					buf.WriteString(viper.GetString("timeout"))
				} else {
					buf.WriteString("NO_CONFIG:")
					buf.WriteString(message)
				}
			},
		}

		// Redirect output to our buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)

		// Use minimal configuration options to test defaults
		configOpts := autoflags.ConfigOptions{}

		err = autoflags.SetupConfig(rootCmd, configOpts)
		require.NoError(t, err)

		// Execute the command with default setup
		rootCmd.SetArgs([]string{})
		err = rootCmd.Execute()
		require.NoError(t, err)

		// Verify default config setup works
		output := buf.String()
		assert.Contains(t, output, "CONFIG_LOADED:", "Config should be loaded with default setup")
		assert.Contains(t, output, defaultConfigPath, "Should use default config location")
		assert.Contains(t, output, ":LOGLEVEL:info", "Should load default config values")
		assert.Contains(t, output, ":TIMEOUT:30", "Should load additional config values")
	})

	t.Run("ConfigFromEnvironmentVariable", func(t *testing.T) {
		setupTest()
		fs, cleanup := setupMockEnvironment(t)
		defer cleanup()

		// Create config file in a custom location
		envConfigPath := "/env/config/app.yaml"
		err := fs.MkdirAll(filepath.Dir(envConfigPath), 0755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, envConfigPath, []byte(createConfigContent("yaml")), 0644)
		require.NoError(t, err)

		// Set environment variable for config file path
		originalConfigEnv := os.Getenv("TESTAPP_CONFIG")
		os.Setenv("TESTAPP_CONFIG", envConfigPath)
		defer func() {
			if originalConfigEnv != "" {
				os.Setenv("TESTAPP_CONFIG", originalConfigEnv)
			} else {
				os.Unsetenv("TESTAPP_CONFIG")
			}
		}()

		// Create a buffer to capture command output
		var buf bytes.Buffer

		// Set up command with a proper run function
		rootCmd := &cobra.Command{
			Use: "testapp",
			Run: func(cmd *cobra.Command, args []string) {
				// Test config discovery inside the command execution
				inUse, message, err := autoflags.UseConfig(func() bool { return true })
				require.NoError(t, err)

				// Write results to buffer so we can check them
				if inUse {
					buf.WriteString("CONFIG_LOADED:")
					buf.WriteString(message)
					buf.WriteString(":LOGLEVEL:")
					buf.WriteString(viper.GetString("loglevel"))
					buf.WriteString(":JSONLOGGING:")
					if viper.GetBool("jsonlogging") {
						buf.WriteString("true")
					} else {
						buf.WriteString("false")
					}
				} else {
					buf.WriteString("NO_CONFIG:")
					buf.WriteString(message)
				}
			},
		}

		// Redirect output to our buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)

		configOpts := autoflags.ConfigOptions{
			AppName: "testapp",
		}

		err = autoflags.SetupConfig(rootCmd, configOpts)
		require.NoError(t, err)

		// Execute the command WITHOUT --config flag (should discover from env var)
		rootCmd.SetArgs([]string{})
		err = rootCmd.Execute()
		require.NoError(t, err)

		// Verify the results from the command execution
		output := buf.String()
		assert.Contains(t, output, "CONFIG_LOADED:", "Config should be loaded from environment variable")
		assert.Contains(t, output, envConfigPath, "Output should contain the env config file path")
		assert.Contains(t, output, "Using config file:", "Output should indicate config file is being used")
		assert.Contains(t, output, ":LOGLEVEL:debug", "Config loglevel should be loaded from env config")
		assert.Contains(t, output, ":JSONLOGGING:true", "Config jsonlogging should be loaded from env config")
	})

	t.Run("CustomFlagNameAndEnvVar", func(t *testing.T) {
		setupTest()
		fs, cleanup := setupMockEnvironment(t)
		defer cleanup()

		// Create config file
		customConfigPath := "/custom/settings.yaml"
		err := fs.MkdirAll(filepath.Dir(customConfigPath), 0755)
		require.NoError(t, err)

		err = afero.WriteFile(fs, customConfigPath, []byte(createConfigContent("yaml")), 0644)
		require.NoError(t, err)

		// Set custom environment variable
		originalCustomEnv := os.Getenv("MYAPP_SETTINGS_FILE")
		os.Setenv("MYAPP_SETTINGS_FILE", customConfigPath)
		defer func() {
			if originalCustomEnv != "" {
				os.Setenv("MYAPP_SETTINGS_FILE", originalCustomEnv)
			} else {
				os.Unsetenv("MYAPP_SETTINGS_FILE")
			}
		}()

		// Create a buffer to capture command output
		var buf bytes.Buffer

		// Set up command with a proper run function
		rootCmd := &cobra.Command{
			Use: "testapp",
			Run: func(cmd *cobra.Command, args []string) {
				// Test config discovery inside the command execution
				inUse, message, err := autoflags.UseConfig(func() bool { return true })
				require.NoError(t, err)

				// Write results to buffer so we can check them
				if inUse {
					buf.WriteString("CONFIG_LOADED:")
					buf.WriteString(message)
					buf.WriteString(":LOGLEVEL:")
					buf.WriteString(viper.GetString("loglevel"))
					buf.WriteString(":JSONLOGGING:")
					if viper.GetBool("jsonlogging") {
						buf.WriteString("true")
					} else {
						buf.WriteString("false")
					}
				} else {
					buf.WriteString("NO_CONFIG:")
					buf.WriteString(message)
				}
			},
		}

		// Redirect output to our buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetErr(&buf)

		configOpts := autoflags.ConfigOptions{
			AppName:  "myapp",
			FlagName: "settings-file",
			EnvVar:   "MYAPP_SETTINGS_FILE",
		}

		err = autoflags.SetupConfig(rootCmd, configOpts)
		require.NoError(t, err)

		// Execute the command (should discover from custom env var)
		rootCmd.SetArgs([]string{})
		err = rootCmd.Execute()
		require.NoError(t, err)

		// Verify custom config setup works
		output := buf.String()
		assert.Contains(t, output, "CONFIG_LOADED:", "Config should be loaded with custom flag/env setup")
		assert.Contains(t, output, customConfigPath, "Should use custom env var config file")
		assert.Contains(t, output, "Using config file:", "Output should indicate config file is being used")
		assert.Contains(t, output, ":LOGLEVEL:debug", "Config loglevel should be loaded")
		assert.Contains(t, output, ":JSONLOGGING:true", "Config jsonlogging should be loaded")
	})
}

func TestSetupOrdering_ErrorConditions(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	t.Run("setup_debug_on_child_command", func(t *testing.T) {
		rootCmd := &cobra.Command{Use: "root"}
		childCmd := &cobra.Command{Use: "child"}
		rootCmd.AddCommand(childCmd)

		// SetupDebug should fail on child command regardless of when it's called
		err := autoflags.SetupDebug(childCmd, autoflags.DebugOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be called on the root command")
	})

	t.Run("setup_config_on_child_command", func(t *testing.T) {
		rootCmd := &cobra.Command{Use: "root"}
		childCmd := &cobra.Command{Use: "child"}
		rootCmd.AddCommand(childCmd)

		// SetupConfig should fail on child command regardless of when it's called
		err := autoflags.SetupConfig(childCmd, autoflags.ConfigOptions{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be called on the root command")
	})
}

func TestSetupOrdering_CustomOptions(t *testing.T) {
	viper.Reset()
	autoflags.SetEnvPrefix("")

	t.Setenv("CUSTOM_DEBUG", "true")

	fs := afero.NewMemMapFs()
	viper.SetFs(fs)
	configContent := "log-level: test-level"
	configPath := "/tmp/custom-settings.yaml"
	err := afero.WriteFile(fs, configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	t.Setenv("CUSTOM_CONFIG", configPath)

	// Pulisci alla fine del test
	defer func() {
		viper.Reset()
		autoflags.SetEnvPrefix("")
		os.Unsetenv("CUSTOM_CONFIG")
		os.Unsetenv("CUSTOM_DEBUG")
	}()

	opts := &OrderingTestOptions{}
	cmd := &cobra.Command{
		Use: "customapp",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, _, err := autoflags.UseConfig(func() bool { return true }); err != nil {
				return err
			}

			return autoflags.Unmarshal(cmd, opts)
		},
	}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	debugOpts := autoflags.DebugOptions{
		FlagName: "debug-mode",
		EnvVar:   "CUSTOM_DEBUG",
	}
	configOpts := autoflags.ConfigOptions{
		AppName:    "customapp",
		FlagName:   "settings",
		ConfigName: "app-settings",
		EnvVar:     "CUSTOM_CONFIG",
	}

	err = autoflags.SetupConfig(cmd, configOpts)
	require.NoError(t, err)
	err = autoflags.Define(cmd, opts)
	require.NoError(t, err)
	err = autoflags.SetupDebug(cmd, debugOpts)
	require.NoError(t, err)

	err = cmd.Execute()
	require.NoError(t, err)

	v := autoflags.GetViper(cmd)

	persistentFlags := cmd.PersistentFlags()
	assert.NotNil(t, persistentFlags.Lookup("debug-mode"))
	assert.NotNil(t, persistentFlags.Lookup("settings"))

	assert.True(t, v.GetBool("debug-mode"), "The 'debug-mode' flag should be true because of CUSTOM_DEBUG env var")

	assert.Equal(t, "test-level", v.GetString("loglevel"), "Viper should load the value from the config file given via CUSTOM_CONFIG env var")
}

type OrderingTestOptions struct {
	LogLevel string `flag:"log-level" flagenv:"true" flagdescr:"logging level"`
	Timeout  int    `flag:"timeout" flagenv:"true" flagdescr:"timeout in seconds"`
	Verbose  bool   `flag:"verbose" flagenv:"true" flagdescr:"verbose output"`
}

func (o *OrderingTestOptions) Attach(c *cobra.Command) {}

func testOrderingScenario(t *testing.T, setupFunc func(*cobra.Command, *OrderingTestOptions) error) {
	// Setup test environment
	viper.Reset()
	autoflags.SetEnvPrefix("")

	// Setup mock filesystem
	fs := afero.NewMemMapFs()
	viper.SetFs(fs)

	// Store original environment values
	originalEnvs := map[string]string{
		"HOME":                  os.Getenv("HOME"),
		"TESTAPP_LOG_LEVEL":     os.Getenv("TESTAPP_LOG_LEVEL"),
		"TESTAPP_TIMEOUT":       os.Getenv("TESTAPP_TIMEOUT"),
		"TESTAPP_VERBOSE":       os.Getenv("TESTAPP_VERBOSE"),
		"TESTAPP_DEBUG_OPTIONS": os.Getenv("TESTAPP_DEBUG_OPTIONS"),
		"TESTAPP_CONFIG":        os.Getenv("TESTAPP_CONFIG"),
	}

	// Cleanup function
	defer func() {
		for key, value := range originalEnvs {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
		viper.Reset()
	}()

	// Set up test environment variables
	mockHome := "/home/testuser"
	os.Setenv("HOME", mockHome)
	os.Setenv("TESTAPP_LOG_LEVEL", "debug")
	os.Setenv("TESTAPP_TIMEOUT", "60")
	os.Setenv("TESTAPP_VERBOSE", "true")
	os.Setenv("TESTAPP_DEBUG_OPTIONS", "true")

	// Create mock directories and config file
	err := fs.MkdirAll(mockHome+"/.testapp", 0755)
	require.NoError(t, err)

	configContent := `log-level: info
timeout: 30
verbose: false`
	configPath := mockHome + "/.testapp/config.yaml"
	err = afero.WriteFile(fs, configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Create command and options
	var captureOut bytes.Buffer
	opts := &OrderingTestOptions{}

	cmd := &cobra.Command{
		Use: "testapp",
		Run: func(cmd *cobra.Command, args []string) {
			// Test that config file discovery works
			inUse, message, err := autoflags.UseConfig(func() bool { return true })
			require.NoError(t, err)

			if inUse {
				captureOut.WriteString("CONFIG_LOADED:")
				captureOut.WriteString(message)
			} else {
				captureOut.WriteString("NO_CONFIG:")
				captureOut.WriteString(message)
			}

			// Test that debug functionality works
			captureOut.WriteRune('\n')
			autoflags.UseDebug(cmd, &captureOut)

			// Capture final values to verify environment variables took precedence
			v := autoflags.GetViper(cmd)
			captureOut.WriteString(":FINAL_LOG_LEVEL:")
			captureOut.WriteString(v.GetString("log-level"))
			captureOut.WriteString(":FINAL_TIMEOUT:")
			captureOut.WriteString(v.GetString("timeout"))
			captureOut.WriteString(":FINAL_VERBOSE:")
			if v.GetBool("verbose") {
				captureOut.WriteString("true")
			} else {
				captureOut.WriteString("false")
			}
		},
	}

	// Redirect command output to our buffer
	cmd.SetOut(&captureOut)
	cmd.SetErr(&captureOut)

	// Execute the setup function with the specific ordering
	err = setupFunc(cmd, opts)
	require.NoError(t, err, "Setup function should succeed regardless of ordering")

	// Verify all expected flags exist
	t.Run("flags_exist", func(t *testing.T) {
		// Check that all flags are present
		flags := cmd.Flags()

		assert.NotNil(t, flags.Lookup("log-level"), "log-level flag should exist")
		assert.NotNil(t, flags.Lookup("timeout"), "timeout flag should exist")
		assert.NotNil(t, flags.Lookup("verbose"), "verbose flag should exist")

		persistentFlags := cmd.PersistentFlags()
		assert.NotNil(t, persistentFlags.Lookup("debug-options"), "debug-options flag should exist")
		assert.NotNil(t, persistentFlags.Lookup("config"), "config flag should exist")
	})

	// Execute the command and verify behavior
	err = cmd.Execute()
	require.NoError(t, err, "Command execution should succeed")

	// Verify the results
	output := captureOut.String()

	t.Run("config_discovery", func(t *testing.T) {
		// Config file should be discovered and loaded
		assert.Contains(t, output, "CONFIG_LOADED:", "Config should be loaded from search paths")
		assert.Contains(t, output, configPath, "Should use the mock config file")
	})

	t.Run("environment_precedence", func(t *testing.T) {
		// Environment variables should take precedence over config file values
		assert.Contains(t, output, ":FINAL_LOG_LEVEL:debug", "Environment LOG_LEVEL should override config")
		assert.Contains(t, output, ":FINAL_TIMEOUT:60", "Environment TIMEOUT should override config")
		assert.Contains(t, output, ":FINAL_VERBOSE:true", "Environment VERBOSE should override config")
	})

	t.Run("debug_functionality", func(t *testing.T) {
		// Debug output should be present since TESTAPP_DEBUG_OPTIONS=true
		assert.Contains(t, output, "Aliases:", "Debug output should contain alias information")
		assert.Contains(t, output, "map[string]string{\"loglevel\":\"log-level\"}", "Debug output should contain alias values")
		assert.Contains(t, output, "Values:", "Debug output should be triggered by environment variable and show final values")
		assert.Contains(t, output, "Env:", "Debug output should contain env information")
		assert.Contains(t, output, "\"timeout\":[]string{\"TESTAPP_TIMEOUT\"}", "Debug output should contain timeout env information")
		assert.Contains(t, output, "\"log-level\":[]string{\"TESTAPP_LOGLEVEL\", \"TESTAPP_LOG_LEVEL\"}", "Debug output should contain log-level env information")
		assert.Contains(t, output, "\"log-level\":\"debug\"", "Debug output should contain log-level final value")
		assert.Contains(t, output, "\"debug-options\":\"true\"", "Debug output should contain debug-options final value")
	})

	// Test flag-based overrides as well
	t.Run("flag_overrides", func(t *testing.T) {
		// Reset output buffer
		captureOut.Reset()

		// Create new command instance with same setup
		flagTestCmd := &cobra.Command{
			Use: "testapp",
			Run: func(cmd *cobra.Command, args []string) {
				v := autoflags.GetViper(cmd)
				captureOut.WriteString("FLAG_LOG_LEVEL:")
				captureOut.WriteString(v.GetString("log-level"))
				captureOut.WriteString(":FLAG_TIMEOUT:")
				captureOut.WriteString(v.GetString("timeout"))
			},
		}
		flagTestCmd.SetOut(&captureOut)
		flagTestCmd.SetErr(&captureOut)

		flagTestOpts := &OrderingTestOptions{}
		err = setupFunc(flagTestCmd, flagTestOpts)
		require.NoError(t, err)

		// Test with explicit flags (should override environment)
		flagTestCmd.SetArgs([]string{"--log-level", "error", "--timeout", "120"})
		err = flagTestCmd.Execute()
		require.NoError(t, err)

		flagOutput := captureOut.String()
		assert.Contains(t, flagOutput, "FLAG_LOG_LEVEL:error", "Explicit flag should override environment")
		assert.Contains(t, flagOutput, "FLAG_TIMEOUT:120", "Explicit flag should override environment")
	})
}

func TestSetupOrdering_AllCombinations(t *testing.T) {
	orderings := []struct {
		name  string
		setup func(*cobra.Command, *OrderingTestOptions) error
	}{
		{
			name: "SetupDebug_SetupConfig_Define",
			setup: func(cmd *cobra.Command, opts *OrderingTestOptions) error {
				if err := autoflags.SetupDebug(cmd, autoflags.DebugOptions{}); err != nil {
					return err
				}
				if err := autoflags.SetupConfig(cmd, autoflags.ConfigOptions{}); err != nil {
					return err
				}

				return autoflags.Define(cmd, opts)
			},
		},
		{
			name: "SetupDebug_Define_SetupConfig",
			setup: func(cmd *cobra.Command, opts *OrderingTestOptions) error {
				if err := autoflags.SetupDebug(cmd, autoflags.DebugOptions{}); err != nil {
					return err
				}
				if err := autoflags.Define(cmd, opts); err != nil {
					return err
				}

				return autoflags.SetupConfig(cmd, autoflags.ConfigOptions{})
			},
		},
		{
			name: "SetupConfig_SetupDebug_Define",
			setup: func(cmd *cobra.Command, opts *OrderingTestOptions) error {
				if err := autoflags.SetupConfig(cmd, autoflags.ConfigOptions{}); err != nil {
					return err
				}
				if err := autoflags.SetupDebug(cmd, autoflags.DebugOptions{}); err != nil {
					return err
				}

				return autoflags.Define(cmd, opts)
			},
		},
		{
			name: "SetupConfig_Define_SetupDebug",
			setup: func(cmd *cobra.Command, opts *OrderingTestOptions) error {
				if err := autoflags.SetupConfig(cmd, autoflags.ConfigOptions{}); err != nil {
					return err
				}
				if err := autoflags.Define(cmd, opts); err != nil {
					return err
				}

				return autoflags.SetupDebug(cmd, autoflags.DebugOptions{})
			},
		},
		{
			name: "Define_SetupDebug_SetupConfig",
			setup: func(cmd *cobra.Command, opts *OrderingTestOptions) error {
				if err := autoflags.Define(cmd, opts); err != nil {
					return err
				}
				if err := autoflags.SetupDebug(cmd, autoflags.DebugOptions{}); err != nil {
					return err
				}

				return autoflags.SetupConfig(cmd, autoflags.ConfigOptions{})
			},
		},
		{
			name: "Define_SetupConfig_SetupDebug",
			setup: func(cmd *cobra.Command, opts *OrderingTestOptions) error {
				if err := autoflags.Define(cmd, opts); err != nil {
					return err
				}
				if err := autoflags.SetupConfig(cmd, autoflags.ConfigOptions{}); err != nil {
					return err
				}
				return autoflags.SetupDebug(cmd, autoflags.DebugOptions{})
			},
		},
	}

	for _, ordering := range orderings {
		t.Run(ordering.name, func(t *testing.T) {
			testOrderingScenario(t, ordering.setup)
		})
	}
}

func TestSetupFunctions_AppNameSync(t *testing.T) {
	autoflags.SetEnvPrefix("")

	rootCmd := &cobra.Command{Use: "testapp"}

	// Call SetupConfig first
	err := autoflags.SetupConfig(rootCmd, autoflags.ConfigOptions{AppName: "myapp"})
	require.NoError(t, err)
	assert.Equal(t, "MYAPP", autoflags.EnvPrefix())

	// Call SetupDebug after without app name
	err = autoflags.SetupDebug(rootCmd, autoflags.DebugOptions{})
	require.NoError(t, err)
	assert.Equal(t, "MYAPP", autoflags.EnvPrefix(), "should use the already set app name")
}

func TestSetupFunctions_NoPrefix_NoAppName_EmptyCommandName(t *testing.T) {
	autoflags.SetEnvPrefix("")

	rootCmd := &cobra.Command{Use: ""}

	// Call SetupConfig first
	err := autoflags.SetupConfig(rootCmd, autoflags.ConfigOptions{})
	require.Error(t, err)
	require.ErrorContains(t, err, "couldn't determine the app name")

	// Call SetupDebug after
	err = autoflags.SetupDebug(rootCmd, autoflags.DebugOptions{})
	require.Error(t, err)
	require.ErrorContains(t, err, "couldn't determine the app name")
}
