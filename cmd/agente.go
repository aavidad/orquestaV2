/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/agentesapp"
	"orquesta/db"
	"orquesta/internal/lanzamientoruntime"
	"orquesta/runtimeagente"
)

type agenteBootstrapAck struct {
	StartOrderID     int64
	BootstrapOrderID int64
	MailboxIDs       []int64
}

var agenteCmd = &cobra.Command{
	Use:   "agente",
	Short: "Operaciones de orquestación para agentes",
}

func int64FlagOpt(cmd *cobra.Command, name string) (*int64, error) {
	value, err := cmd.Flags().GetInt64(name)
	if err != nil {
		return nil, err
	}
	if !cmd.Flags().Changed(name) {
		return nil, nil
	}
	return &value, nil
}

func debeAutoPausarPorAgotamiento(cuotaPct int, motivo string) bool {
	motivoLower := strings.ToLower(strings.TrimSpace(motivo))
	return cuotaPct < 5 ||
		strings.Contains(motivoLower, "token") ||
		strings.Contains(motivoLower, "cuota") ||
		strings.Contains(motivoLower, "rate limit")
}

func registrarAutoPausaLocal(nombre string, minutos int, motivoPausa, detalle string) error {
	if err := db.PausarAgente(nombre, minutos, motivoPausa); err != nil {
		return err
	}
	db.Audit(nombre, "auto_pausa", "sistema", 0, detalle)
	return nil
}

func agenteModoRecuperacionLocalExplicito() bool {
	return strings.TrimSpace(os.Getenv("ORQUESTA_FORCE_LOCAL_DB")) == "1"
}

func agenteErrorServerFirst() error {
	return serverFirstCommandError("agente")
}

type proyectoBundle struct {
	ID      int64  `json:"id"`
	Slug    string `json:"slug"`
	Nombre  string `json:"nombre"`
	RutaAbs string `json:"ruta_abs"`
}

type conectorBundle struct {
	ID         int64  `json:"id"`
	Slug       string `json:"slug"`
	Nombre     string `json:"nombre"`
	Transporte string `json:"transporte"`
	Comando    string `json:"comando"`
}

type sesionBundle struct {
	ID                 int64  `json:"id"`
	Estado             string `json:"estado"`
	CWD                string `json:"cwd"`
	Herramienta        string `json:"herramienta"`
	ExternalSessionID  string `json:"external_session_id"`
	ResumePayloadJSON  string `json:"resume_payload_json"`
	ResumenContinuidad string `json:"resumen_continuidad"`
	Branch             string `json:"branch"`
}

type politicaAgente struct {
	ModoBucle         string `json:"modo_bucle"`
	IntervaloTickSeg  int    `json:"intervalo_tick_seg"`
	ContinuarHasta    string `json:"continuar_hasta"`
	ConsultarProyecto bool   `json:"consultar_proyecto"`
	Modelo            string `json:"modelo,omitempty"`
	Razonamiento      string `json:"razonamiento,omitempty"`
	PerfilTarea       string `json:"perfil_tarea,omitempty"`
}

type agentePrepararOutput struct {
	Agente          string                       `json:"agente"`
	Rol             string                       `json:"rol"`
	Proyecto        proyectoBundle               `json:"proyecto"`
	Conector        conectorBundle               `json:"conector"`
	Politica        politicaAgente               `json:"politica"`
	UltimaSesion    *sesionBundle                `json:"ultima_sesion,omitempty"`
	Plan            *runtimeagente.LaunchPlan    `json:"plan"`
	BootstrapPrompt string                       `json:"bootstrap_prompt,omitempty"`
	Reglas          []*db.Regla                  `json:"reglas"`
	Skills          []*db.Skill                  `json:"skills"`
	Workflows       []*db.Workflow               `json:"workflows"`
	Memoria         []*db.EntidadMemoria         `json:"memoria,omitempty"`
	Bootstrap       *agenteBootstrapRuntimeState `json:"bootstrap,omitempty"`
	ReanimarAt      *time.Time                   `json:"reanimar_at,omitempty"`
	MotivoPausa     string                       `json:"motivo_pausa,omitempty"`
	EstadoCuota     string                       `json:"estado_cuota,omitempty"`
}

type itemLigero struct {
	ID     int64  `json:"id"`
	Codigo string `json:"codigo,omitempty"`
	Titulo string `json:"titulo"`
	Estado string `json:"estado,omitempty"`
}

type agenteTickOutput struct {
	Agente               string         `json:"agente"`
	Proyecto             proyectoBundle `json:"proyecto"`
	Politica             politicaAgente `json:"politica"`
	SesionActiva         *sesionBundle  `json:"sesion_activa,omitempty"`
	AsignadoAProyecto    bool           `json:"asignado_a_proyecto"`
	ProyectoAsignado     string         `json:"proyecto_asignado,omitempty"`
	AccionRecomendada    string         `json:"accion_recomendada"`
	DebePausar           bool           `json:"debe_pausar"`
	Motivo               string         `json:"motivo,omitempty"`
	TareasActivas        []itemLigero   `json:"tareas_activas,omitempty"`
	PropuestasPendientes []itemLigero   `json:"propuestas_pendientes,omitempty"`
	PropuestasAbiertas   []itemLigero   `json:"propuestas_abiertas,omitempty"`
	ConsumoDia           int            `json:"consumo_dia_segundos"`
	LimiteDia            int            `json:"limite_dia_segundos"`
	EstadoCuota          string         `json:"estado_cuota"`
	ReanimarAt           *time.Time     `json:"reanimar_at,omitempty"`
	MotivoPausa          string         `json:"motivo_pausa,omitempty"`
	CuotaPct             int            `json:"cuota_pct,omitempty"`
}

type decisionTickAutonomiaInput struct {
	AsignadoAProyecto     bool
	ProyectoAsignado      string
	EsSupervisorOperativo bool
	SupervisorOperativo   *db.Agente
	PropuestasPendientes  int
	TieneBloqueos         bool
	TieneTrabajo          bool
	BloqueadoPorRuntime   bool
	MotivoBloqueoRuntime  string
	PausarPorPresupuesto  bool
	MotivoPresupuesto     string
	EstadoCuota           string
	BloqueadoPorCuota     bool
	MotivoBloqueoCuota    string
	ReanimarAt            *time.Time
	MotivoPausa           string
	AgenteNombre          string
}

