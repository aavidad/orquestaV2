package orquestaruntimecodexappserver

import (
	"context"

	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func (backend serverCodexAppServerTmuxBackendV0) CleanupActiveShutdownWorkV0(
	ctx context.Context,
	command orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0,
) (orquestaservershutdown.ActiveShutdownWorkCleanupResultV0, error) {
	if !command.CleanupGoalBackends {
		return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{}, nil
	}
	evidenceRefs := compactStringsV0(append(
		append([]string(nil), command.EvidenceRefs...),
		"evidence-ref-codex-app-server-tmux-cleanup-requested",
	))
	cleaned := 0
	if backend.detectTmuxResidueV0(ctx).Active {
		if err := backend.shutdownTmuxSessionForCleanupV0(ctx); err != nil {
			if codexAppServerTmuxIsGenerationConflictV0(err) || codexAppServerTmuxIsObservationTransientV0(err) {
				residualEvidence := "evidence-ref-codex-app-server-tmux-generation-observation-residual"
				if codexAppServerTmuxIsGenerationConflictV0(err) {
					residualEvidence = "evidence-ref-codex-app-server-tmux-generation-conflict-residual"
				}
				return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{
					EvidenceRefs: compactStringsV0(append(
						evidenceRefs,
						residualEvidence,
					)),
				}, nil
			}
			return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{}, err
		}
		if !backend.detectTmuxResidueV0(ctx).Active {
			cleaned++
			evidenceRefs = append(evidenceRefs, "evidence-ref-codex-app-server-tmux-configured-cleaned")
		}
	}
	return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{
		CleanedWorkCount: cleaned,
		EvidenceRefs:     compactStringsV0(evidenceRefs),
	}, nil
}
