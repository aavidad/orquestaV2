package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

func TestIntakeSQLiteRestartAndHistoricalReplay(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	create := system.create(t, "request:intake:create")
	first := system.apply(t, create.Record.State, "request:intake:first", intake.OriginChat,
		"intake-option:audience-personal")
	second := system.apply(t, first.Record.State, "request:intake:second", intake.OriginForm,
		"intake-option:audience-team")

	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	system.repository = openSQLiteIntakeTestRepository(t, system.path)
	service, err := application.NewIntakeService(system.repository)
	sqliteTestNoError(t, err)
	system.service = service

	current, err := system.service.GetIntake(ctx, application.GetIntakeRequest{
		ActorRef: system.principal.ActorRef, ProjectRef: system.project, StateRef: system.stateRef,
	})
	sqliteTestNoError(t, err)
	if current.State.Revision() != second.Record.State.Revision() ||
		!reflect.DeepEqual(application.SnapshotIntake(current.State), application.SnapshotIntake(second.Record.State)) ||
		current.Receipt != second.Record.Receipt {
		t.Fatalf("restart current mismatch: got=%+v want=%+v", current.Receipt, second.Record.Receipt)
	}

	historical, found, err := system.repository.ReplayIntake(ctx, application.IntakeReplayRequest{
		RequestRef:         first.Record.Receipt.RequestRef,
		RequestFingerprint: first.Record.Receipt.RequestFingerprint,
		Operation:          first.Record.Receipt.Operation,
		ActorRef:           system.principal.ActorRef, ProjectRef: system.project, StateRef: system.stateRef,
		AuthorizationReceiptRef: first.Record.Receipt.AuthorizationReceiptRef,
	})
	if err != nil || !found ||
		!reflect.DeepEqual(application.SnapshotIntake(historical.State), application.SnapshotIntake(first.Record.State)) ||
		historical.Receipt != first.Record.Receipt {
		t.Fatalf("historical replay mismatch: found=%v receipt=%+v err=%v", found, historical.Receipt, err)
	}
}

func TestIntakeSQLiteCASRollsBackStaleReceipt(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	create := system.create(t, "request:intake:create-cas")
	first := system.apply(t, create.Record.State, "request:intake:first-cas", intake.OriginChat,
		"intake-option:audience-personal")
	_ = system.apply(t, first.Record.State, "request:intake:second-cas", intake.OriginForm,
		"intake-option:audience-team")

	staleState, err := intake.Apply(first.Record.State, intake.Change{
		StateRef: system.stateRef, ExpectedRevision: first.Record.State.Revision(),
		Origin:  intake.OriginChat,
		Choices: []intake.Choice{{QuestionRef: intake.QuestionRef("intake-question:audience"), OptionRef: intake.OptionRef("intake-option:audience-personal")}},
	})
	sqliteTestNoError(t, err)
	staleDigest, err := application.IntakeStateDigest(staleState)
	sqliteTestNoError(t, err)
	authorization := system.authorizeIntake(
		t, application.IntakeOperationApply, "request:intake:stale",
	)
	requestFingerprint := strings.Repeat("a", 64)
	receipt := application.IntakeReceipt{
		Ref:        "intake-receipt:" + strings.Repeat("b", 64),
		RequestRef: "request:intake:stale", RequestFingerprint: requestFingerprint,
		Operation: application.IntakeOperationApply,
		ActorRef:  system.principal.ActorRef, ProjectRef: system.project, StateRef: system.stateRef,
		PreviousRevision: first.Record.State.Revision(), Revision: staleState.Revision(),
		StateDigest: staleDigest, AuthorizationReceiptRef: authorization.Ref(),
	}
	before := intakeReceiptCount(t, system.repository)
	_, changed, err := system.repository.ApplyIntake(ctx, application.IntakeApplyState{
		RequestRef: receipt.RequestRef, RequestFingerprint: requestFingerprint,
		ActorRef: system.principal.ActorRef, ProjectRef: system.project,
		AuthorizationReceipt: authorization,
		ExpectedRevision:     first.Record.State.Revision(), State: staleState, Receipt: receipt,
	})
	if !application.IsStateError(err, application.StateConflict) || changed {
		t.Fatalf("stale CAS = changed:%v err:%v", changed, err)
	}
	if after := intakeReceiptCount(t, system.repository); after != before {
		t.Fatalf("stale CAS leaked receipt: before=%d after=%d", before, after)
	}
	if _, found, replayErr := system.repository.ReplayIntake(ctx, application.IntakeReplayRequest{
		RequestRef: receipt.RequestRef, RequestFingerprint: receipt.RequestFingerprint,
		Operation: receipt.Operation, ActorRef: receipt.ActorRef, ProjectRef: receipt.ProjectRef,
		StateRef: receipt.StateRef, AuthorizationReceiptRef: receipt.AuthorizationReceiptRef,
	}); replayErr != nil || found {
		t.Fatalf("rolled-back request replay = found:%v err:%v", found, replayErr)
	}
}

