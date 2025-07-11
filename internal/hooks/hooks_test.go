package internalhooks_test

import (
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/leodido/structcli"
	internalenv "github.com/leodido/structcli/internal/env"
	internalhooks "github.com/leodido/structcli/internal/hooks"
	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zapcore"
)

type structcliSuite struct {
	suite.Suite
}

func TestStructCLISuite(t *testing.T) {
	suite.Run(t, new(structcliSuite))
}

func (suite *structcliSuite) SetupTest() {
	// Reset viper state before each test to prevent test pollution
	viper.Reset()
	// Reset global prefix
	structcli.SetEnvPrefix("")
}

// createTempYAMLFile creates a temporary YAML files for testing
func (suite *structcliSuite) createTempYAMLFile(content string) string {
	tmpFile, err := os.CreateTemp("", "structcli_test_*.yaml")
	require.NoError(suite.T(), err)

	_, err = tmpFile.WriteString(content)
	require.NoError(suite.T(), err)

	err = tmpFile.Close()
	require.NoError(suite.T(), err)

	return tmpFile.Name()
}

func (suite *structcliSuite) TestStoreDecodeHookFunc() {
	// Create a command with a flag
	cmd := &cobra.Command{Use: "testcmd"}
	cmd.Flags().String("custom-field", "", "test flag")

	// Create a simple decode method
	decodeMethod := func(input any) (any, error) {
		return input, nil
	}
	decodeValue := reflect.ValueOf(decodeMethod)
	targetType := reflect.TypeOf("")

	// Call StoreDecodeHookFunc
	err := internalhooks.StoreDecodeHookFunc(cmd, "custom-field", decodeValue, targetType)
	require.NoError(suite.T(), err)

	// Assert the scope contains the custom decode hook with correct key
	scope := internalscope.Get(cmd)
	expectedKey := fmt.Sprintf("customDecodeHook_%s_%s", cmd.Name(), "custom-field")

	storedHook, exists := scope.GetCustomDecodeHook(expectedKey)
	require.True(suite.T(), exists, "Custom decode hook should be stored in scope")
	assert.NotNil(suite.T(), storedHook, "Stored hook should not be nil")

	// Assert the flag has the annotation with the correct key
	flag := cmd.Flags().Lookup("custom-field")
	require.NotNil(suite.T(), flag, "Flag should exist")

	annotations := flag.Annotations[internalhooks.FlagDecodeHookAnnotation]
	require.NotNil(suite.T(), annotations, "Flag should have decode hook annotation")
	require.Len(suite.T(), annotations, 1, "Should have exactly one annotation")
	assert.Equal(suite.T(), expectedKey, annotations[0], "Annotation should contain the correct key")
}

type zapcoreLevelOptions struct {
	LogLevel zapcore.Level `default:"info" flagcustom:"true" flagdescr:"the logging level" flagenv:"true"`
}

func (o *zapcoreLevelOptions) Attach(c *cobra.Command) error { return nil }

func (suite *structcliSuite) TestHooks_DefineZapcoreLevelFlag() {
	// Test just defining the flag, without config file
	opts := &zapcoreLevelOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := structcli.Define(cmd, opts)

	require.Nil(suite.T(), err)

	// Check if the flag was created
	flag := cmd.Flags().Lookup("loglevel")
	assert.NotNil(suite.T(), flag)
	assert.Contains(suite.T(), flag.Usage, "{debug,info,warn,error,dpanic,panic,fatal}")
}

func (suite *structcliSuite) TestHooks_ZapcoreLevelFromYAML() {
	// Create a temporary config file
	configContent := `loglevel: debug`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	// Define options with zapcore.Level field
	opts := &zapcoreLevelOptions{}

	cmd := &cobra.Command{Use: "test"}

	// Set up viper to read from our config file
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	// Define flags and unmarshal
	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), zapcore.DebugLevel, opts.LogLevel)
}

type durationOptions struct {
	Timeout time.Duration `flag:"timeout" flagdescr:"request timeout" default:"30s"`
}

func (o *durationOptions) Attach(c *cobra.Command) error { return nil }

