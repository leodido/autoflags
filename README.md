# autoflags

> CLI generation from Go structs

Transform your Go structs into fully-featured command-line interfaces with configuration files, environment variables, flags, validation, and beautiful help output.

All with just a few struct tags: **declare your options in a struct, and let autoflags do the rest**.

Stop writing boilerplate. Start building features. ⋙

## ⚡ Quick Start

```go
package main

import (
	"fmt"
	"log"

	"github.com/leodido/autoflags"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

type Options struct {
	LogLevel zapcore.Level
	Port     int
}

func (o *Options) Attach(c *cobra.Command) error {
	return autoflags.Define(c, o)
}

func main() {
	log.SetFlags(0)
	opts := &Options{}
	cli := &cobra.Command{Use: "myapp"}

	// This single line creates all the options (flags, env vars)
	if err := opts.Attach(cli); err != nil {
		log.Fatalln(err)
	}

	cli.PreRunE = func(c *cobra.Command, args []string) error {
		return autoflags.Unmarshal(c, opts) // Populates struct from flags
	}

	cli.RunE = func(c *cobra.Command, args []string) error {
		fmt.Println(opts)

		return nil
	}

	if err := cli.Execute(); err != nil {
		log.Fatalln(err)
	}
}
```

**That's it**!

```bash
❯ go run examples/minimal/main.go --help
# Usage:
#   myapp [flags]
#
# Flags:
#       --loglevel zapcore.Level    {debug,info,warn,error,dpanic,panic,fatal} (default info)
#       --port int
```

Want automatic environment variables, aliases, shorthand, flag description in usage?

Just annotate your struct with `autoflags` tags.

```go
type Options struct {
	LogLevel zapcore.Level `flag:"level" flagdescr:"Set logging level" flagenv:"true"`
	Port     int           `flagshort:"p" flagdescr:"Server port" flagenv:"true" default:"3000"`
}
```

Here it is.

```bash
❯ go run examples/simple/main.go -h
# Usage:
#   myapp [flags]
#
# Flags:
#       --level zapcore.Level   Set logging level {debug,info,warn,error,dpanic,panic,fatal} (default info)
#   -p, --port int              Server port (default 3000)
```

You got your environment variables.

```bash
❯ MYAPP_LOGLEVEL=debug go run examples/simple/main.go
# &{debug 3000}
```

```bash
❯ MYAPP_LEVEL=warn go run examples/simple/main.go
# &{warn 3000}
```

Flags override environment variables, of course.

```bash
❯ MYAPP_LOGLEVEL=error MYAPP_PORT=9000 go run examples/simple/main.go --level dpanic
# &{dpanic 9000}
```

Built-in custom types like `zapcore.LogLevel` comes with automatic validation.

```bash
❯ MYAPP_LOGLEVEL=debug MYAPP_PORT=9000 go run examples/simple/main.go --level what
# Error: invalid argument "what" for "--level" flag: must be 'debug', 'dpanic', 'error', 'fatal', 'info', 'panic', 'warn'
# Usage:
#   myapp [flags]
#
# Flags:
#       --level zapcore.Level   Set logging level {debug,info,warn,error,dpanic,panic,fatal} (default info)
#   -p, --port int              Server port (default 3000)
#
# invalid argument "what" for "--level" flag: must be 'debug', 'dpanic', 'error', 'fatal', 'info', 'panic', 'warn'
# exit status 1
```

Your CLI now supports:

- 📝 Command-line flags (`--level info`, `-p 8080`)
- 🌍 Environment variables (`MYAPP_PORT=8080`)
- 💦 Options precedence (flags > env vars > config file)
- ✅ Automatic validation and type conversion
- 📚 Beautiful help output with proper grouping

## 📦  Key Features

### 🧩 Declarative Flags Definition

Define flags once using Go struct tags.

No more boilerplate for `Flags().StringVarP`, `Flags().IntVar`, `viper.BindPFlag`, etc.

Yes, you can _nest_ structs too.

```go
type ServerOptions struct {
	// Basic flags
	Host string `flag:"host" flagdescr:"Server host" default:"localhost"`
	Port int    `flagshort:"p" flagdescr:"Server port" flagrequired:"true" flagenv:"true"`

	// Environment variable binding
	APIKey string `flagenv:"true" flagdescr:"API authentication key"`

	// Flag grouping for organized help
	LogLevel zapcore.Level `flag:"log-level" flaggroup:"Logging" flagdescr:"Set log level"`
	LogFile  string        `flag:"log-file" flaggroup:"Logging" flagdescr:"Log file path" flagenv:"true"`

	// Nested structs for organization
	Database DatabaseConfig `flaggroup:"Database"`

	// Custom type
	TargetEnv Environment `flagcustom:"true" flag:"target-env" flagdescr:"Set the target environment"`
}

type DatabaseConfig struct {
	URL      string `flag:"db-url" flagdescr:"Database connection URL"`
	MaxConns int    `flagdescr:"Max database connections" default:"10" flagenv:"true"`
}
```

