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
		fmt.Printf("╔═══════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║           ORQUESTA — ESTADO DEL PROYECTO                 ║\n")
		fmt.Printf("╚═══════════════════════════════════════════════════════════╝\n\n")

		// ─── Agentes activos ─────────────────────────────────────────────
		agentes, err := db.ListarAgentes()
		if err != nil {
			return err
		}
		activos := 0
		for _, a := range agentes {
			if a.Activo {
				activos++
			}
		}
		fmt.Printf("👥 Agentes: %d registrados, %d activos ahora\n", len(agentes), activos)
		for _, a := range agentes {
			estado := "  "
			if a.Activo {
				estado = "🟢"
			}
			presupuesto := ""
			if a.Activo {
				presupuesto = " · presupuesto: " + presupuestoResumenAgente(a.Nombre)
			}
			fmt.Printf("   %s %-15s [%s]%s\n", estado, a.Nombre, a.Rol, presupuesto)
		}
		fmt.Println()

		// ─── Tareas ──────────────────────────────────────────────────────
		counts, err := db.ContarTareasPorEstado()
		if err != nil {
			return err
		}
		totalTareas := 0
		completadasN := 0
		for estado, n := range counts {
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
			if n := counts[string(e)]; n > 0 {
				fmt.Printf("   %-15s %d\n", e, n)
			}
		}
		fmt.Println()

		// ─── Propuestas abiertas ─────────────────────────────────────────
		estado := db.PropuestaAbierta
		abiertas, err := db.ListarPropuestas(&estado)
		if err != nil {
			return err
		}
		fmt.Printf("📣 Propuestas abiertas: %d\n", len(abiertas))
		for _, p := range abiertas {
			ac, des, _, pend, _ := db.ContarVotos(p.ID)
			fmt.Printf("   %s %-40s  ✓%d ✗%d ⏳%d\n",
				p.Codigo, truncar(p.Titulo, 38), ac, des, pend)
		}
		fmt.Println()

		// ─── Tareas en progreso ───────────────────────────────────────────
		enProgreso := db.EstadoEnProgreso
		tareas, err := db.ListarTareas(db.FiltroTareas{Estado: &enProgreso})
		if err != nil {
			return err
		}
		if len(tareas) > 0 {
			fmt.Printf("⚙️  En progreso ahora mismo:\n")
			for _, t := range tareas {
				agente := "—"
				if t.Agente != nil {
					agente = *t.Agente
				}
				fmt.Printf("   [%d] %-40s → %s\n", t.ID, truncar(t.Titulo, 38), agente)
			}
			fmt.Println()
		}

		// ─── Presupuestos en handoff ─────────────────────────────────────
		enRiesgo := 0
		for _, a := range agentes {
			if !a.Activo {
				continue
			}
			p, _, err := db.UltimoPresupuestoAgente(a.Nombre)
			if err != nil {
				continue
			}
			ev, err := db.EvaluarPresupuestoSesion(p)
			if err != nil || !ev.DebeHandoff {
				continue
			}
			if enRiesgo == 0 {
				fmt.Printf("⏳ Presupuestos en handoff preventivo:\n")
			}
			enRiesgo++
			fmt.Printf("   %-15s %s\n", a.Nombre, ev.Motivo)
		}
		if enRiesgo > 0 {
			fmt.Println()
		}

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
