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

	"github.com/spf13/cobra"
	"orquesta/db"
)

var rootCmd = &cobra.Command{
	Use:   "orquesta",
	Short: "CLI de orquestación multi-agente del workspace",
	Long: `Orquesta coordina proyectos, asignaciones, tareas, propuestas y sesiones
de los agentes del workspace. Los agentes deben iniciar sesión al comenzar,
obtener desde la BD sus reglas/skills/workflows y cerrar la sesión al terminar.`,
	SilenceUsage: true,
}

// Execute es el punto de entrada principal.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initDB)

	rootCmd.AddCommand(
		agenteCmd,
		sesionCmd,
		conectorCmd,
		proyectoCmd,
		asignacionCmd,
		lockCmd,
		worktreeCmd,
		tareaCmd,
		propuestaCmd,
		votarCmd,
		configCmd,
		exportarCmd,
		statusCmd,
		logsCmd,
	)
}

func initDB() {
	if shouldPreferServerForCurrentCommand() {
		return
	}
	if err := db.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "error abriendo base de datos: %v\n", err)
		os.Exit(1)
	}
}
