package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Operaciones globales sobre el workspace",
}

var workspaceControlCmd = &cobra.Command{
	Use:   "control",
	Short: "Resume el estado global operativo del workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		report, err := loadWorkspaceControlReport()
		if err != nil {
			return err
		}
		if jsonOut {
			return imprimirJSON(apiWorkspaceControlResponse{Control: report})
		}
		return renderWorkspaceControl(cmd.OutOrStdout(), report)
	},
}

func init() {
	workspaceControlCmd.Flags().Bool("json", false, "Salida JSON")
	workspaceCmd.AddCommand(workspaceControlCmd)
	rootCmd.AddCommand(workspaceCmd)
}

func loadWorkspaceControlReport() (*workspaceControlReport, error) {
	report, ok, err := cargarWorkspaceControlDesdeAPI()
	if err != nil {
		return nil, err
	}
	if !ok || report == nil {
		return nil, serverFirstCommandError("workspace control")
	}
	return report, nil
}

func renderWorkspaceControl(w io.Writer, report *workspaceControlReport) error {
	if report == nil {
		return fmt.Errorf("control de workspace vacío")
	}
	fmt.Fprintf(w, "Workspace:  proyectos=%d\n", report.ActiveProjects)
	fmt.Fprintf(w, "Workers:    conectados=%d trabajando=%d supervisores=%d\n",
		report.WorkersConectados,
		report.WorkersTrabajando,
		report.SupervisoresActivos,
	)
	fmt.Fprintf(w, "Autonomía:  supervisor=%d continuando=%d pendiente=%d confirmada=%d eventos=%d\n",
		report.Autonomia.Supervisando,
		report.Autonomia.Continuando,
		report.Autonomia.ContinuidadPendiente,
		report.Autonomia.WorkConfirmed,
		report.Autonomia.Count,
	)
	fmt.Fprintf(w, "Dispatch:   total=%d pending=%d notified=%d failed=%d confirmed=%d\n",
		report.DeudaDispatch.Total,
		report.DeudaDispatch.Pendientes,
		report.DeudaDispatch.Notificadas,
		report.DeudaDispatch.Fallidas,
		report.DeudaDispatch.WorkConfirmed,
	)
	fmt.Fprintf(w, "Tareas:     ")
	keys := make([]string, 0, len(report.TaskCounts))
	for key := range report.TaskCounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for i, key := range keys {
		if i > 0 {
			fmt.Fprintf(w, " · ")
		}
		fmt.Fprintf(w, "%s=%d", key, report.TaskCounts[key])
	}
	fmt.Fprintln(w)
	if report.AutonomySurface != nil {
		fmt.Fprintf(w, "Autonomy:   %s\n", formatAutonomySurfaceSummary(report.AutonomySurface))
	}
	if len(report.AutonomyHighlights) > 0 {
		fmt.Fprintf(w, "Highlights: %s\n", strings.Join(report.AutonomyHighlights, " | "))
	}
	if len(report.AutonomyProjects) > 0 {
		fmt.Fprintf(w, "Resumen por proyecto:\n")
		for _, item := range report.AutonomyProjects {
			fmt.Fprintf(w, "  - %s: %d evento(s)", strings.TrimSpace(item.Project), item.Events)
			if item.Blocking > 0 {
				fmt.Fprintf(w, " · integracion_bloqueada=%d", item.Blocking)
			}
			if item.LastAt != nil {
				fmt.Fprintf(w, " · %s", item.LastAt.Format("2006-01-02 15:04:05"))
			}
			if len(item.Highlights) > 0 {
				fmt.Fprintf(w, " · %s", strings.Join(item.Highlights, " | "))
			}
			fmt.Fprintln(w)
		}
	}
	if len(report.AutonomyRecent) > 0 {
		fmt.Fprintf(w, "Timeline:\n")
		for _, item := range report.AutonomyRecent {
			fmt.Fprintf(w, "  - %s · %s", strings.TrimSpace(item.Project), strings.TrimSpace(item.Kind))
			if item.TargetAgent != "" {
				fmt.Fprintf(w, " · destino=%s", strings.TrimSpace(item.TargetAgent))
			} else if item.Agent != "" {
				fmt.Fprintf(w, " · agente=%s", strings.TrimSpace(item.Agent))
			}
			fmt.Fprintf(w, " · %s\n", item.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	}
	for _, cockpit := range report.Projects {
		if cockpit == nil || cockpit.Proyecto == nil {
			continue
		}
		fmt.Fprintf(w, "Proyecto:   %s · autonomy=%d · mailbox_rt=%d · orders_rt=%d\n",
			strings.TrimSpace(cockpit.Proyecto.Slug),
			cockpit.AutonomyEvents,
			cockpit.RuntimeMailboxPendiente,
			cockpit.RuntimeOrdersAbiertas,
		)
	}
	return nil
}
