package cmd

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"orquesta/agentesapp"
	"orquesta/capacidadapp"
	"orquesta/db"
	"orquesta/runtimeagente"
)

var capacidadAgentResolverRowsTimeout = 500 * time.Millisecond
var capacidadAgentResolverLocalMinFitness = 6.0

type proveedorScoreAgentePipeline interface {
	GarantizarScoresLocalesAgente(agente, conector string) error
	FitnessAgenteLocal(agente, conector string, pesos map[string]float64) (float64, error)
}

type proveedorScoreAgentePipelineDB struct{}

func (proveedorScoreAgentePipelineDB) GarantizarScoresLocalesAgente(agente, conector string) error {
	return db.GarantizarScoresLocalesAgente(agente, conector)
}

func (proveedorScoreAgentePipelineDB) FitnessAgenteLocal(agente, conector string, pesos map[string]float64) (float64, error) {
	return db.FitnessAgenteLocal(agente, conector, pesos)
}

type resolvedorAgentePipelineOperativo struct {
	rowsProvider interface {
		BuildPanelRows() ([]agentesapp.Row, error)
	}
	scoreProvider proveedorScoreAgentePipeline
}

func (r resolvedorAgentePipelineOperativo) ResolverAgentePipeline(entrada capacidadapp.EntradaResolverAgentePipeline) (string, error) {
	seleccion, err := r.ResolverSeleccionAgentePipeline(entrada)
	if err != nil || seleccion == nil {
		return "", err
	}
	return strings.TrimSpace(seleccion.Agente), nil
}

func (r resolvedorAgentePipelineOperativo) ResolverSeleccionAgentePipeline(entrada capacidadapp.EntradaResolverAgentePipeline) (*capacidadapp.SeleccionAgentePipeline, error) {
	if r.rowsProvider == nil {
		return nil, nil
	}
	rows, err := r.buildRows()
	if err != nil {
		if errors.Is(err, errStatusFetchTimeout) {
			return nil, nil
		}
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	var (
		disponibles []string
		trabajando  []string
	)
	for _, row := range rows {
		if row.Agente == nil || !row.Agente.Habilitado {
			continue
		}
		nombre := strings.TrimSpace(row.Agente.Nombre)
		if nombre == "" || !agenteCompatibleConCarril(nombre, entrada.Carril) {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(row.EstadoOperativo)) {
		case "disponible":
			disponibles = append(disponibles, nombre)
		case "trabajando":
			trabajando = append(trabajando, nombre)
		}
	}
	localIntentado := false
	if agente, fitness, ok, err := r.resolverAgenteLocalConScore(disponibles, entrada); err != nil {
		return nil, err
	} else if strings.TrimSpace(agente) != "" {
		return &capacidadapp.SeleccionAgentePipeline{
			Agente:     agente,
			Estrategia: "local_first",
			Motivo:     motivoSeleccionLocalFirst(agente, fitness, entrada),
		}, nil
	} else {
		localIntentado = ok
	}
	if len(disponibles) > 0 {
		agente := agentePreferidoPorCarrilYFase(disponibles, entrada)
		return &capacidadapp.SeleccionAgentePipeline{
			Agente:     agente,
			Estrategia: "fallback_carril",
			Motivo:     motivoSeleccionFallbackCarril(agente, entrada, localIntentado),
		}, nil
	}
	if carrilPipelineExigeAgenteLibre(entrada.Carril) {
		return nil, nil
	}
	if agente, fitness, _, err := r.resolverAgenteLocalConScore(trabajando, entrada); err != nil {
		return nil, err
	} else if strings.TrimSpace(agente) != "" {
		return &capacidadapp.SeleccionAgentePipeline{
			Agente:     agente,
			Estrategia: "local_first_reuse",
			Motivo:     motivoSeleccionLocalFirst(agente, fitness, entrada),
		}, nil
	}
	if len(trabajando) > 0 {
		return &capacidadapp.SeleccionAgentePipeline{
			Agente:     agentePreferidoPorCarrilYFase(trabajando, entrada),
			Estrategia: "fallback_carril_reuse",
			Motivo:     "no habia local con fitness suficiente y el carril permite reutilizar agente trabajando",
		}, nil
	}
	return nil, nil
}

func (r resolvedorAgentePipelineOperativo) resolverAgenteLocalConScore(candidatos []string, entrada capacidadapp.EntradaResolverAgentePipeline) (string, float64, bool, error) {
	if r.scoreProvider == nil || len(candidatos) == 0 {
		return "", 0, false, nil
	}
	if !carrilPipelinePrefiereLocalConScore(entrada.Carril) {
		return "", 0, false, nil
	}
	pesos := db.PesosScoreMateriaDesdePipeline(strings.TrimSpace(entrada.Fase), strings.TrimSpace(entrada.PerfilTarea), strings.TrimSpace(entrada.Carril))
	type candidatoScore struct {
		agente  string
		fitness float64
	}
	var scored []candidatoScore
	for _, candidato := range candidatos {
		agente := strings.TrimSpace(candidato)
		if agente == "" {
			continue
		}
		conector := strings.TrimSpace(runtimeagente.ConectorPorDefectoAgente(agente))
		if !db.AgenteEsLocalPuntuable(agente, conector) {
			continue
		}
		if err := r.scoreProvider.GarantizarScoresLocalesAgente(agente, conector); err != nil {
			return "", 0, false, err
		}
		fitness, err := r.scoreProvider.FitnessAgenteLocal(agente, conector, pesos)
		if err != nil {
			return "", 0, false, err
		}
		scored = append(scored, candidatoScore{agente: agente, fitness: fitness})
	}
	if len(scored) == 0 {
		return "", 0, false, nil
	}
	slices.SortFunc(scored, func(a, b candidatoScore) int {
		if math.Abs(a.fitness-b.fitness) > 0.001 {
			if a.fitness > b.fitness {
				return -1
			}
			return 1
		}
		return strings.Compare(strings.ToLower(strings.TrimSpace(a.agente)), strings.ToLower(strings.TrimSpace(b.agente)))
	})
	if scored[0].fitness < capacidadAgentResolverLocalMinFitness {
		return "", scored[0].fitness, true, nil
	}
	return scored[0].agente, scored[0].fitness, true, nil
}

func carrilPipelinePrefiereLocalConScore(carril string) bool {
	switch strings.ToLower(strings.TrimSpace(carril)) {
	case "microprogramacion_local":
		return true
	default:
		return false
	}
}

func motivoSeleccionLocalFirst(agente string, fitness float64, entrada capacidadapp.EntradaResolverAgentePipeline) string {
	return "local-first: agente local con fitness suficiente para el carril " + strings.TrimSpace(entrada.Carril) + " (" + strings.TrimSpace(agente) + ", fitness=" + trimFloat1(fitness) + ")"
}

func motivoSeleccionFallbackCarril(agente string, entrada capacidadapp.EntradaResolverAgentePipeline, intentoLocal bool) string {
	if intentoLocal {
		return "fallback de carril: no habia local con fitness suficiente; se usa el agente preferido del carril " + strings.TrimSpace(entrada.Carril) + " (" + strings.TrimSpace(agente) + ")"
	}
	return "seleccion por carril: no aplica local-first para " + strings.TrimSpace(entrada.Carril) + " o no habia candidatos locales puntuables"
}

func trimFloat1(v float64) string {
	return strings.TrimSpace(strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", v), "0"), "."))
}

func (r resolvedorAgentePipelineOperativo) buildRows() ([]agentesapp.Row, error) {
	if r.rowsProvider == nil {
		return nil, nil
	}
	if capacidadAgentResolverRowsTimeout <= 0 {
		return r.rowsProvider.BuildPanelRows()
	}
	type result struct {
		rows []agentesapp.Row
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		rows, err := r.rowsProvider.BuildPanelRows()
		ch <- result{rows: rows, err: err}
	}()
	select {
	case res := <-ch:
		return res.rows, res.err
	case <-time.After(capacidadAgentResolverRowsTimeout):
		return nil, errStatusFetchTimeout
	}
}

func carrilPipelineExigeAgenteLibre(carril string) bool {
	switch strings.ToLower(strings.TrimSpace(carril)) {
	case "premium_worktree", "revision_diff":
		return true
	default:
		return false
	}
}

func agentePreferidoPorCarrilYFase(candidatos []string, entrada capacidadapp.EntradaResolverAgentePipeline) string {
	if len(candidatos) == 0 {
		return ""
	}
	preferencias := preferenciasAgentePorCarrilYFase(entrada)
	if len(preferencias) == 0 {
		return ordenarCandidatosPipelinePorCosto(candidatos)[0]
	}
	for _, pref := range preferencias {
		matches := make([]string, 0, len(candidatos))
		for _, candidato := range candidatos {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(candidato)), pref) {
				matches = append(matches, candidato)
			}
		}
		if len(matches) > 0 {
			return ordenarCandidatosPipelinePorCosto(matches)[0]
		}
	}
	return ordenarCandidatosPipelinePorCosto(candidatos)[0]
}

