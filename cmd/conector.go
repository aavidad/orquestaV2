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

var conectorCmd = &cobra.Command{
	Use:   "conector",
	Short: "Registro de conectores de agentes/modelos",
}

var conectorListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista los conectores registrados",
	RunE: func(cmd *cobra.Command, args []string) error {
		conectores, ok, err := cargarConectoresDesdeAPI()
		if !ok {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			conectores, err = db.ListarConectores()
		}
		if err != nil {
			return err
		}
		if len(conectores) == 0 {
			fmt.Println("No hay conectores registrados.")
			return nil
		}
		fmt.Printf("%-5s %-14s %-16s %-12s %-8s %s\n", "ID", "SLUG", "NOMBRE", "TRANSPORTE", "ACTIVO", "COMANDO")
		fmt.Printf("%-5s %-14s %-16s %-12s %-8s %s\n", "─────", "──────────────", "────────────────", "────────────", "────────", "────────────────────────")
		for _, c := range conectores {
			activo := "no"
			if c.Activo {
				activo = "sí"
			}
			fmt.Printf("%-5d %-14s %-16s %-12s %-8s %s\n", c.ID, c.Slug, c.Nombre, c.Transporte, activo, c.Comando)
		}
		return nil
	},
}

var conectorVerCmd = &cobra.Command{
	Use:   "ver <slug|id>",
	Short: "Muestra el detalle de un conector",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, ok, err := cargarConectorDesdeAPI(args[0])
		if !ok {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			c, err = db.GetConector(args[0])
		}
		if err != nil {
			return err
		}
		fmt.Printf("Conector #%d — %s\n", c.ID, c.Nombre)
		fmt.Printf("  Slug:       %s\n", c.Slug)
		fmt.Printf("  Transporte: %s\n", c.Transporte)
		fmt.Printf("  Comando:    %s\n", c.Comando)
		fmt.Printf("  Args JSON:  %s\n", c.ArgsJSON)
		fmt.Printf("  Env JSON:   %s\n", c.EnvJSON)
		fmt.Printf("  Metadata:   %s\n", c.MetadataJSON)
		fmt.Printf("  Activo:     %t\n", c.Activo)
		return nil
	},
}

var conectorRegistrarCmd = &cobra.Command{
	Use:   "registrar",
	Short: "Registra o actualiza un conector",
	RunE: func(cmd *cobra.Command, args []string) error {
		slug, _ := cmd.Flags().GetString("slug")
		nombre, _ := cmd.Flags().GetString("nombre")
		transporte, _ := cmd.Flags().GetString("transporte")
		comando, _ := cmd.Flags().GetString("comando")
		argsJSON, _ := cmd.Flags().GetString("args-json")
		envJSON, _ := cmd.Flags().GetString("env-json")
		metadataJSON, _ := cmd.Flags().GetString("metadata-json")
		inactivo, _ := cmd.Flags().GetBool("inactivo")

		id, ok, err := registrarConectorPorAPI(apiConectorUpsertRequest{
			Slug:         slug,
			Nombre:       nombre,
			Transporte:   transporte,
			Comando:      comando,
			ArgsJSON:     argsJSON,
			EnvJSON:      envJSON,
			MetadataJSON: metadataJSON,
			Activo:       !inactivo,
		})
		if !ok {
			if err := ensureLocalDB(); err != nil {
				return err
			}
			id, err = db.UpsertConector(&db.Conector{
				Slug:         slug,
				Nombre:       nombre,
				Transporte:   transporte,
				Comando:      comando,
				ArgsJSON:     argsJSON,
				EnvJSON:      envJSON,
				MetadataJSON: metadataJSON,
				Activo:       !inactivo,
			})
		}
		if err != nil {
			return err
		}
		fmt.Printf("✓ Conector registrado/actualizado (id: %d)\n", id)
		return nil
	},
}

func init() {
	conectorRegistrarCmd.Flags().String("slug", "", "Slug único del conector")
	conectorRegistrarCmd.Flags().String("nombre", "", "Nombre descriptivo")
	conectorRegistrarCmd.Flags().String("transporte", "cli", "cli, mcp_stdio, mcp_http, api, otro")
	conectorRegistrarCmd.Flags().String("comando", "", "Comando base del conector")
	conectorRegistrarCmd.Flags().String("args-json", "[]", "Argumentos por defecto en JSON")
	conectorRegistrarCmd.Flags().String("env-json", "{}", "Variables de entorno por defecto en JSON")
	conectorRegistrarCmd.Flags().String("metadata-json", "{}", "Metadata libre en JSON")
	conectorRegistrarCmd.Flags().Bool("inactivo", false, "Registrar como inactivo")

	conectorCmd.AddCommand(conectorListarCmd, conectorVerCmd, conectorRegistrarCmd)
}
