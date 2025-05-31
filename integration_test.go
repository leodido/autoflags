package autoflags_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/go-playground/mold/v4"
	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/validator/v10"
	"github.com/leodido/autoflags"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Package-level variables to hold the initialized instances
var (
	testMolder    *mold.Transformer
	testValidator *validator.Validate
)

// TestMain sets up the molder and validator instances once for all tests in this package.
func TestMain(m *testing.M) {
	// Setup
	testMolder = modifiers.New()
	testValidator = validator.New()

	// Register custom validation functions or tags here
	// testValidator.RegisterValidation("my_custom_tag", myCustomValidationFunc)

	// Run all tests in the package
	exitCode := m.Run()

	// Teardown (if necessary, though not typically for molder/validator instances)
	os.Exit(exitCode)
}

type unmarshalIntegrationOptions struct {
	Name                 string `flag:"name" mod:"trim"`
	Email                string `flag:"email" mod:"trim,lcase" validate:"required,email"`
	Age                  int    `flag:"age" validate:"min=18,max=120"`
	Status               string `flag:"status" mod:"default=active" validate:"required,oneof=active inactive pending"`
	Justification        string `flag:"justification" validate:"required_if=Status pending"`
	SimulatePreMoldError bool
}

// Attach (definition remains the same)
func (o *unmarshalIntegrationOptions) Attach(c *cobra.Command) {
	c.Flags().StringVar(&o.Name, "name", "", "User's name")
	c.Flags().StringVar(&o.Email, "email", "", "User's email address")
	c.Flags().IntVar(&o.Age, "age", 0, "User's age")
	c.Flags().StringVar(&o.Status, "status", "", "User's status (active, inactive, pending)")
	c.Flags().StringVar(&o.Justification, "justification", "", "Justification if status is pending")
}

func (o *unmarshalIntegrationOptions) Transform(ctx context.Context) error {
	if o.SimulatePreMoldError {
		return errors.New("simulated pre-mold transformation error")
	}
	err := testMolder.Struct(ctx, o)
	if err != nil {
		return fmt.Errorf("mold transformation failed: %w", err)
	}
	return nil
}

func (o *unmarshalIntegrationOptions) Validate() []error {
	var errs []error
	err := testValidator.Struct(o)
	if err != nil {
		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			for _, fieldErr := range validationErrs {
				errs = append(errs, fieldErr)
			}
		} else {
			errs = append(errs, fmt.Errorf("validator.Struct() failed unexpectedly: %w", err))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func TestUnmarshal_Integration_WithLibraries(t *testing.T) {
	setupTest := func() {
		viper.Reset()
	}

	t.Run("PreMoldTransformationFails", func(t *testing.T) {
		setupTest()
		cmd := &cobra.Command{Use: "testcmd-premoldfail"}
		opts := &unmarshalIntegrationOptions{
			SimulatePreMoldError: true,
		}
		errDefine := autoflags.Define(cmd, opts)
		require.NoError(t, errDefine)

		err := autoflags.Unmarshal(cmd, opts)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "couldn't transform options:")
		assert.Contains(t, err.Error(), "simulated pre-mold transformation error")
	})

	t.Run("ValidationFails_InvalidEmail_AfterMold", func(t *testing.T) {
		setupTest()
		cmd := &cobra.Command{Use: "testcmd-emailfail"}
		opts := &unmarshalIntegrationOptions{}

		errDefine := autoflags.Define(cmd, opts)
		require.NoError(t, errDefine)

		viper.Set("email", "  NOTANEMAIL@domain  ")
		viper.Set("age", 25)

		err := autoflags.Unmarshal(cmd, opts)

		assert.Error(t, err, "Unmarshal should return an error for invalid email format")
		assert.Contains(t, err.Error(), "invalid options")
		assert.Contains(t, err.Error(), "Error:Field validation for 'Email' failed on the 'email' tag")
		assert.Equal(t, "notanemail@domain", opts.Email)
		assert.Equal(t, "active", opts.Status)
	})

	t.Run("ValidationFails_InvalidAge", func(t *testing.T) {
		setupTest()
		cmd := &cobra.Command{Use: "testcmd-agefail"}
		opts := &unmarshalIntegrationOptions{}

		errDefine := autoflags.Define(cmd, opts)
		require.NoError(t, errDefine)

		viper.Set("email", "valid@example.com")
		viper.Set("age", 5)

		err := autoflags.Unmarshal(cmd, opts)

		assert.Error(t, err, "Unmarshal should return an error for invalid age")
		assert.Contains(t, err.Error(), "invalid options")
		assert.Contains(t, err.Error(), "Error:Field validation for 'Age' failed on the 'min' tag")
	})

	t.Run("ValidationFails_RequiredIf_Justification", func(t *testing.T) {
		setupTest()
		cmd := &cobra.Command{Use: "testcmd-reqif-fail"}
		opts := &unmarshalIntegrationOptions{}

		errDefine := autoflags.Define(cmd, opts)
		require.NoError(t, errDefine)

		viper.Set("email", "valid@example.com")
		viper.Set("age", 30)
		viper.Set("status", "pending")
		viper.Set("justification", "")

		err := autoflags.Unmarshal(cmd, opts)

		assert.Error(t, err, "Unmarshal should return an error if Justification is missing when Status is pending")
		assert.Contains(t, err.Error(), "invalid options")
		assert.Contains(t, err.Error(), "Error:Field validation for 'Justification' failed on the 'required_if' tag")
	})

	t.Run("Success_WithMoldAndValidator", func(t *testing.T) {
		setupTest()
		cmd := &cobra.Command{Use: "testcmd-success-libs"}
		opts := &unmarshalIntegrationOptions{}

		errDefine := autoflags.Define(cmd, opts)
		require.NoError(t, errDefine)

		viper.Set("name", "  Test User  ")
		viper.Set("email", "  USER.TEST@Example.COM  ")
		viper.Set("age", 42)
		viper.Set("status", "inactive")

		err := autoflags.Unmarshal(cmd, opts)

		assert.NoError(t, err, "Unmarshal should succeed")
		assert.Equal(t, "Test User", opts.Name)
		assert.Equal(t, "user.test@example.com", opts.Email)
		assert.Equal(t, 42, opts.Age)
		assert.Equal(t, "inactive", opts.Status)
	})
}
