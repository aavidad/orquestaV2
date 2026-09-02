package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"reflect"
	"time"

	"orquesta/internal/application"
)

func (repository *Repository) ExpiredAgentLaunchContinuationSourceV41(
	ctx context.Context,
	reconciliationAuthorityRef string,
) (application.ExpiredAgentLaunchContinuationSourceV41, bool, error) {
	if !validText(reconciliationAuthorityRef) {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false,
			invalid(errors.New("sqlite.expired_agent_launch_continuation_source_ref_invalid"))
	}
	tx, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false, err
	}
	defer func() { _ = tx.Rollback() }()
	authority, found, err := readTerminalReconciliationAuthorityByRef(ctx, tx, reconciliationAuthorityRef)
	if err != nil || !found {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, found, err
	}
	var jobState, lastError string
	var claimToken, claimedBy sql.NullString
	var claimedUntil sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT state,last_error_code,claim_token,claimed_by,claimed_until
FROM agent_launch_reconciliation_jobs WHERE authority_ref=?`, authority.Ref).Scan(
		&jobState, &lastError, &claimToken, &claimedBy, &claimedUntil,
	)
	if err != nil {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false, mapDatabaseError(err)
	}
	if jobState != "pending" || lastError != "agent.launch_reconciliation_pending" ||
		claimToken.Valid || claimedBy.Valid || claimedUntil.Valid {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false,
			conflict(errors.New("sqlite.expired_agent_launch_continuation_job_not_requeued"))
	}
	rows, err := tx.QueryContext(ctx, `SELECT ref,authority_ref,request_fingerprint,
