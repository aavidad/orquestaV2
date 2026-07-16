package application

import (
	"errors"
	"time"
)

// BuildMailboxRetirement derives the only valid retirement fact for an
// unresolved mailbox addressed to a failed execution. Adapters call this pure
// application helper while applying RecordGoalFailed atomically.
func BuildMailboxRetirement(
	record MailboxRecord,
	failed ExecutionRecord,
) (MailboxRetirement, error) {
	if record.Envelope.Ref.String() == "" ||
		record.Action.Ref != "action:mailbox:"+record.Envelope.Ref.String() ||
		record.Action.Kind != ActionDeliverMailbox ||
		record.Action.GoalRef != record.Envelope.GoalRef ||
		record.Action.WorkItemRef != record.Envelope.ParentWorkItemRef ||
		record.Action.ExecutionRef != record.Envelope.Recipient.ExecutionRef ||
		record.Acknowledgement != nil || record.Retirement != nil ||
		failed.Ref != record.Envelope.Recipient.ExecutionRef ||
		failed.GoalRef != record.Envelope.GoalRef ||
		failed.WorkItemRef != record.Envelope.ParentWorkItemRef ||
		failed.State != ExecutionFailed || !validMailboxText(failed.FailureCode) ||
		failed.FinishedAt.IsZero() || failed.FinishedAt.Before(record.Envelope.AdmittedAt) {
		return MailboxRetirement{}, errors.New("application.mailbox_retirement_invalid")
	}
	retirement := MailboxRetirement{
		MessageRef: record.Envelope.Ref, ActionRef: record.Action.Ref,
		RecipientExecutionRef: failed.Ref, FailureCode: failed.FailureCode,
		RetiredAt: failed.FinishedAt.UTC(),
	}
	candidate := cloneMailboxRecord(record)
	candidate.Retirement = &retirement
	candidate.State = MailboxStateRetired
	if err := ValidateMailboxRetirement(candidate); err != nil {
		return MailboxRetirement{}, err
	}
	return retirement, nil
}

// ValidateMailboxRetirement verifies a projected retirement independently of
// persistence. Recovery additionally binds the cause to the failed execution.
func ValidateMailboxRetirement(record MailboxRecord) error {
	if record.Retirement == nil {
		return errors.New("application.mailbox_retirement_missing")
	}
	retirement := record.Retirement
	if record.Acknowledgement != nil || retirement.MessageRef != record.Envelope.Ref ||
		retirement.ActionRef != record.Action.Ref ||
		retirement.RecipientExecutionRef != record.Envelope.Recipient.ExecutionRef ||
		!validMailboxText(retirement.FailureCode) || retirement.RetiredAt.IsZero() ||
		retirement.RetiredAt.Before(record.Envelope.AdmittedAt) ||
		record.State != MailboxStateRetired {
		return errors.New("application.mailbox_retirement_invalid")
	}
	latest := record.Envelope.AdmittedAt
	for _, attempt := range record.Attempts {
		for _, observed := range []time.Time{
			attempt.ClaimedAt, attempt.DeliveredAt, attempt.ConsumedAt,
		} {
			if observed.After(latest) {
				latest = observed
			}
		}
	}
	if retirement.RetiredAt.Before(latest) {
		return errors.New("application.mailbox_retirement_before_activity")
	}
	return nil
}
