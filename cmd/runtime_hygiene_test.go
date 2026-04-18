package cmd

import (
	"testing"
	"time"

	"orquesta/db"
)

func TestProcesarRuntimeHygieneBatchDifierePurgaHistoricaEnRunner(t *testing.T) {
	resetRuntimeHistoricalMaintenanceGate()
	t.Cleanup(resetRuntimeHistoricalMaintenanceGate)

	prevProcesar := procesarHigieneRuntimesAutonomosBatchFn
	prevReconciliar := reconciliarRuntimeOrdersPendientesHandoffExpiradasFn
	prevTranscript := purgarRuntimeTranscriptRuidoHistoricoFn
	prevHistorico := purgarRuntimeHistoricoFn
	prevOperacional := purgarDatosOperacionalesFn
	t.Cleanup(func() {
		procesarHigieneRuntimesAutonomosBatchFn = prevProcesar
		reconciliarRuntimeOrdersPendientesHandoffExpiradasFn = prevReconciliar
		purgarRuntimeTranscriptRuidoHistoricoFn = prevTranscript
		purgarRuntimeHistoricoFn = prevHistorico
		purgarDatosOperacionalesFn = prevOperacional
	})

	historicalRuns := 0
	operationalRuns := 0
	procesarHigieneRuntimesAutonomosBatchFn = func() (int, error) { return 1, nil }
	reconciliarRuntimeOrdersPendientesHandoffExpiradasFn = func() (int, error) { return 2, nil }
	purgarRuntimeTranscriptRuidoHistoricoFn = func() (int, error) { return 3, nil }
	purgarRuntimeHistoricoFn = func() (*db.PurgaRuntimeHistoricoResultado, error) {
		historicalRuns++
		return &db.PurgaRuntimeHistoricoResultado{
			Handles: &db.PurgaRuntimeHandlesResultado{Deleted: 4},
			Orders:  &db.PurgaRuntimeOrdersResultado{Deleted: 5},
		}, nil
	}
	purgarDatosOperacionalesFn = func() (int, error) {
		operationalRuns++
		return 6, nil
	}

	first, err := (dbAutomationService{}).ProcesarRuntimeHygieneBatch()
	if err != nil {
		t.Fatalf("first batch: %v", err)
	}
	if first != 21 {
		t.Fatalf("first=%d want 21", first)
	}

	second, err := (dbAutomationService{}).ProcesarRuntimeHygieneBatch()
	if err != nil {
		t.Fatalf("second batch: %v", err)
	}
	if second != 6 {
		t.Fatalf("second=%d want 6", second)
	}
	if historicalRuns != 1 {
		t.Fatalf("historicalRuns=%d want 1", historicalRuns)
	}
	if operationalRuns != 1 {
		t.Fatalf("operationalRuns=%d want 1", operationalRuns)
	}
}

func TestProcesarRuntimeHygieneBatchForceHistoricalIgnoraGate(t *testing.T) {
	resetRuntimeHistoricalMaintenanceGate()
	runtimeHistoricalMaintenanceGate.Set(time.Now().UTC().Add(10 * time.Minute))
	t.Cleanup(resetRuntimeHistoricalMaintenanceGate)

	prevProcesar := procesarHigieneRuntimesAutonomosBatchFn
	prevReconciliar := reconciliarRuntimeOrdersPendientesHandoffExpiradasFn
	prevTranscript := purgarRuntimeTranscriptRuidoHistoricoFn
	prevHistorico := purgarRuntimeHistoricoFn
	prevOperacional := purgarDatosOperacionalesFn
	t.Cleanup(func() {
		procesarHigieneRuntimesAutonomosBatchFn = prevProcesar
		reconciliarRuntimeOrdersPendientesHandoffExpiradasFn = prevReconciliar
		purgarRuntimeTranscriptRuidoHistoricoFn = prevTranscript
		purgarRuntimeHistoricoFn = prevHistorico
		purgarDatosOperacionalesFn = prevOperacional
	})

	historicalRuns := 0
	procesarHigieneRuntimesAutonomosBatchFn = func() (int, error) { return 0, nil }
	reconciliarRuntimeOrdersPendientesHandoffExpiradasFn = func() (int, error) { return 0, nil }
	purgarRuntimeTranscriptRuidoHistoricoFn = func() (int, error) { return 0, nil }
	purgarRuntimeHistoricoFn = func() (*db.PurgaRuntimeHistoricoResultado, error) {
		historicalRuns++
		return &db.PurgaRuntimeHistoricoResultado{}, nil
	}
	purgarDatosOperacionalesFn = func() (int, error) { return 0, nil }

	if _, err := procesarRuntimeHygieneBatch(true); err != nil {
		t.Fatalf("force historical: %v", err)
	}
	if historicalRuns != 1 {
		t.Fatalf("historicalRuns=%d want 1", historicalRuns)
	}
}
