package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

type EvidenciaEstadoMarkerV0 struct {
	Store orquestagoal.GoalWorkRunMarkerStorePortV0
}

var _ orquestaestadovivo.FuenteEvidenciaEstadoPortV0 = EvidenciaEstadoMarkerV0{}

func (source EvidenciaEstadoMarkerV0) ListarEvidenciasEstadoV0(
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
	markers, err := source.listarMarkersV0(ctx, filtro)
	if err != nil {
		return nil, err
	}
	evidencias := make([]orquestaestadovivo.EvidenciaEstadoV0, 0, len(markers))
	for _, marker := range markers {
		marker, err = orquestagoal.NewGoalWorkRunMarkerV0(marker)
		if err != nil {
			return nil, err
		}
		evidencia := normalizarEvidenciaEstadoV0(orquestaestadovivo.EvidenciaEstadoV0{
			RunRef:          marker.RunRef,
			GoalRef:         marker.GoalRef,
			ExternalGoalRef: marker.ExternalGoalRef,
			Fuente:          evidenciaEstadoFuenteRunMarkerV0,
			Estado:          strings.TrimSpace(marker.Status),
			EvidenceRefs:    evidenciaEstadoGoalMarkerRefsV0(marker),
		})
		if !evidenciaEstadoPasaFiltroV0(evidencia, filtro) {
			continue
		}
		evidencias = append(evidencias, evidencia)
	}
	return limitarEvidenciasEstadoV0(evidencias, filtro.Limit), nil
}

func (source EvidenciaEstadoMarkerV0) listarMarkersV0(
	ctx context.Context,
	filtro orquestaestadovivo.FiltroEvidenciaEstadoV0,
) ([]orquestagoal.GoalWorkRunMarkerV0, error) {
	if lister, ok := source.Store.(orquestagoal.GoalWorkRunMarkerListPortV0); ok && lister != nil {
		request := orquestagoal.GoalWorkRunMarkerListRequestV0{MaxItems: filtro.Limit}
		if filtro.RunRef != "" {
			request.RunRefs = []string{filtro.RunRef}
		}
		return lister.ListGoalWorkRunMarkersV0(ctx, request)
	}
	if filtro.RunRef == "" {
		return nil, nil
	}
	marker, err := source.Store.LoadGoalWorkRunMarkerV0(ctx, filtro.RunRef)
	if err != nil {
		return nil, nil
	}
	return []orquestagoal.GoalWorkRunMarkerV0{marker}, nil
}
