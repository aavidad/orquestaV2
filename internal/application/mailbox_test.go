package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type mailboxTestSystem struct {
	clock           *mutableClock
	repository      *memoryRepository
	accessStore     *memoryAccessRepository
	agent           *scriptedAgent
	artifacts       *memoryArtifactStore
	orchestrator    *Orchestrator
	project         goal.ProjectRef
	source          identity.Principal
	recipient       identity.Principal
	intruder        identity.Principal
	sourceAccess    Access
	sourceAccesses  []Access
	recipientAccess Access
	intruderAccess  Access
	goalRef         goal.GoalRef
	parentRef       goal.WorkItemRef
	parentExecution goal.ExecutionRef
	children        []goal.WorkItemRef
	childExecutions []goal.ExecutionRef
	childArtifacts  []goal.ArtifactRef
}

func TestMailboxAdmissionIsAtomicAndRequestIdempotent(t *testing.T) {
	ctx := context.Background()
	system := newMailboxTestSystem(t, 1)
	request := system.admissionRequest(0, "mailbox-admit:atomic")

	first, err := system.orchestrator.AdmitMailbox(ctx, system.sourceAccess, request)
	if err != nil || !first.Created {
		t.Fatalf("first admission=%+v err=%v", first, err)
	}
	replayed, err := system.orchestrator.AdmitMailbox(ctx, system.sourceAccess, request)
	if err != nil || replayed.Created || !reflect.DeepEqual(replayed.Record, first.Record) {
		t.Fatalf("admission replay=%+v err=%v", replayed, err)
	}
	claim, err := system.orchestrator.ClaimMailbox(
		ctx, system.recipientAccess, system.claimRequest(first.Record, "mailbox-claim:admission-progress"),
	)
	if err != nil || !claim.Claimed {
		t.Fatalf("claim after admission=%+v err=%v", claim, err)
	}
	replayed, err = system.orchestrator.AdmitMailbox(ctx, system.sourceAccess, request)
	if err != nil || replayed.Created || !reflect.DeepEqual(replayed.Record.Envelope, first.Record.Envelope) ||
		replayed.Record.Admission != first.Record.Admission {
		t.Fatalf("admission replay after progress=%+v err=%v", replayed, err)
	}
	if first.Record.State != MailboxStateAdmitted || len(first.Record.Attempts) != 0 ||
		first.Record.Acknowledgement != nil || first.Record.Admission.Ref == "" ||
		first.Record.Action.Kind != ActionDeliverMailbox || first.Record.Action.Ref == first.Record.Admission.Ref {
		t.Fatalf("admission collapsed distinct facts: %+v", first.Record)
	}
	system.repository.mu.Lock()
	if system.repository.mailboxAdmits != 1 || len(system.repository.mailboxes) != 1 ||
		len(system.repository.actions) != 1 || len(system.repository.events) != 1 {
		t.Fatalf("non-atomic admission: admits=%d mailboxes=%d actions=%d events=%d",
			system.repository.mailboxAdmits, len(system.repository.mailboxes),
			len(system.repository.actions), len(system.repository.events))
	}
	system.repository.mu.Unlock()

	conflict := request
	conflict.Summary = "different semantic payload"
	if _, err = system.orchestrator.AdmitMailbox(ctx, system.sourceAccess, conflict); !IsStateError(err, StateConflict) {
		t.Fatalf("same request_ref changed payload error=%v", err)
	}
	system.assertMailboxCounts(t, [5]int{1, 1, 0, 0, 0})
}

func TestMailboxRejectsDuplicateChildDeliveryRelation(t *testing.T) {
	ctx := context.Background()
	system := newMailboxTestSystem(t, 1)
	first := system.admissionRequest(0, "mailbox-admit:relation-first")
	if admitted, err := system.orchestrator.AdmitMailbox(
		ctx, system.sourceAccess, first,
	); err != nil || !admitted.Created {
		t.Fatalf("first relation admission=%+v err=%v", admitted, err)
	}
	duplicate := first
	duplicate.RequestRef = "mailbox-admit:relation-duplicate"
	if _, err := system.orchestrator.AdmitMailbox(
		ctx, system.sourceAccess, duplicate,
	); !IsStateError(err, StateConflict) {
		t.Fatalf("same child relation admitted twice: %v", err)
	}
	system.repository.mu.Lock()
	mailboxes, actions, events := len(system.repository.mailboxes), len(system.repository.actions), len(system.repository.events)
	system.repository.mu.Unlock()
	if system.mailboxCounts() != ([5]int{1, 0, 0, 0, 0}) ||
		mailboxes != 1 || actions != 1 || events != 1 {
		t.Fatalf("duplicate relation caused writes: counts=%v mailboxes=%d actions=%d events=%d",
			system.mailboxCounts(), mailboxes, actions, events)
	}
}

func TestMailboxEnvelopeLimitRejectsBeforeIdentityOrStateEffects(t *testing.T) {
	ctx := context.Background()
	atLimit := newMailboxTestSystem(t, 1)
	request := atLimit.admissionRequest(0, "mailbox-admit:envelope-limit")
	request.ArtifactRefs = append(request.ArtifactRefs, request.ArtifactRefs...)
	canonical := request
	canonical.ArtifactRefs = canonicalMailboxArtifactRefs(canonical.ArtifactRefs)
	atLimit.orchestrator.maxMailboxEnvelopeBytes = mailboxEnvelopeInputBytes(
		atLimit.source.Ref, atLimit.project, canonical,
	)
	result, err := atLimit.orchestrator.AdmitMailbox(ctx, atLimit.sourceAccess, request)
	if err != nil || !result.Created {
		t.Fatalf("exact envelope limit rejected: result=%+v err=%v", result, err)
	}

	overLimit := newMailboxTestSystem(t, 1)
	request = overLimit.admissionRequest(0, "mailbox-admit:envelope-limit")
	overLimit.orchestrator.maxMailboxEnvelopeBytes = mailboxEnvelopeInputBytes(
		overLimit.source.Ref, overLimit.project, request,
	)
	request.Summary += "x"
	ids := overLimit.orchestrator.ids.(*sequentialIDs)
	ids.mu.Lock()
	idsBefore := ids.next
	ids.mu.Unlock()
	overLimit.accessStore.mu.Lock()
	authorizationsBefore := len(overLimit.accessStore.authorizations)
	overLimit.accessStore.mu.Unlock()
	if _, err = overLimit.orchestrator.AdmitMailbox(
		ctx, overLimit.sourceAccess, request,
	); err == nil || err.Error() != "application.mailbox_envelope_too_large" {
		t.Fatalf("oversized envelope error=%v", err)
	}
	ids.mu.Lock()
	idsAfter := ids.next
	ids.mu.Unlock()
	overLimit.accessStore.mu.Lock()
	authorizationsAfter := len(overLimit.accessStore.authorizations)
	overLimit.accessStore.mu.Unlock()
	if idsAfter != idsBefore || authorizationsAfter != authorizationsBefore ||
		overLimit.mailboxCounts() != ([5]int{}) || len(overLimit.repository.actions) != 0 ||
		len(overLimit.repository.events) != 0 {
		t.Fatalf("oversized envelope caused effects: ids=%d/%d auth=%d/%d counts=%v actions=%d events=%d",
			idsBefore, idsAfter, authorizationsBefore, authorizationsAfter,
			overLimit.mailboxCounts(), len(overLimit.repository.actions), len(overLimit.repository.events))
	}
}

func TestMailboxRejectsNonContractualChildEdgeBeforeEffects(t *testing.T) {
	ctx := context.Background()
	system := newMailboxTestSystem(t, 1)
	record, err := system.repository.GetGoal(ctx, system.goalRef)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := record.Goal.Snapshot()
	nonContractual := false
	for index := range snapshot.WorkItems {
		if snapshot.WorkItems[index].Ref == system.children[0].String() {
			snapshot.WorkItems[index].HandoffRequired = &nonContractual
		}
	}
	legacy, err := goal.RestoreGoal(snapshot)
	if err != nil {
		t.Fatalf("restore legacy Goal: %v", err)
	}
	child, found := legacy.WorkItem(system.children[0])
	if !found || child.HandoffRequired() {
		t.Fatalf("legacy edge fixture is contractual: child=%+v found=%v", child, found)
	}
	record.Goal = legacy
	system.repository.mu.Lock()
	system.repository.records[system.goalRef] = record
	beforeState := [4]int{
		len(system.repository.mailboxes), len(system.repository.mailboxRequests),
		len(system.repository.actions), len(system.repository.events),
	}
	system.repository.mu.Unlock()
	ids := system.orchestrator.ids.(*sequentialIDs)
	ids.mu.Lock()
	idsBefore := ids.next
	ids.mu.Unlock()
	system.accessStore.mu.Lock()
	authorizationsBefore := len(system.accessStore.authorizations)
	system.accessStore.mu.Unlock()

	if _, err = system.orchestrator.AdmitMailbox(
		ctx, system.sourceAccess, system.admissionRequest(0, "mailbox-admit:legacy-non-handoff"),
	); !IsStateError(err, StateConflict) {
		t.Fatalf("legacy non-handoff edge admitted: %v", err)
	}
	after, stateErr := system.repository.GetGoal(ctx, system.goalRef)
	system.repository.mu.Lock()
	afterState := [4]int{
		len(system.repository.mailboxes), len(system.repository.mailboxRequests),
		len(system.repository.actions), len(system.repository.events),
	}
	system.repository.mu.Unlock()
	ids.mu.Lock()
	idsAfter := ids.next
	ids.mu.Unlock()
	system.accessStore.mu.Lock()
	authorizationsAfter := len(system.accessStore.authorizations)
	system.accessStore.mu.Unlock()
	if stateErr != nil || beforeState != afterState || idsBefore != idsAfter ||
		authorizationsBefore != authorizationsAfter || system.mailboxCounts() != ([5]int{}) ||
		!reflect.DeepEqual(after.Goal.Snapshot(), legacy.Snapshot()) {
		t.Fatalf("legacy rejection caused effects: state=%v/%v ids=%d/%d auth=%d/%d mailbox=%v err=%v",
			beforeState, afterState, idsBefore, idsAfter, authorizationsBefore, authorizationsAfter,
			system.mailboxCounts(), stateErr)
	}
}

