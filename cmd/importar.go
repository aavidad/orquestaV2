/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"orquesta/importacionapp"
)

// importarCmd agrupa los comandos de importación de datos históricos.
var importarCmd = &cobra.Command{
	Use:   "importar",
	Short: "Importa historial legado al sistema",
}

var importacionService = importacionapp.NewService(importacionapp.Repository{})

// importarHistorialCmd importa propuestas y votaciones históricas desde el legado markdown.
var importarHistorialCmd = &cobra.Command{
	Use:   "historial",
	Short: "Importa el historial completo de OPs y votaciones (OP-001 a OP-029)",
	Long: `Carga en la base de datos todas las propuestas y sus votaciones históricas
desde el legado markdown previo a la app de orquestación. Este comando sirve
solo para migración histórica y no forma parte del flujo actual de trabajo.

Las propuestas se crean con el código original (OP-001..OP-029) y el estado final
que tenían en el historial legado.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Importando historial de propuestas y votaciones…")
		if err := importacionService.ImportHistory(); err != nil {
			return err
		}
		fmt.Println("✓ Historial importado correctamente.")
		fmt.Println("  Usa 'orquesta status' para ver el estado del proyecto.")
		return nil
	},
}

func init() {
	importarCmd.AddCommand(importarHistorialCmd)
	rootCmd.AddCommand(importarCmd)
}
