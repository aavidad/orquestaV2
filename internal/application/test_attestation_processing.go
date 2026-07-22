package application

import (
	"context"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func (orchestrator *Orchestrator) processAttestTest(ctx context.Context, claim ActionClaim) error {
	record, err := orchestrator.state.GetGoal(ctx, claim.Action.GoalRef)
	if err != nil {
		return err
	}
	if priorEffectAttemptBlocksDispatch(record, claim.Action) {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	if orchestrator.testAttestor == nil {
		return orchestrator.requeueWorkspaceEffect(ctx, claim, "test_attestor.unavailable")
	}
	if err := validateClaimedRecord(claim, record, ActionAttestTest); err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	if err := validateClaimedEffect(claim, orchestrator.clock.Now()); err != nil {
		return orchestrator.quarantine(ctx, claim, err.Error())
	}
	item, _ := record.Goal.WorkItem(claim.Action.WorkItemRef)
	execution, _ := executionForAction(record, claim.Action)
	binding, bindingFound := workspaceBindingForExecution(record, execution.Ref)
	change, changeFound := changeSetByRef(record, claim.Action.ChangeRef)
	if !bindingFound || !changeFound {
		return orchestrator.quarantine(ctx, claim, "application.test_subject_missing")
	}
	subject, err := buildTestSubject(
		record.Goal, item, execution, binding, change, orchestrator.testAttestationPolicy,
	)
	if err != nil || ports.TestSubjectDigest(subject) != claim.Action.EffectIntent.TargetDigest {
		return orchestrator.quarantine(ctx, claim, "application.effect_target_mismatch")
	}
	request := ports.TestAttestationRequest{
		Subject: subject, SubjectDigest: ports.TestSubjectDigest(subject), RequiredTests: item.RequiredTests(),
		IdempotencyKey: claim.Action.EffectIntent.IdempotencyKey,
		RequestedAt:    claim.Action.EffectIntent.CreatedAt,
	}
	run := newTestAttestationRun(request, testSnapshotRequest(subject, change))
	if err := ValidateTestAttestationRun(run); err != nil {
		return orchestrator.quarantine(ctx, claim, ports.TestAttestorContractErrorCode(err))
	}
	if err := orchestrator.validateCurrentAutomaticAuthority(ctx, claim); err != nil {
		return orchestrator.requeueWorkspaceEffect(ctx, claim, err.Error())
	}
	attempt, err := orchestrator.beginNewEffectAttempt(ctx, claim, orchestrator.clock.Now())
	if err != nil {
		return err
	}
	effectCtx, cancel := orchestrator.actionCallContext(ctx, claim)
	result, err := orchestrator.testAttestor.Attest(effectCtx, run)
	cancel()
	if err != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	if err := ports.ValidateTestAttestationResult(request, result); err != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	return orchestrator.completeTestAttestation(
		ctx, claim, record, item, execution, binding, change, run, attempt, result,
	)
}

func (orchestrator *Orchestrator) completeTestAttestation(
	ctx context.Context,
	claim ActionClaim,
	record GoalRecord,
	item goal.WorkItem,
	execution ExecutionRecord,
	binding WorkspaceBinding,
	change ChangeSet,
	run TestAttestationRun,
	attempt EffectAttempt,
	result ports.TestAttestationResult,
) error {
	manifest, err := orchestrator.publishTestArtifact(ctx, result.Manifest)
	if err != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	report, err := orchestrator.publishTestArtifact(ctx, result.Report)
	if err != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	completedAt := orchestrator.clock.Now().UTC()
	if result.FinishedAt.After(completedAt) {
		completedAt = result.FinishedAt.UTC()
	}
	status := EffectStatusAttestedPassed
	if result.Verdict == ports.TestAttestationFailed {
		status = EffectStatusAttestedFailed
	}
	externalReceipt, err := effectReceipt(claim, attempt, result.ReceiptRef, status, unknownUsage(), completedAt)
	if err != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	manifestRecord := testArtifactOccurrence(
		ArtifactKindTestSubjectManifest, "manifest", manifest, record.Goal, item, execution, completedAt,
	)
	reportRecord := testArtifactOccurrence(
		ArtifactKindTestReport, "report", report, record.Goal, item, execution, completedAt,
	)
	attestation, err := requiredTestsAttestation(
		result, manifestRecord, reportRecord, record.Goal, item, execution, binding, change,
		claim, attempt, externalReceipt,
	)
	if err != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	transition, err := orchestrator.testAttestationTransition(
		ctx, record, item, execution, change, result.Verdict, completedAt,
	)
	if err != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	err = orchestrator.state.RecordTestAttested(ctx, TestAttestedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		Goal: transition.goal, Execution: transition.execution,
		ManifestArtifact: manifestRecord, ReportArtifact: reportRecord, Attestation: attestation,
		EffectReceipt: externalReceipt, NewExecutions: transition.newExecutions,
		NewActions: transition.newActions, Events: transition.events, OperationAt: completedAt,
	})
	if err != nil {
		return orchestrator.quarantineUnknownApplied(ctx, claim)
	}
	return nil
}

func newTestAttestationRun(
	request ports.TestAttestationRequest,
	snapshot ports.SnapshotVerificationRequest,
) TestAttestationRun {
	request.RequiredTests = append([]goal.RequiredTestSpec(nil), request.RequiredTests...)
	snapshot.ChangedPaths = append([]string(nil), snapshot.ChangedPaths...)
	snapshot.WriteSet = append([]string(nil), snapshot.WriteSet...)
	return TestAttestationRun{Request: request, Snapshot: snapshot}
}

func (orchestrator *Orchestrator) publishTestArtifact(
	ctx context.Context,
	request ports.PutArtifactRequest,
) (ports.StoredArtifact, error) {
	stored, err := orchestrator.artifacts.Put(ctx, request)
	if err != nil {
		return ports.StoredArtifact{}, err
	}
	if err := ports.ValidateStoredArtifact(request, stored); err != nil {
		return ports.StoredArtifact{}, err
	}
	content, err := orchestrator.artifacts.Get(ctx, stored.Ref, stored.Size)
	if err != nil {
		return ports.StoredArtifact{}, err
	}
	if err := ports.ValidateArtifactContent(content); err != nil || content.Ref != stored.Ref ||
		content.Digest != stored.Digest || content.Size != stored.Size {
		if err != nil {
			return ports.StoredArtifact{}, err
		}
		return ports.StoredArtifact{}, ports.NewArtifactContractError(ports.ArtifactErrorFileChanged, nil)
	}
	return stored, nil
}

func testArtifactOccurrence(
	kind ArtifactKind,
	suffix string,
	stored ports.StoredArtifact,
	aggregate goal.Goal,
	item goal.WorkItem,
	execution ExecutionRecord,
	at time.Time,
) ArtifactRecord {
	return ArtifactRecord{
		OccurrenceRef: "artifact-occurrence:test-" + suffix + ":" + execution.Ref.String(), Kind: kind,
		Stored: stored, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		ExecutionAttempt: execution.AttemptNo, PlanGeneration: execution.PlanGeneration,
		WorkItemGeneration: item.Revision(), AppSpecGeneration: execution.AppSpecGeneration,
		SpecHash: execution.SpecHash, CreatedAt: at.UTC(),
	}
}

func requiredTestsAttestation(
	result ports.TestAttestationResult,
	manifest ArtifactRecord,
	report ArtifactRecord,
	aggregate goal.Goal,
	item goal.WorkItem,
	execution ExecutionRecord,
	binding WorkspaceBinding,
	change ChangeSet,
	claim ActionClaim,
	attempt EffectAttempt,
	effect EffectReceipt,
) (AttestationRecord, error) {
	ref, err := goal.NewAttestationRef("attestation:required-tests:" + execution.Ref.String())
	if err != nil {
		return AttestationRecord{}, err
	}
	verdict := AttestationVerdictPassed
	if result.Verdict == ports.TestAttestationFailed {
		verdict = AttestationVerdictFailed
	}
	return AttestationRecord{
		Ref: ref, Kind: AttestationKindRequiredTests, Verdict: verdict,
		GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		ExecutionAttempt: execution.AttemptNo, PlanGeneration: execution.PlanGeneration,
		WorkItemGeneration: item.Revision(), AppSpecGeneration: execution.AppSpecGeneration,
		SpecHash: execution.SpecHash, ArtifactRef: report.Stored.Ref,
		SubjectDigest: result.SubjectDigest, WorkspaceBindingDigest: binding.Digest(),
		ChangeSetRef: change.Ref, ChangeSetDigest: change.Digest(),
		ManifestArtifactRef: manifest.Stored.Ref, ReportArtifactRef: report.Stored.Ref,
		AttestorRef: result.AttestorRef, ReceiptRef: result.ReceiptRef,
		Tests:     append([]ports.RequiredTestOutcome(nil), result.Tests...),
		PolicyRef: result.PolicyRef, RequiredTestsDigest: item.RequiredTestsDigest(),
		PolicyDigest: result.PolicyDigest, EffectIntentRef: claim.Action.EffectIntent.Ref,
		EffectAttemptRef: attempt.Ref, EffectFence: claim.Fence, EffectReceiptRef: effect.Ref,
		StartedAt: result.StartedAt.UTC(), FinishedAt: result.FinishedAt.UTC(),
		Policy: result.PolicyRef, AcceptedAt: result.FinishedAt.UTC(),
	}, nil
}

type testAttestationTransitionResult struct {
	goal          goal.Goal
	execution     ExecutionRecord
	newExecutions []ExecutionRecord
	newActions    []ActionRecord
	events        []EventRecord
}

func (orchestrator *Orchestrator) testAttestationTransition(
	ctx context.Context,
	record GoalRecord,
	item goal.WorkItem,
	execution ExecutionRecord,
	change ChangeSet,
	verdict ports.TestAttestationVerdict,
	at time.Time,
) (testAttestationTransitionResult, error) {
	if verdict == ports.TestAttestationPassed {
		execution.State = ExecutionAwaitingIntegration
		return testAttestationTransitionResult{
			goal: record.Goal, execution: execution,
			events: []EventRecord{{
				Ref: "event:tests-passed:" + change.Ref.String(), Kind: "test_attestation.passed",
				GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
				OccurredAt: at.UTC(),
			}},
		}, nil
	}
	transitionAt := lifecycleTime(at, record.Goal, item)
	aggregate, err := record.Goal.InterruptWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), execution.Ref,
		goal.WorkItemInterruptExecutionFailed, transitionAt,
	)
	if err != nil {
		return testAttestationTransitionResult{}, err
	}
	execution.State, execution.FinishedAt = ExecutionFailed, transitionAt
	execution.FailureCode = "test_attestor.required_tests_failed"
	existing := replaceExecution(record.Executions, execution)
	newExecutions, newActions, scheduledEvents, err := orchestrator.scheduleHistoricalReady(
		ctx, record, aggregate, existing, transitionAt,
	)
	if err != nil {
		return testAttestationTransitionResult{}, err
	}
	events := []EventRecord{{
		Ref: "event:tests-failed:" + change.Ref.String(), Kind: "test_attestation.failed",
		GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		OccurredAt: transitionAt,
	}}
	events = append(events, scheduledEvents...)
	return testAttestationTransitionResult{
		goal: aggregate, execution: execution, newExecutions: newExecutions,
		newActions: newActions, events: events,
	}, nil
}
