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
	"orquesta/coordinacion"
	"orquesta/db"
)

var worktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Gestión de worktrees aislados por proyecto/tarea",
}

var worktreeListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista worktrees registrados",
	RunE: func(cmd *cobra.Command, args []string) error {
		query := url.Values{}
		if agente, _ := cmd.Flags().GetString("agente"); strings.TrimSpace(agente) != "" {
			query.Set("agente", agente)
		}
		if proyectoRef, _ := cmd.Flags().GetString("proyecto"); strings.TrimSpace(proyectoRef) != "" {
			query.Set("proyecto", proyectoRef)
		}
		if estadoStr, _ := cmd.Flags().GetString("estado"); strings.TrimSpace(estadoStr) != "" {
			query.Set("estado", estadoStr)
		}

		worktrees, ok, err := cargarWorktreesDesdeAPI(query)
		if !ok {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			repo := db.CoordinationWorktreeRepository()
			filter := coordinacion.WorktreeFilter{}
			if agente, _ := cmd.Flags().GetString("agente"); strings.TrimSpace(agente) != "" {
				filter.Agent = &agente
			}
			if proyectoRef, _ := cmd.Flags().GetString("proyecto"); strings.TrimSpace(proyectoRef) != "" {
				proyecto, err := db.GetProyecto(proyectoRef)
				if err != nil {
					return err
				}
				filter.ProjectID = &proyecto.ID
			}
			if estadoStr, _ := cmd.Flags().GetString("estado"); strings.TrimSpace(estadoStr) != "" {
				estado := coordinacion.WorktreeState(estadoStr)
				filter.State = &estado
			}
			worktrees, err = repo.List(filter)
		}
		if err != nil {
			return err
		}
		if len(worktrees) == 0 {
			fmt.Println("No hay worktrees con ese filtro.")
			return nil
		}
		fmt.Printf("%-5s %-14s %-10s %-20s %s\n", "ID", "PROYECTO", "AGENTE", "BRANCH", "RUTA")
		for _, worktree := range worktrees {
			fmt.Printf("%-5d %-14d %-10s %-20s %s\n", worktree.ID, worktree.ProjectID, worktree.Agent, truncar(worktree.Branch, 20), worktree.Path)
		}
		return nil
	},
}

var worktreeResolverCmd = &cobra.Command{
	Use:   "resolver <agente> <proyecto>",
	Short: "Resuelve la ruta del worktree activo de un agente para un proyecto",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		query := url.Values{
			"agente":   {args[0]},
			"proyecto": {args[1]},
			"estado":   {string(coordinacion.WorktreeActive)},
		}

		repo := db.CoordinationWorktreeRepository()
		estado := coordinacion.WorktreeActive
		filter := coordinacion.WorktreeFilter{Agent: &args[0], State: &estado}

		worktrees, ok, err := cargarWorktreesDesdeAPI(query)
		if !ok {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			proyecto, err := db.GetProyecto(args[1])
			if err != nil {
				return err
			}
			filter.ProjectID = &proyecto.ID
			worktrees, err = repo.List(filter)
		}
		if err != nil {
			return err
		}
		if len(worktrees) == 0 {
			return fmt.Errorf("no hay worktree activo para %s en %s", args[0], args[1])
		}
		worktree := worktrees[0]
		if jsonOut {
			return imprimirJSON(worktree)
		}
		fmt.Println(worktree.Path)
		return nil
	},
}

var worktreeCrearCmd = &cobra.Command{
	Use:   "crear <agente> <proyecto>",
	Short: "Crea un worktree aislado para trabajo paralelo",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tareaRaw, _ := cmd.Flags().GetInt64("tarea")
		lockRaw, _ := cmd.Flags().GetInt64("lock")
		name, _ := cmd.Flags().GetString("nombre")
		branch, _ := cmd.Flags().GetString("branch")
		baseRef, _ := cmd.Flags().GetString("base-ref")
		reason, _ := cmd.Flags().GetString("motivo")
		if worktree, ok, err := crearWorktreePorAPI(apiWorktreeRequest{
			Agente:   args[0],
			Proyecto: args[1],
			TareaID:  tareaRaw,
			LockID:   lockRaw,
			Nombre:   name,
			Branch:   branch,
			BaseRef:  baseRef,
			Motivo:   reason,
		}); ok {
			if err != nil {
				return err
			}
			fmt.Printf("✓ Worktree %d creado para %s\n", worktree.ID, worktree.Agent)
			fmt.Printf("  branch=%s\n", worktree.Branch)
			fmt.Printf("  ruta=%s\n", worktree.Path)
			return nil
		}
		svc := newCoordinationService()
		var taskID *int64
		if tareaRaw > 0 {
			taskID = &tareaRaw
		}
		var lockID *int64
		if lockRaw > 0 {
			lockID = &lockRaw
		}
		worktree, err := svc.PrepareWorktree(coordinacion.PrepareWorktreeInput{
			ProjectRef: args[1],
			Agent:      args[0],
			TaskID:     taskID,
			LockID:     lockID,
			Name:       name,
			Branch:     branch,
			BaseRef:    baseRef,
			Reason:     reason,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Worktree %d creado para %s\n", worktree.ID, worktree.Agent)
		fmt.Printf("  branch=%s\n", worktree.Branch)
		fmt.Printf("  ruta=%s\n", worktree.Path)
		return nil
	},
}

var worktreeCerrarCmd = &cobra.Command{
	Use:   "cerrar <id>",
	Short: "Cierra un worktree y opcionalmente lo elimina del repo",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}
		remove, _ := cmd.Flags().GetBool("eliminar")
		reason, _ := cmd.Flags().GetString("motivo")
		if worktree, ok, err := cerrarWorktreePorAPI(id, apiWorktreeRequest{
			Eliminar: remove,
			Motivo:   reason,
		}); ok {
			if err != nil {
				return err
			}
			fmt.Printf("✓ Worktree %d cerrado (%s)\n", worktree.ID, worktree.Path)
			return nil
		}
		svc := newCoordinationService()
		worktree, err := svc.CloseWorktree(id, remove, reason)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Worktree %d cerrado (%s)\n", worktree.ID, worktree.Path)
		return nil
	},
}

func init() {
	worktreeListarCmd.Flags().String("agente", "", "Filtrar por agente")
	worktreeListarCmd.Flags().String("proyecto", "", "Filtrar por proyecto")
	worktreeListarCmd.Flags().String("estado", "", "Filtrar por estado")

	worktreeCrearCmd.Flags().Int64("tarea", 0, "Tarea asociada")
	worktreeCrearCmd.Flags().Int64("lock", 0, "Lock asociado")
	worktreeCrearCmd.Flags().String("nombre", "", "Nombre del worktree")
	worktreeCrearCmd.Flags().String("branch", "", "Branch del worktree")
	worktreeCrearCmd.Flags().String("base-ref", "HEAD", "Referencia base para crear el worktree")
	worktreeCrearCmd.Flags().String("motivo", "", "Motivo del worktree")

	worktreeCerrarCmd.Flags().Bool("eliminar", false, "Eliminar también el worktree del repo")
	worktreeCerrarCmd.Flags().String("motivo", "", "Motivo del cierre")
	worktreeResolverCmd.Flags().Bool("json", false, "Salida JSON")

	worktreeCmd.AddCommand(worktreeListarCmd, worktreeResolverCmd, worktreeCrearCmd, worktreeCerrarCmd)
}
