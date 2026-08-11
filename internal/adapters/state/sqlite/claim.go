package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

type claimCandidate struct {
	action            application.ActionRecord
	projectRef        goal.ProjectRef
	governanceVersion int64
	roleKey           string
	providerRef       string
	modelRef          string
	agentRef          string
	executionState    application.ExecutionState
	executionPurpose  application.ExecutionPurpose
	executionAttempt  int64
	replacesExecution string
	deliveryAttempt   int64
	order             claimCandidateOrder
}

type claimCandidateOrder struct {
	kind, project, goal, availableAt int64
	ref                              string
}

type claimRequirementKey struct {
	goalRef, workItemRef, executionRef string
}

const claimCandidateWindowSize = 16

type claimQueryObservation struct {
	kind      string
	itemCount int
}

type claimQueryObserverContextKey struct{}

func observeClaimQuery(ctx context.Context, observation claimQueryObservation) {
	if observer, ok := ctx.Value(claimQueryObserverContextKey{}).(func(claimQueryObservation)); ok {
		observer(observation)
	}
}

const (
	legacyV4ModelUnattributed = "legacy:v4:model-unattributed"
	legacyV4AgentUnattributed = "legacy:v4:agent-unattributed"
)

func (repository *Repository) ClaimNextAction(
	ctx context.Context,
	request application.ClaimRequest,
) (application.ActionClaim, bool, error) {
	if err := validateClaimRequest(request); err != nil {
		return application.ActionClaim{}, false, err
	}
	transaction, err := beginTransaction(ctx, repository)
	if err != nil {
		return application.ActionClaim{}, false, err
	}
	defer func() { _ = transaction.Rollback() }()

	// The lease starts only after BEGIN IMMEDIATE has acquired the writer. A
	// caller cannot age, extend, or steal a lease by supplying its own clock.
	now, err := repository.transactionTime()
	if err != nil {
		return application.ActionClaim{}, false, err
	}
	if err := requireFreshClaimToken(ctx, transaction, request.Token); err != nil {
		return application.ActionClaim{}, false, err
	}
	workspaceColumns, err := sqliteTableHasColumn(ctx, transaction, "outbox", "change_ref")
	if err != nil {
		return application.ActionClaim{}, false, mapDatabaseError(err)
	}
	var selected claimSelection
	var found bool
	var after *claimCandidateOrder
	for {
		candidates, err := readClaimCandidateWindow(
			ctx, transaction, now, workspaceColumns, request.ExcludeLaunch, after,
		)
		if err != nil {
			return application.ActionClaim{}, false, err
		}
		if len(candidates) == 0 {
			break
		}
		requirements, err := readAgentRequirementsBatch(ctx, transaction, candidates)
		if err != nil {
			return application.ActionClaim{}, false, err
		}
		selected, found, err = selectClaimCandidate(ctx, transaction, candidates, requirements,
			request.Capabilities, request.CapacityCandidates, request.ExcludeLaunch, now)
		if err != nil {
			return application.ActionClaim{}, false, err
		}
		if found || len(candidates) < claimCandidateWindowSize {
			break
		}
		continuation := candidates[len(candidates)-1].order
		after = &continuation
	}
	if !found {
		if err := commit(transaction); err != nil {
			return application.ActionClaim{}, false, err
		}
		return application.ActionClaim{}, false, nil
	}
	leaseDuration := request.LeaseDuration
	if selected.candidate.action.Kind == application.ActionAttestTest && request.AttestTestLeaseDuration > 0 {
		leaseDuration = request.AttestTestLeaseDuration
	}
	leaseUntil, err := safeLeaseUntil(now, leaseDuration)
	if err != nil {
		return application.ActionClaim{}, false, invalid(err)
	}
	claim, err := claimSelectedCandidate(ctx, transaction, request, selected, now, leaseUntil)
	if err != nil {
		return application.ActionClaim{}, false, err
	}
	if err := commit(transaction); err != nil {
		return application.ActionClaim{}, false, err
	}
	return claim, true, nil
}
func validateClaimRequest(request application.ClaimRequest) error {
	if !validText(request.WorkerRef) || !validText(request.Token) {
		return invalid(errors.New("sqlite.claim_identity_invalid"))
	}
	if err := ports.ValidateAgentCapabilities(request.Capabilities); err != nil {
		return invalid(err)
	}
	if request.LeaseDuration <= 0 {
		return invalid(errors.New("sqlite.claim_time_invalid"))
	}
	if request.AttestTestLeaseDuration < 0 {
		return invalid(errors.New("sqlite.claim_time_invalid"))
	}
	if err := application.ValidateBudgetPolicy(request.BudgetPolicy); err != nil {
		return invalid(err)
	}
	ordenados, err := application.OrdenarCandidatosColocacion(request.CapacityCandidates)
	if err != nil || !slices.Equal(ordenados, request.CapacityCandidates) {
		return invalid(errors.New("sqlite.claim_capacity_candidates_invalid"))
	}
	return nil
}