func resolverAccionTickAutonomia(in decisionTickAutonomiaInput) (accion string, debePausar bool, motivo string) {
	switch {
	case in.PausarPorPresupuesto:
		return "pausar_por_cuota", true, strings.TrimSpace(in.MotivoPresupuesto)
	case in.BloqueadoPorCuota:
		motivo = strings.TrimSpace(in.MotivoBloqueoCuota)
		if motivo != "" && agenteMotivoPausaOperativa(motivo) {
			return "pausar_por_cuota", true, "Pausa operativa: " + motivo
		}
		if motivo == "" {
			motivo = "Worker bloqueado por cuota o enfriamiento activo."
		}
		return "pausar_por_cuota", true, motivo
	case strings.TrimSpace(in.EstadoCuota) != "" && strings.TrimSpace(in.EstadoCuota) != "activo":
		prefijo := "Cuota agotada"
		if agenteMotivoPausaOperativa(in.MotivoPausa) {
			prefijo = "Pausa operativa"
		}
		if in.ReanimarAt != nil {
			return "pausar_por_cuota", true, fmt.Sprintf("%s (%s). Reanimación programada para: %s. Motivo: %s",
				prefijo, in.EstadoCuota, in.ReanimarAt.Format("15:04:05"), in.MotivoPausa)
		}
		if agenteMotivoPausaOperativa(in.MotivoPausa) {
			return "pausar_por_cuota", true, "Pausa operativa o modo enfriamiento activo."
		}
		return "pausar_por_cuota", true, "Cuota agotada o modo enfriamiento activo."
	case in.TieneTrabajo && in.BloqueadoPorRuntime:
		motivo = strings.TrimSpace(in.MotivoBloqueoRuntime)
		if motivo == "" {
			motivo = "Runtime no disponible; esperando recuperación automática"
		}
		return "esperar_recuperacion_runtime", false, motivo
	case !in.AsignadoAProyecto && strings.TrimSpace(in.ProyectoAsignado) != "":
		return "pausar_y_reasignar", true, "La asignación activa del agente ha cambiado al proyecto " + strings.TrimSpace(in.ProyectoAsignado)
	case in.EsSupervisorOperativo:
		if in.SupervisorOperativo != nil && !strings.EqualFold(strings.TrimSpace(in.SupervisorOperativo.Nombre), strings.TrimSpace(in.AgenteNombre)) {
			return "supervisar_proyecto", false, "Debes asumir el relevo temporal de la orquestación del proyecto."
		}
		return "supervisar_proyecto", false, "Eres el supervisor operativo del proyecto y debes coordinar el siguiente frente útil."
	case in.PropuestasPendientes > 0:
		return "votar_propuestas_pendientes", false, fmt.Sprintf("Hay %d propuestas pendientes de voto para este proyecto", in.PropuestasPendientes)
	case in.TieneBloqueos:
		return "pedir_intervencion", false, "Hay tareas bloqueadas que requieren resolución"
	case in.TieneTrabajo:
		return "continuar_trabajo", false, "Sigue trabajando hasta completar la tarea o detectar una duda real"
	default:
		return "esperar_o_pedir_tarea", false, "No hay tarea activa asignada en este proyecto"
	}
}

func estadoOperativoBloqueaContinuidad(estado string) bool {
	switch strings.TrimSpace(estado) {
	case "bloqueado_por_runtime", "mailbox_atascada", "caido", "atascado":
		return true
	default:
		return false
	}
}

func construirAgenteTickOutput(agenteNombre string, proyecto *db.Proyecto, sesionActiva *db.Sesion, cuotaPct int) (agenteTickOutput, error) {
	return construirAgenteTickOutputConSnapshot(agenteNombre, proyecto, sesionActiva, cuotaPct, nil)
}

func construirAgenteTickOutputConSnapshot(agenteNombre string, proyecto *db.Proyecto, sesionActiva *db.Sesion, cuotaPct int, snapshot *autonomiaBatchSnapshot) (agenteTickOutput, error) {
	var out agenteTickOutput
	if proyecto == nil {
		return out, fmt.Errorf("proyecto obligatorio")
	}
	if snapshot == nil {
		snapshot = &autonomiaBatchSnapshot{
			asignacionesByAgent:        map[string][]*db.Asignacion{},
			tareasByAgentProject:       map[string][]*db.Tarea{},
			pendingVotesByAgentProject: map[string][]*db.Propuesta{},
			openProposalsByProject:     map[int64][]*db.Propuesta{},
			politicasByProject:         map[int64]*db.ProyectoAutonomia{},
			activeProjectByAgent:       map[string]int64{},
			activeHandleByAgentProject: map[string]bool{},
			supervisorByProject:        map[int64]*db.Agente{},
			supervisorLoaded:           map[int64]struct{}{},
			pauseByAgent:               map[string]autonomiaBudgetPauseDecision{},
			agentesByName:              map[string]*db.Agente{},
			operationalStateByAgent:    map[string]string{},
			operationalDetailByAgent:   map[string]string{},
		}
	}
	start := time.Now()
	logStep := func(step string, since time.Time) {
		if !autonomiaTickDebugEnabled() {
			return
		}
		duration := time.Since(since).Round(time.Millisecond)
		if duration < 250*time.Millisecond {
			return
		}
		log.Printf("orquesta[autonomia-tick] agente=%s proyecto=%s step=%s duration=%s", strings.TrimSpace(agenteNombre), strings.TrimSpace(proyecto.Slug), step, duration)
	}

	politica := cargarPoliticaAgente()
	stepStart := time.Now()
	asignadoAProyecto, proyectoAsignado, err := snapshot.assignment(agenteNombre, proyecto.ID)
	if err != nil {
		return out, err
	}
	logStep("resolver_asignacion", stepStart)

	stepStart = time.Now()
	agente, err := snapshot.agent(agenteNombre)
	if err != nil {
		return out, err
	}
	logStep("get_agente", stepStart)

	stepStart = time.Now()
	tareas, err := snapshot.tasks(agenteNombre, proyecto.ID)
	if err != nil {
		return out, err
	}
	logStep("listar_tareas", stepStart)
	var tareasActivas []itemLigero
	var tieneBloqueos bool
	var tieneTrabajo bool
	for _, tarea := range tareas {
		if tarea.Estado == db.TareaCompletada || tarea.Estado == db.TareaCancelada || tarea.Estado == db.TareaBacklog {
			continue
		}
		tareasActivas = append(tareasActivas, itemLigero{
			ID:     tarea.ID,
			Titulo: tarea.Titulo,
			Estado: string(tarea.Estado),
		})
		if tarea.Estado == db.TareaBloqueada {
			requiereIntervencion, err := tareaBloqueadaRequiereIntervencion(snapshot, agenteNombre, proyecto.ID, agente, tarea, asignadoAProyecto)
			if err != nil {
				return out, err
			}
			if requiereIntervencion {
				tieneBloqueos = true
			}
		}
		if tarea.Estado == db.TareaAsignada || tarea.Estado == db.TareaEnProgreso {
			tieneTrabajo = true
		}
	}

	stepStart = time.Now()
	pendientes, err := snapshot.pendingProjectVotes(agenteNombre, proyecto.ID)
	if err != nil {
		return out, err
	}
	logStep("propuestas_pendientes", stepStart)
	var propuestasPendientes []itemLigero
	for _, propuesta := range pendientes {
		propuestasPendientes = append(propuestasPendientes, itemLigero{
			ID:     propuesta.ID,
			Codigo: propuesta.Codigo,
			Titulo: propuesta.Titulo,
			Estado: string(propuesta.Estado),
		})
	}

	stepStart = time.Now()
	abiertas, err := snapshot.openProjectProposals(proyecto.ID)
	if err != nil {
		return out, err
	}
	logStep("propuestas_abiertas", stepStart)
	var propuestasAbiertas []itemLigero
	for _, propuesta := range abiertas {
		propuestasAbiertas = append(propuestasAbiertas, itemLigero{
			ID:     propuesta.ID,
			Codigo: propuesta.Codigo,
			Titulo: propuesta.Titulo,
			Estado: string(propuesta.Estado),
		})
	}

	out = agenteTickOutput{
		Agente:               agenteNombre,
		Proyecto:             proyectoBundle{ID: proyecto.ID, Slug: proyecto.Slug, Nombre: proyecto.Nombre, RutaAbs: proyecto.RutaAbs},
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
		out.SesionActiva = resumirSesion(sesionActiva)
	}
	stepStart = time.Now()
	esSupervisorOperativo, supervisorOperativo, err := snapshot.supervisorOperativo(proyecto.ID, agenteNombre)
	if err != nil {
		return out, err
	}
	logStep("supervisor_operativo", stepStart)
	stepStart = time.Now()
	pausarPorPresupuesto, motivoPresupuesto, err := snapshot.budgetPause(agenteNombre)
	if err != nil {
		return out, err
	}
	logStep("presupuesto_visible", stepStart)
	stepStart = time.Now()
	estadoOperativo, detalleOperativo, err := snapshot.operationalState(agenteNombre)
	if err != nil {
		return out, err
	}
	logStep("estado_operativo", stepStart)

	out.AccionRecomendada, out.DebePausar, out.Motivo = resolverAccionTickAutonomia(decisionTickAutonomiaInput{
		AsignadoAProyecto:     asignadoAProyecto,
		ProyectoAsignado:      proyectoAsignado,
		EsSupervisorOperativo: esSupervisorOperativo,
		SupervisorOperativo:   supervisorOperativo,
		PropuestasPendientes:  len(propuestasPendientes),
		TieneBloqueos:         tieneBloqueos,
		TieneTrabajo:          tieneTrabajo,
		BloqueadoPorRuntime:   estadoOperativoBloqueaContinuidad(estadoOperativo),
		MotivoBloqueoRuntime:  detalleOperativo,
		PausarPorPresupuesto:  pausarPorPresupuesto,
		MotivoPresupuesto:     motivoPresupuesto,
		EstadoCuota:           agente.EstadoCuota,
		BloqueadoPorCuota:     strings.EqualFold(strings.TrimSpace(estadoOperativo), "bloqueado_por_cuota"),
		MotivoBloqueoCuota:    detalleOperativo,
		ReanimarAt:            agente.ReanimarAt,
		MotivoPausa:           agente.MotivoPausa,
		AgenteNombre:          agenteNombre,
	})
	if autonomiaTickDebugEnabled() {
		duration := time.Since(start).Round(time.Millisecond)
		if duration >= 250*time.Millisecond {
			log.Printf("orquesta[autonomia-tick] agente=%s proyecto=%s total=%s accion=%s", strings.TrimSpace(agenteNombre), strings.TrimSpace(proyecto.Slug), duration, strings.TrimSpace(out.AccionRecomendada))
		}
	}

	return out, nil
}

