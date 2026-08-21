package acceptance_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	agentefalso "orquesta/internal/adapters/agent/fake"
	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func TestV38B11StopReintentaSoloTrasPruebaDurableSinLiquidarPresupuesto(t *testing.T) {
	for _, caso := range []struct {
		nombre, fallo  string
		quiereIntentos int
	}{
		{nombre: "definitely_not_applied", fallo: "definite", quiereIntentos: 2},
		{nombre: "unknown_applied", fallo: "ambiguous", quiereIntentos: 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			sistema := nuevoSistemaCompuertaAV38(t, 1)
			controlador := &controladorStopB11{Adapter: sistema.agente, fallo: caso.fallo}
			sistema.orquestador = componerStopB11(t, sistema, controlador)
			enviado, err := sistema.orquestador.Submit(context.Background(), v06Access(t), application.SubmitRequest{
				RequestRef: "request:v38-b11-retry:" + caso.nombre, Statement: "probar retry durable de stop",
				Confirm: true, Plan: planCompuertaAV38(1),
			})
			if err != nil {
				t.Fatal(err)
			}
			var record application.GoalRecord
			for indice := 0; indice < 3; indice++ {
				if resultado, err := sistema.orquestador.ProcessNext(context.Background(), "worker:v38-b11-launch"); err != nil || !resultado.Processed {
					t.Fatalf("preparar/lanzar %d: resultado=%+v err=%v", indice, resultado, err)
				}
				record, err = sistema.repositorio.GetGoal(context.Background(), enviado.Record.Goal.Ref())
				if err != nil {
					t.Fatal(err)
				}
				if record.Executions[0].State == application.ExecutionRunning {
					break
				}
			}
			if record.Executions[0].State != application.ExecutionRunning {
				t.Fatalf("el agente no quedó ejecutando: %+v", record.Executions[0])
			}
			item := record.Goal.WorkItems()[0]
			execution := record.Executions[0]
			_, err = sistema.orquestador.Control(context.Background(), v06Access(t), application.ControlRequest{
				RequestRef: "control:v38-b11-retry:" + caso.nombre, Operation: application.ControlStop,
				Target: application.ControlTargetExecution, GoalRef: record.Goal.Ref(),
				ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
				ExpectedAppSpecGeneration: record.Goal.AppSpec().Generation(), ExpectedSpecHash: record.Goal.SpecHash(),
				WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(), ExecutionRef: execution.Ref,
				ExpectedExecutionAttempt: execution.AttemptNo, Mode: ports.AgentStopCooperative, Reason: "retry durable",
			})
			if err != nil {
				t.Fatalf("crear stop: err=%v goal=%s/%d item=%s/%d execution=%s/%d",
					err, record.Goal.State(), record.Goal.Revision(), item.State(), item.Revision(), execution.State, execution.AttemptNo)
			}
			liquidacionesAntes, receiptsAntes := len(record.BudgetSettlements), len(record.EffectReceipts)
			resultadoPrimero, errorPrimero := sistema.orquestador.ProcessNext(context.Background(), "worker:v38-b11-stop-1")
			if !resultadoPrimero.Processed || (caso.fallo == "definite" && errorPrimero != nil) ||
				(caso.fallo == "ambiguous" && (errorPrimero == nil || errorPrimero.Error() != "application.effect_unknown_applied")) {
				t.Fatalf("primer stop: resultado=%+v err=%v", resultadoPrimero, errorPrimero)
			}
			intermedio, err := sistema.repositorio.GetGoal(context.Background(), record.Goal.Ref())
			if err != nil {
				t.Fatal(err)
			}
			if len(intermedio.BudgetSettlements) != liquidacionesAntes || len(intermedio.EffectReceipts) != receiptsAntes ||
				(len(intermedio.EffectAttemptOutcomes) == 1) != (caso.fallo == "definite") {
				t.Fatalf("ledger incorrecto tras fallo: outcomes=%d settlements=%d receipts=%d",
					len(intermedio.EffectAttemptOutcomes), len(intermedio.BudgetSettlements), len(intermedio.EffectReceipts))
			}

			sistema.reabrir(t)
			sistema.orquestador = componerStopB11(t, sistema, controlador)
			sistema.reloj.Advance(2 * sistema.politica.QuotaRetryDelay)
			resultado, segundoErr := sistema.orquestador.ProcessNext(context.Background(), "worker:v38-b11-stop-2")
			if caso.fallo == "definite" && (segundoErr != nil || !resultado.Processed || resultado.Action != application.ActionStopAgent) {
				t.Fatalf("prueba durable no reabrió stop: resultado=%+v err=%v", resultado, segundoErr)
			}
			if caso.fallo == "ambiguous" && (segundoErr != nil ||
				(resultado.Processed && resultado.Action == application.ActionStopAgent)) {
				t.Fatalf("unknown_applied se reintentó: resultado=%+v err=%v", resultado, segundoErr)
			}
			controlador.mu.Lock()
			defer controlador.mu.Unlock()
			if len(controlador.claves) != caso.quiereIntentos {
				t.Fatalf("invocaciones stop=%d, quiere=%d", len(controlador.claves), caso.quiereIntentos)
			}
			if len(controlador.claves) == 2 && controlador.claves[0] != controlador.claves[1] {
				t.Fatalf("retry cambió idempotencia: %q != %q", controlador.claves[0], controlador.claves[1])
			}
		})
	}
}

type controladorStopB11 struct {
	*agentefalso.Adapter
	mu     sync.Mutex
	fallo  string
	claves []string
}

func (controlador *controladorStopB11) Stop(ctx context.Context, request ports.AgentStopRequest) (ports.AgentStopReceipt, error) {
	controlador.mu.Lock()
	controlador.claves = append(controlador.claves, request.IdempotencyKey)
	intento := len(controlador.claves)
	controlador.mu.Unlock()
	if controlador.fallo == "ambiguous" || intento == 1 {
		if controlador.fallo == "definite" {
			return ports.AgentStopReceipt{}, stopNoAplicadoB11{}
		}
		return ports.AgentStopReceipt{}, errors.New("ack de stop desconocido")
	}
	return controlador.Adapter.Stop(ctx, request)
}

type stopNoAplicadoB11 struct{}

func (stopNoAplicadoB11) Error() string              { return "stop no aplicado" }
func (stopNoAplicadoB11) DefinitelyNotApplied() bool { return true }

func componerStopB11(t *testing.T, sistema *sistemaCompuertaAV38, agente application.AgentController) *application.Orchestrator {
	t.Helper()
	capacidades, err := sistema.agente.Capabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	orquestador, err := application.New(application.Dependencies{
		State: sistema.repositorio, Access: sistema.repositorio,
		Launcher: sistema.agente, Observer: sistema.agente, Controller: agente,
		Artifacts: sistema.artefactos, Clock: sistema.reloj, IDs: sistema.identidades,
		MaxOutputBytes: 1 << 20, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, MaxChildrenPerParent: 6,
		ClaimLease: time.Minute, AttestTestClaimLease: time.Minute,
		DirectorLeaseDuration: 2 * time.Minute, EffectApprovalTTL: sistema.politica.EffectApprovalTTL,
		BudgetPolicy: sistema.politica, ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities:       capacidades,
		CapacityObservationWait: time.Second,
		CapacitySources: []application.FuenteCapacidadColocacionAgente{{
			PlacementRef: sistema.colocacion, SourceRef: "capacity-source:v38-neutral",
			PoolRef: "capacity-pool:v38-neutral", BaseMedicion: application.BaseMedicionCapacidadBruta,
			Observer: sistema.observador,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return orquestador
}
