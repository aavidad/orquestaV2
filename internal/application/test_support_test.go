package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
	"orquesta/internal/review"
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
	controlRequests     map[string]ControlReplayRequest
	controlRefs         map[string]string
	mailboxes           map[MailboxMessageRef]MailboxRecord
	mailboxRequests     map[string]memoryMailboxMutation
	mailboxAdmits       int
	mailboxClaims       int
	mailboxDeliveries   int
	mailboxConsumptions int
	mailboxResolutions  int
	integrationAdmits   map[string]memoryIntegrationAdmission
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

func appTestNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

type memoryIntegrationAdmission struct {
	requestFingerprint string
	goalRef            goal.GoalRef
	changeRef          ports.ChangeSetRef
	action             ActionRecord
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		records:           make(map[goal.GoalRef]GoalRecord),
		requests:          make(map[string]goal.GoalRef),
		successors:        make(map[goal.AppSpecRef]goal.GoalRef),
		actions:           make(map[string]memoryAction),
		directorLeases:    make(map[goal.GoalRef]DirectorLeaseRecord),
		directorRequests:  make(map[string]memoryDirectorMutation),
		controlRequests:   make(map[string]ControlReplayRequest),
		controlRefs:       make(map[string]string),
		mailboxes:         make(map[MailboxMessageRef]MailboxRecord),
		mailboxRequests:   make(map[string]memoryMailboxMutation),
		integrationAdmits: make(map[string]memoryIntegrationAdmission),
		now:               time.Now,
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
		BudgetEnvelopes:     append([]governance.BudgetEnvelope(nil), state.BudgetEnvelopes...),
		WorkItemAuthorities: append([]WorkItemAuthority(nil), state.WorkItemAuthorities...),
	}
	repository.requests[requestKey] = state.Goal.Ref()
	repository.records[state.Goal.Ref()] = record
	for _, action := range state.Actions {
		if action.EffectIntent.Ref != "" {
			record.EffectIntents = append(record.EffectIntents, action.EffectIntent)
		}
		if action.EffectApproval != nil {
			record.EffectApprovals = append(record.EffectApprovals, *action.EffectApproval)
		}
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	repository.records[state.Goal.Ref()] = record
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

func (repository *memoryRepository) EffectReplay(
	_ context.Context,
	request EffectReplayRequest,
) (EffectApproval, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	record, found := repository.records[request.GoalRef]
	if !found || record.Goal.Project() != request.ProjectRef {
		return EffectApproval{}, false, &StateError{Code: StateNotFound}
	}
	for _, approval := range record.EffectApprovals {
		if approval.RequestRef != request.RequestRef || approval.DecidedBy != request.PrincipalRef {
			continue
		}
		if approval.RequestFingerprint != request.RequestFingerprint || approval.IntentRef != request.IntentRef ||
			approval.IntentDigest != request.IntentDigest {
			return EffectApproval{}, false, &StateError{Code: StateConflict}
		}
		return approval, true, nil
	}
	return EffectApproval{}, false, nil
}

func (repository *memoryRepository) DecideEffect(
	_ context.Context,
	state DecideEffectState,
) (EffectApproval, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	record, found := repository.records[state.GoalRef]
	if !found || record.Goal.Project() != state.ProjectRef || state.OperationAt.IsZero() ||
		!memoryWriteAuthorizationValid(
			state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef,
			identity.PermissionEffectsApprove,
			effectApprovalResourceRef(state.GoalRef, state.IntentRef, state.IntentDigest),
		) {
		return EffectApproval{}, false, &StateError{Code: StateInvalid}
	}
	for _, approval := range record.EffectApprovals {
		if approval.RequestRef == state.RequestRef && approval.DecidedBy == state.PrincipalRef {
			if approval.RequestFingerprint != state.RequestFingerprint {
				return EffectApproval{}, false, &StateError{Code: StateConflict}
			}
			return approval, false, nil
		}
	}
	intent, found := effectIntentByRef(record.EffectIntents, state.IntentRef)
	if !found || intent.Digest != state.IntentDigest || ValidateEffectApproval(intent, state.Approval) != nil {
		return EffectApproval{}, false, &StateError{Code: StateInvalid}
	}
	record.EffectApprovals = append(record.EffectApprovals, state.Approval)
	repository.records[state.GoalRef] = record
	return state.Approval, true, nil
}

func (repository *memoryRepository) RecordEffectAttempt(
	_ context.Context,
	state RecordEffectAttemptState,
) (EffectAttempt, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	action, found := repository.actions[state.Claim.Action.Ref]
	if !found || !memoryClaimMatches(action, state.Claim, state.OperationAt) ||
		validateEffectAttempt(state.Claim, state.Attempt) != nil {
		return EffectAttempt{}, false, &StateError{Code: StateConflict}
	}
	record := repository.records[state.Claim.Action.GoalRef]
	for _, attempt := range record.EffectAttempts {
		if attempt.ActionRef == state.Attempt.ActionRef && attempt.ActionFence == state.Attempt.ActionFence &&
			attempt.Ref != state.Attempt.Ref {
			return EffectAttempt{}, false, &StateError{Code: StateConflict}
		}
		if attempt.Ref == state.Attempt.Ref {
			if !reflect.DeepEqual(attempt, state.Attempt) {
				return EffectAttempt{}, false, &StateError{Code: StateConflict}
			}
			return attempt, false, nil
		}
	}
	record.EffectAttempts = append(record.EffectAttempts, state.Attempt)
	repository.records[record.Goal.Ref()] = record
	return state.Attempt, true, nil
}

func effectAttemptByRef(records []EffectAttempt, ref string) (EffectAttempt, bool) {
	for _, record := range records {
		if record.Ref == ref {
			return record, true
		}
	}
	return EffectAttempt{}, false
}

func (repository *memoryRepository) validateBudgetSettlementLocked(
	record GoalRecord,
	settlement *governance.BudgetSettlement,
) error {
	if settlement == nil {
		return nil
	}
	if governance.ValidateBudgetSettlement(*settlement) != nil {
		return &StateError{Code: StateInvalid}
	}
	if settlement.CausalAttemptRef == "" {
		return nil
	}
	reservation, found := reservationByRef(record.BudgetReservations, settlement.ReservationRef)
	if !found || reservation.Resources != settlement.Reserved || settlement.SettledAt.Before(reservation.ReservedAt) {
		return &StateError{Code: StateInvalid}
	}
	attempt, found := effectAttemptByRef(record.EffectAttempts, settlement.CausalAttemptRef)
	if !found || attempt.ActionRef != reservation.ActionRef || attempt.IntentRef != reservation.EffectIntentRef ||
		reservation.Fence > attempt.ActionFence || !governance.IsExactZeroRelease(*settlement) {
		return &StateError{Code: StateInvalid}
	}
	for _, candidate := range repository.records {
		for _, existing := range candidate.BudgetSettlements {
			if existing.CausalAttemptRef == settlement.CausalAttemptRef {
				return &StateError{Code: StateConflict}
			}
		}
	}
	return nil
}

func (repository *memoryRepository) ControlReplay(
	_ context.Context,
	request ControlReplayRequest,
) (ControlRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	key := memoryControlRequestKey(request.PrincipalRef, request.ProjectRef, request.RequestRef)
	stored, found := repository.controlRequests[key]
	if !found {
		return ControlRecord{}, false, nil
	}
	if stored != request {
		return ControlRecord{}, false, &StateError{Code: StateConflict}
	}
	control, found := repository.controlLocked(repository.controlRefs[key])
	if !found {
		return ControlRecord{}, false, &StateError{Code: StateConflict}
	}
	return control, true, nil
}

func (repository *memoryRepository) ApplyControl(
	_ context.Context,
	state ApplyControlState,
) (ControlRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	request := ControlReplayRequest{
		RequestRef: state.RequestRef, RequestFingerprint: state.RequestFingerprint,
		PrincipalRef: state.PrincipalRef, ProjectRef: state.ProjectRef, GoalRef: state.GoalRef,
	}
	key := memoryControlRequestKey(state.PrincipalRef, state.ProjectRef, state.RequestRef)
	if stored, found := repository.controlRequests[key]; found && state.ExpectedControlStatus == "" {
		if stored != request {
			return ControlRecord{}, false, &StateError{Code: StateConflict}
		}
		control, ok := repository.controlLocked(repository.controlRefs[key])
		if !ok {
			return ControlRecord{}, false, &StateError{Code: StateConflict}
		}
		return control, false, nil
	}
	authorized := memoryWriteAuthorizationValid(
		state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef,
		identity.PermissionGoalsDirect, state.GoalRef.String(),
	)
	if IsReviewCleanupControl(state.Control) {
		authorized = memoryWriteAuthorizationValid(
			state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef,
			identity.PermissionGoalsCreate, state.ProjectRef.String(),
		)
	}
	if !authorized || state.OperationAt.IsZero() {
		return ControlRecord{}, false, &StateError{Code: StateInvalid}
	}
	current, found := repository.records[state.GoalRef]
	if !found || current.Goal.Project() != state.ProjectRef ||
		current.Goal.Revision() != state.ExpectedGoalRevision ||
		current.Goal.PlanGeneration() != state.ExpectedPlanGeneration {
		return ControlRecord{}, false, &StateError{Code: StateConflict}
	}
	if state.ExpectedWorkItemRevision != 0 {
		workItemRef := state.Control.WorkItemRef
		if state.Claim.Action.WorkItemRef.String() != "" {
			workItemRef = state.Claim.Action.WorkItemRef
		}
		item, ok := current.Goal.WorkItem(workItemRef)
		if !ok || item.Revision() != state.ExpectedWorkItemRevision {
			return ControlRecord{}, false, &StateError{Code: StateConflict}
		}
	}
	if state.ExpectedExecutionState != "" {
		executionRef := state.Control.ExecutionRef
		if executionRef.String() == "" {
			executionRef = state.Claim.Action.ExecutionRef
		}
		execution, ok := executionByRef(current.Executions, executionRef)
		if !ok || execution.State != state.ExpectedExecutionState {
			return ControlRecord{}, false, &StateError{Code: StateConflict}
		}
	}
	created := state.ExpectedControlStatus == ""
	if created {
		if state.Control.Status != ControlRequested && state.Control.Status != ControlConfirmed {
			return ControlRecord{}, false, &StateError{Code: StateInvalid}
		}
	} else {
		existing, ok := repository.controlLocked(state.Control.Ref)
		if !ok || existing.Status != state.ExpectedControlStatus ||
			existing.RequestFingerprint != state.RequestFingerprint {
			return ControlRecord{}, false, &StateError{Code: StateConflict}
		}
	}
	preRetiredAction := ""
	if state.SupersededControl != nil {
		old := *state.SupersededControl
		original := old
		original.Status = ControlRequested
		original.SupersededAt = time.Time{}
		original.SupersededByControlRef = ""
		existing, ok := repository.controlLocked(old.Ref)
		wantOldAction := "action:stop:" + old.Ref + ":" + old.ExecutionRef.String()
		wantNewAction := "action:stop:" + state.Control.Ref + ":" + state.Control.ExecutionRef.String()
		if !created || !ok || !reflect.DeepEqual(existing, original) ||
			old.Status != ControlSuperseded || old.Mode != ports.AgentStopCooperative ||
			old.SupersededByControlRef != state.Control.Ref ||
			!old.SupersededAt.Equal(state.OperationAt) ||
			state.Control.Mode != ports.AgentStopForced || state.Control.SupersedesControlRef != old.Ref ||
			old.ExecutionRef != state.Control.ExecutionRef || old.ExecutionAttempt != state.Control.ExecutionAttempt ||
			old.GoalRevision >= state.Control.GoalRevision || old.WorkItemRevision > state.Control.WorkItemRevision ||
			len(state.RetireActionRefs) != 1 || state.RetireActionRefs[0] != wantOldAction ||
			len(state.NewActions) != 1 || state.NewActions[0].Ref != wantNewAction {
			return ControlRecord{}, false, &StateError{Code: StateInvalid}
		}
		current.Controls = replaceControl(current.Controls, old)
		if receipt, retired := repository.retireActionLocked(
			wantOldAction, "control-retire:"+state.Control.Ref,
			state.Control.PrincipalRef.String(), state.OperationAt,
		); retired {
			current.ConsumptionReceipts = append(current.ConsumptionReceipts, receipt)
		} else {
			return ControlRecord{}, false, &StateError{Code: StateConflict}
		}
		preRetiredAction = wantOldAction
	} else if created && state.Control.SupersedesControlRef != "" {
		return ControlRecord{}, false, &StateError{Code: StateInvalid}
	}
	if state.Claim.Token != "" {
		action, ok := repository.actions[state.Claim.Action.Ref]
		if !ok || !memoryClaimMatches(action, state.Claim, state.OperationAt) {
			return ControlRecord{}, false, &StateError{Code: StateConflict}
		}
	}
	if state.RequireMailboxClearForExecutionRef.String() != "" &&
		repository.hasRetiredMailboxRecipientLocked(state.RequireMailboxClearForExecutionRef) {
		return ControlRecord{}, false, &StateError{Code: StateRecipientMailboxActive}
	}
	if state.EffectReceipt != nil {
		attempt, ok := effectAttemptByRef(current.EffectAttempts, state.EffectReceipt.AttemptRef)
		if !ok || validateEffectReceipt(state.Claim, attempt, *state.EffectReceipt) != nil {
			return ControlRecord{}, false, &StateError{Code: StateInvalid}
		}
	}
	if err := repository.validateBudgetSettlementLocked(current, state.BudgetSettlement); err != nil {
		return ControlRecord{}, false, err
	}
	for _, action := range state.NewActions {
		if _, duplicate := repository.actions[action.Ref]; duplicate {
			return ControlRecord{}, false, &StateError{Code: StateConflict}
		}
		if action.EffectIntent.Ref != "" {
			current.EffectIntents = append(current.EffectIntents, action.EffectIntent)
		}
		if action.EffectApproval != nil {
			current.EffectApprovals = append(current.EffectApprovals, *action.EffectApproval)
		}
	}
	for _, cleanup := range state.NewControls {
		if !IsReviewCleanupControl(cleanup) {
			return ControlRecord{}, false, &StateError{Code: StateInvalid}
		}
		current.Controls = append(current.Controls, cleanup)
	}
	current.Goal = state.Goal
	for _, execution := range state.Executions {
		current.Executions = replaceExecution(current.Executions, execution)
	}
	if created {
		current.Controls = append(current.Controls, state.Control)
	} else {
		current.Controls = replaceControl(current.Controls, state.Control)
	}
	if state.Claim.Token != "" {
		receipt := consumptionReceipt(state.Claim, ActionConsumedCompleted, state.ClaimErrorCode, state.OperationAt)
		if state.EffectReceipt != nil {
			receipt.EffectReceiptRef = state.EffectReceipt.Ref
			current.EffectReceipts = append(current.EffectReceipts, *state.EffectReceipt)
		}
		if state.BudgetSettlement != nil {
			current.BudgetSettlements = append(current.BudgetSettlements, *state.BudgetSettlement)
		}
		current.ConsumptionReceipts = append(current.ConsumptionReceipts, receipt)
		delete(repository.actions, state.Claim.Action.Ref)
	}
	for _, ref := range state.RetireActionRefs {
		if ref == preRetiredAction {
			continue
		}
		if receipt, retired := repository.retireActionLocked(
			ref, "control-retire:"+state.Control.Ref, state.Control.PrincipalRef.String(), state.OperationAt,
		); retired {
			current.ConsumptionReceipts = append(current.ConsumptionReceipts, receipt)
		}
	}
	for _, action := range state.NewActions {
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	if state.RetireMailboxForExecutionRef.String() != "" {
		retired := repository.retireControlledMailboxLocked(
			state.RetireMailboxForExecutionRef, state.OperationAt,
		)
		if retired {
			for index := range current.Executions {
				if current.Executions[index].Ref == state.RetireMailboxForExecutionRef {
					current.Executions[index].RecipientMailboxRetired = true
				}
			}
		}
	}
	repository.records[state.GoalRef] = current
	repository.events = append(repository.events, state.Events...)
	repository.controlRequests[key] = request
	repository.controlRefs[key] = state.Control.Ref
	return state.Control, created, nil
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
		parent.State() != goal.WorkItemStateRunning || current.Goal.CancelRequested() || parent.CancelRequested() {
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
	sort.Slice(refs, func(left, right int) bool {
		leftAction := repository.actions[refs[left]].record
		rightAction := repository.actions[refs[right]].record
		leftPriority := memoryActionPriority(leftAction)
		rightPriority := memoryActionPriority(rightAction)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		return refs[left] < refs[right]
	})
	for _, ref := range refs {
		action := repository.actions[ref]
		if action.record.Kind == ActionDeliverMailbox || action.record.AvailableAt.After(now) ||
			(action.token != "" && action.lease.After(now)) {
			continue
		}
		record := repository.records[action.record.GoalRef]
		item, found := record.Goal.WorkItem(action.record.WorkItemRef)
		if !found || repository.workItemLeaseActiveLocked(action.record, now) {
			continue
		}
		if action.record.Kind == ActionLaunchAgent {
			paused, _ := record.Goal.EffectivePause(item.Ref())
			if paused || record.Goal.CancelRequested() || item.CancelRequested() {
				continue
			}
		}
		execution, executionFound := executionForAction(record, action.record)
		if executionFound && action.record.Kind == ActionStopAgent && execution.State == ExecutionDispatching &&
			execution.ExternalRef == "" {
			continue
		}
		if action.record.Kind == ActionObserveAgent && repository.stopPendingLocked(action.record) {
			continue
		}
		if !ports.MatchAgentCapabilities(request.Capabilities, ports.AgentRequirements{
			RoleKey: item.Role().String(), SkillRefs: workItemRefs(item.SkillRefs()),
			ToolRefs: workItemRefs(item.ToolRefs()), CapabilityRefs: workItemRefs(item.CapabilityRefs()),
		}) {
			continue
		}
		if executionFound && (action.record.Kind == ActionObserveAgent || action.record.Kind == ActionStopAgent) &&
			(execution.ProviderRef != request.Capabilities.ProviderRef ||
				execution.ModelRef != request.Capabilities.ModelRef ||
				execution.AgentRef != request.Capabilities.AgentRef) {
			continue
		}
		var approval EffectApproval
		var reservation governance.BudgetReservation
		terminalStop := action.record.Kind == ActionStopAgent && executionFound &&
			(execution.State == ExecutionSucceeded || execution.State == ExecutionFailed ||
				execution.State == ExecutionCanceled || execution.State == ExecutionStopped)
		if action.record.Kind == ActionLaunchAgent || action.record.Kind == ActionPrepareWorkspace ||
			action.record.Kind == ActionCommitChange || action.record.Kind == ActionAttestTest ||
			action.record.Kind == ActionIntegrateChange ||
			action.record.Kind == ActionStopAgent && !terminalStop {
			approval, found = memoryLiveApproval(record, action.record, now)
			if !found {
				continue
			}
		}
		if action.record.Kind == ActionLaunchAgent {
			if ValidateBudgetPolicy(request.BudgetPolicy) != nil {
				return ActionClaim{}, false, &StateError{Code: StateInvalid}
			}
			reservation, found = repository.activeActionReservationLocked(record, action.record)
			if !found {
				fits, quotaErr := repository.budgetFitsLocked(record, action.record.EffectIntent)
				if quotaErr != nil {
					return ActionClaim{}, false, quotaErr
				}
				if !fits {
					action.record.AvailableAt = now.Add(action.record.EffectIntent.QuotaRetryDelay)
					repository.actions[ref] = action
					continue
				}
				reservation = memoryReservation(action.record, action.fence+1, now)
				record.BudgetReservations = append(record.BudgetReservations, reservation)
			}
			if executionFound && execution.State == ExecutionDispatching {
				execution.BudgetReservationRef = reservation.Ref
				execution.EffectIntentRef = action.record.EffectIntent.Ref
				record.Executions = replaceExecution(record.Executions, execution)
			}
			repository.records[record.Goal.Ref()] = record
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
			BudgetReservationRef: reservation.Ref, BudgetReservation: reservation, EffectApproval: approval,
		}, true, nil
	}
	return ActionClaim{}, false, nil
}

func memoryLiveApproval(record GoalRecord, action ActionRecord, now time.Time) (EffectApproval, bool) {
	var latest *EffectApproval
	for index := range record.EffectApprovals {
		candidate := record.EffectApprovals[index]
		if candidate.IntentRef != action.EffectIntent.Ref || candidate.IntentDigest != action.EffectIntent.Digest ||
			ValidateEffectApproval(action.EffectIntent, candidate) != nil {
			continue
		}
		if latest == nil || memoryApprovalLater(candidate, *latest) {
			copy := candidate
			latest = &copy
		}
	}
	if latest == nil || latest.Decision != EffectApproved ||
		(latest.Source == EffectApprovalSourceExplicitDecision && !latest.ExpiresAt.After(now)) {
		return EffectApproval{}, false
	}
	return *latest, true
}

func memoryApprovalLater(left, right EffectApproval) bool {
	if !left.DecidedAt.Equal(right.DecidedAt) {
		return left.DecidedAt.After(right.DecidedAt)
	}
	leftExplicit, rightExplicit := left.Source == EffectApprovalSourceExplicitDecision, right.Source == EffectApprovalSourceExplicitDecision
	if leftExplicit != rightExplicit {
		return leftExplicit
	}
	if (left.Decision == EffectDenied) != (right.Decision == EffectDenied) {
		return left.Decision == EffectDenied
	}
	return left.Ref > right.Ref
}

func memoryReservation(action ActionRecord, fence uint64, at time.Time) governance.BudgetReservation {
	intent := action.EffectIntent
	return governance.BudgetReservation{
		Ref: "budget-reservation:" + action.Ref + ":" + fmt.Sprint(fence), DemandRef: intent.Demand.Ref,
		ActionRef: action.Ref, EffectIntentRef: intent.Ref, ProjectRef: intent.Subject.ProjectRef.String(),
		GoalRef: intent.Subject.GoalRef.String(), WorkItemRef: intent.Subject.WorkItemRef.String(),
		ExecutionRef: intent.Subject.ExecutionRef.String(), PlanGeneration: uint64(intent.Subject.PlanGeneration),
		AppSpecGeneration: uint64(intent.Subject.AppSpecGeneration), WorkItemGeneration: uint64(action.WorkItemGeneration),
		Fence: fence, SpecHash: intent.Subject.SpecHash, PolicyHash: intent.PolicyHash,
		Resources: intent.Demand.Resources, ReservedAt: at,
	}
}

func (repository *memoryRepository) activeActionReservationLocked(
	record GoalRecord,
	action ActionRecord,
) (governance.BudgetReservation, bool) {
	settled := make(map[string]struct{}, len(record.BudgetSettlements))
	for _, settlement := range record.BudgetSettlements {
		settled[settlement.ReservationRef] = struct{}{}
	}
	for _, reservation := range record.BudgetReservations {
		if reservation.ActionRef == action.Ref {
			if _, done := settled[reservation.Ref]; !done {
				return reservation, true
			}
		}
	}
	return governance.BudgetReservation{}, false
}

func (repository *memoryRepository) budgetFitsLocked(
	goalRecord GoalRecord,
	intent EffectIntent,
) (bool, error) {
	deployment, project, goalUsage := governance.ResourceVector{}, governance.ResourceVector{}, governance.ResourceVector{}
	for _, record := range repository.records {
		settled := make(map[string]struct{}, len(record.BudgetSettlements))
		for _, fact := range record.BudgetSettlements {
			settled[fact.ReservationRef] = struct{}{}
		}
		for _, reservation := range record.BudgetReservations {
			if _, done := settled[reservation.Ref]; done {
				continue
			}
			var err error
			deployment, err = governance.Add(deployment, reservation.Resources)
			if err != nil {
				return false, err
			}
			if record.Goal.Project() == goalRecord.Goal.Project() {
				project, err = governance.Add(project, reservation.Resources)
			}
			if record.Goal.Ref() == goalRecord.Goal.Ref() {
				goalUsage, err = governance.Add(goalUsage, reservation.Resources)
			}
			if err != nil {
				return false, err
			}
		}
	}
	limits, err := memoryHistoricalBudgetLimits(goalRecord, intent)
	if err != nil {
		return false, err
	}
	for _, pair := range [][2]governance.ResourceVector{
		{limits[0], deployment}, {limits[1], project}, {limits[2], goalUsage},
	} {
		requested, err := governance.Add(pair[1], intent.Demand.Resources)
		if err != nil {
			return false, err
		}
		fits, err := governance.Fits(pair[0], requested)
		if err != nil || !fits {
			return false, err
		}
	}
	return true, nil
}

func memoryHistoricalBudgetLimits(record GoalRecord, intent EffectIntent) ([3]governance.ResourceVector, error) {
	var limits [3]governance.ResourceVector
	var found [3]bool
	for _, envelope := range record.BudgetEnvelopes {
		if envelope.PolicyHash != intent.PolicyHash || envelope.Revision != intent.PolicyRevision {
			continue
		}
		index := -1
		switch {
		case envelope.Scope == governance.BudgetScopeDeployment:
			index = 0
		case envelope.Scope == governance.BudgetScopeProject && envelope.SubjectRef == intent.Subject.ProjectRef.String():
			index = 1
		case envelope.Scope == governance.BudgetScopeGoal && envelope.SubjectRef == intent.Subject.GoalRef.String():
			index = 2
		}
		if index >= 0 {
			if found[index] {
				return limits, errors.New("test.budget_policy_duplicate")
			}
			limits[index], found[index] = envelope.Limit, true
		}
	}
	if !found[0] || !found[1] || !found[2] {
		return limits, errors.New("test.budget_policy_missing")
	}
	return limits, nil
}

func memoryActionPriority(action ActionRecord) int {
	switch action.Kind {
	case ActionStopAgent:
		return 0
	case ActionLaunchAgent:
		return 1
	case ActionObserveAgent:
		return 2
	default:
		return 3
	}
}

func (repository *memoryRepository) workItemLeaseActiveLocked(candidate ActionRecord, now time.Time) bool {
	for _, action := range repository.actions {
		if action.record.Ref != candidate.Ref && action.record.GoalRef == candidate.GoalRef &&
			action.record.WorkItemRef == candidate.WorkItemRef && action.token != "" && action.lease.After(now) {
			return true
		}
	}
	return false
}

func (repository *memoryRepository) stopPendingLocked(candidate ActionRecord) bool {
	for _, action := range repository.actions {
		if action.record.Kind == ActionStopAgent && action.record.GoalRef == candidate.GoalRef &&
			action.record.WorkItemRef == candidate.WorkItemRef {
			return true
		}
	}
	return false
}

func (repository *memoryRepository) retireActionLocked(
	ref string,
	tokenPrefix string,
	workerRef string,
	at time.Time,
) (ActionConsumptionReceipt, bool) {
	action, found := repository.actions[ref]
	if !found {
		return ActionConsumptionReceipt{}, false
	}
	if action.token == "" {
		action.token = tokenPrefix + ":" + ref
		action.workerRef = workerRef
		action.deliveryAttempt++
		action.fence++
	}
	claim := ActionClaim{
		Action: action.record, Token: action.token, WorkerRef: action.workerRef,
		DeliveryAttempt: action.deliveryAttempt, Fence: action.fence, LeaseUntil: at.Add(time.Nanosecond),
	}
	delete(repository.actions, ref)
	return consumptionReceipt(claim, ActionConsumedCompleted, "", at), true
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
	attempt, found := effectAttemptByRef(record.EffectAttempts, state.EffectReceipt.AttemptRef)
	if !found || validateEffectReceipt(state.Claim, attempt, state.EffectReceipt) != nil {
		return &StateError{Code: StateInvalid}
	}
	if isReviewerExecution(state.Execution) &&
		!reviewExternalRefAvailable(record, state.Execution, state.Execution.ExternalRef) {
		return &StateError{Code: StateConflict}
	}
	record.Executions = replaceExecution(record.Executions, state.Execution)
	receipt := consumptionReceipt(state.Claim, ActionConsumedCompleted, "", state.OperationAt)
	receipt.EffectReceiptRef = state.EffectReceipt.Ref
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, receipt)
	record.EffectReceipts = append(record.EffectReceipts, state.EffectReceipt)
	repository.records[record.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	repository.actions[state.NextAction.Ref] = memoryAction{record: state.NextAction}
	memoryAddActionFacts(&record, state.NextAction)
	repository.records[record.Goal.Ref()] = record
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
	if state.ClearEffectBinding &&
		(state.BudgetSettlement == nil || state.Execution.BudgetReservationRef != "" || state.Execution.EffectIntentRef != "") {
		return &StateError{Code: StateInvalid}
	}
	if state.ClearEffectBinding && !memoryEffectBindingMatchesClaim(current, state.Claim) {
		return &StateError{Code: StateConflict}
	}
	if err := repository.validateBudgetSettlementLocked(record, state.BudgetSettlement); err != nil {
		return err
	}
	record.Executions = replaceExecution(record.Executions, state.Execution)
	if state.BudgetSettlement != nil {
		record.BudgetSettlements = append(record.BudgetSettlements, *state.BudgetSettlement)
	}
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
	if state.ErrorCode == effectUnknownAppliedCode &&
		(state.ClearEffectBinding || state.BudgetSettlement != nil) {
		return &StateError{Code: StateInvalid}
	}
	record := repository.records[state.Claim.Action.GoalRef]
	if err := repository.validateBudgetSettlementLocked(record, state.BudgetSettlement); err != nil {
		return err
	}
	if state.ClearEffectBinding {
		execution, found := executionForAction(record, state.Claim.Action)
		if !found || state.BudgetSettlement == nil || !memoryEffectBindingMatchesClaim(execution, state.Claim) {
			return &StateError{Code: StateInvalid}
		}
		execution.BudgetReservationRef = ""
		execution.EffectIntentRef = ""
		record.Executions = replaceExecution(record.Executions, execution)
	}
	if state.BudgetSettlement != nil {
		record.BudgetSettlements = append(record.BudgetSettlements, *state.BudgetSettlement)
	}
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, consumptionReceipt(state.Claim, ActionConsumedQuarantined, state.ErrorCode, state.OperationAt))
	repository.records[record.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	repository.events = append(repository.events, state.Event)
	return nil
}

func memoryEffectBindingMatchesClaim(execution ExecutionRecord, claim ActionClaim) bool {
	unbound := execution.BudgetReservationRef == "" && execution.EffectIntentRef == ""
	bound := execution.BudgetReservationRef == claim.BudgetReservationRef &&
		execution.EffectIntentRef == claim.Action.EffectIntentRef
	return unbound || bound
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
	if err := repository.validateBudgetSettlementLocked(record, state.BudgetSettlement); err != nil {
		return err
	}
	record.Goal = state.Goal
	record.Executions = replaceExecution(record.Executions, state.FailedExecution)
	record.Executions = append(record.Executions, state.ReplacementExecution)
	if state.BudgetSettlement != nil {
		record.BudgetSettlements = append(record.BudgetSettlements, *state.BudgetSettlement)
	}
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, consumptionReceipt(state.Claim, ActionConsumedCompleted, state.ErrorCode, state.OperationAt))
	repository.records[state.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	repository.actions[state.NextAction.Ref] = memoryAction{record: state.NextAction}
	memoryAddActionFacts(&record, state.NextAction)
	repository.records[state.Goal.Ref()] = record
	repository.events = append(repository.events, state.Events...)
	return nil
}

func (repository *memoryRepository) RecordExecutionInterrupted(
	_ context.Context,
	state ExecutionInterruptedState,
) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if err := repository.validateMutation(
		state.Claim, state.OperationAt, state.ExpectedGoalRevision, state.ExpectedItemRevision,
	); err != nil {
		return err
	}
	if repository.retireControlledMailboxLocked(state.Execution.Ref, state.OperationAt) {
		state.Execution.RecipientMailboxRetired = true
	}
	record := repository.records[state.Goal.Ref()]
	if err := repository.validateBudgetSettlementLocked(record, state.BudgetSettlement); err != nil {
		return err
	}
	record.Goal = state.Goal
	record.Executions = replaceExecution(record.Executions, state.Execution)
	record.Executions = append(record.Executions, state.NewExecutions...)
	if state.BudgetSettlement != nil {
		record.BudgetSettlements = append(record.BudgetSettlements, *state.BudgetSettlement)
	}
	record.ConsumptionReceipts = append(record.ConsumptionReceipts,
		consumptionReceipt(state.Claim, ActionConsumedCompleted, state.Execution.FailureCode, state.OperationAt))
	repository.records[state.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	for _, action := range state.NewActions {
		memoryAddActionFacts(&record, action)
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	repository.records[state.Goal.Ref()] = record
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
	if err := repository.validateBudgetSettlementLocked(record, state.BudgetSettlement); err != nil {
		return err
	}
	record.Goal = state.Goal
	record.Executions = replaceExecution(record.Executions, state.Execution)
	record.Executions = append(record.Executions, state.NewExecutions...)
	record.Artifacts = append(record.Artifacts, state.Artifact)
	record.Attestations = append(record.Attestations, state.Attestation)
	if state.BudgetSettlement != nil {
		record.BudgetSettlements = append(record.BudgetSettlements, *state.BudgetSettlement)
	}
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, consumptionReceipt(state.Claim, ActionConsumedCompleted, "", state.OperationAt))
	repository.records[state.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	for _, action := range state.NewActions {
		memoryAddActionFacts(&record, action)
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	repository.records[state.Goal.Ref()] = record
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
	if err := repository.validateBudgetSettlementLocked(record, state.BudgetSettlement); err != nil {
		return err
	}
	record.Goal = state.Goal
	record.Executions = replaceExecution(record.Executions, state.Execution)
	record.Executions = append(record.Executions, state.NewExecutions...)
	if state.BudgetSettlement != nil {
		record.BudgetSettlements = append(record.BudgetSettlements, *state.BudgetSettlement)
	}
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
		memoryAddActionFacts(&record, action)
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	repository.records[state.Goal.Ref()] = record
	repository.events = append(repository.events, state.Events...)
	return nil
}

// ProjectRepository is deliberately deterministic in the in-memory adapter.
// The durable adapter obtains this relation from the project hierarchy; tests
// only need the same opaque repository identity every time a project is read.
func (repository *memoryRepository) ProjectRepository(_ context.Context, projectRef goal.ProjectRef) (identity.RepositoryRef, error) {
	if projectRef.String() == "" {
		return identity.RepositoryRef{}, &StateError{Code: StateInvalid}
	}
	ref, err := identity.NewRepositoryRef("repository:" + projectRef.String())
	if err != nil {
		return identity.RepositoryRef{}, &StateError{Code: StateInvalid, Cause: err}
	}
	return ref, nil
}

func (repository *memoryRepository) ListPendingChanges(_ context.Context, query PendingChangeQuery) ([]PendingChange, error) {
	if query.ProjectRef.String() == "" || query.RepositoryRef.String() == "" || query.Limit <= 0 {
		return nil, &StateError{Code: StateInvalid}
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	result := make([]PendingChange, 0)
	for _, record := range repository.records {
		if record.Goal.Project() != query.ProjectRef {
			continue
		}
		for _, change := range record.ChangeSets {
			if change.ProjectRef != query.ProjectRef || change.RepositoryRef != query.RepositoryRef ||
				(query.ActorRef.String() != "" && change.ActorRef != query.ActorRef) {
				continue
			}
			var integration IntegrationReceipt
			var observation MergeObservation
			for _, candidate := range record.IntegrationReceipts {
				if candidate.ChangeRef == change.Ref {
					integration = candidate
				}
			}
			for _, candidate := range record.MergeObservations {
				if candidate.ChangeRef == change.Ref {
					observation = candidate
				}
			}
			// A successful integration is no longer pending. Conflicts and stale
			// receipts remain visible as pending repair work.
			if integration.Status == ports.IntegrationStatusIntegrated {
				continue
			}
			binding, found := workspaceBindingForExecution(record, change.ExecutionRef)
			if !found {
				return nil, &StateError{Code: StateConflict}
			}
			result = append(result, PendingChange{Binding: binding, ChangeSet: change, Observation: observation, Integration: integration})
		}
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].ChangeSet.CommittedAt.Equal(result[right].ChangeSet.CommittedAt) {
			return result[left].ChangeSet.Ref.String() < result[right].ChangeSet.Ref.String()
		}
		return result[left].ChangeSet.CommittedAt.Before(result[right].ChangeSet.CommittedAt)
	})
	if len(result) > query.Limit {
		result = result[:query.Limit]
	}
	return result, nil
}

func (repository *memoryRepository) RecordWorkspacePrepared(_ context.Context, state WorkspacePreparedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if err := repository.validateWorkspaceClaim(state.Claim, state.OperationAt); err != nil {
		// Preparing does not advance the Goal revision; validate the exact live
		// item separately because older test plans may begin at revision one.
		return err
	}
	record := repository.records[state.Claim.Action.GoalRef]
	if state.Execution.Ref != state.Claim.Action.ExecutionRef || ValidateWorkspaceBinding(state.Binding) != nil ||
		state.Binding.ExecutionRef != state.Execution.Ref || state.NextAction.Kind != ActionLaunchAgent ||
		state.NextAction.ExecutionRef != state.Execution.Ref {
		return &StateError{Code: StateInvalid}
	}
	if _, found := workspaceBindingForExecution(record, state.Execution.Ref); found {
		return &StateError{Code: StateConflict}
	}
	if _, found := repository.actions[state.NextAction.Ref]; found {
		return &StateError{Code: StateConflict}
	}
	attempt, found := effectAttemptByRef(record.EffectAttempts, state.EffectReceipt.AttemptRef)
	if !found || validateEffectReceipt(state.Claim, attempt, state.EffectReceipt) != nil {
		return &StateError{Code: StateInvalid}
	}
	record.Executions = replaceExecution(record.Executions, state.Execution)
	record.WorkspaceBindings = append(record.WorkspaceBindings, state.Binding)
	record.EffectReceipts = append(record.EffectReceipts, state.EffectReceipt)
	receipt := consumptionReceipt(state.Claim, ActionConsumedCompleted, "", state.OperationAt)
	receipt.EffectReceiptRef = state.EffectReceipt.Ref
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, receipt)
	memoryAddActionFacts(&record, state.NextAction)
	repository.records[record.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	repository.actions[state.NextAction.Ref] = memoryAction{record: state.NextAction}
	repository.events = append(repository.events, state.Event)
	return nil
}

func (repository *memoryRepository) RecordExecutionOutputReady(_ context.Context, state ExecutionOutputReadyState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if err := repository.validateMutation(state.Claim, state.OperationAt, state.ExpectedGoalRevision, state.ExpectedItemRevision); err != nil {
		return err
	}
	record := repository.records[state.Claim.Action.GoalRef]
	if err := repository.validateBudgetSettlementLocked(record, state.BudgetSettlement); err != nil {
		return err
	}
	if state.Execution.Ref != state.Claim.Action.ExecutionRef || state.Execution.State != ExecutionAwaitingCommit ||
		state.NextAction.Kind != ActionCommitChange || state.NextAction.ExecutionRef != state.Execution.Ref ||
		state.Artifact.GoalRef != record.Goal.Ref() || state.Attestation.ExecutionRef != state.Execution.Ref {
		return &StateError{Code: StateInvalid}
	}
	if _, found := repository.actions[state.NextAction.Ref]; found {
		return &StateError{Code: StateConflict}
	}
	record.Executions = replaceExecution(record.Executions, state.Execution)
	record.Artifacts = append(record.Artifacts, state.Artifact)
	record.Attestations = append(record.Attestations, state.Attestation)
	if state.BudgetSettlement != nil {
		record.BudgetSettlements = append(record.BudgetSettlements, *state.BudgetSettlement)
	}
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, consumptionReceipt(state.Claim, ActionConsumedCompleted, "", state.OperationAt))
	memoryAddActionFacts(&record, state.NextAction)
	repository.records[record.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	repository.actions[state.NextAction.Ref] = memoryAction{record: state.NextAction}
	repository.events = append(repository.events, state.Event)
	return nil
}

func (repository *memoryRepository) RecordChangeCommitted(_ context.Context, state ChangeCommittedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if err := repository.validateWorkspaceClaim(state.Claim, state.OperationAt); err != nil {
		return err
	}
	record := repository.records[state.Claim.Action.GoalRef]
	if state.Execution.Ref != state.Claim.Action.ExecutionRef || state.Execution.State != ExecutionAwaitingAttestation ||
		state.ChangeSet.Ref != state.Claim.Action.ChangeRef || state.ChangeSet.ExecutionRef != state.Execution.Ref ||
		ValidateChangeSet(state.ChangeSet) != nil {
		return &StateError{Code: StateInvalid}
	}
	item, itemFound := record.Goal.WorkItem(state.Execution.WorkItemRef)
	if !itemFound || (len(item.RequiredTests()) == 0) != (state.NextAction == nil) {
		return &StateError{Code: StateInvalid}
	}
	if state.NextAction != nil {
		if state.NextAction.Kind != ActionAttestTest || state.NextAction.ExecutionRef != state.Execution.Ref ||
			state.NextAction.ChangeRef != state.ChangeSet.Ref {
			return &StateError{Code: StateInvalid}
		}
		if _, found := repository.actions[state.NextAction.Ref]; found {
			return &StateError{Code: StateConflict}
		}
	}
	if _, found := changeSetByRef(record, state.ChangeSet.Ref); found {
		return &StateError{Code: StateConflict}
	}
	attempt, found := effectAttemptByRef(record.EffectAttempts, state.EffectReceipt.AttemptRef)
	if !found || validateEffectReceipt(state.Claim, attempt, state.EffectReceipt) != nil {
		return &StateError{Code: StateInvalid}
	}
	record.Executions = replaceExecution(record.Executions, state.Execution)
	record.ChangeSets = append(record.ChangeSets, state.ChangeSet)
	record.EffectReceipts = append(record.EffectReceipts, state.EffectReceipt)
	receipt := consumptionReceipt(state.Claim, ActionConsumedCompleted, "", state.OperationAt)
	receipt.EffectReceiptRef = state.EffectReceipt.Ref
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, receipt)
	if state.NextAction != nil {
		memoryAddActionFacts(&record, *state.NextAction)
	}
	repository.records[record.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	if state.NextAction != nil {
		repository.actions[state.NextAction.Ref] = memoryAction{record: *state.NextAction}
	}
	repository.events = append(repository.events, state.Event)
	return nil
}

func (repository *memoryRepository) RecordTestAttested(_ context.Context, state TestAttestedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if err := repository.validateMutation(
		state.Claim, state.OperationAt, state.ExpectedGoalRevision, state.ExpectedItemRevision,
	); err != nil {
		return err
	}
	record := repository.records[state.Claim.Action.GoalRef]
	item, itemFound := record.Goal.WorkItem(state.Claim.Action.WorkItemRef)
	change, changeFound := changeSetByRef(record, state.Claim.Action.ChangeRef)
	binding, bindingFound := workspaceBindingForExecution(record, state.Execution.Ref)
	if !itemFound || !changeFound || !bindingFound || state.Execution.Ref != state.Claim.Action.ExecutionRef ||
		state.Attestation.ExecutionRef != state.Execution.Ref || state.Attestation.ChangeSetRef != change.Ref ||
		state.Attestation.WorkItemGeneration != item.Revision() ||
		state.ManifestArtifact.ExecutionRef != state.Execution.Ref ||
		state.ReportArtifact.ExecutionRef != state.Execution.Ref || len(state.Events) == 0 {
		return &StateError{Code: StateInvalid}
	}
	for _, artifact := range record.Artifacts {
		if artifact.OccurrenceRef == state.ManifestArtifact.OccurrenceRef ||
			artifact.OccurrenceRef == state.ReportArtifact.OccurrenceRef {
			return &StateError{Code: StateConflict}
		}
	}
	for _, attestation := range record.Attestations {
		if attestation.Ref == state.Attestation.Ref {
			return &StateError{Code: StateConflict}
		}
	}
	attempt, found := effectAttemptByRef(record.EffectAttempts, state.EffectReceipt.AttemptRef)
	if !found || validateEffectReceipt(state.Claim, attempt, state.EffectReceipt) != nil {
		return &StateError{Code: StateInvalid}
	}
	policy := TestAttestationPolicy{Ref: state.Attestation.PolicyRef, Digest: state.Attestation.PolicyDigest}
	subject, err := buildTestSubject(record.Goal, item, state.Execution, binding, change, policy)
	if err != nil {
		return &StateError{Code: StateInvalid}
	}
	candidate := cloneGoalRecord(record)
	candidate.Artifacts = append(candidate.Artifacts, state.ManifestArtifact, state.ReportArtifact)
	candidate.Attestations = append(candidate.Attestations, state.Attestation)
	candidate.EffectReceipts = append(candidate.EffectReceipts, state.EffectReceipt)
	passed := state.Attestation.Verdict == AttestationVerdictPassed
	if !RequiredTestsAttestationEvidenceMatches(
		candidate, item, state.Execution, subject, state.Attestation, passed,
	) {
		return &StateError{Code: StateInvalid}
	}
	updatedItem, updatedFound := state.Goal.WorkItem(item.Ref())
	reviewRoundValid := validMemoryReviewRound(state, item)
	if !updatedFound || state.Goal.Ref() != record.Goal.Ref() ||
		(passed && (state.Execution.State != ExecutionAwaitingIntegration ||
			!reflect.DeepEqual(state.Goal.Snapshot(), record.Goal.Snapshot()) ||
			!reviewRoundValid)) ||
		(!passed && (state.Execution.State != ExecutionFailed ||
			updatedItem.State() != goal.WorkItemStateInterrupted || state.Execution.FailureCode == "")) {
		return &StateError{Code: StateInvalid}
	}
	for _, action := range state.NewActions {
		if _, exists := repository.actions[action.Ref]; exists {
			return &StateError{Code: StateConflict}
		}
		memoryAddActionFacts(&candidate, action)
	}
	candidate.Goal = state.Goal
	candidate.Executions = replaceExecution(candidate.Executions, state.Execution)
	candidate.Executions = append(candidate.Executions, state.NewExecutions...)
	receipt := consumptionReceipt(state.Claim, ActionConsumedCompleted, "", state.OperationAt)
	receipt.EffectReceiptRef = state.EffectReceipt.Ref
	candidate.ConsumptionReceipts = append(candidate.ConsumptionReceipts, receipt)
	repository.records[candidate.Goal.Ref()] = candidate
	delete(repository.actions, state.Claim.Action.Ref)
	for _, action := range state.NewActions {
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	repository.events = append(repository.events, state.Events...)
	return nil
}

func validMemoryReviewRound(state TestAttestedState, item goal.WorkItem) bool {
	if state.Execution.Purpose != ExecutionPurposeAuthor || len(item.WriteSet()) == 0 || len(item.RequiredTests()) == 0 {
		return len(state.NewExecutions) == 0 && len(state.NewActions) == 0
	}
	if len(state.NewExecutions) != 2 || len(state.NewActions) != 2 {
		return false
	}
	seen := map[ExecutionPurpose]bool{}
	for index, execution := range state.NewExecutions {
		if !isReviewerExecution(execution) || execution.WorkItemRef != item.Ref() ||
			execution.ReviewSubjectDigest == "" || state.NewActions[index].ExecutionRef != execution.Ref ||
			state.NewActions[index].Kind != ActionLaunchAgent {
			return false
		}
		seen[execution.Purpose] = true
	}
	return seen[ExecutionPurposePrimaryReview] && seen[ExecutionPurposeAdversarialReview]
}

func (repository *memoryRepository) RecordReviewAssessed(_ context.Context, state ReviewAssessedState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if err := repository.validateMutation(state.Claim, state.OperationAt, state.ExpectedGoalRevision, state.ExpectedItemRevision); err != nil {
		return err
	}
	record := repository.records[state.Claim.Action.GoalRef]
	item, found := record.Goal.WorkItem(state.Claim.Action.WorkItemRef)
	role, reviewer := reviewerRole(state.ReviewerExecution)
	if !found || !reviewer || state.ReviewerExecution.Ref != state.Claim.Action.ExecutionRef ||
		state.ReviewerExecution.State != ExecutionSucceeded || state.Review.Role != role ||
		state.Review.ReviewerExecutionRef != state.ReviewerExecution.Ref ||
		state.Review.SubjectDigest != state.ReviewerExecution.ReviewSubjectDigest ||
		state.Artifact.Kind != ArtifactKindReviewAssessment || state.Artifact.ExecutionRef != state.ReviewerExecution.Ref ||
		state.Review.AssessmentArtifactRef != state.Artifact.Stored.Ref.String() {
		return &StateError{Code: StateInvalid}
	}
	if _, err := review.NewAssessment(review.Assessment{SubjectDigest: state.Review.SubjectDigest,
		Role: state.Review.Role, Verdict: state.Review.Verdict,
		ReviewerExecutionRef:     state.Review.ReviewerExecutionRef.String(),
		ReviewerExecutionAttempt: state.Review.ReviewerExecutionAttempt,
		LaunchReceiptRef:         state.Review.LaunchReceiptRef, ReviewerExternalRef: state.Review.ExternalRef,
		AssessmentArtifactRef: state.Review.AssessmentArtifactRef,
		AssessmentDigest:      state.Review.AssessmentDigest, RecordedAt: state.Review.RecordedAt}); err != nil {
		return &StateError{Code: StateInvalid, Cause: err}
	}
	for _, prior := range record.Reviews {
		if prior.SubjectDigest == state.Review.SubjectDigest && prior.Role == state.Review.Role {
			return &StateError{Code: StateConflict}
		}
	}
	if err := repository.validateBudgetSettlementLocked(record, state.BudgetSettlement); err != nil {
		return err
	}
	record.Goal = state.Goal
	record.Executions = replaceExecution(record.Executions, state.ReviewerExecution)
	if state.AuthorExecution.Ref.String() != "" {
		record.Executions = replaceExecution(record.Executions, state.AuthorExecution)
	}
	record.Artifacts = append(record.Artifacts, state.Artifact)
	record.Reviews = append(record.Reviews, state.Review)
	if state.BudgetSettlement != nil {
		record.BudgetSettlements = append(record.BudgetSettlements, *state.BudgetSettlement)
	}
	record.ConsumptionReceipts = append(record.ConsumptionReceipts,
		consumptionReceipt(state.Claim, ActionConsumedCompleted, "", state.OperationAt))
	repository.records[record.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	repository.events = append(repository.events, state.Events...)
	_ = item
	return nil
}

func (repository *memoryRepository) RecordReviewExecutionReplaced(_ context.Context,
	state ReviewExecutionReplacedState,
) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if err := repository.validateMutation(state.Claim, state.OperationAt, state.ExpectedGoalRevision, state.ExpectedItemRevision); err != nil {
		return err
	}
	record := repository.records[state.Claim.Action.GoalRef]
	if !isReviewerExecution(state.FailedExecution) || !isReviewerExecution(state.ReplacementExecution) ||
		state.FailedExecution.State != ExecutionFailed || state.ReplacementExecution.State != ExecutionQueued ||
		state.ReplacementExecution.ReplacesExecutionRef != state.FailedExecution.Ref ||
		state.ReplacementExecution.Purpose != state.FailedExecution.Purpose ||
		state.ReplacementExecution.ReviewSubjectDigest != state.FailedExecution.ReviewSubjectDigest ||
		state.NextAction.ExecutionRef != state.ReplacementExecution.Ref {
		return &StateError{Code: StateInvalid}
	}
	if err := repository.validateBudgetSettlementLocked(record, state.BudgetSettlement); err != nil {
		return err
	}
	record.Executions = replaceExecution(record.Executions, state.FailedExecution)
	record.Executions = append(record.Executions, state.ReplacementExecution)
	if state.DiagnosticArtifact != nil {
		record.Artifacts = append(record.Artifacts, *state.DiagnosticArtifact)
	}
	if state.BudgetSettlement != nil {
		record.BudgetSettlements = append(record.BudgetSettlements, *state.BudgetSettlement)
	}
	record.ConsumptionReceipts = append(record.ConsumptionReceipts,
		consumptionReceipt(state.Claim, ActionConsumedCompleted, state.FailedExecution.FailureCode, state.OperationAt))
	memoryAddActionFacts(&record, state.NextAction)
	repository.records[record.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	repository.actions[state.NextAction.Ref] = memoryAction{record: state.NextAction}
	repository.events = append(repository.events, state.Events...)
	return nil
}

func (repository *memoryRepository) RecordReviewExecutionFailed(_ context.Context,
	state ReviewExecutionFailedState,
) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if err := repository.validateMutation(state.Claim, state.OperationAt, state.ExpectedGoalRevision, state.ExpectedItemRevision); err != nil {
		return err
	}
	record := repository.records[state.Claim.Action.GoalRef]
	if !isReviewerExecution(state.Execution) || state.Execution.State != ExecutionFailed ||
		state.Execution.Ref != state.Claim.Action.ExecutionRef ||
		(state.AuthorExecution.Ref.String() != "" && state.AuthorExecution.State != ExecutionFailed) ||
		(state.AuthorExecution.Ref.String() == "" && len(state.CleanupControls) == 0 && state.ResolvedCleanup == nil) {
		return &StateError{Code: StateInvalid}
	}
	if err := repository.validateBudgetSettlementLocked(record, state.BudgetSettlement); err != nil {
		return err
	}
	for _, actionRef := range state.RetireActionRefs {
		if _, found := repository.actions[actionRef]; !found {
			return &StateError{Code: StateConflict}
		}
	}
	for _, control := range state.CleanupControls {
		if !IsReviewCleanupControl(control) {
			return &StateError{Code: StateInvalid}
		}
		record.Controls = append(record.Controls, control)
	}
	if state.ResolvedCleanup != nil {
		previous, found := controlByRef(record.Controls, state.ResolvedCleanup.Ref)
		if !found || previous.Status != ControlRequested || !IsReviewCleanupControl(*state.ResolvedCleanup) ||
			state.ResolvedCleanup.Status != ControlConfirmed ||
			state.ResolvedCleanup.ExecutionRef != state.Execution.Ref ||
			state.ResolvedCleanup.ExecutionAttempt != state.Execution.AttemptNo {
			return &StateError{Code: StateInvalid}
		}
		record.Controls = replaceControl(record.Controls, *state.ResolvedCleanup)
	}
	for _, action := range state.CleanupActions {
		if _, found := repository.actions[action.Ref]; found {
			return &StateError{Code: StateConflict}
		}
		memoryAddActionFacts(&record, action)
	}
	record.Goal = state.Goal
	record.Executions = replaceExecution(record.Executions, state.Execution)
	if state.AuthorExecution.Ref.String() != "" {
		record.Executions = replaceExecution(record.Executions, state.AuthorExecution)
	}
	for _, retirement := range state.RetiredReviewers {
		previous, found := executionByRef(record.Executions, retirement.Execution.Ref)
		if !found || previous.State != retirement.ExpectedState || !isReviewerExecution(retirement.Execution) ||
			retirement.Execution.State != ExecutionFailed {
			return &StateError{Code: StateConflict}
		}
		record.Executions = replaceExecution(record.Executions, retirement.Execution)
	}
	if state.DiagnosticArtifact != nil {
		record.Artifacts = append(record.Artifacts, *state.DiagnosticArtifact)
	}
	if state.EffectReceipt != nil {
		attempt, found := effectAttemptByRef(record.EffectAttempts, state.EffectReceipt.AttemptRef)
		if !found || validateEffectReceipt(state.Claim, attempt, *state.EffectReceipt) != nil {
			return &StateError{Code: StateInvalid}
		}
		record.EffectReceipts = append(record.EffectReceipts, *state.EffectReceipt)
	}
	if state.BudgetSettlement != nil {
		record.BudgetSettlements = append(record.BudgetSettlements, *state.BudgetSettlement)
	}
	receipt := consumptionReceipt(state.Claim, ActionConsumedCompleted, state.Execution.FailureCode, state.OperationAt)
	if state.QuarantineClaim {
		receipt.Outcome = ActionConsumedQuarantined
	}
	if state.EffectReceipt != nil {
		receipt.EffectReceiptRef = state.EffectReceipt.Ref
	}
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, receipt)
	repository.records[record.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	for _, actionRef := range state.RetireActionRefs {
		if retired, found := repository.retireActionLocked(actionRef,
			"review-round:"+state.Execution.Ref.String(), "system:review-round", state.OperationAt); found {
			record.ConsumptionReceipts = append(record.ConsumptionReceipts, retired)
		} else {
			return &StateError{Code: StateConflict}
		}
	}
	for _, action := range state.CleanupActions {
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	repository.records[record.Goal.Ref()] = record
	repository.events = append(repository.events, state.Events...)
	return nil
}

func (repository *memoryRepository) AdmitIntegration(_ context.Context, state AdmitIntegrationState) (ActionRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if !memoryWriteAuthorizationValid(state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef,
		identity.PermissionChangesIntegrate, state.GoalRef.String()) {
		return ActionRecord{}, false, &StateError{Code: StateInvalid}
	}
	record, found := repository.records[state.GoalRef]
	if !found || record.Goal.Project() != state.ProjectRef {
		return ActionRecord{}, false, &StateError{Code: StateNotFound}
	}
	key := state.PrincipalRef.String() + "\x00" + state.ProjectRef.String() + "\x00" + state.RequestRef
	if previous, exists := repository.integrationAdmits[key]; exists {
		if previous.requestFingerprint != state.RequestFingerprint || previous.goalRef != state.GoalRef ||
			previous.changeRef != state.ChangeRef || previous.action.Ref != state.Action.Ref ||
			previous.action.ExpectedTargetOID != state.Action.ExpectedTargetOID {
			return ActionRecord{}, false, &StateError{Code: StateConflict}
		}
		return cloneActionRecord(previous.action), false, nil
	}
	if state.Action.Kind != ActionIntegrateChange || state.Action.GoalRef != state.GoalRef || state.Action.ChangeRef != state.ChangeRef ||
		state.Action.EffectApproval == nil || state.Action.EffectIntent.Permission != identity.PermissionChangesIntegrate ||
		state.Action.EffectIntent.RequestFingerprint != state.RequestFingerprint ||
		record.Goal.Revision() != state.ExpectedGoalRevision {
		return ActionRecord{}, false, &StateError{Code: StateInvalid}
	}
	item, itemFound := record.Goal.WorkItem(state.Action.WorkItemRef)
	execution, executionFound := executionByRef(record.Executions, state.Action.ExecutionRef)
	change, changeFound := changeSetByRef(record, state.ChangeRef)
	if !itemFound || !executionFound || !changeFound || item.Revision() != state.ExpectedItemRevision ||
		item.State() != goal.WorkItemStateRunning || execution.State != ExecutionAwaitingIntegration ||
		change.ExecutionRef != execution.Ref || change.ProjectRef != state.ProjectRef ||
		!anyRequiredTestsPassForChange(record, item, execution, change) {
		return ActionRecord{}, false, &StateError{Code: StateConflict}
	}
	for _, receipt := range record.IntegrationReceipts {
		if receipt.ChangeRef == state.ChangeRef && receipt.Status == ports.IntegrationStatusIntegrated {
			return ActionRecord{}, false, &StateError{Code: StateConflict}
		}
	}
	if _, found := repository.actions[state.Action.Ref]; found {
		return ActionRecord{}, false, &StateError{Code: StateConflict}
	}
	memoryAddActionFacts(&record, state.Action)
	repository.records[state.GoalRef] = record
	repository.actions[state.Action.Ref] = memoryAction{record: state.Action}
	repository.integrationAdmits[key] = memoryIntegrationAdmission{
		requestFingerprint: state.RequestFingerprint, goalRef: state.GoalRef,
		changeRef: state.ChangeRef, action: cloneActionRecord(state.Action),
	}
	return state.Action, true, nil
}

func cloneActionRecord(action ActionRecord) ActionRecord {
	cloned := action
	if action.EffectApproval != nil {
		approval := *action.EffectApproval
		cloned.EffectApproval = &approval
	}
	return cloned
}

func (repository *memoryRepository) RecordIntegrationResult(_ context.Context, state IntegrationResultState) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if err := repository.validateMutation(state.Claim, state.OperationAt, state.ExpectedGoalRevision, state.ExpectedItemRevision); err != nil {
		return err
	}
	record := repository.records[state.Claim.Action.GoalRef]
	if state.Execution.Ref != state.Claim.Action.ExecutionRef || state.Integration.ChangeRef != state.Claim.Action.ChangeRef ||
		ValidateMergeObservation(state.Observation) != nil || ValidateIntegrationReceipt(state.Integration) != nil {
		return &StateError{Code: StateInvalid}
	}
	for _, previous := range record.IntegrationReceipts {
		if previous.ChangeRef == state.Integration.ChangeRef {
			return &StateError{Code: StateConflict}
		}
	}
	attempt, found := effectAttemptByRef(record.EffectAttempts, state.EffectReceipt.AttemptRef)
	if !found || validateEffectReceipt(state.Claim, attempt, state.EffectReceipt) != nil {
		return &StateError{Code: StateInvalid}
	}
	record.Goal = state.Goal
	record.Executions = replaceExecution(record.Executions, state.Execution)
	record.Executions = append(record.Executions, state.NewExecutions...)
	record.MergeObservations = append(record.MergeObservations, state.Observation)
	record.IntegrationReceipts = append(record.IntegrationReceipts, state.Integration)
	record.EffectReceipts = append(record.EffectReceipts, state.EffectReceipt)
	receipt := consumptionReceipt(state.Claim, ActionConsumedCompleted, "", state.OperationAt)
	receipt.EffectReceiptRef = state.EffectReceipt.Ref
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, receipt)
	for _, action := range state.NewActions {
		if _, exists := repository.actions[action.Ref]; exists {
			return &StateError{Code: StateConflict}
		}
		memoryAddActionFacts(&record, action)
	}
	repository.records[record.Goal.Ref()] = record
	delete(repository.actions, state.Claim.Action.Ref)
	for _, action := range state.NewActions {
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	repository.events = append(repository.events, state.Events...)
	return nil
}

func (repository *memoryRepository) validateWorkspaceClaim(claim ActionClaim, operationAt time.Time) error {
	action, found := repository.actions[claim.Action.Ref]
	if !found || !memoryClaimMatches(action, claim, operationAt) {
		return &StateError{Code: StateConflict}
	}
	record, found := repository.records[claim.Action.GoalRef]
	if !found {
		return &StateError{Code: StateNotFound}
	}
	item, found := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !found || item.Revision() != claim.Action.WorkItemGeneration {
		return &StateError{Code: StateConflict}
	}
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

func (repository *memoryRepository) hasRetiredMailboxRecipientLocked(executionRef goal.ExecutionRef) bool {
	for _, mailbox := range repository.mailboxes {
		if mailbox.Envelope.Recipient.ExecutionRef == executionRef && mailbox.Retirement != nil {
			return true
		}
	}
	return false
}

func (repository *memoryRepository) retireControlledMailboxLocked(
	executionRef goal.ExecutionRef,
	at time.Time,
) bool {
	retired := false
	for messageRef, mailbox := range repository.mailboxes {
		if mailbox.Envelope.Recipient.ExecutionRef != executionRef ||
			mailbox.Acknowledgement != nil || mailbox.Retirement != nil {
			continue
		}
		retirement := MailboxRetirement{
			MessageRef: mailbox.Envelope.Ref, ActionRef: mailbox.Action.Ref,
			RecipientExecutionRef: executionRef,
			FailureCode:           "application.recipient_controlled_terminal", RetiredAt: at.UTC(),
		}
		mailbox.Retirement = &retirement
		mailbox.State = MailboxStateRetired
		repository.mailboxes[messageRef] = cloneMailboxRecord(mailbox)
		delete(repository.actions, mailbox.Action.Ref)
		retired = true
	}
	return retired
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
	for index := range record.Attestations {
		record.Attestations[index].Tests = append(
			[]ports.RequiredTestOutcome(nil), record.Attestations[index].Tests...,
		)
	}
	record.Controls = append([]ControlRecord(nil), record.Controls...)
	record.BudgetEnvelopes = append([]governance.BudgetEnvelope(nil), record.BudgetEnvelopes...)
	record.BudgetReservations = append([]governance.BudgetReservation(nil), record.BudgetReservations...)
	record.BudgetSettlements = append([]governance.BudgetSettlement(nil), record.BudgetSettlements...)
	record.WorkItemAuthorities = append([]WorkItemAuthority(nil), record.WorkItemAuthorities...)
	record.EffectIntents = append([]EffectIntent(nil), record.EffectIntents...)
	record.EffectApprovals = append([]EffectApproval(nil), record.EffectApprovals...)
	record.EffectAttempts = append([]EffectAttempt(nil), record.EffectAttempts...)
	record.EffectReceipts = append([]EffectReceipt(nil), record.EffectReceipts...)
	record.WorkspaceBindings = append([]WorkspaceBinding(nil), record.WorkspaceBindings...)
	record.ChangeSets = append([]ChangeSet(nil), record.ChangeSets...)
	record.MergeObservations = append([]MergeObservation(nil), record.MergeObservations...)
	record.IntegrationReceipts = append([]IntegrationReceipt(nil), record.IntegrationReceipts...)
	record.Reviews = append([]ReviewRecord(nil), record.Reviews...)
	record.ConsumptionReceipts = append([]ActionConsumptionReceipt(nil), record.ConsumptionReceipts...)
	return record
}

func memoryAddActionFacts(record *GoalRecord, action ActionRecord) {
	if action.EffectIntent.Ref != "" {
		record.EffectIntents = append(record.EffectIntents, action.EffectIntent)
	}
	if action.EffectApproval != nil {
		record.EffectApprovals = append(record.EffectApprovals, *action.EffectApproval)
	}
}

func memoryControlRequestKey(
	principal identity.PrincipalRef,
	project goal.ProjectRef,
	requestRef string,
) string {
	return principal.String() + "\x00" + project.String() + "\x00" + requestRef
}

func (repository *memoryRepository) controlLocked(ref string) (ControlRecord, bool) {
	for _, record := range repository.records {
		for _, control := range record.Controls {
			if control.Ref == ref {
				return control, true
			}
		}
	}
	return ControlRecord{}, false
}

func replaceControl(records []ControlRecord, updated ControlRecord) []ControlRecord {
	result := append([]ControlRecord(nil), records...)
	for index := range result {
		if result[index].Ref == updated.Ref {
			result[index] = updated
			return result
		}
	}
	return append(result, updated)
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

func onlyExecution(t *testing.T, record GoalRecord) ExecutionRecord {
	t.Helper()
	var result ExecutionRecord
	found := false
	for _, execution := range record.Executions {
		if isReviewerExecution(execution) {
			continue
		}
		if found {
			t.Fatalf("non-review execution count > 1")
		}
		result, found = execution, true
	}
	if !found {
		t.Fatal("non-review execution missing")
	}
	return result
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
	launchErrorHook                  func()
	launchOverride                   func(ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error)
	launchEntered                    chan struct{}
	launchRelease                    <-chan struct{}
	controlCapabilities              *ports.AgentControlCapabilities
	stopStatus                       ports.AgentStopStatus
	stopOverride                     func(ports.AgentStopRequest) (ports.AgentStopReceipt, error)
	stopCalls                        int
	stopRequests                     []ports.AgentStopRequest
}

// scriptedWorkspaceManager and scriptedVersionControl are contractual test
// adapters. They retain only opaque refs/OIDs: physical workspace paths are
// intentionally unavailable even to application tests.
type scriptedWorkspaceManager struct {
	mu       sync.Mutex
	prepared map[ports.ExecutionWorkspaceRef]ports.WorkspacePrepared
	requests []ports.WorkspacePrepareRequest
}

func (manager *scriptedWorkspaceManager) Prepare(_ context.Context, request ports.WorkspacePrepareRequest) (ports.WorkspacePrepared, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if previous, found := manager.prepared[request.WorkspaceRef]; found {
		return previous, nil
	}
	prepared := ports.WorkspacePrepared{
		WorkspaceRef: request.WorkspaceRef, RepositoryRef: request.RepositoryRef, ExecutionRef: request.ExecutionRef,
		TargetRef: "refs/heads/main", BaseOID: testGitOID('a'), ObjectFormat: ports.GitObjectFormatSHA1,
		WriteSetDigest: request.WriteSetDigest, AdapterRef: "workspace-adapter:test",
		ReceiptRef: "workspace-receipt:" + request.WorkspaceRef.String(), PreparedAt: request.PreparedAt,
	}
	if manager.prepared == nil {
		manager.prepared = make(map[ports.ExecutionWorkspaceRef]ports.WorkspacePrepared)
	}
	manager.prepared[request.WorkspaceRef] = prepared
	manager.requests = append(manager.requests, request)
	return prepared, nil
}

func (manager *scriptedWorkspaceManager) Inspect(_ context.Context, request ports.WorkspaceInspectRequest) (ports.WorkspaceInspection, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	prepared, found := manager.prepared[request.WorkspaceRef]
	if !found {
		return ports.WorkspaceInspection{}, errors.New("test.workspace_not_prepared")
	}
	return ports.WorkspaceInspection{WorkspaceRef: request.WorkspaceRef, RepositoryRef: request.RepositoryRef,
		ExecutionRef: request.ExecutionRef, BaseOID: prepared.BaseOID, HeadOID: prepared.BaseOID,
		TreeOID: testGitOID('b'), WriteSetDigest: prepared.WriteSetDigest, AdapterRef: prepared.AdapterRef,
		InspectedAt: time.Unix(1, 0).UTC()}, nil
}

func (manager *scriptedWorkspaceManager) Release(_ context.Context, request ports.WorkspaceReleaseRequest) (ports.WorkspaceReleaseReceipt, error) {
	return ports.WorkspaceReleaseReceipt{WorkspaceRef: request.WorkspaceRef, RepositoryRef: request.RepositoryRef,
		ExecutionRef: request.ExecutionRef, Released: false, ReceiptRef: "workspace-release:" + request.WorkspaceRef.String(),
		ReleasedAt: request.RequestedAt}, nil
}

type scriptedVersionControl struct {
	mu                  sync.Mutex
	commits             map[ports.ChangeSetRef]ports.CommitResult
	integrations        map[string]ports.IntegrationResult
	commitRequests      []ports.CommitRequest
	integrationRequests []ports.IntegrationRequest
}

func (control *scriptedVersionControl) Commit(_ context.Context, request ports.CommitRequest) (ports.CommitResult, error) {
	control.mu.Lock()
	defer control.mu.Unlock()
	if previous, found := control.commits[request.ChangeSetRef]; found {
		return previous, nil
	}
	paths := append([]string(nil), request.WriteSet...)
	result := ports.CommitResult{ChangeSetRef: request.ChangeSetRef, WorkspaceRef: request.WorkspaceRef,
		RepositoryRef: request.RepositoryRef, ExecutionRef: request.ExecutionRef, BaseOID: request.BaseOID,
		ParentOID: request.BaseOID, HeadOID: testGitOID('c'), TreeOID: testGitOID('d'),
		ObjectFormat: request.ObjectFormat, DiffDigest: testDigest("commit:" + request.ChangeSetRef.String()),
		ChangedPaths: paths, WriteSetDigest: request.WriteSetDigest, ParentChangeRef: request.ParentChangeRef,
		AdapterRef: "version-control:test", ReceiptRef: "commit-receipt:" + request.ChangeSetRef.String(),
		CommittedAt: request.CommittedAt}
	if control.commits == nil {
		control.commits = make(map[ports.ChangeSetRef]ports.CommitResult)
	}
	control.commits[request.ChangeSetRef] = result
	control.commitRequests = append(control.commitRequests, request)
	return result, nil
}

func (control *scriptedVersionControl) PreviewIntegration(_ context.Context, request ports.IntegrationPreviewRequest) (ports.IntegrationPreview, error) {
	return ports.IntegrationPreview{ChangeSetRef: request.ChangeSetRef, RepositoryRef: request.RepositoryRef,
		SourceOID: request.SourceOID, TargetRef: request.TargetRef, TargetOID: request.TargetOID,
		ObjectFormat: request.ObjectFormat, Status: ports.MergeStatusClean, CandidateTreeOID: testGitOID('e'),
		AdapterRef: "version-control:test", ObservedAt: request.RequestedAt}, nil
}

func (control *scriptedVersionControl) Integrate(_ context.Context, request ports.IntegrationRequest) (ports.IntegrationResult, error) {
	control.mu.Lock()
	defer control.mu.Unlock()
	if previous, found := control.integrations[request.IdempotencyKey]; found {
		return previous, nil
	}
	result := ports.IntegrationResult{ChangeSetRef: request.ChangeSetRef, RepositoryRef: request.RepositoryRef,
		SourceOID: request.SourceOID, TargetRef: request.TargetRef, TargetBeforeOID: request.ExpectedTargetOID,
		TargetAfterOID: testGitOID('f'), TreeOID: testGitOID('e'), ObjectFormat: request.ObjectFormat,
		Status: ports.IntegrationStatusIntegrated, MarkerRef: "refs/orquesta/effects:test",
		AdapterRef: "version-control:test", ReceiptRef: "integration-receipt:" + request.ChangeSetRef.String(),
		RecordedAt: request.RequestedAt}
	if control.integrations == nil {
		control.integrations = make(map[string]ports.IntegrationResult)
	}
	control.integrations[request.IdempotencyKey] = result
	control.integrationRequests = append(control.integrationRequests, request)
	return result, nil
}

func testGitOID(value byte) string { return strings.Repeat(string([]byte{value}), 40) }

func testDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

type scriptedTestAttestor struct {
	mu       sync.Mutex
	verdict  ports.TestAttestationVerdict
	runs     []TestAttestationRun
	requests []ports.TestAttestationRequest
	override func(ports.TestAttestationRequest) (ports.TestAttestationResult, error)
}

func (attestor *scriptedTestAttestor) Attest(
	_ context.Context,
	run TestAttestationRun,
) (ports.TestAttestationResult, error) {
	attestor.mu.Lock()
	defer attestor.mu.Unlock()
	request := run.Request
	attestor.runs = append(attestor.runs, run)
	attestor.requests = append(attestor.requests, request)
	if attestor.override != nil {
		return attestor.override(request)
	}
	verdict := attestor.verdict
	if verdict == "" {
		verdict = ports.TestAttestationPassed
	}
	outcomes := make([]ports.RequiredTestOutcome, len(request.RequiredTests))
	for index, spec := range request.RequiredTests {
		exitCode := 0
		if verdict == ports.TestAttestationFailed && index == 0 {
			exitCode = 1
		}
		outcomes[index] = ports.RequiredTestOutcome{
			RequiredTestRef: spec.Ref(), ExitCode: exitCode,
			OutputDigest: testDigest("test-output:" + spec.Ref().String()),
		}
	}
	manifest, err := ports.BuildTestSubjectManifest(request.Subject)
	if err != nil {
		return ports.TestAttestationResult{}, err
	}
	report, err := ports.BuildTestAttestationReport(ports.TestAttestationReportInput{
		SubjectDigest: request.SubjectDigest, Verdict: verdict, Tests: outcomes,
		AttestorRef: "test-attestor:scripted", PolicyRef: request.Subject.PolicyRef,
		PolicyDigest: request.Subject.PolicyDigest,
	})
	if err != nil {
		return ports.TestAttestationResult{}, err
	}
	receiptRef, err := ports.TestAttestationReceiptRef(report)
	if err != nil {
		return ports.TestAttestationResult{}, err
	}
	return ports.TestAttestationResult{
		Subject: request.Subject, SubjectDigest: request.SubjectDigest, Verdict: verdict,
		Manifest: manifest, Report: report, Tests: outcomes, AttestorRef: "test-attestor:scripted",
		ReceiptRef: receiptRef,
		PolicyRef:  request.Subject.PolicyRef, PolicyDigest: request.Subject.PolicyDigest,
		StartedAt: request.RequestedAt, FinishedAt: request.RequestedAt,
	}, nil
}

type definitelyUnappliedPermanentError struct{ message string }

func (err definitelyUnappliedPermanentError) Error() string {
	if err.message == "" {
		return "test.effect_rejected_before_apply"
	}
	return err.message
}

func (definitelyUnappliedPermanentError) DefinitelyNotApplied() bool { return true }

func (agent *scriptedAgent) ControlCapabilities(context.Context) (ports.AgentControlCapabilities, error) {
	if agent.controlCapabilities != nil {
		return *agent.controlCapabilities, nil
	}
	return ports.AgentControlCapabilities{CooperativeStop: true, ForcedStop: true}, nil
}

func (agent *scriptedAgent) Stop(
	_ context.Context,
	request ports.AgentStopRequest,
) (ports.AgentStopReceipt, error) {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	agent.stopCalls++
	agent.stopRequests = append(agent.stopRequests, request)
	if agent.stopOverride != nil {
		return agent.stopOverride(request)
	}
	status := agent.stopStatus
	if status == "" {
		status = ports.AgentStopped
	}
	receipt := ports.AgentStopReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
		ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
		Status: status,
	}
	if status == ports.AgentStopped || status == ports.AgentStopAlreadyStopped || status == ports.AgentStopAlreadyCompleted ||
		status == ports.AgentStopAlreadyFailed {
		receipt.ReceiptRef = "receipt:stop:" + request.ExecutionRef.String()
		receipt.ConfirmedAt = agent.now().UTC()
	}
	return receipt, nil
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
	launchErrorHook := agent.launchErrorHook
	launchOverride := agent.launchOverride
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
		if launchErrorHook != nil {
			launchErrorHook()
		}
		return ports.AgentLaunchReceipt{}, launchErr
	}
	if launchOverride != nil {
		return launchOverride(request)
	}
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: receiptSpecHash,
		ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test",
		ExternalRef:    "external:" + request.ExecutionRef.String(),
		ReceiptRef:     "receipt:launch:" + request.ExecutionRef.String(),
		IdempotencyKey: request.IdempotencyKey, AcceptedAt: now(),
	}, nil
}

func (agent *scriptedAgent) Observe(_ context.Context, execution goal.ExecutionRef) (ports.AgentObservation, error) {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	agent.observationCalls++
	if len(agent.observations) == 0 {
		for _, request := range agent.launchRequests {
			if request.ExecutionRef != execution || request.ArtifactMediaType != review.AssessmentMediaType {
				continue
			}
			role := review.RolePrimary
			if strings.Contains(request.Objective, `"role":"adversarial"`) {
				role = review.RoleAdversarial
			}
			payload, err := json.Marshal(review.Artifact{SchemaVersion: 1,
				SubjectDigest: reviewSubjectDigestFromPrompt(request.Objective), Role: role,
				Verdict: review.VerdictApprove, Summary: "exact subject approved", Findings: []review.Finding{}})
			if err != nil {
				return ports.AgentObservation{}, err
			}
			return ports.AgentObservation{ExecutionRef: execution, SpecHash: agent.launchSpecHashes[execution],
				Status: ports.AgentCompleted, MediaType: review.AssessmentMediaType, Content: payload,
				Usage: unknownUsage(), ObservedAt: agent.now()}, nil
		}
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
	if observation.Usage == (governance.ResourceUsage{}) {
		observation.Usage = unknownUsage()
	}
	return observation, nil
}

func reviewSubjectDigestFromPrompt(objective string) string {
	const evidencePrefix = "Review exact immutable evidence "
	if strings.HasPrefix(objective, evidencePrefix) {
		value := strings.TrimPrefix(objective, evidencePrefix)
		if end := strings.Index(value, ". Inspect with "); end >= 0 {
			var evidence struct {
				SubjectDigest string `json:"subject_digest"`
			}
			if json.Unmarshal([]byte(value[:end]), &evidence) == nil {
				return evidence.SubjectDigest
			}
		}
	}
	const prefix = "Review the exact immutable subject "
	value := strings.TrimPrefix(objective, prefix)
	if end := strings.IndexByte(value, ' '); end >= 0 {
		return value[:end]
	}
	return value
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
		Launcher: agent, Observer: agent, Controller: agent, Artifacts: artifacts,
		WorkspaceManager: &scriptedWorkspaceManager{}, VersionControl: &scriptedVersionControl{},
		TestAttestor: &scriptedTestAttestor{}, TestAttestationPolicy: TestAttestationPolicy{
			Ref: "test-attestation-policy:default", Digest: testDigest("test-attestation-policy:default"),
		},
		Clock: clock, IDs: &sequentialIDs{}, MaxOutputBytes: 1 << 20,
		MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts:    3, ClaimLease: time.Minute, DirectorLeaseDuration: time.Minute,
		MaxChildrenPerParent: 6, EffectApprovalTTL: time.Hour, BudgetPolicy: testBudgetPolicy(clock.Now()),
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: capabilities,
	})
	if err != nil {
		t.Fatalf("new orchestrator: %v", err)
	}
	return orchestrator, artifacts
}

func testBudgetPolicy(at time.Time) BudgetPolicy {
	policyHash := effectAdmissionFingerprint("test-budget-policy")
	limit := governance.ResourceVector{
		Tokens: 1_000_000, MoneyMicros: 1_000_000_000, Currency: "EUR",
		ActiveTimeNS: int64(24 * time.Hour), ProcessSlots: 70, DiskBytes: 1 << 30,
	}
	envelope := func(ref, subject string, scope governance.BudgetScope) governance.BudgetEnvelope {
		return governance.BudgetEnvelope{
			Ref: ref, SubjectRef: subject, Scope: scope, Limit: limit,
			Revision: 1, PolicyHash: policyHash, CreatedAt: at.UTC(),
		}
	}
	return BudgetPolicy{
		DeploymentEnvelope: envelope(
			"budget-envelope:deployment:test:"+policyHash, "deployment:test", governance.BudgetScopeDeployment,
		),
		ProjectEnvelopeTemplate: envelope(
			"budget-envelope:project:template:"+policyHash, "project:template", governance.BudgetScopeProject,
		),
		GoalEnvelopeTemplate: envelope(
			"budget-envelope:goal:template:"+policyHash, "goal:template", governance.BudgetScopeGoal,
		),
		DefaultWorkItemDemand: governance.ResourceVector{
			Tokens: 100, MoneyMicros: 1_000, Currency: "EUR", ActiveTimeNS: int64(time.Minute),
			ProcessSlots: 1, DiskBytes: 1 << 10,
		},
		QuotaRetryDelay: time.Second, EffectApprovalTTL: time.Hour, PolicyHash: policyHash,
	}
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
