package autoflags

import (
	"github.com/stretchr/testify/assert"
)

func (suite *autoflagsSuite) TestCreateConfigC_EmptyGlobalSettings() {
	globalSettings := map[string]any{}
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

	assert.Empty(suite.T(), result, "should return empty map when global settings are empty")
}

func (suite *autoflagsSuite) TestCreateConfigC_MissingCommandSection() {
	globalSettings := map[string]any{
		"loglevel":    "debug",
		"jsonlogging": true,
	}
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

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
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

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
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

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
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

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
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

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
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

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
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

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
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

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
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

	expected := map[string]any{
		"timeout": 30, // command section wins
	}
	assert.Equal(suite.T(), expected, result, "command section should override top-level even with type conflicts")
}
