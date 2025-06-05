package main

import (
	"fmt"

	"github.com/leodido/autoflags"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

type ServerOptions struct {
	// Basic flags
	Host string `flag:"host" flagdescr:"Server host" default:"localhost"`
	Port int    `flag:"port" flagshort:"p" flagdescr:"Server port" flagrequired:"true"`

	// Environment variable binding
	APIKey string `flagenv:"true" flagdescr:"API authentication key"`

	// Flag grouping for organized help
	LogLevel zapcore.Level `flag:"log-level" flagcustom:"true" flaggroup:"Logging" flagdescr:"Set log level"`
	LogFile  string        `flag:"log-file" flaggroup:"Logging" flagdescr:"Log file path"`

	// Nested structs for organization
	Database DatabaseConfig `flaggroup:"Database"`
}

type DatabaseConfig struct {
	URL      string `flag:"db-url" flagdescr:"Database connection URL"`
	MaxConns int    `flag:"db-max-conns" flagdescr:"Max database connections" default:"10" flagenv:"true"`
}

func (o *ServerOptions) Attach(c *cobra.Command) {
	autoflags.Define(c, o)
}

func main() {
	opts := &ServerOptions{}
	cli := &cobra.Command{Use: "mysrv"}

	cli.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		// Load config file if found
		if _, _, err := autoflags.UseConfigSimple(c); err != nil {
			return err
		}

		return nil
	}

	cli.PreRunE = func(c *cobra.Command, args []string) error {
		return autoflags.Unmarshal(c, opts) // Unmarshal all configuration (flags override config file)
	}

	cli.RunE = func(c *cobra.Command, args []string) error {
		fmt.Println(opts)

		return nil
	}

	// This single line creates all the options (flags, env vars)
	opts.Attach(cli)

	// This single line enables the configuration file support
	autoflags.SetupConfig(cli, autoflags.ConfigOptions{AppName: "mysrv"})

	cli.Execute()
}
