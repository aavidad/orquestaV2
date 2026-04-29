package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestAPIAgentesConvergeConStatusYOperationalTrasRefreshDePanel(t *testing.T) {
	prepararDBTemporalCmd(t)
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex3: %v", err)
	}

	now := time.Now().UTC()
	resetAt := now.Add(30 * time.Minute)
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes: []*db.Agente{{
			Nombre:      "Codex3",
			EstadoCuota: "enfriamiento",
			ReanimarAt:  &resetAt,
		}},
		AgentesQuotaBlocked: []*db.Agente{{
			Nombre:      "Codex3",
			EstadoCuota: "enfriamiento",
			ReanimarAt:  &resetAt,
		}},
		WorkersConectados:   0,
		WorkersTrabajando:   0,
		SupervisoresActivos: 0,
		Generado:            now.Add(-15 * time.Second).Format(time.RFC3339),
	}, now, time.Minute)
	storeAgentPanelSnapshot([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "Codex3", Activo: true, EstadoCuota: "activo"},
		EstadoOperativo: "trabajando",
		WorkerAlive:     true,
		WorkerState:     "running",
		OpenTasks:       1,
	}}, now)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recAgentes := httptest.NewRecorder()
	reqAgentes := httptest.NewRequest(http.MethodGet, "/api/agentes", nil)
	mux.ServeHTTP(recAgentes, reqAgentes)
	if recAgentes.Code != http.StatusOK {
		t.Fatalf("status agentes inesperado: %d body=%s", recAgentes.Code, recAgentes.Body.String())
	}
	var agentesResp struct {
		Agentes []*db.Agente `json:"agentes"`
	}
	if err := json.Unmarshal(recAgentes.Body.Bytes(), &agentesResp); err != nil {
		t.Fatalf("decode agentes: %v", err)
	}
	if len(agentesResp.Agentes) == 0 {
		t.Fatalf("agentes inesperados: %+v", agentesResp.Agentes)
	}
	var codex3 *db.Agente
	for _, agente := range agentesResp.Agentes {
		if agente != nil && agente.Nombre == "Codex3" {
			codex3 = agente
			break
		}
	}
	if codex3 == nil {
		t.Fatalf("Codex3 no expuesto via /api/agentes: %+v", agentesResp.Agentes)
	}
	if !codex3.Activo || codex3.EstadoCuota != "activo" {
		t.Fatalf("/api/agentes deberia converger con panel fresco: %+v", codex3)
	}

	recStatus := httptest.NewRecorder()
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	mux.ServeHTTP(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("status global inesperado: %d body=%s", recStatus.Code, recStatus.Body.String())
	}
	var statusResp apiStatusResponse
	if err := json.Unmarshal(recStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if len(statusResp.AgentesQuotaBlocked) != 0 || len(statusResp.AgentesTrabajando) != 1 {
		t.Fatalf("/api/status no convergio con panel fresco: %+v", statusResp)
	}

	recOperational := httptest.NewRecorder()
	reqOperational := httptest.NewRequest(http.MethodGet, "/api/server/operational", nil)
	mux.ServeHTTP(recOperational, reqOperational)
	if recOperational.Code != http.StatusOK {
		t.Fatalf("status operational inesperado: %d body=%s", recOperational.Code, recOperational.Body.String())
	}
	var operationalResp serverOperationalInfo
	if err := json.Unmarshal(recOperational.Body.Bytes(), &operationalResp); err != nil {
		t.Fatalf("decode operational: %v", err)
	}
	if !operationalResp.Operational || operationalResp.State != "ready" || operationalResp.WorkingWorkers != 1 {
		t.Fatalf("/api/server/operational no convergio con panel fresco: %+v", operationalResp)
	}
}

