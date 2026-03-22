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
	"orquesta/coordination"
	"orquesta/db"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Gestión de locks con lease y heartbeat",
}

var lockListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista locks registrados",
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

		repo := db.SQLiteLockRepository{}
		filter := coordination.LockFilter{}
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
			estado := coordination.LockState(estadoStr)
			filter.State = &estado
		}
		locks, ok, err := cargarLocksDesdeAPI(query)
		if !ok {
			locks, err = repo.List(filter)
		}
		if err != nil {
			return err
		}
		if len(locks) == 0 {
			fmt.Println("No hay locks con ese filtro.")
			return nil
		}
		fmt.Printf("%-5s %-10s %-26s %-10s %-8s %s\n", "ID", "TIPO", "CLAVE", "AGENTE", "ESTADO", "EXPIRA")
		for _, lock := range locks {
			fmt.Printf("%-5d %-10s %-26s %-10s %-8s %s\n",
				lock.ID, lock.ScopeType, truncar(lock.ScopeKey, 26), lock.Agent, lock.State, lock.ExpiresAt.Format("2006-01-02 15:04:05"))
		}
		return nil
	},
}

var lockTomarCmd = &cobra.Command{
	Use:   "tomar <agente> <scope-type> <scope-key>",
	Short: "Toma un lock sobre un recurso",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRef, _ := cmd.Flags().GetString("proyecto")
		taskRaw, _ := cmd.Flags().GetInt64("tarea")
		path, _ := cmd.Flags().GetString("ruta")
		branch, _ := cmd.Flags().GetString("branch")
		reason, _ := cmd.Flags().GetString("motivo")
		leaseSeconds, _ := cmd.Flags().GetInt("lease-seconds")
		if lock, ok, err := tomarLockPorAPI(apiLockRequest{
			Agente:       args[0],
			Proyecto:     projectRef,
			TareaID:      taskRaw,
			ScopeType:    args[1],
			ScopeKey:     args[2],
			Ruta:         path,
			Branch:       branch,
			Motivo:       reason,
			LeaseSeconds: leaseSeconds,
		}); ok {
			if err != nil {
				return err
			}
			fmt.Printf("✓ Lock %d tomado por %s (%s:%s)\n", lock.ID, lock.Agent, lock.ScopeType, lock.ScopeKey)
			fmt.Printf("  lease_token=%s\n", lock.LeaseToken)
			fmt.Printf("  expira=%s\n", lock.ExpiresAt.Format("2006-01-02 15:04:05"))
			return nil
		}

		svc := newCoordinationService()
		var projectID *int64
		if strings.TrimSpace(projectRef) != "" {
			proyecto, err := db.GetProyecto(projectRef)
			if err != nil {
				return err
			}
			projectID = &proyecto.ID
		}
		var taskID *int64
		if taskRaw > 0 {
			taskID = &taskRaw
		}
		var sessionID *int64
		if projectID != nil {
			sesion, err := db.GetSesionActiva(args[0], projectID)
			if err == nil && sesion != nil {
				sessionID = &sesion.ID
			}
		}
		lock, err := svc.AcquireLock(coordination.AcquireLockInput{
			ProjectID:    projectID,
			TaskID:       taskID,
			SessionID:    sessionID,
			Agent:        args[0],
			ScopeType:    args[1],
			ScopeKey:     args[2],
			Path:         path,
			Branch:       branch,
			Reason:       reason,
			LeaseSeconds: leaseSeconds,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Lock %d tomado por %s (%s:%s)\n", lock.ID, lock.Agent, lock.ScopeType, lock.ScopeKey)
		fmt.Printf("  lease_token=%s\n", lock.LeaseToken)
		fmt.Printf("  expira=%s\n", lock.ExpiresAt.Format("2006-01-02 15:04:05"))
		return nil
	},
}

var lockRenovarCmd = &cobra.Command{
	Use:   "renovar <id> <agente> <lease-token>",
	Short: "Renueva el lease de un lock activo",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}
		leaseSeconds, _ := cmd.Flags().GetInt("lease-seconds")
		if lock, ok, err := renovarLockPorAPI(id, apiLockRequest{
			Agente:       args[1],
			LeaseToken:   args[2],
			LeaseSeconds: leaseSeconds,
		}); ok {
			if err != nil {
				return err
			}
			fmt.Printf("✓ Lock %d renovado hasta %s\n", lock.ID, lock.ExpiresAt.Format("2006-01-02 15:04:05"))
			return nil
		}
		svc := newCoordinationService()
		lock, err := svc.RenewLock(coordination.RenewLockInput{
			ID:           id,
			Agent:        args[1],
			LeaseToken:   args[2],
			LeaseSeconds: leaseSeconds,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Lock %d renovado hasta %s\n", lock.ID, lock.ExpiresAt.Format("2006-01-02 15:04:05"))
		return nil
	},
}

var lockLiberarCmd = &cobra.Command{
	Use:   "liberar <id> <agente> <lease-token>",
	Short: "Libera un lock activo",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}
		reason, _ := cmd.Flags().GetString("motivo")
		if lock, ok, err := liberarLockPorAPI(id, apiLockRequest{
			Agente:     args[1],
			LeaseToken: args[2],
			Motivo:     reason,
		}); ok {
			if err != nil {
				return err
			}
			fmt.Printf("✓ Lock %d liberado (%s:%s)\n", lock.ID, lock.ScopeType, lock.ScopeKey)
			return nil
		}
		svc := newCoordinationService()
		lock, err := svc.ReleaseLock(coordination.ReleaseLockInput{
			ID:         id,
			Agent:      args[1],
			LeaseToken: args[2],
			Reason:     reason,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Lock %d liberado (%s:%s)\n", lock.ID, lock.ScopeType, lock.ScopeKey)
		return nil
	},
}

func init() {
	lockListarCmd.Flags().String("agente", "", "Filtrar por agente")
	lockListarCmd.Flags().String("proyecto", "", "Filtrar por proyecto")
	lockListarCmd.Flags().String("estado", "", "Filtrar por estado")

	lockTomarCmd.Flags().String("proyecto", "", "Proyecto asociado")
	lockTomarCmd.Flags().Int64("tarea", 0, "Tarea asociada")
	lockTomarCmd.Flags().String("ruta", "", "Ruta asociada al lock")
	lockTomarCmd.Flags().String("branch", "", "Branch asociada al lock")
	lockTomarCmd.Flags().String("motivo", "", "Motivo del lock")
	lockTomarCmd.Flags().Int("lease-seconds", 0, "Duración del lease en segundos")

	lockRenovarCmd.Flags().Int("lease-seconds", 0, "Nueva duración del lease en segundos")
	lockLiberarCmd.Flags().String("motivo", "", "Motivo de liberación")

	lockCmd.AddCommand(lockListarCmd, lockTomarCmd, lockRenovarCmd, lockLiberarCmd)
}
