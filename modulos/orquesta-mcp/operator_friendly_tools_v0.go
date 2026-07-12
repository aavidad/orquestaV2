package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
)

const (
	MCPOperatorFriendlyStatusToolNameV0   = "orquesta.status.v0"
	MCPOperatorFriendlyTasksToolNameV0    = "orquesta.tasks.list.v0"
	MCPOperatorFriendlyProjectsToolNameV0 = "orquesta.projects.list.v0"
	MCPOperatorFriendlyAgentsToolNameV0   = "orquesta.agents.list.v0"
	MCPOperatorFriendlyCommandToolNameV0  = "orquesta.operator.command.v0"
)

type MCPOperatorFriendlyQueryV0 struct {
	RequestID       string            `json:"request_id,omitempty"`
	CorrelationID   string            `json:"correlation_id,omitempty"`
	QueueRef        string            `json:"queue_ref,omitempty"`
	RunRef          string            `json:"run_ref,omitempty"`
	ProjectRef      string            `json:"project_ref,omitempty"`
	Status          string            `json:"status,omitempty"`
	CommandText     string            `json:"command_text,omitempty"`
	Question        string            `json:"question,omitempty"`
	Limit           int               `json:"limit,omitempty"`
	IncludeAgents   mcpFlexibleBoolV0 `json:"include_agents,omitempty"`
	IncludeRunStats mcpFlexibleBoolV0 `json:"include_run_stats,omitempty"`
	IncludeUsage    mcpFlexibleBoolV0 `json:"include_usage,omitempty"`
}

type MCPOperatorFriendlyStatusResultV0 struct {
	Estado      string                                `json:"estado"`
	Tool        string                                `json:"tool"`
	QueueRef    string                                `json:"queue_ref,omitempty"`
	Counts      MCPOperatorFriendlyCountsV0           `json:"counts"`
	Projects    []string                              `json:"projects,omitempty"`
	Tasks       []MCPRunQueueRankedCandidateCompactV0 `json:"tasks,omitempty"`
	Runs        []MCPDirectorStatsToolResultV0        `json:"runs,omitempty"`
	Agents      []MCPOperatorFriendlyAgentV0          `json:"agents,omitempty"`
	NextActions []string                              `json:"next_actions,omitempty"`
	Errores     []MCPValidationIssueV0                `json:"errores_publicos,omitempty"`
}

type MCPOperatorFriendlyCountsV0 struct {
	Scope               string         `json:"scope"`
	Projects            int            `json:"projects"`
	Tasks               int            `json:"tasks"`
	Runs                int            `json:"runs"`
	Agents              int            `json:"agents"`
	ByStatus            map[string]int `json:"by_status,omitempty"`
	QueueLive           bool           `json:"queue_live"`
	ActiveQueueEmpty    bool           `json:"active_queue_empty"`
	TerminalRunsVisible int            `json:"terminal_runs_visible"`
}

type MCPOperatorFriendlyAgentV0 struct {
	RunRef         string `json:"run_ref"`
	ProjectRef     string `json:"project_ref,omitempty"`
	AgentRef       string `json:"agent_ref"`
	Status         string `json:"status,omitempty"`
	InFlight       bool   `json:"in_flight,omitempty"`
	NeedsAttention bool   `json:"needs_attention,omitempty"`
	Progress       any    `json:"progress,omitempty"`
	Usage          any    `json:"usage,omitempty"`
}

