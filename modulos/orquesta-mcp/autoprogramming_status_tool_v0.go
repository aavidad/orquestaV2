package orquestamcp

import (
	"context"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaobservability "orquesta/modulos/orquesta-observability"
)

const (
	MCPAutoprogrammingStatusToolNameV0    = "orquesta.autoprogramming.status.v0"
	MCPAutoprogrammingStatusToolVersionV0 = "v0"
	MCPAutoprogrammingStatusResourceURIV0 = "orquesta://contracts/autoprogramming-status/v0"
	MCPAutoprogrammingStatusEstadoOKV0    = "ok"
	MCPAutoprogrammingStatusEstadoErrorV0 = "error"

	mcpAutoprogrammingDomainSessionSuppressedReasonV0   = "idle_self_improvement_suppressed_by_domain_session"
	mcpAutoprogrammingDomainSessionSuppressedEvidenceV0 = "evidence-ref-idle-self-improvement-domain-session"
	mcpAutoprogrammingRunningStatsMaxV0                 = 8
)

type MCPAutoprogrammingStatusToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPAutoprogrammingStatusToolInputV0 struct {
	RequestID            string            `json:"request_id,omitempty"`
	CorrelationID        string            `json:"correlation_id,omitempty"`
	RunRef               string            `json:"run_ref,omitempty"`
	AppRef               string            `json:"app_ref,omitempty"`
	ExternalJobRef       string            `json:"external_job_ref,omitempty"`
	QueueRef             string            `json:"queue_ref,omitempty"`
	AppRefs              []string          `json:"app_refs,omitempty"`
	QueueLimit           int               `json:"queue_limit,omitempty"`
	OccurredAt           string            `json:"occurred_at,omitempty"`
	IncludeProcessRefs   mcpFlexibleBoolV0 `json:"include_process_refs,omitempty"`
	IncludeAgentProgress mcpFlexibleBoolV0 `json:"include_agent_progress,omitempty"`
	IncludeAgentUsage    mcpFlexibleBoolV0 `json:"include_agent_usage,omitempty"`
}

type MCPAutoprogrammingStatusToolResultV0 struct {
	Estado            string                                                 `json:"estado"`
	RequestID         string                                                 `json:"request_id,omitempty"`
	CorrelationID     string                                                 `json:"correlation_id,omitempty"`
	RunRef            string                                                 `json:"run_ref,omitempty"`
	QueueRef          string                                                 `json:"queue_ref,omitempty"`
	Queue             *MCPRunQueuePriorityToolResultV0                       `json:"queue,omitempty"`
	Run               *MCPDirectorStatsToolResultV0                          `json:"run,omitempty"`
	QueueHealth       *MCPAutoprogrammingQueueHealthV0                       `json:"queue_health,omitempty"`
	StaleRunning      []MCPAutoprogrammingActionableRunV0                    `json:"stale_running,omitempty"`
	ResolvedRuns      []MCPAutoprogrammingActionableRunV0                    `json:"resolved_runs,omitempty"`
	Projects          []MCPAutoprogrammingProjectV0                          `json:"projects,omitempty"`
	Tasks             []MCPAutoprogrammingTaskV0                             `json:"tasks,omitempty"`
	Agents            []MCPAutoprogrammingAgentV0                            `json:"agents,omitempty"`
	Operator          *MCPAutoprogrammingOperatorV0                          `json:"operator,omitempty"`
	EfficiencySummary *MCPAutoprogrammingEfficiencySummaryV0                 `json:"efficiency_summary,omitempty"`
	OpsSnapshot       *orquestaobservability.DirectorAutonomousOpsSnapshotV0 `json:"ops_snapshot,omitempty"`
	Diagnostics       []MCPAutoprogrammingDiagnosticV0                       `json:"diagnostics,omitempty"`
	Errores           []MCPValidationIssueV0                                 `json:"errores_publicos,omitempty"`
}

type MCPAutoprogrammingStatusToolExecutorV0 struct {
	Queue                        MCPTransportRunQueuePriorityExecutorV0
	Stats                        MCPTransportDirectorStatsExecutorV0
	GoalStateStore               orquestagoal.GoalWorkStateStorePortV0
	GoalRunMarkerStore           orquestagoal.GoalWorkRunMarkerStorePortV0
	StatusDiagnostics            []MCPAutoprogrammingDiagnosticV0
	GoalProgressPolicy           MCPAutoprogrammingGoalProgressPolicyV0
	AllowLegacySupervisorActions bool
}

type MCPAutoprogrammingGoalProgressPolicyV0 struct {
	CheckpointOnlyHighConsumptionTokens int64 `json:"checkpoint_only_high_consumption_tokens,omitempty"`
	CheckpointOnlyMaxWaitSeconds        int64 `json:"checkpoint_only_max_wait_seconds,omitempty"`
	NoCheckpointWarningMaxWaitSeconds   int64 `json:"no_checkpoint_warning_max_wait_seconds,omitempty"`
}

func NormalizeMCPAutoprogrammingGoalProgressPolicyV0(
	policy MCPAutoprogrammingGoalProgressPolicyV0,
) MCPAutoprogrammingGoalProgressPolicyV0 {
	if policy.CheckpointOnlyHighConsumptionTokens <= 0 {
		policy.CheckpointOnlyHighConsumptionTokens = mcpAutoprogrammingCheckpointOnlyHighConsumptionTokensDefaultV0
	}
	if policy.CheckpointOnlyMaxWaitSeconds <= 0 {
		policy.CheckpointOnlyMaxWaitSeconds = mcpAutoprogrammingCheckpointOnlyMaxWaitSecondsDefaultV0
	}
	if policy.NoCheckpointWarningMaxWaitSeconds <= 0 {
		policy.NoCheckpointWarningMaxWaitSeconds = mcpAutoprogrammingNoCheckpointWarningMaxWaitSecondsDefaultV0
	}
	return policy
}

