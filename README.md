[![Coverage](https://img.shields.io/codecov/c/github/leodido/structcli.svg?style=for-the-badge)](https://codecov.io/gh/leodido/structcli) [![Documentation](https://img.shields.io/badge/godoc-reference-blue.svg?style=for-the-badge)](https://godoc.org/github.com/leodido/structcli) [![GoReportCard](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/leodido/structcli)

> CLI generation from Go structs

Transform your Go structs into fully-featured command-line interfaces with configuration files, environment variables, flags, validation, and beautiful help output.

> Declare your options in a struct, and let structcli do the rest

You don't need much: just a few struct tags + **structcli**.

Stop writing boilerplate. Start building features.

## ⚡ Quick Start

```go
package main

import (
	"fmt"
	"log"

	"github.com/leodido/structcli"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

type Options struct {
	LogLevel zapcore.Level
	Port     int
}

func (o *Options) Attach(c *cobra.Command) error {
	return structcli.Define(c, o) // This is it
}

func main() {
	log.SetFlags(0)
	opts := &Options{}
	cli := &cobra.Command{Use: "myapp"}

	// This single line creates all the options (flags, env vars, config keys)
	if err := opts.Attach(cli); err != nil {
		log.Fatalln(err)
	}

	cli.PreRunE = func(c *cobra.Command, args []string) error {
		return structcli.Unmarshal(c, opts) // Populates struct from config keys, env variables, flags
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

Just annotate your struct with `structcli` tags.

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

## ⬇️ Install

```bash
go get github.com/leodido/structcli
```

## 📦 Key Features

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
structcli.SetupConfig(rootCmd, config.Options{AppName: "full"})
```

The line above:

- creates `--config` global flag
- creates `FULL_CONFIG` env var
- sets `/etc/full/`, `$HOME/.full/`, `$PWD/.full/` as fallback paths for `config.yaml`

Magic, isn't it?

What's left? Tell your CLI to load the configuration file (if any).

```go
rootC.PersistentPreRunE = func(c *cobra.Command, args []string) error {
	_, configMessage, configErr := structcli.UseConfigSimple(c)
	if configErr != nil {
		return configErr
	}
	if configMessage != "" {
		c.Println(configMessage)
	}

	return nil
}
```

#### 📜 Configuration Is First-Class Citizen

Configuration can mirror your command hierarchy.

Settings can be global (at the top level) or specific to a command or subcommand. The most specific section always takes precedence.

```yaml
# Global settings apply to all commands unless overridden by a specific section.
# `dryrun` matches the `DryRun` struct field name.
dryrun: true
verbose: 1 # A default verbosity level for all commands.

# Config for the `srv` command (`full srv`)
srv:
  # `port` matches the `Port` field name.
  port: 8433
  # `log-level` matches the `flag:"log-level"` tag.
  log-level: "warn"
  # `logfile` matches the `LogFile` field name.
  logfile: /var/log/mysrv.log

  # Flattened keys can set options in nested structs.
  # `db-url` (from `flag:"db-url"` tag) maps to ServerOptions.Database.URL.
  db-url: "postgres://user:pass@db/prod"

# Config for the `usr` command group.
usr:
  # This nested section matches the `usr add` command (`full usr add`).
  # Its settings are ONLY applied to 'usr add'.
  add:
    name: "Config User"
    email: "config.user@example.com"
    age: 42
    # Command specific override
    dry: false

# NOTE: Per the library's design, there is no other fallback other than from the top-level.
# A command like 'usr delete' would ONLY use the global keys above (if those keys/flags are attached to it),
# as an exact 'usr.delete' section is not defined.
```

This configuration system supports:

- **Hierarchical Structure**: Nest keys to match your command path (e.g., `usr: { add: { ... } }`).
- **Strict Precedence**: Only settings from the global scope and the exact command path section are merged. There is no automatic fallback to parent command sections.
- **Flexible Keys**: Use either the struct field name (`lowercase(DryRun)`) or the flag tag (`flag:"log-level"`) as keys.
- **Flattened Keys**: Set options in nested structs easily using a single, flattened key (e.g., `db-url`).

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
structcli.SetupDebug(rootCmd, debug.Options{})
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

### ↪️ Sharing Options Between Commands

In complex CLIs, multiple commands often need access to the same global configuration and shared resources (like a logger or a database connection). `structcli` provides a powerful pattern using the [ContextOptions](/contract.go) interface to achieve this without resorting to global variables, by propagating a single "source of truth" through the command context.

The pattern allows you to:

- Populate a shared options struct once from flags, environment variables, or a config file.
- Initialize "computed state" (like a logger) based on those options.
- Share this single, fully-prepared "source of truth" with any subcommand that needs it.

#### 🍩 In a Nutshell

Create a shared struct that implements the `ContextOptions` interface. This struct will hold both the configuration flags and the computed state (e.g., the logger).

```go
// This struct holds our shared state.
type CommonOptions struct {
    LogLevel zapcore.Level `flag:"loglevel" flagdescr:"Logging level" default:"info"`
    Logger   *zap.Logger   `flagignore:"true"` // This field is computed, not a flag.
}

// The Context/FromContext methods enable the propagation pattern.
func (o *CommonOptions) Context(ctx context.Context) context.Context { /* ... */ }
func (o *CommonOptions) FromContext(ctx context.Context) error { /* ... */ }

// Initialize is a custom method to create the computed state.
func (o *CommonOptions) Initialize() error { /* ... */ }
```

Initialize the state in the root command. Use a `PersistentPreRunE` hook on your root command to populate your struct and initialize any resources.
Invoking `structcli.Unmarshal` will automatically inject the prepared object into the context for all subcommands to use.

```go
rootC.PersistentPreRunE = func(c *cobra.Command, args []string) error {
	// Populate the master `commonOpts` from flags, env, and config file.
	if err := structcli.Unmarshal(c, commonOpts); err != nil {
		return err
	}
	// Use the populated values to initialize the computed state (the logger).
	if err := commonOpts.Initialize(); err != nil {
		return err
	}

	return nil
}
```

Finally, retrieve the state in subcommands. In your subcommand's `RunE`, simply call `.FromContext()` to retrieve the shared, initialized object.

```go
func(c *cobra.Command, args []string) error {
    // Create a receiver and retrieve the master state from the context.
    config := &CommonOptions{}
    if err := config.FromContext(c.Context()); err != nil {
        return err
    }
    config.Logger.Info("Executing subcommand...")

    return nil
},
```

This pattern ensures that subcommands remain decoupled while having access to a consistent, centrally-managed state.

For a complete, runnable implementation of this pattern, see the loginsvc example located in the [/examples/loginsvc](/examples/loginsvc/) directory.

### 🪃 Custom Type Handlers

Declare options (flags, env vars, config file keys) with custom types by implementing two methods on your options struct.

Just implement two methods on your options structs:

- `Define<FieldName>`: return a `pflag.Value` that knows how to handle your custom type, along with an enhanced description.
- `Decode<FieldName>`: decode the input into your custom type.

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

// DefineTargetEnv returns a pflag.Value for the custom Environment type.
func (o *ServerOptions) DefineTargetEnv(name, short, descr string, structField reflect.StructField, fieldValue reflect.Value) (pflag.Value, string) {
    enhancedDesc := descr + " {dev,staging,prod}"
    fieldPtr := fieldValue.Addr().Interface().(*Environment)
    *fieldPtr = "dev" // Set default

    return structclivalues.NewString((*string)(fieldPtr)), enhancedDesc
}

// DecodeTargetEnv converts the string input to the Environment type.
func (o *ServerOptions) DecodeTargetEnv(input any) (any, error) {
	// ... (validation and conversion logic)
    return EnvDevelopment, nil
}

// Attach handles flag definition and shell completion for our custom type.
func (o *ServerOptions) Attach(c *cobra.Command) error {
	if err := structcli.Define(c, o); err != nil {
        return err
    }

    // Register shell completion after the flag has been defined.
    c.RegisterFlagCompletionFunc("target-env", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        return []string{"dev", "staging", "prod"}, cobra.ShellCompDirectiveNoFileComp
    })

    return nil
}
```

In [values](/values/values.go) we provide `pflag.Value` implementations for standard types.

See [full example](examples/full/cli/cli.go) for more details.

### 🎨 Beautiful, Organized Help Output

Organize your `--help` output into logical groups for better readability.

```bash
❯ go run examples/full/main.go --help
# A demonstration of the structcli library with beautiful CLI features
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

For comprehensive documentation and advanced usage patterns, visit the [documentation](https://pkg.go.dev/github.com/leodido/structcli).

Or take a look at the [examples](examples/).

## 🤝 Contributing

Contributions are welcome!

Please feel free to submit a Pull Request.
