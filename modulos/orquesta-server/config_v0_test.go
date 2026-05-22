package orquestaserver

import (
	"testing"

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
