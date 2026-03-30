/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
	"orquesta/internal/rpclocal"
)

var persistenciaCmd = &cobra.Command{
	Use:   "persistencia",
	Short: "Observabilidad del modo de persistencia",
}

var persistenciaInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Muestra estado esperado de persistencia y servidor local",
	RunE: func(cmd *cobra.Command, args []string) error {
		infoPath := rpclocal.DefaultInfoPath()
		storageDriver := db.CurrentStorageDriver()
		storageTarget := db.CurrentStorageDisplayTarget()
		addr := rpclocal.ResolveServerAddr()

		info, recoveredFromHealth, err := loadServerInfoWithHealthFallback(addr)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		pingErr := rpclocal.Ping(ctx, addr)

		renderPersistenciaInfo(cmd.OutOrStdout(), infoPath, storageDriver, storageTarget, addr, info, recoveredFromHealth, err, pingErr)
		return nil
	},
}

var persistenciaVerificarCmd = &cobra.Command{
	Use:   "verificar",
	Short: "Verifica la salud del backend activo desde Orquesta",
	RunE: func(cmd *cobra.Command, args []string) error {
		var resp apiPersistenciaVerificacionResponse
		if ok, err := apiGet("/api/persistencia/verificar", &resp); err != nil {
			return err
		} else if !ok {
			return serverFirstCommandError("persistencia verificar")
		}
		if resp.Informe == nil {
			return fmt.Errorf("la verificacion de persistencia no ha devuelto informe")
		}
		renderVerificacionPersistencia(cmd.OutOrStdout(), resp.Informe)
		return nil
	},
}

func init() {
	persistenciaCmd.AddCommand(persistenciaInfoCmd, persistenciaVerificarCmd)
	rootCmd.AddCommand(persistenciaCmd)
}

func renderPersistenciaInfo(w io.Writer, infoPath, storageDriver, storageTarget, addr string, info *rpclocal.ServerInfo, recoveredFromHealth bool, infoErr, pingErr error) {
	fmt.Fprintf(w, "Modo esperado: servidor local\n")
	fmt.Fprintf(w, "Storage driver: %s\n", storageDriver)
	fmt.Fprintf(w, "Storage target: %s\n", storageTarget)
	fmt.Fprintf(w, "Statefile:     %s\n", infoPath)
	fmt.Fprintf(w, "Addr resuelta: %s\n", addr)

	if infoErr != nil || info == nil {
		fmt.Fprintf(w, "Servidor:      sin state (%v)\n", infoErr)
	} else {
		fmt.Fprintf(w, "Servidor:      pid=%d addr=%s storage=%s %s started_at=%s\n",
			info.PID,
			info.Addr,
			info.StorageDriver,
			resolveServerStorageTarget(info),
			info.StartedAt.Format(time.RFC3339),
		)
		if recoveredFromHealth {
			fmt.Fprintf(w, "Descubrimiento: fallback a healthz por statefile ausente\n")
		}
	}

	if pingErr != nil {
		fmt.Fprintf(w, "Health RPC:    KO (%v)\n", pingErr)
		fmt.Fprintf(w, "Ruta activa:   sin servidor; local solo con ORQUESTA_FORCE_LOCAL_DB=1 o --local y solo para recuperación\n")
		return
	}

	fmt.Fprintf(w, "Health RPC:    OK\n")
	fmt.Fprintf(w, "Ruta activa:   servidor local\n")
}

func renderVerificacionPersistencia(w io.Writer, informe *db.InformePersistencia) {
	if informe == nil {
		fmt.Fprintf(w, "Verificación no disponible\n")
		return
	}
	estado := "OK"
	if !informe.Sano {
		estado = "ERROR"
	}
	fmt.Fprintf(w, "Storage driver: %s\n", strings.TrimSpace(informe.Driver))
	fmt.Fprintf(w, "Storage target: %s\n", strings.TrimSpace(informe.Target))
	fmt.Fprintf(w, "Estado general: %s\n", estado)
	for _, comprobacion := range informe.Comprobaciones {
		detalle := strings.TrimSpace(comprobacion.Detalle)
		if detalle == "" {
			detalle = "-"
		}
		fmt.Fprintf(w, "- [%s] %s: %s\n", strings.ToUpper(strings.TrimSpace(comprobacion.Estado)), strings.TrimSpace(comprobacion.Nombre), detalle)
	}
}
