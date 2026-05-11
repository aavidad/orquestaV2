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

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestaweb "orquesta/modulos/orquesta-web"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
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

func TestAppChangeAPIDelegaEnMCPRequestChangeSinCmdDBRuntimeV0(t *testing.T) {
	executor := &recordingRequestAppChangeExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		RequestAppChange: executor,
		Timeout:          time.Second,
	})
	body := strings.NewReader(`{
		"app_change_request":{
			"run_ref":"run-app-gateway-change-001",
			"change_ref":"change-app-gateway-web-001",
			"user_intent":"Cambiar la web para mostrar vista semanal"
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/agenda-equipo/changes", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.Input.AppChangeRequest.AppRef != "agenda-equipo" ||
		executor.Input.AppChangeRequest.ChangeRef != "change-app-gateway-web-001" {
		t.Fatalf("input=%+v", executor.Input.AppChangeRequest)
	}
}

func TestRunControlYRunQueueAPIDeleganEnMCPPortsV0(t *testing.T) {
	control := &recordingRunControlExecutorV0{}
	queue := &recordingRunQueuePriorityExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		RunControl:       control,
		RunQueuePriority: queue,
		Timeout:          time.Second,
	})

	controlRec := httptest.NewRecorder()
	controlReq := httptest.NewRequest(http.MethodPost, "/api/v0/runs/control", strings.NewReader(`{
		"action":"pause",
		"run_ref":"run-app-gateway-control-001"
	}`))
	controlReq.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(controlRec, controlReq)

	if controlRec.Code != http.StatusOK || control.Input.Action != "pause" {
		t.Fatalf("control status=%d input=%+v body=%s", controlRec.Code, control.Input, controlRec.Body.String())
	}

	queueRec := httptest.NewRecorder()
	queueReq := httptest.NewRequest(http.MethodPost, "/api/v0/runs/queue/priority", strings.NewReader(`{
		"action":"set_priority",
		"run_ref":"run-app-gateway-control-001",
		"priority_score":75
	}`))
	queueReq.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(queueRec, queueReq)

	if queueRec.Code != http.StatusOK ||
		queue.Input.Action != "set_priority" ||
		queue.Input.PriorityScore != 75 {
		t.Fatalf("queue status=%d input=%+v body=%s", queueRec.Code, queue.Input, queueRec.Body.String())
	}
}

func TestAppChangePageDelegaEnAPIInternaSinCmdDBRuntimeV0(t *testing.T) {
	executor := &recordingRequestAppChangeExecutorV0{}
	handler := NewHTTPHandlerV0(ConfigV0{
		RequestAppChange: executor,
		Timeout:          time.Second,
	})
	values := url.Values{}
	values.Set("locale", "es")
	values.Set("run_ref", "run-app-gateway-change-page-001")
	values.Set("app_ref", "agenda-equipo")
	values.Set("change_ref", "change-app-gateway-page-001")
	values.Set("user_intent", "Cambiar la web para vista semanal")
	req := httptest.NewRequest(http.MethodPost, "/app-change", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if executor.Input.AppChangeRequest.ChangeRef != "change-app-gateway-page-001" ||
		!strings.Contains(rec.Body.String(), "question-ref-app-gateway-change-001") {
		t.Fatalf("input=%+v body=%s", executor.Input, rec.Body.String())
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
