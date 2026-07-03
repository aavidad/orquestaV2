package main

import (
	"context"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunfile "orquesta/modulos/orquesta-run-file"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestServerSupervisorWakeupDecoratorsV0DisparanSoloTrasMutacionesEjecutables(t *testing.T) {
	ctx := context.Background()
	var causes []string
	var residentCauses []string
	relay := &serverSupervisorWakeupRelayV0{
		request: func(cause string) bool {
			causes = append(causes, cause)
			return true
		},
		requestResidentDirector: func(cause string) bool {
			residentCauses = append(residentCauses, cause)
			return true
		},
	}

	runStore := serverWakeupRunStoreV0{
		inner:  orquestacionnucleoapp.NewInMemoryRunStoreV0(),
		wakeup: relay,
	}
	if err := runStore.SaveRunV0(ctx, orquestacoreworkflow.OrchestrationRunV0{
		RunID: "run-ref-wakeup-store-001",
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}

	fileStore, err := orquestarunfile.NewRunFileStoreV0(t.TempDir())
	if err != nil {
		t.Fatalf("NewRunFileStoreV0: %v", err)
	}
	queue := serverWakeupRunQueueV0{inner: fileStore, wakeup: relay}
	if _, err := queue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef: "run-ref-wakeup-ready-001", QueueRef: "queue-main", Status: orquestarunqueue.RunStatusReadyV0,
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 ready: %v", err)
	}
	if _, err := queue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef: "run-ref-wakeup-closed-001", QueueRef: "queue-main", Status: orquestarunqueue.RunStatusClosedV0,
	}); err != nil {
		t.Fatalf("SetRunPriorityV0 closed: %v", err)
	}

	control := serverWakeupRunControlV0{inner: fileStore, wakeup: relay}
	if _, err := control.PauseRunV0(ctx, orquestaruncontrol.PauseRunCommandV0{
		RunRef: "run-ref-wakeup-ready-001",
	}); err != nil {
		t.Fatalf("PauseRunV0: %v", err)
	}
	if _, err := control.ResumeRunV0(ctx, orquestaruncontrol.ResumeRunCommandV0{
		RunRef: "run-ref-wakeup-ready-001",
	}); err != nil {
		t.Fatalf("ResumeRunV0: %v", err)
	}
	if _, err := control.CompleteRunControlV0(ctx, orquestaruncontrol.CompleteRunControlCommandV0{
		RunRef:       "run-ref-wakeup-ready-001",
		TargetStatus: orquestaruncontrol.RunControlStatusStoppedV0,
	}); err != nil {
		t.Fatalf("CompleteRunControlV0: %v", err)
	}

	want := []string{"run_store_saved", "run_queue_ready", "run_control_resume", "run_control_complete"}
	if !stringSlicesEqualV0(causes, want) {
		t.Fatalf("causes=%v want=%v", causes, want)
	}
	if !stringSlicesEqualV0(residentCauses, []string{"run_store_saved"}) {
		t.Fatalf("residentCauses=%v", residentCauses)
	}
}

func TestServerWakeupAppChangeStoreV0DisparaAlGuardarSolicitud(t *testing.T) {
	ctx := context.Background()
	var causes []string
	var residentCauses []string
	relay := &serverSupervisorWakeupRelayV0{
		request: func(cause string) bool {
			causes = append(causes, cause)
			return true
		},
		requestResidentDirector: func(cause string) bool {
			residentCauses = append(residentCauses, cause)
			return true
		},
	}
	store := serverWakeupAppChangeStoreV0{
		inner:  orquestaappchange.NewInMemoryAppChangeStoreV0(),
		wakeup: relay,
	}

	if err := store.SaveAppChangeRequestV0(ctx, orquestaappchange.AppChangeRecordV0{
		Request: orquestaappchange.AppChangeRequestV0{
			RequestID:     "request-ref-app-change-wakeup-001",
			CorrelationID: "corr-app-change-wakeup-001",
			RunRef:        "run-ref-app-change-wakeup-001",
			AppRef:        "app-ref-app-change-wakeup-001",
			ChangeRef:     "change-ref-app-change-wakeup-001",
		},
	}); err != nil {
		t.Fatalf("SaveAppChangeRequestV0: %v", err)
	}
	records, err := store.ListAppChangeRecordsV0(ctx, orquestaappchange.AppChangeRecordFilterV0{
		RunRef: "run-ref-app-change-wakeup-001",
	})
	if err != nil || len(records) != 1 {
		t.Fatalf("ListAppChangeRecordsV0 records=%+v err=%v", records, err)
	}
	if !stringSlicesEqualV0(causes, []string{"app_change_saved"}) {
		t.Fatalf("causes=%v", causes)
	}
	if !stringSlicesEqualV0(residentCauses, []string{"app_change_saved"}) {
		t.Fatalf("residentCauses=%v", residentCauses)
	}
}

