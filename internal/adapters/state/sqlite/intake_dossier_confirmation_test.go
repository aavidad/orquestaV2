package sqlite

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/governance"
	"orquesta/internal/intake"
)

type sqliteIntakeDossierConfirmationSystem struct {
	*sqliteIntakeDossierTestSystem
	clock          *sqliteMembershipClock
	external       *sqliteV15External
	policy         application.BudgetPolicy
	ids            *sqliteV15IDs
	orchestrator   *application.Orchestrator
	access         application.Access
	dossier        application.IntakeDossierRecord
	egressPolicies application.EgressPolicyResolver
}

var errCaptureIntakeDossierConfirmation = errors.New(
	"test.capture_intake_dossier_confirmation",
)

type captureIntakeDossierConfirmationRepository struct {
	application.StateRepository
	state application.ConfirmIntakeDossierState
}

func (repository *captureIntakeDossierConfirmationRepository) ConfirmIntakeDossierAndCreateGoal(
	_ context.Context,
	state application.ConfirmIntakeDossierState,
) (application.IntakeDossierConfirmationRecord, bool, error) {
	repository.state = state
	return application.IntakeDossierConfirmationRecord{},
		false, errCaptureIntakeDossierConfirmation
}

func newSQLiteIntakeDossierConfirmationSystem(
	t *testing.T,
) *sqliteIntakeDossierConfirmationSystem {
	return newSQLiteIntakeDossierConfirmationSystemWithEgress(
		t, application.EgressPolicyAuthority{},
	)
}

func newSQLiteIntakeDossierConfirmationSystemWithEgress(
	t *testing.T,
	egressPolicy application.EgressPolicyAuthority,
) *sqliteIntakeDossierConfirmationSystem {
	t.Helper()
	base := newSQLiteIntakeDossierTestSystem(t)
	request := base.prepareRequest(t, "request:intake-dossier:confirmation-source")
	if egressPolicy != (application.EgressPolicyAuthority{}) {
		request.Plan.WorkItems[0].EgressPolicyRef = egressPolicy.PolicyRef.String()
	}
	request.Plan.WorkItems[0].BudgetDemand.Resources.Currency =
		governance.Currency("USD")
	prepared, err := base.dossiers.PrepareIntakeDossier(
		context.Background(), request,
	)
	sqliteTestNoError(t, err)
	if !prepared.Created {
		t.Fatal("confirmation dossier was replayed")
	}
	clock := &sqliteMembershipClock{now: base.now.Add(time.Hour)}
	base.repository.now = clock.Now
	external := newSQLiteV15External(clock)
	policy := sqliteTestBudgetPolicy(clock.Now())
	ids := &sqliteV15IDs{}
	access, err := application.NewAccess(base.principal, base.project)
	sqliteTestNoError(t, err)
	system := &sqliteIntakeDossierConfirmationSystem{
		sqliteIntakeDossierTestSystem: base,
		clock:                         clock,
		external:                      external,
		policy:                        policy,
		ids:                           ids,
		access:                        access,
		dossier:                       prepared.Record,
		egressPolicies: func() application.EgressPolicyResolver {
			if egressPolicy == (application.EgressPolicyAuthority{}) {
				return nil
			}
			return sqliteEgressPolicyResolver{authority: egressPolicy}
		}(),
	}
	system.orchestrator = system.newOrchestrator(t, base.repository)
	return system
}

func (system *sqliteIntakeDossierConfirmationSystem) newOrchestrator(
	t *testing.T,
	state application.StateRepository,
) *application.Orchestrator {
	t.Helper()
	_, fuentes := prepararCapacidadSQLiteV15(t, system.repository, system.clock, 1_000)
	orchestrator, err := application.New(application.Dependencies{
		State: state, WizardGapsStore: system.repository,
		IntakeDossierStore: system.repository,
		Access:             system.repository, Launcher: system.external,
		Observer:   system.external,
		Controller: system.external, Artifacts: system.external,
		Clock: system.clock, IDs: system.ids,
		MaxOutputBytes: 1024, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, MaxChildrenPerParent: 6,
		ClaimLease: time.Minute, DirectorLeaseDuration: 30 * time.Second,
		EffectApprovalTTL: system.policy.EffectApprovalTTL,
		BudgetPolicy:      system.policy,
		ObservationDelay:  time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: sqliteTestCapabilities(),
		CapacitySources:   fuentes, CapacityObservationWait: time.Second,
		EgressPolicies: system.egressPolicies,
	})
	sqliteTestNoError(t, err)
	return orchestrator
}

