package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
	"orquesta/internal/review"
)

func TestSequentialReviewerLaunchesCannotShareExternalProcess(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest)
	agent := system.orchestrator.launcher.(*scriptedAgent)
	agent.launchOverride = func(request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
		return ports.AgentLaunchReceipt{ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef,
			WorkItemRef: request.WorkItemRef, PlanGeneration: request.PlanGeneration,
			AppSpecGeneration: request.AppSpecGeneration, ExecutionAttempt: request.ExecutionAttempt,
			SpecHash: request.SpecHash, ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test",
			ExternalRef: "external:shared-review-process", ReceiptRef: "receipt:" + request.ExecutionRef.String(),
			IdempotencyKey: request.IdempotencyKey, AcceptedAt: system.orchestrator.clock.Now()}, nil
	}
	system.process(t, ActionLaunchAgent, ActionLaunchAgent)
	mid := system.record(t)
	running, reused, duplicateReceipts := 0, 0, 0
	for _, execution := range mid.Executions {
		if isReviewerExecution(execution) && execution.State == ExecutionRunning {
			running++
		}
		if isReviewerExecution(execution) && execution.FailureCode == "review.external_ref_reused" {
			reused++
		}
	}
	for _, receipt := range mid.ConsumptionReceipts {
		if receipt.ErrorCode == "review.external_ref_reused" && receipt.Outcome == ActionConsumedQuarantined &&
			receipt.EffectReceiptRef != "" {
			duplicateReceipts++
		}
	}
	if running != 1 || reused != 1 || duplicateReceipts != 1 || system.actionKindCount(ActionStopAgent) != 1 {
		t.Fatalf("shared process frontier running=%d reused=%d receipts=%d stops=%d", running, reused,
			duplicateReceipts, system.actionKindCount(ActionStopAgent))
	}
	system.process(t, ActionStopAgent)
	assertReviewRoundAborted(t, system, "review.external_ref_reused")
}

func TestReviewAbortAtRecoveredSiblingDispatchWaitsForLaunchReceiptThenStopsOwnedProcess(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest)
	system.repository.mu.Lock()
	record := system.repository.records[system.goalRef]
	var delayed ExecutionRecord
	for index := range record.Executions {
		if !isReviewerExecution(record.Executions[index]) {
			continue
		}
		record.Executions[index].MaxExecutionAttempts = 1
		if delayed.Ref.String() == "" {
			delayed = record.Executions[index]
		}
	}
	system.repository.records[system.goalRef] = record
	system.repository.mu.Unlock()
	system.process(t, ActionLaunchAgent)

	// Recreate the durable frontier after launch preparation but before its
	// provider receipt. The action remains the only authority to resume it.
	system.repository.mu.Lock()
	record = system.repository.records[system.goalRef]
	for index := range record.Executions {
		if record.Executions[index].Ref == delayed.Ref && record.Executions[index].State == ExecutionQueued {
			record.Executions[index].State = ExecutionDispatching
		}
	}
	for ref, action := range system.repository.actions {
		if action.record.ExecutionRef == delayed.Ref && action.record.Kind == ActionLaunchAgent {
			action.record.AvailableAt = system.orchestrator.clock.Now().Add(time.Minute)
			system.repository.actions[ref] = action
		}
	}
	system.repository.records[system.goalRef] = record
	system.repository.mu.Unlock()

	agent := system.orchestrator.launcher.(*scriptedAgent)
	agent.mu.Lock()
	agent.observations = []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "application/vnd.orquesta.review-assessment+json",
		Content: []byte(`{"malformed":true}`),
	}}
	agent.mu.Unlock()
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:dispatch-frontier-observe")
	if err != nil || result.Action != ActionObserveAgent {
		t.Fatalf("terminal reviewer observation result=%+v err=%v", result, err)
	}
	mid := system.record(t)
	item := mid.Goal.WorkItems()[0]
	if item.State() != goal.WorkItemStateRunning || system.actionKindCount(ActionStopAgent) != 0 {
		t.Fatalf("dispatching sibling fake-terminalized: item=%s stops=%d", item.State(), system.actionKindCount(ActionStopAgent))
	}
	dispatching, cleanupMarkers := 0, 0
	for _, execution := range mid.Executions {
		if isReviewerExecution(execution) && execution.State == ExecutionDispatching {
			dispatching++
		}
	}
	for _, control := range mid.Controls {
		if IsReviewCleanupControl(control) && control.Status == ControlRequested {
			cleanupMarkers++
		}
	}
	if dispatching != 1 || cleanupMarkers != 1 {
		t.Fatalf("dispatch cleanup frontier executions=%+v controls=%+v", mid.Executions, mid.Controls)
	}
	system.orchestrator.clock.(*mutableClock).Advance(2 * time.Minute)
	result, err = system.orchestrator.ProcessNext(context.Background(), "worker:dispatch-frontier-launch")
	if err != nil || result.Action != ActionLaunchAgent {
		t.Fatalf("sibling launch completion result=%+v err=%v", result, err)
	}
	if system.actionKindCount(ActionStopAgent) != 1 || system.actionKindCount(ActionObserveAgent) != 0 {
		t.Fatalf("accepted cleanup route stops=%d observes=%d", system.actionKindCount(ActionStopAgent),
			system.actionKindCount(ActionObserveAgent))
	}
	system.process(t, ActionStopAgent)
	assertReviewRoundAborted(t, system, "review.assessment_invalid")
	agent.mu.Lock()
	stopCalls := append([]ports.AgentStopRequest(nil), agent.stopRequests...)
	agent.mu.Unlock()
	if len(stopCalls) != 1 || stopCalls[0].ExecutionRef != delayed.Ref || stopCalls[0].ExternalRef == "" {
		t.Fatalf("owned cleanup stop calls=%+v", stopCalls)
	}
}

