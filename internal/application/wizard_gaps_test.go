package application

import (
	"context"
	"errors"
	"testing"

	"orquesta/internal/intake"
	"orquesta/internal/wizard/catalog"
	"orquesta/internal/wizard/gaps"
)

func TestWizardGapsFirstEvaluationPersistsThroughIntakeWriter(t *testing.T) {
	system, service := newWizardGapsTestSystem(t)
	request := wizardGapsRequest(t, system, "request:wizard-gaps-first", 1)

	result, err := service.ApplyWizardGaps(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Record.State.Revision() != 2 ||
		!result.EvaluationReplayExact ||
		result.Record.Receipt.Operation != IntakeOperationApply ||
		result.Record.Receipt.RequestRef != request.RequestRef ||
		!result.RequestRefReserved ||
		result.RequestOutcome.Kind != WizardGapsRequestOutcomeIntakeMutation ||
		result.RequestOutcome.ReceiptRef != result.Record.Receipt.Ref ||
		result.EvaluatorIdentity != request.EvaluatorIdentity ||
		result.InputDurability.ReceiptRef == "" ||
		result.InputDurability.SourceIntakeReceiptRef == "" ||
		result.InputDurability.Selections != "wizard_gaps_input_receipt" ||
		result.InputDurability.Facts != "wizard_gaps_input_receipt" ||
		result.InputDurability.PackRefs != "wizard_gaps_input_receipt" {
		t.Fatalf("result=%+v durability=%+v", result, result.InputDurability)
	}
	if len(result.Evaluation.Issues()) == 0 ||
		len(result.Record.State.Issues()) != len(result.Evaluation.Issues()) ||
		len(result.Record.State.Questions()) != len(result.Evaluation.Questions()) {
		t.Fatalf(
			"evaluation issues=%d questions=%d state=%+v",
			len(result.Evaluation.Issues()),
			len(result.Evaluation.Questions()),
			SnapshotIntake(result.Record.State),
		)
	}
	assertWizardQuestionProjection(t, result, gaps.DimensionU3)
	assertWizardQuestionProjection(t, result, gaps.DimensionU12)
}

func TestWizardGapsRequestOutcomeValidationFailsClosed(t *testing.T) {
	const intakeReceiptRef = "intake-receipt:" +
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	mutation := ApplyWizardGapsResult{
		Record: IntakeRecord{
			Receipt: IntakeReceipt{Ref: intakeReceiptRef},
		},
		RequestRefReserved: true,
		RequestOutcome: WizardGapsRequestOutcome{
			Kind:       WizardGapsRequestOutcomeIntakeMutation,
			ReceiptRef: intakeReceiptRef,
		},
		InputDurability: WizardGapsInputDurability{
			ReceiptRef: "wizard-gaps-input:" +
				"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			SourceIntakeReceiptRef: intakeReceiptRef,
			SelectionsDigest:       "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			FactsDigest:            "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			PackRefsDigest:         "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		},
	}
	noOp := mutation
	noOp.RequestOutcome = WizardGapsRequestOutcome{
		Kind: WizardGapsRequestOutcomeNoOp,
		ReceiptRef: "wizard-gaps-outcome:" +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	if err := ValidateApplyWizardGapsResult(mutation); err != nil {
		t.Fatalf("valid mutation: %v", err)
	}
	if err := ValidateApplyWizardGapsResult(noOp); err != nil {
		t.Fatalf("valid no-op: %v", err)
	}
	tests := map[string]ApplyWizardGapsResult{
		"unreserved": func() ApplyWizardGapsResult {
			value := mutation
			value.RequestRefReserved = false
			return value
		}(),
		"unknown kind": func() ApplyWizardGapsResult {
			value := mutation
			value.RequestOutcome.Kind = "unknown"
			return value
		}(),
		"mutation receipt mismatch": func() ApplyWizardGapsResult {
			value := mutation
			value.RequestOutcome.ReceiptRef = "intake-receipt:" +
				"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
			return value
		}(),
		"no-op uses snapshot receipt": func() ApplyWizardGapsResult {
			value := noOp
			value.RequestOutcome.ReceiptRef = value.Record.Receipt.Ref
			return value
		}(),
		"unsupported exact evaluation": func() ApplyWizardGapsResult {
			value := mutation
			value.EvaluationReplayExact = true
			return value
		}(),
	}
	for name, result := range tests {
		t.Run(name, func(t *testing.T) {
			if err := ValidateApplyWizardGapsResult(result); err == nil ||
				err.Error() != "application.wizard_gaps_request_outcome_invalid" {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestWizardGapsAddsNewlyActivatedQuestionsWithoutDuplicatingPersistedRefs(
	t *testing.T,
) {
	system, service := newWizardGapsTestSystem(t)
	first, err := service.ApplyWizardGaps(
		context.Background(),
		wizardGapsRequest(t, system, "request:wizard-gaps-base", 1),
	)
	if err != nil {
		t.Fatal(err)
	}
	integration := mustWizardDimensionQuestion(
		t, first.Evaluation, gaps.DimensionU7,
	)
	recommended, found := integration.RecommendedOption()
	if !found {
		t.Fatal("U7 recommendation missing")
	}
	answered, err := system.service.ApplyIntake(
		context.Background(),
		ApplyIntakeRequest{
			RequestRef: "request:wizard-gaps-answer-u7",
			ActorRef:   system.actor,
			ProjectRef: system.project,
			Change: intake.Change{
				StateRef: "intake:shared", ExpectedRevision: 2,
				Origin: intake.OriginForm,
				Choices: []intake.Choice{{
					QuestionRef: intake.QuestionRef(integration.Ref()),
					OptionRef:   intake.OptionRef(recommended.Ref()),
				}},
			},
			AuthorizationReceipt: system.authorizationFor(
				t, IntakeOperationApply, "request:wizard-gaps-answer-u7",
			),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	beforeQuestions := answered.Record.State.Questions()
	changed, err := service.ApplyWizardGaps(
		context.Background(),
		wizardGapsRequest(t, system, "request:wizard-gaps-activated", 3),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !changed.Changed || changed.Record.State.Revision() != 4 ||
		len(changed.Record.State.Questions()) <= len(beforeQuestions) {
		t.Fatalf(
			"changed=%t revision=%d questions=%d/%d",
			changed.Changed,
			changed.Record.State.Revision(),
			len(changed.Record.State.Questions()),
			len(beforeQuestions),
		)
	}
	assertUniqueWizardRefs(t, changed.Record.State)
	if _, found := currentQuestion(
		changed.Record.State,
		"intake-question:wizard.r5",
	); !found {
		t.Fatal("R5 question was not activated by durable U7 decision")
	}
}

func TestWizardGapsExactReplayReconstructsHistoricalIntakeRevision(t *testing.T) {
	system, service := newWizardGapsTestSystem(t)
	request := wizardGapsRequest(t, system, "request:wizard-gaps-replay", 1)
	first, err := service.ApplyWizardGaps(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	question := mustWizardDimensionQuestion(t, first.Evaluation, gaps.DimensionU1)
	recommended, found := question.RecommendedOption()
	if !found {
		t.Fatal("U1 recommendation missing")
	}
	if _, err = system.service.ApplyIntake(
		context.Background(),
		ApplyIntakeRequest{
			RequestRef: "request:wizard-gaps-later",
			ActorRef:   system.actor,
			ProjectRef: system.project,
			Change: intake.Change{
				StateRef: "intake:shared", ExpectedRevision: 2,
				Origin: intake.OriginForm,
				Choices: []intake.Choice{{
					QuestionRef: intake.QuestionRef(question.Ref()),
					OptionRef:   intake.OptionRef(recommended.Ref()),
				}},
			},
			AuthorizationReceipt: system.authorizationFor(
				t, IntakeOperationApply, "request:wizard-gaps-later",
			),
		},
	); err != nil {
		t.Fatal(err)
	}
	applyCalls := system.store.applyCalls
	replayed, err := service.ApplyWizardGaps(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Changed ||
		replayed.RequestOutcome.Kind != WizardGapsRequestOutcomeIntakeMutation ||
		replayed.RequestOutcome.ReceiptRef != first.Record.Receipt.Ref ||
		replayed.Record.Receipt != first.Record.Receipt ||
		replayed.Record.State.Revision() != first.Record.State.Revision() ||
		system.store.applyCalls != applyCalls {
		t.Fatalf(
			"first=%+v replayed=%+v apply_calls=%d/%d",
			first.Record.Receipt,
			replayed.Record.Receipt,
			applyCalls,
			system.store.applyCalls,
		)
	}
}

func TestWizardGapsReplayAcceptsEquivalentInputAndRejectsChangedProjection(
	t *testing.T,
) {
	t.Run("equivalent pack refs", func(t *testing.T) {
		system, service := newWizardGapsTestSystem(t)
		packRef := catalog.DomainPackRefs()[0]
		request := wizardGapsRequest(
			t, system, "request:wizard-gaps-equivalent", 1,
		)
		request.PackRefs = []catalog.PackRef{packRef}
		first, err := service.ApplyWizardGaps(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		request.PackRefs = []catalog.PackRef{packRef, packRef}
		replayed, err := service.ApplyWizardGaps(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if replayed.Changed || replayed.Record.Receipt != first.Record.Receipt {
			t.Fatalf(
				"first=%+v replayed=%+v",
				first.Record.Receipt,
				replayed.Record.Receipt,
			)
		}
	})

	t.Run("same request different change", func(t *testing.T) {
		system, service := newWizardGapsTestSystem(t)
		request := wizardGapsRequest(
			t, system, "request:wizard-gaps-divergent", 1,
		)
		if _, err := service.ApplyWizardGaps(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		request.Facts.Surface = gaps.SurfaceServerService
		if _, err := service.ApplyWizardGaps(
			context.Background(), request,
		); !IsStateError(err, StateConflict) {
			t.Fatalf("divergent replay err=%v", err)
		}
	})
}

func TestWizardGapsRejectsStaleRequestAndLeavesWriterUntouched(t *testing.T) {
	system, service := newWizardGapsTestSystem(t)
	if _, err := service.ApplyWizardGaps(
		context.Background(),
		wizardGapsRequest(t, system, "request:wizard-gaps-current", 1),
	); err != nil {
		t.Fatal(err)
	}
	applyCalls := system.store.applyCalls
	_, err := service.ApplyWizardGaps(
		context.Background(),
		wizardGapsRequest(t, system, "request:wizard-gaps-stale", 1),
	)
	if intake.ErrorCodeOf(err) != intake.ErrorRevisionConflict ||
		system.store.applyCalls != applyCalls {
		t.Fatalf(
			"stale err=%v code=%q apply_calls=%d/%d",
			err,
			intake.ErrorCodeOf(err),
			applyCalls,
			system.store.applyCalls,
		)
	}
}

func TestWizardGapsDerivesDurableFreeTextFromCurrentDecision(t *testing.T) {
	system, service := newWizardGapsTestSystem(t)
	first, err := service.ApplyWizardGaps(
		context.Background(),
		wizardGapsRequest(t, system, "request:wizard-gaps-text-base", 1),
	)
	if err != nil {
		t.Fatal(err)
	}
	question := mustWizardDimensionQuestion(t, first.Evaluation, gaps.DimensionU1)
	var freeTextOption gaps.Option
	found := false
	for _, option := range question.Options() {
		if option.Kind() == gaps.OptionFreeText {
			freeTextOption, found = option, true
			break
		}
	}
	if !found {
		t.Fatal("U1 free-text option missing")
	}
	const answer = "Equipos internos y clínicas asociadas"
	answered, err := system.service.ApplyIntake(
		context.Background(),
		ApplyIntakeRequest{
			RequestRef: "request:wizard-gaps-text-answer",
			ActorRef:   system.actor,
			ProjectRef: system.project,
			Change: intake.Change{
				StateRef: "intake:shared", ExpectedRevision: 2,
				Origin: intake.OriginChat,
				Choices: []intake.Choice{{
					QuestionRef: intake.QuestionRef(question.Ref()),
					OptionRef:   intake.OptionRef(freeTextOption.Ref()),
					AnswerText:  answer,
				}},
			},
			AuthorizationReceipt: system.authorizationFor(
				t, IntakeOperationApply, "request:wizard-gaps-text-answer",
			),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	evaluator := wizardGapsV1Evaluator(t)
	preflight, err := preflightWizardGapsQuestions(answered.Record.State, evaluator)
	if err != nil {
		t.Fatal(err)
	}
	selections, _ := wizardGapsDimensionSelections(
		answered.Record.State,
		preflight.dimensions,
	)
	selection, found := selectionByDimension(selections, gaps.DimensionU1)
	if !found || selection.Option != freeTextOption.Ref() ||
		selection.FreeText != answer {
		t.Fatalf("selection=%+v found=%t", selection, found)
	}
	evaluation, err := evaluateWizardGaps(
		answered.Record.State,
		gaps.Facts{},
		nil,
		evaluator,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, found = wizardQuestionByDimension(
		evaluation,
		gaps.DimensionU1,
	); found {
		t.Fatal("durably answered U1 was emitted as a gap")
	}
}

func TestWizardGapsReconcilesChangedProjectionAndReplaysExactly(t *testing.T) {
	system, service := newWizardGapsTestSystem(t)
	first, err := service.ApplyWizardGaps(
		context.Background(),
		wizardGapsRequest(t, system, "request:wizard-gaps-reopen-base", 1),
	)
	if err != nil {
		t.Fatal(err)
	}
	parent := mustWizardDimensionQuestion(t, first.Evaluation, gaps.DimensionU1)
	child := mustWizardDimensionQuestion(t, first.Evaluation, gaps.DimensionU3)
	parentChoice, _ := parent.RecommendedOption()
	childChoice, _ := child.RecommendedOption()
	answered, err := system.service.ApplyIntake(
		context.Background(),
		ApplyIntakeRequest{
			RequestRef: "request:wizard-gaps-reopen-answer",
			ActorRef:   system.actor,
			ProjectRef: system.project,
			Change: intake.Change{
				StateRef: "intake:shared", ExpectedRevision: 2,
				Origin: intake.OriginForm,
				Choices: []intake.Choice{
					{
						QuestionRef: intake.QuestionRef(parent.Ref()),
						OptionRef:   intake.OptionRef(parentChoice.Ref()),
					},
					{
						QuestionRef: intake.QuestionRef(child.Ref()),
						OptionRef:   intake.OptionRef(childChoice.Ref()),
					},
				},
			},
			AuthorizationReceipt: system.authorizationFor(
				t, IntakeOperationApply, "request:wizard-gaps-reopen-answer",
			),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	var alternative gaps.Option
	for _, option := range parent.Options() {
		if option.Kind() == gaps.OptionPreset && option.Ref() != parentChoice.Ref() {
			alternative = option
			break
		}
	}
	if alternative.Ref() == "" {
		t.Fatal("U1 alternative missing")
	}
	changedParent, err := system.service.ApplyIntake(
		context.Background(),
		ApplyIntakeRequest{
			RequestRef: "request:wizard-gaps-reopen-parent",
			ActorRef:   system.actor,
			ProjectRef: system.project,
			Change: intake.Change{
				StateRef:         "intake:shared",
				ExpectedRevision: answered.Record.State.Revision(),
				Origin:           intake.OriginChat,
				Choices: []intake.Choice{{
					QuestionRef: intake.QuestionRef(parent.Ref()),
					OptionRef:   intake.OptionRef(alternative.Ref()),
				}},
			},
			AuthorizationReceipt: system.authorizationFor(
				t, IntakeOperationApply, "request:wizard-gaps-reopen-parent",
			),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := changedParent.Record.State.CurrentDecision(
		intake.QuestionRef(child.Ref()),
	); found {
		t.Fatal("dependent U3 decision remained current")
	}
	reopened := changedParent.Record.State.ReopenedDecisions()
	if len(reopened) == 0 ||
		reopened[0].QuestionRef != intake.QuestionRef(child.Ref()) {
		t.Fatalf("reopened=%+v", reopened)
	}

	applyCalls := system.store.applyCalls
	request := wizardGapsRequest(
		t,
		system,
		"request:wizard-gaps-reopen-evaluate",
		changedParent.Record.State.Revision(),
	)
	result, err := service.ApplyWizardGaps(
		context.Background(),
		request,
	)
	if err != nil || !result.Changed ||
		system.store.applyCalls != applyCalls+1 {
		var projection *WizardGapsProjectionConflictError
		_ = errors.As(err, &projection)
		t.Fatalf(
			"evaluation err=%v projection=%+v changed=%t calls=%d/%d",
			err,
			projection,
			result.Changed,
			applyCalls,
			system.store.applyCalls,
		)
	}
	activeChild := questionByIntakeRef(
		t,
		result.Record.State.Questions(),
		intake.QuestionRef(child.Ref()),
	)
	if wizardGapsQuestionPayloadEqual(activeChild, child.IntakeQuestion()) {
		t.Fatal("reconciliation kept the stale child projection")
	}
	versions := result.Record.State.QuestionVersions()
	if len(versions) <= len(first.Record.State.QuestionVersions()) {
		t.Fatalf("question versions were not appended: %+v", versions)
	}
	var latest intake.QuestionVersion
	for _, version := range versions {
		if version.Question.Ref == intake.QuestionRef(child.Ref()) {
			latest = version
		}
	}
	if latest.Question.Ref != intake.QuestionRef(child.Ref()) ||
		latest.ReplacesRevision == 0 ||
		latest.Revision != result.Record.State.Revision() {
		t.Fatalf("latest question version = %+v", latest)
	}
	current, getErr := system.service.GetIntake(
		context.Background(),
		GetIntakeRequest{
			ActorRef: system.actor, ProjectRef: system.project,
			StateRef: "intake:shared",
		},
	)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if _, found := current.State.CurrentDecision(
		intake.QuestionRef(child.Ref()),
	); found {
		t.Fatal("Wizard evaluation restored invalidated U3 decision")
	}
	reopened = current.State.ReopenedDecisions()
	if len(reopened) == 0 ||
		reopened[0].QuestionRef != intake.QuestionRef(child.Ref()) {
		t.Fatalf("reconciled reopen=%+v", reopened)
	}

	replayed, replayErr := service.ApplyWizardGaps(context.Background(), request)
	if replayErr != nil || replayed.Changed ||
		replayed.Record.Receipt != result.Record.Receipt ||
		system.store.applyCalls != applyCalls+1 {
		t.Fatalf(
			"replay err=%v changed=%t receipt=%+v/%+v calls=%d",
			replayErr,
			replayed.Changed,
			replayed.Record.Receipt,
			result.Record.Receipt,
			system.store.applyCalls,
		)
	}
}

func questionByIntakeRef(
	t *testing.T,
	questions []intake.Question,
	ref intake.QuestionRef,
) intake.Question {
	t.Helper()
	for _, question := range questions {
		if question.Ref == ref {
			return question
		}
	}
	t.Fatalf("question %s missing", ref)
	return intake.Question{}
}

func TestWizardGapsProjectsExplicitPackIntoDurableInputReceipt(t *testing.T) {
	system, service := newWizardGapsTestSystem(t)
	packRef := catalog.DomainPackRefs()[0]
	request := wizardGapsRequest(t, system, "request:wizard-gaps-pack", 1)
	request.PackRefs = []catalog.PackRef{packRef}

	result, err := service.ApplyWizardGaps(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed ||
		result.InputDurability.PackRefs != "wizard_gaps_input_receipt" ||
		result.InputDurability.PackRefsDigest == "" {
		t.Fatalf("result=%+v", result)
	}
	packQuestions := 0
	for _, question := range result.Evaluation.Questions() {
		if question.PackRef() != packRef {
			continue
		}
		packQuestions++
		projected, found := currentQuestion(
			result.Record.State,
			intake.QuestionRef(question.Ref()),
		)
		if !found ||
			!containsQuestionRef(
				projected.DependsOn,
				"intake-question:wizard.u7",
			) {
			t.Fatalf("pack question=%+v found=%t", projected, found)
		}
	}
	if packQuestions == 0 {
		t.Fatalf("pack %q produced no projected question", packRef.String())
	}
}

func TestWizardGapsNoOpCreatesNoIntakeMutationAndReservesOutcome(t *testing.T) {
	system, service := newWizardGapsTestSystem(t)
	first, err := service.ApplyWizardGaps(
		context.Background(),
		wizardGapsRequest(t, system, "request:wizard-gaps-noop-base", 1),
	)
	if err != nil {
		t.Fatal(err)
	}
	applyCalls := system.store.applyCalls
	result, err := service.ApplyWizardGaps(
		context.Background(),
		wizardGapsRequest(
			t,
			system,
			"request:wizard-gaps-noop",
			first.Record.State.Revision(),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Changed ||
		!result.RequestRefReserved ||
		result.RequestOutcome.Kind != WizardGapsRequestOutcomeNoOp ||
		!validWizardGapsNoOpOutcomeRef(result.RequestOutcome.ReceiptRef) ||
		result.RequestOutcome.ReceiptRef == result.Record.Receipt.Ref ||
		result.Record.Receipt != first.Record.Receipt ||
		result.Record.State.Revision() != first.Record.State.Revision() ||
		system.store.applyCalls != applyCalls ||
		system.store.wizardReserveCalls != 1 {
		t.Fatalf(
			"result=%+v calls=%d/%d",
			result,
			applyCalls,
			system.store.applyCalls,
		)
	}
}

type conflictingWizardGapsOutcomeStore struct {
	*memoryIntakeStore
}

func (store conflictingWizardGapsOutcomeStore) ReplayWizardGapsInput(
	context.Context,
	WizardGapsInputReplayRequest,
) (WizardGapsInputRecord, bool, error) {
	return WizardGapsInputRecord{}, false, nil
}

func (store conflictingWizardGapsOutcomeStore) ReserveWizardGapsNoOp(
	context.Context,
	WizardGapsNoOpReservation,
) (WizardGapsInputRecord, bool, error) {
	return WizardGapsInputRecord{}, false, &StateError{Code: StateConflict}
}

func TestWizardGapsNoOpReservationConflictWithoutReplayFailsClosed(t *testing.T) {
	system, service := newWizardGapsTestSystem(t)
	first, err := service.ApplyWizardGaps(
		context.Background(),
		wizardGapsRequest(t, system, "request:wizard-gaps-conflict-base", 1),
	)
	if err != nil {
		t.Fatal(err)
	}
	service, err = NewWizardGapsService(
		conflictingWizardGapsOutcomeStore{memoryIntakeStore: system.store},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ApplyWizardGaps(
		context.Background(),
		wizardGapsRequest(
			t,
			system,
			"request:wizard-gaps-conflict-noop",
			first.Record.State.Revision(),
		),
	)
	if !IsStateError(err, StateConflict) {
		t.Fatalf("reservation conflict err=%v", err)
	}
}

func TestWizardGapsAuthorizationFailureStopsBeforeIntakeRead(t *testing.T) {
	system, service := newWizardGapsTestSystem(t)
	request := wizardGapsRequest(t, system, "request:wizard-gaps-forbidden", 1)
	request.AuthorizationReceipt = system.authorizationFor(
		t, IntakeOperationCreate, request.RequestRef,
	)
	getCalls := system.store.getCalls
	if _, err := service.ApplyWizardGaps(
		context.Background(), request,
	); !errors.Is(err, ErrForbidden) || system.store.getCalls != getCalls {
		t.Fatalf(
			"err=%v get_calls=%d/%d",
			err,
			getCalls,
			system.store.getCalls,
		)
	}
}

func newWizardGapsTestSystem(
	t *testing.T,
) (intakeTestSystem, *WizardGapsService) {
	t.Helper()
	system := newIntakeTestSystem(t)
	mustCreateIntake(t, system)
	service, err := NewWizardGapsService(system.store)
	if err != nil {
		t.Fatal(err)
	}
	return system, service
}

func wizardGapsRequest(
	t *testing.T,
	system intakeTestSystem,
	requestRef string,
	revision intake.Revision,
) ApplyWizardGapsRequest {
	t.Helper()
	return ApplyWizardGapsRequest{
		RequestRef: requestRef, ActorRef: system.actor,
		ProjectRef: system.project, StateRef: "intake:shared",
		ExpectedRevision: revision, Origin: intake.OriginForm,
		EvaluatorIdentity: gaps.EvaluatorV1Identity(),
		AuthorizationReceipt: system.authorizationFor(
			t, IntakeOperationApply, requestRef,
		),
	}
}

func wizardGapsV1Evaluator(t *testing.T) wizardGapsEvaluator {
	t.Helper()
	evaluator, err := gaps.BuiltInEvaluatorRegistry().Resolve(
		gaps.EvaluatorV1Identity(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return evaluator
}

func mustWizardDimensionQuestion(
	t *testing.T,
	result gaps.Result,
	dimension gaps.DimensionRef,
) gaps.Question {
	t.Helper()
	question, found := wizardQuestionByDimension(result, dimension)
	if !found {
		t.Fatalf("question for %s missing", dimension)
	}
	return question
}

func wizardQuestionByDimension(
	result gaps.Result,
	dimension gaps.DimensionRef,
) (gaps.Question, bool) {
	for _, question := range result.Questions() {
		if question.Dimension() == dimension {
			return question, true
		}
	}
	return gaps.Question{}, false
}

func assertWizardQuestionProjection(
	t *testing.T,
	result ApplyWizardGapsResult,
	dimension gaps.DimensionRef,
) {
	t.Helper()
	source := mustWizardDimensionQuestion(t, result.Evaluation, dimension)
	projected, found := currentQuestion(
		result.Record.State,
		intake.QuestionRef(source.Ref()),
	)
	if !found {
		t.Fatalf("projected question %q missing", source.Ref())
	}
	want := source.IntakeQuestion()
	if len(projected.DependsOn) != len(want.DependsOn) ||
		len(projected.Options) != len(want.Options) {
		t.Fatalf("projected=%+v want=%+v", projected, want)
	}
	for index := range want.DependsOn {
		if projected.DependsOn[index] != want.DependsOn[index] {
			t.Fatalf("projected deps=%v want=%v", projected.DependsOn, want.DependsOn)
		}
	}
	for index := range want.Options {
		if projected.Options[index].AcceptsText != want.Options[index].AcceptsText {
			t.Fatalf("projected options=%+v want=%+v", projected.Options, want.Options)
		}
	}
}

func assertUniqueWizardRefs(t *testing.T, state intake.State) {
	t.Helper()
	issues := make(map[intake.IssueRef]struct{})
	for _, issue := range state.Issues() {
		if _, found := issues[issue.Ref]; found {
			t.Fatalf("duplicate issue %q", issue.Ref)
		}
		issues[issue.Ref] = struct{}{}
	}
	questions := make(map[intake.QuestionRef]struct{})
	for _, question := range state.Questions() {
		if _, found := questions[question.Ref]; found {
			t.Fatalf("duplicate question %q", question.Ref)
		}
		questions[question.Ref] = struct{}{}
	}
}

func currentQuestion(
	state intake.State,
	ref intake.QuestionRef,
) (intake.Question, bool) {
	for _, question := range state.Questions() {
		if question.Ref == ref {
			return question, true
		}
	}
	return intake.Question{}, false
}

func selectionByDimension(
	selections []gaps.Selection,
	dimension gaps.DimensionRef,
) (gaps.Selection, bool) {
	for _, selection := range selections {
		if selection.Dimension == dimension {
			return selection, true
		}
	}
	return gaps.Selection{}, false
}

func containsQuestionRef(
	values []intake.QuestionRef,
	want intake.QuestionRef,
) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
