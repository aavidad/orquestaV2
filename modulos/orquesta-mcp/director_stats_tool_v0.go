package orquestamcp

import (
	"context"
	"strings"

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
	Estado          string                                                 `json:"estado"`
	RequestID       string                                                 `json:"request_id,omitempty"`
	CorrelationID   string                                                 `json:"correlation_id,omitempty"`
	RunRef          string                                                 `json:"run_ref,omitempty"`
	ExternalJob     *MCPDirectorExternalJobStatsV0                         `json:"external_job,omitempty"`
	Goal            *MCPDirectorGoalStatsV0                                `json:"goal,omitempty"`
	Stats           *orquestacionnucleoapp.DirectorRunStatsV0              `json:"stats,omitempty"`
	DecisionContext *orquestaobservability.DirectorDecisionContextV0       `json:"decision_context,omitempty"`
	OpsSnapshot     *orquestaobservability.DirectorAutonomousOpsSnapshotV0 `json:"ops_snapshot,omitempty"`
	Errores         []MCPValidationIssueV0                                 `json:"errores_publicos,omitempty"`
}

type MCPDirectorExternalJobStatsRequestV0 struct {
	RunRef         string `json:"run_ref,omitempty"`
	AppRef         string `json:"app_ref,omitempty"`
	ExternalJobRef string `json:"external_job_ref"`
	CorrelationID  string `json:"correlation_id,omitempty"`
	OccurredAt     string `json:"occurred_at,omitempty"`
}

type MCPDirectorGoalStatsV0 struct {
	DirectorExecutionMode string   `json:"director_execution_mode,omitempty"`
	RunRef                string   `json:"run_ref,omitempty"`
	GoalRef               string   `json:"goal_ref"`
	ExternalGoalRef       string   `json:"external_goal_ref,omitempty"`
	Status                string   `json:"status,omitempty"`
	ClosureStatus         string   `json:"closure_status,omitempty"`
	ClosureAccepted       bool     `json:"closure_accepted,omitempty"`
	ClosureNeedsRework    bool     `json:"closure_needs_rework,omitempty"`
	EvidenceRefs          []string `json:"evidence_refs,omitempty"`
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

type MCPDirectorStatsToolExecutorV0 struct {
	RunStore          orquestacionnucleoapp.RunStorePortV0
	RunControl        orquestaruncontrol.RunControlReaderPortV0
	ProcessRegistry   orquestacionnucleoapp.AgentProcessRegistryPortV0
	ProcessSnapshot   orquestacionnucleoapp.ProcessRuntimeIdentitySnapshotPortV0
	ProgressSource    orquestacionnucleoapp.AgentProgressObservationProviderPortV0
	AgentUsageSource  orquestacionnucleoapp.AgentUsageStatsProviderPortV0
	ExternalJobSource MCPDirectorExternalJobStatsSourcePortV0
	GoalStateSource   MCPDirectorGoalStateSourcePortV0
}

func MCPDirectorStatsDescriptorV0() MCPDirectorStatsToolDescriptorV0 {
	return MCPDirectorStatsToolDescriptorV0{
		Name:        MCPDirectorStatsToolNameV0,
		Version:     MCPDirectorStatsToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,run_ref?,app_ref?,external_job_ref?,occurred_at?,include_process_refs?,include_agent_progress?,include_agent_usage?}",
		Output:      "ok:{run_ref,external_job?,goal?{goal_ref,status,closure_status?},stats{progress,closure},decision_context,ops_snapshot}|error:{errores_publicos}",
		ResourceURI: MCPDirectorStatsResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"consulta el estado por RunStorePortV0 inyectado",
			"enriquece control de procesos solo por AgentProcessRegistryPortV0 opcional",
			"expone cierre bloqueado/ready/cerrado derivado solo del run",
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
	goal := executor.resolveGoalStatsV0(ctx, stats.RunRef)
	applyMCPDirectorGoalRunProjectionV0(goal, &stats)
	decisionContext := buildMCPDirectorDecisionContextV0(run, stats, input.OccurredAt)
	return MCPDirectorStatsToolResultV0{
		Estado:          MCPDirectorStatsEstadoOKV0,
		RequestID:       strings.TrimSpace(input.RequestID),
		CorrelationID:   firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:          stats.RunRef,
		ExternalJob:     externalJob,
		Goal:            goal,
		Stats:           &stats,
		DecisionContext: decisionContext,
		OpsSnapshot:     buildMCPDirectorStatsOpsSnapshotV0(stats, decisionContext, input.OccurredAt),
		Errores:         []MCPValidationIssueV0{},
	}, nil
}

func (executor MCPDirectorStatsToolExecutorV0) resolveGoalStatsV0(
	ctx context.Context,
	runRef string,
) *MCPDirectorGoalStatsV0 {
	if executor.GoalStateSource == nil {
		return nil
	}
	state, err := executor.GoalStateSource.LoadGoalWorkStateV0(ctx, strings.TrimSpace(runRef))
	if err != nil {
		return nil
	}
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return nil
	}
	goal := MCPDirectorGoalStatsV0{
		DirectorExecutionMode: "goal_first",
		RunRef:                strings.TrimSpace(state.RunRef),
		GoalRef:               strings.TrimSpace(state.GoalRef),
		ExternalGoalRef:       strings.TrimSpace(state.ExternalGoalRef),
		Status:                strings.TrimSpace(state.Status),
		EvidenceRefs:          compactStringsMCPV0(state.EvidenceRefs),
	}
	if state.LastClosure != nil {
		goal.ClosureStatus = strings.TrimSpace(state.LastClosure.Status)
		goal.ClosureAccepted = state.LastClosure.Accepted
		goal.ClosureNeedsRework = state.LastClosure.NeedsRework
	}
	return &goal
}

func applyMCPDirectorGoalRunProjectionV0(
	goal *MCPDirectorGoalStatsV0,
	stats *orquestacionnucleoapp.DirectorRunStatsV0,
) {
	if goal == nil || stats == nil || strings.TrimSpace(goal.GoalRef) == "" {
		return
	}
	status := strings.ToLower(strings.TrimSpace(goal.Status))
	closureStatus := strings.ToLower(strings.TrimSpace(goal.ClosureStatus))
	switch {
	case goal.ClosureAccepted || closureStatus == orquestagoal.GoalStatusAcceptedV0:
		stats.Status = "closed"
		stats.Closure = orquestacionnucleoapp.DirectorClosureStatsV0{
			Status: orquestacionnucleoapp.DirectorClosureStatusClosedV0,
			Closed: true,
		}
	case goal.ClosureNeedsRework ||
		closureStatus == orquestagoal.GoalStatusBlockedV0 ||
		status == orquestagoal.GoalStatusBlockedV0 ||
		status == orquestagoal.GoalStatusInvalidV0:
		stats.Status = orquestagoal.GoalStatusBlockedV0
		stats.Closure = orquestacionnucleoapp.DirectorClosureStatsV0{
			Status:      orquestacionnucleoapp.DirectorClosureStatusBlockedV0,
			Blocked:     true,
			BlockedBy:   []string{"goal_first"},
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
			Status: orquestacionnucleoapp.DirectorClosureStatusReadyV0,
			Ready:  true,
		}
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