func TestDispatchingReviewCleanupDefinitelyUnappliedResolvesLocally(t *testing.T) {
	system, delayed := reviewCleanupDispatchFrontier(t)
	agent := system.orchestrator.launcher.(*scriptedAgent)
	agent.launchErr = definitelyUnappliedPermanentError{message: "provider rejected before launch"}
	system.orchestrator.clock.(*mutableClock).Advance(2 * time.Minute)
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:cleanup-unapplied")
	if err != nil || !result.Processed || result.Action != ActionLaunchAgent {
		t.Fatalf("cleanup launch result=%+v err=%v", result, err)
	}
	after := system.record(t)
	item := after.Goal.WorkItems()[0]
	resolved, failed := 0, 0
	for _, control := range after.Controls {
		if IsReviewCleanupControl(control) && control.ExecutionRef == delayed.Ref &&
			control.Status == ControlConfirmed && control.ReceiptRef != "" {
			resolved++
		}
	}
	for _, execution := range after.Executions {
		if execution.Ref == delayed.Ref && execution.State == ExecutionFailed &&
			execution.FailureCode == "review.round_aborted" {
			failed++
		}
	}
	if item.State() != goal.WorkItemStateInterrupted || resolved != 1 || failed != 1 ||
		system.actionKindCount(ActionStopAgent) != 0 {
		t.Fatalf("local cleanup item=%s resolved=%d failed=%d stops=%d", item.State(), resolved, failed,
			system.actionKindCount(ActionStopAgent))
	}
}

func TestDispatchingReviewCleanupUnknownAppliedRemainsObservable(t *testing.T) {
	system, delayed := reviewCleanupDispatchFrontier(t)
	agent := system.orchestrator.launcher.(*scriptedAgent)
	agent.launchErr = errors.New("provider result unknown")
	system.orchestrator.clock.(*mutableClock).Advance(2 * time.Minute)
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:cleanup-unknown")
	if err == nil || !strings.Contains(err.Error(), effectUnknownAppliedCode) ||
		!result.Processed || result.Action != ActionLaunchAgent {
		t.Fatalf("unknown cleanup result=%+v err=%v", result, err)
	}
	after := system.record(t)
	item := after.Goal.WorkItems()[0]
	pending, dispatching := 0, 0
	for _, control := range after.Controls {
		if IsReviewCleanupControl(control) && control.ExecutionRef == delayed.Ref && control.Status == ControlRequested {
			pending++
		}
	}
	for _, execution := range after.Executions {
		if execution.Ref == delayed.Ref && execution.State == ExecutionDispatching {
			dispatching++
		}
	}
	if item.State() != goal.WorkItemStateRunning || pending != 1 || dispatching != 1 {
		t.Fatalf("unknown cleanup lost frontier item=%s pending=%d dispatching=%d", item.State(), pending, dispatching)
	}
}

