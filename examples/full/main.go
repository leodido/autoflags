package main

import (
	"os"

	full_example_cli "github.com/leodido/autoflags/examples/full/cli"
)

func main() {
	c := full_example_cli.NewRootC()
	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}
