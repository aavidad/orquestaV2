package cmd

import (
	"testing"
	"time"
)

func TestPresupuestoObservadoEsDuplicadoReciente(t *testing.T) {
	now := time.Now().UTC()
	if !presupuestoObservadoEsDuplicadoReciente(now.Add(-time.Minute), `{"a":1}`, `{"a":1}`) {
		t.Fatal("deberia considerar duplicada una snapshot identica y reciente")
	}
	if presupuestoObservadoEsDuplicadoReciente(now.Add(-3*time.Minute), `{"a":1}`, `{"a":1}`) {
		t.Fatal("no deberia considerar duplicada una snapshot fuera de ventana")
	}
	if presupuestoObservadoEsDuplicadoReciente(now.Add(-time.Minute), `{"a":1}`, `{"a":2}`) {
		t.Fatal("no deberia considerar duplicadas snapshots distintas")
	}
	if presupuestoObservadoEsDuplicadoReciente(time.Time{}, `{"a":1}`, `{"a":1}`) {
		t.Fatal("sin checked_at previo no deberia deduplicar")
	}
}
