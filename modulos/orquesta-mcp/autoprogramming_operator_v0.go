package orquestamcp

import "strings"

type MCPAutoprogrammingOperatorV0 struct {
	QueueLive        bool                                  `json:"queue_live"`
	ActiveRuns       []MCPAutoprogrammingActiveRunV0       `json:"active_runs,omitempty"`
	ClosureBlockers  []MCPAutoprogrammingClosureBlockerV0  `json:"closure_blockers,omitempty"`
	AgentsInFlight   []MCPAutoprogrammingAgentInFlightV0   `json:"agents_in_flight,omitempty"`
	SupervisorErrors []MCPAutoprogrammingSupervisorErrorV0 `json:"supervisor_errors,omitempty"`
	SafeActions      []MCPAutoprogrammingSafeActionV0      `json:"safe_actions,omitempty"`
}

type MCPAutoprogrammingActiveRunV0 struct {
	RunRef        string `json:"run_ref"`
	AppRef        string `json:"app_ref,omitempty"`
	Status        string `json:"status,omitempty"`
	PriorityScore int    `json:"priority_score,omitempty"`
}

type MCPAutoprogrammingClosureBlockerV0 struct {
	Reason     string   `json:"reason,omitempty"`
	BlockerRef string   `json:"blocker_ref,omitempty"`
	RunRef     string   `json:"run_ref,omitempty"`
	Evidence   []string `json:"evidence_refs,omitempty"`
}

type MCPAutoprogrammingAgentInFlightV0 struct {
	AgentRef       string `json:"agent_ref"`
	RunRef         string `json:"run_ref,omitempty"`
	Status         string `json:"status,omitempty"`
	NeedsAttention bool   `json:"needs_attention,omitempty"`
	CanStop        bool   `json:"can_stop,omitempty"`
}

type MCPAutoprogrammingSupervisorErrorV0 struct {
	Code    string `json:"code"`
	Scope   string `json:"scope,omitempty"`
	Message string `json:"message,omitempty"`
}

type MCPAutoprogrammingSafeActionV0 struct {
	Action       string         `json:"action"`
	Scope        string         `json:"scope,omitempty"`
	RunRef       string         `json:"run_ref,omitempty"`
	Method       string         `json:"method"`
	Endpoint     string         `json:"endpoint"`
	Reason       string         `json:"reason,omitempty"`
	RequiresPost bool           `json:"requires_post"`
	Payload      map[string]any `json:"payload,omitempty"`
}

func newMCPAutoprogrammingOperatorV0(
	queue *MCPRunQueuePriorityToolResultV0,
	run *MCPDirectorStatsToolResultV0,
	diagnostics []MCPAutoprogrammingDiagnosticV0,
	allowLegacySupervisorActions bool,
) *MCPAutoprogrammingOperatorV0 {
	operator := &MCPAutoprogrammingOperatorV0{
		ActiveRuns:      mcpAutoprogrammingActiveRunsV0(queue),
		ClosureBlockers: mcpAutoprogrammingClosureBlockersV0(run),
		AgentsInFlight:  mcpAutoprogrammingAgentsInFlightV0(run),
		SupervisorErrors: mcpAutoprogrammingSupervisorErrorsV0(
			diagnostics,
		),
	}
	operator.QueueLive = queue != nil && queue.Estado == MCPRunQueuePriorityEstadoOKV0
	operator.SafeActions = mcpAutoprogrammingSafeActionsV0(operator, run, allowLegacySupervisorActions)
	return operator
}

func mcpAutoprogrammingActiveRunsV0(
	queue *MCPRunQueuePriorityToolResultV0,
) []MCPAutoprogrammingActiveRunV0 {
	if queue == nil {
		return nil
	}
	out := make([]MCPAutoprogrammingActiveRunV0, 0, len(queue.Ranked))
	for _, item := range queue.Ranked {
		if mcpAutoprogrammingTerminalRunV0(item.Status) {
			continue
		}
		out = append(out, MCPAutoprogrammingActiveRunV0{
			RunRef:        strings.TrimSpace(item.RunRef),
			AppRef:        strings.TrimSpace(item.AppRef),
			Status:        strings.TrimSpace(item.Status),
			PriorityScore: item.PriorityScore,
		})
	}
	return out
}

