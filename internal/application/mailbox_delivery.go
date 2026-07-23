package application

import (
	"context"
	"errors"

	"orquesta/internal/identity"
)

func (orchestrator *Orchestrator) ClaimMailbox(
	ctx context.Context,
	access Access,
	request ClaimMailboxRequest,
) (MailboxClaimResult, error) {
	if orchestrator == nil {
		return MailboxClaimResult{}, errors.New("application.unavailable")
	}
	if err := validateMailboxAddressedRequest(
		request.RequestRef, request.GoalRef, request.MessageRef,
		request.RecipientWorkItemRef, request.RecipientExecutionRef,
	); err != nil {
		return MailboxClaimResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return MailboxClaimResult{}, err
	}
	if _, err := access.authenticatedExecution(request.RecipientExecutionRef); err != nil {
		return MailboxClaimResult{}, err
	}
	fingerprint := mailboxMutationFingerprint(
		MailboxMutationClaim, principal.Ref, projectRef, request.GoalRef, request.MessageRef,
		request.RecipientWorkItemRef, request.RecipientExecutionRef, "", 0, "",
	)
	if err := orchestrator.requireCurrentMailboxAccess(
		ctx, access, principal.Ref, projectRef, identity.PermissionGoalsGet,
	); err != nil {
		return MailboxClaimResult{}, err
	}
	replay, found, err := orchestrator.state.MailboxReplay(ctx, MailboxReplayRequest{
		Kind: MailboxMutationClaim, RequestRef: request.RequestRef,
		RequestFingerprint: fingerprint, PrincipalRef: principal.Ref,
		ProjectRef: projectRef, GoalRef: request.GoalRef, MessageRef: request.MessageRef,
	})
	if err != nil {
		return MailboxClaimResult{}, err
	}
	endpoint := MailboxEndpoint{
		PrincipalRef: principal.Ref, WorkItemRef: request.RecipientWorkItemRef,
		ExecutionRef: request.RecipientExecutionRef,
	}
	if found {
		if err := validateMailboxClaim(replay.Claim, request.RequestRef, request.MessageRef, endpoint, ""); err != nil {
			return MailboxClaimResult{}, err
		}
		return MailboxClaimResult{Claim: cloneMailboxClaim(replay.Claim)}, nil
	}
	authorization, err := orchestrator.mailboxMutationAuthorization(
		ctx, access, MailboxMutationClaim, request.RequestRef, fingerprint, request.MessageRef,
	)
	if err != nil {
		return MailboxClaimResult{}, err
	}
	token, err := orchestrator.ids.NewID(ctx, "mailbox-claim")
	if err != nil {
		return MailboxClaimResult{}, err
	}
	claim, claimed, err := orchestrator.state.ClaimMailbox(ctx, ClaimMailboxState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		AuthorizationReceipt: authorization, PrincipalRef: principal.Ref,
		ProjectRef: projectRef, GoalRef: request.GoalRef, MessageRef: request.MessageRef,
		RecipientExecutionRef: request.RecipientExecutionRef, Token: token,
		LeaseDuration: orchestrator.claimLease, RequestedAt: orchestrator.clock.Now().UTC(),
	})
	if err != nil {
		return MailboxClaimResult{}, err
	}
	if err := validateMailboxClaim(claim, request.RequestRef, request.MessageRef, endpoint, token); err != nil {
		return MailboxClaimResult{}, err
	}
	return MailboxClaimResult{Claim: cloneMailboxClaim(claim), Claimed: claimed}, nil
}

func (orchestrator *Orchestrator) MarkMailboxDelivered(
	ctx context.Context,
	access Access,
	request MarkMailboxDeliveredRequest,
) (MailboxMutationResult, error) {
	if orchestrator == nil {
		return MailboxMutationResult{}, errors.New("application.unavailable")
	}
	if err := validateMailboxClaimMutation(
		request.RequestRef, request.GoalRef, request.MessageRef,
		request.RecipientWorkItemRef, request.RecipientExecutionRef,
		request.ClaimToken, request.Fence,
	); err != nil {
		return MailboxMutationResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return MailboxMutationResult{}, err
	}
	if _, err := access.authenticatedExecution(request.RecipientExecutionRef); err != nil {
		return MailboxMutationResult{}, err
	}
	fingerprint := mailboxMutationFingerprint(
		MailboxMutationDeliver, principal.Ref, projectRef, request.GoalRef, request.MessageRef,
		request.RecipientWorkItemRef, request.RecipientExecutionRef,
		request.ClaimToken, request.Fence, "",
	)
	replay, authorization, err := orchestrator.prepareMailboxMutation(
		ctx, access, MailboxMutationDeliver, request.RequestRef, fingerprint,
		request.GoalRef, request.MessageRef,
	)
	if err != nil {
		return MailboxMutationResult{}, err
	}
	if replay != nil {
		if err := validateMailboxMutationRecord(
			replay.Record, request.MessageRef, principal.Ref, request.RecipientWorkItemRef,
			request.RecipientExecutionRef, request.ClaimToken, request.Fence,
			MailboxStateDelivered, true,
		); err != nil {
			return MailboxMutationResult{}, err
		}
		return MailboxMutationResult{Record: cloneMailboxRecord(replay.Record)}, nil
	}
	deliveryRef, err := orchestrator.ids.NewID(ctx, "mailbox-delivery")
	if err != nil {
		return MailboxMutationResult{}, err
	}
	persisted, changed, err := orchestrator.state.MarkMailboxDelivered(ctx, MarkMailboxDeliveredState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		AuthorizationReceipt: authorization, PrincipalRef: principal.Ref,
		ProjectRef: projectRef, GoalRef: request.GoalRef, MessageRef: request.MessageRef,
		RecipientExecutionRef: request.RecipientExecutionRef, ClaimToken: request.ClaimToken,
		Fence:       request.Fence,
		DeliveryRef: deliveryRef, OperationAt: orchestrator.clock.Now().UTC(),
	})
	if err != nil {
		return MailboxMutationResult{}, err
	}
	if err := validateMailboxMutationRecord(
		persisted, request.MessageRef, principal.Ref, request.RecipientWorkItemRef,
		request.RecipientExecutionRef, request.ClaimToken, request.Fence,
		MailboxStateDelivered, false,
	); err != nil {
		return MailboxMutationResult{}, err
	}
	return MailboxMutationResult{Record: cloneMailboxRecord(persisted), Changed: changed}, nil
}

