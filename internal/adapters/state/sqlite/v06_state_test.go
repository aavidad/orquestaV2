package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestV06ClaimUsesRepositoryClockAndOpaqueCapabilityMatching(t *testing.T) {
	repository, _ := openTestRepository(t)
	state := newCreateFixture(t, "v06-caps", "request:v06-caps", "fingerprint:v06-caps", "actor:v06", "project:v06")
	if _, _, err := createLegacyGoal(t, repository, state); err != nil {
		t.Fatalf("create capability fixture: %v", err)
	}
	itemRef := state.Executions[0].WorkItemRef.String()
	for position, requirement := range []struct{ kind, value string }{
		{kind: "skill", value: "skill:go"},
		{kind: "tool", value: "tool:test"},
		{kind: "capability", value: "capability:patch"},
	} {
		if _, err := repository.db.Exec(`
INSERT INTO work_item_requirement_refs(goal_ref, work_item_ref, kind, value, position)
VALUES (?, ?, ?, ?, ?)`, state.Goal.Ref().String(), itemRef, requirement.kind, requirement.value, position); err != nil {
			t.Fatalf("insert %s requirement: %v", requirement.kind, err)
		}
	}
	base := state.Executions[0].CreatedAt
	repository.now = func() time.Time { return base }
	missing := ports.AgentCapabilities{
		ProviderRef: "provider:v06", ModelRef: "model:v06", AgentRef: "agent:v06",
		RoleKeys: []string{goal.DefaultRoleKey().String()}, SkillRefs: []string{"skill:go"},
		ToolRefs: []string{"tool:test"},
	}
	if _, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:missing-capability", Token: "claim:missing-capability",
		LeaseDuration: time.Minute, Capabilities: missing,
	}); err != nil || found {
		t.Fatalf("insufficient capabilities claimed = found:%v err:%v", found, err)
	}
	exact := missing
	exact.CapabilityRefs = []string{"capability:patch"}
	repository.now = func() time.Time { return base.Add(-time.Nanosecond) }
	if _, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:before-available", Token: "claim:before-available",
		LeaseDuration: time.Minute, Capabilities: exact,
	}); err != nil || found {
		t.Fatalf("repository clock before availability = found:%v err:%v", found, err)
	}
	repository.now = func() time.Time { return base }
	claim, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:exact", Token: "claim:exact", LeaseDuration: time.Minute, Capabilities: exact,
	})
	if err != nil || !found || claim.DeliveryAttempt != 1 || claim.Fence != 1 ||
		!claim.LeaseUntil.Equal(base.Add(time.Minute)) {
		t.Fatalf("exact capability claim = %+v found:%v err:%v", claim, found, err)
	}
}

func TestV06ExecutionReplacementRoundTripsReceiptFenceAndRestart(t *testing.T) {
	repository, path := openTestRepository(t)
	replacement, oldClaim := buildReplacementState(t, repository, "roundtrip")
	repository.now = func() time.Time { return replacement.OperationAt }
	if err := repository.RecordExecutionReplaced(context.Background(), replacement); err != nil {
		t.Fatalf("record replacement: %v", err)
	}
	record, err := repository.GetGoal(context.Background(), replacement.Goal.Ref())
	if err != nil {
		t.Fatalf("read replacement: %v", err)
	}
	if len(record.Executions) != 2 || record.Executions[0].State != application.ExecutionFailed ||
		record.Executions[1].State != application.ExecutionDispatching ||
		record.Executions[1].AttemptNo != 2 || record.Executions[1].ReplacesExecutionRef != record.Executions[0].Ref ||
		len(record.ConsumptionReceipts) != 1 {
		t.Fatalf("replacement round trip = %+v", record)
	}
	receipt := record.ConsumptionReceipts[0]
	if receipt.ActionRef != oldClaim.Action.Ref || receipt.Fence != oldClaim.Fence ||
		receipt.DeliveryAttempt != oldClaim.DeliveryAttempt || receipt.Outcome != application.ActionConsumedCompleted ||
		receipt.ErrorCode != replacement.ErrorCode {
		t.Fatalf("replacement receipt = %+v", receipt)
	}
	if _, err := repository.db.Exec(`UPDATE action_consumption_receipts SET error_code = 'changed' WHERE action_ref = ?`, receipt.ActionRef); err == nil {
		t.Fatal("immutable receipt updated")
	}
	if _, err := repository.db.Exec(`DELETE FROM action_consumption_receipts WHERE action_ref = ?`, receipt.ActionRef); err == nil {
		t.Fatal("immutable receipt deleted")
	}
	stale := application.ActionQuarantinedState{
		Claim: oldClaim, ErrorCode: "application.stale_claim", OperationAt: replacement.OperationAt,
		Event: application.EventRecord{
			Ref: "event:stale:v06", Kind: "action.quarantined", GoalRef: oldClaim.Action.GoalRef,
			WorkItemRef: oldClaim.Action.WorkItemRef, ExecutionRef: oldClaim.Action.ExecutionRef,
			OccurredAt: replacement.OperationAt,
		},
	}
	if err := repository.QuarantineAction(context.Background(), stale); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("stale fenced mutation = %v", err)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("close before replacement restart: %v", err)
	}
	repository, err = Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
		Now: func() time.Time { return replacement.OperationAt },
	})
	if err != nil {
		t.Fatalf("restart replacement repository: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	restarted, err := repository.GetGoal(context.Background(), replacement.Goal.Ref())
	if err != nil || len(restarted.Executions) != 2 || len(restarted.ConsumptionReceipts) != 1 {
		t.Fatalf("replacement restart = %+v err=%v", restarted, err)
	}
	next, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:replacement", Token: "claim:replacement", LeaseDuration: time.Minute,
		Capabilities: sqliteTestCapabilities(),
	})
	if err != nil || !found || next.Action.ExecutionRef != replacement.ReplacementExecution.Ref ||
		next.DeliveryAttempt != 1 || next.Fence != oldClaim.Fence+1 {
		t.Fatalf("replacement claim = %+v found=%v err=%v", next, found, err)
	}
}