func (suite *structcliSuite) TestHooks_DurationFromFlag() {
	// Test setting duration via command line flag
	opts := &durationOptions{}
	cmd := &cobra.Command{Use: "test"}

	structcli.Define(cmd, opts)

	// Set flag value
	err := cmd.Flags().Set("timeout", "45s")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 45*time.Second, opts.Timeout)
}

func (suite *structcliSuite) TestHooks_DurationFromYAMLString() {
	// Test duration from YAML string format
	configContent := `timeout: "2m30s"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &durationOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	expected := 2*time.Minute + 30*time.Second
	assert.Equal(suite.T(), expected, opts.Timeout)
}

func (suite *structcliSuite) TestHooks_DurationFromYAMLNumber() {
	// Test duration from YAML number (nanoseconds)
	configContent := `timeout: 5000000000` // 5 seconds in nanoseconds
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &durationOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 5*time.Second, opts.Timeout)
}

func (suite *structcliSuite) TestHooks_DurationVariousFormats() {
	testCases := []struct {
		name     string
		yaml     string
		expected time.Duration
	}{
		{"milliseconds", `timeout: "500ms"`, 500 * time.Millisecond},
		{"seconds", `timeout: "30s"`, 30 * time.Second},
		{"minutes", `timeout: "5m"`, 5 * time.Minute},
		{"hours", `timeout: "2h"`, 2 * time.Hour},
		{"complex", `timeout: "1h30m45s"`, time.Hour + 30*time.Minute + 45*time.Second},
		{"microseconds", `timeout: "100us"`, 100 * time.Microsecond},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			configFile := suite.createTempYAMLFile(tc.yaml)
			defer os.Remove(configFile)

			opts := &durationOptions{}
			cmd := &cobra.Command{Use: "test"}

			viper.SetConfigFile(configFile)
			require.NoError(t, viper.ReadInConfig())

			structcli.Define(cmd, opts)
			err := structcli.Unmarshal(cmd, opts)

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, opts.Timeout)
		})
	}
}

func (suite *structcliSuite) TestHooks_DurationDefault() {
	// Test that default value is used when no config provided
	opts := &durationOptions{}
	cmd := &cobra.Command{Use: "test"}

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 30*time.Second, opts.Timeout) // from default:"30s"
}

func (suite *structcliSuite) TestHooks_DurationFlagOverridesConfig() {
	// Test flag precedence over config
	configContent := `timeout: "1m"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &durationOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)

	// Set flag value (should override config)
	err := cmd.Flags().Set("timeout", "90s")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 90*time.Second, opts.Timeout) // flag wins over config
}

type stringSliceOptions struct {
	Cgroups []string `flag:"cgroups" flagdescr:"list of cgroups to monitor"`
}

func (o *stringSliceOptions) Attach(c *cobra.Command) error { return nil }

