package orquestadirectorcandidates

import "strings"

func normalizePlanInputV0(
	input CompactBacklogPlanCandidatesInputV0,
) CompactBacklogPlanCandidatesInputV0 {
	input.RunRef = strings.TrimSpace(input.RunRef)
	input.PhaseID = strings.TrimSpace(input.PhaseID)
	if len(input.WorkItems) == 0 {
		return input
	}
	items := make([]CompactBacklogPlanWorkItemV0, 0, len(input.WorkItems))
	for _, item := range input.WorkItems {
		items = append(items, normalizePlanWorkItemV0(item))
	}
	input.WorkItems = items
	return input
}

func normalizePlanWorkItemV0(item CompactBacklogPlanWorkItemV0) CompactBacklogPlanWorkItemV0 {
	item.CandidateRef = strings.TrimSpace(item.CandidateRef)
	item.TaskRef = strings.TrimSpace(item.TaskRef)
	item.SubjectClaimRefs = normalizeRefsV0(item.SubjectClaimRefs)
	item.ScopeClaims = normalizeScopeClaimsV0(item.ScopeClaims)
	item.Commands = normalizeCommandsV0(item.Commands)
	item.Capacity = normalizeCapacityV0(item.Capacity)
	item.Agent = normalizeAgentV0(item.Agent)
	item.GateEvidenceRefs = normalizeRefsV0(item.GateEvidenceRefs)
	item.EvidenceRefs = normalizeRefsV0(item.EvidenceRefs)
	return item
}
