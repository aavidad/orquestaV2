package orquestaappgateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	orquestacore "orquesta/modulos/orquesta-core"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaweb "orquesta/modulos/orquesta-web"
)

func TestNuevaAppPOSTDelegaEnAppDirectorRESTSinCmdDBRuntimeV0(t *testing.T) {
	executor := &recordingArrancarDirectorExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		ArrancarDirector: executor,
		DirectorLimits: orquestaweb.WebArrancarDirectorAppLimitsV0{
			MaxBursts:            4,
			MaxStepsPerBurst:     3,
			MaxDispatchesPerWait: 2,
			MaxCommands:          9,
			MaxOutboxPerCycle:    5,
			MaxExternalWaits:     1,
		},
		Timeout: time.Second,
	})
	values := appGatewayNuevaAppFormValuesV0()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nueva-app", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.Input.AppSpecRequest.Nombre != "Agenda Equipo" ||
		executor.Input.AppSpecRequest.PreferenciasTecnicas.Arquitectura != "hexagonal" {
		t.Fatalf("input=%+v", executor.Input.AppSpecRequest)
	}
	if executor.Input.DirectorExecutionMode != "" {
		t.Fatalf("director_execution_mode normal debe quedar vacio para goal-first automatico: %q", executor.Input.DirectorExecutionMode)
	}
	if executor.Input.MaxBursts != 4 ||
		executor.Input.MaxStepsPerBurst != 3 ||
		executor.Input.MaxDispatchesPerWait != 2 ||
		executor.Input.MaxCommands != 9 ||
		executor.Input.MaxOutboxPerCycle != 5 ||
		executor.Input.MaxExternalWaits != 1 {
		t.Fatalf("limits=%+v", executor.Input)
	}
	body := rec.Body.String()
	for _, want := range []string{"Director arrancado", "Agenda Equipo"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body no contiene %q\n%s", want, body)
		}
	}
}

func TestAppDirectorGoalObserveAPIDelegaEnExecutorRESTV0(t *testing.T) {
	executor := &recordingObserveDirectorGoalExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		ObserveDirectorGoal: executor,
		Timeout:             time.Second,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		orquestamcp.MCPObserveAppDirectorGoalHTTPPathV0,
		strings.NewReader(`{"request_id":"req-goal-observe-app-gateway-001","run_ref":"run-ref-goal-observe-app-gateway-001"}`),
	)
	req.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.Input.RunRef != "run-ref-goal-observe-app-gateway-001" {
		t.Fatalf("input=%+v", executor.Input)
	}
	var result orquestamcp.MCPObserveAppDirectorGoalToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != orquestamcp.MCPObserveAppDirectorGoalEstadoOKV0 ||
		result.GoalRef != "goal-ref-app-gateway-observed-001" ||
		result.RunStatus != "closed" {
		t.Fatalf("result=%+v", result)
	}
}

func TestAppDirectorPreviewAPIDelegaEnExecutorRESTV0(t *testing.T) {
	executor := &recordingPreviewDirectorExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		PreviewDirector: executor,
		Timeout:         time.Second,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		orquestamcp.MCPPreviewDirectorAppHTTPPathV0,
		strings.NewReader(`{"request_id":"req-preview-app-gateway-001","app_spec_request":{"schema_version":"app_spec_request.v0","request_id":"req-preview-app-gateway-001","nombre":"Agenda","objetivo":"Coordinar ensayos","tipo_app":"web","locale":"es-ES"}}`),
	)
	req.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.Input.RequestID != "req-preview-app-gateway-001" ||
		executor.Input.AppSpecRequest.Nombre != "Agenda" {
		t.Fatalf("input=%+v", executor.Input)
	}
}

