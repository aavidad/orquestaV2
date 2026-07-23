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
	"orquesta/internal/ports"
)

type sqliteV19CouncilObserver struct {
	base    *sqliteV15External
	fail    bool
	ballots map[council.Role]council.Ballot
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

func newSQLiteV19CouncilOrchestrator(
	t *testing.T, system *sqliteV15System, observer application.AgentObserver,
) *application.Orchestrator {
	t.Helper()
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
	})
	sqliteTestNoError(t, err)
	return orchestrator
}

func seedSQLiteV19Council(
	t *testing.T, policy council.Policy, fail bool,
) (*sqliteV15System, goal.GoalRef) {
	t.Helper()
	system := newSQLiteV15System(t, 8)
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

func TestSQLiteV19CouncilRetryAndExhaustionPreservePeersAtomically(t *testing.T) {
	system, goalRef := seedSQLiteV19Council(t, council.PolicyAuto, true)
	processed := 0
	for i := 0; i < 50 && processed < 18; i++ {
		result, err := system.orchestrator.ProcessNext(context.Background(), "worker:v19-failure")
		sqliteTestNoError(t, err)
		if result.Processed {
			processed++
		} else {
			system.clock.Advance(time.Second)
		}
	}
	record, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	failed, councilExecutions := 0, 0
	for _, execution := range record.Executions {
		if execution.CouncilSubjectDigest == "" {
			continue
		}
		councilExecutions++
		if execution.State == application.ExecutionFailed {
			failed++
		}
	}
	if councilExecutions != 9 || failed != 9 || len(record.CouncilFacts) != 0 || len(record.CouncilDecisions) != 0 {
		t.Fatalf("executions=%d failed=%d facts=%d decisions=%d", councilExecutions, failed,
			len(record.CouncilFacts), len(record.CouncilDecisions))
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
