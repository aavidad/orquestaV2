package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var agenteScoreCmd = &cobra.Command{
	Use:   "score",
	Short: "Gestiona scoring dinámico de agentes locales por materia",
}

var agenteScoreListCmd = &cobra.Command{
	Use:   "listar [agente]",
	Short: "Lista la matriz de scores por agente y materia",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var agenteRef *string
		if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
			agente := strings.TrimSpace(args[0])
			if err := db.GarantizarScoresLocalesAgente(agente, ""); err != nil {
				return err
			}
			agenteRef = &agente
		}
		items, err := db.ListarAgenteScoresLocales(agenteRef, nil)
		if err != nil {
			return err
		}
		renderAgenteScoresLocales(items)
		return nil
	},
}

var agenteScoreBenchmarkCmd = &cobra.Command{
	Use:   "benchmark <agente>",
	Short: "Registra o ajusta el score base inicial de una materia",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente := strings.TrimSpace(args[0])
		materia, _ := cmd.Flags().GetString("materia")
		score, _ := cmd.Flags().GetFloat64("score")
		detalle, _ := cmd.Flags().GetString("detalle")
		conector, _ := cmd.Flags().GetString("conector")

		item, err := db.RegistrarBenchmarkScoreAgenteLocal(agente, strings.TrimSpace(conector), strings.TrimSpace(materia), score, detalle)
		if err != nil {
			return err
		}
		if item == nil {
			return fmt.Errorf("no se pudo registrar benchmark")
		}
		fmt.Printf("✓ Benchmark %s/%s = %.1f total=%.1f confianza=%.2f\n",
			item.Agente, item.Materia, item.ScoreBase, item.ScoreTotal, item.Confianza)
		return nil
	},
}

var agenteScoreObserveCmd = &cobra.Command{
	Use:   "observar <agente>",
	Short: "Registra una observación mutable a partir de trabajo real",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente := strings.TrimSpace(args[0])
		materia, _ := cmd.Flags().GetString("materia")
		score, _ := cmd.Flags().GetFloat64("score")
		detalle, _ := cmd.Flags().GetString("detalle")
		conector, _ := cmd.Flags().GetString("conector")
		fallo, _ := cmd.Flags().GetBool("fallo")

		item, err := db.RegistrarObservacionScoreAgenteLocal(agente, strings.TrimSpace(conector), strings.TrimSpace(materia), score, !fallo, detalle)
		if err != nil {
			return err
		}
		if item == nil {
			return fmt.Errorf("no se pudo registrar observación")
		}
		fmt.Printf("✓ Observación %s/%s = %.1f total=%.1f confianza=%.2f muestras=%d\n",
			item.Agente, item.Materia, score, item.ScoreTotal, item.Confianza, item.Muestras)
		return nil
	},
}

func init() {
	agenteScoreBenchmarkCmd.Flags().String("materia", "codigo", "Materia a puntuar")
	agenteScoreBenchmarkCmd.Flags().Float64("score", 5.0, "Score base entre 0 y 10")
	agenteScoreBenchmarkCmd.Flags().String("detalle", "", "Detalle del benchmark")
	agenteScoreBenchmarkCmd.Flags().String("conector", "", "Conector explícito si no quieres usar el default del agente")
	_ = agenteScoreBenchmarkCmd.MarkFlagRequired("materia")
	_ = agenteScoreBenchmarkCmd.MarkFlagRequired("score")

	agenteScoreObserveCmd.Flags().String("materia", "codigo", "Materia observada")
	agenteScoreObserveCmd.Flags().Float64("score", 5.0, "Score observado entre 0 y 10")
	agenteScoreObserveCmd.Flags().String("detalle", "", "Detalle de la observación")
	agenteScoreObserveCmd.Flags().String("conector", "", "Conector explícito si no quieres usar el default del agente")
	agenteScoreObserveCmd.Flags().Bool("fallo", false, "Marca la observación como fallo en vez de éxito")
	_ = agenteScoreObserveCmd.MarkFlagRequired("materia")
	_ = agenteScoreObserveCmd.MarkFlagRequired("score")

	agenteScoreCmd.AddCommand(agenteScoreListCmd)
	agenteScoreCmd.AddCommand(agenteScoreBenchmarkCmd)
	agenteScoreCmd.AddCommand(agenteScoreObserveCmd)
	agenteCmd.AddCommand(agenteScoreCmd)
}

func renderAgenteScoresLocales(items []*db.AgenteScoreLocal) {
	if len(items) == 0 {
		fmt.Println("Sin scores locales registrados.")
		return
	}
	type fila struct {
		agente      string
		conector    string
		materias    map[string]float64
		confianza   float64
		totalMuestr int
	}
	porClave := map[string]*fila{}
	for _, item := range items {
		if item == nil {
			continue
		}
		clave := strings.TrimSpace(item.Agente) + "\x00" + strings.TrimSpace(item.ConectorSlug)
		actual := porClave[clave]
		if actual == nil {
			actual = &fila{
				agente:   strings.TrimSpace(item.Agente),
				conector: strings.TrimSpace(item.ConectorSlug),
				materias: map[string]float64{},
			}
			porClave[clave] = actual
		}
		actual.materias[db.NormalizarMateriaScoreAgente(item.Materia)] = item.ScoreTotal
		actual.confianza += item.Confianza
		actual.totalMuestr += item.Muestras
	}
	claves := make([]string, 0, len(porClave))
	for clave := range porClave {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	fmt.Printf("%-16s %-14s %-4s %-4s %-4s %-4s %-4s %-4s %-4s %-4s %-4s %-4s %-4s %-4s %-4s %-4s %-4s %-5s %s\n",
		"AGENTE", "CONECTOR", "COD", "DOC", "ORQ", "BRN", "ANA", "REV", "UI", "TST", "ARQ", "INF", "SEG", "INT", "DEP", "DAT", "PRD", "CONF", "MUESTRAS")
	for _, clave := range claves {
		item := porClave[clave]
		if item == nil {
			continue
		}
		confMedia := 0.0
		if len(item.materias) > 0 {
			confMedia = item.confianza / float64(len(item.materias))
		}
		fmt.Printf("%-16s %-14s %-4.1f %-4.1f %-4.1f %-4.1f %-4.1f %-4.1f %-4.1f %-4.1f %-4.1f %-4.1f %-4.1f %-4.1f %-4.1f %-4.1f %-4.1f %-5.2f %d\n",
			truncar(item.agente, 16),
			truncar(item.conector, 14),
			item.materias["codigo"],
			item.materias["documentacion"],
			item.materias["orquestacion"],
			item.materias["brainstorming"],
			item.materias["analisis"],
			item.materias["revision"],
			item.materias["frontend"],
			item.materias["testing"],
			item.materias["arquitectura"],
			item.materias["infraestructura"],
			item.materias["seguridad"],
			item.materias["integraciones"],
			item.materias["depuracion"],
			item.materias["datos"],
			item.materias["producto"],
			confMedia,
			item.totalMuestr,
		)
	}
}
