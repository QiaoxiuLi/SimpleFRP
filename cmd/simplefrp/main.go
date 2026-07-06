package main

import (
	"fmt"
	"os"

	"github.com/simplefrp/simplefrp/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
