package application

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type testAttestationSystem struct {
	orchestrator *Orchestrator
	repository   *memoryRepository
	control      *scriptedVersionControl
	attestor     *scriptedTestAttestor
	access       Access
	goalRef      goal.GoalRef
}

type generationBindingVersionControl struct {
	*scriptedVersionControl
	previewCalls   int
	integrateCalls int
}

func (control *generationBindingVersionControl) PreviewIntegration(
	ctx context.Context,
	request ports.IntegrationPreviewRequest,
) (ports.IntegrationPreview, error) {
	control.previewCalls++
	return control.scriptedVersionControl.PreviewIntegration(ctx, request)
}

func (control *generationBindingVersionControl) Integrate(
	ctx context.Context,
	request ports.IntegrationRequest,
) (ports.IntegrationResult, error) {
	control.integrateCalls++
	return control.scriptedVersionControl.Integrate(ctx, request)
}

func newTestAttestationSystem(t *testing.T, verdict ports.TestAttestationVerdict) *testAttestationSystem {
	t.Helper()
	clock := &mutableClock{now: time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("candidate output"),
	}}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	control := &scriptedVersionControl{}
	attestor := &scriptedTestAttestor{verdict: verdict}
	orchestrator.versionControl, orchestrator.testAttestor = control, attestor
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	submitted, err := orchestrator.Submit(context.Background(), access, SubmitRequest{
		RequestRef: "request:test-attestation", Statement: "produce and attest candidate",
		Confirm: true, Plan: workspaceWritePlan(),
	})
	appTestNoError(t, err)
	return &testAttestationSystem{
		orchestrator: orchestrator, repository: repository, control: control,
		attestor: attestor, access: access, goalRef: submitted.Record.Goal.Ref(),
	}
}

func (system *testAttestationSystem) process(t *testing.T, actions ...ActionKind) {
	t.Helper()
	for _, want := range actions {
		result, err := system.orchestrator.ProcessNext(context.Background(), "worker:test-attestation")
		if err != nil || result.Action != want {
			t.Fatalf("process=%+v want=%s err=%v", result, want, err)
		}
	}
}

func (system *testAttestationSystem) record(t *testing.T) GoalRecord {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), system.goalRef)
	appTestNoError(t, err)
	return record
}

func (system *testAttestationSystem) processCommit(t *testing.T) {
	t.Helper()
	system.process(t, ActionPrepareWorkspace, ActionLaunchAgent, ActionObserveAgent, ActionCommitChange)
}

func (system *testAttestationSystem) approveReviews(t *testing.T) {
	t.Helper()
	system.process(t, ActionLaunchAgent, ActionLaunchAgent, ActionObserveAgent, ActionObserveAgent)
}

func TestPassingAttestationLeavesChangePending(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	if record.Goal.State() != goal.GoalStateRunning || item.State() != goal.WorkItemStateRunning ||
		record.Executions[0].State != ExecutionAwaitingIntegration || len(record.ChangeSets) != 1 ||
		len(record.IntegrationReceipts) != 0 || len(record.Attestations) != 2 || len(record.Artifacts) != 3 {
		t.Fatalf("PASS advanced too far: %+v", record)
	}
	attestation := record.Attestations[1]
	if attestation.Kind != AttestationKindRequiredTests || attestation.Verdict != AttestationVerdictPassed ||
		attestation.ManifestArtifactRef.String() == "" || attestation.ReportArtifactRef.String() == "" {
		t.Fatalf("typed PASS=%+v", attestation)
	}
	system.repository.mu.Lock()
	pendingActions := len(system.repository.actions)
	system.repository.mu.Unlock()
	if pendingActions != 2 || len(record.Reviews) != 0 {
		t.Fatalf("PASS did not schedule the independent review round: actions=%d reviews=%d", pendingActions, len(record.Reviews))
	}
	system.attestor.mu.Lock()
	attestorCalls := len(system.attestor.runs)
	system.attestor.mu.Unlock()
	if attestorCalls != 1 {
		t.Fatalf("attestation capture calls=%d", attestorCalls)
	}
}