func ordenarCandidatosPipelinePorCosto(candidatos []string) []string {
	ordenados := append([]string(nil), candidatos...)
	slices.SortFunc(ordenados, func(a, b string) int {
		if ai, bi := agentePipelineCostTier(a), agentePipelineCostTier(b); ai != bi {
			return ai - bi
		}
		return strings.Compare(strings.ToLower(strings.TrimSpace(a)), strings.ToLower(strings.TrimSpace(b)))
	})
	return ordenados
}

func preferenciasAgentePorCarrilYFase(entrada capacidadapp.EntradaResolverAgentePipeline) []string {
	carril := strings.ToLower(strings.TrimSpace(entrada.Carril))
	switch carril {
	case "revision_diff":
		return []string{"codex"}
	case "premium_worktree":
		return []string{"codex"}
	case "microprogramacion_local":
		return []string{"codex"}
	default:
		return nil
	}
}

func agenteCompatibleConCarril(nombre, carril string) bool {
	nombre = strings.ToLower(strings.TrimSpace(nombre))
	switch strings.ToLower(strings.TrimSpace(carril)) {
	case "premium_worktree", "revision_diff":
		return strings.HasPrefix(nombre, "codex")
	case "microprogramacion_local":
		if strings.HasPrefix(nombre, "codex") {
			return true
		}
		return runtimeagente.EsConectorFamiliaOllama(runtimeagente.ConectorPorDefectoAgente(nombre), "")
	case "determinista_app":
		return false
	default:
		return true
	}
}

func agentePipelineCostTier(nombre string) int {
	nombre = strings.ToLower(strings.TrimSpace(nombre))
	if strings.HasPrefix(nombre, "codexpg") {
		return 1
	}
	return 0
}
