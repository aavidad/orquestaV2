package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func registrarPresupuestoCritico(t *testing.T, sesionID int64, remainingSeconds int64, checkedAt time.Time) {
	t.Helper()
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:         sesionID,
		WindowKind:       "5h",
		RemainingSeconds: &remainingSeconds,
		BudgetSource:     "manual",
		CheckedAt:        checkedAt,
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}
}

// setHeartbeatStale fuerza el heartbeat de una sesión a un tiempo muy antiguo
// para que sea detectada como candidata a handoff.
func setHeartbeatStale(t *testing.T, sesionID int64) {
	t.Helper()
	old := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	if _, err := DB.Exec(`UPDATE sesiones SET heartbeat_at=? WHERE id=?`, old, sesionID); err != nil {
		t.Fatalf("setHeartbeatStale: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances
		SET last_event_at=?,
		    last_heartbeat_at=?,
		    updated_at=?
		WHERE sesion_id=?`, old, old, old, sesionID); err != nil {
		t.Fatalf("setHeartbeatStale runtime: %v", err)
	}
}

// prepararAgenteConTareaEnProgreso crea un agente con sesión activa y una tarea en progreso.
func prepararAgenteConTareaEnProgreso(t *testing.T, nombre string) (sesionID int64, tareaID int64) {
	t.Helper()
	if err := RegistrarAgente(nombre, "programador"); err != nil {
		t.Fatalf("RegistrarAgente %s: %v", nombre, err)
	}
	sesionID, err := IniciarSesion(nombre)
	if err != nil {
		t.Fatalf("IniciarSesion %s: %v", nombre, err)
	}
	tareaID, err = CrearTarea(&Tarea{
		Titulo:    "Tarea de " + nombre,
		Modulo:    "orquestador",
		Prioridad: PrioridadMedia,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea %s: %v", nombre, err)
	}
	if err := TomarTarea(tareaID, nombre); err != nil {
		t.Fatalf("TomarTarea %s: %v", nombre, err)
	}
	if err := IniciarTarea(tareaID, nombre); err != nil {
		t.Fatalf("IniciarTarea %s: %v", nombre, err)
	}
	return sesionID, tareaID
}

func TestDetectarAgentesAgotadosDevuelveAgentesConHeartbeatAntiguo(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, _ := prepararAgenteConTareaEnProgreso(t, "Codex1")
	setHeartbeatStale(t, sesionID)

	candidatos, err := DetectarAgentesAgotados()
	if err != nil {
		t.Fatalf("DetectarAgentesAgotados: %v", err)
	}
	if len(candidatos) != 1 {
		t.Fatalf("esperaba 1 candidato, got=%d", len(candidatos))
	}
	if candidatos[0].Agente != "Codex1" {
		t.Fatalf("agente inesperado: %s", candidatos[0].Agente)
	}
	if candidatos[0].TareaID == nil {
		t.Fatalf("tarea_id nula en candidato")
	}
	if candidatos[0].SesionID == nil {
		t.Fatalf("sesion_id nula en candidato")
	}
	if candidatos[0].Disparador != "watchdog" || !candidatos[0].RequiereSondeo {
		t.Fatalf("candidato watchdog inesperado: %+v", candidatos[0])
	}
}

func TestDetectarAgentesAgotadosSinHandleActivoNoRequiereSondeo(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, _ := prepararAgenteConTareaEnProgreso(t, "Codex1")
	setHeartbeatStale(t, sesionID)
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE agente=?`, "Codex1"); err != nil {
		t.Fatalf("cerrar handles origen: %v", err)
	}

	candidatos, err := DetectarAgentesAgotados()
	if err != nil {
		t.Fatalf("DetectarAgentesAgotados: %v", err)
	}
	if len(candidatos) != 1 {
		t.Fatalf("esperaba 1 candidato, got=%d", len(candidatos))
	}
	if candidatos[0].Disparador != "watchdog" || candidatos[0].RequiereSondeo {
		t.Fatalf("candidato watchdog sin handle inesperado: %+v", candidatos[0])
	}
}

func TestDetectarAgentesAgotadosIgnoraHeartbeatReciente(t *testing.T) {
	prepararDBTemporal(t)

	// heartbeat_at por defecto es CURRENT_TIMESTAMP → reciente
	_, _ = prepararAgenteConTareaEnProgreso(t, "Codex1")

	candidatos, err := DetectarAgentesAgotados()
	if err != nil {
		t.Fatalf("DetectarAgentesAgotados: %v", err)
	}
	if len(candidatos) != 0 {
		t.Fatalf("no esperaba candidatos con heartbeat reciente, got=%d", len(candidatos))
	}
}

func TestDetectarAgentesAgotadosIgnoraHeartbeatSesionStaleSiRuntimeSigueActivoReciente(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, _ := prepararAgenteConTareaEnProgreso(t, "Codex1")
	setHeartbeatStale(t, sesionID)

	handle, err := GetRuntimeHandleActivoAgente("Codex1")
	if err != nil {
		t.Fatalf("GetRuntimeHandleActivoAgente: %v", err)
	}
	if handle == nil || handle.RuntimeID == nil {
		t.Fatalf("handle activo inesperado: %+v", handle)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances
		SET logical_state='esperando_io',
		    process_state='running',
		    last_event_at=CURRENT_TIMESTAMP,
		    last_heartbeat_at=CURRENT_TIMESTAMP,
		    updated_at=CURRENT_TIMESTAMP
		WHERE id=?`, *handle.RuntimeID); err != nil {
		t.Fatalf("actualizar runtime activo: %v", err)
	}

	candidatos, err := DetectarAgentesAgotados()
	if err != nil {
		t.Fatalf("DetectarAgentesAgotados: %v", err)
	}
	if len(candidatos) != 0 {
		t.Fatalf("no esperaba candidatos si el runtime sigue activo recientemente: %+v", candidatos)
	}
}

func TestDetectarAgentesAgotadosIgnoraHeartbeatSesionStaleSiRuntimeSeResuelvePorSesionID(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, _ := prepararAgenteConTareaEnProgreso(t, "Codex1")
	setHeartbeatStale(t, sesionID)

	handle, err := GetRuntimeHandleActivoAgente("Codex1")
	if err != nil {
		t.Fatalf("GetRuntimeHandleActivoAgente: %v", err)
	}
	if handle == nil || handle.SesionID == nil {
		t.Fatalf("handle activo inesperado: %+v", handle)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET runtime_id=NULL WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("desenlazar runtime_id del handle: %v", err)
	}
	runtimeHandleHotReset()
	if _, err := DB.Exec(`UPDATE runtime_instances
		SET logical_state='esperando_io',
		    process_state='running',
		    last_event_at=CURRENT_TIMESTAMP,
		    last_heartbeat_at=CURRENT_TIMESTAMP,
		    updated_at=CURRENT_TIMESTAMP
		WHERE sesion_id=?`, *handle.SesionID); err != nil {
		t.Fatalf("actualizar runtime activo por sesion: %v", err)
	}

	candidatos, err := DetectarAgentesAgotados()
	if err != nil {
		t.Fatalf("DetectarAgentesAgotados: %v", err)
	}
	if len(candidatos) != 0 {
		t.Fatalf("no esperaba candidatos si el runtime se puede resolver por sesion_id: %+v", candidatos)
	}
}

func TestDetectarAgentesAgotadosExcluyeConHandoffPendiente(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, tareaID := prepararAgenteConTareaEnProgreso(t, "Codex1")
	setHeartbeatStale(t, sesionID)

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if _, err := IniciarSesion("Codex2"); err != nil {
		t.Fatalf("IniciarSesion Codex2: %v", err)
	}
	// Crear una orden de handoff pendiente para Codex1
	if _, err := CrearHandoffAgenteVivo("Codex1", "Codex2", &tareaID, "test", "continuar", ""); err != nil {
		t.Fatalf("CrearHandoffAgenteVivo: %v", err)
	}

	candidatos, err := DetectarAgentesAgotados()
	if err != nil {
		t.Fatalf("DetectarAgentesAgotados: %v", err)
	}
	// Codex1 ya tiene handoff pendiente → no debe aparecer
	for _, c := range candidatos {
		if c.Agente == "Codex1" {
			t.Fatalf("Codex1 no debería aparecer: ya tiene handoff pendiente")
		}
	}
}

func TestSeleccionarTareaHandoffAgenteCanonicalizaAliasCodex(t *testing.T) {
	prepararDBTemporal(t)

	for _, nombre := range []string{"Codex41", "codex41"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Tarea canonical handoff",
		Descripcion: "debe resolverse por alias",
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "test",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex41"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex41"); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}

	tarea, err := seleccionarTareaHandoffAgente("codex41", nil)
	if err != nil {
		t.Fatalf("seleccionarTareaHandoffAgente: %v", err)
	}
	if tarea == nil || tarea.ID != tareaID || tarea.Agente == nil || *tarea.Agente != "Codex41" {
		t.Fatalf("tarea inesperada: %+v", tarea)
	}
}

func TestExisteSondeoWatchdogRecienteCanonicalizaAliasCodex(t *testing.T) {
	prepararDBTemporal(t)

	for _, nombre := range []string{"Codex42", "codex42"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo-watchdog-canonical",
		Nombre:  "Demo Watchdog Canonical",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex42",
		ProyectoID:  &proyectoID,
		Tipo:        "sync_status",
		Estado:      "pendiente",
		PayloadJSON: `{"kind":"watchdog"}`,
	}); err != nil {
		t.Fatalf("EncolarRuntimeOrder: %v", err)
	}

	ok, err := existeSondeoWatchdogReciente("codex42", &proyectoID)
	if err != nil {
		t.Fatalf("existeSondeoWatchdogReciente: %v", err)
	}
	if !ok {
		t.Fatalf("deberia detectar sondeo watchdog reciente por alias")
	}
}

func TestGuardarCheckpointHandoffCanonicalizaAliasCodex(t *testing.T) {
	prepararDBTemporal(t)

	for _, nombre := range []string{"Codex44", "codex44"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	sesionID, err := IniciarSesion("Codex44")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}

	id, err := GuardarCheckpointHandoff(&HandoffCandidato{
		Agente:     "codex44",
		SesionID:   &sesionID,
		Motivo:     "test canonical handoff checkpoint",
		Disparador: "watchdog",
	}, "")
	if err != nil {
		t.Fatalf("GuardarCheckpointHandoff: %v", err)
	}
	cp, err := GetRuntimeCheckpoint(id)
	if err != nil {
		t.Fatalf("GetRuntimeCheckpoint: %v", err)
	}
	if cp == nil || cp.Agente != "Codex44" {
		t.Fatalf("checkpoint inesperado: %+v", cp)
	}
}

func TestDetectarAgentesAgotadosIncluyePresupuestoCriticoConHeartbeatReciente(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, _ := prepararAgenteConTareaEnProgreso(t, "Codex1")
	registrarPresupuestoCritico(t, sesionID, 15, time.Now().UTC())

	candidatos, err := DetectarAgentesAgotados()
	if err != nil {
		t.Fatalf("DetectarAgentesAgotados: %v", err)
	}
	if len(candidatos) != 1 {
		t.Fatalf("esperaba 1 candidato por presupuesto, got=%d", len(candidatos))
	}
	if candidatos[0].Disparador != "presupuesto" || candidatos[0].RequiereSondeo {
		t.Fatalf("candidato presupuesto inesperado: %+v", candidatos[0])
	}
}

func TestDetectarAgentesAgotadosIgnoraPresupuestoObsoletoConHeartbeatReciente(t *testing.T) {
	prepararDBTemporal(t)

	if err := ConfigSet("pool_budget_snapshot_max_age_seconds", "60"); err != nil {
		t.Fatalf("ConfigSet freshness: %v", err)
	}
	sesionID, _ := prepararAgenteConTareaEnProgreso(t, "Codex1")
	registrarPresupuestoCritico(t, sesionID, 15, time.Now().UTC().Add(-10*time.Minute))

	candidatos, err := DetectarAgentesAgotados()
	if err != nil {
		t.Fatalf("DetectarAgentesAgotados: %v", err)
	}
	if len(candidatos) != 0 {
		t.Fatalf("no esperaba candidato con presupuesto obsoleto y heartbeat reciente: %+v", candidatos)
	}
}

func TestDetectarAgentesAgotadosPrefiereTareaDelProyectoActivo(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto-a",
		Nombre:  "Proyecto A",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto-b",
		Nombre:  "Proyecto B",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto B: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoA,
		CWD:         t.TempDir(),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto: %v", err)
	}

	tareaB, err := CrearTarea(&Tarea{
		Titulo:     "Tarea en progreso B",
		Modulo:     "orquestador",
		Prioridad:  PrioridadMedia,
		CreadoPor:  "alberto",
		ProyectoID: &proyectoB,
	})
	if err != nil {
		t.Fatalf("CrearTarea B: %v", err)
	}
	if err := TomarTarea(tareaB, "Codex1"); err != nil {
		t.Fatalf("TomarTarea B: %v", err)
	}
	if err := IniciarTarea(tareaB, "Codex1"); err != nil {
		t.Fatalf("IniciarTarea B: %v", err)
	}

	tareaA, err := CrearTarea(&Tarea{
		Titulo:     "Tarea en progreso A",
		Modulo:     "orquestador",
		Prioridad:  PrioridadMedia,
		CreadoPor:  "alberto",
		ProyectoID: &proyectoA,
	})
	if err != nil {
		t.Fatalf("CrearTarea A: %v", err)
	}
	if err := TomarTarea(tareaA, "Codex1"); err != nil {
		t.Fatalf("TomarTarea A: %v", err)
	}
	if err := IniciarTarea(tareaA, "Codex1"); err != nil {
		t.Fatalf("IniciarTarea A: %v", err)
	}

	setHeartbeatStale(t, sesion.ID)

	candidatos, err := DetectarAgentesAgotados()
	if err != nil {
		t.Fatalf("DetectarAgentesAgotados: %v", err)
	}
	if len(candidatos) != 1 {
		t.Fatalf("esperaba 1 candidato, got=%d", len(candidatos))
	}
	if candidatos[0].TareaID == nil || *candidatos[0].TareaID != tareaA {
		t.Fatalf("deberia elegir la tarea del proyecto activo de la sesion: %+v", candidatos[0])
	}
	if candidatos[0].ProyectoID == nil || *candidatos[0].ProyectoID != proyectoA {
		t.Fatalf("proyecto candidato inesperado: %+v", candidatos[0])
	}
}

func TestDetectarAgentesAgotadosUsaHandleDelProyectoActivoAunqueHayaOtroMasReciente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto-a",
		Nombre:  "Proyecto A",
		RutaAbs: filepath.Join(tmp, "proyecto-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto-b",
		Nombre:  "Proyecto B",
		RutaAbs: filepath.Join(tmp, "proyecto-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto B: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoA,
		CWD:         filepath.Join(tmp, "cwd-a"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto: %v", err)
	}
	if err := UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("UpsertRuntimeHandleDesdeSesion: %v", err)
	}
	tareaA, err := CrearTarea(&Tarea{
		Titulo:     "Tarea en progreso A",
		Modulo:     "orquestador",
		Prioridad:  PrioridadMedia,
		CreadoPor:  "alberto",
		ProyectoID: &proyectoA,
	})
	if err != nil {
		t.Fatalf("CrearTarea A: %v", err)
	}
	if err := TomarTarea(tareaA, "Codex1"); err != nil {
		t.Fatalf("TomarTarea A: %v", err)
	}
	if err := IniciarTarea(tareaA, "Codex1"); err != nil {
		t.Fatalf("IniciarTarea A: %v", err)
	}
	setHeartbeatStale(t, sesion.ID)

	handleA, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil {
		t.Fatalf("GetRuntimeHandleBySesionID A: %v", err)
	}
	if handleA == nil {
		t.Fatalf("faltaba handle del proyecto activo")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	pid := int64(os.Getpid())

	tmuxADir := filepath.Join(tmp, "tmux-a")
	if err := os.MkdirAll(tmuxADir, 0o755); err != nil {
		t.Fatalf("mkdir tmux A: %v", err)
	}
	tmuxAManifestPath := filepath.Join(tmuxADir, "manifest.json")
	tmuxAStatusPath := filepath.Join(tmuxADir, "status.json")
	tmuxAHeartbeatPath := filepath.Join(tmuxADir, "heartbeat.json")
	if err := os.WriteFile(tmuxAManifestPath, []byte(`{"version":1,"agent":"Codex1","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex1-a","tmux_pane_id":"%21","status_path":"`+tmuxAStatusPath+`","heartbeat_path":"`+tmuxAHeartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest A: %v", err)
	}
	if err := os.WriteFile(tmuxAStatusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+jsonNumber(pid)+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status A: %v", err)
	}
	if err := os.WriteFile(tmuxAHeartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+jsonNumber(pid)+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat A: %v", err)
	}
	runtimeAID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		ProyectoID:   &proyectoA,
		SesionID:     &sesion.ID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeInstance A: %v", err)
	}
	metaAJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codex1-a",
		"tmux_pane_id":          "%21",
		"worker_manifest_path":  tmuxAManifestPath,
		"worker_status_path":    tmuxAStatusPath,
		"worker_heartbeat_path": tmuxAHeartbeatPath,
		"rendered_command":      "codex-perfil Codex1",
	})
	if _, err := DB.Exec(`UPDATE runtime_handles
		SET runtime_id=?, transporte='tmux', handle_kind='process', handle_ref='orq-codex1-a',
		    metadata_json=?, last_seen_at=CURRENT_TIMESTAMP
		WHERE id=?`,
		runtimeAID, string(metaAJSON), handleA.ID,
	); err != nil {
		t.Fatalf("actualizar handle A a tmux: %v", err)
	}

	runtimeBID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		ProyectoID:   &proyectoB,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeInstance B: %v", err)
	}
	tmuxBDir := filepath.Join(tmp, "tmux-b")
	if err := os.MkdirAll(tmuxBDir, 0o755); err != nil {
		t.Fatalf("mkdir tmux B: %v", err)
	}
	tmuxBManifestPath := filepath.Join(tmuxBDir, "manifest.json")
	tmuxBStatusPath := filepath.Join(tmuxBDir, "status.json")
	tmuxBHeartbeatPath := filepath.Join(tmuxBDir, "heartbeat.json")
	if err := os.WriteFile(tmuxBManifestPath, []byte(`{"version":1,"agent":"Codex1","driver":"tmux_cli_session","transport":"tmux","tmux_session":"orq-codex1-b","tmux_pane_id":"%22","status_path":"`+tmuxBStatusPath+`","heartbeat_path":"`+tmuxBHeartbeatPath+`"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write manifest B: %v", err)
	}
	if err := os.WriteFile(tmuxBStatusPath, []byte(`{"state":"running","updated_at":"`+now+`","alive":true,"child_pid":`+jsonNumber(pid)+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write status B: %v", err)
	}
	if err := os.WriteFile(tmuxBHeartbeatPath, []byte(`{"alive":true,"heartbeat_at":"`+now+`","started_at":"`+now+`","child_pid":`+jsonNumber(pid)+`}`+"\n"), 0o600); err != nil {
		t.Fatalf("write heartbeat B: %v", err)
	}
	metaBJSON, _ := json.Marshal(map[string]any{
		"driver":                "tmux_cli_session",
		"tmux_session":          "orq-codex1-b",
		"tmux_pane_id":          "%22",
		"worker_manifest_path":  tmuxBManifestPath,
		"worker_status_path":    tmuxBStatusPath,
		"worker_heartbeat_path": tmuxBHeartbeatPath,
		"rendered_command":      "codex-perfil Codex1",
	})
	handleBID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?, CURRENT_TIMESTAMP)`,
		"Codex1", proyectoB, runtimeBID, "tmux", "process", "orq-codex1-b", string(metaBJSON))
	if _, err := DB.Exec(`UPDATE runtime_handles SET last_seen_at = datetime('now','+1 minute') WHERE id = ?`, handleBID); err != nil {
		t.Fatalf("actualizar last_seen_at B: %v", err)
	}
	runtimeHandleHotReset()

	candidatos, err := DetectarAgentesAgotados()
	if err != nil {
		t.Fatalf("DetectarAgentesAgotados: %v", err)
	}
	if len(candidatos) != 1 {
		t.Fatalf("esperaba 1 candidato, got=%d", len(candidatos))
	}
	if candidatos[0].HandleID == nil || *candidatos[0].HandleID != handleA.ID {
		t.Fatalf("deberia usar el handle del proyecto activo, no otro más reciente: %+v", candidatos[0])
	}
}

func TestSeleccionarAgenteReemplazoDevuelveLibre(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	// Codex2 libre (activo=0, habilitado=1 por defecto)

	nombre, err := SeleccionarAgenteReemplazo("Codex1", nil)
	if err != nil {
		t.Fatalf("SeleccionarAgenteReemplazo: %v", err)
	}
	if nombre != "Codex2" {
		t.Fatalf("agente inesperado: %s", nombre)
	}
}

func TestSeleccionarAgenteReemplazoSinDisponibles(t *testing.T) {
	prepararDBTemporal(t)

	// Deshabilitar todos los agentes programadores pre-seeded
	if _, err := DB.Exec(`UPDATE agentes SET habilitado=0 WHERE rol='programador'`); err != nil {
		t.Fatalf("deshabilitar agentes: %v", err)
	}
	if err := RegistrarAgente("SoloAgente", "programador"); err != nil {
		t.Fatalf("registrar SoloAgente: %v", err)
	}

	_, err := SeleccionarAgenteReemplazo("SoloAgente", nil)
	if err == nil {
		t.Fatalf("se esperaba error por no haber reemplazo disponible")
	}
}

func TestSeleccionarAgenteReemplazoExcluyeAgentesOcupados(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}

	nombre, err := SeleccionarAgenteReemplazo("Codex1", []string{"Codex2"})
	if err != nil {
		t.Fatalf("SeleccionarAgenteReemplazo: %v", err)
	}
	if nombre != "Codex3" {
		t.Fatalf("agente inesperado: %s", nombre)
	}
}

