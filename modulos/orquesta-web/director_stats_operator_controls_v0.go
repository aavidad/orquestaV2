package orquestaweb

import "strings"

type WebDirectorStatsClosureV0 struct {
	Status      string   `json:"status,omitempty"`
	Blocked     bool     `json:"blocked,omitempty"`
	Ready       bool     `json:"ready,omitempty"`
	Closed      bool     `json:"closed,omitempty"`
	BlockedBy   []string `json:"blocked_by,omitempty"`
	BlockerRefs []string `json:"blocker_refs,omitempty"`
}

type WebDirectorStatsSafeActionV0 struct {
	Action       string `json:"action"`
	RunRef       string `json:"run_ref,omitempty"`
	Method       string `json:"method"`
	Endpoint     string `json:"endpoint"`
	Reason       string `json:"reason,omitempty"`
	RequiresPost bool   `json:"requires_post"`
}

func directorStatsClosureV0(
	closure WebDirectorClosureStatsContractV0,
) WebDirectorStatsClosureV0 {
	return WebDirectorStatsClosureV0{
		Status:      trimDirectorStatsV0(closure.Status),
		Blocked:     closure.Blocked,
		Ready:       closure.Ready,
		Closed:      closure.Closed,
		BlockedBy:   compactOperationalStringsV0(closure.BlockedBy),
		BlockerRefs: compactOperationalStringsV0(closure.BlockerRefs),
	}
}

func directorStatsSafeActionsV0(
	vm WebDirectorStatsViewModelV0,
) []WebDirectorStatsSafeActionV0 {
	out := []WebDirectorStatsSafeActionV0{}
	runRef := trimDirectorStatsV0(vm.RunRef)
	if runRef == "" {
		return out
	}
	if vm.Counts.AgentsInFlight > 0 || len(vm.Agents) > 0 {
		out = append(out, directorStatsSafeActionV0("supervise", runRef, "agents_in_flight"))
	}
	if vm.Closure.Blocked {
		out = append(out, directorStatsSafeActionV0("review", runRef, strings.Join(vm.Closure.BlockedBy, ",")))
	}
	if vm.Progress.StalledAgents > 0 || vm.Progress.LoopDetectedAgents > 0 ||
		vm.Counts.AgentsNeedAttention > 0 {
		out = append(out, directorStatsSafeActionV0("retry", runRef, "agent_attention"))
	}
	return out
}

func directorStatsSafeActionV0(action string, runRef string, reason string) WebDirectorStatsSafeActionV0 {
	return WebDirectorStatsSafeActionV0{
		Action:       trimDirectorStatsV0(action),
		RunRef:       trimDirectorStatsV0(runRef),
		Method:       "POST",
		Endpoint:     "/api/v0/autoprogramming/supervise",
		Reason:       trimDirectorStatsV0(reason),
		RequiresPost: true,
	}
}
