/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var agenteRegistrarCmd = &cobra.Command{
	Use:   "registrar [nombre]",
	Short: "Registra un agente por la vía canónica de la app",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nombre := ""
		if len(args) == 1 {
			nombre = strings.TrimSpace(args[0])
		}
		rol, _ := cmd.Flags().GetString("rol")
		rol = strings.TrimSpace(rol)
		if rol == "" {
			rol = "programador"
		}
		proveedor, _ := cmd.Flags().GetString("proveedor")
		proveedor = strings.TrimSpace(proveedor)

		switch {
		case nombre != "":
			ok, err := registrarAgentePorAPI(nombre, rol)
			if err != nil {
				return err
			}
			if !ok {
				return serverFirstCommandError("agente registrar")
			}
			fmt.Printf("✓ Agente '%s' [%s] registrado\n", nombre, rol)
			return nil
		case proveedor != "":
			nombreAuto, ok, err := registrarAgenteAutoPorAPI(proveedor, rol)
			if err != nil {
				return err
			}
			if !ok {
				return serverFirstCommandError("agente registrar")
			}
			fmt.Printf("✓ Agente '%s' [%s] registrado automáticamente para proveedor %s\n", nombreAuto, rol, proveedor)
			return nil
		default:
			return fmt.Errorf("debes indicar un nombre o bien --proveedor")
		}
	},
}

func init() {
	agenteRegistrarCmd.Flags().String("rol", "programador", "Rol del agente: programador, documentador, admin")
	agenteRegistrarCmd.Flags().String("proveedor", "", "Proveedor para autogenerar nombre canónico (codex, claude, gemini, ollama, etc.)")
	agenteCmd.AddCommand(agenteRegistrarCmd)
}