func TestServerWakeupDirectorCycleOutboxLedgerV0DisparaAlGuardarPendientes(t *testing.T) {
	ctx := context.Background()
	var causes []string
	var residentCauses []string
	relay := &serverSupervisorWakeupRelayV0{
		request: func(cause string) bool {
			causes = append(causes, cause)
			return true
		},
		requestResidentDirector: func(cause string) bool {
			residentCauses = append(residentCauses, cause)
			return true
		},
	}
	ledger := serverWakeupDirectorCycleOutboxLedgerV0{
		inner:  &fakeServerDirectorCycleOutboxLedgerV0{},
		wakeup: relay,
	}

	saved, issues := ledger.SavePending(ctx, []orquestacoreworkflow.OutboxMessageV0{{
		MessageID:      "message-ref-wakeup-outbox-001",
		RunID:          "run-ref-wakeup-outbox-001",
		TargetPort:     "target-port-test",
		IdempotencyKey: "idem-wakeup-outbox-001",
	}})
	if len(issues) > 0 {
		t.Fatalf("SavePending issues=%+v", issues)
	}
	if len(saved) != 1 {
		t.Fatalf("saved=%+v", saved)
	}
	if !stringSlicesEqualV0(causes, []string{"director_cycle_outbox_saved"}) {
		t.Fatalf("causes=%v", causes)
	}
	if !stringSlicesEqualV0(residentCauses, []string{"director_cycle_outbox_saved"}) {
		t.Fatalf("residentCauses=%v", residentCauses)
	}

	if _, issues := ledger.ListPending(ctx, orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
		RunRef: "run-ref-wakeup-outbox-001",
	}); len(issues) > 0 {
		t.Fatalf("ListPending issues=%+v", issues)
	}
	if !stringSlicesEqualV0(causes, []string{"director_cycle_outbox_saved"}) {
		t.Fatalf("ListPending no debe disparar wakeup: causes=%v", causes)
	}
	if !stringSlicesEqualV0(residentCauses, []string{"director_cycle_outbox_saved"}) {
		t.Fatalf("ListPending no debe disparar wakeup residente: residentCauses=%v", residentCauses)
	}
}

func TestServerWakeupCodexReceiptStoreV0DisparaAlRegistrarDescriptorYSidecar(t *testing.T) {
	ctx := context.Background()
	var causes []string
	var residentCauses []string
	relay := &serverSupervisorWakeupRelayV0{
		request: func(cause string) bool {
			causes = append(causes, cause)
			return true
		},
		requestResidentDirector: func(cause string) bool {
			residentCauses = append(residentCauses, cause)
			return true
		},
	}
	inner := &fakeServerCodexReceiptStoreV0{}
	store := serverWakeupCodexReceiptStoreV0{
		inner:  inner,
		wakeup: relay,
	}

	descriptor := orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{
		DescriptorRef: "codex-receipt-ref-wakeup-001",
		RunID:         "run-ref-receipt-wakeup-001",
		AgentRef:      "agent-ref-receipt-wakeup-001",
	}
	if err := store.RecordCodexReceiptDescriptorV0(ctx, descriptor); err != nil {
		t.Fatalf("RecordCodexReceiptDescriptorV0: %v", err)
	}
	if got, err := store.ListCodexReceiptDescriptorsV0(ctx, orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
		RunID: "run-ref-receipt-wakeup-001",
	}); err != nil || len(got) != 1 {
		t.Fatalf("ListCodexReceiptDescriptorsV0 got=%+v err=%v", got, err)
	}
	if err := store.RecordDirectorAgentDecisionFileConsumptionV0(
		ctx,
		orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0{
			ReceiptRef: "sidecar-receipt-ref-wakeup-001",
			RunID:      "run-ref-receipt-wakeup-001",
			AgentRef:   "agent-ref-receipt-wakeup-001",
			Status:     "pending",
		},
	); err != nil {
		t.Fatalf("RecordDirectorAgentDecisionFileConsumptionV0: %v", err)
	}

	want := []string{"codex_receipt_descriptor_recorded", "director_decision_sidecar_consumed"}
	if !stringSlicesEqualV0(causes, want) {
		t.Fatalf("causes=%v want=%v", causes, want)
	}
	if !stringSlicesEqualV0(residentCauses, want) {
		t.Fatalf("residentCauses=%v want=%v", residentCauses, want)
	}
	if len(inner.sidecars) != 1 {
		t.Fatalf("sidecars=%+v", inner.sidecars)
	}
}

