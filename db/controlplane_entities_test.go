package db

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/internal/controlruntime"
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
		SET started_at = ?, updated_at = ?
		WHERE id IN (?, ?)`,
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

	otherOrder, err := GetRuntimeOrder(otherID)
	if err != nil {
		t.Fatalf("get handoff order: %v", err)
	}
	if otherOrder.Estado != "pendiente" || otherOrder.StartedAt != nil {
		t.Fatalf("handoff order no reencolada: %+v", otherOrder)
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

func TestRuntimeHandlePermiteSendInputInteractivoRespetaCapacidadesYFallbackCodex(t *testing.T) {
	if RuntimeHandlePermiteSendInputInteractivo(nil) != true {
		t.Fatal("nil handle deberia permitir por defecto")
	}
	if RuntimeHandlePermiteSendInputInteractivo(&RuntimeHandle{
		CapabilitiesJSON: `{"can_send_input":false}`,
	}) {
		t.Fatal("can_send_input=false deberia desactivar input interactivo")
	}
	if RuntimeHandlePermiteSendInputInteractivo(&RuntimeHandle{
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"/home/alberto/Trabajo/codex-perfiles/bin/codex-perfil Codex2"}`,
	}) {
		t.Fatal("codex local via PTY deberia desactivar input interactivo")
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
	meta["rendered_command"] = "/home/alberto/Trabajo/codex-perfiles/bin/codex-perfil Codex1"
	meta["can_send_input"] = false
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

func TestRuntimeOrderStartRemotoPersisteSesionYHandleSinPID(t *testing.T) {
	tmp := prepararDBTemporal(t)
	calls := make([]string, 0, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestRuntimeOrderPauseCheckpointeaYSesionQuedaPausada(t *testing.T) {
	tmp := prepararDBTemporal(t)
	calls := make([]string, 0, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	if !strings.Contains(handle.MetadataJSON, `"continuity_prompt"`) || !strings.Contains(handle.MetadataJSON, "handoff listo") {
		t.Fatalf("metadata de handle sin continuidad: %s", handle.MetadataJSON)
	}

	handoffOrder, err := GetRuntimeOrder(handoffID)
	if err != nil {
		t.Fatalf("get handoff: %v", err)
	}
	if handoffOrder.Estado != "completada" {
		t.Fatalf("handoff no completada: %+v", handoffOrder)
	}
	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != TareaEnProgreso {
		t.Fatalf("la tarea deberia quedar en progreso tras la reanudacion real: %s", tarea.Estado)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex1" {
		t.Fatalf("agente de tarea inesperado: %+v", tarea.Agente)
	}
	if !strings.Contains(tarea.Notas, "handoff completado por Codex1") {
		t.Fatalf("notas sin evidencia de handoff completado: %s", tarea.Notas)
	}
	msgsConsumidos, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   strPtrTest("Codex1"),
		ProyectoID: &proyectoID,
		Estado:     strPtrTest("consumido"),
	})
	if err != nil {
		t.Fatalf("mailbox consumido: %v", err)
	}
	if len(msgsConsumidos) != 1 || msgsConsumidos[0].ID != msgID {
		t.Fatalf("mailbox consumido inesperado: %+v", msgsConsumidos)
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
	if !strings.Contains(handle.MetadataJSON, `"bootstrap_prompt"`) || !strings.Contains(handle.MetadataJSON, `"launch_prompt_embedded":true`) {
		t.Fatalf("metadata de handle sin bootstrap embebido: %s", handle.MetadataJSON)
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
		SET started_at = ?, updated_at = ?
		WHERE id IN (?, ?)`,
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

	customOrder, err := GetRuntimeOrder(customID)
	if err != nil {
		t.Fatalf("get custom order: %v", err)
	}
	if customOrder.Estado != "expirada" || customOrder.FinishedAt == nil {
		t.Fatalf("orden no soportada no expirada: %+v", customOrder)
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
