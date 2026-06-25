package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorsupervisedburst "orquesta/modulos/orquesta-director-supervised-burst"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func TestMCPArrancarDirectorAppToolExecutorV0UsaServicioCanonico(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	executor := NewMCPArrancarDirectorAppToolExecutorV0(
		orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				mcpDirectorCapacityDispatcherForTestV0(store, sink, ledger),
				mcpDirectorAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)

	result, err := executor.Execute(context.Background(), validMCPDirectorAppInputForTestV0())
	if err != nil {
		var burstErr orquestadirectorsupervisedburst.DirectorSupervisedBurstErrorV0
		if errors.As(err, &burstErr) {
			t.Fatalf("Execute: %v burst=%#v result=%+v", err, burstErr, result)
		}
		t.Fatalf("Execute: %T %v result=%+v", err, err, result)
	}
	if result.Estado != MCPArrancarDirectorAppEstadoOKV0 ||
		result.DirectorExecutionMode != orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0 ||
		result.PhaseID != string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0) ||
		result.DirectorTask.AgentRequestID == "" ||
		len(result.DirectorTasks) != 1 ||
		len(result.StartedAgents) != 1 {
		t.Fatalf("result=%+v", result)
	}
	if strings.Contains(result.RequestID, "mcp") ||
		strings.Contains(result.CorrelationID, "mcp") ||
		strings.Contains(result.RunRef, "mcp") {
		t.Fatalf("refs internas no deben filtrar adaptador MCP: %+v", result)
	}
}

func TestMCPArrancarDirectorAppToolExecutorV0ExponeGoalFirst(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	launcher := &mcpGoalLauncherForTestV0{}
	executor := NewMCPArrancarDirectorAppToolExecutorV0(
		orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			GoalLauncher: launcher,
			GoalStateStore: &mcpGoalStateStoreForTestV0{
				states: map[string]orquestagoal.GoalWorkStateV0{},
			},
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				mcpDirectorCapacityDispatcherForTestV0(store, sink, ledger),
				mcpDirectorAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)

	result, err := executor.Execute(context.Background(), validMCPDirectorAppInputForTestV0())
	if err != nil {
		t.Fatalf("Execute: %v result=%+v", err, result)
	}
	if result.Estado != MCPArrancarDirectorAppEstadoOKV0 ||
		result.DirectorExecutionMode != orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0 ||
		result.RunRef == "" ||
		result.GoalRef == "" ||
		result.ExternalGoalRef != "thread-ref-mcp-goal-001" ||
		result.GoalStatus != orquestagoal.GoalStatusRunningV0 ||
		len(result.StartedAgents) != 0 {
		t.Fatalf("result=%+v", result)
	}
	if result.GoalLaunchReceipt == nil ||
		result.GoalLaunchReceipt.GoalRef != result.GoalRef ||
		launcher.calls != 1 {
		t.Fatalf("receipt=%+v calls=%d", result.GoalLaunchReceipt, launcher.calls)
	}
}

func TestMCPArrancarDirectorAppToolExecutorV0DevuelveErroresPublicos(t *testing.T) {
	input := validMCPDirectorAppInputForTestV0()
	input.AppSpecRequest.Locale = "locale invalido"

	result, err := NewMCPArrancarDirectorAppToolExecutorV0(
		orquestaappdirectorservice.StartAppDirectorPortsV0{},
	).Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("request invalida no debe ser error de transporte: %v", err)
	}
	if result.Estado != MCPArrancarDirectorAppEstadoErrorV0 || len(result.Errores) == 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPArrancarDirectorAppToolExecutorV0ExponeEquipoDirector(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	batchExecutor := &mcpDirectorBatchLauncherForTestV0{
		executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
			RunStore:      store,
			EventSink:     sink,
			Launcher:      orquestacionnucleoapp.NewFakeLifecycleAgentLauncherV0(),
			OccurredAt:    "2026-05-09T23:03:00Z",
			CorrelationID: "corr-director-agent-team-001",
			RequestedBy:   "orquesta-mcp-test",
		},
	}
	executor := NewMCPArrancarDirectorAppToolExecutorV0(
		orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				mcpDirectorCapacityDispatcherForTestV0(store, sink, ledger),
			},
			BatchDispatchers: []orquestacionnucleoapp.OutboxBatchDispatcherBindingV0{{
				TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
				MaxReady:   4,
				Reader:     ledger,
				Claimer:    ledger,
				Executor:   batchExecutor,
				Acker:      ledger,
			}},
		},
	)
	input := validMCPDirectorAppInputForTestV0()
	input.AppSpecRequest.RequestID = "request-ref-mcp-director-team-001"
	input.AppSpecRequest.Datos = orquestafactory.DatosRequestV0{
		DBRequired:         true,
		NecesidadFuncional: "Guardar contactos y citas mediante un puerto de persistencia.",
	}
	input.AppSpecRequest.Calidad.Pruebas = "alta"
	input.AppSpecRequest.Agentes = orquestafactory.AgentesRequestV0{Autonomia: "alta"}

	result, err := executor.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v result=%+v", err, result)
	}
	if result.Estado != MCPArrancarDirectorAppEstadoOKV0 ||
		len(result.DirectorTasks) != 4 ||
		len(result.StartedAgents) != 4 {
		t.Fatalf("result=%+v", result)
	}
	if batchExecutor.totalBatchSize != 4 || batchExecutor.maxBatchSize > 4 {
		t.Fatalf("batch total=%d max=%d", batchExecutor.totalBatchSize, batchExecutor.maxBatchSize)
	}
}

