package orquestaruntimecodexappserver

import (
	"testing"

	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

func TestCodexAppServerGoalSandboxV0DangerRequiereFronteraContenedorConfirmadaV0(t *testing.T) {
	packet := orquestaruntimecodexgoal.CodexGoalStartPacketV0{
		DirectionContract: orquestaruntimecodexgoal.CodexGoalDirectionContractV0{
			WriteSetEnforcement: orquestaruntimecodexgoal.CodexGoalWriteSetEnforcementV0,
			MinimumSandbox:      orquestaruntimecodexgoal.CodexGoalMinimumSandboxV0,
		},
	}
	t.Setenv("ORQUESTA_CODEX_CONTAINER_SANDBOX_BOUNDARY_CONFIRMED", "")
	if got := codexAppServerGoalSandboxForPacketV0("danger-full-access", packet); got != orquestaruntimecodexgoal.CodexGoalMinimumSandboxV0 {
		t.Fatalf("sandbox sin confirmacion=%q", got)
	}
	t.Setenv("ORQUESTA_CODEX_CONTAINER_SANDBOX_BOUNDARY_CONFIRMED", "1")
	if got := codexAppServerGoalSandboxForPacketV0("danger-full-access", packet); got != "danger-full-access" {
		t.Fatalf("sandbox con frontera confirmada=%q", got)
	}
}
