package main

import (
	"strings"
	"testing"

	orquestarails "orquesta/modulos/orquesta-rails"
)

const goSmokeDiagnosticMaxBytesV0 = 1600

func goSmokeDiagnosticForTestV0(value string) string {
	redacted, _ := orquestarails.RedactOperationalTextForFieldV0("go_smoke_test", "diagnostic", value)
	redacted = strings.TrimSpace(redacted)
	if len(redacted) <= goSmokeDiagnosticMaxBytesV0 {
		return redacted
	}
	return redacted[:goSmokeDiagnosticMaxBytesV0] + "\n[truncated]"
}

func TestGoSmokeDiagnosticForTestV0RedactaYAcota(t *testing.T) {
	got := goSmokeDiagnosticForTestV0("/home/alberto/project\nsecret=value\npayload=raw\n" + strings.Repeat("x", 1700))
	for _, forbidden := range []string{"/home/alberto", "secret=value", "payload=raw"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("diagnostic leaks %q: %s", forbidden, got)
		}
	}
	if len(got) > goSmokeDiagnosticMaxBytesV0+len("\n[truncated]") {
		t.Fatalf("diagnostic not bounded: %d", len(got))
	}
}
