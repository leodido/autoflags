package autoflags_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leodido/autoflags"
	"github.com/leodido/autoflags/options"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type FlagsBaseSuite struct {
	suite.Suite
}

func TestFlagsBaseSuite(t *testing.T) {
	suite.Run(t, new(FlagsBaseSuite))
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
			autoflags.Define(c, tc.input)
			f := c.Flags()
			vip, e := autoflags.Viper(c)
			assert.Nil(t, e)

			assert.NotNil(t, f.Lookup("log-level"))
			assert.Equal(t, "info", vip.Get("log-level"))
			assert.Equal(t, vip.Get("configflags.loglevel"), vip.Get("log-level"))
			assert.NotNil(t, f.Lookup("configflags.endpoint"))
			assert.NotNil(t, f.Lookup("configflags.timeout"))
			assert.NotNil(t, f.Lookup("log-level").Annotations[autoflags.FlagGroupAnnotation])
			assert.Equal(t, confAnnotation, f.Lookup("log-level").Annotations[autoflags.FlagGroupAnnotation])
			assert.NotNil(t, f.Lookup("configflags.endpoint").Annotations[autoflags.FlagGroupAnnotation])
			assert.Equal(t, confAnnotation, f.Lookup("configflags.endpoint").Annotations[autoflags.FlagGroupAnnotation])
			assert.NotNil(t, f.Lookup("configflags.endpoint").Annotations[cobra.BashCompOneRequiredFlag])
			assert.Equal(t, requiredAnnotation, f.Lookup("configflags.endpoint").Annotations[cobra.BashCompOneRequiredFlag])
			assert.NotNil(t, f.Lookup("configflags.timeout").Annotations[autoflags.FlagGroupAnnotation])
			assert.Equal(t, confAnnotation, f.Lookup("configflags.timeout").Annotations[autoflags.FlagGroupAnnotation])
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
			assert.NotNil(t, f.Lookup("deep").Annotations[autoflags.FlagGroupAnnotation])
			assert.Equal(t, deepAnnotation, f.Lookup("deep").Annotations[autoflags.FlagGroupAnnotation])
			assert.NotNil(t, f.Lookup("deep").Annotations[cobra.BashCompOneRequiredFlag])
			assert.Equal(t, requiredAnnotation, f.Lookup("deep").Annotations[cobra.BashCompOneRequiredFlag])
			assert.Equal(t, "output the verdicts (if any) in JSON form", f.Lookup("nest.json").Usage)
			assert.Equal(t, "filter the output using a jq expression", f.Lookup("nest.jq").Usage)
		})
	}
}

type uintTestOptions struct {
	UintField   uint   `flag:"uint-field" flagdescr:"test uint field"`
	Uint8Field  uint8  `flag:"uint8-field" flagdescr:"test uint8 field"`
	Uint16Field uint16 `flag:"uint16-field" flagdescr:"test uint16 field"`
	Uint32Field uint32 `flag:"uint32-field" flagdescr:"test uint32 field"`
	Uint64Field uint64 `flag:"uint64-field" flagdescr:"test uint64 field"`
}

func (o uintTestOptions) Attach(c *cobra.Command)             {}
func (o uintTestOptions) Transform(ctx context.Context) error { return nil }
func (o uintTestOptions) Validate() []error                   { return nil }

func (suite *FlagsBaseSuite) TestDefine_UintTypesSupport() {
	opts := &uintTestOptions{
		UintField:   500,
		Uint8Field:  50,
		Uint16Field: 1000,
		Uint32Field: 100000,
		Uint64Field: 10000000000,
	}
	cmd := &cobra.Command{}

	autoflags.Define(cmd, opts)

	// Test uint
	flagUint := cmd.Flags().Lookup("uint-field")
	assert.NotNil(suite.T(), flagUint, "uint flag should be created")

	err := cmd.Flags().Set("uint-field", "1500")
	assert.NoError(suite.T(), err, "should be able to set uint flag")
	assert.Equal(suite.T(), uint(1500), opts.UintField, "uint struct field should be updated")

	// Test uint8
	flagUint8 := cmd.Flags().Lookup("uint8-field")
	assert.NotNil(suite.T(), flagUint8, "uint8 flag should be created")

	err = cmd.Flags().Set("uint8-field", "100")
	assert.NoError(suite.T(), err, "should be able to set uint8 flag")
	assert.Equal(suite.T(), uint8(100), opts.Uint8Field, "uint8 struct field should be updated")

	// Test uint16
	flag16 := cmd.Flags().Lookup("uint16-field")
	assert.NotNil(suite.T(), flag16, "uint16 flag should be created")

	err = cmd.Flags().Set("uint16-field", "2000")
	assert.NoError(suite.T(), err, "should be able to set uint16 flag")
	assert.Equal(suite.T(), uint16(2000), opts.Uint16Field, "uint16 struct field should be updated")

	// Test uint32
	flag32 := cmd.Flags().Lookup("uint32-field")
	assert.NotNil(suite.T(), flag32, "uint32 flag should be created")

	err = cmd.Flags().Set("uint32-field", "200000")
	assert.NoError(suite.T(), err, "should be able to set uint32 flag")
	assert.Equal(suite.T(), uint32(200000), opts.Uint32Field, "uint32 struct field should be updated")

	// Test uint64
	flag64 := cmd.Flags().Lookup("uint64-field")
	assert.NotNil(suite.T(), flag64, "uint64 flag should be created")

	err = cmd.Flags().Set("uint64-field", "20000000000")
	assert.NoError(suite.T(), err, "should be able to set uint64 flag")
	assert.Equal(suite.T(), uint64(20000000000), opts.Uint64Field, "uint64 struct field should be updated")
}

