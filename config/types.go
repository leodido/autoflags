package config

// SearchPathType represents different strategies for locating configuration files.
type SearchPathType int

const (
	// SearchPathEtc searches in /etc/{app} directory.
	SearchPathEtc SearchPathType = iota
	// SearchPathHomeHidden searches in $HOME/.{app} directory.
	SearchPathHomeHidden
	// SearchPathWorkingDirHidden searches in $PWD/.{app} directory.
	SearchPathWorkingDirHidden
	// SearchPathExecutableDirHidden searches in {executable_dir}/.{app} directory.
	SearchPathExecutableDirHidden
	// SearchPathCustom uses custom paths provided in CustomPaths field.
	SearchPathCustom
)

// Options defines configuration file behavior and search paths.
type Options struct {
	AppName     string
	FlagName    string           // Name of config flag (defaults to "config")
	ConfigName  string           // Config file name without extension (defaults to "config")
	EnvVar      string           // Environment variable (defaults to {APP}_CONFIG)
	SearchPaths []SearchPathType // Search path strategies (defaults to common paths)
	CustomPaths []string         // Custom search paths (when SearchPaths contains SearchPathCustom)
}
