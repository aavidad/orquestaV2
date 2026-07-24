package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type mailboxReplayLocator struct {
	messageRef         string
	requestFingerprint string
	projectRef         string
	goalRef            string
	principalRef       string
	fence              int64
	outcome            string
}

func readGoalMailboxRecords(ctx context.Context, source queryer, goalValue string) ([]application.MailboxRecord, error) {
	persisted, err := sqliteTableHasColumn(ctx, source, "mailbox_envelopes", "ref")
	if err != nil || !persisted {
		return nil, mapDatabaseError(err)
	}
	refs, err := readContractRefs(ctx, source, `
SELECT ref FROM mailbox_envelopes WHERE goal_ref = ? ORDER BY admitted_at, ref`, goalValue)
	if err != nil {
		return nil, err
	}
	records := make([]application.MailboxRecord, 0, len(refs))
	for _, ref := range refs {
		record, err := readMailboxRecord(ctx, source, ref)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func readMailboxRecord(ctx context.Context, source queryer, messageValue string) (application.MailboxRecord, error) {
	var record application.MailboxRecord
	var projectValue, goalValue, kind, sourcePrincipalValue, childValue, sourceExecutionValue string
	var recipientPrincipalValue, parentValue, recipientExecutionValue string
	var admissionAuthorizationRef, admissionProjectValue, admissionGoalValue, admissionPrincipalValue string
	var actionGoalValue, actionItemValue, actionExecutionValue string
	var planGeneration, sourceGeneration, envelopeRecipientGeneration int64
	var actionPlanGeneration, actionRecipientGeneration int64
	var admittedAt, admissionAdmittedAt, actionAvailableAt int64
	var actionRetiredAt sql.NullInt64
	err := source.QueryRowContext(ctx, `
SELECT e.ref, e.request_ref, e.request_fingerprint, e.project_ref, e.goal_ref,
       e.plan_generation, e.kind, e.parent_work_item_ref, e.child_work_item_ref,
       e.source_principal_ref, e.source_execution_ref, e.source_work_item_generation,
       e.recipient_principal_ref, e.recipient_execution_ref, e.recipient_work_item_generation,
       e.summary, e.content_hash, e.admitted_at,
       a.ref, a.request_ref, a.request_fingerprint, a.authorization_receipt_ref,
       a.project_ref, a.goal_ref, a.source_principal_ref, a.admitted_at,
       o.ref, o.kind, o.goal_ref, o.work_item_ref, o.execution_ref,
       o.plan_generation, o.work_item_generation, o.available_at, o.retired_at
FROM mailbox_envelopes e
JOIN mailbox_admission_receipts a ON a.mailbox_message_ref = e.ref
JOIN outbox o ON o.mailbox_message_ref = e.ref AND o.kind = 'deliver_mailbox'
WHERE e.ref = ?`, messageValue).Scan(
		&messageValue, &record.Envelope.RequestRef, &record.Envelope.RequestFingerprint,
		&projectValue, &goalValue, &planGeneration, &kind, &parentValue, &childValue,
		&sourcePrincipalValue, &sourceExecutionValue, &sourceGeneration, &recipientPrincipalValue,
		&recipientExecutionValue, &envelopeRecipientGeneration,
		&record.Envelope.Summary, &record.Envelope.ContentHash,
		&admittedAt, &record.Admission.Ref, &record.Admission.RequestRef,
		&record.Admission.RequestFingerprint, &admissionAuthorizationRef,
		&admissionProjectValue, &admissionGoalValue, &admissionPrincipalValue, &admissionAdmittedAt,
		&record.Action.Ref, &record.Action.Kind, &actionGoalValue, &actionItemValue,
		&actionExecutionValue, &actionPlanGeneration, &actionRecipientGeneration, &actionAvailableAt,
		&actionRetiredAt,
	)
	if err != nil {
		return application.MailboxRecord{}, mapDatabaseError(err)
	}
	var refErr error
	if record.Envelope.Ref, refErr = application.NewMailboxMessageRef(messageValue); refErr != nil {
		return application.MailboxRecord{}, invalid(refErr)
	}
	if record.Envelope.ProjectRef, refErr = goal.NewProjectRef(projectValue); refErr != nil {
		return application.MailboxRecord{}, invalid(refErr)
	}
	if record.Envelope.GoalRef, refErr = goal.NewGoalRef(goalValue); refErr != nil {
		return application.MailboxRecord{}, invalid(refErr)
	}
	if record.Envelope.ParentWorkItemRef, refErr = goal.NewWorkItemRef(parentValue); refErr != nil {
		return application.MailboxRecord{}, invalid(refErr)
	}
	if record.Envelope.ChildWorkItemRef, refErr = goal.NewWorkItemRef(childValue); refErr != nil {
		return application.MailboxRecord{}, invalid(refErr)
	}
	if record.Envelope.Source.PrincipalRef, refErr = identity.NewPrincipalRef(sourcePrincipalValue); refErr != nil {
		return application.MailboxRecord{}, invalid(refErr)
	}
	record.Envelope.Source.WorkItemRef = record.Envelope.ChildWorkItemRef
	if record.Envelope.Source.ExecutionRef, refErr = goal.NewExecutionRef(sourceExecutionValue); refErr != nil {
		return application.MailboxRecord{}, invalid(refErr)
	}
	if record.Envelope.Recipient.PrincipalRef, refErr = identity.NewPrincipalRef(recipientPrincipalValue); refErr != nil {
		return application.MailboxRecord{}, invalid(refErr)
	}
	record.Envelope.Recipient.WorkItemRef = record.Envelope.ParentWorkItemRef
	if record.Envelope.Recipient.ExecutionRef, refErr = goal.NewExecutionRef(recipientExecutionValue); refErr != nil {
		return application.MailboxRecord{}, invalid(refErr)
	}
	if planGeneration <= 0 || sourceGeneration <= 0 || envelopeRecipientGeneration <= 0 ||
		actionPlanGeneration <= 0 || actionRecipientGeneration <= 0 ||
		actionRecipientGeneration != envelopeRecipientGeneration {
		return application.MailboxRecord{}, invalid(errors.New("sqlite.mailbox_generation_invalid"))
	}
	record.Envelope.TargetPlanGeneration = goal.PlanGeneration(planGeneration)
	record.Envelope.Kind = application.MailboxKind(kind)
	record.Envelope.AdmittedAt = time.Unix(0, admittedAt).UTC()
	record.Envelope.ArtifactRefs, err = readMailboxArtifactRefs(ctx, source, messageValue)
	if err != nil {
		return application.MailboxRecord{}, err
	}
	record.Admission.MessageRef = record.Envelope.Ref
	if record.Admission.PrincipalRef, refErr = identity.NewPrincipalRef(admissionPrincipalValue); refErr != nil {
		return application.MailboxRecord{}, invalid(refErr)
	}
	record.Admission.AdmittedAt = time.Unix(0, admissionAdmittedAt).UTC()
	record.Admission.AuthorizationReceipt, err = readAuthorizationReceipt(ctx, source, admissionAuthorizationRef)
	if err != nil {
		return application.MailboxRecord{}, err
	}
	if admissionProjectValue != projectValue || admissionGoalValue != goalValue {
		return application.MailboxRecord{}, invalid(errors.New("sqlite.mailbox_admission_scope_invalid"))
	}
	if record.Action.GoalRef, refErr = goal.NewGoalRef(actionGoalValue); refErr != nil {
		return application.MailboxRecord{}, invalid(refErr)
	}
	if record.Action.WorkItemRef, refErr = goal.NewWorkItemRef(actionItemValue); refErr != nil {
		return application.MailboxRecord{}, invalid(refErr)
	}
	if record.Action.ExecutionRef, refErr = goal.NewExecutionRef(actionExecutionValue); refErr != nil {
		return application.MailboxRecord{}, invalid(refErr)
	}
	record.Action.PlanGeneration = goal.PlanGeneration(actionPlanGeneration)
	record.Action.WorkItemGeneration = goal.Revision(actionRecipientGeneration)
	record.Action.AvailableAt = time.Unix(0, actionAvailableAt).UTC()
	record.Attempts, err = readMailboxAttempts(ctx, source, record)
	if err != nil {
		return application.MailboxRecord{}, err
	}
	record.Acknowledgement, err = readMailboxAcknowledgement(ctx, source, record)
	if err != nil {
		return application.MailboxRecord{}, err
	}
	record.Retirement, err = readMailboxRetirement(ctx, source, record)
	if err != nil {
		return application.MailboxRecord{}, err
	}
	record.State = application.MailboxStateAdmitted
	if len(record.Attempts) > 0 {
		last := record.Attempts[len(record.Attempts)-1]
		record.State = application.MailboxStateClaimed
		if last.DeliveryRef != "" {
			record.State = application.MailboxStateDelivered
		}
		if last.ConsumptionRef != "" {
			record.State = application.MailboxStateConsumed
		}
	}
	if record.Acknowledgement != nil {
		if record.Acknowledgement.Outcome == application.MailboxOutcomeBlocked {
			record.State = application.MailboxStateBlocked
		} else {
			record.State = application.MailboxStateAcknowledged
		}
	}
	if record.Retirement != nil {
		if !actionRetiredAt.Valid || actionRetiredAt.Int64 != record.Retirement.RetiredAt.UnixNano() {
			return application.MailboxRecord{}, invalid(errors.New("sqlite.mailbox_retirement_outbox_invalid"))
		}
		record.State = application.MailboxStateRetired
	} else if actionRetiredAt.Valid {
		return application.MailboxRecord{}, invalid(errors.New("sqlite.mailbox_outbox_retirement_missing"))
	}
	if err := validatePersistedMailboxRecord(record); err != nil {
		return application.MailboxRecord{}, invalid(err)
	}
	return record, nil
}

func readMailboxArtifactRefs(ctx context.Context, source queryer, messageValue string) ([]goal.ArtifactRef, error) {
	rows, err := source.QueryContext(ctx, `
SELECT artifact_ref FROM mailbox_artifact_refs WHERE mailbox_message_ref = ? ORDER BY position`, messageValue)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []goal.ArtifactRef
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, mapDatabaseError(err)
		}
		ref, err := goal.NewArtifactRef(value)
		if err != nil {
			return nil, invalid(err)
		}
		result = append(result, ref)
	}
	return result, mapDatabaseError(rows.Err())
}

