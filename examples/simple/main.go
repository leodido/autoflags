package main

import (
	"fmt"

	"github.com/leodido/autoflags"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

type Options struct {
	LogLevel zapcore.Level `flag:"level" flagdescr:"Set logging level" flagenv:"true"`
	Port     int           `flagshort:"p" flagdescr:"Server port" flagenv:"true" default:"3000"`
}

func (o *Options) Attach(c *cobra.Command) {
	autoflags.Define(c, o)
}

func main() {
	opts := &Options{}
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
