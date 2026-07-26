package application

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
)

type dossierIntakeStore struct {
	mu      sync.RWMutex
	current IntakeRecord
	gets    atomic.Int64
}

func (store *dossierIntakeStore) ReplayIntake(
	context.Context,
	IntakeReplayRequest,
) (IntakeRecord, bool, error) {
	return IntakeRecord{}, false, errors.New("test.unused")
}

func (store *dossierIntakeStore) CreateIntake(
	context.Context,
	IntakeCreateState,
) (IntakeRecord, bool, error) {
	return IntakeRecord{}, false, errors.New("test.unused")
}

func (store *dossierIntakeStore) GetIntake(
	_ context.Context,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	stateRef intake.Ref,
) (IntakeRecord, error) {
	store.gets.Add(1)
	store.mu.RLock()
	defer store.mu.RUnlock()
	if store.current.ActorRef != actorRef || store.current.ProjectRef != projectRef ||
		store.current.State.Ref() != stateRef {
		return IntakeRecord{}, &StateError{Code: StateNotFound}
	}
	return store.current, nil
}

func (store *dossierIntakeStore) ApplyIntake(
	context.Context,
	IntakeApplyState,
) (IntakeRecord, bool, error) {
	return IntakeRecord{}, false, errors.New("test.unused")
}

func (store *dossierIntakeStore) peek() IntakeRecord {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.current
}

type memoryIntakeDossierStore struct {
	mu             sync.Mutex
	intakes        *dossierIntakeStore
	requests       map[string]IntakeDossierRecord
	records        map[string]IntakeDossierRecord
	replayOverride *IntakeDossierRecord
	replays        atomic.Int64
	creates        atomic.Int64
	gets           atomic.Int64
}

func newMemoryIntakeDossierStore(intakes *dossierIntakeStore) *memoryIntakeDossierStore {
	return &memoryIntakeDossierStore{
		intakes: intakes, requests: make(map[string]IntakeDossierRecord),
		records: make(map[string]IntakeDossierRecord),
	}
}

func (store *memoryIntakeDossierStore) ReplayIntakeDossier(
	_ context.Context,
	request IntakeDossierReplayRequest,
) (IntakeDossierRecord, bool, error) {
	store.replays.Add(1)
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.replayLocked(request)
}

func (store *memoryIntakeDossierStore) replayLocked(
	request IntakeDossierReplayRequest,
) (IntakeDossierRecord, bool, error) {
	if store.replayOverride != nil {
		return *store.replayOverride, true, nil
	}
	record, found := store.requests[intakeDossierRequestKey(
		request.ActorRef, request.ProjectRef, request.RequestRef,
	)]
	if !found {
		return IntakeDossierRecord{}, false, nil
	}
	receipt := record.Receipt
	if receipt.RequestFingerprint != request.RequestFingerprint ||
		receipt.StateRef != request.StateRef ||
		receipt.StateRevision != request.ExpectedRevision ||
		receipt.SourceIntakeReceiptRef != request.SourceIntakeReceiptRef ||
		receipt.PlanDigest != request.PlanDigest ||
		receipt.AuthorizationReceiptRef != request.AuthorizationReceiptRef ||
		(request.DossierRef != "" && receipt.DossierRef != request.DossierRef) {
		return IntakeDossierRecord{}, false, &StateError{Code: StateConflict}
	}
	return record, true, nil
}

