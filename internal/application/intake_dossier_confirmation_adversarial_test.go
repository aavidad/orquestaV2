package application

import (
	"context"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
)

type adversarialIntakeDossierConfirmationRepository struct {
	StateRepository

	mu      sync.Mutex
	records map[string]IntakeDossierConfirmationRecord
	calls   int
	rewrite func(*IntakeDossierConfirmationRecord)
}

func (repository *adversarialIntakeDossierConfirmationRepository) ConfirmIntakeDossierAndCreateGoal(
	_ context.Context,
	state ConfirmIntakeDossierState,
) (IntakeDossierConfirmationRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.calls++
	key := state.CreateGoal.RequestedBy.String() + "\x00" +
		state.CreateGoal.Goal.Project().String() + "\x00" +
		state.CreateGoal.RequestRef
	if record, found := repository.records[key]; found {
		if record.Confirmation.RequestFingerprint !=
			state.Confirmation.RequestFingerprint {
			return IntakeDossierConfirmationRecord{}, false,
				&StateError{Code: StateConflict}
		}
		if repository.rewrite != nil {
			repository.rewrite(&record)
		}
		return record, false, nil
	}
	record := IntakeDossierConfirmationRecord{
		Goal: GoalRecord{
			RequestRef:         state.CreateGoal.RequestRef,
			RequestFingerprint: state.CreateGoal.RequestFingerprint,
			RequestedBy:        state.CreateGoal.RequestedBy,
			Goal:               state.CreateGoal.Goal,
			Executions: append(
				[]ExecutionRecord(nil), state.CreateGoal.Executions...,
			),
			BudgetEnvelopes: append(
				[]governance.BudgetEnvelope(nil),
				state.CreateGoal.BudgetEnvelopes...,
			),
			WorkItemAuthorities: append(
				[]WorkItemAuthority(nil),
				state.CreateGoal.WorkItemAuthorities...,
			),
		},
		Confirmation: state.Confirmation,
	}
	for _, action := range state.CreateGoal.Actions {
		if action.EffectIntent.Ref != "" {
			record.Goal.EffectIntents = append(
				record.Goal.EffectIntents, action.EffectIntent,
			)
		}
		if action.EffectApproval != nil {
			record.Goal.EffectApprovals = append(
				record.Goal.EffectApprovals, *action.EffectApproval,
			)
		}
	}
	repository.records[key] = record
	if repository.rewrite != nil {
		repository.rewrite(&record)
	}
	return record, true, nil
}

func (repository *adversarialIntakeDossierConfirmationRepository) callCount() int {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	return repository.calls
}

func TestV23ConfirmIntakeDossierFalseHasNoAuthorizationClockIDOrStateEffect(
	t *testing.T,
) {
	system := newIntakeDossierOrchestratorTestSystem(t)
	clock := &v04CountingClock{
		now: time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC),
	}
	ids := &sequentialIDs{}
	repository := &adversarialIntakeDossierConfirmationRepository{
		StateRepository: system.orchestrator.state,
		records:         make(map[string]IntakeDossierConfirmationRecord),
	}
	system.orchestrator.clock = clock
	system.orchestrator.ids = ids
	system.orchestrator.state = repository

	_, err := system.orchestrator.ConfirmIntakeDossier(
		context.Background(),
		system.access,
		ConfirmIntakeDossierRequest{Confirm: false},
	)
	if err == nil || err.Error() != "application.confirmation_required" {
		t.Fatalf("confirm=false error = %v", err)
	}
	if authorizations := len(intakeAuthorizations(system.accessRepository)); authorizations != 0 {
		t.Fatalf("confirm=false authorizations = %d", authorizations)
	}
	if clock.calls != 0 || ids.next != 0 || repository.callCount() != 0 ||
		system.dossiers.gets.Load() != 0 {
		t.Fatalf(
			"confirm=false effects: clock=%d ids=%d state=%d dossier_gets=%d",
			clock.calls, ids.next, repository.callCount(),
			system.dossiers.gets.Load(),
		)
	}
}

