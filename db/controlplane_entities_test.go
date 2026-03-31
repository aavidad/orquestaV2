package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/coordinacion"
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
)

func TestRuntimeHandleSeSincronizaDesdeSesion(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-001",
		ResumenContinuidad: "continuidad inicial",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}
	if handle == nil {
		t.Fatalf("runtime handle nil")
	}
	if handle.Estado != "activo" {
		t.Fatalf("estado inesperado: %s", handle.Estado)
	}
	if handle.HandleRef != "sess-001" {
		t.Fatalf("handle_ref inesperado: %s", handle.HandleRef)
	}

	pid := int64(4242)
	branch := "feature/controlplane"
	if err := GuardarSesionActiva("Codex1", &proyectoID, SesionUpdate{
		PID:       &pid,
		Branch:    &branch,
		Heartbeat: true,
	}); err != nil {
		t.Fatalf("guardar sesion: %v", err)
	}

	handle, err = GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("reload handle: %v", err)
	}
	if handle.HandleKind != "process" {
		t.Fatalf("handle_kind inesperado: %s", handle.HandleKind)
	}
	if handle.HandleRef != "4242" {
		t.Fatalf("handle_ref tras pid inesperado: %s", handle.HandleRef)
	}
	if handle.RuntimeID == nil {
		t.Fatalf("runtime_id no sincronizado")
	}

	if err := FinSesion("Codex1"); err != nil {
		t.Fatalf("fin sesion: %v", err)
	}
	handle, err = GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("handle final: %v", err)
	}
	if handle == nil || handle.Estado != "cerrado" {
		t.Fatalf("handle no cerrado: %+v", handle)
	}
}

func TestEnviarRuntimeMailboxCoalesceWatchdogPendiente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("antigravity", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	viejoID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "antigravity",
		ProyectoID:  &proyectoID,
		Kind:        "watchdog",
		PayloadJSON: `{"texto":"watchdog viejo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox vieja: %v", err)
	}
	nuevoID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "antigravity",
		ProyectoID:  &proyectoID,
		Kind:        "watchdog",
		PayloadJSON: `{"texto":"watchdog nuevo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox nueva: %v", err)
	}

	pendiente := "pendiente"
	toAgente := "antigravity"
	pendientes, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &pendiente,
	})
	if err != nil {
		t.Fatalf("listar watchdog pendiente: %v", err)
	}
	if len(pendientes) != 1 || pendientes[0].ID != nuevoID {
		t.Fatalf("watchdog pendiente inesperada: %+v", pendientes)
	}

	consumido := "consumido"
	consumidos, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &consumido,
	})
	if err != nil {
		t.Fatalf("listar watchdog consumida: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != viejoID {
		t.Fatalf("watchdog consumida inesperada: %+v", consumidos)
	}
}

func TestEnviarRuntimeMailboxCoalesceNudgePendiente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("antigravity", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	viejoID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "antigravity",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"nudge viejo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox vieja: %v", err)
	}
	nuevoID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "antigravity",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"nudge nuevo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox nueva: %v", err)
	}

	pendiente := "pendiente"
	toAgente := "antigravity"
	pendientes, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &pendiente,
	})
	if err != nil {
		t.Fatalf("listar nudge pendiente: %v", err)
	}
	if len(pendientes) != 1 || pendientes[0].ID != nuevoID {
		t.Fatalf("nudge pendiente inesperada: %+v", pendientes)
	}

	consumido := "consumido"
	consumidos, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &consumido,
	})
	if err != nil {
		t.Fatalf("listar nudge consumida: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != viejoID {
		t.Fatalf("nudge consumida inesperada: %+v", consumidos)
	}
}

func TestEnviarRuntimeMailboxCoalesceFamiliaGuidancePendiente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("antigravity", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	viejoID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "antigravity",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"nudge viejo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox vieja: %v", err)
	}
	nuevoID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "antigravity",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"texto":"autonomia nueva"}`,
	})
	if err != nil {
		t.Fatalf("mailbox nueva: %v", err)
	}

	pendiente := "pendiente"
	toAgente := "antigravity"
	pendientes, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &pendiente,
	})
	if err != nil {
		t.Fatalf("listar guidance pendiente: %v", err)
	}
	if len(pendientes) != 1 || pendientes[0].ID != nuevoID || pendientes[0].Kind != "autonomia" {
		t.Fatalf("guidance pendiente inesperada: %+v", pendientes)
	}

	consumido := "consumido"
	consumidos, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &consumido,
	})
	if err != nil {
		t.Fatalf("listar guidance consumida: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != viejoID || consumidos[0].Kind != "nudge" {
		t.Fatalf("guidance consumida inesperada: %+v", consumidos)
	}
}

func TestRuntimeOrdersMailboxYCheckpoint(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "checkpoint",
		PayloadJSON: `{"motivo":"test"}`,
	})
	if err != nil {
		t.Fatalf("encolar runtime order: %v", err)
	}

	claimed, err := ClaimNextRuntimeOrder("Codex1")
	if err != nil {
		t.Fatalf("claim runtime order: %v", err)
	}
	if claimed == nil || claimed.ID != orderID {
		t.Fatalf("runtime order inesperada: %+v", claimed)
	}
	if claimed.Estado != "tomada" {
		t.Fatalf("estado inesperado tras claim: %s", claimed.Estado)
	}
	if strings.TrimSpace(claimed.LeaseToken) == "" || strings.TrimSpace(claimed.ClaimedBy) == "" || claimed.AttemptCount != 1 || claimed.LeaseExpiresAt == nil {
		t.Fatalf("lease de claim no materializada: %+v", claimed)
	}
	secondClaim, err := ClaimNextRuntimeOrder("Codex1")
	if err != nil {
		t.Fatalf("segundo claim runtime order: %v", err)
	}
	if secondClaim != nil {
		t.Fatalf("no esperaba segunda order claimable: %+v", secondClaim)
	}

	if err := MarcarRuntimeOrderEstado(orderID, "completada", `{"ok":true}`, ""); err != nil {
		t.Fatalf("completar runtime order: %v", err)
	}
	claimed, err = GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get runtime order: %v", err)
	}
	if claimed.Estado != "completada" || claimed.FinishedAt == nil {
		t.Fatalf("runtime order no completada: %+v", claimed)
	}
	if claimed.LeaseExpiresAt != nil || strings.TrimSpace(claimed.LeaseToken) != "" || strings.TrimSpace(claimed.ClaimedBy) != "" {
		t.Fatalf("lease no liberada al completar: %+v", claimed)
	}

	mailID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "Codex1",
		ToAgente:    "Codex2",
		ProyectoID:  &proyectoID,
		Kind:        "consulta",
		PayloadJSON: `{"texto":"revisa esto"}`,
	})
	if err != nil {
		t.Fatalf("enviar mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailID); err != nil {
		t.Fatalf("marcar entregado: %v", err)
	}
	if err := MarcarRuntimeMailboxConsumido(mailID); err != nil {
		t.Fatalf("marcar consumido: %v", err)
	}
	estadoConsumido := "consumido"
	toAgente := "Codex2"
	mensajes, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente: &toAgente,
		Estado:   &estadoConsumido,
	})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(mensajes) != 1 || mensajes[0].ID != mailID {
		t.Fatalf("mailbox inesperado: %+v", mensajes)
	}

	cpID, err := CrearRuntimeCheckpoint(&RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		CheckpointKind: "handoff_prepare",
		Resumen:        "checkpoint de prueba",
		Branch:         "main",
		CWD:            filepath.Join(tmp, "orquestador"),
		PayloadJSON:    `{"archivos":["a.go"]}`,
		ResumeStrategy: "resumen_y_payload",
		Source:         "test",
	})
	if err != nil {
		t.Fatalf("crear runtime checkpoint: %v", err)
	}

	cp, err := UltimoRuntimeCheckpoint("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("ultimo checkpoint: %v", err)
	}
	if cp == nil || cp.ID != cpID {
		t.Fatalf("checkpoint inesperado: %+v", cp)
	}
	if cp.ResumeStrategy != "resumen_y_payload" {
		t.Fatalf("resume strategy inesperada: %s", cp.ResumeStrategy)
	}

	kind := "handoff_prepare"
	source := "test"
	limit := 1
	lista, err := ListarRuntimeCheckpoints(FiltroRuntimeCheckpoints{
		Agente:         &cp.Agente,
		ProyectoID:     &proyectoID,
		CheckpointKind: &kind,
		Source:         &source,
		Limit:          limit,
	})
	if err != nil {
		t.Fatalf("listar runtime checkpoints: %v", err)
	}
	if len(lista) != 1 || lista[0].ID != cpID {
		t.Fatalf("listado de checkpoints inesperado: %+v", lista)
	}
}

func TestEncolarRuntimeOrderRechazaPayloadJSONInvalido(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"texto":"rota"`,
	}); err == nil || !strings.Contains(err.Error(), "payload_json inválido") {
		t.Fatalf("esperaba error de payload inválido, got=%v", err)
	}
}

func TestEjecutarRuntimeOrderSendInstructionFallaConPayloadJSONInvalido(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	res, err := DB.Exec(`
		INSERT INTO runtime_orders (
			agente, proyecto_id, tipo, payload_json, resultado_json, error_text, estado, available_at
		) VALUES (?,?,?,?,?,'', 'pendiente', CURRENT_TIMESTAMP)`,
		"Codex1", proyectoID, "send_instruction", `{"texto":"rota"`, `{}`,
	)
	if err != nil {
		t.Fatalf("insert runtime order invalida: %v", err)
	}
	orderID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get runtime order: %+v err=%v", order, err)
	}

	if err := ejecutarRuntimeOrderSendInstruction(order); err == nil || !strings.Contains(err.Error(), "payload_json inválido") {
		t.Fatalf("esperaba error de payload inválido, got=%v", err)
	}
}

func TestPurgarRuntimeHandlesInactivosBorraYMantieneReferencias(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-purge-runtime-handle",
		Branch:            "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	handleActivo, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handleActivo == nil {
		t.Fatalf("handle activo: %+v err=%v", handleActivo, err)
	}

	res, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado
	) VALUES (?,?,?,?,?,?,?)`,
		"Codex1", proyectoID, runtime.ID, "cli", "process", "stale-1", "cerrado",
	)
	if err != nil {
		t.Fatalf("insert handle cerrado: %v", err)
	}
	handleID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id handle: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handleID,
		Tipo:        "checkpoint",
		PayloadJSON: `{"motivo":"cleanup_test"}`,
	})
	if err != nil {
		t.Fatalf("encolar runtime order: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(orderID, "completada", `{"ok":true}`, ""); err != nil {
		t.Fatalf("completar runtime order: %v", err)
	}
	transcriptID, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtime.ID,
		HandleID:       &handleID,
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "runtime handle viejo",
		NormalizedText: "runtime handle viejo",
		Classification: "note",
	})
	if err != nil {
		t.Fatalf("registrar transcript: %v", err)
	}

	agente := "Codex1"
	resultado, err := PurgarRuntimeHandlesInactivos(FiltroPurgadoRuntimeHandles{Agente: &agente})
	if err != nil {
		t.Fatalf("purgar handles: %v", err)
	}
	if resultado.Deleted != 1 || len(resultado.DeletedIDs) != 1 || resultado.DeletedIDs[0] != handleID {
		t.Fatalf("resultado de purga inesperado: %+v", resultado)
	}
	handleBorrado, err := GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle borrado: %v", err)
	}
	if handleBorrado != nil {
		t.Fatalf("el handle purgado sigue existiendo: %+v", handleBorrado)
	}
	handleActivo, err = GetRuntimeHandle(handleActivo.ID)
	if err != nil || handleActivo == nil || handleActivo.Estado != "activo" {
		t.Fatalf("el handle activo no deberia tocarse: %+v err=%v", handleActivo, err)
	}

	var (
		orderHandleID      sql.NullInt64
		transcriptHandleID sql.NullInt64
	)
	if err := DB.QueryRow(`SELECT handle_id FROM runtime_orders WHERE id = ?`, orderID).Scan(&orderHandleID); err != nil {
		t.Fatalf("leer handle_id runtime order: %v", err)
	}
	if orderHandleID.Valid {
		t.Fatalf("runtime order deberia quedar sin handle_id: %+v", orderHandleID)
	}
	if err := DB.QueryRow(`SELECT handle_id FROM runtime_transcript WHERE id = ?`, transcriptID).Scan(&transcriptHandleID); err != nil {
		t.Fatalf("leer handle_id transcript: %v", err)
	}
	if transcriptHandleID.Valid {
		t.Fatalf("runtime transcript deberia quedar sin handle_id: %+v", transcriptHandleID)
	}
}

func TestPurgarRuntimeHandlesInactivosBloqueaOrdersVivas(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	res, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado
	) VALUES (?,?,?,?,?,?)`,
		"Codex1", proyectoID, "cli", "process", "stale-2", "fallido",
	)
	if err != nil {
		t.Fatalf("insert handle fallido: %v", err)
	}
	handleID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id handle: %v", err)
	}
	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		HandleID:    &handleID,
		Tipo:        "stop",
		PayloadJSON: `{"motivo":"cleanup_test"}`,
	}); err != nil {
		t.Fatalf("encolar runtime order viva: %v", err)
	}

	agente := "Codex1"
	if _, err := PurgarRuntimeHandlesInactivos(FiltroPurgadoRuntimeHandles{Agente: &agente}); err == nil || !strings.Contains(err.Error(), "runtime orders vivas") {
		t.Fatalf("deberia bloquear purga con orders vivas, err=%v", err)
	}
}

func TestPurgarRuntimeOrdersTerminalesBorraSoloTerminalesYDesenlazaMailbox(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"texto":"cleanup terminal"}`,
	})
	if err != nil {
		t.Fatalf("encolar runtime order: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(orderID, "completada", `{"ok":true}`, ""); err != nil {
		t.Fatalf("completar runtime order: %v", err)
	}
	msgID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:     "server",
		ToAgente:       "Codex1",
		ProyectoID:     &proyectoID,
		RuntimeOrderID: &orderID,
		Kind:           "instruction",
		PayloadJSON:    `{"texto":"cleanup terminal"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	orderVivaID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "checkpoint",
		PayloadJSON: `{"motivo":"seguir"}`,
	})
	if err != nil {
		t.Fatalf("encolar runtime order viva: %v", err)
	}

	agente := "Codex1"
	resultado, err := PurgarRuntimeOrdersTerminales(FiltroPurgadoRuntimeOrders{
		Agente:  &agente,
		Estados: []string{"completada"},
		Tipos:   []string{"send_instruction"},
	})
	if err != nil {
		t.Fatalf("purgar runtime orders: %v", err)
	}
	if resultado.Deleted != 1 || len(resultado.DeletedIDs) != 1 || resultado.DeletedIDs[0] != orderID {
		t.Fatalf("resultado de purga inesperado: %+v", resultado)
	}
	orderBorrada, err := GetRuntimeOrder(orderID)
	if err != nil && err != sql.ErrNoRows {
		t.Fatalf("get runtime order borrada: %v", err)
	}
	if orderBorrada != nil {
		t.Fatalf("la runtime order purgada sigue existiendo: %+v", orderBorrada)
	}
	orderViva, err := GetRuntimeOrder(orderVivaID)
	if err != nil || orderViva == nil || orderViva.Estado != "pendiente" {
		t.Fatalf("la runtime order viva no deberia tocarse: %+v err=%v", orderViva, err)
	}
	var mailboxOrderID sql.NullInt64
	if err := DB.QueryRow(`SELECT runtime_order_id FROM runtime_mailbox WHERE id = ?`, msgID).Scan(&mailboxOrderID); err != nil {
		t.Fatalf("leer runtime_order_id mailbox: %v", err)
	}
	if mailboxOrderID.Valid {
		t.Fatalf("mailbox deberia quedar sin runtime_order_id: %+v", mailboxOrderID)
	}
}