func (orchestrator *Orchestrator) ConsumeMailbox(
	ctx context.Context,
	access Access,
	request ConsumeMailboxRequest,
) (MailboxMutationResult, error) {
	if orchestrator == nil {
		return MailboxMutationResult{}, errors.New("application.unavailable")
	}
	if err := validateMailboxClaimMutation(
		request.RequestRef, request.GoalRef, request.MessageRef,
		request.RecipientWorkItemRef, request.RecipientExecutionRef,
		request.ClaimToken, request.Fence,
	); err != nil {
		return MailboxMutationResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return MailboxMutationResult{}, err
	}
	if _, err := access.authenticatedExecution(request.RecipientExecutionRef); err != nil {
		return MailboxMutationResult{}, err
	}
	fingerprint := mailboxMutationFingerprint(
		MailboxMutationConsume, principal.Ref, projectRef, request.GoalRef, request.MessageRef,
		request.RecipientWorkItemRef, request.RecipientExecutionRef,
		request.ClaimToken, request.Fence, "",
	)
	replay, authorization, err := orchestrator.prepareMailboxMutation(
		ctx, access, MailboxMutationConsume, request.RequestRef, fingerprint,
		request.GoalRef, request.MessageRef,
	)
	if err != nil {
		return MailboxMutationResult{}, err
	}
	if replay != nil {
		if err := validateMailboxMutationRecord(
			replay.Record, request.MessageRef, principal.Ref, request.RecipientWorkItemRef,
			request.RecipientExecutionRef, request.ClaimToken, request.Fence,
			MailboxStateConsumed, true,
		); err != nil {
			return MailboxMutationResult{}, err
		}
		return MailboxMutationResult{Record: cloneMailboxRecord(replay.Record)}, nil
	}
	consumptionRef, err := orchestrator.ids.NewID(ctx, "mailbox-consumption")
	if err != nil {
		return MailboxMutationResult{}, err
	}
	endpoint := MailboxEndpoint{
		PrincipalRef: principal.Ref, WorkItemRef: request.RecipientWorkItemRef,
		ExecutionRef: request.RecipientExecutionRef,
	}
	current, err := orchestrator.state.GetMailbox(
		ctx, projectRef, request.GoalRef, request.MessageRef, endpoint,
	)
	if err != nil {
		return MailboxMutationResult{}, err
	}
	if err := validateMailboxMutationRecord(
		current, request.MessageRef, principal.Ref, request.RecipientWorkItemRef,
		request.RecipientExecutionRef, request.ClaimToken, request.Fence,
		MailboxStateDelivered, false,
	); err != nil {
		return MailboxMutationResult{}, err
	}
	now := orchestrator.clock.Now().UTC()
	receipt := ActionConsumptionReceipt{
		ActionRef: current.Action.Ref, Kind: ActionDeliverMailbox, GoalRef: request.GoalRef,
		WorkItemRef: request.RecipientWorkItemRef, ExecutionRef: request.RecipientExecutionRef,
		MailboxMessageRef: request.MessageRef, PlanGeneration: current.Action.PlanGeneration,
		WorkItemGeneration: current.Action.WorkItemGeneration, Fence: request.Fence,
		DeliveryAttempt: request.Fence, ClaimToken: request.ClaimToken,
		WorkerRef: principal.Ref.String(), Outcome: ActionConsumedCompleted,
		ConsumedAt: now,
	}
	persisted, changed, err := orchestrator.state.ConsumeMailbox(ctx, ConsumeMailboxState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		AuthorizationReceipt: authorization, PrincipalRef: principal.Ref,
		ProjectRef: projectRef, GoalRef: request.GoalRef, MessageRef: request.MessageRef,
		RecipientExecutionRef: request.RecipientExecutionRef, ClaimToken: request.ClaimToken,
		Fence:          request.Fence,
		ConsumptionRef: consumptionRef, ConsumptionReceipt: receipt, OperationAt: now,
	})
	if err != nil {
		return MailboxMutationResult{}, err
	}
	if err := validateMailboxMutationRecord(
		persisted, request.MessageRef, principal.Ref, request.RecipientWorkItemRef,
		request.RecipientExecutionRef, request.ClaimToken, request.Fence,
		MailboxStateConsumed, false,
	); err != nil {
		return MailboxMutationResult{}, err
	}
	return MailboxMutationResult{Record: cloneMailboxRecord(persisted), Changed: changed}, nil
}