func TestSeleccionarAgenteReemplazoParaProyectoExigeCompatibilidadGobernanza(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "handoff-gobernanza",
		Nombre:  "handoff-gobernanza",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	reglaID, err := UpsertRegla(&Regla{
		TipoAgente:  "programador",
		Categoria:   "arquitectura",
		Titulo:      "server-first-handoff",
		Descripcion: "usar API",
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("UpsertRegla: %v", err)
	}
	if _, err := GuardarGovernanceOverride("test", &GovernanceOverride{
		ScopeTipo:  GovernanceScopeAgente,
		ScopeRef:   "Codex2",
		Entidad:    GovernanceEntityRegla,
		EntidadID:  reglaID,
		Accion:     GovernanceActionDisable,
		TipoAgente: "programador",
	}); err != nil {
		t.Fatalf("GuardarGovernanceOverride: %v", err)
	}

	nombre, err := SeleccionarAgenteReemplazoParaProyecto("Codex1", &proyectoID, nil)
	if err != nil {
		t.Fatalf("SeleccionarAgenteReemplazoParaProyecto: %v", err)
	}
	if nombre != "Codex3" {
		t.Fatalf("deberia saltar Codex2 por gobernanza incompatible, got=%s", nombre)
	}
}

func TestGuardarCheckpointHandoffCreaRegistro(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, tareaID := prepararAgenteConTareaEnProgreso(t, "Codex1")
	c := &HandoffCandidato{
		Agente:      "Codex1",
		TareaID:     &tareaID,
		SesionID:    &sesionID,
		Inactividad: 2 * time.Hour,
		Motivo:      "heartbeat hace 120 min (umbral: 30 min)",
	}

	checkpointID, err := GuardarCheckpointHandoff(c, "")
	if err != nil {
		t.Fatalf("GuardarCheckpointHandoff: %v", err)
	}
	if checkpointID == 0 {
		t.Fatalf("checkpoint id inesperado: %d", checkpointID)
	}
}

func TestGuardarCheckpointHandoffFallaSinSesion(t *testing.T) {
	prepararDBTemporal(t)

	c := &HandoffCandidato{
		Agente:   "Codex1",
		Motivo:   "test",
		SesionID: nil,
	}
	if _, err := GuardarCheckpointHandoff(c, ""); err == nil {
		t.Fatalf("se esperaba error por SesionID nil")
	}
}

func TestProcesarHandoffsBatchSinCandidatos(t *testing.T) {
	prepararDBTemporal(t)

	n, err := ProcesarHandoffsBatch()
	if err != nil {
		t.Fatalf("ProcesarHandoffsBatch: %v", err)
	}
	if n != 0 {
		t.Fatalf("esperaba 0 handoffs, got=%d", n)
	}
}

func TestProcesarHandoffsBatchEscalaDeSondeoAHandoff(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, _ := prepararAgenteConTareaEnProgreso(t, "Codex1")
	setHeartbeatStale(t, sesionID)

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	// Codex2 libre (no tiene sesión activa)

	n, err := ProcesarHandoffsBatch()
	if err != nil {
		t.Fatalf("ProcesarHandoffsBatch: %v", err)
	}
	if n != 0 {
		t.Fatalf("el primer ciclo debe sondar antes de handoff, got=%d", n)
	}

	estadoPendiente := "pendiente"
	agenteOrigen := "Codex1"
	sondeo, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agenteOrigen, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders origen: %v", err)
	}
	if len(sondeo) != 2 {
		t.Fatalf("esperaba sync_status y nudge en el primer ciclo, got=%d", len(sondeo))
	}

	n, err = ProcesarHandoffsBatch()
	if err != nil {
		t.Fatalf("ProcesarHandoffsBatch segundo ciclo: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 handoff en el segundo ciclo, got=%d", n)
	}

	agente := "Codex2"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "handoff" {
		t.Fatalf("orden de handoff no encontrada para Codex2")
	}
}

