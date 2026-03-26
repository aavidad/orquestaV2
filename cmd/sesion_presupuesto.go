/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var sesionPresupuestoCmd = &cobra.Command{
	Use:   "presupuesto",
	Short: "Registro y consulta de presupuestos de sesión",
}

var sesionPresupuestoRegistrarCmd = &cobra.Command{
	Use:   "registrar",
	Short: "Registra un snapshot de presupuesto para una sesión",
	RunE: func(cmd *cobra.Command, args []string) error {
		sesionID, agente, err := referenciaSesionPresupuesto(cmd)
		if err != nil {
			return err
		}
		poolID, err := int64FlagOpt(cmd, "pool-id")
		if err != nil {
			return err
		}
		startedAt, err := parseTimeFlag(cmd, "window-started-at")
		if err != nil {
			return err
		}
		resetAt, err := parseTimeFlag(cmd, "reset-at")
		if err != nil {
			return err
		}
		remainingSeconds, err := nullableInt64Flag(cmd, "remaining-seconds")
		if err != nil {
			return err
		}
		remainingMessages, err := nullableInt64Flag(cmd, "remaining-messages")
		if err != nil {
			return err
		}
		remainingTokens, err := nullableInt64Flag(cmd, "remaining-tokens")
		if err != nil {
			return err
		}
		remainingCredits, err := nullableFloat64Flag(cmd, "remaining-credits")
		if err != nil {
			return err
		}
		windowKind, _ := cmd.Flags().GetString("window-kind")
		modelSlug, _ := cmd.Flags().GetString("model")
		budgetSource, _ := cmd.Flags().GetString("budget-source")
		rawSnapshot, _ := cmd.Flags().GetString("raw-snapshot")

		req := apiSesionPresupuestoRequest{
			SesionID:          sesionID,
			Agente:            agente,
			PoolID:            poolID,
			ModelSlug:         strings.TrimSpace(modelSlug),
			WindowKind:        strings.TrimSpace(windowKind),
			WindowStartedAt:   startedAt,
			ResetAt:           resetAt,
			RemainingSeconds:  remainingSeconds,
			RemainingMessages: remainingMessages,
			RemainingTokens:   remainingTokens,
			RemainingCredits:  remainingCredits,
			BudgetSource:      strings.TrimSpace(budgetSource),
			RawSnapshotJSON:   strings.TrimSpace(rawSnapshot),
		}
		if resp, ok, err := registrarPresupuestoSesionPorAPI(req); err != nil {
			return err
		} else if ok {
			imprimirRegistroPresupuesto(resp.ID, resp.Presupuesto.SesionID, resp.Evaluacion)
			return nil
		}
		return serverFirstCommandError("sesion presupuesto registrar")
	},
}

var sesionPresupuestoVerCmd = &cobra.Command{
	Use:   "ver",
	Short: "Muestra el último presupuesto de una sesión",
	RunE: func(cmd *cobra.Command, args []string) error {
		sesionID, agente, err := referenciaSesionPresupuesto(cmd)
		if err != nil {
			return err
		}
		if resp, ok, err := cargarPresupuestoSesionDesdeAPI(sesionID, agente); err != nil {
			return err
		} else if ok {
			imprimirDetallePresupuesto(resp.Presupuesto.SesionID, resp.Presupuesto, resp.Evaluacion)
			return nil
		}
		return serverFirstCommandError("sesion presupuesto ver")
	},
}

func init() {
	sesionPresupuestoRegistrarCmd.Flags().Int64("sesion", 0, "ID de sesión")
	sesionPresupuestoRegistrarCmd.Flags().String("agente", "", "Resolver la sesión activa del agente")
	sesionPresupuestoRegistrarCmd.Flags().Int64("pool-id", 0, "Pool de capacidad asociado")
	sesionPresupuestoRegistrarCmd.Flags().String("model", "", "Modelo observado")
	sesionPresupuestoRegistrarCmd.Flags().String("window-kind", "unknown", "Tipo de ventana: unknown, rolling, fixed, session, weekly...")
	sesionPresupuestoRegistrarCmd.Flags().String("window-started-at", "", "Inicio de ventana en RFC3339")
	sesionPresupuestoRegistrarCmd.Flags().String("reset-at", "", "Reset de ventana en RFC3339")
	sesionPresupuestoRegistrarCmd.Flags().Int64("remaining-seconds", -1, "Segundos restantes")
	sesionPresupuestoRegistrarCmd.Flags().Int64("remaining-messages", -1, "Mensajes restantes")
	sesionPresupuestoRegistrarCmd.Flags().Int64("remaining-tokens", -1, "Tokens restantes")
	sesionPresupuestoRegistrarCmd.Flags().Float64("remaining-credits", -1, "Créditos restantes")
	sesionPresupuestoRegistrarCmd.Flags().String("budget-source", "", "Fuente: manual, runtime, api, inferida...")
	sesionPresupuestoRegistrarCmd.Flags().String("raw-snapshot", "{}", "Snapshot bruto en JSON")

	sesionPresupuestoVerCmd.Flags().Int64("sesion", 0, "ID de sesión")
	sesionPresupuestoVerCmd.Flags().String("agente", "", "Resolver la sesión activa del agente")

	sesionPresupuestoCmd.AddCommand(sesionPresupuestoRegistrarCmd, sesionPresupuestoVerCmd)
	sesionCmd.AddCommand(sesionPresupuestoCmd)
}

