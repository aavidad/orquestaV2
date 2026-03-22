/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"orquesta/db"
	"orquesta/runtimectl"
)

var agenteCmd = &cobra.Command{
	Use:   "agente",
	Short: "Control activo y runtime de agentes vivos",
}

var runtimeService = runtimectl.NewService(db.RuntimeRepository{})

var agenteHandleCmd = &cobra.Command{
	Use:   "handle",
	Short: "Gestion de handles runtime de agentes",
}

var agenteHandleRegistrarCmd = &cobra.Command{
	Use:   "registrar <agente>",
	Short: "Registra el handle runtime activo de un agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente := args[0]
		transporte, _ := cmd.Flags().GetString("transporte")
		handleKind, _ := cmd.Flags().GetString("handle-kind")
		handleRef, _ := cmd.Flags().GetString("handle-ref")
		estado, _ := cmd.Flags().GetString("estado")
		metadataJSON, _ := cmd.Flags().GetString("metadata-json")
		sesionID, _ := cmd.Flags().GetInt64("sesion-id")
		proyectoSlug, _ := cmd.Flags().GetString("proyecto")

		var sesionIDPtr *int64
		if cmd.Flags().Changed("sesion-id") {
			sesionIDPtr = &sesionID
		}
		id, err := runtimeService.RegisterHandle(runtimectl.RegisterHandleInput{
			Agente:       agente,
			SesionID:     sesionIDPtr,
			ProyectoSlug: proyectoSlug,
			Transporte:   transporte,
			HandleKind:   handleKind,
			HandleRef:    handleRef,
			Estado:       estado,
			MetadataJSON: metadataJSON,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Runtime handle registrado para %s (id: %d)\n", agente, id)
		return nil
	},
}

var agenteHandleListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista runtime handles registrados",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		estado, _ := cmd.Flags().GetString("estado")
		items, err := runtimeService.ListHandles(agente, estado)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			fmt.Println("No hay runtime handles.")
			return nil
		}
		fmt.Printf("%-12s %-10s %-12s %-14s %s\n", "AGENTE", "ESTADO", "TRANSPORTE", "HANDLE_KIND", "HANDLE_REF")
		fmt.Printf("%-12s %-10s %-12s %-14s %s\n", "────────────", "──────────", "────────────", "──────────────", "────────────────────────")
		for _, item := range items {
			fmt.Printf("%-12s %-10s %-12s %-14s %s\n", item.Agente, item.Estado, item.Transporte, item.HandleKind, item.HandleRef)
		}
		return nil
	},
}

var agenteOrdenCmd = &cobra.Command{
	Use:   "orden",
	Short: "Cola de ordenes runtime",
}

var agenteOrdenListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista runtime orders registradas",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		estado, _ := cmd.Flags().GetString("estado")
		items, err := runtimeService.ListOrders(agente, estado)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			fmt.Println("No hay runtime orders.")
			return nil
		}
		fmt.Printf("%-4s %-12s %-20s %-12s %s\n", "ID", "AGENTE", "TIPO", "ESTADO", "ERROR")
		fmt.Printf("%-4s %-12s %-20s %-12s %s\n", "────", "────────────", "────────────────────", "────────────", "────────────────────────")
		for _, item := range items {
			fmt.Printf("%-4d %-12s %-20s %-12s %s\n", item.ID, item.Agente, item.Tipo, item.Estado, item.ErrorText)
		}
		return nil
	},
}

var agenteOrdenEjecutarCmd = &cobra.Command{
	Use:   "ejecutar <agente>",
	Short: "Ejecuta la siguiente orden pendiente de un agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		item, err := runtimeService.ExecuteNext(args[0])
		if err != nil {
			return err
		}
		if item == nil {
			fmt.Printf("No hay ordenes pendientes para %s.\n", args[0])
			return nil
		}
		fmt.Printf("✓ Runtime order #%d ejecutada (%s)\n", item.ID, item.Tipo)
		return nil
	},
}