func TestV23ConfirmIntakeDossierSameRequestRejectsDifferentDossier(t *testing.T) {
	system := newIntakeDossierOrchestratorTestSystem(t)
	first, second := prepareAdversarialConfirmationDossiers(t, &system)
	repository := &adversarialIntakeDossierConfirmationRepository{
		StateRepository: system.orchestrator.state,
		records:         make(map[string]IntakeDossierConfirmationRecord),
	}
	system.orchestrator.state = repository
	request := ConfirmIntakeDossierRequest{
		RequestRef: "request:confirm-dossier-divergent",
		DossierRef: first,
		Confirm:    true,
	}

	confirmed, err := system.orchestrator.ConfirmIntakeDossier(
		context.Background(), system.access, request,
	)
	if err != nil || !confirmed.Created ||
		confirmed.Confirmation.DossierRef != first {
		t.Fatalf("first confirmation = %+v err=%v", confirmed, err)
	}
	request.DossierRef = second
	if _, err := system.orchestrator.ConfirmIntakeDossier(
		context.Background(), system.access, request,
	); !IsStateError(err, StateConflict) {
		t.Fatalf("divergent dossier replay error = %v", err)
	}
	if repository.callCount() != 2 {
		t.Fatalf("confirmation state calls = %d", repository.callCount())
	}
}

