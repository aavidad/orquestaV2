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
	"orquesta/db"
)

var votarCmd = &cobra.Command{
	Use:   "votar <codigo-op> <agente> <acuerdo|desacuerdo|abstencion> [comentario...]",
	Short: "Vota una propuesta",
	Long: `Registra el voto de un agente en una propuesta OP-XXX.
Reglas de consenso:
  - Si todos los agentes votan acuerdo → consenso automático y propuesta cerrada.
  - Si cualquier agente vota desacuerdo → Alberto decide manualmente.

Ejemplos:
  orquesta votar OP-030 claude acuerdo
  orquesta votar OP-030 codex1 desacuerdo "requiere más análisis de rendimiento"`,
	Args: cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		codigo := args[0]
		agente := args[1]
		posicion := args[2]
		comentario := ""
		if len(args) > 3 {
			comentario = strings.Join(args[3:], " ")
		}

		p, err := db.GetPropuesta(codigo)
		if err != nil {
			return fmt.Errorf("propuesta '%s' no encontrada", codigo)
		}
		if p.Estado != db.PropuestaAbierta {
			return fmt.Errorf("la propuesta %s ya está cerrada (%s)", codigo, p.Estado)
		}

		consenso, err := db.Votar(p.ID, agente, db.PosicionVoto(posicion), comentario)
		if err != nil {
			return err
		}

		fmt.Printf("✓ Voto registrado: %s → %s [%s]\n", agente, codigo, posicion)
		if consenso {
			fmt.Printf("🎉 ¡CONSENSO UNÁNIME! La propuesta %s ha sido aprobada automáticamente.\n", codigo)
		} else {
			ac, des, abs, pend, _ := db.ContarVotos(p.ID)
			fmt.Printf("Estado votos: ✓%d  ✗%d  ～%d  ⏳%d\n", ac, des, abs, pend)
			if des > 0 {
				fmt.Printf("⚠️  Hay %d voto(s) en desacuerdo — Alberto debe resolver antes de continuar.\n", des)
				fmt.Printf("   Usa: orquesta propuesta cerrar %s consenso --por alberto\n", codigo)
			}
		}
		return nil
	},
}
