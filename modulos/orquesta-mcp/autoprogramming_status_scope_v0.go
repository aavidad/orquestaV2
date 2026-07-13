package orquestamcp

import (
	"reflect"
	"strings"
)

const (
	MCPAutoprogrammingStatusScopeLegacyV0 = "legacy"
	MCPAutoprogrammingStatusScopeRunV0    = "run"
	MCPAutoprogrammingStatusScopeQueueV0  = "queue"
	MCPAutoprogrammingStatusScopeAppV0    = "app"
)

// scopeMCPAutoprogrammingStatusResultV0 is deliberately the final projection.
// authoritativeKnownRuns contains only identities exposed by first-class status
// records; it must never be extended with parent, evidence, or payload refs.
func scopeMCPAutoprogrammingStatusResultV0(result MCPAutoprogrammingStatusToolResultV0, input MCPAutoprogrammingStatusToolInputV0) MCPAutoprogrammingStatusToolResultV0 {
	mode, scope := mcpAutoprogrammingStatusScopeV0(input)
	result.ScopeMode, result.Scope = mode, scope
	if mode == MCPAutoprogrammingStatusScopeLegacyV0 {
		return result
	}
	known, apps := mcpAutoprogrammingAuthoritativeKnownRunsV0(result)
	allowed := map[string]bool{}
	switch mode {
	case MCPAutoprogrammingStatusScopeRunV0:
		if known[scope] {
			allowed[scope] = true
		}
	case MCPAutoprogrammingStatusScopeQueueV0:
		if result.Queue != nil && strings.TrimSpace(result.Queue.QueueRef) == scope {
			for _, item := range append(append([]MCPRunQueueRankedCandidateCompactV0{}, result.Queue.Ranked...), result.Queue.Terminal...) {
				if ref := strings.TrimSpace(item.RunRef); known[ref] {
					allowed[ref] = true
				}
			}
		}
	case MCPAutoprogrammingStatusScopeAppV0:
		for ref := range known {
			if apps[ref][scope] {
				allowed[ref] = true
			}
		}
	}
	result.Queue = mcpAutoprogrammingScopedQueueV0(result.Queue, allowed, known)
	if result.Queue == nil {
		result.QueueRef = ""
	}
	result.Run = mcpAutoprogrammingScopedRunV0(result.Run, allowed)
	if result.Run == nil {
		result.RunRef, result.CausalVerdict, result.CausalReasonCode = "", "", ""
	}
	result.Projects = mcpAutoprogrammingScopedProjectsV0(result.Projects, allowed)
	result.Tasks = mcpAutoprogrammingScopedTasksV0(result.Tasks, allowed, known)
	result.Agents = mcpAutoprogrammingScopedAgentsV0(result.Agents, allowed, known)
	result.StaleRunning = mcpAutoprogrammingScopedActionableRunsV0(result.StaleRunning, allowed, known)
	result.ResolvedRuns = mcpAutoprogrammingScopedActionableRunsV0(result.ResolvedRuns, allowed, known)
	result.Diagnostics = mcpAutoprogrammingScopedDiagnosticsV0(result.Diagnostics, allowed, known, mode)
	result.Operator = mcpAutoprogrammingScopedOperatorV0(result.Operator, allowed, known, mode)
	result.QueueHealth, result.EfficiencySummary, result.OpsSnapshot = nil, nil, nil
	result.StaleRunningTotal, result.DiagnosticsTotal = len(result.StaleRunning), len(result.Diagnostics)
	return result
}

func mcpAutoprogrammingStatusScopeV0(input MCPAutoprogrammingStatusToolInputV0) (string, string) {
	mode, scope := strings.ToLower(strings.TrimSpace(input.ScopeMode)), strings.TrimSpace(input.Scope)
	if mode == "" {
		return MCPAutoprogrammingStatusScopeLegacyV0, ""
	}
	return mode, scope
}

