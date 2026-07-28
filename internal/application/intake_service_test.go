package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

type memoryIntakeStore struct {
	current            map[string]IntakeRecord
	requests           map[string]IntakeRecord
	wizardGapsNoOps    map[string]WizardGapsNoOpOutcome
	wizardGapsInputs   map[string]WizardGapsInputRecord
	replayCalls        int
	createCalls        int
	getCalls           int
	applyCalls         int
	wizardReplayCalls  int
	wizardReserveCalls int
}

func newMemoryIntakeStore() *memoryIntakeStore {
	return &memoryIntakeStore{
		current:          make(map[string]IntakeRecord),
		requests:         make(map[string]IntakeRecord),
		wizardGapsNoOps:  make(map[string]WizardGapsNoOpOutcome),
		wizardGapsInputs: make(map[string]WizardGapsInputRecord),
	}
}

func (store *memoryIntakeStore) ReplayIntake(
	_ context.Context,
	request IntakeReplayRequest,
) (IntakeRecord, bool, error) {
	store.replayCalls++
	key := intakeRequestKey(
		request.ActorRef, request.ProjectRef, request.RequestRef,
	)
	record, found := store.requests[intakeRequestKey(
		request.ActorRef, request.ProjectRef, request.RequestRef,
	)]
	if !found {
		if _, reserved := store.wizardGapsNoOps[key]; reserved {
			return IntakeRecord{}, false, &StateError{Code: StateConflict}
		}
		return IntakeRecord{}, false, nil
	}
	if record.Receipt.RequestFingerprint != request.RequestFingerprint ||
		record.Receipt.Operation != request.Operation ||
		record.Receipt.StateRef != request.StateRef ||
		record.Receipt.AuthorizationReceiptRef != request.AuthorizationReceiptRef {
		return IntakeRecord{}, false, &StateError{Code: StateConflict}
	}
	return record, true, nil
}

func (store *memoryIntakeStore) CreateIntake(
	ctx context.Context,
	state IntakeCreateState,
) (IntakeRecord, bool, error) {
	store.createCalls++
	replay := IntakeReplayRequest{
		RequestRef: state.RequestRef, RequestFingerprint: state.RequestFingerprint,
		Operation: IntakeOperationCreate, ActorRef: state.ActorRef,
		ProjectRef: state.ProjectRef, StateRef: state.State.Ref(),
		AuthorizationReceiptRef: state.AuthorizationReceipt.Ref(),
	}
	if record, found, err := store.ReplayIntake(ctx, replay); err != nil || found {
		return record, false, err
	}
	key := intakeStateKey(state.ActorRef, state.ProjectRef, state.State.Ref())
	if _, found := store.current[key]; found {
		return IntakeRecord{}, false, &StateError{Code: StateConflict}
	}
	record := IntakeRecord{
		ActorRef: state.ActorRef, ProjectRef: state.ProjectRef,
		State: state.State, Receipt: state.Receipt,
	}
	store.current[key] = record
	store.requests[intakeRequestKey(
		state.ActorRef, state.ProjectRef, state.RequestRef,
	)] = record
	return record, true, nil
}

func (store *memoryIntakeStore) GetIntake(
	_ context.Context,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	stateRef intake.Ref,
) (IntakeRecord, error) {
	store.getCalls++
	record, found := store.current[intakeStateKey(actorRef, projectRef, stateRef)]
	if !found {
		return IntakeRecord{}, &StateError{Code: StateNotFound}
	}
	return record, nil
}

