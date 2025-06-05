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

	return nil
}

// UseDebug manually triggers debug output for the given options.
//
// Debug output is automatically triggered when the debug flag is enabled.
func UseDebug(c *cobra.Command, w io.Writer) {
	flagName := "debug-options"
	if currentFlagName, ok := c.Annotations[flagDebugAnnotation]; ok {
		flagName = currentFlagName
	}

	v := GetViper(c)
	if !v.GetBool(flagName) {
		return
	}

	var dest io.Writer
	dest = os.Stdout
	if w != nil {
		dest = w
	}

	v.DebugTo(dest)
	fmt.Fprintf(dest, "Values:\n%#v\n", v.AllSettings())
}