func TestDispatchingReviewCleanupDuplicateProcessResolvesWithoutStoppingForeignProcess(t *testing.T) {
	system, delayed := reviewCleanupDispatchFrontier(t)
	record := system.record(t)
	author, _ := authorBound(record, record.Goal.WorkItems()[0])
	agent := system.orchestrator.launcher.(*scriptedAgent)
	agent.launchOverride = func(request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
		return ports.AgentLaunchReceipt{ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef,
			WorkItemRef: request.WorkItemRef, PlanGeneration: request.PlanGeneration,
			AppSpecGeneration: request.AppSpecGeneration, ExecutionAttempt: request.ExecutionAttempt,
			SpecHash: request.SpecHash, ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test",
			ExternalRef: author.ExternalRef, ReceiptRef: "receipt:foreign-process",
			IdempotencyKey: request.IdempotencyKey, AcceptedAt: system.orchestrator.clock.Now()}, nil
	}
	system.orchestrator.clock.(*mutableClock).Advance(2 * time.Minute)
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:cleanup-duplicate")
	if err != nil || !result.Processed || result.Action != ActionLaunchAgent {
		t.Fatalf("duplicate cleanup result=%+v err=%v", result, err)
	}
	after := system.record(t)
	resolved := 0
	for _, control := range after.Controls {
		if IsReviewCleanupControl(control) && control.ExecutionRef == delayed.Ref && control.Status == ControlConfirmed {
			resolved++
		}
	}
	if after.Goal.WorkItems()[0].State() != goal.WorkItemStateInterrupted || resolved != 1 ||
		system.actionKindCount(ActionStopAgent) != 0 || len(agent.stopRequests) != 0 {
		t.Fatalf("duplicate cleanup item=%s resolved=%d stop-actions=%d stop-requests=%d",
			after.Goal.WorkItems()[0].State(), resolved, system.actionKindCount(ActionStopAgent), len(agent.stopRequests))
	}
}

func TestChangesRequestedAssessmentThenUnavailableReviewerDoesNotReplan(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest)
	system.repository.mu.Lock()
	record := system.repository.records[system.goalRef]
	for index := range record.Executions {
		if isReviewerExecution(record.Executions[index]) {
			record.Executions[index].MaxExecutionAttempts = 1
		}
	}
	system.repository.records[system.goalRef] = record
	system.repository.mu.Unlock()
	system.process(t, ActionLaunchAgent, ActionLaunchAgent)
	record = system.record(t)
	var first ExecutionRecord
	for _, action := range system.repository.actions {
		if action.record.Kind != ActionObserveAgent {
			continue
		}
		execution, _ := executionByRef(record.Executions, action.record.ExecutionRef)
		if first.Ref.String() == "" || action.record.Ref < "action:observe:"+first.Ref.String() {
			first = execution
		}
	}
	role, _ := reviewerRole(first)
	payload, err := json.Marshal(review.Artifact{SchemaVersion: 1, SubjectDigest: first.ReviewSubjectDigest,
		Role: role, Verdict: review.VerdictChangesRequested, Summary: "material correction required",
		Findings: []review.Finding{{Code: "review.finding", Severity: review.SeverityHigh,
			EvidenceRef: "change:exact"}}})
	if err != nil {
		t.Fatal(err)
	}
	agent := system.orchestrator.launcher.(*scriptedAgent)
	agent.observations = []ports.AgentObservation{
		{Status: ports.AgentCompleted, MediaType: review.AssessmentMediaType, Content: payload},
		{Status: ports.AgentCompleted, MediaType: review.AssessmentMediaType, Content: []byte(`{"bad":true}`)},
	}
	system.process(t, ActionObserveAgent)
	if system.record(t).Goal.WorkItems()[0].State() != goal.WorkItemStateRunning {
		t.Fatal("one changes_requested assessment must not decide the pair gate")
	}
	system.process(t, ActionObserveAgent)
	after := system.record(t)
	author, _ := authorBound(after, after.Goal.WorkItems()[0])
	if author.State != ExecutionFailed || author.FailureCode != "review.unavailable" ||
		author.FailureCode == string(goal.ReplanCauseReviewChangesRequested) {
		t.Fatalf("incomplete pair failure=%+v", author)
	}
}

