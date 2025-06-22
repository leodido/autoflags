package full_example_cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/leodido/structcli"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fullAppTestCase struct {
	name        string
	args        []string
	envs        map[string]string
	config      string
	configPath  string
	exitOnDebug bool
	assertFunc  func(t *testing.T, output string, err error)
}

func TestFullApplication(t *testing.T) {
	testCases := []fullAppTestCase{
		{
			name: "Missing required options cause error",
			args: []string{"srv"},
			assertFunc: func(t *testing.T, output string, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, `"port" not set`)
			},
		},
		{
			name: "Recognize required options from flag",
			args: []string{"srv", "--port", "9876"},
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				require.Contains(t, output, `"Port": 9876`)
			},
		},
		{
			name:        "Debug with ExitOnDebug=true should NOT run the subcommand Run hook",
			args:        []string{"srv", "--debug-options", "-p", "3333"},
			exitOnDebug: true,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Values:")
				assert.NotContains(t, output, "|--srvC.RunE")
			},
		},
		{
			name:        "Debug with ExitOnDebug=true should NOT run the subcommand RunE hook",
			args:        []string{"usr", "add", "--debug-options", "--email", "leodido@linux.com", "--age", "37"},
			exitOnDebug: true,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "|-rootC.PersistentPreRunE")
				assert.Contains(t, output, "Values:")
				assert.Contains(t, output, "|---add.PreRunE")
				assert.NotContains(t, output, "|---add.RunE")
			},
		},
		{
			name:        "Debug with ExitOnDebug=false should run the subcommands Run hook",
			args:        []string{"srv", "--debug-options", "-p", "3333"},
			exitOnDebug: false,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Values:")
				assert.Contains(t, output, "|--srvC.RunE")
			},
		},
		{
			name:        "Debug with ExitOnDebug=false should run the subcommand RunE hook",
			args:        []string{"usr", "add", "--debug-options", "--email", "leodido@linux.com", "--age", "37"},
			exitOnDebug: false,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "|-rootC.PersistentPreRunE")
				assert.Contains(t, output, "Values:")
				assert.Contains(t, output, "|---add.PreRunE")
				assert.Contains(t, output, "|---add.RunE")
			},
		},
		{
			name: "Recognize required options from env",
			args: []string{"srv"},
			envs: map[string]string{"FULL_SRV_PORT": "4455"},
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				require.Contains(t, output, `"Port": 4455`)
			},
		},
		{
			name: "--debug-options prints out debugging info",
			args: []string{"srv", "--debug-options", "-p", "3333"},
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Aliases:")
				assert.Contains(t, output, "PFlags:")
				assert.Contains(t, output, "Env:")
				assert.Contains(t, output, "Config:")
				assert.Contains(t, output, "Defaults:")
				assert.Contains(t, output, "Values:")
			},
		},
		{
			name: "FULL_DEBUG_OPTIONS env var enables debug output",
			args: []string{"srv", "-p", "3333"},
			envs: map[string]string{"FULL_DEBUG_OPTIONS": "true"},
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Aliases:")
				assert.Contains(t, output, "Values:")
			},
		},
		{
			name: "Default values are applied correctly",
			args: []string{"srv", "--port", "1234"},
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `"Host": "localhost"`)
				assert.Contains(t, output, `"MaxConns": 10`)
				assert.Contains(t, output, `"TargetEnv": "dev"`)
			},
		},
		{
			name: "Custom flag --target-env works correctly",
			args: []string{"srv", "-p", "3333", "--target-env", "staging"},
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `"TargetEnv": "staging"`)
			},
		},
		{
			name: "Error on invalid --target-env value",
			args: []string{"srv", "-p", "1234", "--target-env", "ciao"},
			assertFunc: func(t *testing.T, output string, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "invalid environment: ciao")
			},
		},
		{
			name:       "Values are correctly read from config file in fallback location",
			args:       []string{"srv"},
			configPath: "/etc/full/config.yaml",
			config: `
srv:
  host: "host-from-config"
  port: 6767
  log-level: "warn"
  db-url: "postgres://user:pass@config/mydb"
`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Using config file: /etc/full/config.yaml")
				assert.Contains(t, output, `"Host": "host-from-config"`)
				assert.Contains(t, output, `"LogLevel": "warn"`)
				assert.Contains(t, output, `"Port": 6767`)
				assert.Contains(t, output, `"URL": "postgres://user:pass@config/mydb"`)
			},
		},
		{
			name:       "Values are correctly read from explicit config file and env var override works",
			args:       []string{"srv", "--config", "/some/path/config.yaml"},
			envs:       map[string]string{"FULL_SRV_APIKEY": "1terces", "FULL_SRV_DATABASE_MAXCONNS": "50"},
			configPath: "/some/path/config.yaml",
			config: `
srv:
  host: "host-from-config"
  port: 6767
  apikey: "secret1"
  db-url: "postgres://user:pass@config/mydb"
`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Using config file: /some/path/config.yaml")
				assert.Contains(t, output, `"Host": "host-from-config"`)
				assert.Contains(t, output, `"Port": 6767`)
				assert.Contains(t, output, `"APIKey": "1terces"`)
				assert.Contains(t, output, `"URL": "postgres://user:pass@config/mydb"`)
				assert.Contains(t, output, `"MaxConns": 50`)
			},
		},
		{
			name:       "Values are correctly read from explicit FULL_CONFIG and env var override works",
			args:       []string{"srv"},
			envs:       map[string]string{"FULL_CONFIG": "/some/path/config.yaml", "FULL_SRV_APIKEY": "1terces", "FULL_SRV_DATABASE_MAXCONNS": "50"},
			configPath: "/some/path/config.yaml",
			config: `
srv:
  host: "host-from-config"
  port: 6767
  apikey: "secret1"
  db-url: "postgres://user:pass@config/mydb"
`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, "Using config file: /some/path/config.yaml")
				assert.Contains(t, output, `"Host": "host-from-config"`)
				assert.Contains(t, output, `"Port": 6767`)
				assert.Contains(t, output, `"APIKey": "1terces"`)
				assert.Contains(t, output, `"URL": "postgres://user:pass@config/mydb"`)
				assert.Contains(t, output, `"MaxConns": 50`)
			},
		},
		{
			name: "Count flag verbosity is correctly tallied",
			args: []string{"srv", "version", "-vvv"},
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `"Verbose": 3`)
			},
		},
		{
			name: "Validation fails for invalid user email",
			args: []string{"usr", "add", "--name", "Test", "--email", "not-an-email", "--age", "30"},
			assertFunc: func(t *testing.T, output string, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "invalid options for add")
				assert.ErrorContains(t, err, "Field validation for 'Email' failed on the 'email' tag")
			},
		},
		{
			name: "Transformation (trim, title) is applied to user name",
			// Test the 'usr add' command and its TransformableOptions
			args: []string{"usr", "add", "--name", "  test user  ", "--email", "test@example.com", "--age", "30"},
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `"Name": "Test User"`)
			},
		},
		{
			name: "Flag value overrides both Environment and Config",
			args: []string{"srv", "--port", "1111"},      // Flag has highest precedence
			envs: map[string]string{"FULL_PORT": "2222"}, // Env has middle precedence
			config: `
srv:
	port: 3333 # Config has lowest precedence
`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `"Port": 1111`)
			},
		},
		{
			name:       "Malformed config file returns an error",
			args:       []string{"srv", "-p", "2233"},
			configPath: "/etc/full/config.yaml",
			config:     "srv:\n\tkey: value",
			assertFunc: func(t *testing.T, output string, err error) {
				// We expect an error from Viper when it tries to parse the bad YAML
				require.Error(t, err)
				assert.Contains(t, err.Error(), "error running with config file: /etc/full/config.yaml")
				assert.Contains(t, err.Error(), "parsing config: yaml")
			},
		},
		{
			name:       "Error on type mismatch when config key matches field name",
			args:       []string{"srv", "-p", "1234"},
			configPath: "/etc/full/config.yaml",
			config:     `dryrun: "b"`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "'DryRun' cannot parse value as 'bool'")
				assert.ErrorContains(t, err, `parsing "b": invalid syntax`)
			},
		},
		{
			name:       "Error on type mismatch when config key matches flag tag",
			args:       []string{"srv", "-p", "1234"},
			configPath: "/etc/full/config.yaml",
			config:     `dry: "a"`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "'DryRun' cannot parse value as 'bool'")
				assert.ErrorContains(t, err, `parsing "a": invalid syntax`)
			},
		},
		{
			name:       "Key matches flag tag for top-level field",
			args:       []string{"srv", "-p", "1234"},
			configPath: "/etc/full/config.yaml",
			config: `
srv:
  log-file: "/path/from/flag_tag"
`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `"LogFile": "/path/from/flag_tag"`)
			},
		},
		{
			name:       "Key matches field name for top-level field",
			args:       []string{"srv", "-p", "1234"},
			configPath: "/etc/full/config.yaml",
			config: `
srv:
  logfile: "/path/from/field_name"
`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `"LogFile": "/path/from/field_name"`)
			},
		},
		{
			name:       "Key is flattened flag tag for a nested field (level 1)",
			args:       []string{"srv", "-p", "1234"},
			configPath: "/etc/full/config.yaml",
			config: `
srv:
  db-url: "postgres://user:pass@flattened"
`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `"URL": "postgres://user:pass@flattened"`)
			},
		},
		{
			name:       "Key is flattened flag tag for a deeply nested field (level 2)",
			args:       []string{"srv", "-p", "1234"},
			configPath: "/etc/full/config.yaml",
			config: `
srv:
  deep-setting: "deep_value_from_flag_tag"
`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `    "Setting": "deep_value_from_flag_tag"`)
				assert.Contains(t, output, `      "Setting": "default-deeper-setting"`)

			},
		},
		{
			name:       "Key is dot-notation path for a nested field",
			args:       []string{"srv", "-p", "1234"},
			configPath: "/etc/full/config.yaml",
			config: `
srv:
  database.maxconns: 99
`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `"MaxConns": 99`)
			},
		},
		{
			name:       "Key is field name inside a nested map structure",
			args:       []string{"srv", "-p", "1234"},
			configPath: "/etc/full/config.yaml",
			config: `
srv:
  database:
    maxconns: 88
`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `"MaxConns": 88`)
			},
		},
		{
			name:       "Deeply nested config value",
			args:       []string{"srv", "-p", "1234"},
			configPath: "/etc/full/config.yaml",
			config: `
srv:
  deep:
    deeper:
      setting: "deepest_user_value"
`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `    "Setting": "default-deep-setting"`)
				assert.Contains(t, output, `      "Setting": "deepest_user_value"`)
			},
		},
		{
			name:       "Flattened deeply nested config value is used",
			args:       []string{"srv", "-p", "1234"},
			configPath: "/etc/full/config.yaml",
			config: `
srv:
  deeper-setting: "deepest_value_from_flat_key"
`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `    "Setting": "default-deep-setting"`)
				assert.Contains(t, output, `      "Setting": "deepest_value_from_flat_key"`)
			},
		},
		{
			name:       "More deeply nested struct paths",
			args:       []string{"srv", "-p", "1234"},
			configPath: "/etc/full/config.yaml",
			config: `
srv:
  deep:
    deeper:
      setting: abc
      nodefault: val
    setting: xyz
`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `    "Setting": "xyz"`)
				assert.Contains(t, output, `      "Setting": "abc"`)
				assert.Contains(t, output, `      "NoDefault": "val"`)
			},
		},
		{
			name:       "More deeply nested struct paths mixed with flag alias",
			args:       []string{"srv", "-p", "1234"},
			configPath: "/etc/full/config.yaml",
			config: `
srv:
  deep:
    deeper:
      setting: abc
      nodefault: val
  deep-setting: xyz
`,
			assertFunc: func(t *testing.T, output string, err error) {
				require.NoError(t, err)
				assert.Contains(t, output, `    "Setting": "xyz"`)
				assert.Contains(t, output, `      "Setting": "abc"`)
				assert.Contains(t, output, `      "NoDefault": "val"`)
			},
		},
	}

	setupTest := func(t *testing.T, content string, path string) func() {
		fs := afero.NewMemMapFs()
		viper.SetFs(fs)

		if content != "" && path != "" {
			// Ensure the directory exists and write the config file.
			require.NoError(t, fs.MkdirAll(filepath.Dir(path), 0755))
			require.NoError(t, afero.WriteFile(fs, path, []byte(content), 0644))
		}

		// Return a cleanup function to reset Viper's state after the test.
		return func() {
			viper.Reset()
			structcli.ResetGlobals()
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envs != nil {
				for key, value := range tc.envs {
					t.Setenv(key, value)
				}
			}
			cleanup := setupTest(t, tc.config, tc.configPath)
			defer cleanup()

			c, _ := NewRootC(tc.exitOnDebug)

			// Capture output
			var out bytes.Buffer
			c.SetOut(&out)
			c.SetErr(&out)

			// Set the arguments for this specific test case
			c.SetArgs(tc.args)

			// Execute the command
			executionErr := c.Execute()

			// Run the specific assertions for this test case
			tc.assertFunc(t, out.String(), executionErr)
		})
	}
}