func TestMailboxConcurrentClaimHasOneExactRecipientWinner(t *testing.T) {
	ctx := context.Background()
	system := newMailboxTestSystem(t, 1)
	admitted := system.admit(t, 0, "mailbox-admit:race")
	requests := []ClaimMailboxRequest{
		system.claimRequest(admitted.Record, "mailbox-claim:race-a"),
		system.claimRequest(admitted.Record, "mailbox-claim:race-b"),
	}
	type result struct {
		value MailboxClaimResult
		err   error
	}
	start := make(chan struct{})
	results := make(chan result, len(requests))
	var workers sync.WaitGroup
	for _, request := range requests {
		workers.Add(1)
		go func(request ClaimMailboxRequest) {
			defer workers.Done()
			<-start
			value, err := system.orchestrator.ClaimMailbox(ctx, system.recipientAccess, request)
			results <- result{value: value, err: err}
		}(request)
	}
	close(start)
	workers.Wait()
	close(results)

	winners, alreadyClaimed := 0, 0
	for result := range results {
		switch {
		case result.err == nil && result.value.Claimed:
			winners++
			if result.value.Claim.Attempt.Recipient != admitted.Record.Envelope.Recipient ||
				result.value.Claim.Record.State != MailboxStateClaimed {
				t.Fatalf("claim winner changed exact address: %+v", result.value)
			}
		case IsStateError(result.err, StateAlreadyClaimed):
			alreadyClaimed++
		default:
			t.Fatalf("unexpected concurrent claim result=%+v err=%v", result.value, result.err)
		}
	}
	if winners != 1 || alreadyClaimed != 1 {
		t.Fatalf("claim race winners=%d already_claimed=%d", winners, alreadyClaimed)
	}
	system.assertMailboxCounts(t, [5]int{1, 1, 0, 0, 0})
}

func TestMailboxDeliverActionIsNeverClaimedByGenericScheduler(t *testing.T) {
	ctx := context.Background()
	system := newMailboxTestSystem(t, 1)
	admitted := system.admit(t, 0, "mailbox-admit:generic-scheduler")
	processed, err := system.orchestrator.ProcessNext(ctx, "worker:must-not-consume-mailbox")
	if err != nil || processed.Processed {
		t.Fatalf("generic scheduler consumed mailbox: result=%+v err=%v", processed, err)
	}
	record, err := system.orchestrator.GetMailbox(ctx, system.recipientAccess, GetMailboxRequest{
		GoalRef: system.goalRef, MessageRef: admitted.Record.Envelope.Ref,
		RecipientWorkItemRef: system.parentRef, RecipientExecutionRef: system.parentExecution,
	})
	if err != nil || record.State != MailboxStateAdmitted || len(record.Attempts) != 0 {
		t.Fatalf("generic scheduler changed mailbox: record=%+v err=%v", record, err)
	}
	system.agent.mu.Lock()
	launches, observations := system.agent.launches, system.agent.observationCalls
	system.agent.mu.Unlock()
	if launches != 0 || observations != 0 {
		t.Fatalf("generic scheduler called provider: launches=%d observations=%d", launches, observations)
	}
}

func TestMailboxListUsesDeterministicFIFOOrder(t *testing.T) {
	ctx := context.Background()
	system := newMailboxTestSystem(t, 3)
	first := system.admit(t, 1, "mailbox-admit:fifo-same-time-b")
	second := system.admit(t, 0, "mailbox-admit:fifo-same-time-a")
	if !first.Record.Envelope.AdmittedAt.Equal(second.Record.Envelope.AdmittedAt) {
		t.Fatal("same-time FIFO fixture drifted")
	}
	system.clock.Advance(time.Second)
	third := system.admit(t, 2, "mailbox-admit:fifo-later")

	request := ListMailboxRequest{
		GoalRef: system.goalRef, RecipientWorkItemRef: system.parentRef,
		RecipientExecutionRef: system.parentExecution, Limit: 10,
	}
	listed, err := system.orchestrator.ListMailbox(ctx, system.recipientAccess, request)
	if err != nil || len(listed) != 3 {
		t.Fatalf("ListMailbox FIFO records=%d err=%v", len(listed), err)
	}
	earlier, later := first.Record.Envelope.Ref, second.Record.Envelope.Ref
	if earlier.String() > later.String() {
		earlier, later = later, earlier
	}
	want := []MailboxMessageRef{earlier, later, third.Record.Envelope.Ref}
	for index := range want {
		if listed[index].Envelope.Ref != want[index] {
			t.Fatalf("ListMailbox[%d]=%s want=%s", index,
				listed[index].Envelope.Ref.String(), want[index].String())
		}
	}
	again, err := system.orchestrator.ListMailbox(ctx, system.recipientAccess, request)
	if err != nil || !reflect.DeepEqual(again, listed) {
		t.Fatalf("ListMailbox order changed: first=%+v again=%+v err=%v", listed, again, err)
	}
}

