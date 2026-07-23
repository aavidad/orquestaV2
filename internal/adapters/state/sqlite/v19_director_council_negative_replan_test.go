package sqlite

import (
	"context"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
)

func TestSQLiteCanceledExecutionStateFields(t *testing.T) {
	now := time.Date(2026, 7, 23, 11, 0, 0, 0, time.UTC)
	unlaunched := application.ExecutionRecord{
		State: application.ExecutionCanceled, FailureCode: "application.execution_superseded",
		FinishedAt: now,
	}
	if err := validateExecutionStateFields(unlaunched); err != nil {
		t.Fatalf("unlaunched cancellation: %v", err)
	}
	accepted := unlaunched
	accepted.ProviderRef, accepted.ModelRef = "provider:test", "model:test"
	accepted.AgentRef, accepted.ExternalRef = "agent:test", "external:test"
	accepted.StartedAt, accepted.ProviderAcceptedAt = now.Add(-2*time.Minute), now.Add(-2*time.Minute)
	accepted.DeadlineAt = now.Add(time.Hour)
	accepted.LastObservedAt, accepted.ProviderObservedAt = now.Add(-time.Minute), now.Add(-time.Minute)
	if err := validateExecutionStateFields(accepted); err != nil {
		t.Fatalf("provider-accepted cancellation: %v", err)
	}
	mixed := accepted
	mixed.ModelRef = ""
	if err := validateExecutionStateFields(mixed); err == nil {
		t.Fatal("mixed provider cancellation fields accepted")
	}
	whitespaceProvider := accepted
	whitespaceProvider.ProviderRef = " "
	if err := validateExecutionStateFields(whitespaceProvider); err == nil {
		t.Fatal("whitespace cancellation provider accepted")
	}
	mixedObservation := accepted
	mixedObservation.ProviderObservedAt = time.Time{}
	if err := validateExecutionStateFields(mixedObservation); err == nil {
		t.Fatal("partial cancellation observation accepted")
	}
}

