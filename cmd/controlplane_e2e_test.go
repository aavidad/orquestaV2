/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"orquesta/db"
	"orquesta/internal/controlruntime"
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

	if _, err := db.ProcesarRuntimeOrdersBasicasBatch(); err != nil {
		t.Fatalf("procesar runtime orders basicas batch: %v", err)
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
	if prepHandoff.Plan.WorkingDir != filepath.Join(tmp, "orquestador") {
		t.Fatalf("working dir bootstrap inesperado: %s", prepHandoff.Plan.WorkingDir)
	}
	if !strings.Contains(prepHandoff.Plan.ContinuityPrompt, "handoff completo hacia Codex2") {
		t.Fatalf("continuity prompt sin handoff: %s", prepHandoff.Plan.ContinuityPrompt)
	}
	if !strings.Contains(prepHandoff.Plan.ContinuityPrompt, "mailbox=1") || !strings.Contains(prepHandoff.Plan.ContinuityPrompt, "checkpoint#") {
		t.Fatalf("continuity prompt sin mailbox/checkpoint: %s", prepHandoff.Plan.ContinuityPrompt)
	}

	handoffOrder, err := db.GetRuntimeOrder(handoffID)
	if err != nil {
		t.Fatalf("get handoff order: %v", err)
	}
	if handoffOrder == nil || handoffOrder.Estado != "pendiente" {
		t.Fatalf("handoff order no deberia consumirse en preparar: %+v", handoffOrder)
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
	if len(consumedMailbox.Mailbox) != 0 {
		t.Fatalf("mailbox consumido inesperado: %+v", consumedMailbox.Mailbox)
	}

	var prepStart agentePrepararOutput
	if err := json.Unmarshal(getJSON("/api/agente/preparar?agente=Codex2&proyecto=orquestador", http.StatusOK), &prepStart); err != nil {
		t.Fatalf("decode preparar start: %v", err)
	}
	if prepStart.Bootstrap == nil || prepStart.Bootstrap.Order == nil || prepStart.Bootstrap.Order.ID != handoffID || prepStart.Bootstrap.Order.Tipo != "handoff" {
		t.Fatalf("bootstrap handoff deberia seguir pendiente hasta arranque real: %+v", prepStart.Bootstrap)
	}
	startOrder, err = db.GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order final: %v", err)
	}
	if startOrder == nil || startOrder.Estado != "pendiente" {
		t.Fatalf("start order no deberia mutar en preparar: %+v", startOrder)
	}
}

