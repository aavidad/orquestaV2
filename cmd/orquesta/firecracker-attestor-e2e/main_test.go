//go:build linux

package main

import (
	"bytes"
	"testing"
)

func TestAttestorE2EMainDelegatesCanonicalArgumentValidation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := run(nil, &stdout, &stderr); status != 2 ||
		stderr.String() != "status=failed\ncode=firecracker_attestor_e2e.arguments_invalid\n" ||
		stdout.Len() != 0 {
		t.Fatalf("status/output=%d %q %q", status, stdout.String(), stderr.String())
	}
}
