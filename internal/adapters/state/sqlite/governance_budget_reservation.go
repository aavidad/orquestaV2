package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
)

func readBudgetUsage(ctx context.Context, transaction *sql.Tx, scope governance.BudgetScope, subject string) (governance.ResourceVector, error) {
	filter, arguments := "1 = 1", []any{}
	switch scope {
	case governance.BudgetScopeDeployment:
		filter = "settlement.ref IS NULL"
	case governance.BudgetScopeProject:
		filter, arguments = "reservation.project_ref = ? AND settlement.ref IS NULL", []any{subject}
	case governance.BudgetScopeGoal:
		filter, arguments = "reservation.goal_ref = ?", []any{subject}
	default:
		return governance.ResourceVector{}, invalid(errors.New("sqlite.budget_scope_invalid"))
	}
	rows, err := transaction.QueryContext(ctx, `
SELECT COALESCE(settlement.charged_tokens,reservation.tokens),
 COALESCE(settlement.charged_money_micros,reservation.money_micros),
 COALESCE(settlement.charged_currency,reservation.currency),
 COALESCE(settlement.charged_active_time_ns,reservation.active_time_ns),
 CASE WHEN settlement.ref IS NULL THEN reservation.process_slots ELSE 0 END,
 COALESCE(settlement.charged_disk_bytes,reservation.disk_bytes)
FROM budget_reservations reservation LEFT JOIN budget_settlements settlement ON settlement.reservation_ref=reservation.ref
WHERE `+filter+` ORDER BY reservation.reserved_at,reservation.ref`, arguments...)
	if err != nil {
		return governance.ResourceVector{}, mapDatabaseError(err)
	}
	defer rows.Close()
	usage := governance.ResourceVector{}
	for rows.Next() {
		var vector governance.ResourceVector
		var currency string
		if err := rows.Scan(&vector.Tokens, &vector.MoneyMicros, &currency, &vector.ActiveTimeNS, &vector.ProcessSlots, &vector.DiskBytes); err != nil {
			return governance.ResourceVector{}, mapDatabaseError(err)
		}
		vector.Currency = governance.Currency(currency)
		usage, err = governance.Add(usage, vector)
		if err != nil {
			return governance.ResourceVector{}, invalid(err)
		}
	}
	return usage, mapDatabaseError(rows.Err())
}

func readActiveBudgetReservation(ctx context.Context, source queryer, actionRef string) (governance.BudgetReservation, bool, error) {
	refs, err := readSingleColumn(ctx, source, `
SELECT reservation.ref FROM budget_reservations reservation
LEFT JOIN budget_settlements settlement ON settlement.reservation_ref=reservation.ref
WHERE reservation.action_ref=? AND settlement.ref IS NULL ORDER BY reservation.reserved_at,reservation.ref`, actionRef)
	if err != nil || len(refs) == 0 {
		return governance.BudgetReservation{}, false, err
	}
	if len(refs) != 1 {
		return governance.BudgetReservation{}, false, invalid(errors.New("sqlite.multiple_active_budget_reservations"))
	}
	reservation, err := readBudgetReservation(ctx, source, refs[0])
	return reservation, err == nil, err
}

func readBudgetReservation(ctx context.Context, source queryer, ref string) (governance.BudgetReservation, error) {
	var value governance.BudgetReservation
	var currency string
	var plan, appSpec, item, fence, at int64
	err := source.QueryRowContext(ctx, `
SELECT ref,demand_ref,action_ref,effect_intent_ref,project_ref,goal_ref,work_item_ref,execution_ref,
 plan_generation,app_spec_generation,work_item_generation,fence,spec_hash,policy_hash,tokens,money_micros,
 currency,active_time_ns,process_slots,disk_bytes,reserved_at FROM budget_reservations WHERE ref=?`, ref).Scan(
		&value.Ref, &value.DemandRef, &value.ActionRef, &value.EffectIntentRef, &value.ProjectRef, &value.GoalRef,
		&value.WorkItemRef, &value.ExecutionRef, &plan, &appSpec, &item, &fence, &value.SpecHash, &value.PolicyHash,
		&value.Resources.Tokens, &value.Resources.MoneyMicros, &currency, &value.Resources.ActiveTimeNS,
		&value.Resources.ProcessSlots, &value.Resources.DiskBytes, &at)
	if err != nil {
		return governance.BudgetReservation{}, mapDatabaseError(err)
	}
	if plan <= 0 || appSpec <= 0 || item <= 0 || fence <= 0 {
		return governance.BudgetReservation{}, invalid(errors.New("sqlite.budget_reservation_generation_invalid"))
	}
	value.PlanGeneration, value.AppSpecGeneration = uint64(plan), uint64(appSpec)
	value.WorkItemGeneration, value.Fence = uint64(item), uint64(fence)
	value.Resources.Currency, value.ReservedAt = governance.Currency(currency), time.Unix(0, at).UTC()
	if err := governance.ValidateBudgetReservation(value); err != nil {
		return governance.BudgetReservation{}, invalid(err)
	}
	return value, nil
}

