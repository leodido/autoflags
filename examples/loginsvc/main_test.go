package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leodido/structcli"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type loginsvcTestCase struct {
	name         string
	args         []string
	envs         map[string]string
	config       string
	configPath   string
	passwordPipe string // Used to simulate stdin for password
	assertFunc   func(t *testing.T, output string, err error)
}

func TestLoginSvcApplication(t *testing.T) {
	testCases := []loginsvcTestCase{
		{
			name:         "Default loglevel is info",
			args:         []string{"user", "add", "-u", "test"},
			passwordPipe: "secretpass\n",
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `level:info`, "Default log level should be info")
				assert.Contains(t, output, `"M":"Attempting to add user"`)
				assert.Contains(t, output, "Added user 'test'")
			},
		},
		{
			name: "Flag --loglevel passed to root is propagated",
			// CORRECT SYNTAX: flag is before the subcommand
			args:         []string{"--loglevel", "debug", "user", "add", "-u", "test"},
			passwordPipe: "secretpass\n",
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `level:debug`, "Log level should be debug from root flag")
			},
		},
		{
			name:         "Flag --loglevel passed to intermediate command is propagated",
			args:         []string{"user", "--loglevel", "warn", "add", "-u", "test"},
			passwordPipe: "secretpass\n",
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `level:warn`, "Log level should be warn from user command flag")
			},
		},
		{
			name:         "Flag --loglevel passed to final command is propagated",
			args:         []string{"user", "add", "-u", "test", "--loglevel", "error"},
			passwordPipe: "secretpass\n",
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `level:error`, "Log level should be error from add command flag")
			},
		},
		{
			name:         "ENV variable overrides default",
			args:         []string{"user", "add", "-u", "test"},
			envs:         map[string]string{"LOGINSVC_LOGLEVEL": "warn"},
			passwordPipe: "secretpass\n",
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `level:warn`, "Log level should be warn due to environment variable")
			},
		},
		{
			name:         "Config file is used and propagated",
			args:         []string{"user", "add", "-u", "test", "--config", "/etc/loginsvc/config.yaml"},
			configPath:   "/etc/loginsvc/config.yaml",
			config:       `loglevel: "error"`,
			passwordPipe: "secretpass\n",
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Using config file: /etc/loginsvc/config.yaml")
				assert.Contains(t, output, `level:error`, "Log level should be error due to config file")
			},
		},
		{
			name:       "Flag overrides ENV and Config",
			args:       []string{"--loglevel", "debug", "user", "delete", "-u", "test"},
			envs:       map[string]string{"LOGINSVC_LOGLEVEL": "warn"},
			configPath: "/etc/loginsvc/config.yaml",
			config:     `loglevel: "error"`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Using config file: /etc/loginsvc/config.yaml")
				assert.Contains(t, output, `level:debug`, "Flag (debug) should have precedence over env (warn) and config (error)")
			},
		},
		{
			name: "Required local flag causes error if missing",
			args: []string{"user", "add"}, // Missing --username
			assertFunc: func(t *testing.T, output string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), `required flag(s) "username" not set`)
			},
		},
		{
			name:         "Empty password from stdin causes error",
			args:         []string{"user", "add", "-u", "test"},
			passwordPipe: "\n", // Just press enter
			assertFunc: func(t *testing.T, output string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), `password cannot be empty`)
			},
		},
	}

	// Helper function to set up the test environment
	setupTest := func(t *testing.T, content, path string) func() {
		// Use an in-memory filesystem for tests
		fs := afero.NewMemMapFs()
		viper.SetFs(fs)
		structcli.ResetGlobals()

		if content != "" && path != "" {
			require.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
			require.NoError(t, afero.WriteFile(fs, path, []byte(content), 0644))
		}

		// Return a cleanup function
		return func() {
			viper.Reset()
			structcli.ResetGlobals()
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables for this test case
			if tc.envs != nil {
				for key, value := range tc.envs {
					t.Setenv(key, value)
				}
			}
			cleanup := setupTest(t, tc.config, tc.configPath)
			defer cleanup()

			cmd, err := NewRootCmd()
			require.NoError(t, err)

			// Capture output
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)

			// Simulate stdin for password prompt if needed
			if tc.passwordPipe != "" {
				cmd.SetIn(strings.NewReader(tc.passwordPipe))
			}

			cmd.SetArgs(tc.args)
			executionErr := cmd.Execute()

			tc.assertFunc(t, out.String(), executionErr)
		})
	}
}
