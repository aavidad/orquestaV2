package application

import (
	"context"
	"errors"
	"strings"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func (orchestrator *Orchestrator) AdmitMailbox(
	ctx context.Context,
	access Access,
	request AdmitMailboxRequest,
) (MailboxAdmissionResult, error) {
	if orchestrator == nil {
		return MailboxAdmissionResult{}, errors.New("application.unavailable")
	}
	request.Summary = strings.TrimSpace(request.Summary)
	request.ArtifactRefs = canonicalMailboxArtifactRefs(request.ArtifactRefs)
	if err := validateAdmitMailboxRequest(request); err != nil {
		return MailboxAdmissionResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return MailboxAdmissionResult{}, err
	}
	if _, err := access.authenticatedExecution(request.SourceExecutionRef); err != nil {
		return MailboxAdmissionResult{}, err
	}
	if mailboxEnvelopeInputBytes(principal.Ref, projectRef, request) > orchestrator.maxMailboxEnvelopeBytes {
		return MailboxAdmissionResult{}, errors.New("application.mailbox_envelope_too_large")
	}
	fingerprint := admitMailboxFingerprint(principal.Ref, projectRef, request)
	if err := orchestrator.requireCurrentMailboxAccess(
		ctx, principal.Ref, projectRef, identity.PermissionGoalsDirect,
	); err != nil {
		return MailboxAdmissionResult{}, err
	}
	if err := orchestrator.requireCurrentMailboxAccess(
		ctx, request.RecipientPrincipalRef, projectRef, identity.PermissionGoalsGet,
	); err != nil {
		return MailboxAdmissionResult{}, err
	}
	replay, found, err := orchestrator.state.MailboxReplay(ctx, MailboxReplayRequest{
		Kind: MailboxMutationAdmit, RequestRef: request.RequestRef,
		RequestFingerprint: fingerprint, PrincipalRef: principal.Ref,
		ProjectRef: projectRef, GoalRef: request.GoalRef,
	})
	if err != nil {
		return MailboxAdmissionResult{}, err
	}
	if found {
		if err := validateAdmittedMailboxRecord(
			replay.Record, request, principal.Ref, projectRef, fingerprint, true,
		); err != nil {
			return MailboxAdmissionResult{}, err
		}
		return MailboxAdmissionResult{Record: cloneMailboxRecord(replay.Record)}, nil
	}
	record, err := orchestrator.state.GetGoal(ctx, request.GoalRef)
	if err != nil {
		return MailboxAdmissionResult{}, err
	}
	if record.Goal.Project() != projectRef {
		return MailboxAdmissionResult{}, &StateError{Code: StateNotFound}
	}
	if record.Goal.State() != goal.GoalStateRunning ||
		record.Goal.PlanGeneration() != request.ExpectedPlanGeneration {
		return MailboxAdmissionResult{}, &StateError{Code: StateConflict}
	}
	parent, child, err := mailboxWorkItems(record.Goal, request)
	if err != nil {
		return MailboxAdmissionResult{}, err
	}
	if err := validateMailboxArtifacts(child, request.ArtifactRefs); err != nil {
		return MailboxAdmissionResult{}, err
	}
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx, access, identity.PermissionGoalsDirect, request.GoalRef.String(),
		orchestrator.clock.Now().UTC(), mailboxAuthorizationRequestRef(MailboxMutationAdmit, request.RequestRef, fingerprint),
	)
	if err != nil {
		return MailboxAdmissionResult{}, err
	}
	now := orchestrator.clock.Now().UTC()
	messageID, err := orchestrator.ids.NewID(ctx, "mailbox-message")
	if err != nil {
		return MailboxAdmissionResult{}, err
	}
	messageRef, err := NewMailboxMessageRef(messageID)
	if err != nil {
		return MailboxAdmissionResult{}, err
	}
	admissionRef, err := orchestrator.ids.NewID(ctx, "mailbox-admission")
	if err != nil {
		return MailboxAdmissionResult{}, err
	}
	envelope := MailboxEnvelope{
		Ref: messageRef, RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		ProjectRef: projectRef, GoalRef: request.GoalRef,
		TargetPlanGeneration: request.ExpectedPlanGeneration, Kind: request.Kind,
		ParentWorkItemRef: request.ParentWorkItemRef, ChildWorkItemRef: request.ChildWorkItemRef,
		Source: MailboxEndpoint{
			PrincipalRef: principal.Ref, WorkItemRef: request.ChildWorkItemRef,
			ExecutionRef: request.SourceExecutionRef,
		},
		Recipient: MailboxEndpoint{
			PrincipalRef: request.RecipientPrincipalRef, WorkItemRef: request.ParentWorkItemRef,
			ExecutionRef: request.RecipientExecutionRef,
		},
		Summary: request.Summary, ArtifactRefs: append([]goal.ArtifactRef(nil), request.ArtifactRefs...),
		AdmittedAt: now,
	}
	envelope.ContentHash = mailboxEnvelopeContentHash(envelope)
	admission := MailboxAdmissionReceipt{
		Ref: admissionRef, MessageRef: messageRef, RequestRef: request.RequestRef,
		RequestFingerprint: fingerprint, PrincipalRef: principal.Ref,
		AuthorizationReceipt: authorization, AdmittedAt: now,
	}
	action := ActionRecord{
		Ref: "action:mailbox:" + messageRef.String(), Kind: ActionDeliverMailbox,
		GoalRef: request.GoalRef, WorkItemRef: request.ParentWorkItemRef,
		ExecutionRef:   request.RecipientExecutionRef,
		PlanGeneration: request.ExpectedPlanGeneration, WorkItemGeneration: parent.Revision(),
		AvailableAt: now,
	}
	persisted, created, err := orchestrator.state.AdmitMailbox(ctx, AdmitMailboxState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		AuthorizationReceipt: authorization, Envelope: envelope, Admission: admission,
		Action: action, Event: EventRecord{
			Ref: "event:mailbox-admitted:" + messageRef.String(), Kind: "mailbox.admitted",
			GoalRef: request.GoalRef, WorkItemRef: request.ParentWorkItemRef,
			ExecutionRef: request.RecipientExecutionRef, OccurredAt: now,
		}, OperationAt: now,
	})
	if err != nil {
		return MailboxAdmissionResult{}, err
	}
	if err := validateAdmittedMailboxRecord(
		persisted, request, principal.Ref, projectRef, fingerprint, false,
	); err != nil {
		return MailboxAdmissionResult{}, err
	}
	return MailboxAdmissionResult{Record: cloneMailboxRecord(persisted), Created: created}, nil
}

