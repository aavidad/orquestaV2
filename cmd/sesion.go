/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var sesionCmd = &cobra.Command{
	Use:   "sesion",
	Short: "Gestión de sesiones de agentes",
}

// sesion inicio
var sesionInicioCmd = &cobra.Command{
	Use:   "inicio [agente]",
	Short: "Inicia sesión de un agente y muestra sus reglas, skills y workflows",
	Long: `Inicia la sesión de un agente registrado.
Si se usa --nuevo-codex, registra automáticamente el siguiente codexN disponible.

Ejemplos:
  orquesta sesion inicio claude
  orquesta sesion inicio --nuevo-codex`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return ejecutarInicioSesion(cmd, args, false)
	},
}

func ejecutarInicioSesion(cmd *cobra.Command, args []string, forzarNuevoCodex bool) error {
	nuevoCodex, _ := cmd.Flags().GetBool("nuevo-codex")
	if forzarNuevoCodex {
		nuevoCodex = true
	}

	var agente string
	if !nuevoCodex {
		if len(args) == 0 {
			return fmt.Errorf("indica el nombre del agente o usa --nuevo-codex")
		}
		agente = args[0]
	}

	proyectoRef, _ := cmd.Flags().GetString("proyecto")
	conectorRef, _ := cmd.Flags().GetString("conector")
	cwd, _ := cmd.Flags().GetString("cwd")
	herramienta, _ := cmd.Flags().GetString("herramienta")
	branch, _ := cmd.Flags().GetString("branch")
	externalSessionID, _ := cmd.Flags().GetString("external-session-id")
	resumePayload, _ := cmd.Flags().GetString("resume-payload")
	resumen, _ := cmd.Flags().GetString("resumen")
	host, _ := cmd.Flags().GetString("host")
	pidRaw, _ := cmd.Flags().GetInt64("pid")

	req := apiSesionInicioRequest{
		Agente:            agente,
		NuevoCodex:        nuevoCodex,
		Conector:          conectorRef,
		Proyecto:          proyectoRef,
		CWD:               cwd,
		Herramienta:       herramienta,
		Branch:            branch,
		ExternalSessionID: externalSessionID,
		ResumePayload:     resumePayload,
		Resumen:           resumen,
		Host:              host,
		PID:               pidRaw,
	}

	var resp apiSesionInicioResponse
	if ok, err := apiPost("/api/sesiones/inicio", req, &resp); err != nil {
		return err
	} else if !ok {
		if err := ensureLocalDB(); err != nil {
			return err
		}
		localResp, err := construirSesionInicioResponse(req)
		if err != nil {
			return fmt.Errorf("iniciando sesión: %w", err)
		}
		resp = *localResp
	}

	imprimirInicioSesion(&resp, nuevoCodex)
	return nil
}