type intTestOptions struct {
	IntField   int   `flag:"int-field" flagdescr:"test int field"`
	Int8Field  int8  `flag:"int8-field" flagdescr:"test int8 field"`
	Int16Field int16 `flag:"int16-field" flagdescr:"test int16 field"`
	Int32Field int32 `flag:"int32-field" flagdescr:"test int32 field"`
	Int64Field int64 `flag:"int64-field" flagdescr:"test int64 field"`
}

func (o intTestOptions) Attach(c *cobra.Command)             {}
func (o intTestOptions) Transform(ctx context.Context) error { return nil }
func (o intTestOptions) Validate() []error                   { return nil }

func (suite *FlagsBaseSuite) TestDefine_IntTypesSupport() {
	opts := &intTestOptions{
		IntField:   1000,
		Int8Field:  42,
		Int16Field: 1234,
		Int32Field: 123456,
		Int64Field: 1234567890,
	}
	cmd := &cobra.Command{}

	autoflags.Define(cmd, opts)

	// Test int
	flagInt := cmd.Flags().Lookup("int-field")
	assert.NotNil(suite.T(), flagInt, "int flag should be created")

	err := cmd.Flags().Set("int-field", "2000")
	assert.NoError(suite.T(), err, "should be able to set int flag")
	assert.Equal(suite.T(), int(2000), opts.IntField, "int struct field should be updated")

	// Test int8
	flagInt8 := cmd.Flags().Lookup("int8-field")
	assert.NotNil(suite.T(), flagInt8, "int8 flag should be created")

	err = cmd.Flags().Set("int8-field", "100")
	assert.NoError(suite.T(), err, "should be able to set int8 flag")
	assert.Equal(suite.T(), int8(100), opts.Int8Field, "int8 struct field should be updated")

	// Test int16
	flagInt16 := cmd.Flags().Lookup("int16-field")
	assert.NotNil(suite.T(), flagInt16, "int16 flag should be created")

	err = cmd.Flags().Set("int16-field", "5678")
	assert.NoError(suite.T(), err, "should be able to set int16 flag")
	assert.Equal(suite.T(), int16(5678), opts.Int16Field, "int16 struct field should be updated")

	// Test int32
	flagInt32 := cmd.Flags().Lookup("int32-field")
	assert.NotNil(suite.T(), flagInt32, "int32 flag should be created")

	err = cmd.Flags().Set("int32-field", "987654")
	assert.NoError(suite.T(), err, "should be able to set int32 flag")
	assert.Equal(suite.T(), int32(987654), opts.Int32Field, "int32 struct field should be updated")

	// Test int64
	flagInt64 := cmd.Flags().Lookup("int64-field")
	assert.NotNil(suite.T(), flagInt64, "int64 flag should be created")

	err = cmd.Flags().Set("int64-field", "9876543210")
	assert.NoError(suite.T(), err, "should be able to set int64 flag")
	assert.Equal(suite.T(), int64(9876543210), opts.Int64Field, "int64 struct field should be updated")
}

type countTestOptions struct {
	Verbose int `flag:"verbose" flagshort:"v" type:"count" flagdescr:"verbosity level"`
}

func (o countTestOptions) Attach(c *cobra.Command)             {}
func (o countTestOptions) Transform(ctx context.Context) error { return nil }
func (o countTestOptions) Validate() []error                   { return nil }

