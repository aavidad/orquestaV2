package sqlite

import (
	"context"
	"database/sql"
	"reflect"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func TestAgentLaunchRecoverySurvivesRestartLostAcknowledgementAndClaimRace(t *testing.T) {
	ctx := context.Background()
	system, first, attempt := seedV27AmbiguousEffectAttempt(t, "restart-e2e")
	initialCapacity, initialPlacement, found, err := leerReservaCapacidadAccion(
		ctx, system.repository.db, first.Action.Ref,
	)
	if err != nil || !found || initialCapacity != first.CapacityReservation ||
		initialPlacement != first.ReferenciaColocacion {
		t.Fatalf("initial capacity=%+v placement=%s found=%t err=%s",
			initialCapacity, initialPlacement, found, sqliteTestErrorChain(err))
	}

	system.clock.Advance(first.LeaseUntil.Sub(system.clock.Now()) + time.Nanosecond)
	reconciler := &restartRecoveryReconciler{external: system.external}
	restartSQLiteV15System(t, system)
	system.orchestrator = newRestartRecoveryOrchestrator(t, system, reconciler)

	firstRecovery, found, err := system.orchestrator.ClaimNextAction(
		ctx, "worker:restart-recovery:first", application.ActionClaimSelection{ExcludeLaunch: true},
	)
	if err != nil || !found || firstRecovery.Disposition != application.ActionClaimDispositionRecoverEffect ||
		firstRecovery.RecoveryEffectAttemptRef != attempt.Ref || firstRecovery.Fence <= attempt.ActionFence {
		t.Fatalf("first recovery=%+v found=%t err=%s", firstRecovery, found, sqliteTestErrorChain(err))
	}
	if result, processErr := system.orchestrator.ProcessClaim(ctx, firstRecovery); processErr != nil ||
		!result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("lost acknowledgement result=%+v err=%s", result, sqliteTestErrorChain(processErr))
	}
	assertRestartRecoveryRequeued(t, system, firstRecovery, attempt, initialCapacity, initialPlacement)
	if launches, reconciles, _ := reconciler.snapshot(); launches != 0 || reconciles != 1 {
		t.Fatalf("after lost acknowledgement Launch=%d ReconcileLaunch=%d", launches, reconciles)
	}

	system.clock.Advance(firstRecovery.Action.EffectIntent.QuotaRetryDelay)
	restartSQLiteV15System(t, system)
	system.orchestrator = newRestartRecoveryOrchestrator(t, system, reconciler)
	winner := raceRestartRecoveryClaims(t, system.orchestrator)
	if winner.Disposition != application.ActionClaimDispositionRecoverEffect ||
		winner.RecoveryEffectAttemptRef != attempt.Ref || winner.Fence <= firstRecovery.Fence ||
		winner.DeliveryAttempt != firstRecovery.DeliveryAttempt+1 {
		t.Fatalf("winning recovery claim=%+v first=%+v", winner, firstRecovery)
	}

	if result, processErr := system.orchestrator.ProcessClaim(ctx, winner); processErr != nil ||
		!result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("recovered launch result=%+v err=%s", result, sqliteTestErrorChain(processErr))
	}
	launches, reconciles, requests := reconciler.snapshot()
	if launches != 0 || reconciles != 2 || len(requests) != 2 || !reflect.DeepEqual(requests[0], requests[1]) {
		t.Fatalf("Launch=%d ReconcileLaunch=%d requests_equal=%t requests=%+v",
			launches, reconciles, len(requests) == 2 && reflect.DeepEqual(requests[0], requests[1]), requests)
	}
	wantAuthority := ports.AgentLaunchEffectAuthority{
		AuthorizationReceiptRef: first.Action.EffectIntent.Authority.Ref(),
		EffectApprovalRef:       attempt.ApprovalRef,
		EffectAttemptRef:        attempt.Ref,
		ActionFence:             attempt.ActionFence,
		StartedAt:               attempt.StartedAt,
		ClaimLeaseUntil:         attempt.ClaimLeaseUntil,
		ApprovalExpiresAt:       first.EffectApproval.ExpiresAt,
	}
	if requests[0].EffectAuthority != wantAuthority {
		t.Fatalf("historical authority=%+v want=%+v", requests[0].EffectAuthority, wantAuthority)
	}

	for name, stale := range map[string]application.ActionClaim{
		"requeued claim":   firstRecovery,
		"completed replay": winner,
	} {
		t.Run(name, func(t *testing.T) {
			if _, replayErr := system.orchestrator.ProcessClaim(ctx, stale); replayErr == nil {
				t.Fatalf("stale claim accepted: %+v", stale)
			}
			if launchCount, reconcileCount, _ := reconciler.snapshot(); launchCount != 0 || reconcileCount != 2 {
				t.Fatalf("stale claim crossed provider boundary: Launch=%d ReconcileLaunch=%d",
					launchCount, reconcileCount)
			}
		})
	}

	assertRestartRecoveryCompleted(t, system, first, winner, attempt, initialCapacity, initialPlacement)
	next, found, err := system.orchestrator.ClaimNextAction(
		ctx, "worker:restart-recovery:observe", application.ActionClaimSelection{ExcludeLaunch: true},
	)
	if err != nil || !found || next.Disposition != application.ActionClaimDispositionNormal ||
		next.Action.Kind != application.ActionObserveAgent || next.RecoveryEffectAttemptRef != "" {
		t.Fatalf("next action=%+v found=%t err=%s", next, found, sqliteTestErrorChain(err))
	}
	if _, _, err := validateRecoveryDatabase(ctx, system.repository.db); err != nil {
		t.Fatalf("completed recovery database invalid: %s", sqliteTestErrorChain(err))
	}
}

