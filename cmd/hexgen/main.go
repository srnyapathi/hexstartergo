package main

import (
	"fmt"
	"hexgen/internal/cli"
	"os"
)

func main() {
	parent, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, parent))
}
