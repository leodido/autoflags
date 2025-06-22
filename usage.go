package structcli

import (
	internalusage "github.com/leodido/structcli/internal/usage"
	"github.com/spf13/cobra"
)

// SetupUsage generates and sets a dynamic usage function for the command.
//
// It also groups flags based on the `flaggroup` annotation.
func SetupUsage(c *cobra.Command) {
	internalusage.Setup(c)
}
