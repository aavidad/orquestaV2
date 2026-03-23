/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"database/sql"
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
		sesionID, err := resolverSesionPresupuesto(cmd)
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

		id, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
			SesionID:          sesionID,
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
		})
		if err != nil {
			return err
		}

		p, err := db.UltimoPresupuestoSesion(sesionID)
		if err != nil {
			return err
		}
		ev, err := db.EvaluarPresupuestoSesion(p)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Presupuesto registrado (id: %d) para sesión #%d\n", id, sesionID)
		fmt.Printf("  Estado: %s", ev.Estado)
		if ev.DebeHandoff {
			fmt.Printf("  → handoff preventivo")
		}
		fmt.Println()
		if ev.Motivo != "" {
			fmt.Printf("  Motivo: %s\n", ev.Motivo)
		}
		return nil
	},
}

var sesionPresupuestoVerCmd = &cobra.Command{
	Use:   "ver",
	Short: "Muestra el último presupuesto de una sesión",
	RunE: func(cmd *cobra.Command, args []string) error {
		sesionID, err := resolverSesionPresupuesto(cmd)
		if err != nil {
			return err
		}
		p, err := db.UltimoPresupuestoSesion(sesionID)
		if err == sql.ErrNoRows {
			return fmt.Errorf("la sesión #%d no tiene presupuestos registrados", sesionID)
		}
		if err != nil {
			return err
		}
		ev, err := db.EvaluarPresupuestoSesion(p)
		if err != nil {
			return err
		}

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
		return nil
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

func resolverSesionPresupuesto(cmd *cobra.Command) (int64, error) {
	sesionID, err := cmd.Flags().GetInt64("sesion")
	if err != nil {
		return 0, err
	}
	if sesionID > 0 {
		return sesionID, nil
	}
	agente, err := cmd.Flags().GetString("agente")
	if err != nil {
		return 0, err
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return 0, fmt.Errorf("debe indicar --sesion o --agente")
	}
	sesion, err := db.SesionActivaDeAgente(agente)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("el agente %s no tiene sesión activa", agente)
	}
	if err != nil {
		return 0, err
	}
	return sesion.ID, nil
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