func TestV06ReplacementLateEventConflictRollsBackSnapshotActionReceiptAndAttempt(t *testing.T) {
	repository, _ := openTestRepository(t)
	replacement, claim := buildReplacementState(t, repository, "rollback")
	late := replacement.Events[0]
	if _, err := repository.db.Exec(`
INSERT INTO events(ref, kind, goal_ref, work_item_ref, execution_ref, occurred_at)
VALUES (?, ?, ?, ?, ?, ?)`, late.Ref, late.Kind, late.GoalRef.String(), late.WorkItemRef.String(),
		late.ExecutionRef.String(), requiredTime(late.OccurredAt)); err != nil {
		t.Fatalf("seed late event conflict: %v", err)
	}
	repository.now = func() time.Time { return replacement.OperationAt }
	if err := repository.RecordExecutionReplaced(context.Background(), replacement); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("late replacement conflict = %v", err)
	}
	record, err := repository.GetGoal(context.Background(), replacement.Goal.Ref())
	if err != nil || len(record.Executions) != 1 || record.Executions[0].State != application.ExecutionDispatching ||
		len(record.ConsumptionReceipts) != 0 {
		t.Fatalf("replacement rollback record = %+v err=%v", record, err)
	}
	var completedAt sql.NullInt64
	var activeActions, replacementExecutions int
	if err := repository.db.QueryRow(`SELECT completed_at FROM outbox WHERE ref = ?`, claim.Action.Ref).Scan(&completedAt); err != nil {
		t.Fatalf("read original action after rollback: %v", err)
	}
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM outbox WHERE execution_ref = ?`, replacement.ReplacementExecution.Ref.String()).Scan(&activeActions); err != nil {
		t.Fatalf("read replacement action after rollback: %v", err)
	}
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM executions WHERE ref = ?`, replacement.ReplacementExecution.Ref.String()).Scan(&replacementExecutions); err != nil {
		t.Fatalf("read replacement execution after rollback: %v", err)
	}
	if completedAt.Valid || activeActions != 0 || replacementExecutions != 0 {
		t.Fatalf("partial replacement escaped: completed=%+v actions=%d executions=%d", completedAt, activeActions, replacementExecutions)
	}
}

func TestV06GoalMutationRejectsEventFromDifferentExecutionAndRollsBack(t *testing.T) {
	repository, _ := openTestRepository(t)
	state := newV05CreateFixture(t)
	if _, _, err := createLegacyGoal(t, repository, state); err != nil {
		t.Fatalf("create multi-execution Goal: %v", err)
	}
	claim := mustClaim(t, repository, "worker:v06-event-scope", "claim:v06-event-scope", state.Goal.CreatedAt())
	record, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("read multi-execution Goal: %v", err)
	}
	currentItem, found := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !found {
		t.Fatal("claimed WorkItem missing")
	}
	var currentExecution application.ExecutionRecord
	var otherExecution application.ExecutionRecord
	for _, execution := range record.Executions {
		switch execution.Ref {
		case claim.Action.ExecutionRef:
			currentExecution = execution
		default:
			if otherExecution.Ref.String() == "" {
				otherExecution = execution
			}
		}
	}
	if currentExecution.Ref.String() == "" || otherExecution.Ref.String() == "" {
		t.Fatalf("need two execution scopes: %+v", record.Executions)
	}
	operationAt := state.Goal.CreatedAt().Add(time.Second)
	preparedGoal, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), currentItem.Revision(), currentItem.Ref(), currentExecution.Ref, operationAt,
	)
	if err != nil {
		t.Fatalf("start claimed WorkItem: %v", err)
	}
	currentExecution.State = application.ExecutionDispatching
	if err := repository.RecordLaunchPrepared(context.Background(), application.LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: preparedGoal,
		Execution: currentExecution, OperationAt: operationAt,
		Event: application.EventRecord{
			Ref: "event:v06-cross-scope-prepared", Kind: "execution.dispatching",
			GoalRef: preparedGoal.Ref(), WorkItemRef: currentItem.Ref(), ExecutionRef: currentExecution.Ref,
			OccurredAt: operationAt,
		},
	}); err != nil {
		t.Fatalf("persist claimed WorkItem preparation: %v", err)
	}

	preparedRecord, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("read prepared Goal: %v", err)
	}
	preparedItem, _ := preparedRecord.Goal.WorkItem(currentItem.Ref())
	failedGoal, err := preparedRecord.Goal.FailWorkItem(
		preparedRecord.Goal.Revision(), preparedItem.Revision(), preparedItem.Ref(), operationAt,
	)
	if err != nil {
		t.Fatalf("fail claimed WorkItem in domain: %v", err)
	}
	failedExecution := currentExecution
	failedExecution.State = application.ExecutionFailed
	failedExecution.FailureCode = "test.cross_execution_event"
	failedExecution.FinishedAt = operationAt
	wrongScope := application.EventRecord{
		Ref: "event:v06-cross-execution", Kind: "work_item.failed", GoalRef: failedGoal.Ref(),
		WorkItemRef: otherExecution.WorkItemRef, ExecutionRef: otherExecution.Ref, OccurredAt: operationAt,
	}
	err = repository.RecordGoalFailed(context.Background(), application.GoalFailedState{
		Claim: claim, ExpectedGoalRevision: preparedRecord.Goal.Revision(), ExpectedItemRevision: preparedItem.Revision(),
		Goal: failedGoal, Execution: failedExecution, Events: []application.EventRecord{wrongScope}, OperationAt: operationAt,
	})
	if !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("cross-execution event mutation = %v, want state.invalid", err)
	}
	after, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("read after rejected cross-execution event: %v", err)
	}
	afterItem, _ := after.Goal.WorkItem(currentItem.Ref())
	var persistedCurrent application.ExecutionRecord
	for _, execution := range after.Executions {
		if execution.Ref == currentExecution.Ref {
			persistedCurrent = execution
			break
		}
	}
	if afterItem.State() != goal.WorkItemStateRunning ||
		persistedCurrent.State != application.ExecutionDispatching || len(after.ConsumptionReceipts) != 0 {
		t.Fatalf("cross-execution rejection partially committed: %+v", after)
	}
}

