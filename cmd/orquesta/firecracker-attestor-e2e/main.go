//go:build linux

package main

import (
	"io"
	"os"

	"orquesta/internal/bootstrap/firecracker"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	return firecracker.RunFirecrackerAttestorE2E(arguments, stdout, stderr)
}