func TestPurgarRuntimeOrdersTerminalesBloqueaEstadosVivos(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"texto":"viva"}`,
	}); err != nil {
		t.Fatalf("encolar runtime order viva: %v", err)
	}
	agente := "Codex1"
	if _, err := PurgarRuntimeOrdersTerminales(FiltroPurgadoRuntimeOrders{Agente: &agente, Estados: []string{"pendiente"}}); err == nil || !strings.Contains(err.Error(), "no se permite purgar runtime orders") {
		t.Fatalf("deberia bloquear purga con estado vivo, err=%v", err)
	}
}

func TestPurgarRuntimeOrdersTerminalesRespetaCreatedBefore(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	oldID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"texto":"old terminal"}`,
	})
	if err != nil {
		t.Fatalf("encolar runtime order vieja: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(oldID, "completada", `{"ok":true}`, ""); err != nil {
		t.Fatalf("completar vieja: %v", err)
	}
	recentID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"texto":"recent terminal"}`,
	})
	if err != nil {
		t.Fatalf("encolar runtime order reciente: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(recentID, "completada", `{"ok":true}`, ""); err != nil {
		t.Fatalf("completar reciente: %v", err)
	}
	oldStamp := time.Now().UTC().Add(-2 * time.Hour)
	if _, err := DB.Exec(`UPDATE runtime_orders SET created_at=?, updated_at=? WHERE id=?`, oldStamp, oldStamp, oldID); err != nil {
		t.Fatalf("envejecer order vieja: %v", err)
	}
	cutoff := time.Now().UTC().Add(-60 * time.Minute)
	agente := "Codex1"
	resultado, err := PurgarRuntimeOrdersTerminales(FiltroPurgadoRuntimeOrders{
		Agente:        &agente,
		Estados:       []string{"completada"},
		Tipos:         []string{"send_instruction"},
		CreatedBefore: &cutoff,
	})
	if err != nil {
		t.Fatalf("purgar runtime orders con cutoff: %v", err)
	}
	if resultado.Deleted != 1 || len(resultado.DeletedIDs) != 1 || resultado.DeletedIDs[0] != oldID {
		t.Fatalf("resultado inesperado con cutoff: %+v", resultado)
	}
	if order, err := GetRuntimeOrder(oldID); err != nil && err != sql.ErrNoRows {
		t.Fatalf("get old order: %v", err)
	} else if order != nil {
		t.Fatalf("la order vieja deberia haberse purgado: %+v", order)
	}
	if order, err := GetRuntimeOrder(recentID); err != nil || order == nil {
		t.Fatalf("la order reciente deberia seguir: %+v err=%v", order, err)
	}
}

func TestPurgarRuntimeHistoricoSoloBorraDeudaViejaTerminal(t *testing.T) {
	abrirDBTemporalMemoria(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	if err := ConfigSet("runtime_handles_retention_hours", "24"); err != nil {
		t.Fatalf("ConfigSet handles retention: %v", err)
	}
	if err := ConfigSet("runtime_orders_retention_hours", "72"); err != nil {
		t.Fatalf("ConfigSet orders retention: %v", err)
	}
	if err := ConfigSet("runtime_instances_retention_hours", "72"); err != nil {
		t.Fatalf("ConfigSet runtimes retention: %v", err)
	}
	sesionID, err := IniciarSesion("Codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	sesionID2, err := IniciarSesion("Codex1")
	if err != nil {
		t.Fatalf("IniciarSesion 2: %v", err)
	}
	handleOld, err := GetRuntimeHandleBySesionID(sesionID)
	if err != nil {
		t.Fatalf("GetRuntimeHandleBySesionID old: %v", err)
	}
	if handleOld == nil {
		t.Fatalf("handle old no encontrado")
	}
	handleRecent, err := GetRuntimeHandleBySesionID(sesionID2)
	if err != nil {
		t.Fatalf("GetRuntimeHandleBySesionID recent: %v", err)
	}
	if handleRecent == nil {
		t.Fatalf("handle recent no encontrado")
	}
	handleOldID := handleOld.ID
	handleRecentID := handleRecent.ID
	if _, err := DB.Exec(`UPDATE runtime_handles SET handle_ref = '111', estado = 'cerrado', last_seen_at = ?, created_at = ?, updated_at = ? WHERE id = ?`,
		time.Now().UTC().Add(-48*time.Hour), time.Now().UTC().Add(-48*time.Hour), time.Now().UTC().Add(-48*time.Hour), handleOldID); err != nil {
		t.Fatalf("ajustar old handle: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET handle_ref = '222', estado = 'cerrado', last_seen_at = ?, created_at = ?, updated_at = ? WHERE id = ?`,
		time.Now().UTC().Add(-2*time.Hour), time.Now().UTC().Add(-2*time.Hour), time.Now().UTC().Add(-2*time.Hour), handleRecentID); err != nil {
		t.Fatalf("ajustar recent handle: %v", err)
	}
	orderOldRes, err := DB.Exec(`INSERT INTO runtime_orders (agente, handle_id, tipo, payload_json, resultado_json, estado, created_at, updated_at, available_at) VALUES ('Codex1', ?, 'send_instruction', '{}', '{}', 'fallida', ?, ?, ?)`,
		handleOldID, time.Now().UTC().Add(-96*time.Hour), time.Now().UTC().Add(-96*time.Hour), time.Now().UTC().Add(-96*time.Hour))
	if err != nil {
		t.Fatalf("insert old order: %v", err)
	}
	orderOldID, _ := orderOldRes.LastInsertId()
	orderRecentRes, err := DB.Exec(`INSERT INTO runtime_orders (agente, handle_id, tipo, payload_json, resultado_json, estado, created_at, updated_at, available_at) VALUES ('Codex1', ?, 'send_instruction', '{}', '{}', 'fallida', ?, ?, ?)`,
		handleRecentID, time.Now().UTC().Add(-2*time.Hour), time.Now().UTC().Add(-2*time.Hour), time.Now().UTC().Add(-2*time.Hour))
	if err != nil {
		t.Fatalf("insert recent order: %v", err)
	}
	orderRecentID, _ := orderRecentRes.LastInsertId()
	oldRuntimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		LogicalState: "cerrado",
		ProcessState: "finalizado",
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeInstance old: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances SET last_event_at=?, last_heartbeat_at=?, created_at=?, updated_at=? WHERE id = ?`,
		time.Now().UTC().Add(-96*time.Hour), time.Now().UTC().Add(-96*time.Hour), time.Now().UTC().Add(-96*time.Hour), time.Now().UTC().Add(-96*time.Hour), oldRuntimeID); err != nil {
		t.Fatalf("ajustar old runtime: %v", err)
	}
	recentRuntimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		LogicalState: "cerrado",
		ProcessState: "finalizado",
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeInstance recent: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances SET last_event_at=?, last_heartbeat_at=?, created_at=?, updated_at=? WHERE id = ?`,
		time.Now().UTC().Add(-2*time.Hour), time.Now().UTC().Add(-2*time.Hour), time.Now().UTC().Add(-2*time.Hour), time.Now().UTC().Add(-2*time.Hour), recentRuntimeID); err != nil {
		t.Fatalf("ajustar recent runtime: %v", err)
	}

	resultado, err := PurgarRuntimeHistorico()
	if err != nil {
		t.Fatalf("PurgarRuntimeHistorico: %v", err)
	}
	if resultado == nil || resultado.Handles == nil || resultado.Orders == nil || resultado.Runtimes == nil {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if resultado.Handles.Deleted != 1 || resultado.Orders.Deleted != 1 || resultado.Runtimes.Deleted != 1 {
		t.Fatalf("purga historica inesperada: %+v", resultado)
	}
	if handleOld, err := GetRuntimeHandle(handleOldID); err != nil {
		t.Fatalf("GetRuntimeHandle old: %v", err)
	} else if handleOld != nil {
		t.Fatalf("handle old deberia haber sido purgado: %+v", handleOld)
	}
	if handleRecent, err := GetRuntimeHandle(handleRecentID); err != nil {
		t.Fatalf("GetRuntimeHandle recent: %v", err)
	} else if handleRecent == nil {
		t.Fatalf("handle recent deberia seguir existiendo")
	}
	var countOld, countRecent int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_orders WHERE id = ?`, orderOldID).Scan(&countOld); err != nil {
		t.Fatalf("count old order: %v", err)
	}
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_orders WHERE id = ?`, orderRecentID).Scan(&countRecent); err != nil {
		t.Fatalf("count recent order: %v", err)
	}
	if countOld != 0 || countRecent != 1 {
		t.Fatalf("runtime_orders tras purga inesperadas: old=%d recent=%d", countOld, countRecent)
	}
	if runtimeOld, err := GetRuntime(oldRuntimeID); err != nil && err != sql.ErrNoRows {
		t.Fatalf("GetRuntime old: %v", err)
	} else if runtimeOld != nil {
		t.Fatalf("runtime old deberia haber sido purgado: %+v", runtimeOld)
	}
	if runtimeRecent, err := GetRuntime(recentRuntimeID); err != nil {
		t.Fatalf("GetRuntime recent: %v", err)
	} else if runtimeRecent == nil {
		t.Fatalf("runtime recent deberia seguir existiendo")
	}
}

func TestControlarProcesoRuntimeSoportaHandleLegacySinKind(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	pid := int64(4242)
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		PID:         &pid,
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET handle_kind = '' WHERE id = ?`, handle.ID); err != nil {
		t.Fatalf("vaciar handle_kind: %v", err)
	}

	order := &RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		HandleID:   &handle.ID,
	}
	var got controlruntime.ObjetivoProceso
	aplicado, _, err := controlarProcesoRuntime(order, func(obj controlruntime.ObjetivoProceso) (bool, int, error) {
		got = obj
		return true, 0, nil
	})
	if err != nil || !aplicado {
		t.Fatalf("controlar proceso legacy: aplicado=%t err=%v", aplicado, err)
	}
	if got.PID == nil || *got.PID != pid {
		t.Fatalf("pid legacy inesperado: %+v", got)
	}
}

func TestClaimNextBootstrapRuntimeOrderPriorizaProyectoYTiposBootstrap(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	otroProyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "contabilidad",
		Nombre:  "Contabilidad",
		RutaAbs: filepath.Join(tmp, "contabilidad"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert otro proyecto: %v", err)
	}

	globalStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		Tipo:        "start",
		PayloadJSON: `{"motivo":"global"}`,
	})
	if err != nil {
		t.Fatalf("encolar start global: %v", err)
	}
	proyectoStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"motivo":"proyecto"}`,
	})
	if err != nil {
		t.Fatalf("encolar start proyecto: %v", err)
	}
	handoffID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "handoff",
		PayloadJSON: `{"resumen_continuidad":"handoff preferente"}`,
	})
	if err != nil {
		t.Fatalf("encolar handoff: %v", err)
	}
	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &otroProyectoID,
		Tipo:        "resume",
		PayloadJSON: `{"motivo":"otro proyecto"}`,
	}); err != nil {
		t.Fatalf("encolar resume otro proyecto: %v", err)
	}
	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "sync_status",
		PayloadJSON: `{"motivo":"ignorar"}`,
	}); err != nil {
		t.Fatalf("encolar sync_status: %v", err)
	}

	claimed, err := ClaimNextBootstrapRuntimeOrder("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("claim bootstrap 1: %v", err)
	}
	if claimed == nil || claimed.ID != handoffID || claimed.Tipo != "handoff" {
		t.Fatalf("claim bootstrap 1 inesperado: %+v", claimed)
	}
	if claimed.Estado != "tomada" {
		t.Fatalf("estado claim 1 inesperado: %s", claimed.Estado)
	}

	claimed, err = ClaimNextBootstrapRuntimeOrder("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("claim bootstrap 2: %v", err)
	}
	if claimed == nil || claimed.ID != proyectoStartID || claimed.Tipo != "start" {
		t.Fatalf("claim bootstrap 2 inesperado: %+v", claimed)
	}

	claimed, err = ClaimNextBootstrapRuntimeOrder("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("claim bootstrap 3: %v", err)
	}
	if claimed == nil || claimed.ID != globalStartID || claimed.Tipo != "start" {
		t.Fatalf("claim bootstrap 3 inesperado: %+v", claimed)
	}

	claimed, err = ClaimNextBootstrapRuntimeOrder("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("claim bootstrap 4: %v", err)
	}
	if claimed != nil {
		t.Fatalf("no esperaba más órdenes bootstrap para el proyecto: %+v", claimed)
	}
}

func TestMarcarRuntimeHandlesCerradosPorAgenteRespetaSesionActivaMasNueva(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	s1, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 1: %v", err)
	}
	s2, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "feature/nueva",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 2: %v", err)
	}
	if err := MarcarRuntimeHandlesCerradosPorAgente("Codex1"); err != nil {
		t.Fatalf("marcar handles cerrados: %v", err)
	}

	handle1, err := GetRuntimeHandleBySesionID(s1.ID)
	if err != nil {
		t.Fatalf("handle1: %v", err)
	}
	handle2, err := GetRuntimeHandleBySesionID(s2.ID)
	if err != nil {
		t.Fatalf("handle2: %v", err)
	}
	if handle1 == nil || handle1.Estado != "cerrado" {
		t.Fatalf("handle1 no cerrado: %+v", handle1)
	}
	if handle2 == nil || handle2.Estado != "activo" {
		t.Fatalf("handle2 no deberia cerrarse: %+v", handle2)
	}
}

func TestReconciliarRuntimeHandlesStaleMarcaHandleFallido(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, last_seen_at
	) VALUES (?,?,?,?,?,'activo',?)`,
		"Codex1", proyectoID, "cli", "process", "pid:1", time.Now().UTC().Add(-10*time.Minute)); err != nil {
		t.Fatalf("insert handle stale: %v", err)
	}

	n, err := ReconciliarRuntimeHandlesStale()
	if err != nil {
		t.Fatalf("reconciliar handles stale: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 handle reconciliado, got=%d", n)
	}

	agente := "Codex1"
	handles, err := ListarRuntimeHandles(&agente)
	if err != nil {
		t.Fatalf("listar handles: %v", err)
	}
	if len(handles) != 1 || handles[0].Estado != "fallido" {
		t.Fatalf("handle no marcado como fallido: %+v", handles)
	}
}

func TestReconciliarRuntimeHandlesStaleMantieneHandleVivo(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          int64PtrTest(int64(os.Getpid())),
	})
	if err != nil {
		t.Fatalf("crear runtime: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo',?)`,
		"Codex1", proyectoID, runtimeID, "cli", "process", strconv.Itoa(os.Getpid()), time.Now().UTC().Add(-10*time.Minute)); err != nil {
		t.Fatalf("insert handle vivo: %v", err)
	}

	n, err := ReconciliarRuntimeHandlesStale()
	if err != nil {
		t.Fatalf("reconciliar handles stale: %v", err)
	}
	if n != 0 {
		t.Fatalf("no esperaba handles marcados fallidos si el proceso sigue vivo, got=%d", n)
	}

	handle, err := GetRuntimeHandleActivoAgenteProyecto("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("get handle activo: %v", err)
	}
	if handle == nil || handle.Estado != "activo" {
		t.Fatalf("handle vivo no recuperado como activo: %+v", handle)
	}
}

func TestGetRuntimeHandleActivoAgenteProyectoRecuperaHandleFallidoVivo(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          int64PtrTest(int64(os.Getpid())),
	})
	if err != nil {
		t.Fatalf("crear runtime: %v", err)
	}
	res, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, last_seen_at
	) VALUES (?,?,?,?,?,?, 'fallido', ?)`,
		"Codex1", proyectoID, runtimeID, "cli", "process", strconv.Itoa(os.Getpid()), time.Now().UTC().Add(-10*time.Minute))
	if err != nil {
		t.Fatalf("insert handle fallido vivo: %v", err)
	}
	handleID, _ := res.LastInsertId()

	handle, err := GetRuntimeHandleActivoAgenteProyecto("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("get handle activo con fallback: %v", err)
	}
	if handle == nil || handle.ID != handleID || handle.Estado != "activo" {
		t.Fatalf("handle fallido vivo no recuperado: %+v", handle)
	}
}

func TestGetRuntimeHandleActivoAgenteProyectoDescartaHandleFantasma(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	pidFantasma := int64(99999991)
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "S",
		PID:          &pidFantasma,
	})
	if err != nil {
		t.Fatalf("crear runtime: %v", err)
	}
	res, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?)`,
		"Codex1", proyectoID, runtimeID, "cli", "process", strconv.FormatInt(pidFantasma, 10), time.Now().UTC())
	if err != nil {
		t.Fatalf("insert handle fantasma: %v", err)
	}
	handleID, _ := res.LastInsertId()

	handle, err := GetRuntimeHandleActivoAgenteProyecto("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("get handle activo: %v", err)
	}
	if handle != nil {
		t.Fatalf("no esperaba handle activo para proceso inexistente: %+v", handle)
	}

	fallido, err := GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle fallido: %v", err)
	}
	if fallido == nil || fallido.Estado != "fallido" {
		t.Fatalf("handle fantasma no marcado fallido: %+v", fallido)
	}

	runtime, err := GetRuntime(runtimeID)
	if err != nil {
		t.Fatalf("get runtime: %v", err)
	}
	if runtime == nil || runtime.LogicalState != "fallido" || runtime.ProcessState != "finalizado" {
		t.Fatalf("runtime fantasma no degradado: %+v", runtime)
	}
}

func TestReconciliarRuntimeHandlesStaleMarcaHandleConSesionActivaPeroSinHeartbeat(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET last_seen_at = ?
		WHERE sesion_id = ?`, time.Now().UTC().Add(-10*time.Minute), sesion.ID); err != nil {
		t.Fatalf("envejecer handle: %v", err)
	}

	n, err := ReconciliarRuntimeHandlesStale()
	if err != nil {
		t.Fatalf("reconciliar handles stale: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 handle stale aunque la sesion siga abierta, got=%d", n)
	}

	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}
	if handle == nil || handle.Estado != "fallido" {
		t.Fatalf("handle no marcado como fallido: %+v", handle)
	}
}

func int64PtrTest(v int64) *int64 {
	return &v
}

func TestProcesarRuntimeOrdersBatchCompletaSyncStatusYCheckpoint(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := DB.Exec(`UPDATE config SET valor = '1' WHERE clave = 'runtime_order_batch_size'`); err != nil {
		t.Fatalf("config batch size: %v", err)
	}
	_, err = IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-runner-001",
		ResumenContinuidad: "seguir por aqui",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	syncID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar sync_status: %v", err)
	}
	checkpointID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "checkpoint",
		PayloadJSON: `{"checkpoint_kind":"handoff_prepare","resumen":"checkpoint desde batch"}`,
	})
	if err != nil {
		t.Fatalf("encolar checkpoint: %v", err)
	}

	n, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar batch 1: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 orden procesada en batch 1, got=%d", n)
	}
	n, err = ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar batch 2: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 orden procesada en batch 2, got=%d", n)
	}

	syncOrder, err := GetRuntimeOrder(syncID)
	if err != nil {
		t.Fatalf("get sync order: %v", err)
	}
	if syncOrder.Estado != "completada" {
		t.Fatalf("sync order no completada: %+v", syncOrder)
	}
	var syncResult map[string]any
	if err := json.Unmarshal([]byte(syncOrder.ResultadoJSON), &syncResult); err != nil {
		t.Fatalf("parse sync result: %v", err)
	}
	if ok, _ := syncResult["ok"].(bool); !ok {
		t.Fatalf("sync result no ok: %+v", syncResult)
	}

	checkpointOrder, err := GetRuntimeOrder(checkpointID)
	if err != nil {
		t.Fatalf("get checkpoint order: %v", err)
	}
	if checkpointOrder.Estado != "completada" {
		t.Fatalf("checkpoint order no completada: %+v", checkpointOrder)
	}
	cp, err := UltimoRuntimeCheckpoint("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("ultimo checkpoint: %v", err)
	}
	if cp == nil || cp.CheckpointKind != "handoff_prepare" || cp.Resumen != "checkpoint desde batch" {
		t.Fatalf("checkpoint inesperado: %+v", cp)
	}
}

func TestEjecutarRuntimeOrderStartConsumeMailboxBootstrapSinLease(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "cat-cli",
		Nombre:     "Cat CLI",
		Transporte: "cli",
		Comando:    "cat",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	msgID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"bootstrap durable"}`,
	})
	if err != nil {
		t.Fatalf("enviar runtime mailbox: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"cat-cli"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order: %v", err)
	}
	if err := ejecutarRuntimeOrderStart(startOrder); err != nil {
		t.Fatalf("ejecutar start: %v", err)
	}

	startOrder, err = GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order final: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(startOrder.ResultadoJSON), &result); err != nil {
		t.Fatalf("parse start result: %v", err)
	}
	bootstrap, _ := result["bootstrap"].(map[string]any)
	if int64FromAny(bootstrap["mailbox_count"]) != 1 {
		t.Fatalf("mailbox_count inesperado en start result: %+v", bootstrap)
	}
	if int64FromAny(bootstrap["consumidos"]) != 1 {
		t.Fatalf("consumidos inesperado en start result: %+v", bootstrap)
	}

	estadoConsumido := "consumido"
	toAgente := "Codex1"
	consumidos, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &estadoConsumido,
	})
	if err != nil {
		t.Fatalf("listar runtime mailbox consumido: %v", err)
	}
	found := false
	for _, msg := range consumidos {
		if msg != nil && msg.ID == msgID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("mailbox bootstrap deberia quedar consumida: %+v", consumidos)
	}

	sesion, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion activa: %+v err=%v", sesion, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil || runtime.PID == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	t.Cleanup(func() {
		_, _, _ = controlruntime.DetenerProceso(controlruntime.ObjetivoProceso{PID: runtime.PID})
	})
}

func TestProcesarRuntimeOrdersBatchSyncStatusObservaProcesoLocal(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	pid := int64(os.Getpid())
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		PID:         &pid,
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}

	syncID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar sync_status local: %v", err)
	}
	if _, err := ProcesarRuntimeOrdersBatch(); err != nil {
		t.Fatalf("procesar batch sync local: %v", err)
	}
	syncOrder, err := GetRuntimeOrder(syncID)
	if err != nil || syncOrder == nil {
		t.Fatalf("get sync local: %+v err=%v", syncOrder, err)
	}
	if syncOrder.Estado != "completada" || !strings.Contains(syncOrder.ResultadoJSON, `"observed_process":true`) || !strings.Contains(syncOrder.ResultadoJSON, `"process_alive":true`) {
		t.Fatalf("sync local inesperada: %+v", syncOrder)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil || handle.Estado != "activo" {
		t.Fatalf("handle local inesperado: %+v err=%v", handle, err)
	}
	runtime, err = GetRuntime(runtime.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime local: %+v err=%v", runtime, err)
	}
	if runtime.PID == nil || *runtime.PID != pid {
		t.Fatalf("runtime sin pid supervisado: %+v", runtime)
	}
	if runtime.LastHeartbeatAt == nil || runtime.LastHeartbeatAt.IsZero() {
		t.Fatalf("runtime sin heartbeat tras sync local: %+v", runtime)
	}
	sesion, err = GetSesionByID(sesion.ID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion local: %+v err=%v", sesion, err)
	}
	if sesion.HeartbeatAt == nil || sesion.HeartbeatAt.IsZero() {
		t.Fatalf("sesion sin heartbeat tras sync local: %+v", sesion)
	}
}

