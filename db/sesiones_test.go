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

func TestRegistrarAgenteRehabilitaExistenteRetirado(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente inicial: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET habilitado=0 WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("retirar manualmente agente: %v", err)
	}

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente rehabilitando: %v", err)
	}

	var habilitado int
	if err := DB.QueryRow(`SELECT habilitado FROM agentes WHERE nombre='Codex1'`).Scan(&habilitado); err != nil {
		t.Fatalf("leer habilitado: %v", err)
	}
	if habilitado != 1 {
		t.Fatalf("el agente deberia quedar rehabilitado, got=%d", habilitado)
	}
}

func TestIniciarSesionContextoAsignaPoolCanonicoParaOllamaPoolLocal(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Gemma1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Gemma1: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := GuardarPool(&PoolCapacidad{
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
		t.Fatalf("GuardarPool: %v", err)
	}
	if _, err := GuardarPoolModelo("ollama-gemma4", &PoolModelo{ModelSlug: "gemma4:26b", Activo: true, Prioridad: 10, CosteRelativo: 1}); err != nil {
		t.Fatalf("GuardarPoolModelo: %v", err)
	}
	if _, err := GuardarPoliticaModelo(&PoliticaModelo{
		ScopeTipo:       "perfil",
		ScopeRef:        "implementacion",
		PerfilTarea:     "implementacion",
		PoolSlug:        "ollama-gemma4",
		ModelSlug:       "gemma4:26b",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
	}); err != nil {
		t.Fatalf("GuardarPoliticaModelo: %v", err)
	}

	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Gemma1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "gemma1"),
		Herramienta: "ollama_pool_local",
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto: %v", err)
	}
	pool, err := GetPool("ollama-gemma4")
	if err != nil {
		t.Fatalf("GetPool: %v", err)
	}
	var poolID int64
	if err := DB.QueryRow(`SELECT pool_id FROM sesiones WHERE id = ?`, sesion.ID).Scan(&poolID); err != nil {
		t.Fatalf("leer sesiones.pool_id: %v", err)
	}
	if poolID != pool.ID {
		t.Fatalf("pool_id inesperado: got=%d want=%d", poolID, pool.ID)
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

	mailboxID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "server",
		ToAgente:    "CodexRetiro",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo"}`,
	})
	if err != nil {
		t.Fatalf("EnviarRuntimeMailbox: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "CodexRetiro",
		ProyectoID:  &proyectoID,
		Tipo:        "nudge",
		PayloadJSON: `{"accion":"continuar_trabajo"}`,
	})
	if err != nil {
		t.Fatalf("EnqueueRuntimeOrder nudge: %v", err)
	}
	pauseID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "CodexRetiro",
		ProyectoID:  &proyectoID,
		Tipo:        "pause",
		PayloadJSON: `{"accion":"pause","motivo":"retiro_agente"}`,
	})
	if err != nil {
		t.Fatalf("EnqueueRuntimeOrder pause: %v", err)
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

	mailbox, err := GetRuntimeMailbox(mailboxID)
	if err != nil {
		t.Fatalf("GetRuntimeMailbox: %v", err)
	}
	if mailbox == nil || mailbox.Estado != "cancelado" {
		t.Fatalf("mailbox no cancelada: %+v", mailbox)
	}

	order, err := GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("GetRuntimeOrder nudge: %v", err)
	}
	if order == nil || order.Estado != "cancelada" {
		t.Fatalf("runtime order nudge no cancelada: %+v", order)
	}

	pauseOrder, err := GetRuntimeOrder(pauseID)
	if err != nil {
		t.Fatalf("GetRuntimeOrder pause: %v", err)
	}
	if pauseOrder == nil || pauseOrder.Estado != "pendiente" {
		t.Fatalf("runtime order pause no deberia cancelarse: %+v", pauseOrder)
	}
}

func TestRegistrarAgenteNoRehabilitaAgenteRetirado(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("CodexRetenido", "programador"); err != nil {
		t.Fatalf("RegistrarAgente inicial: %v", err)
	}
	if err := RetirarAgente("CodexRetenido"); err != nil {
		t.Fatalf("RetirarAgente: %v", err)
	}

	if err := RegistrarAgente("CodexRetenido", "programador"); err != nil {
		t.Fatalf("RegistrarAgente repetido: %v", err)
	}

	agente, err := GetAgente("CodexRetenido")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.Habilitado {
		t.Fatalf("RegistrarAgente no deberia rehabilitar un agente retirado: %+v", agente)
	}
}

