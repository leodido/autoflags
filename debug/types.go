package debug

// Options configures the debug functionality for command-line applications.
type Options struct {
	AppName  string
	FlagName string // Name of debug flag (defaults to "debug-options")
	EnvVar   string // Environment variable (defaults to {APP}_DEBUG_OPTIONS)
	Exit     bool   // Forces the CLI to exit after printing the debug information without executing the command's RunE
}
