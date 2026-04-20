package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/coordinacion"
	"orquesta/internal/controlruntime"
	"orquesta/runtimeagente"
)

func enableLegacyPTYLocalRuntimeForTest(t *testing.T) {
	t.Helper()
	t.Setenv("ORQUESTA_ALLOW_LEGACY_PTY", "1")
	t.Setenv("ORQUESTA_TERMINAL_BACKEND", "pty")
}

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

	pid := int64(os.Getpid())
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
	if handle.HandleRef != strconv.FormatInt(pid, 10) {
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

func TestUpsertRuntimeHandleDesdeSesionPreservaTMUXCanonico(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexTMUX", "programador"); err != nil {
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
		Agente:      "CodexTMUX",
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
		t.Fatalf("get handle inicial: %v %+v", err, handle)
	}
	if got := strings.TrimSpace(handle.HandleKind); got != "process" {
		t.Fatalf("precondicion handle_kind inesperada: %q", got)
	}

	arranque := &controlruntime.ProcesoArrancado{
		PID:        int(pid),
		HandleKind: "session",
		HandleRef:  "orq-codextmux-1/%3",
		MetadataJSON: `{"driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codextmux-1","tmux_pane_id":"%3","working_dir":"` +
			filepath.Join(tmp, "orquestador") + `"}`,
		CapabilitiesJSON: `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`,
	}
	if err := actualizarHandleRuntimeArranque(handle.ID, arranque, nil, nil, runtimeagente.ResumeContext{}); err != nil {
		t.Fatalf("actualizar arranque tmux: %v", err)
	}

	heartbeat := true
	if err := GuardarSesionActiva("CodexTMUX", &proyectoID, SesionUpdate{Heartbeat: heartbeat}); err != nil {
		t.Fatalf("guardar sesion: %v", err)
	}

	handle, err = GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle final: %v %+v", err, handle)
	}
	if got := strings.TrimSpace(handle.Transporte); got != "tmux" {
		t.Fatalf("transporte tmux perdido tras upsert de sesion: %q", got)
	}
	if got := strings.TrimSpace(handle.HandleKind); got != "session" {
		t.Fatalf("handle_kind tmux perdido tras upsert de sesion: %q", got)
	}
	if got := strings.TrimSpace(handle.HandleRef); got != "orq-codextmux-1/%3" {
		t.Fatalf("handle_ref tmux perdido tras upsert de sesion: %q", got)
	}
}

func TestReconciliarMetadataSesionActualizaWorkingDirDesdeCWD(t *testing.T) {
	resultado := reconciliarMetadataSesion(
		`{"working_dir":"/tmp/stale/orquestador","driver":"tmux_cli_session"}`,
		`{"cwd":"/srv/orquesta","branch":"main"}`,
	)
	meta := mapFromJSON(resultado)
	if got := strings.TrimSpace(stringFromMap(meta, "working_dir", "")); got != "/srv/orquesta" {
		t.Fatalf("working_dir no sincronizado desde cwd: %q meta=%s", got, resultado)
	}
	if got := strings.TrimSpace(stringFromMap(meta, "cwd", "")); got != "/srv/orquesta" {
		t.Fatalf("cwd no persistido: %q meta=%s", got, resultado)
	}
}

func TestSincronizarRuntimeHandleWorkingDirPriorizaRuntimeCWDSobreMetadataStale(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaProyecto := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(filepath.Join(rutaProyecto, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir git: %v", err)
	}
	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaProyecto,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex3",
		ProyectoID:   &proyectoID,
		Provider:     "openai",
		Connector:    "codex-cli",
		LogicalState: "activo",
		ProcessState: "running",
		CWD:          rutaProyecto,
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		"Codex3", proyectoID, runtimeID, "tmux", "session", "orq-codex3/%7", "cerrado",
		`{"working_dir":"/tmp/TestStale/orquestador","cwd":"/tmp/TestStale/orquestador"}`,
	)
	handle, err := GetRuntimeHandle(handleID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %v %+v", err, handle)
	}
	runtime, err := GetRuntime(runtimeID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %v %+v", err, runtime)
	}

	handle, workingDir, err := SincronizarRuntimeHandleWorkingDir(handle, runtime, "Codex3", &proyectoID)
	if err != nil {
		t.Fatalf("sincronizar working dir: %v", err)
	}
	if workingDir != rutaProyecto {
		t.Fatalf("working dir inesperado: got=%q want=%q", workingDir, rutaProyecto)
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if got := strings.TrimSpace(stringFromMap(meta, "working_dir", "")); got != rutaProyecto {
		t.Fatalf("working_dir del handle no saneado: %q meta=%s", got, handle.MetadataJSON)
	}
	if got := strings.TrimSpace(stringFromMap(meta, "cwd", "")); got != rutaProyecto {
		t.Fatalf("cwd del handle no saneado: %q meta=%s", got, handle.MetadataJSON)
	}
}

func TestSincronizarRuntimeHandleWorkingDirReparaSesionStaleAunqueRuntimeYaSeaCanonico(t *testing.T) {
	tmp := prepararDBTemporal(t)
	rutaProyecto := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(filepath.Join(rutaProyecto, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir git: %v", err)
	}
	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaProyecto,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex3",
		ProyectoID:        &proyectoID,
		CWD:               "/tmp/TestStale/orquestador",
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-stale-codex3",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex3",
		ProyectoID:   &proyectoID,
		SesionID:     &sesion.ID,
		Provider:     "openai",
		Connector:    "codex-cli",
		LogicalState: "activo",
		ProcessState: "running",
		CWD:          rutaProyecto,
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	if err := UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert handle desde sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %v %+v", err, handle)
	}
	if err := ActualizarMetadataRuntimeHandle(handle.ID, `{"working_dir":"/tmp/TestStale/orquestador","cwd":"/tmp/TestStale/orquestador"}`); err != nil {
		t.Fatalf("actualizar metadata handle: %v", err)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("refresh handle: %v %+v", err, handle)
	}
	runtime, err := GetRuntime(runtimeID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %v %+v", err, runtime)
	}

	handle, workingDir, err := SincronizarRuntimeHandleWorkingDir(handle, runtime, "Codex3", &proyectoID)
	if err != nil {
		t.Fatalf("sincronizar working dir: %v", err)
	}
	if workingDir != rutaProyecto {
		t.Fatalf("working dir inesperado: got=%q want=%q", workingDir, rutaProyecto)
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if got := strings.TrimSpace(stringFromMap(meta, "working_dir", "")); got != rutaProyecto {
		t.Fatalf("working_dir del handle no saneado: %q meta=%s", got, handle.MetadataJSON)
	}
	if got := strings.TrimSpace(stringFromMap(meta, "cwd", "")); got != rutaProyecto {
		t.Fatalf("cwd del handle no saneado: %q meta=%s", got, handle.MetadataJSON)
	}
	sesionActualizada, err := GetSesionByID(sesion.ID)
	if err != nil || sesionActualizada == nil {
		t.Fatalf("get sesion actualizada: %v %+v", err, sesionActualizada)
	}
	if sesionActualizada.CWD != rutaProyecto {
		t.Fatalf("la sesion deberia quedar canonizada: got=%q want=%q", sesionActualizada.CWD, rutaProyecto)
	}
}

func TestGetRuntimeOrderDevuelveNilSiNoExiste(t *testing.T) {
	prepararDBTemporal(t)

	order, err := GetRuntimeOrder(999999)
	if err != nil {
		t.Fatalf("get runtime order inexistente: %v", err)
	}
	if order != nil {
		t.Fatalf("runtime order inexistente deberia ser nil: %+v", order)
	}
}

func TestReconciliarRuntimeOrdersPendientesMailboxConsumidoToleraOrderBorradaDelHotIndex(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		Tipo:        "send_instruction",
		PayloadJSON: `{"mailbox_id":123,"texto":"continua trabajo actual","to_agente":"Codex1"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	estado := "pendiente"
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders vivas: %v", err)
	}
	if len(orders) != 1 || orders[0] == nil || orders[0].ID != orderID {
		t.Fatalf("hot index inesperado: %+v", orders)
	}
	if _, err := DB.DB.Exec(`DELETE FROM runtime_orders WHERE id = ?`, orderID); err != nil {
		t.Fatalf("borrar runtime order por fuera del hot index: %v", err)
	}

	processed, err := reconciliarRuntimeOrdersPendientesMailboxConsumido()
	if err != nil {
		t.Fatalf("reconciliar runtime orders con hot index stale: %v", err)
	}
	if processed != 0 {
		t.Fatalf("processed inesperado: %d", processed)
	}
}

func TestResolverSesionParaOrdenToleraSesionFaltanteEnHandleYRuntime(t *testing.T) {
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
		ExternalSessionID: "sess-stale",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}
	if handle == nil {
		t.Fatalf("handle nil")
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("get runtime por sesion: %v", err)
	}
	if runtime == nil {
		t.Fatalf("runtime nil")
	}
	if _, err := DB.Exec(`DELETE FROM sesiones WHERE id = ?`, sesion.ID); err != nil {
		t.Fatalf("borrar sesion: %v", err)
	}

	order := &RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		HandleID:   &handle.ID,
		RuntimeID:  &runtime.ID,
		Tipo:       "checkpoint",
	}
	got, err := resolverSesionParaOrden(order)
	if err != nil {
		t.Fatalf("resolver sesion con sesion faltante: %v", err)
	}
	if got != nil {
		t.Fatalf("sesion inesperada: %+v", got)
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

func TestClaimNextRuntimeOrderToleraAvailableAtLegacyConZonaHoraria(t *testing.T) {
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
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador"}`,
	})
	if err != nil {
		t.Fatalf("encolar runtime order: %v", err)
	}
	legacyAvailableAt := time.Now().UTC().Add(-2 * time.Minute).String()
	if _, err := DB.Exec(`UPDATE runtime_orders SET available_at=? WHERE id=?`, legacyAvailableAt, orderID); err != nil {
		t.Fatalf("forzar available_at legacy: %v", err)
	}

	claimed, err := ClaimNextRuntimeOrder("Codex1")
	if err != nil {
		t.Fatalf("claim runtime order legacy timezone: %v", err)
	}
	if claimed == nil || claimed.ID != orderID {
		t.Fatalf("runtime order legacy inesperada: %+v", claimed)
	}
	if claimed.Estado != "tomada" {
		t.Fatalf("estado inesperado tras claim legacy: %s", claimed.Estado)
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

	orderID := mustInsertID(t, `
		INSERT INTO runtime_orders (
			agente, proyecto_id, tipo, payload_json, resultado_json, error_text, estado, available_at
		) VALUES (?,?,?,?,?,'', 'pendiente', CURRENT_TIMESTAMP)`,
		"Codex1", proyectoID, "send_instruction", `{"texto":"rota"`, `{}`,
	)
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

	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado
	) VALUES (?,?,?,?,?,?,?)`,
		"Codex1", proyectoID, runtime.ID, "cli", "process", "stale-1", "cerrado",
	)
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
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado
	) VALUES (?,?,?,?,?,?)`,
		"Codex1", proyectoID, "cli", "process", "stale-2", "fallido",
	)
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
	if err := ConfigSet("runtime_handles_retention_minutes", "-1"); err != nil {
		t.Fatalf("ConfigSet handles retention minutes: %v", err)
	}
	if err := ConfigSet("runtime_orders_retention_hours", "72"); err != nil {
		t.Fatalf("ConfigSet orders retention: %v", err)
	}
	if err := ConfigSet("runtime_orders_retention_minutes", "-1"); err != nil {
		t.Fatalf("ConfigSet orders retention minutes: %v", err)
	}
	if err := ConfigSet("runtime_instances_retention_hours", "72"); err != nil {
		t.Fatalf("ConfigSet runtimes retention: %v", err)
	}
	if err := ConfigSet("runtime_instances_retention_minutes", "-1"); err != nil {
		t.Fatalf("ConfigSet runtimes retention minutes: %v", err)
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
	orderOldID := mustInsertID(t, `INSERT INTO runtime_orders (agente, handle_id, tipo, payload_json, resultado_json, estado, created_at, updated_at, available_at) VALUES ('Codex1', ?, 'send_instruction', '{}', '{}', 'fallida', ?, ?, ?)`,
		handleOldID, time.Now().UTC().Add(-96*time.Hour), time.Now().UTC().Add(-96*time.Hour), time.Now().UTC().Add(-96*time.Hour))
	orderRecentID := mustInsertID(t, `INSERT INTO runtime_orders (agente, handle_id, tipo, payload_json, resultado_json, estado, created_at, updated_at, available_at) VALUES ('Codex1', ?, 'send_instruction', '{}', '{}', 'fallida', ?, ?, ?)`,
		handleRecentID, time.Now().UTC().Add(-2*time.Hour), time.Now().UTC().Add(-2*time.Hour), time.Now().UTC().Add(-2*time.Hour))
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

func TestPurgarRuntimeHistoricoPriorizaRetencionEnMinutos(t *testing.T) {
	abrirDBTemporalMemoria(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	if err := ConfigSet("runtime_handles_retention_minutes", "10"); err != nil {
		t.Fatalf("ConfigSet handles retention minutes: %v", err)
	}
	if err := ConfigSet("runtime_orders_retention_minutes", "30"); err != nil {
		t.Fatalf("ConfigSet orders retention minutes: %v", err)
	}
	if err := ConfigSet("runtime_instances_retention_minutes", "30"); err != nil {
		t.Fatalf("ConfigSet runtimes retention minutes: %v", err)
	}
	if err := ConfigSet("runtime_handles_retention_hours", "24"); err != nil {
		t.Fatalf("ConfigSet handles retention hours: %v", err)
	}
	if err := ConfigSet("runtime_orders_retention_hours", "72"); err != nil {
		t.Fatalf("ConfigSet orders retention hours: %v", err)
	}
	if err := ConfigSet("runtime_instances_retention_hours", "72"); err != nil {
		t.Fatalf("ConfigSet runtimes retention hours: %v", err)
	}

	sesionID, err := IniciarSesion("Codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesionID)
	if err != nil || handle == nil {
		t.Fatalf("GetRuntimeHandleBySesionID: %+v err=%v", handle, err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='fallido', last_seen_at=?, created_at=?, updated_at=? WHERE id=?`,
		time.Now().UTC().Add(-20*time.Minute), time.Now().UTC().Add(-20*time.Minute), time.Now().UTC().Add(-20*time.Minute), handle.ID); err != nil {
		t.Fatalf("ajustar handle: %v", err)
	}
	orderID := mustInsertID(t, `INSERT INTO runtime_orders (agente, handle_id, tipo, payload_json, resultado_json, estado, created_at, updated_at, available_at) VALUES ('Codex1', ?, 'send_instruction', '{}', '{}', 'fallida', ?, ?, ?)`,
		handle.ID, time.Now().UTC().Add(-40*time.Minute), time.Now().UTC().Add(-40*time.Minute), time.Now().UTC().Add(-40*time.Minute))
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		LogicalState: "degradado",
		ProcessState: "stopped",
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeInstance: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances SET last_event_at=?, last_heartbeat_at=?, created_at=?, updated_at=? WHERE id=?`,
		time.Now().UTC().Add(-40*time.Minute), time.Now().UTC().Add(-40*time.Minute), time.Now().UTC().Add(-40*time.Minute), time.Now().UTC().Add(-40*time.Minute), runtimeID); err != nil {
		t.Fatalf("ajustar runtime: %v", err)
	}

	resultado, err := PurgarRuntimeHistorico()
	if err != nil {
		t.Fatalf("PurgarRuntimeHistorico: %v", err)
	}
	if resultado == nil || resultado.Handles == nil || resultado.Orders == nil || resultado.Runtimes == nil {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if resultado.Handles.Deleted != 1 || resultado.Orders.Deleted != 1 || resultado.Runtimes.Deleted != 1 {
		t.Fatalf("purga por minutos inesperada: %+v", resultado)
	}
	if got, err := GetRuntimeHandle(handle.ID); err != nil {
		t.Fatalf("GetRuntimeHandle tras purga: %v", err)
	} else if got != nil {
		t.Fatalf("handle deberia purgarse por retencion en minutos: %+v", got)
	}
	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_orders WHERE id = ?`, orderID).Scan(&count); err != nil {
		t.Fatalf("count order: %v", err)
	}
	if count != 0 {
		t.Fatalf("order deberia purgarse por retencion en minutos: %d", count)
	}
	if got, err := GetRuntime(runtimeID); err != nil && err != sql.ErrNoRows {
		t.Fatalf("GetRuntime tras purga: %v", err)
	} else if got != nil {
		t.Fatalf("runtime deberia purgarse por retencion en minutos: %+v", got)
	}
}

func TestReconciliarRuntimeOrdersPendientesHandoffExpiradasExpiraPendienteViejaSinRuntimeActivo(t *testing.T) {
	abrirDBTemporalMemoria(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	if err := ConfigSet("runtime_pending_handoff_expiry_minutes", "10"); err != nil {
		t.Fatalf("ConfigSet handoff expiry: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		Tipo:        "handoff",
		PayloadJSON: `{"agente_origen":"Codex0","agente_destino":"Codex1","resumen_continuidad":"handoff vieja"}`,
	})
	if err != nil {
		t.Fatalf("EncolarRuntimeOrder: %v", err)
	}
	old := time.Now().UTC().Add(-20 * time.Minute)
	if _, err := DB.Exec(`UPDATE runtime_orders SET created_at=?, updated_at=?, available_at=? WHERE id=?`, old, old, old, orderID); err != nil {
		t.Fatalf("envejecer handoff: %v", err)
	}

	n, err := ReconciliarRuntimeOrdersPendientesHandoffExpiradas()
	if err != nil {
		t.Fatalf("ReconciliarRuntimeOrdersPendientesHandoffExpiradas: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia expirar 1 handoff vieja, got=%d", n)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("GetRuntimeOrder: %v", err)
	}
	if order == nil || order.Estado != "expirada" {
		t.Fatalf("handoff no expirada: %+v", order)
	}
	if !strings.Contains(order.ErrorText, "handoff pendiente expirada por higiene") {
		t.Fatalf("error_text sin motivo de higiene: %+v", order)
	}
	if !strings.Contains(order.ResultadoJSON, `"pending_handoff_hygiene_timeout"`) {
		t.Fatalf("resultado_json sin reason de higiene: %s", order.ResultadoJSON)
	}
}

func TestReconciliarRuntimeOrdersPendientesHandoffExpiradasRespetaDestinoConHandleOperativo(t *testing.T) {
	abrirDBTemporalMemoria(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	if err := ConfigSet("runtime_pending_handoff_expiry_minutes", "10"); err != nil {
		t.Fatalf("ConfigSet handoff expiry: %v", err)
	}
	sesionID, err := IniciarSesion("Codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if _, err := GetRuntimeHandleBySesionID(sesionID); err != nil {
		t.Fatalf("GetRuntimeHandleBySesionID: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		Tipo:        "handoff",
		PayloadJSON: `{"agente_origen":"Codex0","agente_destino":"Codex1","resumen_continuidad":"handoff viva"}`,
	})
	if err != nil {
		t.Fatalf("EncolarRuntimeOrder: %v", err)
	}
	old := time.Now().UTC().Add(-20 * time.Minute)
	if _, err := DB.Exec(`UPDATE runtime_orders SET created_at=?, updated_at=?, available_at=? WHERE id=?`, old, old, old, orderID); err != nil {
		t.Fatalf("envejecer handoff: %v", err)
	}

	n, err := ReconciliarRuntimeOrdersPendientesHandoffExpiradas()
	if err != nil {
		t.Fatalf("ReconciliarRuntimeOrdersPendientesHandoffExpiradas: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia expirar handoff con handle operativo, got=%d", n)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("GetRuntimeOrder: %v", err)
	}
	if order == nil || order.Estado != "pendiente" {
		t.Fatalf("handoff no deberia cambiar: %+v", order)
	}
}

func TestReconciliarRuntimeOrdersPendientesHandoffExpiradasExpiraRuntimeEstancadaSinHandle(t *testing.T) {
	abrirDBTemporalMemoria(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	if err := ConfigSet("runtime_pending_handoff_expiry_minutes", "10"); err != nil {
		t.Fatalf("ConfigSet handoff expiry: %v", err)
	}
	sesionID, err := IniciarSesion("Codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesionID)
	if err != nil || handle == nil {
		t.Fatalf("GetRuntimeHandleBySesionID: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesionID)
	if err != nil || runtime == nil {
		t.Fatalf("GetRuntimeBySesionID: %+v err=%v", runtime, err)
	}
	old := time.Now().UTC().Add(-20 * time.Minute)
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado', last_seen_at=?, updated_at=? WHERE id=?`, old, old, handle.ID); err != nil {
		t.Fatalf("cerrar handle: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances SET logical_state='disponible', process_state='running', last_event_at=?, last_heartbeat_at=?, updated_at=? WHERE id=?`, old, old, old, runtime.ID); err != nil {
		t.Fatalf("envejecer runtime: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		Tipo:        "handoff",
		PayloadJSON: `{"agente_origen":"Codex0","agente_destino":"Codex1","resumen_continuidad":"handoff estancada"}`,
	})
	if err != nil {
		t.Fatalf("EncolarRuntimeOrder: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET created_at=?, updated_at=?, available_at=? WHERE id=?`, old, old, old, orderID); err != nil {
		t.Fatalf("envejecer handoff: %v", err)
	}

	n, err := ReconciliarRuntimeOrdersPendientesHandoffExpiradas()
	if err != nil {
		t.Fatalf("ReconciliarRuntimeOrdersPendientesHandoffExpiradas: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia expirar handoff con runtime estancada, got=%d", n)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("GetRuntimeOrder: %v", err)
	}
	if order == nil || order.Estado != "expirada" {
		t.Fatalf("handoff no expirada: %+v", order)
	}
}

func TestReconciliarRuntimeOrdersPendientesHandoffExpiradasRespetaRuntimeRecienteSinHandle(t *testing.T) {
	abrirDBTemporalMemoria(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	if err := ConfigSet("runtime_pending_handoff_expiry_minutes", "10"); err != nil {
		t.Fatalf("ConfigSet handoff expiry: %v", err)
	}
	sesionID, err := IniciarSesion("Codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesionID)
	if err != nil || handle == nil {
		t.Fatalf("GetRuntimeHandleBySesionID: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesionID)
	if err != nil || runtime == nil {
		t.Fatalf("GetRuntimeBySesionID: %+v err=%v", runtime, err)
	}
	old := time.Now().UTC().Add(-20 * time.Minute)
	recent := time.Now().UTC().Add(-2 * time.Minute)
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado', last_seen_at=?, updated_at=? WHERE id=?`, old, old, handle.ID); err != nil {
		t.Fatalf("cerrar handle: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances SET logical_state='disponible', process_state='running', last_event_at=?, last_heartbeat_at=?, updated_at=? WHERE id=?`, recent, recent, recent, runtime.ID); err != nil {
		t.Fatalf("refrescar runtime: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		Tipo:        "handoff",
		PayloadJSON: `{"agente_origen":"Codex0","agente_destino":"Codex1","resumen_continuidad":"handoff viva por runtime"}`,
	})
	if err != nil {
		t.Fatalf("EncolarRuntimeOrder: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET created_at=?, updated_at=?, available_at=? WHERE id=?`, old, old, old, orderID); err != nil {
		t.Fatalf("envejecer handoff: %v", err)
	}

	n, err := ReconciliarRuntimeOrdersPendientesHandoffExpiradas()
	if err != nil {
		t.Fatalf("ReconciliarRuntimeOrdersPendientesHandoffExpiradas: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia expirar handoff con runtime reciente, got=%d", n)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("GetRuntimeOrder: %v", err)
	}
	if order == nil || order.Estado != "pendiente" {
		t.Fatalf("handoff no deberia cambiar: %+v", order)
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

	pid := int64(os.Getpid())
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":      "process_exec",
		"working_dir": filepath.Join(tmp, "orquestador"),
	})
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"Codex1", proyectoID, runtimeID, "cli", "", "", string(metaJSON), time.Now().UTC())

	order := &RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtimeID,
		HandleID:   &handleID,
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

func TestControlarProcesoRuntimePromueveHandleTMUXCanonicoSobreLegacy(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexCanonico", "programador"); err != nil {
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
	runtimeLegacyID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexCanonico",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime legacy: %v", err)
	}
	legacyMetaJSON, _ := json.Marshal(map[string]any{
		"driver":           "process_pty_cli",
		"working_dir":      filepath.Join(tmp, "legacy"),
		"rendered_command": "codex-perfil CodexCanonico --model gpt-5.4",
	})
	legacyHandleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexCanonico", proyectoID, runtimeLegacyID, "cli", "process", strconv.Itoa(os.Getpid()), string(legacyMetaJSON), time.Now().UTC())

	runDir := filepath.Join(tmp, "worker")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexCanonico","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexcanonico-001","tmux_pane_id":"%7","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	runtimeTMUXID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexCanonico",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime tmux: %v", err)
	}
	tmuxMetaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codexcanonico-001",
		"tmux_pane_id":          "%7",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"working_dir":           filepath.Join(tmp, "worker"),
		"rendered_command":      "codex-perfil CodexCanonico --model gpt-5.4",
	})
	tmuxHandleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexCanonico", proyectoID, runtimeTMUXID, "tmux", "process", "orq-codexcanonico-001/%7", string(tmuxMetaJSON), time.Now().UTC())

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "CodexCanonico",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtimeLegacyID,
		HandleID:    &legacyHandleID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"CodexCanonico","texto":"hola tmux"}`,
	})
	if err != nil {
		t.Fatalf("encolar order: %v", err)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}

	handleResuelto, err := resolverHandleParaOrden(order)
	if err != nil || handleResuelto == nil {
		t.Fatalf("resolverHandleParaOrden: %+v err=%v", handleResuelto, err)
	}
	if handleResuelto.ID != tmuxHandleID {
		t.Fatalf("deberia promover el handle tmux canónico, got=%+v", handleResuelto)
	}
	order, err = GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order tras resolver: %v", err)
	}
	if order.HandleID == nil || *order.HandleID != tmuxHandleID {
		t.Fatalf("la order deberia quedar reescrita al handle tmux, got=%+v", order)
	}
	if order.RuntimeID == nil || *order.RuntimeID != runtimeTMUXID {
		t.Fatalf("la order deberia quedar reescrita al runtime tmux, got=%+v", order)
	}

	var got controlruntime.ObjetivoProceso
	aplicado, _, err := controlarProcesoRuntime(order, func(obj controlruntime.ObjetivoProceso) (bool, int, error) {
		got = obj
		return true, 0, nil
	})
	if err != nil || !aplicado {
		t.Fatalf("controlarProcesoRuntime: aplicado=%t err=%v", aplicado, err)
	}
	if got.HandleRef != "orq-codexcanonico-001/%7" {
		t.Fatalf("deberia controlar el handle tmux canónico, got=%+v", got)
	}
	if !strings.Contains(got.MetadataJSON, `"driver":"tmux_cli_session"`) {
		t.Fatalf("metadata control deberia ser tmux, got=%s", got.MetadataJSON)
	}

	legacyHandle, err := GetRuntimeHandle(legacyHandleID)
	if err != nil {
		t.Fatalf("get legacy handle: %v", err)
	}
	if legacyHandle == nil || (legacyHandle.Estado != "cerrado" && legacyHandle.Estado != "fallido") {
		t.Fatalf("el handle legacy ya no deberia seguir activo: %+v", legacyHandle)
	}
}

func TestControlarProcesoRuntimeNoCaeAPIDSinHandleParaCLITMUXPreferred(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexNoPID", "programador"); err != nil {
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
		Agente:      "CodexNoPID",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		PID:         &pid,
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := DB.Exec(`DELETE FROM runtime_handles WHERE sesion_id = ?`, sesion.ID); err != nil {
		t.Fatalf("delete runtime handles: %v", err)
	}
	runtimeHandleHotReset()

	order := &RuntimeOrder{
		Agente:     "CodexNoPID",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		Tipo:       "pause",
	}
	var got controlruntime.ObjetivoProceso
	aplicado, _, err := controlarProcesoRuntime(order, func(obj controlruntime.ObjetivoProceso) (bool, int, error) {
		got = obj
		return true, 0, nil
	})
	if err != nil || !aplicado {
		t.Fatalf("controlarProcesoRuntime: aplicado=%t err=%v", aplicado, err)
	}
	if got.PID != nil {
		t.Fatalf("un CLI tmux-preferred sin handle canónico no deberia caer a PID: %+v", got)
	}
	if strings.TrimSpace(got.HandleRef) != "" || strings.TrimSpace(got.HandleKind) != "" {
		t.Fatalf("no deberia heredar handle legacy inexistente: %+v", got)
	}
}

func TestControlarProcesoRuntimeMantienePIDSinHandleParaProcesoGenerico(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("ShellNoPID", "programador"); err != nil {
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
		Agente:      "ShellNoPID",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "shell",
		PID:         &pid,
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := DB.Exec(`DELETE FROM runtime_handles WHERE sesion_id = ?`, sesion.ID); err != nil {
		t.Fatalf("delete runtime handles: %v", err)
	}
	runtimeHandleHotReset()

	order := &RuntimeOrder{
		Agente:     "ShellNoPID",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		Tipo:       "pause",
	}
	var got controlruntime.ObjetivoProceso
	aplicado, _, err := controlarProcesoRuntime(order, func(obj controlruntime.ObjetivoProceso) (bool, int, error) {
		got = obj
		return true, 0, nil
	})
	if err != nil || !aplicado {
		t.Fatalf("controlarProcesoRuntime: aplicado=%t err=%v", aplicado, err)
	}
	if got.PID == nil || *got.PID != pid {
		t.Fatalf("un proceso local genérico sin handle debería seguir usando PID: %+v", got)
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

func TestMarcarRuntimeHandlesCerradosRespetaProyecto(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoAID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoBID, err := UpsertProyecto(&Proyecto{
		Slug:    "otro",
		Nombre:  "Otro",
		RutaAbs: filepath.Join(tmp, "otro"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}

	sA, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoAID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion A: %v", err)
	}
	sB, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoBID,
		CWD:         filepath.Join(tmp, "otro"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion B: %v", err)
	}

	if err := MarcarRuntimeHandlesCerrados("Codex1", &proyectoAID); err != nil {
		t.Fatalf("marcar handles cerrados por proyecto: %v", err)
	}

	handleA, err := GetRuntimeHandleBySesionID(sA.ID)
	if err != nil {
		t.Fatalf("handleA: %v", err)
	}
	handleB, err := GetRuntimeHandleBySesionID(sB.ID)
	if err != nil {
		t.Fatalf("handleB: %v", err)
	}
	if handleA == nil || handleA.Estado != "cerrado" {
		t.Fatalf("handleA no cerrado: %+v", handleA)
	}
	if handleB == nil || handleB.Estado != "activo" {
		t.Fatalf("handleB no deberia cerrarse: %+v", handleB)
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
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, last_seen_at
	) VALUES (?,?,?,?,?,?, 'fallido', ?)`,
		"Codex1", proyectoID, runtimeID, "cli", "process", strconv.Itoa(os.Getpid()), time.Now().UTC().Add(-10*time.Minute))

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
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?)`,
		"Codex1", proyectoID, runtimeID, "cli", "process", strconv.FormatInt(pidFantasma, 10), time.Now().UTC())

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

func TestProcesarRuntimeOrdersBatchRespetaBatchBudget(t *testing.T) {
	prepararDBTemporal(t)

	prevBudget := runtimeOrdersBatchBudgetOverride
	prevExec := ejecutarRuntimeOrderBasicaFn
	t.Cleanup(func() {
		runtimeOrdersBatchBudgetOverride = prevBudget
		ejecutarRuntimeOrderBasicaFn = prevExec
	})

	if err := RegistrarAgente("CodexBudget", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "budget-orders",
		Nombre:  "Budget Orders",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := EncolarRuntimeOrder(&RuntimeOrder{
			Agente:      "CodexBudget",
			ProyectoID:  &proyectoID,
			Tipo:        "stop",
			PayloadJSON: `{"reason":"test_budget"}`,
			Estado:      "pendiente",
		}); err != nil {
			t.Fatalf("encolar stop[%d]: %v", i, err)
		}
	}

	runtimeOrdersBatchBudgetOverride = time.Millisecond
	execCalls := 0
	ejecutarRuntimeOrderBasicaFn = func(order *RuntimeOrder) error {
		execCalls++
		time.Sleep(3 * time.Millisecond)
		return nil
	}

	processed, err := procesarRuntimeOrdersBatchTipos([]string{"stop"})
	if err != nil {
		t.Fatalf("procesarRuntimeOrdersBatchTipos: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed=%d, want=1", processed)
	}
	if execCalls != 1 {
		t.Fatalf("execCalls=%d, want=1", execCalls)
	}

	agente := "CodexBudget"
	estadoPendiente := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("listar pending: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("pending=%d, want=1", len(orders))
	}
}

func TestEjecutarRuntimeOrderStartNoConsumeMailboxBootstrapSinLease(t *testing.T) {
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
	if int64FromAny(bootstrap["consumidos"]) != 0 {
		t.Fatalf("consumidos inesperado en start result: %+v", bootstrap)
	}

	estadoPendiente := "pendiente"
	toAgente := "Codex1"
	pendientes, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &estadoPendiente,
	})
	if err != nil {
		t.Fatalf("listar runtime mailbox pendiente: %v", err)
	}
	found := false
	for _, msg := range pendientes {
		if msg != nil && msg.ID == msgID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("mailbox bootstrap deberia quedar pendiente: %+v", pendientes)
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

func TestEjecutarRuntimeOrderStartNoConsumeMailboxLigadaASendInstruction(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Ollama1", "programador"); err != nil {
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

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Ollama1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Ollama1","texto":"microtarea inline"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	msgID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:     "server",
		ToAgente:       "Ollama1",
		ProyectoID:     &proyectoID,
		RuntimeOrderID: &sendID,
		Kind:           "instruction",
		PayloadJSON:    `{"texto":"microtarea inline"}`,
	})
	if err != nil {
		t.Fatalf("enviar runtime mailbox: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Ollama1",
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
	if int64FromAny(bootstrap["consumidos"]) != 0 {
		t.Fatalf("consumidos inesperado en start result: %+v", bootstrap)
	}

	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("send_instruction no deberia alterarse durante start: %+v", sendOrder)
	}

	estadoPendiente := "pendiente"
	toAgente := "Ollama1"
	pendientes, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &estadoPendiente,
	})
	if err != nil {
		t.Fatalf("listar runtime mailbox pendiente: %v", err)
	}
	found := false
	for _, msg := range pendientes {
		if msg != nil && msg.ID == msgID && msg.RuntimeOrderID != nil && *msg.RuntimeOrderID == sendID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("mailbox ligada a send_instruction deberia seguir pendiente: %+v", pendientes)
	}
}

func TestEjecutarRuntimeOrderStartReencolaSiHaySesionActivaEnOtroProyecto(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	orquestadorID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert orquestador: %v", err)
	}
	pluginsID, err := UpsertProyecto(&Proyecto{
		Slug:    "plugins",
		Nombre:  "Plugins",
		RutaAbs: filepath.Join(tmp, "plugins"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert plugins: %v", err)
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
		Agente:      "Codex2",
		ProyectoID:  &pluginsID,
		CWD:         filepath.Join(tmp, "plugins"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion plugins: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex2",
		ProyectoID:  &orquestadorID,
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
		t.Fatalf("ejecutar start con sesion ajena: %v", err)
	}

	startOrder, err = GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order final: %v", err)
	}
	if startOrder.Estado != "pendiente" {
		t.Fatalf("la start deberia reencolarse, got=%s", startOrder.Estado)
	}
	if !strings.Contains(startOrder.ErrorText, "sesion_activa_en_otro_proyecto") {
		t.Fatalf("start reencolada sin motivo esperado: %s", startOrder.ErrorText)
	}
	if startOrder.AvailableAt.IsZero() || !startOrder.AvailableAt.After(time.Now().UTC().Add(-time.Second)) {
		t.Fatalf("start reencolada sin retry futuro: %+v", startOrder)
	}
	if sesion, err := GetSesionActiva("Codex2", &orquestadorID); err != sql.ErrNoRows || sesion != nil {
		t.Fatalf("no deberia abrir sesion nueva en orquestador: %+v err=%v", sesion, err)
	}
	estadoPendiente := "pendiente"
	agente := "Codex2"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: &pluginsID,
		Estado:     &estadoPendiente,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("listar stop plugins: %v", err)
	}
	foundStop := false
	for _, order := range orders {
		if order != nil && order.Tipo == "stop" {
			foundStop = true
			break
		}
	}
	if !foundStop {
		t.Fatalf("deberia encolar stop para el proyecto ajeno: %+v", orders)
	}
}

func TestEjecutarRuntimeOrderStartCierraSesionFantasmaDelMismoProyecto(t *testing.T) {
	enableLegacyPTYLocalRuntimeForTest(t)
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

	sesionVieja, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion vieja: %v", err)
	}
	staleHeartbeat := time.Now().UTC().Add(-4 * time.Hour)
	if _, err := DB.Exec(`UPDATE sesiones SET heartbeat_at=?, estado='activa', activa=1 WHERE id=?`, staleHeartbeat, sesionVieja.ID); err != nil {
		t.Fatalf("envejecer sesion vieja: %v", err)
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
		t.Fatalf("reload start order: %v", err)
	}
	if startOrder.Estado != "completada" {
		t.Fatalf("start deberia completar tras cerrar sesion fantasma: %+v", startOrder)
	}

	var activa int
	var estado string
	var fin sql.NullTime
	if err := DB.QueryRow(`SELECT activa, estado, fin FROM sesiones WHERE id=?`, sesionVieja.ID).Scan(&activa, &estado, &fin); err != nil {
		t.Fatalf("leer sesion vieja: %v", err)
	}
	if activa != 0 || strings.TrimSpace(estado) != "cerrada" || !fin.Valid {
		t.Fatalf("la sesion vieja deberia quedar cerrada: activa=%d estado=%s fin=%v", activa, estado, fin)
	}

	sesionNueva, err := GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesionNueva == nil {
		t.Fatalf("sesion nueva activa: %+v err=%v", sesionNueva, err)
	}
	if sesionNueva.ID == sesionVieja.ID {
		t.Fatalf("deberia crear una sesion nueva, no reutilizar la fantasma: %+v", sesionNueva)
	}
}

func TestProcesarRuntimeOrdersBatchReencolaStartSiHaySesionActivaEnOtroProyecto(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	orquestadorID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert orquestador: %v", err)
	}
	pluginsID, err := UpsertProyecto(&Proyecto{
		Slug:    "plugins",
		Nombre:  "Plugins",
		RutaAbs: filepath.Join(tmp, "plugins"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert plugins: %v", err)
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
		Agente:      "Codex2",
		ProyectoID:  &pluginsID,
		CWD:         filepath.Join(tmp, "plugins"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion plugins: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex2",
		ProyectoID:  &orquestadorID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"cat-cli"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba procesar al menos una orden de control, got=%d", processed)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order final: %v", err)
	}
	if startOrder.Estado != "pendiente" {
		t.Fatalf("la start deberia seguir pendiente tras reencolarse, got=%s", startOrder.Estado)
	}
	if !strings.Contains(startOrder.ErrorText, "sesion_activa_en_otro_proyecto") {
		t.Fatalf("start reencolada sin motivo esperado: %s", startOrder.ErrorText)
	}
	if startOrder.AvailableAt.IsZero() {
		t.Fatalf("start batch reencolada sin retry futuro: %+v", startOrder)
	}
}

func TestProcesarRuntimeOrdersBatchCierraSesionFantasmaAjenaAntesDeStart(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	orquestadorID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert orquestador: %v", err)
	}
	pluginsID, err := UpsertProyecto(&Proyecto{
		Slug:    "plugins",
		Nombre:  "Plugins",
		RutaAbs: filepath.Join(tmp, "plugins"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert plugins: %v", err)
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
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &pluginsID,
		CWD:         filepath.Join(tmp, "plugins"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion plugins: %v", err)
	}
	staleAt := time.Now().UTC().Add(-10 * time.Minute)
	if _, err := DB.Exec(`UPDATE sesiones SET heartbeat_at=? WHERE id=?`, staleAt, sesion.ID); err != nil {
		t.Fatalf("forzar heartbeat stale: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex2",
		ProyectoID:  &orquestadorID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"cat-cli"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba procesar al menos una orden de control, got=%d", processed)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order final: %v", err)
	}
	if startOrder.Estado != "pendiente" {
		t.Fatalf("la start deberia reencolarse, got=%s", startOrder.Estado)
	}
	if !strings.Contains(startOrder.ErrorText, "sesion_fantasma_en_otro_proyecto") {
		t.Fatalf("start reencolada sin motivo fantasma esperado: %s", startOrder.ErrorText)
	}
	sesionFantasma, err := GetSesionAbierta("Codex2", &pluginsID)
	if err != sql.ErrNoRows || sesionFantasma != nil {
		t.Fatalf("la sesion fantasma deberia quedar cerrada: %+v err=%v", sesionFantasma, err)
	}
}

func TestRuntimeOrderBatchLessPriorizaControlSobreGuidance(t *testing.T) {
	now := time.Now().UTC()
	stop := &RuntimeOrder{ID: 20, Tipo: "stop", CreatedAt: now}
	start := &RuntimeOrder{ID: 21, Tipo: "start", CreatedAt: now}
	nudge := &RuntimeOrder{ID: 22, Tipo: "nudge", CreatedAt: now}

	if !runtimeOrderBatchLess(stop, start) {
		t.Fatalf("stop deberia priorizar sobre start")
	}
	if !runtimeOrderBatchLess(start, nudge) {
		t.Fatalf("start deberia priorizar sobre nudge")
	}
	if runtimeOrderBatchLess(nudge, stop) {
		t.Fatalf("nudge no deberia priorizar sobre stop")
	}
}

func TestAsegurarStopSesionActivaAjenaPropagaHandleYRuntimeSesionObjetivo(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoOrigen, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto origen: %v", err)
	}
	proyectoDestino, err := UpsertProyecto(&Proyecto{
		Slug:    "plugins",
		Nombre:  "Plugins",
		RutaAbs: filepath.Join(tmp, "plugins"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto destino: %v", err)
	}
	sesionAjena, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoDestino,
		CWD:         filepath.Join(tmp, "plugins"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion ajena: %v", err)
	}
	handleAjeno, err := GetRuntimeHandleBySesionID(sesionAjena.ID)
	if err != nil || handleAjeno == nil {
		t.Fatalf("get handle ajeno: %+v err=%v", handleAjeno, err)
	}
	runtimeAjeno, err := GetRuntimeBySesionID(sesionAjena.ID)
	if err != nil || runtimeAjeno == nil {
		t.Fatalf("get runtime ajeno: %+v err=%v", runtimeAjeno, err)
	}
	startOrder := &RuntimeOrder{ID: 17, Agente: "Codex2", ProyectoID: &proyectoOrigen, Tipo: "start"}

	if err := asegurarStopSesionActivaAjena(startOrder, sesionAjena); err != nil {
		t.Fatalf("asegurar stop ajena: %v", err)
	}

	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{
		Agente:     stringPtr("Codex2"),
		ProyectoID: &proyectoDestino,
		Estado:     stringPtr("pendiente"),
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("listar stop ajena: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("esperaba una stop ajena, got=%d", len(orders))
	}
	stopOrder := orders[0]
	if stopOrder.Tipo != "stop" {
		t.Fatalf("tipo inesperado: %+v", stopOrder)
	}
	if stopOrder.HandleID == nil || *stopOrder.HandleID != handleAjeno.ID {
		t.Fatalf("handle no propagado: %+v", stopOrder)
	}
	if stopOrder.RuntimeID == nil || *stopOrder.RuntimeID != runtimeAjeno.ID {
		t.Fatalf("runtime no propagado: %+v", stopOrder)
	}
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

func TestProcesarRuntimeSupervisionBatchRespetaBatchBudget(t *testing.T) {
	tmp := prepararDBTemporal(t)

	prevFn := runtimeSupervisionHandleFn
	prevBudget := runtimeSupervisionBatchBudgetOverride
	t.Cleanup(func() {
		runtimeSupervisionHandleFn = prevFn
		runtimeSupervisionBatchBudgetOverride = prevBudget
	})

	runtimeSupervisionBatchBudgetOverride = 5 * time.Millisecond
	if err := ConfigSet("runtime_supervision_interval_seconds", "1"); err != nil {
		t.Fatalf("config supervision interval: %v", err)
	}
	if err := ConfigSet("runtime_supervision_batch_size", "10"); err != nil {
		t.Fatalf("config supervision batch size: %v", err)
	}
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente 1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente 2: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-budget",
		Nombre:  "Orquestador Budget",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	pid := int64(os.Getpid())
	sesion1, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		PID:         &pid,
	})
	if err != nil {
		t.Fatalf("iniciar sesion 1: %v", err)
	}
	sesion2, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		PID:         &pid,
	})
	if err != nil {
		t.Fatalf("iniciar sesion 2: %v", err)
	}
	handle1, err := GetRuntimeHandleBySesionID(sesion1.ID)
	if err != nil || handle1 == nil {
		t.Fatalf("handle1: %+v err=%v", handle1, err)
	}
	handle2, err := GetRuntimeHandleBySesionID(sesion2.ID)
	if err != nil || handle2 == nil {
		t.Fatalf("handle2: %+v err=%v", handle2, err)
	}
	old := time.Now().UTC().Add(-2 * time.Minute)
	if _, err := DB.Exec(`UPDATE runtime_handles SET last_seen_at = ? WHERE id IN (?, ?)`, old, handle1.ID, handle2.ID); err != nil {
		t.Fatalf("envejecer handles: %v", err)
	}

	calls := 0
	runtimeSupervisionHandleFn = func(handle *RuntimeHandle, runtime *RuntimeInstance, source string) (map[string]any, error) {
		calls++
		time.Sleep(10 * time.Millisecond)
		return map[string]any{"ok": true}, nil
	}

	n, err := ProcesarRuntimeSupervisionBatch()
	if err != nil {
		t.Fatalf("procesar runtime supervision: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia cortar tras una supervision por budget, got=%d", n)
	}
	if calls != 1 {
		t.Fatalf("deberia supervisar solo un handle antes de diferir, calls=%d", calls)
	}
}

func TestObservarProcesoLocalRuntimeMarcaFantasmaTMUXConCurrentPathBorrado(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexStaleTMUX", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	rutaBase := filepath.Join(tmp, "orquestador")
	rutaWorktree := filepath.Join(rutaBase, ".orquesta-worktrees", "orquestador-codexstaletmux")
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
		Agent:     "CodexStaleTMUX",
		Name:      "orquestador-codexstaletmux",
		Path:      rutaWorktree,
		Branch:    "orq-orquestador-codexstaletmux",
		BaseRef:   "HEAD",
		State:     coordinacion.WorktreeActive,
		Reason:    "test",
	}); err != nil {
		t.Fatalf("crear worktree activa: %v", err)
	}
	pid := int64(os.Getpid())
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "CodexStaleTMUX",
		ProyectoID:  &proyectoID,
		CWD:         rutaWorktree,
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

	runDir := filepath.Join(tmp, "worker-stale-tmux")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	invocations := filepath.Join(tmp, "tmux-stale-kill.log")
	fakeTmux := writeFakeTMUXPanePatchWithKillScriptDB(t, tmp, invocations)
	deletedCurrent := filepath.Join(tmp, "deleted-worker-path")
	now := time.Now().UTC().Add(-2 * time.Minute)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexStaleTMUX","driver":"tmux_cli_session","transport":"tmux","tmux_command":"`+fakeTmux+`","tmux_session":"orq-codexstaletmux","tmux_pane_id":"%73","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`","working_dir":"`+rutaWorktree+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now.Format(time.RFC3339Nano)+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`,"working_dir":"`+rutaWorktree+`","current_path":"`+deletedCurrent+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now.Format(time.RFC3339Nano)+`","started_at":"`+now.Format(time.RFC3339Nano)+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","transport":"tmux","tmux_command":"%s","tmux_session":"orq-codexstaletmux","tmux_pane_id":"%%73","working_dir":"%s","cwd":"%s","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, fakeTmux, rutaWorktree, rutaWorktree, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-codexstaletmux/%73', estado='activo', metadata_json=?, capabilities_json=?, last_seen_at=? WHERE id=?`,
		metaJSON, `{"mailbox_delivery_mode":"session_resume","can_send_input":false}`, now, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances SET cwd=?, logical_state='esperando_io', process_state='running', last_heartbeat_at=?, last_event_at=? WHERE id=?`,
		rutaWorktree, now, now, runtime.ID); err != nil {
		t.Fatalf("update runtime: %v", err)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("refresh handle tras update: %+v err=%v", handle, err)
	}

	observed, processResult, err := observarProcesoLocalRuntime(handle, runtime, "runtime_supervision")
	if err != nil {
		t.Fatalf("observar proceso local: %v", err)
	}
	if !observed {
		t.Fatal("deberia observar el runtime tmux stale")
	}
	if alive, _ := processResult["process_alive"].(bool); alive {
		t.Fatalf("processResult deberia marcar process_alive=false: %+v", processResult)
	}
	if got := strings.TrimSpace(stringFromMap(processResult, "process_error", "")); got != "tmux current_path stale or deleted" {
		t.Fatalf("process_error inesperado: %+v", processResult)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}
	if handle.Estado != "fallido" {
		t.Fatalf("handle tmux stale deberia quedar fallido: %+v", handle)
	}
	runtime, err = GetRuntime(runtime.ID)
	if err != nil || runtime == nil {
		t.Fatalf("reload runtime: %+v err=%v", runtime, err)
	}
	if runtime.LogicalState != "fallido" || runtime.ProcessState != "finalizado" {
		t.Fatalf("runtime stale deberia quedar degradado/finalizado: %+v", runtime)
	}
	data, err := os.ReadFile(invocations)
	if err != nil {
		t.Fatalf("leer invocaciones fake tmux: %v", err)
	}
	if !strings.Contains(string(data), "kill-session -t orq-codexstaletmux") {
		t.Fatalf("faltaba kill-session del tmux stale: %s", string(data))
	}
}

func TestProcesarHigieneRuntimesAutonomosBatchPurgaTMUXHuerfanaDeleted(t *testing.T) {
	prepararDBTemporal(t)

	prevCommand := runtimeTMUXCommandPathFn
	prevList := runtimeTMUXListSessionsFn
	prevKill := runtimeTMUXKillSessionFn
	t.Cleanup(func() {
		runtimeTMUXCommandPathFn = prevCommand
		runtimeTMUXListSessionsFn = prevList
		runtimeTMUXKillSessionFn = prevKill
	})

	runtimeTMUXCommandPathFn = func() (string, error) { return "fake-tmux", nil }
	runtimeTMUXListSessionsFn = func(string) ([]controlruntime.TMUXSessionInfo, error) {
		return []controlruntime.TMUXSessionInfo{
			{SessionName: "orq-codex2-064413", CurrentPath: "/tmp/TestAPI/foo/orquestador (deleted)"},
			{SessionName: "orq-codex2-live", CurrentPath: "/home/alberto/Trabajo/orquesta"},
			{SessionName: "user-session", CurrentPath: "/tmp/TestAPI/bar (deleted)"},
		}, nil
	}
	killed := make([]string, 0, 2)
	runtimeTMUXKillSessionFn = func(_ string, sessionName string) error {
		killed = append(killed, strings.TrimSpace(sessionName))
		return nil
	}

	if _, err := DB.Exec(`INSERT INTO runtime_handles (agente, transporte, handle_kind, handle_ref, estado, metadata_json, created_at, updated_at)
		VALUES ('Codex2', 'tmux', 'session', 'orq-codex2-live/%1', 'activo', '{"tmux_session":"orq-codex2-live"}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert runtime handle activo: %v", err)
	}

	n, err := ProcesarHigieneRuntimesAutonomosBatch()
	if err != nil {
		t.Fatalf("procesar higiene runtimes autonomos: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia purgar exactamente una sesion tmux huerfana, got=%d", n)
	}
	if got := strings.Join(killed, ","); got != "orq-codex2-064413" {
		t.Fatalf("kill-session inesperado, got=%q", got)
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
	enableLegacyPTYLocalRuntimeForTest(t)
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
	if !strings.Contains(sendOrder.ResultadoJSON, `"control_real":true`) ||
		!strings.Contains(sendOrder.ResultadoJSON, `"delivery_path":"interactive"`) {
		t.Fatalf("send_instruction deberia gobernar el proceso real: %+v", sendOrder)
	}
	deadline := time.Now().Add(5 * time.Second)
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
			t.Fatalf("el proceso no recibio la instruccion: resultado=%s handle=%+v log=%s", sendOrder.ResultadoJSON, handle, string(logData))
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

func TestRuntimeOrderStartLocalCodexRequiereTMUXPorDefecto(t *testing.T) {
	tmp := prepararDBTemporal(t)
	t.Setenv("PATH", tmp)

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
	err = ejecutarRuntimeOrderStart(startOrder)
	if err == nil {
		t.Fatalf("el arranque local de codex deberia exigir tmux")
	}
	if !strings.Contains(err.Error(), "requiere tmux") {
		t.Fatalf("error de start inesperado: %v", err)
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

func TestRuntimeOrderStopCrossProjectConSesionFantasmaSeCompletaSinHandleNiRuntime(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	_, err = UpsertProyecto(&Proyecto{
		Slug:    "plugins",
		Nombre:  "Plugins",
		RutaAbs: filepath.Join(tmp, "plugins"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}
	pidMuerto := int64(999997)
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoA,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		PID:         &pidMuerto,
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	old := time.Now().UTC().Add(-2 * time.Hour)
	if _, err := DB.Exec(`UPDATE sesiones SET heartbeat_at=? WHERE id=?`, old, sesion.ID); err != nil {
		t.Fatalf("marcar heartbeat stale: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado', runtime_id=NULL WHERE sesion_id=?`, sesion.ID); err != nil {
		t.Fatalf("cerrar handle residual: %v", err)
	}
	if _, err := DB.Exec(`DELETE FROM runtime_instances WHERE sesion_id=?`, sesion.ID); err != nil {
		t.Fatalf("borrar runtime: %v", err)
	}

	stopID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex2",
		ProyectoID:  &proyectoA,
		Tipo:        "stop",
		PayloadJSON: `{"motivo":"stop cross-project stale"}`,
	})
	if err != nil {
		t.Fatalf("encolar stop: %v", err)
	}
	stopOrder, err := GetRuntimeOrder(stopID)
	if err != nil {
		t.Fatalf("get stop order: %v", err)
	}
	if err := ejecutarRuntimeOrderStop(stopOrder); err != nil {
		t.Fatalf("ejecutar stop cross-project stale: %v", err)
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
	if got, _ := result["phantom_session_closed"].(bool); !got {
		t.Fatalf("faltaba phantom_session_closed en resultado: %+v", result)
	}
	sesionAbierta, err := GetSesionAbierta("Codex2", &proyectoA)
	if err != nil && err != sql.ErrNoRows {
		t.Fatalf("get sesion abierta tras stop: %v", err)
	}
	if sesionAbierta != nil {
		t.Fatalf("la sesion fantasma debia quedar cerrada: %+v", sesionAbierta)
	}
}

func TestEjecutarRuntimeOrderStartRechazaAgenteRetirado(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexRetirado", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := AgenteSetHabilitado("CodexRetirado", false); err != nil {
		t.Fatalf("retirar agente: %v", err)
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
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "CodexRetirado",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"accion":"start","por":"orquesta","proyecto":"orquestador"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order: %v", err)
	}
	err = ejecutarRuntimeOrderStart(startOrder)
	if err == nil {
		t.Fatalf("esperaba rechazo de agente retirado")
	}
	if !strings.Contains(err.Error(), "retirado") && !strings.Contains(err.Error(), "deshabilitado") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestCancelarPendientesRuntimeAlDetenerCancelaTrabajoPendienteDelAgente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("GemmaStop", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto",
		Nombre:  "Proyecto",
		RutaAbs: filepath.Join(tmp, "proyecto"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "GemmaStop",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "proyecto"),
		Herramienta:       "ollama_pool_local",
		ExternalSessionID: "ollama-pool-gemmastop-1",
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
	msgID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "GemmaStop",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"haz algo"}`,
	})
	if err != nil {
		t.Fatalf("enviar mailbox: %v", err)
	}
	orderPendienteID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "GemmaStop",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"texto":"haz algo","from_agente":"server","to_agente":"GemmaStop","mailbox_id":%d,"mailbox_kind":"instruction"}`, msgID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando' WHERE id=?`, orderPendienteID); err != nil {
		t.Fatalf("marcar runtime order ejecutando: %v", err)
	}
	stopID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "GemmaStop",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "stop",
		PayloadJSON: `{"motivo":"stop limpio"}`,
	})
	if err != nil {
		t.Fatalf("encolar stop: %v", err)
	}
	stopOrder, err := GetRuntimeOrder(stopID)
	if err != nil {
		t.Fatalf("get stop order: %v", err)
	}
	if err := cancelarPendientesRuntimeAlDetener(stopOrder); err != nil {
		t.Fatalf("cancelar pendientes al detener: %v", err)
	}
	mailbox, err := GetRuntimeMailbox(msgID)
	if err != nil {
		t.Fatalf("get mailbox: %v", err)
	}
	if mailbox == nil || mailbox.Estado != "cancelado" {
		t.Fatalf("mailbox deberia quedar cancelado: %+v", mailbox)
	}
	orderPendiente, err := GetRuntimeOrder(orderPendienteID)
	if err != nil {
		t.Fatalf("get runtime order pendiente: %v", err)
	}
	if orderPendiente == nil || orderPendiente.Estado != "cancelada" {
		t.Fatalf("runtime order pendiente deberia quedar cancelada: %+v", orderPendiente)
	}
	if !strings.Contains(orderPendiente.ErrorText, "runtime_stop_cancelled_pending_work") {
		t.Fatalf("runtime order cancelada sin motivo esperado: %+v", orderPendiente)
	}
}

func TestRuntimeProcesoLocalYaNoViveUsaWorkerEstructuradoRunningAntesQuePID(t *testing.T) {
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

	workerDir := filepath.Join(tmp, "worker-running")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir worker dir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	manifestData, _ := json.Marshal(runtimeagente.WorkerManifest{
		Version:       1,
		Agent:         "Codex1",
		Project:       "orquestador",
		Driver:        "tmux_cli_session",
		Transport:     "tmux",
		CreatedAt:     now.Format(time.RFC3339),
		StartedAt:     now.Format(time.RFC3339),
		ChildPID:      4242,
		StatusPath:    statusPath,
		HeartbeatPath: heartbeatPath,
	})
	statusData, _ := json.Marshal(runtimeagente.WorkerStatus{
		State:     "running",
		UpdatedAt: now.Format(time.RFC3339),
		Alive:     true,
		ChildPID:  4242,
		Agent:     "Codex1",
		Project:   "orquestador",
	})
	heartbeatData, _ := json.Marshal(runtimeagente.WorkerHeartbeat{
		Alive:       true,
		HeartbeatAt: now.Format(time.RFC3339),
		StartedAt:   now.Format(time.RFC3339),
		ChildPID:    4242,
		Agent:       "Codex1",
		Project:     "orquestador",
	})
	for path, data := range map[string][]byte{
		manifestPath:  manifestData,
		statusPath:    statusData,
		heartbeatPath: heartbeatData,
	} {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write worker artifact %s: %v", path, err)
		}
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?,?,?)`,
		"Codex1", proyectoID, "tmux", "session", "orq-codex1-running", "activo", string(metaJSON), now)

	order := &RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		HandleID:   &handleID,
		Tipo:       "stop",
	}
	yaNoVive, pid, err := runtimeProcesoLocalYaNoVive(order)
	if err != nil {
		t.Fatalf("runtimeProcesoLocalYaNoVive: %v", err)
	}
	if yaNoVive {
		t.Fatal("worker running no deberia darse por muerto")
	}
	if pid != 4242 {
		t.Fatalf("pid estructurado inesperado: %d", pid)
	}
}

func TestRuntimeProcesoLocalYaNoViveUsaWorkerEstructuradoStoppedAntesQuePID(t *testing.T) {
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

	workerDir := filepath.Join(tmp, "worker-stopped")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir worker dir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	manifestData, _ := json.Marshal(runtimeagente.WorkerManifest{
		Version:       1,
		Agent:         "Codex1",
		Project:       "orquestador",
		Driver:        "tmux_cli_session",
		Transport:     "tmux",
		CreatedAt:     now.Add(-time.Minute).Format(time.RFC3339),
		StartedAt:     now.Add(-time.Minute).Format(time.RFC3339),
		ChildPID:      4343,
		StatusPath:    statusPath,
		HeartbeatPath: heartbeatPath,
	})
	statusData, _ := json.Marshal(runtimeagente.WorkerStatus{
		State:     "stopped",
		UpdatedAt: now.Format(time.RFC3339),
		Alive:     false,
		ChildPID:  4343,
		Agent:     "Codex1",
		Project:   "orquestador",
	})
	heartbeatData, _ := json.Marshal(runtimeagente.WorkerHeartbeat{
		Alive:       false,
		HeartbeatAt: now.Format(time.RFC3339),
		StartedAt:   now.Add(-time.Minute).Format(time.RFC3339),
		ChildPID:    4343,
		Agent:       "Codex1",
		Project:     "orquestador",
	})
	for path, data := range map[string][]byte{
		manifestPath:  manifestData,
		statusPath:    statusData,
		heartbeatPath: heartbeatData,
	} {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("write worker artifact %s: %v", path, err)
		}
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?,?,?)`,
		"Codex1", proyectoID, "tmux", "session", "orq-codex1-stopped", "cerrado", string(metaJSON), now)

	order := &RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		HandleID:   &handleID,
		Tipo:       "stop",
	}
	yaNoVive, pid, err := runtimeProcesoLocalYaNoVive(order)
	if err != nil {
		t.Fatalf("runtimeProcesoLocalYaNoVive: %v", err)
	}
	if !yaNoVive {
		t.Fatal("worker stopped deberia darse por muerto")
	}
	if pid != 4343 {
		t.Fatalf("pid estructurado inesperado: %d", pid)
	}
}

func TestObjetivoProcesoDiagnosticoParaOrdenNoReinyectaPIDEnCLITMUXPreferredSinHandle(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexDiag", "programador"); err != nil {
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
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexDiag",
		ProyectoID:   &proyectoID,
		Connector:    "codex-cli",
		LogicalState: "fallido",
		ProcessState: "fallido",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	order := &RuntimeOrder{
		Agente:     "CodexDiag",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtimeID,
		Tipo:       "stop",
	}

	obj, err := objetivoProcesoDiagnosticoParaOrden(order)
	if err != nil {
		t.Fatalf("objetivoProcesoDiagnosticoParaOrden: %v", err)
	}
	if obj.PID != nil {
		t.Fatalf("no deberia reinyectar PID para runtime codex-cli sin handle canonico: %+v", obj)
	}

	yaNoVive, gotPID, err := runtimeProcesoLocalYaNoVive(order)
	if err != nil {
		t.Fatalf("runtimeProcesoLocalYaNoVive: %v", err)
	}
	if yaNoVive || gotPID != 0 {
		t.Fatalf("sin handle canonico no deberia diagnosticar por PID en codex-cli: yaNoVive=%v pid=%d", yaNoVive, gotPID)
	}
}

func TestObjetivoProcesoDiagnosticoParaOrdenMantienePIDEnRuntimeLocalNoCLI(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("RunnerLocal", "programador"); err != nil {
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
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "RunnerLocal",
		ProyectoID:   &proyectoID,
		Connector:    "shell-local",
		LogicalState: "degradado",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	order := &RuntimeOrder{
		Agente:     "RunnerLocal",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtimeID,
		Tipo:       "stop",
	}

	obj, err := objetivoProcesoDiagnosticoParaOrden(order)
	if err != nil {
		t.Fatalf("objetivoProcesoDiagnosticoParaOrden: %v", err)
	}
	if obj.PID == nil || *obj.PID != pid {
		t.Fatalf("deberia conservar PID para runtime local no CLI: %+v", obj)
	}
}

func TestObjetivoProcesoDiagnosticoParaOrdenPrefiereRuntimeCanonicoSobreRuntimeIDViejo(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexDiag", "programador"); err != nil {
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
	now := time.Now().UTC().Format(time.RFC3339Nano)
	pid := int64(os.Getpid())

	runtimeLegacyID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexDiag",
		ProyectoID:   &proyectoID,
		Connector:    "codex-cli",
		LogicalState: "fallido",
		ProcessState: "fallido",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("registrar runtime legacy: %v", err)
	}
	legacyMetaJSON, _ := json.Marshal(map[string]any{
		"driver":           "process_pty_cli",
		"rendered_command": "codex-perfil CodexDiag",
	})
	legacyHandleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, CURRENT_TIMESTAMP)`,
		"CodexDiag", proyectoID, runtimeLegacyID, "cli", "process", strconv.Itoa(os.Getpid()), string(legacyMetaJSON))

	tmuxDir := filepath.Join(tmp, "tmux-diag")
	if err := os.MkdirAll(tmuxDir, 0o755); err != nil {
		t.Fatalf("mkdir tmux: %v", err)
	}
	manifestPath := filepath.Join(tmuxDir, "manifest.json")
	statusPath := filepath.Join(tmuxDir, "status.json")
	heartbeatPath := filepath.Join(tmuxDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexDiag","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexdiag-1","tmux_pane_id":"%41","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	runtimeTMUXID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexDiag",
		ProyectoID:   &proyectoID,
		Connector:    "codex-cli",
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("registrar runtime tmux: %v", err)
	}
	tmuxMetaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codexdiag-1",
		"tmux_pane_id":          "%41",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, CURRENT_TIMESTAMP)`,
		"CodexDiag", proyectoID, runtimeTMUXID, "tmux", "session", "orq-codexdiag-1", string(tmuxMetaJSON)); err != nil {
		t.Fatalf("insert handle tmux: %v", err)
	}

	order := &RuntimeOrder{
		Agente:     "CodexDiag",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtimeLegacyID,
		HandleID:   &legacyHandleID,
		Tipo:       "stop",
	}

	obj, err := objetivoProcesoDiagnosticoParaOrden(order)
	if err != nil {
		t.Fatalf("objetivoProcesoDiagnosticoParaOrden: %v", err)
	}
	meta := mapFromJSON(obj.MetadataJSON)
	if got := strings.TrimSpace(stringFromMap(meta, "driver", "")); got != "tmux_cli_session" {
		t.Fatalf("deberia diagnosticar contra el runtime/handle canónico tmux, got=%q meta=%+v", got, meta)
	}
	if obj.PID != nil {
		t.Fatalf("no deberia reinyectar PID desde el runtime legado si ya existe tmux canónico: %+v", obj)
	}
}

func TestObjetivoProcesoSendInstructionNoReinyectaPIDEnCLITMUXPreferredSinHandle(t *testing.T) {
	prepararDBTemporal(t)

	pid := int64(os.Getpid())
	order := &RuntimeOrder{Agente: "CodexSend", Tipo: "send_instruction"}
	runtime := &RuntimeInstance{
		Agente:    "CodexSend",
		Connector: "codex-cli",
		PID:       &pid,
	}

	obj := objetivoProcesoSendInstruction(order, nil, runtime, "/tmp/orquesta")
	if obj.PID != nil {
		t.Fatalf("no deberia reinyectar PID para send_instruction codex-cli sin handle canónico: %+v", obj)
	}
	meta := mapFromJSON(obj.MetadataJSON)
	if got := strings.TrimSpace(stringFromMap(meta, "working_dir", "")); got != "/tmp/orquesta" {
		t.Fatalf("working_dir inesperado: %q meta=%+v", got, meta)
	}
}

func TestObjetivoProcesoSendInstructionMantienePIDEnRuntimeLocalNoCLI(t *testing.T) {
	prepararDBTemporal(t)

	pid := int64(os.Getpid())
	order := &RuntimeOrder{Agente: "RunnerSend", Tipo: "send_instruction"}
	runtime := &RuntimeInstance{
		Agente:    "RunnerSend",
		Connector: "shell-local",
		PID:       &pid,
	}

	obj := objetivoProcesoSendInstruction(order, nil, runtime, "")
	if obj.PID == nil || *obj.PID != pid {
		t.Fatalf("deberia conservar PID para send_instruction local no CLI: %+v", obj)
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
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"/tmp/codex-perfiles/bin/codex-perfil Codex2","can_send_input":false}`,
	}) {
		t.Fatal("metadata can_send_input=false deberia desactivar input interactivo")
	}
	if RuntimeHandlePermiteSendInputInteractivo(&RuntimeHandle{
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"/tmp/codex-perfiles/bin/codex-perfil Codex2"}`,
	}) {
		t.Fatal("codex local via PTY deberia caer a mailbox por defecto aunque arrastre metadata legacy")
	}
	if !RuntimeHandlePermiteSendInputInteractivo(&RuntimeHandle{
		Transporte:   "cli",
		HandleKind:   "process",
		MetadataJSON: `{"driver":"process_pty_cli","stdin_path":"/tmp/pty.stdin","supervisor_ref":"/tmp/ref","rendered_command":"/tmp/codex-perfiles/bin/codex-perfil Codex2","mailbox_delivery_mode":"session_resume","can_send_input":false}`,
	}) {
		t.Fatal("codex process_pty_cli con stdin_path y sin external_session_id deberia admitir input interactivo transitorio")
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
	if !RuntimeHandlePermiteSendInputInteractivo(&RuntimeHandle{
		Transporte:       "tmux",
		HandleKind:       "session",
		HandleRef:        "orq-codex3/%43",
		CapabilitiesJSON: `{"can_send_input":false,"mailbox_delivery_mode":"session_resume"}`,
		MetadataJSON:     `{"driver":"tmux_cli_session","tmux_session":"orq-codex3","tmux_pane_id":"%43","mailbox_delivery_mode":"session_resume","can_send_input":false}`,
	}) {
		t.Fatal("tmux session_resume con pane real deberia admitir input interactivo aunque arrastre can_send_input=false legacy")
	}
	t.Setenv("ORQUESTA_ALLOW_LEGACY_INTERACTIVE_INPUT", "1")
	if RuntimeHandlePermiteSendInputInteractivo(&RuntimeHandle{
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"claude-perfil Claude1","stdin_path":"/tmp/pty.stdin"}`,
	}) {
		t.Fatal("un CLI local tmux-preferred no deberia volver a input interactivo legacy aunque exista opt-in global")
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
		Transporte:       "cli",
		HandleKind:       "process",
		CapabilitiesJSON: `{"mailbox_delivery_mode":"bootstrap_only","can_send_input":false}`,
		MetadataJSON:     `{"driver":"process_pty_cli","stdin_path":"/tmp/pty.stdin","supervisor_ref":"/tmp/ref","rendered_command":"codex-perfil Codex2","external_session_id":"sess-live-2","modo_plan":"resume"}`,
	}); got != runtimeagente.MailboxDeliverySessionResume {
		t.Fatalf("codex supervisado con external_session_id deberia elevarse a session_resume: %s", got)
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
	}); got != runtimeagente.MailboxDeliveryBootstrapOnly {
		t.Fatalf("codex local no interactivo no deberia elevarse a session_resume aunque conserve external_session_id: %s", got)
	}
	if got := RuntimeHandleMailboxDeliveryMode(&RuntimeHandle{
		Transporte:       "tmux",
		HandleKind:       "session",
		CapabilitiesJSON: `{"mailbox_delivery_mode":"session_resume","can_send_input":false}`,
		MetadataJSON:     `{"driver":"tmux_cli_session","mailbox_delivery_mode":"session_resume","can_send_input":false}`,
	}); got != runtimeagente.MailboxDeliveryBootstrapOnly {
		t.Fatalf("session_resume sin external_session_id deberia degradarse a bootstrap_only: %s", got)
	}
	if got := RuntimeHandleMailboxDeliveryMode(&RuntimeHandle{
		Transporte:       "cli",
		HandleKind:       "process",
		CapabilitiesJSON: `{"mailbox_delivery_mode":"session_resume","can_send_input":false}`,
		MetadataJSON:     `{"driver":"process_pty_cli","stdin_path":"/tmp/pty.stdin","supervisor_ref":"/tmp/ref","rendered_command":"codex-perfil Codex2","mailbox_delivery_mode":"session_resume","can_send_input":false}`,
	}); got != runtimeagente.MailboxDeliveryBootstrapOnly {
		t.Fatalf("codex process_pty_cli sin external_session_id deberia forzar bootstrap_only para recuperar TMUX canonico: %s", got)
	}
	if got := RuntimeHandleMailboxDeliveryMode(&RuntimeHandle{
		Transporte:       "tmux",
		HandleKind:       "session",
		CapabilitiesJSON: `{"mailbox_delivery_mode":"bootstrap_only","can_send_input":false}`,
		MetadataJSON:     `{"driver":"tmux_cli_session","mailbox_delivery_mode":"bootstrap_only","external_session_id":"sess-live-tmux","can_send_input":false}`,
	}); got != runtimeagente.MailboxDeliverySessionResume {
		t.Fatalf("tmux bootstrap_only con external_session_id deberia elevarse a session_resume: %s", got)
	}
	if got := RuntimeHandleMailboxDeliveryMode(&RuntimeHandle{
		Transporte:       "cli",
		HandleKind:       "process",
		CapabilitiesJSON: `{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`,
		MetadataJSON:     `{"driver":"process_pty_cli","stdin_path":"/tmp/pty.stdin","rendered_command":"cat-cli Codex2","can_send_input":true,"mailbox_delivery_mode":"interactive"}`,
	}); got != runtimeagente.MailboxDeliveryInteractive {
		t.Fatalf("un process_pty_cli generico con stdin interactivo deberia conservar interactive: %s", got)
	}
	if got := RuntimeHandleMailboxDeliveryMode(&RuntimeHandle{
		Transporte:       "tmux",
		HandleKind:       "session",
		CapabilitiesJSON: `{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`,
		MetadataJSON:     `{"driver":"tmux_cli_session","rendered_command":"cat-cli Codex1","stdin_path":"/tmp/tmux.stdin","can_send_input":true,"mailbox_delivery_mode":"interactive"}`,
	}); got != runtimeagente.MailboxDeliveryInteractive {
		t.Fatalf("un comando con nombre de agente Codex1 no deberia confundirse con codex-cli: %s", got)
	}
}

func TestRuntimeHandleSolicitaSessionResumeRespetaModoDerivado(t *testing.T) {
	handle := &RuntimeHandle{
		CapabilitiesJSON: `{"mailbox_delivery_mode":"bootstrap_only","can_send_input":false}`,
		MetadataJSON:     `{"driver":"tmux_cli_session","transport":"tmux","handle_kind":"session","mailbox_delivery_mode":"bootstrap_only","can_send_input":false}`,
	}
	if !runtimeHandleSolicitaSessionResume(handle, runtimeagente.MailboxDeliverySessionResume) {
		t.Fatal("el modo derivado session_resume debe prevalecer aunque la metadata vieja siga en bootstrap_only")
	}
	if runtimeHandleSolicitaSessionResume(handle, runtimeagente.MailboxDeliveryBootstrapOnly) {
		t.Fatal("sin modo derivado session_resume no deberia solicitarse session_resume")
	}
}

func TestRuntimeOrderSendInstructionHaceFallbackAMailboxCuandoHandleNoAdmiteInputInteractivo(t *testing.T) {
	enableLegacyPTYLocalRuntimeForTest(t)
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

func TestRuntimeOrderSendInstructionSinInputInteractivoCaeAMailboxAunqueExistaSupervisorLegacy(t *testing.T) {
	enableLegacyPTYLocalRuntimeForTest(t)
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
	if !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_id"`) {
		t.Fatalf("resultado sin fallback a mailbox: %s", sendOrder.ResultadoJSON)
	}
	time.Sleep(150 * time.Millisecond)
	if data, err := os.ReadFile(logPath); err == nil && strings.Contains(string(data), "hola supervisor local") {
		t.Fatalf("el proceso no deberia recibir stdin legado: %s", string(data))
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
		"rendered_command":      "/tmp/codex-perfiles/bin/codex-perfil Codex1",
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
	enableLegacyPTYLocalRuntimeForTest(t)
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
	if sendOrder.Estado != "completada" || !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_id"`) {
		t.Fatalf("codex sin input interactivo no deberia usar supervisor_local: %+v", sendOrder)
	}
	if strings.TrimSpace(logPath) != "" {
		time.Sleep(150 * time.Millisecond)
		if data, err := os.ReadFile(logPath); err == nil && strings.Contains(string(data), "hola codex mailbox seguro") {
			t.Fatalf("codex sin input interactivo no deberia recibir stdin local: %s", string(data))
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
		"driver":              "process_pty_cli",
		"rendered_command":    "'" + wrapper + "' 'Codex1'",
		"working_dir":         workingDir,
		"external_session_id": "sess-degradada-123",
		"stdin_path":          filepath.Join(tmp, "pty.stdin"),
		"supervisor_ref":      filepath.Join(tmp, "supervisor.ref"),
		"can_send_input":      false,
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
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle final: %+v err=%v", handle, err)
	}
	if handle.Estado != "activo" {
		t.Fatalf("session_resume deberia reactivar el handle pausado: %+v", handle)
	}
	runtime, err = GetRuntime(runtime.ID)
	if err != nil || runtime == nil {
		t.Fatalf("reload runtime final: %+v err=%v", runtime, err)
	}
	if strings.TrimSpace(runtime.LogicalState) != "activo" {
		t.Fatalf("session_resume deberia reactivar el runtime: %+v", runtime)
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

func TestResolverBootstrapRuntimeLeasePendienteReconoceStartConMailbox(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"microciclo premium"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d}`,
			mailboxID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start lease: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != startID {
		t.Fatalf("lease bootstrap inesperada: %+v", order)
	}
	if startOrderID != 0 {
		t.Fatalf("start_order_id inesperado para start bootstrap: %d", startOrderID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != mailboxID {
		t.Fatalf("mailbox_ids inesperados: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendienteDevuelveMailboxIDsIndependientes(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"microciclo premium"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d}`,
			mailboxID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start lease: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start ejecutando: %v", err)
	}

	_, _, mailboxIDs, _, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente primer ciclo: %v", err)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != mailboxID {
		t.Fatalf("mailbox_ids iniciales inesperados: %+v", mailboxIDs)
	}

	mailboxIDs[0] = 0

	_, _, mailboxIDsAgain, _, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente segundo ciclo: %v", err)
	}
	if len(mailboxIDsAgain) != 1 || mailboxIDsAgain[0] != mailboxID {
		t.Fatalf("mailbox_ids deberian recalcularse sin aliasing: %+v", mailboxIDsAgain)
	}
}

func TestResolverBootstrapRuntimeLeasePendienteReconoceStartPendienteConLeaseVigente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"microciclo premium"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "pendiente",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d}`,
			mailboxID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start lease: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != startID {
		t.Fatalf("lease bootstrap pendiente inesperada: %+v", order)
	}
	if startOrderID != 0 {
		t.Fatalf("start_order_id inesperado para start bootstrap pendiente: %d", startOrderID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != mailboxID {
		t.Fatalf("mailbox_ids inesperados: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendienteNoNormalizaLegacyProcessPTYComoTMUX(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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
	metaJSON := fmt.Sprintf(`{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"%s","external_session_id":"sess-legacy"}`, workingDir)
	if _, err := DB.Exec(`UPDATE runtime_handles SET transporte='cli', handle_kind='process', handle_ref='legacy-codex1', metadata_json=?, last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, metaJSON, handle.ID); err != nil {
		t.Fatalf("marcar handle legacy: %v", err)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"texto":"continua trabajo actual"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "pendiente",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start lease: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != startID {
		t.Fatalf("lease bootstrap pendiente inesperada: %+v", order)
	}
	if startOrderID != 0 {
		t.Fatalf("start_order_id inesperado: %d", startOrderID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != mailboxID {
		t.Fatalf("mailbox_ids inesperados: %+v", mailboxIDs)
	}

	persistedHandle, err := GetRuntimeHandle(handle.ID)
	if err != nil || persistedHandle == nil {
		t.Fatalf("persisted handle: %+v err=%v", persistedHandle, err)
	}
	if strings.TrimSpace(persistedHandle.Transporte) != "cli" || strings.TrimSpace(persistedHandle.HandleKind) != "process" {
		t.Fatalf("resolver lease no deberia normalizar handle legacy a tmux: %+v", persistedHandle)
	}
}

func TestRuntimeMailboxCubiertoPorBootstrapPendienteIgnoraLeaseDeliveredSinConsumo(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"microciclo premium"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "completada",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"delivered","mailbox_ids":[%d],"sesion_id":%d}`,
			mailboxID, sesion.ID,
		),
	}); err != nil {
		t.Fatalf("encolar start delivered: %v", err)
	}

	covered, _, _, err := RuntimeMailboxCubiertoPorBootstrapPendiente(mailboxID, handle, runtime)
	if err != nil {
		t.Fatalf("RuntimeMailboxCubiertoPorBootstrapPendiente: %v", err)
	}
	if covered {
		t.Fatal("una lease delivered sin consumo no deberia seguir bloqueando el mailbox pendiente")
	}
}

func TestRuntimeMailboxCubiertoPorBootstrapPendienteIgnoraLeaseConReceiptUtilAunqueSigaWaitingForEvidence(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"microciclo premium"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "completada",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","delivery_state":"delivered","delivery_receipt_at":"2026-04-14T12:37:01Z","receipt_source":"git_worktree","mailbox_ids":[%d],"sesion_id":%d}`,
			mailboxID, sesion.ID,
		),
	}); err != nil {
		t.Fatalf("encolar start delivered: %v", err)
	}

	covered, _, _, err := RuntimeMailboxCubiertoPorBootstrapPendiente(mailboxID, handle, runtime)
	if err != nil {
		t.Fatalf("RuntimeMailboxCubiertoPorBootstrapPendiente: %v", err)
	}
	if covered {
		t.Fatal("una lease con receipt util no deberia seguir bloqueando el mailbox pendiente aunque lease_state siga waiting_for_evidence")
	}
}

func TestResolverBootstrapRuntimeLeasePendienteReconoceStartCompletadaConLeasePendiente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"microciclo premium"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d}`,
			mailboxID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start lease: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start completada: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != startID {
		t.Fatalf("lease bootstrap inesperada: %+v", order)
	}
	if startOrderID != 0 {
		t.Fatalf("start_order_id inesperado para start bootstrap: %d", startOrderID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != mailboxID {
		t.Fatalf("mailbox_ids inesperados: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendienteReconoceStartCompletadaConLeaseStateCaseInsensitive(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"microciclo premium"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"Waiting_For_Evidence","mailbox_ids":[%d],"sesion_id":%d}`,
			mailboxID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start lease: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start completada: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != startID {
		t.Fatalf("lease bootstrap inesperada: %+v", order)
	}
	if startOrderID != 0 {
		t.Fatalf("start_order_id inesperado para start bootstrap: %d", startOrderID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != mailboxID {
		t.Fatalf("mailbox_ids inesperados: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendientePrefiereLeaseMasRecienteAunqueSeaCompletada(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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

	oldMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap viejo"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox vieja: %v", err)
	}
	oldStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d}`,
			oldMailboxID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start vieja: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, oldStartID); err != nil {
		t.Fatalf("marcar start vieja ejecutando: %v", err)
	}

	newMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap vigente"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox nueva: %v", err)
	}
	newStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"delivered","mailbox_ids":[%d],"sesion_id":%d}`,
			newMailboxID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start nueva: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, newStartID); err != nil {
		t.Fatalf("marcar start nueva completada: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != newStartID {
		t.Fatalf("deberia priorizar la lease bootstrap mas reciente: %+v", order)
	}
	if startOrderID != 0 {
		t.Fatalf("start_order_id inesperado para start bootstrap: %d", startOrderID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != newMailboxID {
		t.Fatalf("mailbox_ids inesperados: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendientePrefiereBootstrapFuenteSobreResumeDerivado(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap fuente vigente"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start bootstrap: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start ejecutando: %v", err)
	}

	resumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			startID, mailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume derivado: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != startID {
		t.Fatalf("deberia priorizar la bootstrap fuente y no el resume derivado: %+v", order)
	}
	if startOrderID != 0 {
		t.Fatalf("start_order_id inesperado para bootstrap fuente: %d", startOrderID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != mailboxID {
		t.Fatalf("mailbox_ids inesperados: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendientePrefiereStartMasRecienteSobreResumeViejoConMismaAfinidad(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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

	oldMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"resume viejo"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox vieja: %v", err)
	}
	oldResumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			oldMailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume vieja: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, oldResumeID); err != nil {
		t.Fatalf("marcar resume vieja ejecutando: %v", err)
	}

	newMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"start vigente"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox nueva: %v", err)
	}
	newStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			newMailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start nueva: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, newStartID); err != nil {
		t.Fatalf("marcar start nueva ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != newStartID {
		t.Fatalf("deberia priorizar la lease mas reciente cuando la afinidad contextual es la misma: %+v", order)
	}
	if startOrderID != 0 {
		t.Fatalf("start_order_id inesperado para start bootstrap: %d", startOrderID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != newMailboxID {
		t.Fatalf("mailbox_ids inesperados: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendientePrefiereLeaseLigadaAlHandleActual(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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

	currentMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap ligado al handle actual"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox actual: %v", err)
	}
	currentStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d,"handle_id":%d,"runtime_id":%d}`,
			currentMailboxID, sesion.ID, handle.ID, runtime.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start actual: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, currentStartID); err != nil {
		t.Fatalf("marcar start actual ejecutando: %v", err)
	}

	otroRuntimeID := mustInsertID(t, `INSERT INTO runtime_instances (
		agente, proyecto_id, logical_state, process_state, created_at, updated_at
	) VALUES (?, ?, 'starting', 'running', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, "Claude1", proyectoID)
	otroHandleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, created_at, updated_at
	) VALUES (?, ?, ?, 'cli', 'process', 'other-handle', 'activo', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		"Claude1", proyectoID, otroRuntimeID)
	foreignMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap ajeno mas reciente"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox ajena: %v", err)
	}
	foreignStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &otroRuntimeID,
		HandleID:   &otroHandleID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"handle_id":%d,"runtime_id":%d}`,
			foreignMailboxID, otroHandleID, otroRuntimeID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start ajena: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, foreignStartID); err != nil {
		t.Fatalf("marcar start ajena ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != currentStartID {
		t.Fatalf("deberia priorizar la lease ligada al handle actual: %+v", order)
	}
	if startOrderID != 0 {
		t.Fatalf("start_order_id inesperado para start bootstrap: %d", startOrderID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != currentMailboxID {
		t.Fatalf("mailbox_ids inesperados: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendientePrefiereLeaseDeSesionActualSobreLeaseSinSesion(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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

	sessionMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap de la sesion actual"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox actual: %v", err)
	}
	sessionStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d}`,
			sessionMailboxID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start actual: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, sessionStartID); err != nil {
		t.Fatalf("marcar start actual ejecutando: %v", err)
	}

	sessionlessMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"lease vieja sin sesion"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox sin sesion: %v", err)
	}
	sessionlessResumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d]}`,
			sessionlessMailboxID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume sin sesion: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, sessionlessResumeID); err != nil {
		t.Fatalf("marcar resume sin sesion ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != sessionStartID {
		t.Fatalf("deberia priorizar la lease ligada a la sesion actual: %+v", order)
	}
	if startOrderID != 0 {
		t.Fatalf("start_order_id inesperado para start bootstrap: %d", startOrderID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != sessionMailboxID {
		t.Fatalf("mailbox_ids inesperados: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendienteHeredaSesionDeResumeDerivadaDesdeStartFuente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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

	sourceMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap fuente de la sesion actual"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox fuente: %v", err)
	}
	sourceStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "completada",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"acked","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			sourceMailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente: %v", err)
	}

	resumeMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"resume derivada vigente"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox resume: %v", err)
	}
	resumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"mailbox_ids":[%d]}`,
			sourceStartID, resumeMailboxID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume derivada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume derivada ejecutando: %v", err)
	}

	foreignMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"lease ajena mas reciente sin sesion"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox ajena: %v", err)
	}
	foreignResumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d]}`,
			foreignMailboxID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume ajena: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, foreignResumeID); err != nil {
		t.Fatalf("marcar resume ajena ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != resumeID {
		t.Fatalf("deberia priorizar la resume derivada que hereda la sesion del start fuente: %+v", order)
	}
	if startOrderID != sourceStartID {
		t.Fatalf("start_order_id inesperado: got=%d want=%d", startOrderID, sourceStartID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != resumeMailboxID {
		t.Fatalf("mailbox_ids inesperados: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendienteHeredaMailboxIDsDesdeStartFuente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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

	sourceMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap fuente vigente"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox fuente: %v", err)
	}
	sourceStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"acked","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			sourceMailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, sourceStartID); err != nil {
		t.Fatalf("marcar start fuente completada: %v", err)
	}

	resumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d}`,
			sourceStartID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume derivada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume derivada ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != resumeID {
		t.Fatalf("deberia priorizar la lease derivada vigente: %+v", order)
	}
	if startOrderID != sourceStartID {
		t.Fatalf("start_order_id inesperado: got=%d want=%d", startOrderID, sourceStartID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != sourceMailboxID {
		t.Fatalf("mailbox_ids heredados inesperados: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendienteRecuperaMailboxVigenteDesdeStartFuenteSiResumeTraeMailboxConsumida(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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

	staleMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"resume stale"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox stale: %v", err)
	}
	if err := MarcarRuntimeMailboxConsumido(staleMailboxID); err != nil {
		t.Fatalf("consumir mailbox stale: %v", err)
	}

	sourceMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap fuente vigente"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox fuente: %v", err)
	}
	sourceStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"acked","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			sourceMailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, sourceStartID); err != nil {
		t.Fatalf("marcar start fuente completada: %v", err)
	}

	resumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"mailbox_ids":[%d]}`,
			sourceStartID, staleMailboxID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume derivada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume derivada ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != resumeID {
		t.Fatalf("deberia priorizar la lease derivada vigente: %+v", order)
	}
	if startOrderID != sourceStartID {
		t.Fatalf("start_order_id inesperado: got=%d want=%d", startOrderID, sourceStartID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != sourceMailboxID {
		t.Fatalf("mailbox_ids recuperados inesperados: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendienteParaMailboxRecuperaMailboxVigenteDesdeStartFuenteSiResumeTraeMailboxConsumida(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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

	staleMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"resume stale"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox stale: %v", err)
	}
	if err := MarcarRuntimeMailboxConsumido(staleMailboxID); err != nil {
		t.Fatalf("consumir mailbox stale: %v", err)
	}

	sourceMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap fuente vigente"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox fuente: %v", err)
	}
	sourceStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"acked","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			sourceMailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, sourceStartID); err != nil {
		t.Fatalf("marcar start fuente completada: %v", err)
	}

	resumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"mailbox_ids":[%d]}`,
			sourceStartID, staleMailboxID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume derivada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume derivada ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendienteParaMailbox(sourceMailboxID, handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendienteParaMailbox: %v", err)
	}
	if order == nil || order.ID != resumeID {
		t.Fatalf("deberia priorizar la lease derivada vigente: %+v", order)
	}
	if startOrderID != sourceStartID {
		t.Fatalf("start_order_id inesperado: got=%d want=%d", startOrderID, sourceStartID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != sourceMailboxID {
		t.Fatalf("mailbox_ids recuperados inesperados: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendienteIgnoraLeaseConMailboxConsumida(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap ya drenado"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxConsumido(mailboxID); err != nil {
		t.Fatalf("consumir mailbox: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"delivered","mailbox_ids":[%d],"sesion_id":%d}`,
			mailboxID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start lease: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start completada: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order != nil || startOrderID != 0 || len(mailboxIDs) != 0 {
		t.Fatalf("no deberia exponer una lease ya drenada: order=%+v start=%d mailbox=%+v", order, startOrderID, mailboxIDs)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
}

func TestResolverBootstrapRuntimeLeaseObservadaReconocePendienteConReceiptUtil(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap observada aun pendiente"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "pendiente",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","delivery_state":"delivered","delivery_receipt_at":"2026-04-14T12:37:01Z","receipt_source":"git_worktree","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start pendiente observada: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeaseObservada(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeaseObservada: %v", err)
	}
	if order == nil || order.ID != startID {
		t.Fatalf("deberia reconocer la start pendiente con receipt util: %+v", order)
	}
	if startOrderID != 0 {
		t.Fatalf("start_order_id inesperado para start fuente: got=%d", startOrderID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != mailboxID {
		t.Fatalf("mailbox_ids inesperados: %+v", mailboxIDs)
	}
}

func TestRuntimeMailboxCubiertoPorBootstrapPendienteReconoceMailboxDeLeaseAnteriorSiSigueVigente(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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

	oldMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap anterior"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox anterior: %v", err)
	}
	oldStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d,"handle_id":%d,"runtime_id":%d}`,
			oldMailboxID, sesion.ID, handle.ID, runtime.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start anterior: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, oldStartID); err != nil {
		t.Fatalf("marcar start anterior ejecutando: %v", err)
	}

	newMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap reciente"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox reciente: %v", err)
	}
	newStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d,"handle_id":%d,"runtime_id":%d}`,
			newMailboxID, sesion.ID, handle.ID, runtime.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start reciente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, newStartID); err != nil {
		t.Fatalf("marcar start reciente ejecutando: %v", err)
	}

	covered, bootstrapOrderID, startOrderID, err := RuntimeMailboxCubiertoPorBootstrapPendiente(oldMailboxID, handle, runtime)
	if err != nil {
		t.Fatalf("RuntimeMailboxCubiertoPorBootstrapPendiente: %v", err)
	}
	if !covered {
		t.Fatal("la mailbox anterior sigue cubierta por su lease bootstrap vigente")
	}
	if bootstrapOrderID != oldStartID || startOrderID != 0 {
		t.Fatalf("lease inesperada para mailbox anterior: bootstrap=%d start=%d", bootstrapOrderID, startOrderID)
	}
}

func TestResolverBootstrapRuntimeLeasePendienteParaMailboxPrefiereResumeDerivadaConMailboxHeredado(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap derivado pendiente"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	sourceStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"acked","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, sourceStartID); err != nil {
		t.Fatalf("marcar start fuente completada: %v", err)
	}

	resumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			sourceStartID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume derivada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume derivada ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendienteParaMailbox(mailboxID, handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendienteParaMailbox: %v", err)
	}
	if order == nil || order.ID != resumeID {
		t.Fatalf("deberia priorizar la resume derivada para mailbox pendiente: %+v", order)
	}
	if startOrderID != sourceStartID {
		t.Fatalf("start_order_id inesperado: got=%d want=%d", startOrderID, sourceStartID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != mailboxID {
		t.Fatalf("mailbox_ids heredados inesperados: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendienteParaMailboxAislaLeasesConcurrentesPorMailbox(t *testing.T) {
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
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}

	mailboxA, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"slice A"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox A: %v", err)
	}
	mailboxB, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"slice B"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox B: %v", err)
	}

	startA, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(`{"lease_state":"acked","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxA, sesion.ID, runtime.ID, handle.ID),
	})
	if err != nil {
		t.Fatalf("encolar start A: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startA); err != nil {
		t.Fatalf("marcar start A completada: %v", err)
	}
	resumeA, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(`{"lease_state":"waiting_for_evidence","start_order_id":%d,"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			startA, sesion.ID, runtime.ID, handle.ID),
	})
	if err != nil {
		t.Fatalf("encolar resume A: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeA); err != nil {
		t.Fatalf("marcar resume A ejecutando: %v", err)
	}

	startB, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(`{"lease_state":"acked","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxB, sesion.ID, runtime.ID, handle.ID),
	})
	if err != nil {
		t.Fatalf("encolar start B: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startB); err != nil {
		t.Fatalf("marcar start B completada: %v", err)
	}
	resumeB, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(`{"lease_state":"waiting_for_evidence","start_order_id":%d,"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			startB, sesion.ID, runtime.ID, handle.ID),
	})
	if err != nil {
		t.Fatalf("encolar resume B: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeB); err != nil {
		t.Fatalf("marcar resume B ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendienteParaMailbox(mailboxA, handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendienteParaMailbox mailboxA: %v", err)
	}
	if order == nil || order.ID != resumeA {
		t.Fatalf("deberia devolver lease de mailbox A, got=%+v want=%d", order, resumeA)
	}
	if startOrderID != startA {
		t.Fatalf("start_order_id inesperado para mailbox A: got=%d want=%d", startOrderID, startA)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != mailboxA {
		t.Fatalf("mailbox_ids inesperados para mailbox A: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendientePrefiereHandoffDerivadaConMailboxHeredado(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap derivada por handoff"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	sourceStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"acked","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, sourceStartID); err != nil {
		t.Fatalf("marcar start fuente completada: %v", err)
	}

	handoffPayload, err := json.Marshal(HandoffPayload{
		AgenteOrigen:       "Codex0",
		AgenteDestino:      "Codex1",
		ResumenContinuidad: "handoff derivada vigente",
	})
	if err != nil {
		t.Fatalf("marshal handoff: %v", err)
	}
	handoffID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "handoff",
		PayloadJSON: string(handoffPayload),
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"runtime_id":%d,"handle_id":%d}`,
			sourceStartID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar handoff derivada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, handoffID); err != nil {
		t.Fatalf("marcar handoff derivada ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != handoffID {
		t.Fatalf("deberia priorizar la handoff derivada vigente: %+v", order)
	}
	if startOrderID != sourceStartID {
		t.Fatalf("start_order_id inesperado: got=%d want=%d", startOrderID, sourceStartID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != mailboxID {
		t.Fatalf("mailbox_ids heredados inesperados: %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendientePrefiereHandoffDerivadaSobreBootstrapFuente(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"handoff derivada sobre start fuente"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start bootstrap: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start ejecutando: %v", err)
	}

	handoffPayload, err := json.Marshal(HandoffPayload{
		AgenteOrigen:       "Codex0",
		AgenteDestino:      "Codex1",
		ResumenContinuidad: "handoff derivada concurrente con start",
	})
	if err != nil {
		t.Fatalf("marshal handoff: %v", err)
	}
	handoffID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "handoff",
		PayloadJSON: string(handoffPayload),
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			startID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar handoff derivada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, handoffID); err != nil {
		t.Fatalf("marcar handoff derivada ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != handoffID {
		t.Fatalf("deberia priorizar la handoff derivada sobre la start fuente cuando ambas son pendientes: %+v", order)
	}
	if startOrderID != startID {
		t.Fatalf("start_order_id inesperado: got=%d want=%d", startOrderID, startID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != mailboxID {
		t.Fatalf("mailbox_ids heredados inesperados: %+v", mailboxIDs)
	}
}

func TestRuntimeMailboxEntregadoPorBootstrapObservadoReconoceMailboxDeLeaseAnterior(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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

	oldMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap observado anterior"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox anterior: %v", err)
	}
	oldStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "completada",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","delivery_state":"delivered","delivery_receipt_at":"2026-04-14T12:37:01Z","receipt_source":"git_worktree","mailbox_ids":[%d],"sesion_id":%d,"handle_id":%d,"runtime_id":%d}`,
			oldMailboxID, sesion.ID, handle.ID, runtime.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start anterior observado: %v", err)
	}

	newMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap observado reciente"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox reciente: %v", err)
	}
	newStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "completada",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","delivery_state":"delivered","delivery_receipt_at":"2026-04-14T12:38:01Z","receipt_source":"git_worktree","mailbox_ids":[%d],"sesion_id":%d,"handle_id":%d,"runtime_id":%d}`,
			newMailboxID, sesion.ID, handle.ID, runtime.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start reciente observado: %v", err)
	}

	observado, bootstrapOrderID, startOrderID, err := RuntimeMailboxEntregadoPorBootstrapObservado(oldMailboxID, handle, runtime)
	if err != nil {
		t.Fatalf("RuntimeMailboxEntregadoPorBootstrapObservado: %v", err)
	}
	if !observado {
		t.Fatal("la mailbox observada anterior deberia seguir resolviendo su lease bootstrap")
	}
	if bootstrapOrderID != oldStartID || startOrderID != 0 {
		t.Fatalf("lease observada inesperada para mailbox anterior: bootstrap=%d start=%d", bootstrapOrderID, startOrderID)
	}
	if newStartID == oldStartID {
		t.Fatal("el setup deberia crear leases distintas")
	}
}

func TestResolverBootstrapRuntimeLeaseObservadaParaMailboxPrefiereStartFuenteConReceiptUtil(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap observado derivado helper"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	sourceStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "completada",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","delivery_state":"delivered","delivery_receipt_at":"2026-04-14T12:37:01Z","receipt_source":"git_worktree","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente observada: %v", err)
	}

	resumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			sourceStartID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume derivada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume derivada ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeaseObservadaParaMailbox(mailboxID, handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeaseObservadaParaMailbox: %v", err)
	}
	if order == nil || order.ID != sourceStartID {
		t.Fatalf("deberia resolver la start fuente observada para mailbox observado: %+v", order)
	}
	if startOrderID != 0 {
		t.Fatalf("start_order_id inesperado para start fuente observada: got=%d", startOrderID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != mailboxID {
		t.Fatalf("mailbox_ids inesperados: %+v", mailboxIDs)
	}
	_ = resumeID
}

func TestRuntimeMailboxCubiertoPorBootstrapPendienteIgnoraLeaseDerivadaObservadaSinReceiptPropio(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap observado fuente"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	sourceStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "completada",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","delivery_state":"delivered","delivery_receipt_at":"2026-04-14T12:37:01Z","receipt_source":"git_worktree","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente observada: %v", err)
	}

	resumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			sourceStartID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume derivada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume derivada ejecutando: %v", err)
	}

	covered, bootstrapOrderID, startOrderID, err := RuntimeMailboxCubiertoPorBootstrapPendiente(mailboxID, handle, runtime)
	if err != nil {
		t.Fatalf("RuntimeMailboxCubiertoPorBootstrapPendiente: %v", err)
	}
	if covered {
		t.Fatalf("la lease derivada no deberia seguir bloqueando si la start fuente ya fue observada: bootstrap=%d start=%d", bootstrapOrderID, startOrderID)
	}
}

func TestResolverBootstrapRuntimeLeasePendienteIgnoraLeaseDerivadaSiStartFuenteYaFueObservada(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap observado fuente"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	sourceStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "completada",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","delivery_state":"delivered","delivery_receipt_at":"2026-04-14T12:37:01Z","receipt_source":"git_worktree","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente observada: %v", err)
	}

	resumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			sourceStartID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume derivada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume derivada ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order != nil || startOrderID != 0 || len(mailboxIDs) != 0 {
		t.Fatalf("no deberia exponer una lease pendiente si la start fuente ya fue observada: order=%+v start=%d mailbox=%+v", order, startOrderID, mailboxIDs)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
}

func TestResolverBootstrapRuntimeLeasePendienteIgnoraHandoffDerivadaSiStartFuenteYaFueObservada(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap observado fuente handoff"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	sourceStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "completada",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","delivery_state":"delivered","delivery_receipt_at":"2026-04-14T12:37:01Z","receipt_source":"git_worktree","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente observada: %v", err)
	}

	handoffPayload, err := json.Marshal(HandoffPayload{
		AgenteOrigen:       "Codex0",
		AgenteDestino:      "Codex1",
		ResumenContinuidad: "handoff derivada desde start ya observada",
	})
	if err != nil {
		t.Fatalf("marshal handoff: %v", err)
	}
	handoffID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "handoff",
		PayloadJSON: string(handoffPayload),
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			sourceStartID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar handoff derivada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, handoffID); err != nil {
		t.Fatalf("marcar handoff derivada ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order != nil || startOrderID != 0 || len(mailboxIDs) != 0 {
		t.Fatalf("no deberia exponer lease pendiente si la start fuente ya fue observada: order=%+v start=%d mailbox=%+v", order, startOrderID, mailboxIDs)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
}

func TestRuntimeMailboxEntregadoPorBootstrapObservadoReconoceLeaseDerivadaSinReceiptPropio(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap observado derivado"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	sourceStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "completada",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","delivery_state":"delivered","delivery_receipt_at":"2026-04-14T12:37:01Z","receipt_source":"git_worktree","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente observada: %v", err)
	}

	resumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			sourceStartID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume derivada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume derivada ejecutando: %v", err)
	}

	observado, bootstrapOrderID, startOrderID, err := RuntimeMailboxEntregadoPorBootstrapObservado(mailboxID, handle, runtime)
	if err != nil {
		t.Fatalf("RuntimeMailboxEntregadoPorBootstrapObservado: %v", err)
	}
	if !observado {
		t.Fatal("la lease derivada deberia heredar el receipt util de la start fuente")
	}
	if bootstrapOrderID != sourceStartID || startOrderID != 0 {
		t.Fatalf("lease observada inesperada: bootstrap=%d start=%d", bootstrapOrderID, startOrderID)
	}
	_ = resumeID
}

func TestRuntimeMailboxEntregadoPorBootstrapObservadoReconoceHandoffDerivadaSinReceiptPropio(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap observado derivado handoff"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	sourceStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "completada",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","delivery_state":"delivered","delivery_receipt_at":"2026-04-14T12:37:01Z","receipt_source":"git_worktree","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente observada: %v", err)
	}

	handoffPayload, err := json.Marshal(HandoffPayload{
		AgenteOrigen:       "Codex0",
		AgenteDestino:      "Codex1",
		ResumenContinuidad: "handoff derivada sin receipt propio",
	})
	if err != nil {
		t.Fatalf("marshal handoff: %v", err)
	}
	handoffID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "handoff",
		PayloadJSON: string(handoffPayload),
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			sourceStartID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar handoff derivada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, handoffID); err != nil {
		t.Fatalf("marcar handoff derivada ejecutando: %v", err)
	}

	observado, bootstrapOrderID, startOrderID, err := RuntimeMailboxEntregadoPorBootstrapObservado(mailboxID, handle, runtime)
	if err != nil {
		t.Fatalf("RuntimeMailboxEntregadoPorBootstrapObservado: %v", err)
	}
	if !observado {
		t.Fatal("la handoff derivada deberia heredar el receipt util de la start fuente")
	}
	if bootstrapOrderID != sourceStartID || startOrderID != 0 {
		t.Fatalf("lease observada inesperada: bootstrap=%d start=%d", bootstrapOrderID, startOrderID)
	}
	_ = handoffID
}

func TestResolverBootstrapRuntimeLeasePendienteRecuperaMailboxVigenteDesdeStartFuenteSiHandoffTraeMailboxConsumida(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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

	staleMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"handoff stale"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox stale: %v", err)
	}
	if err := MarcarRuntimeMailboxConsumido(staleMailboxID); err != nil {
		t.Fatalf("consumir mailbox stale: %v", err)
	}

	sourceMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap fuente vigente handoff"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox fuente: %v", err)
	}
	sourceStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"acked","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			sourceMailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, sourceStartID); err != nil {
		t.Fatalf("marcar start fuente completada: %v", err)
	}

	handoffPayload, err := json.Marshal(HandoffPayload{
		AgenteOrigen:       "Codex0",
		AgenteDestino:      "Codex1",
		ResumenContinuidad: "handoff derivada mailbox consumida",
	})
	if err != nil {
		t.Fatalf("marshal handoff: %v", err)
	}
	handoffID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "handoff",
		PayloadJSON: string(handoffPayload),
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"mailbox_ids":[%d]}`,
			sourceStartID, staleMailboxID,
		),
	})
	if err != nil {
		t.Fatalf("encolar handoff derivada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, handoffID); err != nil {
		t.Fatalf("marcar handoff derivada ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendiente: %v", err)
	}
	if order == nil || order.ID != handoffID {
		t.Fatalf("deberia priorizar la lease handoff derivada vigente: %+v", order)
	}
	if startOrderID != sourceStartID {
		t.Fatalf("start_order_id inesperado: got=%d want=%d", startOrderID, sourceStartID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != sourceMailboxID {
		t.Fatalf("mailbox_ids recuperados inesperados (deberian ser los de la start fuente): %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendienteParaMailboxRecuperaMailboxVigenteDesdeStartFuenteSiHandoffTraeMailboxConsumida(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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

	staleMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"handoff stale para-mailbox"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox stale: %v", err)
	}
	if err := MarcarRuntimeMailboxConsumido(staleMailboxID); err != nil {
		t.Fatalf("consumir mailbox stale: %v", err)
	}

	sourceMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"bootstrap fuente vigente handoff para-mailbox"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox fuente: %v", err)
	}
	sourceStartID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"acked","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			sourceMailboxID, sesion.ID, runtime.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, sourceStartID); err != nil {
		t.Fatalf("marcar start fuente completada: %v", err)
	}

	handoffPayload, err := json.Marshal(HandoffPayload{
		AgenteOrigen:       "Codex0",
		AgenteDestino:      "Codex1",
		ResumenContinuidad: "handoff derivada mailbox consumida para-mailbox",
	})
	if err != nil {
		t.Fatalf("marshal handoff: %v", err)
	}
	handoffID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "handoff",
		PayloadJSON: string(handoffPayload),
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"mailbox_ids":[%d]}`,
			sourceStartID, staleMailboxID,
		),
	})
	if err != nil {
		t.Fatalf("encolar handoff derivada: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, handoffID); err != nil {
		t.Fatalf("marcar handoff derivada ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendienteParaMailbox(sourceMailboxID, handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendienteParaMailbox: %v", err)
	}
	if order == nil || order.ID != handoffID {
		t.Fatalf("deberia priorizar la lease handoff derivada vigente: %+v", order)
	}
	if startOrderID != sourceStartID {
		t.Fatalf("start_order_id inesperado: got=%d want=%d", startOrderID, sourceStartID)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != sourceMailboxID {
		t.Fatalf("mailbox_ids recuperados inesperados (deberian ser los de la start fuente): %+v", mailboxIDs)
	}
}

func TestResolverBootstrapRuntimeLeasePendienteParaMailboxAislaHandoffsConcurrentesPorMailbox(t *testing.T) {
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
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}

	mailboxA, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"handoff A"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox A: %v", err)
	}
	mailboxB, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"handoff B"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox B: %v", err)
	}

	startA, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(`{"lease_state":"acked","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxA, sesion.ID, runtime.ID, handle.ID),
	})
	if err != nil {
		t.Fatalf("encolar start A: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startA); err != nil {
		t.Fatalf("marcar start A completada: %v", err)
	}
	handoffPayloadA, err := json.Marshal(HandoffPayload{
		AgenteOrigen:       "Codex0",
		AgenteDestino:      "Codex1",
		ResumenContinuidad: "handoff A",
	})
	if err != nil {
		t.Fatalf("marshal handoff A: %v", err)
	}
	handoffA, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "handoff",
		PayloadJSON: string(handoffPayloadA),
		ResultadoJSON: fmt.Sprintf(`{"lease_state":"waiting_for_evidence","start_order_id":%d,"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			startA, sesion.ID, runtime.ID, handle.ID),
	})
	if err != nil {
		t.Fatalf("encolar handoff A: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, handoffA); err != nil {
		t.Fatalf("marcar handoff A ejecutando: %v", err)
	}

	startB, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(`{"lease_state":"acked","mailbox_ids":[%d],"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			mailboxB, sesion.ID, runtime.ID, handle.ID),
	})
	if err != nil {
		t.Fatalf("encolar start B: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startB); err != nil {
		t.Fatalf("marcar start B completada: %v", err)
	}
	handoffPayloadB, err := json.Marshal(HandoffPayload{
		AgenteOrigen:       "Codex9",
		AgenteDestino:      "Codex1",
		ResumenContinuidad: "handoff B",
	})
	if err != nil {
		t.Fatalf("marshal handoff B: %v", err)
	}
	handoffB, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "handoff",
		PayloadJSON: string(handoffPayloadB),
		ResultadoJSON: fmt.Sprintf(`{"lease_state":"waiting_for_evidence","start_order_id":%d,"sesion_id":%d,"runtime_id":%d,"handle_id":%d}`,
			startB, sesion.ID, runtime.ID, handle.ID),
	})
	if err != nil {
		t.Fatalf("encolar handoff B: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, handoffB); err != nil {
		t.Fatalf("marcar handoff B ejecutando: %v", err)
	}

	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendienteParaMailbox(mailboxA, handle, runtime)
	if err != nil {
		t.Fatalf("resolverBootstrapRuntimeLeasePendienteParaMailbox mailboxA: %v", err)
	}
	if order == nil || order.ID != handoffA {
		t.Fatalf("deberia devolver handoff ligada a mailbox A, got=%+v want=%d", order, handoffA)
	}
	if startOrderID != startA {
		t.Fatalf("start_order_id inesperado para mailbox A: got=%d want=%d", startOrderID, startA)
	}
	if sesionID != sesion.ID {
		t.Fatalf("sesion_id inesperado: got=%d want=%d", sesionID, sesion.ID)
	}
	if len(mailboxIDs) != 1 || mailboxIDs[0] != mailboxA {
		t.Fatalf("mailbox_ids inesperados para mailbox A: %+v", mailboxIDs)
	}
}

func TestRuntimeBootstrapLeaseTieneReceiptUtil(t *testing.T) {
	cases := []struct {
		name   string
		result map[string]any
		want   bool
	}{
		{"vacio", map[string]any{}, false},
		{"nil", nil, false},
		{"receipt_source presente", map[string]any{"receipt_source": "git_worktree"}, true},
		{"receipt_source vacio", map[string]any{"receipt_source": ""}, false},
		{"delivery_receipt_at presente", map[string]any{"delivery_receipt_at": "2026-04-14T10:00:00Z"}, true},
		{"delivery_receipt_at vacio", map[string]any{"delivery_receipt_at": "  "}, false},
		{"delivery_state delivered", map[string]any{"delivery_state": "delivered"}, true},
		{"delivery_state DELIVERED mayusculas", map[string]any{"delivery_state": "DELIVERED"}, true},
		{"delivery_state waiting_for_evidence", map[string]any{"delivery_state": "waiting_for_evidence"}, false},
		{"delivery_state acked", map[string]any{"delivery_state": "acked"}, false},
	}
	for _, c := range cases {
		got := runtimeBootstrapLeaseTieneReceiptUtil(c.result)
		if got != c.want {
			t.Errorf("[%s] runtimeBootstrapLeaseTieneReceiptUtil = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestRuntimeBootstrapLeaseTieneCoberturaDeclarada(t *testing.T) {
	cases := []struct {
		name   string
		result map[string]any
		want   bool
	}{
		{"vacio", map[string]any{}, false},
		{"nil", nil, false},
		{"mailbox_ids presente", map[string]any{"mailbox_ids": []any{int64(1)}}, true},
		{"mailbox_ids slice vacio", map[string]any{"mailbox_ids": []any{}}, false},
		{"start_order_id presente", map[string]any{"start_order_id": int64(42)}, true},
		{"start_order_id cero", map[string]any{"start_order_id": int64(0)}, false},
		{"ambos presentes", map[string]any{"mailbox_ids": []any{int64(5)}, "start_order_id": int64(3)}, true},
	}
	for _, c := range cases {
		got := runtimeBootstrapLeaseTieneCoberturaDeclarada(c.result)
		if got != c.want {
			t.Errorf("[%s] runtimeBootstrapLeaseTieneCoberturaDeclarada = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestRuntimeBootstrapLeaseStateBlocksMailbox(t *testing.T) {
	wfe := map[string]any{"lease_state": "waiting_for_evidence"}
	wfeWithReceipt := map[string]any{"lease_state": "waiting_for_evidence", "receipt_source": "git_worktree"}
	acked := map[string]any{"lease_state": "acked"}
	inheritedReceipt := map[string]any{"receipt_source": "git_worktree"}

	cases := []struct {
		name      string
		result    map[string]any
		inherited map[string]any
		want      bool
	}{
		{"wfe sin receipt bloquea", wfe, nil, true},
		{"wfe con receipt propio no bloquea", wfeWithReceipt, nil, false},
		{"wfe con receipt heredado no bloquea", wfe, inheritedReceipt, false},
		{"acked no bloquea", acked, nil, false},
		{"vacio no bloquea", map[string]any{}, nil, false},
		{"nil no bloquea", nil, nil, false},
		{"wfe case insensitive", map[string]any{"lease_state": "Waiting_For_Evidence"}, nil, true},
	}
	for _, c := range cases {
		got := runtimeBootstrapLeaseStateBlocksMailbox(c.result, c.inherited)
		if got != c.want {
			t.Errorf("[%s] runtimeBootstrapLeaseStateBlocksMailbox = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestRuntimeOrderMantieneBootstrapLeasePendiente(t *testing.T) {
	cases := []struct {
		name   string
		result map[string]any
		want   bool
	}{
		{"wfe con mailbox_ids", map[string]any{"lease_state": "waiting_for_evidence", "mailbox_ids": []any{int64(1)}}, true},
		{"wfe con start_order_id", map[string]any{"lease_state": "waiting_for_evidence", "start_order_id": int64(7)}, true},
		{"delivered con cobertura", map[string]any{"lease_state": "delivered", "mailbox_ids": []any{int64(2)}}, true},
		{"DELIVERED mayusculas con cobertura", map[string]any{"lease_state": "DELIVERED", "mailbox_ids": []any{int64(2)}}, true},
		{"acked con cobertura", map[string]any{"lease_state": "acked", "mailbox_ids": []any{int64(1)}}, false},
		{"wfe sin cobertura", map[string]any{"lease_state": "waiting_for_evidence"}, false},
		{"vacio", map[string]any{}, false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		got := runtimeOrderMantieneBootstrapLeasePendiente(c.result)
		if got != c.want {
			t.Errorf("[%s] runtimeOrderMantieneBootstrapLeasePendiente = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestRuntimeBootstrapLeaseFromOrder(t *testing.T) {
	// nil order → zeros
	startID, mailboxIDs, sesionID := runtimeBootstrapLeaseFromOrder(nil)
	if startID != 0 || len(mailboxIDs) != 0 || sesionID != 0 {
		t.Errorf("nil order deberia devolver zeros: %d %v %d", startID, mailboxIDs, sesionID)
	}

	// resultado completo
	order := &RuntimeOrder{ResultadoJSON: `{"start_order_id":42,"mailbox_ids":[10,11],"sesion_id":7}`}
	startID, mailboxIDs, sesionID = runtimeBootstrapLeaseFromOrder(order)
	if startID != 42 {
		t.Errorf("start_order_id: got %d, want 42", startID)
	}
	if len(mailboxIDs) != 2 || mailboxIDs[0] != 10 || mailboxIDs[1] != 11 {
		t.Errorf("mailbox_ids: got %v, want [10 11]", mailboxIDs)
	}
	if sesionID != 7 {
		t.Errorf("sesion_id: got %d, want 7", sesionID)
	}

	// resultado vacío → zeros
	orderEmpty := &RuntimeOrder{ResultadoJSON: `{}`}
	startID, mailboxIDs, sesionID = runtimeBootstrapLeaseFromOrder(orderEmpty)
	if startID != 0 || len(mailboxIDs) != 0 || sesionID != 0 {
		t.Errorf("resultado vacio deberia devolver zeros: %d %v %d", startID, mailboxIDs, sesionID)
	}
}

func TestRuntimeBootstrapLeaseMailboxIDs(t *testing.T) {
	orderWithIDs := &RuntimeOrder{ResultadoJSON: `{"mailbox_ids":[5,6]}`}
	orderNoIDs := &RuntimeOrder{ResultadoJSON: `{}`}
	startWithIDs := &RuntimeOrder{ResultadoJSON: `{"mailbox_ids":[100,101]}`}

	// orden tiene propios → los devuelve
	got := runtimeBootstrapLeaseMailboxIDs(orderWithIDs, startWithIDs)
	if len(got) != 2 || got[0] != 5 || got[1] != 6 {
		t.Errorf("deberia devolver mailbox_ids propios: %v", got)
	}

	// orden sin propios, start con ids → hereda start
	got = runtimeBootstrapLeaseMailboxIDs(orderNoIDs, startWithIDs)
	if len(got) != 2 || got[0] != 100 || got[1] != 101 {
		t.Errorf("deberia heredar mailbox_ids de start: %v", got)
	}

	// orden sin propios y sin start → nil
	got = runtimeBootstrapLeaseMailboxIDs(orderNoIDs, nil)
	if got != nil {
		t.Errorf("sin start deberia devolver nil: %v", got)
	}

	// nil order + start con ids → hereda start (cae al fallback)
	got = runtimeBootstrapLeaseMailboxIDs(nil, startWithIDs)
	if len(got) != 2 || got[0] != 100 || got[1] != 101 {
		t.Errorf("nil order con start deberia heredar mailbox_ids de start: %v", got)
	}

	// nil order + nil start → nil
	got = runtimeBootstrapLeaseMailboxIDs(nil, nil)
	if got != nil {
		t.Errorf("nil order y nil start deberia devolver nil: %v", got)
	}
}

func TestRuntimeBootstrapLeaseSesionID(t *testing.T) {
	// nil order → 0
	if got := runtimeBootstrapLeaseSesionID(nil, nil); got != 0 {
		t.Errorf("nil order: got %d, want 0", got)
	}

	// sesion_id en resultado de order → lo usa directamente
	order := &RuntimeOrder{ResultadoJSON: `{"sesion_id":7}`}
	if got := runtimeBootstrapLeaseSesionID(order, nil); got != 7 {
		t.Errorf("sesion_id de order: got %d, want 7", got)
	}

	// sin sesion_id en order, con start que lo tiene → hereda start
	orderNoSesion := &RuntimeOrder{ResultadoJSON: `{}`}
	startWithSesion := &RuntimeOrder{ResultadoJSON: `{"sesion_id":99}`}
	if got := runtimeBootstrapLeaseSesionID(orderNoSesion, startWithSesion); got != 99 {
		t.Errorf("sesion_id heredada de start: got %d, want 99", got)
	}

	// sin sesion_id en ninguno → 0
	if got := runtimeBootstrapLeaseSesionID(orderNoSesion, nil); got != 0 {
		t.Errorf("sin sesion_id: got %d, want 0", got)
	}
}

func TestRuntimeBootstrapLeaseLinkedToStart(t *testing.T) {
	mkOrder := func(id int64, tipo, resultadoJSON string) *RuntimeOrder {
		return &RuntimeOrder{ID: id, Tipo: tipo, ResultadoJSON: resultadoJSON}
	}
	start10 := mkOrder(10, "start", `{}`)
	start10.ID = 10

	cases := []struct {
		name       string
		order      *RuntimeOrder
		startOrder *RuntimeOrder
		want       bool
	}{
		{"order nil", nil, start10, false},
		{"startOrder nil", mkOrder(1, "resume", `{"start_order_id":10}`), nil, false},
		{"startOrder tipo no es start", mkOrder(1, "resume", `{"start_order_id":10}`), mkOrder(10, "resume", `{}`), false},
		{"start_order_id coincide", mkOrder(1, "resume", `{"start_order_id":10}`), start10, true},
		{"start_order_id no coincide", mkOrder(1, "resume", `{"start_order_id":99}`), start10, false},
		{"handoff linked", mkOrder(2, "handoff", `{"start_order_id":10}`), start10, true},
		{"sin start_order_id en resultado", mkOrder(3, "resume", `{}`), start10, false},
	}
	for _, c := range cases {
		got := runtimeBootstrapLeaseLinkedToStart(c.order, c.startOrder)
		if got != c.want {
			t.Errorf("[%s] runtimeBootstrapLeaseLinkedToStart = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestRuntimeBootstrapLeaseTipoPrefiereStart(t *testing.T) {
	mkOrder := func(id int64, tipo, resultadoJSON string) *RuntimeOrder {
		return &RuntimeOrder{ID: id, Tipo: tipo, ResultadoJSON: resultadoJSON}
	}
	start10 := &RuntimeOrder{ID: 10, Tipo: "start", ResultadoJSON: `{}`}
	resumeLinked := mkOrder(1, "resume", `{"start_order_id":10}`)
	handoffLinked := mkOrder(2, "handoff", `{"start_order_id":10}`)
	resumeUnlinked := mkOrder(3, "resume", `{"start_order_id":99}`)

	// resume derivado PIERDE frente a su start fuente
	if runtimeBootstrapLeaseResumeDerivadoPrefiereStart(resumeLinked, start10) != true {
		t.Error("resumeDerivadoPrefiereStart deberia ser true para resume linked")
	}
	if runtimeBootstrapLeaseResumeDerivadoPrefiereStart(resumeUnlinked, start10) != false {
		t.Error("resumeDerivadoPrefiereStart deberia ser false para resume no linked")
	}
	if runtimeBootstrapLeaseResumeDerivadoPrefiereStart(handoffLinked, start10) != false {
		t.Error("resumeDerivadoPrefiereStart deberia ser false para handoff")
	}
	if runtimeBootstrapLeaseResumeDerivadoPrefiereStart(nil, start10) != false {
		t.Error("resumeDerivadoPrefiereStart deberia ser false para order nil")
	}

	// handoff/otro GANA frente a su start fuente
	if runtimeBootstrapLeaseFuenteLigadaPrefiereSobreStart(handoffLinked, start10) != true {
		t.Error("fuenteLigadaPrefiereSobreStart deberia ser true para handoff linked")
	}
	if runtimeBootstrapLeaseFuenteLigadaPrefiereSobreStart(resumeLinked, start10) != false {
		t.Error("fuenteLigadaPrefiereSobreStart deberia ser false para resume")
	}
	if runtimeBootstrapLeaseFuenteLigadaPrefiereSobreStart(handoffLinked, mkOrder(10, "resume", `{}`)) != false {
		t.Error("fuenteLigadaPrefiereSobreStart deberia ser false si startOrder no es tipo start")
	}
}

func TestRuntimeBootstrapLeaseAffinityScore(t *testing.T) {
	handleID := int64(5)
	runtimeID := int64(3)
	sesionID := int64(7)

	handle := &RuntimeHandle{ID: handleID}
	runtime := &RuntimeInstance{ID: runtimeID}

	// handle match via HandleID field: +8
	hid := handleID
	orderHandleField := &RuntimeOrder{HandleID: &hid, ResultadoJSON: `{}`}
	if s := runtimeBootstrapLeaseAffinityScore(orderHandleField, nil, handle, nil, 0); s != 8 {
		t.Errorf("handle field match: want 8, got %d", s)
	}

	// handle match via resultado handle_id: +8
	orderHandleResult := &RuntimeOrder{ResultadoJSON: fmt.Sprintf(`{"handle_id":%d}`, handleID)}
	if s := runtimeBootstrapLeaseAffinityScore(orderHandleResult, nil, handle, nil, 0); s != 8 {
		t.Errorf("handle result match: want 8, got %d", s)
	}

	// runtime match via RuntimeID field: +4
	rid := runtimeID
	orderRuntimeField := &RuntimeOrder{RuntimeID: &rid, ResultadoJSON: `{}`}
	if s := runtimeBootstrapLeaseAffinityScore(orderRuntimeField, nil, nil, runtime, 0); s != 4 {
		t.Errorf("runtime field match: want 4, got %d", s)
	}

	// runtime match via resultado runtime_id: +4
	orderRuntimeResult := &RuntimeOrder{ResultadoJSON: fmt.Sprintf(`{"runtime_id":%d}`, runtimeID)}
	if s := runtimeBootstrapLeaseAffinityScore(orderRuntimeResult, nil, nil, runtime, 0); s != 4 {
		t.Errorf("runtime result match: want 4, got %d", s)
	}

	// sesion match via resultado sesion_id: +2
	orderSesion := &RuntimeOrder{ResultadoJSON: fmt.Sprintf(`{"sesion_id":%d}`, sesionID)}
	if s := runtimeBootstrapLeaseAffinityScore(orderSesion, nil, nil, nil, sesionID); s != 2 {
		t.Errorf("sesion match: want 2, got %d", s)
	}

	// combinado handle+runtime+sesion: +8+4+2 = 14
	orderFull := &RuntimeOrder{
		HandleID:      &hid,
		RuntimeID:     &rid,
		ResultadoJSON: fmt.Sprintf(`{"sesion_id":%d}`, sesionID),
	}
	if s := runtimeBootstrapLeaseAffinityScore(orderFull, nil, handle, runtime, sesionID); s != 14 {
		t.Errorf("combinado handle+runtime+sesion: want 14, got %d", s)
	}

	// handle match via startOrder HandleID: +8
	startWithHandle := &RuntimeOrder{HandleID: &hid, ResultadoJSON: `{}`}
	orderNoHandle := &RuntimeOrder{ResultadoJSON: `{}`}
	if s := runtimeBootstrapLeaseAffinityScore(orderNoHandle, startWithHandle, handle, nil, 0); s != 8 {
		t.Errorf("handle via startOrder field: want 8, got %d", s)
	}

	// nil order: 0
	if s := runtimeBootstrapLeaseAffinityScore(nil, nil, handle, runtime, sesionID); s != 0 {
		t.Errorf("nil order: want 0, got %d", s)
	}
}

func TestRuntimeBootstrapLeaseMailboxIDsConFallbackFuente(t *testing.T) {
	mkOrder := func(id int64, tipo, resultadoJSON string) *RuntimeOrder {
		return &RuntimeOrder{ID: id, Tipo: tipo, ResultadoJSON: resultadoJSON}
	}

	// helper: collapsa [][]int64 a string para comparar
	dump := func(sets [][]int64) string {
		b, _ := json.Marshal(sets)
		return string(b)
	}

	start := mkOrder(10, "start", `{"mailbox_ids":[100,101]}`)

	cases := []struct {
		name       string
		order      *RuntimeOrder
		startOrder *RuntimeOrder
		want       [][]int64
	}{
		{
			"orden con mailbox propios, no linked",
			mkOrder(1, "resume", `{"mailbox_ids":[5,6]}`),
			nil,
			[][]int64{{5, 6}},
		},
		{
			"orden sin mailbox, linked a start → hereda mailbox de start",
			mkOrder(2, "resume", fmt.Sprintf(`{"start_order_id":%d}`, start.ID)),
			start,
			[][]int64{{100, 101}},
		},
		{
			"orden con mailbox distintos a start → ambos conjuntos como fallback",
			mkOrder(3, "resume", fmt.Sprintf(`{"start_order_id":%d,"mailbox_ids":[99]}`, start.ID)),
			start,
			[][]int64{{99}, {100, 101}},
		},
		{
			"orden con mismos mailbox que start → solo un conjunto (sin duplicar)",
			mkOrder(4, "resume", fmt.Sprintf(`{"start_order_id":%d,"mailbox_ids":[100,101]}`, start.ID)),
			start,
			[][]int64{{100, 101}},
		},
		{
			"orden no linked aunque startOrder presente → solo mailbox propios",
			mkOrder(5, "resume", `{"start_order_id":99,"mailbox_ids":[7]}`),
			start,
			[][]int64{{7}},
		},
		{
			"orden sin mailbox y no linked → sin candidatos",
			mkOrder(6, "resume", `{}`),
			nil,
			[][]int64{},
		},
	}
	for _, c := range cases {
		got := runtimeBootstrapLeaseMailboxIDsConFallbackFuente(c.order, c.startOrder)
		if dump(got) != dump(c.want) {
			t.Errorf("[%s] got %s, want %s", c.name, dump(got), dump(c.want))
		}
	}
}

func TestRuntimeBootstrapLeaseSelectionMejora(t *testing.T) {
	mkOrder := func(id int64, tipo, resultadoJSON string) *RuntimeOrder {
		return &RuntimeOrder{ID: id, Tipo: tipo, ResultadoJSON: resultadoJSON}
	}

	start10 := mkOrder(10, "start", `{}`)
	resumeLinked := mkOrder(1, "resume", fmt.Sprintf(`{"start_order_id":%d}`, start10.ID))
	handoffLinked := mkOrder(2, "handoff", fmt.Sprintf(`{"start_order_id":%d}`, start10.ID))
	orderA := mkOrder(20, "start", `{}`) // ID mayor
	orderB := mkOrder(15, "start", `{}`) // ID menor

	const targetSesion = int64(7)

	mk := func(order *RuntimeOrder, sesionID int64, score int) bootstrapRuntimeLeaseSelection {
		return newBootstrapRuntimeLeaseSelection(order, 0, nil, sesionID, score)
	}

	cases := []struct {
		name      string
		candidate bootstrapRuntimeLeaseSelection
		selected  bootstrapRuntimeLeaseSelection
		want      bool
	}{
		// Prioridad sesión
		{"sesion: candidato coincide, selected no", mk(orderA, targetSesion, 0), mk(orderB, 0, 0), true},
		{"sesion: selected coincide, candidato no", mk(orderA, 0, 0), mk(orderB, targetSesion, 0), false},
		// Score
		{"score: candidato mayor", mk(orderA, 0, 9), mk(orderB, 0, 4), true},
		{"score: candidato menor", mk(orderA, 0, 3), mk(orderB, 0, 8), false},
		// Tiebreaker resume derivado pierde frente a su start
		{"tiebreaker: resume derivado pierde frente a start fuente", mk(resumeLinked, 0, 0), mk(start10, 0, 0), false},
		// Tiebreaker start pierde frente a resume derivado (candidate=start, selected=resume)
		{"tiebreaker: start pierde frente a resume derivado", mk(start10, 0, 0), mk(resumeLinked, 0, 0), true},
		// Tiebreaker handoff gana sobre start
		{"tiebreaker: handoff gana sobre start fuente", mk(handoffLinked, 0, 0), mk(start10, 0, 0), true},
		// Tiebreaker start pierde frente a handoff
		{"tiebreaker: start pierde frente a handoff derivado", mk(start10, 0, 0), mk(handoffLinked, 0, 0), false},
		// Default: ID mayor gana
		{"default: ID mayor gana", mk(orderA, 0, 0), mk(orderB, 0, 0), true},
		{"default: ID menor pierde", mk(orderB, 0, 0), mk(orderA, 0, 0), false},
		{"default: ID igual -> true (>=)", mk(orderA, 0, 0), mk(orderA, 0, 0), true},
	}
	for _, c := range cases {
		got := runtimeBootstrapLeaseSelectionMejora(c.candidate, c.selected, targetSesion)
		if got != c.want {
			t.Errorf("[%s] runtimeBootstrapLeaseSelectionMejora = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestAckBootstrapRuntimeSinLeaseEnStartAnexaLeaseAlStart(t *testing.T) {
	prepararDBTemporal(t)

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Claude1",
		Tipo:        "start",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start ejecutando: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil || startOrder == nil {
		t.Fatalf("get start: %+v err=%v", startOrder, err)
	}
	msg := &RuntimeMailboxMessage{ID: 77}
	sesion := &Sesion{ID: 19}
	bootstrap := &bootstrapRuntimeData{
		Mailbox: []*RuntimeMailboxMessage{msg},
	}

	if err := ackBootstrapRuntimeSinLeaseEnStart(startOrder, bootstrap, sesion); err != nil {
		t.Fatalf("ackBootstrapRuntimeSinLeaseEnStart: %v", err)
	}
	startOrder, err = GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("reload start: %v", err)
	}
	if !strings.Contains(startOrder.ResultadoJSON, `"lease_state":"waiting_for_evidence"`) {
		t.Fatalf("start sin lease_state: %s", startOrder.ResultadoJSON)
	}
	if !strings.Contains(startOrder.ResultadoJSON, `"mailbox_ids":[77]`) {
		t.Fatalf("start sin mailbox_ids: %s", startOrder.ResultadoJSON)
	}
	if !strings.Contains(startOrder.ResultadoJSON, `"sesion_id":19`) {
		t.Fatalf("start sin sesion_id: %s", startOrder.ResultadoJSON)
	}
}

func TestAckBootstrapRuntimeLeaseByEvidenceNoConsumeMailboxDesdeStartRunningSinProgress(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Claude1", "programador"); err != nil {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-code",
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

	statusPath := filepath.Join(tmp, "worker-status.json")
	heartbeatPath := filepath.Join(tmp, "worker-heartbeat.json")
	initial := time.Now().UTC().Format(time.RFC3339Nano)
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "starting",
		"alive":      true,
		"updated_at": initial,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": initial,
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=?, estado='activo' WHERE id=?`,
		string(metaJSON), `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`, handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"microciclo premium"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Claude1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d}`,
			mailboxID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start lease: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start ejecutando: %v", err)
	}

	ackAt := time.Now().UTC().Add(2 * time.Second).Format(time.RFC3339Nano)
	statusRaw, _ = json.Marshal(map[string]any{
		"state":      "running",
		"alive":      true,
		"updated_at": ackAt,
	})
	heartbeatRaw, _ = json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": ackAt,
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("rewrite status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("rewrite heartbeat: %v", err)
	}

	if err := AckBootstrapRuntimeLeaseByEvidence(handle, runtime, "test_start_running"); err != nil {
		t.Fatalf("ack bootstrap por evidencia: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start: %v", err)
	}
	if startOrder.Estado != "ejecutando" {
		t.Fatalf("start bootstrap no deberia completarse sin progreso util: %+v", startOrder)
	}
	msgsPendientes, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   strPtrTest("Claude1"),
		ProyectoID: &proyectoID,
		Estado:     strPtrTest("pendiente"),
	})
	if err != nil {
		t.Fatalf("mailbox pendiente: %v", err)
	}
	if len(msgsPendientes) != 1 || msgsPendientes[0].ID != mailboxID {
		t.Fatalf("mailbox deberia seguir pendiente sin progreso util: %+v", msgsPendientes)
	}
}

func TestAckBootstrapRuntimeLeaseByEvidenceConsumeBootstrapTMUXConOutputFrescoSinDiff(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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

	statusPath := filepath.Join(tmp, "worker-status-running.json")
	heartbeatPath := filepath.Join(tmp, "worker-heartbeat-running.json")
	initial := time.Now().UTC().Add(-3 * time.Second)
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "running",
		"alive":                 true,
		"updated_at":            initial.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": initial.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=?, estado='activo', transporte='tmux', handle_kind='session' WHERE id=?`,
		string(metaJSON), `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`, handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload runtime handle: %+v err=%v", handle, err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"microciclo premium"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d}`,
			mailboxID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start lease: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start ejecutando: %v", err)
	}

	ackAt := time.Now().UTC().Add(2 * time.Second).Format(time.RFC3339Nano)
	statusRaw, _ = json.Marshal(map[string]any{
		"state":                 "running",
		"alive":                 true,
		"updated_at":            ackAt,
		"ready_at":              ackAt,
		"last_output_at":        ackAt,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	heartbeatRaw, _ = json.Marshal(map[string]any{
		"alive":          true,
		"heartbeat_at":   ackAt,
		"ready_at":       ackAt,
		"last_output_at": ackAt,
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("rewrite status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("rewrite heartbeat: %v", err)
	}

	if err := AckBootstrapRuntimeLeaseByEvidence(handle, runtime, "test_tmux_running_output"); err != nil {
		t.Fatalf("ack bootstrap por evidencia tmux: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start: %v", err)
	}
	if startOrder.Estado != "completada" {
		t.Fatalf("start bootstrap deberia completarse con output fresco de tmux premium: %+v", startOrder)
	}
	msgsConsumidos, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   strPtrTest("Codex1"),
		ProyectoID: &proyectoID,
		Estado:     strPtrTest("consumido"),
	})
	if err != nil {
		t.Fatalf("mailbox consumido: %v", err)
	}
	if len(msgsConsumidos) != 1 || msgsConsumidos[0].ID != mailboxID {
		t.Fatalf("mailbox deberia consumirse con output fresco de tmux premium: %+v", msgsConsumidos)
	}
	runtime, err = GetRuntime(runtime.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime final: %+v err=%v", runtime, err)
	}
	if runtime.LogicalState != "activo" || runtime.ProcessState != "running" {
		t.Fatalf("el ack bootstrap deberia promover el runtime tmux observado: %+v", runtime)
	}
}

func TestAckBootstrapRuntimeLeaseActualizaStartCompletadaConLeaseAcked(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
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
	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"microciclo premium"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","delivery_state":"delivered","delivery_receipt_at":"2026-04-14T14:20:51Z","receipt_source":"git_worktree","mailbox_ids":[%d],"sesion_id":%d}`,
			mailboxID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada', finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start completada: %v", err)
	}

	if err := AckBootstrapRuntimeLease(startID, 0, []int64{mailboxID}, sesion.ID, "agente_tick"); err != nil {
		t.Fatalf("ack bootstrap runtime lease: %v", err)
	}

	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start: %v", err)
	}
	if startOrder.Estado != "completada" {
		t.Fatalf("start deberia seguir completada: %+v", startOrder)
	}
	if !strings.Contains(startOrder.ResultadoJSON, `"lease_state":"acked"`) {
		t.Fatalf("start completada deberia actualizar lease_state a acked: %s", startOrder.ResultadoJSON)
	}
	if !strings.Contains(startOrder.ResultadoJSON, `"acked_by":"agente_tick"`) {
		t.Fatalf("start completada deberia registrar acked_by: %s", startOrder.ResultadoJSON)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox: %+v err=%v", msg, err)
	}
	if msg.Estado != "consumido" {
		t.Fatalf("mailbox deberia consumirse tras ack bootstrap: %+v", msg)
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

func TestRuntimeOrderSendInstructionPipelinePremiumRetieneNotificadaSiSessionResumeTimeout(t *testing.T) {
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
		ExternalSessionID: "sess-pipeline-timeout",
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

	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                    "process_pty_cli",
		"rendered_command":          "'" + wrapper + "' 'Codex1'",
		"working_dir":               workingDir,
		"external_session_id":       "sess-pipeline-timeout",
		"can_send_input":            false,
		"session_resume_timeout_ms": 10,
	})
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
		Kind:        "pipeline_local",
		PayloadJSON: `{"source":"microrefactor_loop"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"to_agente":    "Codex1",
		"from_agente":  "server",
		"texto":        "continua trabajo actual",
		"mailbox_id":   mailboxID,
		"mailbox_kind": "pipeline_local",
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
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("pipeline_local premium deberia quedar pendiente/notificada tras timeout de session_resume: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"dispatch_state":"notified"`) {
		t.Fatalf("resultado sin dispatch_state notified: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"delivery_state":"notified"`) {
		t.Fatalf("resultado sin delivery_state notified: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `session_resume dispatch pending receipt`) {
		t.Fatalf("resultado sin razon de pending receipt: %s", sendOrder.ResultadoJSON)
	}

	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if !strings.EqualFold(strings.TrimSpace(msg.Estado), "entregado") {
		t.Fatalf("mailbox deberia quedar entregado mientras espera receipt: %+v", msg)
	}
}

func TestRuntimeOrderSendInstructionNudgeTMUXReencolaSiSessionResumeTimeoutSinWorkerSnapshot(t *testing.T) {
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
		ExternalSessionID: "sess-nudge-timeout",
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

	wrapper := filepath.Join(tmp, "codex-perfil")
	script := "#!/usr/bin/env bash\nset -euo pipefail\nsleep 1\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                    "tmux_cli_session",
		"transport":                 "tmux",
		"tmux_session":              "orq-codex1-timeout",
		"tmux_pane_id":              "%88",
		"rendered_command":          "'" + wrapper + "' 'Codex1'",
		"herramienta":               "codex-cli",
		"working_dir":               workingDir,
		"external_session_id":       "sess-nudge-timeout",
		"can_send_input":            false,
		"session_resume_timeout_ms": 10,
	})
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', metadata_json=?, capabilities_json=? WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"continua con el siguiente slice util"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"to_agente":           "Codex1",
		"from_agente":         "server",
		"texto":               "continua con el siguiente slice util",
		"mailbox_id":          mailboxID,
		"mailbox_kind":        "nudge",
		"external_session_id": "sess-nudge-timeout",
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
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("nudge codex tmux deberia reencolarse si falta worker snapshot: %+v", sendOrder)
	}
	if strings.Contains(sendOrder.ResultadoJSON, `"mailbox_only":true`) {
		t.Fatalf("tmux canonico no deberia degradar a mailbox_only: %s", sendOrder.ResultadoJSON)
	}
	if strings.Contains(sendOrder.ResultadoJSON, `runtime_handle_session_resume_mailbox_only:nudge`) {
		t.Fatalf("tmux canonico no deberia marcar mailbox_only:nudge: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"deferred_reason":"worker_snapshot_missing"`) {
		t.Fatalf("resultado sin reason worker_snapshot_missing: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"dispatch_state":"pending"`) {
		t.Fatalf("resultado sin dispatch_state pending: %s", sendOrder.ResultadoJSON)
	}

	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if !strings.EqualFold(strings.TrimSpace(msg.Estado), "pendiente") {
		t.Fatalf("mailbox nudge deberia quedar pendiente para conciliacion durable: %+v", msg)
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
	if !strings.Contains(sendOrder.ResultadoJSON, `runtime_handle_bootstrap_only_mailbox_only:autonomia`) &&
		!strings.Contains(sendOrder.ResultadoJSON, `runtime_handle_session_resume_mailbox_only:autonomia`) &&
		!strings.Contains(sendOrder.ResultadoJSON, `codex_long_text_mailbox_only:autonomia`) {
		t.Fatalf("resultado sin degradacion durable esperada: %s", sendOrder.ResultadoJSON)
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

func TestRuntimeOrderSendInstructionCodexSupervisadoLargoDegradaAMailboxDurable(t *testing.T) {
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
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}

	metaJSON := `{"driver":"process_pty_cli","stdin_path":"` + filepath.Join(tmp, "pty.stdin") + `","supervisor_ref":"` + filepath.Join(tmp, "supervisor.ref") + `","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-codex-long","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryBootstrapOnly + `"}`
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, capsJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"mensaje largo"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"to_agente":    "Codex1",
		"from_agente":  "server",
		"texto":        "Esta instruccion larga de orquesta debe quedar en mailbox durable y no entrar por stdin del supervisor local de Codex.",
		"mailbox_id":   mailboxID,
		"mailbox_kind": "instruction",
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
		t.Fatalf("send_instruction deberia completarse dejando mailbox durable: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_only":true`) {
		t.Fatalf("resultado sin mailbox_only: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `codex_long_text_mailbox_only`) {
		t.Fatalf("resultado sin trazabilidad de degradacion protectora: %s", sendOrder.ResultadoJSON)
	}
	transcript, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: stringPtr("Codex1"), Limit: 20})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	for _, item := range transcript {
		if item != nil && item.Stream == "stdin" && strings.Contains(item.Text, "mailbox durable") {
			t.Fatalf("no deberia inyectar stdin larga a Codex supervisado: %+v", item)
		}
	}
}

func TestRuntimeOrderSendInstructionLongTextReasonNoAplicaATMUXCanonico(t *testing.T) {
	handle := &RuntimeHandle{
		Transporte:   "tmux",
		HandleKind:   "session",
		MetadataJSON: `{"driver":"tmux_cli_session","rendered_command":"codex-perfil Codex1","external_session_id":"sess-canonico","can_send_input":false}`,
	}
	payload := map[string]any{
		"mailbox_kind": "nudge",
	}
	texto := "Corrige rumbo: no abras ni refuerces process_pty_cli ni fallback interactivo legacy."
	if reason := runtimeOrderSendInstructionLongTextReason(handle, payload, texto); reason != "" {
		t.Fatalf("tmux canonico no deberia degradar texto largo a mailbox_only, got=%q", reason)
	}
}

func TestRuntimeOrderSendInstructionCodexGuidanceMailboxQuedaDurable(t *testing.T) {
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
		ExternalSessionID: "sess-guidance-1",
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

	metaJSON := `{"driver":"process_pty_cli","stdin_path":"` + filepath.Join(tmp, "pty.stdin") + `","supervisor_ref":"` + filepath.Join(tmp, "supervisor.ref") + `","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-guidance-1","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, capsJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "orquesta",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"Prueba grande: resume el riesgo y sigue."}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"to_agente":    "Codex1",
		"from_agente":  "orquesta",
		"texto":        "Prueba grande: resume el riesgo y sigue.",
		"mailbox_id":   mailboxID,
		"mailbox_kind": "nudge",
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
		t.Fatalf("send_instruction guidance deberia quedar completada como durable: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_only":true`) {
		t.Fatalf("resultado sin degradacion durable esperada: %s", sendOrder.ResultadoJSON)
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
	transcript, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: stringPtr("Codex1"), Limit: 20})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	for _, item := range transcript {
		if item != nil && item.Stream == "stdin" {
			t.Fatalf("no deberia inyectar guidance viva a Codex inestable: %+v", item)
		}
	}
}

func TestRuntimeOrderSendInstructionNudgeTMUXSessionResumeNoQuedaMailboxOnly(t *testing.T) {
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
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}

	outPath := filepath.Join(tmp, "codex-session-resume-tmux.log")
	wrapper := filepath.Join(tmp, "codex-perfil")
	if err := os.MkdirAll(filepath.Join(tmp, "orquestador"), 0o755); err != nil {
		t.Fatalf("mkdir working dir: %v", err)
	}
	script := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '%s\\n' \"$@\" > '" + strings.ReplaceAll(outPath, "'", `'\''`) + "'\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	meta := map[string]any{
		"driver":              "tmux_cli_session",
		"tmux_session":        "orq-codex1-test",
		"tmux_pane_id":        "%1",
		"rendered_command":    "'" + wrapper + "' 'Codex1'",
		"herramienta":         "codex-cli",
		"working_dir":         filepath.Join(tmp, "orquestador"),
		"external_session_id": "sess-runtime-resume-mailbox-tmux",
		"can_send_input":      false,
	}
	metaJSON, _ := json.Marshal(meta)
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', metadata_json=?, capabilities_json=? WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"continua con el siguiente slice util"}`,
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
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"continua con el siguiente slice util","mailbox_id":%d,"mailbox_kind":"nudge","external_session_id":"sess-runtime-resume-mailbox-tmux"}`, mailboxID),
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
		t.Fatalf("send_instruction tmux canonico deberia evitar mailbox_only aunque falte snapshot: %+v", sendOrder)
	}
	if strings.Contains(sendOrder.ResultadoJSON, `"mailbox_only":true`) {
		t.Fatalf("tmux session_resume canonico no deberia degradar nudge a mailbox_only: %s", sendOrder.ResultadoJSON)
	}
	if strings.Contains(sendOrder.ResultadoJSON, `runtime_handle_session_resume_mailbox_only:nudge`) {
		t.Fatalf("tmux session_resume canonico no deberia marcar reason mailbox_only:nudge: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"deferred_reason":"worker_snapshot_missing"`) {
		t.Fatalf("resultado sin reason worker_snapshot_missing: %s", sendOrder.ResultadoJSON)
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
	oldHandleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, capabilities_json, metadata_json
	) VALUES (?,?,?,?,?,?,?,?,?)`,
		"Codex1", proyectoID, oldRuntimeID, "cli", "process", "999999", "cerrado", string(oldCapsJSON), string(oldMetaJSON))

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

func TestRuntimeOrderSendInstructionMailboxPausadoQuedaDurable(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	rutaBase := filepath.Join(tmp, "orquestador")
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
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               rutaBase,
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-pausado-1",
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
	metaJSON, _ := json.Marshal(map[string]any{
		"unsafe_tty":          true,
		"working_dir":         filepath.Join(rutaBase, ".orquesta-worktrees", "orquesta-codex1"),
		"rendered_command":    "codex-perfil Codex1",
		"external_session_id": "sess-pausado-1",
	})
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='pausado', metadata_json=?, capabilities_json=? WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("update handle pausado: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"continua trabajo actual"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}

	payload, _ := json.Marshal(map[string]any{
		"to_agente":    "Codex1",
		"from_agente":  "server",
		"texto":        "continua trabajo actual",
		"mailbox_id":   mailboxID,
		"mailbox_kind": "instruction",
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
		t.Fatalf("send_instruction pausada deberia quedar completada como durable: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_only":true`) || !strings.Contains(sendOrder.ResultadoJSON, `runtime_handle_pausado_mailbox_only:instruction`) {
		t.Fatalf("resultado sin degradacion durable esperada: %s", sendOrder.ResultadoJSON)
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
	transcript, err := ListarRuntimeTranscript(FiltroRuntimeTranscript{Agente: stringPtr("Codex1"), Limit: 20})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	for _, item := range transcript {
		if item != nil && item.Stream == "stdin" {
			t.Fatalf("no deberia inyectar input vivo sobre handle pausado: %+v", item)
		}
	}
	if runtime.ID <= 0 {
		t.Fatalf("runtimeID invalido: %d", runtime.ID)
	}
}

func TestRuntimeOrderStartInyectaContinuidadAlProcesoReal(t *testing.T) {
	enableLegacyPTYLocalRuntimeForTest(t)
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

func TestRuntimeOrderSendInstructionBootstrapOnlyCodexLocalQuedaDurableAunqueRecupereSesion(t *testing.T) {
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
	if sendOrder.Estado != "pendiente" || !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_only":true`) {
		t.Fatalf("send_instruction bootstrap_only deberia quedar pendiente y durable: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `runtime_handle_bootstrap_only_mailbox_only`) {
		t.Fatalf("resultado sin trazabilidad bootstrap_only durable: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"delivery_state":"queued"`) {
		t.Fatalf("resultado sin delivery_state queued: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.PayloadJSON, `"mailbox_id":`) {
		t.Fatalf("payload sin mailbox_id durable: %s", sendOrder.PayloadJSON)
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("bootstrap_only no deberia ejecutar session_resume, stat err=%v", err)
	}
}

func TestRuntimeOrderSendInstructionCodexLocalNoUsaSessionResumeYQuedaDurable(t *testing.T) {
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
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
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
	if sendOrder.Estado != "pendiente" || !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_only":true`) {
		t.Fatalf("send_instruction local no interactivo deberia quedar pendiente en mailbox durable: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"delivery_state":"queued"`) {
		t.Fatalf("resultado sin delivery_state queued: %s", sendOrder.ResultadoJSON)
	}

	if _, err := os.Stat(outPath); err == nil {
		t.Fatalf("no deberia ejecutar session_resume directo en un Codex local no interactivo")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat resume output: %v", err)
	}

	estadoPendiente := "pendiente"
	msgs, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   strPtrTest("Codex1"),
		ProyectoID: &proyectoID,
		Estado:     &estadoPendiente,
	})
	if err != nil {
		t.Fatalf("listar mailbox durable: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("mailbox durable inesperada: %+v", msgs)
	}
	msg := msgs[0]
	if msg.RuntimeOrderID == nil || *msg.RuntimeOrderID != sendID {
		t.Fatalf("mailbox durable sin runtime_order asociado: %+v", msg)
	}
	if msg.PayloadJSON == "" || !strings.Contains(msg.PayloadJSON, "hola session resume live") {
		t.Fatalf("payload durable inesperado: %+v", msg)
	}
}

func TestRuntimeOrderSendInstructionTMUXInteractiveStartingQuedaDiferidaAMailbox(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Ollama1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		Agente:      "Ollama1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "ollama-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}

	workerDir := filepath.Join(tmp, "worker-starting")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	writeJSON := func(path string, payload map[string]any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-ollama1-starting",
		"tmux_pane_id":          "%4",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
		"can_send_input":        true,
	})
	writeJSON(statusPath, map[string]any{
		"state":                 "starting",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-ollama1-starting","tmux_pane_id":"%%4","can_send_input":true,"mailbox_delivery_mode":"interactive","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(
		`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-ollama1-starting/%4', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Ollama1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Ollama1","texto":"implementa solo la funcion pedida"}`,
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
		t.Fatalf("send_instruction deberia quedar pendiente: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_only":true`) || !strings.Contains(sendOrder.ResultadoJSON, `"deferred":true`) {
		t.Fatalf("resultado sin diferido durable: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `worker_starting`) {
		t.Fatalf("resultado sin motivo worker_starting: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.PayloadJSON, `"mailbox_id":`) {
		t.Fatalf("payload sin mailbox_id durable: %s", sendOrder.PayloadJSON)
	}
}

func TestRuntimeOrderSendInstructionMailboxSinRuntimeActivoQuedaRetenida(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Ollama1", "programador"); err != nil {
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
		ToAgente:    "Ollama1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"implementa solo normalizarIdentificador"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Ollama1",
		ProyectoID: &proyectoID,
		Tipo:       "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Ollama1","texto":"implementa solo normalizarIdentificador","mailbox_id":%d,"mailbox_kind":"instruction"}`,
			mailboxID),
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
		t.Fatalf("send_instruction deberia quedar pendiente: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_only":true`) || !strings.Contains(sendOrder.ResultadoJSON, `"deferred":true`) {
		t.Fatalf("resultado sin retención durable: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `sin runtime activo entregable`) {
		t.Fatalf("resultado sin motivo de retención: %s", sendOrder.ResultadoJSON)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil {
		t.Fatalf("get mailbox: %v", err)
	}
	if msg == nil || msg.Estado != "pendiente" {
		t.Fatalf("mailbox deberia seguir pendiente: %+v", msg)
	}
}

func TestRuntimeOrderSendInstructionDirectaSinRuntimeActivoQuedaRetenidaParaOllama(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Ollama1", "programador"); err != nil {
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

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Ollama1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Ollama1","texto":"implementa solo normalizarIdentificador"}`,
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
		t.Fatalf("send_instruction directa deberia quedar pendiente: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"deferred":true`) {
		t.Fatalf("resultado sin diferido durable: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `sin runtime activo entregable`) {
		t.Fatalf("resultado sin motivo de retencion: %s", sendOrder.ResultadoJSON)
	}
}

func TestRuntimeOrderSendInstructionMailboxSessionResumeQuedaDurable(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		ExternalSessionID: "sess-runtime-resume-mailbox",
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
	outPath := filepath.Join(tmp, "session-resume-mailbox.out")
	script := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '%s\\n' \"$@\" > '" + strings.ReplaceAll(outPath, "'", `'\''`) + "'\n"
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	meta := map[string]any{
		"driver":              "process_pty_cli",
		"rendered_command":    "'" + wrapper + "' 'Codex1'",
		"working_dir":         workingDir,
		"external_session_id": "sess-runtime-resume-mailbox",
		"can_send_input":      false,
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
		Kind:        "nudge",
		PayloadJSON: `{"texto":"continua"}`,
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
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"continua trabajo actual","mailbox_id":%d,"mailbox_kind":"nudge","external_session_id":"sess-runtime-resume-mailbox"}`, mailboxID),
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
		t.Fatalf("send_instruction mailbox session_resume deberia completarse: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_only":true`) ||
		(!strings.Contains(sendOrder.ResultadoJSON, `runtime_handle_session_resume_mailbox_only:nudge`) &&
			!strings.Contains(sendOrder.ResultadoJSON, `runtime_handle_bootstrap_only_mailbox_only:nudge`)) {
		t.Fatalf("resultado sin degradacion durable esperada: %s", sendOrder.ResultadoJSON)
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("session_resume no deberia ejecutarse para mailbox durable, stat err=%v", err)
	}
}

func TestRuntimeOrderSendInstructionMailboxBootstrapOnlyConExternalSessionNoSeElevaASessionResume(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		ExternalSessionID: "sess-bootstrap-only",
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

	meta := map[string]any{
		"driver":                "tmux_cli_session",
		"working_dir":           workingDir,
		"external_session_id":   "sess-bootstrap-only",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
		"can_send_input":        false,
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
		Kind:        "governance_refresh",
		PayloadJSON: `{"texto":"continua trabajo actual"}`,
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
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"continua trabajo actual","mailbox_id":%d,"mailbox_kind":"governance_refresh","external_session_id":"sess-bootstrap-only"}`, mailboxID),
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
		t.Fatalf("send_instruction bootstrap_only deberia completarse durable: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_only":true`) || !strings.Contains(sendOrder.ResultadoJSON, `runtime_handle_bootstrap_only_mailbox_only:governance_refresh`) {
		t.Fatalf("resultado sin degradacion bootstrap_only esperada: %s", sendOrder.ResultadoJSON)
	}
}

func TestRuntimeOrderSendInstructionMailboxBootstrapTMUXReadyMarcaMailboxEntregado(t *testing.T) {
	tmp := prepararDBTemporal(t)
	fakeTmux := writeFakeTMUXDispatchScriptDB(t, tmp, "active_after_enter")
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		ExternalSessionID: "sess-bootstrap-only",
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

	workerDir := filepath.Join(tmp, "worker-bootstrap-dispatch")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex1-bootstrap",
		"tmux_pane_id":          "%77",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
		"can_send_input":        false,
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"ready_at":              now.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
		"ready_at":     now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	meta := map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_command":          fakeTmux,
		"tmux_pane_id":          "%77",
		"tmux_session":          "orq-codex1-bootstrap",
		"working_dir":           workingDir,
		"external_session_id":   "sess-bootstrap-only",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
		"can_send_input":        false,
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
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
		Kind:        "governance_refresh",
		PayloadJSON: `{"texto":"continua trabajo actual"}`,
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
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"continua trabajo actual","mailbox_id":%d,"mailbox_kind":"governance_refresh","external_session_id":"sess-bootstrap-only"}`, mailboxID),
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
		t.Fatalf("la orden deberia quedar pendiente tras delivered, got=%+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"delivery_state":"notified"`) {
		t.Fatalf("resultado sin delivery_state notified: %s", sendOrder.ResultadoJSON)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox: %+v err=%v", msg, err)
	}
	if msg.Estado != "entregado" {
		t.Fatalf("mailbox deberia quedar entregado: %+v", msg)
	}
}

func TestRuntimeOrderSendInstructionMicroprogramacionTMUXInteractiveEsperaReceipt(t *testing.T) {
	tmp := prepararDBTemporal(t)
	fakeTmux := writeFakeTMUXDispatchScriptDB(t, tmp, "active_after_enter")
	if err := RegistrarAgente("Ollama1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		Agente:      "Ollama1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "ollama-cli",
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

	workerDir := filepath.Join(tmp, "worker-micro-ready")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	writeJSON := func(path string, payload map[string]any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_command":          fakeTmux,
		"tmux_session":          "orq-ollama1-micro",
		"tmux_pane_id":          "%77",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
		"can_send_input":        true,
	})
	writeJSON(statusPath, map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"ready_at":              now.Format(time.RFC3339Nano),
		"updated_at":            now.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"ready_at":     now.Format(time.RFC3339Nano),
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_session":"orq-ollama1-micro","tmux_pane_id":"%%77","can_send_input":true,"mailbox_delivery_mode":"interactive","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`,
		fakeTmux, manifestPath, statusPath, heartbeatPath,
	)
	if _, err := DB.Exec(
		`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-ollama1-micro/%77', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Ollama1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Ollama1","source":"microprogramacion","microprogramacion":{"especificacion_id":1,"write_set":["identidad/normalizar.go"]},"texto":"PATCH_UNIFICADO: cambia solo identidad/normalizar.go"}`,
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
		t.Fatalf("microprogramacion interactiva deberia esperar receipt: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"dispatch_state":"notified"`) {
		t.Fatalf("resultado sin dispatch_state notified: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"delivery_state":"notified"`) {
		t.Fatalf("resultado sin delivery_state notified: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.PayloadJSON, `"mailbox_id":`) {
		t.Fatalf("payload sin mailbox durable para receipt: %s", sendOrder.PayloadJSON)
	}
	msg, err := GetRuntimeMailboxByRuntimeOrderID(sendID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox por runtime_order: %+v err=%v", msg, err)
	}
	if msg.Estado != "entregado" {
		t.Fatalf("mailbox de microprogramacion deberia quedar entregado: %+v", msg)
	}
}

func TestRuntimeOrderSendInstructionMicroprogramacionOllamaPoolEsperaRecibo(t *testing.T) {
	order := &RuntimeOrder{Agente: "Gemma1"}
	handle := &RuntimeHandle{
		Transporte:       "api",
		MetadataJSON:     `{"driver":"ollama_pool_local","pool_local":true,"can_send_input":true}`,
		CapabilitiesJSON: `{"can_send_input":true}`,
	}
	payload := map[string]any{
		"source": "microprogramacion",
		"microprogramacion": map[string]any{
			"especificacion_id": 9,
		},
	}
	if !runtimeOrderSendInstructionDebeEsperarReceiptInteractivo(order, handle, payload) {
		t.Fatalf("ollama_pool_local deberia esperar recibo para microprogramacion")
	}
}

func TestObjetivoProcesoSendInstructionIncluyeRuntimeOrderIDEnMetadata(t *testing.T) {
	order := &RuntimeOrder{ID: 77, Agente: "Gemma1"}
	handle := &RuntimeHandle{
		HandleKind:   "session",
		HandleRef:    "ollama-pool-1",
		MetadataJSON: `{"driver":"ollama_pool_local","transport":"api"}`,
	}

	obj := objetivoProcesoSendInstruction(order, handle, nil, "/tmp/orquesta")
	meta := mapFromJSON(obj.MetadataJSON)
	if got := int64FromAny(meta["runtime_order_id"]); got != 77 {
		t.Fatalf("runtime_order_id inesperado: %v meta=%s", got, obj.MetadataJSON)
	}
	if got := strings.TrimSpace(stringFromMap(meta, "working_dir", "")); got != "/tmp/orquesta" {
		t.Fatalf("working_dir inesperado: %q meta=%s", got, obj.MetadataJSON)
	}
}

func TestRuntimeOrderSendInstructionReceiptBloqueoIgnoraSnapshotOllamaPoolLocal(t *testing.T) {
	handle := &RuntimeHandle{
		Transporte:   "api",
		MetadataJSON: `{"driver":"ollama_pool_local","transport":"api","pool_local":true}`,
	}
	if !runtimeOrderSendInstructionReceiptBloqueoIgnoraSnapshot(handle) {
		t.Fatalf("ollama_pool_local no deberia bloquear receipt por ausencia de worker snapshot")
	}
}

func TestProcesarRuntimeOrdersBatchCompletaMailboxEntregadoConProgresoPosterior(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		ExternalSessionID: "sess-bootstrap-only",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-bootstrap-progress")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	deliveredAt := now.Add(-2 * time.Minute)
	progressAt := now.Add(-time.Minute)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex1-bootstrap",
		"tmux_pane_id":          "%77",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
		"can_send_input":        false,
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "running",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"ready_at":              deliveredAt.Format(time.RFC3339Nano),
		"last_progress_at":      progressAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":            true,
		"heartbeat_at":     now.Format(time.RFC3339Nano),
		"ready_at":         deliveredAt.Format(time.RFC3339Nano),
		"last_progress_at": progressAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-codex1-bootstrap","tmux_pane_id":"%%77","mailbox_delivery_mode":"%s","can_send_input":false,"worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, runtimeagente.MailboxDeliveryBootstrapOnly, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "governance_refresh",
		PayloadJSON: `{"hash":"canon"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"continua trabajo actual","mailbox_id":%d,"mailbox_kind":"governance_refresh"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	orderBaseline := deliveredAt.Add(30 * time.Second)
	if _, err := DB.Exec(`UPDATE runtime_orders SET created_at=?, updated_at=? WHERE id=?`, orderBaseline, orderBaseline, sendID); err != nil {
		t.Fatalf("backdate order created_at: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar la orden entregada, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("la orden deberia completarse con progreso posterior: %+v", sendOrder)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "consumido" {
		t.Fatalf("mailbox deberia quedar consumido: %+v", msg)
	}
}

func TestProcesarRuntimeOrdersBatchAutonomiaNoCompletaSoloPorProgresoPosterior(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		Agente:            "Claude1",
		ProyectoID:        &proyectoID,
		CWD:               workingDir,
		Herramienta:       "claude-code",
		ExternalSessionID: "sess-claude-autonomia",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-autonomia-progress")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	deliveredAt := now.Add(-2 * time.Minute)
	progressAt := now.Add(-time.Minute)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-claude1-ready",
		"tmux_pane_id":          "%12",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
		"can_send_input":        true,
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "running",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"ready_at":              deliveredAt.Format(time.RFC3339Nano),
		"last_progress_at":      progressAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":            true,
		"heartbeat_at":     now.Format(time.RFC3339Nano),
		"ready_at":         deliveredAt.Format(time.RFC3339Nano),
		"last_progress_at": progressAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-claude1-ready","tmux_pane_id":"%%12","mailbox_delivery_mode":"interactive","can_send_input":true,"worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, `{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Claude1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"seguir frente activo"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Claude1","texto":"continua trabajo actual","mailbox_id":%d,"mailbox_kind":"autonomia"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("deberia procesar al menos una orden, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if sendOrder.Estado == "completada" {
		t.Fatalf("autonomia no deberia completarse solo por progreso: %+v", sendOrder)
	}
	if strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"last_progress"`) {
		t.Fatalf("autonomia no deberia marcarse con last_progress: %s", sendOrder.ResultadoJSON)
	}
}

func TestProcesarRuntimeOrdersBatchCompletaMailboxEntregadoConSalidaPosterior(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		ExternalSessionID: "sess-bootstrap-output",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-bootstrap-output")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	deliveredAt := now.Add(-2 * time.Minute)
	outputAt := now.Add(-30 * time.Second)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex1-bootstrap",
		"tmux_pane_id":          "%77",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
		"can_send_input":        false,
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "running",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"ready_at":              deliveredAt.Format(time.RFC3339Nano),
		"last_output_at":        outputAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":          true,
		"heartbeat_at":   now.Format(time.RFC3339Nano),
		"ready_at":       deliveredAt.Format(time.RFC3339Nano),
		"last_output_at": outputAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-codex1-bootstrap","tmux_pane_id":"%%77","mailbox_delivery_mode":"%s","can_send_input":false,"worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, runtimeagente.MailboxDeliveryBootstrapOnly, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "governance_refresh",
		PayloadJSON: `{"hash":"canon"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"continua trabajo actual","mailbox_id":%d,"mailbox_kind":"governance_refresh"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	orderBaseline := deliveredAt.Add(30 * time.Second)
	if _, err := DB.Exec(`UPDATE runtime_orders SET created_at=?, updated_at=? WHERE id=?`, orderBaseline, orderBaseline, sendID); err != nil {
		t.Fatalf("backdate order created_at: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar la orden entregada, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("la orden deberia completarse con salida posterior: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"last_output"`) {
		t.Fatalf("resultado sin receipt_source last_output: %s", sendOrder.ResultadoJSON)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "consumido" {
		t.Fatalf("mailbox deberia quedar consumido: %+v", msg)
	}
}

func TestProcesarRuntimeOrdersBatchMicroprogramacionConTranscriptBloqueoFallaOrden(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Qwen1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "proyecto")
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto",
		Nombre:  "Proyecto smoke",
		RutaAbs: workingDir,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Qwen1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "ollama-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-micro-bloqueo")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	deliveredAt := now.Add(-2 * time.Minute)
	outputAt := now.Add(-30 * time.Second)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-qwen1-micro",
		"tmux_pane_id":          "%88",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
		"can_send_input":        true,
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "running",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"ready_at":              deliveredAt.Format(time.RFC3339Nano),
		"last_output_at":        outputAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":          true,
		"heartbeat_at":   now.Format(time.RFC3339Nano),
		"ready_at":       deliveredAt.Format(time.RFC3339Nano),
		"last_output_at": outputAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-qwen1-micro","tmux_pane_id":"%%88","mailbox_delivery_mode":"%s","can_send_input":true,"worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, runtimeagente.MailboxDeliveryInteractive, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, `{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Qwen1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"source":"microprogramacion"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Qwen1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Qwen1","texto":"micro","source":"microprogramacion","microprogramacion":{"especificacion_id":1,"archivo_objetivo":"identidad/normalizar.go","simbolo_objetivo":"NormalizarIdentificador","write_set":["identidad/normalizar.go"]},"mailbox_id":%d,"mailbox_kind":"instruction"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	runtimeID := int64(0)
	if handle.RuntimeID != nil {
		runtimeID = *handle.RuntimeID
	}
	if runtimeID <= 0 {
		t.Fatalf("runtimeID invalido en handle: %+v", handle)
	}
	if _, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtimeID,
		HandleID:       &handle.ID,
		Agente:         "Qwen1",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "BLOQUEO: contexto insuficiente",
		NormalizedText: "bloqueo: contexto insuficiente",
	}); err != nil {
		t.Fatalf("registrar transcript bloqueo: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar la orden microprogramada bloqueada, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "fallida" {
		t.Fatalf("la orden deberia fallar por bloqueo explicito del agente: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"transcript_blocked"`) {
		t.Fatalf("resultado sin receipt_source transcript_blocked: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"blocked":true`) {
		t.Fatalf("resultado sin blocked=true: %s", sendOrder.ResultadoJSON)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "consumido" {
		t.Fatalf("mailbox deberia quedar consumido tras bloqueo explicito: %+v", msg)
	}
}

func TestProcesarRuntimeOrdersBatchInteractiveStartingNoAceptaReceiptLastOutput(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-interactive-starting")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	deliveredAt := now.Add(-2 * time.Minute)
	outputAt := now.Add(-30 * time.Second)
	writeJSON := func(path string, payload map[string]any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-gemini1-starting",
		"tmux_pane_id":          "%88",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
		"can_send_input":        true,
	})
	writeJSON(statusPath, map[string]any{
		"state":                 "starting",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"last_output_at":        outputAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":          true,
		"heartbeat_at":   now.Format(time.RFC3339Nano),
		"last_output_at": outputAt.Format(time.RFC3339Nano),
	})
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-gemini1-starting","tmux_pane_id":"%%88","can_send_input":true,"mailbox_delivery_mode":"interactive","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(
		`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-gemini1-starting/%88', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Gemini1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"kind":"pipeline_local"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Gemini1","texto":"continua trabajo actual","mailbox_id":%d,"mailbox_kind":"pipeline_local"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	order, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := retenerRuntimeOrderSendInstructionNotificada(order, map[string]any{
		"to_agente":    "Gemini1",
		"texto":        "continua trabajo actual",
		"mailbox_id":   mailboxID,
		"mailbox_kind": "pipeline_local",
	}, "interactive dispatch pending receipt", deliveredAt); err != nil {
		t.Fatalf("retenerRuntimeOrderSendInstructionNotificada: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reevaluar la orden notificada, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("interactive starting no deberia completar la orden por last_output: %+v", sendOrder)
	}
	if strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"last_output"`) {
		t.Fatalf("interactive starting no deberia aceptar receipt_source last_output: %s", sendOrder.ResultadoJSON)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "entregado" && msg.Estado != "pendiente" {
		t.Fatalf("mailbox no deberia quedar consumido mientras el worker sigue starting: %+v", msg)
	}
}

func TestRuntimeOrderSendInstructionTextoPrefiereInstructionParaPipelineLocal(t *testing.T) {
	payload := map[string]any{
		"mailbox_kind": "pipeline_local",
		"texto":        "continua trabajo actual",
		"instruction":  "FASE: implementacion\nTAREA: #535",
	}
	if got := runtimeOrderSendInstructionTexto(payload); got != "FASE: implementacion\nTAREA: #535" {
		t.Fatalf("pipeline_local deberia priorizar instruction real, got=%q", got)
	}
}

func TestProcesarRuntimeOrdersBatchInteractivePipelineLocalReadyNoAceptaReceiptLastOutput(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-interactive-ready")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	deliveredAt := now.Add(-2 * time.Minute)
	outputAt := now.Add(-30 * time.Second)
	writeJSON := func(path string, payload map[string]any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-gemini1-ready",
		"tmux_pane_id":          "%90",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
		"can_send_input":        true,
	})
	writeJSON(statusPath, map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"ready_at":              deliveredAt.Format(time.RFC3339Nano),
		"last_output_at":        outputAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":          true,
		"heartbeat_at":   now.Format(time.RFC3339Nano),
		"ready_at":       deliveredAt.Format(time.RFC3339Nano),
		"last_output_at": outputAt.Format(time.RFC3339Nano),
	})
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-gemini1-ready","tmux_pane_id":"%%90","can_send_input":true,"mailbox_delivery_mode":"interactive","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(
		`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-gemini1-ready/%90', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Gemini1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"kind":"pipeline_local"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Gemini1","texto":"continua trabajo actual","mailbox_id":%d,"mailbox_kind":"pipeline_local"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	order, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := retenerRuntimeOrderSendInstructionNotificada(order, map[string]any{
		"to_agente":    "Gemini1",
		"texto":        "continua trabajo actual",
		"mailbox_id":   mailboxID,
		"mailbox_kind": "pipeline_local",
	}, "interactive dispatch pending receipt", deliveredAt); err != nil {
		t.Fatalf("retenerRuntimeOrderSendInstructionNotificada: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reevaluar la orden notificada, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("pipeline_local ready no deberia completar la orden solo por last_output: %+v", sendOrder)
	}
	if strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"last_output"`) {
		t.Fatalf("pipeline_local ready no deberia aceptar receipt_source last_output: %s", sendOrder.ResultadoJSON)
	}
}

func TestProcesarRuntimeOrdersBatchNoAceptaLastOutputPrevioALaCreacionDeLaOrden(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-interactive-baseline")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	deliveredAt := now.Add(-2 * time.Minute)
	outputAt := now.Add(-30 * time.Second)
	writeJSON := func(path string, payload map[string]any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-gemini1-baseline",
		"tmux_pane_id":          "%91",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
		"can_send_input":        true,
	})
	writeJSON(statusPath, map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"ready_at":              deliveredAt.Format(time.RFC3339Nano),
		"last_output_at":        outputAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":          true,
		"heartbeat_at":   now.Format(time.RFC3339Nano),
		"ready_at":       deliveredAt.Format(time.RFC3339Nano),
		"last_output_at": outputAt.Format(time.RFC3339Nano),
	})
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-gemini1-baseline","tmux_pane_id":"%%91","can_send_input":true,"mailbox_delivery_mode":"interactive","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(
		`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-gemini1-baseline/%91', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Gemini1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"kind":"pipeline_local"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Gemini1","texto":"continua trabajo actual","mailbox_id":%d,"mailbox_kind":"pipeline_local"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	order, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := retenerRuntimeOrderSendInstructionNotificada(order, map[string]any{
		"to_agente":    "Gemini1",
		"texto":        "continua trabajo actual",
		"mailbox_id":   mailboxID,
		"mailbox_kind": "pipeline_local",
	}, "interactive dispatch pending receipt", deliveredAt); err != nil {
		t.Fatalf("retenerRuntimeOrderSendInstructionNotificada: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reevaluar la orden notificada, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("no deberia aceptar last_output previo a la creacion de la orden: %+v", sendOrder)
	}
	if strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"last_output"`) {
		t.Fatalf("no deberia aceptar receipt_source last_output previo a la orden: %s", sendOrder.ResultadoJSON)
	}
}

func TestProcesarRuntimeOrdersBatchNoAceptaWorkerActivityPreviaALaCreacionDeLaOrden(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
	workerDir := filepath.Join(tmp, "worker-premium-baseline")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	deliveredAt := now.Add(-2 * time.Minute)
	activityAt := now.Add(-30 * time.Second)
	writeJSON := func(path string, payload map[string]any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"driver":                "process_pty_cli",
		"transport":             "pty_broker",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
		"can_send_input":        false,
	})
	writeJSON(statusPath, map[string]any{
		"state":                 "running",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"last_output_at":        activityAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":          true,
		"heartbeat_at":   now.Format(time.RFC3339Nano),
		"last_output_at": activityAt.Format(time.RFC3339Nano),
	})
	metaJSON := fmt.Sprintf(`{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","stdin_path":"%s","supervisor_ref":"%s","mailbox_delivery_mode":"session_resume","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`,
		filepath.Join(workerDir, "pty.stdin"),
		workerDir,
		manifestPath,
		statusPath,
		heartbeatPath,
	)
	if _, err := DB.Exec(
		`UPDATE runtime_handles SET transporte='cli', handle_kind='process', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false,"mailbox_delivery_mode":"session_resume"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"haz una micro refactorizacion","texto":"continua trabajo actual"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","instruction":"haz una micro refactorizacion","texto":"continua trabajo actual","mailbox_id":%d,"mailbox_kind":"pipeline_local"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	order, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := retenerRuntimeOrderSendInstructionNotificada(order, map[string]any{
		"to_agente":    "Codex1",
		"instruction":  "haz una micro refactorizacion",
		"texto":        "continua trabajo actual",
		"mailbox_id":   mailboxID,
		"mailbox_kind": "pipeline_local",
	}, "interactive dispatch pending receipt", deliveredAt); err != nil {
		t.Fatalf("retenerRuntimeOrderSendInstructionNotificada: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reevaluar la orden notificada, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("no deberia aceptar worker_activity previa a la creacion de la orden: %+v", sendOrder)
	}
	if strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"worker_activity"`) {
		t.Fatalf("no deberia aceptar receipt_source worker_activity previo a la orden: %s", sendOrder.ResultadoJSON)
	}
}

func TestRuntimeOrderSendInstructionDebeEsperarReceiptInteractivoIncluyePipelineLocal(t *testing.T) {
	order := &RuntimeOrder{Tipo: "send_instruction"}
	handle := &RuntimeHandle{
		Transporte:       "tmux",
		CapabilitiesJSON: `{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`,
		MetadataJSON:     `{"driver":"tmux_cli_session","can_send_input":true,"mailbox_delivery_mode":"interactive"}`,
	}
	if !runtimeOrderSendInstructionDebeEsperarReceiptInteractivo(order, handle, map[string]any{
		"mailbox_id":   1,
		"mailbox_kind": "pipeline_local",
	}) {
		t.Fatal("pipeline_local interactivo deberia esperar receipt")
	}
}

func TestRuntimeOrderSendInstructionDebeEsperarReceiptInteractivoIncluyePipelineLocalSessionResumePremium(t *testing.T) {
	order := &RuntimeOrder{Tipo: "send_instruction"}
	handle := &RuntimeHandle{
		Transporte:       "cli",
		CapabilitiesJSON: `{"can_send_input":false,"mailbox_delivery_mode":"session_resume"}`,
		MetadataJSON:     `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","stdin_path":"/tmp/codex.stdin","supervisor_ref":"/tmp/codex.run","external_session_id":"sess-codex-1","mailbox_delivery_mode":"session_resume","can_send_input":false}`,
	}
	payload := map[string]any{
		"mailbox_id":   1,
		"mailbox_kind": "pipeline_local",
	}
	if !runtimeOrderSendInstructionDebeEsperarReceiptInteractivo(order, handle, payload) {
		t.Fatal("pipeline_local premium por session_resume deberia esperar receipt")
	}
}

func TestRuntimeOrderSendInstructionPermiteReceiptLastOutputNoAceptaPipelinePremiumBootstrapOnly(t *testing.T) {
	handle := &RuntimeHandle{
		Transporte:       "tmux",
		CapabilitiesJSON: `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`,
		MetadataJSON:     `{"driver":"tmux_cli_session","mailbox_delivery_mode":"bootstrap_only"}`,
	}
	snap := &runtimeagente.WorkerSnapshot{
		Manifest: &runtimeagente.WorkerManifest{
			Driver:              "tmux_cli_session",
			Transport:           "tmux",
			MailboxDeliveryMode: runtimeagente.MailboxDeliveryBootstrapOnly,
		},
		Status: &runtimeagente.WorkerStatus{
			State: "ready",
			Alive: true,
		},
		Heartbeat: &runtimeagente.WorkerHeartbeat{
			Alive: true,
		},
	}
	if runtimeOrderSendInstructionPermiteReceiptLastOutput(handle, snap, map[string]any{
		"mailbox_kind": "pipeline_local",
	}) {
		t.Fatal("pipeline_local premium no deberia aceptar last_output aunque el handle siga bootstrap_only")
	}
}

func TestRuntimeOrderSendInstructionPermiteReceiptLastOutputNoAceptaNudgeTMUXSessionResume(t *testing.T) {
	handle := &RuntimeHandle{
		Transporte:       "tmux",
		CapabilitiesJSON: `{"can_send_input":false,"mailbox_delivery_mode":"session_resume"}`,
		MetadataJSON:     `{"driver":"tmux_cli_session","can_send_input":false,"mailbox_delivery_mode":"session_resume","external_session_id":"sess-codex-1"}`,
	}
	snap := &runtimeagente.WorkerSnapshot{
		Manifest: &runtimeagente.WorkerManifest{
			Driver:              "tmux_cli_session",
			Transport:           "tmux",
			MailboxDeliveryMode: runtimeagente.MailboxDeliverySessionResume,
		},
		Status: &runtimeagente.WorkerStatus{
			State: "ready",
			Alive: true,
		},
		Heartbeat: &runtimeagente.WorkerHeartbeat{
			Alive: true,
		},
	}
	if runtimeOrderSendInstructionPermiteReceiptLastOutput(handle, snap, map[string]any{
		"mailbox_kind": "nudge",
	}) {
		t.Fatal("nudge tmux session_resume no deberia aceptar last_output como receipt")
	}
}

func TestRuntimeOrderSendInstructionPermiteReceiptLastOutputNoAceptaAutonomiaContinuarTrabajoTMUXSessionResume(t *testing.T) {
	handle := &RuntimeHandle{
		Transporte:       "tmux",
		CapabilitiesJSON: `{"can_send_input":false,"mailbox_delivery_mode":"session_resume"}`,
		MetadataJSON:     `{"driver":"tmux_cli_session","can_send_input":false,"mailbox_delivery_mode":"session_resume","external_session_id":"sess-codex-1"}`,
	}
	snap := &runtimeagente.WorkerSnapshot{
		Manifest: &runtimeagente.WorkerManifest{
			Driver:              "tmux_cli_session",
			Transport:           "tmux",
			MailboxDeliveryMode: runtimeagente.MailboxDeliverySessionResume,
		},
		Status: &runtimeagente.WorkerStatus{
			State: "ready",
			Alive: true,
		},
		Heartbeat: &runtimeagente.WorkerHeartbeat{
			Alive: true,
		},
	}
	if runtimeOrderSendInstructionPermiteReceiptLastOutput(handle, snap, map[string]any{
		"mailbox_kind": "autonomia",
		"accion":       "continuar_trabajo",
	}) {
		t.Fatal("autonomia continuar_trabajo tmux session_resume no deberia aceptar last_output como receipt")
	}
}

func TestRuntimeOrderSendInstructionPremiumActivityEvidenceAceptaProcessPTYConActividadPosterior(t *testing.T) {
	now := time.Now().UTC()
	lastOutput := now.Add(-5 * time.Second)
	handle := &RuntimeHandle{
		Transporte:       "cli",
		CapabilitiesJSON: `{"can_send_input":false,"mailbox_delivery_mode":"session_resume"}`,
		MetadataJSON:     `{"driver":"process_pty_cli","mailbox_delivery_mode":"session_resume"}`,
	}
	snap := &runtimeagente.WorkerSnapshot{
		Manifest: &runtimeagente.WorkerManifest{
			Driver:              "process_pty_cli",
			Transport:           "pty_broker",
			MailboxDeliveryMode: runtimeagente.MailboxDeliverySessionResume,
		},
		Status: &runtimeagente.WorkerStatus{
			State:        "running",
			Alive:        true,
			LastOutputAt: lastOutput.Format(time.RFC3339Nano),
		},
		Heartbeat: &runtimeagente.WorkerHeartbeat{
			Alive:       true,
			HeartbeatAt: now.Format(time.RFC3339Nano),
		},
	}
	ok, source, receiptAt, err := runtimeOrderSendInstructionPremiumActivityEvidence(nil, handle, snap, map[string]any{
		"mailbox_kind": "pipeline_local",
	}, now.Add(-10*time.Second))
	if err != nil {
		t.Fatalf("premium activity evidence: %v", err)
	}
	if !ok || source != "worker_activity" || !receiptAt.Equal(lastOutput) {
		t.Fatalf("evidencia inesperada ok=%t source=%q receiptAt=%s", ok, source, receiptAt)
	}
}

func TestProcesarRuntimeOrdersBatchNudgeTMUXSessionResumeAceptaTMUXPaneActivityPosterior(t *testing.T) {
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
	workerDir := filepath.Join(tmp, "worker-codex-tmux-nudge-activity")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	fakeTmux := writeFakeTMUXGeminiAcceptEditsScriptDB(t, tmp)
	writeJSON := func(path string, payload map[string]any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	deliveredAt := time.Now().UTC().Add(-30 * time.Second)
	writeJSON(manifestPath, map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_command":          fakeTmux,
		"tmux_session":          "orq-codex1-nudge-activity",
		"tmux_pane_id":          "%97",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
		"can_send_input":        false,
	})
	writeJSON(statusPath, map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"updated_at":            deliveredAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"heartbeat_at": deliveredAt.Format(time.RFC3339Nano),
	})
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_session":"orq-codex1-nudge-activity","tmux_pane_id":"%%97","mailbox_delivery_mode":"%s","can_send_input":false,"external_session_id":"sess-codex1-nudge-activity","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, fakeTmux, runtimeagente.MailboxDeliverySessionResume, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false,"mailbox_delivery_mode":"session_resume"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"continua con el siguiente slice util"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"continua con el siguiente slice util","mailbox_id":%d,"mailbox_kind":"nudge","external_session_id":"sess-codex1-nudge-activity"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	order, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := retenerRuntimeOrderSendInstructionNotificada(order, map[string]any{
		"to_agente":    "Codex1",
		"texto":        "continua con el siguiente slice util",
		"mailbox_id":   mailboxID,
		"mailbox_kind": "nudge",
	}, "session_resume dispatch pending receipt", deliveredAt); err != nil {
		t.Fatalf("retenerRuntimeOrderSendInstructionNotificada: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	activityAt := time.Now().UTC()
	writeJSON(statusPath, map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"updated_at":            activityAt.Format(time.RFC3339Nano),
		"last_output_at":        activityAt.Format(time.RFC3339Nano),
		"last_progress_at":      activityAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":            true,
		"heartbeat_at":     activityAt.Format(time.RFC3339Nano),
		"last_output_at":   activityAt.Format(time.RFC3339Nano),
		"last_progress_at": activityAt.Format(time.RFC3339Nano),
	})

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar el nudge tmux con actividad de pane, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("nudge tmux session_resume deberia completarse con actividad en pane: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"tmux_pane_activity"`) {
		t.Fatalf("resultado sin receipt_source tmux_pane_activity: %s", sendOrder.ResultadoJSON)
	}
	if strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"last_output"`) {
		t.Fatalf("nudge tmux session_resume no deberia cerrarse por last_output: %s", sendOrder.ResultadoJSON)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "consumido" {
		t.Fatalf("mailbox deberia quedar consumido tras actividad de pane: %+v", msg)
	}
}

func TestProcesarRuntimeOrdersBatchAutonomiaContinuarTrabajoTMUXSessionResumeAceptaTMUXPaneActivityPosterior(t *testing.T) {
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
	workerDir := filepath.Join(tmp, "worker-codex-tmux-autonomia-activity")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	fakeTmux := writeFakeTMUXGeminiAcceptEditsScriptDB(t, tmp)
	writeJSON := func(path string, payload map[string]any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	deliveredAt := time.Now().UTC().Add(-30 * time.Second)
	writeJSON(manifestPath, map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_command":          fakeTmux,
		"tmux_session":          "orq-codex1-autonomia-activity",
		"tmux_pane_id":          "%98",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
		"can_send_input":        false,
	})
	writeJSON(statusPath, map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"updated_at":            deliveredAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"heartbeat_at": deliveredAt.Format(time.RFC3339Nano),
	})
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_session":"orq-codex1-autonomia-activity","tmux_pane_id":"%%98","mailbox_delivery_mode":"%s","can_send_input":false,"external_session_id":"sess-codex1-autonomia-activity","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, fakeTmux, runtimeagente.MailboxDeliverySessionResume, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false,"mailbox_delivery_mode":"session_resume"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","instruction":"sigue con el siguiente slice util","texto":"sigue con el siguiente slice util"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","accion":"continuar_trabajo","texto":"sigue con el siguiente slice util","mailbox_id":%d,"mailbox_kind":"autonomia","external_session_id":"sess-codex1-autonomia-activity","post_remediation":true,"remediation_kind":"reassign","verification_key":"reassign:41:Codex0:Codex1","origin_agent":"Codex0","tarea_id":41}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	order, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := retenerRuntimeOrderSendInstructionNotificada(order, map[string]any{
		"to_agente":        "Codex1",
		"accion":           "continuar_trabajo",
		"texto":            "sigue con el siguiente slice util",
		"mailbox_id":       mailboxID,
		"mailbox_kind":     "autonomia",
		"post_remediation": true,
		"remediation_kind": "reassign",
		"verification_key": "reassign:41:Codex0:Codex1",
		"origin_agent":     "Codex0",
		"tarea_id":         int64(41),
	}, "session_resume dispatch pending receipt", deliveredAt); err != nil {
		t.Fatalf("retenerRuntimeOrderSendInstructionNotificada: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	activityAt := time.Now().UTC()
	writeJSON(statusPath, map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"updated_at":            activityAt.Format(time.RFC3339Nano),
		"last_output_at":        activityAt.Format(time.RFC3339Nano),
		"last_progress_at":      activityAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":            true,
		"heartbeat_at":     activityAt.Format(time.RFC3339Nano),
		"last_output_at":   activityAt.Format(time.RFC3339Nano),
		"last_progress_at": activityAt.Format(time.RFC3339Nano),
	})

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar la autonomia tmux con actividad de pane, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("autonomia tmux session_resume deberia completarse con actividad en pane: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"tmux_pane_activity"`) {
		t.Fatalf("resultado sin receipt_source tmux_pane_activity: %s", sendOrder.ResultadoJSON)
	}
	for _, fragment := range []string{
		`"post_remediation":true`,
		`"remediation_kind":"reassign"`,
		`"verification_key":"reassign:41:Codex0:Codex1"`,
		`"origin_agent":"Codex0"`,
		`"tarea_id":41`,
	} {
		if !strings.Contains(sendOrder.ResultadoJSON, fragment) {
			t.Fatalf("resultado sin %s: %s", fragment, sendOrder.ResultadoJSON)
		}
	}
	if strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"last_output"`) {
		t.Fatalf("autonomia tmux session_resume no deberia cerrarse por last_output: %s", sendOrder.ResultadoJSON)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "consumido" {
		t.Fatalf("mailbox deberia quedar consumido tras actividad de pane: %+v", msg)
	}
}

func TestRetenerRuntimeOrderSendInstructionNotificadaPreservaTrackingPostRemediation(t *testing.T) {
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
	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","texto":"sigue","verification_key":"reassign:41:Codex0:Codex1"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","accion":"continuar_trabajo","mailbox_id":%d,"mailbox_kind":"autonomia","post_remediation":true,"remediation_kind":"reassign","verification_key":"reassign:41:Codex0:Codex1","origin_agent":"Codex0","tarea_id":41}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	notifiedAt := time.Now().UTC()
	if err := retenerRuntimeOrderSendInstructionNotificada(order, map[string]any{
		"to_agente":        "Codex1",
		"accion":           "continuar_trabajo",
		"mailbox_id":       mailboxID,
		"mailbox_kind":     "autonomia",
		"post_remediation": true,
		"remediation_kind": "reassign",
		"verification_key": "reassign:41:Codex0:Codex1",
		"origin_agent":     "Codex0",
		"tarea_id":         int64(41),
	}, "waiting receipt", notifiedAt); err != nil {
		t.Fatalf("retenerRuntimeOrderSendInstructionNotificada: %v", err)
	}
	order, err = GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	for _, fragment := range []string{
		`"dispatch_state":"notified"`,
		`"post_remediation":true`,
		`"remediation_kind":"reassign"`,
		`"verification_key":"reassign:41:Codex0:Codex1"`,
		`"origin_agent":"Codex0"`,
		`"tarea_id":41`,
	} {
		if !strings.Contains(order.ResultadoJSON, fragment) {
			t.Fatalf("resultado notificado sin %s: %s", fragment, order.ResultadoJSON)
		}
	}
	if strings.Contains(order.ResultadoJSON, `"mailbox_only":true`) {
		t.Fatalf("retencion notificada no deberia marcar mailbox_only sin reason legacy: %s", order.ResultadoJSON)
	}
}

func TestCompletarRuntimeOrderSendInstructionDiferidaAMailboxNoMarcaMailboxOnlySinReasonLegacy(t *testing.T) {
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
	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"sigue"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","mailbox_id":%d,"mailbox_kind":"nudge","texto":"sigue"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := completarRuntimeOrderSendInstructionDiferidaAMailbox(order, map[string]any{
		"to_agente":    "Codex1",
		"mailbox_id":   mailboxID,
		"mailbox_kind": "nudge",
		"texto":        "sigue",
	}, "mailbox ya consumido"); err != nil {
		t.Fatalf("completarRuntimeOrderSendInstructionDiferidaAMailbox: %v", err)
	}
	order, err = GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if strings.Contains(order.ResultadoJSON, `"mailbox_only":true`) {
		t.Fatalf("resultado no deberia marcar mailbox_only para cierre no legacy: %s", order.ResultadoJSON)
	}
}

func TestProcesarRuntimeOrdersBatchNudgeTMUXSessionResumeAceptaActividadDeTranscriptPosterior(t *testing.T) {
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
	workerDir := filepath.Join(tmp, "worker-codex-tmux-transcript-activity")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	fakeTmux := writeFakeTMUXGeminiAcceptEditsScriptDB(t, tmp)
	writeJSON := func(path string, payload map[string]any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	deliveredAt := time.Now().UTC().Add(-30 * time.Second)
	staleAt := deliveredAt.Add(-5 * time.Second)
	writeJSON(manifestPath, map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_command":          fakeTmux,
		"tmux_session":          "orq-codex1-transcript-activity",
		"tmux_pane_id":          "%99",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
		"can_send_input":        false,
	})
	writeJSON(statusPath, map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"updated_at":            staleAt.Format(time.RFC3339Nano),
		"last_output_at":        staleAt.Format(time.RFC3339Nano),
		"last_progress_at":      staleAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":            true,
		"heartbeat_at":     staleAt.Format(time.RFC3339Nano),
		"last_output_at":   staleAt.Format(time.RFC3339Nano),
		"last_progress_at": staleAt.Format(time.RFC3339Nano),
	})
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_session":"orq-codex1-transcript-activity","tmux_pane_id":"%%99","mailbox_delivery_mode":"%s","can_send_input":false,"external_session_id":"sess-codex1-transcript-activity","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, fakeTmux, runtimeagente.MailboxDeliverySessionResume, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false,"mailbox_delivery_mode":"session_resume"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"continua con el siguiente slice util"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"continua con el siguiente slice util","mailbox_id":%d,"mailbox_kind":"nudge","external_session_id":"sess-codex1-transcript-activity"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	order, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := retenerRuntimeOrderSendInstructionNotificada(order, map[string]any{
		"to_agente":    "Codex1",
		"texto":        "continua con el siguiente slice util",
		"mailbox_id":   mailboxID,
		"mailbox_kind": "nudge",
	}, "session_resume dispatch pending receipt", deliveredAt); err != nil {
		t.Fatalf("retenerRuntimeOrderSendInstructionNotificada: %v", err)
	}
	activityAt := time.Now().UTC()
	if _, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtime.ID,
		HandleID:       &handle.ID,
		Agente:         "Codex1",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "• Ran go test ./db -run 'TestResolverBootstrapRuntimeLeasePendiente.*' -count=1",
		NormalizedText: "ran go test ./db -run testresolverbootstrapruntimeleasependiente count=1",
		CreatedAt:      activityAt,
	}); err != nil {
		t.Fatalf("registrar transcript: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar el nudge tmux con transcript posterior, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("nudge tmux session_resume deberia completarse con transcript posterior: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"tmux_transcript_activity"`) {
		t.Fatalf("resultado sin receipt_source tmux_transcript_activity: %s", sendOrder.ResultadoJSON)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "consumido" {
		t.Fatalf("mailbox deberia quedar consumido tras transcript posterior: %+v", msg)
	}
}

func TestProcesarRuntimeOrdersBatchNoAceptaTMUXPaneActivityPreviaALaCreacionDeLaOrden(t *testing.T) {
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
	workerDir := filepath.Join(tmp, "worker-codex-tmux-stale-activity")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	fakeTmux := writeFakeTMUXGeminiAcceptEditsScriptDB(t, tmp)
	writeJSON := func(path string, payload map[string]any) {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	deliveredAt := time.Now().UTC().Add(-2 * time.Minute)
	staleActivityAt := time.Now().UTC().Add(-20 * time.Second)
	writeJSON(manifestPath, map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_command":          fakeTmux,
		"tmux_session":          "orq-codex1-stale-activity",
		"tmux_pane_id":          "%98",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
		"can_send_input":        false,
	})
	writeJSON(statusPath, map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"updated_at":            staleActivityAt.Format(time.RFC3339Nano),
		"last_output_at":        staleActivityAt.Format(time.RFC3339Nano),
		"last_progress_at":      staleActivityAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":            true,
		"heartbeat_at":     staleActivityAt.Format(time.RFC3339Nano),
		"last_output_at":   staleActivityAt.Format(time.RFC3339Nano),
		"last_progress_at": staleActivityAt.Format(time.RFC3339Nano),
	})
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_session":"orq-codex1-stale-activity","tmux_pane_id":"%%98","mailbox_delivery_mode":"%s","can_send_input":false,"external_session_id":"sess-codex1-stale-activity","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, fakeTmux, runtimeagente.MailboxDeliverySessionResume, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false,"mailbox_delivery_mode":"session_resume"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"continua con el siguiente slice util"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"continua con el siguiente slice util","mailbox_id":%d,"mailbox_kind":"nudge","external_session_id":"sess-codex1-stale-activity"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	order, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order: %v", err)
	}
	if err := retenerRuntimeOrderSendInstructionNotificada(order, map[string]any{
		"to_agente":    "Codex1",
		"texto":        "continua con el siguiente slice util",
		"mailbox_id":   mailboxID,
		"mailbox_kind": "nudge",
	}, "session_resume dispatch pending receipt", deliveredAt); err != nil {
		t.Fatalf("retenerRuntimeOrderSendInstructionNotificada: %v", err)
	}
	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reevaluar el nudge tmux notificado, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("no deberia aceptar tmux_pane_activity previa a la creacion de la orden: %+v", sendOrder)
	}
	if strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"tmux_pane_activity"`) {
		t.Fatalf("no deberia aceptar receipt_source tmux_pane_activity previo a la orden: %s", sendOrder.ResultadoJSON)
	}
}

func TestProcesarRuntimeOrdersBatchMicroprogramacionConTranscriptPatchCompletaOrden(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Qwen1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "proyecto")
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto",
		Nombre:  "Proyecto smoke",
		RutaAbs: workingDir,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Qwen1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "ollama-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-micro-patch")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	deliveredAt := now.Add(-2 * time.Minute)
	outputAt := now.Add(-30 * time.Second)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-qwen1-micro",
		"tmux_pane_id":          "%89",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
		"can_send_input":        true,
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "running",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"ready_at":              deliveredAt.Format(time.RFC3339Nano),
		"last_output_at":        outputAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":          true,
		"heartbeat_at":   now.Format(time.RFC3339Nano),
		"ready_at":       deliveredAt.Format(time.RFC3339Nano),
		"last_output_at": outputAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-qwen1-micro","tmux_pane_id":"%%89","mailbox_delivery_mode":"%s","can_send_input":true,"worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, runtimeagente.MailboxDeliveryInteractive, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, `{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Qwen1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"source":"microprogramacion"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Qwen1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Qwen1","texto":"micro","source":"microprogramacion","microprogramacion":{"especificacion_id":1,"archivo_objetivo":"identidad/normalizar.go","simbolo_objetivo":"NormalizarIdentificador","write_set":["identidad/normalizar.go"]},"mailbox_id":%d,"mailbox_kind":"instruction"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	runtimeID := int64(0)
	if handle.RuntimeID != nil {
		runtimeID = *handle.RuntimeID
	}
	if runtimeID <= 0 {
		t.Fatalf("runtimeID invalido en handle: %+v", handle)
	}
	if _, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtimeID,
		HandleID:       &handle.ID,
		Agente:         "Qwen1",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "PATCH_UNIFICADO:\n--- a/identidad/normalizar.go\n+++ b/identidad/normalizar.go",
		NormalizedText: "patch_unificado diff --git",
	}); err != nil {
		t.Fatalf("registrar transcript patch: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar la orden microprogramada con patch, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("la orden deberia completarse con transcript patch: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"transcript_patch"`) {
		t.Fatalf("resultado sin receipt_source transcript_patch: %s", sendOrder.ResultadoJSON)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "consumido" {
		t.Fatalf("mailbox deberia quedar consumido tras transcript patch: %+v", msg)
	}
}

func TestProcesarRuntimeOrdersBatchMicroprogramacionConPanePatchCompletaOrden(t *testing.T) {
	tmp := prepararDBTemporal(t)
	fakeTmux := writeFakeTMUXPanePatchScriptDB(t, tmp)
	if err := RegistrarAgente("Qwen1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "proyecto")
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto",
		Nombre:  "Proyecto smoke",
		RutaAbs: workingDir,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Qwen1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "ollama-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-micro-pane-patch")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	deliveredAt := now.Add(-2 * time.Minute)
	heartbeatAt := now.Add(-15 * time.Second)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_command":          fakeTmux,
		"tmux_session":          "orq-qwen1-pane-patch",
		"tmux_pane_id":          "%90",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
		"can_send_input":        true,
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "running",
		"alive":                 true,
		"updated_at":            heartbeatAt.Format(time.RFC3339Nano),
		"ready_at":              deliveredAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": heartbeatAt.Format(time.RFC3339Nano),
		"ready_at":     deliveredAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_session":"orq-qwen1-pane-patch","tmux_pane_id":"%%90","mailbox_delivery_mode":"%s","can_send_input":true,"worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, fakeTmux, runtimeagente.MailboxDeliveryInteractive, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, `{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Qwen1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"source":"microprogramacion"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Qwen1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Qwen1","texto":"micro","source":"microprogramacion","microprogramacion":{"especificacion_id":1,"archivo_objetivo":"identidad/normalizar.go","simbolo_objetivo":"NormalizarIdentificador","write_set":["identidad/normalizar.go"]},"mailbox_id":%d,"mailbox_kind":"instruction"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar la orden microprogramada con patch en pane, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("la orden deberia completarse con pane patch: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"tmux_pane_patch"`) {
		t.Fatalf("resultado sin receipt_source tmux_pane_patch: %s", sendOrder.ResultadoJSON)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "consumido" {
		t.Fatalf("mailbox deberia quedar consumido tras pane patch: %+v", msg)
	}
}

func TestRuntimeMicroprogramacionPatchDetectedAceptaBloquesFile(t *testing.T) {
	if !runtimeMicroprogramacionPatchDetected("// FILE: identidad/normalizar.go\npackage identidad\n", "") {
		t.Fatalf("deberia detectar bloques FILE como evidencia de entrega")
	}
}

func TestRuntimeOrderSendInstructionTranscriptPatchEvidenceAceptaFilesSinSnapshot(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "proyecto")
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto",
		Nombre:  "Proyecto",
		RutaAbs: workingDir,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Gemma1",
		ProyectoID:        &proyectoID,
		CWD:               workingDir,
		Herramienta:       "ollama_pool_local",
		ExternalSessionID: "sess-ollama-files",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, `{"driver":"ollama_pool_local","transport":"api","pool_local":true}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Gemma1",
		ProyectoID:   &proyectoID,
		SesionID:     handle.SesionID,
		LogicalState: "activo",
		ProcessState: "corriendo",
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET runtime_id=? WHERE id=?`, runtimeID, handle.ID); err != nil {
		t.Fatalf("asociar handle a runtime: %v", err)
	}
	handle.RuntimeID = &runtimeID
	if _, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtimeID,
		HandleID:       &handle.ID,
		Agente:         "Gemma1",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "// FILE: cabeceras/merge_headers.go\npackage cabeceras\n",
		NormalizedText: "// file: cabeceras/merge_headers.go package cabeceras",
	}); err != nil {
		t.Fatalf("registrar transcript files: %v", err)
	}
	ok, source, _, err := runtimeOrderSendInstructionTranscriptPatchEvidence(&RuntimeInstance{ID: runtimeID}, handle, time.Now().UTC().Add(-time.Minute))
	if err != nil {
		t.Fatalf("transcript patch evidence: %v", err)
	}
	if !ok || source != "transcript_patch" {
		t.Fatalf("receipt inesperado ok=%v source=%q", ok, source)
	}
}

func TestProcesarRuntimeOrdersBatchPipelinePremiumNoAceptaSoloActividadTMUXComoReceipt(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		Agente:            "Gemini1",
		ProyectoID:        &proyectoID,
		CWD:               workingDir,
		Herramienta:       "gemini-cli",
		ExternalSessionID: "sess-gemini-premium-activity",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-gemini-premium-activity")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	fakeTmux := writeFakeTMUXGeminiAcceptEditsScriptDB(t, tmp)
	now := time.Now().UTC()
	deliveredAt := now.Add(-30 * time.Second)
	activityAt := now.Add(-5 * time.Second)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_command":          fakeTmux,
		"tmux_session":          "orq-gemini1-premium-activity",
		"tmux_pane_id":          "%92",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
		"can_send_input":        true,
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"updated_at":            activityAt.Format(time.RFC3339Nano),
		"last_output_at":        activityAt.Format(time.RFC3339Nano),
		"last_progress_at":      activityAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":            true,
		"heartbeat_at":     activityAt.Format(time.RFC3339Nano),
		"last_output_at":   activityAt.Format(time.RFC3339Nano),
		"last_progress_at": activityAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_session":"orq-gemini1-premium-activity","tmux_pane_id":"%%92","mailbox_delivery_mode":"%s","can_send_input":true,"worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, fakeTmux, runtimeagente.MailboxDeliveryInteractive, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, `{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Gemini1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"source":"microrefactor_loop"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Gemini1","texto":"microrefactor_loop","mailbox_id":%d,"mailbox_kind":"pipeline_local","source":"microrefactor_loop","task_id":530}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "pendiente" && sendOrder.Estado != "tomada" && sendOrder.Estado != "ejecutando" {
		t.Fatalf("pipeline_local premium no deberia completarse solo por actividad de pane: %+v (processed=%d)", sendOrder, processed)
	}
	if strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"tmux_pane_activity"`) {
		t.Fatalf("pipeline_local premium no deberia aceptar receipt_source tmux_pane_activity: %s", sendOrder.ResultadoJSON)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado == "consumido" {
		t.Fatalf("mailbox no deberia quedar consumido solo por actividad de pane: %+v", msg)
	}
}

func TestProcesarRuntimeOrdersBatchPipelinePremiumAceptaActividadTranscriptComoReceipt(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		Agente:            "Gemini1",
		ProyectoID:        &proyectoID,
		CWD:               workingDir,
		Herramienta:       "gemini-cli",
		ExternalSessionID: "sess-gemini-premium-transcript",
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
	workerDir := filepath.Join(tmp, "worker-gemini-premium-transcript")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	fakeTmux := writeFakeTMUXGeminiAcceptEditsScriptDB(t, tmp)
	now := time.Now().UTC()
	deliveredAt := now.Add(-30 * time.Second)
	staleActivityAt := now.Add(-20 * time.Second)
	transcriptAt := now.Add(-5 * time.Second)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_command":          fakeTmux,
		"tmux_session":          "orq-gemini1-premium-transcript",
		"tmux_pane_id":          "%93",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
		"can_send_input":        true,
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"updated_at":            staleActivityAt.Format(time.RFC3339Nano),
		"last_output_at":        staleActivityAt.Format(time.RFC3339Nano),
		"last_progress_at":      staleActivityAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":            true,
		"heartbeat_at":     staleActivityAt.Format(time.RFC3339Nano),
		"last_output_at":   staleActivityAt.Format(time.RFC3339Nano),
		"last_progress_at": staleActivityAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_session":"orq-gemini1-premium-transcript","tmux_pane_id":"%%93","mailbox_delivery_mode":"%s","can_send_input":true,"worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, fakeTmux, runtimeagente.MailboxDeliveryInteractive, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, `{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Gemini1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"source":"microrefactor_loop"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Gemini1","texto":"microrefactor_loop","mailbox_id":%d,"mailbox_kind":"pipeline_local","source":"microrefactor_loop","task_id":530}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	if _, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtime.ID,
		HandleID:       &handle.ID,
		Agente:         "Gemini1",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		CreatedAt:      transcriptAt,
		Text:           "• Explored\n└ Read controlplane_support.go, controlplane_entities.go\n",
		NormalizedText: "explored read controlplane_support.go controlplane_entities.go",
	}); err != nil {
		t.Fatalf("registrar transcript: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar la orden premium con transcript posterior, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("pipeline_local premium deberia completarse con transcript posterior: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"tmux_transcript_activity"`) {
		t.Fatalf("resultado sin receipt_source tmux_transcript_activity: %s", sendOrder.ResultadoJSON)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "consumido" {
		t.Fatalf("mailbox deberia quedar consumido tras transcript posterior: %+v", msg)
	}
}

func TestRuntimeHandleListaParaDispatchInteractivoTMUXRequiereSnapshot(t *testing.T) {
	handle := &RuntimeHandle{
		Transporte:   "tmux",
		MetadataJSON: `{"driver":"tmux_cli_session","tmux_session":"orq-gemini1-missing","tmux_pane_id":"%93","mailbox_delivery_mode":"interactive"}`,
	}
	ready, reason := runtimeHandleListaParaDispatchInteractivo(handle, time.Now().UTC())
	if ready {
		t.Fatalf("tmux interactivo sin snapshot no deberia estar listo")
	}
	if reason != "worker_snapshot_missing" {
		t.Fatalf("motivo inesperado: %q", reason)
	}
}

func TestCompletarRuntimeOrderSendInstructionDiferidaAMailboxPipelinePremiumRetienePendiente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Gemini1", "programador"); err != nil {
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
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Gemini1","texto":"continua trabajo actual","mailbox_kind":"pipeline_local"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	order, err := GetRuntimeOrder(sendID)
	if err != nil || order == nil {
		t.Fatalf("get order: %+v err=%v", order, err)
	}
	payload := map[string]any{
		"to_agente":    "Gemini1",
		"texto":        "continua trabajo actual",
		"mailbox_kind": "pipeline_local",
	}
	if err := completarRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "runtime no disponible para entrega inmediata"); err != nil {
		t.Fatalf("completar diferida mailbox: %v", err)
	}
	order, err = GetRuntimeOrder(sendID)
	if err != nil || order == nil {
		t.Fatalf("get order final: %+v err=%v", order, err)
	}
	if order.Estado != "pendiente" {
		t.Fatalf("pipeline_local premium no deberia quedar completada por mailbox durable: %+v", order)
	}
	if !strings.Contains(order.ResultadoJSON, `"delivery_state":"queued"`) {
		t.Fatalf("resultado sin delivery_state queued: %s", order.ResultadoJSON)
	}
	if strings.Contains(order.ResultadoJSON, `"dispatch_state":"delivered"`) {
		t.Fatalf("pipeline_local premium no deberia marcarse delivered por mailbox durable: %s", order.ResultadoJSON)
	}
}

func TestRuntimeOrderSendInstructionPermiteReceiptTranscriptPatchSoloParaGit(t *testing.T) {
	if !runtimeOrderSendInstructionPermiteReceiptTranscriptPatch(map[string]any{
		"source": "microprogramacion",
		"microprogramacion": map[string]any{
			"formato_salida": "git_worktree+evidencia",
		},
	}) {
		t.Fatalf("deberia permitir transcript_patch para git_worktree")
	}
	if runtimeOrderSendInstructionPermiteReceiptTranscriptPatch(map[string]any{
		"source": "microprogramacion",
		"microprogramacion": map[string]any{
			"formato_salida": "ficheros+evidencia",
		},
	}) {
		t.Fatalf("no deberia permitir transcript_patch para ficheros+evidencia")
	}
}

func TestUpsertRuntimeHandleDesdeSesionPoolLocalPreservaMetadataRemota(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto",
		Nombre:  "Proyecto",
		RutaAbs: filepath.Join(tmp, "proyecto"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Gemma1",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "proyecto"),
		Herramienta:       "ollama_pool_local",
		ExternalSessionID: "ollama-pool-1",
		ResumePayloadJSON: `{"driver":"ollama_pool_local","transport":"api","endpoint":"http://127.0.0.1:17714","input_path":"/api/runtime/ollama-pool/input","status_path":"/api/runtime/ollama-pool/status","stop_path":"/api/runtime/ollama-pool/stop","mailbox_delivery_mode":"interactive","can_send_input":true,"perfil_ejecucion":{"driver":"ollama_pool_local","transport":"api","endpoint":"http://127.0.0.1:17714","input_path":"/api/runtime/ollama-pool/input","status_path":"/api/runtime/ollama-pool/status","stop_path":"/api/runtime/ollama-pool/stop","mailbox_delivery_mode":"interactive","can_send_input":true}}`,
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get runtime handle: %+v err=%v", handle, err)
	}
	meta := mapFromJSON(handle.MetadataJSON)
	if got := strings.TrimSpace(stringFromMap(meta, "driver", "")); got != "ollama_pool_local" {
		t.Fatalf("driver inesperado: %q meta=%s", got, handle.MetadataJSON)
	}
	if got := strings.TrimSpace(stringFromMap(meta, "transport", "")); got != "api" {
		t.Fatalf("transport inesperado: %q meta=%s", got, handle.MetadataJSON)
	}
	if got := strings.TrimSpace(stringFromMap(meta, "endpoint", "")); got != "http://127.0.0.1:17714" {
		t.Fatalf("endpoint inesperado: %q meta=%s", got, handle.MetadataJSON)
	}
	if got := strings.TrimSpace(stringFromMap(meta, "input_path", "")); got != "/api/runtime/ollama-pool/input" {
		t.Fatalf("input_path inesperado: %q meta=%s", got, handle.MetadataJSON)
	}
	caps := mapFromJSON(handle.CapabilitiesJSON)
	if got := runtimeagente.NormalizeMailboxDeliveryMode(stringFromMap(caps, "mailbox_delivery_mode", "")); got != runtimeagente.MailboxDeliveryInteractive {
		t.Fatalf("mailbox_delivery_mode inesperado: %q caps=%s", got, handle.CapabilitiesJSON)
	}
	if !boolFromMap(caps, "can_send_input") {
		t.Fatalf("capabilities sin can_send_input: %s", handle.CapabilitiesJSON)
	}
}

func TestProcesarRuntimeOrdersBatchMicroprogramacionIgnoraEcoSiBloqueoYCierraSesionOllama(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Qwen1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "demo")
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo",
		Nombre:  "Demo",
		RutaAbs: workingDir,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Qwen1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "ollama-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-micro-pane-patch-stale")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	invocations := filepath.Join(tmp, "tmux-qwen-autostop.log")
	fakeTmux := writeFakeTMUXPanePatchWithKillScriptDB(t, tmp, invocations)
	now := time.Now().UTC()
	deliveredAt := now.Add(-2 * time.Minute)
	staleAt := now.Add(-3 * time.Minute)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_command":          fakeTmux,
		"tmux_session":          "orq-qwen1-pane-stale",
		"tmux_pane_id":          "%91",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
		"can_send_input":        true,
		"rendered_command":      "ollama run qwen2.5-coder:14b",
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "running",
		"alive":                 true,
		"updated_at":            staleAt.Format(time.RFC3339Nano),
		"ready_at":              staleAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": staleAt.Format(time.RFC3339Nano),
		"ready_at":     staleAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_session":"orq-qwen1-pane-stale","tmux_pane_id":"%%91","mailbox_delivery_mode":"%s","can_send_input":true,"rendered_command":"ollama run qwen2.5-coder:14b","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, fakeTmux, runtimeagente.MailboxDeliveryInteractive, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, `{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Qwen1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"source":"microprogramacion"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Qwen1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Qwen1","texto":"PROTOCOLO_ORQUESTA_MICRO SI_BLOQUEO=BLOQUEO: <motivo>","source":"microprogramacion","microprogramacion":{"especificacion_id":1,"archivo_objetivo":"identidad/normalizar.go","simbolo_objetivo":"NormalizarIdentificador","write_set":["identidad/normalizar.go"]},"mailbox_id":%d,"mailbox_kind":"instruction"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	runtimeID := int64(0)
	if handle.RuntimeID != nil {
		runtimeID = *handle.RuntimeID
	}
	if runtimeID <= 0 {
		t.Fatalf("runtimeID invalido en handle: %+v", handle)
	}
	if _, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtimeID,
		HandleID:       &handle.ID,
		Agente:         "Qwen1",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "PROTOCOLO_ORQUESTA_MICRO SI_BLOQUEO=BLOQUEO: <motivo concreto>",
		NormalizedText: "protocolo_orquesta_micro si_bloqueo=bloqueo: motivo concreto",
		CreatedAt:      now,
	}); err != nil {
		t.Fatalf("registrar transcript eco: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar la orden microprogramada con patch en pane y heartbeat stale, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("la orden deberia completarse ignorando el eco SI_BLOQUEO: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"receipt_source":"tmux_pane_patch"`) {
		t.Fatalf("resultado sin receipt_source tmux_pane_patch: %s", sendOrder.ResultadoJSON)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "consumido" {
		t.Fatalf("mailbox deberia quedar consumido tras pane patch: %+v", msg)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle final: %+v err=%v", handle, err)
	}
	if handle.Estado != "cerrado" {
		t.Fatalf("el handle ollama deberia autocerrarse al terminar la microtarea: %+v", handle)
	}
	data, err := os.ReadFile(invocations)
	if err != nil {
		t.Fatalf("leer invocaciones fake tmux: %v", err)
	}
	if !strings.Contains(string(data), "kill-session -t orq-qwen1-pane-stale") {
		t.Fatalf("faltaba kill-session de autocierre ollama: %s", string(data))
	}
}

func TestReconciliarRuntimeOrdersPendientesControlObsoletasCompletaResumeSiElWorkerYaSeRecupero(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		ExternalSessionID: "sess-resume-obsolete",
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

	statusPath := filepath.Join(tmp, "worker-status.json")
	heartbeatPath := filepath.Join(tmp, "worker-heartbeat.json")
	readyAt := time.Now().UTC()
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"alive":      true,
		"ready_at":   readyAt.Format(time.RFC3339Nano),
		"updated_at": readyAt.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"ready_at":     readyAt.Format(time.RFC3339Nano),
		"heartbeat_at": readyAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	meta := map[string]any{
		"driver":                "tmux_cli_session",
		"working_dir":           workingDir,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"external_session_id":   "sess-resume-obsolete",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	}
	metaJSON, _ := json.Marshal(meta)
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='activo', metadata_json=?, capabilities_json=?, updated_at=? WHERE id=?`, string(metaJSON), string(capsJSON), readyAt, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances SET logical_state='esperando_io', updated_at=? WHERE id=?`, readyAt, runtime.ID); err != nil {
		t.Fatalf("update runtime: %v", err)
	}

	resumeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "resume",
		PayloadJSON: `{"accion":"resume","motivo":"test"}`,
	})
	if err != nil {
		t.Fatalf("encolar resume: %v", err)
	}
	old := readyAt.Add(-15 * time.Minute)
	if _, err := DB.Exec(`UPDATE runtime_orders SET created_at=?, available_at=?, updated_at=? WHERE id=?`, old, old, old, resumeID); err != nil {
		t.Fatalf("retroceder resume: %v", err)
	}

	processed, err := reconciliarRuntimeOrdersPendientesControlObsoletas()
	if err != nil {
		t.Fatalf("reconciliar pendientes obsoletas: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed=%d, want 1", processed)
	}
	order, err := GetRuntimeOrder(resumeID)
	if err != nil {
		t.Fatalf("get resume final: %v", err)
	}
	if order == nil || order.Estado != "completada" || !strings.Contains(order.ResultadoJSON, `"obsoleta":true`) {
		t.Fatalf("resume obsoleta inesperada: %+v", order)
	}
}

func TestReconciliarRuntimeOrdersPendientesControlObsoletasCompletaStartSiWorkerYaEstaListo(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		ExternalSessionID: "sess-start-ready",
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

	statusPath := filepath.Join(tmp, "worker-start-status.json")
	heartbeatPath := filepath.Join(tmp, "worker-start-heartbeat.json")
	heartbeatAt := time.Now().UTC()
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"alive":      true,
		"updated_at": heartbeatAt.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": heartbeatAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	meta := map[string]any{
		"driver":                "tmux_cli_session",
		"working_dir":           workingDir,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"external_session_id":   "sess-start-ready",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	}
	metaJSON, _ := json.Marshal(meta)
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='activo', metadata_json=?, capabilities_json=?, updated_at=? WHERE id=?`, string(metaJSON), string(capsJSON), heartbeatAt, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances SET logical_state='esperando_io', updated_at=? WHERE id=?`, heartbeatAt, runtime.ID); err != nil {
		t.Fatalf("update runtime: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "start",
		PayloadJSON: `{"accion":"start","motivo":"test_redundant_start"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}

	processed, err := reconciliarRuntimeOrdersPendientesControlObsoletas()
	if err != nil {
		t.Fatalf("reconciliar pendientes obsoletas: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed=%d, want 1", processed)
	}
	order, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start final: %v", err)
	}
	if order == nil || order.Estado != "completada" {
		t.Fatalf("start redundante inesperada: %+v", order)
	}
	if !strings.Contains(order.ResultadoJSON, `"reason":"runtime_worker_already_running"`) &&
		!strings.Contains(order.ResultadoJSON, `"reason":"runtime_worker_recovered_after_order"`) {
		t.Fatalf("resultado sin reason de worker activo: %s", order.ResultadoJSON)
	}
	runtime, err = GetRuntime(runtime.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime final: %+v err=%v", runtime, err)
	}
	if runtime.LogicalState != "activo" {
		t.Fatalf("el runtime deberia promoverse a activo cuando el worker ya esta running: %+v", runtime)
	}
	if runtime.ProcessState != "running" {
		t.Fatalf("process_state final inesperado: %+v", runtime)
	}
}

func TestReconciliarRuntimeOrdersPendientesControlObsoletasCompletaStopSiYaNoHayRuntimeActivo(t *testing.T) {
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
	stopID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "stop",
		PayloadJSON: `{"accion":"stop","motivo":"test_runtime_absent"}`,
	})
	if err != nil {
		t.Fatalf("encolar stop: %v", err)
	}

	processed, err := reconciliarRuntimeOrdersPendientesControlObsoletas()
	if err != nil {
		t.Fatalf("reconciliar pendientes obsoletas: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed=%d, want 1", processed)
	}
	order, err := GetRuntimeOrder(stopID)
	if err != nil {
		t.Fatalf("get stop final: %v", err)
	}
	if order == nil || order.Estado != "completada" {
		t.Fatalf("stop redundante inesperada: %+v", order)
	}
	if !strings.Contains(order.ResultadoJSON, `"reason":"runtime_already_stopped"`) {
		t.Fatalf("resultado sin runtime_already_stopped: %s", order.ResultadoJSON)
	}
}

func TestRuntimeOrderControlEstadoDeseadoSatisfechoCuentaWorkerRunningSinReadyAt(t *testing.T) {
	tmp := prepararDBTemporal(t)
	statusPath := filepath.Join(tmp, "helper-worker-status.json")
	heartbeatPath := filepath.Join(tmp, "helper-worker-heartbeat.json")
	now := time.Now().UTC()
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"alive":      true,
		"updated_at": now.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	order := &RuntimeOrder{Tipo: "start"}
	handle := &RuntimeHandle{
		Estado:       "activo",
		MetadataJSON: string(metaJSON),
	}
	satisfied, reason := runtimeOrderControlEstadoDeseadoSatisfecho(order, nil, handle)
	if !satisfied {
		t.Fatalf("worker running deberia satisfacer start")
	}
	if reason != "runtime_worker_already_running" {
		t.Fatalf("reason inesperado: %s", reason)
	}
}

func TestRuntimeOrderControlEstadoDeseadoSatisfechoCuentaWorkerStartingComoArranqueEnCurso(t *testing.T) {
	tmp := prepararDBTemporal(t)
	statusPath := filepath.Join(tmp, "helper-worker-status-starting.json")
	heartbeatPath := filepath.Join(tmp, "helper-worker-heartbeat-starting.json")
	now := time.Now().UTC()
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "starting",
		"alive":      true,
		"updated_at": now.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "process_pty_cli",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	order := &RuntimeOrder{Tipo: "start"}
	handle := &RuntimeHandle{
		Estado:       "activo",
		Transporte:   "cli",
		HandleKind:   "process",
		MetadataJSON: string(metaJSON),
	}
	satisfied, reason := runtimeOrderControlEstadoDeseadoSatisfecho(order, nil, handle)
	if !satisfied {
		t.Fatalf("worker starting deberia satisfacer start mientras arranca")
	}
	if reason != "runtime_worker_already_running" {
		t.Fatalf("reason inesperado: %s", reason)
	}
}

func TestRuntimeOrderControlEstadoDeseadoSatisfechoNoCuentaWorkerRunningSiHayStopVivaPrevia(t *testing.T) {
	tmp := prepararDBTemporal(t)
	statusPath := filepath.Join(tmp, "helper-worker-status-running-stop-previa.json")
	heartbeatPath := filepath.Join(tmp, "helper-worker-heartbeat-running-stop-previa.json")
	now := time.Now().UTC()
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"alive":      true,
		"updated_at": now.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
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
	stopID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "stop",
		PayloadJSON: `{"accion":"stop","por":"orquesta","motivo":"restart"}`,
	})
	if err != nil {
		t.Fatalf("encolar stop: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"accion":"start","por":"orquesta","motivo":"restart","proyecto":"orquestador"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	if stopID >= startID {
		t.Fatalf("esperaba stop previa a start, stop=%d start=%d", stopID, startID)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order: %v", err)
	}
	handle := &RuntimeHandle{
		Agente:       "Codex1",
		ProyectoID:   &proyectoID,
		Estado:       "activo",
		Transporte:   "tmux",
		MetadataJSON: string(metaJSON),
	}
	satisfied, reason := runtimeOrderControlEstadoDeseadoSatisfecho(startOrder, nil, handle)
	if satisfied {
		t.Fatalf("start no deberia satisfacerse con worker running si hay stop viva previa: reason=%s", reason)
	}
}

func TestRuntimeObservedLogicalStateFromStructuredWorkerPromueveTMUXReadyARuntimeActivo(t *testing.T) {
	tmp := prepararDBTemporal(t)
	statusPath := filepath.Join(tmp, "worker-observed-status.json")
	heartbeatPath := filepath.Join(tmp, "worker-observed-heartbeat.json")
	now := time.Now().UTC()
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "ready",
		"alive":      true,
		"updated_at": now.Format(time.RFC3339Nano),
		"ready_at":   now.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
		"ready_at":     now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	handle := &RuntimeHandle{
		Estado:       "activo",
		Transporte:   "tmux",
		HandleKind:   "session",
		MetadataJSON: string(metaJSON),
	}
	logicalState, processState, ok := runtimeObservedLogicalStateFromStructuredWorker(handle)
	if !ok {
		t.Fatalf("worker tmux ready deberia promover estado observado")
	}
	if logicalState != "activo" {
		t.Fatalf("logicalState inesperado: %s", logicalState)
	}
	if processState != "running" {
		t.Fatalf("processState inesperado: %s", processState)
	}
}

func TestAplicarEstadoLocalObservadoActualizaHandleEnMemoriaParaPromocionTMUX(t *testing.T) {
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
	workdir := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         workdir,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	statusPath := filepath.Join(tmp, "worker-local-observed-status.json")
	heartbeatPath := filepath.Join(tmp, "worker-local-observed-heartbeat.json")
	now := time.Now().UTC()
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "ready",
		"alive":      true,
		"updated_at": now.Format(time.RFC3339Nano),
		"ready_at":   now.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
		"ready_at":     now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET transporte='tmux',
		    handle_kind='session'
		WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("update handle transport: %v", err)
	}
	handle.Transporte = "tmux"
	handle.HandleKind = "session"

	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	estado := &controlruntime.EstadoLocal{
		Vivo:             true,
		PID:              1234,
		HandleEstado:     "activo",
		LogicalState:     "disponible",
		ProcessState:     "running",
		MetadataJSON:     string(metaJSON),
		CapabilitiesJSON: `{}`,
	}

	if err := aplicarEstadoLocalObservado(handle, estado); err != nil {
		t.Fatalf("aplicarEstadoLocalObservado: %v", err)
	}
	logicalState, processState, ok := runtimeObservedLogicalStateFromStructuredWorker(handle)
	if !ok {
		t.Fatal("deberia observar el worker TMUX desde el handle actualizado en memoria")
	}
	if logicalState != "activo" {
		t.Fatalf("logicalState inesperado: %s", logicalState)
	}
	if processState != "running" {
		t.Fatalf("processState inesperado: %s", processState)
	}
}

func TestRevivirRuntimeHandlePromueveRuntimeTMUXReadyARuntimeActivo(t *testing.T) {
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
	workdir := filepath.Join(tmp, "orquestador")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         workdir,
		Herramienta: "codex-cli",
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

	statusPath := filepath.Join(tmp, "worker-revive-status.json")
	heartbeatPath := filepath.Join(tmp, "worker-revive-heartbeat.json")
	now := time.Now().UTC()
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "ready",
		"alive":      true,
		"updated_at": now.Format(time.RFC3339Nano),
		"ready_at":   now.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
		"ready_at":     now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET transporte='tmux',
		    handle_kind='session',
		    metadata_json=?
		WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	if _, err := DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='esperando_io',
		    process_state='starting'
		WHERE id=?`, runtime.ID); err != nil {
		t.Fatalf("update runtime: %v", err)
	}

	actualizado, err := GetRuntimeHandle(handle.ID)
	if err != nil || actualizado == nil {
		t.Fatalf("get handle actualizado: %+v err=%v", actualizado, err)
	}
	if _, err := revivirRuntimeHandle(actualizado, "activo", nil); err != nil {
		t.Fatalf("revivirRuntimeHandle: %v", err)
	}
	runtime, err = GetRuntime(runtime.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime tras revive: %+v err=%v", runtime, err)
	}
	if runtime.LogicalState != "activo" {
		t.Fatalf("logical_state inesperado: %s", runtime.LogicalState)
	}
	if runtime.ProcessState != "running" {
		t.Fatalf("process_state inesperado: %s", runtime.ProcessState)
	}
}

func TestMarcarRuntimeHandleSupersededDetieneProcesoPTYCliObsoleto(t *testing.T) {
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
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})
	pid := int64(cmd.Process.Pid)
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		ProyectoID:   &proyectoID,
		PID:          &pid,
		LogicalState: "esperando_io",
		ProcessState: "running",
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver": "process_pty_cli",
	})
	handleID := mustInsertID(t, `
		INSERT INTO runtime_handles (agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json)
		VALUES (?,?,?,?,?,?,?,?)`,
		"Codex1", proyectoID, runtimeID, "cli", "process", strconv.FormatInt(pid, 10), "activo", string(metaJSON))
	handle, err := GetRuntimeHandle(handleID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if err := marcarRuntimeHandleSuperseded(handle); err != nil {
		t.Fatalf("supersede handle: %v", err)
	}
	if alive, _, err := controlruntime.ProcesoVivo(controlruntime.ObjetivoProceso{PID: &pid}); err != nil {
		t.Fatalf("ProcesoVivo: %v", err)
	} else if alive {
		procStat, readErr := os.ReadFile(filepath.Join("/proc", strconv.FormatInt(pid, 10), "stat"))
		if readErr != nil {
			if !os.IsNotExist(readErr) {
				t.Fatalf("leer /proc stat: %v", readErr)
			}
		} else {
			raw := string(procStat)
			closing := strings.LastIndex(raw, ")")
			state := ""
			if closing >= 0 && len(raw) > closing+2 {
				state = string(raw[closing+2])
			}
			if state != "Z" {
				t.Fatalf("el proceso obsoleto deberia quedar detenido o zombie mientras el test no hace wait, state=%q", state)
			}
		}
	}
	finalHandle, err := GetRuntimeHandle(handle.ID)
	if err != nil || finalHandle == nil {
		t.Fatalf("get final handle: %+v err=%v", finalHandle, err)
	}
	if finalHandle.Estado != "cerrado" {
		t.Fatalf("estado handle inesperado: %+v", finalHandle)
	}
	runtime, err := GetRuntime(runtimeID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if runtime.LogicalState != "cerrado" || runtime.ProcessState != "finalizado" {
		t.Fatalf("runtime final inesperado: %+v", runtime)
	}
}

func TestRuntimeOrderControlEstadoDeseadoSatisfechoNoConfiaTMUXSinSesion(t *testing.T) {
	tmp := prepararDBTemporal(t)
	statusPath := filepath.Join(tmp, "helper-worker-status-missing-session.json")
	heartbeatPath := filepath.Join(tmp, "helper-worker-heartbeat-missing-session.json")
	fakeTmux := filepath.Join(tmp, "fake-tmux-missing-session")
	now := time.Now().UTC()
	if err := os.WriteFile(fakeTmux, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"alive":      true,
		"updated_at": now.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_command":          fakeTmux,
		"tmux_session":          "orq-missing-session",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	order := &RuntimeOrder{Tipo: "start"}
	handle := &RuntimeHandle{
		Estado:       "activo",
		MetadataJSON: string(metaJSON),
	}
	satisfied, reason := runtimeOrderControlEstadoDeseadoSatisfecho(order, nil, handle)
	if satisfied {
		t.Fatalf("worker tmux sin sesion no deberia satisfacer start: %s", reason)
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
	if agente.PresupuestoSesionPct != nil {
		t.Fatalf("provider_backoff agotado no deberia conservar porcentaje de sesion derivado: %+v", agente)
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

func TestEjecutarRuntimeOrderSendInstructionBackoffCompletaSiMailboxYaFueConsumido(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert handle: %v", err)
	}
	handle, err := GetRuntimeHandleActivoAgenteProyecto("Codex1", &proyectoID)
	if err != nil || handle == nil {
		t.Fatalf("get handle activo: %v %+v", err, handle)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`,
		`{"rendered_command":"'/tmp/codex-perfil' 'Codex1'","working_dir":"`+filepath.Join(tmp, "orquestador")+`","can_send_input":false}`,
		`{"can_send_input":false,"mailbox_delivery_mode":"session_resume"}`,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		ProyectoID:   &proyectoID,
		Provider:     "openai",
		Connector:    "codex-cli",
		LogicalState: "activo",
		ProcessState: "running",
		SesionID:     &sesion.ID,
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET runtime_id=? WHERE id=?`, runtimeID, handle.ID); err != nil {
		t.Fatalf("attach runtime: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "governance_refresh",
		PayloadJSON: `{"hash":"canon"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxConsumido(mailboxID); err != nil {
		t.Fatalf("consumir mailbox: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtimeID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"refresh","mailbox_id":%d,"mailbox_kind":"governance_refresh","external_session_id":"019d43a6-fbab-7323-b845-35a5d4267d18"}`, mailboxID),
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
		t.Fatalf("la orden deberia completarse si el mailbox ya fue consumido: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_only":true`) {
		t.Fatalf("resultado sin mailbox_only: %s", sendOrder.ResultadoJSON)
	}
	if strings.Contains(sendOrder.ResultadoJSON, `"retry_after"`) {
		t.Fatalf("no deberia reencolar retry si mailbox ya fue consumido: %s", sendOrder.ResultadoJSON)
	}
}

func TestProcesarRuntimeOrdersBatchReconciliaPendienteSiMailboxYaFueConsumido(t *testing.T) {
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
	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "governance_refresh",
		PayloadJSON: `{"hash":"canon"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxConsumido(mailboxID); err != nil {
		t.Fatalf("consumir mailbox: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"refresh","mailbox_id":%d,"mailbox_kind":"governance_refresh","external_session_id":"stale"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET available_at = ? WHERE id = ?`, time.Now().UTC().Add(10*time.Minute), sendID); err != nil {
		t.Fatalf("posponer orden: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar al menos una orden, obtuvo %d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "completada" {
		t.Fatalf("la orden deberia completarse al reconciliar mailbox consumido: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"mailbox_only":true`) {
		t.Fatalf("resultado sin mailbox_only: %s", sendOrder.ResultadoJSON)
	}
}

func TestProcesarRuntimeOrdersBatchRearMaMailboxFaltantePendiente(t *testing.T) {
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
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"refresh","mailbox_id":999999,"mailbox_kind":"governance_refresh"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET available_at = ? WHERE id = ?`, time.Now().UTC().Add(10*time.Minute), sendID); err != nil {
		t.Fatalf("posponer orden: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar al menos una orden, obtuvo %d", processed)
	}

	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("la orden deberia seguir pendiente tras rearmar la mailbox: %+v", sendOrder)
	}
	payload := mapFromJSON(sendOrder.PayloadJSON)
	mailboxID := runtimeOrderSendInstructionMailboxID(payload)
	if mailboxID <= 0 || mailboxID == 999999 {
		t.Fatalf("mailbox_id no rearmado en payload: %s", sendOrder.PayloadJSON)
	}
	if !strings.Contains(sendOrder.ErrorText, "mailbox_missing_rearmed") {
		t.Fatalf("error_text sin motivo de mailbox rearmada: %s", sendOrder.ErrorText)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "pendiente" {
		t.Fatalf("mailbox rearmada deberia quedar pendiente: %+v", msg)
	}
	if msg.RuntimeOrderID == nil || *msg.RuntimeOrderID != sendID {
		t.Fatalf("mailbox rearmada sin runtime_order_id correcto: %+v", msg)
	}
}

func TestProcesarRuntimeOrdersBatchMantienePendienteSiMailboxYaFueEntregado(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		ExternalSessionID: "sess-bootstrap-pending",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-bootstrap-pending")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	deliveredAt := now.Add(-15 * time.Second)
	staleOutputAt := deliveredAt.Add(-5 * time.Second)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex1-pending",
		"tmux_pane_id":          "%66",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
		"can_send_input":        false,
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "running",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"ready_at":              deliveredAt.Add(-10 * time.Second).Format(time.RFC3339Nano),
		"last_output_at":        staleOutputAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":          true,
		"heartbeat_at":   now.Format(time.RFC3339Nano),
		"ready_at":       deliveredAt.Add(-10 * time.Second).Format(time.RFC3339Nano),
		"last_output_at": staleOutputAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-codex1-pending","tmux_pane_id":"%%66","mailbox_delivery_mode":"%s","can_send_input":false,"worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, runtimeagente.MailboxDeliveryBootstrapOnly, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "governance_refresh",
		PayloadJSON: `{"hash":"canon"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"refresh","mailbox_id":%d,"mailbox_kind":"governance_refresh","external_session_id":"stale"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET available_at = ? WHERE id = ?`, time.Now().UTC().Add(10*time.Minute), sendID); err != nil {
		t.Fatalf("posponer orden: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar al menos una orden, obtuvo %d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("la orden deberia seguir pendiente si el mailbox solo fue entregado: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"delivery_state":"notified"`) {
		t.Fatalf("resultado sin delivery_state notified: %s", sendOrder.ResultadoJSON)
	}
}

func TestRetenerRuntimeOrderSendInstructionDiferidaAMailboxRecreaMailboxTerminal(t *testing.T) {
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
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"refresh","mailbox_kind":"governance_refresh"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	runtimeOrderID := sendID
	oldMailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:     "server",
		ToAgente:       "Codex1",
		ProyectoID:     &proyectoID,
		RuntimeOrderID: &runtimeOrderID,
		Kind:           "governance_refresh",
		PayloadJSON:    `{"to_agente":"Codex1","texto":"refresh","mailbox_kind":"governance_refresh"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox vieja: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(oldMailboxID); err != nil {
		t.Fatalf("entregar mailbox vieja: %v", err)
	}
	if err := MarcarRuntimeMailboxConsumido(oldMailboxID); err != nil {
		t.Fatalf("consumir mailbox vieja: %v", err)
	}

	order, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	payload := map[string]any{
		"to_agente":    "Codex1",
		"texto":        "refresh",
		"mailbox_id":   oldMailboxID,
		"mailbox_kind": "governance_refresh",
	}
	if err := retenerRuntimeOrderSendInstructionDiferidaAMailbox(order, payload, "sin runtime activo entregable"); err != nil {
		t.Fatalf("retener order: %v", err)
	}

	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	updatedPayload := mapFromJSON(sendOrder.PayloadJSON)
	newMailboxID := runtimeOrderSendInstructionMailboxID(updatedPayload)
	if newMailboxID <= 0 || newMailboxID == oldMailboxID {
		t.Fatalf("no se recreo mailbox nueva: old=%d payload=%s", oldMailboxID, sendOrder.PayloadJSON)
	}
	oldMsg, err := GetRuntimeMailbox(oldMailboxID)
	if err != nil || oldMsg == nil {
		t.Fatalf("get old mailbox: %+v err=%v", oldMsg, err)
	}
	if oldMsg.Estado != "consumido" {
		t.Fatalf("mailbox vieja no deberia tocarse: %+v", oldMsg)
	}
	newMsg, err := GetRuntimeMailbox(newMailboxID)
	if err != nil || newMsg == nil {
		t.Fatalf("get new mailbox: %+v err=%v", newMsg, err)
	}
	if newMsg.Estado != "pendiente" {
		t.Fatalf("mailbox nueva deberia quedar pendiente: %+v", newMsg)
	}
	if newMsg.RuntimeOrderID == nil || *newMsg.RuntimeOrderID != sendID {
		t.Fatalf("mailbox nueva sin runtime_order_id correcto: %+v", newMsg)
	}
}

func TestProcesarRuntimeOrdersBatchReencolaMailboxEntregadoSinReciboTrasTimeout(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		ExternalSessionID: "sess-bootstrap-timeout",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-bootstrap-timeout")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	deliveredAt := now.Add(-3 * time.Minute)
	staleOutputAt := deliveredAt.Add(-30 * time.Second)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex1-timeout",
		"tmux_pane_id":          "%88",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
		"can_send_input":        false,
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "running",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"ready_at":              deliveredAt.Format(time.RFC3339Nano),
		"last_output_at":        staleOutputAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":          true,
		"heartbeat_at":   now.Format(time.RFC3339Nano),
		"ready_at":       deliveredAt.Format(time.RFC3339Nano),
		"last_output_at": staleOutputAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-codex1-timeout","tmux_pane_id":"%%88","mailbox_delivery_mode":"%s","can_send_input":false,"worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, runtimeagente.MailboxDeliveryBootstrapOnly, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "governance_refresh",
		PayloadJSON: `{"hash":"canon"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"continua trabajo actual","mailbox_id":%d,"mailbox_kind":"governance_refresh"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar la orden entregada stale, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("la orden deberia reencolarse pendiente tras timeout de recibo: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"delivery_state":"queued"`) {
		t.Fatalf("resultado sin delivery_state queued: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ErrorText, "delivery receipt timeout") {
		t.Fatalf("error_text sin timeout de recibo: %s", sendOrder.ErrorText)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "pendiente" {
		t.Fatalf("mailbox deberia rearmarse a pendiente: %+v", msg)
	}
}

func TestProcesarRuntimeOrdersBatchOllamaTimeoutReciboCierraSesionAntesDeReencolar(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Qwen1", "programador"); err != nil {
		t.Fatalf("registrar Qwen1: %v", err)
	}
	workingDir := filepath.Join(tmp, "demo")
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo",
		Nombre:  "Demo",
		RutaAbs: workingDir,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Qwen1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "ollama-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-qwen-timeout")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	invocations := filepath.Join(tmp, "tmux-qwen-timeout.log")
	fakeTmux := writeFakeTMUXNoPatchWithKillScriptDB(t, tmp, invocations)
	now := time.Now().UTC()
	deliveredAt := now.Add(-3 * time.Minute)
	staleOutputAt := now.Add(-4 * time.Minute)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_command":          fakeTmux,
		"tmux_session":          "orq-qwen1-timeout",
		"tmux_pane_id":          "%92",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
		"can_send_input":        true,
		"rendered_command":      "ollama run qwen2.5-coder:14b",
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "running",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"ready_at":              deliveredAt.Format(time.RFC3339Nano),
		"last_output_at":        staleOutputAt.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryInteractive,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":          true,
		"heartbeat_at":   now.Format(time.RFC3339Nano),
		"ready_at":       deliveredAt.Format(time.RFC3339Nano),
		"last_output_at": staleOutputAt.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":"%s","tmux_session":"orq-qwen1-timeout","tmux_pane_id":"%%92","mailbox_delivery_mode":"%s","can_send_input":true,"rendered_command":"ollama run qwen2.5-coder:14b","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, fakeTmux, runtimeagente.MailboxDeliveryInteractive, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, `{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Qwen1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"source":"microprogramacion"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Qwen1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Qwen1","texto":"micro","source":"microprogramacion","microprogramacion":{"especificacion_id":1,"archivo_objetivo":"identidad/normalizar.go","simbolo_objetivo":"NormalizarIdentificador","write_set":["identidad/normalizar.go"]},"mailbox_id":%d,"mailbox_kind":"instruction"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar la orden entregada stale de ollama, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("la orden ollama deberia reencolarse pendiente tras timeout de recibo: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ErrorText, "delivery receipt timeout") {
		t.Fatalf("error_text sin timeout de recibo: %s", sendOrder.ErrorText)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "pendiente" {
		t.Fatalf("mailbox deberia rearmarse a pendiente: %+v", msg)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle final: %+v err=%v", handle, err)
	}
	if handle.Estado != "cerrado" {
		t.Fatalf("el handle ollama deberia cerrarse antes del reintento: %+v", handle)
	}
	data, err := os.ReadFile(invocations)
	if err != nil {
		t.Fatalf("leer invocaciones fake tmux: %v", err)
	}
	if !strings.Contains(string(data), "kill-session -t orq-qwen1-timeout") {
		t.Fatalf("faltaba kill-session tras timeout ollama: %s", string(data))
	}
}

func TestProcesarRuntimeOrdersBatchReencolaMailboxEntregadoSiWorkerYaEstaParado(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		ExternalSessionID: "sess-bootstrap-stopped",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-bootstrap-stopped")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	deliveredAt := now.Add(-15 * time.Second)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex1-stopped",
		"tmux_pane_id":          "%99",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
		"can_send_input":        false,
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "stopped",
		"alive":                 false,
		"updated_at":            now.Format(time.RFC3339Nano),
		"ready_at":              deliveredAt.Add(-10 * time.Second).Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
		"exit_error":            "tmux pane finalizado",
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        false,
		"heartbeat_at": now.Format(time.RFC3339Nano),
		"ready_at":     deliveredAt.Add(-10 * time.Second).Format(time.RFC3339Nano),
		"exit_error":   "tmux pane finalizado",
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-codex1-stopped","tmux_pane_id":"%%99","mailbox_delivery_mode":"%s","can_send_input":false,"worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, runtimeagente.MailboxDeliveryBootstrapOnly, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=?, estado='fallido' WHERE id=?`, metaJSON, `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "governance_refresh",
		PayloadJSON: `{"hash":"canon"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"continua trabajo actual","mailbox_id":%d,"mailbox_kind":"governance_refresh"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar la orden entregada con worker parado, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("la orden deberia reencolarse pendiente cuando el worker ya esta parado: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"delivery_state":"queued"`) {
		t.Fatalf("resultado sin delivery_state queued: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ErrorText, "worker_not_alive_after_notified_dispatch") {
		t.Fatalf("error_text sin motivo de worker parado: %s", sendOrder.ErrorText)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "pendiente" {
		t.Fatalf("mailbox deberia rearmarse a pendiente cuando el worker ya esta parado: %+v", msg)
	}
}

func TestProcesarRuntimeOrdersBatchToleraRuntimeIDStaleEnOrdenEntregada(t *testing.T) {
	tmp := prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	workingDir := filepath.Join(tmp, "orquestador")
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
		ExternalSessionID: "sess-bootstrap-stopped-stale-runtime",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-bootstrap-stale-runtime")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir workerDir: %v", err)
	}
	manifestPath := filepath.Join(workerDir, "manifest.json")
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	deliveredAt := now.Add(-15 * time.Second)
	manifestRaw, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex1-stopped-stale-runtime",
		"tmux_pane_id":          "%101",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
		"can_send_input":        false,
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "stopped",
		"alive":                 false,
		"updated_at":            now.Format(time.RFC3339Nano),
		"ready_at":              deliveredAt.Add(-10 * time.Second).Format(time.RFC3339Nano),
		"mailbox_delivery_mode": runtimeagente.MailboxDeliveryBootstrapOnly,
		"exit_error":            "tmux pane finalizado",
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        false,
		"heartbeat_at": now.Format(time.RFC3339Nano),
		"ready_at":     deliveredAt.Add(-10 * time.Second).Format(time.RFC3339Nano),
		"exit_error":   "tmux pane finalizado",
	})
	if err := os.WriteFile(manifestPath, append(manifestRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-codex1-stopped-stale-runtime","tmux_pane_id":"%%101","mailbox_delivery_mode":"%s","can_send_input":false,"worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, runtimeagente.MailboxDeliveryBootstrapOnly, manifestPath, statusPath, heartbeatPath)
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, capabilities_json=?, estado='fallido' WHERE id=?`, metaJSON, `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "governance_refresh",
		PayloadJSON: `{"hash":"canon"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_mailbox SET delivered_at=? WHERE id=?`, deliveredAt, mailboxID); err != nil {
		t.Fatalf("backdate delivered_at: %v", err)
	}
	staleRuntimeID := int64(999999)
	sendID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &staleRuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","texto":"continua trabajo actual","mailbox_id":%d,"mailbox_kind":"governance_refresh"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders con runtime_id stale: %v", err)
	}
	if processed < 1 {
		t.Fatalf("se esperaba reconciliar la orden entregada con runtime_id stale, got=%d", processed)
	}
	sendOrder, err := GetRuntimeOrder(sendID)
	if err != nil {
		t.Fatalf("get send order final: %v", err)
	}
	if sendOrder.Estado != "pendiente" {
		t.Fatalf("la orden deberia reencolarse pendiente con runtime_id stale: %+v", sendOrder)
	}
	if !strings.Contains(sendOrder.ResultadoJSON, `"delivery_state":"queued"`) {
		t.Fatalf("resultado sin delivery_state queued: %s", sendOrder.ResultadoJSON)
	}
	if !strings.Contains(sendOrder.ErrorText, "worker_not_alive_after_notified_dispatch") {
		t.Fatalf("error_text sin motivo de worker parado: %s", sendOrder.ErrorText)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "pendiente" {
		t.Fatalf("mailbox deberia rearmarse a pendiente con runtime_id stale: %+v", msg)
	}
}

func TestRuntimeOrderControlObsoletaPorWorkerRecuperadoNoCuentaHeartbeatViejoTMUX(t *testing.T) {
	tmp := prepararDBTemporal(t)
	baseline := time.Now().UTC()
	runDir := filepath.Join(tmp, "runtime")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir runDir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	startedAt := baseline.Add(-30 * time.Minute).Format(time.RFC3339Nano)
	readyAt := baseline.Add(-29 * time.Minute).Format(time.RFC3339Nano)
	heartbeatAt := baseline.Add(30 * time.Second).Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex2","driver":"tmux_cli_session","transport":"tmux","started_at":"`+startedAt+`","tmux_session":"orq-codex2","tmux_pane_id":"%1"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+heartbeatAt+`","alive":true,"ready_at":"`+readyAt+`","last_output_at":"`+readyAt+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+heartbeatAt+`","started_at":"`+startedAt+`","ready_at":"`+readyAt+`","last_output_at":"`+readyAt+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","rendered_command":"codex-perfil Codex2","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, manifestPath, statusPath, heartbeatPath)
	order := &RuntimeOrder{Tipo: "start", CreatedAt: baseline}
	handle := &RuntimeHandle{Estado: "activo", MetadataJSON: metaJSON}
	if runtimeOrderControlObsoletaPorWorkerRecuperado(order, nil, handle) {
		t.Fatalf("no deberia considerar recuperado un worker viejo solo por heartbeat fresco")
	}
}

func TestRuntimeOrderControlObsoletaPorWorkerRecuperadoCuentaReadyNuevoTMUX(t *testing.T) {
	tmp := prepararDBTemporal(t)
	baseline := time.Now().UTC()
	runDir := filepath.Join(tmp, "runtime")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir runDir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	startedAt := baseline.Add(5 * time.Second).Format(time.RFC3339Nano)
	readyAt := baseline.Add(10 * time.Second).Format(time.RFC3339Nano)
	heartbeatAt := baseline.Add(15 * time.Second).Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex2","driver":"tmux_cli_session","transport":"tmux","started_at":"`+startedAt+`","tmux_session":"orq-codex2","tmux_pane_id":"%1"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"ready","updated_at":"`+heartbeatAt+`","alive":true,"ready_at":"`+readyAt+`","last_output_at":"`+readyAt+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+heartbeatAt+`","started_at":"`+startedAt+`","ready_at":"`+readyAt+`","last_output_at":"`+readyAt+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","rendered_command":"codex-perfil Codex2","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, manifestPath, statusPath, heartbeatPath)
	order := &RuntimeOrder{Tipo: "start", CreatedAt: baseline}
	handle := &RuntimeHandle{Estado: "activo", MetadataJSON: metaJSON}
	if !runtimeOrderControlObsoletaPorWorkerRecuperado(order, nil, handle) {
		t.Fatalf("deberia considerar recuperado un worker nuevo listo tras la orden")
	}
}

func TestAplicarEstadoLocalObservadoCompactaPendingTranscript(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert handle: %v", err)
	}
	handle, err := GetRuntimeHandleActivoAgenteProyecto("Codex1", &proyectoID)
	if err != nil || handle == nil {
		t.Fatalf("get handle activo: %v %+v", err, handle)
	}

	estado := &controlruntime.EstadoLocal{
		Vivo:         true,
		HandleEstado: "activo",
		MetadataJSON: `{"transcript_log_pending":"\u001b]11;?\u0007\u001b[;m hola \u001b[0m   mundo \u0000","working_dir":"` + filepath.Join(tmp, "orquestador") + `"}`,
	}
	if err := aplicarEstadoLocalObservado(handle, estado); err != nil {
		t.Fatalf("aplicar estado local: %v", err)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil {
		t.Fatalf("recargar handle: %v", err)
	}
	meta := mapFromJSON(handle.MetadataJSON)
	got := strings.TrimSpace(stringFromMap(meta, "transcript_log_pending", ""))
	if got != "hola    mundo" {
		t.Fatalf("pending compactado inesperado: %q", got)
	}
}

func TestCompactarMetadataRuntimeHandleResumeSummary(t *testing.T) {
	largo := "Linea 1 de continuidad muy larga.\nLinea 2 con detalle adicional que no debe vivir cruda en metadata del handle."
	meta := map[string]any{
		"resumen_continuidad": largo,
	}
	compactarMetadataRuntimeHandle(meta)
	raw, _ := json.Marshal(meta)
	if strings.Contains(string(raw), `"resumen_continuidad":"`) {
		t.Fatalf("metadata no deberia conservar resumen_continuidad crudo: %s", string(raw))
	}
	if !strings.Contains(string(raw), `"resumen_continuidad_summary":"Linea 1 de continuidad muy larga."`) {
		t.Fatalf("faltó resumen resumido: %s", string(raw))
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

func TestRuntimeOrderPauseTMUXBootstrapOnlySeSatisfaceCerrandoSesion(t *testing.T) {
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
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	invocations := filepath.Join(tmp, "tmux-pause.log")
	fakeTmux := filepath.Join(tmp, "tmux")
	script := "#!/usr/bin/env bash\n" +
		"printf '%s\\n' \"$*\" >> " + strconv.Quote(invocations) + "\n" +
		"exit 0\n"
	if err := os.WriteFile(fakeTmux, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}

	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":%q,"tmux_session":"orq-codex1-bootstrap","mailbox_delivery_mode":"bootstrap_only","can_send_input":false}`, fakeTmux)
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`
	if _, err := DB.Exec(`UPDATE runtime_handles SET transporte='cli', handle_kind='process', handle_ref=?, metadata_json=?, capabilities_json=? WHERE id=?`, "12345", metaJSON, capsJSON, handle.ID); err != nil {
		t.Fatalf("update handle tmux bootstrap: %v", err)
	}

	pauseID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "pause",
		PayloadJSON: `{"motivo":"quota cooldown"}`,
	})
	if err != nil {
		t.Fatalf("encolar pause: %v", err)
	}
	pauseOrder, err := GetRuntimeOrder(pauseID)
	if err != nil || pauseOrder == nil {
		t.Fatalf("get pause order: %+v err=%v", pauseOrder, err)
	}
	if err := ejecutarRuntimeOrderPause(pauseOrder); err != nil {
		t.Fatalf("ejecutar pause tmux bootstrap_only: %v", err)
	}

	pauseOrder, err = GetRuntimeOrder(pauseID)
	if err != nil || pauseOrder == nil {
		t.Fatalf("get pause order final: %+v err=%v", pauseOrder, err)
	}
	if pauseOrder.Estado != "completada" || !strings.Contains(pauseOrder.ResultadoJSON, `"paused_via_stop":true`) {
		t.Fatalf("pause tmux bootstrap_only deberia completarse via stop: %+v", pauseOrder)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle final: %+v err=%v", handle, err)
	}
	if handle.Estado != "cerrado" {
		t.Fatalf("el handle bootstrap_only pausado via stop deberia quedar cerrado: %+v", handle)
	}
	sesion, err = GetSesionActiva("Codex1", &proyectoID)
	if err != nil || sesion == nil {
		t.Fatalf("sesion tras pause tmux: %+v err=%v", sesion, err)
	}
	if sesion.Estado != "pausada" {
		t.Fatalf("sesion tras pause tmux inesperada: %+v", sesion)
	}

	data, err := os.ReadFile(invocations)
	if err != nil {
		t.Fatalf("read fake tmux log: %v", err)
	}
	if !strings.Contains(string(data), "kill-session -t orq-codex1-bootstrap") {
		t.Fatalf("faltaba kill-session para pause tmux bootstrap_only: %s", string(data))
	}
}

func TestRuntimeOrderPauseTMUXOllamaInteractiveSeSatisfaceCerrandoSesion(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Ollama1", "programador"); err != nil {
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
		Agente:      "Ollama1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "ollama-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	invocations := filepath.Join(tmp, "tmux-pause-ollama.log")
	fakeTmux := filepath.Join(tmp, "tmux-ollama")
	script := "#!/usr/bin/env bash\n" +
		"printf '%s\\n' \"$*\" >> " + strconv.Quote(invocations) + "\n" +
		"exit 0\n"
	if err := os.WriteFile(fakeTmux, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}

	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_command":%q,"tmux_session":"orq-ollama1-interactive","mailbox_delivery_mode":"interactive","can_send_input":true,"rendered_command":"ollama run qwen2.5-coder:14b"}`, fakeTmux)
	capsJSON := `{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`
	if _, err := DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref=?, metadata_json=?, capabilities_json=? WHERE id=?`, "orq-ollama1-interactive/%91", metaJSON, capsJSON, handle.ID); err != nil {
		t.Fatalf("update handle tmux ollama: %v", err)
	}

	pauseID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Ollama1",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "pause",
		PayloadJSON: `{"motivo":"sin trabajo"}`,
	})
	if err != nil {
		t.Fatalf("encolar pause: %v", err)
	}
	pauseOrder, err := GetRuntimeOrder(pauseID)
	if err != nil || pauseOrder == nil {
		t.Fatalf("get pause order: %+v err=%v", pauseOrder, err)
	}
	if err := ejecutarRuntimeOrderPause(pauseOrder); err != nil {
		t.Fatalf("ejecutar pause ollama interactive: %v", err)
	}

	pauseOrder, err = GetRuntimeOrder(pauseID)
	if err != nil || pauseOrder == nil {
		t.Fatalf("get pause order final: %+v err=%v", pauseOrder, err)
	}
	if pauseOrder.Estado != "completada" || !strings.Contains(pauseOrder.ResultadoJSON, `"paused_via_stop":true`) {
		t.Fatalf("pause ollama interactive deberia completarse via stop: %+v", pauseOrder)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle final: %+v err=%v", handle, err)
	}
	if handle.Estado != "cerrado" {
		t.Fatalf("el handle ollama pausado via stop deberia quedar cerrado: %+v", handle)
	}

	data, err := os.ReadFile(invocations)
	if err != nil {
		t.Fatalf("read fake tmux log: %v", err)
	}
	if !strings.Contains(string(data), "kill-session -t orq-ollama1-interactive") {
		t.Fatalf("faltaba kill-session para pause ollama interactive: %s", string(data))
	}
}

func TestRuntimeOrderPauseTMUXBootstrapOnlyYaMuertoQuedaCompletada(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		PID:         &pidMuerto,
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"tmux_cli_session","tmux_session":"orq-codex1-dead","mailbox_delivery_mode":"bootstrap_only","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`
	if _, err := DB.Exec(`UPDATE runtime_handles SET transporte='cli', handle_kind='process', handle_ref=?, metadata_json=?, capabilities_json=? WHERE id=?`, strconv.FormatInt(pidMuerto, 10), metaJSON, capsJSON, handle.ID); err != nil {
		t.Fatalf("update handle tmux bootstrap: %v", err)
	}

	pauseID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "pause",
		PayloadJSON: `{"motivo":"quota cooldown"}`,
	})
	if err != nil {
		t.Fatalf("encolar pause: %v", err)
	}
	pauseOrder, err := GetRuntimeOrder(pauseID)
	if err != nil || pauseOrder == nil {
		t.Fatalf("get pause order: %+v err=%v", pauseOrder, err)
	}
	if err := ejecutarRuntimeOrderPause(pauseOrder); err != nil {
		t.Fatalf("ejecutar pause tmux bootstrap_only ya muerto: %v", err)
	}

	pauseOrder, err = GetRuntimeOrder(pauseID)
	if err != nil || pauseOrder == nil {
		t.Fatalf("get pause order final: %+v err=%v", pauseOrder, err)
	}
	if pauseOrder.Estado != "completada" || !strings.Contains(pauseOrder.ResultadoJSON, `"already_stopped":true`) {
		t.Fatalf("pause tmux bootstrap_only ya muerto deberia completarse como already_stopped: %+v", pauseOrder)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle final: %+v err=%v", handle, err)
	}
	if handle.Estado != "cerrado" {
		t.Fatalf("el handle bootstrap_only ya muerto deberia quedar cerrado: %+v", handle)
	}
}

func TestResolverHandleParaOrdenConservaHandleExplicitoParaPauseAunqueNoSeaActivoCanonico(t *testing.T) {
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
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"tmux_cli_session","tmux_session":"orq-codex1-stale","mailbox_delivery_mode":"bootstrap_only","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='fallido', transporte='cli', handle_kind='process', handle_ref='999997', metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, capsJSON, handle.ID); err != nil {
		t.Fatalf("update handle tmux bootstrap stale: %v", err)
	}

	order := &RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		HandleID:   &handle.ID,
		Tipo:       "pause",
	}
	resolved, err := resolverHandleParaOrden(order)
	if err != nil {
		t.Fatalf("resolverHandleParaOrden: %v", err)
	}
	if resolved == nil || resolved.ID != handle.ID {
		t.Fatalf("deberia conservar el handle explicito stale para pause: %+v", resolved)
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

func TestRuntimeHandlePreservesExternalSessionNoImplicaPreservarSoloPorExternalSessionID(t *testing.T) {
	handle := &RuntimeHandle{
		MetadataJSON:     `{"external_session_id":"sess-remote","driver":"tmux_cli_session"}`,
		CapabilitiesJSON: `{"can_stop":true}`,
	}
	if runtimeHandlePreservesExternalSession(handle) {
		t.Fatalf("no deberia preservar solo por external_session_id cuando no hay señal explicita")
	}
}

func TestRuntimeHandlePreservesExternalSessionRespetaCanStopWithoutReauthFalse(t *testing.T) {
	handle := &RuntimeHandle{
		MetadataJSON:     `{"external_session_id":"sess-remote","driver":"tmux_cli_session"}`,
		CapabilitiesJSON: `{"can_stop":true,"can_stop_without_reauth":false}`,
	}
	if !runtimeHandlePreservesExternalSession(handle) {
		t.Fatalf("deberia preservar cuando can_stop_without_reauth=false")
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
		t.Fatalf("get handoff tras delivered: %v", err)
	}
	if handoffOrder.Estado != "ejecutando" {
		t.Fatalf("handoff no deberia completarse sin progreso util: %+v", handoffOrder)
	}
	if !strings.Contains(handoffOrder.ResultadoJSON, `"lease_state":"delivered"`) {
		t.Fatalf("handoff deberia quedar entregada pero no consumida: %s", handoffOrder.ResultadoJSON)
	}
	tarea, err = GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea tras delivered: %v", err)
	}
	if tarea.Estado != TareaAsignada {
		t.Fatalf("la tarea no deberia pasar a progreso solo con delivered: %s", tarea.Estado)
	}
	msgsPendientes, err := ListarRuntimeMailbox(FiltroRuntimeMailbox{
		ToAgente:   strPtrTest("Codex1"),
		ProyectoID: &proyectoID,
		Estado:     strPtrTest("pendiente"),
	})
	if err != nil {
		t.Fatalf("mailbox pendiente tras delivered: %v", err)
	}
	if len(msgsPendientes) != 1 || msgsPendientes[0].ID != msgID {
		t.Fatalf("mailbox pendiente inesperado tras delivered: %+v", msgsPendientes)
	}

	meta := mapFromJSON(handle.MetadataJSON)
	statusPath := strings.TrimSpace(stringFromMap(meta, "worker_status_path", ""))
	heartbeatPath := strings.TrimSpace(stringFromMap(meta, "worker_heartbeat_path", ""))
	if statusPath == "" || heartbeatPath == "" {
		t.Fatalf("faltan rutas de worker en metadata: %+v", meta)
	}
	ackAt := time.Now().UTC().Add(2 * time.Second).Format(time.RFC3339Nano)
	statusData := map[string]any{}
	if raw, err := os.ReadFile(statusPath); err != nil {
		t.Fatalf("read worker status: %v", err)
	} else if err := json.Unmarshal(raw, &statusData); err != nil {
		t.Fatalf("decode worker status: %v", err)
	}
	statusData["state"] = "running"
	statusData["alive"] = true
	statusData["ready_at"] = ackAt
	statusData["last_progress_at"] = ackAt
	statusData["last_output_at"] = ackAt
	if raw, err := json.Marshal(statusData); err != nil {
		t.Fatalf("marshal worker status: %v", err)
	} else if err := os.WriteFile(statusPath, append(raw, '\n'), 0o600); err != nil {
		t.Fatalf("write worker status: %v", err)
	}
	heartbeatData := map[string]any{}
	if raw, err := os.ReadFile(heartbeatPath); err != nil {
		t.Fatalf("read worker heartbeat: %v", err)
	} else if err := json.Unmarshal(raw, &heartbeatData); err != nil {
		t.Fatalf("decode worker heartbeat: %v", err)
	}
	heartbeatData["alive"] = true
	heartbeatData["ready_at"] = ackAt
	heartbeatData["last_progress_at"] = ackAt
	heartbeatData["last_output_at"] = ackAt
	heartbeatData["heartbeat_at"] = ackAt
	if raw, err := json.Marshal(heartbeatData); err != nil {
		t.Fatalf("marshal worker heartbeat: %v", err)
	} else if err := os.WriteFile(heartbeatPath, append(raw, '\n'), 0o600); err != nil {
		t.Fatalf("write worker heartbeat: %v", err)
	}
	if err := AckBootstrapRuntimeLeaseByEvidence(handle, runtime, "test_runtime_output"); err != nil {
		t.Fatalf("ack bootstrap final por evidencia: %v", err)
	}
	handoffOrder, err = GetRuntimeOrder(handoffID)
	if err != nil {
		t.Fatalf("get handoff tras ack final: %v", err)
	}
	if handoffOrder.Estado != "completada" {
		t.Fatalf("handoff no completada tras progreso util: %+v", handoffOrder)
	}
	tarea, err = GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea tras ack final: %v", err)
	}
	if tarea.Estado != TareaEnProgreso {
		t.Fatalf("la tarea deberia quedar en progreso tras el ack real: %s", tarea.Estado)
	}
	if !strings.Contains(tarea.Notas, "handoff completado por Codex1") {
		t.Fatalf("notas sin evidencia de handoff completado tras ack real: %s", tarea.Notas)
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

func TestRegistrarEvidenciaHandoffReanudadoPorOrdenEmiteNotificacion(t *testing.T) {
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
	tareaID, err := CrearTarea(&Tarea{
		Titulo:     "Continuar handoff",
		ProyectoID: &proyectoID,
		Prioridad:  PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if _, err := DB.Exec(`UPDATE tareas SET agente='Codex1', estado='asignada' WHERE id=?`, tareaID); err != nil {
		t.Fatalf("preparar tarea asignada: %v", err)
	}

	payloadJSON, err := json.Marshal(HandoffPayload{
		AgenteOrigen:       "Codex0",
		AgenteDestino:      "Codex1",
		TareaID:            &tareaID,
		ResumenContinuidad: "handoff listo",
		ExternalSessionID:  "sess-handoff-unit",
	})
	if err != nil {
		t.Fatalf("marshal handoff: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "handoff",
		PayloadJSON: string(payloadJSON),
	})
	if err != nil {
		t.Fatalf("encolar handoff: %v", err)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get order: %+v err=%v", order, err)
	}

	for {
		select {
		case <-CanalNotificaciones:
		default:
			goto handoffNotifDrained
		}
	}

handoffNotifDrained:
	if err := registrarEvidenciaHandoffReanudadoPorOrden(order, 1234); err != nil {
		t.Fatalf("registrarEvidenciaHandoffReanudadoPorOrden: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Estado != TareaEnProgreso {
		t.Fatalf("tarea sin pasar a progreso: %+v", tarea)
	}
	if !strings.Contains(tarea.Notas, "handoff completado por Codex1 en sesion #1234") {
		t.Fatalf("notas sin evidencia de handoff: %s", tarea.Notas)
	}

	var notif *EventoNotificacion
	for {
		select {
		case item := <-CanalNotificaciones:
			if item.Tipo == "runtime_handoff" {
				copy := item
				notif = &copy
			}
		default:
			goto handoffNotifChecked
		}
	}

handoffNotifChecked:
	if notif == nil {
		t.Fatal("faltaba notificacion runtime_handoff")
	}
	if notif.Payload == nil {
		t.Fatalf("runtime_handoff sin payload util: %+v", notif)
	}
	if got, _ := notif.Payload["order_id"].(int64); got != orderID {
		t.Fatalf("runtime_handoff con order_id inesperado: %+v", notif.Payload)
	}
	if got, _ := notif.Payload["task_id"].(int64); got != tareaID {
		t.Fatalf("runtime_handoff con task_id inesperado: %+v", notif.Payload)
	}
	if !strings.Contains(notif.Texto, "Codex0 -> Codex1") || !strings.Contains(notif.Texto, "handoff listo") {
		t.Fatalf("runtime_handoff sin resumen util: %+v", notif)
	}
}

func TestRuntimeOrderStartPoolLocalReutilizaSesionActivaExistente(t *testing.T) {
	tmp := prepararDBTemporal(t)
	srv := newTestHTTPServerOrSkip(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/launch":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"external_session_id": "ollama-pool-1",
				"handle_ref":          "ollama-pool-1",
				"handle_kind":         "api",
				"logical_state":       "ready",
				"capabilities": map[string]any{
					"can_send_input": true,
					"can_stop":       true,
				},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}
	}))
	defer srv.Close()

	if err := RegistrarAgente("GemmaPool1", "programador"); err != nil {
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
		Slug:       "ollama_pool_local",
		Nombre:     "Ollama Pool Local",
		Transporte: "api",
		Comando:    srv.URL,
		MetadataJSON: `{
			"familia":"ollama",
			"pool_compartido":true,
			"default_task_profile":"implementacion",
			"default_reasoning_effort":"high",
			"launch_path":"/launch",
			"input_path":"/input",
			"stop_path":"/stop",
			"status_path":"/status"
		}`,
		Activo: true,
	}); err != nil {
		t.Fatalf("upsert conector pool: %v", err)
	}

	sesionInicial, err := IniciarSesionContexto(SesionInicio{
		Agente:             "GemmaPool1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "ollama_pool_local",
		ExternalSessionID:  "ollama-pool-1",
		ResumePayloadJSON:  `{"driver":"ollama_pool_local","modelo":"gemma4:26b"}`,
		ResumenContinuidad: "continuidad previa",
	})
	if err != nil {
		t.Fatalf("iniciar sesion activa: %v", err)
	}

	var sesionesAntes int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM sesiones WHERE agente=? AND proyecto_id=?`, "GemmaPool1", proyectoID).Scan(&sesionesAntes); err != nil {
		t.Fatalf("count sesiones antes: %v", err)
	}

	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "GemmaPool1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","conector":"ollama_pool_local","modelo":"gemma4:26b","perfil_tarea":"implementacion"}`,
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

	sesionActiva, err := GetSesionActiva("GemmaPool1", &proyectoID)
	if err != nil || sesionActiva == nil {
		t.Fatalf("sesion activa: %+v err=%v", sesionActiva, err)
	}
	if sesionActiva.ID != sesionInicial.ID {
		t.Fatalf("deberia reutilizar la sesion inicial, got=%d want=%d", sesionActiva.ID, sesionInicial.ID)
	}
	if sesionActiva.ExternalSessionID != "ollama-pool-1" {
		t.Fatalf("external_session_id inesperado: %s", sesionActiva.ExternalSessionID)
	}

	var sesionesDespues int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM sesiones WHERE agente=? AND proyecto_id=?`, "GemmaPool1", proyectoID).Scan(&sesionesDespues); err != nil {
		t.Fatalf("count sesiones despues: %v", err)
	}
	if sesionesDespues != sesionesAntes {
		t.Fatalf("no deberia crear una sesion nueva, before=%d after=%d", sesionesAntes, sesionesDespues)
	}

	handle, err := GetRuntimeHandleBySesionID(sesionActiva.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle activo: %+v err=%v", handle, err)
	}
	if handle.SesionID == nil || *handle.SesionID != sesionActiva.ID {
		t.Fatalf("handle no ligado a la sesion reutilizada: %+v", handle)
	}
	if strings.Contains(handle.MetadataJSON, `"supervisor_driver"`) || strings.Contains(handle.MetadataJSON, `"log_path"`) {
		t.Fatalf("un handle remoto de pool local no debería activar supervisor local: %s", handle.MetadataJSON)
	}

	runtime, err := GetRuntimeBySesionID(sesionActiva.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	if strings.TrimSpace(runtime.LogicalState) == "" {
		t.Fatalf("runtime deberia quedar con logical_state util: %+v", runtime)
	}
}

func TestInferirTransporteArranqueRemoteSessionNoSeConvierteATmux(t *testing.T) {
	got := inferirTransporteArranque(&controlruntime.ProcesoArrancado{
		HandleKind:   "session",
		HandleRef:    "ollama-pool-1",
		MetadataJSON: `{"driver":"remote_http","transport":"api","handle_ref":"ollama-pool-1"}`,
	})
	if got != "api" {
		t.Fatalf("transporte inesperado para session remota: %q", got)
	}
}

func TestResolverDestinoRuntimeOrderSendInstructionRecuperaSesionActivaPoolLocal(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("GemmaPool1", "programador"); err != nil {
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
		Slug:       "ollama_pool_local",
		Nombre:     "Ollama Pool Local",
		Transporte: "api",
		Comando:    "http://127.0.0.1:17715",
		MetadataJSON: `{
			"driver":"ollama_pool_local",
			"transport":"api",
			"input_path":"/api/runtime/ollama-pool/input",
			"status_path":"/api/runtime/ollama-pool/status",
			"stop_path":"/api/runtime/ollama-pool/stop",
			"mailbox_delivery_mode":"interactive"
		}`,
		Activo: true,
	})
	if err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	resumePayload := `{
		"driver":"ollama_pool_local",
		"transport":"api",
		"endpoint":"http://127.0.0.1:17715",
		"input_path":"/api/runtime/ollama-pool/input",
		"status_path":"/api/runtime/ollama-pool/status",
		"stop_path":"/api/runtime/ollama-pool/stop",
		"mailbox_delivery_mode":"interactive",
		"can_send_input":true,
		"pool_compartido":true,
		"perfil_ejecucion":{
			"driver":"ollama_pool_local",
			"transport":"api",
			"endpoint":"http://127.0.0.1:17715",
			"input_path":"/api/runtime/ollama-pool/input",
			"status_path":"/api/runtime/ollama-pool/status",
			"stop_path":"/api/runtime/ollama-pool/stop",
			"mailbox_delivery_mode":"interactive",
			"can_send_input":true,
			"pool_compartido":true,
			"modelo":"gemma4:26b",
			"perfil_tarea":"implementacion",
			"pool_slug":"ollama-gemma4"
		}
	}`
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "GemmaPool1",
		ConectorID:        &conectorID,
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "ollama_pool_local",
		ExternalSessionID: "ollama-pool-1",
		ResumePayloadJSON: resumePayload,
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle inicial: %+v err=%v", handle, err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='fallido' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("marcar handle fallido: %v", err)
	}
	runtimeHandleHotReset()

	order := &RuntimeOrder{
		Agente:     "GemmaPool1",
		ProyectoID: &proyectoID,
		Tipo:       "send_instruction",
	}
	runtime, resolvedHandle, err := resolverDestinoRuntimeOrderSendInstruction(order, nil, nil)
	if err != nil {
		t.Fatalf("resolver destino send_instruction: %v", err)
	}
	if resolvedHandle == nil {
		t.Fatalf("deberia recuperar handle desde sesion activa pool local")
	}
	if runtime == nil {
		t.Fatalf("deberia recuperar runtime desde sesion activa pool local")
	}
	if resolvedHandle.SesionID == nil || *resolvedHandle.SesionID != sesion.ID {
		t.Fatalf("handle recuperado no pertenece a la sesion activa: %+v", resolvedHandle)
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

func TestRetenerRuntimeOrderSendInstructionNotificadaNoReabreOrdenTerminal(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto",
		Nombre:  "Proyecto",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Gemma1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"source":"microprogramacion","mailbox_id":1,"microprogramacion":{"especificacion_id":1,"archivo_objetivo":"cabeceras/merge_headers.go","simbolo_objetivo":"FusionarCabecerasCanonicas","write_set":["cabeceras/merge_headers.go"]}}`,
	})
	if err != nil {
		t.Fatalf("crear runtime order: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(orderID, "completada", `{"delivery_state":"delivered","receipt_source":"ollama_pool_local_materializada"}`, ""); err != nil {
		t.Fatalf("marcar completada: %v", err)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get runtime order: %+v err=%v", order, err)
	}
	if !runtimeOrderYaTieneEntregaValida(order) {
		t.Fatalf("la orden deberia quedar marcada como entrega valida: %+v", order)
	}
	payload := map[string]any{
		"mailbox_id":        int64(1),
		"mailbox_kind":      "instruction",
		"to_agente":         "Gemma1",
		"from_agente":       "server",
		"microprogramacion": map[string]any{"especificacion_id": 1},
	}
	if err := retenerRuntimeOrderSendInstructionNotificada(order, payload, "interactive dispatch pending receipt", time.Now().UTC()); err != nil {
		t.Fatalf("retenerRuntimeOrderSendInstructionNotificada: %v", err)
	}
	order, err = GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get runtime order posterior: %+v err=%v", order, err)
	}
	if order.Estado != "completada" {
		t.Fatalf("la orden terminal no deberia reabrirse: %+v", order)
	}
	if !strings.Contains(order.ResultadoJSON, `"receipt_source":"ollama_pool_local_materializada"`) {
		t.Fatalf("resultado terminal alterado: %s", order.ResultadoJSON)
	}
}

func TestRuntimeOrderSendInstructionReceiptAtOrAfterAceptaMismoInstante(t *testing.T) {
	baseline := time.Date(2026, time.April, 12, 9, 39, 44, 0, time.UTC)
	if !runtimeOrderSendInstructionReceiptAtOrAfter(baseline, baseline) {
		t.Fatalf("deberia aceptar receipt en el mismo instante del baseline")
	}
	if runtimeOrderSendInstructionReceiptAtOrAfter(baseline.Add(-time.Nanosecond), baseline) {
		t.Fatalf("no deberia aceptar receipt anterior al baseline")
	}
}

func TestRuntimeOrderSendInstructionReceiptBaselineRespetaCreacionDeOrdenAunqueMailboxYaEsteEntregada(t *testing.T) {
	deliveredAt := time.Date(2026, time.April, 14, 13, 2, 8, 0, time.UTC)
	createdAt := deliveredAt.Add(14 * time.Second)
	order := &RuntimeOrder{
		CreatedAt: createdAt,
	}
	msg := &RuntimeMailboxMessage{
		CreatedAt:   deliveredAt.Add(-5 * time.Second),
		DeliveredAt: &deliveredAt,
	}
	baseline := runtimeOrderSendInstructionReceiptBaseline(order, msg)
	if !baseline.Equal(createdAt) {
		t.Fatalf("baseline deberia elevarse a la creacion de la orden cuando mailbox ya estaba entregada: got=%s want=%s", baseline.Format(time.RFC3339Nano), createdAt.Format(time.RFC3339Nano))
	}
}

func TestEjecutarRuntimeOrderSendInstructionCompletaPorTranscriptAunqueMailboxSigaPendiente(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto",
		Nombre:  "Proyecto",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Gemma1",
		ProyectoID:        &proyectoID,
		CWD:               t.TempDir(),
		Herramienta:       "ollama_pool_local",
		ExternalSessionID: "ollama-pool-1",
		ResumePayloadJSON: `{"driver":"ollama_pool_local","transport":"api","pool_local":true}`,
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET transporte='api', estado='activo', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":true}`, `{"driver":"ollama_pool_local","transport":"api","pool_local":true,"can_send_input":true}`, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	msgID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Gemma1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"source":"microprogramacion","texto":"haz el cambio"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Gemma1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"mailbox_id":%d,"mailbox_kind":"instruction","source":"microprogramacion","microprogramacion":{"especificacion_id":2,"archivo_objetivo":"microprogramacionapp/extractor_entrega.go","write_set":["microprogramacionapp/extractor_entrega.go"]},"texto":"haz el cambio"}`, msgID),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	notifiedAt := time.Now().UTC().Add(-time.Minute)
	if err := retenerRuntimeOrderSendInstructionNotificada(&RuntimeOrder{ID: orderID, ResultadoJSON: `{}`}, map[string]any{
		"mailbox_id":   msgID,
		"mailbox_kind": "instruction",
		"source":       "microprogramacion",
		"microprogramacion": map[string]any{
			"especificacion_id": 2,
			"archivo_objetivo":  "microprogramacionapp/extractor_entrega.go",
			"write_set":         []any{"microprogramacionapp/extractor_entrega.go"},
		},
	}, "interactive dispatch pending receipt", notifiedAt); err != nil {
		t.Fatalf("retener notificada: %v", err)
	}
	if _, err := RegistrarRuntimeTranscript(&RuntimeTranscriptEntry{
		RuntimeID:      runtime.ID,
		HandleID:       &handle.ID,
		Agente:         "Gemma1",
		ProyectoID:     &proyectoID,
		Stream:         "pty_out",
		Text:           "// FILE: microprogramacionapp/extractor_entrega.go\npackage microprogramacionapp\n",
		NormalizedText: strings.ToLower("// FILE: microprogramacionapp/extractor_entrega.go\npackage microprogramacionapp\n"),
	}); err != nil {
		t.Fatalf("registrar transcript: %v", err)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get order: %+v err=%v", order, err)
	}
	if err := ejecutarRuntimeOrderSendInstruction(order); err != nil {
		t.Fatalf("ejecutar send_instruction: %v", err)
	}
	order, err = GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get order posterior: %+v err=%v", order, err)
	}
	if order.Estado != "completada" {
		t.Fatalf("la orden deberia completarse por evidencia de transcript: %+v", order)
	}
	if !strings.Contains(order.ResultadoJSON, `"dispatch_state":"delivered"`) {
		t.Fatalf("resultado sin dispatch_state delivered: %s", order.ResultadoJSON)
	}
	if !strings.Contains(order.ResultadoJSON, `"delivery_state":"delivered"`) {
		t.Fatalf("resultado sin delivery_state delivered: %s", order.ResultadoJSON)
	}
	msg, err := GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox posterior: %+v err=%v", msg, err)
	}
	if !strings.EqualFold(strings.TrimSpace(msg.Estado), "consumido") {
		t.Fatalf("la mailbox deberia quedar consumida: %+v", msg)
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

func TestGetRuntimeHandleBySesionIDNormalizaTMUXCanonicoEnMemoria(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexTMUX", "programador"); err != nil {
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
		Agente:             "CodexTMUX",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-tmux-normalize",
		ResumenContinuidad: "continuidad tmux",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	metaJSON := `{"driver":"tmux_cli_session","transporte":"tmux","tmux_session":"orq-codextmux-101500","tmux_pane_id":"%3"}`
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET transporte='cli',
		    handle_kind='process',
		    handle_ref='12345',
		    metadata_json=?
		WHERE sesion_id=?`, metaJSON, sesion.ID); err != nil {
		t.Fatalf("inyectar handle tmux legacy: %v", err)
	}

	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if got := strings.TrimSpace(handle.Transporte); got != "tmux" {
		t.Fatalf("transporte no normalizado: %q", got)
	}
	if got := strings.TrimSpace(handle.HandleKind); got != "session" {
		t.Fatalf("handle_kind no normalizado: %q", got)
	}
	if got := strings.TrimSpace(handle.HandleRef); got != "orq-codextmux-101500/%3" {
		t.Fatalf("handle_ref no normalizado: %q", got)
	}
}

func TestGetRuntimeHandleBySesionIDNoDevuelveSnapshotCalienteEliminado(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("CodexSesionHot", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	sesionID, err := IniciarSesion("CodexSesionHot")
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesionID)
	if err != nil || handle == nil {
		t.Fatalf("get handle inicial: %+v err=%v", handle, err)
	}

	runtimeHandleHotReset()
	if err := runtimeHandleHotEnsureLoaded(); err != nil {
		t.Fatalf("runtimeHandleHotEnsureLoaded: %v", err)
	}

	if _, err := DB.Exec(`DELETE FROM runtime_handles WHERE id = ?`, handle.ID); err != nil {
		t.Fatalf("borrar handle persistido: %v", err)
	}

	cached, err := GetRuntimeHandleBySesionID(sesionID)
	if err != nil {
		t.Fatalf("get handle tras borrar persistido: %v", err)
	}
	if cached != nil {
		t.Fatalf("no deberia devolver snapshot caliente eliminado, got=%+v", cached)
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
	legacyMeta := `{"bootstrap_prompt":"Bootstrap legacy para listado.","launch_prompt_embedded":true,"rendered_command":"'/tmp/codex-perfiles/bin/codex-perfil' 'Codex1' '--model' 'gpt-5.4' 'Bootstrap de Orquesta para Codex1.\nLinea 2'","wrapped_command":"script -q -e -f -c '/tmp/codex-perfiles/bin/codex-perfil Codex1 Bootstrap de Orquesta para Codex1.' '/tmp/pty.log'"}`
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
	if got := stringFromMap(meta, "rendered_command", ""); got != "'/tmp/codex-perfiles/bin/codex-perfil' 'Codex1'" {
		t.Fatalf("rendered_command no compactado: %q meta=%s", got, handle.MetadataJSON)
	}
	if got := stringFromMap(meta, "wrapped_command", ""); got != "'script' '-q' '-e' '-f' '<omitted>' '/tmp/pty.log'" {
		t.Fatalf("wrapped_command no compactado: %q meta=%s", got, handle.MetadataJSON)
	}
	if got, ok := meta["rendered_command_len"].(float64); !ok || int(got) <= len([]rune(stringFromMap(meta, "rendered_command", ""))) {
		t.Fatalf("falta rendered_command_len util: %+v meta=%s", meta["rendered_command_len"], handle.MetadataJSON)
	}
}

func TestListarRuntimeHandlesNormalizaTMUXCanonicoEnMemoria(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("CodexTMUXList", "programador"); err != nil {
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
		Agente:             "CodexTMUXList",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-tmux-list",
		ResumenContinuidad: "continuidad tmux list",
		Branch:             "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	metaJSON := `{"driver":"tmux_cli_session","tmux_session":"orq-codextmux-list","tmux_pane_id":"%9"}`
	if _, err := DB.Exec(`
		UPDATE runtime_handles
		SET transporte='cli',
		    handle_kind='process',
		    handle_ref='77777',
		    metadata_json=?
		WHERE sesion_id=?`, metaJSON, sesion.ID); err != nil {
		t.Fatalf("inyectar handle legacy: %v", err)
	}

	agente := "CodexTMUXList"
	handles, err := ListarRuntimeHandles(&agente)
	if err != nil {
		t.Fatalf("listar handles: %v", err)
	}
	if len(handles) == 0 {
		t.Fatalf("sin handles listados")
	}
	handle := handles[0]
	if got := strings.TrimSpace(handle.Transporte); got != "tmux" {
		t.Fatalf("transporte no normalizado: %q", got)
	}
	if got := strings.TrimSpace(handle.HandleKind); got != "session" {
		t.Fatalf("handle_kind no normalizado: %q", got)
	}
	if got := strings.TrimSpace(handle.HandleRef); got != "orq-codextmux-list/%9" {
		t.Fatalf("handle_ref no normalizado: %q", got)
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

func TestProcesarRuntimeOrdersBatchMantieneHandoffBootstrapPendiente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex0", "programador"); err != nil {
		t.Fatalf("registrar Codex0: %v", err)
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

	handoffPayload, err := json.Marshal(HandoffPayload{
		AgenteOrigen:       "Codex0",
		AgenteDestino:      "Codex1",
		Motivo:             "traspaso",
		ResumenContinuidad: "handoff pendiente para bootstrap",
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
	syncID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "sync_status",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar sync_status: %v", err)
	}

	n, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba procesar solo sync_status y reservar handoff para bootstrap, got=%d", n)
	}

	syncOrder, err := GetRuntimeOrder(syncID)
	if err != nil {
		t.Fatalf("get sync order: %v", err)
	}
	if syncOrder == nil || syncOrder.Estado != "completada" {
		t.Fatalf("sync order no completada: %+v", syncOrder)
	}

	handoffOrder, err := GetRuntimeOrder(handoffID)
	if err != nil {
		t.Fatalf("get handoff order: %v", err)
	}
	if handoffOrder == nil || handoffOrder.Estado != "pendiente" {
		t.Fatalf("handoff bootstrap no deberia consumirse en el batch: %+v", handoffOrder)
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

func TestProcesarRuntimeOrdersBatchReconciliarStaleAntesDeDespachar(t *testing.T) {
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
	old := time.Now().UTC().Add(-10 * time.Minute)
	if _, err := DB.Exec(`
		UPDATE runtime_orders
		SET started_at = ?, updated_at = ?, lease_expires_at = ?
		WHERE id = ?`,
		old, old, old, orderID,
	); err != nil {
		t.Fatalf("envejecer order: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("deberia reconciliar al menos una order stale, got=%d", processed)
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if order.Estado != "pendiente" && order.Estado != "completada" {
		t.Fatalf("la orden stale no deberia seguir ejecutando: %+v", order)
	}
}

func TestMarcarRuntimeOrderEjecutandoLimpiaDeferredEnStart(t *testing.T) {
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
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	resultadoPrevio := `{"deferred":true,"deferred_reason":"sesion_fantasma_en_otro_proyecto:10","retry_after":"2026-04-17T22:24:11Z"}`
	if _, err := DB.Exec(`UPDATE runtime_orders SET resultado_json=? WHERE id=?`, resultadoPrevio, orderID); err != nil {
		t.Fatalf("sembrar resultado previo: %v", err)
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if err := MarcarRuntimeOrderEjecutando(order); err != nil {
		t.Fatalf("marcar ejecutando: %v", err)
	}

	order, err = GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("reload order: %v", err)
	}
	resultado := mapFromJSON(order.ResultadoJSON)
	if boolFromMap(resultado, "deferred") {
		t.Fatalf("deferred no deberia persistir al volver a ejecutando: %+v", resultado)
	}
	if reason := strings.TrimSpace(stringFromMap(resultado, "deferred_reason", "")); reason != "" {
		t.Fatalf("deferred_reason no deberia persistir al volver a ejecutando: %+v", resultado)
	}
	if retry := strings.TrimSpace(stringFromMap(resultado, "retry_after", "")); retry != "" {
		t.Fatalf("retry_after no deberia persistir al volver a ejecutando: %+v", resultado)
	}
	if !boolFromMap(resultado, "in_progress") {
		t.Fatalf("faltaba in_progress en resultado ejecutando: %+v", resultado)
	}
	if phase := strings.TrimSpace(stringFromMap(resultado, "phase", "")); phase != "launching" {
		t.Fatalf("phase inesperada: %+v", resultado)
	}
}

func TestProcesarRuntimeOrdersBatchCompletaStartEjecutandoSiRuntimeYaEstaOperativo(t *testing.T) {
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
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances SET logical_state='activo', process_state='running' WHERE id=?`, runtime.ID); err != nil {
		t.Fatalf("promover runtime: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtime.ID,
		HandleID:    &handle.ID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(orderID, "ejecutando", `{"in_progress":true,"phase":"launching"}`, ""); err != nil {
		t.Fatalf("marcar ejecutando: %v", err)
	}

	processed, err := ProcesarRuntimeOrdersBatch()
	if err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if processed < 1 {
		t.Fatalf("deberia reconciliar al menos una orden, got=%d", processed)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("reload order: %v", err)
	}
	if order.Estado != "completada" {
		t.Fatalf("la start ejecutando deberia completarse al ver runtime operativo: %+v", order)
	}
	if !strings.Contains(order.ResultadoJSON, `"reason":"runtime_worker_already_running"`) &&
		!strings.Contains(order.ResultadoJSON, `"reason":"runtime_already_running"`) &&
		!strings.Contains(order.ResultadoJSON, `"reason":"runtime_worker_recovered_after_order"`) {
		t.Fatalf("resultado inesperado: %s", order.ResultadoJSON)
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

func TestReconciliarRuntimeOrdersStaleCompletaStartSiYaHayRuntimeRecuperado(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("GemmaApp8", "programador"); err != nil {
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
		Agente:      "GemmaApp8",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"accion":"start","proyecto":"orquestador"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	if err := MarcarRuntimeOrderEstado(orderID, "ejecutando", `{}`, ""); err != nil {
		t.Fatalf("marcar ejecutando: %v", err)
	}
	old := time.Now().UTC().Add(-10 * time.Minute)
	if _, err := DB.Exec(`UPDATE runtime_orders SET started_at=?, updated_at=?, lease_expires_at=? WHERE id=?`, old, old, old, orderID); err != nil {
		t.Fatalf("envejecer order: %v", err)
	}

	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "GemmaApp8",
		ProyectoID:   &proyectoID,
		LogicalState: "disponible",
		ProcessState: "running",
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"driver":     "remote_http",
		"transport":  "api",
		"connector":  "ollama_pool_local",
		"pool_local": true,
	})
	if err != nil {
		t.Fatalf("marshal metadata handle: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, CURRENT_TIMESTAMP)`,
		"GemmaApp8", proyectoID, runtimeID, "api", "session", "ollama-pool-gemmaapp8", string(metaJSON)); err != nil {
		t.Fatalf("insert handle: %v", err)
	}

	n, err := ReconciliarRuntimeOrdersStale()
	if err != nil {
		t.Fatalf("reconciliar runtime orders stale: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba completar 1 orden stale satisfecha, got=%d", n)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if order.Estado != "completada" {
		t.Fatalf("la orden stale deberia quedar completada: %+v", order)
	}
	if !strings.Contains(order.ResultadoJSON, `"obsoleta":true`) {
		t.Fatalf("resultado sin marca obsoleta: %s", order.ResultadoJSON)
	}
}

func TestRuntimeOrderControlEstadoDeseadoSatisfechoStartApiActivo(t *testing.T) {
	runtime := &RuntimeInstance{
		LogicalState: "disponible",
		ProcessState: "running",
	}
	handle := &RuntimeHandle{
		Transporte: "api",
		Estado:     "activo",
	}
	order := &RuntimeOrder{Tipo: "start"}
	ok, reason := runtimeOrderControlEstadoDeseadoSatisfecho(order, runtime, handle)
	if !ok || reason != "runtime_handle_api_active" {
		t.Fatalf("deberia satisfacer start para api/session activa, ok=%v reason=%q", ok, reason)
	}
}

func TestRuntimeOrderControlEstadoDeseadoSatisfechoStartSinHandlePeroConSesionActiva(t *testing.T) {
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
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		ProyectoID:   &proyectoID,
		SesionID:     &sesion.ID,
		Connector:    "codex-cli",
		LogicalState: "activo",
		ProcessState: "running",
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	runtime, err := GetRuntime(runtimeID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	order := &RuntimeOrder{Agente: "Codex1", ProyectoID: &proyectoID, Tipo: "start"}
	ok, reason := runtimeOrderControlEstadoDeseadoSatisfecho(order, runtime, nil)
	if !ok || reason != "runtime_already_running" {
		t.Fatalf("deberia satisfacer start con runtime+sesion activa sin handle, ok=%v reason=%q", ok, reason)
	}
}

func TestReconciliarRuntimeOrdersPendientesControlObsoletasNoCompletaStartSiHayStopVivaPrevia(t *testing.T) {
	tmp := prepararDBTemporal(t)
	workerDir := filepath.Join(tmp, "worker-running-stop-previa")
	if err := os.MkdirAll(workerDir, 0o755); err != nil {
		t.Fatalf("mkdir worker dir: %v", err)
	}
	statusPath := filepath.Join(workerDir, "status.json")
	heartbeatPath := filepath.Join(workerDir, "heartbeat.json")
	now := time.Now().UTC()
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"alive":      true,
		"updated_at": now.Format(time.RFC3339Nano),
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
	})
	if err := os.WriteFile(statusPath, append(statusRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, append(heartbeatRaw, '\n'), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
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
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances SET logical_state='activo', process_state='running' WHERE id=?`, runtime.ID); err != nil {
		t.Fatalf("promover runtime: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='activo', metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("actualizar handle: %v", err)
	}
	stopID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "stop",
		PayloadJSON: `{"accion":"stop","por":"orquesta","motivo":"restart"}`,
	})
	if err != nil {
		t.Fatalf("encolar stop: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"accion":"start","por":"orquesta","motivo":"restart","proyecto":"orquestador"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	if stopID >= startID {
		t.Fatalf("esperaba stop previa a start, stop=%d start=%d", stopID, startID)
	}
	processed, err := reconciliarRuntimeOrdersPendientesControlObsoletas()
	if err != nil {
		t.Fatalf("reconciliar stale: %v", err)
	}
	if processed != 0 {
		t.Fatalf("no deberia completar start con stop viva previa, processed=%d", processed)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start final: %v", err)
	}
	if startOrder.Estado != "pendiente" {
		t.Fatalf("start no deberia completarse: %+v", startOrder)
	}
}

func TestEjecutarRuntimeOrderStartCompletaSiRuntimeYaActivoSinHandle(t *testing.T) {
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
	if _, err := DB.Exec(`DELETE FROM runtime_handles WHERE sesion_id=?`, sesion.ID); err != nil {
		t.Fatalf("borrar handles: %v", err)
	}
	runtime, err := GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime sesion: %+v err=%v", runtime, err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances SET logical_state='activo', process_state='running' WHERE id=?`, runtime.ID); err != nil {
		t.Fatalf("promover runtime: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"accion":"start","por":"orquesta","proyecto":"orquestador"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	startOrder, err := GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start order: %v", err)
	}
	if err := ejecutarRuntimeOrderStart(startOrder); err != nil {
		t.Fatalf("ejecutar start redundante: %v", err)
	}
	startOrder, err = GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start final: %v", err)
	}
	if startOrder.Estado != "completada" {
		t.Fatalf("start redundante deberia completarse: %+v", startOrder)
	}
	if !strings.Contains(startOrder.ResultadoJSON, `"reason":"runtime_already_running"`) {
		t.Fatalf("resultado inesperado: %s", startOrder.ResultadoJSON)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("deberia materializar handle canónico al completar start: handle=%+v err=%v", handle, err)
	}
	if got := strings.TrimSpace(handle.Agente); got != "Codex1" {
		t.Fatalf("handle materializado con agente inesperado: %+v", handle)
	}
}

func TestRuntimeHandleOmitePromptArranqueInteractivoParaOllamaPoolLocal(t *testing.T) {
	plan := &runtimeagente.LaunchPlan{
		Driver: "remote_http",
	}
	runtime := &RuntimeInstance{
		Connector: "ollama_pool_local",
	}
	handle := &RuntimeHandle{
		Transporte:   "api",
		MetadataJSON: `{"driver":"ollama_pool_local","transport":"api"}`,
	}
	if !runtimeHandleOmitePromptArranqueInteractivo(handle, runtime, plan) {
		t.Fatal("ollama_pool_local no deberia inyectar prompt de arranque interactivo")
	}
}

func TestRuntimeOrderPromptArranqueDebeDiferirsePorTrustTMUX(t *testing.T) {
	handle := &RuntimeHandle{
		Transporte:   "tmux",
		MetadataJSON: `{"driver":"tmux_cli_session","tmux_pane_id":"%1","tmux_session":"orq-gemini1"}`,
	}
	if !runtimeOrderPromptArranqueDebeDiferirse(handle, fmt.Errorf("tmux pane requiere carpeta de confianza")) {
		t.Fatal("el prompt de arranque deberia diferirse cuando el worker esta bloqueado por trust")
	}
	if !runtimeOrderPromptArranqueDebeDiferirse(handle, fmt.Errorf("tmux pane requiere autenticacion manual")) {
		t.Fatal("el prompt de arranque deberia diferirse cuando el worker esta bloqueado por auth")
	}
	if runtimeOrderPromptArranqueDebeDiferirse(handle, fmt.Errorf("fallo fatal irrelevante")) {
		t.Fatal("errores no relacionados no deberian diferir el prompt de arranque")
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

func TestReconciliarRuntimeOrderStaleCompletaBootstrapSiYaGeneroStartDeSeguimiento(t *testing.T) {
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
		PayloadJSON: `{"motivo":"test"}`,
	})
	if err != nil {
		t.Fatalf("encolar handoff: %v", err)
	}
	startID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"accion":"start"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	old := time.Now().UTC().Add(-10 * time.Minute)
	resultado := fmt.Sprintf(`{"start_order_id":%d,"delivery_state":"delivered"}`, startID)
	if _, err := DB.Exec(`
		UPDATE runtime_orders
		SET estado='tomada',
		    started_at=?,
		    updated_at=?,
		    available_at=?,
		    resultado_json=?
		WHERE id = ?`,
		old, old, old, resultado, handoffID,
	); err != nil {
		t.Fatalf("envejecer handoff: %v", err)
	}

	applied, err := reconciliarRuntimeOrderStale(handoffID, "handoff", time.Now().UTC(), time.Now().UTC().Add(-time.Minute))
	if err != nil {
		t.Fatalf("reconciliar order stale: %v", err)
	}
	if !applied {
		t.Fatalf("la reconciliacion stale deberia completar la handoff bootstrap obsoleta")
	}

	order, err := GetRuntimeOrder(handoffID)
	if err != nil {
		t.Fatalf("get handoff: %v", err)
	}
	if order == nil || order.Estado != "completada" {
		t.Fatalf("handoff final inesperada: %+v", order)
	}
	if !strings.Contains(order.ResultadoJSON, `"superseded_reason":"bootstrap_followup_order_exists"`) {
		t.Fatalf("resultado sin supersede de bootstrap followup: %s", order.ResultadoJSON)
	}
	if !strings.Contains(order.ResultadoJSON, `"obsoleta":true`) {
		t.Fatalf("resultado sin marca obsoleta: %s", order.ResultadoJSON)
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

func TestReconciliarRuntimeOrderStaleSendInstructionUsaEstadoActualDeMailbox(t *testing.T) {
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
		Kind:        "nudge",
		PayloadJSON: `{"texto":"sigue"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxConsumido(mailboxID); err != nil {
		t.Fatalf("consumir mailbox: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"to_agente":"Codex1","mailbox_id":%d,"mailbox_kind":"nudge","texto":"sigue"}`, mailboxID),
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

	applied, err := reconciliarRuntimeOrderStale(orderID, "send_instruction", time.Now().UTC(), time.Now().UTC().Add(-time.Minute))
	if err != nil {
		t.Fatalf("reconciliar stale send_instruction: %v", err)
	}
	if !applied {
		t.Fatalf("reconciliar stale deberia resolver la send_instruction desde mailbox consumido")
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if order.Estado != "completada" {
		t.Fatalf("estado final inesperado: %+v", order)
	}
	if strings.Contains(order.ErrorText, "reencolada tras stale del control plane") {
		t.Fatalf("no deberia caer en stale generico: %+v", order)
	}
	if strings.Contains(order.ResultadoJSON, `"mailbox_only":true`) {
		t.Fatalf("resultado final no deberia recaer en mailbox_only legacy: %s", order.ResultadoJSON)
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

func writeFakeTMUXDispatchScriptDB(t *testing.T, dir, mode string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "fake-tmux-dispatch-db")
	stateDir := filepath.Join(dir, "fake-tmux-dispatch-db-state")
	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
state_dir=%q
mode=%q
mkdir -p "$state_dir"
printf '%%s\n' "$*" >> "$state_dir/invocations.log"
cmd="${1:-}"
shift || true
pane_state_file="$state_dir/pane_state"
prompt_file="$state_dir/prompt"
if [[ ! -f "$pane_state_file" ]]; then
  printf 'ready\n' > "$pane_state_file"
fi
case "$cmd" in
  display-message)
    printf 'orq-test|worker|%%77|12345|0|bash|%s\n' "$state_dir"
    ;;
  capture-pane)
    pane_state="$(cat "$pane_state_file" 2>/dev/null || printf 'ready')"
    prompt="$(cat "$prompt_file" 2>/dev/null || true)"
    case "$pane_state" in
      ready)
        printf '%%s\n' '> ready'
        ;;
      draft)
        printf '%%s\n' "> $prompt"
        ;;
      active)
        printf '%%s\n' 'Esc to interrupt'
        ;;
      *)
        printf '%%s\n' '> ready'
        ;;
    esac
    ;;
  send-keys)
    if [[ "${1:-}" == "-t" ]]; then
      shift 2
    fi
    if [[ "${1:-}" == "-l" ]]; then
      shift
      printf '%%s' "$*" > "$prompt_file"
      printf 'draft\n' > "$pane_state_file"
      exit 0
    fi
    key="${1:-}"
    if [[ "$key" == "Enter" || "$key" == "C-m" ]]; then
      if [[ "$mode" == "active_after_enter" ]]; then
        printf 'active\n' > "$pane_state_file"
      else
        printf 'draft\n' > "$pane_state_file"
      fi
      exit 0
    fi
    exit 0
    ;;
  *)
    echo "unsupported fake tmux command: $cmd" >&2
    exit 1
    ;;
esac
`, stateDir, mode, "%s")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux dispatch db: %v", err)
	}
	return scriptPath
}

func writeFakeTMUXPanePatchScriptDB(t *testing.T, dir string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "fake-tmux-pane-patch")
	patch := strings.Join([]string{
		"```diff",
		"--- identidad/normalizar_identificador.go",
		"+++ identidad/normalizar_identificador.go",
		"@@ -1,4 +1,10 @@",
		" package identidad",
		"+import (",
		"+  \"regexp\"",
		"+  \"strings\"",
		"+)",
		"",
		" Evidencia:",
		" - patch aplicado",
	}, "\n")
	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
cmd="${1:-}"
shift || true
case "$cmd" in
  has-session)
    exit 0
    ;;
  capture-pane)
    cat <<'EOF'
%s
EOF
    ;;
  *)
    exit 0
    ;;
esac
`, patch)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux pane patch: %v", err)
	}
	return scriptPath
}

func writeFakeTMUXGeminiAcceptEditsScriptDB(t *testing.T, dir string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "fake-tmux-gemini-accept-edits")
	capture := strings.Join([]string{
		"Refactoring Control Plane Logic",
		"Consolidation",
		"Shift+Tab to accept edits",
	}, "\n")
	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
cmd="${1:-}"
shift || true
case "$cmd" in
  has-session)
    exit 0
    ;;
  capture-pane)
    cat <<'EOF'
%s
EOF
    ;;
  *)
    exit 0
    ;;
esac
`, capture)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux gemini accept edits: %v", err)
	}
	return scriptPath
}

func writeFakeTMUXPanePatchWithKillScriptDB(t *testing.T, dir, invocations string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "fake-tmux-pane-patch-kill")
	patch := strings.Join([]string{
		"```diff",
		"--- identidad/normalizar_identificador.go",
		"+++ identidad/normalizar_identificador.go",
		"@@ -1,4 +1,10 @@",
		" package identidad",
		"+import \"strings\"",
		"+",
		"+func NormalizarIdentificadorTecnico(input string) string {",
		"+  return strings.ToLower(strings.TrimSpace(input))",
		"+}",
		"```",
	}, "\n")
	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
cmd="${1:-}"
shift || true
printf "%%s %%s\n" "$cmd" "$*" >> %q
case "$cmd" in
  has-session)
    exit 0
    ;;
  capture-pane)
    cat <<'EOF'
%s
EOF
    ;;
  kill-session)
    exit 0
    ;;
  *)
    exit 0
    ;;
esac
`, invocations, patch)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux pane patch kill: %v", err)
	}
	return scriptPath
}

func writeFakeTMUXNoPatchWithKillScriptDB(t *testing.T, dir, invocations string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "fake-tmux-no-patch-kill")
	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
cmd="${1:-}"
shift || true
printf "%%s %%s\n" "$cmd" "$*" >> %q
case "$cmd" in
  has-session)
    exit 0
    ;;
  capture-pane)
    cat <<'EOF'
>>> Send a message (/? for help)
EOF
    ;;
  kill-session)
    exit 0
    ;;
  *)
    exit 0
    ;;
esac
`, invocations)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux no patch kill: %v", err)
	}
	return scriptPath
}

func TestRuntimeOrderSendInstructionReceiptEvidenceReconoceDispatchLedger(t *testing.T) {
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
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	traceDir := filepath.Join(tmp, "trace-codex1")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}

	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		CWD:               workingDir,
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-ledger-1",
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

	metaJSON, err := json.Marshal(map[string]any{
		"driver":              "tmux_cli_session",
		"transport":           "tmux",
		"trace_dir":           traceDir,
		"external_session_id": "sess-ledger-1",
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET metadata_json=?, estado='activo' WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}
	handle, err = GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"texto":"continuar trabajo"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "send_instruction",
		PayloadJSON: fmt.Sprintf(
			`{"mailbox_id":%d,"mailbox_kind":"autonomia","texto":"continuar trabajo","external_session_id":"sess-ledger-1","delivery_attempt_signature":"sig-ledger-1"}`,
			mailboxID,
		),
	})
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get runtime order: %v", err)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil {
		t.Fatalf("get mailbox: %v", err)
	}
	payload := mapFromJSON(order.PayloadJSON)
	recordedAt := time.Date(2026, 4, 20, 10, 11, 12, 0, time.UTC)
	if err := controlruntime.RecordDispatchLedgerFromMetadataJSON(handle.MetadataJSON, controlruntime.DispatchLedgerRecordInput{
		RuntimeOrderID:           order.ID,
		MailboxID:                mailboxID,
		HandleID:                 handle.ID,
		DeliveryAttemptSignature: "sig-ledger-1",
		ExternalSessionID:        "sess-ledger-1",
		DispatchState:            "delivered",
		DeliveryState:            "consumed",
		ReceiptSource:            "dispatch_ledger",
		Reason:                   "test receipt",
		RecordedAt:               recordedAt,
	}); err != nil {
		t.Fatalf("record dispatch ledger: %v", err)
	}

	confirmed, source, receiptAt, err := runtimeOrderSendInstructionReceiptEvidence(order, payload, msg)
	if err != nil {
		t.Fatalf("runtimeOrderSendInstructionReceiptEvidence: %v", err)
	}
	if !confirmed {
		t.Fatal("deberia reconocer receipt desde dispatch ledger")
	}
	if source != "dispatch_ledger" {
		t.Fatalf("receipt source inesperado: %q", source)
	}
	if !receiptAt.Equal(recordedAt) {
		t.Fatalf("receiptAt inesperado: got=%s want=%s", receiptAt.Format(time.RFC3339Nano), recordedAt.Format(time.RFC3339Nano))
	}
}

func TestRuntimeOrderSendInstructionReceiptEvidenceReconoceMailboxConsumida(t *testing.T) {
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
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"continuar trabajo"}`,
	})
	if err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(mailboxID); err != nil {
		t.Fatalf("entregar mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxConsumido(mailboxID); err != nil {
		t.Fatalf("consumir mailbox: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: fmt.Sprintf(`{"mailbox_id":%d,"mailbox_kind":"pipeline_local","texto":"continuar trabajo"}`, mailboxID),
	})
	if err != nil {
		t.Fatalf("encolar order: %v", err)
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get runtime order: %v", err)
	}
	msg, err := GetRuntimeMailbox(mailboxID)
	if err != nil {
		t.Fatalf("get mailbox: %v", err)
	}
	payload := mapFromJSON(order.PayloadJSON)

	confirmed, source, receiptAt, err := runtimeOrderSendInstructionReceiptEvidence(order, payload, msg)
	if err != nil {
		t.Fatalf("runtimeOrderSendInstructionReceiptEvidence: %v", err)
	}
	if !confirmed {
		t.Fatal("deberia reconocer receipt desde mailbox consumida")
	}
	if source != "runtime_mailbox_consumed" {
		t.Fatalf("receipt source inesperado: %q", source)
	}
	if msg.ConsumedAt == nil || receiptAt.IsZero() || !receiptAt.Equal(msg.ConsumedAt.UTC()) {
		t.Fatalf("receiptAt inesperado: got=%s consumedAt=%v", receiptAt.Format(time.RFC3339Nano), msg.ConsumedAt)
	}
}