func MCPAutoprogrammingStatusDescriptorV0() MCPAutoprogrammingStatusToolDescriptorV0 {
	return MCPAutoprogrammingStatusToolDescriptorV0{
		Name:        MCPAutoprogrammingStatusToolNameV0,
		Version:     MCPAutoprogrammingStatusToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,run_ref?,external_job_ref?,queue_ref?,app_refs?,queue_limit?,operator_advice?}",
		Output:      "ok:{queue?,run?,queue_health?,stale_running?,projects?,tasks?,agents?,operator?,efficiency_summary?,ops_snapshot?,diagnostics?}|error:{errores_publicos,diagnostics?,operator_advice?}",
		ResourceURI: MCPAutoprogrammingStatusResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"estado de cola via run_queue.priority inyectado",
			"estado de run via director.stats inyectado",
			"queue_health separa queued/running_live/running_stale/blocked/lost/completed/failed sin mutar cola",
			"stale_running lista runs accionables que no deben competir silenciosamente con olas nuevas",
			"proyecta proyectos tareas y agentes compactos para filtros externos",
			"diagnostico solo resume puertos y errores publicos",
			"acciones de supervision legacy solo aparecen si la composicion las habilita explicitamente",
			"operator_advice se conserva como observacion no bloqueante",
			"sin DB runtime filesystem Codex ni proveedor concreto",
		},
	}
}

