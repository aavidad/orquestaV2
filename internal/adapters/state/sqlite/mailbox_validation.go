package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func requireMailboxLineage(
	ctx context.Context,
	source queryer,
	state application.AdmitMailboxState,
) (int64, int64, error) {
	var parentState, parentExecution, childState, childParent, childExecution string
	var childHandoffRequired int
	var parentRevision, childRevision int64
	if err := source.QueryRowContext(ctx, `
SELECT p.state, p.revision, p.execution_ref,
       c.state, c.revision, c.parent_ref, c.execution_ref, c.handoff_required
FROM work_items p
JOIN work_items c ON c.goal_ref = p.goal_ref AND c.parent_ref = p.ref
WHERE p.goal_ref = ? AND p.ref = ? AND c.ref = ?`,
		state.Envelope.GoalRef.String(), state.Envelope.ParentWorkItemRef.String(),
		state.Envelope.ChildWorkItemRef.String(),
	).Scan(
		&parentState, &parentRevision, &parentExecution,
		&childState, &childRevision, &childParent, &childExecution, &childHandoffRequired,
	); err != nil {
		return 0, 0, mapDatabaseError(err)
	}
	if parentExecution != state.Envelope.Recipient.ExecutionRef.String() ||
		childParent != state.Envelope.ParentWorkItemRef.String() ||
		parentState != string(goal.WorkItemStateRunning) ||
		childState != string(goal.WorkItemStateSucceeded) ||
		childExecution != state.Envelope.Source.ExecutionRef.String() || childHandoffRequired != 1 ||
		parentRevision <= 0 || childRevision <= 0 {
		return 0, 0, conflict(errors.New("sqlite.mailbox_lineage_conflict"))
	}
	return parentRevision, childRevision, nil
}

func requireMailboxArtifacts(ctx context.Context, source queryer, envelope application.MailboxEnvelope) error {
	for _, artifactRef := range envelope.ArtifactRefs {
		var count int
		if err := source.QueryRowContext(ctx, `
SELECT COUNT(*) FROM artifacts WHERE ref = ? AND goal_ref = ? AND work_item_ref = ?`,
			artifactRef.String(), envelope.GoalRef.String(), envelope.ChildWorkItemRef.String(),
		).Scan(&count); err != nil {
			return mapDatabaseError(err)
		}
		if count != 1 {
			return conflict(errors.New("sqlite.mailbox_artifact_scope_conflict"))
		}
	}
	return nil
}

func requireMailboxRecipient(
	record application.MailboxRecord,
	projectRef goal.ProjectRef,
	goalRef goal.GoalRef,
	principalRef identity.PrincipalRef,
	executionRef goal.ExecutionRef,
) error {
	if record.Envelope.ProjectRef != projectRef || record.Envelope.GoalRef != goalRef ||
		record.Envelope.Recipient.PrincipalRef != principalRef ||
		record.Envelope.Recipient.ExecutionRef != executionRef {
		return stateError(application.StateNotFound, errors.New("sqlite.mailbox_recipient_not_found"))
	}
	return nil
}

func requireMailboxCurrentRecipient(
	ctx context.Context,
	source queryer,
	record application.MailboxRecord,
) error {
	var projectValue, goalState, itemState string
	var planGeneration, itemRevision int64
	var executionValue sql.NullString
	if err := source.QueryRowContext(ctx, `
SELECT g.project_ref, g.state, g.plan_generation,
       wi.state, wi.revision, wi.execution_ref
FROM goals g
JOIN work_items wi ON wi.goal_ref = g.ref
WHERE g.ref = ? AND wi.ref = ?`,
		record.Envelope.GoalRef.String(), record.Envelope.ParentWorkItemRef.String(),
	).Scan(
		&projectValue, &goalState, &planGeneration,
		&itemState, &itemRevision, &executionValue,
	); err != nil {
		return mapDatabaseError(err)
	}
	if projectValue != record.Envelope.ProjectRef.String() ||
		goalState != string(goal.GoalStateRunning) ||
		planGeneration < int64(record.Envelope.TargetPlanGeneration) ||
		itemState != string(goal.WorkItemStateRunning) ||
		itemRevision != int64(record.Action.WorkItemGeneration) || !executionValue.Valid ||
		executionValue.String != record.Envelope.Recipient.ExecutionRef.String() {
		return conflict(errors.New("sqlite.mailbox_recipient_obsolete"))
	}
	return nil
}

