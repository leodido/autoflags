package autoflags

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func (suite *autoflagsSuite) TestBindEnv_FirstCall() {
	v := viper.New()
	cmd := suite.createTestC("dns", map[string][]string{
		"freeze": {"S4SONIC_DNS_FREEZE"},
		"cgroup": {"S4SONIC_DNS_CGROUP"},
	})

	// First call should bind environment variables
	bindEnv(v, cmd)

	// Check that the boundEnvs tracking map is updated
	assert.True(suite.T(), boundEnvs["dns"]["freeze"], "freeze flag should be marked as bound")
	assert.True(suite.T(), boundEnvs["dns"]["cgroup"], "cgroup flag should be marked as bound")
}

func (suite *autoflagsSuite) TestBindEnv_SecondCallSameCommand() {
	v := viper.New()
	cmd := suite.createTestC("dns", map[string][]string{
		"freeze": {"S4SONIC_DNS_FREEZE"},
		"cgroup": {"S4SONIC_DNS_CGROUP"},
	})

	// First call
	bindEnv(v, cmd)

	// Add a new flag to simulate second call (like dnsOpts.Attach after commonOpts.Attach)
	cmd.Flags().String("new-flag", "", "new test flag")
	_ = cmd.Flags().SetAnnotation("new-flag", FlagEnvsAnnotation, []string{"S4SONIC_DNS_NEW_FLAG"})

	// Second call should not bind existing flags again, but should bind new flag
	bindEnv(v, cmd)

	// Check that existing flags are still marked as bound (no duplicates)
	assert.True(suite.T(), boundEnvs["dns"]["freeze"], "freeze flag should remain bound")
	assert.True(suite.T(), boundEnvs["dns"]["cgroup"], "cgroup flag should remain bound")
	// New flag should be bound
	assert.True(suite.T(), boundEnvs["dns"]["new-flag"], "new-flag should be bound")
}

func (suite *autoflagsSuite) TestBindEnv_DifferentCommands() {
	v1 := viper.New()
	v2 := viper.New()

	dnsCmd := suite.createTestC("dns", map[string][]string{
		"freeze": {"S4SONIC_DNS_FREEZE"},
	})

	ttyCmd := suite.createTestC("tty", map[string][]string{
		"freeze": {"S4SONIC_TTY_FREEZE"}, // Same flag name, different command
	})

	// Bind for both commands
	bindEnv(v1, dnsCmd)
	bindEnv(v2, ttyCmd)

	// Both commands should have their flags bound independently
	assert.True(suite.T(), boundEnvs["dns"]["freeze"], "dns freeze flag should be bound")
	assert.True(suite.T(), boundEnvs["tty"]["freeze"], "tty freeze flag should be bound")

	// Commands should be tracked separately - verify both keys exist
	assert.Contains(suite.T(), boundEnvs, "dns", "dns command should be tracked")
	assert.Contains(suite.T(), boundEnvs, "tty", "tty command should be tracked")
	assert.Len(suite.T(), boundEnvs, 2, "should have exactly 2 command entries")
}

func (suite *autoflagsSuite) TestBindEnv_FlagsWithoutEnvAnnotations() {
	v := viper.New()
	cmd := suite.createTestC("dns", map[string][]string{
		"freeze": {"S4SONIC_DNS_FREEZE"}, // Has env annotation
		"no-env": {},                     // No env annotation
	})

	bindEnv(v, cmd)

	// Only flags with env annotations should be tracked
	assert.True(suite.T(), boundEnvs["dns"]["freeze"], "freeze flag should be bound")
	assert.False(suite.T(), boundEnvs["dns"]["no-env"], "no-env flag should not be bound")
}

func (suite *autoflagsSuite) TestBindEnv_EmptyCommand() {
	v := viper.New()
	cmd := &cobra.Command{Use: "empty"}

	// Should not panic with empty command
	bindEnv(v, cmd)

	// Should initialize tracking map but have no entries
	assert.NotNil(suite.T(), boundEnvs["empty"], "empty command should have initialized tracking map")
	assert.Empty(suite.T(), boundEnvs["empty"], "empty command should have no bound flags")
}
