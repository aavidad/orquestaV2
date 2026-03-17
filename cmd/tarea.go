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
	"orquesta/db"
)

var tareaCmd = &cobra.Command{
	Use:   "tarea",
	Short: "Gestión de tareas",
}

var tareaListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista tareas (por defecto: libres y en progreso)",
	RunE: func(cmd *cobra.Command, args []string) error {
		estadoStr, _ := cmd.Flags().GetString("estado")
		agente, _ := cmd.Flags().GetString("agente")
		modulo, _ := cmd.Flags().GetString("modulo")
		propuestaCodigo, _ := cmd.Flags().GetString("propuesta")

		f := db.FiltroTareas{Agente: &agente, Modulo: &modulo}
		if estadoStr != "" {
			e := db.EstadoTarea(estadoStr)
			f.Estado = &e
		}
		if agente == "" {
			f.Agente = nil
		}
		if modulo == "" {
			f.Modulo = nil
		}
		if propuestaCodigo != "" {
			p, err := db.GetPropuesta(propuestaCodigo)
			if err != nil {
				return fmt.Errorf("propuesta '%s' no encontrada", propuestaCodigo)
			}
			f.PropuestaID = &p.ID
		}

		tareas, err := db.ListarTareas(f)
		if err != nil {
			return err
		}
		if len(tareas) == 0 {
			fmt.Println("No hay tareas con ese filtro.")
			return nil
		}
		fmt.Printf("%-5s %-10s %-10s %-12s %-8s %s\n",
			"ID", "ESTADO", "PRIORIDAD", "AGENTE", "MÓDULO", "TÍTULO")
		fmt.Printf("%-5s %-10s %-10s %-12s %-8s %s\n",
			"─────", "──────────", "──────────", "────────────", "────────", "─────────────────────────────")
		for _, t := range tareas {
			agente := "—"
			if t.Agente != nil {
				agente = *t.Agente
			}
			fmt.Printf("%-5d %-10s %-10s %-12s %-8s %s\n",
				t.ID, t.Estado, t.Prioridad, agente, t.Modulo, t.Titulo)
		}
		return nil
	},
}

var tareaVerCmd = &cobra.Command{
	Use:   "ver <id>",
	Short: "Muestra detalle de una tarea",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		t, err := db.GetTarea(id)
		if err != nil {
			return err
		}
		agente := "—"
		if t.Agente != nil {
			agente = *t.Agente
		}
		fmt.Printf("Tarea #%d — %s\n", t.ID, t.Titulo)
		fmt.Printf("  Estado:      %s\n", t.Estado)
		fmt.Printf("  Prioridad:   %s\n", t.Prioridad)
		fmt.Printf("  Módulo:      %s\n", t.Modulo)
		fmt.Printf("  Agente:      %s\n", agente)
		fmt.Printf("  Creado por:  %s\n", t.CreadoPor)
		if t.Descripcion != "" {
			fmt.Printf("  Descripción: %s\n", t.Descripcion)
		}
		if t.Notas != "" {
			fmt.Printf("  Notas:       %s\n", t.Notas)
		}
		fmt.Printf("  Creada:      %s\n", t.CreatedAt.Format("2006-01-02 15:04"))
		return nil
	},
}

var tareaNuevaCmd = &cobra.Command{
	Use:   "nueva",
	Short: "Crea una nueva tarea",
	RunE: func(cmd *cobra.Command, args []string) error {
		titulo, _ := cmd.Flags().GetString("titulo")
		desc, _ := cmd.Flags().GetString("descripcion")
		modulo, _ := cmd.Flags().GetString("modulo")
		prioridad, _ := cmd.Flags().GetString("prioridad")
		creadorPor, _ := cmd.Flags().GetString("por")
		agente, _ := cmd.Flags().GetString("agente")
		propuestaCodigo, _ := cmd.Flags().GetString("propuesta")

		if titulo == "" {
			return fmt.Errorf("--titulo es obligatorio")
		}

		t := &db.Tarea{
			Titulo:      titulo,
			Descripcion: desc,
			Modulo:      modulo,
			Prioridad:   db.PrioridadTarea(prioridad),
			CreadoPor:   creadorPor,
		}

		// Vincular a propuesta si se indicó
		if propuestaCodigo != "" {
			p, err := db.GetPropuesta(propuestaCodigo)
			if err != nil {
				return fmt.Errorf("propuesta '%s' no encontrada", propuestaCodigo)
			}
			t.PropuestaID = &p.ID
		}
		id, err := db.CrearTarea(t)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Tarea #%d creada: %s\n", id, titulo)

		// Si se especificó agente, asignar directamente
		if agente != "" {
			if err := db.TomarTarea(id, agente); err != nil {
				fmt.Printf("  ⚠ No se pudo asignar a %s: %v\n", agente, err)
			} else {
				fmt.Printf("  → Asignada a %s\n", agente)
			}
		}
		return nil
	},
}

var tareaTomar = &cobra.Command{
	Use:   "tomar <id> <agente>",
	Short: "Asigna una tarea libre a un agente",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		if err := db.TomarTarea(id, args[1]); err != nil {
			return err
		}
		fmt.Printf("✓ Tarea #%d asignada a %s\n", id, args[1])
		return nil
	},
}