func TestMailboxRejectsWrongRecipientAndSuccessorExecution(t *testing.T) {
	ctx := context.Background()
	system := newMailboxTestSystem(t, 1)
	admitted := system.admit(t, 0, "mailbox-admit:address")

	wrongPrincipal := system.claimRequest(admitted.Record, "mailbox-claim:wrong-principal")
	if _, err := system.orchestrator.ClaimMailbox(ctx, system.intruderAccess, wrongPrincipal); !IsStateError(err, StateConflict) {
		t.Fatalf("wrong principal claim error=%v", err)
	}
	successorExecution := mailboxMustRef(t, "execution:mailbox-parent-successor", goal.NewExecutionRef)
	successorAccess := mailboxMustExecutionAccess(t, system.recipient, system.project, successorExecution)
	wrongExecution := system.claimRequest(admitted.Record, "mailbox-claim:successor-execution")
	if _, err := system.orchestrator.ClaimMailbox(ctx, successorAccess, wrongExecution); !errors.Is(err, errForbidden) {
		t.Fatalf("successor execution claim error=%v", err)
	}
	system.assertMailboxCounts(t, [5]int{1, 0, 0, 0, 0})

	claim, err := system.orchestrator.ClaimMailbox(
		ctx, system.recipientAccess, system.claimRequest(admitted.Record, "mailbox-claim:exact"),
	)
	if err != nil || !claim.Claimed || claim.Claim.Attempt.Recipient != admitted.Record.Envelope.Recipient {
		t.Fatalf("exact recipient claim=%+v err=%v", claim, err)
	}
	if _, err = system.orchestrator.MarkMailboxDelivered(
		ctx, system.recipientAccess, system.deliverRequest(claim.Claim, "mailbox-deliver:address"),
	); err != nil {
		t.Fatalf("deliver exact recipient: %v", err)
	}
	if _, err = system.orchestrator.ConsumeMailbox(ctx, system.recipientAccess, ConsumeMailboxRequest{
		RequestRef: "mailbox-consume:address", GoalRef: system.goalRef,
		MessageRef: admitted.Record.Envelope.Ref, RecipientWorkItemRef: system.parentRef,
		RecipientExecutionRef: system.parentExecution, ClaimToken: claim.Claim.Attempt.ClaimToken,
		Fence: claim.Claim.Attempt.Fence,
	}); err != nil {
		t.Fatalf("consume exact recipient: %v", err)
	}
	current, _ := system.repository.GetGoal(ctx, system.goalRef)
	stolen := ResolveMailboxRequest{
		RequestRef: "mailbox-ack:stolen-principal", GoalRef: system.goalRef,
		MessageRef: admitted.Record.Envelope.Ref, RecipientWorkItemRef: system.parentRef,
		RecipientExecutionRef: system.parentExecution, ClaimToken: claim.Claim.Attempt.ClaimToken,
		Fence:                claim.Claim.Attempt.Fence,
		ExpectedGoalRevision: current.Goal.Revision(), ExpectedPlanGeneration: current.Goal.PlanGeneration(),
		EffectOrReworkRef: "effect:stolen-principal",
	}
	if _, err = system.orchestrator.AcknowledgeMailbox(ctx, system.intruderAccess, AcknowledgeMailboxRequest(stolen)); err == nil {
		t.Fatal("intruder acknowledged with stolen token/fence")
	}
	stolen.RequestRef = "mailbox-ack:successor-execution"
	stolen.EffectOrReworkRef = "effect:successor-execution"
	if _, err = system.orchestrator.AcknowledgeMailbox(ctx, successorAccess, AcknowledgeMailboxRequest(stolen)); !errors.Is(err, errForbidden) {
		t.Fatal("successor execution acknowledged predecessor mailbox")
	}
	if got := system.mailboxCounts(); got != ([5]int{1, 1, 1, 1, 0}) {
		t.Fatalf("rejected ACK wrote resolution: %v", got)
	}
	if current, _ = system.repository.GetGoal(ctx, system.goalRef); len(current.Goal.ChildHandoffResolutions()) != 0 {
		t.Fatal("rejected ACK resolved child handoff")
	}
	if _, err = system.orchestrator.GetMailbox(ctx, system.intruderAccess, GetMailboxRequest{
		GoalRef: system.goalRef, MessageRef: admitted.Record.Envelope.Ref,
		RecipientWorkItemRef: system.parentRef, RecipientExecutionRef: system.parentExecution,
	}); !IsStateError(err, StateNotFound) {
		t.Fatalf("intruder GetMailbox error=%v", err)
	}
	if _, err = system.orchestrator.GetMailbox(ctx, successorAccess, GetMailboxRequest{
		GoalRef: system.goalRef, MessageRef: admitted.Record.Envelope.Ref,
		RecipientWorkItemRef: system.parentRef, RecipientExecutionRef: system.parentExecution,
	}); !errors.Is(err, errForbidden) {
		t.Fatalf("successor GetMailbox error=%v", err)
	}
	listed, err := system.orchestrator.ListMailbox(ctx, system.intruderAccess, ListMailboxRequest{
		GoalRef: system.goalRef, RecipientWorkItemRef: system.parentRef,
		RecipientExecutionRef: system.parentExecution, Limit: 10,
	})
	if err != nil || len(listed) != 0 {
		t.Fatalf("intruder enumerated mailbox: records=%d err=%v", len(listed), err)
	}
	listed, err = system.orchestrator.ListMailbox(ctx, successorAccess, ListMailboxRequest{
		GoalRef: system.goalRef, RecipientWorkItemRef: system.parentRef,
		RecipientExecutionRef: system.parentExecution, Limit: 10,
	})
	if !errors.Is(err, errForbidden) || len(listed) != 0 {
		t.Fatalf("successor enumerated mailbox: records=%d err=%v", len(listed), err)
	}
}

func TestMailboxExpiredClaimReclaimsAfterCrash(t *testing.T) {
	ctx := context.Background()
	system := newMailboxTestSystem(t, 1)
	admitted := system.admit(t, 0, "mailbox-admit:reclaim")
	firstRequest := system.claimRequest(admitted.Record, "mailbox-claim:crashed")
	first, err := system.orchestrator.ClaimMailbox(ctx, system.recipientAccess, firstRequest)
	if err != nil || !first.Claimed {
		t.Fatalf("first claim=%+v err=%v", first, err)
	}
	assertHistoricalMailboxClaimReplay(t, system, firstRequest, first.Claim)
	system.clock.Advance(time.Minute + time.Second)
	assertHistoricalMailboxClaimReplay(t, system, firstRequest, first.Claim)
	second, err := system.orchestrator.ClaimMailbox(
		ctx, system.recipientAccess, system.claimRequest(admitted.Record, "mailbox-claim:recovered"),
	)
	if err != nil || !second.Claimed || second.Claim.Attempt.Fence != 2 ||
		second.Claim.Attempt.ClaimToken == first.Claim.Attempt.ClaimToken ||
		len(second.Claim.Record.Attempts) != 2 || second.Claim.Record.Attempts[0] != first.Claim.Attempt {
		t.Fatalf("reclaim did not preserve/fence attempts: second=%+v err=%v", second, err)
	}
	if _, err = system.orchestrator.ClaimMailbox(ctx, system.recipientAccess, firstRequest); !IsStateError(err, StateConflict) {
		t.Fatalf("superseded claim replay error=%v", err)
	}
	staleDelivery := system.deliverRequest(first.Claim, "mailbox-deliver:stale")
	if _, err = system.orchestrator.MarkMailboxDelivered(ctx, system.recipientAccess, staleDelivery); !IsStateError(err, StateConflict) {
		t.Fatalf("stale delivery fence error=%v", err)
	}
	currentDelivery := system.deliverRequest(second.Claim, "mailbox-deliver:recovered")
	if delivered, err := system.orchestrator.MarkMailboxDelivered(ctx, system.recipientAccess, currentDelivery); err != nil || !delivered.Changed {
		t.Fatalf("recovered delivery=%+v err=%v", delivered, err)
	}
	system.assertMailboxCounts(t, [5]int{1, 2, 1, 0, 0})
}

func TestMailboxClaimReplayPreservesHistoricalFrontierWithoutRenewal(t *testing.T) {
	ctx := context.Background()
	expired := newMailboxTestSystem(t, 1)
	expiredAdmission := expired.admit(t, 0, "mailbox-admit:claim-history-expired")
	expiredRequest := expired.claimRequest(expiredAdmission.Record, "mailbox-claim:claim-history-expired")
	expiredClaim, err := expired.orchestrator.ClaimMailbox(ctx, expired.recipientAccess, expiredRequest)
	if err != nil || !expiredClaim.Claimed {
		t.Fatalf("claim before expiry=%+v err=%v", expiredClaim, err)
	}
	assertHistoricalMailboxClaimReplay(t, expired, expiredRequest, expiredClaim.Claim)
	expired.clock.Advance(time.Minute + time.Second)
	assertHistoricalMailboxClaimReplay(t, expired, expiredRequest, expiredClaim.Claim)

	system := newMailboxTestSystem(t, 1)
	admitted := system.admit(t, 0, "mailbox-admit:claim-history")
	claimRequest := system.claimRequest(admitted.Record, "mailbox-claim:claim-history")
	claimed, err := system.orchestrator.ClaimMailbox(ctx, system.recipientAccess, claimRequest)
	if err != nil || !claimed.Claimed {
		t.Fatalf("claim history=%+v err=%v", claimed, err)
	}
	delivered, err := system.orchestrator.MarkMailboxDelivered(
		ctx, system.recipientAccess,
		system.deliverRequest(claimed.Claim, "mailbox-deliver:claim-history"),
	)
	if err != nil || !delivered.Changed {
		t.Fatalf("deliver history=%+v err=%v", delivered, err)
	}
	assertHistoricalMailboxClaimReplay(t, system, claimRequest, claimed.Claim)
	consumed, err := system.orchestrator.ConsumeMailbox(ctx, system.recipientAccess, ConsumeMailboxRequest{
		RequestRef: "mailbox-consume:claim-history", GoalRef: system.goalRef,
		MessageRef: admitted.Record.Envelope.Ref, RecipientWorkItemRef: system.parentRef,
		RecipientExecutionRef: system.parentExecution, ClaimToken: claimed.Claim.Attempt.ClaimToken,
		Fence: claimed.Claim.Attempt.Fence,
	})
	if err != nil || !consumed.Changed || consumed.Record.State != MailboxStateConsumed {
		t.Fatalf("consume history=%+v err=%v", consumed, err)
	}
	assertHistoricalMailboxClaimReplay(t, system, claimRequest, claimed.Claim)
	record, err := system.repository.GetGoal(ctx, system.goalRef)
	if err != nil || len(record.ConsumptionReceipts) != 1 ||
		record.ConsumptionReceipts[0].DeliveryAttempt != claimed.Claim.Attempt.Fence {
		t.Fatalf("generic receipt did not derive attempt from fence: record=%+v err=%v", record, err)
	}

	beforeStale := system.mailboxCounts()
	if _, err = system.orchestrator.MarkMailboxDelivered(
		ctx, system.recipientAccess,
		system.deliverRequest(claimed.Claim, "mailbox-deliver:claim-history-stale"),
	); !IsStateError(err, StateConflict) {
		t.Fatalf("consumed mailbox accepted stale delivery: %v", err)
	}
	if got := system.mailboxCounts(); got != beforeStale {
		t.Fatalf("stale delivery wrote mailbox state: got=%v want=%v", got, beforeStale)
	}

	flow := consumedMailboxFlow{admission: admitted, claim: claimed.Claim}
	resolution := system.resolveRequest(flow, "mailbox-ack:claim-history", "effect:claim-history")
	if _, err = system.orchestrator.AcknowledgeMailbox(
		ctx, system.recipientAccess, AcknowledgeMailboxRequest(resolution),
	); err != nil {
		t.Fatalf("terminal ACK: %v", err)
	}
	if _, err = system.orchestrator.ClaimMailbox(
		ctx, system.recipientAccess, claimRequest,
	); !IsStateError(err, StateConflict) {
		t.Fatalf("terminal ACK exposed historical claim replay: %v", err)
	}
}