func tareaBloqueadaRequiereIntervencion(snapshot *autonomiaBatchSnapshot, agenteNombre string, proyectoID int64, agente *db.Agente, tarea *db.Tarea, asignadoAProyecto bool) (bool, error) {
	if tarea == nil || tarea.Estado != db.TareaBloqueada {
		return false, nil
	}
	motivo, err := db.MotivoBloqueoActivoTarea(tarea.ID)
	if err != nil {
		return true, err
	}
	activoEnProyecto := false
	if snapshot != nil && strings.TrimSpace(agenteNombre) != "" && proyectoID > 0 {
		activoEnProyecto, err = snapshot.autonomiaActivoEnProyecto(agenteNombre, proyectoID, agente)
		if err != nil {
			return true, err
		}
	}
	if activoEnProyecto {
		asignadoAProyecto = true
	}
	return agentesapp.BloqueoAutonomiaRequiereIntervencion(agenteNombre, proyectoID, asignadoAProyecto, nil, agente, strings.TrimSpace(motivo)), nil
}

func autonomiaTickDebugEnabled() bool {
	base := parseBoolDebug(strings.TrimSpace(os.Getenv("ORQUESTA_DEBUG")), false)
	return parseBoolDebug(strings.TrimSpace(os.Getenv("ORQUESTA_DEBUG_CONTROL_PLANE")), base)
}

func agenteMotivoPausaOperativa(motivo string) bool {
	motivo = strings.ToLower(strings.TrimSpace(motivo))
	return strings.Contains(motivo, "runtime_panic") || strings.Contains(motivo, "runtime_crash")
}

func agenteDebePausarPorPresupuesto(nombre string) (bool, string, error) {
	p, _, err := db.UltimoPresupuestoAgente(nombre)
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

func agenteDebePausarPorPresupuestoVisible(agente *db.Agente) (bool, string) {
	if agente == nil || agente.PresupuestoStale {
		return false, ""
	}
	switch strings.ToLower(strings.TrimSpace(agente.PresupuestoEstado)) {
	case "agotado":
		return true, "Presupuesto crítico. Pausa y relevo recomendados: presupuesto agotado"
	case "handoff_preventivo":
		motivo := "cuota restante baja"
		if agente.CuotaRestantePct != nil {
			motivo = fmt.Sprintf("cuota restante visible %d%%", *agente.CuotaRestantePct)
			if ventana := strings.TrimSpace(agente.PresupuestoVentana); ventana != "" {
				motivo += " en " + ventana
			}
		}
		return true, "Presupuesto crítico. Pausa y relevo recomendados: " + motivo
	default:
		return false, ""
	}
}

func resumeContextNoVacio(resume runtimeagente.ResumeContext) bool {
	return lanzamientoruntime.ResumeContextNoVacio(resume)
}

var agentePrepararCmd = &cobra.Command{
	Use:   "preparar <agente>",
	Short: "Resuelve el bundle de arranque/reanudación para un agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agenteNombre := args[0]
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		conectorRef, _ := cmd.Flags().GetString("conector")
		modelo, _ := cmd.Flags().GetString("modelo")
		razonamiento, _ := cmd.Flags().GetString("razonamiento")
		perfilTarea, _ := cmd.Flags().GetString("perfil")
		jsonOut, _ := cmd.Flags().GetBool("json")
		campo, _ := cmd.Flags().GetString("campo")

		if strings.TrimSpace(proyectoRef) == "" {
			return fmt.Errorf("debes indicar --proyecto")
		}

		params := url.Values{
			"agente":   []string{agenteNombre},
			"proyecto": []string{proyectoRef},
		}
		if conectorRef != "" {
			params.Set("conector", conectorRef)
		}
		if modelo != "" {
			params.Set("modelo", modelo)
		}
		if razonamiento != "" {
			params.Set("razonamiento", razonamiento)
		}
		if perfilTarea != "" {
			params.Set("perfil", perfilTarea)
		}
		var out agentePrepararOutput
		if ok, err := apiGetQuery("/api/agente/preparar", params, &out); err != nil {
			return err
		} else if ok {
			if strings.TrimSpace(campo) != "" {
				v, err := valorCampoAgentePreparar(out, campo)
				if err != nil {
					return err
				}
				fmt.Println(v)
				return nil
			}
			return imprimirAgentePreparar(out, jsonOut)
		} else if !agenteModoRecuperacionLocalExplicito() {
			return agenteErrorServerFirst()
		}
		return serverFirstCommandError("agente preparar")
	},
}

