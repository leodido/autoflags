package autoflags

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

type AutoflagsSuite struct {
	suite.Suite
}

func TestAutoflagsSuite(t *testing.T) {
	suite.Run(t, new(AutoflagsSuite))
}

func (suite *AutoflagsSuite) SetupTest() {
	// Reset the global boundEnvs map before each test
	boundEnvs = make(map[string]map[string]bool)
}

// createTestC creates a command with flags that have environment annotations
func (suite *AutoflagsSuite) createTestC(name string, flagsWithEnvs map[string][]string) *cobra.Command {
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