func TestReviewCleanupControlUsesOnlyExactAuthorLaunchAuthority(t *testing.T) {
	system, delayed := reviewCleanupDispatchFrontier(t)
	record := system.record(t)
	var cleanup ControlRecord
	for _, candidate := range record.Controls {
		if IsReviewCleanupControl(candidate) && candidate.ExecutionRef == delayed.Ref {
			cleanup = candidate
		}
	}
	if err := ValidatePersistedControlRecord(cleanup); err != nil {
		t.Fatalf("valid cleanup authority rejected: %v", err)
	}
	otherPrincipal, _ := identity.NewPrincipalRef("principal:other")
	otherProject, _ := goal.NewProjectRef("project:other")
	mutations := []func(*ControlRecord){
		func(value *ControlRecord) { value.PrincipalRef = otherPrincipal },
		func(value *ControlRecord) { value.ProjectRef = otherProject },
		func(value *ControlRecord) {
			value.AuthorizationReceipt = alteredCleanupAuthority(t, *value, identity.PermissionGoalsGet,
				value.ProjectRef.String(), "authorization-receipt:wrong-permission")
		},
		func(value *ControlRecord) {
			value.AuthorizationReceipt = alteredCleanupAuthority(t, *value, identity.PermissionGoalsCreate,
				"project:other", "authorization-receipt:wrong-resource")
		},
		func(value *ControlRecord) {
			value.RequestedAt = value.AuthorizationReceipt.RecordedAt().Add(-time.Nanosecond)
		},
	}
	for index, mutate := range mutations {
		candidate := cleanup
		mutate(&candidate)
		if ValidatePersistedControlRecord(candidate) == nil {
			t.Fatalf("cleanup authority mutation %d accepted", index)
		}
	}
	normal := cleanup
	normal.Ref, normal.RequestRef, normal.Reason = "control:not-cleanup", "request:not-cleanup", "ordinary stop"
	normal.RequestFingerprint = controlFingerprint(normal.PrincipalRef, normal.ProjectRef, ControlRequest{
		RequestRef: normal.RequestRef, Operation: normal.Operation, Target: normal.Target,
		GoalRef: normal.GoalRef, ExpectedGoalRevision: normal.GoalRevision,
		ExpectedPlanGeneration: normal.PlanGeneration, ExpectedAppSpecGeneration: normal.AppSpecGeneration,
		ExpectedSpecHash: normal.SpecHash, WorkItemRef: normal.WorkItemRef,
		ExpectedWorkItemRevision: normal.WorkItemRevision, ExecutionRef: normal.ExecutionRef,
		ExpectedExecutionAttempt: normal.ExecutionAttempt, Mode: normal.Mode, Reason: normal.Reason,
	})
	if ValidatePersistedControlRecord(normal) == nil {
		t.Fatal("ordinary control used review cleanup authority exception")
	}
}