func TestIntakeSQLiteDivergentReplayAndScopeIsolation(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:intake:scope")
	receipt := created.Record.Receipt

	_, found, err := system.repository.ReplayIntake(ctx, application.IntakeReplayRequest{
		RequestRef: receipt.RequestRef, RequestFingerprint: strings.Repeat("f", 64),
		Operation: receipt.Operation, ActorRef: receipt.ActorRef, ProjectRef: receipt.ProjectRef,
		StateRef: receipt.StateRef, AuthorizationReceiptRef: receipt.AuthorizationReceiptRef,
	})
	if !application.IsStateError(err, application.StateConflict) || found {
		t.Fatalf("divergent replay = found:%v err:%v", found, err)
	}
	otherProject := mustRef(t, "project:intake-other", goal.NewProjectRef)
	otherPrincipal := testPrincipal(
		t, "principal:intake-other", "actor:intake-other", identity.PrincipalKindHuman,
	)
	provisionTestAccess(
		t, system.repository, otherPrincipal, otherProject, identity.RoleProjectOwner, system.now,
	)
	otherAuthorizationRequestRef, err := application.IntakeAuthorizationRequestRef(
		application.IntakeOperationCreate, "request:intake:other-scope",
	)
	sqliteTestNoError(t, err)
	otherAuthorization := authorizeTest(
		t, system.repository, otherPrincipal, otherProject, identity.PermissionGoalsCreate,
		otherProject.String(), otherAuthorizationRequestRef, system.now,
	)
	otherService, err := application.NewIntakeService(system.repository)
	sqliteTestNoError(t, err)
	other, err := otherService.CreateIntake(ctx, application.CreateIntakeRequest{
		RequestRef: "request:intake:other-scope",
		ActorRef:   otherPrincipal.ActorRef, ProjectRef: otherProject,
		StateRef: system.stateRef, Policy: intake.Policy{MaxQuestionRounds: 2},
		AuthorizationReceipt: otherAuthorization,
	})
	if err != nil || !other.Changed || other.Record.State.Ref() != system.stateRef {
		t.Fatalf("same ref in other scope = changed:%v receipt=%+v err=%v",
			other.Changed, other.Record.Receipt, err)
	}
	if _, err := system.repository.GetIntake(
		ctx, system.principal.ActorRef, otherProject, system.stateRef,
	); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("cross-project get = %v", err)
	}
	if _, err := system.repository.GetIntake(
		ctx, otherPrincipal.ActorRef, system.project, system.stateRef,
	); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("cross-actor get = %v", err)
	}
}

