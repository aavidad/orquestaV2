package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionAndInvalidCommandDoNotStartRuntime(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"version"}, &stdout, &stderr); code != 0 || strings.TrimSpace(stdout.String()) != version {
		t.Fatalf("version: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"unknown"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "solicitud") {
		t.Fatalf("invalid: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
