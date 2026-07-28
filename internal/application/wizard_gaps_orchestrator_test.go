package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/intake"
	"orquesta/internal/wizard/gaps"
)

func TestOrchestratorWizardGapsBindsAuthenticatedScopeAndSharedWriter(t *testing.T) {
	system := newIntakeOrchestratorTestSystem(t)
	var err error
	system.orchestrator.wizardGaps, err = NewWizardGapsService(
		system.store,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = system.orchestrator.CreateIntake(
		context.Background(),
		system.access,
		CreateIntakeRequest{
			RequestRef: "request:wizard-gaps-orchestrator-create",
			StateRef:   "intake:wizard-gaps-orchestrator",
			Policy:     intake.Policy{MaxQuestionRounds: 3},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	spoofedActor, _ := goal.NewActorRef("actor:spoofed")
	spoofedProject, _ := goal.NewProjectRef("project:spoofed")
	result, err := system.orchestrator.ApplyWizardGaps(
		context.Background(),
		system.access,
		ApplyWizardGapsRequest{
			RequestRef:        "request:wizard-gaps-orchestrator-apply",
			ActorRef:          spoofedActor,
			ProjectRef:        spoofedProject,
			StateRef:          "intake:wizard-gaps-orchestrator",
			ExpectedRevision:  1,
			Origin:            intake.OriginForm,
			EvaluatorIdentity: gaps.EvaluatorV1Identity(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Record.ActorRef != system.actor ||
		result.Record.ProjectRef != system.project ||
		result.Record.State.Revision() != 2 ||
		result.Record.Receipt.AuthorizationReceiptRef == "" {
		t.Fatalf("result=%+v", result)
	}
	if _, found := system.store.current[intakeStateKey(
		spoofedActor,
		spoofedProject,
		"intake:wizard-gaps-orchestrator",
	)]; found {
		t.Fatal("spoofed actor/project reached shared Intake writer")
	}
}

func TestOrchestratorDeniedWizardGapsDoesNotReachSharedWriter(t *testing.T) {
	system := newIntakeOrchestratorTestSystem(t)
	var err error
	system.orchestrator.wizardGaps, err = NewWizardGapsService(
		system.store,
	)
	if err != nil {
		t.Fatal(err)
	}
	system.accessRepository.defaultRole = ""
	_, err = system.orchestrator.ApplyWizardGaps(
		context.Background(),
		system.access,
		ApplyWizardGapsRequest{
			RequestRef:        "request:wizard-gaps-orchestrator-denied",
			StateRef:          "intake:wizard-gaps-denied",
			ExpectedRevision:  1,
			Origin:            intake.OriginForm,
			EvaluatorIdentity: gaps.EvaluatorV1Identity(),
		},
	)
	if !errors.Is(err, ErrForbidden) ||
		system.store.getCalls != 0 || system.store.applyCalls != 0 {
		t.Fatalf(
			"denied err=%v reads=%d applies=%d",
			err,
			system.store.getCalls,
			system.store.applyCalls,
		)
	}
}

func TestOrchestratorWithoutIntakeStoreRejectsWizardGaps(t *testing.T) {
	clock := &mutableClock{
		now: time.Date(2026, 7, 28, 9, 0, 0, 0, time.UTC),
	}
	orchestrator, _ := newTestOrchestrator(
		t,
		newMemoryRepository(),
		clock,
		&scriptedAgent{now: clock.Now},
	)
	actorRef, projectRef := testScope(t)
	_, err := orchestrator.ApplyWizardGaps(
		context.Background(),
		accessForScope(t, actorRef, projectRef),
		ApplyWizardGapsRequest{},
	)
	if err == nil || err.Error() != "application.unavailable" {
		t.Fatalf("unavailable err=%v", err)
	}
}

func TestOrchestratorRejectsPartialWizardGapsStoreComposition(t *testing.T) {
	store := newMemoryIntakeStore()
	for name, dependencies := range map[string]Dependencies{
		"split_brain": {
			IntakeStore: store, WizardGapsStore: newMemoryIntakeStore(),
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := New(dependencies)
			if err == nil ||
				err.Error() != "application.wizard_gaps_store_composition_invalid" {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
