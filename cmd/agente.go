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
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
	"orquesta/internal/lanzamientoruntime"
	"orquesta/runtimeagente"
)

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

func construirAgenteTickOutput(agenteNombre string, proyecto *db.Proyecto, sesionActiva *db.Sesion, cuotaPct int) (agenteTickOutput, error) {
	var out agenteTickOutput
	if proyecto == nil {
		return out, fmt.Errorf("proyecto obligatorio")
	}

	politica := cargarPoliticaAgente()
	asignadoAProyecto, proyectoAsignado, err := resolverAsignacionActiva(agenteNombre, proyecto.ID)
	if err != nil {
		return out, err
	}

	tareas, err := tareasService.List(db.FiltroTareas{Agente: &agenteNombre, ProyectoID: &proyecto.ID})
	if err != nil {
		return out, err
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

	pendientes, err := propuestasService.ListPendingProjectVotes(agenteNombre, proyecto.Slug)
	if err != nil {
		return out, err
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
	abiertas, err := propuestasService.ListByProject(&estadoAbierta, proyecto.Slug)
	if err != nil {
		return out, err
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

	agente, err := runtimesService.GetAgent(agenteNombre)
	if err != nil {
		return out, err
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
	esSupervisorOperativo, supervisorOperativo, err := db.EsSupervisorAutonomiaOperativo(proyecto.ID, agenteNombre)
	if err != nil {
		return out, err
	}
	pausarPorPresupuesto, motivoPresupuesto, err := agenteDebePausarPorPresupuesto(agenteNombre)
	if err != nil {
		return out, err
	}

	switch {
	case pausarPorPresupuesto:
		out.AccionRecomendada = "pausar_por_cuota"
		out.DebePausar = true
		out.Motivo = motivoPresupuesto
	case agente.EstadoCuota != "activo":
		out.AccionRecomendada = "pausar_por_cuota"
		out.DebePausar = true
		if agente.ReanimarAt != nil {
			out.Motivo = fmt.Sprintf("Cuota agotada (%s). Reanimación programada para: %s. Motivo: %s",
				agente.EstadoCuota, agente.ReanimarAt.Format("15:04:05"), agente.MotivoPausa)
		} else {
			out.Motivo = "Cuota agotada o modo enfriamiento activo."
		}
	case !asignadoAProyecto && proyectoAsignado != "":
		out.AccionRecomendada = "pausar_y_reasignar"
		out.DebePausar = true
		out.Motivo = "La asignación activa del agente ha cambiado al proyecto " + proyectoAsignado
	case esSupervisorOperativo:
		out.AccionRecomendada = "supervisar_proyecto"
		if supervisorOperativo != nil && !strings.EqualFold(strings.TrimSpace(supervisorOperativo.Nombre), strings.TrimSpace(agenteNombre)) {
			out.Motivo = "Debes asumir el relevo temporal de la orquestación del proyecto."
		} else {
			out.Motivo = "Eres el supervisor operativo del proyecto y debes coordinar el siguiente frente útil."
		}
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

			stopRefresh := make(chan struct{})
			refreshDone := vigilarRefreshRuntimeMailbox(stopRefresh, agente, proyecto)

			// Goroutine Silenciosa: Vigilancia de cuota y resets
			done := make(chan bool)
			go func() {
				scanner := bufio.NewScanner(pr)
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
				done <- true
			}()

			_ = proc.Wait()
			close(stopRefresh)
			pw.Close()
			<-done
			<-refreshDone

			fmt.Println("\n🏁 [Orquesta] Proceso finalizado. Reiniciando bucle de vigilancia...")
			time.Sleep(5 * time.Second)
		}
	},
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
