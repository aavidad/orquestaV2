/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var agenteCmd = &cobra.Command{
	Use:   "agente",
	Short: "Control activo de agentes vivos",
}

var agenteHandleCmd = &cobra.Command{
	Use:   "handle",
	Short: "Gestión de handles vivos de runtime",
}

var agenteHandleRegistrarCmd = &cobra.Command{
	Use:   "registrar <agente>",
	Short: "Registra o reemplaza el handle vivo de un agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sesionID, err := int64FlagOpt(cmd, "sesion")
		if err != nil {
			return err
		}
		transporte, _ := cmd.Flags().GetString("transporte")
		handleKind, _ := cmd.Flags().GetString("kind")
		handleRef, _ := cmd.Flags().GetString("ref")
		estado, _ := cmd.Flags().GetString("estado")
		metadata, _ := cmd.Flags().GetString("metadata")

		id, err := db.RegistrarRuntimeHandle(&db.RuntimeHandle{
			Agente:       strings.TrimSpace(args[0]),
			SesionID:     sesionID,
			Transporte:   strings.TrimSpace(transporte),
			HandleKind:   strings.TrimSpace(handleKind),
			HandleRef:    strings.TrimSpace(handleRef),
			Estado:       strings.TrimSpace(estado),
			MetadataJSON: strings.TrimSpace(metadata),
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Runtime handle registrado para %s (id: %d)\n", args[0], id)
		return nil
	},
}

var agenteHandleListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista los handles vivos registrados",
	RunE: func(cmd *cobra.Command, args []string) error {
		handles, err := db.ListarRuntimeHandles()
		if err != nil {
			return err
		}
		if len(handles) == 0 {
			fmt.Println("No hay runtime handles registrados.")
			return nil
		}
		fmt.Printf("%-5s %-12s %-10s %-12s %-20s %s\n", "ID", "AGENTE", "ESTADO", "TRANSPORTE", "KIND", "REF")
		for _, h := range handles {
			fmt.Printf("%-5d %-12s %-10s %-12s %-20s %s\n", h.ID, h.Agente, h.Estado, h.Transporte, h.HandleKind, h.HandleRef)
		}
		return nil
	},
}

var agenteOrdenCmd = &cobra.Command{
	Use:   "orden",
	Short: "Gestión de órdenes hacia agentes vivos",
}

var agenteOrdenListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista órdenes de runtime",
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, _ := cmd.Flags().GetString("agente")
		estado, _ := cmd.Flags().GetString("estado")
		orders, err := db.ListarRuntimeOrders(strings.TrimSpace(agente), strings.TrimSpace(estado))
		if err != nil {
			return err
		}
		if len(orders) == 0 {
			fmt.Println("No hay runtime orders con ese filtro.")
			return nil
		}
		fmt.Printf("%-5s %-12s %-14s %-12s %s\n", "ID", "AGENTE", "TIPO", "ESTADO", "CREADA")
		for _, o := range orders {
			fmt.Printf("%-5d %-12s %-14s %-12s %s\n", o.ID, o.Agente, o.Tipo, o.Estado, o.CreatedAt.Format("2006-01-02 15:04"))
		}
		return nil
	},
}

var agenteOrdenCrearCmd = &cobra.Command{
	Use:   "crear <agente>",
	Short: "Crea una orden de runtime para un agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sesionID, err := int64FlagOpt(cmd, "sesion")
		if err != nil {
			return err
		}
		tipo, _ := cmd.Flags().GetString("tipo")
		payload, _ := cmd.Flags().GetString("payload")
		estado, _ := cmd.Flags().GetString("estado")

		agente := strings.TrimSpace(args[0])
		tipo = strings.TrimSpace(tipo)
		payload = strings.TrimSpace(payload)
		estado = strings.TrimSpace(estado)
		switch tipo {
		case "enviar_instruccion", "pausar", "continuar", "handoff":
		default:
			return fmt.Errorf("tipo inválido: %s", tipo)
		}
		if payload == "" {
			payload = "{}"
		}
		if !json.Valid([]byte(payload)) {
			return fmt.Errorf("payload JSON inválido")
		}

		id, err := db.CrearRuntimeOrder(&db.RuntimeOrder{
			Agente:      agente,
			SesionID:    sesionID,
			Tipo:        tipo,
			PayloadJSON: payload,
			Estado:      estado,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Runtime order creada para %s (id: %d)\n", agente, id)
		return nil
	},
}

var agenteOrdenActualizarCmd = &cobra.Command{
	Use:   "actualizar <id> <estado>",
	Short: "Actualiza el estado de una runtime order",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		if err := db.ActualizarRuntimeOrderEstado(id, strings.TrimSpace(args[1])); err != nil {
			return err
		}
		fmt.Printf("✓ Runtime order #%d actualizada a %s\n", id, args[1])
		return nil
	},
}

var agenteHandoffCmd = &cobra.Command{
	Use:   "handoff <agente-origen> <agente-destino>",
	Short: "Crea un handoff real y reasigna la tarea al agente destino",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tareaID, err := int64FlagOpt(cmd, "tarea")
		if err != nil {
			return err
		}
		resumen, _ := cmd.Flags().GetString("resumen")
		externalSessionID, _ := cmd.Flags().GetString("external-session-id")
		motivo, _ := cmd.Flags().GetString("motivo")

		orderID, err := db.CrearHandoffAgenteVivo(
			strings.TrimSpace(args[0]),
			strings.TrimSpace(args[1]),
			tareaID,
			strings.TrimSpace(motivo),
			strings.TrimSpace(resumen),
			strings.TrimSpace(externalSessionID),
		)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Handoff creado %s → %s (runtime_order: %d)\n", args[0], args[1], orderID)
		if tareaID != nil {
			fmt.Printf("  Tarea #%d reasignada al agente destino\n", *tareaID)
		}
		return nil
	},
}

