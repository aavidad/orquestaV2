package main

import (
	"context"
	"path/filepath"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

func TestIdleSelfImprovementBlockersV0NoInfiereAuthBloqueadaDeAssessmentCritico(t *testing.T) {
	stateDir := t.TempDir()
	saveIdleSelfImprovementBlockerRunForTestV0(t, stateDir, orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:     orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:             "run-ref-provider-auth-blocked-001",
		Status:            orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		LostAgents:        []string{"agent-ref-provider-auth-blocked-001"},
		DirectorQuestions: []string{"question-ref-provider-auth-blocked-001"},
		AgentAssessments: []string{orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-provider-auth-blocked-001",
			AgentRequestID: "agent-ref-provider-auth-blocked-001",
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityCriticalV0,
		})},
	})
	supervisor := serverStackSupervisorV0{stateDir: stateDir}

	got, err := supervisor.IdleSelfImprovementBlockersV0(
		context.Background(),
		orquestaserver.IdleSelfImprovementBlockerRequestV0{},
	)
	if err != nil {
		t.Fatalf("IdleSelfImprovementBlockersV0: %v", err)
	}
	if got.Blocked || len(got.RunRefs) != 0 || got.Reason != "" ||
		containsStringServerStackForTestV0(got.EvidenceRefs, "evidence-ref-auth-config-blocker") {
		t.Fatalf("assessment critico generico no debe bloquear automejora=%+v", got)
	}
}

func TestIdleSelfImprovementBlockersV0IgnoraRunCerradoOCriticoNoAuth(t *testing.T) {
	stateDir := t.TempDir()
	saveIdleSelfImprovementBlockerRunForTestV0(t, stateDir, orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:     orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:             "run-ref-provider-auth-closed-001",
		Status:            orquestacoreworkflow.OrchestrationRunStatusClosedV0,
		LostAgents:        []string{"agent-ref-provider-auth-closed-001"},
		DirectorQuestions: []string{"question-ref-provider-auth-closed-001"},
		AgentAssessments: []string{orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-provider-auth-closed-001",
			AgentRequestID: "agent-ref-provider-auth-closed-001",
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityCriticalV0,
		})},
	})
	saveIdleSelfImprovementBlockerRunForTestV0(t, stateDir, orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:     orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:             "run-ref-provider-auth-high-001",
		Status:            orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		LostAgents:        []string{"agent-ref-provider-auth-high-001"},
		DirectorQuestions: []string{"question-ref-provider-auth-high-001"},
		AgentAssessments: []string{orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-provider-auth-high-001",
			AgentRequestID: "agent-ref-provider-auth-high-001",
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityHighV0,
		})},
	})
	supervisor := serverStackSupervisorV0{stateDir: stateDir}

	got, err := supervisor.IdleSelfImprovementBlockersV0(
		context.Background(),
		orquestaserver.IdleSelfImprovementBlockerRequestV0{},
	)
	if err != nil {
		t.Fatalf("IdleSelfImprovementBlockersV0: %v", err)
	}
	if got.Blocked || len(got.RunRefs) != 0 {
		t.Fatalf("blocker inesperado=%+v", got)
	}
}

func TestIdleSelfImprovementBlockersV0BloqueaProviderUnauthorizedActivoEnColaV0(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()
	runRef := "run-ref-provider-unauthorized-001"
	saveIdleSelfImprovementBlockerRunForTestV0(t, stateDir, providerIssueRunForTestV0(runRef, "codex_app_server_provider_unauthorized"))
	store := orquestarunmemory.NewRunMemoryStoreV0()
	if _, err := store.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      "queue-main",
		AppRef:        "project-ref-orquesta",
		Status:        orquestarunqueue.RunStatusReadyV0,
		PriorityScore: 10,
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 ready: %v", err)
	}
	supervisor := serverStackSupervisorV0{
		stateDir: stateDir,
		stack: &orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunControl: store,
				RunQueue:   store,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
		},
	}

	got, err := supervisor.IdleSelfImprovementBlockersV0(ctx, orquestaserver.IdleSelfImprovementBlockerRequestV0{})
	if err != nil {
		t.Fatalf("IdleSelfImprovementBlockersV0: %v", err)
	}
	if !got.Blocked ||
		got.Reason != idleSelfImprovementProviderAuthBlockedReasonV0 ||
		!containsStringServerStackForTestV0(got.RunRefs, runRef) ||
		got.RecoveryAction != idleSelfImprovementProviderAuthRecoveryActionV0 {
		t.Fatalf("provider unauthorized no bloquea automejora=%+v", got)
	}
}

