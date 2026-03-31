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

var agentePresupuestoCmd = &cobra.Command{
	Use:   "presupuesto",
	Short: "Lista la telemetría de presupuesto visible por agente",
	RunE: func(cmd *cobra.Command, args []string) error {
		activos, _ := cmd.Flags().GetBool("activos")
		jsonOut, _ := cmd.Flags().GetBool("json")
		resp, ok, err := listarAgentesPresupuestoPorAPI(activos)
		if err != nil {
			return err
		}
		if !ok {
			return agenteErrorServerFirst()
		}
		if jsonOut {
			return imprimirJSON(resp)
		}
		for _, agente := range resp.Agentes {
			if agente == nil {
				continue
			}
			fmt.Printf("%-16s", agente.Nombre)
			if detalle := resumenCuotaAgente(agente); detalle != "" {
				fmt.Printf(" %s", detalle)
			}
			fmt.Println()
		}
		return nil
	},
}

var agenteCuentasCmd = &cobra.Command{
	Use:   "cuentas",
	Short: "Lista nombre de agente y cuenta observada",
	RunE: func(cmd *cobra.Command, args []string) error {
		activos, _ := cmd.Flags().GetBool("activos")
		jsonOut, _ := cmd.Flags().GetBool("json")
		resp, ok, err := listarAgentesCuentasPorAPI(activos)
		if err != nil {
			return err
		}
		if !ok {
			return agenteErrorServerFirst()
		}
		if jsonOut {
			return imprimirJSON(resp)
		}
		for _, agente := range resp.Agentes {
			fmt.Printf("%-16s", agente.Nombre)
			if agente.CuentaEmail != "" {
				fmt.Printf(" %s", agente.CuentaEmail)
			} else {
				fmt.Printf(" —")
			}
			if agente.CuentaUsuario != "" {
				fmt.Printf(" (%s)", agente.CuentaUsuario)
			}
			fmt.Println()
		}
		return nil
	},
}

func init() {
	for _, sub := range []*cobra.Command{agentePresupuestoCmd, agenteCuentasCmd} {
		sub.Flags().Bool("activos", false, "Mostrar solo agentes activos")
		sub.Flags().Bool("json", false, "Emitir JSON crudo")
		agenteCmd.AddCommand(sub)
	}
}