type restartRecoveryTemporaryError struct{}

func (restartRecoveryTemporaryError) Error() string   { return "sqlite.recovery_ack_lost" }
func (restartRecoveryTemporaryError) Temporary() bool { return true }

type restartRecoveryReconciler struct {
	external *sqliteV15External
	mu       sync.Mutex
	launches int
	requests []ports.AgentLaunchRequest
	receipt  ports.AgentLaunchReceipt
}

func (reconciler *restartRecoveryReconciler) Capabilities(ctx context.Context) (ports.AgentCapabilities, error) {
	return reconciler.external.Capabilities(ctx)
}

func (reconciler *restartRecoveryReconciler) Launch(
	context.Context, ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	reconciler.mu.Lock()
	defer reconciler.mu.Unlock()
	reconciler.launches++
	return ports.AgentLaunchReceipt{}, restartRecoveryTemporaryError{}
}

func (reconciler *restartRecoveryReconciler) ReconcileLaunch(
	_ context.Context, request ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	reconciler.mu.Lock()
	defer reconciler.mu.Unlock()
	reconciler.requests = append(reconciler.requests, cloneRestartRecoveryRequest(request))
	if len(reconciler.requests) == 1 {
		reconciler.receipt = restartRecoveryReceipt(request)
		return ports.AgentLaunchReceipt{}, restartRecoveryTemporaryError{}
	}
	return reconciler.receipt, nil
}

func (reconciler *restartRecoveryReconciler) snapshot() (
	int, int, []ports.AgentLaunchRequest,
) {
	reconciler.mu.Lock()
	defer reconciler.mu.Unlock()
	requests := make([]ports.AgentLaunchRequest, len(reconciler.requests))
	for index := range reconciler.requests {
		requests[index] = cloneRestartRecoveryRequest(reconciler.requests[index])
	}
	return reconciler.launches, len(reconciler.requests), requests
}

func cloneRestartRecoveryRequest(request ports.AgentLaunchRequest) ports.AgentLaunchRequest {
	request.PhaseInputRefs = append([]string(nil), request.PhaseInputRefs...)
	request.PhaseCriterionRefs = append([]string(nil), request.PhaseCriterionRefs...)
	request.SkillRefs = append([]string(nil), request.SkillRefs...)
	request.ToolRefs = append([]string(nil), request.ToolRefs...)
	request.CapabilityRefs = append([]string(nil), request.CapabilityRefs...)
	request.WriteSet = append([]string(nil), request.WriteSet...)
	return request
}

func restartRecoveryReceipt(request ports.AgentLaunchRequest) ports.AgentLaunchReceipt {
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, LaunchActionFence: request.EffectAuthority.ActionFence,
		SpecHash:                    request.SpecHash,
		ProviderRef:                 sqliteTestCapabilities().ProviderRef,
		ModelRef:                    sqliteTestCapabilities().ModelRef,
		AgentRef:                    sqliteTestCapabilities().AgentRef,
		ExternalRef:                 "external:recovered:" + request.ExecutionRef.String(),
		IdempotencyKey:              request.IdempotencyKey,
		ReceiptRef:                  "provider-receipt:recovered:" + request.ExecutionRef.String(),
		AcceptedAt:                  request.EffectAuthority.ClaimLeaseUntil.Add(-time.Nanosecond),
		RequierePreservacionEntorno: request.RequierePreservacionEntorno,
	}
}

