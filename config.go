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

// ConfigOptions defines configuration file behavior
type ConfigOptions struct {
	AppName     string           // For default paths and env var name (defaults to the name of the root command)
	FlagName    string           // Name of config flag (defaults to "config")
	ConfigName  string           // Config file name without extension (defaults to "config")
	EnvVar      string           // Environment variable (defaults to {APPNAME}_CONFIG)
	SearchPaths []SearchPathType // Search path strategies (defaults to common paths)
	CustomPaths []string         // Custom search paths (when SearchPaths contains SearchPathCustom)
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

	// Determine the app name
	appName := cfgOpts.AppName
	if appName == "" {
		appName = rootC.Name()
	}
	if appName == "" {
		return fmt.Errorf("couldn't determine the app name")
	}

	// Automatically set app name as the environment prefix
	if cfgOpts.AppName == "" {
		SetEnvPrefix(appName)
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
			cfgOpts.EnvVar = fmt.Sprintf("%s_CONFIG", normEnv(appName))
		}
	} else {
		cfgOpts.EnvVar = normEnv(cfgOpts.EnvVar)
	}
	if len(cfgOpts.SearchPaths) == 0 {
		cfgOpts.SearchPaths = defaultSearchPaths
	}

	descr := genDescription(appName, cfgOpts)
	configFile := ""

	// Add persistent flag to root command
	rootC.PersistentFlags().StringVar(&configFile, cfgOpts.FlagName, configFile, descr)

	// Add filename completion
	extensions := []string{"yaml", "yml", "json", "toml"}
	if err := rootC.MarkPersistentFlagFilename(cfgOpts.FlagName, extensions...); err != nil {
		return fmt.Errorf("couldn't set filename completion: %w", err)
	}

	// Set up viper configuration
	cobra.OnInitialize(func() {
		setupConfig(configFile, appName, cfgOpts)
	})

	// Store cleanup function
	cobra.OnFinalize(func() {
		viper.Reset()
	})

	return nil
}

// genDescription creates a description based on the search paths
func genDescription(appName string, opts ConfigOptions) string {
	templatePaths := resolveSearchPaths(opts.SearchPaths, opts.CustomPaths, appName, true)

	if len(templatePaths) == 0 {
		return "config file"
	}

	// Limit to first 3 examples to keep description reasonable
	if len(templatePaths) > 3 {
		templatePaths = templatePaths[:3]
		return fmt.Sprintf("config file (fallbacks to: {%s}/%s.{yaml,json,toml})", strings.Join(templatePaths, ","), opts.ConfigName)
	}

	return fmt.Sprintf("config file (fallbacks to: {%s}/%s.{yaml,json,toml})", strings.Join(templatePaths, ","), opts.ConfigName)
}

// setupConfig handles the viper initialization
func setupConfig(configFile string, appName string, opts ConfigOptions) {
	if cfgFile := strings.TrimSpace(configFile); cfgFile != "" {
		// Use explicit config file
		viper.SetConfigFile(configFile)

		return
	}

	if envConfigPath := strings.TrimSpace(os.Getenv(opts.EnvVar)); envConfigPath != "" {
		viper.SetConfigFile(envConfigPath)

		return
	}

	searchPaths := resolveSearchPaths(opts.SearchPaths, opts.CustomPaths, appName, false)
	for _, searchPath := range searchPaths {
		viper.AddConfigPath(searchPath)
	}

	// Viper will automatically try different extensions
	viper.SetConfigName(opts.ConfigName)
}

// resolveSearchPaths converts SearchPathType strategies to paths
// When mask=true, returns template paths for descriptions (e.g., $HOME, $PWD)
// When mask=false, returns actual resolved paths for viper
// appName is guaranteed to be non-empty by SetupConfig
func resolveSearchPaths(pathTypes []SearchPathType, customPaths []string, appName string, mask bool) []string {
	var paths []string
	customIndex := 0

	for _, pathType := range pathTypes {
		switch pathType {
		case SearchPathEtc:
			paths = append(paths, path.Join("/etc", appName))

		case SearchPathHomeHidden:
			if mask {
				paths = append(paths, path.Join("$HOME", fmt.Sprintf(".%s", appName)))
			} else {
				if home, _ := os.UserHomeDir(); home != "" {
					paths = append(paths, path.Join(home, fmt.Sprintf(".%s", appName)))
				}
			}

		case SearchPathWorkingDirHidden:
			if mask {
				paths = append(paths, path.Join("$PWD", fmt.Sprintf(".%s", appName)))
			} else {
				if pwd, _ := os.Getwd(); pwd != "" {
					paths = append(paths, path.Join(pwd, fmt.Sprintf(".%s", appName)))
				}
			}

		case SearchPathExecutableDirHidden:
			if mask {
				paths = append(paths, path.Join("{executable_dir}", fmt.Sprintf(".%s", appName)))
			} else {
				if exec, _ := os.Executable(); exec != "" {
					execDir := filepath.Dir(exec)
					paths = append(paths, path.Join(execDir, fmt.Sprintf(".%s", appName)))
				}
			}

		case SearchPathCustom:
			if customIndex < len(customPaths) {
				customPath := customPaths[customIndex]
				if mask {
					// For masked paths, show template with {APP} replaced but don't resolve env vars
					templatePath := strings.ReplaceAll(customPath, "{APP}", appName)
					paths = append(paths, templatePath)
				} else {
					// For actual paths, fully resolve environment variables and placeholders
					expandedPath := resolveSearchPath(customPath, appName)
					paths = append(paths, expandedPath)
				}
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
