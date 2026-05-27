package orquestaappdirectorservice

import (
	"context"
	"errors"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorsupervisedburst "orquesta/modulos/orquesta-director-supervised-burst"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestComposeStartAppDirectorProviderV0IncludesReviewReworkReplanSource(t *testing.T) {
	source := &recordingReviewReworkReplanSourceV0{}
	provider := composeStartAppDirectorProviderV0(
		orquestacionnucleoapp.StaticCandidateProviderV0{},
		StartAppDirectorPortsV0{ReviewReworkReplanSource: source},
		"director-service-test",
	)

	_, err := provider.BuildSchedulerCandidatesV0(context.Background(), orquestacionnucleoapp.SchedulerCandidateRequestV0{
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID:        "run-ref-service-review-rework-001",
			CurrentPhase: orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		},
		OccurredAt:    "2026-05-10T10:05:00Z",
		CorrelationID: "corr-service-review-rework-001",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if !source.called {
		t.Fatalf("review rework replan source no fue invocado")
	}
}

func TestComposeStartAppDirectorProviderV0PassesTaskWriterToReviewReworkReplanSource(t *testing.T) {
	runRef := "run-ref-service-review-rework-split-001"
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(serviceReviewOriginalTaskV0(runRef))
	provider := composeStartAppDirectorProviderV0(
		orquestacionnucleoapp.StaticCandidateProviderV0{},
		StartAppDirectorPortsV0{
			DirectorTaskStore:        taskStore,
			ReviewReworkReplanSource: splitReviewReworkReplanSourceV0{},
		},
		"director-service-test",
	)

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), orquestacionnucleoapp.SchedulerCandidateRequestV0{
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID:        runRef,
			CurrentPhase: orquestacoreworkflow.OrchestrationPhaseRevisionV0,
			Tasks:        []string{"task-ref-service-review-original-001"},
			ReworkRequests: []string{
				"rework-request-ref-service-review-split-001#review_result:review-result-ref-service-review-split-001",
			},
		},
		OccurredAt:    "2026-05-10T10:07:00Z",
		CorrelationID: "corr-service-review-rework-split-001",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ReplanFollowupCandidates) != 1 {
		t.Fatalf("replan followups=%d candidates=%+v", len(candidates.ReplanFollowupCandidates), candidates)
	}
	followups := candidates.ReplanFollowupCandidates[0].ReplanFollowupsInput.MicrotaskCandidates
	if len(followups) != 1 || followups[0].Payload.Task.TaskID != "task-ref-service-review-split-a" {
		t.Fatalf("microtareas de replan no compuestas: %+v", followups)
	}
	if _, err := taskStore.LoadWorkflowTasksV0(context.Background(), runRef, []string{"task-ref-service-review-split-a"}); err != nil {
		t.Fatalf("microtarea split no guardada por DirectorTaskStore: %v", err)
	}
}

func TestComposeStartAppDirectorProviderV0IncludesReviewGateSource(t *testing.T) {
	source := &recordingReviewGateSourceV0{}
	provider := composeStartAppDirectorProviderV0(
		orquestacionnucleoapp.StaticCandidateProviderV0{},
		StartAppDirectorPortsV0{ReviewGateSource: source},
		"director-service-test",
	)

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), orquestacionnucleoapp.SchedulerCandidateRequestV0{
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID:        "run-ref-service-review-gate-001",
			CurrentPhase: orquestacoreworkflow.OrchestrationPhaseRevisionV0,
			Deliveries:   []string{"delivery-ref-service-review-gate-001"},
			Agents:       []string{"agent-ref-service-review-gate-001"},
			StartedAgents: []string{
				"agent-ref-service-review-gate-001",
			},
		},
		OccurredAt:    "2026-05-10T10:05:00Z",
		CorrelationID: "corr-service-review-gate-001",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if !source.called || len(candidates.ReviewGateCandidates) != 1 {
		t.Fatalf("review gate no fue invocado: called=%v candidates=%+v", source.called, candidates)
	}
}

func TestComposeStartAppDirectorProviderV0IncludesQualityGateReplanProvider(t *testing.T) {
	provider := composeStartAppDirectorProviderV0(
		orquestacionnucleoapp.StaticCandidateProviderV0{},
		StartAppDirectorPortsV0{},
		"director-service-test",
	)

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), orquestacionnucleoapp.SchedulerCandidateRequestV0{
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID:        "run-ref-service-quality-gate-replan-001",
			CurrentPhase: orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Tasks:        []string{"task-ref-service-quality-gate-replan-001"},
			QualityGates: []string{
				"quality-gate-ref-service-quality-gate-replan-001#decision:blocked#subject:task-ref-service-quality-gate-replan-001",
			},
			ReplanDecisions: []string{
				"replan-ref-service-quality-gate-replan-001#source:quality-gate-ref-service-quality-gate-replan-001#task:task-ref-service-quality-gate-replan-001#action:retry_task#followups:capacity-ref-service-quality-gate-replan-001+agent-ref-service-quality-gate-replan-001",
			},
		},
		OccurredAt:    "2026-05-21T17:05:00Z",
		CorrelationID: "corr-service-quality-gate-replan-001",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if len(candidates.ReplanFollowupCandidates) != 1 {
		t.Fatalf("quality gate replan provider no compuesto: candidates=%+v", candidates)
	}
}

