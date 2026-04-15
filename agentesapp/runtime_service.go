package agentesapp

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"orquesta/db"
	"orquesta/internal/bootstrapruntime"
	"orquesta/internal/lanzamientoruntime"
	"orquesta/runtimeagente"
)

const liveOperationalStateTimeout = 750 * time.Millisecond

type ProjectBundle struct {
	ID      int64  `json:"id"`
	Slug    string `json:"slug"`
	Nombre  string `json:"nombre"`
	RutaAbs string `json:"ruta_abs"`
}

type ConnectorBundle struct {
	ID         int64  `json:"id"`
	Slug       string `json:"slug"`
	Nombre     string `json:"nombre"`
	Transporte string `json:"transporte"`
	Comando    string `json:"comando"`
}

type SessionBundle struct {
	ID                 int64  `json:"id"`
	Estado             string `json:"estado"`
	CWD                string `json:"cwd"`
	Herramienta        string `json:"herramienta"`
	ExternalSessionID  string `json:"external_session_id"`
	ResumePayloadJSON  string `json:"resume_payload_json"`
	ResumenContinuidad string `json:"resumen_continuidad"`
	Branch             string `json:"branch"`
}

type Policy struct {
	ModoBucle         string `json:"modo_bucle"`
	IntervaloTickSeg  int    `json:"intervalo_tick_seg"`
	ContinuarHasta    string `json:"continuar_hasta"`
	ConsultarProyecto bool   `json:"consultar_proyecto"`
	Modelo            string `json:"modelo,omitempty"`
	Razonamiento      string `json:"razonamiento,omitempty"`
	PerfilTarea       string `json:"perfil_tarea,omitempty"`
}

type LightItem struct {
	ID     int64  `json:"id"`
	Codigo string `json:"codigo,omitempty"`
	Titulo string `json:"titulo"`
	Estado string `json:"estado,omitempty"`
}

type PrepareInput struct {
	Agente       string
	Proyecto     string
	Conector     string
	Modelo       string
	Razonamiento string
	Perfil       string
}

type PrepareOutput struct {
	Agente          string                    `json:"agente"`
	Rol             string                    `json:"rol"`
	Proyecto        ProjectBundle             `json:"proyecto"`
	Conector        ConnectorBundle           `json:"conector"`
	Politica        Policy                    `json:"politica"`
	UltimaSesion    *SessionBundle            `json:"ultima_sesion,omitempty"`
	Plan            *runtimeagente.LaunchPlan `json:"plan"`
	BootstrapPrompt string                    `json:"bootstrap_prompt,omitempty"`
	Reglas          []*db.Regla               `json:"reglas"`
	Skills          []*db.Skill               `json:"skills"`
	Workflows       []*db.Workflow            `json:"workflows"`
	Memoria         []*db.EntidadMemoria      `json:"memoria,omitempty"`
	Bootstrap       *bootstrapruntime.State   `json:"bootstrap,omitempty"`
	ReanimarAt      *time.Time                `json:"reanimar_at,omitempty"`
	MotivoPausa     string                    `json:"motivo_pausa,omitempty"`
	EstadoCuota     string                    `json:"estado_cuota,omitempty"`
}

type TickInput struct {
	Agente              string
	Proyecto            string
	Host                string
	PID                 int64
	CuotaPct            int
	Finalizado          bool
	Motivo              string
	AckStartOrderID     int64
	AckBootstrapOrderID int64
	AckMailboxIDs       []int64
}

type TickOutput struct {
	Agente               string         `json:"agente"`
	Proyecto             ProjectBundle  `json:"proyecto"`
	Politica             Policy         `json:"politica"`
	SesionActiva         *SessionBundle `json:"sesion_activa,omitempty"`
	AsignadoAProyecto    bool           `json:"asignado_a_proyecto"`
	ProyectoAsignado     string         `json:"proyecto_asignado,omitempty"`
	AccionRecomendada    string         `json:"accion_recomendada"`
	DebePausar           bool           `json:"debe_pausar"`
	Motivo               string         `json:"motivo,omitempty"`
	TareasActivas        []LightItem    `json:"tareas_activas,omitempty"`
	PropuestasPendientes []LightItem    `json:"propuestas_pendientes,omitempty"`
	PropuestasAbiertas   []LightItem    `json:"propuestas_abiertas,omitempty"`
	ConsumoDia           int            `json:"consumo_dia_segundos"`
	LimiteDia            int            `json:"limite_dia_segundos"`
	EstadoCuota          string         `json:"estado_cuota"`
	ReanimarAt           *time.Time     `json:"reanimar_at,omitempty"`
	MotivoPausa          string         `json:"motivo_pausa,omitempty"`
	CuotaPct             int            `json:"cuota_pct,omitempty"`
}

