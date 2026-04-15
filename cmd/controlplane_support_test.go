package cmd

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/capacidadapp"
	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/internal/controlruntime"
	"orquesta/microprogramacionapp"
	"orquesta/reviewapp"
	"orquesta/runtimeagente"
	"orquesta/runtimesapp"
)

func timePtr(v time.Time) *time.Time { return &v }

func TestProcesarSupervisionAutonomaBatchEncolaSupervisionYRegistraCiclo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.RegistrarAgente("CodexA", "admin"); err != nil {
		t.Fatalf("registrar agente alternativo: %v", err)
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if err := db.ActivarAsignacion("CodexA", proyectoID, "supervision alterna"); err != nil {
		t.Fatalf("activar asignacion alterna: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexA",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion alterna: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("procesar supervision: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 supervision, got=%d", n)
	}

	agente := "CodexSupervisor"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" || !strings.Contains(orders[0].PayloadJSON, `"accion":"supervisar_proyecto"`) {
		t.Fatalf("supervision no encolada: %+v", orders)
	}
	kind := "supervision"
	cycles, err := db.ListarAutonomiaCiclos(db.FiltroAutonomiaCiclos{ProyectoID: &proyectoID, Kind: &kind, Limit: 10})
	if err != nil {
		t.Fatalf("listar autonomia ciclos: %v", err)
	}
	if len(cycles) != 1 || cycles[0].Agente != "CodexSupervisor" {
		t.Fatalf("ciclo supervision inesperado: %+v", cycles)
	}
	policy, err := db.GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto autonomia: %v", err)
	}
	if policy.LastSupervisionAt == nil {
		t.Fatal("last_supervision_at debería informarse")
	}
}

func TestRevalidarYVerificarAgenteDisponibleParaTrabajoNoCaeAStatusSiFaltaEnBD(t *testing.T) {
	prepararDBTemporalCmd(t)

	prevStatus := statusService
	defer func() { statusService = prevStatus }()
	statusService = stubStatusService{
		response: apiStatusResponse{
			Agentes:           []*db.Agente{{Nombre: "CodexFantasma", EstadoCuota: "activo"}},
			AgentesActivos:    []*db.Agente{{Nombre: "CodexFantasma", EstadoCuota: "activo"}},
			AgentesTrabajando: []*db.Agente{{Nombre: "CodexFantasma", EstadoCuota: "activo"}},
		},
	}

	err := revalidarYVerificarAgenteDisponibleParaTrabajo("CodexFantasma")
	if err == nil {
		t.Fatal("deberia fallar si el agente no existe en BD")
	}
	if !strings.Contains(err.Error(), "no encontrado") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestConstruirInstruccionMailboxInteractivoAceptaPipelineLocal(t *testing.T) {
	msg := &db.RuntimeMailboxMessage{
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"haz el siguiente slice pequeño","texto":"fallback"}`,
	}
	got, ok := construirInstruccionMailboxInteractivo(msg)
	if !ok {
		t.Fatal("pipeline_local deberia producir instruccion interactiva")
	}
	if got != "fallback" && got != "haz el siguiente slice pequeño" {
		t.Fatalf("instruccion pipeline_local inesperada: %q", got)
	}
}

func TestRefrescarPresupuestoSesionObservadoDesdeSesionPrefiereHandleTMUXCanonico(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex7", "programador"); err != nil {
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
		Agente:            "Codex7",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-codex7-budget",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	base := filepath.Join(tmp, "codex-perfiles")
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	wrapper := filepath.Join(base, "bin", "codex-perfil")
	script := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" != "Codex7" || "${2:-}" != "status-json" ]]; then
  exit 9
fi
cat <<'JSON'
{"source":"codex_profile_status","profile":"Codex7","checked_at":"2026-04-07T09:40:00Z","observed_at":"2026-04-07T09:39:30Z","session_path":"/tmp/sess-7.jsonl","session_id":"sess-codex7-budget","account":{"email":"alberto@avidad.com","user":"Alberto","plan_type":"team"},"rate_limits":{"primary":{"used_percent":18,"left_percent":82,"window_minutes":300,"resets_at":"2026-04-07T11:40:00Z"},"secondary":{"used_percent":41,"left_percent":59,"window_minutes":10080,"resets_at":"2026-04-14T06:47:00Z"},"credits":null,"plan_type":"team"}}
JSON
`
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	runDir := filepath.Join(tmp, "runtime", "Codex7", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	writeJSON := func(path string, payload map[string]any) {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	writeJSON(manifestPath, map[string]any{
		"version":                1,
		"agent":                  "Codex7",
		"driver":                 "tmux_cli_session",
		"transport":              "tmux",
		"profile":                "Codex7",
		"profile_status_wrapper": wrapper,
		"tmux_session":           "orq-codex7-budget",
		"tmux_pane_id":           "%3",
		"status_path":            statusPath,
		"heartbeat_path":         heartbeatPath,
		"external_session_id":    "sess-codex7-budget",
	})
	writeJSON(statusPath, map[string]any{
		"state":               "running",
		"updated_at":          now,
		"alive":               true,
		"child_pid":           7007,
		"external_session_id": "sess-codex7-budget",
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":               true,
		"heartbeat_at":        now,
		"child_pid":           7007,
		"external_session_id": "sess-codex7-budget",
	})

	pid := int64(7007)
	runtimeID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "Codex7",
		ProyectoID:   &proyectoID,
		SesionID:     &sesion.ID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                 "tmux_cli_session",
		"tmux_session":           "orq-codex7-budget",
		"tmux_pane_id":           "%3",
		"worker_manifest_path":   manifestPath,
		"worker_status_path":     statusPath,
		"worker_heartbeat_path":  heartbeatPath,
		"profile_status_wrapper": wrapper,
		"profile_name":           "Codex7",
		"rendered_command":       "wrapper opaco sin perfil visible",
		"external_session_id":    "sess-codex7-budget",
	})
	if _, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, CURRENT_TIMESTAMP)`,
		"Codex7", proyectoID, runtimeID, "tmux", "session", "orq-codex7-budget/%3", string(metaJSON)); err != nil {
		t.Fatalf("insert handle: %v", err)
	}

	ok, err := refrescarPresupuestoSesionObservadoDesdeSesion(sesion)
	if err != nil {
		t.Fatalf("refrescar presupuesto desde sesion: %v", err)
	}
	if !ok {
		t.Fatal("deberia refrescar presupuesto via handle tmux canonico")
	}
	ultimo, err := db.UltimoPresupuestoSesionPorFuente(sesion.ID, "codex_profile_status")
	if err != nil {
		t.Fatalf("ultimo presupuesto codex_profile_status: %v", err)
	}
	if ultimo == nil {
		t.Fatal("deberia persistir presupuesto codex_profile_status")
	}
	agente, err := db.GetAgente("Codex7")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agente.CuentaEmail != "alberto@avidad.com" {
		t.Fatalf("cuenta inesperada tras refresh canonico: %+v", agente)
	}
	if agente.PresupuestoFuente != "codex_profile_status" {
		t.Fatalf("fuente de presupuesto inesperada: %+v", agente)
	}
}

func TestRuntimeRecuperacionSesionObjetivoPrefiereRuntimePrincipalCanonicoSiElHandleFreshNoEstaEnlazado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexRecover", "programador"); err != nil {
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

	resSesionLegacy, err := db.DB.Exec(`INSERT INTO sesiones (
		agente, proyecto_id, activa, estado, herramienta, host, heartbeat_at
	) VALUES (?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		"CodexRecover", proyectoID, 0, "cerrada", "codex-cli", "test-host")
	if err != nil {
		t.Fatalf("insert sesion legacy: %v", err)
	}
	sesionLegacyID, _ := resSesionLegacy.LastInsertId()

	resSesionTMUX, err := db.DB.Exec(`INSERT INTO sesiones (
		agente, proyecto_id, activa, estado, herramienta, host, heartbeat_at
	) VALUES (?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		"CodexRecover", proyectoID, 1, "activa", "codex-cli", "test-host")
	if err != nil {
		t.Fatalf("insert sesion tmux: %v", err)
	}
	sesionTMUXID, _ := resSesionTMUX.LastInsertId()

	now := time.Now().UTC()
	nowText := now.Format(time.RFC3339Nano)
	pid := int64(os.Getpid())

	runtimeLegacyID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "CodexRecover",
		ProyectoID:   &proyectoID,
		SesionID:     &sesionLegacyID,
		Connector:    "codex-cli",
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
		Model:        "gpt-4.1",
	})
	if err != nil {
		t.Fatalf("registrar runtime legacy: %v", err)
	}
	legacyMetaJSON, _ := json.Marshal(map[string]any{
		"driver":           "process_pty_cli",
		"rendered_command": "'/tmp/codex-perfiles/bin/codex-perfil' 'CuentaLegacy'",
		"working_dir":      filepath.Join(tmp, "legacy"),
	})
	if _, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, sesion_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexRecover", proyectoID, runtimeLegacyID, sesionLegacyID, "cli", "process", strconv.Itoa(os.Getpid()), string(legacyMetaJSON), now); err != nil {
		t.Fatalf("insert handle legacy: %v", err)
	}

	runtimeTMUXID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "CodexRecover",
		ProyectoID:   &proyectoID,
		SesionID:     &sesionTMUXID,
		Connector:    "codex-cli",
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
		Model:        "gpt-5.4",
	})
	if err != nil {
		t.Fatalf("registrar runtime tmux: %v", err)
	}

	runDir := filepath.Join(tmp, "runtime", "CodexRecover", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	writeJSON := func(path string, payload map[string]any) {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSON(manifestPath, map[string]any{
		"version":             1,
		"agent":               "CodexRecover",
		"driver":              "tmux_cli_session",
		"transport":           "tmux",
		"tmux_session":        "orq-codexrecover-1",
		"tmux_pane_id":        "%11",
		"status_path":         statusPath,
		"heartbeat_path":      heartbeatPath,
		"external_session_id": "sess-codexrecover",
	})
	writeJSON(statusPath, map[string]any{
		"state":      "running",
		"updated_at": nowText,
		"alive":      true,
		"child_pid":  os.Getpid(),
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"heartbeat_at": nowText,
		"started_at":   nowText,
		"child_pid":    os.Getpid(),
	})
	tmuxMetaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codexrecover-1",
		"tmux_pane_id":          "%11",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	resTMUX, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?, 'activo', ?, ?)`,
		"CodexRecover", proyectoID, "tmux", "session", "orq-codexrecover-1", string(tmuxMetaJSON), now)
	if err != nil {
		t.Fatalf("insert handle tmux: %v", err)
	}
	tmuxHandleID, _ := resTMUX.LastInsertId()

	sesionLegacy, err := db.GetSesionByID(sesionLegacyID)
	if err != nil {
		t.Fatalf("get sesion legacy: %v", err)
	}
	handle, runtime, err := runtimeRecuperacionSesionObjetivo(sesionLegacy)
	if err != nil {
		t.Fatalf("runtimeRecuperacionSesionObjetivo: %v", err)
	}
	if handle == nil || handle.ID != tmuxHandleID {
		t.Fatalf("deberia preferir el handle tmux canónico reciente: %+v", handle)
	}
	if runtime == nil || runtime.ID != runtimeTMUXID {
		t.Fatalf("deberia preferir el runtime principal canónico en vez del runtime de la sesión vieja: %+v", runtime)
	}
}

func TestRefrescarPresupuestoSesionObservadoDesdeSesionResuelveConectorPorSlug(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSlug", "programador"); err != nil {
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

	base := filepath.Join(tmp, "codex-perfiles")
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	wrapper := filepath.Join(base, "bin", "codex-perfil")
	script := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" != "CodexSlug" || "${2:-}" != "status-json" ]]; then
  exit 5
fi
cat <<'JSON'
{"source":"codex_profile_status","profile":"CodexSlug","checked_at":"2026-04-07T10:20:00Z","observed_at":"2026-04-07T10:19:30Z","session_path":"/tmp/sess-slug.jsonl","session_id":"sess-codexslug-budget","account":{"email":"slug@avidad.com","user":"Codex Slug","plan_type":"team"},"rate_limits":{"primary":{"used_percent":21,"left_percent":79,"window_minutes":300,"resets_at":"2026-04-07T13:20:00Z"},"secondary":{"used_percent":44,"left_percent":56,"window_minutes":10080,"resets_at":"2026-04-14T10:20:00Z"},"credits":null,"plan_type":"team"}}
JSON
`
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	conectorID, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-cli",
		Nombre:     "Codex CLI",
		Transporte: "cli",
		Comando:    wrapper,
		ArgsJSON:   `["{{agent}}"]`,
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:            "CodexSlug",
		ConectorID:        &conectorID,
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-codexslug-budget",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	sesion.ConectorID = nil
	sesion.ConectorSlug = "codex-cli"

	ok, err := refrescarPresupuestoSesionObservadoDesdeSesion(sesion)
	if err != nil {
		t.Fatalf("refrescar presupuesto desde sesion por slug: %v", err)
	}
	if !ok {
		t.Fatal("deberia refrescar presupuesto resolviendo el conector por slug")
	}
	ultimo, err := db.UltimoPresupuestoSesionPorFuente(sesion.ID, "codex_profile_status")
	if err != nil {
		t.Fatalf("ultimo presupuesto codex_profile_status: %v", err)
	}
	if ultimo == nil {
		t.Fatal("deberia persistir presupuesto codex_profile_status por slug")
	}
	agente, err := db.GetAgente("CodexSlug")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agente.CuentaEmail != "slug@avidad.com" || agente.PresupuestoFuente != "codex_profile_status" {
		t.Fatalf("refresh por slug inesperado: %+v", agente)
	}
}

func TestRuntimeHandleCanonicoDesdeSesionPresupuestoPrefiereLaSesionConcreta(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexBudgetSession", "programador"); err != nil {
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

	sesionA, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexBudgetSession",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador-a"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("sesion A: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE sesiones SET activa=0 WHERE id=?`, sesionA.ID); err != nil {
		t.Fatalf("cerrar sesion A: %v", err)
	}
	sesionB, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexBudgetSession",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador-b"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("sesion B: %v", err)
	}

	pid := int64(os.Getpid())
	runtimeAID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "CodexBudgetSession",
		SesionID:     &sesionA.ID,
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("runtime A: %v", err)
	}
	handleA, err := db.GetRuntimeHandleBySesionID(sesionA.ID)
	if err != nil || handleA == nil {
		t.Fatalf("get handle A: %+v err=%v", handleA, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles
		SET proyecto_id=?, runtime_id=?, transporte='tmux', handle_kind='session', handle_ref='orq-a/%1',
		    estado='activo', metadata_json=?, last_seen_at=?
		WHERE id=?`,
		proyectoID, runtimeAID,
		`{"driver":"tmux_cli_session","tmux_session":"orq-a","account_email":"a@avidad.com"}`,
		time.Now().UTC().Add(-time.Minute), handleA.ID); err != nil {
		t.Fatalf("actualizar handle A: %v", err)
	}
	handleAID := handleA.ID

	runtimeBID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "CodexBudgetSession",
		SesionID:     &sesionB.ID,
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("runtime B: %v", err)
	}
	handleB, err := db.GetRuntimeHandleBySesionID(sesionB.ID)
	if err != nil || handleB == nil {
		t.Fatalf("get handle B: %+v err=%v", handleB, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles
		SET proyecto_id=?, runtime_id=?, transporte='tmux', handle_kind='session', handle_ref='orq-b/%2',
		    estado='activo', metadata_json=?, last_seen_at=?
		WHERE id=?`,
		proyectoID, runtimeBID,
		`{"driver":"tmux_cli_session","tmux_session":"orq-b","account_email":"b@avidad.com"}`,
		time.Now().UTC(), handleB.ID); err != nil {
		t.Fatalf("actualizar handle B: %v", err)
	}

	handle := runtimeHandleCanonicoDesdeSesionPresupuesto(sesionA)
	if handle == nil || handle.ID != handleAID {
		t.Fatalf("deberia priorizar el handle de la sesion concreta, got=%+v", handle)
	}
}

func TestRuntimeHandleCanonicoParaPresupuestoPrefiereTMUXSobreLegacy(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex8", "programador"); err != nil {
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
		Agente:            "Codex8",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-codex8-budget",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	base := filepath.Join(tmp, "codex-perfiles")
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	wrapper := filepath.Join(base, "bin", "codex-perfil")
	script := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" != "Codex8" || "${2:-}" != "status-json" ]]; then
  exit 7
fi
cat <<'JSON'
{"source":"codex_profile_status","profile":"Codex8","checked_at":"2026-04-07T09:50:00Z","observed_at":"2026-04-07T09:49:30Z","session_path":"/tmp/sess-8.jsonl","session_id":"sess-codex8-budget","account":{"email":"berserk@avidad.com","user":"Berserk","plan_type":"team"},"rate_limits":{"primary":{"used_percent":24,"left_percent":76,"window_minutes":300,"resets_at":"2026-04-07T11:50:00Z"},"secondary":{"used_percent":74,"left_percent":26,"window_minutes":10080,"resets_at":"2026-04-09T18:35:00Z"},"credits":null,"plan_type":"team"}}
JSON
`
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}

	pid := int64(8008)
	runtimeLegacyID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "Codex8",
		ProyectoID:   &proyectoID,
		SesionID:     &sesion.ID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime legacy: %v", err)
	}
	legacyMetaJSON, _ := json.Marshal(map[string]any{
		"driver":           "process_pty_cli",
		"rendered_command": "codex-perfil Codex8 --model gpt-5.4",
		"working_dir":      filepath.Join(tmp, "legacy"),
	})
	resLegacy, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, CURRENT_TIMESTAMP)`,
		"Codex8", proyectoID, runtimeLegacyID, "cli", "process", strconv.Itoa(os.Getpid()), string(legacyMetaJSON))
	if err != nil {
		t.Fatalf("insert legacy handle: %v", err)
	}
	legacyHandleID, _ := resLegacy.LastInsertId()

	runDir := filepath.Join(tmp, "runtime", "Codex8", "run-1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	writeJSON := func(path string, payload map[string]any) {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	writeJSON(manifestPath, map[string]any{
		"version":                1,
		"agent":                  "Codex8",
		"driver":                 "tmux_cli_session",
		"transport":              "tmux",
		"profile":                "Codex8",
		"profile_status_wrapper": wrapper,
		"tmux_session":           "orq-codex8-budget",
		"tmux_pane_id":           "%8",
		"status_path":            statusPath,
		"heartbeat_path":         heartbeatPath,
		"external_session_id":    "sess-codex8-budget",
	})
	writeJSON(statusPath, map[string]any{
		"state":               "running",
		"updated_at":          now,
		"alive":               true,
		"child_pid":           8008,
		"external_session_id": "sess-codex8-budget",
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":               true,
		"heartbeat_at":        now,
		"child_pid":           8008,
		"external_session_id": "sess-codex8-budget",
	})

	runtimeTMUXID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "Codex8",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime tmux: %v", err)
	}
	tmuxMetaJSON, _ := json.Marshal(map[string]any{
		"driver":                 "tmux_cli_session",
		"tmux_session":           "orq-codex8-budget",
		"tmux_pane_id":           "%8",
		"worker_manifest_path":   manifestPath,
		"worker_status_path":     statusPath,
		"worker_heartbeat_path":  heartbeatPath,
		"profile_status_wrapper": wrapper,
		"profile_name":           "Codex8",
		"external_session_id":    "sess-codex8-budget",
		"rendered_command":       "wrapper opaco",
	})
	if _, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, CURRENT_TIMESTAMP)`,
		"Codex8", proyectoID, runtimeTMUXID, "tmux", "session", "orq-codex8-budget/%8", string(tmuxMetaJSON)); err != nil {
		t.Fatalf("insert tmux handle: %v", err)
	}

	legacyHandle, err := db.GetRuntimeHandle(legacyHandleID)
	if err != nil {
		t.Fatalf("get legacy handle: %v", err)
	}
	canonico := runtimeHandleCanonicoParaPresupuesto(legacyHandle)
	if canonico == nil {
		t.Fatal("deberia resolver un handle canónico para presupuesto")
	}
	if canonico.ID == legacyHandleID || strings.TrimSpace(canonico.HandleRef) != "orq-codex8-budget/%8" {
		t.Fatalf("deberia preferir el tmux canónico sobre el legacy: %+v", canonico)
	}
}

func TestRevalidarYVerificarAgenteDisponibleParaTrabajoRechazaAgenteAtascado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("CodexAtascado", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente atascado",
		Descripcion: "El worker esta vivo pero sin progreso real",
		ProyectoID:  &proyectoID,
		Modulo:      "runtime-adapters",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexAtascado"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexAtascado"); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}

	metaJSON := structuredWorkerMetadataAtascadoRevalidacionTest(t, time.Now().UTC(), time.Now().UTC().Add(-30*time.Minute))
	insertRuntimeHandleControlLifecycleTest(t, "CodexAtascado", metaJSON)

	err = revalidarYVerificarAgenteDisponibleParaTrabajo("CodexAtascado")
	if err == nil {
		t.Fatal("deberia rechazar un agente atascado")
	}
	if !strings.Contains(err.Error(), "atascado") {
		t.Fatalf("deberia rechazar por estado operativo atascado, got=%v", err)
	}
}

func TestRowDebeMigrarRuntimeLegacyATMUX(t *testing.T) {
	now := time.Now().UTC()
	handle := &db.RuntimeHandle{
		Agente:       "Codex8",
		Transporte:   "cli",
		HandleKind:   "process",
		Estado:       "activo",
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex8 --model gpt-5.4"}`,
	}
	row := agentesapp.Row{
		Agente: &db.Agente{
			Nombre:      "Codex8",
			EstadoCuota: "activo",
		},
		Handle:          handle,
		WorkerAlive:     true,
		WorkerHeartbeat: timePtr(now),
		WorkerState:     "running",
	}
	if !rowDebeMigrarRuntimeLegacyATMUX(row, now) {
		t.Fatal("un worker legacy process_pty_cli fresco de Codex deberia migrar a tmux")
	}
}

func TestRowDebeMigrarRuntimeLegacyATMUXTambienDetectaClaude(t *testing.T) {
	now := time.Now().UTC()
	handle := &db.RuntimeHandle{
		Agente:       "Claude1",
		Transporte:   "cli",
		HandleKind:   "process",
		Estado:       "activo",
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"claude-perfil Claude1 --model sonnet"}`,
	}
	row := agentesapp.Row{
		Agente: &db.Agente{
			Nombre:      "Claude1",
			EstadoCuota: "activo",
		},
		Handle:          handle,
		WorkerAlive:     true,
		WorkerHeartbeat: timePtr(now),
		WorkerState:     "running",
	}
	if !rowDebeMigrarRuntimeLegacyATMUX(row, now) {
		t.Fatal("un worker legacy process_pty_cli fresco de Claude deberia migrar a tmux")
	}
}

func structuredWorkerMetadataAtascadoRevalidacionTest(t *testing.T, heartbeatAt, progressAt time.Time) string {
	t.Helper()
	tmp := t.TempDir()
	heartbeatAt = heartbeatAt.UTC()
	progressAt = progressAt.UTC()
	workingDir := filepath.Join(tmp, "repo")
	runDir := filepath.Join(workingDir, ".orquesta-runtime", "codexatascado", "20260407-010203-000000001")
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	manifestPath := filepath.Join(tmp, "manifest.json")
	runtimeManifestPath := filepath.Join(runDir, "runtime.json")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir runDir: %v", err)
	}
	statusRaw := fmt.Sprintf(`{"state":"running","updated_at":"%s","alive":true,"child_pid":%d,"last_output_at":"%s","last_progress_at":"%s"}`,
		heartbeatAt.Format(time.RFC3339Nano),
		os.Getpid(),
		progressAt.Format(time.RFC3339Nano),
		progressAt.Format(time.RFC3339Nano),
	)
	heartbeatRaw := fmt.Sprintf(`{"alive":true,"heartbeat_at":"%s","started_at":"%s","child_pid":%d,"last_output_at":"%s","last_progress_at":"%s"}`,
		heartbeatAt.Format(time.RFC3339Nano),
		heartbeatAt.Add(-time.Minute).Format(time.RFC3339Nano),
		os.Getpid(),
		progressAt.Format(time.RFC3339Nano),
		progressAt.Format(time.RFC3339Nano),
	)
	manifestRaw := fmt.Sprintf(`{"version":1,"agent":"CodexAtascado","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexatascado-1","tmux_pane_id":"%%1","status_path":"%s","heartbeat_path":"%s","started_at":"%s","working_dir":"%s","child_pid":%d,"runtime_manifest_path":"%s"}`,
		statusPath,
		heartbeatPath,
		heartbeatAt.Add(-time.Minute).Format(time.RFC3339Nano),
		workingDir,
		os.Getpid(),
		runtimeManifestPath,
	)
	runtimeManifestRaw := fmt.Sprintf(`{"agente":"CodexAtascado","proyecto":"orquestador","working_dir":"%s","driver":"tmux_cli_session","transport":"tmux","pid":%d,"supervisor_ref":"%s","tmux_session":"orq-codexatascado-1","tmux_pane_id":"%%1","status_path":"%s","heartbeat_path":"%s","mailbox_delivery_mode":"session_resume","created_at":"%s"}`,
		workingDir,
		os.Getpid(),
		runDir,
		statusPath,
		heartbeatPath,
		heartbeatAt.Add(-time.Minute).Format(time.RFC3339Nano),
	)
	if err := os.WriteFile(statusPath, []byte(statusRaw), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(heartbeatRaw), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte(manifestRaw), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(runtimeManifestPath, []byte(runtimeManifestRaw+"\n"), 0o600); err != nil {
		t.Fatalf("write runtime manifest: %v", err)
	}
	return fmt.Sprintf(`{"driver":"tmux_cli_session","working_dir":"%s","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`,
		workingDir,
		manifestPath,
		statusPath,
		heartbeatPath,
	)
}

func TestRowDebeMigrarRuntimeLegacyATMUXNoMigraTMUXNiCuotaBloqueada(t *testing.T) {
	now := time.Now().UTC()
	tmuxRow := agentesapp.Row{
		Agente: &db.Agente{
			Nombre:      "Codex8",
			EstadoCuota: "activo",
		},
		Handle: &db.RuntimeHandle{
			Agente:       "Codex8",
			Transporte:   "cli",
			HandleKind:   "process",
			Estado:       "activo",
			MetadataJSON: `{"driver":"tmux_cli_session","rendered_command":"codex-perfil Codex8 --model gpt-5.4"}`,
		},
		WorkerAlive:     true,
		WorkerHeartbeat: timePtr(now),
		WorkerState:     "running",
	}
	if rowDebeMigrarRuntimeLegacyATMUX(tmuxRow, now) {
		t.Fatal("un worker tmux ya sano no deberia migrarse otra vez")
	}

	blockedRow := agentesapp.Row{
		Agente: &db.Agente{
			Nombre:      "Codex8",
			EstadoCuota: "enfriamiento",
		},
		Handle: &db.RuntimeHandle{
			Agente:       "Codex8",
			Transporte:   "cli",
			HandleKind:   "process",
			Estado:       "activo",
			MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex8 --model gpt-5.4"}`,
		},
		WorkerAlive:     true,
		WorkerHeartbeat: timePtr(now),
		WorkerState:     "running",
	}
	if rowDebeMigrarRuntimeLegacyATMUX(blockedRow, now) {
		t.Fatal("un agente bloqueado por cuota no deberia rearmarse a tmux")
	}
}

func TestErrorMigracionRuntimeLegacyTMUXIgnorable(t *testing.T) {
	casosTrue := []error{
		nil,
		fmt.Errorf("tmux kill-session: can't find session: orq-claude2-083122"),
		fmt.Errorf("tmux kill-session: no server running on /tmp/tmux-1000/default"),
		fmt.Errorf("failed to connect to server"),
	}
	for _, err := range casosTrue {
		if !errorMigracionRuntimeLegacyTMUXIgnorable(err) {
			t.Fatalf("deberia ignorar error=%v", err)
		}
	}
	if errorMigracionRuntimeLegacyTMUXIgnorable(fmt.Errorf("permiso denegado")) {
		t.Fatal("no deberia ignorar errores no relacionados con tmux obsoleto")
	}
}

func TestOrdenarHandlesParaPresupuestoVivoPrefiereTMUXSobreLegacy(t *testing.T) {
	legacy := &db.RuntimeHandle{
		ID:           1,
		Agente:       "Codex8",
		Estado:       "activo",
		Transporte:   "cli",
		HandleKind:   "process",
		MetadataJSON: `{"driver":"process_pty_cli","stdin_path":"/tmp/pty.stdin","log_path":"/tmp/pty.log"}`,
	}
	tmux := &db.RuntimeHandle{
		ID:           2,
		Agente:       "Codex8",
		Estado:       "activo",
		Transporte:   "tmux",
		HandleKind:   "process",
		MetadataJSON: `{"driver":"tmux_cli_session","log_path":"/tmp/tmux.log"}`,
	}
	handles := []*db.RuntimeHandle{legacy, tmux}

	ordenarHandlesParaPresupuestoVivo(handles)

	if handles[0] == nil || handles[0].ID != tmux.ID {
		t.Fatalf("deberia priorizar tmux sobre legacy para presupuesto vivo: %+v", handles)
	}
}

func TestListarRuntimeHandlesPresupuestoAgentePrefiereCanonicosRecientes(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexBudget", "programador"); err != nil {
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

	pid := int64(os.Getpid())
	runtimeLegacyID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "CodexBudget",
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
		"rendered_command": "codex-perfil CodexBudget --model gpt-5.4",
	})
	if _, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexBudget", proyectoID, runtimeLegacyID, "cli", "process", strconv.Itoa(os.Getpid()), string(legacyMetaJSON), time.Now().UTC()); err != nil {
		t.Fatalf("insert handle legacy: %v", err)
	}

	runDir := filepath.Join(tmp, "tmux-worker")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir tmux: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"CodexBudget","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codexbudget-1","tmux_pane_id":"%17","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	runtimeTMUXID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "CodexBudget",
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
		"tmux_session":          "orq-codexbudget-1",
		"tmux_pane_id":          "%17",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"rendered_command":      "codex-perfil CodexBudget",
	})
	resTMUX, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, ?)`,
		"CodexBudget", proyectoID, runtimeTMUXID, "tmux", "session", "orq-codexbudget-1", string(tmuxMetaJSON), time.Now().UTC())
	if err != nil {
		t.Fatalf("insert handle tmux: %v", err)
	}
	tmuxHandleID, _ := resTMUX.LastInsertId()

	handles, err := listarRuntimeHandlesPresupuestoAgente("CodexBudget")
	if err != nil {
		t.Fatalf("listarRuntimeHandlesPresupuestoAgente: %v", err)
	}
	if len(handles) != 1 {
		t.Fatalf("deberia quedarse solo con el canónico caliente, got=%d %+v", len(handles), handles)
	}
	if handles[0].ID != tmuxHandleID {
		t.Fatalf("deberia priorizar el handle tmux canónico, got=%+v", handles[0])
	}
}

func TestScoreHandlePresupuestoVivoPenalizaLegacySinOptIn(t *testing.T) {
	t.Setenv("ORQUESTA_ALLOW_LEGACY_LIVE_STATUS", "")
	legacy := &db.RuntimeHandle{
		ID:           1,
		Agente:       "Codex8",
		Estado:       "activo",
		Transporte:   "cli",
		HandleKind:   "process",
		MetadataJSON: `{"driver":"process_pty_cli","stdin_path":"/tmp/pty.stdin","log_path":"/tmp/pty.log"}`,
	}
	tmux := &db.RuntimeHandle{
		ID:           2,
		Agente:       "Codex8",
		Estado:       "activo",
		Transporte:   "tmux",
		HandleKind:   "process",
		MetadataJSON: `{"driver":"tmux_cli_session","log_path":"/tmp/tmux.log"}`,
	}
	if scoreHandlePresupuestoVivo(legacy) >= scoreHandlePresupuestoVivo(tmux) {
		t.Fatalf("legacy no deberia puntuar mejor que tmux sin opt-in explicito")
	}
}

func TestScoreHandlePresupuestoVivoNoPremiaProcessSiElWorkerEsTMUX(t *testing.T) {
	tmuxProcess := &db.RuntimeHandle{
		ID:           2,
		Agente:       "Codex8",
		Estado:       "activo",
		Transporte:   "tmux",
		HandleKind:   "process",
		MetadataJSON: `{"driver":"tmux_cli_session","log_path":"/tmp/tmux.log"}`,
	}
	tmuxSession := &db.RuntimeHandle{
		ID:           3,
		Agente:       "Codex8",
		Estado:       "activo",
		Transporte:   "tmux",
		HandleKind:   "session",
		MetadataJSON: `{"driver":"tmux_cli_session","log_path":"/tmp/tmux.log"}`,
	}
	if scoreHandlePresupuestoVivo(tmuxProcess) != scoreHandlePresupuestoVivo(tmuxSession) {
		t.Fatalf("tmux no deberia puntuar distinto solo por arrastrar handle_kind=process")
	}
}

func TestRefrescarPresupuestoSesionObservadoHandleOmiteLegacyTMUXPorDefecto(t *testing.T) {
	prepararDBTemporalCmd(t)
	t.Setenv("ORQUESTA_ALLOW_LEGACY_LIVE_STATUS", "")
	t.Setenv("PATH", t.TempDir())
	handle := &db.RuntimeHandle{
		ID:           1,
		Agente:       "Codex8",
		Estado:       "activo",
		Transporte:   "cli",
		HandleKind:   "process",
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex8 --model gpt-5.4","stdin_path":"/tmp/pty.stdin"}`,
	}

	refrescado, agente, err := refrescarPresupuestoSesionObservadoHandle(handle)
	if err != nil {
		t.Fatalf("refrescar presupuesto legacy: %v", err)
	}
	if agente != "Codex8" {
		t.Fatalf("agente=%q", agente)
	}
	_ = refrescado
}

func TestObjetivoProcesoPresupuestoDesdeHandleOmitePIDParaLegacyYTMUX(t *testing.T) {
	legacy := &db.RuntimeHandle{
		Agente:       "Codex8",
		Transporte:   "cli",
		HandleKind:   "process",
		HandleRef:    strconv.Itoa(os.Getpid()),
		Estado:       "activo",
		MetadataJSON: `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex8 --model gpt-5.4"}`,
	}
	obj := objetivoProcesoPresupuestoDesdeHandle(legacy, `{"profile_name":"Codex8"}`)
	if obj.PID != nil {
		t.Fatalf("legacy tmux-preferred no deberia inyectar PID: %+v", obj)
	}

	legacyGeneric := &db.RuntimeHandle{
		Agente:       "WorkerLegacy",
		Transporte:   "cli",
		HandleKind:   "process",
		HandleRef:    strconv.Itoa(os.Getpid()),
		Estado:       "activo",
		MetadataJSON: `{"driver":"process_pty_cli","stdin_path":"/tmp/pty.stdin"}`,
	}
	obj = objetivoProcesoPresupuestoDesdeHandle(legacyGeneric, `{"profile_name":"WorkerLegacy"}`)
	if obj.PID != nil {
		t.Fatalf("cualquier process_pty_cli legacy no deberia inyectar PID: %+v", obj)
	}

	tmux := &db.RuntimeHandle{
		Agente:       "Codex8",
		Transporte:   "tmux",
		HandleKind:   "session",
		HandleRef:    "orq-codex8/%8",
		Estado:       "activo",
		MetadataJSON: `{"driver":"tmux_cli_session","tmux_session":"orq-codex8"}`,
	}
	obj = objetivoProcesoPresupuestoDesdeHandle(tmux, `{"profile_name":"Codex8"}`)
	if obj.PID != nil {
		t.Fatalf("tmux canónico no deberia inyectar PID: %+v", obj)
	}

	plain := &db.RuntimeHandle{
		Agente:       "Worker1",
		Transporte:   "cli",
		HandleKind:   "process",
		HandleRef:    strconv.Itoa(os.Getpid()),
		Estado:       "activo",
		MetadataJSON: `{"driver":"local_process"}`,
	}
	obj = objetivoProcesoPresupuestoDesdeHandle(plain, `{"profile_name":"Worker1"}`)
	if obj.PID == nil || *obj.PID != int64(os.Getpid()) {
		t.Fatalf("un proceso no legacy deberia conservar PID: %+v", obj)
	}
}

func TestRefrescarPresupuestoHandleDesdeObjetivoHeredaPerfilDesdeSesionTMUX(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexProfile", "programador"); err != nil {
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

	base := filepath.Join(tmp, "codex-perfiles")
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	wrapper := filepath.Join(base, "bin", "codex-perfil")
	script := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" != "CodexProfile" || "${2:-}" != "status-json" ]]; then
  exit 6
fi
cat <<'JSON'
{"source":"codex_profile_status","profile":"CodexProfile","checked_at":"2026-04-07T11:10:00Z","observed_at":"2026-04-07T11:09:30Z","session_path":"/tmp/sess-profile.jsonl","session_id":"sess-codexprofile-budget","account":{"email":"profile@avidad.com","user":"Codex Profile","plan_type":"team"},"rate_limits":{"primary":{"used_percent":11,"left_percent":89,"window_minutes":300,"resets_at":"2026-04-07T13:10:00Z"},"secondary":{"used_percent":33,"left_percent":67,"window_minutes":10080,"resets_at":"2026-04-14T11:10:00Z"},"credits":null,"plan_type":"team"}}
JSON
`
	if err := os.WriteFile(wrapper, []byte(script), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	conectorID, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-cli",
		Nombre:     "Codex CLI",
		Transporte: "cli",
		Comando:    wrapper,
		ArgsJSON:   `["{{agent}}"]`,
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:            "CodexProfile",
		ConectorID:        &conectorID,
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-codexprofile-budget",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	pid := int64(9101)
	runtimeID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "CodexProfile",
		ProyectoID:   &proyectoID,
		SesionID:     &sesion.ID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("crear runtime: %v", err)
	}
	handleMetaJSON, _ := json.Marshal(map[string]any{
		"driver":              "tmux_cli_session",
		"tmux_session":        "orq-codexprofile-1",
		"tmux_pane_id":        "%4",
		"external_session_id": "sess-codexprofile-budget",
	})
	res, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		"CodexProfile", proyectoID, runtimeID, "tmux", "session", "orq-codexprofile-1/%4", "activo", string(handleMetaJSON))
	if err != nil {
		t.Fatalf("insert handle: %v", err)
	}
	handleID, _ := res.LastInsertId()
	handle, err := db.GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}
	metaJSON := metadataPresupuestoDesdeHandle(handle)
	if !strings.Contains(metaJSON, wrapper) || !strings.Contains(metaJSON, "CodexProfile") {
		t.Fatalf("metadata enriquecida inesperada: %s", metaJSON)
	}
	observed, err := controlruntime.ObserveCodexProfileStatus(controlruntime.ObjetivoProceso{MetadataJSON: metaJSON})
	if err != nil {
		t.Fatalf("observe codex profile status: %v", err)
	}
	if observed == nil || strings.TrimSpace(observed.AccountEmail) != "profile@avidad.com" {
		t.Fatalf("observacion codex_profile_status inesperada: %+v", observed)
	}
	if observed.ObservedAt.IsZero() {
		t.Fatalf("observacion inicial sin observed_at: %+v", observed)
	}
	synced, _, _, err := db.SincronizarRuntimeHandleSupervisado(handle, nil, "test_budget_refresh")
	if err != nil {
		t.Fatalf("sync handle: %v", err)
	}
	syncedMetaJSON := mergeBudgetMetadataJSON(metadataPresupuestoDesdeHandle(synced), metaJSON)
	obj := controlruntime.ObjetivoProceso{
		PID:          int64PtrFromHandleRef(synced.HandleKind, synced.HandleRef),
		HandleKind:   strings.TrimSpace(synced.HandleKind),
		HandleRef:    strings.TrimSpace(synced.HandleRef),
		MetadataJSON: syncedMetaJSON,
	}
	observed, err = controlruntime.ObserveCodexProfileStatus(obj)
	if err != nil {
		t.Fatalf("observe codex profile status tras sync: %v", err)
	}
	if observed == nil || strings.TrimSpace(observed.AccountEmail) != "profile@avidad.com" {
		t.Fatalf("observacion tras sync inesperada: %+v metadata=%s", observed, syncedMetaJSON)
	}
	if observed.ObservedAt.IsZero() {
		t.Fatalf("observacion tras sync sin observed_at: %+v", observed)
	}
	okViaSesion, err := refrescarPresupuestoSesionObservadoDesdeSesion(sesion)
	if err != nil {
		t.Fatalf("refresh directo via sesion: %v", err)
	}
	if !okViaSesion {
		t.Fatal("la sesion canonica deberia refrescar presupuesto por si sola")
	}
	ok, err := refrescarPresupuestoHandleDesdeObjetivo(handle, "CodexProfile", controlruntime.ObjetivoProceso{MetadataJSON: syncedMetaJSON})
	if err != nil {
		t.Fatalf("refrescar presupuesto desde objetivo broker-first: %v", err)
	}
	if !ok {
		t.Fatal("el helper broker-first deberia persistir presupuesto con metadata enriquecida")
	}
	ultimo, err := db.UltimoPresupuestoSesionPorFuente(sesion.ID, "codex_profile_status")
	if err != nil {
		t.Fatalf("ultimo presupuesto: %v", err)
	}
	if ultimo == nil {
		t.Fatal("deberia persistir presupuesto por codex_profile_status")
	}
	agenteDB, err := db.GetAgente("CodexProfile")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agenteDB.CuentaEmail != "profile@avidad.com" || agenteDB.PresupuestoFuente != "codex_profile_status" {
		t.Fatalf("refresh desde handle tmux con sesion enriquecida inesperado: %+v", agenteDB)
	}
}

func TestProcesarSupervisionAutonomaBatchArrancaSupervisorPreferidoSinSesion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("procesar supervision: %v", err)
	}
	if n != 1 {
		t.Fatalf("debería contar supervisión en el mismo ciclo de arranque, got=%d", n)
	}

	asignacion, err := db.GetAsignacionActivaAgente("CodexSupervisor")
	if err != nil {
		agente := "CodexSupervisor"
		estado := "pendiente"
		orders, _ := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
		t.Fatalf("get asignacion activa: %v; orders=%+v", err, orders)
	}
	if asignacion.ProyectoID != proyectoID {
		t.Fatalf("asignacion activa inesperada: %+v", asignacion)
	}

	agente := "CodexSupervisor"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var hasStart, hasNudge bool
	for _, order := range orders {
		if order == nil {
			continue
		}
		if order.Tipo == "start" {
			hasStart = true
		}
		if order.Tipo == "nudge" && strings.Contains(order.PayloadJSON, `"accion":"supervisar_proyecto"`) {
			hasNudge = true
		}
	}
	if !hasStart || !hasNudge {
		t.Fatalf("debería encolar start y nudge de supervisión: %+v", orders)
	}
}

func TestProcesarSupervisionAutonomaBatchAceptaSupervisorPreferidoCaseInsensitive(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-case-insensitive",
		Nombre:  "Orquestador Case Insensitive",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "codexsupervisor",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("procesar supervision: %v", err)
	}
	if n != 1 {
		t.Fatalf("debería contar supervisión con nombre case-insensitive, got=%d", n)
	}

	asignacion, err := db.GetAsignacionActivaAgente("CodexSupervisor")
	if err != nil {
		t.Fatalf("get asignacion activa: %v", err)
	}
	if asignacion.ProyectoID != proyectoID {
		t.Fatalf("asignacion activa inesperada: %+v", asignacion)
	}

	agente := "CodexSupervisor"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var hasStart, hasNudge bool
	for _, order := range orders {
		if order == nil {
			continue
		}
		if order.Tipo == "start" {
			hasStart = true
		}
		if order.Tipo == "nudge" && strings.Contains(order.PayloadJSON, `"accion":"supervisar_proyecto"`) {
			hasNudge = true
		}
	}
	if !hasStart || !hasNudge {
		t.Fatalf("debería encolar start y nudge al supervisor canónico: %+v", orders)
	}
}

func TestProcesarSupervisionAutonomaBatchNoRepiteSupervisionPeriodicaConSupervisorOperativo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("primer ciclo supervision: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba bootstrap inicial de supervision, got=%d", n)
	}

	agente := "CodexSupervisor"
	completada := "completada"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &completada})
	if err != nil {
		t.Fatalf("listar orders completadas: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no esperaba completadas antes de materializar la prueba: %+v", orders)
	}

	pendiente := "pendiente"
	orders, err = db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar orders pendientes: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("esperaba una sola order inicial, got=%d", len(orders))
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders
		SET estado='completada',
		    started_at=datetime('now','-10 minutes'),
		    finished_at=datetime('now','-10 minutes'),
		    updated_at=datetime('now','-10 minutes')
		WHERE id=?`, orders[0].ID); err != nil {
		t.Fatalf("cerrar order inicial: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE proyectos_autonomia
		SET last_supervision_at=datetime('now','-10 minutes')
		WHERE proyecto_id=?`, proyectoID); err != nil {
		t.Fatalf("forzar supervision vencida: %v", err)
	}

	n, err = procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("segundo ciclo supervision: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia repetir supervision periodica con supervisor operativo, got=%d", n)
	}

	orders, err = db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar todas las orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("no deberia crear una segunda order periodica: %+v", orders)
	}
}

func TestProcesarSupervisionAutonomaBatchNoRepiteSupervisionPeriodicaSiYaEmitioNudgeReciente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	if _, err := db.DB.Exec(`UPDATE proyectos_autonomia
		SET last_supervision_at=datetime('now','-10 minutes')
		WHERE proyecto_id=?`, proyectoID); err != nil {
		t.Fatalf("forzar supervision vencida: %v", err)
	}

	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	if proyecto == nil {
		t.Fatal("proyecto nil")
	}
	encolada, err := encolarNudgeAutonomiaDetallado("CodexSupervisor", proyecto, "supervisar_proyecto", "reciente", "instruction reciente", map[string]any{
		"supervision_cycle_kind": "supervision",
	})
	if err != nil {
		t.Fatalf("encolar nudge reciente: %v", err)
	}
	if !encolada {
		t.Fatal("faltaba nudge reciente")
	}
	agente := "CodexSupervisor"
	estadoPendiente := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("listar runtime orders pendientes: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("faltaba order pendiente inicial: %+v", orders)
	}
	orderID := orders[0].ID
	if _, err := db.DB.Exec(`UPDATE runtime_orders
		SET estado='completada',
		    started_at=datetime('now','-2 minutes'),
		    finished_at=datetime('now','-2 minutes'),
		    updated_at=datetime('now','-2 minutes')
		WHERE id=?`, orderID); err != nil {
		t.Fatalf("cerrar nudge reciente: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("segundo ciclo supervision: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia repetir supervision periodica si ya emitio un nudge reciente, got=%d", n)
	}

	orders, err = db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	var supervisionNudges int
	for _, order := range orders {
		if order != nil && order.Tipo == "nudge" && strings.Contains(order.PayloadJSON, `"accion":"supervisar_proyecto"`) {
			supervisionNudges++
		}
	}
	if supervisionNudges != 1 {
		t.Fatalf("no deberia crear un segundo nudge de supervision reciente: %+v", orders)
	}
}

func TestProcesarSupervisionAutonomaBatchNoRepiteSupervisionSiSupervisorYaTieneTrabajoActivo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCreateTasks:      true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	taskID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Supervision ya en curso",
		Descripcion: "Auditoria real del proyecto",
		ProyectoID:  &proyectoID,
		Modulo:      "autonomia",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(taskID, "CodexSupervisor"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(taskID, "CodexSupervisor"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE proyectos_autonomia
		SET last_supervision_at=datetime('now','-10 minutes')
		WHERE proyecto_id=?`, proyectoID); err != nil {
		t.Fatalf("forzar supervision vencida: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("procesar supervision: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia repetir supervision si el supervisor ya tiene trabajo activo, got=%d", n)
	}

	agente := "CodexSupervisor"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "nudge" && strings.Contains(order.PayloadJSON, `"accion":"supervisar_proyecto"`) {
			t.Fatalf("no deberia crear nudge de supervision sobre supervisor con trabajo activo: %+v", orders)
		}
	}
}

func TestProcesarSupervisionAutonomaBatchGeneraBacklogInicialSiAutoCreateTasks(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCreateTasks:      true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("procesar supervision: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 supervision, got=%d", n)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) < 5 {
		t.Fatalf("debería generar un backlog inicial completo, tareas=%+v", tareas)
	}
	var briefing, investigacion, arquitectura bool
	var briefingActiva bool
	var backlog int
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Titulo {
		case "Definir briefing funcional de Orquestador":
			briefing = true
			if tarea.Estado == db.TareaEnProgreso && tarea.Agente != nil && *tarea.Agente == "CodexSupervisor" {
				briefingActiva = true
			}
		case "Revisar catálogo y referencias de Orquestador",
			"Revisar catalogo y referencias de Orquestador":
			investigacion = true
		case "Cerrar arquitectura hexagonal y modular de Orquestador",
			"Cerrar arquitectura base de Orquestador":
			arquitectura = true
		case "Autonomía: revisar backlog y abrir siguiente frente útil":
			t.Fatalf("no debería crear tarea semilla cuando el proyecto aún no tiene backlog: %+v", tarea)
		}
		if tarea.Estado == db.TareaBacklog {
			backlog++
		}
	}
	if !briefing || !investigacion || !arquitectura {
		t.Fatalf("faltan tareas base del plan inicial, tareas=%+v", tareas)
	}
	if !briefingActiva {
		t.Fatalf("el briefing debería arrancarse en el supervisor para poner en marcha el plan, tareas=%+v", tareas)
	}
	if backlog == 0 {
		t.Fatalf("el backlog inicial debería dejar trabajo dependiente en backlog, tareas=%+v", tareas)
	}

	kind := "supervision"
	cycles, err := db.ListarAutonomiaCiclos(db.FiltroAutonomiaCiclos{ProyectoID: &proyectoID, Kind: &kind, Limit: 10})
	if err != nil {
		t.Fatalf("listar ciclos: %v", err)
	}
	if len(cycles) != 1 || !strings.Contains(cycles[0].DecisionJSON, `"auto_created_plan":true`) {
		t.Fatalf("ciclo supervision sin traza de auto_create_tasks: %+v", cycles)
	}
}

func TestProcesarSupervisionAutonomaBatchReabreFrenteSiSoloHayHistoricoCerrado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCreateTasks:      true,
		AutoCloseProject:     false,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente anterior",
		Descripcion: "Ya completado",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea historica: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexSupervisor"); err != nil {
		t.Fatalf("tomar tarea historica: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexSupervisor"); err != nil {
		t.Fatalf("iniciar tarea historica: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexSupervisor", "hecho"); err != nil {
		t.Fatalf("completar tarea historica: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("procesar supervision: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 supervision con reapertura de frente, got=%d", n)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) != 2 {
		t.Fatalf("debería conservar histórico y abrir un frente nuevo, tareas=%+v", tareas)
	}
	var abiertas int
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		if tarea.Estado == db.TareaEnProgreso && tarea.Titulo == "Autonomía: revisar backlog y abrir siguiente frente útil" {
			abiertas++
		}
	}
	if abiertas != 1 {
		t.Fatalf("debería abrir exactamente una nueva tarea semilla en progreso, tareas=%+v", tareas)
	}
}

func TestProcesarSupervisionAutonomaBatchNoCreaTareaSemillaSiAutoCreateTasksDesactivado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		AutoCreateTasks:      false,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	if _, err := procesarSupervisionAutonomaBatch(); err != nil {
		t.Fatalf("procesar supervision: %v", err)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) != 0 {
		t.Fatalf("no debería crear tarea semilla con auto_create_tasks desactivado: %+v", tareas)
	}
}

func TestProcesarSupervisionAutonomaBatchCreaMicrocicloPremiumInicialConAutoCreateTasksDesactivado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"estado":"microrefactor_loop_activo"}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       false,
		AutoCreateTasks:      false,
		AutoCloseProject:     false,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       1,
		Prioridad:        100,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("procesar supervision: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 supervision con continuidad premium inicial, got=%d", n)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) != 1 {
		t.Fatalf("deberia crear una unica microtarea premium inicial, tareas=%+v", tareas)
	}
	if !strings.Contains(strings.TrimSpace(tareas[0].Notas), microcicloRefactorNotasTag) {
		t.Fatalf("la tarea inicial deberia ser microciclo premium, tarea=%+v", tareas[0])
	}
}

func TestProcesarSupervisionAutonomaBatchEscalaAFrentePremiumMayorTrasAgotarMicrociclo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"estado":"microrefactor_loop_activo"}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       false,
		AutoCreateTasks:      false,
		AutoCloseProject:     false,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       1,
		Prioridad:        100,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      microcicloRefactorTituloDefault,
		Descripcion: "Frente premium anterior",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea historica: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexSupervisor"); err != nil {
		t.Fatalf("tomar tarea historica: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexSupervisor"); err != nil {
		t.Fatalf("iniciar tarea historica: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexSupervisor", "hecho"); err != nil {
		t.Fatalf("completar tarea historica: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarSupervisionAutonomaBatch()
	if err != nil {
		t.Fatalf("procesar supervision: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 supervision con continuidad premium, got=%d", n)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) != 2 {
		t.Fatalf("deberia conservar historico y abrir un frente premium nuevo, tareas=%+v", tareas)
	}
	var abierta *db.Tarea
	for _, tarea := range tareas {
		if tarea == nil || tarea.ID == tareaID {
			continue
		}
		if tarea.Estado == db.TareaEnProgreso {
			abierta = tarea
		}
	}
	if abierta == nil {
		t.Fatalf("deberia abrir una nueva tarea premium en progreso, tareas=%+v", tareas)
	}
	if strings.Contains(strings.TrimSpace(abierta.Notas), microcicloRefactorNotasTag) {
		t.Fatalf("tras agotar el microciclo no deberia reciclar otra microtarea, tarea=%+v", abierta)
	}
	if !strings.Contains(strings.TrimSpace(abierta.Notas), "autonomia:premium_frontier") {
		t.Fatalf("deberia abrir un frente premium mayor canonico, tarea=%+v", abierta)
	}
	if !strings.Contains(strings.TrimSpace(abierta.Titulo), "siguiente frente mayor útil") {
		t.Fatalf("la tarea nueva deberia reflejar escalado a frente mayor, tarea=%+v", abierta)
	}
	if !strings.Contains(abierta.Descripcion, "WRITE_SET") || !strings.Contains(abierta.Descripcion, "crea exactamente una tarea nueva") {
		t.Fatalf("la semilla premium deberia exigir contrato explicito y secuencia operativa, tarea=%+v", abierta)
	}
	if abierta.Agente == nil || strings.TrimSpace(*abierta.Agente) != "CodexSupervisor" {
		t.Fatalf("la nueva tarea premium deberia quedar asignada al supervisor, tarea=%+v", abierta)
	}
}

func TestProcesarReviewGatesBatchCreaGateYEncolaRevision(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.RegistrarAgente("CodexA", "admin"); err != nil {
		t.Fatalf("registrar agente alternativo: %v", err)
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexReviewer", proyectoID, "review"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if err := db.ActivarAsignacion("CodexA", proyectoID, "review alterna"); err != nil {
		t.Fatalf("activar asignacion alterna: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexReviewer",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexA",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion alterna: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Cerrar autonomia",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexReviewer", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	worktree, err := db.CoordinationWorktreeSQLRepository{}.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		TaskID:    &tareaID,
		Agent:     "CodexReviewer",
		Name:      "orquestador-codexreviewer-t1",
		Path:      filepath.Join(tmp, "wt-review"),
		Branch:    "orq/orquestador/CodexReviewer/t1",
		BaseRef:   "master",
		State:     coordinacion.WorktreeActive,
		Reason:    "review_test",
	})
	if err != nil {
		t.Fatalf("crear worktree activa: %v", err)
	}

	n, err := procesarReviewGatesBatch()
	if err != nil {
		t.Fatalf("procesar review gates: %v", err)
	}
	if n != 2 {
		t.Fatalf("esperaba 2 acciones de review (gate + nudge), got=%d", n)
	}

	gates, err := db.ListarReviewGates(db.FiltroReviewGates{ProyectoID: &proyectoID, Limit: 10})
	if err != nil {
		t.Fatalf("listar review gates: %v", err)
	}
	if len(gates) != 1 {
		t.Fatalf("esperaba 1 review gate, got=%d", len(gates))
	}
	if gates[0].ReviewerAgente != "CodexReviewer" || gates[0].Estado != db.ReviewGateEnRevision {
		t.Fatalf("review gate inesperado: %+v", gates[0])
	}
	if gates[0].TareaID == nil || *gates[0].TareaID != tareaID {
		t.Fatalf("el gate debería quedar ligado a la tarea revisada, gate=%+v", gates[0])
	}
	if gates[0].WorktreeID == nil || *gates[0].WorktreeID != worktree.ID {
		t.Fatalf("el gate debería quedar ligado al worktree activo, gate=%+v", gates[0])
	}
	agente := "CodexReviewer"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" || !strings.Contains(orders[0].PayloadJSON, `"accion":"ejecutar_review_gate"`) {
		t.Fatalf("review nudge inesperado: %+v", orders)
	}
	policy, err := db.GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto autonomia: %v", err)
	}
	if policy.EstadoAutonomia != db.AutonomiaProyectoEsperandoReview {
		t.Fatalf("estado autonomia inesperado: %s", policy.EstadoAutonomia)
	}
}

func TestProcesarReviewGatesBatchCreaSolicitudMergeTrasGateAprobadoReactivaProyecto(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Cerrar frente para merge",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexReviewer", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	worktree, err := db.CoordinationWorktreeSQLRepository{}.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		TaskID:    &tareaID,
		Agent:     "CodexReviewer",
		Name:      "orquestador-codexreviewer-merge",
		Path:      filepath.Join(tmp, "wt-merge"),
		Branch:    "orq/orquestador/CodexReviewer/t-merge",
		BaseRef:   "master",
		State:     coordinacion.WorktreeActive,
		Reason:    "merge_test",
	})
	if err != nil {
		t.Fatalf("crear worktree activa: %v", err)
	}
	now := time.Now().UTC()
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:     &proyectoID,
		TareaID:        &tareaID,
		WorktreeID:     &worktree.ID,
		RequestedBy:    "orquesta",
		ReviewerAgente: "CodexReviewer",
		Estado:         db.ReviewGateAprobado,
		ResolvedAt:     &now,
	}); err != nil {
		t.Fatalf("crear review gate aprobado: %v", err)
	}

	n, err := procesarReviewGatesBatch()
	if err != nil {
		t.Fatalf("procesar review gates: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 acción de auto-merge, got=%d", n)
	}
	merges, err := db.ListarGitMerges(&proyectoID, "")
	if err != nil {
		t.Fatalf("listar merges: %v", err)
	}
	if len(merges) != 1 {
		t.Fatalf("debería crear una única solicitud de merge, merges=%+v", merges)
	}
	if merges[0].SourceBranch != "orq/orquestador/CodexReviewer/t-merge" || merges[0].TargetBranch != "master" {
		t.Fatalf("solicitud de merge inesperada: %+v", merges[0])
	}
	if merges[0].Estado != "aprobado" {
		t.Fatalf("la solicitud automática debería quedar aprobada, merge=%+v", merges[0])
	}
	if !strings.Contains(merges[0].Notas, "review gate") {
		t.Fatalf("la solicitud de merge debería dejar trazabilidad del gate, merge=%+v", merges[0])
	}

	n, err = procesarReviewGatesBatch()
	if err != nil {
		t.Fatalf("reprocesar review gates: %v", err)
	}
	if n != 0 {
		t.Fatalf("no debería duplicar la solicitud de merge, got=%d", n)
	}
	merges, err = db.ListarGitMerges(&proyectoID, "")
	if err != nil {
		t.Fatalf("listar merges tras reproceso: %v", err)
	}
	if len(merges) != 1 {
		t.Fatalf("no debería duplicar merges, merges=%+v", merges)
	}
}

func TestProcesarReviewGatesBatchArrancaReviewerPreferidoSinSesion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Cerrar autonomia",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexReviewer", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}

	n, err := procesarReviewGatesBatch()
	if err != nil {
		t.Fatalf("procesar review gates: %v", err)
	}
	if n != 0 {
		t.Fatalf("no debería contar review hasta que el reviewer arranque, got=%d", n)
	}

	asignacion, err := db.GetAsignacionActivaAgente("CodexReviewer")
	if err != nil {
		agente := "CodexReviewer"
		estado := "pendiente"
		orders, _ := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
		t.Fatalf("get asignacion activa: %v; orders=%+v", err, orders)
	}
	if asignacion.ProyectoID != proyectoID {
		t.Fatalf("asignacion activa inesperada: %+v", asignacion)
	}

	agente := "CodexReviewer"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("debería encolar start para reviewer preferido: %+v", orders)
	}
}

func TestProcesarReviewGatesBatchReabreCorreccionTrasCambiosPedidos(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
		t.Fatalf("registrar reviewer: %v", err)
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoEsperandoReview,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if err := db.ActivarAsignacion("CodexReviewer", proyectoID, "review"); err != nil {
		t.Fatalf("activar asignacion reviewer: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexReviewer",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion reviewer: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexReviewer", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	gateID, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:     &proyectoID,
		RequestedBy:    "CodexReviewer",
		ReviewerAgente: "CodexReviewer",
		Estado:         db.ReviewGateCambiosPed,
		SeverityMax:    "high",
		FindingsJSON:   `{"findings":["faltan tests","ajustar arquitectura"]}`,
	})
	if err != nil {
		t.Fatalf("crear review gate: %v", err)
	}

	n, err := procesarReviewGatesBatch()
	if err != nil {
		t.Fatalf("procesar review gates: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 acción de corrección, got=%d", n)
	}

	agente := "CodexSupervisor"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" || !strings.Contains(orders[0].PayloadJSON, `"accion":"resolver_review_feedback"`) {
		todas, _ := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{ProyectoID: &proyectoID, Estado: &estado})
		t.Fatalf("nudge de corrección inesperado: agente=%+v todas=%+v", orders, todas)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"gate_id":`) || !strings.Contains(orders[0].PayloadJSON, `"review_findings":"`) {
		t.Fatalf("payload de corrección incompleto: %s", orders[0].PayloadJSON)
	}
	gate, err := db.GetReviewGate(gateID)
	if err != nil {
		t.Fatalf("get review gate: %v", err)
	}
	if gate == nil || gate.Estado != db.ReviewGateEnRevision {
		t.Fatalf("gate debería reabrirse a en_revision: %+v", gate)
	}
	policy, err := db.GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto autonomia: %v", err)
	}
	if policy.EstadoAutonomia != db.AutonomiaProyectoActiva {
		t.Fatalf("estado autonomia debería volver a activo, got=%s", policy.EstadoAutonomia)
	}
	kind := "review_feedback"
	cycles, err := supervisionService.ListCycles("orquestador", &kind, 5)
	if err != nil {
		t.Fatalf("listar ciclos review_feedback: %v", err)
	}
	if len(cycles) != 1 || cycles[0] == nil || cycles[0].Agente != "CodexSupervisor" {
		t.Fatalf("ciclo review_feedback inesperado: %+v", cycles)
	}
}

func TestProcesarReviewGatesBatchEncolaCorreccionTrasCambiosPedidos(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
		t.Fatalf("registrar reviewer: %v", err)
	}
	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoEsperandoReview,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexReviewer", proyectoID, "review"); err != nil {
		t.Fatalf("activar asignacion reviewer: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "correccion"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexReviewer",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion reviewer: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexReviewer", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:     &proyectoID,
		RequestedBy:    "orquesta",
		ReviewerAgente: "CodexReviewer",
		Estado:         db.ReviewGateCambiosPed,
		SeverityMax:    "high",
		FindingsJSON:   `{"findings":["falta cerrar el refactor server-first"]}`,
	}); err != nil {
		t.Fatalf("crear review gate cambios_pedidos: %v", err)
	}

	n, err := procesarReviewGatesBatch()
	if err != nil {
		t.Fatalf("procesar review gates: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 nudge de correccion, got=%d", n)
	}

	agente := "CodexSupervisor"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" || !strings.Contains(orders[0].PayloadJSON, `"accion":"resolver_review_feedback"`) {
		todas, _ := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{ProyectoID: &proyectoID, Estado: &estado})
		t.Fatalf("nudge de correccion inesperado: agente=%+v todas=%+v", orders, todas)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"gate_id":1`) {
		t.Fatalf("payload de correccion sin gate_id: %s", orders[0].PayloadJSON)
	}
	kind := "review_feedback"
	cycles, err := db.ListarAutonomiaCiclos(db.FiltroAutonomiaCiclos{ProyectoID: &proyectoID, Kind: &kind, Limit: 10})
	if err != nil {
		t.Fatalf("listar ciclos review_feedback: %v", err)
	}
	if len(cycles) != 1 || cycles[0].Agente != "CodexSupervisor" {
		t.Fatalf("ciclo review_feedback inesperado: %+v", cycles)
	}
	gates, err := db.ListarReviewGates(db.FiltroReviewGates{ProyectoID: &proyectoID, Limit: 10})
	if err != nil {
		t.Fatalf("listar gates: %v", err)
	}
	if len(gates) != 1 || gates[0].Estado != db.ReviewGateEnRevision {
		t.Fatalf("gate deberia reabrirse en_revision: %+v", gates)
	}
	policy, err := db.GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto autonomia: %v", err)
	}
	if policy.EstadoAutonomia != db.AutonomiaProyectoActiva {
		t.Fatalf("el proyecto deberia reactivarse para corregir, got=%s", policy.EstadoAutonomia)
	}
}

func TestProcesarReviewGatesBatchCreaSolicitudMergeTrasGateAprobado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
		t.Fatalf("registrar reviewer: %v", err)
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoEsperandoReview,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexReviewer", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	worktree, err := db.CoordinationWorktreeSQLRepository{}.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		TaskID:    &tareaID,
		Agent:     "CodexReviewer",
		Name:      "orquestador-codexreviewer-t3",
		Path:      filepath.Join(tmp, "wt-approved"),
		Branch:    "orq/orquestador/CodexReviewer/t3",
		BaseRef:   "master",
		State:     coordinacion.WorktreeActive,
		Reason:    "review_approved",
	})
	if err != nil {
		t.Fatalf("crear worktree activa: %v", err)
	}
	now := time.Now().UTC()
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:     &proyectoID,
		TareaID:        &tareaID,
		WorktreeID:     &worktree.ID,
		RequestedBy:    "orquesta",
		ReviewerAgente: "CodexReviewer",
		Estado:         db.ReviewGateAprobado,
		SeverityMax:    "high",
		ResolvedAt:     &now,
	}); err != nil {
		t.Fatalf("crear review gate aprobado: %v", err)
	}

	n, err := procesarReviewGatesBatch()
	if err != nil {
		t.Fatalf("procesar review gates: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 creación de solicitud de merge, got=%d", n)
	}

	merges, err := db.ListarGitMerges(&proyectoID, "")
	if err != nil {
		t.Fatalf("listar git merges: %v", err)
	}
	if len(merges) != 1 {
		t.Fatalf("debería crear una solicitud de merge, merges=%+v", merges)
	}
	if merges[0].SourceBranch != "orq/orquestador/CodexReviewer/t3" || merges[0].TargetBranch != "master" || merges[0].Estado != "aprobado" {
		t.Fatalf("solicitud de merge inesperada: %+v", merges[0])
	}
	if !strings.Contains(merges[0].MetadataJSON, `"review_gate":`) || !strings.Contains(merges[0].MetadataJSON, `"auto_created":true`) {
		t.Fatalf("metadata de merge incompleta: %s", merges[0].MetadataJSON)
	}
	policy, err := db.GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto autonomia: %v", err)
	}
	if policy.EstadoAutonomia != db.AutonomiaProyectoActiva {
		t.Fatalf("estado autonomia debería volver a activo tras review aprobada, got=%s", policy.EstadoAutonomia)
	}
}

func TestProyectoTerminadoAutonomamenteEsperaReviewAprobada(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "Codex1", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}

	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	terminado, _, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil {
		t.Fatalf("proyectoTerminadoAutonomamente sin review: %v", err)
	}
	if terminado {
		t.Fatal("no debería cerrar proyecto sin review aprobada")
	}
	now := time.Now().UTC()
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:  &proyectoID,
		RequestedBy: "orquesta",
		Estado:      db.ReviewGateAprobado,
		ResolvedAt:  &now,
	}); err != nil {
		t.Fatalf("crear review gate aprobado: %v", err)
	}
	terminado, motivo, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil {
		t.Fatalf("proyectoTerminadoAutonomamente con review: %v", err)
	}
	if !terminado {
		t.Fatalf("debería cerrar con review aprobada, motivo=%s", motivo)
	}
	if !strings.Contains(motivo, "review gate") {
		t.Fatalf("motivo de cierre sin referencia a review: %s", motivo)
	}
}

func TestProyectoTerminadoAutonomamenteEsperaMergeFusionadoSiExisteSolicitud(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	repo := prepararRepoGitAutonomia(t, filepath.Join(tmp, "orquestador"))

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repo,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "Codex1", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	now := time.Now().UTC()
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:  &proyectoID,
		RequestedBy: "orquesta",
		Estado:      db.ReviewGateAprobado,
		ResolvedAt:  &now,
	}); err != nil {
		t.Fatalf("crear review gate aprobado: %v", err)
	}
	if _, err := db.GuardarGitMerge(&db.GitMerge{
		ProyectoID:   proyectoID,
		SourceBranch: "feature/x",
		TargetBranch: "master",
		RequestedBy:  "orquesta",
		Estado:       "aprobado",
		MetadataJSON: `{"auto_created":true,"source":"review_gate_approved"}`,
	}); err != nil {
		t.Fatalf("guardar git merge: %v", err)
	}

	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	terminado, motivo, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil {
		t.Fatalf("proyectoTerminadoAutonomamente esperando merge: %v", err)
	}
	if terminado {
		t.Fatalf("no debería cerrar con merge aprobado pendiente, motivo=%s", motivo)
	}
	if !strings.Contains(motivo, "esperando integración merge") {
		t.Fatalf("motivo inesperado esperando merge: %s", motivo)
	}

	merges, err := db.ListarGitMerges(&proyectoID, "")
	if err != nil {
		t.Fatalf("listar merges: %v", err)
	}
	if len(merges) != 1 {
		t.Fatalf("esperaba 1 merge, got=%d", len(merges))
	}
	merges[0].Estado = "fusionado"
	merges[0].CommitMerge = "def456"
	if _, err := db.GuardarGitMerge(merges[0]); err != nil {
		t.Fatalf("actualizar merge fusionado: %v", err)
	}
	terminado, motivo, err = proyectoTerminadoAutonomamente(proyecto)
	if err != nil {
		t.Fatalf("proyectoTerminadoAutonomamente con merge fusionado: %v", err)
	}
	if !terminado {
		t.Fatalf("debería cerrar con merge fusionado, motivo=%s", motivo)
	}
	if !strings.Contains(motivo, "fusionado") {
		t.Fatalf("motivo de cierre sin merge fusionado: %s", motivo)
	}
}

func TestProyectoTerminadoAutonomamenteRespetaAutoCloseProjectFalse(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewRequired:       false,
		AutoCloseProject:     false,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "Codex1", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}

	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	terminado, _, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil {
		t.Fatalf("proyectoTerminadoAutonomamente auto_close=false: %v", err)
	}
	if terminado {
		t.Fatal("no debería considerar terminado un proyecto con auto_close_project=false")
	}
}

func TestProyectoTerminadoAutonomamenteRespetaAutoCloseProject(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewRequired:       true,
		AutoCloseProject:     false,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "Codex1", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	now := time.Now().UTC()
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:  &proyectoID,
		RequestedBy: "orquesta",
		Estado:      db.ReviewGateAprobado,
		ResolvedAt:  &now,
	}); err != nil {
		t.Fatalf("crear review gate aprobado: %v", err)
	}

	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	terminado, motivo, err := proyectoTerminadoAutonomamente(proyecto)
	if err != nil {
		t.Fatalf("proyectoTerminadoAutonomamente: %v", err)
	}
	if terminado {
		t.Fatalf("no debería cerrar con auto_close_project desactivado, motivo=%s", motivo)
	}
}

func TestConstruirInstruccionSupervisionRespetaAutoCreateTasks(t *testing.T) {
	proyecto := &db.Proyecto{ID: 1, Slug: "orquestador"}
	supervisor := &db.Agente{Nombre: "CodexSupervisor", Rol: "admin"}

	conBacklog := construirInstruccionSupervision(&db.ProyectoAutonomia{
		Enabled:         true,
		ObjetivoGeneral: "Terminar la app",
		AutoCreateTasks: true,
	}, proyecto, supervisor, "faltan frentes")
	if !strings.Contains(conBacklog, "crea o reajusta tareas") {
		t.Fatalf("la instrucción debería permitir crear backlog automáticamente: %s", conBacklog)
	}

	sinBacklog := construirInstruccionSupervision(&db.ProyectoAutonomia{
		Enabled:         true,
		ObjetivoGeneral: "Terminar la app",
		AutoCreateTasks: false,
	}, proyecto, supervisor, "faltan frentes")
	if strings.Contains(sinBacklog, "crea o reajusta tareas") {
		t.Fatalf("la instrucción no debería permitir crear backlog automáticamente: %s", sinBacklog)
	}
	if !strings.Contains(sinBacklog, "sin crear tareas nuevas automáticamente") {
		t.Fatalf("la instrucción debería reflejar el modo sin auto_create_tasks: %s", sinBacklog)
	}
}

func TestProcesarAutonomiaAgentesBatchEncolaPausePorReasignacion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoA, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-a",
		Nombre:  "Orquestador A",
		RutaAbs: filepath.Join(tmp, "orquestador-a"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-b",
		Nombre:  "Orquestador B",
		RutaAbs: filepath.Join(tmp, "orquestador-b"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoB, "cambio de frente"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoA,
		CWD:         filepath.Join(tmp, "orquestador-a"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "pause" {
		t.Fatalf("pause no encolada: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"accion":"pause"`) {
		t.Fatalf("payload pause inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarCierreProyectoSesionRespetaAutoCloseProject(t *testing.T) {
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewRequired:       true,
		AutoCloseProject:     false,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "cat-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "Codex1", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	now := time.Now().UTC()
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:  &proyectoID,
		RequestedBy: "orquesta",
		Estado:      db.ReviewGateAprobado,
		ResolvedAt:  &now,
	}); err != nil {
		t.Fatalf("crear review gate aprobado: %v", err)
	}

	n, err := procesarCierreProyectoSesion(sesion)
	if err != nil {
		t.Fatalf("procesar cierre proyecto: %v", err)
	}
	if n != 0 {
		t.Fatalf("no debería cerrar con auto_close_project desactivado, got=%d", n)
	}
	op, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo == db.ProyectoOperativoCerrado {
		t.Fatalf("el proyecto no debería quedar cerrado: %+v", op)
	}
}

func TestProcesarAutonomiaAgentesBatchEncolaNudgePorPropuestasPendientes(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := db.CrearPropuesta(&db.Propuesta{
		Titulo:       "Nueva política",
		Descripcion:  "Pendiente de votar",
		ProyectoID:   &proyectoID,
		Tipo:         "otro",
		PropuestoPor: "alberto",
		Distribuidor: "claude",
	}); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" {
		t.Fatalf("nudge no encolado: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"kind":"autonomia"`) ||
		!strings.Contains(orders[0].PayloadJSON, `"accion":"votar_propuestas_pendientes"`) {
		t.Fatalf("payload nudge inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarCierreProyectoAutonomiaSinSesionCierraProyectoYAutonomia(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	repo := prepararRepoGitAutonomia(t, filepath.Join(tmp, "orquestador"))

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repo,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoCerrando,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "Codex1", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	now := time.Now().UTC()
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:  &proyectoID,
		RequestedBy: "orquesta",
		Estado:      db.ReviewGateAprobado,
		ResolvedAt:  &now,
	}); err != nil {
		t.Fatalf("crear review gate aprobado: %v", err)
	}
	if _, err := db.GuardarGitMerge(&db.GitMerge{
		ProyectoID:   proyectoID,
		SourceBranch: "feature/x",
		TargetBranch: "master",
		RequestedBy:  "orquesta",
		Estado:       "fusionado",
		CommitMerge:  "def456",
		MetadataJSON: `{"auto_created":true,"source":"review_gate_approved"}`,
	}); err != nil {
		t.Fatalf("guardar git merge fusionado: %v", err)
	}

	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	ok, err := procesarCierreProyectoAutonomiaSinSesion(proyecto)
	if err != nil {
		t.Fatalf("procesar cierre proyecto sin sesion: %v", err)
	}
	if !ok {
		t.Fatal("debería cerrar el proyecto sin sesión activa")
	}
	op, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != db.ProyectoOperativoCerrado {
		t.Fatalf("proyecto no cerrado: %+v", op)
	}
	policy, err := supervisionService.GetProjectPolicy("orquestador")
	if err != nil {
		t.Fatalf("get project policy: %v", err)
	}
	if policy == nil {
		t.Fatal("debería existir policy")
	}
	if policy.EstadoAutonomia != db.AutonomiaProyectoCerrado {
		t.Fatalf("estado autonomia inesperado: got=%q want=%q", policy.EstadoAutonomia, db.AutonomiaProyectoCerrado)
	}
}

func TestProcesarAutonomiaAgentesBatchNoEncolaNudgePorContinuarTrabajoEnSesionActiva(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-remote",
		Nombre:     "Codex Remote",
		Transporte: "api",
		Comando:    "http://localhost:9999",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Seguir implementando",
		Descripcion: "Trabajo activo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia forzar nudge pasivo en sesion activa, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear nudge pasivo: %+v", orders)
	}
}

func TestProcesarDecisionContinuacionSesionActivaAutonomiaEncolaNudgeTMUXStale(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex9", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-stale-continue",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex9", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil || proyecto == nil {
		t.Fatalf("get proyecto: %+v err=%v", proyecto, err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Seguir implementando en stale tmux",
		Descripcion: "Trabajo activo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex9"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex9"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex9",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtimeInst, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtimeInst == nil {
		t.Fatalf("runtime: %+v err=%v", runtimeInst, err)
	}

	traceDir := filepath.Join(tmp, "runtime", "codex9")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	now := time.Now().UTC()
	staleAt := now.Add(-10 * time.Minute)
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex9","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex9-continue","tmux_pane_id":"%31","created_at":"`+now.Format(time.RFC3339Nano)+`","started_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"ready","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+staleAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+staleAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+staleAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+staleAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex9-continue",
		"tmux_pane_id":          "%31",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-codex9-continue/%31', metadata_json=?, estado='activo' WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("activar handle tmux: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='activo', process_state='running', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtimeInst.ID); err != nil {
		t.Fatalf("activar runtime: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	n, err := procesarDecisionContinuacionSesionActivaAutonomia(sesion, proyecto, agenteTickOutput{
		AccionRecomendada: "continuar_trabajo",
		Motivo:            "Sigue trabajando hasta completar la tarea o detectar una duda real",
	}, nil)
	if err != nil {
		t.Fatalf("procesarDecisionContinuacionSesionActivaAutonomia: %v", err)
	}
	if n == 0 {
		t.Fatal("deberia producir al menos una accion autonoma para la sesion tmux stale")
	}

	agente := "Codex9"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("deberia crear un nudge, got=%d", len(orders))
	}
	if orders[0].Tipo != "nudge" || !strings.Contains(orders[0].PayloadJSON, `"accion":"continuar_trabajo"`) {
		t.Fatalf("order inesperada: %+v", orders[0])
	}
	if !strings.Contains(orders[0].PayloadJSON, `"tarea_id":`+strconv.FormatInt(tareaID, 10)) {
		t.Fatalf("deberia incluir tarea activa en payload: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarDecisionContinuacionSesionActivaAutonomiaUsaInstructionEspecificaParaSemillaPremium(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex9Seed", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-seed-continue",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex9Seed", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil || proyecto == nil {
		t.Fatalf("get proyecto: %+v err=%v", proyecto, err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Autonomía premium: abrir siguiente frente mayor útil",
		Descripcion: "Semilla premium activa",
		ProyectoID:  &proyectoID,
		Modulo:      "autonomia",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       "autonomia:premium_frontier",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex9Seed"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex9Seed"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex9Seed",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtimeInst, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtimeInst == nil {
		t.Fatalf("runtime: %+v err=%v", runtimeInst, err)
	}
	traceDir := filepath.Join(tmp, "runtime", "codex9seed")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	now := time.Now().UTC()
	staleAt := now.Add(-10 * time.Minute)
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	manifestPath := filepath.Join(traceDir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex9Seed","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex9-seed","tmux_pane_id":"%31","created_at":"`+now.Format(time.RFC3339Nano)+`","started_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"ready","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+staleAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+staleAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write worker status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+staleAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+staleAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex9-seed",
		"tmux_pane_id":          "%31",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-codex9-seed/%31', metadata_json=?, estado='activo' WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("activar handle tmux: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='activo', process_state='running', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtimeInst.ID); err != nil {
		t.Fatalf("activar runtime: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	n, err := procesarDecisionContinuacionSesionActivaAutonomia(sesion, proyecto, agenteTickOutput{
		AccionRecomendada: "continuar_trabajo",
		Motivo:            "Sigue trabajando hasta completar la tarea o detectar una duda real",
	}, nil)
	if err != nil {
		t.Fatalf("procesarDecisionContinuacionSesionActivaAutonomia: %v", err)
	}
	if n == 0 {
		t.Fatal("deberia producir un nudge útil para la semilla premium activa")
	}

	agente := "Codex9Seed"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("deberia crear un nudge, got=%d", len(orders))
	}
	if !strings.Contains(orders[0].PayloadJSON, `"instruction":"Esta semilla premium no autoriza programar un frente amplio.`) {
		t.Fatalf("deberia usar instruction específica para la semilla premium: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarDecisionContinuacionSesionActivaAutonomiaEncolaNudgeTMUXStaleConHandleCanonicoNoEnlazado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex9Canon", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-stale-continue-canon",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex9Canon", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil || proyecto == nil {
		t.Fatalf("get proyecto: %+v err=%v", proyecto, err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Seguir implementando en stale tmux canonico",
		Descripcion: "Trabajo activo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex9Canon"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex9Canon"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex9Canon",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handleEnlazado, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handleEnlazado == nil {
		t.Fatalf("handle enlazado: %+v err=%v", handleEnlazado, err)
	}
	if _, err := db.DB.Exec(`DELETE FROM runtime_handles WHERE id=?`, handleEnlazado.ID); err != nil {
		t.Fatalf("delete linked handle: %v", err)
	}

	traceDir := filepath.Join(tmp, "runtime", "codex9canon")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	now := time.Now().UTC()
	staleAt := now.Add(-10 * time.Minute)
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex9Canon","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex9canon-continue","tmux_pane_id":"%36","created_at":"`+now.Format(time.RFC3339Nano)+`","started_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"ready","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`","ready_at":"`+staleAt.Format(time.RFC3339Nano)+`","last_output_at":"`+staleAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+staleAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now.Format(time.RFC3339Nano)+`","ready_at":"`+staleAt.Format(time.RFC3339Nano)+`","last_output_at":"`+staleAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+staleAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex9canon-continue",
		"tmux_pane_id":          "%36",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	capsJSON, err := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	if err != nil {
		t.Fatalf("marshal caps: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, capabilities_json, last_seen_at
	) VALUES (?,?,?,?,?,'activo',?,?,CURRENT_TIMESTAMP)`,
		"Codex9Canon", proyectoID, "tmux", "session", "orq-codex9canon-continue/%36", string(metaJSON), string(capsJSON)); err != nil {
		t.Fatalf("insert canonical handle: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	n, err := procesarDecisionContinuacionSesionActivaAutonomia(sesion, proyecto, agenteTickOutput{
		AccionRecomendada: "continuar_trabajo",
		Motivo:            "Sigue trabajando hasta completar la tarea o detectar una duda real",
	}, nil)
	if err != nil {
		t.Fatalf("procesarDecisionContinuacionSesionActivaAutonomia: %v", err)
	}
	if n == 0 {
		t.Fatal("deberia producir una accion autonoma usando el handle canónico no enlazado")
	}

	agente := "Codex9Canon"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("deberia crear un nudge, got=%d", len(orders))
	}
	if orders[0].Tipo != "nudge" || !strings.Contains(orders[0].PayloadJSON, `"accion":"continuar_trabajo"`) {
		t.Fatalf("order inesperada: %+v", orders[0])
	}
	if !strings.Contains(orders[0].PayloadJSON, `"tarea_id":`+strconv.FormatInt(tareaID, 10)) {
		t.Fatalf("deberia incluir tarea activa en payload: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarDecisionContinuacionSesionActivaAutonomiaNoEncolaNudgeTMUXFresco(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex10", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-fresh-continue",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex10", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil || proyecto == nil {
		t.Fatalf("get proyecto: %+v err=%v", proyecto, err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Seguir implementando en fresh tmux",
		Descripcion: "Trabajo activo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex10"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex10"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex10",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtimeInst, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtimeInst == nil {
		t.Fatalf("runtime: %+v err=%v", runtimeInst, err)
	}

	traceDir := filepath.Join(tmp, "runtime", "codex10")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	now := time.Now().UTC()
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex10","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex10-continue","tmux_pane_id":"%32","created_at":"`+now.Format(time.RFC3339Nano)+`","started_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"ready","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+now.Format(time.RFC3339Nano)+`","last_progress_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+now.Format(time.RFC3339Nano)+`","last_progress_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex10-continue",
		"tmux_pane_id":          "%32",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-codex10-continue/%32', metadata_json=?, estado='activo' WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("activar handle tmux: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='activo', process_state='running', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtimeInst.ID); err != nil {
		t.Fatalf("activar runtime: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	if _, err := procesarDecisionContinuacionSesionActivaAutonomia(sesion, proyecto, agenteTickOutput{
		AccionRecomendada: "continuar_trabajo",
		Motivo:            "Sigue trabajando hasta completar la tarea o detectar una duda real",
	}, nil); err != nil {
		t.Fatalf("procesarDecisionContinuacionSesionActivaAutonomia: %v", err)
	}

	agente := "Codex10"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear nudge: %+v", orders)
	}
}

func TestProcesarDecisionContinuacionSesionActivaAutonomiaReintentaTrasSendInstructionAbiertaSinQuemarThrottle(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	autonomiaContinueNudgeGate.Reset()

	if err := db.RegistrarAgente("Codex10Retry", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-retry-continue",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex10Retry", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil || proyecto == nil {
		t.Fatalf("get proyecto: %+v err=%v", proyecto, err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Reintento tras send abierta",
		Descripcion: "Trabajo activo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex10Retry"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex10Retry"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex10Retry",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtimeInst, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtimeInst == nil {
		t.Fatalf("runtime: %+v err=%v", runtimeInst, err)
	}

	traceDir := filepath.Join(tmp, "runtime", "codex10retry")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	now := time.Now().UTC()
	staleAt := now.Add(-10 * time.Minute)
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex10Retry","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex10retry","tmux_pane_id":"%34","created_at":"`+now.Format(time.RFC3339Nano)+`","started_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"ready","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+staleAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+staleAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+staleAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+staleAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex10retry",
		"tmux_pane_id":          "%34",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-codex10retry/%34', metadata_json=?, estado='activo' WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("activar handle tmux: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='activo', process_state='running', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtimeInst.ID); err != nil {
		t.Fatalf("activar runtime: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	payloadJSON := `{"from_agente":"server","to_agente":"Codex10Retry","kind":"autonomia","accion":"continuar_trabajo","texto":"seguir"}`
	if _, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex10Retry",
		ProyectoID:  &proyectoID,
		RuntimeID:   &runtimeInst.ID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: payloadJSON,
	}); err != nil {
		t.Fatalf("encolar send_instruction abierta: %v", err)
	}

	n, err := procesarDecisionContinuacionSesionActivaAutonomia(sesion, proyecto, agenteTickOutput{
		AccionRecomendada: "continuar_trabajo",
		Motivo:            "Sigue trabajando hasta completar la tarea o detectar una duda real",
	}, nil)
	if err != nil {
		t.Fatalf("primer intento continuar_trabajo: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia encolar nudge si ya hay send_instruction abierta, got=%d", n)
	}

	agente := "Codex10Retry"
	estadoPendiente := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("listar orders pendientes: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "send_instruction" {
		t.Fatalf("estado inicial inesperado: %+v", orders)
	}
	if err := db.MarcarRuntimeOrderEstado(orders[0].ID, "completada", `{"delivery_state":"delivered"}`, ""); err != nil {
		t.Fatalf("completar send_instruction abierta: %v", err)
	}

	n, err = procesarDecisionContinuacionSesionActivaAutonomia(sesion, proyecto, agenteTickOutput{
		AccionRecomendada: "continuar_trabajo",
		Motivo:            "Sigue trabajando hasta completar la tarea o detectar una duda real",
	}, nil)
	if err != nil {
		t.Fatalf("segundo intento continuar_trabajo: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reintentar y encolar nudge tras cerrarse la send_instruction, got=%d", n)
	}

	orders, err = db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("listar orders finales: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" {
		t.Fatalf("deberia quedar un nudge pendiente, got=%+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"accion":"continuar_trabajo"`) ||
		!strings.Contains(orders[0].PayloadJSON, `"tarea_id":`+strconv.FormatInt(tareaID, 10)) {
		t.Fatalf("payload nudge inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestSesionActivaTMUXStaleParaContinuacionPrefiereProgressSobreOutput(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex11Live", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-progress-priority",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex11Live",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	traceDir := filepath.Join(tmp, "runtime", "codex11live")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	now := time.Now().UTC()
	progressAt := now.Add(-10 * time.Minute)
	outputAt := now.Add(-30 * time.Second)
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex11Live","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex11live","tmux_pane_id":"%33","created_at":"`+now.Format(time.RFC3339Nano)+`","started_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"ready","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+outputAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+progressAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+outputAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+progressAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex11live",
		"tmux_pane_id":          "%33",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-codex11live/%33', metadata_json=?, estado='activo' WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("activar handle tmux: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()
	handle, err = db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}

	if !sesionActivaTMUXStaleParaContinuacion(handle, now, 5*time.Minute) {
		t.Fatal("deberia considerar stale una sesion con output fresco pero progreso antiguo")
	}
}

func TestSesionActivaTMUXStaleParaContinuacionPrefiereReadySobreOutputSinProgress(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex11Ready", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-ready-priority",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex11Ready",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	traceDir := filepath.Join(tmp, "runtime", "codex11ready")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	now := time.Now().UTC()
	readyAt := now.Add(-10 * time.Minute)
	outputAt := now.Add(-30 * time.Second)
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex11Ready","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex11ready","tmux_pane_id":"%34","created_at":"`+now.Format(time.RFC3339Nano)+`","started_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"ready","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`","ready_at":"`+readyAt.Format(time.RFC3339Nano)+`","last_output_at":"`+outputAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now.Format(time.RFC3339Nano)+`","ready_at":"`+readyAt.Format(time.RFC3339Nano)+`","last_output_at":"`+outputAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex11ready",
		"tmux_pane_id":          "%34",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-codex11ready/%34', metadata_json=?, estado='activo' WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("activar handle tmux: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()
	handle, err = db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}

	if !sesionActivaTMUXStaleParaContinuacion(handle, now, 5*time.Minute) {
		t.Fatal("deberia considerar stale una sesion con output fresco pero ready antiguo sin progreso")
	}
}

func TestSesionActivaDebeRecibirNudgeContinuacionUsaHandleCanonicoNoEnlazado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex11Canon", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-canonico-stale",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex11Canon",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	linkHandle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || linkHandle == nil {
		t.Fatalf("handle enlazado: %+v err=%v", linkHandle, err)
	}
	if _, err := db.DB.Exec(`DELETE FROM runtime_handles WHERE id=?`, linkHandle.ID); err != nil {
		t.Fatalf("delete linked handle: %v", err)
	}

	traceDir := filepath.Join(tmp, "runtime", "codex11canon")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	now := time.Now().UTC()
	readyAt := now.Add(-10 * time.Minute)
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex11Canon","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex11canon","tmux_pane_id":"%35","created_at":"`+now.Format(time.RFC3339Nano)+`","started_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"ready","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`","ready_at":"`+readyAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now.Format(time.RFC3339Nano)+`","ready_at":"`+readyAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex11canon",
		"tmux_pane_id":          "%35",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	capsJSON, err := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	if err != nil {
		t.Fatalf("marshal caps: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, capabilities_json, last_seen_at
	) VALUES (?,?,?,?,?,'activo',?,?,CURRENT_TIMESTAMP)`,
		"Codex11Canon", proyectoID, "tmux", "session", "orq-codex11canon", string(metaJSON), string(capsJSON)); err != nil {
		t.Fatalf("insert canonical handle: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Continuar trabajo con handle canonico",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex11Canon"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex11Canon"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	gotTaskID, debe, err := sesionActivaDebeRecibirNudgeContinuacion(sesion, &db.Proyecto{ID: proyectoID, Slug: "orquestador-canonico-stale"})
	if err != nil {
		t.Fatalf("sesionActivaDebeRecibirNudgeContinuacion: %v", err)
	}
	if !debe {
		t.Fatal("deberia detectar stale usando handle canónico no enlazado")
	}
	if gotTaskID != tareaID {
		t.Fatalf("tarea objetivo inesperada: got=%d want=%d", gotTaskID, tareaID)
	}
}

func TestSesionActivaDebeRecibirNudgeContinuacionIgnoraGuidanceDurableMailboxOnlyVigente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex11Guidance", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-guidance-vigente",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex11Guidance",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Continuidad guidance durable",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex11Guidance"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex11Guidance"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	traceDir := filepath.Join(tmp, "runtime", "codex11guidance")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	now := time.Now().UTC()
	readyAt := now.Add(-10 * time.Minute)
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex11Guidance","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex11guidance","tmux_pane_id":"%36","created_at":"`+now.Format(time.RFC3339Nano)+`","started_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"ready","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`","ready_at":"`+readyAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now.Format(time.RFC3339Nano)+`","ready_at":"`+readyAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex11guidance",
		"tmux_pane_id":          "%36",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
		"can_send_input":        false,
		"external_session_id":   "sess-codex11guidance",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	capsJSON, err := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	if err != nil {
		t.Fatalf("marshal caps: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-codex11guidance/%36', metadata_json=?, capabilities_json=?, estado='activo' WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("activar handle tmux: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()
	handle, err = db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}
	if !sesionActivaTMUXStaleParaContinuacion(handle, now, 5*time.Minute) {
		t.Fatal("precondicion rota: la sesion deberia considerarse stale")
	}

	signature := runtimeMailboxDeliveryAttemptSignature(handle, "sess-codex11guidance")
	payloadJSON := fmt.Sprintf(`{"to_agente":"Codex11Guidance","accion":"continuar_trabajo","texto":"sigue con el siguiente slice util","mailbox_kind":"autonomia","external_session_id":"sess-codex11guidance","delivery_attempt_signature":"%s"}`, signature)
	resultadoJSON := `{"mailbox_only":true,"delivery_state":"delivered","deferred_reason":"runtime_handle_session_resume_mailbox_only:autonomia"}`
	if _, err := runtimesService.CreateRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex11Guidance",
		ProyectoID:    &proyectoID,
		RuntimeID:     handle.RuntimeID,
		HandleID:      &handle.ID,
		Tipo:          "send_instruction",
		PayloadJSON:   payloadJSON,
		ResultadoJSON: resultadoJSON,
		Estado:        "completada",
		FinishedAt:    &now,
		UpdatedAt:     now,
	}); err != nil {
		t.Fatalf("crear send_instruction mailbox_only: %v", err)
	}

	_, debe, err := sesionActivaDebeRecibirNudgeContinuacion(sesion, &db.Proyecto{ID: proyectoID})
	if err != nil {
		t.Fatalf("evaluar continuidad: %v", err)
	}
	if debe {
		t.Fatal("no deberia pedir nudge si ya existe guidance durable mailbox_only vigente")
	}
}

func TestSesionActivaDebeRecibirNudgeContinuacionReemiteMailboxOnlyCompletadaCaducada(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex11GuidanceOld", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-guidance-old",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex11GuidanceOld", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Continuidad guidance vieja",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex11GuidanceOld"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex11GuidanceOld"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex11GuidanceOld",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtimeInst, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtimeInst == nil {
		t.Fatalf("runtime: %+v err=%v", runtimeInst, err)
	}
	now := time.Now().UTC()
	old := now.Add(-(autonomiaContinueNudgeInterval() + time.Minute))
	traceDir := filepath.Join(tmp, "runtime", "codex11guidanceold")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex11GuidanceOld","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex11guidanceold","tmux_pane_id":"%41","created_at":"`+now.Format(time.RFC3339Nano)+`","started_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"ready","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+old.Format(time.RFC3339Nano)+`","last_progress_at":"`+old.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+old.Format(time.RFC3339Nano)+`","last_progress_at":"`+old.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","transport":"tmux","external_session_id":"sess-guidance-old","tmux_session":"orq-codex11guidanceold","tmux_pane_id":"%%41","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, manifestPath, statusPath, heartbeatPath)
	capsJSON := `{"mailbox_delivery_mode":"session_resume"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-codex11guidanceold/%41', capabilities_json=?, metadata_json=?, estado='activo', updated_at=? WHERE id=?`, capsJSON, metaJSON, now, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='activo', process_state='running', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtimeInst.ID); err != nil {
		t.Fatalf("activar runtime: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	signature := runtimeMailboxDeliveryAttemptSignature(handle, "sess-guidance-old")
	payloadJSON := fmt.Sprintf(`{"to_agente":"Codex11GuidanceOld","accion":"continuar_trabajo","texto":"sigue con el siguiente slice util","mailbox_kind":"autonomia","external_session_id":"sess-guidance-old","delivery_attempt_signature":"%s"}`, signature)
	resultadoJSON := `{"mailbox_only":true,"delivery_state":"delivered","deferred_reason":"runtime_handle_session_resume_mailbox_only:autonomia"}`
	if _, err := runtimesService.CreateRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex11GuidanceOld",
		ProyectoID:    &proyectoID,
		RuntimeID:     handle.RuntimeID,
		HandleID:      &handle.ID,
		Tipo:          "send_instruction",
		PayloadJSON:   payloadJSON,
		ResultadoJSON: resultadoJSON,
		Estado:        "completada",
		FinishedAt:    &old,
		UpdatedAt:     old,
	}); err != nil {
		t.Fatalf("crear send_instruction mailbox_only vieja: %v", err)
	}

	_, debe, err := sesionActivaDebeRecibirNudgeContinuacion(sesion, &db.Proyecto{ID: proyectoID})
	if err != nil {
		t.Fatalf("evaluar continuidad: %v", err)
	}
	if !debe {
		t.Fatal("deberia reemitir nudge si la guidance mailbox_only completada ya caducó para sesión activa")
	}
}

func TestSesionActivaDebeRecibirNudgeContinuacionIgnoraSendInstructionAbiertaVigente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex11GuidanceOpen", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-guidance-open",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex11GuidanceOpen",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Continuidad guidance abierta",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex11GuidanceOpen"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex11GuidanceOpen"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	traceDir := filepath.Join(tmp, "runtime", "codex11guidanceopen")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	now := time.Now().UTC()
	readyAt := now.Add(-10 * time.Minute)
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex11GuidanceOpen","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex11guidanceopen","tmux_pane_id":"%37","created_at":"`+now.Format(time.RFC3339Nano)+`","started_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"ready","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`","ready_at":"`+readyAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now.Format(time.RFC3339Nano)+`","ready_at":"`+readyAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex11guidanceopen",
		"tmux_pane_id":          "%37",
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
		"can_send_input":        false,
		"external_session_id":   "sess-codex11guidanceopen",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	capsJSON, err := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	if err != nil {
		t.Fatalf("marshal caps: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-codex11guidanceopen/%37', metadata_json=?, capabilities_json=?, estado='activo' WHERE id=?`, string(metaJSON), string(capsJSON), handle.ID); err != nil {
		t.Fatalf("activar handle tmux: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()
	handle, err = db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}
	if !sesionActivaTMUXStaleParaContinuacion(handle, now, 5*time.Minute) {
		t.Fatal("precondicion rota: la sesion deberia considerarse stale")
	}

	signature := runtimeMailboxDeliveryAttemptSignature(handle, "sess-codex11guidanceopen")
	payloadJSON := fmt.Sprintf(`{"to_agente":"Codex11GuidanceOpen","accion":"continuar_trabajo","texto":"sigue con el siguiente slice util","mailbox_kind":"autonomia","external_session_id":"sess-codex11guidanceopen","delivery_attempt_signature":"%s"}`, signature)
	if _, err := runtimesService.CreateRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex11GuidanceOpen",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: payloadJSON,
		Estado:      "ejecutando",
		StartedAt:   &now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("crear send_instruction abierta: %v", err)
	}

	_, debe, err := sesionActivaDebeRecibirNudgeContinuacion(sesion, &db.Proyecto{ID: proyectoID})
	if err != nil {
		t.Fatalf("evaluar continuidad: %v", err)
	}
	if debe {
		t.Fatal("no deberia pedir nudge si ya existe una send_instruction de continuidad abierta para la misma sesion")
	}
}

func TestProcesarAutonomiaAgentesBatchNoEncolaNudgePorEsperarOPedirTareaEnSesionActiva(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia forzar nudge pasivo en sesion activa, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear nudge pasivo: %+v", orders)
	}
}

func TestAutonomiaContinueNudgeIntervalDefault(t *testing.T) {
	prepararDBTemporalCmd(t)
	if got := autonomiaContinueNudgeInterval(); got != time.Minute {
		t.Fatalf("autonomiaContinueNudgeInterval default inesperado: %s", got)
	}
}

func TestProcesarAutonomiaAgentesBatchAutoasignaTrabajoASesionActivaIdle(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.ConfigSet("server_autobootstrap_project_slug", "orquestador"); err != nil {
		t.Fatalf("config project slug: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_worker_agents", "Codex1"); err != nil {
		t.Fatalf("config worker agents: %v", err)
	}
	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		MaxWorkers:        1,
		SupervisorAgente:  "CodexSupervisor",
		ReserveSupervisor: true,
		EstadoAutonomia:   db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "cat-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Siguiente frente útil",
		Descripcion: "Debe entrar en sesión activa idle",
		ProyectoID:  &proyectoID,
		Modulo:      "planocontrol",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	n, err := procesarAutonomiaSesionActiva(sesion)
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia autoasignar y encolar nudge útil, got=%d", n)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex1" || tarea.Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea deberia quedar autoasignada y arrancada en la sesion viva: %+v", tarea)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" {
		t.Fatalf("deberia encolar un nudge útil tras autoasignar: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"accion":"continuar_trabajo"`) || !strings.Contains(orders[0].PayloadJSON, `"tarea_id":`) {
		t.Fatalf("payload nudge inesperado: %s", orders[0].PayloadJSON)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"instruction":"toma tarea asignada y sigue"`) {
		t.Fatalf("instruction de autoasignacion demasiado larga o inesperada: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarAutonomiaAgentesBatchSesionActivaIdleAbreFrentePremiumMayorSiMicrocicloAgotado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.ConfigSet("server_autobootstrap_project_slug", "orquestador"); err != nil {
		t.Fatalf("config project slug: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_worker_agents", "Codex1"); err != nil {
		t.Fatalf("config worker agents: %v", err)
	}
	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.RegistrarAgente("QwenCoder2", "programador"); err != nil {
		t.Fatalf("registrar agente ajeno: %v", err)
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
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		Prioridad:        100,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		MaxWorkers:        1,
		SupervisorAgente:  "CodexSupervisor",
		ReserveSupervisor: true,
		EstadoAutonomia:   db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion worker: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "cat-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion worker: %v", err)
	}
	tareaMicroID, err := db.CrearTarea(&db.Tarea{
		Titulo:      microcicloRefactorTituloDefault,
		Descripcion: "Microfrente premium agotado",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea microciclo agotada: %v", err)
	}
	if err := db.TomarTarea(tareaMicroID, "CodexSupervisor"); err != nil {
		t.Fatalf("tomar tarea microciclo agotada: %v", err)
	}
	if err := db.IniciarTarea(tareaMicroID, "CodexSupervisor"); err != nil {
		t.Fatalf("iniciar tarea microciclo agotada: %v", err)
	}
	if err := db.CompletarTarea(tareaMicroID, "CodexSupervisor", "agotado"); err != nil {
		t.Fatalf("completar tarea microciclo agotada: %v", err)
	}
	tareaAjenaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Trabajo ajeno sigue abierto",
		Descripcion: "No debe bloquear abrir siguiente frente premium mayor",
		ProyectoID:  &proyectoID,
		Modulo:      "otro-modulo",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea ajena: %v", err)
	}
	if err := db.TomarTarea(tareaAjenaID, "QwenCoder2"); err != nil {
		t.Fatalf("tomar tarea ajena: %v", err)
	}
	if err := db.IniciarTarea(tareaAjenaID, "QwenCoder2"); err != nil {
		t.Fatalf("iniciar tarea ajena: %v", err)
	}

	n, err := procesarAutonomiaSesionActiva(sesion)
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia abrir frente premium mayor y encolar continuidad, got=%d", n)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	var abierta *db.Tarea
	for _, tarea := range tareas {
		if tarea == nil || tarea.ID == tareaMicroID || tarea.ID == tareaAjenaID {
			continue
		}
		if tarea.Estado == db.TareaEnProgreso && tarea.Agente != nil && strings.TrimSpace(*tarea.Agente) == "Codex1" {
			abierta = tarea
			break
		}
	}
	if abierta == nil {
		t.Fatalf("deberia abrir una nueva tarea premium mayor para Codex1, tareas=%+v", tareas)
	}
	if !strings.Contains(strings.TrimSpace(abierta.Notas), "autonomia:premium_frontier") {
		t.Fatalf("la tarea nueva deberia quedar marcada como premium_frontier: %+v", abierta)
	}
	if !strings.Contains(abierta.Descripcion, "tómala o reanúdala") || !strings.Contains(abierta.Descripcion, "no abras varios frentes") {
		t.Fatalf("la descripcion del frente premium deberia imponer la secuencia canónica: %+v", abierta)
	}
	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "nudge" {
		t.Fatalf("deberia encolar un nudge útil tras abrir frente premium mayor: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"accion":"continuar_trabajo"`) || !strings.Contains(orders[0].PayloadJSON, `"tarea_id":`+strconv.FormatInt(abierta.ID, 10)) {
		t.Fatalf("payload nudge inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestAsegurarTrabajoContinuoPremiumNoDeclaraAgotadoSiQuedaMicrocicloAbierto(t *testing.T) {
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
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       1,
		Prioridad:        100,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:       proyectoID,
		Enabled:          true,
		MaxWorkers:       1,
		SupervisorAgente: "Codex1",
		EstadoAutonomia:  db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	policy, err := db.GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get autonomia: %v", err)
	}
	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	agente, err := db.GetAgente("Codex1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}

	tareaHistoricaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      microcicloRefactorTituloDefault,
		Descripcion: "Microfrente premium historico",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea historica: %v", err)
	}
	if err := db.TomarTarea(tareaHistoricaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea historica: %v", err)
	}
	if err := db.IniciarTarea(tareaHistoricaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea historica: %v", err)
	}
	if err := db.CompletarTarea(tareaHistoricaID, "Codex1", "hecho"); err != nil {
		t.Fatalf("completar tarea historica: %v", err)
	}

	tareaAbiertaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      microcicloRefactorTituloDefault,
		Descripcion: descripcionMicrocicloDefault(proyecto),
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea abierta: %v", err)
	}

	res, err := asegurarTrabajoContinuoPremium(policy, proyecto, agente)
	if err != nil {
		t.Fatalf("asegurar trabajo continuo premium: %v", err)
	}
	if res.Created || res.SeedTaskID != tareaAbiertaID {
		t.Fatalf("deberia reutilizar la microtarea abierta en vez de abrir semilla premium: %+v", res)
	}

	tareaAbierta, err := db.GetTarea(tareaAbiertaID)
	if err != nil {
		t.Fatalf("get tarea abierta: %v", err)
	}
	if tareaAbierta.Estado != db.TareaEnProgreso || tareaAbierta.Agente == nil || strings.TrimSpace(*tareaAbierta.Agente) != "Codex1" {
		t.Fatalf("la microtarea abierta deberia quedar tomada y en progreso: %+v", tareaAbierta)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	for _, tarea := range tareas {
		if tarea == nil || tarea.ID == tareaHistoricaID || tarea.ID == tareaAbiertaID {
			continue
		}
		if strings.Contains(strings.TrimSpace(tarea.Notas), "autonomia:premium_frontier") {
			t.Fatalf("no deberia abrir una semilla premium nueva si queda microciclo abierto: %+v", tarea)
		}
	}
}

func TestProcesarAutonomiaAgentesBatchSesionActivaIdleCierraSemillaPremiumSiTomaFrenteAcotado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	autonomiaIdleAutoassignGate.Reset()

	if err := db.ConfigSet("server_autobootstrap_project_slug", "orquestador"); err != nil {
		t.Fatalf("config project slug: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_worker_agents", "Codex1"); err != nil {
		t.Fatalf("config worker agents: %v", err)
	}
	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		MaxWorkers:        1,
		SupervisorAgente:  "CodexSupervisor",
		ReserveSupervisor: true,
		EstadoAutonomia:   db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "cat-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	semillaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Autonomía premium: abrir siguiente frente mayor útil",
		Descripcion: "Semilla premium activa",
		ProyectoID:  &proyectoID,
		Modulo:      "autonomia",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       "autonomia:premium_frontier",
	})
	if err != nil {
		t.Fatalf("crear semilla premium: %v", err)
	}
	if err := db.TomarTarea(semillaID, "Codex1"); err != nil {
		t.Fatalf("tomar semilla premium: %v", err)
	}
	if err := db.IniciarTarea(semillaID, "Codex1"); err != nil {
		t.Fatalf("iniciar semilla premium: %v", err)
	}

	tareaAcotadaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Cerrar continuidad premium TMUX",
		Descripcion: "Frente premium acotado.\nWRITE_SET: cmd/controlplane_support.go, cmd/controlplane_support_test.go\nTests minimos: go test ./cmd -run 'TestProcesarAutonomiaAgentesBatch.*'",
		ProyectoID:  &proyectoID,
		Modulo:      "autonomia",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea acotada: %v", err)
	}

	n, err := procesarAutonomiaSesionActiva(sesion)
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia autoasignar el frente acotado y encolar continuidad, got=%d", n)
	}

	semilla, err := db.GetTarea(semillaID)
	if err != nil {
		t.Fatalf("get semilla premium: %v", err)
	}
	if semilla.Estado != db.TareaCompletada {
		t.Fatalf("la semilla premium deberia quedar completada tras derivar a un frente real: %+v", semilla)
	}

	tareaAcotada, err := db.GetTarea(tareaAcotadaID)
	if err != nil {
		t.Fatalf("get tarea acotada: %v", err)
	}
	if tareaAcotada.Agente == nil || strings.TrimSpace(*tareaAcotada.Agente) != "Codex1" || tareaAcotada.Estado != db.TareaAsignada {
		t.Fatalf("el frente acotado deberia quedar asignado al agente activo: %+v", tareaAcotada)
	}
}

func TestProcesarAutonomiaAgentesBatchSesionActivaIdleCierraSemillaPremiumSiTomaMicrocicloLibre(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	autonomiaIdleAutoassignGate.Reset()

	if err := db.ConfigSet("server_autobootstrap_project_slug", "orquestador"); err != nil {
		t.Fatalf("config project slug: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_worker_agents", "Codex1"); err != nil {
		t.Fatalf("config worker agents: %v", err)
	}
	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		MaxWorkers:        1,
		SupervisorAgente:  "CodexSupervisor",
		ReserveSupervisor: true,
		EstadoAutonomia:   db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "cat-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	semillaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Autonomía premium: abrir siguiente frente mayor útil",
		Descripcion: "Semilla premium activa",
		ProyectoID:  &proyectoID,
		Modulo:      "autonomia",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       "autonomia:premium_frontier",
	})
	if err != nil {
		t.Fatalf("crear semilla premium: %v", err)
	}
	if err := db.TomarTarea(semillaID, "Codex1"); err != nil {
		t.Fatalf("tomar semilla premium: %v", err)
	}
	if err := db.IniciarTarea(semillaID, "Codex1"); err != nil {
		t.Fatalf("iniciar semilla premium: %v", err)
	}

	tareaMicroID, err := db.CrearTarea(&db.Tarea{
		Titulo:      microcicloRefactorTituloDefault,
		Descripcion: descripcionMicrocicloDefault(&db.Proyecto{Slug: "orquestador", RutaAbs: filepath.Join(tmp, "orquestador")}),
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear microtarea libre: %v", err)
	}

	n, err := procesarAutonomiaSesionActiva(sesion)
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia autoasignar la microtarea libre y encolar continuidad, got=%d", n)
	}

	semilla, err := db.GetTarea(semillaID)
	if err != nil {
		t.Fatalf("get semilla premium: %v", err)
	}
	if semilla.Estado != db.TareaCompletada {
		t.Fatalf("la semilla premium deberia quedar completada tras derivar a una microtarea real: %+v", semilla)
	}

	tareaMicro, err := db.GetTarea(tareaMicroID)
	if err != nil {
		t.Fatalf("get microtarea: %v", err)
	}
	if tareaMicro.Agente == nil || strings.TrimSpace(*tareaMicro.Agente) != "Codex1" || tareaMicro.Estado != db.TareaAsignada {
		t.Fatalf("la microtarea deberia quedar asignada al agente activo: %+v", tareaMicro)
	}
}

func TestAutonomiaIdleAutoassignShouldAttemptThrottle(t *testing.T) {
	prepararDBTemporalCmd(t)
	autonomiaIdleAutoassignGate.Reset()

	if !autonomiaIdleAutoassignShouldAttempt("Codex1", 1) {
		t.Fatalf("primer intento deberia permitirse")
	}
	if autonomiaIdleAutoassignShouldAttempt("Codex1", 1) {
		t.Fatalf("segundo intento inmediato no deberia permitirse")
	}
	if !autonomiaIdleAutoassignShouldAttempt("Codex1", 2) {
		t.Fatalf("proyecto distinto deberia permitir intento")
	}
	if !autonomiaIdleAutoassignShouldAttempt("Codex2", 1) {
		t.Fatalf("agente distinto deberia permitir intento")
	}
}

func TestProcesarAutonomiaSesionActivaCompactaFrentesPremiumAsignadosDuplicados(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("claude1", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("claude1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "claude1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "claude-code",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	tareaGeneralID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Micro-refactorización cíclica del control plane",
		Descripcion: "Frente general del control plane.",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea general: %v", err)
	}
	if err := db.TomarTarea(tareaGeneralID, "claude1"); err != nil {
		t.Fatalf("tomar tarea general: %v", err)
	}

	tareaAcotadaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Micro-refactorización cíclica del control plane: runtime mailbox/session_resume",
		Descripcion: "Frente premium acotado.\nWrite-set preferente: cmd/controlplane_support.go, cmd/controlplane_support_test.go\nTests minimos: go test ./cmd -run 'TestProcesarAutonomiaSesionActivaCompactaFrentesPremiumAsignadosDuplicados$'",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea acotada: %v", err)
	}
	if err := db.TomarTarea(tareaAcotadaID, "claude1"); err != nil {
		t.Fatalf("tomar tarea acotada: %v", err)
	}

	n, err := procesarAutonomiaSesionActiva(sesion)
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia compactar el frente premium duplicado, got=%d", n)
	}

	tareaGeneral, err := db.GetTarea(tareaGeneralID)
	if err != nil {
		t.Fatalf("get tarea general: %v", err)
	}
	if tareaGeneral.Estado != db.TareaBacklog || tareaGeneral.Agente != nil {
		t.Fatalf("la tarea general deberia volver a backlog al compactar duplicados: %+v", tareaGeneral)
	}

	tareaAcotada, err := db.GetTarea(tareaAcotadaID)
	if err != nil {
		t.Fatalf("get tarea acotada: %v", err)
	}
	if tareaAcotada.Estado != db.TareaAsignada || tareaAcotada.Agente == nil || !strings.EqualFold(strings.TrimSpace(*tareaAcotada.Agente), "claude1") {
		t.Fatalf("la tarea acotada deberia quedar como frente canónico: %+v", tareaAcotada)
	}
}

func TestProcesarCompactacionExclusividadPremiumSesionActivaMueveRestosALaBacklog(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	orqID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto orquestador: %v", err)
	}
	otroID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "multi-app",
		Nombre:  "Multi App",
		RutaAbs: filepath.Join(tmp, "multi-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto otro: %v", err)
	}
	if err := db.ActivarAsignacion("claude1", orqID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "claude1",
		ProyectoID:  &orqID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "claude-code",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	keepID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente premium acotado",
		Descripcion: "WRITE_SET: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestProcesarCompactacionExclusividadPremiumSesionActivaMueveRestosALaBacklog$'",
		ProyectoID:  &orqID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear frente premium: %v", err)
	}
	if err := db.TomarTarea(keepID, "claude1"); err != nil {
		t.Fatalf("tomar frente premium: %v", err)
	}

	legacyID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Tarea legacy",
		Descripcion: "Tarea amplia sin contrato bounded",
		ProyectoID:  &orqID,
		Modulo:      "legacy",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea legacy: %v", err)
	}
	if err := db.TomarTarea(legacyID, "claude1"); err != nil {
		t.Fatalf("tomar tarea legacy: %v", err)
	}

	ajenaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Tarea ajena",
		Descripcion: "No debe seguir en un premium exclusivo de otro proyecto",
		ProyectoID:  &otroID,
		Modulo:      "api",
		Prioridad:   db.PrioridadMedia,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea ajena: %v", err)
	}
	if err := db.TomarTarea(ajenaID, "claude1"); err != nil {
		t.Fatalf("tomar tarea ajena: %v", err)
	}

	n, err := procesarCompactacionExclusividadPremiumSesionActiva(sesion, nil)
	if err != nil {
		t.Fatalf("compactar exclusividad: %v", err)
	}
	if n != 2 {
		t.Fatalf("deberia mover dos restos a backlog, got=%d", n)
	}

	keep, err := db.GetTarea(keepID)
	if err != nil {
		t.Fatalf("get keep: %v", err)
	}
	if keep.Estado != db.TareaAsignada || keep.Agente == nil || !strings.EqualFold(strings.TrimSpace(*keep.Agente), "claude1") {
		t.Fatalf("el frente premium bounded deberia conservarse: %+v", keep)
	}

	legacy, err := db.GetTarea(legacyID)
	if err != nil {
		t.Fatalf("get legacy: %v", err)
	}
	if legacy.Estado != db.TareaBacklog || legacy.Agente != nil {
		t.Fatalf("la tarea legacy del mismo proyecto deberia volver a backlog: %+v", legacy)
	}

	ajena, err := db.GetTarea(ajenaID)
	if err != nil {
		t.Fatalf("get ajena: %v", err)
	}
	if ajena.Estado != db.TareaBacklog || ajena.Agente != nil {
		t.Fatalf("la tarea de otro proyecto deberia volver a backlog: %+v", ajena)
	}
}

func TestProcesarCompactacionExclusividadPremiumSesionActivaRespetaSemillaYManualTakeover(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	frontierID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Semilla premium",
		Descripcion: "Abrir siguiente frente premium útil",
		ProyectoID:  &proyectoID,
		Modulo:      "autonomia",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       "autonomia:premium_frontier",
	})
	if err != nil {
		t.Fatalf("crear semilla: %v", err)
	}
	if err := db.TomarTarea(frontierID, "Codex1"); err != nil {
		t.Fatalf("tomar semilla: %v", err)
	}

	manualID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Tarea manual",
		Descripcion: "Trabajo fuera de flota automatizada",
		ProyectoID:  &proyectoID,
		Modulo:      "manual",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
		Notas:       "Asumida manualmente fuera de la flota automatizada",
	})
	if err != nil {
		t.Fatalf("crear tarea manual: %v", err)
	}
	if err := db.TomarTarea(manualID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea manual: %v", err)
	}

	n, err := procesarCompactacionExclusividadPremiumSesionActiva(sesion, nil)
	if err != nil {
		t.Fatalf("compactar exclusividad: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia mover ni semilla ni takeover manual, got=%d", n)
	}
}

func TestProcesarCompactacionFrentesPremiumSesionActivaCompactaBloqueadasDuplicadas(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("claude1", "programador"); err != nil {
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
		Agente:      "claude1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "claude-code",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	keepID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente premium bloqueado canónico",
		Descripcion: "WRITE_SET: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestProcesarCompactacionFrentesPremiumSesionActivaCompactaBloqueadasDuplicadas$'",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear keep: %v", err)
	}
	if err := db.TomarTarea(keepID, "claude1"); err != nil {
		t.Fatalf("tomar keep: %v", err)
	}
	if err := db.BloquearTarea(keepID, "orquesta", "Bloqueada automáticamente por degradación operativa sin relevo sano"); err != nil {
		t.Fatalf("bloquear keep: %v", err)
	}

	dupID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente premium bloqueado duplicado",
		Descripcion: "WRITE_SET: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestProcesarCompactacionFrentesPremiumSesionActivaCompactaBloqueadasDuplicadas$'",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear dup: %v", err)
	}
	if err := db.TomarTarea(dupID, "claude1"); err != nil {
		t.Fatalf("tomar dup: %v", err)
	}
	if err := db.BloquearTarea(dupID, "orquesta", "Bloqueada automáticamente por degradación operativa sin relevo sano"); err != nil {
		t.Fatalf("bloquear dup: %v", err)
	}

	n, err := procesarCompactacionFrentesPremiumSesionActiva(sesion, nil)
	if err != nil {
		t.Fatalf("compactar frentes: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia compactar una bloqueada duplicada, got=%d", n)
	}

	keep, _ := db.GetTarea(keepID)
	dup, _ := db.GetTarea(dupID)
	mantenidas := 0
	backlog := 0
	for _, tarea := range []*db.Tarea{keep, dup} {
		switch tarea.Estado {
		case db.TareaBloqueada:
			if tarea.Agente == nil || !strings.EqualFold(strings.TrimSpace(*tarea.Agente), "claude1") {
				t.Fatalf("la bloqueada retenida debe seguir asignada a claude1: %+v", tarea)
			}
			mantenidas++
		case db.TareaBacklog:
			if tarea.Agente != nil {
				t.Fatalf("la duplicada compactada debe volver a backlog sin agente: %+v", tarea)
			}
			backlog++
		default:
			t.Fatalf("estado inesperado tras compactar: %+v", tarea)
		}
	}
	if mantenidas != 1 || backlog != 1 {
		t.Fatalf("debería quedar exactamente una bloqueada y una en backlog, mantenidas=%d backlog=%d", mantenidas, backlog)
	}
}

func TestProcesarCompactacionExclusividadPremiumBatchMueveRestosSinSesionViva(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("claude1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	orqID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto orquestador: %v", err)
	}
	otroID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "api",
		Nombre:  "API",
		RutaAbs: filepath.Join(tmp, "api"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto api: %v", err)
	}
	if err := db.ActivarAsignacion("claude1", orqID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion exclusiva: %v", err)
	}

	keepID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente premium acotado",
		Descripcion: "WRITE_SET: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestProcesarCompactacionExclusividadPremiumBatchMueveRestosSinSesionViva$'",
		ProyectoID:  &orqID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea keep: %v", err)
	}
	if err := db.TomarTarea(keepID, "claude1"); err != nil {
		t.Fatalf("tomar keep: %v", err)
	}

	legacyID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Tarea legacy",
		Descripcion: "Sin contrato acotado",
		ProyectoID:  &orqID,
		Modulo:      "legacy",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea legacy: %v", err)
	}
	if err := db.TomarTarea(legacyID, "claude1"); err != nil {
		t.Fatalf("tomar legacy: %v", err)
	}

	ajenaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Tarea ajena",
		Descripcion: "No debe seguir en un premium exclusivo ajeno",
		ProyectoID:  &otroID,
		Modulo:      "api",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea ajena: %v", err)
	}
	if err := db.TomarTarea(ajenaID, "claude1"); err != nil {
		t.Fatalf("tomar ajena: %v", err)
	}

	rows := []agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "claude1"},
		Asignacion:      &db.Asignacion{Agente: "claude1", ProyectoID: orqID, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva, Nota: "microciclo_exclusivo"},
		EstadoOperativo: "bloqueado_por_runtime",
	}}
	tareasActivasPorAgente := map[string][]*db.Tarea{
		"claude1": {
			{ID: keepID, Agente: ptrString("claude1"), Estado: db.TareaAsignada},
			{ID: legacyID, Agente: ptrString("claude1"), Estado: db.TareaAsignada},
			{ID: ajenaID, Agente: ptrString("claude1"), Estado: db.TareaAsignada},
		},
	}

	n, err := procesarCompactacionExclusividadPremiumBatch(rows, tareasActivasPorAgente)
	if err != nil {
		t.Fatalf("compactar batch: %v", err)
	}
	if n != 2 {
		t.Fatalf("deberia mover dos restos a backlog, got=%d", n)
	}

	keep, _ := db.GetTarea(keepID)
	if keep.Estado != db.TareaAsignada || keep.Agente == nil || !strings.EqualFold(strings.TrimSpace(*keep.Agente), "claude1") {
		t.Fatalf("el frente bounded deberia mantenerse: %+v", keep)
	}
	legacy, _ := db.GetTarea(legacyID)
	if legacy.Estado != db.TareaBacklog || legacy.Agente != nil {
		t.Fatalf("la legacy deberia volver a backlog: %+v", legacy)
	}
	ajena, _ := db.GetTarea(ajenaID)
	if ajena.Estado != db.TareaBacklog || ajena.Agente != nil {
		t.Fatalf("la ajena deberia volver a backlog: %+v", ajena)
	}
}

func TestProcesarDecisionPausaSesionActivaAutonomiaUsaStopCuandoLaAsignacionCambioDeProyecto(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "multi-app",
		Nombre:  "Multi App",
		RutaAbs: filepath.Join(tmp, "multi-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	proyecto, err := db.GetProyecto("multi-app")
	if err != nil || proyecto == nil {
		t.Fatalf("get proyecto: %+v err=%v", proyecto, err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "multi-app"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarDecisionPausaSesionActivaAutonomia(sesion, proyecto, agenteTickOutput{
		AccionRecomendada: "pausar_y_reasignar",
		Motivo:            "La asignación activa del agente ha cambiado al proyecto orquestador",
	}, nil)
	if err != nil {
		t.Fatalf("procesar decision pausa: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia encolar una orden de control, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != agenteControlAccionStop {
		t.Fatalf("deberia encolar stop al cambiar la asignacion de proyecto: %+v", orders)
	}
}

func TestProcesarRuntimesFueraDeAsignacionActivaBatchEncolaStopParaProyectoViejo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	orqID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert orquestador: %v", err)
	}
	multiID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "multi-app",
		Nombre:  "Multi App",
		RutaAbs: filepath.Join(tmp, "multi-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert multi-app: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", orqID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}

	rows := []agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "Codex1"},
		Asignacion:      &db.Asignacion{Agente: "Codex1", ProyectoID: orqID, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva, Nota: "microciclo_exclusivo"},
		Runtime:         &db.RuntimeInstance{ProyectoID: &multiID, LogicalState: "activo"},
		Handle:          &db.RuntimeHandle{ProyectoID: &multiID, Estado: "activo"},
		EstadoOperativo: "trabajando",
	}}

	n, err := procesarRuntimesFueraDeAsignacionActivaBatch(rows, time.Now().UTC())
	if err != nil {
		t.Fatalf("procesar runtimes fuera de asignacion: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia encolar un stop para el proyecto viejo, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &multiID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != agenteControlAccionStop {
		t.Fatalf("deberia encolar stop sobre el runtime del proyecto viejo: %+v", orders)
	}
}

func TestProcesarRuntimesFueraDeAsignacionActivaPorAsignacionBatchEncolaStopParaRuntimeInvisibleEnRows(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	orqID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert orquestador: %v", err)
	}
	multiID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "multi-app",
		Nombre:  "Multi App",
		RutaAbs: filepath.Join(tmp, "multi-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert multi-app: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", orqID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &multiID,
		CWD:         filepath.Join(tmp, "multi-app"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	runtimeActual, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtimeActual == nil {
		t.Fatalf("get runtime: %+v err=%v", runtimeActual, err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='activo', process_state='running' WHERE id=?`, runtimeActual.ID); err != nil {
		t.Fatalf("activar runtime: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='activo' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("activar handle: %v", err)
	}

	n, err := procesarRuntimesFueraDeAsignacionActivaPorAsignacionBatch()
	if err != nil {
		t.Fatalf("procesar runtimes fuera de asignacion por asignacion: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia encolar un stop para el runtime invisible en rows, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &multiID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != agenteControlAccionStop {
		t.Fatalf("deberia encolar stop sobre el proyecto viejo aunque no venga en rows: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchProcesaSesionesMultiplesDelMismoAgente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	orqID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert orquestador: %v", err)
	}
	multiID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "multi-app",
		Nombre:  "Multi App",
		RutaAbs: filepath.Join(tmp, "multi-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert multi-app: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", orqID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &orqID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion orquestador: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &multiID,
		CWD:         filepath.Join(tmp, "multi-app"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion multi-app: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia batch: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia actuar sobre la sesion stale del mismo agente, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &multiID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	foundStop := false
	for _, order := range orders {
		if order.Tipo == agenteControlAccionStop {
			foundStop = true
			break
		}
	}
	if !foundStop {
		t.Fatalf("deberia encolar stop para la sesion stale del proyecto viejo: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchTimeoutSesionNoBloqueaDegradados(t *testing.T) {
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
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	prevSesionFn := procesarAutonomiaSesionActivaBatchFn
	prevDegradadosFn := procesarAgentesDegradadosAutonomiaBatchFn
	prevTimeout := autonomiaSessionStepTimeoutOverride
	t.Cleanup(func() {
		procesarAutonomiaSesionActivaBatchFn = prevSesionFn
		procesarAgentesDegradadosAutonomiaBatchFn = prevDegradadosFn
		autonomiaSessionStepTimeoutOverride = prevTimeout
	})

	autonomiaSessionStepTimeoutOverride = 10 * time.Millisecond
	procesarAutonomiaSesionActivaBatchFn = func(_ *db.Sesion, _ *autonomiaBatchSnapshot) (int, error) {
		time.Sleep(50 * time.Millisecond)
		return 0, nil
	}
	degradadosLlamados := 0
	procesarAgentesDegradadosAutonomiaBatchFn = func() (int, error) {
		degradadosLlamados++
		return 0, nil
	}

	if _, err := procesarAutonomiaAgentesBatch(); err != nil {
		t.Fatalf("procesar autonomia batch: %v", err)
	}
	if degradadosLlamados != 1 {
		t.Fatalf("el batch de degradados deberia ejecutarse aunque una sesion timeout, got=%d", degradadosLlamados)
	}
}

func TestFiltrarSesionesAutonomiaRelevantesDescartaSesionesNoActivasRealesYDeduplica(t *testing.T) {
	proyectoID := int64(3)
	base := time.Date(2026, 4, 15, 9, 0, 0, 0, time.UTC)
	sesiones := []*db.Sesion{
		{ID: 1, Agente: "Codex1", ProyectoID: &proyectoID, Activa: true, Estado: "activa", Inicio: base},
		{ID: 2, Agente: "Codex1", ProyectoID: &proyectoID, Activa: true, Estado: "activa", Inicio: base.Add(1 * time.Minute)},
		{ID: 3, Agente: "Codex1", ProyectoID: &proyectoID, Activa: true, Estado: "cerrada", Inicio: base.Add(2 * time.Minute)},
		{ID: 4, Agente: "Gemini1", ProyectoID: &proyectoID, Activa: true, Estado: "fallida", Inicio: base.Add(3 * time.Minute)},
		{ID: 5, Agente: "claude1", ProyectoID: &proyectoID, Activa: true, Estado: "pausada", Inicio: base.Add(4 * time.Minute)},
		{ID: 6, Agente: "sinproyecto", Activa: true, Estado: "activa", Inicio: base.Add(5 * time.Minute)},
	}

	filtradas := filtrarSesionesAutonomiaRelevantes(sesiones)
	if len(filtradas) != 2 {
		t.Fatalf("deberia conservar solo sesiones relevantes deduplicadas, got=%d", len(filtradas))
	}
	ids := map[string]int64{}
	for _, sesion := range filtradas {
		ids[sesion.Agente] = sesion.ID
	}
	if ids["Codex1"] != 2 {
		t.Fatalf("deberia quedarse con la sesion mas reciente por agente/proyecto para Codex1: %+v", filtradas)
	}
	if ids["claude1"] != 5 {
		t.Fatalf("deberia conservar la sesion pausada relevante de claude1: %+v", filtradas)
	}
}

func TestRowProyectoIDPreferidoPriorizaAsignacionActivaSobreRuntimeViejo(t *testing.T) {
	asignado := int64(10)
	runtimeViejo := int64(20)
	row := agentesapp.Row{
		Asignacion: &db.Asignacion{ProyectoID: asignado, Estado: db.AsignacionActiva},
		Runtime:    &db.RuntimeInstance{ProyectoID: &runtimeViejo},
		Handle:     &db.RuntimeHandle{ProyectoID: &runtimeViejo},
	}

	proyectoID := rowProyectoIDPreferido(row)
	if proyectoID == nil || *proyectoID != asignado {
		t.Fatalf("deberia priorizar el proyecto de la asignacion activa, got=%v", proyectoID)
	}
}

func TestRowProyectoIDPreferidoUsaRuntimeSiNoHayAsignacion(t *testing.T) {
	runtimeID := int64(20)
	row := agentesapp.Row{
		Runtime: &db.RuntimeInstance{ProyectoID: &runtimeID},
		Handle:  &db.RuntimeHandle{ProyectoID: &runtimeID},
	}

	proyectoID := rowProyectoIDPreferido(row)
	if proyectoID == nil || *proyectoID != runtimeID {
		t.Fatalf("deberia usar el runtime si no hay asignacion, got=%v", proyectoID)
	}
}

func TestProcesarAutonomiaAgentesBatchNoEncolaNudgePorSupervisarProyectoEnSesionActiva(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "Codex1",
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia forzar nudge de supervisor en sesion activa, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear nudge supervisor pasivo: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchNoRepiteNudgeRecienteMaterializado(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Seguir implementando",
		Descripcion: "Trabajo activo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	resetAt := time.Now().UTC().Add(2 * time.Hour)
	remaining := int64(10)
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:          sesion.ID,
		WindowKind:        "5h",
		ResetAt:           &resetAt,
		RemainingMessages: &remaining,
		BudgetSource:      "test",
		CheckedAt:         time.Now().UTC(),
	}); err != nil {
		t.Fatalf("registrar presupuesto: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia inicial: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia forzar nudge pasivo inicial, got=%d", n)
	}
	if _, err := db.ProcesarRuntimeOrdersBatch(); err != nil {
		t.Fatalf("procesar runtime orders: %v", err)
	}
	if _, err := procesarRuntimeMailboxBatch(); err != nil {
		t.Fatalf("procesar runtime mailbox: %v", err)
	}

	n, err = procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia repetida: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia repetir el nudge reciente, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var nudges, sendInstructions int
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch order.Tipo {
		case "nudge":
			nudges++
		case "send_instruction":
			sendInstructions++
			if order.HandleID == nil || *order.HandleID != handle.ID {
				t.Fatalf("send_instruction sin handle esperado: %+v", order)
			}
		}
	}
	if nudges != 0 || sendInstructions != 0 {
		t.Fatalf("ordenes de autonomia duplicadas: nudges=%d send_instruction=%d orders=%+v", nudges, sendInstructions, orders)
	}
}

func TestResetReanimacionEncolaResumeCuandoHayHandlePausado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()
	statusCacheState.mu.Lock()
	statusCacheState.ok = true
	statusCacheState.value = apiStatusResponse{Generado: "stale"}
	statusCacheState.expires = time.Now().UTC().Add(time.Minute)
	statusCacheState.mu.Unlock()

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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-remote",
		Nombre:     "Codex Remote",
		Transporte: "api",
		Comando:    "http://localhost:9999",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar tras enfriamiento",
		Descripcion: "Trabajo activo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	resetAt := time.Now().UTC().Add(2 * time.Hour)
	remaining := int64(10)
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:          sesion.ID,
		WindowKind:        "5h",
		ResetAt:           &resetAt,
		RemainingMessages: &remaining,
		BudgetSource:      "test",
		CheckedAt:         time.Now().UTC(),
	}); err != nil {
		t.Fatalf("registrar presupuesto: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"tmux_cli_session","tmux_session":"orq-codex1-resume","mailbox_delivery_mode":"session_resume","external_session_id":"sess-resume-1","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"session_resume"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado', metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, capsJSON, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}
	if err := reactivarAgenteTrasReanimacion("Codex1"); err != nil {
		t.Fatalf("reactivar agente: %v", err)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("reactivacion no encolada: %+v", orders)
	}
	if orders[0].Tipo != "resume" && orders[0].Tipo != "start" {
		t.Fatalf("tipo de reactivacion inesperado: %+v", orders[0])
	}
	if !strings.Contains(orders[0].PayloadJSON, `"accion":"resume"`) &&
		!strings.Contains(orders[0].PayloadJSON, `"accion":"start"`) {
		t.Fatalf("payload de reactivacion inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestResetReanimacionEncolaStartSiHandlePausadoTMUXBootstrapOnly(t *testing.T) {
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
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar worker tmux pausado",
		Descripcion: "Trabajo activo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-codex1-bootstrap","mailbox_delivery_mode":"bootstrap_only","can_send_input":false}`)
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado', metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, capsJSON, handle.ID); err != nil {
		t.Fatalf("pausar handle tmux: %v", err)
	}
	pendienteAntes, err := existeRuntimeOrderAbiertaAutonomia("Codex1", &proyectoID, "resume", "start", "handoff")
	if err != nil {
		t.Fatalf("preflight open orders: %v", err)
	}
	if err := reactivarAgenteTrasReanimacion("Codex1"); err != nil {
		t.Fatalf("reactivar agente: %v", err)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		handleReactivacion, _ := resolverHandleReactivacionAgente("Codex1", &proyectoID)
		todas, _ := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
		t.Fatalf("start no encolado para tmux bootstrap_only pausado: open_before=%t pendientes=%+v todas=%+v handle=%+v", pendienteAntes, orders, todas, handleReactivacion)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"accion":"start"`) {
		t.Fatalf("payload start inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestResetReanimacionEncolaStartSiHandleFallidoConMailboxPendiente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"tmux_cli_session","tmux_session":"orq-gemini1-restart","mailbox_delivery_mode":"interactive","can_send_input":true}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='fallido', metadata_json=? WHERE id=?`, metaJSON, handle.ID); err != nil {
		t.Fatalf("fallar handle: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Gemini1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"reanuda trabajo premium"}`,
	}); err != nil {
		t.Fatalf("mailbox pendiente: %v", err)
	}

	if err := reactivarAgenteTrasReanimacion("Gemini1"); err != nil {
		t.Fatalf("reactivar agente: %v", err)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("start no encolado para handle fallido con mailbox pendiente: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"accion":"start"`) {
		t.Fatalf("payload start inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestReactivarAgenteTrasReanimacionOmitePoolLocalCompartidoSinCapacidad(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("GemmaBusy", "programador"); err != nil {
		t.Fatalf("registrar agente ocupado: %v", err)
	}
	if err := db.RegistrarAgente("GemmaNuevo", "programador"); err != nil {
		t.Fatalf("registrar agente nuevo: %v", err)
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
	if _, err := db.GuardarPool(&db.PoolCapacidad{
		Slug:                "ollama-gemma4",
		Proveedor:           "Ollama",
		Runtime:             "ollama",
		Plan:                "local",
		CapacidadTotal:      1,
		PermiteModelosMulti: true,
		PoliticaHandoff:     "preventivo",
		FuenteTelemetria:    "manual",
		MetadataJSON:        `{"conector_canonico":"ollama_pool_local","modelo_preferente":"gemma4:26b","slots_maximos":1}`,
		Activo:              true,
	}); err != nil {
		t.Fatalf("guardar pool: %v", err)
	}
	if _, err := db.GuardarPoolModelo("ollama-gemma4", &db.PoolModelo{ModelSlug: "gemma4:26b", Activo: true, Prioridad: 10, CosteRelativo: 1}); err != nil {
		t.Fatalf("guardar pool modelo: %v", err)
	}
	if _, err := db.GuardarPoliticaModelo(&db.PoliticaModelo{
		ScopeTipo:       "perfil",
		ScopeRef:        "implementacion",
		PerfilTarea:     "implementacion",
		PoolSlug:        "ollama-gemma4",
		ModelSlug:       "gemma4:26b",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
	}); err != nil {
		t.Fatalf("guardar politica modelo: %v", err)
	}
	sesionBusy, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "GemmaBusy",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "gemma-busy"),
		Herramienta: "ollama_pool_local",
	})
	if err != nil {
		t.Fatalf("iniciar sesion busy: %v", err)
	}
	pool, err := db.GetPool("ollama-gemma4")
	if err != nil {
		t.Fatalf("get pool: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE sesiones SET pool_id = ? WHERE id = ?`, pool.ID, sesionBusy.ID); err != nil {
		t.Fatalf("asignar pool a sesion busy: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Reactivar GemmaNuevo",
		Descripcion: "Debe esperar slot libre del pool local compartido",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "GemmaNuevo"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "GemmaNuevo"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	if err := reactivarAgenteTrasReanimacion("GemmaNuevo"); err != nil {
		t.Fatalf("reactivar agente: %v", err)
	}

	agente := "GemmaNuevo"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia encolar start si el pool local compartido ya no tiene slots: %+v", orders)
	}
}

func TestProcesarRuntimeMailboxInteractivoBatchOmiteHandlesSinInputInteractivo(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false}`,
		`{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","can_send_input":false}`,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"seguir frente"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxInteractivoBatch()
	if err != nil {
		t.Fatalf("procesar mailbox interactivo: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia crear send_instruction interactiva, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia crear send_instruction para handle sin input interactivo: %+v", order)
		}
	}

	estado := "pendiente"
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(mailbox) != 1 || mailbox[0].ID != msgID {
		t.Fatalf("mailbox pendiente inesperada: %+v", mailbox)
	}
}

func TestProcesarRuntimeMailboxInteractivoBatchEsperaTMUXStarting(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Ollama1", "programador"); err != nil {
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
		Agente:      "Ollama1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "ollama-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
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
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-ollama1-starting/%4', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":true,"mailbox_delivery_mode":"interactive"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Ollama1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"implementa solo la funcion pedida"}`,
	}); err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxInteractivoBatch()
	if err != nil {
		t.Fatalf("procesar mailbox interactivo: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia crear send_instruction mientras tmux sigue starting, got=%d", n)
	}

	agente := "Ollama1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia encolar send_instruction con worker starting: %+v", order)
		}
	}
}

func TestProcesarRuntimeMailboxSupervisorLocalBatchEncolaSendInstructionParaCodexSupervisado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()
	t.Setenv("ORQUESTA_ALLOW_LEGACY_SUPERVISOR_HOT_INPUT", "1")

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","stdin_path":"` + filepath.Join(tmp, "pty.stdin") + `","supervisor_ref":"` + filepath.Join(tmp, "supervisor.ref") + `","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-supervisor-local","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryBootstrapOnly + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"seguir frente"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxSupervisorLocalBatch()
	if err != nil {
		t.Fatalf("procesar mailbox supervisor local: %v", err)
	}
	if n != 0 {
		t.Fatalf("el batch supervisor_local legacy ya no deberia materializar send_instruction, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia crear send_instruction por supervisor_local legacy: %+v", order)
		}
	}

	pendiente := "pendiente"
	mailboxPendiente, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailboxPendiente) != 1 || mailboxPendiente[0].ID != msgID {
		t.Fatalf("la mailbox durable debe seguir pendiente hasta entrega real: %+v", mailboxPendiente)
	}
}

func TestProcesarRuntimeMailboxBatchCoordinaRestartParaHandlesBootstrapOnly(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET transporte='cli', handle_kind='process', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false,"mailbox_delivery_mode":"coordinated_restart"}`,
		`{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","can_send_input":false,"mailbox_delivery_mode":"coordinated_restart"}`,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgViejo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"mensaje viejo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox vieja: %v", err)
	}
	msgNuevo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"mensaje nuevo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox nueva: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox batch: %v", err)
	}
	if n != 0 {
		t.Fatalf("codex-cli no deberia reiniciarse por mailbox rutinario, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear ordenes de restart: %+v", orders)
	}

	pendiente := "pendiente"
	consumido := "consumido"
	mailboxPendiente, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailboxPendiente) != 1 || mailboxPendiente[0].ID != msgNuevo {
		t.Fatalf("deberia quedar solo la mailbox mas reciente pendiente: %+v", mailboxPendiente)
	}
	mailboxConsumido, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &consumido})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	if len(mailboxConsumido) != 1 || mailboxConsumido[0].ID != msgViejo {
		t.Fatalf("mailbox consumida inesperada: %+v", mailboxConsumido)
	}
}

func TestProcesarRuntimeMailboxBatchNoDuplicaReinicioCoordinadoAbierto(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET transporte='cli', handle_kind='process', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false,"mailbox_delivery_mode":"coordinated_restart"}`,
		`{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","can_send_input":false,"mailbox_delivery_mode":"coordinated_restart"}`,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"seguir"}`,
	}); err != nil {
		t.Fatalf("mailbox: %v", err)
	}
	if _, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		HandleID:    &handle.ID,
		Tipo:        agenteControlAccionStop,
		PayloadJSON: `{"accion":"stop","proyecto":"orquestador","motivo":"ya pendiente"}`,
	}); err != nil {
		t.Fatalf("encolar stop: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox batch: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia duplicar reinicio coordinado, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != agenteControlAccionStop {
		t.Fatalf("ordenes pendientes inesperadas: %+v", orders)
	}
}

func TestProcesarRuntimeMailboxBatchCoordinaRestartParaTMUXBootstrapOnly(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexTMUX", "programador"); err != nil {
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
		Agente:      "CodexTMUX",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET transporte='cli', handle_kind='process', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`,
		`{"driver":"tmux_cli_session","tmux_session":"orq-codextmux-1","tmux_pane_id":"%1","can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgViejo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "CodexTMUX",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"mensaje viejo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox vieja: %v", err)
	}
	msgNuevo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "CodexTMUX",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"mensaje nuevo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox nueva: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("el worker tmux bootstrap_only deberia coordinar restart, got=%d", n)
	}

	agente := "CodexTMUX"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("deberia crear stop/start coordinados, got=%d %+v", len(orders), orders)
	}
	tipos := map[string]bool{}
	for _, order := range orders {
		if order != nil {
			tipos[order.Tipo] = true
		}
	}
	if !tipos["stop"] || !tipos["start"] {
		t.Fatalf("ordenes coordinadas inesperadas: %+v", orders)
	}

	pendiente := "pendiente"
	consumido := "consumido"
	mailboxPendiente, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailboxPendiente) != 1 || mailboxPendiente[0].ID != msgNuevo {
		t.Fatalf("deberia quedar solo la mailbox mas reciente pendiente: %+v", mailboxPendiente)
	}
	mailboxConsumido, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &consumido})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	if len(mailboxConsumido) != 1 || mailboxConsumido[0].ID != msgViejo {
		t.Fatalf("mailbox consumida inesperada: %+v", mailboxConsumido)
	}
}

func TestRuntimeHandleRequiereCoordinatedRestartMailboxAceptaTMUXCanonicoBootstrapOnly(t *testing.T) {
	handle := &db.RuntimeHandle{
		Transporte:       "tmux",
		HandleKind:       "session",
		MetadataJSON:     `{"driver":"tmux_cli_session","mailbox_delivery_mode":"bootstrap_only"}`,
		CapabilitiesJSON: `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`,
	}
	if !runtimeHandleRequiereCoordinatedRestartMailbox(handle) {
		t.Fatalf("un worker tmux bootstrap_only debe coordinar restart aunque el handle ya sea canónico")
	}
}

func TestRuntimeHandleRequiereCoordinatedRestartMailboxNoAplicaSiTMUXYaTieneSesionExterna(t *testing.T) {
	handle := &db.RuntimeHandle{
		Transporte:       "tmux",
		HandleKind:       "session",
		MetadataJSON:     `{"driver":"tmux_cli_session","mailbox_delivery_mode":"bootstrap_only","external_session_id":"sess-live-tmux"}`,
		CapabilitiesJSON: `{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`,
	}
	if runtimeHandleRequiereCoordinatedRestartMailbox(handle) {
		t.Fatalf("un worker tmux con external_session_id ya no deberia coordinar restart")
	}
}

func TestProcesarRuntimeMailboxBatchEncolaSendInstructionParaTMUXBootstrapOnlyReady(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexTMUX", "programador"); err != nil {
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
		Agente:      "CodexTMUX",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-ready")
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
		"tmux_session":          "orq-codextmux-ready",
		"tmux_pane_id":          "%1",
		"mailbox_delivery_mode": "bootstrap_only",
		"can_send_input":        false,
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "ready",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"ready_at":              now.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": "bootstrap_only",
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
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-codextmux-ready","tmux_pane_id":"%%1","can_send_input":false,"mailbox_delivery_mode":"bootstrap_only","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, manifestPath, statusPath, heartbeatPath)
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET transporte='cli', handle_kind='process', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	handle, err = db.GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}
	if !runtimeHandleListaParaDispatchBootstrapTMUX(handle) {
		t.Fatalf("el handle deberia estar listo para bootstrap tmux: %+v", handle)
	}

	msgViejo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "CodexTMUX",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"mensaje viejo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox vieja: %v", err)
	}
	msgNuevo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "CodexTMUX",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"mensaje nuevo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox nueva: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox batch: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia materializar send_instruction tmux ready, got=%d", n)
	}

	agente := "CodexTMUX"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	foundSend := false
	for _, order := range orders {
		if order == nil {
			continue
		}
		if order.Tipo == "send_instruction" {
			foundSend = true
			if !strings.Contains(order.PayloadJSON, `"mailbox_id":`+strconv.FormatInt(msgNuevo, 10)) {
				t.Fatalf("payload send_instruction inesperado: %s", order.PayloadJSON)
			}
			continue
		}
		if order.Tipo == "stop" || order.Tipo == "start" {
			t.Fatalf("no deberia coordinar restart cuando el worker tmux esta ready: %+v", orders)
		}
	}
	if !foundSend {
		t.Fatalf("deberia crear una send_instruction pendiente, got=%+v", orders)
	}

	pendiente := "pendiente"
	consumido := "consumido"
	mailboxPendiente, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailboxPendiente) != 1 || mailboxPendiente[0].ID != msgNuevo {
		t.Fatalf("deberia quedar solo la mailbox mas reciente pendiente: %+v", mailboxPendiente)
	}
	mailboxConsumido, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &consumido})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	if len(mailboxConsumido) != 1 || mailboxConsumido[0].ID != msgViejo {
		t.Fatalf("mailbox consumida inesperada: %+v", mailboxConsumido)
	}
}

func TestProcesarRuntimeMailboxBatchNoCoordinaRestartParaTMUXBootstrapOnlyRunning(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	workerDir := filepath.Join(tmp, "worker-running")
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
		"tmux_session":          "orq-gemini1-running",
		"tmux_pane_id":          "%1",
		"mailbox_delivery_mode": "bootstrap_only",
		"can_send_input":        false,
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":                 "running",
		"alive":                 true,
		"updated_at":            now.Format(time.RFC3339Nano),
		"mailbox_delivery_mode": "bootstrap_only",
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
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
	metaJSON := fmt.Sprintf(`{"driver":"tmux_cli_session","tmux_session":"orq-gemini1-running","tmux_pane_id":"%%1","can_send_input":false,"mailbox_delivery_mode":"bootstrap_only","worker_manifest_path":"%s","worker_status_path":"%s","worker_heartbeat_path":"%s"}`, manifestPath, statusPath, heartbeatPath)
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false,"mailbox_delivery_mode":"bootstrap_only"}`,
		metaJSON,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Gemini1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"mensaje nuevo"}`,
	}); err != nil {
		t.Fatalf("mailbox nueva: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox batch: %v", err)
	}
	if n != 0 {
		t.Fatalf("un worker tmux bootstrap_only running no deberia coordinar restart, got=%d", n)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && (order.Tipo == "stop" || order.Tipo == "start") {
			t.Fatalf("no deberia crear stop/start coordinados cuando el worker tmux esta running: %+v", orders)
		}
	}
}

func TestEncolarSendInstructionDesdeRuntimeMailboxConservaMetadataMicroprogramacion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Ollama1", "programador"); err != nil {
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
		Agente:      "Ollama1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "ollama-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-ollama1/%4', metadata_json='{"driver":"tmux_cli_session","tmux_session":"orq-ollama1","tmux_pane_id":"%4","can_send_input":true,"mailbox_delivery_mode":"interactive"}', capabilities_json='{"can_send_input":true,"mailbox_delivery_mode":"interactive"}' WHERE id=?`,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	handle, err = db.GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Ollama1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"source":"microprogramacion","microprogramacion":{"especificacion_id":7,"write_set":["identidad/normalizar.go"]},"texto":"PATCH_UNIFICADO: cambia solo identidad/normalizar.go"}`,
	})
	if err != nil {
		t.Fatalf("enviar runtime mailbox: %v", err)
	}
	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox: %+v err=%v", msg, err)
	}

	orderID, err := encolarSendInstructionDesdeRuntimeMailbox(msg, handle, "PATCH_UNIFICADO: cambia solo identidad/normalizar.go", "")
	if err != nil {
		t.Fatalf("encolar send_instruction desde mailbox: %v", err)
	}
	order, err := db.GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get order: %+v err=%v", order, err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(order.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode payload order: %v", err)
	}
	if got := strings.TrimSpace(stringMapValue(payload, "source")); got != "microprogramacion" {
		t.Fatalf("source perdida en materializacion: %q payload=%s", got, order.PayloadJSON)
	}
	micro, _ := payload["microprogramacion"].(map[string]any)
	if id, ok := micro["especificacion_id"].(float64); !ok || int64(id) != 7 {
		t.Fatalf("metadata microprogramacion perdida: %+v payload=%s", micro, order.PayloadJSON)
	}
	if id, ok := payload["mailbox_id"].(float64); !ok || int64(id) != msgID {
		t.Fatalf("mailbox_id inesperado: %s", order.PayloadJSON)
	}
}

func TestConstruirInstruccionMailboxInteractivoPipelineLocalPrefiereInstruction(t *testing.T) {
	msg := &db.RuntimeMailboxMessage{
		Kind:        "pipeline_local",
		PayloadJSON: `{"texto":"microrefactor_loop","instruction":"FASE: implementacion\nTAREA: #533 Micro-refactorización cíclica del control plane"}`,
	}
	texto, ok := construirInstruccionMailboxInteractivo(msg)
	if !ok {
		t.Fatal("deberia construir instruccion para pipeline_local")
	}
	if !strings.Contains(texto, "TAREA: #533") || strings.Contains(texto, "microrefactor_loop") {
		t.Fatalf("pipeline_local deberia priorizar instruction real, got=%q", texto)
	}
}

func TestEncolarSendInstructionDesdeRuntimeMailboxDespiertaRuntimeOrders(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Gemini1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"slice de prueba"}`,
	})
	if err != nil {
		t.Fatalf("enviar runtime mailbox: %v", err)
	}
	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox: %+v err=%v", msg, err)
	}
	previousWake := wakeRuntimeOrdersAfterMailbox
	wakeCalls := 0
	wakeRuntimeOrdersAfterMailbox = func() bool {
		wakeCalls++
		return true
	}
	defer func() {
		wakeRuntimeOrdersAfterMailbox = previousWake
	}()

	orderID, err := encolarSendInstructionDesdeRuntimeMailbox(msg, handle, "continua trabajo actual", "")
	if err != nil {
		t.Fatalf("encolar send_instruction desde mailbox: %v", err)
	}
	if orderID <= 0 {
		t.Fatalf("order id inesperado: %d", orderID)
	}
	if wakeCalls != 1 {
		t.Fatalf("runtime_orders deberia despertarse exactamente una vez, got=%d", wakeCalls)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchEncolaSendInstructionSinConsumirMailbox(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-batch-resume","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"instruccion session resume"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia crear una sola send_instruction por session_resume, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var send *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			send = order
			break
		}
	}
	if send == nil {
		t.Fatalf("faltaba send_instruction")
	}
	if !strings.Contains(send.PayloadJSON, `"mailbox_id":`+strconv.FormatInt(msgID, 10)) {
		t.Fatalf("send_instruction sin mailbox_id: %s", send.PayloadJSON)
	}
	if !strings.Contains(send.PayloadJSON, `"external_session_id":"sess-batch-resume"`) {
		t.Fatalf("send_instruction sin external_session_id: %s", send.PayloadJSON)
	}

	pendiente := "pendiente"
	mailboxPendiente, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailboxPendiente) != 1 || mailboxPendiente[0].ID != msgID {
		t.Fatalf("la mailbox debe seguir pendiente hasta entregar de verdad: %+v", mailboxPendiente)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchUsaFallbackInteractivoParaProcessPTYSinExternalSession(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","supervisor_ref":"codex1-supervisor","stdin_path":"` + filepath.Join(tmp, "codex.stdin") + `","mailbox_delivery_mode":"session_resume","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='cli', handle_kind='process', capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"haz una micro refactorizacion concreta","texto":"continua trabajo actual"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia materializar una send_instruction por fallback interactivo, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var send *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			send = order
			break
		}
	}
	if send == nil {
		t.Fatal("faltaba send_instruction")
	}
	if strings.Contains(send.PayloadJSON, `"external_session_id":`) {
		t.Fatalf("no deberia incluir external_session_id en el fallback interactivo: %s", send.PayloadJSON)
	}
	if !strings.Contains(send.PayloadJSON, `"mailbox_id":`+strconv.FormatInt(msgID, 10)) {
		t.Fatalf("send_instruction sin mailbox_id: %s", send.PayloadJSON)
	}

	pendiente := "pendiente"
	mailboxPendiente, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailboxPendiente) != 1 || mailboxPendiente[0].ID != msgID {
		t.Fatalf("la mailbox debe seguir pendiente hasta entregar de verdad: %+v", mailboxPendiente)
	}
}

func TestRuntimeHandleAdmiteFallbackInteractivoTransitorioParaProcessPTYConStdin(t *testing.T) {
	handle := &db.RuntimeHandle{
		Transporte:       "cli",
		HandleKind:       "process",
		MetadataJSON:     `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","mailbox_delivery_mode":"session_resume","can_send_input":false,"stdin_path":"/tmp/codex.stdin","supervisor_ref":"/tmp/codex-supervisor"}`,
		CapabilitiesJSON: `{"can_send_input":false,"mailbox_delivery_mode":"session_resume"}`,
	}
	if !runtimeHandleAdmiteFallbackInteractivoTransitorio(handle) {
		t.Fatal("un process_pty_cli con stdin_path deberia degradar a fallback interactivo transitorio")
	}
}

func TestExisteRuntimeOrderAbiertaPorHandleEnSnapshotIgnoraEjecutandoConLeaseCaducado(t *testing.T) {
	proyectoID := int64(7)
	handleID := int64(33)
	expired := time.Now().UTC().Add(-time.Minute)
	msg := &db.RuntimeMailboxMessage{
		ID:         41,
		ToAgente:   "GemmaProgramador4",
		ProyectoID: &proyectoID,
	}
	snapshot := &runtimeMailboxBatchSnapshot{
		ordersByAgentProject: map[string][]*db.RuntimeOrder{
			runtimeMailboxBatchKey("GemmaProgramador4", &proyectoID): {
				{
					ID:             99,
					Agente:         "GemmaProgramador4",
					ProyectoID:     &proyectoID,
					Tipo:           "send_instruction",
					Estado:         "ejecutando",
					HandleID:       &handleID,
					LeaseExpiresAt: &expired,
				},
			},
		},
		ordersLoaded: map[string]struct{}{
			runtimeMailboxBatchKey("GemmaProgramador4", &proyectoID): {},
		},
	}
	got, err := existeRuntimeOrderAbiertaPorHandleEnSnapshot(snapshot, msg, handleID, "send_instruction")
	if err != nil {
		t.Fatalf("existeRuntimeOrderAbiertaPorHandleEnSnapshot: %v", err)
	}
	if got {
		t.Fatalf("una orden ejecutando con lease caducado no deberia bloquear la mailbox")
	}
}

func TestExisteIntentoSendInstructionMailboxParaHandleEnSnapshotIgnoraEjecutandoConLeaseCaducado(t *testing.T) {
	proyectoID := int64(7)
	handleID := int64(33)
	expired := time.Now().UTC().Add(-time.Minute)
	msg := &db.RuntimeMailboxMessage{
		ID:         41,
		ToAgente:   "GemmaProgramador4",
		ProyectoID: &proyectoID,
	}
	handle := &db.RuntimeHandle{ID: handleID}
	snapshot := &runtimeMailboxBatchSnapshot{
		ordersByAgentProject: map[string][]*db.RuntimeOrder{
			runtimeMailboxBatchKey("GemmaProgramador4", &proyectoID): {
				{
					ID:             99,
					Agente:         "GemmaProgramador4",
					ProyectoID:     &proyectoID,
					Tipo:           "send_instruction",
					Estado:         "ejecutando",
					HandleID:       &handleID,
					LeaseExpiresAt: &expired,
					PayloadJSON:    `{"mailbox_id":41}`,
				},
			},
		},
		ordersLoaded: map[string]struct{}{
			runtimeMailboxBatchKey("GemmaProgramador4", &proyectoID): {},
		},
	}
	got, err := existeIntentoSendInstructionMailboxParaHandleEnSnapshot(snapshot, msg, handle, "")
	if err != nil {
		t.Fatalf("existeIntentoSendInstructionMailboxParaHandleEnSnapshot: %v", err)
	}
	if got {
		t.Fatalf("una send_instruction ejecutando con lease caducado no deberia deduplicar el reintento limpio")
	}
}

func TestRuntimeOrderMatchesMailboxSendInstructionAttempt(t *testing.T) {
	handleID := int64(33)
	order := &db.RuntimeOrder{
		Tipo:        "send_instruction",
		HandleID:    &handleID,
		PayloadJSON: `{"mailbox_id":41}`,
	}
	if !runtimeOrderMatchesMailboxSendInstructionAttempt(order, 41, handleID) {
		t.Fatalf("deberia reconocer la send_instruction ligada al mailbox y handle")
	}
	if runtimeOrderMatchesMailboxSendInstructionAttempt(order, 42, handleID) {
		t.Fatalf("no deberia mezclar mailbox distinto")
	}
	if runtimeOrderMatchesMailboxSendInstructionAttempt(order, 41, handleID+1) {
		t.Fatalf("no deberia mezclar handle distinto")
	}
}

func TestRuntimeOrderMatchesMailboxSendInstructionAttemptSinHandleLigado(t *testing.T) {
	order := &db.RuntimeOrder{
		Tipo:        "send_instruction",
		PayloadJSON: `{"mailbox_id":41}`,
	}
	if !runtimeOrderMatchesMailboxSendInstructionAttempt(order, 41, 33) {
		t.Fatalf("deberia mantener la semantica legacy cuando la orden no esta ligada a un handle concreto")
	}
}

func TestRuntimeOrderCompletedMailboxAttemptStillBlocksPorSignature(t *testing.T) {
	now := time.Now().UTC()
	order := &db.RuntimeOrder{
		Estado:        "completada",
		ResultadoJSON: `{"mailbox_only":true}`,
		PayloadJSON:   `{"delivery_attempt_signature":"session_resume|handle:33|session:sess-1"}`,
		UpdatedAt:     now,
	}
	if !runtimeOrderCompletedMailboxAttemptStillBlocks(order, "", "session_resume|handle:33|session:sess-1") {
		t.Fatalf("deberia bloquear cuando la signature coincide")
	}
	if runtimeOrderCompletedMailboxAttemptStillBlocks(order, "", "session_resume|handle:33|session:sess-2") {
		t.Fatalf("no deberia bloquear cuando la signature cambia")
	}
}

func TestRuntimeOrderCompletedMailboxAttemptStillBlocksPorSessionID(t *testing.T) {
	now := time.Now().UTC()
	order := &db.RuntimeOrder{
		Estado:        "completada",
		ResultadoJSON: `{"mailbox_only":true}`,
		PayloadJSON:   `{"external_session_id":"sess-1"}`,
		UpdatedAt:     now,
	}
	if !runtimeOrderCompletedMailboxAttemptStillBlocks(order, "sess-1", "") {
		t.Fatalf("deberia bloquear cuando la external session coincide")
	}
	if runtimeOrderCompletedMailboxAttemptStillBlocks(order, "sess-2", "") {
		t.Fatalf("no deberia bloquear cuando la external session cambia")
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchUsaRuntimePrincipalCanonicoSiHandleFreshNoEstaEnlazado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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

	resSesion, err := db.DB.Exec(`INSERT INTO sesiones (
		agente, proyecto_id, activa, estado, herramienta, external_session_id, host, heartbeat_at
	) VALUES (?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		"Codex1", proyectoID, 1, "activa", "codex-cli", "sess-canonico", "test-host")
	if err != nil {
		t.Fatalf("insert sesion: %v", err)
	}
	sesionID, _ := resSesion.LastInsertId()

	pid := int64(os.Getpid())
	runtimeID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:            "Codex1",
		ProyectoID:        &proyectoID,
		SesionID:          &sesionID,
		Connector:         "codex-cli",
		ExternalSessionID: "sess-canonico",
		LogicalState:      "esperando_io",
		ProcessState:      "running",
		PID:               &pid,
		Model:             "gpt-5.4",
	})
	if err != nil {
		t.Fatalf("registrar runtime tmux: %v", err)
	}

	runDir := filepath.Join(tmp, "runtime", "Codex1", "run-session-resume")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	writeJSON := func(path string, payload map[string]any) {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	writeJSON(manifestPath, map[string]any{
		"version":        1,
		"agent":          "Codex1",
		"driver":         "tmux_cli_session",
		"transport":      "tmux",
		"tmux_session":   "orq-codex1-sr",
		"tmux_pane_id":   "%9",
		"status_path":    statusPath,
		"heartbeat_path": heartbeatPath,
	})
	writeJSON(statusPath, map[string]any{
		"state":      "running",
		"updated_at": now,
		"alive":      true,
		"child_pid":  os.Getpid(),
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"heartbeat_at": now,
		"started_at":   now,
		"child_pid":    os.Getpid(),
	})
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codex1-sr",
		"tmux_pane_id":          "%9",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	capsJSON, _ := json.Marshal(map[string]any{
		"can_send_input":        false,
		"mailbox_delivery_mode": runtimeagente.MailboxDeliverySessionResume,
	})
	if _, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, capabilities_json, last_seen_at
	) VALUES (?,?,?,?,?, 'activo', ?, ?, CURRENT_TIMESTAMP)`,
		"Codex1", proyectoID, "tmux", "session", "orq-codex1-sr", string(metaJSON), string(capsJSON)); err != nil {
		t.Fatalf("insert handle tmux: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"instruccion session resume canonica"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia crear una send_instruction usando el runtime principal canónico, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var send *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			send = order
			break
		}
	}
	if send == nil {
		t.Fatalf("faltaba send_instruction")
	}
	if !strings.Contains(send.PayloadJSON, `"mailbox_id":`+strconv.FormatInt(msgID, 10)) {
		t.Fatalf("send_instruction sin mailbox_id: %s", send.PayloadJSON)
	}
	if !strings.Contains(send.PayloadJSON, `"external_session_id":"sess-canonico"`) {
		t.Fatalf("send_instruction sin external_session_id canónico: %s", send.PayloadJSON)
	}
	if send.RuntimeID == nil || *send.RuntimeID != runtimeID {
		t.Fatalf("send_instruction deberia apuntar al runtime canónico: %+v", send)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchEncolaSendInstructionParaWatchdog(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-watchdog"}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "watchdog",
		PayloadJSON: `{"texto":"confirma estado o reanuda tick"}`,
	})
	if err != nil {
		t.Fatalf("mailbox watchdog: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia crear una sola send_instruction para watchdog, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var send *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			send = order
			break
		}
	}
	if send == nil {
		t.Fatalf("faltaba send_instruction")
	}
	if !strings.Contains(send.PayloadJSON, `"mailbox_id":`+strconv.FormatInt(msgID, 10)) {
		t.Fatalf("send_instruction sin mailbox_id: %s", send.PayloadJSON)
	}
	if !strings.Contains(send.PayloadJSON, `"texto":"confirma estado o reanuda tick"`) {
		t.Fatalf("send_instruction watchdog sin texto: %s", send.PayloadJSON)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchRespetaLeaseBootstrapPendiente(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-bootstrap-resume","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"texto":"arranque autonomo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}
	startID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador"}`,
	})
	if err != nil {
		t.Fatalf("encolar start: %v", err)
	}
	resumeID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"mailbox_ids":[%d],"sesion_id":%d}`,
			startID, msgID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume bootstrap: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume ejecutando: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia crear send_instruction si el mailbox ya esta cubierto por bootstrap, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia existir send_instruction duplicada: %+v", order)
		}
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchRespetaLeaseBootstrapPendienteTMUXCanonico(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	metaJSON := `{"driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex1-mailbox","tmux_pane_id":"%21","external_session_id":"sess-premium-tmux","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle tmux: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"haz una micro refactorizacion concreta","texto":"continua trabajo actual"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}
	startID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d,"handle_id":%d,"runtime_id":%d}`,
			msgID, sesion.ID, handle.ID, runtime.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start bootstrap tmux: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start ejecutando: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia crear send_instruction para tmux canonico si bootstrap ya cubre el mailbox, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia existir send_instruction duplicada en tmux canonico: %+v", order)
		}
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchConsumeMailboxBootstrapObservadoTMUXCanonico(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	metaJSON := `{"driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex1-mailbox","tmux_pane_id":"%21","external_session_id":"sess-premium-tmux","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle tmux: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"haz una micro refactorizacion concreta","texto":"continua trabajo actual"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}
	startID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "completada",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","delivery_state":"delivered","delivery_receipt_at":"2026-04-14T12:37:01Z","receipt_source":"git_worktree","mailbox_ids":[%d],"sesion_id":%d,"handle_id":%d,"runtime_id":%d}`,
			msgID, sesion.ID, handle.ID, runtime.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start bootstrap tmux observado: %v", err)
	}
	_ = startID

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia consumir la mailbox bootstrap observada, got=%d", n)
	}

	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if !strings.EqualFold(strings.TrimSpace(msg.Estado), "consumido") {
		t.Fatalf("mailbox bootstrap observada deberia quedar consumida: %+v", msg)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia existir send_instruction duplicada para bootstrap observado: %+v", order)
		}
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchTMUXSessionResumeEsperaReadyAntesDeEncolar(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	runDir := filepath.Join(tmp, "runtime", "Codex1", "session-resume-running")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	writeJSON := func(path string, payload map[string]any) {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	writeJSON(manifestPath, map[string]any{
		"version":        1,
		"agent":          "Codex1",
		"driver":         "tmux_cli_session",
		"transport":      "tmux",
		"tmux_session":   "orq-codex1-session-resume",
		"tmux_pane_id":   "%33",
		"status_path":    statusPath,
		"heartbeat_path": heartbeatPath,
	})
	writeJSON(statusPath, map[string]any{
		"state":      "running",
		"updated_at": now,
		"alive":      true,
		"child_pid":  os.Getpid(),
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"heartbeat_at": now,
		"started_at":   now,
		"child_pid":    os.Getpid(),
	})
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex1-session-resume",
		"tmux_pane_id":          "%33",
		"external_session_id":   "sess-codex1-running",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle tmux: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"haz una micro refactorizacion concreta","texto":"continua trabajo actual"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia encolar send_instruction si el worker tmux aun esta running y no ready, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia existir send_instruction antes de ready: %+v", order)
		}
	}

	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "pendiente" {
		t.Fatalf("la mailbox deberia seguir pendiente hasta que tmux este ready: %+v", msg)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchTMUXSessionResumeDespachaConWorkerIdle(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}

	runDir := filepath.Join(tmp, "runtime", "Codex1", "session-resume-idle")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	writeJSON := func(path string, payload map[string]any) {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s: %v", path, err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	writeJSON(manifestPath, map[string]any{
		"version":        1,
		"agent":          "Codex1",
		"driver":         "tmux_cli_session",
		"transport":      "tmux",
		"tmux_session":   "orq-codex1-session-resume-idle",
		"tmux_pane_id":   "%44",
		"status_path":    statusPath,
		"heartbeat_path": heartbeatPath,
	})
	writeJSON(statusPath, map[string]any{
		"state":      "idle",
		"updated_at": now,
		"alive":      true,
		"child_pid":  os.Getpid(),
	})
	writeJSON(heartbeatPath, map[string]any{
		"alive":        true,
		"heartbeat_at": now,
		"started_at":   now,
		"child_pid":    os.Getpid(),
	})
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"transport":             "tmux",
		"tmux_session":          "orq-codex1-session-resume-idle",
		"tmux_pane_id":          "%44",
		"external_session_id":   "sess-codex1-idle",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle tmux: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"haz una micro refactorizacion concreta","texto":"continua trabajo actual"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia encolar send_instruction para worker tmux idle, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var send *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			send = order
			break
		}
	}
	if send == nil {
		t.Fatalf("faltaba send_instruction para worker tmux idle")
	}
	if !strings.Contains(send.PayloadJSON, `"mailbox_id":`+strconv.FormatInt(msgID, 10)) {
		t.Fatalf("send_instruction sin mailbox_id: %s", send.PayloadJSON)
	}
	if !strings.Contains(send.PayloadJSON, `"external_session_id":"sess-codex1-idle"`) {
		t.Fatalf("send_instruction sin external_session_id: %s", send.PayloadJSON)
	}

	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "pendiente" {
		t.Fatalf("la mailbox debe seguir pendiente hasta entregar de verdad: %+v", msg)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchConsumeMailboxBootstrapObservadoPendiente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	metaJSON := `{"driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex1-mailbox","tmux_pane_id":"%21","external_session_id":"sess-premium-tmux","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle tmux: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"haz una micro refactorizacion concreta","texto":"continua trabajo actual"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}
	startID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "pendiente",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","delivery_state":"delivered","delivery_receipt_at":"2026-04-14T12:37:01Z","receipt_source":"git_worktree","mailbox_ids":[%d],"sesion_id":%d,"handle_id":%d,"runtime_id":%d}`,
			msgID, sesion.ID, handle.ID, runtime.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start pendiente observado: %v", err)
	}
	_ = startID

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia consumir la mailbox bootstrap observada aunque la start siga pendiente, got=%d", n)
	}

	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if !strings.EqualFold(strings.TrimSpace(msg.Estado), "consumido") {
		t.Fatalf("mailbox bootstrap observada deberia quedar consumida: %+v", msg)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia existir send_instruction duplicada para bootstrap pendiente observada: %+v", order)
		}
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchConsumeMailboxBootstrapObservadoDerivadoSinReceiptPropio(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	metaJSON := `{"driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex1-mailbox","tmux_pane_id":"%21","external_session_id":"sess-premium-tmux","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle tmux: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"haz una micro refactorizacion concreta","texto":"continua trabajo actual"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}
	sourceStartID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		Estado:     "completada",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","delivery_state":"delivered","delivery_receipt_at":"2026-04-14T12:37:01Z","receipt_source":"git_worktree","mailbox_ids":[%d],"sesion_id":%d,"handle_id":%d,"runtime_id":%d}`,
			msgID, sesion.ID, handle.ID, runtime.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start bootstrap observada: %v", err)
	}
	resumeID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"sesion_id":%d,"handle_id":%d,"runtime_id":%d}`,
			sourceStartID, sesion.ID, handle.ID, runtime.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume derivada: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume derivada ejecutando: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia consumir la mailbox derivada ya observada por la start fuente, got=%d", n)
	}

	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox final: %+v err=%v", msg, err)
	}
	if !strings.EqualFold(strings.TrimSpace(msg.Estado), "consumido") {
		t.Fatalf("mailbox bootstrap derivada observada deberia quedar consumida: %+v", msg)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia existir send_instruction duplicada para bootstrap derivada observada: %+v", order)
		}
	}
}

func TestReconciliarRuntimeMailboxGuidanceDurableEnInboxBatchConMailboxMarcaEntregadoYEscribeInbox(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	repoDir := filepath.Join(tmp, "repo-inbox-durable")
	worktreeDir := filepath.Join(tmp, "wt-codex1-inbox-durable")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.MkdirAll(worktreeDir, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repoDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Seguir frente durable",
		Descripcion: "Frente actual: cerrar guidance durable en inbox.",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if _, err := (db.CoordinationWorktreeSQLRepository{}).Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		TaskID:    &tareaID,
		Agent:     "Codex1",
		Name:      "orquestador-codex1-inbox-durable",
		Path:      worktreeDir,
		Branch:    "orq/orquestador/codex1/inbox-durable",
		BaseRef:   "main",
		State:     coordinacion.WorktreeActive,
		Reason:    "test_runtime_mailbox_guidance_durable_inbox",
	}); err != nil {
		t.Fatalf("crear worktree activa: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         worktreeDir,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex1-inbox-durable","tmux_pane_id":"%66","mailbox_delivery_mode":"session_resume","can_send_input":false,"external_session_id":"sess-inbox-durable"}`
	capsJSON := `{"mailbox_delivery_mode":"session_resume","can_send_input":false}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-codex1-inbox-durable/%66', estado='activo', metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, capsJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","instruction":"Sigue con la tarea activa y cierra el siguiente slice útil del frente actual dentro del write-set y tests definidos.","texto":"seguir frente"}`,
	})
	if err != nil {
		t.Fatalf("mailbox durable: %v", err)
	}
	signature := runtimeMailboxDeliveryAttemptSignature(handle, "sess-inbox-durable")
	payloadJSON := fmt.Sprintf(`{"mailbox_id":%d,"mailbox_kind":"autonomia","external_session_id":"sess-inbox-durable","delivery_attempt_signature":"%s","to_agente":"Codex1"}`, msgID, signature)
	orderID, err := runtimesService.CreateRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: payloadJSON,
	})
	if err != nil {
		t.Fatalf("create send_instruction mailbox_only: %v", err)
	}
	if err := db.MarcarRuntimeOrderEstado(orderID, "completada", `{"mailbox_only":true,"delivery_state":"delivered","deferred_reason":"runtime_handle_session_resume_mailbox_only:autonomia"}`, ""); err != nil {
		t.Fatalf("marcar send_instruction mailbox_only completada: %v", err)
	}

	pendiente := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: ptrString("Codex1"), ProyectoID: &proyectoID, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	snapshot := newRuntimeMailboxBatchSnapshot()
	resolvedHandle, err := snapshot.activeHandle("Codex1", &proyectoID)
	if err != nil || resolvedHandle == nil {
		t.Fatalf("resolver handle activo: %+v err=%v", resolvedHandle, err)
	}
	orders, err := snapshot.ordersForAgentProject("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("listar orders snapshot: %v", err)
	}
	if len(orders) == 0 {
		t.Fatalf("deberia existir la send_instruction mailbox_only")
	}
	if ok, err := runtimeMailboxGuidanceDurableTieneEntregaMailboxOnly(snapshot, mailbox[0], resolvedHandle); err != nil {
		t.Fatalf("validar mailbox_only vigente: %v", err)
	} else if !ok {
		t.Fatalf("la send_instruction mailbox_only deberia seguir vigente para la sesion actual: current_session=%q current_signature=%q order_payload=%s order_result=%s handle_id=%d resolved_handle_id=%d",
			stringFromMetadataJSON(resolvedHandle.MetadataJSON, "external_session_id"),
			runtimeMailboxDeliveryAttemptSignature(resolvedHandle, stringFromMetadataJSON(resolvedHandle.MetadataJSON, "external_session_id")),
			orders[0].PayloadJSON,
			orders[0].ResultadoJSON,
			handle.ID,
			resolvedHandle.ID,
		)
	}
	n, err := reconciliarRuntimeMailboxGuidanceDurableEnInboxBatchConMailbox(mailbox, map[int64]struct{}{}, snapshot)
	if err != nil {
		t.Fatalf("reconciliar guidance durable inbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reconciliar exactamente una guidance durable, got=%d", n)
	}

	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox: %+v err=%v", msg, err)
	}
	if msg.Estado != "entregado" {
		t.Fatalf("la mailbox durable deberia quedar entregada, got=%s", msg.Estado)
	}
	rawInbox, err := os.ReadFile(filepath.Join(worktreeDir, ".orquesta-inbox.md"))
	if err != nil {
		t.Fatalf("leer inbox durable: %v", err)
	}
	inbox := string(rawInbox)
	if !strings.Contains(inbox, "Seguir frente durable") {
		t.Fatalf("la inbox deberia conservar la tarea activa, got=%q", inbox)
	}
	if !strings.Contains(inbox, "Guidance Durable") || !strings.Contains(inbox, "Sigue con la tarea activa") {
		t.Fatalf("la inbox durable deberia incluir la guidance vigente, got=%q", inbox)
	}
}

func TestReconciliarRuntimeMailboxGuidanceDurableEnInboxBatchConMailboxIgnoraMensajesSinHandleYSigueConElValido(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.RegistrarAgente("CodexFantasma", "programador"); err != nil {
		t.Fatalf("registrar agente fantasma: %v", err)
	}
	repoDir := filepath.Join(tmp, "repo-inbox-durable-skip")
	worktreeDir := filepath.Join(tmp, "wt-codex1-inbox-durable-skip")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.MkdirAll(worktreeDir, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repoDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Seguir frente durable",
		Descripcion: "Frente actual: cerrar guidance durable en inbox.",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if _, err := (db.CoordinationWorktreeSQLRepository{}).Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		TaskID:    &tareaID,
		Agent:     "Codex1",
		Name:      "orquestador-codex1-inbox-durable-skip",
		Path:      worktreeDir,
		Branch:    "orq/orquestador/codex1/inbox-durable-skip",
		BaseRef:   "main",
		State:     coordinacion.WorktreeActive,
		Reason:    "test_runtime_mailbox_guidance_durable_inbox_skip",
	}); err != nil {
		t.Fatalf("crear worktree activa: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         worktreeDir,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex1-inbox-durable-skip","tmux_pane_id":"%67","mailbox_delivery_mode":"session_resume","can_send_input":false,"external_session_id":"sess-inbox-durable-skip"}`
	capsJSON := `{"mailbox_delivery_mode":"session_resume","can_send_input":false}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-codex1-inbox-durable-skip/%67', estado='activo', metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, capsJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "CodexFantasma",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","instruction":"ignorar","texto":"ignorar"}`,
	}); err != nil {
		t.Fatalf("mailbox fantasma: %v", err)
	}
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","instruction":"Sigue con la tarea activa y cierra el siguiente slice útil.","texto":"seguir frente"}`,
	})
	if err != nil {
		t.Fatalf("mailbox durable: %v", err)
	}
	signature := runtimeMailboxDeliveryAttemptSignature(handle, "sess-inbox-durable-skip")
	payloadJSON := fmt.Sprintf(`{"mailbox_id":%d,"mailbox_kind":"autonomia","external_session_id":"sess-inbox-durable-skip","delivery_attempt_signature":"%s","to_agente":"Codex1"}`, msgID, signature)
	orderID, err := runtimesService.CreateRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: payloadJSON,
	})
	if err != nil {
		t.Fatalf("create send_instruction mailbox_only: %v", err)
	}
	if err := db.MarcarRuntimeOrderEstado(orderID, "completada", `{"mailbox_only":true,"delivery_state":"delivered","deferred_reason":"runtime_handle_session_resume_mailbox_only:autonomia"}`, ""); err != nil {
		t.Fatalf("marcar send_instruction mailbox_only completada: %v", err)
	}

	pendiente := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{ProyectoID: &proyectoID, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	n, err := reconciliarRuntimeMailboxGuidanceDurableEnInboxBatchConMailbox(mailbox, map[int64]struct{}{}, newRuntimeMailboxBatchSnapshot())
	if err != nil {
		t.Fatalf("reconciliar guidance durable inbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reconciliar solo la mailbox valida, got=%d", n)
	}
	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox: %+v err=%v", msg, err)
	}
	if msg.Estado != "entregado" {
		t.Fatalf("la mailbox valida deberia quedar entregada, got=%s", msg.Estado)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchNoConsumeMailboxCubiertaPorBootstrapSinEvidencia(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	statusPath := filepath.Join(tmp, "worker-status.json")
	heartbeatPath := filepath.Join(tmp, "worker-heartbeat.json")
	initial := time.Now().UTC().Format(time.RFC3339Nano)
	if err := os.WriteFile(statusPath, []byte(`{"state":"starting","alive":true,"updated_at":"`+initial+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+initial+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-bootstrap-resume","worker_status_path":"` + statusPath + `","worker_heartbeat_path":"` + heartbeatPath + `","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='cli', handle_kind='process', capabilities_json=?, metadata_json=?, estado='activo' WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"haz una micro refactorizacion concreta","texto":"continua trabajo actual"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}
	startID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d}`,
			msgID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start lease: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start ejecutando: %v", err)
	}

	ackAt := time.Now().UTC().Add(2 * time.Second).Format(time.RFC3339Nano)
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","alive":true,"updated_at":"`+ackAt+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("rewrite status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+ackAt+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("rewrite heartbeat: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia consumir el mailbox bootstrap sin evidencia util, got=%d", n)
	}

	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("mailbox: %+v err=%v", msg, err)
	}
	if msg.Estado != "pendiente" {
		t.Fatalf("mailbox deberia seguir pendiente: %+v", msg)
	}
	startOrder, err := db.GetRuntimeOrder(startID)
	if err != nil {
		t.Fatalf("get start: %v", err)
	}
	if startOrder.Estado != "ejecutando" {
		t.Fatalf("start bootstrap no deberia completarse sin evidencia util: %+v", startOrder)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchRespetaLeaseBootstrapPendienteAunqueOtraLeaseSeaMasReciente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-bootstrap-stacked","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='cli', handle_kind='process', capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	oldMsgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"slice anterior","texto":"continua slice anterior"}`,
	})
	if err != nil {
		t.Fatalf("mailbox anterior: %v", err)
	}
	oldStartID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d,"handle_id":%d}`,
			oldMsgID, sesion.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start anterior: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, oldStartID); err != nil {
		t.Fatalf("marcar start anterior ejecutando: %v", err)
	}

	newMsgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"slice reciente","texto":"continua slice reciente"}`,
	})
	if err != nil {
		t.Fatalf("mailbox reciente: %v", err)
	}
	newStartID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d,"handle_id":%d}`,
			newMsgID, sesion.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start reciente: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, newStartID); err != nil {
		t.Fatalf("marcar start reciente ejecutando: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatchConMailbox([]*db.RuntimeMailboxMessage{{
		ID:          oldMsgID,
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"slice anterior","texto":"continua slice anterior"}`,
	}}, map[int64]struct{}{}, newRuntimeMailboxBatchSnapshot())
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia crear send_instruction si la mailbox anterior sigue cubierta por su lease, got=%d", n)
	}

	msg, err := db.GetRuntimeMailbox(oldMsgID)
	if err != nil || msg == nil {
		t.Fatalf("mailbox anterior final: %+v err=%v", msg, err)
	}
	if msg.Estado != "pendiente" {
		t.Fatalf("mailbox anterior deberia seguir pendiente: %+v", msg)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia existir send_instruction duplicada para mailbox anterior cubierta: %+v", order)
		}
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchRespetaLeaseDerivadaSinMailboxIDsPropios(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-bootstrap-linked","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='cli', handle_kind='process', capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"slice ligado","texto":"continua slice ligado"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}
	startID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"acked","mailbox_ids":[%d],"sesion_id":%d,"handle_id":%d}`,
			msgID, sesion.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start fuente completada: %v", err)
	}

	resumeID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		HandleID:   &handle.ID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"sesion_id":%d,"handle_id":%d}`,
			startID, sesion.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume derivada: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume derivada ejecutando: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatchConMailbox([]*db.RuntimeMailboxMessage{{
		ID:          msgID,
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"slice ligado","texto":"continua slice ligado"}`,
	}}, map[int64]struct{}{}, newRuntimeMailboxBatchSnapshot())
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia crear send_instruction si la lease derivada hereda mailbox_ids de la start fuente, got=%d", n)
	}

	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "pendiente" {
		t.Fatalf("mailbox deberia seguir pendiente: %+v", msg)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia existir send_instruction duplicada para mailbox cubierta por lease derivada: %+v", order)
		}
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchRespetaLeaseDerivadaConMailboxPropiaConsumidaSiStartFuenteSigueVigente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-bootstrap-linked-stale","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='cli', handle_kind='process', capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	staleMailboxID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"slice stale","texto":"continua slice stale"}`,
	})
	if err != nil {
		t.Fatalf("mailbox stale: %v", err)
	}
	if err := db.MarcarRuntimeMailboxConsumido(staleMailboxID); err != nil {
		t.Fatalf("consumir mailbox stale: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"slice ligado","texto":"continua slice ligado"}`,
	})
	if err != nil {
		t.Fatalf("mailbox vigente: %v", err)
	}
	startID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"acked","mailbox_ids":[%d],"sesion_id":%d,"handle_id":%d}`,
			msgID, sesion.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start fuente: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET estado='completada', started_at=CURRENT_TIMESTAMP, finished_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start fuente completada: %v", err)
	}

	resumeID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		HandleID:   &handle.ID,
		Tipo:       "resume",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","start_order_id":%d,"mailbox_ids":[%d],"sesion_id":%d,"handle_id":%d}`,
			startID, staleMailboxID, sesion.ID, handle.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar resume derivada: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, resumeID); err != nil {
		t.Fatalf("marcar resume derivada ejecutando: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatchConMailbox([]*db.RuntimeMailboxMessage{{
		ID:          msgID,
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"slice ligado","texto":"continua slice ligado"}`,
	}}, map[int64]struct{}{}, newRuntimeMailboxBatchSnapshot())
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia crear send_instruction si la lease derivada recupera el mailbox vigente desde la start fuente, got=%d", n)
	}

	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("mailbox final: %+v err=%v", msg, err)
	}
	if msg.Estado != "pendiente" {
		t.Fatalf("mailbox deberia seguir pendiente: %+v", msg)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia existir send_instruction duplicada para mailbox cubierta por la start fuente vigente: %+v", order)
		}
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchNoRematerializaMailboxOnlyEnMismaSesion(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-dedupe","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"instruccion durable"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	payload := fmt.Sprintf(`{"to_agente":"Codex1","from_agente":"server","texto":"instruccion durable","mailbox_id":%d,"mailbox_kind":"instruction","external_session_id":"sess-dedupe","delivery_attempt_signature":"session_resume|handle:%d|session:sess-dedupe"}`, msgID, handle.ID)
	resultado := fmt.Sprintf(`{"ok":true,"mailbox_id":%d,"deferred":true,"deferred_reason":"runtime no disponible para entrega inmediata","mailbox_only":true}`, msgID)
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex1",
		ProyectoID:    &proyectoID,
		RuntimeID:     handle.RuntimeID,
		HandleID:      &handle.ID,
		Tipo:          "send_instruction",
		PayloadJSON:   payload,
		ResultadoJSON: resultado,
		Estado:        "completada",
	}); err != nil {
		t.Fatalf("encolar send_instruction previa: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia rematerializar send_instruction mailbox_only en la misma sesion, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var sendCount int
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			sendCount++
		}
	}
	if sendCount != 1 {
		t.Fatalf("no deberia crear una nueva send_instruction: %d", sendCount)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchFallbackInteractivoRespetaLeaseBootstrapPendiente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","supervisor_ref":"codex1-supervisor","stdin_path":"` + filepath.Join(tmp, "codex.stdin") + `","mailbox_delivery_mode":"session_resume","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='cli', handle_kind='process', capabilities_json=?, metadata_json=?, estado='activo' WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"instruction":"haz una micro refactorizacion concreta","texto":"continua trabajo actual"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}
	startID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d}`,
			msgID, sesion.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start lease: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start ejecutando: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia crear send_instruction si el fallback interactivo ya esta cubierto por bootstrap, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia existir send_instruction duplicada: %+v", order)
		}
	}
	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("mailbox: %+v err=%v", msg, err)
	}
	if msg.Estado != "pendiente" {
		t.Fatalf("mailbox deberia seguir pendiente mientras la lease bootstrap siga activa: %+v", msg)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchPermiteReintentoDeMailboxOnlyLegacySinFirma(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-live","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"legacy mailbox"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	payload := fmt.Sprintf(`{"to_agente":"Codex1","from_agente":"server","texto":"legacy mailbox","mailbox_id":%d,"mailbox_kind":"instruction","external_session_id":"sess-live"}`, msgID)
	resultado := fmt.Sprintf(`{"ok":true,"mailbox_id":%d,"deferred":true,"deferred_reason":"runtime no disponible para entrega inmediata","mailbox_only":true}`, msgID)
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex1",
		ProyectoID:    &proyectoID,
		RuntimeID:     handle.RuntimeID,
		HandleID:      &handle.ID,
		Tipo:          "send_instruction",
		PayloadJSON:   payload,
		ResultadoJSON: resultado,
		Estado:        "completada",
	}); err != nil {
		t.Fatalf("encolar send_instruction legacy: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia permitir un reintento cuando falta firma de intento, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var sendCount int
	var newest *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			sendCount++
			if newest == nil || order.ID > newest.ID {
				newest = order
			}
		}
	}
	if sendCount != 2 {
		t.Fatalf("deberia crear una nueva send_instruction para el retry util, got=%d", sendCount)
	}
	if newest == nil || !strings.Contains(newest.PayloadJSON, `"delivery_attempt_signature":"session_resume|handle:`) {
		t.Fatalf("la orden nueva deberia persistir la firma del intento: %+v", newest)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchRematerializaMailboxOnlyCaducado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	old := time.Now().UTC().Add(-2 * runtimeMailboxOnlyDedupeTTL())

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-expirada","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"instruccion durable caducada"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	payload := fmt.Sprintf(`{"to_agente":"Codex1","from_agente":"server","texto":"instruccion durable caducada","mailbox_id":%d,"mailbox_kind":"instruction","external_session_id":"sess-expirada","delivery_attempt_signature":"session_resume|handle:%d|session:sess-expirada"}`, msgID, handle.ID)
	resultado := fmt.Sprintf(`{"ok":true,"mailbox_id":%d,"deferred":true,"deferred_reason":"runtime no disponible para entrega inmediata","mailbox_only":true}`, msgID)
	orderID, err := int64(0), error(nil)
	if orderID, err = db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex1",
		ProyectoID:    &proyectoID,
		RuntimeID:     handle.RuntimeID,
		HandleID:      &handle.ID,
		Tipo:          "send_instruction",
		PayloadJSON:   payload,
		ResultadoJSON: resultado,
		Estado:        "completada",
		FinishedAt:    &old,
		UpdatedAt:     old,
	}); err != nil {
		t.Fatalf("encolar send_instruction previa: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET finished_at=?, updated_at=? WHERE id=?`,
		old,
		old,
		orderID,
	); err != nil {
		t.Fatalf("envejecer send_instruction previa: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia rematerializar send_instruction mailbox_only caducada, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var sendCount int
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			sendCount++
		}
	}
	if sendCount != 2 {
		t.Fatalf("deberia crear una nueva send_instruction tras caducar el intento previo: %d", sendCount)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchNoRematerializaGuidanceCaducadaEnMismaSesion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	old := time.Now().UTC().Add(-2 * runtimeMailboxOnlyDedupeTTL())

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-guidance","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"texto":"reevalua bloqueo y sigue"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	payload := fmt.Sprintf(`{"to_agente":"Codex1","from_agente":"server","texto":"reevalua bloqueo y sigue","mailbox_id":%d,"mailbox_kind":"autonomia","external_session_id":"sess-guidance","delivery_attempt_signature":"session_resume|handle:%d|session:sess-guidance"}`, msgID, handle.ID)
	resultado := fmt.Sprintf(`{"ok":true,"mailbox_id":%d,"deferred":true,"deferred_reason":"session_resume timeout: Perfil activo: Codex1","mailbox_only":true}`, msgID)
	orderID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex1",
		ProyectoID:    &proyectoID,
		RuntimeID:     handle.RuntimeID,
		HandleID:      &handle.ID,
		Tipo:          "send_instruction",
		PayloadJSON:   payload,
		ResultadoJSON: resultado,
		Estado:        "completada",
		FinishedAt:    &old,
		UpdatedAt:     old,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction previa: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET finished_at=?, updated_at=? WHERE id=?`, old, old, orderID); err != nil {
		t.Fatalf("envejecer send_instruction previa: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia rematerializar guidance caducada en la misma sesion, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var sendCount int
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			sendCount++
		}
	}
	if sendCount != 1 {
		t.Fatalf("no deberia crear una nueva send_instruction guidance: %d", sendCount)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchPermiteGuidanceDurableAunqueBootstrapSigaPendiente(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	metaJSON := `{"driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex1-guidance-bootstrap","tmux_pane_id":"%68","mailbox_delivery_mode":"session_resume","can_send_input":false,"external_session_id":"sess-guidance-bootstrap"}`
	capsJSON := `{"mailbox_delivery_mode":"session_resume","can_send_input":false}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET transporte='tmux', handle_kind='session', handle_ref='orq-codex1-guidance-bootstrap/%68', estado='activo', metadata_json=?, capabilities_json=? WHERE id=?`, metaJSON, capsJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","instruction":"Sigue con el siguiente slice util.","texto":"seguir frente"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}
	startID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyectoID,
		RuntimeID:  &runtime.ID,
		HandleID:   &handle.ID,
		Tipo:       "start",
		ResultadoJSON: fmt.Sprintf(
			`{"lease_state":"waiting_for_evidence","mailbox_ids":[%d],"sesion_id":%d,"handle_id":%d,"runtime_id":%d}`,
			msgID, sesion.ID, handle.ID, runtime.ID,
		),
	})
	if err != nil {
		t.Fatalf("encolar start bootstrap: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_orders SET estado='ejecutando', started_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?`, startID); err != nil {
		t.Fatalf("marcar start ejecutando: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia rematerializar guidance durable aunque bootstrap siga pendiente, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var sendOrder *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" && order.ID != startID {
			sendOrder = order
		}
	}
	if sendOrder == nil {
		t.Fatal("deberia crear una send_instruction para guidance durable")
	}
	if !strings.Contains(sendOrder.PayloadJSON, `"mailbox_id":`+strconv.FormatInt(msgID, 10)) {
		t.Fatalf("payload inesperado: %s", sendOrder.PayloadJSON)
	}
}

func TestEncolarSendInstructionDesdeRuntimeMailboxCompactaNudgeCodex(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
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
		Agente:      "Codex4",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"'/home/alberto/Trabajo/codex-perfiles/bin/codex-perfil' 'Codex4'","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-codex4","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=?, handle_ref=? WHERE id=?`, capsJSON, metaJSON, strconv.Itoa(os.Getpid()), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	handle, err = db.GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("refrescar handle: %+v err=%v", handle, err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex4",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"Se te ha asignado automaticamente la tarea #411. Entra en Orquesta, revisa el contexto vivo y continua hasta cerrarla."}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	orderID, err := encolarSendInstructionDesdeRuntimeMailbox(&db.RuntimeMailboxMessage{
		ID:          msgID,
		FromAgente:  "server",
		ToAgente:    "Codex4",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"Se te ha asignado automaticamente la tarea #411. Entra en Orquesta, revisa el contexto vivo y continua hasta cerrarla."}`,
	}, handle, "Se te ha asignado automaticamente la tarea #411. Entra en Orquesta, revisa el contexto vivo y continua hasta cerrarla.", "sess-codex4")
	if err != nil {
		t.Fatalf("encolar send_instruction: %v", err)
	}
	order, err := db.GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get order: %+v err=%v", order, err)
	}
	if !strings.Contains(order.PayloadJSON, `"texto":"toma tarea asignada y sigue"`) {
		t.Fatalf("payload no compactado para Codex: %s", order.PayloadJSON)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchCoalesceNudgeAunqueHayaOrdenAbierta(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-nudge"}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"instruccion abierta"}`,
	}); err != nil {
		t.Fatalf("encolar send_instruction existente: %v", err)
	}

	viejoID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"nudge viejo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox vieja: %v", err)
	}
	nuevoID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"nudge nuevo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox nueva: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia crear send_instruction nueva con orden abierta, got=%d", n)
	}

	agente := "Codex1"
	pendiente := "pendiente"
	mailboxPendiente, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailboxPendiente) != 1 || mailboxPendiente[0].ID != nuevoID {
		t.Fatalf("la mailbox coalescible deberia conservar solo la ultima: %+v", mailboxPendiente)
	}

	consumido := "consumido"
	mailboxConsumida, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &consumido})
	if err != nil {
		t.Fatalf("listar mailbox consumida: %v", err)
	}
	foundOld := false
	for _, msg := range mailboxConsumida {
		if msg != nil && msg.ID == viejoID {
			foundOld = true
			break
		}
	}
	if !foundOld {
		t.Fatalf("la mailbox vieja deberia quedar consumida tras coalesce: %+v", mailboxConsumida)
	}
}

func TestProcesarRuntimeMailboxSessionResumeBatchPermiteReentregaEnNuevaSesionConFirmaDistinta(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	// Handle activo con external_session_id=sess-b (sesión nueva)
	metaJSON := `{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-b","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliverySessionResume + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"instruccion en nueva sesion"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	// Orden previa de sess-a (firma diferente a la actual sess-b), reciente (no caducada)
	oldSignature := fmt.Sprintf("session_resume|handle:%d|session:sess-a", handle.ID)
	payload := fmt.Sprintf(`{"to_agente":"Codex1","from_agente":"server","texto":"instruccion en nueva sesion","mailbox_id":%d,"mailbox_kind":"instruction","external_session_id":"sess-a","delivery_attempt_signature":"%s"}`, msgID, oldSignature)
	resultado := fmt.Sprintf(`{"ok":true,"mailbox_id":%d,"deferred":true,"deferred_reason":"runtime no disponible para entrega inmediata","mailbox_only":true}`, msgID)
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex1",
		ProyectoID:    &proyectoID,
		RuntimeID:     handle.RuntimeID,
		HandleID:      &handle.ID,
		Tipo:          "send_instruction",
		PayloadJSON:   payload,
		ResultadoJSON: resultado,
		Estado:        "completada",
	}); err != nil {
		t.Fatalf("encolar send_instruction sesion anterior: %v", err)
	}

	n, err := procesarRuntimeMailboxSessionResumeBatch()
	if err != nil {
		t.Fatalf("procesar mailbox session resume: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia permitir reentrega en sesion nueva aunque haya mailbox_only reciente de sesion anterior, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var sendCount int
	var newest *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			sendCount++
			if newest == nil || order.ID > newest.ID {
				newest = order
			}
		}
	}
	if sendCount != 2 {
		t.Fatalf("deberia crear una nueva send_instruction para la sesion nueva, got=%d", sendCount)
	}
	newSignature := fmt.Sprintf("session_resume|handle:%d|session:sess-b", handle.ID)
	if newest == nil || !strings.Contains(newest.PayloadJSON, newSignature) {
		t.Fatalf("la orden nueva deberia tener la firma de la sesion actual: %+v", newest)
	}
}

func TestProcesarRuntimeMailboxBatchConsumeWatchdogSinHandleActivo(t *testing.T) {
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

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "watchdog",
		PayloadJSON: `{"texto":"heartbeat obsoleto"}`,
	})
	if err != nil {
		t.Fatalf("mailbox watchdog: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reconciliar exactamente un watchdog sin handle, got=%d", n)
	}

	pendiente := "pendiente"
	toAgente := "Codex1"
	pendientes, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &pendiente,
	})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(pendientes) != 0 {
		t.Fatalf("no deberia quedar watchdog pendiente sin handle: %+v", pendientes)
	}

	consumido := "consumido"
	consumidos, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &consumido,
	})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != msgID || consumidos[0].Kind != "watchdog" {
		t.Fatalf("watchdog consumido inesperado: %+v", consumidos)
	}

	logs, err := db.ListarAuditoria(db.FiltroAuditoria{Limite: 200})
	if err != nil {
		t.Fatalf("listar auditoria: %v", err)
	}
	found := false
	for _, item := range logs {
		if item != nil && item.Accion == "runtime_mailbox_watchdog_sin_handle" && item.EntidadID == msgID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("faltaba auditoria de watchdog sin handle")
	}
}

func TestProcesarRuntimeMailboxBatchConsumeAgenteSinVida(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex6", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex6",
		Kind:        "nudge",
		PayloadJSON: `{"texto":"fuera de flota activa"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar runtime mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reconciliar exactamente un mailbox zombie, got=%d", n)
	}

	agente := "Codex6"
	estado := "pendiente"
	pendientes, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(pendientes) != 0 {
		t.Fatalf("no deberia quedar mailbox pendiente para agente sin vida: %+v", pendientes)
	}
	estado = "consumido"
	consumidos, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != msgID {
		t.Fatalf("mailbox consumido inesperado: %+v", consumidos)
	}
}

func TestProcesarRuntimeMailboxBatchNoConsumeAgenteSinHandlePeroConTrabajo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex6", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Codex6", proyectoID, "frente activo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Trabajo real",
		Descripcion: "No debe consumirse el mailbox",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex6"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex6",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"retoma el trabajo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar runtime mailbox: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia reconciliar mailbox de agente con trabajo real, got=%d", n)
	}

	agente := "Codex6"
	estado := "pendiente"
	pendientes, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(pendientes) != 1 || pendientes[0].ID != msgID {
		t.Fatalf("mailbox pendiente inesperado: %+v", pendientes)
	}
}

func TestProcesarRuntimeMailboxBatchReabrePoolLocalCompartidoSinHandle(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("GemmaMailbox", "programador"); err != nil {
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
	if _, err := capacidadService.AsegurarPoolLocalCompartido(capacidadapp.EntradaAsegurarPoolLocalCompartido{
		PoolSlug:               "ollama-gemma4",
		Proveedor:              "Ollama",
		Runtime:                "ollama",
		ModeloPreferente:       "gemma4:26b",
		SlotsMaximos:           1,
		ConectorCanonico:       "ollama_pool_local",
		ConectorCompatibilidad: "ollama-cli",
		ExperimentalCompat:     true,
	}); err != nil {
		t.Fatalf("guardar pool: %v", err)
	}
	if _, err := db.GuardarPoolModelo("ollama-gemma4", &db.PoolModelo{ModelSlug: "gemma4:26b", Activo: true, Prioridad: 10, CosteRelativo: 1}); err != nil {
		t.Fatalf("guardar pool modelo: %v", err)
	}
	if _, err := db.GuardarPoliticaModelo(&db.PoliticaModelo{
		ScopeTipo:       "perfil",
		ScopeRef:        "implementacion",
		PerfilTarea:     "implementacion",
		PoolSlug:        "ollama-gemma4",
		ModelSlug:       "gemma4:26b",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
	}); err != nil {
		t.Fatalf("guardar politica modelo: %v", err)
	}
	if err := db.ActivarAsignacion("GemmaMailbox", proyectoID, "frente activo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "GemmaMailbox",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "ollama_pool_local",
	})
	if err != nil {
		t.Fatalf("iniciar sesion pool local: %v", err)
	}
	if err := db.FinSesion("GemmaMailbox"); err != nil {
		t.Fatalf("cerrar sesion pool local: %v", err)
	}
	if sesion == nil {
		t.Fatal("sesion pool local nula")
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "GemmaMailbox",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"continua la microtarea"}`,
	}); err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar runtime mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reactivar pool local compartido, got=%d", n)
	}

	agente := "GemmaMailbox"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != agenteControlAccionStart {
		t.Fatalf("deberia encolar start para reabrir el pool local: %+v", orders)
	}
	mailboxEstado := "pendiente"
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &mailboxEstado})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailbox) != 1 {
		t.Fatalf("la mailbox debe seguir pendiente tras reactivar el pool: %+v", mailbox)
	}
}

func TestProcesarRuntimeMailboxBatchReactivaPremiumSinHandleActivo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "frente premium"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"tmux_cli_session","tmux_session":"orq-gemini1-restart","mailbox_delivery_mode":"interactive","can_send_input":true}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='fallido', metadata_json=? WHERE id=?`, metaJSON, handle.ID); err != nil {
		t.Fatalf("fallar handle: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Gemini1",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"reanuda trabajo premium"}`,
	}); err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar runtime mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reactivar runtime premium sin handle activo, got=%d", n)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != agenteControlAccionStart {
		t.Fatalf("deberia encolar start para reactivar premium: %+v", orders)
	}
}

func TestProcesarRuntimeMailboxBatchReactivaPremiumAutonomiaSinHandleConTrabajoAsignado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "frente premium"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar continuidad premium",
		Descripcion: "Trabajo premium ya asignado",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Gemini1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","texto":"reanuda el frente premium asignado"}`,
	}); err != nil {
		t.Fatalf("mailbox autonomia: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar runtime mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reactivar premium con mailbox autonomia y trabajo asignado, got=%d", n)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != agenteControlAccionStart {
		t.Fatalf("deberia encolar start para reactivar premium con continuidad: %+v", orders)
	}
}

func TestProcesarRuntimeMailboxBatchConsumeAutonomiaDeAgenteRetirado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexRetirado", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("CodexRetirado", proyectoID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if err := db.RetirarAgente("CodexRetirado"); err != nil {
		t.Fatalf("retirar agente: %v", err)
	}
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "CodexRetirado",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","texto":"retoma el frente"}`,
	})
	if err != nil {
		t.Fatalf("mailbox autonomia: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar runtime mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia consumir mailbox de agente retirado, got=%d", n)
	}

	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil {
		t.Fatalf("get runtime mailbox: %v", err)
	}
	if msg == nil || !strings.EqualFold(strings.TrimSpace(msg.Estado), "consumido") {
		t.Fatalf("mailbox deberia quedar consumido: %+v", msg)
	}

	agente := "CodexRetirado"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia encolar start para agente retirado: %+v", orders)
	}
}

func TestProcesarRuntimeMailboxBatchReactivaPremiumAutonomiaSinHandleIgnoraCapacidadPoolLocal(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente premium: %v", err)
	}
	if err := db.RegistrarAgente("GemmaBusy", "programador"); err != nil {
		t.Fatalf("registrar agente pool busy: %v", err)
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
	if _, err := db.GuardarPool(&db.PoolCapacidad{
		Slug:                "ollama-gemma4",
		Proveedor:           "Ollama",
		Runtime:             "ollama",
		Plan:                "local",
		CapacidadTotal:      1,
		PermiteModelosMulti: true,
		PoliticaHandoff:     "preventivo",
		FuenteTelemetria:    "manual",
		MetadataJSON:        `{"conector_canonico":"ollama_pool_local","modelo_preferente":"gemma4:26b","slots_maximos":1}`,
		Activo:              true,
	}); err != nil {
		t.Fatalf("guardar pool: %v", err)
	}
	if _, err := db.GuardarPoolModelo("ollama-gemma4", &db.PoolModelo{ModelSlug: "gemma4:26b", Activo: true, Prioridad: 10, CosteRelativo: 1}); err != nil {
		t.Fatalf("guardar pool modelo: %v", err)
	}
	if _, err := db.GuardarPoliticaModelo(&db.PoliticaModelo{
		ScopeTipo:       "perfil",
		ScopeRef:        "implementacion",
		PerfilTarea:     "implementacion",
		PoolSlug:        "ollama-gemma4",
		ModelSlug:       "gemma4:26b",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
	}); err != nil {
		t.Fatalf("guardar politica modelo: %v", err)
	}
	sesionBusy, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "GemmaBusy",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "gemma-busy"),
		Herramienta: "ollama_pool_local",
	})
	if err != nil {
		t.Fatalf("iniciar sesion busy: %v", err)
	}
	pool, err := db.GetPool("ollama-gemma4")
	if err != nil {
		t.Fatalf("get pool: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE sesiones SET pool_id = ? WHERE id = ?`, pool.ID, sesionBusy.ID); err != nil {
		t.Fatalf("asignar pool a sesion busy: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "frente premium"); err != nil {
		t.Fatalf("activar asignacion premium: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar continuidad premium con pool ocupado",
		Descripcion: "Trabajo premium ya asignado",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea premium: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Gemini1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","texto":"reanuda el frente premium asignado"}`,
	}); err != nil {
		t.Fatalf("mailbox autonomia: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar runtime mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reactivar premium aunque el pool local este ocupado, got=%d", n)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != agenteControlAccionStart {
		t.Fatalf("deberia encolar start para premium aunque la politica de modelo use pool local: %+v", orders)
	}
}

func TestProcesarRuntimeMailboxBatchReactivaPremiumSiHandleCanonicoRecienteNoEstaActivo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "frente premium"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar continuidad premium con handle residual",
		Descripcion: "Trabajo premium ya asignado",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"tmux_cli_session","tmux_session":"orq-gemini1-stale","mailbox_delivery_mode":"interactive","can_send_input":true}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='fallido', metadata_json=?, last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, metaJSON, handle.ID); err != nil {
		t.Fatalf("dejar handle residual: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Gemini1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","texto":"reanuda el frente premium asignado"}`,
	}); err != nil {
		t.Fatalf("mailbox autonomia: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar runtime mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reactivar premium aunque exista handle canonico reciente no activo, got=%d", n)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != agenteControlAccionStart {
		t.Fatalf("deberia encolar start para reactivar premium con handle residual: %+v", orders)
	}
}

func TestProcesarRuntimeMailboxBatchReactivaPremiumSinProyectoEnMailbox(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "frente premium"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"tmux_cli_session","tmux_session":"orq-gemini1-restart","mailbox_delivery_mode":"interactive","can_send_input":true}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='fallido', metadata_json=? WHERE id=?`, metaJSON, handle.ID); err != nil {
		t.Fatalf("fallar handle: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Gemini1",
		Kind:        "nudge",
		PayloadJSON: `{"texto":"reanuda trabajo premium"}`,
	}); err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar runtime mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reactivar runtime premium aunque el mailbox no traiga proyecto, got=%d", n)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != agenteControlAccionStart {
		t.Fatalf("deberia encolar start premium inferido desde contexto del agente: %+v", orders)
	}
}

func TestProcesarRuntimeMailboxBatchConsumeAgenteFueraDeFlotaConAsignacionAutomatica(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.ConfigSet("server_autobootstrap_supervisor_agent", "Codex1"); err != nil {
		t.Fatalf("config supervisor: %v", err)
	}
	if err := db.ConfigSet("server_autobootstrap_worker_agents", "Codex2,Codex3,Codex4,Codex5"); err != nil {
		t.Fatalf("config workers: %v", err)
	}
	if err := db.RegistrarAgente("Codex6", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Codex6", proyectoID, "reactivacion_automatica"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex6",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"texto":"fuera de flota activa"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar runtime mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reconciliar mailbox de agente fuera de flota, got=%d", n)
	}

	agente := "Codex6"
	estado := "consumido"
	consumidos, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar consumidos: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != msgID {
		t.Fatalf("mailbox consumido inesperado: %+v", consumidos)
	}
}

func TestProcesarRuntimeMailboxBatchConsumeWatchdogEnEnfriamiento(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar enfriamiento: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "watchdog",
		PayloadJSON: `{"texto":"no deberia despertar al agente en enfriamiento"}`,
	})
	if err != nil {
		t.Fatalf("mailbox watchdog: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reconciliar exactamente un watchdog en enfriamiento, got=%d", n)
	}

	pendiente := "pendiente"
	toAgente := "Codex1"
	pendientes, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &pendiente,
	})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(pendientes) != 0 {
		t.Fatalf("no deberia quedar watchdog pendiente en enfriamiento: %+v", pendientes)
	}

	consumido := "consumido"
	consumidos, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &consumido,
	})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != msgID {
		t.Fatalf("watchdog consumido inesperado: %+v", consumidos)
	}

	logs, err := db.ListarAuditoria(db.FiltroAuditoria{Limite: 100})
	if err != nil {
		t.Fatalf("listar auditoria: %v", err)
	}
	found := false
	for _, item := range logs {
		if item != nil && item.Accion == "runtime_mailbox_watchdog_enfriamiento" && item.EntidadID == msgID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("faltaba auditoria de watchdog en enfriamiento")
	}
}

func TestProcesarRuntimeMailboxBatchConsumeGovernanceRefreshEnEnfriamiento(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar enfriamiento: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        db.MailboxKindGovernanceRefresh,
		PayloadJSON: `{"motivo":"quota_cooldown"}`,
	})
	if err != nil {
		t.Fatalf("mailbox governance_refresh: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reconciliar exactamente un governance_refresh en enfriamiento, got=%d", n)
	}

	pendiente := "pendiente"
	toAgente := "Codex1"
	pendientes, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &pendiente,
	})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(pendientes) != 0 {
		t.Fatalf("no deberia quedar governance_refresh pendiente en enfriamiento: %+v", pendientes)
	}

	consumido := "consumido"
	consumidos, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &consumido,
	})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != msgID {
		t.Fatalf("governance_refresh consumido inesperado: %+v", consumidos)
	}

	logs, err := db.ListarAuditoria(db.FiltroAuditoria{Limite: 200})
	if err != nil {
		t.Fatalf("listar auditoria: %v", err)
	}
	found := false
	for _, item := range logs {
		if item != nil && item.Accion == "runtime_mailbox_refresh_enfriamiento" && item.EntidadID == msgID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("faltaba auditoria de governance_refresh en enfriamiento")
	}
}

func TestProcesarRuntimeMailboxBatchConsumeInstructionEnEnfriamiento(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar enfriamiento: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"no deberia quedar pendiente en enfriamiento"}`,
	})
	if err != nil {
		t.Fatalf("mailbox instruction: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reconciliar exactamente una instruction en enfriamiento, got=%d", n)
	}

	pendiente := "pendiente"
	toAgente := "Codex1"
	pendientes, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &pendiente,
	})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(pendientes) != 0 {
		t.Fatalf("no deberia quedar instruction pendiente en enfriamiento: %+v", pendientes)
	}

	consumido := "consumido"
	consumidos, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &consumido,
	})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != msgID {
		t.Fatalf("instruction consumida inesperada: %+v", consumidos)
	}

	logs, err := db.ListarAuditoria(db.FiltroAuditoria{Limite: 200})
	if err != nil {
		t.Fatalf("listar auditoria: %v", err)
	}
	found := false
	for _, item := range logs {
		if item != nil && item.Accion == "runtime_mailbox_instruction_enfriamiento" && item.EntidadID == msgID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("faltaba auditoria de instruction en enfriamiento")
	}
}

func TestProcesarRuntimeMailboxBatchDeduplicaPipelineLocalEnCuota(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
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
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "claude-code",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Claude1'`); err != nil {
		t.Fatalf("marcar enfriamiento: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
			FromAgente:  "server",
			ToAgente:    "Claude1",
			ProyectoID:  &proyectoID,
			Kind:        "pipeline_local",
			PayloadJSON: fmt.Sprintf(`{"accion":"continuar_trabajo","seq":%d}`, i),
		}); err != nil {
			t.Fatalf("mailbox pipeline_local %d: %v", i, err)
		}
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox: %v", err)
	}
	if n < 2 {
		t.Fatalf("deberia reconciliar al menos dos duplicados en cuota, got=%d", n)
	}

	pendiente := "pendiente"
	toAgente := "Claude1"
	pendientes, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &pendiente,
	})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(pendientes) != 1 {
		t.Fatalf("deberia quedar un solo pipeline_local pendiente, got=%d %+v", len(pendientes), pendientes)
	}
	if pendientes[0] == nil || !strings.EqualFold(strings.TrimSpace(pendientes[0].Kind), "pipeline_local") {
		t.Fatalf("mailbox restante inesperado: %+v", pendientes)
	}
}

func TestProcesarRuntimeMailboxBatchDeduplicaPipelineLocalConHandlePausado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Claude1", "programador"); err != nil {
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
		Agente:      "Claude1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "claude-code",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
			FromAgente:  "server",
			ToAgente:    "Claude1",
			ProyectoID:  &proyectoID,
			Kind:        "pipeline_local",
			PayloadJSON: fmt.Sprintf(`{"accion":"continuar_trabajo","seq":%d}`, i),
		}); err != nil {
			t.Fatalf("mailbox pipeline_local %d: %v", i, err)
		}
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox: %v", err)
	}
	if n < 2 {
		t.Fatalf("deberia reconciliar al menos dos duplicados con handle pausado, got=%d", n)
	}

	pendiente := "pendiente"
	toAgente := "Claude1"
	pendientes, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyectoID,
		Estado:     &pendiente,
	})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(pendientes) != 1 {
		t.Fatalf("deberia quedar un solo pipeline_local pendiente, got=%d %+v", len(pendientes), pendientes)
	}
	if pendientes[0] == nil || !strings.EqualFold(strings.TrimSpace(pendientes[0].Kind), "pipeline_local") {
		t.Fatalf("mailbox restante inesperado: %+v", pendientes)
	}
}

func TestProcesarRuntimeMailboxInteractivoBatchNoDuplicaSendInstructionAbierta(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if handle.RuntimeID == nil {
		t.Fatalf("runtime handle sin runtime asociado: %+v", handle)
	}

	existenteID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		RuntimeID:   handle.RuntimeID,
		HandleID:    &handle.ID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex1","texto":"instruccion abierta"}`,
	})
	if err != nil {
		t.Fatalf("encolar send_instruction existente: %v", err)
	}
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"otra instruccion"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxInteractivoBatch()
	if err != nil {
		t.Fatalf("procesar mailbox interactivo: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia crear send_instruction nueva, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var sendInstructions []int64
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			sendInstructions = append(sendInstructions, order.ID)
		}
	}
	if len(sendInstructions) != 1 || sendInstructions[0] != existenteID {
		t.Fatalf("send_instruction inesperadas: %+v", sendInstructions)
	}

	estado := "pendiente"
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(mailbox) != 1 || mailbox[0].ID != msgID {
		t.Fatalf("mailbox pendiente inesperada: %+v", mailbox)
	}
}

func TestProcesarRuntimeMailboxInteractivoBatchCoalesceAutonomiaPendiente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if handle.RuntimeID == nil {
		t.Fatalf("runtime handle sin runtime asociado: %+v", handle)
	}
	capsJSON := `{"can_send_input":true,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryInteractive + `"}`
	metaJSON := `{"driver":"tmux_cli_session","transporte":"tmux","tmux_session":"orq-codex1","tmux_pane_id":"%1","stdin_path":"` + filepath.Join(tmp, "tmux.stdin") + `","rendered_command":"cat-cli Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","can_send_input":true,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryInteractive + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle interactivo: %v", err)
	}
	handle, err = db.GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}
	if got := db.RuntimeHandleMailboxDeliveryMode(handle); got != runtimeagente.MailboxDeliveryInteractive {
		t.Fatalf("modo interactivo inesperado: %s", got)
	}
	if !db.RuntimeHandlePermiteSendInputInteractivo(handle) {
		t.Fatalf("el handle tmux interactivo deberia permitir input")
	}

	msgViejo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"mensaje viejo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox vieja: %v", err)
	}
	msgNuevo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"mensaje nuevo"}`,
	})
	if err != nil {
		t.Fatalf("mailbox nueva: %v", err)
	}

	n, err := procesarRuntimeMailboxInteractivoBatch()
	if err != nil {
		t.Fatalf("procesar mailbox interactivo: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia crear una sola send_instruction, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var send *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			send = order
			break
		}
	}
	if send == nil {
		t.Fatalf("faltaba send_instruction")
	}
	if !strings.Contains(send.PayloadJSON, "mensaje nuevo") {
		t.Fatalf("deberia usar la autonomia mas reciente: %s", send.PayloadJSON)
	}

	pendiente := "pendiente"
	consumido := "consumido"
	mailboxPendiente, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailboxPendiente) != 1 || mailboxPendiente[0].ID != msgNuevo {
		t.Fatalf("deberia quedar pendiente la autonomia vigente hasta entrega real: %+v", mailboxPendiente)
	}
	mailboxConsumido, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &consumido})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	if len(mailboxConsumido) != 1 {
		t.Fatalf("solo deberia quedar consumida la autonomia supersedida: %+v", mailboxConsumido)
	}
	idsConsumidos := map[int64]struct{}{}
	for _, msg := range mailboxConsumido {
		if msg != nil {
			idsConsumidos[msg.ID] = struct{}{}
		}
	}
	if _, ok := idsConsumidos[msgViejo]; !ok {
		t.Fatalf("faltaba autonomia vieja consumida")
	}
	if _, ok := idsConsumidos[msgNuevo]; ok {
		t.Fatalf("la autonomia nueva no deberia marcarse consumida antes de la entrega real")
	}
}

func TestProcesarRuntimeMailboxInteractivoBatchNoRematerializaMailboxOnlyEnMismoHandle(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "cat-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if handle.RuntimeID == nil {
		t.Fatalf("runtime handle sin runtime asociado: %+v", handle)
	}
	capsJSON := `{"can_send_input":true,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryInteractive + `"}`
	metaJSON := `{"driver":"tmux_cli_session","transporte":"tmux","tmux_session":"orq-codex1","tmux_pane_id":"%1","stdin_path":"` + filepath.Join(tmp, "tmux.stdin") + `","rendered_command":"cat-cli Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","can_send_input":true}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"instruccion interactiva durable"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	payload := fmt.Sprintf(`{"to_agente":"Codex1","from_agente":"server","texto":"instruccion interactiva durable","mailbox_id":%d,"mailbox_kind":"instruction","delivery_attempt_signature":"interactive|handle:%d|session:"}`, msgID, handle.ID)
	resultado := fmt.Sprintf(`{"ok":true,"mailbox_id":%d,"deferred":true,"deferred_reason":"runtime no disponible para entrega inmediata","mailbox_only":true}`, msgID)
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex1",
		ProyectoID:    &proyectoID,
		RuntimeID:     handle.RuntimeID,
		HandleID:      &handle.ID,
		Tipo:          "send_instruction",
		PayloadJSON:   payload,
		ResultadoJSON: resultado,
		Estado:        "completada",
	}); err != nil {
		t.Fatalf("encolar send_instruction previa: %v", err)
	}

	n, err := procesarRuntimeMailboxInteractivoBatch()
	if err != nil {
		t.Fatalf("procesar mailbox interactivo: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia rematerializar send_instruction mailbox_only en el mismo handle, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var sendCount int
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			sendCount++
		}
	}
	if sendCount != 1 {
		t.Fatalf("no deberia crear una nueva send_instruction: %d", sendCount)
	}
}

func TestProcesarRuntimeMailboxInteractivoBatchIgnoraLegacyProcessPTY(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "cat-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	capsJSON := `{"can_send_input":true,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryInteractive + `"}`
	metaJSON := `{"driver":"process_pty_cli","stdin_path":"` + filepath.Join(tmp, "pty.stdin") + `","rendered_command":"cat-cli Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","can_send_input":true,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryInteractive + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle legacy: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"legacy interactive"}`,
	}); err != nil {
		t.Fatalf("mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxInteractivoBatch()
	if err != nil {
		t.Fatalf("procesar mailbox interactivo: %v", err)
	}
	if n != 0 {
		t.Fatalf("un handle legacy process_pty_cli no deberia materializar interactive, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			t.Fatalf("no deberia crear send_instruction interactive legacy: %+v", order)
		}
	}
}

func TestProcesarRuntimeMailboxInteractivoBatchCoalesceInstructionPendiente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetRuntimeMailboxReevaluationGate()

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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if handle.RuntimeID == nil {
		t.Fatalf("runtime handle sin runtime asociado: %+v", handle)
	}
	capsJSON := `{"can_send_input":true,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryInteractive + `"}`
	metaJSON := `{"driver":"tmux_cli_session","transporte":"tmux","tmux_session":"orq-codex1","tmux_pane_id":"%1","stdin_path":"` + filepath.Join(tmp, "tmux.stdin") + `","rendered_command":"cat-cli Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","can_send_input":true,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryInteractive + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle interactivo: %v", err)
	}
	handle, err = db.GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}
	if got := db.RuntimeHandleMailboxDeliveryMode(handle); got != runtimeagente.MailboxDeliveryInteractive {
		t.Fatalf("modo interactivo inesperado: %s", got)
	}
	if !db.RuntimeHandlePermiteSendInputInteractivo(handle) {
		t.Fatalf("el handle tmux interactivo deberia permitir input")
	}

	msgViejo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"instruccion vieja"}`,
	})
	if err != nil {
		t.Fatalf("mailbox vieja: %v", err)
	}
	msgNuevo, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"instruccion nueva"}`,
	})
	if err != nil {
		t.Fatalf("mailbox nueva: %v", err)
	}

	n, err := procesarRuntimeMailboxInteractivoBatch()
	if err != nil {
		t.Fatalf("procesar mailbox interactivo: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia crear una sola send_instruction, got=%d", n)
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var send *db.RuntimeOrder
	for _, order := range orders {
		if order != nil && order.Tipo == "send_instruction" {
			send = order
			break
		}
	}
	if send == nil {
		t.Fatalf("faltaba send_instruction")
	}
	if !strings.Contains(send.PayloadJSON, "instruccion nueva") {
		t.Fatalf("deberia usar la instruction mas reciente: %s", send.PayloadJSON)
	}

	pendiente := "pendiente"
	consumido := "consumido"
	mailboxPendiente, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &pendiente})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailboxPendiente) != 1 || mailboxPendiente[0].ID != msgNuevo {
		t.Fatalf("deberia quedar pendiente la instruction vigente hasta entrega real: %+v", mailboxPendiente)
	}
	mailboxConsumido, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, Estado: &consumido})
	if err != nil {
		t.Fatalf("listar mailbox consumido: %v", err)
	}
	idsConsumidos := map[int64]struct{}{}
	for _, msg := range mailboxConsumido {
		if msg != nil {
			idsConsumidos[msg.ID] = struct{}{}
		}
	}
	if _, ok := idsConsumidos[msgViejo]; !ok {
		t.Fatalf("faltaba instruction vieja consumida")
	}
	if _, ok := idsConsumidos[msgNuevo]; ok {
		t.Fatalf("la instruction nueva no deberia marcarse consumida antes de la entrega real")
	}
}

func TestEncolarNudgeAutonomiaDetalladoNoDuplicaMailboxPendienteEnHandleNoInteractivo(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(
		`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`,
		`{"can_send_input":false}`,
		`{"driver":"process_pty_cli","rendered_command":"codex-perfil Codex1","can_send_input":false}`,
		handle.ID,
	); err != nil {
		t.Fatalf("update handle: %v", err)
	}

	runtimeOrderID := int64(91)
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:     "server",
		ToAgente:       "Codex1",
		ProyectoID:     &proyectoID,
		RuntimeOrderID: &runtimeOrderID,
		Kind:           "autonomia",
		PayloadJSON:    `{"accion":"continuar_trabajo","motivo":"seguir frente","texto":"seguir"}`,
	}); err != nil {
		t.Fatalf("crear mailbox existente: %v", err)
	}

	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	encolada, err := encolarNudgeAutonomiaDetallado("Codex1", proyecto, "continuar_trabajo", "seguir frente", "sigue trabajando", nil)
	if err != nil {
		t.Fatalf("encolar nudge autonomia: %v", err)
	}
	if encolada {
		t.Fatal("no deberia encolar un nuevo nudge cuando ya existe mailbox pendiente equivalente")
	}

	agente := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear orden adicional: %+v", orders)
	}

	estado := "pendiente"
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(mailbox) != 1 {
		t.Fatalf("mailbox inesperado tras dedupe: %+v", mailbox)
	}
}

func TestResetReanimacionEncolaStartCuandoNoHayRuntimePeroSiTrabajo(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar sin runtime",
		Descripcion: "Trabajo asignado",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar reanimacion: %v", err)
	}

	if err := (dbAutomationService{}).ResetReanimacion("Codex1"); err != nil {
		t.Fatalf("reset reanimacion: %v", err)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("start no encolado: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"accion":"start"`) {
		t.Fatalf("payload start inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestResetReanimacionCancelaHandoffObsoletoAntesDeArrancar(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar con handoff viejo",
		Descripcion: "Trabajo asignado",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	handoffID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "handoff",
		PayloadJSON: `{"resumen_continuidad":"handoff obsoleto"}`,
	})
	if err != nil {
		t.Fatalf("encolar handoff: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar reanimacion: %v", err)
	}

	if err := (dbAutomationService{}).ResetReanimacion("Codex1"); err != nil {
		t.Fatalf("reset reanimacion: %v", err)
	}

	handoff, err := db.GetRuntimeOrder(handoffID)
	if err != nil || handoff == nil {
		t.Fatalf("get handoff: %+v err=%v", handoff, err)
	}
	if handoff.Estado != "cancelada" {
		t.Fatalf("el handoff obsoleto deberia quedar cancelado: %+v", handoff)
	}
	if !strings.Contains(handoff.ResultadoJSON, `"superseded_reason":"manual_rehabilitation"`) {
		t.Fatalf("resultado handoff sin superseded_reason esperado: %s", handoff.ResultadoJSON)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var starts int
	for _, order := range orders {
		if order != nil && order.Tipo == "start" {
			starts++
		}
	}
	if starts != 1 {
		t.Fatalf("deberia encolar un unico start nuevo tras cancelar handoff obsoleto: %+v", orders)
	}
}

func TestEncolarContinuacionTareaReasignadaIncluyePayloadCanonico(t *testing.T) {
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
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}

	encolada, err := encolarContinuacionTareaReasignada(
		"Codex1",
		proyecto,
		77,
		"Gemma1",
		"worker_degradado",
		"continúa con la tarea reasignada y deja evidencia de avance",
		nil,
	)
	if err != nil {
		t.Fatalf("encolar continuacion: %v", err)
	}
	if !encolada {
		t.Fatal("debería encolar nudge")
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("runtime orders inesperadas: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"accion":"continuar_trabajo"`) ||
		!strings.Contains(orders[0].PayloadJSON, `"tarea_id":77`) ||
		!strings.Contains(orders[0].PayloadJSON, `"reasignada_desde":"Gemma1"`) ||
		!strings.Contains(orders[0].PayloadJSON, `"motivo":"worker_degradado"`) {
		t.Fatalf("payload inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestEncolarContinuacionTareaReasignadaPreservaExtras(t *testing.T) {
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
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}

	encolada, err := encolarContinuacionTareaReasignada(
		"Codex1",
		proyecto,
		88,
		"Claude1",
		"sobrecarga_operativa",
		"continúa con la tarea redistribuida y deja evidencia de avance",
		map[string]any{"redistribuida_desde": "Claude1"},
	)
	if err != nil {
		t.Fatalf("encolar continuacion: %v", err)
	}
	if !encolada {
		t.Fatal("debería encolar nudge")
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("runtime orders inesperadas: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"redistribuida_desde":"Claude1"`) {
		t.Fatalf("payload extras inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestEncolarContinuacionTareaReasignadaSiCorrespondeEvitaDuplicadoPendiente(t *testing.T) {
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
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	if _, err := encolarContinuacionTareaReasignada("Codex1", proyecto, 99, "Gemma1", "worker_degradado", "continúa", nil); err != nil {
		t.Fatalf("encolar continuacion inicial: %v", err)
	}

	if err := encolarContinuacionTareaReasignadaSiCorresponde("Codex1", &proyectoID, 99, "Gemma1", "worker_degradado", "continúa", nil); err != nil {
		t.Fatalf("encolar continuacion condicionada: %v", err)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("no deberia duplicar nudge pendiente: %+v", orders)
	}
}

func TestReasignarYArrancarTareaAutonomiaReasignaIniciaYAnota(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente origen: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente relevo: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{Slug: "orquestador", Nombre: "Orquestador", RutaAbs: filepath.Join(t.TempDir(), "orquestador"), Tipo: db.ProyectoRepo, Activo: true})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{Titulo: "Frente degradado", ProyectoID: &proyectoID, Prioridad: db.PrioridadAlta, CreadoPor: "tester"})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemma1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	if err := reasignarYArrancarTareaAutonomia(tareaID, "Codex1", "Reasignada automáticamente por degradación"); err != nil {
		t.Fatalf("reasignarYArrancarTareaAutonomia: %v", err)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente == nil || *tarea.Agente != "Codex1" || tarea.Estado != db.TareaEnProgreso {
		t.Fatalf("tarea inesperada: %+v", tarea)
	}
	if !strings.Contains(tarea.Notas, "Reasignada automáticamente por degradación") {
		t.Fatalf("nota inesperada: %+v", tarea)
	}
}

func TestBloquearTareaAutonomiaSinRelevoBloqueaYAnota(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemma1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{Slug: "orquestador", Nombre: "Orquestador", RutaAbs: filepath.Join(t.TempDir(), "orquestador"), Tipo: db.ProyectoRepo, Activo: true})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{Titulo: "Frente sin relevo", ProyectoID: &proyectoID, Prioridad: db.PrioridadAlta, CreadoPor: "tester"})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemma1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Gemma1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	if err := bloquearTareaAutonomiaSinRelevo(tareaID, "Gemma1", "Sobrecarga operativa: sin relevo sano disponible", "Bloqueada automáticamente por sobrecarga"); err != nil {
		t.Fatalf("bloquearTareaAutonomiaSinRelevo: %v", err)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Estado != db.TareaBloqueada {
		t.Fatalf("tarea no bloqueada: %+v", tarea)
	}
	if !strings.Contains(tarea.Notas, "Bloqueada automáticamente por sobrecarga") {
		t.Fatalf("nota inesperada: %+v", tarea)
	}
}

func TestProcesarTareasActivasFueraDeOrquestacionBatchBloqueaYAnota(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-fuera-flota",
		Nombre:  "Orquestador Fuera Flota",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Tarea huerfana",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET agente='CodexFantasma', estado=? WHERE id=?`, db.EstadoEnProgreso, tareaID); err != nil {
		t.Fatalf("forzar tarea huerfana: %v", err)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}

	procesadas, err := procesarTareasActivasFueraDeOrquestacionBatch(map[string][]*db.Tarea{"CodexFantasma": {tarea}}, map[string]agentesapp.Row{})
	if err != nil {
		t.Fatalf("procesar tareas fuera de orquestacion: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia bloquear exactamente una tarea, got=%d", procesadas)
	}

	tarea, err = db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea tras procesado: %v", err)
	}
	if tarea.Estado != db.EstadoBloqueada {
		t.Fatalf("la tarea deberia quedar bloqueada, got=%s", tarea.Estado)
	}
	if !strings.Contains(strings.ToLower(tarea.Notas), "fuera de orquestación") {
		t.Fatalf("nota inesperada: %+v", tarea)
	}
}

func TestResetReanimacionEncolaStartSiProyectoTieneBacklogRecuperable(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex7", "programador"); err != nil {
		t.Fatalf("registrar agente Codex7: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-reanimacion-backlog",
		Nombre:  "Orquestador Reanimacion Backlog",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente libre"); err != nil {
		t.Fatalf("activar asignacion Codex1: %v", err)
	}
	if err := db.ActivarAsignacion("Codex7", proyectoID, "frente cargado"); err != nil {
		t.Fatalf("activar asignacion Codex7: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Backlog bloqueado por sobrecarga",
		Descripcion: "Trabajo pendiente",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Codex7", "Sobrecarga operativa: sin relevo sano disponible"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar reanimacion: %v", err)
	}

	if err := (dbAutomationService{}).ResetReanimacion("Codex1"); err != nil {
		t.Fatalf("reset reanimacion: %v", err)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("deberia encolar start por backlog recuperable del proyecto: %+v", orders)
	}
}

func TestResetReanimacionNoArrancaSoloPorBloqueoHumanoAjeno(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex7", "programador"); err != nil {
		t.Fatalf("registrar agente Codex7: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-reanimacion-bloqueo-humano",
		Nombre:  "Orquestador Reanimacion Bloqueo Humano",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente libre"); err != nil {
		t.Fatalf("activar asignacion Codex1: %v", err)
	}
	if err := db.ActivarAsignacion("Codex7", proyectoID, "frente bloqueado"); err != nil {
		t.Fatalf("activar asignacion Codex7: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Esperando humano",
		Descripcion: "Trabajo externo",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Codex7", "esperando respuesta humana"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar reanimacion: %v", err)
	}

	if err := (dbAutomationService{}).ResetReanimacion("Codex1"); err != nil {
		t.Fatalf("reset reanimacion: %v", err)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia arrancar solo por bloqueo humano ajeno: %+v", orders)
	}
}

func TestResetReanimacionConservaCooldownSiFallaReactivacion(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO asignaciones (agente, proyecto_id, estado, nota) VALUES ('Codex1', 999999, 'activa', 'forzar error de proyecto')`); err != nil {
		t.Fatalf("insert asignacion inconsistente: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar reanimacion: %v", err)
	}

	if err := (dbAutomationService{}).ResetReanimacion("Codex1"); err == nil {
		t.Fatalf("esperaba error al reactivar con proyecto inconsistente")
	}

	agente, err := db.GetAgente("Codex1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agente == nil {
		t.Fatalf("agente nil")
	}
	if !strings.EqualFold(strings.TrimSpace(agente.EstadoCuota), "enfriamiento") {
		t.Fatalf("estado_cuota inesperado: %+v", agente)
	}
	if agente.ReanimarAt == nil {
		t.Fatalf("reanimar_at no deberia limpiarse si falla la reactivacion: %+v", agente)
	}
	if strings.TrimSpace(agente.MotivoPausa) == "" {
		t.Fatalf("motivo_pausa no deberia vaciarse si falla la reactivacion: %+v", agente)
	}
}

func TestResetReanimacionSostieneCooldownSiLaCuotaVisibleSigueAgotada(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	sesionID, err := db.IniciarSesion("Codex1")
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	now := time.Now().UTC()
	resetPrimary := now.Add(30 * time.Minute)
	resetSecondary := now.Add(72 * time.Hour)
	raw := `{"rate_limits":{"primary":{"used_percent":20,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":100,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetSecondary.Unix(), 10) + `}}}`
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "5h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: raw,
		CheckedAt:       now,
	}); err != nil {
		t.Fatalf("registrar presupuesto: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar reanimacion: %v", err)
	}

	if err := (dbAutomationService{}).ResetReanimacion("Codex1"); err != nil {
		t.Fatalf("reset reanimacion: %v", err)
	}

	agente, err := db.GetAgente("Codex1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agente == nil {
		t.Fatalf("agente nil")
	}
	if strings.TrimSpace(agente.EstadoCuota) != "enfriamiento" {
		t.Fatalf("estado_cuota inesperado: %+v", agente)
	}
	if agente.ReanimarAt == nil || agente.ReanimarAt.Before(resetSecondary.Add(-time.Minute)) {
		t.Fatalf("deberia sostener cooldown hasta el reset visible semanal: %+v want>=%s", agente, resetSecondary.Format(time.RFC3339))
	}
	estado := "pendiente"
	nombre := "Codex1"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &nombre, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia reactivar ni encolar ordenes mientras la cuota siga agotada: %+v", orders)
	}
}

func TestResetReanimacionNoArrancaSiLaCuentaCompartidaYaEstaOcupada(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.ConfigSet("runtime_shared_account_active_ceiling", "1"); err != nil {
		t.Fatalf("config shared account ceiling: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente Codex2: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-reanimacion-cuenta",
		Nombre:  "Orquestador Reanimacion Cuenta",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente retenido"); err != nil {
		t.Fatalf("activar asignacion Codex1: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar cuando haya cuenta libre",
		Descripcion: "Trabajo retenido por cuenta compartida",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	sesionID, err := db.IniciarSesion("Codex2")
	if err != nil {
		t.Fatalf("iniciar sesion Codex2: %v", err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET proyecto_id=?,
		    transporte='tmux',
		    handle_kind='session',
		    handle_ref='orq-codex2',
		    estado='activo',
		    metadata_json=?,
		    last_seen_at=CURRENT_TIMESTAMP
		WHERE sesion_id=?`,
		proyectoID,
		`{"driver":"tmux_cli_session","account_email":"maritere@avidad.com","account_user":"maritere"}`,
		sesionID,
	); err != nil {
		t.Fatalf("actualizar handle activo Codex2: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO runtime_handles (
			agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
		) VALUES (?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		"Codex1", proyectoID, "tmux", "session", "orq-codex1", "pausado",
		`{"driver":"tmux_cli_session","account_email":"maritere@avidad.com","account_user":"maritere"}`,
	); err != nil {
		t.Fatalf("registrar handle pausado Codex1: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='activo', motivo_pausa='' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar reanimacion: %v", err)
	}

	if err := (dbAutomationService{}).ResetReanimacion("Codex1"); err != nil {
		t.Fatalf("reset reanimacion: %v", err)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia reactivar si la cuenta compartida ya esta ocupada: %+v", orders)
	}
}

func TestPresupuestoAgenteDebeRevalidarseAhoraParaBloqueadoStale(t *testing.T) {
	checkedAt := time.Now().UTC().Add(-2 * time.Hour)
	if !presupuestoAgenteDebeRevalidarseAhora(&db.Agente{
		Nombre:               "Codex6",
		EstadoCuota:          "enfriamiento",
		PresupuestoStale:     true,
		PresupuestoCheckedAt: &checkedAt,
	}, time.Hour, true) {
		t.Fatalf("deberia revalidar un agente bloqueado y stale tras la ventana")
	}
}

func TestPresupuestoAgenteNoRevalidaAntesDeTiempo(t *testing.T) {
	checkedAt := time.Now().UTC().Add(-30 * time.Second)
	if presupuestoAgenteDebeRevalidarseAhora(&db.Agente{
		Nombre:               "Codex3",
		EstadoCuota:          "activo",
		PresupuestoCheckedAt: &checkedAt,
	}, time.Minute, false) {
		t.Fatalf("no deberia revalidar antes del preflight mínimo")
	}
}

func TestPresupuestoPrimerUsoSesionGate(t *testing.T) {
	presupuestoPrimerUsoSesionGate.Reset()

	if !presupuestoPrimerUsoSesion("Codex3") {
		t.Fatalf("el primer uso de la sesion deberia forzar refresh")
	}
	if presupuestoPrimerUsoSesion("Codex3") {
		t.Fatalf("el segundo uso de la misma sesion no deberia volver a marcar primer uso")
	}
	if !presupuestoPrimerUsoSesion("Codex4") {
		t.Fatalf("otro agente deberia mantener su propio primer uso")
	}
}

func TestRowPermiteAutoRecuperacionConWorkerFreshYSoloBloqueadas(t *testing.T) {
	now := time.Now().UTC()
	row := agentesapp.Row{
		EstadoOperativo:           "bloqueado",
		OpenTasks:                 0,
		BlockedTasks:              3,
		WorkerState:               "running",
		WorkerAlive:               true,
		WorkerHeartbeat:           &now,
		WorkerMailboxDeliveryMode: runtimeagente.MailboxDeliverySessionResume,
		WorkerExternalSessionID:   "sess-codex7",
	}
	if !rowPermiteAutoRecuperacion(row, now) {
		t.Fatalf("un worker vivo con backlog bloqueado deberia permitir auto recuperacion")
	}
}

func TestRowPermiteAutoRecuperacionRechazaWorkerSinContinuidadConSoloBloqueadas(t *testing.T) {
	now := time.Now().UTC()
	row := agentesapp.Row{
		EstadoOperativo:           "bloqueado",
		OpenTasks:                 0,
		BlockedTasks:              3,
		WorkerState:               "running",
		WorkerAlive:               true,
		WorkerHeartbeat:           &now,
		WorkerMailboxDeliveryMode: runtimeagente.MailboxDeliveryBootstrapOnly,
	}
	if rowPermiteAutoRecuperacion(row, now) {
		t.Fatalf("un worker bootstrap_only no deberia auto-recuperar backlog bloqueado")
	}
}

func TestRowPermiteAutoRecuperacionAceptaTMUXBootstrapOnlyFrescoConSoloBloqueadas(t *testing.T) {
	now := time.Now().UTC()
	row := agentesapp.Row{
		EstadoOperativo:           "bloqueado",
		OpenTasks:                 0,
		BlockedTasks:              2,
		WorkerState:               "starting",
		WorkerAlive:               true,
		WorkerHeartbeat:           &now,
		WorkerDriver:              "tmux_cli_session",
		WorkerTMUXSession:         "orq-gemini1-053508",
		WorkerMailboxDeliveryMode: runtimeagente.MailboxDeliveryBootstrapOnly,
	}
	if !rowPermiteAutoRecuperacion(row, now) {
		t.Fatalf("un worker tmux bootstrap_only fresco deberia reactivar backlog bloqueado")
	}
}

func TestProcesarAutonomiaAgentesBatchEncolaPausePorBloqueoHumano(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Esperando respuesta humana",
		Descripcion: "Bloqueo operativo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Codex1", "esperando respuesta humana"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "pause" {
		t.Fatalf("pause no encolada: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, "bloqueo_humano:esperando respuesta humana") {
		t.Fatalf("payload pause inesperado: %s", orders[0].PayloadJSON)
	}
	op, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != db.ProyectoOperativoEsperandoHumano {
		t.Fatalf("estado operativo inesperado: %+v", op)
	}
}

func TestProcesarAutonomiaAgentesBatchNoDuplicaPauseSiYaEstaPausadoPorCuota(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Agente en enfriamiento",
		Descripcion: "No debe crear pause duplicada",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET reanimar_at=CURRENT_TIMESTAMP, estado_cuota='enfriamiento', motivo_pausa='cuota' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar enfriamiento: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia bloquear la tarea activa sin duplicar pause, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil && err != sql.ErrNoRows {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear pause duplicada: %+v", orders)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Estado != db.EstadoBloqueada {
		t.Fatalf("la tarea deberia quedar bloqueada al estar el agente en cuota: %+v", tarea)
	}
}

func TestProcesarAutonomiaAgentesBatchAparcaSesionBloqueadaYPausaAsignacion(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Consulta a humano",
		Descripcion: "Esperando desbloqueo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if err := db.BloquearTarea(tareaID, "Codex1", "esperando decision humana"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE sesiones SET estado='pausada' WHERE id=?`, sesion.ID); err != nil {
		t.Fatalf("pausar sesion: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma, got=%d", n)
	}

	sesionRecargada, err := db.GetSesionByID(sesion.ID)
	if err != nil {
		t.Fatalf("get sesion: %v", err)
	}
	if sesionRecargada.Activa {
		t.Fatalf("la sesion deberia quedar aparcada: %+v", sesionRecargada)
	}
	agenteRecargado, err := db.GetAgente("Codex1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agenteRecargado.Activo || agenteRecargado.EstadoSesion != "" {
		t.Fatalf("agente no liberado tras aparcado: %+v", agenteRecargado)
	}
	handles, err := db.ListarRuntimeHandles(&agenteRecargado.Nombre)
	if err != nil {
		t.Fatalf("listar handles: %v", err)
	}
	if len(handles) == 0 || handles[0].Estado != "cerrado" {
		t.Fatalf("handle no cerrado tras aparcado: %+v", handles)
	}
	agente := "Codex1"
	asignaciones, err := db.ListarAsignaciones(db.FiltroAsignaciones{Agente: &agente})
	if err != nil {
		t.Fatalf("listar asignaciones: %v", err)
	}
	if len(asignaciones) == 0 || asignaciones[0].Estado != db.AsignacionPausada {
		t.Fatalf("asignacion no pausada: %+v", asignaciones)
	}
	op, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != db.ProyectoOperativoEsperandoHumano {
		t.Fatalf("estado operativo inesperado tras aparcado: %+v", op)
	}
}

func TestProcesarAutonomiaAgentesBatchRespetaEstadoOperativoProyectoEsperandoHumano(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.MarcarProyectoEsperandoHumano(proyectoID, "esperando decision externa"); err != nil {
		t.Fatalf("marcar proyecto esperando humano: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "pause" {
		t.Fatalf("pause no encolada por estado operativo del proyecto: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, "bloqueo_humano:esperando decision externa") {
		t.Fatalf("payload pause inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarAutonomiaAgentesBatchReanudaSesionPausadaSiHayTrabajoActivoYBloqueoHumanoStale(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaActivaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Implementar cambio",
		Descripcion: "Trabajo activo real",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea activa: %v", err)
	}
	if err := db.TomarTarea(tareaActivaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea activa: %v", err)
	}
	if err := db.IniciarTarea(tareaActivaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea activa: %v", err)
	}
	tareaBloqueadaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Esperando respuesta",
		Descripcion: "Bloqueo parcial",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea bloqueada: %v", err)
	}
	if err := db.TomarTarea(tareaBloqueadaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea bloqueada: %v", err)
	}
	if err := db.IniciarTarea(tareaBloqueadaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea bloqueada: %v", err)
	}
	if err := db.BloquearTarea(tareaBloqueadaID, "Codex1", "esperando decision humana"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE sesiones SET estado='pausada' WHERE id=?`, sesion.ID); err != nil {
		t.Fatalf("pausar sesion: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}
	if err := db.PausarAsignacion("Codex1", proyectoID, "bloqueo_humano:esperando decision humana"); err != nil {
		t.Fatalf("pausar asignacion: %v", err)
	}
	if err := db.MarcarProyectoEsperandoHumano(proyectoID, "esperando decision humana"); err != nil {
		t.Fatalf("marcar proyecto esperando humano: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n < 0 {
		t.Fatalf("resultado inesperado: %d", n)
	}

	op, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != db.ProyectoOperativoActivo {
		t.Fatalf("el proyecto deberia reactivarse al detectar trabajo activo: %+v", op)
	}
	asignaciones, err := db.ListarAsignaciones(db.FiltroAsignaciones{Agente: strPtr("Codex1")})
	if err != nil {
		t.Fatalf("listar asignaciones: %v", err)
	}
	if len(asignaciones) == 0 || asignaciones[0].Estado != db.AsignacionActiva {
		t.Fatalf("la asignacion deberia reactivarse: %+v", asignaciones)
	}
	sesionRecargada, err := db.GetSesionByID(sesion.ID)
	if err != nil {
		t.Fatalf("get sesion: %v", err)
	}
	if !strings.EqualFold(strings.TrimSpace(sesionRecargada.Estado), "activa") {
		t.Fatalf("la sesion deberia quedar activa: %+v", sesionRecargada)
	}
	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "pause" && strings.Contains(order.PayloadJSON, "bloqueo_humano:") {
			t.Fatalf("no deberia volver a encolar una pausa por bloqueo humano: %+v", orders)
		}
	}
}

func TestProcesarAutonomiaAgentesBatchNoPropagaBloqueoHumanoDesdeAutobloqueoDeOtroAgente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	for _, agente := range []string{"Codex3", "Codex7"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
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

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Autobloqueo runtime",
		Descripcion: "No debe bloquear a otros agentes del proyecto",
		ProyectoID:  &proyectoID,
		Modulo:      "runtime",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Codex7", "Agente Codex7 en estado bloqueado_por_runtime: pausado"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex3",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE sesiones SET estado='activa' WHERE id=?`, sesion.ID); err != nil {
		t.Fatalf("activar sesion: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n < 0 {
		t.Fatalf("resultado inesperado: %d", n)
	}

	op, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != db.ProyectoOperativoActivo {
		t.Fatalf("el proyecto no deberia quedar esperando_humano por autobloqueo de otro agente: %+v", op)
	}
	agente := "Codex3"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	for _, order := range orders {
		if order != nil && order.Tipo == "pause" && strings.Contains(order.PayloadJSON, "bloqueo_humano:") {
			t.Fatalf("no deberia pausar a otro agente por autobloqueo ajeno: %+v", orders)
		}
	}
}

func TestProcesarAutonomiaAgentesBatchReanudaSesionPausadaConTrabajoActivoYProyectoOperativo(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Reanudar worker",
		Descripcion: "Trabajo activo real",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE sesiones SET estado='pausada' WHERE id=?`, sesion.ID); err != nil {
		t.Fatalf("pausar sesion: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}
	if err := db.PausarAsignacion("Codex1", proyectoID, "pausa espuria"); err != nil {
		t.Fatalf("pausar asignacion: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n < 0 {
		t.Fatalf("resultado inesperado: %d", n)
	}

	asignaciones, err := db.ListarAsignaciones(db.FiltroAsignaciones{Agente: strPtr("Codex1")})
	if err != nil {
		t.Fatalf("listar asignaciones: %v", err)
	}
	if len(asignaciones) == 0 || asignaciones[0].Estado != db.AsignacionActiva {
		t.Fatalf("la asignacion deberia reactivarse: %+v", asignaciones)
	}
	sesionRecargada, err := db.GetSesionByID(sesion.ID)
	if err != nil {
		t.Fatalf("get sesion: %v", err)
	}
	if !strings.EqualFold(strings.TrimSpace(sesionRecargada.Estado), "activa") {
		t.Fatalf("la sesion deberia quedar activa: %+v", sesionRecargada)
	}
}

func TestProcesarAutonomiaAgentesBatchReutilizaHandleActivoAlReactivarSesionPausada(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Seguir trabajo",
		Descripcion: "Sesion pausada con handle vivo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE sesiones SET estado='pausada' WHERE id=?`, sesion.ID); err != nil {
		t.Fatalf("pausar sesion: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n < 1 {
		t.Fatalf("esperaba al menos una decision de reactivacion, got=%d", n)
	}

	sesionRecargada, err := db.GetSesionByID(sesion.ID)
	if err != nil {
		t.Fatalf("get sesion: %v", err)
	}
	if !strings.EqualFold(strings.TrimSpace(sesionRecargada.Estado), "activa") {
		t.Fatalf("la sesion deberia quedar activa: %+v", sesionRecargada)
	}
	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	foundResume := false
	for _, order := range orders {
		if order == nil {
			continue
		}
		if order.Tipo == "start" {
			t.Fatalf("no deberia duplicar start con handle ya activo: %+v", orders)
		}
		if order.Tipo == "resume" {
			foundResume = true
		}
	}
	if !foundResume {
		t.Fatalf("deberia encolar resume para una sesion pausada con handle activo: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchCierraProyectoTerminado(t *testing.T) {
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:       proyectoID,
		Enabled:          true,
		ObjetivoGeneral:  "Terminar la app",
		AutoCloseProject: true,
		EstadoAutonomia:  db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Entrega final",
		Descripcion: "Proyecto listo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "Codex1", "entrega completada"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma, got=%d", n)
	}

	op, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != db.ProyectoOperativoCerrado {
		t.Fatalf("proyecto no marcado cerrado: %+v", op)
	}
	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "pause" || !strings.Contains(orders[0].PayloadJSON, "proyecto_terminado:") {
		t.Fatalf("pause de cierre inesperada: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchRecuperaRuntimeRemotoDegradado(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	conectorID, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-remote",
		Nombre:     "Codex Remote",
		Transporte: "api",
		Comando:    "http://localhost:9999",
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Recuperar runtime remoto",
		Descripcion: "Trabajo activo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ConectorID:         &conectorID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-remote",
		ExternalSessionID:  "sess-remote-degraded",
		ResumenContinuidad: "continuidad activa",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET transporte='api',
		    estado='fallido',
		    metadata_json='{\"remote_sync_failures\":3,\"remote_last_error\":\"adapter down\"}'
		WHERE id = ?`, handle.ID); err != nil {
		t.Fatalf("degradar handle: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='degradado', process_state='remote_status_error'
		WHERE id = ?`, runtime.ID); err != nil {
		t.Fatalf("degradar runtime: %v", err)
	}

	sesionActual, err := db.GetSesionByID(sesion.ID)
	if err != nil || sesionActual == nil {
		t.Fatalf("get sesion actual: %+v err=%v", sesionActual, err)
	}
	n, err := procesarRecuperacionRuntimeDegradadoSesion(sesionActual)
	if err != nil {
		t.Fatalf("procesar recuperacion degradada: %v", err)
	}
	if n != 2 {
		t.Fatalf("esperaba 2 decisiones autonomas (checkpoint + resume), got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("orders autonomas inesperadas: %+v", orders)
	}
	var checkpointFound, resumeFound bool
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch order.Tipo {
		case "checkpoint":
			if order.RuntimeID == nil || *order.RuntimeID != runtime.ID {
				t.Fatalf("checkpoint deberia usar el runtime canónico: %+v", order)
			}
			checkpointFound = strings.Contains(order.PayloadJSON, "remote_runtime_degraded")
		case "resume":
			resumeFound = strings.Contains(order.PayloadJSON, `"motivo":"remote_runtime_degraded"`)
		}
	}
	if !checkpointFound || !resumeFound {
		t.Fatalf("recuperacion autonoma remota incompleta: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchRecuperaRuntimeRemotoDegradadoSinSesionExternaRelanzaStart(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	conectorID, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-remote",
		Nombre:     "Codex Remote",
		Transporte: "api",
		Comando:    "http://localhost:9999",
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ConectorID:         &conectorID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-remote",
		ResumenContinuidad: "continuidad activa",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET transporte='api',
		    handle_kind='session',
		    handle_ref=?,
		    estado='fallido',
		    metadata_json='{"remote_sync_failures":3,"remote_last_error":"adapter down"}'
		WHERE id = ?`, strconv.FormatInt(sesion.ID, 10), handle.ID); err != nil {
		t.Fatalf("degradar handle: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='degradado', process_state='remote_status_error'
		WHERE id = ?`, runtime.ID); err != nil {
		t.Fatalf("degradar runtime: %v", err)
	}

	sesionActual, err := db.GetSesionByID(sesion.ID)
	if err != nil || sesionActual == nil {
		t.Fatalf("get sesion actual: %+v err=%v", sesionActual, err)
	}
	n, err := procesarRecuperacionRuntimeDegradadoSesion(sesionActual)
	if err != nil {
		t.Fatalf("procesar recuperacion degradada: %v", err)
	}
	if n != 2 {
		t.Fatalf("esperaba 2 decisiones autonomas (checkpoint + start), got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("orders autonomas inesperadas: %+v", orders)
	}
	var checkpointFound, startFound bool
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch order.Tipo {
		case "checkpoint":
			if order.RuntimeID == nil || *order.RuntimeID != runtime.ID {
				t.Fatalf("checkpoint deberia usar el runtime canónico: %+v", order)
			}
			checkpointFound = strings.Contains(order.PayloadJSON, "remote_runtime_degraded")
		case "start":
			startFound = strings.Contains(order.PayloadJSON, `"motivo":"remote_runtime_degraded"`)
		}
	}
	if !checkpointFound || !startFound {
		t.Fatalf("recuperacion autonoma remota por start incompleta: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchRecuperaOllamaPoolLocalDegradadoConStart(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("GemmaPool1", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("GemmaPool1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	conectorID, err := db.UpsertConector(&db.Conector{
		Slug:       "ollama_pool_local",
		Nombre:     "Ollama Pool Local",
		Transporte: "api",
		Comando:    "http://127.0.0.1:17731",
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:            "GemmaPool1",
		ConectorID:        &conectorID,
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "orquestador"),
		Herramienta:       "ollama_pool_local",
		ExternalSessionID: "ollama-pool-1",
		ResumePayloadJSON: `{"driver":"ollama_pool_local","transport":"api","endpoint":"http://127.0.0.1:17731","input_path":"/api/runtime/ollama-pool/input","status_path":"/api/runtime/ollama-pool/status","stop_path":"/api/runtime/ollama-pool/stop","mailbox_delivery_mode":"interactive","can_send_input":true}`,
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET transporte='api',
		    handle_kind='session',
		    handle_ref='ollama-pool-1',
		    estado='fallido',
		    metadata_json='{"driver":"ollama_pool_local","transport":"api","pool_local":true,"external_session_id":"ollama-pool-1"}'
		WHERE id = ?`, handle.ID); err != nil {
		t.Fatalf("degradar handle: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='degradado', process_state='remote_status_error'
		WHERE id = ?`, runtime.ID); err != nil {
		t.Fatalf("degradar runtime: %v", err)
	}

	sesionActual, err := db.GetSesionByID(sesion.ID)
	if err != nil || sesionActual == nil {
		t.Fatalf("get sesion actual: %+v err=%v", sesionActual, err)
	}
	n, err := procesarRecuperacionRuntimeDegradadoSesion(sesionActual)
	if err != nil {
		t.Fatalf("procesar recuperacion degradada: %v", err)
	}
	if n != 2 {
		t.Fatalf("esperaba 2 decisiones autonomas (checkpoint + start), got=%d", n)
	}

	agente := "GemmaPool1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var checkpointFound, startFound, resumeFound bool
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch order.Tipo {
		case "checkpoint":
			checkpointFound = strings.Contains(order.PayloadJSON, "remote_runtime_degraded")
		case "start":
			startFound = strings.Contains(order.PayloadJSON, `"motivo":"remote_runtime_degraded"`)
		case "resume":
			resumeFound = true
		}
	}
	if !checkpointFound || !startFound || resumeFound {
		t.Fatalf("recuperacion ollama pool local inesperada: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchRecuperaRuntimeLocalFallidoRelanzaStart(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Recuperar runtime local",
		Descripcion: "Hay trabajo real pendiente",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	conector, err := db.GetConector("codex-cli")
	if err != nil || conector == nil {
		t.Fatalf("get conector codex-cli: %+v err=%v", conector, err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ConectorID:         &conector.ID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ResumePayloadJSON:  db.MergeResumePayloadPerfilEjecucion("", "implementacion", "gemma4:26b", "medium"),
		ResumenContinuidad: "continuidad activa",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET transporte='cli',
		    handle_kind='process',
		    handle_ref='999999',
		    estado='fallido'
		WHERE id = ?`, handle.ID); err != nil {
		t.Fatalf("degradar handle local: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='fallido', process_state='fallido'
		WHERE id = ?`, runtime.ID); err != nil {
		t.Fatalf("degradar runtime local: %v", err)
	}

	sesionActual, err := db.GetSesionByID(sesion.ID)
	if err != nil || sesionActual == nil {
		t.Fatalf("get sesion actual: %+v err=%v", sesionActual, err)
	}
	n, err := procesarRecuperacionRuntimeDegradadoSesion(sesionActual)
	if err != nil {
		t.Fatalf("procesar recuperacion local fallida: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma (start), got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("recuperacion autonoma local inesperada: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"motivo":"local_runtime_failed"`) {
		t.Fatalf("start de recuperacion local sin motivo esperado: %s", orders[0].PayloadJSON)
	}
	for _, token := range []string{`"perfil":"implementacion"`, `"modelo":"gemma4:26b"`, `"razonamiento":"medium"`} {
		if !strings.Contains(orders[0].PayloadJSON, token) {
			t.Fatalf("start de recuperacion local sin perfil persistido %s: %s", token, orders[0].PayloadJSON)
		}
	}
}

func TestProcesarAutonomiaAgentesBatchNoRelanzaRuntimeLocalFallidoSinTrabajoArrancable(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	conector, err := db.GetConector("codex-cli")
	if err != nil || conector == nil {
		t.Fatalf("get conector codex-cli: %+v err=%v", conector, err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ConectorID:         &conector.ID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ResumenContinuidad: "continuidad activa",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET transporte='cli',
		    handle_kind='process',
		    handle_ref='999999',
		    estado='fallido'
		WHERE id = ?`, handle.ID); err != nil {
		t.Fatalf("degradar handle local: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='fallido', process_state='fallido'
		WHERE id = ?`, runtime.ID); err != nil {
		t.Fatalf("degradar runtime local: %v", err)
	}

	sesionActual, err := db.GetSesionByID(sesion.ID)
	if err != nil || sesionActual == nil {
		t.Fatalf("get sesion actual: %+v err=%v", sesionActual, err)
	}
	n, err := procesarRecuperacionRuntimeDegradadoSesion(sesionActual)
	if err != nil {
		t.Fatalf("procesar recuperacion local fallida sin trabajo: %v", err)
	}
	if n != 0 {
		t.Fatalf("sin trabajo arrancable no deberia relanzar start, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("sin trabajo arrancable no deberia encolar ordenes: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchNoRelanzaRuntimeLocalSiSigueVivo(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Runtime local vivo",
		Descripcion: "No debe relanzarse si el proceso observado sigue vivo",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	conector, err := db.GetConector("codex-cli")
	if err != nil || conector == nil {
		t.Fatalf("get conector codex-cli: %+v err=%v", conector, err)
	}
	pid := int64(os.Getpid())
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ConectorID:         &conector.ID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		PID:                &pid,
		ResumenContinuidad: "continuidad activa",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET transporte='cli',
		    handle_kind='process',
		    handle_ref=?,
		    estado='fallido'
		WHERE id = ?`, strconv.FormatInt(pid, 10), handle.ID); err != nil {
		t.Fatalf("degradar handle local vivo: %v", err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET pid=?, logical_state='fallido', process_state='fallido'
		WHERE id = ?`, pid, runtime.ID); err != nil {
		t.Fatalf("degradar runtime local vivo: %v", err)
	}

	sesionActual, err := db.GetSesionByID(sesion.ID)
	if err != nil || sesionActual == nil {
		t.Fatalf("get sesion actual: %+v err=%v", sesionActual, err)
	}
	n, err := procesarRecuperacionRuntimeDegradadoSesion(sesionActual)
	if err != nil {
		t.Fatalf("procesar recuperacion local viva: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia relanzar start si el proceso local sigue vivo, got=%d", n)
	}

	handle, err = db.GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle tras recuperacion: %+v err=%v", handle, err)
	}
	if handle.Estado != "activo" {
		t.Fatalf("el handle vivo deberia revivir a activo: %+v", handle)
	}
	runtime, err = db.GetRuntime(runtime.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime tras recuperacion: %+v err=%v", runtime, err)
	}
	if strings.EqualFold(strings.TrimSpace(runtime.ProcessState), "fallido") {
		t.Fatalf("el runtime vivo no deberia seguir fallido: %+v", runtime)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia encolar ordenes al revivir handle vivo: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchNoRecuperaSesionViejaSiYaHayTMUXOperativoReciente(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "No duplicar recovery",
		Descripcion: "Un tmux activo reciente debe ganar sobre una sesion vieja degradada",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	conector, err := db.GetConector("codex-cli")
	if err != nil || conector == nil {
		t.Fatalf("get conector codex-cli: %+v err=%v", conector, err)
	}

	sesionVieja, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ConectorID:         &conector.ID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ResumenContinuidad: "sesion vieja degradada",
	})
	if err != nil {
		t.Fatalf("iniciar sesion vieja: %v", err)
	}
	handleViejo, err := db.GetRuntimeHandleBySesionID(sesionVieja.ID)
	if err != nil || handleViejo == nil {
		t.Fatalf("get handle viejo: %+v err=%v", handleViejo, err)
	}
	runtimeViejo, err := db.GetRuntimeBySesionID(sesionVieja.ID)
	if err != nil || runtimeViejo == nil {
		t.Fatalf("get runtime viejo: %+v err=%v", runtimeViejo, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET transporte='cli',
		    handle_kind='process',
		    handle_ref='999999',
		    estado='fallido'
		WHERE id = ?`, handleViejo.ID); err != nil {
		t.Fatalf("degradar handle viejo: %v", err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='fallido', process_state='fallido'
		WHERE id = ?`, runtimeViejo.ID); err != nil {
		t.Fatalf("degradar runtime viejo: %v", err)
	}

	sesionNueva, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ConectorID:         &conector.ID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-codex1-fresh",
		ResumenContinuidad: "sesion nueva tmux",
	})
	if err != nil {
		t.Fatalf("iniciar sesion nueva: %v", err)
	}
	handleNuevo, err := db.GetRuntimeHandleBySesionID(sesionNueva.ID)
	if err != nil || handleNuevo == nil {
		t.Fatalf("get handle nuevo: %+v err=%v", handleNuevo, err)
	}
	runtimeNuevo, err := db.GetRuntimeBySesionID(sesionNueva.ID)
	if err != nil || runtimeNuevo == nil {
		t.Fatalf("get runtime nuevo: %+v err=%v", runtimeNuevo, err)
	}

	workerDir := filepath.Join(tmp, "worker-fresh")
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
		ChildPID:      4444,
		StatusPath:    statusPath,
		HeartbeatPath: heartbeatPath,
	})
	statusData, _ := json.Marshal(runtimeagente.WorkerStatus{
		State:     "running",
		UpdatedAt: now.Format(time.RFC3339),
		Alive:     true,
		ChildPID:  4444,
		Agent:     "Codex1",
		Project:   "orquestador",
	})
	heartbeatData, _ := json.Marshal(runtimeagente.WorkerHeartbeat{
		Alive:       true,
		HeartbeatAt: now.Format(time.RFC3339),
		StartedAt:   now.Add(-time.Minute).Format(time.RFC3339),
		ChildPID:    4444,
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
		"driver":                 "tmux_cli_session",
		"tmux_session":           "orq-codex1-fresh",
		"worker_manifest_path":   manifestPath,
		"worker_status_path":     statusPath,
		"worker_heartbeat_path":  heartbeatPath,
		"external_session_id":    "sess-codex1-fresh",
		"mailbox_delivery_mode":  runtimeagente.MailboxDeliverySessionResume,
		"profile_status_wrapper": "/tmp/codex-perfil",
	})
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET transporte='tmux',
		    handle_kind='session',
		    handle_ref='orq-codex1-fresh/%1',
		    estado='activo',
		    runtime_id=?,
		    metadata_json=?,
		    last_seen_at=?
		WHERE id = ?`, runtimeNuevo.ID, string(metaJSON), now, handleNuevo.ID); err != nil {
		t.Fatalf("actualizar handle nuevo: %v", err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET logical_state='esperando_io', process_state='running'
		WHERE id = ?`, runtimeNuevo.ID); err != nil {
		t.Fatalf("actualizar runtime nuevo: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	sesionActual, err := db.GetSesionByID(sesionVieja.ID)
	if err != nil || sesionActual == nil {
		t.Fatalf("get sesion actual: %+v err=%v", sesionActual, err)
	}
	n, err := procesarRecuperacionRuntimeDegradadoSesion(sesionActual)
	if err != nil {
		t.Fatalf("procesar recuperacion vieja con tmux fresco: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia relanzar recovery si ya existe tmux operativo reciente, got=%d", n)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia encolar ordenes con tmux reciente operativo: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchRecuperaSesionActivaSinHandleConTrabajoArrancable(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "frente premium"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar continuidad premium",
		Descripcion: "Trabajo premium ya asignado",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	conector, err := db.GetConector("gemini-cli")
	if err != nil || conector == nil {
		t.Fatalf("get conector gemini-cli: %+v err=%v", conector, err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Gemini1",
		ConectorID:         &conector.ID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "gemini-cli",
		ResumenContinuidad: "continuidad activa",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`DELETE FROM runtime_handles WHERE id = ?`, handle.ID); err != nil {
		t.Fatalf("borrar handle: %v", err)
	}

	sesionActual, err := db.GetSesionByID(sesion.ID)
	if err != nil || sesionActual == nil {
		t.Fatalf("get sesion actual: %+v err=%v", sesionActual, err)
	}
	n, err := procesarRecuperacionRuntimeDegradadoSesion(sesionActual)
	if err != nil {
		t.Fatalf("procesar recuperacion sin handle: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma (start), got=%d", n)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("recuperacion autonoma sin handle inesperada: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"motivo":"local_runtime_missing"`) {
		t.Fatalf("start de recuperacion sin handle sin motivo esperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchReactivaPremiumSinRuntimeNiHandle(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("QwenCoder1", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("QwenCoder1", proyectoID, "frente premium"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar continuidad premium",
		Descripcion: "Trabajo premium ya asignado",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "QwenCoder1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "QwenCoder1"); err != nil {
		t.Fatalf("arrancar tarea: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "QwenCoder1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"esperar_o_pedir_tarea","bootstrap":true,"texto":"arranque_autonomo_servidor"}`,
	}); err != nil {
		t.Fatalf("mailbox autonomia: %v", err)
	}

	n, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia reactivar premium sin runtime/handle, got=%d", n)
	}

	agente := "QwenCoder1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("deberia encolar start para premium sin runtime/handle: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"motivo":"agente_sin_runtime_activo"`) {
		t.Fatalf("start sin motivo esperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchReactivaPremiumConHandleFallidoYRuntimeDegradado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar frente con runtime degradado",
		Descripcion: "Trabajo premium ya asignado",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	runtimeActual, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtimeActual == nil {
		t.Fatalf("get runtime: %+v err=%v", runtimeActual, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='fallido' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("degradar handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='fallido', process_state='fallido' WHERE id=?`, runtimeActual.ID); err != nil {
		t.Fatalf("degradar runtime: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	n, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia reactivar premium con handle fallido/runtime degradado, got=%d", n)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) == 0 || orders[0].Tipo != "start" {
		t.Fatalf("deberia encolar start para runtime degradado: %+v", orders)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchReactivaPremiumConTareaBloqueadaYRuntimeCaido(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar frente bloqueado",
		Descripcion: "Trabajo premium bloqueado por degradacion operativa",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Gemini1", "Agente Gemini1 en estado bloqueado_por_runtime: runtime principal stale"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	runtimeActual, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtimeActual == nil {
		t.Fatalf("get runtime: %+v err=%v", runtimeActual, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='fallido' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("degradar handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='fallido', process_state='fallido' WHERE id=?`, runtimeActual.ID); err != nil {
		t.Fatalf("degradar runtime: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	n, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia reactivar premium con tarea bloqueada y runtime caido, got=%d", n)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) == 0 || orders[0].Tipo != "start" {
		t.Fatalf("deberia encolar start para tarea bloqueada recuperable: %+v", orders)
	}
	actual, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if actual.Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea bloqueada deberia reactivarse al reponer runtime: %+v", actual)
	}
}


func TestDesbloquearTareasBloqueadasRecuperablesSinRuntimeReabreBloqueoSinRelevoSano(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "microciclo_exclusivo"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Retomar frente bloqueado sin relevo sano",
		Descripcion: "Trabajo premium bloqueado por degradacion operativa sin relevo",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "orquesta", "Bloqueada automáticamente por degradación operativa sin relevo sano"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}
	tareasBloqueadasPorAgente := map[string][]*db.Tarea{
		"Gemini1": {{ID: tareaID, Agente: ptrString("Gemini1"), ProyectoID: &proyectoID, Estado: db.TareaBloqueada}},
	}
	resumenBloqueos, err := db.ListarResumenBloqueos()
	if err != nil {
		t.Fatalf("listar resumen bloqueos: %v", err)
	}
	bloqueosPorTarea := map[int64]db.ResumenBloqueo{}
	for _, bloqueo := range resumenBloqueos {
		bloqueosPorTarea[bloqueo.ID] = bloqueo
	}

	n, err := desbloquearTareasBloqueadasRecuperablesSinRuntime("Gemini1", proyectoID, tareasBloqueadasPorAgente["Gemini1"], bloqueosPorTarea)
	if err != nil {
		t.Fatalf("desbloquear tareas recuperables sin runtime: %v", err)
	}
	if n != 1 {
		t.Fatalf("deberia reabrir la tarea bloqueada, got=%d", n)
	}
	actual, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if actual.Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea deberia volver a en_progreso: %+v", actual)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchAutoasignaPremiumLibreSinSesion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		Prioridad:        100,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "frente premium"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO runtime_handles (agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at)
		VALUES ('Gemini1', ?, 'tmux', 'session', 'orq-gemini1-fallida', 'fallido', '{"driver":"tmux_cli_session"}', CURRENT_TIMESTAMP)
	`, proyectoID); err != nil {
		t.Fatalf("insert handle fallido: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      microcicloRefactorTituloDefault,
		Descripcion: descripcionMicrocicloDefault(&db.Proyecto{Slug: "orquestador", RutaAbs: filepath.Join(tmp, "orquestador")}),
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea libre: %v", err)
	}

	n, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia autoasignar premium libre sin sesion y encolar start, got=%d", n)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente == nil || strings.TrimSpace(*tarea.Agente) != "Gemini1" || tarea.Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea libre deberia quedar tomada y arrancada en Gemini1: %+v", tarea)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("deberia encolar start para premium idle sin sesion: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"motivo":"premium_idle_autoassigned"`) {
		t.Fatalf("start sin motivo esperado: %s", orders[0].PayloadJSON)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchAbreFrentePremiumSiNoHayLibreDeterminista(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente actual: %v", err)
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
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		Prioridad:        100,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:       proyectoID,
		Enabled:          true,
		MaxWorkers:       3,
		SupervisorAgente: "orquesta",
		EstadoAutonomia:  db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("Gemini1", proyectoID, "frente premium"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO runtime_handles (agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at)
		VALUES ('Gemini1', ?, 'tmux', 'session', 'orq-gemini1-fallida', 'fallido', '{"driver":"tmux_cli_session"}', CURRENT_TIMESTAMP)
	`, proyectoID); err != nil {
		t.Fatalf("insert handle fallido: %v", err)
	}
	tareaActualID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Trabajo premium ya en progreso",
		Descripcion: "WRITE_SET: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestProcesarAutonomiaAgentesBatch.*'",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea actual: %v", err)
	}
	if err := db.TomarTarea(tareaActualID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea actual: %v", err)
	}
	if err := db.IniciarTarea(tareaActualID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea actual: %v", err)
	}

	n, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia abrir o reutilizar un frente premium y encolar start, got=%d", n)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	var abierta *db.Tarea
	for _, tarea := range tareas {
		if tarea == nil || tarea.ID == tareaActualID {
			continue
		}
		if tarea.Agente != nil && strings.TrimSpace(*tarea.Agente) == "Gemini1" &&
			(tarea.Estado == db.TareaAsignada || tarea.Estado == db.TareaEnProgreso) {
			abierta = tarea
			break
		}
	}
	if abierta == nil {
		t.Fatalf("deberia dejar un frente premium activo para Gemini1, tareas=%+v", tareas)
	}

	agente := "Gemini1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("deberia encolar start para premium idle con frente derivado: %+v", orders)
	}
}

func TestProcesarAutonomiaAgentesBatchBloqueaProyectoPorCircuitoAbiertoConector(t *testing.T) {
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
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-remote",
		Nombre:     "Codex Remote",
		Transporte: "api",
		Comando:    "http://localhost:9999",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-remote",
		ExternalSessionID:  "sess-remote-blocked",
		ResumenContinuidad: "continuidad activa",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_handles
		SET transporte='api', estado='fallido', metadata_json='{"conector":"codex-remote","remote_sync_failures":3}'
		WHERE id = ?`, handle.ID); err != nil {
		t.Fatalf("degradar handle: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("get runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`
		UPDATE runtime_instances
		SET connector='codex-remote', logical_state='degradado', process_state='remote_status_error'
		WHERE id = ?`, runtime.ID); err != nil {
		t.Fatalf("degradar runtime: %v", err)
	}
	conector, err := db.GetConector("codex-remote")
	if err != nil {
		t.Fatalf("get conector: %v", err)
	}
	cooldown := time.Now().UTC().Add(time.Minute)
	if err := db.UpsertConectorOperacion(&db.ConectorOperacion{
		ConectorID:         conector.ID,
		EstadoOperativo:    db.ConectorOperativoCircuitoAbierto,
		Motivo:             "remote_connector_unavailable",
		FallosConsecutivos: 3,
		CooldownUntil:      &cooldown,
	}); err != nil {
		t.Fatalf("upsert conector operacion: %v", err)
	}

	n, err := procesarAutonomiaAgentesBatch()
	if err != nil {
		t.Fatalf("procesar autonomia: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 decision autonoma, got=%d", n)
	}

	op, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != db.ProyectoOperativoBloqueadoExterno || !strings.Contains(op.Motivo, "conector:codex-remote:circuito_abierto") {
		t.Fatalf("proyecto no bloqueado por conector: %+v", op)
	}
	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "pause" || !strings.Contains(orders[0].PayloadJSON, "conector:codex-remote:circuito_abierto") {
		t.Fatalf("el daemon deberia pausar y no relanzar mientras el circuito esta abierto: %+v", orders)
	}
}

func TestProcesarRuntimeTranscriptBatchEncolaGuiaAutomatica(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	logPath := filepath.Join(tmp, "codex-transcript.log")
	if err := os.WriteFile(logPath, []byte("¿me dejas seguir con el refactor?\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"log_path":       logPath,
		"driver":         "process_pty_cli",
		"stdin_path":     filepath.Join(tmp, "pty.stdin"),
		"supervisor_ref": filepath.Join(tmp, "supervisor.ref"),
		"rendered_command": filepath.Join(
			tmp, "codex-perfiles", "bin", "codex-perfil",
		) + " Codex1",
		"can_send_input": false,
	})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesar transcript batch: %v", err)
	}
	if n < 2 {
		t.Fatalf("esperaba ingestión + señal procesada, got=%d", n)
	}

	agente := "Codex1"
	transcript, err := db.ListarRuntimeTranscript(db.FiltroRuntimeTranscript{Agente: &agente, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(transcript) == 0 || transcript[0].HandledAt == nil {
		t.Fatalf("se esperaba transcript gestionado: %+v", transcript)
	}
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "send_instruction" {
		t.Fatalf("send_instruction no encolada: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"classification":"approval_request"`) {
		t.Fatalf("payload de guía automática inesperado: %s", orders[0].PayloadJSON)
	}
	if !strings.Contains(orders[0].PayloadJSON, "Aprobado automaticamente") {
		t.Fatalf("la guía automática debería aprobar explícitamente acciones normales: %s", orders[0].PayloadJSON)
	}
	kind := "auto_guidance_sent"
	events, err := db.ListarRuntimeEvents(db.FiltroRuntimeEvents{
		Agente:     &agente,
		ProyectoID: &proyectoID,
		Kind:       &kind,
		Limit:      5,
	})
	if err != nil {
		t.Fatalf("listar runtime events: %v", err)
	}
	if len(events) != 1 || events[0] == nil {
		t.Fatalf("evento auto_guidance_sent inesperado: %+v", events)
	}
	if events[0].Payload == nil {
		t.Fatalf("el evento debería exponer payload parseado: %+v", events[0])
	}
	if got, _ := events[0].Payload["classification"].(string); got != "approval_request" {
		t.Fatalf("clasificación del evento inesperada: %+v", events[0].Payload)
	}
	if got, _ := events[0].Payload["signal_text"].(string); !strings.Contains(got, "refactor") {
		t.Fatalf("signal_text del evento inesperado: %+v", events[0].Payload)
	}
	instruction, _ := events[0].Payload["instruction"].(map[string]any)
	if instruction == nil {
		t.Fatalf("el evento debería incluir instruction útil: %+v", events[0].Payload)
	}
	if got, _ := instruction["to_agente"].(string); got != "Codex1" {
		t.Fatalf("instruction.to_agente inesperado: %+v", instruction)
	}
	if got, _ := instruction["texto"].(string); !strings.Contains(got, "Aprobado automaticamente") {
		t.Fatalf("instruction.texto sin guía útil: %+v", instruction)
	}
	var notif *db.EventoNotificacion
	for {
		select {
		case item := <-db.CanalNotificaciones:
			if item.Tipo == "runtime_auto_guidance" && item.Agente == "Codex1" {
				copy := item
				notif = &copy
			}
		default:
			goto notificationsChecked
		}
	}

notificationsChecked:
	if notif == nil {
		t.Fatalf("faltaba notificación runtime_auto_guidance en el canal")
	}
	if !strings.Contains(notif.Texto, "refactor") {
		t.Fatalf("notificación runtime_auto_guidance sin contexto útil: %+v", notif)
	}
	if notif.Payload == nil {
		t.Fatalf("notificación runtime_auto_guidance sin payload útil: %+v", notif)
	}
	if got, _ := notif.Payload["classification"].(string); got != "approval_request" {
		t.Fatalf("payload de notificación inesperado: %+v", notif.Payload)
	}
}

func TestConstruirRespuestaSignalTranscriptAplicaPoliticaPermisos(t *testing.T) {
	t.Parallel()

	cases := []struct {
		nombre         string
		classification string
		texto          string
		wantContains   []string
	}{
		{
			nombre:         "aprueba_trabajo_normal",
			classification: "approval_request",
			texto:          "me dejas seguir con el refactor y ejecutar go test ./...?",
			wantContains:   []string{"Aprobado automaticamente", "go test", "No necesitas confirmacion humana"},
		},
		{
			nombre:         "rechaza_accion_destructiva",
			classification: "approval_request",
			texto:          "quieres que haga git reset --hard y rm -rf de los cambios?",
			wantContains:   []string{"No autorizado automaticamente", "git reset --hard", "workspace"},
		},
		{
			nombre:         "detecta_dependencia_externa",
			classification: "waiting_human",
			texto:          "quedo a la espera, faltan credenciales oauth y token externo",
			wantContains:   []string{"No inventes credenciales", "otro frente util"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.nombre, func(t *testing.T) {
			t.Parallel()
			resp := construirRespuestaSignalTranscript(&db.RuntimeTranscriptEntry{
				Classification: tc.classification,
				NormalizedText: tc.texto,
				Text:           tc.texto,
			})
			for _, token := range tc.wantContains {
				if !strings.Contains(resp, token) {
					t.Fatalf("respuesta sin %q:\n%s", token, resp)
				}
			}
		})
	}
}

func TestProcesarRuntimeTranscriptBatchDespiertaSupervisorPorSignal(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar worker: %v", err)
	}
	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion worker: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle worker: %+v err=%v", handle, err)
	}
	logPath := filepath.Join(tmp, "codex-transcript-supervisor.log")
	if err := os.WriteFile(logPath, []byte("quedo a la espera de tu respuesta\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"log_path":         logPath,
		"driver":           "process_pty_cli",
		"stdin_path":       filepath.Join(tmp, "pty.stdin"),
		"supervisor_ref":   filepath.Join(tmp, "supervisor.ref"),
		"rendered_command": filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil") + " Codex1",
		"can_send_input":   false,
	})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesar transcript batch: %v", err)
	}
	if n < 2 {
		t.Fatalf("esperaba ingestión + señal procesada, got=%d", n)
	}

	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	var workerGuide *db.RuntimeOrder
	var supervisorNudge *db.RuntimeOrder
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch {
		case order.Agente == "Codex1" && order.Tipo == "send_instruction":
			workerGuide = order
		case order.Agente == "CodexSupervisor" && order.Tipo == "nudge":
			supervisorNudge = order
		}
	}
	if workerGuide == nil || !strings.Contains(workerGuide.PayloadJSON, `"classification":"waiting_human"`) {
		t.Fatalf("guía automática del worker inesperada: %+v", workerGuide)
	}
	if supervisorNudge == nil || !strings.Contains(supervisorNudge.PayloadJSON, `"accion":"inspeccionar_transcript_signal"`) {
		t.Fatalf("nudge al supervisor inesperado: %+v", supervisorNudge)
	}
}

func TestProcesarRuntimeTranscriptBatchDespiertaReviewerPorReadyForReview(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar worker: %v", err)
	}
	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
		t.Fatalf("registrar reviewer: %v", err)
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexReviewer", proyectoID, "review"); err != nil {
		t.Fatalf("activar asignacion reviewer: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexReviewer",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	}); err != nil {
		t.Fatalf("iniciar sesion reviewer: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion worker: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle worker: %+v err=%v", handle, err)
	}
	logPath := filepath.Join(tmp, "codex-transcript-review.log")
	if err := os.WriteFile(logPath, []byte("está listo para revisión final\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"log_path":         logPath,
		"driver":           "process_pty_cli",
		"stdin_path":       filepath.Join(tmp, "pty.stdin"),
		"supervisor_ref":   filepath.Join(tmp, "supervisor.ref"),
		"rendered_command": filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil") + " Codex1",
		"can_send_input":   false,
	})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesar transcript batch: %v", err)
	}
	if n < 2 {
		t.Fatalf("esperaba ingestión + señal procesada, got=%d", n)
	}

	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	var workerGuide *db.RuntimeOrder
	var reviewerNudge *db.RuntimeOrder
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch {
		case order.Agente == "Codex1" && order.Tipo == "send_instruction":
			workerGuide = order
		case order.Agente == "CodexReviewer" && order.Tipo == "nudge":
			reviewerNudge = order
		}
	}
	if workerGuide == nil || !strings.Contains(workerGuide.PayloadJSON, `"classification":"ready_for_review"`) {
		t.Fatalf("guía automática del worker inesperada: %+v", workerGuide)
	}
	if reviewerNudge == nil || !strings.Contains(reviewerNudge.PayloadJSON, `"accion":"inspeccionar_ready_for_review"`) {
		t.Fatalf("nudge al reviewer inesperado: %+v", reviewerNudge)
	}
}

func TestSignalTranscriptDebeIgnorarseAutoGuiaBootstrapPoolLocal(t *testing.T) {
	cases := []string{
		"**Bootstrap de Orquesta completado.**\n**Estado de sesión:** Activa.",
		"**Bootstrap de sesión GemmaSeniorAuto procesado.**\n\n**Estado del Agente:** listo.",
		"Bootstrap aceptado. **GemmaProgramador1** inicializado y operativo en `/tmp/proyecto`.\n\n**Estado de la sesión:**\n* **Rol:** Programador.\n\n**Quedo a la espera de la asignación de una tarea o una propuesta OP para iniciar la ejecución.**",
		"Bootstrap de GemmaApp 3 completado.\n\n**Estado de la sesión:**\n* **Agente:** GemmaApp3\n* **Perfil:** `implementacion`\n* **Estado de tareas:** Sin tareas asignadas actualmente.\n\n**Acciones inmediatas:**\nMe encuentro en estado **IDLE**.\n\nQuedo a la espera de una asignación.",
		"Bootstrap de Orquesta para GemmaApp7 procesado correctamente.\n\nEstado Operativo:\nTareas: No se detectan tareas activas.\nSesión: Confirmada como activa\nAcción: En modo Standby",
	}
	for _, texto := range cases {
		item := &db.RuntimeTranscriptEntry{
			Classification: "waiting_human",
			Text:           texto,
		}
		if !signalTranscriptDebeIgnorarseAutoGuia(item) {
			t.Fatalf("el bootstrap del pool local no deberia generar auto-guia waiting_human: %q", texto)
		}
	}
}

func TestControlPlaneBatchTimeoutEfectivoRespetaTimeoutOllama(t *testing.T) {
	prev := os.Getenv("ORQUESTA_OLLAMA_TIMEOUT_MS")
	if err := os.Setenv("ORQUESTA_OLLAMA_TIMEOUT_MS", "600000"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_OLLAMA_TIMEOUT_MS", prev)
	}()
	got := controlPlaneBatchTimeoutEfectivo()
	if got < 10*time.Minute {
		t.Fatalf("timeout efectivo demasiado corto: %s", got)
	}
}

func TestProcesarRuntimeTranscriptBatchNeedsReplanAbreTareaYDespiertaSupervisor(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar worker: %v", err)
	}
	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		SupervisorAgente:     "CodexSupervisor",
		ReviewRequired:       true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion worker: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle worker: %+v err=%v", handle, err)
	}
	logPath := filepath.Join(tmp, "codex-transcript-needs-replan.log")
	if err := os.WriteFile(logPath, []byte("¿Qué hago ahora? no tengo claro el siguiente paso\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesar transcript batch: %v", err)
	}
	if n < 2 {
		t.Fatalf("esperaba ingestión + señal procesada, got=%d", n)
	}

	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	var workerGuide *db.RuntimeOrder
	var supervisorNudge *db.RuntimeOrder
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch {
		case order.Agente == "Codex1" && order.Tipo == "send_instruction":
			workerGuide = order
		case order.Agente == "CodexSupervisor" && order.Tipo == "nudge":
			supervisorNudge = order
		}
	}
	if workerGuide == nil || !strings.Contains(workerGuide.PayloadJSON, `"classification":"needs_replan"`) {
		t.Fatalf("guía automática del worker inesperada: %+v", workerGuide)
	}
	if supervisorNudge == nil || !strings.Contains(supervisorNudge.PayloadJSON, `"accion":"inspeccionar_transcript_signal"`) {
		t.Fatalf("nudge al supervisor inesperado: %+v", supervisorNudge)
	}
	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	var replanTask *db.Tarea
	for _, tarea := range tareas {
		if tarea == nil || tarea.Titulo != autonomiaReplanTaskTitle {
			continue
		}
		replanTask = tarea
		break
	}
	if replanTask == nil {
		t.Fatalf("debería crear una tarea explícita de replanificación, tareas=%+v", tareas)
	}
	if replanTask.Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea de replan debería arrancarse para el supervisor, got=%s", replanTask.Estado)
	}
	if replanTask.Agente == nil || *replanTask.Agente != "CodexSupervisor" {
		t.Fatalf("la tarea de replan debería quedar en el supervisor, tarea=%+v", replanTask)
	}
	if !strings.Contains(replanTask.Notas, "autonomia:needs_replan") {
		t.Fatalf("la tarea de replan debería quedar marcada como needs_replan, tarea=%+v", replanTask)
	}
}

func TestProcesarRuntimeTranscriptBatchNoGuiaAlWorkerEnRuntimePanic(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar worker: %v", err)
	}
	if err := db.RegistrarAgente("CodexSupervisor", "admin"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:       proyectoID,
		Enabled:          true,
		ObjetivoGeneral:  "Terminar la app",
		SupervisorAgente: "CodexSupervisor",
		EstadoAutonomia:  db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSupervisor",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	}); err != nil {
		t.Fatalf("iniciar sesion supervisor: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion worker: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle worker: %+v err=%v", handle, err)
	}
	logPath := filepath.Join(tmp, "codex-transcript-panic.log")
	if err := os.WriteFile(logPath, []byte("thread 'main' panicked at src/ui.rs:1:1\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"log_path":         logPath,
		"driver":           "process_pty_cli",
		"stdin_path":       filepath.Join(tmp, "pty.stdin"),
		"supervisor_ref":   filepath.Join(tmp, "supervisor.ref"),
		"rendered_command": filepath.Join(tmp, "codex-perfiles", "bin", "codex-perfil") + " Codex1",
		"can_send_input":   false,
	})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesar transcript batch: %v", err)
	}
	if n < 2 {
		t.Fatalf("esperaba ingestión + señal procesada, got=%d", n)
	}

	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	var workerGuide *db.RuntimeOrder
	var supervisorNudge *db.RuntimeOrder
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch {
		case order.Agente == "Codex1" && order.Tipo == "send_instruction":
			workerGuide = order
		case order.Agente == "CodexSupervisor" && order.Tipo == "nudge":
			supervisorNudge = order
		}
	}
	if workerGuide != nil {
		t.Fatalf("no debería guiar al worker tras runtime_panic: %+v", workerGuide)
	}
	if supervisorNudge == nil || !strings.Contains(supervisorNudge.PayloadJSON, `"accion":"inspeccionar_transcript_signal"`) {
		t.Fatalf("nudge al supervisor inesperado: %+v", supervisorNudge)
	}
	agenteInfo, err := db.GetAgente("Codex1")
	if err != nil || agenteInfo == nil {
		t.Fatalf("get agente: agente=%+v err=%v", agenteInfo, err)
	}
	if !strings.Contains(agenteInfo.MotivoPausa, "runtime_panic") {
		t.Fatalf("motivo_pausa sin trazabilidad de runtime_panic: %+v", agenteInfo)
	}
	var estadoCuota string
	var reanimarAt sql.NullTime
	if err := db.DB.QueryRow(`SELECT estado_cuota, reanimar_at FROM agentes WHERE nombre=?`, "Codex1").Scan(&estadoCuota, &reanimarAt); err != nil {
		t.Fatalf("leer estado persistido de agente: %v", err)
	}
	if estadoCuota != "enfriamiento" {
		t.Fatalf("runtime_panic deberia persistir enfriamiento, got=%s agente=%+v", estadoCuota, agenteInfo)
	}
	if !reanimarAt.Valid {
		t.Fatalf("runtime_panic deberia persistir reanimar_at, agente=%+v", agenteInfo)
	}
	handle, err = db.GetRuntimeHandle(handle.ID)
	if err != nil || handle == nil {
		t.Fatalf("reload handle: %+v err=%v", handle, err)
	}
}

func TestProcesarRuntimeTranscriptBatchResuelveReviewGateDesdeReviewer(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
		t.Fatalf("registrar reviewer: %v", err)
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
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewerAgente:       "CodexReviewer",
		ReviewRequired:       true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.ActivarAsignacion("CodexReviewer", proyectoID, "review"); err != nil {
		t.Fatalf("activar asignacion reviewer: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Cerrar review",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "CodexReviewer", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	gateID, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:     &proyectoID,
		TareaID:        &tareaID,
		RequestedBy:    "orquesta",
		ReviewerAgente: "CodexReviewer",
		Estado:         db.ReviewGateEnRevision,
		SeverityMax:    "high",
	})
	if err != nil {
		t.Fatalf("crear review gate: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexReviewer",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion reviewer: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle reviewer: %+v err=%v", handle, err)
	}
	logPath := filepath.Join(tmp, "codex-transcript-review-approved.log")
	if err := os.WriteFile(logPath, []byte("Review aprobada, LGTM\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{"log_path": logPath})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle metadata: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesar transcript batch: %v", err)
	}
	if n < 2 {
		t.Fatalf("esperaba ingestión + resolución de gate, got=%d", n)
	}
	gate, err := db.GetReviewGate(gateID)
	if err != nil {
		t.Fatalf("get review gate: %v", err)
	}
	if gate == nil || gate.Estado != db.ReviewGateAprobado {
		t.Fatalf("el review gate debería quedar aprobado, gate=%+v", gate)
	}
	if !strings.Contains(gate.FindingsJSON, `"classification":"review_approved"`) {
		t.Fatalf("los findings deberían reflejar el transcript que resolvió el gate: %s", gate.FindingsJSON)
	}
	agente := "CodexReviewer"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no debería dejar nudges o instrucciones pendientes al resolver el gate directamente: %+v", orders)
	}
}

func TestProcesarRuntimeTranscriptBatchRegistraEntregaGitActiva(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	repoDir := filepath.Join(tmp, "repo-git-transcript")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdTest(t, repoDir, "init", "-b", "main")
	runGitCmdTest(t, repoDir, "config", "user.email", "test@example.com")
	runGitCmdTest(t, repoDir, "config", "user.name", "Test User")
	if err := os.MkdirAll(filepath.Join(repoDir, "modulo"), 0o755); err != nil {
		t.Fatalf("mkdir modulo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "modulo", "worker.go"), []byte("package modulo\n\nfunc Worker() string { return \"old\" }\n"), 0o644); err != nil {
		t.Fatalf("write worker: %v", err)
	}
	runGitCmdTest(t, repoDir, "add", ".")
	runGitCmdTest(t, repoDir, "commit", "-m", "init")

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repoDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	worktree, err := newCoordinationService().PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: "orquestador",
		Agent:      "Codex1",
		Name:       "wt-codex1-git-transcript",
		Branch:     "orq/orquestador/codex1-git-transcript",
		BaseRef:    "main",
		Reason:     "test_runtime_transcript_git",
	})
	if err != nil {
		t.Fatalf("prepare worktree: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &projectID,
		CWD:         worktree.Path,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle: %+v err=%v", handle, err)
	}
	if err := os.WriteFile(filepath.Join(worktree.Path, "modulo", "worker.go"), []byte("package modulo\n\nfunc Worker() string { return \"new\" }\n"), 0o644); err != nil {
		t.Fatalf("write worker in worktree: %v", err)
	}

	specID, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
		ProyectoID:        &projectID,
		Titulo:            "modulo/worker.go::Worker",
		ArchivoObjetivo:   "modulo/worker.go",
		SimboloObjetivo:   "Worker",
		Descripcion:       "Actualiza Worker por git",
		WriteSet:          []string{"modulo/worker.go"},
		TestsObligatorios: []string{"go test ./... -count=1"},
		FormatoSalida:     "git_worktree+evidencia",
		CreadoPor:         "alberto",
	})
	if err != nil {
		t.Fatalf("crear especificacion: %v", err)
	}
	despacho, err := runtimesService.DispatchMicroprogramacionInstruction(runtimesapp.MicroprogramacionDispatchRequest{
		AgenteDestino:     "Codex1",
		ProyectoID:        &projectID,
		Mensaje:           "Implementa Worker por git",
		EspecificacionID:  specID,
		ArchivoObjetivo:   "modulo/worker.go",
		SimboloObjetivo:   "Worker",
		WriteSet:          []string{"modulo/worker.go"},
		TestsObligatorios: []string{"go test ./... -count=1"},
		FormatoSalida:     "git_worktree+evidencia",
	})
	if err != nil {
		t.Fatalf("dispatch microprogramacion: %v", err)
	}
	if _, err := runtimesService.RegistrarSalidaObservada("Codex1", &projectID, "he terminado la microtarea"); err != nil {
		t.Fatalf("registrar salida observada: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesarRuntimeTranscriptBatch: %v", err)
	}
	if n < 1 {
		t.Fatalf("se esperaba al menos una entrega registrada, got=%d", n)
	}
	order, err := runtimesService.GetRuntimeOrder(despacho.RuntimeOrderID)
	if err != nil || order == nil {
		t.Fatalf("get runtime order: %+v err=%v", order, err)
	}
	if order.Estado != "completada" || !strings.Contains(order.ResultadoJSON, `"receipt_source":"git_worktree"`) {
		t.Fatalf("runtime order sin cierre git: %+v", order)
	}
	merges, err := db.ListarGitMerges(&projectID, "pendiente")
	if err != nil || len(merges) == 0 {
		t.Fatalf("listar merges: %+v err=%v", merges, err)
	}
	if merges[0].SourceBranch != worktree.Branch || merges[0].TargetBranch != "main" {
		t.Fatalf("merge inesperado: %+v", merges[0])
	}
	if handle.ID <= 0 {
		t.Fatalf("handle invalido: %+v", handle)
	}
}

func TestProcesarRuntimeTranscriptBatchRegistraEntregaGitPremiumActiva(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	repoDir := filepath.Join(tmp, "repo-git-transcript-premium")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdTest(t, repoDir, "init", "-b", "main")
	runGitCmdTest(t, repoDir, "config", "user.email", "test@example.com")
	runGitCmdTest(t, repoDir, "config", "user.name", "Test User")
	if err := os.MkdirAll(filepath.Join(repoDir, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "cmd", "controlplane_support.go"), []byte("package cmd\n\nfunc premiumWorker() string { return \"old\" }\n"), 0o644); err != nil {
		t.Fatalf("write worker: %v", err)
	}
	runGitCmdTest(t, repoDir, "add", ".")
	runGitCmdTest(t, repoDir, "commit", "-m", "init")

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repoDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	worktree, err := newCoordinationService().PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: "orquestador",
		Agent:      "Codex1",
		Name:       "wt-codex1-git-transcript-premium",
		Branch:     "orq/orquestador/codex1-git-transcript-premium",
		BaseRef:    "main",
		Reason:     "test_runtime_transcript_git_premium",
	})
	if err != nil {
		t.Fatalf("prepare worktree: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &projectID,
		CWD:         worktree.Path,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := db.GetRuntimeHandleBySesionID(sesion.ID); err != nil {
		t.Fatalf("runtime handle: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree.Path, "cmd", "controlplane_support.go"), []byte("package cmd\n\nfunc premiumWorker() string { return \"new\" }\n"), 0o644); err != nil {
		t.Fatalf("write worker in worktree: %v", err)
	}
	rawPayload := `{"source":"pipeline_local","carril":"premium_worktree","tarea_objetivo_id":530,"write_set":["cmd/controlplane_support.go"],"worktree_id":` + strconv.FormatInt(worktree.ID, 10) + `,"ruta_worktree":"` + worktree.Path + `","branch_worktree":"` + worktree.Branch + `","base_ref_worktree":"main"}`
	orderID, err := runtimesService.CreateRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &projectID,
		Tipo:        "send_instruction",
		Estado:      "ejecutando",
		PayloadJSON: rawPayload,
	})
	if err != nil {
		t.Fatalf("create runtime order premium: %v", err)
	}
	if _, err := runtimesService.RegistrarSalidaObservada("Codex1", &projectID, "he terminado el cambio premium"); err != nil {
		t.Fatalf("registrar salida observada premium: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesarRuntimeTranscriptBatch premium: %v", err)
	}
	if n < 1 {
		t.Fatalf("se esperaba al menos una entrega premium registrada, got=%d", n)
	}
	order, err := runtimesService.GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get runtime order premium: %+v err=%v", order, err)
	}
	if order.Estado != "completada" || !strings.Contains(order.ResultadoJSON, `"receipt_source":"git_worktree"`) {
		t.Fatalf("runtime order premium sin cierre git: %+v", order)
	}
	merges, err := db.ListarGitMerges(&projectID, "pendiente")
	if err != nil || len(merges) == 0 {
		t.Fatalf("listar merges premium: %+v err=%v", merges, err)
	}
	if merges[0].SourceBranch != worktree.Branch || merges[0].TargetBranch != "main" {
		t.Fatalf("merge premium inesperado: %+v", merges[0])
	}
	if !strings.Contains(merges[0].MetadataJSON, `"source":"premium_runtime"`) {
		t.Fatalf("metadata premium inesperada: %s", merges[0].MetadataJSON)
	}
}

func TestResolverPoliticaEntregaRuntimeTranscriptClasificaEscenarios(t *testing.T) {
	projectID := int64(42)
	cases := []struct {
		name         string
		entry        *db.RuntimeTranscriptEntry
		wantScenario runtimeTranscriptEntregaEscenario
		wantMaterial bool
		wantGit      bool
	}{
		{
			name: "codigo_patch",
			entry: &db.RuntimeTranscriptEntry{
				ProyectoID:     &projectID,
				Stream:         "assistant",
				Text:           "*** Begin Patch\n*** Update File: cmd/x.go\n@@\n-fail\n+ok\n*** End Patch\n",
				NormalizedText: "*** begin patch *** update file: cmd/x.go @@ -fail +ok *** end patch",
			},
			wantScenario: runtimeTranscriptEntregaEscenarioCodigoPatch,
			wantMaterial: true,
			wantGit:      true,
		},
		{
			name: "consulta_cli",
			entry: &db.RuntimeTranscriptEntry{
				ProyectoID:     &projectID,
				Stream:         "assistant",
				Text:           "¿Puedo seguir con otro archivo o prefieres que espere?",
				NormalizedText: "¿puedo seguir con otro archivo o prefieres que espere?",
				Classification: "approval_request",
			},
			wantScenario: runtimeTranscriptEntregaEscenarioConsulta,
		},
		{
			name: "plan",
			entry: &db.RuntimeTranscriptEntry{
				ProyectoID:     &projectID,
				Stream:         "assistant",
				Text:           "Primero voy a revisar el modulo y luego ajustar los tests.",
				NormalizedText: "primero voy a revisar el modulo y luego ajustar los tests.",
			},
			wantScenario: runtimeTranscriptEntregaEscenarioPlan,
		},
		{
			name: "finalizacion",
			entry: &db.RuntimeTranscriptEntry{
				ProyectoID:     &projectID,
				Stream:         "assistant",
				Text:           "He terminado la microtarea",
				NormalizedText: "he terminado la microtarea",
			},
			wantScenario: runtimeTranscriptEntregaEscenarioFinalizacion,
			wantGit:      true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolverPoliticaEntregaRuntimeTranscript(tc.entry)
			if got.Escenario != tc.wantScenario {
				t.Fatalf("escenario inesperado: got=%s want=%s", got.Escenario, tc.wantScenario)
			}
			if got.PermiteMaterializar != tc.wantMaterial {
				t.Fatalf("PermiteMaterializar inesperado: got=%t want=%t", got.PermiteMaterializar, tc.wantMaterial)
			}
			if got.PermiteRegistrarGit != tc.wantGit {
				t.Fatalf("PermiteRegistrarGit inesperado: got=%t want=%t", got.PermiteRegistrarGit, tc.wantGit)
			}
		})
	}
}

func TestProcesarEntregasDesdeRuntimeTranscriptIgnoraConsultaDelAgente(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	if err := db.RegistrarAgente("GemmaConsulta", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: tmp,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	specID, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
		ProyectoID:      &projectID,
		Titulo:          "modulo/worker.go::Worker",
		ArchivoObjetivo: "modulo/worker.go",
		SimboloObjetivo: "Worker",
		Descripcion:     "No debe tratar una consulta como entrega",
		WriteSet:        []string{"modulo/worker.go"},
		TestsObligatorios: []string{
			"go test ./modulo -count=1",
		},
		FormatoSalida: "ficheros+evidencia",
		CreadoPor:     "alberto",
	})
	if err != nil {
		t.Fatalf("crear especificacion: %v", err)
	}
	despacho, err := runtimesService.DispatchMicroprogramacionInstruction(runtimesapp.MicroprogramacionDispatchRequest{
		AgenteDestino:    "GemmaConsulta",
		ProyectoID:       &projectID,
		Mensaje:          "Implementa el slice",
		EspecificacionID: specID,
		ArchivoObjetivo:  "modulo/worker.go",
		SimboloObjetivo:  "Worker",
		WriteSet:         []string{"modulo/worker.go"},
		TestsObligatorios: []string{
			"go test ./modulo -count=1",
		},
		FormatoSalida: "ficheros+evidencia",
	})
	if err != nil {
		t.Fatalf("dispatch microprogramacion: %v", err)
	}
	runtimeID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "GemmaConsulta",
		ProyectoID:   &projectID,
		LogicalState: "disponible",
		ProcessState: "running",
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"driver":    "remote_http",
		"transport": "api",
		"connector": "ollama_pool_local",
	})
	if err != nil {
		t.Fatalf("marshal metadata handle: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, CURRENT_TIMESTAMP)`,
		"GemmaConsulta", projectID, runtimeID, "api", "session", "ollama-pool-gemmaconsulta", string(metaJSON)); err != nil {
		t.Fatalf("insert handle: %v", err)
	}
	if _, err := runtimesService.RegistrarSalidaObservada("GemmaConsulta", &projectID, "¿Puedo seguir con otro archivo o prefieres que espere?"); err != nil {
		t.Fatalf("registrar salida observada: %v", err)
	}

	n, err := procesarEntregasDesdeRuntimeTranscript()
	if err != nil {
		t.Fatalf("procesarEntregasDesdeRuntimeTranscript: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia registrar entrega para una consulta, got=%d", n)
	}
	order, err := runtimesService.GetRuntimeOrder(despacho.RuntimeOrderID)
	if err != nil || order == nil {
		t.Fatalf("get runtime order: %+v err=%v", order, err)
	}
	if order.Estado == "cancelada" || order.Estado == "completada" {
		t.Fatalf("la consulta no deberia cerrar ni cancelar la orden original: %+v", order)
	}
	estadoPendiente := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: stringPtr("GemmaConsulta"), ProyectoID: &projectID, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("listar runtime orders pendientes: %v", err)
	}
	for _, item := range orders {
		if item == nil || item.ID == despacho.RuntimeOrderID || item.Tipo != "send_instruction" {
			continue
		}
		if strings.Contains(item.PayloadJSON, "CORRECCION_OBLIGATORIA") {
			t.Fatalf("no deberia abrir correccion por una consulta: %s", item.PayloadJSON)
		}
	}
}

func TestProcesarRuntimeTranscriptBatchRegistraEntregaGitPremiumActivaSinTranscript(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	repoDir := filepath.Join(tmp, "repo-git-transcript-premium-no-transcript")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdTest(t, repoDir, "init", "-b", "main")
	runGitCmdTest(t, repoDir, "config", "user.email", "test@example.com")
	runGitCmdTest(t, repoDir, "config", "user.name", "Test User")
	if err := os.MkdirAll(filepath.Join(repoDir, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "cmd", "controlplane_support.go"), []byte("package cmd\n\nfunc premiumWorkerNoTranscript() string { return \"old\" }\n"), 0o644); err != nil {
		t.Fatalf("write worker: %v", err)
	}
	runGitCmdTest(t, repoDir, "add", ".")
	runGitCmdTest(t, repoDir, "commit", "-m", "init")

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repoDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	worktree, err := newCoordinationService().PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: "orquestador",
		Agent:      "Codex1",
		Name:       "wt-codex1-git-transcript-premium-no-transcript",
		Branch:     "orq/orquestador/codex1-git-transcript-premium-no-transcript",
		BaseRef:    "main",
		Reason:     "test_runtime_transcript_git_premium_no_transcript",
	})
	if err != nil {
		t.Fatalf("prepare worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree.Path, "cmd", "controlplane_support.go"), []byte("package cmd\n\nfunc premiumWorkerNoTranscript() string { return \"new\" }\n"), 0o644); err != nil {
		t.Fatalf("write worker in worktree: %v", err)
	}
	rawPayload := `{"source":"pipeline_local","carril":"premium_worktree","tarea_objetivo_id":541,"write_set":["cmd/controlplane_support.go"],"worktree_id":` + strconv.FormatInt(worktree.ID, 10) + `,"ruta_worktree":"` + worktree.Path + `","branch_worktree":"` + worktree.Branch + `","base_ref_worktree":"main"}`
	orderID, err := runtimesService.CreateRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &projectID,
		Tipo:        "send_instruction",
		Estado:      "pendiente",
		PayloadJSON: rawPayload,
	})
	if err != nil {
		t.Fatalf("create runtime order premium: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesarRuntimeTranscriptBatch premium sin transcript: %v", err)
	}
	if n < 1 {
		t.Fatalf("se esperaba al menos una entrega premium registrada sin transcript, got=%d", n)
	}
	order, err := runtimesService.GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get runtime order premium: %+v err=%v", order, err)
	}
	if order.Estado != "completada" || !strings.Contains(order.ResultadoJSON, `"receipt_source":"git_worktree"`) {
		t.Fatalf("runtime order premium sin cierre git: %+v", order)
	}
	merges, err := db.ListarGitMerges(&projectID, "pendiente")
	if err != nil || len(merges) == 0 {
		t.Fatalf("listar merges premium: %+v err=%v", merges, err)
	}
	if merges[0].SourceBranch != worktree.Branch || merges[0].TargetBranch != "main" {
		t.Fatalf("merge premium inesperado: %+v", merges[0])
	}
}

func TestProcesarRuntimeTranscriptBatchRegistraEntregaGitPremiumActivaSinTranscriptConMailboxKindPipelineLocal(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	repoDir := filepath.Join(tmp, "repo-git-transcript-premium-no-transcript-mailbox-kind")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdTest(t, repoDir, "init", "-b", "main")
	runGitCmdTest(t, repoDir, "config", "user.email", "test@example.com")
	runGitCmdTest(t, repoDir, "config", "user.name", "Test User")
	if err := os.MkdirAll(filepath.Join(repoDir, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "cmd", "controlplane_support.go"), []byte("package cmd\n\nfunc premiumWorkerNoTranscriptMailboxKind() string { return \"old\" }\n"), 0o644); err != nil {
		t.Fatalf("write worker: %v", err)
	}
	runGitCmdTest(t, repoDir, "add", ".")
	runGitCmdTest(t, repoDir, "commit", "-m", "init")

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repoDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	worktree, err := newCoordinationService().PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: "orquestador",
		Agent:      "Codex1",
		Name:       "wt-codex1-git-transcript-premium-no-transcript-mailbox-kind",
		Branch:     "orq/orquestador/codex1-git-transcript-premium-no-transcript-mailbox-kind",
		BaseRef:    "main",
		Reason:     "test_runtime_transcript_git_premium_no_transcript_mailbox_kind",
	})
	if err != nil {
		t.Fatalf("prepare worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree.Path, "cmd", "controlplane_support.go"), []byte("package cmd\n\nfunc premiumWorkerNoTranscriptMailboxKind() string { return \"new\" }\n"), 0o644); err != nil {
		t.Fatalf("write worker in worktree: %v", err)
	}
	rawPayload := `{"kind":"pipeline_local","mailbox_kind":"pipeline_local","carril":"premium_worktree","tarea_objetivo_id":542,"write_set":["cmd/controlplane_support.go"],"worktree_id":` + strconv.FormatInt(worktree.ID, 10) + `,"ruta_worktree":"` + worktree.Path + `","branch_worktree":"` + worktree.Branch + `","base_ref_worktree":"main"}`
	orderID, err := runtimesService.CreateRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &projectID,
		Tipo:        "send_instruction",
		Estado:      "pendiente",
		PayloadJSON: rawPayload,
	})
	if err != nil {
		t.Fatalf("create runtime order premium mailbox kind: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesarRuntimeTranscriptBatch premium mailbox kind: %v", err)
	}
	if n < 1 {
		t.Fatalf("se esperaba al menos una entrega premium registrada sin transcript usando mailbox_kind, got=%d", n)
	}
	order, err := runtimesService.GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get runtime order premium mailbox kind: %+v err=%v", order, err)
	}
	if order.Estado != "completada" || !strings.Contains(order.ResultadoJSON, `"receipt_source":"git_worktree"`) {
		t.Fatalf("runtime order premium mailbox kind sin cierre git: %+v", order)
	}
	merges, err := db.ListarGitMerges(&projectID, "pendiente")
	if err != nil || len(merges) == 0 {
		t.Fatalf("listar merges premium mailbox kind: %+v err=%v", merges, err)
	}
	if merges[0].SourceBranch != worktree.Branch || merges[0].TargetBranch != "main" {
		t.Fatalf("merge premium mailbox kind inesperado: %+v", merges[0])
	}
}

func TestProcesarRuntimeTranscriptBatchRegistraEntregaGitPremiumDesdeBootstrapLease(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	repoDir := filepath.Join(tmp, "repo-git-transcript-premium-bootstrap")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdTest(t, repoDir, "init", "-b", "main")
	runGitCmdTest(t, repoDir, "config", "user.email", "test@example.com")
	runGitCmdTest(t, repoDir, "config", "user.name", "Test User")
	if err := os.MkdirAll(filepath.Join(repoDir, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "cmd", "controlplane_support.go"), []byte("package cmd\n\nfunc premiumWorkerBootstrap() string { return \"old\" }\n"), 0o644); err != nil {
		t.Fatalf("write worker: %v", err)
	}
	runGitCmdTest(t, repoDir, "add", ".")
	runGitCmdTest(t, repoDir, "commit", "-m", "init")

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repoDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	worktree, err := newCoordinationService().PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: "orquestador",
		Agent:      "Codex1",
		Name:       "wt-codex1-git-transcript-premium-bootstrap",
		Branch:     "orq/orquestador/codex1-git-transcript-premium-bootstrap",
		BaseRef:    "main",
		Reason:     "test_runtime_transcript_git_premium_bootstrap",
	})
	if err != nil {
		t.Fatalf("prepare worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree.Path, "cmd", "controlplane_support.go"), []byte("package cmd\n\nfunc premiumWorkerBootstrap() string { return \"new\" }\n"), 0o644); err != nil {
		t.Fatalf("write worker in worktree: %v", err)
	}
	msgID, err := runtimesService.SendRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &projectID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"source":"pipeline_local","carril":"premium_worktree","tarea_objetivo_id":560,"write_set":["cmd/controlplane_support.go"],"worktree_id":` + strconv.FormatInt(worktree.ID, 10) + `,"ruta_worktree":"` + worktree.Path + `","branch_worktree":"` + worktree.Branch + `","base_ref_worktree":"main"}`,
	})
	if err != nil {
		t.Fatalf("create runtime mailbox premium bootstrap: %v", err)
	}
	if err := runtimesService.MarkRuntimeMailboxDelivered(msgID); err != nil {
		t.Fatalf("mark mailbox delivered: %v", err)
	}
	orderID, err := runtimesService.CreateRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex1",
		ProyectoID:    &projectID,
		Tipo:          "start",
		Estado:        "completada",
		ResultadoJSON: fmt.Sprintf(`{"lease_state":"delivered","mailbox_ids":[%d],"sesion_id":12}`, msgID),
	})
	if err != nil {
		t.Fatalf("create runtime order premium bootstrap: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesarRuntimeTranscriptBatch premium bootstrap: %v", err)
	}
	if n < 1 {
		t.Fatalf("se esperaba al menos una entrega premium bootstrap registrada, got=%d", n)
	}
	order, err := runtimesService.GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get runtime order premium bootstrap: %+v err=%v", order, err)
	}
	if order.Estado != "completada" || !strings.Contains(order.ResultadoJSON, `"receipt_source":"git_worktree"`) {
		t.Fatalf("runtime order premium bootstrap sin cierre git: %+v", order)
	}
	msg, err := runtimesService.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox premium bootstrap: %+v err=%v", msg, err)
	}
	if msg.Estado != "consumido" {
		t.Fatalf("mailbox premium bootstrap deberia quedar consumido: %+v", msg)
	}
	merges, err := db.ListarGitMerges(&projectID, "pendiente")
	if err != nil || len(merges) == 0 {
		t.Fatalf("listar merges premium bootstrap: %+v err=%v", merges, err)
	}
	if merges[0].SourceBranch != worktree.Branch || merges[0].TargetBranch != "main" {
		t.Fatalf("merge premium bootstrap inesperado: %+v", merges[0])
	}
}

func TestProcesarRuntimeTranscriptBatchRegistraEntregaGitPremiumDesdeBootstrapLeaseCaseInsensitive(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	repoDir := filepath.Join(tmp, "repo-git-transcript-premium-bootstrap-case")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdTest(t, repoDir, "init", "-b", "main")
	runGitCmdTest(t, repoDir, "config", "user.email", "test@example.com")
	runGitCmdTest(t, repoDir, "config", "user.name", "Test User")
	if err := os.MkdirAll(filepath.Join(repoDir, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "cmd", "controlplane_support.go"), []byte("package cmd\n\nfunc premiumWorkerBootstrapCase() string { return \"old\" }\n"), 0o644); err != nil {
		t.Fatalf("write worker: %v", err)
	}
	runGitCmdTest(t, repoDir, "add", ".")
	runGitCmdTest(t, repoDir, "commit", "-m", "init")

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repoDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	worktree, err := newCoordinationService().PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: "orquestador",
		Agent:      "Codex1",
		Name:       "wt-codex1-git-transcript-premium-bootstrap-case",
		Branch:     "orq/orquestador/codex1-git-transcript-premium-bootstrap-case",
		BaseRef:    "main",
		Reason:     "test_runtime_transcript_git_premium_bootstrap_case",
	})
	if err != nil {
		t.Fatalf("prepare worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree.Path, "cmd", "controlplane_support.go"), []byte("package cmd\n\nfunc premiumWorkerBootstrapCase() string { return \"new\" }\n"), 0o644); err != nil {
		t.Fatalf("write worker in worktree: %v", err)
	}
	msgID, err := runtimesService.SendRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex1",
		ProyectoID:  &projectID,
		Kind:        "pipeline_local",
		PayloadJSON: `{"source":"pipeline_local","carril":"premium_worktree","tarea_objetivo_id":561,"write_set":["cmd/controlplane_support.go"],"worktree_id":` + strconv.FormatInt(worktree.ID, 10) + `,"ruta_worktree":"` + worktree.Path + `","branch_worktree":"` + worktree.Branch + `","base_ref_worktree":"main"}`,
	})
	if err != nil {
		t.Fatalf("create runtime mailbox premium bootstrap case: %v", err)
	}
	if err := runtimesService.MarkRuntimeMailboxDelivered(msgID); err != nil {
		t.Fatalf("mark mailbox delivered: %v", err)
	}
	orderID, err := runtimesService.CreateRuntimeOrder(&db.RuntimeOrder{
		Agente:        "Codex1",
		ProyectoID:    &projectID,
		Tipo:          "start",
		Estado:        "completada",
		ResultadoJSON: fmt.Sprintf(`{"lease_state":"Delivered","mailbox_ids":[%d],"sesion_id":12}`, msgID),
	})
	if err != nil {
		t.Fatalf("create runtime order premium bootstrap case: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesarRuntimeTranscriptBatch premium bootstrap case: %v", err)
	}
	if n < 1 {
		t.Fatalf("se esperaba al menos una entrega premium bootstrap case registrada, got=%d", n)
	}
	order, err := runtimesService.GetRuntimeOrder(orderID)
	if err != nil || order == nil {
		t.Fatalf("get runtime order premium bootstrap case: %+v err=%v", order, err)
	}
	if order.Estado != "completada" || !strings.Contains(order.ResultadoJSON, `"receipt_source":"git_worktree"`) {
		t.Fatalf("runtime order premium bootstrap case sin cierre git: %+v", order)
	}
	msg, err := runtimesService.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox premium bootstrap case: %+v err=%v", msg, err)
	}
	if msg.Estado != "consumido" {
		t.Fatalf("mailbox premium bootstrap case deberia quedar consumido: %+v", msg)
	}
}

func TestProcesarRuntimeTranscriptBatchIgnoraBootstrapPoolLocalEnEntregas(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("GemmaAppTest", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	specID, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
		ProyectoID:        &projectID,
		Titulo:            "microprogramacionapp/extractor_entrega.go::recortarSeccionEvidencia",
		ArchivoObjetivo:   "microprogramacionapp/extractor_entrega.go",
		SimboloObjetivo:   "recortarSeccionEvidencia",
		Descripcion:       "No debe procesar bootstrap como entrega",
		WriteSet:          []string{"microprogramacionapp/extractor_entrega.go", "microprogramacionapp/extractor_entrega_test.go"},
		TestsObligatorios: []string{"go test ./microprogramacionapp -run TestExtraerArchivosEntregaRecortaSeccionEvidenciaConEspacios -count=1"},
		FormatoSalida:     "ficheros+evidencia",
		CreadoPor:         "alberto",
	})
	if err != nil {
		t.Fatalf("crear especificacion: %v", err)
	}
	despacho, err := runtimesService.DispatchMicroprogramacionInstruction(runtimesapp.MicroprogramacionDispatchRequest{
		AgenteDestino:     "GemmaAppTest",
		ProyectoID:        &projectID,
		Mensaje:           "Implementa la correccion pedida",
		EspecificacionID:  specID,
		ArchivoObjetivo:   "microprogramacionapp/extractor_entrega.go",
		SimboloObjetivo:   "recortarSeccionEvidencia",
		WriteSet:          []string{"microprogramacionapp/extractor_entrega.go", "microprogramacionapp/extractor_entrega_test.go"},
		TestsObligatorios: []string{"go test ./microprogramacionapp -run TestExtraerArchivosEntregaRecortaSeccionEvidenciaConEspacios -count=1"},
		FormatoSalida:     "ficheros+evidencia",
	})
	if err != nil {
		t.Fatalf("dispatch microprogramacion: %v", err)
	}
	runtimeID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "GemmaAppTest",
		ProyectoID:   &projectID,
		LogicalState: "disponible",
		ProcessState: "running",
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"driver":    "remote_http",
		"transport": "api",
		"connector": "ollama_pool_local",
	})
	if err != nil {
		t.Fatalf("marshal metadata handle: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, CURRENT_TIMESTAMP)`,
		"GemmaAppTest", projectID, runtimeID, "api", "session", "ollama-pool-gemmaapptest", string(metaJSON)); err != nil {
		t.Fatalf("insert handle: %v", err)
	}
	bootstrap := "Bootstrap de Orquesta para GemmaAppTest procesado correctamente.\n\nEstado Operativo:\nTareas: No se detectan tareas activas.\nSesión: Confirmada como activa\nAcción: En modo Standby"
	if _, err := runtimesService.RegistrarSalidaObservada("GemmaAppTest", &projectID, bootstrap); err != nil {
		t.Fatalf("registrar salida observada: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesarRuntimeTranscriptBatch: %v", err)
	}
	if n != 0 {
		t.Fatalf("bootstrap no deberia registrar entrega, got=%d", n)
	}
	order, err := runtimesService.GetRuntimeOrder(despacho.RuntimeOrderID)
	if err != nil || order == nil {
		t.Fatalf("get runtime order: %+v err=%v", order, err)
	}
	if order.Estado != "pendiente" {
		t.Fatalf("la orden no deberia cambiar por bootstrap, got=%s", order.Estado)
	}
	estadoCancelada := "cancelada"
	otras, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &order.Agente, ProyectoID: &projectID, Estado: &estadoCancelada})
	if err != nil {
		t.Fatalf("listar runtime orders canceladas: %v", err)
	}
	if len(otras) != 0 {
		t.Fatalf("bootstrap no deberia superseder ni cancelar ordenes: %+v", otras)
	}
}

func TestProcesarRuntimeTranscriptBatchIgnoraStdinComoEntrega(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("GemmaAppStdin", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	specID, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
		ProyectoID:        &projectID,
		Titulo:            "microprogramacionapp/extractor_entrega.go::recortarSeccionEvidencia",
		ArchivoObjetivo:   "microprogramacionapp/extractor_entrega.go",
		SimboloObjetivo:   "recortarSeccionEvidencia",
		Descripcion:       "No debe procesar stdin como entrega",
		WriteSet:          []string{"microprogramacionapp/extractor_entrega.go", "microprogramacionapp/extractor_entrega_test.go"},
		TestsObligatorios: []string{"go test ./microprogramacionapp -run TestExtraerArchivosEntregaRecortaSeccionEvidenciaConEspacios -count=1"},
		FormatoSalida:     "ficheros+evidencia",
		CreadoPor:         "alberto",
	})
	if err != nil {
		t.Fatalf("crear especificacion: %v", err)
	}
	despacho, err := runtimesService.DispatchMicroprogramacionInstruction(runtimesapp.MicroprogramacionDispatchRequest{
		AgenteDestino:     "GemmaAppStdin",
		ProyectoID:        &projectID,
		Mensaje:           "Implementa la correccion pedida",
		EspecificacionID:  specID,
		ArchivoObjetivo:   "microprogramacionapp/extractor_entrega.go",
		SimboloObjetivo:   "recortarSeccionEvidencia",
		WriteSet:          []string{"microprogramacionapp/extractor_entrega.go", "microprogramacionapp/extractor_entrega_test.go"},
		TestsObligatorios: []string{"go test ./microprogramacionapp -run TestExtraerArchivosEntregaRecortaSeccionEvidenciaConEspacios -count=1"},
		FormatoSalida:     "ficheros+evidencia",
	})
	if err != nil {
		t.Fatalf("dispatch microprogramacion: %v", err)
	}
	runtimeID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "GemmaAppStdin",
		ProyectoID:   &projectID,
		LogicalState: "disponible",
		ProcessState: "running",
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"driver":    "remote_http",
		"transport": "api",
		"connector": "ollama_pool_local",
	})
	if err != nil {
		t.Fatalf("marshal metadata handle: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, CURRENT_TIMESTAMP)`,
		"GemmaAppStdin", projectID, runtimeID, "api", "session", "ollama-pool-gemmaappstdin", string(metaJSON)); err != nil {
		t.Fatalf("insert handle: %v", err)
	}
	stdinTexto := "PROTOCOLO_MICROPROGRAMACION_INLINE\n// FILE: microprogramacionapp/extractor_entrega.go\npackage roto"
	if _, err := runtimesService.RegistrarSalidaObservada("GemmaAppStdin", &projectID, stdinTexto); err != nil {
		t.Fatalf("registrar salida observada: %v", err)
	}
	items, err := db.ListarRuntimeTranscript(db.FiltroRuntimeTranscript{Agente: stringPtr("GemmaAppStdin"), ProyectoID: &projectID, Limit: 10})
	if err != nil {
		t.Fatalf("listar transcript: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("se esperaba transcript")
	}
	items[0].Stream = "stdin"
	if _, err := db.DB.Exec(`UPDATE runtime_transcript SET stream='stdin' WHERE id=?`, items[0].ID); err != nil {
		t.Fatalf("forzar stream stdin: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesarRuntimeTranscriptBatch: %v", err)
	}
	if n != 0 {
		t.Fatalf("stdin no deberia registrarse como entrega, got=%d", n)
	}
	order, err := runtimesService.GetRuntimeOrder(despacho.RuntimeOrderID)
	if err != nil || order == nil {
		t.Fatalf("get runtime order: %+v err=%v", order, err)
	}
	if order.Estado != "pendiente" {
		t.Fatalf("la orden no deberia cambiar por transcript stdin, got=%s", order.Estado)
	}
}

func TestProcesarRuntimeTranscriptBatchReencolaCorreccionSiEntregaEsInvalida(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("GemmaAppCorreccion", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	projectID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "orquestador", "microprogramacionapp"), 0o755); err != nil {
		t.Fatalf("mkdir proyecto: %v", err)
	}
	specID, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
		ProyectoID:        &projectID,
		Titulo:            "microprogramacionapp/extractor_entrega.go::recortarSeccionEvidencia",
		ArchivoObjetivo:   "microprogramacionapp/extractor_entrega.go",
		SimboloObjetivo:   "recortarSeccionEvidencia",
		Descripcion:       "Debe corregir salida invalida",
		WriteSet:          []string{"microprogramacionapp/extractor_entrega.go", "microprogramacionapp/extractor_entrega_test.go"},
		TestsObligatorios: []string{"go test ./microprogramacionapp -run TestExtraerArchivosEntregaRecortaSeccionEvidenciaConEspacios -count=1"},
		FormatoSalida:     "ficheros+evidencia",
		CreadoPor:         "alberto",
	})
	if err != nil {
		t.Fatalf("crear especificacion: %v", err)
	}
	despacho, err := runtimesService.DispatchMicroprogramacionInstruction(runtimesapp.MicroprogramacionDispatchRequest{
		AgenteDestino:     "GemmaAppCorreccion",
		ProyectoID:        &projectID,
		Mensaje:           "Implementa la correccion pedida",
		EspecificacionID:  specID,
		ArchivoObjetivo:   "microprogramacionapp/extractor_entrega.go",
		SimboloObjetivo:   "recortarSeccionEvidencia",
		WriteSet:          []string{"microprogramacionapp/extractor_entrega.go", "microprogramacionapp/extractor_entrega_test.go"},
		TestsObligatorios: []string{"go test ./microprogramacionapp -run TestExtraerArchivosEntregaRecortaSeccionEvidenciaConEspacios -count=1"},
		FormatoSalida:     "ficheros+evidencia",
	})
	if err != nil {
		t.Fatalf("dispatch microprogramacion: %v", err)
	}
	runtimeID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:       "GemmaAppCorreccion",
		ProyectoID:   &projectID,
		LogicalState: "disponible",
		ProcessState: "running",
	})
	if err != nil {
		t.Fatalf("registrar runtime: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"driver":    "remote_http",
		"transport": "api",
		"connector": "ollama_pool_local",
	})
	if err != nil {
		t.Fatalf("marshal metadata handle: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, CURRENT_TIMESTAMP)`,
		"GemmaAppCorreccion", projectID, runtimeID, "api", "session", "ollama-pool-gemmaappcorreccion", string(metaJSON)); err != nil {
		t.Fatalf("insert handle: %v", err)
	}

	respuestaInvalida := `// FILE: microprogramacionapp/extractor_entrega.go
package microprogramacionapp

func recortarSeccionEvidencia(contenido string) string {
	return
`
	if _, err := runtimesService.RegistrarSalidaObservada("GemmaAppCorreccion", &projectID, respuestaInvalida); err != nil {
		t.Fatalf("registrar salida observada: %v", err)
	}

	n, err := procesarRuntimeTranscriptBatch()
	if err != nil {
		t.Fatalf("procesarRuntimeTranscriptBatch: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia contabilizar la reencolacion de correccion, got=%d", n)
	}
	order, err := runtimesService.GetRuntimeOrder(despacho.RuntimeOrderID)
	if err != nil || order == nil {
		t.Fatalf("get runtime order: %+v err=%v", order, err)
	}
	if order.Estado != "cancelada" {
		t.Fatalf("la orden original deberia quedar cancelada tras la correccion, got=%s", order.Estado)
	}
	estadoPendiente := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: stringPtr("GemmaAppCorreccion"), ProyectoID: &projectID, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("listar runtime orders pendientes: %v", err)
	}
	if len(orders) == 0 {
		t.Fatal("se esperaba una runtime order de correccion pendiente")
	}
	var encontrada bool
	for _, item := range orders {
		if item == nil || item.ID == despacho.RuntimeOrderID || item.Tipo != "send_instruction" {
			continue
		}
		if !strings.Contains(item.PayloadJSON, "CORRECCION_OBLIGATORIA") {
			t.Fatalf("payload de correccion inesperado: %s", item.PayloadJSON)
		}
		encontrada = true
	}
	if !encontrada {
		t.Fatal("no se encontro la runtime order de correccion esperada")
	}
}

type fakeResolvedorPipelineControlPlane struct {
	agente string
}

func (f fakeResolvedorPipelineControlPlane) ResolverAgentePipeline(capacidadapp.EntradaResolverAgentePipeline) (string, error) {
	return strings.TrimSpace(f.agente), nil
}

type fakeDespachadorPipelineControlPlane struct {
	llamadas int
	ultimas  []capacidadapp.SolicitudDespachoPipeline
}

func (f *fakeDespachadorPipelineControlPlane) DespacharPipeline(entrada capacidadapp.SolicitudDespachoPipeline) (*capacidadapp.ResultadoDespachoPipeline, error) {
	f.llamadas++
	f.ultimas = append(f.ultimas, entrada)
	startID := int64(1000 + f.llamadas)
	runtimeOrderID := int64(2000 + f.llamadas)
	return &capacidadapp.ResultadoDespachoPipeline{
		Estado:         "encolado",
		Motivo:         "fake",
		StartOrderID:   &startID,
		RuntimeOrderID: &runtimeOrderID,
	}, nil
}

func TestProcesarPipelineLocalBatchDespachaProyectoActivo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetPipelineLocalDispatchGate()
	capacidadService.SetAgentResolver(fakeResolvedorPipelineControlPlane{agente: "Codex1"})
	dispatcher := &fakeDespachadorPipelineControlPlane{}
	capacidadService.SetPipelineDispatcher(dispatcher)
	t.Cleanup(func() {
		resetPipelineLocalDispatchGate()
		capacidadService.SetAgentResolver(resolvedorAgentePipelineOperativo{rowsProvider: agentesService})
		capacidadService.SetPipelineDispatcher(despachadorPipelineOperativo{})
	})

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
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Implementar pipeline premium",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	procesadas, err := procesarPipelineLocalBatch()
	if err != nil {
		t.Fatalf("procesarPipelineLocalBatch: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("procesadas=%d, want=1", procesadas)
	}
	if dispatcher.llamadas != 1 {
		t.Fatalf("dispatcher.llamadas=%d, want=1", dispatcher.llamadas)
	}
	if len(dispatcher.ultimas) != 1 || dispatcher.ultimas[0].Despacho == nil {
		t.Fatalf("despacho no registrado: %+v", dispatcher.ultimas)
	}
	if dispatcher.ultimas[0].ProyectoSlug != "orquestador" {
		t.Fatalf("proyecto inesperado: %+v", dispatcher.ultimas[0])
	}
	if strings.TrimSpace(dispatcher.ultimas[0].Despacho.AgenteSugerido) != "Codex1" {
		t.Fatalf("agente sugerido inesperado: %+v", dispatcher.ultimas[0].Despacho)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Estado != db.TareaEnProgreso {
		t.Fatalf("estado de tarea inesperado: %+v", tarea)
	}
}

func TestProcesarPipelineLocalBatchRespetaCooldownDeDespacho(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetPipelineLocalDispatchGate()
	capacidadService.SetAgentResolver(fakeResolvedorPipelineControlPlane{agente: "Codex1"})
	dispatcher := &fakeDespachadorPipelineControlPlane{}
	capacidadService.SetPipelineDispatcher(dispatcher)
	t.Cleanup(func() {
		resetPipelineLocalDispatchGate()
		capacidadService.SetAgentResolver(resolvedorAgentePipelineOperativo{rowsProvider: agentesService})
		capacidadService.SetPipelineDispatcher(despachadorPipelineOperativo{})
	})

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
	if _, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Implementar pipeline premium",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "tester",
	}); err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	procesadas, err := procesarPipelineLocalBatch()
	if err != nil {
		t.Fatalf("primer procesarPipelineLocalBatch: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("primer procesadas=%d, want=1", procesadas)
	}
	procesadas, err = procesarPipelineLocalBatch()
	if err != nil {
		t.Fatalf("segundo procesarPipelineLocalBatch: %v", err)
	}
	if procesadas != 0 {
		t.Fatalf("segundo procesadas=%d, want=0", procesadas)
	}
	if dispatcher.llamadas != 1 {
		t.Fatalf("dispatcher.llamadas=%d, want=1", dispatcher.llamadas)
	}
}

func TestProcesarPipelineLocalBatchCierraProyectoEnEstadoCerrandoSinSesion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetPipelineLocalDispatchGate()
	dispatcher := &fakeDespachadorPipelineControlPlane{}
	capacidadService.SetPipelineDispatcher(dispatcher)
	t.Cleanup(func() {
		resetPipelineLocalDispatchGate()
		capacidadService.SetPipelineDispatcher(despachadorPipelineOperativo{})
	})

	repo := prepararRepoGitAutonomia(t, filepath.Join(tmp, "orquestador"))
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repo,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoCerrando,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Ultimo frente",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.CompletarTarea(tareaID, "Codex1", "abc123"); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}
	now := time.Now().UTC()
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:  &proyectoID,
		RequestedBy: "orquesta",
		Estado:      db.ReviewGateAprobado,
		ResolvedAt:  &now,
	}); err != nil {
		t.Fatalf("crear review gate aprobado: %v", err)
	}
	if _, err := db.GuardarGitMerge(&db.GitMerge{
		ProyectoID:   proyectoID,
		SourceBranch: "feature/x",
		TargetBranch: "master",
		RequestedBy:  "orquesta",
		Estado:       "fusionado",
		CommitMerge:  "def456",
		MetadataJSON: `{"auto_created":true,"source":"review_gate_approved"}`,
	}); err != nil {
		t.Fatalf("guardar git merge fusionado: %v", err)
	}

	procesadas, err := procesarPipelineLocalBatch()
	if err != nil {
		t.Fatalf("procesarPipelineLocalBatch: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("procesadas=%d, want=1", procesadas)
	}
	if dispatcher.llamadas != 0 {
		t.Fatalf("dispatcher no debería llamarse al cerrar proyecto, got=%d", dispatcher.llamadas)
	}
	op, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != db.ProyectoOperativoCerrado {
		t.Fatalf("proyecto no cerrado: %+v", op)
	}
	policy, err := supervisionService.GetProjectPolicy("orquestador")
	if err != nil {
		t.Fatalf("get project policy: %v", err)
	}
	if policy == nil || policy.EstadoAutonomia != db.AutonomiaProyectoCerrado {
		t.Fatalf("estado autonomia inesperado: %+v", policy)
	}
}

func TestAsegurarTareaReplanAutonomiaSignalCreaFrenteParaSupervisor(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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
	policy := &db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app sin intervención humana",
		DefinitionOfDoneJSON: `{"tests":"green"}`,
		SupervisorAgente:     "CodexSupervisor",
	}
	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	supervisor := &db.Agente{Nombre: "CodexSupervisor", Rol: "admin"}
	item := &db.RuntimeTranscriptEntry{
		ID:             77,
		ProyectoID:     &proyectoID,
		Agente:         "CodexWorker",
		Classification: "needs_replan",
		Text:           "No tengo claro el siguiente paso útil",
	}

	note, err := asegurarTareaReplanAutonomiaSignal(item, policy, proyecto, supervisor)
	if err != nil {
		t.Fatalf("asegurarTareaReplanAutonomiaSignal: %v", err)
	}
	if !strings.HasPrefix(note, "replan_task_created:") {
		t.Fatalf("nota inesperada: %s", note)
	}
	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) != 1 {
		t.Fatalf("debería crear exactamente una tarea de replan, tareas=%+v", tareas)
	}
	if tareas[0].Titulo != autonomiaReplanTaskTitle {
		t.Fatalf("título de replan inesperado: %+v", tareas[0])
	}
	if tareas[0].Estado != db.TareaEnProgreso {
		t.Fatalf("la tarea de replan debería arrancarse para el supervisor, tarea=%+v", tareas[0])
	}
	if tareas[0].Agente == nil || *tareas[0].Agente != "CodexSupervisor" {
		t.Fatalf("la tarea de replan debería quedar en el supervisor, tarea=%+v", tareas[0])
	}
}

func TestAsegurarSolicitudMergeDesdeGateAprobadoCreaSolicitud(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Cerrar review",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	worktree, err := db.CoordinationWorktreeSQLRepository{}.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		TaskID:    &tareaID,
		Agent:     "CodexReviewer",
		Name:      "orquestador-codexreviewer-t1",
		Path:      filepath.Join(tmp, "wt-review-merge"),
		Branch:    "orq/orquestador/CodexReviewer/t1",
		BaseRef:   "master",
		State:     coordinacion.WorktreeActive,
		Reason:    "review_merge",
	})
	if err != nil {
		t.Fatalf("crear worktree: %v", err)
	}
	proyecto, err := db.GetProyecto(strconv.FormatInt(proyectoID, 10))
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	gate := &reviewapp.Gate{
		ID:             91,
		ProyectoID:     proyectoID,
		ProyectoSlug:   proyecto.Slug,
		TareaID:        &tareaID,
		WorktreeID:     &worktree.ID,
		ReviewerAgente: "CodexReviewer",
		Estado:         reviewapp.GateStateApproved,
	}

	created, err := asegurarSolicitudMergeDesdeGateAprobado(proyecto, gate)
	if err != nil {
		t.Fatalf("asegurarSolicitudMergeDesdeGateAprobado: %v", err)
	}
	if !created {
		t.Fatal("debería crear una solicitud de merge para un gate aprobado con worktree válido")
	}
	merges, err := db.ListarGitMerges(&proyectoID, "")
	if err != nil {
		t.Fatalf("listar merges: %v", err)
	}
	if len(merges) != 1 {
		t.Fatalf("debería crear exactamente una solicitud de merge, merges=%+v", merges)
	}
	if merges[0].SourceBranch != worktree.Branch || merges[0].TargetBranch != "master" || merges[0].Estado != "aprobado" {
		t.Fatalf("solicitud de merge inesperada: %+v", merges[0])
	}
	if !strings.Contains(merges[0].MetadataJSON, `"source":"review_gate_approved"`) {
		t.Fatalf("metadata de merge inesperada: %s", merges[0].MetadataJSON)
	}
}

func TestProcesarGitMergesBatchFusionaCierraWorktreeYCompletaTarea(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	repo := prepararRepoGitAutonomia(t, filepath.Join(tmp, "orquestador"))

	if err := db.RegistrarAgente("CodexReviewer", "admin"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: repo,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertProyectoAutonomia(&db.ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		ReviewRequired:       true,
		AutoCloseProject:     true,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert proyecto autonomia: %v", err)
	}
	if _, err := db.RegistrarFaseProyecto(&db.FaseProyecto{
		Proyecto: "orquestador",
		Nombre:   "integracion",
		Orden:    90,
		Peso:     1,
		Estado:   "activa",
	}); err != nil {
		t.Fatalf("registrar fase integracion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Integrar cambio",
		ProyectoID: &proyectoID,
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "orquesta",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexReviewer"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	now := time.Now().UTC()
	if _, err := db.CrearReviewGate(&db.ReviewGate{
		ProyectoID:  &proyectoID,
		TareaID:     &tareaID,
		RequestedBy: "orquesta",
		Estado:      db.ReviewGateAprobado,
		ResolvedAt:  &now,
	}); err != nil {
		t.Fatalf("crear review gate aprobado: %v", err)
	}
	worktree, err := newCoordinationService().PrepareWorktree(coordinacion.PrepareWorktreeInput{
		ProjectRef: "orquestador",
		Agent:      "CodexReviewer",
		TaskID:     &tareaID,
		Reason:     "review_merge",
	})
	if err != nil {
		t.Fatalf("prepare worktree: %v", err)
	}
	cmdGitAutonomia(t, worktree.Path, "config", "user.name", "Orquesta Test")
	cmdGitAutonomia(t, worktree.Path, "config", "user.email", "orquesta@example.test")
	if err := os.WriteFile(filepath.Join(worktree.Path, "feature.txt"), []byte("hola\n"), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	cmdGitAutonomia(t, worktree.Path, "add", "feature.txt")
	cmdGitAutonomia(t, worktree.Path, "commit", "-m", "feat: add feature")
	cmdGitAutonomia(t, repo, "checkout", "--detach")

	metadataJSON, err := json.Marshal(map[string]any{
		"auto_created": true,
		"source":       "review_gate_approved",
		"review_gate":  91,
		"worktree_id":  worktree.ID,
		"tarea_id":     tareaID,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if _, err := db.GuardarGitMerge(&db.GitMerge{
		ProyectoID:   proyectoID,
		SourceBranch: worktree.Branch,
		TargetBranch: "master",
		RequestedBy:  "CodexReviewer",
		Estado:       "aprobado",
		MetadataJSON: string(metadataJSON),
	}); err != nil {
		t.Fatalf("guardar git merge: %v", err)
	}

	n, err := procesarGitMergesBatch()
	if err != nil {
		t.Fatalf("procesar git merges: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 merge procesado, got=%d", n)
	}
	merges, err := db.ListarGitMerges(&proyectoID, "")
	if err != nil {
		t.Fatalf("listar merges: %v", err)
	}
	if len(merges) != 1 || merges[0].Estado != "fusionado" {
		if len(merges) == 0 {
			t.Fatalf("merge no fusionado correctamente: lista vacía")
		}
		t.Fatalf("merge no fusionado correctamente: %+v", *merges[0])
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != db.TareaCompletada || strings.TrimSpace(tarea.CommitCierre) != strings.TrimSpace(merges[0].CommitMerge) {
		t.Fatalf("tarea no completada con commit de merge: %+v merge=%+v", tarea, merges[0])
	}
	wt, err := db.CoordinationWorktreeRepository().GetByID(worktree.ID)
	if err != nil {
		t.Fatalf("get worktree: %v", err)
	}
	if wt.State != coordinacion.WorktreeClosed {
		t.Fatalf("worktree debería quedar cerrada: %+v", wt)
	}
	cmdGitAutonomia(t, repo, "checkout", "master")
	if _, err := os.Stat(worktree.Path); !os.IsNotExist(err) {
		t.Fatalf("la ruta de worktree debería haberse eliminado: err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "feature.txt")); err != nil {
		t.Fatalf("feature.txt debería quedar fusionado en repo base: %v", err)
	}
	fases, err := db.ListarFasesProyecto("orquestador")
	if err != nil {
		t.Fatalf("listar fases: %v", err)
	}
	var integracion *db.FaseProyecto
	for _, fase := range fases {
		if fase != nil && strings.EqualFold(strings.TrimSpace(fase.Nombre), "integracion") {
			integracion = fase
			break
		}
	}
	if integracion == nil {
		t.Fatal("debería existir la fase integracion")
	}
	if !strings.EqualFold(strings.TrimSpace(integracion.Estado), "completada") {
		t.Fatalf("integracion debería quedar completada: %+v", integracion)
	}
	policy, err := supervisionService.GetProjectPolicy("orquestador")
	if err != nil {
		t.Fatalf("get project policy: %v", err)
	}
	if policy == nil {
		t.Fatal("debería existir policy de autonomia")
	}
	if policy.EstadoAutonomia != db.AutonomiaProyectoCerrando {
		t.Fatalf("estado autonomia inesperado: got=%q want=%q", policy.EstadoAutonomia, db.AutonomiaProyectoCerrando)
	}
}

func TestProcesarRuntimeMailboxBatchMaterializaAutonomiaMailbox(t *testing.T) {
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
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	metaJSON := `{"driver":"process_pty_cli","stdin_path":"` + filepath.Join(tmp, "pty.stdin") + `","supervisor_ref":"` + filepath.Join(tmp, "supervisor.ref") + `","rendered_command":"codex-perfil Codex1","working_dir":"` + filepath.Join(tmp, "orquestador") + `","external_session_id":"sess-batch-main","can_send_input":false}`
	capsJSON := `{"can_send_input":false,"mailbox_delivery_mode":"` + runtimeagente.MailboxDeliveryBootstrapOnly + `"}`
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET capabilities_json=?, metadata_json=? WHERE id=?`, capsJSON, metaJSON, handle.ID); err != nil {
		t.Fatalf("update handle supervisor local: %v", err)
	}
	if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "orquesta",
		ToAgente:    "Codex1",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","motivo":"seguir el frente activo"}`,
	}); err != nil {
		t.Fatalf("crear mailbox: %v", err)
	}

	n, err := procesarRuntimeMailboxBatch()
	if err != nil {
		t.Fatalf("procesar mailbox batch: %v", err)
	}
	if n != 0 {
		t.Fatalf("el batch general ya no deberia materializar runtime_mailbox por supervisor_local legacy, got=%d", n)
	}
	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("el batch general no deberia crear send_instruction legacy: %+v", orders)
	}
	mailboxEstado := "pendiente"
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &agente, ProyectoID: &proyectoID, Estado: &mailboxEstado})
	if err != nil {
		t.Fatalf("listar mailbox: %v", err)
	}
	if len(mailbox) != 1 {
		t.Fatalf("la mailbox durable debe seguir pendiente tras materialización: %+v", mailbox)
	}
}

func TestProcesarAutonomiaSesionActivaPersisteEnfriamientoPorCuota(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex5", "programador"); err != nil {
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
	if err := db.ActivarAsignacion("Codex5", proyectoID, "frente agotado"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex5",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Frente detenido por cuota",
		Descripcion: "Debe pausar y persistir enfriamiento",
		ProyectoID:  &proyectoID,
		Modulo:      "web",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex5"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	started := time.Now().UTC().Add(-10 * time.Minute)
	reset := time.Now().UTC().Add(90 * time.Minute)
	credits := 0.0
	if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
		SesionID:         sesion.ID,
		WindowKind:       "5h",
		WindowStartedAt:  &started,
		ResetAt:          &reset,
		RemainingCredits: &credits,
		BudgetSource:     "codex_token_count_observed",
		CheckedAt:        time.Now().UTC(),
	}); err != nil {
		t.Fatalf("registrar presupuesto: %v", err)
	}

	n, err := procesarAutonomiaSesionActiva(sesion)
	if err != nil {
		t.Fatalf("procesar autonomia sesion: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 accion autonoma, got=%d", n)
	}
	agente, err := db.GetAgente("Codex5")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if strings.TrimSpace(agente.EstadoCuota) != "enfriamiento" {
		t.Fatalf("deberia quedar en enfriamiento: %+v", agente)
	}
	if agente.ReanimarAt == nil || agente.ReanimarAt.Before(reset.Add(-time.Minute)) || agente.ReanimarAt.After(reset.Add(time.Minute)) {
		t.Fatalf("reanimar_at deberia alinearse con reset del presupuesto: %+v reset=%s", agente, reset)
	}
	agenteNombre := "Codex5"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agenteNombre, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "pause" {
		t.Fatalf("deberia encolar una pause: %+v", orders)
	}
}

func TestProcesarPresupuestoSesionObservadoBatchToleraHandleRoto(t *testing.T) {
	resetRuntimeBudgetObservationBackgroundGate()
	t.Cleanup(resetRuntimeBudgetObservationBackgroundGate)

	tmp := prepararDBTemporalCmd(t)
	base := filepath.Join(tmp, "codex-perfiles")
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	sessionsDir := filepath.Join(base, "homes", "CodexBueno", "sessions", "2026", "03", "31")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions bueno: %v", err)
	}
	authPath := filepath.Join(base, "homes", "CodexBueno", "auth.json")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
		t.Fatalf("mkdir auth bueno: %v", err)
	}
	if err := os.WriteFile(authPath, []byte(`{"profile":{"email":"bueno@example.com","name":"Cuenta Buena"}}`), 0o600); err != nil {
		t.Fatalf("write auth bueno: %v", err)
	}
	resetPrimary := time.Date(2026, 3, 31, 14, 0, 0, 0, time.UTC)
	resetSecondary := time.Date(2026, 4, 6, 7, 26, 0, 0, time.UTC)
	sessionPath := filepath.Join(sessionsDir, "rollout-test.jsonl")
	lines := []string{
		`{"timestamp":"2026-03-31T10:00:00Z","type":"session_meta","payload":{"id":"sess-buena","timestamp":"2026-03-31T10:00:00Z","cwd":"` + filepath.Join(tmp, "orquestador") + `"}}`,
		`{"timestamp":"2026-03-31T10:05:00Z","type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":12,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":43,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetSecondary.Unix(), 10) + `},"credits":null,"plan_type":"plus"}}}`,
	}
	if err := os.WriteFile(sessionPath, []byte(lines[0]+"\n"+lines[1]+"\n"), 0o600); err != nil {
		t.Fatalf("write session bueno: %v", err)
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
	for _, nombre := range []string{"CodexBueno", "CodexRoto"} {
		if err := db.RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("registrar %s: %v", nombre, err)
		}
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      nombre,
			ProyectoID:  &proyectoID,
			CWD:         filepath.Join(tmp, "orquestador"),
			Herramienta: "codex-cli",
		})
		if err != nil {
			t.Fatalf("iniciar sesion %s: %v", nombre, err)
		}
		if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
			t.Fatalf("upsert handle %s: %v", nombre, err)
		}
	}
	bueno, err := db.GetRuntimeHandleActivoAgenteProyecto("CodexBueno", &proyectoID)
	if err != nil || bueno == nil {
		t.Fatalf("get handle bueno: %v %+v", err, bueno)
	}
	rendidoBueno := filepath.Join(base, "bin", "codex-perfil") + " CodexBueno"
	metaBueno := map[string]any{
		"rendered_command":    rendidoBueno,
		"working_dir":         filepath.Join(tmp, "orquestador"),
		"external_session_id": "sess-buena",
		"started_at":          time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}
	metaBuenoJSON, _ := json.Marshal(metaBueno)
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaBuenoJSON), bueno.ID); err != nil {
		t.Fatalf("update handle bueno: %v", err)
	}

	roto, err := db.GetRuntimeHandleActivoAgenteProyecto("CodexRoto", &proyectoID)
	if err != nil || roto == nil {
		t.Fatalf("get handle roto: %v %+v", err, roto)
	}
	badRoot := filepath.Join(tmp, "codex-roto")
	badSessionsDir := filepath.Join(badRoot, "homes", "CodexRoto", "sessions", "2026", "03", "31")
	if err := os.MkdirAll(filepath.Join(badRoot, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin roto: %v", err)
	}
	if err := os.MkdirAll(badSessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions roto: %v", err)
	}
	if err := os.Chmod(badSessionsDir, 0o000); err != nil {
		t.Fatalf("chmod sessions roto: %v", err)
	}
	defer func() { _ = os.Chmod(badSessionsDir, 0o755) }()
	rendidoRoto := filepath.Join(badRoot, "bin", "codex-perfil") + " CodexRoto"
	metaRoto := map[string]any{
		"rendered_command":    rendidoRoto,
		"working_dir":         filepath.Join(tmp, "orquestador"),
		"external_session_id": "sess-rota",
		"started_at":          time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}
	metaRotoJSON, _ := json.Marshal(metaRoto)
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaRotoJSON), roto.ID); err != nil {
		t.Fatalf("update handle roto: %v", err)
	}

	procesados, err := procesarPresupuestoSesionObservadoBatch()
	if err != nil {
		t.Fatalf("procesar presupuesto observado: %v", err)
	}
	if procesados != 1 {
		t.Fatalf("deberia procesar solo el handle sano: %d", procesados)
	}
	agenteBueno, err := db.GetAgente("CodexBueno")
	if err != nil {
		t.Fatalf("get agente bueno: %v", err)
	}
	if agenteBueno.PresupuestoCheckedAt == nil || agenteBueno.CuentaEmail != "bueno@example.com" {
		t.Fatalf("deberia persistir telemetria del handle sano: %+v", agenteBueno)
	}
}

func TestProcesarPresupuestoSesionObservadoBatchRespetaIntervaloBackground(t *testing.T) {
	resetRuntimeBudgetObservationBackgroundGate()
	t.Cleanup(resetRuntimeBudgetObservationBackgroundGate)

	tmp := prepararDBTemporalCmd(t)
	base := filepath.Join(tmp, "codex-perfiles")
	if err := os.MkdirAll(filepath.Join(base, "bin"), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	sessionsDir := filepath.Join(base, "homes", "Codex1", "sessions", "2026", "03", "31")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	authPath := filepath.Join(base, "homes", "Codex1", "auth.json")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
		t.Fatalf("mkdir auth dir: %v", err)
	}
	if err := os.WriteFile(authPath, []byte(`{"profile":{"email":"codex1@example.com","name":"Cuenta Codex1"}}`), 0o600); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	sessionPath := filepath.Join(sessionsDir, "rollout-test.jsonl")
	if err := os.WriteFile(sessionPath, []byte(
		`{"timestamp":"2026-03-31T10:00:00Z","type":"session_meta","payload":{"id":"sess-123","timestamp":"2026-03-31T10:00:00Z","cwd":"`+filepath.Join(tmp, "orquestador")+`"}}`+"\n"+
			`{"timestamp":"2026-03-31T10:05:00Z","type":"event_msg","payload":{"type":"token_count","rate_limits":{"primary":{"used_percent":12,"window_minutes":300,"resets_at":1774958400},"secondary":{"used_percent":43,"window_minutes":10080,"resets_at":1775300000}}}}`+"\n",
	), 0o600); err != nil {
		t.Fatalf("write session: %v", err)
	}
	if err := db.ConfigSet("runtime_budget_background_observation_interval_seconds", "120"); err != nil {
		t.Fatalf("config observation interval: %v", err)
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
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert handle: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("get handle by sesion: %+v err=%v", handle, err)
	}
	rendered := filepath.Join(base, "bin", "codex-perfil") + " Codex1"
	runDir := filepath.Join(tmp, "runtime-worker")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir worker: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	workerNow := time.Date(2026, 3, 31, 10, 6, 0, 0, time.UTC)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex1","driver":"tmux_cli_session","transport":"tmux","profile":"Codex1","profile_status_wrapper":"`+filepath.Join(base, "bin", "codex-perfil")+`","tmux_session":"orq-codex1-budget","tmux_pane_id":"%1","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+workerNow.Format(time.RFC3339Nano)+`","alive":true,"child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+workerNow.Format(time.RFC3339Nano)+`","started_at":"`+workerNow.Format(time.RFC3339Nano)+`","child_pid":`+strconv.Itoa(os.Getpid())+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	meta := map[string]any{
		"driver":                "tmux_cli_session",
		"rendered_command":      rendered,
		"working_dir":           filepath.Join(tmp, "orquestador"),
		"external_session_id":   "sess-123",
		"started_at":            time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	}
	metaJSON, _ := json.Marshal(meta)
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	handle, err = db.GetRuntimeHandleActivoAgenteProyecto("Codex1", &proyectoID)
	if err != nil || handle == nil {
		t.Fatalf("get handle activo: %+v err=%v", handle, err)
	}

	first, err := procesarPresupuestoSesionObservadoBatch()
	if err != nil {
		t.Fatalf("primer batch: %v", err)
	}
	if first != 1 {
		t.Fatalf("el primer batch deberia observar una vez, got=%d", first)
	}
	second, err := procesarPresupuestoSesionObservadoBatch()
	if err != nil {
		t.Fatalf("segundo batch: %v", err)
	}
	if second != 0 {
		t.Fatalf("el segundo batch deberia quedar throttled, got=%d", second)
	}
}

func TestRefrescarPresupuestoSesionObservadoAgenteUsaUltimaSesionClaudeSinHandle(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()
	statusCacheState.mu.Lock()
	statusCacheState.ok = true
	statusCacheState.value = apiStatusResponse{Generado: "stale"}
	statusCacheState.expires = time.Now().UTC().Add(time.Minute)
	statusCacheState.mu.Unlock()
	configHome := filepath.Join(tmp, "claude-home")
	if err := os.MkdirAll(configHome, 0o755); err != nil {
		t.Fatalf("mkdir config home: %v", err)
	}
	claims := base64.RawURLEncoding.EncodeToString([]byte(`{"email":"claude1@example.com","name":"Claude Uno"}`))
	if err := os.WriteFile(filepath.Join(configHome, "credentials.json"), []byte(`{"oauth":{"access_token":"header.`+claims+`.sig","expires_at":"2026-04-01T12:00:00Z"}}`), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}
	prevConfigHome := os.Getenv("CLAUDE_CONFIG_HOME")
	if err := os.Setenv("CLAUDE_CONFIG_HOME", configHome); err != nil {
		t.Fatalf("setenv CLAUDE_CONFIG_HOME: %v", err)
	}
	defer func() {
		if prevConfigHome == "" {
			_ = os.Unsetenv("CLAUDE_CONFIG_HOME")
		} else {
			_ = os.Setenv("CLAUDE_CONFIG_HOME", prevConfigHome)
		}
	}()
	workingDir := filepath.Join(tmp, "repo")
	sessionsDir := filepath.Join(workingDir, ".claude", "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionsDir, "session-1.json"), []byte(`{"version":1,"messages":[{"role":"assistant","blocks":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1200,"output_tokens":300,"cache_creation_input_tokens":50,"cache_read_input_tokens":20}}]}`), 0o600); err != nil {
		t.Fatalf("write managed session: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: workingDir,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("claude1", "programador"); err != nil {
		t.Fatalf("registrar claude1: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "claude1",
		ProyectoID:  &proyectoID,
		CWD:         workingDir,
		Herramienta: "claude-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion claude1: %v", err)
	}

	procesados, err := refrescarPresupuestoSesionObservadoAgente("claude1")
	if err != nil {
		t.Fatalf("refresh presupuesto observado claude1: %v", err)
	}
	if procesados != 1 {
		t.Fatalf("deberia refrescar por sesion sin handle: %d", procesados)
	}
	agente, err := db.GetAgente("claude1")
	if err != nil {
		t.Fatalf("get agente claude1: %v", err)
	}
	if agente.CuentaEmail != "claude1@example.com" || agente.CuentaUsuario != "Claude Uno" {
		t.Fatalf("identidad observada inesperada: %+v", agente)
	}
	if agente.PresupuestoCheckedAt == nil || strings.TrimSpace(agente.PresupuestoFuente) != "claude_rust_session_observed" {
		t.Fatalf("deberia persistir telemetria claude observada por sesion: %+v", agente)
	}
	statusCacheState.mu.Lock()
	cacheValida := statusCacheState.ok
	statusCacheState.mu.Unlock()
	if cacheValida {
		t.Fatalf("el refresh de presupuesto deberia invalidar la snapshot de status")
	}
}

func TestEncolarControlAgenteLocalInvalidaSnapshotStatus(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()
	statusCacheState.mu.Lock()
	statusCacheState.ok = true
	statusCacheState.value = apiStatusResponse{Generado: "stale"}
	statusCacheState.expires = time.Now().UTC().Add(time.Minute)
	statusCacheState.mu.Unlock()

	if err := db.RegistrarAgente("CodexStart", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-status-cache",
		Nombre:  "Orquestador Status Cache",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	now := time.Now().UTC()
	if err := db.UpsertAgenteIdentidadObservadaCanonica("CodexStart", "acc-codex-start", "codexstart@example.com", "Codex Start", "test", &now); err != nil {
		t.Fatalf("upsert identidad observada: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='activo', reanimar_at=NULL, motivo_pausa='' WHERE nombre=?`, "CodexStart"); err != nil {
		t.Fatalf("activar cuota: %v", err)
	}

	if _, _, err := encolarControlAgenteLocal(apiAgenteControlRequest{
		Agente:   "CodexStart",
		Proyecto: strconv.FormatInt(proyectoID, 10),
		Accion:   "arrancar",
		Por:      "test",
		Motivo:   "status-cache",
	}); err != nil {
		t.Fatalf("encolar control agente: %v", err)
	}

	statusCacheState.mu.Lock()
	cacheValida := statusCacheState.ok
	statusCacheState.mu.Unlock()
	if cacheValida {
		t.Fatalf("encolar control agente deberia invalidar la snapshot de status")
	}
}

func prepararRepoGitAutonomia(t *testing.T, repo string) string {
	t.Helper()
	cmdGitAutonomia(t, "", "init", "-b", "master", repo)
	cmdGitAutonomia(t, repo, "config", "user.name", "Orquesta Test")
	cmdGitAutonomia(t, repo, "config", "user.email", "orquesta@example.test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	cmdGitAutonomia(t, repo, "add", "README.md")
	cmdGitAutonomia(t, repo, "commit", "-m", "base")
	return repo
}

func cmdGitAutonomia(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestProcesarAgentesDegradadosAutonomiaBatchReasignaATrabajadorSano(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	autonomiaDegradedTaskGate.Reset()
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
	for _, agente := range []string{"CodexBloqueado", "CodexSano"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
		if err := db.ActivarAsignacion(agente, proyectoID, "frente activo"); err != nil {
			t.Fatalf("activar asignacion %s: %v", agente, err)
		}
	}
	sesionBloqueada, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexBloqueado",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion bloqueada: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesionBloqueada); err != nil {
		t.Fatalf("upsert handle bloqueado: %v", err)
	}
	handleBloqueado, err := db.GetRuntimeHandleActivoAgenteProyecto("CodexBloqueado", &proyectoID)
	if err != nil || handleBloqueado == nil {
		t.Fatalf("get handle bloqueado: %+v err=%v", handleBloqueado, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='fallido', last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, handleBloqueado.ID); err != nil {
		t.Fatalf("marcar handle fallido: %v", err)
	}
	sesionSana, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSano",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion sana: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesionSana); err != nil {
		t.Fatalf("upsert handle sano: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Recuperar worker degradado",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexBloqueado"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexBloqueado"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	autonomiaDegradedTaskGate.Set(strconv.FormatInt(tareaID, 10), time.Now().UTC().Add(-autonomiaDegradedTaskCooldown-time.Minute))

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia procesar una tarea, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "CodexSano" || tarea.Estado != db.EstadoEnProgreso {
		t.Fatalf("tarea no reasignada correctamente: %+v", tarea)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchBloqueaSinRelevoSano(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	autonomiaDegradedTaskGate.Reset()
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-bloqueado",
		Nombre:  "Orquestador Bloqueado",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("CodexBloqueado", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ActivarAsignacion("CodexBloqueado", proyectoID, "frente unico"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexBloqueado",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert handle: %v", err)
	}
	handle, err := db.GetRuntimeHandleActivoAgenteProyecto("CodexBloqueado", &proyectoID)
	if err != nil || handle == nil {
		t.Fatalf("get handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='fallido', last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("marcar handle fallido: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Bloquear por falta de relevo",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexBloqueado"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexBloqueado"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia procesar una tarea, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != db.EstadoBloqueada {
		t.Fatalf("la tarea deberia quedar bloqueada, got=%s", tarea.Estado)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchBloqueaSinRelevoSiAgenteBloqueadoPorCuota(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	autonomiaDegradedTaskGate.Reset()

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-cuota-bloqueada",
		Nombre:  "Orquestador Cuota Bloqueada",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("CodexBloqueado", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ActivarAsignacion("CodexBloqueado", proyectoID, "frente sin cuota"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='cuota visible agotada', reanimar_at=datetime('now','+2 hours') WHERE nombre='CodexBloqueado'`); err != nil {
		t.Fatalf("marcar cuota agotada: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Bloquear por cuota sin relevo",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexBloqueado"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexBloqueado"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia bloquear la tarea sin relevo cuando el agente no tiene cuota, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != db.EstadoBloqueada {
		t.Fatalf("la tarea deberia quedar bloqueada, got=%s", tarea.Estado)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchReasignaSiAgenteBloqueadoPorCuotaTieneRelevoSano(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	autonomiaDegradedTaskGate.Reset()

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-cuota-con-relevo",
		Nombre:  "Orquestador Cuota con Relevo",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for _, agente := range []string{"CodexBloqueado", "CodexLibre"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
		if err := db.ActivarAsignacion(agente, proyectoID, "frente bounded"); err != nil {
			t.Fatalf("activar asignacion %s: %v", agente, err)
		}
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='cuota visible agotada', reanimar_at=datetime('now','+2 hours') WHERE nombre='CodexBloqueado'`); err != nil {
		t.Fatalf("marcar cuota agotada: %v", err)
	}
	sesionLibre, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexLibre",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion relevo: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesionLibre); err != nil {
		t.Fatalf("upsert handle relevo: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Reasignar por cuota con relevo",
		Descripcion: "Frente premium acotado.\nWRITE_SET: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestProcesarAgentesDegradadosAutonomiaBatchReasignaSiAgenteBloqueadoPorCuotaTieneRelevoSano$'",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexBloqueado"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexBloqueado"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia reasignar la tarea bloqueada por cuota al relevo sano, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != db.EstadoEnProgreso || tarea.Agente == nil || *tarea.Agente != "CodexLibre" {
		t.Fatalf("la tarea deberia quedar reasignada al relevo sano: %+v", tarea)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchNoSaturaUnicoRelevoSano(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	autonomiaDegradedTaskGate.Reset()
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-capacidad",
		Nombre:  "Orquestador Capacidad",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for _, agente := range []string{"CodexBloqueado", "CodexSano"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
		if err := db.ActivarAsignacion(agente, proyectoID, "frente capacidad"); err != nil {
			t.Fatalf("activar asignacion %s: %v", agente, err)
		}
	}
	sesionBloqueada, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexBloqueado",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion bloqueada: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesionBloqueada); err != nil {
		t.Fatalf("upsert handle bloqueado: %v", err)
	}
	handleBloqueado, err := db.GetRuntimeHandleActivoAgenteProyecto("CodexBloqueado", &proyectoID)
	if err != nil || handleBloqueado == nil {
		t.Fatalf("get handle bloqueado: %+v err=%v", handleBloqueado, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='fallido', last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, handleBloqueado.ID); err != nil {
		t.Fatalf("marcar handle fallido: %v", err)
	}
	sesionSana, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSano",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion sana: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesionSana); err != nil {
		t.Fatalf("upsert handle sano: %v", err)
	}
	for i := 0; i < autonomiaWorkerOpenTasksCeiling; i++ {
		tareaSanaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      fmt.Sprintf("Trabajo ya activo %d", i+1),
			Descripcion: "test",
			ProyectoID:  &proyectoID,
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea sana %d: %v", i+1, err)
		}
		if err := db.TomarTarea(tareaSanaID, "CodexSano"); err != nil {
			t.Fatalf("tomar tarea sana %d: %v", i+1, err)
		}
		if err := db.IniciarTarea(tareaSanaID, "CodexSano"); err != nil {
			t.Fatalf("iniciar tarea sana %d: %v", i+1, err)
		}
	}
	tareaBloqueadaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "No saturar relevo",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea bloqueada: %v", err)
	}
	if err := db.TomarTarea(tareaBloqueadaID, "CodexBloqueado"); err != nil {
		t.Fatalf("tomar tarea bloqueada: %v", err)
	}
	if err := db.IniciarTarea(tareaBloqueadaID, "CodexBloqueado"); err != nil {
		t.Fatalf("iniciar tarea bloqueada: %v", err)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia procesar una tarea, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaBloqueadaID)
	if err != nil {
		t.Fatalf("get tarea bloqueada: %v", err)
	}
	if tarea.Estado != db.EstadoBloqueada {
		t.Fatalf("la tarea deberia quedar bloqueada para no saturar al relevo, got=%s", tarea.Estado)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchPriorizaAgenteConMasTareasBloqueadas(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	autonomiaDegradedTaskGate.Reset()

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-prioridad-bloqueadas",
		Nombre:  "Orquestador Prioridad Bloqueadas",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for _, agente := range []string{"CodexBloqueadoA", "CodexBloqueadoB", "CodexSano"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
		if err := db.ActivarAsignacion(agente, proyectoID, "frente prioridad"); err != nil {
			t.Fatalf("activar asignacion %s: %v", agente, err)
		}
	}

	for _, agente := range []string{"CodexBloqueadoA", "CodexBloqueadoB"} {
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      agente,
			ProyectoID:  &proyectoID,
			CWD:         filepath.Join(tmp, "orquestador"),
			Herramienta: "codex-cli",
		})
		if err != nil {
			t.Fatalf("iniciar sesion %s: %v", agente, err)
		}
		if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
			t.Fatalf("upsert handle %s: %v", agente, err)
		}
		handle, err := db.GetRuntimeHandleActivoAgenteProyecto(agente, &proyectoID)
		if err != nil || handle == nil {
			t.Fatalf("get handle %s: %+v err=%v", agente, handle, err)
		}
		if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='fallido', last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, handle.ID); err != nil {
			t.Fatalf("marcar handle fallido %s: %v", agente, err)
		}
	}

	sesionSana, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSano",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion sana: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesionSana); err != nil {
		t.Fatalf("upsert handle sano: %v", err)
	}
	for i := 0; i < autonomiaBlockedTaskRecoveryOpenTasksCeiling; i++ {
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      fmt.Sprintf("Trabajo sano %d", i+1),
			Descripcion: "test",
			ProyectoID:  &proyectoID,
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea sana %d: %v", i+1, err)
		}
		if err := db.TomarTarea(tareaID, "CodexSano"); err != nil {
			t.Fatalf("tomar tarea sana %d: %v", i+1, err)
		}
		if err := db.IniciarTarea(tareaID, "CodexSano"); err != nil {
			t.Fatalf("iniciar tarea sana %d: %v", i+1, err)
		}
	}

	crearTareaBloqueada := func(agente, titulo string) int64 {
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      titulo,
			Descripcion: "test",
			ProyectoID:  &proyectoID,
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea bloqueada %s: %v", titulo, err)
		}
		if err := db.TomarTarea(tareaID, agente); err != nil {
			t.Fatalf("tomar tarea bloqueada %s: %v", titulo, err)
		}
		if err := db.IniciarTarea(tareaID, agente); err != nil {
			t.Fatalf("iniciar tarea bloqueada %s: %v", titulo, err)
		}
		if err := db.BloquearTarea(tareaID, agente, "Agente "+agente+" en estado mailbox_atascada: mailbox pendiente con fallos de control"); err != nil {
			t.Fatalf("bloquear tarea %s: %v", titulo, err)
		}
		return tareaID
	}

	tareaA := crearTareaBloqueada("CodexBloqueadoA", "Bloqueada A")
	tareaB1 := crearTareaBloqueada("CodexBloqueadoB", "Bloqueada B1")
	tareaB2 := crearTareaBloqueada("CodexBloqueadoB", "Bloqueada B2")

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia procesar solo una tarea bloqueada, got=%d", procesadas)
	}

	tarea, err := db.GetTarea(tareaA)
	if err != nil {
		t.Fatalf("get tarea A: %v", err)
	}
	if tarea.Estado != db.EstadoBloqueada {
		t.Fatalf("la tarea del agente con menos bloqueadas deberia seguir bloqueada, got=%s", tarea.Estado)
	}

	reactivadas := 0
	for _, tareaID := range []int64{tareaB1, tareaB2} {
		tarea, err := db.GetTarea(tareaID)
		if err != nil {
			t.Fatalf("get tarea B %d: %v", tareaID, err)
		}
		if tarea.Estado == db.EstadoEnProgreso && tarea.Agente != nil && *tarea.Agente == "CodexSano" {
			reactivadas++
		}
	}
	if reactivadas != 1 {
		t.Fatalf("deberia priorizar una tarea del agente con mas bloqueadas, reactivadas=%d", reactivadas)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchRespetaCooldownReasignacionAutomatica(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	autonomiaDegradedTaskGate.Reset()

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-cooldown",
		Nombre:  "Orquestador Cooldown",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for _, agente := range []string{"CodexBloqueado", "CodexSano"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
		if err := db.ActivarAsignacion(agente, proyectoID, "frente cooldown"); err != nil {
			t.Fatalf("activar asignacion %s: %v", agente, err)
		}
	}
	sesionBloqueada, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexBloqueado",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion bloqueada: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesionBloqueada); err != nil {
		t.Fatalf("upsert handle bloqueado: %v", err)
	}
	handleBloqueado, err := db.GetRuntimeHandleActivoAgenteProyecto("CodexBloqueado", &proyectoID)
	if err != nil || handleBloqueado == nil {
		t.Fatalf("get handle bloqueado: %+v err=%v", handleBloqueado, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='fallido', last_seen_at=CURRENT_TIMESTAMP WHERE id=?`, handleBloqueado.ID); err != nil {
		t.Fatalf("marcar handle fallido: %v", err)
	}
	sesionSana, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexSano",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion sana: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesionSana); err != nil {
		t.Fatalf("upsert handle sano: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "No rebotar inmediatamente",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "CodexBloqueado"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "CodexBloqueado"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.AnotarTarea(tareaID, "orquesta", "Reasignada automáticamente desde Codex8 a CodexBloqueado: Agente Codex8 en estado mailbox_atascada"); err != nil {
		t.Fatalf("anotar tarea: %v", err)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if procesadas != 0 {
		t.Fatalf("no deberia reintervenir una tarea recien reasignada, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "CodexBloqueado" || tarea.Estado != db.EstadoEnProgreso {
		t.Fatalf("la tarea no deberia haberse tocado: %+v", tarea)
	}
}

func TestSeleccionarRelevoAutonomiaDescartaCandidatoEnMailboxAtascada(t *testing.T) {
	proyectoID := int64(42)
	tarea := &db.Tarea{ID: 7, ProyectoID: &proyectoID}
	rows := []agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexMailbox"},
			Asignacion:      &db.Asignacion{Agente: "CodexMailbox", ProyectoID: proyectoID, Estado: db.AsignacionActiva},
			EstadoOperativo: "mailbox_atascada",
			MailboxPending:  1,
		},
		{
			Agente:          &db.Agente{Nombre: "CodexSano"},
			Asignacion:      &db.Asignacion{Agente: "CodexSano", ProyectoID: proyectoID, Estado: db.AsignacionActiva},
			EstadoOperativo: "disponible",
		},
	}

	relevo := seleccionarRelevoAutonomia(rows, map[string]int{"CodexMailbox": 0, "CodexSano": 0}, tarea, "CodexBloqueado")
	if relevo != "CodexSano" {
		t.Fatalf("deberia descartar candidato atascado por mailbox, got=%q", relevo)
	}
}

func TestSeleccionarRelevoAutonomiaPermiteHistorialDeFallosSiElEstadoActualEsDisponible(t *testing.T) {
	proyectoID := int64(42)
	tarea := &db.Tarea{ID: 8, ProyectoID: &proyectoID}
	rows := []agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexConHistorial"},
			Asignacion:      &db.Asignacion{Agente: "CodexConHistorial", ProyectoID: proyectoID, Estado: db.AsignacionActiva},
			EstadoOperativo: "disponible",
			OrdersFailed:    7,
		},
	}

	relevo := seleccionarRelevoAutonomia(rows, map[string]int{"CodexConHistorial": 0}, tarea, "CodexBloqueado")
	if relevo != "CodexConHistorial" {
		t.Fatalf("deberia permitir relevo disponible aunque tenga fallos historicos, got=%q", relevo)
	}
}

func TestSeleccionarRelevoAutonomiaDescartaPremiumConTrabajoAbiertoSiLaTareaEsAcotada(t *testing.T) {
	proyectoID := int64(42)
	tarea := &db.Tarea{
		ID:          88,
		ProyectoID:  &proyectoID,
		Descripcion: "Frente premium acotado.\nWRITE_SET: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestSeleccionarRelevoAutonomiaDescartaPremiumConTrabajoAbiertoSiLaTareaEsAcotada$'",
	}
	rows := []agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexOcupado"},
			Asignacion:      &db.Asignacion{Agente: "CodexOcupado", ProyectoID: proyectoID, Estado: db.AsignacionActiva},
			EstadoOperativo: "disponible",
			OpenTasks:       1,
		},
		{
			Agente:          &db.Agente{Nombre: "CodexLibre"},
			Asignacion:      &db.Asignacion{Agente: "CodexLibre", ProyectoID: proyectoID, Estado: db.AsignacionActiva},
			EstadoOperativo: "disponible",
			OpenTasks:       0,
		},
	}

	relevo := seleccionarRelevoAutonomia(rows, map[string]int{"CodexOcupado": 1, "CodexLibre": 0}, tarea, "CodexBloqueado")
	if relevo != "CodexLibre" {
		t.Fatalf("deberia preferir el premium sin trabajo abierto para un frente acotado, got=%q", relevo)
	}
}

func TestSeleccionarRelevoAutonomiaDescartaPremiumExclusivoConTrabajoAbiertoParaTareaNoAcotada(t *testing.T) {
	proyectoID := int64(42)
	tarea := &db.Tarea{
		ID:         89,
		ProyectoID: &proyectoID,
		Titulo:     "Tarea legacy amplia",
	}
	rows := []agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexExclusivo"},
			Asignacion:      &db.Asignacion{Agente: "CodexExclusivo", ProyectoID: proyectoID, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva, Nota: "microciclo_exclusivo"},
			EstadoOperativo: "disponible",
			OpenTasks:       1,
		},
		{
			Agente:          &db.Agente{Nombre: "CodexLibre"},
			Asignacion:      &db.Asignacion{Agente: "CodexLibre", ProyectoID: proyectoID, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva},
			EstadoOperativo: "disponible",
			OpenTasks:       0,
		},
	}

	relevo := seleccionarRelevoAutonomia(rows, map[string]int{"CodexExclusivo": 1, "CodexLibre": 0}, tarea, "CodexBloqueado")
	if relevo != "CodexLibre" {
		t.Fatalf("deberia descartar al premium exclusivo ocupado para tarea no acotada, got=%q", relevo)
	}
}

func TestSeleccionarRelevoAutonomiaDescartaPremiumExclusivoConTrabajoAbiertoParaOtroProyecto(t *testing.T) {
	proyectoAsignado := int64(42)
	proyectoAjeno := int64(77)
	tarea := &db.Tarea{
		ID:          90,
		ProyectoID:  &proyectoAjeno,
		Descripcion: "Frente premium acotado.\nWRITE_SET: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestSeleccionarRelevoAutonomiaDescartaPremiumExclusivoConTrabajoAbiertoParaOtroProyecto$'",
	}
	rows := []agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexExclusivo"},
			Asignacion:      &db.Asignacion{Agente: "CodexExclusivo", ProyectoID: proyectoAsignado, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva, Nota: "microciclo_exclusivo"},
			EstadoOperativo: "disponible",
			OpenTasks:       1,
		},
		{
			Agente:          &db.Agente{Nombre: "CodexLibre"},
			Asignacion:      &db.Asignacion{Agente: "CodexLibre", ProyectoID: proyectoAjeno, ProyectoSlug: "multi-app", Estado: db.AsignacionActiva},
			EstadoOperativo: "disponible",
			OpenTasks:       0,
		},
	}

	relevo := seleccionarRelevoAutonomia(rows, map[string]int{"CodexExclusivo": 1, "CodexLibre": 0}, tarea, "CodexBloqueado")
	if relevo != "CodexLibre" {
		t.Fatalf("deberia descartar al premium exclusivo ocupado para otro proyecto, got=%q", relevo)
	}
}

func TestSeleccionarRelevoAutonomiaPermitePremiumExclusivoLibreParaFrenteAcotadoDelMismoProyecto(t *testing.T) {
	proyectoID := int64(42)
	tarea := &db.Tarea{
		ID:          91,
		ProyectoID:  &proyectoID,
		Descripcion: "Frente premium acotado.\nWRITE_SET: cmd/controlplane_support.go\nTests minimos: go test ./cmd -run 'TestSeleccionarRelevoAutonomiaPermitePremiumExclusivoLibreParaFrenteAcotadoDelMismoProyecto$'",
	}
	rows := []agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexExclusivo"},
			Asignacion:      &db.Asignacion{Agente: "CodexExclusivo", ProyectoID: proyectoID, ProyectoSlug: "orquestador", Estado: db.AsignacionActiva, Nota: "microciclo_exclusivo"},
			EstadoOperativo: "disponible",
			OpenTasks:       0,
		},
	}

	relevo := seleccionarRelevoAutonomia(rows, map[string]int{"CodexExclusivo": 0}, tarea, "CodexBloqueado")
	if relevo != "CodexExclusivo" {
		t.Fatalf("deberia permitir al premium exclusivo libre en su propio carril acotado, got=%q", relevo)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchRedistribuyeSobrecargaWorkerSano(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-sobrecarga",
		Nombre:  "Orquestador Sobrecarga",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for _, agente := range []string{"CodexCargado", "CodexLibre"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
		if err := db.ActivarAsignacion(agente, proyectoID, "frente sobrecarga"); err != nil {
			t.Fatalf("activar asignacion %s: %v", agente, err)
		}
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      agente,
			ProyectoID:  &proyectoID,
			CWD:         filepath.Join(tmp, "orquestador"),
			Herramienta: "codex-cli",
		})
		if err != nil {
			t.Fatalf("iniciar sesion %s: %v", agente, err)
		}
		if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
			t.Fatalf("upsert handle %s: %v", agente, err)
		}
	}

	ids := make([]int64, 0, 3)
	for i := 0; i < autonomiaWorkerOpenTasksCeiling+1; i++ {
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      fmt.Sprintf("Sobrecarga %d", i+1),
			Descripcion: "test",
			ProyectoID:  &proyectoID,
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea %d: %v", i+1, err)
		}
		if err := db.TomarTarea(tareaID, "CodexCargado"); err != nil {
			t.Fatalf("tomar tarea %d: %v", i+1, err)
		}
		if err := db.IniciarTarea(tareaID, "CodexCargado"); err != nil {
			t.Fatalf("iniciar tarea %d: %v", i+1, err)
		}
		ids = append(ids, tareaID)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados/sobrecarga: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia redistribuir una tarea por sobrecarga, got=%d", procesadas)
	}

	reasignadas := 0
	for _, tareaID := range ids {
		tarea, err := db.GetTarea(tareaID)
		if err != nil {
			t.Fatalf("get tarea %d: %v", tareaID, err)
		}
		if tarea.Agente != nil && *tarea.Agente == "CodexLibre" && tarea.Estado == db.EstadoEnProgreso {
			reasignadas++
		}
	}
	if reasignadas != 1 {
		t.Fatalf("deberia haber exactamente una tarea redistribuida al relevo sano, got=%d", reasignadas)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchBloqueaSobrecargaSinRelevoSano(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-sobrecarga-bloqueo",
		Nombre:  "Orquestador Sobrecarga Bloqueo",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("CodexCargado", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ActivarAsignacion("CodexCargado", proyectoID, "frente sobrecarga"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexCargado",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert handle: %v", err)
	}

	ids := make([]int64, 0, autonomiaWorkerOpenTasksCeiling+1)
	for i := 0; i < autonomiaWorkerOpenTasksCeiling+1; i++ {
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      fmt.Sprintf("Sobrecarga sin relevo %d", i+1),
			Descripcion: "test",
			ProyectoID:  &proyectoID,
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea %d: %v", i+1, err)
		}
		if err := db.TomarTarea(tareaID, "CodexCargado"); err != nil {
			t.Fatalf("tomar tarea %d: %v", i+1, err)
		}
		if err := db.IniciarTarea(tareaID, "CodexCargado"); err != nil {
			t.Fatalf("iniciar tarea %d: %v", i+1, err)
		}
		ids = append(ids, tareaID)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados/sobrecarga: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia bloquear una tarea por sobrecarga sin relevo, got=%d", procesadas)
	}

	bloqueadas := 0
	for _, tareaID := range ids {
		tarea, err := db.GetTarea(tareaID)
		if err != nil {
			t.Fatalf("get tarea %d: %v", tareaID, err)
		}
		if tarea.Estado == db.EstadoBloqueada {
			bloqueadas++
		}
	}
	if bloqueadas != 1 {
		t.Fatalf("deberia quedar una tarea bloqueada por sobrecarga, got=%d", bloqueadas)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchRecuperaBloqueoPorSobrecargaCuandoHayHueco(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-sobrecarga-recupera",
		Nombre:  "Orquestador Sobrecarga Recupera",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("CodexCapaz", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ActivarAsignacion("CodexCapaz", proyectoID, "frente sobrecarga"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "CodexCapaz",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert handle: %v", err)
	}

	tareaActivaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Activa",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea activa: %v", err)
	}
	if err := db.TomarTarea(tareaActivaID, "CodexCapaz"); err != nil {
		t.Fatalf("tomar tarea activa: %v", err)
	}
	if err := db.IniciarTarea(tareaActivaID, "CodexCapaz"); err != nil {
		t.Fatalf("iniciar tarea activa: %v", err)
	}

	tareaBloqueadaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Bloqueada por sobrecarga",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea bloqueada: %v", err)
	}
	if err := db.TomarTarea(tareaBloqueadaID, "CodexCapaz"); err != nil {
		t.Fatalf("tomar tarea bloqueada: %v", err)
	}
	if err := db.IniciarTarea(tareaBloqueadaID, "CodexCapaz"); err != nil {
		t.Fatalf("iniciar tarea bloqueada: %v", err)
	}
	if err := db.BloquearTarea(tareaBloqueadaID, "CodexCapaz", "Sobrecarga operativa: sin relevo sano disponible"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados/sobrecarga: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia recuperar una tarea bloqueada por sobrecarga al volver a haber hueco, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaBloqueadaID)
	if err != nil {
		t.Fatalf("get tarea recuperada: %v", err)
	}
	if tarea.Estado != db.EstadoEnProgreso || tarea.Agente == nil || *tarea.Agente != "CodexCapaz" {
		t.Fatalf("la tarea bloqueada por sobrecarga deberia volver a en_progreso en el mismo agente: %+v", tarea)
	}
}

func TestAutonomiaOpenTasksCeilingForRowElevaTMUXFresh(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	now := time.Now().UTC()
	traceDir := filepath.Join(tmp, "runtime", "codex7")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	manifestRaw, _ := json.Marshal(map[string]any{
		"version":    1,
		"agent":      "Codex7",
		"driver":     "tmux_cli_session",
		"transport":  "tmux",
		"created_at": now.Format(time.RFC3339Nano),
		"started_at": now.Format(time.RFC3339Nano),
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
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
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	hb := now
	row := agentesapp.Row{
		Handle:          &db.RuntimeHandle{MetadataJSON: string(metaJSON)},
		WorkerState:     "running",
		WorkerAlive:     true,
		WorkerHeartbeat: &hb,
		WorkerUpdatedAt: &hb,
		EstadoOperativo: "trabajando",
	}

	if got := autonomiaOpenTasksCeilingForRow(row, now); got != autonomiaTMUXWorkerOpenTasksCeilingDefault {
		t.Fatalf("ceiling tmux inesperado: %d", got)
	}
}

func TestAutonomiaOpenTasksCeilingForRowElevaTMUXFreshDesdeRowCanonico(t *testing.T) {
	_ = prepararDBTemporalCmd(t)
	now := time.Now().UTC()
	row := agentesapp.Row{
		WorkerDriver:    "tmux_cli_session",
		WorkerTransport: "tmux",
		WorkerState:     "running",
		WorkerAlive:     true,
		WorkerHeartbeat: timePtr(now),
		EstadoOperativo: "trabajando",
	}

	if got := autonomiaOpenTasksCeilingForRow(row, now); got != autonomiaTMUXWorkerOpenTasksCeilingDefault {
		t.Fatalf("ceiling tmux inesperado desde row canonico: %d", got)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchRecuperaBloqueoPorSobrecargaHastaTechoTMUX(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	now := time.Now().UTC()
	traceDir := filepath.Join(tmp, "runtime", "codex7")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	manifestRaw, _ := json.Marshal(map[string]any{
		"version":    1,
		"agent":      "Codex7",
		"driver":     "tmux_cli_session",
		"transport":  "tmux",
		"created_at": now.Format(time.RFC3339Nano),
		"started_at": now.Format(time.RFC3339Nano),
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
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
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-sobrecarga-tmux",
		Nombre:  "Orquestador Sobrecarga TMUX",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex7", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ActivarAsignacion("Codex7", proyectoID, "frente sobrecarga tmux"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex7",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='activo', metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='esperando_io', process_state='running', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtime.ID); err != nil {
		t.Fatalf("update runtime: %v", err)
	}

	for i := 0; i < autonomiaWorkerOpenTasksCeiling; i++ {
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      fmt.Sprintf("Activa TMUX %d", i+1),
			Descripcion: "test",
			ProyectoID:  &proyectoID,
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea activa %d: %v", i+1, err)
		}
		if err := db.TomarTarea(tareaID, "Codex7"); err != nil {
			t.Fatalf("tomar tarea activa %d: %v", i+1, err)
		}
		if err := db.IniciarTarea(tareaID, "Codex7"); err != nil {
			t.Fatalf("iniciar tarea activa %d: %v", i+1, err)
		}
	}

	tareaBloqueadaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Bloqueada por sobrecarga con TMUX",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea bloqueada: %v", err)
	}
	if err := db.TomarTarea(tareaBloqueadaID, "Codex7"); err != nil {
		t.Fatalf("tomar tarea bloqueada: %v", err)
	}
	if err := db.IniciarTarea(tareaBloqueadaID, "Codex7"); err != nil {
		t.Fatalf("iniciar tarea bloqueada: %v", err)
	}
	if err := db.BloquearTarea(tareaBloqueadaID, "Codex7", "Sobrecarga operativa: sin relevo sano disponible"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados/sobrecarga: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia reactivar una tercera tarea en worker tmux sano, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaBloqueadaID)
	if err != nil {
		t.Fatalf("get tarea recuperada: %v", err)
	}
	if tarea.Estado != db.EstadoEnProgreso || tarea.Agente == nil || *tarea.Agente != "Codex7" {
		t.Fatalf("la tarea bloqueada por sobrecarga deberia volver a en_progreso en Codex7: %+v", tarea)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchReasignaBloqueoPorSobrecargaASiHayRelevo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-sobrecarga-relevo",
		Nombre:  "Orquestador Sobrecarga Relevo",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	for _, agente := range []string{"CodexCargado", "CodexLibre"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
		if err := db.ActivarAsignacion(agente, proyectoID, "frente sobrecarga"); err != nil {
			t.Fatalf("activar asignacion %s: %v", agente, err)
		}
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      agente,
			ProyectoID:  &proyectoID,
			CWD:         filepath.Join(tmp, "orquestador"),
			Herramienta: "codex-cli",
		})
		if err != nil {
			t.Fatalf("iniciar sesion %s: %v", agente, err)
		}
		if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
			t.Fatalf("upsert handle %s: %v", agente, err)
		}
	}

	for i := 0; i < autonomiaWorkerOpenTasksCeiling; i++ {
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      fmt.Sprintf("Carga %d", i+1),
			Descripcion: "test",
			ProyectoID:  &proyectoID,
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear carga %d: %v", i+1, err)
		}
		if err := db.TomarTarea(tareaID, "CodexCargado"); err != nil {
			t.Fatalf("tomar carga %d: %v", i+1, err)
		}
		if err := db.IniciarTarea(tareaID, "CodexCargado"); err != nil {
			t.Fatalf("iniciar carga %d: %v", i+1, err)
		}
	}

	tareaBloqueadaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Bloqueada por sobrecarga con relevo",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea bloqueada: %v", err)
	}
	if err := db.TomarTarea(tareaBloqueadaID, "CodexCargado"); err != nil {
		t.Fatalf("tomar tarea bloqueada: %v", err)
	}
	if err := db.IniciarTarea(tareaBloqueadaID, "CodexCargado"); err != nil {
		t.Fatalf("iniciar tarea bloqueada: %v", err)
	}
	if err := db.BloquearTarea(tareaBloqueadaID, "CodexCargado", "Sobrecarga operativa: sin relevo sano disponible"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados/sobrecarga: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia reasignar una tarea bloqueada por sobrecarga a un relevo sano, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaBloqueadaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != db.EstadoEnProgreso || tarea.Agente == nil || *tarea.Agente != "CodexLibre" {
		t.Fatalf("la tarea deberia reasignarse al relevo sano: %+v", tarea)
	}
}

func TestSeleccionarRelevoAutonomiaConLimiteDescartaWorkerTrabajandoConDosTareas(t *testing.T) {
	proyectoID := int64(42)
	tarea := &db.Tarea{ID: 9, ProyectoID: &proyectoID}
	rows := []agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexCargado"},
			Asignacion:      &db.Asignacion{Agente: "CodexCargado", ProyectoID: proyectoID, Estado: db.AsignacionActiva},
			EstadoOperativo: "trabajando",
			OpenTasks:       2,
		},
	}

	relevo := seleccionarRelevoAutonomiaConLimite(rows, map[string]int{"CodexCargado": 2}, tarea, "CodexBloqueado", autonomiaBlockedTaskRecoveryOpenTasksCeiling)
	if relevo != "" {
		t.Fatalf("no deberia reutilizar un relevo que ya alcanzo el techo operativo de tareas, got=%q", relevo)
	}
}

func TestSeleccionarRelevoAutonomiaConLimiteAceptaTMUXFreshConDosTareas(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	now := time.Now().UTC()
	traceDir := filepath.Join(tmp, "runtime", "codex-tmux")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	manifestRaw, _ := json.Marshal(map[string]any{
		"version":    1,
		"agent":      "CodexTMUX",
		"driver":     "tmux_cli_session",
		"transport":  "tmux",
		"created_at": now.Format(time.RFC3339Nano),
		"started_at": now.Format(time.RFC3339Nano),
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":        true,
		"heartbeat_at": now.Format(time.RFC3339Nano),
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
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	proyectoID := int64(42)
	tarea := &db.Tarea{ID: 11, ProyectoID: &proyectoID}
	hb := now
	rows := []agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexTMUX"},
			Asignacion:      &db.Asignacion{Agente: "CodexTMUX", ProyectoID: proyectoID, Estado: db.AsignacionActiva},
			EstadoOperativo: "trabajando",
			OpenTasks:       2,
			WorkerState:     "running",
			WorkerAlive:     true,
			WorkerHeartbeat: &hb,
			Handle:          &db.RuntimeHandle{MetadataJSON: string(metaJSON)},
		},
	}

	relevo := seleccionarRelevoAutonomiaConLimite(rows, map[string]int{"CodexTMUX": 2}, tarea, "CodexBloqueado", autonomiaBlockedTaskRecoveryOpenTasksCeiling)
	if relevo != "CodexTMUX" {
		t.Fatalf("deberia aceptar relevo tmux fresco con dos tareas, got=%q", relevo)
	}
}

func TestSeleccionarRelevoAutonomiaConLimiteDescartaWorkerTrabajandoConMasDeDosTareas(t *testing.T) {
	proyectoID := int64(42)
	tarea := &db.Tarea{ID: 10, ProyectoID: &proyectoID}
	rows := []agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexSaturado"},
			Asignacion:      &db.Asignacion{Agente: "CodexSaturado", ProyectoID: proyectoID, Estado: db.AsignacionActiva},
			EstadoOperativo: "trabajando",
			OpenTasks:       3,
		},
	}

	relevo := seleccionarRelevoAutonomiaConLimite(rows, map[string]int{"CodexSaturado": 3}, tarea, "CodexBloqueado", autonomiaBlockedTaskRecoveryOpenTasksCeiling)
	if relevo != "" {
		t.Fatalf("no deberia reutilizar un relevo trabajando con mas de dos tareas abiertas, got=%q", relevo)
	}
}

func TestSeleccionarRelevoAutonomiaBloqueadaAceptaTMUXFreshConTresTareas(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	now := time.Now().UTC()
	traceDir := filepath.Join(tmp, "runtime", "codex-tmux-burst")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	manifestRaw, _ := json.Marshal(map[string]any{
		"version":    1,
		"agent":      "CodexTMUX",
		"driver":     "tmux_cli_session",
		"transport":  "tmux",
		"created_at": now.Format(time.RFC3339Nano),
		"started_at": now.Format(time.RFC3339Nano),
	})
	statusRaw, _ := json.Marshal(map[string]any{
		"state":          "running",
		"updated_at":     now.Format(time.RFC3339Nano),
		"last_output_at": now.Format(time.RFC3339Nano),
		"alive":          true,
	})
	heartbeatRaw, _ := json.Marshal(map[string]any{
		"alive":          true,
		"heartbeat_at":   now.Format(time.RFC3339Nano),
		"last_output_at": now.Format(time.RFC3339Nano),
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
	metaJSON, _ := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	proyectoID := int64(42)
	tarea := &db.Tarea{ID: 12, ProyectoID: &proyectoID}
	hb := now
	rows := []agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexTMUX"},
			Asignacion:      &db.Asignacion{Agente: "CodexTMUX", ProyectoID: proyectoID, Estado: db.AsignacionActiva},
			EstadoOperativo: "trabajando",
			OpenTasks:       3,
			WorkerState:     "running",
			WorkerAlive:     true,
			WorkerHeartbeat: &hb,
			Handle:          &db.RuntimeHandle{MetadataJSON: string(metaJSON)},
		},
	}

	relevo := seleccionarRelevoAutonomiaBloqueada(rows, map[string]int{"CodexTMUX": 3}, tarea, "CodexBloqueado")
	if relevo != "CodexTMUX" {
		t.Fatalf("deberia aceptar relevo tmux fresco con tres tareas en recuperación bloqueada, got=%q", relevo)
	}
}

func TestSeleccionarRelevoAutonomiaBloqueadaAceptaTMUXFreshConTresTareasDesdeRowCanonico(t *testing.T) {
	_ = prepararDBTemporalCmd(t)
	now := time.Now().UTC()
	proyectoID := int64(42)
	tarea := &db.Tarea{ID: 14, ProyectoID: &proyectoID}
	rows := []agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexTMUX"},
			Asignacion:      &db.Asignacion{Agente: "CodexTMUX", ProyectoID: proyectoID, Estado: db.AsignacionActiva},
			EstadoOperativo: "trabajando",
			OpenTasks:       3,
			WorkerDriver:    "tmux_cli_session",
			WorkerTransport: "tmux",
			WorkerState:     "running",
			WorkerAlive:     true,
			WorkerHeartbeat: timePtr(now),
		},
	}

	relevo := seleccionarRelevoAutonomiaBloqueada(rows, map[string]int{"CodexTMUX": 3}, tarea, "CodexBloqueado")
	if relevo != "CodexTMUX" {
		t.Fatalf("deberia aceptar relevo tmux fresco con tres tareas desde row canonico, got=%q", relevo)
	}
}

func TestSeleccionarRelevoAutonomiaBloqueadaAceptaWorkerConDosTareasParaRecuperarBacklog(t *testing.T) {
	proyectoID := int64(42)
	tarea := &db.Tarea{ID: 15, ProyectoID: &proyectoID}
	rows := []agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexSano"},
			Asignacion:      &db.Asignacion{Agente: "CodexSano", ProyectoID: proyectoID, Estado: db.AsignacionActiva},
			EstadoOperativo: "trabajando",
			OpenTasks:       2,
		},
	}

	relevo := seleccionarRelevoAutonomiaBloqueada(rows, map[string]int{"CodexSano": 2}, tarea, "CodexBloqueado")
	if relevo != "CodexSano" {
		t.Fatalf("deberia permitir un burst controlado para recuperar backlog bloqueado, got=%q", relevo)
	}
}

func TestSeleccionarRelevoAutonomiaBloqueadaDescartaWorkerNoTMUXConTresTareas(t *testing.T) {
	_ = prepararDBTemporalCmd(t)
	proyectoID := int64(42)
	tarea := &db.Tarea{ID: 13, ProyectoID: &proyectoID}
	rows := []agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexNoTMUX"},
			Asignacion:      &db.Asignacion{Agente: "CodexNoTMUX", ProyectoID: proyectoID, Estado: db.AsignacionActiva},
			EstadoOperativo: "trabajando",
			OpenTasks:       3,
		},
	}

	relevo := seleccionarRelevoAutonomiaBloqueada(rows, map[string]int{"CodexNoTMUX": 3}, tarea, "CodexBloqueado")
	if relevo != "" {
		t.Fatalf("no deberia aceptar un relevo no tmux con tres tareas abiertas, got=%q", relevo)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchRecuperaTareaBloqueadaSiElWorkerVuelve(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()
	statusCacheState.mu.Lock()
	statusCacheState.ok = true
	statusCacheState.value = apiStatusResponse{Generado: "stale"}
	statusCacheState.expires = time.Now().UTC().Add(time.Minute)
	statusCacheState.mu.Unlock()
	now := time.Now().UTC()
	statusPath := filepath.Join(tmp, "status.json")
	heartbeatPath := filepath.Join(tmp, "heartbeat.json")
	statusRaw, _ := json.Marshal(map[string]any{
		"state":      "running",
		"updated_at": now.Format(time.RFC3339Nano),
		"alive":      true,
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
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-recuperacion-bloqueo",
		Nombre:  "Orquestador Recuperacion Bloqueo",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex7", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ActivarAsignacion("Codex7", proyectoID, "frente recuperable"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex7",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='activo', metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='esperando_io', process_state='running', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtime.ID); err != nil {
		t.Fatalf("update runtime: %v", err)
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Recuperar tarea bloqueada por degradacion",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Codex7", "Agente Codex7 en estado bloqueado_por_runtime: pausado"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia reactivar una tarea, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != db.EstadoEnProgreso {
		t.Fatalf("la tarea deberia volver a en_progreso, got=%s", tarea.Estado)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex7" {
		t.Fatalf("la tarea deberia seguir en Codex7: %+v", tarea)
	}
	bloqueos, err := db.ListarResumenBloqueos()
	if err != nil {
		t.Fatalf("listar bloqueos: %v", err)
	}
	for _, bloqueo := range bloqueos {
		if bloqueo.ID == tareaID {
			t.Fatalf("el bloqueo deberia quedar resuelto: %+v", bloqueo)
		}
	}
	statusCacheState.mu.Lock()
	cacheValida := statusCacheState.ok
	statusCacheState.mu.Unlock()
	if cacheValida {
		t.Fatalf("la recuperacion automatica deberia invalidar la snapshot de status")
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchRecuperaTareaBloqueadaSiElWorkerSigueVivoPeroPausado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex7", "programador"); err != nil {
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
	traceDir := filepath.Join(tmp, "runtime", "codex7")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","alive":true,"updated_at":"`+time.Now().UTC().Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"state":"running","alive":true,"timestamp":"`+time.Now().UTC().Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	meta := map[string]any{
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	if err := db.ActivarAsignacion("Codex7", proyectoID, "frente recuperable"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex7",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado', metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='pausado', process_state='stopped', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtime.ID); err != nil {
		t.Fatalf("update runtime: %v", err)
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Recuperar tarea de worker pausado",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Codex7", "Agente Codex7 en estado bloqueado_por_runtime: pausado"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia reactivar una tarea de worker vivo aunque siga pausado, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != db.EstadoEnProgreso {
		t.Fatalf("la tarea deberia volver a en_progreso, got=%s", tarea.Estado)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchRecuperaTareaBloqueadaConTMUXBootstrapOnlyFresco(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	traceDir := filepath.Join(tmp, "runtime", "gemini1")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	now := time.Now().UTC()
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(statusPath, []byte(`{"state":"starting","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"state":"starting","alive":true,"timestamp":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-gemini1-bootstrap",
		"mailbox_delivery_mode": string(runtimeagente.MailboxDeliveryBootstrapOnly),
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	if err := db.ActivarAsignacion("Gemini1", proyectoID, "frente recuperable"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='activo', metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='esperando_io', process_state='running', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtime.ID); err != nil {
		t.Fatalf("update runtime: %v", err)
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Recuperar tarea bloqueada de TMUX bootstrap",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Gemini1", "Agente Gemini1 en estado bloqueado_por_runtime: worker recuperado"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia reactivar una tarea bloqueada con tmux bootstrap_only fresco, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != db.EstadoEnProgreso {
		t.Fatalf("la tarea deberia volver a en_progreso, got=%s", tarea.Estado)
	}
	if tarea.Agente == nil || *tarea.Agente != "Gemini1" {
		t.Fatalf("la tarea deberia seguir en Gemini1: %+v", tarea)
	}
}

func TestProcesarAutonomiaSesionActivaRecuperaTareaBloqueadaConTMUXBootstrapOnlyFresco(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Gemini1", "programador"); err != nil {
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
	traceDir := filepath.Join(tmp, "runtime", "gemini1-sesion")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	now := time.Now().UTC()
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(statusPath, []byte(`{"state":"starting","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"state":"starting","alive":true,"timestamp":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-gemini1-bootstrap",
		"mailbox_delivery_mode": string(runtimeagente.MailboxDeliveryBootstrapOnly),
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	if err := db.ActivarAsignacion("Gemini1", proyectoID, "frente recuperable"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Gemini1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "gemini-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='activo', metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='esperando_io', process_state='running', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtime.ID); err != nil {
		t.Fatalf("update runtime: %v", err)
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Recuperar tarea bloqueada desde sesion activa",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Gemini1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Gemini1", "Agente Gemini1 en estado bloqueado_por_runtime: worker recuperado"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	procesadas, err := procesarAutonomiaSesionActiva(sesion)
	if err != nil {
		t.Fatalf("procesar autonomia sesion activa: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia reactivar una tarea bloqueada desde la sesion activa, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != db.EstadoEnProgreso {
		t.Fatalf("la tarea deberia volver a en_progreso, got=%s", tarea.Estado)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchRecuperaTareaBloqueadaConMotivoLegacyDegradado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
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
	traceDir := filepath.Join(tmp, "runtime", "codex3")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	now := time.Now().UTC()
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"state":"running","alive":true,"timestamp":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	meta := map[string]any{
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	if err := db.ActivarAsignacion("Codex3", proyectoID, "frente legacy"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex3",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='activo', metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	runtime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtime == nil {
		t.Fatalf("runtime: %+v err=%v", runtime, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='esperando_io', process_state='running', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtime.ID); err != nil {
		t.Fatalf("update runtime: %v", err)
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Recuperar tarea legacy degradada",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex3"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex3"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Codex3", "Agente degradado: mailbox pendiente y runtime no disponible para entrega inmediata"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia reactivar una tarea legacy de agente degradado, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != db.EstadoEnProgreso {
		t.Fatalf("la tarea legacy deberia volver a en_progreso, got=%s", tarea.Estado)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchReasignaTareaBloqueadaARelevoSano(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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
	for _, agente := range []string{"Codex8", "Codex7"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
		if err := db.ActivarAsignacion(agente, proyectoID, "frente"); err != nil {
			t.Fatalf("activar asignacion %s: %v", agente, err)
		}
	}

	// Codex8: degradado con tarea bloqueada propia.
	sesionBloqueada, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex8",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion degradada: %v", err)
	}
	handleBloqueado, err := db.GetRuntimeHandleBySesionID(sesionBloqueada.ID)
	if err != nil || handleBloqueado == nil {
		t.Fatalf("handle degradado: %+v err=%v", handleBloqueado, err)
	}
	runtimeBloqueado, err := db.GetRuntimeBySesionID(sesionBloqueada.ID)
	if err != nil || runtimeBloqueado == nil {
		t.Fatalf("runtime degradado: %+v err=%v", runtimeBloqueado, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handleBloqueado.ID); err != nil {
		t.Fatalf("pausar handle degradado: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='pausado', process_state='stopped', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtimeBloqueado.ID); err != nil {
		t.Fatalf("pausar runtime degradado: %v", err)
	}

	// Codex7: relevo sano disponible con worker estructurado fresco.
	traceDir := filepath.Join(tmp, "runtime", "codex7")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	now := time.Now().UTC()
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"state":"running","alive":true,"timestamp":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata relevo: %v", err)
	}
	sesionSana, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex7",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion relevo: %v", err)
	}
	handleSano, err := db.GetRuntimeHandleBySesionID(sesionSana.ID)
	if err != nil || handleSano == nil {
		t.Fatalf("handle relevo: %+v err=%v", handleSano, err)
	}
	runtimeSano, err := db.GetRuntimeBySesionID(sesionSana.ID)
	if err != nil || runtimeSano == nil {
		t.Fatalf("runtime relevo: %+v err=%v", runtimeSano, err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='activo', metadata_json=? WHERE id=?`, string(metaJSON), handleSano.ID); err != nil {
		t.Fatalf("activar handle relevo: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='esperando_io', process_state='running', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtimeSano.ID); err != nil {
		t.Fatalf("activar runtime relevo: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Reasignar tarea bloqueada",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex8"); err != nil {
		t.Fatalf("tomar tarea degradada: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex8"); err != nil {
		t.Fatalf("iniciar tarea degradada: %v", err)
	}
	if err := db.BloquearTarea(tareaID, "Codex8", "Agente Codex8 en estado bloqueado_por_runtime: pausado"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia reasignar una tarea bloqueada a relevo sano, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != db.EstadoEnProgreso {
		t.Fatalf("la tarea deberia quedar en_progreso, got=%s", tarea.Estado)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex7" {
		t.Fatalf("la tarea deberia quedar reasignada a Codex7: %+v", tarea)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchReiniciaWorkerTMUXAtascado(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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
	if err := db.RegistrarAgente("Codex7", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ActivarAsignacion("Codex7", proyectoID, "frente"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex7",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtimeInst, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtimeInst == nil {
		t.Fatalf("runtime: %+v err=%v", runtimeInst, err)
	}

	traceDir := filepath.Join(tmp, "runtime", "codex7")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	now := time.Now().UTC()
	lastOutput := now.Add(-5 * time.Hour)
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex7","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex7-test","created_at":"`+now.Format(time.RFC3339Nano)+`","started_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+lastOutput.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+lastOutput.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='activo', metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("activar handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='esperando_io', process_state='running', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtimeInst.ID); err != nil {
		t.Fatalf("activar runtime: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Worker tmux atascado",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	rows, buildErr := agentesService.BuildPanelRows()
	if buildErr != nil {
		t.Fatalf("build rows: %v", buildErr)
	}
	t.Logf("DEBUG rows=%d", len(rows))
	for _, row := range rows {
		if row.Agente != nil {
			t.Logf("DEBUG agente=%s estado=%s opentasks=%d blocked=%d mailbox=%d workerDriver=%s workerAlive=%v", row.Agente.Nombre, row.EstadoOperativo, row.OpenTasks, row.BlockedTasks, row.MailboxPending, row.WorkerDriver, row.WorkerAlive)
		}
		if row.Agente != nil && strings.EqualFold(strings.TrimSpace(row.Agente.Nombre), "Codex7") {
			runtimeState := ""
			if row.Runtime != nil {
				runtimeState = strings.TrimSpace(row.Runtime.LogicalState)
			}
			handleState := ""
			if row.Handle != nil {
				handleState = strings.TrimSpace(row.Handle.Estado)
			}
			t.Logf("DEBUG row estado=%s detalle=%s opentasks=%d blocked=%d mailbox=%d workerDriver=%s workerAlive=%v workerHeartbeat=%v workerUpdated=%v progress=%v output=%v runtime=%s handle=%s controlOpen=%d ordersFailed=%d",
				row.EstadoOperativo,
				row.DetalleOperativo,
				row.OpenTasks,
				row.BlockedTasks,
				row.MailboxPending,
				row.WorkerDriver,
				row.WorkerAlive,
				row.WorkerHeartbeat,
				row.WorkerUpdatedAt,
				row.WorkerLastProgress,
				row.WorkerLastOutput,
				runtimeState,
				handleState,
				row.ControlOrdersOpen,
				row.OrdersFailed,
			)
		}
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia reiniciar un worker atascado, got=%d", procesadas)
	}
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: ptrString("Codex7")})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	var stopCount, startCount int
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch strings.TrimSpace(order.Tipo) {
		case "stop":
			stopCount++
		case "start":
			startCount++
		}
	}
	if stopCount == 0 || startCount == 0 {
		t.Fatalf("deberia encolar stop/start coordinado, stop=%d start=%d orders=%+v", stopCount, startCount, orders)
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchEscalaTareaActivaSiWorkerTMUXAtascadoReincide(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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
	if err := db.RegistrarAgente("Codex7", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ActivarAsignacion("Codex7", proyectoID, "frente"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex7",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	runtimeInst, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || runtimeInst == nil {
		t.Fatalf("runtime: %+v err=%v", runtimeInst, err)
	}

	traceDir := filepath.Join(tmp, "runtime", "codex7")
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	now := time.Now().UTC()
	staleAt := now.Add(-5 * time.Hour)
	manifestPath := filepath.Join(traceDir, "manifest.json")
	statusPath := filepath.Join(traceDir, "status.json")
	heartbeatPath := filepath.Join(traceDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex7","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex7-test","created_at":"`+now.Format(time.RFC3339Nano)+`","started_at":"`+now.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","alive":true,"updated_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+staleAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+staleAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now.Format(time.RFC3339Nano)+`","last_output_at":"`+staleAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+staleAt.Format(time.RFC3339Nano)+`"}`), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, err := json.Marshal(map[string]any{
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET estado='activo', metadata_json=? WHERE id=?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("activar handle: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE runtime_instances SET logical_state='esperando_io', process_state='running', updated_at=CURRENT_TIMESTAMP WHERE id=?`, runtimeInst.ID); err != nil {
		t.Fatalf("activar runtime: %v", err)
	}
	db.ResetRuntimeHandlesHotCache()

	for i := 0; i < 2; i++ {
		orderID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
			Agente:     "Codex7",
			ProyectoID: &proyectoID,
			Tipo:       "stop",
			PayloadJSON: fmt.Sprintf(`{"accion":"stop","motivo":"worker_atascado_%d","por":"orquesta"}`,
				i),
		})
		if err != nil {
			t.Fatalf("encolar stop reciente %d: %v", i, err)
		}
		if err := db.MarcarRuntimeOrderEstado(orderID, "completada", `{"ok":true}`, ""); err != nil {
			t.Fatalf("completar stop reciente %d: %v", i, err)
		}
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Worker tmux atascado reincidente",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("deberia escalar la tarea activa del worker reincidente, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != db.EstadoBloqueada {
		t.Fatalf("la tarea deberia quedar bloqueada, got=%s", tarea.Estado)
	}
	if !strings.Contains(strings.ToLower(tarea.Notas), "atasco persistente") {
		t.Fatalf("la tarea deberia dejar nota de atasco persistente: %s", tarea.Notas)
	}
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: ptrString("Codex7")})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	stopCount := 0
	for _, order := range orders {
		if order != nil && strings.TrimSpace(order.Tipo) == "stop" {
			stopCount++
		}
	}
	if stopCount != 2 {
		t.Fatalf("no deberia encolar mas reinicios al escalar, got stop=%d", stopCount)
	}
}

func TestProcesarWorkersAtascadosAutonomiaBatchReiniciaConMailboxPendienteAunqueNoHayaOpenTasks(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-stale-mailbox",
		Nombre:  "Orquestador Stale Mailbox",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := db.ActivarAsignacion("Codex2", proyectoID, "frente"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex2",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := db.UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert handle: %v", err)
	}
	handle, err := db.GetRuntimeHandleActivoAgenteProyecto("Codex2", &proyectoID)
	if err != nil || handle == nil {
		t.Fatalf("handle=%+v err=%v", handle, err)
	}
	now := time.Now().UTC()
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex2",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"pedir_intervencion","texto":"hay trabajo pendiente"}`,
	})
	if err != nil {
		t.Fatalf("mailbox: %v", err)
	}
	rows := []agentesapp.Row{{
		Agente:           &db.Agente{Nombre: "Codex2"},
		Asignacion:       &db.Asignacion{Agente: "Codex2", ProyectoID: proyectoID, Estado: db.AsignacionActiva},
		Handle:           handle,
		WorkerDriver:     "tmux_cli_session",
		WorkerState:      "running",
		WorkerAlive:      true,
		WorkerHeartbeat:  ptrTime(now),
		WorkerUpdatedAt:  ptrTime(now),
		MailboxPending:   1,
		BlockedTasks:     1,
		EstadoOperativo:  "atascado",
		DetalleOperativo: "worker sin progreso reciente",
	}}
	procesadas, err := procesarWorkersAtascadosAutonomiaBatch(rows, now)
	if err != nil {
		t.Fatalf("procesar workers atascados: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("procesadas=%d, want 1", procesadas)
	}
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: ptrString("Codex2")})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	stopCount := 0
	startCount := 0
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch strings.TrimSpace(order.Tipo) {
		case "stop":
			stopCount++
		case "start":
			startCount++
		}
	}
	if stopCount == 0 || startCount == 0 {
		t.Fatalf("deberia encolar restart coordinado, stop=%d start=%d orders=%+v", stopCount, startCount, orders)
	}
	msg, err := db.GetRuntimeMailbox(msgID)
	if err != nil || msg == nil {
		t.Fatalf("get mailbox: %+v err=%v", msg, err)
	}
	if msg.Estado != "pendiente" {
		t.Fatalf("la mailbox debe seguir pendiente para el restart durable, got=%s", msg.Estado)
	}
}

func TestExisteRuntimeOrderAutonomiaRecienteReconoceStopCoordinadoSinKindAutonomia(t *testing.T) {
	_ = prepararDBTemporalCmd(t)
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex7",
		ProyectoID:  &proyectoID,
		Tipo:        "stop",
		PayloadJSON: `{"accion":"stop","motivo":"worker_atascado","por":"orquesta"}`,
	}); err != nil {
		t.Fatalf("encolar stop: %v", err)
	}
	ok, err := existeRuntimeOrderAutonomiaReciente("Codex7", &proyectoID, "stop", "stop", time.Hour)
	if err != nil {
		t.Fatalf("existeRuntimeOrderAutonomiaReciente: %v", err)
	}
	if !ok {
		t.Fatalf("deberia detectar stop coordinado reciente aunque no tenga kind=autonomia")
	}
}

func TestDegradedTaskShouldInterveneRespetaAsumidaManualFueraDeFlota(t *testing.T) {
	tarea := &db.Tarea{
		ID:    77,
		Notas: "Asumida manualmente fuera de la flota automatica: el frente lo lleva Alberto.",
	}
	if degradedTaskShouldIntervene(tarea, time.Now().UTC()) {
		t.Fatalf("no deberia intervenir una tarea marcada como asumida manualmente")
	}
}

func TestProcesarAgentesDegradadosAutonomiaBatchBloqueaTareaActivaDeAgenteFueraDeOrquestacion(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Tarea huérfana por agente retirado",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET agente='Codex4', estado=? WHERE id=?`, db.EstadoEnProgreso, tareaID); err != nil {
		t.Fatalf("forzar tarea huérfana: %v", err)
	}

	procesadas, err := procesarAgentesDegradadosAutonomiaBatch()
	if err != nil {
		t.Fatalf("procesar agentes degradados: %v", err)
	}
	if procesadas != 1 {
		t.Fatalf("debería bloquear la tarea huérfana, got=%d", procesadas)
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != db.EstadoBloqueada {
		t.Fatalf("la tarea debería quedar bloqueada, got=%s", tarea.Estado)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex4" {
		t.Fatalf("la tarea debe conservar el último agente para trazabilidad: %+v", tarea)
	}
	bloqueos, err := db.ListarResumenBloqueos()
	if err != nil {
		t.Fatalf("listar bloqueos: %v", err)
	}
	encontrado := false
	for _, bloqueo := range bloqueos {
		if bloqueo.ID != tareaID {
			continue
		}
		if !strings.Contains(strings.ToLower(bloqueo.Motivo), "fuera de orquestación") {
			t.Fatalf("motivo de bloqueo inesperado: %+v", bloqueo)
		}
		encontrado = true
	}
	if !encontrado {
		t.Fatalf("debería existir un resumen de bloqueo para la tarea huérfana")
	}
}

func TestProcesarTareasActivasFueraDeOrquestacionBatchRespetaOperadorManualFueraDeFlota(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

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
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Tarea asumida por operador manual",
		Descripcion: "test",
		ProyectoID:  &proyectoID,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE tareas SET agente='alberto', estado=? WHERE id=?`, db.EstadoEnProgreso, tareaID); err != nil {
		t.Fatalf("forzar tarea manual: %v", err)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	tareasActivas := map[string][]*db.Tarea{
		"alberto": {tarea},
	}
	procesadas, err := procesarTareasActivasFueraDeOrquestacionBatch(tareasActivas, map[string]agentesapp.Row{})
	if err != nil {
		t.Fatalf("procesar tareas fuera de orquestacion: %v", err)
	}
	if procesadas != 0 {
		t.Fatalf("no deberia bloquear una tarea asumida por operador manual, got=%d", procesadas)
	}
	tarea, err = db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea tras procesado: %v", err)
	}
	if tarea.Estado != db.EstadoEnProgreso {
		t.Fatalf("la tarea manual debe seguir en progreso, got=%s", tarea.Estado)
	}
}

func TestConsumirRuntimeMailboxObsoletaPorWorkerSiProcedeConsumeNudgePrevioAStartedAt(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
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

	startedAt := time.Now().UTC()
	runDir := filepath.Join(tmp, "tmux-mailbox")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir runDir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	startedRaw := startedAt.Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex3","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex3-mailbox","tmux_pane_id":"%21","started_at":"`+startedRaw+`","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+startedRaw+`","alive":true,"last_output_at":"`+startedRaw+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+startedRaw+`","started_at":"`+startedRaw+`","last_output_at":"`+startedRaw+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codex3-mailbox",
		"tmux_pane_id":          "%21",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	res, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, capabilities_json, last_seen_at
	) VALUES (?,?,?,?,?,?,?, '{}', CURRENT_TIMESTAMP)`,
		"Codex3", proyectoID, "tmux", "session", "orq-codex3-mailbox", "activo", string(metaJSON))
	if err != nil {
		t.Fatalf("insert handle: %v", err)
	}
	handleID, _ := res.LastInsertId()
	handle, err := db.GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex3",
		ProyectoID:  &proyectoID,
		Kind:        "nudge",
		PayloadJSON: `{"texto":"sigue"}`,
	})
	if err != nil {
		t.Fatalf("enviar mailbox: %v", err)
	}
	oldCreatedAt := startedAt.Add(-10 * time.Minute)
	if _, err := db.DB.Exec(`UPDATE runtime_mailbox SET created_at=? WHERE id=?`, oldCreatedAt, msgID); err != nil {
		t.Fatalf("retroceder created_at: %v", err)
	}
	msgs, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: strPtr("Codex3")})
	if err != nil || len(msgs) == 0 {
		t.Fatalf("listar mailbox: %v len=%d", err, len(msgs))
	}

	consumida, err := consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msgs[0], handle, "session_resume", time.Now().UTC())
	if err != nil {
		t.Fatalf("consumir mailbox obsoleta: %v", err)
	}
	if !consumida {
		t.Fatalf("debería consumir el nudge obsoleto")
	}

	estado := "consumido"
	consumidos, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: strPtr("Codex3"), Estado: &estado})
	if err != nil {
		t.Fatalf("listar consumidos: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != msgID {
		t.Fatalf("el mailbox debería quedar consumido, got=%+v", consumidos)
	}
}

func TestConsumirRuntimeMailboxObsoletaPorWorkerSiProcedeConsumeAutonomiaDeTareaYaNoActiva(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("claude1", "programador"); err != nil {
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

	runDir := filepath.Join(tmp, "tmux-mailbox-obsoleta-tarea")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir runDir: %v", err)
	}
	startedAt := time.Now().UTC().Add(-30 * time.Minute)
	startedRaw := startedAt.Format(time.RFC3339Nano)
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"claude1","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-claude1-obsoleta","tmux_pane_id":"%33","started_at":"`+startedRaw+`","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+startedRaw+`","alive":true,"last_output_at":"`+startedRaw+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+startedRaw+`","started_at":"`+startedRaw+`","last_output_at":"`+startedRaw+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-claude1-obsoleta",
		"tmux_pane_id":          "%33",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	res, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, capabilities_json, last_seen_at
	) VALUES (?,?,?,?,?,?,?, '{}', CURRENT_TIMESTAMP)`,
		"claude1", proyectoID, "tmux", "session", "orq-claude1-obsoleta/%33", "activo", string(metaJSON))
	if err != nil {
		t.Fatalf("insert handle: %v", err)
	}
	handleID, _ := res.LastInsertId()
	handle, err := db.GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}

	tareaViejaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Micro-refactorización cíclica del control plane",
		Descripcion: "Frente general del control plane.",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea vieja: %v", err)
	}
	if err := db.TomarTarea(tareaViejaID, "claude1"); err != nil {
		t.Fatalf("tomar tarea vieja: %v", err)
	}

	tareaActualID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Micro-refactorización cíclica del control plane: runtime mailbox/session_resume",
		Descripcion: "Frente premium acotado.\nWrite-set preferente: cmd/controlplane_support.go, cmd/controlplane_support_test.go\nTests minimos: go test ./cmd -run 'TestConsumirRuntimeMailboxObsoletaPorWorkerSiProcedeConsumeAutonomiaDeTareaYaNoActiva$'",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea actual: %v", err)
	}
	if err := db.TomarTarea(tareaActualID, "claude1"); err != nil {
		t.Fatalf("tomar tarea actual: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente: "server",
		ToAgente:   "claude1",
		ProyectoID: &proyectoID,
		Kind:       "autonomia",
		PayloadJSON: fmt.Sprintf(
			`{"accion":"continuar_trabajo","kind":"autonomia","tarea_id":%d,"texto":"sigue"}`,
			tareaViejaID,
		),
	})
	if err != nil {
		t.Fatalf("enviar mailbox: %v", err)
	}
	msgs, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: strPtr("claude1"), Estado: strPtr("pendiente")})
	if err != nil || len(msgs) == 0 {
		t.Fatalf("listar mailbox: %v len=%d", err, len(msgs))
	}

	consumida, err := consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msgs[0], handle, "bootstrap_tmux", time.Now().UTC())
	if err != nil {
		t.Fatalf("consumir mailbox obsoleta por tarea: %v", err)
	}
	if !consumida {
		t.Fatalf("deberia consumir la autonomia que apunta a una tarea ya no activa")
	}

	estado := "consumido"
	consumidos, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: strPtr("claude1"), Estado: &estado})
	if err != nil {
		t.Fatalf("listar consumidos: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != msgID {
		t.Fatalf("la mailbox deberia quedar consumida, got=%+v", consumidos)
	}
}

func TestConsumirRuntimeMailboxObsoletaPorWorkerSiProcedeConsumeAutonomiaConProgresoPosterior(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("claude1", "programador"); err != nil {
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

	runDir := filepath.Join(tmp, "tmux-mailbox-progress")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir runDir: %v", err)
	}
	now := time.Now().UTC()
	startedAt := now.Add(-30 * time.Minute)
	progressAt := now.Add(-1 * time.Minute)
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"claude1","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-claude1-progress","tmux_pane_id":"%34","started_at":"`+startedAt.Format(time.RFC3339Nano)+`","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+progressAt.Format(time.RFC3339Nano)+`","alive":true,"last_output_at":"`+progressAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+progressAt.Format(time.RFC3339Nano)+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+progressAt.Format(time.RFC3339Nano)+`","started_at":"`+startedAt.Format(time.RFC3339Nano)+`","last_output_at":"`+progressAt.Format(time.RFC3339Nano)+`","last_progress_at":"`+progressAt.Format(time.RFC3339Nano)+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-claude1-progress",
		"tmux_pane_id":          "%34",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	res, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, capabilities_json, last_seen_at
	) VALUES (?,?,?,?,?,?,?, '{}', CURRENT_TIMESTAMP)`,
		"claude1", proyectoID, "tmux", "session", "orq-claude1-progress/%34", "activo", string(metaJSON))
	if err != nil {
		t.Fatalf("insert handle: %v", err)
	}
	handleID, _ := res.LastInsertId()
	handle, err := db.GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}

	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Micro-refactorización cíclica del control plane: runtime mailbox/session_resume",
		Descripcion: "Frente premium acotado.\nWrite-set preferente: cmd/controlplane_support.go, cmd/controlplane_support_test.go\nTests minimos: go test ./cmd -run 'TestConsumirRuntimeMailboxObsoletaPorWorkerSiProcedeConsumeAutonomiaConProgresoPosterior$'",
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "claude1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente: "server",
		ToAgente:   "claude1",
		ProyectoID: &proyectoID,
		Kind:       "autonomia",
		PayloadJSON: fmt.Sprintf(
			`{"accion":"continuar_trabajo","kind":"autonomia","tarea_id":%d,"texto":"sigue"}`,
			tareaID,
		),
	})
	if err != nil {
		t.Fatalf("enviar mailbox: %v", err)
	}
	msgCreatedAt := now.Add(-5 * time.Minute)
	if _, err := db.DB.Exec(`UPDATE runtime_mailbox SET created_at=? WHERE id=?`, msgCreatedAt, msgID); err != nil {
		t.Fatalf("retroceder created_at: %v", err)
	}
	msgs, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: strPtr("claude1"), Estado: strPtr("pendiente")})
	if err != nil || len(msgs) == 0 {
		t.Fatalf("listar mailbox: %v len=%d", err, len(msgs))
	}

	consumida, err := consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msgs[0], handle, "bootstrap_tmux", time.Now().UTC())
	if err != nil {
		t.Fatalf("consumir mailbox por progreso: %v", err)
	}
	if !consumida {
		t.Fatalf("deberia consumir la autonomia cuando ya hubo progreso posterior en la misma tarea")
	}

	estado := "consumido"
	consumidos, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: strPtr("claude1"), Estado: &estado})
	if err != nil {
		t.Fatalf("listar consumidos: %v", err)
	}
	if len(consumidos) != 1 || consumidos[0].ID != msgID {
		t.Fatalf("la mailbox deberia quedar consumida, got=%+v", consumidos)
	}
}

func TestConsumirRuntimeMailboxObsoletaPorWorkerSiProcedeNoConsumeInstructionDurable(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
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

	startedAt := time.Now().UTC()
	runDir := filepath.Join(tmp, "tmux-mailbox-instruction")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir runDir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	startedRaw := startedAt.Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex3","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex3-mailbox2","tmux_pane_id":"%22","started_at":"`+startedRaw+`","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+startedRaw+`","alive":true,"last_output_at":"`+startedRaw+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+startedRaw+`","started_at":"`+startedRaw+`","last_output_at":"`+startedRaw+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codex3-mailbox2",
		"tmux_pane_id":          "%22",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	res, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, capabilities_json, last_seen_at
	) VALUES (?,?,?,?,?,?,?, '{}', CURRENT_TIMESTAMP)`,
		"Codex3", proyectoID, "tmux", "session", "orq-codex3-mailbox2", "activo", string(metaJSON))
	if err != nil {
		t.Fatalf("insert handle: %v", err)
	}
	handleID, _ := res.LastInsertId()
	handle, err := db.GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}

	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "Codex3",
		ProyectoID:  &proyectoID,
		Kind:        "instruction",
		PayloadJSON: `{"texto":"haz esto"}`,
	})
	if err != nil {
		t.Fatalf("enviar mailbox: %v", err)
	}
	oldCreatedAt := startedAt.Add(-10 * time.Minute)
	if _, err := db.DB.Exec(`UPDATE runtime_mailbox SET created_at=? WHERE id=?`, oldCreatedAt, msgID); err != nil {
		t.Fatalf("retroceder created_at: %v", err)
	}
	msgs, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: strPtr("Codex3")})
	if err != nil || len(msgs) == 0 {
		t.Fatalf("listar mailbox: %v len=%d", err, len(msgs))
	}

	consumida, err := consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msgs[0], handle, "session_resume", time.Now().UTC())
	if err != nil {
		t.Fatalf("evaluar mailbox durable: %v", err)
	}
	if consumida {
		t.Fatalf("una instruction durable no debería consumirse automáticamente")
	}

	estado := "pendiente"
	pendientes, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: strPtr("Codex3"), Estado: &estado})
	if err != nil {
		t.Fatalf("listar pendientes: %v", err)
	}
	if len(pendientes) != 1 || pendientes[0].ID != msgID {
		t.Fatalf("la instruction debería seguir pendiente, got=%+v", pendientes)
	}
}

func TestConsumirRuntimeMailboxObsoletaPorWorkerSiProcedeNoConsumeMailboxConRuntimeOrderViva(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
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

	startedAt := time.Now().UTC()
	runDir := filepath.Join(tmp, "tmux-mailbox-linked-order")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir runDir: %v", err)
	}
	manifestPath := filepath.Join(runDir, "manifest.json")
	statusPath := filepath.Join(runDir, "status.json")
	heartbeatPath := filepath.Join(runDir, "heartbeat.json")
	startedRaw := startedAt.Format(time.RFC3339Nano)
	if err := os.WriteFile(manifestPath, []byte(`{"version":1,"agent":"Codex3","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex3-mailbox3","tmux_pane_id":"%23","started_at":"`+startedRaw+`","status_path":"`+statusPath+`","heartbeat_path":"`+heartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(statusPath, []byte(`{"state":"running","updated_at":"`+startedRaw+`","alive":true,"last_output_at":"`+startedRaw+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status: %v", err)
	}
	if err := os.WriteFile(heartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+startedRaw+`","started_at":"`+startedRaw+`","last_output_at":"`+startedRaw+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codex3-mailbox3",
		"tmux_pane_id":          "%23",
		"worker_manifest_path":  manifestPath,
		"worker_status_path":    statusPath,
		"worker_heartbeat_path": heartbeatPath,
	})
	res, err := db.DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, capabilities_json, last_seen_at
	) VALUES (?,?,?,?,?,?,?, '{}', CURRENT_TIMESTAMP)`,
		"Codex3", proyectoID, "tmux", "session", "orq-codex3-mailbox3/%23", "activo", string(metaJSON))
	if err != nil {
		t.Fatalf("insert handle: %v", err)
	}
	handleID, _ := res.LastInsertId()
	handle, err := db.GetRuntimeHandle(handleID)
	if err != nil {
		t.Fatalf("get handle: %v", err)
	}

	runtimeOrderID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex3",
		ProyectoID:  &proyectoID,
		Tipo:        "send_instruction",
		PayloadJSON: `{"to_agente":"Codex3","texto":"sigue","mailbox_kind":"nudge"}`,
	})
	if err != nil {
		t.Fatalf("encolar runtime order: %v", err)
	}
	msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
		FromAgente:     "server",
		ToAgente:       "Codex3",
		ProyectoID:     &proyectoID,
		RuntimeOrderID: &runtimeOrderID,
		Kind:           "nudge",
		PayloadJSON:    `{"texto":"sigue"}`,
	})
	if err != nil {
		t.Fatalf("enviar mailbox: %v", err)
	}
	oldCreatedAt := startedAt.Add(-10 * time.Minute)
	if _, err := db.DB.Exec(`UPDATE runtime_mailbox SET created_at=? WHERE id=?`, oldCreatedAt, msgID); err != nil {
		t.Fatalf("retroceder created_at: %v", err)
	}
	msgs, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: strPtr("Codex3")})
	if err != nil || len(msgs) == 0 {
		t.Fatalf("listar mailbox: %v len=%d", err, len(msgs))
	}

	consumida, err := consumirRuntimeMailboxObsoletaPorWorkerSiProcede(msgs[0], handle, "session_resume", time.Now().UTC())
	if err != nil {
		t.Fatalf("evaluar mailbox ligada a orden: %v", err)
	}
	if consumida {
		t.Fatalf("una mailbox ligada a runtime_order viva no deberia consumirse fuera de la orden")
	}

	estadoPend := "pendiente"
	pendientes, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: strPtr("Codex3"), Estado: &estadoPend})
	if err != nil {
		t.Fatalf("listar pendientes: %v", err)
	}
	if len(pendientes) != 1 || pendientes[0].ID != msgID {
		t.Fatalf("la mailbox ligada a orden viva deberia seguir pendiente, got=%+v", pendientes)
	}
	order, err := db.GetRuntimeOrder(runtimeOrderID)
	if err != nil {
		t.Fatalf("get runtime order: %v", err)
	}
	if order == nil || order.Estado != "pendiente" {
		t.Fatalf("la runtime order deberia seguir viva, got=%+v", order)
	}
}
