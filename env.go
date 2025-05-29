package autoflags

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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
	prefix = fmt.Sprintf("%s%s", strings.TrimSuffix(str, envSep), envSep)
}

// boundEnvs tracks which environment variable have been bound for each command to prevent duplicates
var boundEnvs = make(map[string]map[string]bool)

func bindEnv(v *viper.Viper, c *cobra.Command) {
	cName := c.Name()
	if boundEnvs[cName] == nil {
		boundEnvs[cName] = make(map[string]bool)
	}

	c.Flags().VisitAll(func(f *pflag.Flag) {
		if envs, defineEnv := f.Annotations[FlagEnvsAnnotation]; defineEnv {
			// Only bind if we haven't already bound this env var for this command
			if !boundEnvs[cName][f.Name] {
				boundEnvs[cName][f.Name] = true
				input := []string{f.Name}
				input = append(input, envs...)
				v.BindEnv(input...)
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

			ret = append(ret, prefix+envRep.Replace(strings.ToUpper(envPath)))
			if alias != "" && path != alias {
				ret = append(ret, prefix+envRep.Replace(strings.ToUpper(envAlias)))
			}
		}
	}

	return ret, defineEnv
}

// FIXME: if a flag has flagrequired="true" and flagenv:"true" than flagrequired takes precedence and it forces you to always use the --flag
// FIXME: no real way to circumvent this... document it
