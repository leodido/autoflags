package autoflags

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type autoflagsSuite struct {
	suite.Suite
}

func TestAutoflagsSuite(t *testing.T) {
	suite.Run(t, new(autoflagsSuite))
}

func (suite *autoflagsSuite) SetupTest() {
	// Reset the global boundEnvs map before each test
	boundEnvs = make(map[string]map[string]bool)
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
