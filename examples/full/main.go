package main

import (
	"log"

	full_example_cli "github.com/leodido/autoflags/examples/full/cli"
)

func main() {
	log.SetFlags(0)
	c := full_example_cli.NewRootC()

	if err := c.Execute(); err != nil {
		log.Fatalln(err)
	}
}
