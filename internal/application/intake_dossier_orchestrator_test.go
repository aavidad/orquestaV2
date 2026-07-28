package application

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func (store *dossierIntakeStore) ReplayWizardGapsInput(
	context.Context,
	WizardGapsInputReplayRequest,
) (WizardGapsInputRecord, bool, error) {
	return WizardGapsInputRecord{}, false, errors.New("test.unused")
}

func (store *dossierIntakeStore) ReserveWizardGapsNoOp(
	context.Context,
	WizardGapsNoOpReservation,
) (WizardGapsInputRecord, bool, error) {
	return WizardGapsInputRecord{}, false, errors.New("test.unused")
}

func (store *dossierIntakeStore) ApplyWizardGapsMutation(
	context.Context,
	WizardGapsMutationReservation,
) (WizardGapsInputRecord, bool, error) {
	return WizardGapsInputRecord{}, false, errors.New("test.unused")
}

func TestOrchestratorIntakeDossierUsesAuthenticatedScopeAndExactPermissions(t *testing.T) {
	ctx := context.Background()
	system := newIntakeDossierOrchestratorTestSystem(t)
	request := system.dossier.request(t, "request:intake-dossier-orchestrator")
	spoofedActor, _ := goal.NewActorRef("actor:spoofed")
	spoofedProject, _ := goal.NewProjectRef("project:spoofed")
	request.ActorRef = spoofedActor
	request.ProjectRef = spoofedProject
	request.AuthorizationReceipt = identity.AuthorizationReceipt{}

	prepared, err := system.orchestrator.PrepareIntakeDossier(
		ctx, system.access, request,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !prepared.Created ||
		prepared.Record.ActorRef != system.dossier.record.ActorRef ||
		prepared.Record.ProjectRef != system.dossier.record.ProjectRef {
		t.Fatalf("authenticated preparation = %+v", prepared)
	}
	request.ActorRef = goal.ActorRef{}
	request.ProjectRef = goal.ProjectRef{}
	replayed, err := system.orchestrator.PrepareIntakeDossier(
		ctx, system.access, request,
	)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Created ||
		replayed.Record.Receipt.Ref != prepared.Record.Receipt.Ref ||
		system.dossier.intakes.gets.Load() != 1 ||
		system.dossier.dossiers.creates.Load() != 1 {
		t.Fatalf(
			"replay=%+v intake_gets=%d dossier_creates=%d",
			replayed,
			system.dossier.intakes.gets.Load(),
			system.dossier.dossiers.creates.Load(),
		)
	}

	got, err := system.orchestrator.GetIntakeDossier(
		ctx,
		system.access,
		GetIntakeDossierRequest{
			ActorRef: spoofedActor, ProjectRef: spoofedProject,
			DossierRef: prepared.Record.Dossier.Ref(),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ActorRef != system.dossier.record.ActorRef ||
		got.ProjectRef != system.dossier.record.ProjectRef ||
		got.Dossier.Ref() != prepared.Record.Dossier.Ref() {
		t.Fatalf("authenticated get = %+v", got)
	}

	authorizations := intakeAuthorizations(system.accessRepository)
	if len(authorizations) != 3 {
		t.Fatalf("authorization count = %d", len(authorizations))
	}
	for _, authorization := range authorizations[:2] {
		if authorization.Principal().ActorRef != system.dossier.record.ActorRef ||
			authorization.ProjectRef() != system.dossier.record.ProjectRef ||
			authorization.Permission() != identity.PermissionGoalsCreate ||
			authorization.ResourceRef() != system.dossier.record.ProjectRef.String() ||
			authorization.RequestRef() !=
				"authorization-request:intake-dossier-generate:"+
					request.RequestRef {
			t.Fatalf("preparation authorization = %+v", authorization)
		}
	}
	read := authorizations[2]
	if read.Principal().ActorRef != system.dossier.record.ActorRef ||
		read.ProjectRef() != system.dossier.record.ProjectRef ||
		read.Permission() != identity.PermissionGoalsGet ||
		read.ResourceRef() != string(prepared.Record.Dossier.Ref()) {
		t.Fatalf("read authorization = %+v", read)
	}
}

func TestOrchestratorIntakeDossierDenialStopsBeforeStores(t *testing.T) {
	system := newIntakeDossierOrchestratorTestSystem(t)
	system.accessRepository.defaultRole = identity.RoleViewer
	request := system.dossier.request(t, "request:intake-dossier-denied")

	_, err := system.orchestrator.PrepareIntakeDossier(
		context.Background(), system.access, request,
	)
	if !errors.Is(err, ErrForbidden) ||
		system.dossier.intakes.gets.Load() != 0 ||
		system.dossier.dossiers.creates.Load() != 0 {
		t.Fatalf(
			"denied prepare err=%v intake_gets=%d dossier_creates=%d",
			err,
			system.dossier.intakes.gets.Load(),
			system.dossier.dossiers.creates.Load(),
		)
	}

	system.accessRepository.defaultRole = identity.RolePlatformAdmin
	prepared, err := system.orchestrator.PrepareIntakeDossier(
		context.Background(), system.access, request,
	)
	if err != nil {
		t.Fatal(err)
	}
	beforeGets := system.dossiers.gets.Load()
	system.accessRepository.defaultRole = ""
	_, err = system.orchestrator.GetIntakeDossier(
		context.Background(),
		system.access,
		GetIntakeDossierRequest{DossierRef: prepared.Record.Dossier.Ref()},
	)
	if !IsStateError(err, StateNotFound) ||
		system.dossiers.gets.Load() != beforeGets {
		t.Fatalf(
			"denied get err=%v dossier_gets=%d/%d",
			err, system.dossiers.gets.Load(), beforeGets,
		)
	}
}

func TestOrchestratorWizardDossierBindsAuthenticatedScopeAndAuthorization(
	t *testing.T,
) {
	system := newIntakeDossierOrchestratorTestSystem(t)
	request := wizardDossierTestRequest(
		t,
		system.dossier,
		"request:wizard-dossier-orchestrator",
		"template:build_app",
	)
	spoofedActor, _ := goal.NewActorRef("actor:spoofed")
	spoofedProject, _ := goal.NewProjectRef("project:spoofed")
	request.ActorRef = spoofedActor
	request.ProjectRef = spoofedProject
	request.AuthorizationReceipt = identity.AuthorizationReceipt{}

	result, err := system.orchestrator.PrepareWizardDossier(
		context.Background(),
		system.access,
		request,
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Record.ActorRef != system.dossier.record.ActorRef ||
		result.Record.ProjectRef != system.dossier.record.ProjectRef ||
		result.StagePlanProjection.TemplateRef != request.TemplateRef {
		t.Fatalf("wizard dossier=%+v", result)
	}
	authorizations := intakeAuthorizations(system.accessRepository)
	if len(authorizations) != 1 {
		t.Fatalf("authorizations=%d", len(authorizations))
	}
	authorization := authorizations[0]
	if authorization.Principal().ActorRef != system.dossier.record.ActorRef ||
		authorization.ProjectRef() != system.dossier.record.ProjectRef ||
		authorization.Permission() != identity.PermissionGoalsCreate ||
		authorization.RequestRef() !=
			"authorization-request:intake-dossier-generate:"+
				request.RequestRef {
		t.Fatalf("authorization=%+v", authorization)
	}
}

func TestOrchestratorWithoutIntakeDossierStoreReturnsUnavailable(t *testing.T) {
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
	_, err := orchestrator.PrepareIntakeDossier(
		context.Background(), access, PrepareIntakeDossierRequest{},
	)
	assertUnavailable(err)
	_, err = orchestrator.PrepareWizardDossier(
		context.Background(), access, PrepareWizardDossierRequest{},
	)
	assertUnavailable(err)
	_, err = orchestrator.GetIntakeDossier(
		context.Background(), access, GetIntakeDossierRequest{},
	)
	assertUnavailable(err)
}

type intakeDossierOrchestratorTestSystem struct {
	orchestrator     *Orchestrator
	dossier          intakeDossierTestSystem
	dossiers         *countingIntakeDossierStore
	accessRepository *memoryAccessRepository
	access           Access
}

type countingIntakeDossierStore struct {
	*memoryIntakeDossierStore
	gets atomic.Uint64
}

func (store *countingIntakeDossierStore) GetIntakeDossier(
	ctx context.Context,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	dossierRef IntakeDossierRef,
) (IntakeDossierRecord, error) {
	store.gets.Add(1)
	return store.memoryIntakeDossierStore.GetIntakeDossier(
		ctx, actorRef, projectRef, dossierRef,
	)
}

func newIntakeDossierOrchestratorTestSystem(
	t *testing.T,
) intakeDossierOrchestratorTestSystem {
	t.Helper()
	clock := &mutableClock{now: time.Date(2026, 7, 26, 10, 30, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	repository.now = clock.Now
	accessRepository := newMemoryAccessRepository()
	dossier := newIntakeDossierTestSystem(t)
	dossiers := &countingIntakeDossierStore{
		memoryIntakeDossierStore: dossier.dossiers,
	}
	agent := &scriptedAgent{now: clock.Now}
	capabilities, err := agent.Capabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	orchestrator, err := New(Dependencies{
		State: repository, WizardGapsStore: dossier.intakes,
		IntakeDossierStore: dossiers, Access: accessRepository,
		Launcher: agent, Observer: agent, Controller: agent,
		Artifacts:        newMemoryArtifactStore(),
		WorkspaceManager: &scriptedWorkspaceManager{},
		VersionControl:   &scriptedVersionControl{},
		TestAttestor:     &scriptedTestAttestor{},
		TestAttestationPolicy: TestAttestationPolicy{
			Ref:    "test-attestation-policy:intake-dossier",
			Digest: testDigest("test-attestation-policy:intake-dossier"),
		},
		Clock: clock, IDs: &sequentialIDs{}, MaxOutputBytes: 1 << 20,
		MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts:    3, MaxChildrenPerParent: 6,
		ClaimLease: time.Minute, DirectorLeaseDuration: time.Minute,
		EffectApprovalTTL: time.Hour, BudgetPolicy: testBudgetPolicy(clock.Now()),
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: capabilities,
	})
	if err != nil {
		t.Fatal(err)
	}
	access, err := NewAccess(
		intakeDossierPrincipal(t, dossier.record.ActorRef),
		dossier.record.ProjectRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	return intakeDossierOrchestratorTestSystem{
		orchestrator: orchestrator, dossier: dossier, dossiers: dossiers,
		accessRepository: accessRepository, access: access,
	}
}

func intakeDossierPrincipal(
	t *testing.T,
	actorRef goal.ActorRef,
) identity.Principal {
	t.Helper()
	principalRef, err := identity.NewPrincipalRef("principal:intake-dossier-orchestrator")
	if err != nil {
		t.Fatal(err)
	}
	principal, err := identity.NewPrincipal(
		principalRef, actorRef, identity.PrincipalKindHuman, "test",
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

var _ IntakeDossierStore = (*countingIntakeDossierStore)(nil)
