package sqlite

import (
	"context"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/review"
)

type sqliteDirectorCouncilOneOfTamper struct {
	application.StateRepository
	called bool
	err    error
}

func (repository *sqliteDirectorCouncilOneOfTamper) ApplyDirectorPlan(
	ctx context.Context,
	state application.ApplyDirectorPlanState,
) (application.DirectorDecisionRecord, bool, error) {
	repository.called = true
	state.Decision.CouncilSubjectDigest = application.CouncilSubjectDigest("sha256:" + strings.Repeat("a", 64))
	state.Decision.CouncilDecisionRef = "council-decision:unexpected"
	state.Decision.CouncilDecisionDigest = application.CouncilSubjectDigest("sha256:" + strings.Repeat("b", 64))
	decision, created, err := repository.StateRepository.ApplyDirectorPlan(ctx, state)
	repository.err = err
	return decision, created, err
}

func TestSQLiteDirectorReviewChangesRequestedReplanRestart(t *testing.T) {
	ctx := context.Background()
	attestor := &sqliteTestAttestor{}
	system, goalRef := seedSQLiteV17Committed(t, attestor)
	system.external.mu.Lock()
	system.external.reviewVerdict = review.VerdictChangesRequested
	system.external.mu.Unlock()
	processSQLiteV16Actions(t, system,
		application.ActionAttestTest,
		application.ActionLaunchAgent, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionObserveAgent,
	)

	before, err := system.repository.GetGoal(ctx, goalRef)
	sqliteTestNoError(t, err)
	source := before.Goal.WorkItems()[0]
	authorRef, bound := source.Execution()
	author, authorFound := sqliteExecutionByRef(before.Executions, authorRef)
	interrupt, interrupted := source.InterruptCause()
	change := before.ChangeSets[0]
	if !bound || !authorFound || !interrupted ||
		source.State() != goal.WorkItemStateInterrupted ||
		interrupt != goal.WorkItemInterruptExecutionFailed ||
		author.State != application.ExecutionFailed ||
		author.FailureCode != string(goal.ReplanCauseReviewChangesRequested) ||
		!application.FailedReviewPreservesCandidate(before, source, author, change) {
		t.Fatalf("review changes source=%+v author=%+v", source, author)
	}

	lease, err := system.orchestrator.ClaimDirector(ctx, system.access, application.ClaimDirectorRequest{
		RequestRef: "director-claim:v18-review-changes", GoalRef: goalRef,
	})
	sqliteTestNoError(t, err)
	request := application.ProposeDirectorPlanRequest{
		RequestRef: "director-plan:v18-review-changes", GoalRef: goalRef,
		ExpectedGoalRevision: before.Goal.Revision(), ExpectedPlanGeneration: before.Goal.PlanGeneration(),
		LeaseToken: lease.Lease.Token, LeaseFence: lease.Lease.Fence,
		Cause: goal.ReplanCauseReviewChangesRequested, SourceWorkItemRef: source.Ref(),
		ExpectedWorkItemRevision: source.Revision(), SourceExecutionRef: author.Ref,
		SourceExecutionAttempt: author.AttemptNo, Reason: "repair exact review findings",
		Plan: application.PlanSpec{WorkItems: []application.WorkItemSpec{{
			Key: "review-successor", Objective: "repair exact review findings",
			Phase: source.Phase().String(), Role: source.Role().String(),
			WriteSet: []string{"internal/v18-review-rework"}, CouncilPolicy: council.PolicyAuto,
			RequiredTests:  sqliteRequiredTestSpecs("required-test:v18-review-rework"),
			OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	}

	tamperState := &sqliteDirectorCouncilOneOfTamper{StateRepository: system.repository}
	tampered := newSQLiteV16OrchestratorWithStateAndAttestor(t, system, tamperState, attestor)
	if _, err = tampered.ProposeDirectorPlan(ctx, system.access, request); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("review replan accepted unexpected Council fence: called=%v repository=%s result=%s",
			tamperState.called, sqliteTestErrorChain(tamperState.err), sqliteTestErrorChain(err))
	}
	var decisions int
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT COUNT(*) FROM director_decisions
WHERE goal_ref=?`, goalRef.String()).Scan(&decisions))
	if decisions != 0 {
		t.Fatalf("invalid Council one-of persisted %d Director decisions", decisions)
	}

	result, err := system.orchestrator.ProposeDirectorPlan(ctx, system.access, request)
	if err != nil || !result.Created || result.Decision.Cause != goal.ReplanCauseReviewChangesRequested {
		t.Fatalf("review changes replan=%+v err=%s", result, sqliteTestErrorChain(err))
	}
	assertSQLiteReviewChangesReplanState(t, system.repository, goalRef, source.Ref(), author.Ref)

	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.orchestrator = newSQLiteV16OrchestratorWithAttestor(t, system, attestor)
	assertSQLiteReviewChangesReplanState(t, system.repository, goalRef, source.Ref(), author.Ref)
	if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err != nil {
		t.Fatalf("review changes recovery: %v", err)
	}
	replay, err := system.orchestrator.ProposeDirectorPlan(ctx, system.access, request)
	if err != nil || replay.Created || replay.Decision != result.Decision {
		t.Fatalf("review changes replay=%+v err=%v", replay, err)
	}
}

func assertSQLiteReviewChangesReplanState(
	t *testing.T,
	repository *Repository,
	goalRef goal.GoalRef,
	sourceRef goal.WorkItemRef,
	authorRef goal.ExecutionRef,
) {
	t.Helper()
	record, err := repository.GetGoal(context.Background(), goalRef)
	if err != nil {
		t.Fatalf("review replan read: %s", sqliteTestErrorChain(err))
	}
	source, sourceFound := record.Goal.WorkItem(sourceRef)
	author, authorFound := sqliteExecutionByRef(record.Executions, authorRef)
	items := record.Goal.WorkItems()
	successor := items[len(items)-1]
	reworkOf, linked := successor.ReworkOf()
	if !sourceFound || source.State() != goal.WorkItemStateSuperseded || !authorFound ||
		author.State != application.ExecutionFailed ||
		author.FailureCode != string(goal.ReplanCauseReviewChangesRequested) ||
		!linked || reworkOf != source.Ref() || len(successor.RequiredTests()) != 1 {
		t.Fatalf("review replan source=%+v author=%+v successor=%+v", source, author, successor)
	}
	var cause, subject, decisionRef, decisionDigest string
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT cause,council_subject_digest,
COALESCE(council_decision_ref,''),COALESCE(council_decision_digest,'')
FROM director_decisions WHERE goal_ref=?`, goalRef.String()).
		Scan(&cause, &subject, &decisionRef, &decisionDigest))
	if cause != string(goal.ReplanCauseReviewChangesRequested) ||
		subject != "" || decisionRef != "" || decisionDigest != "" {
		t.Fatalf("review Director decision cause=%q Council=%q/%q/%q",
			cause, subject, decisionRef, decisionDigest)
	}
}