func (s *Service) BuildPrepare(input PrepareInput) (*PrepareOutput, error) {
	agenteNombre := strings.TrimSpace(input.Agente)
	proyectoRef := strings.TrimSpace(input.Proyecto)
	if agenteNombre == "" || proyectoRef == "" {
		return nil, fmt.Errorf("debes indicar agente y proyecto")
	}
	start := time.Now()
	prepareDebugf("BuildPrepare start agente=%s proyecto=%s conector=%s", agenteNombre, proyectoRef, strings.TrimSpace(input.Conector))
	defer func() {
		prepareDebugf("BuildPrepare done agente=%s proyecto=%s duration=%s", agenteNombre, proyectoRef, time.Since(start).Round(time.Millisecond))
	}()

	stepStart := time.Now()
	agente, err := s.store.GetAgent(agenteNombre)
	if err != nil {
		return nil, err
	}
	prepareDebugf("BuildPrepare step=get_agent agente=%s duration=%s", agenteNombre, time.Since(stepStart).Round(time.Millisecond))
	agenteRuntime := *agente
	if strings.EqualFold(strings.TrimSpace(input.Agente), strings.TrimSpace(agente.Nombre)) && strings.TrimSpace(input.Agente) != "" {
		agenteRuntime.Nombre = strings.TrimSpace(input.Agente)
	}
	stepStart = time.Now()
	proyecto, err := s.store.GetProject(proyectoRef)
	if err != nil {
		return nil, err
	}
	prepareDebugf("BuildPrepare step=get_project proyecto=%s duration=%s", proyectoRef, time.Since(stepStart).Round(time.Millisecond))
	stepStart = time.Now()
	ultima, err := s.store.GetLastSession(agenteNombre, &proyecto.ID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == sql.ErrNoRows {
		ultima = nil
	}
	prepareDebugf("BuildPrepare step=get_last_session agente=%s duration=%s", agenteNombre, time.Since(stepStart).Round(time.Millisecond))
	stepStart = time.Now()
	conector, err := s.resolvePrepareConnector(agenteNombre, strings.TrimSpace(input.Conector), ultima)
	if err != nil {
		return nil, err
	}
	prepareDebugf("BuildPrepare step=resolve_connector conector=%s duration=%s", strings.TrimSpace(conector.Slug), time.Since(stepStart).Round(time.Millisecond))
	stepStart = time.Now()
	catalogo, err := s.store.ResolveGovernanceCatalogForContext(agente.Rol, &proyecto.ID, agente.Nombre)
	if err != nil {
		return nil, err
	}
	prepareDebugf("BuildPrepare step=resolve_governance duration=%s", time.Since(stepStart).Round(time.Millisecond))
	stepStart = time.Now()
	memoria, err := s.store.ListMemoryEntities(db.FiltroEntidadesMemoria{ProyectoID: &proyecto.ID})
	if err != nil {
		return nil, err
	}
	prepareDebugf("BuildPrepare step=list_memory count=%d duration=%s", len(memoria), time.Since(stepStart).Round(time.Millisecond))
	stepStart = time.Now()
	tareas, err := s.store.ListTasks(db.FiltroTareas{Agente: &agenteNombre, ProyectoID: &proyecto.ID})
	if err != nil {
		return nil, err
	}
	prepareDebugf("BuildPrepare step=list_tasks count=%d duration=%s", len(tareas), time.Since(stepStart).Round(time.Millisecond))
	stepStart = time.Now()
	propuestasPendientes, err := s.store.ListProjectPendingVotes(agenteNombre, proyecto.ID)
	if err != nil {
		return nil, err
	}
	prepareDebugf("BuildPrepare step=list_pending_votes count=%d duration=%s", len(propuestasPendientes), time.Since(stepStart).Round(time.Millisecond))
	stepStart = time.Now()
	// Resolución hexagonal de política de modelo (OP-HEX)
	modeloSolicitado := strings.TrimSpace(input.Modelo)
	razonamientoSolicitado := strings.TrimSpace(input.Razonamiento)
	perfilSolicitado := strings.TrimSpace(input.Perfil)
	var resolucionModelo *db.ResolucionModelo

	if s.modelPolicyProvider != nil && (modeloSolicitado == "" || razonamientoSolicitado == "") {
		agentePolicy := agenteNombre
		resolucion, err := s.modelPolicyProvider.ResolveModelPolicy(db.ResolverPoliticaInput{
			AgenteNombre: &agentePolicy,
			ProyectoSlug: proyecto.Slug,
			PerfilTarea:  perfilSolicitado,
		})
		if err == nil && resolucion != nil {
			resolucionModelo = resolucion
			if modeloSolicitado == "" {
				modeloSolicitado = resolucion.ModelSlug
			}
			if razonamientoSolicitado == "" {
				razonamientoSolicitado = resolucion.ReasoningEffort
			}
			if perfilSolicitado == "" {
				perfilSolicitado = resolucion.PerfilTarea
			}
		}
	}
	conectorRuntime := runtimeagente.ConnectorConfig{
		Slug:         strings.TrimSpace(conector.Slug),
		Nombre:       strings.TrimSpace(conector.Nombre),
		Transporte:   strings.TrimSpace(conector.Transporte),
		Comando:      strings.TrimSpace(conector.Comando),
		ArgsJSON:     strings.TrimSpace(conector.ArgsJSON),
		EnvJSON:      strings.TrimSpace(conector.EnvJSON),
		MetadataJSON: strings.TrimSpace(conector.MetadataJSON),
		Activo:       conector.Activo,
	}
	if !runtimeagente.ModeloCompatibleConConector(conectorRuntime, modeloSolicitado) {
		modeloSolicitado = ""
	}
	if ref := s.preferirConectorPoolLocalCompartido(agenteNombre, strings.TrimSpace(input.Conector), ultima, resolucionModelo); ref != "" {
		conector, err = s.store.GetConnector(ref)
		if err != nil {
			return nil, err
		}
	}

	prep, err := lanzamientoruntime.PrepararDesdeDatos(&agenteRuntime, proyecto, conector, ultima, modeloSolicitado, razonamientoSolicitado, perfilSolicitado)
	if err != nil {
		return nil, err
	}
	prepareDebugf("BuildPrepare step=prepare_runtime duration=%s", time.Since(stepStart).Round(time.Millisecond))
	stepStart = time.Now()
	bootstrapPrompt := buildBootstrapPrompt(&agenteRuntime, proyecto, prep.Plan, catalogo, memoria, tareas, propuestasPendientes)
	if prep.Plan != nil {
		prep.Plan.BootstrapPrompt = strings.TrimSpace(bootstrapPrompt)
		if err := runtimeagente.ApplyLaunchPromptMetadata(prep.Plan, conector.MetadataJSON); err != nil {
			return nil, fmt.Errorf("metadata_json inválido para '%s': %w", strings.TrimSpace(conector.Slug), err)
		}
	}
	prepareDebugf("BuildPrepare step=build_prompt duration=%s", time.Since(stepStart).Round(time.Millisecond))

	out := &PrepareOutput{
		Agente: agenteRuntime.Nombre,
		Rol:    agenteRuntime.Rol,
		Proyecto: ProjectBundle{
			ID:      proyecto.ID,
			Slug:    proyecto.Slug,
			Nombre:  proyecto.Nombre,
			RutaAbs: proyecto.RutaAbs,
		},
		Conector: ConnectorBundle{
			ID:         conector.ID,
			Slug:       conector.Slug,
			Nombre:     conector.Nombre,
			Transporte: conector.Transporte,
			Comando:    conector.Comando,
		},
		Politica:        s.loadPolicy(),
		Plan:            prep.Plan,
		BootstrapPrompt: bootstrapPrompt,
		Reglas:          catalogo.Reglas,
		Skills:          catalogo.Skills,
		Workflows:       catalogo.Workflows,
		Memoria:         memoria,
		Bootstrap:       prep.Bootstrap,
		ReanimarAt:      agente.ReanimarAt,
		MotivoPausa:     agente.MotivoPausa,
		EstadoCuota:     agente.EstadoCuota,
	}
	out.Politica.Modelo = prep.Plan.Modelo
	out.Politica.Razonamiento = prep.Plan.Razonamiento
	out.Politica.PerfilTarea = prep.Plan.PerfilTarea
	if ultima != nil {
		out.UltimaSesion = summarizeSession(ultima)
	}
	return out, nil
}

func prepareDebugf(format string, args ...any) {
	if !prepareDebugEnabled() {
		return
	}
	log.Printf("orquesta[prepare] "+format, args...)
}

func prepareDebugEnabled() bool {
	for _, key := range []string{"ORQUESTA_DEBUG_PREPARE", "ORQUESTA_DEBUG"} {
		value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
		switch value {
		case "1", "true", "yes", "on", "si", "sí":
			return true
		}
	}
	return false
}

func (s *Service) ProcessTick(input TickInput) (*TickOutput, error) {
	agenteNombre := strings.TrimSpace(input.Agente)
	proyectoRef := strings.TrimSpace(input.Proyecto)
	if agenteNombre == "" || proyectoRef == "" {
		return nil, fmt.Errorf("debes indicar agente y proyecto")
	}
	proyecto, err := s.store.GetProject(proyectoRef)
	if err != nil {
		return nil, err
	}
	sesionActiva, err := s.store.GetActiveSession(agenteNombre, &proyecto.ID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == sql.ErrNoRows {
		sesionActiva = nil
	}
	if sesionActiva != nil {
		upd := db.SesionUpdate{Heartbeat: true}
		if host := strings.TrimSpace(input.Host); host != "" {
			upd.Host = &host
		}
		if input.PID > 0 {
			upd.PID = &input.PID
		}
		if err := s.store.SaveActiveSession(agenteNombre, &proyecto.ID, upd); err != nil {
			return nil, err
		}
		if input.Finalizado && shouldAutoPauseForBudget(input.CuotaPct, input.Motivo) {
			if err := s.autoPause(agenteNombre, 60, "Auto-pausa por agotamiento: "+input.Motivo, "Detección de agotamiento en tick final: "+input.Motivo); err != nil {
				return nil, err
			}
		}
		if err := db.AckBootstrapRuntimeLease(input.AckStartOrderID, input.AckBootstrapOrderID, input.AckMailboxIDs, sesionActiva.ID, "agente_tick"); err != nil {
			return nil, err
		}
		sesionActiva, err = s.store.GetActiveSession(agenteNombre, &proyecto.ID)
		if err != nil {
			return nil, err
		}
	}
	out, err := s.buildTickOutput(agenteNombre, proyecto, sesionActiva, input.CuotaPct)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) resolvePrepareConnector(agente, conectorRef string, ultima *db.Sesion) (*db.Conector, error) {
	ref := strings.TrimSpace(conectorRef)
	if ref == "" && ultima != nil {
		if strings.TrimSpace(ultima.ConectorSlug) != "" &&
			runtimeagente.ConectorCompatibleConAgente(agente, strings.TrimSpace(ultima.ConectorSlug), "") {
			ref = strings.TrimSpace(ultima.ConectorSlug)
		} else if ultima.ConectorID != nil && *ultima.ConectorID > 0 &&
			runtimeagente.ConectorCompatibleConAgente(agente, "", strings.TrimSpace(ultima.Herramienta)) {
			ref = strconv.FormatInt(*ultima.ConectorID, 10)
		}
	}
	if ref == "" {
		ref = runtimeagente.ConectorPorDefectoAgente(agente)
	}
	conector, err := s.store.GetConnector(ref)
	if err != nil {
		if ref != runtimeagente.ConectorPorDefectoAgente(agente) {
			return nil, fmt.Errorf("no se pudo resolver el conector para %s: %w", strings.TrimSpace(agente), err)
		}
		return nil, err
	}
	return conector, nil
}

func (s *Service) preferirConectorPoolLocalCompartido(agente, conectorRef string, ultima *db.Sesion, resolucion *db.ResolucionModelo) string {
	if !runtimeagente.EsConectorFamiliaOllama(runtimeagente.ConectorPorDefectoAgente(agente), "") {
		return ""
	}
	if strings.TrimSpace(conectorRef) != "" {
		return ""
	}
	if ultima != nil && strings.TrimSpace(ultima.ConectorSlug) != "" {
		return ""
	}
	if resolucion == nil || strings.TrimSpace(resolucion.PoolSlug) == "" {
		return ""
	}
	pool, err := s.store.GetPool(strings.TrimSpace(resolucion.PoolSlug))
	if err != nil || pool == nil {
		return ""
	}
	meta := mapFromJSON(strings.TrimSpace(pool.MetadataJSON))
	if !strings.EqualFold(strings.TrimSpace(pool.Runtime), "ollama") {
		return ""
	}
	if !metadataBoolDefault(meta, "experimental_compat", true) && strings.TrimSpace(metadataString(meta, "conector_canonico")) == "" {
		return ""
	}
	if strings.EqualFold(strings.TrimSpace(metadataString(meta, "conector_canonico")), "ollama_pool_local") {
		return "ollama_pool_local"
	}
	if strings.EqualFold(strings.TrimSpace(pool.Plan), "local") {
		return "ollama_pool_local"
	}
	return ""
}

func mapFromJSON(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{}
	}
	return out
}

func metadataString(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	if value, ok := meta[strings.TrimSpace(key)].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func metadataBoolDefault(meta map[string]any, key string, fallback bool) bool {
	if meta == nil {
		return fallback
	}
	value, ok := meta[strings.TrimSpace(key)]
	if !ok {
		return fallback
	}
	if b, ok := value.(bool); ok {
		return b
	}
	return fallback
}

func (s *Service) loadPolicy() Policy {
	return Policy{
		ModoBucle:         s.configOrDefault("agent_loop_mode", "sticky"),
		IntervaloTickSeg:  s.configIntOrDefault("agent_tick_seconds", 30),
		ContinuarHasta:    "tarea_terminal_o_duda_real",
		ConsultarProyecto: true,
	}
}

func buildBootstrapPrompt(agente *db.Agente, proyecto *db.Proyecto, plan *runtimeagente.LaunchPlan, catalogo *db.GovernanceCatalog, memoria []*db.EntidadMemoria, tareas []*db.Tarea, propuestas []*db.Propuesta) string {
	if agente == nil || proyecto == nil {
		return ""
	}
	resumenProyecto := ""
	resumenGobernanza := ""
	if db.DB != nil {
		_, resumenProyecto = db.BuildProjectContextSummary(strings.TrimSpace(agente.Nombre), proyecto)
		_, resumenGobernanza = db.BuildGovernanceContextSummaryFromCatalog(catalogo)
	}
	return db.BuildLaunchBootstrapPrompt(agente, proyecto, plan, catalogo, memoria, tareas, propuestas, resumenProyecto, resumenGobernanza)
}

func (s *Service) buildTickOutput(agenteNombre string, proyecto *db.Proyecto, sesionActiva *db.Sesion, cuotaPct int) (*TickOutput, error) {
	if proyecto == nil {
		return nil, fmt.Errorf("proyecto obligatorio")
	}
	now := time.Now().UTC()
	politica := s.loadPolicy()
	row, asignaciones, tareas, err := s.buildOperationalRowContextForAgentCompact(agenteNombre, now)
	if err != nil {
		return nil, err
	}
	if sesionActiva == nil {
		sesionActiva = row.Sesion
	}
	asignadoAProyecto, proyectoAsignado := resolveActiveAssignmentFromAssignments(asignaciones, proyecto.ID)
	var (
		tareasActivas []LightItem
		tieneBloqueos bool
		tieneTrabajo  bool
	)
	for _, tarea := range tareas {
		if tarea == nil || tarea.Estado == db.TareaCompletada || tarea.Estado == db.TareaCancelada || tarea.Estado == db.TareaBacklog {
			continue
		}
		if tarea.Estado == db.TareaBloqueada {
			requiereIntervencion, err := s.blockedTaskNeedsIntervention(agenteNombre, proyecto.ID, sesionActiva, tarea)
			if err != nil {
				return nil, err
			}
			if requiereIntervencion {
				tieneBloqueos = true
			}
			continue
		}
		tareasActivas = append(tareasActivas, LightItem{ID: tarea.ID, Titulo: tarea.Titulo, Estado: string(tarea.Estado)})
		if tarea.Estado == db.TareaAsignada || tarea.Estado == db.TareaEnProgreso {
			tieneTrabajo = true
		}
	}
	agente := row.Agente
	pausarPorPresupuesto, motivoPresupuesto, err := s.shouldPauseByFreshBudget(agenteNombre)
	if err != nil {
		return nil, err
	}
	estadoOperativo := strings.TrimSpace(row.EstadoOperativo)
	detalleOperativo := strings.TrimSpace(row.DetalleOperativo)
	runtimeBloqueado := estadoOperativoBloqueaContinuidad(estadoOperativo)
	if tieneTrabajo && !runtimeBloqueado {
		bloqueado, detalle, err := s.runtimeBlockedFallback(agenteNombre, proyecto.ID)
		if err != nil {
			return nil, err
		}
		if bloqueado {
			runtimeBloqueado = true
			estadoOperativo = "bloqueado_por_runtime"
			detalleOperativo = firstNonEmpty(strings.TrimSpace(detalle), strings.TrimSpace(detalleOperativo))
		}
	}

	out := &TickOutput{
		Agente: agenteNombre,
		Proyecto: ProjectBundle{
			ID:      proyecto.ID,
			Slug:    proyecto.Slug,
			Nombre:  proyecto.Nombre,
			RutaAbs: proyecto.RutaAbs,
		},
		Politica:          politica,
		AsignadoAProyecto: asignadoAProyecto,
		ProyectoAsignado:  proyectoAsignado,
		TareasActivas:     tareasActivas,
		ConsumoDia:        agente.ConsumoDiaSegundos,
		LimiteDia:         agente.LimiteDiaSegundos,
		EstadoCuota:       agente.EstadoCuota,
		ReanimarAt:        agente.ReanimarAt,
		MotivoPausa:       agente.MotivoPausa,
		CuotaPct:          cuotaPct,
	}
	if sesionActiva != nil {
		out.SesionActiva = summarizeSession(sesionActiva)
	}

	switch {
	case pausarPorPresupuesto:
		out.AccionRecomendada = "pausar_por_cuota"
		out.DebePausar = true
		out.Motivo = motivoPresupuesto
	case strings.EqualFold(strings.TrimSpace(estadoOperativo), "bloqueado_por_cuota"):
		out.AccionRecomendada = "pausar_por_cuota"
		out.DebePausar = true
		out.Motivo = firstNonEmpty(strings.TrimSpace(detalleOperativo), "Worker bloqueado por cuota o enfriamiento activo.")
	case strings.TrimSpace(agente.EstadoCuota) != "" && agente.EstadoCuota != "activo":
		out.AccionRecomendada = "pausar_por_cuota"
		out.DebePausar = true
		if agente.ReanimarAt != nil {
			out.Motivo = fmt.Sprintf("Cuota agotada (%s). Reanimación programada para: %s. Motivo: %s", agente.EstadoCuota, agente.ReanimarAt.Format("15:04:05"), agente.MotivoPausa)
		} else {
			out.Motivo = "Cuota agotada o modo enfriamiento activo."
		}
	case tieneTrabajo && runtimeBloqueado:
		out.AccionRecomendada = "esperar_recuperacion_runtime"
		out.Motivo = firstNonEmpty(strings.TrimSpace(detalleOperativo), "Runtime no disponible; esperando recuperación automática")
	case !asignadoAProyecto && proyectoAsignado != "":
		out.AccionRecomendada = "pausar_y_reasignar"
		out.DebePausar = true
		out.Motivo = "La asignación activa del agente ha cambiado al proyecto " + proyectoAsignado
	case tieneBloqueos:
		out.AccionRecomendada = "pedir_intervencion"
		out.Motivo = "Hay tareas bloqueadas que requieren resolución"
	case tieneTrabajo:
		out.AccionRecomendada = "continuar_trabajo"
		out.Motivo = "Sigue trabajando hasta completar la tarea o detectar una duda real"
	default:
		pendientes, err := s.store.ListProjectPendingVotes(agenteNombre, proyecto.ID)
		if err != nil {
			return nil, err
		}
		var propuestasPendientes []LightItem
		for _, propuesta := range pendientes {
			propuestasPendientes = append(propuestasPendientes, LightItem{
				ID:     propuesta.ID,
				Codigo: propuesta.Codigo,
				Titulo: propuesta.Titulo,
				Estado: string(propuesta.Estado),
			})
		}
		out.PropuestasPendientes = propuestasPendientes
		if len(propuestasPendientes) > 0 {
			out.AccionRecomendada = "votar_propuestas_pendientes"
			out.Motivo = fmt.Sprintf("Hay %d propuestas pendientes de voto para este proyecto", len(propuestasPendientes))
			return out, nil
		}
		abiertas, err := s.store.ListProjectOpenProposals(proyecto.ID)
		if err != nil {
			return nil, err
		}
		var propuestasAbiertas []LightItem
		for _, propuesta := range abiertas {
			propuestasAbiertas = append(propuestasAbiertas, LightItem{
				ID:     propuesta.ID,
				Codigo: propuesta.Codigo,
				Titulo: propuesta.Titulo,
				Estado: string(propuesta.Estado),
			})
		}
		out.PropuestasAbiertas = propuestasAbiertas
		out.AccionRecomendada = "esperar_o_pedir_tarea"
		out.Motivo = "No hay tarea activa asignada en este proyecto"
	}
	return out, nil
}

func (s *Service) runtimeBlockedFallback(agenteNombre string, proyectoID int64) (bool, string, error) {
	agenteNombre = strings.TrimSpace(agenteNombre)
	if agenteNombre == "" || proyectoID <= 0 {
		return false, "", nil
	}
	handleFilter := agenteNombre
	handles, err := s.store.ListPassiveRuntimeHandles(&handleFilter)
	if err != nil {
		return false, "", err
	}
	var latestHandle *db.RuntimeHandle
	for _, handle := range handles {
		if handle == nil || handle.ProyectoID == nil || *handle.ProyectoID != proyectoID {
			continue
		}
		if latestHandle == nil || runtimeHandleMoment(handle).After(runtimeHandleMoment(latestHandle)) {
			latestHandle = handle
		}
		switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
		case "activo", "active", "pausado", "paused":
			return false, "", nil
		}
	}
	runtimes, err := s.store.ListRuntimes(db.FiltroRuntimes{Agente: &agenteNombre})
	if err != nil {
		return false, "", err
	}
	var latestRuntime *db.RuntimeInstance
	for _, runtime := range runtimes {
		if runtime == nil || runtime.ProyectoID == nil || *runtime.ProyectoID != proyectoID {
			continue
		}
		if latestRuntime == nil || runtimeMoment(runtime).After(runtimeMoment(latestRuntime)) {
			latestRuntime = runtime
		}
		switch strings.ToLower(strings.TrimSpace(runtime.LogicalState)) {
		case "activo", "active", "running", "iniciando", "starting", "esperando_io", "waiting_io", "pausado", "paused":
			return false, "", nil
		}
	}
	if latestHandle != nil {
		return true, firstNonEmpty(strings.TrimSpace(latestHandle.Estado), "handle no operativo"), nil
	}
	if latestRuntime != nil {
		return true, firstNonEmpty(strings.TrimSpace(latestRuntime.LogicalState), strings.TrimSpace(latestRuntime.ProcessState), "runtime no operativo"), nil
	}
	return false, "", nil
}

func estadoOperativoBloqueaContinuidad(estado string) bool {
	switch strings.TrimSpace(estado) {
	case "bloqueado_por_runtime", "mailbox_atascada", "caido", "atascado":
		return true
	default:
		return false
	}
}

func (s *Service) liveOperationalState(agenteNombre string) (string, string, error) {
	row, err := s.buildOperationalRowForAgent(agenteNombre, time.Now().UTC())
	if err != nil {
		return "", "", err
	}
	return strings.TrimSpace(row.EstadoOperativo), strings.TrimSpace(row.DetalleOperativo), nil
}

func resolveLiveOperationalState(loadRows func() ([]Row, error), agenteNombre string, timeout time.Duration) (string, string, error) {
	agenteNombre = strings.TrimSpace(agenteNombre)
	if agenteNombre == "" || loadRows == nil {
		return "", "", nil
	}
	if timeout <= 0 {
		timeout = liveOperationalStateTimeout
	}
	type result struct {
		rows []Row
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		rows, err := loadRows()
		ch <- result{rows: rows, err: err}
	}()
	select {
	case res := <-ch:
		if res.err != nil {
			return "", "", res.err
		}
		for _, row := range res.rows {
			if row.Agente == nil || !strings.EqualFold(strings.TrimSpace(row.Agente.Nombre), agenteNombre) {
				continue
			}
			return strings.TrimSpace(row.EstadoOperativo), strings.TrimSpace(row.DetalleOperativo), nil
		}
		return "", "", nil
	case <-time.After(timeout):
		return "", "", nil
	}
}

func (s *Service) blockedTaskNeedsIntervention(agenteNombre string, proyectoID int64, sesionActiva *db.Sesion, tarea *db.Tarea) (bool, error) {
	if tarea == nil || tarea.Estado != db.TareaBloqueada {
		return false, nil
	}
	motivo, err := db.MotivoBloqueoActivoTarea(tarea.ID)
	if err != nil {
		return true, err
	}
	agente, err := s.store.GetAgent(agenteNombre)
	if err != nil {
		return true, err
	}
	asignaciones, err := s.store.ListAssignments(db.FiltroAsignaciones{Agente: &agenteNombre, ProyectoID: &proyectoID})
	if err != nil {
		return true, err
	}
	asignadoAProyecto := false
	for _, asignacion := range asignaciones {
		if asignacion != nil && asignacion.Estado == db.AsignacionActiva && asignacion.ProyectoID == proyectoID {
			asignadoAProyecto = true
			break
		}
	}
	return BloqueoAutonomiaRequiereIntervencion(agenteNombre, proyectoID, asignadoAProyecto, sesionActiva, agente, strings.TrimSpace(motivo)), nil
}

func bloqueoAutonomiaAgenteRecuperable(agente, bloqueadoPor, motivo string) bool {
	agente = strings.TrimSpace(agente)
	bloqueadoPor = strings.TrimSpace(bloqueadoPor)
	motivo = strings.TrimSpace(motivo)
	if agente == "" || motivo == "" {
		return false
	}
	if strings.HasPrefix(motivo, "Agente "+agente+" en estado ") {
		return true
	}
	if strings.EqualFold(bloqueadoPor, agente) && strings.HasPrefix(motivo, "Agente degradado:") {
		return true
	}
	motivoLower := strings.ToLower(motivo)
	if strings.EqualFold(bloqueadoPor, agente) && strings.HasPrefix(motivoLower, strings.ToLower("Sobrecarga operativa:")) {
		return true
	}
	return false
}

func BloqueoAutonomiaRequiereIntervencion(agenteNombre string, proyectoID int64, asignadoAProyecto bool, sesionActiva *db.Sesion, agente *db.Agente, motivo string) bool {
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return true
	}
	if sesionActiva != nil &&
		sesionActiva.ProyectoID != nil &&
		*sesionActiva.ProyectoID == proyectoID &&
		strings.EqualFold(strings.TrimSpace(sesionActiva.Agente), strings.TrimSpace(agenteNombre)) &&
		bloqueoAutonomiaAgenteRecuperable(agenteNombre, agenteNombre, motivo) {
		return false
	}
	if !asignadoAProyecto || !bloqueoAutonomiaRecuperableSinSesionActiva(agenteNombre, motivo) {
		return true
	}
	if agente != nil {
		estadoCuota := strings.ToLower(strings.TrimSpace(agente.EstadoCuota))
		if estadoCuota != "" && estadoCuota != "activo" {
			return false
		}
	}
	return false
}

