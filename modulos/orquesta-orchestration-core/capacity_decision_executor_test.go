package orquestacionnucleoapp

import (
	"context"
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func TestCapacityDecisionExecutorV0AdvancesFromCapacityToAgentOutbox(t *testing.T) {
	runRef := "run-nucleo-capacity-decision-001"
	store := NewInMemoryRunStoreV0(mustActiveProgrammingRunV0(t, runRef))
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: StaticCandidateProviderV0{Candidates: SchedulerCandidateSetV0{
			WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
				workCandidateWithAgentV0(runRef),
			},
		}},
		OutboxLedger: ledger,
		MaxCommands:  4,
	}

	first := runTestBurstV0(t, service, runRef, "corr-capacity-first-001")
	if first.Burst.FinalAction != orquestadirectorsupervisor.DirectorSupervisorActionWaitOutboxV0 {
		t.Fatalf("first action=%s", first.Burst.FinalAction)
	}
	if !containsNucleoRefV0(first.Run.CapacityRequests, "capacity-ref-001") {
		t.Fatalf("capacity requests=%v", first.Run.CapacityRequests)
	}

	dispatch, err := RunOutboxDispatchOnceV0(context.Background(), OutboxDispatchOnceRequestV0{
		RunRef:     runRef,
		TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: CapacityDecisionExecutorV0{
			RunStore:      store,
			EventSink:     sink,
			OccurredAt:    "2026-05-08T10:10:00Z",
			CorrelationID: "corr-capacity-decision-001",
			EvidenceRefs:  []string{"evidence-ref-capacity-executor-001"},
		},
		Acker: ledger,
	})
	if err != nil {
		t.Fatalf("dispatch capacity decision: %v", err)
	}
	if dispatch.Status != string(orquestaoutboxdispatch.RunOutboxDispatchOnceDispatchedV0) {
		t.Fatalf("dispatch=%+v", dispatch)
	}

	decided, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("load decided run: %v", err)
	}
	if len(decided.CapacityDecisions) != 1 {
		t.Fatalf("capacity decisions=%v", decided.CapacityDecisions)
	}

	second := runTestBurstV0(t, service, runRef, "corr-agent-request-001")
	if second.Burst.FinalAction != orquestadirectorsupervisor.DirectorSupervisorActionWaitOutboxV0 {
		t.Fatalf("second action=%s", second.Burst.FinalAction)
	}
	if !containsNucleoRefV0(second.Run.Agents, "agent-ref-001") {
		t.Fatalf("agents=%v", second.Run.Agents)
	}
	assertOneAgentLaunchOutboxV0(t, ledger, runRef)
}

func TestCapacityDecisionExecutorV0DelegaEnPolicyPort(t *testing.T) {
	runRef := "run-nucleo-capacity-policy-001"
	run := mustActiveProgrammingRunV0(t, runRef)
	run.CapacityRequests = []string{"capacity-ref-policy-001"}
	store := NewInMemoryRunStoreV0(run)
	sink := NewInMemoryEventSinkV0()
	intent := capacityDecisionIntentForPolicyTestV0(t, runRef)
	_, err := CapacityDecisionExecutorV0{
		RunStore:   store,
		EventSink:  sink,
		Policy:     fakeCapacityPolicyPortV0{},
		OccurredAt: "2026-05-24T10:10:00Z",
	}.ExecuteOutboxDispatchV0(intent)
	if err != nil {
		t.Fatalf("ExecuteOutboxDispatchV0: %v", err)
	}
	events, err := sink.LoadRunEventsV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadRunEventsV0: %v", err)
	}
	var payload orquestacoreworkflow.CapacityDecidedPayloadV0
	for _, event := range events {
		if event.EventType == orquestacoreworkflow.OrchestrationEventCapacityDecidedV0 {
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				t.Fatalf("payload: %v", err)
			}
		}
	}
	if payload.Tier != orquestacoreworkflow.OrchestrationCapacityHighV0 ||
		payload.ReasoningEffort != orquestacoreworkflow.OrchestrationCapacityMediumV0 {
		t.Fatalf("payload=%+v", payload)
	}
	for _, want := range []string{
		"capacity_policy_ref:capacity-policy-ref-policy-test",
		"capacity_pool_ref:capacity-pool-ref-policy-test",
		"capacity_model_ref:capacity-model-ref-policy-test",
		"capacity_quota_ref:capacity-quota-ref-policy-test",
		"capacity_motivo:policy-port-test",
	} {
		if !containsNucleoRefV0(payload.EvidenceRefs, want) {
			t.Fatalf("evidence refs=%v, falta %s", payload.EvidenceRefs, want)
		}
	}
}

