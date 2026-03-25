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
  - Solo hay consenso automático si todos los votantes habilitados votan acuerdo.
  - Además, se exigen al menos 2 votos de acuerdo y al menos 2 votos de acuerdo de agentes no autores.
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

		if ok, err := apiPost("/api/propuestas/"+codigo+"/accion", apiPropuestaAccionRequest{
			Accion:     "votar",
			Agente:     agente,
			Posicion:   posicion,
			Comentario: strings.TrimSpace(comentario),
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			var detalle apiPropuestaDetalleResponse
			if _, err := apiGet("/api/propuestas/"+codigo, &detalle); err != nil {
				return err
			}
			fmt.Printf("✓ Voto registrado: %s → %s [%s]\n", agente, codigo, posicion)
			if detalle.Propuesta != nil && detalle.Propuesta.Estado == db.PropuestaConsenso {
				fmt.Printf("🎉 ¡CONSENSO VÁLIDO! La propuesta %s ha sido aprobada automáticamente.\n", codigo)
				return nil
			}
			fmt.Printf("Estado votos: ✓%d  ✗%d  ～%d  ⏳%d\n",
				detalle.ConteoVotos.Acuerdo,
				detalle.ConteoVotos.Desacuerdo,
				detalle.ConteoVotos.Abstencion,
				detalle.ConteoVotos.Pendiente,
			)
			if detalle.ConteoVotos.Desacuerdo > 0 {
				fmt.Printf("⚠️  Hay %d voto(s) en desacuerdo — Alberto debe resolver antes de continuar.\n", detalle.ConteoVotos.Desacuerdo)
				fmt.Printf("   Usa: orquesta propuesta cerrar %s consenso --por alberto\n", codigo)
			}
			return nil
		}

		if err := ensureLocalDB(); err != nil {
			return err
		}
		p, err := db.GetPropuesta(codigo)
		if err != nil {
			return fmt.Errorf("propuesta '%s' no encontrada", codigo)
		}
		if p.Estado != db.PropuestaAbierta {
			// Permitir voto tardío si el agente no ha emitido posición definitiva
			votos, err := db.VotosDePropuesta(p.ID)
			if err != nil {
				return err
			}
			sinVoto := true
			for _, v := range votos {
				if strings.EqualFold(v.Agente, strings.TrimSpace(agente)) {
					if v.Posicion != db.VotoPendiente {
						return fmt.Errorf("la propuesta %s ya está cerrada (%s)", codigo, p.Estado)
					}
					sinVoto = false
					break
				}
			}
			_ = sinVoto // permitido: sin entrada o con pendiente
		}

		consenso, err := db.Votar(p.ID, agente, db.PosicionVoto(posicion), strings.TrimSpace(comentario))
		if err != nil {
			return err
		}

		fmt.Printf("✓ Voto registrado: %s → %s [%s]\n", agente, codigo, posicion)
		if consenso {
			fmt.Printf("🎉 ¡CONSENSO VÁLIDO! La propuesta %s ha sido aprobada automáticamente.\n", codigo)
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

func init() {
	votarCmd.Flags().String("agente", "", "Agente que emite el voto (requerido si no va en posición)")
	votarCmd.Flags().String("comentario", "", "Comentario del voto")
}
