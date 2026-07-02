package orquestaserver

import (
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
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
			if !supervisorExecutionHasExternalWaitV0(execution) {
				continue
			}
			block.Blocked = true
			block.RunRefs = append(block.RunRefs, execution.RunRef)
			block.EvidenceRefs = append(block.EvidenceRefs, execution.EvidenceRefs...)
			block.EvidenceRefs = append(block.EvidenceRefs, idleSelfImprovementDiagnosticEvidenceV0(execution.Diagnostics)...)
		}
		for _, skip := range tick.Result.Skips {
			if !supervisorExternalWaitValueV0(skip.Reason, skip.Status) {
				continue
			}
			block.Blocked = true
			block.RunRefs = append(block.RunRefs, skip.RunRef)
		}
	}
	if !block.Blocked && supervisorResultHasExternalWaitV0(result) {
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
