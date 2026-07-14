//go:build !unix

package codex

import "os/exec"

func configureProcessGroup(*exec.Cmd) {}

// Non-Unix compositions need an OS-specific containment primitive, such as a
// Windows Job Object, before claiming descendant cleanup.
func cleanupProcessGroup(*exec.Cmd) error { return nil }