func TestProcesarRuntimeSupervisionBatchSupervisaProcesoLocalSinDuplicar(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := ConfigSet("runtime_supervision_interval_seconds", "1"); err != nil {
		t.Fatalf("config supervision interval: %v", err)
	}
	if err := ConfigSet("runtime_supervision_batch_size", "1"); err != nil {
		t.Fatalf("config supervision batch size: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	pid := int64(os.Getpid())
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		PID:         &pid,
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	old := time.Now().UTC().Add(-2 * time.Minute)
	if _, err := DB.Exec(`UPDATE runtime_handles SET last_seen_at = ? WHERE id = ?`, old, handle.ID); err != nil {
		t.Fatalf("envejecer handle: %v", err)
	}

	n, err := ProcesarRuntimeSupervisionBatch()
	if err != nil {
		t.Fatalf("procesar runtime supervision: %v", err)
	}
	if n != 1 {
		t.Fatalf("runtime supervision inesperada, got=%d", n)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle tras supervision: %+v err=%v", handle, err)
	}
	if handle.LastSeenAt == nil || !handle.LastSeenAt.After(old) {
		t.Fatalf("handle sin last_seen actualizado: %+v", handle)
	}
	n, err = ProcesarRuntimeSupervisionBatch()
	if err != nil {
		t.Fatalf("runtime supervision repetida: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia repetir supervision fresca, got=%d", n)
	}
}

func TestReconciliarRuntimeHandlesStaleMarcaHandleActivoConSesionAbierta(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-stale-open",
		ResumenContinuidad: "seguir",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET last_seen_at = ?
		WHERE sesion_id = ?`,
		time.Now().UTC().Add(-10*time.Minute), sesion.ID,
	); err != nil {
		t.Fatalf("envejecer handle: %v", err)
	}

	n, err := ReconciliarRuntimeHandlesStale()
	if err != nil {
		t.Fatalf("reconciliar handles stale: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 handle stale reconciliado, got=%d", n)
	}

	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}
	if handle == nil || handle.Estado != "fallido" {
		t.Fatalf("handle no marcado como fallido: %+v", handle)
	}
}

func TestReconciliarRuntimeOrdersStaleReencolaBasicasYExpiraNoSoportadas(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	syncID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar sync order: %v", err)
	}
	otherID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "handoff",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar handoff order: %v", err)
	}
	if _, err := ClaimNextRuntimeOrder("Codex1"); err != nil {
		t.Fatalf("claim sync order: %v", err)
	}
	if _, err := ClaimNextRuntimeOrder("Codex1"); err != nil {
		t.Fatalf("claim handoff order: %v", err)
	}
	if _, err := DB.Exec(`
		UPDATE runtime_orders
		SET started_at = ?, updated_at = ?, lease_expires_at = ?
		WHERE id IN (?, ?)`,
		time.Now().UTC().Add(-10*time.Minute),
		time.Now().UTC().Add(-10*time.Minute),
		time.Now().UTC().Add(-10*time.Minute),
		syncID, otherID,
	); err != nil {
		t.Fatalf("envejecer orders: %v", err)
	}

	n, err := ReconciliarRuntimeOrdersStale()
	if err != nil {
		t.Fatalf("reconciliar orders stale: %v", err)
	}
	if n != 2 {
		t.Fatalf("esperaba 2 orders stale reconciliadas, got=%d", n)
	}

	syncOrder, err := GetRuntimeOrder(syncID)
	if err != nil {
		t.Fatalf("get sync order: %v", err)
	}
	if syncOrder.Estado != "pendiente" || syncOrder.StartedAt != nil {
		t.Fatalf("sync order no reencolada: %+v", syncOrder)
	}
	if syncOrder.LeaseExpiresAt != nil || strings.TrimSpace(syncOrder.LeaseToken) != "" || strings.TrimSpace(syncOrder.ClaimedBy) != "" {
		t.Fatalf("sync order sin lease limpia tras stale: %+v", syncOrder)
	}

	otherOrder, err := GetRuntimeOrder(otherID)
	if err != nil {
		t.Fatalf("get handoff order: %v", err)
	}
	if otherOrder.Estado != "pendiente" || otherOrder.StartedAt != nil {
		t.Fatalf("handoff order no reencolada: %+v", otherOrder)
	}
	if otherOrder.LeaseExpiresAt != nil || strings.TrimSpace(otherOrder.LeaseToken) != "" || strings.TrimSpace(otherOrder.ClaimedBy) != "" {
		t.Fatalf("handoff order sin lease limpia tras stale: %+v", otherOrder)
	}
}

func TestProcesarRuntimeOrdersBatchIgnoraTiposNoSoportadosYResuelveProyectoDeLaOrden(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "orquesta-a",
		Nombre:  "Orquesta A",
		RutaAbs: filepath.Join(tmp, "orquesta-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "orquesta-b",
		Nombre:  "Orquesta B",
		RutaAbs: filepath.Join(tmp, "orquesta-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}

	sesionA, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoA,
		CWD:                filepath.Join(tmp, "orquesta-a"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-a",
		ResumenContinuidad: "seguir en A",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion A: %v", err)
	}
	runtimeA, err := GetRuntimeBySesionID(sesionA.ID)
	if err != nil || runtimeA == nil {
		t.Fatalf("runtime A: %+v err=%v", runtimeA, err)
	}

	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoB,
		CWD:                filepath.Join(tmp, "orquesta-b"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-b",
		ResumenContinuidad: "seguir en B",
		Branch:             "feature/b",
	}); err != nil {
		t.Fatalf("iniciar sesion B: %v", err)
	}

	unsupportedID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoA,
		Tipo:        "tipo_no_soportado",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar unsupported: %v", err)
	}
	syncID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoA,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar sync: %v", err)
	}

	n, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 order procesada, got=%d", n)
	}

	unsupportedOrder, err := GetRuntimeOrder(unsupportedID)
	if err != nil {
		t.Fatalf("get unsupported order: %v", err)
	}
	if unsupportedOrder.Estado != "pendiente" {
		t.Fatalf("la order no soportada no deberia reclamarse: %+v", unsupportedOrder)
	}

	syncOrder, err := GetRuntimeOrder(syncID)
	if err != nil {
		t.Fatalf("get sync order: %v", err)
	}
	if syncOrder.Estado != "completada" {
		t.Fatalf("sync order no completada: %+v", syncOrder)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(syncOrder.ResultadoJSON), &result); err != nil {
		t.Fatalf("parse sync result: %v", err)
	}
	if sinHandle, _ := result["sin_handle"].(bool); !sinHandle {
		t.Fatalf("esperaba sin_handle=true por cambio de proyecto: %+v", result)
	}
	runtimeID, ok := result["runtime_id"].(float64)
	if !ok || int64(runtimeID) != runtimeA.ID {
		t.Fatalf("runtime resuelto incorrecto: %+v, runtimeA=%d", result, runtimeA.ID)
	}
}

func TestEjecutarRuntimeOrderCheckpointReutilizaCheckpointExistente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-checkpoint",
		ResumenContinuidad: "seguir por aqui",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("runtime por sesion: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "checkpoint",
		PayloadJSON: `{"checkpoint_kind":"handoff_prepare","resumen":"checkpoint reintentado"}`,
	})
	if err != nil {
		t.Fatalf("encolar checkpoint: %v", err)
	}
	existenteID, err := CrearRuntimeCheckpoint(&RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		SesionID:       &sesion.ID,
		RuntimeID:      &runtime.ID,
		CheckpointKind: "handoff_prepare",
		Resumen:        "checkpoint reintentado",
		Branch:         "main",
		CWD:            filepath.Join(tmp, "orquestador"),
		PayloadJSON:    `{"checkpoint_kind":"handoff_prepare","resumen":"checkpoint reintentado"}`,
		ResumeStrategy: "resumen_y_payload",
		Source:         "runtime_order:" + jsonNumber(orderID),
	})
	if err != nil {
		t.Fatalf("crear checkpoint existente: %v", err)
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if err := ejecutarRuntimeOrderCheckpoint(order); err != nil {
		t.Fatalf("ejecutar checkpoint: %v", err)
	}

	cp, err := GetRuntimeCheckpointBySource("runtime_order:" + jsonNumber(orderID))
	if err != nil {
		t.Fatalf("get checkpoint por source: %v", err)
	}
	if cp == nil || cp.ID != existenteID {
		t.Fatalf("checkpoint reutilizado incorrecto: %+v", cp)
	}

	order, err = GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if order.Estado != "completada" {
		t.Fatalf("order no completada: %+v", order)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(order.ResultadoJSON), &result); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	checkpointID, ok := result["checkpoint_id"].(float64)
	if !ok || int64(checkpointID) != existenteID {
		t.Fatalf("resultado no reutiliza checkpoint existente: %+v", result)
	}
}

func TestRuntimeOrderStartYSendInstructionGobiernanProcesoReal(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "cat-cli",
		Nombre:     "Cat CLI",
		Transporte: "cli",
		Comando:    "cat",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"cat-cli"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order: %v", err)
	}
	if err := ejecutarRuntimeOrderStart(startOrder); err != nil {
		t.Fatalf("ejecutar start: %v", err)
	}

	startOrder, err = GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order final: %v", err)
	}
	if startOrder.Estado != "completada" {
		t.Fatalf("start order no completada: %+v", startOrder)
	}

	sesion, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion activa: %+v err=%v", sesion, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil || runtime.PID == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	t.Cleanup(func() {
		_, _, _ = controlruntime.DetenerProceso(controlruntime.ObjetivoProceso{
			PID: runtime.PID,
		})
	})

	var meta map[string]any
	if err := json.Unmarshal([]byte(handle.MetadataJSON), &meta); err != nil {
		t.Fatalf("metadata handle: %v", err)
	}
	stdinPath, _ := meta["stdin_path"].(string)
	logPath, _ := meta["log_path"].(string)
	if strings.TrimSpace(stdinPath) == "" || strings.TrimSpace(logPath) == "" {
		t.Fatalf("metadata de proceso incompleta: %+v", meta)
	}

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"hola runtime"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(sendOrder); err != nil {
		t.Fatalf("ejecutar send_instruction: %v", err)
	}
	sendOrder, err = GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("send_instruction no completada: %+v", sendOrder)
	}

	deadline := time.Now().Add(2 * time.Second)
	var logData []byte
	for {
		logData, err = os.ReadFile(logPath)
		if err == nil && strings.Contains(string(logData), "hola runtime") {
			break
		}
		if time.Now().After(deadline) {
			if err != nil {
				t.Fatalf("leer log: %v", err)
			}
			t.Fatalf("el proceso no recibio la instruccion: %s", string(logData))
		}
		time.Sleep(10 * time.Millisecond)
	}
	agente := "Codex1"
	transcript, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	foundInput := false
	for _, item := range transcript {
		if item != nil && item.Stream == "stdin" && strings.Contains(item.Text, "hola runtime") {
			foundInput = true
			break
		}
	}
	if !foundInput {
		t.Fatalf("send_instruction deberia quedar registrada en transcript: %+v", transcript)
	}

	stopID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "stop",
		PayloadJSON: `{"motivo":"cierre local"}`,
	})
	if err != nil {
		t.Fatalf("encolar stop: %v", err)
	}
	stopOrder, err := GetRuntimeOrder(stopID)
	if err != nil {
		t.Fatalf("get stop order: %v", err)
	}
	if err := ejecutarRuntimeOrderStop(stopOrder); err != nil {
		t.Fatalf("ejecutar stop: %v", err)
	}
	stopOrder, err = GetRuntimeOrder(stopID)
	if err != nil {
		t.Fatalf("get stop order final: %v", err)
	}
	if stopOrder.Estado != "completada" {
		t.Fatalf("stop no completada: %+v", stopOrder)
	}
	sesionActiva, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil && err != sql.ErrNoRows {
		t.Fatalf("get sesion activa tras stop: %v", err)
	}
	if sesionActiva != nil {
		t.Fatalf("la sesion debia quedar cerrada tras stop: %+v", sesionActiva)
	}
	sesion, err = GetSesionByID(sesion.ID)
	if err != nil || sesion == nil {
		t.Fatalf("get sesion final: %+v err=%v", sesion, err)
	}
	if sesion.Activa || sesion.Fin == nil {
		t.Fatalf("la sesion no quedó cerrada: %+v", sesion)
	}
}

func TestRuntimeOrderStartLocalCodexPermiteInputInteractivoPorDefecto(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	codexStub := filepath.Join(tmp, "codex")
	if err := os.WriteFile(codexStub, []byte("#!/bin/sh\nexec cat\n"), 0o755); err != nil {
		t.Fatalf("write codex stub: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "codex-cli",
		Nombre:     "Codex CLI",
		Transporte: "cli",
		Comando:    codexStub,
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"codex-cli"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order: %v", err)
	}
	if err := ejecutarRuntimeOrderStart(startOrder); err != nil {
		t.Fatalf("ejecutar start: %v", err)
	}

	sesion, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion activa: %+v err=%v", sesion, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil || runtime.PID == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	t.Cleanup(func() {
		_, _, _ = controlruntime.DetenerProceso(controlruntime.ObjetivoProceso{PID: runtime.PID})
	})

	if RuntimeHandlePermiteSendInputInteractivo(handle) {
		t.Fatalf("el handle local de codex no deberia permitir input interactivo por defecto: %+v", handle)
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(handle.MetadataJSON), &meta); err != nil {
		t.Fatalf("metadata handle: %v", err)
	}
	if got, ok := meta["can_send_input"].(bool); !ok || got {
		t.Fatalf("metadata deberia fijar can_send_input=false: %+v", meta)
	}
	var caps map[string]any
	if err := json.Unmarshal([]byte(handle.CapabilitiesJSON), &caps); err != nil {
		t.Fatalf("capabilities handle: %v", err)
	}
	if got, ok := caps["can_send_input"].(bool); !ok || got {
		t.Fatalf("capabilities deberian fijar can_send_input=false: %+v", caps)
	}
	if got := RuntimeHandleMailboxDeliveryMode(handle); got != runtimeagente.MailboxDeliveryBootstrapOnly {
		t.Fatalf("mailbox_delivery_mode inesperado: %s", got)
	}
}

func TestRuntimeOrderStopCierraHandleMuertoSinControlReal(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	pidMuerto := int64(999999)
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		PID:         &pidMuerto,
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime por sesion: %+v err=%v", runtime, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle por sesion: %+v err=%v", handle, err)
	}

	stopID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "stop",
		PayloadJSON: `{"motivo":"limpieza de pruebas"}`,
	})
	if err != nil {
		t.Fatalf("encolar stop: %v", err)
	}
	stopOrder, err := GetRuntimeOrder(stopID)
	if err != nil {
		t.Fatalf("get stop order: %v", err)
	}
	if err := ejecutarRuntimeOrderStop(stopOrder); err != nil {
		t.Fatalf("ejecutar stop sobre proceso muerto: %v", err)
	}

	stopOrder, err = GetRuntimeOrder(stopID)
	if err != nil {
		t.Fatalf("get stop order final: %v", err)
	}
	if stopOrder.Estado != "completada" {
		t.Fatalf("stop no completada: %+v", stopOrder)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(stopOrder.ResultadoJSON), &result); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if got, _ := result["control_real"].(bool); got {
		t.Fatalf("stop sobre proceso muerto no deberia marcar control_real=true: %+v", result)
	}
	if got, _ := result["already_stopped"].(bool); !got {
		t.Fatalf("faltaba marca already_stopped en resultado: %+v", result)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle final: %+v err=%v", handle, err)
	}
	if handle.Estado != "cerrado" {
		t.Fatalf("handle no cerrada tras stop tolerante: %+v", handle)
	}
	sesionActiva, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil && err != sql.ErrNoRows {
		t.Fatalf("sesion activa tras stop: %v", err)
	}
	if sesionActiva != nil {
		t.Fatalf("la sesion debia quedar cerrada tras stop tolerante: %+v", sesionActiva)
	}
}

func TestRuntimeOrderStopAparcaSesionAbiertaAunqueEsteStale(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	pidMuerto := int64(999998)
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-stale-stop",
		PID:               &pidMuerto,
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	old := time.Now().UTC().Add(-2 * time.Hour)
	if _, err := DB.Exec(`UPDATE sesiones SET heartbeat_at=? WHERE id=?`, old, sesion.ID); err != nil {
		t.Fatalf("marcar heartbeat stale: %v", err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime por sesion: %+v err=%v", runtime, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle por sesion: %+v err=%v", handle, err)
	}

	stopID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "stop",
		PayloadJSON: `{"motivo":"aparcar sesion stale"}`,
	})
	if err != nil {
		t.Fatalf("encolar stop: %v", err)
	}
	stopOrder, err := GetRuntimeOrder(stopID)
	if err != nil {
		t.Fatalf("get stop order: %v", err)
	}
	if err := ejecutarRuntimeOrderStop(stopOrder); err != nil {
		t.Fatalf("ejecutar stop sobre sesion stale: %v", err)
	}

	sesionAbierta, err := GetSesionAbierta("Codex1", &proyectoID)
	if err != nil && err != sql.ErrNoRows {
		t.Fatalf("get sesion abierta tras stop: %v", err)
	}
	if sesionAbierta != nil {
		t.Fatalf("la sesion abierta stale debia quedar aparcada: %+v", sesionAbierta)
	}
	sesionFinal, err := GetSesionByID(sesion.ID)
	if err != nil || sesionFinal == nil {
		t.Fatalf("get sesion final: %+v err=%v", sesionFinal, err)
	}
	if sesionFinal.Activa {
		t.Fatalf("la sesion stale debia quedar inactiva: %+v", sesionFinal)
	}
	if sesionFinal.Estado != "pausada" && sesionFinal.Estado != "cerrada" {
		t.Fatalf("la sesion stale debia quedar reconciliada tras stop: %+v", sesionFinal)
	}
}

func TestRuntimeHandlePermiteSendInputInteractivoRespetaCapacidadesExplicitas(t *testing.T) {
	if RuntimeHandlePermiteSendInputInteractivo(nil) != true {
		t.Fatal("nil handle deberia permitir por defecto")
	}
	if RuntimeHandlePermiteSendInputInteractivo(&RuntimeHandle{
		CapabilitiesJSON: `{"can_send_input":false}`,
	}) {
		t.Fatal("can_send_input=false deberia desactivar input interactivo")
	}
	if RuntimeHandlePermiteSendInputInteractivo(&RuntimeHandle{
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"/home/alberto/Trabajo/codex-perfiles/bin/codex-perfil Codex2","can_send_input":false}`,
	}) {
		t.Fatal("metadata can_send_input=false deberia desactivar input interactivo")
	}
	if RuntimeHandlePermiteSendInputInteractivo(&RuntimeHandle{
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"/home/alberto/Trabajo/codex-perfiles/bin/codex-perfil Codex2"}`,
	}) {
		t.Fatal("codex local via PTY deberia caer a mailbox por defecto aunque arrastre metadata legacy")
	}
	if RuntimeHandlePermiteSendInputInteractivo(&RuntimeHandle{
		Transporte:   "cli",
		HandleKind:   "process",
		MetadataJSON: `{"herramienta":"codex-cli","external_session_id":"sess-live-1"}`,
	}) {
		t.Fatal("codex cli con metadata degradada no deberia anunciar input interactivo")
	}
	if RuntimeHandlePermiteSendInputInteractivo(&RuntimeHandle{
		Transporte:   "cli",
		HandleKind:   "process",
		MetadataJSON: `{"driver":"process_pty_cli","supervision_mode":"attached"}`,
	}) {
		t.Fatal("proceso local sin stdin_path no deberia anunciar input interactivo")
	}
}

func TestRuntimeHandlePermiteEntregaCalienteSupervisadaExigeSupervisorYStdin(t *testing.T) {
	if RuntimeHandlePermiteEntregaCalienteSupervisada(nil) {
		t.Fatal("nil handle no deberia permitir entrega caliente supervisada")
	}
	if RuntimeHandlePermiteEntregaCalienteSupervisada(&RuntimeHandle{
		MetadataJSON: `{"driver":"process_pty_cli","stdin_path":"/tmp/pty.stdin"}`,
	}) {
		t.Fatal("sin supervisor_ref no deberia permitir entrega caliente supervisada")
	}
	if RuntimeHandlePermiteEntregaCalienteSupervisada(&RuntimeHandle{
		MetadataJSON: `{"driver":"process_pty_cli","supervisor_ref":"/tmp/ref"}`,
	}) {
		t.Fatal("sin stdin_path no deberia permitir entrega caliente supervisada")
	}
	if !RuntimeHandlePermiteEntregaCalienteSupervisada(&RuntimeHandle{
		MetadataJSON: `{"driver":"process_pty_cli","stdin_path":"/tmp/pty.stdin","supervisor_ref":"/tmp/ref","rendered_command":"cat","can_send_input":false}`,
	}) {
		t.Fatal("con supervisor_ref y stdin_path deberia permitir entrega caliente supervisada")
	}
	if !RuntimeHandlePermiteEntregaCalienteSupervisada(&RuntimeHandle{
		MetadataJSON: `{"driver":"process_pty_cli","stdin_path":"/tmp/pty.stdin","supervisor_ref":"/tmp/ref","rendered_command":"codex-perfil Codex2","can_send_input":false}`,
	}) {
		t.Fatal("codex PTY supervisado deberia permitir entrega caliente sin reabrir stdin interactivo")
	}
	if RuntimeHandlePermiteEntregaCalienteSupervisada(&RuntimeHandle{
		MetadataJSON: `{"driver":"process_pty_cli","stdin_path":"/tmp/pty.stdin","supervisor_ref":"/tmp/ref","rendered_command":"codex-perfil Codex2","can_send_input":false,"disable_supervisor_hot_input":true}`,
	}) {
		t.Fatal("un handle degradado no deberia permitir entrega caliente supervisada")
	}
}

func TestRuntimeHandleMailboxDeliveryModeRespetaCapacidadesYFallbacks(t *testing.T) {
	if got := RuntimeHandleMailboxDeliveryMode(nil); got != runtimeagente.MailboxDeliveryInteractive {
		t.Fatalf("modo default inesperado para nil: %s", got)
	}
	if got := RuntimeHandleMailboxDeliveryMode(&RuntimeHandle{
		CapabilitiesJSON: `{"mailbox_delivery_mode":"bootstrap_only","can_send_input":false}`,
	}); got != runtimeagente.MailboxDeliveryBootstrapOnly {
		t.Fatalf("modo explicito inesperado: %s", got)
	}
	if got := RuntimeHandleMailboxDeliveryMode(&RuntimeHandle{
		Transporte:       "cli",
		HandleKind:       "process",
		CapabilitiesJSON: `{"mailbox_delivery_mode":"bootstrap_only","can_send_input":false}`,
		MetadataJSON:     `{"driver":"process_pty_cli","stdin_path":"/tmp/pty.stdin","supervisor_ref":"/tmp/ref","rendered_command":"codex-perfil Codex2"}`,
	}); got != runtimeagente.MailboxDeliveryBootstrapOnly {
		t.Fatalf("codex supervisado deberia preservar bootstrap_only como modo durable base: %s", got)
	}
	if got := RuntimeHandleMailboxDeliveryMode(&RuntimeHandle{
		Transporte:   "cli",
		HandleKind:   "process",
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex2"}`,
	}); got != runtimeagente.MailboxDeliveryBootstrapOnly {
		t.Fatalf("codex local deberia quedarse en bootstrap_only: %s", got)
	}
	if got := RuntimeHandleMailboxDeliveryMode(&RuntimeHandle{
		Transporte:   "cli",
		HandleKind:   "process",
		MetadataJSON: `{"herramienta":"codex-cli","external_session_id":"sess-live-1"}`,
	}); got != runtimeagente.MailboxDeliverySessionResume {
		t.Fatalf("codex con metadata degradada pero external_session_id deberia usar session_resume: %s", got)
	}
}

