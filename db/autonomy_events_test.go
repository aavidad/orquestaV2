package db

import "testing"

func TestRegistrarYListarAutonomyEventsSobreAuditLog(t *testing.T) {
	prepararDBTemporal(t)

	projectID := int64(7)
	taskID := int64(11)
	runtimeID := int64(13)
	handleID := int64(17)
	err := RegistrarAutonomyEvent(&AutonomyEvent{
		Kind:      "handoff_requested",
		Actor:     "orquesta",
		ProjectID: &projectID,
		TaskID:    &taskID,
		RuntimeID: &runtimeID,
		HandleID:  &handleID,
		Source:    "control_plane",
		Reason:    "bloqueado_por_runtime",
		StateDelta: map[string]any{
			"last_autonomy_state":  "handoff",
			"continuity_pending":   false,
			"runtime_state_before": "degradado",
		},
		ArtifactsRef: []string{"artifact:test:1"},
	})
	if err != nil {
		t.Fatalf("registrar autonomy event: %v", err)
	}

	kind := "handoff_requested"
	items, err := ListarAutonomyEvents(FiltroAutonomyEvents{Kind: &kind, Limite: 10})
	if err != nil {
		t.Fatalf("listar autonomy events: %v", err)
	}
	if len(items) != 1 || items[0] == nil {
		t.Fatalf("autonomy events inesperados: %+v", items)
	}
	item := items[0]
	if item.Kind != "handoff_requested" || item.Actor != "orquesta" || item.Source != "control_plane" {
		t.Fatalf("evento inesperado: %+v", item)
	}
	if item.TaskID == nil || *item.TaskID != taskID || item.ProjectID == nil || *item.ProjectID != projectID {
		t.Fatalf("ids inesperados: %+v", item)
	}
	if item.StateDelta["last_autonomy_state"] != "handoff" {
		t.Fatalf("state_delta inesperado: %+v", item.StateDelta)
	}
	if len(item.ArtifactsRef) != 1 || item.ArtifactsRef[0] != "artifact:test:1" {
		t.Fatalf("artifacts inesperados: %+v", item.ArtifactsRef)
	}
}

func TestListarAutonomyEventsIgnoraAuditoriaAjena(t *testing.T) {
	prepararDBTemporal(t)
	Audit("Codex1", "accion_normal", "tarea", 1, "detalle")

	items, err := ListarAutonomyEvents(FiltroAutonomyEvents{Limite: 10})
	if err != nil {
		t.Fatalf("listar autonomy events: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("no deberia mezclar auditoria normal con autonomy events: %+v", items)
	}
}
