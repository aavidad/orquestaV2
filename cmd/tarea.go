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
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		modulo, _ := cmd.Flags().GetString("modulo")
		propuestaCodigo, _ := cmd.Flags().GetString("propuesta")
		jsonOut, _ := cmd.Flags().GetBool("json")
		tsvOut, _ := cmd.Flags().GetBool("tsv")
		if jsonOut && tsvOut {
			return fmt.Errorf("usa solo uno de --json o --tsv")
		}

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
			return err
		} else if ok {
			tareas = tareasResp.Tareas
			projectSlugs, _, _ = apiProjectSlugMap()
		} else {
			return serverFirstCommandError("tarea listar")
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
				proyecto := projectLabel(projectSlugs, t.ProyectoID, "")
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
			proyecto := projectLabel(projectSlugs, t.ProyectoID, "—")
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
		if _, err := strconv.ParseInt(args[0], 10, 64); err != nil {
			return fmt.Errorf("id inválido")
		}
		var t *db.Tarea
		projectSlugs := map[int64]string{}
		var tareaResp apiTareaResponse
		if ok, err := apiGet("/api/tareas/"+args[0], &tareaResp); err != nil {
			return err
		} else if ok {
			t = tareaResp.Tarea
			projectSlugs, _, _ = apiProjectSlugMap()
		} else {
			return serverFirstCommandError("tarea ver")
		}
		agente := "—"
		if t.Agente != nil {
			agente = *t.Agente
		}
		fmt.Printf("Tarea #%d — %s\n", t.ID, t.Titulo)
		fmt.Printf("  Estado:      %s\n", t.Estado)
		fmt.Printf("  Prioridad:   %s\n", t.Prioridad)
		if t.ProyectoID != nil {
			fmt.Printf("  Proyecto:    %s\n", projectLabel(projectSlugs, t.ProyectoID, strconv.FormatInt(*t.ProyectoID, 10)))
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
		return serverFirstCommandError("tarea nueva")
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
		if ok, err := apiPost("/api/tareas/"+args[0]+"/accion", apiTareaAccionRequest{
			Accion: "tomar",
			Agente: args[1],
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Tarea #%d asignada a %s\n", id, args[1])
			return nil
		}
		return serverFirstCommandError("tarea tomar")
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
		if ok, err := apiPost("/api/tareas/"+args[0]+"/accion", apiTareaAccionRequest{
			Accion: "iniciar",
			Agente: args[1],
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Tarea #%d en progreso (%s)\n", id, args[1])
			return nil
		}
		return serverFirstCommandError("tarea iniciar")
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
		return serverFirstCommandError("tarea completar")
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
		return serverFirstCommandError("tarea bloquear")
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
		return serverFirstCommandError("tarea nota")
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
		if ok, err := apiPost("/api/tareas/"+args[0]+"/accion", apiTareaAccionRequest{
			Accion:      "reasignar",
			NuevoAgente: nuevoAgente,
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Tarea #%d reasignada a %s\n", id, nuevoAgente)
			return nil
		}
		return serverFirstCommandError("tarea reasignar")
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
		return serverFirstCommandError("tarea desbloquear")
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
		if ok, err := apiPost("/api/tareas/"+args[0]+"/accion", apiTareaAccionRequest{
			Accion: "backlog",
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Tarea #%d movida a backlog\n", id)
			return nil
		}
		return serverFirstCommandError("tarea backlog")
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

		var solicitud *db.RefineriaSolicitud
		var resp struct {
			OK        bool                   `json:"ok"`
			Solicitud *db.RefineriaSolicitud `json:"solicitud"`
		}
		if ok, err := apiPost("/api/tareas/"+args[0]+"/refineria", apiRefineriaRequest{
			Agente:  agente,
			Rama:    rama,
			Dir:     dir,
			CmdTest: cmdTest,
		}, &resp); err != nil {
			return err
		} else if ok {
			solicitud = resp.Solicitud
		}
		if solicitud == nil {
			return serverFirstCommandError("tarea refineria")
		}
		fmt.Printf("✓ Tarea #%d enviada a Refinería (solicitud #%d)\n", id, solicitud.ID)
		fmt.Printf("  rama=%s  dir=%s  cmd=%s\n", solicitud.Rama, solicitud.DirTrabajo, solicitud.CmdTest)
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
			return serverFirstCommandError("tarea cancelar")
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

func projectLabel(projectSlugs map[int64]string, projectID *int64, empty string) string {
	if projectID == nil {
		return empty
	}
	if slug := strings.TrimSpace(projectSlugs[*projectID]); slug != "" {
		return slug
	}
	return strconv.FormatInt(*projectID, 10)
}

// tareaNotasCmd muestra las notas de una tarea (solo lectura, sin ver todo el detalle).
var tareaNotasCmd = &cobra.Command{
	Use:   "notas <id>",
	Short: "Muestra las notas registradas en una tarea",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := strconv.ParseInt(args[0], 10, 64); err != nil {
			return fmt.Errorf("id inválido: %s", args[0])
		}
		var t *db.Tarea
		var resp apiTareaResponse
		if ok, err := apiGet("/api/tareas/"+args[0], &resp); err != nil {
			return err
		} else if ok {
			t = resp.Tarea
		} else {
			return serverFirstCommandError("tarea notas")
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
		return serverFirstCommandError("tarea contrato")
	},
}