var agenteTickCmd = &cobra.Command{
	Use:   "tick <agente>",
	Short: "Actualiza heartbeat y devuelve el estado operativo del agente en su proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agenteNombre := args[0]
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		jsonOut, _ := cmd.Flags().GetBool("json")
		host, _ := cmd.Flags().GetString("host")
		pidRaw, _ := cmd.Flags().GetInt64("pid")
		cuotaPct, _ := cmd.Flags().GetInt("cuota-pct")
		finalizado, _ := cmd.Flags().GetBool("finalizado")
		motivo, _ := cmd.Flags().GetString("motivo")

		if strings.TrimSpace(proyectoRef) == "" {
			return fmt.Errorf("debes indicar --proyecto")
		}

		var out agenteTickOutput
		if ok, err := apiPost("/api/agente/tick", map[string]any{
			"agente":     agenteNombre,
			"proyecto":   proyectoRef,
			"host":       host,
			"pid":        pidRaw,
			"cuota_pct":  cuotaPct,
			"finalizado": finalizado,
			"motivo":     motivo,
		}, &out); err != nil {
			return err
		} else if ok {
			return imprimirAgenteTick(out, jsonOut)
		} else if !agenteModoRecuperacionLocalExplicito() {
			return agenteErrorServerFirst()
		}
		return serverFirstCommandError("agente tick")
	},
}

var agenteOverviewCmd = &cobra.Command{
	Use:   "overview <agente>",
	Short: "Muestra el contexto operativo completo de un agente desde la app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agenteNombre := args[0]
		jsonOut, _ := cmd.Flags().GetBool("json")

		var out apiAgenteOverviewResponse
		if ok, err := apiGet(fmt.Sprintf("/api/agentes/%s/overview", url.PathEscape(agenteNombre)), &out); err != nil {
			return err
		} else if ok {
			return imprimirAgenteOverview(out, jsonOut)
		} else if !agenteModoRecuperacionLocalExplicito() {
			return agenteErrorServerFirst()
		}
		return serverFirstCommandError("agente overview")
	},
}

var agenteReanimacionesCmd = &cobra.Command{
	Use:   "reanimaciones",
	Short: "Lista reanimaciones automáticas visibles por la app",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		includeFuture, _ := cmd.Flags().GetBool("all")
		activeOnly, _ := cmd.Flags().GetBool("activos")

		var out apiAgenteReanimationsResponse
		path := "/api/agentes/reanimaciones"
		params := url.Values{}
		if includeFuture {
			params.Set("all", "true")
		}
		if activeOnly {
			params.Set("activos", "true")
		}
		if len(params) > 0 {
			path += "?" + params.Encode()
		}
		if ok, err := apiGet(path, &out); err != nil {
			return err
		} else if ok {
			return imprimirAgenteReanimaciones(out, jsonOut)
		} else if !agenteModoRecuperacionLocalExplicito() {
			return agenteErrorServerFirst()
		}
		return serverFirstCommandError("agente reanimaciones")
	},
}

func init() {
	agentePrepararCmd.Flags().String("proyecto", "", "Proyecto cuyo bundle se quiere preparar")
	agentePrepararCmd.Flags().String("conector", "", "Conector a usar; si se omite se usa el de la última sesión del proyecto")
	agentePrepararCmd.Flags().String("modelo", "", "Modelo concreto a usar para esta tarea")
	agentePrepararCmd.Flags().String("razonamiento", "", "Nivel de razonamiento o esfuerzo para esta tarea")
	agentePrepararCmd.Flags().String("perfil", "", "Perfil de tarea: orquestacion, analisis, script, implementacion, revision, etc.")
	agentePrepararCmd.Flags().String("campo", "", "Emitir solo un campo (bootstrap-prompt, continuity-prompt, command, working-dir, mode, external-session-id, branch)")
	agentePrepararCmd.Flags().Bool("json", false, "Salida JSON")

	agenteTickCmd.Flags().String("proyecto", "", "Proyecto en el que reporta el agente")
	agenteTickCmd.Flags().String("host", "", "Host del proceso o runtime actual")
	agenteTickCmd.Flags().Int64("pid", 0, "PID del proceso local si aplica")
	agenteTickCmd.Flags().Int("cuota-pct", 0, "Consumo de cuota estimado en porcentaje")
	agenteTickCmd.Flags().Bool("finalizado", false, "Marca que el turno actual del agente ha finalizado")
	agenteTickCmd.Flags().String("motivo", "", "Motivo del estado actual")
	agenteTickCmd.Flags().Bool("json", false, "Salida JSON")
	agenteOverviewCmd.Flags().Bool("json", false, "Salida JSON")
	agenteReanimacionesCmd.Flags().Bool("all", false, "Incluir reanimaciones futuras además de las vencidas")
	agenteReanimacionesCmd.Flags().Bool("activos", false, "Limitar a agentes habilitados")
	agenteReanimacionesCmd.Flags().Bool("json", false, "Salida JSON")

	agenteEjecutarCmd.Flags().StringP("proyecto", "p", "", "Proyecto en el que ejecutar")
	agenteEjecutarCmd.Flags().StringP("conector", "c", "", "Conector a usar")
	agenteEjecutarCmd.Flags().String("modelo", "", "Modelo a usar")
	agenteEjecutarCmd.Flags().Int("pausa-minutos", 60, "Minutos de espera si se detecta bloqueo")
	agenteAdoptarContextoCmd.Flags().StringP("proyecto", "p", "", "Proyecto cuyo contexto actual se entrega al orquestador")
	agenteAdoptarContextoCmd.Flags().StringP("conector", "c", "", "Conector preferido para continuar el frente")
	agenteAdoptarContextoCmd.Flags().String("cwd", "", "Directorio de trabajo actual; por defecto el cwd del proceso")
	agenteAdoptarContextoCmd.Flags().String("branch", "", "Rama actual; por defecto se intenta detectar con git")
	agenteAdoptarContextoCmd.Flags().String("herramienta", "", "Herramienta de origen de la sesión (por defecto codex-cli)")
	agenteAdoptarContextoCmd.Flags().String("external-session-id", "", "Identificador externo de la sesión si existe")
	agenteAdoptarContextoCmd.Flags().String("resume-payload", "", "Payload JSON adicional para la continuidad")
	agenteAdoptarContextoCmd.Flags().String("resumen", "", "Resumen corto del estado actual a conservar")
	agenteAdoptarContextoCmd.Flags().String("nota", "", "Nota operativa adicional para el takeover")
	agenteHandoffCmd.Flags().Int64("tarea", 0, "Tarea viva a reasignar durante el handoff")
	agenteHandoffCmd.Flags().String("motivo", "", "Motivo del handoff")
	agenteHandoffCmd.Flags().String("resumen", "", "Resumen de continuidad para el agente destino")
	agenteHandoffCmd.Flags().String("external-session-id", "", "External session id de continuidad si existe")
	agenteReasignarVivoCmd.Flags().String("motivo", "", "Motivo del handoff")
	agenteReasignarVivoCmd.Flags().String("resumen", "", "Resumen de continuidad para el agente destino")
	agenteReasignarVivoCmd.Flags().String("external-session-id", "", "External session id de continuidad si existe")
	agenteFusionarCmd.Flags().String("respaldo-destino", "", "Directorio destino del backup previo obligatorio")
	agenteFusionarCmd.Flags().String("etiqueta", "", "Etiqueta opcional para el backup previo")
	agenteFusionarCmd.Flags().Int("retener", 0, "Número máximo de backups a conservar en el destino")

	agenteCmd.AddCommand(
		agentePrepararCmd,
		agenteTickCmd,
		agenteOverviewCmd,
		agenteReanimacionesCmd,
		agenteAdoptarContextoCmd,
		agentePurgarCmd,
		agentePausarCmd,
		agenteRehabilitarCmd,
		agenteFusionarCmd,
		agenteEjecutarCmd,
		agenteHandoffCmd,
		agenteReasignarVivoCmd,
	)
}

