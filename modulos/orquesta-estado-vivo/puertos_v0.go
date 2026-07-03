package orquestaestadovivo

import "context"

type FiltroEvidenciaEstadoV0 struct {
	RunRef  string
	GoalRef string
	Limit   int
}

type FuenteEvidenciaEstadoPortV0 interface {
	ListarEvidenciasEstadoV0(
		ctx context.Context,
		filtro FiltroEvidenciaEstadoV0,
	) ([]EvidenciaEstadoV0, error)
}
