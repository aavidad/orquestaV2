package orquestaappcodexstack

import "testing"

func TestCodexAreaV0ClasificaImplementacionComoProgramacion(t *testing.T) {
	if got := codexAreaV0("implementacion", "task-ref-stack-agenda-001"); got != "programacion" {
		t.Fatalf("area=%s", got)
	}
}
