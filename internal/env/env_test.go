package internalenv_test

import (
	"testing"

	"github.com/leodido/structcli"
	internalenv "github.com/leodido/structcli/internal/env"
	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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

// createTestC creates a command with flags that have environment annotations
func (suite *structcliSuite) createTestC(name string, flagsWithEnvs map[string][]string) *cobra.Command {
	cmd := &cobra.Command{
		Use: name,
	}

	for flagName, envVars := range flagsWithEnvs {
		cmd.Flags().String(flagName, "", "test flag")
		if len(envVars) > 0 {
			_ = cmd.Flags().SetAnnotation(flagName, internalenv.FlagAnnotation, envVars)
		}
	}

	return cmd
}

func (suite *structcliSuite) TestBindEnv_FirstCall() {
	cmd := suite.createTestC("dns", map[string][]string{
		"freeze": {"S4SONIC_DNS_FREEZE"},
		"cgroup": {"S4SONIC_DNS_CGROUP"},
	})

	internalenv.BindEnv(cmd)

	// Get the scope and check bound envs
	scope := internalscope.Get(cmd)
	boundEnvs := scope.GetBoundEnvs()

	assert.True(suite.T(), boundEnvs["freeze"], "freeze flag should be marked as bound")
	assert.True(suite.T(), boundEnvs["cgroup"], "cgroup flag should be marked as bound")
}

func (suite *structcliSuite) TestBindEnv_SecondCallSameCommand() {
	cmd := suite.createTestC("dns", map[string][]string{
		"freeze": {"S4SONIC_DNS_FREEZE"},
		"cgroup": {"S4SONIC_DNS_CGROUP"},
	})

	// First call
	internalenv.BindEnv(cmd)

	// Add a new flag to simulate second call (like dnsOpts.Attach after commonOpts.Attach)
	cmd.Flags().String("new-flag", "", "new test flag")
	_ = cmd.Flags().SetAnnotation("new-flag", internalenv.FlagAnnotation, []string{"S4SONIC_DNS_NEW_FLAG"})

	// Second call should not bind existing flags again, but should bind new flag
	internalenv.BindEnv(cmd)

	// Check bound envs
	scope := internalscope.Get(cmd)
	boundEnvs := scope.GetBoundEnvs()

	// Check that existing flags are still marked as bound (no duplicates)
	assert.True(suite.T(), boundEnvs["freeze"], "freeze flag should remain bound")
	assert.True(suite.T(), boundEnvs["cgroup"], "cgroup flag should remain bound")
	// New flag should be bound
	assert.True(suite.T(), boundEnvs["new-flag"], "new-flag should be bound")
}

func (suite *structcliSuite) TestBindEnv_DifferentCommands() {
	dnsCmd := suite.createTestC("dns", map[string][]string{
		"freeze": {"S4SONIC_DNS_FREEZE"},
	})

	ttyCmd := suite.createTestC("tty", map[string][]string{
		"freeze": {"S4SONIC_TTY_FREEZE"}, // Same flag name, different command
	})

	// Bind for both commands
	internalenv.BindEnv(dnsCmd)
	internalenv.BindEnv(ttyCmd)

	// Both commands should have their flags bound independently
	dnsScope := internalscope.Get(dnsCmd)
	dnsBoundEnvs := dnsScope.GetBoundEnvs()
	assert.True(suite.T(), dnsBoundEnvs["freeze"], "dns freeze flag should be bound")

	ttyScope := internalscope.Get(ttyCmd)
	ttyBoundEnvs := ttyScope.GetBoundEnvs()
	assert.True(suite.T(), ttyBoundEnvs["freeze"], "tty freeze flag should be bound")

	// Commands should be isolated - verify they have separate scopes
	assert.NotSame(suite.T(), dnsScope, ttyScope, "commands should have separate scopes")
	assert.Len(suite.T(), dnsBoundEnvs, 1, "dns should have exactly 1 bound env")
	assert.Len(suite.T(), ttyBoundEnvs, 1, "tty should have exactly 1 bound env")
}

func (suite *structcliSuite) TestBindEnv_FlagsWithoutEnvAnnotations() {
	cmd := suite.createTestC("dns", map[string][]string{
		"freeze": {"S4SONIC_DNS_FREEZE"}, // Has env annotation
		"no-env": {},                     // No env annotation
	})

	internalenv.BindEnv(cmd)

	// Only flags with env annotations should be tracked
	scope := internalscope.Get(cmd)
	boundEnvs := scope.GetBoundEnvs()

	assert.True(suite.T(), boundEnvs["freeze"], "freeze flag should be bound")
	assert.False(suite.T(), boundEnvs["no-env"], "no-env flag should not be bound")
}

func (suite *structcliSuite) TestBindEnv_EmptyCommand() {
	cmd := &cobra.Command{Use: "empty"}

	// Should not panic with empty command
	internalenv.BindEnv(cmd)

	// Should have scope but no bound envs
	scope := internalscope.Get(cmd)
	boundEnvs := scope.GetBoundEnvs()

	assert.NotNil(suite.T(), scope, "empty command should have a scope")
	assert.Empty(suite.T(), boundEnvs, "empty command should have no bound flags")
}

func (suite *structcliSuite) TestGetOrSetAppName_Consistency() {
	tests := []struct {
		descr          string
		setup          func()
		name           string
		cName          string
		expected       string
		expectedPrefix string
	}{
		{
			descr:          "provided name with no existing prefix",
			setup:          func() { structcli.SetEnvPrefix("") },
			name:           "myapp",
			cName:          "cmd",
			expected:       "myapp",
			expectedPrefix: "MYAPP",
		},
		{
			descr:          "fallback to command name",
			setup:          func() { structcli.SetEnvPrefix("") },
			name:           "",
			cName:          "mycmd",
			expected:       "mycmd",
			expectedPrefix: "MYCMD",
		},
		{
			descr:          "no given app name, use existing prefix",
			setup:          func() { structcli.SetEnvPrefix("already-existing") },
			name:           "",
			cName:          "cmd",
			expected:       "ALREADY_EXISTING",
			expectedPrefix: "ALREADY_EXISTING",
		},
		{
			descr:          "no prefix, no given app name, no command name",
			setup:          func() { structcli.SetEnvPrefix("") },
			name:           "",
			cName:          "",
			expected:       "",
			expectedPrefix: "",
		},
		{
			descr:          "prefix, no given app name, no command name",
			setup:          func() { structcli.SetEnvPrefix("prepre") },
			name:           "",
			cName:          "",
			expected:       "PREPRE",
			expectedPrefix: "PREPRE",
		},
		{
			descr:          "uppercase prefix, no given app name, no command name",
			setup:          func() { structcli.SetEnvPrefix("UPPERC") },
			name:           "",
			cName:          "",
			expected:       "UPPERC",
			expectedPrefix: "UPPERC",
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.descr, func(t *testing.T) {
			tt.setup()
			result := structcli.GetOrSetAppName(tt.name, tt.cName)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.expectedPrefix, structcli.EnvPrefix())
		})
	}
}