func imprimirInicioSesion(resp *apiSesionInicioResponse, nuevoCodex bool) {
	if resp == nil || resp.Sesion == nil {
		return
	}
	sesion := resp.Sesion
	agente := sesion.Agente
	if nuevoCodex {
		fmt.Printf("✓ Nuevo agente Codex registrado como: %s\n\n", agente)
	}

	fmt.Printf("═══════════════════════════════════════════════════════════\n")
	fmt.Printf("  SESIÓN INICIADA — agente: %s  (sesion_id: %d)\n", agente, sesion.ID)
	fmt.Printf("═══════════════════════════════════════════════════════════\n\n")
	if sesion.ProyectoID != nil {
		fmt.Printf("📁 Proyecto: %s (%s)\n", sesion.ProyectoSlug, sesion.ProyectoNombre)
	}
	if strings.TrimSpace(sesion.CWD) != "" {
		fmt.Printf("📂 CWD: %s\n", sesion.CWD)
	}
	if strings.TrimSpace(sesion.Herramienta) != "" {
		fmt.Printf("🛠  Herramienta: %s\n", sesion.Herramienta)
	}
	if sesion.ConectorSlug != "" {
		fmt.Printf("🔌 Conector: %s\n", sesion.ConectorSlug)
	}
	if strings.TrimSpace(sesion.Branch) != "" {
		fmt.Printf("🌿 Branch: %s\n", sesion.Branch)
	}
	if previo := resp.SesionPrevia; previo != nil && previo.ID > 0 && (strings.TrimSpace(previo.ResumenContinuidad) != "" || strings.TrimSpace(previo.ExternalSessionID) != "" || strings.TrimSpace(previo.ResumePayloadJSON) != "") {
		fmt.Printf("\n♻️  CONTEXTO REANUDABLE DETECTADO:\n")
		fmt.Printf("   • sesión previa: %d\n", previo.ID)
		if previo.ProyectoSlug != "" {
			fmt.Printf("   • proyecto: %s\n", previo.ProyectoSlug)
		}
		if previo.ExternalSessionID != "" {
			fmt.Printf("   • external_session_id: %s\n", previo.ExternalSessionID)
		}
		if previo.Branch != "" {
			fmt.Printf("   • branch previa: %s\n", previo.Branch)
		}
		if previo.ResumenContinuidad != "" {
			fmt.Printf("   • resumen: %s\n", previo.ResumenContinuidad)
		}
		fmt.Println()
	}

	rol := resp.Rol
	if rol == "" || rol == "admin" {
		fmt.Printf("Sesión iniciada. Rol: %s\n", rol)
		return
	}

	if len(resp.PropuestasPendientes) > 0 {
		fmt.Printf("⚠️  PROPUESTAS PENDIENTES DE TU VOTO (%d):\n", len(resp.PropuestasPendientes))
		for _, p := range resp.PropuestasPendientes {
			fmt.Printf("   • %s — %s\n", p.Codigo, p.Titulo)
		}
		fmt.Println()
	}

	if len(resp.Reglas) > 0 {
		fmt.Printf("📋 REGLAS ACTIVAS (%s):\n", strings.ToUpper(rol))
		catActual := ""
		for _, r := range resp.Reglas {
			if r.Categoria != catActual {
				catActual = r.Categoria
				fmt.Printf("\n  [%s]\n", strings.ToUpper(catActual))
			}
			fmt.Printf("  • %s: %s\n", r.Titulo, r.Descripcion)
		}
		fmt.Println()
	}

	if len(resp.Skills) > 0 {
		fmt.Printf("🛠  SKILLS DISPONIBLES:\n")
		for _, s := range resp.Skills {
			fmt.Printf("  • %-30s — %s\n", s.Nombre, s.CuandoUsar)
		}
		fmt.Println()
	}

	if wf := resp.Workflow; wf != nil {
		var pasos []string
		if err := json.Unmarshal([]byte(wf.Pasos), &pasos); err == nil {
			fmt.Printf("📌 WORKFLOW — %s:\n", strings.ToUpper(wf.Nombre))
			for _, paso := range pasos {
				fmt.Printf("  %s\n", paso)
			}
			fmt.Println()
		}
	}

	fmt.Printf("─── Listo. Usa 'orquesta status' para ver el estado del proyecto. ───\n")
}

var sesionGuardarCmd = &cobra.Command{
	Use:   "guardar <agente>",
	Short: "Guarda contexto de la sesión activa para reanudarla después",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente := args[0]
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		cwd, _ := cmd.Flags().GetString("cwd")
		herramienta, _ := cmd.Flags().GetString("herramienta")
		branch, _ := cmd.Flags().GetString("branch")
		externalSessionID, _ := cmd.Flags().GetString("external-session-id")
		resumePayload, _ := cmd.Flags().GetString("resume-payload")
		resumen, _ := cmd.Flags().GetString("resumen")
		host, _ := cmd.Flags().GetString("host")
		estado, _ := cmd.Flags().GetString("estado")
		pidRaw, _ := cmd.Flags().GetInt64("pid")

		if ok, err := apiPost("/api/sesiones/guardar", apiSesionGuardarRequest{
			Agente:            agente,
			Proyecto:          proyectoRef,
			CWD:               cwd,
			Herramienta:       herramienta,
			Branch:            branch,
			ExternalSessionID: externalSessionID,
			ResumePayload:     resumePayload,
			Resumen:           resumen,
			Host:              host,
			Estado:            estado,
			PID:               pidRaw,
		}, &apiSesionResponse{}); err != nil {
			return err
		} else if !ok {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			var proyectoID *int64
			if strings.TrimSpace(proyectoRef) != "" {
				p, err := db.GetProyecto(proyectoRef)
				if err != nil {
					return err
				}
				proyectoID = &p.ID
			}
			upd := db.SesionUpdate{Heartbeat: true}
			if cwd != "" {
				upd.CWD = &cwd
			}
			if herramienta != "" {
				upd.Herramienta = &herramienta
			}
			if branch != "" {
				upd.Branch = &branch
			}
			if externalSessionID != "" {
				upd.ExternalSessionID = &externalSessionID
			}
			if resumePayload != "" {
				upd.ResumePayloadJSON = &resumePayload
			}
			if resumen != "" {
				upd.ResumenContinuidad = &resumen
			}
			if host != "" {
				upd.Host = &host
			}
			if estado != "" {
				upd.Estado = &estado
			}
			if pidRaw > 0 {
				upd.PID = &pidRaw
			}
			if err := db.GuardarSesionActiva(agente, proyectoID, upd); err != nil {
				return err
			}
		}
		fmt.Printf("✓ Contexto de sesión guardado para %s\n", agente)
		return nil
	},
}