var agenteEjecutarCmd = &cobra.Command{
	Use:   "ejecutar <agente>",
	Short: "Lanza y vigila automáticamente la ejecución de un agente (Mando y Control)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente := args[0]
		proyecto, _ := cmd.Flags().GetString("proyecto")
		conector, _ := cmd.Flags().GetString("conector")
		modelo, _ := cmd.Flags().GetString("modelo")
		pausaMin, _ := cmd.Flags().GetInt("pausa-minutos")

		if proyecto == "" {
			return fmt.Errorf("debes indicar --proyecto")
		}

		fmt.Printf("🚀 [Orquesta] Iniciando supervisión automática para: %s\n", agente)

		for {
			// 1. Preparar
			params := url.Values{"agente": {agente}, "proyecto": {proyecto}}
			if conector != "" {
				params.Set("conector", conector)
			}
			if modelo != "" {
				params.Set("modelo", modelo)
			}

			var prep agentePrepararOutput
			if ok, err := apiGetQuery("/api/agente/preparar", params, &prep); err != nil {
				return fmt.Errorf("error preparando agente: %w", err)
			} else if !ok {
				return serverFirstCommandError("agente ejecutar")
			}

			if prep.EstadoCuota == "enfriamiento" || prep.EstadoCuota == "agotado" {
				fmt.Printf("💤 [Orquesta] Agente en pausa. Reanimación estimada: %s\n", prep.ReanimarAt.Format("15:04"))
				time.Sleep(1 * time.Minute)
				continue
			}

			// 2. Ejecutar (Modo PTY con 'script' - Funcionalidad Total confirmada por USER)
			fullCmdStr := runtimeagente.RenderCommand(prep.Plan)
			fmt.Printf("🎬 [Orquesta] Ejecutando: %s\n", fullCmdStr)

			// script -q (quiet) -e (mantiene exit code) -c (comando)
			wrappedCmd := fmt.Sprintf("script -q -e -c %q /dev/null", fullCmdStr)
			proc := exec.Command("bash", "-c", wrappedCmd)

			proc.Dir = prep.Plan.WorkingDir
			proc.Env = append(os.Environ(), "ORQUESTA_AGENTE="+agente, "ORQUESTA_PROYECTO="+proyecto)
			for k, v := range prep.Plan.Env {
				proc.Env = append(proc.Env, k+"="+v)
			}

			// Tubería de espionaje para la cuota (OP-084)
			pr, pw := io.Pipe()

			// Conexión TTY Real (OP-TTY)
			proc.Stdin = os.Stdin
			// Usamos MultiWriter solo para la salida, para poder "espiar" sin romper el visual
			proc.Stdout = io.MultiWriter(os.Stdout, pw)
			proc.Stderr = io.MultiWriter(os.Stderr, pw)

			if err := proc.Start(); err != nil {
				return fmt.Errorf("error arrancando proceso: %w", err)
			}

			if err := iniciarSesionActivaAgenteEjecutor(prep, agente, proyecto, proc.Process.Pid); err != nil {
				_ = proc.Process.Kill()
				return fmt.Errorf("error registrando sesión activa: %w", err)
			}
			if err := ackBootstrapAgenteEjecutor(prep, agente, proyecto, proc.Process.Pid); err != nil {
				_ = proc.Process.Kill()
				return fmt.Errorf("error confirmando bootstrap runtime: %w", err)
			}

			stopRefresh := make(chan struct{})
			refreshDone := vigilarRefreshRuntimeMailbox(stopRefresh, agente, proyecto)

			// Goroutine Silenciosa: Vigilancia de cuota y resets
			done := make(chan bool)
			go func() {
				scanner := bufio.NewScanner(pr)
				buf := make([]byte, 64*1024)
				scanner.Buffer(buf, 10*1024*1024)
				for scanner.Scan() {
					line := scanner.Text()
					lower := strings.ToLower(line)
					if strings.Contains(lower, "hit your usage limit") ||
						strings.Contains(lower, "try again at") ||
						strings.Contains(lower, "resets") {

						fmt.Printf("\n🚨 [Orquesta] DETECTADO BLOQUEO EXTERNO: %s\n", line)

						pausaFinal := pausaMin
						re2 := regexp.MustCompile(`resets (\d+):(\d+)( on (\d+) (\w{3}))?`)
						matches := re2.FindAllStringSubmatch(line, -1)
						var selectedTarget time.Time
						now := time.Now()
						for _, m := range matches {
							h, _ := strconv.Atoi(m[1])
							mi, _ := strconv.Atoi(m[2])
							target := time.Date(now.Year(), now.Month(), now.Day(), h, mi, 0, 0, time.Local)
							if m[4] != "" {
								d, _ := strconv.Atoi(m[4])
								target = time.Date(now.Year(), now.Month(), d, h, mi, 0, 0, time.Local)
							}
							if target.Before(now) {
								target = target.AddDate(0, 0, 1)
							}
							if selectedTarget.IsZero() || target.Before(selectedTarget) {
								selectedTarget = target
							}
						}
						if !selectedTarget.IsZero() {
							pausaFinal = int(time.Until(selectedTarget).Minutes()) + 1
						}

						motivoPausa := "Detección inteligente: " + line
						detalle := fmt.Sprintf("Bloqueo detectado: %s", line)
						if ok, err := registrarPausaAgentePorAPI(
							agente,
							pausaFinal,
							motivoPausa,
							"auto_pausa",
							"sistema",
							detalle,
						); !ok {
							fmt.Fprintf(os.Stderr, "\n[Orquesta] aviso: no se pudo registrar la auto-pausa porque el servidor no esta disponible\n")
						} else if err != nil {
							fmt.Fprintf(os.Stderr, "\n[Orquesta] aviso: no se pudo registrar la auto-pausa via API (%v)\n", err)
						}
						fmt.Printf("⏸ [Orquesta] Agente pausado por %d min.\n", pausaFinal)
						_ = proc.Process.Kill()
						break
					}
				}
				if err := scanner.Err(); err != nil {
					fmt.Fprintf(os.Stderr, "\n[Orquesta] aviso: scanner de cuota degradado (%v); drenando pipe\n", err)
				}
				_, _ = io.Copy(io.Discard, pr)
				done <- true
			}()

			_ = proc.Wait()
			_ = finalizarSesionActivaAgenteEjecutor(agente)
			close(stopRefresh)
			pw.Close()
			<-done
			<-refreshDone

			fmt.Println("\n🏁 [Orquesta] Proceso finalizado. Reiniciando bucle de vigilancia (cooldown)...")
			time.Sleep(60 * time.Second)
		}
	},
}

func hostLocalAgenteEjecutor() string {
	host, err := os.Hostname()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(host)
}