func TestMailboxMutationReplayPreservesHistoricalFrontiersAfterTerminalState(t *testing.T) {
	t.Run("acknowledged", func(t *testing.T) {
		ctx := context.Background()
		system := newMailboxTestSystem(t, 1)
		flow := system.consume(t, 0, "mutation-history-ack")
		request := system.resolveRequest(
			flow, "mailbox-ack:mutation-history", "effect:mutation-history",
		)
		if _, err := system.orchestrator.AcknowledgeMailbox(
			ctx, system.recipientAccess, AcknowledgeMailboxRequest(request),
		); err != nil {
			t.Fatalf("ACK terminal fixture: %v", err)
		}
		assertHistoricalMailboxProgressReplay(t, system, flow.claim, "mutation-history-ack")
	})

	t.Run("retired", func(t *testing.T) {
		ctx := context.Background()
		system := newMailboxTestSystem(t, 1)
		admission := system.admit(t, 0, "mailbox-admit:mutation-history-retired")
		frontier := advanceMailboxRetirementFrontier(
			t, system, admission, "mutation-history-retired", 3,
		)
		beforeFailure, err := system.repository.GetGoal(ctx, system.goalRef)
		if err != nil {
			t.Fatal(err)
		}
		seedParentObservationAction(t, system)
		system.clock.Advance(time.Second)
		system.agent.mu.Lock()
		system.agent.observations = append(system.agent.observations, ports.AgentObservation{
			Status: ports.AgentFailed, ErrorCode: "agent.mutation_history_retired",
			SpecHash: beforeFailure.Goal.SpecHash(),
		})
		system.agent.mu.Unlock()
		if processed, processErr := system.orchestrator.ProcessNext(
			ctx, "worker:mutation-history-retired",
		); processErr != nil || !processed.Processed {
			t.Fatalf("retirement fixture result=%+v err=%v", processed, processErr)
		}
		retired, err := system.repository.GetMailbox(
			ctx, system.project, system.goalRef, admission.Record.Envelope.Ref,
			admission.Record.Envelope.Recipient,
		)
		if err != nil || retired.State != MailboxStateRetired || retired.Retirement == nil {
			t.Fatalf("retirement fixture=%+v err=%v", retired, err)
		}
		assertHistoricalMailboxProgressReplay(t, system, frontier.claim, "mutation-history-retired")
	})

	t.Run("superseded_delivery", func(t *testing.T) {
		ctx := context.Background()
		system := newMailboxTestSystem(t, 1)
		admission := system.admit(t, 0, "mailbox-admit:mutation-history-superseded")
		firstRequest := system.claimRequest(admission.Record, "mailbox-claim:mutation-history-first")
		first, err := system.orchestrator.ClaimMailbox(ctx, system.recipientAccess, firstRequest)
		if err != nil || !first.Claimed {
			t.Fatalf("first claim=%+v err=%v", first, err)
		}
		deliveryRequest := system.deliverRequest(first.Claim, "mailbox-deliver:mutation-history-first")
		if delivered, deliverErr := system.orchestrator.MarkMailboxDelivered(
			ctx, system.recipientAccess, deliveryRequest,
		); deliverErr != nil || !delivered.Changed {
			t.Fatalf("first delivery=%+v err=%v", delivered, deliverErr)
		}
		system.clock.Advance(time.Minute + time.Second)
		second, err := system.orchestrator.ClaimMailbox(
			ctx, system.recipientAccess,
			system.claimRequest(admission.Record, "mailbox-claim:mutation-history-second"),
		)
		if err != nil || !second.Claimed || second.Claim.Attempt.Fence != 2 {
			t.Fatalf("second claim=%+v err=%v", second, err)
		}
		before := second.Claim.Record
		counts := system.mailboxCounts()
		replayed, err := system.orchestrator.MarkMailboxDelivered(
			ctx, system.recipientAccess, deliveryRequest,
		)
		if err != nil || replayed.Changed || replayed.Record.State != MailboxStateDelivered ||
			len(replayed.Record.Attempts) != 1 || replayed.Record.Attempts[0].Fence != 1 {
			t.Fatalf("superseded delivery replay=%+v err=%v", replayed, err)
		}
		after, stateErr := system.repository.GetMailbox(
			ctx, system.project, system.goalRef, admission.Record.Envelope.Ref,
			admission.Record.Envelope.Recipient,
		)
		if stateErr != nil || !reflect.DeepEqual(after, before) || system.mailboxCounts() != counts {
			t.Fatalf("superseded delivery replay changed current frontier: before=%+v after=%+v err=%v",
				before, after, stateErr)
		}
	})
}

func TestMailboxSurvivesMonotonicPlanAppendForSameRecipientExecution(t *testing.T) {
	ctx := context.Background()
	system := newMailboxTestSystem(t, 1)
	admitted := system.admit(t, 0, "mailbox-admit:before-plan-append")

	system.repository.mu.Lock()
	record := system.repository.records[system.goalRef]
	phase := record.Goal.Phases()[0]
	appended, err := goal.NewWorkItem(goal.NewWorkItemInput{
		Ref:  mailboxMustRef(t, "work-item:mailbox-appended", goal.NewWorkItemRef),
		Goal: system.goalRef, Actor: system.source.ActorRef, Project: system.project,
		Objective: "independent later work", CreatedAt: system.clock.Now(), Phase: phase.Key(),
	})
	if err != nil {
		system.repository.mu.Unlock()
		t.Fatal(err)
	}
	items := append(record.Goal.WorkItems(), appended)
	plan, err := goal.NewPlan(goal.PlanInput{
		Generation: record.Goal.PlanGeneration() + 1,
		Phases:     record.Goal.Phases(), WorkItems: items,
	})
	if err == nil {
		record.Goal, err = record.Goal.ApplyPlan(record.Goal.Revision(), plan)
	}
	if err != nil {
		system.repository.mu.Unlock()
		t.Fatal(err)
	}
	system.repository.records[system.goalRef] = record
	system.repository.mu.Unlock()

	claim, err := system.orchestrator.ClaimMailbox(
		ctx, system.recipientAccess,
		system.claimRequest(admitted.Record, "mailbox-claim:after-plan-append"),
	)
	if err != nil || !claim.Claimed {
		t.Fatalf("claim after monotonic plan append=%+v err=%v", claim, err)
	}
	if _, err = system.orchestrator.MarkMailboxDelivered(
		ctx, system.recipientAccess,
		system.deliverRequest(claim.Claim, "mailbox-deliver:after-plan-append"),
	); err != nil {
		t.Fatal(err)
	}
	if _, err = system.orchestrator.ConsumeMailbox(ctx, system.recipientAccess, ConsumeMailboxRequest{
		RequestRef: "mailbox-consume:after-plan-append", GoalRef: system.goalRef,
		MessageRef: admitted.Record.Envelope.Ref, RecipientWorkItemRef: system.parentRef,
		RecipientExecutionRef: system.parentExecution, ClaimToken: claim.Claim.Attempt.ClaimToken,
		Fence: claim.Claim.Attempt.Fence,
	}); err != nil {
		t.Fatal(err)
	}
	current, _ := system.repository.GetGoal(ctx, system.goalRef)
	resolve := system.resolveRequest(
		consumedMailboxFlow{admission: admitted, claim: claim.Claim},
		"mailbox-ack:after-plan-append", "effect:integrated-after-append", current.Goal.Revision(),
	)
	resolve.ExpectedPlanGeneration = current.Goal.PlanGeneration()
	ack, err := system.orchestrator.AcknowledgeMailbox(
		ctx, system.recipientAccess, AcknowledgeMailboxRequest(resolve),
	)
	if err != nil || !ack.Created || ack.Acknowledgement.TargetPlanGeneration != 1 {
		t.Fatalf("ACK after monotonic plan append=%+v err=%v", ack, err)
	}
}

