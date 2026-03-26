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

type runtimeDiagnosticoData struct {
	Agente           string
	Proyecto         string
	Fuente           string
	Runtimes         []runtimeRow
	Handles          []*db.RuntimeHandle
	Orders           []*db.RuntimeOrder
	MailboxPendiente []*db.RuntimeMailboxMessage
	Checkpoints      []*db.RuntimeCheckpoint
}

var runtimeDiagnosticoCmd = &cobra.Command{
	Use:   "diagnostico",
	Short: "Resume el estado operativo de un agente en el runtime",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		proyecto, _ := cmd.Flags().GetString("proyecto")
		limit, _ := cmd.Flags().GetInt("limit")
		data, err := cargarRuntimeDiagnostico(strings.TrimSpace(agente), strings.TrimSpace(proyecto), limit)
		if err != nil {
			return err
		}
		renderRuntimeDiagnostico(data, limit)
		return nil
	},
}

func cargarRuntimeDiagnostico(agente, proyecto string, limit int) (*runtimeDiagnosticoData, error) {
	if strings.TrimSpace(agente) == "" {
		return nil, fmt.Errorf("--agente es obligatorio")
	}
	if limit <= 0 {
		limit = 5
	}
	if data, ok, err := cargarRuntimeDiagnosticoDesdeAPI(agente, proyecto, limit); ok {
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	return nil, serverFirstCommandError("runtime diagnostico")
}

func cargarRuntimeDiagnosticoDesdeAPI(agente, proyecto string, limit int) (*runtimeDiagnosticoData, bool, error) {
	query := url.Values{"agente": []string{agente}}
	if proyecto != "" {
		query.Set("proyecto", proyecto)
	}
	tree, ok, err := cargarArbolRuntimesDesdeAPI(query)
	if !ok || err != nil {
		return nil, ok, err
	}

	handles, ok, err := cargarRuntimeHandlesDesdeAPI(url.Values{"agente": []string{agente}})
	if !ok || err != nil {
		return nil, ok, err
	}
	orders, ok, err := cargarRuntimeOrdersDesdeAPI(query)
	if !ok || err != nil {
		return nil, ok, err
	}
	mailQuery := url.Values{
		"to_agente": []string{agente},
		"estado":    []string{"pendiente"},
	}
	if proyecto != "" {
		mailQuery.Set("proyecto", proyecto)
	}
	mailbox, ok, err := cargarRuntimeMailboxDesdeAPI(mailQuery)
	if !ok || err != nil {
		return nil, ok, err
	}
	checkpoints, ok, err := cargarRuntimeCheckpointsDesdeAPI(agente, proyecto, "", "", limit)
	if !ok || err != nil {
		return nil, ok, err
	}

	return &runtimeDiagnosticoData{
		Agente:           agente,
		Proyecto:         proyecto,
		Fuente:           "api",
		Runtimes:         aplanarArbolAPI(tree),
		Handles:          handles,
		Orders:           orders,
		MailboxPendiente: mailbox,
		Checkpoints:      checkpoints,
	}, true, nil
}

func renderRuntimeDiagnostico(data *runtimeDiagnosticoData, limit int) {
	if data == nil {
		return
	}
	if limit <= 0 {
		limit = 5
	}
	fmt.Printf("╔═══════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║              ORQUESTA — DIAGNÓSTICO RUNTIME             ║\n")
	fmt.Printf("╚═══════════════════════════════════════════════════════════╝\n\n")
	fmt.Printf("Agente:     %s\n", data.Agente)
	fmt.Printf("Proyecto:   %s\n", valorVacio(data.Proyecto))
	fmt.Printf("Fuente:     %s\n", data.Fuente)
	fmt.Printf("Runtimes:   %d\n", len(data.Runtimes))
	fmt.Printf("Handles:    %d\n", len(data.Handles))
	fmt.Printf("Órdenes:    %d total (%s)\n", len(data.Orders), resumirEstadosRuntimeOrders(data.Orders))
	fmt.Printf("Mailbox:    %d pendiente(s)\n", len(data.MailboxPendiente))
	fmt.Printf("Checkpoints:%d recientes\n", len(data.Checkpoints))

	if len(data.Runtimes) > 0 {
		fmt.Println("\nRuntimes")
		_ = imprimirArbolRuntimes(data.Runtimes, false)
	}
	if len(data.Handles) > 0 {
		fmt.Println("\nHandles")
		_ = imprimirRuntimeHandles(data.Handles)
	}
	orders := runtimeOrdersRelevantes(data.Orders, limit)
	if len(orders) > 0 {
		fmt.Println("\nÓrdenes relevantes")
		_ = imprimirRuntimeOrders(orders)
	}
	if len(data.MailboxPendiente) > 0 {
		fmt.Println("\nMailbox pendiente")
		_ = imprimirRuntimeMailbox(data.MailboxPendiente)
	}
	if len(data.Checkpoints) > 0 {
		fmt.Println("\nCheckpoints recientes")
		_ = imprimirRuntimeDiagnosticoCheckpoints(data.Checkpoints)
	}
}

func resumirEstadosRuntimeOrders(orders []*db.RuntimeOrder) string {
	if len(orders) == 0 {
		return "sin órdenes"
	}
	counts := map[string]int{}
	for _, order := range orders {
		if order == nil {
			continue
		}
		counts[strings.TrimSpace(order.Estado)]++
	}
	prioridad := []string{"pendiente", "tomada", "ejecutando", "fallida", "completada", "cancelada", "expirada"}
	partes := make([]string, 0, len(prioridad))
	for _, estado := range prioridad {
		if counts[estado] > 0 {
			partes = append(partes, fmt.Sprintf("%s=%d", estado, counts[estado]))
			delete(counts, estado)
		}
	}
	for estado, n := range counts {
		if strings.TrimSpace(estado) == "" {
			continue
		}
		partes = append(partes, fmt.Sprintf("%s=%d", estado, n))
	}
	if len(partes) == 0 {
		return "sin órdenes"
	}
	return strings.Join(partes, ", ")
}

func runtimeOrdersRelevantes(orders []*db.RuntimeOrder, limit int) []*db.RuntimeOrder {
	if limit <= 0 {
		limit = 5
	}
	relevantes := make([]*db.RuntimeOrder, 0, limit)
	for _, order := range orders {
		if order == nil {
			continue
		}
		switch strings.TrimSpace(order.Estado) {
		case "pendiente", "tomada", "ejecutando", "fallida":
			relevantes = append(relevantes, order)
		}
		if len(relevantes) >= limit {
			return relevantes
		}
	}
	if len(relevantes) > 0 {
		return relevantes
	}
	if len(orders) <= limit {
		return orders
	}
	return orders[:limit]
}

func imprimirRuntimeDiagnosticoCheckpoints(checkpoints []*db.RuntimeCheckpoint) error {
	if len(checkpoints) == 0 {
		fmt.Println("No hay checkpoints para ese filtro.")
		return nil
	}
	fmt.Printf("%-5s %-12s %-16s %-18s %-16s %-18s %s\n", "ID", "AGENTE", "KIND", "SOURCE", "BRANCH", "CREADO", "RESUMEN")
	for _, cp := range checkpoints {
		if cp == nil {
			continue
		}
		fmt.Printf("%-5d %-12s %-16s %-18s %-16s %-18s %s\n",
			cp.ID, truncar(cp.Agente, 12), truncar(cp.CheckpointKind, 16), truncar(cp.Source, 18),
			truncar(cp.Branch, 16), cp.CreatedAt.Format("2006-01-02 15:04:05"), truncar(cp.Resumen, 48))
	}
	return nil
}

func init() {
	runtimeDiagnosticoCmd.Flags().String("agente", "", "Agente a diagnosticar")
	runtimeDiagnosticoCmd.Flags().String("proyecto", "", "Proyecto asociado al runtime")
	runtimeDiagnosticoCmd.Flags().Int("limit", 5, "Numero maximo de órdenes y checkpoints a mostrar")
	_ = runtimeDiagnosticoCmd.MarkFlagRequired("agente")
	runtimeCmd.AddCommand(runtimeDiagnosticoCmd)
}
