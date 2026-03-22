/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/agentruntime"
	"orquesta/db"
)

var agenteCmd = &cobra.Command{
	Use:   "agente",
	Short: "Operaciones de orquestación para agentes",
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
	Agente       string                   `json:"agente"`
	Rol          string                   `json:"rol"`
	Proyecto     proyectoBundle           `json:"proyecto"`
	Conector     conectorBundle           `json:"conector"`
	Politica     politicaAgente           `json:"politica"`
	UltimaSesion *sesionBundle            `json:"ultima_sesion,omitempty"`
	Plan         *agentruntime.LaunchPlan `json:"plan"`
	Reglas       []*db.Regla              `json:"reglas"`
	Skills       []*db.Skill              `json:"skills"`
	Workflows    []*db.Workflow           `json:"workflows"`
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
			return imprimirAgentePreparar(out, jsonOut)
		}

		if err := ensureLocalDB(); err != nil {
			return err
		}

		agente, err := db.GetAgente(agenteNombre)
		if err != nil {
			return err
		}
		proyecto, err := db.GetProyecto(proyectoRef)
		if err != nil {
			return err
		}

		var ultima *db.Sesion
		ultima, err = db.ObtenerUltimaSesion(agenteNombre, &proyecto.ID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		conector, err := resolverConectorPreparacion(conectorRef, ultima)
		if err != nil {
			return err
		}

		reglas, err := db.GetReglasAgente(agente.Rol)
		if err != nil {
			return err
		}
		skills, err := db.GetSkillsAgente(agente.Rol)
		if err != nil {
			return err
		}
		workflows, err := db.GetWorkflowsAgente(agente.Rol)
		if err != nil {
			return err
		}

		req := agentruntime.LaunchRequest{
			Agente:       agente.Nombre,
			Rol:          agente.Rol,
			ProyectoSlug: proyecto.Slug,
			ProyectoRuta: proyecto.RutaAbs,
			Modelo:       strings.TrimSpace(modelo),
			Razonamiento: strings.TrimSpace(razonamiento),
			PerfilTarea:  strings.TrimSpace(perfilTarea),
			Conector: agentruntime.ConnectorConfig{
				Slug:         conector.Slug,
				Nombre:       conector.Nombre,
				Transporte:   conector.Transporte,
				Comando:      conector.Comando,
				ArgsJSON:     conector.ArgsJSON,
				EnvJSON:      conector.EnvJSON,
				MetadataJSON: conector.MetadataJSON,
				Activo:       conector.Activo,
			},
		}
		if ultima != nil {
			req.Resume = agentruntime.ResumeContext{
				ExternalSessionID:  ultima.ExternalSessionID,
				ResumePayloadJSON:  ultima.ResumePayloadJSON,
				ResumenContinuidad: ultima.ResumenContinuidad,
				Branch:             ultima.Branch,
				CWD:                ultima.CWD,
			}
		}
		plan, err := agentruntime.DefaultRegistry().Prepare(req)
		if err != nil {
			return err
		}

		out = agentePrepararOutput{
			Agente: agente.Nombre,
			Rol:    agente.Rol,
			Proyecto: proyectoBundle{
				ID:      proyecto.ID,
				Slug:    proyecto.Slug,
				Nombre:  proyecto.Nombre,
				RutaAbs: proyecto.RutaAbs,
			},
			Conector: conectorBundle{
				ID:         conector.ID,
				Slug:       conector.Slug,
				Nombre:     conector.Nombre,
				Transporte: conector.Transporte,
				Comando:    conector.Comando,
			},
			Politica:  cargarPoliticaAgente(),
			Plan:      plan,
			Reglas:    reglas,
			Skills:    skills,
			Workflows: workflows,
		}
		out.Politica.Modelo = plan.Modelo
		out.Politica.Razonamiento = plan.Razonamiento
		out.Politica.PerfilTarea = plan.PerfilTarea
		if ultima != nil {
			out.UltimaSesion = resumirSesion(ultima)
		}

		return imprimirAgentePreparar(out, jsonOut)
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

		if strings.TrimSpace(proyectoRef) == "" {
			return fmt.Errorf("debes indicar --proyecto")
		}

		var out agenteTickOutput
		if ok, err := apiPost("/api/agente/tick", map[string]any{
			"agente":   agenteNombre,
			"proyecto": proyectoRef,
			"host":     host,
			"pid":      pidRaw,
		}, &out); err != nil {
			return err
		} else if ok {
			return imprimirAgenteTick(out, jsonOut)
		}

		if err := ensureLocalDB(); err != nil {
			return err
		}

		proyecto, err := db.GetProyecto(proyectoRef)
		if err != nil {
			return err
		}

		sesionActiva, err := db.GetSesionActiva(agenteNombre, &proyecto.ID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if sesionActiva != nil {
			upd := db.SesionUpdate{Heartbeat: true}
			if host != "" {
				upd.Host = &host
			}
			if pidRaw > 0 {
				upd.PID = &pidRaw
			}
			if err := db.GuardarSesionActiva(agenteNombre, &proyecto.ID, upd); err != nil {
				return err
			}
			sesionActiva, err = db.GetSesionActiva(agenteNombre, &proyecto.ID)
			if err != nil {
				return err
			}
		}

		politica := cargarPoliticaAgente()
		asignadoAProyecto, proyectoAsignado, err := resolverAsignacionActiva(agenteNombre, proyecto.ID)
		if err != nil {
			return err
		}

		tareas, err := db.ListarTareas(db.FiltroTareas{Agente: &agenteNombre, ProyectoID: &proyecto.ID})
		if err != nil {
			return err
		}
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
				tieneBloqueos = true
			}
			if tarea.Estado == db.TareaAsignada || tarea.Estado == db.TareaEnProgreso {
				tieneTrabajo = true
			}
		}

		pendientes, err := db.PropuestasPendientesVotoProyecto(agenteNombre, &proyecto.ID)
		if err != nil {
			return err
		}
		var propuestasPendientes []itemLigero
		for _, propuesta := range pendientes {
			propuestasPendientes = append(propuestasPendientes, itemLigero{
				ID:     propuesta.ID,
				Codigo: propuesta.Codigo,
				Titulo: propuesta.Titulo,
				Estado: string(propuesta.Estado),
			})
		}

		estadoAbierta := db.PropuestaAbierta
		abiertas, err := db.ListarPropuestas(&estadoAbierta, &proyecto.ID)
		if err != nil {
			return err
		}
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
		}
		if sesionActiva != nil {
			out.SesionActiva = resumirSesion(sesionActiva)
		}

		switch {
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

		return imprimirAgenteTick(out, jsonOut)
	},
}

