package orquestamcp

import (
	"context"
	"strings"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	mcpDirectorStatsEstadoVivoEvidenceLimitV0 = 32

	mcpDirectorStatsEstadoVivoProcesoVivoV0      = "estado_vivo_proceso_vivo"
	mcpDirectorStatsEstadoVivoConflictoV0        = "estado_vivo_conflicto"
	mcpDirectorStatsEstadoVivoHuerfanoV0         = "estado_vivo_huerfano"
	mcpDirectorStatsEstadoVivoDesconocidoV0      = "estado_vivo_desconocido"
	mcpDirectorStatsEstadoVivoBloqueadoV0        = "estado_vivo_bloqueado"
	mcpDirectorStatsEstadoVivoTerminalReworkV0   = "estado_vivo_terminal_rework"
	mcpDirectorStatsEstadoVivoEntregadoParcialV0 = "estado_vivo_entregado_parcial"
	mcpDirectorStatsEstadoVivoSolicitadoV0       = "estado_vivo_solicitado"
	mcpDirectorStatsEstadoVivoLanzadoV0          = "estado_vivo_lanzado"
	mcpDirectorStatsEstadoVivoErrorV0            = "estado_vivo_error"

	mcpDirectorStatsEvidenceEstadoVivoErrorV0       = "evidence-ref-director-stats-estado-vivo-error"
	mcpDirectorStatsEvidenceEstadoVivoDesconocidoV0 = "evidence-ref-director-stats-estado-vivo-desconocido"
)

func (executor MCPDirectorStatsToolExecutorV0) applyEstadoVivoProjectionV0(
	ctx context.Context,
	input MCPDirectorStatsToolInputV0,
	stats *orquestacionnucleoapp.DirectorRunStatsV0,
) {
	if executor.EstadoVivoSource == nil || stats == nil {
		return
	}
	runRef := strings.TrimSpace(stats.RunRef)
	if runRef == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	evidencias, err := executor.EstadoVivoSource.ListarEvidenciasEstadoV0(ctx, orquestaestadovivo.FiltroEvidenciaEstadoV0{
		RunRef: runRef,
		Limit:  mcpDirectorStatsEstadoVivoEvidenceLimitV0,
	})
	if err != nil {
		markMCPDirectorStatsBlockedByEstadoVivoV0(
			stats,
			mcpDirectorStatsEstadoVivoErrorV0,
			"estado_vivo",
			"fuente de estado vivo no disponible",
			[]string{mcpDirectorStatsEvidenceEstadoVivoErrorV0},
		)
		return
	}
	projection := orquestaestadovivo.ConstruirProyeccionCicloVidaV0(
		evidencias,
		nowForEstadoVivoMCPAutoprogrammingV0(input.OccurredAt),
		mcpAutoprogrammingEstadoVivoHuerfanoV0,
	)
	node, ok := mcpDirectorStatsEstadoVivoNodeForRunV0(projection, runRef)
	if !ok {
		return
	}
	applyMCPDirectorStatsEstadoVivoNodeV0(stats, node)
}

func mcpDirectorStatsEstadoVivoNodeForRunV0(
	projection orquestaestadovivo.ProyeccionCicloVidaV0,
	runRef string,
) (orquestaestadovivo.NodoCicloVidaV0, bool) {
	runRef = strings.TrimSpace(runRef)
	for _, node := range projection.Nodos {
		if strings.TrimSpace(node.RunRef) == runRef {
			return node, true
		}
	}
	if len(projection.Nodos) == 1 && strings.TrimSpace(projection.Nodos[0].RunRef) == "" {
		node := projection.Nodos[0]
		node.RunRef = runRef
		return node, true
	}
	return orquestaestadovivo.NodoCicloVidaV0{}, false
}

