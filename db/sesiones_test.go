package db

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestIniciarSesionDevuelveIDPersistido(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("codex-sesion", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}

	id, err := IniciarSesion("codex-sesion")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if id == 0 {
		t.Fatalf("id de sesion no valido: %d", id)
	}

	var persistedID int64
	if err := DB.QueryRow(`
		SELECT id
		FROM sesiones
		WHERE agente = ? AND activa = 1
		ORDER BY id DESC
		LIMIT 1`, "codex-sesion").Scan(&persistedID); err != nil {
		t.Fatalf("select sesion activa: %v", err)
	}
	if persistedID != id {
		t.Fatalf("id devuelto %d distinto del persistido %d", id, persistedID)
	}
}

func TestRegistrarCodexUsaNombreCanonicoYRespetaExistentes(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex1: %v", err)
	}
	if err := RegistrarAgente("codex2", "programador"); err != nil {
		t.Fatalf("RegistrarAgente codex2: %v", err)
	}

	nombre, err := RegistrarCodex()
	if err != nil {
		t.Fatalf("RegistrarCodex: %v", err)
	}
	if nombre != "Codex3" {
		t.Fatalf("nombre inesperado: %s", nombre)
	}
}

func TestRegistrarAgenteAutoUsaPrefijoCanonicoSegunProveedor(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("Claude1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Claude1: %v", err)
	}
	if err := RegistrarAgente("gemini2", "programador"); err != nil {
		t.Fatalf("RegistrarAgente gemini2: %v", err)
	}

	nombreClaude, err := RegistrarAgenteAuto("anthropic", "documentador")
	if err != nil {
		t.Fatalf("RegistrarAgenteAuto anthropic: %v", err)
	}
	if nombreClaude != "Claude2" {
		t.Fatalf("nombre Claude inesperado: %s", nombreClaude)
	}

	nombreGemini, err := RegistrarAgenteAuto("google", "programador")
	if err != nil {
		t.Fatalf("RegistrarAgenteAuto google: %v", err)
	}
	if nombreGemini != "Gemini3" {
		t.Fatalf("nombre Gemini inesperado: %s", nombreGemini)
	}
}

func TestResolverAgentePorNombreCIAmbiguoSinCoincidenciaExactaFalla(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex1: %v", err)
	}
	if err := RegistrarAgente("CODEX1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente CODEX1: %v", err)
	}

	_, _, _, err := resolverAgentePorNombreCI("codex1")
	if err == nil {
		t.Fatalf("se esperaba error por ambiguedad de mayusculas/minusculas")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "ambigu") {
		t.Fatalf("error inesperado: %v", err)
	}

	nombre, _, _, err := resolverAgentePorNombreCI("Codex1")
	if err != nil {
		t.Fatalf("coincidencia exacta deberia seguir funcionando: %v", err)
	}
	if nombre != "Codex1" {
		t.Fatalf("nombre canonico inesperado: %q", nombre)
	}
}

func TestEliminarAgenteBloqueaTareasActivas(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("Codex7", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Coordinar runtime order",
		Descripcion: "no borrar agente con trabajo vivo",
		Modulo:      "controlplane",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}

	err = EliminarAgente("Codex7")
	if err == nil {
		t.Fatalf("esperaba bloqueo al eliminar agente con tareas activas")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "tarea") {
		t.Fatalf("error inesperado: %v", err)
	}
	if _, err := GetAgente("Codex7"); err != nil {
		t.Fatalf("el agente no deberia haberse borrado: %v", err)
	}
}

