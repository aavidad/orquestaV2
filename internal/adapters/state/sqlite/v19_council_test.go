package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

type sqliteV19CouncilObserver struct {
	base             *sqliteV15External
	fail             bool
	terminalSecurity bool
	ballots          map[council.Role]council.Ballot
}

func (observer *sqliteV19CouncilObserver) Observe(
	ctx context.Context, executionRef goal.ExecutionRef,
) (ports.AgentObservation, error) {
	observer.base.mu.Lock()
	request, requested := observer.base.launchRequests[executionRef]
	var receipt ports.AgentLaunchReceipt
	for _, candidate := range observer.base.launches {
		if candidate.ExecutionRef == executionRef {
			receipt = candidate
			break
		}
	}
	observer.base.mu.Unlock()
	if !requested || request.ArtifactMediaType != council.ContributionMediaType {
		return observer.base.Observe(ctx, executionRef)
	}
	if observer.terminalSecurity {
		return ports.AgentObservation{ExecutionRef: executionRef, SpecHash: receipt.SpecHash,
			Status: ports.AgentFailed, FailureDisposition: ports.AgentFailureDispositionTerminalSecurity,
			ErrorCode: "provider.security_failure", Usage: observer.base.observationUsage,
			ObservedAt: observer.base.clock.Now()}, nil
	}
	if observer.fail {
		return ports.AgentObservation{ExecutionRef: executionRef, SpecHash: receipt.SpecHash,
			Status: ports.AgentCompleted, MediaType: council.ContributionMediaType,
			Content: []byte(`{"invalid":true}`), Usage: observer.base.observationUsage,
			ObservedAt: observer.base.clock.Now()}, nil
	}
	role := council.RoleProposer
	for _, candidate := range council.Roles() {
		if strings.Contains(request.Objective, `"role":"`+string(candidate)+`"`) {
			role = candidate
		}
	}
	var evidence struct {
		SubjectDigest string `json:"subject_digest"`
		ReviewGate    string `json:"review_gate_digest"`
	}
	prefix := "Assess exact approved V18 evidence "
	value := strings.TrimPrefix(request.Objective, prefix)
	if end := strings.Index(value, ". Return exactly"); end < 0 ||
		json.Unmarshal([]byte(value[:end]), &evidence) != nil {
		return ports.AgentObservation{}, errors.New("sqlite.v19_council_evidence_invalid")
	}
	ballot := council.BallotAccept
	if configured := observer.ballots[role]; configured != "" {
		ballot = configured
	}
	refs := []council.Evidence{{Kind: "review_gate", Ref: evidence.ReviewGate}}
	if ballot == council.BallotSecurityVeto {
		refs = append(refs, council.Evidence{Kind: "security_veto", Ref: "evidence:sqlite-v19-security-veto"})
	}
	content, _ := json.Marshal(council.Contribution{Schema: council.ContributionSchema,
		SubjectDigest: evidence.SubjectDigest, Role: role, Body: "sqlite exact contribution",
		Ballot: ballot, Evidence: refs})
	return ports.AgentObservation{ExecutionRef: executionRef, SpecHash: receipt.SpecHash,
		Status: ports.AgentCompleted, MediaType: council.ContributionMediaType,
		Content: content, Usage: observer.base.observationUsage, ObservedAt: observer.base.clock.Now()}, nil
}

func (observer *sqliteV19CouncilObserver) ObserveAgent(
	ctx context.Context,
	request ports.AgentObserveRequest,
) (ports.AgentObservation, error) {
	return observer.Observe(ctx, request.ExecutionRef)
}

func newSQLiteV19CouncilOrchestrator(
	t *testing.T, system *sqliteV15System, observer application.AgentObserver,
) *application.Orchestrator {
	t.Helper()
	_, fuentes := prepararCapacidadSQLiteV15(t, system.repository, system.clock, 1_000)
	orchestrator, err := application.New(application.Dependencies{
		State: system.repository, Access: system.repository, Launcher: system.external, Observer: observer,
		Controller: system.external, Artifacts: system.external, WorkspaceManager: &sqliteTestWorkspaceManager{},
		VersionControl: &sqliteTestVersionControl{}, TestAttestor: &sqliteTestAttestor{},
		TestAttestationPolicy: application.TestAttestationPolicy{
			Ref: "test-attestation-policy:sqlite", Digest: strings.Repeat("b", 64)},
		Clock: system.clock, IDs: system.ids, MaxOutputBytes: 1024, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, MaxChildrenPerParent: 6, ClaimLease: time.Minute,
		AttestTestClaimLease: 20 * time.Minute, DirectorLeaseDuration: 30 * time.Second,
		EffectApprovalTTL: system.policy.EffectApprovalTTL, BudgetPolicy: system.policy,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour, AgentCapabilities: sqliteTestCapabilities(),
		CapacitySources: fuentes, CapacityObservationWait: time.Second,
	})
	sqliteTestNoError(t, err)
	return orchestrator
}