func TestProcesarHandoffsBatchSinHandleHaceHandoffDirecto(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, tareaID := prepararAgenteConTareaEnProgreso(t, "Codex1")
	setHeartbeatStale(t, sesionID)
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE agente=?`, "Codex1"); err != nil {
		t.Fatalf("cerrar handles origen: %v", err)
	}

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}

	n, err := ProcesarHandoffsBatch()
	if err != nil {
		t.Fatalf("ProcesarHandoffsBatch: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 handoff directo sin sondeo, got=%d", n)
	}

	estadoPendiente := "pendiente"
	agenteOrigen := "Codex1"
	ordersOrigen, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agenteOrigen, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders origen: %v", err)
	}
	for _, order := range ordersOrigen {
		if order.Tipo == "sync_status" || order.Tipo == "nudge" {
			t.Fatalf("no deberia crear sondeo watchdog sin handle: %+v", ordersOrigen)
		}
	}

	agenteDestino := "Codex2"
	ordersDestino, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agenteDestino, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders destino: %v", err)
	}
	if len(ordersDestino) != 1 || ordersDestino[0].Tipo != "handoff" {
		t.Fatalf("handoff directo inesperado: %+v", ordersDestino)
	}
	kind := "handoff_requested"
	events, err := ListarAutonomyEvents(FiltroAutonomyEvents{
		Kind:   &kind,
		TaskID: &tareaID,
		Limite: 10,
	})
	if err != nil {
		t.Fatalf("ListarAutonomyEvents: %v", err)
	}
	if len(events) != 1 || events[0] == nil {
		t.Fatalf("autonomy events inesperados: %+v", events)
	}
	ev := events[0]
	if ev.Source != "handoff_manager" {
		t.Fatalf("source inesperado: %+v", ev)
	}
	if fmt.Sprint(ev.StateDelta["agente_origen"]) != "Codex1" ||
		fmt.Sprint(ev.StateDelta["agente_destino"]) != "Codex2" ||
		fmt.Sprint(ev.StateDelta["handoff_mode"]) != "stale" ||
		fmt.Sprint(ev.StateDelta["last_autonomy_action"]) != "handoff_requested" {
		t.Fatalf("state_delta handoff_requested inesperado: %+v", ev.StateDelta)
	}
	if len(ev.ArtifactsRef) != 1 || !strings.HasPrefix(ev.ArtifactsRef[0], "runtime_checkpoint:") {
		t.Fatalf("artifacts_ref inesperado: %+v", ev)
	}
	cp, err := UltimoRuntimeCheckpoint("Codex1", nil)
	if err != nil {
		t.Fatalf("UltimoRuntimeCheckpoint: %v", err)
	}
	if cp == nil {
		t.Fatalf("checkpoint handoff no encontrado")
	}
	if ev.ArtifactsRef[0] != fmt.Sprintf("runtime_checkpoint:%d", cp.ID) {
		t.Fatalf("artifacts_ref inesperado: %+v checkpoint=%+v", ev.ArtifactsRef, cp)
	}
	kind = "handoff_completed"
	events, err = ListarAutonomyEvents(FiltroAutonomyEvents{
		Kind:   &kind,
		TaskID: &tareaID,
		Limite: 10,
	})
	if err != nil {
		t.Fatalf("ListarAutonomyEvents handoff_completed: %v", err)
	}
	if len(events) != 1 || events[0] == nil {
		t.Fatalf("autonomy events handoff_completed inesperados: %+v", events)
	}
	ev = events[0]
	if ev.Source != "handoff_manager" {
		t.Fatalf("source handoff_completed inesperado: %+v", ev)
	}
	if fmt.Sprint(ev.StateDelta["agente_origen"]) != "Codex1" ||
		fmt.Sprint(ev.StateDelta["agente_destino"]) != "Codex2" ||
		fmt.Sprint(ev.StateDelta["handoff_mode"]) != "stale" ||
		fmt.Sprint(ev.StateDelta["task_state"]) != string(EstadoAsignada) ||
		fmt.Sprint(ev.StateDelta["last_autonomy_action"]) != "handoff_completed" ||
		fmt.Sprint(ev.StateDelta["runtime_order_id"]) != fmt.Sprint(ordersDestino[0].ID) {
		t.Fatalf("state_delta handoff_completed inesperado: %+v", ev.StateDelta)
	}
	if len(ev.ArtifactsRef) != 1 || ev.ArtifactsRef[0] != fmt.Sprintf("runtime_checkpoint:%d", cp.ID) {
		t.Fatalf("artifacts_ref handoff_completed inesperado: %+v", ev)
	}
	if err := RegistrarAutonomyEventHandoffCompleted(ordersDestino[0].ID, "handoff_manager", &tareaID, nil, "stale", cp.ID); err != nil {
		t.Fatalf("RegistrarAutonomyEventHandoffCompleted idempotente: %v", err)
	}
	events, err = ListarAutonomyEvents(FiltroAutonomyEvents{
		Kind:   &kind,
		TaskID: &tareaID,
		Limite: 10,
	})
	if err != nil {
		t.Fatalf("ListarAutonomyEvents handoff_completed segunda vez: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("handoff_completed no debe duplicarse, got=%d %+v", len(events), events)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("GetTarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex2" || tarea.Estado != EstadoAsignada {
		t.Fatalf("tarea reasignada inesperada: %+v", tarea)
	}
	sesion, err := GetSesionByID(sesionID)
	if err != nil {
		t.Fatalf("GetSesionByID: %v", err)
	}
	if sesion.Activa || sesion.Estado != "pausada" {
		t.Fatalf("sesion origen deberia quedar aparcada: %+v", sesion)
	}
}

func TestRegistrarAutonomyEventHandoffCompletedExigeConsolidacionReal(t *testing.T) {
	prepararDBTemporal(t)

	_, tareaID := prepararAgenteConTareaEnProgreso(t, "Codex1")
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex2: %v", err)
	}

	payloadJSON := fmt.Sprintf(`{"agente_origen":"Codex1","agente_destino":"Codex2","tarea_id":%d,"motivo":"test"}`, tareaID)
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex2",
		Tipo:        "handoff",
		PayloadJSON: payloadJSON,
	})
	if err != nil {
		t.Fatalf("EncolarRuntimeOrder handoff: %v", err)
	}

	err = RegistrarAutonomyEventHandoffCompleted(orderID, "handoff_manager", &tareaID, nil, "stale", 0)
	if err == nil {
		t.Fatal("esperaba error si la tarea aún no quedó consolidada en el destino")
	}

	kind := "handoff_completed"
	events, err := ListarAutonomyEvents(FiltroAutonomyEvents{
		Kind:   &kind,
		TaskID: &tareaID,
		Limite: 10,
	})
	if err != nil {
		t.Fatalf("ListarAutonomyEvents: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("no esperaba handoff_completed sin consolidación real: %+v", events)
	}
}

func TestRegistrarAutonomyEventHandoffFailedSinRuntimeOrderEsIdempotente(t *testing.T) {
	prepararDBTemporal(t)

	_, tareaID := prepararAgenteConTareaEnProgreso(t, "Codex1")

	if err := RegistrarAutonomyEventHandoffFailed(0, "handoff_manager", &tareaID, nil, "Codex1", "Codex2", "stale", "request", "fallo al crear handoff", 0); err != nil {
		t.Fatalf("RegistrarAutonomyEventHandoffFailed: %v", err)
	}
	if err := RegistrarAutonomyEventHandoffFailed(0, "handoff_manager", &tareaID, nil, "Codex1", "Codex2", "stale", "request", "fallo al crear handoff", 0); err != nil {
		t.Fatalf("RegistrarAutonomyEventHandoffFailed idempotente: %v", err)
	}

	kind := "handoff_failed"
	events, err := ListarAutonomyEvents(FiltroAutonomyEvents{
		Kind:   &kind,
		TaskID: &tareaID,
		Limite: 10,
	})
	if err != nil {
		t.Fatalf("ListarAutonomyEvents: %v", err)
	}
	if len(events) != 1 || events[0] == nil {
		t.Fatalf("handoff_failed inesperado: %+v", events)
	}
	ev := events[0]
	if ev.Source != "handoff_manager" || ev.Reason != "fallo al crear handoff" {
		t.Fatalf("handoff_failed metadata inesperada: %+v", ev)
	}
	if fmt.Sprint(ev.StateDelta["agente_origen"]) != "Codex1" ||
		fmt.Sprint(ev.StateDelta["agente_destino"]) != "Codex2" ||
		fmt.Sprint(ev.StateDelta["handoff_mode"]) != "stale" ||
		fmt.Sprint(ev.StateDelta["handoff_stage"]) != "request" ||
		fmt.Sprint(ev.StateDelta["last_autonomy_action"]) != "handoff_failed" {
		t.Fatalf("state_delta handoff_failed inesperado: %+v", ev.StateDelta)
	}
	if _, ok := ev.StateDelta["runtime_order_id"]; ok {
		t.Fatalf("runtime_order_id no debería existir sin orden real: %+v", ev.StateDelta)
	}
}

func TestRegistrarAutonomyEventHandoffFailedConOrdenRealUsaRuntimeOrderYNoDuplica(t *testing.T) {
	prepararDBTemporal(t)

	_, tareaID := prepararAgenteConTareaEnProgreso(t, "Codex1")
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex2: %v", err)
	}

	payloadJSON := fmt.Sprintf(`{"agente_origen":"Codex1","agente_destino":"Codex2","tarea_id":%d,"motivo":"watchdog"}`, tareaID)
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex2",
		Tipo:        "handoff",
		PayloadJSON: payloadJSON,
	})
	if err != nil {
		t.Fatalf("EncolarRuntimeOrder handoff: %v", err)
	}

	if err := RegistrarAutonomyEventHandoffFailed(orderID, "handoff_manager", &tareaID, nil, "", "", "stale", "consolidation", "la tarea no quedó consolidada", 7); err != nil {
		t.Fatalf("RegistrarAutonomyEventHandoffFailed: %v", err)
	}
	if err := RegistrarAutonomyEventHandoffFailed(orderID, "handoff_manager", &tareaID, nil, "", "", "stale", "consolidation", "la tarea no quedó consolidada", 7); err != nil {
		t.Fatalf("RegistrarAutonomyEventHandoffFailed idempotente: %v", err)
	}

	kind := "handoff_failed"
	events, err := ListarAutonomyEvents(FiltroAutonomyEvents{
		Kind:   &kind,
		TaskID: &tareaID,
		Limite: 10,
	})
	if err != nil {
		t.Fatalf("ListarAutonomyEvents: %v", err)
	}
	if len(events) != 1 || events[0] == nil {
		t.Fatalf("handoff_failed inesperado: %+v", events)
	}
	ev := events[0]
	if fmt.Sprint(ev.StateDelta["runtime_order_id"]) != fmt.Sprint(orderID) ||
		fmt.Sprint(ev.StateDelta["agente_origen"]) != "Codex1" ||
		fmt.Sprint(ev.StateDelta["agente_destino"]) != "Codex2" ||
		fmt.Sprint(ev.StateDelta["handoff_mode"]) != "stale" ||
		fmt.Sprint(ev.StateDelta["handoff_stage"]) != "consolidation" ||
		fmt.Sprint(ev.StateDelta["task_state"]) != string(EstadoEnProgreso) ||
		fmt.Sprint(ev.StateDelta["last_autonomy_action"]) != "handoff_failed" {
		t.Fatalf("state_delta handoff_failed con orden inesperado: %+v", ev.StateDelta)
	}
	if len(ev.ArtifactsRef) != 1 || ev.ArtifactsRef[0] != "runtime_checkpoint:7" {
		t.Fatalf("artifacts_ref handoff_failed inesperado: %+v", ev.ArtifactsRef)
	}
}

func TestRegistrarAutonomyEventHandoffFailedRechazaOrdenNoHandoff(t *testing.T) {
	prepararDBTemporal(t)

	_, tareaID := prepararAgenteConTareaEnProgreso(t, "Codex1")
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente: "Codex1",
		Tipo:   "nudge",
	})
	if err != nil {
		t.Fatalf("EncolarRuntimeOrder nudge: %v", err)
	}

	err = RegistrarAutonomyEventHandoffFailed(orderID, "handoff_manager", &tareaID, nil, "Codex1", "Codex2", "live", "request", "orden inválida", 0)
	if err == nil {
		t.Fatal("esperaba error para runtime_order que no es handoff")
	}

	kind := "handoff_failed"
	events, err := ListarAutonomyEvents(FiltroAutonomyEvents{
		Kind:   &kind,
		TaskID: &tareaID,
		Limite: 10,
	})
	if err != nil {
		t.Fatalf("ListarAutonomyEvents: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("no esperaba handoff_failed con orden no handoff: %+v", events)
	}
}

func TestProcesarHandoffsBatchPorPresupuestoNoRequiereSondeo(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, tareaID := prepararAgenteConTareaEnProgreso(t, "Codex1")
	registrarPresupuestoCritico(t, sesionID, 15, time.Now().UTC())

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}

	n, err := ProcesarHandoffsBatch()
	if err != nil {
		t.Fatalf("ProcesarHandoffsBatch: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 handoff por presupuesto, got=%d", n)
	}

	agenteDestino := "Codex2"
	estadoPendiente := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agenteDestino, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders destino: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "handoff" {
		t.Fatalf("handoff presupuesto inesperado: %+v", orders)
	}

	agenteOrigen := "Codex1"
	ordersOrigen, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agenteOrigen, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders origen: %v", err)
	}
	for _, order := range ordersOrigen {
		if order == nil {
			continue
		}
		if order.Tipo == "sync_status" || order.Tipo == "nudge" {
			t.Fatalf("no deberia sondar watchdog en handoff por presupuesto: %+v", ordersOrigen)
		}
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("GetTarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex2" || tarea.Estado != TareaAsignada {
		t.Fatalf("tarea no reasignada por presupuesto: %+v", tarea)
	}
}

func TestProcesarHandoffsBatchSinHandleEscalaDirectoAHandoff(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, tareaID := prepararAgenteConTareaEnProgreso(t, "Codex1")
	setHeartbeatStale(t, sesionID)
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='cerrado' WHERE agente=?`, "Codex1"); err != nil {
		t.Fatalf("cerrar handles origen: %v", err)
	}

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}

	n, err := ProcesarHandoffsBatch()
	if err != nil {
		t.Fatalf("ProcesarHandoffsBatch: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 handoff directo sin sondeo, got=%d", n)
	}

	agenteDestino := "Codex2"
	estadoPendiente := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agenteDestino, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders destino: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "handoff" {
		t.Fatalf("handoff directo inesperado: %+v", orders)
	}

	agenteOrigen := "Codex1"
	ordersOrigen, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agenteOrigen, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders origen: %v", err)
	}
	for _, order := range ordersOrigen {
		if order == nil {
			continue
		}
		if order.Tipo == "sync_status" || order.Tipo == "nudge" {
			t.Fatalf("no deberia sondar watchdog cuando falta handle activo: %+v", ordersOrigen)
		}
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("GetTarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex2" || tarea.Estado != TareaAsignada {
		t.Fatalf("tarea no reasignada por handoff directo: %+v", tarea)
	}
}

func TestProcesarHandoffsBatchNoSondeaAgenteEnEnfriamientoPausado(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, _ := prepararAgenteConTareaEnProgreso(t, "Codex1")
	setHeartbeatStale(t, sesionID)
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE agente=?`, "Codex1"); err != nil {
		t.Fatalf("pausar handle origen: %v", err)
	}
	if err := PausarAgenteHasta("Codex1", time.Now().UTC().Add(30*time.Minute), "Auto-pausa por agotamiento: usage limit de proveedor"); err != nil {
		t.Fatalf("pausar agente hasta: %v", err)
	}

	candidatos, err := DetectarAgentesAgotados()
	if err != nil {
		t.Fatalf("DetectarAgentesAgotados: %v", err)
	}
	for _, candidato := range candidatos {
		if candidato != nil && candidato.Agente == "Codex1" {
			t.Fatalf("Codex1 no deberia entrar como watchdog si esta en enfriamiento con handle pausado: %+v", candidato)
		}
	}

	n, err := ProcesarHandoffsBatch()
	if err != nil {
		t.Fatalf("ProcesarHandoffsBatch: %v", err)
	}
	if n != 0 {
		t.Fatalf("no deberia generar handoff ni sondeo para agente en enfriamiento, got=%d", n)
	}

	agenteOrigen := "Codex1"
	estadoPendiente := "pendiente"
	ordersOrigen, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agenteOrigen, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders origen: %v", err)
	}
	for _, order := range ordersOrigen {
		if order == nil {
			continue
		}
		if order.Tipo == "sync_status" || order.Tipo == "nudge" || order.Tipo == "handoff" {
			t.Fatalf("no deberia crear ruido watchdog para enfriamiento: %+v", ordersOrigen)
		}
	}
}
