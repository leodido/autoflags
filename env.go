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
	FlagEnvsAnnotation = "___flagenvs"
)

func SetEnvPrefix(str string) {
	if str == "" {
		prefix = ""
		return
	}

	prefix = fmt.Sprintf("%s%s", strings.TrimSuffix(normEnv(str), envSep), envSep)
}

func EnvPrefix() string {
	return strings.TrimSuffix(prefix, envSep)
}

func normEnv(str string) string {
	return envRep.Replace(strings.ToUpper(str))
}

func bindEnv(c *cobra.Command) {
	s := getScope(c)

	c.Flags().VisitAll(func(f *pflag.Flag) {
		if envs, defineEnv := f.Annotations[FlagEnvsAnnotation]; defineEnv {
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