func seedSQLiteV19Council(
	t *testing.T, policy council.Policy, fail bool,
) (*sqliteV15System, goal.GoalRef) {
	t.Helper()
	system := newSQLiteV15System(t, 8)
	return seedSQLiteV19CouncilOnSystem(t, system, policy, fail)
}

func seedSQLiteV19CouncilOnSystem(
	t *testing.T,
	system *sqliteV15System,
	policy council.Policy,
	fail bool,
) (*sqliteV15System, goal.GoalRef) {
	t.Helper()
	observer := &sqliteV19CouncilObserver{base: system.external, fail: fail}
	system.orchestrator = newSQLiteV19CouncilOrchestrator(t, system, observer)
	result, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: "request:v19-council:" + string(policy), Statement: "persist council lifecycle", Confirm: true,
		Plan: &application.PlanSpec{Phases: []application.PhaseSpec{{
			Ref: "phase-instance:v19-council", Key: "phase:v19-council", TemplateRef: "phase-template:v19-council",
		}}, WorkItems: []application.WorkItemSpec{{
			Key: "writer", Objective: "write isolated council evidence", Phase: "phase:v19-council",
			Role: "role:writer", WriteSet: []string{"internal/v19"}, CouncilPolicy: policy,
			RequiredTests:  sqliteRequiredTestSpecs("required-test:v19-council"),
			OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	})
	sqliteTestNoError(t, err)
	processSQLiteV16Actions(t, system, application.ActionPrepareWorkspace, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionCommitChange, application.ActionAttestTest,
		application.ActionLaunchAgent, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionObserveAgent)
	return system, result.Record.Goal.Ref()
}

func TestSQLiteV19TerminalSecurityCouncilCleanupRestartAndReplay(t *testing.T) {
	system, goalRef := seedSQLiteV19Council(t, council.PolicyAuto, false)
	processSQLiteV16Actions(t, system,
		application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionLaunchAgent,
		application.ActionObserveAgent)
	system.orchestrator = newSQLiteV19CouncilOrchestrator(t, system,
		&sqliteV19CouncilObserver{base: system.external, terminalSecurity: true})
	processSQLiteV16Actions(t, system, application.ActionObserveAgent)
	mid, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	failed, running, replacements := 0, 0, 0
	for _, execution := range mid.Executions {
		if execution.CouncilSubjectDigest == "" {
			continue
		}
		if execution.State == application.ExecutionFailed && execution.FailureCode == "provider.security_failure" {
			failed++
		}
		if execution.State == application.ExecutionRunning {
			running++
		}
		if execution.ReplacesExecutionRef.String() != "" {
			replacements++
		}
	}
	if failed != 1 || running != 1 || replacements != 0 || len(mid.CouncilFacts) != 1 ||
		len(mid.CouncilDecisions) != 0 {
		t.Fatalf("terminal Council mid failed=%d running=%d replacements=%d facts=%d decisions=%d",
			failed, running, replacements, len(mid.CouncilFacts), len(mid.CouncilDecisions))
	}
	processSQLiteV16Actions(t, system, application.ActionStopAgent)
	closed, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	item := closed.Goal.WorkItems()[0]
	authorFailed, peersStopped := 0, 0
	for _, execution := range closed.Executions {
		switch {
		case execution.Purpose == application.ExecutionPurposeAuthor &&
			execution.State == application.ExecutionFailed && execution.FailureCode == "council.unavailable":
			authorFailed++
		case execution.CouncilSubjectDigest != "" && execution.State == application.ExecutionStopped &&
			execution.FailureCode == "council.round_aborted":
			peersStopped++
		}
	}
	if closed.Goal.State() == goal.GoalStateSucceeded || item.State() != goal.WorkItemStateInterrupted ||
		authorFailed != 1 || peersStopped != 1 || len(closed.CouncilFacts) != 1 ||
		len(closed.CouncilDecisions) != 0 || len(closed.IntegrationReceipts) != 0 {
		t.Fatalf("terminal Council closed state=%s item=%s author=%d peers=%d facts=%d decisions=%d integrations=%d",
			closed.Goal.State(), item.State(), authorFailed, peersStopped, len(closed.CouncilFacts),
			len(closed.CouncilDecisions), len(closed.IntegrationReceipts))
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("terminal Council recovery validation: %v", err)
	}
	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.orchestrator = newSQLiteV19CouncilOrchestrator(t, system,
		&sqliteV19CouncilObserver{base: system.external, terminalSecurity: true})
	restarted, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	if restarted.Goal.WorkItems()[0].State() != goal.WorkItemStateInterrupted ||
		len(restarted.CouncilDecisions) != 0 {
		t.Fatalf("restart terminal Council item=%s decisions=%d",
			restarted.Goal.WorkItems()[0].State(), len(restarted.CouncilDecisions))
	}
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:terminal-council-replay")
	if err != nil || result.Processed {
		t.Fatalf("terminal Council replay result=%+v err=%v", result, err)
	}
}

func TestSQLiteV19CouncilRoundFactsDecisionReplayRestartAndConcurrency(t *testing.T) {
	system, goalRef := seedSQLiteV19Council(t, council.PolicyAuto, false)
	record, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	if len(record.CouncilRounds) != 1 {
		t.Fatalf("rounds=%d", len(record.CouncilRounds))
	}
	queued, succeeded := 0, 0
	for _, execution := range record.Executions {
		if execution.CouncilSubjectDigest == "" {
			continue
		}
		if execution.State == application.ExecutionQueued {
			queued++
		}
		if execution.State == application.ExecutionSucceeded {
			succeeded++
		}
	}
	if queued != 3 || succeeded != 0 || len(record.CouncilFacts) != 0 {
		t.Fatalf("Council peer prepared/accepted/observed queued=%d succeeded=%d facts=%d",
			queued, succeeded, len(record.CouncilFacts))
	}
	round, authority := record.CouncilRounds[0], record.WorkItemAuthorities[0]
	replayed, created, err := system.repository.OpenCouncilRound(context.Background(),
		application.OpenCouncilRoundState{RequestRef: round.RequestRef, RequestFingerprint: round.RequestFingerprint,
			AuthorizationReceipt: authority.AuthorizationReceipt, PrincipalRef: authority.PrincipalRef,
			ProjectRef: record.Goal.Project(), GoalRef: record.Goal.Ref(),
			ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: record.Goal.WorkItems()[0].Revision(),
			Round: round, OperationAt: system.clock.Now()})
	if err != nil || created || replayed != round {
		t.Fatalf("round replay=%+v created=%t err=%v", replayed, created, err)
	}
	bad := round
	bad.RequestFingerprint = strings.Repeat("f", 64)
	if _, _, err := system.repository.OpenCouncilRound(context.Background(),
		application.OpenCouncilRoundState{RequestRef: round.RequestRef, RequestFingerprint: bad.RequestFingerprint,
			AuthorizationReceipt: authority.AuthorizationReceipt, PrincipalRef: authority.PrincipalRef,
			ProjectRef: record.Goal.Project(), GoalRef: record.Goal.Ref(),
			ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: record.Goal.WorkItems()[0].Revision(),
			Round: bad, OperationAt: system.clock.Now()}); err == nil {
		t.Fatal("cross-substituted round replay accepted")
	}
	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.orchestrator = newSQLiteV19CouncilOrchestrator(t, system,
		&sqliteV19CouncilObserver{base: system.external})

	start := make(chan struct{})
	results := make(chan int, 2)
	errs := make(chan error, 2)
	var workers sync.WaitGroup
	for i := 0; i < 2; i++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			<-start
			result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v19:"+string(rune('a'+index)))
			if err != nil {
				err = fmt.Errorf("%s", sqliteTestErrorChain(err))
			}
			if err == nil && result.Processed && result.Action != application.ActionLaunchAgent {
				err = errors.New("unexpected council claim")
			}
			if result.Processed {
				results <- 1
			} else {
				results <- 0
			}
			errs <- err
		}(i)
	}
	close(start)
	workers.Wait()
	close(results)
	close(errs)
	launched := 0
	for value := range results {
		launched += value
	}
	for err := range errs {
		sqliteTestNoError(t, err)
	}
	for launched < 3 {
		processSQLiteV16Actions(t, system, application.ActionLaunchAgent)
		launched++
	}
	record, err = system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	running := 0
	for _, execution := range record.Executions {
		if execution.CouncilSubjectDigest != "" && execution.State == application.ExecutionRunning {
			running++
		}
	}
	if running != 3 {
		t.Fatalf("Council accepted peers running=%d", running)
	}
	processSQLiteV16Actions(t, system,
		application.ActionObserveAgent, application.ActionObserveAgent, application.ActionObserveAgent)
	record, err = system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	if len(record.CouncilFacts) != 3 || len(record.CouncilDecisions) != 1 ||
		record.CouncilDecisions[0].Decision.Outcome != council.OutcomeAccepted {
		t.Fatalf("facts=%d decisions=%+v", len(record.CouncilFacts), record.CouncilDecisions)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("Council recovery: %v", err)
	}
	admitSQLiteV18Integration(t, system, goalRef)
	admittedRecord, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	replayedAdmission, err := system.orchestrator.IntegrateChange(context.Background(), system.access,
		application.IntegrateChangeRequest{
			RequestRef: "request:v18-e2e-integration", GoalRef: goalRef,
			ChangeRef:         admittedRecord.ChangeSets[0].Ref,
			ExpectedTargetOID: admittedRecord.WorkspaceBindings[0].BaseOID,
		})
	if err != nil || replayedAdmission.Created ||
		replayedAdmission.Action.Kind != application.ActionIntegrateChange ||
		replayedAdmission.Action.CouncilResolution == nil ||
		replayedAdmission.Action.EffectIntent.CouncilResolution == nil {
		t.Fatalf("Council pointer-valued admission replay=%+v err=%v", replayedAdmission, err)
	}
	var outboxSubject, intentSubject, resolutionKind string
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT action.council_subject_digest,
action.council_resolution_kind,intent.council_subject_digest FROM outbox action
JOIN effect_intents intent ON intent.ref=action.effect_intent_ref
WHERE action.kind='integrate_change' AND action.completed_at IS NULL`).Scan(
		&outboxSubject, &resolutionKind, &intentSubject))
	if outboxSubject != string(record.CouncilDecisions[0].SubjectDigest) ||
		intentSubject != outboxSubject || resolutionKind != "accepted_round" {
		t.Fatalf("integration resolution outbox=%q intent=%q kind=%q", outboxSubject, intentSubject, resolutionKind)
	}
	processSQLiteV16Actions(t, system, application.ActionIntegrateChange)
	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	record, err = system.repository.GetGoal(context.Background(), goalRef)
	if err != nil || record.Goal.State() != goal.GoalStateSucceeded || len(record.CouncilDecisions) != 1 {
		t.Fatalf("integrated restart state=%s decisions=%d err=%v",
			record.Goal.State(), len(record.CouncilDecisions), err)
	}
}

func TestSQLiteV19CouncilRetryThenExhaustionAbortsCohortAtomically(t *testing.T) {
	system, goalRef := seedSQLiteV19Council(t, council.PolicyAuto, true)
	for i := 0; i < 100; i++ {
		result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v19-failure")
		sqliteTestNoError(t, err)
		record, getErr := system.repository.GetGoal(context.Background(), goalRef)
		sqliteTestNoError(t, getErr)
		if !result.Processed && record.Goal.WorkItems()[0].State() == goal.WorkItemStateInterrupted {
			break
		}
		if !result.Processed {
			system.clock.Advance(time.Second)
		}
	}
	record, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	failed, replacements, exhaustedLeaves, stopped := 0, 0, 0, 0
	for _, execution := range record.Executions {
		if execution.CouncilSubjectDigest == "" {
			continue
		}
		if execution.State == application.ExecutionFailed {
			failed++
		}
		if execution.ReplacesExecutionRef.String() != "" {
			replacements++
		}
		if execution.State == application.ExecutionFailed &&
			execution.FailureCode == "council.contribution_invalid" &&
			execution.AttemptNo == execution.MaxExecutionAttempts {
			exhaustedLeaves++
		}
		if execution.State == application.ExecutionStopped && execution.FailureCode == "council.round_aborted" {
			stopped++
		}
	}
	item := record.Goal.WorkItems()[0]
	authorFailures := 0
	for _, execution := range record.Executions {
		if execution.Purpose == application.ExecutionPurposeAuthor &&
			execution.State == application.ExecutionFailed && execution.FailureCode == "council.unavailable" {
			authorFailures++
		}
	}
	if replacements == 0 || exhaustedLeaves != 1 || failed == 0 ||
		item.State() != goal.WorkItemStateInterrupted || authorFailures != 1 ||
		len(record.CouncilFacts) != 0 || len(record.CouncilDecisions) != 0 ||
		len(record.IntegrationReceipts) != 0 {
		t.Fatalf("replacements=%d exhausted=%d failed=%d stopped=%d item=%s author=%d facts=%d decisions=%d integrations=%d",
			replacements, exhaustedLeaves, failed, stopped, item.State(), authorFailures,
			len(record.CouncilFacts), len(record.CouncilDecisions), len(record.IntegrationReceipts))
	}
	var persistedFacts, persistedDecisions int
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT COUNT(*) FROM council_facts`).Scan(&persistedFacts))
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT COUNT(*) FROM council_decisions`).Scan(&persistedDecisions))
	if persistedFacts != 0 || persistedDecisions != 0 {
		t.Fatalf("failed contribution partially persisted facts=%d decisions=%d", persistedFacts, persistedDecisions)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("failed Council recovery: %v", err)
	}
	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.orchestrator = newSQLiteV19CouncilOrchestrator(t, system,
		&sqliteV19CouncilObserver{base: system.external, fail: true})
	restarted, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	if restarted.Goal.WorkItems()[0].State() != goal.WorkItemStateInterrupted ||
		len(restarted.CouncilDecisions) != 0 {
		t.Fatalf("restart exhausted Council item=%s decisions=%d",
			restarted.Goal.WorkItems()[0].State(), len(restarted.CouncilDecisions))
	}
	result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v19-failure-replay")
	if err != nil || result.Processed {
		t.Fatalf("exhausted Council replay result=%+v err=%v", result, err)
	}
}

func TestSQLiteV19PreparedCouncilRetryClosesLocallyAfterPeerSettlement(t *testing.T) {
	system := newSQLiteV15System(t, 8)
	for _, envelope := range []*governance.BudgetEnvelope{
		&system.policy.DeploymentEnvelope,
		&system.policy.ProjectEnvelopeTemplate,
		&system.policy.GoalEnvelopeTemplate,
	} {
		envelope.Limit.Tokens = 700
		envelope.Limit.MoneyMicros = 700
	}
	system, goalRef := seedSQLiteV19CouncilOnSystem(t, system, council.PolicyAuto, true)
	processSQLiteV16Actions(t, system,
		application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionLaunchAgent,
		application.ActionObserveAgent,
	)
	queued, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	var retry application.ExecutionRecord
	for _, execution := range queued.Executions {
		if execution.CouncilSubjectDigest != "" && execution.AttemptNo == 2 {
			retry = execution
			break
		}
	}
	if retry.Ref.String() == "" || retry.State != application.ExecutionQueued {
		t.Fatalf("Council retry not queued: %+v", queued.Executions)
	}

	system.clock.Advance(system.policy.QuotaRetryDelay)
	retryClaim := claimSQLiteV15(t, system, "claim:v19-council-prepared-retry")
	if retryClaim.Action.ExecutionRef != retry.Ref ||
		retryClaim.Disposition != application.ActionClaimDispositionNormal {
		t.Fatalf("Council prepared retry claim=%+v want=%s", retryClaim, retry.Ref)
	}
	_, sessionRef := prepareSQLiteV15RetryLaunchWithSession(t, system, retryClaim)

	// Model the process crash after RecordLaunchPrepared. The launch lease
	// expires, while a bounded scheduler deferral lets an independent Council
	// peer publish the later durable settlement first.
	system.clock.Advance(2 * time.Minute)
	mustV10Exec(t, system.repository.db, `
