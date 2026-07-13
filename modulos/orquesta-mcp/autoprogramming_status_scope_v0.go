package orquestamcp

import "strings"

const (
	MCPAutoprogrammingStatusScopeLegacyV0 = "legacy"
	MCPAutoprogrammingStatusScopeRunV0    = "run"
	MCPAutoprogrammingStatusScopeQueueV0  = "queue"
)

// scopeMCPAutoprogrammingStatusResultV0 applies the public status selector only
// after every upstream projection is available. Run refs are opaque: membership
// is always exact and is learned from this response, never from a prefix.
func scopeMCPAutoprogrammingStatusResultV0(
	result MCPAutoprogrammingStatusToolResultV0,
	input MCPAutoprogrammingStatusToolInputV0,
) MCPAutoprogrammingStatusToolResultV0 {
	mode, scope := mcpAutoprogrammingStatusScopeV0(input)
	result.ScopeMode, result.Scope = mode, scope
	if mode == MCPAutoprogrammingStatusScopeLegacyV0 {
		return result
	}

	knownRuns := mcpAutoprogrammingKnownRunsV0(result)
	allowedRuns := map[string]bool{}
	switch mode {
	case MCPAutoprogrammingStatusScopeRunV0:
		if knownRuns[scope] {
			allowedRuns[scope] = true
		}
	case MCPAutoprogrammingStatusScopeQueueV0:
		if result.Queue != nil && strings.TrimSpace(result.Queue.QueueRef) == scope {
			for _, candidate := range append(append([]MCPRunQueueRankedCandidateCompactV0{}, result.Queue.Ranked...), result.Queue.Terminal...) {
				if runRef := strings.TrimSpace(candidate.RunRef); knownRuns[runRef] {
					allowedRuns[runRef] = true
				}
			}
		}
	}

	result.Queue = mcpAutoprogrammingScopedQueueV0(result.Queue, allowedRuns)
	if result.Queue == nil {
		result.QueueRef = ""
	}
	result.Run = mcpAutoprogrammingScopedRunV0(result.Run, allowedRuns)
	if result.Run == nil {
		result.RunRef = ""
		result.CausalVerdict = ""
		result.CausalReasonCode = ""
	}
	result.Projects = mcpAutoprogrammingScopedProjectsV0(result.Projects, allowedRuns)
	result.Tasks = mcpAutoprogrammingScopedTasksV0(result.Tasks, allowedRuns)
	result.Agents = mcpAutoprogrammingScopedAgentsV0(result.Agents, allowedRuns)
	result.StaleRunning = mcpAutoprogrammingScopedActionableRunsV0(result.StaleRunning, allowedRuns)
	result.ResolvedRuns = mcpAutoprogrammingScopedActionableRunsV0(result.ResolvedRuns, allowedRuns)
	result.Diagnostics = mcpAutoprogrammingScopedDiagnosticsV0(result.Diagnostics, allowedRuns, mode)
	result.Operator = mcpAutoprogrammingScopedOperatorV0(result.Operator, allowedRuns, mode)
	// Aggregate values have no exact run identity and would otherwise describe
	// discarded runs. They remain available only to the legacy broad response.
	result.QueueHealth = nil
	result.EfficiencySummary = nil
	result.OpsSnapshot = nil
	result.StaleRunningTotal = len(result.StaleRunning)
	result.DiagnosticsTotal = len(result.Diagnostics)
	return result
}

func mcpAutoprogrammingStatusScopeV0(input MCPAutoprogrammingStatusToolInputV0) (string, string) {
	mode := strings.ToLower(strings.TrimSpace(input.ScopeMode))
	scope := strings.TrimSpace(input.Scope)
	if mode == "" {
		mode = MCPAutoprogrammingStatusScopeLegacyV0
	}
	if mode == MCPAutoprogrammingStatusScopeLegacyV0 {
		return mode, ""
	}
	if mode == MCPAutoprogrammingStatusScopeRunV0 && scope == "" {
		scope = strings.TrimSpace(input.RunRef)
	}
	if mode == MCPAutoprogrammingStatusScopeQueueV0 && scope == "" {
		scope = strings.TrimSpace(input.QueueRef)
	}
	return mode, scope
}