func TestMailboxAcknowledgementReplayNeverRedelivers(t *testing.T) {
	ctx := context.Background()
	system := newMailboxTestSystem(t, 1)
	flow := system.consume(t, 0, "replay")
	system.clock.Advance(time.Minute + time.Second)
	if _, err := system.orchestrator.ClaimMailbox(
		ctx, system.recipientAccess, system.claimRequest(flow.admission.Record, "mailbox-claim:post-consume-crash"),
	); !IsStateError(err, StateConflict) {
		t.Fatalf("consumed mailbox reclaimed after crash: %v", err)
	}
	request := system.resolveRequest(flow, "mailbox-ack:replay", "effect:parent-integrated")
	first, err := system.orchestrator.AcknowledgeMailbox(ctx, system.recipientAccess, AcknowledgeMailboxRequest(request))
	if err != nil || !first.Created || first.Acknowledgement.Outcome != MailboxOutcomeAcknowledged {
		t.Fatalf("first ACK=%+v err=%v", first, err)
	}
	if first.Acknowledgement.Ref == flow.admission.Record.Admission.Ref ||
		first.Acknowledgement.Ref == flow.claim.Attempt.ClaimToken {
		t.Fatal("admission, claim and recipient ACK collapsed into one fact")
	}
	counts := system.mailboxCounts()
	replayed, err := system.orchestrator.AcknowledgeMailbox(ctx, system.recipientAccess, AcknowledgeMailboxRequest(request))
	if err != nil || replayed.Created || replayed.Acknowledgement != first.Acknowledgement {
		t.Fatalf("ACK replay=%+v err=%v", replayed, err)
	}
	if got := system.mailboxCounts(); got != counts {
		t.Fatalf("ACK replay caused delivery/state write: got=%v want=%v", got, counts)
	}
	admissionReplay, err := system.orchestrator.AdmitMailbox(
		ctx, system.sourceAccess, system.admissionRequest(0, "mailbox-admit:replay"),
	)
	if err != nil || admissionReplay.Created ||
		admissionReplay.Record.Envelope.Ref != flow.admission.Record.Envelope.Ref ||
		admissionReplay.Record.State != MailboxStateAcknowledged {
		t.Fatalf("advanced admission replay=%+v err=%v", admissionReplay, err)
	}
	if _, err := system.orchestrator.ClaimMailbox(
		ctx, system.recipientAccess, system.claimRequest(flow.admission.Record, "mailbox-claim:after-ack"),
	); err == nil {
		t.Fatal("terminal ACK mailbox redelivered")
	}
	record, err := system.orchestrator.GetMailbox(ctx, system.recipientAccess, GetMailboxRequest{
		GoalRef: system.goalRef, MessageRef: flow.admission.Record.Envelope.Ref,
		RecipientWorkItemRef: system.parentRef, RecipientExecutionRef: system.parentExecution,
	})
	if err != nil || record.State != MailboxStateAcknowledged || len(record.Attempts) != 1 {
		t.Fatalf("terminal mailbox=%+v err=%v", record, err)
	}
	system.repository.mu.Lock()
	_, actionStillPending := system.repository.actions[record.Action.Ref]
	system.repository.mu.Unlock()
	if actionStillPending {
		t.Fatal("ACK left deliver_mailbox action pending")
	}
}

func TestMailboxParentClosureRequiresEveryChildResolution(t *testing.T) {
	system := newMailboxTestSystem(t, 2)
	ctx := context.Background()
	before, err := system.repository.GetGoal(ctx, system.goalRef)
	if err != nil {
		t.Fatal(err)
	}
	parent, _ := before.Goal.WorkItem(system.parentRef)
	_, err = before.Goal.SucceedWorkItem(
		before.Goal.Revision(), parent.Revision(), system.parentRef,
		[]goal.ArtifactRef{mailboxMustRef(t, "artifact:parent-before", goal.NewArtifactRef)},
		[]goal.AttestationRef{mailboxMustRef(t, "attestation:parent-before", goal.NewAttestationRef)},
		system.clock.Now(),
	)
	if goal.ErrorCodeOf(err) != goal.ErrorChildHandoffsPending {
		t.Fatalf("parent closed without child observations: %v", err)
	}

	first := system.consume(t, 0, "parent-first")
	firstState, _ := system.repository.GetGoal(ctx, system.goalRef)
	if _, err := system.orchestrator.AcknowledgeMailbox(
		ctx, system.recipientAccess,
		AcknowledgeMailboxRequest(system.resolveRequest(first, "mailbox-ack:parent-first", "effect:first-integrated", firstState.Goal.Revision())),
	); err != nil {
		t.Fatalf("first child ACK: %v", err)
	}
	afterFirst, _ := system.repository.GetGoal(ctx, system.goalRef)
	parent, _ = afterFirst.Goal.WorkItem(system.parentRef)
	_, err = afterFirst.Goal.SucceedWorkItem(
		afterFirst.Goal.Revision(), parent.Revision(), system.parentRef,
		[]goal.ArtifactRef{mailboxMustRef(t, "artifact:parent-one", goal.NewArtifactRef)},
		[]goal.AttestationRef{mailboxMustRef(t, "attestation:parent-one", goal.NewAttestationRef)},
		system.clock.Now().Add(time.Second),
	)
	if goal.ErrorCodeOf(err) != goal.ErrorChildHandoffsPending {
		t.Fatalf("one child ACK unlocked parent: %v", err)
	}

	second := system.consume(t, 1, "parent-second")
	secondState, _ := system.repository.GetGoal(ctx, system.goalRef)
	if _, err := system.orchestrator.BlockMailbox(
		ctx, system.recipientAccess,
		BlockMailboxRequest(system.resolveRequest(second, "mailbox-block:parent-second", "rework:child-blocked", secondState.Goal.Revision())),
	); err != nil {
		t.Fatalf("second child blockage: %v", err)
	}
	afterBoth, _ := system.repository.GetGoal(ctx, system.goalRef)
	resolutions := afterBoth.Goal.ChildHandoffResolutions()
	if len(resolutions) != 2 || resolutions[0].Outcome() != goal.ChildHandoffAcknowledged ||
		resolutions[1].Outcome() != goal.ChildHandoffBlocked {
		t.Fatalf("parent observations=%+v", resolutions)
	}
	parent, _ = afterBoth.Goal.WorkItem(system.parentRef)
	finished, err := afterBoth.Goal.SucceedWorkItem(
		afterBoth.Goal.Revision(), parent.Revision(), system.parentRef,
		[]goal.ArtifactRef{mailboxMustRef(t, "artifact:parent-final", goal.NewArtifactRef)},
		[]goal.AttestationRef{mailboxMustRef(t, "attestation:parent-final", goal.NewAttestationRef)},
		system.clock.Now().Add(time.Second),
	)
	if err != nil {
		t.Fatalf("all child resolutions did not unlock parent: %v", err)
	}
	if outcome, closable := finished.ClosableOutcome(); !closable || outcome != goal.GoalOutcomeSucceeded {
		t.Fatalf("resolved Goal closure=%q/%v", outcome, closable)
	}
}

func TestMailboxParentCompletedObservationWaitsWithoutArtifactOrAttemptExhaustion(t *testing.T) {
	ctx := context.Background()
	system := newMailboxTestSystem(t, 1)
	record, err := system.repository.GetGoal(ctx, system.goalRef)
	if err != nil {
		t.Fatal(err)
	}
	parent, _ := record.Goal.WorkItem(system.parentRef)
	action := ActionRecord{
		Ref: "action:observe:" + system.parentExecution.String(), Kind: ActionObserveAgent,
		GoalRef: system.goalRef, WorkItemRef: system.parentRef, ExecutionRef: system.parentExecution,
		PlanGeneration: record.Goal.PlanGeneration(), WorkItemGeneration: parent.Revision(),
		AvailableAt: system.clock.Now(),
	}
	system.repository.mu.Lock()
	system.repository.actions[action.Ref] = memoryAction{record: action}
	system.repository.mu.Unlock()
	for attempt := 0; attempt < 5; attempt++ {
		system.agent.mu.Lock()
		system.agent.observations = append(system.agent.observations, ports.AgentObservation{
			ExecutionRef: system.parentExecution, Status: ports.AgentCompleted,
			MediaType: "text/plain", Content: []byte("parent result must wait"),
			SpecHash: record.Goal.SpecHash(), ObservedAt: system.clock.Now(),
		})
		system.agent.mu.Unlock()
		processed, processErr := system.orchestrator.ProcessNext(ctx, "worker:mailbox-parent")
		if processErr != nil || !processed.Processed || processed.Action != ActionObserveAgent {
			t.Fatalf("completed parent observation %d: result=%+v err=%v", attempt+1, processed, processErr)
		}
		system.clock.Advance(time.Second)
	}
	after, err := system.repository.GetGoal(ctx, system.goalRef)
	if err != nil {
		t.Fatal(err)
	}
	parentAfter, _ := after.Goal.WorkItem(system.parentRef)
	parentExecution, found := executionForAction(after, action)
	if !found || parentAfter.State() != goal.WorkItemStateRunning ||
		parentExecution.State != ExecutionRunning || parentExecution.AttemptNo != 1 ||
		len(after.Executions) != len(record.Executions) || len(after.Artifacts) != 0 ||
		len(after.Attestations) != 0 || len(after.ConsumptionReceipts) != 0 ||
		!reflect.DeepEqual(after.Goal.Snapshot(), record.Goal.Snapshot()) {
		t.Fatalf("pending handoff consumed parent result/attempt: %+v", after)
	}
	system.agent.mu.Lock()
	observationCalls := system.agent.observationCalls
	system.agent.mu.Unlock()
	system.artifacts.mu.Lock()
	artifactWrites := len(system.artifacts.content)
	system.artifacts.mu.Unlock()
	system.repository.mu.Lock()
	pendingAction, stillPending := system.repository.actions[action.Ref]
	system.repository.mu.Unlock()
	if observationCalls != 5 || artifactWrites != 0 || !stillPending || pendingAction.token != "" ||
		!pendingAction.record.AvailableAt.Equal(system.clock.Now()) {
		t.Fatalf("wait guard calls=%d artifacts=%d pending=%v action=%+v",
			observationCalls, artifactWrites, stillPending, pendingAction)
	}
}

