package sqlite

import (
	"context"
	"reflect"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/intake"
	"orquesta/internal/wizard/gaps"
)

func TestWizardGapsSQLiteNoOpOutcomeReplaysHistoricalStateAfterRestart(
	t *testing.T,
) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:wizard-gaps-sqlite-create")
	service, err := application.NewWizardGapsService(
		system.service, system.repository,
	)
	sqliteTestNoError(t, err)
	request := func(requestRef string, revision intake.Revision) application.ApplyWizardGapsRequest {
		return application.ApplyWizardGapsRequest{
			RequestRef: requestRef, ActorRef: system.principal.ActorRef,
			ProjectRef: system.project, StateRef: system.stateRef,
			ExpectedRevision: revision, Origin: intake.OriginForm,
			EvaluatorIdentity: gaps.EvaluatorV1Identity(),
			AuthorizationReceipt: system.authorizeIntake(
				t, application.IntakeOperationApply, requestRef,
			),
		}
	}
	seed, err := service.ApplyWizardGaps(
		ctx,
		request("request:wizard-gaps-sqlite-seed", created.Record.State.Revision()),
	)
	sqliteTestNoError(t, err)
	const noOpRequestRef = "request:wizard-gaps-sqlite-noop"
	noOpRequest := request(noOpRequestRef, seed.Record.State.Revision())
	noOp, err := service.ApplyWizardGaps(ctx, noOpRequest)
	sqliteTestNoError(t, err)
	if noOp.Changed || !noOp.RequestRefReserved ||
		noOp.EvaluationReplayExact ||
		noOp.RequestOutcome.Kind !=
			application.WizardGapsRequestOutcomeNoOp ||
		noOp.RequestOutcome.ReceiptRef == "" ||
		noOp.RequestOutcome.ReceiptRef == noOp.Record.Receipt.Ref ||
		noOp.Record.Receipt != seed.Record.Receipt {
		t.Fatalf("no-op=%+v seed=%+v", noOp, seed)
	}
	var outcomeCount, mutationCount int
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM wizard_gaps_outcomes WHERE request_ref=?`,
		noOpRequestRef,
	).Scan(&outcomeCount))
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM intake_receipts WHERE request_ref=?`,
		noOpRequestRef,
	).Scan(&mutationCount))
	if outcomeCount != 1 || mutationCount != 0 {
		t.Fatalf("outcomes=%d mutation_receipts=%d", outcomeCount, mutationCount)
	}

	laterRequestRef := "request:wizard-gaps-sqlite-later"
	later, err := system.service.ApplyIntake(ctx, application.ApplyIntakeRequest{
		RequestRef: laterRequestRef, ActorRef: system.principal.ActorRef,
		ProjectRef: system.project,
		Change: intake.Change{
			StateRef: system.stateRef, ExpectedRevision: seed.Record.State.Revision(),
			Origin: intake.OriginForm,
			Choices: []intake.Choice{{
				QuestionRef: "intake-question:wizard.u1",
				OptionRef:   "intake-option:wizard.u1.team",
			}},
		},
		AuthorizationReceipt: system.authorizeIntake(
			t, application.IntakeOperationApply, laterRequestRef,
		),
	})
	sqliteTestNoError(t, err)
	if later.Record.State.Revision() != noOp.Record.State.Revision()+1 {
		t.Fatalf("later revision=%d", later.Record.State.Revision())
	}
	if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err != nil {
		t.Fatalf("recovery validation: %s", intakeTestErrorChain(err))
	}
	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteIntakeTestRepository(t, system.path)
	system.service, err = application.NewIntakeService(system.repository)
	sqliteTestNoError(t, err)
	service, err = application.NewWizardGapsService(
		system.service, system.repository,
	)
	sqliteTestNoError(t, err)

	replayed, err := service.ApplyWizardGaps(ctx, noOpRequest)
	sqliteTestNoError(t, err)
	if replayed.Changed || !replayed.RequestRefReserved ||
		replayed.EvaluationReplayExact ||
		replayed.RequestOutcome != noOp.RequestOutcome ||
		replayed.Record.Receipt != noOp.Record.Receipt ||
		!reflect.DeepEqual(replayed.Evaluation, noOp.Evaluation) {
		t.Fatalf("replayed=%+v no-op=%+v", replayed, noOp)
	}

	divergent := noOpRequest
	divergent.ExpectedRevision = later.Record.State.Revision()
	if _, err := service.ApplyWizardGaps(
		ctx, divergent,
	); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("divergent replay err=%v", err)
	}
}