func mcpAutoprogrammingKnownRunsV0(result MCPAutoprogrammingStatusToolResultV0) map[string]bool {
	known := map[string]bool{}
	add := func(value string) {
		if value = strings.TrimSpace(value); value != "" {
			known[value] = true
		}
	}
	add(result.RunRef)
	if result.Run != nil {
		add(result.Run.RunRef)
		if result.Run.Stats != nil {
			add(result.Run.Stats.RunRef)
		}
	}
	if result.Queue != nil {
		for _, item := range append(append([]MCPRunQueueRankedCandidateCompactV0{}, result.Queue.Ranked...), result.Queue.Terminal...) {
			add(item.RunRef)
		}
	}
	for _, item := range result.Projects {
		for _, ref := range item.RunRefs {
			add(ref)
		}
	}
	for _, item := range result.Tasks {
		add(item.RunRef)
	}
	for _, item := range result.Agents {
		add(item.RunRef)
	}
	for _, item := range append(append([]MCPAutoprogrammingActionableRunV0{}, result.StaleRunning...), result.ResolvedRuns...) {
		add(item.RunRef)
	}
	if result.Operator != nil {
		for _, item := range result.Operator.ActiveRuns {
			add(item.RunRef)
		}
		for _, item := range result.Operator.ClosureBlockers {
			add(item.RunRef)
		}
		for _, item := range result.Operator.AgentsInFlight {
			add(item.RunRef)
		}
		for _, item := range result.Operator.SafeActions {
			add(item.RunRef)
			mcpAutoprogrammingCollectPayloadRunRefsV0(item.Payload, add)
		}
	}
	for _, item := range result.Diagnostics {
		if ref, ok := mcpAutoprogrammingRunRefFromScopeV0(item.Scope); ok {
			add(ref)
		}
	}
	return known
}

func mcpAutoprogrammingCollectPayloadRunRefsV0(value any, add func(string)) {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			switch strings.ToLower(strings.TrimSpace(key)) {
			case "run_ref":
				if ref, ok := nested.(string); ok {
					add(ref)
				}
			case "run_refs":
				switch refs := nested.(type) {
				case []string:
					for _, ref := range refs {
						add(ref)
					}
				case []any:
					for _, ref := range refs {
						if value, ok := ref.(string); ok {
							add(value)
						}
					}
				}
			}
			mcpAutoprogrammingCollectPayloadRunRefsV0(nested, add)
		}
	case []any:
		for _, nested := range typed {
			mcpAutoprogrammingCollectPayloadRunRefsV0(nested, add)
		}
	case []string:
		for _, nested := range typed {
			mcpAutoprogrammingCollectPayloadRunRefsV0(nested, add)
		}
	}
}

