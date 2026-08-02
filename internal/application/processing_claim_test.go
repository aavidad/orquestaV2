// Este fichero acredita que el reclamo global y el despacho interno conservan
// el contrato público de ProcessNext al separarse en dos pasos privados.
package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestProcessNextPreservesClaimRequestAndEmptyOutcomes(t *testing.T) {
	now := time.Date(2026, 7, 30, 22, 0, 0, 0, time.UTC)
	expectedError := errors.New("state.claim_failed")
	cases := []struct {
		name    string
		err     error
		wantErr error
	}{
		{name: "not_found"},
		{name: "error", err: expectedError, wantErr: expectedError},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			state := &processingClaimState{claimErr: test.err}
			capabilities := ports.AgentCapabilities{
				ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test",
				RoleKeys: []string{"role:worker"}, SkillRefs: []string{"skill:test"},
			}
			policy := testBudgetPolicy(now)
			orchestrator := &Orchestrator{
				state: state, ids: &sequentialIDs{}, claimLease: 3 * time.Minute,
				attestTestClaimLease: 7 * time.Minute,
				agentCapabilities:    capabilities, budgetPolicy: policy,
			}

			result, err := orchestrator.ProcessNext(context.Background(), "worker:exact")

			if result != (ProcessResult{}) || !errors.Is(err, test.wantErr) {
				t.Fatalf("outcome result=%+v err=%v", result, err)
			}
			wantRequest := ClaimRequest{
				WorkerRef: "worker:exact", Token: "claim:001", LeaseDuration: 3 * time.Minute,
				AttestTestLeaseDuration: 7 * time.Minute,
				Capabilities:            capabilities, BudgetPolicy: policy,
			}
			if state.claimCalls != 1 || !reflect.DeepEqual(state.request, wantRequest) {
				t.Fatalf("claim calls=%d request=%+v want=%+v", state.claimCalls, state.request, wantRequest)
			}
		})
	}
}

func TestClaimNextActionPresentaCapacidadYFallaCerradoSoloParaLanzamientos(t *testing.T) {
	now := agentCapacityBaseTime().Add(time.Minute)
	for _, caso := range []struct {
		nombre                         string
		accion                         ActionKind
		excluir, ausente, errorCuota   bool
		agotada, obsoleta, errorFuente bool
	}{
		{nombre: "disponible", accion: ActionStopAgent},
		{nombre: "control_explicito", accion: ActionObserveAgent, excluir: true},
		{nombre: "ausente", accion: ActionStopAgent, ausente: true},
		{nombre: "error_cuota", accion: ActionStopAgent, errorCuota: true},
		{nombre: "agotada", accion: ActionStopAgent, agotada: true},
		{nombre: "obsoleta", accion: ActionStopAgent, obsoleta: true},
		{nombre: "fuente", accion: ActionObserveAgent, errorFuente: true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			estado := &processingClaimState{claim: ActionClaim{Action: ActionRecord{Kind: caso.accion}}, found: true, cuotas: make(map[string]AgentQuotaObservationRecord)}
			fuente, observador := prepararFuenteCapacidadClaim(t, estado, now, caso.nombre)
			cuota := estado.cuotas[fuente.PlacementRef.String()]
			switch {
			case caso.ausente:
				delete(estado.cuotas, fuente.PlacementRef.String())
			case caso.errorCuota:
				estado.quotaErr = errors.New("state.quota_unavailable")
			case caso.agotada:
				cuota.Status = AgentQuotaExhausted
				estado.cuotas[fuente.PlacementRef.String()] = cuota
			case caso.obsoleta:
				cuota.ExpiresAt = now
				estado.cuotas[fuente.PlacementRef.String()] = cuota
			case caso.errorFuente:
				observador.err = errors.New("source.unavailable")
			}
			orquestador := &Orchestrator{state: estado, ids: &sequentialIDs{}, clock: agentCapacityClock{now: now}, claimLease: time.Minute,
				budgetPolicy: testBudgetPolicy(now), capacitySources: []FuenteCapacidadColocacionAgente{fuente}, capacityObservationWait: time.Second}
			reclamo, encontrado, err := orquestador.ClaimNextAction(context.Background(), "worker:control", ActionClaimSelection{ExcludeLaunch: caso.excluir})
			admitida := caso.nombre == "disponible"
			if err != nil || !encontrado || reclamo.Action.Kind != caso.accion || estado.request.ExcludeLaunch == admitida ||
				(len(estado.request.CapacityCandidates) == 1) != admitida {
				t.Fatalf("reclamo=%+v encontrado=%v request=%+v error=%v", reclamo, encontrado, estado.request, err)
			}
		})
	}
}