func TestV23ConfirmIntakeDossierRejectsSubstitutedPersistedBindings(t *testing.T) {
	tests := []struct {
		name    string
		rewrite func(IntakeDossierRef) func(*IntakeDossierConfirmationRecord)
	}{
		{
			name: "dossier",
			rewrite: func(other IntakeDossierRef) func(*IntakeDossierConfirmationRecord) {
				return func(record *IntakeDossierConfirmationRecord) {
					record.Confirmation.DossierRef = other
				}
			},
		},
		{
			name: "goal",
			rewrite: func(IntakeDossierRef) func(*IntakeDossierConfirmationRecord) {
				return func(record *IntakeDossierConfirmationRecord) {
					record.Confirmation.GoalRef, _ =
						goal.NewGoalRef("goal:substituted-confirmation")
				}
			},
		},
		{
			name: "spec_hash",
			rewrite: func(IntakeDossierRef) func(*IntakeDossierConfirmationRecord) {
				return func(record *IntakeDossierConfirmationRecord) {
					record.Confirmation.SpecHash =
						"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				}
			},
		},
		{
			name: "authorization",
			rewrite: func(IntakeDossierRef) func(*IntakeDossierConfirmationRecord) {
				return func(record *IntakeDossierConfirmationRecord) {
					record.Confirmation.AuthorizationReceiptRef =
						"authorization-receipt:substituted"
				}
			},
		},
		{
			name: "receipt",
			rewrite: func(IntakeDossierRef) func(*IntakeDossierConfirmationRecord) {
				return func(record *IntakeDossierConfirmationRecord) {
					last := len(record.Confirmation.Ref) - 1
					replacement := byte('0')
					if record.Confirmation.Ref[last] == replacement {
						replacement = '1'
					}
					record.Confirmation.Ref =
						record.Confirmation.Ref[:last] + string(replacement)
				}
			},
		},
		{
			name: "executions",
			rewrite: func(IntakeDossierRef) func(*IntakeDossierConfirmationRecord) {
				return func(record *IntakeDossierConfirmationRecord) {
					record.Goal.Executions = nil
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			system := newIntakeDossierOrchestratorTestSystem(t)
			dossierRef, otherDossierRef :=
				prepareAdversarialConfirmationDossiers(t, &system)
			repository := &adversarialIntakeDossierConfirmationRepository{
				StateRepository: system.orchestrator.state,
				records:         make(map[string]IntakeDossierConfirmationRecord),
				rewrite:         test.rewrite(otherDossierRef),
			}
			system.orchestrator.state = repository

			_, err := system.orchestrator.ConfirmIntakeDossier(
				context.Background(),
				system.access,
				ConfirmIntakeDossierRequest{
					RequestRef: "request:confirm-substituted-" + test.name,
					DossierRef: dossierRef,
					Confirm:    true,
				},
			)
			if !IsStateError(err, StateConflict) {
				t.Fatalf("substituted %s binding accepted: %v", test.name, err)
			}
		})
	}
}

func TestV23ConfirmIntakeDossierReplayRejectsIncompleteLiveRecord(t *testing.T) {
	system := newIntakeDossierOrchestratorTestSystem(t)
	dossierRef, _ := prepareAdversarialConfirmationDossiers(t, &system)
	repository := &adversarialIntakeDossierConfirmationRepository{
		StateRepository: system.orchestrator.state,
		records:         make(map[string]IntakeDossierConfirmationRecord),
	}
	system.orchestrator.state = repository
	request := ConfirmIntakeDossierRequest{
		RequestRef: "request:confirm-incomplete-live-record",
		DossierRef: dossierRef,
		Confirm:    true,
	}
	if result, err := system.orchestrator.ConfirmIntakeDossier(
		context.Background(), system.access, request,
	); err != nil || !result.Created {
		t.Fatalf("initial confirmation=%+v err=%v", result, err)
	}
	repository.rewrite = func(record *IntakeDossierConfirmationRecord) {
		record.Goal.Executions = nil
	}
	if _, err := system.orchestrator.ConfirmIntakeDossier(
		context.Background(), system.access, request,
	); !IsStateError(err, StateConflict) {
		t.Fatalf("incomplete live replay accepted: %v", err)
	}
}

func TestV23GenericSubmitDoesNotCreateDossierConfirmationBinding(t *testing.T) {
	system := newIntakeDossierOrchestratorTestSystem(t)
	repository := &adversarialIntakeDossierConfirmationRepository{
		StateRepository: system.orchestrator.state,
		records:         make(map[string]IntakeDossierConfirmationRecord),
	}
	system.orchestrator.state = repository

	result, err := system.orchestrator.Submit(
		context.Background(),
		system.access,
		SubmitRequest{
			RequestRef: "request:generic-submit-without-dossier",
			Statement:  "generic submission remains compatible",
			Confirm:    true,
		},
	)
	if err != nil || !result.Created {
		t.Fatalf("generic Submit = %+v err=%v", result, err)
	}
	if repository.callCount() != 0 ||
		result.Record.Goal.AppSpec().Reason() != initialAppSpecReason {
		t.Fatalf(
			"generic Submit acquired dossier binding: calls=%d reason=%q",
			repository.callCount(), result.Record.Goal.AppSpec().Reason(),
		)
	}
}

func prepareAdversarialConfirmationDossiers(
	t *testing.T,
	system *intakeDossierOrchestratorTestSystem,
) (IntakeDossierRef, IntakeDossierRef) {
	t.Helper()
	firstRequest := system.dossier.request(
		t, "request:prepare-confirmation-adversarial-first",
	)
	first, err := system.orchestrator.PrepareIntakeDossier(
		context.Background(), system.access, firstRequest,
	)
	if err != nil {
		t.Fatal(err)
	}
	secondRequest := system.dossier.request(
		t, "request:prepare-confirmation-adversarial-second",
	)
	secondRequest.Input.Sections[0].Markdown += " alcance alternativo"
	second, err := system.orchestrator.PrepareIntakeDossier(
		context.Background(), system.access, secondRequest,
	)
	if err != nil {
		t.Fatal(err)
	}
	if first.Record.Dossier.Ref() == second.Record.Dossier.Ref() {
		t.Fatal("adversarial dossiers did not diverge")
	}
	return first.Record.Dossier.Ref(), second.Record.Dossier.Ref()
}