func TestDirectorStatsPageDelegaEnDirectorStatsRESTSinCmdDBRuntimeV0(t *testing.T) {
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:         "run-app-gateway-stats-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-app-gateway-stats-001", "task-app-gateway-stats-002"},
		ClosedTasks:   []string{"task-app-gateway-stats-001"},
		Agents:        []string{"agent-app-gateway-stats-001"},
		StartedAgents: []string{"agent-app-gateway-stats-001"},
	}
	handler := NewHTTPHandlerV0(ConfigV0{
		DirectorStats: orquestamcp.MCPDirectorStatsToolExecutorV0{
			RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		},
		Timeout: time.Second,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/director-stats?run_ref="+url.QueryEscape(run.RunID)+"&include_agent_progress=true",
		nil,
	)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var page orquestaweb.WebDirectorStatsPageV0
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if page.ViewModel.RunRef != run.RunID ||
		page.ViewModel.FaseActual != string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0) ||
		page.ViewModel.Counts.TasksTotal != 2 ||
		page.ViewModel.Counts.TasksOpen != 1 ||
		page.ViewModel.Counts.AgentsStarted != 1 {
		t.Fatalf("page=%+v", page)
	}

	apiRec := httptest.NewRecorder()
	apiReq := httptest.NewRequest(
		http.MethodPost,
		"/api/v0/director/stats",
		strings.NewReader(`{"request_id":"request-ref-app-gateway-stats-001","correlation_id":"corr-app-gateway-stats-001","run_ref":"`+run.RunID+`"}`),
	)
	apiReq.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(apiRec, apiReq)
	if apiRec.Code != http.StatusOK {
		t.Fatalf("api status=%d body=%s", apiRec.Code, apiRec.Body.String())
	}
	var apiResult orquestamcp.MCPDirectorStatsToolResultV0
	if err := json.NewDecoder(apiRec.Body).Decode(&apiResult); err != nil {
		t.Fatalf("decode api stats: %v", err)
	}
	if apiResult.DecisionContext == nil ||
		apiResult.DecisionContext.Progress.TasksOpen != 1 ||
		apiResult.DecisionContext.Lifecycle.AgentsRunning != 1 ||
		len(apiResult.DecisionContext.Activity) == 0 {
		t.Fatalf("decision_context incompleto=%+v", apiResult.DecisionContext)
	}
	if err := orquestaobservability.ValidateDirectorDecisionContextV0(*apiResult.DecisionContext); err != nil {
		t.Fatalf("decision_context invalido: %v", err)
	}
}

func TestDomainWorkAPIDelegaEnExecutorRESTSinOPESNiDBRuntimeV0(t *testing.T) {
	executor := &recordingDomainWorkExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		DomainWork: executor,
		Timeout:    time.Second,
	})
	input := orquestamcp.MCPDomainWorkToolInputV0{
		RequestID:     "request-ref-domain-work-app-gateway-001",
		CorrelationID: "corr-domain-work-app-gateway-001",
		Action:        orquestamcp.MCPDomainWorkActionCreateJobV0,
		JobRequest: orquestadomainwork.DomainWorkJobRequestV0{
			SchemaVersion:  orquestadomainwork.DomainWorkJobRequestSchemaV0,
			RequestID:      "request-ref-domain-work-app-gateway-001",
			CorrelationID:  "corr-domain-work-app-gateway-001",
			IdempotencyKey: "idem-domain-work-app-gateway-001",
			RequestedBy:    "director",
			DomainRef:      "domain-opes",
			WorkKind:       "syllabus",
			Objective:      "crear temario inicial",
		},
	}
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/domain-work", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.Input.Action != orquestamcp.MCPDomainWorkActionCreateJobV0 ||
		executor.Input.JobRequest.DomainRef != "domain-opes" {
		t.Fatalf("input no delegado=%+v", executor.Input)
	}
	var result orquestamcp.MCPDomainWorkToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != orquestamcp.MCPDomainWorkEstadoOKV0 ||
		result.Job == nil ||
		result.Job.JobRef != "job-domain-work-app-gateway-001" {
		t.Fatalf("result=%+v", result)
	}
}