func extraerAckBootstrapAgente(prep agentePrepararOutput) agenteBootstrapAck {
	var ack agenteBootstrapAck
	if prep.Bootstrap == nil {
		return ack
	}
	if prep.Bootstrap.Order != nil {
		ack.BootstrapOrderID = prep.Bootstrap.Order.ID
		var result map[string]any
		if err := json.Unmarshal([]byte(strings.TrimSpace(prep.Bootstrap.Order.ResultadoJSON)), &result); err == nil {
			if raw, ok := result["start_order_id"]; ok {
				switch v := raw.(type) {
				case float64:
					ack.StartOrderID = int64(v)
				case int64:
					ack.StartOrderID = v
				case int:
					ack.StartOrderID = int64(v)
				}
			}
		}
	}
	for _, msg := range prep.Bootstrap.Mailbox {
		if msg == nil || msg.ID <= 0 {
			continue
		}
		ack.MailboxIDs = append(ack.MailboxIDs, msg.ID)
	}
	return ack
}

func iniciarSesionActivaAgenteEjecutor(prep agentePrepararOutput, agente, proyecto string, pid int) error {
	if prep.Plan == nil {
		return nil
	}
	req := apiSesionInicioRequest{
		Agente:      strings.TrimSpace(agente),
		Conector:    strings.TrimSpace(prep.Conector.Slug),
		Proyecto:    strings.TrimSpace(proyecto),
		CWD:         strings.TrimSpace(prep.Plan.WorkingDir),
		Herramienta: strings.TrimSpace(prep.Conector.Slug),
		Branch:      strings.TrimSpace(prep.Plan.Branch),
		Host:        hostLocalAgenteEjecutor(),
		PID:         int64(pid),
	}
	var resp apiSesionInicioResponse
	ok, err := apiPost("/api/sesiones/inicio", req, &resp)
	if !ok {
		return serverFirstCommandError("agente ejecutar")
	}
	return err
}

func ackBootstrapAgenteEjecutor(prep agentePrepararOutput, agente, proyecto string, pid int) error {
	ack := extraerAckBootstrapAgente(prep)
	if ack.StartOrderID <= 0 && ack.BootstrapOrderID <= 0 && len(ack.MailboxIDs) == 0 {
		return nil
	}
	var out agenteTickOutput
	ok, err := apiPost("/api/agente/tick", map[string]any{
		"agente":                 strings.TrimSpace(agente),
		"proyecto":               strings.TrimSpace(proyecto),
		"host":                   hostLocalAgenteEjecutor(),
		"pid":                    int64(pid),
		"ack_start_order_id":     ack.StartOrderID,
		"ack_bootstrap_order_id": ack.BootstrapOrderID,
		"ack_mailbox_ids":        ack.MailboxIDs,
	}, &out)
	if !ok {
		return serverFirstCommandError("agente ejecutar")
	}
	return err
}

func finalizarSesionActivaAgenteEjecutor(agente string) error {
	ok, err := apiPost("/api/sesiones/fin", apiSesionFinRequest{Agente: strings.TrimSpace(agente)}, &map[string]any{})
	if !ok {
		return serverFirstCommandError("agente ejecutar")
	}
	return err
}

var agenteHandoffCmd = &cobra.Command{
	Use:   "handoff <agente-origen> <agente-destino>",
	Short: "Crea un handoff real y reasigna la tarea al agente destino",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tareaID, err := int64FlagOpt(cmd, "tarea")
		if err != nil {
			return err
		}
		resumen, _ := cmd.Flags().GetString("resumen")
		externalSessionID, _ := cmd.Flags().GetString("external-session-id")
		motivo, _ := cmd.Flags().GetString("motivo")

		orderID, ok, err := crearHandoffAgentePorAPI(
			strings.TrimSpace(args[0]),
			strings.TrimSpace(args[1]),
			tareaID,
			strings.TrimSpace(motivo),
			strings.TrimSpace(resumen),
			strings.TrimSpace(externalSessionID),
		)
		if !ok {
			return serverFirstCommandError("agente handoff")
		}
		if err != nil {
			return err
		}
		fmt.Printf("✓ Handoff creado %s → %s (runtime_order: %d)\n", args[0], args[1], orderID)
		if tareaID != nil {
			fmt.Printf("  Tarea #%d reasignada al agente destino\n", *tareaID)
		}
		return nil
	},
}

var agenteReasignarVivoCmd = &cobra.Command{
	Use:   "reasignar-vivo <tarea-id> <agente-origen> <agente-destino>",
	Short: "Atajo de handoff con reasignación explícita de tarea viva",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		tareaID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("tarea-id inválido")
		}
		resumen, _ := cmd.Flags().GetString("resumen")
		externalSessionID, _ := cmd.Flags().GetString("external-session-id")
		motivo, _ := cmd.Flags().GetString("motivo")

		orderID, ok, err := crearHandoffAgentePorAPI(
			strings.TrimSpace(args[1]),
			strings.TrimSpace(args[2]),
			&tareaID,
			strings.TrimSpace(motivo),
			strings.TrimSpace(resumen),
			strings.TrimSpace(externalSessionID),
		)
		if !ok {
			return serverFirstCommandError("agente reasignar-vivo")
		}
		if err != nil {
			return err
		}
		fmt.Printf("✓ Reasignación viva creada sobre tarea #%d (runtime_order: %d)\n", tareaID, orderID)
		return nil
	},
}

var agentePausarCmd = &cobra.Command{
	Use:   "pausar <nombre> <minutos> <motivo...>",
	Short: "Registra una pausa forzada (rate-limit externo) para un agente",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		nombre := args[0]
		minutos, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("duración (minutos) inválida: %s", args[1])
		}
		motivo := strings.Join(args[2:], " ")

		if ok, err := pausarAgentePorAPI(nombre, minutos, motivo); ok {
			if err != nil {
				return err
			}
		} else {
			return serverFirstCommandError("agente pausar")
		}
		fmt.Printf("✓ Agente %s pausado por %d minutos. Orquesta lo reanimará automáticamente.\n", nombre, minutos)
		return nil
	},
}

var agentePurgarCmd = &cobra.Command{
	Use:   "eliminar <nombre>",
	Short: "Elimina un agente de la base de datos (solo Alberto)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nombre := args[0]
		if ok, err := eliminarAgentePorAPI(nombre); ok {
			if err != nil {
				return err
			}
		} else {
			return serverFirstCommandError("agente eliminar")
		}
		fmt.Printf("✓ Agente %s eliminado correctamente\n", nombre)
		return nil
	},
}

var agenteRehabilitarCmd = &cobra.Command{
	Use:   "rehabilitar <agente>",
	Short: "Despierta forzosamente a un agente (limpia cuotas y pausas)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nombre := args[0]
		ok, err := resetReanimacionAgentePorAPI(nombre)
		if !ok {
			return serverFirstCommandError("agente rehabilitar")
		}
		if err != nil {
			return fmt.Errorf("error rehabilitando agente: %w", err)
		}
		fmt.Printf("✓ Agente %s rehabilitado correctamente y listo para trabajar.\n", nombre)
		return nil
	},
}