func (system *sqliteIntakeDossierConfirmationSystem) confirm(
	requestRef string,
) (application.ConfirmIntakeDossierResult, error) {
	return system.orchestrator.ConfirmIntakeDossier(
		context.Background(), system.access,
		application.ConfirmIntakeDossierRequest{
			RequestRef: requestRef, DossierRef: system.dossier.Dossier.Ref(),
			Confirm: true,
		},
	)
}

func TestIntakeDossierConfirmationSQLiteAtomicReplayRestartAndFreeze(
	t *testing.T,
) {
	system := newSQLiteIntakeDossierConfirmationSystem(t)
	requestRef := "request:intake-dossier:confirm-restart"
	first, err := system.confirm(requestRef)
	sqliteTestNoError(t, err)
	if !first.Created || first.Confirmation.GoalRef != first.Record.Goal.Ref() {
		t.Fatalf("first confirmation=%+v", first)
	}
	if got := tableCount(
		t, system.repository, "intake_dossier_confirmations",
	); got != 1 {
		t.Fatalf("confirmation rows=%d", got)
	}
	if got := tableCount(t, system.repository, "goals"); got != 1 {
		t.Fatalf("goal rows=%d", got)
	}
	if got := tableCount(t, system.repository, "outbox"); got == 0 {
		t.Fatal("confirmation created no causal outbox")
	}

	replayed, err := system.confirm(requestRef)
	sqliteTestNoError(t, err)
	assertSameIntakeDossierConfirmation(t, first, replayed, false)
	system.assertFrozen(t)

	path := system.path
	sqliteTestNoError(t, system.repository.Close())
	restarted := openSQLiteIntakeTestRepository(t, path)
	t.Cleanup(func() { _ = restarted.Close() })
	system.repository = restarted
	system.service, err = application.NewIntakeService(restarted)
	sqliteTestNoError(t, err)
	system.dossiers, err = application.NewIntakeDossierService(restarted, restarted)
	sqliteTestNoError(t, err)
	system.ids = &sqliteV15IDs{next: 100}
	system.orchestrator = system.newOrchestrator(t, restarted)

	afterRestart, err := system.confirm(requestRef)
	sqliteTestNoError(t, err)
	assertSameIntakeDossierConfirmation(t, first, afterRestart, false)
	system.assertFrozen(t)
}

func TestIntakeDossierConfirmationEgressReplayUsesHistoricalAuthorityWithRetiredResolver(
	t *testing.T,
) {
	authority := sqliteEgressAuthority(
		t, "egress-policy:dossier-sqlite-replay", `{"revision":1}`,
	)
	system := newSQLiteIntakeDossierConfirmationSystemWithEgress(t, authority)
	const requestRef = "request:intake-dossier:confirm-egress-replay"
	created, err := system.confirm(requestRef)
	sqliteTestNoError(t, err)
	if !created.Created || len(created.Record.WorkItemAuthorities) == 0 ||
		created.Record.WorkItemAuthorities[0].EgressPolicy != authority {
		t.Fatalf("created egress confirmation=%+v", created)
	}

	system.egressPolicies = nil
	system.orchestrator = system.newOrchestrator(t, system.repository)
	replayed, err := system.confirm(requestRef)
	sqliteTestNoError(t, err)
	if replayed.Created || replayed.Confirmation != created.Confirmation ||
		len(replayed.Record.WorkItemAuthorities) == 0 ||
		replayed.Record.WorkItemAuthorities[0].EgressPolicy != authority {
		t.Fatalf("historical egress confirmation replay=%+v", replayed)
	}
}

func TestIntakeDossierConfirmationSQLiteConcurrentExactReplayAndDivergence(
	t *testing.T,
) {
	t.Run("same request", func(t *testing.T) {
		system := newSQLiteIntakeDossierConfirmationSystem(t)
		const workers = 8
		results := make([]application.ConfirmIntakeDossierResult, workers)
		errs := make([]error, workers)
		var wait sync.WaitGroup
		for index := range workers {
			wait.Add(1)
			go func(index int) {
				defer wait.Done()
				results[index], errs[index] = system.confirm(
					"request:intake-dossier:confirm-race",
				)
			}(index)
		}
		wait.Wait()
		created := 0
		for index, err := range errs {
			if err != nil {
				t.Fatalf("worker %d: %s", index, intakeTestErrorChain(err))
			}
			if results[index].Created {
				created++
			}
			assertSameIntakeDossierConfirmation(
				t, results[0], results[index], results[index].Created,
			)
		}
		if created != 1 || tableCount(
			t, system.repository, "intake_dossier_confirmations",
		) != 1 || tableCount(t, system.repository, "goals") != 1 {
			t.Fatalf("concurrent exact replay created=%d", created)
		}
	})

	t.Run("different requests", func(t *testing.T) {
		system := newSQLiteIntakeDossierConfirmationSystem(t)
		var wait sync.WaitGroup
		errs := make([]error, 2)
		for index := range errs {
			wait.Add(1)
			go func(index int) {
				defer wait.Done()
				_, errs[index] = system.confirm(
					"request:intake-dossier:confirm-divergent-" + string(rune('a'+index)),
				)
			}(index)
		}
		wait.Wait()
		successes, conflicts := 0, 0
		for _, err := range errs {
			switch {
			case err == nil:
				successes++
			case application.IsStateError(err, application.StateConflict):
				conflicts++
			default:
				t.Fatalf("divergent confirmation: %s", intakeTestErrorChain(err))
			}
		}
		if successes != 1 || conflicts != 1 ||
			tableCount(t, system.repository, "goals") != 1 ||
			tableCount(
				t, system.repository, "intake_dossier_confirmations",
			) != 1 {
			t.Fatalf(
				"divergent results success=%d conflict=%d", successes, conflicts,
			)
		}
	})
}