func (suite *structcliSuite) TestHooks_StringSliceFromFlag() {
	// Test setting string slice via command line flag
	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	structcli.Define(cmd, opts)

	// Set flag value (simulating command line)
	err := cmd.Flags().Set("cgroups", "group1,group2,group3")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group1", "group2", "group3"}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceFromYAMLCommaSeparated() {
	// Test hook converting comma-separated string from YAML to []string
	configContent := `cgroups: "group1,group2,group3"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group1", "group2", "group3"}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceFromYAMLArray() {
	// Test YAML array directly (no hook needed, mapstructure handles this)
	configContent := `cgroups:
  - group1
  - group2
  - group3`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group1", "group2", "group3"}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceEmptyString() {
	// Test hook behavior with empty string
	configContent := `cgroups: ""`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	// StringToSliceHookFunc with empty string results in []string{""}
	assert.Equal(suite.T(), []string{}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceSingleValue() {
	// Test hook with single value (no commas)
	configContent := `cgroups: "single-group"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"single-group"}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceWithSpaces() {
	// Test hook with values containing spaces
	configContent := `cgroups: "group with spaces,another group,normal"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group with spaces", "another group", "normal"}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceMultipleFlags() {
	// Test setting multiple flag values
	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	structcli.Define(cmd, opts)

	// Set flag multiple times
	err := cmd.Flags().Set("cgroups", "group1")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("cgroups", "group2")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("cgroups", "group3")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group1", "group2", "group3"}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceFlagOverridesConfig() {
	// Test that flag values override config values
	configContent := `cgroups: "config1,config2"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)

	// Set flag value (should override config)
	err := cmd.Flags().Set("cgroups", "flag1,flag2,flag3")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	// Flag values should win over config values
	assert.Equal(suite.T(), []string{"flag1", "flag2", "flag3"}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceYAMLSpecialCharacters() {
	// Test hook with special characters that might cause issues
	configContent := `cgroups: "group-1_test,group:2@domain,group[3]"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group-1_test", "group:2@domain", "group[3]"}, opts.Cgroups)
}

func (suite *structcliSuite) TestHooks_StringSliceEmptyAfterSplit() {
	// Test hook behavior with leading/trailing commas
	configContent := `cgroups: ",group1,,group2,"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	// Should include empty strings from the split
	assert.Equal(suite.T(), []string{"", "group1", "", "group2", ""}, opts.Cgroups)
}

type intSliceOptions struct {
	Ports []int `flag:"ports" flagdescr:"list of ports to listen on"`
}

func (o *intSliceOptions) Attach(c *cobra.Command) error { return nil }

func (suite *structcliSuite) TestHooks_IntSliceFromFlag() {
	// Test setting int slice via command line flag
	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	structcli.Define(cmd, opts)

	// Set flag value (simulating command line)
	err := cmd.Flags().Set("ports", "8080,9090,3000")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080, 9090, 3000}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceFromYAMLCommaSeparated() {
	// Test hook converting comma-separated string from YAML to []int
	configContent := `ports: "8080,9090,3000"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080, 9090, 3000}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceFromYAMLArray() {
	// Test YAML array directly (no hook needed, mapstructure handles this)
	configContent := `ports:
  - 8080
  - 9090
  - 3000`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080, 9090, 3000}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceEmptyString() {
	// Test hook behavior with empty string
	configContent := `ports: ""`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	// StringToIntSliceHookFunc with empty string results in empty slice
	assert.Equal(suite.T(), []int{}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceSingleValue() {
	// Test hook with single value (no commas)
	configContent := `ports: "8080"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceWithSpaces() {
	// Test hook with values containing spaces (should be trimmed)
	configContent := `ports: " 8080 , 9090 , 3000 "`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080, 9090, 3000}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceNegativeNumbers() {
	// Test hook with negative numbers
	configContent := `ports: "-1,0,8080,-9090"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{-1, 0, 8080, -9090}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceMultipleFlags() {
	// Test setting multiple flag values
	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	structcli.Define(cmd, opts)

	// Set flag multiple times
	err := cmd.Flags().Set("ports", "8080")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("ports", "9090")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("ports", "3000")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080, 9090, 3000}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceFlagOverridesConfig() {
	// Test that flag values override config values
	configContent := `ports: "8080,9090"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)

	// Set flag value (should override config)
	err := cmd.Flags().Set("ports", "3000,4000,5000")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	// Flag values should win over config values
	assert.Equal(suite.T(), []int{3000, 4000, 5000}, opts.Ports)
}

func (suite *structcliSuite) TestHooks_IntSliceInvalidInteger() {
	// Test hook error handling with invalid integer
	configContent := `ports: "8080,invalid,3000"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid integer")
	assert.Contains(suite.T(), err.Error(), "invalid")
	assert.Contains(suite.T(), err.Error(), "couldn't unmarshal config to options:")
}

func (suite *structcliSuite) TestHooks_IntSliceFloatNumber() {
	// Test hook error handling with float number
	configContent := `ports: "8080,90.5,3000"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid integer")
	assert.Contains(suite.T(), err.Error(), "couldn't unmarshal config to options:")
	assert.Contains(suite.T(), err.Error(), "90.5")
}

func (suite *structcliSuite) TestHooks_IntSliceOutOfRange() {
	// Test hook error handling with number out of int range
	configContent := `ports: "8080,99999999999999999999,3000"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid integer")
	assert.Contains(suite.T(), err.Error(), "couldn't unmarshal config to options:")
}

type requiredWithEnvRuntimeOptions struct {
	RequiredEnvFlag string `flag:"required-env-flag" flagrequired:"true" flagenv:"true" flagdescr:"required flag with env"`
	OptionalEnvFlag string `flag:"optional-env-flag" flagenv:"true" flagdescr:"optional flag with env"`
}

func (o *requiredWithEnvRuntimeOptions) Attach(c *cobra.Command) error { return nil }

func (suite *structcliSuite) TestFlagrequired_WithEnvRuntimeBehavior() {
	suite.T().Run("required_flag_with_env_var_set", func(t *testing.T) {
		structcli.SetEnvPrefix("AUTOFLAGS")
		defer structcli.SetEnvPrefix("")

		// Set the environment variable that will be used
		envVarName := "AUTOFLAGS_TEST_REQUIRED_ENV_FLAG"
		originalEnv := os.Getenv(envVarName)
		defer func() {
			if originalEnv == "" {
				os.Unsetenv(envVarName)
			} else {
				os.Setenv(envVarName, originalEnv)
			}
		}()

		os.Setenv(envVarName, "env-value")

		opts := &requiredWithEnvRuntimeOptions{}
		cmd := &cobra.Command{Use: "test"}

		structcli.Define(cmd, opts)

		// Verify both annotations are set
		flags := cmd.Flags()
		requiredEnvFlag := flags.Lookup("required-env-flag")
		assert.NotNil(t, requiredEnvFlag, "required-env-flag should exist")

		// Should have both required and env annotations
		requiredAnnotation := requiredEnvFlag.Annotations[cobra.BashCompOneRequiredFlag]
		assert.NotNil(t, requiredAnnotation, "should have required annotation")
		assert.Equal(t, []string{"true"}, requiredAnnotation)

		envAnnotation := requiredEnvFlag.Annotations[internalenv.FlagAnnotation]
		assert.NotNil(t, envAnnotation, "should have env annotation")
		assert.Contains(t, envAnnotation, envVarName, "should contain the correct env var name")

		// Test that Unmarshal works with environment variable
		err := structcli.Unmarshal(cmd, opts)
		assert.NoError(t, err, "should unmarshal successfully with env var set")
		assert.Equal(t, "env-value", opts.RequiredEnvFlag, "should get value from environment")

		// Compare with optional env flag behavior
		optionalEnvFlag := flags.Lookup("optional-env-flag")
		assert.NotNil(t, optionalEnvFlag, "optional-env-flag should exist")

		optionalRequiredAnnotation := optionalEnvFlag.Annotations[cobra.BashCompOneRequiredFlag]
		assert.Nil(t, optionalRequiredAnnotation, "optional flag should not have required annotation")

		optionalEnvAnnotation := optionalEnvFlag.Annotations[internalenv.FlagAnnotation]
		assert.NotNil(t, optionalEnvAnnotation, "optional flag should have env annotation")
	})
}

func (suite *structcliSuite) TestFlagrequired_WithEnvMissingValue() {
	// Test what happens when a required+env flag has no env var set and no flag provided
	suite.T().Run("required_flag_with_no_env_var", func(t *testing.T) {
		structcli.SetEnvPrefix("AUTOFLAGS")
		defer structcli.SetEnvPrefix("")

		// Ensure the env vars are not set
		envVarNames := []string{
			"AUTOFLAGS_TEST_REQUIREDENVFLAG",
			"AUTOFLAGS_TEST_REQUIRED_ENV_FLAG",
		}

		originalEnvs := make(map[string]string)
		defer func() {
			for _, envVar := range envVarNames {
				if originalVal, exists := originalEnvs[envVar]; exists && originalVal != "" {
					os.Setenv(envVar, originalVal)
				} else {
					os.Unsetenv(envVar)
				}
			}
		}()

		for _, envVar := range envVarNames {
			originalEnvs[envVar] = os.Getenv(envVar)
			os.Unsetenv(envVar)
		}

		opts := &requiredWithEnvRuntimeOptions{}
		cmd := &cobra.Command{Use: "test"}

		structcli.Define(cmd, opts)

		// Since the flag is required and no env var is set, this should work fine
		// because structcli doesn't enforce cobra's required flags during Unmarshal
		err := structcli.Unmarshal(cmd, opts)
		assert.NoError(t, err, "Unmarshal should succeed even with missing required flag")
		assert.Equal(t, "", opts.RequiredEnvFlag, "should have empty value when no env var or flag set")

		// The required enforcement would happen when cobra validates the command execution,
		// not during the structcli Unmarshal phase
	})
}

func (suite *structcliSuite) TestFlagrequired_WithEnvConfigFile() {
	// Test that required flags work with config files too
	suite.T().Run("required_flag_from_config", func(t *testing.T) {
		configContent := `required-env-flag: "config-value"`
		configFile := suite.createTempYAMLFile(configContent)
		defer os.Remove(configFile)

		opts := &requiredWithEnvRuntimeOptions{}
		cmd := &cobra.Command{Use: "test"}

		viper.SetConfigFile(configFile)
		require.NoError(t, viper.ReadInConfig())

		structcli.Define(cmd, opts)
		err := structcli.Unmarshal(cmd, opts)

		assert.NoError(t, err, "should unmarshal successfully with config file")
		assert.Equal(t, "config-value", opts.RequiredEnvFlag, "should get value from config")
	})
}

func (suite *structcliSuite) TestHooks_ZapcoreLevelFromYAML_InvalidLevel() {
	configContent := `loglevel: "invalidlevelstring"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &zapcoreLevelOptions{}
	cmd := &cobra.Command{Use: "testinvalidlevel"}

	structcli.Define(cmd, opts)

	viper.SetConfigFile(configFile)
	errRead := viper.ReadInConfig()
	require.NoError(suite.T(), errRead, "Failed to read test config file")

	err := structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err, "Unmarshal should return an error for invalid zapcore.Level")
	assert.Contains(suite.T(), err.Error(), "couldn't unmarshal config to options:", "Error should be wrapped by Unmarshal")
	assert.Contains(suite.T(), err.Error(), "invalid string for zapcore.Level 'invalidlevelstring'", "Error should contain the specific hook error message")
}