var agenteFusionarCmd = &cobra.Command{
	Use:   "fusionar <origen> <destino>",
	Short: "Fusiona un agente duplicado sobre otro con backup previo obligatorio",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		origen := strings.TrimSpace(args[0])
		destino := strings.TrimSpace(args[1])
		destinoRespaldo, _ := cmd.Flags().GetString("respaldo-destino")
		etiqueta, _ := cmd.Flags().GetString("etiqueta")
		retener, _ := cmd.Flags().GetInt("retener")
		if etiqueta == "" {
			etiqueta = fmt.Sprintf("fusion_%s_%s", strings.ToLower(origen), strings.ToLower(destino))
		}

		var (
			rutaRespaldo string
			resultado    *db.FusionAgentesResultado
		)
		if resp, ok, apiErr := fusionarAgentePorAPI(origen, destino, destinoRespaldo, etiqueta, retener); ok {
			if apiErr != nil {
				return apiErr
			}
			rutaRespaldo = resp.RutaRespaldo
			resultado = resp.Resultado
		} else {
			return serverFirstCommandError("agente fusionar")
		}

		fmt.Printf("✓ Agente %s fusionado sobre %s\n", origen, destino)
		fmt.Printf("  Respaldo: %s\n", rutaRespaldo)
		if resultado != nil {
			fmt.Printf("  Referencias actualizadas: %d\n", resultado.TotalActualizaciones())
			fmt.Printf("  Votos descartados por colisión: %d\n", resultado.VotosDescartados)
		}
		return nil
	},
}

func resolverConectorPreparacion(agente string, conectorRef string, ultima *db.Sesion) (*db.Conector, error) {
	return lanzamientoruntime.ResolverConector(agente, conectorRef, ultima)
}

func cargarPoliticaAgente() politicaAgente {
	return politicaAgente{
		ModoBucle:         configOrDefault("agent_loop_mode", "sticky"),
		IntervaloTickSeg:  configIntOrDefault("agent_tick_seconds", 30),
		ContinuarHasta:    "tarea_terminal_o_duda_real",
		ConsultarProyecto: true,
	}
}

func resolverAsignacionActiva(agente string, proyectoID int64) (bool, string, error) {
	estado := db.AsignacionActiva
	asignaciones, err := db.ListarAsignaciones(db.FiltroAsignaciones{
		Agente: &agente,
		Estado: &estado,
	})
	if err != nil {
		return false, "", err
	}
	if len(asignaciones) == 0 {
		return false, "", nil
	}
	for _, asignacion := range asignaciones {
		if asignacion.ProyectoID == proyectoID {
			return true, asignacion.ProyectoSlug, nil
		}
	}
	return false, asignaciones[0].ProyectoSlug, nil
}

func resumirSesion(s *db.Sesion) *sesionBundle {
	if s == nil {
		return nil
	}
	return &sesionBundle{
		ID:                 s.ID,
		Estado:             s.Estado,
		CWD:                s.CWD,
		Herramienta:        s.Herramienta,
		ExternalSessionID:  s.ExternalSessionID,
		ResumePayloadJSON:  s.ResumePayloadJSON,
		ResumenContinuidad: s.ResumenContinuidad,
		Branch:             s.Branch,
	}
}