func TestSQLiteDirectorCouncilNegativeReplanRestart(t *testing.T) {
	tests := []struct {
		outcome council.Outcome
		ballots map[council.Role]council.Ballot
	}{
		{council.OutcomeRejected, map[council.Role]council.Ballot{
			council.RoleProposer: council.BallotReject,
			council.RoleCritic:   council.BallotReject,
			council.RoleArbiter:  council.BallotAccept,
		}},
		{council.OutcomeNoConsensus, map[council.Role]council.Ballot{
			council.RoleProposer: council.BallotAccept,
			council.RoleCritic:   council.BallotReject,
			council.RoleArbiter:  council.BallotAbstain,
		}},
		{council.OutcomeBlockedSecurity, map[council.Role]council.Ballot{
			council.RoleProposer: council.BallotAccept,
			council.RoleCritic:   council.BallotAccept,
			council.RoleArbiter:  council.BallotSecurityVeto,
		}},
	}
	for _, test := range tests {
		t.Run(string(test.outcome), func(t *testing.T) {
			system, goalRef := seedSQLiteV19Council(t, council.PolicyAuto, false)
			system.orchestrator = newSQLiteV19CouncilOrchestrator(t, system,
				&sqliteV19CouncilObserver{base: system.external, ballots: test.ballots})
			processSQLiteV16Actions(t, system,
				application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionLaunchAgent,
				application.ActionObserveAgent, application.ActionObserveAgent, application.ActionObserveAgent)

			record, err := system.repository.GetGoal(context.Background(), goalRef)
			sqliteTestNoError(t, err)
			source := record.Goal.WorkItems()[0]
			authorRef, bound := source.Execution()
			author, authorFound := sqliteExecutionByRef(record.Executions, authorRef)
			if !bound || !authorFound || author.State != application.ExecutionAwaitingIntegration ||
				len(record.CouncilDecisions) != 1 ||
				record.CouncilDecisions[0].Decision.Outcome != test.outcome {
				t.Fatalf("negative Council source=%+v author=%+v decisions=%+v",
					source, author, record.CouncilDecisions)
			}
			decision := record.CouncilDecisions[0]
			lease, err := system.orchestrator.ClaimDirector(context.Background(), system.access,
				application.ClaimDirectorRequest{
					RequestRef: "director-claim:v19-" + string(test.outcome), GoalRef: goalRef,
				})
			sqliteTestNoError(t, err)
			request := application.ProposeDirectorPlanRequest{
				RequestRef: "director-plan:v19-" + string(test.outcome), GoalRef: goalRef,
				ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
				LeaseToken: lease.Lease.Token, LeaseFence: lease.Lease.Fence,
				Cause: goal.ReplanCauseGovernanceDecision, SourceWorkItemRef: source.Ref(),
				ExpectedWorkItemRevision: source.Revision(), SourceExecutionRef: author.Ref,
				SourceExecutionAttempt: author.AttemptNo, CouncilSubjectDigest: decision.SubjectDigest,
				CouncilDecisionRef: decision.Ref, CouncilDecisionDigest: decision.DecisionDigest,
				Reason: "persist exact negative Council replan",
				Plan: application.PlanSpec{WorkItems: []application.WorkItemSpec{{
					Key: "successor-" + string(test.outcome), Objective: "repair negative Council result",
					Phase: source.Phase().String(), Role: source.Role().String(),
					WriteSet: []string{"internal/v19-rework"}, CouncilPolicy: council.PolicyRequired,
					RequiredTests:  sqliteRequiredTestSpecs("required-test:v19-rework-" + string(test.outcome)),
					OutputContract: goal.OutputContractEvidenceBundle,
				}}},
			}
			result, err := system.orchestrator.ProposeDirectorPlan(context.Background(), system.access, request)
			if err != nil || !result.Created ||
				result.Decision.CouncilSubjectDigest != decision.SubjectDigest ||
				result.Decision.CouncilDecisionRef != decision.Ref ||
				result.Decision.CouncilDecisionDigest != decision.DecisionDigest {
				t.Fatalf("negative Council replan=%+v err=%s", result, sqliteTestErrorChain(err))
			}
			assertSQLiteCouncilReplanState(t, system.repository, goalRef, source.Ref(), author.Ref, decision)

			sqliteTestNoError(t, system.repository.Close())
			system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
			system.orchestrator = newSQLiteV19CouncilOrchestrator(t, system,
				&sqliteV19CouncilObserver{base: system.external, ballots: test.ballots})
			assertSQLiteCouncilReplanState(t, system.repository, goalRef, source.Ref(), author.Ref, decision)
			if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
				t.Fatalf("negative Council recovery: %v", err)
			}
			replay, err := system.orchestrator.ProposeDirectorPlan(context.Background(), system.access, request)
			if err != nil || replay.Created || replay.Decision != result.Decision {
				t.Fatalf("negative Council restart replay=%+v err=%v", replay, err)
			}

			if test.outcome == council.OutcomeRejected {
				rewriteRecoveryTrigger(t, system.repository.db, "director_decisions_immutable_update", func() {
					mustV10Exec(t, system.repository.db, `UPDATE director_decisions
SET council_decision_digest='sha256:`+strings.Repeat("f", 64)+`'
WHERE ref=?`, result.Decision.Ref)
				})
				if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil {
					t.Fatal("tampered Council Director decision passed recovery")
				}
			}
		})
	}
}

func assertSQLiteCouncilReplanState(t *testing.T, repository *Repository, goalRef goal.GoalRef,
	sourceRef goal.WorkItemRef, authorRef goal.ExecutionRef, decision application.CouncilDecisionRecord,
) {
	t.Helper()
	record, err := repository.GetGoal(context.Background(), goalRef)
	if err != nil {
		t.Fatalf("Council replan read: %s", sqliteTestErrorChain(err))
	}
	source, sourceFound := record.Goal.WorkItem(sourceRef)
	author, authorFound := sqliteExecutionByRef(record.Executions, authorRef)
	successor := record.Goal.WorkItems()[len(record.Goal.WorkItems())-1]
	reworkOf, linked := successor.ReworkOf()
	policy, declared := successor.CouncilPolicy()
	var subject, decisionRef, decisionDigest string
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT council_subject_digest,
council_decision_ref,council_decision_digest FROM director_decisions
WHERE council_decision_ref=?`, decision.Ref).Scan(&subject, &decisionRef, &decisionDigest))
	if !sourceFound || source.State() != goal.WorkItemStateSuperseded || !authorFound ||
		author.State != application.ExecutionCanceled ||
		author.FailureCode != "application.execution_superseded" ||
		!linked || reworkOf != source.Ref() || !declared || policy != council.PolicyRequired ||
		subject != string(decision.SubjectDigest) || decisionRef != decision.Ref ||
		decisionDigest != string(decision.DecisionDigest) {
		t.Fatalf("Council replan source=%+v author=%+v successor=%+v decision=%q/%q/%q",
			source, author, successor, subject, decisionRef, decisionDigest)
	}
}
