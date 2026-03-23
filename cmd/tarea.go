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
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		modulo, _ := cmd.Flags().GetString("modulo")
		propuestaCodigo, _ := cmd.Flags().GetString("propuesta")
		jsonOut, _ := cmd.Flags().GetBool("json")
		tsvOut, _ := cmd.Flags().GetBool("tsv")
		if jsonOut && tsvOut {
			return fmt.Errorf("usa solo uno de --json o --tsv")
		}

<<<<<<< HEAD
		var tareas []*db.Tarea
		projectSlugs := map[int64]string{}
		params := url.Values{}
		if estadoStr != "" {
			params.Set("estado", estadoStr)
		}
		if agente != "" {
			params.Set("agente", agente)
		}
		if proyectoRef != "" {
			params.Set("proyecto", proyectoRef)
		}
		if modulo != "" {
			params.Set("modulo", modulo)
		}
		if propuestaCodigo != "" {
			params.Set("propuesta", propuestaCodigo)
		}
		var tareasResp apiTareasResponse
		if ok, err := apiGetQuery("/api/tareas", params, &tareasResp); err != nil {
=======
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
>>>>>>> origin/orq-orquestador-codex2
			return err
		} else if ok {
			tareas = tareasResp.Tareas
			projectSlugs, _, _ = apiProjectSlugMap()
		} else {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			f := db.FiltroTareas{Agente: &agente, Modulo: &modulo}
			if estadoStr != "" {
				e := db.EstadoTarea(estadoStr)
				f.Estado = &e
			}
			if agente == "" {
				f.Agente = nil
			}
			if proyectoRef != "" {
				p, err := db.GetProyecto(proyectoRef)
				if err != nil {
					return err
				}
				f.ProyectoID = &p.ID
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
			var err error
			tareas, err = db.ListarTareas(f)
			if err != nil {
				return err
			}
		}
		if len(tareas) == 0 {
			if jsonOut {
				return imprimirJSON([]*db.Tarea{})
			}
			fmt.Println("No hay tareas con ese filtro.")
			return nil
		}
		if jsonOut {
			return imprimirJSON(tareas)
		}
		if tsvOut {
			for _, t := range tareas {
				agente := ""
				if t.Agente != nil {
					agente = *t.Agente
				}
				proyecto := ""
				if t.ProyectoID != nil {
					if slug := projectSlugs[*t.ProyectoID]; slug != "" {
						proyecto = slug
					} else if p, err := db.GetProyecto(strconv.FormatInt(*t.ProyectoID, 10)); err == nil {
						proyecto = p.Slug
					}
				}
				titulo := strings.NewReplacer("\t", " ", "\n", " ", "\r", " ").Replace(t.Titulo)
				fmt.Printf("%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
					t.ID, t.Estado, t.Prioridad, agente, proyecto, t.Modulo, titulo)
			}
			return nil
		}
		fmt.Printf("%-5s %-10s %-10s %-12s %-12s %-8s %s\n",
			"ID", "ESTADO", "PRIORIDAD", "AGENTE", "PROYECTO", "MÓDULO", "TÍTULO")
		fmt.Printf("%-5s %-10s %-10s %-12s %-12s %-8s %s\n",
			"─────", "──────────", "──────────", "────────────", "────────────", "────────", "─────────────────────────────")
		for _, t := range tareas {
			agente := "—"
			if t.Agente != nil {
				agente = *t.Agente
			}
			proyecto := "—"
			if t.ProyectoID != nil {
				if slug := projectSlugs[*t.ProyectoID]; slug != "" {
					proyecto = slug
				} else if p, err := db.GetProyecto(strconv.FormatInt(*t.ProyectoID, 10)); err == nil {
					proyecto = p.Slug
				}
			}
			fmt.Printf("%-5d %-10s %-10s %-12s %-12s %-8s %s\n",
				t.ID, t.Estado, t.Prioridad, agente, proyecto, t.Modulo, t.Titulo)
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
<<<<<<< HEAD
		projectSlugs := map[int64]string{}
		var tareaResp apiTareaResponse
		if ok, err := apiGet("/api/tareas/"+args[0], &tareaResp); err != nil {
=======
		if serverURL := activeServerURL(); serverURL != "" {
			t, err = fetchServerTaskDetail(serverURL, id)
		} else {
			t, err = taskService.Get(id)
		}
		if err != nil {
>>>>>>> origin/orq-orquestador-codex2
			return err
		} else if ok {
			t = tareaResp.Tarea
			projectSlugs, _, _ = apiProjectSlugMap()
		} else {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			t, err = db.GetTarea(id)
			if err != nil {
				return err
			}
		}
		agente := "—"
		if t.Agente != nil {
			agente = *t.Agente
		}
		fmt.Printf("Tarea #%d — %s\n", t.ID, t.Titulo)
		fmt.Printf("  Estado:      %s\n", t.Estado)
		fmt.Printf("  Prioridad:   %s\n", t.Prioridad)
		if t.ProyectoID != nil {
			if slug := projectSlugs[*t.ProyectoID]; slug != "" {
				fmt.Printf("  Proyecto:    %s\n", slug)
			} else if p, err := db.GetProyecto(strconv.FormatInt(*t.ProyectoID, 10)); err == nil {
				fmt.Printf("  Proyecto:    %s\n", p.Slug)
			}
		}
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
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		modulo, _ := cmd.Flags().GetString("modulo")
		prioridad, _ := cmd.Flags().GetString("prioridad")
		creadorPor, _ := cmd.Flags().GetString("por")
		agente, _ := cmd.Flags().GetString("agente")
		propuestaCodigo, _ := cmd.Flags().GetString("propuesta")

		if titulo == "" {
			return fmt.Errorf("--titulo es obligatorio")
		}

<<<<<<< HEAD
		var resp struct {
			OK    bool      `json:"ok"`
			Tarea *db.Tarea `json:"tarea"`
		}
		if ok, err := apiPost("/api/tareas", apiTareaCrearRequest{
			Titulo:      titulo,
			Descripcion: desc,
			Proyecto:    proyectoRef,
			Modulo:      modulo,
			Prioridad:   prioridad,
			CreadoPor:   creadorPor,
			Propuesta:   propuestaCodigo,
			Agente:      agente,
		}, &resp); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Tarea #%d creada: %s\n", resp.Tarea.ID, titulo)
			if agente != "" {
				fmt.Printf("  → Asignada a %s\n", agente)
			}
			return nil
		}

		if err := ensureLocalDB(); err != nil {
			return err
		}
		t := &db.Tarea{
			Titulo:      titulo,
			Descripcion: desc,
			Modulo:      modulo,
			Prioridad:   db.PrioridadTarea(prioridad),
			CreadoPor:   creadorPor,
		}
		if proyectoRef != "" {
			p, err := db.GetProyecto(proyectoRef)
			if err != nil {
				return err
			}
			t.ProyectoID = &p.ID
		}

		// Vincular a propuesta si se indicó
		if propuestaCodigo != "" {
			p, err := db.GetPropuesta(propuestaCodigo)
=======
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
>>>>>>> origin/orq-orquestador-codex2
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
<<<<<<< HEAD
		if ok, err := apiPost("/api/tareas/"+args[0]+"/accion", apiTareaAccionRequest{
			Accion: "tomar",
			Agente: args[1],
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Tarea #%d asignada a %s\n", id, args[1])
			return nil
		}
		if err := ensureLocalDB(); err != nil {
			return err
		}
		if err := db.TomarTarea(id, args[1]); err != nil {
=======
		if serverURL := activeServerURL(); serverURL != "" {
			if err := submitServerTaskAction(serverURL, id, "tomar", map[string]any{"agente": args[1]}); err != nil {
				return err
			}
		} else if err := taskService.Take(id, args[1]); err != nil {
>>>>>>> origin/orq-orquestador-codex2
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
<<<<<<< HEAD
		if ok, err := apiPost("/api/tareas/"+args[0]+"/accion", apiTareaAccionRequest{
			Accion: "iniciar",
			Agente: args[1],
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Tarea #%d en progreso (%s)\n", id, args[1])
			return nil
		}
		if err := ensureLocalDB(); err != nil {
			return err
		}
		if err := db.IniciarTarea(id, args[1]); err != nil {
=======
		if serverURL := activeServerURL(); serverURL != "" {
			if err := submitServerTaskAction(serverURL, id, "iniciar", map[string]any{"agente": args[1]}); err != nil {
				return err
			}
		} else if err := taskService.Start(id, args[1]); err != nil {
>>>>>>> origin/orq-orquestador-codex2
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
<<<<<<< HEAD
		if ok, err := apiPost("/api/tareas/"+args[0]+"/accion", apiTareaAccionRequest{
			Accion: "completar",
			Agente: args[1],
			Commit: commit,
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Tarea #%d completada\n", id)
			return nil
		}
		if err := ensureLocalDB(); err != nil {
			return err
		}
		if err := db.CompletarTarea(id, args[1], commit); err != nil {
=======
		if serverURL := activeServerURL(); serverURL != "" {
			if err := submitServerTaskAction(serverURL, id, "completar", map[string]any{"agente": args[1], "commit": commit}); err != nil {
				return err
			}
		} else if err := taskService.Complete(id, args[1], commit); err != nil {
>>>>>>> origin/orq-orquestador-codex2
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
<<<<<<< HEAD
		if ok, err := apiPost("/api/tareas/"+args[0]+"/accion", apiTareaAccionRequest{
			Accion: "bloquear",
			Agente: args[1],
			Motivo: motivo,
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Tarea #%d bloqueada: %s\n", id, motivo)
			return nil
		}
		if err := ensureLocalDB(); err != nil {
			return err
		}
		if err := db.BloquearTarea(id, args[1], motivo); err != nil {
=======
		if serverURL := activeServerURL(); serverURL != "" {
			if err := submitServerTaskAction(serverURL, id, "bloquear", map[string]any{"agente": args[1], "motivo": motivo}); err != nil {
				return err
			}
		} else if err := taskService.Block(id, args[1], motivo); err != nil {
>>>>>>> origin/orq-orquestador-codex2
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
<<<<<<< HEAD
		if ok, err := apiPost("/api/tareas/"+args[0]+"/accion", apiTareaAccionRequest{
			Accion: "nota",
			Agente: args[1],
			Nota:   nota,
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Nota añadida a tarea #%d\n", id)
			return nil
		}
		if err := ensureLocalDB(); err != nil {
			return err
		}
		if err := db.AnotarTarea(id, args[1], nota); err != nil {
=======
		if serverURL := activeServerURL(); serverURL != "" {
			if err := submitServerTaskAction(serverURL, id, "nota", map[string]any{"agente": args[1], "nota": nota}); err != nil {
				return err
			}
		} else if err := taskService.Note(id, args[1], nota); err != nil {
>>>>>>> origin/orq-orquestador-codex2
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
<<<<<<< HEAD
		if ok, err := apiPost("/api/tareas/"+args[0]+"/accion", apiTareaAccionRequest{
			Accion:      "reasignar",
			NuevoAgente: nuevoAgente,
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Tarea #%d reasignada a %s\n", id, nuevoAgente)
			return nil
		}
		if err := ensureLocalDB(); err != nil {
			return err
		}
		_, err = db.DB.Exec(
			`UPDATE tareas SET agente=?, estado='asignada' WHERE id=?`, nuevoAgente, id)
		if err != nil {
=======
		if err := taskService.Reassign(id, nuevoAgente); err != nil {
>>>>>>> origin/orq-orquestador-codex2
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
<<<<<<< HEAD
		if ok, err := apiPost("/api/tareas/"+args[0]+"/accion", apiTareaAccionRequest{
			Accion:     "desbloquear",
			Agente:     args[1],
			Resolucion: resolucion,
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Tarea #%d desbloqueada: %s\n", id, resolucion)
			return nil
		}
		if err := ensureLocalDB(); err != nil {
			return err
		}
		if err := db.DesbloquearTarea(id, args[1], resolucion); err != nil {
=======
		if serverURL := activeServerURL(); serverURL != "" {
			if err := submitServerTaskAction(serverURL, id, "desbloquear", map[string]any{"agente": args[1], "resolucion": resolucion}); err != nil {
				return err
			}
		} else if err := taskService.Unblock(id, args[1], resolucion); err != nil {
>>>>>>> origin/orq-orquestador-codex2
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
<<<<<<< HEAD
		if ok, err := apiPost("/api/tareas/"+args[0]+"/accion", apiTareaAccionRequest{
			Accion: "backlog",
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Tarea #%d movida a backlog\n", id)
			return nil
		}
		if err := ensureLocalDB(); err != nil {
			return err
		}
		_, err = db.DB.Exec(`UPDATE tareas SET estado='backlog', agente=NULL WHERE id=?`, id)
		if err != nil {
=======
		if err := taskService.MoveToBacklog(id); err != nil {
>>>>>>> origin/orq-orquestador-codex2
			return err
		}
		fmt.Printf("✓ Tarea #%d movida a backlog\n", id)
		return nil
	},
}

var tareaRefineriaCmd = &cobra.Command{
	Use:   "refineria <id> <agente>",
	Short: "Envía una tarea a la Refinería para validación de tests antes de completarla (OP-093)",
	Long: `Encola la tarea en el Canal de Refinería.

El control plane ejecutará el comando de tests en el directorio indicado.
Si los tests pasan la tarea se completa automáticamente; si fallan, recibirás
un mensaje en tu buzón con el output de error para que corrijas y reintentes.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		agente := args[1]
		rama, _ := cmd.Flags().GetString("rama")
		dir, _ := cmd.Flags().GetString("dir")
		cmdTest, _ := cmd.Flags().GetString("cmd")

		if err := ensureLocalDB(); err != nil {
			return err
		}
		s, err := db.SolicitarRefineria(id, agente, rama, dir, cmdTest)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Tarea #%d enviada a Refinería (solicitud #%d)\n", id, s.ID)
		fmt.Printf("  rama=%s  dir=%s  cmd=%s\n", s.Rama, s.DirTrabajo, s.CmdTest)
		fmt.Println("  El control plane procesará la solicitud en el próximo ciclo (≤15s).")
		fmt.Println("  Consulta tu buzón con: orquesta runtime mailbox --agente " + agente)
		return nil
	},
}

var tareaCancelarCmd = &cobra.Command{
	Use:   "cancelar <id> <agente> [motivo]",
	Short: "Marca una tarea como cancelada",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		motivo := "duplicado o error"
		if len(args) > 2 {
			motivo = strings.Join(args[2:], " ")
		}
		if ok, err := apiPost("/api/tareas/"+args[0]+"/accion", apiTareaAccionRequest{
			Accion: "cancelar",
			Agente: args[1],
			Motivo: motivo,
		}, &map[string]any{}); err != nil {
			return err
		} else if !ok {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			if err := db.CancelarTarea(id, args[1], motivo); err != nil {
				return err
			}
		}
		fmt.Printf("✓ Tarea #%d cancelada: %s\n", id, motivo)
		return nil
	},
}

func init() {
	tareaListarCmd.Flags().Bool("json", false, "Salida JSON")
	tareaListarCmd.Flags().Bool("tsv", false, "Salida TSV pensada para scripts")
	tareaListarCmd.Flags().String("estado", "", "Filtrar por estado (libre, asignada, en_progreso, completada, bloqueada, backlog)")
	tareaListarCmd.Flags().String("agente", "", "Filtrar por agente")
	tareaListarCmd.Flags().String("proyecto", "", "Filtrar por proyecto")
	tareaListarCmd.Flags().String("modulo", "", "Filtrar por módulo")
	tareaListarCmd.Flags().String("propuesta", "", "Filtrar por propuesta (ej: OP-030)")

	tareaNuevaCmd.Flags().String("titulo", "", "Título de la tarea (obligatorio)")
	tareaNuevaCmd.Flags().String("descripcion", "", "Descripción")
	tareaNuevaCmd.Flags().String("proyecto", "", "Proyecto al que pertenece")
	tareaNuevaCmd.Flags().String("modulo", "", "Módulo al que pertenece")
	tareaNuevaCmd.Flags().String("prioridad", "media", "Prioridad: alta, media, baja")
	tareaNuevaCmd.Flags().String("por", "alberto", "Creado por")
	tareaNuevaCmd.Flags().String("agente", "", "Asignar directamente a este agente")
	tareaNuevaCmd.Flags().String("propuesta", "", "Código de propuesta vinculada (ej: OP-030)")

	tareaCompletarCmd.Flags().String("commit", "", "Hash o referencia del commit de cierre")
	tareaBloquearCmd.Flags().String("motivo", "", "Motivo del bloqueo")
	tareaDesbloquearCmd.Flags().String("resolucion", "", "Cómo se resolvió el bloqueo")
	tareaRefineriaCmd.Flags().String("rama", "", "Rama o worktree con los cambios")
	tareaRefineriaCmd.Flags().String("dir", "", "Directorio de trabajo donde ejecutar los tests")
	tareaRefineriaCmd.Flags().String("cmd", "go test ./...", "Comando de tests a ejecutar")

	tareaCmd.AddCommand(
		tareaListarCmd, tareaVerCmd, tareaNuevaCmd,
		tareaTomar, tareaIniciarCmd, tareaCompletarCmd,
		tareaBloquearCmd, tareaNotaCmd, tareaNotasCmd,
		tareaReasignarCmd, tareaDesbloquearCmd, tareaBacklogCmd,
		tareaContratoCmd, tareaCancelarCmd, tareaRefineriaCmd,
	)
}

// tareaNotasCmd muestra las notas de una tarea (solo lectura, sin ver todo el detalle).
var tareaNotasCmd = &cobra.Command{
	Use:   "notas <id>",
	Short: "Muestra las notas registradas en una tarea",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido: %s", args[0])
		}
		var t *db.Tarea
		var resp apiTareaResponse
		if ok, err := apiGet("/api/tareas/"+args[0], &resp); err != nil {
			return err
		} else if ok {
			t = resp.Tarea
		} else {
			t, err = db.GetTarea(id)
			if err != nil {
				return fmt.Errorf("tarea #%d no encontrada", id)
			}
		}
		fmt.Printf("Notas de la tarea #%d — %s\n", t.ID, t.Titulo)
		fmt.Println("─────────────────────────────────────────")
		if strings.TrimSpace(t.Notas) == "" {
			fmt.Println("(sin notas)")
		} else {
			// Cada nota está separada por saltos de línea; las mostramos numeradas.
			lineas := strings.Split(strings.TrimSpace(t.Notas), "\n")
			for i, l := range lineas {
				if strings.TrimSpace(l) != "" {
					fmt.Printf("  %d. %s\n", i+1, l)
				}
			}
		}
		return nil
	},
}

// tareaContratoCmd marca una tarea como con contrato/interfaz de E/S definido (OP-069).
var tareaContratoCmd = &cobra.Command{
	Use:   "contrato <id> <agente>",
	Short: "Marca una tarea como con contrato/interfaz de E/S definido (OP-069)",
	Long: `Registra que la tarea tiene sus entradas y salidas documentadas,
permitiendo que las tareas que dependen de ella puedan ser tomadas.

OP-069: ninguna tarea dependiente puede tomarse o iniciarse hasta que
su predecesora tenga el contrato definido o esté completada.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido: %s", args[0])
		}
		if ok, err := apiPost("/api/tareas/"+args[0]+"/accion", apiTareaAccionRequest{
			Accion: "contrato",
			Agente: args[1],
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Contrato definido para tarea #%d\n", id)
			return nil
		}
		if err := ensureLocalDB(); err != nil {
			return err
		}
		if err := db.DefinirContrato(id, args[1]); err != nil {
			return err
		}
		fmt.Printf("✓ Contrato definido para tarea #%d\n", id)
		return nil
	},
}
