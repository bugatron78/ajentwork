package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ajentwork/internal/help"
	"ajentwork/internal/render"
)

func main() {
	outputPath := flag.String("output", "", "write the generated man page to this path instead of stdout")
	flag.Parse()

	page := render.ManPage(help.DefaultRegistry(), time.Now().UTC())
	if *outputPath == "" {
		fmt.Print(page)
		return
	}

	path := filepath.Clean(*outputPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "create output directory: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, []byte(page), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write man page: %v\n", err)
		os.Exit(1)
	}
}
