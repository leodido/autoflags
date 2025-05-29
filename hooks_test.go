package autoflags

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

type zapcoreLevelOptions struct {
	LogLevel zapcore.Level `default:"info" flagcustom:"true" flagdescr:"the logging level" flagenv:"true"`
}

func (o *zapcoreLevelOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestHooks_DefineZapcoreLevelFlag() {
	// Test just defining the flag, without config file
	opts := &zapcoreLevelOptions{}
	cmd := &cobra.Command{Use: "test"}

	Define(cmd, opts)

	// Check if the flag was created
	flag := cmd.Flags().Lookup("loglevel")
	assert.NotNil(suite.T(), flag)
}

func (suite *autoflagsSuite) TestHooks_ZapcoreLevelFromYAML() {
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
	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), zapcore.DebugLevel, opts.LogLevel)
}

type durationOptions struct {
	Timeout time.Duration `flag:"timeout" flagdescr:"request timeout" default:"30s"`
}

func (o *durationOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestHooks_DurationFromFlag() {
	// Test setting duration via command line flag
	opts := &durationOptions{}
	cmd := &cobra.Command{Use: "test"}

	Define(cmd, opts)

	// Set flag value
	err := cmd.Flags().Set("timeout", "45s")
	require.NoError(suite.T(), err)

	err = Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 45*time.Second, opts.Timeout)
}

func (suite *autoflagsSuite) TestHooks_DurationFromYAMLString() {
	// Test duration from YAML string format
	configContent := `timeout: "2m30s"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &durationOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	expected := 2*time.Minute + 30*time.Second
	assert.Equal(suite.T(), expected, opts.Timeout)
}

func (suite *autoflagsSuite) TestHooks_DurationFromYAMLNumber() {
	// Test duration from YAML number (nanoseconds)
	configContent := `timeout: 5000000000` // 5 seconds in nanoseconds
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &durationOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 5*time.Second, opts.Timeout)
}

func (suite *autoflagsSuite) TestHooks_DurationVariousFormats() {
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

			viper.Reset()
			viper.SetConfigFile(configFile)
			require.NoError(t, viper.ReadInConfig())

			Define(cmd, opts)
			err := Unmarshal(cmd, opts)

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, opts.Timeout)
		})
	}
}

func (suite *autoflagsSuite) TestHooks_DurationDefault() {
	// Test that default value is used when no config provided
	opts := &durationOptions{}
	cmd := &cobra.Command{Use: "test"}

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 30*time.Second, opts.Timeout) // from default:"30s"
}

func (suite *autoflagsSuite) TestHooks_DurationFlagOverridesConfig() {
	// Test flag precedence over config
	configContent := `timeout: "1m"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &durationOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)

	// Set flag value (should override config)
	err := cmd.Flags().Set("timeout", "90s")
	require.NoError(suite.T(), err)

	err = Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 90*time.Second, opts.Timeout) // flag wins over config
}

type stringSliceOptions struct {
	Cgroups []string `flag:"cgroups" flagdescr:"list of cgroups to monitor"`
}

func (o *stringSliceOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestHooks_StringSliceFromFlag() {
	// Test setting string slice via command line flag
	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	Define(cmd, opts)

	// Set flag value (simulating command line)
	err := cmd.Flags().Set("cgroups", "group1,group2,group3")
	require.NoError(suite.T(), err)

	err = Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group1", "group2", "group3"}, opts.Cgroups)
}

func (suite *autoflagsSuite) TestHooks_StringSliceFromYAMLCommaSeparated() {
	// Test hook converting comma-separated string from YAML to []string
	configContent := `cgroups: "group1,group2,group3"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group1", "group2", "group3"}, opts.Cgroups)
}

func (suite *autoflagsSuite) TestHooks_StringSliceFromYAMLArray() {
	// Test YAML array directly (no hook needed, mapstructure handles this)
	configContent := `cgroups:
  - group1
  - group2
  - group3`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group1", "group2", "group3"}, opts.Cgroups)
}

func (suite *autoflagsSuite) TestHooks_StringSliceEmptyString() {
	// Test hook behavior with empty string
	configContent := `cgroups: ""`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	// StringToSliceHookFunc with empty string results in []string{""}
	assert.Equal(suite.T(), []string{}, opts.Cgroups)
}

func (suite *autoflagsSuite) TestHooks_StringSliceSingleValue() {
	// Test hook with single value (no commas)
	configContent := `cgroups: "single-group"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"single-group"}, opts.Cgroups)
}

func (suite *autoflagsSuite) TestHooks_StringSliceWithSpaces() {
	// Test hook with values containing spaces
	configContent := `cgroups: "group with spaces,another group,normal"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group with spaces", "another group", "normal"}, opts.Cgroups)
}

func (suite *autoflagsSuite) TestHooks_StringSliceMultipleFlags() {
	// Test setting multiple flag values
	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	Define(cmd, opts)

	// Set flag multiple times
	err := cmd.Flags().Set("cgroups", "group1")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("cgroups", "group2")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("cgroups", "group3")
	require.NoError(suite.T(), err)

	err = Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group1", "group2", "group3"}, opts.Cgroups)
}

func (suite *autoflagsSuite) TestHooks_StringSliceFlagOverridesConfig() {
	// Test that flag values override config values
	configContent := `cgroups: "config1,config2"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)

	// Set flag value (should override config)
	err := cmd.Flags().Set("cgroups", "flag1,flag2,flag3")
	require.NoError(suite.T(), err)

	err = Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	// Flag values should win over config values
	assert.Equal(suite.T(), []string{"flag1", "flag2", "flag3"}, opts.Cgroups)
}