func TestClaimNextActionAplicaUnSoloTimeoutAlLoteDeFuentes(t *testing.T) {
	now, espera := agentCapacityBaseTime().Add(time.Minute), 20*time.Millisecond
	estado := &processingClaimState{cuotas: make(map[string]AgentQuotaObservationRecord)}
	var fuentes []FuenteCapacidadColocacionAgente
	for indice := range 20 {
		fuente, observador := prepararFuenteCapacidadClaim(t, estado, now, fmt.Sprintf("timeout:%d", indice))
		observador.esperar = true
		fuentes = append(fuentes, fuente)
	}
	orquestador := &Orchestrator{state: estado, ids: &sequentialIDs{}, clock: agentCapacityClock{now: now}, claimLease: time.Minute,
		budgetPolicy: testBudgetPolicy(now), capacitySources: fuentes, capacityObservationWait: espera}
	inicio := time.Now()
	if _, _, err := orquestador.ClaimNextAction(context.Background(), "worker:timeout", ActionClaimSelection{}); err != nil {
		t.Fatal(err)
	}
	if duracion := time.Since(inicio); duracion >= 5*espera || !estado.request.ExcludeLaunch {
		t.Fatalf("timeout multiplicado: duración=%s request=%+v", duracion, estado.request)
	}
}

func TestProcessClaimPreservesDispositionAndDispatch(t *testing.T) {
	goalRef, err := goal.NewGoalRef("goal:processing-claim")
	if err != nil {
		t.Fatal(err)
	}
	knownError := errors.New("state.known_action")
	cases := []struct {
		name              string
		kind              ActionKind
		disposition       ActionClaimDisposition
		getGoalErr        error
		wantErr           error
		wantStateConflict bool
		wantGetGoal       int
		wantQuarantine    int
	}{
		{
			name: "irreversible", kind: ActionLaunchAgent,
			disposition: ActionClaimDispositionRetryBudgetIrreversible,
			getGoalErr:  knownError, wantErr: knownError, wantGetGoal: 1,
		},
		{
			name: "invalid_disposition", kind: ActionLaunchAgent,
			disposition: "invalid", wantStateConflict: true,
		},
		{
			name: "known_action", kind: ActionRevokeSession,
			getGoalErr: knownError, wantErr: knownError, wantGetGoal: 1,
		},
		{
			name: "unknown_action", kind: ActionKind("unknown"),
			wantQuarantine: 1,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			state := &processingClaimState{getGoalErr: test.getGoalErr}
			orchestrator := &Orchestrator{
				state: state,
				clock: &mutableClock{now: time.Date(2026, 7, 30, 22, 0, 0, 0, time.UTC)},
			}
			claim := ActionClaim{
				Action:      ActionRecord{Ref: "action:test", GoalRef: goalRef, Kind: test.kind},
				Disposition: test.disposition, DeliveryAttempt: 1,
			}

			result, processErr := orchestrator.processClaim(context.Background(), claim)

			if !result.Processed || result.GoalRef != goalRef || result.Action != test.kind {
				t.Fatalf("result=%+v", result)
			}
			if test.wantStateConflict {
				if !IsStateError(processErr, StateConflict) {
					t.Fatalf("error=%v want state conflict", processErr)
				}
			} else if test.wantErr != nil {
				if !errors.Is(processErr, test.wantErr) {
					t.Fatalf("error=%v want=%v", processErr, test.wantErr)
				}
			} else if processErr == nil || processErr.Error() != "application.action_kind_invalid:unknown" {
				t.Fatalf("unknown action error=%v", processErr)
			}
			if state.getGoalCalls != test.wantGetGoal || state.quarantineCalls != test.wantQuarantine {
				t.Fatalf("calls get_goal=%d quarantine=%d", state.getGoalCalls, state.quarantineCalls)
			}
		})
	}
}