func (suite *FlagsBaseSuite) TestDefine_CountFlagSupport() {
	opts := &countTestOptions{Verbose: 0}
	cmd := &cobra.Command{}

	autoflags.Define(cmd, opts)

	// Verify the flag was created
	flagVerbose := cmd.Flags().Lookup("verbose")
	assert.NotNil(suite.T(), flagVerbose, "verbose count flag should be created")

	// Verify short flag exists
	shortFlag := cmd.Flags().ShorthandLookup("v")
	assert.NotNil(suite.T(), shortFlag, "verbose short flag should be created")

	// Test count behavior - each flag usage increments the value
	err := cmd.Flags().Set("verbose", "3") // Should set to 3
	assert.NoError(suite.T(), err, "should be able to set count flag")
	assert.Equal(suite.T(), 3, opts.Verbose, "count flag should be set to 3")

	// Reset and test incremental behavior (this simulates -vvv)
	opts.Verbose = 0
	cmd.Flags().Set("verbose", "1") // First -v
	cmd.Flags().Set("verbose", "2") // Second -v (simulating -vv)
	cmd.Flags().Set("verbose", "3") // Third -v (simulating -vvv)

	assert.Equal(suite.T(), 3, opts.Verbose, "count flag should increment to 3")
}

type sliceTestOptions struct {
	StringSliceField []string `flag:"strings" flagshort:"s" flagdescr:"string slice field"`
	IntSliceField    []int    `flag:"ints" flagshort:"i" flagdescr:"int slice field"`
}

func (o sliceTestOptions) Attach(c *cobra.Command)             {}
func (o sliceTestOptions) Transform(ctx context.Context) error { return nil }
func (o sliceTestOptions) Validate() []error                   { return nil }

func (suite *FlagsBaseSuite) TestDefine_SliceSupport() {
	opts := &sliceTestOptions{
		StringSliceField: []string{"default1", "default2"},
		IntSliceField:    []int{1, 2, 3},
	}
	cmd := &cobra.Command{}

	autoflags.Define(cmd, opts)

	// Test string slice (should be supported)
	flagStrings := cmd.Flags().Lookup("strings")
	assert.NotNil(suite.T(), flagStrings, "string slice flag should be created")

	err := cmd.Flags().Set("strings", "value1,value2,value3")
	assert.NoError(suite.T(), err, "should be able to set string slice flag")

	expected := []string{"value1", "value2", "value3"}
	assert.Equal(suite.T(), expected, opts.StringSliceField, "string slice field should be updated")

	// Test int slice (should be supported)
	flagInts := cmd.Flags().Lookup("ints")
	assert.NotNil(suite.T(), flagInts, "int slice flag should be created")

	err = cmd.Flags().Set("ints", "10,20,30")
	assert.NoError(suite.T(), err, "should be able to set int slice flag")

	expectedInts := []int{10, 20, 30}
	assert.Equal(suite.T(), expectedInts, opts.IntSliceField, "int slice field should be updated")
}

func (suite *FlagsBaseSuite) TestDefine_NilPointerHandling() {
	// Test with nil pointer: it should not panic and should create same flags as zero-valued struct
	var nilOpts *testOptions = nil
	cmd1 := &cobra.Command{}

	assert.NotPanics(suite.T(), func() {
		autoflags.Define(cmd1, nilOpts)
	})

	// Should create same flags as zero-valued struct
	zeroOpts := &testOptions{}
	cmd2 := &cobra.Command{}
	autoflags.Define(cmd2, zeroOpts)

	// Count defined flags
	nilFlags := 0
	cmd1.Flags().VisitAll(func(flag *pflag.Flag) { nilFlags++ })

	zeroFlags := 0
	cmd2.Flags().VisitAll(func(flag *pflag.Flag) { zeroFlags++ })

	assert.Equal(suite.T(), zeroFlags, nilFlags, "nil pointer should create same flags as zero-valued struct")
}

type serverMode string

const (
	development serverMode = "dev"
	staging     serverMode = "staging"
	production  serverMode = "prod"
)

type comprehensiveCustomOptions struct {
	ServerMode serverMode `flagcustom:"true" flag:"server-mode" flagshort:"m" flagdescr:"set server mode"`
	SomeConfig string     `flagcustom:"true" flag:"some-config" flagshort:"c" flagdescr:"config file path"`
	NoMethod   string     `flagcustom:"true" flag:"no-method" flagdescr:"this should not appear"`
	NormalFlag string     `flag:"normal-flag" flagdescr:"normal description"`
}

