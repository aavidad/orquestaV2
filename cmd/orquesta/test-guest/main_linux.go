//go:build linux

package main

import (
	"os"

	"orquesta/internal/bootstrap/firecracker"
)

func main() {
	os.Exit(run(os.Args))
}

func run(arguments []string) int {
	return firecracker.RunTestGuest(arguments)
}