func TestMCPArrancarDirectorAppHTTPHandlerV0SirveBridgeREST(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	executor := NewMCPArrancarDirectorAppToolExecutorV0(
		orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				mcpDirectorCapacityDispatcherForTestV0(store, sink, ledger),
				mcpDirectorAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(validMCPDirectorAppInputForTestV0()); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPArrancarDirectorAppHTTPPathV0, body)
	req.Header.Set("X-Correlation-ID", "corr-http-director-001")

	NewMCPArrancarDirectorAppHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Correlation-ID"); got == "" {
		t.Fatalf("correlation header vacio")
	}
	var result MCPArrancarDirectorAppToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPArrancarDirectorAppEstadoOKV0 ||
		result.RunRef == "" ||
		len(result.StartedAgents) != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPArrancarDirectorAppHTTPHandlerV0RechazaOKSinRunRef(t *testing.T) {
	executor := &fakeMCPArrancarDirectorAppHTTPExecutorV0{
		result: MCPArrancarDirectorAppToolResultV0{
			Estado:                MCPArrancarDirectorAppEstadoOKV0,
			CorrelationID:         "corr-http-director-sin-run-001",
			DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
			GoalRef:               "goal-ref-http-director-sin-run-001",
			GoalStatus:            orquestagoal.GoalStatusRunningV0,
		},
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(validMCPDirectorAppInputForTestV0()); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPArrancarDirectorAppHTTPPathV0, body)
	req.Header.Set("X-Correlation-ID", "corr-http-director-sin-run-001")

	NewMCPArrancarDirectorAppHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPArrancarDirectorAppToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPArrancarDirectorAppEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "run_ref_requerido" ||
		result.Errores[0].Field != "run_ref" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPArrancarDirectorAppTransportV0NormalizaOKSinRunRef(t *testing.T) {
	executor := &fakeMCPArrancarDirectorAppHTTPExecutorV0{
		result: MCPArrancarDirectorAppToolResultV0{
			Estado:                MCPArrancarDirectorAppEstadoOKV0,
			DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
			GoalRef:               "goal-ref-transport-director-sin-run-001",
			GoalStatus:            orquestagoal.GoalStatusRunningV0,
		},
	}
	input := validMCPDirectorAppInputForTestV0()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	payload, err := mcpArrancarDirectorAppTransportHandlerV0(executor)(context.Background(), raw)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	var result MCPArrancarDirectorAppToolResultV0
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.Estado != MCPArrancarDirectorAppEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "run_ref_requerido" ||
		result.Errores[0].Field != "run_ref" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPArrancarDirectorAppHTTPHandlerV0ExecutorErrorExponeCampoPublico(t *testing.T) {
	executor := &fakeMCPArrancarDirectorAppHTTPExecutorV0{
		err: orquestaappdirectorservice.AppDirectorServiceIssueV0{Field: "ports.run_store"},
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(validMCPDirectorAppInputForTestV0()); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, MCPArrancarDirectorAppHTTPPathV0, body)

	NewMCPArrancarDirectorAppHTTPHandlerV0(executor).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result MCPArrancarDirectorAppToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPArrancarDirectorAppEstadoErrorV0 || len(result.Errores) != 1 {
		t.Fatalf("result=%+v", result)
	}
	issue := result.Errores[0]
	if issue.Code != "arrancar_director_http_error" ||
		issue.Field != "ports.run_store" ||
		!strings.Contains(issue.Message, "ports.run_store") {
		t.Fatalf("issue=%+v", issue)
	}
}

type mcpGoalLauncherForTestV0 struct {
	calls int
}

func (launcher *mcpGoalLauncherForTestV0) LaunchGoalWorkV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	launcher.calls++
	return orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: "thread-ref-mcp-goal-001",
		EvidenceRefs:    []string{"evidence-ref-mcp-goal-launched-001"},
	}, nil
}

