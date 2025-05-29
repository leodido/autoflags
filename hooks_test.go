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
