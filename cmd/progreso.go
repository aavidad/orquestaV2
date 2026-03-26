/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var progresoCmd = &cobra.Command{
	Use:   "progreso",
	Short: "Fases y progreso real por proyecto y tarea",
}

var progresoVerCmd = &cobra.Command{
	Use:   "ver <proyecto>",
	Short: "Muestra el resumen de progreso real de un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto := strings.TrimSpace(args[0])
		resumen, ok, err := cargarResumenProgresoDesdeAPI(proyecto)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("progreso ver")
		}
		fmt.Printf("Proyecto: %s\n", resumen.Proyecto)
		fmt.Printf("Progreso: %.1f%%\n", resumen.ProgresoPct)
		fmt.Printf("Tareas:   %d total, %d completadas\n", resumen.TareasTotales, resumen.TareasCompletadas)
		if len(resumen.Fases) > 0 {
			fmt.Println("Fases:")
			for _, fase := range resumen.Fases {
				fmt.Printf("  [%d] %s  %.1f%%  (%d/%d tareas)\n",
					fase.Fase.Orden, fase.Fase.Nombre, fase.ProgresoPct, fase.TareasCompletadas, fase.TareasTotales)
			}
		}
		if len(resumen.TareasSinFase) > 0 {
			fmt.Println("Tareas sin fase:")
			for _, tarea := range resumen.TareasSinFase {
				fmt.Printf("  #%d %.1f%% %s\n", tarea.Tarea.ID, tarea.ProgresoPct, tarea.Tarea.Titulo)
			}
		}
		return nil
	},
}

var progresoFaseCmd = &cobra.Command{
	Use:   "fase",
	Short: "Gestión de fases de proyecto",
}

var progresoFaseListarCmd = &cobra.Command{
	Use:   "listar <proyecto>",
	Short: "Lista fases de un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto := strings.TrimSpace(args[0])
		fases, ok, err := cargarFasesProgresoDesdeAPI(proyecto)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("progreso fase listar")
		}
		if len(fases) == 0 {
			fmt.Println("No hay fases para ese proyecto.")
			return nil
		}
		fmt.Printf("%-5s %-6s %-8s %-10s %s\n", "ID", "ORDEN", "PESO", "ESTADO", "NOMBRE")
		for _, fase := range fases {
			fmt.Printf("%-5d %-6d %-8.1f %-10s %s\n", fase.ID, fase.Orden, fase.Peso, fase.Estado, fase.Nombre)
		}
		return nil
	},
}

var progresoFaseRegistrarCmd = &cobra.Command{
	Use:   "registrar <proyecto>",
	Short: "Registra una fase para un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto := strings.TrimSpace(args[0])
		nombre, _ := cmd.Flags().GetString("nombre")
		descripcion, _ := cmd.Flags().GetString("descripcion")
		orden, _ := cmd.Flags().GetInt64("orden")
		peso, _ := cmd.Flags().GetFloat64("peso")
		estado, _ := cmd.Flags().GetString("estado")
		req := apiProgresoFaseRegistrarRequest{
			Proyecto:    proyecto,
			Nombre:      strings.TrimSpace(nombre),
			Descripcion: strings.TrimSpace(descripcion),
			Orden:       orden,
			Peso:        peso,
			Estado:      strings.TrimSpace(estado),
		}
		if resp, ok, err := registrarFaseProgresoPorAPI(proyecto, req); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Fase #%d registrada\n", resp.ID)
			return nil
		}
		return serverFirstCommandError("progreso fase registrar")
	},
}

