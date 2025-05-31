package autoflags_test

import (
	"context"
	"fmt"
	"reflect"
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
			vip := autoflags.GetViper(c)

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

type requiredFlagsTestOptions struct {
	RequiredFlag     string `flag:"required-flag" flagrequired:"true" flagdescr:"this flag is required"`
	NotRequiredFlag  string `flag:"not-required-flag" flagrequired:"false" flagdescr:"this flag is not required"`
	DefaultFlag      string `flag:"default-flag" flagdescr:"this flag has no flagrequired tag"`
	RequiredWithDesc string `flagrequired:"true" flagdescr:"required flag without custom name"`
}

func (o *requiredFlagsTestOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagrequired_BasicFunctionality() {
	opts := &requiredFlagsTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	autoflags.Define(cmd, opts)

	flags := cmd.Flags()

	// Test required flag
	requiredFlag := flags.Lookup("required-flag")
	assert.NotNil(suite.T(), requiredFlag, "required-flag should exist")

	// Check if the flag is marked as required using cobra's annotation
	requiredAnnotation := requiredFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.NotNil(suite.T(), requiredAnnotation, "required-flag should have required annotation")
	assert.Equal(suite.T(), []string{"true"}, requiredAnnotation, "required annotation should be 'true'")

	// Test not required flag
	notRequiredFlag := flags.Lookup("not-required-flag")
	assert.NotNil(suite.T(), notRequiredFlag, "not-required-flag should exist")

	notRequiredAnnotation := notRequiredFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.Nil(suite.T(), notRequiredAnnotation, "not-required-flag should not have required annotation")

	// Test default flag (no flagrequired tag)
	defaultFlag := flags.Lookup("default-flag")
	assert.NotNil(suite.T(), defaultFlag, "default-flag should exist")

	defaultAnnotation := defaultFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.Nil(suite.T(), defaultAnnotation, "default-flag should not have required annotation")

	// Test required flag without custom name
	autoNamedFlag := flags.Lookup("requiredwithdesc")
	assert.NotNil(suite.T(), autoNamedFlag, "requiredwithdesc should exist")

	autoNamedAnnotation := autoNamedFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.NotNil(suite.T(), autoNamedAnnotation, "requiredwithdesc should have required annotation")
	assert.Equal(suite.T(), []string{"true"}, autoNamedAnnotation, "required annotation should be 'true'")
}

type nestedRequiredFlagsOptions struct {
	TopLevel     string               `flag:"top-level" flagrequired:"true" flagdescr:"top level required flag"`
	NestedStruct nestedRequiredStruct `flaggroup:"Nested"`
}

type nestedRequiredStruct struct {
	NestedRequired    string `flag:"nested-required" flagrequired:"true" flagdescr:"nested required flag"`
	NestedNotRequired string `flag:"nested-not-required" flagdescr:"nested not required flag"`
}

func (o *nestedRequiredFlagsOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagrequired_NestedStructs() {
	opts := &nestedRequiredFlagsOptions{}
	cmd := &cobra.Command{Use: "test"}

	autoflags.Define(cmd, opts)

	flags := cmd.Flags()

	// Test top-level required flag
	topLevelFlag := flags.Lookup("top-level")
	assert.NotNil(suite.T(), topLevelFlag, "top-level should exist")

	topLevelAnnotation := topLevelFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.NotNil(suite.T(), topLevelAnnotation, "top-level should have required annotation")
	assert.Equal(suite.T(), []string{"true"}, topLevelAnnotation)

	// Test nested required flag
	nestedRequiredFlag := flags.Lookup("nested-required")
	assert.NotNil(suite.T(), nestedRequiredFlag, "nested-required should exist")

	nestedRequiredAnnotation := nestedRequiredFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.NotNil(suite.T(), nestedRequiredAnnotation, "nested-required should have required annotation")
	assert.Equal(suite.T(), []string{"true"}, nestedRequiredAnnotation)

	// Test nested not required flag
	nestedNotRequiredFlag := flags.Lookup("nested-not-required")
	assert.NotNil(suite.T(), nestedNotRequiredFlag, "nested-not-required should exist")

	nestedNotRequiredAnnotation := nestedNotRequiredFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.Nil(suite.T(), nestedNotRequiredAnnotation, "nested-not-required should not have required annotation")
}

type invalidBooleanRequiredOptions struct {
	InvalidTrue   string `flag:"invalid-true" flagrequired:"yes" flagdescr:"invalid boolean value"`
	InvalidFalse  string `flag:"invalid-false" flagrequired:"no" flagdescr:"invalid boolean value"`
	EmptyRequired string `flag:"empty-required" flagrequired:"" flagdescr:"empty flagrequired value"`
	CaseVariation string `flag:"case-variation" flagrequired:"True" flagdescr:"case variation test"`
}

func (o *invalidBooleanRequiredOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagrequired_InvalidBooleanValues() {
	opts := &invalidBooleanRequiredOptions{}
	cmd := &cobra.Command{Use: "test"}

	autoflags.Define(cmd, opts)

	flags := cmd.Flags()

	// Test invalid "yes" - should be treated as false since strconv.ParseBool returns false for invalid values
	invalidTrueFlag := flags.Lookup("invalid-true")
	assert.NotNil(suite.T(), invalidTrueFlag, "invalid-true should exist")

	invalidTrueAnnotation := invalidTrueFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.Nil(suite.T(), invalidTrueAnnotation, "invalid-true should not have required annotation due to invalid boolean")

	// Test invalid "no" - should be treated as false
	invalidFalseFlag := flags.Lookup("invalid-false")
	assert.NotNil(suite.T(), invalidFalseFlag, "invalid-false should exist")

	invalidFalseAnnotation := invalidFalseFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.Nil(suite.T(), invalidFalseAnnotation, "invalid-false should not have required annotation due to invalid boolean")

	// Test empty value - should be treated as false
	emptyRequiredFlag := flags.Lookup("empty-required")
	assert.NotNil(suite.T(), emptyRequiredFlag, "empty-required should exist")

	emptyRequiredAnnotation := emptyRequiredFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.Nil(suite.T(), emptyRequiredAnnotation, "empty-required should not have required annotation due to empty value")

	// Test case variation - "True" should work since strconv.ParseBool accepts it
	caseVariationFlag := flags.Lookup("case-variation")
	assert.NotNil(suite.T(), caseVariationFlag, "case-variation should exist")

	caseVariationAnnotation := caseVariationFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.NotNil(suite.T(), caseVariationAnnotation, "case-variation should have required annotation since 'True' is valid")
	assert.Equal(suite.T(), []string{"true"}, caseVariationAnnotation)
}

type multipleTypesRequiredOptions struct {
	RequiredString    string   `flag:"required-string" flagrequired:"true" flagdescr:"required string"`
	RequiredInt       int      `flag:"required-int" flagrequired:"true" flagdescr:"required int"`
	RequiredBool      bool     `flag:"required-bool" flagrequired:"true" flagdescr:"required bool"`
	RequiredSlice     []string `flag:"required-slice" flagrequired:"true" flagdescr:"required slice"`
	NotRequiredString string   `flag:"not-required-string" flagrequired:"false" flagdescr:"not required string"`
	NotRequiredInt    int      `flag:"not-required-int" flagrequired:"false" flagdescr:"not required int"`
}

func (o *multipleTypesRequiredOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagrequired_MultipleTypes() {
	opts := &multipleTypesRequiredOptions{}
	cmd := &cobra.Command{Use: "test"}

	autoflags.Define(cmd, opts)

	flags := cmd.Flags()

	// Test all required flags
	requiredFlags := []string{"required-string", "required-int", "required-bool", "required-slice"}
	for _, flagName := range requiredFlags {
		flag := flags.Lookup(flagName)
		assert.NotNil(suite.T(), flag, "%s should exist", flagName)

		annotation := flag.Annotations[cobra.BashCompOneRequiredFlag]
		assert.NotNil(suite.T(), annotation, "%s should have required annotation", flagName)
		assert.Equal(suite.T(), []string{"true"}, annotation, "%s required annotation should be 'true'", flagName)
	}

	// Test all not required flags
	notRequiredFlags := []string{"not-required-string", "not-required-int"}
	for _, flagName := range notRequiredFlags {
		flag := flags.Lookup(flagName)
		assert.NotNil(suite.T(), flag, "%s should exist", flagName)

		annotation := flag.Annotations[cobra.BashCompOneRequiredFlag]
		assert.Nil(suite.T(), annotation, "%s should not have required annotation", flagName)
	}
}

type requiredWithOtherTagsOptions struct {
	RequiredWithDefault string `flag:"required-default" flagrequired:"true" default:"default-value" flagdescr:"required with default"`
	RequiredWithGroup   string `flag:"required-group" flagrequired:"true" flaggroup:"TestGroup" flagdescr:"required with group"`
	RequiredWithShort   string `flag:"required-short" flagrequired:"true" flagshort:"r" flagdescr:"required with short"`
	RequiredWithEnv     string `flag:"required-env" flagrequired:"true" flagenv:"true" flagdescr:"required with env"`
}

func (o *requiredWithOtherTagsOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagrequired_CombinedWithOtherTags() {
	opts := &requiredWithOtherTagsOptions{}
	cmd := &cobra.Command{Use: "test"}

	autoflags.Define(cmd, opts)

	flags := cmd.Flags()

	// Test required with default
	requiredDefaultFlag := flags.Lookup("required-default")
	assert.NotNil(suite.T(), requiredDefaultFlag, "required-default should exist")

	requiredDefaultAnnotation := requiredDefaultFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.NotNil(suite.T(), requiredDefaultAnnotation, "required-default should have required annotation")
	assert.Equal(suite.T(), []string{"true"}, requiredDefaultAnnotation)
	assert.Equal(suite.T(), "default-value", requiredDefaultFlag.DefValue, "required-default should have default value")

	// Test required with group
	requiredGroupFlag := flags.Lookup("required-group")
	assert.NotNil(suite.T(), requiredGroupFlag, "required-group should exist")

	requiredGroupAnnotation := requiredGroupFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.NotNil(suite.T(), requiredGroupAnnotation, "required-group should have required annotation")
	assert.Equal(suite.T(), []string{"true"}, requiredGroupAnnotation)

	groupAnnotation := requiredGroupFlag.Annotations[autoflags.FlagGroupAnnotation]
	assert.NotNil(suite.T(), groupAnnotation, "required-group should have group annotation")
	assert.Equal(suite.T(), []string{"TestGroup"}, groupAnnotation)

	// Test required with short
	requiredShortFlag := flags.Lookup("required-short")
	assert.NotNil(suite.T(), requiredShortFlag, "required-short should exist")

	requiredShortAnnotation := requiredShortFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.NotNil(suite.T(), requiredShortAnnotation, "required-short should have required annotation")
	assert.Equal(suite.T(), []string{"true"}, requiredShortAnnotation)
	assert.Equal(suite.T(), "r", requiredShortFlag.Shorthand, "required-short should have shorthand")

	// Test required with env
	requiredEnvFlag := flags.Lookup("required-env")
	assert.NotNil(suite.T(), requiredEnvFlag, "required-env should exist")

	requiredEnvAnnotation := requiredEnvFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.NotNil(suite.T(), requiredEnvAnnotation, "required-env should have required annotation")
	assert.Equal(suite.T(), []string{"true"}, requiredEnvAnnotation)

	envAnnotation := requiredEnvFlag.Annotations[autoflags.FlagEnvsAnnotation]
	assert.NotNil(suite.T(), envAnnotation, "required-env should have env annotation")
}

type embeddedStruct struct {
	Value string `flag:"embedded-value" flagdescr:"embedded field"`
}

func (e embeddedStruct) GetValue() string { return e.Value }

type testInterface interface {
	GetValue() string
}

type testOptionsWithInterface struct {
	NormalField    string        `flag:"normal" flagdescr:"normal field"`
	InterfaceField testInterface // Interface fields can create addressability issues
}

func (o testOptionsWithInterface) Attach(c *cobra.Command) {}

type simpleTestStruct struct {
	Field string `flag:"test-field" flagdescr:"test field"`
}

func (o simpleTestStruct) Attach(c *cobra.Command) {}

type addressabilityTestOptions struct {
	StringField string `flag:"string-field" flagdescr:"string field"`
	IntField    int    `flag:"int-field" flagdescr:"int field"`
}

func (o addressabilityTestOptions) Attach(c *cobra.Command) {}

type deepNested struct {
	Value string `flag:"deep-value" flagdescr:"deep nested value"`
}

type middleNested struct {
	Deep deepNested
}

type topLevelNested struct {
	Middle middleNested
	Direct string `flag:"direct" flagdescr:"direct field"`
}

func (o topLevelNested) Attach(c *cobra.Command) {}

type canAddrTestOptions struct {
	Field string `flag:"field" flagdescr:"test field"`
}

func (o canAddrTestOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestDefine_NonAddressableFields() {
	suite.T().Run("interface_with_embedded_struct", func(t *testing.T) {
		// Create options with interface containing a struct
		opts := &testOptionsWithInterface{
			NormalField:    "test",
			InterfaceField: embeddedStruct{Value: "interface-value"},
		}

		cmd := &cobra.Command{Use: "test"}

		// This should not panic even if interface fields cause addressability issues
		assert.NotPanics(t, func() {
			autoflags.Define(cmd, opts)
		})

		// Verify that the normal field was processed
		normalFlag := cmd.Flags().Lookup("normal")
		assert.NotNil(t, normalFlag, "normal field should be processed")
	})

	suite.T().Run("manually_created_non_addressable_value", func(t *testing.T) {
		// Create a non-addressable value by using reflect.ValueOf on a struct value
		structValue := reflect.ValueOf(simpleTestStruct{Field: "test"})

		// Verify this creates a non-addressable value
		assert.False(t, structValue.CanAddr(), "struct value should not be addressable")

		// The field from this struct should also be non-addressable
		if structValue.NumField() > 0 {
			fieldValue := structValue.Field(0)
			assert.False(t, fieldValue.CanAddr(), "field from non-addressable struct should not be addressable")
		}
	})

	suite.T().Run("value_vs_pointer_addressability", func(t *testing.T) {
		// Test with struct value (potentially non-addressable)
		structValue := addressabilityTestOptions{StringField: "test", IntField: 42}
		cmd1 := &cobra.Command{Use: "test1"}

		assert.NotPanics(t, func() {
			autoflags.Define(cmd1, structValue)
		})

		// Test with struct pointer (should be addressable)
		structPtr := &addressabilityTestOptions{StringField: "test", IntField: 42}
		cmd2 := &cobra.Command{Use: "test2"}

		assert.NotPanics(t, func() {
			autoflags.Define(cmd2, structPtr)
		})

		// Both should create the same flags
		flag1 := cmd1.Flags().Lookup("string-field")
		flag2 := cmd2.Flags().Lookup("string-field")

		assert.NotNil(t, flag1, "struct value should create flags")
		assert.NotNil(t, flag2, "struct pointer should create flags")
		assert.Equal(t, flag1.Usage, flag2.Usage, "both should create equivalent flags")
	})

	suite.T().Run("complex_nested_non_addressable", func(t *testing.T) {
		// Create with struct value (not pointer)
		opts := topLevelNested{
			Middle: middleNested{
				Deep: deepNested{Value: "nested"},
			},
			Direct: "direct-value",
		}

		cmd := &cobra.Command{Use: "test"}

		// Should handle complex nesting without panicking
		assert.NotPanics(t, func() {
			autoflags.Define(cmd, opts)
		})

		// Should process the direct field
		directFlag := cmd.Flags().Lookup("direct")
		assert.NotNil(t, directFlag, "direct field should be processed")

		// Should process nested fields
		deepFlag := cmd.Flags().Lookup("deep-value")
		assert.NotNil(t, deepFlag, "deep nested field should be processed")
	})
}

func (suite *FlagsBaseSuite) TestDefine_CanAddrValidation() {
	suite.T().Run("ensure_canaddr_prevents_panic", func(t *testing.T) {
		// Use reflection to create a scenario that would panic without CanAddr() check
		structValue := reflect.ValueOf(canAddrTestOptions{Field: "test"})

		// Verify the field is not addressable
		if structValue.NumField() > 0 {
			field := structValue.Field(0)
			assert.False(t, field.CanAddr(), "field should not be addressable")

			// This would panic if we called UnsafeAddr() without checking CanAddr()
			assert.Panics(t, func() {
				_ = field.UnsafeAddr() // This should panic
			}, "UnsafeAddr() should panic on non-addressable field")
		}

		// But the Define() function should handle this gracefully
		cmd := &cobra.Command{Use: "test"}
		assert.NotPanics(t, func() {
			autoflags.Define(cmd, canAddrTestOptions{Field: "test"})
		})
	})
}

type flagCustomTestOptions struct {
	ValidCustom   string `flagcustom:"true" flag:"valid-custom" flagdescr:"should use custom handler"`
	InvalidCustom string `flagcustom:"invalid" flag:"invalid-custom" flagdescr:"has invalid flagcustom value"`
	EmptyCustom   string `flagcustom:"" flag:"empty-custom" flagdescr:"has empty flagcustom value"`
	FalseCustom   string `flagcustom:"false" flag:"false-custom" flagdescr:"explicitly false custom"`
	NormalField   string `flag:"normal" flagdescr:"normal field without flagcustom"`
}

func (o *flagCustomTestOptions) DefineValidCustom(c *cobra.Command, typename, name, short, descr string) {
	c.Flags().String(name, "CUSTOM_DEFAULT", descr+" [CUSTOM]")
}

func (o *flagCustomTestOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagCustom_WithValidation_ShouldReturnError() {
	opts := &flagCustomTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid flagcustom value")
	assert.Contains(suite.T(), err.Error(), "flagcustom", "Error should mention flagcustom")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
	assert.Contains(suite.T(), err.Error(), "InvalidCustom", "Error should mention the field name")
}

func (suite *FlagsBaseSuite) TestFlagCustom_WithValidation_OptionsPattern() {
	opts := &flagCustomTestOptions{}
	cmd1 := &cobra.Command{Use: "test1"}
	cmd2 := &cobra.Command{Use: "test2"}

	// Without validation - should not return error (backward compatible)
	err1 := autoflags.Define(cmd1, opts)
	assert.NoError(suite.T(), err1, "Without validation should not return error")

	// With validation - should return error
	err2 := autoflags.Define(cmd2, opts, autoflags.WithValidation())
	assert.Error(suite.T(), err2, "With validation should return error")
}

type validFlagCustomOptions struct {
	TrueCustom  string `flagcustom:"true" flag:"true-custom" flagdescr:"should use custom"`
	FalseCustom string `flagcustom:"false" flag:"false-custom" flagdescr:"should not use custom"`
	EmptyCustom string `flagcustom:"" flag:"empty-custom" flagdescr:"should not use custom"`
	NoCustom    string `flag:"no-custom" flagdescr:"should not use custom"`
}

func (o *validFlagCustomOptions) DefineTrueCustom(c *cobra.Command, typename, name, short, descr string) {
	c.Flags().String(name, "CUSTOM_VALUE", descr+" [CUSTOM]")
}

func (o *validFlagCustomOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagCustom_WithValidation_ValidValues() {
	opts := &validFlagCustomOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.NoError(suite.T(), err, "Should not return error for valid flagcustom values")

	// Check that flags are created correctly
	trueFlag := cmd.Flags().Lookup("true-custom")
	falseFlag := cmd.Flags().Lookup("false-custom")
	emptyFlag := cmd.Flags().Lookup("empty-custom")
	noFlag := cmd.Flags().Lookup("no-custom")

	// Only the true custom should use custom handler
	assert.Equal(suite.T(), "CUSTOM_VALUE", trueFlag.DefValue, "flagcustom='true' should use custom handler")
	assert.NotEqual(suite.T(), "CUSTOM_VALUE", falseFlag.DefValue, "flagcustom='false' should not use custom handler")
	assert.NotEqual(suite.T(), "CUSTOM_VALUE", emptyFlag.DefValue, "flagcustom='' should not use custom handler")
	assert.NotEqual(suite.T(), "CUSTOM_VALUE", noFlag.DefValue, "no flagcustom should not use custom handler")
}

type flagCustomEdgeCasesOptions struct {
	CaseTrue   string `flagcustom:"True" flag:"case-true" flagdescr:"capital True"`
	CaseFalse  string `flagcustom:"FALSE" flag:"case-false" flagdescr:"capital FALSE"`
	NumberOne  string `flagcustom:"1" flag:"number-one" flagdescr:"number 1"`
	NumberZero string `flagcustom:"0" flag:"number-zero" flagdescr:"number 0"`
	WithSpaces string `flagcustom:" true " flag:"with-spaces" flagdescr:"spaces around true"`
}

func (o *flagCustomEdgeCasesOptions) DefineCaseTrue(c *cobra.Command, typename, name, short, descr string) {
	c.Flags().String(name, "CUSTOM_TRUE", descr)
}

func (o *flagCustomEdgeCasesOptions) DefineNumberOne(c *cobra.Command, typename, name, short, descr string) {
	c.Flags().String(name, "CUSTOM_ONE", descr)
}

func (o *flagCustomEdgeCasesOptions) Attach(c *cobra.Command) {}

type validEdgeCasesOptions struct {
	CaseTrue   string `flagcustom:"True" flag:"case-true" flagdescr:"capital True"`
	CaseFalse  string `flagcustom:"FALSE" flag:"case-false" flagdescr:"capital FALSE"`
	NumberOne  string `flagcustom:"1" flag:"number-one" flagdescr:"number 1"`
	NumberZero string `flagcustom:"0" flag:"number-zero" flagdescr:"number 0"`
}

func (o *validEdgeCasesOptions) DefineCaseTrue(c *cobra.Command, typename, name, short, descr string) {
	c.Flags().String(name, "CUSTOM_TRUE", descr)
}

func (o *validEdgeCasesOptions) DefineNumberOne(c *cobra.Command, typename, name, short, descr string) {
	c.Flags().String(name, "CUSTOM_ONE", descr)
}

func (o *validEdgeCasesOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagCustom_EdgeCases_ValidValues() {
	opts := &validEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// These should all pass validation
	err := autoflags.Define(cmd, opts, autoflags.WithValidation())
	assert.NoError(suite.T(), err, "Should not return error for valid edge case values")

	// Check behavior
	caseTrueFlag := cmd.Flags().Lookup("case-true")
	caseFalseFlag := cmd.Flags().Lookup("case-false")
	numberOneFlag := cmd.Flags().Lookup("number-one")
	numberZeroFlag := cmd.Flags().Lookup("number-zero")

	// strconv.ParseBool accepts these case variations and numbers
	assert.Equal(suite.T(), "CUSTOM_TRUE", caseTrueFlag.DefValue, "ParseBool should accept 'True'")
	assert.NotEqual(suite.T(), "CUSTOM_TRUE", caseFalseFlag.DefValue, "ParseBool should accept 'FALSE' as false")
	assert.Equal(suite.T(), "CUSTOM_ONE", numberOneFlag.DefValue, "ParseBool should accept '1' as true")
	assert.NotEqual(suite.T(), "CUSTOM_ONE", numberZeroFlag.DefValue, "ParseBool should accept '0' as false")
}

func (suite *FlagsBaseSuite) TestFlagCustom_EdgeCases_WithValidation_ShouldReturnError() {
	opts := &flagCustomEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Spaces should cause validation error
	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for flagcustom value with spaces")
	assert.Contains(suite.T(), err.Error(), "flagcustom", "Error should mention flagcustom")
	assert.Contains(suite.T(), err.Error(), " true ", "Error should mention the invalid value with spaces")
	assert.Contains(suite.T(), err.Error(), "WithSpaces", "Error should mention the field name")
}

type flagEnvTestOptions struct {
	ValidEnv   string `flagenv:"true" flag:"valid-env" flagdescr:"should have env binding"`
	InvalidEnv string `flagenv:"invalid" flag:"invalid-env" flagdescr:"has invalid flagenv value"`
	EmptyEnv   string `flagenv:"" flag:"empty-env" flagdescr:"has empty flagenv value"`
	FalseEnv   string `flagenv:"false" flag:"false-env" flagdescr:"explicitly false env"`
	NormalFlag string `flag:"normal" flagdescr:"normal field without flagenv"`
}

func (o *flagEnvTestOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagenv_WithValidation_ShouldReturnError() {
	opts := &flagEnvTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid flagenv value")
	assert.Contains(suite.T(), err.Error(), "flagenv", "Error should mention flagenv")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
	assert.Contains(suite.T(), err.Error(), "InvalidEnv", "Error should mention the field name")
}

func (suite *FlagsBaseSuite) TestFlagenv_WithValidation_OptionsPattern() {
	opts := &flagEnvTestOptions{}
	cmd1 := &cobra.Command{Use: "test1"}
	cmd2 := &cobra.Command{Use: "test2"}

	// Without validation - should not return error (backward compatible)
	err1 := autoflags.Define(cmd1, opts)
	assert.NoError(suite.T(), err1, "Without validation should not return error")

	// With validation - should return error
	err2 := autoflags.Define(cmd2, opts, autoflags.WithValidation())
	assert.Error(suite.T(), err2, "With validation should return error")
}

type validFlagEnvOptions struct {
	TrueEnv  string `flagenv:"true" flag:"true-env" flagdescr:"should have env"`
	FalseEnv string `flagenv:"false" flag:"false-env" flagdescr:"should not have env"`
	EmptyEnv string `flagenv:"" flag:"empty-env" flagdescr:"should not have env"`
	NoEnv    string `flag:"no-env" flagdescr:"should not have env"`
}

func (o *validFlagEnvOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagenv_WithValidation_ValidValues() {
	opts := &validFlagEnvOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.NoError(suite.T(), err, "Should not return error for valid flagenv values")

	// Check that flags are created correctly
	trueFlag := cmd.Flags().Lookup("true-env")
	falseFlag := cmd.Flags().Lookup("false-env")
	emptyFlag := cmd.Flags().Lookup("empty-env")
	noFlag := cmd.Flags().Lookup("no-env")

	// Check environment annotations
	trueEnvAnnotation := trueFlag.Annotations[autoflags.FlagEnvsAnnotation]
	falseEnvAnnotation := falseFlag.Annotations[autoflags.FlagEnvsAnnotation]
	emptyEnvAnnotation := emptyFlag.Annotations[autoflags.FlagEnvsAnnotation]
	noEnvAnnotation := noFlag.Annotations[autoflags.FlagEnvsAnnotation]

	// Only the true env should have environment binding
	assert.NotNil(suite.T(), trueEnvAnnotation, "flagenv='true' should have env annotation")
	assert.Nil(suite.T(), falseEnvAnnotation, "flagenv='false' should not have env annotation")
	assert.Nil(suite.T(), emptyEnvAnnotation, "flagenv='' should not have env annotation")
	assert.Nil(suite.T(), noEnvAnnotation, "no flagenv should not have env annotation")
}

type flagEnvEdgeCasesOptions struct {
	CaseTrue   string `flagenv:"True" flag:"case-true" flagdescr:"capital True"`
	CaseFalse  string `flagenv:"FALSE" flag:"case-false" flagdescr:"capital FALSE"`
	NumberOne  string `flagenv:"1" flag:"number-one" flagdescr:"number 1"`
	NumberZero string `flagenv:"0" flag:"number-zero" flagdescr:"number 0"`
	WithSpaces string `flagenv:" true " flag:"with-spaces" flagdescr:"spaces around true"`
}

func (o *flagEnvEdgeCasesOptions) Attach(c *cobra.Command) {}

type validEnvEdgeCasesOptions struct {
	CaseTrue   string `flagenv:"True" flag:"case-true" flagdescr:"capital True"`
	CaseFalse  string `flagenv:"FALSE" flag:"case-false" flagdescr:"capital FALSE"`
	NumberOne  string `flagenv:"1" flag:"number-one" flagdescr:"number 1"`
	NumberZero string `flagenv:"0" flag:"number-zero" flagdescr:"number 0"`
}

func (o *validEnvEdgeCasesOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagenv_EdgeCases_ValidValues() {
	opts := &validEnvEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// These should all pass validation
	err := autoflags.Define(cmd, opts, autoflags.WithValidation())
	assert.NoError(suite.T(), err, "Should not return error for valid edge case values")

	// Check behavior
	caseTrueFlag := cmd.Flags().Lookup("case-true")
	caseFalseFlag := cmd.Flags().Lookup("case-false")
	numberOneFlag := cmd.Flags().Lookup("number-one")
	numberZeroFlag := cmd.Flags().Lookup("number-zero")

	// strconv.ParseBool accepts these case variations and numbers
	caseTrueAnnotation := caseTrueFlag.Annotations[autoflags.FlagEnvsAnnotation]
	caseFalseAnnotation := caseFalseFlag.Annotations[autoflags.FlagEnvsAnnotation]
	numberOneAnnotation := numberOneFlag.Annotations[autoflags.FlagEnvsAnnotation]
	numberZeroAnnotation := numberZeroFlag.Annotations[autoflags.FlagEnvsAnnotation]

	assert.NotNil(suite.T(), caseTrueAnnotation, "ParseBool should accept 'True' as true")
	assert.Nil(suite.T(), caseFalseAnnotation, "ParseBool should accept 'FALSE' as false")
	assert.NotNil(suite.T(), numberOneAnnotation, "ParseBool should accept '1' as true")
	assert.Nil(suite.T(), numberZeroAnnotation, "ParseBool should accept '0' as false")
}

func (suite *FlagsBaseSuite) TestFlagenv_EdgeCases_WithValidation_ShouldReturnError() {
	opts := &flagEnvEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Spaces should cause validation error
	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for flagenv value with spaces")
	assert.Contains(suite.T(), err.Error(), "flagenv", "Error should mention flagenv")
	assert.Contains(suite.T(), err.Error(), " true ", "Error should mention the invalid value with spaces")
	assert.Contains(suite.T(), err.Error(), "WithSpaces", "Error should mention the field name")
}

type nestedFlagEnvOptions struct {
	TopLevel     string          `flag:"top-level" flagenv:"true" flagdescr:"top level env flag"`
	NestedStruct nestedEnvStruct `flaggroup:"Nested"`
}

type nestedEnvStruct struct {
	ValidNestedEnv   string `flag:"nested-valid" flagenv:"true" flagdescr:"nested valid env"`
	InvalidNestedEnv string `flag:"nested-invalid" flagenv:"invalid" flagdescr:"nested invalid env"`
}

func (o *nestedFlagEnvOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagenv_NestedStructs_WithValidation() {
	opts := &nestedFlagEnvOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid nested flagenv value")
	assert.Contains(suite.T(), err.Error(), "flagenv", "Error should mention flagenv")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
	assert.Contains(suite.T(), err.Error(), "NestedStruct.InvalidNestedEnv", "Error should mention the nested field name")
}

type multipleInvalidEnvOptions struct {
	InvalidEnv1 string `flagenv:"yes" flag:"invalid1" flagdescr:"first invalid"`
	InvalidEnv2 string `flagenv:"no" flag:"invalid2" flagdescr:"second invalid"`
	ValidEnv    string `flagenv:"true" flag:"valid" flagdescr:"valid env"`
}

func (o *multipleInvalidEnvOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagenv_MultipleInvalid_ReturnsFirstError() {
	opts := &multipleInvalidEnvOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid flagenv values")
	assert.Contains(suite.T(), err.Error(), "flagenv", "Error should mention flagenv")
	// Should return the first error encountered (InvalidEnv1)
	assert.Contains(suite.T(), err.Error(), "InvalidEnv1", "Error should mention the first invalid field")
}

type flagEnvCombinedOptions struct {
	EnvWithCustom   string `flagenv:"true" flagcustom:"true" flag:"env-custom" flagdescr:"env with custom"`
	EnvWithRequired string `flagenv:"true" flagrequired:"true" flag:"env-required" flagdescr:"env with required"`
	EnvWithGroup    string `flagenv:"true" flaggroup:"TestGroup" flag:"env-group" flagdescr:"env with group"`
	InvalidEnvValid string `flagenv:"invalid" flagcustom:"true" flag:"invalid-env-valid" flagdescr:"invalid env with valid custom"`
}

func (o *flagEnvCombinedOptions) DefineEnvWithCustom(c *cobra.Command, typename, name, short, descr string) {
	c.Flags().String(name, "CUSTOM_DEFAULT", descr+" [CUSTOM]")
}

func (o *flagEnvCombinedOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagenv_CombinedWithOtherTags() {
	opts := &flagEnvCombinedOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Should fail due to invalid flagenv, even though flagcustom is valid
	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid flagenv value")
	assert.Contains(suite.T(), err.Error(), "flagenv", "Error should mention flagenv")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
}

type bothInvalidOptions struct {
	InvalidBoth string `flagenv:"invalid" flagcustom:"invalid" flag:"invalid-both" flagdescr:"both invalid"`
}

func (o *bothInvalidOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagenv_BothInvalid_ReturnsFirstError() {
	opts := &bothInvalidOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid tag values")
	// Should return the first error (flagcustom is validated first in the current implementation)
	assert.Contains(suite.T(), err.Error(), "flagcustom", "Error should mention the first invalid tag")
}

type flagEnvValidationTimingOptions struct {
	ValidEnv   string `flagenv:"true" flag:"valid-env" flagdescr:"valid env binding"`
	InvalidEnv string `flagenv:"invalid" flag:"invalid-env" flagdescr:"invalid env value"`
	NoEnv      string `flag:"no-env" flagdescr:"no env binding"`
}

func (o *flagEnvValidationTimingOptions) Attach(c *cobra.Command) {}

type backwardCompatOptions struct {
	ShouldWork  string `flagenv:"true" flag:"should-work" flagdescr:"should have env"`
	ShouldFail  string `flagenv:"invalid" flag:"should-fail" flagdescr:"invalid but silent"`
	YesNo       string `flagenv:"yes" flag:"yes-no" flagdescr:"common mistake"`
	EmptyString string `flagenv:"" flag:"empty" flagdescr:"empty should be false"`
}

func (o *backwardCompatOptions) Attach(c *cobra.Command) {}

type errorMessageOptions struct {
	BadValue string `flagenv:"maybe" flag:"bad-value" flagdescr:"bad boolean value"`
}

func (o *errorMessageOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagenv_ValidationTiming_EarlyValidationPreventsLaterErrors() {
	opts := &flagEnvValidationTimingOptions{}
	cmd := &cobra.Command{Use: "test"}

	// With validation enabled, should fail at Define() time
	err := autoflags.Define(cmd, opts, autoflags.WithValidation())
	assert.Error(suite.T(), err, "Should fail during Define() with validation enabled")
	assert.Contains(suite.T(), err.Error(), "flagenv", "Error should be about flagenv validation")

	// Without validation enabled, should succeed at Define() time
	opts2 := &flagEnvValidationTimingOptions{}
	cmd2 := &cobra.Command{Use: "test2"}

	err2 := autoflags.Define(cmd2, opts2) // No WithValidation()
	assert.NoError(suite.T(), err2, "Should succeed during Define() without validation")

	// Verify that the invalid flagenv value is silently treated as false (backward compatibility)
	invalidFlag := cmd2.Flags().Lookup("invalid-env")
	assert.NotNil(suite.T(), invalidFlag, "Invalid env flag should still be created")

	invalidEnvAnnotation := invalidFlag.Annotations[autoflags.FlagEnvsAnnotation]
	assert.Nil(suite.T(), invalidEnvAnnotation, "Invalid flagenv should be treated as false (no env binding)")

	// Verify that valid flagenv still works
	validFlag := cmd2.Flags().Lookup("valid-env")
	assert.NotNil(suite.T(), validFlag, "Valid env flag should be created")

	validEnvAnnotation := validFlag.Annotations[autoflags.FlagEnvsAnnotation]
	assert.NotNil(suite.T(), validEnvAnnotation, "Valid flagenv should have env binding")
}

func (suite *FlagsBaseSuite) TestFlagenv_BackwardCompatibility_SilentFailureWithoutValidation() {
	opts := &backwardCompatOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Should not fail without validation
	err := autoflags.Define(cmd, opts)
	assert.NoError(suite.T(), err, "Should not fail without validation enabled")

	// Check the behavior matches expectations
	flags := cmd.Flags()

	shouldWorkFlag := flags.Lookup("should-work")
	shouldFailFlag := flags.Lookup("should-fail")
	yesNoFlag := flags.Lookup("yes-no")
	emptyFlag := flags.Lookup("empty")

	// Check environment annotations
	shouldWorkAnnotation := shouldWorkFlag.Annotations[autoflags.FlagEnvsAnnotation]
	shouldFailAnnotation := shouldFailFlag.Annotations[autoflags.FlagEnvsAnnotation]
	yesNoAnnotation := yesNoFlag.Annotations[autoflags.FlagEnvsAnnotation]
	emptyAnnotation := emptyFlag.Annotations[autoflags.FlagEnvsAnnotation]

	assert.NotNil(suite.T(), shouldWorkAnnotation, "flagenv='true' should work")
	assert.Nil(suite.T(), shouldFailAnnotation, "flagenv='invalid' should be treated as false")
	assert.Nil(suite.T(), yesNoAnnotation, "flagenv='yes' should be treated as false")
	assert.Nil(suite.T(), emptyAnnotation, "flagenv='' should be treated as false")
}

func (suite *FlagsBaseSuite) TestFlagenv_ErrorMessages_ContainExpectedContent() {
	opts := &errorMessageOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid flagenv")

	errorMsg := err.Error()

	// These are the expected components of a FieldError
	assert.Contains(suite.T(), errorMsg, "BadValue", "Error should contain field name")
	assert.Contains(suite.T(), errorMsg, "flagenv", "Error should contain tag name")
	assert.Contains(suite.T(), errorMsg, "maybe", "Error should contain tag value")
	assert.Contains(suite.T(), errorMsg, "invalid boolean value", "Error should contain message")
}

type flagIgnoreTestOptions struct {
	ValidIgnore   string `flagignore:"true" flag:"valid-ignore" flagdescr:"should be ignored"`
	InvalidIgnore string `flagignore:"invalid" flag:"invalid-ignore" flagdescr:"has invalid flagignore value"`
	EmptyIgnore   string `flagignore:"" flag:"empty-ignore" flagdescr:"has empty flagignore value"`
	FalseIgnore   string `flagignore:"false" flag:"false-ignore" flagdescr:"explicitly false ignore"`
	NormalFlag    string `flag:"normal" flagdescr:"normal field without flagignore"`
}

func (o *flagIgnoreTestOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagignore_WithValidation_ShouldReturnError() {
	opts := &flagIgnoreTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid flagignore value")
	assert.Contains(suite.T(), err.Error(), "flagignore", "Error should mention flagignore")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
	assert.Contains(suite.T(), err.Error(), "InvalidIgnore", "Error should mention the field name")
}

func (suite *FlagsBaseSuite) TestFlagignore_WithValidation_OptionsPattern() {
	opts := &flagIgnoreTestOptions{}
	cmd1 := &cobra.Command{Use: "test1"}
	cmd2 := &cobra.Command{Use: "test2"}

	// Without validation - should not return error (backward compatible)
	err1 := autoflags.Define(cmd1, opts)
	assert.NoError(suite.T(), err1, "Without validation should not return error")

	// With validation - should return error
	err2 := autoflags.Define(cmd2, opts, autoflags.WithValidation())
	assert.Error(suite.T(), err2, "With validation should return error")
}

type validFlagIgnoreOptions struct {
	TrueIgnore  string `flagignore:"true" flag:"true-ignore" flagdescr:"should be ignored"`
	FalseIgnore string `flagignore:"false" flag:"false-ignore" flagdescr:"should not be ignored"`
	EmptyIgnore string `flagignore:"" flag:"empty-ignore" flagdescr:"should not be ignored"`
	NoIgnore    string `flag:"no-ignore" flagdescr:"should not be ignored"`
}

func (o *validFlagIgnoreOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagignore_WithValidation_ValidValues() {
	opts := &validFlagIgnoreOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.NoError(suite.T(), err, "Should not return error for valid flagignore values")

	// Check that flags are created/ignored correctly
	trueIgnoreFlag := cmd.Flags().Lookup("true-ignore")
	falseIgnoreFlag := cmd.Flags().Lookup("false-ignore")
	emptyIgnoreFlag := cmd.Flags().Lookup("empty-ignore")
	noIgnoreFlag := cmd.Flags().Lookup("no-ignore")

	// Only the true ignore should be skipped
	assert.Nil(suite.T(), trueIgnoreFlag, "flagignore='true' should skip flag creation")
	assert.NotNil(suite.T(), falseIgnoreFlag, "flagignore='false' should create flag")
	assert.NotNil(suite.T(), emptyIgnoreFlag, "flagignore='' should create flag")
	assert.NotNil(suite.T(), noIgnoreFlag, "no flagignore should create flag")
}

type flagIgnoreEdgeCasesOptions struct {
	CaseTrue   string `flagignore:"True" flag:"case-true" flagdescr:"capital True"`
	CaseFalse  string `flagignore:"FALSE" flag:"case-false" flagdescr:"capital FALSE"`
	NumberOne  string `flagignore:"1" flag:"number-one" flagdescr:"number 1"`
	NumberZero string `flagignore:"0" flag:"number-zero" flagdescr:"number 0"`
	WithSpaces string `flagignore:" true " flag:"with-spaces" flagdescr:"spaces around true"`
}

func (o *flagIgnoreEdgeCasesOptions) Attach(c *cobra.Command) {}

type validIgnoreEdgeCasesOptions struct {
	CaseTrue   string `flagignore:"True" flag:"case-true" flagdescr:"capital True"`
	CaseFalse  string `flagignore:"FALSE" flag:"case-false" flagdescr:"capital FALSE"`
	NumberOne  string `flagignore:"1" flag:"number-one" flagdescr:"number 1"`
	NumberZero string `flagignore:"0" flag:"number-zero" flagdescr:"number 0"`
}

func (o *validIgnoreEdgeCasesOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagignore_EdgeCases_ValidValues() {
	opts := &validIgnoreEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// These should all pass validation
	err := autoflags.Define(cmd, opts, autoflags.WithValidation())
	assert.NoError(suite.T(), err, "Should not return error for valid edge case values")

	// Check behavior - strconv.ParseBool accepts these case variations and numbers
	caseTrueFlag := cmd.Flags().Lookup("case-true")
	caseFalseFlag := cmd.Flags().Lookup("case-false")
	numberOneFlag := cmd.Flags().Lookup("number-one")
	numberZeroFlag := cmd.Flags().Lookup("number-zero")

	assert.Nil(suite.T(), caseTrueFlag, "ParseBool should accept 'True' as true (ignore flag)")
	assert.NotNil(suite.T(), caseFalseFlag, "ParseBool should accept 'FALSE' as false (create flag)")
	assert.Nil(suite.T(), numberOneFlag, "ParseBool should accept '1' as true (ignore flag)")
	assert.NotNil(suite.T(), numberZeroFlag, "ParseBool should accept '0' as false (create flag)")
}

func (suite *FlagsBaseSuite) TestFlagignore_EdgeCases_WithValidation_ShouldReturnError() {
	opts := &flagIgnoreEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Spaces should cause validation error
	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for flagignore value with spaces")
	assert.Contains(suite.T(), err.Error(), "flagignore", "Error should mention flagignore")
	assert.Contains(suite.T(), err.Error(), " true ", "Error should mention the invalid value with spaces")
	assert.Contains(suite.T(), err.Error(), "WithSpaces", "Error should mention the field name")
}

type nestedFlagIgnoreOptions struct {
	TopLevel     string             `flag:"top-level" flagignore:"false" flagdescr:"top level flag"`
	NestedStruct nestedIgnoreStruct `flaggroup:"Nested"`
}

type nestedIgnoreStruct struct {
	ValidNestedIgnore   string `flag:"nested-valid" flagignore:"true" flagdescr:"nested ignored flag"`
	InvalidNestedIgnore string `flag:"nested-invalid" flagignore:"invalid" flagdescr:"nested invalid ignore"`
}

func (o *nestedFlagIgnoreOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagignore_NestedStructs_WithValidation() {
	opts := &nestedFlagIgnoreOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid nested flagignore value")
	assert.Contains(suite.T(), err.Error(), "flagignore", "Error should mention flagignore")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
	assert.Contains(suite.T(), err.Error(), "NestedStruct.InvalidNestedIgnore", "Error should mention the nested field name")
}

type multipleInvalidIgnoreOptions struct {
	InvalidIgnore1 string `flagignore:"yes" flag:"invalid1" flagdescr:"first invalid"`
	InvalidIgnore2 string `flagignore:"no" flag:"invalid2" flagdescr:"second invalid"`
	ValidIgnore    string `flagignore:"true" flag:"valid" flagdescr:"valid ignore"`
}

func (o *multipleInvalidIgnoreOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagignore_MultipleInvalid_ReturnsFirstError() {
	opts := &multipleInvalidIgnoreOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid flagignore values")
	assert.Contains(suite.T(), err.Error(), "flagignore", "Error should mention flagignore")
	// Should return the first error encountered (InvalidIgnore1)
	assert.Contains(suite.T(), err.Error(), "InvalidIgnore1", "Error should mention the first invalid field")
}

type flagIgnoreCombinedOptions struct {
	IgnoreWithCustom   string `flagignore:"true" flagcustom:"true" flag:"ignore-custom" flagdescr:"ignore with custom"`
	IgnoreWithRequired string `flagignore:"false" flagrequired:"true" flag:"ignore-required" flagdescr:"ignore with required"`
	IgnoreWithGroup    string `flagignore:"false" flaggroup:"TestGroup" flag:"ignore-group" flagdescr:"ignore with group"`
	InvalidIgnoreValid string `flagignore:"invalid" flagenv:"true" flag:"invalid-ignore-valid" flagdescr:"invalid ignore with valid env"`
}

func (o *flagIgnoreCombinedOptions) DefineIgnoreWithCustom(c *cobra.Command, typename, name, short, descr string) {
	c.Flags().String(name, "CUSTOM_DEFAULT", descr+" [CUSTOM]")
}

func (o *flagIgnoreCombinedOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagignore_CombinedWithOtherTags() {
	opts := &flagIgnoreCombinedOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Should fail due to invalid flagignore, even though other tags are valid
	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid flagignore value")
	assert.Contains(suite.T(), err.Error(), "flagignore", "Error should mention flagignore")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
}

type allThreeInvalidOptions struct {
	InvalidAll string `flagignore:"invalid" flagenv:"invalid" flagcustom:"invalid" flag:"invalid-all" flagdescr:"all three invalid"`
}

func (o *allThreeInvalidOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagignore_AllThreeInvalid_ReturnsFirstError() {
	opts := &allThreeInvalidOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid tag values")
	// Should return the first error (flagcustom is validated first in the current implementation)
	assert.Contains(suite.T(), err.Error(), "flagcustom", "Error should mention the first invalid tag")
}

// Additional struct definitions for timing and compatibility tests

type flagIgnoreValidationTimingOptions struct {
	ValidIgnore   string `flagignore:"true" flag:"valid-ignore" flagdescr:"valid ignore"`
	InvalidIgnore string `flagignore:"invalid" flag:"invalid-ignore" flagdescr:"invalid ignore value"`
	NoIgnore      string `flag:"no-ignore" flagdescr:"no ignore tag"`
}

func (o *flagIgnoreValidationTimingOptions) Attach(c *cobra.Command) {}

type backwardIgnoreCompatOptions struct {
	ShouldWork  string `flagignore:"true" flag:"should-work" flagdescr:"should be ignored"`
	ShouldFail  string `flagignore:"invalid" flag:"should-fail" flagdescr:"invalid but silent"`
	YesNo       string `flagignore:"yes" flag:"yes-no" flagdescr:"common mistake"`
	EmptyString string `flagignore:"" flag:"empty" flagdescr:"empty should be false"`
}

func (o *backwardIgnoreCompatOptions) Attach(c *cobra.Command) {}

type errorIgnoreMessageOptions struct {
	BadValue string `flagignore:"maybe" flag:"bad-value" flagdescr:"bad boolean value"`
}

func (o *errorIgnoreMessageOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagignore_ValidationTiming_EarlyValidationPreventsLaterErrors() {
	opts := &flagIgnoreValidationTimingOptions{}
	cmd := &cobra.Command{Use: "test"}

	// With validation enabled, should fail at Define() time
	err := autoflags.Define(cmd, opts, autoflags.WithValidation())
	assert.Error(suite.T(), err, "Should fail during Define() with validation enabled")
	assert.Contains(suite.T(), err.Error(), "flagignore", "Error should be about flagignore validation")

	// Without validation enabled, should succeed at Define() time
	opts2 := &flagIgnoreValidationTimingOptions{}
	cmd2 := &cobra.Command{Use: "test2"}

	err2 := autoflags.Define(cmd2, opts2) // No WithValidation()
	assert.NoError(suite.T(), err2, "Should succeed during Define() without validation")

	// Verify that the invalid flagignore value is silently treated as false (backward compatibility)
	invalidFlag := cmd2.Flags().Lookup("invalid-ignore")
	assert.NotNil(suite.T(), invalidFlag, "Invalid ignore flag should still be created (treated as false)")

	// Verify that valid flagignore still works
	validFlag := cmd2.Flags().Lookup("valid-ignore")
	assert.Nil(suite.T(), validFlag, "Valid flagignore='true' should skip flag creation")

	// Verify normal flags still work
	normalFlag := cmd2.Flags().Lookup("no-ignore")
	assert.NotNil(suite.T(), normalFlag, "Normal flag should be created")
}

func (suite *FlagsBaseSuite) TestFlagignore_BackwardCompatibility_SilentFailureWithoutValidation() {
	opts := &backwardIgnoreCompatOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Should not fail without validation
	err := autoflags.Define(cmd, opts)
	assert.NoError(suite.T(), err, "Should not fail without validation enabled")

	// Check the behavior matches expectations
	flags := cmd.Flags()

	shouldWorkFlag := flags.Lookup("should-work") // flagignore="true"
	shouldFailFlag := flags.Lookup("should-fail") // flagignore="invalid"
	yesNoFlag := flags.Lookup("yes-no")           // flagignore="yes"
	emptyFlag := flags.Lookup("empty")            // flagignore=""

	// Check flag creation behavior
	assert.Nil(suite.T(), shouldWorkFlag, "flagignore='true' should ignore flag")
	assert.NotNil(suite.T(), shouldFailFlag, "flagignore='invalid' should be treated as false (create flag)")
	assert.NotNil(suite.T(), yesNoFlag, "flagignore='yes' should be treated as false (create flag)")
	assert.NotNil(suite.T(), emptyFlag, "flagignore='' should be treated as false (create flag)")
}

func (suite *FlagsBaseSuite) TestFlagignore_ErrorMessages_ContainExpectedContent() {
	opts := &errorIgnoreMessageOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid flagignore")

	errorMsg := err.Error()

	// These are the expected components of a FieldError
	assert.Contains(suite.T(), errorMsg, "BadValue", "Error should contain field name")
	assert.Contains(suite.T(), errorMsg, "flagignore", "Error should contain tag name")
	assert.Contains(suite.T(), errorMsg, "maybe", "Error should contain tag value")
	assert.Contains(suite.T(), errorMsg, "invalid boolean value", "Error should contain message")
}

type flagRequiredTestOptions struct {
	ValidRequired   string `flagrequired:"true" flag:"valid-required" flagdescr:"should be required"`
	InvalidRequired string `flagrequired:"invalid" flag:"invalid-required" flagdescr:"has invalid flagrequired value"`
	EmptyRequired   string `flagrequired:"" flag:"empty-required" flagdescr:"has empty flagrequired value"`
	FalseRequired   string `flagrequired:"false" flag:"false-required" flagdescr:"explicitly false required"`
	NormalFlag      string `flag:"normal" flagdescr:"normal field without flagrequired"`
}

func (o *flagRequiredTestOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagrequired_WithValidation_ShouldReturnError() {
	opts := &flagRequiredTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid flagrequired value")
	assert.Contains(suite.T(), err.Error(), "flagrequired", "Error should mention flagrequired")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
	assert.Contains(suite.T(), err.Error(), "InvalidRequired", "Error should mention the field name")
}

func (suite *FlagsBaseSuite) TestFlagrequired_WithValidation_OptionsPattern() {
	opts := &flagRequiredTestOptions{}
	cmd1 := &cobra.Command{Use: "test1"}
	cmd2 := &cobra.Command{Use: "test2"}

	// Without validation - should not return error (backward compatible)
	err1 := autoflags.Define(cmd1, opts)
	assert.NoError(suite.T(), err1, "Without validation should not return error")

	// With validation - should return error
	err2 := autoflags.Define(cmd2, opts, autoflags.WithValidation())
	assert.Error(suite.T(), err2, "With validation should return error")
}

type validFlagRequiredOptions struct {
	TrueRequired  string `flagrequired:"true" flag:"true-required" flagdescr:"should be required"`
	FalseRequired string `flagrequired:"false" flag:"false-required" flagdescr:"should not be required"`
	EmptyRequired string `flagrequired:"" flag:"empty-required" flagdescr:"should not be required"`
	NoRequired    string `flag:"no-required" flagdescr:"should not be required"`
}

func (o *validFlagRequiredOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagrequired_WithValidation_ValidValues() {
	opts := &validFlagRequiredOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.NoError(suite.T(), err, "Should not return error for valid flagrequired values")

	// Check that flags are marked as required/optional correctly
	trueRequiredFlag := cmd.Flags().Lookup("true-required")
	falseRequiredFlag := cmd.Flags().Lookup("false-required")
	emptyRequiredFlag := cmd.Flags().Lookup("empty-required")
	noRequiredFlag := cmd.Flags().Lookup("no-required")

	// Check required annotations (cobra uses BashCompOneRequiredFlag annotation)
	trueRequiredAnnotation := trueRequiredFlag.Annotations[cobra.BashCompOneRequiredFlag]
	falseRequiredAnnotation := falseRequiredFlag.Annotations[cobra.BashCompOneRequiredFlag]
	emptyRequiredAnnotation := emptyRequiredFlag.Annotations[cobra.BashCompOneRequiredFlag]
	noRequiredAnnotation := noRequiredFlag.Annotations[cobra.BashCompOneRequiredFlag]

	// Only the true required should be marked as required
	assert.NotNil(suite.T(), trueRequiredAnnotation, "flagrequired='true' should mark flag as required")
	assert.Equal(suite.T(), []string{"true"}, trueRequiredAnnotation, "required annotation should be 'true'")
	assert.Nil(suite.T(), falseRequiredAnnotation, "flagrequired='false' should not mark flag as required")
	assert.Nil(suite.T(), emptyRequiredAnnotation, "flagrequired='' should not mark flag as required")
	assert.Nil(suite.T(), noRequiredAnnotation, "no flagrequired should not mark flag as required")
}

type flagRequiredEdgeCasesOptions struct {
	CaseTrue   string `flagrequired:"True" flag:"case-true" flagdescr:"capital True"`
	CaseFalse  string `flagrequired:"FALSE" flag:"case-false" flagdescr:"capital FALSE"`
	NumberOne  string `flagrequired:"1" flag:"number-one" flagdescr:"number 1"`
	NumberZero string `flagrequired:"0" flag:"number-zero" flagdescr:"number 0"`
	WithSpaces string `flagrequired:" true " flag:"with-spaces" flagdescr:"spaces around true"`
}

func (o *flagRequiredEdgeCasesOptions) Attach(c *cobra.Command) {}

type validRequiredEdgeCasesOptions struct {
	CaseTrue   string `flagrequired:"True" flag:"case-true" flagdescr:"capital True"`
	CaseFalse  string `flagrequired:"FALSE" flag:"case-false" flagdescr:"capital FALSE"`
	NumberOne  string `flagrequired:"1" flag:"number-one" flagdescr:"number 1"`
	NumberZero string `flagrequired:"0" flag:"number-zero" flagdescr:"number 0"`
}

func (o *validRequiredEdgeCasesOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagrequired_EdgeCases_ValidValues() {
	opts := &validRequiredEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// These should all pass validation
	err := autoflags.Define(cmd, opts, autoflags.WithValidation())
	assert.NoError(suite.T(), err, "Should not return error for valid edge case values")

	// Check behavior - strconv.ParseBool accepts these case variations and numbers
	caseTrueFlag := cmd.Flags().Lookup("case-true")
	caseFalseFlag := cmd.Flags().Lookup("case-false")
	numberOneFlag := cmd.Flags().Lookup("number-one")
	numberZeroFlag := cmd.Flags().Lookup("number-zero")

	// Check required annotations
	caseTrueAnnotation := caseTrueFlag.Annotations[cobra.BashCompOneRequiredFlag]
	caseFalseAnnotation := caseFalseFlag.Annotations[cobra.BashCompOneRequiredFlag]
	numberOneAnnotation := numberOneFlag.Annotations[cobra.BashCompOneRequiredFlag]
	numberZeroAnnotation := numberZeroFlag.Annotations[cobra.BashCompOneRequiredFlag]

	assert.NotNil(suite.T(), caseTrueAnnotation, "ParseBool should accept 'True' as true (mark as required)")
	assert.Nil(suite.T(), caseFalseAnnotation, "ParseBool should accept 'FALSE' as false (not required)")
	assert.NotNil(suite.T(), numberOneAnnotation, "ParseBool should accept '1' as true (mark as required)")
	assert.Nil(suite.T(), numberZeroAnnotation, "ParseBool should accept '0' as false (not required)")
}

func (suite *FlagsBaseSuite) TestFlagrequired_EdgeCases_WithValidation_ShouldReturnError() {
	opts := &flagRequiredEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Spaces should cause validation error
	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for flagrequired value with spaces")
	assert.Contains(suite.T(), err.Error(), "flagrequired", "Error should mention flagrequired")
	assert.Contains(suite.T(), err.Error(), " true ", "Error should mention the invalid value with spaces")
	assert.Contains(suite.T(), err.Error(), "WithSpaces", "Error should mention the field name")
}

type nestedFlagRequiredOptions struct {
	TopLevel     string                           `flag:"top-level" flagrequired:"false" flagdescr:"top level flag"`
	NestedStruct nestedValidInvalidRequiredStruct `flaggroup:"Nested"`
}

type nestedValidInvalidRequiredStruct struct {
	ValidNestedRequired   string `flag:"nested-valid" flagrequired:"true" flagdescr:"nested required flag"`
	InvalidNestedRequired string `flag:"nested-invalid" flagrequired:"invalid" flagdescr:"nested invalid required"`
}

func (o *nestedFlagRequiredOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagrequired_NestedStructs_WithValidation() {
	opts := &nestedFlagRequiredOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid nested flagrequired value")
	assert.Contains(suite.T(), err.Error(), "flagrequired", "Error should mention flagrequired")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
	assert.Contains(suite.T(), err.Error(), "NestedStruct.InvalidNestedRequired", "Error should mention the nested field name")
}

type multipleInvalidRequiredOptions struct {
	InvalidRequired1 string `flagrequired:"yes" flag:"invalid1" flagdescr:"first invalid"`
	InvalidRequired2 string `flagrequired:"no" flag:"invalid2" flagdescr:"second invalid"`
	ValidRequired    string `flagrequired:"true" flag:"valid" flagdescr:"valid required"`
}

func (o *multipleInvalidRequiredOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagrequired_MultipleInvalid_ReturnsFirstError() {
	opts := &multipleInvalidRequiredOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid flagrequired values")
	assert.Contains(suite.T(), err.Error(), "flagrequired", "Error should mention flagrequired")
	// Should return the first error encountered (InvalidRequired1)
	assert.Contains(suite.T(), err.Error(), "InvalidRequired1", "Error should mention the first invalid field")
}

type flagRequiredCombinedOptions struct {
	RequiredWithCustom   string `flagrequired:"true" flagcustom:"true" flag:"required-custom" flagdescr:"required with custom"`
	RequiredWithEnv      string `flagrequired:"true" flagenv:"true" flag:"required-env" flagdescr:"required with env"`
	RequiredWithGroup    string `flagrequired:"false" flaggroup:"TestGroup" flag:"required-group" flagdescr:"required with group"`
	InvalidRequiredValid string `flagrequired:"invalid" flagignore:"false" flag:"invalid-required-valid" flagdescr:"invalid required with valid ignore"`
}

func (o *flagRequiredCombinedOptions) DefineRequiredWithCustom(c *cobra.Command, typename, name, short, descr string) {
	c.Flags().String(name, "CUSTOM_DEFAULT", descr+" [CUSTOM]")
}

func (o *flagRequiredCombinedOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagrequired_CombinedWithOtherTagstWihValidation() {
	opts := &flagRequiredCombinedOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Should fail due to invalid flagrequired, even though other tags are valid
	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid flagrequired value")
	assert.Contains(suite.T(), err.Error(), "flagrequired", "Error should mention flagrequired")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
}

type allFourInvalidOptions struct {
	InvalidAll string `flagrequired:"invalid" flagignore:"invalid" flagenv:"invalid" flagcustom:"invalid" flag:"invalid-all" flagdescr:"all four invalid"`
}

func (o *allFourInvalidOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagrequired_AllFourInvalid_ReturnsFirstError() {
	opts := &allFourInvalidOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid tag values")
	// Should return the first error (flagcustom is validated first in the current implementation)
	assert.Contains(suite.T(), err.Error(), "flagcustom", "Error should mention the first invalid tag")
}

// Additional struct definitions for timing and compatibility tests

type flagRequiredValidationTimingOptions struct {
	ValidRequired   string `flagrequired:"true" flag:"valid-required" flagdescr:"valid required"`
	InvalidRequired string `flagrequired:"invalid" flag:"invalid-required" flagdescr:"invalid required value"`
	NoRequired      string `flag:"no-required" flagdescr:"no required tag"`
}

func (o *flagRequiredValidationTimingOptions) Attach(c *cobra.Command) {}

type backwardRequiredCompatOptions struct {
	ShouldWork  string `flagrequired:"true" flag:"should-work" flagdescr:"should be required"`
	ShouldFail  string `flagrequired:"invalid" flag:"should-fail" flagdescr:"invalid but silent"`
	YesNo       string `flagrequired:"yes" flag:"yes-no" flagdescr:"common mistake"`
	EmptyString string `flagrequired:"" flag:"empty" flagdescr:"empty should be false"`
}

func (o *backwardRequiredCompatOptions) Attach(c *cobra.Command) {}

type errorRequiredMessageOptions struct {
	BadValue string `flagrequired:"maybe" flag:"bad-value" flagdescr:"bad boolean value"`
}

func (o *errorRequiredMessageOptions) Attach(c *cobra.Command) {}

func (suite *FlagsBaseSuite) TestFlagrequired_ValidationTiming_EarlyValidationPreventsLaterErrors() {
	opts := &flagRequiredValidationTimingOptions{}
	cmd := &cobra.Command{Use: "test"}

	// With validation enabled, should fail at Define() time
	err := autoflags.Define(cmd, opts, autoflags.WithValidation())
	assert.Error(suite.T(), err, "Should fail during Define() with validation enabled")
	assert.Contains(suite.T(), err.Error(), "flagrequired", "Error should be about flagrequired validation")

	// Without validation enabled, should succeed at Define() time
	opts2 := &flagRequiredValidationTimingOptions{}
	cmd2 := &cobra.Command{Use: "test2"}

	err2 := autoflags.Define(cmd2, opts2) // No WithValidation()
	assert.NoError(suite.T(), err2, "Should succeed during Define() without validation")

	// Verify that the invalid flagrequired value is silently treated as false (backward compatibility)
	invalidFlag := cmd2.Flags().Lookup("invalid-required")
	assert.NotNil(suite.T(), invalidFlag, "Invalid required flag should still be created")

	invalidRequiredAnnotation := invalidFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.Nil(suite.T(), invalidRequiredAnnotation, "Invalid flagrequired should be treated as false (not required)")

	// Verify that valid flagrequired still works
	validFlag := cmd2.Flags().Lookup("valid-required")
	assert.NotNil(suite.T(), validFlag, "Valid required flag should be created")

	validRequiredAnnotation := validFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.NotNil(suite.T(), validRequiredAnnotation, "Valid flagrequired should mark flag as required")

	// Verify normal flags still work
	normalFlag := cmd2.Flags().Lookup("no-required")
	assert.NotNil(suite.T(), normalFlag, "Normal flag should be created")
}

func (suite *FlagsBaseSuite) TestFlagrequired_BackwardCompatibility_SilentFailureWithoutValidation() {
	opts := &backwardRequiredCompatOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Should not fail without validation
	err := autoflags.Define(cmd, opts)
	assert.NoError(suite.T(), err, "Should not fail without validation enabled")

	// Check the behavior matches expectations
	flags := cmd.Flags()

	shouldWorkFlag := flags.Lookup("should-work") // flagrequired="true"
	shouldFailFlag := flags.Lookup("should-fail") // flagrequired="invalid"
	yesNoFlag := flags.Lookup("yes-no")           // flagrequired="yes"
	emptyFlag := flags.Lookup("empty")            // flagrequired=""

	// Check required annotations
	shouldWorkAnnotation := shouldWorkFlag.Annotations[cobra.BashCompOneRequiredFlag]
	shouldFailAnnotation := shouldFailFlag.Annotations[cobra.BashCompOneRequiredFlag]
	yesNoAnnotation := yesNoFlag.Annotations[cobra.BashCompOneRequiredFlag]
	emptyAnnotation := emptyFlag.Annotations[cobra.BashCompOneRequiredFlag]

	assert.NotNil(suite.T(), shouldWorkAnnotation, "flagrequired='true' should mark flag as required")
	assert.Nil(suite.T(), shouldFailAnnotation, "flagrequired='invalid' should be treated as false (not required)")
	assert.Nil(suite.T(), yesNoAnnotation, "flagrequired='yes' should be treated as false (not required)")
	assert.Nil(suite.T(), emptyAnnotation, "flagrequired='' should be treated as false (not required)")
}

func (suite *FlagsBaseSuite) TestFlagrequired_ErrorMessages_ContainExpectedContent() {
	opts := &errorRequiredMessageOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := autoflags.Define(cmd, opts, autoflags.WithValidation())

	assert.Error(suite.T(), err, "Should return error for invalid flagrequired")

	errorMsg := err.Error()

	// These are the expected components of a FieldError
	assert.Contains(suite.T(), errorMsg, "BadValue", "Error should contain field name")
	assert.Contains(suite.T(), errorMsg, "flagrequired", "Error should contain tag name")
	assert.Contains(suite.T(), errorMsg, "maybe", "Error should contain tag value")
	assert.Contains(suite.T(), errorMsg, "invalid boolean value", "Error should contain message")
}
