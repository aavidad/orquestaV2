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

var agentePresupuestoCmd = &cobra.Command{
	Use:   "presupuesto",
	Short: "Lista la telemetría de presupuesto visible por agente",
	RunE: func(cmd *cobra.Command, args []string) error {
		activos, _ := cmd.Flags().GetBool("activos")
		jsonOut, _ := cmd.Flags().GetBool("json")
		refresh, _ := cmd.Flags().GetBool("refresh")
		agenteFiltro, _ := cmd.Flags().GetString("agente")
		if refresh {
			if _, ok, err := refrescarAgentesPresupuestoPorAPI(agenteFiltro); err != nil {
				return err
			} else if !ok {
				return agenteErrorServerFirst()
			}
		}
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
			fmt.Printf(" %s", resumenCuentaObservadaAgente(agente))
			fmt.Println()
		}
		return nil
	},
}

var agenteRankingCuentasCmd = &cobra.Command{
	Use:   "ranking-cuentas",
	Short: "Lista cuentas observadas ordenadas por presupuesto disponible",
	RunE: func(cmd *cobra.Command, args []string) error {
		activos, _ := cmd.Flags().GetBool("activos")
		jsonOut, _ := cmd.Flags().GetBool("json")
		resp, ok, err := listarAgentesRankingCuentasPorAPI(activos)
		if err != nil {
			return err
		}
		if !ok {
			return agenteErrorServerFirst()
		}
		if jsonOut {
			return imprimirJSON(resp)
		}
		for i, cuenta := range resp.Cuentas {
			fmt.Printf("%d. %-18s %s", i+1, cuenta.CuentaClave, resumenRankingCuenta(cuenta))
			fmt.Println()
		}
		return nil
	},
}

func resumenCuentaObservadaAgente(agente apiAgenteCuentaItem) string {
	email := strings.TrimSpace(agente.CuentaEmail)
	usuario := strings.TrimSpace(agente.CuentaUsuario)
	switch {
	case email != "" && usuario != "" && !strings.EqualFold(email, usuario):
		return fmt.Sprintf("%s (%s)", email, usuario)
	case email != "":
		return email
	case usuario != "":
		return "usuario " + usuario
	default:
		return "—"
	}
}

func resumenRankingCuenta(cuenta apiCuentaPresupuestoItem) string {
	base := resumenCuentaObservadaAgente(apiAgenteCuentaItem{
		CuentaEmail:   cuenta.CuentaEmail,
		CuentaUsuario: cuenta.CuentaUsuario,
	})
	detalle := ""
	switch cuenta.Criterio {
	case "remaining_tokens":
		if cuenta.RemainingTokens != nil {
			detalle = fmt.Sprintf("tokens %d", *cuenta.RemainingTokens)
		}
	case "remaining_credits":
		if cuenta.RemainingCredits != nil {
			detalle = fmt.Sprintf("credits %.2f", *cuenta.RemainingCredits)
		}
	case "remaining_messages":
		if cuenta.RemainingMessages != nil {
			detalle = fmt.Sprintf("mensajes %d", *cuenta.RemainingMessages)
		}
	case "remaining_seconds":
		if cuenta.RemainingSeconds != nil {
			detalle = fmt.Sprintf("segundos %d", *cuenta.RemainingSeconds)
		}
	case "cuota_pct":
		if cuenta.CuotaRestantePct != nil {
			detalle = fmt.Sprintf("%s %d%%", etiquetaCuotaVisibleCuenta(cuenta.PresupuestoStale, cuenta.PresupuestoFuente), *cuenta.CuotaRestantePct)
		}
	default:
		if cuenta.ObservedUsageTokens != nil {
			detalle = fmt.Sprintf("uso %d tok", *cuenta.ObservedUsageTokens)
		}
		if cuenta.ObservedUsageCostUSD != nil {
			if detalle != "" {
				detalle += " · "
			}
			detalle += fmt.Sprintf("coste est. $%.4f", *cuenta.ObservedUsageCostUSD)
		}
	}
	if cuenta.PresupuestoVentana != "" {
		if detalle != "" {
			detalle += " · "
		}
		detalle += "ventana " + cuenta.PresupuestoVentana
	}
	if cuenta.PresupuestoStale {
		if detalle != "" {
			detalle += " · "
		}
		detalle += "stale"
		if cuenta.PresupuestoCheckedAt != nil && !cuenta.PresupuestoCheckedAt.IsZero() {
			if age := edadPresupuestoObservado(cuenta.PresupuestoCheckedAt); age != "" {
				detalle += " (" + age + ")"
			}
		}
	} else if cuenta.ObservedUsageAt != nil && !cuenta.ObservedUsageAt.IsZero() {
		if detalle != "" {
			detalle += " · "
		}
		if age := edadPresupuestoObservado(cuenta.ObservedUsageAt); age != "" {
			detalle += "uso observado " + age
		}
	}
	if len(cuenta.Agentes) > 0 {
		if detalle != "" {
			detalle += " · "
		}
		detalle += "agentes " + strings.Join(cuenta.Agentes, ",")
	}
	if detalle == "" {
		return base
	}
	if base == "—" {
		return detalle
	}
	return base + " · " + detalle
}

func init() {
	for _, sub := range []*cobra.Command{agentePresupuestoCmd, agenteCuentasCmd, agenteRankingCuentasCmd} {
		sub.Flags().Bool("activos", false, "Mostrar solo agentes activos")
		sub.Flags().Bool("json", false, "Emitir JSON crudo")
		if sub == agentePresupuestoCmd {
			sub.Flags().Bool("refresh", false, "Refrescar telemetría observada antes de listar")
			sub.Flags().String("agente", "", "Refrescar solo este agente")
		}
		agenteCmd.AddCommand(sub)
	}
}
