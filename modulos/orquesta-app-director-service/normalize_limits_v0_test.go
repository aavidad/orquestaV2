package orquestaappdirectorservice

import (
	"testing"

	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestNormalizeStartAppDirectorRequestV0AcotaPresupuestoAlRunner(t *testing.T) {
	request := normalizeStartAppDirectorRequestV0(StartAppDirectorRequestV0{
		AppSpecRequest:    orquestafactory.AppSpecRequestV0{RequestID: "req-normalize-budget-start"},
		MaxCommands:       24,
		MaxOutboxPerCycle: 24,
	})

	if request.MaxCommands != orquestadirectorrunner.DirectorCycleMaxCommandsV0 {
		t.Fatalf("max_commands=%d", request.MaxCommands)
	}
	if request.MaxOutboxPerCycle != orquestadirectorrunner.DirectorCycleMaxOutboxV0 {
		t.Fatalf("max_outbox=%d", request.MaxOutboxPerCycle)
	}
}

func TestNormalizeContinueAppDirectorRequestV0AcotaPresupuestoAlRunner(t *testing.T) {
	request := normalizeContinueAppDirectorRequestV0(ContinueAppDirectorRequestV0{
		RunRef:            "run-normalize-budget-continue",
		MaxCommands:       24,
		MaxOutboxPerCycle: 24,
	})

	if request.MaxCommands != orquestadirectorrunner.DirectorCycleMaxCommandsV0 {
		t.Fatalf("max_commands=%d", request.MaxCommands)
	}
	if request.MaxOutboxPerCycle != orquestadirectorrunner.DirectorCycleMaxOutboxV0 {
		t.Fatalf("max_outbox=%d", request.MaxOutboxPerCycle)
	}
}
