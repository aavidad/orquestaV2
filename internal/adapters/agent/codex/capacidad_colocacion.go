package codex

import (
	"orquesta/internal/application"
	"orquesta/internal/ports"
)

const referenciaFuenteCapacidadCodex application.AgentCapacitySourceRef = "capacity-source:codex:process"

func (adaptador *Adapter) DescribirCapacidadColocaciones() ([]application.DescriptorCapacidadColocacionAgente, error) {
	if adaptador == nil {
		return nil, &Error{Code: CodeStateInvalid}
	}
	if adaptador.accountProfileBindingRef == "" {
		return nil, nil
	}
	perfil := adaptador.accountProfileBindingRef
	colocacion, err := ports.NewAgentPlacementRef("placement:" + perfil)
	if err != nil || !validAccountProfileRef(perfil) {
		return nil, &Error{Code: CodePoolConfigInvalid, Cause: err}
	}
	return []application.DescriptorCapacidadColocacionAgente{{PlacementRef: colocacion,
		SourceRef: referenciaFuenteCapacidadCodex, PoolRef: "capacity-pool:" + application.AgentCapacityPoolRef(perfil),
		BaseMedicion: application.BaseMedicionCapacidadBruta, Plazas: 1}}, nil
}

func (pool *Pool) DescribirCapacidadColocaciones() ([]application.DescriptorCapacidadColocacionAgente, error) {
	if pool == nil {
		return nil, &Error{Code: CodeStateInvalid}
	}
	resultado := make([]application.DescriptorCapacidadColocacionAgente, 0, len(pool.profiles))
	for _, perfil := range pool.profiles {
		descriptores, err := perfil.adapter.DescribirCapacidadColocaciones()
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, descriptores...)
	}
	return resultado, nil
}