func validateMailboxReplayRequest(request application.MailboxReplayRequest) error {
	if !validText(request.RequestRef) || !validText(request.RequestFingerprint) ||
		request.PrincipalRef.String() == "" || request.ProjectRef.String() == "" ||
		request.GoalRef.String() == "" {
		return errors.New("sqlite.mailbox_replay_invalid")
	}
	switch request.Kind {
	case application.MailboxMutationAdmit:
		if request.MessageRef.String() != "" {
			return errors.New("sqlite.mailbox_replay_message_unexpected")
		}
	case application.MailboxMutationClaim, application.MailboxMutationDeliver,
		application.MailboxMutationConsume, application.MailboxMutationAcknowledge,
		application.MailboxMutationBlock:
		if request.MessageRef.String() == "" {
			return errors.New("sqlite.mailbox_replay_message_missing")
		}
	default:
		return errors.New("sqlite.mailbox_replay_kind_invalid")
	}
	return nil
}

func validateAdmitMailboxState(state application.AdmitMailboxState) error {
	envelope := state.Envelope
	if !validText(state.RequestRef) || !validText(state.RequestFingerprint) ||
		state.AuthorizationReceipt.Ref() == "" || envelope.Ref.String() == "" ||
		envelope.RequestRef != state.RequestRef || envelope.RequestFingerprint != state.RequestFingerprint ||
		envelope.ProjectRef.String() == "" || envelope.GoalRef.String() == "" ||
		envelope.TargetPlanGeneration == 0 || envelope.ParentWorkItemRef.String() == "" ||
		envelope.ChildWorkItemRef.String() == "" || envelope.ParentWorkItemRef == envelope.ChildWorkItemRef ||
		envelope.Source.PrincipalRef.String() == "" || envelope.Source.WorkItemRef != envelope.ChildWorkItemRef ||
		envelope.Source.ExecutionRef.String() == "" || envelope.Recipient.PrincipalRef.String() == "" ||
		envelope.Recipient.WorkItemRef != envelope.ParentWorkItemRef || envelope.Recipient.ExecutionRef.String() == "" ||
		!validText(envelope.Summary) || !validMailboxContentHash(envelope.ContentHash) ||
		envelope.ContentHash != application.MailboxEnvelopeContentHash(envelope) || envelope.AdmittedAt.IsZero() ||
		!validMailboxKind(envelope.Kind) || state.Admission.Ref == "" ||
		state.Admission.MessageRef != envelope.Ref || state.Admission.RequestRef != state.RequestRef ||
		state.Admission.RequestFingerprint != state.RequestFingerprint ||
		state.Admission.PrincipalRef != envelope.Source.PrincipalRef ||
		!state.Admission.AdmittedAt.Equal(envelope.AdmittedAt) ||
		state.Action.Kind != application.ActionDeliverMailbox ||
		state.Action.Ref != "action:mailbox:"+envelope.Ref.String() ||
		state.Action.GoalRef != envelope.GoalRef || state.Action.WorkItemRef != envelope.ParentWorkItemRef ||
		state.Action.ExecutionRef != envelope.Recipient.ExecutionRef ||
		state.Action.PlanGeneration != envelope.TargetPlanGeneration ||
		state.Action.WorkItemGeneration == 0 || !state.Action.AvailableAt.Equal(envelope.AdmittedAt) ||
		state.Event.GoalRef != envelope.GoalRef || state.Event.WorkItemRef != envelope.ParentWorkItemRef ||
		state.Event.ExecutionRef != envelope.Recipient.ExecutionRef || state.Event.Kind != "mailbox.admitted" ||
		state.OperationAt.IsZero() || !state.OperationAt.Equal(envelope.AdmittedAt) {
		return errors.New("sqlite.mailbox_admit_invalid")
	}
	request := state.AuthorizationReceipt.Decision().Request()
	if request.Principal().Ref != envelope.Source.PrincipalRef || request.ProjectRef() != envelope.ProjectRef ||
		request.Permission() != identity.PermissionGoalsDirect || request.ResourceRef() != envelope.GoalRef.String() {
		return errors.New("sqlite.mailbox_admit_authorization_invalid")
	}
	for _, artifactRef := range envelope.ArtifactRefs {
		if artifactRef.String() == "" {
			return errors.New("sqlite.mailbox_artifact_invalid")
		}
	}
	return validateEvent(state.Event)
}

