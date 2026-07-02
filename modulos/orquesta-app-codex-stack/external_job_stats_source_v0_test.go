package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
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

func TestCodexStackExternalJobStatsSourceV0NoCompletaPadreEntregadoConWriteSetEstrecho(t *testing.T) {
	fixture := newCodexStackExternalJobSubrolesStatsFixtureV0(true)
	fixture.run.DeliveredTasks = append(fixture.run.DeliveredTasks, fixture.parent.TaskID)
	fixture.run.ClosedTasks = append(fixture.run.ClosedTasks, fixture.parent.TaskID)
	fixture.source.RunStore = orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.run)

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

func TestCodexStackExternalJobStatsSourceV0ExponeColisionAckPadreSubrol(t *testing.T) {
	fixture := newCodexStackExternalJobSubrolesStatsFixtureV0(false)
	fixture.run.DeliveredTasks = []string{fixture.parent.TaskID}
	fixture.run.ClosedTasks = []string{fixture.parent.ChildTaskRefs[0]}
	fixture.run.ReviewResults = []string{
		"review-result-ref-parent-subrole-collision-001#review_result:changes_requested#gate-issue:" +
			orquestaruntimecodex.CodexAgentAckInvalidParentSubroleCollisionEvidenceV0,
	}
	fixture.source.RunStore = orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.run)

	stats, ok, err := fixture.source.ResolveDirectorExternalJobStatsV0(
		context.Background(),
		fixture.request,
	)

	if err != nil {
		t.Fatalf("ResolveDirectorExternalJobStatsV0: %v", err)
	}
	if !ok ||
		stats.Status != codexStackExternalJobStatusParentRunningWithChildAckCollisionV0 ||
		stats.StatusReason != codexStackExternalJobStatusReasonChildAckCollisionV0 ||
		!codexStackStringInSetForTestV0(stats.IssueRefs, fixture.parent.TaskID) ||
		!codexStackExternalJobDiagnosticForTestV0(stats.Diagnostics, codexStackExternalJobStatusParentRunningWithChildAckCollisionV0) {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestCodexStackExternalJobStatsSourceV0PriorizaLostSobreStarted(t *testing.T) {
	fixture := newCodexStackExternalJobSubrolesStatsFixtureV0(false)
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(fixture.parent.TaskID)
	fixture.run.StartedAgents = []string{agentRef}
	fixture.run.LostAgents = []string{agentRef}
	fixture.run.DeliveredTasks = nil
	fixture.run.ClosedTasks = nil
	fixture.source.RunStore = orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.run)

	stats, ok, err := fixture.source.ResolveDirectorExternalJobStatsV0(
		context.Background(),
		fixture.request,
	)

	if err != nil {
		t.Fatalf("ResolveDirectorExternalJobStatsV0: %v", err)
	}
	if !ok || stats.Status != "lost" {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestCodexStackExternalJobStatsSourceV0GoalFirstRunningNoQuedaRegistered(t *testing.T) {
	fixture := newCodexStackExternalJobGoalFirstStatsFixtureV0(t, orquestagoal.GoalStatusRunningV0, false)

	stats, ok, err := fixture.source.ResolveDirectorExternalJobStatsV0(
		context.Background(),
		fixture.request,
	)

	if err != nil {
		t.Fatalf("ResolveDirectorExternalJobStatsV0: %v", err)
	}
	if !ok ||
		stats.Status != "running" ||
		stats.StatusReason != codexStackExternalJobStatusReasonGoalFirstRunningV0 ||
		stats.DirectorExecutionMode != "goal_first" ||
		stats.GoalRef != fixture.goalRef ||
		stats.TaskRef != "" ||
		stats.AgentRef != "" ||
		!codexStackExternalJobDiagnosticForTestV0(stats.Diagnostics, codexStackExternalJobStatusReasonGoalFirstRunningV0) {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestCodexStackExternalJobStatsSourceV0GoalFirstProyectaMetadataOPESAudioV0(t *testing.T) {
	fixture := newCodexStackExternalJobGoalFirstStatsFixtureV0(t, orquestagoal.GoalStatusBlockedV0, false)
	state, err := fixture.source.GoalStateStore.LoadGoalWorkStateV0(context.Background(), fixture.runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	state.Spec.ContextRefs = append(state.Spec.ContextRefs,
		orquestagoal.GoalContextRefV0{Ref: "current_phase=tts", Purpose: "fase actual OPES"},
		orquestagoal.GoalContextRefV0{Ref: "provider_timeout=true", Purpose: "timeout proveedor audio"},
		orquestagoal.GoalContextRefV0{Ref: "domain_counters={\"segments_pending\":7}", Purpose: "contadores audio"},
	)
	if err := fixture.source.GoalStateStore.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	stats, ok, err := fixture.source.ResolveDirectorExternalJobStatsV0(
		context.Background(),
		fixture.request,
	)

	if err != nil {
		t.Fatalf("ResolveDirectorExternalJobStatsV0: %v", err)
	}
	if !ok ||
		stats.Status != "blocked" ||
		stats.CurrentPhase != "tts" ||
		stats.RetryFromPhase != "tts" ||
		stats.OperationalReason != "provider_timeout" ||
		stats.DomainCounters["segments_pending"] != 7 {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestCodexStackExternalJobStatsSourceV0GoalFirstAceptadoCompletaJob(t *testing.T) {
	fixture := newCodexStackExternalJobGoalFirstStatsFixtureV0(t, orquestagoal.GoalStatusCompleteV0, true)

	stats, ok, err := fixture.source.ResolveDirectorExternalJobStatsV0(
		context.Background(),
		fixture.request,
	)

	if err != nil {
		t.Fatalf("ResolveDirectorExternalJobStatsV0: %v", err)
	}
	if !ok ||
		stats.Status != "completed" ||
		stats.StatusReason != codexStackExternalJobStatusReasonGoalFirstClosureAcceptedV0 ||
		!stats.ClosureAccepted ||
		stats.ClosureStatus != orquestagoal.GoalStatusAcceptedV0 ||
		!codexStackStringInSetForTestV0(stats.DeliveryRefs, "domain-receipt-ref-goal-first-external-job-001") ||
		!codexStackExternalJobDiagnosticForTestV0(stats.Diagnostics, codexStackExternalJobStatusReasonGoalFirstClosureAcceptedV0) {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestCodexStackExternalJobStatsSourceV0GoalFirstAcceptedConIssuesNoCompletaJob(t *testing.T) {
	fixture := newCodexStackExternalJobGoalFirstStatsFixtureV0(t, orquestagoal.GoalStatusCompleteV0, true)
	state, err := fixture.source.GoalStateStore.LoadGoalWorkStateV0(context.Background(), fixture.runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	state.LastClosure.Issues = []orquestagoal.GoalWorkIssueV0{{
		Code:  codexStackOPESFinalPackageTopicQualityMissingIssueV0,
		Field: goalDomainReceiptOPESFinalPackageTopicQualityFieldV0,
	}}
	if err := fixture.source.GoalStateStore.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	stats, ok, err := fixture.source.ResolveDirectorExternalJobStatsV0(
		context.Background(),
		fixture.request,
	)

	if err != nil {
		t.Fatalf("ResolveDirectorExternalJobStatsV0: %v", err)
	}
	if !ok ||
		stats.Status != "blocked" ||
		stats.StatusReason != codexStackExternalJobStatusReasonGoalFirstClosureBlockedV0 ||
		!stats.ClosureAccepted ||
		!codexStackExternalJobDiagnosticForTestV0(stats.Diagnostics, codexStackExternalJobStatusReasonGoalFirstClosureBlockedV0) {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestCodexStackExternalJobStatsSourceV0GoalFirstAcceptedSinReceiptNoCompletaJob(t *testing.T) {
	fixture := newCodexStackExternalJobGoalFirstStatsFixtureV0(t, orquestagoal.GoalStatusCompleteV0, true)
	state, err := fixture.source.GoalStateStore.LoadGoalWorkStateV0(context.Background(), fixture.runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	state.LastResult.DomainReceiptRefs = nil
	if err := fixture.source.GoalStateStore.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	stats, ok, err := fixture.source.ResolveDirectorExternalJobStatsV0(
		context.Background(),
		fixture.request,
	)

	if err != nil {
		t.Fatalf("ResolveDirectorExternalJobStatsV0: %v", err)
	}
	if !ok ||
		stats.Status != "blocked" ||
		stats.StatusReason != codexStackExternalJobStatusReasonGoalFirstClosureMissingDomainReceiptV0 ||
		!stats.ClosureAccepted ||
		codexStackStringInSetForTestV0(stats.DeliveryRefs, "domain-receipt-ref-goal-first-external-job-001") ||
		!codexStackExternalJobDiagnosticForTestV0(stats.Diagnostics, codexStackExternalJobStatusReasonGoalFirstClosureMissingDomainReceiptV0) {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestCodexStackExternalJobStatsSourceV0GoalFirstSinStateNoPareceLegacyRegistrado(t *testing.T) {
	fixture := newCodexStackExternalJobGoalFirstStatsFixtureV0(t, orquestagoal.GoalStatusRunningV0, false)
	fixture.source.GoalStateStore = newGoalFirstQueueStateStoreForTestV0()

	stats, ok, err := fixture.source.ResolveDirectorExternalJobStatsV0(
		context.Background(),
		fixture.request,
	)

	if err != nil {
		t.Fatalf("ResolveDirectorExternalJobStatsV0: %v", err)
	}
	if !ok ||
		stats.Status != codexStackExternalJobStatusGoalStateMissingV0 ||
		stats.StatusReason != codexStackExternalJobStatusReasonGoalFirstStateMissingV0 ||
		stats.DirectorExecutionMode != "goal_first" ||
		stats.TaskRef != "" ||
		stats.AgentRef != "" ||
		!codexStackExternalJobDiagnosticForTestV0(stats.Diagnostics, codexStackExternalJobStatusReasonGoalFirstStateMissingV0) {
		t.Fatalf("stats=%+v", stats)
	}
}

func TestCodexStackExternalJobStatsSourceV0GoalFirstMarcadoSinStateNoUsaAppSpecLegacy(t *testing.T) {
	fixture := newCodexStackExternalJobGoalFirstStatsFixtureV0(t, orquestagoal.GoalStatusRunningV0, false)
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	saveCodexStackGoalFirstRunMarkerForTestV0(t, context.Background(), goalStates, fixture.runRef)
	fixture.run.AppSpecRef = "app-spec-neutral-domain-work-opes"
	fixture.source.RunStore = orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.run)
	fixture.source.GoalStateStore = newGoalFirstQueueStateStoreForTestV0()
	fixture.source.GoalFirstRunMarkerStore = goalStates

	stats, ok, err := fixture.source.ResolveDirectorExternalJobStatsV0(
		context.Background(),
		fixture.request,
	)

	if err != nil {
		t.Fatalf("ResolveDirectorExternalJobStatsV0: %v", err)
	}
	if !ok ||
		stats.Status != codexStackExternalJobStatusGoalStateMissingV0 ||
		stats.StatusReason != codexStackExternalJobStatusReasonGoalFirstStateMissingV0 ||
		stats.DirectorExecutionMode != "goal_first" ||
		stats.TaskRef != "" ||
		stats.AgentRef != "" ||
		!codexStackStringInSetForTestV0(stats.EvidenceRefs, "evidence-ref-external-job-goal-first-marker") ||
		!codexStackExternalJobDiagnosticForTestV0(stats.Diagnostics, codexStackExternalJobStatusReasonGoalFirstStateMissingV0) {
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

type codexStackExternalJobGoalFirstStatsFixtureV0 struct {
	source  CodexStackExternalJobStatsSourceV0
	request orquestamcp.MCPDirectorExternalJobStatsRequestV0
	runRef  string
	goalRef string
	run     orquestacoreworkflow.OrchestrationRunV0
}

func newCodexStackExternalJobGoalFirstStatsFixtureV0(
	t *testing.T,
	status string,
	closureAccepted bool,
) codexStackExternalJobGoalFirstStatsFixtureV0 {
	t.Helper()
	const (
		runRef    = "run-ref-external-job-goal-first-001"
		changeRef = "opes-job-job-ref-external-job-goal-first-001"
		jobRef    = "job-ref-external-job-goal-first-001"
	)
	goalRef := "goal-ref-external-job-goal-first-001"
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "opes",
		AppSpecRef:    "app-spec-external-work-opes",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases:        orquestacoreworkflow.OrchestrationPhaseCatalogV0(),
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
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	state := orquestagoal.GoalWorkStateV0{
		SchemaVersion:   orquestagoal.GoalWorkStateSchemaV0,
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: "thread-ref-external-job-goal-first-001",
		Status:          status,
		Spec: orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       goalRef,
			RunRef:        runRef,
			ProjectRef:    "opes",
			DomainRef:     "opes",
			WorkKind:      "draft_content_block",
			Objective:     "Resolver trabajo externo OPES por goal-first.",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet: []orquestagoal.GoalWriteScopeV0{{
				Path: "external/opes/draft_content_block/job-ref-external-job-goal-first-001",
			}},
			AcceptanceCriteria: []string{"entrega OPES trazable"},
			ArtifactContracts: []orquestagoal.GoalArtifactContractV0{{
				ArtifactRef:  "artifact-ref-goal-first-external-job-001",
				ArtifactType: "content_block",
				Required:     true,
			}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         goalRef,
			ExternalGoalRef: "thread-ref-external-job-goal-first-001",
			EvidenceRefs:    []string{"evidence-ref-goal-first-external-job-launch"},
		},
		EvidenceRefs: []string{"evidence-ref-goal-first-external-job-state"},
	}
	if closureAccepted {
		state.LastResult = &orquestagoal.GoalWorkResultV0{
			SchemaVersion:     orquestagoal.GoalWorkResultSchemaV0,
			Status:            orquestagoal.GoalStatusCompleteV0,
			GoalRef:           goalRef,
			ArtifactRefs:      []string{"artifact-ref-goal-first-external-job-001"},
			DomainReceiptRefs: []string{"domain-receipt-ref-goal-first-external-job-001"},
			EvidenceRefs:      []string{"evidence-ref-goal-first-external-job-result"},
		}
		state.LastClosure = &orquestagoal.GoalClosureValidationV0{
			Status:       orquestagoal.GoalStatusAcceptedV0,
			Accepted:     true,
			EvidenceRefs: []string{"evidence-ref-goal-first-external-job-closure"},
		}
	}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	return codexStackExternalJobGoalFirstStatsFixtureV0{
		source: CodexStackExternalJobStatsSourceV0{
			RunStore:                orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			AppChangeStore:          orquestaappchange.NewInMemoryAppChangeStoreV0(record),
			ReceiptStore:            orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
			TaskStore:               orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(),
			GoalStateStore:          goalStates,
			GoalFirstRunMarkerStore: goalStates,
		},
		request: orquestamcp.MCPDirectorExternalJobStatsRequestV0{
			AppRef:         "opes",
			ExternalJobRef: jobRef,
		},
		runRef:  runRef,
		goalRef: goalRef,
		run:     run,
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