func TestIntakeDossierConfirmationSQLiteRollsBackGoalAndOutboxOnReceiptFailure(
	t *testing.T,
) {
	system := newSQLiteIntakeDossierConfirmationSystem(t)
	_, err := system.repository.db.Exec(`
CREATE TRIGGER test_confirmation_abort
BEFORE INSERT ON intake_dossier_confirmations
BEGIN SELECT RAISE(ABORT, 'test.confirmation_abort'); END`)
	sqliteTestNoError(t, err)
	requestRef := "request:intake-dossier:confirm-rollback"
	if _, err := system.confirm(requestRef); !application.IsStateError(
		err, application.StateConflict,
	) {
		t.Fatalf("injected confirmation failure=%s", intakeTestErrorChain(err))
	}
	for _, table := range []string{
		"intake_dossier_confirmations", "goals", "outbox",
	} {
		if got := tableCount(t, system.repository, table); got != 0 {
			t.Fatalf("%s escaped rollback: %d", table, got)
		}
	}
	_, err = system.repository.db.Exec(`DROP TRIGGER test_confirmation_abort`)
	sqliteTestNoError(t, err)
	retried, err := system.confirm(requestRef)
	sqliteTestNoError(t, err)
	if !retried.Created {
		t.Fatal("rollback retry was not created")
	}
}

func TestIntakeDossierConfirmationSQLiteRejectsAdvancedIntakeAtomically(
	t *testing.T,
) {
	system := newSQLiteIntakeDossierConfirmationSystem(t)
	advanced := system.apply(
		t, system.current.State, "request:intake-dossier:advance-before-confirm",
		intake.OriginChat, intake.OptionRef("intake-option:audience-personal"),
	)
	if advanced.Record.State.Revision() <= system.dossier.Dossier.StateRevision() {
		t.Fatal("test did not advance intake past dossier")
	}
	_, err := system.confirm("request:intake-dossier:confirm-stale")
	if !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("stale dossier confirmation=%s", intakeTestErrorChain(err))
	}
	for _, table := range []string{
		"intake_dossier_confirmations", "goals", "outbox",
	} {
		if got := tableCount(t, system.repository, table); got != 0 {
			t.Fatalf("%s escaped stale-source rollback: %d", table, got)
		}
	}
}

func TestIntakeDossierConfirmationSQLiteRejectsSpoofedAuthorization(
	t *testing.T,
) {
	system := newSQLiteIntakeDossierConfirmationSystem(t)
	capture := &captureIntakeDossierConfirmationRepository{
		StateRepository: system.repository,
	}
	system.orchestrator = system.newOrchestrator(t, capture)
	_, err := system.confirm("request:intake-dossier:confirm-auth-spoof")
	if !errors.Is(err, errCaptureIntakeDossierConfirmation) {
		t.Fatalf("capture state=%s", intakeTestErrorChain(err))
	}
	candidate := capture.state
	candidate.CreateGoal.AuthorizationReceipt = system.authorize(
		t, "authorization-request:intake-dossier:unrelated",
	)
	if _, _, err := system.repository.ConfirmIntakeDossierAndCreateGoal(
		context.Background(), candidate,
	); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("spoofed authorization=%s", intakeTestErrorChain(err))
	}
	for _, table := range []string{
		"intake_dossier_confirmations", "goals", "outbox",
	} {
		if got := tableCount(t, system.repository, table); got != 0 {
			t.Fatalf("%s escaped spoof rejection: %d", table, got)
		}
	}
}

