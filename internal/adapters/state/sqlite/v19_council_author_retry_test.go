package sqlite

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestSQLiteV19CouncilAuthorRetryPassReviewsAndThreeMemberRoundSurvivesRestart(t *testing.T) {
	system := newSQLiteV15System(t, 8)
	observer := &sqliteV19CouncilObserver{base: system.external}
	system.orchestrator = newSQLiteV19CouncilOrchestrator(t, system, observer)
	submitted, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: "request:v19-council-author-retry", Statement: "retry author before exact Council", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:v19-council-author-retry", Key: "phase:v19-council-author-retry",
				TemplateRef: "phase-template:v19-council-author-retry",
			}},
			WorkItems: []application.WorkItemSpec{{
				Key: "writer", Objective: "persist exact retried author evidence",
				Phase: "phase:v19-council-author-retry", Role: "role:writer",
				WriteSet:       []string{"internal/v19-author-retry"},
				CouncilPolicy:  council.PolicyAuto,
				RequiredTests:  sqliteRequiredTestSpecs("required-test:v19-council-author-retry"),
				OutputContract: goal.OutputContractEvidenceBundle,
			}},
		},
	})
	sqliteTestNoError(t, err)
	goalRef := submitted.Record.Goal.Ref()

	system.external.mu.Lock()
	system.external.observationStatus = ports.AgentFailed
	system.external.observationError = "provider.retryable_failure"
	system.external.mu.Unlock()
	processSQLiteV16Actions(t, system,
		application.ActionPrepareWorkspace, application.ActionLaunchAgent, application.ActionObserveAgent)

	system.external.mu.Lock()
	system.external.observationStatus = ports.AgentCompleted
	system.external.observationError = ""
	system.external.mu.Unlock()
	system.clock.Advance(time.Second)
	processSQLiteV16Actions(t, system,
		application.ActionPrepareWorkspace, application.ActionLaunchAgent, application.ActionObserveAgent,
		application.ActionCommitChange, application.ActionAttestTest,
		application.ActionLaunchAgent, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionObserveAgent)

	beforeRestart, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	assertSQLiteV19CouncilAuthorRetryFrontier(t, beforeRestart)
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("pre-restart recovery validation: %v", err)
	}

	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.orchestrator = newSQLiteV19CouncilOrchestrator(t, system,
		&sqliteV19CouncilObserver{base: system.external})
	restarted, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	assertSQLiteV19CouncilAuthorRetryFrontier(t, restarted)

	processSQLiteV16Actions(t, system,
		application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionObserveAgent, application.ActionObserveAgent)
	decided, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	if len(decided.CouncilFacts) != 3 || len(decided.CouncilDecisions) != 1 ||
		decided.CouncilDecisions[0].Decision.Outcome != council.OutcomeAccepted {
		t.Fatalf("three-member Council did not decide: facts=%d decisions=%+v",
			len(decided.CouncilFacts), decided.CouncilDecisions)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("decided recovery validation: %v", err)
	}
}

func assertSQLiteV19CouncilAuthorRetryFrontier(t *testing.T, record application.GoalRecord) {
	t.Helper()
	failedAuthors, currentAuthors, passes, councilMembers := 0, 0, 0, 0
	for _, execution := range record.Executions {
		switch {
		case execution.Purpose == application.ExecutionPurposeAuthor &&
			execution.State == application.ExecutionFailed:
			failedAuthors++
		case execution.Purpose == application.ExecutionPurposeAuthor &&
			execution.State == application.ExecutionAwaitingIntegration &&
			execution.AttemptNo == 2:
			currentAuthors++
			if len(record.ChangeSets) != 1 || record.ChangeSets[0].ExecutionRef != execution.Ref {
				t.Fatalf("ChangeSet does not name retried author: changes=%+v author=%+v",
					record.ChangeSets, execution)
			}
		case execution.CouncilSubjectDigest != "":
			councilMembers++
		}
	}
	for _, attestation := range record.Attestations {
		if attestation.Kind == application.AttestationKindRequiredTests &&
			attestation.Verdict == application.AttestationVerdictPassed {
			passes++
		}
	}
	if failedAuthors != 1 || currentAuthors != 1 || passes != 1 ||
		len(record.Reviews) != 2 || len(record.CouncilRounds) != 1 || councilMembers != 3 {
		t.Fatalf("retry/PASS/reviews/Council frontier invalid: failed=%d current=%d PASS=%d reviews=%d rounds=%d members=%d",
			failedAuthors, currentAuthors, passes, len(record.Reviews), len(record.CouncilRounds), councilMembers)
	}
	if err := application.ValidatePersistedCouncilSubject(record, record.CouncilRounds[0].Subject); err != nil {
		t.Fatalf("persisted retried author Council subject invalid: %v", err)
	}
}