func readMailboxAttempts(
	ctx context.Context,
	source queryer,
	record application.MailboxRecord,
) ([]application.MailboxDeliveryAttempt, error) {
	rows, err := source.QueryContext(ctx, `
SELECT action_ref, project_ref, recipient_principal_ref, fence, claim_token,
       expected_goal_revision, expected_plan_generation,
       claim_request_ref, claim_request_fingerprint, claim_authorization_receipt_ref,
       claimed_at, lease_until,
       delivery_request_ref, delivery_request_fingerprint,
       delivery_authorization_receipt_ref, delivery_ref, delivered_at,
       consumption_request_ref, consumption_request_fingerprint,
       consumption_authorization_receipt_ref, consumption_ref, consumed_at
FROM mailbox_delivery_attempts
WHERE mailbox_message_ref = ? ORDER BY fence`, record.Envelope.Ref.String())
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []application.MailboxDeliveryAttempt
	for rows.Next() {
		var attempt application.MailboxDeliveryAttempt
		var projectValue, principalValue string
		var claimFingerprint, claimAuthorizationRef string
		var fence, expectedGoalRevision, expectedPlanGeneration, claimedAt, leaseUntil int64
		var deliveryRequest, deliveryFingerprint, deliveryAuthorizationRef sql.NullString
		var consumptionRequest, consumptionFingerprint, consumptionAuthorizationRef sql.NullString
		var deliveryRef, consumptionRef sql.NullString
		var deliveredAt, consumedAt sql.NullInt64
		if err := rows.Scan(
			&attempt.ActionRef, &projectValue,
			&principalValue,
			&fence, &attempt.ClaimToken, &expectedGoalRevision, &expectedPlanGeneration,
			&attempt.ClaimRequestRef,
			&claimFingerprint, &claimAuthorizationRef, &claimedAt, &leaseUntil,
			&deliveryRequest, &deliveryFingerprint, &deliveryAuthorizationRef,
			&deliveryRef, &deliveredAt, &consumptionRequest, &consumptionFingerprint,
			&consumptionAuthorizationRef, &consumptionRef, &consumedAt,
		); err != nil {
			return nil, mapDatabaseError(err)
		}
		if fence <= 0 || expectedGoalRevision <= 0 ||
			expectedPlanGeneration < int64(record.Envelope.TargetPlanGeneration) {
			return nil, invalid(errors.New("sqlite.mailbox_attempt_generation_invalid"))
		}
		attempt.MessageRef = record.Envelope.Ref
		var refErr error
		if attempt.Recipient.PrincipalRef, refErr = identity.NewPrincipalRef(principalValue); refErr != nil {
			return nil, invalid(refErr)
		}
		attempt.Recipient.WorkItemRef = record.Envelope.Recipient.WorkItemRef
		attempt.Recipient.ExecutionRef = record.Envelope.Recipient.ExecutionRef
		attempt.Fence = uint64(fence)
		attempt.ExpectedGoalRevision = goal.Revision(expectedGoalRevision)
		attempt.ExpectedPlanGeneration = goal.PlanGeneration(expectedPlanGeneration)
		attempt.ClaimedAt = time.Unix(0, claimedAt).UTC()
		attempt.LeaseUntil = time.Unix(0, leaseUntil).UTC()
		if attempt.ActionRef != record.Action.Ref || projectValue != record.Envelope.ProjectRef.String() ||
			!validText(attempt.ClaimRequestRef) || !validText(claimFingerprint) ||
			!validText(attempt.ClaimToken) || !attempt.LeaseUntil.After(attempt.ClaimedAt) {
			return nil, invalid(errors.New("sqlite.mailbox_attempt_scope_invalid"))
		}
		if claimFingerprint != application.MailboxMutationFingerprint(
			application.MailboxMutationClaim, attempt.Recipient.PrincipalRef,
			record.Envelope.ProjectRef, record.Envelope.GoalRef, record.Envelope.Ref,
			attempt.Recipient.WorkItemRef, attempt.Recipient.ExecutionRef, "", 0, "",
		) {
			return nil, invalid(errors.New("sqlite.mailbox_claim_fingerprint_invalid"))
		}
		claimAuthorization, err := readAuthorizationReceipt(ctx, source, claimAuthorizationRef)
		if err != nil {
			return nil, err
		}
		if err := validateMailboxAuthorization(
			claimAuthorization, attempt.Recipient.PrincipalRef,
			record.Envelope.ProjectRef, record.Envelope.Ref,
		); err != nil || claimAuthorization.RecordedAt().After(attempt.ClaimedAt) {
			return nil, invalid(errors.New("sqlite.mailbox_claim_authorization_invalid"))
		}
		deliveryPresent := deliveryRequest.Valid || deliveryFingerprint.Valid ||
			deliveryAuthorizationRef.Valid || deliveryRef.Valid || deliveredAt.Valid
		if deliveryPresent != (deliveryRequest.Valid && deliveryFingerprint.Valid &&
			deliveryAuthorizationRef.Valid && deliveryRef.Valid && deliveredAt.Valid) {
			return nil, invalid(errors.New("sqlite.mailbox_delivery_partial"))
		}
		if deliveryRef.Valid {
			if !validText(deliveryRequest.String) || !validText(deliveryFingerprint.String) ||
				!validText(deliveryRef.String) {
				return nil, invalid(errors.New("sqlite.mailbox_delivery_invalid"))
			}
			attempt.DeliveryRef = deliveryRef.String
			attempt.DeliveredAt = restoredTime(deliveredAt)
			if deliveryFingerprint.String != application.MailboxMutationFingerprint(
				application.MailboxMutationDeliver, attempt.Recipient.PrincipalRef,
				record.Envelope.ProjectRef, record.Envelope.GoalRef, record.Envelope.Ref,
				attempt.Recipient.WorkItemRef, attempt.Recipient.ExecutionRef, attempt.ClaimToken,
				attempt.Fence, "",
			) {
				return nil, invalid(errors.New("sqlite.mailbox_delivery_fingerprint_invalid"))
			}
			deliveryAuthorization, err := readAuthorizationReceipt(ctx, source, deliveryAuthorizationRef.String)
			if err != nil {
				return nil, err
			}
			if err := validateMailboxAuthorization(
				deliveryAuthorization, attempt.Recipient.PrincipalRef,
				record.Envelope.ProjectRef, record.Envelope.Ref,
			); err != nil || deliveryAuthorization.RecordedAt().After(attempt.DeliveredAt) ||
				attempt.DeliveredAt.Before(attempt.ClaimedAt) || !attempt.DeliveredAt.Before(attempt.LeaseUntil) {
				return nil, invalid(errors.New("sqlite.mailbox_delivery_authorization_invalid"))
			}
		}
		consumptionPresent := consumptionRequest.Valid || consumptionFingerprint.Valid ||
			consumptionAuthorizationRef.Valid || consumptionRef.Valid || consumedAt.Valid
		if consumptionPresent != (consumptionRequest.Valid && consumptionFingerprint.Valid &&
			consumptionAuthorizationRef.Valid && consumptionRef.Valid && consumedAt.Valid) ||
			(consumptionPresent && !deliveryPresent) {
			return nil, invalid(errors.New("sqlite.mailbox_consumption_partial"))
		}
		if consumptionRef.Valid {
			if !validText(consumptionRequest.String) || !validText(consumptionFingerprint.String) ||
				!validText(consumptionRef.String) {
				return nil, invalid(errors.New("sqlite.mailbox_consumption_invalid"))
			}
			attempt.ConsumptionRef = consumptionRef.String
			attempt.ConsumedAt = restoredTime(consumedAt)
			if consumptionFingerprint.String != application.MailboxMutationFingerprint(
				application.MailboxMutationConsume, attempt.Recipient.PrincipalRef,
				record.Envelope.ProjectRef, record.Envelope.GoalRef, record.Envelope.Ref,
				attempt.Recipient.WorkItemRef, attempt.Recipient.ExecutionRef, attempt.ClaimToken,
				attempt.Fence, "",
			) {
				return nil, invalid(errors.New("sqlite.mailbox_consumption_fingerprint_invalid"))
			}
			consumptionAuthorization, err := readAuthorizationReceipt(ctx, source, consumptionAuthorizationRef.String)
			if err != nil {
				return nil, err
			}
			if err := validateMailboxAuthorization(
				consumptionAuthorization, attempt.Recipient.PrincipalRef,
				record.Envelope.ProjectRef, record.Envelope.Ref,
			); err != nil || consumptionAuthorization.RecordedAt().After(attempt.ConsumedAt) ||
				attempt.ConsumedAt.Before(attempt.DeliveredAt) || !attempt.ConsumedAt.Before(attempt.LeaseUntil) {
				return nil, invalid(errors.New("sqlite.mailbox_consumption_authorization_invalid"))
			}
		}
		result = append(result, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return result, nil
}

func readMailboxAcknowledgement(
	ctx context.Context,
	source queryer,
	record application.MailboxRecord,
) (*application.MailboxAcknowledgement, error) {
	var ack application.MailboxAcknowledgement
	var authorizationRef, projectValue, principalValue, outcome string
	var expectedGoalRevision, expectedPlanGeneration, acknowledgedAt int64
	err := source.QueryRowContext(ctx, `
SELECT ref, request_ref, request_fingerprint, authorization_receipt_ref,
       action_ref, project_ref, expected_goal_revision, expected_plan_generation,
       recipient_principal_ref, outcome, effect_or_rework_ref, acked_at
FROM mailbox_delivery_acks WHERE mailbox_message_ref = ?`, record.Envelope.Ref.String()).Scan(
		&ack.Ref, &ack.RequestRef, &ack.RequestFingerprint, &authorizationRef,
		&ack.ActionRef, &projectValue, &expectedGoalRevision, &expectedPlanGeneration,
		&principalValue, &outcome, &ack.EffectOrReworkRef, &acknowledgedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	ack.MessageRef = record.Envelope.Ref
	ack.ProjectRef = record.Envelope.ProjectRef
	ack.GoalRef = record.Envelope.GoalRef
	ack.TargetPlanGeneration = record.Envelope.TargetPlanGeneration
	ack.ParentWorkItemRef = record.Envelope.ParentWorkItemRef
	ack.ChildWorkItemRef = record.Envelope.ChildWorkItemRef
	var refErr error
	if ack.Recipient.PrincipalRef, refErr = identity.NewPrincipalRef(principalValue); refErr != nil {
		return nil, invalid(refErr)
	}
	ack.Recipient.WorkItemRef = record.Envelope.Recipient.WorkItemRef
	ack.Recipient.ExecutionRef = record.Envelope.Recipient.ExecutionRef
	if projectValue != record.Envelope.ProjectRef.String() ||
		expectedGoalRevision <= 0 || expectedPlanGeneration < int64(record.Envelope.TargetPlanGeneration) {
		return nil, invalid(errors.New("sqlite.mailbox_ack_generation_invalid"))
	}
	ack.Outcome = application.MailboxOutcome(outcome)
	ack.AcknowledgedAt = time.Unix(0, acknowledgedAt).UTC()
	ack.AuthorizationReceipt, err = readAuthorizationReceipt(ctx, source, authorizationRef)
	if err != nil {
		return nil, err
	}
	if len(record.Attempts) == 0 {
		return nil, invalid(errors.New("sqlite.mailbox_ack_attempt_missing"))
	}
	last := record.Attempts[len(record.Attempts)-1]
	ack.Fence = last.Fence
	if last.ConsumptionRef == "" || ack.AuthorizationReceipt.RecordedAt().After(ack.AcknowledgedAt) ||
		ack.AcknowledgedAt.Before(last.ConsumedAt) {
		return nil, invalid(errors.New("sqlite.mailbox_ack_binding_invalid"))
	}
	kind := application.MailboxMutationAcknowledge
	if ack.Outcome == application.MailboxOutcomeBlocked {
		kind = application.MailboxMutationBlock
	}
	extra := strconv.FormatInt(expectedGoalRevision, 10) + "\x00" +
		strconv.FormatInt(expectedPlanGeneration, 10) + "\x00" + ack.EffectOrReworkRef
	if ack.RequestFingerprint != application.MailboxMutationFingerprint(
		kind, ack.Recipient.PrincipalRef, ack.ProjectRef, ack.GoalRef, ack.MessageRef,
		ack.Recipient.WorkItemRef, ack.Recipient.ExecutionRef, last.ClaimToken,
		ack.Fence, extra,
	) {
		return nil, invalid(errors.New("sqlite.mailbox_ack_fingerprint_invalid"))
	}
	return &ack, nil
}

func readMailboxRetirement(
	ctx context.Context,
	source queryer,
	record application.MailboxRecord,
) (*application.MailboxRetirement, error) {
	var retirement application.MailboxRetirement
	var messageValue, executionValue string
	var retiredAt int64
	err := source.QueryRowContext(ctx, `
SELECT mailbox_message_ref, action_ref, recipient_execution_ref, failure_code, retired_at
FROM mailbox_retirements WHERE mailbox_message_ref = ?`, record.Envelope.Ref.String()).Scan(
		&messageValue, &retirement.ActionRef, &executionValue, &retirement.FailureCode, &retiredAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	var refErr error
	if retirement.MessageRef, refErr = application.NewMailboxMessageRef(messageValue); refErr != nil {
		return nil, invalid(refErr)
	}
	if retirement.RecipientExecutionRef, refErr = goal.NewExecutionRef(executionValue); refErr != nil {
		return nil, invalid(refErr)
	}
	retirement.RetiredAt = time.Unix(0, retiredAt).UTC()
	return &retirement, nil
}

func readMailboxReplay(
	ctx context.Context,
	source queryer,
	request application.MailboxReplayRequest,
) (application.MailboxReplayRecord, bool, error) {
	locator, found, err := findMailboxReplayLocator(ctx, source, request)
	if err != nil || !found {
		return application.MailboxReplayRecord{}, found, err
	}
	if locator.requestFingerprint != request.RequestFingerprint || locator.principalRef != request.PrincipalRef.String() ||
		locator.projectRef != request.ProjectRef.String() || locator.goalRef != request.GoalRef.String() ||
		(request.MessageRef.String() != "" && locator.messageRef != request.MessageRef.String()) {
		return application.MailboxReplayRecord{}, false, conflict(errors.New("sqlite.mailbox_replay_conflict"))
	}
	record, err := readMailboxRecord(ctx, source, locator.messageRef)
	if err != nil {
		return application.MailboxReplayRecord{}, false, err
	}
	if err := validateRecoveredMailboxTerminalCause(ctx, source, record); err != nil {
		return application.MailboxReplayRecord{}, false, invalid(err)
	}
	replay := application.MailboxReplayRecord{}
	if request.Kind == application.MailboxMutationClaim &&
		(record.Acknowledgement != nil || record.Retirement != nil) {
		return application.MailboxReplayRecord{}, false,
			conflict(errors.New("sqlite.mailbox_claim_terminal"))
	}
	switch request.Kind {
	case application.MailboxMutationAdmit:
	case application.MailboxMutationClaim, application.MailboxMutationDeliver, application.MailboxMutationConsume:
		if locator.fence <= 0 {
			return application.MailboxReplayRecord{}, false, invalid(errors.New("sqlite.mailbox_replay_fence_invalid"))
		}
		index := -1
		for current := range record.Attempts {
			if record.Attempts[current].Fence == uint64(locator.fence) {
				index = current
				break
			}
		}
		if index < 0 {
			return application.MailboxReplayRecord{}, false, invalid(errors.New("sqlite.mailbox_replay_attempt_missing"))
		}
		if request.Kind == application.MailboxMutationClaim {
			if index != len(record.Attempts)-1 {
				return application.MailboxReplayRecord{}, false,
					conflict(errors.New("sqlite.mailbox_claim_superseded"))
			}
			record.Attempts = append([]application.MailboxDeliveryAttempt(nil), record.Attempts[:index+1]...)
			last := &record.Attempts[index]
			last.DeliveryRef = ""
			last.DeliveredAt = time.Time{}
			last.ConsumptionRef = ""
			last.ConsumedAt = time.Time{}
			record.Acknowledgement = nil
			record.Retirement = nil
			record.State = application.MailboxStateClaimed
			replay.Claim = application.MailboxClaim{Record: record, Attempt: *last}
			break
		}
		record.Attempts = append([]application.MailboxDeliveryAttempt(nil), record.Attempts[:index+1]...)
		last := &record.Attempts[index]
		record.Acknowledgement = nil
		record.Retirement = nil
		switch request.Kind {
		case application.MailboxMutationDeliver:
			last.ConsumptionRef = ""
			last.ConsumedAt = time.Time{}
			record.State = application.MailboxStateDelivered
		case application.MailboxMutationConsume:
			record.State = application.MailboxStateConsumed
		}
	case application.MailboxMutationAcknowledge, application.MailboxMutationBlock:
		if record.Acknowledgement == nil || string(record.Acknowledgement.Outcome) != locator.outcome {
			return application.MailboxReplayRecord{}, false, invalid(errors.New("sqlite.mailbox_replay_ack_missing"))
		}
		replay.Acknowledgement = *record.Acknowledgement
	default:
		return application.MailboxReplayRecord{}, false, invalid(errors.New("sqlite.mailbox_replay_kind_invalid"))
	}
	replay.Record = record
	return replay, true, nil
}

func findMailboxReplayLocator(
	ctx context.Context,
	source queryer,
	request application.MailboxReplayRequest,
) (mailboxReplayLocator, bool, error) {
	var locator mailboxReplayLocator
	var row *sql.Row
	switch request.Kind {
	case application.MailboxMutationAdmit:
		row = source.QueryRowContext(ctx, `
SELECT mailbox_message_ref, request_fingerprint, project_ref, goal_ref, source_principal_ref, 0, ''
FROM mailbox_admission_receipts
WHERE source_principal_ref = ? AND project_ref = ? AND request_ref = ?`,
			request.PrincipalRef.String(), request.ProjectRef.String(), request.RequestRef)
	case application.MailboxMutationClaim:
		row = source.QueryRowContext(ctx, `
SELECT attempt.mailbox_message_ref, attempt.claim_request_fingerprint,
       attempt.project_ref, envelope.goal_ref, attempt.recipient_principal_ref, attempt.fence, ''
FROM mailbox_delivery_attempts attempt
JOIN mailbox_envelopes envelope ON envelope.ref = attempt.mailbox_message_ref
WHERE attempt.recipient_principal_ref = ? AND attempt.project_ref = ? AND attempt.claim_request_ref = ?`,
			request.PrincipalRef.String(), request.ProjectRef.String(), request.RequestRef)
	case application.MailboxMutationDeliver:
		row = source.QueryRowContext(ctx, `
SELECT attempt.mailbox_message_ref, attempt.delivery_request_fingerprint,
       attempt.project_ref, envelope.goal_ref, attempt.recipient_principal_ref, attempt.fence, ''
FROM mailbox_delivery_attempts attempt
JOIN mailbox_envelopes envelope ON envelope.ref = attempt.mailbox_message_ref
WHERE attempt.recipient_principal_ref = ? AND attempt.project_ref = ? AND attempt.delivery_request_ref = ?`,
			request.PrincipalRef.String(), request.ProjectRef.String(), request.RequestRef)
	case application.MailboxMutationConsume:
		row = source.QueryRowContext(ctx, `
SELECT attempt.mailbox_message_ref, attempt.consumption_request_fingerprint,
       attempt.project_ref, envelope.goal_ref, attempt.recipient_principal_ref, attempt.fence, ''
FROM mailbox_delivery_attempts attempt
JOIN mailbox_envelopes envelope ON envelope.ref = attempt.mailbox_message_ref
WHERE attempt.recipient_principal_ref = ? AND attempt.project_ref = ? AND attempt.consumption_request_ref = ?`,
			request.PrincipalRef.String(), request.ProjectRef.String(), request.RequestRef)
	case application.MailboxMutationAcknowledge, application.MailboxMutationBlock:
		wantOutcome := string(application.MailboxOutcomeAcknowledged)
		if request.Kind == application.MailboxMutationBlock {
			wantOutcome = string(application.MailboxOutcomeBlocked)
		}
		row = source.QueryRowContext(ctx, `
SELECT ack.mailbox_message_ref, ack.request_fingerprint, ack.project_ref, envelope.goal_ref,
       ack.recipient_principal_ref, 0, ack.outcome
FROM mailbox_delivery_acks ack
JOIN mailbox_envelopes envelope ON envelope.ref = ack.mailbox_message_ref
WHERE ack.recipient_principal_ref = ? AND ack.project_ref = ? AND ack.request_ref = ? AND ack.outcome = ?`,
			request.PrincipalRef.String(), request.ProjectRef.String(), request.RequestRef, wantOutcome)
	default:
		return mailboxReplayLocator{}, false, invalid(errors.New("sqlite.mailbox_replay_kind_invalid"))
	}
	err := row.Scan(
		&locator.messageRef, &locator.requestFingerprint, &locator.projectRef,
		&locator.goalRef, &locator.principalRef, &locator.fence, &locator.outcome,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return mailboxReplayLocator{}, false, nil
	}
	if err != nil {
		return mailboxReplayLocator{}, false, mapDatabaseError(err)
	}
	return locator, true, nil
}
