package application

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestReviewAttachmentKeepsAuthorAsOnlyWorkItemAuthority(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	author, found := authorBound(record, item)
	if !found || author.Purpose != ExecutionPurposeAuthor {
		t.Fatalf("author binding=%+v found=%v", author, found)
	}
	for _, execution := range record.Executions {
		if !isReviewerExecution(execution) {
			continue
		}
		attachment, err := reviewAttached(record, item, execution, system.orchestrator.testAttestationPolicy)
		if err != nil || attachment.Author.Ref != author.Ref || attachment.Subject.Digest() != execution.ReviewSubjectDigest {
			t.Fatalf("attachment=%+v err=%v", attachment, err)
		}
		bound, _ := item.Execution()
		if bound != author.Ref || bound == execution.Ref {
			t.Fatalf("reviewer replaced author binding: bound=%s reviewer=%s", bound, execution.Ref)
		}
	}
}

func TestReviewerLaunchRequestCarriesCanonicalExactEvidenceAndNoWriteSet(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	phase, _ := phaseForWorkItem(record.Goal, item)
	var participant ExecutionRecord
	for _, execution := range record.Executions {
		if isReviewerExecution(execution) {
			participant = execution
			break
		}
	}
	attachment, err := reviewAttached(record, item, participant, system.orchestrator.testAttestationPolicy)
	appTestNoError(t, err)
	attachment.Change.ChangedPaths = []string{"internal/path, with space.go"}
	role, _ := reviewerRole(participant)
	request, err := reviewerAgentLaunchRequestFromAttachment(record, item, participant, phase, role, attachment)
	appTestNoError(t, err)
	if len(request.WriteSet) != 0 || request.ExecutionWorkspaceRef != participant.ExecutionWorkspaceRef ||
		!strings.Contains(request.Objective, "git diff "+attachment.Change.BaseOID+".."+attachment.Change.HeadOID) {
		t.Fatalf("review request not exact/read-only: %+v", request)
	}
	prefix := "Review exact immutable evidence "
	end := strings.Index(request.Objective, ". Inspect with ")
	if !strings.HasPrefix(request.Objective, prefix) || end < 0 {
		t.Fatalf("evidence envelope missing: %q", request.Objective)
	}
	var evidence reviewLaunchEvidence
	if err := json.Unmarshal([]byte(request.Objective[len(prefix):end]), &evidence); err != nil ||
		len(evidence.ChangedPaths) != 1 || evidence.ChangedPaths[0] != "internal/path, with space.go" ||
		evidence.SubjectDigest != participant.ReviewSubjectDigest || evidence.ChangeDigest == "" ||
		len(evidence.RequiredTests) == 0 {
		t.Fatalf("canonical evidence=%+v err=%v", evidence, err)
	}
}

func TestReviewerLaunchRejectsAuthorProcessReuseBeforeObservation(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest)
	record := system.record(t)
	author := onlyExecution(t, record)
	agent := system.orchestrator.launcher.(*scriptedAgent)
	agent.launchOverride = func(request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
		return ports.AgentLaunchReceipt{ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef,
			WorkItemRef: request.WorkItemRef, PlanGeneration: request.PlanGeneration,
			AppSpecGeneration: request.AppSpecGeneration, ExecutionAttempt: request.ExecutionAttempt,
			SpecHash: request.SpecHash, ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test",
			ExternalRef: author.ExternalRef, ReceiptRef: "receipt:duplicate-process", IdempotencyKey: request.IdempotencyKey,
			AcceptedAt: system.orchestrator.clock.Now()}, nil
	}
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:duplicate-review-process")
	if !result.Processed || result.Action != ActionLaunchAgent || err != nil {
		t.Fatalf("duplicate process result=%+v err=%v", result, err)
	}
	assertReviewRoundAborted(t, system, "review.external_ref_reused")
}