original_effect_attempt_ref,job_fence,delivery_attempt,claim_token,worker_ref,started_at,claim_lease_until
FROM agent_launch_reconciliation_attempts WHERE authority_ref=? ORDER BY started_at,ref LIMIT 2`, authority.Ref)
	if err != nil {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false, mapDatabaseError(err)
	}
	var attempts []application.TerminalAgentLaunchReconciliationAttempt
	for rows.Next() {
		var value application.TerminalAgentLaunchReconciliationAttempt
		var fence, delivery, started, lease int64
		if scanErr := rows.Scan(&value.Ref, &value.AuthorityRef, &value.RequestFingerprint,
			&value.OriginalEffectAttemptRef, &fence, &delivery, &value.ClaimToken,
			&value.WorkerRef, &started, &lease); scanErr != nil {
			_ = rows.Close()
			return application.ExpiredAgentLaunchContinuationSourceV41{}, false, mapDatabaseError(scanErr)
		}
		if fence <= 0 || delivery <= 0 || lease <= started {
			_ = rows.Close()
			return application.ExpiredAgentLaunchContinuationSourceV41{}, false,
				invalid(errors.New("sqlite.expired_agent_launch_continuation_attempt_invalid"))
		}
		value.JobFence, value.DeliveryAttempt = uint64(fence), uint64(delivery)
		value.StartedAt, value.ClaimLeaseUntil = time.Unix(0, started).UTC(), time.Unix(0, lease).UTC()
		attempts = append(attempts, value)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false, mapDatabaseError(err)
	}
	if err := rows.Close(); err != nil {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false, mapDatabaseError(err)
	}
	if len(attempts) == 0 {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false,
			conflict(errors.New("sqlite.expired_agent_launch_continuation_attempt_missing"))
	}
	// Only the first requeued attempt is authoritative. Later retries are not
	// copied into V41 and cannot replace its causal identity.
	reconciliationAttempt := attempts[0]
	if reconciliationAttempt.AuthorityRef != authority.Ref ||
		reconciliationAttempt.RequestFingerprint != authority.RequestFingerprint ||
		reconciliationAttempt.OriginalEffectAttemptRef != authority.EffectAttemptRef ||
		reconciliationAttempt.JobFence <= authority.ActionFence {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false,
			conflict(errors.New("sqlite.expired_agent_launch_continuation_attempt_conflict"))
	}
	record, err := readGoalRecord(ctx, tx, authority.GoalRef.String())
	if err != nil {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false, err
	}
	intent, intentFound := effectIntentByRefSQLite(record.EffectIntents, authority.EffectIntentRef)
	attempt, attemptFound := effectAttemptByRefSQLite(record.EffectAttempts, authority.EffectAttemptRef)
	approval, approvalFound := exactEffectApprovalSQLite(record.EffectApprovals, attempt.ApprovalRef)
	if !intentFound || !attemptFound || !approvalFound {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false,
			conflict(errors.New("sqlite.expired_agent_launch_continuation_history_missing"))
	}
	var availableAt int64
	if err := tx.QueryRowContext(ctx, `SELECT available_at FROM outbox WHERE ref=?`, authority.ActionRef).Scan(&availableAt); err != nil {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false, mapDatabaseError(err)
	}
	budget, budgetActive, err := readActiveBudgetReservation(ctx, tx, authority.ActionRef)
	if err != nil {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false, err
	}
	capacity, placement, capacityActive, err := leerReservaCapacidadAccion(ctx, tx, authority.ActionRef)
	if err != nil {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false, err
	}
	if !budgetActive || !capacityActive || capacity.State != application.AgentCapacityQuarantined {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false,
			conflict(errors.New("sqlite.expired_agent_launch_continuation_reservation_missing"))
	}
	claim := application.ActionClaim{
		Action: application.ActionRecord{
			Ref: authority.ActionRef, Kind: application.ActionLaunchAgent,
			GoalRef: authority.GoalRef, WorkItemRef: authority.WorkItemRef,
			ExecutionRef: authority.ExecutionRef, EffectIntentRef: authority.EffectIntentRef,
			EffectIntent: intent, PlanGeneration: authority.PlanGeneration,
			WorkItemGeneration: authority.WorkItemGeneration,
			AvailableAt:        time.Unix(0, availableAt).UTC(),
		},
		Token: reconciliationAttempt.ClaimToken, WorkerRef: reconciliationAttempt.WorkerRef,
		DeliveryAttempt: reconciliationAttempt.DeliveryAttempt, Fence: reconciliationAttempt.JobFence,
		Disposition:              application.ActionClaimDispositionReconcileTerminalLaunch,
		RecoveryEffectAttemptRef: authority.EffectAttemptRef,
		BudgetReservationRef:     budget.Ref, BudgetReservation: budget,
		CapacityReservation: capacity, ReferenciaColocacion: placement,
		EffectApproval: approval, LeaseUntil: reconciliationAttempt.ClaimLeaseUntil,
		TerminalReconciliationRef:         authority.Ref,
		TerminalReconciliationFingerprint: authority.RequestFingerprint,
	}
	if _, selectErr := application.SelectAgentLaunchRecoveryAttempt(record, claim); selectErr != nil {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false, conflict(selectErr)
	}
	if err := commit(tx); err != nil {
		return application.ExpiredAgentLaunchContinuationSourceV41{}, false, err
	}
	return application.ExpiredAgentLaunchContinuationSourceV41{
		Authority: authority, ReconciliationAttempt: reconciliationAttempt, Claim: claim,
	}, true, nil
}

func (repository *Repository) RecordExpiredAgentLaunchContinuationV41(
	ctx context.Context,
	record application.ExpiredAgentLaunchContinuationRecordV41,
) (application.ExpiredAgentLaunchContinuationRecordV41, bool, error) {
	record.PreparedAt = record.PreparedAt.UTC().Round(0)
	if validateExpiredAgentLaunchContinuationRecordV41(record) != nil {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, false,
			invalid(errors.New("sqlite.expired_agent_launch_continuation_invalid"))
	}
	tx, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, false, err
	}
	defer func() { _ = tx.Rollback() }()
	if replay, found, readErr := readExpiredAgentLaunchContinuationV41(
		ctx, tx, record.ReconciliationAuthorityRef,
	); readErr != nil {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, false, readErr
	} else if found {
		if !reflect.DeepEqual(replay, record) {
			return application.ExpiredAgentLaunchContinuationRecordV41{}, false,
				conflict(errors.New("sqlite.expired_agent_launch_continuation_replay_conflict"))
		}
		if err := commit(tx); err != nil {
			return application.ExpiredAgentLaunchContinuationRecordV41{}, false, err
		}
		return cloneExpiredAgentLaunchContinuationRecordV41(replay), false, nil
	}
	if err := requireExpiredAgentLaunchContinuationSourceV41(ctx, tx, record); err != nil {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, false, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO agent_launch_expired_continuation_subjects(
ref,reconciliation_authority_ref,project_ref,goal_ref,work_item_ref,execution_ref,action_ref,
effect_intent_ref,effect_intent_digest,effect_attempt_ref,plan_generation,work_item_generation,
action_fence,request_key_sha256,original_request_sha256,amv_launch_ref,amv_execution_ref,amv_run_ref,
amv_fence,amv_generation,amv_cid,amv_identity_sha256,source_digest,
profile_descriptor_bytes,profile_descriptor_bytes_sha256,plan_bytes,plan_bytes_sha256,
concession_bytes,concession_bytes_sha256,manifest_bytes,manifest_bytes_sha256,manifest_sha256,prepared_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		record.SubjectRef, record.ReconciliationAuthorityRef, record.ProjectRef, record.GoalRef,
		record.WorkItemRef, record.ExecutionRef, record.ActionRef, record.EffectIntentRef,
		record.EffectIntentDigest, record.EffectAttemptRef, int64(record.PlanGeneration),
		int64(record.WorkItemGeneration), int64(record.ActionFence), record.RequestKeySHA256,
		record.OriginalRequestSHA256, record.AMVLaunchRef, record.AMVExecutionRef, record.AMVRunRef,
		int64(record.AMVFence), int64(record.AMVGeneration), int64(record.AMVCID),
		record.AMVIdentitySHA256, record.SourceDigest, record.ProfileDescriptorBytes,
		record.ProfileDescriptorSHA256, record.PlanBytes, record.PlanSHA256,
		record.ConcessionBytes, record.ConcessionSHA256, record.ManifestBytes,
		record.ManifestBytesSHA256, record.ManifestSHA256, requiredTime(record.PreparedAt))
	if err != nil {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, false, mapDatabaseError(err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO agent_launch_expired_continuation_authorities(
authority_ref,subject_ref,reconciliation_attempt_ref,manifest_sha256,authority_sha256,
authority_bytes,authority_bytes_sha256,key_id,key_epoch,trust_revision,public_key,signature,
issued_unix_ms,expires_unix_ms,admitted_unix_ms) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		record.AuthorityRef, record.SubjectRef, record.ReconciliationAttemptRef, record.ManifestSHA256,
		record.AuthoritySHA256, record.AuthorityBytes, record.AuthorityBytesSHA256, record.KeyID,
		int64(record.KeyEpoch), int64(record.TrustRevision), record.PublicKey, record.Signature,
		int64(record.IssuedUnixMS), int64(record.ExpiresUnixMS), int64(record.AdmittedUnixMS))
	if err != nil {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, false, mapDatabaseError(err)
	}
	if err := validateRecoveryV41ExpiredAgentLaunchContinuation(ctx, tx); err != nil {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, false, err
	}
	if err := commit(tx); err != nil {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, false, err
	}
	return cloneExpiredAgentLaunchContinuationRecordV41(record), true, nil
}

func (repository *Repository) ExpiredAgentLaunchContinuationV41(
	ctx context.Context,
	reconciliationAuthorityRef string,
) (application.ExpiredAgentLaunchContinuationRecordV41, bool, error) {
	tx, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, false, err
	}
	defer func() { _ = tx.Rollback() }()
	record, found, err := readExpiredAgentLaunchContinuationV41(ctx, tx, reconciliationAuthorityRef)
	if err != nil {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, false, err
	}
	if err := commit(tx); err != nil {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, false, err
	}
	return cloneExpiredAgentLaunchContinuationRecordV41(record), found, nil
}

func requireExpiredAgentLaunchContinuationSourceV41(
	ctx context.Context,
	tx *sql.Tx,
	record application.ExpiredAgentLaunchContinuationRecordV41,
) error {
	var count int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*)
FROM agent_launch_reconciliation_authorities authority
JOIN agent_launch_reconciliation_attempts attempt ON attempt.ref=?
 AND attempt.authority_ref=authority.ref
 AND attempt.original_effect_attempt_ref=authority.effect_attempt_ref
 AND attempt.request_fingerprint=authority.request_fingerprint
WHERE authority.ref=? AND authority.project_ref=? AND authority.goal_ref=?
 AND authority.work_item_ref=? AND authority.execution_ref=? AND authority.action_ref=?
 AND authority.effect_intent_ref=? AND authority.effect_intent_digest=?
 AND authority.effect_attempt_ref=? AND authority.plan_generation=?
 AND authority.work_item_generation=? AND authority.action_fence=?`,
		record.ReconciliationAttemptRef, record.ReconciliationAuthorityRef, record.ProjectRef,
		record.GoalRef, record.WorkItemRef, record.ExecutionRef, record.ActionRef,
		record.EffectIntentRef, record.EffectIntentDigest, record.EffectAttemptRef,
		int64(record.PlanGeneration), int64(record.WorkItemGeneration), int64(record.ActionFence)).Scan(&count)
	if err != nil {
		return mapDatabaseError(err)
	}
	if count != 1 {
		return conflict(errors.New("sqlite.expired_agent_launch_continuation_source_conflict"))
	}
	return nil
}

