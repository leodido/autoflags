package autoflags

import (
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func newTestCmd(path string) *cobra.Command {
	rootC := &cobra.Command{Use: "app"}
	parentC := rootC
	ret := &cobra.Command{Use: path}
	parentC.AddCommand(ret)

	return ret
}

func (suite *autoflagsSuite) TestCreateConfigC_EmptyGlobalSettings() {
	globalSettings := map[string]any{}

	result := createConfigC(globalSettings, newTestCmd("dns"))

	assert.Empty(suite.T(), result, "should return empty map when global settings are empty")
}

func (suite *autoflagsSuite) TestCreateConfigC_MissingCommandSection() {
	globalSettings := map[string]any{
		"loglevel":    "debug",
		"jsonlogging": true,
	}

	result := createConfigC(globalSettings, newTestCmd("dns"))

	expected := map[string]any{
		"loglevel":    "debug",
		"jsonlogging": true,
	}
	assert.Equal(suite.T(), expected, result, "should include only top-level settings when command section is missing")
}

func (suite *autoflagsSuite) TestCreateConfigC_WithCommandSection() {
	globalSettings := map[string]any{
		"loglevel":    "debug",
		"jsonlogging": true,
		"dns": map[string]any{
			"freeze": true,
			"cgroup": []string{"test"},
		},
	}

	result := createConfigC(globalSettings, newTestCmd("dns"))

	expected := map[string]any{
		"loglevel":    "debug",
		"jsonlogging": true,
		"freeze":      true,
		"cgroup":      []string{"test"},
	}
	assert.Equal(suite.T(), expected, result, "should merge top-level settings with promoted command-specific settings")
}

func (suite *autoflagsSuite) TestCreateConfigC_CommandSectionNotMap() {
	globalSettings := map[string]any{
		"loglevel": "debug",
		"dns":      "invalid-not-a-map",
		"tty":      42,
	}

	result := createConfigC(globalSettings, newTestCmd("dns"))

	expected := map[string]any{
		"loglevel": "debug",
		"dns":      "invalid-not-a-map",
		"tty":      42,
	}
	assert.Equal(suite.T(), expected, result, "should include command section as-is when it's not a map")
}

func (suite *autoflagsSuite) TestCreateConfigC_CommandSectionOverridesTopLevel() {
	globalSettings := map[string]any{
		"freeze":   false,
		"loglevel": "info",
		"dns": map[string]any{
			"freeze":   true,    // should override top-level
			"loglevel": "debug", // should override top-level
			"cgroup":   []string{"test"},
		},
	}

	result := createConfigC(globalSettings, newTestCmd("dns"))

	expected := map[string]any{
		"freeze":   true,             // from dns section
		"loglevel": "debug",          // from dns section
		"cgroup":   []string{"test"}, // from dns section
	}
	assert.Equal(suite.T(), expected, result, "command-specific settings should override top-level settings")
}

func (suite *autoflagsSuite) TestCreateConfigC_MultipleCommandSections() {
	globalSettings := map[string]any{
		"loglevel": "info",
		"dns": map[string]any{
			"freeze": true,
		},
		"tty": map[string]any{
			"ignore-comms": []string{"bash"},
		},
	}

	result := createConfigC(globalSettings, newTestCmd("dns"))

	expected := map[string]any{
		"loglevel": "info",
		"freeze":   true,
		// tty section should be excluded
	}
	assert.Equal(suite.T(), expected, result, "should only include the specific command section, excluding other command sections")
}

func (suite *autoflagsSuite) TestCreateConfigC_NestedCommandConfigurations() {
	globalSettings := map[string]any{
		"shared-setting": "value",
		"dns": map[string]any{
			"freeze": true,
			"nested": map[string]any{
				"deep-setting": "deep-value",
			},
		},
	}

	result := createConfigC(globalSettings, newTestCmd("dns"))

	expected := map[string]any{
		"shared-setting": "value",
		"freeze":         true,
		"nested": map[string]any{
			"deep-setting": "deep-value",
		},
	}
	assert.Equal(suite.T(), expected, result, "should preserve nested structures within command sections")
}

func (suite *autoflagsSuite) TestCreateConfigC_EmptyCommandSection() {
	globalSettings := map[string]any{
		"loglevel": "debug",
		"dns":      map[string]any{},
	}

	result := createConfigC(globalSettings, newTestCmd("dns"))

	expected := map[string]any{
		"loglevel": "debug",
	}
	assert.Equal(suite.T(), expected, result, "should handle empty command sections gracefully")
}

func (suite *autoflagsSuite) TestCreateConfigC_NilCommandSection() {
	globalSettings := map[string]any{
		"loglevel": "debug",
		"dns":      nil,
	}

	result := createConfigC(globalSettings, newTestCmd("dns"))

	expected := map[string]any{
		"loglevel": "debug",
		"dns":      nil,
	}
	assert.Equal(suite.T(), expected, result, "should handle nil command sections as non-maps")
}

func (suite *autoflagsSuite) TestCreateConfigC_TypeConflicts() {
	globalSettings := map[string]any{
		"timeout": "30s", // string at top level
		"dns": map[string]any{
			"timeout": 30, // int in command section
		},
	}

	result := createConfigC(globalSettings, newTestCmd("dns"))

	expected := map[string]any{
		"timeout": 30, // command section wins
	}
	assert.Equal(suite.T(), expected, result, "command section should override top-level even with type conflicts")
}

func (suite *autoflagsSuite) TestCreateConfigC_NestedSubcommand() {
	globalSettings := map[string]any{
		"toplevel": true,
		"usr": map[string]any{
			"intermediate": "should be ignored",
			"add": map[string]any{
				"name": "Leonardo",
				"age":  37,
			},
		},
	}

	rootCmd := &cobra.Command{Use: "app"}
	usrCmd := &cobra.Command{Use: "usr"}
	addCmd := &cobra.Command{Use: "add"}
	rootCmd.AddCommand(usrCmd)
	usrCmd.AddCommand(addCmd)

	result := createConfigC(globalSettings, addCmd)

	expected := map[string]any{
		"toplevel": true,
		"name":     "Leonardo",
		"age":      37,
	}
	assert.Equal(suite.T(), expected, result, "should merge top-level and deepest subcommand settings, ignoring intermediate")
}

func (suite *autoflagsSuite) TestCreateConfigC_NestedSubcommandFallback() {
	globalSettings := map[string]any{
		"toplevel": true,
		"usr": map[string]any{
			"email": "user@default.com",
			"perms": "read",
			"add":   map[string]any{}, // The 'add' section exists but it is empty
		},
	}
	// Mimics "app usr add"
	rootCmd := &cobra.Command{Use: "app"}
	usrCmd := &cobra.Command{Use: "usr"}
	addCmd := &cobra.Command{Use: "add"}
	rootCmd.AddCommand(usrCmd)
	usrCmd.AddCommand(addCmd)

	result := createConfigC(globalSettings, addCmd)

	// Fallback only from root level settings
	// Since 'add' is empty, it should not override the parent's settings
	expected := map[string]any{
		"toplevel": true,
	}
	assert.Equal(suite.T(), expected, result, "should use the deepest path found, even if empty, not parent's settings")
}

func (suite *autoflagsSuite) TestCreateConfigC_NestedSubcommandFallbackFromParent() {
	globalSettings := map[string]any{
		"toplevel": true,
		"usr": map[string]any{
			"email": "user@default.com",
			"perms": "read",
			// No 'delete' section
		},
	}

	// Mimics "app usr delete"
	rootCmd := &cobra.Command{Use: "app"}
	usrCmd := &cobra.Command{Use: "usr"}
	deleteCmd := &cobra.Command{Use: "delete"}
	rootCmd.AddCommand(usrCmd)
	usrCmd.AddCommand(deleteCmd)

	result := createConfigC(globalSettings, deleteCmd)

	// Since 'usr.delete' doesn't exist, nor top-level config keys for 'delete' exist...
	expected := map[string]any{
		"toplevel": true,
	}
	assert.Equal(suite.T(), expected, result, "should fall back to the parent command's settings if specific one is not found")
}
