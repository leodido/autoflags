# autoflags

> CLI generation from Go structs

Transform your Go structs into fully-featured command-line interfaces with configuration files, environment variables, validation, and beautiful help output.

All with just a few struct tags: declare your options in a struct, and let autoflags do the rest.

Stop writing boilerplate. Start building features. ⚡

## ⚡ Quick Start

```go
package main

import (
	"fmt"

	"github.com/leodido/autoflags"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
)

type Options struct {
	LogLevel zapcore.Level `flag:"log-level" flagcustom:"true" flagdescr:"Set logging level" default:"info" flagenv:"true"`
	Port     int           `flag:"port" flagshort:"p" flagdescr:"Server port" flagenv:"true"`
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
```

**That's it**!

```bash
❯ go run examples/simple/main.go --help
# Usage:
#   myapp [flags]
#
# Flags:
#       --log-level zapcore.Level   Set logging level {debug,info,warn,error,dpanic,panic,fatal} (default info)
#   -p, --port int                  Server port
```

```bash
❯ MYAPP_LOG_LEVEL=debug MYAPP_PORT=9000 go run examples/simple/main.go --log-level warn
# &{warn 9000}
```

```bash
❯ MYAPP_LOG_LEVEL=debug MYAPP_PORT=9000 go run examples/simple/main.go --log-level what
# Error: invalid argument "what" for "--log-level" flag: must be 'debug', 'dpanic', 'error', 'fatal', 'info', 'panic', 'warn'
# Usage:
#   myapp [flags]
#
# Flags:
#       --log-level zapcore.Level   Set logging level {debug,info,warn,error,dpanic,panic,fatal} (default info)
#   -p, --port int                  Server port
```

Your CLI now supports:

- 📝 Command-line flags (`--log-level info`, `-p 8080`)
- 🌍 Environment variables (`MYAPP_PORT=8080`)
- 💦 Options precedence (flags > env vars > config file)
- ✅ Automatic validation and type conversion
- 📚 Beautiful help output with proper grouping

## Key Features

### 🏗️ Declarative Flags Definition

Define flags once using Go struct tags.

No more boilerplate for `Flags().StringVarP`, `Flags().IntVar`, `viper.BindPFlag`, etc.

Yes, you can _nest_ structs too.

```go
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
```

### 🔧 Automatic Environment Variable Binding

Automatically generate environment variables binding them to configuration files (YAML, JSON, TOML, etc.) and flags.

From the previous options struct, assuming the CLI name is "mysrv", you get the following env vars automatically:

- `MYSRV_DB_MAX_CONNS`, `MYSRV_DATABASE_MAXCONNS`
- `MYSRV_APIKEY`

This is how the environment variables name are computed:

- `<cli_name>_[<struct_name>_]<struct_field_name>`
- `<cli_name>_[<struct_name>_]<struct_flag_tag_value>`

### 🎨 Beautiful, Organized Help Output

Organize your `--help` output into logical groups for better readability.

```bash
❯ go run examples/with_config_file/main.go --help
# Usage:
#   mysrv [flags]
#
# Flags:
#       --apikey string   API authentication key
#       --host string     Server host (default "localhost")
#   -p, --port int        Server port
#
# Database Flags:
#       --db-max-conns int   Max database connections (default 10)
#       --db-url string      Database connection URL
#
# Logging Flags:
#       --log-file string           Log file path
#       --log-level zapcore.Level   Set log level {debug,info,warn,error,dpanic,panic,fatal} (default info)
```

### 📁 One Line Configuration File Support

Add configuration file searching in standard paths (`/etc/{APP}`, `$HOME/.{APP}`, `$PWD/.{APP}`) with a single line of code.

```go
// Set up config file discovery
autoflags.SetupConfig(rootCmd, autoflags.ConfigOptions{
    AppName: "myapp",
    // Creates --config global flag
    // Creates MYAPP_CONFIG env var
    // Automatically falls back to search /etc/myapp/, $HOME/.myapp/, ./.myapp/ for config.yaml
})
```

### ✅ Built-in Validation & Transformation

Supports validation, transformation, and custom flag type definitions through simple interfaces.

```go
type UserOptions struct {
    Email string `flag:"email" flagdescr:"User email" validate:"required,email"`
    Age   int    `flag:"age" flagdescr:"User age" validate:"min=18,max=120"`
    Name  string `flag:"name" flagdescr:"User name" mod:"trim,title"`
}

func (o *UserOptions) Validate() []error {
    return validator.New().Struct(o) // Automatic validation
}

func (o *UserOptions) Transform(ctx context.Context) error {
    return mold.New().Struct(ctx, o) // Automatic transformation
}
```

### 🎯 Advanced Features

#### Custom Type Handlers

TODO: ...

#### Command-Specific Configuration

```yaml
# Global settings
timeout: 30
log-level: info

# Command-specific overrides
serve:
  timeout: 300
  port: 8080

migrate:
  timeout: 3600
  dry-run: true
```

TODO: ...

#### Debug Support

```go
autoflags.SetupDebug(rootCmd, autoflags.DebugOptions{})
// Adds --debug-options flag for troubleshooting config/env/flags resolution
```

## Available Struct Tags

Use these tags in your struct fields to control the behavior:

| Tag | Description | Example |
|-----|-------------|---------|
| `flag` | Sets a custom name for the flag (otherwise, generated from the field name) | `flag:"log-level"` |
| `flagshort` | Sets a single-character shorthand for the flag | `flagshort:"l"` |
| `flagdescr` | Provides the usage/help text for the flag | `flagdescr:"Logging level"` |
| `default` | Sets the default value for the flag | `default:"info"` |
| `flagenv` | Enables binding to an environment variable (`"true"`/`"false"`) | `flagenv:"true"` |
| `flagrequired` | Marks the flag as required (`"true"`/`"false"`) | `flagrequired:"true"` |
| `flaggroup` | Assigns the flag to a group in the help message | `flaggroup:"Database"` |
| `flagignore` | Skips creating a flag for this field (`"true"`/`"false"`) | `flagignore:"true"` |
| `flagcustom` | Uses a custom `Define<FieldName>` method for advanced flag creation | `flagcustom:"true"` |
| `flagtype` | Specifies a special flag type. Currently supports `count` | `flagtype:"count"` |

## 📖 Documentation

For comprehensive documentation and advanced usage patterns, visit the [documentation](https://pkg.go.dev/github.com/leodido/autoflags).

Or take a look at the [examples](examples/).

## 🤝 Contributing

Contributions are welcome!

Please feel free to submit a Pull Request.
