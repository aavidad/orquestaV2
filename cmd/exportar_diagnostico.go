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
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var exportarDiagnosticoCmd = &cobra.Command{
	Use:   "diagnostico",
	Short: "Exporta un snapshot de observabilidad y diagnóstico",
	RunE: func(cmd *cobra.Command, args []string) error {
		limitAudit, _ := cmd.Flags().GetInt("audit-limit")
		asJSON, _ := cmd.Flags().GetBool("json")

		var snapshot *db.SnapshotDiagnostico
		query := url.Values{"audit_limit": []string{fmt.Sprintf("%d", limitAudit)}}
		var resp apiDiagnosticoResponse
		if ok, err := apiGetQuery("/api/diagnostico", query, &resp); err != nil {
			return err
		} else if ok {
			snapshot = &resp.Diagnostico
		} else {
			var err error
			snapshot, err = db.ConstruirSnapshotDiagnostico(limitAudit)
			if err != nil {
				return err
			}
		}

		if asJSON {
			out, err := json.MarshalIndent(snapshot, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		}

		var sb strings.Builder
		sb.WriteString("# Orquesta — Diagnóstico\n\n")
		sb.WriteString(fmt.Sprintf("_Generado: %s_\n\n", snapshot.GeneradoEn.Format("2006-01-02 15:04:05")))

		agentesActivos := 0
		for _, agente := range snapshot.Agentes {
			if agente.Activo {
				agentesActivos++
			}
		}
		sb.WriteString("## Resumen\n\n")
		sb.WriteString(fmt.Sprintf("- Agentes registrados: %d\n", len(snapshot.Agentes)))
		sb.WriteString(fmt.Sprintf("- Agentes activos: %d\n", agentesActivos))
		sb.WriteString(fmt.Sprintf("- Sesiones activas: %d\n", len(snapshot.SesionesActivas)))
		sb.WriteString(fmt.Sprintf("- Propuestas abiertas: %d\n", len(snapshot.PropuestasAbiertas)))
		sb.WriteString(fmt.Sprintf("- Locks activos: %d\n", len(snapshot.LocksActivos)))
		sb.WriteString(fmt.Sprintf("- Worktrees activos: %d\n", len(snapshot.WorktreesActivos)))
		sb.WriteString("\n")

		sb.WriteString("## Conteo de tareas\n\n")
		for _, estado := range []string{"libre", "asignada", "en_progreso", "bloqueada", "completada", "backlog", "cancelada"} {
			if n, ok := snapshot.ConteoTareas[estado]; ok {
				sb.WriteString(fmt.Sprintf("- %s: %d\n", estado, n))
			}
		}
		sb.WriteString("\n")

		if len(snapshot.SesionesActivas) > 0 {
			sb.WriteString("## Sesiones activas\n\n")
			sb.WriteString("| ID | Agente | Proyecto | Estado | Branch | External ID |\n")
			sb.WriteString("|----|--------|----------|--------|--------|-------------|\n")
			for _, sesion := range snapshot.SesionesActivas {
				proyecto := "—"
				if sesion.ProyectoSlug != "" {
					proyecto = sesion.ProyectoSlug
				}
				branch := "—"
				if sesion.Branch != "" {
					branch = sesion.Branch
				}
				externalID := "—"
				if sesion.ExternalSessionID != "" {
					externalID = sesion.ExternalSessionID
				}
				sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %s |\n",
					sesion.ID, sesion.Agente, proyecto, sesion.Estado, branch, externalID))
			}
			sb.WriteString("\n")
		}

		if len(snapshot.TareasBloqueadas) > 0 {
			sb.WriteString("## Tareas bloqueadas\n\n")
			sb.WriteString("| ID | Título | Agente | Módulo |\n")
			sb.WriteString("|----|--------|--------|--------|\n")
			for _, tarea := range snapshot.TareasBloqueadas {
				agente := "—"
				if tarea.Agente != nil {
					agente = *tarea.Agente
				}
				sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s |\n", tarea.ID, tarea.Titulo, agente, tarea.Modulo))
			}
			sb.WriteString("\n")
		}

		if len(snapshot.PropuestasAbiertas) > 0 {
			sb.WriteString("## Propuestas abiertas\n\n")
			sb.WriteString("| Código | Título | ✓ | ✗ | ~ | ⏳ |\n")
			sb.WriteString("|--------|--------|---|---|---|----|\n")
			for _, propuesta := range snapshot.PropuestasAbiertas {
				sb.WriteString(fmt.Sprintf("| %s | %s | %d | %d | %d | %d |\n",
					propuesta.Propuesta.Codigo, propuesta.Propuesta.Titulo,
					propuesta.Acuerdo, propuesta.Desacuerdo, propuesta.Abstencion, propuesta.Pendiente))
			}
			sb.WriteString("\n")
		}

		if len(snapshot.Config) > 0 {
			sb.WriteString("## Configuración\n\n")
			sb.WriteString("| Clave | Valor |\n")
			sb.WriteString("|-------|-------|\n")
			for _, k := range db.ClavesOrdenadasConfig(snapshot.Config) {
				sb.WriteString(fmt.Sprintf("| %s | %s |\n", k, snapshot.Config[k]))
			}
			sb.WriteString("\n")
		}

		if len(snapshot.AuditoriaReciente) > 0 {
			sb.WriteString("## Auditoría reciente\n\n")
			sb.WriteString("| Fecha | Agente | Acción | Entidad | ID | Detalle |\n")
			sb.WriteString("|-------|--------|--------|---------|----|---------|\n")
			for _, entrada := range snapshot.AuditoriaReciente {
				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d | %s |\n",
					entrada.CreatedAt.Format("2006-01-02 15:04"),
					entrada.Agente, entrada.Accion, entrada.Entidad, entrada.EntidadID, entrada.Detalle))
			}
		}

		fmt.Print(sb.String())
		return nil
	},
}

func init() {
	exportarDiagnosticoCmd.Flags().Int("audit-limit", 20, "Número de entradas de auditoría a incluir")
	exportarDiagnosticoCmd.Flags().Bool("json", false, "Emitir el diagnóstico en JSON en lugar de Markdown")
	exportarCmd.AddCommand(exportarDiagnosticoCmd)
}
