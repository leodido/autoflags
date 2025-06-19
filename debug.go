package autoflags

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// DebugOptions configures the debug functionality for command-line applications.
type DebugOptions struct {
	AppName  string
	FlagName string // Name of debug flag (defaults to "debug-options")
	EnvVar   string // Environment variable (defaults to {APP}_DEBUG_OPTIONS)
	Exit     bool   // Forces the CLI to exit after printing the debug information without executing the command's RunE
}

const (
	flagDebugAnnotation = "___leodido_autoflags_debugflagname"
)

// SetupDebug creates the --debug-options global flag and sets up debug behavior.
//
// Works only for the root command.
func SetupDebug(rootC *cobra.Command, debugOpts DebugOptions) error {
	if rootC.Parent() != nil {
		return fmt.Errorf("SetupDebug must be called on the root command")
	}

	// Determine app name from root command
	appName := GetOrSetAppName(debugOpts.AppName, rootC.Name())
	if appName == "" {
		return fmt.Errorf("couldn't determine the app name")
	}

	// Compute flag and environment variable names
	flagName := debugOpts.FlagName
	if flagName == "" {
		flagName = "debug-options"
	}
	envvName := normEnv(debugOpts.EnvVar)
	if debugOpts.EnvVar == "" {
		normFlagName := normEnv(flagName)
		if currentPrefix := EnvPrefix(); currentPrefix != "" {
			envvName = fmt.Sprintf("%s_%s", currentPrefix, normFlagName)
		} else {
			envvName = fmt.Sprintf("%s_%s", normEnv(appName), normFlagName)
		}
	}

	// Store the actual debug options flag name in the root command annotations
	if rootC.Annotations == nil {
		rootC.Annotations = make(map[string]string)
	}
	rootC.Annotations[flagDebugAnnotation] = flagName

	// Add persistent flag to root command
	rootC.PersistentFlags().Bool(flagName, false, "enable debug output for options")

	// Add environment annotation
	rootC.PersistentFlags().SetAnnotation(flagName, flagEnvsAnnotation, []string{envvName})

	// Ensure environment binding happens
	cobra.OnInitialize(func() {
		bindEnv(rootC)
	})

	// Wrap all commands run hooks
	if debugOpts.Exit {
		recursiveWrapC(rootC)
	}

	// Regenerate usage templates for any commands already processed by Define()
	SetupUsage(rootC)

	return nil
}

func recursiveWrapC(c *cobra.Command) {
	if c.RunE != nil {
		originalRunE := c.RunE
		c.RunE = func(c *cobra.Command, args []string) error {
			if IsDebugActive(c) {
				return nil // Exit cleanly without running the original function
			}
			return originalRunE(c, args)
		}
	} else if c.Run != nil {
		// Handle non-error returning Run as well
		originalRun := c.Run
		c.Run = func(c *cobra.Command, args []string) {
			if IsDebugActive(c) {
				return // Exit cleanly
			}
			originalRun(c, args)
		}
	}

	// Recurse into subcommands
	for _, sub := range c.Commands() {
		recursiveWrapC(sub)
	}
}

// IsDebugActive checks if the debug option is set for the command c, either through a command-line flag or an environment variable.
func IsDebugActive(c *cobra.Command) bool {
	debugFlagName := "debug-options"
	if currentFlagName, ok := c.Annotations[flagDebugAnnotation]; ok {
		debugFlagName = currentFlagName
	}

	isActive := false
	rootC := c.Root()

	// Let's first check the flag directly
	if debugFlag := rootC.PersistentFlags().Lookup(debugFlagName); debugFlag != nil {
		if debugFlag.Changed {
			isActive = true
		}
	}

	// Check viper for other sources (eg, environment variable)
	if !isActive {
		rootV := GetViper(rootC)
		if rootV.GetBool(debugFlagName) {
			isActive = true
		}
	}

	return isActive
}

// UseDebug manually triggers debug output for the given options.
//
// Debug output is automatically triggered when the debug flag is enabled.
func UseDebug(c *cobra.Command, w io.Writer) {
	if !IsDebugActive(c) {
		return
	}

	var dest io.Writer
	dest = os.Stdout
	if w != nil {
		dest = w
	}

	// The action of printing debug info is local
	v := GetViper(c)
	v.DebugTo(dest)
	fmt.Fprintf(dest, "Values:\n%#v\n", v.AllSettings())
}
