package db

import "testing"

func TestUpsertSupervisorPipelineStateYListar(t *testing.T) {
	prepararDBTemporal(t)
	item, err := UpsertSupervisorPipelineState(UpsertSupervisorPipelineStateInput{
		Supervisor:    "OpenClaw",
		ProyectoSlug:  "orquestador",
		PipelineName:  "autopilot",
		CurrentPhase:  "review",
		Status:        "active",
		ArtifactsJSON: `{"gates":2}`,
		MetadataJSON:  `{"mode":"review"}`,
	})
	if err != nil {
		t.Fatalf("upsert pipeline state: %v", err)
	}
	if item == nil || item.CurrentPhase != "review" {
		t.Fatalf("pipeline state inesperado: %+v", item)
	}
	item, err = UpsertSupervisorPipelineState(UpsertSupervisorPipelineStateInput{
		Supervisor:   "OpenClaw",
		ProyectoSlug: "orquestador",
		PipelineName: "autopilot",
		CurrentPhase: "merge",
		Status:       "blocked",
	})
	if err != nil {
		t.Fatalf("upsert pipeline state 2: %v", err)
	}
	if item == nil || item.CurrentPhase != "merge" || item.Status != "blocked" {
		t.Fatalf("pipeline state 2 inesperado: %+v", item)
	}
	items, err := ListarSupervisorPipelineStates(FiltroSupervisorPipelineStates{
		Supervisor:   "OpenClaw",
		ProyectoSlug: "orquestador",
	})
	if err != nil {
		t.Fatalf("listar pipeline states: %v", err)
	}
	if len(items) != 1 || items[0] == nil || items[0].PipelineName != "autopilot" {
		t.Fatalf("pipeline states inesperados: %+v", items)
	}
}
