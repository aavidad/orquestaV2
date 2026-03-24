/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestAPIControlPlaneOrquestacionEndToEnd(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	for _, agente := range []string{"Codex0", "Codex1", "Codex2"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar %s: %v", agente, err)
		}
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesionOrigen, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador", "sesion-codex1"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-codex1",
		ResumenContinuidad: "sesion activa de control plane",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion origen: %v", err)
	}
	runtimeOrigen, err := db.GetRuntimeBySesionID(sesionOrigen.ID)
	if err != nil || runtimeOrigen == nil {
		t.Fatalf("runtime origen: runtime=%+v err=%v", runtimeOrigen, err)
	}
	handleOrigen, err := db.GetRuntimeHandleBySesionID(sesionOrigen.ID)
	if err != nil || handleOrigen == nil {
		t.Fatalf("handle origen: handle=%+v err=%v", handleOrigen, err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	postJSON := func(path string, body any, wantCode int) []byte {
		t.Helper()
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)
		if rec.Code != wantCode {
			t.Fatalf("status inesperado POST %s: %d body=%s", path, rec.Code, rec.Body.String())
		}
		return rec.Body.Bytes()
	}
	getJSON := func(path string, wantCode int) []byte {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(rec, req)
		if rec.Code != wantCode {
			t.Fatalf("status inesperado GET %s: %d body=%s", path, rec.Code, rec.Body.String())
		}
		return rec.Body.Bytes()
	}
	decodeID := func(body []byte) int64 {
		t.Helper()
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode id: %v body=%s", err, string(body))
		}
		id, _ := payload["id"].(float64)
		return int64(id)
	}

	startID := decodeID(postJSON("/api/agente/control", apiAgenteControlRequest{
		Agente:       "Codex2",
		Proyecto:     "orquestador",
		Accion:       "arrancar",
		Conector:     "codex-cli",
		Modelo:       "gpt-5.4",
		Razonamiento: "high",
		Perfil:       "implementacion",
		Motivo:       "arranque e2e",
		Por:          "test",
	}, http.StatusCreated))
	pauseID := decodeID(postJSON("/api/agente/control", apiAgenteControlRequest{
		Agente:   "Codex1",
		Proyecto: "orquestador",
		Accion:   "pausar",
		Motivo:   "pausa e2e",
		Por:      "test",
	}, http.StatusCreated))
	resumeID := decodeID(postJSON("/api/agente/control", apiAgenteControlRequest{
		Agente:   "Codex1",
		Proyecto: "orquestador",
		Accion:   "continuar",
		Motivo:   "resume e2e",
		Por:      "test",
	}, http.StatusCreated))
	stopID := decodeID(postJSON("/api/agente/control", apiAgenteControlRequest{
		Agente:   "Codex1",
		Proyecto: "orquestador",
		Accion:   "detener",
		Motivo:   "stop e2e",
		Por:      "test",
	}, http.StatusCreated))
	sendInstructionID := decodeID(postJSON("/api/runtime-orders", map[string]any{
		"agente":   "Codex1",
		"proyecto": "orquestador",
		"tipo":     "enviar_instruccion",
		"payload":  `{"texto":"revisa logs y responde"}`,
	}, http.StatusCreated))
	nudgeID := decodeID(postJSON("/api/runtime-orders", map[string]any{
		"agente":   "Codex2",
		"proyecto": "orquestador",
		"tipo":     "nudge",
		"payload":  `{"from_agente":"Codex1","to_agente":"Codex2","kind":"handoff_note","texto":"ponte al dia"}`,
	}, http.StatusCreated))
	checkpointOrderID := decodeID(postJSON("/api/runtime-orders", map[string]any{
		"agente":   "Codex1",
		"proyecto": "orquestador",
		"tipo":     "checkpoint",
		"payload":  `{"checkpoint_kind":"handoff_prepare","resumen":"checkpoint e2e codex1"}`,
	}, http.StatusCreated))
	seedCheckpointID := decodeID(postJSON("/api/runtime-checkpoints", map[string]any{
		"agente":          "Codex2",
		"proyecto":        "orquestador",
		"checkpoint_kind": "handoff_prepare",
		"resumen":         "checkpoint semilla codex2",
		"branch":          "feature/codex2",
		"cwd":             filepath.Join(tmp, "orquestador", "checkpoint-codex2"),
		"payload":         `{"paso":"bootstrap"}`,
		"resume_strategy": "resumen_y_payload",
		"source":          "e2e:codex2",
	}, http.StatusCreated))

	handoffResp := postJSON("/api/agente/handoff", apiRuntimeHandoffRequest{
		AgenteOrigen:      "Codex1",
		AgenteDestino:     "Codex2",
		Motivo:            "traspaso e2e",
		Resumen:           "handoff completo hacia Codex2",
		ExternalSessionID: "sess-handoff-codex2",
	}, http.StatusCreated)
	handoffID := decodeID(handoffResp)

	for _, tc := range []struct {
		id   int64
		tipo string
	}{
		{id: pauseID, tipo: agenteControlAccionPause},
		{id: resumeID, tipo: agenteControlAccionResume},
		{id: stopID, tipo: agenteControlAccionStop},
	} {
		order, err := db.GetRuntimeOrder(tc.id)
		if err != nil {
			t.Fatalf("get order %s: %v", tc.tipo, err)
		}
		if order == nil || order.Tipo != tc.tipo {
			t.Fatalf("orden %s inesperada: %+v", tc.tipo, order)
		}
		if order.HandleID == nil || *order.HandleID != handleOrigen.ID {
			t.Fatalf("handle_id inesperado en %s: %+v", tc.tipo, order)
		}
		if order.RuntimeID == nil || *order.RuntimeID != runtimeOrigen.ID {
			t.Fatalf("runtime_id inesperado en %s: %+v", tc.tipo, order)
		}
	}

	if _, err := db.ProcesarRuntimeOrdersBatch(); err != nil {
		t.Fatalf("procesar runtime orders batch: %v", err)
	}

	checkpointOrder, err := db.GetRuntimeOrder(checkpointOrderID)
	if err != nil {
		t.Fatalf("get checkpoint order: %v", err)
	}
	if checkpointOrder == nil || checkpointOrder.Estado != "completada" {
		t.Fatalf("checkpoint order no completada: %+v", checkpointOrder)
	}
	nudgeOrder, err := db.GetRuntimeOrder(nudgeID)
	if err != nil {
		t.Fatalf("get nudge order: %v", err)
	}
	if nudgeOrder == nil || nudgeOrder.Estado != "completada" {
		t.Fatalf("nudge order no completada: %+v", nudgeOrder)
	}

	var listCodex1 struct {
		Orders []*db.RuntimeOrder `json:"orders"`
	}
	if err := json.Unmarshal(getJSON("/api/runtime-orders?agente=Codex1", http.StatusOK), &listCodex1); err != nil {
		t.Fatalf("decode orders codex1: %v", err)
	}
	tiposCodex1 := make(map[string]bool, len(listCodex1.Orders))
	for _, order := range listCodex1.Orders {
		tiposCodex1[order.Tipo] = true
	}
	for _, tipo := range []string{"pause", "resume", "stop", "enviar_instruccion", "checkpoint"} {
		if !tiposCodex1[tipo] {
			t.Fatalf("faltaba tipo %s en orders codex1: %+v", tipo, listCodex1.Orders)
		}
	}
	sendInstructionOrder, err := db.GetRuntimeOrder(sendInstructionID)
	if err != nil {
		t.Fatalf("get send_instruction order: %v", err)
	}
	if sendInstructionOrder == nil || sendInstructionOrder.Tipo != "enviar_instruccion" || sendInstructionOrder.Estado != "pendiente" {
		t.Fatalf("send_instruction inesperada: %+v", sendInstructionOrder)
	}

	var latestCheckpoint struct {
		Checkpoint *db.RuntimeCheckpoint `json:"checkpoint"`
	}
	if err := json.Unmarshal(getJSON("/api/runtime-checkpoints/latest?agente=Codex1&proyecto=orquestador", http.StatusOK), &latestCheckpoint); err != nil {
		t.Fatalf("decode latest checkpoint: %v", err)
	}
	if latestCheckpoint.Checkpoint == nil || latestCheckpoint.Checkpoint.Resumen != "checkpoint e2e codex1" {
		t.Fatalf("latest checkpoint inesperado: %+v", latestCheckpoint.Checkpoint)
	}

	var pendingMailbox struct {
		Mailbox []*db.RuntimeMailboxMessage `json:"mailbox"`
	}
	if err := json.Unmarshal(getJSON("/api/runtime-mailbox?to_agente=Codex2&proyecto=orquestador&estado=pendiente", http.StatusOK), &pendingMailbox); err != nil {
		t.Fatalf("decode mailbox pendiente: %v", err)
	}
	if len(pendingMailbox.Mailbox) != 1 || pendingMailbox.Mailbox[0].RuntimeOrderID == nil || *pendingMailbox.Mailbox[0].RuntimeOrderID != nudgeID {
		t.Fatalf("mailbox pendiente inesperado: %+v", pendingMailbox.Mailbox)
	}

	var prepHandoff agentePrepararOutput
	if err := json.Unmarshal(getJSON("/api/agente/preparar?agente=Codex2&proyecto=orquestador", http.StatusOK), &prepHandoff); err != nil {
		t.Fatalf("decode preparar handoff: %v", err)
	}
	if prepHandoff.Bootstrap == nil || prepHandoff.Bootstrap.Order == nil || prepHandoff.Bootstrap.Order.ID != handoffID {
		t.Fatalf("bootstrap handoff inesperado: %+v", prepHandoff.Bootstrap)
	}
	if len(prepHandoff.Bootstrap.Mailbox) != 1 || prepHandoff.Bootstrap.Mailbox[0].Kind != "handoff_note" {
		t.Fatalf("mailbox bootstrap inesperado: %+v", prepHandoff.Bootstrap.Mailbox)
	}
	if prepHandoff.Bootstrap.Checkpoint == nil || prepHandoff.Bootstrap.Checkpoint.ID != seedCheckpointID {
		t.Fatalf("checkpoint bootstrap inesperado: %+v", prepHandoff.Bootstrap.Checkpoint)
	}
	if prepHandoff.Plan == nil || prepHandoff.Plan.Modo != "resume" {
		t.Fatalf("plan bootstrap handoff inesperado: %+v", prepHandoff.Plan)
	}
	if prepHandoff.Plan.WorkingDir != filepath.Join(tmp, "orquestador", "checkpoint-codex2") {
		t.Fatalf("working dir bootstrap inesperado: %s", prepHandoff.Plan.WorkingDir)
	}
	if !strings.Contains(prepHandoff.Plan.ContinuityPrompt, "handoff completo hacia Codex2") {
		t.Fatalf("continuity prompt sin handoff: %s", prepHandoff.Plan.ContinuityPrompt)
	}
	if !strings.Contains(prepHandoff.Plan.ContinuityPrompt, `"mailbox"`) || !strings.Contains(prepHandoff.Plan.ContinuityPrompt, `"checkpoint"`) {
		t.Fatalf("continuity prompt sin mailbox/checkpoint: %s", prepHandoff.Plan.ContinuityPrompt)
	}

	handoffOrder, err := db.GetRuntimeOrder(handoffID)
	if err != nil {
		t.Fatalf("get handoff order: %v", err)
	}
	if handoffOrder == nil || handoffOrder.Estado != "completada" {
		t.Fatalf("handoff order no completada: %+v", handoffOrder)
	}
	startOrder, err := db.GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order tras primer prepare: %v", err)
	}
	if startOrder == nil || startOrder.Estado != "pendiente" {
		t.Fatalf("start order deberia seguir pendiente tras el primer prepare: %+v", startOrder)
	}

	var consumedMailbox struct {
		Mailbox []*db.RuntimeMailboxMessage `json:"mailbox"`
	}
	if err := json.Unmarshal(getJSON("/api/runtime-mailbox?to_agente=Codex2&proyecto=orquestador&estado=consumido", http.StatusOK), &consumedMailbox); err != nil {
		t.Fatalf("decode mailbox consumido: %v", err)
	}
	if len(consumedMailbox.Mailbox) != 1 {
		t.Fatalf("mailbox consumido inesperado: %+v", consumedMailbox.Mailbox)
	}

	var prepStart agentePrepararOutput
	if err := json.Unmarshal(getJSON("/api/agente/preparar?agente=Codex2&proyecto=orquestador", http.StatusOK), &prepStart); err != nil {
		t.Fatalf("decode preparar start: %v", err)
	}
	if prepStart.Bootstrap == nil || prepStart.Bootstrap.Order == nil || prepStart.Bootstrap.Order.ID != startID || prepStart.Bootstrap.Order.Tipo != "start" {
		t.Fatalf("bootstrap start inesperado: %+v", prepStart.Bootstrap)
	}
	startOrder, err = db.GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order final: %v", err)
	}
	if startOrder == nil || startOrder.Estado != "completada" {
		t.Fatalf("start order no completada: %+v", startOrder)
	}
}