type slogLevelOptions struct {
	LogLevel slog.Level `default:"info" flag:"log-level" flagdescr:"the logging level" flagenv:"true"`
}

func (o *slogLevelOptions) Attach(c *cobra.Command) error { return nil }

func (suite *structcliSuite) TestHooks_DefineSlogLevelFlag() {
	// Test just defining the flag, without config file
	opts := &slogLevelOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := structcli.Define(cmd, opts)

	require.Nil(suite.T(), err)

	// Check if the flag was created
	flag := cmd.Flags().Lookup("log-level")
	assert.NotNil(suite.T(), flag)
	assert.Contains(suite.T(), flag.Usage, "{debug,info,warn,error}")
}

func (suite *structcliSuite) TestHooks_SlogLevelFromYAML() {
	// Create a temporary config file
	configContent := `log-level: debug`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	// Define options with slog.Level field
	opts := &slogLevelOptions{}

	cmd := &cobra.Command{Use: "test"}

	// Set up viper to read from our config file
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	// Define flags and unmarshal
	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), slog.LevelDebug, opts.LogLevel)
}

func (suite *structcliSuite) TestHooks_SlogLevelFromEnv() {
	// Store original environment
	originalEnv := os.Getenv("TESTAPP_LOG_LEVEL")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("TESTAPP_LOG_LEVEL")
		} else {
			os.Setenv("TESTAPP_LOG_LEVEL", originalEnv)
		}
		// Reset state after test
		viper.Reset()
		structcli.SetEnvPrefix("")
	}()

	// Set environment variable (note: log-level becomes LOG_LEVEL)
	os.Setenv("TESTAPP_LOG_LEVEL", "warn")

	// IMPORTANT: Set environment prefix BEFORE defining flags
	structcli.SetEnvPrefix("TESTAPP")

	opts := &slogLevelOptions{}
	cmd := &cobra.Command{Use: "testapp"}

	// Define flags and unmarshal
	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), slog.LevelWarn, opts.LogLevel)
}

