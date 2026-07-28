package sqlite

import (
	"context"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestSQLiteDirectorFailedAttestationReplanSurvivesImmediateReadRestartAndRecovery(t *testing.T) {
	ctx := context.Background()
	attestor := &sqliteTestAttestor{verdict: ports.TestAttestationFailed}
	system, goalRef := seedSQLiteV17Committed(t, attestor)
	processSQLiteV16Actions(t, system, application.ActionAttestTest)

	before, err := system.repository.GetGoal(ctx, goalRef)
	sqliteTestNoError(t, err)
	source := before.Goal.WorkItems()[0]
	authorRef, bound := source.Execution()
	author, authorFound := sqliteExecutionByRef(before.Executions, authorRef)
	interruptCause, interrupted := source.InterruptCause()
	if !bound || !authorFound || !interrupted ||
		source.State() != goal.WorkItemStateInterrupted ||
		interruptCause != goal.WorkItemInterruptExecutionFailed ||
		author.State != application.ExecutionFailed ||
		author.FailureCode != "test_attestor.required_tests_failed" {
		t.Fatalf("failed attestation source=%+v author=%+v", source, author)
	}

	lease, err := system.orchestrator.ClaimDirector(ctx, system.access, application.ClaimDirectorRequest{
		RequestRef: "director-claim:v23-failed-attestation", GoalRef: goalRef,
	})
	sqliteTestNoError(t, err)
	request := application.ProposeDirectorPlanRequest{
		RequestRef: "director-plan:v23-failed-attestation", GoalRef: goalRef,
		ExpectedGoalRevision: before.Goal.Revision(), ExpectedPlanGeneration: before.Goal.PlanGeneration(),
		LeaseToken: lease.Lease.Token, LeaseFence: lease.Lease.Fence,
		Cause: goal.ReplanCauseExecutionFailed, SourceWorkItemRef: source.Ref(),
		ExpectedWorkItemRevision: source.Revision(), SourceExecutionRef: author.Ref,
		SourceExecutionAttempt: author.AttemptNo, Reason: "repair failed required tests",
		Plan: application.PlanSpec{WorkItems: []application.WorkItemSpec{{
			Key: "failed-attestation-successor", Objective: "repair failed required tests",
			Phase: source.Phase().String(), Role: source.Role().String(),
			WriteSet: []string{"internal/v23-failed-attestation-rework"}, CouncilPolicy: council.PolicyAuto,
			RequiredTests:  sqliteRequiredTestSpecs("required-test:v23-failed-attestation-rework"),
			OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	}
	result, err := system.orchestrator.ProposeDirectorPlan(ctx, system.access, request)
	if err != nil || !result.Created || result.Decision.Cause != goal.ReplanCauseExecutionFailed {
		t.Fatalf("failed attestation replan=%+v err=%s", result, sqliteTestErrorChain(err))
	}
	assertSQLiteFailedAttestationReplanState(t, system.repository, goalRef, source.Ref(), author.Ref)

	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.orchestrator = newSQLiteV16OrchestratorWithAttestor(t, system, attestor)
	assertSQLiteFailedAttestationReplanState(t, system.repository, goalRef, source.Ref(), author.Ref)
	if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err != nil {
		t.Fatalf("failed attestation recovery: %v", err)
	}
	replay, err := system.orchestrator.ProposeDirectorPlan(ctx, system.access, request)
	if err != nil || replay.Created || replay.Decision != result.Decision {
		t.Fatalf("failed attestation replay=%+v err=%v", replay, err)
	}
}

func assertSQLiteFailedAttestationReplanState(
	t *testing.T,
	repository *Repository,
	goalRef goal.GoalRef,
	sourceRef goal.WorkItemRef,
	authorRef goal.ExecutionRef,
) {
	t.Helper()
	record, err := repository.GetGoal(context.Background(), goalRef)
	if err != nil {
		t.Fatalf("failed attestation replan read: %s", sqliteTestErrorChain(err))
	}
	source, sourceFound := record.Goal.WorkItem(sourceRef)
	author, authorFound := sqliteExecutionByRef(record.Executions, authorRef)
	items := record.Goal.WorkItems()
	successor := items[len(items)-1]
	reworkOf, linked := successor.ReworkOf()
	if !sourceFound || source.State() != goal.WorkItemStateSuperseded ||
		!authorFound || author.State != application.ExecutionFailed ||
		author.FailureCode != "test_attestor.required_tests_failed" ||
		!linked || reworkOf != source.Ref() || len(successor.RequiredTests()) != 1 {
		t.Fatalf("failed attestation source=%+v author=%+v successor=%+v", source, author, successor)
	}
}