func TestRuntimeOrderSendInstructionHaceFallbackAMailboxCuandoHandleNoAdmiteInputInteractivo(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "cat-cli",
		Nombre:     "Cat CLI",
		Transporte: "cli",
		Comando:    "cat",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"cat-cli"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order: %v", err)
	}
	if err := ejecutarRuntimeOrderStart(startOrder); err != nil {
		t.Fatalf("ejecutar start: %v", err)
	}

	sesion, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion activa: %+v err=%v", sesion, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil || runtime.PID == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	t.Cleanup(func() {
		_, _, _ = controlruntime.DetenerProceso(controlruntime.ObjetivoProceso{PID: runtime.PID})
	})

	var meta map[string]any
	if err := json.Unmarshal([]byte(handle.MetadataJSON), &meta); err != nil {
		t.Fatalf("metadata handle: %v", err)
	}
	logPath, _ := meta["log_path"].(string)
	meta["driver"] = "process_pty_cli"
	meta["can_send_input"] = false
	meta["supervisor_ref"] = ""
	metaJSON, _ := json.Marshal(meta)
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":       false,
		"can_checkpoint":       true,
		"can_resume":           true,
		"can_capture_pid":      true,
		"can_track_continuity": true,
		"can_pause":            true,
		"can_stop":             true,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("update handle caps: %v", err)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"hola mailbox fallback"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(sendOrder); err != nil {
		t.Fatalf("ejecutar send_instruction: %v", err)
	}
	sendOrder, err = GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("send_instruction no completada: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_id"`) {
		t.Fatalf("send_instruction deberia caer a mailbox: %s", sendOrder.ResultadoJSON)
	}
	if strings.TrimSpace(logPath) != "" {
		time.Sleep(150 * time.Millisecond)
		if data, err := os.ReadFile(logPath); err == nil && strings.Contains(string(data), "hola mailbox fallback") {
			t.Fatalf("el proceso no deberia recibir input interactivo: %s", string(data))
		}
	}
}

func TestRuntimeOrderSendInstructionEntregaEnCalientePorSupervisorLocalSeguroAunqueCanSendInputSeaFalse(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "cat-cli",
		Nombre:     "Cat CLI",
		Transporte: "cli",
		Comando:    "cat",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"cat-cli"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order: %v", err)
	}
	if err := ejecutarRuntimeOrderStart(startOrder); err != nil {
		t.Fatalf("ejecutar start: %v", err)
	}

	sesion, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion activa: %+v err=%v", sesion, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil || runtime.PID == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	t.Cleanup(func() {
		_, _, _ = controlruntime.DetenerProceso(controlruntime.ObjetivoProceso{PID: runtime.PID})
	})

	var meta map[string]any
	if err := json.Unmarshal([]byte(handle.MetadataJSON), &meta); err != nil {
		t.Fatalf("metadata handle: %v", err)
	}
	logPath, _ := meta["log_path"].(string)
	if strings.TrimSpace(logPath) == "" {
		t.Fatalf("log_path vacio en metadata: %+v", meta)
	}
	if strings.TrimSpace(stringFromMap(meta, "stdin_path", "")) == "" || strings.TrimSpace(stringFromMap(meta, "supervisor_ref", "")) == "" {
		t.Fatalf("handle de prueba sin supervisor_ref/stdin_path: %+v", meta)
	}
	meta["driver"] = "process_pty_cli"
	meta["rendered_command"] = "cat"
	meta["can_send_input"] = false
	metaJSON, _ := json.Marshal(meta)
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"can_checkpoint":        true,
		"can_resume":            true,
		"can_capture_pid":       true,
		"can_track_continuity":  true,
		"can_pause":             true,
		"can_stop":              true,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("update handle caps: %v", err)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"hola supervisor local"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(sendOrder); err != nil {
		t.Fatalf("ejecutar send_instruction: %v", err)
	}
	sendOrder, err = GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("send_instruction no completada: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"delivery_path":"supervisor_local"`) {
		t.Fatalf("resultado sin delivery_path supervisor_local: %s", sendOrder.ResultadoJSON)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		data, err := os.ReadFile(logPath)
		if err == nil && strings.Contains(string(data), "hola supervisor local") {
			break
		}
		if time.Now().After(deadline) {
			if err != nil {
				t.Fatalf("leer log: %v", err)
			}
			t.Fatalf("el proceso no recibio la instruccion por supervisor local")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestGuardarSesionActivaPreservaMetadataRicaDelHandle(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-orig-1",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}

	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "process_pty_cli",
		"stdin_path":            filepath.Join(tmp, "pty.stdin"),
		"stdin_raw_path":        filepath.Join(tmp, "pty.stdin.raw"),
		"supervisor_ref":        filepath.Join(tmp, "supervisor.ref"),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
		"rendered_command":      "/home/alberto/Trabajo/codex-perfiles/bin/codex-perfil Codex1",
		"external_session_id":   "sess-orig-1",
		"herramienta":           "codex-cli",
		"cwd":                   filepath.Join(tmp, "orquestador"),
		"branch":                "orq-orquesta-codex1",
	})
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"can_checkpoint":        true,
		"can_resume":            true,
		"can_capture_pid":       true,
		"can_track_continuity":  true,
		"can_pause":             true,
		"can_stop":              true,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("update handle rico: %v", err)
	}

	nuevoExternal := "sess-live-2"
	nuevaBranch := "orq-orquesta-codex1-live"
	if err := GuardarSesionActiva("Codex1", &proyectoID, SesionUpdate{
		ExternalSessionID: &nuevoExternal,
		Branch:            &nuevaBranch,
		Heartbeat:         true,
	}); err != nil {
		t.Fatalf("guardar sesion: %v", err)
	}

	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if got := strings.TrimSpace(stringFromMap(meta, "stdin_path", "")); got == "" {
		t.Fatalf("stdin_path deberia preservarse: %+v", meta)
	}
	if got := strings.TrimSpace(stringFromMap(meta, "supervisor_ref", "")); got == "" {
		t.Fatalf("supervisor_ref deberia preservarse: %+v", meta)
	}
	if got := strings.TrimSpace(stringFromMap(meta, "driver", "")); got != "process_pty_cli" {
		t.Fatalf("driver rico perdido: %+v", meta)
	}
	if got := strings.TrimSpace(stringFromMap(meta, "external_session_id", "")); got != nuevoExternal {
		t.Fatalf("external_session_id no sincronizada: got=%q meta=%+v", got, meta)
	}
	if got := strings.TrimSpace(stringFromMap(meta, "branch", "")); got != nuevaBranch {
		t.Fatalf("branch no sincronizada: got=%q meta=%+v", got, meta)
	}
	caps := mapFromJSON(handle.CapabilitiesJSON)
	if boolFromMap(caps, "can_send_input") {
		t.Fatalf("can_send_input=false deberia preservarse: %+v", caps)
	}
	if got := runtimeagente.NormalizeMailboxDeliveryMode(stringFromMap(caps, "mailbox_delivery_mode", "")); got != runtimeagente.MailboxDeliveryBootstrapOnly {
		t.Fatalf("mailbox_delivery_mode rico perdido: got=%q caps=%+v", got, caps)
	}
}

func TestRuntimeOrderSendInstructionCodexSupervisadoUsaSupervisorLocal(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "cat-cli",
		Nombre:     "Cat CLI",
		Transporte: "cli",
		Comando:    "cat",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"cat-cli"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order: %v", err)
	}
	if err := ejecutarRuntimeOrderStart(startOrder); err != nil {
		t.Fatalf("ejecutar start: %v", err)
	}

	sesion, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion activa: %+v err=%v", sesion, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil || runtime.PID == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	t.Cleanup(func() {
		_, _, _ = controlruntime.DetenerProceso(controlruntime.ObjetivoProceso{PID: runtime.PID})
	})

	var meta map[string]any
	if err := json.Unmarshal([]byte(handle.MetadataJSON), &meta); err != nil {
		t.Fatalf("metadata handle: %v", err)
	}
	logPath, _ := meta["log_path"].(string)
	meta["driver"] = "process_pty_cli"
	meta["can_send_input"] = false
	metaJSON, _ := json.Marshal(meta)
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"can_checkpoint":        true,
		"can_resume":            true,
		"can_capture_pid":       true,
		"can_track_continuity":  true,
		"can_pause":             true,
		"can_stop":              true,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("update handle caps: %v", err)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"hola codex mailbox seguro"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(sendOrder); err != nil {
		t.Fatalf("ejecutar send_instruction: %v", err)
	}
	sendOrder, err = GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" || !strings.Contains(sendOrder.ResultadoJSON, `"delivery_path":"supervisor_local"`) {
		t.Fatalf("codex supervisado deberia usar supervisor_local: %+v", sendOrder)
	}
	if strings.TrimSpace(logPath) != "" {
		deadline := time.Now().Add(2 * time.Second)
		for {
			data, err := os.ReadFile(logPath)
			if err == nil && strings.Contains(string(data), "hola codex mailbox seguro") {
				break
			}
			if time.Now().After(deadline) {
				if err != nil {
					t.Fatalf("leer log: %v", err)
				}
				t.Fatalf("codex supervisado deberia recibir la instruccion por supervisor local: %s", string(data))
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func TestRuntimeOrderSendInstructionCodexSupervisadoDegradadoCaeASessionResume(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir working dir: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: workingDir,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               workingDir,
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-degradada-123",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}

	wrapper := filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(wrapper), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	logPath := filepath.Join(tmp, "resume.log")
	script := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '%s\\n' \"$*\" >>'" + logPath + "'\nexit 0\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                       "process_pty_cli",
		"rendered_command":             "'" + wrapper + "' 'Codex1'",
		"working_dir":                  workingDir,
		"external_session_id":          "sess-degradada-123",
		"stdin_path":                   filepath.Join(tmp, "pty.stdin"),
		"supervisor_ref":               filepath.Join(tmp, "supervisor.ref"),
		"can_send_input":               false,
		"disable_supervisor_hot_input": true,
	})
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"hola degradada"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(sendOrder); err != nil {
		t.Fatalf("ejecutar send_instruction: %v", err)
	}
	sendOrder, err = GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" || !strings.Contains(sendOrder.ResultadoJSON, `"delivery_path":"session_resume"`) {
		t.Fatalf("codex degradado deberia caer a session_resume: %+v", sendOrder)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("leer log resume: %v", err)
	}
	if !strings.Contains(string(data), "exec resume sess-degradada-123 hola degradada") {
		t.Fatalf("session_resume no ejecutado como se esperaba: %q", string(data))
	}
}

func TestRuntimeOrderSendInstructionMailboxSupersedeSesionObsoleta(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	sesionVieja, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-old-1",
	})
	if err != nil {
		t.Fatalf("iniciar sesion vieja: %v", err)
	}
	if err := FinSesion("Codex1"); err != nil {
		t.Fatalf("fin sesion vieja: %v", err)
	}
	sesionNueva, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-new-2",
	})
	if err != nil {
		t.Fatalf("iniciar sesion nueva: %v", err)
	}
	handleNuevo, err := GetRuntimeHandleBySesionID(sesionNueva.ID)
	if err != nil || handleNuevo == nil {
		t.Fatalf("handle nuevo: %+v err=%v", handleNuevo, err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"texto":"hola stale"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	payload, _ := json.Marshal(map[string]any{
		"to_agente":           "Codex1",
		"from_agente":         "server",
		"texto":               "hola stale",
		"mailbox_id":          mailboxID,
		"mailbox_kind":        "autonomia",
		"external_session_id": "sess-old-1",
	})
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   handleNuevo.RuntimeID,
		HandleID:    &handleNuevo.ID,
		Tipo:        "send_instruction",
		PayloadJSON: string(payload),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(sendOrder); err != nil {
		t.Fatalf("ejecutar send_instruction: %v", err)
	}
	sendOrder, err = GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("la orden stale deberia quedar completada por supersede: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"superseded":true`) || !strings.Contains(sendOrder.ResultadoJSON, `"superseded_reason":"external_session_id_changed"`) {
		t.Fatalf("resultado sin supersede: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"external_session_id":"sess-new-2"`) {
		t.Fatalf("resultado sin external_session_id nueva: %s", sendOrder.ResultadoJSON)
	}
	if sesionVieja.ID == sesionNueva.ID {
		t.Fatalf("las sesiones deberian ser distintas")
	}
}

func TestRuntimeOrderSendInstructionMailboxSeCompletaDejandoLaVerdadEnMailboxSiNoHayHandleEntregable(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"hola diferida"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"to_agente":    "Codex1",
		"from_agente":  "server",
		"texto":        "hola diferida",
		"mailbox_id":   mailboxID,
		"mailbox_kind": "instruction",
	})
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: string(payload),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(sendOrder); err != nil {
		t.Fatalf("ejecutar send_instruction: %v", err)
	}

	sendOrder, err = GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("send_instruction de mailbox deberia completarse dejando mailbox pendiente: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"deferred":true`) || !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_only":true`) {
		t.Fatalf("resultado sin marca de diferido: %s", sendOrder.ResultadoJSON)
	}

	agente := "Codex1"
	estado := "pendiente"
	mailbox, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   &agente,
		ProyectoID: &proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailbox) != 1 || mailbox[0].ID != mailboxID {
		t.Fatalf("mailbox pendiente inesperada: %+v", mailbox)
	}
}

func TestRuntimeOrderSendInstructionMailboxCubiertaPorBootstrapSeCompletaSinReintento(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-bootstrap-1",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"texto":"hola bootstrap"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	resumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"mailbox_ids":[%d],"sesion_id":%d}`,
			startID, mailboxID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume bootstrap: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume ejecutando: %v", err)
	}

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"hola bootstrap","mailbox_id":%d,"mailbox_kind":"autonomia"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(sendOrder); err != nil {
		t.Fatalf("ejecutar send_instruction: %v", err)
	}

	sendOrder, err = GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("send_instruction cubierta por bootstrap deberia completarse: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"superseded_reason":"covered_by_bootstrap_lease"`) {
		t.Fatalf("resultado sin supersede de bootstrap: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"bootstrap_order_id":`) {
		t.Fatalf("resultado sin bootstrap_order_id: %s", sendOrder.ResultadoJSON)
	}
}

func TestRuntimeOrderSendInstructionMailboxSeReencolaSiSessionResumeFalla(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	workingDir := filepath.Join(tmp, "orquestador")
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               workingDir,
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-fail-123",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}

	wrapper := filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(wrapper), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	script := "#!/usr/bin/env bash\nset -euo pipefail\necho fallo session resume >&2\nexit 1\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	metaJSON, _ := json.Marshal(map[string]any{
		"driver":              "process_pty_cli",
		"rendered_command":    "'" + wrapper + "' 'Codex1'",
		"working_dir":         workingDir,
		"external_session_id": "sess-fail-123",
		"can_send_input":      false,
	})
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"hola resume"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"to_agente":    "Codex1",
		"from_agente":  "server",
		"texto":        "hola resume",
		"mailbox_id":   mailboxID,
		"mailbox_kind": "instruction",
	})
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: string(payload),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(sendOrder); err != nil {
		t.Fatalf("ejecutar send_instruction: %v", err)
	}

	sendOrder, err = GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("send_instruction deberia reencolarse tras fallo session_resume: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ErrorText, "session_resume fallo") {
		t.Fatalf("error_text sin trazabilidad de session_resume: %s", sendOrder.ErrorText)
	}

	agente := "Codex1"
	estado := "pendiente"
	mailbox, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   &agente,
		ProyectoID: &proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailbox) != 1 || mailbox[0].ID != mailboxID {
		t.Fatalf("mailbox pendiente inesperada: %+v", mailbox)
	}
}

func TestRuntimeOrderSendInstructionAutonomiaTimeoutDegradaAMailboxDurable(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir working dir: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: workingDir,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               workingDir,
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-autonomia-timeout",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}

	wrapper := filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(wrapper), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	script := "#!/usr/bin/env bash\nset -euo pipefail\nsleep 1\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	meta := map[string]any{
		"driver":                    "process_pty_cli",
		"rendered_command":          "'" + wrapper + "' 'Codex1'",
		"working_dir":               workingDir,
		"external_session_id":       "sess-autonomia-timeout",
		"can_send_input":            false,
		"session_resume_timeout_ms": 10,
	}
	metaJSON, _ := json.Marshal(meta)
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"to_agente":    "Codex1",
		"from_agente":  "server",
		"texto":        "Orquesta: continua de forma autonoma",
		"mailbox_id":   mailboxID,
		"mailbox_kind": "autonomia",
	})
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: string(payload),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(sendOrder); err != nil {
		t.Fatalf("ejecutar send_instruction: %v", err)
	}

	sendOrder, err = GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("send_instruction autonomia deberia degradar a mailbox durable: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_only":true`) || !strings.Contains(sendOrder.ResultadoJSON, `"deferred":true`) {
		t.Fatalf("resultado sin marca mailbox_only: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"deferred_reason":"session_resume timeout`) {
		t.Fatalf("resultado sin trazabilidad de timeout: %s", sendOrder.ResultadoJSON)
	}

	agente := "Codex1"
	estado := "pendiente"
	mailbox, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   &agente,
		ProyectoID: &proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailbox) != 1 || mailbox[0].ID != mailboxID {
		t.Fatalf("mailbox pendiente inesperada: %+v", mailbox)
	}
}

