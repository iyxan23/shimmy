package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"shimmy/internal/generator"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "shimmy:", err)
		os.Exit(1)
	}
}

func run() error {
	var input string
	var iface string
	var output string

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&input, "input", "", "Go source file containing target interface")
	fs.StringVar(&iface, "type", "", "interface name to generate")
	fs.StringVar(&output, "output", "", "output path for generated file")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	if err := validateFlags(input, iface, output); err != nil {
		return err
	}

	out, err := generator.GenerateFromFile(input, iface)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	if err := os.WriteFile(output, out, 0o644); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}

	return nil
}

func validateFlags(input, iface, output string) error {
	var missing []string
	if input == "" {
		missing = append(missing, "-input")
	}
	if iface == "" {
		missing = append(missing, "-type")
	}
	if output == "" {
		missing = append(missing, "-output")
	}

	if len(missing) > 0 {
		return errors.New("missing required flag(s): " + joinFlags(missing))
	}
	return nil
}

func joinFlags(flags []string) string {
	if len(flags) == 1 {
		return flags[0]
	}
	out := flags[0]
	for i := 1; i < len(flags); i++ {
		out += ", " + flags[i]
	}
	return out
}