func TestIntakeSQLiteConcurrentCASAdmitsOneReceipt(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:intake:race-create")
	change := sqliteIntakeAudienceChange(
		created.Record.State, system.stateRef, intake.OriginChat,
		intake.OptionRef("intake-option:audience-personal"),
	)
	authorizations := []identity.AuthorizationReceipt{
		system.authorizeIntake(
			t, application.IntakeOperationApply, "request:intake:race-a",
		),
		system.authorizeIntake(
			t, application.IntakeOperationApply, "request:intake:race-b",
		),
	}
	requests := make([]application.ApplyIntakeRequest, 2)
	for index := range requests {
		suffix := string(rune('a' + index))
		requests[index] = application.ApplyIntakeRequest{
			RequestRef: "request:intake:race-" + suffix,
			ActorRef:   system.principal.ActorRef, ProjectRef: system.project,
			AuthorizationReceipt: authorizations[index],
			Change:               change,
		}
	}
	type outcome struct {
		changed bool
		err     error
	}
	start := make(chan struct{})
	outcomes := make(chan outcome, len(requests))
	for _, request := range requests {
		request := request
		go func() {
			<-start
			result, err := system.service.ApplyIntake(ctx, request)
			outcomes <- outcome{changed: result.Changed, err: err}
		}()
	}
	close(start)
	var accepted, conflicts int
	for range requests {
		result := <-outcomes
		switch {
		case result.err == nil && result.changed:
			accepted++
		case !result.changed &&
			(application.IsStateError(result.err, application.StateConflict) ||
				intake.ErrorCodeOf(result.err) == intake.ErrorRevisionConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent CAS outcome: changed=%v err=%v", result.changed, result.err)
		}
	}
	if accepted != 1 || conflicts != 1 {
		t.Fatalf("concurrent CAS accepted=%d conflicts=%d", accepted, conflicts)
	}
	if count := intakeReceiptCount(t, system.repository); count != 2 {
		t.Fatalf("concurrent loser leaked receipt: count=%d", count)
	}
	transaction, err := system.repository.db.BeginTx(ctx, nil)
	sqliteTestNoError(t, err)
	if recoveryErr := validateRecoveryV23Intake(ctx, transaction); recoveryErr != nil {
		_ = transaction.Rollback()
		t.Fatalf("concurrent winner left invalid history: %s", intakeTestErrorChain(recoveryErr))
	}
	sqliteTestNoError(t, transaction.Rollback())
}

func TestIntakeSQLiteRejectsStaleAuthorizationWithoutReceipt(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:intake:auth-create")
	next, err := intake.Apply(created.Record.State, intake.Change{
		StateRef: system.stateRef, ExpectedRevision: 1, Origin: intake.OriginChat,
		Issues: []intake.Issue{{
			Ref: intake.IssueRef("intake-issue:auth"), Kind: intake.IssueGap,
			Field: "auth", DetailKey: intake.MessageKey("intake.issue.auth.missing"),
		}},
		Questions: []intake.Question{{
			Ref:         intake.QuestionRef("intake-question:auth"),
			DerivedFrom: []intake.IssueRef{intake.IssueRef("intake-issue:auth")},
			PromptKey:   intake.MessageKey("intake.question.auth.prompt"),
			WhyKey:      intake.MessageKey("intake.question.auth.why"),
			Options: []intake.Option{{
				Ref:          intake.OptionRef("intake-option:auth-safe"),
				LabelKey:     intake.MessageKey("intake.option.auth.safe.label"),
				RationaleKey: intake.MessageKey("intake.option.auth.safe.rationale"),
				Recommended:  true,
			}},
		}},
	})
	sqliteTestNoError(t, err)
	authorization := system.authorizeIntake(
		t, application.IntakeOperationApply, "request:intake:after-revoke",
	)
	digest, err := application.IntakeStateDigest(next)
	sqliteTestNoError(t, err)
	fingerprint := strings.Repeat("e", 64)
	receipt := application.IntakeReceipt{
		Ref:        "intake-receipt:" + strings.Repeat("f", 64),
		RequestRef: "request:intake:after-revoke", RequestFingerprint: fingerprint,
		Operation: application.IntakeOperationApply,
		ActorRef:  system.principal.ActorRef, ProjectRef: system.project, StateRef: system.stateRef,
		PreviousRevision: 1, Revision: 2, StateDigest: digest,
		AuthorizationReceiptRef: authorization.Ref(),
	}
	_, err = system.repository.db.ExecContext(ctx, `
UPDATE project_memberships
SET revision = revision + 1, status = 'revoked',
    revoked_by_ref = ?, revoked_at = ?
WHERE principal_ref = ? AND project_ref = ?`,
		system.principal.Ref.String(), system.now.Add(time.Minute).UnixNano(),
		system.principal.Ref.String(), system.project.String(),
	)
	sqliteTestNoError(t, err)
	transaction, err := system.repository.db.BeginTx(ctx, nil)
	sqliteTestNoError(t, err)
	if recoveryErr := validateRecoveryV23Intake(ctx, transaction); recoveryErr != nil {
		_ = transaction.Rollback()
		t.Fatalf("historical authorization after membership revocation: %s",
			intakeTestErrorChain(recoveryErr))
	}
	sqliteTestNoError(t, transaction.Rollback())
	before := intakeReceiptCount(t, system.repository)
	_, changed, err := system.repository.ApplyIntake(ctx, application.IntakeApplyState{
		RequestRef: receipt.RequestRef, RequestFingerprint: receipt.RequestFingerprint,
		ActorRef: system.principal.ActorRef, ProjectRef: system.project,
		AuthorizationReceipt: authorization,
		ExpectedRevision:     1, State: next, Receipt: receipt,
	})
	if !application.IsStateError(err, application.StateConflict) || changed {
		t.Fatalf("stale authorization = changed:%v err:%v", changed, err)
	}
	if count := intakeReceiptCount(t, system.repository); count != before {
		t.Fatalf("stale authorization leaked receipt: before=%d after=%d", before, count)
	}
}

func TestIntakeSQLiteSnapshotCodec(t *testing.T) {
	state, err := intake.NewState("intake:codec", intake.Policy{MaxQuestionRounds: 4})
	sqliteTestNoError(t, err)
	encoded, err := encodeIntakeState(state)
	if err != nil {
		t.Fatalf("encode snapshot: %v cause=%v", err, errors.Unwrap(err))
	}
	digest, err := application.IntakeStateDigest(state)
	sqliteTestNoError(t, err)
	if got := intakeStateDigest(encoded); got != digest {
		t.Fatalf("snapshot digest = %s, want %s", got, digest)
	}
	var snapshot application.IntakeSnapshot
	sqliteTestNoError(t, json.Unmarshal(encoded, &snapshot))
	if _, err := application.RestoreIntake(snapshot); err != nil {
		t.Fatalf("restore encoded snapshot: %s json=%s", intakeTestErrorChain(err), encoded)
	}
}

func TestIntakeSQLiteRejectsNonCausalAuthorizationWithoutRows(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	const requestRef = "request:intake:causal"
	authorization := system.authorizeIntake(
		t, application.IntakeOperationCreate, "request:intake:foreign",
	)
	state, err := intake.NewState(system.stateRef, intake.Policy{MaxQuestionRounds: 4})
	sqliteTestNoError(t, err)
	digest, err := application.IntakeStateDigest(state)
	sqliteTestNoError(t, err)
	fingerprint := canonicalFingerprint(
		"orquesta.intake.create.v1", system.principal.ActorRef.String(),
		system.project.String(), string(system.stateRef), "4", authorization.Ref(),
	)
	receipt := application.IntakeReceipt{
		RequestRef: requestRef, RequestFingerprint: fingerprint,
		Operation: application.IntakeOperationCreate,
		ActorRef:  system.principal.ActorRef, ProjectRef: system.project,
		StateRef: system.stateRef, Revision: state.Revision(),
		StateDigest: digest, AuthorizationReceiptRef: authorization.Ref(),
	}
	receipt.Ref = v23TestIntakeReceiptRef(receipt)
	_, changed, err := system.repository.CreateIntake(ctx, application.IntakeCreateState{
		RequestRef: requestRef, RequestFingerprint: fingerprint,
		ActorRef: system.principal.ActorRef, ProjectRef: system.project,
		AuthorizationReceipt: authorization, State: state, Receipt: receipt,
	})
	if err == nil || changed {
		t.Fatalf("non-causal authorization = changed:%v err:%v", changed, err)
	}
	var states, receipts int
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM intake_states`,
	).Scan(&states))
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM intake_receipts`,
	).Scan(&receipts))
	if states != 0 || receipts != 0 {
		t.Fatalf("non-causal authorization leaked state/receipt = %d/%d", states, receipts)
	}
}

