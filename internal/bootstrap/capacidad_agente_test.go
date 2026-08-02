package bootstrap

import (
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func TestComponerFuentesCapacidadAgenteConservaCatalogoOpacoYBruto(t *testing.T) {
	colocacion, _ := ports.NewAgentPlacementRef("placement:opaque")
	descriptor := application.DescriptorCapacidadColocacionAgente{
		PlacementRef: colocacion, SourceRef: "source:codex", PoolRef: "pool:opaque",
		BaseMedicion: application.BaseMedicionCapacidadBruta, Plazas: 1,
	}
	agente := &agenteCatalogoCapacidadPrueba{descriptores: []application.DescriptorCapacidadColocacionAgente{descriptor}}
	fuentes, err := componerFuentesCapacidadAgente(agente, time.Minute, func() time.Time { return time.Unix(100, 0).UTC() }, true)
	if err != nil || len(fuentes) != 1 || fuentes[0].PlacementRef != colocacion ||
		fuentes[0].BaseMedicion != application.BaseMedicionCapacidadBruta || fuentes[0].Observer == nil {
		t.Fatalf("fuentes=%+v error=%v", fuentes, err)
	}
}

func TestComponerFuentesCapacidadAgenteFallaCerradoSinCatalogoOBasemedicion(t *testing.T) {
	if _, err := componerFuentesCapacidadAgente(&agenteSinCatalogoCapacidad{}, time.Minute, time.Now, true); err == nil {
		t.Fatal("catálogo requerido ausente aceptado")
	}
	if fuentes, err := componerFuentesCapacidadAgente(&agenteCatalogoCapacidadPrueba{}, time.Minute, time.Now, true); err != nil || fuentes != nil {
		t.Fatalf("catálogo vacío activó la compuerta antes del cutover: %+v %v", fuentes, err)
	}
	colocacion, _ := ports.NewAgentPlacementRef("placement:invalid")
	agente := &agenteCatalogoCapacidadPrueba{descriptores: []application.DescriptorCapacidadColocacionAgente{{
		PlacementRef: colocacion, SourceRef: "source:codex", PoolRef: "pool:invalid", Plazas: 1,
	}}}
	if _, err := componerFuentesCapacidadAgente(agente, time.Minute, time.Now, true); err == nil {
		t.Fatal("medición no bruta aceptada")
	}
}

type agenteCatalogoCapacidadPrueba struct {
	AgentAdapter
	descriptores []application.DescriptorCapacidadColocacionAgente
}

func (agente *agenteCatalogoCapacidadPrueba) DescribirCapacidadColocaciones() ([]application.DescriptorCapacidadColocacionAgente, error) {
	return agente.descriptores, nil
}

type agenteSinCatalogoCapacidad struct{ AgentAdapter }