func (executor MCPAutoprogrammingStatusToolExecutorV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingStatusToolInputV0,
) (MCPAutoprogrammingStatusToolResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result := newMCPAutoprogrammingStatusBaseV0(input)
	result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingStatusConfiguredDiagnosticsV0(executor.StatusDiagnostics)...)
	okCount := 0
	if executor.Queue != nil {
		queue, err := executor.Queue.Execute(ctx, mcpAutoprogrammingQueueInputV0(input))
		if err != nil {
			result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingDiagnosticV0("queue_error", "queue", "cola no disponible"))
		} else {
			result.Queue = &queue
			result.QueueRef = queue.QueueRef
			if queue.Estado == MCPRunQueuePriorityEstadoOKV0 {
				okCount++
			}
			result.Diagnostics = append(result.Diagnostics, diagnosticsFromIssuesMCPAutoprogrammingV0("queue", queue.Errores)...)
			result.Diagnostics = append(result.Diagnostics, diagnosticsFromSuppressedQueueMCPAutoprogrammingV0(&queue)...)
		}
	} else {
		result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingDiagnosticV0("queue_unbound", "queue", "run_queue no configurado"))
	}
	if wantsMCPAutoprogrammingRunStatusV0(input) {
		ok, run, diagnostics, err := executor.executeRunStatusV0(ctx, input)
		if err != nil {
			return result, err
		}
		result.Run = run
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		if ok {
			okCount++
			result.RunRef = run.RunRef
		}
	}
	if mcpAutoprogrammingReplanAmplificationBlockedV0(result.Run) {
		result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingDiagnosticV0(
			"supervisor_replan_amplification_blocked",
			"run",
			"supervision no recomendada: replan/stop repetidos sin entregas, reviews ni cierres",
		))
	}
	goalStates := executor.goalStatesForAutoprogrammingStatusV0(ctx, result, input)
	if len(goalStates) > 0 {
		okCount++
	}
	goalStatesByRunRef := mcpAutoprogrammingGoalStatesByRunRefV0(goalStates)
	goalRunMarkers := executor.goalRunMarkersForAutoprogrammingStatusV0(ctx, result, input, goalStatesByRunRef)
	if len(goalRunMarkers) > 0 {
		okCount++
	}
	goalRunMarkersByRunRef := mcpAutoprogrammingGoalRunMarkersByRunRefV0(goalRunMarkers)
	goalFirstRunRefs := mcpAutoprogrammingMergeGoalFirstRunRefSetsV0(
		mcpAutoprogrammingGoalFirstRunRefSetFromStatesV0(goalStates),
		mcpAutoprogrammingGoalFirstRunRefSetFromMarkersV0(goalRunMarkers),
	)
	observedRuns, observedDiagnostics := executor.executeQueueRunningStatsV0(ctx, input, result.Queue, result.Run, goalStatesByRunRef)
	result.Diagnostics = append(result.Diagnostics, observedDiagnostics...)
	result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingRunningStatsSampleLimitedDiagnosticsV0(result.Queue, executor.Stats != nil, result.Run, observedRuns...)...)
	for _, observed := range observedRuns {
		if observed == nil {
			continue
		}
		result.Diagnostics = append(result.Diagnostics, diagnosticsFromRunProgressIssuesMCPAutoprogrammingV0(observed.Stats)...)
	}
	if mcpAutoprogrammingNeedsRunStatsForSafeSupervisionV0(result.Queue, result.Run) &&
		!mcpAutoprogrammingQueueRunningStatsObservedV0(result.Queue, result.Run, observedRuns...) {
		result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingDiagnosticV0(
			"run_stats_required_for_safe_supervision",
			"queue",
			"supervision de cola no declarada segura sin consultar stats del run candidato",
		))
	}
	if mcpAutoprogrammingQueueEmptyOrNotVisibleV0(result.Queue, result.Run) {
		result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingDiagnosticV0(
			"queue_empty_or_not_visible",
			"queue",
			"cola sin candidatos visibles; no declarar supervision de cola como accion segura",
		))
	}
	healthRun, healthObservedRuns := mcpAutoprogrammingPreferredRunStatsForQueueHealthV0(result.Run, observedRuns...)
	result.Diagnostics = append(
		result.Diagnostics,
		diagnosticsFromQueuedNotDispatchedMCPAutoprogrammingV0(
			result.Queue,
			mcpAutoprogrammingGoalFirstRunStructSetV0(goalFirstRunRefs),
			append([]*MCPDirectorStatsToolResultV0{result.Run}, observedRuns...)...,
		)...,
	)
	if okCount == 0 {
		result.Estado = MCPAutoprogrammingStatusEstadoErrorV0
		result.Errores = []MCPValidationIssueV0{{
			Code:    "autoprogramming_status_no_disponible",
			Field:   "ports",
			Message: "estado de autoprogramacion no disponible",
		}}
	}
	result.Projects = buildMCPAutoprogrammingProjectsV0(result.Queue, result.Run)
	result.Tasks = buildMCPAutoprogrammingTasksV0(result.Run, goalFirstRunRefs)
	result.Agents = buildMCPAutoprogrammingAgentsV0(result.Run, goalFirstRunRefs)
	result.QueueHealth = buildMCPAutoprogrammingQueueHealthV0(result.Queue, goalStatesByRunRef, goalRunMarkersByRunRef, healthRun, healthObservedRuns...)
	result.StaleRunning = buildMCPAutoprogrammingStaleRunningV0(result.Queue, goalStatesByRunRef, goalRunMarkersByRunRef, healthRun, healthObservedRuns...)
	observedByRunRef := mcpAutoprogrammingObservedRunsByRefV0(healthRun, healthObservedRuns...)
	blockedGoalActions, resolvedGoalActions := mcpAutoprogrammingGoalFirstBlockedActionsV0(
		goalStates,
		observedByRunRef,
		executor.GoalProgressPolicy,
		input.OccurredAt,
	)
	result.StaleRunning = append(result.StaleRunning, blockedGoalActions...)
	result.ResolvedRuns = append(result.ResolvedRuns, resolvedGoalActions...)
	result.StaleRunning = append(result.StaleRunning, mcpAutoprogrammingQAFailedPublicTextActionsV0(observedByRunRef)...)
	result.StaleRunning = append(result.StaleRunning, mcpAutoprogrammingArtifactPathsOmittedActionsV0(observedByRunRef)...)
	result.StaleRunning = append(result.StaleRunning, mcpAutoprogrammingMissingTerminalReceiptActionsV0(observedByRunRef)...)
	result.StaleRunning = append(result.StaleRunning, mcpAutoprogrammingRequiredTestEvidenceMissingActionsV0(observedByRunRef)...)
	result.StaleRunning = append(result.StaleRunning, mcpAutoprogrammingPhase0CompleteNonPublishableActionsV0(observedByRunRef)...)
	result.StaleRunning = append(result.StaleRunning, mcpAutoprogrammingPartialArtifactsWrittenActionsV0(observedByRunRef)...)
	result.Diagnostics = append(result.Diagnostics, diagnosticsFromStaleRunningMCPAutoprogrammingV0(result.StaleRunning)...)
	result.Operator = newMCPAutoprogrammingOperatorV0(
		result.Queue,
		result.Run,
		result.Diagnostics,
		executor.AllowLegacySupervisorActions,
	)
	result.Operator = mcpAutoprogrammingOperatorWithGoalFirstActiveRunsV0(result.Operator, goalStates, goalRunMarkers)
	if !executor.AllowLegacySupervisorActions &&
		mcpAutoprogrammingLegacySupervisorActionCandidateV0(result.Operator, result.Run, goalFirstRunRefs) {
		result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingDiagnosticV0(
			"legacy_supervisor_actions_disabled",
			"operator",
			"supervision legacy no publicada como accion segura sin opt-in; usar goal-first/observe_goal o activar compatibilidad legacy",
		))
	}
	for _, goalState := range goalStates {
		result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingGoalFirstDiagnosticsV0(goalState)...)
	}
	for _, marker := range goalRunMarkers {
		result.Diagnostics = append(result.Diagnostics, mcpAutoprogrammingGoalFirstStateMissingDiagnosticsV0(marker)...)
	}
	if len(goalRunMarkers) > 0 {
		result.Operator = mcpAutoprogrammingOperatorWithGoalFirstStateMissingActionsV0(
			result.Operator,
			executor.AllowLegacySupervisorActions,
			mcpAutoprogrammingGoalRunMarkerRunRefsV0(goalRunMarkers)...,
		)
	}
	if len(goalStates) > 0 {
		attentionOnlyGoalRunRefs := mcpAutoprogrammingGoalStateAttentionOnlyRunRefsV0(goalStates)
		if len(attentionOnlyGoalRunRefs) > 0 {
			result.Operator = mcpAutoprogrammingOperatorWithGoalFirstStateMissingActionsV0(
				result.Operator,
				executor.AllowLegacySupervisorActions,
				attentionOnlyGoalRunRefs...,
			)
		}
		result.Operator = mcpAutoprogrammingOperatorWithGoalFirstActionsV0(
			result.Operator,
			executor.AllowLegacySupervisorActions,
			mcpAutoprogrammingGoalStateObservableRunRefsV0(goalStates)...,
		)
	}
	result.EfficiencySummary = buildMCPAutoprogrammingEfficiencySummaryV0(
		result.Queue,
		result.Run,
		result.Operator,
		result.QueueHealth,
		result.Diagnostics,
	)
	result.OpsSnapshot = buildMCPAutoprogrammingOpsSnapshotV0(result.Queue, result.Run, result.Operator, result.StaleRunning, input.OccurredAt)
	return result, nil
}