func TestV06MutationEventKindsMustMatchTransitions(t *testing.T) {
	state := newCreateFixture(t, "v06-event-kind", "request:v06-event-kind", "fingerprint:v06-event-kind", "actor:v06", "project:v06")
	item := onlyItem(t, state.Goal)
	claim := application.ActionClaim{
		Action: state.Actions[0], Token: "claim:v06-event-kind", WorkerRef: "worker:v06-event-kind",
		DeliveryAttempt: 1, Fence: 1, LeaseUntil: state.Goal.CreatedAt().Add(time.Minute),
	}
	preparedGoal, err := state.Goal.StartWorkItem(
		state.Goal.Revision(), item.Revision(), item.Ref(), state.Executions[0].Ref, state.Goal.CreatedAt(),
	)
	if err != nil {
		t.Fatalf("prepare event-kind Goal: %v", err)
	}
	preparedExecution := state.Executions[0]
	preparedExecution.State = application.ExecutionDispatching
	wrongPrepared := application.LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: state.Goal.Revision(), Goal: preparedGoal,
		Execution: preparedExecution, OperationAt: state.Goal.CreatedAt(),
		Event: application.EventRecord{
			Ref: "event:v06-wrong-prepared", Kind: "execution.accepted", GoalRef: preparedGoal.Ref(),
			WorkItemRef: item.Ref(), ExecutionRef: preparedExecution.Ref, OccurredAt: state.Goal.CreatedAt(),
		},
	}
	if err := validateLaunchPrepared(wrongPrepared); err == nil {
		t.Fatal("LaunchPrepared accepted execution.accepted event")
	}

	runningExecution := preparedExecution
	runningExecution.State = application.ExecutionRunning
	runningExecution.ProviderRef = "provider:v06"
	runningExecution.ModelRef = "model:v06"
	runningExecution.AgentRef = "agent:v06"
	runningExecution.ExternalRef = "external:v06"
	runningExecution.StartedAt = state.Goal.CreatedAt()
	runningExecution.DeadlineAt = state.Goal.CreatedAt().Add(time.Hour)
	runningExecution.ProviderAcceptedAt = state.Goal.CreatedAt()
	preparedItem, _ := preparedGoal.WorkItem(item.Ref())
	wrongAccepted := application.LaunchAcceptedState{
		Claim: claim, Execution: runningExecution, OperationAt: state.Goal.CreatedAt(),
		NextAction: application.ActionRecord{
			Ref: "action:v06-wrong-accepted", Kind: application.ActionObserveAgent,
			GoalRef: preparedGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: runningExecution.Ref,
			PlanGeneration: preparedGoal.PlanGeneration(), WorkItemGeneration: preparedItem.Revision(),
			AvailableAt: state.Goal.CreatedAt(),
		},
		Event: application.EventRecord{
			Ref: "event:v06-wrong-accepted", Kind: "execution.dispatching", GoalRef: preparedGoal.Ref(),
			WorkItemRef: item.Ref(), ExecutionRef: runningExecution.Ref, OccurredAt: state.Goal.CreatedAt(),
		},
	}
	if err := validateLaunchAccepted(wrongAccepted); err == nil {
		t.Fatal("LaunchAccepted accepted execution.dispatching event")
	}
	wrongQuarantine := application.ActionQuarantinedState{
		Claim: claim, ErrorCode: "test.quarantine", OperationAt: state.Goal.CreatedAt(),
		Event: application.EventRecord{
			Ref: "event:v06-wrong-quarantine", Kind: "execution.failed", GoalRef: claim.Action.GoalRef,
			WorkItemRef: claim.Action.WorkItemRef, ExecutionRef: claim.Action.ExecutionRef,
			OccurredAt: state.Goal.CreatedAt(),
		},
	}
	if err := validateQuarantined(wrongQuarantine); err == nil {
		t.Fatal("Quarantine accepted execution.failed event")
	}

	repository, _ := openTestRepository(t)
	replacement, _ := buildReplacementState(t, repository, "wrong-event-kind")
	replacement.Events[1].Kind = "execution.queued"
	if err := validateExecutionReplaced(replacement); err == nil {
		t.Fatal("execution replacement accepted execution.queued event")
	}

	scheduled := state.Executions[0]
	wrongScheduled := application.EventRecord{
		Ref: "event:v06-wrong-scheduled", Kind: "execution.failed", GoalRef: state.Goal.Ref(),
		WorkItemRef: scheduled.WorkItemRef, ExecutionRef: scheduled.Ref, OccurredAt: state.Goal.CreatedAt(),
	}
	if err := validateExactMutationEvents(
		[]application.EventRecord{wrongScheduled}, state.Goal,
		[]eventSemantic{newEventSemantic("execution.queued", scheduled.WorkItemRef, scheduled.Ref)}, false,
	); err == nil {
		t.Fatal("scheduled execution accepted non-queued event")
	}
}