func TestProcessNextDispatchesOneFoundClaimWithSameContext(t *testing.T) {
	goalRef, err := goal.NewGoalRef("goal:processing-found")
	if err != nil {
		t.Fatal(err)
	}
	witness := errors.New("state.get_goal_witness")
	claim := ActionClaim{
		Action: ActionRecord{
			Ref:  "action:revoke-execution-session:execution:test",
			Kind: ActionRevokeSession, GoalRef: goalRef,
		},
		Token: "claim:durable", WorkerRef: "worker:durable", DeliveryAttempt: 3, Fence: 2,
	}
	state := &processingClaimState{claim: claim, found: true, getGoalErr: witness}
	ids := &processingClaimIDs{}
	orchestrator := &Orchestrator{state: state, ids: ids}
	ctx := context.WithValue(context.Background(), processingContextKey{}, "same")

	result, processErr := orchestrator.ProcessNext(ctx, "worker:dispatch")

	wantResult := ProcessResult{Processed: true, GoalRef: goalRef, Action: ActionRevokeSession}
	if result != wantResult || !errors.Is(processErr, witness) || !reflect.DeepEqual(state.claim, claim) {
		t.Fatalf("result=%+v error=%v claim=%+v", result, processErr, state.claim)
	}
	if ids.calls != 1 || state.claimCalls != 1 || state.getGoalCalls != 1 ||
		state.claimContext != ctx || state.getGoalContext != ctx || state.getGoalRef != goalRef {
		t.Fatalf("calls ids=%d claim=%d dispatch=%d same_context=%v/%v goal=%s",
			ids.calls, state.claimCalls, state.getGoalCalls,
			state.claimContext == ctx, state.getGoalContext == ctx, state.getGoalRef)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if replay, replayErr := orchestrator.ProcessNext(canceled, "worker:dispatch"); replay != (ProcessResult{}) ||
		!errors.Is(replayErr, context.Canceled) || ids.calls != 2 ||
		state.claimCalls != 1 || state.getGoalCalls != 1 {
		t.Fatalf("canceled replay=%+v err=%v ids=%d claim=%d dispatch=%d",
			replay, replayErr, ids.calls, state.claimCalls, state.getGoalCalls)
	}
}

type processingContextKey struct{}

type processingClaimIDs struct {
	sequentialIDs
	calls int
}

func (ids *processingClaimIDs) NewID(ctx context.Context, prefix string) (string, error) {
	ids.calls++
	return ids.sequentialIDs.NewID(ctx, prefix)
}

type processingClaimState struct {
	StateRepository
	request         ClaimRequest
	claim           ActionClaim
	found           bool
	claimErr        error
	getGoalErr      error
	claimCalls      int
	getGoalCalls    int
	quarantineCalls int
	claimContext    context.Context
	getGoalContext  context.Context
	getGoalRef      goal.GoalRef
	cuotas          map[string]AgentQuotaObservationRecord
	quotaErr        error
}

func (state *processingClaimState) CurrentAgentQuotaObservation(_ context.Context, colocacion ports.AgentPlacementRef) (AgentQuotaObservationRecord, bool, error) {
	cuota, encontrada := state.cuotas[colocacion.String()]
	return cuota, encontrada, state.quotaErr
}

func (state *processingClaimState) ClaimNextAction(
	ctx context.Context,
	request ClaimRequest,
) (ActionClaim, bool, error) {
	state.claimCalls++
	state.claimContext = ctx
	state.request = request
	return state.claim, state.found, state.claimErr
}

func (state *processingClaimState) GetGoal(
	ctx context.Context,
	goalRef goal.GoalRef,
) (GoalRecord, error) {
	state.getGoalCalls++
	state.getGoalContext = ctx
	state.getGoalRef = goalRef
	return GoalRecord{}, state.getGoalErr
}

func (state *processingClaimState) QuarantineAction(
	_ context.Context,
	_ ActionQuarantinedState,
) error {
	state.quarantineCalls++
	return nil
}

type observadorCapacidadClaim struct {
	observacion AgentCapacityObservation
	err         error
	esperar     bool
}

func prepararFuenteCapacidadClaim(t *testing.T, estado *processingClaimState, ahora time.Time, nombre string) (FuenteCapacidadColocacionAgente, *observadorCapacidadClaim) {
	colocacion, _ := ports.NewAgentPlacementRef("placement:" + nombre)
	cuota := placementQuotaRecord(ahora)
	cuota.PlacementRef, cuota.Ref, cuota.IdempotencyKey = colocacion, "quota:"+nombre, "quota-key:"+nombre
	estado.cuotas[colocacion.String()] = cuota
	observacion := validAgentCapacityObservation(t)
	observacion.PoolRef = AgentCapacityPoolRef("pool:" + nombre)
	observador := &observadorCapacidadClaim{observacion: observacion}
	return FuenteCapacidadColocacionAgente{colocacion, observacion.SourceRef, observacion.PoolRef,
		BaseMedicionCapacidadBruta, observador}, observador
}

func (observador *observadorCapacidadClaim) ObserveCapacity(ctx context.Context, _ AgentCapacitySourceRef, _ AgentCapacityPoolRef) (AgentCapacityObservation, error) {
	if observador.esperar {
		<-ctx.Done()
		return AgentCapacityObservation{}, ctx.Err()
	}
	return observador.observacion, observador.err
}
