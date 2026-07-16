package application

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

func (orchestrator *Orchestrator) AcknowledgeMailbox(
	ctx context.Context,
	access Access,
	request AcknowledgeMailboxRequest,
) (MailboxResolutionResult, error) {
	return orchestrator.resolveMailbox(ctx, access, ResolveMailboxRequest(request), MailboxOutcomeAcknowledged)
}

func (orchestrator *Orchestrator) BlockMailbox(
	ctx context.Context,
	access Access,
	request BlockMailboxRequest,
) (MailboxResolutionResult, error) {
	return orchestrator.resolveMailbox(ctx, access, ResolveMailboxRequest(request), MailboxOutcomeBlocked)
}

func (orchestrator *Orchestrator) resolveMailbox(
	ctx context.Context,
	access Access,
	request ResolveMailboxRequest,
	outcome MailboxOutcome,
) (MailboxResolutionResult, error) {
	if orchestrator == nil {
		return MailboxResolutionResult{}, errors.New("application.unavailable")
	}
	request.EffectOrReworkRef = strings.TrimSpace(request.EffectOrReworkRef)
	if err := validateMailboxClaimMutation(
		request.RequestRef, request.GoalRef, request.MessageRef,
		request.RecipientWorkItemRef, request.RecipientExecutionRef,
		request.ClaimToken, request.Fence,
	); err != nil || request.ExpectedGoalRevision == 0 || request.ExpectedPlanGeneration == 0 ||
		!validMailboxText(request.EffectOrReworkRef) {
		if err != nil {
			return MailboxResolutionResult{}, err
		}
		return MailboxResolutionResult{}, errors.New("application.mailbox_resolution_invalid")
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return MailboxResolutionResult{}, err
	}
	if _, err := access.authenticatedExecution(request.RecipientExecutionRef); err != nil {
		return MailboxResolutionResult{}, err
	}
	kind := MailboxMutationAcknowledge
	if outcome == MailboxOutcomeBlocked {
		kind = MailboxMutationBlock
	}
	fingerprint := mailboxMutationFingerprint(
		kind, principal.Ref, projectRef, request.GoalRef, request.MessageRef,
		request.RecipientWorkItemRef, request.RecipientExecutionRef,
		request.ClaimToken, request.Fence,
		strconv.FormatUint(uint64(request.ExpectedGoalRevision), 10)+"\x00"+
			strconv.FormatUint(uint64(request.ExpectedPlanGeneration), 10)+"\x00"+request.EffectOrReworkRef,
	)
	replay, authorization, err := orchestrator.prepareMailboxMutation(
		ctx, access, kind, request.RequestRef, fingerprint, request.GoalRef, request.MessageRef,
	)
	if err != nil {
		return MailboxResolutionResult{}, err
	}
	if replay != nil {
		if err := validateMailboxAcknowledgement(
			replay.Acknowledgement, request, principal.Ref, projectRef, fingerprint, outcome,
			replay.Record.Envelope.TargetPlanGeneration,
		); err != nil {
			return MailboxResolutionResult{}, err
		}
		return MailboxResolutionResult{Acknowledgement: replay.Acknowledgement}, nil
	}
	endpoint := MailboxEndpoint{
		PrincipalRef: principal.Ref, WorkItemRef: request.RecipientWorkItemRef,
		ExecutionRef: request.RecipientExecutionRef,
	}
	mailbox, err := orchestrator.state.GetMailbox(
		ctx, projectRef, request.GoalRef, request.MessageRef, endpoint,
	)
	if err != nil {
		return MailboxResolutionResult{}, err
	}
	if err := validateMailboxMutationRecord(
		mailbox, request.MessageRef, principal.Ref, request.RecipientWorkItemRef,
		request.RecipientExecutionRef, request.ClaimToken, request.Fence,
		MailboxStateConsumed, false,
	); err != nil {
		return MailboxResolutionResult{}, err
	}
	current, err := orchestrator.state.GetGoal(ctx, request.GoalRef)
	if err != nil {
		return MailboxResolutionResult{}, err
	}
	if current.Goal.Project() != projectRef {
		return MailboxResolutionResult{}, &StateError{Code: StateNotFound}
	}
	if current.Goal.Revision() != request.ExpectedGoalRevision ||
		current.Goal.PlanGeneration() != request.ExpectedPlanGeneration ||
		current.Goal.PlanGeneration() < mailbox.Envelope.TargetPlanGeneration {
		return MailboxResolutionResult{}, &StateError{Code: StateConflict}
	}
	now := orchestrator.clock.Now().UTC()
	ackRef, err := orchestrator.ids.NewID(ctx, "mailbox-acknowledgement")
	if err != nil {
		return MailboxResolutionResult{}, err
	}
	updated, err := BuildMailboxResolutionGoal(MailboxResolutionGoalInput{
		Current: current.Goal, ExpectedRevision: request.ExpectedGoalRevision,
		ParentWorkItemRef: mailbox.Envelope.ParentWorkItemRef,
		ChildWorkItemRef:  mailbox.Envelope.ChildWorkItemRef,
		MessageRef:        request.MessageRef, Outcome: outcome, ReceiptRef: ackRef, ResolvedAt: now,
	})
	if err != nil {
		return MailboxResolutionResult{}, err
	}
	events := []EventRecord{{
		Ref:  "event:mailbox-" + string(outcome) + ":" + request.MessageRef.String(),
		Kind: "mailbox." + string(outcome), GoalRef: request.GoalRef,
		WorkItemRef:  request.RecipientWorkItemRef,
		ExecutionRef: request.RecipientExecutionRef, OccurredAt: now,
	}}
	if updated.IsTerminal() {
		events = append(events, EventRecord{
			Ref:  "event:goal-" + string(updated.State()) + ":" + request.MessageRef.String(),
			Kind: "goal." + string(updated.State()), GoalRef: request.GoalRef, OccurredAt: now,
		})
	}
	acknowledgement := MailboxAcknowledgement{
		Ref: ackRef, MessageRef: request.MessageRef,
		ActionRef:  "action:mailbox:" + request.MessageRef.String(),
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		ProjectRef: projectRef, GoalRef: request.GoalRef,
		TargetPlanGeneration: mailbox.Envelope.TargetPlanGeneration,
		ParentWorkItemRef:    mailbox.Envelope.ParentWorkItemRef,
		ChildWorkItemRef:     mailbox.Envelope.ChildWorkItemRef,
		Recipient:            endpoint, Fence: request.Fence,
		Outcome: outcome, EffectOrReworkRef: request.EffectOrReworkRef,
		AcknowledgedAt: now, AuthorizationReceipt: authorization,
	}
	state := ResolveMailboxState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		AuthorizationReceipt: authorization, PrincipalRef: principal.Ref,
		ProjectRef: projectRef, GoalRef: request.GoalRef, MessageRef: request.MessageRef,
		RecipientExecutionRef: request.RecipientExecutionRef, ClaimToken: request.ClaimToken,
		Fence:                  request.Fence,
		ExpectedGoalRevision:   request.ExpectedGoalRevision,
		ExpectedPlanGeneration: request.ExpectedPlanGeneration,
		Goal:                   updated, Acknowledgement: acknowledgement,
		Events: events, OperationAt: now,
	}
	var persisted MailboxAcknowledgement
	var created bool
	if outcome == MailboxOutcomeBlocked {
		persisted, created, err = orchestrator.state.BlockMailbox(ctx, state)
	} else {
		persisted, created, err = orchestrator.state.AcknowledgeMailbox(ctx, state)
	}
	if err != nil {
		return MailboxResolutionResult{}, err
	}
	if err := validateMailboxAcknowledgement(
		persisted, request, principal.Ref, projectRef, fingerprint, outcome,
		mailbox.Envelope.TargetPlanGeneration,
	); err != nil {
		return MailboxResolutionResult{}, err
	}
	return MailboxResolutionResult{Acknowledgement: persisted, Created: created}, nil
}