func mailboxWorkItems(aggregate goal.Goal, request AdmitMailboxRequest) (goal.WorkItem, goal.WorkItem, error) {
	parent, parentFound := aggregate.WorkItem(request.ParentWorkItemRef)
	child, childFound := aggregate.WorkItem(request.ChildWorkItemRef)
	if !parentFound || !childFound {
		return goal.WorkItem{}, goal.WorkItem{}, &StateError{Code: StateNotFound}
	}
	childParent, hasParent := child.Parent()
	childExecution, hasChildExecution := child.Execution()
	parentExecution, hasParentExecution := parent.Execution()
	parentAddressable := parent.State() == goal.WorkItemStateRunning
	if !hasParent || childParent != parent.Ref() || !child.HandoffRequired() ||
		child.State() != goal.WorkItemStateSucceeded ||
		!hasChildExecution || childExecution != request.SourceExecutionRef ||
		!parentAddressable || !hasParentExecution ||
		parentExecution != request.RecipientExecutionRef {
		return goal.WorkItem{}, goal.WorkItem{}, &StateError{Code: StateConflict}
	}
	return parent, child, nil
}

func validateMailboxArtifacts(child goal.WorkItem, requested []goal.ArtifactRef) error {
	available := make(map[goal.ArtifactRef]struct{}, len(child.Artifacts()))
	for _, ref := range child.Artifacts() {
		available[ref] = struct{}{}
	}
	for _, ref := range requested {
		if _, ok := available[ref]; !ok {
			return &StateError{Code: StateConflict}
		}
	}
	return nil
}