func bloqueoAutonomiaRecuperableSinSesionActiva(agente, motivo string) bool {
	if bloqueoAutonomiaAgenteRecuperable(agente, agente, motivo) {
		return true
	}
	motivoLower := strings.ToLower(strings.TrimSpace(motivo))
	switch {
	case strings.Contains(motivoLower, "degradación operativa sin relevo sano"):
		return true
	case strings.Contains(motivoLower, "degradacion operativa sin relevo sano"):
		return true
	default:
		return false
	}
}

func (s *Service) shouldPauseByFreshBudget(agente string) (bool, string, error) {
	p, _, err := s.store.GetLatestAgentBudget(strings.TrimSpace(agente))
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", err
	}
	if p == nil || !db.PresupuestoSesionFresco(p) {
		return false, "", nil
	}
	ev, err := db.EvaluarPresupuestoSesion(p)
	if err != nil {
		return false, "", err
	}
	if ev == nil || !ev.DebeHandoff {
		return false, "", nil
	}
	motivo := strings.TrimSpace(ev.Motivo)
	if motivo == "" {
		motivo = "presupuesto crítico"
	}
	return true, "Presupuesto crítico. Pausa y relevo recomendados: " + motivo, nil
}

func (s *Service) resolveActiveAssignment(agente string, proyectoID int64) (bool, string, error) {
	estado := db.AsignacionActiva
	asignaciones, err := s.store.ListAssignments(db.FiltroAsignaciones{Agente: &agente, Estado: &estado})
	if err != nil {
		return false, "", err
	}
	asignadoAProyecto, proyectoAsignado := resolveActiveAssignmentFromAssignments(asignaciones, proyectoID)
	return asignadoAProyecto, proyectoAsignado, nil
}