func (system *sqliteIntakeDossierConfirmationSystem) assertFrozen(t *testing.T) {
	t.Helper()
	applyRequest := "request:intake-dossier:frozen-apply"
	change := sqliteIntakeAudienceChange(
		system.current.State, system.stateRef, intake.OriginChat,
		intake.OptionRef("intake-option:audience-personal"),
	)
	_, err := system.service.ApplyIntake(
		context.Background(),
		application.ApplyIntakeRequest{
			RequestRef: applyRequest, ActorRef: system.principal.ActorRef,
			ProjectRef: system.project, Change: change,
			AuthorizationReceipt: system.authorizeIntake(
				t, application.IntakeOperationApply, applyRequest,
			),
		},
	)
	if !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("confirmed intake remained mutable: %s", intakeTestErrorChain(err))
	}

	prepare := system.prepareRequest(t, "request:intake-dossier:frozen-prepare")
	prepare.Input.Objective += " alternativo"
	_, err = system.dossiers.PrepareIntakeDossier(context.Background(), prepare)
	if !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("confirmed dossier accepted mutation: %s", intakeTestErrorChain(err))
	}
}

func assertSameIntakeDossierConfirmation(
	t *testing.T,
	want application.ConfirmIntakeDossierResult,
	got application.ConfirmIntakeDossierResult,
	created bool,
) {
	t.Helper()
	if got.Created != created ||
		got.Confirmation != want.Confirmation ||
		!reflect.DeepEqual(got.Record.Goal.Snapshot(), want.Record.Goal.Snapshot()) ||
		!reflect.DeepEqual(got.Record.Executions, want.Record.Executions) {
		t.Fatal("confirmation replay changed durable result")
	}
}

func TestV23DossierConfirmationRecoveryRejectsTamperedBinding(t *testing.T) {
	system := newSQLiteIntakeDossierConfirmationSystem(t)
	confirmed, err := system.confirm("request:intake-dossier:confirm-recovery")
	sqliteTestNoError(t, err)
	rewriteRecoveryTrigger(
		t, system.repository.db,
		"intake_dossier_confirmations_immutable_update",
		func() {
			_, err := system.repository.db.Exec(`
UPDATE intake_dossier_confirmations
SET plan_digest=state_digest
WHERE ref=?`, confirmed.Confirmation.Ref)
			sqliteTestNoError(t, err)
		},
	)
	if _, _, err := validateRecoveryDatabase(
		context.Background(), system.repository.db,
	); err == nil || !recoveryErrorContains(
		err, "sqlite.recovery_v23_intake_dossier_confirmation_binding_invalid",
	) {
		t.Fatalf("tampered confirmation passed recovery: %s", intakeTestErrorChain(err))
	}
}

func TestV23DossierConfirmationRecoveryAcceptsLivePlanGeneration(t *testing.T) {
	system := newSQLiteIntakeDossierConfirmationSystem(t)
	confirmed, err := system.confirm(
		"request:intake-dossier:confirm-recovery-generation",
	)
	sqliteTestNoError(t, err)
	_, err = system.repository.db.Exec(
		`UPDATE goals SET plan_generation=2, revision=revision+1 WHERE ref=?`,
		confirmed.Confirmation.GoalRef.String(),
	)
	sqliteTestNoError(t, err)
	if _, _, err := validateRecoveryDatabase(
		context.Background(), system.repository.db,
	); err != nil {
		t.Fatalf("live plan generation failed recovery: %s", intakeTestErrorChain(err))
	}
}

func TestV23DossierConfirmationRecoveryValidatesSemanticReceipt(t *testing.T) {
	system := newSQLiteIntakeDossierConfirmationSystem(t)
	confirmed, err := system.confirm(
		"request:intake-dossier:confirm-recovery-semantic",
	)
	sqliteTestNoError(t, err)
	rewriteRecoveryTrigger(
		t, system.repository.db,
		"intake_dossier_confirmations_immutable_update",
		func() {
			tamperedRef := confirmed.Confirmation.Ref[:len(confirmed.Confirmation.Ref)-1]
			if confirmed.Confirmation.Ref[len(confirmed.Confirmation.Ref)-1] == '0' {
				tamperedRef += "1"
			} else {
				tamperedRef += "0"
			}
			_, err := system.repository.db.Exec(`
UPDATE intake_dossier_confirmations
SET ref=?
WHERE ref=?`, tamperedRef, confirmed.Confirmation.Ref)
			sqliteTestNoError(t, err)
		},
	)
	if _, _, err := validateRecoveryDatabase(
		context.Background(), system.repository.db,
	); err == nil || !recoveryErrorContains(
		err, "sqlite.recovery_v23_intake_dossier_confirmation_semantic_invalid",
	) {
		t.Fatalf(
			"semantically invalid receipt passed recovery: %s",
			intakeTestErrorChain(err),
		)
	}
}