func (suite *autoflagsSuite) TestHooks_StringSliceYAMLSpecialCharacters() {
	// Test hook with special characters that might cause issues
	configContent := `cgroups: "group-1_test,group:2@domain,group[3]"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"group-1_test", "group:2@domain", "group[3]"}, opts.Cgroups)
}

func (suite *autoflagsSuite) TestHooks_StringSliceEmptyAfterSplit() {
	// Test hook behavior with leading/trailing commas
	configContent := `cgroups: ",group1,,group2,"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &stringSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	// Should include empty strings from the split
	assert.Equal(suite.T(), []string{"", "group1", "", "group2", ""}, opts.Cgroups)
}

type intSliceOptions struct {
	Ports []int `flag:"ports" flagdescr:"list of ports to listen on"`
}

func (o *intSliceOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestHooks_IntSliceFromFlag() {
	// Test setting int slice via command line flag
	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	Define(cmd, opts)

	// Set flag value (simulating command line)
	err := cmd.Flags().Set("ports", "8080,9090,3000")
	require.NoError(suite.T(), err)

	err = Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080, 9090, 3000}, opts.Ports)
}

func (suite *autoflagsSuite) TestHooks_IntSliceFromYAMLCommaSeparated() {
	// Test hook converting comma-separated string from YAML to []int
	configContent := `ports: "8080,9090,3000"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080, 9090, 3000}, opts.Ports)
}

func (suite *autoflagsSuite) TestHooks_IntSliceFromYAMLArray() {
	// Test YAML array directly (no hook needed, mapstructure handles this)
	configContent := `ports:
  - 8080
  - 9090
  - 3000`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080, 9090, 3000}, opts.Ports)
}

func (suite *autoflagsSuite) TestHooks_IntSliceEmptyString() {
	// Test hook behavior with empty string
	configContent := `ports: ""`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	// StringToIntSliceHookFunc with empty string results in empty slice
	assert.Equal(suite.T(), []int{}, opts.Ports)
}

func (suite *autoflagsSuite) TestHooks_IntSliceSingleValue() {
	// Test hook with single value (no commas)
	configContent := `ports: "8080"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080}, opts.Ports)
}

func (suite *autoflagsSuite) TestHooks_IntSliceWithSpaces() {
	// Test hook with values containing spaces (should be trimmed)
	configContent := `ports: " 8080 , 9090 , 3000 "`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080, 9090, 3000}, opts.Ports)
}

func (suite *autoflagsSuite) TestHooks_IntSliceNegativeNumbers() {
	// Test hook with negative numbers
	configContent := `ports: "-1,0,8080,-9090"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{-1, 0, 8080, -9090}, opts.Ports)
}

func (suite *autoflagsSuite) TestHooks_IntSliceMultipleFlags() {
	// Test setting multiple flag values
	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	Define(cmd, opts)

	// Set flag multiple times
	err := cmd.Flags().Set("ports", "8080")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("ports", "9090")
	require.NoError(suite.T(), err)
	err = cmd.Flags().Set("ports", "3000")
	require.NoError(suite.T(), err)

	err = Unmarshal(cmd, opts)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []int{8080, 9090, 3000}, opts.Ports)
}

func (suite *autoflagsSuite) TestHooks_IntSliceFlagOverridesConfig() {
	// Test that flag values override config values
	configContent := `ports: "8080,9090"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)

	// Set flag value (should override config)
	err := cmd.Flags().Set("ports", "3000,4000,5000")
	require.NoError(suite.T(), err)

	err = Unmarshal(cmd, opts)

	assert.NoError(suite.T(), err)
	// Flag values should win over config values
	assert.Equal(suite.T(), []int{3000, 4000, 5000}, opts.Ports)
}

func (suite *autoflagsSuite) TestHooks_IntSliceInvalidInteger() {
	// Test hook error handling with invalid integer
	configContent := `ports: "8080,invalid,3000"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid integer")
	assert.Contains(suite.T(), err.Error(), "invalid")
}

func (suite *autoflagsSuite) TestHooks_IntSliceFloatNumber() {
	// Test hook error handling with float number
	configContent := `ports: "8080,90.5,3000"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid integer")
	assert.Contains(suite.T(), err.Error(), "90.5")
}

func (suite *autoflagsSuite) TestHooks_IntSliceOutOfRange() {
	// Test hook error handling with number out of int range
	configContent := `ports: "8080,99999999999999999999,3000"`
	configFile := suite.createTempYAMLFile(configContent)
	defer os.Remove(configFile)

	opts := &intSliceOptions{}
	cmd := &cobra.Command{Use: "test"}

	viper.Reset()
	viper.SetConfigFile(configFile)
	require.NoError(suite.T(), viper.ReadInConfig())

	Define(cmd, opts)
	err := Unmarshal(cmd, opts)

	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "invalid integer")
}
