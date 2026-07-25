//go:build !linux

package main

import (
	"fmt"
	"os"

	"orquesta/internal/adapters/attestor/firecrackerlauncher"
)

func main() {
	_, _ = fmt.Fprintf(
		os.Stderr,
		"code=%s\n",
		firecrackerlauncher.CodeUnavailable,
	)
	os.Exit(1)
}