func (store *memoryIntakeStore) ApplyIntake(
	ctx context.Context,
	state IntakeApplyState,
) (IntakeRecord, bool, error) {
	store.applyCalls++
	replay := IntakeReplayRequest{
		RequestRef: state.RequestRef, RequestFingerprint: state.RequestFingerprint,
		Operation: IntakeOperationApply, ActorRef: state.ActorRef,
		ProjectRef: state.ProjectRef, StateRef: state.State.Ref(),
		AuthorizationReceiptRef: state.AuthorizationReceipt.Ref(),
	}
	if record, found, err := store.ReplayIntake(ctx, replay); err != nil || found {
		return record, false, err
	}
	key := intakeStateKey(state.ActorRef, state.ProjectRef, state.State.Ref())
	current, found := store.current[key]
	if !found {
		return IntakeRecord{}, false, &StateError{Code: StateNotFound}
	}
	if current.State.Revision() != state.ExpectedRevision ||
		state.State.Revision() != state.ExpectedRevision+1 {
		return IntakeRecord{}, false, &StateError{Code: StateConflict}
	}
	record := IntakeRecord{
		ActorRef: state.ActorRef, ProjectRef: state.ProjectRef,
		State: state.State, Receipt: state.Receipt,
	}
	store.current[key] = record
	store.requests[intakeRequestKey(
		state.ActorRef, state.ProjectRef, state.RequestRef,
	)] = record
	return record, true, nil
}

func (store *memoryIntakeStore) ReplayWizardGapsNoOp(
	_ context.Context,
	request WizardGapsNoOpReplayRequest,
) (WizardGapsNoOpOutcome, bool, error) {
	store.wizardReplayCalls++
	key := intakeRequestKey(
		request.ActorRef, request.ProjectRef, request.RequestRef,
	)
	outcome, found := store.wizardGapsNoOps[key]
	if !found {
		return WizardGapsNoOpOutcome{}, false, nil
	}
	if outcome.RequestFingerprint != request.RequestFingerprint ||
		outcome.StateRef != request.StateRef ||
		outcome.ExpectedRevision != request.ExpectedRevision ||
		outcome.EvaluatorIdentity != request.EvaluatorIdentity ||
		outcome.AuthorizationReceiptRef != request.AuthorizationReceiptRef {
		return WizardGapsNoOpOutcome{}, false, &StateError{Code: StateConflict}
	}
	return outcome, true, nil
}

func (store *memoryIntakeStore) ReplayWizardGapsInput(
	_ context.Context,
	request WizardGapsInputReplayRequest,
) (WizardGapsInputRecord, bool, error) {
	store.wizardReplayCalls++
	record, found := store.wizardGapsInputs[intakeRequestKey(
		request.ActorRef, request.ProjectRef, request.RequestRef,
	)]
	if !found {
		return WizardGapsInputRecord{}, false, nil
	}
	if err := validateWizardGapsInputRecord(request, record); err != nil {
		return WizardGapsInputRecord{}, false, err
	}
	return record, true, nil
}

func (store *memoryIntakeStore) ReserveWizardGapsNoOp(
	ctx context.Context,
	reservation WizardGapsNoOpReservation,
) (WizardGapsInputRecord, bool, error) {
	store.wizardReserveCalls++
	outcome := reservation.Outcome
	request := WizardGapsInputReplayRequest{
		RequestRef:              reservation.Input.RequestRef,
		RequestFingerprint:      reservation.Input.RequestFingerprint,
		ActorRef:                reservation.Input.ActorRef,
		ProjectRef:              reservation.Input.ProjectRef,
		StateRef:                reservation.Input.StateRef,
		ExpectedRevision:        reservation.Input.ExpectedRevision,
		EvaluatorIdentity:       reservation.Input.EvaluatorIdentity,
		AuthorizationReceiptRef: reservation.Input.AuthorizationReceiptRef,
	}
	if replayed, found, err := store.ReplayWizardGapsInput(
		ctx, request,
	); err != nil || found {
		return replayed, false, err
	}
	if _, reserved := store.requests[intakeRequestKey(
		outcome.ActorRef, outcome.ProjectRef, outcome.RequestRef,
	)]; reserved {
		return WizardGapsInputRecord{}, false, &StateError{Code: StateConflict}
	}
	key := intakeStateKey(outcome.ActorRef, outcome.ProjectRef, outcome.StateRef)
	current, found := store.current[key]
	if !found {
		return WizardGapsInputRecord{}, false, &StateError{Code: StateNotFound}
	}
	if current.State.Revision() != outcome.ExpectedRevision ||
		current.Receipt.Ref != outcome.SourceIntakeReceiptRef ||
		current.Receipt != outcome.Record.Receipt {
		return WizardGapsInputRecord{}, false, &StateError{Code: StateConflict}
	}
	requestKey := intakeRequestKey(
		outcome.ActorRef, outcome.ProjectRef, outcome.RequestRef,
	)
	record := WizardGapsInputRecord{
		Receipt: reservation.Input, SourceRecord: current, OutcomeRecord: current,
	}
	if err := ValidateWizardGapsInputRecord(record); err != nil {
		return WizardGapsInputRecord{}, false, err
	}
	store.wizardGapsNoOps[requestKey] = outcome
	store.wizardGapsInputs[requestKey] = record
	return record, true, nil
}