func (suite *structcliSuite) TestHooks_SlogLevelFromFlag() {
	opts := &slogLevelOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Define flags
	err := structcli.Define(cmd, opts)
	require.NoError(suite.T(), err)

	// Set flag value directly instead of using Execute
	err = cmd.Flags().Set("log-level", "error")
	require.NoError(suite.T(), err)

	err = structcli.Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), slog.LevelError, opts.LogLevel)
}

func (suite *structcliSuite) TestHooks_SlogLevelInvalidValue() {
	// Create a temporary config file with invalid level
	configContent := `log-level: invalid`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &slogLevelOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Set up viper to read from our config file
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	// Define flags and unmarshal
	structcli.Define(cmd, opts)
	err := structcli.Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid string for slog.Level")
}

func (suite *structcliSuite) TestHooks_SlogLevelWithOffsets() {
	// Test the ERROR+2, INFO-4 feature
	testCases := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR+2", slog.LevelError + 2},
		{"INFO-4", slog.LevelInfo - 4},
		{"WARN+1", slog.LevelWarn + 1},
		{"DEBUG-2", slog.LevelDebug - 2},
	}

	for _, tc := range testCases {
		suite.T().Run(fmt.Sprintf("level_%s", tc.input), func(t *testing.T) {
			// Reset viper state for each subtest
			viper.Reset()
			structcli.SetEnvPrefix("")

			// Create config file with the test level
			configContent := fmt.Sprintf(`log-level: %s`, tc.input)
			configFile := suite.createTempYAMLFile(configContent)
			defer os.Remove(configFile)

			opts := &slogLevelOptions{}
			cmd := &cobra.Command{Use: "test"}

			// Set up viper to read from our config file
			viper.SetConfigFile(configFile)
			require.NoError(t, viper.ReadInConfig())

			// Define flags and unmarshal
			err := structcli.Define(cmd, opts)
			require.NoError(t, err)

			err = structcli.Unmarshal(cmd, opts)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, opts.LogLevel,
				"Level %s should parse to %d but got %d", tc.input, tc.expected, opts.LogLevel)
		})
	}
}

