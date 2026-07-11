package orquestaserver

import (
	"testing"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestNormalizeConfigV0RespetaMaxTicksConfiguradoV0(t *testing.T) {
	config := NormalizeConfigV0(ConfigV0{
		SupervisorCommand: orquestarunsupervisor.RunSupervisorCommandV0{
			MaxTicks: 7,
		},
	})

	if config.SupervisorCommand.MaxTicks != 7 {
		t.Fatalf("max_ticks=%d want=7", config.SupervisorCommand.MaxTicks)
	}
}

func TestNormalizeConfigV0UsaMaxTicksSeguroPorDefectoV0(t *testing.T) {
	for _, maxTicks := range []int{0, -3} {
		config := NormalizeConfigV0(ConfigV0{
			SupervisorCommand: orquestarunsupervisor.RunSupervisorCommandV0{
				MaxTicks: maxTicks,
			},
		})

		if config.SupervisorCommand.MaxTicks != DefaultSupervisorMaxTicksV0 {
			t.Fatalf("input=%d max_ticks=%d want=%d", maxTicks, config.SupervisorCommand.MaxTicks, DefaultSupervisorMaxTicksV0)
		}
	}
}

func TestNormalizeConfigV0AjustaObjetivoAutomejoraAlMaximoDeTandaV0(t *testing.T) {
	config := NormalizeConfigV0(ConfigV0{
		IdleSelfImprovementMaxRequests: 8,
		IdleSelfImprovementTargetQueue: 3,
	})

	if config.IdleSelfImprovementTargetQueue != 8 {
		t.Fatalf("target_queue=%d want=8", config.IdleSelfImprovementTargetQueue)
	}
}

func TestNormalizeConfigV0AutomejoraDefaultDiezPadresV0(t *testing.T) {
	config := NormalizeConfigV0(ConfigV0{})

	if config.IdleSelfImprovementMaxRequests != 10 ||
		config.IdleSelfImprovementTargetQueue != 10 {
		t.Fatalf("idle defaults max_requests=%d target_queue=%d",
			config.IdleSelfImprovementMaxRequests,
			config.IdleSelfImprovementTargetQueue,
		)
	}
}

func TestNormalizeConfigV0AutomejoraIdleDefaultYDesactivacionV0(t *testing.T) {
	config := NormalizeConfigV0(ConfigV0{})
	if config.IdleSelfImprovementDisabled ||
		config.IdleSelfImprovementAfter != DefaultIdleSelfImprovementAfterV0 {
		t.Fatalf("idle self improvement default config=%+v", config)
	}
	if config.IdleSelfImprovementAfter != 60*time.Second {
		t.Fatalf("idle self improvement default config=%+v", config)
	}

	disabled := NormalizeConfigV0(ConfigV0{
		IdleSelfImprovementDisabled: true,
		IdleSelfImprovementAfter:    37 * time.Second,
	})
	if !disabled.IdleSelfImprovementDisabled ||
		disabled.IdleSelfImprovementAfter != 0 {
		t.Fatalf("idle self improvement disabled config=%+v", disabled)
	}
}

func TestNormalizeConfigV0ObservadorGoalFirstActivoPorDefectoV0(t *testing.T) {
	config := NormalizeConfigV0(ConfigV0{})

	if !config.GoalObserverEnabled ||
		config.GoalObserverMaxItems != DefaultGoalObserverMaxItemsV0 ||
		config.GoalObserverInterval != DefaultTickIntervalV0 ||
		config.GoalObserverTimeout != DefaultGoalObserverTimeoutV0 {
		t.Fatalf("goal observer config=%+v", config)
	}
}

func TestDefaultGoalObserverTimeoutV0AllowsDurableAttestation(t *testing.T) {
	if DefaultGoalObserverTimeoutV0 < 10*time.Minute {
		t.Fatalf("goal observer timeout=%s, want durable attestation window", DefaultGoalObserverTimeoutV0)
	}
}

func TestNormalizeConfigV0PermiteDesactivarObservadorGoalFirstV0(t *testing.T) {
	config := NormalizeConfigV0(ConfigV0{
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           false,
		GoalObserverMaxItems:          12,
		GoalObserverTimeout:           3 * time.Second,
	})

	if config.GoalObserverEnabled ||
		config.GoalObserverMaxItems != 12 ||
		config.GoalObserverTimeout != 3*time.Second {
		t.Fatalf("goal observer config=%+v", config)
	}
}

func TestNormalizeConfigV0ObservadorGoalFirstHeredaTickSiNoTieneIntervaloPropioV0(t *testing.T) {
	config := NormalizeConfigV0(ConfigV0{
		TickInterval: 17 * time.Second,
	})

	if config.GoalObserverInterval != 17*time.Second {
		t.Fatalf("goal observer interval=%s want=17s", config.GoalObserverInterval)
	}
}

func TestNormalizeConfigV0ObservadorGoalFirstPermiteIntervaloPropioV0(t *testing.T) {
	config := NormalizeConfigV0(ConfigV0{
		TickInterval:         time.Hour,
		GoalObserverInterval: 1500 * time.Millisecond,
	})

	if config.GoalObserverInterval != 1500*time.Millisecond {
		t.Fatalf("goal observer interval=%s want=1500ms", config.GoalObserverInterval)
	}
}
