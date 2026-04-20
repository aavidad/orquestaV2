package cmd

import (
	"testing"

	"orquesta/db"
)

func TestProcesarRuntimeTranscriptBatchCedeAnteHotPathAPI(t *testing.T) {
	apiActiveTickRequests.Store(1)
	defer apiActiveTickRequests.Store(0)
	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesarRuntimeTranscriptBatch: %v", err)
	}
	if n != 0 {
		t.Fatalf("deberia ceder ante hot path api, count=%d", n)
	}
}

func TestProcesarPresupuestoSesionObservadoBatchCedeAnteHotPathAPI(t *testing.T) {
	apiActiveTickRequests.Store(1)
	defer apiActiveTickRequests.Store(0)
	n, err := procesarPresupuestoSesionObservadoBatch()
	if err != nil {
		t.Fatalf("procesarPresupuestoSesionObservadoBatch: %v", err)
	}
	if n != 0 {
		t.Fatalf("deberia ceder ante hot path api, count=%d", n)
	}
}

func TestProcesarRuntimeMailboxBatchConFiltroCedeAnteHotPathAPI(t *testing.T) {
	apiActiveTickRequests.Store(1)
	defer apiActiveTickRequests.Store(0)
	n, err := procesarRuntimeMailboxBatchConFiltro(db.FiltroRuntimeMailbox{})
	if err != nil {
		t.Fatalf("procesarRuntimeMailboxBatchConFiltro: %v", err)
	}
	if n != 0 {
		t.Fatalf("deberia ceder ante hot path api, count=%d", n)
	}
}