func TestAdversarialLaunchRejectsPrimaryProcessReuseBeforeObservation(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest, ActionLaunchAgent)
	record := system.record(t)
	var primary ExecutionRecord
	for _, execution := range record.Executions {
		if isReviewerExecution(execution) && execution.State == ExecutionRunning {
			primary = execution
		}
	}
	if primary.State != ExecutionRunning || primary.ExternalRef == "" {
		t.Fatalf("primary not running: %+v", primary)
	}
	agent := system.orchestrator.launcher.(*scriptedAgent)
	agent.launchOverride = func(request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
		return ports.AgentLaunchReceipt{ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef,
			WorkItemRef: request.WorkItemRef, PlanGeneration: request.PlanGeneration,
			AppSpecGeneration: request.AppSpecGeneration, ExecutionAttempt: request.ExecutionAttempt,
			SpecHash: request.SpecHash, ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test",
			ExternalRef: primary.ExternalRef, ReceiptRef: "receipt:duplicate-reviewer-process",
			IdempotencyKey: request.IdempotencyKey, AcceptedAt: system.orchestrator.clock.Now()}, nil
	}
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:duplicate-reviewer-process")
	if !result.Processed || result.Action != ActionLaunchAgent || err != nil {
		t.Fatalf("duplicate reviewer process result=%+v err=%v", result, err)
	}
	assertReviewRoundAborted(t, system, "review.external_ref_reused")
}

func assertReviewRoundAborted(t *testing.T, system *testAttestationSystem, currentCode string) {
	t.Helper()
	for attempts := 0; attempts < 4 && system.actionKindCount(ActionStopAgent) != 0; attempts++ {
		system.process(t, ActionStopAgent)
	}
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	if item.State() != goal.WorkItemStateInterrupted || system.actionKindCount(ActionLaunchAgent) != 0 ||
		system.actionKindCount(ActionObserveAgent) != 0 || system.actionKindCount(ActionStopAgent) != 0 {
		t.Fatalf("review round remained live: item=%s launches=%d observes=%d stops=%d", item.State(),
			system.actionKindCount(ActionLaunchAgent), system.actionKindCount(ActionObserveAgent),
			system.actionKindCount(ActionStopAgent))
	}
	authorFailures, reviewerTerminals := 0, 0
	for _, execution := range record.Executions {
		switch {
		case execution.Purpose == ExecutionPurposeAuthor && execution.State == ExecutionFailed &&
			execution.FailureCode == "review.unavailable":
			authorFailures++
		case isReviewerExecution(execution) &&
			(execution.State == ExecutionFailed || execution.State == ExecutionStopped):
			if execution.FailureCode != currentCode && execution.FailureCode != "review.round_aborted" {
				t.Fatalf("reviewer terminal code=%s", execution.FailureCode)
			}
			reviewerTerminals++
		}
	}
	if authorFailures != 1 || reviewerTerminals != 2 {
		t.Fatalf("terminal cohort author=%d reviewers=%d", authorFailures, reviewerTerminals)
	}
	terminalReceipts := 0
	for _, receipt := range record.ConsumptionReceipts {
		if receipt.ErrorCode != currentCode {
			continue
		}
		if currentCode == "review.external_ref_reused" {
			if receipt.Outcome == ActionConsumedQuarantined && receipt.EffectReceiptRef != "" {
				terminalReceipts++
			}
		} else if receipt.Outcome == ActionConsumedCompleted {
			terminalReceipts++
		}
	}
	if terminalReceipts != 1 {
		t.Fatalf("review failure receipts=%d code=%s", terminalReceipts, currentCode)
	}
}

func (system *testAttestationSystem) actionKindCount(kind ActionKind) int {
	system.repository.mu.Lock()
	defer system.repository.mu.Unlock()
	count := 0
	for _, action := range system.repository.actions {
		if action.record.Kind == kind {
			count++
		}
	}
	return count
}

func TestReviewAttachmentRejectsCrossWorkspaceAndStaleSubject(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	var reviewer ExecutionRecord
	for _, execution := range record.Executions {
		if isReviewerExecution(execution) {
			reviewer = execution
			break
		}
	}
	foreignWorkspace, err := ports.NewExecutionWorkspaceRef("execution-workspace:foreign")
	appTestNoError(t, err)
	foreignItem, err := goal.NewWorkItemRef("work-item:foreign")
	appTestNoError(t, err)
	mutations := []func(*ExecutionRecord){
		func(value *ExecutionRecord) { value.ExecutionWorkspaceRef = foreignWorkspace },
		func(value *ExecutionRecord) { value.ReviewSubjectDigest = "sha256:" + testDigest("stale") },
		func(value *ExecutionRecord) { value.WorkItemRef = foreignItem },
	}
	for index, mutate := range mutations {
		candidate := reviewer
		mutate(&candidate)
		if _, err := reviewAttached(record, item, candidate, system.orchestrator.testAttestationPolicy); err == nil {
			t.Fatalf("invalid attachment %d accepted", index)
		}
	}
}
