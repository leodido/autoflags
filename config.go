package autoflags

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ConfigOptions defines configuration file behavior
type ConfigOptions struct {
	AppName     string   // For default paths and env var name // FIXME: use prefix from SetEnvPrefix?
	FlagName    string   // Name of config flag (defaults to "config")
	EnvVar      string   // Environment variable (defaults to {APPNAME}_CONFIG)
	SearchPaths []string // Custom search paths (if empty, uses defaults) // TODO: create enum of possible search paths
	Description string   // Flag description (if empty, uses default)
}

// SetupConfig creates the --config persistent flag and sets up viper search paths
func SetupConfig(rootC *cobra.Command, cfgOpts ConfigOptions) error {
	if rootC.Parent() != nil {
		return fmt.Errorf("SetupConfig must be called on the root command")
	}

	// Apply defaults
	if cfgOpts.FlagName == "" {
		cfgOpts.FlagName = "config"
	}
	if cfgOpts.EnvVar == "" && cfgOpts.AppName != "" {
		cfgOpts.EnvVar = fmt.Sprintf("%s_CONFIG", strings.ToUpper(cfgOpts.AppName))
	}
	// FIXME: set the default search paths
	if cfgOpts.Description == "" {
		if cfgOpts.AppName != "" {
			cfgOpts.Description = fmt.Sprintf("config file (fallbacks to {/etc/%[1]s,$PWD/.%[1]s,$HOME/.%[1]s}/config.yaml)", cfgOpts.AppName)
		} else {
			cfgOpts.Description = "config file (searches in default locations if not specified)"
		}
	}

	// Create the config file variable
	configFile := ""

	// Add persistent flag to root command
	rootC.PersistentFlags().StringVar(&configFile, cfgOpts.FlagName, configFile, cfgOpts.Description)

	// Add filename completion
	if err := rootC.MarkPersistentFlagFilename(cfgOpts.FlagName, "yaml"); err != nil {
		return fmt.Errorf("failed to set filename completion: %w", err)
	}

	// Set up environment variable binding if specified
	if cfgOpts.EnvVar != "" {
		if err := rootC.PersistentFlags().SetAnnotation(cfgOpts.FlagName, FlagEnvsAnnotation, []string{cfgOpts.EnvVar}); err != nil {
			return fmt.Errorf("failed to set environment variable annotation: %w", err)
		}
	}

	// Set up viper configuration
	cobra.OnInitialize(func() {
		setupConfig(configFile, cfgOpts)
	})

	// Store cleanup function
	cobra.OnFinalize(func() {
		viper.Reset()
	})

	return nil
}

// setupConfig handles the viper initialization
func setupConfig(configFile string, opts ConfigOptions) {
	if configFile != "" {
		// Use explicit config file
		viper.SetConfigFile(configFile)

		return
	}

	// FIXME: this can be simplied if we always have search paths (default or user provided)
	// Set up search paths
	if len(opts.SearchPaths) > 0 {
		// Use custom search paths
		for _, searchPath := range opts.SearchPaths {
			expandedPath := resolveSearchPath(searchPath, opts.AppName)
			viper.AddConfigPath(expandedPath)
		}
	} else if opts.AppName != "" {
		// Use default search paths
		home, _ := os.UserHomeDir()
		exec, _ := os.Executable()

		viper.AddConfigPath(path.Join("/etc", opts.AppName))
		viper.AddConfigPath(path.Join(filepath.Dir(exec), fmt.Sprintf(".%s", opts.AppName)))
		viper.AddConfigPath(path.Join(home, fmt.Sprintf(".%s", opts.AppName)))
	}

	viper.SetConfigName(opts.FlagName)
	viper.SetConfigType("yaml") // FIXME: should we make this configurable
}

// resolveSearchPath expands environment variables and placeholders in config paths
func resolveSearchPath(searchPath, appName string) string {
	expanded := os.ExpandEnv(searchPath)
	expanded = strings.ReplaceAll(expanded, "{APP}", appName)
	// Handle $PWD specially since os.ExpandEnv might not handle it
	if strings.Contains(expanded, "$PWD") {
		pwd, _ := os.Getwd()
		expanded = strings.ReplaceAll(expanded, "$PWD", pwd)
	}

	return expanded
}

func UseConfig(readWhen func() bool) (inUse bool, mes string, err error) {
	// Use the readWhen function to determine if we should read config
	if readWhen != nil && !readWhen() {
		return false, "", nil
	}

	if err := viper.ReadInConfig(); err == nil {
		return true, fmt.Sprintf("Using config file: %s", viper.ConfigFileUsed()), nil
	} else {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, ignore...
			return false, "Running without a configuration file", nil
		} else {
			// Config file was found but another error was produced
			return false, "", fmt.Errorf("Error running with config file: %s: %v", viper.ConfigFileUsed(), err)
		}
	}
}

// UseConfigSimple is a simpler version of UseConfig that uses cmd.IsAvailableCommand() as the readWhen function
//
// It does not check for the config file when the command is not available (eg., help)
func UseConfigSimple(c *cobra.Command) (inUse bool, message string, err error) {
	return UseConfig(func() bool {
		return c.IsAvailableCommand()
	})
}
