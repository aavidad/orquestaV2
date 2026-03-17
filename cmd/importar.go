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
	"orquesta/db"
)

// importarCmd agrupa los comandos de importación de datos históricos.
var importarCmd = &cobra.Command{
	Use:   "importar",
	Short: "Importa datos históricos desde ficheros markdown al sistema",
}

// importarHistorialCmd importa todas las OPs y votaciones históricas de Opinion.md
var importarHistorialCmd = &cobra.Command{
	Use:   "historial",
	Short: "Importa el historial completo de OPs y votaciones (OP-001 a OP-029)",
	Long: `Carga en la base de datos todas las propuestas y sus votaciones históricas
desde los ficheros Opinion.md y archivados. Útil para dejar constancia de cómo
se tomaron las decisiones de arquitectura del proyecto.

Las propuestas se crean con el código original (OP-001..OP-029) y el estado final
que tenían en los ficheros Markdown.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Importando historial de propuestas y votaciones…")
		if err := db.ImportarHistorialOPs(); err != nil {
			return err
		}
		fmt.Println("Importando tareas de Ola 2…")
		if err := db.ImportarTareasOla2(); err != nil {
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