func TestV06RunningExecutionIdentityIsWriteOnceAndFencesTerminalMutation(t *testing.T) {
	repository, _ := openTestRepository(t)
	record, at := createRunningV06Fixture(t, repository, "identity-write-once")
	execution := record.Executions[0]
	if _, err := repository.db.Exec(`
UPDATE executions SET provider_ref = provider_ref, model_ref = model_ref,
    agent_ref = agent_ref, external_ref = external_ref
WHERE ref = ?`, execution.Ref.String()); err != nil {
		t.Fatalf("idempotent provider identity update: %v", err)
	}
	if _, err := repository.db.Exec(`UPDATE executions SET model_ref = 'model:tampered' WHERE ref = ?`, execution.Ref.String()); err == nil {
		t.Fatal("write-once provider identity was changed directly")
	}
	repository.now = func() time.Time { return at }
	claim := mustClaim(t, repository, "worker:v06-terminal", "claim:v06-terminal", at)
	item := onlyItem(t, record.Goal)
	finishedAt := at.Add(time.Second)
	succeededGoal, err := record.Goal.SucceedWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(),
		[]goal.ArtifactRef{mustRef(t, "artifact:v06-identity", goal.NewArtifactRef)},
		[]goal.AttestationRef{mustRef(t, "attestation:v06-identity", goal.NewAttestationRef)},
		finishedAt,
	)
	if err == nil {
		succeededGoal, err = succeededGoal.Close(succeededGoal.Revision(), goal.GoalOutcomeSucceeded, finishedAt)
	}
	if err != nil {
		t.Fatalf("close identity fixture in domain: %v", err)
	}
	tampered := execution
	tampered.State = application.ExecutionSucceeded
	tampered.ModelRef = "model:tampered"
	tampered.FinishedAt = finishedAt
	artifact := application.ArtifactRecord{
		Stored: ports.StoredArtifact{
			Ref: mustRef(t, "artifact:v06-identity", goal.NewArtifactRef), Digest: "sha256:v06-identity",
			MediaType: "text/plain", Size: 1,
		},
		GoalRef: succeededGoal.Ref(), WorkItemRef: item.Ref(), CreatedAt: finishedAt,
	}
	attestation := application.AttestationRecord{
		Ref: mustRef(t, "attestation:v06-identity", goal.NewAttestationRef), GoalRef: succeededGoal.Ref(),
		WorkItemRef: item.Ref(), ExecutionRef: tampered.Ref, ArtifactRef: artifact.Stored.Ref,
		Policy: "test.identity", AcceptedAt: finishedAt,
	}
	err = repository.RecordGoalSucceeded(context.Background(), application.GoalSucceededState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		Goal: succeededGoal, Execution: tampered, Artifact: artifact, Attestation: attestation,
		Events: []application.EventRecord{
			{Ref: "event:v06-identity-work-succeeded", Kind: "work_item.succeeded", GoalRef: succeededGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: tampered.Ref, OccurredAt: finishedAt},
			{Ref: "event:v06-identity-goal-succeeded", Kind: "goal.succeeded", GoalRef: succeededGoal.Ref(), OccurredAt: finishedAt},
		},
		OperationAt: finishedAt,
	})
	if !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("tampered running-to-terminal identity mutation = %v", err)
	}
	after, err := repository.GetGoal(context.Background(), record.Goal.Ref())
	if err != nil || after.Executions[0].State != application.ExecutionRunning ||
		after.Executions[0].ModelRef != execution.ModelRef || len(after.ConsumptionReceipts) != 1 {
		t.Fatalf("identity fence rollback = %+v err=%v", after, err)
	}
}

func TestV06ReplacementCreatedAtCannotPrecedeFailedFinishInValidatorOrTrigger(t *testing.T) {
	repository, _ := openTestRepository(t)
	replacement, _ := buildReplacementState(t, repository, "created-at-order")
	replacement.ReplacementExecution.CreatedAt = replacement.FailedExecution.FinishedAt.Add(-time.Nanosecond)
	if err := repository.RecordExecutionReplaced(context.Background(), replacement); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("replacement before failed finish = %v", err)
	}

	transaction, err := repository.db.Begin()
	if err != nil {
		t.Fatalf("begin replacement trigger test: %v", err)
	}
	defer transaction.Rollback()
	failed := replacement.FailedExecution
	if _, err := transaction.Exec(`
UPDATE executions SET state = 'failed', finished_at = ?, failure_code = ? WHERE ref = ?`,
		requiredTime(failed.FinishedAt), failed.FailureCode, failed.Ref.String(),
	); err != nil {
		t.Fatalf("seed failed execution for trigger: %v", err)
	}
	badRef := "execution:v06-trigger-created-at"
	_, err = transaction.Exec(`
INSERT INTO executions(
    ref, goal_ref, work_item_ref, attempt_no, max_execution_attempts,
    replaces_execution_ref, plan_generation, app_spec_generation, spec_hash,
    state, artifact_media_type, idempotency_key, max_output_bytes,
    provider_ref, model_ref, agent_ref, external_ref, created_at,
    deadline_at, started_at, provider_accepted_at, last_observed_at,
    provider_observed_at, finished_at, failure_code
)
SELECT ?, goal_ref, work_item_ref, attempt_no + 1, max_execution_attempts,
       ref, plan_generation, app_spec_generation, spec_hash,
       'dispatching', artifact_media_type, ?, max_output_bytes,
       '', '', '', '', ?, NULL, NULL, NULL, NULL, NULL, NULL, ''
FROM executions WHERE ref = ?`,
		badRef, "execution:"+badRef, requiredTime(replacement.ReplacementExecution.CreatedAt), failed.Ref.String(),
	)
	if err == nil {
		t.Fatal("replacement timestamp trigger accepted creation before failed finish")
	}
}