func TestIntegrateChangeRequiresExactPassAndRejectsFailedMissingInvalid(t *testing.T) {
	admit := func(t *testing.T, system *testAttestationSystem) (IntegrateChangeResult, error) {
		record := system.record(t)
		return system.orchestrator.IntegrateChange(context.Background(), system.access, IntegrateChangeRequest{
			RequestRef: "request:integrate:test-attestation", GoalRef: record.Goal.Ref(),
			ChangeRef: record.ChangeSets[0].Ref, ExpectedTargetOID: record.WorkspaceBindings[0].BaseOID,
		})
	}
	cases := []struct {
		name    string
		verdict ports.TestAttestationVerdict
		attest  bool
		tamper  bool
		pass    bool
	}{
		{"missing", ports.TestAttestationPassed, false, false, false},
		{"failed", ports.TestAttestationFailed, true, false, false},
		{"invalid", ports.TestAttestationPassed, true, true, false},
		{"exact", ports.TestAttestationPassed, true, false, true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			system := newTestAttestationSystem(t, test.verdict)
			system.processCommit(t)
			if test.attest {
				system.process(t, ActionAttestTest)
				if test.verdict == ports.TestAttestationPassed {
					system.approveReviews(t)
				}
			}
			if test.tamper {
				system.repository.mu.Lock()
				record := system.repository.records[system.goalRef]
				record.Attestations[len(record.Attestations)-1].SubjectDigest = testDigest("tampered")
				system.repository.records[system.goalRef] = record
				system.repository.mu.Unlock()
			}
			result, err := admit(t, system)
			if test.pass && (err != nil || !result.Created || result.Action.Kind != ActionIntegrateChange) {
				t.Fatalf("exact PASS result=%+v err=%v", result, err)
			}
			if !test.pass && !IsStateError(err, StateConflict) {
				t.Fatalf("invalid admission result=%+v err=%v", result, err)
			}
		})
	}
}

func TestPassingAttestationIsBoundToExactWorkItemGeneration(t *testing.T) {
	ctx := context.Background()
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	control := &generationBindingVersionControl{scriptedVersionControl: system.control}
	system.orchestrator.versionControl = control
	system.processCommit(t)
	system.process(t, ActionAttestTest)
	system.approveReviews(t)

	passed := system.record(t)
	item := passed.Goal.WorkItems()[0]
	change := passed.ChangeSets[0]
	execution := passed.Executions[0]
	attestation := passed.Attestations[len(passed.Attestations)-1]
	passGeneration := item.Revision()
	if attestation.WorkItemGeneration != passGeneration || len(system.attestor.requests) != 1 ||
		system.attestor.requests[0].Subject.WorkItemGeneration != passGeneration {
		t.Fatalf("PASS generation: item=%d attestation=%d requests=%+v",
			passGeneration, attestation.WorkItemGeneration, system.attestor.requests)
	}

	admitted, err := system.orchestrator.IntegrateChange(ctx, system.access, IntegrateChangeRequest{
		RequestRef: "request:integrate:generation-n", GoalRef: passed.Goal.Ref(),
		ChangeRef: change.Ref, ExpectedTargetOID: passed.WorkspaceBindings[0].BaseOID,
	})
	if err != nil || !admitted.Created || admitted.Action.Kind != ActionIntegrateChange {
		t.Fatalf("integration at generation N: result=%+v reviews=%+v executions=%+v err=%v", admitted,
			system.record(t).Reviews, system.record(t).Executions, err)
	}

	aggregate := passed.Goal
	for index, paused := range []bool{true, false} {
		currentItem, _ := aggregate.WorkItem(item.Ref())
		aggregate, err = aggregate.SetWorkItemPaused(
			aggregate.Revision(), currentItem.Revision(), currentItem.Ref(), paused,
			time.Date(2026, 7, 22, 8, 0, index+1, 0, time.UTC),
		)
		if err != nil {
			t.Fatalf("set work item paused=%v: %v", paused, err)
		}
	}
	system.repository.mu.Lock()
	current := system.repository.records[system.goalRef]
	current.Goal = aggregate
	system.repository.records[system.goalRef] = current
	system.repository.mu.Unlock()

	advanced := system.record(t)
	advancedItem, _ := advanced.Goal.WorkItem(item.Ref())
	if advancedItem.Revision() != passGeneration+2 || advancedItem.Paused() ||
		len(advanced.Attestations) != len(passed.Attestations) ||
		advanced.Attestations[len(advanced.Attestations)-1].WorkItemGeneration != passGeneration {
		t.Fatalf("causal advance lost history: pass=%d current=%d paused=%v attestations=%+v",
			passGeneration, advancedItem.Revision(), advancedItem.Paused(), advanced.Attestations)
	}
	if _, active := requiredTestsPassForChange(
		advanced, advancedItem, execution, change, system.orchestrator.testAttestationPolicy,
	); active {
		t.Fatal("historical PASS remained active after work item generation advanced")
	}

	if _, err = system.orchestrator.IntegrateChange(ctx, system.access, IntegrateChangeRequest{
		RequestRef: "request:integrate:generation-advanced", GoalRef: advanced.Goal.Ref(),
		ChangeRef: change.Ref, ExpectedTargetOID: advanced.WorkspaceBindings[0].BaseOID,
	}); !IsStateError(err, StateConflict) {
		t.Fatalf("advanced generation admitted integration: %v", err)
	}
	processed, processErr := system.orchestrator.ProcessNext(ctx, "worker:generation-binding")
	if processErr == nil || processErr.Error() != "application.required_tests_pass_missing" ||
		processed.Action != ActionIntegrateChange || control.previewCalls != 0 || control.integrateCalls != 0 {
		t.Fatalf("stale PASS gate: processed=%+v err=%v preview=%d integrate=%d",
			processed, processErr, control.previewCalls, control.integrateCalls)
	}
	if record := system.record(t); len(record.IntegrationReceipts) != 0 {
		t.Fatalf("stale PASS produced integration receipt: %+v", record.IntegrationReceipts)
	}
}

