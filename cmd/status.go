/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Muestra el estado global del proyecto",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := loadStatusSummary()
		if err != nil {
			return err
		}
		renderStatusSummary(ctx)
		return nil
	},
}

func statusModoRecuperacionLocalExplicito() bool {
	return localRecoveryEnabled()
}

func statusErrorServerFirst() error {
	return fmt.Errorf("este comando exige servidor/daemon de Orquesta; usa --local solo en recuperacion explicita o exporta ORQUESTA_FORCE_LOCAL_DB=1")
}

func truncar(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

func mapToValues(entries map[string]string) url.Values {
	values := url.Values{}
	for key, value := range entries {
		if value != "" {
			values.Set(key, value)
		}
	}
	return values
}
