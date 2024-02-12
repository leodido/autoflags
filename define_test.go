package autoflags

import (
	"context"
	"testing"
	"time"

	"github.com/leodido/autoflags/options"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type FlagsBaseSuite struct {
	suite.Suite
}

func TestFlagsBaseSuite(t *testing.T) {
	suite.Run(t, new(FlagsBaseSuite))
}

func (suite *FlagsBaseSuite) TestDefine() {
	cases := []struct {
		desc  string
		input options.Options
	}{
		{
			"flags definition from struct reference",
			&testOptions{},
		},
		{
			"flags definition from struct",
			testOptions{},
		},
	}

	confAnnotation := []string{"Configuration"}
	requiredAnnotation := []string{"true"}
	deepAnnotation := []string{"Deep"}

	for _, tc := range cases {
		suite.T().Run(tc.desc, func(t *testing.T) {
			c := &cobra.Command{}
			Define(c, tc.input)
			f := c.Flags()
			vip, e := Viper(c)
			assert.Nil(t, e)

			assert.NotNil(t, f.Lookup("log-level"))
			assert.Equal(t, "info", vip.Get("log-level"))
			assert.Equal(t, vip.Get("configflags.loglevel"), vip.Get("log-level"))
			assert.NotNil(t, f.Lookup("configflags.endpoint"))
			assert.NotNil(t, f.Lookup("configflags.timeout"))
			assert.NotNil(t, f.Lookup("log-level").Annotations[FlagGroupAnnotation])
			assert.Equal(t, confAnnotation, f.Lookup("log-level").Annotations[FlagGroupAnnotation])
			assert.NotNil(t, f.Lookup("configflags.endpoint").Annotations[FlagGroupAnnotation])
			assert.Equal(t, confAnnotation, f.Lookup("configflags.endpoint").Annotations[FlagGroupAnnotation])
			assert.NotNil(t, f.Lookup("configflags.endpoint").Annotations[cobra.BashCompOneRequiredFlag])
			assert.Equal(t, requiredAnnotation, f.Lookup("configflags.endpoint").Annotations[cobra.BashCompOneRequiredFlag])
			assert.NotNil(t, f.Lookup("configflags.timeout").Annotations[FlagGroupAnnotation])
			assert.Equal(t, confAnnotation, f.Lookup("configflags.timeout").Annotations[FlagGroupAnnotation])
			assert.Equal(t, "set the logging level", f.Lookup("log-level").Usage)
			assert.Equal(t, "the listen.dev endpoint emitting the verdicts", f.Lookup("configflags.endpoint").Usage)
			assert.Equal(t, "set the timeout, in seconds", f.Lookup("configflags.timeout").Usage)

			assert.NotNil(t, f.Lookup("nest.json"))
			assert.Nil(t, f.Lookup("nest.json").Annotations)
			assert.NotNil(t, f.Lookup("nest.jq"))
			assert.Nil(t, f.Lookup("nest.jq").Annotations)
			assert.NotNil(t, f.ShorthandLookup("q"))
			assert.Nil(t, f.ShorthandLookup("q").Annotations)
			assert.NotNil(t, f.Lookup("deep"))
			assert.NotNil(t, f.ShorthandLookup("d"))
			assert.Equal(t, "deepdown", vip.Get("nest.deep.deep"))
			assert.Equal(t, vip.Get("nest.deep.deep"), vip.Get("deep"))
			assert.NotNil(t, f.Lookup("deep").Annotations[FlagGroupAnnotation])
			assert.Equal(t, deepAnnotation, f.Lookup("deep").Annotations[FlagGroupAnnotation])
			assert.NotNil(t, f.Lookup("deep").Annotations[cobra.BashCompOneRequiredFlag])
			assert.Equal(t, requiredAnnotation, f.Lookup("deep").Annotations[cobra.BashCompOneRequiredFlag])
			assert.Equal(t, "output the verdicts (if any) in JSON form", f.Lookup("nest.json").Usage)
			assert.Equal(t, "filter the output using a jq expression", f.Lookup("nest.jq").Usage)
		})
	}
}

type ConfigFlags struct {
	LogLevel string `default:"info" flag:"log-level" flagdescr:"set the logging level" flaggroup:"Config"`
	Timeout  int    `flagdescr:"set the timeout, in seconds" flagset:"Config"`
	Endpoint string `flagdescr:"the listen.dev endpoint emitting the verdicts" flaggroup:"Config" flagrequired:"true"`
}

type DeepFlags struct {
	Deep time.Duration `default:"deepdown" flagdescr:"deep flag" flag:"deep" flagshort:"d" flaggroup:"Deep"`
}

type JSONFlags struct {
	JSON bool      `flagdescr:"output the verdicts (if any) in JSON form"`
	JQ   string    `flagshort:"q" flagdescr:"filter the output using a jq expression"`
	Deep DeepFlags `flagrequired:"true"`
}

type testOptions struct {
	ConfigFlags `flaggroup:"Configuration"`
	Nest        JSONFlags
}

func (o testOptions) Attach(c *cobra.Command)             {}
func (o testOptions) Transform(ctx context.Context) error { return nil }
func (o testOptions) Validate() []error                   { return nil }