func TestFailedAttestationPersistsEvidenceAndLeavesChangePending(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationFailed)
	system.processCommit(t)
	system.process(t, ActionAttestTest)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	attestation := record.Attestations[len(record.Attestations)-1]
	if item.State() != goal.WorkItemStateInterrupted || record.Executions[0].State != ExecutionFailed ||
		record.Executions[0].FailureCode != "test_attestor.required_tests_failed" || len(record.ChangeSets) != 1 ||
		len(record.IntegrationReceipts) != 0 || attestation.Verdict != AttestationVerdictFailed ||
		attestation.Kind != AttestationKindRequiredTests || len(attestation.Tests) != 1 ||
		attestation.Tests[0].ExitCode == 0 || attestation.ManifestArtifactRef.String() == "" ||
		attestation.ReportArtifactRef.String() == "" {
		t.Fatalf("failed evidence not durable: %+v", record)
	}
}

func TestAgentObservationAndLaunchReceiptCannotProveRequiredTests(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	record := system.record(t)
	if len(record.Attestations) != 1 || record.Attestations[0].Kind != AttestationKindArtifactProvenance ||
		record.Attestations[0].Verdict != AttestationVerdictObserved {
		t.Fatalf("agent observation became test evidence: %+v", record.Attestations)
	}
	for _, receipt := range record.EffectReceipts {
		if receipt.Status == EffectStatusAttestedPassed {
			t.Fatalf("launch/commit receipt became PASS: %+v", receipt)
		}
	}
	_, err := system.orchestrator.IntegrateChange(context.Background(), system.access, IntegrateChangeRequest{
		RequestRef: "request:integrate:agent-evidence", GoalRef: record.Goal.Ref(),
		ChangeRef: record.ChangeSets[0].Ref, ExpectedTargetOID: record.WorkspaceBindings[0].BaseOID,
	})
	if !IsStateError(err, StateConflict) {
		t.Fatalf("agent evidence admitted integration: %v", err)
	}
}

func TestLegacyWriteCandidateWithoutRequiredTestsRemainsPendingUnattestable(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.repository.mu.Lock()
	record := system.repository.records[system.goalRef]
	snapshot := record.Goal.Snapshot()
	snapshot.SchemaVersion = 6
	for index := range snapshot.WorkItems {
		snapshot.WorkItems[index].RequiredTests = nil
	}
	legacy, err := goal.RestoreGoal(snapshot)
	if err != nil {
		system.repository.mu.Unlock()
		t.Fatal(err)
	}
	var removedIntent string
	for ref, action := range system.repository.actions {
		if action.record.GoalRef == system.goalRef && action.record.Kind == ActionAttestTest {
			removedIntent = action.record.EffectIntentRef
			delete(system.repository.actions, ref)
		}
	}
	record.Goal = legacy
	intents := record.EffectIntents[:0]
	for _, intent := range record.EffectIntents {
		if intent.Ref != removedIntent {
			intents = append(intents, intent)
		}
	}
	record.EffectIntents = intents
	approvals := record.EffectApprovals[:0]
	for _, approval := range record.EffectApprovals {
		if approval.IntentRef != removedIntent {
			approvals = append(approvals, approval)
		}
	}
	record.EffectApprovals = approvals
	system.repository.records[system.goalRef] = record
	system.repository.mu.Unlock()

	legacyRecord := system.record(t)
	if len(legacyRecord.Goal.WorkItems()[0].RequiredTests()) != 0 ||
		legacyRecord.Executions[0].State != ExecutionAwaitingAttestation {
		t.Fatalf("legacy candidate=%+v", legacyRecord)
	}
	if result, err := system.orchestrator.ProcessNext(context.Background(), "worker:legacy"); err != nil || result.Processed {
		t.Fatalf("legacy candidate became attestable: result=%+v err=%v", result, err)
	}
	_, err = system.orchestrator.IntegrateChange(context.Background(), system.access, IntegrateChangeRequest{
		RequestRef: "request:integrate:legacy", GoalRef: legacyRecord.Goal.Ref(),
		ChangeRef: legacyRecord.ChangeSets[0].Ref, ExpectedTargetOID: legacyRecord.WorkspaceBindings[0].BaseOID,
	})
	if !IsStateError(err, StateConflict) {
		t.Fatalf("legacy candidate integrated without tests: %v", err)
	}
	if len(system.attestor.requests) != 0 || len(legacyRecord.ChangeSets) != 1 ||
		len(legacyRecord.IntegrationReceipts) != 0 {
		t.Fatalf("legacy candidate mutated: %+v", legacyRecord)
	}
}