func newRestartRecoveryOrchestrator(
	t *testing.T, system *sqliteV15System, launcher application.AgentLauncher,
) *application.Orchestrator {
	t.Helper()
	_, sources := prepararCapacidadSQLiteV15(t, system.repository, system.clock, 1_000)
	orchestrator, err := application.New(application.Dependencies{
		State: system.repository, Access: system.repository, Launcher: launcher,
		Observer: system.external, Controller: system.external, Artifacts: system.external,
		Clock: system.clock, IDs: system.ids, MaxOutputBytes: 1024,
		MaxMailboxEnvelopeBytes: 64 << 10, MaxExecutionAttempts: 3, MaxChildrenPerParent: 6,
		ClaimLease: time.Minute, DirectorLeaseDuration: 30 * time.Second,
		EffectApprovalTTL: system.policy.EffectApprovalTTL, BudgetPolicy: system.policy,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: sqliteTestCapabilities(), CapacitySources: sources,
		CapacityObservationWait: time.Second,
	})
	sqliteTestNoError(t, err)
	return orchestrator
}

type restartRecoveryClaimResult struct {
	claim application.ActionClaim
	found bool
	err   error
}

func raceRestartRecoveryClaims(
	t *testing.T, orchestrator *application.Orchestrator,
) application.ActionClaim {
	t.Helper()
	ready := make(chan struct{}, 2)
	start := make(chan struct{})
	results := make(chan restartRecoveryClaimResult, 2)
	for index := 0; index < 2; index++ {
		index := index
		go func() {
			ready <- struct{}{}
			<-start
			claim, found, err := orchestrator.ClaimNextAction(
				context.Background(), "worker:restart-recovery:race:"+string(rune('a'+index)),
				application.ActionClaimSelection{ExcludeLaunch: true},
			)
			results <- restartRecoveryClaimResult{claim: claim, found: found, err: err}
		}()
	}
	<-ready
	<-ready
	close(start)
	winners := make([]application.ActionClaim, 0, 1)
	for index := 0; index < 2; index++ {
		result := <-results
		if result.err != nil {
			t.Fatalf("concurrent recovery claim: %s", sqliteTestErrorChain(result.err))
		}
		if result.found {
			winners = append(winners, result.claim)
		} else if result.claim != (application.ActionClaim{}) {
			t.Fatalf("losing claim returned authority: %+v", result.claim)
		}
	}
	if len(winners) != 1 {
		t.Fatalf("concurrent recovery winners=%d claims=%+v", len(winners), winners)
	}
	return winners[0]
}

func assertRestartRecoveryRequeued(
	t *testing.T,
	system *sqliteV15System,
	claim application.ActionClaim,
	attempt application.EffectAttempt,
	initialCapacity application.AgentCapacityReservation,
	initialPlacement ports.AgentPlacementRef,
) {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	execution, found := sqliteExecutionByRef(record.Executions, claim.Action.ExecutionRef)
	capacity, placement, capacityFound, capacityErr := leerReservaCapacidadAccion(
		context.Background(), system.repository.db, claim.Action.Ref,
	)
	outbox := sqliteV28RecoveryRequeueOutbox(t, system.repository.db, claim.Action.Ref)
	if !found || execution.State != application.ExecutionDispatching ||
		execution.BudgetReservationRef != claim.BudgetReservationRef ||
		execution.EffectIntentRef != claim.Action.EffectIntentRef ||
		len(record.EffectAttempts) != 1 || record.EffectAttempts[0] != attempt ||
		len(record.EffectReceipts) != 0 || len(record.ConsumptionReceipts) != 0 ||
		len(record.BudgetReservations) != 1 || record.BudgetReservations[0] != claim.BudgetReservation ||
		len(record.BudgetSettlements) != 0 || capacityErr != nil || !capacityFound ||
		capacity != initialCapacity || placement != initialPlacement ||
		outbox.token.Valid || outbox.worker.Valid || outbox.lease.Valid ||
		!outbox.recoveryRef.Valid || outbox.recoveryRef.String != attempt.Ref ||
		outbox.fence != int64(claim.Fence) || outbox.deliveryAttempt != int64(claim.DeliveryAttempt) ||
		outbox.lastError != sqliteV28ReconciliationPendingCode {
		t.Fatalf("execution=%+v found=%t attempts=%+v receipts=%+v consumptions=%+v budgets=%+v settlements=%+v capacity=%+v placement=%s capacity_found=%t capacity_err=%v outbox=%+v",
			execution, found, record.EffectAttempts, record.EffectReceipts, record.ConsumptionReceipts,
			record.BudgetReservations, record.BudgetSettlements, capacity, placement,
			capacityFound, capacityErr, outbox)
	}
}