func applyMCPDirectorStatsEstadoVivoNodeV0(
	stats *orquestacionnucleoapp.DirectorRunStatsV0,
	node orquestaestadovivo.NodoCicloVidaV0,
) {
	if stats == nil {
		return
	}
	evidenceRefs := evidenceRefsFromEstadoVivoNodeMCPDirectorStatsV0(node)
	switch node.Fase {
	case orquestaestadovivo.FaseTerminalAceptadoV0:
		stats.Status = "closed"
		stats.Closure = orquestacionnucleoapp.DirectorClosureStatsV0{
			Status:      orquestacionnucleoapp.DirectorClosureStatusClosedV0,
			Closed:      true,
			BlockerRefs: nil,
		}
	case orquestaestadovivo.FaseProcesoVivoV0:
		stats.Status = mcpDirectorStatsEstadoVivoProcesoVivoV0
		stats.Closure = orquestacionnucleoapp.DirectorClosureStatsV0{
			Status:      orquestacionnucleoapp.DirectorClosureStatusBlockedV0,
			Blocked:     true,
			BlockedBy:   []string{mcpDirectorStatsEstadoVivoProcesoVivoV0},
			BlockerRefs: compactStringsMCPV0(evidenceRefs),
		}
		capMCPDirectorStatsPercentBelowCompleteV0(stats)
	case orquestaestadovivo.FaseConflictoV0:
		markMCPDirectorStatsBlockedByEstadoVivoV0(
			stats,
			mcpDirectorStatsEstadoVivoConflictoV0,
			"estado_vivo.conflicto",
			"estado vivo incompatible: hay evidencia terminal y proceso vivo para el mismo run",
			evidenceRefs,
		)
	case orquestaestadovivo.FaseHuerfanoV0:
		markMCPDirectorStatsBlockedByEstadoVivoV0(
			stats,
			mcpDirectorStatsEstadoVivoHuerfanoV0,
			"estado_vivo.huerfano",
			"estado vivo huerfano: marcador sin proceso ni terminal reciente",
			evidenceRefs,
		)
	case orquestaestadovivo.FaseDesconocidoV0:
		markMCPDirectorStatsBlockedByEstadoVivoV0(
			stats,
			mcpDirectorStatsEstadoVivoDesconocidoV0,
			"estado_vivo.desconocido",
			"estado vivo desconocido: falta evidencia suficiente para cierre verde",
			compactStringsMCPV0(append([]string{mcpDirectorStatsEvidenceEstadoVivoDesconocidoV0}, evidenceRefs...)),
		)
	case orquestaestadovivo.FaseBloqueadoV0:
		markMCPDirectorStatsBlockedByEstadoVivoV0(
			stats,
			mcpDirectorStatsEstadoVivoBloqueadoV0,
			"estado_vivo.bloqueado",
			"estado vivo bloqueado",
			evidenceRefs,
		)
	case orquestaestadovivo.FaseTerminalReworkV0:
		markMCPDirectorStatsBlockedByEstadoVivoV0(
			stats,
			mcpDirectorStatsEstadoVivoTerminalReworkV0,
			"estado_vivo.terminal_rework",
			"estado vivo terminal requiere rework",
			evidenceRefs,
		)
	case orquestaestadovivo.FaseEntregadoParcialV0:
		markMCPDirectorStatsBlockedByEstadoVivoV0(
			stats,
			mcpDirectorStatsEstadoVivoEntregadoParcialV0,
			"estado_vivo.entregado_parcial",
			"estado vivo con entrega parcial pendiente de cierre",
			evidenceRefs,
		)
	case orquestaestadovivo.FaseSolicitadoV0:
		markMCPDirectorStatsBlockedByEstadoVivoV0(
			stats,
			mcpDirectorStatsEstadoVivoSolicitadoV0,
			"estado_vivo.solicitado",
			"estado vivo solicitado o en cola",
			evidenceRefs,
		)
	case orquestaestadovivo.FaseLanzadoV0:
		markMCPDirectorStatsBlockedByEstadoVivoV0(
			stats,
			mcpDirectorStatsEstadoVivoLanzadoV0,
			"estado_vivo.lanzado",
			"estado vivo lanzado sin terminal aceptado",
			evidenceRefs,
		)
	}
}

func markMCPDirectorStatsBlockedByEstadoVivoV0(
	stats *orquestacionnucleoapp.DirectorRunStatsV0,
	code string,
	field string,
	message string,
	evidenceRefs []string,
) {
	if stats == nil {
		return
	}
	code = strings.TrimSpace(code)
	stats.Status = code
	stats.Closure.Status = orquestacionnucleoapp.DirectorClosureStatusBlockedV0
	stats.Closure.Blocked = true
	stats.Closure.Ready = false
	stats.Closure.Closed = false
	stats.Closure.BlockedBy = compactStringsMCPV0(append(stats.Closure.BlockedBy, code))
	stats.Closure.BlockerRefs = compactStringsMCPV0(append(stats.Closure.BlockerRefs, evidenceRefs...))
	stats.Progress.Issues = append(stats.Progress.Issues, orquestacionnucleoapp.DirectorProgressIssueV0{
		Code:    code,
		Field:   strings.TrimSpace(field),
		Message: strings.TrimSpace(message),
	})
	capMCPDirectorStatsPercentBelowCompleteV0(stats)
}

func capMCPDirectorStatsPercentBelowCompleteV0(stats *orquestacionnucleoapp.DirectorRunStatsV0) {
	if stats == nil || stats.Progress.PercentComplete < 100 {
		return
	}
	if stats.Progress.TasksTotal <= 0 {
		stats.Progress.PercentComplete = 0
		return
	}
	stats.Progress.PercentComplete = 99
}

func evidenceRefsFromEstadoVivoNodeMCPDirectorStatsV0(
	node orquestaestadovivo.NodoCicloVidaV0,
) []string {
	out := make([]string, 0)
	for _, evidencia := range node.Evidencias {
		out = append(out, evidencia.EvidenceRefs...)
	}
	return compactStringsMCPV0(out)
}