func mcpAutoprogrammingStatusScopeIssueV0(input MCPAutoprogrammingStatusToolInputV0) *MCPValidationIssueV0 {
	mode, scope := mcpAutoprogrammingStatusScopeV0(input)
	if strings.TrimSpace(input.ScopeMode) == "" {
		if strings.TrimSpace(input.Scope) != "" {
			return &MCPValidationIssueV0{Code: "autoprogramming_status_scope_invalid", Field: "scope_mode", Message: "scope requiere scope_mode"}
		}
		return nil // pre-selector callers remain legacy-compatible.
	}
	if mode != MCPAutoprogrammingStatusScopeLegacyV0 && mode != MCPAutoprogrammingStatusScopeRunV0 && mode != MCPAutoprogrammingStatusScopeQueueV0 && mode != MCPAutoprogrammingStatusScopeAppV0 {
		return &MCPValidationIssueV0{Code: "autoprogramming_status_scope_invalid", Field: "scope_mode", Message: "scope_mode no soportado"}
	}
	if (mode == MCPAutoprogrammingStatusScopeLegacyV0 && scope != "") || (mode != MCPAutoprogrammingStatusScopeLegacyV0 && scope == "") {
		return &MCPValidationIssueV0{Code: "autoprogramming_status_scope_invalid", Field: "scope", Message: "selector scope incompleto o invalido"}
	}
	return nil
}

func mcpAutoprogrammingAuthoritativeKnownRunsV0(result MCPAutoprogrammingStatusToolResultV0) (map[string]bool, map[string]map[string]bool) {
	known, apps := map[string]bool{}, map[string]map[string]bool{}
	add := func(ref, app string) {
		ref, app = strings.TrimSpace(ref), strings.TrimSpace(app)
		if ref == "" {
			return
		}
		known[ref] = true
		if app != "" {
			if apps[ref] == nil {
				apps[ref] = map[string]bool{}
			}
			apps[ref][app] = true
		}
	}
	add(result.RunRef, "")
	if result.Run != nil {
		add(result.Run.RunRef, "")
		if result.Run.Stats != nil {
			add(result.Run.Stats.RunRef, result.Run.Stats.ProjectRef)
		}
	}
	if result.Queue != nil {
		for _, item := range append(append([]MCPRunQueueRankedCandidateCompactV0{}, result.Queue.Ranked...), result.Queue.Terminal...) {
			add(item.RunRef, item.AppRef)
		}
	}
	for _, item := range result.Projects {
		for _, ref := range item.RunRefs {
			add(ref, item.AppRef)
		}
	}
	for _, item := range result.Tasks {
		add(item.RunRef, "")
	}
	for _, item := range result.Agents {
		add(item.RunRef, "")
	}
	for _, item := range append(append([]MCPAutoprogrammingActionableRunV0{}, result.StaleRunning...), result.ResolvedRuns...) {
		add(item.RunRef, item.AppRef)
	}
	if result.Operator != nil {
		for _, item := range result.Operator.ActiveRuns {
			add(item.RunRef, item.AppRef)
		}
		for _, item := range result.Operator.ClosureBlockers {
			add(item.RunRef, "")
		}
		for _, item := range result.Operator.AgentsInFlight {
			add(item.RunRef, "")
		}
		for _, item := range result.Operator.SafeActions {
			add(item.RunRef, "")
		}
	}
	return known, apps
}

