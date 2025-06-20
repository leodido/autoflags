package autoflags

import (
	"fmt"
	"io"
	"os"

	"github.com/leodido/autoflags/debug"
	internalcmd "github.com/leodido/autoflags/internal/cmd"
	internaldebug "github.com/leodido/autoflags/internal/debug"
	internalenv "github.com/leodido/autoflags/internal/env"
	"github.com/spf13/cobra"
)

// SetupDebug creates the --debug-options global flag and sets up debug behavior.
//
// Works only for the root command.
func SetupDebug(rootC *cobra.Command, debugOpts debug.Options) error {
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
	envvName := internalenv.NormEnv(debugOpts.EnvVar)
	if debugOpts.EnvVar == "" {
		normFlagName := internalenv.NormEnv(flagName)
		if currentPrefix := EnvPrefix(); currentPrefix != "" {
			envvName = fmt.Sprintf("%s_%s", currentPrefix, normFlagName)
		} else {
			envvName = fmt.Sprintf("%s_%s", internalenv.NormEnv(appName), normFlagName)
		}
	}

	// Store the actual debug options flag name in the root command annotations
	if rootC.Annotations == nil {
		rootC.Annotations = make(map[string]string)
	}
	rootC.Annotations[internaldebug.FlagAnnotation] = flagName

	// Add persistent flag to root command
	rootC.PersistentFlags().Bool(flagName, false, "enable debug output for options")

	// Add environment annotation
	rootC.PersistentFlags().SetAnnotation(flagName, internalenv.FlagAnnotation, []string{envvName})

	// Ensure environment binding happens
	cobra.OnInitialize(func() {
		internalenv.BindEnv(rootC)
	})

	// Wrap all commands run hooks
	if debugOpts.Exit {
		internalcmd.RecursivelyWrapRun(rootC)
	}

	// Regenerate usage templates for any commands already processed by Define()
	SetupUsage(rootC)

	return nil
}

// IsDebugActive checks if the debug option is set for the command c, either through a command-line flag or an environment variable.
func IsDebugActive(c *cobra.Command) bool {
	return internaldebug.IsDebugActive(c)
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
