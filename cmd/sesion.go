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
		nuevoCodex, _ := cmd.Flags().GetBool("nuevo-codex")

		var agente string
		if nuevoCodex {
			nombre, err := db.RegistrarCodex()
			if err != nil {
				return fmt.Errorf("registrando codex: %w", err)
			}
			agente = nombre
			fmt.Printf("✓ Nuevo agente Codex registrado como: %s\n\n", agente)
		} else {
			if len(args) == 0 {
				return fmt.Errorf("indica el nombre del agente o usa --nuevo-codex")
			}
			agente = args[0]
		}

		sesionID, err := db.IniciarSesion(agente)
		if err != nil {
			return fmt.Errorf("iniciando sesión: %w", err)
		}

		fmt.Printf("═══════════════════════════════════════════════════════════\n")
		fmt.Printf("  SESIÓN INICIADA — agente: %s  (sesion_id: %d)\n", agente, sesionID)
		fmt.Printf("═══════════════════════════════════════════════════════════\n\n")

		// Obtener el rol del agente para cargar su configuración
		agentes, err := db.ListarAgentes()
		if err != nil {
			return err
		}
		var rol string
		for _, a := range agentes {
			if a.Nombre == agente {
				rol = a.Rol
				break
			}
		}
		if rol == "" || rol == "admin" {
			// admin no necesita briefing
			fmt.Printf("Sesión iniciada. Rol: %s\n", rol)
			return nil
		}

		// ─── Propuestas pendientes de voto ───────────────────────────────
		pendientes, err := db.PropuestasPendientesVoto(agente)
		if err == nil && len(pendientes) > 0 {
			fmt.Printf("⚠️  PROPUESTAS PENDIENTES DE TU VOTO (%d):\n", len(pendientes))
			for _, p := range pendientes {
				fmt.Printf("   • %s — %s\n", p.Codigo, p.Titulo)
			}
			fmt.Println()
		}

		// ─── Reglas ──────────────────────────────────────────────────────
		reglas, err := db.GetReglasAgente(rol)
		if err != nil {
			return err
		}
		if len(reglas) > 0 {
			fmt.Printf("📋 REGLAS ACTIVAS (%s):\n", strings.ToUpper(rol))
			catActual := ""
			for _, r := range reglas {
				if r.Categoria != catActual {
					catActual = r.Categoria
					fmt.Printf("\n  [%s]\n", strings.ToUpper(catActual))
				}
				fmt.Printf("  • %s: %s\n", r.Titulo, r.Descripcion)
			}
			fmt.Println()
		}

		// ─── Skills disponibles ──────────────────────────────────────────
		skills, err := db.GetSkillsAgente(rol)
		if err != nil {
			return err
		}
		if len(skills) > 0 {
			fmt.Printf("🛠  SKILLS DISPONIBLES:\n")
			for _, s := range skills {
				fmt.Printf("  • %-30s — %s\n", s.Nombre, s.CuandoUsar)
			}
			fmt.Println()
		}

		// ─── Workflow de inicio de sesión ─────────────────────────────────
		wf, err := db.GetWorkflow(rol, "inicio-sesion")
		if err == nil {
			var pasos []string
			if err2 := json.Unmarshal([]byte(wf.Pasos), &pasos); err2 == nil {
				fmt.Printf("📌 WORKFLOW — %s:\n", strings.ToUpper(wf.Nombre))
				for _, paso := range pasos {
					fmt.Printf("  %s\n", paso)
				}
				fmt.Println()
			}
		}

		fmt.Printf("─── Listo. Usa 'orquesta status' para ver el estado del proyecto. ───\n")
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

		// Mostrar workflow de fin de sesión
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
		fmt.Printf("✓ Sesión cerrada — agente: %s\n", agente)
		return nil
	},
}

// sesion listar
var sesionListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista todos los agentes y su estado de sesión",
	RunE: func(cmd *cobra.Command, args []string) error {
		agentes, err := db.ListarAgentes()
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
		nombre, err := db.RegistrarCodex()
		if err != nil {
			return fmt.Errorf("registrando codex: %w", err)
		}
		fmt.Printf("✓ Agente registrado como: %s\n", nombre)
		sesionID, err := db.IniciarSesion(nombre)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Sesión iniciada (id: %d)\n", sesionID)
		return nil
	},
}

func init() {
	sesionInicioCmd.Flags().Bool("nuevo-codex", false, "Registra un nuevo agente Codex con nombre automático")
	sesionCmd.AddCommand(sesionInicioCmd, sesionFinCmd, sesionListarCmd, sesionNuevoCodexCmd)
}