func (store *memoryIntakeDossierStore) CreateIntakeDossier(
	_ context.Context,
	state IntakeDossierCreateState,
) (IntakeDossierRecord, bool, error) {
	store.creates.Add(1)
	store.mu.Lock()
	defer store.mu.Unlock()
	replay := IntakeDossierReplayRequest{
		RequestRef: state.RequestRef, RequestFingerprint: state.RequestFingerprint,
		ActorRef: state.ActorRef, ProjectRef: state.ProjectRef,
		DossierRef: state.Dossier.Ref(), StateRef: state.Dossier.StateRef(),
		ExpectedRevision:        state.ExpectedRevision,
		SourceIntakeReceiptRef:  state.SourceIntakeReceiptRef,
		PlanDigest:              state.Dossier.PlanDigest(),
		AuthorizationReceiptRef: state.AuthorizationReceipt.Ref(),
	}
	if record, found, err := store.replayLocked(replay); err != nil || found {
		return record, false, err
	}
	current := store.intakes.peek()
	if current.ActorRef != state.ActorRef || current.ProjectRef != state.ProjectRef ||
		current.State.Ref() != state.Dossier.StateRef() ||
		current.State.Revision() != state.ExpectedRevision ||
		current.Receipt.Ref != state.SourceIntakeReceiptRef {
		return IntakeDossierRecord{}, false, &StateError{Code: StateConflict}
	}
	record := IntakeDossierRecord{
		ActorRef: state.ActorRef, ProjectRef: state.ProjectRef,
		Dossier: state.Dossier, Receipt: state.Receipt,
	}
	store.requests[intakeDossierRequestKey(
		state.ActorRef, state.ProjectRef, state.RequestRef,
	)] = record
	recordKey := intakeDossierRecordKey(
		state.ActorRef, state.ProjectRef, state.Dossier.Ref(),
	)
	if _, found := store.records[recordKey]; !found {
		store.records[recordKey] = record
	}
	return record, true, nil
}

func (store *memoryIntakeDossierStore) GetIntakeDossier(
	_ context.Context,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	dossierRef IntakeDossierRef,
) (IntakeDossierRecord, error) {
	store.gets.Add(1)
	store.mu.Lock()
	defer store.mu.Unlock()
	record, found := store.records[intakeDossierRecordKey(actorRef, projectRef, dossierRef)]
	if !found {
		return IntakeDossierRecord{}, &StateError{Code: StateNotFound}
	}
	return record, nil
}

func TestIntakeDossierServiceExactReplaySurvivesLaterIntakeWithoutReread(t *testing.T) {
	system := newIntakeDossierTestSystem(t)
	request := system.request(t, "request:intake-dossier-race")
	first, err := system.service.PrepareIntakeDossier(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Created {
		t.Fatal("first preparation was not created")
	}
	if got := IntakeDossierGenerationFingerprint(
		first.Record.Dossier, request.AuthorizationReceipt.Ref(),
	); got != first.Record.Receipt.RequestFingerprint {
		t.Fatalf("recovery fingerprint = %q want %q", got, first.Record.Receipt.RequestFingerprint)
	}

	system.advanceIntake(t)
	system.intakes.gets.Store(0)
	const callers = 32
	results := make(chan IntakeDossierResult, callers)
	errs := make(chan error, callers)
	var group sync.WaitGroup
	group.Add(callers)
	for range callers {
		go func() {
			defer group.Done()
			result, replayErr := system.service.PrepareIntakeDossier(
				context.Background(), request,
			)
			results <- result
			errs <- replayErr
		}()
	}
	group.Wait()
	close(results)
	close(errs)
	for replayErr := range errs {
		if replayErr != nil {
			t.Errorf("concurrent replay: %v", replayErr)
		}
	}
	for result := range results {
		if result.Created || result.Record.Receipt != first.Record.Receipt ||
			result.Record.Dossier.Ref() != first.Record.Dossier.Ref() {
			t.Errorf("non-exact replay: %+v", result)
		}
	}
	if gets := system.intakes.gets.Load(); gets != 0 {
		t.Fatalf("exact replay reread current intake %d times", gets)
	}
	if creates := system.dossiers.creates.Load(); creates != 1 {
		t.Fatalf("exact replay entered create %d times", creates)
	}
}

func TestIntakeDossierServiceConcurrentFirstPreparationCreatesOneReceipt(t *testing.T) {
	system := newIntakeDossierTestSystem(t)
	request := system.request(t, "request:intake-dossier-first-race")
	const callers = 32
	results := make(chan IntakeDossierResult, callers)
	errs := make(chan error, callers)
	var group sync.WaitGroup
	group.Add(callers)
	for range callers {
		go func() {
			defer group.Done()
			result, err := system.service.PrepareIntakeDossier(context.Background(), request)
			results <- result
			errs <- err
		}()
	}
	group.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("concurrent first preparation: %v", err)
		}
	}
	var created int
	var receipt IntakeDossierGenerationReceipt
	for result := range results {
		if result.Created {
			created++
		}
		if receipt.Ref == "" {
			receipt = result.Record.Receipt
		} else if result.Record.Receipt != receipt {
			t.Errorf("race returned divergent receipt: got=%+v want=%+v",
				result.Record.Receipt, receipt)
		}
	}
	if created != 1 || system.dossiers.creates.Load() == 0 {
		t.Fatalf("race created=%d create_calls=%d", created, system.dossiers.creates.Load())
	}
}

