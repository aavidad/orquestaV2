package application

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type synchronizedDivergentEgressResolver struct {
	mu          sync.Mutex
	authorities []EgressPolicyAuthority
	refs        []EgressPolicyRef
	err         error
}

func (resolver *synchronizedDivergentEgressResolver) ResolveEgressPolicy(
	_ context.Context,
	ref EgressPolicyRef,
) (EgressPolicyAuthority, error) {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	resolver.refs = append(resolver.refs, ref)
	if resolver.err != nil {
		return EgressPolicyAuthority{}, resolver.err
	}
	return resolver.authorities[(len(resolver.refs)-1)%len(resolver.authorities)], nil
}

func (resolver *synchronizedDivergentEgressResolver) callCount() int {
	resolver.mu.Lock()
	defer resolver.mu.Unlock()
	return len(resolver.refs)
}

type intakeDossierReplayBarrierRepository struct {
	StateRepository
	arrived chan<- struct{}
	release <-chan struct{}
}

func (repository *intakeDossierReplayBarrierRepository) ReplayIntakeDossierConfirmation(
	ctx context.Context,
	_ GoalSubmissionReplayRequest,
) (IntakeDossierConfirmationRecord, bool, error) {
	select {
	case repository.arrived <- struct{}{}:
	case <-ctx.Done():
		return IntakeDossierConfirmationRecord{}, false, ctx.Err()
	}
	select {
	case <-repository.release:
		return IntakeDossierConfirmationRecord{}, false, nil
	case <-ctx.Done():
		return IntakeDossierConfirmationRecord{}, false, ctx.Err()
	}
}

type intakeDossierStateWithoutReplay struct{ StateRepository }

func (repository *memoryRepository) ReplayIntakeDossierConfirmation(
	_ context.Context,
	request GoalSubmissionReplayRequest,
) (IntakeDossierConfirmationRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	key := request.RequestedBy.String() + "\x00" + request.ProjectRef.String() + "\x00" + request.RequestRef
	record, found := repository.dossierConfirmations[key]
	if !found {
		return IntakeDossierConfirmationRecord{}, false, nil
	}
	if record.Confirmation.RequestFingerprint != request.RequestFingerprint ||
		record.Confirmation.PrincipalRef != request.RequestedBy ||
		record.Confirmation.ProjectRef != request.ProjectRef {
		return IntakeDossierConfirmationRecord{}, false, &StateError{Code: StateConflict}
	}
	live, found := repository.records[record.Confirmation.GoalRef]
	if !found {
		return IntakeDossierConfirmationRecord{}, false, &StateError{Code: StateConflict}
	}
	record.Goal = live
	return cloneIntakeDossierConfirmationRecord(record), true, nil
}

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

func TestConfirmIntakeDossierEgressReplayUsesHistoricalAuthorityWithoutResolver(t *testing.T) {
	ctx := context.Background()
	system := newIntakeDossierOrchestratorTestSystem(t)
	historical := testEgressPolicyAuthority(t, "egress-policy:dossier-historical", `{"revision":1}`)
	prepared := prepareEgressIntakeDossier(
		t, system, "request:intake-dossier-egress-replay-prepare",
		[]EgressPolicyAuthority{historical},
	)
	request := ConfirmIntakeDossierRequest{
		RequestRef: "request:intake-dossier-egress-replay",
		DossierRef: prepared.Dossier.Ref(), Confirm: true,
	}
	system.orchestrator.egressPolicies = &egressPolicyResolverStub{authority: historical}
	created, err := system.orchestrator.ConfirmIntakeDossier(ctx, system.access, request)
	if err != nil || !created.Created {
		t.Fatalf("created=%+v err=%v", created, err)
	}

	rotated := &egressPolicyResolverStub{authority: testEgressPolicyAuthority(
		t, historical.PolicyRef.String(), `{"revision":2}`,
	)}
	system.orchestrator.egressPolicies = rotated
	rotatedReplay, err := system.orchestrator.ConfirmIntakeDossier(ctx, system.access, request)
	if err != nil || rotatedReplay.Created || len(rotated.refs) != 0 ||
		rotatedReplay.Record.WorkItemAuthorities[0].EgressPolicy != historical {
		t.Fatalf("rotated replay=%+v calls=%v err=%v", rotatedReplay, rotated.refs, err)
	}

	failing := &egressPolicyResolverStub{err: errors.New("catalog unavailable")}
	system.orchestrator.egressPolicies = failing
	failedReplay, err := system.orchestrator.ConfirmIntakeDossier(ctx, system.access, request)
	if err != nil || failedReplay.Created || len(failing.refs) != 0 ||
		failedReplay.Record.WorkItemAuthorities[0].EgressPolicy != historical {
		t.Fatalf("failed replay=%+v calls=%v err=%v", failedReplay, failing.refs, err)
	}

	system.orchestrator.egressPolicies = nil
	retiredReplay, err := system.orchestrator.ConfirmIntakeDossier(ctx, system.access, request)
	if err != nil || retiredReplay.Created ||
		retiredReplay.Record.WorkItemAuthorities[0].EgressPolicy != historical {
		t.Fatalf("retired replay=%+v err=%v", retiredReplay, err)
	}
}