func init() {
	agentePrepararCmd.Flags().String("proyecto", "", "Proyecto cuyo bundle se quiere preparar")
	agentePrepararCmd.Flags().String("conector", "", "Conector a usar; si se omite se usa el de la última sesión del proyecto")
	agentePrepararCmd.Flags().String("modelo", "", "Modelo concreto a usar para esta tarea")
	agentePrepararCmd.Flags().String("razonamiento", "", "Nivel de razonamiento o esfuerzo para esta tarea")
	agentePrepararCmd.Flags().String("perfil", "", "Perfil de tarea: orquestacion, analisis, script, implementacion, revision, etc.")
	agentePrepararCmd.Flags().Bool("json", false, "Salida JSON")

	agenteTickCmd.Flags().String("proyecto", "", "Proyecto a vigilar")
	agenteTickCmd.Flags().String("host", "", "Host del proceso del agente")
	agenteTickCmd.Flags().Int64("pid", 0, "PID del proceso del agente")
	agenteTickCmd.Flags().Bool("json", false, "Salida JSON")

	agenteCmd.AddCommand(agentePrepararCmd, agenteTickCmd)
}

func resolverConectorPreparacion(conectorRef string, ultima *db.Sesion) (*db.Conector, error) {
	if strings.TrimSpace(conectorRef) != "" {
		return db.GetConector(conectorRef)
	}
	if ultima != nil && ultima.ConectorID != nil {
		return db.GetConector(strconv.FormatInt(*ultima.ConectorID, 10))
	}
	return nil, fmt.Errorf("debes indicar --conector o disponer de una sesión previa con conector asociado")
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
	v, err := db.ConfigGet(clave)
	if err != nil || strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func configIntOrDefault(clave string, fallback int) int {
	v, err := db.ConfigGet(clave)
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
		fmt.Printf("Plan:      %s\n", agentruntime.RenderCommand(out.Plan))
		fmt.Printf("Modo:      %s (native_resume=%t)\n", out.Plan.Modo, out.Plan.NativeResume)
		if out.Plan.ContinuityPrompt != "" {
			fmt.Printf("Continuidad: %s\n", out.Plan.ContinuityPrompt)
		}
	}
	fmt.Printf("Reglas:    %d  Skills: %d  Workflows: %d\n", len(out.Reglas), len(out.Skills), len(out.Workflows))
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
	fmt.Printf("Tareas activas: %d  Propuestas pendientes: %d  Propuestas abiertas: %d\n", len(out.TareasActivas), len(out.PropuestasPendientes), len(out.PropuestasAbiertas))
	return nil
}
