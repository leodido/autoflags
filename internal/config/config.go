package internalconfig

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/leodido/structcli/config"
	"github.com/spf13/viper"
)

// SetupConfig handles the viper initialization
func SetupConfig(configFile string, appName string, opts config.Options) {
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
func resolveSearchPaths(pathTypes []config.SearchPathType, customPaths []string, appName string, mask bool) []string {
	var paths []string
	customPathsUsed := false // Track if we've already added custom paths

	for _, pathType := range pathTypes {
		switch pathType {
		case config.SearchPathEtc:
			paths = append(paths, path.Join("/etc", appName))

		case config.SearchPathHomeHidden:
			if mask {
				paths = append(paths, path.Join("$HOME", fmt.Sprintf(".%s", appName)))
			} else {
				if home, _ := os.UserHomeDir(); home != "" {
					paths = append(paths, path.Join(home, fmt.Sprintf(".%s", appName)))
				}
			}

		case config.SearchPathWorkingDirHidden:
			if mask {
				paths = append(paths, path.Join("$PWD", fmt.Sprintf(".%s", appName)))
			} else {
				if pwd, _ := os.Getwd(); pwd != "" {
					paths = append(paths, path.Join(pwd, fmt.Sprintf(".%s", appName)))
				}
			}

		case config.SearchPathExecutableDirHidden:
			if mask {
				paths = append(paths, path.Join("{executable_dir}", fmt.Sprintf(".%s", appName)))
			} else {
				if exec, _ := os.Executable(); exec != "" {
					execDir := filepath.Dir(exec)
					paths = append(paths, path.Join(execDir, fmt.Sprintf(".%s", appName)))
				}
			}

		case config.SearchPathCustom:
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

// Description creates a description based on the search paths
func Description(appName string, opts config.Options) string {
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
