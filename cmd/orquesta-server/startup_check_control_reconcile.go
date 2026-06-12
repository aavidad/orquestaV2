package main

import (
	"context"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

func (check serverStartupCheckV0) completeStartupRunControlV0(
	ctx context.Context,
	entry startupCandidateCleanupV0,
) error {
	if !entry.CompleteControlPending {
		return nil
	}
	reason := entry.CompleteControlReason
	if reason == "" {
		reason = "shutdown/reload: control terminal reconciliado"
	}
	_, err := check.Stack.Stores.RunControl.CompleteRunControlV0(
		ctx,
		orquestaruncontrol.CompleteRunControlCommandV0{
			RunRef:         entry.Candidate.RunRef,
			TargetStatus:   entry.CompleteControlStatus,
			RequestedBy:    "orquesta-server-startup",
			Reason:         reason,
			IdempotencyKey: "idem-orquesta-server-startup-complete-" + startupSafeRefPartV0(entry.Candidate.RunRef),
			EvidenceRefs: []string{
				"evidence-ref-orquesta-startup-shutdown-reload-control-completed",
			},
		},
	)
	return err
}
