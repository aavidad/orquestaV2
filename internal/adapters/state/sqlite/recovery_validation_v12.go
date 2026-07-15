package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type recoveryDirectorLease struct {
	row     directorLeaseRow
	project goal.ProjectRef
}

type recoveryDirectorReceipt struct {
	row        directorLeaseReceiptRow
	requestRef string
}

func validateRecoveryV12Director(ctx context.Context, transaction *sql.Tx) error {
	leases, err := validateRecoveryV12DirectorLeases(ctx, transaction)
	if err != nil {
		return err
	}
	if err := validateRecoveryV12DirectorReceipts(ctx, transaction, leases); err != nil {
		return err
	}
	return validateRecoveryV12DirectorDecisions(ctx, transaction, leases)
}

func validateRecoveryV12DirectorLeases(
	ctx context.Context,
	transaction *sql.Tx,
) (map[goal.GoalRef]recoveryDirectorLease, error) {
	rows, err := transaction.QueryContext(ctx, `
SELECT goal_ref, project_ref, principal_ref, token, fence, lease_until,
       claim_request_ref, claim_request_fingerprint, claim_authorization_receipt_ref,
       renew_request_ref, renew_request_fingerprint, renew_authorization_receipt_ref,
       updated_at
FROM director_leases ORDER BY goal_ref`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[goal.GoalRef]recoveryDirectorLease)
	for rows.Next() {
		var goalValue string
		var stored directorLeaseRow
		if err := rows.Scan(
			&goalValue, &stored.projectRef, &stored.principalRef, &stored.token,
			&stored.fence, &stored.leaseUntil, &stored.claimRequestRef,
			&stored.claimRequestFingerprint, &stored.claimAuthorizationRef,
			&stored.renewRequestRef, &stored.renewRequestFingerprint,
			&stored.renewAuthorizationRef, &stored.updatedAt,
		); err != nil {
			return nil, err
		}
		goalRef, err := goal.NewGoalRef(goalValue)
		if err != nil {
			return nil, err
		}
		projectRef, err := goal.NewProjectRef(stored.projectRef)
		if err != nil {
			return nil, err
		}
		principalRef, err := identity.NewPrincipalRef(stored.principalRef)
		if err != nil {
			return nil, err
		}
		if !validText(stored.token) || stored.fence <= 0 || stored.leaseUntil <= stored.updatedAt ||
			!validText(stored.claimRequestRef) || !validText(stored.claimRequestFingerprint) ||
			!validText(stored.claimAuthorizationRef) ||
			(stored.renewRequestRef.Valid != stored.renewRequestFingerprint.Valid) ||
			(stored.renewRequestRef.Valid != stored.renewAuthorizationRef.Valid) {
			return nil, errors.New("sqlite.recovery_director_lease_invalid")
		}
		if err := requireDirectorGoalScope(ctx, transaction, goalRef, projectRef); err != nil {
			return nil, err
		}
		claimAuthorization, err := readAuthorizationReceipt(ctx, transaction, stored.claimAuthorizationRef)
		if err != nil {
			return nil, err
		}
		if err := validateRecoveryDirectorAuthorization(
			claimAuthorization, principalRef, projectRef, goalRef,
		); err != nil {
			return nil, err
		}
		if stored.renewAuthorizationRef.Valid {
			renewAuthorization, err := readAuthorizationReceipt(
				ctx, transaction, stored.renewAuthorizationRef.String,
			)
			if err != nil {
				return nil, err
			}
			if err := validateRecoveryDirectorAuthorization(
				renewAuthorization, principalRef, projectRef, goalRef,
			); err != nil {
				return nil, err
			}
		}
		result[goalRef] = recoveryDirectorLease{row: stored, project: projectRef}
	}
	return result, rows.Err()
}