func TestAPIAgentesPanelCanonicalExponeDTOCanonico(t *testing.T) {
	prepararDBTemporalCmd(t)
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()

	now := time.Now().UTC()
	storeAgentPanelSnapshot([]agentesapp.Row{{
		Agente:           &db.Agente{Nombre: "Codex7", Rol: "programador", Activo: true, Habilitado: true},
		EstadoOperativo:  "trabajando",
		DetalleOperativo: "worker ready",
		WorkerAlive:      true,
		WorkerState:      "running",
		MailboxPending:   2,
		OpenTasks:        3,
		BlockedTasks:     0,
		CurrentTask:      &agentesapp.TaskFocus{TaskID: 40, Title: "runtime mailbox", State: db.TareaEnProgreso},
		DominantOrder:    &agentesapp.OrderFocus{OrderID: 91, Type: "send_instruction", State: "pendiente"},
	}}, now)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agentes?vista=panel&schema=canonical", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiAgentesPanelCanonicalResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode canonical panel: %v", err)
	}
	if len(resp.Agents) != 1 || resp.Agents[0] == nil || resp.Agents[0].Entity == nil {
		t.Fatalf("agents inesperados: %+v", resp.Agents)
	}
	if resp.Agents[0].Entity.Name != "Codex7" || resp.Agents[0].Entity.OperationalState != "trabajando" {
		t.Fatalf("entity inesperada: %+v", resp.Agents[0].Entity)
	}
	if resp.Agents[0].OpenTasks != 3 || resp.Agents[0].MailboxPendingVisible != 2 {
		t.Fatalf("panel canonical inesperado: %+v", resp.Agents[0])
	}
	if resp.Agents[0].Entity.MultitaskDebt != 2 || resp.Agents[0].Entity.WorkerGapCount != 0 {
		t.Fatalf("panel canonical sin deuda multitarea esperada: %+v", resp.Agents[0].Entity)
	}
	if !strings.Contains(resp.Agents[0].Entity.TaskWorkerHint, "deuda multitarea=2") {
		t.Fatalf("panel canonical sin explicacion multitarea esperada: %+v", resp.Agents[0].Entity)
	}
}

func TestAPIAgenteOverviewExponeAgentCanonicoSinRomperDetail(t *testing.T) {
	prepararDBTemporalCmd(t)

	prevBuilder := apiAgentDetailBuilder
	defer func() { apiAgentDetailBuilder = prevBuilder }()

	apiAgentDetailBuilder = func(nombre string, compact bool) (*agentesapp.Detail, error) {
		return &agentesapp.Detail{
			Row: agentesapp.Row{
				Agente:                   &db.Agente{Nombre: nombre, Rol: "programador", Activo: true, Habilitado: true},
				MailboxPending:           3,
				MailboxActionablePending: 1,
				MailboxContinuityPending: 1,
				MailboxTotal:             7,
				OpenTasks:                2,
				OrdersOpen:               1,
				Checkpoints:              4,
			},
			Entity:                &agentesapp.AgentEntity{Name: nombre, Role: "programador", ActiveNow: true, OperationalState: "trabajando"},
			MailboxPendingVisible: 2,
			MailboxTotalCount:     7,
		}, nil
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agentes/Codex7/overview?compact=true", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiAgenteOverviewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode overview: %v", err)
	}
	if resp.Detail == nil || resp.Detail.Entity == nil || resp.Detail.Entity.Name != "Codex7" {
		t.Fatalf("detail legacy inesperado: %+v", resp.Detail)
	}
	if resp.Agent == nil || resp.Agent.Entity == nil || resp.Agent.Entity.Name != "Codex7" {
		t.Fatalf("agent canónico inesperado: %+v", resp.Agent)
	}
	if resp.Agent.OpenTasks != 2 || resp.Agent.MailboxPendingVisible != 2 || resp.Agent.MailboxTotal != 7 {
		t.Fatalf("agent canónico sin contadores esperados: %+v", resp.Agent)
	}
}