func TestFunctionContractAPIDelegaEnIndiceReadOnlyV0(t *testing.T) {
	index := &recordingFunctionContractIndexV0{
		ListResult: orquestacore.ListFunctionContractsResultV0{
			Items: []orquestacore.FunctionContractSummaryV0{{
				FunctionContractRef: "contract:function:app-gateway:v0",
				Estado:              orquestacore.FunctionContractEstadoEvidenciaInsuficienteV0,
			}},
			Warnings: []string{"function_contract_payload_no_materializado"},
		},
	}
	handler := NewHTTPHandlerV0(ConfigV0{
		FunctionContracts: index,
		Timeout:           time.Second,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/core/function-contracts/list", strings.NewReader(`{"page":{"limit":1}}`))
	req.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || index.ListCalls != 1 {
		t.Fatalf("status=%d calls=%d body=%s", rec.Code, index.ListCalls, rec.Body.String())
	}
	var result orquestacore.ListFunctionContractsResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].FunctionContractRef != "contract:function:app-gateway:v0" {
		t.Fatalf("result=%+v", result)
	}
}

func TestInProcessTransportV0PreservaRequestYResponse(t *testing.T) {
	transport := InProcessTransportV0{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v0/test" {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("X-Test", r.Header.Get("X-Test"))
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("ok"))
	})}
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest(http.MethodPatch, InternalBaseURLV0+"/api/v0/test", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("X-Test", "preservado")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted || resp.Header.Get("X-Test") != "preservado" {
		t.Fatalf("response=%d headers=%v", resp.StatusCode, resp.Header)
	}
}

type recordingArrancarDirectorExecutorV0 struct {
	Input orquestamcp.MCPArrancarDirectorAppToolInputV0
}

type recordingPreviewDirectorExecutorV0 struct {
	Input orquestamcp.MCPArrancarDirectorAppToolInputV0
}

type recordingFunctionContractIndexV0 struct {
	ListResult orquestacore.ListFunctionContractsResultV0
	ListCalls  int
}

func (index *recordingFunctionContractIndexV0) ListFunctionContractsV0(
	_ context.Context,
	_ orquestacore.ListFunctionContractsRequestV0,
) (orquestacore.ListFunctionContractsResultV0, error) {
	index.ListCalls++
	return index.ListResult, nil
}

func (index *recordingFunctionContractIndexV0) ViewFunctionContractV0(
	_ context.Context,
	_ orquestacore.ViewFunctionContractRequestV0,
) (orquestacore.ViewFunctionContractResultV0, error) {
	return orquestacore.ViewFunctionContractResultV0{}, nil
}

func (executor *recordingArrancarDirectorExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPArrancarDirectorAppToolInputV0,
) (orquestamcp.MCPArrancarDirectorAppToolResultV0, error) {
	executor.Input = input
	return orquestamcp.MCPArrancarDirectorAppToolResultV0{
		Estado:        orquestamcp.MCPArrancarDirectorAppEstadoOKV0,
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		RunRef:        "run-app-gateway-director-001",
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		DirectorTasks: []orquestamcp.MCPDirectorTaskCompactV0{{
			TaskRef:        "task-app-gateway-director-001",
			AgentRequestID: "agent-app-gateway-director-001",
			Capacity:       "high",
		}},
		StartedAgents: []string{"agent-app-gateway-director-001"},
	}, nil
}

func (executor *recordingPreviewDirectorExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPArrancarDirectorAppToolInputV0,
) (orquestamcp.MCPPreviewDirectorAppToolResultV0, error) {
	executor.Input = input
	return orquestamcp.MCPPreviewDirectorAppToolResultV0{
		Estado:    orquestamcp.MCPArrancarDirectorAppEstadoOKV0,
		RequestID: input.RequestID,
		RunRef:    "run-app-gateway-preview-001",
	}, nil
}

type recordingObserveDirectorGoalExecutorV0 struct {
	Input orquestamcp.MCPObserveAppDirectorGoalToolInputV0
}

func (executor *recordingObserveDirectorGoalExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPObserveAppDirectorGoalToolInputV0,
) (orquestamcp.MCPObserveAppDirectorGoalToolResultV0, error) {
	executor.Input = input
	return orquestamcp.MCPObserveAppDirectorGoalToolResultV0{
		Estado:       orquestamcp.MCPObserveAppDirectorGoalEstadoOKV0,
		RequestID:    input.RequestID,
		RunRef:       input.RunRef,
		RunStatus:    "closed",
		GoalRef:      "goal-ref-app-gateway-observed-001",
		GoalStatus:   "complete",
		EvidenceRefs: []string{"evidence-ref-app-gateway-goal-observed-001"},
	}, nil
}

type recordingRequestAppChangeExecutorV0 struct {
	Input orquestamcp.MCPRequestAppChangeToolInputV0
}

