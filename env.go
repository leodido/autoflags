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

func bindEnv(v *viper.Viper, c *cobra.Command) {
	c.Flags().VisitAll(func(f *pflag.Flag) {
		if envs, defineEnv := f.Annotations[FlagEnvsAnnotation]; defineEnv {
			input := []string{f.Name}
			input = append(input, envs...)
			v.BindEnv(input...)
		}
	})
}

func getEnv(f reflect.StructField, inherit bool, path, alias string) ([]string, bool) {
	ret := []string{}

	env := f.Tag.Get("flagenv")
	defineEnv, _ := strconv.ParseBool(env)

	if defineEnv || inherit {
		if f.Type.Kind() != reflect.Struct {
			ret = append(ret, prefix+envRep.Replace(strings.ToUpper(path)))
			if alias != "" && path != alias {
				ret = append(ret, prefix+envRep.Replace(strings.ToUpper(alias)))
			}
		}
	}

	return ret, defineEnv
}

// FIXME: if a flag has flagrequired="true" and flagenv:"true" than flagrequired takes precedence and it forces you to always use the --flag
// FIXME: no real way to circumvent this... document it
