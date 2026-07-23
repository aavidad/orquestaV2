package sqlite

import (
	"context"
	"sync"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/review"
)

func TestFakeAgentsGitSQLiteCASAttestorReviewsEndToEnd(t *testing.T) {
	attestor := &sqliteTestAttestor{}
	system, goalRef := seedSQLiteV17Committed(t, attestor)
	processSQLiteV16Actions(t, system,
		application.ActionAttestTest,
		application.ActionLaunchAgent, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionObserveAgent,
	)
	admitSQLiteV18Integration(t, system, goalRef)
	processSQLiteV16Actions(t, system, application.ActionIntegrateChange)
	assertSQLiteV18ReviewClosure(t, system, goalRef)
}

func TestReviewCrashFrontiersReplayWithoutDuplicateLaunchOrDecision(t *testing.T) {
	frontiers := []struct {
		name   string
		before []application.ActionKind
		after  []application.ActionKind
		admit  bool
	}{
		{
			name:   "before_reviews",
			before: []application.ActionKind{application.ActionAttestTest},
			after: []application.ActionKind{application.ActionLaunchAgent, application.ActionLaunchAgent,
				application.ActionObserveAgent, application.ActionObserveAgent},
		},
		{
			name: "after_primary",
			before: []application.ActionKind{application.ActionAttestTest,
				application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionObserveAgent},
			after: []application.ActionKind{application.ActionObserveAgent},
		},
		{
			name: "after_both",
			before: []application.ActionKind{application.ActionAttestTest,
				application.ActionLaunchAgent, application.ActionLaunchAgent,
				application.ActionObserveAgent, application.ActionObserveAgent},
		},
		{
			name: "after_integration_admission",
			before: []application.ActionKind{application.ActionAttestTest,
				application.ActionLaunchAgent, application.ActionLaunchAgent,
				application.ActionObserveAgent, application.ActionObserveAgent},
			admit: true,
		},
	}
	for _, frontier := range frontiers {
		t.Run(frontier.name, func(t *testing.T) {
			attestor := &sqliteTestAttestor{}
			system, goalRef := seedSQLiteV17Committed(t, attestor)
			processSQLiteV16Actions(t, system, frontier.before...)
			if frontier.admit {
				admitSQLiteV18Integration(t, system, goalRef)
			}
			sqliteTestNoError(t, system.repository.Close())
			system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
			system.orchestrator = newSQLiteV16OrchestratorWithAttestor(t, system, attestor)
			processSQLiteV16Actions(t, system, frontier.after...)
			if !frontier.admit {
				admitSQLiteV18Integration(t, system, goalRef)
			}
			processSQLiteV16Actions(t, system, application.ActionIntegrateChange)
			assertSQLiteV18ReviewClosure(t, system, goalRef)
		})
	}
}

func TestReviewRetryIntegratesAndReopensWithHistoricalAttempts(t *testing.T) {
	attestor := &sqliteTestAttestor{}
	system, goalRef := seedSQLiteV17Committed(t, attestor)
	processSQLiteV16Actions(t, system,
		application.ActionAttestTest, application.ActionLaunchAgent, application.ActionLaunchAgent,
	)
	rewriteRecoveryTrigger(t, system.repository.db, "executions_identity_immutable", func() {
		mustV10Exec(t, system.repository.db, `UPDATE executions SET max_execution_attempts=2
WHERE purpose IN ('primary_review','adversarial_review')`)
	})
	system.external.mu.Lock()
	system.external.reviewContent = []byte(`{"malformed":"retry-once"}`)
	system.external.mu.Unlock()
	processSQLiteV16Actions(t, system, application.ActionObserveAgent)
	system.external.mu.Lock()
	system.external.reviewContent = nil
	system.external.mu.Unlock()
	processSQLiteV16Actions(t, system,
		application.ActionObserveAgent, application.ActionLaunchAgent, application.ActionObserveAgent,
	)
	admitSQLiteV18Integration(t, system, goalRef)
	processSQLiteV16Actions(t, system, application.ActionIntegrateChange)

	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.orchestrator = newSQLiteV16OrchestratorWithAttestor(t, system, attestor)
	record, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	reviewers := 0
	for _, execution := range record.Executions {
		if execution.Purpose == application.ExecutionPurposePrimaryReview ||
			execution.Purpose == application.ExecutionPurposeAdversarialReview {
			reviewers++
		}
	}
	if record.Goal.State() != goal.GoalStateSucceeded || reviewers != 3 || len(record.Reviews) != 2 ||
		len(record.IntegrationReceipts) != 1 {
		t.Fatalf("reopened retry closure state=%s reviewers=%d reviews=%d integrations=%d",
			record.Goal.State(), reviewers, len(record.Reviews), len(record.IntegrationReceipts))
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("reopened retry closure recovery: %v", err)
	}
}