func TestIntakeSQLiteRejectsForgedFingerprintAndReceiptWithoutRows(t *testing.T) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	const requestRef = "request:intake:forged-create"
	authorization := system.authorizeIntake(
		t, application.IntakeOperationCreate, requestRef,
	)
	state, err := intake.NewState(system.stateRef, intake.Policy{MaxQuestionRounds: 4})
	sqliteTestNoError(t, err)
	digest, err := application.IntakeStateDigest(state)
	sqliteTestNoError(t, err)
	receipt := application.IntakeReceipt{
		RequestRef: requestRef, RequestFingerprint: strings.Repeat("9", 64),
		Operation: application.IntakeOperationCreate,
		ActorRef:  system.principal.ActorRef, ProjectRef: system.project,
		StateRef: system.stateRef, PreviousRevision: 0, Revision: state.Revision(),
		StateDigest: digest, AuthorizationReceiptRef: authorization.Ref(),
	}
	receipt.Ref = v23TestIntakeReceiptRef(receipt)
	_, changed, err := system.repository.CreateIntake(ctx, application.IntakeCreateState{
		RequestRef: requestRef, RequestFingerprint: receipt.RequestFingerprint,
		ActorRef: system.principal.ActorRef, ProjectRef: system.project,
		AuthorizationReceipt: authorization, State: state, Receipt: receipt,
	})
	if err == nil || changed {
		t.Fatalf("forged fingerprint/receipt = changed:%v err:%v", changed, err)
	}
	var states, receipts int
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM intake_states`,
	).Scan(&states))
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM intake_receipts`,
	).Scan(&receipts))
	if states != 0 || receipts != 0 {
		t.Fatalf("forged fingerprint/receipt leaked state/receipt = %d/%d", states, receipts)
	}
}