func mcpAutoprogrammingScopedQueueV0(queue *MCPRunQueuePriorityToolResultV0, allowed, known map[string]bool) *MCPRunQueuePriorityToolResultV0 {
	if queue == nil {
		return nil
	}
	copy := *queue
	copy.Ranked = mcpAutoprogrammingScopedCandidatesV0(queue.Ranked, allowed, known)
	copy.Terminal = mcpAutoprogrammingScopedCandidatesV0(queue.Terminal, allowed, known)
	if len(copy.Ranked) == 0 && len(copy.Terminal) == 0 {
		return nil
	}
	copy.Count = len(copy.Ranked)
	if copy.Updated != nil && !allowed[strings.TrimSpace(copy.Updated.RunRef)] {
		copy.Updated = nil
	} else if copy.Updated != nil {
		updated := mcpAutoprogrammingSanitizedCandidateV0(*copy.Updated, allowed, known)
		copy.Updated = &updated
	}
	return &copy
}
func mcpAutoprogrammingScopedCandidatesV0(values []MCPRunQueueRankedCandidateCompactV0, allowed, known map[string]bool) []MCPRunQueueRankedCandidateCompactV0 {
	out := []MCPRunQueueRankedCandidateCompactV0{}
	for _, value := range values {
		if allowed[strings.TrimSpace(value.RunRef)] {
			out = append(out, mcpAutoprogrammingSanitizedCandidateV0(value, allowed, known))
		}
	}
	return out
}
func mcpAutoprogrammingSanitizedCandidateV0(value MCPRunQueueRankedCandidateCompactV0, allowed, known map[string]bool) MCPRunQueueRankedCandidateCompactV0 {
	value.ParentRunRef = mcpAutoprogrammingScopedOptionalRunRefV0(value.ParentRunRef, allowed, known)
	value.SupersedesRunRef = mcpAutoprogrammingScopedOptionalRunRefV0(value.SupersedesRunRef, allowed, known)
	value.EvidenceRefs = mcpAutoprogrammingScopedSemanticRefsV0(value.EvidenceRefs, allowed, known)
	return value
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
func mcpAutoprogrammingScopedTasksV0(values []MCPAutoprogrammingTaskV0, allowed, known map[string]bool) []MCPAutoprogrammingTaskV0 {
	out := []MCPAutoprogrammingTaskV0{}
	for _, value := range values {
		if allowed[strings.TrimSpace(value.RunRef)] {
			value.EvidenceRefs = mcpAutoprogrammingScopedSemanticRefsV0(value.EvidenceRefs, allowed, known)
			out = append(out, value)
		}
	}
	return out
}
func mcpAutoprogrammingScopedAgentsV0(values []MCPAutoprogrammingAgentV0, allowed, known map[string]bool) []MCPAutoprogrammingAgentV0 {
	out := []MCPAutoprogrammingAgentV0{}
	for _, value := range values {
		if allowed[strings.TrimSpace(value.RunRef)] {
			value.EvidenceRefs = mcpAutoprogrammingScopedSemanticRefsV0(value.EvidenceRefs, allowed, known)
			out = append(out, value)
		}
	}
	return out
}
func mcpAutoprogrammingScopedActionableRunsV0(values []MCPAutoprogrammingActionableRunV0, allowed, known map[string]bool) []MCPAutoprogrammingActionableRunV0 {
	out := []MCPAutoprogrammingActionableRunV0{}
	for _, value := range values {
		if allowed[strings.TrimSpace(value.RunRef)] {
			value.SampleRefs = mcpAutoprogrammingScopedSemanticRefsV0(value.SampleRefs, allowed, known)
			value.EvidenceRefs = mcpAutoprogrammingScopedSemanticRefsV0(value.EvidenceRefs, allowed, known)
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
func mcpAutoprogrammingScopedOptionalRunRefV0(value string, allowed, known map[string]bool) string {
	value = strings.TrimSpace(value)
	if known[value] && !allowed[value] {
		return ""
	}
	return value
}
func mcpAutoprogrammingScopedSemanticRefsV0(values []string, allowed, known map[string]bool) []string {
	out := []string{}
	for _, value := range values {
		if value = mcpAutoprogrammingScopedOptionalRunRefV0(value, allowed, known); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func mcpAutoprogrammingScopedDiagnosticsV0(values []MCPAutoprogrammingDiagnosticV0, allowed, known map[string]bool, mode string) []MCPAutoprogrammingDiagnosticV0 {
	out := []MCPAutoprogrammingDiagnosticV0{}
	for _, value := range values {
		if ref, ok := mcpAutoprogrammingRunRefFromScopeV0(value.Scope); ok && !allowed[ref] {
			continue
		}
		if _, ok := mcpAutoprogrammingRunRefFromScopeV0(value.Scope); !ok && mode != MCPAutoprogrammingStatusScopeQueueV0 && strings.TrimSpace(value.Scope) == "queue" {
			continue
		}
		value.SampleRefs = mcpAutoprogrammingScopedSemanticRefsV0(value.SampleRefs, allowed, known)
		value.EvidenceRefs = mcpAutoprogrammingScopedSemanticRefsV0(value.EvidenceRefs, allowed, known)
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
func mcpAutoprogrammingScopedOperatorV0(operator *MCPAutoprogrammingOperatorV0, allowed, known map[string]bool, mode string) *MCPAutoprogrammingOperatorV0 {
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
			item.Evidence = mcpAutoprogrammingScopedSemanticRefsV0(item.Evidence, allowed, known)
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
		if strings.TrimSpace(item.Scope) == "queue" && mode != MCPAutoprogrammingStatusScopeQueueV0 {
			continue
		}
		if ref, ok := mcpAutoprogrammingRunRefFromScopeV0(item.Scope); !ok || allowed[ref] {
			copy.SupervisorErrors = append(copy.SupervisorErrors, item)
		}
	}
	copy.SafeActions = []MCPAutoprogrammingSafeActionV0{}
	for _, action := range operator.SafeActions {
		if sanitized, ok := mcpAutoprogrammingScopedSafeActionV0(action, allowed, known, mode); ok {
			copy.SafeActions = append(copy.SafeActions, sanitized)
		}
	}
	copy.QueueLive = mode == MCPAutoprogrammingStatusScopeQueueV0 && len(copy.ActiveRuns) > 0
	return &copy
}
func mcpAutoprogrammingScopedSafeActionV0(action MCPAutoprogrammingSafeActionV0, allowed, known map[string]bool, mode string) (MCPAutoprogrammingSafeActionV0, bool) {
	if ref := strings.TrimSpace(action.RunRef); ref != "" && !allowed[ref] {
		return action, false
	}
	if ref, ok := mcpAutoprogrammingRunRefFromScopeV0(action.Scope); ok && !allowed[ref] {
		return action, false
	}
	if strings.TrimSpace(action.RunRef) == "" && strings.TrimSpace(action.Scope) == "queue" && mode != MCPAutoprogrammingStatusScopeQueueV0 {
		return action, false
	}
	payload, ok := mcpAutoprogrammingScopedPayloadV0(action.Payload, allowed, known)
	if !ok {
		return action, false
	}
	action.Payload = payload.(map[string]any)
	return action, true
}

// Payloads may contain maps, slices, pointers or typed Go structs. A semantic
// run reference can only survive when it is allowed; unknown opaque values stay
// opaque rather than being guessed from a prefix.
func mcpAutoprogrammingScopedPayloadV0(value any, allowed, known map[string]bool) (any, bool) {
	out, ok := mcpAutoprogrammingScopedPayloadReflectV0(reflect.ValueOf(value), "", allowed, known)
	if !ok {
		return nil, false
	}
	return out.Interface(), true
}
func mcpAutoprogrammingScopedPayloadReflectV0(value reflect.Value, key string, allowed, known map[string]bool) (reflect.Value, bool) {
	if !value.IsValid() {
		return value, true
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return value, true
		}
		out, ok := mcpAutoprogrammingScopedPayloadReflectV0(value.Elem(), key, allowed, known)
		if !ok {
			return reflect.Value{}, false
		}
		boxed := reflect.New(value.Type()).Elem()
		boxed.Set(out)
		return boxed, true
	}
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return value, true
		}
		out, ok := mcpAutoprogrammingScopedPayloadReflectV0(value.Elem(), key, allowed, known)
		if !ok {
			return reflect.Value{}, false
		}
		copy := reflect.New(value.Type().Elem())
		copy.Elem().Set(out)
		return copy, true
	}
	normalized := strings.ToLower(strings.TrimSpace(key))
	semantic := value
	for semantic.IsValid() && semantic.Kind() == reflect.Interface && !semantic.IsNil() {
		semantic = semantic.Elem()
	}
	if normalized == "run_ref" && semantic.IsValid() && semantic.Kind() == reflect.String {
		if !allowed[strings.TrimSpace(semantic.String())] {
			return reflect.Value{}, false
		}
		return value, true
	}
	if normalized == "run_refs" {
		if semantic.Kind() != reflect.Slice && semantic.Kind() != reflect.Array {
			return reflect.Value{}, false
		}
		for i := 0; i < semantic.Len(); i++ {
			item := semantic.Index(i)
			for item.IsValid() && item.Kind() == reflect.Interface && !item.IsNil() {
				item = item.Elem()
			}
			if !item.IsValid() || item.Kind() != reflect.String || !allowed[strings.TrimSpace(item.String())] {
				return reflect.Value{}, false
			}
		}
		return value, true
	}
	switch value.Kind() {
	case reflect.Map:
		if value.Type().Key().Kind() != reflect.String {
			return value, true
		}
		copy := reflect.MakeMapWithSize(value.Type(), value.Len())
		it := value.MapRange()
		for it.Next() {
			out, ok := mcpAutoprogrammingScopedPayloadReflectV0(it.Value(), it.Key().String(), allowed, known)
			if !ok {
				return reflect.Value{}, false
			}
			copy.SetMapIndex(it.Key(), out)
		}
		return copy, true
	case reflect.Slice:
		copy := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := 0; i < value.Len(); i++ {
			out, ok := mcpAutoprogrammingScopedPayloadReflectV0(value.Index(i), "", allowed, known)
			if !ok {
				return reflect.Value{}, false
			}
			copy.Index(i).Set(out)
		}
		return copy, true
	case reflect.Struct:
		copy := reflect.New(value.Type()).Elem()
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)
			if !copy.Field(i).CanSet() {
				copy.Field(i).Set(value.Field(i))
				continue
			}
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "" {
				name = field.Name
			}
			out, ok := mcpAutoprogrammingScopedPayloadReflectV0(value.Field(i), name, allowed, known)
			if !ok {
				return reflect.Value{}, false
			}
			copy.Field(i).Set(out)
		}
		return copy, true
	}
	return value, true
}