var agenteReasignarVivoCmd = &cobra.Command{
	Use:   "reasignar-vivo <tarea-id> <agente-origen> <agente-destino>",
	Short: "Atajo de handoff con reasignación explícita de tarea viva",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		tareaID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("tarea-id inválido")
		}
		resumen, _ := cmd.Flags().GetString("resumen")
		externalSessionID, _ := cmd.Flags().GetString("external-session-id")
		motivo, _ := cmd.Flags().GetString("motivo")
		orderID, err := db.CrearHandoffAgenteVivo(
			strings.TrimSpace(args[1]),
			strings.TrimSpace(args[2]),
			&tareaID,
			strings.TrimSpace(motivo),
			strings.TrimSpace(resumen),
			strings.TrimSpace(externalSessionID),
		)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Reasignación viva creada sobre tarea #%d (runtime_order: %d)\n", tareaID, orderID)
		return nil
	},
}

var agenteVerCmd = &cobra.Command{
	Use:   "ver <agente>",
	Short: "Muestra el handle activo, sesión y presupuesto del agente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente := strings.TrimSpace(args[0])
		handle, handleErr := db.RuntimeHandleActivo(agente)
		sesion, sesionErr := db.SesionActivaDeAgente(agente)
		p, _, presupuestoErr := db.UltimoPresupuestoAgente(agente)

		fmt.Printf("Agente: %s\n", agente)
		if sesionErr == nil && sesion != nil {
			fmt.Printf("  Sesión activa: #%d\n", sesion.ID)
		} else {
			fmt.Printf("  Sesión activa: —\n")
		}
		if handleErr == nil && handle != nil {
			fmt.Printf("  Handle activo: %s [%s/%s]\n", handle.HandleRef, handle.Transporte, handle.HandleKind)
		} else {
			fmt.Printf("  Handle activo: —\n")
		}
		if presupuestoErr == nil && p != nil {
			ev, err := db.EvaluarPresupuestoSesion(p)
			if err != nil {
				return err
			}
			fmt.Printf("  Presupuesto:   %s", ev.Estado)
			if ev.Motivo != "" {
				fmt.Printf(" (%s)", ev.Motivo)
			}
			fmt.Println()
		} else {
			fmt.Printf("  Presupuesto:   —\n")
		}

		if handleErr != nil && handleErr != sql.ErrNoRows {
			return handleErr
		}
		if sesionErr != nil && sesionErr != sql.ErrNoRows {
			return sesionErr
		}
		return nil
	},
}

func init() {
	agenteHandleRegistrarCmd.Flags().Int64("sesion", 0, "Sesión activa vinculada al handle")
	agenteHandleRegistrarCmd.Flags().String("transporte", "", "Transporte: pty, process, mcp_session, remote_api")
	agenteHandleRegistrarCmd.Flags().String("kind", "", "Tipo de handle: pty, process, mcp_session, remote_api")
	agenteHandleRegistrarCmd.Flags().String("ref", "", "Referencia concreta del handle")
	agenteHandleRegistrarCmd.Flags().String("estado", "activo", "Estado del handle")
	agenteHandleRegistrarCmd.Flags().String("metadata", "{}", "Metadata JSON")

	agenteOrdenListarCmd.Flags().String("agente", "", "Filtrar por agente")
	agenteOrdenListarCmd.Flags().String("estado", "", "Filtrar por estado")
	agenteOrdenCrearCmd.Flags().Int64("sesion", 0, "Sesión activa vinculada a la orden")
	agenteOrdenCrearCmd.Flags().String("tipo", "", "Tipo de orden")
	agenteOrdenCrearCmd.Flags().String("payload", "{}", "Payload JSON de la orden")
	agenteOrdenCrearCmd.Flags().String("estado", "pendiente", "Estado inicial de la orden")
	_ = agenteOrdenCrearCmd.MarkFlagRequired("tipo")

	agenteHandoffCmd.Flags().Int64("tarea", 0, "Tarea a reasignar al relevo")
	agenteHandoffCmd.Flags().String("motivo", "", "Motivo del handoff")
	agenteHandoffCmd.Flags().String("resumen", "", "Resumen de continuidad para el relevo")
	agenteHandoffCmd.Flags().String("external-session-id", "", "External session id conocido")

	agenteReasignarVivoCmd.Flags().String("motivo", "", "Motivo del relevo")
	agenteReasignarVivoCmd.Flags().String("resumen", "", "Resumen de continuidad")
	agenteReasignarVivoCmd.Flags().String("external-session-id", "", "External session id conocido")

	agenteHandleCmd.AddCommand(agenteHandleRegistrarCmd, agenteHandleListarCmd)
	agenteOrdenCmd.AddCommand(agenteOrdenListarCmd, agenteOrdenCrearCmd, agenteOrdenActualizarCmd)
	agenteCmd.AddCommand(agenteVerCmd, agenteHandleCmd, agenteOrdenCmd, agenteHandoffCmd, agenteReasignarVivoCmd)
	rootCmd.AddCommand(agenteCmd)
}