func (executor MCPAutoprogrammingStatusToolExecutorV0) goalStatesForAutoprogrammingStatusV0(
	ctx context.Context,
	result MCPAutoprogrammingStatusToolResultV0,
	input MCPAutoprogrammingStatusToolInputV0,
) []orquestagoal.GoalWorkStateV0 {
	if executor.GoalStateStore == nil {
		return nil
	}
	runRefs := executor.goalRunRefsForAutoprogrammingStatusV0(result, input)
	seen := map[string]bool{}
	out := make([]orquestagoal.GoalWorkStateV0, 0, len(runRefs))
	for _, runRef := range compactStringsMCPV0(runRefs) {
		if seen[runRef] {
			continue
		}
		seen[runRef] = true
		state, err := executor.GoalStateStore.LoadGoalWorkStateV0(ctx, runRef)
		if err != nil ||
			strings.TrimSpace(state.RunRef) == "" ||
			strings.TrimSpace(state.GoalRef) == "" {
			continue
		}
		out = append(out, state)
	}
	if lister, ok := executor.GoalStateStore.(orquestagoal.GoalWorkStateListPortV0); ok {
		listRequests := []orquestagoal.GoalWorkStateListRequestV0{
			{
				Statuses: []string{
					orquestagoal.GoalStatusRunningV0,
					orquestagoal.GoalStatusBlockedV0,
					orquestagoal.GoalStatusInvalidV0,
				},
				MaxItems: mcpAutoprogrammingRunningStatsMaxV0,
			},
			{
				Statuses: []string{orquestagoal.GoalStatusCompleteV0},
				MaxItems: mcpAutoprogrammingRunningStatsMaxV0,
			},
		}
		for _, listRequest := range listRequests {
			listed, err := lister.ListGoalWorkStatesV0(ctx, listRequest)
			if err != nil {
				continue
			}
			for _, state := range listed {
				runRef := strings.TrimSpace(state.RunRef)
				if runRef == "" ||
					seen[runRef] ||
					strings.TrimSpace(state.GoalRef) == "" ||
					!mcpAutoprogrammingGoalStateVisibleForStatusV0(state) {
					continue
				}
				seen[runRef] = true
				out = append(out, state)
			}
		}
	}
	return out
}

func (executor MCPAutoprogrammingStatusToolExecutorV0) goalRunMarkersForAutoprogrammingStatusV0(
	ctx context.Context,
	result MCPAutoprogrammingStatusToolResultV0,
	input MCPAutoprogrammingStatusToolInputV0,
	goalStatesByRunRef map[string]orquestagoal.GoalWorkStateV0,
) []orquestagoal.GoalWorkRunMarkerV0 {
	store := executor.goalRunMarkerStoreForAutoprogrammingStatusV0()
	if store == nil {
		return nil
	}
	runRefs := executor.goalRunRefsForAutoprogrammingStatusV0(result, input)
	seen := map[string]bool{}
	out := make([]orquestagoal.GoalWorkRunMarkerV0, 0, len(runRefs))
	for _, runRef := range runRefs {
		runRef = strings.TrimSpace(runRef)
		if runRef == "" || seen[runRef] {
			continue
		}
		seen[runRef] = true
		if _, ok := goalStatesByRunRef[runRef]; ok {
			continue
		}
		marker, err := store.LoadGoalWorkRunMarkerV0(ctx, runRef)
		if err != nil {
			continue
		}
		marker, err = orquestagoal.NewGoalWorkRunMarkerV0(marker)
		if err != nil || strings.TrimSpace(marker.RunRef) == "" {
			continue
		}
		out = append(out, marker)
	}
	if lister, ok := store.(orquestagoal.GoalWorkRunMarkerListPortV0); ok {
		listed, err := lister.ListGoalWorkRunMarkersV0(ctx, orquestagoal.GoalWorkRunMarkerListRequestV0{
			ActiveOnly: true,
			MaxItems:   mcpAutoprogrammingRunningStatsMaxV0,
		})
		if err == nil {
			for _, marker := range listed {
				marker, markerErr := orquestagoal.NewGoalWorkRunMarkerV0(marker)
				runRef := strings.TrimSpace(marker.RunRef)
				if markerErr != nil || runRef == "" || seen[runRef] {
					continue
				}
				seen[runRef] = true
				if _, ok := goalStatesByRunRef[runRef]; ok {
					continue
				}
				out = append(out, marker)
			}
		}
	}
	return out
}

func (executor MCPAutoprogrammingStatusToolExecutorV0) goalRunMarkerStoreForAutoprogrammingStatusV0() orquestagoal.GoalWorkRunMarkerStorePortV0 {
	if executor.GoalRunMarkerStore != nil {
		return executor.GoalRunMarkerStore
	}
	store, _ := executor.GoalStateStore.(orquestagoal.GoalWorkRunMarkerStorePortV0)
	return store
}

func (executor MCPAutoprogrammingStatusToolExecutorV0) goalRunRefsForAutoprogrammingStatusV0(
	result MCPAutoprogrammingStatusToolResultV0,
	input MCPAutoprogrammingStatusToolInputV0,
) []string {
	runRefs := compactStringsMCPV0([]string{result.RunRef, input.RunRef})
	if result.Queue != nil {
		for _, candidate := range append(
			append([]MCPRunQueueRankedCandidateCompactV0{}, result.Queue.Ranked...),
			result.Queue.Terminal...,
		) {
			if mcpAutoprogrammingTerminalRunV0(candidate.Status) {
				continue
			}
			runRefs = append(runRefs, strings.TrimSpace(candidate.RunRef))
		}
	}
	return compactStringsMCPV0(runRefs)
}