func mcpAutoprogrammingScopedQueueV0(queue *MCPRunQueuePriorityToolResultV0, allowed map[string]bool) *MCPRunQueuePriorityToolResultV0 {
	if queue == nil {
		return nil
	}
	copy := *queue
	copy.Ranked = mcpAutoprogrammingScopedCandidatesV0(queue.Ranked, allowed)
	copy.Terminal = mcpAutoprogrammingScopedCandidatesV0(queue.Terminal, allowed)
	if len(copy.Ranked) == 0 && len(copy.Terminal) == 0 {
		return nil
	}
	copy.Count = len(copy.Ranked)
	if copy.Updated != nil && !allowed[strings.TrimSpace(copy.Updated.RunRef)] {
		copy.Updated = nil
	}
	return &copy
}
func mcpAutoprogrammingScopedCandidatesV0(values []MCPRunQueueRankedCandidateCompactV0, allowed map[string]bool) []MCPRunQueueRankedCandidateCompactV0 {
	out := []MCPRunQueueRankedCandidateCompactV0{}
	for _, value := range values {
		if allowed[strings.TrimSpace(value.RunRef)] {
			out = append(out, value)
		}
	}
	return out
}
func mcpAutoprogrammingScopedRunV0(run *MCPDirectorStatsToolResultV0, allowed map[string]bool) *MCPDirectorStatsToolResultV0 {
	if run == nil || run.Stats == nil || !allowed[strings.TrimSpace(firstNonEmptyMCPV0(run.RunRef, run.Stats.RunRef))] {
		return nil
	}
	return run
}
func mcpAutoprogrammingScopedProjectsV0(values []MCPAutoprogrammingProjectV0, allowed map[string]bool) []MCPAutoprogrammingProjectV0 {
	out := []MCPAutoprogrammingProjectV0{}
	for _, value := range values {
		value.RunRefs = mcpAutoprogrammingScopedRefsV0(value.RunRefs, allowed)
		if len(value.RunRefs) > 0 {
			out = append(out, value)
		}
	}
	return out
}
func mcpAutoprogrammingScopedTasksV0(values []MCPAutoprogrammingTaskV0, allowed map[string]bool) []MCPAutoprogrammingTaskV0 {
	out := []MCPAutoprogrammingTaskV0{}
	for _, value := range values {
		if allowed[strings.TrimSpace(value.RunRef)] {
			out = append(out, value)
		}
	}
	return out
}
func mcpAutoprogrammingScopedAgentsV0(values []MCPAutoprogrammingAgentV0, allowed map[string]bool) []MCPAutoprogrammingAgentV0 {
	out := []MCPAutoprogrammingAgentV0{}
	for _, value := range values {
		if allowed[strings.TrimSpace(value.RunRef)] {
			out = append(out, value)
		}
	}
	return out
}
func mcpAutoprogrammingScopedActionableRunsV0(values []MCPAutoprogrammingActionableRunV0, allowed map[string]bool) []MCPAutoprogrammingActionableRunV0 {
	out := []MCPAutoprogrammingActionableRunV0{}
	for _, value := range values {
		if allowed[strings.TrimSpace(value.RunRef)] {
			out = append(out, value)
		}
	}
	return out
}
func mcpAutoprogrammingScopedRefsV0(values []string, allowed map[string]bool) []string {
	out := []string{}
	for _, value := range values {
		if allowed[strings.TrimSpace(value)] {
			out = append(out, value)
		}
	}
	return out
}

func mcpAutoprogrammingScopedDiagnosticsV0(values []MCPAutoprogrammingDiagnosticV0, allowed map[string]bool, mode string) []MCPAutoprogrammingDiagnosticV0 {
	out := []MCPAutoprogrammingDiagnosticV0{}
	for _, value := range values {
		if ref, ok := mcpAutoprogrammingRunRefFromScopeV0(value.Scope); ok {
			if !allowed[ref] {
				continue
			}
		} else if mode == MCPAutoprogrammingStatusScopeRunV0 && strings.TrimSpace(value.Scope) == "queue" {
			continue
		}
		out = append(out, value)
	}
	return out
}
func mcpAutoprogrammingRunRefFromScopeV0(scope string) (string, bool) {
	for _, token := range strings.Fields(scope) {
		if strings.HasPrefix(token, "run:") {
			ref := strings.TrimSpace(strings.TrimPrefix(token, "run:"))
			return ref, ref != ""
		}
	}
	return "", false
}