var agenteEnviarCmd = &cobra.Command{
	Use:   "enviar <agente>",
	Short: "Encola una instruccion para un agente vivo",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mensaje, err := resolverTextoFlagOPosicional(cmd, args, 1, "mensaje")
		if err != nil {
			return err
		}
		proyectoSlug, _ := cmd.Flags().GetString("proyecto")
		id, err := runtimeService.EnqueueInstruction(args[0], proyectoSlug, mensaje)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Orden #%d encolada para %s\n", id, args[0])
		return nil
	},
}

var agentePausarCmd = &cobra.Command{
	Use:   "pausar <agente>",
	Short: "Encola una orden de pausa para un agente vivo",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proyectoSlug, _ := cmd.Flags().GetString("proyecto")
		id, err := runtimeService.EnqueuePause(args[0], proyectoSlug)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Orden #%d de pausa encolada para %s\n", id, args[0])
		return nil
	},
}

var agenteContinuarCmd = &cobra.Command{
	Use:   "continuar <agente>",
	Short: "Encola una orden de continuacion para un agente pausado",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proyectoSlug, _ := cmd.Flags().GetString("proyecto")
		id, err := runtimeService.EnqueueContinue(args[0], proyectoSlug)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Orden #%d de continuacion encolada para %s\n", id, args[0])
		return nil
	},
}

var agenteHandoffCmd = &cobra.Command{
	Use:   "handoff <agente-origen> <agente-destino>",
	Short: "Encola un handoff entre agentes",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		proyectoSlug, _ := cmd.Flags().GetString("proyecto")
		id, err := runtimeService.EnqueueHandoff(args[0], args[1], proyectoSlug)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Orden #%d de handoff encolada: %s -> %s\n", id, args[0], args[1])
		return nil
	},
}

func init() {
	agenteHandleRegistrarCmd.Flags().String("transporte", "local", "Transporte del handle runtime")
	agenteHandleRegistrarCmd.Flags().String("handle-kind", "pty", "Tipo de handle: pty|process|mcp_session|remote_api")
	agenteHandleRegistrarCmd.Flags().String("handle-ref", "", "Referencia del handle runtime")
	agenteHandleRegistrarCmd.Flags().String("estado", "activo", "Estado del handle")
	agenteHandleRegistrarCmd.Flags().String("metadata-json", "{}", "Metadata JSON del handle")
	agenteHandleRegistrarCmd.Flags().Int64("sesion-id", 0, "Sesion asociada")
	agenteHandleRegistrarCmd.Flags().String("proyecto", "", "Slug del proyecto asociado")
	_ = agenteHandleRegistrarCmd.MarkFlagRequired("handle-ref")

	agenteHandleListarCmd.Flags().String("agente", "", "Filtrar por agente")
	agenteHandleListarCmd.Flags().String("estado", "", "Filtrar por estado")

	agenteOrdenListarCmd.Flags().String("agente", "", "Filtrar por agente")
	agenteOrdenListarCmd.Flags().String("estado", "", "Filtrar por estado")

	agenteEnviarCmd.Flags().String("mensaje", "", "Mensaje a enviar")
	agenteEnviarCmd.Flags().String("proyecto", "", "Slug del proyecto")
	agentePausarCmd.Flags().String("proyecto", "", "Slug del proyecto")
	agenteContinuarCmd.Flags().String("proyecto", "", "Slug del proyecto")
	agenteHandoffCmd.Flags().String("proyecto", "", "Slug del proyecto")

	agenteHandleCmd.AddCommand(agenteHandleRegistrarCmd, agenteHandleListarCmd)
	agenteOrdenCmd.AddCommand(agenteOrdenListarCmd, agenteOrdenEjecutarCmd)
	agenteCmd.AddCommand(agenteHandleCmd, agenteOrdenCmd, agenteEnviarCmd, agentePausarCmd, agenteContinuarCmd, agenteHandoffCmd)
	rootCmd.AddCommand(agenteCmd)
}
