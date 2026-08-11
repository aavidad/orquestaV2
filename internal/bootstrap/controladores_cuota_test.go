package bootstrap

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/config"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

type controladorCuotaPrueba struct {
	inicial        error
	esperarInicial func(context.Context) error
	configuracion  application.ConfiguracionControladoresCuotaAgente
	cerrado        atomic.Int64
}

func (controlador *controladorCuotaPrueba) EsperarInicial(ctx context.Context) error {
	if controlador.esperarInicial != nil {
		return controlador.esperarInicial(ctx)
	}
	return controlador.inicial
}

func (controlador *controladorCuotaPrueba) Cerrar(context.Context) error {
	controlador.cerrado.Add(1)
	return nil
}

func (controlador *controladorCuotaPrueba) publicarDisponible(ctx context.Context) error {
	colocacion, err := ports.NewAgentPlacementRef("placement:test")
	if err != nil {
		return err
	}
	ahora := controlador.configuracion.Ahora().Round(0).UTC()
	return controlador.configuracion.Sumidero(ctx, application.AgentQuotaObservation{
		PlacementRef: colocacion,
		WindowRef:    "window:bootstrap:late",
		Status:       application.AgentQuotaAvailable,
		Quality:      governance.UsageQualityExact,
		ObservedAt:   ahora,
		ExpiresAt:    ahora.Add(controlador.configuracion.VigenciaObservacion),
	}, nil)
}

type agenteCuotaPrueba struct {
	AgentAdapter
	controladores []application.ControladorCuotaAgente
}

func (agente *agenteCuotaPrueba) IniciarControladoresCuota(
	_ context.Context, configuracion application.ConfiguracionControladoresCuotaAgente,
) ([]application.ControladorCuotaAgente, error) {
	for _, controlador := range agente.controladores {
		if prueba, ok := controlador.(*controladorCuotaPrueba); ok {
			prueba.configuracion = configuracion
		}
	}
	return agente.controladores, nil
}

func (agente *agenteCuotaPrueba) DescribirCapacidadColocaciones() ([]application.DescriptorCapacidadColocacionAgente, error) {
	catalogo, disponible := agente.AgentAdapter.(catalogoCapacidadColocacionAgente)
	if !disponible {
		return nil, nil
	}
	return catalogo.DescribirCapacidadColocaciones()
}

type estadoCuotaBootstrapPrueba struct{ application.StateRepository }
type artefactosCuotaBootstrapPrueba struct{ application.ArtifactStore }

func TestAbrirControladoresCuotaConservaTodosAnteFalloInicial(t *testing.T) {
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
	if err != nil || len(controladores) != 2 || primero.cerrado.Load() != 0 || segundo.cerrado.Load() != 0 {
		t.Fatalf("controladores=%v error=%v cierres=%d/%d", controladores, err, primero.cerrado.Load(), segundo.cerrado.Load())
	}
	if err := cerrarControladoresCuota(controladores, setup.snapshot.ServerShutdownTimeout()); err != nil {
		t.Fatal(err)
	}
	if primero.cerrado.Load() != 1 || segundo.cerrado.Load() != 1 {
		t.Fatalf("cierre final no exacto: %d/%d", primero.cerrado.Load(), segundo.cerrado.Load())
	}
}

func TestBuildCuotaInicialAusenteMantieneServidorYLaunchFailClosedHastaObservacion(t *testing.T) {
	raiz := t.TempDir()
	configuracion := writeTestConfig(t, raiz)
	replaceTestConfigValue(t, configuracion,
		"[runtime.codex]\n",
		"[runtime.capacity]\nobservation_timeout = \"20ms\"\n\n[runtime.codex]\n",
	)
	controlador := &controladorCuotaPrueba{
		esperarInicial: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	var lanzamientos atomic.Int64
	runtime, err := Build(context.Background(), Options{
		ConfigPath: configuracion,
		Version:    "quota-startup-test",
		AgentFactory: func(_ config.Snapshot, reloj application.Clock) (AgentAdapter, error) {
			return &agenteCuotaPrueba{
				AgentAdapter:  newCountingAgent(reloj, &lanzamientos),
				controladores: []application.ControladorCuotaAgente{controlador},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("Build con cuota inicial ausente: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		_ = runtime.Shutdown(ctx)
	})
	if controlador.cerrado.Load() != 0 {
		t.Fatalf("Build cerró el controlador vivo %d veces", controlador.cerrado.Load())
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("Start con cuota inicial ausente: %v", err)
	}
	goalRef := submitTestGoal(t, runtime, "request:quota-startup-late")
	time.Sleep(100 * time.Millisecond)
	if lanzamientos.Load() != 0 {
		t.Fatalf("launch sin observación de cuota real: %d", lanzamientos.Load())
	}
	if err := controlador.publicarDisponible(context.Background()); err != nil {
		t.Fatalf("publicar observación posterior: %v", err)
	}
	terminal := waitTerminalGoal(t, runtime, goalRef)
	if terminal.Goal.State() != goal.GoalStateSucceeded || lanzamientos.Load() != 1 {
		t.Fatalf("observación posterior no habilitó launch: state=%s launches=%d", terminal.Goal.State(), lanzamientos.Load())
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	if err := runtime.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if controlador.cerrado.Load() != 1 {
		t.Fatalf("cierres del controlador=%d, want 1", controlador.cerrado.Load())
	}
}
