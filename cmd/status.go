/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
<<<<<<< HEAD
	"net/url"
=======
	"strings"
>>>>>>> origin/orq-orquestador-codex2

	"github.com/spf13/cobra"
	"orquesta/db"
)

type statusContext struct {
	resumen *estadoResumen
	backend *serverInfo
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Muestra el estado global del proyecto",
	RunE: func(cmd *cobra.Command, args []string) error {
<<<<<<< HEAD
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
			presupuesto := ""
			if a.Activo {
				presupuesto = " · presupuesto: " + presupuestoResumenAgente(a.Nombre)
			}
			fmt.Printf("   %s %-15s [%s]%s\n", estado, a.Nombre, a.Rol, presupuesto)
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
			if len(p.Votos) == 0 && db.DB != nil {
				ac, des, _, pend, _ = db.ContarVotos(p.ID)
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

=======
		ctx, err := loadStatusSummary()
		if err != nil {
			return err
		}
		renderStatusSummary(ctx)
>>>>>>> origin/orq-orquestador-codex2
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

<<<<<<< HEAD
func mapToValues(entries map[string]string) url.Values {
	values := url.Values{}
	for key, value := range entries {
		if value != "" {
			values.Set(key, value)
		}
	}
	return values
=======
func loadStatusSummary() (*statusContext, error) {
	if serverURL := activeServerURL(); serverURL != "" {
		resumen, err := fetchServerStatus(serverURL)
		if err != nil {
			return nil, err
		}
		backend, err := fetchServerInfo(serverURL)
		if err != nil {
			backend = nil
		}
		return &statusContext{resumen: resumen, backend: backend}, nil
	}
	resumen, err := buildEstadoResumen()
	if err != nil {
		return nil, err
	}
	return &statusContext{resumen: resumen}, nil
}

func renderStatusSummary(ctx *statusContext) {
	if ctx == nil {
		return
	}

	if lines := serverInfoLines(ctx.backend); len(lines) > 0 {
		for _, line := range lines {
			fmt.Println(line)
		}
		fmt.Println()
	}

	resumen := ctx.resumen
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

func serverInfoLines(info *serverInfo) []string {
	if info == nil {
		return nil
	}

	backend := "backend remoto"
	if info.Name != "" {
		backend = info.Name
		if info.Version != "" {
			backend += " " + info.Version
		}
	} else if info.Version != "" {
		backend = info.Version
	}

	details := make([]string, 0, 5)
	if info.StorageMode != "" {
		details = append(details, "modo "+info.StorageMode)
	}
	if info.StorageDriver != "" {
		details = append(details, "driver "+info.StorageDriver)
	}
	if info.SQLPlaceholder != "" {
		details = append(details, "placeholder "+info.SQLPlaceholder)
	}
	if info.BootstrapSchema {
		details = append(details, "bootstrap schema")
	}
	if info.QueryRebinding {
		details = append(details, "query rebinding")
	}

	lines := []string{fmt.Sprintf("🖥️  Backend activo: %s", backend)}
	if len(details) > 0 {
		lines = append(lines, fmt.Sprintf("   %s", strings.Join(details, " | ")))
	}
	if len(info.Capabilities) > 0 {
		lines = append(lines, fmt.Sprintf("   capacidades: %s", strings.Join(info.Capabilities, ", ")))
	}
	return lines
>>>>>>> origin/orq-orquestador-codex2
}