func mcpAutoprogrammingScopedOperatorV0(operator *MCPAutoprogrammingOperatorV0, allowed map[string]bool, mode string) *MCPAutoprogrammingOperatorV0 {
	if operator == nil {
		return nil
	}
	copy := *operator
	copy.ActiveRuns = []MCPAutoprogrammingActiveRunV0{}
	for _, item := range operator.ActiveRuns {
		if allowed[strings.TrimSpace(item.RunRef)] {
			copy.ActiveRuns = append(copy.ActiveRuns, item)
		}
	}
	copy.ClosureBlockers = []MCPAutoprogrammingClosureBlockerV0{}
	for _, item := range operator.ClosureBlockers {
		if allowed[strings.TrimSpace(item.RunRef)] {
			copy.ClosureBlockers = append(copy.ClosureBlockers, item)
		}
	}
	copy.AgentsInFlight = []MCPAutoprogrammingAgentInFlightV0{}
	for _, item := range operator.AgentsInFlight {
		if allowed[strings.TrimSpace(item.RunRef)] {
			copy.AgentsInFlight = append(copy.AgentsInFlight, item)
		}
	}
	copy.SupervisorErrors = []MCPAutoprogrammingSupervisorErrorV0{}
	for _, item := range operator.SupervisorErrors {
		if ref, ok := mcpAutoprogrammingRunRefFromScopeV0(item.Scope); !ok || allowed[ref] {
			copy.SupervisorErrors = append(copy.SupervisorErrors, item)
		}
	}
	copy.SafeActions = []MCPAutoprogrammingSafeActionV0{}
	for _, action := range operator.SafeActions {
		if mcpAutoprogrammingScopedSafeActionV0(action, allowed, mode) {
			copy.SafeActions = append(copy.SafeActions, action)
		}
	}
	copy.QueueLive = mode == MCPAutoprogrammingStatusScopeQueueV0 && len(copy.ActiveRuns) > 0
	return &copy
}
func mcpAutoprogrammingScopedSafeActionV0(action MCPAutoprogrammingSafeActionV0, allowed map[string]bool, mode string) bool {
	if ref := strings.TrimSpace(action.RunRef); ref != "" && !allowed[ref] {
		return false
	}
	if ref, ok := mcpAutoprogrammingRunRefFromScopeV0(action.Scope); ok && !allowed[ref] {
		return false
	}
	if strings.TrimSpace(action.RunRef) == "" && strings.TrimSpace(action.Scope) == "queue" && mode != MCPAutoprogrammingStatusScopeQueueV0 {
		return false
	}
	_, ok := mcpAutoprogrammingScopedPayloadV0(action.Payload, allowed)
	return ok
}
func mcpAutoprogrammingScopedPayloadV0(value any, allowed map[string]bool) (any, bool) {
	switch typed := value.(type) {
	case nil:
		return nil, true
	case map[string]any:
		out := map[string]any{}
		for key, nested := range typed {
			normalized, ok := mcpAutoprogrammingScopedPayloadV0(nested, allowed)
			if !ok {
				return nil, false
			}
			switch strings.ToLower(strings.TrimSpace(key)) {
			case "run_ref":
				ref, ok := normalized.(string)
				if !ok || !allowed[strings.TrimSpace(ref)] {
					return nil, false
				}
			case "run_refs":
				if !mcpAutoprogrammingPayloadRunRefsAllowedV0(normalized, allowed) {
					return nil, false
				}
			}
			out[key] = normalized
		}
		return out, true
	case []any:
		out := make([]any, 0, len(typed))
		for _, nested := range typed {
			normalized, ok := mcpAutoprogrammingScopedPayloadV0(nested, allowed)
			if !ok {
				return nil, false
			}
			out = append(out, normalized)
		}
		return out, true
	case []string:
		out := make([]string, 0, len(typed))
		for _, nested := range typed {
			out = append(out, nested)
		}
		return out, true
	default:
		return value, true
	}
}
func mcpAutoprogrammingPayloadRunRefsAllowedV0(value any, allowed map[string]bool) bool {
	switch refs := value.(type) {
	case []string:
		for _, ref := range refs {
			if !allowed[strings.TrimSpace(ref)] {
				return false
			}
		}
		return true
	case []any:
		for _, raw := range refs {
			ref, ok := raw.(string)
			if !ok || !allowed[strings.TrimSpace(ref)] {
				return false
			}
		}
		return true
	default:
		return false
	}
}
