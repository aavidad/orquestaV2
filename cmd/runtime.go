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
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
)

type runtimeRow struct {
	ID           int64
	Nivel        int
	Agente       string
	Proyecto     string
	Provider     string
	Connector    string
	Estado       string
	PID          string
	Hijos        int
	Branch       string
	Modelo       string
	Razonamiento string
	Ultima       string
}

var runtimeCmd = &cobra.Command{
	Use:   "runtime",
	Short: "Inspección read-only de runtimes y agentes hijos",
}

var runtimeListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista runtimes con estado pasivo y jerarquía",
	RunE: func(cmd *cobra.Command, args []string) error {
		query := url.Values{}
		agente, _ := cmd.Flags().GetString("agente")
		proyecto, _ := cmd.Flags().GetString("proyecto")
		activos, _ := cmd.Flags().GetString("activos")
		if strings.TrimSpace(agente) != "" {
			query.Set("agente", agente)
		}
		if strings.TrimSpace(proyecto) != "" {
			query.Set("proyecto", proyecto)
		}
		if strings.TrimSpace(activos) != "" {
			query.Set("activos", activos)
		}

		if tree, ok, err := cargarArbolRuntimesDesdeAPI(query); ok {
			if err != nil {
				return err
			}
			return imprimirArbolRuntimes(aplanarArbolAPI(tree), false)
		}

		filter := db.FiltroRuntimes{}
		if strings.TrimSpace(agente) != "" {
			filter.Agente = &agente
		}
		if strings.TrimSpace(proyecto) != "" {
			p, err := db.GetProyecto(proyecto)
			if err != nil {
				return err
			}
			filter.ProyectoID = &p.ID
		}
		if strings.TrimSpace(activos) != "" {
			switch activos {
			case "true":
				v := true
				filter.Activos = &v
			case "false":
				v := false
				filter.Activos = &v
			default:
				return fmt.Errorf("activos inválido")
			}
		}
		treeLocal, err := db.ConstruirArbolRuntimes(filter)
		if err != nil {
			return err
		}
		return imprimirArbolRuntimes(aplanarArbolLocal(treeLocal), false)
	},
}

var runtimeVerCmd = &cobra.Command{
	Use:   "ver <id>",
	Short: "Muestra el detalle de un runtime y sus muestras recientes",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}

		if detail, ok, err := cargarDetalleRuntimeDesdeAPI(id); ok {
			if err != nil {
				return err
			}
			return imprimirDetalleRuntime(detail.Runtime, detail.Samples)
		}

		runtime, err := db.GetRuntime(id)
		if err != nil {
			return err
		}
		samples, err := db.ListarMuestrasRuntime(id, 20)
		if err != nil {
			return err
		}
		return imprimirDetalleRuntime(runtime, samples)
	},
}