func TestRuntimeOrderSendInstructionRebindeaAlHandleActivoYEvitaWorkingDirObsoleto(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	rutaBase := filepath.Join(tmp, "orquestador")
	rutaWorktree := filepath.Join(rutaBase, ".orquesta-worktrees", "orquesta-codex1")
	rutaObsoleta := filepath.Join(tmp, "orquesta-validate-launch")
	if err := os.MkdirAll(rutaWorktree, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaBase,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := (CoordinationWorktreeSQLRepository{}).Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		Agent:     "Codex1",
		Name:      "orquesta-codex1",
		Path:      rutaWorktree,
		Branch:    "orq-orquesta-codex1",
		BaseRef:   "HEAD",
		State:     coordinacion.WorktreeActive,
		Reason:    "test",
	}); err != nil {
		t.Fatalf("crear worktree: %v", err)
	}

	oldRuntimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		LogicalState:      "cerrado",
		ProcessState:      "finalizado",
		CWD:               rutaObsoleta,
		ExternalSessionID: "sess-obsoleta",
	})
	if err != nil {
		t.Fatalf("crear runtime obsoleto: %v", err)
	}
	oldMetaJSON, _ := json.Marshal(map[string]any{
		"driver":              "process_pty_cli",
		"rendered_command":    "'/tmp/obsoleto' 'Codex1'",
		"working_dir":         rutaObsoleta,
		"external_session_id": "sess-obsoleta",
		"can_send_input":      false,
	})
	oldCapsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	res, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, capabilities_json, metadata_json
	) VALUES (?,?,?,?,?,?,?,?,?)`,
		"Codex1", proyectoID, oldRuntimeID, "cli", "process", "999999", "cerrado", string(oldCapsJSON), string(oldMetaJSON))
	if err != nil {
		t.Fatalf("insert handle obsoleto: %v", err)
	}
	oldHandleID, _ := res.LastInsertId()

	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               rutaWorktree,
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-live-999",
	})
	if err != nil {
		t.Fatalf("iniciar sesion activa: %v", err)
	}
	activeHandle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || activeHandle == nil {
		t.Fatalf("runtime handle activo: %+v err=%v", activeHandle, err)
	}
	activeRuntime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || activeRuntime == nil {
		t.Fatalf("runtime activo: %+v err=%v", activeRuntime, err)
	}

	wrapper := filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(wrapper), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	outPath := filepath.Join(tmp, "rebind-session-resume.out")
	script := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '%s\\n' \"$PWD\" > '" + strings.ReplaceAll(outPath, "'", `'\''`) + "'\nprintf '%s\\n' \"$@\" >> '" + strings.ReplaceAll(outPath, "'", `'\''`) + "'\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	activeMetaJSON, _ := json.Marshal(map[string]any{
		"driver":              "process_pty_cli",
		"rendered_command":    "'" + wrapper + "' 'Codex1'",
		"working_dir":         rutaWorktree,
		"external_session_id": "sess-live-999",
		"can_send_input":      false,
	})
	activeCapsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, string(activeMetaJSON), string(activeCapsJSON), activeHandle.ID); err != nil {
		t.Fatalf("update handle activo: %v", err)
	}

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &oldRuntimeID,
		HandleID:    &oldHandleID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"hola rebind"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(sendOrder); err != nil {
		t.Fatalf("ejecutar send_instruction: %v", err)
	}

	sendOrder, err = GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("send_instruction deberia completar tras rebind: %+v", sendOrder)
	}
	if sendOrder.HandleID == nil || *sendOrder.HandleID != activeHandle.ID {
		t.Fatalf("la orden deberia rebindear al handle activo: %+v", sendOrder)
	}
	if sendOrder.RuntimeID == nil || *sendOrder.RuntimeID != activeRuntime.ID {
		t.Fatalf("la orden deberia rebindear al runtime activo: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"handle_id":`+strconv.FormatInt(activeHandle.ID, 10)) {
		t.Fatalf("resultado sin handle activo: %s", sendOrder.ResultadoJSON)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read resume output: %v", err)
	}
	got := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(got) < 6 {
		t.Fatalf("salida rebind incompleta: %q", got)
	}
	if got[0] != rutaWorktree {
		t.Fatalf("session_resume deberia usar la worktree activa: got=%q want=%q", got[0], rutaWorktree)
	}
	wantArgs := []string{"Codex1", "exec", "resume", "sess-live-999", "hola rebind"}
	if strings.Join(got[1:], "|") != strings.Join(wantArgs, "|") {
		t.Fatalf("argv rebind inesperado: got=%q want=%q", got[1:], wantArgs)
	}
}

func TestRuntimeOrderStartInyectaContinuidadAlProcesoReal(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "cat-cli",
		Nombre:     "Cat CLI",
		Transporte: "cli",
		Comando:    "cat",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}

	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ResumenContinuidad: "seguir con la firma final",
		ResumePayloadJSON:  `{"foo":"bar"}`,
		Branch:             "main",
	}); err != nil {
		t.Fatalf("iniciar sesion previa: %v", err)
	}
	if err := AparcarSesionActiva("Codex1", &proyectoID); err != nil {
		t.Fatalf("aparcar sesion previa: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"cat-cli"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order: %v", err)
	}
	if err := ejecutarRuntimeOrderStart(startOrder); err != nil {
		t.Fatalf("ejecutar start: %v", err)
	}

	sesion, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion activa: %+v err=%v", sesion, err)
	}
	if strings.Contains(sesion.ResumePayloadJSON, `"bootstrap_prompt"`) || strings.Contains(sesion.ResumePayloadJSON, `"continuity_prompt"`) {
		t.Fatalf("la sesion no deberia persistir metadatos ephemeros de arranque: %s", sesion.ResumePayloadJSON)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil || runtime.PID == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	t.Cleanup(func() {
		_, _, _ = controlruntime.DetenerProceso(controlruntime.ObjetivoProceso{PID: runtime.PID})
	})

	agente := "Codex1"
	entries, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 20})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	found := false
	for _, item := range entries {
		if item == nil || item.Stream != "stdin" {
			continue
		}
		if strings.Contains(item.Text, "Bootstrap de Orquesta") {
			if len(item.Text) > 100 {
				t.Fatalf("el prompt inicial saneado deberia quedar acotado para PTY: %q", item.Text)
			}
			for _, r := range item.Text {
				if r < 32 || r > 126 {
					t.Fatalf("el prompt inicial saneado deberia quedar en ASCII seguro: %q", item.Text)
				}
			}
			if strings.Contains(strings.ToLower(item.Text), "checkpoint de referencia") {
				t.Fatalf("el prompt inicial PTY no deberia arrastrar el resumen largo completo: %q", item.Text)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no se inyectó el prompt inicial en el transcript stdin: %+v", entries)
	}
}

func TestRuntimeOrderSendInstructionUsaSessionResumeCodexLocal(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir working dir: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: workingDir,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}

	wrapper := filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(wrapper), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	outPath := filepath.Join(tmp, "session-resume.out")
	script := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '%s\\n' \"$@\" > '" + strings.ReplaceAll(outPath, "'", `'\''`) + "'\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	startedAt := time.Now().UTC()
	sessionDir := filepath.Join(filepath.Dir(filepath.Dir(wrapper)), "homes", "Codex1", "sessions", startedAt.In(time.Local).Format("2006"), startedAt.In(time.Local).Format("01"), startedAt.In(time.Local).Format("02"))
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}
	sessionFile := filepath.Join(sessionDir, "rollout-session-resume.jsonl")
	sessionMeta := `{"timestamp":"` + startedAt.Add(time.Second).Format(time.RFC3339Nano) + `","type":"session_meta","payload":{"id":"sess-runtime-resume","timestamp":"` + startedAt.Add(time.Second).Format(time.RFC3339Nano) + `","cwd":"` + workingDir + `"}}` + "\n"
	if err := os.WriteFile(sessionFile, []byte(sessionMeta), 0o644); err != nil {
		t.Fatalf("write session meta: %v", err)
	}

	meta := map[string]any{
		"driver":           "process_pty_cli",
		"rendered_command": "'" + wrapper + "' 'Codex1'",
		"working_dir":      workingDir,
		"started_at":       startedAt.Format(time.RFC3339Nano),
		"can_send_input":   false,
	}
	metaJSON, _ := json.Marshal(meta)
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"hola session resume"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(sendOrder); err != nil {
		t.Fatalf("ejecutar send_instruction: %v", err)
	}
	sendOrder, err = GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" || !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_delivery":"session_resume"`) {
		t.Fatalf("send_instruction deberia usar session_resume: %+v", sendOrder)
	}
	sesion, err = GetSesionByID(sesion.ID)
	if err != nil || sesion == nil {
		t.Fatalf("get sesion final: %+v err=%v", sesion, err)
	}
	if sesion.ExternalSessionID != "sess-runtime-resume" {
		t.Fatalf("external_session_id de sesion inesperado: %+v", sesion)
	}
	runtime, err = GetRuntime(runtime.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime final: %+v err=%v", runtime, err)
	}
	if runtime.ExternalSessionID != "sess-runtime-resume" {
		t.Fatalf("external_session_id de runtime inesperado: %+v", runtime)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read resume output: %v", err)
	}
	got := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{"Codex1", "exec", "resume", "sess-runtime-resume", "hola session resume"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("argv session_resume inesperado: got=%q want=%q", got, want)
	}
}

func TestRuntimeOrderSendInstructionSessionResumeRecuperaWorkingDirPreferida(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	rutaBase := filepath.Join(tmp, "orquestador")
	rutaWorktree := filepath.Join(rutaBase, ".orquesta-worktrees", "orquesta-codex1")
	rutaInvalida := filepath.Join(tmp, "orquesta-validate-launch")
	if err := os.MkdirAll(rutaWorktree, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	if err := os.MkdirAll(rutaInvalida, 0o755); err != nil {
		t.Fatalf("mkdir ruta invalida: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaBase, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaWorktree, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod worktree: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaBase,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := (CoordinationWorktreeSQLRepository{}).Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		Agent:     "Codex1",
		Name:      "orquesta-codex1",
		Path:      rutaWorktree,
		Branch:    "orq-orquesta-codex1",
		BaseRef:   "HEAD",
		State:     coordinacion.WorktreeActive,
		Reason:    "test",
	}); err != nil {
		t.Fatalf("crear worktree: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               rutaInvalida,
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-runtime-resume-live",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}

	wrapper := filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(wrapper), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	outPath := filepath.Join(tmp, "session-resume-preferred.out")
	script := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '%s\\n' \"$PWD\" > '" + strings.ReplaceAll(outPath, "'", `'\''`) + "'\nprintf '%s\\n' \"$@\" >> '" + strings.ReplaceAll(outPath, "'", `'\''`) + "'\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	meta := map[string]any{
		"driver":              "process_pty_cli",
		"rendered_command":    "'" + wrapper + "' 'Codex1'",
		"working_dir":         rutaInvalida,
		"external_session_id": "sess-runtime-resume-live",
		"can_send_input":      false,
	}
	metaJSON, _ := json.Marshal(meta)
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"hola session resume live"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(sendOrder); err != nil {
		t.Fatalf("ejecutar send_instruction: %v", err)
	}
	sendOrder, err = GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" || !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_delivery":"session_resume"`) {
		t.Fatalf("send_instruction deberia usar session_resume: %+v", sendOrder)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read resume output: %v", err)
	}
	got := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(got) < 6 {
		t.Fatalf("salida session_resume incompleta: %q", got)
	}
	if got[0] != rutaWorktree {
		t.Fatalf("session_resume deberia ejecutarse en la worktree activa: got=%q want=%q", got[0], rutaWorktree)
	}
	wantArgs := []string{"Codex1", "exec", "resume", "sess-runtime-resume-live", "hola session resume live"}
	if strings.Join(got[1:], "|") != strings.Join(wantArgs, "|") {
		t.Fatalf("argv session_resume inesperado: got=%q want=%q", got[1:], wantArgs)
	}

	sesion, err = GetSesionByID(sesion.ID)
	if err != nil || sesion == nil {
		t.Fatalf("get sesion final: %+v err=%v", sesion, err)
	}
	if sesion.CWD != rutaWorktree {
		t.Fatalf("cwd de sesion no saneado: got=%s want=%s", sesion.CWD, rutaWorktree)
	}
	runtime, err = GetRuntime(runtime.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime final: %+v err=%v", runtime, err)
	}
	if runtime.CWD != rutaWorktree {
		t.Fatalf("cwd de runtime no saneado: got=%s want=%s", runtime.CWD, rutaWorktree)
	}
}

func TestRuntimeOrderSendInstructionSessionResumePausaPorCuotaProveedor(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir working dir: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: workingDir,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}

	wrapper := filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil")
	if err := os.MkdirAll(filepath.Dir(wrapper), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	script := "#!/usr/bin/env bash\nset -euo pipefail\necho \"Perfil activo: Codex1\" >&2\necho \"CODEX_HOME: /tmp/codex\" >&2\necho \"Consejo: login\" >&2\necho \"ERROR: You've hit your usage limit. Upgrade to Pro, purchase more credits or try again at 12:56 AM.\" >&2\nexit 1\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	startedAt := time.Now().UTC()
	sessionDir := filepath.Join(filepath.Dir(filepath.Dir(wrapper)), "homes", "Codex1", "sessions", startedAt.In(time.Local).Format("2006"), startedAt.In(time.Local).Format("01"), startedAt.In(time.Local).Format("02"))
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}
	sessionFile := filepath.Join(sessionDir, "rollout-session-resume-quota.jsonl")
	sessionMeta := `{"timestamp":"` + startedAt.Add(time.Second).Format(time.RFC3339Nano) + `","type":"session_meta","payload":{"id":"sess-runtime-resume","timestamp":"` + startedAt.Add(time.Second).Format(time.RFC3339Nano) + `","cwd":"` + workingDir + `"}}` + "\n"
	if err := os.WriteFile(sessionFile, []byte(sessionMeta), 0o644); err != nil {
		t.Fatalf("write session meta: %v", err)
	}

	meta := map[string]any{
		"driver":           "process_pty_cli",
		"rendered_command": "'" + wrapper + "' 'Codex1'",
		"working_dir":      workingDir,
		"started_at":       startedAt.Format(time.RFC3339Nano),
		"can_send_input":   false,
	}
	metaJSON, _ := json.Marshal(meta)
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"hola session resume quota"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"hola session resume quota","mailbox_id":%d,"mailbox_kind":"instruction"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(sendOrder); err != nil {
		t.Fatalf("ejecutar send_instruction quota: %v", err)
	}

	sendOrder, err = GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("send_instruction por cuota deberia quedar pendiente con backoff: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"provider_backoff"`) && !strings.Contains(sendOrder.ResultadoJSON, `"deferred"`) {
		t.Fatalf("faltan metadatos de backoff en resultado: %+v", sendOrder)
	}
	if !strings.Contains(strings.ToLower(sendOrder.ErrorText), "enfriamiento") {
		t.Fatalf("faltaba detalle de enfriamiento en error_text: %+v", sendOrder)
	}

	agente, err := GetAgente("Codex1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agente.EstadoCuota != "enfriamiento" {
		t.Fatalf("estado_cuota inesperado: %+v", agente)
	}
	if agente.ReanimarAt == nil {
		t.Fatalf("reanimar_at no deberia ser nil: %+v", agente)
	}
	if !strings.Contains(agente.MotivoPausa, "Auto-pausa por agotamiento") {
		t.Fatalf("motivo_pausa inesperado: %+v", agente)
	}
	presupuesto, sesionActiva, err := UltimoPresupuestoAgente("Codex1")
	if err != nil {
		t.Fatalf("ultimo presupuesto agente: %v", err)
	}
	if sesionActiva == nil || sesionActiva.ID != sesion.ID {
		t.Fatalf("sesion activa inesperada: %+v", sesionActiva)
	}
	if presupuesto == nil {
		t.Fatalf("deberia haberse registrado presupuesto provider_backoff")
	}
	if presupuesto.BudgetSource != "provider_backoff" {
		t.Fatalf("budget source inesperado: %+v", presupuesto)
	}
	if presupuesto.WindowKind != "provider" {
		t.Fatalf("window kind inesperado: %+v", presupuesto)
	}
	if presupuesto.RemainingMessages == nil || *presupuesto.RemainingMessages != 0 {
		t.Fatalf("remaining messages inesperado: %+v", presupuesto)
	}
	if presupuesto.ResetAt == nil || !presupuesto.ResetAt.After(presupuesto.CheckedAt) {
		t.Fatalf("reset_at inesperado: %+v", presupuesto)
	}
	if !strings.Contains(presupuesto.RawSnapshotJSON, `"account_user":"Codex1"`) {
		t.Fatalf("snapshot sin identidad observada: %s", presupuesto.RawSnapshotJSON)
	}
	agente, err = GetAgente("Codex1")
	if err != nil {
		t.Fatalf("get agente enriquecido: %v", err)
	}
	if agente.PresupuestoSesionPct == nil || *agente.PresupuestoSesionPct != 0 {
		t.Fatalf("porcentaje de sesion inesperado: %+v", agente)
	}
	if agente.PresupuestoVentana != "provider" {
		t.Fatalf("ventana efectiva inesperada: %+v", agente)
	}
	if agente.CuentaUsuario != "Codex1" {
		t.Fatalf("usuario observado inesperado: %+v", agente)
	}
}

