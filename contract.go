package autoflags

import (
	"context"

	"github.com/spf13/cobra"
)

// Options represents a struct that can define command-line flags, env vars, config file keys.
//
// Types implementing this interface can be used with Define() to automatically generate flags from struct fields.
type Options interface {
	Attach(*cobra.Command)
}

// ValidatableOptions extends Options with validation capabilities.
//
// The Validate method is called automatically during Unmarshal().
type ValidatableOptions interface {
	Validate() []error
}

// TransformableOptions extends Options with transformation capabilities.
//
// The Transform method is called automatically during Unmarshal() before validation.
type TransformableOptions interface {
	Transform(context.Context) error
}

// CommonOptions extends Options with context manipulation capabilities.
//
// The Context method is called automatically during Unmarshal() to modify the command context.
type CommonOptions interface {
	Options
	Context(context.Context) context.Context
}