func mcpOperatorFriendlyTransportToolsV0(
	bindings MCPTransportBindingsV0,
) []MCPTransportToolEnvelopeV0 {
	return []MCPTransportToolEnvelopeV0{
		mcpTransportToolEnvelopeV0(
			MCPOperatorFriendlyStatusToolNameV0,
			"v0",
			"orquesta://operator/friendly/status/v0",
			"envelope:{request_id?,correlation_id?,queue_ref?,run_ref?,project_ref?,status?,limit?,include_agents?,include_run_stats?,include_usage?}; no required fields; use for general status/como va; counts scope is active_queue",
			"ok:{counts:{scope,projects,tasks,runs,agents,queue_live,active_queue_empty,terminal_runs_visible},projects,tasks,runs?,agents?,next_actions}|error:{errores_publicos}",
			mcpOperatorFriendlyStatusTransportHandlerV0(bindings, MCPOperatorFriendlyStatusToolNameV0),
		),
		mcpTransportToolEnvelopeV0(
			MCPOperatorFriendlyTasksToolNameV0,
			"v0",
			"orquesta://operator/friendly/tasks/v0",
			"envelope:{request_id?,correlation_id?,queue_ref?,project_ref?,status?,limit?}; no required fields; list active queue tasks",
			"ok:{counts,tasks,projects}|error:{errores_publicos}",
			mcpOperatorFriendlyStatusTransportHandlerV0(bindings, MCPOperatorFriendlyTasksToolNameV0),
		),
		mcpTransportToolEnvelopeV0(
			MCPOperatorFriendlyProjectsToolNameV0,
			"v0",
			"orquesta://operator/friendly/projects/v0",
			"envelope:{request_id?,correlation_id?,queue_ref?,project_ref?,status?,limit?}; no required fields; list active projects from queue",
			"ok:{counts,projects}|error:{errores_publicos}",
			mcpOperatorFriendlyStatusTransportHandlerV0(bindings, MCPOperatorFriendlyProjectsToolNameV0),
		),
		mcpTransportToolEnvelopeV0(
			MCPOperatorFriendlyAgentsToolNameV0,
			"v0",
			"orquesta://operator/friendly/agents/v0",
			"envelope:{request_id?,correlation_id?,queue_ref?,run_ref?,project_ref?,status?,limit?,include_usage?}; no required fields; list agents using active queued runs",
			"ok:{counts,agents,runs?}|error:{errores_publicos}",
			mcpOperatorFriendlyStatusTransportHandlerV0(bindings, MCPOperatorFriendlyAgentsToolNameV0),
		),
		mcpTransportToolEnvelopeV0(
			MCPOperatorFriendlyCommandToolNameV0,
			"v0",
			"orquesta://operator/friendly/command/v0",
			"envelope:{request_id?,correlation_id?,command_text?,question?,queue_ref?,run_ref?,project_ref?,status?,limit?}; no internal refs required; routes simple human commands",
			"ok:{counts,projects,tasks,runs?,agents?,next_actions}|error:{errores_publicos}",
			mcpOperatorFriendlyStatusTransportHandlerV0(bindings, MCPOperatorFriendlyCommandToolNameV0),
		),
	}
}