func TestRuntimeOrderStartRemotoPersisteSesionYHandleSinPID(t *testing.T) {
	tmp := prepararDBTemporal(t)
	calls := make([]string, 0, 2)
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"external_session_id": "sess-remote-42",
			"handle_ref":          "remote-handle-42",
			"capabilities": map[string]any{
				"can_send_input": true,
				"can_pause":      true,
				"can_stop":       true,
			},
		})
	}))
	defer srv.Close()

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "codex-remote",
		Nombre:     "Codex Remote",
		Transporte: "api",
		Comando:    srv.URL,
		MetadataJSON: `{
			"launch_path": "/launch",
			"pause_path": "/pause",
			"continue_path": "/continue",
			"stop_path": "/stop",
			"input_path": "/input"
		}`,
		Activo: true,
	}); err != nil {
		t.Fatalf("upsert conector remoto: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"codex-remote"}`,
	})
	if err != nil {
		t.Fatalf("encolar start remoto: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start remoto: %v", err)
	}
	if err := ejecutarRuntimeOrderStart(startOrder); err != nil {
		t.Fatalf("ejecutar start remoto: %v", err)
	}

	sesion, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion remota: %+v err=%v", sesion, err)
	}
	if sesion.PID != nil {
		t.Fatalf("la sesion remota no debe tener PID: %+v", sesion)
	}
	if sesion.ExternalSessionID != "sess-remote-42" {
		t.Fatalf("external_session_id inesperado: %+v", sesion)
	}

	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle remoto: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime remoto: %+v err=%v", runtime, err)
	}
	if runtime.PID != nil {
		t.Fatalf("el runtime remoto no debe exponer PID: %+v", runtime)
	}
	if handle.HandleKind != "session" || handle.HandleRef != "remote-handle-42" {
		t.Fatalf("handle remoto inesperado: %+v", handle)
	}
	if !strings.Contains(handle.MetadataJSON, `"endpoint":"`+srv.URL+`"`) {
		t.Fatalf("metadata remota inesperada: %s", handle.MetadataJSON)
	}
	if aplicado, _, err := controlruntime.EnviarInstruccionProceso(controlruntime.ObjetivoProceso{
		HandleKind:   handle.HandleKind,
		HandleRef:    handle.HandleRef,
		MetadataJSON: handle.MetadataJSON,
	}, "hola remoto directo"); err != nil || !aplicado {
		t.Fatalf("controlruntime directo no aplicó input remoto: aplicado=%t err=%v metadata=%s", aplicado, err, handle.MetadataJSON)
	}
	calls = calls[:0]

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"hola remoto"}`,
	})
	if err != nil {
		t.Fatalf("encolar send remoto: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send remoto: %v", err)
	}
	handleDesdeOrden, err := GetRuntimeHandle(*sendOrder.HandleID)
	if err != nil || handleDesdeOrden == nil {
		t.Fatalf("handle desde orden: %+v err=%v", handleDesdeOrden, err)
	}
	if aplicado, _, err := controlruntime.EnviarInstruccionProceso(controlruntime.ObjetivoProceso{
		HandleKind:   handleDesdeOrden.HandleKind,
		HandleRef:    handleDesdeOrden.HandleRef,
		MetadataJSON: handleDesdeOrden.MetadataJSON,
	}, "hola remoto desde orden"); err != nil || !aplicado {
		t.Fatalf("controlruntime desde orden no aplicó input remoto: aplicado=%t err=%v handle=%+v", aplicado, err, handleDesdeOrden)
	}
	handleResuelto, err := resolverHandleParaOrden(sendOrder)
	if err != nil || handleResuelto == nil {
		t.Fatalf("resolverHandleParaOrden: %+v err=%v", handleResuelto, err)
	}
	if handleResuelto.HandleKind != handleDesdeOrden.HandleKind || handleResuelto.HandleRef != handleDesdeOrden.HandleRef || handleResuelto.MetadataJSON != handleDesdeOrden.MetadataJSON {
		t.Fatalf("handle resuelto distinto al directo: resuelto=%+v directo=%+v", handleResuelto, handleDesdeOrden)
	}
	if aplicado, _, err := controlarProcesoRuntime(sendOrder, func(obj controlruntime.ObjetivoProceso) (bool, int, error) {
		return controlruntime.EnviarInstruccionProceso(obj, "hola remoto via controlarProcesoRuntime")
	}); err != nil || !aplicado {
		t.Fatalf("controlarProcesoRuntime no aplicó input remoto: aplicado=%t err=%v order=%+v", aplicado, err, sendOrder)
	}
	calls = calls[:0]
	if err := ejecutarRuntimeOrderSendInstruction(sendOrder); err != nil {
		t.Fatalf("send remoto: %v", err)
	}
	sendOrder, err = GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send remoto final: %v", err)
	}
	if sendOrder.Estado != "completada" || !strings.Contains(sendOrder.ResultadoJSON, `"control_real":true`) {
		t.Fatalf("resultado send remoto inesperado: %+v", sendOrder)
	}
	if got := strings.Join(calls, ","); strings.Contains(got, "/launch") || !strings.Contains(got, "/input") {
		t.Fatalf("llamadas remotas inesperadas: %s", got)
	}

	for _, tc := range []struct {
		nombre      string
		tipo        string
		wantPath    string
		wantLogical string
		wantEstado  string
	}{
		{nombre: "pause", tipo: "pause", wantPath: "/pause", wantLogical: "pausado", wantEstado: "pausado"},
		{nombre: "resume", tipo: "resume", wantPath: "/continue", wantLogical: "activo", wantEstado: "activo"},
		{nombre: "stop", tipo: "stop", wantPath: "/stop", wantLogical: "cerrado", wantEstado: "cerrado"},
	} {
		calls = calls[:0]
		if tc.tipo == "resume" {
			if _, err := DB.Exec(`UPDATE runtime_handles SET estado='fallido' WHERE id = ?`, handle.ID); err != nil {
				t.Fatalf("marcar handle fallido antes de resume: %v", err)
			}
		}
		orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
			Agente:      "Codex1",
			ProyectoID:  &proyectoID,
			RuntimeID:   handle.RuntimeID,
			HandleID:    &handle.ID,
			Tipo:        tc.tipo,
			PayloadJSON: `{"motivo":"test remoto"}`,
		})
		if err != nil {
			t.Fatalf("encolar %s remoto: %v", tc.nombre, err)
		}
		order, err := GetRuntimeOrder(orderID)
		if err != nil {
			t.Fatalf("get %s remoto: %v", tc.nombre, err)
		}
		switch tc.tipo {
		case "pause":
			err = ejecutarRuntimeOrderPause(order)
		case "resume":
			err = ejecutarRuntimeOrderResume(order)
		case "stop":
			err = ejecutarRuntimeOrderStop(order)
		}
		if err != nil {
			t.Fatalf("%s remoto: %v", tc.nombre, err)
		}
		order, err = GetRuntimeOrder(orderID)
		if err != nil {
			t.Fatalf("get %s remoto final: %v", tc.nombre, err)
		}
		if order.Estado != "completada" || !strings.Contains(order.ResultadoJSON, `"control_real":true`) {
			t.Fatalf("resultado %s remoto inesperado: %+v", tc.nombre, order)
		}
		runtime, err = GetRuntime(*handle.RuntimeID)
		if err != nil || runtime == nil {
			t.Fatalf("runtime tras %s remoto: %+v err=%v", tc.nombre, runtime, err)
		}
		if runtime.LogicalState != tc.wantLogical {
			t.Fatalf("logical_state tras %s remoto inesperado: %+v", tc.nombre, runtime)
		}
		handle, err = GetRuntimeHandle(handle.ID)
		if err != nil || handle == nil {
			t.Fatalf("handle tras %s remoto: %+v err=%v", tc.nombre, handle, err)
		}
		if handle.Estado != tc.wantEstado {
			t.Fatalf("estado handle tras %s remoto inesperado: %+v", tc.nombre, handle)
		}
		if got := strings.Join(calls, ","); !strings.Contains(got, tc.wantPath) {
			t.Fatalf("llamadas %s remoto inesperadas: %s", tc.nombre, got)
		}
		if tc.tipo == "stop" {
			sesionActiva, err := GetSesionActiva("Codex1", &proyectoID)
			if err != nil && err != sql.ErrNoRows {
				t.Fatalf("sesion activa tras stop remoto: %v", err)
			}
			if sesionActiva != nil {
				t.Fatalf("la sesion remota debia quedar cerrada tras stop: %+v", sesionActiva)
			}
			sesion, err = GetSesionByID(sesion.ID)
			if err != nil || sesion == nil {
				t.Fatalf("get sesion remota final: %+v err=%v", sesion, err)
			}
			if sesion.Activa || sesion.Fin == nil {
				t.Fatalf("la sesion remota no quedó cerrada: %+v", sesion)
			}
		}
	}
}

func TestProcesarRuntimeOrdersBatchDespachaCicloDeVidaRemoto(t *testing.T) {
	tmp := prepararDBTemporal(t)
	calls := make([]string, 0, 8)
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/launch":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"external_session_id": "sess-remote-batch",
				"handle_ref":          "remote-handle-batch",
				"capabilities": map[string]any{
					"can_send_input": true,
					"can_pause":      true,
					"can_stop":       true,
				},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}
	}))
	defer srv.Close()

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "codex-remote-batch",
		Nombre:     "Codex Remote Batch",
		Transporte: "api",
		Comando:    srv.URL,
		MetadataJSON: `{
			"launch_path": "/launch",
			"pause_path": "/pause",
			"continue_path": "/continue",
			"stop_path": "/stop",
			"input_path": "/input"
		}`,
		Activo: true,
	}); err != nil {
		t.Fatalf("upsert conector remoto: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"codex-remote-batch"}`,
	})
	if err != nil {
		t.Fatalf("encolar start remoto batch: %v", err)
	}
	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("ProcesarRuntimeOrdersBatch start: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed start=%d", processed)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil || startOrder == nil {
		t.Fatalf("get start order: %+v err=%v", startOrder, err)
	}
	if startOrder.Estado != "completada" {
		t.Fatalf("start batch no completada: %+v", startOrder)
	}
	if got := strings.Join(calls, ","); !strings.Contains(got, "/launch") {
		t.Fatalf("batch start sin launch remoto: %s", got)
	}

	sesion, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion remota batch: %+v err=%v", sesion, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle remoto batch: %+v err=%v", handle, err)
	}

	for _, tc := range []struct {
		nombre   string
		tipo     string
		payload  string
		wantPath string
	}{
		{nombre: "send", tipo: "send_instruction", payload: `{"to_agente":"Codex1","texto":"hola batch"}`, wantPath: "/input"},
		{nombre: "pause", tipo: "pause", payload: `{"motivo":"pause batch"}`, wantPath: "/pause"},
		{nombre: "resume", tipo: "resume", payload: `{"motivo":"resume batch"}`, wantPath: "/continue"},
		{nombre: "stop", tipo: "stop", payload: `{"motivo":"stop batch"}`, wantPath: "/stop"},
	} {
		calls = calls[:0]
		orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
			Agente:      "Codex1",
			ProyectoID:  &proyectoID,
			RuntimeID:   handle.RuntimeID,
			HandleID:    &handle.ID,
			Tipo:        tc.tipo,
			PayloadJSON: tc.payload,
		})
		if err != nil {
			t.Fatalf("encolar %s batch: %v", tc.nombre, err)
		}
		processed, err := ProcesarRuntimeOrdersBatch()
		if err != nil {
			t.Fatalf("ProcesarRuntimeOrdersBatch %s: %v", tc.nombre, err)
		}
		if processed != 1 {
			t.Fatalf("processed %s=%d", tc.nombre, processed)
		}
		order, err := GetRuntimeOrder(orderID)
		if err != nil || order == nil {
			t.Fatalf("get %s order: %+v err=%v", tc.nombre, order, err)
		}
		if order.Estado != "completada" {
			t.Fatalf("%s batch no completada: %+v", tc.nombre, order)
		}
		if got := strings.Join(calls, ","); !strings.Contains(got, tc.wantPath) {
			t.Fatalf("llamadas %s batch inesperadas: %s", tc.nombre, got)
		}
	}

	sesionActiva, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil && err != sql.ErrNoRows {
		t.Fatalf("sesion activa tras stop batch: %v", err)
	}
	if sesionActiva != nil {
		t.Fatalf("la sesion batch debia quedar cerrada: %+v", sesionActiva)
	}
}

func TestProcesarRuntimeOrdersBatchNoEjecutaHandoffBootstrapPendiente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	handoffID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "handoff",
		PayloadJSON: `{"agente_origen":"Codex0","agente_destino":"Codex1","resumen_continuidad":"handoff pendiente"}`,
	})
	if err != nil {
		t.Fatalf("encolar handoff: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("ProcesarRuntimeOrdersBatch: %v", err)
	}
	if processed != 0 {
		t.Fatalf("el batch no deberia ejecutar handoff bootstrap, processed=%d", processed)
	}

	order, err := GetRuntimeOrder(handoffID)
	if err != nil || order == nil {
		t.Fatalf("get handoff: %+v err=%v", order, err)
	}
	if order.Estado != "pendiente" {
		t.Fatalf("handoff bootstrap no deberia salir de pendiente: %+v", order)
	}

	claimed, err := claimBootstrapRuntimeOrderParaStart("Codex1", &proyectoID, 0)
	if err != nil {
		t.Fatalf("claim bootstrap handoff: %v", err)
	}
	if claimed == nil || claimed.ID != handoffID || claimed.Estado != "tomada" {
		t.Fatalf("claim bootstrap inesperado: %+v", claimed)
	}
}

func TestProcesarRuntimeOrdersBatchEncolaReinicioParaHandoffBootstrapConHandleActivo(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion activa: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get handle activo: %v", err)
	}
	if handle == nil || handle.Estado != "activo" {
		t.Fatalf("handle activo inesperado: %+v", handle)
	}

	handoffID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "handoff",
		PayloadJSON: `{"agente_origen":"Codex0","agente_destino":"Codex1","resumen_continuidad":"handoff pendiente"}`,
	})
	if err != nil {
		t.Fatalf("encolar handoff: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("ProcesarRuntimeOrdersBatch: %v", err)
	}
	if processed != 1 {
		t.Fatalf("deberia promover un reinicio coordinado para el handoff pendiente, processed=%d", processed)
	}

	order, err := GetRuntimeOrder(handoffID)
	if err != nil || order == nil {
		t.Fatalf("get handoff: %+v err=%v", order, err)
	}
	if order.Estado != "pendiente" {
		t.Fatalf("handoff bootstrap deberia seguir pendiente hasta el siguiente start: %+v", order)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar ordenes pendientes: %v", err)
	}
	var stops, starts, handoffs int
	for _, item := range orders {
		if item == nil {
			continue
		}
		switch item.Tipo {
		case "stop":
			stops++
		case "start":
			starts++
		case "handoff":
			handoffs++
		}
	}
	if handoffs != 1 || stops != 1 || starts != 1 {
		t.Fatalf("ordenes bootstrap/reinicio inesperadas: handoffs=%d stops=%d starts=%d orders=%+v", handoffs, stops, starts, orders)
	}
}

func TestRuntimeOrderPauseCheckpointeaYSesionQuedaPausada(t *testing.T) {
	tmp := prepararDBTemporal(t)
	calls := make([]string, 0, 8)
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/launch":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"external_session_id": "sess-remote-pause",
				"handle_ref":          "remote-handle-pause",
				"capabilities": map[string]any{
					"can_send_input": true,
					"can_pause":      true,
					"can_stop":       true,
				},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}
	}))
	defer srv.Close()

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "codex-remote-pause",
		Nombre:     "Codex Remote Pause",
		Transporte: "api",
		Comando:    srv.URL,
		MetadataJSON: `{
			"launch_path": "/launch",
			"pause_path": "/pause",
			"continue_path": "/continue",
			"stop_path": "/stop",
			"input_path": "/input"
		}`,
		Activo: true,
	}); err != nil {
		t.Fatalf("upsert conector remoto: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"codex-remote-pause"}`,
	})
	if err != nil {
		t.Fatalf("encolar start remoto: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order: %v", err)
	}
	if err := ejecutarRuntimeOrderStart(startOrder); err != nil {
		t.Fatalf("ejecutar start: %v", err)
	}

	sesion, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion activa: %+v err=%v", sesion, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle remoto: %+v err=%v", handle, err)
	}

	pauseID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "pause",
		PayloadJSON: `{"motivo":"esperando humano"}`,
	})
	if err != nil {
		t.Fatalf("encolar pause: %v", err)
	}
	pauseOrder, err := GetRuntimeOrder(pauseID)
	if err != nil {
		t.Fatalf("get pause order: %v", err)
	}
	if err := ejecutarRuntimeOrderPause(pauseOrder); err != nil {
		t.Fatalf("ejecutar pause: %v", err)
	}
	cp, err := GetRuntimeCheckpointBySource("runtime_order:" + jsonNumber(pauseID) + ":pause")
	if err != nil || cp == nil {
		t.Fatalf("checkpoint pause: %+v err=%v", cp, err)
	}
	sesion, err = GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion tras pause: %+v err=%v", sesion, err)
	}
	if sesion.Estado != "pausada" {
		t.Fatalf("estado de sesion tras pause inesperado: %+v", sesion)
	}
	reglaID, err := UpsertRegla(&Regla{
		TipoAgente:  "programador",
		Categoria:   "arquitectura",
		Titulo:      "regla-resume-reconcile",
		Descripcion: "regla para comprobar refresh en resume",
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("upsert regla: %v", err)
	}
	if _, err := GuardarGovernanceOverride("tester", &GovernanceOverride{
		TipoAgente: "programador",
		ScopeTipo:  GovernanceScopeProyecto,
		ScopeRef:   "orquestador",
		Entidad:    GovernanceEntityRegla,
		EntidadID:  reglaID,
		Accion:     GovernanceActionDisable,
	}); err != nil {
		t.Fatalf("guardar override: %v", err)
	}
	mailboxAntes, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   stringPtr("Codex1"),
		ProyectoID: &proyectoID,
	})
	if err != nil {
		t.Fatalf("mailbox antes de resume: %v", err)
	}

	resumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "resume",
		PayloadJSON: `{"motivo":"desbloqueado"}`,
	})
	if err != nil {
		t.Fatalf("encolar resume: %v", err)
	}
	resumeOrder, err := GetRuntimeOrder(resumeID)
	if err != nil {
		t.Fatalf("get resume order: %v", err)
	}
	if err := ejecutarRuntimeOrderResume(resumeOrder); err != nil {
		t.Fatalf("ejecutar resume: %v", err)
	}
	sesion, err = GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion tras resume: %+v err=%v", sesion, err)
	}
	if sesion.Estado != "activa" {
		t.Fatalf("estado de sesion tras resume inesperado: %+v", sesion)
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(sesion.ResumePayloadJSON), &envelope); err != nil {
		t.Fatalf("resume payload invalido: %v (%s)", err, sesion.ResumePayloadJSON)
	}
	governanceCatalog, ok := envelope["governance_catalog"].(map[string]any)
	if !ok {
		t.Fatalf("resume payload sin governance_catalog: %+v", envelope)
	}
	if got := governanceCatalog["resolucion_actual"]; got != "rol+proyecto" {
		t.Fatalf("resolucion_actual inesperada tras resume: %#v", got)
	}
	if !strings.Contains(sesion.ResumenContinuidad, "Catálogo efectivo") {
		t.Fatalf("resumen continuidad no reconciliado: %s", sesion.ResumenContinuidad)
	}
	mailboxDespues, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   stringPtr("Codex1"),
		ProyectoID: &proyectoID,
	})
	if err != nil {
		t.Fatalf("mailbox despues de resume: %v", err)
	}
	if len(mailboxDespues) <= len(mailboxAntes) {
		t.Fatalf("resume deberia haber emitido governance_refresh propio: antes=%d despues=%d", len(mailboxAntes), len(mailboxDespues))
	}
	foundResumeReconcile := false
	for _, msg := range mailboxDespues {
		if msg == nil || msg.Kind != MailboxKindGovernanceRefresh {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(msg.PayloadJSON), &payload); err != nil {
			t.Fatalf("payload mailbox invalido: %v (%s)", err, msg.PayloadJSON)
		}
		if payload["motivo"] != "resume_reconcile" {
			continue
		}
		foundResumeReconcile = true
		if payload["hash"] != governanceCatalog["hash"] {
			t.Fatalf("hash de refresh inesperado: got=%#v want=%#v", payload["hash"], governanceCatalog["hash"])
		}
	}
	if !foundResumeReconcile {
		t.Fatalf("no se emitio governance_refresh resume_reconcile: %+v", mailboxDespues)
	}
	if got := strings.Join(calls, ","); !strings.Contains(got, "/pause") || !strings.Contains(got, "/continue") {
		t.Fatalf("llamadas pause/resume inesperadas: %s", got)
	}
}

func TestRuntimeOrderStopPreservaSesionOAuthSinCerrarla(t *testing.T) {
	tmp := prepararDBTemporal(t)
	calls := make([]string, 0, 8)
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/launch":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"external_session_id": "sess-remote-oauth",
				"handle_ref":          "remote-handle-oauth",
				"capabilities": map[string]any{
					"can_send_input":          true,
					"can_pause":               true,
					"can_stop":                true,
					"can_stop_without_reauth": false,
				},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}
	}))
	defer srv.Close()

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "codex-remote-oauth",
		Nombre:     "Codex Remote OAuth",
		Transporte: "api",
		Comando:    srv.URL,
		MetadataJSON: `{
			"launch_path": "/launch",
			"pause_path": "/pause",
			"continue_path": "/continue",
			"stop_path": "/stop",
			"input_path": "/input",
			"auth_mode": "oauth",
			"preserve_external_session_on_stop": true
		}`,
		Activo: true,
	}); err != nil {
		t.Fatalf("upsert conector oauth: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"codex-remote-oauth"}`,
	})
	if err != nil {
		t.Fatalf("encolar start remoto: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order: %v", err)
	}
	if err := ejecutarRuntimeOrderStart(startOrder); err != nil {
		t.Fatalf("ejecutar start: %v", err)
	}

	sesion, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion activa: %+v err=%v", sesion, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle remoto: %+v err=%v", handle, err)
	}

	stopID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "stop",
		PayloadJSON: `{"motivo":"reubicar temporalmente"}`,
	})
	if err != nil {
		t.Fatalf("encolar stop: %v", err)
	}
	stopOrder, err := GetRuntimeOrder(stopID)
	if err != nil {
		t.Fatalf("get stop order: %v", err)
	}
	if err := ejecutarRuntimeOrderStop(stopOrder); err != nil {
		t.Fatalf("ejecutar stop preservado: %v", err)
	}

	cp, err := GetRuntimeCheckpointBySource("runtime_order:" + jsonNumber(stopID) + ":stop")
	if err != nil || cp == nil {
		t.Fatalf("checkpoint stop preservado: %+v err=%v", cp, err)
	}
	sesionActiva, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil && err != sql.ErrNoRows {
		t.Fatalf("sesion activa tras stop preservado: %v", err)
	}
	if sesionActiva != nil {
		t.Fatalf("no debía quedar sesión activa tras aparcar: %+v", sesionActiva)
	}
	sesion, err = GetSesionByID(sesion.ID)
	if err != nil || sesion == nil {
		t.Fatalf("get sesion preservada: %+v err=%v", sesion, err)
	}
	if sesion.Activa || sesion.Fin != nil || sesion.Estado != "pausada" {
		t.Fatalf("sesion preservada inesperada: %+v", sesion)
	}
	runtime, err := GetRuntime(*handle.RuntimeID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime preservado: %+v err=%v", runtime, err)
	}
	if runtime.LogicalState != "pausado" {
		t.Fatalf("logical_state preservado inesperado: %+v", runtime)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle preservado: %+v err=%v", handle, err)
	}
	if handle.Estado != "pausado" {
		t.Fatalf("estado handle preservado inesperado: %+v", handle)
	}
	if got := strings.Join(calls, ","); !strings.Contains(got, "/pause") || strings.Contains(got, "/stop") {
		t.Fatalf("llamadas stop preservado inesperadas: %s", got)
	}
}

