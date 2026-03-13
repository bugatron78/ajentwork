package main

import (
	"os"

	"ajentwork/internal/cli"
)

func main() {
	runner := cli.NewRunner(os.Stdout, os.Stderr)
	os.Exit(runner.Run(os.Args[1:]))
}
