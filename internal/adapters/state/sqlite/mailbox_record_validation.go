package sqlite

import (
	"encoding/hex"
	"errors"
	"strings"

	"orquesta/internal/application"
	"orquesta/internal/identity"
)

// validatePersistedMailboxRecord validates the causal projection returned by
// the canonical reader. Database constraints remain defense in depth; recovery
// must also reject rows imported with constraints disabled.
func validatePersistedMailboxRecord(record application.MailboxRecord) error {
	envelope := record.Envelope
	if envelope.Ref.String() == "" || !validText(envelope.RequestRef) ||
		!validText(envelope.RequestFingerprint) || envelope.ProjectRef.String() == "" ||
		envelope.GoalRef.String() == "" || envelope.TargetPlanGeneration == 0 ||
		!validMailboxKind(envelope.Kind) || envelope.ParentWorkItemRef.String() == "" ||
		envelope.ChildWorkItemRef.String() == "" || envelope.ParentWorkItemRef == envelope.ChildWorkItemRef ||
		envelope.Source.PrincipalRef.String() == "" ||
		envelope.Source.WorkItemRef != envelope.ChildWorkItemRef || envelope.Source.ExecutionRef.String() == "" ||
		envelope.Recipient.PrincipalRef.String() == "" ||
		envelope.Recipient.WorkItemRef != envelope.ParentWorkItemRef ||
		envelope.Recipient.ExecutionRef.String() == "" ||
		envelope.Source.ExecutionRef == envelope.Recipient.ExecutionRef ||
		!validText(envelope.Summary) || !validMailboxContentHash(envelope.ContentHash) ||
		envelope.RequestFingerprint != application.MailboxAdmissionFingerprint(envelope) ||
		envelope.ContentHash != application.MailboxEnvelopeContentHash(envelope) ||
		envelope.AdmittedAt.IsZero() {
		return errors.New("sqlite.mailbox_record_envelope_invalid")
	}
	seenArtifacts := make(map[string]struct{}, len(envelope.ArtifactRefs))
	for _, artifactRef := range envelope.ArtifactRefs {
		if artifactRef.String() == "" {
			return errors.New("sqlite.mailbox_record_artifact_invalid")
		}
		if _, duplicate := seenArtifacts[artifactRef.String()]; duplicate {
			return errors.New("sqlite.mailbox_record_artifact_duplicate")
		}
		seenArtifacts[artifactRef.String()] = struct{}{}
	}

	admission := record.Admission
	if admission.Ref == "" || admission.MessageRef != envelope.Ref ||
		admission.RequestRef != envelope.RequestRef ||
		admission.RequestFingerprint != envelope.RequestFingerprint ||
		admission.PrincipalRef != envelope.Source.PrincipalRef ||
		!admission.AdmittedAt.Equal(envelope.AdmittedAt) ||
		admission.AuthorizationReceipt.Ref() == "" ||
		admission.AuthorizationReceipt.RecordedAt().After(admission.AdmittedAt) {
		return errors.New("sqlite.mailbox_record_admission_invalid")
	}
	admissionDecision := admission.AuthorizationReceipt.Decision()
	admissionRequest := admissionDecision.Request()
	if admissionDecision.Outcome() != identity.AuthorizationAllowed ||
		admissionRequest.Principal().Ref != envelope.Source.PrincipalRef ||
		admissionRequest.ProjectRef() != envelope.ProjectRef ||
		admissionRequest.Permission() != identity.PermissionGoalsDirect ||
		admissionRequest.ResourceRef() != envelope.GoalRef.String() {
		return errors.New("sqlite.mailbox_record_admission_authorization_invalid")
	}

	action := record.Action
	if action.Ref != "action:mailbox:"+envelope.Ref.String() ||
		action.Kind != application.ActionDeliverMailbox || action.GoalRef != envelope.GoalRef ||
		action.WorkItemRef != envelope.ParentWorkItemRef ||
		action.ExecutionRef != envelope.Recipient.ExecutionRef ||
		action.PlanGeneration != envelope.TargetPlanGeneration || action.WorkItemGeneration == 0 ||
		!action.AvailableAt.Equal(envelope.AdmittedAt) {
		return errors.New("sqlite.mailbox_record_action_invalid")
	}

	wantState := application.MailboxStateAdmitted
	for index, attempt := range record.Attempts {
		if attempt.MessageRef != envelope.Ref || attempt.ActionRef != action.Ref ||
			attempt.Recipient != envelope.Recipient || !validText(attempt.ClaimRequestRef) ||
			!validText(attempt.ClaimToken) || attempt.Fence != uint64(index+1) ||
			attempt.ExpectedGoalRevision == 0 ||
			attempt.ExpectedPlanGeneration < envelope.TargetPlanGeneration ||
			attempt.ClaimedAt.IsZero() ||
			!attempt.LeaseUntil.After(attempt.ClaimedAt) {
			return errors.New("sqlite.mailbox_record_attempt_invalid")
		}
		if index > 0 {
			previous := record.Attempts[index-1]
			if attempt.ClaimedAt.Before(previous.LeaseUntil) || !previous.ConsumedAt.IsZero() {
				return errors.New("sqlite.mailbox_record_attempt_chain_invalid")
			}
		}
		delivered := attempt.DeliveryRef != "" || !attempt.DeliveredAt.IsZero()
		if delivered != (validText(attempt.DeliveryRef) && !attempt.DeliveredAt.IsZero()) ||
			(delivered && (attempt.DeliveredAt.Before(attempt.ClaimedAt) ||
				!attempt.DeliveredAt.Before(attempt.LeaseUntil))) {
			return errors.New("sqlite.mailbox_record_delivery_invalid")
		}
		consumed := attempt.ConsumptionRef != "" || !attempt.ConsumedAt.IsZero()
		if consumed != (validText(attempt.ConsumptionRef) && !attempt.ConsumedAt.IsZero()) ||
			(consumed && (!delivered || attempt.ConsumedAt.Before(attempt.DeliveredAt) ||
				!attempt.ConsumedAt.Before(attempt.LeaseUntil))) {
			return errors.New("sqlite.mailbox_record_consumption_invalid")
		}
		wantState = application.MailboxStateClaimed
		if delivered {
			wantState = application.MailboxStateDelivered
		}
		if consumed {
			wantState = application.MailboxStateConsumed
		}
	}

	if record.Acknowledgement != nil {
		ack := *record.Acknowledgement
		if len(record.Attempts) == 0 || wantState != application.MailboxStateConsumed {
			return errors.New("sqlite.mailbox_record_ack_without_consumption")
		}
		last := record.Attempts[len(record.Attempts)-1]
		if ack.Ref == "" || ack.MessageRef != envelope.Ref || ack.ActionRef != action.Ref ||
			!validText(ack.RequestRef) || !validText(ack.RequestFingerprint) ||
			ack.ProjectRef != envelope.ProjectRef || ack.GoalRef != envelope.GoalRef ||
			ack.TargetPlanGeneration != envelope.TargetPlanGeneration ||
			ack.ParentWorkItemRef != envelope.ParentWorkItemRef ||
			ack.ChildWorkItemRef != envelope.ChildWorkItemRef || ack.Recipient != envelope.Recipient ||
			ack.Fence != last.Fence ||
			!validText(ack.EffectOrReworkRef) || ack.AcknowledgedAt.Before(last.ConsumedAt) ||
			ack.AuthorizationReceipt.Ref() == "" ||
			ack.AuthorizationReceipt.RecordedAt().After(ack.AcknowledgedAt) {
			return errors.New("sqlite.mailbox_record_ack_invalid")
		}
		if ack.Outcome != application.MailboxOutcomeAcknowledged &&
			ack.Outcome != application.MailboxOutcomeBlocked {
			return errors.New("sqlite.mailbox_record_ack_outcome_invalid")
		}
		if ack.AuthorizationReceipt.Decision().Outcome() != identity.AuthorizationAllowed ||
			validateMailboxAuthorization(
				ack.AuthorizationReceipt, envelope.Recipient.PrincipalRef,
				envelope.ProjectRef, envelope.Ref,
			) != nil {
			return errors.New("sqlite.mailbox_record_ack_authorization_invalid")
		}
		if ack.Outcome == application.MailboxOutcomeBlocked {
			wantState = application.MailboxStateBlocked
		} else {
			wantState = application.MailboxStateAcknowledged
		}
	}
	if record.Retirement != nil {
		if record.Acknowledgement != nil || application.ValidateMailboxRetirement(record) != nil {
			return errors.New("sqlite.mailbox_record_retirement_invalid")
		}
		wantState = application.MailboxStateRetired
	}
	if record.State != wantState {
		return errors.New("sqlite.mailbox_record_state_invalid")
	}
	return nil
}

func validMailboxContentHash(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32 && value == strings.ToLower(value)
}