func mcpAutoprogrammingGoalStateRunRefsV0(
	states []orquestagoal.GoalWorkStateV0,
) []string {
	out := make([]string, 0, len(states))
	for _, state := range states {
		out = append(out, strings.TrimSpace(state.RunRef))
	}
	return compactStringsMCPV0(out)
}

func mcpAutoprogrammingGoalStateObservableRunRefsV0(
	states []orquestagoal.GoalWorkStateV0,
) []string {
	out := make([]string, 0, len(states))
	for _, state := range states {
		if mcpAutoprogrammingGoalStateObserveRequiredV0(state) {
			out = append(out, strings.TrimSpace(state.RunRef))
		}
	}
	return compactStringsMCPV0(out)
}

func mcpAutoprogrammingGoalStateAttentionOnlyRunRefsV0(
	states []orquestagoal.GoalWorkStateV0,
) []string {
	out := make([]string, 0, len(states))
	for _, state := range states {
		if mcpAutoprogrammingGoalStateNeedsAttentionV0(state) &&
			!mcpAutoprogrammingGoalStateObserveRequiredV0(state) {
			out = append(out, strings.TrimSpace(state.RunRef))
		}
	}
	return compactStringsMCPV0(out)
}

func mcpAutoprogrammingGoalRunMarkerRunRefsV0(
	markers []orquestagoal.GoalWorkRunMarkerV0,
) []string {
	out := make([]string, 0, len(markers))
	for _, marker := range markers {
		out = append(out, strings.TrimSpace(marker.RunRef))
	}
	return compactStringsMCPV0(out)
}

func mcpAutoprogrammingGoalStatesByRunRefV0(
	states []orquestagoal.GoalWorkStateV0,
) map[string]orquestagoal.GoalWorkStateV0 {
	out := map[string]orquestagoal.GoalWorkStateV0{}
	for _, state := range states {
		runRef := strings.TrimSpace(state.RunRef)
		if runRef == "" || strings.TrimSpace(state.GoalRef) == "" {
			continue
		}
		out[runRef] = state
	}
	return out
}

func mcpAutoprogrammingGoalRunMarkersByRunRefV0(
	markers []orquestagoal.GoalWorkRunMarkerV0,
) map[string]orquestagoal.GoalWorkRunMarkerV0 {
	out := map[string]orquestagoal.GoalWorkRunMarkerV0{}
	for _, marker := range markers {
		runRef := strings.TrimSpace(marker.RunRef)
		if runRef == "" {
			continue
		}
		out[runRef] = marker
	}
	return out
}

func mcpAutoprogrammingGoalFirstRunRefSetFromStatesV0(
	states []orquestagoal.GoalWorkStateV0,
) map[string]bool {
	out := map[string]bool{}
	for _, state := range states {
		runRef := strings.TrimSpace(state.RunRef)
		if runRef == "" || strings.TrimSpace(state.GoalRef) == "" {
			continue
		}
		out[runRef] = true
	}
	return out
}

func mcpAutoprogrammingGoalFirstRunRefSetFromMarkersV0(
	markers []orquestagoal.GoalWorkRunMarkerV0,
) map[string]bool {
	out := map[string]bool{}
	for _, marker := range markers {
		runRef := strings.TrimSpace(marker.RunRef)
		if runRef != "" {
			out[runRef] = true
		}
	}
	return out
}

func mcpAutoprogrammingMergeGoalFirstRunRefSetsV0(
	sets ...map[string]bool,
) map[string]bool {
	out := map[string]bool{}
	for _, set := range sets {
		for runRef, ok := range set {
			runRef = strings.TrimSpace(runRef)
			if ok && runRef != "" {
				out[runRef] = true
			}
		}
	}
	return out
}

func mcpAutoprogrammingGoalFirstRunStructSetV0(
	set map[string]bool,
) map[string]struct{} {
	out := map[string]struct{}{}
	for runRef, ok := range set {
		runRef = strings.TrimSpace(runRef)
		if ok && runRef != "" {
			out[runRef] = struct{}{}
		}
	}
	return out
}

func mcpAutoprogrammingGoalFirstDiagnosticsV0(
	state orquestagoal.GoalWorkStateV0,
) []MCPAutoprogrammingDiagnosticV0 {
	if mcpAutoprogrammingGoalStateNeedsAttentionV0(state) &&
		!mcpAutoprogrammingGoalStateObserveRequiredV0(state) {
		return []MCPAutoprogrammingDiagnosticV0{{
			Code:  "autoprogramming_goal_first_blocked",
			Scope: "run:" + strings.TrimSpace(state.RunRef),
			Message: strings.Join(compactStringsMCPV0([]string{
				"goal_ref=" + strings.TrimSpace(state.GoalRef),
				"goal_status=" + strings.TrimSpace(state.Status),
				"action=review_replan_goal_first",
			}), " "),
			EvidenceRefs: compactStringsMCPV0(append(
				[]string{mcpAutoprogrammingEvidenceGoalFirstBlockedV0},
				state.EvidenceRefs...,
			)),
		}}
	}
	return []MCPAutoprogrammingDiagnosticV0{{
		Code:  "autoprogramming_goal_first_observe_required",
		Scope: "run:" + strings.TrimSpace(state.RunRef),
		Message: strings.Join(compactStringsMCPV0([]string{
			"goal_ref=" + strings.TrimSpace(state.GoalRef),
			"goal_status=" + strings.TrimSpace(state.Status),
			"action=observe_goal",
		}), " "),
		EvidenceRefs: compactStringsMCPV0(append(
			[]string{"evidence-ref-autoprogramming-status-goal-first-observe-required"},
			state.EvidenceRefs...,
		)),
	}}
}