func TestConfirmIntakeDossierEgressReplayRejectsCrossWorkItemPolicySwap(t *testing.T) {
	ctx := context.Background()
	system := newIntakeDossierOrchestratorTestSystem(t)
	first := testEgressPolicyAuthority(t, "egress-policy:dossier-first", `{"item":1}`)
	second := testEgressPolicyAuthority(t, "egress-policy:dossier-second", `{"item":2}`)
	prepared := prepareEgressIntakeDossier(
		t, system, "request:intake-dossier-egress-swap-prepare",
		[]EgressPolicyAuthority{first, second},
	)
	system.orchestrator.egressPolicies = &mappedEgressPolicyResolver{authorities: map[EgressPolicyRef]EgressPolicyAuthority{
		first.PolicyRef: first, second.PolicyRef: second,
	}}
	request := ConfirmIntakeDossierRequest{
		RequestRef: "request:intake-dossier-egress-swap",
		DossierRef: prepared.Dossier.Ref(), Confirm: true,
	}
	created, err := system.orchestrator.ConfirmIntakeDossier(ctx, system.access, request)
	if err != nil {
		t.Fatal(err)
	}
	repository := system.orchestrator.state.(*memoryRepository)
	repository.mu.Lock()
	record := repository.records[created.Record.Goal.Ref()]
	record.WorkItemAuthorities[0].EgressPolicy, record.WorkItemAuthorities[1].EgressPolicy =
		record.WorkItemAuthorities[1].EgressPolicy, record.WorkItemAuthorities[0].EgressPolicy
	repository.records[created.Record.Goal.Ref()] = record
	repository.mu.Unlock()
	system.orchestrator.egressPolicies = nil

	if _, err := system.orchestrator.ConfirmIntakeDossier(ctx, system.access, request); !IsStateError(err, StateConflict) {
		t.Fatalf("cross-work-item dossier authority swap replayed: %v", err)
	}
}

func TestConfirmIntakeDossierEgressConcurrentReplayMissReturnsOneHistoricalWinner(t *testing.T) {
	ctx := context.Background()
	system := newIntakeDossierOrchestratorTestSystem(t)
	first := testEgressPolicyAuthority(t, "egress-policy:dossier-race", `{"revision":1}`)
	second := testEgressPolicyAuthority(t, first.PolicyRef.String(), `{"revision":2}`)
	prepared := prepareEgressIntakeDossier(
		t, system, "request:intake-dossier-egress-race-prepare",
		[]EgressPolicyAuthority{first},
	)
	resolver := &synchronizedDivergentEgressResolver{authorities: []EgressPolicyAuthority{first, second}}
	system.orchestrator.egressPolicies = resolver
	underlying := system.orchestrator.state
	arrived := make(chan struct{}, 2)
	release := make(chan struct{})
	system.orchestrator.state = &intakeDossierReplayBarrierRepository{
		StateRepository: underlying, arrived: arrived, release: release,
	}
	request := ConfirmIntakeDossierRequest{
		RequestRef: "request:intake-dossier-egress-race",
		DossierRef: prepared.Dossier.Ref(), Confirm: true,
	}
	results := make([]ConfirmIntakeDossierResult, 2)
	errs := make([]error, 2)
	var wait sync.WaitGroup
	for index := range results {
		wait.Add(1)
		go func() {
			defer wait.Done()
			results[index], errs[index] = system.orchestrator.ConfirmIntakeDossier(ctx, system.access, request)
		}()
	}
	<-arrived
	<-arrived
	close(release)
	wait.Wait()
	created := 0
	for index, err := range errs {
		if err != nil {
			t.Fatalf("worker %d: %v", index, err)
		}
		if results[index].Created {
			created++
		}
	}
	if created != 1 || resolver.callCount() != 2 ||
		results[0].Confirmation != results[1].Confirmation ||
		results[0].Record.WorkItemAuthorities[0].EgressPolicy !=
			results[1].Record.WorkItemAuthorities[0].EgressPolicy {
		t.Fatalf("created=%d resolver_calls=%d results=%+v", created, resolver.callCount(), results)
	}
}

