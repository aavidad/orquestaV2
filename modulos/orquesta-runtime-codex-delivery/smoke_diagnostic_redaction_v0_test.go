package orquestaruntimecodexdelivery

import (
	"strings"
	"testing"
)

func TestCodexRealSmokeDiagnosticRedactionV0(t *testing.T) {
	input := "/home/alberto/project\naccess_token=secret\nprompt=raw\n" + strings.Repeat("x", 1700)
	got := realSmokeDiagnosticTextV0("codex_real_smoke_test", input)
	for _, forbidden := range []string{"/home/alberto", "access_token=secret", "prompt=raw"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("diagnostic leaks %q: %s", forbidden, got)
		}
	}
	if len(got) > codexRealSmokeDiagnosticMaxBytesV0+len("\n[truncated]") {
		t.Fatalf("diagnostic not bounded: %d", len(got))
	}
}