func TestProcesarRuntimeOrdersBatchSyncStatusObservaEstadoRemoto(t *testing.T) {
	tmp := prepararDBTemporal(t)
	calls := make([]string, 0, 4)
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/launch":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"external_session_id": "sess-remote-sync",
				"handle_ref":          "remote-handle-sync",
				"capabilities": map[string]any{
					"can_send_input": true,
					"can_pause":      true,
					"can_stop":       true,
				},
			})
		case "/status":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"handle": map[string]any{
					"estado": "degradado",
					"metadata": map[string]any{
						"remote_version": "1.2.3",
					},
					"capabilities": map[string]any{
						"can_send_input": true,
						"can_pause":      false,
					},
				},
				"runtime": map[string]any{
					"logical_state": "esperando",
					"process_state": "idle",
				},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}
	}))
	defer srv.Close()

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "codex-remote-sync",
		Nombre:     "Codex Remote Sync",
		Transporte: "api",
		Comando:    srv.URL,
		MetadataJSON: `{
			"launch_path": "/launch",
			"status_path": "/status"
		}`,
		Activo: true,
	}); err != nil {
		t.Fatalf("upsert conector remoto: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"codex-remote-sync"}`,
	})
	if err != nil {
		t.Fatalf("encolar start remoto sync: %v", err)
	}
	if _, err := ProcesarRuntimeOrdersBatch(); err != nil {
		t.Fatalf("procesar batch start: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil || startOrder == nil || startOrder.Estado != "completada" {
		t.Fatalf("start remoto sync inesperado: %+v err=%v", startOrder, err)
	}
	sesion, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion remota sync: %+v err=%v", sesion, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle remoto sync: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime remoto sync: %+v err=%v", runtime, err)
	}

	syncID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar sync remoto: %v", err)
	}
	if _, err := ProcesarRuntimeOrdersBatch(); err != nil {
		t.Fatalf("procesar batch sync: %v", err)
	}
	syncOrder, err := GetRuntimeOrder(syncID)
	if err != nil || syncOrder == nil {
		t.Fatalf("get sync remoto: %+v err=%v", syncOrder, err)
	}
	if syncOrder.Estado != "completada" || !strings.Contains(syncOrder.ResultadoJSON, `"observed_remote":true`) {
		t.Fatalf("sync remoto no observado: %+v", syncOrder)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle remoto sync: %+v err=%v", handle, err)
	}
	if handle.Estado != "activo" ||
		!strings.Contains(handle.MetadataJSON, `"remote_version":"1.2.3"`) ||
		!strings.Contains(handle.MetadataJSON, `"remote_handle_state":"degradado"`) {
		t.Fatalf("handle remoto sync inesperado: %+v", handle)
	}
	runtime, err = GetRuntime(runtime.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime remoto sync: %+v err=%v", runtime, err)
	}
	if runtime.LogicalState != "esperando" || runtime.ProcessState != "idle" {
		t.Fatalf("runtime remoto sync inesperado: %+v", runtime)
	}
	if got := strings.Join(calls, ","); !strings.Contains(got, "/status") {
		t.Fatalf("sync remoto sin llamada status: %s", got)
	}
}

func TestProcesarRuntimeOrdersBatchSyncStatusDegradaRuntimeTrasFallosRemotos(t *testing.T) {
	tmp := prepararDBTemporal(t)

	calls := make([]string, 0, 8)
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/launch":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"external_session_id": "sess-remote-sync-fail",
				"handle_ref":          "remote-sync-fail",
			})
		case "/status":
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"adapter down"}`))
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}
	}))
	defer srv.Close()

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := ConfigSet("runtime_remote_sync_failure_threshold", "2"); err != nil {
		t.Fatalf("config threshold: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "codex-remote-sync-fail",
		Nombre:     "Codex Remote Sync Fail",
		Transporte: "api",
		Comando:    srv.URL,
		MetadataJSON: `{
			"launch_path": "/launch",
			"status_path": "/status"
		}`,
		Activo: true,
	}); err != nil {
		t.Fatalf("upsert conector remoto: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"codex-remote-sync-fail"}`,
	})
	if err != nil {
		t.Fatalf("encolar start remoto: %v", err)
	}
	if _, err := ProcesarRuntimeOrdersBatch(); err != nil {
		t.Fatalf("procesar batch start: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil || startOrder == nil || startOrder.Estado != "completada" {
		t.Fatalf("start remoto inesperado: %+v err=%v", startOrder, err)
	}
	sesion, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion remota: %+v err=%v", sesion, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle remoto: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime remoto: %+v err=%v", runtime, err)
	}

	for intento := 1; intento <= 2; intento++ {
		syncID, err := EncolarRuntimeOrder(&RuntimeOrder{
			Agente:      "Codex1",
			ProyectoID:  &proyectoID,
			RuntimeID:   handle.RuntimeID,
			HandleID:    &handle.ID,
			Tipo:        "sync_status",
			PayloadJSON: `{}`,
		})
		if err != nil {
			t.Fatalf("encolar sync remoto intento %d: %v", intento, err)
		}
		if _, err := ProcesarRuntimeOrdersBatch(); err != nil {
			t.Fatalf("procesar batch sync intento %d: %v", intento, err)
		}
		syncOrder, err := GetRuntimeOrder(syncID)
		if err != nil || syncOrder == nil {
			t.Fatalf("get sync remoto intento %d: %+v err=%v", intento, syncOrder, err)
		}
		if syncOrder.Estado != "completada" || !strings.Contains(syncOrder.ResultadoJSON, `"observed_remote_error":true`) {
			t.Fatalf("sync remoto intento %d deberia completar con error observado: %+v", intento, syncOrder)
		}
		handle, err = GetRuntimeHandle(handle.ID)
		if err != nil || handle == nil {
			t.Fatalf("get handle intento %d: %+v err=%v", intento, handle, err)
		}
		if !strings.Contains(handle.MetadataJSON, `"remote_sync_failures":`) {
			t.Fatalf("handle sin contador de fallos remoto tras intento %d: %+v", intento, handle)
		}
	}

	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle final: %+v err=%v", handle, err)
	}
	if handle.Estado != "fallido" || !strings.Contains(handle.MetadataJSON, `"remote_sync_failures":2`) || !strings.Contains(handle.MetadataJSON, `"remote_last_error":"adaptador remoto devolvió 503`) {
		t.Fatalf("handle remoto degradado inesperado: %+v", handle)
	}
	runtime, err = GetRuntime(runtime.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime final: %+v err=%v", runtime, err)
	}
	if runtime.LogicalState != "degradado" || runtime.ProcessState != "remote_status_error" {
		t.Fatalf("runtime remoto degradado inesperado: %+v", runtime)
	}
	if got := strings.Join(calls, ","); strings.Count(got, "/status") < 2 {
		t.Fatalf("esperaba al menos dos status remotos, got=%s", got)
	}
}

func TestRuntimeOrderStartRechazaConectorConCircuitoAbierto(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	conectorID, err := UpsertConector(&Conector{
		Slug:       "codex-remote-open",
		Nombre:     "Codex Remote Open",
		Transporte: "api",
		Comando:    "http://127.0.0.1:65535",
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("upsert conector remoto: %v", err)
	}
	cooldown := time.Now().UTC().Add(time.Minute)
	if err := UpsertConectorOperacion(&ConectorOperacion{
		ConectorID:         conectorID,
		EstadoOperativo:    ConectorOperativoCircuitoAbierto,
		Motivo:             "remote_connector_unavailable",
		FallosConsecutivos: 3,
		CooldownUntil:      &cooldown,
	}); err != nil {
		t.Fatalf("upsert conector operacion: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"codex-remote-open"}`,
	})
	if err != nil {
		t.Fatalf("encolar start remoto: %v", err)
	}
	if _, err := ProcesarRuntimeOrdersBatch(); err != nil {
		t.Fatalf("procesar batch start: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil || startOrder == nil {
		t.Fatalf("get start order: %+v err=%v", startOrder, err)
	}
	if startOrder.Estado != "fallida" || !strings.Contains(startOrder.ErrorText, "circuito abierto del conector") {
		t.Fatalf("start con circuito abierto deberia fallar de forma explicita: %+v", startOrder)
	}
}

func TestRuntimeOrderStartLimpiaSesionRemotaSiFallaAltaLocal(t *testing.T) {
	tmp := prepararDBTemporal(t)
	calls := make([]string, 0, 4)
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/launch":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"external_session_id": "sess-remote-cleanup",
				"handle_ref":          "remote-handle-cleanup",
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}
	}))
	defer srv.Close()

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "codex-remote-cleanup",
		Nombre:     "Codex Remote Cleanup",
		Transporte: "api",
		Comando:    srv.URL,
		MetadataJSON: `{
			"launch_path": "/launch",
			"stop_path": "/stop"
		}`,
		Activo: true,
	}); err != nil {
		t.Fatalf("upsert conector remoto: %v", err)
	}
	if _, err := DB.Exec(`
		CREATE TRIGGER fail_runtime_insert
		BEFORE INSERT ON runtime_instances
		BEGIN
			SELECT RAISE(FAIL, 'runtime insert forced failure');
		END;
	`); err != nil {
		t.Fatalf("crear trigger: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"codex-remote-cleanup"}`,
	})
	if err != nil {
		t.Fatalf("encolar start remoto cleanup: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil || startOrder == nil {
		t.Fatalf("get start order: %+v err=%v", startOrder, err)
	}
	if err := ejecutarRuntimeOrderStart(startOrder); err == nil || !strings.Contains(err.Error(), "runtime insert forced failure") {
		t.Fatalf("esperaba error de alta local, got=%v", err)
	}
	got := strings.Join(calls, ",")
	if !strings.Contains(got, "/launch") || !strings.Contains(got, "/stop") {
		t.Fatalf("cleanup remoto inesperado: %s", got)
	}
}

func TestRuntimeOrderStartIntegraBootstrapDeHandoffMailboxYCheckpoint(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:       "cat-cli",
		Nombre:     "Cat CLI",
		Transporte: "cli",
		Comando:    "cat",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador", "sesion-previa"),
		Herramienta:        "codex-cli",
		ResumenContinuidad: "continuidad previa",
		ResumePayloadJSON:  `{"previo":true}`,
	}); err != nil {
		t.Fatalf("iniciar sesion previa: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:     "Cerrar relevo",
		Modulo:     "orquestador",
		Prioridad:  PrioridadAlta,
		CreadoPor:  "alberto",
		ProyectoID: &proyectoID,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if _, err := DB.Exec(`UPDATE tareas SET estado='asignada' WHERE id=?`, tareaID); err != nil {
		t.Fatalf("marcar tarea asignada: %v", err)
	}

	handoffPayload, err := json.Marshal(HandoffPayload{
		AgenteOrigen:       "Codex0",
		AgenteDestino:      "Codex1",
		TareaID:            &tareaID,
		Motivo:             "traspaso",
		ResumenContinuidad: "handoff listo",
		ExternalSessionID:  "sess-handoff",
	})
	if err != nil {
		t.Fatalf("marshal handoff: %v", err)
	}
	handoffID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "handoff",
		PayloadJSON: string(handoffPayload),
	})
	if err != nil {
		t.Fatalf("encolar handoff: %v", err)
	}
	msgID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "Coordinador",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "handoff_note",
		PayloadJSON: `{"texto":"revisa el checkpoint"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}
	if _, err := CrearRuntimeCheckpoint(&RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		CheckpointKind: "handoff_prepare",
		Resumen:        "checkpoint reciente",
		Branch:         "feature/bootstrap",
		CWD:            filepath.Join(tmp, "orquestador", "checkpoint"),
		PayloadJSON:    `{"archivos":["a.go","b.go"]}`,
		ResumeStrategy: "resumen_y_payload",
		Source:         "test:bootstrap-start",
	}); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO worktrees (proyecto_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?)`,
		proyectoID,
		"Codex1",
		"wt-codex1",
		filepath.Join(tmp, "orquestador", ".orquesta-worktrees", "wt-codex1"),
		"feature/wt-codex1",
		"HEAD",
		"activa",
		"continuidad",
	); err != nil {
		t.Fatalf("crear worktree: %v", err)
	}
	if _, err := CrearPropuesta(&Propuesta{
		Codigo:       "OP-902",
		Titulo:       "Definir integracion",
		Descripcion:  "Pendiente de cierre",
		ProyectoID:   &proyectoID,
		Tipo:         "arquitectura",
		PropuestoPor: "alberto",
		Distribuidor: "orquesta",
	}); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"cat-cli"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start: %v", err)
	}
	if err := ejecutarRuntimeOrderStart(startOrder); err != nil {
		t.Fatalf("ejecutar start: %v", err)
	}

	sesion, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion activa: %+v err=%v", sesion, err)
	}
	if sesion.ExternalSessionID != "sess-handoff" {
		t.Fatalf("external_session_id inesperado: %s", sesion.ExternalSessionID)
	}
	if !strings.Contains(sesion.ResumePayloadJSON, `"mailbox"`) || !strings.Contains(sesion.ResumePayloadJSON, `"checkpoint"`) {
		t.Fatalf("resume payload sin bootstrap: %s", sesion.ResumePayloadJSON)
	}
	if !strings.Contains(sesion.ResumePayloadJSON, `"project_context"`) || !strings.Contains(sesion.ResumePayloadJSON, "feature/wt-codex1") || !strings.Contains(sesion.ResumePayloadJSON, "OP-902") {
		t.Fatalf("resume payload sin mapa operativo: %s", sesion.ResumePayloadJSON)
	}
	if !strings.Contains(sesion.ResumenContinuidad, "handoff listo") || !strings.Contains(sesion.ResumenContinuidad, "Mailbox: 1 mensaje(s) inyectados") {
		t.Fatalf("resumen continuidad inesperado: %s", sesion.ResumenContinuidad)
	}
	if !strings.Contains(sesion.ResumenContinuidad, "Worktree activa en feature/wt-codex1") || !strings.Contains(sesion.ResumenContinuidad, "1 propuesta(s) abiertas") {
		t.Fatalf("resumen continuidad sin mapa operativo: %s", sesion.ResumenContinuidad)
	}
	if sesion.Branch != "feature/bootstrap" {
		t.Fatalf("branch inesperada: %s", sesion.Branch)
	}

	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if strings.Contains(handle.MetadataJSON, `"continuity_prompt"`) {
		t.Fatalf("metadata de handle no deberia conservar continuity_prompt crudo: %s", handle.MetadataJSON)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}

	handoffOrder, err := GetRuntimeOrder(handoffID)
	if err != nil {
		t.Fatalf("get handoff: %v", err)
	}
	if handoffOrder.Estado != "ejecutando" {
		t.Fatalf("handoff deberia esperar ack de evidencia: %+v", handoffOrder)
	}
	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != TareaAsignada {
		t.Fatalf("la tarea no deberia pasar a progreso antes del ack real: %s", tarea.Estado)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex1" {
		t.Fatalf("agente de tarea inesperado: %+v", tarea.Agente)
	}
	if strings.Contains(tarea.Notas, "handoff completado por Codex1") {
		t.Fatalf("no deberia registrar evidencia de handoff antes del ack: %s", tarea.Notas)
	}
	msgsConsumidos, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   strPtrTest("Codex1"),
		ProyectoID: &proyectoID,
		Estado:     strPtrTest("pendiente"),
	})
	if err != nil {
		t.Fatalf("mailbox pendiente: %v", err)
	}
	if len(msgsConsumidos) != 1 || msgsConsumidos[0].ID != msgID {
		t.Fatalf("mailbox pendiente inesperado: %+v", msgsConsumidos)
	}
	if err := AckBootstrapRuntimeLeaseByEvidence(handle, runtime, "test_runtime_output"); err != nil {
		t.Fatalf("ack bootstrap por evidencia: %v", err)
	}
	handoffOrder, err = GetRuntimeOrder(handoffID)
	if err != nil {
		t.Fatalf("get handoff tras ack: %v", err)
	}
	if handoffOrder.Estado != "completada" {
		t.Fatalf("handoff no completada tras ack: %+v", handoffOrder)
	}
	tarea, err = GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea tras ack: %v", err)
	}
	if tarea.Estado != TareaEnProgreso {
		t.Fatalf("la tarea deberia quedar en progreso tras el ack real: %s", tarea.Estado)
	}
	if !strings.Contains(tarea.Notas, "handoff completado por Codex1") {
		t.Fatalf("notas sin evidencia de handoff completado tras ack: %s", tarea.Notas)
	}
	msgsConsumidos, err = ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   strPtrTest("Codex1"),
		ProyectoID: &proyectoID,
		Estado:     strPtrTest("consumido"),
	})
	if err != nil {
		t.Fatalf("mailbox consumido tras ack: %v", err)
	}
	if len(msgsConsumidos) != 1 || msgsConsumidos[0].ID != msgID {
		t.Fatalf("mailbox consumido inesperado tras ack: %+v", msgsConsumidos)
	}
}

func TestRuntimeOrderStartEmbebeLaunchPromptMultilineaCuandoConectorLoDeclara(t *testing.T) {
	tmp := prepararDBTemporal(t)
	argLog := filepath.Join(tmp, "launch-argv.log")
	promptLog := filepath.Join(tmp, "launch-prompt.log")
	launcher := filepath.Join(tmp, "fake-launcher.sh")
	script := "#!/usr/bin/env bash\nset -eu\nprintf '%s\\n' \"$@\" > " + strconv.Quote(argLog) + "\nprintf '%s' \"${!#}\" > " + strconv.Quote(promptLog) + "\nsleep 30\n"
	if err := os.WriteFile(launcher, []byte(script), 0o755); err != nil {
		t.Fatalf("write launcher: %v", err)
	}

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := UpsertConector(&Conector{
		Slug:         "codex-launcher",
		Nombre:       "Codex Launcher",
		Transporte:   "cli",
		Comando:      launcher,
		MetadataJSON: `{"launch_prompt_positional":true}`,
		Activo:       true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:     "Implementar bootstrap real",
		Modulo:     "orquestador",
		Prioridad:  PrioridadAlta,
		CreadoPor:  "alberto",
		ProyectoID: &proyectoID,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if _, err := DB.Exec(`UPDATE tareas SET estado='asignada' WHERE id=?`, tareaID); err != nil {
		t.Fatalf("marcar tarea asignada: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"codex-launcher"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order: %v", err)
	}
	if err := ejecutarRuntimeOrderStart(startOrder); err != nil {
		t.Fatalf("ejecutar start: %v", err)
	}

	sesion, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion activa: %+v err=%v", sesion, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil || runtime.PID == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	t.Cleanup(func() {
		_, _, _ = controlruntime.DetenerProceso(controlruntime.ObjetivoProceso{PID: runtime.PID})
	})

	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if strings.Contains(handle.MetadataJSON, `"bootstrap_prompt"`) || !strings.Contains(handle.MetadataJSON, `"launch_prompt_embedded":true`) {
		t.Fatalf("metadata de handle no deberia conservar bootstrap_prompt crudo: %s", handle.MetadataJSON)
	}

	deadline := time.Now().Add(2 * time.Second)
	var (
		argvData   []byte
		promptData []byte
	)
	for {
		promptData, err = os.ReadFile(promptLog)
		if err == nil && strings.Contains(string(promptData), "Bootstrap de Orquesta para Codex1.") {
			break
		}
		if time.Now().After(deadline) {
			if err != nil {
				t.Fatalf("leer prompt log: %v", err)
			}
			argvData, _ = os.ReadFile(argLog)
			t.Fatalf("el launcher no recibió el bootstrap esperado. prompt=%q argv=%s", string(promptData), string(argvData))
		}
		time.Sleep(10 * time.Millisecond)
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
	entries, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: &agente, Limit: 20})
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

