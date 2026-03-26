/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var sesionHistorialCmd = &cobra.Command{
	Use:   "historial",
	Short: "Lista sesiones con filtros de inspección",
	RunE: func(cmd *cobra.Command, args []string) error {
		params := url.Values{}
		proyectoRef, _ := cmd.Flags().GetString("proyecto")

		if agente, _ := cmd.Flags().GetString("agente"); strings.TrimSpace(agente) != "" {
			agente = strings.TrimSpace(agente)
			params.Set("agente", agente)
		}
		if strings.TrimSpace(proyectoRef) != "" {
			params.Set("proyecto", strings.TrimSpace(proyectoRef))
		}
		if activaStr, _ := cmd.Flags().GetString("activa"); strings.TrimSpace(activaStr) != "" {
			params.Set("activa", strings.TrimSpace(activaStr))
			switch activaStr {
			case "true":
			case "false":
			default:
				return fmt.Errorf("activa debe ser true o false")
			}
		}
		if estado, _ := cmd.Flags().GetString("estado"); strings.TrimSpace(estado) != "" {
			estado = strings.TrimSpace(estado)
			params.Set("estado", estado)
		}

		var sesiones []*db.Sesion
		var resp apiSesionesInspeccionResponse
		if ok, err := apiGetQuery("/api/sesiones", params, &resp); err != nil {
			return err
		} else if ok {
			sesiones = resp.Sesiones
		} else {
			return serverFirstCommandError("sesion historial")
		}

		fmt.Printf("%-5s %-12s %-14s %-8s %-12s %-14s %s\n", "ID", "AGENTE", "PROYECTO", "ACTIVA", "ESTADO", "HERRAMIENTA", "EXTERNAL_ID")
		fmt.Printf("%-5s %-12s %-14s %-8s %-12s %-14s %s\n", "─────", "────────────", "──────────────", "────────", "────────────", "──────────────", "────────────")
		for _, sesion := range sesiones {
			proyecto := "—"
			if sesion.ProyectoSlug != "" {
				proyecto = sesion.ProyectoSlug
			}
			activa := "no"
			if sesion.Activa {
				activa = "sí"
			}
			herramienta := "—"
			if sesion.Herramienta != "" {
				herramienta = sesion.Herramienta
			}
			externalID := "—"
			if sesion.ExternalSessionID != "" {
				externalID = sesion.ExternalSessionID
			}
			fmt.Printf("%-5d %-12s %-14s %-8s %-12s %-14s %s\n",
				sesion.ID, sesion.Agente, proyecto, activa, sesion.Estado, herramienta, externalID)
		}
		return nil
	},
}

var sesionVerCmd = &cobra.Command{
	Use:   "ver <id>",
	Short: "Muestra el detalle de una sesión por ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		var s *db.Sesion
		var resp apiSesionResponse
		if ok, err := apiGet("/api/sesiones/"+strconv.FormatInt(id, 10), &resp); err != nil {
			return err
		} else if ok {
			s = resp.Sesion
		} else {
			return serverFirstCommandError("sesion ver")
		}
		fmt.Printf("Sesión %d\n", s.ID)
		fmt.Printf("  Agente:      %s\n", s.Agente)
		if s.ProyectoSlug != "" {
			fmt.Printf("  Proyecto:    %s\n", s.ProyectoSlug)
		}
		fmt.Printf("  Activa:      %t\n", s.Activa)
		fmt.Printf("  Estado:      %s\n", s.Estado)
		if s.Herramienta != "" {
			fmt.Printf("  Herramienta: %s\n", s.Herramienta)
		}
		if s.ConectorSlug != "" {
			fmt.Printf("  Conector:    %s\n", s.ConectorSlug)
		}
		if s.CWD != "" {
			fmt.Printf("  CWD:         %s\n", s.CWD)
		}
		if s.Branch != "" {
			fmt.Printf("  Branch:      %s\n", s.Branch)
		}
		if s.ExternalSessionID != "" {
			fmt.Printf("  External ID: %s\n", s.ExternalSessionID)
		}
		if s.ResumenContinuidad != "" {
			fmt.Printf("  Resumen:     %s\n", s.ResumenContinuidad)
		}
		if s.Host != "" {
			fmt.Printf("  Host:        %s\n", s.Host)
		}
		if s.PID != nil {
			fmt.Printf("  PID:         %d\n", *s.PID)
		}
		if s.HeartbeatAt != nil {
			fmt.Printf("  Heartbeat:   %s\n", s.HeartbeatAt.Format("2006-01-02 15:04:05"))
		}
		return nil
	},
}

func init() {
	sesionHistorialCmd.Flags().String("agente", "", "Filtra por agente")
	sesionHistorialCmd.Flags().String("proyecto", "", "Filtra por proyecto")
	sesionHistorialCmd.Flags().String("activa", "", "Filtra por activa=true|false")
	sesionHistorialCmd.Flags().String("estado", "", "Filtra por estado lógico")

	sesionCmd.AddCommand(sesionHistorialCmd, sesionVerCmd)
}