func readExpiredAgentLaunchContinuationV41(
	ctx context.Context,
	source queryer,
	reconciliationAuthorityRef string,
) (application.ExpiredAgentLaunchContinuationRecordV41, bool, error) {
	var record application.ExpiredAgentLaunchContinuationRecordV41
	var planGeneration, workItemGeneration, actionFence int64
	var amvFence, amvGeneration, amvCID int64
	var preparedAt int64
	var keyEpoch, trustRevision, issued, expires, admitted int64
	err := source.QueryRowContext(ctx, `SELECT
subject.ref,subject.reconciliation_authority_ref,authority.reconciliation_attempt_ref,
subject.project_ref,subject.goal_ref,subject.work_item_ref,subject.execution_ref,subject.action_ref,
subject.effect_intent_ref,subject.effect_intent_digest,subject.effect_attempt_ref,
subject.plan_generation,subject.work_item_generation,subject.action_fence,
subject.request_key_sha256,subject.original_request_sha256,subject.amv_launch_ref,
subject.amv_execution_ref,subject.amv_run_ref,subject.amv_fence,subject.amv_generation,
subject.amv_cid,subject.amv_identity_sha256,subject.source_digest,
subject.profile_descriptor_bytes,subject.profile_descriptor_bytes_sha256,
subject.plan_bytes,subject.plan_bytes_sha256,subject.concession_bytes,subject.concession_bytes_sha256,
subject.manifest_bytes,subject.manifest_bytes_sha256,subject.manifest_sha256,subject.prepared_at,
authority.authority_ref,authority.authority_sha256,authority.authority_bytes,
authority.authority_bytes_sha256,authority.key_id,authority.key_epoch,authority.trust_revision,
authority.public_key,authority.signature,authority.issued_unix_ms,authority.expires_unix_ms,
authority.admitted_unix_ms
FROM agent_launch_expired_continuation_subjects subject
JOIN agent_launch_expired_continuation_authorities authority ON authority.subject_ref=subject.ref
WHERE subject.reconciliation_authority_ref=?`, reconciliationAuthorityRef).Scan(
		&record.SubjectRef, &record.ReconciliationAuthorityRef, &record.ReconciliationAttemptRef,
		&record.ProjectRef, &record.GoalRef, &record.WorkItemRef, &record.ExecutionRef, &record.ActionRef,
		&record.EffectIntentRef, &record.EffectIntentDigest, &record.EffectAttemptRef,
		&planGeneration, &workItemGeneration, &actionFence, &record.RequestKeySHA256,
		&record.OriginalRequestSHA256, &record.AMVLaunchRef, &record.AMVExecutionRef, &record.AMVRunRef,
		&amvFence, &amvGeneration, &amvCID, &record.AMVIdentitySHA256, &record.SourceDigest,
		&record.ProfileDescriptorBytes, &record.ProfileDescriptorSHA256, &record.PlanBytes,
		&record.PlanSHA256, &record.ConcessionBytes, &record.ConcessionSHA256, &record.ManifestBytes,
		&record.ManifestBytesSHA256, &record.ManifestSHA256, &preparedAt, &record.AuthorityRef,
		&record.AuthoritySHA256, &record.AuthorityBytes, &record.AuthorityBytesSHA256, &record.KeyID,
		&keyEpoch, &trustRevision, &record.PublicKey, &record.Signature, &issued, &expires, &admitted)
	if errors.Is(err, sql.ErrNoRows) {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, false, nil
	}
	if err != nil {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, false, mapDatabaseError(err)
	}
	if planGeneration <= 0 || workItemGeneration <= 0 || actionFence <= 0 || amvFence <= 0 ||
		amvGeneration <= 0 || amvCID < 3 || keyEpoch <= 0 || trustRevision <= 0 || issued <= 0 ||
		expires <= issued || admitted < issued || admitted >= expires {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, false,
			invalid(errors.New("sqlite.expired_agent_launch_continuation_counters_invalid"))
	}
	record.PlanGeneration, record.WorkItemGeneration, record.ActionFence =
		uint64(planGeneration), uint64(workItemGeneration), uint64(actionFence)
	record.AMVFence, record.AMVGeneration, record.AMVCID = uint64(amvFence), uint64(amvGeneration), uint32(amvCID)
	record.PreparedAt = time.Unix(0, preparedAt).UTC()
	record.KeyEpoch, record.TrustRevision = uint64(keyEpoch), uint64(trustRevision)
	record.IssuedUnixMS, record.ExpiresUnixMS, record.AdmittedUnixMS = uint64(issued), uint64(expires), uint64(admitted)
	if err := validateExpiredAgentLaunchContinuationRecordV41(record); err != nil {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, false, invalid(err)
	}
	return record, true, nil
}

