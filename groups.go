package autoflags

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	localGroupID  = "<local>"
	globalGroupID = "Global"
)

const (
	flagGroupAnnotation = "___leodido_autoflags_flaggroups"
)

// Groups returns a map of flag groups for the given command.
//
// It organizes flags by their group annotation, with ungrouped flags placed in a default local group.
func Groups(c *cobra.Command) map[string]*pflag.FlagSet {
	groups := map[string]*pflag.FlagSet{
		"<origin>": c.LocalFlags(),
	}
	delete(groups, "<origin>")

	addTo := func(f *pflag.Flag, groupID string) {
		if groups[groupID] == nil {
			groups[groupID] = pflag.NewFlagSet(c.Name(), pflag.ContinueOnError)
		}
		groups[groupID].AddFlag(f)
	}

	c.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
		if len(f.Annotations) == 0 {
			addTo(f, localGroupID)
		} else {
			if annotations, ok := f.Annotations[flagGroupAnnotation]; ok {
				g := annotations[0]
				if groups[g] == nil {
					groups[g] = pflag.NewFlagSet(c.Name(), pflag.ContinueOnError)
				}
				groups[g].AddFlag(f)
			} else {
				addTo(f, localGroupID)
			}
		}
	})

	if c.HasPersistentFlags() {
		c.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			addTo(f, globalGroupID)
		})
	}

	return groups
}
