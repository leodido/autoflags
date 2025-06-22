package internalcmd

import (
	internaldebug "github.com/leodido/structcli/internal/debug"
	"github.com/spf13/cobra"
)

func RecursivelyWrapRun(c *cobra.Command) {
	if c.RunE != nil {
		originalRunE := c.RunE
		c.RunE = func(c *cobra.Command, args []string) error {
			if internaldebug.IsDebugActive(c) {
				return nil // Exit cleanly without running the original function
			}
			return originalRunE(c, args)
		}
	} else if c.Run != nil {
		// Handle non-error returning Run as well
		originalRun := c.Run
		c.Run = func(c *cobra.Command, args []string) {
			if internaldebug.IsDebugActive(c) {
				return // Exit cleanly
			}
			originalRun(c, args)
		}
	}

	// Recurse into subcommands
	for _, sub := range c.Commands() {
		RecursivelyWrapRun(sub)
	}
}
