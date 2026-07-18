package sqlite

import (
	"context"
	"errors"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
)

func readBudgetEnvelopesForGoal(ctx context.Context, source queryer, projectRef, goalRef string) ([]governance.BudgetEnvelope, error) {
	refs, err := readSingleColumn(ctx, source, `
SELECT envelope.ref FROM budget_envelopes envelope WHERE (envelope.scope='goal' AND envelope.subject_ref=?)
 OR ((envelope.scope='project' AND envelope.subject_ref=?) OR envelope.scope='deployment') AND EXISTS
 (SELECT 1 FROM budget_envelopes goal_envelope WHERE goal_envelope.scope='goal'
  AND goal_envelope.subject_ref=? AND goal_envelope.policy_hash=envelope.policy_hash)
ORDER BY envelope.policy_hash,envelope.scope,envelope.subject_ref,envelope.ref`, goalRef, projectRef, goalRef)
	if err != nil {
		return nil, err
	}
	return readLedger(refs, func(ref string) (governance.BudgetEnvelope, error) { return readBudgetEnvelope(ctx, source, ref) })
}

func readBudgetReservationsForGoal(ctx context.Context, source queryer, goalRef string) ([]governance.BudgetReservation, error) {
	refs, err := readSingleColumn(ctx, source, `SELECT ref FROM budget_reservations WHERE goal_ref=? ORDER BY reserved_at,ref`, goalRef)
	if err != nil {
		return nil, err
	}
	return readLedger(refs, func(ref string) (governance.BudgetReservation, error) { return readBudgetReservation(ctx, source, ref) })
}

func readBudgetSettlementsForGoal(ctx context.Context, source queryer, goalRef string) ([]governance.BudgetSettlement, error) {
	rows, err := source.QueryContext(ctx, `
SELECT settlement.ref,settlement.reservation_ref,reserved_tokens,reserved_money_micros,reserved_currency,reserved_active_time_ns,reserved_process_slots,reserved_disk_bytes,
 observed_tokens,observed_money_micros,observed_currency,observed_active_time_ns,observed_process_slots,observed_disk_bytes,observed_known,observed_quality,
 charged_tokens,charged_money_micros,charged_currency,charged_active_time_ns,charged_process_slots,charged_disk_bytes,
 released_tokens,released_money_micros,released_currency,released_active_time_ns,released_process_slots,released_disk_bytes,
 overrun_tokens,overrun_money_micros,overrun_currency,overrun_active_time_ns,overrun_process_slots,overrun_disk_bytes,settled_at
FROM budget_settlements settlement JOIN budget_reservations reservation ON reservation.ref=settlement.reservation_ref
WHERE reservation.goal_ref=? ORDER BY settled_at,settlement.ref`, goalRef)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []governance.BudgetSettlement
	for rows.Next() {
		var v governance.BudgetSettlement
		var rc, oc, cc, lc, xc string
		var known, at int64
		if err := rows.Scan(&v.Ref, &v.ReservationRef, &v.Reserved.Tokens, &v.Reserved.MoneyMicros, &rc, &v.Reserved.ActiveTimeNS, &v.Reserved.ProcessSlots, &v.Reserved.DiskBytes, &v.Observed.Resources.Tokens, &v.Observed.Resources.MoneyMicros, &oc, &v.Observed.Resources.ActiveTimeNS, &v.Observed.Resources.ProcessSlots, &v.Observed.Resources.DiskBytes, &known, &v.Observed.Quality, &v.Charged.Tokens, &v.Charged.MoneyMicros, &cc, &v.Charged.ActiveTimeNS, &v.Charged.ProcessSlots, &v.Charged.DiskBytes, &v.Released.Tokens, &v.Released.MoneyMicros, &lc, &v.Released.ActiveTimeNS, &v.Released.ProcessSlots, &v.Released.DiskBytes, &v.Overrun.Tokens, &v.Overrun.MoneyMicros, &xc, &v.Overrun.ActiveTimeNS, &v.Overrun.ProcessSlots, &v.Overrun.DiskBytes, &at); err != nil {
			return nil, mapDatabaseError(err)
		}
		v.Reserved.Currency = governance.Currency(rc)
		v.Observed.Resources.Currency = governance.Currency(oc)
		v.Observed.Known = governance.ResourceDimensions(known)
		v.Charged.Currency = governance.Currency(cc)
		v.Released.Currency = governance.Currency(lc)
		v.Overrun.Currency = governance.Currency(xc)
		v.SettledAt = time.Unix(0, at).UTC()
		if err := governance.ValidateBudgetSettlement(v); err != nil {
			return nil, invalid(err)
		}
		result = append(result, v)
	}
	return result, mapDatabaseError(rows.Err())
}

