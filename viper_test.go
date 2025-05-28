package autoflags

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ViperTestSuite struct {
	suite.Suite
}

func TestViperTestSuite(t *testing.T) {
	suite.Run(t, new(ViperTestSuite))
}

func (suite *ViperTestSuite) TestCreateConfigC_EmptyGlobalSettings() {
	globalSettings := map[string]interface{}{}
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

	assert.Empty(suite.T(), result, "should return empty map when global settings are empty")
}

func (suite *ViperTestSuite) TestCreateConfigC_MissingCommandSection() {
	globalSettings := map[string]interface{}{
		"loglevel":    "debug",
		"jsonlogging": true,
	}
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

	expected := map[string]interface{}{
		"loglevel":    "debug",
		"jsonlogging": true,
	}
	assert.Equal(suite.T(), expected, result, "should include only top-level settings when command section is missing")
}

func (suite *ViperTestSuite) TestCreateConfigC_WithCommandSection() {
	globalSettings := map[string]interface{}{
		"loglevel":    "debug",
		"jsonlogging": true,
		"dns": map[string]interface{}{
			"freeze": true,
			"cgroup": []string{"test"},
		},
	}
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

	expected := map[string]interface{}{
		"loglevel":    "debug",
		"jsonlogging": true,
		"freeze":      true,
		"cgroup":      []string{"test"},
	}
	assert.Equal(suite.T(), expected, result, "should merge top-level settings with promoted command-specific settings")
}

func (suite *ViperTestSuite) TestCreateConfigC_CommandSectionNotMap() {
	globalSettings := map[string]interface{}{
		"loglevel": "debug",
		"dns":      "invalid-not-a-map",
		"tty":      42,
	}
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

	expected := map[string]interface{}{
		"loglevel": "debug",
		"dns":      "invalid-not-a-map",
		"tty":      42,
	}
	assert.Equal(suite.T(), expected, result, "should include command section as-is when it's not a map")
}

func (suite *ViperTestSuite) TestCreateConfigC_CommandSectionOverridesTopLevel() {
	globalSettings := map[string]interface{}{
		"freeze":   false,
		"loglevel": "info",
		"dns": map[string]interface{}{
			"freeze":   true,    // should override top-level
			"loglevel": "debug", // should override top-level
			"cgroup":   []string{"test"},
		},
	}
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

	expected := map[string]interface{}{
		"freeze":   true,             // from dns section
		"loglevel": "debug",          // from dns section
		"cgroup":   []string{"test"}, // from dns section
	}
	assert.Equal(suite.T(), expected, result, "command-specific settings should override top-level settings")
}

func (suite *ViperTestSuite) TestCreateConfigC_MultipleCommandSections() {
	globalSettings := map[string]interface{}{
		"loglevel": "info",
		"dns": map[string]interface{}{
			"freeze": true,
		},
		"tty": map[string]interface{}{
			"ignore-comms": []string{"bash"},
		},
	}
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

	expected := map[string]interface{}{
		"loglevel": "info",
		"freeze":   true,
		// tty section should be excluded
	}
	assert.Equal(suite.T(), expected, result, "should only include the specific command section, excluding other command sections")
}

func (suite *ViperTestSuite) TestCreateConfigC_NestedCommandConfigurations() {
	globalSettings := map[string]interface{}{
		"shared-setting": "value",
		"dns": map[string]interface{}{
			"freeze": true,
			"nested": map[string]interface{}{
				"deep-setting": "deep-value",
			},
		},
	}
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

	expected := map[string]interface{}{
		"shared-setting": "value",
		"freeze":         true,
		"nested": map[string]interface{}{
			"deep-setting": "deep-value",
		},
	}
	assert.Equal(suite.T(), expected, result, "should preserve nested structures within command sections")
}

func (suite *ViperTestSuite) TestCreateConfigC_EmptyCommandSection() {
	globalSettings := map[string]interface{}{
		"loglevel": "debug",
		"dns":      map[string]interface{}{},
	}
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

	expected := map[string]interface{}{
		"loglevel": "debug",
	}
	assert.Equal(suite.T(), expected, result, "should handle empty command sections gracefully")
}

func (suite *ViperTestSuite) TestCreateConfigC_NilCommandSection() {
	globalSettings := map[string]interface{}{
		"loglevel": "debug",
		"dns":      nil,
	}
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

	expected := map[string]interface{}{
		"loglevel": "debug",
		"dns":      nil,
	}
	assert.Equal(suite.T(), expected, result, "should handle nil command sections as non-maps")
}

func (suite *ViperTestSuite) TestCreateConfigC_TypeConflicts() {
	globalSettings := map[string]interface{}{
		"timeout": "30s", // string at top level
		"dns": map[string]interface{}{
			"timeout": 30, // int in command section
		},
	}
	commandName := "dns"

	result := createConfigC(globalSettings, commandName)

	expected := map[string]interface{}{
		"timeout": 30, // command section wins
	}
	assert.Equal(suite.T(), expected, result, "command section should override top-level even with type conflicts")
}
