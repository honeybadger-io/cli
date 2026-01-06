// Package main is the entry point for the Honeybadger CLI application.
package main

import (
	"fmt"
	"os"

	"github.com/honeybadger-io/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