UPDATE outbox SET available_at=? WHERE ref=?`,
		requiredTime(system.clock.Now().Add(time.Hour)), retryClaim.Action.Ref)
	system.external.observationUsage = governance.ResourceUsage{
		Resources: governance.ResourceVector{
			Tokens: 300, MoneyMicros: 300, Currency: governance.Currency("USD"),
		},
		Known: governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
	}
	system.orchestrator = newSQLiteV19CouncilOrchestrator(t, system,
		&sqliteV19CouncilObserver{base: system.external})
	processSQLiteV16Actions(t, system, application.ActionObserveAgent)
	mustV10Exec(t, system.repository.db, `
UPDATE outbox SET available_at=?
WHERE kind='observe_agent' AND completed_at IS NULL`,
		requiredTime(system.clock.Now().Add(time.Hour)))
	mustV10Exec(t, system.repository.db, `
UPDATE outbox SET available_at=? WHERE ref=?`,
		requiredTime(system.clock.Now()), retryClaim.Action.Ref)

	revokeSQLiteV15Owner(t, system)
	system.clock.Advance(2 * time.Minute)
	if result, processErr := system.orchestrator.ProcessNext(
		context.Background(), "worker:v19-council-prepared-park",
	); processErr != nil || result.Processed {
		t.Fatalf("Council prepared parking result=%+v err=%v", result, processErr)
	}
	parked, err := system.repository.GetGoal(context.Background(), goalRef)
	parkedRetry, found := sqliteExecutionByRef(parked.Executions, retry.Ref)
	if err != nil || !found || parkedRetry.State != application.ExecutionDispatching ||
		parkedRetry.BudgetReservationRef != "" || parkedRetry.EffectIntentRef != "" ||
		parkedRetry.ExecutionSessionRef != sessionRef {
		t.Fatalf("Council parked retry=%+v found=%v err=%v", parkedRetry, found, err)
	}

	system.clock.Advance(system.policy.QuotaRetryDelay)
	launchesBefore := system.external.launchCalls
	processSQLiteV16Actions(t, system, application.ActionLaunchAgent)
	final, err := system.repository.GetGoal(context.Background(), goalRef)
	failed, found := sqliteExecutionByRef(final.Executions, retry.Ref)
	var consumed, revokeActions, attempts int
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT COUNT(*) FROM action_consumption_receipts
WHERE execution_ref=? AND kind='launch_agent'
 AND error_code LIKE 'budget.retry_irreversible:%'`, retry.Ref.String()).Scan(&consumed))
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT COUNT(*) FROM outbox
WHERE execution_ref=? AND kind='revoke_execution_session' AND completed_at IS NULL`,
		retry.Ref.String()).Scan(&revokeActions))
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT COUNT(*) FROM effect_attempts WHERE execution_ref=?`,
		retry.Ref.String()).Scan(&attempts))
	if err != nil || !found || failed.State != application.ExecutionFailed ||
		failed.FailureCode != "council.contribution_invalid" ||
		failed.ExecutionSessionRef != sessionRef ||
		consumed != 1 || revokeActions != 1 || attempts != 0 ||
		system.external.launchCalls != launchesBefore {
		t.Fatalf("Council local closure failed=%+v found=%v consumed=%d revoke=%d attempts=%d calls=%d/%d err=%v",
			failed, found, consumed, revokeActions, attempts, system.external.launchCalls, launchesBefore, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("prepared Council recovery: %v cause=%v", err, errors.Unwrap(err))
	}
}

func TestSQLiteV19CouncilSkipReplayAndRestart(t *testing.T) {
	system, goalRef := seedSQLiteV19Council(t, council.PolicySkipByOperator, false)
	record, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	item := record.Goal.WorkItems()[0]
	request := application.SkipCouncilRequest{RequestRef: "request:v19-skip", GoalRef: goalRef,
		ChangeRef: record.ChangeSets[0].Ref, ExpectedGoalRevision: record.Goal.Revision(),
		ExpectedItemRevision: item.Revision(), Reason: "operator reviewed approved gate"}
	first, err := system.orchestrator.SkipCouncil(context.Background(), system.access, request)
	if err != nil {
		t.Fatal(sqliteTestErrorChain(err))
	}
	second, err := system.orchestrator.SkipCouncil(context.Background(), system.access, request)
	if err != nil || !first.Created || second.Created || first.Skip != second.Skip {
		t.Fatalf("skip replay first=%+v second=%+v err=%v", first, second, err)
	}
	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	record, err = system.repository.GetGoal(context.Background(), goalRef)
	if err != nil || len(record.CouncilSkips) != 1 || record.CouncilSkips[0] != first.Skip {
		t.Fatalf("skip restart=%+v err=%v", record.CouncilSkips, err)
	}
}
