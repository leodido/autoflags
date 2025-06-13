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

// SearchPathType represents different strategies for locating configuration files.
type SearchPathType int

const (
	// SearchPathEtc searches in /etc/{app} directory.
	SearchPathEtc SearchPathType = iota
	// SearchPathHomeHidden searches in $HOME/.{app} directory.
	SearchPathHomeHidden
	// SearchPathWorkingDirHidden searches in $PWD/.{app} directory.
	SearchPathWorkingDirHidden
	// SearchPathExecutableDirHidden searches in {executable_dir}/.{app} directory.
	SearchPathExecutableDirHidden
	// SearchPathCustom uses custom paths provided in CustomPaths field.
	SearchPathCustom
)

// ConfigOptions defines configuration file behavior and search paths.
type ConfigOptions struct {
	AppName     string
	FlagName    string           // Name of config flag (defaults to "config")
	ConfigName  string           // Config file name without extension (defaults to "config")
	EnvVar      string           // Environment variable (defaults to {APP}_CONFIG)
	SearchPaths []SearchPathType // Search path strategies (defaults to common paths)
	CustomPaths []string         // Custom search paths (when SearchPaths contains SearchPathCustom)
}

var defaultSearchPaths = []SearchPathType{
	SearchPathEtc,
	SearchPathExecutableDirHidden,
	SearchPathHomeHidden,
	SearchPathWorkingDirHidden,
}

// SetupConfig creates the --config global flag and sets up viper search paths.
//
// Works only for the root command.
func SetupConfig(rootC *cobra.Command, cfgOpts ConfigOptions) error {
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

	// Regenerate usage templates for any commands already processed by Define()
	SetupUsage(rootC)

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
	customPathsUsed := false // Track if we've already added custom paths

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
			// Add all custom paths at this position only once
			if !customPathsUsed {
				for _, customPath := range customPaths {
					if mask {
						// For masked paths, show template with {APP} replaced but don't resolve env vars
						templatePath := strings.ReplaceAll(customPath, "{APP}", appName)
						paths = append(paths, templatePath)
					} else {
						// For actual paths, fully resolve environment variables and placeholders
						expandedPath := resolveSearchPath(customPath, appName)
						paths = append(paths, expandedPath)
					}
				}
				customPathsUsed = true
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
