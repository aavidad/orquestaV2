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

	"github.com/spf13/cobra"
	"orquesta/db"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Muestra el estado global del proyecto",
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			agentes             []*db.Agente
			counts              map[string]int
			proyectos           []*db.Proyecto
			asignacionesActivas map[int64]int
			sesionesPorProyecto map[int64]int
			abiertas            []*db.Propuesta
			tareas              []*db.Tarea
		)
		var statusResp apiStatusResponse
		if ok, err := apiGet("/api/status", &statusResp); err != nil {
			return err
		} else if ok {
			agentes = statusResp.Agentes
			counts = statusResp.ConteoTareas
			proyectos = statusResp.Proyectos
			asignacionesActivas = statusResp.AsignacionesActivas
			sesionesPorProyecto = statusResp.SesionesActivas
			abiertas = statusResp.PropuestasAbiertas
			for _, t := range counts {
				_ = t
			}
			var tareasResp apiTareasResponse
			if ok, err := apiGetQuery("/api/tareas", mapToValues(map[string]string{"estado": string(db.EstadoEnProgreso)}), &tareasResp); err != nil {
				return err
			} else if ok {
				tareas = tareasResp.Tareas
			}
		} else {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			var err error
			agentes, err = db.ListarAgentes()
			if err != nil {
				return err
			}
			counts, err = db.ContarTareasPorEstado()
			if err != nil {
				return err
			}
			proyectos, err = db.ListarProyectos(db.FiltroProyectos{})
			if err != nil {
				return err
			}
			asignacionesActivas, err = db.ContarAsignacionesActivasPorProyecto()
			if err != nil {
				return err
			}
			sesionesActivas, err := db.ListarSesionesActivas()
			if err != nil {
				return err
			}
			sesionesPorProyecto = make(map[int64]int)
			for _, s := range sesionesActivas {
				if s.ProyectoID != nil {
					sesionesPorProyecto[*s.ProyectoID]++
				}
			}
			estado := db.PropuestaAbierta
			abiertas, err = db.ListarPropuestas(&estado, nil)
			if err != nil {
				return err
			}
			enProgreso := db.EstadoEnProgreso
			tareas, err = db.ListarTareas(db.FiltroTareas{Estado: &enProgreso})
			if err != nil {
				return err
			}
		}

		fmt.Printf("╔═══════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║           ORQUESTA — ESTADO DEL PROYECTO                 ║\n")
		fmt.Printf("╚═══════════════════════════════════════════════════════════╝\n\n")

		// ─── Agentes activos ─────────────────────────────────────────────
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
			fmt.Printf("   %s %-15s [%s]\n", estado, a.Nombre, a.Rol)
		}
		fmt.Println()

		// ─── Proyectos y asignaciones ───────────────────────────────────
		if len(proyectos) > 0 {
			fmt.Printf("🗂  Proyectos registrados: %d\n", len(proyectos))
			for _, p := range proyectos {
				fmt.Printf("   %-12s [%s]  asignados=%d  sesiones_activas=%d\n",
					p.Slug, p.Tipo, asignacionesActivas[p.ID], sesionesPorProyecto[p.ID])
			}
			fmt.Println()
		}

		// ─── Tareas ──────────────────────────────────────────────────────
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
		fmt.Printf("📣 Propuestas abiertas: %d\n", len(abiertas))
		for _, p := range abiertas {
			ac, des, _, pend := 0, 0, 0, 0
			for _, v := range p.Votos {
				switch v.Posicion {
				case "acuerdo":
					ac++
				case "desacuerdo":
					des++
				case "pendiente":
					pend++
				}
			}
			fmt.Printf("   %s %-40s  ✓%d ✗%d ⏳%d\n",
				p.Codigo, truncar(p.Titulo, 38), ac, des, pend)
		}
		fmt.Println()

		// ─── Tareas en progreso ───────────────────────────────────────────
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

func mapToValues(entries map[string]string) url.Values {
	values := url.Values{}
	for key, value := range entries {
		if value != "" {
			values.Set(key, value)
		}
	}
	return values
}
