package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type memoryRepository struct {
	mu                  sync.Mutex
	records             map[goal.GoalRef]GoalRecord
	requests            map[string]goal.GoalRef
	successors          map[goal.AppSpecRef]goal.GoalRef
	actions             map[string]memoryAction
	events              []EventRecord
	directorLeases      map[goal.GoalRef]DirectorLeaseRecord
	directorRequests    map[string]memoryDirectorMutation
	mailboxes           map[MailboxMessageRef]MailboxRecord
	mailboxRequests     map[string]memoryMailboxMutation
	mailboxAdmits       int
	mailboxClaims       int
	mailboxDeliveries   int
	mailboxConsumptions int
	mailboxResolutions  int
	now                 func() time.Time
}

type memoryMailboxMutation struct {
	request MailboxReplayRequest
	replay  MailboxReplayRecord
}

type memoryAction struct {
	record          ActionRecord
	token           string
	workerRef       string
	deliveryAttempt uint64
	fence           uint64
	lease           time.Time
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		records:          make(map[goal.GoalRef]GoalRecord),
		requests:         make(map[string]goal.GoalRef),
		successors:       make(map[goal.AppSpecRef]goal.GoalRef),
		actions:          make(map[string]memoryAction),
		directorLeases:   make(map[goal.GoalRef]DirectorLeaseRecord),
		directorRequests: make(map[string]memoryDirectorMutation),
		mailboxes:        make(map[MailboxMessageRef]MailboxRecord),
		mailboxRequests:  make(map[string]memoryMailboxMutation),
		now:              time.Now,
	}
}