func mcpAutoprogrammingGoalStateVisibleForStatusV0(
	state orquestagoal.GoalWorkStateV0,
) bool {
	return mcpAutoprogrammingGoalStateObserveRequiredV0(state) ||
		mcpAutoprogrammingGoalStateNeedsAttentionV0(state)
}

func mcpAutoprogrammingGoalStateObserveRequiredV0(
	state orquestagoal.GoalWorkStateV0,
) bool {
	return orquestagoal.GoalWorkStatePendingObservationV0(state)
}

func mcpAutoprogrammingGoalStateNeedsAttentionV0(
	state orquestagoal.GoalWorkStateV0,
) bool {
	if state.LastClosure != nil {
		closureStatus := strings.ToLower(strings.TrimSpace(state.LastClosure.Status))
		if state.LastClosure.NeedsRework || closureStatus == orquestagoal.GoalStatusBlockedV0 {
			return true
		}
	}
	switch strings.ToLower(strings.TrimSpace(state.Status)) {
	case orquestagoal.GoalStatusBlockedV0, orquestagoal.GoalStatusInvalidV0:
		return true
	default:
		return false
	}
}

func mcpAutoprogrammingGoalFirstStateMissingDiagnosticsV0(
	marker orquestagoal.GoalWorkRunMarkerV0,
) []MCPAutoprogrammingDiagnosticV0 {
	return []MCPAutoprogrammingDiagnosticV0{{
		Code:  "autoprogramming_goal_first_state_missing",
		Scope: "run:" + strings.TrimSpace(marker.RunRef),
		Message: strings.Join(compactStringsMCPV0([]string{
			"goal_ref=" + strings.TrimSpace(marker.GoalRef),
			"goal_status=" + strings.TrimSpace(marker.Status),
			"action=repair_goal_state",
		}), " "),
		EvidenceRefs: compactStringsMCPV0(append(
			[]string{"evidence-ref-autoprogramming-status-goal-first-state-missing"},
			marker.EvidenceRefs...,
		)),
	}}
}

func mcpAutoprogrammingQueueRunningStatsObservedV0(
	queue *MCPRunQueuePriorityToolResultV0,
	run *MCPDirectorStatsToolResultV0,
	observedRuns ...*MCPDirectorStatsToolResultV0,
) bool {
	if queue == nil || queue.Estado != MCPRunQueuePriorityEstadoOKV0 || len(queue.Ranked) == 0 {
		return false
	}
	observed := mcpAutoprogrammingObservedRunsByRefV0(run, observedRuns...)
	for _, candidate := range queue.Ranked {
		if strings.ToLower(strings.TrimSpace(candidate.Status)) != "running" {
			continue
		}
		if _, ok := observed[strings.TrimSpace(candidate.RunRef)]; !ok {
			return false
		}
	}
	return len(observed) > 0
}

func (executor MCPAutoprogrammingStatusToolExecutorV0) executeQueueRunningStatsV0(
	ctx context.Context,
	input MCPAutoprogrammingStatusToolInputV0,
	queue *MCPRunQueuePriorityToolResultV0,
	explicitRun *MCPDirectorStatsToolResultV0,
	goalStatesByRunRef map[string]orquestagoal.GoalWorkStateV0,
) ([]*MCPDirectorStatsToolResultV0, []MCPAutoprogrammingDiagnosticV0) {
	if executor.Stats == nil || queue == nil || queue.Estado != MCPRunQueuePriorityEstadoOKV0 {
		return nil, nil
	}
	explicitRunRef := ""
	if explicitRun != nil && explicitRun.Stats != nil {
		explicitRunRef = strings.TrimSpace(explicitRun.Stats.RunRef)
	}
	seen := map[string]bool{}
	if explicitRunRef != "" {
		seen[explicitRunRef] = true
	}
	out := make([]*MCPDirectorStatsToolResultV0, 0)
	diagnostics := []MCPAutoprogrammingDiagnosticV0{}
	if mcpAutoprogrammingExplicitRunNeedsInternalLiveStatsV0(input, queue, explicitRun) {
		stats, statsDiagnostics := executor.executeAutoprogrammingInternalLiveStatsV0(ctx, input, explicitRunRef)
		diagnostics = append(diagnostics, statsDiagnostics...)
		if stats != nil {
			out = append(out, stats)
		}
	}
	for _, candidate := range queue.Ranked {
		if len(out) >= mcpAutoprogrammingRunningStatsMaxV0 {
			break
		}
		runRef := strings.TrimSpace(candidate.RunRef)
		if runRef == "" ||
			seen[runRef] ||
			strings.ToLower(strings.TrimSpace(candidate.Status)) != "running" {
			continue
		}
		seen[runRef] = true
		stats, statsDiagnostics := executor.executeAutoprogrammingInternalLiveStatsV0(ctx, input, runRef)
		diagnostics = append(diagnostics, statsDiagnostics...)
		if stats != nil {
			out = append(out, stats)
		}
	}
	for _, candidate := range queue.Terminal {
		if len(out) >= mcpAutoprogrammingRunningStatsMaxV0 {
			break
		}
		runRef := strings.TrimSpace(candidate.RunRef)
		if runRef == "" || seen[runRef] {
			continue
		}
		if _, ok := goalStatesByRunRef[runRef]; !ok {
			continue
		}
		seen[runRef] = true
		stats, statsDiagnostics := executor.executeAutoprogrammingInternalLiveStatsV0(ctx, input, runRef)
		diagnostics = append(diagnostics, statsDiagnostics...)
		if stats != nil {
			out = append(out, stats)
		}
	}
	for runRef := range goalStatesByRunRef {
		if len(out) >= mcpAutoprogrammingRunningStatsMaxV0 {
			break
		}
		runRef = strings.TrimSpace(runRef)
		if runRef == "" || seen[runRef] {
			continue
		}
		seen[runRef] = true
		stats, statsDiagnostics := executor.executeAutoprogrammingInternalLiveStatsV0(ctx, input, runRef)
		diagnostics = append(diagnostics, statsDiagnostics...)
		if stats != nil {
			out = append(out, stats)
		}
	}
	return out, diagnostics
}

