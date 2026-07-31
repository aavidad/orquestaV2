package bootstrap

import (
	"context"
	"errors"
	"testing"

	"orquesta/internal/application"
)

type controladorCuotaPrueba struct {
	inicial error
	cerrado int
}

func (controlador *controladorCuotaPrueba) EsperarInicial(context.Context) error {
	return controlador.inicial
}

func (controlador *controladorCuotaPrueba) Cerrar(context.Context) error {
	controlador.cerrado++
	return nil
}

type agenteCuotaPrueba struct {
	AgentAdapter
	controladores []application.ControladorCuotaAgente
}

func (agente *agenteCuotaPrueba) IniciarControladoresCuota(
	context.Context, application.ConfiguracionControladoresCuotaAgente,
) ([]application.ControladorCuotaAgente, error) {
	return agente.controladores, nil
}

type estadoCuotaBootstrapPrueba struct{ application.StateRepository }
type artefactosCuotaBootstrapPrueba struct{ application.ArtifactStore }

func TestAbrirControladoresCuotaRecogeTodosAnteFalloInicial(t *testing.T) {
	setup, err := prepareBuildSetup(context.Background(), writeTestConfig(t, t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	primero := &controladorCuotaPrueba{}
	segundo := &controladorCuotaPrueba{inicial: errors.New("cuota inicial no disponible")}
	agente := &agenteCuotaPrueba{controladores: []application.ControladorCuotaAgente{primero, segundo}}
	controladores, err := abrirControladoresCuota(
		context.Background(), setup, agente,
		estadoCuotaBootstrapPrueba{}, artefactosCuotaBootstrapPrueba{},
	)
	if controladores != nil || err == nil || primero.cerrado != 1 || segundo.cerrado != 1 {
		t.Fatalf("controladores=%v error=%v cierres=%d/%d", controladores, err, primero.cerrado, segundo.cerrado)
	}
}