func TestGetRuntimeHandleBySesionIDCompactaMetadataLegacy(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-legacy",
		ResumenContinuidad: "continuidad inicial",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	legacyMeta := `{"bootstrap_prompt":"Bootstrap de Orquesta para Codex1.\nSegunda linea.","continuity_prompt":"Retoma la tarea 416 desde el checkpoint."}`
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE sesion_id=?`, legacyMeta, sesion.ID); err != nil {
		t.Fatalf("inyectar metadata legacy: %v", err)
	}

	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if strings.Contains(handle.MetadataJSON, `"bootstrap_prompt"`) || strings.Contains(handle.MetadataJSON, `"continuity_prompt"`) {
		t.Fatalf("metadata legacy no compactada: %s", handle.MetadataJSON)
	}
	if !strings.Contains(handle.MetadataJSON, `"bootstrap_prompt_summary":"Bootstrap de Orquesta para Codex1."`) {
		t.Fatalf("falta resumen bootstrap: %s", handle.MetadataJSON)
	}
	if !strings.Contains(handle.MetadataJSON, `"continuity_prompt_summary":"Retoma la tarea 416 desde el checkpoint."`) {
		t.Fatalf("falta resumen continuity: %s", handle.MetadataJSON)
	}
	if !strings.Contains(handle.MetadataJSON, `"bootstrap_prompt_len":49`) {
		t.Fatalf("falta longitud bootstrap: %s", handle.MetadataJSON)
	}

	var persisted string
	if err := DB.QueryRow(`SELECT metadata_json FROM runtime_handles WHERE id = ?`, handle.ID).Scan(&persisted); err != nil {
		t.Fatalf("leer metadata persistida: %v", err)
	}
	if strings.Contains(persisted, `"bootstrap_prompt"`) || strings.Contains(persisted, `"continuity_prompt"`) {
		t.Fatalf("metadata persistida sigue legacy: %s", persisted)
	}
}

func TestListarRuntimeHandlesCompactaMetadataLegacy(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-lista",
		ResumenContinuidad: "continuidad",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	legacyMeta := `{"bootstrap_prompt":"Bootstrap legacy para listado.","launch_prompt_embedded":true,"rendered_command":"'/home/alberto/Trabajo/codex-perfiles/bin/codex-perfil' 'Codex1' '--model' 'gpt-5.4' 'Bootstrap de Orquesta para Codex1.\nLinea 2'","wrapped_command":"script -q -e -f -c '/home/alberto/Trabajo/codex-perfiles/bin/codex-perfil Codex1 Bootstrap de Orquesta para Codex1.' '/tmp/pty.log'"}`
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE sesion_id=?`, legacyMeta, sesion.ID); err != nil {
		t.Fatalf("inyectar metadata legacy: %v", err)
	}

	agente := "Codex1"
	handles, err := ListarRuntimeHandles(&agente)
	if err != nil {
		t.Fatalf("listar handles: %v", err)
	}
	if len(handles) == 0 {
		t.Fatalf("sin handles listados")
	}
	handle := handles[0]
	if strings.Contains(handle.MetadataJSON, `"bootstrap_prompt"`) {
		t.Fatalf("metadata listada sigue legacy: %s", handle.MetadataJSON)
	}
	if !strings.Contains(handle.MetadataJSON, `"bootstrap_prompt_summary":"Bootstrap legacy para listado."`) {
		t.Fatalf("falta resumen bootstrap en listado: %s", handle.MetadataJSON)
	}
	if !strings.Contains(handle.MetadataJSON, `"launch_prompt_embedded":true`) {
		t.Fatalf("se perdió launch_prompt_embedded: %s", handle.MetadataJSON)
	}
	if strings.Contains(handle.MetadataJSON, "Bootstrap de Orquesta para Codex1.") {
		t.Fatalf("comando crudo no deberia persistirse en listado: %s", handle.MetadataJSON)
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if got := stringFromMap(meta, "rendered_command", ""); got != "'/home/alberto/Trabajo/codex-perfiles/bin/codex-perfil' 'Codex1'" {
		t.Fatalf("rendered_command no compactado: %q meta=%s", got, handle.MetadataJSON)
	}
	if got := stringFromMap(meta, "wrapped_command", ""); got != "'script' '-q' '-e' '-f' '<omitted>' '/tmp/pty.log'" {
		t.Fatalf("wrapped_command no compactado: %q meta=%s", got, handle.MetadataJSON)
	}
	if got, ok := meta["rendered_command_len"].(float64); !ok || int(got) <= len([]rune(stringFromMap(meta, "rendered_command", ""))) {
		t.Fatalf("falta rendered_command_len util: %+v meta=%s", meta["rendered_command_len"], handle.MetadataJSON)
	}
}

func strPtrTest(v string) *string { return &v }

func TestProcesarRuntimeOrdersBatchCompletaNudgeEnMailbox(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		Tipo:        "nudge",
		PayloadJSON: `{"from_agente":"Codex1","to_agente":"Codex2","kind":"nudge","texto":"retoma la tarea y desbloquea el siguiente paso"}`,
	})
	if err != nil {
		t.Fatalf("encolar nudge: %v", err)
	}

	n, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("processed inesperado: %d", n)
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if order == nil || order.Estado != "completada" {
		t.Fatalf("order no completada: %+v", order)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(order.ResultadoJSON), &result); err != nil {
		t.Fatalf("parse resultado nudge: %v", err)
	}
	mailboxID, ok := result["mailbox_id"].(float64)
	if !ok || int64(mailboxID) <= 0 {
		t.Fatalf("resultado nudge sin mailbox_id: %+v", result)
	}

	toAgente := "Codex2"
	estado := "pendiente"
	mailbox, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente: &toAgente,
		Estado:   &estado,
	})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(mailbox) != 1 {
		t.Fatalf("mailbox inesperado: %+v", mailbox)
	}
	if mailbox[0].RuntimeOrderID == nil || *mailbox[0].RuntimeOrderID != orderID {
		t.Fatalf("runtime_order_id inesperado en mailbox: %+v", mailbox[0])
	}
	if mailbox[0].FromAgente != "Codex1" || mailbox[0].Kind != "nudge" {
		t.Fatalf("mailbox nudge inesperado: %+v", mailbox[0])
	}
}

func TestEjecutarRuntimeOrderNudgeReutilizaMailboxExistente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		Tipo:        "nudge",
		PayloadJSON: `{"from_agente":"Codex1","to_agente":"Codex2","texto":"reintento nudge"}`,
	})
	if err != nil {
		t.Fatalf("encolar nudge: %v", err)
	}

	runtimeOrderID := orderID
	existenteID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:     "Codex1",
		ToAgente:       "Codex2",
		ProyectoID:     &proyectoID,
		RuntimeOrderID: &runtimeOrderID,
		Kind:           "nudge",
		PayloadJSON:    `{"from_agente":"Codex1","to_agente":"Codex2","texto":"reintento nudge"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox existente: %v", err)
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if err := ejecutarRuntimeOrderNudge(order); err != nil {
		t.Fatalf("ejecutar nudge: %v", err)
	}

	reutilizado, err := GetRuntimeMailboxByRuntimeOrderID(orderID)
	if err != nil {
		t.Fatalf("get mailbox por runtime order: %v", err)
	}
	if reutilizado == nil || reutilizado.ID != existenteID {
		t.Fatalf("mailbox reutilizado incorrecto: %+v", reutilizado)
	}

	order, err = GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if order.Estado != "completada" {
		t.Fatalf("order no completada: %+v", order)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(order.ResultadoJSON), &result); err != nil {
		t.Fatalf("parse resultado nudge: %v", err)
	}
	if reused, _ := result["reused"].(bool); !reused {
		t.Fatalf("resultado nudge deberia marcar reused: %+v", result)
	}
	mailboxID, ok := result["mailbox_id"].(float64)
	if !ok || int64(mailboxID) != existenteID {
		t.Fatalf("resultado nudge no reutiliza mailbox existente: %+v", result)
	}
}

func TestProcesarRuntimeOrdersBatchCompletaDiscordiaEnMailbox(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("alberto", "admin"); err != nil {
		t.Fatalf("registrar alberto: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "alberto",
		ProyectoID:  &proyectoID,
		Tipo:        "discordia",
		PayloadJSON: `{"from_agente":"Codex1","to_agente":"alberto","kind":"discordia","contra_agente":"Codex2","motivo":"desacuerdo tecnico"}`,
	})
	if err != nil {
		t.Fatalf("encolar discordia: %v", err)
	}

	n, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("processed inesperado: %d", n)
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if order == nil || order.Estado != "completada" {
		t.Fatalf("order no completada: %+v", order)
	}

	toAgente := "alberto"
	estado := "pendiente"
	mailbox, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente: &toAgente,
		Estado:   &estado,
	})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(mailbox) != 1 {
		t.Fatalf("mailbox inesperado: %+v", mailbox)
	}
	if mailbox[0].RuntimeOrderID == nil || *mailbox[0].RuntimeOrderID != orderID {
		t.Fatalf("runtime_order_id inesperado en mailbox: %+v", mailbox[0])
	}
	if mailbox[0].FromAgente != "Codex1" || mailbox[0].Kind != "discordia" {
		t.Fatalf("mailbox discordia inesperado: %+v", mailbox[0])
	}
}

func TestReconciliarRuntimeOrdersStaleRecuperaBasicasYExpiraNoSoportadas(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := DB.Exec(`UPDATE config SET valor = '1' WHERE clave = 'runtime_order_stale_seconds'`); err != nil {
		t.Fatalf("config stale seconds: %v", err)
	}

	basicID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar basica: %v", err)
	}
	customID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "custom_dispatch",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar custom: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(basicID, "ejecutando", `{}`, ""); err != nil {
		t.Fatalf("marcar ejecutando basica: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(customID, "tomada", `{}`, ""); err != nil {
		t.Fatalf("marcar tomada custom: %v", err)
	}
	if _, err := DB.Exec(`
		UPDATE runtime_orders
		SET started_at = ?, updated_at = ?, lease_expires_at = ?
		WHERE id IN (?, ?)`,
		time.Now().UTC().Add(-10*time.Minute),
		time.Now().UTC().Add(-10*time.Minute),
		time.Now().UTC().Add(-10*time.Minute),
		basicID, customID,
	); err != nil {
		t.Fatalf("envejecer orders: %v", err)
	}

	n, err := ReconciliarRuntimeOrdersStale()
	if err != nil {
		t.Fatalf("reconciliar runtime orders stale: %v", err)
	}
	if n != 2 {
		t.Fatalf("esperaba recuperar/expirar 2 ordenes, got=%d", n)
	}

	basicOrder, err := GetRuntimeOrder(basicID)
	if err != nil {
		t.Fatalf("get basic order: %v", err)
	}
	if basicOrder.Estado != "pendiente" || basicOrder.StartedAt != nil {
		t.Fatalf("orden basica no reencolada: %+v", basicOrder)
	}
	if basicOrder.LeaseExpiresAt != nil || strings.TrimSpace(basicOrder.LeaseToken) != "" || strings.TrimSpace(basicOrder.ClaimedBy) != "" {
		t.Fatalf("orden basica sin lease limpia: %+v", basicOrder)
	}

	customOrder, err := GetRuntimeOrder(customID)
	if err != nil {
		t.Fatalf("get custom order: %v", err)
	}
	if customOrder.Estado != "expirada" || customOrder.FinishedAt == nil {
		t.Fatalf("orden no soportada no expirada: %+v", customOrder)
	}
	if customOrder.LeaseExpiresAt != nil || strings.TrimSpace(customOrder.LeaseToken) != "" || strings.TrimSpace(customOrder.ClaimedBy) != "" {
		t.Fatalf("orden expirada conserva lease: %+v", customOrder)
	}
}

func TestReconciliarRuntimeOrdersStaleRespetaCutoffConfigurado(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := DB.Exec(`UPDATE config SET valor = '60' WHERE clave = 'runtime_order_stale_seconds'`); err != nil {
		t.Fatalf("config stale seconds: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar order: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(orderID, "ejecutando", `{}`, ""); err != nil {
		t.Fatalf("marcar ejecutando: %v", err)
	}
	recent := time.Now().UTC().Add(-10 * time.Second)
	leaseFuture := time.Now().UTC().Add(50 * time.Second)
	if _, err := DB.Exec(`
		UPDATE runtime_orders
		SET started_at = ?, updated_at = ?, lease_expires_at = ?
		WHERE id = ?`,
		recent, recent, leaseFuture, orderID,
	); err != nil {
		t.Fatalf("rejuvenecer order: %v", err)
	}

	n, err := ReconciliarRuntimeOrdersStale()
	if err != nil {
		t.Fatalf("reconciliar runtime orders stale: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia reconciliar una orden aun dentro del cutoff, got=%d", n)
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if order.Estado != "ejecutando" {
		t.Fatalf("la orden deberia seguir ejecutando: %+v", order)
	}
}

func TestClaimRuntimeOrderSendInstructionUsaLeaseCorta(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := DB.Exec(`UPDATE config SET valor='120' WHERE clave='runtime_order_lease_seconds'`); err != nil {
		t.Fatalf("config runtime_order_lease_seconds: %v", err)
	}
	if _, err := DB.Exec(`UPDATE config SET valor='35' WHERE clave='runtime_send_instruction_lease_seconds'`); err != nil {
		t.Fatalf("config runtime_send_instruction_lease_seconds: %v", err)
	}

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"hola"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	syncID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar sync_status: %v", err)
	}

	before := time.Now().UTC()
	sendOrder, err := claimRuntimeOrderByID(sendID)
	if err != nil {
		t.Fatalf("claim send_instruction: %v", err)
	}
	syncOrder, err := claimRuntimeOrderByID(syncID)
	if err != nil {
		t.Fatalf("claim sync_status: %v", err)
	}
	if sendOrder == nil || sendOrder.LeaseExpiresAt == nil {
		t.Fatalf("send_instruction sin lease: %+v", sendOrder)
	}
	if syncOrder == nil || syncOrder.LeaseExpiresAt == nil {
		t.Fatalf("sync_status sin lease: %+v", syncOrder)
	}

	sendLease := sendOrder.LeaseExpiresAt.Sub(before)
	syncLease := syncOrder.LeaseExpiresAt.Sub(before)
	if sendLease > 45*time.Second {
		t.Fatalf("lease send_instruction demasiado larga: %s", sendLease)
	}
	if syncLease < 110*time.Second {
		t.Fatalf("lease sync_status demasiado corta: %s", syncLease)
	}
	if !(sendLease < syncLease) {
		t.Fatalf("send_instruction deberia reclamar lease mas corta: send=%s sync=%s", sendLease, syncLease)
	}
}

func TestReconciliarRuntimeOrdersStaleRecuperaLeaseExpiradaSinEsperarOtroCutoff(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := DB.Exec(`UPDATE config SET valor = '60' WHERE clave = 'runtime_order_stale_seconds'`); err != nil {
		t.Fatalf("config stale seconds: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar order: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(orderID, "ejecutando", `{}`, ""); err != nil {
		t.Fatalf("marcar ejecutando: %v", err)
	}
	expiredLease := time.Now().UTC().Add(-10 * time.Second)
	if _, err := DB.Exec(`
		UPDATE runtime_orders
		SET started_at = ?, updated_at = ?, lease_expires_at = ?
		WHERE id = ?`,
		time.Now().UTC(), time.Now().UTC(), expiredLease, orderID,
	); err != nil {
		t.Fatalf("expirar lease: %v", err)
	}

	n, err := ReconciliarRuntimeOrdersStale()
	if err != nil {
		t.Fatalf("reconciliar runtime orders stale: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia recuperar la orden por lease expirada, got=%d", n)
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if order.Estado != "pendiente" {
		t.Fatalf("orden no reencolada tras lease expirada: %+v", order)
	}
}

func TestReconciliarRuntimeOrderStaleNoPisaOrdenYaCompletada(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar order: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(orderID, "ejecutando", `{}`, ""); err != nil {
		t.Fatalf("marcar ejecutando: %v", err)
	}
	if _, err := DB.Exec(`
		UPDATE runtime_orders
		SET started_at = ?, updated_at = ?
		WHERE id = ?`,
		time.Now().UTC().Add(-10*time.Minute),
		time.Now().UTC().Add(-10*time.Minute),
		orderID,
	); err != nil {
		t.Fatalf("envejecer order: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(orderID, "completada", `{"ok":true}`, ""); err != nil {
		t.Fatalf("marcar completada: %v", err)
	}

	applied, err := reconciliarRuntimeOrderStale(orderID, "sync_status", time.Now().UTC(), time.Now().UTC().Add(-time.Minute))
	if err != nil {
		t.Fatalf("reconciliar order stale: %v", err)
	}
	if applied {
		t.Fatalf("la reconciliacion stale no deberia aplicar sobre una orden completada")
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if order.Estado != "completada" {
		t.Fatalf("estado inesperado tras reconciliacion: %+v", order)
	}
}

func TestReconciliarRuntimeOrderStaleNoPisaOrdenReclamadaDeNuevo(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "custom_dispatch",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar order: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(orderID, "tomada", `{}`, ""); err != nil {
		t.Fatalf("marcar tomada: %v", err)
	}
	if _, err := DB.Exec(`
		UPDATE runtime_orders
		SET started_at = ?, updated_at = ?
		WHERE id = ?`,
		time.Now().UTC().Add(-10*time.Minute),
		time.Now().UTC().Add(-10*time.Minute),
		orderID,
	); err != nil {
		t.Fatalf("envejecer order: %v", err)
	}
	if _, err := DB.Exec(`
		UPDATE runtime_orders
		SET started_at = ?, updated_at = ?
		WHERE id = ?`,
		time.Now().UTC(),
		time.Now().UTC(),
		orderID,
	); err != nil {
		t.Fatalf("refrescar timestamps: %v", err)
	}

	applied, err := reconciliarRuntimeOrderStale(orderID, "custom_dispatch", time.Now().UTC(), time.Now().UTC().Add(-time.Minute))
	if err != nil {
		t.Fatalf("reconciliar order stale: %v", err)
	}
	if applied {
		t.Fatalf("la reconciliacion stale no deberia aplicar si la orden ya fue reclamada de nuevo")
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if order.Estado != "tomada" || order.FinishedAt != nil {
		t.Fatalf("orden modificada indebidamente por reconciliacion stale: %+v", order)
	}
}

func TestProcesarRuntimeOrdersBatchDejaPendienteOrdenNoSoportada(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-a",
		ResumenContinuidad: "seguir",
		Branch:             "main",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	customID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "custom_dispatch",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar custom: %v", err)
	}
	syncID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar sync: %v", err)
	}

	n, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba procesar solo la orden soportada, got=%d", n)
	}

	customOrder, err := GetRuntimeOrder(customID)
	if err != nil {
		t.Fatalf("get custom order: %v", err)
	}
	if customOrder.Estado != "pendiente" {
		t.Fatalf("la orden no soportada no deberia tocarse: %+v", customOrder)
	}

	syncOrder, err := GetRuntimeOrder(syncID)
	if err != nil {
		t.Fatalf("get sync order: %v", err)
	}
	if syncOrder.Estado != "completada" {
		t.Fatalf("la orden soportada deberia completarse: %+v", syncOrder)
	}
}

func TestProcesarRuntimeOrdersBatchResuelveRuntimePorProyecto(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-a",
		Nombre:  "Orquestador A",
		RutaAbs: filepath.Join(tmp, "orquestador-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-b",
		Nombre:  "Orquestador B",
		RutaAbs: filepath.Join(tmp, "orquestador-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}

	sesionA, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoA,
		CWD:                filepath.Join(tmp, "orquestador-a"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-a",
		ResumenContinuidad: "continuidad A",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion A: %v", err)
	}
	runtimeA, err := GetRuntimeBySesionID(sesionA.ID)
	if err != nil || runtimeA == nil {
		t.Fatalf("runtime A no disponible: runtime=%+v err=%v", runtimeA, err)
	}

	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoB,
		CWD:                filepath.Join(tmp, "orquestador-b"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-b",
		ResumenContinuidad: "continuidad B",
		Branch:             "main",
	}); err != nil {
		t.Fatalf("iniciar sesion B: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoA,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar sync: %v", err)
	}

	n, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 orden procesada, got=%d", n)
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(order.ResultadoJSON), &result); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	gotRuntimeID, ok := result["runtime_id"].(float64)
	if !ok {
		t.Fatalf("runtime_id ausente en resultado: %+v", result)
	}
	if int64(gotRuntimeID) != runtimeA.ID {
		t.Fatalf("runtime incorrecto; esperado proyecto A=%d, got=%v", runtimeA.ID, gotRuntimeID)
	}
}