func alteredCleanupAuthority(t *testing.T, control ControlRecord, permission identity.Permission,
	resourceRef, receiptRef string,
) identity.AuthorizationReceipt {
	t.Helper()
	original := control.AuthorizationReceipt
	request, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: original.Decision().Request().RequestRef(),
		Principal:  original.Decision().Request().Principal(), ProjectRef: control.ProjectRef,
		Permission: permission, ResourceRef: resourceRef, RequestedAt: original.Decision().Request().RequestedAt(),
	})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
		Request: request, Outcome: identity.AuthorizationAllowed, Role: identity.RolePlatformAdmin,
		ReasonCode: "access.allowed", DecidedAt: original.Decision().DecidedAt(),
	})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := identity.NewAuthorizationReceipt(identity.AuthorizationReceiptInput{
		Ref: receiptRef, Decision: decision, RecordedAt: original.RecordedAt(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func reviewCleanupDispatchFrontier(t *testing.T) (*testAttestationSystem, ExecutionRecord) {
	t.Helper()
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest)
	system.repository.mu.Lock()
	record := system.repository.records[system.goalRef]
	for index := range record.Executions {
		if isReviewerExecution(record.Executions[index]) {
			record.Executions[index].MaxExecutionAttempts = 1
		}
	}
	system.repository.records[system.goalRef] = record
	system.repository.mu.Unlock()
	system.process(t, ActionLaunchAgent)
	system.repository.mu.Lock()
	record = system.repository.records[system.goalRef]
	var delayed ExecutionRecord
	for index := range record.Executions {
		if isReviewerExecution(record.Executions[index]) && record.Executions[index].State == ExecutionQueued {
			record.Executions[index].State = ExecutionDispatching
			delayed = record.Executions[index]
		}
	}
	for ref, action := range system.repository.actions {
		if action.record.ExecutionRef == delayed.Ref && action.record.Kind == ActionLaunchAgent {
			action.record.AvailableAt = system.orchestrator.clock.Now().Add(time.Minute)
			system.repository.actions[ref] = action
		}
	}
	system.repository.records[system.goalRef] = record
	system.repository.mu.Unlock()
	if delayed.Ref.String() == "" {
		t.Fatal("queued review sibling missing")
	}
	agent := system.orchestrator.launcher.(*scriptedAgent)
	agent.observations = []ports.AgentObservation{{Status: ports.AgentCompleted,
		MediaType: review.AssessmentMediaType, Content: []byte(`{"malformed":true}`)}}
	system.process(t, ActionObserveAgent)
	return system, delayed
}

func TestCancelReviewCohortStopsOnceAndClosesWorkItemOnce(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest, ActionLaunchAgent)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	grantReviewControl(t, system)
	request := reviewControlRequest(record, "control:cancel-review-cohort", ControlCancel,
		ControlTargetWorkItem, item.Ref(), goal.ExecutionRef{})
	result, err := system.orchestrator.Control(context.Background(), system.access, request)
	if err != nil || result.Control.Status != ControlRequested || system.actionKindCount(ActionStopAgent) != 1 {
		t.Fatalf("cancel cohort result=%+v stops=%d err=%v", result, system.actionKindCount(ActionStopAgent), err)
	}
	system.process(t, ActionStopAgent)
	after := system.record(t)
	item, _ = after.Goal.WorkItem(item.Ref())
	control, found := controlByRef(after.Controls, result.Control.Ref)
	if !found || item.State() != goal.WorkItemStateCanceled || control.Status != ControlConfirmed ||
		system.actionKindCount(ActionStopAgent) != 0 || system.actionKindCount(ActionObserveAgent) != 0 ||
		system.actionKindCount(ActionLaunchAgent) != 0 {
		t.Fatalf("cancel cohort not quiescent: item=%s control=%+v actions=%d", item.State(), control,
			system.actionKindCount(ActionStopAgent)+system.actionKindCount(ActionObserveAgent)+system.actionKindCount(ActionLaunchAgent))
	}
	states := map[ExecutionState]int{}
	for _, execution := range after.Executions {
		states[execution.State]++
	}
	if states[ExecutionStopped] != 1 || states[ExecutionCanceled] != 2 {
		t.Fatalf("cancel cohort states=%+v", states)
	}
}

func TestStopExactReviewerAbortsAttachedRoundWithoutRebindingAuthor(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	grantReviewControl(t, system)
	author, found := authorBound(record, item)
	if !found {
		t.Fatal("author binding missing")
	}
	var target ExecutionRecord
	for _, execution := range record.Executions {
		if isReviewerExecution(execution) && execution.State == ExecutionRunning {
			target = execution
			break
		}
	}
	request := reviewControlRequest(record, "control:stop-exact-reviewer", ControlStop,
		ControlTargetExecution, item.Ref(), target.Ref)
	result, err := system.orchestrator.Control(context.Background(), system.access, request)
	if err != nil || result.Control.Status != ControlRequested || system.actionKindCount(ActionStopAgent) != 1 {
		t.Fatalf("stop reviewer result=%+v actions=%d err=%v", result, system.actionKindCount(ActionStopAgent), err)
	}
	system.process(t, ActionStopAgent)
	if system.actionKindCount(ActionStopAgent) != 1 {
		t.Fatalf("review cleanup stop actions=%d", system.actionKindCount(ActionStopAgent))
	}
	system.process(t, ActionStopAgent)
	after := system.record(t)
	item, _ = after.Goal.WorkItem(item.Ref())
	bound, _ := item.Execution()
	if item.State() != goal.WorkItemStateInterrupted || bound != author.Ref ||
		system.actionKindCount(ActionObserveAgent) != 0 || system.actionKindCount(ActionStopAgent) != 0 {
		t.Fatalf("reviewer stop frontier item=%s bound=%s actions=%d", item.State(), bound,
			system.actionKindCount(ActionObserveAgent)+system.actionKindCount(ActionStopAgent))
	}
	stopped, retired, failedAuthor := 0, 0, 0
	for _, execution := range after.Executions {
		switch {
		case execution.Ref == target.Ref && execution.State == ExecutionStopped:
			stopped++
		case isReviewerExecution(execution) && execution.Ref != target.Ref &&
			execution.State == ExecutionStopped && execution.FailureCode == "review.round_aborted":
			retired++
		case execution.Ref == author.Ref && execution.State == ExecutionFailed &&
			execution.FailureCode == "review.unavailable":
			failedAuthor++
		}
	}
	if stopped != 1 || retired != 1 || failedAuthor != 1 {
		t.Fatalf("stop terminal cohort stopped=%d retired=%d author=%d", stopped, retired, failedAuthor)
	}
}

func TestMalformedReviewerPayloadPersistsRedactedDiagnosticBeforeAbort(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest)
	system.repository.mu.Lock()
	record := system.repository.records[system.goalRef]
	for index := range record.Executions {
		if isReviewerExecution(record.Executions[index]) {
			record.Executions[index].MaxExecutionAttempts = 1
		}
	}
	system.repository.records[system.goalRef] = record
	system.repository.mu.Unlock()
	system.process(t, ActionLaunchAgent, ActionLaunchAgent)
	const secretPayload = `{"token":"must-not-be-persisted","verdict":"approve"}`
	agent := system.orchestrator.launcher.(*scriptedAgent)
	agent.mu.Lock()
	agent.observations = []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "application/vnd.orquesta.review-assessment+json",
		Content: []byte(secretPayload),
	}}
	agent.mu.Unlock()
	system.process(t, ActionObserveAgent)
	after := system.record(t)
	assertReviewRoundAborted(t, system, "review.assessment_invalid")
	var diagnostic ArtifactRecord
	for _, artifact := range after.Artifacts {
		if artifact.Kind == ArtifactKindReviewDiagnostic {
			diagnostic = artifact
		}
	}
	if diagnostic.Stored.Ref.String() == "" {
		t.Fatal("redacted diagnostic artifact missing")
	}
	content, err := system.orchestrator.artifacts.Get(context.Background(), diagnostic.Stored.Ref, diagnostic.Stored.Size)
	if err != nil {
		t.Fatal(err)
	}
	var evidence reviewDiagnosticEvidence
	if json.Unmarshal(content.Content, &evidence) != nil || evidence.ErrorCode != "review.assessment_invalid" ||
		evidence.PayloadBytes != int64(len(secretPayload)) || evidence.PayloadSHA256 == "" ||
		strings.Contains(string(content.Content), "must-not-be-persisted") {
		t.Fatalf("diagnostic evidence=%+v content=%s", evidence, content.Content)
	}
}

