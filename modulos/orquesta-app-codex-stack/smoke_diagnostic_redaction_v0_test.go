package orquestaappcodexstack

import (
	"strings"
	"testing"
)

func TestCodexStackRealSmokeDiagnosticRedactionV0(t *testing.T) {
	input := "/home/alberto/project\naccess_token=secret\ntranscript=raw\n" + strings.Repeat("x", 1700)
	got := codexStackRealSmokeTruncateV0(input)
	for _, forbidden := range []string{"/home/alberto", "access_token=secret", "transcript=raw"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("diagnostic leaks %q: %s", forbidden, got)
		}
	}
	if len(got) > 1600+len("\n[truncated]") {
		t.Fatalf("diagnostic not bounded: %d", len(got))
	}
}
