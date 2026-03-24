package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var auditCmd = &cobra.Command{
	Use:   "auditoria",
	Short: "Consulta el log de auditoría del sistema",
}

var auditListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista los últimos registros de auditoría",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		accion, _ := cmd.Flags().GetString("accion")
		entidad, _ := cmd.Flags().GetString("entidad")
		limite, _ := cmd.Flags().GetInt("limite")

		f := db.FiltroAuditoria{Limite: limite}
		if agente != "" {
			f.Agente = &agente
		}
		if accion != "" {
			f.Accion = &accion
		}
		if entidad != "" {
			f.Entidad = &entidad
		}

		logs := []*db.LogAuditoria(nil)
		query := url.Values{"limit": []string{fmt.Sprintf("%d", limite)}}
		if strings.TrimSpace(agente) != "" {
			query.Set("agente", strings.TrimSpace(agente))
		}
		if strings.TrimSpace(accion) != "" {
			query.Set("accion", strings.TrimSpace(accion))
		}
		if strings.TrimSpace(entidad) != "" {
			query.Set("entidad", strings.TrimSpace(entidad))
		}

		var resp apiAuditResponse
		if ok, err := apiGetQuery("/api/audit", query, &resp); err != nil {
			return err
		} else if ok {
			for _, entry := range resp.Audit {
				entryCopy := entry
				logs = append(logs, &db.LogAuditoria{
					Agente:    entryCopy.Agente,
					Accion:    entryCopy.Accion,
					Entidad:   entryCopy.Entidad,
					EntidadID: entryCopy.EntidadID,
					Detalle:   entryCopy.Detalle,
					CreatedAt: entryCopy.CreatedAt,
				})
			}
		} else {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			localLogs, err := db.ListarAuditoria(f)
			if err != nil {
				return err
			}
			logs = localLogs
		}

		if len(logs) == 0 {
			fmt.Println("No hay registros que coincidan.")
			return nil
		}

		fmt.Printf("%-5s %-16s %-18s %-12s %s\n", "ID", "AGENTE", "ACCIÓN", "ENTIDAD", "DETALLE")
		fmt.Printf("%-5s %-16s %-18s %-12s %s\n", "────", "────────────────", "──────────────────", "────────────", "─────────────────────────────")
		for _, l := range logs {
			detalle := strings.ReplaceAll(l.Detalle, "\n", " ")
			if len(detalle) > 60 {
				detalle = detalle[:57] + "..."
			}
			fmt.Printf("%-5d %-16s %-18s %-12s %s\n", l.ID, l.Agente, l.Accion, l.Entidad, detalle)
		}
		return nil
	},
}

func init() {
	auditListarCmd.Flags().String("agente", "", "Filtrar por agente")
	auditListarCmd.Flags().String("accion", "", "Filtrar por acción")
	auditListarCmd.Flags().String("entidad", "", "Filtrar por entidad (tarea, propuesta, etc.)")
	auditListarCmd.Flags().Int("limite", 50, "Número máximo de registros a mostrar")

	auditCmd.AddCommand(auditListarCmd)
	rootCmd.AddCommand(auditCmd)
}