func TestWizardGapsSQLiteConcurrentExactNoOpReservesOneOutcome(t *testing.T) {
	ctx := context.Background()
	system, service, seed := newSQLiteWizardGapsNoOpTestSystem(t)
	request := sqliteWizardGapsRequest(
		t,
		system,
		"request:wizard-gaps-sqlite-race-exact",
		seed.Record.State.Revision(),
		intake.OriginForm,
	)
	type callResult struct {
		result application.ApplyWizardGapsResult
		err    error
	}
	start := make(chan struct{})
	results := make(chan callResult, 2)
	for range 2 {
		go func() {
			<-start
			result, err := service.ApplyWizardGaps(ctx, request)
			results <- callResult{result: result, err: err}
		}()
	}
	close(start)
	var outcomes []application.WizardGapsRequestOutcome
	for range 2 {
		result := <-results
		if result.err != nil ||
			result.result.Changed ||
			result.result.RequestOutcome.Kind !=
				application.WizardGapsRequestOutcomeNoOp {
			t.Fatalf("result=%+v err=%v", result.result, result.err)
		}
		outcomes = append(outcomes, result.result.RequestOutcome)
	}
	if !reflect.DeepEqual(outcomes[0], outcomes[1]) {
		t.Fatalf("outcomes=%+v", outcomes)
	}
	assertSQLiteWizardGapsRequestEffects(
		t, system, request.RequestRef, 1, 0,
	)
}

func TestWizardGapsSQLiteConcurrentDivergentNoOpAdmitsOnePayload(t *testing.T) {
	ctx := context.Background()
	system, service, seed := newSQLiteWizardGapsNoOpTestSystem(t)
	request := sqliteWizardGapsRequest(
		t,
		system,
		"request:wizard-gaps-sqlite-race-divergent",
		seed.Record.State.Revision(),
		intake.OriginForm,
	)
	divergent := request
	divergent.Origin = intake.OriginChat
	requests := []application.ApplyWizardGapsRequest{request, divergent}
	type callResult struct {
		result application.ApplyWizardGapsResult
		err    error
	}
	start := make(chan struct{})
	results := make(chan callResult, len(requests))
	for _, candidate := range requests {
		candidate := candidate
		go func() {
			<-start
			result, err := service.ApplyWizardGaps(ctx, candidate)
			results <- callResult{result: result, err: err}
		}()
	}
	close(start)
	var accepted, conflicts int
	for range requests {
		result := <-results
		switch {
		case result.err == nil &&
			!result.result.Changed &&
			result.result.RequestOutcome.Kind ==
				application.WizardGapsRequestOutcomeNoOp:
			accepted++
		case application.IsStateError(result.err, application.StateConflict):
			conflicts++
		default:
			t.Fatalf("result=%+v err=%v", result.result, result.err)
		}
	}
	if accepted != 1 || conflicts != 1 {
		t.Fatalf("accepted=%d conflicts=%d", accepted, conflicts)
	}
	assertSQLiteWizardGapsRequestEffects(
		t, system, request.RequestRef, 1, 0,
	)
}