func readWorkItemAuthoritiesForGoal(ctx context.Context, source queryer, goalRef string) ([]application.WorkItemAuthority, error) {
	rows, err := source.QueryContext(ctx, `SELECT work_item_ref,principal_ref,permission,source,authorization_receipt_ref,recorded_at FROM work_item_authorities WHERE goal_ref=? ORDER BY work_item_ref`, goalRef)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []application.WorkItemAuthority
	for rows.Next() {
		var value application.WorkItemAuthority
		var work, principal, permission, approvalSource, receipt string
		var at int64
		if err := rows.Scan(&work, &principal, &permission, &approvalSource, &receipt, &at); err != nil {
			return nil, mapDatabaseError(err)
		}
		var err error
		value.WorkItemRef, err = goal.NewWorkItemRef(work)
		if err != nil {
			return nil, invalid(err)
		}
		value.PrincipalRef, err = identity.NewPrincipalRef(principal)
		if err != nil {
			return nil, invalid(err)
		}
		value.Permission = identity.Permission(permission)
		value.Source = application.EffectApprovalSource(approvalSource)
		value.RecordedAt = time.Unix(0, at).UTC()
		value.AuthorizationReceipt, err = readAuthorizationReceipt(ctx, source, receipt)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, mapDatabaseError(rows.Err())
}

func readEffectIntentsForGoal(ctx context.Context, source queryer, goalRef string) ([]application.EffectIntent, error) {
	refs, err := readSingleColumn(ctx, source, `SELECT ref FROM effect_intents WHERE goal_ref=? ORDER BY created_at,ref`, goalRef)
	if err != nil {
		return nil, err
	}
	return readLedger(refs, func(ref string) (application.EffectIntent, error) { return readEffectIntent(ctx, source, ref) })
}

func readEffectApprovalsForGoal(ctx context.Context, source queryer, goalRef string) ([]application.EffectApproval, error) {
	refs, err := readSingleColumn(ctx, source, `SELECT ref FROM effect_approvals WHERE goal_ref=? ORDER BY decided_at,ref`, goalRef)
	if err != nil {
		return nil, err
	}
	return readLedger(refs, func(ref string) (application.EffectApproval, error) {
		value, found, err := readEffectApprovalByRef(ctx, source, ref)
		if err == nil && !found {
			err = invalid(errors.New("sqlite.effect_approval_missing"))
		}
		return value, err
	})
}

func readEffectAttemptsForGoal(ctx context.Context, source queryer, goalRef string) ([]application.EffectAttempt, error) {
	rows, err := source.QueryContext(ctx, `SELECT action_ref,action_fence FROM effect_attempts WHERE goal_ref=? ORDER BY started_at,ref`, goalRef)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	type key struct {
		action string
		fence  uint64
	}
	var keys []key
	for rows.Next() {
		var k key
		var fence int64
		if err := rows.Scan(&k.action, &fence); err != nil || fence <= 0 {
			if err == nil {
				err = errors.New("sqlite.effect_attempt_fence_invalid")
			}
			return nil, invalid(err)
		}
		k.fence = uint64(fence)
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	result := make([]application.EffectAttempt, 0, len(keys))
	for _, k := range keys {
		value, found, err := readEffectAttemptByFence(ctx, source, k.action, k.fence)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, invalid(errors.New("sqlite.effect_attempt_missing"))
		}
		result = append(result, value)
	}
	return result, nil
}

func readEffectReceiptsForGoal(ctx context.Context, source queryer, goalRef string) ([]application.EffectReceipt, error) {
	rows, err := source.QueryContext(ctx, `
SELECT ref,intent_ref,intent_digest,approval_ref,attempt_ref,project_ref,goal_ref,work_item_ref,execution_ref,plan_generation,app_spec_generation,spec_hash,actor_ref,
 action_ref,action_fence,idempotency_key,external_ref,status,usage_tokens,usage_money_micros,usage_currency,usage_active_time_ns,usage_process_slots,usage_disk_bytes,usage_known,usage_quality,confirmed_at
FROM effect_receipts WHERE goal_ref=? ORDER BY confirmed_at,ref`, goalRef)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []application.EffectReceipt
	for rows.Next() {
		var v application.EffectReceipt
		var project, storedGoal, work, execution, actor, currency string
		var plan, appSpec, fence, known, at int64
		if err := rows.Scan(&v.Ref, &v.IntentRef, &v.IntentDigest, &v.ApprovalRef, &v.AttemptRef, &project, &storedGoal, &work, &execution, &plan, &appSpec, &v.Subject.SpecHash, &actor, &v.ActionRef, &fence, &v.IdempotencyKey, &v.ExternalRef, &v.Status, &v.Usage.Resources.Tokens, &v.Usage.Resources.MoneyMicros, &currency, &v.Usage.Resources.ActiveTimeNS, &v.Usage.Resources.ProcessSlots, &v.Usage.Resources.DiskBytes, &known, &v.Usage.Quality, &at); err != nil {
			return nil, mapDatabaseError(err)
		}
		if err := restoreEffectSubject(&v.Subject, project, storedGoal, work, execution, plan, appSpec, actor); err != nil {
			return nil, err
		}
		if fence <= 0 {
			return nil, invalid(errors.New("sqlite.effect_receipt_fence_invalid"))
		}
		v.ActionFence = uint64(fence)
		v.Usage.Resources.Currency = governance.Currency(currency)
		v.Usage.Known = governance.ResourceDimensions(known)
		v.ConfirmedAt = time.Unix(0, at).UTC()
		if governance.ValidateResourceUsage(v.Usage) != nil || !validText(v.Ref) || !validText(v.ExternalRef) ||
			!validSQLiteEffectStatus(v.Status) {
			return nil, invalid(errors.New("sqlite.effect_receipt_invalid"))
		}
		result = append(result, v)
	}
	return result, mapDatabaseError(rows.Err())
}

func readSingleColumn(ctx context.Context, source queryer, query string, args ...any) ([]string, error) {
	rows, err := source.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, mapDatabaseError(err)
		}
		result = append(result, value)
	}
	return result, mapDatabaseError(rows.Err())
}

func readLedger[T any](refs []string, load func(string) (T, error)) ([]T, error) {
	result := make([]T, 0, len(refs))
	for _, ref := range refs {
		value, err := load(ref)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}
