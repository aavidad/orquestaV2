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

	"github.com/spf13/cobra"
	"orquesta/db"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Muestra el log de auditoría del sistema",
	Long: `Muestra las últimas acciones registradas en el log de auditoría (audit_log).
Permite filtrar por agente, acción o entidad para rastrear cambios y eventos.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		agenteF, _ := cmd.Flags().GetString("agente")
		accionF, _ := cmd.Flags().GetString("accion")
		entidadF, _ := cmd.Flags().GetString("entidad")

		query := url.Values{"limit": []string{fmt.Sprintf("%d", limit)}}
		if strings.TrimSpace(agenteF) != "" {
			query.Set("agente", strings.TrimSpace(agenteF))
		}
		if strings.TrimSpace(accionF) != "" {
			query.Set("accion", strings.TrimSpace(accionF))
		}
		if strings.TrimSpace(entidadF) != "" {
			query.Set("entidad", strings.TrimSpace(entidadF))
		}

		var entries []db.AuditEntry
		var resp apiAuditResponse
		if ok, err := apiGetQuery("/api/audit", query, &resp); err != nil {
			return err
		} else if ok {
			entries = resp.Audit
		} else {
			return serverFirstCommandError("logs")
		}

		if len(entries) == 0 {
			fmt.Println("No hay entradas en el log de auditoría.")
			return nil
		}

		fmt.Printf("%-18s %-12s %-16s %-10s %-8s %s\n",
			"FECHA", "AGENTE", "ACCIÓN", "ENTIDAD", "ID", "DETALLE")
		fmt.Printf("%-18s %-12s %-16s %-10s %-8s %s\n",
			"──────────────────", "────────────", "────────────────", "──────────", "────────", "──────────────────────────────────")

		for _, e := range entries {
			if agenteF != "" && !strings.Contains(strings.ToLower(e.Agente), strings.ToLower(agenteF)) {
				continue
			}
			if accionF != "" && !strings.Contains(strings.ToLower(e.Accion), strings.ToLower(accionF)) {
				continue
			}
			if entidadF != "" && !strings.Contains(strings.ToLower(e.Entidad), strings.ToLower(entidadF)) {
				continue
			}

			entidadID := "—"
			if e.EntidadID > 0 {
				entidadID = fmt.Sprintf("%d", e.EntidadID)
			}

			fmt.Printf("%-18s %-12s %-16s %-10s %-8s %s\n",
				e.CreatedAt.Format("2006-01-02 15:04"),
				truncar(e.Agente, 12),
				truncar(e.Accion, 16),
				truncar(e.Entidad, 10),
				entidadID,
				e.Detalle,
			)
		}

		return nil
	},
}

func init() {
	logsCmd.Flags().Int("limit", 50, "Número de entradas a mostrar")
	logsCmd.Flags().String("agente", "", "Filtrar por agente")
	logsCmd.Flags().String("accion", "", "Filtrar por acción")
	logsCmd.Flags().String("entidad", "", "Filtrar por entidad (tarea, propuesta, sesion...)")
}
