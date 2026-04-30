package cmd

import (
	"testing"
	"time"

	"orquesta/db"
)

func TestBuildOpenClawOperationalInfoWithStatusUsaFallbackCanonicoSiTimeout(t *testing.T) {
	prevFetcher := controlPlaneOperationalInfoFetcher
	prevTimeout := openClawOperationalInfoTimeout
	t.Cleanup(func() {
		controlPlaneOperationalInfoFetcher = prevFetcher
		openClawOperationalInfoTimeout = prevTimeout
	})

	controlPlaneOperationalInfoFetcher = func() (serverOperationalInfo, error) {
		time.Sleep(40 * time.Millisecond)
		return serverOperationalInfo{State: "ready", Operational: true, ActiveAgents: 99}, nil
	}
	openClawOperationalInfoTimeout = 10 * time.Millisecond

	info := buildOpenClawOperationalInfoWithStatus(apiStatusResponse{
		Generado:          "2026-04-29T08:31:00Z",
		AgentesActivos:    []*db.Agente{{Nombre: "Codex4", Activo: true}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex4", Activo: true}},
		TareasEnProgreso:  []tareaLite{{ID: 40, Agente: "Codex4", Estado: db.TareaEnProgreso}},
	})

	if info.ActiveAgents != 1 || info.WorkingAgents != 1 {
		t.Fatalf("deberia caer al status canónico si timeout: %+v", info)
	}
	if !info.Operational || info.State != "ready" {
		t.Fatalf("fallback canónico inesperado: %+v", info)
	}
}

func TestBuildOpenClawOperationalInfoWithStatusPrefiereFetcherSano(t *testing.T) {
	prevFetcher := controlPlaneOperationalInfoFetcher
	prevTimeout := openClawOperationalInfoTimeout
	t.Cleanup(func() {
		controlPlaneOperationalInfoFetcher = prevFetcher
		openClawOperationalInfoTimeout = prevTimeout
	})

	controlPlaneOperationalInfoFetcher = func() (serverOperationalInfo, error) {
		return serverOperationalInfo{
			State:            "ready",
			Operational:      true,
			Reason:           "control_plane_responsive",
			ActiveAgents:     5,
			WorkingAgents:    5,
			ConnectedWorkers: 4,
			WorkingWorkers:   4,
			Generated:        "2026-04-29T08:31:05Z",
		}, nil
	}
	openClawOperationalInfoTimeout = 50 * time.Millisecond

	info := buildOpenClawOperationalInfoWithStatus(apiStatusResponse{
		Generado:          "2026-04-29T08:31:00Z",
		AgentesActivos:    []*db.Agente{{Nombre: "Codex4", Activo: true}},
		AgentesTrabajando: []*db.Agente{{Nombre: "Codex4", Activo: true}},
		TareasEnProgreso:  []tareaLite{{ID: 40, Agente: "Codex4", Estado: db.TareaEnProgreso}},
	})

	if info.ActiveAgents != 5 || info.WorkingAgents != 5 || info.Generated != "2026-04-29T08:31:05Z" {
		t.Fatalf("deberia preferir fetcher sano: %+v", info)
	}
}

func TestBuildOpenClawOperationalInfoWithStatusPrefiereStatusMasFrescoQueFetcherViejo(t *testing.T) {
	prevFetcher := controlPlaneOperationalInfoFetcher
	prevTimeout := openClawOperationalInfoTimeout
	t.Cleanup(func() {
		controlPlaneOperationalInfoFetcher = prevFetcher
		openClawOperationalInfoTimeout = prevTimeout
	})

	controlPlaneOperationalInfoFetcher = func() (serverOperationalInfo, error) {
		return serverOperationalInfo{
			State:            "ready",
			Operational:      true,
			Reason:           "control_plane_responsive",
			ActiveAgents:     2,
			WorkingAgents:    2,
			ConnectedWorkers: 1,
			WorkingWorkers:   1,
			Generated:        "2026-04-29T08:31:00Z",
		}, nil
	}
	openClawOperationalInfoTimeout = 50 * time.Millisecond

	info := buildOpenClawOperationalInfoWithStatus(apiStatusResponse{
		Generado:         "2026-04-29T08:31:10Z",
		TareasEnProgreso: []tareaLite{{ID: 40, Agente: "", Estado: db.TareaEnProgreso}},
	})

	if info.Generated != "2026-04-29T08:31:10Z" || info.Operational || info.State != "degraded" || info.Reason != "tasks_without_workers" {
		t.Fatalf("deberia preferir fallback fresco del status: %+v", info)
	}
}
