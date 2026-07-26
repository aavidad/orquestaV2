//go:build !linux

package main

import (
	"os"

	"orquesta/internal/bootstrap/firecracker"
)

func main() {
	os.Exit(firecracker.RunFirecrackerLauncher(os.Args[1:], os.Stdout, os.Stderr))
}
