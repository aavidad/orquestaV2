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
		dbPath := db.CurrentDBPath()
		addr := rpclocal.ResolveServerAddr()

		info, err := rpclocal.LoadServerInfo()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		pingErr := rpclocal.Ping(ctx, addr)

		renderPersistenciaInfo(cmd.OutOrStdout(), infoPath, dbPath, addr, info, err, pingErr)
		return nil
	},
}

func init() {
	persistenciaCmd.AddCommand(persistenciaInfoCmd)
	rootCmd.AddCommand(persistenciaCmd)
}

func renderPersistenciaInfo(w io.Writer, infoPath, dbPath, addr string, info *rpclocal.ServerInfo, infoErr, pingErr error) {
	fmt.Fprintf(w, "Modo esperado: servidor local\n")
	fmt.Fprintf(w, "DB objetivo:   %s\n", dbPath)
	fmt.Fprintf(w, "Statefile:     %s\n", infoPath)
	fmt.Fprintf(w, "Addr resuelta: %s\n", addr)

	if infoErr != nil || info == nil {
		fmt.Fprintf(w, "Servidor:      sin state (%v)\n", infoErr)
	} else {
		fmt.Fprintf(w, "Servidor:      pid=%d addr=%s started_at=%s\n", info.PID, info.Addr, info.StartedAt.Format(time.RFC3339))
	}

	if pingErr != nil {
		fmt.Fprintf(w, "Health RPC:    KO (%v)\n", pingErr)
		fmt.Fprintf(w, "Ruta activa:   sin servidor; local solo con --local o ORQUESTA_ALLOW_LOCAL_FALLBACK=1\n")
		return
	}

	fmt.Fprintf(w, "Health RPC:    OK\n")
	fmt.Fprintf(w, "Ruta activa:   servidor local\n")
}