func requireFreshClaimToken(ctx context.Context, tx *sql.Tx, token string) error {
	var exists int
	err := tx.QueryRowContext(ctx, `SELECT 1 FROM (
SELECT claim_token FROM outbox WHERE claim_token=? UNION ALL
SELECT claim_token FROM action_consumption_receipts WHERE claim_token=?) LIMIT 1`, token, token).Scan(&exists)
	if err == nil {
		return stateError(application.StateAlreadyClaimed, errors.New("sqlite.claim_token_reused"))
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return mapDatabaseError(err)
}

type claimSelection struct {
	candidate                claimCandidate
	approval                 application.EffectApproval
	reservation              governance.BudgetReservation
	reservationExists        bool
	disposition              application.ActionClaimDisposition
	budgetExhaustion         application.RetryBudgetExhaustion
	recoveryEffectAttemptRef string
	capacidad                seleccionCapacidadReclamo
	recovery                 *claimRecoverySeed
}

type claimRecoverySeed struct {
	record     application.GoalRecord
	candidates []claimRecoveryCandidate
}

type claimRecoveryCandidate struct {
	attemptRef string
	approval   application.EffectApproval
}

type claimRecoveryState uint8

const (
	claimRecoveryAbsent claimRecoveryState = iota
	claimRecoveryBlocking
	claimRecoveryResolved
	claimRecoveryParked
)

func selectClaimCandidate(
	ctx context.Context, tx *sql.Tx, candidates []claimCandidate,
	requirements map[claimRequirementKey]ports.AgentRequirements,
	capabilities ports.AgentCapabilities, capacityCandidates []application.AgentCapacityPlacementCandidate,
	excludeLaunch bool, now time.Time,
) (claimSelection, bool, error) {
	for index := range candidates {
		candidate := &candidates[index]
		recovery, recoveryState, err := prepareLaunchRecoveryClaim(ctx, tx, candidate, now)
		if err != nil {
			return claimSelection{}, false, err
		}
		if recoveryState == claimRecoveryParked ||
			recoveryState == claimRecoveryResolved && excludeLaunch {
			continue
		}
		if recoveryState == claimRecoveryBlocking {
			matches, err := claimCandidateMatches(*candidate, requirements, capabilities)
			if err != nil {
				return claimSelection{}, false, err
			}
			if !matches {
				continue
			}
			return recovery, true, nil
		}
		exhaustion, local, err := admitIrreversibleRetryBeforeExternal(ctx, tx, candidate)
		if err != nil {
			return claimSelection{}, false, err
		}
		if local {
			return claimSelection{
				candidate: *candidate, disposition: application.ActionClaimDispositionRetryBudgetIrreversible,
				budgetExhaustion: exhaustion,
			}, true, nil
		}
		matches, err := claimCandidateMatches(*candidate, requirements, capabilities)
		if err != nil {
			return claimSelection{}, false, err
		}
		if !matches {
			continue
		}
		approval, admitted, err := admitClaimEffect(ctx, tx, candidate, now)
		if err != nil {
			return claimSelection{}, false, err
		}
		if !admitted {
			continue
		}
		reservation, exists, capacity, err := admitClaimBudget(ctx, tx, *candidate, now)
		if err != nil {
			return claimSelection{}, false, err
		}
		if !capacity {
			continue
		}
		fisica, disponible, err := seleccionarCapacidadReclamo(ctx, tx, *candidate, capacityCandidates, now)
		if err != nil {
			return claimSelection{}, false, err
		}
		if !disponible {
			continue
		}
		return claimSelection{
			candidate: *candidate, approval: approval, reservation: reservation, reservationExists: exists,
			disposition: application.ActionClaimDispositionNormal, capacidad: fisica,
		}, true, nil
	}
	return claimSelection{}, false, nil
}

// prepareLaunchRecoveryClaim detects a physical footprint before any normal
// admission. Once a launch has crossed that boundary it may only continue by
// recovering the exact historical authority; it must never fall back to a new
// approval, budget reservation or capacity observation.
func prepareLaunchRecoveryClaim(
	ctx context.Context,
	tx *sql.Tx,
	candidate *claimCandidate,
	now time.Time,
) (claimSelection, claimRecoveryState, error) {
	if candidate.governanceVersion != 1 || candidate.action.Kind != application.ActionLaunchAgent ||
		candidate.action.EffectIntentRef == "" {
		return claimSelection{}, claimRecoveryAbsent, nil
	}
	var footprints int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM effect_attempts
WHERE action_ref=? AND intent_ref=?`, candidate.action.Ref, candidate.action.EffectIntentRef).Scan(&footprints); err != nil {
		return claimSelection{}, claimRecoveryAbsent, mapDatabaseError(err)
	}
	if footprints == 0 {
		return claimSelection{}, claimRecoveryAbsent, nil
	}
	intent, found, err := requireCandidateEffectIntent(ctx, tx, *candidate)
	if err != nil {
		return claimSelection{}, claimRecoveryBlocking, err
	}
	if !found {
		return claimSelection{}, claimRecoveryBlocking, conflict(errors.New("sqlite.claim_recovery_effect_intent_missing"))
	}
	candidate.action.EffectIntent = intent
	record, err := readGoalRecord(ctx, tx, candidate.action.GoalRef.String())
	if err != nil {
		return claimSelection{}, claimRecoveryBlocking, err
	}
	attempt, preflightErr := application.PreflightAgentLaunchRecoveryAttempt(record, candidate.action)
	if preflightErr != nil {
		if !application.AgentLaunchHasBlockingEffectAttempt(record, candidate.action) {
			return claimSelection{}, claimRecoveryResolved, nil
		}
		return claimSelection{}, claimRecoveryBlocking,
			conflict(errors.New("sqlite.claim_recovery_effect_selection_conflict"))
	}
	approval, unique := exactRecoveryApproval(record.EffectApprovals, attempt.ApprovalRef)
	if !unique {
		return claimSelection{}, claimRecoveryBlocking,
			conflict(errors.New("sqlite.claim_recovery_effect_approval_conflict"))
	}
	current, err := authorizationMembershipCurrent(ctx, tx, intent.Authority)
	if err != nil {
		return claimSelection{}, claimRecoveryBlocking, err
	}
	if current {
		current, err = authorizationMembershipCurrent(ctx, tx, approval.AuthorizationReceipt)
		if err != nil {
			return claimSelection{}, claimRecoveryBlocking, err
		}
	}
	if !current {
		if err := parkLaunchWithStaleAdmission(ctx, tx, *candidate, intent, now); err != nil {
			return claimSelection{}, claimRecoveryBlocking, err
		}
		return claimSelection{}, claimRecoveryParked, nil
	}
	reservation, found, err := readActiveBudgetReservation(ctx, tx, candidate.action.Ref)
	if err != nil {
		return claimSelection{}, claimRecoveryBlocking, err
	}
	if !found || !reservationMatchesCandidate(reservation, *candidate, intent.PolicyHash) {
		return claimSelection{}, claimRecoveryBlocking, conflict(errors.New("sqlite.claim_recovery_budget_reservation_conflict"))
	}
	capacity, placement, found, err := leerReservaCapacidadAccion(ctx, tx, candidate.action.Ref)
	if err != nil {
		return claimSelection{}, claimRecoveryBlocking, err
	}
	if !found {
		return claimSelection{}, claimRecoveryBlocking, conflict(errors.New("sqlite.claim_recovery_capacity_reservation_missing"))
	}
	return claimSelection{
		candidate:         *candidate,
		reservation:       reservation,
		reservationExists: true,
		capacidad: seleccionCapacidadReclamo{
			activa: true, repetida: true, reserva: capacity, colocacion: placement,
		},
		recovery: &claimRecoverySeed{record: record, candidates: []claimRecoveryCandidate{{
			attemptRef: attempt.Ref, approval: approval,
		}}},
	}, claimRecoveryBlocking, nil
}

func exactRecoveryApproval(
	approvals []application.EffectApproval,
	ref string,
) (application.EffectApproval, bool) {
	var selected application.EffectApproval
	matches := 0
	for _, approval := range approvals {
		if approval.Ref == ref {
			selected = approval
			matches++
		}
	}
	return selected, matches == 1
}

func claimCandidateMatches(
	candidate claimCandidate, requirementsByCandidate map[claimRequirementKey]ports.AgentRequirements,
	capabilities ports.AgentCapabilities,
) (bool, error) {
	if candidate.action.Kind == application.ActionPrepareWorkspace || candidate.action.Kind == application.ActionCommitChange ||
		candidate.action.Kind == application.ActionAttestTest || candidate.action.Kind == application.ActionIntegrateChange ||
		candidate.action.Kind == application.ActionAdmitMailbox || candidate.action.Kind == application.ActionRevokeSession {
		return true, nil
	}
	// Terminal stops settle locally; no provider identity is needed.
	if terminalStopSettlement(candidate) {
		return true, nil
	}
	requirements, ok := requirementsByCandidate[requirementKey(candidate)]
	if !ok {
		return false, invalid(errors.New("sqlite.claim_requirements_missing"))
	}
	if !ports.MatchAgentCapabilities(capabilities, requirements) {
		return false, nil
	}
	if candidate.action.Kind == application.ActionObserveAgent || candidate.action.Kind == application.ActionStopAgent {
		return observeIdentityMatches(candidate, capabilities), nil
	}
	return true, nil
}

func admitClaimEffect(
	ctx context.Context, tx *sql.Tx, candidate *claimCandidate, now time.Time,
) (application.EffectApproval, bool, error) {
	if candidate.governanceVersion != 1 {
		return application.EffectApproval{}, true, nil
	}
	if terminalStopSettlement(*candidate) {
		intent, err := requireTerminalStopIntent(ctx, tx, *candidate)
		candidate.action.EffectIntent = intent
		return application.EffectApproval{}, err == nil, err
	}
	if candidate.action.Kind != application.ActionLaunchAgent && candidate.action.Kind != application.ActionQuiesceAgent &&
		candidate.action.Kind != application.ActionPreserveAgentEnvironment &&
		candidate.action.Kind != application.ActionCloseAgentEnvironment && candidate.action.Kind != application.ActionStopAgent &&
		candidate.action.Kind != application.ActionPrepareWorkspace && candidate.action.Kind != application.ActionCommitChange &&
		candidate.action.Kind != application.ActionAttestTest && candidate.action.Kind != application.ActionIntegrateChange {
		return application.EffectApproval{}, true, nil
	}
	intent, approval, admitted, err := requireEffectAdmission(ctx, tx, *candidate, now)
	if err != nil {
		return application.EffectApproval{}, false, err
	}
	if !admitted {
		if candidate.action.Kind == application.ActionLaunchAgent && intent.Ref != "" {
			err = parkLaunchWithStaleAdmission(ctx, tx, *candidate, intent, now)
		}
		return application.EffectApproval{}, false, err
	}
	candidate.action.EffectIntent = intent
	return approval, true, nil
}

func admitClaimBudget(
	ctx context.Context, tx *sql.Tx, candidate claimCandidate, now time.Time,
) (governance.BudgetReservation, bool, bool, error) {
	if candidate.governanceVersion != 1 || candidate.action.Kind != application.ActionLaunchAgent {
		return governance.BudgetReservation{}, false, true, nil
	}
	reservation, exists, capacity, err := prepareBudgetAdmission(ctx, tx, candidate)
	if err != nil || capacity {
		return reservation, exists, capacity, err
	}
	err = deferQuotaLimitedAction(ctx, tx, candidate.action.Ref, now, candidate.action.EffectIntent.QuotaRetryDelay)
	return governance.BudgetReservation{}, false, false, err
}

func admitIrreversibleRetryBeforeExternal(
	ctx context.Context,
	tx *sql.Tx,
	candidate *claimCandidate,
) (application.RetryBudgetExhaustion, bool, error) {
	if candidate.governanceVersion != 1 || candidate.action.Kind != application.ActionLaunchAgent ||
		candidate.executionAttempt <= 1 || candidate.replacesExecution == "" {
		return application.RetryBudgetExhaustion{}, false, nil
	}
	intent, found, err := requireCandidateEffectIntent(ctx, tx, *candidate)
	if err != nil || !found {
		return application.RetryBudgetExhaustion{}, false, err
	}
	candidate.action.EffectIntent = intent
	_, _, capacity, err := prepareBudgetAdmission(ctx, tx, *candidate)
	if err != nil || capacity {
		return application.RetryBudgetExhaustion{}, false, err
	}
	disposition, evidence, err := classifyQueuedRetryBudget(ctx, tx, *candidate)
	if err != nil {
		return application.RetryBudgetExhaustion{}, false, err
	}
	return evidence, disposition == application.RetryBudgetIrreversible, nil
}

func classifyQueuedRetryBudget(
	ctx context.Context,
	tx *sql.Tx,
	candidate claimCandidate,
) (application.RetryBudgetDisposition, application.RetryBudgetExhaustion, error) {
	envelopes, err := requireBudgetEnvelopes(
		ctx, tx, candidate.projectRef, candidate.action.GoalRef,
		candidate.action.EffectIntent.PolicyHash, candidate.action.EffectIntent.PolicyRevision,
	)
	if err != nil {
		return "", application.RetryBudgetExhaustion{}, err
	}
	var goalEnvelope governance.BudgetEnvelope
	for _, envelope := range envelopes {
		if envelope.Scope == governance.BudgetScopeGoal {
			goalEnvelope = envelope
			break
		}
	}
	settlements, err := readBudgetSettlementsForGoal(ctx, tx, candidate.action.GoalRef.String())
	if err != nil {
		return "", application.RetryBudgetExhaustion{}, err
	}
	disposition, evidence, err := application.ClassifyGoalRetryBudget(
		goalEnvelope, settlements, candidate.action.EffectIntent.Demand,
	)
	if err != nil {
		return "", application.RetryBudgetExhaustion{}, invalid(err)
	}
	return disposition, evidence, nil
}

func claimSelectedCandidate(
	ctx context.Context, tx *sql.Tx, request application.ClaimRequest,
	selected claimSelection, now, leaseUntil time.Time,
) (application.ActionClaim, error) {
	candidate := selected.candidate
	if candidate.deliveryAttempt < 0 || uint64(candidate.deliveryAttempt) >= maxSQLiteInteger {
		return application.ActionClaim{}, invalid(fmt.Errorf("sqlite.action_attempt_invalid"))
	}
	recoveryColumn, err := sqliteTableHasColumn(ctx, tx, "outbox", "recovery_effect_attempt_ref")
	if err != nil {
		return application.ActionClaim{}, mapDatabaseError(err)
	}
	if !recoveryColumn && (selected.recovery != nil ||
		selected.disposition == application.ActionClaimDispositionRecoverEffect) {
		return application.ActionClaim{}, conflict(errors.New("sqlite.claim_recovery_effect_schema_missing"))
	}
	deliveryAttempt, fence := candidate.deliveryAttempt+1, int64(0)
	err = tx.QueryRowContext(ctx, `INSERT INTO work_item_fences(goal_ref,work_item_ref,fence) VALUES(?,?,1)
ON CONFLICT(goal_ref,work_item_ref) DO UPDATE SET fence=work_item_fences.fence+1 RETURNING fence`,
		candidate.action.GoalRef.String(), candidate.action.WorkItemRef.String()).Scan(&fence)
	if err != nil {
		return application.ActionClaim{}, mapDatabaseError(err)
	}
	if fence <= 0 {
		return application.ActionClaim{}, invalid(errors.New("sqlite.claim_fence_invalid"))
	}
	if selected.recovery != nil {
		selected, err = finalizeLaunchRecoveryClaim(
			selected, request, leaseUntil, uint64(deliveryAttempt), uint64(fence),
		)
		if err != nil {
			return application.ActionClaim{}, err
		}
	}
	reservation := selected.reservation
	if candidate.governanceVersion == 1 && candidate.action.Kind == application.ActionLaunchAgent {
		if selected.disposition == application.ActionClaimDispositionNormal && !selected.reservationExists {
			reservation, err = insertBudgetReservation(ctx, tx, candidate, uint64(fence), now)
		}
		if err == nil && selected.disposition == application.ActionClaimDispositionNormal {
			err = bindClaimedLaunchExecution(ctx, tx, candidate, reservation)
		}
		if err != nil {
			return application.ActionClaim{}, err
		}
	}
	marker := ""
	if selected.disposition == application.ActionClaimDispositionRetryBudgetIrreversible {
		marker, err = application.RetryBudgetExhaustionMarker(selected.budgetExhaustion)
		if err != nil {
			return application.ActionClaim{}, invalid(err)
		}
	}
	recoveryAssignment := ""
	arguments := []any{request.Token, request.WorkerRef, requiredTime(leaseUntil), deliveryAttempt, fence}
	if recoveryColumn {
		recoveryAssignment = ",recovery_effect_attempt_ref=?"
		arguments = append(arguments, nullableString(selected.recoveryEffectAttemptRef))
	}
	arguments = append(arguments, marker, marker, candidate.action.Ref, requiredTime(now), requiredTime(now))
	result, err := tx.ExecContext(ctx, `UPDATE outbox SET claim_token=?,claimed_by=?,claimed_until=?,delivery_attempt=?,fence=?`+recoveryAssignment+`
 ,last_error_code=CASE WHEN ?<>'' THEN ? ELSE last_error_code END
WHERE ref=? AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL
AND available_at<=? AND (claim_token IS NULL OR claimed_until<=?)`, arguments...)
	if err != nil {
		return application.ActionClaim{}, mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return application.ActionClaim{}, err
	}
	reservaCapacidad, colocacion, err := reservarCapacidadReclamo(ctx, tx, selected.capacidad, candidate, uint64(fence), now)
	if err != nil {
		return application.ActionClaim{}, err
	}
	claim := application.ActionClaim{Action: candidate.action, Token: request.Token, WorkerRef: request.WorkerRef,
		DeliveryAttempt: uint64(deliveryAttempt), Fence: uint64(fence),
		Disposition: selected.disposition, RecoveryEffectAttemptRef: selected.recoveryEffectAttemptRef,
		RetryBudgetExhaustion: selected.budgetExhaustion,
		BudgetReservationRef:  reservation.Ref,
		BudgetReservation:     reservation, CapacityReservation: reservaCapacidad, ReferenciaColocacion: colocacion,
		EffectApproval: selected.approval, LeaseUntil: leaseUntil}
	if candidate.governanceVersion == 1 && candidate.action.Kind == application.ActionLaunchAgent {
		if err := advanceFairness(ctx, tx, candidate.projectRef, candidate.action.GoalRef, now); err != nil {
			return application.ActionClaim{}, err
		}
	}
	return claim, nil
}

func finalizeLaunchRecoveryClaim(
	selected claimSelection,
	request application.ClaimRequest,
	leaseUntil time.Time,
	deliveryAttempt, fence uint64,
) (claimSelection, error) {
	if selected.recovery == nil {
		return selected, nil
	}
	winners := make([]claimRecoveryCandidate, 0, 1)
	for _, candidate := range selected.recovery.candidates {
		claim := application.ActionClaim{
			Action: selected.candidate.action, Token: request.Token, WorkerRef: request.WorkerRef,
			DeliveryAttempt: deliveryAttempt, Fence: fence,
			Disposition:              application.ActionClaimDispositionRecoverEffect,
			RecoveryEffectAttemptRef: candidate.attemptRef,
			BudgetReservationRef:     selected.reservation.Ref,
			BudgetReservation:        selected.reservation,
			CapacityReservation:      selected.capacidad.reserva,
			ReferenciaColocacion:     selected.capacidad.colocacion,
			EffectApproval:           candidate.approval,
			LeaseUntil:               leaseUntil,
		}
		if _, err := application.SelectAgentLaunchRecoveryAttempt(selected.recovery.record, claim); err == nil {
			winners = append(winners, candidate)
		}
	}
	if len(winners) != 1 {
		return claimSelection{}, conflict(errors.New("sqlite.claim_recovery_effect_selection_conflict"))
	}
	selected.disposition = application.ActionClaimDispositionRecoverEffect
	selected.recoveryEffectAttemptRef = winners[0].attemptRef
	selected.approval = winners[0].approval
	return selected, nil
}

func observeIdentityMatches(candidate claimCandidate, capabilities ports.AgentCapabilities) bool {
	if candidate.providerRef != capabilities.ProviderRef {
		return false
	}
	if candidate.modelRef == legacyV4ModelUnattributed && candidate.agentRef == legacyV4AgentUnattributed {
		return true
	}
	return candidate.modelRef == capabilities.ModelRef && candidate.agentRef == capabilities.AgentRef
}

func terminalStopSettlement(candidate claimCandidate) bool {
	if candidate.action.Kind != application.ActionStopAgent {
		return false
	}
	switch candidate.executionState {
	case application.ExecutionSucceeded, application.ExecutionFailed,
		application.ExecutionCanceled, application.ExecutionStopped:
		return true
	default:
		return false
	}
}

func requireTerminalStopIntent(
	ctx context.Context,
	transaction *sql.Tx,
	candidate claimCandidate,
) (application.EffectIntent, error) {
	intent, err := readEffectIntent(ctx, transaction, candidate.action.EffectIntentRef)
	if err != nil {
		return application.EffectIntent{}, err
	}
	if intent.ActionRef != candidate.action.Ref || intent.ActionKind != application.ActionStopAgent ||
		intent.Kind != application.EffectKindAgentStop || intent.Subject.ProjectRef != candidate.projectRef ||
		intent.Subject.GoalRef != candidate.action.GoalRef ||
		intent.Subject.WorkItemRef != candidate.action.WorkItemRef ||
		intent.Subject.ExecutionRef != candidate.action.ExecutionRef ||
		intent.Subject.PlanGeneration != candidate.action.PlanGeneration {
		return application.EffectIntent{}, invalid(errors.New("sqlite.terminal_stop_effect_intent_causal_invalid"))
	}
	return intent, nil
}

const claimCandidatesQuery = `
SELECT o.ref, o.kind, o.goal_ref, o.work_item_ref, o.execution_ref,
       o.control_ref, %s, o.effect_intent_ref, o.governance_version,
       o.plan_generation, o.work_item_generation, o.available_at, g.project_ref,
       o.delivery_attempt, wi.role_key, e.state, e.purpose,
	   e.provider_ref, e.model_ref, e.agent_ref,e.attempt_no,COALESCE(e.replaces_execution_ref,''),
       CASE o.kind WHEN 'stop_agent' THEN 0 WHEN 'prepare_workspace' THEN 1 WHEN 'launch_agent' THEN 2 WHEN 'commit_change' THEN 3 WHEN 'attest_test' THEN 4 WHEN 'integrate_change' THEN 5 WHEN 'quiesce_agent' THEN 6 WHEN 'preserve_agent_environment' THEN 7 WHEN 'close_agent_environment' THEN 8 WHEN 'observe_agent' THEN 9 ELSE 10 END,
       CASE WHEN o.kind = 'launch_agent' THEN COALESCE(project_cursor.ordinal, 0) ELSE 0 END,
       CASE WHEN o.kind = 'launch_agent' THEN COALESCE(goal_cursor.ordinal, 0) ELSE 0 END
FROM outbox o
JOIN work_items wi ON wi.goal_ref = o.goal_ref AND wi.ref = o.work_item_ref
JOIN goals g ON g.ref = o.goal_ref
JOIN executions e
  ON e.goal_ref = o.goal_ref AND e.work_item_ref = o.work_item_ref AND e.ref = o.execution_ref
LEFT JOIN fairness_cursors project_cursor
  ON project_cursor.scope = 'project' AND project_cursor.subject_ref = g.project_ref
LEFT JOIN fairness_cursors goal_cursor
  ON goal_cursor.scope = 'goal' AND goal_cursor.subject_ref = g.ref
WHERE o.completed_at IS NULL
  AND o.retired_at IS NULL
  AND o.quarantined_at IS NULL
  AND o.kind IN ('launch_agent', 'observe_agent', 'quiesce_agent', 'preserve_agent_environment', 'close_agent_environment', 'stop_agent', 'prepare_workspace', 'commit_change', 'attest_test', 'integrate_change', 'admit_mailbox', 'revoke_execution_session')
  AND (o.governance_version = 1 OR o.kind = 'observe_agent'
       OR (o.kind = 'stop_agent' AND e.state IN ('succeeded', 'failed', 'canceled', 'stopped'))
       OR (o.governance_version = 0 AND o.last_error_code <> 'governance.legacy_reauthorization_required'))
  AND (o.kind NOT IN ('quiesce_agent','preserve_agent_environment','close_agent_environment') OR (
      o.governance_version=1 AND o.effect_intent_ref IS NOT NULL
      AND EXISTS (
       SELECT 1 FROM effect_intents lifecycle_intent
       WHERE lifecycle_intent.ref=o.effect_intent_ref AND lifecycle_intent.action_ref=o.ref
        AND lifecycle_intent.action_kind=o.kind AND lifecycle_intent.goal_ref=o.goal_ref
        AND lifecycle_intent.work_item_ref=o.work_item_ref
        AND lifecycle_intent.execution_ref=o.execution_ref
        AND lifecycle_intent.plan_generation=o.plan_generation
        AND lifecycle_intent.kind=CASE o.kind
         WHEN 'quiesce_agent' THEN 'agent_quiesce'
         WHEN 'preserve_agent_environment' THEN 'agent_environment_preserve'
         WHEN 'close_agent_environment' THEN 'agent_environment_close' END)
      AND EXISTS (
       SELECT 1 FROM agent_environment_lifecycles lifecycle
       WHERE lifecycle.execution_ref=o.execution_ref AND lifecycle.goal_ref=o.goal_ref
        AND lifecycle.work_item_ref=o.work_item_ref
        AND lifecycle.token_state=CASE o.kind
         WHEN 'quiesce_agent' THEN 'active'
         WHEN 'preserve_agent_environment' THEN 'quiesced'
         WHEN 'close_agent_environment' THEN 'preserved' END
        AND ((o.kind='quiesce_agent' AND lifecycle.revision=1
              AND lifecycle.claim_action_ref IS NULL AND lifecycle.attempt_ref IS NULL
              AND lifecycle.next_action_ref IS NULL AND lifecycle.ready_to_finalize=0)
             OR lifecycle.next_action_ref=o.ref OR lifecycle.claim_action_ref=o.ref))
  ))
  AND o.available_at <= ?
  AND (o.claim_token IS NULL OR o.claimed_until <= ?)
  AND NOT EXISTS (
      SELECT 1 FROM outbox leased
      WHERE leased.goal_ref = o.goal_ref AND leased.work_item_ref = o.work_item_ref
        AND leased.kind IN ('launch_agent', 'observe_agent', 'quiesce_agent', 'preserve_agent_environment', 'close_agent_environment', 'stop_agent', 'prepare_workspace', 'commit_change', 'attest_test', 'integrate_change', 'admit_mailbox')
        AND leased.ref <> o.ref AND leased.completed_at IS NULL
        AND leased.retired_at IS NULL AND leased.quarantined_at IS NULL
        AND leased.claim_token IS NOT NULL AND leased.claimed_until > ?
  )
  AND (
      o.kind <> 'launch_agent' OR e.state = 'dispatching'
      OR (e.state = 'queued' AND g.paused = 0 AND g.cancel_requested = 0
          AND wi.paused = 0 AND wi.cancel_requested = 0)
  )
  AND (o.kind <> 'stop_agent' OR (
      (e.state = 'running' AND e.external_ref <> '')
      OR e.state IN ('succeeded', 'failed', 'canceled', 'stopped')
  ))
  AND (o.kind <> 'prepare_workspace' OR e.state = 'queued')
  AND (o.kind <> 'commit_change' OR e.state = 'awaiting_commit')
  AND (o.kind <> 'attest_test' OR e.state = 'awaiting_attestation')
  AND (o.kind <> 'integrate_change' OR (e.state = 'awaiting_integration' AND EXISTS (
      SELECT 1 FROM attestations attestation
      WHERE attestation.kind='required_tests' AND attestation.verdict='passed'
       AND attestation.goal_ref=o.goal_ref AND attestation.work_item_ref=o.work_item_ref
       AND attestation.execution_ref=o.execution_ref AND attestation.change_set_ref=o.change_ref
       AND attestation.plan_generation=o.plan_generation
       AND attestation.work_item_generation=o.work_item_generation
  )))
  AND (o.kind <> 'admit_mailbox' OR (e.state='succeeded' AND wi.state='succeeded' AND wi.handoff_required=1))
  AND (o.kind <> 'revoke_execution_session' OR e.state IN ('succeeded','failed','canceled','stopped'))
  AND (o.kind <> 'observe_agent' OR NOT EXISTS (
      SELECT 1 FROM outbox stop
      WHERE stop.goal_ref = o.goal_ref AND stop.execution_ref = o.execution_ref
        AND stop.kind = 'stop_agent' AND stop.completed_at IS NULL
        AND stop.governance_version = 1
        AND stop.retired_at IS NULL AND stop.quarantined_at IS NULL
  ))
  AND (? = 0 OR o.kind <> 'launch_agent' OR EXISTS (
      SELECT 1 FROM effect_attempts recovery_attempt
      WHERE recovery_attempt.action_ref=o.ref
        AND recovery_attempt.intent_ref=o.effect_intent_ref
  ))
  AND (? = 0 OR (
      CASE o.kind WHEN 'stop_agent' THEN 0 WHEN 'prepare_workspace' THEN 1 WHEN 'launch_agent' THEN 2 WHEN 'commit_change' THEN 3 WHEN 'attest_test' THEN 4 WHEN 'integrate_change' THEN 5 WHEN 'quiesce_agent' THEN 6 WHEN 'preserve_agent_environment' THEN 7 WHEN 'close_agent_environment' THEN 8 WHEN 'observe_agent' THEN 9 ELSE 10 END,
      CASE WHEN o.kind = 'launch_agent' THEN COALESCE(project_cursor.ordinal, 0) ELSE 0 END,
      CASE WHEN o.kind = 'launch_agent' THEN COALESCE(goal_cursor.ordinal, 0) ELSE 0 END,
      o.available_at,
      o.ref
  ) > (?, ?, ?, ?, ?))
-- Stop is urgent. Governed launches use hierarchical round-robin before
-- their FIFO tie-break; observations run after no launch fits admission.
ORDER BY CASE o.kind WHEN 'stop_agent' THEN 0 WHEN 'prepare_workspace' THEN 1 WHEN 'launch_agent' THEN 2 WHEN 'commit_change' THEN 3 WHEN 'attest_test' THEN 4 WHEN 'integrate_change' THEN 5 WHEN 'quiesce_agent' THEN 6 WHEN 'preserve_agent_environment' THEN 7 WHEN 'close_agent_environment' THEN 8 WHEN 'observe_agent' THEN 9 ELSE 10 END,
	     CASE WHEN o.kind = 'launch_agent' THEN COALESCE(project_cursor.ordinal, 0) ELSE 0 END,
	     CASE WHEN o.kind = 'launch_agent' THEN COALESCE(goal_cursor.ordinal, 0) ELSE 0 END,
	     o.available_at,
	     o.ref
LIMIT ?`

func readClaimCandidateWindow(
	ctx context.Context,
	transaction *sql.Tx,
	now time.Time,
	workspaceColumns bool,
	excludeLaunch bool,
	after *claimCandidateOrder,
) ([]claimCandidate, error) {
	changeProjection := "'' AS change_ref, '' AS expected_target_oid, '' AS review_gate_digest, '', NULL, NULL, NULL, NULL"
	if workspaceColumns {
		councilColumns, err := sqliteTableHasColumn(ctx, transaction, "outbox", "council_subject_digest")
		if err != nil {
			return nil, mapDatabaseError(err)
		}
		changeProjection = "o.change_ref, o.expected_target_oid, o.review_gate_digest, '', NULL, NULL, NULL, NULL"
		if councilColumns {
			changeProjection = `o.change_ref,o.expected_target_oid,o.review_gate_digest,o.council_subject_digest,
o.council_decision_ref,o.council_decision_digest,o.council_skip_ref,o.council_skip_digest`
		}
	}
	continuation, enabled := claimCandidateOrder{}, 0
	if after != nil {
		continuation, enabled = *after, 1
	}
	observeClaimQuery(ctx, claimQueryObservation{kind: "candidate_window", itemCount: claimCandidateWindowSize})
	rows, err := transaction.QueryContext(ctx, fmt.Sprintf(claimCandidatesQuery, changeProjection),
		requiredTime(now), requiredTime(now), requiredTime(now), storedBool(excludeLaunch), enabled,
		continuation.kind, continuation.project, continuation.goal, continuation.availableAt, continuation.ref,
		claimCandidateWindowSize)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	var result []claimCandidate
	for rows.Next() {
		candidate, err := scanClaimCandidate(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	observeClaimQuery(ctx, claimQueryObservation{kind: "candidate_result", itemCount: len(result)})
	return result, nil
}

func scanClaimCandidate(rows *sql.Rows) (claimCandidate, error) {
	var candidate claimCandidate
	var kind, goalValue, itemValue, executionValue, projectValue string
	var controlRef, effectIntentRef sql.NullString
	var changeRef, expectedTarget, reviewGateDigest string
	var councilSubject string
	var councilDecisionRef, councilDecisionDigest, councilSkipRef, councilSkipDigest sql.NullString
	var planGeneration, itemGeneration, availableAt int64
	err := rows.Scan(&candidate.action.Ref, &kind, &goalValue, &itemValue, &executionValue,
		&controlRef, &changeRef, &expectedTarget, &reviewGateDigest, &councilSubject, &councilDecisionRef,
		&councilDecisionDigest, &councilSkipRef, &councilSkipDigest, &effectIntentRef, &candidate.governanceVersion, &planGeneration,
		&itemGeneration, &availableAt, &projectValue, &candidate.deliveryAttempt,
		&candidate.roleKey, &candidate.executionState, &candidate.executionPurpose, &candidate.providerRef,
		&candidate.modelRef, &candidate.agentRef, &candidate.executionAttempt, &candidate.replacesExecution,
		&candidate.order.kind,
		&candidate.order.project, &candidate.order.goal)
	if err != nil {
		return candidate, mapDatabaseError(err)
	}
	if planGeneration <= 0 || itemGeneration <= 0 || candidate.deliveryAttempt < 0 ||
		(candidate.governanceVersion != 0 && candidate.governanceVersion != 1) {
		return candidate, invalid(errors.New("sqlite.action_generation_invalid"))
	}
	candidate.action.Kind = application.ActionKind(kind)
	if controlRef.Valid {
		candidate.action.ControlRef = controlRef.String
	}
	if changeRef != "" {
		candidate.action.ChangeRef, _ = ports.NewChangeSetRef(changeRef)
	}
	candidate.action.ExpectedTargetOID = expectedTarget
	candidate.action.ReviewGateDigest = reviewGateDigest
	candidate.action.CouncilResolution, err = restoreCouncilResolution(
		councilSubject, councilDecisionRef, councilDecisionDigest, councilSkipRef, councilSkipDigest,
	)
	if err != nil {
		return candidate, err
	}
	if effectIntentRef.Valid {
		candidate.action.EffectIntentRef = effectIntentRef.String
	}
	var refErr error
	if candidate.projectRef, refErr = goal.NewProjectRef(projectValue); refErr != nil {
		return candidate, invalid(refErr)
	}
	if candidate.action.GoalRef, refErr = goal.NewGoalRef(goalValue); refErr != nil {
		return candidate, invalid(refErr)
	}
	if candidate.action.WorkItemRef, refErr = goal.NewWorkItemRef(itemValue); refErr != nil {
		return candidate, invalid(refErr)
	}
	if candidate.action.ExecutionRef, refErr = goal.NewExecutionRef(executionValue); refErr != nil {
		return candidate, invalid(refErr)
	}
	candidate.action.PlanGeneration = goal.PlanGeneration(planGeneration)
	candidate.action.WorkItemGeneration = goal.Revision(itemGeneration)
	candidate.action.AvailableAt = time.Unix(0, availableAt).UTC()
	candidate.order.availableAt = availableAt
	candidate.order.ref = candidate.action.Ref
	if err := validateAction(candidate.action); err != nil {
		return candidate, invalid(err)
	}
	return candidate, nil
}

func readAgentRequirementsBatch(
	ctx context.Context,
	transaction *sql.Tx,
	candidates []claimCandidate,
) (map[claimRequirementKey]ports.AgentRequirements, error) {
	requirements := make(map[claimRequirementKey]ports.AgentRequirements, len(candidates))
	keys := make([]claimRequirementKey, 0, len(candidates))
	args := make([]any, 0, len(candidates)*2)
	for _, candidate := range candidates {
		if !candidateNeedsAgentRequirements(candidate) {
			continue
		}
		key := requirementKey(candidate)
		if _, exists := requirements[key]; exists {
			continue
		}
		requirements[key] = ports.AgentRequirements{RoleKey: candidate.roleKey}
		keys = append(keys, key)
		if candidate.executionPurpose == application.ExecutionPurposePrimaryReview ||
			candidate.executionPurpose == application.ExecutionPurposeAdversarialReview {
			requirements[key] = ports.AgentRequirements{RoleKey: "role:reviewer"}
			continue
		}
		args = append(args, key.goalRef, key.workItemRef, key.executionRef)
	}
	if len(args) == 0 {
		return requirements, nil
	}
	values := strings.TrimSuffix(strings.Repeat("(?, ?, ?),", len(args)/3), ",")
	query := `WITH requested(goal_ref, work_item_ref, execution_ref) AS (VALUES ` + values + `)
SELECT refs.goal_ref, refs.work_item_ref, requested.execution_ref, refs.kind, refs.value
FROM requested
JOIN work_item_requirement_refs refs
  ON refs.goal_ref = requested.goal_ref AND refs.work_item_ref = requested.work_item_ref
ORDER BY refs.goal_ref, refs.work_item_ref, refs.kind, refs.position`
	observeClaimQuery(ctx, claimQueryObservation{kind: "requirements_batch", itemCount: len(keys)})
	rows, err := transaction.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, mapDatabaseError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var goalRef, workItemRef, executionRef, kind, value string
		if err := rows.Scan(&goalRef, &workItemRef, &executionRef, &kind, &value); err != nil {
			return nil, mapDatabaseError(err)
		}
		key := claimRequirementKey{goalRef: goalRef, workItemRef: workItemRef, executionRef: executionRef}
		item := requirements[key]
		switch kind {
		case "skill":
			item.SkillRefs = append(item.SkillRefs, value)
		case "tool":
			item.ToolRefs = append(item.ToolRefs, value)
		case "capability":
			item.CapabilityRefs = append(item.CapabilityRefs, value)
		default:
			return nil, invalid(errors.New("sqlite.requirement_kind_invalid"))
		}
		requirements[key] = item
	}
	if err := rows.Err(); err != nil {
		return nil, mapDatabaseError(err)
	}
	return requirements, nil
}

func candidateNeedsAgentRequirements(candidate claimCandidate) bool {
	switch candidate.action.Kind {
	case application.ActionPrepareWorkspace, application.ActionCommitChange,
		application.ActionAttestTest, application.ActionIntegrateChange,
		application.ActionAdmitMailbox, application.ActionRevokeSession:
		return false
	default:
		return !terminalStopSettlement(candidate)
	}
}

func requirementKey(candidate claimCandidate) claimRequirementKey {
	return claimRequirementKey{
		goalRef: candidate.action.GoalRef.String(), workItemRef: candidate.action.WorkItemRef.String(),
		executionRef: candidate.action.ExecutionRef.String(),
	}
}

func requireClaim(ctx context.Context, transaction *sql.Tx, claim application.ActionClaim) error {
	if err := validateClaim(claim); err != nil {
		return invalid(err)
	}
	var kind, goalValue, itemValue, executionValue string
	var controlRef sql.NullString
	var changeRef, expectedTarget, reviewGateDigest string
	var availableAt, planGeneration, itemGeneration int64
	var token, worker, recoveryEffectAttemptRef sql.NullString
	var leaseUntil, completedAt, quarantinedAt sql.NullInt64
	var deliveryAttempt, fence, currentFence int64
	var lastErrorCode string
	workspaceColumns, err := sqliteTableHasColumn(ctx, transaction, "outbox", "change_ref")
	if err != nil {
		return mapDatabaseError(err)
	}
	changeProjection := "'' AS change_ref, '' AS expected_target_oid, '' AS review_gate_digest"
	if workspaceColumns {
		changeProjection = "o.change_ref, o.expected_target_oid, o.review_gate_digest"
	}
	recoveryProjection := "NULL AS recovery_effect_attempt_ref"
	recoveryColumn, err := sqliteTableHasColumn(ctx, transaction, "outbox", "recovery_effect_attempt_ref")
	if err != nil {
		return mapDatabaseError(err)
	}
	if recoveryColumn {
		recoveryProjection = "o.recovery_effect_attempt_ref"
	}
	err = transaction.QueryRowContext(ctx, fmt.Sprintf(`
SELECT o.kind, o.goal_ref, o.work_item_ref, o.execution_ref, o.control_ref, %s,
       o.plan_generation, o.work_item_generation, o.available_at,
       o.claim_token, o.claimed_by, o.claimed_until, o.delivery_attempt, o.fence,
	       o.completed_at, o.quarantined_at, o.last_error_code, wf.fence,
	       %s
FROM outbox o
JOIN work_item_fences wf ON wf.goal_ref = o.goal_ref AND wf.work_item_ref = o.work_item_ref
WHERE o.ref = ?`, changeProjection, recoveryProjection), claim.Action.Ref).Scan(
		&kind, &goalValue, &itemValue, &executionValue, &controlRef, &changeRef, &expectedTarget, &reviewGateDigest,
		&planGeneration, &itemGeneration, &availableAt,
		&token, &worker, &leaseUntil, &deliveryAttempt, &fence,
		&completedAt, &quarantinedAt, &lastErrorCode, &currentFence, &recoveryEffectAttemptRef,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return conflict(err)
	}
	if err != nil {
		return mapDatabaseError(err)
	}
	if completedAt.Valid || quarantinedAt.Valid || !token.Valid || !worker.Valid || !leaseUntil.Valid ||
		token.String != claim.Token || worker.String != claim.WorkerRef ||
		deliveryAttempt != int64(claim.DeliveryAttempt) || fence != int64(claim.Fence) || currentFence != int64(claim.Fence) ||
		leaseUntil.Int64 != requiredTime(claim.LeaseUntil) || kind != string(claim.Action.Kind) ||
		goalValue != claim.Action.GoalRef.String() || itemValue != claim.Action.WorkItemRef.String() ||
		executionValue != claim.Action.ExecutionRef.String() ||
		controlRef.String != claim.Action.ControlRef ||
		changeRef != claim.Action.ChangeRef.String() || expectedTarget != claim.Action.ExpectedTargetOID ||
		reviewGateDigest != claim.Action.ReviewGateDigest ||
		planGeneration != int64(claim.Action.PlanGeneration) ||
		itemGeneration != int64(claim.Action.WorkItemGeneration) ||
		availableAt != requiredTime(claim.Action.AvailableAt) {
		return conflict(errors.New("sqlite.claim_cas_conflict"))
	}
	marker, markerErr := application.RetryBudgetExhaustionMarker(claim.RetryBudgetExhaustion)
	switch claim.Disposition {
	case application.ActionClaimDispositionNormal:
		if recoveryEffectAttemptRef.Valid || strings.HasPrefix(lastErrorCode, application.RetryBudgetExhaustionMarkerPrefix) {
			return conflict(errors.New("sqlite.claim_disposition_conflict"))
		}
	case application.ActionClaimDispositionRetryBudgetIrreversible:
		if recoveryEffectAttemptRef.Valid || markerErr != nil || lastErrorCode != marker {
			return conflict(errors.New("sqlite.claim_disposition_conflict"))
		}
	case application.ActionClaimDispositionRecoverEffect:
		if !recoveryEffectAttemptRef.Valid || recoveryEffectAttemptRef.String != claim.RecoveryEffectAttemptRef ||
			strings.HasPrefix(lastErrorCode, application.RetryBudgetExhaustionMarkerPrefix) {
			return conflict(errors.New("sqlite.claim_disposition_conflict"))
		}
		record, err := readGoalRecord(ctx, transaction, claim.Action.GoalRef.String())
		if err != nil {
			return err
		}
		attempt, err := application.SelectAgentLaunchRecoveryAttempt(record, claim)
		if err != nil || attempt.Ref != recoveryEffectAttemptRef.String {
			return conflict(errors.New("sqlite.claim_recovery_effect_conflict"))
		}
	default:
		return conflict(errors.New("sqlite.claim_disposition_invalid"))
	}
	if claim.Action.Kind == application.ActionLaunchAgent && claim.Action.EffectIntentRef != "" &&
		(claim.Disposition == application.ActionClaimDispositionNormal ||
			claim.Disposition == application.ActionClaimDispositionRecoverEffect) {
		reserva, colocacion, encontrada, err := leerReservaCapacidadAccion(ctx, transaction, claim.Action.Ref)
		if err != nil {
			return err
		}
		if !encontrada || colocacion != claim.ReferenciaColocacion || !coincideReservaCapacidadReclamo(reserva, claim.CapacityReservation) {
			return conflict(errors.New("sqlite.claim_capacity_conflict"))
		}
	}
	return nil
}

func completeClaim(
	ctx context.Context,
	transaction *sql.Tx,
	claim application.ActionClaim,
	at time.Time,
	errorCode string,
	quarantined bool,
) error {
	return completeClaimWithEffect(ctx, transaction, claim, at, errorCode, quarantined, nil)
}

func completeClaimWithEffect(
	ctx context.Context,
	transaction *sql.Tx,
	claim application.ActionClaim,
	at time.Time,
	errorCode string,
	quarantined bool,
	effect *application.EffectReceipt,
) error {
	if claim.Disposition == application.ActionClaimDispositionRetryBudgetIrreversible {
		marker, err := application.RetryBudgetExhaustionMarker(claim.RetryBudgetExhaustion)
		if err != nil {
			return invalid(err)
		}
		errorCode = marker
	}
	var quarantineAt any
	outcome := application.ActionConsumedCompleted
	if quarantined {
		quarantineAt = requiredTime(at)
		outcome = application.ActionConsumedQuarantined
	}
	result, err := transaction.ExecContext(ctx, `
UPDATE outbox
SET completed_at = ?, quarantined_at = ?, last_error_code = ?
WHERE ref = ? AND claim_token = ? AND claimed_by = ? AND claimed_until = ?
  AND delivery_attempt = ? AND fence = ?
  AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL`,
		requiredTime(at), quarantineAt, errorCode,
		claim.Action.Ref, claim.Token, claim.WorkerRef, requiredTime(claim.LeaseUntil),
		int64(claim.DeliveryAttempt), int64(claim.Fence),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	if err := requireOneRow(result); err != nil {
		return err
	}
	if effect != nil {
		if err := insertEffectReceipt(ctx, transaction, claim, *effect, at); err != nil {
			return err
		}
	}
	version := int64(0)
	if claim.Action.EffectIntentRef != "" {
		version = 1
	}
	receipt := application.ActionConsumptionReceipt{
		ActionRef: claim.Action.Ref, Kind: claim.Action.Kind, GoalRef: claim.Action.GoalRef,
		WorkItemRef: claim.Action.WorkItemRef, ExecutionRef: claim.Action.ExecutionRef,
		PlanGeneration: claim.Action.PlanGeneration, WorkItemGeneration: claim.Action.WorkItemGeneration,
		Fence: claim.Fence, DeliveryAttempt: claim.DeliveryAttempt, ClaimToken: claim.Token,
		WorkerRef: claim.WorkerRef, Outcome: outcome, ErrorCode: errorCode, ConsumedAt: at,
		ChangeRef: claim.Action.ChangeRef,
	}
	if effect != nil {
		receipt.EffectReceiptRef = effect.Ref
	}
	if err := insertActionConsumptionReceipt(ctx, transaction, receipt, version, effect); err != nil {
		return err
	}
	return scheduleExecutionSessionRevocation(ctx, transaction, claim.Action.GoalRef,
		claim.Action.WorkItemRef, claim.Action.ExecutionRef, claim.Action.Kind, at)
}

func scheduleExecutionSessionRevocation(
	ctx context.Context, tx *sql.Tx, goalRef goal.GoalRef, itemRef goal.WorkItemRef,
	executionRef goal.ExecutionRef, completedKind application.ActionKind, at time.Time,
) error {
	if completedKind == application.ActionRevokeSession {
		return nil
	}
	supported, err := sqliteTableHasColumn(ctx, tx, "outbox", "admission_request_ref")
	if err != nil || !supported {
		return err
	}
	var state, purpose string
	var handoff int64
	var plan, revision int64
	err = tx.QueryRowContext(ctx, `SELECT e.state,e.purpose,w.handoff_required,e.plan_generation,w.revision
FROM executions e JOIN work_items w ON w.goal_ref=e.goal_ref AND w.ref=e.work_item_ref
WHERE e.goal_ref=? AND e.work_item_ref=? AND e.ref=? AND e.execution_session_ref<>''`,
		goalRef.String(), itemRef.String(), executionRef.String(),
	).Scan(&state, &purpose, &handoff, &plan, &revision)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return mapDatabaseError(err)
	}
	if state != string(application.ExecutionSucceeded) && state != string(application.ExecutionFailed) &&
		state != string(application.ExecutionCanceled) && state != string(application.ExecutionStopped) {
		return nil
	}
	if state == string(application.ExecutionSucceeded) && handoff == 1 &&
		(purpose == string(application.ExecutionPurposeWork) || purpose == string(application.ExecutionPurposeAuthor)) &&
		completedKind != application.ActionAdmitMailbox {
		return nil
	}
	ref := "action:revoke-execution-session:" + executionRef.String()
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox WHERE ref=?`, ref).Scan(&exists); err != nil {
		return mapDatabaseError(err)
	}
	if exists != 0 {
		return nil
	}
	return insertAction(ctx, tx, application.ActionRecord{
		Ref: ref, Kind: application.ActionRevokeSession, GoalRef: goalRef,
		WorkItemRef: itemRef, ExecutionRef: executionRef,
		PlanGeneration: goal.PlanGeneration(plan), WorkItemGeneration: goal.Revision(revision), AvailableAt: at,
	})
}

