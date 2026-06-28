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

func TestNormalizeConfigV0ObservadorGoalFirstActivoPorDefectoV0(t *testing.T) {
	config := NormalizeConfigV0(ConfigV0{})

	if !config.GoalObserverEnabled ||
		config.GoalObserverMaxItems != DefaultGoalObserverMaxItemsV0 ||
		config.GoalObserverInterval != DefaultTickIntervalV0 {
		t.Fatalf("goal observer config=%+v", config)
	}
}

func TestNormalizeConfigV0PermiteDesactivarObservadorGoalFirstV0(t *testing.T) {
	config := NormalizeConfigV0(ConfigV0{
		GoalObserverEnabledConfigured: true,
		GoalObserverEnabled:           false,
		GoalObserverMaxItems:          12,
	})

	if config.GoalObserverEnabled || config.GoalObserverMaxItems != 12 {
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
