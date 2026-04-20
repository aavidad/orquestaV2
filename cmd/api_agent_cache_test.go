package cmd

import (
	"testing"
	"time"

	"orquesta/db"
)

func TestAPINombreAgenteCanonicoCacheaLookups(t *testing.T) {
	prevFn := apiGetAgentFn
	prevTTL := apiAgentCanonicalCacheTTL
	t.Cleanup(func() {
		apiGetAgentFn = prevFn
		apiAgentCanonicalCacheTTL = prevTTL
		apiResetAgentCanonicalCache()
	})
	apiResetAgentCanonicalCache()
	apiAgentCanonicalCacheTTL = time.Minute

	calls := 0
	apiGetAgentFn = func(ref string) (*db.Agente, error) {
		calls++
		return &db.Agente{Nombre: "Codex1"}, nil
	}

	if got := apiNombreAgenteCanonico("codex1"); got != "Codex1" {
		t.Fatalf("nombre canonico inesperado: %q", got)
	}
	if got := apiNombreAgenteCanonico("codex1"); got != "Codex1" {
		t.Fatalf("nombre canonico inesperado en cache: %q", got)
	}
	if got := apiNombreAgenteCanonico("Codex1"); got != "Codex1" {
		t.Fatalf("nombre canonico inesperado en alias canonico: %q", got)
	}
	if calls != 1 {
		t.Fatalf("deberia resolver agente una sola vez, calls=%d", calls)
	}
}
