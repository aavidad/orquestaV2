package orquestaappchangedirectorsource

import (
	"context"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
)

func TestAppChangeDirectorDecisionSourceV0ProyectaAudioAssetOPES(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "opes",
		InterfaceRefs: []string{"opes-rest-v0", "opes-mcp-v0"},
		WorkKind:      "generate_audio_asset",
		WorkRefs:      []string{"opes-job-audio-001", "opes-topic-001"},
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
	task := decisions[6].CreateMicrotask.Task
	if task.Title != "Generar audio accesible externo" ||
		task.Summary != appChangeAudioTaskSummaryV0() ||
		!stringInSetV0(task.WriteSet, "external/opes/generate_audio_asset") ||
		!stringInSetV0(task.AcceptanceCriteria, "Devolver audio_asset con manifest de idioma, formato, duracion y refs/checksums de artefactos.") ||
		!stringInSetV0(task.AcceptanceCriteria, "No incluir rutas locales, proveedor, GPU, modelo ni procesos internos en el payload publico.") ||
		!stringInSetV0(task.RequiredTests, "validar contrato audio OPES") ||
		!stringInSetV0(task.RequiredTests, "validar artifact_type=audio_asset") ||
		!stringInSetV0(task.RequiredTests, "validar que no filtra proveedor, GPU, modelo, rutas ni procesos internos") {
		t.Fatalf("task=%+v", task)
	}
}

func TestAppChangeDirectorDecisionSourceV0AudioAssetExternoSinOPES(t *testing.T) {
	record := appChangeRecordForSourceTestV0()
	record.Request.AllowedWriteSet = nil
	record.Request.ExternalWork = &orquestaappchange.AppChangeExternalWorkV0{
		ProjectRef:    "external-editorial",
		InterfaceRefs: []string{"domain-contract-v0"},
		WorkKind:      "audio_tema",
		WorkRefs:      []string{"job-audio-001", "topic-001"},
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
	task := decisions[6].CreateMicrotask.Task
	if task.Title != "Generar audio accesible externo" ||
		!stringInSetV0(task.RequiredTests, "validar contrato audio externo") ||
		stringInSetV0(task.RequiredTests, "validar contrato audio OPES") ||
		containsFragmentInSetForTestV0(task.AcceptanceCriteria, "edge-tts") ||
		containsFragmentInSetForTestV0(task.AcceptanceCriteria, "Microsoft") {
		t.Fatalf("task=%+v", task)
	}
}