func referenciaSesionPresupuesto(cmd *cobra.Command) (int64, string, error) {
	sesionID, err := cmd.Flags().GetInt64("sesion")
	if err != nil {
		return 0, "", err
	}
	agente, err := cmd.Flags().GetString("agente")
	if err != nil {
		return 0, "", err
	}
	agente = strings.TrimSpace(agente)
	if sesionID <= 0 && agente == "" {
		return 0, "", fmt.Errorf("debe indicar --sesion o --agente")
	}
	return sesionID, agente, nil
}

func imprimirRegistroPresupuesto(id, sesionID int64, ev *db.EvaluacionPresupuesto) {
	fmt.Printf("✓ Presupuesto registrado (id: %d) para sesión #%d\n", id, sesionID)
	fmt.Printf("  Estado: %s", ev.Estado)
	if ev.DebeHandoff {
		fmt.Printf("  → handoff preventivo")
	}
	fmt.Println()
	if ev.Motivo != "" {
		fmt.Printf("  Motivo: %s\n", ev.Motivo)
	}
}

func imprimirDetallePresupuesto(sesionID int64, p *db.PresupuestoSesion, ev *db.EvaluacionPresupuesto) {
	fmt.Printf("╔══════════════════════════════════════════════╗\n")
	fmt.Printf("║  PRESUPUESTO — sesión #%d\n", sesionID)
	fmt.Printf("╚══════════════════════════════════════════════╝\n")
	fmt.Printf("Estado:         %s\n", ev.Estado)
	fmt.Printf("Budget source:  %s\n", p.BudgetSource)
	fmt.Printf("Window kind:    %s\n", p.WindowKind)
	fmt.Printf("Model slug:     %s\n", valorString(p.ModelSlug))
	if p.WindowStartedAt != nil {
		fmt.Printf("Window start:   %s\n", p.WindowStartedAt.Format(time.RFC3339))
	}
	if p.ResetAt != nil {
		fmt.Printf("Reset at:       %s\n", p.ResetAt.Format(time.RFC3339))
	}
	if p.RemainingSeconds != nil {
		fmt.Printf("Remaining secs: %d\n", *p.RemainingSeconds)
	}
	if p.RemainingMessages != nil {
		fmt.Printf("Remaining msg:  %d\n", *p.RemainingMessages)
	}
	if p.RemainingTokens != nil {
		fmt.Printf("Remaining tok:  %d\n", *p.RemainingTokens)
	}
	if p.RemainingCredits != nil {
		fmt.Printf("Remaining cred: %.2f\n", *p.RemainingCredits)
	}
	if ev.RemainingRatio != nil {
		fmt.Printf("Ratio restante: %.2f\n", *ev.RemainingRatio)
	}
	if ev.Motivo != "" {
		fmt.Printf("Motivo:         %s\n", ev.Motivo)
	}
	fmt.Printf("Checked at:     %s\n", p.CheckedAt.Format(time.RFC3339))
}

func parseTimeFlag(cmd *cobra.Command, nombre string) (*time.Time, error) {
	v, err := cmd.Flags().GetString(nombre)
	if err != nil {
		return nil, err
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, fmt.Errorf("--%s debe ir en RFC3339", nombre)
	}
	return &t, nil
}

func nullableInt64Flag(cmd *cobra.Command, nombre string) (*int64, error) {
	v, err := cmd.Flags().GetInt64(nombre)
	if err != nil {
		return nil, err
	}
	if v < 0 {
		return nil, nil
	}
	return &v, nil
}

func nullableFloat64Flag(cmd *cobra.Command, nombre string) (*float64, error) {
	v, err := cmd.Flags().GetFloat64(nombre)
	if err != nil {
		return nil, err
	}
	if v < 0 {
		return nil, nil
	}
	return &v, nil
}

func valorString(v string) string {
	if strings.TrimSpace(v) == "" {
		return "—"
	}
	return v
}

func presupuestoResumenAgente(agente string) string {
	p, _, err := db.UltimoPresupuestoAgente(agente)
	if err != nil {
		return "—"
	}
	ev, err := db.EvaluarPresupuestoSesion(p)
	if err != nil {
		return "—"
	}
	resumen := ev.Estado
	if p.RemainingSeconds != nil {
		resumen += fmt.Sprintf(" (%ds)", *p.RemainingSeconds)
	}
	if ev.RemainingRatio != nil {
		resumen += fmt.Sprintf(" %.0f%%", *ev.RemainingRatio*100)
	}
	return resumen
}

func parseInt64Arg(v string) (*int64, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil, err
	}
	return &n, nil
}
