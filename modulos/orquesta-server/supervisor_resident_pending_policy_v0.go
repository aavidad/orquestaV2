package orquestaserver

import (
	"strings"

	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

const idleSelfImprovementResidentPendingBlockedReasonV0 = "resident_pending_without_dispatch"

type residentPendingDispatchBlockV0 struct {
	Blocked      bool
	RunRefs      []string
	EvidenceRefs []string
}

func detectResidentPendingWithoutDispatchV0(
	result orquestarunsupervisor.RunSupervisorResultV0,
) residentPendingDispatchBlockV0 {
	block := residentPendingDispatchBlockV0{}
	for _, tick := range result.Ticks {
		for _, execution := range tick.Result.Executions {
			if !residentPendingValueV0(execution.Outcome, execution.QueueStatus) ||
				supervisorLiveEvidenceV0(execution.Outcome, execution.QueueStatus, execution.EvidenceRefs) ||
				supervisorDiagnosticsHaveLiveProcessV0(execution.Diagnostics) {
				continue
			}
			block.Blocked = true
			block.RunRefs = append(block.RunRefs, execution.RunRef)
			block.EvidenceRefs = append(block.EvidenceRefs, execution.EvidenceRefs...)
			block.EvidenceRefs = append(block.EvidenceRefs, idleSelfImprovementDiagnosticEvidenceV0(execution.Diagnostics)...)
		}
		for _, skip := range tick.Result.Skips {
			if !residentPendingValueV0(skip.Reason, skip.Status) {
				continue
			}
			if supervisorDiagnosticsHaveLiveProcessForRunV0(result.Diagnostics, skip.RunRef) {
				continue
			}
			block.Blocked = true
			block.RunRefs = append(block.RunRefs, skip.RunRef)
		}
	}
	if !block.Blocked && residentPendingValueV0(result.StopProjection.WaitingReasons...) {
		block.Blocked = true
		block.RunRefs = idleSelfImprovementKnownRunRefsV0(result)
		block.EvidenceRefs = append(block.EvidenceRefs, result.StopProjection.EvidenceRefs...)
	}
	block.EvidenceRefs = append(block.EvidenceRefs, residentPendingDiagnosticEvidenceV0(result.Diagnostics)...)
	block.RunRefs = compactConfigStringsV0(block.RunRefs)
	block.EvidenceRefs = compactConfigStringsV0(block.EvidenceRefs)
	return block
}

func residentPendingValueV0(values ...string) bool {
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case "resident_director_pending", idleSelfImprovementResidentPendingBlockedReasonV0:
			return true
		}
	}
	return false
}

func residentPendingDiagnosticEvidenceV0(
	diagnostics []orquestaruncoordinator.RunDrainDiagnosticV0,
) []string {
	var refs []string
	for _, diagnostic := range diagnostics {
		if residentPendingValueV0(diagnostic.Kind, diagnostic.Status, diagnostic.FinalAction) {
			refs = append(refs, diagnostic.EvidenceRefs...)
			refs = append(refs, diagnostic.FirstPendingRefs...)
		}
	}
	return compactConfigStringsV0(refs)
}