func (store *memoryIntakeStore) ApplyWizardGapsMutation(
	ctx context.Context,
	reservation WizardGapsMutationReservation,
) (WizardGapsInputRecord, bool, error) {
	request := WizardGapsInputReplayRequest{
		RequestRef:              reservation.Input.RequestRef,
		RequestFingerprint:      reservation.Input.RequestFingerprint,
		ActorRef:                reservation.Input.ActorRef,
		ProjectRef:              reservation.Input.ProjectRef,
		StateRef:                reservation.Input.StateRef,
		ExpectedRevision:        reservation.Input.ExpectedRevision,
		EvaluatorIdentity:       reservation.Input.EvaluatorIdentity,
		AuthorizationReceiptRef: reservation.Input.AuthorizationReceiptRef,
	}
	if replayed, found, err := store.ReplayWizardGapsInput(
		ctx, request,
	); err != nil || found {
		return replayed, false, err
	}
	requestKey := intakeRequestKey(
		reservation.Input.ActorRef,
		reservation.Input.ProjectRef,
		reservation.Input.RequestRef,
	)
	if _, found := store.requests[requestKey]; found {
		return WizardGapsInputRecord{}, false, &StateError{
			Code: StateConflict, Cause: errors.New("test.wizard_input_request_exists"),
		}
	}
	stateKey := intakeStateKey(
		reservation.Input.ActorRef,
		reservation.Input.ProjectRef,
		reservation.Input.StateRef,
	)
	source, found := store.current[stateKey]
	if !found {
		return WizardGapsInputRecord{}, false, &StateError{Code: StateNotFound}
	}
	state := reservation.Intake
	if source.State.Revision() != state.ExpectedRevision ||
		state.State.Revision() != state.ExpectedRevision+1 ||
		source.Receipt.Ref != reservation.Input.SourceIntakeReceiptRef {
		return WizardGapsInputRecord{}, false, &StateError{
			Code: StateConflict, Cause: errors.New("test.wizard_input_source_mismatch"),
		}
	}
	outcome := IntakeRecord{
		ActorRef: state.ActorRef, ProjectRef: state.ProjectRef,
		State: state.State, Receipt: state.Receipt,
	}
	record := WizardGapsInputRecord{
		Receipt: reservation.Input, SourceRecord: source, OutcomeRecord: outcome,
	}
	if err := ValidateWizardGapsInputRecord(record); err != nil {
		return WizardGapsInputRecord{}, false, &StateError{
			Code: StateConflict,
			Cause: errors.Join(
				errors.New("test.wizard_input_record_invalid"),
				err,
			),
		}
	}
	store.current[stateKey] = outcome
	store.requests[requestKey] = outcome
	store.wizardGapsInputs[requestKey] = record
	store.applyCalls++
	return record, true, nil
}

