package sqlite

import (
	"context"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestSQLiteDirectorFailedAttestationReplanSurvivesPolicyRotationReadRestartAndRecovery(t *testing.T) {
	ctx := context.Background()
	attestor := &sqliteTestAttestor{verdict: ports.TestAttestationFailed}
	system, goalRef := seedSQLiteV23FailedAttestationForBudgetRotation(t, attestor)
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
	historicalPolicy := system.policy
	sqliteTestNoError(t, system.repository.Close())
	system.policy = rotatedSQLiteV15Policy(system.policy, system.clock.Now())
	system.policy.DefaultWorkItemDemand.DiskBytes = 64 << 20
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.orchestrator = newSQLiteV16OrchestratorWithStateAttestorSessionsAndMaxOutput(
		t, system, system.repository, attestor, nil, 64<<20,
	)

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
	assertSQLiteFailedAttestationReplanState(
		t, system.repository, goalRef, source.Ref(), author.Ref,
		source.BudgetDemand().Resources.DiskBytes, author.MaxOutputBytes, historicalPolicy.PolicyHash,
	)

	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.orchestrator = newSQLiteV16OrchestratorWithStateAttestorSessionsAndMaxOutput(
		t, system, system.repository, attestor, nil, 64<<20,
	)
	assertSQLiteFailedAttestationReplanState(
		t, system.repository, goalRef, source.Ref(), author.Ref,
		source.BudgetDemand().Resources.DiskBytes, author.MaxOutputBytes, historicalPolicy.PolicyHash,
	)
	if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err != nil {
		t.Fatalf("failed attestation recovery: %v", err)
	}
	replay, err := system.orchestrator.ProposeDirectorPlan(ctx, system.access, request)
	if err != nil || replay.Created || replay.Decision != result.Decision {
		t.Fatalf("failed attestation replay=%+v err=%v", replay, err)
	}
}

func seedSQLiteV23FailedAttestationForBudgetRotation(
	t *testing.T,
	attestor *sqliteTestAttestor,
) (*sqliteV15System, goal.GoalRef) {
	t.Helper()
	system := newSQLiteV15System(t, 4)
	system.orchestrator = newSQLiteV16OrchestratorWithStateAttestorSessionsAndMaxOutput(
		t, system, system.repository, attestor, nil, 1<<20,
	)
	result, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: "request:v23-rotated-test-attestor",
		Statement:  "persist exact test attestation evidence across a budget rotation",
		Confirm:    true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:v23-rotated-test-attestor", Key: "phase:v23-rotated-test-attestor",
				TemplateRef: "phase-template:v23-rotated-test-attestor",
			}},
			WorkItems: []application.WorkItemSpec{{
				Key: "writer", Objective: "produce a tested isolated change",
				Phase: "phase:v23-rotated-test-attestor", Role: "role:writer",
				WriteSet:       []string{"internal/v23-rotated-budget"},
				CouncilPolicy:  council.PolicyAuto,
				RequiredTests:  sqliteRequiredTestSpecs("required-test:v23-rotated-budget"),
				OutputContract: goal.OutputContractEvidenceBundle,
			}},
		},
	})
	sqliteTestNoError(t, err)
	processSQLiteV16Actions(t, system,
		application.ActionPrepareWorkspace,
		application.ActionLaunchAgent,
		application.ActionObserveAgent,
		application.ActionCommitChange,
	)
	return system, result.Record.Goal.Ref()
}

func assertSQLiteFailedAttestationReplanState(
	t *testing.T,
	repository *Repository,
	goalRef goal.GoalRef,
	sourceRef goal.WorkItemRef,
	authorRef goal.ExecutionRef,
	diskBytes int64,
	maxOutputBytes int64,
	policyHash string,
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
	successorExecution, successorExecutionFound := sqliteExecutionForWorkItem(record.Executions, successor.Ref())
	successorIntent, successorIntentFound := sqliteEffectIntentForExecution(record.EffectIntents, successorExecution.Ref)
	reworkOf, linked := successor.ReworkOf()
	if !sourceFound || source.State() != goal.WorkItemStateSuperseded ||
		!authorFound || author.State != application.ExecutionFailed ||
		author.FailureCode != "test_attestor.required_tests_failed" ||
		!linked || reworkOf != source.Ref() || len(successor.RequiredTests()) != 1 ||
		successor.BudgetDemand().Resources.DiskBytes != diskBytes ||
		!successorExecutionFound || successorExecution.MaxOutputBytes != maxOutputBytes ||
		!successorIntentFound || successorIntent.PolicyHash != policyHash {
		t.Fatalf("failed attestation source=%+v author=%+v successor=%+v execution=%+v intent=%+v",
			source, author, successor, successorExecution, successorIntent)
	}
}

func sqliteExecutionForWorkItem(
	records []application.ExecutionRecord,
	ref goal.WorkItemRef,
) (application.ExecutionRecord, bool) {
	for _, record := range records {
		if record.WorkItemRef == ref {
			return record, true
		}
	}
	return application.ExecutionRecord{}, false
}

func sqliteEffectIntentForExecution(
	records []application.EffectIntent,
	ref goal.ExecutionRef,
) (application.EffectIntent, bool) {
	for _, record := range records {
		if record.Subject.ExecutionRef == ref {
			return record, true
		}
	}
	return application.EffectIntent{}, false
}