func TestIdleSelfImprovementBlockersV0NotificaProviderPausadoUnaVezV0(t *testing.T) {
	stateDir := t.TempDir()
	runRef := "run-ref-provider-notification-001"
	saveIdleSelfImprovementBlockerRunForTestV0(t, stateDir, providerIssueRunForTestV0(runRef, "codex_app_server_provider_unauthorized"))
	sender := &fakeOperatorNotificationSenderV0{}
	supervisor := serverStackSupervisorV0{
		stateDir:         stateDir,
		operatorNotifier: newOperatorTaskTerminalNotifierV0("telegram:39995054", sender),
	}

	got, err := supervisor.IdleSelfImprovementBlockersV0(context.Background(), orquestaserver.IdleSelfImprovementBlockerRequestV0{})
	if err != nil {
		t.Fatalf("IdleSelfImprovementBlockersV0: %v", err)
	}
	if !got.Blocked ||
		sender.calls != 1 ||
		!containsStringServerStackForTestV0(got.EvidenceRefs, "evidence-ref-operator-terminal-notification") ||
		!containsStringServerStackForTestV0(got.EvidenceRefs, "evidence-ref-server-notification") {
		t.Fatalf("provider blocker no notifico correctamente got=%+v calls=%d", got, sender.calls)
	}
	_, err = supervisor.IdleSelfImprovementBlockersV0(context.Background(), orquestaserver.IdleSelfImprovementBlockerRequestV0{})
	if err != nil {
		t.Fatalf("IdleSelfImprovementBlockersV0 retry: %v", err)
	}
	if sender.calls != 1 {
		t.Fatalf("notificacion provider duplicada calls=%d", sender.calls)
	}
}

func TestIdleSelfImprovementBlockersV0BloqueaUsageLimitActivoV0(t *testing.T) {
	stateDir := t.TempDir()
	runRef := "run-ref-provider-limited-001"
	run := providerIssueRunForTestV0(runRef, "codex_app_server_goal_provider_limited")
	run.DirectorQuestions = append(run.DirectorQuestions, "provider_usage_limit_retry_after")
	saveIdleSelfImprovementBlockerRunForTestV0(t, stateDir, run)
	supervisor := serverStackSupervisorV0{stateDir: stateDir}

	got, err := supervisor.IdleSelfImprovementBlockersV0(context.Background(), orquestaserver.IdleSelfImprovementBlockerRequestV0{})
	if err != nil {
		t.Fatalf("IdleSelfImprovementBlockersV0: %v", err)
	}
	if !got.Blocked || !containsStringServerStackForTestV0(got.RunRefs, runRef) {
		t.Fatalf("usage limit no bloquea automejora=%+v", got)
	}
}