var sesionContinuarCmd = &cobra.Command{
	Use:   "continuar <agente>",
	Short: "Muestra la última sesión reanudable de un agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente := args[0]
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		cwd, _ := cmd.Flags().GetString("cwd")
		jsonOut, _ := cmd.Flags().GetBool("json")
		campo, _ := cmd.Flags().GetString("campo")
		var s *db.Sesion
		params := mapToValues(map[string]string{
			"agente":   agente,
			"proyecto": proyectoRef,
			"cwd":      cwd,
		})
		var resp apiSesionResponse
		if ok, err := apiGetQuery("/api/sesiones/continuar", params, &resp); err != nil {
			return err
		} else if ok {
			s = resp.Sesion
		} else {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			var proyectoID *int64
			if strings.TrimSpace(proyectoRef) != "" {
				p, err := db.GetProyecto(proyectoRef)
				if err != nil {
					return err
				}
				proyectoID = &p.ID
			}
			var err error
			s, err = db.ObtenerUltimaSesionConFiltro(agente, proyectoID, cwd)
			if err != nil {
				return err
			}
		}
		if jsonOut {
			return imprimirJSON(s)
		}
		if strings.TrimSpace(campo) != "" {
			v, err := valorCampoSesion(s, campo)
			if err != nil {
				return err
			}
			fmt.Println(v)
			return nil
		}
		fmt.Printf("Sesión %d\n", s.ID)
		fmt.Printf("  Agente:      %s\n", s.Agente)
		if s.ProyectoSlug != "" {
			fmt.Printf("  Proyecto:    %s\n", s.ProyectoSlug)
		}
		if s.CWD != "" {
			fmt.Printf("  CWD:         %s\n", s.CWD)
		}
		if s.Herramienta != "" {
			fmt.Printf("  Herramienta: %s\n", s.Herramienta)
		}
		if s.ExternalSessionID != "" {
			fmt.Printf("  External ID: %s\n", s.ExternalSessionID)
		}
		if s.Branch != "" {
			fmt.Printf("  Branch:      %s\n", s.Branch)
		}
		if s.ResumenContinuidad != "" {
			fmt.Printf("  Resumen:     %s\n", s.ResumenContinuidad)
		}
		if s.ResumePayloadJSON != "" {
			fmt.Printf("  Resume JSON: %s\n", s.ResumePayloadJSON)
		}
		return nil
	},
}

func valorCampoSesion(s *db.Sesion, campo string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("sesión nil")
	}
	switch strings.TrimSpace(strings.ToLower(campo)) {
	case "id":
		return strconv.FormatInt(s.ID, 10), nil
	case "agente":
		return s.Agente, nil
	case "proyecto":
		return s.ProyectoSlug, nil
	case "cwd":
		return s.CWD, nil
	case "herramienta":
		return s.Herramienta, nil
	case "external-session-id", "external_session_id":
		return s.ExternalSessionID, nil
	case "branch":
		return s.Branch, nil
	case "resumen", "resumen_continuidad":
		return s.ResumenContinuidad, nil
	case "resume-payload", "resume_payload_json":
		return s.ResumePayloadJSON, nil
	case "estado":
		return s.Estado, nil
	case "host":
		return s.Host, nil
	case "pid":
		if s.PID == nil {
			return "", nil
		}
		return strconv.FormatInt(*s.PID, 10), nil
	default:
		return "", fmt.Errorf("campo de sesión no soportado: %s", campo)
	}
}

// sesion fin
var sesionFinCmd = &cobra.Command{
	Use:   "fin [agente]",
	Short: "Cierra la sesión activa de un agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente := args[0]

		if ok, err := apiPost("/api/sesiones/fin", apiSesionFinRequest{Agente: agente}, &map[string]any{}); err != nil {
			return err
		} else if !ok {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			agentes, _ := db.ListarAgentes()
			var rol string
			for _, a := range agentes {
				if a.Nombre == agente {
					rol = a.Rol
					break
				}
			}
			if rol != "" && rol != "admin" {
				wf, err := db.GetWorkflow(rol, "fin-sesion")
				if err == nil {
					var pasos []string
					if err2 := json.Unmarshal([]byte(wf.Pasos), &pasos); err2 == nil {
						fmt.Printf("📌 CHECKLIST DE CIERRE:\n")
						for _, paso := range pasos {
							fmt.Printf("  %s\n", paso)
						}
						fmt.Println()
					}
				}
			}
			if err := db.FinSesion(agente); err != nil {
				return err
			}
		}
		fmt.Printf("✓ Sesión cerrada — agente: %s\n", agente)
		return nil
	},
}

