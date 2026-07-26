//go:build linux

package main

import (
	"bytes"
	"testing"
)

func TestLauncherMainDelegatesCanonicalArgumentValidation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := run(nil, &stdout, &stderr); status != 2 ||
		stderr.String() != "code=test_attestor.firecracker_launcher_config_invalid\n" ||
		stdout.Len() != 0 {
		t.Fatalf("status/output=%d %q %q", status, stdout.String(), stderr.String())
	}
}