func TestIntakeSQLiteRejectsInvalidRequestRefsWithoutRows(t *testing.T) {
	tests := []string{
		strings.Repeat("a", 513),
		"request:intake:\x00invalid",
		"request:intake:\ninvalid",
		"request:intake:\rinvalid",
	}
	for index, requestRef := range tests {
		t.Run(string(rune('a'+index)), func(t *testing.T) {
			system := newSQLiteIntakeTestSystem(t)
			result, err := system.service.CreateIntake(
				context.Background(),
				application.CreateIntakeRequest{
					RequestRef: requestRef, ActorRef: system.principal.ActorRef,
					ProjectRef: system.project, StateRef: system.stateRef,
					Policy: intake.Policy{MaxQuestionRounds: 4},
				},
			)
			if err == nil || result.Changed {
				t.Fatalf("invalid request ref = changed:%v err:%v", result.Changed, err)
			}
			var states, receipts, authorizations int
			sqliteTestNoError(t, system.repository.db.QueryRow(
				`SELECT COUNT(*) FROM intake_states`,
			).Scan(&states))
			sqliteTestNoError(t, system.repository.db.QueryRow(
				`SELECT COUNT(*) FROM intake_receipts`,
			).Scan(&receipts))
			sqliteTestNoError(t, system.repository.db.QueryRow(
				`SELECT COUNT(*) FROM authorization_receipts`,
			).Scan(&authorizations))
			if states != 0 || receipts != 0 || authorizations != 0 {
				t.Fatalf(
					"invalid request leaked state/receipt/authorization = %d/%d/%d",
					states, receipts, authorizations,
				)
			}
		})
	}
}

type sqliteIntakeTestSystem struct {
	path       string
	repository *Repository
	service    *application.IntakeService
	principal  identity.Principal
	project    goal.ProjectRef
	stateRef   intake.Ref
	now        time.Time
}

func newSQLiteIntakeTestSystem(t *testing.T) *sqliteIntakeTestSystem {
	t.Helper()
	path := t.TempDir() + "/private/state.db"
	repository := openSQLiteIntakeTestRepository(t, path)
	principal := testPrincipal(
		t, "principal:intake-owner", "actor:intake-owner", identity.PrincipalKindHuman,
	)
	project := mustRef(t, "project:intake", goal.NewProjectRef)
	now := time.Date(2026, 7, 26, 9, 0, 0, 0, time.UTC)
	provisionTestAccess(t, repository, principal, project, identity.RoleProjectOwner, now)
	service, err := application.NewIntakeService(repository)
	sqliteTestNoError(t, err)
	return &sqliteIntakeTestSystem{
		path: path, repository: repository, service: service,
		principal: principal, project: project, stateRef: intake.Ref("intake:test"), now: now,
	}
}

func openSQLiteIntakeTestRepository(t *testing.T, path string) *Repository {
	t.Helper()
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
	})
	if err != nil {
		t.Fatalf("open intake repository: %v cause=%v", err, errors.Unwrap(err))
	}
	return repository
}

func (system *sqliteIntakeTestSystem) authorize(
	t *testing.T,
	requestRef string,
) identity.AuthorizationReceipt {
	t.Helper()
	return authorizeTest(
		t, system.repository, system.principal, system.project,
		identity.PermissionGoalsCreate, system.project.String(), requestRef, system.now,
	)
}

