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
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Gestión de configuración global",
}

var configVerCmd = &cobra.Command{
	Use:   "ver [clave]",
	Short: "Muestra toda la configuración o el valor de una clave",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			resp, ok, err := cargarConfigDesdeAPI(args[0])
			if err != nil {
				return fmt.Errorf("clave '%s' no encontrada", args[0])
			}
			if !ok {
				return serverFirstCommandError("config ver")
			}
			fmt.Printf("%s = %s\n", args[0], resp.Valor)
			return nil
		}
		resp, ok, err := cargarConfigDesdeAPI("")
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("config ver")
		}
		for k, v := range resp.Config {
			fmt.Printf("%-30s = %s\n", k, v)
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <clave> <valor>",
	Short: "Establece el valor de una clave de configuración",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ok, err := configurarValorPorAPI(args[0], args[1])
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("config set")
		}
		fmt.Printf("✓ %s = %s\n", args[0], args[1])
		return nil
	},
}

var configAgenteNuevoCmd = &cobra.Command{
	Use:   "agente-nuevo <nombre> <rol>",
	Short: "Registra un nuevo agente (rol: programador, documentador, admin)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ok, err := registrarAgentePorAPI(args[0], args[1])
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("config agente-nuevo")
		}
		fmt.Printf("✓ Agente '%s' [%s] registrado\n", args[0], args[1])
		return nil
	},
}

var configAgenteRetirarCmd = &cobra.Command{
	Use:   "agente-retirar <nombre>",
	Short: "Retira a un agente del equipo (no puede votar ni trabajar)",
	Long: `Deshabilita al agente: cierra su sesión, elimina sus votos pendientes en propuestas
abiertas (para no bloquear el consenso) y le impide iniciar nuevas sesiones.
Sus contribuciones históricas se conservan.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nombre := args[0]
		if nombre == "alberto" {
			return fmt.Errorf("no puedes retirar al administrador")
		}
		ok, err := retirarAgentePorAPI(nombre)
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("config agente-retirar")
		}
		fmt.Printf("✓ Agente '%s' retirado del equipo.\n", nombre)
		fmt.Printf("  Sus votos pendientes en propuestas abiertas han sido eliminados.\n")
		fmt.Printf("  El sistema reevaluará el consenso automáticamente en el próximo voto.\n")
		return nil
	},
}

var configAgenteRehabilitarCmd = &cobra.Command{
	Use:   "agente-rehabilitar <nombre>",
	Short: "Reactiva a un agente retirado",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ok, err := rehabilitarAgentePorAPI(args[0])
		if err != nil {
			return err
		}
		if !ok {
			return serverFirstCommandError("config agente-rehabilitar")
		}
		fmt.Printf("✓ Agente '%s' rehabilitado.\n", args[0])
		fmt.Printf("  Debe iniciar sesión con: orquesta sesion inicio %s\n", args[0])
		return nil
	},
}

func init() {
	configCmd.AddCommand(
		configVerCmd, configSetCmd,
		configAgenteNuevoCmd, configAgenteRetirarCmd, configAgenteRehabilitarCmd,
	)
}
