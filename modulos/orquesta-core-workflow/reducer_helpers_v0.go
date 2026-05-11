package orquestacoreworkflow

import "strings"

func ensureRunCanApplyEventV0(current OrchestrationRunV0, event OrchestrationEventV0) error {
	if orchestrationRunIsEmptyV0(current) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "run")
	}
	if strings.TrimSpace(current.RunID) != strings.TrimSpace(event.RunID) {
		return eventErrorV0(ErrSecuenciaInvalidaV0, "run_id")
	}
	if issues := ValidateOrchestrationRunV0(current); len(issues) > 0 {
		return issues[0]
	}
	return nil
}

func firstPhaseIDV0(phases []OrchestrationPhaseV0) OrchestrationPhaseIDV0 {
	if len(phases) == 0 {
		return ""
	}
	return phases[0].ID
}

func runContainsPhaseV0(run OrchestrationRunV0, phaseID OrchestrationPhaseIDV0) bool {
	for _, phase := range run.Phases {
		if normalizePhaseIDV0(phase.ID) == phaseID {
			return true
		}
	}
	return false
}

func runPhaseIsCurrentV0(run OrchestrationRunV0, phaseID OrchestrationPhaseIDV0) bool {
	return normalizePhaseIDV0(run.CurrentPhase) == normalizePhaseIDV0(phaseID) &&
		runContainsPhaseV0(run, phaseID)
}

func appendUniqueCompactRefV0(refs []string, ref string) []string {
	compact := strings.TrimSpace(ref)
	for _, existing := range refs {
		if strings.TrimSpace(existing) == compact {
			return refs
		}
	}
	return append(refs, compact)
}
