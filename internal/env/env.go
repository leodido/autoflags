package internalenv

import (
	"reflect"
	"strconv"
	"strings"

	internalscope "github.com/leodido/structcli/internal/scope"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	Prefix = ""
	EnvSep = "_"
	envRep = strings.NewReplacer("-", EnvSep, ".", EnvSep)
)

const (
	FlagAnnotation = "___leodido_structcli_flagenvs"
)

func NormEnv(str string) string {
	return envRep.Replace(strings.ToUpper(str))
}

func GetEnv(f reflect.StructField, inherit bool, path, alias, envPrefix string) ([]string, bool) {
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
				appName := strings.ToLower(strings.TrimSuffix(Prefix, "_"))
				if envPrefix != appName {
					envPath = envPrefix + "." + path
					if alias != "" {
						envAlias = envPrefix + "." + alias
					}
				}
			}

			ret = append(ret, Prefix+NormEnv(envPath))
			if alias != "" && path != alias {
				ret = append(ret, Prefix+NormEnv(envAlias))
			}
		}
	}

	return ret, defineEnv
}

func BindEnv(c *cobra.Command) {
	s := internalscope.Get(c)

	c.Flags().VisitAll(func(f *pflag.Flag) {
		if envs, defineEnv := f.Annotations[FlagAnnotation]; defineEnv {
			// Only bind if we haven't already bound this env var for this command
			if !s.IsEnvBound(f.Name) {
				s.SetBound(f.Name)
				input := []string{f.Name}
				input = append(input, envs...)
				s.Viper().BindEnv(input...)
			}
		}
	})
}
