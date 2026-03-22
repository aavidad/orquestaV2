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
	Short: "CLI de orquestación multi-agente — PlataformaMunicipal",
	Long: `Orquesta coordina tareas, propuestas y sesiones de los agentes de desarrollo
de ContaGrx. Los agentes deben iniciar sesión al comenzar y cerrarla al terminar.`,
	SilenceUsage: true,
}

// Execute es el punto de entrada principal.
func Execute() {
	defer db.Close()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initDB)

	rootCmd.AddCommand(
		sesionCmd,
		tareaCmd,
		propuestaCmd,
		votarCmd,
		configCmd,
		exportarCmd,
		statusCmd,
	)
}

func initDB() {
	if shouldBypassLocalDB(os.Args[1:]) {
		return
	}
	if err := db.Open(); err != nil {
		fmt.Fprintf(os.Stderr, "error abriendo base de datos: %v\n", err)
		os.Exit(1)
	}
}