func mcpOperatorFriendlyStatusTransportHandlerV0(
	bindings MCPTransportBindingsV0,
	toolName string,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPOperatorFriendlyQueryV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		result, err := executeMCPOperatorFriendlyStatusV0(ctx, bindings, toolName, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}

func executeMCPOperatorFriendlyStatusV0(
	ctx context.Context,
	bindings MCPTransportBindingsV0,
	toolName string,
	input MCPOperatorFriendlyQueryV0,
) (MCPOperatorFriendlyStatusResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if toolName == MCPOperatorFriendlyCommandToolNameV0 {
		toolName = operatorFriendlyToolForCommandV0(input)
	}
	showRuns := bool(input.IncludeRunStats) ||
		bool(input.IncludeAgents) ||
		toolName == MCPOperatorFriendlyAgentsToolNameV0 ||
		strings.TrimSpace(input.RunRef) != ""
	includeAgents := bool(input.IncludeAgents) || toolName == MCPOperatorFriendlyAgentsToolNameV0
	if includeAgents {
		showRuns = true
	}
	out := MCPOperatorFriendlyStatusResultV0{
		Estado: "ok",
		Tool:   toolName,
		Counts: MCPOperatorFriendlyCountsV0{
			Scope:    "active_queue",
			ByStatus: map[string]int{},
		},
		Errores: []MCPValidationIssueV0{},
	}
	queue, queueOK, err := mcpOperatorFriendlyQueueV0(ctx, bindings, input)
	if err != nil {
		return out, err
	}
	if !queueOK {
		out.Estado = "error"
		out.Errores = append(out.Errores, MCPValidationIssueV0{
			Code:    "run_queue_no_disponible",
			Field:   "queue",
			Message: "cola no disponible por MCP",
		})
		return out, nil
	}
	out.QueueRef = queue.QueueRef
	out.Counts.TerminalRunsVisible = len(queue.Terminal)
	baseTasks := filterMCPFriendlyTasksByProjectV0(queue.Ranked, input)
	if bindings.DirectorStats != nil && len(baseTasks) > 0 {
		runs, agents := mcpOperatorFriendlyRunStatsV0(ctx, bindings, input, baseTasks, includeAgents)
		baseTasks = projectMCPFriendlyLiveTaskStatusesV0(baseTasks, runs)
		if showRuns {
			out.Runs = runs
			out.Agents = agents
		}
	}
	out.Tasks = filterMCPFriendlyTasksV0(baseTasks, input)
	out.Projects = projectsFromMCPFriendlyTasksV0(out.Tasks)
	out.Counts.QueueLive = true
	out.Counts.ActiveQueueEmpty = len(queue.Ranked) == 0
	out.Counts.Tasks = len(out.Tasks)
	out.Counts.Projects = len(out.Projects)
	out.Counts.ByStatus = statusCountsFromMCPFriendlyTasksV0(out.Tasks)
	if showRuns {
		out.Counts.Runs = len(out.Runs)
		out.Counts.Agents = len(out.Agents)
	}
	out.NextActions = mcpOperatorFriendlyNextActionsV0(toolName, out)
	if toolName == MCPOperatorFriendlyTasksToolNameV0 {
		out.Runs = nil
		out.Agents = nil
	}
	if toolName == MCPOperatorFriendlyProjectsToolNameV0 {
		out.Tasks = nil
		out.Runs = nil
		out.Agents = nil
	}
	if toolName == MCPOperatorFriendlyAgentsToolNameV0 {
		out.Tasks = nil
		out.Projects = nil
	}
	return out, nil
}

func mcpOperatorFriendlyQueueV0(
	ctx context.Context,
	bindings MCPTransportBindingsV0,
	input MCPOperatorFriendlyQueryV0,
) (MCPRunQueuePriorityToolResultV0, bool, error) {
	if bindings.RunQueuePriority == nil {
		return MCPRunQueuePriorityToolResultV0{}, false, nil
	}
	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}
	result, err := bindings.RunQueuePriority.Execute(ctx, MCPRunQueuePriorityToolInputV0{
		RequestID:            input.RequestID,
		CorrelationID:        input.CorrelationID,
		Action:               MCPRunQueuePriorityActionRankV0,
		QueueRef:             input.QueueRef,
		Limit:                limit,
		IncludeNonExecutable: true,
	})
	if err != nil {
		return MCPRunQueuePriorityToolResultV0{}, false, err
	}
	return result, result.Estado == MCPRunQueuePriorityEstadoOKV0, nil
}

func mcpOperatorFriendlyRunStatsV0(
	ctx context.Context,
	bindings MCPTransportBindingsV0,
	input MCPOperatorFriendlyQueryV0,
	tasks []MCPRunQueueRankedCandidateCompactV0,
	includeAgents bool,
) ([]MCPDirectorStatsToolResultV0, []MCPOperatorFriendlyAgentV0) {
	runRefs := compactStringsMCPV0([]string{input.RunRef})
	if len(runRefs) == 0 {
		runRefs = runRefsFromMCPFriendlyTasksV0(tasks)
	}
	projectByRun := projectRefsByMCPFriendlyRunV0(tasks)
	runs := make([]MCPDirectorStatsToolResultV0, 0, len(runRefs))
	agents := []MCPOperatorFriendlyAgentV0{}
	for _, runRef := range runRefs {
		run, err := bindings.DirectorStats.Execute(ctx, MCPDirectorStatsToolInputV0{
			RequestID:            input.RequestID,
			CorrelationID:        input.CorrelationID,
			RunRef:               runRef,
			IncludeProcessRefs:   includeAgents,
			IncludeAgentProgress: includeAgents,
			IncludeAgentUsage:    bool(input.IncludeUsage),
		})
		if err != nil || run.Estado != MCPDirectorStatsEstadoOKV0 {
			continue
		}
		runs = append(runs, run)
		if includeAgents && run.Stats != nil {
			runRef := mcpFriendlyRunRefV0(run)
			for _, agent := range run.Stats.Agents {
				agents = append(agents, MCPOperatorFriendlyAgentV0{
					RunRef:         runRef,
					ProjectRef:     projectByRun[runRef],
					AgentRef:       agent.AgentRequestID,
					Status:         agent.Status,
					InFlight:       agent.InFlight,
					NeedsAttention: agent.NeedsAttention,
					Progress:       agent.LastProgress,
					Usage:          agent.Usage,
				})
			}
		}
	}
	return runs, agents
}
