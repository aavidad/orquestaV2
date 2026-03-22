/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Muestra el estado global del proyecto",
	RunE: func(cmd *cobra.Command, args []string) error {
		resumen, err := loadStatusSummary()
		if err != nil {
			return err
		}
		renderStatusSummary(resumen)
		return nil
	},
}

func truncar(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

func loadStatusSummary() (*estadoResumen, error) {
	if serverURL := activeServerURL(); serverURL != "" {
		return fetchServerStatus(serverURL)
	}
	return buildEstadoResumen()
}

func renderStatusSummary(resumen *estadoResumen) {
	fmt.Printf("╔═══════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║           ORQUESTA — ESTADO DEL PROYECTO                 ║\n")
	fmt.Printf("╚═══════════════════════════════════════════════════════════╝\n\n")

	activos := len(resumen.AgentesActivos)
	fmt.Printf("👥 Agentes: %d activos ahora\n", activos)
	for _, a := range resumen.AgentesActivos {
		fmt.Printf("   🟢 %-15s [%s]\n", a.Nombre, a.Rol)
	}
	if activos == 0 {
		fmt.Printf("   — sin agentes activos\n")
	}
	fmt.Println()

	totalTareas := 0
	completadasN := 0
	for estado, n := range resumen.TareasPorEstado {
		totalTareas += n
		if estado == string(db.EstadoCompletada) {
			completadasN = n
		}
	}
	pctStr := ""
	if totalTareas > 0 {
		pct := float64(completadasN) * 100.0 / float64(totalTareas)
		pctStr = fmt.Sprintf("  progreso: %d/%d completadas (%.0f%%)", completadasN, totalTareas, pct)
	}
	fmt.Printf("📋 Tareas —%s:\n", pctStr)
	order := []db.EstadoTarea{db.EstadoEnProgreso, db.EstadoAsignada, db.EstadoLibre, db.EstadoBacklog, db.EstadoCompletada, db.EstadoBloqueada, db.EstadoCancelada}
	for _, e := range order {
		if n := resumen.TareasPorEstado[string(e)]; n > 0 {
			fmt.Printf("   %-15s %d\n", e, n)
		}
	}
	fmt.Println()

	fmt.Printf("📣 Propuestas abiertas: %d\n", len(resumen.PropuestasAbiertas))
	for _, p := range resumen.PropuestasAbiertas {
		fmt.Printf("   %s %-40s  ✓%d ✗%d ⏳%d\n",
			p.Codigo, truncar(p.Titulo, 38), p.Acuerdo, p.Desacuerdo, p.Pendiente)
	}
	fmt.Println()

	if len(resumen.TareasActivas) > 0 {
		fmt.Printf("⚙️  En progreso ahora mismo:\n")
		for _, t := range resumen.TareasActivas {
			agente := "—"
			if t.Agente != "" {
				agente = t.Agente
			}
			fmt.Printf("   [%d] %-40s → %s\n", t.ID, truncar(t.Titulo, 38), agente)
		}
		fmt.Println()
	}
}
