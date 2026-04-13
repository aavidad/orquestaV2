package cmd

import (
	"slices"
	"strings"

	"orquesta/agentesapp"
	"orquesta/capacidadapp"
)

type resolvedorAgentePipelineOperativo struct {
	rowsProvider interface {
		BuildPanelRows() ([]agentesapp.Row, error)
	}
}

func (r resolvedorAgentePipelineOperativo) ResolverAgentePipeline(entrada capacidadapp.EntradaResolverAgentePipeline) (string, error) {
	if r.rowsProvider == nil {
		return "", nil
	}
	rows, err := r.rowsProvider.BuildPanelRows()
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", nil
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
	if len(disponibles) > 0 {
		return agentePreferidoPorCarrilYFase(disponibles, entrada), nil
	}
	if len(trabajando) > 0 {
		return agentePreferidoPorCarrilYFase(trabajando, entrada), nil
	}
	return "", nil
}

func agentePreferidoPorCarrilYFase(candidatos []string, entrada capacidadapp.EntradaResolverAgentePipeline) string {
	if len(candidatos) == 0 {
		return ""
	}
	preferencias := preferenciasAgentePorCarrilYFase(entrada)
	if len(preferencias) == 0 {
		return candidatos[0]
	}
	for _, pref := range preferencias {
		for _, candidato := range candidatos {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(candidato)), pref) {
				return candidato
			}
		}
	}
	ordenados := append([]string(nil), candidatos...)
	slices.SortFunc(ordenados, func(a, b string) int {
		return strings.Compare(strings.ToLower(strings.TrimSpace(a)), strings.ToLower(strings.TrimSpace(b)))
	})
	return ordenados[0]
}

func preferenciasAgentePorCarrilYFase(entrada capacidadapp.EntradaResolverAgentePipeline) []string {
	carril := strings.ToLower(strings.TrimSpace(entrada.Carril))
	fase := strings.ToLower(strings.TrimSpace(entrada.Fase))
	switch carril {
	case "revision_diff":
		return []string{"claude", "gemini", "codex"}
	case "premium_worktree":
		switch fase {
		case "especificacion":
			return []string{"gemini", "claude", "codex"}
		case "implementacion", "correccion":
			return []string{"codex", "claude", "gemini"}
		default:
			return []string{"codex", "claude", "gemini"}
		}
	case "microprogramacion_local":
		return []string{"gemma", "ollama"}
	default:
		return nil
	}
}

func agenteCompatibleConCarril(nombre, carril string) bool {
	nombre = strings.ToLower(strings.TrimSpace(nombre))
	switch strings.ToLower(strings.TrimSpace(carril)) {
	case "premium_worktree", "revision_diff":
		return strings.HasPrefix(nombre, "codex") || strings.HasPrefix(nombre, "claude") || strings.HasPrefix(nombre, "gemini")
	case "microprogramacion_local":
		return strings.HasPrefix(nombre, "gemma") || strings.HasPrefix(nombre, "ollama")
	case "determinista_app":
		return false
	default:
		return true
	}
}