func (repository *memoryRepository) CreateGoal(_ context.Context, state CreateGoalState) (GoalRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	intent := state.Goal.AppSpec().Intent()
	if !memoryWriteAuthorizationValid(
		state.AuthorizationReceipt, state.RequestedBy, state.Goal.Project(), identity.PermissionGoalsCreate,
		state.Goal.Project().String(),
	) {
		return GoalRecord{}, false, &StateError{Code: StateInvalid}
	}
	requestKey := state.RequestedBy.String() + "\x00" + state.Goal.Project().String() + "\x00" + state.RequestRef
	if ref, ok := repository.requests[requestKey]; ok {
		record := repository.records[ref]
		if record.RequestFingerprint != state.RequestFingerprint ||
			record.Goal.Actor() != state.Goal.Actor() ||
			record.Goal.Project() != state.Goal.Project() ||
			record.Goal.AppSpec().Intent().Statement() != intent.Statement() ||
			record.Goal.AppSpec().Objective() != state.Goal.AppSpec().Objective() {
			return GoalRecord{}, false, &StateError{Code: StateConflict}
		}
		return cloneGoalRecord(record), false, nil
	}
	record := GoalRecord{
		RequestRef: state.RequestRef, RequestFingerprint: state.RequestFingerprint,
		RequestedBy: state.RequestedBy,
		Goal:        state.Goal, Executions: append([]ExecutionRecord(nil), state.Executions...),
	}
	repository.requests[requestKey] = state.Goal.Ref()
	repository.records[state.Goal.Ref()] = record
	for _, action := range state.Actions {
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	repository.events = append(repository.events, state.Events...)
	return cloneGoalRecord(record), true, nil
}

func (repository *memoryRepository) AmendGoal(_ context.Context, state AmendGoalState) (GoalRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if !memoryWriteAuthorizationValid(
		state.AuthorizationReceipt, state.RequestedBy, state.ProjectRef, identity.PermissionGoalsAmend,
		state.SourceGoalRef.String(),
	) {
		return GoalRecord{}, false, &StateError{Code: StateInvalid}
	}
	requestKey := state.RequestedBy.String() + "\x00" + state.ProjectRef.String() + "\x00" + state.RequestRef
	if ref, ok := repository.requests[requestKey]; ok {
		record := repository.records[ref]
		if record.RequestFingerprint != state.RequestFingerprint {
			return GoalRecord{}, false, &StateError{Code: StateConflict}
		}
		return cloneGoalRecord(record), false, nil
	}
	source, ok := repository.records[state.SourceGoalRef]
	if !ok {
		return GoalRecord{}, false, &StateError{Code: StateNotFound}
	}
	if source.Goal.Project() != state.ProjectRef ||
		source.Goal.Revision() != state.ExpectedSourceRevision ||
		source.Goal.SpecHash() != state.ExpectedSourceSpecHash || !source.Goal.IsTerminal() {
		return GoalRecord{}, false, &StateError{Code: StateConflict}
	}
	if _, exists := repository.successors[source.Goal.AppSpec().Ref()]; exists {
		return GoalRecord{}, false, &StateError{Code: StateConflict}
	}
	successorSpec := state.Successor.AppSpec()
	parentRef, hasParent := successorSpec.ParentRef()
	if state.Successor.Actor() != source.Goal.Actor() || state.Successor.Project() != state.ProjectRef ||
		state.Successor.State() != goal.GoalStatePending || state.Successor.WorkItemCount() != 0 ||
		!hasParent || parentRef != source.Goal.AppSpec().Ref() ||
		successorSpec.ParentHash() != source.Goal.SpecHash() ||
		successorSpec.Generation() != source.Goal.AppSpec().Generation()+1 {
		return GoalRecord{}, false, &StateError{Code: StateConflict}
	}
	record := GoalRecord{
		RequestRef: state.RequestRef, RequestFingerprint: state.RequestFingerprint,
		RequestedBy: state.RequestedBy,
		Goal:        state.Successor,
	}
	repository.requests[requestKey] = state.Successor.Ref()
	repository.records[state.Successor.Ref()] = record
	repository.successors[source.Goal.AppSpec().Ref()] = state.Successor.Ref()
	repository.events = append(repository.events, state.Events...)
	return cloneGoalRecord(record), true, nil
}

func (repository *memoryRepository) GetGoal(_ context.Context, ref goal.GoalRef) (GoalRecord, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	record, ok := repository.records[ref]
	if !ok {
		return GoalRecord{}, &StateError{Code: StateNotFound}
	}
	return cloneGoalRecord(record), nil
}

func (repository *memoryRepository) ListGoals(_ context.Context, project goal.ProjectRef, limit int) ([]GoalSummary, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	result := make([]GoalSummary, 0, len(repository.records))
	for _, record := range repository.records {
		if record.Goal.Project() != project {
			continue
		}
		closedAt, _ := record.Goal.ClosedAt()
		appSpec := record.Goal.AppSpec()
		intent := appSpec.Intent()
		result = append(result, GoalSummary{
			Ref: record.Goal.Ref(), IntentRef: intent.Ref(), AppSpecRef: appSpec.Ref(),
			AppSpecGeneration: appSpec.Generation(), SpecHash: appSpec.Hash(),
			ActorRef: record.Goal.Actor(), ProjectRef: record.Goal.Project(),
			Statement: intent.Statement(), State: record.Goal.State(),
			Revision: record.Goal.Revision(), CreatedAt: record.Goal.CreatedAt(),
			ClosedAt: closedAt, ArtifactCount: len(record.Artifacts),
		})
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].CreatedAt.After(result[right].CreatedAt)
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (repository *memoryRepository) Status(_ context.Context, project goal.ProjectRef) (RepositoryStatus, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	status := RepositoryStatus{}
	for _, record := range repository.records {
		if record.Goal.Project() != project {
			continue
		}
		status.Goals++
		if record.Goal.State() == goal.GoalStateRunning {
			status.RunningGoals++
		}
	}
	for _, action := range repository.actions {
		if record, ok := repository.records[action.record.GoalRef]; ok && record.Goal.Project() == project {
			status.PendingActions++
		}
	}
	for _, event := range repository.events {
		if record, ok := repository.records[event.GoalRef]; ok && record.Goal.Project() == project && event.Kind == "action.quarantined" {
			status.QuarantinedActions++
		}
	}
	return status, nil
}

func (repository *memoryRepository) MailboxReplay(
	_ context.Context,
	request MailboxReplayRequest,
) (MailboxReplayRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	return repository.mailboxReplayLocked(request)
}

func (repository *memoryRepository) AdmitMailbox(
	_ context.Context,
	state AdmitMailboxState,
) (MailboxRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	request := MailboxReplayRequest{
		Kind: MailboxMutationAdmit, RequestRef: state.RequestRef,
		RequestFingerprint: state.RequestFingerprint,
		PrincipalRef:       state.Admission.PrincipalRef, ProjectRef: state.Envelope.ProjectRef,
		GoalRef: state.Envelope.GoalRef,
	}
	if replay, found, err := repository.mailboxReplayLocked(request); err != nil || found {
		return cloneMailboxRecord(replay.Record), false, err
	}
	if !memoryMailboxAuthorizationValid(
		state.AuthorizationReceipt, state.Admission.PrincipalRef, state.Envelope.ProjectRef,
		identity.PermissionGoalsDirect, state.Envelope.GoalRef.String(),
	) {
		return MailboxRecord{}, false, &StateError{Code: StateInvalid}
	}
	record, exists := repository.records[state.Envelope.GoalRef]
	if !exists || record.Goal.Project() != state.Envelope.ProjectRef ||
		record.Goal.State() != goal.GoalStateRunning ||
		record.Goal.PlanGeneration() != state.Envelope.TargetPlanGeneration {
		return MailboxRecord{}, false, &StateError{Code: StateConflict}
	}
	admissionRequest := AdmitMailboxRequest{
		RequestRef: state.RequestRef, GoalRef: state.Envelope.GoalRef,
		ExpectedPlanGeneration: state.Envelope.TargetPlanGeneration, Kind: state.Envelope.Kind,
		ParentWorkItemRef:     state.Envelope.ParentWorkItemRef,
		ChildWorkItemRef:      state.Envelope.ChildWorkItemRef,
		SourceExecutionRef:    state.Envelope.Source.ExecutionRef,
		RecipientPrincipalRef: state.Envelope.Recipient.PrincipalRef,
		RecipientExecutionRef: state.Envelope.Recipient.ExecutionRef,
		Summary:               state.Envelope.Summary,
		ArtifactRefs:          append([]goal.ArtifactRef(nil), state.Envelope.ArtifactRefs...),
	}
	parent, child, err := mailboxWorkItems(record.Goal, admissionRequest)
	if err != nil || validateMailboxArtifacts(child, admissionRequest.ArtifactRefs) != nil {
		return MailboxRecord{}, false, &StateError{Code: StateConflict}
	}
	candidate := MailboxRecord{
		Envelope: state.Envelope, Admission: state.Admission, Action: state.Action,
		State: MailboxStateAdmitted,
	}
	if _, duplicate := repository.mailboxes[state.Envelope.Ref]; duplicate ||
		state.Envelope.Source.PrincipalRef != state.Admission.PrincipalRef ||
		state.Envelope.RequestRef != state.RequestRef ||
		state.Envelope.RequestFingerprint != state.RequestFingerprint ||
		state.Admission.MessageRef != state.Envelope.Ref ||
		state.Admission.RequestRef != state.RequestRef ||
		state.Admission.RequestFingerprint != state.RequestFingerprint ||
		state.Admission.AuthorizationReceipt != state.AuthorizationReceipt ||
		state.OperationAt.IsZero() || !state.OperationAt.Equal(state.Envelope.AdmittedAt) ||
		!state.OperationAt.Equal(state.Admission.AdmittedAt) ||
		state.Action.WorkItemGeneration != parent.Revision() ||
		state.Event.Kind != "mailbox.admitted" || state.Event.GoalRef != state.Envelope.GoalRef ||
		state.Event.WorkItemRef != state.Envelope.ParentWorkItemRef ||
		state.Event.ExecutionRef != state.Envelope.Recipient.ExecutionRef ||
		!state.Event.OccurredAt.Equal(state.OperationAt) ||
		validateAdmittedMailboxRecord(
			candidate, admissionRequest, state.Admission.PrincipalRef,
			state.Envelope.ProjectRef, state.RequestFingerprint, false,
		) != nil {
		return MailboxRecord{}, false, &StateError{Code: StateInvalid}
	}
	if _, duplicate := repository.actions[state.Action.Ref]; duplicate {
		return MailboxRecord{}, false, &StateError{Code: StateConflict}
	}
	for _, existing := range repository.mailboxes {
		if existing.Envelope.GoalRef == state.Envelope.GoalRef &&
			existing.Envelope.ParentWorkItemRef == state.Envelope.ParentWorkItemRef &&
			existing.Envelope.ChildWorkItemRef == state.Envelope.ChildWorkItemRef {
			return MailboxRecord{}, false, &StateError{Code: StateConflict}
		}
	}
	candidate = cloneMailboxRecord(candidate)
	repository.mailboxes[state.Envelope.Ref] = candidate
	repository.actions[state.Action.Ref] = memoryAction{record: state.Action}
	repository.events = append(repository.events, state.Event)
	repository.mailboxAdmits++
	repository.mailboxRequests[memoryMailboxRequestKey(request)] = memoryMailboxMutation{
		request: request, replay: MailboxReplayRecord{Record: cloneMailboxRecord(candidate)},
	}
	return cloneMailboxRecord(candidate), true, nil
}

func (repository *memoryRepository) ClaimMailbox(
	_ context.Context,
	state ClaimMailboxState,
) (MailboxClaim, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	request := MailboxReplayRequest{
		Kind: MailboxMutationClaim, RequestRef: state.RequestRef,
		RequestFingerprint: state.RequestFingerprint, PrincipalRef: state.PrincipalRef,
		ProjectRef: state.ProjectRef, GoalRef: state.GoalRef, MessageRef: state.MessageRef,
	}
	if replay, found, err := repository.mailboxReplayLocked(request); err != nil || found {
		return cloneMailboxClaim(replay.Claim), false, err
	}
	if !memoryMailboxAuthorizationValid(
		state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef,
		identity.PermissionGoalsGet, state.MessageRef.String(),
	) || state.Token == "" || state.LeaseDuration <= 0 || state.RequestedAt.IsZero() {
		return MailboxClaim{}, false, &StateError{Code: StateInvalid}
	}
	record, exists := repository.mailboxes[state.MessageRef]
	if !exists || record.Envelope.ProjectRef != state.ProjectRef ||
		record.Envelope.GoalRef != state.GoalRef ||
		record.Envelope.Recipient.PrincipalRef != state.PrincipalRef ||
		record.Envelope.Recipient.ExecutionRef != state.RecipientExecutionRef ||
		record.Acknowledgement != nil {
		return MailboxClaim{}, false, &StateError{Code: StateConflict}
	}
	if err := repository.mailboxCurrentRecipientLocked(record); err != nil {
		return MailboxClaim{}, false, err
	}
	if record.State != MailboxStateAdmitted && record.State != MailboxStateClaimed &&
		record.State != MailboxStateDelivered {
		return MailboxClaim{}, false, &StateError{Code: StateConflict}
	}
	now := repository.now().UTC()
	if len(record.Attempts) > 0 {
		latest := record.Attempts[len(record.Attempts)-1]
		if now.Before(latest.LeaseUntil) {
			return MailboxClaim{}, false, &StateError{Code: StateAlreadyClaimed}
		}
	}
	next := uint64(len(record.Attempts) + 1)
	attempt := MailboxDeliveryAttempt{
		MessageRef: state.MessageRef, ActionRef: record.Action.Ref,
		Recipient: record.Envelope.Recipient, ClaimRequestRef: state.RequestRef,
		ClaimToken: state.Token, Fence: next,
		ClaimedAt: now, LeaseUntil: now.Add(state.LeaseDuration),
	}
	if state.RequestedAt.After(now) || !attempt.LeaseUntil.After(attempt.ClaimedAt) {
		return MailboxClaim{}, false, &StateError{Code: StateInvalid}
	}
	record.Attempts = append(record.Attempts, attempt)
	record.State = MailboxStateClaimed
	repository.mailboxes[state.MessageRef] = cloneMailboxRecord(record)
	claim := MailboxClaim{Record: cloneMailboxRecord(record), Attempt: attempt}
	repository.mailboxClaims++
	repository.mailboxRequests[memoryMailboxRequestKey(request)] = memoryMailboxMutation{
		request: request, replay: MailboxReplayRecord{Record: cloneMailboxRecord(record), Claim: cloneMailboxClaim(claim)},
	}
	return cloneMailboxClaim(claim), true, nil
}

func (repository *memoryRepository) MarkMailboxDelivered(
	_ context.Context,
	state MarkMailboxDeliveredState,
) (MailboxRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	request := mailboxReplayRequestForDelivery(MailboxMutationDeliver, state.RequestRef, state.RequestFingerprint,
		state.PrincipalRef, state.ProjectRef, state.GoalRef, state.MessageRef)
	if replay, found, err := repository.mailboxReplayLocked(request); err != nil || found {
		return cloneMailboxRecord(replay.Record), false, err
	}
	record, attempt, err := repository.mailboxAttemptLocked(
		state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef, state.GoalRef,
		state.MessageRef, state.RecipientExecutionRef, state.ClaimToken,
		state.Fence, true,
	)
	if err != nil || record.State != MailboxStateClaimed || state.DeliveryRef == "" || state.OperationAt.IsZero() {
		if err != nil {
			return MailboxRecord{}, false, err
		}
		return MailboxRecord{}, false, &StateError{Code: StateConflict}
	}
	attempt.DeliveryRef = state.DeliveryRef
	attempt.DeliveredAt = state.OperationAt.UTC()
	record.Attempts[len(record.Attempts)-1] = attempt
	record.State = MailboxStateDelivered
	repository.mailboxes[state.MessageRef] = cloneMailboxRecord(record)
	repository.mailboxDeliveries++
	repository.mailboxRequests[memoryMailboxRequestKey(request)] = memoryMailboxMutation{
		request: request, replay: MailboxReplayRecord{Record: cloneMailboxRecord(record)},
	}
	return cloneMailboxRecord(record), true, nil
}

func (repository *memoryRepository) ConsumeMailbox(
	_ context.Context,
	state ConsumeMailboxState,
) (MailboxRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	request := mailboxReplayRequestForDelivery(MailboxMutationConsume, state.RequestRef, state.RequestFingerprint,
		state.PrincipalRef, state.ProjectRef, state.GoalRef, state.MessageRef)
	if replay, found, err := repository.mailboxReplayLocked(request); err != nil || found {
		return cloneMailboxRecord(replay.Record), false, err
	}
	record, attempt, err := repository.mailboxAttemptLocked(
		state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef, state.GoalRef,
		state.MessageRef, state.RecipientExecutionRef, state.ClaimToken,
		state.Fence, true,
	)
	if err != nil || record.State != MailboxStateDelivered || state.ConsumptionRef == "" ||
		state.OperationAt.IsZero() || attempt.DeliveryRef == "" || attempt.DeliveredAt.IsZero() {
		if err != nil {
			return MailboxRecord{}, false, err
		}
		return MailboxRecord{}, false, &StateError{Code: StateConflict}
	}
	receipt := state.ConsumptionReceipt
	if receipt.ActionRef != record.Action.Ref || receipt.Kind != ActionDeliverMailbox ||
		receipt.GoalRef != state.GoalRef || receipt.WorkItemRef != record.Action.WorkItemRef ||
		receipt.ExecutionRef != record.Action.ExecutionRef || receipt.MailboxMessageRef != state.MessageRef ||
		receipt.PlanGeneration != record.Action.PlanGeneration ||
		receipt.WorkItemGeneration != record.Action.WorkItemGeneration ||
		receipt.Fence != state.Fence || receipt.DeliveryAttempt != state.Fence ||
		receipt.ClaimToken != state.ClaimToken || receipt.WorkerRef != state.PrincipalRef.String() ||
		receipt.Outcome != ActionConsumedCompleted || receipt.ErrorCode != "" ||
		!receipt.ConsumedAt.Equal(state.OperationAt) {
		return MailboxRecord{}, false, &StateError{Code: StateInvalid}
	}
	goalRecord, exists := repository.records[state.GoalRef]
	if !exists {
		return MailboxRecord{}, false, &StateError{Code: StateConflict}
	}
	attempt.ConsumptionRef = state.ConsumptionRef
	attempt.ConsumedAt = state.OperationAt.UTC()
	record.Attempts[len(record.Attempts)-1] = attempt
	record.State = MailboxStateConsumed
	repository.mailboxes[state.MessageRef] = cloneMailboxRecord(record)
	goalRecord.ConsumptionReceipts = append(goalRecord.ConsumptionReceipts, receipt)
	repository.records[state.GoalRef] = goalRecord
	delete(repository.actions, record.Action.Ref)
	repository.mailboxConsumptions++
	repository.mailboxRequests[memoryMailboxRequestKey(request)] = memoryMailboxMutation{
		request: request, replay: MailboxReplayRecord{Record: cloneMailboxRecord(record)},
	}
	return cloneMailboxRecord(record), true, nil
}

func (repository *memoryRepository) AcknowledgeMailbox(
	ctx context.Context,
	state ResolveMailboxState,
) (MailboxAcknowledgement, bool, error) {
	return repository.resolveMailbox(ctx, state, MailboxOutcomeAcknowledged)
}

func (repository *memoryRepository) BlockMailbox(
	ctx context.Context,
	state ResolveMailboxState,
) (MailboxAcknowledgement, bool, error) {
	return repository.resolveMailbox(ctx, state, MailboxOutcomeBlocked)
}

func (repository *memoryRepository) GetMailbox(
	_ context.Context,
	projectRef goal.ProjectRef,
	goalRef goal.GoalRef,
	messageRef MailboxMessageRef,
	recipient MailboxEndpoint,
) (MailboxRecord, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	record, exists := repository.mailboxes[messageRef]
	if !exists || record.Envelope.ProjectRef != projectRef || record.Envelope.GoalRef != goalRef ||
		record.Envelope.Recipient != recipient {
		return MailboxRecord{}, &StateError{Code: StateNotFound}
	}
	return cloneMailboxRecord(record), nil
}

func (repository *memoryRepository) ListMailbox(
	_ context.Context,
	projectRef goal.ProjectRef,
	goalRef goal.GoalRef,
	recipient MailboxEndpoint,
	limit int,
) ([]MailboxRecord, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if limit <= 0 {
		return nil, &StateError{Code: StateInvalid}
	}
	result := make([]MailboxRecord, 0)
	for _, record := range repository.mailboxes {
		if record.Envelope.ProjectRef == projectRef && record.Envelope.GoalRef == goalRef &&
			record.Envelope.Recipient == recipient {
			result = append(result, cloneMailboxRecord(record))
		}
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].Envelope.AdmittedAt.Equal(result[right].Envelope.AdmittedAt) {
			return result[left].Envelope.Ref.String() < result[right].Envelope.Ref.String()
		}
		return result[left].Envelope.AdmittedAt.Before(result[right].Envelope.AdmittedAt)
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (repository *memoryRepository) resolveMailbox(
	_ context.Context,
	state ResolveMailboxState,
	outcome MailboxOutcome,
) (MailboxAcknowledgement, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	kind := MailboxMutationAcknowledge
	goalOutcome := goal.ChildHandoffAcknowledged
	if outcome == MailboxOutcomeBlocked {
		kind = MailboxMutationBlock
		goalOutcome = goal.ChildHandoffBlocked
	}
	request := mailboxReplayRequestForDelivery(kind, state.RequestRef, state.RequestFingerprint,
		state.PrincipalRef, state.ProjectRef, state.GoalRef, state.MessageRef)
	if replay, found, err := repository.mailboxReplayLocked(request); err != nil || found {
		return replay.Acknowledgement, false, err
	}
	record, attempt, err := repository.mailboxAttemptLocked(
		state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef, state.GoalRef,
		state.MessageRef, state.RecipientExecutionRef, state.ClaimToken,
		state.Fence, false,
	)
	if err != nil || record.State != MailboxStateConsumed || record.Acknowledgement != nil ||
		state.OperationAt.IsZero() || state.ExpectedGoalRevision == 0 ||
		state.ExpectedPlanGeneration < record.Envelope.TargetPlanGeneration {
		if err != nil {
			return MailboxAcknowledgement{}, false, err
		}
		return MailboxAcknowledgement{}, false, &StateError{Code: StateConflict}
	}
	current, exists := repository.records[state.GoalRef]
	if !exists || current.Goal.Revision() != state.ExpectedGoalRevision ||
		current.Goal.PlanGeneration() != state.ExpectedPlanGeneration {
		return MailboxAcknowledgement{}, false, &StateError{Code: StateConflict}
	}
	ack := state.Acknowledgement
	if ack.MessageRef != state.MessageRef || ack.ActionRef != record.Action.Ref ||
		ack.RequestRef != state.RequestRef || ack.RequestFingerprint != state.RequestFingerprint ||
		ack.ProjectRef != state.ProjectRef || ack.GoalRef != state.GoalRef ||
		ack.TargetPlanGeneration != record.Envelope.TargetPlanGeneration ||
		ack.ParentWorkItemRef != record.Envelope.ParentWorkItemRef ||
		ack.ChildWorkItemRef != record.Envelope.ChildWorkItemRef ||
		ack.Recipient != record.Envelope.Recipient || ack.Fence != state.Fence ||
		ack.Outcome != outcome || ack.Ref == "" ||
		ack.EffectOrReworkRef == "" || !ack.AcknowledgedAt.Equal(state.OperationAt) ||
		ack.AuthorizationReceipt != state.AuthorizationReceipt ||
		attempt.ConsumptionRef == "" || attempt.ConsumedAt.IsZero() {
		return MailboxAcknowledgement{}, false, &StateError{Code: StateInvalid}
	}
	want, err := current.Goal.ResolveChildHandoff(
		state.ExpectedGoalRevision, record.Envelope.ParentWorkItemRef,
		record.Envelope.ChildWorkItemRef, state.MessageRef.String(), goalOutcome,
		ack.Ref, state.OperationAt,
	)
	if err == nil {
		if closeOutcome, closable := want.ClosableOutcome(); closable {
			want, err = want.Close(want.Revision(), closeOutcome, state.OperationAt)
		}
	}
	if err != nil || !reflect.DeepEqual(want.Snapshot(), state.Goal.Snapshot()) {
		return MailboxAcknowledgement{}, false, &StateError{Code: StateConflict}
	}
	if len(state.Events) == 0 || state.Events[0].Kind != "mailbox."+string(outcome) ||
		state.Events[0].GoalRef != state.GoalRef ||
		state.Events[0].WorkItemRef != record.Envelope.ParentWorkItemRef ||
		state.Events[0].ExecutionRef != record.Envelope.Recipient.ExecutionRef ||
		!state.Events[0].OccurredAt.Equal(state.OperationAt) {
		return MailboxAcknowledgement{}, false, &StateError{Code: StateInvalid}
	}
	if state.Goal.IsTerminal() {
		if len(state.Events) != 2 || state.Events[1].Kind != "goal."+string(state.Goal.State()) ||
			state.Events[1].GoalRef != state.GoalRef || !state.Events[1].OccurredAt.Equal(state.OperationAt) {
			return MailboxAcknowledgement{}, false, &StateError{Code: StateInvalid}
		}
	} else if len(state.Events) != 1 {
		return MailboxAcknowledgement{}, false, &StateError{Code: StateInvalid}
	}
	record.Acknowledgement = &ack
	if outcome == MailboxOutcomeBlocked {
		record.State = MailboxStateBlocked
	} else {
		record.State = MailboxStateAcknowledged
	}
	current.Goal = state.Goal
	repository.records[state.GoalRef] = current
	repository.mailboxes[state.MessageRef] = cloneMailboxRecord(record)
	repository.events = append(repository.events, state.Events...)
	repository.mailboxResolutions++
	repository.mailboxRequests[memoryMailboxRequestKey(request)] = memoryMailboxMutation{
		request: request,
		replay:  MailboxReplayRecord{Record: cloneMailboxRecord(record), Acknowledgement: ack},
	}
	return ack, true, nil
}

func (repository *memoryRepository) mailboxReplayLocked(
	request MailboxReplayRequest,
) (MailboxReplayRecord, bool, error) {
	mutation, exists := repository.mailboxRequests[memoryMailboxRequestKey(request)]
	if !exists {
		return MailboxReplayRecord{}, false, nil
	}
	if mutation.request != request {
		return MailboxReplayRecord{}, false, &StateError{Code: StateConflict}
	}
	if request.Kind == MailboxMutationClaim || request.Kind == MailboxMutationDeliver ||
		request.Kind == MailboxMutationConsume {
		current, ok := repository.mailboxes[request.MessageRef]
		if !ok || len(current.Attempts) == 0 {
			return MailboxReplayRecord{}, false, &StateError{Code: StateConflict}
		}
		stored := mutation.replay.Record
		if len(stored.Attempts) == 0 {
			return MailboxReplayRecord{}, false, &StateError{Code: StateConflict}
		}
		storedAttempt := stored.Attempts[len(stored.Attempts)-1]
		attemptIndex := -1
		for index := range current.Attempts {
			if current.Attempts[index].Fence == storedAttempt.Fence {
				attemptIndex = index
				break
			}
		}
		if attemptIndex < 0 || !sameMailboxClaimFact(current.Attempts[attemptIndex], storedAttempt) {
			return MailboxReplayRecord{}, false, &StateError{Code: StateConflict}
		}
		currentAttempt := current.Attempts[attemptIndex]
		switch request.Kind {
		case MailboxMutationClaim:
			if attemptIndex != len(current.Attempts)-1 || current.Acknowledgement != nil || current.Retirement != nil ||
				current.State == MailboxStateAcknowledged || current.State == MailboxStateBlocked ||
				current.State == MailboxStateRetired {
				return MailboxReplayRecord{}, false, &StateError{Code: StateConflict}
			}
		case MailboxMutationDeliver:
			if storedAttempt.DeliveryRef == "" || storedAttempt.DeliveredAt.IsZero() ||
				currentAttempt.DeliveryRef != storedAttempt.DeliveryRef ||
				!currentAttempt.DeliveredAt.Equal(storedAttempt.DeliveredAt) {
				return MailboxReplayRecord{}, false, &StateError{Code: StateConflict}
			}
		case MailboxMutationConsume:
			if storedAttempt.DeliveryRef == "" || storedAttempt.DeliveredAt.IsZero() ||
				currentAttempt.DeliveryRef != storedAttempt.DeliveryRef ||
				!currentAttempt.DeliveredAt.Equal(storedAttempt.DeliveredAt) ||
				storedAttempt.ConsumptionRef == "" || storedAttempt.ConsumedAt.IsZero() ||
				currentAttempt.ConsumptionRef != storedAttempt.ConsumptionRef ||
				!currentAttempt.ConsumedAt.Equal(storedAttempt.ConsumedAt) {
				return MailboxReplayRecord{}, false, &StateError{Code: StateConflict}
			}
		}
	}
	replay := mutation.replay
	if request.Kind == MailboxMutationAdmit {
		if current, ok := repository.mailboxes[mutation.replay.Record.Envelope.Ref]; ok {
			replay.Record = current
		}
	}
	replay.Record = cloneMailboxRecord(replay.Record)
	replay.Claim = cloneMailboxClaim(replay.Claim)
	return replay, true, nil
}

func sameMailboxClaimFact(left, right MailboxDeliveryAttempt) bool {
	return left.MessageRef == right.MessageRef && left.ActionRef == right.ActionRef &&
		left.Recipient == right.Recipient && left.ClaimRequestRef == right.ClaimRequestRef &&
		left.ClaimToken == right.ClaimToken && left.Fence == right.Fence &&
		left.ClaimedAt.Equal(right.ClaimedAt) && left.LeaseUntil.Equal(right.LeaseUntil)
}

func (repository *memoryRepository) mailboxAttemptLocked(
	authorization identity.AuthorizationReceipt,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	goalRef goal.GoalRef,
	messageRef MailboxMessageRef,
	executionRef goal.ExecutionRef,
	claimToken string,
	fence uint64,
	requireLive bool,
) (MailboxRecord, MailboxDeliveryAttempt, error) {
	if !memoryMailboxAuthorizationValid(
		authorization, principalRef, projectRef, identity.PermissionGoalsGet, messageRef.String(),
	) {
		return MailboxRecord{}, MailboxDeliveryAttempt{}, &StateError{Code: StateInvalid}
	}
	record, exists := repository.mailboxes[messageRef]
	if !exists || record.Envelope.ProjectRef != projectRef || record.Envelope.GoalRef != goalRef ||
		record.Envelope.Recipient.PrincipalRef != principalRef ||
		record.Envelope.Recipient.ExecutionRef != executionRef || len(record.Attempts) == 0 {
		return MailboxRecord{}, MailboxDeliveryAttempt{}, &StateError{Code: StateConflict}
	}
	if err := repository.mailboxCurrentRecipientLocked(record); err != nil {
		return MailboxRecord{}, MailboxDeliveryAttempt{}, err
	}
	attempt := record.Attempts[len(record.Attempts)-1]
	if attempt.ClaimToken != claimToken || attempt.Fence != fence ||
		(requireLive && !repository.now().UTC().Before(attempt.LeaseUntil)) {
		return MailboxRecord{}, MailboxDeliveryAttempt{}, &StateError{Code: StateConflict}
	}
	return cloneMailboxRecord(record), attempt, nil
}

func (repository *memoryRepository) mailboxCurrentRecipientLocked(record MailboxRecord) error {
	current, exists := repository.records[record.Envelope.GoalRef]
	if !exists || current.Goal.Project() != record.Envelope.ProjectRef ||
		current.Goal.State() != goal.GoalStateRunning ||
		current.Goal.PlanGeneration() < record.Envelope.TargetPlanGeneration {
		return &StateError{Code: StateConflict}
	}
	parent, exists := current.Goal.WorkItem(record.Envelope.ParentWorkItemRef)
	executionRef, hasExecution := parent.Execution()
	if !exists || !hasExecution || executionRef != record.Envelope.Recipient.ExecutionRef ||
		parent.State() != goal.WorkItemStateRunning {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func mailboxReplayRequestForDelivery(
	kind MailboxMutationKind,
	requestRef string,
	fingerprint string,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	goalRef goal.GoalRef,
	messageRef MailboxMessageRef,
) MailboxReplayRequest {
	return MailboxReplayRequest{
		Kind: kind, RequestRef: requestRef, RequestFingerprint: fingerprint,
		PrincipalRef: principalRef, ProjectRef: projectRef, GoalRef: goalRef,
		MessageRef: messageRef,
	}
}

func memoryMailboxRequestKey(request MailboxReplayRequest) string {
	return request.PrincipalRef.String() + "\x00" + request.ProjectRef.String() + "\x00" +
		string(request.Kind) + "\x00" + request.RequestRef
}

func memoryMailboxAuthorizationValid(
	receipt identity.AuthorizationReceipt,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	permission identity.Permission,
	resourceRef string,
) bool {
	request := receipt.Decision().Request()
	return receipt.Decision().Outcome() == identity.AuthorizationAllowed &&
		identity.RoleAllows(receipt.Decision().Role(), permission) &&
		request.RequestRef() != "" && request.Principal().Ref == principalRef &&
		request.ProjectRef() == projectRef && request.Permission() == permission &&
		request.ResourceRef() == resourceRef
}

func (repository *memoryRepository) ClaimNextAction(_ context.Context, request ClaimRequest) (ActionClaim, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	now := repository.now().UTC()
	refs := make([]string, 0, len(repository.actions))
	for ref := range repository.actions {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	for _, ref := range refs {
		action := repository.actions[ref]
		if action.record.Kind == ActionDeliverMailbox || action.record.AvailableAt.After(now) ||
			(action.token != "" && action.lease.After(now)) {
			continue
		}
		record := repository.records[action.record.GoalRef]
		item, found := record.Goal.WorkItem(action.record.WorkItemRef)
		if !found || !ports.MatchAgentCapabilities(request.Capabilities, ports.AgentRequirements{
			RoleKey: item.Role().String(), SkillRefs: workItemRefs(item.SkillRefs()),
			ToolRefs: workItemRefs(item.ToolRefs()), CapabilityRefs: workItemRefs(item.CapabilityRefs()),
		}) {
			continue
		}
		action.token = request.Token
		action.workerRef = request.WorkerRef
		action.deliveryAttempt++
		action.fence++
		action.lease = now.Add(request.LeaseDuration)
		repository.actions[ref] = action
		return ActionClaim{
			Action: action.record, Token: action.token, WorkerRef: action.workerRef,
			DeliveryAttempt: action.deliveryAttempt, Fence: action.fence, LeaseUntil: action.lease,
		}, true, nil
	}
	return ActionClaim{}, false, nil
}

func (repository *memoryRepository) RecordLaunchPrepared(_ context.Context, state LaunchPreparedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	action, ok := repository.actions[state.Claim.Action.Ref]
	if !ok || !memoryClaimMatches(action, state.Claim, state.OperationAt) {
		return &StateError{Code: StateConflict}
	}
	record := repository.records[state.Goal.Ref()]
	if record.Goal.Revision() != state.ExpectedGoalRevision {
		return &StateError{Code: StateConflict}
	}
	record.Goal = state.Goal
	record.Executions = replaceExecution(record.Executions, state.Execution)
	repository.records[state.Goal.Ref()] = record
	repository.events = append(repository.events, state.Event)
	return nil
}

func (repository *memoryRepository) RecordLaunchAccepted(_ context.Context, state LaunchAcceptedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	action, ok := repository.actions[state.Claim.Action.Ref]
	if !ok || !memoryClaimMatches(action, state.Claim, state.OperationAt) {
		return &StateError{Code: StateConflict}
	}
	record := repository.records[state.Claim.Action.GoalRef]
	record.Executions = replaceExecution(record.Executions, state.Execution)
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, consumptionReceipt(state.Claim, ActionConsumedCompleted, "", state.OperationAt))
	repository.records[record.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	repository.actions[state.NextAction.Ref] = memoryAction{record: state.NextAction}
	repository.events = append(repository.events, state.Event)
	return nil
}

func (repository *memoryRepository) RequeueAction(_ context.Context, state ActionRequeuedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	action, ok := repository.actions[state.Claim.Action.Ref]
	if !ok || !memoryClaimMatches(action, state.Claim, state.OperationAt) {
		return &StateError{Code: StateConflict}
	}
	record, ok := repository.records[state.Claim.Action.GoalRef]
	current, found := executionForAction(record, state.Claim.Action)
	if !ok || !found || current.Ref != state.Execution.Ref {
		return &StateError{Code: StateConflict}
	}
	record.Executions = replaceExecution(record.Executions, state.Execution)
	repository.records[record.Goal.Ref()] = record
	action.record.AvailableAt = state.AvailableAt
	action.token = ""
	action.workerRef = ""
	action.lease = time.Time{}
	repository.actions[action.record.Ref] = action
	return nil
}

func (repository *memoryRepository) QuarantineAction(_ context.Context, state ActionQuarantinedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	action, ok := repository.actions[state.Claim.Action.Ref]
	if !ok || !memoryClaimMatches(action, state.Claim, state.OperationAt) {
		return &StateError{Code: StateConflict}
	}
	record := repository.records[state.Claim.Action.GoalRef]
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, consumptionReceipt(state.Claim, ActionConsumedQuarantined, state.ErrorCode, state.OperationAt))
	repository.records[record.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	repository.events = append(repository.events, state.Event)
	return nil
}

func (repository *memoryRepository) RecordExecutionReplaced(_ context.Context, state ExecutionReplacedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if err := repository.validateMutation(state.Claim, state.OperationAt, state.ExpectedGoalRevision, state.ExpectedItemRevision); err != nil {
		return err
	}
	if repository.hasUnresolvedMailboxRecipientLocked(state.FailedExecution.Ref) {
		return &StateError{Code: StateRecipientMailboxActive}
	}
	record := repository.records[state.Goal.Ref()]
	record.Goal = state.Goal
	record.Executions = replaceExecution(record.Executions, state.FailedExecution)
	record.Executions = append(record.Executions, state.ReplacementExecution)
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, consumptionReceipt(state.Claim, ActionConsumedCompleted, state.ErrorCode, state.OperationAt))
	repository.records[state.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	repository.actions[state.NextAction.Ref] = memoryAction{record: state.NextAction}
	repository.events = append(repository.events, state.Events...)
	return nil
}

func (repository *memoryRepository) RecordGoalSucceeded(_ context.Context, state GoalSucceededState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if err := repository.validateMutation(state.Claim, state.OperationAt, state.ExpectedGoalRevision, state.ExpectedItemRevision); err != nil {
		return err
	}
	record := repository.records[state.Goal.Ref()]
	record.Goal = state.Goal
	record.Executions = replaceExecution(record.Executions, state.Execution)
	record.Executions = append(record.Executions, state.NewExecutions...)
	record.Artifacts = append(record.Artifacts, state.Artifact)
	record.Attestations = append(record.Attestations, state.Attestation)
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, consumptionReceipt(state.Claim, ActionConsumedCompleted, "", state.OperationAt))
	repository.records[state.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	for _, action := range state.NewActions {
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	repository.events = append(repository.events, state.Events...)
	return nil
}

func (repository *memoryRepository) RecordGoalFailed(_ context.Context, state GoalFailedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if err := repository.validateMutation(state.Claim, state.OperationAt, state.ExpectedGoalRevision, state.ExpectedItemRevision); err != nil {
		return err
	}
	retirements := make(map[MailboxMessageRef]MailboxRetirement)
	for messageRef, mailbox := range repository.mailboxes {
		if mailbox.Envelope.Recipient.ExecutionRef != state.Execution.Ref ||
			mailbox.Acknowledgement != nil || mailbox.Retirement != nil {
			continue
		}
		retirement, err := BuildMailboxRetirement(mailbox, state.Execution)
		if err != nil {
			return &StateError{Code: StateInvalid, Cause: err}
		}
		retirements[messageRef] = retirement
	}
	record := repository.records[state.Goal.Ref()]
	record.Goal = state.Goal
	record.Executions = replaceExecution(record.Executions, state.Execution)
	record.Executions = append(record.Executions, state.NewExecutions...)
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, consumptionReceipt(state.Claim, ActionConsumedCompleted, state.Execution.FailureCode, state.OperationAt))
	repository.records[state.Goal.Ref()] = record
	for messageRef, retirement := range retirements {
		mailbox := repository.mailboxes[messageRef]
		mailbox.Retirement = &retirement
		mailbox.State = MailboxStateRetired
		repository.mailboxes[messageRef] = cloneMailboxRecord(mailbox)
		delete(repository.actions, mailbox.Action.Ref)
	}
	delete(repository.actions, state.Claim.Action.Ref)
	for _, action := range state.NewActions {
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	repository.events = append(repository.events, state.Events...)
	return nil
}

func (repository *memoryRepository) hasUnresolvedMailboxRecipientLocked(executionRef goal.ExecutionRef) bool {
	for _, mailbox := range repository.mailboxes {
		if mailbox.Envelope.Recipient.ExecutionRef == executionRef &&
			mailbox.Acknowledgement == nil && mailbox.Retirement == nil {
			return true
		}
	}
	return false
}

func (repository *memoryRepository) validateMutation(claim ActionClaim, operationAt time.Time, goalRevision, itemRevision goal.Revision) error {
	action, ok := repository.actions[claim.Action.Ref]
	if !ok || !memoryClaimMatches(action, claim, operationAt) {
		return &StateError{Code: StateConflict}
	}
	record, ok := repository.records[claim.Action.GoalRef]
	if !ok || record.Goal.Revision() != goalRevision {
		return &StateError{Code: StateConflict}
	}
	item, ok := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !ok || item.Revision() != itemRevision {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func cloneGoalRecord(record GoalRecord) GoalRecord {
	record.Executions = append([]ExecutionRecord(nil), record.Executions...)
	record.Artifacts = append([]ArtifactRecord(nil), record.Artifacts...)
	record.Attestations = append([]AttestationRecord(nil), record.Attestations...)
	record.ConsumptionReceipts = append([]ActionConsumptionReceipt(nil), record.ConsumptionReceipts...)
	return record
}

func memoryWriteAuthorizationValid(
	receipt identity.AuthorizationReceipt,
	requestedBy identity.PrincipalRef,
	projectRef goal.ProjectRef,
	permission identity.Permission,
	resourceRef string,
) bool {
	request := receipt.Decision().Request()
	return receipt.Decision().Outcome() == identity.AuthorizationAllowed &&
		identity.RoleAllows(receipt.Decision().Role(), permission) &&
		request.RequestRef() != "" && request.Principal().Ref == requestedBy &&
		request.ProjectRef() == projectRef && request.Permission() == permission &&
		request.ResourceRef() == resourceRef
}

func memoryClaimMatches(action memoryAction, claim ActionClaim, operationAt time.Time) bool {
	return action.token == claim.Token && action.workerRef == claim.WorkerRef &&
		action.deliveryAttempt == claim.DeliveryAttempt && action.fence == claim.Fence &&
		action.lease.Equal(claim.LeaseUntil) && operationAt.Before(action.lease)
}

func consumptionReceipt(claim ActionClaim, outcome ActionConsumptionOutcome, code string, at time.Time) ActionConsumptionReceipt {
	return ActionConsumptionReceipt{
		ActionRef: claim.Action.Ref, Kind: claim.Action.Kind,
		GoalRef: claim.Action.GoalRef, WorkItemRef: claim.Action.WorkItemRef,
		ExecutionRef: claim.Action.ExecutionRef, PlanGeneration: claim.Action.PlanGeneration,
		WorkItemGeneration: claim.Action.WorkItemGeneration, Fence: claim.Fence,
		DeliveryAttempt: claim.DeliveryAttempt, ClaimToken: claim.Token,
		WorkerRef: claim.WorkerRef, Outcome: outcome, ErrorCode: code, ConsumedAt: at.UTC(),
	}
}

func onlyExecution(t *testing.T, record GoalRecord) ExecutionRecord {
	t.Helper()
	if len(record.Executions) != 1 {
		t.Fatalf("execution count = %d, want 1", len(record.Executions))
	}
	return record.Executions[0]
}

type mutableClock struct {
	mu  sync.Mutex
	now time.Time
}

func (clock *mutableClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *mutableClock) Advance(duration time.Duration) {
	clock.mu.Lock()
	clock.now = clock.now.Add(duration)
	clock.mu.Unlock()
}

type sequentialIDs struct {
	mu   sync.Mutex
	next int
}

func (ids *sequentialIDs) NewID(ctx context.Context, prefix string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	ids.mu.Lock()
	defer ids.mu.Unlock()
	ids.next++
	return fmt.Sprintf("%s:%03d", prefix, ids.next), nil
}

type memoryArtifactStore struct {
	mu      sync.Mutex
	content map[goal.ArtifactRef]ports.ArtifactContent
}

func newMemoryArtifactStore() *memoryArtifactStore {
	return &memoryArtifactStore{content: make(map[goal.ArtifactRef]ports.ArtifactContent)}
}

func (store *memoryArtifactStore) Put(_ context.Context, request ports.PutArtifactRequest) (ports.StoredArtifact, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	digest := sha256.Sum256(request.Content)
	digestText := hex.EncodeToString(digest[:])
	ref, err := goal.NewArtifactRef("artifact:sha256:" + digestText)
	if err != nil {
		return ports.StoredArtifact{}, err
	}
	stored := ports.StoredArtifact{Ref: ref, Digest: digestText, MediaType: request.MediaType, Size: int64(len(request.Content))}
	store.content[ref] = ports.ArtifactContent{Ref: ref, Digest: stored.Digest, Size: stored.Size, Content: append([]byte(nil), request.Content...)}
	return stored, nil
}

func (store *memoryArtifactStore) Get(_ context.Context, ref goal.ArtifactRef, expectedSize int64) (ports.ArtifactContent, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	content, ok := store.content[ref]
	if !ok {
		return ports.ArtifactContent{}, errors.New("artifact.not_found")
	}
	if content.Size != expectedSize {
		return ports.ArtifactContent{}, errors.New("artifact.size_mismatch")
	}
	content.Content = append([]byte(nil), content.Content...)
	return content, nil
}

type scriptedAgent struct {
	mu                               sync.Mutex
	now                              func() time.Time
	launches                         int
	observationCalls                 int
	launchRequests                   []ports.AgentLaunchRequest
	launchSpecHashes                 map[goal.ExecutionRef]string
	receiptSpecHashOverride          string
	preserveEmptyReceiptSpecHash     bool
	preserveEmptyObservationSpecHash bool
	observations                     []ports.AgentObservation
	launchErr                        error
	launchEntered                    chan struct{}
	launchRelease                    <-chan struct{}
}

func (agent *scriptedAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return testAgentCapabilities(), nil
}

func testAgentCapabilities() ports.AgentCapabilities {
	return ports.AgentCapabilities{
		ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test", Unrestricted: true,
	}
}

func (agent *scriptedAgent) Launch(_ context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	agent.mu.Lock()
	agent.launches++
	agent.launchRequests = append(agent.launchRequests, request)
	if agent.launchSpecHashes == nil {
		agent.launchSpecHashes = make(map[goal.ExecutionRef]string)
	}
	agent.launchSpecHashes[request.ExecutionRef] = request.SpecHash
	entered := agent.launchEntered
	release := agent.launchRelease
	now := agent.now
	launchErr := agent.launchErr
	receiptSpecHash := agent.receiptSpecHashOverride
	if receiptSpecHash == "" && !agent.preserveEmptyReceiptSpecHash {
		receiptSpecHash = request.SpecHash
	}
	agent.mu.Unlock()
	if entered != nil {
		select {
		case entered <- struct{}{}:
		default:
		}
	}
	if release != nil {
		<-release
	}
	if launchErr != nil {
		return ports.AgentLaunchReceipt{}, launchErr
	}
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: receiptSpecHash,
		ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test",
		ExternalRef:    "external:" + request.ExecutionRef.String(),
		IdempotencyKey: request.IdempotencyKey, AcceptedAt: now(),
	}, nil
}

func (agent *scriptedAgent) Observe(_ context.Context, execution goal.ExecutionRef) (ports.AgentObservation, error) {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	agent.observationCalls++
	if len(agent.observations) == 0 {
		return ports.AgentObservation{}, errors.New("test.no_observation")
	}
	observation := agent.observations[0]
	agent.observations = agent.observations[1:]
	observation.ExecutionRef = execution
	if observation.SpecHash == "" && !agent.preserveEmptyObservationSpecHash {
		observation.SpecHash = agent.launchSpecHashes[execution]
	}
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = agent.now()
	}
	return observation, nil
}

func newTestOrchestrator(t interface{ Fatalf(string, ...any) }, repository *memoryRepository, clock *mutableClock, agent *scriptedAgent) (*Orchestrator, *memoryArtifactStore) {
	return newTestOrchestratorWithAccess(t, repository, newMemoryAccessRepository(), clock, agent)
}

func newTestOrchestratorWithAccess(
	t interface{ Fatalf(string, ...any) },
	repository *memoryRepository,
	accessRepository AccessRepository,
	clock *mutableClock,
	agent *scriptedAgent,
) (*Orchestrator, *memoryArtifactStore) {
	artifacts := newMemoryArtifactStore()
	repository.now = clock.Now
	capabilities, err := agent.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("agent capabilities: %v", err)
	}
	orchestrator, err := New(Dependencies{
		State: repository, Access: accessRepository,
		Launcher: agent, Observer: agent, Artifacts: artifacts,
		Clock: clock, IDs: &sequentialIDs{}, MaxOutputBytes: 1 << 20,
		MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts:    3, ClaimLease: time.Minute, DirectorLeaseDuration: time.Minute,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: capabilities,
	})
	if err != nil {
		t.Fatalf("new orchestrator: %v", err)
	}
	return orchestrator, artifacts
}

func testScope(t interface{ Fatalf(string, ...any) }) (goal.ActorRef, goal.ProjectRef) {
	actor, err := goal.NewActorRef("actor:test")
	if err != nil {
		t.Fatalf("actor ref: %v", err)
	}
	project, err := goal.NewProjectRef("project:test")
	if err != nil {
		t.Fatalf("project ref: %v", err)
	}
	return actor, project
}