func mcpAutoprogrammingRunningStatsSampleLimitedDiagnosticsV0(
	queue *MCPRunQueuePriorityToolResultV0,
	statsConfigured bool,
	run *MCPDirectorStatsToolResultV0,
	observedRuns ...*MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingDiagnosticV0 {
	if !statsConfigured || queue == nil || queue.Estado != MCPRunQueuePriorityEstadoOKV0 {
		return nil
	}
	runningTotal := 0
	for _, candidate := range queue.Ranked {
		if strings.ToLower(strings.TrimSpace(candidate.Status)) == "running" {
			runningTotal++
		}
	}
	if runningTotal <= mcpAutoprogrammingRunningStatsMaxV0 {
		return nil
	}
	observed := mcpAutoprogrammingObservedRunsByRefV0(run, observedRuns...)
	if len(observed) >= runningTotal {
		return nil
	}
	return []MCPAutoprogrammingDiagnosticV0{mcpAutoprogrammingDiagnosticV0(
		"running_stats_sample_limited",
		"queue",
		"status observo una muestra acotada de runs running; usar run_ref o queue_global_status para inspeccion completa",
	)}
}

func (executor MCPAutoprogrammingStatusToolExecutorV0) executeAutoprogrammingInternalLiveStatsV0(
	ctx context.Context,
	input MCPAutoprogrammingStatusToolInputV0,
	runRef string,
) (*MCPDirectorStatsToolResultV0, []MCPAutoprogrammingDiagnosticV0) {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return nil, nil
	}
	statsInput := mcpAutoprogrammingStatsInputV0(input)
	statsInput.RunRef = runRef
	statsInput.ExternalJobRef = ""
	statsInput.IncludeProcessRefs = true
	statsInput.IncludeAgentProgress = true
	stats, err := executor.Stats.Execute(ctx, statsInput)
	if err != nil || stats.Estado != MCPDirectorStatsEstadoOKV0 || stats.Stats == nil {
		return nil, []MCPAutoprogrammingDiagnosticV0{mcpAutoprogrammingDiagnosticV0(
			mcpAutoprogrammingActionRunningWithoutRecentStatsV0,
			"run:"+runRef,
			"run en cola como running sin stats/liveness verificables",
		)}
	}
	return &stats, nil
}

func mcpAutoprogrammingExplicitRunNeedsInternalLiveStatsV0(
	input MCPAutoprogrammingStatusToolInputV0,
	queue *MCPRunQueuePriorityToolResultV0,
	run *MCPDirectorStatsToolResultV0,
) bool {
	if run == nil || run.Stats == nil {
		return false
	}
	runRef := strings.TrimSpace(run.Stats.RunRef)
	if runRef == "" {
		return false
	}
	candidate, ok := queueCandidateForRunMCPAutoprogrammingHealthV0(queue, runRef)
	if !ok || strings.ToLower(strings.TrimSpace(candidate.Status)) != "running" {
		return false
	}
	if bool(input.IncludeProcessRefs) && bool(input.IncludeAgentProgress) {
		return false
	}
	return !mcpAutoprogrammingRunStatsLiveV0(run)
}

func mcpAutoprogrammingPreferredRunStatsForQueueHealthV0(
	run *MCPDirectorStatsToolResultV0,
	observedRuns ...*MCPDirectorStatsToolResultV0,
) (*MCPDirectorStatsToolResultV0, []*MCPDirectorStatsToolResultV0) {
	preferred := run
	runRef := ""
	if run != nil && run.Stats != nil {
		runRef = strings.TrimSpace(run.Stats.RunRef)
	}
	out := make([]*MCPDirectorStatsToolResultV0, 0, len(observedRuns))
	for _, observed := range observedRuns {
		if observed == nil || observed.Stats == nil {
			continue
		}
		if runRef != "" && strings.TrimSpace(observed.Stats.RunRef) == runRef {
			preferred = observed
			continue
		}
		out = append(out, observed)
	}
	return preferred, out
}

func (executor MCPAutoprogrammingStatusToolExecutorV0) executeRunStatusV0(
	ctx context.Context,
	input MCPAutoprogrammingStatusToolInputV0,
) (bool, *MCPDirectorStatsToolResultV0, []MCPAutoprogrammingDiagnosticV0, error) {
	if executor.Stats == nil {
		return false, nil, []MCPAutoprogrammingDiagnosticV0{
			mcpAutoprogrammingDiagnosticV0("run_stats_unbound", "run", "director_stats no configurado"),
		}, nil
	}
	run, err := executor.Stats.Execute(ctx, mcpAutoprogrammingStatsInputV0(input))
	if err != nil {
		return false, nil, []MCPAutoprogrammingDiagnosticV0{
			mcpAutoprogrammingDiagnosticV0("run_stats_error", "run", "estado de run no disponible"),
		}, nil
	}
	diagnostics := diagnosticsFromIssuesMCPAutoprogrammingV0("run", run.Errores)
	diagnostics = append(diagnostics, diagnosticsFromRunProgressIssuesMCPAutoprogrammingV0(run.Stats)...)
	return run.Estado == MCPDirectorStatsEstadoOKV0, &run, diagnostics, nil
}

