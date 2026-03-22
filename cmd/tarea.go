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

	"github.com/spf13/cobra"
	"orquesta/db"
	"orquesta/taskapp"
)

var tareaCmd = &cobra.Command{
	Use:   "tarea",
	Short: "Gestión de tareas",
}

var taskService = taskapp.NewService(taskapp.Repository{})

var tareaListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista tareas (por defecto: libres y en progreso)",
	RunE: func(cmd *cobra.Command, args []string) error {
		estadoStr, _ := cmd.Flags().GetString("estado")
		agente, _ := cmd.Flags().GetString("agente")
		modulo, _ := cmd.Flags().GetString("modulo")
		propuestaCodigo, _ := cmd.Flags().GetString("propuesta")

		var (
			tareas []*db.Tarea
			err    error
		)
		if serverURL := activeServerURL(); serverURL != "" {
			query := url.Values{}
			if estadoStr != "" {
				query.Set("estado", estadoStr)
			}
			if agente != "" {
				query.Set("agente", agente)
			}
			if modulo != "" {
				query.Set("modulo", modulo)
			}
			if propuestaCodigo != "" {
				query.Set("propuesta", propuestaCodigo)
			}
			tareas, err = fetchServerTasks(serverURL, query)
		} else {
			filtro := db.FiltroTareas{}
			if estadoStr != "" {
				e := db.EstadoTarea(estadoStr)
				filtro.Estado = &e
			}
			if agente != "" {
				filtro.Agente = &agente
			}
			if modulo != "" {
				filtro.Modulo = &modulo
			}
			if propuestaCodigo != "" {
				propuestaID, err := taskService.ResolveProposalID(propuestaCodigo)
				if err != nil {
					return fmt.Errorf("propuesta '%s' no encontrada", propuestaCodigo)
				}
				filtro.PropuestaID = propuestaID
			}
			tareas, err = taskService.List(filtro)
		}
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
		var t *db.Tarea
		if serverURL := activeServerURL(); serverURL != "" {
			t, err = fetchServerTaskDetail(serverURL, id)
		} else {
			t, err = taskService.Get(id)
		}
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

		if serverURL := activeServerURL(); serverURL != "" {
			id, err := submitServerCreateTask(serverURL, map[string]any{
				"titulo":      titulo,
				"descripcion": desc,
				"modulo":      modulo,
				"prioridad":   prioridad,
				"creado_por":  creadorPor,
				"agente":      agente,
				"propuesta":   propuestaCodigo,
			})
			if err != nil {
				return err
			}
			fmt.Printf("✓ Tarea #%d creada: %s\n", id, titulo)
			if agente != "" {
				fmt.Printf("  → Asignada a %s\n", agente)
			}
			return nil
		}
		id, err := taskService.Create(taskapp.CreateTaskInput{
			Titulo:          titulo,
			Descripcion:     desc,
			Modulo:          modulo,
			Prioridad:       db.PrioridadTarea(prioridad),
			CreadoPor:       creadorPor,
			Agente:          agente,
			PropuestaCodigo: propuestaCodigo,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Tarea #%d creada: %s\n", id, titulo)
		if agente != "" {
			fmt.Printf("  → Asignada a %s\n", agente)
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
		if serverURL := activeServerURL(); serverURL != "" {
			if err := submitServerTaskAction(serverURL, id, "tomar", map[string]any{"agente": args[1]}); err != nil {
				return err
			}
		} else if err := taskService.Take(id, args[1]); err != nil {
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
		if serverURL := activeServerURL(); serverURL != "" {
			if err := submitServerTaskAction(serverURL, id, "iniciar", map[string]any{"agente": args[1]}); err != nil {
				return err
			}
		} else if err := taskService.Start(id, args[1]); err != nil {
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
		if serverURL := activeServerURL(); serverURL != "" {
			if err := submitServerTaskAction(serverURL, id, "completar", map[string]any{"agente": args[1], "commit": commit}); err != nil {
				return err
			}
		} else if err := taskService.Complete(id, args[1], commit); err != nil {
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
		if serverURL := activeServerURL(); serverURL != "" {
			if err := submitServerTaskAction(serverURL, id, "bloquear", map[string]any{"agente": args[1], "motivo": motivo}); err != nil {
				return err
			}
		} else if err := taskService.Block(id, args[1], motivo); err != nil {
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
		if serverURL := activeServerURL(); serverURL != "" {
			if err := submitServerTaskAction(serverURL, id, "nota", map[string]any{"agente": args[1], "nota": nota}); err != nil {
				return err
			}
		} else if err := taskService.Note(id, args[1], nota); err != nil {
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
		if err := taskService.Reassign(id, nuevoAgente); err != nil {
			return err
		}
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
		if serverURL := activeServerURL(); serverURL != "" {
			if err := submitServerTaskAction(serverURL, id, "desbloquear", map[string]any{"agente": args[1], "resolucion": resolucion}); err != nil {
				return err
			}
		} else if err := taskService.Unblock(id, args[1], resolucion); err != nil {
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
		if err := taskService.MoveToBacklog(id); err != nil {
			return err
		}
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
