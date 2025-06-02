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

// SearchPathType represents different search path strategies for the configuration file
type SearchPathType int

const (
	// SearchPathEtc represents /etc/{app}
	SearchPathEtc SearchPathType = iota
	// SearchPathHomeHidden represents $HOME/.{app}
	SearchPathHomeHidden
	// SearchPathWorkingDirHidden represents $PWD/.{app}
	SearchPathWorkingDirHidden
	// SearchPathExecutableDirHidden represents {executable_dir}/.{app}
	SearchPathExecutableDirHidden
	// SearchPathCustom represents a custom path (must be provided in CustomPaths)
	SearchPathCustom
)

// ConfigType represents supported configuration file types
type ConfigType string

const (
	ConfigTypeYAML ConfigType = "yaml"
	ConfigTypeJSON ConfigType = "json"
	ConfigTypeTOML ConfigType = "toml"
)

// getConfigExtensions returns file extensions for config completion based on type
func getConfigExtensions(configType ConfigType) []string {
	switch configType {
	case ConfigTypeYAML:
		return []string{"yaml", "yml"}
	case ConfigTypeJSON:
		return []string{"json"}
	case ConfigTypeTOML:
		return []string{"toml"}
	default:
		return []string{"yaml", "yml", "json", "toml"}
	}
}

// ConfigOptions defines configuration file behavior
type ConfigOptions struct {
	AppName     string           // For default paths and env var name // FIXME: use prefix from SetEnvPrefix? use rootC name?
	FlagName    string           // Name of config flag (defaults to "config")
	FileName    string           // Config file name without extension (defaults to "config")
	ConfigType  ConfigType       // Config file type (defaults to yaml) // FIXME: should this be an array? does viper support config file that can be either yaml or json?
	EnvVar      string           // Environment variable (defaults to {APPNAME}_CONFIG)
	SearchPaths []SearchPathType // Search path strategies (defaults to common paths)
	CustomPaths []string         // Custom search paths (used with SearchPathCustom)
	Description string           // Flag description (if empty, uses default) // FIXME: not sure we need this (we could always generate it)
}

var defaultSearchPaths = []SearchPathType{
	SearchPathEtc,
	SearchPathExecutableDirHidden,
	SearchPathHomeHidden,
	SearchPathWorkingDirHidden,
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
	if cfgOpts.FileName == "" {
		cfgOpts.FileName = "config"
	}
	if cfgOpts.ConfigType == "" {
		cfgOpts.ConfigType = ConfigTypeYAML
	}
	if cfgOpts.EnvVar == "" && cfgOpts.AppName != "" {
		cfgOpts.EnvVar = fmt.Sprintf("%s_CONFIG", strings.ToUpper(cfgOpts.AppName))
	}
	if len(cfgOpts.SearchPaths) == 0 {
		cfgOpts.SearchPaths = defaultSearchPaths
	}
	if cfgOpts.Description == "" {
		if cfgOpts.AppName != "" {
			// FIXME: generate fallbacks to from SearchPaths
			cfgOpts.Description = fmt.Sprintf("config file (fallbacks to {/etc/%[1]s,$PWD/.%[1]s,$HOME/.%[1]s}/%s.%s))", cfgOpts.AppName, cfgOpts.FileName, cfgOpts.ConfigType)
		} else {
			cfgOpts.Description = "config file (searches in default locations if not specified)"
		}
	}

	// Create the config file variable
	configFile := ""

	// Add persistent flag to root command
	rootC.PersistentFlags().StringVar(&configFile, cfgOpts.FlagName, configFile, cfgOpts.Description)

	// Add filename completion
	extensions := getConfigExtensions(cfgOpts.ConfigType)
	if err := rootC.MarkPersistentFlagFilename(cfgOpts.FlagName, extensions...); err != nil {
		return fmt.Errorf("couldn't set filename completion: %w", err)
	}

	// Set up environment variable binding if specified
	if cfgOpts.EnvVar != "" {
		if err := rootC.PersistentFlags().SetAnnotation(cfgOpts.FlagName, FlagEnvsAnnotation, []string{cfgOpts.EnvVar}); err != nil {
			return fmt.Errorf("couldn't set environment variable annotation: %w", err)
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

	searchPaths := resolveSearchPaths(opts.SearchPaths, opts.CustomPaths, opts.AppName)
	for _, searchPath := range searchPaths {
		viper.AddConfigPath(searchPath)
	}

	viper.SetConfigName(opts.FileName)
	viper.SetConfigType(string(opts.ConfigType))
}

// resolveSearchPaths converts SearchPathType strategies to actual paths
func resolveSearchPaths(pathTypes []SearchPathType, customPaths []string, appName string) []string {
	var paths []string
	customIndex := 0

	for _, pathType := range pathTypes {
		switch pathType {
		case SearchPathEtc:
			if appName != "" {
				paths = append(paths, path.Join("/etc", appName))
			}

		case SearchPathHomeHidden:
			if appName != "" {
				if home, _ := os.UserHomeDir(); home != "" {
					paths = append(paths, path.Join(home, fmt.Sprintf(".%s", appName)))
				}
			}

		case SearchPathWorkingDirHidden:
			if appName != "" {
				if pwd, _ := os.Getwd(); pwd != "" {
					paths = append(paths, path.Join(pwd, fmt.Sprintf(".%s", appName)))
				}
			}

		case SearchPathExecutableDirHidden:
			if appName != "" {
				if exec, _ := os.Executable(); exec != "" {
					execDir := filepath.Dir(exec)
					paths = append(paths, path.Join(execDir, fmt.Sprintf(".%s", appName)))
				}
			}

		case SearchPathCustom:
			if customIndex < len(customPaths) {
				expandedPath := resolveSearchPath(customPaths[customIndex], appName)
				paths = append(paths, expandedPath)
				customIndex++
			}
		}
	}

	return paths
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