func TestIdleSelfImprovementBlockersV0IgnoraRunPausadoPorControlOCola(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()
	runRef := "run-ref-provider-auth-paused-001"
	saveIdleSelfImprovementBlockerRunForTestV0(t, stateDir, providerIssueRunForTestV0(runRef, "codex_app_server_provider_unauthorized"))
	store := orquestarunmemory.NewRunMemoryStoreV0()
	if _, err := store.PauseRunV0(ctx, orquestaruncontrol.PauseRunCommandV0{
		RunRef:      runRef,
		RequestedBy: "test",
		Reason:      "duplicado pausado",
	}); err != nil {
		t.Fatalf("PauseRunV0: %v", err)
	}
	if _, err := store.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      "queue-main",
		AppRef:        "project-ref-orquesta",
		Status:        orquestarunqueue.RunStatusPausedV0,
		PriorityScore: 0,
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}
	supervisor := serverStackSupervisorV0{
		stateDir: stateDir,
		stack: &orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunControl: store,
				RunQueue:   store,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
		},
	}

	got, err := supervisor.IdleSelfImprovementBlockersV0(ctx, orquestaserver.IdleSelfImprovementBlockerRequestV0{})
	if err != nil {
		t.Fatalf("IdleSelfImprovementBlockersV0: %v", err)
	}
	if got.Blocked || len(got.RunRefs) != 0 {
		t.Fatalf("blocker pausado no debe bloquear=%+v", got)
	}
}

func TestIdleSelfImprovementBlockersV0IgnoraRunHistoricoFueraDeColaYControlV0(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()
	runRef := "run-ref-provider-auth-historico-001"
	saveIdleSelfImprovementBlockerRunForTestV0(t, stateDir, providerAuthBlockedRunForTestV0(runRef))
	store := orquestarunmemory.NewRunMemoryStoreV0()
	supervisor := serverStackSupervisorV0{
		stateDir: stateDir,
		stack: &orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunControl: store,
				RunQueue:   store,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
		},
	}

	got, err := supervisor.IdleSelfImprovementBlockersV0(ctx, orquestaserver.IdleSelfImprovementBlockerRequestV0{})
	if err != nil {
		t.Fatalf("IdleSelfImprovementBlockersV0: %v", err)
	}
	if got.Blocked || len(got.RunRefs) != 0 {
		t.Fatalf("run historico fuera de cola/control no debe bloquear automejora=%+v", got)
	}
}

func TestIdleSelfImprovementBlockersV0NoBloqueaRunAuthInferidoActivoEnColaV0(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()
	runRef := "run-ref-provider-auth-ready-001"
	saveIdleSelfImprovementBlockerRunForTestV0(t, stateDir, providerAuthBlockedRunForTestV0(runRef))
	store := orquestarunmemory.NewRunMemoryStoreV0()
	if _, err := store.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      "queue-main",
		AppRef:        "project-ref-orquesta",
		Status:        orquestarunqueue.RunStatusReadyV0,
		PriorityScore: 10,
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 ready: %v", err)
	}
	supervisor := serverStackSupervisorV0{
		stateDir: stateDir,
		stack: &orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunControl: store,
				RunQueue:   store,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
		},
	}

	got, err := supervisor.IdleSelfImprovementBlockersV0(ctx, orquestaserver.IdleSelfImprovementBlockerRequestV0{})
	if err != nil {
		t.Fatalf("IdleSelfImprovementBlockersV0: %v", err)
	}
	if got.Blocked || len(got.RunRefs) != 0 {
		t.Fatalf("run auth inferido en cola no debe bloquear=%+v", got)
	}
}