See [full example](examples/full/cli/cli.go) for more details.

### 🛠️ Automatic Environment Variable Binding

Automatically generate environment variables binding them to configuration files (YAML, JSON, TOML, etc.) and flags.

From the previous options struct, you get the following env vars automatically:

- `FULL_SRV_PORT`
- `FULL_SRV_APIKEY`
- `FULL_SRV_DATABASE_MAXCONNS`
- `FULL_SRV_LOGFILE`, `FULL_SRV_LOG_FILE`

Every struct field with the `flagenv:"true"` tag gets an environment variable (two if the struct field also has the `flag:"..."` tag, see struct field `LogFile`).

The prefix of the environment variable name is the CLI name plus the command name to which those options are attached to.

### ⚙️ Configuration File Support

Easily set up configuration file discovery (flag, environment variable, and fallback paths) with a single line of code.

```go
//
autoflags.SetupConfig(rootCmd, autoflags.ConfigOptions{AppName: "full"})
```

The line above:

- creates `--config` global flag
- creates `FULL_CONFIG` env var
- sets `/etc/full/`, `$HOME/.full/`, `$PWD/.full/` as fallback paths for `config.yaml`

Magic, isn't it?

What's left? Tell your CLI to load the configuration file (if any).

```go
rootC.PersistentPreRunE = func(c *cobra.Command, args []string) error {
	_, configMessage, configErr := autoflags.UseConfigSimple(c)
	if configErr != nil {
		return configErr
	}
	if configMessage != "" {
		c.Println(configMessage)
	}

	return nil
}
```

### ✅ Built-in Validation & Transformation

Supports validation, transformation, and custom flag type definitions through simple interfaces.

Just make your struct implement `ValidatableOptions` and `TransformableOptions` interfaces.

```go
type UserConfig struct {
	Email string `flag:"email" flagdescr:"User email" validate:"email"`
	Age   int    `flag:"age" flagdescr:"User age" validate:"min=18,max=120"`
	Name  string `flag:"name" flagdescr:"User name" mod:"trim,title"`
}

func (o *ServerOptions) Validate(ctx context.Context) []error {
    // Automatic validation
}

func (o *ServerOptions) Transform(ctx context.Context) error {
    // Automatic transformation
}
```

See a full working example [here](examples/full/cli/cli.go).

### 🚧 Automatic Debugging Support

Create a `--debug-options` flag (plus a `FULL_DEBUG_OPTIONS` env var) for troubleshooting config/env/flags resolution.

```go
autoflags.SetupDebug(rootCmd, autoflags.DebugOptions{})
```

```bash
❯ go run examples/full/main.go srv --debug-options --config examples/full/config.yaml -p 3333
#
# Aliases:
# map[string]string{"database.url":"db-url", "logfile":"log-file", "loglevel":"log-level", "targetenv":"target-env"}
# Override:
# map[string]interface {}{}
# PFlags:
# map[string]viper.FlagValue{"apikey":viper.pflagValue{flag:(*pflag.Flag)(0x14000109ea0)}, "database.maxconns":viper.pflagValue{flag:(*pflag.Flag)(0x140002181e0)}, "db-url":viper.pflagValue{flag:(*pflag.Flag)(0x14000218140)}, "host":viper.pflagValue{flag:(*pflag.Flag)(0x14000109d60)}, "log-file":viper.pflagValue{flag:(*pflag.Flag)(0x140002180a0)}, "log-level":viper.pflagValue{flag:(*pflag.Flag)(0x14000218000)}, "port":viper.pflagValue{flag:(*pflag.Flag)(0x14000109e00)}, "target-env":viper.pflagValue{flag:(*pflag.Flag)(0x14000218320)}}
# Env:
# map[string][]string{"apikey":[]string{"SRV_APIKEY"}, "database.maxconns":[]string{"SRV_DATABASE_MAXCONNS"}, "log-file":[]string{"SRV_LOGFILE", "SRV_LOG_FILE"}}
# Key/Value Store:
# map[string]interface {}{}
# Config:
# map[string]interface {}{"apikey":"secret-api-key", "database":map[string]interface {}{"maxconns":3}, "db-url":"postgres://user:pass@localhost/mydb", "host":"production-server", "log-file":"/var/log/mysrv.log", "log-level":"debug", "port":8443}
# Defaults:
# map[string]interface {}{"database":map[string]interface {}{"maxconns":"10"}, "host":"localhost"}
# Values:
# map[string]interface {}{"apikey":"secret-api-key", "database":map[string]interface {}{"maxconns":3, "url":"postgres://user:pass@localhost/mydb"}, "db-url":"postgres://user:pass@localhost/mydb", "host":"production-server", "log-file":"/var/log/mysrv.log", "log-level":"debug", "logfile":"/var/log/mysrv.log", "loglevel":"debug", "port":3333, "target-env":"dev", "targetenv":"dev"}
```