func (o *comprehensiveCustomOptions) DefineServerMode(c *cobra.Command, typename, name, short, descr string) {
	enhancedDesc := descr + fmt.Sprintf(" (%s,%s,%s)", string(development), string(staging), string(production))
	c.Flags().StringP(name, short, string(development), enhancedDesc)

	// Add shell completion
	c.RegisterFlagCompletionFunc(name, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{string(development), string(staging), string(production)}, cobra.ShellCompDirectiveDefault
	})
}

func (o *comprehensiveCustomOptions) DefineSomeConfig(c *cobra.Command, typename, name, short, descr string) {
	enhancedDesc := descr + " (must be .yaml, .yml, or .json)"
	c.Flags().StringP(name, short, "", enhancedDesc)

	c.RegisterFlagCompletionFunc(name, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"yaml", "yml", "json"}, cobra.ShellCompDirectiveFilterFileExt
	})
}

func (o *comprehensiveCustomOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagcustom_ComprehensiveScenarios() {
	opts := &comprehensiveCustomOptions{}

	c := &cobra.Command{Use: "test"}
	autoflags.Define(c, opts)

	f := c.Flags()

	modeFlag := f.Lookup("server-mode")
	assert.NotNil(suite.T(), modeFlag, "server-mode flag should be defined")
	assert.Equal(suite.T(), "set server mode (dev,staging,prod)", modeFlag.Usage)

	configFlag := f.Lookup("some-config")
	assert.NotNil(suite.T(), configFlag, "config flag should be defined")
	assert.Equal(suite.T(), "config file path (must be .yaml, .yml, or .json)", configFlag.Usage)

	normalFlag := f.Lookup("normal-flag")
	assert.NotNil(suite.T(), normalFlag, "normal flags should still work")

	missingFlag := f.Lookup("no-method")
	assert.Nil(suite.T(), missingFlag, "flags without methods should be skipped")
}

type nestedStruct struct {
	Value string `flagdescr:"nested value"`
}

type structFieldOptions struct {
	Nest         nestedStruct `flagcustom:"true"`
	methodCalled bool
}

func (o *structFieldOptions) DefineNest(c *cobra.Command, typename, name, short, descr string) {
	o.methodCalled = true
}

func (o *structFieldOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagcustom_EdgeCases() {
	// Test struct fields (should be ignored)
	structOpts := &structFieldOptions{}
	c1 := &cobra.Command{Use: "test1"}
	autoflags.Define(c1, structOpts)

	assert.False(suite.T(), structOpts.methodCalled, "custom methods should not be called for struct fields")

	nestedFlag := c1.Flags().Lookup("nest.value")
	assert.NotNil(suite.T(), nestedFlag, "nested fields should be processed normally")
}

type envAnnotationsTestOptions struct {
	HasEnv string `flagenv:"true" flag:"has-env" flagdescr:"this will have len(envs) > 0"`
	NoEnv  string `flag:"no-env" flagdescr:"this will have len(envs) == 0"`
}

func (o *envAnnotationsTestOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestEnvAnnotations_WhenEnvsNotEmpty() {
	autoflags.SetEnvPrefix("TEST")

	opts := &envAnnotationsTestOptions{}
	c := &cobra.Command{Use: "test"}
	autoflags.Define(c, opts)

	f := c.Flags()

	// Case 1: len(envs) > 0 - should set annotation
	flagWithEnv := f.Lookup("has-env")
	assert.NotNil(suite.T(), flagWithEnv, "flag should exist")

	// The critical test: verify annotation was set
	envAnnotation := flagWithEnv.Annotations[autoflags.FlagEnvsAnnotation]
	assert.NotNil(suite.T(), envAnnotation, "annotation should be set when len(envs) > 0")
	assert.Greater(suite.T(), len(envAnnotation), 0, "annotation should contain env vars")
	assert.Contains(suite.T(), envAnnotation, "TEST_HAS_ENV", "should contain expected env var")
}

func (suite *FlagsBaseSuite) TestEnvAnnotations_WhenEnvsEmpty() {
	opts := &envAnnotationsTestOptions{}
	c := &cobra.Command{Use: "test"}
	autoflags.Define(c, opts)

	f := c.Flags()

	// Case 2: len(envs) == 0 - should NOT set annotation
	flagWithoutEnv := f.Lookup("no-env")
	assert.NotNil(suite.T(), flagWithoutEnv, "flag should exist")

	// The critical test: verify annotation was NOT set
	envAnnotation := flagWithoutEnv.Annotations[autoflags.FlagEnvsAnnotation]
	assert.Nil(suite.T(), envAnnotation, "annotation should NOT be set when len(envs) == 0")
}