func buildReplacementState(
	t *testing.T,
	repository *Repository,
	suffix string,
) (application.ExecutionReplacedState, application.ActionClaim) {
	t.Helper()
	state := newCreateFixture(t, "v06-"+suffix, "request:v06-"+suffix, "fingerprint:v06-"+suffix, "actor:v06", "project:v06")
	if _, _, err := createLegacyGoal(t, repository, state); err != nil {
		t.Fatalf("create replacement fixture: %v", err)
	}
	claim := mustClaim(t, repository, "worker:v06-"+suffix, "claim:v06-"+suffix, state.Executions[0].CreatedAt)
	record, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("read replacement fixture: %v", err)
	}
	item := onlyItem(t, record.Goal)
	preparedAt := state.Executions[0].CreatedAt.Add(time.Second)
	preparedGoal, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), record.Executions[0].Ref, preparedAt,
	)
	if err != nil {
		t.Fatalf("prepare replacement Goal: %v", err)
	}
	preparedExecution := record.Executions[0]
	preparedExecution.State = application.ExecutionDispatching
	if err := repository.RecordLaunchPrepared(context.Background(), application.LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: preparedGoal,
		Execution: preparedExecution, OperationAt: preparedAt,
		Event: application.EventRecord{
			Ref: "event:v06-prepared:" + suffix, Kind: "execution.dispatching", GoalRef: preparedGoal.Ref(),
			WorkItemRef: item.Ref(), ExecutionRef: preparedExecution.Ref, OccurredAt: preparedAt,
		},
	}); err != nil {
		t.Fatalf("record prepared replacement fixture: %v", err)
	}
	current, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("read prepared replacement fixture: %v", err)
	}
	currentItem := onlyItem(t, current.Goal)
	replacedAt := preparedAt.Add(time.Second)
	replacementRef := mustRef(t, "execution:v06-replacement:"+suffix, goal.NewExecutionRef)
	replacedGoal, err := current.Goal.ReplaceWorkItemExecution(
		current.Goal.Revision(), currentItem.Revision(), currentItem.Ref(),
		current.Executions[0].Ref, replacementRef, replacedAt,
	)
	if err != nil {
		t.Fatalf("replace WorkItem execution: %v", err)
	}
	failed := current.Executions[0]
	failed.State = application.ExecutionFailed
	failed.FailureCode = "agent.launch_failed"
	failed.FinishedAt = replacedAt
	replacement := current.Executions[0]
	replacement.Ref = replacementRef
	replacement.AttemptNo++
	replacement.ReplacesExecutionRef = failed.Ref
	replacement.State = application.ExecutionDispatching
	replacement.IdempotencyKey = "execution:" + replacementRef.String()
	replacement.CreatedAt = replacedAt
	replacement.FinishedAt = time.Time{}
	replacement.FailureCode = ""
	updatedItem := onlyItem(t, replacedGoal)
	next := application.ActionRecord{
		Ref: "action:launch:" + replacementRef.String(), Kind: application.ActionLaunchAgent,
		GoalRef: replacedGoal.Ref(), WorkItemRef: updatedItem.Ref(), ExecutionRef: replacementRef,
		PlanGeneration: replacedGoal.PlanGeneration(), WorkItemGeneration: updatedItem.Revision(), AvailableAt: replacedAt,
	}
	return application.ExecutionReplacedState{
		Claim: claim, ExpectedGoalRevision: current.Goal.Revision(), ExpectedItemRevision: currentItem.Revision(),
		Goal: replacedGoal, FailedExecution: failed, ReplacementExecution: replacement,
		NextAction: next, ErrorCode: failed.FailureCode, OperationAt: replacedAt,
		Events: []application.EventRecord{
			{Ref: "event:v06-failed:" + suffix, Kind: "execution.failed", GoalRef: replacedGoal.Ref(), WorkItemRef: updatedItem.Ref(), ExecutionRef: failed.Ref, OccurredAt: replacedAt},
			{Ref: "event:v06-dispatching:" + suffix, Kind: "execution.dispatching", GoalRef: replacedGoal.Ref(), WorkItemRef: updatedItem.Ref(), ExecutionRef: replacement.Ref, OccurredAt: replacedAt},
		},
	}, claim
}

func createRunningV06Fixture(
	t *testing.T,
	repository *Repository,
	suffix string,
) (application.GoalRecord, time.Time) {
	t.Helper()
	state := newCreateFixture(t, "v06-"+suffix, "request:v06-"+suffix, "fingerprint:v06-"+suffix, "actor:v06", "project:v06")
	if _, _, err := createLegacyGoal(t, repository, state); err != nil {
		t.Fatalf("create running V6 fixture: %v", err)
	}
	claim := mustClaim(t, repository, "worker:v06-"+suffix, "claim:v06-"+suffix, state.Goal.CreatedAt())
	record, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("read queued V6 fixture: %v", err)
	}
	item := onlyItem(t, record.Goal)
	at := state.Goal.CreatedAt().Add(time.Second)
	preparedGoal, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), record.Executions[0].Ref, at,
	)
	if err != nil {
		t.Fatalf("start running V6 fixture: %v", err)
	}
	execution := record.Executions[0]
	execution.State = application.ExecutionDispatching
	if err := repository.RecordLaunchPrepared(context.Background(), application.LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: preparedGoal,
		Execution: execution, OperationAt: at,
		Event: application.EventRecord{
			Ref: "event:v06-running-dispatching:" + suffix, Kind: "execution.dispatching", GoalRef: preparedGoal.Ref(),
			WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at,
		},
	}); err != nil {
		t.Fatalf("persist running V6 preparation: %v", err)
	}
	execution.State = application.ExecutionRunning
	execution.ProviderRef = "provider:codex"
	execution.ModelRef = "model:codex"
	execution.AgentRef = "agent:codex"
	execution.ExternalRef = "external:v06:" + suffix
	execution.StartedAt = at
	execution.DeadlineAt = at.Add(time.Hour)
	execution.ProviderAcceptedAt = at
	preparedItem, _ := preparedGoal.WorkItem(item.Ref())
	if err := repository.RecordLaunchAccepted(context.Background(), application.LaunchAcceptedState{
		Claim: claim, Execution: execution, OperationAt: at,
		NextAction: application.ActionRecord{
			Ref: "action:v06-running-observe:" + suffix, Kind: application.ActionObserveAgent,
			GoalRef: preparedGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
			PlanGeneration: preparedGoal.PlanGeneration(), WorkItemGeneration: preparedItem.Revision(), AvailableAt: at,
		},
		Event: application.EventRecord{
			Ref: "event:v06-running-accepted:" + suffix, Kind: "execution.accepted", GoalRef: preparedGoal.Ref(),
			WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at,
		},
	}); err != nil {
		t.Fatalf("persist running V6 acceptance: %v", err)
	}
	running, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("read running V6 fixture: %v", err)
	}
	return running, at
}