func (system *sqliteIntakeTestSystem) authorizeIntake(
	t *testing.T,
	operation application.IntakeOperation,
	requestRef string,
) identity.AuthorizationReceipt {
	t.Helper()
	authorizationRequestRef, err := application.IntakeAuthorizationRequestRef(
		operation, requestRef,
	)
	sqliteTestNoError(t, err)
	return system.authorize(t, authorizationRequestRef)
}

func (system *sqliteIntakeTestSystem) create(
	t *testing.T,
	requestRef string,
) application.IntakeResult {
	t.Helper()
	result, err := system.service.CreateIntake(context.Background(), application.CreateIntakeRequest{
		RequestRef: requestRef, ActorRef: system.principal.ActorRef, ProjectRef: system.project,
		StateRef: system.stateRef, Policy: intake.Policy{MaxQuestionRounds: 4},
		AuthorizationReceipt: system.authorizeIntake(
			t, application.IntakeOperationCreate, requestRef,
		),
	})
	if err != nil {
		t.Fatalf("create intake: %s", intakeTestErrorChain(err))
	}
	if !result.Changed || result.Record.State.Revision() != 1 {
		t.Fatalf("create result = changed:%v revision:%d", result.Changed, result.Record.State.Revision())
	}
	return result
}

func (system *sqliteIntakeTestSystem) apply(
	t *testing.T,
	current intake.State,
	requestRef string,
	origin intake.Origin,
	optionRef intake.OptionRef,
) application.IntakeResult {
	t.Helper()
	change := sqliteIntakeAudienceChange(current, system.stateRef, origin, optionRef)
	result, err := system.service.ApplyIntake(context.Background(), application.ApplyIntakeRequest{
		RequestRef: requestRef, ActorRef: system.principal.ActorRef, ProjectRef: system.project,
		Change: change, AuthorizationReceipt: system.authorizeIntake(
			t, application.IntakeOperationApply, requestRef,
		),
	})
	if err != nil {
		t.Fatalf("apply intake: %s", intakeTestErrorChain(err))
	}
	if !result.Changed || result.Record.State.Revision() != current.Revision()+1 {
		t.Fatalf("apply result = changed:%v revision:%d", result.Changed, result.Record.State.Revision())
	}
	return result
}

func sqliteIntakeAudienceChange(
	current intake.State,
	stateRef intake.Ref,
	origin intake.Origin,
	optionRef intake.OptionRef,
) intake.Change {
	change := intake.Change{
		StateRef: stateRef, ExpectedRevision: current.Revision(), Origin: origin,
		Choices: []intake.Choice{{
			QuestionRef: intake.QuestionRef("intake-question:audience"), OptionRef: optionRef,
		}},
	}
	if current.Revision() == 1 {
		change.Issues = []intake.Issue{{
			Ref: intake.IssueRef("intake-issue:audience"), Kind: intake.IssueGap,
			Field: "audience", DetailKey: intake.MessageKey("intake.issue.audience.missing"),
		}}
		change.Questions = []intake.Question{{
			Ref:         intake.QuestionRef("intake-question:audience"),
			DerivedFrom: []intake.IssueRef{intake.IssueRef("intake-issue:audience")},
			PromptKey:   intake.MessageKey("intake.question.audience.prompt"),
			WhyKey:      intake.MessageKey("intake.question.audience.why"),
			Options: []intake.Option{
				{
					Ref:          intake.OptionRef("intake-option:audience-team"),
					LabelKey:     intake.MessageKey("intake.option.audience.team.label"),
					RationaleKey: intake.MessageKey("intake.option.audience.team.rationale"),
					Recommended:  true,
				},
				{
					Ref:          intake.OptionRef("intake-option:audience-personal"),
					LabelKey:     intake.MessageKey("intake.option.audience.personal.label"),
					RationaleKey: intake.MessageKey("intake.option.audience.personal.rationale"),
				},
			},
		}}
	}
	return change
}

func intakeReceiptCount(t *testing.T, repository *Repository) int {
	t.Helper()
	var count int
	if err := repository.db.QueryRow("SELECT COUNT(*) FROM intake_receipts").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func intakeTestErrorChain(err error) string {
	var values []string
	for err != nil {
		values = append(values, err.Error())
		err = errors.Unwrap(err)
	}
	return strings.Join(values, " -> ")
}
