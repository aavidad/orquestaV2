package orquestaappgateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaweb "orquesta/modulos/orquesta-web"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestAppGatewayDirectorAPIYStatsWebCompartenRunStoreV0(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	handler := NewHTTPHandlerV0(ConfigV0{
		ArrancarDirector: orquestamcp.NewMCPArrancarDirectorAppToolExecutorV0(
			orquestaappdirectorservice.StartAppDirectorPortsV0{
				RunStore:     store,
				EventSink:    sink,
				OutboxLedger: ledger,
				Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
					appGatewayCapacityDispatcherForTestV0(store, sink, ledger),
					appGatewayAgentLauncherDispatcherForTestV0(store, sink, ledger),
				},
			},
		),
		DirectorStats: orquestamcp.MCPDirectorStatsToolExecutorV0{
			RunStore: store,
		},
		Timeout: time.Second,
	})

	directorResult := postAppGatewayDirectorAPIV0(t, handler)
	if directorResult.RunRef == "" || len(directorResult.StartedAgents) != 1 {
		t.Fatalf("directorResult=%+v", directorResult)
	}

	rec := httptest.NewRecorder()
	statsURL := "/director-stats?run_ref=" + url.QueryEscape(directorResult.RunRef)
	req := httptest.NewRequest(http.MethodGet, statsURL, nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("stats status=%d body=%s", rec.Code, rec.Body.String())
	}
	var page orquestaweb.WebDirectorStatsPageV0
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode stats page: %v", err)
	}
	if page.ViewModel.RunRef != directorResult.RunRef ||
		page.ViewModel.Counts.Brainstorms != 1 ||
		page.ViewModel.Counts.TasksTotal != 0 ||
		page.ViewModel.Counts.AgentsStarted != 1 ||
		len(page.ViewModel.Agents) != 1 {
		t.Fatalf("page=%+v director=%+v", page, directorResult)
	}
}

func postAppGatewayDirectorAPIV0(
	t *testing.T,
	handler http.Handler,
) orquestamcp.MCPArrancarDirectorAppToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orquestamcp.MCPArrancarDirectorAppToolInputV0{
		RequestID:     "request-ref-app-gateway-real-flow-001",
		CorrelationID: "corr-app-gateway-real-flow-001",
		AppSpecRequest: orquestafactory.AppSpecRequestV0{
			SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
			RequestID:     "request-ref-app-gateway-real-flow-001",
			Source:        "orquesta-app-gateway-test",
			Locale:        "es-ES",
			Nombre:        "Agenda API Web",
			Objetivo:      "Gestionar agenda con API REST y web",
			TipoApp:       "web",
			Plataformas:   []string{"web"},
			PreferenciasTecnicas: orquestafactory.PreferenciasTecnicasV0{
				Lenguaje:     "go",
				Arquitectura: "hexagonal",
			},
			Datos: orquestafactory.DatosRequestV0{
				DBRequired:         true,
				NecesidadFuncional: "guardar eventos",
			},
			Deploy: orquestafactory.DeployRequestV0{Target: "contenedor"},
			Agentes: orquestafactory.AgentesRequestV0{
				Autonomia: "media",
			},
		},
	}); err != nil {
		t.Fatalf("encode director input: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/apps/director", body)
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("director status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPArrancarDirectorAppToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode director result: %v", err)
	}
	if result.Estado != orquestamcp.MCPArrancarDirectorAppEstadoOKV0 {
		t.Fatalf("director result=%+v", result)
	}
	return result
}

func appGatewayCapacityDispatcherForTestV0(
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
			OccurredAt:      "2026-05-10T05:00:00Z",
			CorrelationID:   "corr-app-gateway-capacity",
			RequestedBy:     "orquesta-app-gateway-test",
		},
		Acker: ledger,
	}
}

func appGatewayAgentLauncherDispatcherForTestV0(
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
			OccurredAt:    "2026-05-10T05:01:00Z",
			CorrelationID: "corr-app-gateway-agent",
			RequestedBy:   "orquesta-app-gateway-test",
		},
		Acker: ledger,
	}
}