func configOrDefault(clave, fallback string) string {
	v, err := controlPlaneConfigGetCached(clave)
	if err != nil || strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func configIntOrDefault(clave string, fallback int) int {
	v, err := controlPlaneConfigGetCached(clave)
	if err != nil {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func imprimirJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func valorOGuion(v string) string {
	if strings.TrimSpace(v) == "" {
		return "—"
	}
	return v
}

func valorCampoAgentePreparar(out agentePrepararOutput, campo string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(campo)) {
	case "bootstrap-prompt", "bootstrap_prompt":
		return strings.TrimSpace(out.BootstrapPrompt), nil
	case "continuity-prompt", "continuity_prompt":
		if out.Plan == nil {
			return "", nil
		}
		return strings.TrimSpace(out.Plan.ContinuityPrompt), nil
	case "command", "comando":
		if out.Plan == nil {
			return "", nil
		}
		return runtimeagente.RenderCommand(out.Plan), nil
	case "working-dir", "working_dir", "cwd":
		if out.Plan != nil && strings.TrimSpace(out.Plan.WorkingDir) != "" {
			return strings.TrimSpace(out.Plan.WorkingDir), nil
		}
		return strings.TrimSpace(out.Proyecto.RutaAbs), nil
	case "mode", "modo":
		if out.Plan == nil {
			return "", nil
		}
		return strings.TrimSpace(out.Plan.Modo), nil
	case "external-session-id", "external_session_id":
		if out.UltimaSesion == nil {
			return "", nil
		}
		return strings.TrimSpace(out.UltimaSesion.ExternalSessionID), nil
	case "branch":
		if out.UltimaSesion == nil {
			return "", nil
		}
		return strings.TrimSpace(out.UltimaSesion.Branch), nil
	default:
		return "", fmt.Errorf("campo de agente preparar no soportado: %s", campo)
	}
}

func imprimirAgentePreparar(out agentePrepararOutput, jsonOut bool) error {
	if jsonOut {
		return imprimirJSON(out)
	}
	fmt.Printf("Agente:    %s [%s]\n", out.Agente, out.Rol)
	fmt.Printf("Proyecto:  %s (%s)\n", out.Proyecto.Slug, out.Proyecto.RutaAbs)
	fmt.Printf("Conector:  %s [%s]\n", out.Conector.Slug, out.Conector.Transporte)
	fmt.Printf("Política:  modo=%s tick=%ds continuar_hasta=%s\n", out.Politica.ModoBucle, out.Politica.IntervaloTickSeg, out.Politica.ContinuarHasta)
	if out.Plan != nil && (out.Plan.Modelo != "" || out.Plan.Razonamiento != "" || out.Plan.PerfilTarea != "") {
		fmt.Printf("Ejecución: modelo=%s razonamiento=%s perfil=%s\n", valorOGuion(out.Plan.Modelo), valorOGuion(out.Plan.Razonamiento), valorOGuion(out.Plan.PerfilTarea))
	}
	if out.UltimaSesion != nil {
		fmt.Printf("Sesión previa: %d estado=%s branch=%s\n", out.UltimaSesion.ID, out.UltimaSesion.Estado, valorOGuion(out.UltimaSesion.Branch))
	}
	if out.Plan != nil {
		fmt.Printf("Plan:      %s\n", runtimeagente.RenderCommand(out.Plan))
		fmt.Printf("Modo:      %s (native_resume=%t)\n", out.Plan.Modo, out.Plan.NativeResume)
		if out.Plan.ContinuityPrompt != "" {
			fmt.Printf("Continuidad: %s\n", out.Plan.ContinuityPrompt)
		}
	}
	if strings.TrimSpace(out.BootstrapPrompt) != "" {
		fmt.Printf("Bootstrap: %s\n", truncar(out.BootstrapPrompt, 220))
	}
	if out.EstadoCuota != "activo" {
		msg := fmt.Sprintf("⚠️  AVISO: Agente en estado [%s]", out.EstadoCuota)
		if out.ReanimarAt != nil {
			msg += fmt.Sprintf(" hasta las %s", out.ReanimarAt.Format("15:04:05"))
		}
		fmt.Println(msg)
	}
	fmt.Printf("Reglas:    %d  Skills: %d  Workflows: %d  Memoria: %d\n", len(out.Reglas), len(out.Skills), len(out.Workflows), len(out.Memoria))
	if len(out.Memoria) > 0 {
		for _, entidad := range out.Memoria {
			fmt.Printf("Memoria:   %s [%s] %s\n", entidad.Nombre, entidad.Tipo, truncar(entidad.ValorJSON, 72))
		}
	}
	return nil
}

func imprimirAgenteTick(out agenteTickOutput, jsonOut bool) error {
	if jsonOut {
		return imprimirJSON(out)
	}
	fmt.Printf("Agente:    %s\n", out.Agente)
	fmt.Printf("Proyecto:  %s\n", out.Proyecto.Slug)
	fmt.Printf("Política:  modo=%s tick=%ds consultar_proyecto=%t\n", out.Politica.ModoBucle, out.Politica.IntervaloTickSeg, out.Politica.ConsultarProyecto)
	fmt.Printf("Acción:    %s\n", out.AccionRecomendada)
	if out.Motivo != "" {
		fmt.Printf("Motivo:    %s\n", out.Motivo)
	}
	fmt.Printf("Asignado:  %t", out.AsignadoAProyecto)
	if out.ProyectoAsignado != "" && !out.AsignadoAProyecto {
		fmt.Printf(" (activo en %s)", out.ProyectoAsignado)
	}
	fmt.Println()
	if out.EstadoCuota != "activo" {
		fmt.Printf("Cuota:     [%s] Consumido: %dh %dm / %dh\n",
			out.EstadoCuota, out.ConsumoDia/3600, (out.ConsumoDia%3600)/60, out.LimiteDia/3600)
		if out.ReanimarAt != nil {
			fmt.Printf("Siguiente: %s (Motivo: %s)\n", out.ReanimarAt.Format("15:04:05"), out.MotivoPausa)
		}
	}
	fmt.Printf("Tareas activas: %d  Propuestas pendientes: %d  Propuestas abiertas: %d\n", len(out.TareasActivas), len(out.PropuestasPendientes), len(out.PropuestasAbiertas))
	return nil
}

func imprimirAgenteOverview(out apiAgenteOverviewResponse, jsonOut bool) error {
	if jsonOut {
		return imprimirJSON(out)
	}
	if out.Detail == nil {
		return fmt.Errorf("overview vacio")
	}
	detail := out.Detail
	row := detail.Row
	fmt.Printf("Agente:    %s\n", strings.TrimSpace(detail.Entity.Name))
	if detail.Entity != nil && strings.TrimSpace(detail.Entity.Role) != "" {
		fmt.Printf("Rol:       %s\n", strings.TrimSpace(detail.Entity.Role))
	}
	fmt.Printf("Operativo: %s", strings.TrimSpace(row.EstadoOperativo))
	if strings.TrimSpace(row.DetalleOperativo) != "" {
		fmt.Printf(" — %s", strings.TrimSpace(row.DetalleOperativo))
	}
	fmt.Println()
	if row.Asignacion != nil {
		fmt.Printf("Asignado:  %s [%s] nota=%s\n",
			strings.TrimSpace(row.Asignacion.ProyectoSlug),
			strings.TrimSpace(string(row.Asignacion.Estado)),
			strings.TrimSpace(row.Asignacion.Nota))
	}
	if row.Sesion != nil {
		fmt.Printf("Sesión:    #%d %s %s\n", row.Sesion.ID, strings.TrimSpace(row.Sesion.Estado), strings.TrimSpace(row.Sesion.Herramienta))
	}
	if row.Runtime != nil {
		fmt.Printf("Runtime:   #%d %s\n", row.Runtime.ID, strings.TrimSpace(row.Runtime.LogicalState))
	}
	if row.Handle != nil {
		fmt.Printf("Handle:    #%d %s %s\n", row.Handle.ID, strings.TrimSpace(row.Handle.Estado), strings.TrimSpace(row.Handle.Transporte))
	}
	pendientes, cubiertas := detail.MailboxPendingVisible, detail.MailboxCoveredBootstrap
	fmt.Printf("Mailbox:   %d pendiente(s)", pendientes)
	if cubiertas > 0 {
		fmt.Printf(" · %d cubierta(s)", cubiertas)
	}
	fmt.Printf(" / %d total\n", len(detail.Mailbox))
	abiertas, bloqueadas := resumirLeasesOverview(detail.Entity.Leases)
	fmt.Printf("Tareas:    %d lease(s)", len(detail.Entity.Leases))
	if abiertas > 0 || bloqueadas > 0 {
		fmt.Printf(" · abiertas=%d bloqueadas=%d", abiertas, bloqueadas)
	}
	fmt.Println()
	for _, lease := range detail.Entity.Leases {
		fmt.Printf("  - #%d [%s] %s\n", lease.TaskID, strings.TrimSpace(string(lease.State)), strings.TrimSpace(lease.Title))
	}
	if len(detail.Asignaciones) > 0 {
		fmt.Printf("Asignaciones recientes: %d\n", len(detail.Asignaciones))
		for _, asignacion := range detail.Asignaciones {
			if asignacion == nil {
				continue
			}
			fmt.Printf("  - proyecto=%s estado=%s nota=%s\n",
				strings.TrimSpace(asignacion.ProyectoSlug),
				strings.TrimSpace(string(asignacion.Estado)),
				strings.TrimSpace(asignacion.Nota))
		}
	}
	return nil
}

func resumirLeasesOverview(leases []agentesapp.WorkLease) (abiertas, bloqueadas int) {
	for _, lease := range leases {
		switch lease.State {
		case db.EstadoBloqueada:
			bloqueadas++
		case db.EstadoAsignada, db.EstadoEnProgreso:
			abiertas++
		}
	}
	return abiertas, bloqueadas
}

func imprimirAgenteReanimaciones(out apiAgenteReanimationsResponse, jsonOut bool) error {
	if jsonOut {
		return imprimirJSON(out)
	}
	if len(out.Rows) == 0 {
		fmt.Println("Sin reanimaciones visibles.")
		return nil
	}
	fmt.Printf("Reanimaciones: %d\n", len(out.Rows))
	for _, row := range out.Rows {
		estado := "programada"
		if row.Due {
			estado = "vencida"
		}
		fmt.Printf("- %s [%s] %s", strings.TrimSpace(row.Name), estado, strings.TrimSpace(row.EstadoCuota))
		if row.ReanimarAt != nil {
			fmt.Printf(" @ %s", row.ReanimarAt.Local().Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
		if strings.TrimSpace(row.AssignmentProject) != "" {
			fmt.Printf("  proyecto=%s", strings.TrimSpace(row.AssignmentProject))
			if len(row.Leases) > 0 {
				fmt.Printf(" lease=#%d", row.Leases[0].TaskID)
			}
			fmt.Println()
		}
		if strings.TrimSpace(row.OperationalState) != "" {
			fmt.Printf("  operativo=%s", strings.TrimSpace(row.OperationalState))
			if strings.TrimSpace(row.OperationalDetail) != "" {
				fmt.Printf(" — %s", strings.TrimSpace(row.OperationalDetail))
			}
			fmt.Println()
		}
		if strings.TrimSpace(row.MotivoPausa) != "" {
			fmt.Printf("  motivo=%s\n", strings.TrimSpace(row.MotivoPausa))
		}
		fmt.Printf("  runtime=%s handle=%s worker=%s mailbox=%d abiertas=%d bloqueadas=%d\n",
			strings.TrimSpace(row.RuntimeState),
			strings.TrimSpace(row.HandleState),
			strings.TrimSpace(row.WorkerState),
			row.MailboxPending,
			row.OpenTasks,
			row.BlockedTasks)
	}
	return nil
}
