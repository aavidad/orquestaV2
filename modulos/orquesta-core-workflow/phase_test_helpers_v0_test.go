package orquestacoreworkflow

import (
	"strings"
	"testing"
)

func mustHandlerProgramacionRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	return mustHandlerRunInPhaseV0(t, OrchestrationPhaseProgramacionV0)
}

func mustHandlerRunInPhaseV0(t *testing.T, phase OrchestrationPhaseIDV0) OrchestrationRunV0 {
	t.Helper()
	run := mustHandlerStartedRunV0(t)
	if normalizePhaseIDV0(run.CurrentPhase) == normalizePhaseIDV0(phase) {
		return run
	}
	key := strings.ReplaceAll(string(phase), "_", "-")
	return mustApplySingleCommandEventV0(t, run, mustOpenPhaseCommandV0(t, "cmd-open-"+key, "idem-open-"+key, phase))
}

func mustReducerProgramacionRunV0(t *testing.T) OrchestrationRunV0 {
	t.Helper()
	return mustReducerRunInPhaseV0(t, OrchestrationPhaseProgramacionV0)
}

func mustReducerRunInPhaseV0(t *testing.T, phase OrchestrationPhaseIDV0) OrchestrationRunV0 {
	t.Helper()
	run := mustReducerStartedRunV0(t)
	if normalizePhaseIDV0(run.CurrentPhase) == normalizePhaseIDV0(phase) {
		return run
	}
	key := strings.ReplaceAll(string(phase), "_", "-")
	return mustApplyReducerEventV0(t, run, mustReducerPhaseOpenedEventV0(t, "evt-open-"+key, run.LastSequence+1, phase))
}
