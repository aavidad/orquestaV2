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
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var exportarCmd = &cobra.Command{
	Use:   "exportar",
	Short: "Exporta el estado de la BD a Markdown (para git history y auditoría)",
}

var exportarEstadoCmd = &cobra.Command{
	Use:   "estado",
	Short: "Exporta tareas, propuestas y agentes a Markdown",
	RunE: func(cmd *cobra.Command, args []string) error {
		var sb strings.Builder

		sb.WriteString("# Orquesta — Estado exportado\n\n")
		sb.WriteString(fmt.Sprintf("_Generado: %s_\n\n", time.Now().Format("2006-01-02 15:04:05")))

		// ─── Agentes ──────────────────────────────────────────────────────
		sb.WriteString("## Agentes\n\n")
		sb.WriteString("| Agente | Rol | Activo | Última sesión |\n")
		sb.WriteString("|--------|-----|--------|---------------|\n")
		var agentes []*db.Agente
		var propuestas []*db.Propuesta
		var tareas []*db.Tarea
		usaAPI := false
		var agentesResp apiAgentesResponse
		if ok, err := apiGet("/api/agentes", &agentesResp); err != nil {
			return err
		} else if ok {
			usaAPI = true
			agentes = agentesResp.Agentes
			var propuestasResp apiPropuestasResponse
			if _, err := apiGet("/api/propuestas", &propuestasResp); err != nil {
				return err
			}
			propuestas = propuestasResp.Propuestas
			var tareasResp apiTareasResponse
			if _, err := apiGet("/api/tareas", &tareasResp); err != nil {
				return err
			}
			tareas = tareasResp.Tareas
		} else {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			agentes, _ = db.ListarAgentes()
			propuestas, _ = db.ListarPropuestas(nil, nil)
			tareas, _ = db.ListarTareas(db.FiltroTareas{})
		}
		for _, a := range agentes {
			activo := "no"
			if a.Activo {
				activo = "sí"
			}
			ultima := "—"
			if a.UltimaSesion != nil {
				ultima = a.UltimaSesion.Format("2006-01-02 15:04")
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", a.Nombre, a.Rol, activo, ultima))
		}
		sb.WriteString("\n")

		// ─── Propuestas ────────────────────────────────────────────────────
		sb.WriteString("## Propuestas\n\n")
		for _, p := range propuestas {
			if len(p.Votos) == 0 {
				var resp apiPropuestaDetalleResponse
				if ok, err := apiGet("/api/propuestas/"+p.Codigo, &resp); err == nil && ok && resp.Propuesta != nil {
					p = resp.Propuesta
				} else if !usaAPI && db.DB != nil {
					p.Votos, _ = db.ResumenVotos(p.ID)
				}
			}
			cerrada := ""
			if p.CerradaAt != nil {
				cerrada = fmt.Sprintf(" _(cerrada: %s)_", p.CerradaAt.Format("2006-01-02"))
			}
			sb.WriteString(fmt.Sprintf("### %s — %s\n\n", p.Codigo, p.Titulo))
			sb.WriteString(fmt.Sprintf("- **Estado:** %s%s\n", p.Estado, cerrada))
			sb.WriteString(fmt.Sprintf("- **Tipo:** %s\n", p.Tipo))
			sb.WriteString(fmt.Sprintf("- **Propuesto por:** %s  —  %s\n",
				p.PropuestoPor, p.CreatedAt.Format("2006-01-02")))
			if p.Descripcion != "" {
				sb.WriteString(fmt.Sprintf("- **Descripción:** %s\n", p.Descripcion))
			}

			if len(p.Votos) > 0 {
				sb.WriteString("\n**Votos:**\n\n")
				sb.WriteString("| Agente | Posición | Comentario |\n")
				sb.WriteString("|--------|----------|------------|\n")
				for _, v := range p.Votos {
					sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
						v.Agente, v.Posicion, v.Comentario))
				}
			}
			sb.WriteString("\n")
		}

		// ─── Tareas ────────────────────────────────────────────────────────
		sb.WriteString("## Tareas\n\n")
		sb.WriteString("| # | Título | Estado | Prioridad | Agente | Módulo |\n")
		sb.WriteString("|---|--------|--------|-----------|--------|--------|\n")
		total := len(tareas)
		completadas := 0
		for _, t := range tareas {
			agente := "—"
			if t.Agente != nil {
				agente = *t.Agente
			}
			if t.Estado == db.EstadoCompletada {
				completadas++
			}
			sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %s |\n",
				t.ID, t.Titulo, t.Estado, t.Prioridad, agente, t.Modulo))
		}
		if total > 0 {
			pct := float64(completadas) * 100.0 / float64(total)
			sb.WriteString(fmt.Sprintf("\n**Progreso: %d/%d completadas (%.0f%%)**\n\n", completadas, total, pct))
		}

		fmt.Print(sb.String())
		return nil
	},
}

var exportarAuditCmd = &cobra.Command{
	Use:   "audit [n]",
	Short: "Exporta las últimas N entradas del log de auditoría (default: 50)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit := 50
		if len(args) == 1 {
			if _, err := fmt.Sscanf(args[0], "%d", &limit); err != nil {
				return fmt.Errorf("n debe ser un número")
			}
		}
		var entries []db.AuditEntry
		var auditResp apiAuditResponse
		if ok, err := apiGetQuery("/api/audit", url.Values{"limit": []string{fmt.Sprintf("%d", limit)}}, &auditResp); err != nil {
			return err
		} else if ok {
			entries = auditResp.Audit
		} else {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			var err error
			entries, err = db.AuditLog(limit)
			if err != nil {
				return err
			}
		}
		fmt.Printf("# Audit Log (últimas %d entradas)\n\n", limit)
		fmt.Printf("| Fecha | Agente | Acción | Entidad | ID | Detalle |\n")
		fmt.Printf("|-------|--------|--------|---------|----|---------|\n")
		for _, e := range entries {
			fmt.Printf("| %s | %s | %s | %s | %d | %s |\n",
				e.CreatedAt.Format("2006-01-02 15:04"),
				e.Agente, e.Accion, e.Entidad, e.EntidadID, e.Detalle)
		}
		return nil
	},
}

func init() {
	exportarCmd.AddCommand(exportarEstadoCmd, exportarAuditCmd)
}
