package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var proyectoSharedContextCmd = &cobra.Command{
	Use:   "contexto-compartido",
	Short: "Gestiona contexto compartido selectivo por proyecto",
}

var proyectoSharedContextListCmd = &cobra.Command{
	Use:   "listar <proyecto>",
	Short: "Lista el contexto compartido activo de un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		tipo, _ := cmd.Flags().GetString("tipo")
		limit, _ := cmd.Flags().GetInt("limit")

		resp, ok, err := listarContextoCompartidoProyectoPorAPI(args[0], strings.TrimSpace(agente), strings.TrimSpace(tipo), limit)
		if err != nil {
			return err
		}
		if !ok || resp == nil {
			return serverFirstCommandError("proyecto contexto-compartido listar")
		}
		if strings.TrimSpace(resp.Summary) != "" {
			fmt.Println(resp.Summary)
		}
		if len(resp.Items) == 0 {
			fmt.Println("Sin contexto compartido activo.")
			return nil
		}
		fmt.Printf("%-5s %-12s %-12s %-5s %s\n", "ID", "TIPO", "AGENTE", "PESO", "TITULO")
		for _, item := range resp.Items {
			if item == nil {
				continue
			}
			ag := strings.TrimSpace(item.Agente)
			if ag == "" {
				ag = "—"
			}
			fmt.Printf("%-5d %-12s %-12s %-5.1f %s\n",
				item.ID,
				truncar(item.Tipo, 12),
				truncar(ag, 12),
				item.Peso,
				item.Titulo,
			)
		}
		return nil
	},
}

var proyectoSharedContextCreateCmd = &cobra.Command{
	Use:   "anotar <proyecto>",
	Short: "Anota una pieza de contexto compartido durable para un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		titulo, _ := cmd.Flags().GetString("titulo")
		tipo, _ := cmd.Flags().GetString("tipo")
		agente, _ := cmd.Flags().GetString("agente")
		detalle, _ := cmd.Flags().GetString("detalle")
		payloadJSON, _ := cmd.Flags().GetString("payload-json")
		peso, _ := cmd.Flags().GetFloat64("peso")
		origen, _ := cmd.Flags().GetString("origen")
		expiresAt, _ := cmd.Flags().GetString("expires-at")

		resp, ok, err := anotarContextoCompartidoProyectoPorAPI(args[0], apiProyectoSharedContextCreateRequest{
			Agente:      strings.TrimSpace(agente),
			Tipo:        strings.TrimSpace(tipo),
			Titulo:      strings.TrimSpace(titulo),
			Detalle:     strings.TrimSpace(detalle),
			PayloadJSON: strings.TrimSpace(payloadJSON),
			Peso:        peso,
			Origen:      strings.TrimSpace(origen),
			ExpiresAt:   strings.TrimSpace(expiresAt),
		})
		if err != nil {
			return err
		}
		if !ok || resp == nil {
			return serverFirstCommandError("proyecto contexto-compartido anotar")
		}
		fmt.Printf("✓ Contexto compartido anotado con id %d\n", resp.ID)
		return nil
	},
}

func init() {
	proyectoSharedContextListCmd.Flags().String("agente", "", "Filtra contexto general y específico del agente")
	proyectoSharedContextListCmd.Flags().String("tipo", "", "Filtra por tipo")
	proyectoSharedContextListCmd.Flags().Int("limit", 20, "Máximo de items a devolver")

	proyectoSharedContextCreateCmd.Flags().String("titulo", "", "Titulo corto del contexto")
	proyectoSharedContextCreateCmd.Flags().String("tipo", "decision", "Tipo de contexto: decision, restriccion, hallazgo, followup...")
	proyectoSharedContextCreateCmd.Flags().String("agente", "", "Agente destinatario opcional; vacío = general del proyecto")
	proyectoSharedContextCreateCmd.Flags().String("detalle", "", "Detalle humano breve")
	proyectoSharedContextCreateCmd.Flags().String("payload-json", "", "Payload JSON opcional")
	proyectoSharedContextCreateCmd.Flags().Float64("peso", 1, "Peso relativo para priorizar inyección")
	proyectoSharedContextCreateCmd.Flags().String("origen", "humano", "Origen del contexto")
	proyectoSharedContextCreateCmd.Flags().String("expires-at", "", "Caducidad RFC3339 opcional")
	_ = proyectoSharedContextCreateCmd.MarkFlagRequired("titulo")

	proyectoSharedContextCmd.AddCommand(proyectoSharedContextListCmd, proyectoSharedContextCreateCmd)
	proyectoCmd.AddCommand(proyectoSharedContextCmd)
}