func TestComposeStartAppDirectorProviderV0PassesWorkflowTaskProfileResolver(t *testing.T) {
	runRef := "run-ref-service-profile-resolver-001"
	task := serviceWorkflowProfileResolverTaskV0(runRef)
	resolver := &recordingWorkflowTaskProfileResolverForTestV0{
		resolution: orquestacionnucleoapp.WorkflowTaskProfileResolutionV0{
			ProfileKind:                orquestacoreworkflow.WorkProfileRequiredTestsV0,
			Role:                       "pruebas-inyectadas",
			ReasonCode:                 "profile-resolver-test",
			CapacitySummary:            "Capacidad inyectada por composicion.",
			AgentSummary:               "Agente inyectado por composicion.",
			MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
			EvidenceRefs:               []string{"evidence-ref-profile-resolver-001"},
		},
	}
	provider := composeStartAppDirectorProviderV0(
		orquestacionnucleoapp.StaticCandidateProviderV0{},
		StartAppDirectorPortsV0{
			DirectorTaskStore:           orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
			WorkflowTaskProfileResolver: resolver,
		},
		"director-service-test",
	)

	candidates, err := provider.BuildSchedulerCandidatesV0(context.Background(), orquestacionnucleoapp.SchedulerCandidateRequestV0{
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID:        runRef,
			CurrentPhase: orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			Tasks:        []string{task.TaskID},
		},
		OccurredAt:    "2026-05-22T10:11:00Z",
		CorrelationID: "corr-service-profile-resolver-001",
	})
	if err != nil {
		t.Fatalf("BuildSchedulerCandidatesV0: %v", err)
	}
	if !resolver.called || resolver.lastTaskRef != task.TaskID {
		t.Fatalf("profile resolver no invocado: called=%v task=%s", resolver.called, resolver.lastTaskRef)
	}
	if len(candidates.WorkCandidates) != 1 {
		t.Fatalf("work candidates=%+v", candidates.WorkCandidates)
	}
	candidate := candidates.WorkCandidates[0]
	if candidate.AgentCandidate == nil || candidate.AgentCandidate.Payload.Role != "pruebas-inyectadas" {
		t.Fatalf("agent profile no aplicado: %+v", candidate.AgentCandidate)
	}
	if candidate.CapacityCandidate == nil ||
		candidate.CapacityCandidate.Payload.ReasonCode != "profile-resolver-test" ||
		candidate.CapacityCandidate.Payload.MinimumRecommendedCapacity != orquestacoreworkflow.OrchestrationCapacityHighV0 {
		t.Fatalf("capacity profile no aplicado: %+v", candidate.CapacityCandidate)
	}
}

func TestContinueAppDirectorV0ProcessesReviewGateSource(t *testing.T) {
	runRef := "run-ref-service-review-gate-continue-001"
	run := orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "project-ref-service-review-gate-continue-001",
		AppSpecRef:    "appspec-ref-service-review-gate-continue-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
		Phases: []orquestacoreworkflow.OrchestrationPhaseV0{
			{
				ID:                  orquestacoreworkflow.OrchestrationPhaseRevisionV0,
				Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
				RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
			},
		},
		Tasks:         []string{"task-ref-service-review-gate-continue-001"},
		Agents:        []string{"agent-ref-service-review-gate-001"},
		StartedAgents: []string{"agent-ref-service-review-gate-001"},
		Deliveries:    []string{"delivery-ref-service-review-gate-001"},
	}
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	result, err := ContinueAppDirectorV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-10T10:06:00Z",
		CorrelationID:        "corr-service-review-gate-continue-001",
		MaxBursts:            4,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 2,
		MaxCommands:          20,
		MaxOutboxPerCycle:    8,
	}, StartAppDirectorPortsV0{
		RunStore:         store,
		EventSink:        sink,
		OutboxLedger:     ledger,
		ReviewGateSource: &recordingReviewGateSourceV0{},
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			serviceCapacityDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		var burstErr orquestadirectorsupervisedburst.DirectorSupervisedBurstErrorV0
		if errors.As(err, &burstErr) {
			t.Fatalf("ContinueAppDirectorV0: %v issues=%+v", err, burstErr.Issues)
		}
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if len(result.Run.Reviews) != 1 ||
		len(result.Run.ReviewResults) != 1 ||
		len(result.Run.ReworkRequests) != 1 {
		t.Fatalf("review gate no aplicado: %+v", result.Run)
	}
}

type recordingReviewReworkReplanSourceV0 struct {
	called bool
}

func (source *recordingReviewReworkReplanSourceV0) BuildReviewReworkReplanPlansV0(
	context.Context,
	orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
) ([]orquestacionnucleoapp.ReviewReworkReplanPlanV0, error) {
	source.called = true
	return nil, nil
}

type splitReviewReworkReplanSourceV0 struct{}