func TestRetirarAgentePausaAsignacionesYLiberaTrabajo(t *testing.T) {
	dir := prepararDBTemporal(t)
	if err := RegistrarAgente("CodexRetiro", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo-retiro",
		Nombre:  "Demo Retiro",
		RutaAbs: filepath.Join(dir, "demo-retiro"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if err := ActivarAsignacion("CodexRetiro", proyectoID, "frente_activo"); err != nil {
		t.Fatalf("ActivarAsignacion: %v", err)
	}
	if _, err := IniciarSesion("CodexRetiro"); err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}

	tareaAsignada, err := CrearTarea(&Tarea{
		Titulo:      "Tarea asignada",
		Descripcion: "debe volver al pool",
		ProyectoID:  &proyectoID,
		Modulo:      "backend",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea asignada: %v", err)
	}
	if err := TomarTarea(tareaAsignada, "CodexRetiro"); err != nil {
		t.Fatalf("TomarTarea asignada: %v", err)
	}

	tareaViva, err := CrearTarea(&Tarea{
		Titulo:      "Tarea viva",
		Descripcion: "debe volver al pool conservando nota",
		ProyectoID:  &proyectoID,
		Modulo:      "frontend",
		Prioridad:   PrioridadMedia,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea viva: %v", err)
	}
	if err := TomarTarea(tareaViva, "CodexRetiro"); err != nil {
		t.Fatalf("TomarTarea viva: %v", err)
	}
	if err := IniciarTarea(tareaViva, "CodexRetiro"); err != nil {
		t.Fatalf("IniciarTarea viva: %v", err)
	}

	if err := RetirarAgente("CodexRetiro"); err != nil {
		t.Fatalf("RetirarAgente: %v", err)
	}

	agente, err := GetAgente("CodexRetiro")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.Habilitado || agente.Activo {
		t.Fatalf("estado de agente inesperado: %+v", agente)
	}

	agenteNombre := "CodexRetiro"
	asignaciones, err := ListarAsignaciones(FiltroAsignaciones{Agente: &agenteNombre})
	if err != nil {
		t.Fatalf("ListarAsignaciones: %v", err)
	}
	if len(asignaciones) == 0 || asignaciones[0].Estado != AsignacionPausada {
		t.Fatalf("asignacion no pausada: %+v", asignaciones)
	}
	if !strings.Contains(asignaciones[0].Nota, "agente_retirado") {
		t.Fatalf("nota de asignacion inesperada: %+v", asignaciones[0])
	}

	for _, id := range []int64{tareaAsignada, tareaViva} {
		tarea, err := GetTarea(id)
		if err != nil {
			t.Fatalf("GetTarea %d: %v", id, err)
		}
		if tarea.Estado != TareaLibre {
			t.Fatalf("tarea %d no liberada: %+v", id, tarea)
		}
		if tarea.Agente != nil {
			t.Fatalf("tarea %d mantiene agente: %+v", id, tarea)
		}
		if !strings.Contains(strings.ToLower(tarea.Notas), "agente retirado automáticamente") {
			t.Fatalf("tarea %d sin anotacion de retiro: %+v", id, tarea)
		}
	}

	var activas int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM sesiones WHERE agente=? AND activa=1`, "CodexRetiro").Scan(&activas); err != nil {
		t.Fatalf("count sesiones activas: %v", err)
	}
	if activas != 0 {
		t.Fatalf("esperaba 0 sesiones activas, got=%d", activas)
	}
}

func TestListarAgentesAlineaEstadoVisibleConSesiones(t *testing.T) {
	prepararDBTemporal(t)
	for _, agente := range []string{"CodexVisible1", "CodexVisible2"} {
		if err := RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", agente, err)
		}
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='pensando' WHERE nombre='CodexVisible1'`); err != nil {
		t.Fatalf("marcar estado visible1: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET activo=1, estado_sesion='disponible' WHERE nombre='CodexVisible2'`); err != nil {
		t.Fatalf("marcar CodexVisible2 activo: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO sesiones (agente, activa, estado, herramienta, host) VALUES (?,?,?,?,?)`,
		"CodexVisible1", 1, "activa", "codex", "localhost",
	); err != nil {
		t.Fatalf("insert sesion activa visible: %v", err)
	}

	agentes, err := ListarAgentes()
	if err != nil {
		t.Fatalf("ListarAgentes: %v", err)
	}
	estado := map[string]*Agente{}
	for _, agente := range agentes {
		estado[agente.Nombre] = agente
	}

	if !estado["CodexVisible1"].Activo || estado["CodexVisible1"].EstadoSesion != "pensando" {
		t.Fatalf("CodexVisible1 visible inesperado: %+v", estado["CodexVisible1"])
	}
	if estado["CodexVisible2"].Activo || estado["CodexVisible2"].EstadoSesion != "" {
		t.Fatalf("CodexVisible2 visible inesperado: %+v", estado["CodexVisible2"])
	}

	visible1, err := GetAgente("CodexVisible1")
	if err != nil {
		t.Fatalf("GetAgente visible1: %v", err)
	}
	if !visible1.Activo || visible1.EstadoSesion != "pensando" {
		t.Fatalf("GetAgente visible1 inesperado: %+v", visible1)
	}

	visible2, err := GetAgente("CodexVisible2")
	if err != nil {
		t.Fatalf("GetAgente visible2: %v", err)
	}
	if visible2.Activo || visible2.EstadoSesion != "" {
		t.Fatalf("GetAgente visible2 inesperado: %+v", visible2)
	}
}

func TestListarAgentesOcultaSesionZombiPeroMantieneHandleActivo(t *testing.T) {
	prepararDBTemporal(t)
	for _, agente := range []string{"CodexFresh", "CodexZombie", "CodexHandle"} {
		if err := RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", agente, err)
		}
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='pensando' WHERE nombre IN ('CodexFresh','CodexZombie','CodexHandle')`); err != nil {
		t.Fatalf("marcar estado visible: %v", err)
	}
	old := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	if _, err := DB.Exec(`
		INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at)
		VALUES (?,?,?,?,?,CURRENT_TIMESTAMP),
		       (?,?,?,?,?,?)`,
		"CodexFresh", 1, "activa", "codex", "localhost",
		"CodexZombie", 1, "activa", "codex", "localhost", old,
	); err != nil {
		t.Fatalf("insert sesiones visibles: %v", err)
	}
	var handleSesionID int64
	if err := DB.QueryRow(`
		INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at)
		VALUES (?,?,?,?,?,?)
		RETURNING id`,
		"CodexHandle", 1, "activa", "codex", "localhost", old,
	).Scan(&handleSesionID); err != nil {
		t.Fatalf("insert sesion CodexHandle: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO runtime_handles (
			agente, sesion_id, transporte, handle_kind, handle_ref, estado,
			capabilities_json, metadata_json, last_seen_at
		) VALUES (?,?,?,?,?,'activo','{}','{}',CURRENT_TIMESTAMP)`,
		"CodexHandle", handleSesionID, "cli", "session", "sess-codex-handle",
	); err != nil {
		t.Fatalf("insert handle CodexHandle: %v", err)
	}

	agentes, err := ListarAgentes()
	if err != nil {
		t.Fatalf("ListarAgentes: %v", err)
	}
	estado := map[string]*Agente{}
	for _, agente := range agentes {
		estado[agente.Nombre] = agente
	}

	if !estado["CodexFresh"].Activo {
		t.Fatalf("CodexFresh deberia seguir activo: %+v", estado["CodexFresh"])
	}
	if estado["CodexZombie"].Activo || estado["CodexZombie"].EstadoSesion != "" {
		t.Fatalf("CodexZombie no deberia verse activo: %+v", estado["CodexZombie"])
	}
	if !estado["CodexHandle"].Activo {
		t.Fatalf("CodexHandle deberia seguir activo por handle vivo: %+v", estado["CodexHandle"])
	}

	zombie, err := GetSesionActivaOperativa("CodexZombie", nil)
	if err == nil || zombie != nil {
		t.Fatalf("CodexZombie no deberia tener sesion operativa: %+v err=%v", zombie, err)
	}
	live, err := GetSesionActivaOperativa("CodexHandle", nil)
	if err != nil || live == nil {
		t.Fatalf("CodexHandle deberia tener sesion operativa por handle activo: sesion=%+v err=%v", live, err)
	}
}

func TestListarAgentesIgnoraSesionesConHeartbeatObsoleto(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("CodexZombie", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	if _, err := IniciarSesion("CodexZombie"); err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}

	old := time.Now().UTC().Add(-10 * time.Minute)
	if _, err := DB.Exec(`UPDATE agentes SET activo=1, estado_sesion='pensando' WHERE nombre='CodexZombie'`); err != nil {
		t.Fatalf("marcar agente activo: %v", err)
	}
	if _, err := DB.Exec(`UPDATE sesiones SET heartbeat_at=? WHERE agente=? AND activa=1`, old, "CodexZombie"); err != nil {
		t.Fatalf("marcar heartbeat obsoleto: %v", err)
	}

	agente, err := GetAgente("CodexZombie")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.Activo || agente.EstadoSesion != "" {
		t.Fatalf("el agente no deberia seguir visible como activo: %+v", agente)
	}

	sesiones, err := ListarSesionesActivas()
	if err != nil {
		t.Fatalf("ListarSesionesActivas: %v", err)
	}
	for _, sesion := range sesiones {
		if sesion != nil && sesion.Agente == "CodexZombie" {
			t.Fatalf("la sesion stale no deberia aparecer como operativa: %+v", sesion)
		}
	}
}

func TestListarAgentesOcultaAgenteEnEnfriamientoAunqueTengaSesionOperativa(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("CodexCooldown", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='pensando', estado_cuota='enfriamiento', reanimar_at=CURRENT_TIMESTAMP WHERE nombre='CodexCooldown'`); err != nil {
		t.Fatalf("marcar enfriamiento: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at) VALUES (?,?,?,?,?,CURRENT_TIMESTAMP)`,
		"CodexCooldown", 1, "activa", "codex", "localhost",
	); err != nil {
		t.Fatalf("insert sesion: %v", err)
	}

	agente, err := GetAgente("CodexCooldown")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.Activo {
		t.Fatalf("el agente en enfriamiento no deberia verse activo: %+v", agente)
	}
	if agente.EstadoSesion != "pensando" {
		t.Fatalf("estado visible inesperado: %+v", agente)
	}
}

func TestListarAgentesOcultaAgentePausadoAunqueMantengaHeartbeat(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("CodexPausado", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='pausada' WHERE nombre='CodexPausado'`); err != nil {
		t.Fatalf("marcar estado sesion: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at) VALUES (?,?,?,?,?,CURRENT_TIMESTAMP)`,
		"CodexPausado", 1, "pausada", "codex", "localhost",
	); err != nil {
		t.Fatalf("insert sesion: %v", err)
	}

	agente, err := GetAgente("CodexPausado")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.Activo {
		t.Fatalf("el agente pausado no deberia verse activo: %+v", agente)
	}
	if agente.EstadoSesion != "pausada" {
		t.Fatalf("estado visible inesperado: %+v", agente)
	}
}

func TestSesionRecienteNoCuentaComoOperativaSiSuUltimoHandleYaFallo(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("CodexFallo", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	var sesionID int64
	if err := DB.QueryRow(`
		INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at)
		VALUES (?,?,?,?,?,CURRENT_TIMESTAMP)
		RETURNING id`,
		"CodexFallo", 1, "activa", "codex-cli", "localhost",
	).Scan(&sesionID); err != nil {
		t.Fatalf("insert sesion: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO runtime_handles (
			agente, sesion_id, transporte, handle_kind, handle_ref, estado,
			capabilities_json, metadata_json, last_seen_at, updated_at, created_at
		) VALUES (?,?,?,?,?,'fallido','{}','{}',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		"CodexFallo", sesionID, "cli", "process", "9999",
	); err != nil {
		t.Fatalf("insert handle fallido: %v", err)
	}

	live, err := GetSesionActivaOperativa("CodexFallo", nil)
	if err == nil || live != nil {
		t.Fatalf("CodexFallo no deberia tener sesion operativa tras handle fallido: %+v err=%v", live, err)
	}

	sesiones, err := ListarSesionesActivas()
	if err != nil {
		t.Fatalf("ListarSesionesActivas: %v", err)
	}
	for _, sesion := range sesiones {
		if sesion != nil && sesion.Agente == "CodexFallo" {
			t.Fatalf("CodexFallo no deberia salir en sesiones operativas: %+v", sesion)
		}
	}

	agente, err := GetAgente("CodexFallo")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.Activo || agente.EstadoSesion != "" {
		t.Fatalf("CodexFallo no deberia seguir visible como activo: %+v", agente)
	}
}

func TestListarAgentesOcultaHandleActivoSiSuRuntimePrincipalYaEstaStale(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("CodexRuntimeStale", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	var sesionID int64
	if err := DB.QueryRow(`
		INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at)
		VALUES (?,?,?,?,?,CURRENT_TIMESTAMP)
		RETURNING id`,
		"CodexRuntimeStale", 1, "activa", "codex-cli", "localhost",
	).Scan(&sesionID); err != nil {
		t.Fatalf("insert sesion: %v", err)
	}
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:            "CodexRuntimeStale",
		SesionID:          &sesionID,
		Provider:          "openai",
		Connector:         "codex-cli",
		LogicalState:      "esperando_io",
		ProcessState:      "running",
		ExternalSessionID: "sess-runtime-stale",
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeInstance: %v", err)
	}
	old := time.Now().UTC().Add(-15 * time.Minute)
	if _, err := DB.Exec(`
		UPDATE runtime_instances
		SET last_event_at=?, last_heartbeat_at=?, updated_at=?
		WHERE id = ?`,
		old, old, old, runtimeID,
	); err != nil {
		t.Fatalf("stale runtime: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO runtime_handles (
			agente, sesion_id, runtime_id, transporte, handle_kind, handle_ref, estado,
			capabilities_json, metadata_json, last_seen_at, updated_at, created_at
		) VALUES (?,?,?,?,?,'7777','activo','{}','{}',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		"CodexRuntimeStale", sesionID, runtimeID, "cli", "process",
	); err != nil {
		t.Fatalf("insert handle activo: %v", err)
	}

	agente, err := GetAgente("CodexRuntimeStale")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.Activo || agente.EstadoSesion != "" {
		t.Fatalf("CodexRuntimeStale no deberia verse activo: %+v", agente)
	}

	sesiones, err := ListarSesionesActivas()
	if err != nil {
		t.Fatalf("ListarSesionesActivas: %v", err)
	}
	for _, sesion := range sesiones {
		if sesion != nil && sesion.Agente == "CodexRuntimeStale" {
			t.Fatalf("CodexRuntimeStale no deberia salir en sesiones operativas: %+v", sesion)
		}
	}
}

func TestGetAgenteNoMuestraActivoSiLaSemanalObservadaStaleYaEstaAgotada(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	sesionID, err := IniciarSesion("codex1")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	now := time.Now().UTC()
	resetPrimary := now.Add(2 * time.Hour)
	resetWeekly := now.Add(4 * 24 * time.Hour)
	raw := `{"rate_limits":{"primary":{"used_percent":5,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":100,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetWeekly.Unix(), 10) + `}}}`
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:        sesionID,
		WindowKind:      "5h",
		BudgetSource:    "codex_token_count_observed",
		RawSnapshotJSON: raw,
		CheckedAt:       now.Add(-6 * time.Hour),
	}); err != nil {
		t.Fatalf("RegistrarPresupuestoSesion: %v", err)
	}

	agente, err := GetAgente("codex1")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.Activo {
		t.Fatalf("codex1 no deberia verse activo con semanal observada agotada: %+v", agente)
	}
	if agente.EstadoCuota != "agotado" {
		t.Fatalf("estado_cuota inesperado: %+v", agente)
	}
	if agente.CuotaRestantePct == nil || *agente.CuotaRestantePct != 0 {
		t.Fatalf("la cuota visible deberia agotarse por semanal observada stale: %+v", agente)
	}
	if agente.PresupuestoVentana != "weekly" {
		t.Fatalf("la ventana efectiva deberia seguir siendo weekly: %+v", agente)
	}
}
