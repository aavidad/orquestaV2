package application

import (
	"errors"
	"time"

	"orquesta/internal/goal"
)

// MailboxResolutionGoalInput contains only the causal facts required to apply
// one mailbox resolution to the immutable Goal aggregate.
type MailboxResolutionGoalInput struct {
	Current           goal.Goal
	ExpectedRevision  goal.Revision
	ParentWorkItemRef goal.WorkItemRef
	ChildWorkItemRef  goal.WorkItemRef
	MessageRef        MailboxMessageRef
	Outcome           MailboxOutcome
	ReceiptRef        string
	ResolvedAt        time.Time
}

// BuildMailboxResolutionGoal is the single pure application transition for a
// child-handoff resolution. Adapters persist its result; they never reproduce
// ResolveChildHandoff or Goal closure rules.
func BuildMailboxResolutionGoal(input MailboxResolutionGoalInput) (goal.Goal, error) {
	handoffOutcome, err := mailboxResolutionGoalOutcome(input.Outcome)
	if err != nil {
		return goal.Goal{}, err
	}
	updated, err := input.Current.ResolveChildHandoff(
		input.ExpectedRevision, input.ParentWorkItemRef, input.ChildWorkItemRef,
		input.MessageRef.String(), handoffOutcome, input.ReceiptRef, input.ResolvedAt,
	)
	if err != nil {
		return goal.Goal{}, err
	}
	if closureOutcome, closable := updated.ClosableOutcome(); closable {
		return updated.Close(updated.Revision(), closureOutcome, input.ResolvedAt)
	}
	return updated, nil
}

func mailboxResolutionGoalOutcome(outcome MailboxOutcome) (goal.ChildHandoffOutcome, error) {
	switch outcome {
	case MailboxOutcomeAcknowledged:
		return goal.ChildHandoffAcknowledged, nil
	case MailboxOutcomeBlocked:
		return goal.ChildHandoffBlocked, nil
	default:
		return "", errors.New("application.mailbox_resolution_outcome_invalid")
	}
}
