package cmd

import (
	"testing"

	"orquesta/db"
)

func TestAutonomiaBatchSnapshotBloqueoSummaryMapCacheaResultado(t *testing.T) {
	prepararDBTemporalCmd(t)

	snapshot := &autonomiaBatchSnapshot{
		bloqueosPorTarea: map[int64]db.ResumenBloqueo{},
	}

	got, err := snapshot.bloqueoSummaryMap()
	if err != nil {
		t.Fatalf("bloqueoSummaryMap first: %v", err)
	}
	if got == nil {
		t.Fatal("bloqueoSummaryMap deberia devolver mapa no nil")
	}

	snapshot.bloqueosPorTarea[77] = db.ResumenBloqueo{ID: 77, Motivo: "cacheado"}
	got, err = snapshot.bloqueoSummaryMap()
	if err != nil {
		t.Fatalf("bloqueoSummaryMap second: %v", err)
	}
	if got[77].Motivo != "cacheado" {
		t.Fatalf("deberia reutilizar cache cargada: %+v", got[77])
	}
}