func TestV06MigrationPreservesExecutionForeignKeysAfterRemovingWorkItemUnique(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v06-fk", "orquesta.sqlite")
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if err != nil {
		t.Fatalf("open V06 FK repository: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	var violations int
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&violations); err != nil || violations != 0 {
		t.Fatalf("V06 foreign key check = %d err=%v", violations, err)
	}
}

func TestV06MigrationV4RunningObserveClaimsByProviderWithDoubleSentinel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v4-running-observe", "orquesta.sqlite")
	base := seedV4Database(t, path, "running_observe")
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: func() time.Time { return base },
	})
	if err != nil {
		rootCause := err
		for errors.Unwrap(rootCause) != nil {
			rootCause = errors.Unwrap(rootCause)
		}
		t.Fatalf("migrate V4 running observation: %v root_cause=%v", err, rootCause)
	}
	t.Cleanup(func() { _ = repository.Close() })
	record, err := repository.GetGoal(context.Background(), mustRef(t, "goal:v4-state", goal.NewGoalRef))
	if err != nil || len(record.Executions) != 1 {
		t.Fatalf("read migrated V4 running observation: %+v err=%v", record, err)
	}
	execution := record.Executions[0]
	if execution.State != application.ExecutionRunning ||
		execution.ProviderRef != "provider:v4" ||
		execution.ModelRef != legacyV4ModelUnattributed ||
		execution.AgentRef != legacyV4AgentUnattributed {
		t.Fatalf("migrated V4 identity = %+v", execution)
	}
	var removedLegacyColumn int
	if err := repository.db.QueryRow(`
SELECT COUNT(*) FROM pragma_table_info('executions') WHERE name = 'max_attempts'`).Scan(&removedLegacyColumn); err != nil {
		t.Fatalf("inspect V5 execution schema: %v", err)
	}
	if removedLegacyColumn != 0 {
		t.Fatalf("legacy executions.max_attempts survived V5: %d", removedLegacyColumn)
	}
	if _, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v4-wrong-provider", Token: "claim:v4-wrong-provider", LeaseDuration: time.Minute,
		Capabilities: ports.AgentCapabilities{
			ProviderRef: "provider:other", ModelRef: "model:current", AgentRef: "agent:current", Unrestricted: true,
		},
	}); err != nil || found {
		t.Fatalf("wrong provider claimed legacy observation = found:%v err:%v", found, err)
	}
	claim, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v4-provider", Token: "claim:v4-provider", LeaseDuration: time.Minute,
		Capabilities: ports.AgentCapabilities{
			ProviderRef: "provider:v4", ModelRef: "model:current", AgentRef: "agent:current", Unrestricted: true,
		},
	})
	if err != nil || !found || claim.Action.Kind != application.ActionObserveAgent ||
		claim.Action.ExecutionRef != execution.Ref {
		t.Fatalf("provider-compatible legacy observation claim = %+v found:%v err:%v", claim, found, err)
	}
}

func TestV06ObserveClaimRequiresExactCurrentProviderModelAndAgent(t *testing.T) {
	repository, _ := openTestRepository(t)
	state := newCreateFixture(t, "v06-exact-observe", "request:v06-exact-observe", "fingerprint:v06-exact-observe", "actor:v06", "project:v06")
	if _, _, err := createLegacyGoal(t, repository, state); err != nil {
		t.Fatalf("create V6 exact identity fixture: %v", err)
	}
	claim := mustClaim(t, repository, "worker:v06-launch", "claim:v06-launch", state.Goal.CreatedAt())
	record, err := repository.GetGoal(context.Background(), state.Goal.Ref())
	if err != nil {
		t.Fatalf("read V6 exact identity fixture: %v", err)
	}
	item := onlyItem(t, record.Goal)
	at := state.Goal.CreatedAt().Add(time.Second)
	preparedGoal, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), record.Executions[0].Ref, at,
	)
	if err != nil {
		t.Fatalf("start V6 exact identity WorkItem: %v", err)
	}
	execution := record.Executions[0]
	execution.State = application.ExecutionDispatching
	if err := repository.RecordLaunchPrepared(context.Background(), application.LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: preparedGoal,
		Execution: execution, OperationAt: at,
		Event: application.EventRecord{
			Ref: "event:v06-exact-dispatching", Kind: "execution.dispatching", GoalRef: preparedGoal.Ref(),
			WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at,
		},
	}); err != nil {
		t.Fatalf("prepare V6 exact identity launch: %v", err)
	}
	execution.State = application.ExecutionRunning
	execution.ProviderRef = "provider:v06-exact"
	execution.ModelRef = "model:v06-exact"
	execution.AgentRef = "agent:v06-exact"
	execution.ExternalRef = "external:v06-exact"
	execution.StartedAt = at
	execution.DeadlineAt = at.Add(time.Hour)
	execution.ProviderAcceptedAt = at
	preparedItem, _ := preparedGoal.WorkItem(item.Ref())
	if err := repository.RecordLaunchAccepted(context.Background(), application.LaunchAcceptedState{
		Claim: claim, Execution: execution, OperationAt: at,
		NextAction: application.ActionRecord{
			Ref: "action:v06-exact-observe", Kind: application.ActionObserveAgent,
			GoalRef: preparedGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
			PlanGeneration: preparedGoal.PlanGeneration(), WorkItemGeneration: preparedItem.Revision(), AvailableAt: at,
		},
		Event: application.EventRecord{
			Ref: "event:v06-exact-accepted", Kind: "execution.accepted", GoalRef: preparedGoal.Ref(),
			WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at,
		},
	}); err != nil {
		t.Fatalf("accept V6 exact identity launch: %v", err)
	}
	repository.now = func() time.Time { return at }
	if _, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v06-wrong-model", Token: "claim:v06-wrong-model", LeaseDuration: time.Minute,
		Capabilities: ports.AgentCapabilities{
			ProviderRef: execution.ProviderRef, ModelRef: "model:other", AgentRef: execution.AgentRef, Unrestricted: true,
		},
	}); err != nil || found {
		t.Fatalf("non-exact V6 identity claimed observation = found:%v err:%v", found, err)
	}
	claimed, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:v06-exact", Token: "claim:v06-exact", LeaseDuration: time.Minute,
		Capabilities: ports.AgentCapabilities{
			ProviderRef: execution.ProviderRef, ModelRef: execution.ModelRef, AgentRef: execution.AgentRef, Unrestricted: true,
		},
	})
	if err != nil || !found || claimed.Action.ExecutionRef != execution.Ref {
		t.Fatalf("exact V6 identity observation claim = %+v found:%v err:%v", claimed, found, err)
	}
}

