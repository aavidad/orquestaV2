/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"os"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Gestión de locks con lease y heartbeat",
}

func coordinacionModoRecuperacionLocalExplicito() bool {
	return strings.TrimSpace(os.Getenv("ORQUESTA_FORCE_LOCAL_DB")) == "1"
}

func coordinacionErrorServerFirst() error {
	return fmt.Errorf("este comando exige servidor/daemon de Orquesta; usa --local solo en recuperacion explicita o exporta ORQUESTA_FORCE_LOCAL_DB=1")
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

		locks, ok, err := cargarLocksDesdeAPI(query)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("lock listar")
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
		lock, ok, err := tomarLockPorAPI(apiLockRequest{
			Agente:       args[0],
			Proyecto:     projectRef,
			TareaID:      taskRaw,
			ScopeType:    args[1],
			ScopeKey:     args[2],
			Ruta:         path,
			Branch:       branch,
			Motivo:       reason,
			LeaseSeconds: leaseSeconds,
		})
		if ok {
			if err != nil {
				return err
			}
			fmt.Printf("✓ Lock %d tomado por %s (%s:%s)\n", lock.ID, lock.Agent, lock.ScopeType, lock.ScopeKey)
			fmt.Printf("  lease_token=%s\n", lock.LeaseToken)
			fmt.Printf("  expira=%s\n", lock.ExpiresAt.Format("2006-01-02 15:04:05"))
			return nil
		}
		if err != nil {
			return err
		}
		return serverFirstCommandError("lock tomar")
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
		lock, ok, err := renovarLockPorAPI(id, apiLockRequest{
			Agente:       args[1],
			LeaseToken:   args[2],
			LeaseSeconds: leaseSeconds,
		})
		if ok {
			if err != nil {
				return err
			}
			fmt.Printf("✓ Lock %d renovado hasta %s\n", lock.ID, lock.ExpiresAt.Format("2006-01-02 15:04:05"))
			return nil
		}
		if err != nil {
			return err
		}
		return serverFirstCommandError("lock renovar")
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
		lock, ok, err := liberarLockPorAPI(id, apiLockRequest{
			Agente:     args[1],
			LeaseToken: args[2],
			Motivo:     reason,
		})
		if ok {
			if err != nil {
				return err
			}
			fmt.Printf("✓ Lock %d liberado (%s:%s)\n", lock.ID, lock.ScopeType, lock.ScopeKey)
			return nil
		}
		if err != nil {
			return err
		}
		return serverFirstCommandError("lock liberar")
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