func TestConcurrentReviewClaimsKeepOneDecisionPerRole(t *testing.T) {
	attestor := &sqliteTestAttestor{}
	system, goalRef := seedSQLiteV17Committed(t, attestor)
	processSQLiteV16Actions(t, system,
		application.ActionAttestTest, application.ActionLaunchAgent, application.ActionLaunchAgent,
	)
	type outcome struct {
		result application.ProcessResult
		err    error
	}
	start, outcomes := make(chan struct{}), make(chan outcome, 2)
	var workers sync.WaitGroup
	for index := 0; index < 2; index++ {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			<-start
			result, err := system.orchestrator.ProcessNext(context.Background(),
				"worker:v18-review-concurrent:"+string(rune('a'+worker)))
			outcomes <- outcome{result: result, err: err}
		}(index)
	}
	close(start)
	workers.Wait()
	close(outcomes)
	processed := 0
	for value := range outcomes {
		if value.err != nil {
			t.Fatalf("concurrent review: %v", value.err)
		}
		if value.result.Processed {
			if value.result.Action != application.ActionObserveAgent {
				t.Fatalf("concurrent action=%+v", value.result)
			}
			processed++
		}
	}
	if processed < 1 || processed > 2 {
		t.Fatalf("concurrent review decisions=%d want one or two fenced claims", processed)
	}
	if processed == 1 {
		processSQLiteV16Actions(t, system, application.ActionObserveAgent)
	}
	assertSQLiteV18ReviewPair(t, system, goalRef)
}

func admitSQLiteV18Integration(t *testing.T, system *sqliteV15System, goalRef goal.GoalRef) {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	if policy, found := record.Goal.WorkItems()[0].CouncilPolicy(); found && policy == council.PolicyAuto &&
		len(record.CouncilDecisions) == 0 {
		processSQLiteV16Actions(t, system,
			application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionLaunchAgent,
			application.ActionObserveAgent, application.ActionObserveAgent, application.ActionObserveAgent)
		record, err = system.repository.GetGoal(context.Background(), goalRef)
		sqliteTestNoError(t, err)
	}
	result, err := system.orchestrator.IntegrateChange(context.Background(), system.access,
		application.IntegrateChangeRequest{
			RequestRef: "request:v18-e2e-integration", GoalRef: goalRef,
			ChangeRef: record.ChangeSets[0].Ref, ExpectedTargetOID: record.WorkspaceBindings[0].BaseOID,
		})
	if err != nil || !result.Created || result.Action.Kind != application.ActionIntegrateChange {
		t.Fatalf("V18 integration admission result=%+v err=%v", result, err)
	}
}

func assertSQLiteV18ReviewPair(t *testing.T, system *sqliteV15System, goalRef goal.GoalRef) {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	roles, executions, launches, processes := map[review.Role]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, fact := range record.Reviews {
		if roles[fact.Role] || executions[fact.ReviewerExecutionRef.String()] || launches[fact.LaunchReceiptRef] ||
			processes[fact.ExternalRef] || fact.Verdict != review.VerdictApprove {
			t.Fatalf("duplicate or invalid review fact: %+v", record.Reviews)
		}
		roles[fact.Role], executions[fact.ReviewerExecutionRef.String()] = true, true
		launches[fact.LaunchReceiptRef], processes[fact.ExternalRef] = true, true
	}
	if len(record.Reviews) != 2 || !roles[review.RolePrimary] || !roles[review.RoleAdversarial] {
		t.Fatalf("V18 review pair=%+v", record.Reviews)
	}
}

func assertSQLiteV18ReviewClosure(t *testing.T, system *sqliteV15System, goalRef goal.GoalRef) {
	t.Helper()
	assertSQLiteV18ReviewPair(t, system, goalRef)
	record, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	if record.Goal.State() != goal.GoalStateSucceeded || len(record.IntegrationReceipts) != 1 {
		t.Fatalf("V18 closure state=%s receipts=%d", record.Goal.State(), len(record.IntegrationReceipts))
	}
	system.external.mu.Lock()
	launches := system.external.launchCalls
	system.external.mu.Unlock()
	if launches != 6 {
		t.Fatalf("participant launches=%d want author+review pair+Council cohort", launches)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("V18 closure recovery: %v", err)
	}
}
