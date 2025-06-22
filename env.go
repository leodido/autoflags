package structcli

import (
	"fmt"
	"strings"

	internalenv "github.com/leodido/structcli/internal/env"
)

// GetOrSetAppName resolves the app name consistently.
//
// When name is given, use it (and set as prefix if none exists).
// When cName is given, use it if no prefix exists, or if existing prefix matches cName.
// Otherwise, when an environment prefix already exists, return the app name that corresponds to it.
// Finally, it falls back to empty string.
func GetOrSetAppName(name, cName string) string {
	// If a name was explicitly given then use it
	if name != "" {
		if EnvPrefix() == "" {
			// Also as a prefix if there's not one already
			SetEnvPrefix(name)
		}

		return name
	}

	existingPrefix := EnvPrefix()

	// When command name is given
	if cName != "" {
		if existingPrefix == "" {
			// No existing prefix, set it and return command name
			SetEnvPrefix(cName)

			return cName
		} else if strings.EqualFold(existingPrefix, cName) {
			// Existing prefix matches command name (case-insensitive)
			// This means the prefix was set by the command name, return command name

			return cName
		} else {
			// Existing prefix doesn't match command name
			// This means the prefix was set by an explicit AppName
			// Return the lowercase version of the prefix (to match original app name case)

			return existingPrefix
		}
	}

	// No command name provided, use existing prefix if available
	if existingPrefix != "" {
		return existingPrefix
	}

	return ""
}

// SetEnvPrefix sets the global environment variable prefix for the application.
//
// The prefix is automatically appended with an underscore when generating environment variable names.
func SetEnvPrefix(str string) {
	if str == "" {
		internalenv.Prefix = ""

		return
	}

	internalenv.Prefix = fmt.Sprintf("%s%s", strings.TrimSuffix(internalenv.NormEnv(str), internalenv.EnvSep), internalenv.EnvSep)
}

// EnvPrefix returns the current global environment variable prefix without the trailing underscore.
func EnvPrefix() string {
	return strings.TrimSuffix(internalenv.Prefix, internalenv.EnvSep)
}
