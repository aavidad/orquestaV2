package application

import (
	"context"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestConfirmIntakeDossierCreatesRunningGoalFromExactDossier(t *testing.T) {
	ctx := context.Background()
	system := newIntakeDossierOrchestratorTestSystem(t)
	prepared, err := system.orchestrator.PrepareIntakeDossier(
		ctx,
		system.access,
		system.dossier.request(t, "request:intake-dossier-confirm-prepare"),
	)
	if err != nil {
		t.Fatal(err)
	}
	repository := system.orchestrator.state.(*memoryRepository)
	beforeActions := len(repository.actions)
	beforeEvents := len(repository.events)

	const requestRef = "request:intake-dossier-confirm"
	result, err := system.orchestrator.ConfirmIntakeDossier(
		ctx,
		system.access,
		ConfirmIntakeDossierRequest{
			RequestRef: requestRef,
			DossierRef: prepared.Record.Dossier.Ref(),
			Confirm:    true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	aggregate := result.Record.Goal
	confirmation := result.Confirmation
	if !result.Created ||
		aggregate.State() != goal.GoalStateRunning ||
		aggregate.PlanGeneration() != 1 ||
		aggregate.WorkItemCount() !=
			len(prepared.Record.Dossier.Plan().WorkItems) ||
		aggregate.AppSpec().Reason() !=
			IntakeDossierConfirmationAppSpecReason(
				prepared.Record.Dossier.Ref(),
			) {
		t.Fatalf(
			"created=%t state=%q plan_generation=%d work_items=%d reason=%q",
			result.Created,
			aggregate.State(),
			aggregate.PlanGeneration(),
			aggregate.WorkItemCount(),
			aggregate.AppSpec().Reason(),
		)
	}
	if len(result.Record.Executions) != 1 ||
		len(repository.actions) != beforeActions+1 ||
		len(repository.events) <= beforeEvents {
		t.Fatalf(
			"executions=%d actions=%d/%d events=%d/%d",
			len(result.Record.Executions),
			len(repository.actions),
			beforeActions,
			len(repository.events),
			beforeEvents,
		)
	}
	if confirmation.RequestRef != requestRef ||
		confirmation.DossierRef != prepared.Record.Dossier.Ref() ||
		confirmation.DossierDigest != prepared.Record.Dossier.Digest() ||
		confirmation.PlanDigest != prepared.Record.Dossier.PlanDigest() ||
		confirmation.StateRef != prepared.Record.Dossier.StateRef() ||
		confirmation.StateRevision !=
			prepared.Record.Dossier.StateRevision() ||
		confirmation.StateDigest != prepared.Record.Dossier.StateDigest() ||
		confirmation.SourceIntakeReceiptRef !=
			prepared.Record.Dossier.SourceIntakeReceiptRef() ||
		confirmation.GoalRef != aggregate.Ref() ||
		confirmation.AppSpecRef != aggregate.AppSpec().Ref() ||
		confirmation.SpecHash != aggregate.SpecHash() ||
		confirmation.ConfirmedAt != aggregate.AppSpec().ConfirmedAt() {
		t.Fatalf("confirmation does not bind exact dossier and Goal: %+v", confirmation)
	}
	expected := buildIntakeDossierConfirmation(
		requestRef,
		confirmation.RequestFingerprint,
		system.access.principal,
		prepared.Record.Dossier,
		aggregate,
		confirmation.AuthorizationReceiptRef,
	)
	if confirmation != expected {
		t.Fatalf("confirmation=%+v want=%+v", confirmation, expected)
	}
	confirmationKey := system.access.principal.Ref.String() + "\x00" +
		system.access.projectRef.String() + "\x00" + requestRef
	if stored, found := repository.dossierConfirmations[confirmationKey]; !found ||
		stored.Confirmation != confirmation ||
		!reflect.DeepEqual(
			stored.Goal.Goal.Snapshot(),
			aggregate.Snapshot(),
		) {
		t.Fatalf("durable confirmation found=%t record=%+v", found, stored)
	}

	authorizations := intakeAuthorizations(system.accessRepository)
	if len(authorizations) != 2 {
		t.Fatalf("authorization count=%d want=2", len(authorizations))
	}
	authorization := authorizations[1]
	if authorization.RequestRef() !=
		intakeDossierConfirmationAuthorizationPrefix+requestRef ||
		authorization.Principal() != system.access.principal ||
		authorization.ProjectRef() != system.access.projectRef ||
		authorization.Permission() != identity.PermissionGoalsCreate ||
		authorization.ResourceRef() != system.access.projectRef.String() {
		t.Fatalf("confirmation authorization=%+v", authorization)
	}
}

func TestConfirmIntakeDossierExactReplayAndSingleGoalPerDossier(t *testing.T) {
	ctx := context.Background()
	system := newIntakeDossierOrchestratorTestSystem(t)
	prepared, err := system.orchestrator.PrepareIntakeDossier(
		ctx,
		system.access,
		system.dossier.request(t, "request:intake-dossier-replay-prepare"),
	)
	if err != nil {
		t.Fatal(err)
	}
	request := ConfirmIntakeDossierRequest{
		RequestRef: "request:intake-dossier-confirm-replay",
		DossierRef: prepared.Record.Dossier.Ref(),
		Confirm:    true,
	}
	first, err := system.orchestrator.ConfirmIntakeDossier(
		ctx, system.access, request,
	)
	if err != nil {
		t.Fatal(err)
	}
	repository := system.orchestrator.state.(*memoryRepository)
	beforeRecords := len(repository.records)
	beforeConfirmations := len(repository.dossierConfirmations)
	beforeActions := len(repository.actions)
	beforeEvents := len(repository.events)

	replayed, err := system.orchestrator.ConfirmIntakeDossier(
		ctx, system.access, request,
	)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Created ||
		replayed.Confirmation != first.Confirmation ||
		!reflect.DeepEqual(
			replayed.Record.Goal.Snapshot(),
			first.Record.Goal.Snapshot(),
		) ||
		!reflect.DeepEqual(replayed.Record.Executions, first.Record.Executions) {
		t.Fatalf("non-exact replay: first=%+v replay=%+v", first, replayed)
	}
	assertConfirmationRepositoryCounts(
		t,
		repository,
		beforeRecords,
		beforeConfirmations,
		beforeActions,
		beforeEvents,
	)

	_, err = system.orchestrator.ConfirmIntakeDossier(
		ctx,
		system.access,
		ConfirmIntakeDossierRequest{
			RequestRef: "request:intake-dossier-confirm-second",
			DossierRef: prepared.Record.Dossier.Ref(),
			Confirm:    true,
		},
	)
	if !IsStateError(err, StateConflict) {
		t.Fatalf("second request for confirmed dossier error=%v", err)
	}
	assertConfirmationRepositoryCounts(
		t,
		repository,
		beforeRecords,
		beforeConfirmations,
		beforeActions,
		beforeEvents,
	)
}

func TestConfirmIntakeDossierExactReplayAcceptsLiveGoalProgress(t *testing.T) {
	ctx := context.Background()
	system := newIntakeDossierOrchestratorTestSystem(t)
	prepared, err := system.orchestrator.PrepareIntakeDossier(
		ctx,
		system.access,
		system.dossier.request(
			t, "request:intake-dossier-live-replay-prepare",
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	request := ConfirmIntakeDossierRequest{
		RequestRef: "request:intake-dossier-live-replay",
		DossierRef: prepared.Record.Dossier.Ref(),
		Confirm:    true,
	}
	first, err := system.orchestrator.ConfirmIntakeDossier(
		ctx, system.access, request,
	)
	if err != nil {
		t.Fatal(err)
	}
	repository := system.orchestrator.state.(*memoryRepository)
	record := repository.records[first.Record.Goal.Ref()]
	execution := record.Executions[0]
	item, found := record.Goal.WorkItem(execution.WorkItemRef)
	if !found {
		t.Fatal("initial execution has no WorkItem")
	}
	startedAt := record.Goal.CreatedAt().Add(time.Minute)
	progressed, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(),
		execution.Ref, startedAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	record.Goal = progressed
	record.Executions[0].State = ExecutionRunning
	record.Executions[0].StartedAt = startedAt
	record.Executions[0].ProviderAcceptedAt = startedAt
	record.Executions[0].ProviderRef = "provider:live-replay"
	record.Executions[0].ModelRef = "model:live-replay"
	record.Executions[0].AgentRef = "agent:live-replay"
	record.Executions[0].ExternalRef = "external:live-replay"
	repository.records[record.Goal.Ref()] = record

	replayed, err := system.orchestrator.ConfirmIntakeDossier(
		ctx, system.access, request,
	)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Created ||
		replayed.Record.Goal.Revision() != progressed.Revision() ||
		replayed.Record.Goal.Snapshot().WorkItems[0].State !=
			goal.WorkItemStateRunning ||
		replayed.Confirmation != first.Confirmation {
		t.Fatalf("live replay=%+v", replayed)
	}
}

func assertConfirmationRepositoryCounts(
	t *testing.T,
	repository *memoryRepository,
	records int,
	confirmations int,
	actions int,
	events int,
) {
	t.Helper()
	if len(repository.records) != records ||
		len(repository.dossierConfirmations) != confirmations ||
		len(repository.actions) != actions ||
		len(repository.events) != events {
		t.Fatalf(
			"records=%d/%d confirmations=%d/%d actions=%d/%d events=%d/%d",
			len(repository.records),
			records,
			len(repository.dossierConfirmations),
			confirmations,
			len(repository.actions),
			actions,
			len(repository.events),
			events,
		)
	}
}