func validateClaimMailboxState(state application.ClaimMailboxState) error {
	if !validText(state.RequestRef) || !validText(state.RequestFingerprint) ||
		state.AuthorizationReceipt.Ref() == "" || state.PrincipalRef.String() == "" ||
		state.ProjectRef.String() == "" || state.GoalRef.String() == "" ||
		state.MessageRef.String() == "" || state.RecipientExecutionRef.String() == "" ||
		!validText(state.Token) || state.LeaseDuration <= 0 || state.RequestedAt.IsZero() {
		return errors.New("sqlite.mailbox_claim_invalid")
	}
	return validateMailboxAuthorization(
		state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef, state.MessageRef,
	)
}

func validateMarkMailboxDeliveredState(state application.MarkMailboxDeliveredState) error {
	if !validText(state.RequestRef) || !validText(state.RequestFingerprint) ||
		state.AuthorizationReceipt.Ref() == "" || state.PrincipalRef.String() == "" ||
		state.ProjectRef.String() == "" || state.GoalRef.String() == "" ||
		state.MessageRef.String() == "" || state.RecipientExecutionRef.String() == "" ||
		!validText(state.ClaimToken) || state.Fence == 0 || state.Fence > maxSQLiteInteger ||
		!validText(state.DeliveryRef) || state.OperationAt.IsZero() {
		return errors.New("sqlite.mailbox_delivery_invalid")
	}
	return validateMailboxAuthorization(
		state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef, state.MessageRef,
	)
}

func validateConsumeMailboxState(state application.ConsumeMailboxState) error {
	if !validText(state.RequestRef) || !validText(state.RequestFingerprint) ||
		state.AuthorizationReceipt.Ref() == "" || state.PrincipalRef.String() == "" ||
		state.ProjectRef.String() == "" || state.GoalRef.String() == "" ||
		state.MessageRef.String() == "" || state.RecipientExecutionRef.String() == "" ||
		!validText(state.ClaimToken) || state.Fence == 0 || state.Fence > maxSQLiteInteger ||
		!validText(state.ConsumptionRef) || state.OperationAt.IsZero() {
		return errors.New("sqlite.mailbox_consumption_invalid")
	}
	if err := validateMailboxAuthorization(
		state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef, state.MessageRef,
	); err != nil {
		return err
	}
	return validateConsumptionReceipt(state.ConsumptionReceipt)
}

func validateResolveMailboxState(state application.ResolveMailboxState, outcome application.MailboxOutcome) error {
	ack := state.Acknowledgement
	if !validText(state.RequestRef) || !validText(state.RequestFingerprint) ||
		state.AuthorizationReceipt.Ref() == "" || state.PrincipalRef.String() == "" ||
		state.ProjectRef.String() == "" || state.GoalRef.String() == "" || state.MessageRef.String() == "" ||
		state.RecipientExecutionRef.String() == "" || !validText(state.ClaimToken) ||
		state.Fence == 0 || state.Fence > maxSQLiteInteger || state.ExpectedGoalRevision == 0 ||
		state.ExpectedPlanGeneration == 0 || state.OperationAt.IsZero() ||
		ack.Ref == "" || ack.MessageRef != state.MessageRef || ack.RequestRef != state.RequestRef ||
		ack.RequestFingerprint != state.RequestFingerprint || ack.ProjectRef != state.ProjectRef ||
		ack.GoalRef != state.GoalRef || ack.TargetPlanGeneration == 0 ||
		ack.TargetPlanGeneration > state.ExpectedPlanGeneration ||
		ack.Recipient.PrincipalRef != state.PrincipalRef || ack.Recipient.ExecutionRef != state.RecipientExecutionRef ||
		ack.Fence != state.Fence ||
		ack.Outcome != outcome || !validText(ack.EffectOrReworkRef) || ack.AcknowledgedAt.IsZero() ||
		!ack.AcknowledgedAt.Equal(state.OperationAt) || len(state.Events) == 0 {
		return errors.New("sqlite.mailbox_resolution_invalid")
	}
	if err := validateMailboxAuthorization(
		state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef, state.MessageRef,
	); err != nil {
		return err
	}
	if _, err := goal.RestoreGoal(state.Goal.Snapshot()); err != nil {
		return err
	}
	resolutionEvent := false
	for _, event := range state.Events {
		if err := validateEvent(event); err != nil || event.GoalRef != state.GoalRef ||
			!event.OccurredAt.Equal(state.OperationAt) {
			return errors.New("sqlite.mailbox_resolution_event_invalid")
		}
		if event.Kind == "mailbox."+string(outcome) &&
			event.WorkItemRef == ack.ParentWorkItemRef &&
			event.ExecutionRef == state.RecipientExecutionRef {
			resolutionEvent = true
		}
	}
	if !resolutionEvent {
		return errors.New("sqlite.mailbox_resolution_event_missing")
	}
	return nil
}

