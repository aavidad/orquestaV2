package application

import (
	"context"
	"testing"
)

func TestMailboxTerminalFactsRejectNilAndMismatchedFactsOnReplayAndQuery(t *testing.T) {
	corruptions := []struct {
		name   string
		mutate func(*MailboxRecord)
	}{
		{
			name: "acknowledged_without_ack",
			mutate: func(record *MailboxRecord) {
				record.Acknowledgement = nil
			},
		},
		{
			name: "blocked_without_ack",
			mutate: func(record *MailboxRecord) {
				record.State = MailboxStateBlocked
				record.Acknowledgement = nil
			},
		},
		{
			name: "acknowledged_with_blocked_outcome",
			mutate: func(record *MailboxRecord) {
				acknowledgement := *record.Acknowledgement
				acknowledgement.Outcome = MailboxOutcomeBlocked
				record.Acknowledgement = &acknowledgement
			},
		},
		{
			name: "blocked_with_acknowledged_outcome",
			mutate: func(record *MailboxRecord) {
				record.State = MailboxStateBlocked
			},
		},
		{
			name: "ack_on_nonterminal_state",
			mutate: func(record *MailboxRecord) {
				record.State = MailboxStateConsumed
			},
		},
		{
			name: "retired_without_retirement",
			mutate: func(record *MailboxRecord) {
				record.State = MailboxStateRetired
				record.Acknowledgement = nil
			},
		},
		{
			name: "retirement_on_nonterminal_state",
			mutate: func(record *MailboxRecord) {
				acknowledgedAt := record.Acknowledgement.AcknowledgedAt
				record.State = MailboxStateConsumed
				record.Acknowledgement = nil
				record.Retirement = &MailboxRetirement{
					MessageRef: record.Envelope.Ref, ActionRef: record.Action.Ref,
					RecipientExecutionRef: record.Envelope.Recipient.ExecutionRef,
					FailureCode:           "agent.adversarial_retirement", RetiredAt: acknowledgedAt,
				}
			},
		},
	}

	for _, corruption := range corruptions {
		corruption := corruption
		for _, surface := range []string{"admission_replay", "query"} {
			surface := surface
			t.Run(corruption.name+"/"+surface, func(t *testing.T) {
				ctx := context.Background()
				suffix := "terminal-facts-" + corruption.name + "-" + surface
				system := newMailboxTestSystem(t, 1)
				flow := system.consume(t, 0, suffix)
				request := system.resolveRequest(
					flow, "mailbox-ack:"+suffix, "effect:"+suffix,
				)
				if result, err := system.orchestrator.AcknowledgeMailbox(
					ctx, system.recipientAccess, AcknowledgeMailboxRequest(request),
				); err != nil || !result.Created {
					t.Fatalf("terminal fixture=%+v err=%v", result, err)
				}

				messageRef := flow.admission.Record.Envelope.Ref
				system.repository.mu.Lock()
				record := cloneMailboxRecord(system.repository.mailboxes[messageRef])
				corruption.mutate(&record)
				system.repository.mailboxes[messageRef] = cloneMailboxRecord(record)
				system.repository.mu.Unlock()

				var err error
				switch surface {
				case "admission_replay":
					_, err = system.orchestrator.AdmitMailbox(
						ctx, system.sourceAccess,
						system.admissionRequest(0, "mailbox-admit:"+suffix),
					)
				case "query":
					_, err = system.orchestrator.GetMailbox(ctx, system.recipientAccess, GetMailboxRequest{
						GoalRef: system.goalRef, MessageRef: messageRef,
						RecipientWorkItemRef:  system.parentRef,
						RecipientExecutionRef: system.parentExecution,
					})
				default:
					t.Fatalf("unknown surface %q", surface)
				}
				if !IsStateError(err, StateConflict) {
					t.Fatalf("corrupt terminal facts accepted: record=%+v err=%v", record, err)
				}
			})
		}
	}
}