func TestAPIAgenteOverviewCanonicoExplicaGapConStatus(t *testing.T) {
	prepararDBTemporalCmd(t)

	prevBuilder := apiAgentDetailBuilder
	defer func() { apiAgentDetailBuilder = prevBuilder }()

	apiAgentDetailBuilder = func(nombre string, compact bool) (*agentesapp.Detail, error) {
		return &agentesapp.Detail{
			Row: agentesapp.Row{
				Agente:    &db.Agente{Nombre: nombre, Rol: "programador", Activo: false, Habilitado: true},
				OpenTasks: 2,
			},
			Entity:                &agentesapp.AgentEntity{Name: nombre, Role: "programador", ActiveNow: false, OperationalState: "sin_tarea"},
			MailboxPendingVisible: 0,
			MailboxTotalCount:     0,
		}, nil
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agentes/CodexGap/overview?compact=true", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiAgenteOverviewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode overview gap: %v", err)
	}
	if resp.Agent == nil || resp.Agent.Entity == nil {
		t.Fatalf("agent canónico inesperado: %+v", resp.Agent)
	}
	if resp.Agent.Entity.WorkerGapCount != 2 {
		t.Fatalf("worker gap inesperado: %+v", resp.Agent.Entity)
	}
	if !strings.Contains(resp.Agent.Entity.TaskWorkerHint, "/api/status.agentesActivos") {
		t.Fatalf("falta explicacion de discrepancia con status: %+v", resp.Agent.Entity)
	}
}

func TestAPIAgentesPanelCanonicalDerivaActiveNowDesdeEstadoOperativoVivo(t *testing.T) {
	prepararDBTemporalCmd(t)
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()

	now := time.Now().UTC()
	storeAgentPanelSnapshot([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "CodexLive", Rol: "programador", Activo: false, Habilitado: true, EstadoCuota: "activo"},
		EstadoOperativo: "trabajando",
		WorkerAlive:     true,
		WorkerState:     "ready",
		WorkerHeartbeat: &now,
		OpenTasks:       1,
		CurrentTask:     &agentesapp.TaskFocus{TaskID: 36, Title: "slice activo", State: db.TareaEnProgreso},
	}}, now)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agentes?vista=panel&schema=canonical", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiAgentesPanelCanonicalResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode canonical panel: %v", err)
	}
	if len(resp.Agents) != 1 || resp.Agents[0] == nil || resp.Agents[0].Entity == nil {
		t.Fatalf("agents inesperados: %+v", resp.Agents)
	}
	if !resp.Agents[0].Entity.ActiveNow {
		t.Fatalf("active_now deberia derivarse del estado operativo vivo: %+v", resp.Agents[0].Entity)
	}
}

func TestAPIAgentesPanelCanonicalNoPromueveSesionSolaComoActiveNow(t *testing.T) {
	prepararDBTemporalCmd(t)
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()
	prevBuilder := apiAgentPanelRowsBuilder
	apiAgentPanelRowsBuilder = nil
	defer func() { apiAgentPanelRowsBuilder = prevBuilder }()

	now := time.Now().UTC()
	storeAgentPanelSnapshot([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "CodexStale", Rol: "programador", Activo: false, Habilitado: true, EstadoCuota: "activo"},
		EstadoOperativo: "sin_tarea",
		Sesion:          &db.Sesion{ID: 77, Agente: "CodexStale", Estado: "activa", Activa: true},
		OpenTasks:       2,
	}}, now)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agentes?vista=panel&schema=canonical", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiAgentesPanelCanonicalResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode canonical panel: %v", err)
	}
	if len(resp.Agents) != 1 || resp.Agents[0] == nil || resp.Agents[0].Entity == nil {
		t.Fatalf("agents inesperados: %+v", resp.Agents)
	}
	if resp.Agents[0].Entity.ActiveNow {
		t.Fatalf("una sesion sola sin worker/runtime/handle vivo no deberia marcar active_now: %+v", resp.Agents[0].Entity)
	}
	if resp.Agents[0].Entity.WorkerGapCount != 2 {
		t.Fatalf("deberia exponer gap tareas-vs-worker por agente: %+v", resp.Agents[0].Entity)
	}
	if !strings.Contains(resp.Agents[0].Entity.TaskWorkerHint, "/api/status.agentesActivos") {
		t.Fatalf("deberia explicar la discrepancia con status: %+v", resp.Agents[0].Entity)
	}
}
