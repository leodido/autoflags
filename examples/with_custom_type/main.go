package main

import (
	"fmt"

	"github.com/leodido/autoflags"
	"github.com/spf13/cobra"
)

type ServerMode string

const (
	Development ServerMode = "dev"
	Staging     ServerMode = "staging"
	Production  ServerMode = "prod"
)

type AppOpts struct {
	Mode ServerMode `flagcustom:"true" flag:"server-mode" flagshort:"m" flagdescr:"Set server mode"`
	Port int        `flag:"port" flagdescr:"Server port" default:"8080"`
}

// Custom flag definition method
func (o *AppOpts) DefineMode(c *cobra.Command, typename string, name, short, descr string) {
	// TODO: complete
	description := descr + fmt.Sprintf(" (%s,%s,%s)", Development, Staging, Production)
	fmt.Println(description)
	// c.Flags().StringVarP(name, short, string(Development), description)

	// // Add shell completion
	// c.RegisterFlagCompletionFunc(name, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// 	return []string{string(Development), string(Staging), string(Production)}, cobra.ShellCompDirectiveDefault
	// })
}

func (o *AppOpts) Attach(c *cobra.Command) {
	autoflags.Define(c, o)
}

func main() {
	opts := &AppOpts{}
	cli := &cobra.Command{Use: "myapp"}

	// This single line creates all the options (flags, env vars)
	opts.Attach(cli)

	cli.PreRunE = func(c *cobra.Command, args []string) error {
		return autoflags.Unmarshal(c, opts) // Populates struct from flags/env
	}

	cli.RunE = func(c *cobra.Command, args []string) error {
		fmt.Println(opts)

		return nil
	}

	cli.Execute()
}