func reservationMatchesCandidate(value governance.BudgetReservation, candidate claimCandidate, policyHash string) bool {
	intent := candidate.action.EffectIntent
	return value.DemandRef == intent.Demand.Ref && value.ActionRef == candidate.action.Ref && value.EffectIntentRef == intent.Ref &&
		value.ProjectRef == candidate.projectRef.String() && value.GoalRef == candidate.action.GoalRef.String() &&
		value.WorkItemRef == candidate.action.WorkItemRef.String() && value.ExecutionRef == candidate.action.ExecutionRef.String() &&
		value.PlanGeneration == uint64(candidate.action.PlanGeneration) && value.AppSpecGeneration == uint64(intent.Subject.AppSpecGeneration) &&
		value.WorkItemGeneration == uint64(candidate.action.WorkItemGeneration) && value.SpecHash == intent.Subject.SpecHash &&
		value.PolicyHash == policyHash && value.Resources == intent.Demand.Resources
}

func requireClaimBudgetReservation(ctx context.Context, tx *sql.Tx, claim application.ActionClaim, candidate claimCandidate) error {
	if claim.BudgetReservationRef == "" || claim.BudgetReservation.Ref != claim.BudgetReservationRef {
		return conflict(errors.New("sqlite.claim_budget_reservation_missing"))
	}
	stored, err := readBudgetReservation(ctx, tx, claim.BudgetReservationRef)
	if err != nil {
		return err
	}
	if stored != claim.BudgetReservation || stored.Fence > claim.Fence || !reservationMatchesCandidate(stored, candidate, candidate.action.EffectIntent.PolicyHash) {
		return conflict(errors.New("sqlite.claim_budget_reservation_conflict"))
	}
	var settled int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM budget_settlements WHERE reservation_ref=?`, stored.Ref).Scan(&settled); err != nil {
		return mapDatabaseError(err)
	}
	if settled != 0 {
		return conflict(errors.New("sqlite.claim_budget_reservation_settled"))
	}
	return nil
}

func insertBudgetReservation(ctx context.Context, tx *sql.Tx, candidate claimCandidate, fence uint64, now time.Time) (governance.BudgetReservation, error) {
	intent := candidate.action.EffectIntent
	value := governance.BudgetReservation{
		Ref:       deterministicRef("budget-reservation", canonicalFingerprint(candidate.action.Ref, strconv.FormatUint(fence, 10), intent.Digest)),
		DemandRef: intent.Demand.Ref, ActionRef: candidate.action.Ref, EffectIntentRef: intent.Ref,
		ProjectRef: candidate.projectRef.String(), GoalRef: candidate.action.GoalRef.String(),
		WorkItemRef: candidate.action.WorkItemRef.String(), ExecutionRef: candidate.action.ExecutionRef.String(),
		PlanGeneration: uint64(candidate.action.PlanGeneration), AppSpecGeneration: uint64(intent.Subject.AppSpecGeneration),
		WorkItemGeneration: uint64(candidate.action.WorkItemGeneration), Fence: fence, SpecHash: intent.Subject.SpecHash,
		PolicyHash: intent.PolicyHash, Resources: intent.Demand.Resources, ReservedAt: now,
	}
	if err := governance.ValidateBudgetReservation(value); err != nil {
		return governance.BudgetReservation{}, invalid(err)
	}
	r := value.Resources
	_, err := tx.ExecContext(ctx, `
INSERT INTO budget_reservations(ref,demand_ref,action_ref,effect_intent_ref,project_ref,goal_ref,work_item_ref,execution_ref,
 plan_generation,app_spec_generation,work_item_generation,fence,spec_hash,policy_hash,tokens,money_micros,currency,
 active_time_ns,process_slots,disk_bytes,reserved_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		value.Ref, value.DemandRef, value.ActionRef, value.EffectIntentRef, value.ProjectRef, value.GoalRef,
		value.WorkItemRef, value.ExecutionRef, int64(value.PlanGeneration), int64(value.AppSpecGeneration),
		int64(value.WorkItemGeneration), int64(value.Fence), value.SpecHash, value.PolicyHash, r.Tokens, r.MoneyMicros,
		string(r.Currency), r.ActiveTimeNS, r.ProcessSlots, r.DiskBytes, requiredTime(value.ReservedAt))
	if err != nil {
		return governance.BudgetReservation{}, mapDatabaseError(err)
	}
	return value, nil
}