// sesion listar
var sesionListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista todos los agentes y su estado de sesión",
	RunE: func(cmd *cobra.Command, args []string) error {
		var agentes []*db.Agente
		var resp apiAgentesResponse
		if ok, err := apiGet("/api/agentes", &resp); err != nil {
			return err
		} else if ok {
			agentes = resp.Agentes
		} else {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			var err error
			agentes, err = db.ListarAgentes()
			if err != nil {
				return err
			}
		}
		fmt.Printf("%-15s %-15s %-8s %s\n", "AGENTE", "ROL", "ACTIVO", "ÚLTIMA SESIÓN")
		fmt.Printf("%-15s %-15s %-8s %s\n",
			"───────────────", "───────────────", "────────", "────────────────────")
		for _, a := range agentes {
			activo := "no"
			if a.Activo {
				activo = "SÍ"
			}
			ultima := "—"
			if a.UltimaSesion != nil {
				ultima = a.UltimaSesion.Format("2006-01-02 15:04")
			}
			fmt.Printf("%-15s %-15s %-8s %s\n", a.Nombre, a.Rol, activo, ultima)
		}
		return nil
	},
}

// sesion nuevo-codex (atajo directo)
var sesionNuevoCodexCmd = &cobra.Command{
	Use:   "nuevo-codex",
	Short: "Registra e inicia sesión para un nuevo agente Codex (auto-nombrado codex1, codex2…)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return ejecutarInicioSesion(cmd, nil, true)
	},
}

func init() {
	sesionInicioCmd.Flags().Bool("nuevo-codex", false, "Registra un nuevo agente Codex con nombre automático")
	sesionInicioCmd.Flags().String("conector", "", "Conector/model runtime de la sesión")
	sesionInicioCmd.Flags().String("proyecto", "", "Proyecto asignado a la sesión")
	sesionInicioCmd.Flags().String("cwd", "", "Directorio de trabajo de la sesión")
	sesionInicioCmd.Flags().String("herramienta", "", "Herramienta/runtine del agente (codex, claude, etc.)")
	sesionInicioCmd.Flags().String("branch", "", "Branch asociada a la sesión")
	sesionInicioCmd.Flags().String("external-session-id", "", "ID externo de sesión reanudable")
	sesionInicioCmd.Flags().String("resume-payload", "", "Payload JSON para reanudar contexto")
	sesionInicioCmd.Flags().String("resumen", "", "Resumen corto de continuidad")
	sesionInicioCmd.Flags().String("host", "", "Host donde corre el agente")
	sesionInicioCmd.Flags().Int64("pid", 0, "PID del proceso del agente")

	sesionGuardarCmd.Flags().String("proyecto", "", "Proyecto de la sesión activa")
	sesionGuardarCmd.Flags().String("cwd", "", "Directorio de trabajo actual")
	sesionGuardarCmd.Flags().String("herramienta", "", "Herramienta/runtine del agente")
	sesionGuardarCmd.Flags().String("branch", "", "Branch actual")
	sesionGuardarCmd.Flags().String("external-session-id", "", "ID externo de sesión reanudable")
	sesionGuardarCmd.Flags().String("resume-payload", "", "Payload JSON para reanudar contexto")
	sesionGuardarCmd.Flags().String("resumen", "", "Resumen corto de continuidad")
	sesionGuardarCmd.Flags().String("host", "", "Host donde corre el agente")
	sesionGuardarCmd.Flags().String("estado", "pausada", "Estado lógico de la sesión al guardarla")
	sesionGuardarCmd.Flags().Int64("pid", 0, "PID del proceso del agente")

	sesionContinuarCmd.Flags().String("proyecto", "", "Proyecto concreto cuya sesión quieres reanudar")
	sesionContinuarCmd.Flags().String("cwd", "", "Directorio de trabajo concreto cuya sesión quieres reanudar")
	sesionContinuarCmd.Flags().Bool("json", false, "Salida JSON")
	sesionContinuarCmd.Flags().String("campo", "", "Emitir solo un campo (id, agente, proyecto, cwd, herramienta, external-session-id, branch, resumen, resume-payload, estado, host, pid)")

	sesionCmd.AddCommand(sesionInicioCmd, sesionFinCmd, sesionListarCmd, sesionNuevoCodexCmd, sesionGuardarCmd, sesionContinuarCmd)
}