func TestIntakeDossierServiceRejectsDivergentReplayAndStaleSource(t *testing.T) {
	system := newIntakeDossierTestSystem(t)
	request := system.request(t, "request:intake-dossier-divergent")
	if _, err := system.service.PrepareIntakeDossier(context.Background(), request); err != nil {
		t.Fatal(err)
	}

	system.intakes.gets.Store(0)
	creates := system.dossiers.creates.Load()
	divergent := request
	divergent.Input = validIntakeDossierInput()
	divergent.Input.Sections[0].Markdown += " changed"
	if _, err := system.service.PrepareIntakeDossier(
		context.Background(), divergent,
	); !IsStateError(err, StateConflict) {
		t.Fatalf("divergent replay error = %v", err)
	}
	if system.intakes.gets.Load() != 0 || system.dossiers.creates.Load() != creates {
		t.Fatal("divergent replay reached intake or write")
	}

	records := validIntakeChain(t)
	stale := system.request(t, "request:intake-dossier-stale")
	stale.ExpectedRevision = records[1].State.Revision()
	stale.SourceIntakeReceiptRef = records[1].Receipt.Ref
	gets := system.intakes.gets.Load()
	if _, err := system.service.PrepareIntakeDossier(
		context.Background(), stale,
	); !IsStateError(err, StateConflict) {
		t.Fatalf("stale preparation error = %v", err)
	}
	if system.intakes.gets.Load() != gets+1 ||
		system.dossiers.creates.Load() != creates {
		t.Fatal("stale source was written")
	}
}

func TestIntakeDossierServiceRequiresRequestSpecificGoalsCreateAuthorization(t *testing.T) {
	system := newIntakeDossierTestSystem(t)
	request := system.request(t, "request:intake-dossier-authorized")
	request.RequestRef = "request:intake-dossier-other"
	if _, err := system.service.PrepareIntakeDossier(
		context.Background(), request,
	); !errors.Is(err, ErrForbidden) {
		t.Fatalf("reused authorization error = %v", err)
	}
	if system.intakes.gets.Load() != 0 || system.dossiers.creates.Load() != 0 {
		t.Fatal("unauthorized request reached state")
	}
}

