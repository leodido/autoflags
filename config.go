package autoflags

import (
	"fmt"

	"github.com/leodido/autoflags/config"
	internalconfig "github.com/leodido/autoflags/internal/config"
	internalenv "github.com/leodido/autoflags/internal/env"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var defaultSearchPaths = []config.SearchPathType{
	config.SearchPathEtc,
	config.SearchPathExecutableDirHidden,
	config.SearchPathHomeHidden,
	config.SearchPathWorkingDirHidden,
}

// SetupConfig creates the --config global flag and sets up viper search paths.
//
// Works only for the root command.
func SetupConfig(rootC *cobra.Command, cfgOpts config.Options) error {
	if rootC.Parent() != nil {
		return fmt.Errorf("SetupConfig must be called on the root command")
	}

	// Determine the app name
	appName := GetOrSetAppName(cfgOpts.AppName, rootC.Name())
	if appName == "" {
		return fmt.Errorf("couldn't determine the app name")
	}

	// Apply defaults
	if cfgOpts.FlagName == "" {
		cfgOpts.FlagName = "config"
	}
	if cfgOpts.ConfigName == "" {
		cfgOpts.ConfigName = "config"
	}
	if cfgOpts.EnvVar == "" {
		if cfgOpts.AppName == "" {
			if currentPrefix := EnvPrefix(); currentPrefix != "" {
				cfgOpts.EnvVar = fmt.Sprintf("%s_CONFIG", currentPrefix)
			}
		} else {
			cfgOpts.EnvVar = fmt.Sprintf("%s_CONFIG", internalenv.NormEnv(appName))
		}
	} else {
		cfgOpts.EnvVar = internalenv.NormEnv(cfgOpts.EnvVar)
	}
	if len(cfgOpts.SearchPaths) == 0 {
		cfgOpts.SearchPaths = defaultSearchPaths
	}

	configFile := ""

	// Add persistent flag to root command
	rootC.PersistentFlags().StringVar(&configFile, cfgOpts.FlagName, configFile, internalconfig.Description(appName, cfgOpts))

	// Add filename completion
	extensions := []string{"yaml", "yml", "json", "toml"}
	if err := rootC.MarkPersistentFlagFilename(cfgOpts.FlagName, extensions...); err != nil {
		return fmt.Errorf("couldn't set filename completion: %w", err)
	}

	// Set up viper configuration
	cobra.OnInitialize(func() {
		internalconfig.SetupConfig(configFile, appName, cfgOpts)
	})

	// Store cleanup function
	cobra.OnFinalize(func() {
		viper.Reset()
	})

	// Regenerate usage templates for any commands already processed by Define()
	SetupUsage(rootC)

	return nil
}

// UseConfig attempts to read the configuration file based on the provided condition.
//
// The readWhen function determines whether config reading should be attempted.
// Returns whether config was loaded, a status message, and any error encountered.
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
			return false, "", fmt.Errorf("error running with config file: %s: %v", viper.ConfigFileUsed(), err)
		}
	}
}

// UseConfigSimple is a simpler version of UseConfig that uses cmd.IsAvailableCommand() as the readWhen function.
//
// It does not check for the config file when the command is not available (eg., help).
func UseConfigSimple(c *cobra.Command) (inUse bool, message string, err error) {
	return UseConfig(func() bool {
		return c.IsAvailableCommand()
	})
}
