//go:build !linux

package firecracker

import (
	"fmt"
	"io"

	"orquesta/internal/adapters/attestor/firecrackerlauncher"
)

// RunFirecrackerLauncher reports that physical launch is unavailable off Linux.
func RunFirecrackerLauncher(_ []string, _, stderr io.Writer) int {
	_, _ = fmt.Fprintf(
		stderr,
		"code=%s\n",
		firecrackerlauncher.CodeUnavailable,
	)
	return 1
}
