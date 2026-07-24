package application

import (
	"context"
	"errors"
	"sort"
	"strings"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func (orchestrator *Orchestrator) prepareMailboxMutation(
	ctx context.Context,
	access Access,
	kind MailboxMutationKind,
	requestRef string,
	fingerprint string,
	goalRef goal.GoalRef,
	messageRef MailboxMessageRef,
) (*MailboxReplayRecord, identity.AuthorizationReceipt, error) {
	principal, projectRef, err := access.values()
	if err != nil {
		return nil, identity.AuthorizationReceipt{}, err
	}
	if err := orchestrator.requireCurrentMailboxAccess(
		ctx, access, principal.Ref, projectRef, identity.PermissionGoalsGet,
	); err != nil {
		return nil, identity.AuthorizationReceipt{}, err
	}
	replay, found, err := orchestrator.state.MailboxReplay(ctx, MailboxReplayRequest{
		Kind: kind, RequestRef: requestRef, RequestFingerprint: fingerprint,
		PrincipalRef: principal.Ref, ProjectRef: projectRef, GoalRef: goalRef,
		MessageRef: messageRef,
	})
	if err != nil {
		return nil, identity.AuthorizationReceipt{}, err
	}
	if found {
		return &replay, identity.AuthorizationReceipt{}, nil
	}
	authorization, err := orchestrator.mailboxMutationAuthorization(
		ctx, access, kind, requestRef, fingerprint, messageRef,
	)
	return nil, authorization, err
}

func (orchestrator *Orchestrator) mailboxMutationAuthorization(
	ctx context.Context,
	access Access,
	kind MailboxMutationKind,
	requestRef string,
	fingerprint string,
	messageRef MailboxMessageRef,
) (identity.AuthorizationReceipt, error) {
	return orchestrator.authorizeIdempotentWithRequestRef(
		ctx, access, identity.PermissionGoalsGet, messageRef.String(),
		orchestrator.clock.Now().UTC(), mailboxAuthorizationRequestRef(kind, requestRef, fingerprint),
	)
}

func (orchestrator *Orchestrator) requireCurrentMailboxAccess(
	ctx context.Context,
	access Access,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	permission identity.Permission,
) error {
	if access.executionServiceBound() {
		principal, boundProject, err := access.values()
		if err != nil || principal.Ref != principalRef || boundProject != projectRef {
			return errForbidden
		}
		return nil
	}
	membership, err := orchestrator.access.Membership(ctx, principalRef, projectRef)
	if err != nil {
		if IsStateError(err, StateNotFound) {
			return errForbidden
		}
		return err
	}
	if membership.PrincipalRef() != principalRef || membership.ProjectRef() != projectRef ||
		!membership.IsActive() || !identity.RoleAllows(membership.Role(), permission) {
		return errForbidden
	}
	return nil
}

func validateAdmitMailboxRequest(request AdmitMailboxRequest) error {
	if !validMailboxText(request.RequestRef) || request.GoalRef.String() == "" ||
		request.ExpectedPlanGeneration == 0 || !validMailboxKind(request.Kind) ||
		request.ParentWorkItemRef.String() == "" || request.ChildWorkItemRef.String() == "" ||
		request.ParentWorkItemRef == request.ChildWorkItemRef || request.SourceExecutionRef.String() == "" ||
		request.RecipientPrincipalRef.String() == "" || request.RecipientExecutionRef.String() == "" ||
		!validMailboxText(request.Summary) {
		return errors.New("application.mailbox_admission_invalid")
	}
	for _, ref := range request.ArtifactRefs {
		if ref.String() == "" {
			return errors.New("application.mailbox_artifact_ref_invalid")
		}
	}
	return nil
}

func validateMailboxAddressedRequest(
	requestRef string,
	goalRef goal.GoalRef,
	messageRef MailboxMessageRef,
	workItemRef goal.WorkItemRef,
	executionRef goal.ExecutionRef,
) error {
	if !validMailboxText(requestRef) || goalRef.String() == "" || messageRef.String() == "" ||
		workItemRef.String() == "" || executionRef.String() == "" {
		return errors.New("application.mailbox_request_invalid")
	}
	return nil
}

func validateMailboxClaimMutation(
	requestRef string,
	goalRef goal.GoalRef,
	messageRef MailboxMessageRef,
	workItemRef goal.WorkItemRef,
	executionRef goal.ExecutionRef,
	claimToken string,
	fence uint64,
) error {
	if err := validateMailboxAddressedRequest(
		requestRef, goalRef, messageRef, workItemRef, executionRef,
	); err != nil {
		return err
	}
	if !validMailboxText(claimToken) || fence == 0 {
		return errors.New("application.mailbox_claim_invalid")
	}
	return nil
}

func validateAdmittedMailboxRecord(
	record MailboxRecord,
	request AdmitMailboxRequest,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	fingerprint string,
	allowAdvanced bool,
) error {
	envelope := record.Envelope
	if envelope.Ref.String() == "" || envelope.RequestRef != request.RequestRef ||
		envelope.RequestFingerprint != fingerprint || envelope.ProjectRef != projectRef ||
		envelope.GoalRef != request.GoalRef || envelope.TargetPlanGeneration != request.ExpectedPlanGeneration ||
		envelope.Kind != request.Kind || envelope.ParentWorkItemRef != request.ParentWorkItemRef ||
		envelope.ChildWorkItemRef != request.ChildWorkItemRef ||
		envelope.Source != (MailboxEndpoint{PrincipalRef: principalRef, WorkItemRef: request.ChildWorkItemRef, ExecutionRef: request.SourceExecutionRef}) ||
		envelope.Recipient != (MailboxEndpoint{PrincipalRef: request.RecipientPrincipalRef, WorkItemRef: request.ParentWorkItemRef, ExecutionRef: request.RecipientExecutionRef}) ||
		envelope.Summary != request.Summary || !equalMailboxArtifactRefs(envelope.ArtifactRefs, request.ArtifactRefs) ||
		envelope.ContentHash != mailboxEnvelopeContentHash(envelope) || envelope.AdmittedAt.IsZero() ||
		record.Admission.Ref == "" || record.Admission.MessageRef != envelope.Ref ||
		record.Admission.RequestRef != request.RequestRef || record.Admission.RequestFingerprint != fingerprint ||
		record.Admission.PrincipalRef != principalRef || !record.Admission.AdmittedAt.Equal(envelope.AdmittedAt) ||
		record.Action.Ref != "action:mailbox:"+envelope.Ref.String() ||
		record.Action.Kind != ActionDeliverMailbox || record.Action.GoalRef != request.GoalRef ||
		record.Action.WorkItemRef != request.ParentWorkItemRef ||
		record.Action.ExecutionRef != request.RecipientExecutionRef ||
		record.Action.PlanGeneration != request.ExpectedPlanGeneration ||
		record.Action.WorkItemGeneration == 0 || record.Action.AvailableAt != envelope.AdmittedAt {
		return &StateError{Code: StateConflict}
	}
	if (!allowAdvanced && (record.State != MailboxStateAdmitted || len(record.Attempts) != 0 || record.Acknowledgement != nil)) ||
		(allowAdvanced && !validMailboxState(record.State)) || validateMailboxTerminalFacts(record) != nil {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func validateMailboxClaim(
	claim MailboxClaim,
	requestRef string,
	messageRef MailboxMessageRef,
	endpoint MailboxEndpoint,
	expectedToken string,
) error {
	if err := validateMailboxRecordAddress(
		claim.Record, claim.Record.Envelope.ProjectRef, claim.Record.Envelope.GoalRef,
		messageRef, endpoint,
	); err != nil {
		return err
	}
	attempt := claim.Attempt
	if !mailboxStateReached(claim.Record.State, MailboxStateClaimed) || len(claim.Record.Attempts) == 0 ||
		attempt.MessageRef != messageRef || attempt.Recipient != endpoint ||
		!validMailboxText(attempt.ActionRef) || attempt.ClaimRequestRef != requestRef ||
		!validMailboxText(attempt.ClaimToken) || attempt.Fence == 0 ||
		attempt.ExpectedGoalRevision == 0 ||
		attempt.ExpectedPlanGeneration < claim.Record.Envelope.TargetPlanGeneration ||
		attempt.ClaimedAt.IsZero() || !attempt.LeaseUntil.After(attempt.ClaimedAt) {
		return &StateError{Code: StateConflict}
	}
	found := false
	for _, persisted := range claim.Record.Attempts {
		if persisted == attempt {
			found = true
			break
		}
	}
	if !found || (expectedToken != "" &&
		(attempt.ClaimToken != expectedToken || claim.Record.State != MailboxStateClaimed ||
			claim.Record.Attempts[len(claim.Record.Attempts)-1] != attempt)) {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func validateMailboxMutationRecord(
	record MailboxRecord,
	messageRef MailboxMessageRef,
	principalRef identity.PrincipalRef,
	workItemRef goal.WorkItemRef,
	executionRef goal.ExecutionRef,
	claimToken string,
	fence uint64,
	wantState MailboxState,
	allowAdvanced bool,
) error {
	endpoint := MailboxEndpoint{PrincipalRef: principalRef, WorkItemRef: workItemRef, ExecutionRef: executionRef}
	if err := validateMailboxRecordAddress(
		record, record.Envelope.ProjectRef, record.Envelope.GoalRef, messageRef, endpoint,
	); err != nil {
		return err
	}
	if ((!allowAdvanced && record.State != wantState) ||
		(allowAdvanced && !mailboxStateReached(record.State, wantState))) || len(record.Attempts) == 0 {
		return &StateError{Code: StateConflict}
	}
	attempt := record.Attempts[len(record.Attempts)-1]
	if attempt.Recipient != endpoint || attempt.ClaimToken != claimToken ||
		attempt.Fence != fence {
		return &StateError{Code: StateConflict}
	}
	if wantState == MailboxStateDelivered && (attempt.DeliveryRef == "" || attempt.DeliveredAt.IsZero()) {
		return &StateError{Code: StateConflict}
	}
	if wantState == MailboxStateConsumed &&
		(attempt.DeliveryRef == "" || attempt.DeliveredAt.IsZero() ||
			attempt.ConsumptionRef == "" || attempt.ConsumedAt.IsZero()) {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func mailboxStateReached(got, want MailboxState) bool {
	if got == want {
		return true
	}
	switch want {
	case MailboxStateClaimed:
		return got == MailboxStateDelivered || got == MailboxStateConsumed ||
			got == MailboxStateAcknowledged || got == MailboxStateBlocked
	case MailboxStateDelivered:
		return got == MailboxStateConsumed || got == MailboxStateAcknowledged || got == MailboxStateBlocked
	case MailboxStateConsumed:
		return got == MailboxStateAcknowledged || got == MailboxStateBlocked
	default:
		return false
	}
}

func validateMailboxAcknowledgement(
	ack MailboxAcknowledgement,
	request ResolveMailboxRequest,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	fingerprint string,
	outcome MailboxOutcome,
	targetPlanGeneration goal.PlanGeneration,
) error {
	endpoint := MailboxEndpoint{PrincipalRef: principalRef, WorkItemRef: request.RecipientWorkItemRef, ExecutionRef: request.RecipientExecutionRef}
	if ack.Ref == "" || ack.MessageRef != request.MessageRef ||
		ack.ActionRef != "action:mailbox:"+request.MessageRef.String() ||
		ack.RequestRef != request.RequestRef || ack.RequestFingerprint != fingerprint ||
		ack.ProjectRef != projectRef || ack.GoalRef != request.GoalRef ||
		ack.TargetPlanGeneration != targetPlanGeneration || ack.Recipient != endpoint ||
		ack.Fence != request.Fence ||
		ack.Outcome != outcome || ack.EffectOrReworkRef != request.EffectOrReworkRef ||
		ack.AcknowledgedAt.IsZero() {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func validateMailboxRecordAddress(
	record MailboxRecord,
	projectRef goal.ProjectRef,
	goalRef goal.GoalRef,
	messageRef MailboxMessageRef,
	endpoint MailboxEndpoint,
) error {
	if record.Envelope.Ref != messageRef || record.Envelope.ProjectRef != projectRef ||
		record.Envelope.GoalRef != goalRef || record.Envelope.Recipient != endpoint ||
		record.Envelope.ContentHash != mailboxEnvelopeContentHash(record.Envelope) ||
		record.Action.Ref != "action:mailbox:"+messageRef.String() ||
		record.Action.Kind != ActionDeliverMailbox || record.Action.GoalRef != goalRef ||
		record.Action.WorkItemRef != endpoint.WorkItemRef ||
		record.Action.ExecutionRef != endpoint.ExecutionRef ||
		record.Action.PlanGeneration != record.Envelope.TargetPlanGeneration ||
		validateMailboxTerminalFacts(record) != nil {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func validateMailboxTerminalFacts(record MailboxRecord) error {
	switch record.State {
	case MailboxStateAcknowledged:
		if record.Acknowledgement == nil || record.Retirement != nil ||
			record.Acknowledgement.Outcome != MailboxOutcomeAcknowledged {
			return errors.New("application.mailbox_terminal_fact_invalid")
		}
	case MailboxStateBlocked:
		if record.Acknowledgement == nil || record.Retirement != nil ||
			record.Acknowledgement.Outcome != MailboxOutcomeBlocked {
			return errors.New("application.mailbox_terminal_fact_invalid")
		}
	case MailboxStateRetired:
		if record.Retirement == nil || record.Acknowledgement != nil {
			return errors.New("application.mailbox_terminal_fact_invalid")
		}
		return ValidateMailboxRetirement(record)
	default:
		if record.Acknowledgement == nil && record.Retirement == nil {
			return nil
		}
		return errors.New("application.mailbox_terminal_fact_invalid")
	}
	return nil
}

func validMailboxKind(kind MailboxKind) bool {
	return kind == MailboxKindChildDelivery
}

func validMailboxText(value string) bool {
	return validApplicationRef(value) && !strings.ContainsRune(value, '\x00')
}

func validMailboxState(state MailboxState) bool {
	switch state {
	case MailboxStateAdmitted, MailboxStateClaimed, MailboxStateDelivered,
		MailboxStateConsumed, MailboxStateAcknowledged, MailboxStateBlocked,
		MailboxStateRetired:
		return true
	default:
		return false
	}
}

func canonicalMailboxArtifactRefs(refs []goal.ArtifactRef) []goal.ArtifactRef {
	byValue := make(map[string]goal.ArtifactRef, len(refs))
	for _, ref := range refs {
		byValue[ref.String()] = ref
	}
	values := make([]string, 0, len(byValue))
	for value := range byValue {
		values = append(values, value)
	}
	sort.Strings(values)
	result := make([]goal.ArtifactRef, 0, len(values))
	for _, value := range values {
		result = append(result, byValue[value])
	}
	return result
}

func equalMailboxArtifactRefs(left, right []goal.ArtifactRef) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func cloneMailboxRecord(record MailboxRecord) MailboxRecord {
	record.Envelope.ArtifactRefs = append([]goal.ArtifactRef(nil), record.Envelope.ArtifactRefs...)
	record.Attempts = append([]MailboxDeliveryAttempt(nil), record.Attempts...)
	if record.Acknowledgement != nil {
		ack := *record.Acknowledgement
		record.Acknowledgement = &ack
	}
	if record.Retirement != nil {
		retirement := *record.Retirement
		record.Retirement = &retirement
	}
	return record
}

func cloneMailboxClaim(claim MailboxClaim) MailboxClaim {
	claim.Record = cloneMailboxRecord(claim.Record)
	return claim
}
