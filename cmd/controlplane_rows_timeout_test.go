package cmd

import (
	"errors"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestBuildPanelRowsForControlPlaneTimeout(t *testing.T) {
	prevFetcher := statusRowsFetcher
	prevTimeout := controlPlanePanelRowsTimeout
	defer func() {
		statusRowsFetcher = prevFetcher
		controlPlanePanelRowsTimeout = prevTimeout
	}()

	block := make(chan struct{})
	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		<-block
		return nil, nil
	}
	controlPlanePanelRowsTimeout = 20 * time.Millisecond

	start := time.Now()
	rows, err := buildPanelRowsForControlPlane()
	elapsed := time.Since(start)
	close(block)
	if !errors.Is(err, errStatusFetchTimeout) {
		t.Fatalf("deberia devolver timeout, rows=%+v err=%v", rows, err)
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("deberia degradar rapido, elapsed=%s", elapsed)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchDetalladoDegradaSiRowsTimeout(t *testing.T) {
	prepararDBTemporalCmd(t)

	prevRowsFetcher := statusRowsFetcher
	prevRowsTimeout := controlPlanePanelRowsTimeout
	prevReanimations := runtimeProcessReanimationsBatchFn
	defer func() {
		statusRowsFetcher = prevRowsFetcher
		controlPlanePanelRowsTimeout = prevRowsTimeout
		runtimeProcessReanimationsBatchFn = prevReanimations
	}()

	block := make(chan struct{})
	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		<-block
		return nil, nil
	}
	controlPlanePanelRowsTimeout = 20 * time.Millisecond
	runtimeProcessReanimationsBatchFn = func() apiRuntimeProcessReanimationsResponse {
		return apiRuntimeProcessReanimationsResponse{
			OK:          true,
			Reactivated: 2,
		}
	}

	got, err := procesarAgentesDegradadosAutonomiaBatchDetallado()
	close(block)
	if err != nil {
		t.Fatalf("procesarAgentesDegradadosAutonomiaBatchDetallado: %v", err)
	}
	if got.Count != 2 || got.ReactivatedWithoutRuntime != 0 || got.GhostAssignmentsCompacted != 0 {
		t.Fatalf("deberia degradar limpio conservando solo reanimaciones, got=%+v", got)
	}
}

func TestProcesarDecisionEsperarRecuperacionRuntimeSesionActivaAutonomiaOmiteRowsTimeout(t *testing.T) {
	prepararDBTemporalCmd(t)

	prevRowsFetcher := statusRowsFetcher
	prevRowsTimeout := controlPlanePanelRowsTimeout
	prevRecovery := procesarRecuperacionRuntimeDegradadoSesionFn
	defer func() {
		statusRowsFetcher = prevRowsFetcher
		controlPlanePanelRowsTimeout = prevRowsTimeout
		procesarRecuperacionRuntimeDegradadoSesionFn = prevRecovery
	}()

	block := make(chan struct{})
	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		<-block
		return nil, nil
	}
	controlPlanePanelRowsTimeout = 20 * time.Millisecond
	procesarRecuperacionRuntimeDegradadoSesionFn = func(_ *db.Sesion) (int, error) {
		return 0, nil
	}

	proyectoID := int64(7)
	sesion := &db.Sesion{Agente: "Codex3", ProyectoID: &proyectoID, Estado: "activa"}
	proyecto := &db.Proyecto{ID: proyectoID, Slug: "orquestador"}
	n, err := procesarDecisionEsperarRecuperacionRuntimeSesionActivaAutonomia(sesion, proyecto)
	close(block)
	if err != nil {
		t.Fatalf("procesarDecisionEsperarRecuperacionRuntimeSesionActivaAutonomia: %v", err)
	}
	if n != 0 {
		t.Fatalf("deberia omitir filas timeout sin actuar, got=%d", n)
	}
}

func TestIntentarReasignacionAutomaticaBlockedSignalTranscriptOmiteRowsTimeout(t *testing.T) {
	prepararDBTemporalCmd(t)

	prevRowsFetcher := statusRowsFetcher
	prevRowsTimeout := controlPlanePanelRowsTimeout
	defer func() {
		statusRowsFetcher = prevRowsFetcher
		controlPlanePanelRowsTimeout = prevRowsTimeout
	}()

	block := make(chan struct{})
	statusRowsFetcher = func() ([]agentesapp.Row, error) {
		<-block
		return nil, nil
	}
	controlPlanePanelRowsTimeout = 20 * time.Millisecond

	proyectoID := int64(9)
	agente := "Codex3"
	item := &db.RuntimeTranscriptEntry{Agente: "Codex3"}
	proyecto := &db.Proyecto{ID: proyectoID, Slug: "orquestador"}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Blocked transcript",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Estado:      db.TareaEnProgreso,
		Agente:      &agente,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "test",
	})
	if err != nil {
		close(block)
		t.Fatalf("crear tarea: %v", err)
	}
	if tareaID <= 0 {
		close(block)
		t.Fatalf("tarea inesperada: %d", tareaID)
	}

	note, reassigned, err := intentarReasignacionAutomaticaBlockedSignalTranscript(item, proyecto)
	close(block)
	if err != nil {
		t.Fatalf("intentarReasignacionAutomaticaBlockedSignalTranscript: %v", err)
	}
	if note != "" || reassigned {
		t.Fatalf("deberia degradar limpio si rows timeouta, got note=%q reassigned=%t", note, reassigned)
	}
}