func TestMailboxFailedRecipientNeedsNoForgedChildResolution(t *testing.T) {
	ctx := context.Background()
	system := newMailboxTestSystem(t, 1)
	record, err := system.repository.GetGoal(ctx, system.goalRef)
	if err != nil {
		t.Fatal(err)
	}
	parent, _ := record.Goal.WorkItem(system.parentRef)
	failedAt := system.clock.Now()
	failed, err := record.Goal.FailWorkItem(
		record.Goal.Revision(), parent.Revision(), parent.Ref(), failedAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	record.Goal = failed
	for index := range record.Executions {
		if record.Executions[index].Ref != system.parentExecution {
			continue
		}
		record.Executions[index].State = ExecutionFailed
		record.Executions[index].FinishedAt = failedAt
		record.Executions[index].FailureCode = "agent.parent_failed"
	}
	system.repository.mu.Lock()
	system.repository.records[system.goalRef] = record
	system.repository.mu.Unlock()
	if _, err = system.orchestrator.AdmitMailbox(
		ctx, system.sourceAccess, system.admissionRequest(0, "mailbox-admit:failed-recipient"),
	); !IsStateError(err, StateConflict) {
		t.Fatalf("mailbox admitted to failed recipient: %v", err)
	}
	outcome, closable := failed.ClosableOutcome()
	if !closable || outcome != goal.GoalOutcomeFailed {
		t.Fatalf("failed parent closure=%q/%v", outcome, closable)
	}
	closed, err := failed.Close(failed.Revision(), outcome, failedAt)
	if err != nil || closed.State() != goal.GoalStateFailed || len(closed.ChildHandoffResolutions()) != 0 {
		t.Fatalf("failed parent required forged resolution: goal=%+v err=%v", closed.Snapshot(), err)
	}
	if system.mailboxCounts() != ([5]int{}) {
		t.Fatalf("failed recipient wrote mailbox facts: %v", system.mailboxCounts())
	}
}

func TestMailboxAcknowledgedAndBlockedReceiptsAreImmutable(t *testing.T) {
	ctx := context.Background()
	system := newMailboxTestSystem(t, 2)
	ackFlow := system.consume(t, 0, "immutable-ack")
	ackState, _ := system.repository.GetGoal(ctx, system.goalRef)
	ackRequest := system.resolveRequest(
		ackFlow, "mailbox-ack:immutable", "effect:immutable", ackState.Goal.Revision(),
	)
	ack, err := system.orchestrator.AcknowledgeMailbox(ctx, system.recipientAccess, AcknowledgeMailboxRequest(ackRequest))
	if err != nil || !ack.Created {
		t.Fatalf("immutable ACK=%+v err=%v", ack, err)
	}

	blockFlow := system.consume(t, 1, "immutable-block")
	blockState, _ := system.repository.GetGoal(ctx, system.goalRef)
	blockRequest := system.resolveRequest(
		blockFlow, "mailbox-block:immutable", "rework:immutable", blockState.Goal.Revision(),
	)
	blocked, err := system.orchestrator.BlockMailbox(ctx, system.recipientAccess, BlockMailboxRequest(blockRequest))
	if err != nil || !blocked.Created || blocked.Acknowledgement.Outcome != MailboxOutcomeBlocked {
		t.Fatalf("immutable blockage=%+v err=%v", blocked, err)
	}
	for _, terminal := range []struct {
		name     string
		flow     consumedMailboxFlow
		claimRef string
	}{
		{name: "acknowledged", flow: ackFlow, claimRef: "mailbox-claim:immutable-ack"},
		{name: "blocked", flow: blockFlow, claimRef: "mailbox-claim:immutable-block"},
	} {
		if _, replayErr := system.orchestrator.ClaimMailbox(
			ctx, system.recipientAccess,
			system.claimRequest(terminal.flow.admission.Record, terminal.claimRef),
		); !IsStateError(replayErr, StateConflict) {
			t.Fatalf("%s mailbox exposed historical claim replay: %v", terminal.name, replayErr)
		}
	}

	changedReplay := ackRequest
	changedReplay.EffectOrReworkRef = "effect:contradiction"
	if _, err := system.orchestrator.AcknowledgeMailbox(ctx, system.recipientAccess, AcknowledgeMailboxRequest(changedReplay)); !IsStateError(err, StateConflict) {
		t.Fatalf("ACK receipt rewrite error=%v", err)
	}
	currentState, _ := system.repository.GetGoal(ctx, system.goalRef)
	if _, err := system.orchestrator.BlockMailbox(ctx, system.recipientAccess, BlockMailboxRequest(ResolveMailboxRequest{
		RequestRef: "mailbox-block:contradict-ack", GoalRef: ackRequest.GoalRef,
		MessageRef: ackRequest.MessageRef, RecipientWorkItemRef: ackRequest.RecipientWorkItemRef,
		RecipientExecutionRef: ackRequest.RecipientExecutionRef, ClaimToken: ackRequest.ClaimToken,
		Fence:                ackRequest.Fence,
		ExpectedGoalRevision: currentState.Goal.Revision(), ExpectedPlanGeneration: ackRequest.ExpectedPlanGeneration,
		EffectOrReworkRef: "rework:contradiction",
	})); err == nil {
		t.Fatal("acknowledged receipt changed to blocked")
	}

	record, err := system.orchestrator.GetMailbox(ctx, system.recipientAccess, GetMailboxRequest{
		GoalRef: system.goalRef, MessageRef: ackRequest.MessageRef,
		RecipientWorkItemRef: system.parentRef, RecipientExecutionRef: system.parentExecution,
	})
	if err != nil || record.Acknowledgement == nil || *record.Acknowledgement != ack.Acknowledgement {
		t.Fatalf("persisted ACK changed: record=%+v err=%v", record, err)
	}
	record.Attempts[0].ClaimToken = "mailbox-claim:mutated-copy"
	record.Acknowledgement.EffectOrReworkRef = "effect:mutated-copy"
	again, err := system.orchestrator.GetMailbox(ctx, system.recipientAccess, GetMailboxRequest{
		GoalRef: system.goalRef, MessageRef: ackRequest.MessageRef,
		RecipientWorkItemRef: system.parentRef, RecipientExecutionRef: system.parentExecution,
	})
	if err != nil || again.Attempts[0].ClaimToken == record.Attempts[0].ClaimToken ||
		again.Acknowledgement.EffectOrReworkRef != ack.Acknowledgement.EffectOrReworkRef {
		t.Fatalf("mailbox record exposed mutable receipt: %+v err=%v", again, err)
	}
}

type consumedMailboxFlow struct {
	admission MailboxAdmissionResult
	claim     MailboxClaim
}

func newMailboxTestSystem(t *testing.T, childCount int) *mailboxTestSystem {
	t.Helper()
	base := time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC)
	clock := &mutableClock{now: base}
	repository := newMemoryRepository()
	accessStore := newMemoryAccessRepository()
	accessStore.defaultRole = ""
	agent := &scriptedAgent{now: clock.Now}
	orchestrator, artifacts := newTestOrchestratorWithAccess(t, repository, accessStore, clock, agent)
	project := mailboxMustRef(t, "project:mailbox", goal.NewProjectRef)
	source := testPrincipal(t, "principal:mailbox-source", "actor:mailbox-source", identity.PrincipalKindService)
	recipient := testPrincipal(t, "principal:mailbox-recipient", "actor:mailbox-recipient", identity.PrincipalKindService)
	intruder := testPrincipal(t, "principal:mailbox-intruder", "actor:mailbox-intruder", identity.PrincipalKindService)
	for _, principal := range []identity.Principal{source, recipient, intruder} {
		membership, err := identity.NewMembership(identity.MembershipInput{
			PrincipalRef: principal.Ref, ProjectRef: project, Role: identity.RoleOperator,
			Revision: 1, Status: identity.MembershipActive,
			GrantedBy: principal.Ref, GrantedAt: base.Add(-time.Hour),
		})
		if err != nil {
			t.Fatal(err)
		}
		accessStore.seedMembership(membership)
	}
	goalRef := mailboxMustRef(t, "goal:mailbox", goal.NewGoalRef)
	intent, err := goal.NewIntentManifest(goal.IntentManifestInput{
		Ref: mailboxMustRef(t, "intent:mailbox", goal.NewIntentRef), Actor: source.ActorRef,
		Project: project, Statement: "coordinate exact child handoffs", SubmittedAt: base.Add(-30 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	spec, err := goal.NewInitialAppSpec(goal.AppSpecInput{
		Ref: mailboxMustRef(t, "app-spec:mailbox", goal.NewAppSpecRef), Intent: intent,
		Objective: "close only after child handoffs", Reason: "operator.confirmed",
		ConfirmedBy: source.ActorRef, ConfirmedAt: base.Add(-29 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	aggregate, err := goal.NewGoal(goalRef, spec, base.Add(-28*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	phase, err := goal.NewPhaseInstance(goal.DefaultPhaseKey())
	if err != nil {
		t.Fatal(err)
	}
	parentRef := mailboxMustRef(t, "work-item:mailbox-parent", goal.NewWorkItemRef)
	parent, err := goal.NewWorkItem(goal.NewWorkItemInput{
		Ref: parentRef, Goal: goalRef, Actor: source.ActorRef, Project: project,
		Objective: "integrate child handoffs", CreatedAt: base.Add(-27 * time.Minute), Phase: phase.Key(),
	})
	if err != nil {
		t.Fatal(err)
	}
	items := []goal.WorkItem{parent}
	children := make([]goal.WorkItemRef, 0, childCount)
	childExecutions := make([]goal.ExecutionRef, 0, childCount)
	childArtifacts := make([]goal.ArtifactRef, 0, childCount)
	for index := 0; index < childCount; index++ {
		childRef := mailboxMustRef(t, fmt.Sprintf("work-item:mailbox-child-%d", index+1), goal.NewWorkItemRef)
		child, err := goal.NewWorkItem(goal.NewWorkItemInput{
			Ref: childRef, Goal: goalRef, Actor: source.ActorRef, Project: project,
			Objective: fmt.Sprintf("produce child %d result", index+1),
			CreatedAt: base.Add(-27 * time.Minute), Phase: phase.Key(), Parent: parentRef,
			HandoffRequired: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, child)
		children = append(children, childRef)
		childExecutions = append(childExecutions,
			mailboxMustRef(t, fmt.Sprintf("execution:mailbox-child-%d", index+1), goal.NewExecutionRef))
		childArtifacts = append(childArtifacts,
			mailboxMustRef(t, fmt.Sprintf("artifact:mailbox-child-%d", index+1), goal.NewArtifactRef))
	}
	plan, err := goal.NewPlan(goal.PlanInput{
		Generation: 1, Phases: []goal.PhaseInstance{phase}, WorkItems: items,
	})
	if err != nil {
		t.Fatal(err)
	}
	aggregate, err = aggregate.ApplyPlan(aggregate.Revision(), plan)
	if err != nil {
		t.Fatal(err)
	}
	aggregate, err = aggregate.Start(aggregate.Revision(), base.Add(-26*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	parentExecution := mailboxMustRef(t, "execution:mailbox-parent", goal.NewExecutionRef)
	parent, _ = aggregate.WorkItem(parentRef)
	aggregate, err = aggregate.StartWorkItem(
		aggregate.Revision(), parent.Revision(), parentRef, parentExecution, base.Add(-25*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	for index, childRef := range children {
		child, _ := aggregate.WorkItem(childRef)
		aggregate, err = aggregate.StartWorkItem(
			aggregate.Revision(), child.Revision(), childRef, childExecutions[index], base.Add(-25*time.Minute),
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	for index, childRef := range children {
		child, _ := aggregate.WorkItem(childRef)
		aggregate, err = aggregate.SucceedWorkItem(
			aggregate.Revision(), child.Revision(), childRef,
			[]goal.ArtifactRef{childArtifacts[index]},
			[]goal.AttestationRef{mailboxMustRef(t, fmt.Sprintf("attestation:mailbox-child-%d", index+1), goal.NewAttestationRef)},
			base.Add(-20*time.Minute+time.Duration(index)*time.Second),
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	executions := []ExecutionRecord{{
		Ref: parentExecution, GoalRef: goalRef, WorkItemRef: parentRef,
		AttemptNo: 1, MaxExecutionAttempts: 3, PlanGeneration: 1,
		AppSpecGeneration: spec.Generation(), SpecHash: spec.Hash(), State: ExecutionRunning,
		ArtifactMediaType: "text/plain", IdempotencyKey: "execution:" + parentExecution.String(),
		MaxOutputBytes: 1 << 20, ProviderRef: "provider:test", ModelRef: "model:test",
		AgentRef: "agent:test", ExternalRef: "external:" + parentExecution.String(),
		CreatedAt: base.Add(-25 * time.Minute), StartedAt: base.Add(-25 * time.Minute),
		ProviderAcceptedAt: base.Add(-25 * time.Minute), DeadlineAt: base.Add(time.Hour),
	}}
	for index, childRef := range children {
		executions = append(executions, ExecutionRecord{
			Ref: childExecutions[index], GoalRef: goalRef, WorkItemRef: childRef,
			AttemptNo: 1, MaxExecutionAttempts: 3, PlanGeneration: 1,
			AppSpecGeneration: spec.Generation(), SpecHash: spec.Hash(), State: ExecutionSucceeded,
			ArtifactMediaType: "text/plain", IdempotencyKey: "execution:" + childExecutions[index].String(),
			MaxOutputBytes: 1 << 20, ProviderRef: "provider:test", ModelRef: "model:test",
			AgentRef: "agent:test", ExternalRef: "external:" + childExecutions[index].String(),
			CreatedAt: base.Add(-25 * time.Minute), StartedAt: base.Add(-25 * time.Minute),
			ProviderAcceptedAt: base.Add(-25 * time.Minute), DeadlineAt: base.Add(time.Hour),
			FinishedAt: base.Add(-20*time.Minute + time.Duration(index)*time.Second),
		})
	}
	repository.records[goalRef] = GoalRecord{
		RequestRef: "request:mailbox-fixture", RequestFingerprint: "fixture",
		RequestedBy: source.Ref, Goal: aggregate, Executions: executions,
	}
	sourceAccesses := make([]Access, len(childExecutions))
	for index, executionRef := range childExecutions {
		sourceAccesses[index] = mailboxMustExecutionAccess(t, source, project, executionRef)
	}
	recipientAccess := mailboxMustExecutionAccess(t, recipient, project, parentExecution)
	intruderAccess := mailboxMustExecutionAccess(t, intruder, project, parentExecution)
	return &mailboxTestSystem{
		clock: clock, repository: repository, accessStore: accessStore, agent: agent,
		artifacts: artifacts, orchestrator: orchestrator,
		project: project, source: source, recipient: recipient, intruder: intruder,
		sourceAccess: sourceAccesses[0], sourceAccesses: sourceAccesses,
		recipientAccess: recipientAccess, intruderAccess: intruderAccess,
		goalRef: goalRef, parentRef: parentRef, parentExecution: parentExecution,
		children: children, childExecutions: childExecutions, childArtifacts: childArtifacts,
	}
}

func (system *mailboxTestSystem) admissionRequest(index int, requestRef string) AdmitMailboxRequest {
	return AdmitMailboxRequest{
		RequestRef: requestRef, GoalRef: system.goalRef, ExpectedPlanGeneration: 1,
		Kind: MailboxKindChildDelivery, ParentWorkItemRef: system.parentRef,
		ChildWorkItemRef: system.children[index], SourceExecutionRef: system.childExecutions[index],
		RecipientPrincipalRef: system.recipient.Ref, RecipientExecutionRef: system.parentExecution,
		Summary:      fmt.Sprintf("compact result from child %d", index+1),
		ArtifactRefs: []goal.ArtifactRef{system.childArtifacts[index]},
	}
}

func (system *mailboxTestSystem) admit(t *testing.T, index int, requestRef string) MailboxAdmissionResult {
	t.Helper()
	result, err := system.orchestrator.AdmitMailbox(
		context.Background(), system.sourceAccesses[index], system.admissionRequest(index, requestRef),
	)
	if err != nil || !result.Created {
		t.Fatalf("admit child %d: result=%+v err=%v", index, result, err)
	}
	return result
}

func (system *mailboxTestSystem) claimRequest(record MailboxRecord, requestRef string) ClaimMailboxRequest {
	return ClaimMailboxRequest{
		RequestRef: requestRef, GoalRef: system.goalRef, MessageRef: record.Envelope.Ref,
		RecipientWorkItemRef: system.parentRef, RecipientExecutionRef: system.parentExecution,
	}
}

func (system *mailboxTestSystem) deliverRequest(claim MailboxClaim, requestRef string) MarkMailboxDeliveredRequest {
	return MarkMailboxDeliveredRequest{
		RequestRef: requestRef, GoalRef: system.goalRef, MessageRef: claim.Record.Envelope.Ref,
		RecipientWorkItemRef: system.parentRef, RecipientExecutionRef: system.parentExecution,
		ClaimToken: claim.Attempt.ClaimToken, Fence: claim.Attempt.Fence,
	}
}

func (system *mailboxTestSystem) consume(t *testing.T, index int, suffix string) consumedMailboxFlow {
	t.Helper()
	ctx := context.Background()
	admission := system.admit(t, index, "mailbox-admit:"+suffix)
	claimResult, err := system.orchestrator.ClaimMailbox(
		ctx, system.recipientAccess, system.claimRequest(admission.Record, "mailbox-claim:"+suffix),
	)
	if err != nil || !claimResult.Claimed {
		t.Fatalf("claim %s: result=%+v err=%v", suffix, claimResult, err)
	}
	claim := claimResult.Claim
	if delivered, err := system.orchestrator.MarkMailboxDelivered(
		ctx, system.recipientAccess, system.deliverRequest(claim, "mailbox-deliver:"+suffix),
	); err != nil || !delivered.Changed {
		t.Fatalf("deliver %s: result=%+v err=%v", suffix, delivered, err)
	}
	consumed, err := system.orchestrator.ConsumeMailbox(ctx, system.recipientAccess, ConsumeMailboxRequest{
		RequestRef: "mailbox-consume:" + suffix, GoalRef: system.goalRef,
		MessageRef: admission.Record.Envelope.Ref, RecipientWorkItemRef: system.parentRef,
		RecipientExecutionRef: system.parentExecution, ClaimToken: claim.Attempt.ClaimToken,
		Fence: claim.Attempt.Fence,
	})
	if err != nil || !consumed.Changed || consumed.Record.State != MailboxStateConsumed {
		t.Fatalf("consume %s: result=%+v err=%v", suffix, consumed, err)
	}
	return consumedMailboxFlow{admission: admission, claim: claim}
}

func (system *mailboxTestSystem) resolveRequest(
	flow consumedMailboxFlow,
	requestRef string,
	effectRef string,
	expectedRevision ...goal.Revision,
) ResolveMailboxRequest {
	revision := goal.Revision(0)
	if len(expectedRevision) > 0 {
		revision = expectedRevision[0]
	} else {
		system.repository.mu.Lock()
		revision = system.repository.records[system.goalRef].Goal.Revision()
		system.repository.mu.Unlock()
	}
	return ResolveMailboxRequest{
		RequestRef: requestRef, GoalRef: system.goalRef, MessageRef: flow.admission.Record.Envelope.Ref,
		RecipientWorkItemRef: system.parentRef, RecipientExecutionRef: system.parentExecution,
		ClaimToken: flow.claim.Attempt.ClaimToken, Fence: flow.claim.Attempt.Fence,
		ExpectedGoalRevision:   revision,
		ExpectedPlanGeneration: 1, EffectOrReworkRef: effectRef,
	}
}

func (system *mailboxTestSystem) mailboxCounts() [5]int {
	system.repository.mu.Lock()
	defer system.repository.mu.Unlock()
	return [5]int{
		system.repository.mailboxAdmits,
		system.repository.mailboxClaims,
		system.repository.mailboxDeliveries,
		system.repository.mailboxConsumptions,
		system.repository.mailboxResolutions,
	}
}

func (system *mailboxTestSystem) assertMailboxCounts(t *testing.T, want [5]int) {
	t.Helper()
	if got := system.mailboxCounts(); got != want {
		t.Fatalf("mailbox writes=%v want=%v", got, want)
	}
}

func assertHistoricalMailboxClaimReplay(
	t *testing.T,
	system *mailboxTestSystem,
	request ClaimMailboxRequest,
	want MailboxClaim,
) {
	t.Helper()
	ctx := context.Background()
	endpoint := want.Record.Envelope.Recipient
	before, err := system.repository.GetMailbox(
		ctx, system.project, system.goalRef, request.MessageRef, endpoint,
	)
	if err != nil {
		t.Fatal(err)
	}
	counts := system.mailboxCounts()
	ids := system.orchestrator.ids.(*sequentialIDs)
	ids.mu.Lock()
	idCount := ids.next
	ids.mu.Unlock()
	system.accessStore.mu.Lock()
	authorizations := len(system.accessStore.authorizations)
	system.accessStore.mu.Unlock()

	replayed, err := system.orchestrator.ClaimMailbox(ctx, system.recipientAccess, request)
	if err != nil || replayed.Claimed || !reflect.DeepEqual(replayed.Claim, want) {
		t.Fatalf("historical claim replay=%+v want=%+v err=%v", replayed, want, err)
	}
	after, stateErr := system.repository.GetMailbox(
		ctx, system.project, system.goalRef, request.MessageRef, endpoint,
	)
	ids.mu.Lock()
	idsAfter := ids.next
	ids.mu.Unlock()
	system.accessStore.mu.Lock()
	authorizationsAfter := len(system.accessStore.authorizations)
	system.accessStore.mu.Unlock()
	if stateErr != nil || !reflect.DeepEqual(after, before) || system.mailboxCounts() != counts ||
		idsAfter != idCount || authorizationsAfter != authorizations {
		t.Fatalf("claim replay caused effects: state=%v/%v counts=%v/%v ids=%d/%d auth=%d/%d err=%v",
			before, after, counts, system.mailboxCounts(), idCount, idsAfter,
			authorizations, authorizationsAfter, stateErr)
	}
}

func assertHistoricalMailboxProgressReplay(
	t *testing.T,
	system *mailboxTestSystem,
	claim MailboxClaim,
	suffix string,
) {
	t.Helper()
	ctx := context.Background()
	messageRef := claim.Record.Envelope.Ref
	endpoint := claim.Record.Envelope.Recipient
	before, err := system.repository.GetMailbox(
		ctx, system.project, system.goalRef, messageRef, endpoint,
	)
	if err != nil {
		t.Fatal(err)
	}
	counts := system.mailboxCounts()
	ids := system.orchestrator.ids.(*sequentialIDs)
	ids.mu.Lock()
	idCount := ids.next
	ids.mu.Unlock()
	system.accessStore.mu.Lock()
	authorizations := len(system.accessStore.authorizations)
	system.accessStore.mu.Unlock()

	delivery, err := system.orchestrator.MarkMailboxDelivered(
		ctx, system.recipientAccess,
		system.deliverRequest(claim, "mailbox-deliver:"+suffix),
	)
	if err != nil || delivery.Changed || delivery.Record.State != MailboxStateDelivered ||
		delivery.Record.Acknowledgement != nil || delivery.Record.Retirement != nil ||
		len(delivery.Record.Attempts) == 0 ||
		delivery.Record.Attempts[len(delivery.Record.Attempts)-1].ConsumptionRef != "" {
		t.Fatalf("historical delivery replay=%+v err=%v", delivery, err)
	}
	consumption, err := system.orchestrator.ConsumeMailbox(
		ctx, system.recipientAccess, ConsumeMailboxRequest{
			RequestRef: "mailbox-consume:" + suffix, GoalRef: system.goalRef,
			MessageRef: messageRef, RecipientWorkItemRef: system.parentRef,
			RecipientExecutionRef: system.parentExecution, ClaimToken: claim.Attempt.ClaimToken,
			Fence: claim.Attempt.Fence,
		},
	)
	if err != nil || consumption.Changed || consumption.Record.State != MailboxStateConsumed ||
		consumption.Record.Acknowledgement != nil || consumption.Record.Retirement != nil ||
		len(consumption.Record.Attempts) == 0 ||
		consumption.Record.Attempts[len(consumption.Record.Attempts)-1].ConsumptionRef == "" {
		t.Fatalf("historical consumption replay=%+v err=%v", consumption, err)
	}
	after, stateErr := system.repository.GetMailbox(
		ctx, system.project, system.goalRef, messageRef, endpoint,
	)
	ids.mu.Lock()
	idsAfter := ids.next
	ids.mu.Unlock()
	system.accessStore.mu.Lock()
	authorizationsAfter := len(system.accessStore.authorizations)
	system.accessStore.mu.Unlock()
	if stateErr != nil || !reflect.DeepEqual(after, before) || system.mailboxCounts() != counts ||
		idsAfter != idCount || authorizationsAfter != authorizations {
		t.Fatalf("historical mutation replay caused effects: counts=%v/%v ids=%d/%d auth=%d/%d err=%v",
			counts, system.mailboxCounts(), idCount, idsAfter,
			authorizations, authorizationsAfter, stateErr)
	}
}

func mailboxMustRef[T any](t *testing.T, value string, constructor func(string) (T, error)) T {
	t.Helper()
	ref, err := constructor(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func mailboxMustExecutionAccess(
	t *testing.T,
	principal identity.Principal,
	projectRef goal.ProjectRef,
	executionRef goal.ExecutionRef,
) Access {
	t.Helper()
	access, err := NewExecutionAccess(principal, projectRef, executionRef)
	if err != nil {
		t.Fatal(err)
	}
	return access
}
