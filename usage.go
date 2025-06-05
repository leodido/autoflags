package autoflags

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

const (
	usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

%s{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
	noFlagsTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
	cUsageRegenAnnotation = "___leodido_autoflags_c_usage_regen"
)

// setUsage generates the flag usages of the flags local to the input command.
//
// It also groups the flags by the FlagGroupAnnotation annotation.
func setUsage(c *cobra.Command) {
	groups := Groups(c)

	usages := ""
	if lFlags, ok := groups[localGroupID]; ok {
		usages += "Flags:\n"
		usages += lFlags.FlagUsages()
		delete(groups, localGroupID)
	}

	groupKeys := maps.Keys(groups)
	sort.Strings(groupKeys)

	for _, group := range groupKeys {
		flags := groups[group]
		if usages != "" {
			usages += "\n"
		}
		usages += fmt.Sprintf("%s Flags:\n", group)
		usages += flags.FlagUsages()
	}
	usages = strings.TrimSuffix(usages, "\n")

	s := fmt.Sprintf(usageTemplate, usages)
	if usages == "" {
		s = noFlagsTemplate
	}

	c.SetUsageTemplate(s)
}

// markForUsageRegeneration marks a command as needing usage regeneration
// if persistent flags are added later by setup functions
func markForUsageRegeneration(c *cobra.Command) {
	if c.Annotations == nil {
		c.Annotations = make(map[string]string)
	}
	c.Annotations[cUsageRegenAnnotation] = "true"
}

// regenerateUsage regenerates usage templates for the commands if it has been marked as needing regeneration by Define().
//
// It also does the same for child commands.
func regenerateUsage(c *cobra.Command) {
	// Check if this command was marked for regeneration
	if c.Annotations != nil && c.Annotations[cUsageRegenAnnotation] == "true" {
		setUsage(c)
		// Clean up the marker since regeneration is complete
		delete(c.Annotations, cUsageRegenAnnotation)
	}

	// Recursively check all subcommands
	for _, subCmd := range c.Commands() {
		regenerateUsage(subCmd)
	}
}
