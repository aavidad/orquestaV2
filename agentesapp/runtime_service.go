package agentesapp

import (
	"database/sql"
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
	prep, err := lanzamientoruntime.PrepararDesdeDatos(&agenteRuntime, proyecto, conector, ultima, strings.TrimSpace(input.Modelo), strings.TrimSpace(input.Razonamiento), strings.TrimSpace(input.Perfil))
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
	return s.buildTickOutput(agenteNombre, proyecto, sesionActiva, input.CuotaPct)
}

func (s *Service) resolvePrepareConnector(agente, conectorRef string, ultima *db.Sesion) (*db.Conector, error) {
	ref := strings.TrimSpace(conectorRef)
	if ref == "" && ultima != nil {
		if strings.TrimSpace(ultima.ConectorSlug) != "" {
			ref = strings.TrimSpace(ultima.ConectorSlug)
		} else if ultima.ConectorID != nil && *ultima.ConectorID > 0 {
			ref = strconv.FormatInt(*ultima.ConectorID, 10)
		}
	}
	if ref == "" {
		ref = "codex-cli"
	}
	conector, err := s.store.GetConnector(ref)
	if err != nil {
		if ref != "codex-cli" {
			return nil, fmt.Errorf("no se pudo resolver el conector para %s: %w", strings.TrimSpace(agente), err)
		}
		return nil, err
	}
	return conector, nil
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
	politica := s.loadPolicy()
	asignadoAProyecto, proyectoAsignado, err := s.resolveActiveAssignment(agenteNombre, proyecto.ID)
	if err != nil {
		return nil, err
	}
	tareas, err := s.store.ListTasks(db.FiltroTareas{Agente: &agenteNombre, ProyectoID: &proyecto.ID})
	if err != nil {
		return nil, err
	}
	var (
		tareasActivas []LightItem
		tieneBloqueos bool
		tieneTrabajo  bool
	)
	for _, tarea := range tareas {
		if tarea == nil || tarea.Estado == db.TareaCompletada || tarea.Estado == db.TareaCancelada || tarea.Estado == db.TareaBacklog {
			continue
		}
		tareasActivas = append(tareasActivas, LightItem{ID: tarea.ID, Titulo: tarea.Titulo, Estado: string(tarea.Estado)})
		if tarea.Estado == db.TareaBloqueada {
			tieneBloqueos = true
		}
		if tarea.Estado == db.TareaAsignada || tarea.Estado == db.TareaEnProgreso {
			tieneTrabajo = true
		}
	}
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
	agente, err := s.store.GetAgent(agenteNombre)
	if err != nil {
		return nil, err
	}
	pausarPorPresupuesto, motivoPresupuesto, err := s.shouldPauseByFreshBudget(agenteNombre)
	if err != nil {
		return nil, err
	}

	out := &TickOutput{
		Agente: agenteNombre,
		Proyecto: ProjectBundle{
			ID:      proyecto.ID,
			Slug:    proyecto.Slug,
			Nombre:  proyecto.Nombre,
			RutaAbs: proyecto.RutaAbs,
		},
		Politica:             politica,
		AsignadoAProyecto:    asignadoAProyecto,
		ProyectoAsignado:     proyectoAsignado,
		TareasActivas:        tareasActivas,
		PropuestasPendientes: propuestasPendientes,
		PropuestasAbiertas:   propuestasAbiertas,
		ConsumoDia:           agente.ConsumoDiaSegundos,
		LimiteDia:            agente.LimiteDiaSegundos,
		EstadoCuota:          agente.EstadoCuota,
		ReanimarAt:           agente.ReanimarAt,
		MotivoPausa:          agente.MotivoPausa,
		CuotaPct:             cuotaPct,
	}
	if sesionActiva != nil {
		out.SesionActiva = summarizeSession(sesionActiva)
	}

	switch {
	case pausarPorPresupuesto:
		out.AccionRecomendada = "pausar_por_cuota"
		out.DebePausar = true
		out.Motivo = motivoPresupuesto
	case strings.TrimSpace(agente.EstadoCuota) != "" && agente.EstadoCuota != "activo":
		out.AccionRecomendada = "pausar_por_cuota"
		out.DebePausar = true
		if agente.ReanimarAt != nil {
			out.Motivo = fmt.Sprintf("Cuota agotada (%s). Reanimación programada para: %s. Motivo: %s", agente.EstadoCuota, agente.ReanimarAt.Format("15:04:05"), agente.MotivoPausa)
		} else {
			out.Motivo = "Cuota agotada o modo enfriamiento activo."
		}
	case !asignadoAProyecto && proyectoAsignado != "":
		out.AccionRecomendada = "pausar_y_reasignar"
		out.DebePausar = true
		out.Motivo = "La asignación activa del agente ha cambiado al proyecto " + proyectoAsignado
	case len(propuestasPendientes) > 0:
		out.AccionRecomendada = "votar_propuestas_pendientes"
		out.Motivo = fmt.Sprintf("Hay %d propuestas pendientes de voto para este proyecto", len(propuestasPendientes))
	case tieneBloqueos:
		out.AccionRecomendada = "pedir_intervencion"
		out.Motivo = "Hay tareas bloqueadas que requieren resolución"
	case tieneTrabajo:
		out.AccionRecomendada = "continuar_trabajo"
		out.Motivo = "Sigue trabajando hasta completar la tarea o detectar una duda real"
	default:
		out.AccionRecomendada = "esperar_o_pedir_tarea"
		out.Motivo = "No hay tarea activa asignada en este proyecto"
	}
	return out, nil
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
	if len(asignaciones) == 0 {
		return false, "", nil
	}
	for _, asignacion := range asignaciones {
		if asignacion != nil && asignacion.ProyectoID == proyectoID {
			return true, asignacion.ProyectoSlug, nil
		}
	}
	return false, asignaciones[0].ProyectoSlug, nil
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
