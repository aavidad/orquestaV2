package orquestarunqueue

import (
	"sort"
	"strings"
)

func ProjectRunQueueAttemptsV0(candidates []RunSchedulingCandidateV0) []RunQueueAttemptProjectionV0 {
	groups := map[string]*RunQueueAttemptProjectionV0{}
	order := []string{}
	candidatesByGroup := map[string][]RunSchedulingCandidateV0{}
	superseded := map[string]map[string]bool{}

	for _, raw := range candidates {
		candidate := normalizeRunSchedulingCandidateV0(raw)
		groupRef := runQueueAttemptGroupRefV0(candidate)
		if groupRef == "" {
			groupRef = "run:" + candidate.RunRef
		}
		projection, ok := groups[groupRef]
		if !ok {
			projection = &RunQueueAttemptProjectionV0{
				GroupRef:     groupRef,
				AttemptGroup: candidate.AttemptGroup,
				StatusCounts: map[string]int{},
			}
			groups[groupRef] = projection
			order = append(order, groupRef)
		}
		candidatesByGroup[groupRef] = append(candidatesByGroup[groupRef], candidate)
		if projection.OriginalRunRef == "" && strings.TrimSpace(candidate.ParentRunRef) == "" {
			projection.OriginalRunRef = candidate.RunRef
		}
		projection.RunRefs = compactRunQueueStringsV0(append(projection.RunRefs, candidate.RunRef))
		projection.EvidenceRefs = compactRunQueueStringsV0(append(projection.EvidenceRefs, candidate.EvidenceRefs...))
		status := strings.TrimSpace(candidate.Status)
		if status != "" {
			projection.StatusCounts[status]++
		}
		if candidate.ParentRunRef != "" || candidate.SupersedesRunRef != "" || candidate.RescueReason != "" {
			projection.RescueRunRefs = compactRunQueueStringsV0(append(projection.RescueRunRefs, candidate.RunRef))
		}
		if candidate.SupersedesRunRef != "" {
			if superseded[groupRef] == nil {
				superseded[groupRef] = map[string]bool{}
			}
			superseded[groupRef][candidate.SupersedesRunRef] = true
		}
	}

	for groupRef, projection := range groups {
		active, ok := activeRunQueueAttemptV0(candidatesByGroup[groupRef], superseded[groupRef])
		if !ok {
			continue
		}
		projection.ActiveAttemptRef = active.RunRef
		projection.ParentRunRef = active.ParentRunRef
		projection.SupersedesRunRef = active.SupersedesRunRef
		projection.RescueReason = active.RescueReason
		projection.Status = strings.TrimSpace(active.Status)
	}

	sort.Strings(order)
	out := make([]RunQueueAttemptProjectionV0, 0, len(order))
	for _, groupRef := range order {
		out = append(out, *groups[groupRef])
	}
	return out
}

func activeRunQueueAttemptV0(
	candidates []RunSchedulingCandidateV0,
	superseded map[string]bool,
) (RunSchedulingCandidateV0, bool) {
	var active RunSchedulingCandidateV0
	hasActive := false
	for _, candidate := range candidates {
		if superseded[candidate.RunRef] {
			continue
		}
		if runQueueCandidateMoreActiveV0(candidate, active, hasActive) {
			active = candidate
			hasActive = true
		}
	}
	return active, hasActive
}

func normalizeRunSchedulingCandidateV0(candidate RunSchedulingCandidateV0) RunSchedulingCandidateV0 {
	candidate.RunRef = strings.TrimSpace(candidate.RunRef)
	candidate.AppRef = strings.TrimSpace(candidate.AppRef)
	candidate.Status = strings.TrimSpace(candidate.Status)
	candidate.FairnessGroupRef = strings.TrimSpace(candidate.FairnessGroupRef)
	candidate.AttemptGroup = NormalizeRunQueueAttemptGroupV0(candidate.AttemptGroup)
	candidate.ParentRunRef = strings.TrimSpace(candidate.ParentRunRef)
	candidate.SupersedesRunRef = strings.TrimSpace(candidate.SupersedesRunRef)
	candidate.RescueReason = strings.TrimSpace(candidate.RescueReason)
	candidate.EvidenceRefs = compactRunQueueStringsV0(candidate.EvidenceRefs)
	candidate.WorksetClaims = cloneRunQueueWorksetClaimsV0(candidate.WorksetClaims)
	return candidate
}

func runQueueAttemptGroupRefV0(candidate RunSchedulingCandidateV0) string {
	group := NormalizeRunQueueAttemptGroupV0(candidate.AttemptGroup)
	if group.GroupRef != "" {
		return group.GroupRef
	}
	parts := []string{}
	if group.ConsumerRef != "" {
		parts = append(parts, "consumer:"+group.ConsumerRef)
	}
	if group.ObjectiveRef != "" {
		parts = append(parts, "objective:"+group.ObjectiveRef)
	}
	if group.WorkItemRef != "" {
		parts = append(parts, "work:"+group.WorkItemRef)
	}
	for _, ref := range group.WriteSetRefs {
		parts = append(parts, "write:"+ref)
	}
	return strings.Join(parts, "|")
}

// RunQueueAttemptGroupRefV0 devuelve la identidad causal normalizada del grupo
// para que los consumidores validen enlaces entre intentos sin reimplementar
// la derivacion.
func RunQueueAttemptGroupRefV0(candidate RunSchedulingCandidateV0) string {
	return runQueueAttemptGroupRefV0(normalizeRunSchedulingCandidateV0(candidate))
}

func runQueueCandidateMoreActiveV0(candidate RunSchedulingCandidateV0, current RunSchedulingCandidateV0, hasCurrent bool) bool {
	if !hasCurrent {
		return true
	}
	candidateExecutable := IsExecutableRunStatusV0(candidate.Status)
	currentExecutable := IsExecutableRunStatusV0(current.Status)
	if candidateExecutable != currentExecutable {
		return candidateExecutable
	}
	if !candidate.UpdatedAt.Equal(current.UpdatedAt) {
		return candidate.UpdatedAt.After(current.UpdatedAt)
	}
	if candidate.PriorityScore != current.PriorityScore {
		return candidate.PriorityScore > current.PriorityScore
	}
	return candidate.RunRef > current.RunRef
}
