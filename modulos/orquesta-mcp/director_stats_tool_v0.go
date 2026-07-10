package orquestamcp

import (
	"context"
	"strings"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

const (
	MCPDirectorStatsToolNameV0    = "orquesta.director.stats.v0"
	MCPDirectorStatsToolVersionV0 = "v0"
	MCPDirectorStatsResourceURIV0 = "orquesta://contracts/director-stats/v0"
	MCPDirectorStatsEstadoOKV0    = "ok"
	MCPDirectorStatsEstadoErrorV0 = "error"
)

type MCPDirectorStatsToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPDirectorStatsToolInputV0 struct {
	RequestID            string `json:"request_id,omitempty"`
	CorrelationID        string `json:"correlation_id,omitempty"`
	RunRef               string `json:"run_ref"`
	AppRef               string `json:"app_ref,omitempty"`
	ExternalJobRef       string `json:"external_job_ref,omitempty"`
	OccurredAt           string `json:"occurred_at,omitempty"`
	IncludeProcessRefs   bool   `json:"include_process_refs,omitempty"`
	IncludeAgentProgress bool   `json:"include_agent_progress,omitempty"`
	IncludeAgentUsage    bool   `json:"include_agent_usage,omitempty"`
}

type MCPDirectorStatsToolResultV0 struct {
	Estado           string                                                 `json:"estado"`
	RequestID        string                                                 `json:"request_id,omitempty"`
	CorrelationID    string                                                 `json:"correlation_id,omitempty"`
	RunRef           string                                                 `json:"run_ref,omitempty"`
	CausalVerdict    string                                                 `json:"causal_verdict,omitempty"`
	CausalReasonCode string                                                 `json:"causal_reason_code,omitempty"`
	ExternalJob      *MCPDirectorExternalJobStatsV0                         `json:"external_job,omitempty"`
	Goal             *MCPDirectorGoalStatsV0                                `json:"goal,omitempty"`
	Stats            *orquestacionnucleoapp.DirectorRunStatsV0              `json:"stats,omitempty"`
	DecisionContext  *orquestaobservability.DirectorDecisionContextV0       `json:"decision_context,omitempty"`
	OpsSnapshot      *orquestaobservability.DirectorAutonomousOpsSnapshotV0 `json:"ops_snapshot,omitempty"`
	Errores          []MCPValidationIssueV0                                 `json:"errores_publicos,omitempty"`
}

type MCPDirectorExternalJobStatsRequestV0 struct {
	RunRef         string `json:"run_ref,omitempty"`
	AppRef         string `json:"app_ref,omitempty"`
	ExternalJobRef string `json:"external_job_ref"`
	CorrelationID  string `json:"correlation_id,omitempty"`
	OccurredAt     string `json:"occurred_at,omitempty"`
}

type MCPDirectorGoalStatsV0 struct {
	DirectorExecutionMode    string         `json:"director_execution_mode,omitempty"`
	RunRef                   string         `json:"run_ref,omitempty"`
	GoalRef                  string         `json:"goal_ref"`
	ExternalGoalRef          string         `json:"external_goal_ref,omitempty"`
	Status                   string         `json:"status,omitempty"`
	ClosureStatus            string         `json:"closure_status,omitempty"`
	ClosureAccepted          bool           `json:"closure_accepted,omitempty"`
	ClosureNeedsRework       bool           `json:"closure_needs_rework,omitempty"`
	CurrentPhase             string         `json:"current_phase,omitempty"`
	RetryFromPhase           string         `json:"retry_from_phase,omitempty"`
	OperationalReason        string         `json:"operational_reason,omitempty"`
	DomainCounters           map[string]int `json:"domain_counters,omitempty"`
	ContextBudgetTotalBytes  int64          `json:"context_budget_total_bytes,omitempty"`
	StaticPromptBytes        int64          `json:"static_prompt_bytes,omitempty"`
	QueriedContextBytes      int64          `json:"queried_context_bytes,omitempty"`
	MaterializedContextBytes int64          `json:"materialized_context_bytes,omitempty"`
	DynamicContextBytes      int64          `json:"dynamic_context_bytes,omitempty"`
	CodeContextCacheStatus   string         `json:"code_context_cache_status,omitempty"`
	ArtifactRefs             []string       `json:"artifact_refs,omitempty"`
	DomainReceiptRefs        []string       `json:"domain_receipt_refs,omitempty"`
	ExpectedReceiptRefs      []string       `json:"expected_terminal_receipt_refs,omitempty"`
	IssueCodes               []string       `json:"issue_codes,omitempty"`
	EvidenceRefs             []string       `json:"evidence_refs,omitempty"`
}

type MCPDirectorExternalJobStatsV0 struct {
	AppRef                string                               `json:"app_ref,omitempty"`
	JobRef                string                               `json:"job_ref"`
	WorkKind              string                               `json:"work_kind,omitempty"`
	ChangeRef             string                               `json:"change_ref,omitempty"`
	RunRef                string                               `json:"run_ref,omitempty"`
	TaskRef               string                               `json:"task_ref,omitempty"`
	AgentRef              string                               `json:"agent_ref,omitempty"`
	DirectorExecutionMode string                               `json:"director_execution_mode,omitempty"`
	GoalRef               string                               `json:"goal_ref,omitempty"`
	ExternalGoalRef       string                               `json:"external_goal_ref,omitempty"`
	GoalStatus            string                               `json:"goal_status,omitempty"`
	ClosureStatus         string                               `json:"closure_status,omitempty"`
	ClosureAccepted       bool                                 `json:"closure_accepted,omitempty"`
	ClosureNeedsRework    bool                                 `json:"closure_needs_rework,omitempty"`
	Status                string                               `json:"status,omitempty"`
	StatusReason          string                               `json:"status_reason,omitempty"`
	CurrentPhase          string                               `json:"current_phase,omitempty"`
	RetryFromPhase        string                               `json:"retry_from_phase,omitempty"`
	OperationalReason     string                               `json:"operational_reason,omitempty"`
	DomainCounters        map[string]int                       `json:"domain_counters,omitempty"`
	DeliveryRefs          []string                             `json:"delivery_refs,omitempty"`
	IssueRefs             []string                             `json:"issue_refs,omitempty"`
	EvidenceRefs          []string                             `json:"evidence_refs,omitempty"`
	Diagnostics           []MCPDirectorExternalJobDiagnosticV0 `json:"diagnostics,omitempty"`
}

type MCPDirectorExternalJobDiagnosticV0 struct {
	Code         string   `json:"code"`
	Scope        string   `json:"scope,omitempty"`
	Message      string   `json:"message,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type MCPDirectorExternalJobStatsSourcePortV0 interface {
	ResolveDirectorExternalJobStatsV0(
		context.Context,
		MCPDirectorExternalJobStatsRequestV0,
	) (MCPDirectorExternalJobStatsV0, bool, error)
}

type MCPDirectorGoalStateSourcePortV0 interface {
	LoadGoalWorkStateV0(context.Context, string) (orquestagoal.GoalWorkStateV0, error)
}

type MCPDirectorGoalRunMarkerSourcePortV0 interface {
	LoadGoalWorkRunMarkerV0(context.Context, string) (orquestagoal.GoalWorkRunMarkerV0, error)
}

type MCPDirectorGoalMaterializedRefsSourcePortV0 interface {
	ResolveDirectorGoalMaterializedRefsV0(
		context.Context,
		orquestagoal.GoalWorkStateV0,
	) (MCPDirectorGoalMaterializedRefsV0, bool, error)
}

type MCPDirectorGoalMaterializedRefsV0 struct {
	ArtifactRefs        []string `json:"artifact_refs,omitempty"`
	DomainReceiptRefs   []string `json:"domain_receipt_refs,omitempty"`
	ExpectedReceiptRefs []string `json:"expected_terminal_receipt_refs,omitempty"`
	EvidenceRefs        []string `json:"evidence_refs,omitempty"`
	IssueCodes          []string `json:"issue_codes,omitempty"`
}

type MCPDirectorStatsToolExecutorV0 struct {
	RunStore                   orquestacionnucleoapp.RunStorePortV0
	RunControl                 orquestaruncontrol.RunControlReaderPortV0
	ProcessRegistry            orquestacionnucleoapp.AgentProcessRegistryPortV0
	ProcessSnapshot            orquestacionnucleoapp.ProcessRuntimeIdentitySnapshotPortV0
	ProgressSource             orquestacionnucleoapp.AgentProgressObservationProviderPortV0
	AgentUsageSource           orquestacionnucleoapp.AgentUsageStatsProviderPortV0
	EstadoVivoSource           orquestaestadovivo.FuenteEvidenciaEstadoPortV0
	ExternalJobSource          MCPDirectorExternalJobStatsSourcePortV0
	GoalStateSource            MCPDirectorGoalStateSourcePortV0
	GoalMarkerSource           MCPDirectorGoalRunMarkerSourcePortV0
	GoalMaterializedRefsSource MCPDirectorGoalMaterializedRefsSourcePortV0
	OperationalPlanStateStore  orquestacionnucleoapp.OperationalDirectorPlanStateStorePortV0
}

func MCPDirectorStatsDescriptorV0() MCPDirectorStatsToolDescriptorV0 {
	return MCPDirectorStatsToolDescriptorV0{
		Name:        MCPDirectorStatsToolNameV0,
		Version:     MCPDirectorStatsToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,run_ref?,app_ref?,external_job_ref?,occurred_at?,include_process_refs?,include_agent_progress?,include_agent_usage?}",
		Output:      "ok:{run_ref,external_job?{status,status_reason?,external_goal_ref?,issue_refs?,evidence_refs?,diagnostics?},goal?{goal_ref,external_goal_ref?,status,closure_status?,context_budget_total_bytes?,static_prompt_bytes?,dynamic_context_bytes?,code_context_cache_status?,artifact_refs?,domain_receipt_refs?,expected_terminal_receipt_refs?,issue_codes?,evidence_refs?},stats{progress,closure},decision_context,ops_snapshot}|error:{errores_publicos}",
		ResourceURI: MCPDirectorStatsResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"consulta el estado por RunStorePortV0 inyectado",
			"enriquece control de procesos solo por AgentProcessRegistryPortV0 opcional",
			"expone cierre bloqueado/ready/cerrado derivado del run y PlanState operativo inyectado",
			"devuelve refs opacas sin rutas locales credenciales trazas sensibles ni detalles de runtime",
		},
	}
}

func (executor MCPDirectorStatsToolExecutorV0) Execute(
	ctx context.Context,
	input MCPDirectorStatsToolInputV0,
) (MCPDirectorStatsToolResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	runRef := strings.TrimSpace(input.RunRef)
	externalJobRef := strings.TrimSpace(input.ExternalJobRef)
	var externalJob *MCPDirectorExternalJobStatsV0
	if runRef == "" && externalJobRef != "" {
		resolved, ok, err := executor.resolveExternalJobStatsV0(ctx, input, "")
		if err != nil {
			return newMCPDirectorStatsErrorV0(input, "external_job_stats_error", "external_job_ref", "external job stats no disponible"), nil
		}
		if !ok || strings.TrimSpace(resolved.RunRef) == "" {
			return newMCPDirectorStatsErrorV0(input, "external_job_no_disponible", "external_job_ref", "external job no disponible"), nil
		}
		externalJob = &resolved
		runRef = resolved.RunRef
	}
	if runRef == "" {
		return newMCPDirectorStatsErrorV0(input, "run_ref_requerido", "run_ref", "run_ref requerido"), nil
	}
	if executor.RunStore == nil {
		return newMCPDirectorStatsErrorV0(input, "run_store_no_disponible", "run_store", "run_store requerido"), nil
	}
	run, err := executor.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		if externalJobRef != "" {
			resolved, ok, resolveErr := executor.resolveExternalJobStatsV0(ctx, input, "")
			if resolveErr != nil {
				return newMCPDirectorStatsErrorV0(input, "external_job_stats_error", "external_job_ref", "external job stats no disponible"), nil
			}
			if ok && strings.TrimSpace(resolved.RunRef) != "" {
				externalJob = &resolved
				runRef = strings.TrimSpace(resolved.RunRef)
				run, err = executor.RunStore.LoadRunV0(ctx, runRef)
			}
		}
	}
	if err != nil {
		return newMCPDirectorStatsErrorV0(input, "run_no_disponible", "run_ref", "run no disponible"), nil
	}
	if externalJobRef != "" && externalJob == nil {
		resolved, ok, err := executor.resolveExternalJobStatsV0(ctx, input, runRef)
		if err != nil {
			return newMCPDirectorStatsErrorV0(input, "external_job_stats_error", "external_job_ref", "external job stats no disponible"), nil
		}
		if !ok {
			return newMCPDirectorStatsErrorV0(input, "external_job_no_disponible", "external_job_ref", "external job no disponible"), nil
		}
		externalJob = &resolved
	}
	stats := orquestacionnucleoapp.BuildDirectorRunStatsV0(run)
	if executor.ProcessRegistry != nil || input.IncludeAgentProgress || input.IncludeAgentUsage {
		var progressSource orquestacionnucleoapp.AgentProgressObservationProviderPortV0
		if input.IncludeAgentProgress {
			progressSource = executor.ProgressSource
		}
		var usageSource orquestacionnucleoapp.AgentUsageStatsProviderPortV0
		if input.IncludeAgentUsage {
			usageSource = executor.AgentUsageSource
		}
		stats = orquestacionnucleoapp.BuildDirectorRunStatsWithTelemetryPortsAndProcessSnapshotsV0(
			ctx,
			run,
			executor.ProcessRegistry,
			executor.ProcessSnapshot,
			progressSource,
			usageSource,
			orquestacionnucleoapp.DirectorProgressSourceRequestV0{
				OccurredAt:        input.OccurredAt,
				CorrelationID:     input.CorrelationID,
				IncludeAgentUsage: input.IncludeAgentUsage,
			},
		)
	}
	if !input.IncludeProcessRefs {
		clearMCPDirectorStatsProcessRefsV0(&stats)
	}
	executor.applyRunControlProjectionV0(ctx, runRef, &stats)
	enrichMCPDirectorStatsRequestedAgentNotStartedV0(&stats)
	enrichMCPDirectorStatsExternalWorkStoppedNoDeliveryV0(&stats)
	enrichMCPDirectorStatsExternalWorkNoAgentMaterializedV0(&stats)
	goal := executor.resolveGoalStatsV0(ctx, stats.RunRef)
	applyMCPDirectorGoalRunProjectionV0(goal, &stats)
	causalVerdict, causalReasonCode := executor.applyEstadoVivoProjectionV0(ctx, input, &stats)
	executor.applyOperationalPlanStateProjectionV0(ctx, stats.RunRef, &stats)
	decisionContext := buildMCPDirectorDecisionContextV0(run, stats, input.OccurredAt)
	return MCPDirectorStatsToolResultV0{
		Estado:           MCPDirectorStatsEstadoOKV0,
		RequestID:        strings.TrimSpace(input.RequestID),
		CorrelationID:    firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:           stats.RunRef,
		CausalVerdict:    causalVerdict,
		CausalReasonCode: causalReasonCode,
		ExternalJob:      externalJob,
		Goal:             goal,
		Stats:            &stats,
		DecisionContext:  decisionContext,
		OpsSnapshot:      buildMCPDirectorStatsOpsSnapshotV0(stats, decisionContext, input.OccurredAt),
		Errores:          []MCPValidationIssueV0{},
	}, nil
}

func (executor MCPDirectorStatsToolExecutorV0) applyOperationalPlanStateProjectionV0(
	ctx context.Context,
	runRef string,
	stats *orquestacionnucleoapp.DirectorRunStatsV0,
) {
	if executor.OperationalPlanStateStore == nil || stats == nil {
		return
	}
	runRef = strings.TrimSpace(runRef)
	planRef := mcpDirectorStatsDefaultOperationalPlanRefV0(runRef)
	if runRef == "" || planRef == "" {
		return
	}
	state, err := executor.OperationalPlanStateStore.LoadOperationalDirectorPlanStateV0(ctx, runRef, planRef)
	if err != nil || state.Status != orquestacionnucleoapp.OperationalDirectorPlanStateClosedV0 {
		return
	}
	if mcpDirectorStatsHasHardEstadoVivoBlockV0(stats) {
		return
	}
	stats.Status = "closed"
	stats.Closure = orquestacionnucleoapp.DirectorClosureStatsV0{
		Status: orquestacionnucleoapp.DirectorClosureStatusClosedV0,
		Closed: true,
	}
	stats.Progress.PercentComplete = 100
}

func mcpDirectorStatsDefaultOperationalPlanRefV0(runRef string) string {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return ""
	}
	return "operational-director-plan-director-decisions-" + mcpDirectorStatsSafeRefPartV0(runRef)
}

func mcpDirectorStatsSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}

func mcpDirectorStatsHasHardEstadoVivoBlockV0(
	stats *orquestacionnucleoapp.DirectorRunStatsV0,
) bool {
	if stats == nil {
		return false
	}
	for _, code := range []string{
		mcpDirectorStatsEstadoVivoProcesoVivoV0,
		mcpDirectorStatsEstadoVivoConflictoV0,
		mcpDirectorStatsEstadoVivoBloqueadoV0,
		mcpDirectorStatsEstadoVivoTerminalReworkV0,
	} {
		if strings.TrimSpace(stats.Status) == code ||
			containsStringMCPV0(stats.Closure.BlockedBy, code) {
			return true
		}
	}
	return false
}

func (executor MCPDirectorStatsToolExecutorV0) resolveGoalStatsV0(
	ctx context.Context,
	runRef string,
) *MCPDirectorGoalStatsV0 {
	if executor.GoalStateSource != nil {
		state, err := executor.GoalStateSource.LoadGoalWorkStateV0(ctx, strings.TrimSpace(runRef))
		if err == nil {
			state, err = orquestagoal.NewGoalWorkStateV0(state)
			if err == nil {
				goal := mcpDirectorGoalStatsFromStateV0(state)
				executor.applyGoalMaterializedRefsV0(ctx, state, goal)
				return goal
			}
		}
	}
	return executor.resolveGoalMarkerStatsV0(ctx, runRef)
}

func (executor MCPDirectorStatsToolExecutorV0) applyGoalMaterializedRefsV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
	goal *MCPDirectorGoalStatsV0,
) {
	if executor.GoalMaterializedRefsSource == nil || goal == nil {
		return
	}
	resolved, ok, err := executor.GoalMaterializedRefsSource.ResolveDirectorGoalMaterializedRefsV0(ctx, state)
	if err != nil {
		goal.IssueCodes = compactStringsMCPV0(append(goal.IssueCodes, "goal_materialized_refs_unavailable"))
		return
	}
	if !ok {
		return
	}
	goal.ArtifactRefs = compactStringsMCPV0(append(goal.ArtifactRefs, resolved.ArtifactRefs...))
	goal.DomainReceiptRefs = compactStringsMCPV0(append(goal.DomainReceiptRefs, resolved.DomainReceiptRefs...))
	goal.ExpectedReceiptRefs = compactStringsMCPV0(append(goal.ExpectedReceiptRefs, resolved.ExpectedReceiptRefs...))
	goal.EvidenceRefs = compactStringsMCPV0(append(goal.EvidenceRefs, resolved.EvidenceRefs...))
	goal.IssueCodes = compactStringsMCPV0(append(goal.IssueCodes, resolved.IssueCodes...))
}

func mcpDirectorGoalStatsFromStateV0(
	state orquestagoal.GoalWorkStateV0,
) *MCPDirectorGoalStatsV0 {
	metadata := mcpGoalWorkStateDomainOperationalMetadataV0(state)
	goal := MCPDirectorGoalStatsV0{
		DirectorExecutionMode: "goal_first",
		RunRef:                strings.TrimSpace(state.RunRef),
		GoalRef:               strings.TrimSpace(state.GoalRef),
		ExternalGoalRef:       strings.TrimSpace(state.ExternalGoalRef),
		Status:                strings.TrimSpace(state.Status),
		CurrentPhase:          strings.TrimSpace(metadata.CurrentPhase),
		RetryFromPhase:        strings.TrimSpace(metadata.RetryFromPhase),
		OperationalReason:     strings.TrimSpace(metadata.OperationalReason),
		DomainCounters:        mergeMCPDomainOperationalCountersV0(metadata.DomainCounters),
		EvidenceRefs:          mcpDirectorGoalEvidenceRefsFromStateV0(state),
	}
	if state.LastResult != nil {
		goal.ArtifactRefs = compactStringsMCPV0(state.LastResult.ArtifactRefs)
		goal.DomainReceiptRefs = compactStringsMCPV0(state.LastResult.DomainReceiptRefs)
	}
	applyMCPDirectorGoalContextBudgetV0(&goal, mcpGoalContextBudgetFromStateV0(state))
	goal.IssueCodes = mcpDirectorGoalIssueCodesFromStateV0(state)
	if state.LastClosure != nil {
		goal.ClosureStatus = strings.TrimSpace(state.LastClosure.Status)
		goal.ClosureAccepted = state.LastClosure.Accepted
		goal.ClosureNeedsRework = state.LastClosure.NeedsRework
	}
	return &goal
}

func applyMCPDirectorGoalContextBudgetV0(
	goal *MCPDirectorGoalStatsV0,
	metric orquestagoal.GoalContextBudgetV0,
) *MCPDirectorGoalStatsV0 {
	if goal == nil {
		return nil
	}
	metric = orquestagoal.NormalizeGoalContextBudgetV0(metric)
	if orquestagoal.GoalContextBudgetEmptyV0(metric) {
		return goal
	}
	goal.ContextBudgetTotalBytes = metric.ContextBudgetTotalBytes
	goal.StaticPromptBytes = metric.StaticPromptBytes
	goal.QueriedContextBytes = metric.QueriedContextBytes
	goal.MaterializedContextBytes = metric.MaterializedContextBytes
	goal.DynamicContextBytes = metric.DynamicContextBytes
	goal.CodeContextCacheStatus = strings.TrimSpace(metric.CodeContextCacheStatus)
	return goal
}

func mcpGoalContextBudgetFromStateV0(
	state orquestagoal.GoalWorkStateV0,
) orquestagoal.GoalContextBudgetV0 {
	metric := state.ContextBudget
	metric = orquestagoal.MergeGoalContextBudgetV0(metric, state.LaunchReceipt.ContextBudget)
	if state.LastResult != nil {
		metric = orquestagoal.MergeGoalContextBudgetV0(metric, state.LastResult.ContextBudget)
	}
	return metric
}

func mcpGoalContextBudgetFromMarkerV0(
	marker orquestagoal.GoalWorkRunMarkerV0,
) orquestagoal.GoalContextBudgetV0 {
	metric := marker.ContextBudget
	if marker.LaunchReceipt != nil {
		metric = orquestagoal.MergeGoalContextBudgetV0(metric, marker.LaunchReceipt.ContextBudget)
	}
	return metric
}

func (executor MCPDirectorStatsToolExecutorV0) resolveGoalMarkerStatsV0(
	ctx context.Context,
	runRef string,
) *MCPDirectorGoalStatsV0 {
	source := executor.goalMarkerSourceV0()
	if source == nil {
		return nil
	}
	marker, err := source.LoadGoalWorkRunMarkerV0(ctx, strings.TrimSpace(runRef))
	if err != nil {
		return nil
	}
	marker, err = orquestagoal.NewGoalWorkRunMarkerV0(marker)
	if err != nil {
		return nil
	}
	goal := &MCPDirectorGoalStatsV0{
		DirectorExecutionMode: "goal_first",
		RunRef:                strings.TrimSpace(marker.RunRef),
		GoalRef:               strings.TrimSpace(marker.GoalRef),
		ExternalGoalRef:       strings.TrimSpace(marker.ExternalGoalRef),
		Status:                "goal_first_state_missing",
		ClosureStatus:         orquestagoal.GoalStatusBlockedV0,
		ClosureNeedsRework:    true,
		EvidenceRefs: compactStringsMCPV0(append(
			[]string{"evidence-ref-director-stats-goal-first-state-missing"},
			marker.EvidenceRefs...,
		)),
	}
	return applyMCPDirectorGoalContextBudgetV0(goal, mcpGoalContextBudgetFromMarkerV0(marker))
}

func (executor MCPDirectorStatsToolExecutorV0) goalMarkerSourceV0() MCPDirectorGoalRunMarkerSourcePortV0 {
	if executor.GoalMarkerSource != nil {
		return executor.GoalMarkerSource
	}
	source, _ := executor.GoalStateSource.(MCPDirectorGoalRunMarkerSourcePortV0)
	return source
}

func applyMCPDirectorGoalRunProjectionV0(
	goal *MCPDirectorGoalStatsV0,
	stats *orquestacionnucleoapp.DirectorRunStatsV0,
) {
	if goal == nil || stats == nil {
		return
	}
	status := strings.ToLower(strings.TrimSpace(goal.Status))
	if strings.TrimSpace(goal.GoalRef) == "" && status != "goal_first_state_missing" {
		return
	}
	closureStatus := strings.ToLower(strings.TrimSpace(goal.ClosureStatus))
	switch {
	case goal.ClosureAccepted || closureStatus == orquestagoal.GoalStatusAcceptedV0:
		stats.Status = "closed"
		stats.Closure = orquestacionnucleoapp.DirectorClosureStatsV0{
			Status: orquestacionnucleoapp.DirectorClosureStatusClosedV0,
			Closed: true,
		}
	case mcpDirectorGoalOperationalReasonBlocksV0(goal.OperationalReason):
		stats.Status = orquestagoal.GoalStatusBlockedV0
		stats.Closure = orquestacionnucleoapp.DirectorClosureStatsV0{
			Status:      orquestacionnucleoapp.DirectorClosureStatusBlockedV0,
			Blocked:     true,
			BlockedBy:   []string{strings.TrimSpace(goal.OperationalReason)},
			BlockerRefs: compactStringsMCPV0(goal.EvidenceRefs),
		}
	case goal.ClosureNeedsRework ||
		closureStatus == orquestagoal.GoalStatusBlockedV0 ||
		status == orquestagoal.GoalStatusBlockedV0 ||
		status == orquestagoal.GoalStatusInvalidV0 ||
		status == "goal_first_state_missing":
		blockedBy := "goal_first"
		if status == "goal_first_state_missing" {
			blockedBy = "goal_first_state_missing"
		}
		stats.Status = status
		if stats.Status == "" {
			stats.Status = orquestagoal.GoalStatusBlockedV0
		}
		if status != "goal_first_state_missing" {
			stats.Status = orquestagoal.GoalStatusBlockedV0
		}
		stats.Closure = orquestacionnucleoapp.DirectorClosureStatsV0{
			Status:      orquestacionnucleoapp.DirectorClosureStatusBlockedV0,
			Blocked:     true,
			BlockedBy:   []string{blockedBy},
			BlockerRefs: compactStringsMCPV0(goal.EvidenceRefs),
		}
	case status == orquestagoal.GoalStatusCompleteV0:
		stats.Status = "goal_complete_pending_closure"
		stats.Closure = orquestacionnucleoapp.DirectorClosureStatsV0{
			Status: orquestacionnucleoapp.DirectorClosureStatusReadyV0,
			Ready:  true,
		}
	default:
		stats.Status = orquestagoal.GoalStatusRunningV0
		stats.Closure = orquestacionnucleoapp.DirectorClosureStatsV0{
			Status:      orquestacionnucleoapp.DirectorClosureStatusBlockedV0,
			Blocked:     true,
			BlockedBy:   []string{"goal_first_running"},
			BlockerRefs: compactStringsMCPV0(goal.EvidenceRefs),
		}
	}
	applyMCPDirectorGoalProgressProjectionV0(goal, stats)
}

func applyMCPDirectorGoalProgressProjectionV0(
	goal *MCPDirectorGoalStatsV0,
	stats *orquestacionnucleoapp.DirectorRunStatsV0,
) {
	if goal == nil || stats == nil {
		return
	}
	derivedDeliveries := compactStringsMCPV0(append(
		append([]string{}, goal.DomainReceiptRefs...),
		goal.ArtifactRefs...,
	))
	if len(derivedDeliveries) > 0 {
		stats.Refs.Deliveries = compactStringsMCPV0(append(stats.Refs.Deliveries, derivedDeliveries...))
		stats.Counts.Deliveries = len(stats.Refs.Deliveries)
	}
	if !stats.Closure.Blocked || stats.Progress.TasksTotal > 0 {
		return
	}
	stats.Progress.PercentComplete = 0
	if containsStringMCPV0(goal.IssueCodes, MCPGoalFirstQAFailedPublicTextV0) {
		stats.Status = MCPGoalFirstQAFailedPublicTextV0
		stats.Closure.BlockedBy = compactStringsMCPV0(append(
			stats.Closure.BlockedBy,
			MCPGoalFirstQAFailedPublicTextV0,
		))
		stats.Closure.BlockerRefs = compactStringsMCPV0(append(stats.Closure.BlockerRefs, goal.EvidenceRefs...))
		stats.Progress.Issues = append(stats.Progress.Issues, orquestacionnucleoapp.DirectorProgressIssueV0{
			Code:    MCPGoalFirstQAFailedPublicTextV0,
			Field:   "goal_first.qa_public_text",
			Message: "goal_first materialized recoverable artifacts, but structured QA failed public/editorial text; rework public text before closure",
		})
		return
	}
	if containsStringMCPV0(goal.IssueCodes, MCPGoalFirstArtifactPathsOmittedMaterializedV0) {
		stats.Status = MCPGoalFirstArtifactPathsOmittedMaterializedV0
		stats.Closure.BlockedBy = compactStringsMCPV0(append(
			stats.Closure.BlockedBy,
			MCPGoalFirstArtifactPathsOmittedMaterializedV0,
		))
		stats.Closure.BlockerRefs = compactStringsMCPV0(append(stats.Closure.BlockerRefs, goal.EvidenceRefs...))
		stats.Progress.Issues = append(stats.Progress.Issues, orquestacionnucleoapp.DirectorProgressIssueV0{
			Code:    MCPGoalFirstArtifactPathsOmittedMaterializedV0,
			Field:   "goal_first.artifact_paths",
			Message: "goal_first terminal receipt omitted materialized artifact paths from the declared write_set; repair receipt before closure",
		})
		return
	}
	if containsStringMCPV0(goal.IssueCodes, MCPGoalFirstTerminalArtifactMissingAfterCompleteV0) {
		stats.Status = MCPGoalFirstTerminalArtifactMissingAfterCompleteV0
		stats.Closure.BlockedBy = compactStringsMCPV0(append(
			stats.Closure.BlockedBy,
			MCPGoalFirstTerminalArtifactMissingAfterCompleteV0,
		))
		stats.Closure.BlockerRefs = compactStringsMCPV0(append(stats.Closure.BlockerRefs, goal.EvidenceRefs...))
		stats.Progress.Issues = append(stats.Progress.Issues, orquestacionnucleoapp.DirectorProgressIssueV0{
			Code:    MCPGoalFirstTerminalArtifactMissingAfterCompleteV0,
			Field:   "goal_first.artifact_paths",
			Message: "goal_first terminal state declares artifact paths that are missing from the current write_set; replan or restore artifacts before closure",
		})
		return
	}
	if containsStringMCPV0(goal.IssueCodes, MCPGoalFirstOutOfScopeMaterializedArtifactsV0) {
		stats.Status = MCPGoalFirstOutOfScopeMaterializedArtifactsV0
		stats.Closure.BlockedBy = compactStringsMCPV0(append(
			stats.Closure.BlockedBy,
			MCPGoalFirstOutOfScopeMaterializedArtifactsV0,
		))
		stats.Closure.BlockerRefs = compactStringsMCPV0(append(stats.Closure.BlockerRefs, goal.EvidenceRefs...))
		stats.Progress.Issues = append(stats.Progress.Issues, orquestacionnucleoapp.DirectorProgressIssueV0{
			Code:    MCPGoalFirstOutOfScopeMaterializedArtifactsV0,
			Field:   "goal_first.write_set",
			Message: "goal_first materialized OPES artifacts outside the declared write_set; preserve them as recoverable input and rework with corrected scope before closure",
		})
		return
	}
	if containsStringMCPV0(goal.IssueCodes, mcpAutoprogrammingActionRuntimeWriteSetViolationV0) {
		stats.Status = mcpAutoprogrammingActionRuntimeWriteSetViolationV0
		stats.Closure.BlockedBy = compactStringsMCPV0(append(
			stats.Closure.BlockedBy,
			mcpAutoprogrammingActionRuntimeWriteSetViolationV0,
		))
		stats.Closure.BlockerRefs = compactStringsMCPV0(append(stats.Closure.BlockerRefs, goal.EvidenceRefs...))
		stats.Progress.Issues = append(stats.Progress.Issues, orquestacionnucleoapp.DirectorProgressIssueV0{
			Code:    mcpAutoprogrammingActionRuntimeWriteSetViolationV0,
			Field:   "goal_first.write_set",
			Message: "goal_first runtime detected file changes outside the declared write_set; preserve changed paths as evidence and rework with corrected scope before closure",
		})
		return
	}
	if containsStringMCPV0(goal.IssueCodes, MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0) {
		stats.Status = MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0
		stats.Closure.BlockedBy = compactStringsMCPV0(append(
			stats.Closure.BlockedBy,
			MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0,
		))
		stats.Closure.BlockerRefs = compactStringsMCPV0(append(stats.Closure.BlockerRefs, goal.EvidenceRefs...))
		stats.Progress.Issues = append(stats.Progress.Issues, orquestacionnucleoapp.DirectorProgressIssueV0{
			Code:    MCPGoalFirstMissingTerminalReceiptAfterArtifactsPassV0,
			Field:   "goal_first.receipt",
			Message: "goal_first has validated artifacts and QA pass evidence, but no terminal goal/domain receipt; repair receipt before declaring closure",
		})
		return
	}
	if containsStringMCPV0(goal.IssueCodes, MCPGoalFirstRequiredTestEvidenceMissingV0) {
		stats.Status = MCPGoalFirstRequiredTestEvidenceMissingV0
		stats.Closure.BlockedBy = compactStringsMCPV0(append(
			stats.Closure.BlockedBy,
			MCPGoalFirstRequiredTestEvidenceMissingV0,
		))
		stats.Closure.BlockerRefs = compactStringsMCPV0(append(stats.Closure.BlockerRefs, goal.EvidenceRefs...))
		stats.Progress.Issues = append(stats.Progress.Issues, orquestacionnucleoapp.DirectorProgressIssueV0{
			Code:    MCPGoalFirstRequiredTestEvidenceMissingV0,
			Field:   "goal_first.required_tests",
			Message: "goal_first terminal receipt declares passed required tests without evidence refs; repair receipt with durable test evidence before closure",
		})
		return
	}
	if containsStringMCPV0(goal.IssueCodes, MCPGoalFirstPhase0CompleteNonPublishableV0) {
		stats.Status = MCPGoalFirstPhase0CompleteNonPublishableV0
		stats.Closure.BlockedBy = compactStringsMCPV0(append(
			stats.Closure.BlockedBy,
			MCPGoalFirstPhase0CompleteNonPublishableV0,
		))
		stats.Closure.BlockerRefs = compactStringsMCPV0(append(stats.Closure.BlockerRefs, goal.EvidenceRefs...))
		stats.Progress.Issues = append(stats.Progress.Issues, orquestacionnucleoapp.DirectorProgressIssueV0{
			Code:    MCPGoalFirstPhase0CompleteNonPublishableV0,
			Field:   "goal_first.phase0",
			Message: "goal_first materialized a phase 0 checkpoint delivery; artifacts are recoverable but not publishable closure, continue from phase 0 before final package",
		})
		return
	}
	if containsStringMCPV0(goal.IssueCodes, MCPGoalFirstPartialArtifactsWrittenV0) {
		stats.Status = MCPGoalFirstPartialArtifactsWrittenV0
		stats.Closure.BlockedBy = compactStringsMCPV0(append(
			stats.Closure.BlockedBy,
			MCPGoalFirstPartialArtifactsWrittenV0,
		))
		stats.Closure.BlockerRefs = compactStringsMCPV0(append(stats.Closure.BlockerRefs, goal.EvidenceRefs...))
		stats.Progress.Issues = append(stats.Progress.Issues, orquestacionnucleoapp.DirectorProgressIssueV0{
			Code:    MCPGoalFirstPartialArtifactsWrittenV0,
			Field:   "goal_first.partial_artifacts",
			Message: "goal_first materialized recoverable artifacts without terminal receipt; review partial artifacts and continue or repair receipt before closure",
		})
		return
	}
	if len(derivedDeliveries) > 0 {
		stats.Closure.BlockedBy = compactStringsMCPV0(append(stats.Closure.BlockedBy, "blocked_with_partial_delivery"))
		stats.Progress.Issues = append(stats.Progress.Issues, orquestacionnucleoapp.DirectorProgressIssueV0{
			Code:    "goal_first_blocked_with_partial_delivery",
			Field:   "goal_first",
			Message: "goal_first blocked with artifact/domain receipt refs but no observed workflow tasks; requires QA/replan before closure",
		})
		return
	}
	blockedCode := "goal_first_blocked_no_artifacts"
	if containsStringMCPV0(goal.IssueCodes, "codex_app_server_goal_active_timeout") ||
		containsStringMCPV0(goal.EvidenceRefs, "codex_app_server_goal_active_timeout") {
		stats.Closure.BlockedBy = compactStringsMCPV0(append(stats.Closure.BlockedBy, "blocked_no_artifacts_timeout"))
	} else {
		stats.Closure.BlockedBy = compactStringsMCPV0(append(stats.Closure.BlockedBy, "blocked_no_artifacts"))
	}
	stats.Progress.Issues = append(stats.Progress.Issues, orquestacionnucleoapp.DirectorProgressIssueV0{
		Code:    blockedCode,
		Field:   "goal_first",
		Message: "goal_first blocked with zero observed workflow tasks and no artifact/domain receipt refs; do not report 100 percent complete",
	})
}

func mcpDirectorGoalEvidenceRefsFromStateV0(state orquestagoal.GoalWorkStateV0) []string {
	refs := append([]string{}, state.EvidenceRefs...)
	refs = append(refs, state.LaunchReceipt.EvidenceRefs...)
	if state.LastResult != nil {
		refs = append(refs, state.LastResult.EvidenceRefs...)
	}
	if state.LastClosure != nil {
		refs = append(refs, state.LastClosure.EvidenceRefs...)
	}
	return compactStringsMCPV0(refs)
}

func mcpDirectorGoalIssueCodesFromStateV0(state orquestagoal.GoalWorkStateV0) []string {
	var codes []string
	for _, issue := range state.LaunchReceipt.Issues {
		codes = append(codes, issue.Code)
	}
	if state.LastResult != nil {
		for _, issue := range state.LastResult.Issues {
			codes = append(codes, issue.Code)
		}
	}
	if state.LastClosure != nil {
		for _, issue := range state.LastClosure.Issues {
			codes = append(codes, issue.Code)
		}
	}
	return compactStringsMCPV0(codes)
}

func containsStringMCPV0(values []string, needle string) bool {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return false
	}
	for _, value := range values {
		if strings.Contains(strings.TrimSpace(value), needle) {
			return true
		}
	}
	return false
}

func mcpDirectorGoalOperationalReasonBlocksV0(reason string) bool {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "provider_timeout", "running_no_recent_progress":
		return true
	default:
		return false
	}
}

func (executor MCPDirectorStatsToolExecutorV0) applyRunControlProjectionV0(
	ctx context.Context,
	runRef string,
	stats *orquestacionnucleoapp.DirectorRunStatsV0,
) {
	if executor.RunControl == nil || stats == nil {
		return
	}
	state, err := executor.RunControl.ReadRunControlStateV0(ctx, orquestaruncontrol.RunControlReadRequestV0{
		RunRef: strings.TrimSpace(runRef),
	})
	if err != nil {
		return
	}
	orquestacionnucleoapp.ApplyDirectorRunControlStateV0(
		stats,
		string(state.Status),
		state.CheckpointRecorded,
		state.Forced,
		state.EvidenceRefs,
	)
}

func clearMCPDirectorStatsProcessRefsV0(stats *orquestacionnucleoapp.DirectorRunStatsV0) {
	if stats == nil {
		return
	}
	for index := range stats.Agents {
		stats.Agents[index].Process = nil
	}
}

func (executor MCPDirectorStatsToolExecutorV0) resolveExternalJobStatsV0(
	ctx context.Context,
	input MCPDirectorStatsToolInputV0,
	runRef string,
) (MCPDirectorExternalJobStatsV0, bool, error) {
	if executor.ExternalJobSource == nil {
		return MCPDirectorExternalJobStatsV0{}, false, nil
	}
	return executor.ExternalJobSource.ResolveDirectorExternalJobStatsV0(
		ctx,
		MCPDirectorExternalJobStatsRequestV0{
			RunRef:         firstNonEmptyMCPV0(runRef, input.RunRef),
			AppRef:         strings.TrimSpace(input.AppRef),
			ExternalJobRef: strings.TrimSpace(input.ExternalJobRef),
			CorrelationID:  firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
			OccurredAt:     strings.TrimSpace(input.OccurredAt),
		},
	)
}

func newMCPDirectorStatsErrorV0(
	input MCPDirectorStatsToolInputV0,
	code string,
	field string,
	message string,
) MCPDirectorStatsToolResultV0 {
	return MCPDirectorStatsToolResultV0{
		Estado:        MCPDirectorStatsEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:        strings.TrimSpace(input.RunRef),
		Errores: []MCPValidationIssueV0{{
			Code:    strings.TrimSpace(code),
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(message),
		}},
	}
}