func resolveActiveAssignmentFromAssignments(asignaciones []*db.Asignacion, proyectoID int64) (bool, string) {
	if len(asignaciones) == 0 {
		return false, ""
	}
	for _, asignacion := range asignaciones {
		if asignacion != nil && asignacion.Estado == db.AsignacionActiva && asignacion.ProyectoID == proyectoID {
			return true, strings.TrimSpace(asignacion.ProyectoSlug)
		}
	}
	for _, asignacion := range asignaciones {
		if asignacion == nil || asignacion.Estado != db.AsignacionActiva {
			continue
		}
		return false, strings.TrimSpace(asignacion.ProyectoSlug)
	}
	return false, ""
}

func shouldAutoPauseForBudget(cuotaPct int, motivo string) bool {
	motivoLower := strings.ToLower(strings.TrimSpace(motivo))
	return cuotaPct < 5 || strings.Contains(motivoLower, "token") || strings.Contains(motivoLower, "cuota") || strings.Contains(motivoLower, "rate limit")
}

func summarizeSession(sesion *db.Sesion) *SessionBundle {
	if sesion == nil {
		return nil
	}
	return &SessionBundle{
		ID:                 sesion.ID,
		Estado:             sesion.Estado,
		CWD:                sesion.CWD,
		Herramienta:        sesion.Herramienta,
		ExternalSessionID:  sesion.ExternalSessionID,
		ResumePayloadJSON:  sesion.ResumePayloadJSON,
		ResumenContinuidad: sesion.ResumenContinuidad,
		Branch:             sesion.Branch,
	}
}

func (s *Service) configOrDefault(clave, fallback string) string {
	valor, err := s.store.ConfigGet(clave)
	if err != nil || strings.TrimSpace(valor) == "" {
		return fallback
	}
	return valor
}

func (s *Service) configIntOrDefault(clave string, fallback int) int {
	valor, err := s.store.ConfigGet(clave)
	if err != nil {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(valor))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func (s *Service) autoPause(nombre string, minutos int, motivoPausa, detalle string) error {
	if err := s.store.PauseAgent(nombre, minutos, motivoPausa); err != nil {
		return err
	}
	s.store.Audit(nombre, "auto_pausa", "sistema", 0, detalle)
	return nil
}