func TestIdleSelfImprovementBlockersV0IgnoraRunAuthRecuperadoPorControlV0(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()
	runRef := "run-ref-provider-auth-recovered-001"
	saveIdleSelfImprovementBlockerRunForTestV0(t, stateDir, providerAuthBlockedRunForTestV0(runRef))
	store := orquestarunmemory.NewRunMemoryStoreV0()
	if _, err := store.PutRunControlStateV0(ctx, orquestaruncontrol.RunControlStateV0{
		RunRef:       runRef,
		Status:       orquestaruncontrol.RunControlStatusRunningV0,
		EvidenceRefs: []string{idleSelfImprovementProviderAuthRecoveredEvidenceV0},
		Meta: orquestaruncontrol.RunControlMetaV0{
			Reason: idleSelfImprovementProviderAuthRecoveredReasonV0,
		},
	}); err != nil {
		t.Fatalf("PutRunControlStateV0 recovered: %v", err)
	}
	if _, err := store.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      "queue-main",
		AppRef:        "project-ref-orquesta",
		Status:        orquestarunqueue.RunStatusReadyV0,
		PriorityScore: 10,
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 ready: %v", err)
	}
	supervisor := serverStackSupervisorV0{
		stateDir: stateDir,
		stack: &orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunControl: store,
				RunQueue:   store,
			},
			RunQueue: orquestaappcodexstack.RunQueueConfigV0{QueueRef: "queue-main"},
		},
	}

	got, err := supervisor.IdleSelfImprovementBlockersV0(ctx, orquestaserver.IdleSelfImprovementBlockerRequestV0{})
	if err != nil {
		t.Fatalf("IdleSelfImprovementBlockersV0: %v", err)
	}
	if got.Blocked || len(got.RunRefs) != 0 {
		t.Fatalf("run auth recuperado no debe bloquear=%+v", got)
	}
}

func TestIdleSelfImprovementBlockersV0NoBloqueaScopePorAuthInferidaV0(t *testing.T) {
	stateDir := t.TempDir()
	saveIdleSelfImprovementBlockerRunForTestV0(t, stateDir, providerAuthBlockedRunForTestV0("run-ref-provider-auth-old-001"))
	saveIdleSelfImprovementBlockerRunForTestV0(t, stateDir, providerAuthBlockedRunForTestV0("run-ref-provider-auth-current-001"))
	supervisor := serverStackSupervisorV0{stateDir: stateDir}

	got, err := supervisor.IdleSelfImprovementBlockersV0(
		context.Background(),
		orquestaserver.IdleSelfImprovementBlockerRequestV0{
			KnownRunRefs: []string{"run-ref-provider-auth-current-001"},
		},
	)
	if err != nil {
		t.Fatalf("IdleSelfImprovementBlockersV0: %v", err)
	}
	if got.Blocked || len(got.RunRefs) != 0 {
		t.Fatalf("blocker=%+v", got)
	}
}

func providerAuthBlockedRunForTestV0(runRef string) orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion:     orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:             runRef,
		Status:            orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		LostAgents:        []string{"agent-ref-" + runRef},
		DirectorQuestions: []string{"question-ref-" + runRef},
		AgentAssessments: []string{orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
			AssessmentRef:  "assessment-ref-" + runRef,
			AgentRequestID: "agent-ref-" + runRef,
			Verdict:        orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
			Action:         orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
			Severity:       orquestacoreworkflow.AgentAssessmentSeverityCriticalV0,
		})},
	}
}

func providerIssueRunForTestV0(runRef string, issueCode string) orquestacoreworkflow.OrchestrationRunV0 {
	run := providerAuthBlockedRunForTestV0(runRef)
	run.AgentAssessments = []string{orquestacoreworkflow.AgentAssessmentProjectionRefV0(orquestacoreworkflow.AgentWorkAssessedPayloadV0{
		AssessmentRef:  "assessment-ref-" + issueCode + "-001",
		AgentRequestID: "agent-ref-" + runRef,
		Verdict:        orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
		Action:         orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
		Severity:       orquestacoreworkflow.AgentAssessmentSeverityCriticalV0,
	})}
	run.DirectorQuestions = append(run.DirectorQuestions, issueCode)
	return run
}

func saveIdleSelfImprovementBlockerRunForTestV0(
	t *testing.T,
	stateDir string,
	run orquestacoreworkflow.OrchestrationRunV0,
) {
	t.Helper()
	store, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{
		RootDir: filepath.Join(stateDir, "orchestration-state"),
	})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	if err := store.SaveRunV0(context.Background(), run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
}

func containsStringServerStackForTestV0(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
