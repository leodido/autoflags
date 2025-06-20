package internalusage

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// flagUsages generates a string containing the usage information for a set of flags.
//
// It trims trailing whitespace from the final output.
func flagUsages(f *pflag.FlagSet) string {
	return strings.TrimRight(f.FlagUsages(), " \n") + "\n"
}

// rpad adds padding to the right of a string.
func rpad(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(template, s)
}

// tmpl is a helper function that writes a string to the provided writer.
func tmpl(w io.Writer, text string) error {
	_, err := w.Write([]byte(text))
	return err
}

// Setup generates and sets a dynamic usage function for the command.
//
// It also groups flags based on the `flaggroup` annotation.
func Setup(c *cobra.Command) {
	c.SetUsageFunc(func(c *cobra.Command) error {
		var b strings.Builder

		// Usage Line
		b.WriteString("Usage:")
		if c.Runnable() {
			b.WriteString("\n  ")
			b.WriteString(c.UseLine())
		}
		if c.HasAvailableSubCommands() {
			b.WriteString("\n  ")
			b.WriteString(c.CommandPath())
			b.WriteString(" [command]")
		}
		b.WriteString("\n")

		// Aliases
		if c.HasAvailableSubCommands() && len(c.Aliases) > 0 {
			b.WriteString("\nAliases:\n  ")
			b.WriteString(c.NameAndAliases())
			b.WriteString("\n")
		}

		// Examples
		if len(c.Example) > 0 {
			b.WriteString("\nExamples:\n")
			b.WriteString(c.Example)
			b.WriteString("\n")
		}

		// Available Commands
		if c.HasAvailableSubCommands() {
			b.WriteString("\nAvailable Commands:\n")
			for _, cmd := range c.Commands() {
				if !cmd.IsAvailableCommand() && cmd.Name() != "help" {
					continue
				}
				b.WriteString(fmt.Sprintf("  %s %s\n", rpad(cmd.Name(), c.NamePadding()), cmd.Short))
			}
		}

		// Local and grouped flags
		groups := Groups(c)

		// Print default "Flags" group first, if it exists
		if lFlags, ok := groups[localGroupID]; ok && lFlags.HasFlags() {
			b.WriteString("\nFlags:\n")
			b.WriteString(flagUsages(lFlags))
			delete(groups, localGroupID)
		}

		// Then print all other custom groups
		groupKeys := make([]string, 0, len(groups))
		for k := range groups {
			groupKeys = append(groupKeys, k)
		}
		sort.Strings(groupKeys)

		for _, groupName := range groupKeys {
			if groupName == globalGroupID {
				continue // Handle global flags last
			}
			flags := groups[groupName]
			if flags.HasFlags() {
				b.WriteString(fmt.Sprintf("\n%s Flags:\n", groupName))
				b.WriteString(flagUsages(flags))
			}
		}

		// Now, print the Global flags which were collected by the Groups() function
		if gFlags, ok := groups[globalGroupID]; ok && gFlags.HasFlags() {
			b.WriteString("\nGlobal Flags:\n")
			b.WriteString(flagUsages(gFlags))
		}

		// Help Topics and "use" command
		if c.HasHelpSubCommands() {
			b.WriteString("\nAdditional help topics:\n")
			for _, cmd := range c.Commands() {
				if cmd.IsAdditionalHelpTopicCommand() {
					b.WriteString(fmt.Sprintf("  %s %s\n", rpad(cmd.CommandPath(), c.CommandPathPadding()), cmd.Short))
				}
			}
		}

		if c.HasAvailableSubCommands() {
			b.WriteString(fmt.Sprintf("\nUse \"%s [command] --help\" for more information about a command.\n", c.CommandPath()))
		}

		return tmpl(c.OutOrStderr(), b.String())
	})
}