func TestServerWakeupDomainWorkArtifactSubmissionLedgerV0DisparaAlRegistrar(t *testing.T) {
	ctx := context.Background()
	var causes []string
	var residentCauses []string
	relay := &serverSupervisorWakeupRelayV0{
		request: func(cause string) bool {
			causes = append(causes, cause)
			return true
		},
		requestResidentDirector: func(cause string) bool {
			residentCauses = append(residentCauses, cause)
			return true
		},
	}
	ledger := serverWakeupDomainWorkArtifactSubmissionLedgerV0{
		inner:  orquestaappcodexstack.NewInMemoryDomainWorkArtifactSubmissionLedgerV0(),
		wakeup: relay,
	}
	record := orquestaappcodexstack.DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "domain-work-artifact-wakeup-001",
		Status:         orquestaappcodexstack.DomainWorkArtifactSubmissionStatusAcceptedV0,
		DomainRef:      "opes",
		JobRef:         "job-ref-wakeup-domain-work-001",
		ArtifactRef:    "artifact-ref-wakeup-domain-work-001",
		ArtifactType:   orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0,
	}

	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, record); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	exists, err := ledger.HasDomainWorkArtifactSubmissionV0(ctx, record.IdempotencyKey)
	if err != nil || !exists {
		t.Fatalf("HasDomainWorkArtifactSubmissionV0 exists=%v err=%v", exists, err)
	}
	records, err := ledger.ListDomainWorkArtifactSubmissionsV0(
		ctx,
		orquestaappcodexstack.DomainWorkArtifactSubmissionRecordFilterV0{Status: record.Status},
	)
	if err != nil || len(records) != 1 {
		t.Fatalf("ListDomainWorkArtifactSubmissionsV0 records=%+v err=%v", records, err)
	}

	want := []string{"domain_work_artifact_recorded"}
	if !stringSlicesEqualV0(causes, want) {
		t.Fatalf("causes=%v want=%v", causes, want)
	}
	if !stringSlicesEqualV0(residentCauses, want) {
		t.Fatalf("residentCauses=%v want=%v", residentCauses, want)
	}
}