func TestListarAgentesAlineaEstadoVisibleConSesiones(t *testing.T) {
	prepararDBTemporal(t)
	for _, agente := range []string{"CodexVisible1", "CodexVisible2"} {
		if err := RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", agente, err)
		}
	}
	if _, err := DB.Exec(`UPDATE agentes SET activo=1, estado_sesion='disponible' WHERE nombre='CodexVisible2'`); err != nil {
		t.Fatalf("marcar CodexVisible2 activo: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "CodexVisible1",
		CWD:         "/tmp/orquesta-codex-visible1",
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("IniciarSesionContexto visible1: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='pensando' WHERE nombre='CodexVisible1'`); err != nil {
		t.Fatalf("marcar estado visible1: %v", err)
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

func TestListarAgentesEstadoLigeroConSesionesActivasNoEnriquecePresupuesto(t *testing.T) {
	prepararDBTemporal(t)
	for _, agente := range []string{"CodexLight1", "CodexLight2"} {
		if err := RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", agente, err)
		}
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_cuota='agotado', motivo_pausa='legacy' WHERE nombre='CodexLight1'`); err != nil {
		t.Fatalf("actualizar cuota legacy: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "CodexLight1",
		CWD:         "/tmp/orquesta-codex-light1",
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto CodexLight1: %v", err)
	}
	if err := UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("UpsertRuntimeHandleDesdeSesion: %v", err)
	}
	sesiones, err := ListarSesionesActivasOperativas()
	if err != nil {
		t.Fatalf("ListarSesionesActivasOperativas: %v", err)
	}

	agentes, err := ListarAgentesEstadoLigeroConSesionesActivas(sesiones)
	if err != nil {
		t.Fatalf("ListarAgentesEstadoLigeroConSesionesActivas: %v", err)
	}
	porNombre := map[string]*Agente{}
	for _, agente := range agentes {
		if agente != nil {
			porNombre[agente.Nombre] = agente
		}
	}
	agente := porNombre["CodexLight1"]
	if agente == nil {
		t.Fatalf("faltaba CodexLight1")
	}
	if agente.EstadoSesion != "disponible" {
		t.Fatalf("estado visible inesperado: %+v", agente)
	}
	if agente.EstadoCuota != "agotado" {
		t.Fatalf("estado cuota inesperado: %+v", agente)
	}
	if agente.ReanimarAt != nil {
		t.Fatalf("el listado ligero no deberia rehidratar presupuesto detallado: %+v", agente)
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
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "CodexFresh",
		CWD:         "/tmp/orquesta-codex-fresh",
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("IniciarSesionContexto CodexFresh: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at)
		VALUES (?,?,?,?,?,?)`,
		"CodexZombie", 1, "activa", "codex", "localhost", old,
	); err != nil {
		t.Fatalf("insert sesion zombie: %v", err)
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
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "CodexCooldown",
		CWD:         "/tmp/orquesta-codex-cooldown",
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("IniciarSesionContexto: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='pensando', estado_cuota='enfriamiento', reanimar_at=CURRENT_TIMESTAMP WHERE nombre='CodexCooldown'`); err != nil {
		t.Fatalf("marcar enfriamiento: %v", err)
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
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "CodexPausado",
		CWD:         "/tmp/orquesta-codex-pausado",
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='pausada' WHERE nombre='CodexPausado'`); err != nil {
		t.Fatalf("marcar estado sesion: %v", err)
	}
	if _, err := DB.Exec(`UPDATE sesiones SET estado='pausada' WHERE id=?`, sesion.ID); err != nil {
		t.Fatalf("marcar sesion pausada: %v", err)
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

func TestSesionRecienteSinRuntimeNiHandleNoCuentaComoOperativa(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("CodexSinSoporte", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "CodexSinSoporte",
		CWD:         "/tmp/orquesta-codex-sin-soporte",
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("IniciarSesionContexto: %v", err)
	}
	if _, err := DB.Exec(`DELETE FROM runtime_handles WHERE agente=?`, "CodexSinSoporte"); err != nil {
		t.Fatalf("delete runtime_handles: %v", err)
	}
	if _, err := DB.Exec(`DELETE FROM runtime_instances WHERE agente=?`, "CodexSinSoporte"); err != nil {
		t.Fatalf("delete runtime_instances: %v", err)
	}
	runtimeHandleHotReset()

	live, err := GetSesionActivaOperativa("CodexSinSoporte", nil)
	if err == nil || live != nil {
		t.Fatalf("CodexSinSoporte no deberia tener sesion operativa sin runtime/handle: %+v err=%v", live, err)
	}

	sesiones, err := ListarSesionesActivas()
	if err != nil {
		t.Fatalf("ListarSesionesActivas: %v", err)
	}
	for _, sesion := range sesiones {
		if sesion != nil && sesion.Agente == "CodexSinSoporte" {
			t.Fatalf("CodexSinSoporte no deberia salir en sesiones operativas: %+v", sesion)
		}
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

func TestGetAgenteConservaPausaOperativaVisibleSinSesionActiva(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("CodexRuntimePanic", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	reanimarAt := time.Now().UTC().Add(10 * time.Minute)
	if _, err := DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', reanimar_at=?, motivo_pausa='Auto-pausa por runtime_panic: transcript=10576' WHERE nombre='CodexRuntimePanic'`, reanimarAt); err != nil {
		t.Fatalf("actualizar agente: %v", err)
	}

	agente, err := GetAgente("CodexRuntimePanic")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.EstadoCuota != "enfriamiento" {
		t.Fatalf("estado_cuota operativo no deberia limpiarse: %+v", agente)
	}
	if agente.ReanimarAt == nil || agente.ReanimarAt.IsZero() {
		t.Fatalf("reanimar_at operativo no deberia limpiarse: %+v", agente)
	}
	if !strings.Contains(agente.MotivoPausa, "runtime_panic") {
		t.Fatalf("motivo_pausa operativo inesperado: %+v", agente)
	}
}

func TestGetAgentePrefiereIdentidadCuentaDesdeHandleCanonico(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("CodexCuenta", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	sesionLegacyID, err := IniciarSesion("CodexCuenta")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	pid := int64(9101)
	runtimeLegacyID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexCuenta",
		SesionID:     &sesionLegacyID,
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeInstance legacy: %v", err)
	}
	legacyMeta := `{"driver":"process_pty_cli","account_email":"legacy@avidad.com","account_user":"Legacy"}`
	if _, err := DB.Exec(`
		INSERT INTO runtime_handles (
			agente, sesion_id, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado,
			capabilities_json, metadata_json, last_seen_at, updated_at, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		"CodexCuenta", nil, proyectoID, runtimeLegacyID, "cli", "process", "9101", "activo", "{}", legacyMeta,
	); err != nil {
		t.Fatalf("insert legacy handle: %v", err)
	}

	runtimeTMUXID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "CodexCuenta",
		SesionID:     &sesionLegacyID,
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pid,
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeInstance tmux: %v", err)
	}
	tmuxMeta := `{"driver":"tmux_cli_session","tmux_session":"orq-codexcuenta-1","account_email":"canonico@avidad.com","account_user":"Canonico"}`
	if _, err := DB.Exec(`
		INSERT INTO runtime_handles (
			agente, sesion_id, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado,
			capabilities_json, metadata_json, last_seen_at, updated_at, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		"CodexCuenta", nil, proyectoID, runtimeTMUXID, "tmux", "session", "orq-codexcuenta-1/%1", "activo", "{}", tmuxMeta,
	); err != nil {
		t.Fatalf("insert tmux handle: %v", err)
	}

	agente, err := GetAgente("CodexCuenta")
	if err != nil {
		t.Fatalf("GetAgente: %v", err)
	}
	if agente.CuentaEmail != "canonico@avidad.com" {
		t.Fatalf("deberia preferir identidad desde handle canónico tmux, got=%+v", agente)
	}
	if agente.CuentaUsuario != "Canonico" {
		t.Fatalf("usuario inesperado: %+v", agente)
	}
}