func TestV06MigrationCompletedActionWithoutClaimRollsBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v4-completed-without-claim", "orquesta.sqlite")
	seedV4Database(t, path, "completed_without_claim")
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if repository != nil {
		_ = repository.Close()
		t.Fatal("V4 completed action without claim unexpectedly migrated")
	}
	var stateErr *application.StateError
	if !errors.As(err, &stateErr) || stateErr.Code != application.StateInvalid || stateErr.Cause == nil ||
		!strings.Contains(stateErr.Cause.Error(), "sqlite.legacy_completed_action_claim_missing") {
		t.Fatalf("V4 incomplete consumption migration error = %v cause=%v", err, stateErr)
	}
	database, openErr := sql.Open(driverName, buildDSN(path, testBusyTimeout.Milliseconds()))
	if openErr != nil {
		t.Fatalf("inspect rolled-back V4 database: %v", openErr)
	}
	defer database.Close()
	var version, receipt, staging, completedWithoutClaim int
	if err := database.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("rolled-back V4 user_version: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = 5").Scan(&receipt); err != nil {
		t.Fatalf("rolled-back V5 migration receipt: %v", err)
	}
	if err := database.QueryRow(`
SELECT COUNT(*) FROM sqlite_schema
WHERE type = 'table' AND name IN ('executions_v5', 'action_consumption_receipts')`).Scan(&staging); err != nil {
		t.Fatalf("rolled-back V5 schema: %v", err)
	}
	if err := database.QueryRow(`
SELECT COUNT(*) FROM outbox WHERE completed_at IS NOT NULL AND claim_token IS NULL`).Scan(&completedWithoutClaim); err != nil {
		t.Fatalf("rolled-back incomplete V4 action: %v", err)
	}
	if version != 4 || receipt != 0 || staging != 0 || completedWithoutClaim != 1 {
		t.Fatalf("partial V5 migration escaped: version=%d receipt=%d staging=%d incomplete=%d",
			version, receipt, staging, completedWithoutClaim)
	}
}

func TestV06MigrationFailureAfterStagingDDLRestoresEntireV4Schema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v4-staging-failure", "orquesta.sqlite")
	seedV4Database(t, path, "staging_failure")
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if repository != nil {
		_ = repository.Close()
		t.Fatal("V4 staging failpoint unexpectedly migrated")
	}
	if err == nil {
		t.Fatal("V4 staging failpoint returned no error")
	}
	database, openErr := sql.Open(driverName, buildDSN(path, testBusyTimeout.Milliseconds()))
	if openErr != nil {
		t.Fatalf("inspect V4 staging rollback: %v", openErr)
	}
	defer database.Close()
	var version, receipt, staging, legacyColumns, originalRows int
	if err := database.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("staging rollback user_version: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = 5").Scan(&receipt); err != nil {
		t.Fatalf("staging rollback migration receipt: %v", err)
	}
	if err := database.QueryRow(`
SELECT COUNT(*) FROM sqlite_schema
WHERE type = 'table' AND name IN ('executions_v5', 'attestations_v5', 'events_v5', 'outbox_v5', 'action_consumption_receipts')`).Scan(&staging); err != nil {
		t.Fatalf("staging rollback tables: %v", err)
	}
	if err := database.QueryRow(`
SELECT COUNT(*) FROM pragma_table_info('executions') WHERE name = 'max_attempts'`).Scan(&legacyColumns); err != nil {
		t.Fatalf("staging rollback V4 schema: %v", err)
	}
	if err := database.QueryRow(`
SELECT COUNT(*)
FROM executions e
JOIN outbox o ON o.execution_ref = e.ref
WHERE e.ref = 'execution:v4-state' AND e.state = 'failed'
  AND o.ref = 'action:v4-state' AND o.kind = 'observe_agent'`).Scan(&originalRows); err != nil {
		t.Fatalf("staging rollback V4 rows: %v", err)
	}
	if version != 4 || receipt != 0 || staging != 0 || legacyColumns != 1 || originalRows != 1 {
		t.Fatalf("staging rollback incomplete: version=%d receipt=%d staging=%d legacy=%d rows=%d",
			version, receipt, staging, legacyColumns, originalRows)
	}
}

