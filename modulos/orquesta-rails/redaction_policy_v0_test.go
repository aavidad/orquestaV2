package orquestarails

import (
	"strings"
	"testing"
)

func TestRedactOperationalTextForFieldV0(t *testing.T) {
	t.Setenv(DetailProhibitedRailsEnvV0, "on")
	t.Setenv(DetailProhibitedRailsScopeEnvV0, "codex_wave_tail.*")

	input := strings.Join([]string{
		`access_token=abc123`,
		`Authorization: Bearer token-real`,
		`prompt=texto completo no publicable`,
		`/home/alberto/proyecto/secreto`,
		`{"client_secret":"valor"}`,
	}, "\n")
	got, changed := RedactOperationalTextForFieldV0("codex_wave_tail", "stdout", input)
	if !changed {
		t.Fatalf("redaction not applied")
	}
	for _, forbidden := range []string{"abc123", "token-real", "texto completo", "/home/alberto", "valor"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("forbidden %q leaked in %q", forbidden, got)
		}
	}
	for _, want := range []string{"access_token=<redacted>", "Bearer <redacted>", "<home-path-redacted>"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}
