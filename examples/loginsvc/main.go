package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/leodido/autoflags"
	"github.com/leodido/autoflags/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"
)

// -------------------------------------------------
// 1. Shared Options and Computed State
// -------------------------------------------------

// CommonOptions holds the global configuration and the computed state (Logger).
type CommonOptions struct {
	LogLevel zapcore.Level `flag:"loglevel" flagdescr:"Logging level (debug, info, error)" default:"info" flagenv:"true"`
	// The Logger is our "computed state". It's not a flag itself, but it's initialized based on the LogLevel flag.
	Logger *zap.Logger `flagignore:"true"`
}

type commonOptionsKey struct{}

// Attach is a convenience wrapper around autoflags.Define.
func (o *CommonOptions) Attach(c *cobra.Command) error {
	return autoflags.Define(c, o)
}

// Context implements the "setter" part of the ContextOptions contract.
// It injects the populated and initialized object into the context.
func (o *CommonOptions) Context(ctx context.Context) context.Context {
	return context.WithValue(ctx, commonOptionsKey{}, o)
}

// FromContext implements the "getter" part of the contract.
// It retrieves the shared object from the context.
func (o *CommonOptions) FromContext(ctx context.Context) error {
	value, ok := ctx.Value(commonOptionsKey{}).(*CommonOptions)
	if !ok {
		return fmt.Errorf("CommonOptions not found in context")
	}
	*o = *value

	return nil
}

// Initialize creates the computed state (the logger).
func (o *CommonOptions) Initialize(c *cobra.Command) error {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.MessageKey = "M"
	encoderCfg.LevelKey = "L"
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(c.OutOrStdout()),
		o.LogLevel,
	)
	logger := zap.New(core)
	if logger == nil {
		return fmt.Errorf("could not initialize logger")
	}
	o.Logger = logger

	return nil
}

// -------------------------------------------------
// 2. Local Options for Subcommands
// -------------------------------------------------

type AddUserOptions struct {
	Username string `flag:"username" flagshort:"u" flagdescr:"Username of the new user" flagrequired:"true"`
	Password string `flagignore:"true"`
}

func (o *AddUserOptions) Attach(c *cobra.Command) error {
	return autoflags.Define(c, o)
}

type DeleteUserOptions struct {
	Username string `flag:"username" flagshort:"u" flagdescr:"Username of the user to delete" flagrequired:"true"`
}

func (o *DeleteUserOptions) Attach(c *cobra.Command) error {
	return autoflags.Define(c, o)
}

// -------------------------------------------------
// 3. CLI Construction
// -------------------------------------------------

func readPassword(cmd *cobra.Command) (string, error) {
	if term.IsTerminal(int(syscall.Stdin)) {
		fmt.Fprint(cmd.OutOrStdout(), "Enter Password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(cmd.OutOrStdout())
		if err != nil {
			return "", fmt.Errorf("could not read password: %w", err)
		}
		return string(bytePassword), nil
	}
	reader := bufio.NewReader(cmd.InOrStdin())
	password, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("could not read password from pipe: %w", err)
	}

	return strings.TrimSpace(password), nil
}

func makeUserAddCmd() *cobra.Command {
	commonOpts := &CommonOptions{} // Receiver for shared state.
	opts := &AddUserOptions{}      // Receiver for local flags.

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Adds a new user",
		RunE: func(c *cobra.Command, args []string) error {
			// Step 1: Retrieve the master, initialized state from the context.
			if err := commonOpts.FromContext(c.Context()); err != nil {
				return err
			}

			// Step 2: Unmarshal local flags for this command.
			if err := autoflags.Unmarshal(c, opts); err != nil {
				return err
			}

			// Step 3: Handle secure input.
			password, err := readPassword(c)
			if err != nil || password == "" {
				return fmt.Errorf("password cannot be empty")
			}
			opts.Password = password

			// Step 4: Use both the shared logger and local options.
			c.Printf("level:%s\n", commonOpts.LogLevel.String())
			commonOpts.Logger.Info("Attempting to add user", zap.String("user", opts.Username))
			fmt.Fprintf(c.OutOrStdout(), "Added user '%s' with the provided password.\n", opts.Username)

			return nil
		},
	}
	// Define the flags specific to this command.
	opts.Attach(addCmd)
	commonOpts.Attach(addCmd)

	return addCmd
}

func makeUserDeleteCmd() *cobra.Command {
	commonOpts := &CommonOptions{} // Receiver for shared state.
	opts := &DeleteUserOptions{}

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes a user",
		RunE: func(c *cobra.Command, args []string) error {
			if err := commonOpts.FromContext(c.Context()); err != nil {
				return err
			}
			if err := autoflags.Unmarshal(c, opts); err != nil {
				return err
			}
			c.Printf("level:%s\n", commonOpts.LogLevel.String())
			commonOpts.Logger.Warn("Attempting to delete user", zap.String("user", opts.Username))
			fmt.Fprintf(c.OutOrStdout(), "Deleted user '%s'.\n", opts.Username)

			return nil
		},
	}
	opts.Attach(deleteCmd)
	commonOpts.Attach(deleteCmd)

	return deleteCmd
}

func makeUserCmd() *cobra.Command {
	// This command groups `add` and `delete`.
	// It also needs the shared flags defined on it so that `loginsvc user --loglevel debug ...` works.
	// configOpts := &CommonOptions{}
	userCmd := &cobra.Command{
		Use:   "user",
		Short: "Manages users",
	}

	// Attach the shared options here to solve the flag parsing problem.
	// configOpts.Attach(userCmd)
	userCmd.AddCommand(makeUserAddCmd())
	userCmd.AddCommand(makeUserDeleteCmd())

	return userCmd
}

func NewRootCmd() (*cobra.Command, error) {
	// This is the "master" instance that will become the single source of truth.
	commonOpts := &CommonOptions{}

	rootCmd := &cobra.Command{Use: "loginsvc"}

	// Attach the shared options to the root command as well.
	commonOpts.Attach(rootCmd)

	// This hook runs for ALL command invocations after parsing but before execution.
	rootCmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		_, configMessage, configErr := autoflags.UseConfigSimple(c)
		if configErr != nil {
			return configErr
		}
		if configMessage != "" {
			c.Println(configMessage)
		}
		// Populate the master `commonOpts` from flags, env, and config file.
		if err := autoflags.Unmarshal(c, commonOpts); err != nil {
			return err
		}
		// Use the populated values to initialize the computed state (the logger).
		if err := commonOpts.Initialize(c); err != nil {
			return err
		}
		// `Unmarshal` has already called `commonOpts.Context()` at this point, injecting our fully initialized master object into the context.

		return nil
	}

	rootCmd.AddCommand(makeUserCmd())

	if err := autoflags.SetupConfig(rootCmd, config.Options{}); err != nil {
		return nil, err
	}
	return rootCmd, nil
}

func main() {
	cmd, err := NewRootCmd()
	if err != nil {
		log.Fatalf("Error creating command: %v", err)
	}
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
