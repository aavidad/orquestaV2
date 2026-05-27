package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestMCPRealSmokeCommandV0RequiereConfirmacion(t *testing.T) {
	t.Setenv(mcpRealSmokeConfirmEnvV0, "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := mcpRealSmokeCommandV0(&stdout, &stderr)
	if exitCode != 2 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", exitCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), mcpRealSmokeConfirmEnvV0) {
		t.Fatalf("stderr=%s", stderr.String())
	}
}
