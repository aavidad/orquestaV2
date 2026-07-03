package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

type EvidenciaEstadoGoalStateV0 struct {
	Store orquestagoal.GoalWorkStateStorePortV0
}

var _ orquestaestadovivo.FuenteEvidenciaEstadoPortV0 = EvidenciaEstadoGoalStateV0{}

func (source EvidenciaEstadoGoalStateV0) ListarEvidenciasEstadoV0(
	ctx context.Context,
	filtro orquestaestadovivo.FiltroEvidenciaEstadoV0,
) ([]orquestaestadovivo.EvidenciaEstadoV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	filtro = normalizarFiltroEvidenciaEstadoV0(filtro)
	if source.Store == nil {
		return nil, nil
	}
	states, err := source.listarGoalStatesV0(ctx, filtro)
	if err != nil {
		return nil, err
	}
	evidencias := make([]orquestaestadovivo.EvidenciaEstadoV0, 0, len(states))
	for _, state := range states {
		state, err = orquestagoal.NewGoalWorkStateV0(state)
		if err != nil {
			return nil, err
		}
		evidencia := normalizarEvidenciaEstadoV0(orquestaestadovivo.EvidenciaEstadoV0{
			RunRef:          state.RunRef,
			GoalRef:         state.GoalRef,
			ExternalGoalRef: state.ExternalGoalRef,
			Fuente:          evidenciaEstadoFuenteGoalStateV0,
			Estado:          strings.TrimSpace(state.Status),
			EvidenceRefs:    evidenciaEstadoGoalStateRefsV0(state),
		})
		if !evidenciaEstadoPasaFiltroV0(evidencia, filtro) {
			continue
		}
		evidencias = append(evidencias, evidencia)
	}
	return limitarEvidenciasEstadoV0(evidencias, filtro.Limit), nil
}

func (source EvidenciaEstadoGoalStateV0) listarGoalStatesV0(
	ctx context.Context,
	filtro orquestaestadovivo.FiltroEvidenciaEstadoV0,
) ([]orquestagoal.GoalWorkStateV0, error) {
	if lister, ok := source.Store.(orquestagoal.GoalWorkStateListPortV0); ok && lister != nil {
		request := orquestagoal.GoalWorkStateListRequestV0{MaxItems: filtro.Limit}
		if filtro.RunRef != "" {
			request.RunRefs = []string{filtro.RunRef}
		}
		return lister.ListGoalWorkStatesV0(ctx, request)
	}
	if filtro.RunRef == "" {
		return nil, nil
	}
	state, err := source.Store.LoadGoalWorkStateV0(ctx, filtro.RunRef)
	if err != nil {
		return nil, nil
	}
	return []orquestagoal.GoalWorkStateV0{state}, nil
}
