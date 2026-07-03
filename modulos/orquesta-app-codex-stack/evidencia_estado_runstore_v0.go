package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type EvidenciaEstadoRunStoreV0 struct {
	Store orquestacionnucleoapp.RunStorePortV0
}

var _ orquestaestadovivo.FuenteEvidenciaEstadoPortV0 = EvidenciaEstadoRunStoreV0{}

func (source EvidenciaEstadoRunStoreV0) ListarEvidenciasEstadoV0(
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
	if source.Store == nil || filtro.RunRef == "" {
		return nil, nil
	}
	run, err := source.Store.LoadRunV0(ctx, filtro.RunRef)
	if err != nil {
		if orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
			return nil, nil
		}
		return nil, err
	}
	evidencia := normalizarEvidenciaEstadoV0(orquestaestadovivo.EvidenciaEstadoV0{
		RunRef:       run.RunID,
		Fuente:       evidenciaEstadoFuenteRunStoreV0,
		Estado:       strings.TrimSpace(string(run.Status)),
		EvidenceRefs: evidenciaEstadoRunRefsV0(run),
	})
	if !evidenciaEstadoPasaFiltroV0(evidencia, filtro) {
		return nil, nil
	}
	return limitarEvidenciasEstadoV0([]orquestaestadovivo.EvidenciaEstadoV0{evidencia}, filtro.Limit), nil
}
