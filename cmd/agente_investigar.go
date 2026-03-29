package cmd

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/agentesapp"
)

var agenteInvestigarCmd = &cobra.Command{
	Use:   "investigar <texto...>",
	Short: "Busca evidencia de actividad de agentes y la atribuye por transcript/traza",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.TrimSpace(strings.Join(args, " "))
		proyecto, _ := cmd.Flags().GetString("proyecto")
		limit, _ := cmd.Flags().GetInt("limit")
		out, ok, err := investigarAgentesPorAPI(query, proyecto, limit)
		if ok {
			if err != nil {
				return err
			}
			return imprimirAgenteInvestigacion(out)
		}
		return serverFirstCommandError("agente investigar")
	},
}

func imprimirAgenteInvestigacion(out *agentesapp.InvestigationReport) error {
	if out == nil {
		fmt.Println("No hay investigación disponible.")
		return nil
	}
	proyecto := "todos"
	if out.Proyecto != nil && strings.TrimSpace(out.Proyecto.Slug) != "" {
		proyecto = strings.TrimSpace(out.Proyecto.Slug)
	}
	fmt.Printf("Consulta:  %s\n", strings.TrimSpace(out.Query))
	fmt.Printf("Proyecto:  %s\n", proyecto)
	fmt.Printf("Coincidencias: %d\n", out.TotalMatches)
	if len(out.Results) == 0 {
		fmt.Println("No hay evidencias con ese filtro.")
		return nil
	}
	for _, result := range out.Results {
		if result == nil || result.Agente == nil {
			continue
		}
		fmt.Printf("\nAgente: %s", result.Agente.Nombre)
		if strings.TrimSpace(result.Agente.Rol) != "" {
			fmt.Printf(" [%s]", strings.TrimSpace(result.Agente.Rol))
		}
		fmt.Printf("  tareas_abiertas=%d", result.OpenTasks)
		if result.Runtime != nil && strings.TrimSpace(result.Runtime.Branch) != "" {
			fmt.Printf("  branch=%s", strings.TrimSpace(result.Runtime.Branch))
		}
		fmt.Println()
		for _, match := range result.Matches {
			if match == nil || match.Transcript == nil {
				continue
			}
			signal := strings.TrimSpace(match.Transcript.Classification)
			if signal == "" {
				signal = strings.TrimSpace(match.Transcript.Stream)
			}
			fmt.Printf("  - [%s] %s\n", match.Transcript.CreatedAt.Format("2006-01-02 15:04:05"), signal)
			fmt.Printf("    %s\n", truncar(strings.TrimSpace(match.Transcript.Text), 180))
			if strings.TrimSpace(match.TraceDir) != "" {
				fmt.Printf("    traza: %s\n", match.TraceDir)
			}
			if strings.TrimSpace(match.LogPath) != "" {
				fmt.Printf("    log:   %s\n", match.LogPath)
			}
		}
	}
	return nil
}

func apiHandlerAgenteInvestigar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodGet) {
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("q obligatoria"))
		return
	}
	proyecto := strings.TrimSpace(r.URL.Query().Get("proyecto"))
	limit := 25
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			apiError(w, http.StatusBadRequest, fmt.Errorf("limit inválido"))
			return
		}
		limit = value
	}
	out, err := agentesService.Investigate(query, proyecto, limit)
	if err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	apiWriteJSON(w, http.StatusOK, apiAgenteInvestigacionResponse{Investigacion: out})
}

func init() {
	agenteInvestigarCmd.Flags().StringP("proyecto", "p", "", "Proyecto sobre el que investigar")
	agenteInvestigarCmd.Flags().Int("limit", 25, "Número máximo de evidencias a recuperar")
	agenteCmd.AddCommand(agenteInvestigarCmd)
}