func newMCPAutoprogrammingStatusBaseV0(
	input MCPAutoprogrammingStatusToolInputV0,
) MCPAutoprogrammingStatusToolResultV0 {
	return MCPAutoprogrammingStatusToolResultV0{
		Estado:        MCPAutoprogrammingStatusEstadoOKV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:        strings.TrimSpace(input.RunRef),
		QueueRef:      strings.TrimSpace(input.QueueRef),
		Diagnostics:   []MCPAutoprogrammingDiagnosticV0{},
		Errores:       []MCPValidationIssueV0{},
	}
}

func mcpAutoprogrammingQueueInputV0(
	input MCPAutoprogrammingStatusToolInputV0,
) MCPRunQueuePriorityToolInputV0 {
	return MCPRunQueuePriorityToolInputV0{
		RequestID:            input.RequestID,
		CorrelationID:        input.CorrelationID,
		Action:               MCPRunQueuePriorityActionRankV0,
		QueueRef:             input.QueueRef,
		AppRefs:              input.AppRefs,
		Limit:                input.QueueLimit,
		IncludeNonExecutable: true,
		OccurredAt:           input.OccurredAt,
	}
}

func mcpAutoprogrammingStatsInputV0(
	input MCPAutoprogrammingStatusToolInputV0,
) MCPDirectorStatsToolInputV0 {
	return MCPDirectorStatsToolInputV0{
		RequestID:            input.RequestID,
		CorrelationID:        input.CorrelationID,
		RunRef:               input.RunRef,
		AppRef:               input.AppRef,
		ExternalJobRef:       input.ExternalJobRef,
		OccurredAt:           input.OccurredAt,
		IncludeProcessRefs:   bool(input.IncludeProcessRefs),
		IncludeAgentProgress: bool(input.IncludeAgentProgress),
		IncludeAgentUsage:    bool(input.IncludeAgentUsage),
	}
}

func wantsMCPAutoprogrammingRunStatusV0(input MCPAutoprogrammingStatusToolInputV0) bool {
	return strings.TrimSpace(input.RunRef) != "" || strings.TrimSpace(input.ExternalJobRef) != ""
}

func diagnosticsFromIssuesMCPAutoprogrammingV0(
	scope string,
	issues []MCPValidationIssueV0,
) []MCPAutoprogrammingDiagnosticV0 {
	out := make([]MCPAutoprogrammingDiagnosticV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, mcpAutoprogrammingDiagnosticV0(issue.Code, scope, issue.Message))
	}
	return out
}

func mcpAutoprogrammingDiagnosticV0(
	code string,
	scope string,
	message string,
) MCPAutoprogrammingDiagnosticV0 {
	return MCPAutoprogrammingDiagnosticV0{
		Code:    strings.TrimSpace(code),
		Scope:   strings.TrimSpace(scope),
		Message: strings.TrimSpace(message),
	}
}

func mcpAutoprogrammingStatusConfiguredDiagnosticsV0(
	diagnostics []MCPAutoprogrammingDiagnosticV0,
) []MCPAutoprogrammingDiagnosticV0 {
	if len(diagnostics) == 0 {
		return nil
	}
	out := make([]MCPAutoprogrammingDiagnosticV0, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		clean := MCPAutoprogrammingDiagnosticV0{
			Code:         strings.TrimSpace(diagnostic.Code),
			Scope:        strings.TrimSpace(diagnostic.Scope),
			Message:      strings.TrimSpace(diagnostic.Message),
			EvidenceRefs: compactStringsMCPV0(diagnostic.EvidenceRefs),
		}
		if clean.Code == "" {
			continue
		}
		out = append(out, clean)
	}
	return out
}

func diagnosticsFromSuppressedQueueMCPAutoprogrammingV0(
	queue *MCPRunQueuePriorityToolResultV0,
) []MCPAutoprogrammingDiagnosticV0 {
	if queue == nil {
		return nil
	}
	out := []MCPAutoprogrammingDiagnosticV0{}
	for _, candidate := range queue.Terminal {
		if !suppressedDomainSessionQueueCandidateMCPAutoprogrammingV0(candidate) {
			continue
		}
		out = append(out, MCPAutoprogrammingDiagnosticV0{
			Code:         mcpAutoprogrammingDomainSessionSuppressedReasonV0,
			Scope:        "queue",
			Message:      "autoprogramacion stale suprimida por sesion de dominio",
			EvidenceRefs: compactStringsMCPV0(append([]string{mcpAutoprogrammingDomainSessionSuppressedEvidenceV0}, candidate.EvidenceRefs...)),
		})
	}
	return out
}

func suppressedDomainSessionQueueCandidateMCPAutoprogrammingV0(
	candidate MCPRunQueueRankedCandidateCompactV0,
) bool {
	if strings.TrimSpace(candidate.RescueReason) == mcpAutoprogrammingDomainSessionSuppressedReasonV0 {
		return true
	}
	for _, ref := range candidate.EvidenceRefs {
		if strings.TrimSpace(ref) == mcpAutoprogrammingDomainSessionSuppressedEvidenceV0 {
			return true
		}
	}
	return false
}
