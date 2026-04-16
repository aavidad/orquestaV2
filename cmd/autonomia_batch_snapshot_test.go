package cmd

import "testing"

func TestAutonomiaBatchSnapshotOperationalStateCacheaPorAgente(t *testing.T) {
	prev := autonomiaOperationalStateResolver
	defer func() { autonomiaOperationalStateResolver = prev }()

	llamadas := 0
	autonomiaOperationalStateResolver = func(agente string) (string, string, error) {
		llamadas++
		if agente != "Codex1" {
			t.Fatalf("agente inesperado: %s", agente)
		}
		return "trabajando", "worker fresco", nil
	}

	snapshot := &autonomiaBatchSnapshot{
		operationalStateByAgent:  map[string]string{},
		operationalDetailByAgent: map[string]string{},
		operationalStateResolved: map[string]struct{}{},
	}

	estado, detalle, err := snapshot.operationalState("Codex1")
	if err != nil {
		t.Fatalf("operationalState first: %v", err)
	}
	if estado != "trabajando" || detalle != "worker fresco" {
		t.Fatalf("estado/detalle inesperados: %q / %q", estado, detalle)
	}
	estado, detalle, err = snapshot.operationalState("Codex1")
	if err != nil {
		t.Fatalf("operationalState second: %v", err)
	}
	if estado != "trabajando" || detalle != "worker fresco" {
		t.Fatalf("estado/detalle inesperados en cache: %q / %q", estado, detalle)
	}
	if llamadas != 1 {
		t.Fatalf("resolver deberia llamarse una sola vez, got=%d", llamadas)
	}
}
