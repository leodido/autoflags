package main

import (
	"log"

	full_example_cli "github.com/leodido/structcli/examples/full/cli"
)

func main() {
	log.SetFlags(0)
	c, e := full_example_cli.NewRootC(true)
	if e != nil {
		log.Fatalln(e)
	}

	if err := c.Execute(); err != nil {
		log.Fatalln(err)
	}
}