func TestConfirmIntakeDossierRequiresReplayCapabilityOnlyForEgress(t *testing.T) {
	ctx := context.Background()
	egressSystem := newIntakeDossierOrchestratorTestSystem(t)
	policy := testEgressPolicyAuthority(t, "egress-policy:dossier-required-reader", `{"revision":1}`)
	prepared := prepareEgressIntakeDossier(
		t, egressSystem, "request:intake-dossier-egress-reader-prepare",
		[]EgressPolicyAuthority{policy},
	)
	resolver := &egressPolicyResolverStub{authority: policy}
	egressSystem.orchestrator.egressPolicies = resolver
	egressSystem.orchestrator.state = intakeDossierStateWithoutReplay{
		StateRepository: egressSystem.orchestrator.state,
	}
	_, err := egressSystem.orchestrator.ConfirmIntakeDossier(ctx, egressSystem.access, ConfirmIntakeDossierRequest{
		RequestRef: "request:intake-dossier-egress-reader",
		DossierRef: prepared.Dossier.Ref(), Confirm: true,
	})
	if err == nil || err.Error() != "application.egress_policy_replay_reader_required" || len(resolver.refs) != 0 {
		t.Fatalf("egress state without replay err=%v resolver_calls=%v", err, resolver.refs)
	}

	plainSystem := newIntakeDossierOrchestratorTestSystem(t)
	plainPrepared, err := plainSystem.orchestrator.PrepareIntakeDossier(
		ctx, plainSystem.access,
		plainSystem.dossier.request(t, "request:intake-dossier-plain-reader-prepare"),
	)
	if err != nil {
		t.Fatal(err)
	}
	plainSystem.orchestrator.state = intakeDossierStateWithoutReplay{
		StateRepository: plainSystem.orchestrator.state,
	}
	plainResult, err := plainSystem.orchestrator.ConfirmIntakeDossier(ctx, plainSystem.access, ConfirmIntakeDossierRequest{
		RequestRef: "request:intake-dossier-plain-reader",
		DossierRef: plainPrepared.Record.Dossier.Ref(), Confirm: true,
	})
	if err != nil || !plainResult.Created {
		t.Fatalf("plain state without replay result=%+v err=%v", plainResult, err)
	}
}

func prepareEgressIntakeDossier(
	t *testing.T,
	system intakeDossierOrchestratorTestSystem,
	requestRef string,
	authorities []EgressPolicyAuthority,
) IntakeDossierRecord {
	t.Helper()
	request := system.dossier.request(t, requestRef)
	request.Plan.WorkItems[0].EgressPolicyRef = authorities[0].PolicyRef.String()
	for index := 1; index < len(authorities); index++ {
		item := request.Plan.WorkItems[0]
		item.Key += ":" + authorities[index].PolicyRef.String()
		item.Objective += " " + authorities[index].PolicyRef.String()
		item.RequiredTests = nil
		item.EgressPolicyRef = authorities[index].PolicyRef.String()
		request.Plan.WorkItems = append(request.Plan.WorkItems, item)
	}
	prepared, err := system.orchestrator.PrepareIntakeDossier(
		context.Background(), system.access, request,
	)
	if err != nil {
		t.Fatal(err)
	}
	return prepared.Record
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