func TestIntakeDossierServiceGetReturnsValidatedDefensiveRecord(t *testing.T) {
	system := newIntakeDossierTestSystem(t)
	request := system.request(t, "request:intake-dossier-get")
	prepared, err := system.service.PrepareIntakeDossier(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	prepared.Record.Dossier.sections[0].Markdown = "first-result mutation"
	got, err := system.service.GetIntakeDossier(context.Background(), GetIntakeDossierRequest{
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		DossierRef: prepared.Record.Dossier.Ref(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Dossier.Sections()[0].Markdown == "first-result mutation" {
		t.Fatal("create result leaked dossier backing slices")
	}
	got.Dossier.sections[0].Markdown = "same-package mutation"
	gotAgain, err := system.service.GetIntakeDossier(context.Background(), GetIntakeDossierRequest{
		ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
		DossierRef: prepared.Record.Dossier.Ref(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotAgain.Dossier.Sections()[0].Markdown == "same-package mutation" {
		t.Fatal("get leaked dossier backing slices")
	}
}

func TestIntakeDossierServiceGetKeepsFirstCanonicalGenerationReceipt(t *testing.T) {
	system := newIntakeDossierTestSystem(t)
	firstRequest := system.request(t, "request:intake-dossier-canonical-first")
	first, err := system.service.PrepareIntakeDossier(context.Background(), firstRequest)
	if err != nil {
		t.Fatal(err)
	}
	secondRequest := system.request(t, "request:intake-dossier-canonical-second")
	second, err := system.service.PrepareIntakeDossier(context.Background(), secondRequest)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Created || !second.Created ||
		first.Record.Dossier.Ref() != second.Record.Dossier.Ref() ||
		first.Record.Receipt.Ref == second.Record.Receipt.Ref {
		t.Fatalf("distinct generations invalid: first=%+v second=%+v", first, second)
	}
	canonical, err := system.service.GetIntakeDossier(
		context.Background(),
		GetIntakeDossierRequest{
			ActorRef: firstRequest.ActorRef, ProjectRef: firstRequest.ProjectRef,
			DossierRef: first.Record.Dossier.Ref(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if canonical.Receipt != first.Record.Receipt {
		t.Fatalf("Get replaced first receipt: got=%+v first=%+v second=%+v",
			canonical.Receipt, first.Record.Receipt, second.Record.Receipt)
	}
	replayedSecond, err := system.service.PrepareIntakeDossier(
		context.Background(), secondRequest,
	)
	if err != nil {
		t.Fatal(err)
	}
	if replayedSecond.Created || replayedSecond.Record.Receipt != second.Record.Receipt {
		t.Fatalf("second request lost exact receipt: %+v", replayedSecond)
	}
}

func TestIntakeDossierServiceRejectsCoherentlyRewrittenAdapterFingerprint(t *testing.T) {
	tests := []struct {
		name string
		call func(context.Context, intakeDossierTestSystem, PrepareIntakeDossierRequest, IntakeDossierRef) error
	}{
		{
			name: "get",
			call: func(
				ctx context.Context,
				system intakeDossierTestSystem,
				request PrepareIntakeDossierRequest,
				dossierRef IntakeDossierRef,
			) error {
				_, err := system.service.GetIntakeDossier(ctx, GetIntakeDossierRequest{
					ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
					DossierRef: dossierRef,
				})
				return err
			},
		},
		{
			name: "replay",
			call: func(
				ctx context.Context,
				system intakeDossierTestSystem,
				request PrepareIntakeDossierRequest,
				_ IntakeDossierRef,
			) error {
				_, err := system.service.PrepareIntakeDossier(ctx, request)
				return err
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			system := newIntakeDossierTestSystem(t)
			request := system.request(t, "request:intake-dossier-rewritten-"+test.name)
			prepared, err := system.service.PrepareIntakeDossier(
				context.Background(), request,
			)
			if err != nil {
				t.Fatal(err)
			}
			rewritten := coherentlyRewriteDossierFingerprint(t, prepared.Record)
			system.dossiers.mu.Lock()
			system.dossiers.records[intakeDossierRecordKey(
				request.ActorRef, request.ProjectRef, prepared.Record.Dossier.Ref(),
			)] = rewritten
			system.dossiers.requests[intakeDossierRequestKey(
				request.ActorRef, request.ProjectRef, request.RequestRef,
			)] = rewritten
			if test.name == "replay" {
				system.dossiers.replayOverride = &rewritten
			}
			system.dossiers.mu.Unlock()

			if err := test.call(
				context.Background(), system, request, prepared.Record.Dossier.Ref(),
			); !IsStateError(err, StateConflict) {
				t.Fatalf("coherent fingerprint rewrite error = %v", err)
			}
		})
	}
}

type intakeDossierTestSystem struct {
	service  *IntakeDossierService
	intakes  *dossierIntakeStore
	dossiers *memoryIntakeDossierStore
	record   IntakeRecord
}

func newIntakeDossierTestSystem(t *testing.T) intakeDossierTestSystem {
	t.Helper()
	record := validIntakeChain(t)[2]
	intakes := &dossierIntakeStore{current: record}
	dossiers := newMemoryIntakeDossierStore(intakes)
	service, err := NewIntakeDossierService(intakes, dossiers)
	if err != nil {
		t.Fatal(err)
	}
	return intakeDossierTestSystem{
		service: service, intakes: intakes, dossiers: dossiers, record: record,
	}
}

func (system intakeDossierTestSystem) request(
	t *testing.T,
	requestRef string,
) PrepareIntakeDossierRequest {
	t.Helper()
	return PrepareIntakeDossierRequest{
		RequestRef: requestRef,
		ActorRef:   system.record.ActorRef, ProjectRef: system.record.ProjectRef,
		StateRef:               system.record.State.Ref(),
		ExpectedRevision:       system.record.State.Revision(),
		SourceIntakeReceiptRef: system.record.Receipt.Ref,
		Plan:                   validIntakeDossierPlan(), Input: validIntakeDossierInput(),
		AuthorizationReceipt: intakeDossierAuthorization(
			t, system.record.ActorRef, system.record.ProjectRef, requestRef,
		),
	}
}

func (system intakeDossierTestSystem) advanceIntake(t *testing.T) {
	t.Helper()
	current := system.intakes.peek()
	change := intake.Change{
		StateRef: current.State.Ref(), ExpectedRevision: current.State.Revision(),
		Origin: intake.OriginChat,
		Issues: []intake.Issue{{
			Ref: "intake-issue:later", Kind: intake.IssueGap,
			Field: "later", DetailKey: "intake.issue.later.missing",
		}},
		Questions: []intake.Question{{
			Ref:         "intake-question:later",
			DerivedFrom: []intake.IssueRef{"intake-issue:later"},
			PromptKey:   "intake.question.later.prompt",
			WhyKey:      "intake.question.later.why",
			Options: []intake.Option{
				{
					Ref:          "intake-option:later-yes",
					LabelKey:     "intake.option.later.yes.label",
					RationaleKey: "intake.option.later.yes.rationale",
					Recommended:  true,
				},
				{
					Ref:          "intake-option:later-no",
					LabelKey:     "intake.option.later.no.label",
					RationaleKey: "intake.option.later.no.rationale",
				},
			},
		}},
	}
	next, err := intake.Apply(current.State, change)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(change)
	if err != nil {
		t.Fatal(err)
	}
	const authorizationReceiptRef = "authorization-receipt:intake-later"
	fingerprint := fingerprintFields(
		"orquesta.intake.apply.v1",
		current.ActorRef.String(), current.ProjectRef.String(),
		string(encoded), authorizationReceiptRef,
	)
	receipt, err := buildIntakeReceipt(
		IntakeOperationApply, "request:intake-later", fingerprint,
		current.ActorRef, current.ProjectRef, next, current.State.Revision(),
		authorizationReceiptRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	later := IntakeRecord{
		ActorRef: current.ActorRef, ProjectRef: current.ProjectRef,
		State: next, Receipt: receipt,
	}
	if err := validateStoredIntakeRecord(
		later.ActorRef, later.ProjectRef, later.State.Ref(), later,
	); err != nil {
		t.Fatal(err)
	}
	system.intakes.mu.Lock()
	system.intakes.current = later
	system.intakes.mu.Unlock()
}

func intakeDossierAuthorization(
	t *testing.T,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	requestRef string,
) identity.AuthorizationReceipt {
	t.Helper()
	principalRef, err := identity.NewPrincipalRef("principal:intake-dossier-test")
	if err != nil {
		t.Fatal(err)
	}
	principal, err := identity.NewPrincipal(
		principalRef, actorRef, identity.PrincipalKindHuman, "test",
	)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	authorizationRequestRef, err := IntakeDossierAuthorizationRequestRef(requestRef)
	if err != nil {
		t.Fatal(err)
	}
	request, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: authorizationRequestRef, Principal: principal,
		ProjectRef: projectRef, Permission: identity.PermissionGoalsCreate,
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

func coherentlyRewriteDossierFingerprint(
	t *testing.T,
	record IntakeDossierRecord,
) IntakeDossierRecord {
	t.Helper()
	rewrittenFingerprint := fingerprintFields(
		"test.adapter.rewritten-intake-dossier-fingerprint.v1",
		record.Receipt.RequestFingerprint,
	)
	if rewrittenFingerprint == record.Receipt.RequestFingerprint {
		t.Fatal("test did not rewrite fingerprint")
	}
	receipt, err := BuildIntakeDossierGenerationReceipt(
		record.Receipt.RequestRef, rewrittenFingerprint,
		record.Dossier, record.Receipt.AuthorizationReceiptRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	record.Receipt = receipt
	return record
}

func intakeDossierRequestKey(
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	requestRef string,
) string {
	return actorRef.String() + "\x00" + projectRef.String() + "\x00" + requestRef
}

func intakeDossierRecordKey(
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	dossierRef IntakeDossierRef,
) string {
	return actorRef.String() + "\x00" + projectRef.String() + "\x00" + string(dossierRef)
}

var _ IntakeStore = (*dossierIntakeStore)(nil)
var _ IntakeDossierStore = (*memoryIntakeDossierStore)(nil)