func (executor *recordingRequestAppChangeExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPRequestAppChangeToolInputV0,
) (orquestamcp.MCPRequestAppChangeToolResultV0, error) {
	executor.Input = input
	return orquestamcp.MCPRequestAppChangeToolResultV0{
		Estado:              orquestamcp.MCPRequestAppChangeEstadoOKV0,
		RunRef:              input.AppChangeRequest.RunRef,
		AppRef:              input.AppChangeRequest.AppRef,
		ChangeRef:           input.AppChangeRequest.ChangeRef,
		DirectorQuestionRef: "question-ref-app-gateway-change-001",
	}, nil
}

type recordingRunControlExecutorV0 struct {
	Input orquestamcp.MCPRunControlToolInputV0
}

func (executor *recordingRunControlExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPRunControlToolInputV0,
) (orquestamcp.MCPRunControlToolResultV0, error) {
	executor.Input = input
	return orquestamcp.MCPRunControlToolResultV0{
		Estado: orquestamcp.MCPRunControlEstadoOKV0,
		Action: input.Action,
		RunRef: input.RunRef,
		Status: "paused",
	}, nil
}

type recordingRunQueuePriorityExecutorV0 struct {
	Input orquestamcp.MCPRunQueuePriorityToolInputV0
}

func (executor *recordingRunQueuePriorityExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPRunQueuePriorityToolInputV0,
) (orquestamcp.MCPRunQueuePriorityToolResultV0, error) {
	executor.Input = input
	return orquestamcp.MCPRunQueuePriorityToolResultV0{
		Estado:        orquestamcp.MCPRunQueuePriorityEstadoOKV0,
		Action:        input.Action,
		CorrelationID: input.CorrelationID,
		Count:         1,
	}, nil
}

type recordingRunSupervisorExecutorV0 struct {
	Input orquestamcp.MCPRunSupervisorToolInputV0
}

func (executor *recordingRunSupervisorExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
) (orquestamcp.MCPRunSupervisorToolResultV0, error) {
	executor.Input = input
	return orquestamcp.MCPRunSupervisorToolResultV0{
		Estado:     orquestamcp.MCPRunSupervisorEstadoOKV0,
		RunRef:     input.RunRef,
		StopReason: "max_ticks",
		Ticks:      input.MaxTicks,
	}, nil
}

type recordingDomainWorkExecutorV0 struct {
	Input orquestamcp.MCPDomainWorkToolInputV0
}

func (executor *recordingDomainWorkExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPDomainWorkToolInputV0,
) (orquestamcp.MCPDomainWorkToolResultV0, error) {
	executor.Input = input
	return orquestamcp.MCPDomainWorkToolResultV0{
		Estado:        orquestamcp.MCPDomainWorkEstadoOKV0,
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		Action:        input.Action,
		Job: &orquestadomainwork.DomainWorkJobV0{
			SchemaVersion:  orquestadomainwork.DomainWorkJobSchemaV0,
			Status:         orquestadomainwork.DomainWorkStatusAcceptedV0,
			JobRef:         "job-domain-work-app-gateway-001",
			DomainRef:      input.JobRequest.DomainRef,
			WorkKind:       input.JobRequest.WorkKind,
			CorrelationID:  input.CorrelationID,
			IdempotencyKey: input.JobRequest.IdempotencyKey,
		},
	}, nil
}

func appGatewayNuevaAppFormValuesV0() url.Values {
	values := url.Values{}
	values.Set("request_id", "request-ref-app-gateway-001")
	values.Set("locale", "es")
	values.Set("nombre", "Agenda Equipo")
	values.Set("objetivo", "Coordinar tareas y reuniones")
	values.Set("descripcion", "Agenda interna con API y web")
	values.Set("tipo_app", "web")
	values.Add("plataformas", "web")
	values.Set("preferencias_tecnicas.arquitectura", "hexagonal")
	values.Set("preferencias_tecnicas.lenguaje", "go")
	values.Set("i18n.enabled", "true")
	values.Set("i18n.default_locale", "es")
	values.Set("datos.db_required", "true")
	values.Set("datos.necesidad_funcional", "guardar eventos")
	values.Set("deploy.target", "docker")
	values.Set("calidad.pruebas", "contract")
	values.Set("calidad.observabilidad", "true")
	values.Set("agentes.revision_humana", "true")
	values.Set("agentes.autonomia", "supervisada")
	return values
}