type mcpGoalStateStoreForTestV0 struct {
	states map[string]orquestagoal.GoalWorkStateV0
}

func (store *mcpGoalStateStoreForTestV0) SaveGoalWorkStateV0(
	_ context.Context,
	state orquestagoal.GoalWorkStateV0,
) error {
	normalized, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return err
	}
	store.states[normalized.RunRef] = normalized
	return nil
}

func (store *mcpGoalStateStoreForTestV0) LoadGoalWorkStateV0(
	_ context.Context,
	runRef string,
) (orquestagoal.GoalWorkStateV0, error) {
	return store.states[runRef], nil
}

func validMCPDirectorAppInputForTestV0() MCPArrancarDirectorAppToolInputV0 {
	observability := true
	return MCPArrancarDirectorAppToolInputV0{
		RequestID:     "request-ref-mcp-director-001",
		CorrelationID: "corr-mcp-director-001",
		AppSpecRequest: orquestafactory.AppSpecRequestV0{
			SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
			RequestID:     "request-ref-mcp-director-001",
			Source:        "orquesta-web",
			Locale:        "es-ES",
			Nombre:        "Agenda",
			Objetivo:      "Gestionar contactos y citas desde una API y una web.",
			TipoApp:       "mixed",
			PreferenciasTecnicas: orquestafactory.PreferenciasTecnicasV0{
				Lenguaje:     "go",
				Arquitectura: "hexagonal",
			},
			Calidad: orquestafactory.CalidadRequestV0{
				Pruebas:        "media",
				Accesibilidad:  "basica",
				Observabilidad: &observability,
			},
		},
	}
}

type fakeMCPArrancarDirectorAppHTTPExecutorV0 struct {
	result MCPArrancarDirectorAppToolResultV0
	err    error
}

func (executor *fakeMCPArrancarDirectorAppHTTPExecutorV0) Execute(
	_ context.Context,
	_ MCPArrancarDirectorAppToolInputV0,
) (MCPArrancarDirectorAppToolResultV0, error) {
	return executor.result, executor.err
}

func mcpDirectorCapacityDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.CapacityDecisionExecutorV0{
			RunStore:        store,
			EventSink:       sink,
			Tier:            orquestacoreworkflow.OrchestrationCapacityHighV0,
			ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityHighV0,
			OccurredAt:      "2026-05-09T23:00:00Z",
			CorrelationID:   "corr-director-capacity-001",
			RequestedBy:     "orquesta-mcp-test",
		},
		Acker: ledger,
	}
}

func mcpDirectorAgentLauncherDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
			RunStore:      store,
			EventSink:     sink,
			Launcher:      orquestacionnucleoapp.NewFakeLifecycleAgentLauncherV0(),
			OccurredAt:    "2026-05-09T23:01:00Z",
			CorrelationID: "corr-director-agent-001",
			RequestedBy:   "orquesta-mcp-test",
		},
		Acker: ledger,
	}
}

type mcpDirectorBatchLauncherForTestV0 struct {
	executor       orquestacionnucleoapp.AgentLauncherExecutorV0
	totalBatchSize int
	maxBatchSize   int
}

func (executor *mcpDirectorBatchLauncherForTestV0) ExecuteOutboxDispatchBatchV0(
	ctx context.Context,
	intents []orquestaoutboxdispatch.DispatchIntentV0,
) ([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0, error) {
	executor.totalBatchSize += len(intents)
	if len(intents) > executor.maxBatchSize {
		executor.maxBatchSize = len(intents)
	}
	acks := make([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0, 0, len(intents))
	for _, intent := range intents {
		execution, err := executor.executor.ExecuteOutboxDispatchV0(intent)
		if err != nil {
			return nil, err
		}
		acks = append(acks, orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
			MessageID:    intent.MessageID,
			RunID:        intent.RunID,
			TargetPort:   intent.TargetPort,
			Status:       orquestaoutboxdispatch.OutboxDispatchAckObservationSuccessV0,
			DispatchRef:  execution.DispatchRef,
			EvidenceRefs: execution.EvidenceRefs,
		})
	}
	return acks, nil
}