func assertRestartRecoveryCompleted(
	t *testing.T,
	system *sqliteV15System,
	first application.ActionClaim,
	winner application.ActionClaim,
	attempt application.EffectAttempt,
	initialCapacity application.AgentCapacityReservation,
	initialPlacement ports.AgentPlacementRef,
) {
	t.Helper()
	record, err := system.repository.GetGoal(context.Background(), winner.Action.GoalRef)
	sqliteTestNoError(t, err)
	execution, found := sqliteExecutionByRef(record.Executions, winner.Action.ExecutionRef)
	if !found || execution.State != application.ExecutionRunning ||
		len(record.EffectAttempts) != 1 || record.EffectAttempts[0] != attempt ||
		len(record.EffectReceipts) != 1 || len(record.ConsumptionReceipts) != 1 ||
		len(record.BudgetReservations) != 1 || record.BudgetReservations[0] != first.BudgetReservation ||
		len(record.BudgetSettlements) != 0 {
		t.Fatalf("execution=%+v found=%t attempts=%+v receipts=%+v consumptions=%+v budgets=%+v settlements=%+v",
			execution, found, record.EffectAttempts, record.EffectReceipts,
			record.ConsumptionReceipts, record.BudgetReservations, record.BudgetSettlements)
	}
	receipt, consumed := record.EffectReceipts[0], record.ConsumptionReceipts[0]
	if receipt.AttemptRef != attempt.Ref || receipt.ActionFence != attempt.ActionFence ||
		!receipt.ConfirmedAt.Equal(attempt.ClaimLeaseUntil.Add(-time.Nanosecond)) ||
		consumed.Fence != winner.Fence || consumed.EffectReceiptRef != receipt.Ref ||
		!consumed.ConsumedAt.After(receipt.ConfirmedAt) {
		t.Fatalf("attempt=%+v receipt=%+v consumed=%+v winner_fence=%d",
			attempt, receipt, consumed, winner.Fence)
	}
	capacity, placement, capacityFound, capacityErr := leerReservaCapacidadAccion(
		context.Background(), system.repository.db, winner.Action.Ref,
	)
	if capacityErr != nil || !capacityFound || placement != initialPlacement ||
		capacity.Ref != initialCapacity.Ref || capacity.Fence != initialCapacity.Fence ||
		capacity.Allocation != initialCapacity.Allocation ||
		capacity.State != application.AgentCapacityConsumed ||
		capacity.Revision != initialCapacity.Revision+1 ||
		capacity.LastCauseRef != receipt.Ref || capacity.LastTransitionRef == "" ||
		capacity.UpdatedAt != consumed.ConsumedAt || !capacity.SettledAt.IsZero() {
		t.Fatalf("capacity=%+v placement=%s found=%t err=%v initial=%+v receipt=%+v consumed=%+v",
			capacity, placement, capacityFound, capacityErr, initialCapacity, receipt, consumed)
	}
	assertRestartRecoveryCapacityTransition(t, system.repository.db, initialCapacity, receipt, consumed)
}

func assertRestartRecoveryCapacityTransition(
	t *testing.T,
	database *sql.DB,
	initial application.AgentCapacityReservation,
	receipt application.EffectReceipt,
	consumed application.ActionConsumptionReceipt,
) {
	t.Helper()
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM agent_capacity_transitions WHERE reservation_ref=?`, initial.Ref).Scan(&count); err != nil || count != 1 {
		t.Fatalf("capacity transitions=%d err=%v", count, err)
	}
	var ref, reservationRef, projectRef, outcome, cause, causeRef string
	var attemptRef, receiptRef, idempotency string
	var fence, expectedRevision, revision, recordedAt int64
	err := database.QueryRow(`
SELECT ref,reservation_ref,project_ref,fence,expected_revision,revision,outcome,cause_kind,
       cause_ref,effect_attempt_ref,effect_receipt_ref,idempotency_key,recorded_at
FROM agent_capacity_transitions WHERE reservation_ref=?`, initial.Ref).Scan(
		&ref, &reservationRef, &projectRef, &fence, &expectedRevision, &revision, &outcome, &cause,
		&causeRef, &attemptRef, &receiptRef, &idempotency, &recordedAt,
	)
	if err != nil || ref == "" || idempotency == "" || reservationRef != initial.Ref ||
		projectRef != initial.ProjectRef.String() || fence != int64(initial.Fence) ||
		expectedRevision != int64(initial.Revision) || revision != int64(initial.Revision+1) ||
		outcome != string(application.AgentCapacityConsumed) ||
		cause != string(application.AgentCapacityCauseEffectReceipt) ||
		causeRef != receipt.Ref || attemptRef != receipt.AttemptRef || receiptRef != receipt.Ref ||
		recordedAt != consumed.ConsumedAt.UnixNano() {
		t.Fatalf("transition ref=%q reservation=%q project=%q fence=%d expected=%d revision=%d outcome=%q cause=%q cause_ref=%q attempt=%q receipt=%q idempotency=%q recorded=%d err=%v",
			ref, reservationRef, projectRef, fence, expectedRevision, revision, outcome, cause,
			causeRef, attemptRef, receiptRef, idempotency, recordedAt, err)
	}
}
