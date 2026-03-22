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
	Use:   "votar <codigo-op> <acuerdo|desacuerdo|abstencion> [comentario...]",
	Short: "Vota una propuesta",
	Long: `Registra el voto de un agente en una propuesta OP-XXX.
Reglas de consenso:
  - Si todos los agentes votan acuerdo → consenso automático y propuesta cerrada.
  - Si cualquier agente vota desacuerdo → Alberto decide manualmente.

Ejemplos:
  orquesta votar OP-030 acuerdo --agente claude
  orquesta votar OP-030 desacuerdo --agente codex1 --comentario "requiere más análisis de rendimiento"
  orquesta votar OP-030 claude acuerdo`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		codigo := args[0]
		agente, err := resolverValorFlag(cmd, "agente")
		if err != nil {
			return err
		}
		posicion := ""
		comentario, _ := cmd.Flags().GetString("comentario")

		switch {
		case len(args) >= 3:
			if agente != "" && agente != strings.TrimSpace(args[1]) {
				return fmt.Errorf("el agente posicional (%s) no coincide con --agente (%s)", args[1], agente)
			}
			agente = strings.TrimSpace(args[1])
			posicion = args[2]
			if len(args) > 3 {
				comentario = strings.Join(args[3:], " ")
			}
		case len(args) == 2:
			posicion = args[1]
			if strings.TrimSpace(agente) == "" {
				return fmt.Errorf("--agente es obligatorio cuando no se pasa el agente como argumento posicional")
			}
		default:
			return fmt.Errorf("uso inválido de votar")
		}

		switch db.PosicionVoto(posicion) {
		case db.VotoAcuerdo, db.VotoDesacuerdo, db.VotoAbstencion:
		default:
			return fmt.Errorf("posición '%s' no válida", posicion)
		}

		if serverURL := activeServerURL(); serverURL != "" {
			res, err := submitServerVote(serverURL, codigo, agente, db.PosicionVoto(posicion), strings.TrimSpace(comentario))
			if err != nil {
				return err
			}
			renderVoteSummary(agente, codigo, posicion, res.ConsensoAlcanzado, res.Conteo["acuerdo"], res.Conteo["desacuerdo"], res.Conteo["abstencion"], res.Conteo["pendiente"])
			return nil
		}
		result, err := proposalService.VoteDetail(codigo, agente, db.PosicionVoto(posicion), strings.TrimSpace(comentario))
		if err != nil {
			return err
		}
		renderVoteSummary(agente, codigo, posicion, result.Consenso, result.Acuerdo, result.Desacuerdo, result.Abstencion, result.Pendiente)
		return nil
	},
}

func init() {
	votarCmd.Flags().String("agente", "", "Agente que emite el voto (requerido si no va en posición)")
	votarCmd.Flags().String("comentario", "", "Comentario del voto")
}

func renderVoteSummary(agente, codigo, posicion string, consenso bool, acuerdo, desacuerdo, abstencion, pendiente int) {
	fmt.Printf("✓ Voto registrado: %s → %s [%s]\n", agente, codigo, posicion)
	if consenso {
		fmt.Printf("🎉 ¡CONSENSO UNÁNIME! La propuesta %s ha sido aprobada automáticamente.\n", codigo)
		return
	}
	fmt.Printf("Estado votos: ✓%d  ✗%d  ～%d  ⏳%d\n", acuerdo, desacuerdo, abstencion, pendiente)
	if desacuerdo > 0 {
		fmt.Printf("⚠️  Hay %d voto(s) en desacuerdo — Alberto debe resolver antes de continuar.\n", desacuerdo)
		fmt.Printf("   Usa: orquesta propuesta cerrar %s consenso --por alberto\n", codigo)
	}
}
