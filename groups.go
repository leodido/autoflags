package autoflags

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	localGroupID = "<local>"
)

const (
	FlagGroupAnnotation = "___flaggroup"
)

func Groups(c *cobra.Command) map[string]*pflag.FlagSet {
	localGroupID := "<local>"
	groups := map[string]*pflag.FlagSet{
		"<origin>": c.LocalFlags(),
	}
	delete(groups, "<origin>")

	addToLocal := func(f *pflag.Flag) {
		if groups[localGroupID] == nil {
			groups[localGroupID] = pflag.NewFlagSet(c.Name(), pflag.ContinueOnError)
		}
		groups[localGroupID].AddFlag(f)
	}

	c.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if len(f.Annotations) == 0 {
			addToLocal(f)
		} else {
			if annotations, ok := f.Annotations[FlagGroupAnnotation]; ok {
				g := annotations[0]
				if groups[g] == nil {
					groups[g] = pflag.NewFlagSet(c.Name(), pflag.ContinueOnError)
				}
				groups[g].AddFlag(f)
			} else {
				addToLocal(f)
			}
		}
	})

	return groups
}
