package orquestaappchangedirectorsource

import (
	"context"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
)

func TestAppChangeDirectorDecisionSourceV0ProyectaVisualAssetOPES(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
		WorkKind:      "generate_visual_asset",
		WorkRefs:      []string{"opes-job-visual-001", "opes-topic-001"},
	}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if task.Title != "Generar recurso visual OPES" ||
		task.Summary != appChangeVisualTaskSummaryV0() ||
		!stringInSetV0(task.WriteSet, "external/opes/generate_visual_asset") ||
		!stringInSetV0(task.AcceptanceCriteria, "Devolver visual_asset con title, caption, alt_text, format y body.") ||
		!stringInSetV0(task.AcceptanceCriteria, "Preferir SVG autocontenido cuando format=svg; sin scripts, eventos JavaScript, foreignObject ni URLs remotas.") ||
		!stringInSetV0(task.AcceptanceCriteria, "No usar placeholders como arte final; si falta contenido visual suficiente, guardar brief o maqueta y nota de rework.") ||
		!stringInSetV0(task.RequiredTests, "validar contrato visual OPES") ||
		!stringInSetV0(task.RequiredTests, "validar SVG seguro si format=svg") ||
		!stringInSetV0(task.RequiredTests, "validar estado de placeholder, maqueta o arte final") {
		t.Fatalf("task=%+v", task)
	}
}

func TestAppChangeDirectorDecisionSourceV0ProyectaVisualAssetExternoSinOPES(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "external-editorial",
		InterfaceRefs: []string{"domain-contract-v0"},
		WorkKind:      "generate_visual_asset",
		WorkRefs:      []string{"job-visual-001", "topic-001"},
	}
	store := orquestaappchange.NewInMemoryAppChangeStoreV0(record)
	run := appChangeRunForSourceTestV0(record)

	decisions, err := (AppChangeDirectorDecisionSourceV0{Store: store}).
		ListDirectorAgentDecisionsV0(
			context.Background(),
			orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{Run: run},
		)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	task := microtaskDecisionForTestV0(t, decisions).CreateMicrotask.Task
	if task.Title != "Generar recurso visual externo" ||
		!stringInSetV0(task.RequiredTests, "validar contrato visual externo") ||
		stringInSetV0(task.RequiredTests, "validar contrato visual OPES") {
		t.Fatalf("task=%+v", task)
	}
}
