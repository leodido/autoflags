package autoflags

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (suite *autoflagsSuite) TestBindEnvironmentVariables_PersistentRootFlags_GlobalViper() {
	originalPrefix := prefix
	SetEnvPrefix("S4SONIC")
	defer SetEnvPrefix(originalPrefix)

	cmdName := "rootapp"
	flagName := "global-freeze"
	expectedEnvVar := "S4SONIC_GLOBAL_FREEZE"

	cmd := suite.createTestC(cmdName, map[string][]string{
		flagName: {expectedEnvVar},
	}, true) // true for global flags

	runCtx := suite.getDefineContextForEnvTest(cmd, true)

	require.True(suite.T(), runCtx.isGlobalV, "Viper instance for registration should be global")
	require.Equal(suite.T(), viper.GetViper(), runCtx.targetV, "targetV should be the global Viper instance")

	runCtx.bindEnvironmentVariables()

	require.True(suite.T(), runCtx.scope.isEnvBound(flagName), "Persistent flag '%s' should be marked as bound in scope", flagName)

	err := os.Setenv(expectedEnvVar, "global_value_from_env")
	require.NoError(suite.T(), err)
	defer os.Unsetenv(expectedEnvVar)

	require.Equal(suite.T(), "global_value_from_env", runCtx.targetV.GetString(flagName),
		"Global viper should resolve persistent flag '%s' from its bound environment variable '%s'", flagName, expectedEnvVar)
}

func (suite *autoflagsSuite) TestBindEnvironmentVariables_LocalFlags_FirstCall() {
	originalPrefix := prefix
	SetEnvPrefix("S4SONIC")
	defer SetEnvPrefix(originalPrefix)

	cmdName := "dns"
	flagName := "freeze"
	expectedEnvVar := "S4SONIC_DNS_FREEZE"

	cmd := suite.createTestC(cmdName, map[string][]string{
		flagName: {expectedEnvVar},
	}, false)

	runCtx := suite.getDefineContextForEnvTest(cmd, false) // false for local flags

	runCtx.bindEnvironmentVariables()

	require.True(suite.T(), runCtx.scope.isEnvBound(flagName), "Flag '%s' should be marked as bound in scope", flagName)

	err := os.Setenv(expectedEnvVar, "value_from_env")
	require.NoError(suite.T(), err)
	defer os.Unsetenv(expectedEnvVar)

	require.Equal(suite.T(), "value_from_env", runCtx.targetV.GetString(flagName),
		"Scoped viper for command '%s' should resolve flag '%s' from its bound environment variable '%s'", cmdName, flagName, expectedEnvVar)
}

func (suite *autoflagsSuite) TestBindEnvironmentVariables_Idempotency() {
	cmd := suite.createTestC("idempotencycmd", map[string][]string{
		"myflag": {"MYAPP_IDEM_FLAG"},
	}, false)
	runCtx := suite.getDefineContextForEnvTest(cmd, false)

	runCtx.bindEnvironmentVariables()
	require.True(suite.T(), runCtx.scope.isEnvBound("myflag"), "Flag should be bound after first call")
	// Count the numer of bound env vars into scope
	countBefore := len(runCtx.scope.getBoundEnvs())

	// Second call should not bind existing flags again, but should bind new flag
	runCtx.bindEnvironmentVariables()
	countAfter := len(runCtx.scope.getBoundEnvs())

	require.Equal(suite.T(), countBefore, countAfter, "Number of uniquely bound envs in scope should not change on second call")
	assert.True(suite.T(), runCtx.scope.isEnvBound("myflag"), "Flag should still be marked as bound after second call")
}

func (suite *autoflagsSuite) TestBindEnvironmentVariables_EmptyFlagSet() {
	cmd := &cobra.Command{Use: "emptyflags"}

	runCtx := suite.getDefineContextForEnvTest(cmd, false)
	runCtx.targetF = cmd.Flags()

	require.NotPanics(suite.T(), func() {
		runCtx.bindEnvironmentVariables()
	}, "bindEnvironmentVariables should not panic with an empty FlagSet")

	require.Empty(suite.T(), runCtx.scope.getBoundEnvs(), "No envs should be bound in scope for an empty FlagSet")
}
