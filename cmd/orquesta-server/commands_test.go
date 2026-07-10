package main

import (
	"strings"
	"testing"
)

func TestRunMainV0WithoutCommandReturnsUsageError(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder
	if code := runMain(nil, &stdout, &stderr); code != 2 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "comando requerido") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}
