package application

import "context"

func (orchestrator *Orchestrator) obtenerCandidatosCapacidad(
	ctx context.Context, excluirLanzamiento bool,
) ([]AgentCapacityPlacementCandidate, error) {
	if excluirLanzamiento || len(orchestrator.capacitySources) == 0 {
		return nil, nil
	}
	lote, cancelar := context.WithTimeout(ctx, orchestrator.capacityObservationWait)
	defer cancelar()
	ahora := orchestrator.clock.Now()
	candidatos := make([]AgentCapacityPlacementCandidate, 0, len(orchestrator.capacitySources))
	for _, fuente := range orchestrator.capacitySources {
		cuota, encontrada, err := orchestrator.state.CurrentAgentQuotaObservation(lote, fuente.PlacementRef)
		if err != nil || !encontrada {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			continue
		}
		razon, err := DecideAgentQuotaGate(ahora, &cuota)
		if err != nil || razon != AgentCapacityAdmissionAvailable {
			continue
		}
		observacion, observarErr := fuente.Observer.ObserveCapacity(lote, fuente.SourceRef, fuente.PoolRef)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if observarErr != nil {
			continue
		}
		entrega, err := NuevaEntregaObservacionCapacidad(observacion)
		if err != nil || observacion.SourceRef != fuente.SourceRef || observacion.PoolRef != fuente.PoolRef {
			continue
		}
		candidatos = append(candidatos, AgentCapacityPlacementCandidate{
			PlacementRef: fuente.PlacementRef, Physical: entrega,
			Quota: AgentPlacementObservationPresentation{ObservationRef: cuota.Ref, ObservationRevision: cuota.Revision},
		})
	}
	return OrdenarCandidatosColocacion(candidatos)
}
