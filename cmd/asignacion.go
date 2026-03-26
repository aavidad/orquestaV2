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
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var asignacionCmd = &cobra.Command{
	Use:   "asignacion",
	Short: "Asignación de agentes a proyectos",
}

func asignacionModoRecuperacionLocalExplicito() bool {
	return strings.TrimSpace(os.Getenv("ORQUESTA_FORCE_LOCAL_DB")) == "1"
}

func asignacionErrorServerFirst() error {
	return fmt.Errorf("este comando exige servidor/daemon de Orquesta; usa --local solo en recuperacion explicita o exporta ORQUESTA_FORCE_LOCAL_DB=1")
}

var asignacionListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista asignaciones de agentes a proyectos",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		estadoStr, _ := cmd.Flags().GetString("estado")

		query := url.Values{}
		if agente != "" {
			query.Set("agente", agente)
		}
		if proyectoRef != "" {
			query.Set("proyecto", proyectoRef)
		}
		if estadoStr != "" {
			query.Set("estado", estadoStr)
		}

		asignaciones, ok, err := cargarAsignacionesDesdeAPI(query)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("asignacion listar")
		}
		if len(asignaciones) == 0 {
			fmt.Println("No hay asignaciones con ese filtro.")
			return nil
		}
		fmt.Printf("%-5s %-12s %-15s %-12s %s\n", "ID", "AGENTE", "PROYECTO", "ESTADO", "NOTA")
		fmt.Printf("%-5s %-12s %-15s %-12s %s\n", "─────", "────────────", "───────────────", "────────────", "────────────────────────────")
		for _, a := range asignaciones {
			fmt.Printf("%-5d %-12s %-15s %-12s %s\n", a.ID, a.Agente, a.ProyectoSlug, a.Estado, truncar(a.Nota, 30))
		}
		return nil
	},
}

var asignacionActivarCmd = &cobra.Command{
	Use:   "activar <agente> <proyecto>",
	Short: "Activa la asignación de un agente a un proyecto",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		nota, _ := cmd.Flags().GetString("nota")
		ok, err := activarAsignacionPorAPI(apiAsignacionActivarRequest{
			Agente:   args[0],
			Proyecto: args[1],
			Nota:     nota,
		})
		if err != nil {
			return err
		}
		if ok {
			fmt.Printf("✓ %s asignado a %s\n", args[0], args[1])
			return nil
		}
		return serverFirstCommandError("asignacion activar")
	},
}

func init() {
	asignacionListarCmd.Flags().String("agente", "", "Filtrar por agente")
	asignacionListarCmd.Flags().String("proyecto", "", "Filtrar por proyecto")
	asignacionListarCmd.Flags().String("estado", "", "Filtrar por estado")
	asignacionActivarCmd.Flags().String("nota", "", "Nota de asignación")

	asignacionCmd.AddCommand(asignacionListarCmd, asignacionActivarCmd)
}
