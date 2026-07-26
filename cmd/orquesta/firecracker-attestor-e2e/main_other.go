//go:build !linux

package main

import (
	"fmt"
	"os"
)

func main() {
	_, _ = fmt.Fprintln(os.Stderr, "status=failed\ncode=firecracker_attestor_e2e.unsupported")
	os.Exit(1)
}
