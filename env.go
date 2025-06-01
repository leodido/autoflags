package autoflags

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

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
	prefix = fmt.Sprintf("%s%s", strings.TrimSuffix(str, envSep), envSep)
}

func (ctx *defineContext) bindEnvironmentVariables() {
	ctx.targetF.VisitAll(func(f *pflag.Flag) {
		if envs, defineEnv := f.Annotations[FlagEnvsAnnotation]; defineEnv && len(envs) > 0 {
			// Only bind if we haven't already bound this env var for this command
			if !ctx.scope.isEnvBound(f.Name) {
				ctx.scope.bindEnv(f.Name)

				envBindingArgs := []string{f.Name}
				envBindingArgs = append(envBindingArgs, envs...)
				ctx.targetV.BindEnv(envBindingArgs...)
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