func bindClaimedLaunchExecution(ctx context.Context, tx *sql.Tx, candidate claimCandidate, reservation governance.BudgetReservation) error {
	result, err := tx.ExecContext(ctx, `
UPDATE executions SET governance_version=1,budget_reservation_ref=?,effect_intent_ref=?,launch_receipt_ref=NULL
WHERE ref=? AND goal_ref=? AND work_item_ref=? AND state IN ('queued','dispatching') AND
 ((governance_version=0 AND budget_reservation_ref IS NULL AND effect_intent_ref IS NULL AND launch_receipt_ref IS NULL)
 OR (governance_version=1 AND budget_reservation_ref=? AND effect_intent_ref=? AND launch_receipt_ref IS NULL))`,
		reservation.Ref, candidate.action.EffectIntent.Ref, candidate.action.ExecutionRef.String(),
		candidate.action.GoalRef.String(), candidate.action.WorkItemRef.String(), reservation.Ref, candidate.action.EffectIntent.Ref)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func clearClaimEffectBinding(ctx context.Context, tx *sql.Tx, claim application.ActionClaim) error {
	result, err := tx.ExecContext(ctx, `
UPDATE executions SET governance_version=0,budget_reservation_ref=NULL,effect_intent_ref=NULL,launch_receipt_ref=NULL
WHERE ref=? AND goal_ref=? AND work_item_ref=? AND governance_version=1
 AND budget_reservation_ref=? AND effect_intent_ref=? AND launch_receipt_ref IS NULL`,
		claim.Action.ExecutionRef.String(), claim.Action.GoalRef.String(), claim.Action.WorkItemRef.String(),
		claim.BudgetReservationRef, claim.Action.EffectIntentRef)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func deferQuotaLimitedAction(ctx context.Context, tx *sql.Tx, actionRef string, now time.Time, delay time.Duration) error {
	availableAt, err := safeLeaseUntil(now, delay)
	if err != nil {
		return invalid(err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE outbox SET available_at=?,last_error_code='budget.temporarily_unavailable'
WHERE ref=? AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL
 AND (claim_token IS NULL OR claimed_until<=?)`, requiredTime(availableAt), actionRef, requiredTime(now))
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

// parkLaunchWithStaleAdmission handles the crash window after reservation but
// before a provider attempt. Exact zero evidence releases that reservation;
// once an attempt exists the binding stays charged and reusable with the same
// external idempotency key until authority is restored or reconciled.
func parkLaunchWithStaleAdmission(
	ctx context.Context,
	tx *sql.Tx,
	candidate claimCandidate,
	intent application.EffectIntent,
	now time.Time,
) error {
	reservation, found, err := readActiveBudgetReservation(ctx, tx, candidate.action.Ref)
	if err != nil {
		return err
	}
	if found {
		var attempts int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM effect_attempts
WHERE action_ref=? AND action_fence>=?`, candidate.action.Ref, int64(reservation.Fence)).Scan(&attempts); err != nil {
			return mapDatabaseError(err)
		}
		if attempts == 0 {
			usage := governance.ResourceUsage{
				Resources: governance.ResourceVector{Currency: reservation.Resources.Currency},
				Known:     governance.AllResourceDimensions, Quality: governance.UsageQualityExact,
			}
			settlement, err := governance.Reconcile(reservation, usage)
			if err != nil {
				return invalid(err)
			}
			settlement.SettledAt = now
			candidate.action.EffectIntentRef, candidate.action.EffectIntent = intent.Ref, intent
			claim := application.ActionClaim{
				Action: candidate.action, BudgetReservationRef: reservation.Ref,
				BudgetReservation: reservation,
			}
			if err := clearClaimEffectBinding(ctx, tx, claim); err != nil {
				return err
			}
			if err := insertBudgetSettlement(ctx, tx, settlement); err != nil {
				return err
			}
		}
	}
	availableAt, err := safeLeaseUntil(now, intent.QuotaRetryDelay)
	if err != nil {
		return invalid(err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE outbox
SET available_at=?,claim_token=NULL,claimed_by=NULL,claimed_until=NULL,
    last_error_code='governance.effect_approval_required'
WHERE ref=? AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL
 AND (claim_token IS NULL OR claimed_until<=?)`, requiredTime(availableAt), candidate.action.Ref, requiredTime(now))
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func advanceFairness(ctx context.Context, tx *sql.Tx, projectRef goal.ProjectRef, goalRef goal.GoalRef, now time.Time) error {
	var ordinal int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(ordinal),0)+1 FROM fairness_cursors`).Scan(&ordinal); err != nil {
		return mapDatabaseError(err)
	}
	if ordinal <= 0 {
		return invalid(fmt.Errorf("sqlite.fairness_ordinal_invalid"))
	}
	for _, cursor := range []struct{ scope, subject string }{{"project", projectRef.String()}, {"goal", goalRef.String()}} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO fairness_cursors(scope,subject_ref,ordinal,updated_at) VALUES(?,?,?,?)
ON CONFLICT(scope,subject_ref) DO UPDATE SET ordinal=excluded.ordinal,updated_at=excluded.updated_at`,
			cursor.scope, cursor.subject, ordinal, requiredTime(now)); err != nil {
			return mapDatabaseError(err)
		}
	}
	return nil
}
