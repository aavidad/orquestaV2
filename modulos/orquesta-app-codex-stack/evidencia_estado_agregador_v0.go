package orquestaappcodexstack

import (
	"context"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
)

type EvidenciaEstadoAgregadorV0 struct {
	Fuentes []orquestaestadovivo.FuenteEvidenciaEstadoPortV0
}

var _ orquestaestadovivo.FuenteEvidenciaEstadoPortV0 = EvidenciaEstadoAgregadorV0{}

func (agregador EvidenciaEstadoAgregadorV0) ListarEvidenciasEstadoV0(
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
	evidencias := []orquestaestadovivo.EvidenciaEstadoV0{}
	for _, fuente := range agregador.Fuentes {
		if fuenteEvidenciaEstadoNilV0(fuente) {
			continue
		}
		parciales, err := fuente.ListarEvidenciasEstadoV0(ctx, filtro)
		if err != nil {
			return nil, err
		}
		for _, parcial := range parciales {
			parcial = normalizarEvidenciaEstadoV0(parcial)
			if !evidenciaEstadoPasaFiltroV0(parcial, filtro) {
				continue
			}
			evidencias = append(evidencias, parcial)
			if filtro.Limit > 0 && len(evidencias) >= filtro.Limit {
				return limitarEvidenciasEstadoV0(evidencias, filtro.Limit), nil
			}
		}
	}
	return limitarEvidenciasEstadoV0(evidencias, filtro.Limit), nil
}