func runTestBurstV0(
	t *testing.T,
	service ServiceV0,
	runRef string,
	correlationID string,
) SupervisedBurstResultV0 {
	t.Helper()
	result, err := service.RunSupervisedBurstV0(context.Background(), SupervisedBurstRequestV0{
		RunRef:        runRef,
		OccurredAt:    "2026-05-08T10:00:00Z",
		MaxSteps:      3,
		CorrelationID: correlationID,
		EvidenceRefs:  []string{"evidence-ref-" + correlationID},
	})
	if err != nil {
		t.Fatalf("run burst: %v", err)
	}
	return result
}

func assertOneAgentLaunchOutboxV0(
	t *testing.T,
	ledger *InMemoryOutboxLedgerV0,
	runRef string,
) {
	t.Helper()
	pending, issues := ledger.ListPending(context.Background(),
		orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
			RunRef:     runRef,
			TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		},
	)
	if len(issues) > 0 {
		t.Fatalf("outbox issues=%+v", issues)
	}
	if len(pending) != 1 {
		t.Fatalf("pending agent outbox=%+v", pending)
	}
	if pending[0].MessageType != orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0 {
		t.Fatalf("message type=%s", pending[0].MessageType)
	}
}

func workCandidateWithAgentV0(runRef string) orquestadirectorscheduler.SchedulableWorkCandidateV0 {
	candidate := workCandidateV0(runRef)
	candidate.AgentCandidate = &orquestadirectorscheduler.SchedulerAgentCommandCandidateV0{
		ClaimRef:    "claim-ref-001",
		CommandMeta: commandMetaV0(runRef, "cmd-agent-001", "idem-agent-001"),
		Payload: orquestacoreworkflow.RequestAgentCommandPayloadV0{
			AgentRequestID:     "agent-ref-001",
			PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			TaskRef:            "task-ref-001",
			CapacityRequestRef: "capacity-ref-001",
			Role:               "implementacion",
			Summary:            "Agente para siguiente tarea.",
			EvidenceRefs:       []string{"evidence-ref-agent-001"},
		},
	}
	candidate.GateEvidenceRefs = []string{"evidence-ref-gate-001"}
	return candidate
}

func containsNucleoRefV0(values []string, ref string) bool {
	for _, value := range values {
		if value == ref {
			return true
		}
	}
	return false
}

type fakeCapacityPolicyPortV0 struct{}

func (fakeCapacityPolicyPortV0) DecideCapacityV0(
	_ context.Context,
	request CapacityDecisionPolicyRequestV0,
) (CapacityDecisionPolicyDecisionV0, error) {
	return CapacityDecisionPolicyDecisionV0{
		DecisionRef:     request.DecisionRef,
		Tier:            orquestacoreworkflow.OrchestrationCapacityHighV0,
		ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityMediumV0,
		Summary:         "Decision por policy fake.",
		PolicyRef:       "capacity-policy-ref-policy-test",
		PoolRef:         "capacity-pool-ref-policy-test",
		ModelRef:        "capacity-model-ref-policy-test",
		QuotaRef:        "capacity-quota-ref-policy-test",
		Motivos:         []string{"policy-port-test"},
		EvidenceRefs:    []string{"evidence-ref-policy-port-test"},
	}, nil
}

func capacityDecisionIntentForPolicyTestV0(
	t *testing.T,
	runRef string,
) orquestaoutboxdispatch.DispatchIntentV0 {
	t.Helper()
	payload, err := json.Marshal(orquestacoreworkflow.CapacityDecisionRequestV0{
		CapacityRequestID:          "capacity-ref-policy-001",
		RunID:                      runRef,
		PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		ReasonCode:                 "policy-port",
		Summary:                    "Validar puerto de politica.",
		MinimumRecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityLowV0,
		EvidenceRefs:               []string{"evidence-ref-policy-request"},
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return orquestaoutboxdispatch.DispatchIntentV0{
		MessageID:   "outbox-capacity-policy-001",
		MessageType: orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0,
		RunID:       runRef,
		TargetPort:  orquestacoreworkflow.OutboxTargetCapacityV0,
		Payload:     payload,
	}
}

func containsNucleoAssessmentRefV0(values []string, ref string) bool {
	for _, value := range values {
		if orquestacoreworkflow.AgentAssessmentProjectionIDV0(value) == ref {
			return true
		}
	}
	return false
}