func validateRecoveryV12DirectorReceipts(
	ctx context.Context,
	transaction *sql.Tx,
	leases map[goal.GoalRef]recoveryDirectorLease,
) error {
	rows, err := transaction.QueryContext(ctx, `
SELECT ref, action, request_ref, request_fingerprint, authorization_receipt_ref,
       goal_ref, project_ref, principal_ref, fence, lease_until, occurred_at
FROM director_lease_receipts
ORDER BY goal_ref, fence, occurred_at, ref`)
	if err != nil {
		return err
	}
	var receipts []recoveryDirectorReceipt
	for rows.Next() {
		var receipt recoveryDirectorReceipt
		if err := rows.Scan(
			&receipt.row.ref, &receipt.row.action, &receipt.requestRef,
			&receipt.row.requestFingerprint, &receipt.row.authorizationRef,
			&receipt.row.goalRef, &receipt.row.projectRef, &receipt.row.principalRef,
			&receipt.row.fence, &receipt.row.leaseUntil, &receipt.row.occurredAt,
		); err != nil {
			_ = rows.Close()
			return err
		}
		receipts = append(receipts, receipt)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	claims := make(map[goal.GoalRef]map[int64]recoveryDirectorReceipt)
	lastUntil := make(map[string]int64)
	latest := make(map[string]recoveryDirectorReceipt)
	for _, receipt := range receipts {
		stored := receipt.row
		goalRef, err := goal.NewGoalRef(stored.goalRef)
		if err != nil {
			return err
		}
		projectRef, err := goal.NewProjectRef(stored.projectRef)
		if err != nil {
			return err
		}
		principalRef, err := identity.NewPrincipalRef(stored.principalRef)
		if err != nil {
			return err
		}
		active, found := leases[goalRef]
		if !found || active.project != projectRef || stored.fence <= 0 ||
			stored.fence > active.row.fence || stored.leaseUntil <= stored.occurredAt ||
			!validText(receipt.requestRef) || !validText(stored.requestFingerprint) ||
			!validText(stored.authorizationRef) {
			return errors.New("sqlite.recovery_director_receipt_invalid")
		}
		wantRef := deterministicRef("director-lease-receipt", canonicalFingerprint(
			"director-lease-receipt.v1", stored.action, principalRef.String(), receipt.requestRef,
			stored.requestFingerprint, goalRef.String(), projectRef.String(),
		))
		if stored.ref != wantRef {
			return errors.New("sqlite.recovery_director_receipt_ref_invalid")
		}
		authorization, err := readAuthorizationReceipt(ctx, transaction, stored.authorizationRef)
		if err != nil {
			return err
		}
		if err := validateRecoveryDirectorAuthorization(
			authorization, principalRef, projectRef, goalRef,
		); err != nil {
			return err
		}
		if stored.occurredAt < requiredTime(authorization.RecordedAt()) {
			return errors.New("sqlite.recovery_director_receipt_authorization_time_invalid")
		}
		key := fmt.Sprintf("%s\x00%d", goalRef.String(), stored.fence)
		switch stored.action {
		case "claim":
			if stored.fence > 1 {
				previousKey := fmt.Sprintf("%s\x00%d", goalRef.String(), stored.fence-1)
				previousUntil, exists := lastUntil[previousKey]
				if !exists || stored.occurredAt < previousUntil {
					return errors.New("sqlite.recovery_director_takeover_before_expiry")
				}
			}
			if _, exists := claims[goalRef]; !exists {
				claims[goalRef] = make(map[int64]recoveryDirectorReceipt)
			}
			if _, duplicate := claims[goalRef][stored.fence]; duplicate {
				return errors.New("sqlite.recovery_director_claim_duplicate")
			}
			claims[goalRef][stored.fence] = receipt
			lastUntil[key] = stored.leaseUntil
		case "renew":
			claim, exists := claims[goalRef][stored.fence]
			if !exists || claim.row.principalRef != stored.principalRef ||
				stored.leaseUntil <= lastUntil[key] {
				return errors.New("sqlite.recovery_director_renew_chain_invalid")
			}
			lastUntil[key] = stored.leaseUntil
		default:
			return errors.New("sqlite.recovery_director_receipt_action_invalid")
		}
		latest[key] = receipt
	}
	for goalRef, active := range leases {
		goalClaims := claims[goalRef]
		if int64(len(goalClaims)) != active.row.fence {
			return errors.New("sqlite.recovery_director_claim_chain_incomplete")
		}
		for fence := int64(1); fence <= active.row.fence; fence++ {
			if _, exists := goalClaims[fence]; !exists {
				return errors.New("sqlite.recovery_director_claim_fence_missing")
			}
		}
		claim := goalClaims[active.row.fence]
		if claim.requestRef != active.row.claimRequestRef ||
			claim.row.requestFingerprint != active.row.claimRequestFingerprint ||
			claim.row.authorizationRef != active.row.claimAuthorizationRef ||
			claim.row.principalRef != active.row.principalRef ||
			claim.row.leaseUntil > active.row.leaseUntil {
			return errors.New("sqlite.recovery_director_active_claim_binding_invalid")
		}
		latestReceipt, exists := latest[fmt.Sprintf("%s\x00%d", goalRef.String(), active.row.fence)]
		if !exists || latestReceipt.row.leaseUntil != active.row.leaseUntil ||
			latestReceipt.row.occurredAt != active.row.updatedAt {
			return errors.New("sqlite.recovery_director_active_latest_receipt_invalid")
		}
		switch latestReceipt.row.action {
		case "claim":
			if active.row.renewRequestRef.Valid || latestReceipt.requestRef != active.row.claimRequestRef {
				return errors.New("sqlite.recovery_director_active_claim_latest_invalid")
			}
		case "renew":
			if !active.row.renewRequestRef.Valid ||
				latestReceipt.requestRef != active.row.renewRequestRef.String ||
				latestReceipt.row.requestFingerprint != active.row.renewRequestFingerprint.String ||
				latestReceipt.row.authorizationRef != active.row.renewAuthorizationRef.String {
				return errors.New("sqlite.recovery_director_active_renew_binding_invalid")
			}
		default:
			return errors.New("sqlite.recovery_director_active_latest_receipt_invalid")
		}
	}
	return nil
}

func validateRecoveryV12DirectorDecisions(
	ctx context.Context,
	transaction *sql.Tx,
	leases map[goal.GoalRef]recoveryDirectorLease,
) error {
	type goalState struct {
		revision   int64
		generation int64
	}
	currentGoals := make(map[goal.GoalRef]goalState)
	goalRows, err := transaction.QueryContext(ctx, `
SELECT ref, revision, plan_generation FROM goals ORDER BY ref`)
	if err != nil {
		return err
	}
	for goalRows.Next() {
		var goalValue string
		var revision, generation int64
		if err := goalRows.Scan(&goalValue, &revision, &generation); err != nil {
			_ = goalRows.Close()
			return err
		}
		goalRef, err := goal.NewGoalRef(goalValue)
		if err != nil {
			_ = goalRows.Close()
			return err
		}
		if revision <= 0 || generation <= 0 {
			_ = goalRows.Close()
			return errors.New("sqlite.recovery_director_goal_state_invalid")
		}
		currentGoals[goalRef] = goalState{revision: revision, generation: generation}
	}
	if err := goalRows.Err(); err != nil {
		_ = goalRows.Close()
		return err
	}
	if err := goalRows.Close(); err != nil {
		return err
	}

	rows, err := transaction.QueryContext(ctx, `
SELECT ref, request_ref, request_fingerprint, authorization_receipt_ref,
       goal_ref, project_ref, principal_ref, lease_fence,
       source_goal_revision, source_plan_generation,
       applied_goal_revision, applied_plan_generation, reason, decided_at
FROM director_decisions
ORDER BY goal_ref, applied_plan_generation, ref`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type decisionState struct {
		appliedRevision   int64
		appliedGeneration int64
	}
	latest := make(map[goal.GoalRef]decisionState)
	for rows.Next() {
		var ref, requestRef, fingerprint, authorizationRef string
		var goalValue, projectValue, principalValue, reason string
		var fence, sourceRevision, sourceGeneration, appliedRevision, appliedGeneration, decidedAt int64
		if err := rows.Scan(
			&ref, &requestRef, &fingerprint, &authorizationRef,
			&goalValue, &projectValue, &principalValue, &fence,
			&sourceRevision, &sourceGeneration, &appliedRevision, &appliedGeneration,
			&reason, &decidedAt,
		); err != nil {
			return err
		}
		goalRef, err := goal.NewGoalRef(goalValue)
		if err != nil {
			return err
		}
		projectRef, err := goal.NewProjectRef(projectValue)
		if err != nil {
			return err
		}
		principalRef, err := identity.NewPrincipalRef(principalValue)
		if err != nil {
			return err
		}
		active, found := leases[goalRef]
		current, currentFound := currentGoals[goalRef]
		if !found || active.project != projectRef || fence <= 0 || fence > active.row.fence ||
			sourceRevision <= 0 || sourceGeneration < 0 || appliedRevision != sourceRevision+1 ||
			appliedGeneration != sourceGeneration+1 || !validText(ref) || !validText(requestRef) ||
			!validText(fingerprint) || !validText(reason) || decidedAt == 0 || !currentFound ||
			appliedRevision > current.revision || appliedGeneration > current.generation {
			return errors.New("sqlite.recovery_director_decision_invalid")
		}
		if previous, exists := latest[goalRef]; exists &&
			(sourceGeneration != previous.appliedGeneration ||
				sourceRevision < previous.appliedRevision) {
			return errors.New("sqlite.recovery_director_decision_chain_invalid")
		}
		authorization, err := readAuthorizationReceipt(ctx, transaction, authorizationRef)
		if err != nil {
			return err
		}
		if err := validateRecoveryDirectorAuthorization(
			authorization, principalRef, projectRef, goalRef,
		); err != nil {
			return err
		}
		if decidedAt < requiredTime(authorization.RecordedAt()) {
			return errors.New("sqlite.recovery_director_decision_authorization_time_invalid")
		}
		var causalLease int
		if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM director_lease_receipts
WHERE goal_ref = ? AND project_ref = ? AND principal_ref = ? AND fence = ?
  AND occurred_at <= ? AND lease_until > ?`,
			goalValue, projectValue, principalValue, fence, decidedAt, decidedAt,
		).Scan(&causalLease); err != nil {
			return err
		}
		if causalLease == 0 {
			return errors.New("sqlite.recovery_director_decision_lease_invalid")
		}
		var causalEvent int
		if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM events
WHERE ref = ? AND kind = 'director.plan_applied' AND goal_ref = ?
  AND work_item_ref IS NULL AND execution_ref IS NULL AND occurred_at = ?`,
			"event:director-plan-applied:"+ref, goalValue, decidedAt,
		).Scan(&causalEvent); err != nil {
			return err
		}
		if causalEvent != 1 {
			return errors.New("sqlite.recovery_director_decision_event_invalid")
		}
		latest[goalRef] = decisionState{
			appliedRevision: appliedRevision, appliedGeneration: appliedGeneration,
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for goalRef, decision := range latest {
		current := currentGoals[goalRef]
		if decision.appliedGeneration != current.generation {
			return errors.New("sqlite.recovery_director_decision_tail_invalid")
		}
	}
	return nil
}

func validateRecoveryDirectorAuthorization(
	receipt identity.AuthorizationReceipt,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	goalRef goal.GoalRef,
) error {
	decision := receipt.Decision()
	request := decision.Request()
	if request.Principal().Ref != principalRef || request.ProjectRef() != projectRef ||
		request.Permission() != identity.PermissionGoalsDirect || request.ResourceRef() != goalRef.String() ||
		decision.Outcome() != identity.AuthorizationAllowed ||
		!identity.RoleAllows(decision.Role(), identity.PermissionGoalsDirect) {
		return errors.New("sqlite.recovery_director_authorization_invalid")
	}
	return nil
}
