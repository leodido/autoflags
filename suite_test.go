package autoflags

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type autoflagsSuite struct {
	suite.Suite
}

func TestAutoflagsSuite(t *testing.T) {
	suite.Run(t, new(autoflagsSuite))
}

func (suite *autoflagsSuite) SetupTest() {
	// Reset viper state before each test to prevent test pollution
	viper.Reset()
	// Reset global prefix
	SetEnvPrefix("")
}

// createTestC creates a command with flags that have environment annotations
func (suite *autoflagsSuite) createTestC(name string, flagsWithEnvs map[string][]string) *cobra.Command {
	cmd := &cobra.Command{
		Use: name,
	}

	for flagName, envVars := range flagsWithEnvs {
		cmd.Flags().String(flagName, "", "test flag")
		if len(envVars) > 0 {
			_ = cmd.Flags().SetAnnotation(flagName, FlagEnvsAnnotation, envVars)
		}
	}

	return cmd
}

// createTempYAMLFile creates a temporary YAML files for testing
func (suite *autoflagsSuite) createTempYAMLFile(content string) string {
	tmpFile, err := os.CreateTemp("", "autoflags_test_*.yaml")
	require.NoError(suite.T(), err)

	_, err = tmpFile.WriteString(content)
	require.NoError(suite.T(), err)

	err = tmpFile.Close()
	require.NoError(suite.T(), err)

	return tmpFile.Name()
}
