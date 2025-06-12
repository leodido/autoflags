package autoflags

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	autoflagserrors "github.com/leodido/autoflags/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

type configFlags struct {
	LogLevel string `default:"info" flag:"log-level" flagdescr:"set the logging level" flaggroup:"Config"`
	Timeout  int    `flagdescr:"set the timeout, in seconds" flagset:"Config"`
	Endpoint string `flagdescr:"the listen.dev endpoint emitting the verdicts" flaggroup:"Config" flagrequired:"true"`
}

type deepFlags struct {
	Deep time.Duration `default:"deepdown" flagdescr:"deep flag" flag:"deep" flagshort:"d" flaggroup:"Deep"`
}

type jsonFlags struct {
	JSON bool      `flagdescr:"output the verdicts (if any) in JSON form"`
	JQ   string    `flagshort:"q" flagdescr:"filter the output using a jq expression"`
	Deep deepFlags `flagrequired:"true"`
}

type testOptions struct {
	configFlags `flaggroup:"Configuration"`
	Nest        jsonFlags
}

func (o testOptions) Attach(c *cobra.Command)             {}
func (o testOptions) Transform(ctx context.Context) error { return nil }
func (o testOptions) Validate() []error                   { return nil }

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

func (suite *autoflagsSuite) TestDefine_UintTypesSupport() {
	opts := &uintTestOptions{
		UintField:   500,
		Uint8Field:  50,
		Uint16Field: 1000,
		Uint32Field: 100000,
		Uint64Field: 10000000000,
	}
	cmd := &cobra.Command{}

	Define(cmd, opts)

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

func (suite *autoflagsSuite) TestDefine_IntTypesSupport() {
	opts := &intTestOptions{
		IntField:   1000,
		Int8Field:  42,
		Int16Field: 1234,
		Int32Field: 123456,
		Int64Field: 1234567890,
	}
	cmd := &cobra.Command{}

	Define(cmd, opts)

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

func (suite *autoflagsSuite) TestDefine_CountFlagSupport() {
	opts := &countTestOptions{Verbose: 0}
	cmd := &cobra.Command{}

	Define(cmd, opts)

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

func (suite *autoflagsSuite) TestDefine_SliceSupport() {
	opts := &sliceTestOptions{
		StringSliceField: []string{"default1", "default2"},
		IntSliceField:    []int{1, 2, 3},
	}
	cmd := &cobra.Command{}

	Define(cmd, opts)

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

func (suite *autoflagsSuite) TestDefine_NilPointerHandling() {
	// Test with nil pointer: it should not panic and should create same flags as zero-valued struct
	var nilOpts *testOptions = nil
	cmd1 := &cobra.Command{}

	assert.NotPanics(suite.T(), func() {
		Define(cmd1, nilOpts)
	})

	// Should create same flags as zero-valued struct
	zeroOpts := &testOptions{}
	cmd2 := &cobra.Command{}
	Define(cmd2, zeroOpts)

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
	LogLevel   zapcore.Level `flagcustom:"true" flagdescr:"log level"`
	ServerMode serverMode    `flagcustom:"true" flag:"server-mode" flagshort:"m" flagdescr:"set server mode"`
	SomeConfig string        `flagcustom:"true" flag:"some-config" flagshort:"c" flagdescr:"config file path"`
	NormalFlag string        `flag:"normal-flag" flagdescr:"normal description"`
}

func (o *comprehensiveCustomOptions) DefineServerMode(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	enhancedDesc := descr + fmt.Sprintf(" (%s,%s,%s)", string(development), string(staging), string(production))
	c.Flags().StringP(name, short, string(development), enhancedDesc)

	// Add shell completion
	c.RegisterFlagCompletionFunc(name, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{string(development), string(staging), string(production)}, cobra.ShellCompDirectiveDefault
	})
}

func (o *comprehensiveCustomOptions) DecodeServerMode(input any) (any, error) {
	return "", nil
}

func (o *comprehensiveCustomOptions) DefineSomeConfig(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	enhancedDesc := descr + " (must be .yaml, .yml, or .json)"
	c.Flags().StringP(name, short, "", enhancedDesc)

	c.RegisterFlagCompletionFunc(name, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"yaml", "yml", "json"}, cobra.ShellCompDirectiveFilterFileExt
	})
}

func (o *comprehensiveCustomOptions) DecodeSomeConfig(input any) (any, error) {
	return "", nil
}

func (o *comprehensiveCustomOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagcustom_ComprehensiveScenarios() {
	opts := &comprehensiveCustomOptions{}

	c := &cobra.Command{Use: "test"}
	err := Define(c, opts)
	require.NoError(suite.T(), err, "define should work for custom flags too")

	f := c.Flags()

	logLevelFlag := f.Lookup("loglevel")
	assert.NotNil(suite.T(), logLevelFlag, "log-level flag should be defined disregarding having or not having the flagcustom")
	assert.Equal(suite.T(), "log level {debug,info,warn,error,dpanic,panic,fatal}", logLevelFlag.Usage)

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

func (o *structFieldOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagcustom_EdgeCases() {
	// Test struct fields (should be ignored)
	structOpts := &structFieldOptions{}
	c1 := &cobra.Command{Use: "test1"}
	err := Define(c1, structOpts)

	require.Error(suite.T(), err, "custom methods should not be called for struct fields")
	require.ErrorIs(suite.T(), err, autoflagserrors.ErrInvalidTagUsage)
	require.Contains(suite.T(), err.Error(), "cannot be used on struct types")
	assert.Contains(suite.T(), err.Error(), "Nest")
	assert.False(suite.T(), structOpts.methodCalled, "custom methods should not be called for struct fields")
}

type envAnnotationsTestOptions struct {
	HasEnv string `flagenv:"true" flag:"has-env" flagdescr:"this will have len(envs) > 0"`
	NoEnv  string `flag:"no-env" flagdescr:"this will have len(envs) == 0"`
}

func (o *envAnnotationsTestOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestEnvAnnotations_WhenEnvsNotEmpty() {
	SetEnvPrefix("TEST")

	opts := &envAnnotationsTestOptions{}
	c := &cobra.Command{Use: "test"}
	Define(c, opts)

	f := c.Flags()

	// Case 1: len(envs) > 0 - should set annotation
	flagWithEnv := f.Lookup("has-env")
	assert.NotNil(suite.T(), flagWithEnv, "flag should exist")

	// The critical test: verify annotation was set
	envAnnotation := flagWithEnv.Annotations[flagEnvsAnnotation]
	assert.NotNil(suite.T(), envAnnotation, "annotation should be set when len(envs) > 0")
	assert.Greater(suite.T(), len(envAnnotation), 0, "annotation should contain env vars")
	assert.Contains(suite.T(), envAnnotation, "TEST_HAS_ENV", "should contain expected env var")
}

func (suite *autoflagsSuite) TestEnvAnnotations_WhenEnvsEmpty() {
	opts := &envAnnotationsTestOptions{}
	c := &cobra.Command{Use: "test"}
	Define(c, opts)

	f := c.Flags()

	// Case 2: len(envs) == 0 - should NOT set annotation
	flagWithoutEnv := f.Lookup("no-env")
	assert.NotNil(suite.T(), flagWithoutEnv, "flag should exist")

	// The critical test: verify annotation was NOT set
	envAnnotation := flagWithoutEnv.Annotations[flagEnvsAnnotation]
	assert.Nil(suite.T(), envAnnotation, "annotation should NOT be set when len(envs) == 0")
}

type requiredFlagsTestOptions struct {
	RequiredFlag     string `flag:"required-flag" flagrequired:"true" flagdescr:"this flag is required"`
	NotRequiredFlag  string `flag:"not-required-flag" flagrequired:"false" flagdescr:"this flag is not required"`
	DefaultFlag      string `flag:"default-flag" flagdescr:"this flag has no flagrequired tag"`
	RequiredWithDesc string `flagrequired:"true" flagdescr:"required flag without custom name"`
}

func (o *requiredFlagsTestOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagrequired_BasicFunctionality() {
	opts := &requiredFlagsTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	Define(cmd, opts)

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

func (suite *autoflagsSuite) TestFlagrequired_NestedStructs() {
	opts := &nestedRequiredFlagsOptions{}
	cmd := &cobra.Command{Use: "test"}

	Define(cmd, opts)

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

type validBooleanRequiredOptions struct {
	EmptyRequired string `flag:"empty-required" flagrequired:"" flagdescr:"empty flagrequired value"`
	CaseVariation string `flag:"case-variation" flagrequired:"True" flagdescr:"case variation test"`
}

func (o *validBooleanRequiredOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagrequired_ValidBooleanEdgeCases() {
	opts := &validBooleanRequiredOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)
	require.NoError(suite.T(), err)

	flags := cmd.Flags()

	// Test empty value - should be treated as false
	emptyRequiredFlag := flags.Lookup("empty-required")
	require.NotNil(suite.T(), emptyRequiredFlag, "empty-required should exist")

	emptyRequiredAnnotation := emptyRequiredFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.Nil(suite.T(), emptyRequiredAnnotation, "empty-required should not have required annotation due to empty value")

	// Test case variation - "True" should work since strconv.ParseBool accepts it
	caseVariationFlag := flags.Lookup("case-variation")
	require.NotNil(suite.T(), caseVariationFlag, "case-variation should exist")

	caseVariationAnnotation := caseVariationFlag.Annotations[cobra.BashCompOneRequiredFlag]
	assert.NotNil(suite.T(), caseVariationAnnotation, "case-variation should have required annotation since 'True' is valid")
	assert.Equal(suite.T(), []string{"true"}, caseVariationAnnotation)
}

type singleInvalidRequiredOption struct {
	InvalidTrue string `flag:"invalid-true" flagrequired:"yes" flagdescr:"invalid boolean value"`
}

func (o *singleInvalidRequiredOption) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagrequired_InvalidBooleanValue() {
	opts := &singleInvalidRequiredOption{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)
	require.Error(suite.T(), err, "Should return error for invalid flagrequired value")
	assert.Contains(suite.T(), err.Error(), "flagrequired")
	assert.Contains(suite.T(), err.Error(), "yes")
	assert.Contains(suite.T(), err.Error(), "InvalidTrue")
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

func (suite *autoflagsSuite) TestFlagrequired_MultipleTypes() {
	opts := &multipleTypesRequiredOptions{}
	cmd := &cobra.Command{Use: "test"}

	Define(cmd, opts)

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

func (suite *autoflagsSuite) TestFlagrequired_CombinedWithOtherTags() {
	opts := &requiredWithOtherTagsOptions{}
	cmd := &cobra.Command{Use: "test"}

	Define(cmd, opts)

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

	groupAnnotation := requiredGroupFlag.Annotations[flagGroupAnnotation]
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

	envAnnotation := requiredEnvFlag.Annotations[flagEnvsAnnotation]
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

func (suite *autoflagsSuite) TestDefine_NonAddressableFields() {
	suite.T().Run("interface_with_embedded_struct", func(t *testing.T) {
		// Create options with interface containing a struct
		opts := &testOptionsWithInterface{
			NormalField:    "test",
			InterfaceField: embeddedStruct{Value: "interface-value"},
		}

		cmd := &cobra.Command{Use: "test"}

		// This should not panic even if interface fields cause addressability issues
		assert.NotPanics(t, func() {
			Define(cmd, opts)
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
			Define(cmd1, structValue)
		})

		// Test with struct pointer (should be addressable)
		structPtr := &addressabilityTestOptions{StringField: "test", IntField: 42}
		cmd2 := &cobra.Command{Use: "test2"}

		assert.NotPanics(t, func() {
			Define(cmd2, structPtr)
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
			Define(cmd, opts)
		})

		// Should process the direct field
		directFlag := cmd.Flags().Lookup("direct")
		assert.NotNil(t, directFlag, "direct field should be processed")

		// Should process nested fields
		deepFlag := cmd.Flags().Lookup("deep-value")
		assert.NotNil(t, deepFlag, "deep nested field should be processed")
	})
}

func (suite *autoflagsSuite) TestDefine_CanAddrValidation() {
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
			Define(cmd, canAddrTestOptions{Field: "test"})
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

func (o *flagCustomTestOptions) DefineValidCustom(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "CUSTOM_DEFAULT", descr+" [CUSTOM]")
}

func (o *flagCustomTestOptions) DecodeValidCustom(input any) (any, error) {
	return input, nil
}

func (o *flagCustomTestOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagCustom_ShouldReturnError() {
	opts := &flagCustomTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

	assert.Error(suite.T(), err, "Should return error for invalid flagcustom value")
	assert.Contains(suite.T(), err.Error(), "flagcustom", "Error should mention flagcustom")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
	assert.Contains(suite.T(), err.Error(), "InvalidCustom", "Error should mention the field name")
}

type validFlagCustomOptions struct {
	TrueCustom  string `flagcustom:"true" flag:"true-custom" flagdescr:"should use custom"`
	FalseCustom string `flagcustom:"false" flag:"false-custom" flagdescr:"should not use custom"`
	EmptyCustom string `flagcustom:"" flag:"empty-custom" flagdescr:"should not use custom"`
	NoCustom    string `flag:"no-custom" flagdescr:"should not use custom"`
}

func (o *validFlagCustomOptions) DefineTrueCustom(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "CUSTOM_VALUE", descr+" [CUSTOM]")
}

func (o *validFlagCustomOptions) DecodeTrueCustom(input any) (any, error) {
	return input, nil
}

func (o *validFlagCustomOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagCustom_ValidValues() {
	opts := &validFlagCustomOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

	require.NoError(suite.T(), err, "Should not return error for valid flagcustom values")

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

func (o *flagCustomEdgeCasesOptions) DefineCaseTrue(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "CUSTOM_TRUE", descr)
}

func (o *flagCustomEdgeCasesOptions) DecodeCaseTrue(input any) (any, error) {
	return input, nil
}

func (o *flagCustomEdgeCasesOptions) DefineNumberOne(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "CUSTOM_ONE", descr)
}

func (o *flagCustomEdgeCasesOptions) DecodeNumberOne(input any) (any, error) {
	return input, nil
}

func (o *flagCustomEdgeCasesOptions) Attach(c *cobra.Command) {}

type validEdgeCasesOptions struct {
	CaseTrue   string `flagcustom:"True" flag:"case-true" flagdescr:"capital True"`
	CaseFalse  string `flagcustom:"FALSE" flag:"case-false" flagdescr:"capital FALSE"`
	NumberOne  string `flagcustom:"1" flag:"number-one" flagdescr:"number 1"`
	NumberZero string `flagcustom:"0" flag:"number-zero" flagdescr:"number 0"`
}

func (o *validEdgeCasesOptions) DefineCaseTrue(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "CUSTOM_TRUE", descr)
}

func (o *validEdgeCasesOptions) DecodeCaseTrue(input any) (any, error) {
	return input, nil
}

func (o *validEdgeCasesOptions) DefineNumberOne(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "CUSTOM_ONE", descr)
}

func (o *validEdgeCasesOptions) DecodeNumberOne(input any) (any, error) {
	return input, nil
}

func (o *validEdgeCasesOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagCustom_EdgeCases_ValidValues() {
	opts := &validEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// These should all pass validation
	err := Define(cmd, opts)
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

func (suite *autoflagsSuite) TestFlagCustom_EdgeCases_ShouldReturnError() {
	opts := &flagCustomEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Spaces should cause validation error
	err := Define(cmd, opts)

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

func (suite *autoflagsSuite) TestFlagenv_ShouldReturnError() {
	opts := &flagEnvTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

	assert.Error(suite.T(), err, "Should return error for invalid flagenv value")
	assert.Contains(suite.T(), err.Error(), "flagenv", "Error should mention flagenv")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
	assert.Contains(suite.T(), err.Error(), "InvalidEnv", "Error should mention the field name")
}

type validFlagEnvOptions struct {
	TrueEnv  string `flagenv:"true" flag:"true-env" flagdescr:"should have env"`
	FalseEnv string `flagenv:"false" flag:"false-env" flagdescr:"should not have env"`
	EmptyEnv string `flagenv:"" flag:"empty-env" flagdescr:"should not have env"`
	NoEnv    string `flag:"no-env" flagdescr:"should not have env"`
}

func (o *validFlagEnvOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagenv_ValidValues() {
	opts := &validFlagEnvOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

	assert.NoError(suite.T(), err, "Should not return error for valid flagenv values")

	// Check that flags are created correctly
	trueFlag := cmd.Flags().Lookup("true-env")
	falseFlag := cmd.Flags().Lookup("false-env")
	emptyFlag := cmd.Flags().Lookup("empty-env")
	noFlag := cmd.Flags().Lookup("no-env")

	// Check environment annotations
	trueEnvAnnotation := trueFlag.Annotations[flagEnvsAnnotation]
	falseEnvAnnotation := falseFlag.Annotations[flagEnvsAnnotation]
	emptyEnvAnnotation := emptyFlag.Annotations[flagEnvsAnnotation]
	noEnvAnnotation := noFlag.Annotations[flagEnvsAnnotation]

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

func (suite *autoflagsSuite) TestFlagenv_EdgeCases_ValidValues() {
	opts := &validEnvEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// These should all pass validation
	err := Define(cmd, opts)
	assert.NoError(suite.T(), err, "Should not return error for valid edge case values")

	// Check behavior
	caseTrueFlag := cmd.Flags().Lookup("case-true")
	caseFalseFlag := cmd.Flags().Lookup("case-false")
	numberOneFlag := cmd.Flags().Lookup("number-one")
	numberZeroFlag := cmd.Flags().Lookup("number-zero")

	// strconv.ParseBool accepts these case variations and numbers
	caseTrueAnnotation := caseTrueFlag.Annotations[flagEnvsAnnotation]
	caseFalseAnnotation := caseFalseFlag.Annotations[flagEnvsAnnotation]
	numberOneAnnotation := numberOneFlag.Annotations[flagEnvsAnnotation]
	numberZeroAnnotation := numberZeroFlag.Annotations[flagEnvsAnnotation]

	assert.NotNil(suite.T(), caseTrueAnnotation, "ParseBool should accept 'True' as true")
	assert.Nil(suite.T(), caseFalseAnnotation, "ParseBool should accept 'FALSE' as false")
	assert.NotNil(suite.T(), numberOneAnnotation, "ParseBool should accept '1' as true")
	assert.Nil(suite.T(), numberZeroAnnotation, "ParseBool should accept '0' as false")
}

func (suite *autoflagsSuite) TestFlagenv_EdgeCases_ShouldReturnError() {
	opts := &flagEnvEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Spaces should cause validation error
	err := Define(cmd, opts)

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

func (suite *autoflagsSuite) TestFlagenv_NestedStructs_WithValidation() {
	opts := &nestedFlagEnvOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

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

func (suite *autoflagsSuite) TestFlagenv_MultipleInvalid_ReturnsFirstError() {
	opts := &multipleInvalidEnvOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

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

func (o *flagEnvCombinedOptions) DefineEnvWithCustom(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "CUSTOM_DEFAULT", descr+" [CUSTOM]")
}

func (o *flagEnvCombinedOptions) DecodeEnvWithCustom(input any) (any, error) {
	return input, nil
}

func (o *flagEnvCombinedOptions) DefineInvalidEnvValid(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "INVALID", descr+" [INVALID]")
}

func (o *flagEnvCombinedOptions) DecodeInvalidEnvValid(input any) (any, error) {
	return input, nil
}

func (o *flagEnvCombinedOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagenv_CombinedWithOtherTags() {
	opts := &flagEnvCombinedOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Should fail due to invalid flagenv, even though flagcustom is valid
	err := Define(cmd, opts)

	require.Error(suite.T(), err, "Should return error for invalid flagenv value")
	assert.Contains(suite.T(), err.Error(), "flagenv", "Error should mention flagenv")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
	assert.Contains(suite.T(), err.Error(), "boolean", "Error should mention that the value must be boolean")
}

type bothInvalidOptions struct {
	InvalidBoth string `flagenv:"invalid" flagcustom:"invalid" flag:"invalid-both" flagdescr:"both invalid"`
}

func (o *bothInvalidOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagenv_BothInvalid_ReturnsFirstError() {
	opts := &bothInvalidOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

	assert.Error(suite.T(), err, "Should return error for invalid tag values")
	// Should return the first error (flagcustom is validated first in the current implementation)
	assert.Contains(suite.T(), err.Error(), "flagcustom", "Error should mention the first invalid tag")
}

type errorMessageOptions struct {
	BadValue string `flagenv:"maybe" flag:"bad-value" flagdescr:"bad boolean value"`
}

func (o *errorMessageOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagenv_ErrorMessages_ContainExpectedContent() {
	opts := &errorMessageOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

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

func (suite *autoflagsSuite) TestFlagignore_ShouldReturnError() {
	opts := &flagIgnoreTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

	assert.Error(suite.T(), err, "Should return error for invalid flagignore value")
	assert.Contains(suite.T(), err.Error(), "flagignore", "Error should mention flagignore")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
	assert.Contains(suite.T(), err.Error(), "InvalidIgnore", "Error should mention the field name")
}

type validFlagIgnoreOptions struct {
	TrueIgnore  string `flagignore:"true" flag:"true-ignore" flagdescr:"should be ignored"`
	FalseIgnore string `flagignore:"false" flag:"false-ignore" flagdescr:"should not be ignored"`
	EmptyIgnore string `flagignore:"" flag:"empty-ignore" flagdescr:"should not be ignored"`
	NoIgnore    string `flag:"no-ignore" flagdescr:"should not be ignored"`
}

func (o *validFlagIgnoreOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagignore_ValidValues() {
	opts := &validFlagIgnoreOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

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

type validIgnoreEdgeCasesOptions struct {
	CaseTrue   string `flagignore:"True" flag:"case-true" flagdescr:"capital True"`
	CaseFalse  string `flagignore:"FALSE" flag:"case-false" flagdescr:"capital FALSE"`
	NumberOne  string `flagignore:"1" flag:"number-one" flagdescr:"number 1"`
	NumberZero string `flagignore:"0" flag:"number-zero" flagdescr:"number 0"`
}

func (o *validIgnoreEdgeCasesOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagignore_EdgeCases_ValidValues() {
	opts := &validIgnoreEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// These should all pass validation
	err := Define(cmd, opts)
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

type flagIgnoreEdgeCasesOptions struct {
	CaseTrue   string `flagignore:"True" flag:"case-true" flagdescr:"capital True"`
	CaseFalse  string `flagignore:"FALSE" flag:"case-false" flagdescr:"capital FALSE"`
	NumberOne  string `flagignore:"1" flag:"number-one" flagdescr:"number 1"`
	NumberZero string `flagignore:"0" flag:"number-zero" flagdescr:"number 0"`
	WithSpaces string `flagignore:" true " flag:"with-spaces" flagdescr:"spaces around true"`
}

func (o *flagIgnoreEdgeCasesOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagignore_EdgeCases_ShouldReturnError() {
	opts := &flagIgnoreEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Spaces should cause validation error
	err := Define(cmd, opts)

	require.Error(suite.T(), err, "Should return error for flagignore value with spaces")
	assert.Contains(suite.T(), err.Error(), "flagignore", "Error should mention flagignore")
	assert.Contains(suite.T(), err.Error(), " true ", "Error should mention the invalid value with spaces")
	assert.Contains(suite.T(), err.Error(), "WithSpaces", "Error should mention the field name")
}

type flagIgnoreNestedStruct struct {
	Value string
}

type flagIgnoreStructFieldOptions struct {
	Nest flagIgnoreNestedStruct `flagignore:"true"`
}

func (o *flagIgnoreStructFieldOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagignore_OnStruct_ShouldReturnError() {
	opts := &flagIgnoreStructFieldOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Spaces should cause validation error
	err := Define(cmd, opts)

	require.Error(suite.T(), err, "Should return error when flagignore is used on struct type")
	assert.Contains(suite.T(), err.Error(), "flagignore", "Error should mention flagignore")
	assert.Contains(suite.T(), err.Error(), "Nest", "Error should mention field name")
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

func (suite *autoflagsSuite) TestFlagignore_NestedStructs_WithValidation() {
	opts := &nestedFlagIgnoreOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

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

func (suite *autoflagsSuite) TestFlagignore_MultipleInvalid_ReturnsFirstError() {
	opts := &multipleInvalidIgnoreOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

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

func (o *flagIgnoreCombinedOptions) DefineIgnoreWithCustom(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "CUSTOM_DEFAULT", descr+" [CUSTOM]")
}

func (o *flagIgnoreCombinedOptions) DecodeIgnoreWithCustom(input any) (any, error) {
	return input, nil
}

func (o *flagIgnoreCombinedOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagignore_CombinedWithOtherTags() {
	opts := &flagIgnoreCombinedOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Should fail due to invalid flagignore, even though other tags are valid
	err := Define(cmd, opts)

	assert.Error(suite.T(), err, "Should return error for invalid flagignore value")
	assert.Contains(suite.T(), err.Error(), "flagignore", "Error should mention flagignore")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
	assert.Contains(suite.T(), err.Error(), "boolean", "Error should mention that the value must be boolean")
}

type allThreeInvalidOptions struct {
	InvalidAll string `flagignore:"invalid" flagenv:"invalid" flagcustom:"invalid" flag:"invalid-all" flagdescr:"all three invalid"`
}

func (o *allThreeInvalidOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagignore_AllThreeInvalid_ReturnsFirstError() {
	opts := &allThreeInvalidOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

	assert.Error(suite.T(), err, "Should return error for invalid tag values")
	// Should return the first error (flagcustom is validated first in the current implementation)
	assert.Contains(suite.T(), err.Error(), "flagcustom", "Error should mention the first invalid tag")
}

type errorIgnoreMessageOptions struct {
	BadValue string `flagignore:"maybe" flag:"bad-value" flagdescr:"bad boolean value"`
}

func (o *errorIgnoreMessageOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagignore_ErrorMessages_ContainExpectedContent() {
	opts := &errorIgnoreMessageOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

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

func (suite *autoflagsSuite) TestFlagrequired_ShouldReturnError() {
	opts := &flagRequiredTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

	assert.Error(suite.T(), err, "Should return error for invalid flagrequired value")
	assert.Contains(suite.T(), err.Error(), "flagrequired", "Error should mention flagrequired")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
	assert.Contains(suite.T(), err.Error(), "InvalidRequired", "Error should mention the field name")
}

type validFlagRequiredOptions struct {
	TrueRequired  string `flagrequired:"true" flag:"true-required" flagdescr:"should be required"`
	FalseRequired string `flagrequired:"false" flag:"false-required" flagdescr:"should not be required"`
	EmptyRequired string `flagrequired:"" flag:"empty-required" flagdescr:"should not be required"`
	NoRequired    string `flag:"no-required" flagdescr:"should not be required"`
}

func (o *validFlagRequiredOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagrequired_ValidValues() {
	opts := &validFlagRequiredOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

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

type validRequiredEdgeCasesOptions struct {
	CaseTrue   string `flagrequired:"True" flag:"case-true" flagdescr:"capital True"`
	CaseFalse  string `flagrequired:"FALSE" flag:"case-false" flagdescr:"capital FALSE"`
	NumberOne  string `flagrequired:"1" flag:"number-one" flagdescr:"number 1"`
	NumberZero string `flagrequired:"0" flag:"number-zero" flagdescr:"number 0"`
}

func (o *validRequiredEdgeCasesOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagrequired_EdgeCases_ValidValues() {
	opts := &validRequiredEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// These should all pass validation
	err := Define(cmd, opts)
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

type flagRequiredConflicting struct {
	Conflict string `flagrequired:"true" flagignore:"true"`
}

func (o *flagRequiredConflicting) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagrequired_Conflicting_ShouldReturnError() {
	opts := &flagRequiredConflicting{}
	cmd := &cobra.Command{Use: "test"}

	// Spaces should cause validation error
	err := Define(cmd, opts)

	require.Error(suite.T(), err, "Should return error for flagrequired together with flagignore")
	require.ErrorIs(suite.T(), err, autoflagserrors.ErrConflictingTags)
	assert.Contains(suite.T(), err.Error(), "flagrequired", "Error should mention flagrequired")
	assert.Contains(suite.T(), err.Error(), "flagignore", "Error should mention flagignore")
	assert.Contains(suite.T(), err.Error(), "mutually exclusive", "Error should mention that flagrequied and flagignore are mutually exclusive")
}

type flagRequiredEdgeCasesOptions struct {
	CaseTrue   string `flagrequired:"True" flag:"case-true" flagdescr:"capital True"`
	CaseFalse  string `flagrequired:"FALSE" flag:"case-false" flagdescr:"capital FALSE"`
	NumberOne  string `flagrequired:"1" flag:"number-one" flagdescr:"number 1"`
	NumberZero string `flagrequired:"0" flag:"number-zero" flagdescr:"number 0"`
	WithSpaces string `flagrequired:" true " flag:"with-spaces" flagdescr:"spaces around true"`
}

func (o *flagRequiredEdgeCasesOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagrequired_EdgeCases_ShouldReturnError() {
	opts := &flagRequiredEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Spaces should cause validation error
	err := Define(cmd, opts)

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

func (suite *autoflagsSuite) TestFlagrequired_NestedStructs_WithValidation() {
	opts := &nestedFlagRequiredOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

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

func (suite *autoflagsSuite) TestFlagrequired_MultipleInvalid_ReturnsFirstError() {
	opts := &multipleInvalidRequiredOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

	assert.Error(suite.T(), err, "Should return error for invalid flagrequired values")
	assert.Contains(suite.T(), err.Error(), "flagrequired", "Error should mention flagrequired")
	// Should return the first error encountered (InvalidRequired1)
	assert.Contains(suite.T(), err.Error(), "InvalidRequired1", "Error should mention the first invalid field")
}

type allFourInvalidOptions struct {
	InvalidAll string `flagrequired:"invalid" flagignore:"invalid" flagenv:"invalid" flagcustom:"invalid" flag:"invalid-all" flagdescr:"all four invalid"`
}

func (o *allFourInvalidOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagrequired_AllFourInvalid_ReturnsFirstError() {
	opts := &allFourInvalidOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

	assert.Error(suite.T(), err, "Should return error for invalid tag values")
	// Should return the first error (flagcustom is validated first in the current implementation)
	assert.Contains(suite.T(), err.Error(), "flagcustom", "Error should mention the first invalid tag")
}

type errorRequiredMessageOptions struct {
	BadValue string `flagrequired:"maybe" flag:"bad-value" flagdescr:"bad boolean value"`
}

func (o *errorRequiredMessageOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagrequired_ErrorMessages_ContainExpectedContent() {
	opts := &errorRequiredMessageOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

	assert.Error(suite.T(), err, "Should return error for invalid flagrequired")

	errorMsg := err.Error()

	// These are the expected components of a FieldError
	assert.Contains(suite.T(), errorMsg, "BadValue", "Error should contain field name")
	assert.Contains(suite.T(), errorMsg, "flagrequired", "Error should contain tag name")
	assert.Contains(suite.T(), errorMsg, "maybe", "Error should contain tag value")
	assert.Contains(suite.T(), errorMsg, "invalid boolean value", "Error should contain message")
}

type exclusionsNestedStruct struct {
	NestedFlag     string `flag:"nested-flag" flagdescr:"nested flag"`
	ExcludedNested string `flag:"excluded-nested" flagdescr:"should be excluded"`
}

func (suite *autoflagsSuite) TestWithExclusions_BasicExclusion() {
	opts := &exclusionsTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	Define(cmd, opts, WithExclusions("excluded-flag"))

	flags := cmd.Flags()

	// Normal flag should be created
	normalFlag := flags.Lookup("normal-flag")
	require.NotNil(suite.T(), normalFlag, "--normal-flag should be created")

	// Excluded flag should not be created
	excludedFlag := flags.Lookup("excluded-flag")
	require.Nil(suite.T(), excludedFlag, "--excluded-flag should not be created")

	// Other flags should be created
	aliasFlag := flags.Lookup("alias-flag")
	require.NotNil(suite.T(), aliasFlag, "--alias-flag should be created")

	// The aliases are not case insensitive
	caseFlag := flags.Lookup("Case-Flag")
	require.NotNil(suite.T(), caseFlag, "--case-flag should be created")

	noAlias := flags.Lookup("noalias")
	require.NotNil(suite.T(), noAlias, "--noalias should be created")
}

type exclusionsTestOptions struct {
	NormalFlag   string `flag:"normal-flag" flagdescr:"should be created"`
	ExcludedFlag string `flag:"excluded-flag" flagdescr:"should be excluded"`
	AliasFlag    string `flag:"alias-flag" flagdescr:"has alias"`
	CaseFlag     string `flag:"Case-Flag" flagdescr:"mixed case flag"`
	NoAlias      string `flagdescr:"no alias"`
	NestedStruct exclusionsNestedStruct
}

func (o *exclusionsTestOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestWithExclusions_MultipleExclusions() {
	opts := &exclusionsTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	Define(cmd, opts, WithExclusions("excluded-flag", "--noalias"))

	flags := cmd.Flags()

	// Normal flags should be created
	normalFlag := flags.Lookup("normal-flag")
	assert.NotNil(suite.T(), normalFlag, "--normal-flag should be created")

	caseFlag := flags.Lookup("Case-Flag")
	assert.NotNil(suite.T(), caseFlag, "--case-flag should be created")

	// Both excluded flags should not be created
	excludedFlag := flags.Lookup("excluded-flag")
	assert.Nil(suite.T(), excludedFlag, "--excluded-flag should not be created")

	noAlias := flags.Lookup("noalias")
	assert.Nil(suite.T(), noAlias, "--alias-flag should not be created")
}

func (suite *autoflagsSuite) TestWithExclusions_CaseInsensitive() {
	opts := &exclusionsTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Exclude using different case than the flag definition
	Define(cmd, opts, WithExclusions("CASE-FLAG"))

	flags := cmd.Flags()

	// Case flag should be excluded despite case difference
	caseFlag := flags.Lookup("case-flag")
	assert.Nil(suite.T(), caseFlag, "case-flag should be excluded (case insensitive)")

	// Other flags should be created
	normalFlag := flags.Lookup("normal-flag")
	assert.NotNil(suite.T(), normalFlag, "normal-flag should be created")
}

func (suite *autoflagsSuite) TestWithExclusions_NestedStructFlags() {
	opts := &exclusionsTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	Define(cmd, opts, WithExclusions("excluded-nested"))

	flags := cmd.Flags()

	// Normal nested flag should be created
	nestedFlag := flags.Lookup("nested-flag")
	assert.NotNil(suite.T(), nestedFlag, "nested-flag should be created")

	// Excluded nested flag should not be created
	excludedNested := flags.Lookup("excluded-nested")
	assert.Nil(suite.T(), excludedNested, "excluded-nested should not be created")

	// Top-level flags should be created
	normalFlag := flags.Lookup("normal-flag")
	assert.NotNil(suite.T(), normalFlag, "normal-flag should be created")
}

func (suite *autoflagsSuite) TestWithExclusions_NestedPath() {
	opts := &exclusionsTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Test excluding using the full nested path (<field_name>.<field_name>)
	Define(cmd, opts, WithExclusions("nestedstruct.excludednested"))

	flags := cmd.Flags()

	// Should exclude the nested flag using its full path
	excludedNested := flags.Lookup("excluded-nested")
	assert.Nil(suite.T(), excludedNested, "excluded-nested should be excluded using full path")

	// Other flags should be created
	nestedFlag := flags.Lookup("nested-flag")
	assert.NotNil(suite.T(), nestedFlag, "nested-flag should be created")
}

type exclusionsAliasTestOptions struct {
	FlagWithAlias string `flag:"custom-name" flagdescr:"flag with custom name"`
	NormalFlag    string `flagdescr:"auto-named flag"`
}

func (o *exclusionsAliasTestOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestWithExclusions_AliasExclusion() {
	opts := &exclusionsAliasTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Exclude using the alias name
	Define(cmd, opts, WithExclusions("custom-name"))

	flags := cmd.Flags()

	// Flag with alias should be excluded when excluded by alias
	aliasedFlag := flags.Lookup("custom-name")
	assert.Nil(suite.T(), aliasedFlag, "flag should be excluded when alias is excluded")

	// Normal flag should be created
	normalFlag := flags.Lookup("normalflag")
	assert.NotNil(suite.T(), normalFlag, "normalflag should be created")
}

func (suite *autoflagsSuite) TestWithExclusions_PathVsAlias() {
	opts := &exclusionsAliasTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Exclude using the path name (not the alias)
	Define(cmd, opts, WithExclusions("flagwithalias"))

	flags := cmd.Flags()

	// Flag should be excluded when excluded by path
	aliasedFlag := flags.Lookup("custom-name")
	assert.Nil(suite.T(), aliasedFlag, "flag should be excluded when path is excluded")

	// Normal flag should be created
	normalFlag := flags.Lookup("normalflag")
	assert.NotNil(suite.T(), normalFlag, "normalflag should be created")
}

func (suite *autoflagsSuite) TestWithExclusions_CommandSpecific() {
	opts := &exclusionsTestOptions{}
	cmd1 := &cobra.Command{Use: "command1"}
	cmd2 := &cobra.Command{Use: "command2"}

	// Apply exclusions to command1 only
	Define(cmd1, opts, WithExclusions("excluded-flag"))
	Define(cmd2, opts) // No exclusions

	flags1 := cmd1.Flags()
	flags2 := cmd2.Flags()

	// command1 should have the flag excluded
	excludedFlag1 := flags1.Lookup("excluded-flag")
	assert.Nil(suite.T(), excludedFlag1, "excluded-flag should not be created in command1")

	// command2 should have the flag created
	excludedFlag2 := flags2.Lookup("excluded-flag")
	assert.NotNil(suite.T(), excludedFlag2, "excluded-flag should be created in command2")

	// Both commands should have normal flags
	normalFlag1 := flags1.Lookup("normal-flag")
	normalFlag2 := flags2.Lookup("normal-flag")
	assert.NotNil(suite.T(), normalFlag1, "normal-flag should be created in command1")
	assert.NotNil(suite.T(), normalFlag2, "normal-flag should be created in command2")
}

func (suite *autoflagsSuite) TestWithExclusions_EmptyExclusions() {
	opts := &exclusionsTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Test with empty exclusions list
	Define(cmd, opts, WithExclusions())

	flags := cmd.Flags()

	// All flags should be created
	normalFlag := flags.Lookup("normal-flag")
	excludedFlag := flags.Lookup("excluded-flag")
	aliasFlag := flags.Lookup("alias-flag")

	assert.NotNil(suite.T(), normalFlag, "normal-flag should be created")
	assert.NotNil(suite.T(), excludedFlag, "excluded-flag should be created when not excluded")
	assert.NotNil(suite.T(), aliasFlag, "alias-flag should be created")
}

func (suite *autoflagsSuite) TestWithExclusions_NoExclusionsOption() {
	opts := &exclusionsTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Test without any exclusions option
	Define(cmd, opts)

	flags := cmd.Flags()

	// All flags should be created
	normalFlag := flags.Lookup("normal-flag")
	excludedFlag := flags.Lookup("excluded-flag")
	aliasFlag := flags.Lookup("alias-flag")

	assert.NotNil(suite.T(), normalFlag, "normal-flag should be created")
	assert.NotNil(suite.T(), excludedFlag, "excluded-flag should be created when no exclusions")
	assert.NotNil(suite.T(), aliasFlag, "alias-flag should be created")
}

func (suite *autoflagsSuite) TestWithExclusions_DuplicateExclusions() {
	opts := &exclusionsTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Test with duplicate exclusions (should be handled gracefully)
	Define(cmd, opts, WithExclusions("excluded-flag", "excluded-flag", "alias-flag"))

	flags := cmd.Flags()

	// Should work the same as without duplicates
	excludedFlag := flags.Lookup("excluded-flag")
	aliasFlag := flags.Lookup("alias-flag")
	normalFlag := flags.Lookup("normal-flag")

	assert.Nil(suite.T(), excludedFlag, "excluded-flag should not be created")
	assert.Nil(suite.T(), aliasFlag, "alias-flag should not be created")
	assert.NotNil(suite.T(), normalFlag, "normal-flag should be created")
}

type exclusionsSpecialCasesOptions struct {
	DashedFlag     string `flag:"flag-with-dashes" flagdescr:"flag with dashes"`
	UnderscoreFlag string `flag:"flag_with_underscores" flagdescr:"flag with underscores"`
	CamelCaseFlag  string `flag:"CamelCase"`
	NumberFlag     string `flag:"flag123" flagdescr:"flag with numbers"`
}

func (o *exclusionsSpecialCasesOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestWithExclusions_SpecialCharacters() {
	opts := &exclusionsSpecialCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	Define(cmd, opts, WithExclusions("flag-with-dashes", "flag_with_underscores", "camelcase"))

	flags := cmd.Flags()

	// Flags with special characters should be properly excluded
	dashedFlag := flags.Lookup("flag-with-dashes")
	underscoreFlag := flags.Lookup("flag_with_underscores")
	camelcaseFlag := flags.Lookup("CamelCase")
	numberFlag := flags.Lookup("flag123")

	require.Nil(suite.T(), dashedFlag, "--flag-with-dashes should be excluded")
	require.Nil(suite.T(), underscoreFlag, "--flag_with_underscores should be excluded")
	require.Nil(suite.T(), camelcaseFlag, "--CamelCase should be excluded")
	require.NotNil(suite.T(), numberFlag, "flag123 should be created")
}

type flagShortTestOptions struct {
	ValidShort     string `flagshort:"v" flag:"valid-short" flagdescr:"should use single char shorthand"`
	InvalidShort   string `flagshort:"verb" flag:"invalid-short" flagdescr:"has invalid multi-char flagshort"`
	EmptyShort     string `flagshort:"" flag:"empty-short" flagdescr:"has empty flagshort value"`
	AnotherInvalid string `flagshort:"abc" flag:"another-invalid" flagdescr:"another multi-char shorthand"`
	NormalField    string `flag:"normal" flagdescr:"normal field without flagshort"`
}

func (o *flagShortTestOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagshort_AlwaysValidated_ShouldReturnError() {
	opts := &flagShortTestOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Multi-character shorthand should ALWAYS return error (regardless of WithValidation)
	err := Define(cmd, opts)

	require.Error(suite.T(), err, "Should always return error for invalid flagshort value")
	require.ErrorIs(suite.T(), err, autoflagserrors.ErrInvalidShorthand)
	assert.Contains(suite.T(), err.Error(), "shorthand", "Error should mention shorthand")
	assert.Contains(suite.T(), err.Error(), "verb", "Error should mention the invalid value")
	assert.Contains(suite.T(), err.Error(), "InvalidShort", "Error should mention the field name")
	assert.Contains(suite.T(), err.Error(), "field 'InvalidShort': shorthand flag 'verb' must be a single character", "Error should have correct message")
}

type flagShortNestedStruct struct {
	Value string
}

type flagShortStructFieldOptions struct {
	Nest flagShortNestedStruct `flagshort:"n"`
}

func (o *flagShortStructFieldOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagshort_OnStruct_ShouldReturnError() {
	opts := &flagShortStructFieldOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Spaces should cause validation error
	err := Define(cmd, opts)

	require.Error(suite.T(), err, "Should return error when flagshort is used on struct type")
	assert.Contains(suite.T(), err.Error(), "flagshort", "Error should mention flagshort")
	assert.Contains(suite.T(), err.Error(), "Nest", "Error should mention field name")
}

type flagShortEdgeCasesOptions struct {
	SingleChar  string `flagshort:"x" flag:"single" flagdescr:"single character"`
	TwoChars    string `flagshort:"ab" flag:"two-chars" flagdescr:"two characters"`
	ThreeChars  string `flagshort:"xyz" flag:"three-chars" flagdescr:"three characters"`
	WithSpaces  string `flagshort:" v " flag:"with-spaces" flagdescr:"spaces around char"`
	SpecialChar string `flagshort:"@" flag:"special" flagdescr:"special character"`
	NumberChar  string `flagshort:"1" flag:"number" flagdescr:"number character"`
}

func (o *flagShortEdgeCasesOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagshort_EdgeCases_InvalidValues_AlwaysError() {
	opts := &flagShortEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// Multi-character shorthand should always cause error
	err := Define(cmd, opts)

	require.Error(suite.T(), err, "Should always return error for multi-character flagshort values")
	require.ErrorIs(suite.T(), err, autoflagserrors.ErrInvalidShorthand)
	assert.Contains(suite.T(), err.Error(), "shorthand", "Error should mention shorthand")
	// Should contain one of the invalid values
	errorContainsInvalid := strings.Contains(err.Error(), "ab") ||
		strings.Contains(err.Error(), "xyz") ||
		strings.Contains(err.Error(), " v ")
	assert.True(suite.T(), errorContainsInvalid, "Error should mention one of the invalid multi-char values")
}

type validShortEdgeCasesOptions struct {
	SingleChar  string `flagshort:"x" flag:"single" flagdescr:"single character"`
	SpecialChar string `flagshort:"@" flag:"special" flagdescr:"special character"`
	NumberChar  string `flagshort:"1" flag:"number" flagdescr:"number character"`
}

func (o *validShortEdgeCasesOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagshort_EdgeCases_ValidValues() {
	opts := &validShortEdgeCasesOptions{}
	cmd := &cobra.Command{Use: "test"}

	// These should all pass validation
	err := Define(cmd, opts)
	assert.NoError(suite.T(), err, "Should not return error for valid single-character edge cases")

	// Check behavior
	flags := cmd.Flags()
	singleFlag := flags.Lookup("single")
	specialFlag := flags.Lookup("special")
	numberFlag := flags.Lookup("number")

	assert.Equal(suite.T(), "x", singleFlag.Shorthand, "Should accept normal single char")
	assert.Equal(suite.T(), "@", specialFlag.Shorthand, "Should accept special character")
	assert.Equal(suite.T(), "1", numberFlag.Shorthand, "Should accept number character")
}

type nestedFlagShortOptions struct {
	TopLevel     string            `flag:"top-level" flagshort:"t" flagdescr:"top level flag"`
	NestedStruct nestedShortStruct `flaggroup:"Nested"`
}

type nestedShortStruct struct {
	ValidNestedShort   string `flag:"nested-valid" flagshort:"n" flagdescr:"nested valid short"`
	InvalidNestedShort string `flag:"nested-invalid" flagshort:"invalid" flagdescr:"nested invalid short"`
}

func (o *nestedFlagShortOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestFlagshort_NestedStructs_AlwaysValidated() {
	opts := &nestedFlagShortOptions{}
	cmd := &cobra.Command{Use: "test"}

	err := Define(cmd, opts)

	require.Error(suite.T(), err, "Should always return error for invalid nested flagshort value")
	require.ErrorIs(suite.T(), err, autoflagserrors.ErrInvalidShorthand)
	assert.Contains(suite.T(), err.Error(), "shorthand", "Error should mention shorthand")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention the invalid value")
	assert.Contains(suite.T(), err.Error(), "NestedStruct.InvalidNestedShort", "Error should mention the nested field name")
}

type simpleValidOptions struct {
	Name    string `flag:"name" flagdescr:"user name"`
	Port    int    `flag:"port" flagdescr:"server port"`
	Verbose bool   `flag:"verbose" flagshort:"v" flagdescr:"verbose output"`
}

func (o simpleValidOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestDefine_InvalidValueFallback() {
	suite.T().Run("nil_interface_returns_error", func(t *testing.T) {
		// Test with a nil interface (not a nil pointer to a specific type)
		var nilInterface Options
		cmd := &cobra.Command{Use: "test"}

		// This should return an error, not panic
		err := Define(cmd, nilInterface)

		require.Error(t, err, "nil interface should return error")
		assert.Contains(t, err.Error(), "cannot define flags")
		assert.Contains(t, err.Error(), "invalid input value of type 'nil'")

		// Should create no flags since there's an error
		flagCount := 0
		cmd.Flags().VisitAll(func(flag *pflag.Flag) { flagCount++ })
		assert.Equal(t, 0, flagCount, "nil interface should create no flags")
	})

	suite.T().Run("nil_typed_interface_succeeds", func(t *testing.T) {
		// Test with a typed nil interface using a simple struct
		var typedNil *simpleValidOptions = nil
		var nilInterface Options = typedNil
		cmd := &cobra.Command{Use: "test"}

		// This should succeed using the fallback path
		err := Define(cmd, nilInterface)

		require.NoError(t, err, "typed nil interface should succeed via fallback")

		// Should create flags as if it was a zero-valued struct
		normalOpts := &simpleValidOptions{}
		cmd2 := &cobra.Command{Use: "test2"}
		Define(cmd2, normalOpts)

		nilInterfaceFlags := 0
		cmd.Flags().VisitAll(func(flag *pflag.Flag) { nilInterfaceFlags++ })

		normalFlags := 0
		cmd2.Flags().VisitAll(func(flag *pflag.Flag) { normalFlags++ })

		assert.Equal(t, normalFlags, nilInterfaceFlags, "typed nil interface should create same flags as normal struct")
	})

	suite.T().Run("direct_nil_pointer_succeeds", func(t *testing.T) {
		// Test with a direct nil pointer
		var nilPtr *simpleValidOptions = nil
		cmd := &cobra.Command{Use: "test"}

		// Should succeed via fallback
		err := Define(cmd, nilPtr)

		require.NoError(t, err, "direct nil pointer should succeed via fallback")

		// Should create flags as if it was a zero-valued struct
		normalOpts := &simpleValidOptions{}
		cmd2 := &cobra.Command{Use: "test2"}
		Define(cmd2, normalOpts)

		nilPtrFlags := 0
		cmd.Flags().VisitAll(func(flag *pflag.Flag) { nilPtrFlags++ })

		normalFlags := 0
		cmd2.Flags().VisitAll(func(flag *pflag.Flag) { normalFlags++ })

		assert.Equal(t, normalFlags, nilPtrFlags, "direct nil pointer should create same flags as normal struct")
	})
}

func (suite *autoflagsSuite) TestDefine_GetValueEdgeCases() {
	suite.T().Run("zero_value_struct", func(t *testing.T) {
		// Test with a zero-valued struct (not pointer)
		opts := simpleValidOptions{} // zero value, not pointer
		cmd := &cobra.Command{Use: "test"}

		err := Define(cmd, opts)
		assert.NoError(t, err, "zero-value struct should work")

		// Should create flags normally
		flagCount := 0
		cmd.Flags().VisitAll(func(flag *pflag.Flag) { flagCount++ })
		assert.Greater(t, flagCount, 0, "zero-value struct should create flags")
	})

	suite.T().Run("compare_pointer_vs_value_handling", func(t *testing.T) {
		// Compare behavior between pointer and value for the same struct
		optsValue := simpleValidOptions{}
		optsPtr := &simpleValidOptions{}

		cmd1 := &cobra.Command{Use: "test1"}
		cmd2 := &cobra.Command{Use: "test2"}

		// Both should work without errors
		err1 := Define(cmd1, optsValue) // struct value
		err2 := Define(cmd2, optsPtr)   // struct pointer

		assert.NoError(t, err1, "struct value should work")
		assert.NoError(t, err2, "struct pointer should work")

		// Should create equivalent flags
		flags1 := []string{}
		cmd1.Flags().VisitAll(func(flag *pflag.Flag) { flags1 = append(flags1, flag.Name) })

		flags2 := []string{}
		cmd2.Flags().VisitAll(func(flag *pflag.Flag) { flags2 = append(flags2, flag.Name) })

		assert.ElementsMatch(t, flags1, flags2, "pointer and value should create equivalent flags")
	})
}

func (suite *autoflagsSuite) TestDefine_ReflectionEdgeCases() {
	suite.T().Run("interface_containing_nil_pointer", func(t *testing.T) {
		// Create an interface that contains a nil pointer
		var nilPtr *simpleValidOptions = nil
		var opts Options = nilPtr
		cmd := &cobra.Command{Use: "test"}

		// This should trigger the fallback path and succeed
		err := Define(cmd, opts)
		assert.NoError(t, err, "interface containing nil pointer should succeed via fallback")

		// Verify it behaves like a normal zero-valued struct
		normalOpts := &simpleValidOptions{}
		cmd2 := &cobra.Command{Use: "test2"}
		Define(cmd2, normalOpts)

		// Count flags
		nilInterfaceFlags := 0
		cmd.Flags().VisitAll(func(flag *pflag.Flag) { nilInterfaceFlags++ })

		normalFlags := 0
		cmd2.Flags().VisitAll(func(flag *pflag.Flag) { normalFlags++ })

		assert.Equal(t, normalFlags, nilInterfaceFlags, "interface containing nil pointer should create same flags as normal struct")
	})

	suite.T().Run("invalid_reflection_scenarios", func(t *testing.T) {
		// Test various edge cases that might result in invalid reflection
		testCases := []struct {
			name        string
			opts        Options
			shouldError bool
		}{
			{
				name:        "untyped_nil",
				opts:        nil,
				shouldError: true, // Only untyped nil should error
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cmd := &cobra.Command{Use: "test"}

				// Should not panic regardless of input
				err := Define(cmd, tc.opts)

				if tc.shouldError {
					require.Error(t, err, "should return error for %s", tc.name)
					require.ErrorIs(t, err, autoflagserrors.ErrInputValue)
				} else {
					require.NoError(t, err, "should succeed for %s", tc.name)
				}
			})
		}
	})
}

func (suite *autoflagsSuite) TestDefine_NilPointerHandling_Extended() {
	suite.T().Run("nil_interface_vs_nil_pointer", func(t *testing.T) {
		// Test the difference between nil interface and nil pointer
		var nilInterface Options = nil           // untyped nil - should error
		var nilPointer *simpleValidOptions = nil // typed nil - should succeed

		cmd1 := &cobra.Command{Use: "test1"}
		cmd2 := &cobra.Command{Use: "test2"}

		// nil interface should error
		err1 := Define(cmd1, nilInterface)
		assert.Error(t, err1, "untyped nil interface should error")

		// nil pointer should succeed
		err2 := Define(cmd2, nilPointer)
		assert.NoError(t, err2, "typed nil pointer should succeed")

		// nil pointer should create same flags as normal struct
		normalOpts := &simpleValidOptions{}
		cmd3 := &cobra.Command{Use: "test3"}
		Define(cmd3, normalOpts)

		nilPointerFlags := 0
		cmd2.Flags().VisitAll(func(flag *pflag.Flag) { nilPointerFlags++ })

		normalFlags := 0
		cmd3.Flags().VisitAll(func(flag *pflag.Flag) { normalFlags++ })

		assert.Equal(t, normalFlags, nilPointerFlags, "nil pointer should create same flags as normal struct")
	})

	suite.T().Run("getValue_fallback_behavior", func(t *testing.T) {
		// Specifically test that the getValue fallback works correctly
		var opts Options = (*simpleValidOptions)(nil)
		cmd := &cobra.Command{Use: "test"}

		// This should use the fallback: getValue(getValuePtr(o).Interface())
		err := Define(cmd, opts)
		assert.NoError(t, err, "fallback path should work without errors")

		// Should create flags as if it was a zero-valued struct
		flagCount := 0
		cmd.Flags().VisitAll(func(flag *pflag.Flag) { flagCount++ })

		// Compare with a normal zero-valued struct
		normalOpts := &simpleValidOptions{}
		cmd2 := &cobra.Command{Use: "test2"}
		Define(cmd2, normalOpts)

		normalFlagCount := 0
		cmd2.Flags().VisitAll(func(flag *pflag.Flag) { normalFlagCount++ })

		assert.Equal(t, normalFlagCount, flagCount, "fallback path should create same flags as normal zero-valued struct")
	})
}

type missingDecodeHookOptions struct {
	CustomField string `flagcustom:"true" flag:"custom-field"`
}

func (o *missingDecodeHookOptions) DefineCustomField(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "default", descr)
}

func (o *missingDecodeHookOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestValidateCustomFlag_MissingDecodeHook() {
	opts := &missingDecodeHookOptions{}
	cmd := &cobra.Command{Use: "test"}

	// This should error because the decode hook is missing
	err := Define(cmd, opts)

	require.Error(suite.T(), err, "Should return error when decode hook is missing")
	assert.Contains(suite.T(), err.Error(), "missing decode hook", "Error should mention missing decode hook")
	assert.Contains(suite.T(), err.Error(), "DecodeCustomField", "Error should mention the expected decode hook name")
	assert.Contains(suite.T(), err.Error(), "CustomField", "Error should mention the field name")
}

type missingDefineHookOptions struct {
	CustomField string `flagcustom:"true" flag:"custom-field"`
}

// No DefineCustomField method - define hook missing
func (o *missingDefineHookOptions) DecodeCustomField(input any) (any, error) {
	return input, nil
}

func (o *missingDefineHookOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestValidateCustomFlag_MissingDefineHook() {
	opts := &missingDefineHookOptions{}
	cmd := &cobra.Command{Use: "test"}

	// This should error because the define hook is missing
	err := Define(cmd, opts)

	require.Error(suite.T(), err, "Should return error when define hook is missing")
	require.ErrorIs(suite.T(), err, autoflagserrors.ErrMissingDefineHook)
	assert.Contains(suite.T(), err.Error(), "missing define hook", "Error should mention missing define hook")
	assert.Contains(suite.T(), err.Error(), "DefineCustomField", "Error should mention the expected define hook name")
	assert.Contains(suite.T(), err.Error(), "CustomField", "Error should mention the field name")
}

type wrongDefineSignatureOptions struct {
	CustomField string `flagcustom:"true" flag:"custom-field"`
}

func (o *wrongDefineSignatureOptions) DefineCustomField(wrongParam string) {
	// Wrong signature - should have (c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value)
}

func (o *wrongDefineSignatureOptions) DecodeCustomField(input any) (any, error) {
	return input, nil
}

func (o *wrongDefineSignatureOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestValidateCustomFlag_WrongDefineSignature() {
	opts := &wrongDefineSignatureOptions{}
	cmd := &cobra.Command{Use: "test"}

	// This should error because define hook has wrong signature
	err := Define(cmd, opts)

	require.Error(suite.T(), err, "Should return error when define hook has wrong signature")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention invalid signature")
	assert.Contains(suite.T(), err.Error(), "DefineCustomField", "Error should mention the hook name")
	assert.Contains(suite.T(), err.Error(), "define hook", "Error should identify it as a define hook error")

	var fx DefineHookFunc
	require.Contains(suite.T(), err.Error(), signature(fx))
}

type wrongDecodeSignatureOptions struct {
	CustomField string `flagcustom:"true" flag:"custom-field"`
}

func (o *wrongDecodeSignatureOptions) DefineCustomField(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "default", descr)
}

func (o *wrongDecodeSignatureOptions) DecodeCustomField(wrongParam string, anotherParam int) {
	// Wrong signature - should have (input any) (any, error)
}

func (o *wrongDecodeSignatureOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestValidateCustomFlag_WrongDecodeSignature() {
	opts := &wrongDecodeSignatureOptions{}
	cmd := &cobra.Command{Use: "test"}

	// This should error because decode hook has wrong signature
	err := Define(cmd, opts)

	require.Error(suite.T(), err, "Should return error when decode hook has wrong signature")
	assert.Contains(suite.T(), err.Error(), "invalid", "Error should mention invalid signature")
	assert.Contains(suite.T(), err.Error(), "DecodeCustomField", "Error should mention the hook name")
	assert.Contains(suite.T(), err.Error(), "decode hook", "Error should identify it as a decode hook error")

	var fx DecodeHookFunc
	require.Contains(suite.T(), err.Error(), signature(fx))
}

type wrongDecodeReturnOptions struct {
	CustomField string `flagcustom:"true" flag:"custom-field"`
}

func (o *wrongDecodeReturnOptions) DefineCustomField(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "default", descr)
}

func (o *wrongDecodeReturnOptions) DecodeCustomField(input any) string {
	// Wrong signature - should return (any, error), not just string
	return "value"
}

func (o *wrongDecodeReturnOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestValidateCustomFlag_WrongDecodeReturnSignature() {
	opts := &wrongDecodeReturnOptions{}
	cmd := &cobra.Command{Use: "test"}

	// This should error because decode hook returns wrong number of values
	err := Define(cmd, opts)

	require.Error(suite.T(), err, "Should return error when decode hook returns wrong number of values")
	assert.Contains(suite.T(), err.Error(), "(interface {}, error)", "Error should mention correct return signature")
	assert.Contains(suite.T(), err.Error(), "DecodeCustomField", "Error should mention the hook name")
}

type correctHooksOptions struct {
	CustomField string `flagcustom:"true" flag:"custom-field"`
}

func (o *correctHooksOptions) DefineCustomField(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "default", descr)
}

func (o *correctHooksOptions) DecodeCustomField(input any) (any, error) {
	return input, nil
}

func (o *correctHooksOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestValidateCustomFlag_CorrectHooks() {
	opts := &correctHooksOptions{}
	cmd := &cobra.Command{Use: "test"}

	// This should succeed because both hooks are correctly defined
	err := Define(cmd, opts)

	assert.NoError(suite.T(), err, "Should not return error when both hooks are correctly defined")

	// Verify that the flag was actually created
	flag := cmd.Flags().Lookup("custom-field")
	assert.NotNil(suite.T(), flag, "Custom flag should be created")
	assert.Equal(suite.T(), "default", flag.DefValue, "Flag should have default value from define hook")
}

func (suite *autoflagsSuite) TestValidateCustomFlag_ErrorTypes() {
	suite.T().Run("missing_decode_hook_error_type", func(t *testing.T) {
		opts := &missingDecodeHookOptions{}
		cmd := &cobra.Command{Use: "test"}

		err := Define(cmd, opts)

		require.Error(t, err)

		// Should be wrapped in a MissingDecodeHookError
		var missingErr *autoflagserrors.MissingDecodeHookError
		assert.True(t, errors.As(err, &missingErr), "Should be MissingDecodeHookError type")
		if missingErr != nil {
			assert.Equal(t, "CustomField", missingErr.FieldName)
			assert.Equal(t, "DecodeCustomField", missingErr.ExpectedHook)
		}
	})

	suite.T().Run("wrong_define_signature_error_type", func(t *testing.T) {
		opts := &wrongDefineSignatureOptions{}
		cmd := &cobra.Command{Use: "test"}

		err := Define(cmd, opts)

		require.Error(t, err)

		// Should be wrapped in an InvalidDefineHookSignatureError
		var defineErr *autoflagserrors.InvalidDefineHookSignatureError
		assert.True(t, errors.As(err, &defineErr), "Should be InvalidDefineHookSignatureError type")
		if defineErr != nil {
			assert.Equal(t, "CustomField", defineErr.FieldName)
			assert.Equal(t, "DefineCustomField", defineErr.HookName)
		}
	})

	suite.T().Run("wrong_decode_signature_error_type", func(t *testing.T) {
		opts := &wrongDecodeSignatureOptions{}
		cmd := &cobra.Command{Use: "test"}

		err := Define(cmd, opts)

		require.Error(t, err)

		// Should be wrapped in an InvalidDecodeHookSignatureError
		var decodeErr *autoflagserrors.InvalidDecodeHookSignatureError
		assert.True(t, errors.As(err, &decodeErr), "Should be InvalidDecodeHookSignatureError type")
		if decodeErr != nil {
			assert.Equal(t, "CustomField", decodeErr.FieldName)
			assert.Equal(t, "DecodeCustomField", decodeErr.HookName)
		}
	})
}

type multipleCustomOptions struct {
	GoodField string `flagcustom:"true" flag:"good-field"`
	BadField  string `flagcustom:"true" flag:"bad-field"`
}

func (o *multipleCustomOptions) DefineGoodField(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "default", descr)
}

func (o *multipleCustomOptions) DecodeGoodField(input any) (any, error) {
	return input, nil
}

func (o *multipleCustomOptions) DefineBadField(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "default", descr)
}

func (o *multipleCustomOptions) Attach(c *cobra.Command) {}

type nestedCustomStruct struct {
	CustomField string `flagcustom:"true" flag:"nested-custom"`
}

func (n *nestedCustomStruct) DefineCustomField(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	c.Flags().String(name, "nested-default", descr)
}

func (n *nestedCustomStruct) DecodeCustomField(input any) (any, error) {
	return input, nil
}

type parentOptions struct {
	Nested nestedCustomStruct
}

func (o *parentOptions) Attach(c *cobra.Command) {}

func (suite *autoflagsSuite) TestValidateCustomFlag_EdgeCases() {
	suite.T().Run("multiple_custom_fields", func(t *testing.T) {
		// Test struct with multiple custom fields where one is wrong

		opts := &multipleCustomOptions{}
		cmd := &cobra.Command{Use: "test"}

		err := Define(cmd, opts)

		require.Error(t, err, "Should fail when one custom field is missing decode hook")
		assert.Contains(t, err.Error(), "BadField", "Should mention the problematic field")
		assert.Contains(t, err.Error(), "DecodeBadField", "Should mention the missing decode hook")
	})

	suite.T().Run("nested_struct_with_custom_field", func(t *testing.T) {
		// Test nested struct containing custom field

		opts := &parentOptions{}
		cmd := &cobra.Command{Use: "test"}

		err := Define(cmd, opts)

		assert.NoError(t, err, "Should handle nested struct with custom field correctly")

		flag := cmd.Flags().Lookup("nested-custom")
		assert.NotNil(t, flag, "Nested custom flag should be created")
	})
}
