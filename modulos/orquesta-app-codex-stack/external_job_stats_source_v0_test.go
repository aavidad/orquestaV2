package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackExternalJobStatsSourceV0MarcaIntegracionPendienteTrasSubroles(t *testing.T) {
	fixture := newCodexStackExternalJobSubrolesStatsFixtureV0(false)

	stats, ok, err := fixture.source.ResolveDirectorExternalJobStatsV0(
		context.Background(),
		fixture.request,
	)

	if err != nil {
		t.Fatalf("ResolveDirectorExternalJobStatsV0: %v", err)
	}
	if !ok ||
		stats.Status != codexStackExternalJobStatusIntegrationRequiredV0 ||
		stats.StatusReason != codexStackExternalJobStatusReasonParentIntegrationPendingV0 ||
		!codexStackStringInSetForTestV0(stats.IssueRefs, fixture.parent.TaskID) ||
		!codexStackExternalJobDiagnosticForTestV0(stats.Diagnostics, "external_job_parent_integration_pending") {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestCodexStackExternalJobStatsSourceV0ExponeNarrowingLegacyComoRazon(t *testing.T) {
	fixture := newCodexStackExternalJobSubrolesStatsFixtureV0(true)

	stats, ok, err := fixture.source.ResolveDirectorExternalJobStatsV0(
		context.Background(),
		fixture.request,
	)

	if err != nil {
		t.Fatalf("ResolveDirectorExternalJobStatsV0: %v", err)
	}
	if !ok ||
		stats.Status != codexStackExternalJobStatusIntegrationRequiredV0 ||
		stats.StatusReason != codexStackExternalJobStatusReasonProductNotConsolidatedDueWriteSetNarrowingV0 ||
		!codexStackExternalJobDiagnosticForTestV0(stats.Diagnostics, codexStackExternalJobStatusReasonProductNotConsolidatedDueWriteSetNarrowingV0) {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestCodexStackExternalJobStatsSourceV0DistingueParentAckConCohorteAbierta(t *testing.T) {
	fixture := newCodexStackExternalJobSubrolesStatsFixtureV0(false)
	fixture.run.DeliveredTasks = []string{fixture.parent.TaskID}
	fixture.run.ClosedTasks = []string{fixture.parent.ChildTaskRefs[0]}
	fixture.source.RunStore = orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.run)

	stats, ok, err := fixture.source.ResolveDirectorExternalJobStatsV0(
		context.Background(),
		fixture.request,
	)

	if err != nil {
		t.Fatalf("ResolveDirectorExternalJobStatsV0: %v", err)
	}
	if !ok ||
		stats.Status != codexStackExternalJobStatusParentAckReceivedV0 ||
		stats.StatusReason != codexStackExternalJobStatusReasonCohortOpenV0 ||
		!codexStackExternalJobDiagnosticForTestV0(stats.Diagnostics, "external_job_parent_ack_received_cohort_open") {
		t.Fatalf("stats=%+v", stats)
	}
}

type codexStackExternalJobSubrolesStatsFixtureV0 struct {
	source  CodexStackExternalJobStatsSourceV0
	request orquestamcp.MCPDirectorExternalJobStatsRequestV0
	parent  orquestacoreworkflow.WorkflowTaskV0
	run     orquestacoreworkflow.OrchestrationRunV0
}

func newCodexStackExternalJobSubrolesStatsFixtureV0(
	coordinationOnlyParent bool,
) codexStackExternalJobSubrolesStatsFixtureV0 {
	const (
		runRef    = "run-ref-external-job-stats-subroles-001"
		changeRef = "opes-job-job-ref-external-job-stats-subroles-001"
		jobRef    = "job-ref-external-job-stats-subroles-001"
	)
	parentRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(changeRef)
	childRefs := []string{
		parentRef + "-subrole-fuentes",
		parentRef + "-subrole-reutilizacion",
		parentRef + "-subrole-redaccion",
		parentRef + "-subrole-visuales",
		parentRef + "-subrole-tests-tutor",
		parentRef + "-subrole-html-rag-audio-qa",
	}
	parentWriteSet := []string{
		"temas/tema_032",
		"temas/tema_032/coordinacion",
	}
	if coordinationOnlyParent {
		parentWriteSet = []string{"temas/tema_032/coordinacion"}
	}
	parent := codexStackExternalJobStatsTaskForTestV0(runRef, parentRef, "", parentWriteSet)
	parent.ChildTaskRefs = append([]string(nil), childRefs...)
	parent.MaxChildAgents = len(childRefs)
	parent.DependsOn = append([]string(nil), childRefs...)
	tasks := []orquestacoreworkflow.WorkflowTaskV0{parent}
	for _, childRef := range childRefs {
		child := codexStackExternalJobStatsTaskForTestV0(
			runRef,
			childRef,
			parentRef,
			[]string{"temas/tema_032/subroles/redaccion"},
		)
		tasks = append(tasks, child)
	}
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:  orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:          runRef,
		ProjectRef:     "opes",
		AppSpecRef:     "opes",
		Status:         orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:   orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:          append([]string{parentRef}, childRefs...),
		DeliveredTasks: append([]string(nil), childRefs...),
		ClosedTasks:    append([]string(nil), childRefs...),
	}
	record := orquestaappchange.AppChangeRecordV0{
		Request: orquestaappchange.AppChangeRequestV0{
			RunRef:    runRef,
			AppRef:    "opes",
			ChangeRef: changeRef,
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef: "opes",
				JobRef:     jobRef,
				WorkKind:   "draft_content_block",
			},
		},
	}
	return codexStackExternalJobSubrolesStatsFixtureV0{
		source: CodexStackExternalJobStatsSourceV0{
			RunStore:       orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			AppChangeStore: orquestaappchange.NewInMemoryAppChangeStoreV0(record),
			ReceiptStore:   orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
			TaskStore:      orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(tasks...),
		},
		request: orquestamcp.MCPDirectorExternalJobStatsRequestV0{
			AppRef:         "opes",
			ExternalJobRef: jobRef,
		},
		parent: parent,
		run:    run,
	}
}

func codexStackExternalJobStatsTaskForTestV0(
	runRef string,
	taskRef string,
	parentRef string,
	writeSet []string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             taskRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileDomainWorkV0,
		Title:              "Trabajo externo OPES",
		WriteSet:           append([]string(nil), writeSet...),
		AcceptanceCriteria: []string{"entrega de dominio trazable"},
		ParentTaskRef:      parentRef,
	}
}

func codexStackExternalJobDiagnosticForTestV0(
	diagnostics []orquestamcp.MCPDirectorExternalJobDiagnosticV0,
	code string,
) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
