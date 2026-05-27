package main

import (
	"strings"
	"testing"
)

func TestMCPRealSmokeMarshalV0DevuelveInternalInvariantSinPanic(t *testing.T) {
	_, err := mcpMarshalMCPRealSmokeV0(make(chan string))
	if err == nil {
		t.Fatalf("err nil")
	}
	if got := err.Error(); got != "internal_invariant:mcp_smoke_arguments_json" ||
		strings.Contains(got, "chan") {
		t.Fatalf("err=%q", got)
	}
}