func validateExpiredAgentLaunchContinuationRecordV41(
	record application.ExpiredAgentLaunchContinuationRecordV41,
) error {
	for _, value := range []string{
		record.SubjectRef, record.ReconciliationAuthorityRef, record.ReconciliationAttemptRef,
		record.ProjectRef, record.GoalRef, record.WorkItemRef, record.ExecutionRef, record.ActionRef,
		record.EffectIntentRef, record.EffectAttemptRef, record.AMVLaunchRef, record.AMVExecutionRef,
		record.AMVRunRef, record.AuthorityRef, record.KeyID,
	} {
		if !validText(value) {
			return errors.New("sqlite.expired_agent_launch_continuation_ref_invalid")
		}
	}
	for _, value := range []string{
		record.EffectIntentDigest, record.RequestKeySHA256, record.OriginalRequestSHA256,
		record.AMVIdentitySHA256, record.SourceDigest, record.ProfileDescriptorSHA256,
		record.PlanSHA256, record.ConcessionSHA256, record.ManifestBytesSHA256,
		record.ManifestSHA256, record.AuthoritySHA256, record.AuthorityBytesSHA256,
	} {
		if !validDigest(value) {
			return errors.New("sqlite.expired_agent_launch_continuation_digest_invalid")
		}
	}
	if record.PlanGeneration == 0 || record.WorkItemGeneration == 0 || record.ActionFence == 0 ||
		record.PlanGeneration > math.MaxInt64 || record.WorkItemGeneration > math.MaxInt64 ||
		record.ActionFence > math.MaxInt64 || record.AMVFence > math.MaxInt64 ||
		record.AMVGeneration > math.MaxInt64 || record.KeyEpoch > math.MaxInt64 ||
		record.TrustRevision > math.MaxInt64 || record.IssuedUnixMS > math.MaxInt64 ||
		record.ExpiresUnixMS > math.MaxInt64 || record.AdmittedUnixMS > math.MaxInt64 ||
		record.ActionFence != record.AMVFence || record.PlanGeneration != record.AMVGeneration || record.AMVCID < 3 ||
		len(record.ProfileDescriptorBytes) == 0 || len(record.ProfileDescriptorBytes) > 65_536 ||
		len(record.PlanBytes) == 0 || len(record.PlanBytes) > 65_536 ||
		len(record.ConcessionBytes) == 0 || len(record.ConcessionBytes) > 65_536 ||
		len(record.ManifestBytes) == 0 || len(record.ManifestBytes) > 131_072 ||
		len(record.AuthorityBytes) == 0 || len(record.AuthorityBytes) > 131_072 ||
		len(record.PublicKey) != 32 || len(record.Signature) != 64 || record.PreparedAt.IsZero() ||
		record.KeyEpoch == 0 || record.TrustRevision == 0 || record.IssuedUnixMS == 0 ||
		record.ExpiresUnixMS <= record.IssuedUnixMS || record.ExpiresUnixMS-record.IssuedUnixMS > 300_000 ||
		record.AdmittedUnixMS < record.IssuedUnixMS || record.AdmittedUnixMS >= record.ExpiresUnixMS {
		return errors.New("sqlite.expired_agent_launch_continuation_shape_invalid")
	}
	if continuationBytesSHA256(record.ProfileDescriptorBytes) != record.ProfileDescriptorSHA256 ||
		continuationBytesSHA256(record.PlanBytes) != record.PlanSHA256 ||
		continuationBytesSHA256(record.ConcessionBytes) != record.ConcessionSHA256 ||
		continuationBytesSHA256(record.ManifestBytes) != record.ManifestBytesSHA256 ||
		continuationBytesSHA256(record.AuthorityBytes) != record.AuthorityBytesSHA256 {
		return errors.New("sqlite.expired_agent_launch_continuation_bytes_invalid")
	}
	return nil
}

func cloneExpiredAgentLaunchContinuationRecordV41(
	record application.ExpiredAgentLaunchContinuationRecordV41,
) application.ExpiredAgentLaunchContinuationRecordV41 {
	record.ProfileDescriptorBytes = append([]byte(nil), record.ProfileDescriptorBytes...)
	record.PlanBytes = append([]byte(nil), record.PlanBytes...)
	record.ConcessionBytes = append([]byte(nil), record.ConcessionBytes...)
	record.ManifestBytes = append([]byte(nil), record.ManifestBytes...)
	record.AuthorityBytes = append([]byte(nil), record.AuthorityBytes...)
	record.PublicKey = append([]byte(nil), record.PublicKey...)
	record.Signature = append([]byte(nil), record.Signature...)
	return record
}

var _ application.ExpiredAgentLaunchContinuationStoreV41 = (*Repository)(nil)
var _ application.ExpiredAgentLaunchContinuationSourceStoreV41 = (*Repository)(nil)
