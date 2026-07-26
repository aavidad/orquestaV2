package sqlite

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
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

func TestReviewQueuedRetryClosesLocallyWhenPeerSettlementExhaustsBudget(t *testing.T) {
	attestor := &sqliteTestAttestor{}
	system := newSQLiteV15System(t, 4)
	for _, envelope := range []*governance.BudgetEnvelope{
		&system.policy.DeploymentEnvelope,
		&system.policy.ProjectEnvelopeTemplate,
		&system.policy.GoalEnvelopeTemplate,
	} {
		envelope.Limit.Tokens = 400
		envelope.Limit.MoneyMicros = 400
	}
	system.orchestrator = newSQLiteV16OrchestratorWithAttestor(t, system, attestor)
	submitted, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: "request:v18-review-late-budget", Statement: "close exhausted reviewer retry", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:v18-review-late-budget", Key: "phase:v18-review-late-budget",
				TemplateRef: "phase-template:v18-review-late-budget",
			}},
			WorkItems: []application.WorkItemSpec{{
				Key: "writer", Objective: "produce a reviewed isolated change",
				Phase: "phase:v18-review-late-budget", Role: "role:writer",
				WriteSet:       []string{"internal/v18-budget"},
				CouncilPolicy:  council.PolicyAuto,
				RequiredTests:  sqliteRequiredTestSpecs("required-test:v18-budget"),
				OutputContract: goal.OutputContractEvidenceBundle,
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	processSQLiteV16Actions(t, system,
		application.ActionPrepareWorkspace, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionCommitChange,
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
	queued, err := system.repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	retryRef := ""
	for _, execution := range queued.Executions {
		if execution.AttemptNo == 2 && (execution.Purpose == application.ExecutionPurposePrimaryReview ||
			execution.Purpose == application.ExecutionPurposeAdversarialReview) {
			retryRef = execution.Ref.String()
		}
	}
	if err != nil || retryRef == "" {
		t.Fatalf("review retry not queued record=%+v err=%v", queued, err)
	}
	system.clock.Advance(system.policy.QuotaRetryDelay)
	retryClaim := claimSQLiteV15(t, system, "claim:v18-review-prepared-retry")
	if retryClaim.Action.ExecutionRef.String() != retryRef ||
		retryClaim.Disposition != application.ActionClaimDispositionNormal {
		t.Fatalf("review prepared retry claim=%+v want execution=%s", retryClaim, retryRef)
	}
	_, sessionRef := prepareSQLiteV15RetryLaunchWithSession(t, system, retryClaim)
	system.clock.Advance(2 * time.Minute)
	mustV10Exec(t, system.repository.db, `
UPDATE outbox SET available_at=? WHERE ref=?`,
		requiredTime(system.clock.Now().Add(time.Hour)), retryClaim.Action.Ref)
	system.external.mu.Lock()
	system.external.reviewContent = nil
	system.external.observationUsage = governance.ResourceUsage{
		Resources: governance.ResourceVector{
			Tokens: 200, MoneyMicros: 200, Currency: governance.Currency("USD"),
		},
		Known: governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
	}
	system.external.mu.Unlock()
	processSQLiteV16Actions(t, system, application.ActionObserveAgent)
	mustV10Exec(t, system.repository.db, `
UPDATE outbox SET available_at=? WHERE ref=?`,
		requiredTime(system.clock.Now()), retryClaim.Action.Ref)
	revokeSQLiteV15Owner(t, system)
	system.clock.Advance(2 * time.Minute)
	if result, processErr := system.orchestrator.ProcessNext(
		context.Background(), "worker:v18-review-prepared-park",
	); processErr != nil || result.Processed {
		t.Fatalf("review prepared parking result=%+v err=%v", result, processErr)
	}
	system.clock.Advance(system.policy.QuotaRetryDelay)
	launchesBefore := system.external.launchCalls
	processSQLiteV16Actions(t, system, application.ActionLaunchAgent)
	record, err := system.repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	var retry application.ExecutionRecord
	for _, execution := range record.Executions {
		if execution.Ref.String() == retryRef {
			retry = execution
			break
		}
	}
	var consumed, revokeActions int
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT COUNT(*) FROM action_consumption_receipts
WHERE execution_ref=? AND kind='launch_agent'
 AND error_code LIKE 'budget.retry_irreversible:%'`, retryRef).Scan(&consumed))
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT COUNT(*) FROM outbox
WHERE execution_ref=? AND kind='revoke_execution_session' AND completed_at IS NULL`,
		retryRef).Scan(&revokeActions))
	if err != nil || retry.State != application.ExecutionFailed || consumed != 1 ||
		retry.ExecutionSessionRef != sessionRef || revokeActions != 1 ||
		system.external.launchCalls != launchesBefore {
		t.Fatalf("review late budget record=%+v retry=%+v consumed=%d revoke=%d calls=%d/%d err=%v",
			record, retry, consumed, revokeActions, system.external.launchCalls, launchesBefore, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("review late budget recovery: %v cause=%v", err, errors.Unwrap(err))
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