func mcpAutoprogrammingClosureBlockersV0(
	run *MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingClosureBlockerV0 {
	if run == nil || run.Stats == nil || !run.Stats.Closure.Blocked {
		return nil
	}
	reasons := compactStringsMCPV0(run.Stats.Closure.BlockedBy)
	refs := compactStringsMCPV0(run.Stats.Closure.BlockerRefs)
	total := maxMCPAutoprogrammingV0(len(reasons), len(refs))
	out := make([]MCPAutoprogrammingClosureBlockerV0, 0, total)
	for i := 0; i < total; i++ {
		out = append(out, MCPAutoprogrammingClosureBlockerV0{
			Reason:     mcpAutoprogrammingAtV0(reasons, i),
			BlockerRef: mcpAutoprogrammingAtV0(refs, i),
			RunRef:     strings.TrimSpace(run.RunRef),
			Evidence:   refs,
		})
	}
	return out
}

func mcpAutoprogrammingAgentsInFlightV0(
	run *MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingAgentInFlightV0 {
	if run == nil || run.Stats == nil {
		return nil
	}
	out := []MCPAutoprogrammingAgentInFlightV0{}
	for _, agent := range run.Stats.Agents {
		if !agent.InFlight {
			continue
		}
		out = append(out, MCPAutoprogrammingAgentInFlightV0{
			AgentRef:       strings.TrimSpace(agent.AgentRequestID),
			RunRef:         strings.TrimSpace(run.RunRef),
			Status:         strings.TrimSpace(agent.Status),
			NeedsAttention: agent.NeedsAttention,
			CanStop:        agent.CanStop,
		})
	}
	return out
}

func mcpAutoprogrammingSupervisorErrorsV0(
	diagnostics []MCPAutoprogrammingDiagnosticV0,
) []MCPAutoprogrammingSupervisorErrorV0 {
	out := []MCPAutoprogrammingSupervisorErrorV0{}
	for _, item := range diagnostics {
		code := strings.TrimSpace(item.Code)
		if !strings.HasSuffix(code, "_error") &&
			!strings.Contains(code, "supervisor") {
			continue
		}
		out = append(out, MCPAutoprogrammingSupervisorErrorV0{
			Code:    code,
			Scope:   strings.TrimSpace(item.Scope),
			Message: strings.TrimSpace(item.Message),
		})
	}
	return out
}

func mcpAutoprogrammingSafeActionsV0(
	operator *MCPAutoprogrammingOperatorV0,
	run *MCPDirectorStatsToolResultV0,
	allowLegacySupervisorActions bool,
) []MCPAutoprogrammingSafeActionV0 {
	out := []MCPAutoprogrammingSafeActionV0{}
	if operator == nil {
		return out
	}
	if !allowLegacySupervisorActions {
		return out
	}
	blockedByReplanAmplification := mcpAutoprogrammingReplanAmplificationBlockedV0(run)
	queueNeedsRunStats := mcpAutoprogrammingOperatorHasDiagnosticV0(
		operator.SupervisorErrors,
		"run_stats_required_for_safe_supervision",
	)
	if !blockedByReplanAmplification && !queueNeedsRunStats && len(operator.ActiveRuns) > 0 {
		out = append(out, mcpAutoprogrammingSafeActionV0("supervise", "queue", ""))
	}
	runRef := ""
	if run != nil {
		runRef = strings.TrimSpace(run.RunRef)
	}
	if runRef != "" && !blockedByReplanAmplification {
		out = append(out, mcpAutoprogrammingSafeActionV0("supervise", "run", runRef))
	}
	if len(operator.ClosureBlockers) > 0 && !blockedByReplanAmplification {
		out = append(out, mcpAutoprogrammingSafeActionV0("review", "closure", runRef))
	}
	if !blockedByReplanAmplification && !queueNeedsRunStats && (len(operator.SupervisorErrors) > 0 || mcpAutoprogrammingNeedsRetryV0(operator)) {
		out = append(out, mcpAutoprogrammingSafeActionV0("retry", "supervisor", runRef))
	}
	return out
}

func mcpAutoprogrammingReplanAmplificationBlockedV0(
	run *MCPDirectorStatsToolResultV0,
) bool {
	if run == nil || run.Stats == nil {
		return false
	}
	counts := run.Stats.Counts
	if counts.ReplanDecisions < 3 || counts.AgentsStarted < 3 {
		return false
	}
	if counts.Deliveries > 0 || counts.Reviews > 0 || counts.ReviewResults > 0 || counts.Closures > 0 {
		return false
	}
	return counts.AgentsStopRequested >= counts.ReplanDecisions-1 && counts.AgentsStarted > counts.AgentsDelivered
}

func mcpAutoprogrammingQueueEmptyOrNotVisibleV0(
	queue *MCPRunQueuePriorityToolResultV0,
	run *MCPDirectorStatsToolResultV0,
) bool {
	if queue == nil || queue.Estado != MCPRunQueuePriorityEstadoOKV0 || len(queue.Ranked) > 0 {
		return false
	}
	return run == nil || run.Stats == nil || strings.TrimSpace(run.RunRef) == ""
}

func mcpAutoprogrammingNeedsRunStatsForSafeSupervisionV0(
	queue *MCPRunQueuePriorityToolResultV0,
	run *MCPDirectorStatsToolResultV0,
) bool {
	if queue == nil || queue.Estado != MCPRunQueuePriorityEstadoOKV0 || len(queue.Ranked) == 0 {
		return false
	}
	if run != nil && run.Stats != nil && strings.TrimSpace(run.RunRef) != "" {
		return false
	}
	for _, item := range queue.Ranked {
		if strings.TrimSpace(item.RunRef) != "" && !mcpAutoprogrammingTerminalRunV0(item.Status) {
			return true
		}
	}
	return false
}

func mcpAutoprogrammingSafeActionV0(action string, scope string, runRef string) MCPAutoprogrammingSafeActionV0 {
	action = strings.TrimSpace(action)
	scope = strings.TrimSpace(scope)
	runRef = strings.TrimSpace(runRef)
	return MCPAutoprogrammingSafeActionV0{
		Action:       action,
		Scope:        scope,
		RunRef:       runRef,
		Method:       "POST",
		Endpoint:     MCPAutoprogrammingSuperviseHTTPPathV0,
		Reason:       scope,
		RequiresPost: true,
		Payload:      mcpAutoprogrammingSafeActionPayloadV0(action, scope, runRef),
	}
}

func mcpAutoprogrammingSafeActionPayloadV0(
	action string,
	scope string,
	runRef string,
) map[string]any {
	if action != "supervise" {
		return nil
	}
	switch scope {
	case "queue":
		return map[string]any{
			"director_execution_mode": "legacy_director_loop",
			"resident_mode":           true,
			"max_ticks":               24,
			"max_runs_per_tick":       70,
			"max_executions":          70,
			"max_dispatches_per_wait": 70,
			"max_outbox_per_cycle":    70,
		}
	case "run":
		if runRef == "" {
			return nil
		}
		return map[string]any{
			"director_execution_mode": "legacy_director_loop",
			"run_ref":                 runRef,
			"max_ticks":               4,
			"max_dispatches_per_wait": 16,
			"max_outbox_per_cycle":    16,
		}
	default:
		return nil
	}
}

func mcpAutoprogrammingObserveGoalSafeActionV0(runRef string) MCPAutoprogrammingSafeActionV0 {
	return MCPAutoprogrammingSafeActionV0{
		Action:       "observe_goal",
		Scope:        "run",
		RunRef:       strings.TrimSpace(runRef),
		Method:       "POST",
		Endpoint:     MCPAutoprogrammingObserveGoalHTTPPathV0,
		Reason:       "goal_first",
		RequiresPost: true,
	}
}

func mcpAutoprogrammingObserveActiveGoalsSafeActionV0(runRefs []string) MCPAutoprogrammingSafeActionV0 {
	return MCPAutoprogrammingSafeActionV0{
		Action:       "observe_active_goals",
		Scope:        "goals",
		Method:       "POST",
		Endpoint:     MCPAutoprogrammingObserveActiveGoalsHTTPPathV0,
		Reason:       "goal_first_batch",
		RequiresPost: true,
		Payload: map[string]any{
			"run_refs":     compactStringsMCPV0(runRefs),
			"requested_by": "orquesta-autoprogramming-status",
		},
	}
}

func mcpAutoprogrammingOperatorWithGoalFirstActionsV0(
	operator *MCPAutoprogrammingOperatorV0,
	allowLegacySupervisorActions bool,
	runRefs ...string,
) *MCPAutoprogrammingOperatorV0 {
	if operator == nil {
		return operator
	}
	goalFirst := mcpAutoprogrammingGoalFirstRunRefSetV0(runRefs)
	if len(goalFirst) == 0 {
		return operator
	}
	next := make([]MCPAutoprogrammingSafeActionV0, 0, len(operator.SafeActions)+len(goalFirst))
	if len(goalFirst) > 1 {
		next = append(next, mcpAutoprogrammingObserveActiveGoalsSafeActionV0(runRefs))
	}
	for _, runRef := range runRefs {
		runRef = strings.TrimSpace(runRef)
		if runRef == "" || !goalFirst[runRef] {
			continue
		}
		next = append(next, mcpAutoprogrammingObserveGoalSafeActionV0(runRef))
	}
	if allowLegacySupervisorActions {
		next = append(next, mcpAutoprogrammingLegacyRunActionsForActiveRunsV0(operator, goalFirst)...)
	}
	for _, action := range operator.SafeActions {
		if mcpAutoprogrammingGoalFirstSuppressesLegacyActionV0(operator, action, goalFirst) {
			continue
		}
		next = append(next, action)
	}
	operator.SafeActions = mcpAutoprogrammingDeduplicateSafeActionsV0(next)
	return operator
}

func mcpAutoprogrammingGoalFirstSuppressesLegacyActionV0(
	operator *MCPAutoprogrammingOperatorV0,
	action MCPAutoprogrammingSafeActionV0,
	goalFirst map[string]bool,
) bool {
	switch strings.TrimSpace(action.Action) {
	case "supervise", "retry", "review":
	default:
		return false
	}
	if goalFirst[strings.TrimSpace(action.RunRef)] {
		return true
	}
	return strings.TrimSpace(action.Action) == "supervise" &&
		strings.TrimSpace(action.Scope) == "queue" &&
		mcpAutoprogrammingQueueHasGoalFirstRunV0(operator, goalFirst)
}

func mcpAutoprogrammingGoalFirstRunRefSetV0(
	runRefs []string,
) map[string]bool {
	out := map[string]bool{}
	for _, runRef := range runRefs {
		runRef = strings.TrimSpace(runRef)
		if runRef != "" {
			out[runRef] = true
		}
	}
	return out
}

func mcpAutoprogrammingLegacySupervisorActionCandidateV0(
	operator *MCPAutoprogrammingOperatorV0,
	run *MCPDirectorStatsToolResultV0,
	goalFirst map[string]bool,
) bool {
	if operator == nil {
		return false
	}
	blockedByReplanAmplification := mcpAutoprogrammingReplanAmplificationBlockedV0(run)
	if blockedByReplanAmplification {
		return false
	}
	queueNeedsRunStats := mcpAutoprogrammingOperatorHasDiagnosticV0(
		operator.SupervisorErrors,
		"run_stats_required_for_safe_supervision",
	)
	if !queueNeedsRunStats {
		for _, active := range operator.ActiveRuns {
			runRef := strings.TrimSpace(active.RunRef)
			if runRef != "" && !goalFirst[runRef] {
				return true
			}
		}
	}
	runRef := ""
	if run != nil {
		runRef = strings.TrimSpace(run.RunRef)
	}
	if runRef != "" && !goalFirst[runRef] {
		return true
	}
	if len(operator.ClosureBlockers) > 0 && (runRef == "" || !goalFirst[runRef]) {
		return true
	}
	return !queueNeedsRunStats &&
		(len(operator.SupervisorErrors) > 0 || mcpAutoprogrammingNeedsRetryV0(operator))
}

func mcpAutoprogrammingQueueHasGoalFirstRunV0(
	operator *MCPAutoprogrammingOperatorV0,
	goalFirst map[string]bool,
) bool {
	if operator == nil || len(operator.ActiveRuns) == 0 {
		return false
	}
	for _, active := range operator.ActiveRuns {
		if goalFirst[strings.TrimSpace(active.RunRef)] {
			return true
		}
	}
	return false
}

func mcpAutoprogrammingLegacyRunActionsForActiveRunsV0(
	operator *MCPAutoprogrammingOperatorV0,
	goalFirst map[string]bool,
) []MCPAutoprogrammingSafeActionV0 {
	if operator == nil {
		return nil
	}
	out := []MCPAutoprogrammingSafeActionV0{}
	for _, active := range operator.ActiveRuns {
		runRef := strings.TrimSpace(active.RunRef)
		if runRef == "" || goalFirst[runRef] {
			continue
		}
		out = append(out, mcpAutoprogrammingSafeActionV0("supervise", "run", runRef))
		if len(out) >= 8 {
			break
		}
	}
	return out
}

func mcpAutoprogrammingDeduplicateSafeActionsV0(
	actions []MCPAutoprogrammingSafeActionV0,
) []MCPAutoprogrammingSafeActionV0 {
	seen := map[string]bool{}
	out := make([]MCPAutoprogrammingSafeActionV0, 0, len(actions))
	for _, action := range actions {
		key := strings.Join([]string{
			strings.TrimSpace(action.Action),
			strings.TrimSpace(action.Scope),
			strings.TrimSpace(action.RunRef),
			strings.TrimSpace(action.Endpoint),
		}, "|")
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, action)
	}
	return out
}

func mcpAutoprogrammingTerminalRunV0(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "closed", "cancelled", "canceled", "failed", "stopped":
		return true
	default:
		return false
	}
}

func mcpAutoprogrammingNeedsRetryV0(operator *MCPAutoprogrammingOperatorV0) bool {
	for _, agent := range operator.AgentsInFlight {
		if agent.NeedsAttention {
			return true
		}
	}
	return false
}

func mcpAutoprogrammingOperatorHasDiagnosticV0(
	errors []MCPAutoprogrammingSupervisorErrorV0,
	code string,
) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	for _, item := range errors {
		if strings.TrimSpace(item.Code) == code {
			return true
		}
	}
	return false
}

func mcpAutoprogrammingAtV0(values []string, index int) string {
	if index >= 0 && index < len(values) {
		return strings.TrimSpace(values[index])
	}
	return ""
}

func maxMCPAutoprogrammingV0(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