func TestIntakeServiceCreatesAndAppliesChatAndFormToOneState(t *testing.T) {
	system := newIntakeTestSystem(t)
	created, err := system.service.CreateIntake(context.Background(), CreateIntakeRequest{
		RequestRef: "request:intake-create", ActorRef: system.actor,
		ProjectRef: system.project, StateRef: "intake:shared",
		Policy:               intake.Policy{MaxQuestionRounds: 3},
		AuthorizationReceipt: system.authorizationFor(t, IntakeOperationCreate, "request:intake-create"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created.Changed || created.Record.State.Revision() != 1 ||
		created.Record.Receipt.Operation != IntakeOperationCreate ||
		created.Record.Receipt.PreviousRevision != 0 {
		t.Fatalf("create result = %+v", created)
	}

	chat, err := system.service.ApplyIntake(context.Background(), ApplyIntakeRequest{
		RequestRef: "request:intake-chat", ActorRef: system.actor,
		ProjectRef: system.project, Change: intakeQuestionChange(1, intake.OriginChat),
		AuthorizationReceipt: system.authorizationFor(t, IntakeOperationApply, "request:intake-chat"),
	})
	if err != nil {
		t.Fatal(err)
	}
	formChange := intake.Change{
		StateRef: "intake:shared", ExpectedRevision: 2, Origin: intake.OriginForm,
		Choices: []intake.Choice{{
			QuestionRef: "intake-question:audience",
			OptionRef:   "intake-option:audience-personal",
		}},
	}
	form, err := system.service.ApplyIntake(context.Background(), ApplyIntakeRequest{
		RequestRef: "request:intake-form", ActorRef: system.actor,
		ProjectRef: system.project, Change: formChange,
		AuthorizationReceipt: system.authorizationFor(t, IntakeOperationApply, "request:intake-form"),
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err := system.service.GetIntake(context.Background(), GetIntakeRequest{
		ActorRef: system.actor, ProjectRef: system.project, StateRef: "intake:shared",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !chat.Changed || !form.Changed || current.State.Revision() != 3 ||
		current.State.Ref() != created.Record.State.Ref() ||
		current.ActorRef != system.actor || current.ProjectRef != system.project {
		t.Fatalf("shared state create=%+v chat=%+v form=%+v current=%+v",
			created, chat, form, current)
	}
	history := current.State.History()
	if len(history) != 2 || history[0].Origin != intake.OriginChat ||
		history[1].Origin != intake.OriginForm {
		t.Fatalf("history = %+v", history)
	}
}

func TestIntakeServiceReplayReturnsExactReceiptAfterLaterMutation(t *testing.T) {
	system := newIntakeTestSystem(t)
	mustCreateIntake(t, system)
	request := ApplyIntakeRequest{
		RequestRef: "request:intake-replay", ActorRef: system.actor,
		ProjectRef: system.project, Change: intakeQuestionChange(1, intake.OriginChat),
		AuthorizationReceipt: system.authorizationFor(t, IntakeOperationApply, "request:intake-replay"),
	}
	first, err := system.service.ApplyIntake(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	_, err = system.service.ApplyIntake(context.Background(), ApplyIntakeRequest{
		RequestRef: "request:intake-later", ActorRef: system.actor,
		ProjectRef: system.project,
		Change: intake.Change{
			StateRef: "intake:shared", ExpectedRevision: 2, Origin: intake.OriginForm,
			Choices: []intake.Choice{{
				QuestionRef: "intake-question:audience",
				OptionRef:   "intake-option:audience-team",
			}},
		},
		AuthorizationReceipt: system.authorizationFor(t, IntakeOperationApply, "request:intake-later"),
	})
	if err != nil {
		t.Fatal(err)
	}
	applyCalls := system.store.applyCalls
	replayed, err := system.service.ApplyIntake(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Changed || replayed.Record.Receipt != first.Record.Receipt ||
		replayed.Record.State.Revision() != first.Record.State.Revision() ||
		system.store.applyCalls != applyCalls {
		t.Fatalf("replay first=%+v replayed=%+v apply_calls=%d/%d",
			first, replayed, applyCalls, system.store.applyCalls)
	}
}

func TestIntakeServiceRejectsStaleAndDivergentRequestsWithoutWrite(t *testing.T) {
	system := newIntakeTestSystem(t)
	mustCreateIntake(t, system)
	_, err := system.service.ApplyIntake(context.Background(), ApplyIntakeRequest{
		RequestRef: "request:intake-original", ActorRef: system.actor,
		ProjectRef: system.project, Change: intakeQuestionChange(1, intake.OriginChat),
		AuthorizationReceipt: system.authorizationFor(t, IntakeOperationApply, "request:intake-original"),
	})
	if err != nil {
		t.Fatal(err)
	}
	applyCalls := system.store.applyCalls
	_, err = system.service.ApplyIntake(context.Background(), ApplyIntakeRequest{
		RequestRef: "request:intake-stale", ActorRef: system.actor,
		ProjectRef: system.project, Change: intakeQuestionChange(1, intake.OriginForm),
		AuthorizationReceipt: system.authorizationFor(t, IntakeOperationApply, "request:intake-stale"),
	})
	if intake.ErrorCodeOf(err) != intake.ErrorRevisionConflict ||
		system.store.applyCalls != applyCalls {
		t.Fatalf("stale err=%v apply_calls=%d/%d", err, applyCalls, system.store.applyCalls)
	}

	divergent := intakeQuestionChange(1, intake.OriginChat)
	divergent.Origin = intake.OriginForm
	_, err = system.service.ApplyIntake(context.Background(), ApplyIntakeRequest{
		RequestRef: "request:intake-original", ActorRef: system.actor,
		ProjectRef: system.project, Change: divergent,
		AuthorizationReceipt: system.authorizationFor(t, IntakeOperationApply, "request:intake-original"),
	})
	if !IsStateError(err, StateConflict) || system.store.applyCalls != applyCalls {
		t.Fatalf("divergent err=%v apply_calls=%d/%d", err, applyCalls, system.store.applyCalls)
	}
}

func TestIntakeServiceValidatesCompleteMutationBeforeStore(t *testing.T) {
	system := newIntakeTestSystem(t)
	_, err := system.service.CreateIntake(context.Background(), CreateIntakeRequest{
		RequestRef: "request:invalid-policy", ActorRef: system.actor,
		ProjectRef: system.project, StateRef: "intake:invalid", Policy: intake.Policy{},
		AuthorizationReceipt: system.authorizationFor(t, IntakeOperationCreate, "request:invalid-policy"),
	})
	if intake.ErrorCodeOf(err) != intake.ErrorInvalidArgument ||
		system.store.replayCalls != 0 || system.store.createCalls != 0 {
		t.Fatalf("invalid create err=%v replay=%d create=%d",
			err, system.store.replayCalls, system.store.createCalls)
	}

	mustCreateIntake(t, system)
	invalid := intakeQuestionChange(1, intake.OriginChat)
	invalid.Questions[0].Options[0].Recommended = false
	applyCalls := system.store.applyCalls
	_, err = system.service.ApplyIntake(context.Background(), ApplyIntakeRequest{
		RequestRef: "request:invalid-question", ActorRef: system.actor,
		ProjectRef: system.project, Change: invalid,
		AuthorizationReceipt: system.authorizationFor(t, IntakeOperationApply, "request:invalid-question"),
	})
	if intake.ErrorCodeOf(err) != intake.ErrorRecommendationCount ||
		system.store.applyCalls != applyCalls {
		t.Fatalf("invalid apply err=%v apply_calls=%d/%d",
			err, applyCalls, system.store.applyCalls)
	}
	record, getErr := system.service.GetIntake(context.Background(), GetIntakeRequest{
		ActorRef: system.actor, ProjectRef: system.project, StateRef: "intake:shared",
	})
	if getErr != nil || record.State.Revision() != 1 {
		t.Fatalf("invalid write changed state revision=%d err=%v", record.State.Revision(), getErr)
	}
}

func TestIntakeServiceRejectsAuthorizationOutsideExactActorProjectScope(t *testing.T) {
	system := newIntakeTestSystem(t)
	otherActor, err := goal.NewActorRef("actor:intake-other")
	if err != nil {
		t.Fatal(err)
	}
	_, err = system.service.CreateIntake(context.Background(), CreateIntakeRequest{
		RequestRef: "request:foreign-authorization", ActorRef: otherActor,
		ProjectRef: system.project, StateRef: "intake:foreign",
		Policy:               intake.Policy{MaxQuestionRounds: 3},
		AuthorizationReceipt: system.authorizationFor(t, IntakeOperationCreate, "request:foreign-authorization"),
	})
	if !errors.Is(err, ErrForbidden) ||
		system.store.replayCalls != 0 || system.store.createCalls != 0 {
		t.Fatalf("foreign authorization err=%v replay=%d create=%d",
			err, system.store.replayCalls, system.store.createCalls)
	}
}

func TestIntakeServiceRejectsAuthorizationForAnotherMutationBeforeStore(t *testing.T) {
	system := newIntakeTestSystem(t)
	authorization := system.authorizationFor(
		t, IntakeOperationCreate, "request:intake-another-mutation",
	)
	_, err := system.service.CreateIntake(context.Background(), CreateIntakeRequest{
		RequestRef: "request:intake-bound-mutation", ActorRef: system.actor,
		ProjectRef: system.project, StateRef: "intake:bound-mutation",
		Policy:               intake.Policy{MaxQuestionRounds: 3},
		AuthorizationReceipt: authorization,
	})
	if !errors.Is(err, ErrForbidden) ||
		system.store.replayCalls != 0 ||
		system.store.createCalls != 0 ||
		system.store.applyCalls != 0 {
		t.Fatalf("foreign mutation authorization err=%v replay=%d create=%d apply=%d",
			err, system.store.replayCalls, system.store.createCalls, system.store.applyCalls)
	}

	authorization = system.authorizationFor(
		t, IntakeOperationCreate, "request:intake-bound-apply",
	)
	_, err = system.service.ApplyIntake(context.Background(), ApplyIntakeRequest{
		RequestRef: "request:intake-bound-apply", ActorRef: system.actor,
		ProjectRef: system.project, Change: intakeQuestionChange(1, intake.OriginChat),
		AuthorizationReceipt: authorization,
	})
	if !errors.Is(err, ErrForbidden) ||
		system.store.replayCalls != 0 ||
		system.store.createCalls != 0 ||
		system.store.applyCalls != 0 {
		t.Fatalf("foreign operation authorization err=%v replay=%d create=%d apply=%d",
			err, system.store.replayCalls, system.store.createCalls, system.store.applyCalls)
	}
}

func TestIntakeServiceGetReturnsTypedInvalidRefBeforeStoreRead(t *testing.T) {
	system := newIntakeTestSystem(t)
	_, err := system.service.GetIntake(context.Background(), GetIntakeRequest{
		ActorRef: system.actor, ProjectRef: system.project, StateRef: "form:not-intake",
	})
	if intake.ErrorCodeOf(err) != intake.ErrorInvalidRef || system.store.getCalls != 0 {
		t.Fatalf("invalid get err=%v code=%q reads=%d",
			err, intake.ErrorCodeOf(err), system.store.getCalls)
	}
}

func TestIntakeSnapshotRestoresOnlyThroughValidatedReplay(t *testing.T) {
	system := newIntakeTestSystem(t)
	mustCreateIntake(t, system)
	result, err := system.service.ApplyIntake(context.Background(), ApplyIntakeRequest{
		RequestRef: "request:intake-snapshot", ActorRef: system.actor,
		ProjectRef: system.project, Change: intakeQuestionChange(1, intake.OriginChat),
		AuthorizationReceipt: system.authorizationFor(t, IntakeOperationApply, "request:intake-snapshot"),
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := SnapshotIntake(result.Record.State)
	restored, err := RestoreIntake(snapshot)
	if err != nil || !reflectIntakeStateEqual(restored, result.Record.State) {
		t.Fatalf("restore err=%v state=%+v", err, SnapshotIntake(restored))
	}
	tampered := snapshot
	tampered.History = append([]intake.Mutation(nil), snapshot.History...)
	tampered.History[0].QuestionRound = 2
	if _, err = RestoreIntake(tampered); !IsStateError(err, StateInvalid) {
		t.Fatalf("tampered snapshot err=%v", err)
	}
	overflow := snapshot
	overflow.History = append([]intake.Mutation(nil), snapshot.History...)
	overflow.History[0].IssuesAdded = int(^uint(0) >> 1)
	if _, err = RestoreIntake(overflow); !IsStateError(err, StateInvalid) {
		t.Fatalf("overflow snapshot err=%v", err)
	}
}

type intakeTestSystem struct {
	store   *memoryIntakeStore
	service *IntakeService
	actor   goal.ActorRef
	project goal.ProjectRef
}

func newIntakeTestSystem(t *testing.T) intakeTestSystem {
	t.Helper()
	store := newMemoryIntakeStore()
	service, err := NewIntakeService(store)
	if err != nil {
		t.Fatal(err)
	}
	actor, err := goal.NewActorRef("actor:intake-test")
	if err != nil {
		t.Fatal(err)
	}
	project, err := goal.NewProjectRef("project:intake-test")
	if err != nil {
		t.Fatal(err)
	}
	return intakeTestSystem{
		store: store, service: service, actor: actor, project: project,
	}
}

func (system intakeTestSystem) authorizationFor(
	t *testing.T,
	operation IntakeOperation,
	requestRef string,
) identity.AuthorizationReceipt {
	t.Helper()
	return intakeAuthorization(t, system.actor, system.project, operation, requestRef)
}

func mustCreateIntake(t *testing.T, system intakeTestSystem) IntakeResult {
	t.Helper()
	result, err := system.service.CreateIntake(context.Background(), CreateIntakeRequest{
		RequestRef: "request:intake-create", ActorRef: system.actor,
		ProjectRef: system.project, StateRef: "intake:shared",
		Policy:               intake.Policy{MaxQuestionRounds: 3},
		AuthorizationReceipt: system.authorizationFor(t, IntakeOperationCreate, "request:intake-create"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func intakeQuestionChange(revision intake.Revision, origin intake.Origin) intake.Change {
	return intake.Change{
		StateRef: "intake:shared", ExpectedRevision: revision, Origin: origin,
		Issues: []intake.Issue{{
			Ref: "intake-issue:audience", Kind: intake.IssueGap,
			Field: "audience", DetailKey: "intake.issue.audience.missing",
		}},
		Questions: []intake.Question{{
			Ref: "intake-question:audience",
			DerivedFrom: []intake.IssueRef{
				"intake-issue:audience",
			},
			PromptKey: "intake.question.audience.prompt",
			WhyKey:    "intake.question.audience.why",
			Options: []intake.Option{
				{
					Ref:          "intake-option:audience-team",
					LabelKey:     "intake.option.audience.team.label",
					RationaleKey: "intake.option.audience.team.rationale",
					Recommended:  true,
				},
				{
					Ref:          "intake-option:audience-personal",
					LabelKey:     "intake.option.audience.personal.label",
					RationaleKey: "intake.option.audience.personal.rationale",
				},
			},
		}},
	}
}

func intakeStateKey(
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	stateRef intake.Ref,
) string {
	return actorRef.String() + "\x00" + projectRef.String() + "\x00" + string(stateRef)
}

func intakeRequestKey(
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	requestRef string,
) string {
	return actorRef.String() + "\x00" + projectRef.String() + "\x00" + requestRef
}

func intakeAuthorization(
	t *testing.T,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	operation IntakeOperation,
	mutationRequestRef string,
) identity.AuthorizationReceipt {
	t.Helper()
	principalRef, err := identity.NewPrincipalRef("principal:intake-test")
	if err != nil {
		t.Fatal(err)
	}
	principal, err := identity.NewPrincipal(
		principalRef, actorRef, identity.PrincipalKindHuman, "test",
	)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 7, 26, 8, 0, 0, 0, time.UTC)
	authorizationRequestRef, err := IntakeAuthorizationRequestRef(operation, mutationRequestRef)
	if err != nil {
		t.Fatal(err)
	}
	request, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: authorizationRequestRef,
		Principal:  principal, ProjectRef: projectRef,
		Permission:  identity.PermissionGoalsCreate,
		ResourceRef: projectRef.String(), RequestedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := identity.NewAuthorizationDecision(identity.AuthorizationDecisionInput{
		Request: request, Outcome: identity.AuthorizationAllowed,
		Role: identity.RoleProjectOwner, MembershipRevision: 1,
		ReasonCode: "allowed", DecidedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := identity.NewAuthorizationReceipt(identity.AuthorizationReceiptInput{
		Ref:      "authorization-receipt:" + authorizationRequestRef,
		Decision: decision, RecordedAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

var _ IntakeStore = (*memoryIntakeStore)(nil)