func cargarArbolRuntimesDesdeAPI(query url.Values) ([]*apiRuntimeTreeNode, bool, error) {
	var resp apiRuntimeTreeResponse
	ok, err := apiGetQuery("/api/runtimes/tree", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Runtimes, true, nil
}

func cargarDetalleRuntimeDesdeAPI(id int64) (*apiRuntimeDetailResponse, bool, error) {
	var resp apiRuntimeDetailResponse
	ok, err := apiGet(fmt.Sprintf("/api/runtimes/%d", id), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func aplanarArbolAPI(nodes []*apiRuntimeTreeNode) []runtimeRow {
	rows := make([]runtimeRow, 0, len(nodes))
	aplanarNodosRuntimeAPI(nodes, 0, &rows)
	return rows
}

func aplanarNodosRuntimeAPI(nodes []*apiRuntimeTreeNode, nivel int, out *[]runtimeRow) {
	for _, node := range nodes {
		if node == nil || node.Runtime == nil {
			continue
		}
		*out = append(*out, runtimeRowDesdeModelo(node.Runtime, nivel, len(node.Hijos)))
		aplanarNodosRuntimeAPI(node.Hijos, nivel+1, out)
	}
}

func aplanarArbolLocal(nodes []*db.RuntimeTreeNode) []runtimeRow {
	rows := make([]runtimeRow, 0, len(nodes))
	aplanarNodosRuntimeLocal(nodes, 0, &rows)
	return rows
}

func aplanarNodosRuntimeLocal(nodes []*db.RuntimeTreeNode, nivel int, out *[]runtimeRow) {
	for _, node := range nodes {
		if node == nil || node.Runtime == nil {
			continue
		}
		*out = append(*out, runtimeRowDesdeModelo(node.Runtime, nivel, len(node.Hijos)))
		aplanarNodosRuntimeLocal(node.Hijos, nivel+1, out)
	}
}

func runtimeRowDesdeModelo(r *db.RuntimeInstance, nivel, hijos int) runtimeRow {
	if r == nil {
		return runtimeRow{}
	}
	pid := "—"
	if r.PID != nil {
		pid = strconv.FormatInt(*r.PID, 10)
	}
	proyecto := r.ProyectoSlug
	if strings.TrimSpace(proyecto) == "" {
		proyecto = "—"
	}
	ultima := "—"
	if r.LastHeartbeatAt != nil {
		ultima = r.LastHeartbeatAt.Format("2006-01-02 15:04:05")
	} else if r.LastEventAt != nil {
		ultima = r.LastEventAt.Format("2006-01-02 15:04:05")
	}
	return runtimeRow{
		ID:           r.ID,
		Nivel:        nivel,
		Agente:       r.Agente,
		Proyecto:     proyecto,
		Provider:     valorVacio(r.Provider),
		Connector:    valorVacio(r.Connector),
		Estado:       valorVacio(r.LogicalState),
		PID:          pid,
		Hijos:        hijos,
		Branch:       valorVacio(r.Branch),
		Modelo:       valorVacio(r.Model),
		Razonamiento: valorVacio(r.Reasoning),
		Ultima:       ultima,
	}
}

func imprimirArbolRuntimes(rows []runtimeRow, jsonOut bool) error {
	if len(rows) == 0 {
		fmt.Println("No hay runtimes con ese filtro.")
		return nil
	}
	if jsonOut {
		for _, row := range rows {
			fmt.Printf("%+v\n", row)
		}
		return nil
	}
	fmt.Printf("%-5s %-11s %-14s %-15s %-12s %-14s %-8s %-5s %-18s %s\n",
		"ID", "ESTADO", "AGENTE", "PROYECTO", "PROVIDER", "CONNECTOR", "PID", "HIJOS", "BRANCH", "ULTIMA")
	for _, row := range rows {
		indent := strings.Repeat("  ", row.Nivel)
		agente := indent + row.Agente
		fmt.Printf("%-5d %-11s %-14s %-15s %-12s %-14s %-8s %-5d %-18s %s\n",
			row.ID, truncar(row.Estado, 11), truncar(agente, 14), truncar(row.Proyecto, 15),
			truncar(row.Provider, 12), truncar(row.Connector, 14), row.PID, row.Hijos,
			truncar(row.Branch, 18), row.Ultima)
	}
	return nil
}

func imprimirDetalleRuntime(r *db.RuntimeInstance, samples []*db.RuntimeTelemetrySample) error {
	if r == nil {
		return fmt.Errorf("runtime no encontrado")
	}
	fmt.Printf("Runtime #%d — %s\n", r.ID, r.Agente)
	fmt.Printf("  Proyecto:     %s\n", valorVacio(r.ProyectoSlug))
	fmt.Printf("  Estado:       %s\n", valorVacio(r.LogicalState))
	fmt.Printf("  Proceso:      %s\n", valorVacio(r.ProcessState))
	fmt.Printf("  Provider:     %s\n", valorVacio(r.Provider))
	fmt.Printf("  Connector:    %s\n", valorVacio(r.Connector))
	fmt.Printf("  PID:          %s\n", formatoIntPtr(r.PID))
	fmt.Printf("  PPID:         %s\n", formatoIntPtr(r.PPID))
	fmt.Printf("  Hijos:        %d\n", r.ChildCount)
	fmt.Printf("  Hilos:        %d\n", r.ThreadCount)
	fmt.Printf("  Modelo:       %s\n", valorVacio(r.Model))
	fmt.Printf("  Razonamiento: %s\n", valorVacio(r.Reasoning))
	fmt.Printf("  Perfil:       %s\n", valorVacio(r.TaskProfile))
	fmt.Printf("  CWD:          %s\n", valorVacio(r.CWD))
	fmt.Printf("  Branch:       %s\n", valorVacio(r.Branch))
	fmt.Printf("  Sesión:       %s\n", valorVacio(r.ExternalSessionID))
	fmt.Printf("  Última señal:  %s\n", formatoTiempoPtr(r.LastHeartbeatAt))
	fmt.Printf("  Último evento: %s\n", formatoTiempoPtr(r.LastEventAt))

	if len(samples) > 0 {
		fmt.Println("\nMuestras recientes:")
		fmt.Printf("%-18s %-9s %-9s %-9s %-8s %-8s %-10s %s\n", "CREADA", "CPU%", "MEM", "RSS", "HIJOS", "HILOS", "FUENTE", "ESTADO")
		for _, s := range samples {
			fmt.Printf("%-18s %-9.1f %-9d %-9d %-8d %-8d %-10s %s\n",
				s.CreatedAt.Format("2006-01-02 15:04:05"), s.CPUPct, s.MemBytes, s.RSSBytes,
				s.ChildCount, s.ThreadCount, truncar(s.Source, 10), truncar(s.LogicalState, 18))
		}
	}
	return nil
}

func formatoIntPtr(v *int64) string {
	if v == nil {
		return "—"
	}
	return strconv.FormatInt(*v, 10)
}

func formatoTiempoPtr(v *time.Time) string {
	if v == nil {
		return "—"
	}
	return v.Format("2006-01-02 15:04:05")
}

func init() {
	runtimeListarCmd.Flags().String("agente", "", "Filtrar por agente")
	runtimeListarCmd.Flags().String("proyecto", "", "Filtrar por proyecto")
	runtimeListarCmd.Flags().String("activos", "", "Filtrar por activos=true|false")
	runtimeCmd.AddCommand(runtimeListarCmd, runtimeVerCmd)
	rootCmd.AddCommand(runtimeCmd)
}
