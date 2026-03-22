/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
	"orquesta/sessionapp"
)

var sesionCmd = &cobra.Command{
	Use:   "sesion",
	Short: "Gestión de sesiones de agentes",
}

var sessionService = sessionapp.NewService(sessionapp.Repository{})

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
		nuevoCodex, _ := cmd.Flags().GetBool("nuevo-codex")
		if serverURL := activeServerURL(); serverURL != "" {
			agente := ""
			if !nuevoCodex {
				if len(args) == 0 {
					return fmt.Errorf("indica el nombre del agente o usa --nuevo-codex")
				}
				agente = args[0]
			}
			res, err := submitServerSessionStart(serverURL, agente, nuevoCodex)
			if err != nil {
				return fmt.Errorf("iniciando sesión: %w", err)
			}
			renderSessionStartResult(res, nuevoCodex)
			return nil
		}
		agente := ""
		if !nuevoCodex {
			if len(args) == 0 {
				return fmt.Errorf("indica el nombre del agente o usa --nuevo-codex")
			}
			agente = args[0]
		}
		result, err := sessionService.Start(agente, nuevoCodex)
		if err != nil {
			return fmt.Errorf("iniciando sesión: %w", err)
		}
		renderSessionStartResult(&serverSessionStartResult{
			Agente:               result.Agente,
			SesionID:             result.SesionID,
			Rol:                  result.Rol,
			PropuestasPendientes: result.PropuestasPendientes,
			Reglas:               result.Reglas,
			Skills:               result.Skills,
			WorkflowPasos:        result.WorkflowPasos,
		}, nuevoCodex)
		return nil
	},
}

// sesion fin
var sesionFinCmd = &cobra.Command{
	Use:   "fin [agente]",
	Short: "Cierra la sesión activa de un agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente := args[0]
		if serverURL := activeServerURL(); serverURL != "" {
			if err := submitServerSessionFinish(serverURL, agente); err != nil {
				return err
			}
			fmt.Printf("✓ Sesión cerrada — agente: %s\n", agente)
			return nil
		}
		result, err := sessionService.Finish(agente)
		if err != nil {
			return err
		}
		if result.Rol != "" && result.Rol != "admin" && len(result.WorkflowPasos) > 0 {
			fmt.Printf("📌 CHECKLIST DE CIERRE:\n")
			for _, paso := range result.WorkflowPasos {
				fmt.Printf("  %s\n", paso)
			}
			fmt.Println()
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
		var (
			agentes []*db.Agente
			err     error
		)
		if serverURL := activeServerURL(); serverURL != "" {
			agentes, err = fetchServerAgents(serverURL)
		} else {
			agentes, err = sessionService.ListAgents()
		}
		if err != nil {
			return err
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
		if serverURL := activeServerURL(); serverURL != "" {
			res, err := submitServerSessionStart(serverURL, "", true)
			if err != nil {
				return fmt.Errorf("iniciando sesión: %w", err)
			}
			renderSessionStartResult(res, true)
			return nil
		}
		result, err := sessionService.Start("", true)
		if err != nil {
			return fmt.Errorf("iniciando sesión: %w", err)
		}
		fmt.Printf("✓ Agente registrado como: %s\n", result.Agente)
		fmt.Printf("✓ Sesión iniciada (id: %d)\n", result.SesionID)
		return nil
	},
}

func init() {
	sesionInicioCmd.Flags().Bool("nuevo-codex", false, "Registra un nuevo agente Codex con nombre automático")
	sesionCmd.AddCommand(sesionInicioCmd, sesionFinCmd, sesionListarCmd, sesionNuevoCodexCmd)
}

func renderSessionStartResult(res *serverSessionStartResult, nuevoCodex bool) {
	if nuevoCodex {
		fmt.Printf("✓ Nuevo agente Codex registrado como: %s\n\n", res.Agente)
	}
	fmt.Printf("═══════════════════════════════════════════════════════════\n")
	fmt.Printf("  SESIÓN INICIADA — agente: %s  (sesion_id: %d)\n", res.Agente, res.SesionID)
	fmt.Printf("═══════════════════════════════════════════════════════════\n\n")

	if res.Rol == "" || res.Rol == "admin" {
		fmt.Printf("Sesión iniciada. Rol: %s\n", res.Rol)
		return
	}
	if len(res.PropuestasPendientes) > 0 {
		fmt.Printf("⚠️  PROPUESTAS PENDIENTES DE TU VOTO (%d):\n", len(res.PropuestasPendientes))
		for _, p := range res.PropuestasPendientes {
			fmt.Printf("   • %s — %s\n", p.Codigo, p.Titulo)
		}
		fmt.Println()
	}
	if len(res.Reglas) > 0 {
		fmt.Printf("📋 REGLAS ACTIVAS (%s):\n", strings.ToUpper(res.Rol))
		catActual := ""
		for _, r := range res.Reglas {
			if r.Categoria != catActual {
				catActual = r.Categoria
				fmt.Printf("\n  [%s]\n", strings.ToUpper(catActual))
			}
			fmt.Printf("  • %s: %s\n", r.Titulo, r.Descripcion)
		}
		fmt.Println()
	}
	if len(res.Skills) > 0 {
		fmt.Printf("🛠  SKILLS DISPONIBLES:\n")
		for _, s := range res.Skills {
			fmt.Printf("  • %-30s — %s\n", s.Nombre, s.CuandoUsar)
		}
		fmt.Println()
	}
	if len(res.WorkflowPasos) > 0 {
		fmt.Printf("📌 WORKFLOW — INICIO-SESION:\n")
		for _, paso := range res.WorkflowPasos {
			fmt.Printf("  %s\n", paso)
		}
		fmt.Println()
	}
	fmt.Printf("─── Listo. Usa 'orquesta status' para ver el estado del proyecto. ───\n")
}
