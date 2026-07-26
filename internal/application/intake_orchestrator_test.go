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

func TestOrchestratorIntakeUsesAuthenticatedScopeInsteadOfSpoofedRequestFields(t *testing.T) {
	ctx := context.Background()
	system := newIntakeOrchestratorTestSystem(t)
	otherActor, _ := goal.NewActorRef("actor:spoofed")
	otherProject, _ := goal.NewProjectRef("project:spoofed")

	created, err := system.orchestrator.CreateIntake(ctx, system.access, CreateIntakeRequest{
		RequestRef: "request:intake-wrapper", ActorRef: otherActor,
		ProjectRef: otherProject, StateRef: "intake:authenticated",
		Policy: intake.Policy{MaxQuestionRounds: 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Record.ActorRef != system.actor ||
		created.Record.ProjectRef != system.project ||
		created.Record.Receipt.AuthorizationReceiptRef == "" {
		t.Fatalf("authenticated create = %+v", created.Record)
	}
	change := intakeQuestionChange(1, intake.OriginChat)
	change.StateRef = "intake:authenticated"
	applied, err := system.orchestrator.ApplyIntake(ctx, system.access, ApplyIntakeRequest{
		RequestRef: "request:intake-wrapper-apply", ActorRef: otherActor,
		ProjectRef: otherProject, Change: change,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := system.orchestrator.GetIntake(ctx, system.access, GetIntakeRequest{
		ActorRef: otherActor, ProjectRef: otherProject, StateRef: "intake:authenticated",
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied.Record.ActorRef != system.actor ||
		applied.Record.ProjectRef != system.project ||
		got.State.Revision() != 2 {
		t.Fatalf("authenticated apply/get = %+v / %+v", applied.Record, got)
	}
	if _, found := system.store.current[intakeStateKey(
		otherActor, otherProject, "intake:authenticated",
	)]; found {
		t.Fatal("spoofed actor/project created an intake identity")
	}

	authorizations := intakeAuthorizations(system.accessRepository)
	if len(authorizations) != 3 {
		t.Fatalf("authorization count = %d", len(authorizations))
	}
	for _, authorization := range authorizations[:2] {
		if authorization.Principal().ActorRef != system.actor ||
			authorization.ProjectRef() != system.project ||
			authorization.Permission() != identity.PermissionGoalsCreate ||
			authorization.ResourceRef() != system.project.String() {
			t.Fatalf("mutation authorization scope = %+v", authorization)
		}
	}
	if authorizations[0].RequestRef() !=
		"authorization-request:intake-create:request:intake-wrapper" ||
		authorizations[1].RequestRef() !=
			"authorization-request:intake-apply:request:intake-wrapper-apply" {
		t.Fatalf("mutation authorization refs = %q / %q",
			authorizations[0].RequestRef(), authorizations[1].RequestRef())
	}
	read := authorizations[2]
	if read.Principal().ActorRef != system.actor ||
		read.ProjectRef() != system.project ||
		read.Permission() != identity.PermissionGoalsGet ||
		read.ResourceRef() != "intake:authenticated" {
		t.Fatalf("read authorization scope = %+v", read)
	}
}

func TestOrchestratorDeniedIntakeMutationDoesNotReachWriter(t *testing.T) {
	system := newIntakeOrchestratorTestSystem(t)
	system.accessRepository.defaultRole = identity.RoleViewer
	_, err := system.orchestrator.CreateIntake(
		context.Background(), system.access, CreateIntakeRequest{
			RequestRef: "request:intake-denied", StateRef: "intake:denied",
			Policy: intake.Policy{MaxQuestionRounds: 3},
		},
	)
	if !errors.Is(err, ErrForbidden) ||
		system.store.replayCalls != 0 || system.store.createCalls != 0 {
		t.Fatalf("denied err=%v replay=%d create=%d",
			err, system.store.replayCalls, system.store.createCalls)
	}
}

func TestOrchestratorGetIntakeRequiresGoalsGetBeforeStoreRead(t *testing.T) {
	system := newIntakeOrchestratorTestSystem(t)
	_, err := system.orchestrator.CreateIntake(
		context.Background(), system.access, CreateIntakeRequest{
			RequestRef: "request:intake-readable", StateRef: "intake:readable",
			Policy: intake.Policy{MaxQuestionRounds: 3},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	beforeReads := system.store.getCalls
	system.accessRepository.defaultRole = ""
	_, err = system.orchestrator.GetIntake(
		context.Background(), system.access,
		GetIntakeRequest{StateRef: "intake:readable"},
	)
	if !IsStateError(err, StateNotFound) || system.store.getCalls != beforeReads {
		t.Fatalf("denied read err=%v reads=%d/%d", err, beforeReads, system.store.getCalls)
	}
}

func TestOrchestratorWithoutIntakeStoreReturnsUnavailable(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 7, 26, 9, 0, 0, 0, time.UTC)}
	orchestrator, _ := newTestOrchestrator(
		t, newMemoryRepository(), clock, &scriptedAgent{now: clock.Now},
	)
	actorRef, projectRef := testScope(t)
	access := accessForScope(t, actorRef, projectRef)
	assertUnavailable := func(err error) {
		t.Helper()
		if err == nil || err.Error() != "application.unavailable" {
			t.Fatalf("unavailable err=%v", err)
		}
	}
	_, err := orchestrator.CreateIntake(context.Background(), access, CreateIntakeRequest{})
	assertUnavailable(err)
	_, err = orchestrator.GetIntake(context.Background(), access, GetIntakeRequest{})
	assertUnavailable(err)
	_, err = orchestrator.ApplyIntake(context.Background(), access, ApplyIntakeRequest{})
	assertUnavailable(err)
}

type intakeOrchestratorTestSystem struct {
	orchestrator     *Orchestrator
	store            *memoryIntakeStore
	accessRepository *memoryAccessRepository
	access           Access
	actor            goal.ActorRef
	project          goal.ProjectRef
}

func newIntakeOrchestratorTestSystem(t *testing.T) intakeOrchestratorTestSystem {
	t.Helper()
	clock := &mutableClock{now: time.Date(2026, 7, 26, 8, 30, 0, 0, time.UTC)}
	accessRepository := newMemoryAccessRepository()
	orchestrator, _ := newTestOrchestratorWithAccess(
		t, newMemoryRepository(), accessRepository, clock,
		&scriptedAgent{now: clock.Now},
	)
	store := newMemoryIntakeStore()
	var err error
	orchestrator.intake, err = NewIntakeService(store)
	if err != nil {
		t.Fatal(err)
	}
	actorRef, projectRef := testScope(t)
	return intakeOrchestratorTestSystem{
		orchestrator: orchestrator, store: store,
		accessRepository: accessRepository,
		access:           accessForScope(t, actorRef, projectRef),
		actor:            actorRef, project: projectRef,
	}
}

func intakeAuthorizations(repository *memoryAccessRepository) []identity.AuthorizationRequest {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	return append([]identity.AuthorizationRequest(nil), repository.authorizations...)
}
