//go:build linux

package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestWriteSupervisorFailureUsesOnlySafeStructuredFallback(t *testing.T) {
	const sensitive = "open /srv/private/operator/spec.json: permission denied"
	var output bytes.Buffer
	writeSupervisorFailure(&output, errors.New(sensitive))
	if output.String() != "status=failed\nstage=supervisor\ncode=firecracker_attestor_e2e.failed\n" {
		t.Fatalf("unexpected diagnostic: %q", output.String())
	}
	if strings.Contains(output.String(), sensitive) ||
		strings.Contains(output.String(), "/srv/") ||
		strings.Contains(output.String(), "permission denied") {
		t.Fatalf("sensitive cause leaked: %q", output.String())
	}
}
