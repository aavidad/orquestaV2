package acceptance_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	agentefalso "orquesta/internal/adapters/agent/fake"
	artefactosfs "orquesta/internal/adapters/artifact/filesystem"
	estadosqlite "orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/ports"
)

func TestV38CompuertaANeutralConservaCohortesYCapacidadParcial(t *testing.T) {
	casos := []struct {
		cohorte, plazas int
	}{
		{cohorte: 1, plazas: 1},
		{cohorte: 16, plazas: 5},
		{cohorte: 70, plazas: 10},
		{cohorte: 500, plazas: 20},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(fmt.Sprintf("cohorte_%d_capacidad_%d", caso.cohorte, caso.plazas), func(t *testing.T) {
			sistema := nuevoSistemaCompuertaAV38(t, caso.plazas)
			resultado, err := sistema.orquestador.Submit(
				context.Background(), v06Access(t), application.SubmitRequest{
					RequestRef: fmt.Sprintf("request:v38-gate-a:%d:%d", caso.cohorte, caso.plazas),
					Statement:  "acreditar demanda lógica completa con capacidad física parcial",
					Confirm:    true,
					Plan:       planCompuertaAV38(caso.cohorte),
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			if trabajos := resultado.Record.Goal.WorkItems(); len(trabajos) != caso.cohorte || len(resultado.Record.Executions) != caso.cohorte {
				t.Fatalf("cohorte=%d trabajos/ejecuciones=%d/%d",
					caso.cohorte, len(trabajos), len(resultado.Record.Executions))
			}

			for indice := 0; indice < caso.plazas; indice++ {
				reclamo, encontrado, err := sistema.orquestador.ClaimNextAction(
					context.Background(), fmt.Sprintf("worker:v38-a:%03d", indice),
					application.ActionClaimSelection{},
				)
				if err != nil || !encontrado || reclamo.Action.Kind != application.ActionLaunchAgent ||
					reclamo.ReferenciaColocacion != sistema.colocacion ||
					application.ValidateAgentCapacityReservation(reclamo.CapacityReservation) != nil {
					t.Fatalf("reclamo físico %d/%d inválido: encontrado=%v reclamo=%+v err=%v",
						indice+1, caso.plazas, encontrado, reclamo, err)
				}
			}
			if reclamo, encontrado, err := sistema.orquestador.ClaimNextAction(
				context.Background(), "worker:v38-a:saturado", application.ActionClaimSelection{},
			); err != nil || encontrado || reclamo.Action.Ref != "" {
				t.Fatalf("la capacidad parcial reclamó trabajo adicional: encontrado=%v reclamo=%+v err=%v",
					encontrado, reclamo, err)
			}

			sistema.reabrir(t)
			if reclamo, encontrado, err := sistema.orquestador.ClaimNextAction(
				context.Background(), "worker:v38-a:reinicio", application.ActionClaimSelection{},
			); err != nil || encontrado || reclamo.Action.Ref != "" {
				t.Fatalf("el reinicio perdió las reservas físicas: encontrado=%v reclamo=%+v err=%v",
					encontrado, reclamo, err)
			}
		})
	}
}

func TestV38CompuertaANeutralRefrescaSinKVMNiReintentoCiego(t *testing.T) {
	sistema := nuevoSistemaCompuertaAV38(t, 1)
	if _, err := sistema.orquestador.Submit(
		context.Background(), v06Access(t), application.SubmitRequest{
			RequestRef: "request:v38-gate-a:refresh", Statement: "refrescar capacidad neutral",
			Confirm: true, Plan: planCompuertaAV38(1),
		},
	); err != nil {
		t.Fatal(err)
	}
	sistema.observador.caducar()
	if reclamo, encontrado, err := sistema.orquestador.ClaimNextAction(
		context.Background(), "worker:v38-a:obsoleto", application.ActionClaimSelection{},
	); err != nil || encontrado || reclamo.Action.Ref != "" {
		t.Fatalf("la observación obsoleta abrió un lanzamiento: encontrado=%v reclamo=%+v err=%v",
			encontrado, reclamo, err)
	}
	sistema.observador.refrescar()
	reclamo, encontrado, err := sistema.orquestador.ClaimNextAction(
		context.Background(), "worker:v38-a:refrescado", application.ActionClaimSelection{},
	)
	if err != nil || !encontrado || reclamo.Action.Kind != application.ActionLaunchAgent ||
		reclamo.ReferenciaColocacion != sistema.colocacion {
		t.Fatalf("la observación fresca no reabrió el mismo trabajo: encontrado=%v reclamo=%+v err=%v",
			encontrado, reclamo, err)
	}
}

func TestV38CompuertaANeutralExigeContratosEjecutablesSinFirecracker(t *testing.T) {
	fixture := v38Fixture(t)
	if fixture.Subgates.A.Scope != "neutral_elastic_core" || fixture.Subgates.A.KVM != "not_required" ||
		fixture.Subgates.A.Firecracker != "not_required" || fixture.Subgates.A.StatusEffect != "cannot_accredit_v38" {
		t.Fatalf("frontera de compuerta A inválida: %+v", fixture.Subgates.A)
	}
	raiz := evidenceRepositoryRoot(t)
	pruebas := v13ReadGoTests(t,
		filepath.Join(raiz, "internal", "application"),
		filepath.Join(raiz, "internal", "adapters", "state", "sqlite"),
		filepath.Join(raiz, "internal", "bootstrap"),
	)
	for _, nombre := range []string{
		"TestV38LogicalCohortsDeriveAndScheduleWithoutStaticCeiling",
		"TestSchedulerRespetaCapacidadFisicaDurableSinLimiteLocal",
		"TestSchedulerPassesTheExactFencedClaimToConcurrentProcessing",
		"TestSchedulerKeepsStopAndObserveMovingWhileLaunchesAreSaturated",
		"TestReclamoCapacidadReservaAtomicaYRepiteSinReseleccion",
		"TestAgentPlacementRecoveryReopensAndRejectsSemanticTampering",
		"TestMailboxListUsesDeterministicFIFOOrder",
		"TestMailboxAcknowledgementReplayNeverRedelivers",
		"TestMailboxRestartPreservesEveryCausalFrontier",
		"TestControlsStopCrashReplayConvergesWithoutDuplicateEffect",
		"TestSQLiteTerminalStopSettlesAfterRestartWithReplacementAgentRouting",
		"TestPreservacionEntornoEsDurableIdempotenteYCausal",
	} {
		if !strings.Contains(pruebas, "func "+nombre+"(") {
			t.Errorf("la compuerta A carece de la prueba ejecutable %s", nombre)
		}
	}
	for _, ruta := range []string{
		"internal/application/candidatos_capacidad_agente.go",
		"internal/application/agent_environment.go",
		"internal/bootstrap/scheduler.go",
	} {
		contenido, err := os.ReadFile(filepath.Join(raiz, ruta))
		if err != nil {
			t.Fatal(err)
		}
		texto := strings.ToLower(string(contenido))
		for _, prohibido := range []string{"/dev/kvm", "firecracker", "jailer", "agentmicrovm"} {
			if strings.Contains(texto, prohibido) {
				t.Errorf("la compuerta neutral %s depende de %q", ruta, prohibido)
			}
		}
	}
}

type sistemaCompuertaAV38 struct {
	rutaEstado  string
	repositorio *estadosqlite.Repository
	orquestador *application.Orchestrator
	reloj       *v06Clock
	identidades *v06IDs
	agente      *agentefalso.Adapter
	artefactos  *artefactosfs.Store
	observador  *observadorCapacidadV38
	colocacion  ports.AgentPlacementRef
	politica    application.BudgetPolicy
}

func nuevoSistemaCompuertaAV38(t *testing.T, plazas int) *sistemaCompuertaAV38 {
	t.Helper()
	ctx := context.Background()
	reloj := &v06Clock{now: time.Date(2026, 8, 2, 8, 0, 0, 0, time.UTC)}
	rutaEstado := v06PrivateDatabasePath(t, "v38-gate-a.db")
	repositorio := v06OpenSQLite(t, ctx, rutaEstado, reloj)
	raizArtefactos := t.TempDir()
	if err := os.Chmod(raizArtefactos, 0o700); err != nil {
		t.Fatal(err)
	}
	artefactos, err := artefactosfs.Open(raizArtefactos)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = artefactos.Close() })
	agente, err := agentefalso.New(agentefalso.Config{
		ProviderRef: "provider:v38-neutral", MediaType: "text/plain",
		Content: []byte("resultado neutral V38"), Now: reloj.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	capacidades, err := agente.Capabilities(ctx)
	if err != nil {
		t.Fatal(err)
	}
	colocacion, err := ports.NewAgentPlacementRef("placement:v38-neutral")
	if err != nil {
		t.Fatal(err)
	}
	observador := &observadorCapacidadV38{reloj: reloj, plazas: int64(plazas), revision: 1}
	politica := v06BudgetPolicy(t, reloj.Now())
	sistema := &sistemaCompuertaAV38{
		rutaEstado: rutaEstado, repositorio: repositorio, reloj: reloj,
		identidades: &v06IDs{}, agente: agente, artefactos: artefactos,
		observador: observador, colocacion: colocacion, politica: politica,
	}
	sistema.registrarCuota(t)
	sistema.orquestador = sistema.componer(t, capacidades)
	t.Cleanup(func() { _ = sistema.repositorio.Close() })
	return sistema
}

func (sistema *sistemaCompuertaAV38) componer(
	t *testing.T,
	capacidades ports.AgentCapabilities,
) *application.Orchestrator {
	t.Helper()
	orquestador, err := application.New(application.Dependencies{
		State: sistema.repositorio, Access: sistema.repositorio,
		Launcher: sistema.agente, Observer: sistema.agente, Controller: sistema.agente,
		Artifacts: sistema.artefactos, Clock: sistema.reloj, IDs: sistema.identidades,
		MaxOutputBytes: 1 << 20, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, MaxChildrenPerParent: 6,
		ClaimLease: time.Minute, AttestTestClaimLease: time.Minute,
		DirectorLeaseDuration: 2 * time.Minute, EffectApprovalTTL: sistema.politica.EffectApprovalTTL,
		BudgetPolicy: sistema.politica, ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: capacidades, CapacityObservationWait: time.Second,
		CapacitySources: []application.FuenteCapacidadColocacionAgente{{
			PlacementRef: sistema.colocacion, SourceRef: "capacity-source:v38-neutral",
			PoolRef: "capacity-pool:v38-neutral", BaseMedicion: application.BaseMedicionCapacidadBruta,
			Observer: sistema.observador,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return orquestador
}

func (sistema *sistemaCompuertaAV38) registrarCuota(t *testing.T) {
	t.Helper()
	ahora := sistema.reloj.Now()
	err := application.RegistrarObservacionCuota(
		context.Background(), sistema.repositorio, sistema.artefactos,
		application.AgentQuotaObservation{
			PlacementRef: sistema.colocacion, WindowRef: "quota-window:v38-neutral",
			Status: application.AgentQuotaAvailable, Quality: governance.UsageQualityExact,
			ObservedAt: ahora, ExpiresAt: ahora.Add(2 * time.Hour),
		},
		[]byte(`{"estado":"disponible"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
}

func (sistema *sistemaCompuertaAV38) reabrir(t *testing.T) {
	t.Helper()
	if err := sistema.repositorio.Close(); err != nil {
		t.Fatal(err)
	}
	repositorio, err := estadosqlite.Open(context.Background(), estadosqlite.Options{
		Path: sistema.rutaEstado, BusyTimeout: 5 * time.Second,
		MaxOpenConnections: 4, Now: sistema.reloj.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	sistema.repositorio = repositorio
	capacidades, err := sistema.agente.Capabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	sistema.orquestador = sistema.componer(t, capacidades)
}

func planCompuertaAV38(tamano int) *application.PlanSpec {
	trabajos := make([]application.WorkItemSpec, tamano)
	for indice := range trabajos {
		trabajos[indice] = application.WorkItemSpec{
			Key: fmt.Sprintf("trabajo-%03d", indice), Objective: "ejecutar trabajo neutral",
			Phase: "phase:v38-gate-a", Role: "role:worker",
			OutputContract: goal.OutputContractEvidenceBundle,
		}
	}
	return &application.PlanSpec{
		Phases: []application.PhaseSpec{{
			Ref: "phase-instance:v38-gate-a", Key: "phase:v38-gate-a",
			TemplateRef: "phase-template:v38-gate-a",
		}},
		WorkItems: trabajos,
	}
}

type observadorCapacidadV38 struct {
	mu       sync.Mutex
	reloj    *v06Clock
	plazas   int64
	revision int
	caducado bool
}

func (observador *observadorCapacidadV38) ObserveCapacity(
	ctx context.Context,
	fuente application.AgentCapacitySourceRef,
	pool application.AgentCapacityPoolRef,
) (application.AgentCapacityObservation, error) {
	if err := ctx.Err(); err != nil {
		return application.AgentCapacityObservation{}, err
	}
	observador.mu.Lock()
	defer observador.mu.Unlock()
	ahora := observador.reloj.Now()
	observada, expira := ahora, ahora.Add(time.Hour)
	if observador.caducado {
		observada, expira = ahora.Add(-2*time.Hour), ahora.Add(-time.Hour)
	}
	noAplicable := application.AgentCapacityDimension{
		Applicability: application.AgentCapacityApplicabilityNotApplicable,
	}
	return application.AgentCapacityObservation{
		SourceRef: fuente, PoolRef: pool,
		WindowRef: application.AgentCapacityWindowRef(fmt.Sprintf("capacity-window:v38:%d", observador.revision)),
		Status:    application.AgentCapacityAvailable, Quality: governance.UsageQualityExact,
		ObservedAt: observada, ExpiresAt: expira,
		Resources: application.AgentCapacityResources{
			Slots: application.AgentCapacityDimension{
				Applicability: application.AgentCapacityApplicabilityApplicable,
				Limit:         application.AgentCapacityAmount{Present: true, Value: observador.plazas},
				Remaining:     application.AgentCapacityAmount{Present: true, Value: observador.plazas},
			},
			Seconds: noAplicable, Messages: noAplicable, Tokens: noAplicable, Credits: noAplicable,
		},
	}, nil
}

func (observador *observadorCapacidadV38) caducar() {
	observador.mu.Lock()
	defer observador.mu.Unlock()
	observador.caducado = true
}

func (observador *observadorCapacidadV38) refrescar() {
	observador.mu.Lock()
	defer observador.mu.Unlock()
	observador.caducado = false
	observador.revision++
}
