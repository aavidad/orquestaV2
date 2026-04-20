package cmd

import "testing"

func TestProcesarAgentesDegradadosAutonomiaBatchDetalladoIncluyeReanimaciones(t *testing.T) {
	prepararDBTemporalCmd(t)

	prev := runtimeProcessReanimationsBatchFn
	runtimeProcessReanimationsBatchFn = func() apiRuntimeProcessReanimationsResponse {
		return apiRuntimeProcessReanimationsResponse{
			OK:              true,
			Candidates:      3,
			Reactivated:     2,
			CapacityBlocked: 1,
		}
	}
	t.Cleanup(func() {
		runtimeProcessReanimationsBatchFn = prev
	})

	got, err := procesarAgentesDegradadosAutonomiaBatchDetallado()
	if err != nil {
		t.Fatalf("procesarAgentesDegradadosAutonomiaBatchDetallado: %v", err)
	}
	if got.Count != 2 {
		t.Fatalf("deberia sumar reanimaciones automáticas al count: %+v", got)
	}
}