func seedV4Database(t *testing.T, path, mode string) time.Time {
	t.Helper()
	if err := preparePrivateDatabase(path); err != nil {
		t.Fatalf("prepare V4 path: %v", err)
	}
	database, err := sql.Open(driverName, buildDSN(path, testBusyTimeout.Milliseconds()))
	if err != nil {
		t.Fatalf("open V4 seed database: %v", err)
	}
	defer database.Close()
	database.SetMaxOpenConns(1)
	connection, err := database.Conn(context.Background())
	if err != nil {
		t.Fatalf("open V4 seed connection: %v", err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(context.Background(), "PRAGMA foreign_keys = OFF"); err != nil {
		t.Fatalf("disable V4 seed foreign keys: %v", err)
	}
	transaction, err := connection.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin V4 seed: %v", err)
	}
	defer transaction.Rollback()
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("load V4 migrations: %v", err)
	}
	base := time.Date(2026, 7, 14, 9, 0, 0, 0, time.UTC)
	for index := 0; index < 4; index++ {
		migration := migrations[index]
		if _, err := transaction.Exec(migration.preSQL); err != nil {
			t.Fatalf("apply V%d seed schema: %v", migration.version, err)
		}
		if index == 0 {
			seedV1Goal(t, transaction, "v4-state", "running", 3, "pending", 1, "queued", base, false)
		}
		if migration.backfill {
			if err := backfillAppSpecs(context.Background(), transaction); err != nil {
				t.Fatalf("backfill V%d seed: %v", migration.version, err)
			}
		}
		if strings.TrimSpace(migration.postSQL) != "" {
			if _, err := transaction.Exec(migration.postSQL); err != nil {
				t.Fatalf("finish V%d seed schema: %v", migration.version, err)
			}
		}
		if _, err := transaction.Exec(
			"INSERT INTO schema_migrations(version, name, checksum) VALUES (?, ?, ?)",
			migration.version, migration.name, migration.checksum,
		); err != nil {
			t.Fatalf("record V%d seed migration: %v", migration.version, err)
		}
	}
	if _, err := transaction.Exec("PRAGMA user_version = 4"); err != nil {
		t.Fatalf("set V4 seed user_version: %v", err)
	}
	switch mode {
	case "running_observe":
		if _, err := transaction.Exec(`UPDATE goals SET revision = 4 WHERE ref = 'goal:v4-state'`); err != nil {
			t.Fatalf("seed V4 running Goal: %v", err)
		}
		if _, err := transaction.Exec(`
UPDATE work_items
SET state = 'running', revision = 2, started_at = ?, execution_ref = 'execution:v4-state'
WHERE goal_ref = 'goal:v4-state' AND ref = 'work-item:v4-state'`, requiredTime(base)); err != nil {
			t.Fatalf("seed V4 running WorkItem: %v", err)
		}
		if _, err := transaction.Exec(`
UPDATE executions
SET state = 'running', provider_ref = 'provider:v4', external_ref = 'external:v4',
    started_at = ?, deadline_at = ?, provider_accepted_at = ?
WHERE goal_ref = 'goal:v4-state' AND ref = 'execution:v4-state'`,
			requiredTime(base), requiredTime(base.Add(time.Hour)), requiredTime(base),
		); err != nil {
			t.Fatalf("seed V4 running observation: %v", err)
		}
		if _, err := transaction.Exec(`
UPDATE outbox
SET ref = 'action:observe:v4-state', kind = 'observe_agent'
WHERE goal_ref = 'goal:v4-state' AND ref = 'action:v4-state'`); err != nil {
			t.Fatalf("seed V4 observation action: %v", err)
		}
	case "completed_without_claim":
		if _, err := transaction.Exec(`
UPDATE outbox SET completed_at = available_at
WHERE goal_ref = 'goal:v4-state' AND ref = 'action:v4-state'`); err != nil {
			t.Fatalf("seed V4 completed action without claim: %v", err)
		}
	case "staging_failure":
		if _, err := transaction.Exec(`
UPDATE work_items
SET state = 'failed', revision = 1, started_at = ?, finished_at = ?, execution_ref = 'execution:v4-state'
WHERE goal_ref = 'goal:v4-state' AND ref = 'work-item:v4-state'`, requiredTime(base), requiredTime(base)); err != nil {
			t.Fatalf("seed V4 staging-failure WorkItem: %v", err)
		}
		if _, err := transaction.Exec(`
UPDATE executions
SET state = 'failed', deadline_at = NULL, started_at = NULL,
    provider_ref = '', external_ref = '', provider_accepted_at = NULL,
    last_observed_at = NULL, provider_observed_at = NULL,
    finished_at = ?, failure_code = 'test.v4_staging_failure'
WHERE goal_ref = 'goal:v4-state' AND ref = 'execution:v4-state'`, requiredTime(base)); err != nil {
			t.Fatalf("seed V4 staging-failure execution: %v", err)
		}
		if _, err := transaction.Exec(`
UPDATE outbox SET kind = 'observe_agent'
WHERE goal_ref = 'goal:v4-state' AND ref = 'action:v4-state'`); err != nil {
			t.Fatalf("seed V4 staging-failure action: %v", err)
		}
	default:
		t.Fatalf("unknown V4 seed mode %q", mode)
	}
	if err := transaction.Commit(); err != nil {
		t.Fatalf("commit V4 seed: %v", err)
	}
	if _, err := connection.ExecContext(context.Background(), "PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("restore V4 seed foreign keys: %v", err)
	}
	return base
}

func TestRepositoryRejectsIncoherentProjectionAfterRestart(t *testing.T) {
	repository, path := openTestRepository(t)
	state := newCreateFixture(t, "v06-incoherent", "request:v06-incoherent", "fingerprint:v06-incoherent", "actor:v06", "project:v06")
	if _, _, err := createLegacyGoal(t, repository, state); err != nil {
		t.Fatalf("create incoherent fixture: %v", err)
	}
	// SQL shape permits dispatching while the authoritative WorkItem is still
	// pending. Runtime reads must reject that projection instead of treating it
	// as lifecycle truth.
	if _, err := repository.db.Exec(
		`UPDATE executions SET state = 'dispatching' WHERE ref = ?`,
		state.Executions[0].Ref.String(),
	); err != nil {
		t.Fatalf("seed incoherent execution projection: %v", err)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("close incoherent repository: %v", err)
	}

	restarted, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	if err != nil {
		t.Fatalf("reopen incoherent repository: %v", err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	_, err = restarted.GetGoal(context.Background(), state.Goal.Ref())
	var stateErr *application.StateError
	if !errors.As(err, &stateErr) || stateErr.Code != application.StateInvalid || stateErr.Cause == nil ||
		!strings.Contains(stateErr.Cause.Error(), "sqlite.goal_record_execution_binding_invalid") {
		t.Fatalf("incoherent projection read = %v cause=%v, want state.invalid binding error", err, stateErr)
	}
}