func insertActionConsumptionReceipt(
	ctx context.Context, tx *sql.Tx, receipt application.ActionConsumptionReceipt,
	governanceVersion int64, effect *application.EffectReceipt,
) error {
	persisted, err := sqliteTableHasColumn(ctx, tx, "action_consumption_receipts", "governance_version")
	if err != nil {
		return mapDatabaseError(err)
	}
	if persisted {
		workspaceColumns, columnErr := sqliteTableHasColumn(ctx, tx, "action_consumption_receipts", "change_ref")
		if columnErr != nil {
			return mapDatabaseError(columnErr)
		}
		if !workspaceColumns {
			_, err = tx.ExecContext(ctx, `
INSERT INTO action_consumption_receipts(
    action_ref, governance_version, kind, goal_ref, work_item_ref, execution_ref,
    plan_generation, work_item_generation, fence, delivery_attempt,
    claim_token, worker_ref, outcome, error_code, consumed_at, effect_receipt_ref
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				receipt.ActionRef, governanceVersion, string(receipt.Kind), receipt.GoalRef.String(),
				receipt.WorkItemRef.String(), receipt.ExecutionRef.String(), int64(receipt.PlanGeneration),
				int64(receipt.WorkItemGeneration), int64(receipt.Fence), int64(receipt.DeliveryAttempt),
				receipt.ClaimToken, receipt.WorkerRef, string(receipt.Outcome), receipt.ErrorCode,
				requiredTime(receipt.ConsumedAt), nullableString(receipt.EffectReceiptRef),
			)
			return mapDatabaseError(err)
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO action_consumption_receipts(
    action_ref, governance_version, kind, goal_ref, work_item_ref, execution_ref,
    change_ref,
    plan_generation, work_item_generation, fence, delivery_attempt,
    claim_token, worker_ref, outcome, error_code, consumed_at, effect_receipt_ref
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			receipt.ActionRef, governanceVersion, string(receipt.Kind), receipt.GoalRef.String(),
			receipt.WorkItemRef.String(), receipt.ExecutionRef.String(), receipt.ChangeRef.String(), int64(receipt.PlanGeneration),
			int64(receipt.WorkItemGeneration), int64(receipt.Fence), int64(receipt.DeliveryAttempt),
			receipt.ClaimToken, receipt.WorkerRef, string(receipt.Outcome), receipt.ErrorCode,
			requiredTime(receipt.ConsumedAt), nullableString(receipt.EffectReceiptRef),
		)
		return mapDatabaseError(err)
	}
	var effectStatus, effectAt any
	if effect != nil {
		effectStatus, effectAt = effect.Status, requiredTime(effect.ConfirmedAt)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO action_consumption_receipts(
    action_ref, kind, goal_ref, work_item_ref, execution_ref,
    plan_generation, work_item_generation, fence, delivery_attempt,
    claim_token, worker_ref, outcome, error_code, consumed_at,
    effect_receipt_ref, effect_status, effect_confirmed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		receipt.ActionRef, string(receipt.Kind), receipt.GoalRef.String(), receipt.WorkItemRef.String(),
		receipt.ExecutionRef.String(), int64(receipt.PlanGeneration), int64(receipt.WorkItemGeneration),
		int64(receipt.Fence), int64(receipt.DeliveryAttempt), receipt.ClaimToken, receipt.WorkerRef,
		string(receipt.Outcome), receipt.ErrorCode, requiredTime(receipt.ConsumedAt),
		nullableString(receipt.EffectReceiptRef), effectStatus, effectAt,
	)
	return mapDatabaseError(err)
}

func releaseClaimForRetry(
	ctx context.Context,
	transaction *sql.Tx,
	state application.ActionRequeuedState,
) error {
	result, err := transaction.ExecContext(ctx, `
UPDATE outbox
SET available_at = ?, claim_token = NULL, claimed_by = NULL, claimed_until = NULL,
    last_error_code = ?
WHERE ref = ? AND claim_token = ? AND claimed_by = ? AND claimed_until = ?
  AND delivery_attempt = ? AND fence = ?
  AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL`,
		requiredTime(state.AvailableAt), state.ErrorCode,
		state.Claim.Action.Ref, state.Claim.Token, state.Claim.WorkerRef,
		requiredTime(state.Claim.LeaseUntil), int64(state.Claim.DeliveryAttempt), int64(state.Claim.Fence),
	)
	if err != nil {
		return mapDatabaseError(err)
	}
	return requireOneRow(result)
}

func (repository *Repository) transactionTime() (time.Time, error) {
	if repository == nil || repository.now == nil {
		return time.Time{}, invalid(errors.New("sqlite.clock_unavailable"))
	}
	now := repository.now().Round(0).UTC()
	if now.IsZero() {
		return time.Time{}, invalid(errors.New("sqlite.clock_invalid"))
	}
	return now, nil
}