### 🪃 Custom Type Handlers

Declare options (flags, env vars, config file keys) with custom types.

Just implement two methods on your options structs:

- `Define<FieldName>` for defining the custom flag, its description, etc.
- `Decode<FieldName>` for converting any input to your custom type (or erroring out)

```go
type Environment string

const (
	EnvDevelopment Environment = "dev"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "prod"
)

type ServerOptions struct {
	...
	// Custom type
	TargetEnv Environment `flagcustom:"true" flag:"target-env" flagdescr:"Set the target environment"`
}

// DefineTargetEnv defines the custom flag for Environment with autocompletion
func (o *ServerOptions) DefineTargetEnv(c *cobra.Command, name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) {
	...
}

// DecodeTargetEnv converts string input to Environment type with validation
func (o *ServerOptions) DecodeTargetEnv(input any) (any, error) {
	...
}
```

See [full example](examples/full/cli/cli.go) for more details.

### 📜 Configuration Is First-Class Citizen

```yaml
# Global settings
# Command-specific overrides
```

TODO: complete

### 🎨 Beautiful, Organized Help Output

Organize your `--help` output into logical groups for better readability.

```bash
❯ go run examples/full/main.go --help
# A demonstration of the autoflags library with beautiful CLI features
#
# Usage:
#   full [flags]
#   full [command]
#
# Available Commands:
#   completion  Generate the autocompletion script for the specified shell
#   help        Help about any command
#   srv         Start the server
#   usr         User management
#
# Global Flags:
#       --config string   config file (fallbacks to: {/etc/full,{executable_dir}/.full,$HOME/.full}/config.{yaml,json,toml})
#       --debug-options   enable debug output for options
#
# Utility Flags:
#       --dry-run
#   -v, --verbose count
```

```bash
❯ go run examples/full/main.go srv --help
# Start the server with the specified configuration
#
# Usage:
#   full srv [flags]
#   full srv [command]
#
# Available Commands:
#   version     Print version information
#
# Flags:
#       --apikey string       API authentication key
#       --host string         Server host (default "localhost")
#   -p, --port int            Server port
#       --target-env string   Set the target environment {dev,staging,prod} (default "dev")
#
# Database Flags:
#       --database.maxconns int   Max database connections (default 10)
#       --db-url string           Database connection URL
#
# Logging Flags:
#       --log-file string           Log file path
#       --log-level zapcore.Level   Set log level {debug,info,warn,error,dpanic,panic,fatal} (default info)
#
# Global Flags:
#       --config string   config file (fallbacks to: {/etc/full,{executable_dir}/.full,$HOME/.full}/config.{yaml,json,toml})
#       --debug-options   enable debug output for options
#
# Use "full srv [command] --help" for more information about a command.
```

## 🏷️ Available Struct Tags

Use these tags in your struct fields to control the behavior:

| Tag            | Description                                                                                                                             | Example                     |
| -------------- | --------------------------------------------------------------------------------------------------------------------------------------- | --------------------------- |
| `flag`         | Sets a custom name for the flag (otherwise, generated from the field name)                                                              | `flag:"log-level"`          |
| `flagshort`    | Sets a single-character shorthand for the flag                                                                                          | `flagshort:"l"`             |
| `flagdescr`    | Provides the help text for the flag                                                                                                     | `flagdescr:"Logging level"` |
| `default`      | Sets the default value for the flag                                                                                                     | `default:"info"`            |
| `flagenv`      | Enables binding to an environment variable (`"true"`/`"false"`)                                                                         | `flagenv:"true"`            |
| `flagrequired` | Marks the flag as required (`"true"`/`"false"`)                                                                                         | `flagrequired:"true"`       |
| `flaggroup`    | Assigns the flag to a group in the help message                                                                                         | `flaggroup:"Database"`      |
| `flagignore`   | Skips creating a flag for this field (`"true"`/`"false"`)                                                                               | `flagignore:"true"`         |
| `flagcustom`   | Uses a custom `Define<FieldName>` method for advanced flag creation and a custom `Decode<FieldName>` method for advanced value decoding | `flagcustom:"true"`         |
| `flagtype`     | Specifies a special flag type. Currently supports `count`                                                                               | `flagtype:"count"`          |

## 📖 Documentation

For comprehensive documentation and advanced usage patterns, visit the [documentation](https://pkg.go.dev/github.com/leodido/autoflags).

Or take a look at the [examples](examples/).

## 🤝 Contributing

Contributions are welcome!

Please feel free to submit a Pull Request.
