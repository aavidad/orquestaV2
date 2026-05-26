package orquestaweb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestWebDirectorStatsViewModelV0ProyectaStalledInformativoSinAtencion(t *testing.T) {
	stats := webDirectorStatsFixtureV0()

	vm := NewWebDirectorStatsViewModelV0(stats)

	if vm.Estado != WebDirectorStatsEstadoOKV0 ||
		vm.Progress.StalledAgents != 1 ||
		vm.Counts.AgentsNeedAttention != 0 ||
		vm.Counts.Brainstorms != 1 ||
		vm.Resumen.UsageTotalTokens != 1750 ||
		len(vm.Tasks) != 2 ||
		vm.Agents[0].ProgressStatus == "" {
		t.Fatalf("vm=%+v", vm)
	}
}

func TestDirectorStatsWebEndpointV0GETConsultaCliente(t *testing.T) {
	client := &fakeDirectorStatsClientV0{VM: NewWebDirectorStatsViewModelV0(webDirectorStatsFixtureV0())}
	endpoint := NewDirectorStatsWebEndpointV0(client)
	req := httptest.NewRequest(
		http.MethodGet,
		"/director-stats?run_ref=run-web-director-stats-001&include_agent_progress=true",
		nil,
	)
	rec := httptest.NewRecorder()

	endpoint.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || client.Query.RunRef != "run-web-director-stats-001" ||
		!client.Query.IncludeAgentProgress {
		t.Fatalf("code=%d query=%+v body=%s", rec.Code, client.Query, rec.Body.String())
	}
	var page WebDirectorStatsPageV0
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if page.ViewModel.Progress.StalledAgents != 1 {
		t.Fatalf("page=%+v", page)
	}
}

func TestDirectorStatsWebEndpointV0GETPreparaRefreshSemitiempoReal(t *testing.T) {
	client := &fakeDirectorStatsClientV0{VM: NewWebDirectorStatsViewModelV0(webDirectorStatsFixtureV0())}
	endpoint := NewDirectorStatsWebEndpointV0(client)
	req := httptest.NewRequest(
		http.MethodGet,
		"/director-stats?run_ref=run-web-director-stats-001&locale=es",
		nil,
	)
	rec := httptest.NewRecorder()

	endpoint.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	if !client.Query.IncludeAgentProgress ||
		!client.Query.IncludeProcessRefs ||
		client.Query.IncludeAgentUsage {
		t.Fatalf("query debe pedir uso solo con include_agent_usage explicito: %+v", client.Query)
	}
	var page WebDirectorStatsPageV0
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if !page.Refresh.Enabled ||
		page.Refresh.Method != http.MethodGet ||
		page.Refresh.IntervalMillis != WebDirectorStatsRefreshIntervalMsV0 ||
		!strings.Contains(page.Refresh.Href, "include_process_refs=true") ||
		!strings.Contains(page.Refresh.Href, "include_agent_progress=true") ||
		strings.Contains(page.Refresh.Href, "include_agent_usage=true") ||
		!strings.Contains(page.Refresh.Href, "run_ref=run-web-director-stats-001") {
		t.Fatalf("refresh=%+v", page.Refresh)
	}
	if page.ViewModel.Textos.Refresh != "Actualizar progreso de agentes" {
		t.Fatalf("texto refresh=%q", page.ViewModel.Textos.Refresh)
	}
}

func TestDirectorStatsWebEndpointV0GETRespetaUsoOptIn(t *testing.T) {
	client := &fakeDirectorStatsClientV0{VM: NewWebDirectorStatsViewModelV0(webDirectorStatsFixtureV0())}
	endpoint := NewDirectorStatsWebEndpointV0(client)
	req := httptest.NewRequest(
		http.MethodGet,
		"/director-stats?run_ref=run-web-director-stats-001&include_agent_usage=true",
		nil,
	)
	rec := httptest.NewRecorder()

	endpoint.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !client.Query.IncludeAgentUsage {
		t.Fatalf("code=%d query=%+v body=%s", rec.Code, client.Query, rec.Body.String())
	}
	var page WebDirectorStatsPageV0
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if !strings.Contains(page.Refresh.Href, "include_agent_usage=true") {
		t.Fatalf("refresh debe preservar uso opt-in: %+v", page.Refresh)
	}
}

