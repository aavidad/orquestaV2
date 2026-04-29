package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		Agente:          &db.Agente{Nombre: "Codex7", Rol: "programador", Activo: true, Habilitado: true},
		EstadoOperativo: "trabajando",
		DetalleOperativo:"worker ready",
		WorkerAlive:     true,
		WorkerState:     "running",
		MailboxPending:  2,
		OpenTasks:       1,
		BlockedTasks:    0,
		CurrentTask:     &agentesapp.TaskFocus{TaskID: 40, Title: "runtime mailbox", State: db.TareaEnProgreso},
		DominantOrder:   &agentesapp.OrderFocus{OrderID: 91, Type: "send_instruction", State: "pendiente"},
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
	if resp.Agents[0].OpenTasks != 1 || resp.Agents[0].MailboxPendingVisible != 2 {
		t.Fatalf("panel canonical inesperado: %+v", resp.Agents[0])
	}
}