func TestWizardGapsSQLiteConcurrentNoOpAndMutationReserveOneEffect(t *testing.T) {
	ctx := context.Background()
	system, service, seed := newSQLiteWizardGapsNoOpTestSystem(t)
	const requestRef = "request:wizard-gaps-sqlite-race-noop-mutation"
	wizardRequest := sqliteWizardGapsRequest(
		t,
		system,
		requestRef,
		seed.Record.State.Revision(),
		intake.OriginForm,
	)
	authorization := wizardRequest.AuthorizationReceipt
	questions := seed.Evaluation.Questions()
	if len(questions) == 0 {
		t.Fatal("seed has no Wizard question")
	}
	option, found := questions[0].RecommendedOption()
	if !found {
		t.Fatal("seed Wizard question has no recommendation")
	}
	mutationRequest := application.ApplyIntakeRequest{
		RequestRef: requestRef, ActorRef: system.principal.ActorRef,
		ProjectRef: system.project,
		Change: intake.Change{
			StateRef:         system.stateRef,
			ExpectedRevision: seed.Record.State.Revision(),
			Origin:           intake.OriginForm,
			Choices: []intake.Choice{{
				QuestionRef: intake.QuestionRef(questions[0].Ref()),
				OptionRef:   intake.OptionRef(option.Ref()),
			}},
		},
		AuthorizationReceipt: authorization,
	}
	type callResult struct {
		kind string
		err  error
	}
	start := make(chan struct{})
	results := make(chan callResult, 2)
	go func() {
		<-start
		result, err := service.ApplyWizardGaps(ctx, wizardRequest)
		kind := ""
		if err == nil {
			kind = string(result.RequestOutcome.Kind)
		}
		results <- callResult{kind: kind, err: err}
	}()
	go func() {
		<-start
		result, err := system.service.ApplyIntake(ctx, mutationRequest)
		kind := ""
		if err == nil && result.Record.Receipt.Ref != "" {
			kind = string(application.WizardGapsRequestOutcomeIntakeMutation)
		}
		results <- callResult{kind: kind, err: err}
	}()
	close(start)
	var accepted, conflicts int
	for range 2 {
		result := <-results
		switch {
		case result.err == nil && result.kind != "":
			accepted++
		case application.IsStateError(result.err, application.StateConflict) ||
			intake.ErrorCodeOf(result.err) == intake.ErrorRevisionConflict:
			conflicts++
		default:
			t.Fatalf("kind=%q err=%v", result.kind, result.err)
		}
	}
	if accepted != 1 || conflicts != 1 {
		t.Fatalf("accepted=%d conflicts=%d", accepted, conflicts)
	}
	var outcomeCount, mutationCount int
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM wizard_gaps_outcomes WHERE request_ref=?`,
		requestRef,
	).Scan(&outcomeCount))
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM intake_receipts WHERE request_ref=?`,
		requestRef,
	).Scan(&mutationCount))
	if outcomeCount+mutationCount != 1 {
		t.Fatalf(
			"outcomes=%d mutation_receipts=%d",
			outcomeCount,
			mutationCount,
		)
	}
	current, err := system.service.GetIntake(ctx, application.GetIntakeRequest{
		ActorRef: system.principal.ActorRef, ProjectRef: system.project,
		StateRef: system.stateRef,
	})
	sqliteTestNoError(t, err)
	wantRevision := seed.Record.State.Revision() + intake.Revision(mutationCount)
	if current.State.Revision() != wantRevision {
		t.Fatalf(
			"revision=%d want=%d outcomes=%d mutations=%d",
			current.State.Revision(),
			wantRevision,
			outcomeCount,
			mutationCount,
		)
	}
	if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err != nil {
		t.Fatalf("recovery: %s", intakeTestErrorChain(err))
	}
}

func newSQLiteWizardGapsNoOpTestSystem(
	t *testing.T,
) (
	*sqliteIntakeTestSystem,
	*application.WizardGapsService,
	application.ApplyWizardGapsResult,
) {
	t.Helper()
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:wizard-gaps-sqlite-race-create")
	service, err := application.NewWizardGapsService(
		system.service, system.repository,
	)
	sqliteTestNoError(t, err)
	seed, err := service.ApplyWizardGaps(
		context.Background(),
		sqliteWizardGapsRequest(
			t,
			system,
			"request:wizard-gaps-sqlite-race-seed",
			created.Record.State.Revision(),
			intake.OriginForm,
		),
	)
	sqliteTestNoError(t, err)
	if !seed.Changed ||
		seed.RequestOutcome.Kind !=
			application.WizardGapsRequestOutcomeIntakeMutation {
		t.Fatalf("seed=%+v", seed)
	}
	return system, service, seed
}

func sqliteWizardGapsRequest(
	t *testing.T,
	system *sqliteIntakeTestSystem,
	requestRef string,
	revision intake.Revision,
	origin intake.Origin,
) application.ApplyWizardGapsRequest {
	t.Helper()
	return application.ApplyWizardGapsRequest{
		RequestRef: requestRef, ActorRef: system.principal.ActorRef,
		ProjectRef: system.project, StateRef: system.stateRef,
		ExpectedRevision: revision, Origin: origin,
		EvaluatorIdentity: gaps.EvaluatorV1Identity(),
		AuthorizationReceipt: system.authorizeIntake(
			t, application.IntakeOperationApply, requestRef,
		),
	}
}

func assertSQLiteWizardGapsRequestEffects(
	t *testing.T,
	system *sqliteIntakeTestSystem,
	requestRef string,
	wantOutcomes int,
	wantMutations int,
) {
	t.Helper()
	var outcomes, mutations int
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM wizard_gaps_outcomes WHERE request_ref=?`,
		requestRef,
	).Scan(&outcomes))
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM intake_receipts WHERE request_ref=?`,
		requestRef,
	).Scan(&mutations))
	if outcomes != wantOutcomes || mutations != wantMutations {
		t.Fatalf(
			"outcomes=%d/%d mutations=%d/%d",
			outcomes,
			wantOutcomes,
			mutations,
			wantMutations,
		)
	}
}