func validateMailboxAuthorization(
	receipt identity.AuthorizationReceipt,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	messageRef application.MailboxMessageRef,
) error {
	decision := receipt.Decision()
	request := decision.Request()
	if decision.Outcome() != identity.AuthorizationAllowed ||
		request.Principal().Ref != principalRef || request.ProjectRef() != projectRef ||
		request.Permission() != identity.PermissionGoalsGet || request.ResourceRef() != messageRef.String() {
		return errors.New("sqlite.mailbox_authorization_invalid")
	}
	return nil
}

func validateMailboxConsumptionReceipt(
	record application.MailboxRecord,
	advance mailboxAdvance,
	now time.Time,
) error {
	receipt := advance.consumptionReceipt
	if err := validateConsumptionReceipt(receipt); err != nil {
		return err
	}
	if receipt.ActionRef != record.Action.Ref || receipt.Kind != application.ActionDeliverMailbox ||
		receipt.GoalRef != advance.goalRef || receipt.WorkItemRef != record.Envelope.ParentWorkItemRef ||
		receipt.ExecutionRef != advance.executionRef || receipt.MailboxMessageRef != advance.messageRef ||
		receipt.PlanGeneration != record.Action.PlanGeneration ||
		receipt.WorkItemGeneration != record.Action.WorkItemGeneration || receipt.Fence != advance.fence ||
		receipt.DeliveryAttempt != advance.fence || receipt.ClaimToken != advance.claimToken ||
		receipt.WorkerRef != advance.principalRef.String() || receipt.Outcome != application.ActionConsumedCompleted ||
		receipt.ErrorCode != "" || receipt.ConsumedAt.IsZero() || now.Before(receipt.ConsumedAt) {
		return errors.New("sqlite.mailbox_consumption_receipt_conflict")
	}
	return nil
}

func validateMailboxResolutionTransition(
	current goal.Goal,
	state application.ResolveMailboxState,
	outcome application.MailboxOutcome,
) error {
	ack := state.Acknowledgement
	expected, err := application.BuildMailboxResolutionGoal(application.MailboxResolutionGoalInput{
		Current: current, ExpectedRevision: state.ExpectedGoalRevision,
		ParentWorkItemRef: ack.ParentWorkItemRef, ChildWorkItemRef: ack.ChildWorkItemRef,
		MessageRef: state.MessageRef, Outcome: outcome, ReceiptRef: ack.Ref,
		ResolvedAt: state.OperationAt,
	})
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(expected.Snapshot(), state.Goal.Snapshot()) {
		return errors.New("sqlite.mailbox_goal_handoff_conflict")
	}
	return nil
}

func mailboxReplayRequestFromClaim(state application.ClaimMailboxState) application.MailboxReplayRequest {
	return application.MailboxReplayRequest{
		Kind: application.MailboxMutationClaim, RequestRef: state.RequestRef,
		RequestFingerprint: state.RequestFingerprint, PrincipalRef: state.PrincipalRef,
		ProjectRef: state.ProjectRef, GoalRef: state.GoalRef, MessageRef: state.MessageRef,
	}
}

func validMailboxKind(kind application.MailboxKind) bool {
	return kind == application.MailboxKindChildDelivery
}

// validateRecoveryV13Mailbox crosses every mailbox table through the same
// canonical reader used at runtime. FKs/triggers then own row-level binding.
