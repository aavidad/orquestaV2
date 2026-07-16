package sqlite

import (
	"context"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/identity"
)

func (repository *Repository) AcknowledgeMailbox(
	ctx context.Context,
	state application.ResolveMailboxState,
) (application.MailboxAcknowledgement, bool, error) {
	return repository.resolveMailbox(ctx, state, application.MailboxOutcomeAcknowledged)
}

func (repository *Repository) BlockMailbox(
	ctx context.Context,
	state application.ResolveMailboxState,
) (application.MailboxAcknowledgement, bool, error) {
	return repository.resolveMailbox(ctx, state, application.MailboxOutcomeBlocked)
}

func (repository *Repository) resolveMailbox(
	ctx context.Context,
	state application.ResolveMailboxState,
	outcome application.MailboxOutcome,
) (application.MailboxAcknowledgement, bool, error) {
	if err := validateResolveMailboxState(state, outcome); err != nil {
		return application.MailboxAcknowledgement{}, false, invalid(err)
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.MailboxAcknowledgement{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()
	if _, err := requirePersistedAuthorization(
		ctx, transaction, state.AuthorizationReceipt, state.PrincipalRef,
		state.ProjectRef, identity.PermissionGoalsGet, state.MessageRef.String(),
	); err != nil {
		return application.MailboxAcknowledgement{}, false, err
	}
	kind := application.MailboxMutationAcknowledge
	if outcome == application.MailboxOutcomeBlocked {
		kind = application.MailboxMutationBlock
	}
	replay, found, err := readMailboxReplay(ctx, transaction, application.MailboxReplayRequest{
		Kind: kind, RequestRef: state.RequestRef, RequestFingerprint: state.RequestFingerprint,
		PrincipalRef: state.PrincipalRef, ProjectRef: state.ProjectRef,
		GoalRef: state.GoalRef, MessageRef: state.MessageRef,
	})
	if err != nil {
		return application.MailboxAcknowledgement{}, false, err
	}
	if found {
		if err := commit(transaction); err != nil {
			return application.MailboxAcknowledgement{}, false, err
		}
		return replay.Acknowledgement, false, nil
	}
	current, err := readMailboxRecord(ctx, transaction, state.MessageRef.String())
	if err != nil {
		return application.MailboxAcknowledgement{}, false, err
	}
	if err := requireMailboxRecipient(current, state.ProjectRef, state.GoalRef, state.PrincipalRef, state.RecipientExecutionRef); err != nil {
		return application.MailboxAcknowledgement{}, false, err
	}
	if current.Retirement != nil {
		return application.MailboxAcknowledgement{}, false, conflict(errors.New("sqlite.mailbox_retired"))
	}
	if err := requireMailboxCurrentRecipient(ctx, transaction, current); err != nil {
		return application.MailboxAcknowledgement{}, false, err
	}
	if state.Acknowledgement.TargetPlanGeneration != current.Envelope.TargetPlanGeneration ||
		state.ExpectedPlanGeneration < current.Envelope.TargetPlanGeneration {
		return application.MailboxAcknowledgement{}, false,
			conflict(errors.New("sqlite.mailbox_plan_generation_conflict"))
	}
	if current.State != application.MailboxStateConsumed || len(current.Attempts) == 0 {
		return application.MailboxAcknowledgement{}, false, conflict(errors.New("sqlite.mailbox_not_consumed"))
	}
	attempt := current.Attempts[len(current.Attempts)-1]
	if attempt.ClaimToken != state.ClaimToken || attempt.Fence != state.Fence {
		return application.MailboxAcknowledgement{}, false, conflict(errors.New("sqlite.mailbox_ack_fence_conflict"))
	}
	storedGoal, err := readGoalRecord(ctx, transaction, state.GoalRef.String())
	if err != nil {
		return application.MailboxAcknowledgement{}, false, err
	}
	if storedGoal.Goal.Revision() != state.ExpectedGoalRevision ||
		storedGoal.Goal.PlanGeneration() != state.ExpectedPlanGeneration {
		return application.MailboxAcknowledgement{}, false, conflict(errors.New("sqlite.mailbox_goal_revision_conflict"))
	}
	if err := validateMailboxResolutionTransition(storedGoal.Goal, state, outcome); err != nil {
		return application.MailboxAcknowledgement{}, false, conflict(err)
	}
	if err := updateGoalCAS(ctx, transaction, state.Goal, state.ExpectedGoalRevision); err != nil {
		return application.MailboxAcknowledgement{}, false, err
	}
	ack := state.Acknowledgement
	_, err = transaction.ExecContext(ctx, `
INSERT INTO mailbox_delivery_acks(
    ref, request_ref, request_fingerprint, authorization_receipt_ref,
    mailbox_message_ref, action_ref, project_ref,
    expected_goal_revision, expected_plan_generation, recipient_principal_ref,
    outcome, effect_or_rework_ref, acked_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ack.Ref, state.RequestRef, state.RequestFingerprint, state.AuthorizationReceipt.Ref(),
		state.MessageRef.String(), current.Action.Ref, state.ProjectRef.String(),
		int64(state.ExpectedGoalRevision), int64(state.ExpectedPlanGeneration),
		state.PrincipalRef.String(), string(outcome), ack.EffectOrReworkRef,
		requiredTime(ack.AcknowledgedAt),
	)
	if err != nil {
		return application.MailboxAcknowledgement{}, false, mapDatabaseError(err)
	}
	_, err = transaction.ExecContext(ctx, `
INSERT INTO goal_child_handoff_resolutions(
    goal_ref, parent_work_item_ref, child_work_item_ref, mailbox_message_ref,
    outcome, receipt_ref, resolved_at
) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		state.GoalRef.String(), current.Envelope.ParentWorkItemRef.String(),
		current.Envelope.ChildWorkItemRef.String(), state.MessageRef.String(),
		string(outcome), ack.Ref, requiredTime(ack.AcknowledgedAt),
	)
	if err != nil {
		return application.MailboxAcknowledgement{}, false, mapDatabaseError(err)
	}
	if err := insertEvents(ctx, transaction, state.Events); err != nil {
		return application.MailboxAcknowledgement{}, false, err
	}
	persisted, err := readMailboxRecord(ctx, transaction, state.MessageRef.String())
	if err != nil {
		return application.MailboxAcknowledgement{}, false, err
	}
	if persisted.Acknowledgement == nil {
		return application.MailboxAcknowledgement{}, false, conflict(errors.New("sqlite.mailbox_ack_missing_after_write"))
	}
	if _, err := readGoalRecord(ctx, transaction, state.GoalRef.String()); err != nil {
		return application.MailboxAcknowledgement{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.MailboxAcknowledgement{}, false, err
	}
	return *persisted.Acknowledgement, true, nil
}
