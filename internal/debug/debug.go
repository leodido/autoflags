package internaldebug

import (
	internalscope "github.com/leodido/autoflags/internal/scope"
	"github.com/spf13/cobra"
)

const (
	FlagAnnotation = "___leodido_autoflags_debugflagname"
)

// IsDebugActive checks if the debug option is set for the command c, either through a command-line flag or an environment variable.
func IsDebugActive(c *cobra.Command) bool {
	debugFlagName := "debug-options"
	if currentFlagName, ok := c.Annotations[FlagAnnotation]; ok {
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
		rootS := internalscope.Get(rootC)
		rootV := rootS.Viper()
		if rootV.GetBool(debugFlagName) {
			isActive = true
		}
	}

	return isActive
}
