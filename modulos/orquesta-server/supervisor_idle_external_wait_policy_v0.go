package orquestaserver

import (
	"strings"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	stopreason "orquesta/modulos/orquesta-run-supervisor/stopreason"
)

const idleSelfImprovementExternalWaitBlockedReasonV0 = "active_external_wait"

type idleSelfImprovementExternalWaitBlockV0 struct {
	Blocked      bool
	RunRefs      []string
	EvidenceRefs []string
}

func detectIdleSelfImprovementExternalWaitBlockV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
) idleSelfImprovementExternalWaitBlockV0 {
	block := idleSelfImprovementExternalWaitBlockV0{}
	for _, tick := range result.Ticks {
		for _, execution := range tick.Result.Executions {
			if !idleSelfImprovementExternalWaitOutcomeV0(execution.Outcome, execution.QueueStatus) {
				continue
			}
			block.Blocked = true
			block.RunRefs = append(block.RunRefs, execution.RunRef)
			block.EvidenceRefs = append(block.EvidenceRefs, execution.EvidenceRefs...)
			block.EvidenceRefs = append(block.EvidenceRefs, idleSelfImprovementDiagnosticEvidenceV0(execution.Diagnostics)...)
		}
		for _, skip := range tick.Result.Skips {
			if !idleSelfImprovementExternalWaitOutcomeV0(skip.Reason, skip.Status) {
				continue
			}
			block.Blocked = true
			block.RunRefs = append(block.RunRefs, skip.RunRef)
		}
	}
	if !block.Blocked && result.StopProjection.PublicReason == stopreason.PublicReasonWaitExternalV0 {
		block.Blocked = true
		block.RunRefs = idleSelfImprovementKnownRunRefsV0(result)
		block.EvidenceRefs = append(block.EvidenceRefs, result.StopProjection.EvidenceRefs...)
	}
	block.RunRefs = compactConfigStringsV0(block.RunRefs)
	block.EvidenceRefs = compactConfigStringsV0(append(
		block.EvidenceRefs,
		idleSelfImprovementDiagnosticEvidenceV0(result.Diagnostics)...,
	))
	return block
}

func idleSelfImprovementExternalWaitOutcomeV0(values ...string) bool {
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case "wait_external", "candidate_pending":
			return true
		}
	}
	return false
}

func idleSelfImprovementDiagnosticEvidenceV0(
	diagnostics []orquestaruncoordinator.RunDrainDiagnosticV0,
) []string {
	var refs []string
	for _, diagnostic := range diagnostics {
		refs = append(refs, diagnostic.EvidenceRefs...)
		refs = append(refs, diagnostic.FirstPendingRefs...)
	}
	return compactConfigStringsV0(refs)
}
