package options

import (
	"context"

	"github.com/spf13/cobra"
)

// Options can be directly attached to cobra.Command instances.
type Options interface {
	Attach(*cobra.Command)
}

type ValidatableOptions interface {
	Validate() []error
}

type TransformableOptions interface {
	Transform(context.Context) error
}

type CommonOptions interface {
	Context(context.Context) context.Context
}