var tareaIniciarCmd = &cobra.Command{
	Use:   "iniciar <id> <agente>",
	Short: "Marca una tarea como 'en_progreso'",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		if err := db.IniciarTarea(id, args[1]); err != nil {
			return err
		}
		fmt.Printf("✓ Tarea #%d en progreso (%s)\n", id, args[1])
		return nil
	},
}

var tareaCompletarCmd = &cobra.Command{
	Use:   "completar <id> <agente>",
	Short: "Marca una tarea como completada",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		commit, _ := cmd.Flags().GetString("commit")
		if err := db.CompletarTarea(id, args[1], commit); err != nil {
			return err
		}
		fmt.Printf("✓ Tarea #%d completada\n", id)
		return nil
	},
}

var tareaBloquearCmd = &cobra.Command{
	Use:   "bloquear <id> <agente> [motivo]",
	Short: "Bloquea una tarea indicando el motivo",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		motivo, err := resolverTextoFlagOPosicional(cmd, args, 2, "motivo")
		if err != nil {
			return err
		}
		if err := db.BloquearTarea(id, args[1], motivo); err != nil {
			return err
		}
		fmt.Printf("✓ Tarea #%d bloqueada: %s\n", id, motivo)
		return nil
	},
}

var tareaNotaCmd = &cobra.Command{
	Use:   "nota <id> <agente> <nota...>",
	Short: "Añade una nota a una tarea",
	Args:  cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		nota := strings.Join(args[2:], " ")
		if err := db.AnotarTarea(id, args[1], nota); err != nil {
			return err
		}
		fmt.Printf("✓ Nota añadida a tarea #%d\n", id)
		return nil
	},
}

var tareaReasignarCmd = &cobra.Command{
	Use:   "reasignar <id> <nuevo-agente>",
	Short: "Reasigna una tarea a otro agente (solo Alberto)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		nuevoAgente := args[1]
		_, err = db.DB.Exec(
			`UPDATE tareas SET agente=?, estado='asignada' WHERE id=?`, nuevoAgente, id)
		if err != nil {
			return err
		}
		db.Audit("alberto", "reasignar_tarea", "tarea", id, nuevoAgente)
		fmt.Printf("✓ Tarea #%d reasignada a %s\n", id, nuevoAgente)
		return nil
	},
}

var tareaDesbloquearCmd = &cobra.Command{
	Use:   "desbloquear <id> <agente> [resolución...]",
	Short: "Desbloquea una tarea indicando cómo se resolvió",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		resolucion, err := resolverTextoFlagOPosicional(cmd, args, 2, "resolucion")
		if err != nil {
			return err
		}
		if err := db.DesbloquearTarea(id, args[1], resolucion); err != nil {
			return err
		}
		fmt.Printf("✓ Tarea #%d desbloqueada: %s\n", id, resolucion)
		return nil
	},
}

var tareaBacklogCmd = &cobra.Command{
	Use:   "backlog <id>",
	Short: "Mueve una tarea a backlog (pendiente para el futuro)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		_, err = db.DB.Exec(`UPDATE tareas SET estado='backlog', agente=NULL WHERE id=?`, id)
		if err != nil {
			return err
		}
		db.Audit("alberto", "backlog_tarea", "tarea", id, "")
		fmt.Printf("✓ Tarea #%d movida a backlog\n", id)
		return nil
	},
}

func init() {
	tareaListarCmd.Flags().String("estado", "", "Filtrar por estado (libre, asignada, en_progreso, completada, bloqueada, backlog)")
	tareaListarCmd.Flags().String("agente", "", "Filtrar por agente")
	tareaListarCmd.Flags().String("modulo", "", "Filtrar por módulo")
	tareaListarCmd.Flags().String("propuesta", "", "Filtrar por propuesta (ej: OP-030)")

	tareaNuevaCmd.Flags().String("titulo", "", "Título de la tarea (obligatorio)")
	tareaNuevaCmd.Flags().String("descripcion", "", "Descripción")
	tareaNuevaCmd.Flags().String("modulo", "", "Módulo al que pertenece")
	tareaNuevaCmd.Flags().String("prioridad", "media", "Prioridad: alta, media, baja")
	tareaNuevaCmd.Flags().String("por", "alberto", "Creado por")
	tareaNuevaCmd.Flags().String("agente", "", "Asignar directamente a este agente")
	tareaNuevaCmd.Flags().String("propuesta", "", "Código de propuesta vinculada (ej: OP-030)")

	tareaCompletarCmd.Flags().String("commit", "", "Hash o referencia del commit de cierre")
	tareaBloquearCmd.Flags().String("motivo", "", "Motivo del bloqueo")
	tareaDesbloquearCmd.Flags().String("resolucion", "", "Cómo se resolvió el bloqueo")

	tareaCmd.AddCommand(
		tareaListarCmd, tareaVerCmd, tareaNuevaCmd,
		tareaTomar, tareaIniciarCmd, tareaCompletarCmd,
		tareaBloquearCmd, tareaNotaCmd,
		tareaReasignarCmd, tareaDesbloquearCmd, tareaBacklogCmd,
	)
}