var progresoFaseActualizarCmd = &cobra.Command{
	Use:   "actualizar <id>",
	Short: "Actualiza una fase existente",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		req := apiProgresoFaseActualizarRequest{}
		if cmd.Flags().Changed("proyecto") {
			v, _ := cmd.Flags().GetString("proyecto")
			v = strings.TrimSpace(v)
			req.Proyecto = &v
		}
		if cmd.Flags().Changed("nombre") {
			v, _ := cmd.Flags().GetString("nombre")
			v = strings.TrimSpace(v)
			req.Nombre = &v
		}
		if cmd.Flags().Changed("descripcion") {
			v, _ := cmd.Flags().GetString("descripcion")
			v = strings.TrimSpace(v)
			req.Descripcion = &v
		}
		if cmd.Flags().Changed("orden") {
			v, _ := cmd.Flags().GetInt64("orden")
			req.Orden = &v
		}
		if cmd.Flags().Changed("peso") {
			v, _ := cmd.Flags().GetFloat64("peso")
			req.Peso = &v
		}
		if cmd.Flags().Changed("estado") {
			v, _ := cmd.Flags().GetString("estado")
			v = strings.TrimSpace(v)
			req.Estado = &v
		}
		if resp, ok, err := actualizarFaseProgresoPorAPI(id, req); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Fase #%d actualizada\n", resp.ID)
			return nil
		}
		return serverFirstCommandError("progreso fase actualizar")
	},
}

var progresoTareaCmd = &cobra.Command{
	Use:   "tarea",
	Short: "Gestión del avance real por tarea",
}

var progresoTareaRegistrarCmd = &cobra.Command{
	Use:   "registrar <tarea-id>",
	Short: "Registra o actualiza el avance real de una tarea",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tareaID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("tarea-id inválido")
		}
		proyecto, _ := cmd.Flags().GetString("proyecto")
		faseID, err := int64FlagOpt(cmd, "fase")
		if err != nil {
			return err
		}
		pct, _ := cmd.Flags().GetFloat64("pct")
		por, err := resolverAgenteMemoria(cmd)
		if err != nil {
			return err
		}
		req := apiProgresoTareaRegistrarRequest{
			Proyecto:       strings.TrimSpace(proyecto),
			FaseID:         faseID,
			ProgresoPct:    pct,
			ActualizadoPor: por,
		}
		if ok, err := registrarAvanceProgresoPorAPI(tareaID, req); err != nil {
			return err
		} else if !ok {
			return serverFirstCommandError("progreso tarea registrar")
		}
		fmt.Printf("✓ Avance registrado para tarea #%d\n", tareaID)
		return nil
	},
}

func init() {
	progresoFaseRegistrarCmd.Flags().String("nombre", "", "Nombre de la fase")
	progresoFaseRegistrarCmd.Flags().String("descripcion", "", "Descripción")
	progresoFaseRegistrarCmd.Flags().Int64("orden", 100, "Orden")
	progresoFaseRegistrarCmd.Flags().Float64("peso", 1.0, "Peso relativo")
	progresoFaseRegistrarCmd.Flags().String("estado", "pendiente", "Estado: pendiente, activa, bloqueada, completada")
	progresoFaseActualizarCmd.Flags().String("proyecto", "", "Proyecto")
	progresoFaseActualizarCmd.Flags().String("nombre", "", "Nombre")
	progresoFaseActualizarCmd.Flags().String("descripcion", "", "Descripción")
	progresoFaseActualizarCmd.Flags().Int64("orden", 0, "Orden")
	progresoFaseActualizarCmd.Flags().Float64("peso", 0, "Peso")
	progresoFaseActualizarCmd.Flags().String("estado", "", "Estado")
	progresoTareaRegistrarCmd.Flags().String("proyecto", "", "Proyecto")
	progresoTareaRegistrarCmd.Flags().Int64("fase", 0, "ID de fase")
	progresoTareaRegistrarCmd.Flags().Float64("pct", 0, "Porcentaje real 0..100")
	progresoTareaRegistrarCmd.Flags().String("por", "", "Actualizado por")
	progresoTareaRegistrarCmd.Flags().String("agente", "", "Alias de --por")

	progresoFaseCmd.AddCommand(progresoFaseListarCmd, progresoFaseRegistrarCmd, progresoFaseActualizarCmd)
	progresoTareaCmd.AddCommand(progresoTareaRegistrarCmd)
	progresoCmd.AddCommand(progresoVerCmd, progresoFaseCmd, progresoTareaCmd)
	rootCmd.AddCommand(progresoCmd)
}