func TestRESTDirectorStatsClientV0DecodificaStats(t *testing.T) {
	stats := webDirectorStatsFixtureV0()
	server := newWebHTTPTestServerV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"stats": stats})
	}))
	defer server.Close()
	client := NewRESTDirectorStatsClientV0(server.URL, 0)

	vm, err := client.ConsultarDirectorStats(context.Background(), WebDirectorStatsQueryV0{
		RunRef:               stats.RunRef,
		IncludeAgentProgress: true,
	})
	if err != nil {
		t.Fatalf("ConsultarDirectorStats: %v", err)
	}
	if vm.RunRef != stats.RunRef || vm.Progress.StalledAgents != 1 {
		t.Fatalf("vm=%+v", vm)
	}
}

func TestDirectorStatsWebEndpointV0POSTFormSinRunRefDevuelveErrorPublico(t *testing.T) {
	endpoint := NewDirectorStatsWebEndpointV0(&fakeDirectorStatsClientV0{})
	req := httptest.NewRequest(http.MethodPost, "/director-stats", strings.NewReader("run_ref="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	endpoint.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
}

type fakeDirectorStatsClientV0 struct {
	Query WebDirectorStatsQueryV0
	VM    WebDirectorStatsViewModelV0
	Err   error
}

func (client *fakeDirectorStatsClientV0) ConsultarDirectorStats(
	_ context.Context,
	query WebDirectorStatsQueryV0,
) (WebDirectorStatsViewModelV0, error) {
	client.Query = query
	if client.Err != nil {
		return WebDirectorStatsViewModelV0{}, client.Err
	}
	return client.VM, nil
}

func webDirectorStatsFixtureV0() WebDirectorRunStatsContractV0 {
	run := orquestacoreworkflow.OrchestrationRunV0{
		RunID:         "run-web-director-stats-001",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:         []string{"task-web-stats-001", "task-web-stats-002"},
		Brainstorms:   []string{"brainstorm-web-stats-001"},
		StartedAgents: []string{"agent-web-stats-001", "agent-web-stats-002"},
		Agents:        []string{"agent-web-stats-001", "agent-web-stats-002"},
	}
	stats := orquestacionnucleoapp.BuildDirectorRunStatsWithObservationsV0(
		run,
		[]orquestacionnucleoapp.AgentProgressObservationV0{
			webDirectorProgressObservationV0(run.RunID, "agent-web-stats-001", "task-web-stats-001", orquestaruntime.AgentProgressingV0),
			webDirectorProgressObservationV0(run.RunID, "agent-web-stats-002", "task-web-stats-002", orquestaruntime.AgentStalledV0),
		},
		nil,
	)
	raw, err := json.Marshal(stats)
	if err != nil {
		panic(err)
	}
	var out WebDirectorRunStatsContractV0
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	out.UsageSummary = &WebDirectorRunUsageSummaryV0{
		AgentsObserved: 2,
		QuotaStatus:    "limited",
		TotalTokens:    1750,
	}
	return out
}

func webDirectorProgressObservationV0(
	runRef string,
	agentRef string,
	taskRef string,
	status orquestaruntime.AgentProgressStatusV0,
) orquestacionnucleoapp.AgentProgressObservationV0 {
	return orquestacionnucleoapp.AgentProgressObservationV0{
		TaskRef: taskRef,
		Report: orquestaruntime.AgentProgressReportV0{
			ReportID:            "agent-progress-report-ref-" + agentRef,
			RunID:               runRef,
			AgentRequestID:      agentRef,
			Status:              status,
			NoProgressTicks:     1,
			RepeatedActionCount: 0,
			Summary:             "Progreso compacto para web.",
			EvidenceRefs:        []string{"evidence-ref-" + agentRef},
		},
	}
}