func (suite *structcliSuite) TestStringToSlogLevelHookFunc_TypeGuard() {
	type testStruct struct {
		NotSlogLevel string `mapstructure:"level"`
	}

	opts := &testStruct{}

	// Use our slog hook directly with mapstructure
	// This will force the hook to be called even for non-slog.Level fields
	slogHook := internalhooks.StringToSlogLevelHookFunc()

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: slogHook, // Force slog hook to be used for all fields
		Result:     opts,
	})
	require.NoError(suite.T(), err)

	input := map[string]any{"level": "warn"}

	err = decoder.Decode(input)
	assert.NoError(suite.T(), err)

	// The string field should remain unchanged because the type guard
	// should have returned early when target type != slog.Level
	assert.Equal(suite.T(), "warn", opts.NotSlogLevel, "String should pass through unchanged due to type guard condition")
}

func (suite *structcliSuite) TestStringToZapcoreLevelHookFunc_TypeGuard() {
	type testStruct struct {
		NotZapcoreLevel string `mapstructure:"level"`
	}

	opts := &testStruct{}

	// Use our zapcore hook directly with mapstructure
	// This will force the hook to be called even for non-zapcore.Level fields
	zapcoreHook := internalhooks.StringToZapcoreLevelHookFunc()

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: zapcoreHook, // Force zapcore hook to be used for all fields
		Result:     opts,
	})
	require.NoError(suite.T(), err)

	// Input data - string that would be valid for zapcore.Level
	input := map[string]any{"level": "error"}

	err = decoder.Decode(input)
	assert.NoError(suite.T(), err)

	// The string field should remain unchanged because the type guard
	// should have returned early when target type != zapcore.Level
	assert.Equal(suite.T(), "error", opts.NotZapcoreLevel, "String should pass through unchanged due to type guard condition")
}
