package autoflags

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	envSep = "_"
	envRep = strings.NewReplacer("-", envSep, ".", envSep)
	prefix = ""
)

const (
	flagEnvsAnnotation = "___leodido_autoflags_flagenvs"
)

// GetOrSetAppName resolves the app name consistently.
//
// When name is given, use it (and set as prefix if none exists).
// When cName is given, use it if no prefix exists, or if existing prefix matches cName.
// Otherwise, when an environment prefix already exists, return the app name that corresponds to it.
// Finally, it falls back to empty string.
func GetOrSetAppName(name, cName string) string {
	// If a name was explicitly given then use it
	if name != "" {
		if EnvPrefix() == "" {
			// Also as a prefix if there's not one already
			SetEnvPrefix(name)
		}

		return name
	}

	existingPrefix := EnvPrefix()

	// When command name is given
	if cName != "" {
		if existingPrefix == "" {
			// No existing prefix, set it and return command name
			SetEnvPrefix(cName)

			return cName
		} else if strings.EqualFold(existingPrefix, cName) {
			// Existing prefix matches command name (case-insensitive)
			// This means the prefix was set by the command name, return command name

			return cName
		} else {
			// Existing prefix doesn't match command name
			// This means the prefix was set by an explicit AppName
			// Return the lowercase version of the prefix (to match original app name case)

			return existingPrefix
		}
	}

	// No command name provided, use existing prefix if available
	if existingPrefix != "" {
		return existingPrefix
	}

	return ""
}

// SetEnvPrefix sets the global environment variable prefix for the application.
//
// The prefix is automatically appended with an underscore when generating environment variable names.
func SetEnvPrefix(str string) {
	if str == "" {
		prefix = ""

		return
	}

	prefix = fmt.Sprintf("%s%s", strings.TrimSuffix(normEnv(str), envSep), envSep)
}

// EnvPrefix returns the current global environment variable prefix without the trailing underscore.
func EnvPrefix() string {
	return strings.TrimSuffix(prefix, envSep)
}

func normEnv(str string) string {
	return envRep.Replace(strings.ToUpper(str))
}

func bindEnv(c *cobra.Command) {
	s := getScope(c)

	c.Flags().VisitAll(func(f *pflag.Flag) {
		if envs, defineEnv := f.Annotations[flagEnvsAnnotation]; defineEnv {
			// Only bind if we haven't already bound this env var for this command
			if !s.isEnvBound(f.Name) {
				s.setBound(f.Name)
				input := []string{f.Name}
				input = append(input, envs...)
				s.viper().BindEnv(input...)
			}
		}
	})
}

func getEnv(f reflect.StructField, inherit bool, path, alias, envPrefix string) ([]string, bool) {
	ret := []string{}

	env := f.Tag.Get("flagenv")
	defineEnv, _ := strconv.ParseBool(env)

	if defineEnv || inherit {
		if f.Type.Kind() != reflect.Struct {
			envPath := path
			envAlias := alias

			// Apply env prefix to current env variable
			// But avoid double prefixing if the given prefix matches the global prefix (usually the CLI/app name)
			if envPrefix != "" {
				// Extract app name from prefix (remove trailing underscore and lowercase)
				appName := strings.ToLower(strings.TrimSuffix(prefix, "_"))
				if envPrefix != appName {
					envPath = envPrefix + "." + path
					if alias != "" {
						envAlias = envPrefix + "." + alias
					}
				}
			}

			ret = append(ret, prefix+normEnv(envPath))
			if alias != "" && path != alias {
				ret = append(ret, prefix+normEnv(envAlias))
			}
		}
	}

	return ret, defineEnv
}