func reviewControlRequest(record GoalRecord, ref string, operation ControlOperation, target ControlTarget,
	itemRef goal.WorkItemRef, executionRef goal.ExecutionRef,
) ControlRequest {
	item, _ := record.Goal.WorkItem(itemRef)
	request := ControlRequest{
		RequestRef: ref, Operation: operation, Target: target, GoalRef: record.Goal.Ref(),
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: record.Goal.AppSpec().Generation(), ExpectedSpecHash: record.Goal.SpecHash(),
		WorkItemRef: itemRef, ExpectedWorkItemRevision: item.Revision(), Reason: "review cohort control",
	}
	if executionRef.String() != "" {
		execution, _ := executionByRef(record.Executions, executionRef)
		request.ExecutionRef, request.ExpectedExecutionAttempt = executionRef, execution.AttemptNo
	}
	if operation == ControlStop {
		request.Mode = ports.AgentStopCooperative
	}
	return request
}

func grantReviewControl(t *testing.T, system *testAttestationSystem) {
	t.Helper()
	principal, project, err := system.access.values()
	if err != nil {
		t.Fatal(err)
	}
	store, ok := system.orchestrator.access.(*memoryAccessRepository)
	if !ok {
		t.Fatal("memory access adapter missing")
	}
	seedDirectorMembership(t, store, principal, project, identity.RoleProjectOwner, system.orchestrator.clock.Now())
}