func TestServerWakeupGoalStateStoreV0DisparaSupervisorSoloConTerminalV0(t *testing.T) {
	ctx := context.Background()
	var causes []string
	var goalCauses []string
	relay := &serverSupervisorWakeupRelayV0{
		request: func(cause string) bool {
			causes = append(causes, cause)
			return true
		},
		requestGoalObservation: func(cause string) bool {
			goalCauses = append(goalCauses, cause)
			return true
		},
	}
	store := serverWakeupGoalStateStoreV0{
		inner:  newFakeServerGoalStateStoreV0(),
		wakeup: relay,
	}

	if err := store.SaveGoalWorkStateV0(ctx, orquestagoal.GoalWorkStateV0{
		RunRef:  "run-ref-goal-wakeup-running",
		GoalRef: "goal-ref-goal-wakeup-running",
		Status:  orquestagoal.GoalStatusRunningV0,
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 running: %v", err)
	}
	if err := store.SaveGoalWorkStateV0(ctx, orquestagoal.GoalWorkStateV0{
		RunRef:  "run-ref-goal-wakeup-complete",
		GoalRef: "goal-ref-goal-wakeup-complete",
		Status:  orquestagoal.GoalStatusCompleteV0,
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 complete: %v", err)
	}
	if err := store.SaveGoalWorkRunMarkerV0(ctx, orquestagoal.GoalWorkRunMarkerV0{
		RunRef:  "run-ref-goal-marker-running",
		GoalRef: "goal-ref-goal-marker-running",
		Status:  orquestagoal.GoalStatusRunningV0,
	}); err != nil {
		t.Fatalf("SaveGoalWorkRunMarkerV0 running: %v", err)
	}
	if err := store.SaveGoalWorkRunMarkerV0(ctx, orquestagoal.GoalWorkRunMarkerV0{
		RunRef:  "run-ref-goal-marker-blocked",
		GoalRef: "goal-ref-goal-marker-blocked",
		Status:  orquestagoal.GoalStatusBlockedV0,
	}); err != nil {
		t.Fatalf("SaveGoalWorkRunMarkerV0 blocked: %v", err)
	}

	if !stringSlicesEqualV0(causes, []string{"goal_state_terminal", "goal_run_marker_terminal"}) {
		t.Fatalf("causes=%v", causes)
	}
	wantGoalCauses := []string{
		"goal_state_saved",
		"goal_state_saved",
		"goal_run_marker_saved",
		"goal_run_marker_saved",
	}
	if !stringSlicesEqualV0(goalCauses, wantGoalCauses) {
		t.Fatalf("goalCauses=%v want=%v", goalCauses, wantGoalCauses)
	}
}

type fakeServerDirectorCycleOutboxLedgerV0 struct {
	pending []orquestacoreworkflow.OutboxMessageV0
}

func (ledger *fakeServerDirectorCycleOutboxLedgerV0) SavePending(
	_ context.Context,
	messages []orquestacoreworkflow.OutboxMessageV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	ledger.pending = append(ledger.pending, messages...)
	return append([]orquestacoreworkflow.OutboxMessageV0(nil), messages...), nil
}

func (ledger *fakeServerDirectorCycleOutboxLedgerV0) ListPending(
	_ context.Context,
	_ orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0,
) ([]orquestacoreworkflow.OutboxMessageV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	return append([]orquestacoreworkflow.OutboxMessageV0(nil), ledger.pending...), nil
}

func (ledger *fakeServerDirectorCycleOutboxLedgerV0) ListPendingOutboxV0(
	_ orquestaoutboxdispatch.PendingOutboxFilterV0,
) ([]orquestaoutboxdispatch.OutboxPendingEntryV0, []orquestaoutboxdispatch.DispatchIssueV0) {
	return nil, nil
}

func (ledger *fakeServerDirectorCycleOutboxLedgerV0) ClaimOutboxDispatchV0(
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) (orquestaoutboxdispatch.OutboxDispatchClaimResultV0, []orquestaoutboxdispatch.DispatchIssueV0) {
	return orquestaoutboxdispatch.OutboxDispatchClaimResultV0{
		Claimed: true, MessageID: claim.MessageID, TargetPort: claim.TargetPort,
	}, nil
}

func (ledger *fakeServerDirectorCycleOutboxLedgerV0) AckOutboxDispatchV0(
	_ orquestaoutboxdispatch.OutboxDispatchAckV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	return nil
}

type fakeServerCodexReceiptStoreV0 struct {
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0
	sidecars    []orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0
}

func (store *fakeServerCodexReceiptStoreV0) RecordCodexReceiptDescriptorV0(
	_ context.Context,
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) error {
	store.descriptors = append(store.descriptors, descriptor)
	return nil
}

func (store *fakeServerCodexReceiptStoreV0) ListCodexReceiptDescriptorsV0(
	_ context.Context,
	_ orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0,
) ([]orquestaruntimecodexdelivery.CodexReceiptDescriptorV0, error) {
	return append([]orquestaruntimecodexdelivery.CodexReceiptDescriptorV0(nil), store.descriptors...), nil
}

func (store *fakeServerCodexReceiptStoreV0) RecordDirectorAgentDecisionFileConsumptionV0(
	_ context.Context,
	receipt orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0,
) error {
	store.sidecars = append(store.sidecars, receipt)
	return nil
}

type fakeServerGoalStateStoreV0 struct {
	states  map[string]orquestagoal.GoalWorkStateV0
	markers map[string]orquestagoal.GoalWorkRunMarkerV0
}

func newFakeServerGoalStateStoreV0() *fakeServerGoalStateStoreV0 {
	return &fakeServerGoalStateStoreV0{
		states:  map[string]orquestagoal.GoalWorkStateV0{},
		markers: map[string]orquestagoal.GoalWorkRunMarkerV0{},
	}
}

func (store *fakeServerGoalStateStoreV0) SaveGoalWorkStateV0(
	_ context.Context,
	state orquestagoal.GoalWorkStateV0,
) error {
	store.states[state.RunRef] = state
	return nil
}

func (store *fakeServerGoalStateStoreV0) LoadGoalWorkStateV0(
	_ context.Context,
	runRef string,
) (orquestagoal.GoalWorkStateV0, error) {
	return store.states[runRef], nil
}

func (store *fakeServerGoalStateStoreV0) SaveGoalWorkRunMarkerV0(
	_ context.Context,
	marker orquestagoal.GoalWorkRunMarkerV0,
) error {
	store.markers[marker.RunRef] = marker
	return nil
}

func (store *fakeServerGoalStateStoreV0) LoadGoalWorkRunMarkerV0(
	_ context.Context,
	runRef string,
) (orquestagoal.GoalWorkRunMarkerV0, error) {
	return store.markers[runRef], nil
}

func (store *fakeServerGoalStateStoreV0) ListGoalWorkRunMarkersV0(
	_ context.Context,
	_ orquestagoal.GoalWorkRunMarkerListRequestV0,
) ([]orquestagoal.GoalWorkRunMarkerV0, error) {
	out := make([]orquestagoal.GoalWorkRunMarkerV0, 0, len(store.markers))
	for _, marker := range store.markers {
		out = append(out, marker)
	}
	return out, nil
}

func (store *fakeServerGoalStateStoreV0) ListGoalWorkStatesV0(
	_ context.Context,
	_ orquestagoal.GoalWorkStateListRequestV0,
) ([]orquestagoal.GoalWorkStateV0, error) {
	out := make([]orquestagoal.GoalWorkStateV0, 0, len(store.states))
	for _, state := range store.states {
		out = append(out, state)
	}
	return out, nil
}

func stringSlicesEqualV0(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