func TestAPIControlPlaneArranqueRealConBootstrapMultilinea(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	argLog := filepath.Join(tmp, "launch-argv.log")
	promptLog := filepath.Join(tmp, "launch-prompt.log")
	launcher := filepath.Join(tmp, "fake-launcher.sh")
	script := "#!/usr/bin/env bash\nset -eu\nprintf '%s\\n' \"$@\" > " + strconv.Quote(argLog) + "\nprintf '%s' \"${!#}\" > " + strconv.Quote(promptLog) + "\nexec sleep 30\n"
	if err := os.WriteFile(launcher, []byte(script), 0o755); err != nil {
		t.Fatalf("write launcher: %v", err)
	}

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
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
	if _, err := db.UpsertConector(&db.Conector{
		Slug:         "codex-launcher",
		Nombre:       "Codex Launcher",
		Transporte:   "cli",
		Comando:      launcher,
		MetadataJSON: `{"launch_prompt_positional":true}`,
		Activo:       true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Implementar bootstrap real",
		Modulo:     "orquestador",
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
		ProyectoID: &proyectoID,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET estado='asignada' WHERE id=?`, tareaID); err != nil {
		t.Fatalf("marcar tarea asignada: %v", err)
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
		Agente:       "Codex1",
		Proyecto:     "orquestador",
		Accion:       "arrancar",
		Conector:     "codex-launcher",
		Modelo:       "gpt-5.4",
		Razonamiento: "high",
		Perfil:       "implementacion",
		Motivo:       "arranque real e2e",
		Por:          "test",
	}, http.StatusCreated))

	ctx, cancel := context.WithCancel(context.Background())
	runner := newControlPlaneRunner(nil, false)
	runner.NotificationFeed = nil
	runner.InitNotifications = nil
	runner.Notifier = nil
	runner.ReanimacionCada = time.Hour
	runner.SaludCada = time.Hour
	runner.PlanificacionCada = time.Hour
	runner.ControlPlaneCada = 20 * time.Millisecond
	t.Cleanup(func() {
		cancel()
		runner.Wait()
	})
	runner.Start(ctx)

	deadline := time.Now().Add(3 * time.Second)
	var (
		argvData   []byte
		promptData []byte
		startOrder *db.RuntimeOrder
	)
	for {
		promptData, err = os.ReadFile(promptLog)
		if err == nil && strings.Contains(string(promptData), "Bootstrap de Orquesta para Codex1.") {
			startOrder, _ = db.GetRuntimeOrder(startID)
			if startOrder != nil && startOrder.Estado == "completada" {
				break
			}
		}
		if time.Now().After(deadline) {
			if err != nil {
				t.Fatalf("leer prompt log: %v", err)
			}
			argvData, _ = os.ReadFile(argLog)
			startOrder, _ = db.GetRuntimeOrder(startID)
			t.Fatalf("el launcher no recibió el bootstrap esperado. order=%+v prompt=%q argv=%s", startOrder, string(promptData), string(argvData))
		}
		time.Sleep(10 * time.Millisecond)
	}

	sesion, err := db.GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion activa: %+v err=%v", sesion, err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil || runtime.PID == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	t.Cleanup(func() {
		_, _, _ = controlruntime.DetenerProceso(controlruntime.ObjetivoProceso{PID: runtime.PID})
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if err := syscall.Kill(int(*runtime.PID), 0); err != nil {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	})

	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if strings.Contains(handle.MetadataJSON, `"bootstrap_prompt"`) || !strings.Contains(handle.MetadataJSON, `"launch_prompt_embedded":true`) {
		t.Fatalf("metadata de handle no deberia conservar bootstrap_prompt crudo: %s", handle.MetadataJSON)
	}
	if startOrder == nil || startOrder.Estado != "completada" {
		t.Fatalf("start order no completada: %+v", startOrder)
	}

	argvData, _ = os.ReadFile(argLog)
	prompt := string(promptData)
	expectedPrefix := strings.Join([]string{
		"Bootstrap de Orquesta para Codex1.",
		"Rol: programador. Proyecto: orquestador.",
		"Directorio de trabajo: " + filepath.Join(tmp, "orquestador") + ".",
	}, "\n")
	if !strings.Contains(prompt, expectedPrefix) {
		t.Fatalf("el bootstrap embebido deberia conservar las primeras lineas completas. prompt=%q argv=%s", prompt, string(argvData))
	}
	if !strings.Contains(prompt, "\nTareas activas: #"+strconv.FormatInt(tareaID, 10)+" [asignada] Implementar bootstrap real.") {
		t.Fatalf("el prompt embebido deberia incluir la tarea activa en una linea propia. prompt=%q argv=%s", prompt, string(argvData))
	}
	if strings.Count(prompt, "\n") < 5 {
		t.Fatalf("el bootstrap embebido deberia seguir siendo multilinea. prompt=%q argv=%s", prompt, string(argvData))
	}

	agente := "Codex1"
	entries, err := db.ListarRuntimeTranscript(db.FiltroRuntimeTranscript{Agente: &agente, Limit: 20})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	foundSystem := false
	for _, item := range entries {
		if item == nil || item.Stream != "system" {
			continue
		}
		if strings.Contains(item.Text, "prompt inicial quedó embebido") {
			foundSystem = true
			break
		}
	}
	if !foundSystem {
		t.Fatalf("el transcript deberia reflejar el prompt inicial embebido: %+v", entries)
	}
}

func TestControlPlaneRunnerExponeEventoAutoGuidancePorAPI(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
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
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador", "sesion-codex1"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-codex1-openclaw-events",
		ResumenContinuidad: "sesion activa para eventos openclaw",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	logPath := filepath.Join(tmp, "codex-transcript-openclaw.log")
	if err := os.WriteFile(logPath, []byte("¿me dejas seguir con el refactor?\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	ctx, cancel := context.WithCancel(context.Background())
	runner := newControlPlaneRunner(nil, false)
	runner.NotificationFeed = nil
	runner.InitNotifications = nil
	runner.Notifier = nil
	runner.ReanimacionCada = time.Hour
	runner.SaludCada = time.Hour
	runner.PlanificacionCada = time.Hour
	runner.ControlPlaneCada = 20 * time.Millisecond
	t.Cleanup(func() {
		cancel()
		runner.Wait()
	})
	runner.Start(ctx)

	deadline := time.Now().Add(5 * time.Second)
	var (
		eventPayload map[string]any
		order        *db.RuntimeOrder
	)
	for time.Now().Before(deadline) {
		var resp struct {
			Events []*db.RuntimeEvent `json:"events"`
		}
		if err := json.Unmarshal(getJSONTestNoBusy(t, mux, "/api/runtime-events?agente=Codex1&proyecto=orquestador&kind=auto_guidance_sent", http.StatusOK), &resp); err != nil {
			t.Fatalf("decode runtime events: %v", err)
		}
		if len(resp.Events) == 0 || resp.Events[0] == nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		eventPayload = resp.Events[0].Payload
		if eventPayload == nil {
			if err := json.Unmarshal([]byte(resp.Events[0].PayloadJSON), &eventPayload); err != nil {
				t.Fatalf("decode runtime event payload: %v payload=%s", err, resp.Events[0].PayloadJSON)
			}
		}
		orderID, _ := eventPayload["runtime_order_id"].(float64)
		if int64(orderID) <= 0 {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		order, err = getRuntimeOrderTestNoBusy(int64(orderID))
		if err != nil {
			t.Fatalf("get runtime order: %v", err)
		}
		if order != nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if order == nil {
		body := string(getJSONTestNoBusy(t, mux, "/api/runtime-events?agente=Codex1&proyecto=orquestador", http.StatusOK))
		t.Fatalf("no apareció auto_guidance_sent visible por API a tiempo: %s", body)
	}
	if order.Tipo != "send_instruction" {
		t.Fatalf("runtime order asociada inesperada: %+v", order)
	}
	if !strings.Contains(order.PayloadJSON, `"classification":"approval_request"`) {
		t.Fatalf("send_instruction sin clasificación de approval_request: %s", order.PayloadJSON)
	}
	classification, _ := eventPayload["classification"].(string)
	if strings.TrimSpace(classification) != "approval_request" {
		t.Fatalf("payload del evento sin clasificación esperada: %+v", eventPayload)
	}
	transcriptID, _ := eventPayload["transcript_id"].(float64)
	if int64(transcriptID) <= 0 {
		t.Fatalf("payload del evento sin transcript_id útil: %+v", eventPayload)
	}
	signalText, _ := eventPayload["signal_text"].(string)
	if !strings.Contains(signalText, "refactor") {
		t.Fatalf("payload del evento sin signal_text accionable: %+v", eventPayload)
	}
	instruction, _ := eventPayload["instruction"].(map[string]any)
	if instruction == nil {
		t.Fatalf("payload del evento sin instruction útil: %+v", eventPayload)
	}
	toAgente, _ := instruction["to_agente"].(string)
	if strings.TrimSpace(toAgente) != "Codex1" {
		t.Fatalf("instruction.to_agente inesperado: %+v", instruction)
	}
	texto, _ := instruction["texto"].(string)
	if !strings.Contains(texto, "Aprobado automaticamente") {
		t.Fatalf("instruction.texto sin guía útil para OpenClaw: %+v", instruction)
	}
}

func TestControlPlaneRunnerRecuperaRuntimeOrderStaleYLaProcesaEndToEnd(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
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
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador", "sesion-codex1"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-codex1-stale",
		ResumenContinuidad: "sesion activa para stale e2e",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := db.GetRuntimeBySesionID(sesion.ID); err != nil {
		t.Fatalf("runtime por sesion: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE config SET valor = '1' WHERE clave = 'runtime_order_stale_seconds'`); err != nil {
		t.Fatalf("config stale seconds: %v", err)
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
	decodeID := func(body []byte) int64 {
		t.Helper()
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode id: %v body=%s", err, string(body))
		}
		id, _ := payload["id"].(float64)
		return int64(id)
	}

	orderID := decodeID(postJSON("/api/runtime-orders", map[string]any{
		"agente":   "Codex1",
		"proyecto": "orquestador",
		"tipo":     "checkpoint",
		"payload":  `{"checkpoint_kind":"handoff_prepare","resumen":"checkpoint stale e2e"}`,
	}, http.StatusCreated))
	if err := db.MarcarRuntimeOrderEstado(orderID, "ejecutando", `{}`, ""); err != nil {
		t.Fatalf("marcar order ejecutando: %v", err)
	}
	staleAt := time.Now().UTC().Add(-10 * time.Minute)
	if _, err := db.DB.Exec(`
		UPDATE runtime_orders
		SET started_at = ?, updated_at = ?
		WHERE id = ?`,
		staleAt, staleAt, orderID,
	); err != nil {
		t.Fatalf("envejecer order: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runner := newControlPlaneRunner(nil, false)
	runner.NotificationFeed = nil
	runner.InitNotifications = nil
	runner.Notifier = nil
	runner.ReanimacionCada = time.Hour
	runner.SaludCada = time.Hour
	runner.PlanificacionCada = time.Hour
	runner.ControlPlaneCada = 20 * time.Millisecond
	t.Cleanup(func() {
		cancel()
		runner.Wait()
	})
	runner.Start(ctx)

	deadline := time.Now().Add(2 * time.Second)
	var sawBatch bool
	for time.Now().Before(deadline) {
		order, err := getRuntimeOrderTestNoBusy(orderID)
		if err != nil {
			if isSQLiteBusyTestErr(err) {
				time.Sleep(20 * time.Millisecond)
				continue
			}
			t.Fatalf("get runtime order: %v", err)
		}
		if !sawBatch {
			accion := "runtime_orders_batch"
			logs, err := listarAuditoriaTestNoBusy(db.FiltroAuditoria{Accion: &accion, Limite: 20})
			if err != nil {
				if isSQLiteBusyTestErr(err) {
					time.Sleep(20 * time.Millisecond)
					continue
				}
				t.Fatalf("listar auditoria batch: %v", err)
			}
			sawBatch = len(logs) > 0
		}
		if order != nil && order.Estado == "completada" && sawBatch {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	order, err := getRuntimeOrderTestNoBusy(orderID)
	if err != nil {
		t.Fatalf("get runtime order final: %v", err)
	}
	if order == nil || order.Estado != "completada" {
		t.Fatalf("runtime order stale no completada: %+v", order)
	}
	if order.StartedAt == nil || !order.StartedAt.After(staleAt) {
		t.Fatalf("runtime order stale sin reproceso efectivo, started_at=%v stale_at=%v order=%+v", order.StartedAt, staleAt, order)
	}

	cp, err := getRuntimeCheckpointBySourceTestNoBusy("runtime_order:" + strconv.FormatInt(orderID, 10))
	if err != nil {
		t.Fatalf("checkpoint por source: %v", err)
	}
	if cp == nil || cp.Resumen != "checkpoint stale e2e" || cp.CheckpointKind != "handoff_prepare" {
		t.Fatalf("checkpoint stale e2e inesperado: %+v", cp)
	}

	if !sawBatch {
		var batchLogs []*db.LogAuditoria
		var err error
		if !sawBatch {
			accion := "runtime_orders_batch"
			batchLogs, err = listarAuditoriaTestNoBusy(db.FiltroAuditoria{Accion: &accion, Limite: 20})
			if err != nil {
				t.Fatalf("listar auditoria batch final: %v", err)
			}
		}
		t.Fatalf("auditoria control plane incompleta: batch=%v batch_logs=%+v", sawBatch, batchLogs)
	}
	cancel()
	time.Sleep(60 * time.Millisecond)
}

func getRuntimeOrderTestNoBusy(id int64) (*db.RuntimeOrder, error) {
	var (
		order *db.RuntimeOrder
		err   error
	)
	for intento := 0; intento < 30; intento++ {
		order, err = db.GetRuntimeOrder(id)
		if err == nil || !isSQLiteBusyTestErr(err) {
			return order, err
		}
		time.Sleep(20 * time.Millisecond)
	}
	return order, err
}

func getRuntimeCheckpointBySourceTestNoBusy(source string) (*db.RuntimeCheckpoint, error) {
	var (
		cp  *db.RuntimeCheckpoint
		err error
	)
	for intento := 0; intento < 30; intento++ {
		cp, err = db.GetRuntimeCheckpointBySource(source)
		if err == nil || !isSQLiteBusyTestErr(err) {
			return cp, err
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cp, err
}

func listarAuditoriaTestNoBusy(filter db.FiltroAuditoria) ([]*db.LogAuditoria, error) {
	var (
		logs []*db.LogAuditoria
		err  error
	)
	for intento := 0; intento < 30; intento++ {
		logs, err = db.ListarAuditoria(filter)
		if err == nil || !isSQLiteBusyTestErr(err) {
			return logs, err
		}
		time.Sleep(20 * time.Millisecond)
	}
	return logs, err
}

func isSQLiteBusyTestErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "sqlite_busy")
}

func getJSONTestNoBusy(t *testing.T, mux *http.ServeMux, path string, wantCode int) []byte {
	t.Helper()
	var lastBody string
	var lastCode int
	for intento := 0; intento < 30; intento++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(rec, req)
		if rec.Code == wantCode {
			return rec.Body.Bytes()
		}
		lastCode = rec.Code
		lastBody = rec.Body.String()
		if isSQLiteBusyTestBody(lastBody) {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		break
	}
	t.Fatalf("status inesperado GET %s: %d body=%s", path, lastCode, lastBody)
	return nil
}

func isSQLiteBusyTestBody(body string) bool {
	msg := strings.ToLower(strings.TrimSpace(body))
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "sqlite_busy")
}
