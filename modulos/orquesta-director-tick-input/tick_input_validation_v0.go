package orquestadirectortickinput

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	maxTickInputRefsV0   = 1024
	maxTickInputStringV0 = 600
)

func validateDirectorTickInputBuildRequestV0(input DirectorTickInputBuildRequestV0) error {
	if strings.TrimSpace(input.TickRef) == "" {
		return tickInputErrorV0("tick_ref")
	}
	if strings.TrimSpace(input.OccurredAt) == "" {
		return tickInputErrorV0("occurred_at")
	}
	if strings.TrimSpace(input.Run.RunID) == "" {
		return tickInputErrorV0("run.run_id")
	}
	if strings.TrimSpace(string(input.Run.CurrentPhase)) == "" {
		return tickInputErrorV0("run.current_phase")
	}
	if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(input.Run); len(issues) > 0 {
		return tickInputErrorV0("run." + issues[0].Field)
	}
	if refsInvalidTickInputV0(input.PendingOutboxRefs) || refsInvalidTickInputV0(input.EvidenceRefs) {
		return tickInputErrorV0("refs")
	}
	return nil
}

func refsInvalidTickInputV0(refs []string) bool {
	// El tick-input transporta refs/evidencias opacas. No corta por forma:
	// normaliza y deja que el director corrija lo ambiguo.
	return false
}

func tickInputErrorV0(field string) DirectorTickInputBuildErrorV0 {
	return DirectorTickInputBuildErrorV0{Code: ErrDirectorTickInputBuildInvalidoV0, Field: field}
}
