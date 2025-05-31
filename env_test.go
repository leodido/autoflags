package autoflags

import (
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func (suite *autoflagsSuite) TestBindEnv_FirstCall() {
	cmd := suite.createTestC("dns", map[string][]string{
		"freeze": {"S4SONIC_DNS_FREEZE"},
		"cgroup": {"S4SONIC_DNS_CGROUP"},
	})

	v := GetViper(cmd)
	bindEnv(v, cmd)

	// Get the scope and check bound envs
	scope := getScope(cmd)
	boundEnvs := scope.getBoundEnvs()

	assert.True(suite.T(), boundEnvs["freeze"], "freeze flag should be marked as bound")
	assert.True(suite.T(), boundEnvs["cgroup"], "cgroup flag should be marked as bound")
}

func (suite *autoflagsSuite) TestBindEnv_SecondCallSameCommand() {
	cmd := suite.createTestC("dns", map[string][]string{
		"freeze": {"S4SONIC_DNS_FREEZE"},
		"cgroup": {"S4SONIC_DNS_CGROUP"},
	})

	v := GetViper(cmd)

	// First call
	bindEnv(v, cmd)

	// Add a new flag to simulate second call (like dnsOpts.Attach after commonOpts.Attach)
	cmd.Flags().String("new-flag", "", "new test flag")
	_ = cmd.Flags().SetAnnotation("new-flag", FlagEnvsAnnotation, []string{"S4SONIC_DNS_NEW_FLAG"})

	// Second call should not bind existing flags again, but should bind new flag
	bindEnv(v, cmd)

	// Check bound envs
	scope := getScope(cmd)
	boundEnvs := scope.getBoundEnvs()

	// Check that existing flags are still marked as bound (no duplicates)
	assert.True(suite.T(), boundEnvs["freeze"], "freeze flag should remain bound")
	assert.True(suite.T(), boundEnvs["cgroup"], "cgroup flag should remain bound")
	// New flag should be bound
	assert.True(suite.T(), boundEnvs["new-flag"], "new-flag should be bound")
}

func (suite *autoflagsSuite) TestBindEnv_DifferentCommands() {
	dnsCmd := suite.createTestC("dns", map[string][]string{
		"freeze": {"S4SONIC_DNS_FREEZE"},
	})

	ttyCmd := suite.createTestC("tty", map[string][]string{
		"freeze": {"S4SONIC_TTY_FREEZE"}, // Same flag name, different command
	})

	// Bind for both commands
	v1 := GetViper(dnsCmd)
	bindEnv(v1, dnsCmd)

	v2 := GetViper(ttyCmd)
	bindEnv(v2, ttyCmd)

	// Both commands should have their flags bound independently
	dnsScope := getScope(dnsCmd)
	dnsBoundEnvs := dnsScope.getBoundEnvs()
	assert.True(suite.T(), dnsBoundEnvs["freeze"], "dns freeze flag should be bound")

	ttyScope := getScope(ttyCmd)
	ttyBoundEnvs := ttyScope.getBoundEnvs()
	assert.True(suite.T(), ttyBoundEnvs["freeze"], "tty freeze flag should be bound")

	// Commands should be isolated - verify they have separate scopes
	assert.NotSame(suite.T(), dnsScope, ttyScope, "commands should have separate scopes")
	assert.Len(suite.T(), dnsBoundEnvs, 1, "dns should have exactly 1 bound env")
	assert.Len(suite.T(), ttyBoundEnvs, 1, "tty should have exactly 1 bound env")
}

func (suite *autoflagsSuite) TestBindEnv_FlagsWithoutEnvAnnotations() {
	cmd := suite.createTestC("dns", map[string][]string{
		"freeze": {"S4SONIC_DNS_FREEZE"}, // Has env annotation
		"no-env": {},                     // No env annotation
	})

	v := GetViper(cmd)
	bindEnv(v, cmd)

	// Only flags with env annotations should be tracked
	scope := getScope(cmd)
	boundEnvs := scope.getBoundEnvs()

	assert.True(suite.T(), boundEnvs["freeze"], "freeze flag should be bound")
	assert.False(suite.T(), boundEnvs["no-env"], "no-env flag should not be bound")
}

func (suite *autoflagsSuite) TestBindEnv_EmptyCommand() {
	cmd := &cobra.Command{Use: "empty"}

	v := GetViper(cmd)

	// Should not panic with empty command
	bindEnv(v, cmd)

	// Should have scope but no bound envs
	scope := getScope(cmd)
	boundEnvs := scope.getBoundEnvs()

	assert.NotNil(suite.T(), scope, "empty command should have a scope")
	assert.Empty(suite.T(), boundEnvs, "empty command should have no bound flags")
}
