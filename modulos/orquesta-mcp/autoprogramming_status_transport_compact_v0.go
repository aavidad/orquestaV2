package orquestamcp

import "encoding/json"

const (
	mcpAutoprogrammingStatusTransportTargetBytesV0 = 48 * 1024
	mcpAutoprogrammingStatusTransportListLimitV0   = 6
	mcpAutoprogrammingStatusTransportRefsLimitV0   = 4
)

type mcpAutoprogrammingStatusOutputProjectionV0 struct {
	Mode               string `json:"mode"`
	Truncated          bool   `json:"truncated"`
	ObservedBytes      int    `json:"observed_bytes"`
	ReturnedBytes      int    `json:"returned_bytes,omitempty"`
	TargetBytes        int    `json:"target_bytes"`
	QueueRankedTotal   int    `json:"queue_ranked_total,omitempty"`
	QueueTerminalTotal int    `json:"queue_terminal_total,omitempty"`
	StaleRunningTotal  int    `json:"stale_running_total,omitempty"`
	ResolvedRunsTotal  int    `json:"resolved_runs_total,omitempty"`
	ProjectsTotal      int    `json:"projects_total,omitempty"`
	TasksTotal         int    `json:"tasks_total,omitempty"`
	AgentsTotal        int    `json:"agents_total,omitempty"`
	DiagnosticsTotal   int    `json:"diagnostics_total,omitempty"`
	EvidenceRefsTotal  int    `json:"evidence_refs_total,omitempty"`
	DetailTool         string `json:"detail_tool,omitempty"`
	DetailHTTPPath     string `json:"detail_http_path,omitempty"`
}

type mcpAutoprogrammingStatusTransportProjectedResultV0 struct {
	MCPAutoprogrammingStatusToolResultV0
	OperatorAdvice   []MCPAutoprogrammingOperatorAdviceV0        `json:"operator_advice,omitempty"`
	OutputProjection *mcpAutoprogrammingStatusOutputProjectionV0 `json:"output_projection,omitempty"`
}

func marshalMCPAutoprogrammingStatusTransportV0(
	result MCPAutoprogrammingStatusToolResultV0,
	operatorAdvice []MCPAutoprogrammingOperatorAdviceV0,
) (json.RawMessage, error) {
	full := mcpAutoprogrammingStatusTransportProjectedResultV0{
		MCPAutoprogrammingStatusToolResultV0: result,
		OperatorAdvice:                       operatorAdvice,
	}
	payload, err := json.Marshal(full)
	if err != nil || len(payload) <= mcpAutoprogrammingStatusTransportTargetBytesV0 {
		return payload, err
	}
	projection := newMCPAutoprogrammingStatusOutputProjectionV0(result, len(payload))
	compact := compactMCPAutoprogrammingStatusTransportV0(result)
	projected := mcpAutoprogrammingStatusTransportProjectedResultV0{
		MCPAutoprogrammingStatusToolResultV0: compact,
		OperatorAdvice:                       compactMCPAutoprogrammingAdviceV0(operatorAdvice),
		OutputProjection:                     &projection,
	}
	payload, err = json.Marshal(projected)
	if err != nil {
		return nil, err
	}
	if len(payload) > mcpAutoprogrammingStatusTransportTargetBytesV0 {
		projected.MCPAutoprogrammingStatusToolResultV0 = minimalMCPAutoprogrammingStatusTransportV0(compact)
		payload, err = json.Marshal(projected)
		if err != nil {
			return nil, err
		}
	}
	if len(payload) > mcpAutoprogrammingStatusTransportTargetBytesV0 {
		projected.MCPAutoprogrammingStatusToolResultV0 = strictMinimalMCPAutoprogrammingStatusTransportV0(compact)
		payload, err = json.Marshal(projected)
		if err != nil {
			return nil, err
		}
	}
	if len(payload) > mcpAutoprogrammingStatusTransportTargetBytesV0 {
		projected.OperatorAdvice = nil
		payload, err = json.Marshal(projected)
		if err != nil {
			return nil, err
		}
	}
	projection.ReturnedBytes = len(payload)
	projected.OutputProjection = &projection
	payload, err = json.Marshal(projected)
	if err != nil {
		return nil, err
	}
	if projection.ReturnedBytes != len(payload) {
		projection.ReturnedBytes = len(payload)
		projected.OutputProjection = &projection
		return json.Marshal(projected)
	}
	return payload, nil
}

func strictMinimalMCPAutoprogrammingStatusTransportV0(
	result MCPAutoprogrammingStatusToolResultV0,
) MCPAutoprogrammingStatusToolResultV0 {
	return MCPAutoprogrammingStatusToolResultV0{
		Estado:           compactMCPAutoprogrammingStatusScalarV0(result.Estado),
		RequestID:        compactMCPAutoprogrammingStatusScalarV0(result.RequestID),
		CorrelationID:    compactMCPAutoprogrammingStatusScalarV0(result.CorrelationID),
		RunRef:           compactMCPAutoprogrammingStatusScalarV0(result.RunRef),
		CausalVerdict:    compactMCPAutoprogrammingStatusScalarV0(result.CausalVerdict),
		CausalReasonCode: compactMCPAutoprogrammingStatusScalarV0(result.CausalReasonCode),
		ScopeMode:        compactMCPAutoprogrammingStatusScalarV0(result.ScopeMode),
		Scope:            compactMCPAutoprogrammingStatusScalarV0(result.Scope),
		QueueRef:         compactMCPAutoprogrammingStatusScalarV0(result.QueueRef),
		Diagnostics: []MCPAutoprogrammingDiagnosticV0{{
			Code:    "mcp_status_output_compacted",
			Scope:   "transport",
			Message: "salida MCP compactada al resumen minimo; usa output_projection y el endpoint HTTP para detalle completo",
			EvidenceRefs: []string{
				"evidence-ref-mcp-autoprogramming-status-output-compacted",
			},
		}},
	}
}

func compactMCPAutoprogrammingStatusScalarV0(value string) string {
	const limit = 512
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func newMCPAutoprogrammingStatusOutputProjectionV0(
	result MCPAutoprogrammingStatusToolResultV0,
	observedBytes int,
) mcpAutoprogrammingStatusOutputProjectionV0 {
	projection := mcpAutoprogrammingStatusOutputProjectionV0{
		Mode: "compact", Truncated: true,
		ObservedBytes: observedBytes, TargetBytes: mcpAutoprogrammingStatusTransportTargetBytesV0,
		StaleRunningTotal: result.StaleRunningTotal, ResolvedRunsTotal: len(result.ResolvedRuns),
		ProjectsTotal: len(result.Projects), TasksTotal: len(result.Tasks), AgentsTotal: len(result.Agents),
		DiagnosticsTotal: result.DiagnosticsTotal, EvidenceRefsTotal: len(result.EvidenceRefs),
		DetailTool: MCPAutoprogrammingStatusToolNameV0, DetailHTTPPath: MCPAutoprogrammingStatusHTTPPathV0,
	}
	if projection.StaleRunningTotal == 0 {
		projection.StaleRunningTotal = len(result.StaleRunning)
	}
	if projection.DiagnosticsTotal == 0 {
		projection.DiagnosticsTotal = len(result.Diagnostics)
	}
	if result.Queue != nil {
		projection.QueueRankedTotal = len(result.Queue.Ranked)
		projection.QueueTerminalTotal = len(result.Queue.Terminal)
	}
	return projection
}

func compactMCPAutoprogrammingStatusTransportV0(
	result MCPAutoprogrammingStatusToolResultV0,
) MCPAutoprogrammingStatusToolResultV0 {
	compact := result
	compact.Queue = compactMCPAutoprogrammingStatusQueueV0(result.Queue)
	compact.Run = compactMCPAutoprogrammingStatusRunV0(result.Run)
	compact.StaleRunning = compactMCPAutoprogrammingStatusActionableRunsV0(result.StaleRunning)
	compact.ResolvedRuns = compactMCPAutoprogrammingStatusActionableRunsV0(result.ResolvedRuns)
	compact.Projects = compactMCPAutoprogrammingStatusProjectsV0(result.Projects)
	compact.Tasks = compactMCPAutoprogrammingStatusTasksV0(result.Tasks)
	compact.Agents = compactMCPAutoprogrammingStatusAgentsV0(result.Agents)
	compact.Operator = compactMCPAutoprogrammingStatusOperatorV0(result.Operator)
	compact.OpsSnapshot = nil
	compact.Diagnostics = compactMCPAutoprogrammingStatusDiagnosticsV0(result.Diagnostics)
	compact.EvidenceRefs = limitMCPAutoprogrammingStatusStringsV0(result.EvidenceRefs, 12)
	compact.Errores = limitMCPAutoprogrammingStatusItemsV0(result.Errores, mcpAutoprogrammingStatusTransportListLimitV0)
	if compact.EfficiencySummary != nil {
		summary := *compact.EfficiencySummary
		summary.Reasons = limitMCPAutoprogrammingStatusStringsV0(summary.Reasons, 8)
		compact.EfficiencySummary = &summary
	}
	compact.Diagnostics = append(compact.Diagnostics, MCPAutoprogrammingDiagnosticV0{
		Code: "mcp_status_output_compacted", Scope: "transport",
		Message:      "salida MCP compactada; usa output_projection y el endpoint HTTP para detalle completo",
		EvidenceRefs: []string{"evidence-ref-mcp-autoprogramming-status-output-compacted"},
	})
	return compact
}

func minimalMCPAutoprogrammingStatusTransportV0(
	result MCPAutoprogrammingStatusToolResultV0,
) MCPAutoprogrammingStatusToolResultV0 {
	minimal := MCPAutoprogrammingStatusToolResultV0{
		Estado: result.Estado, RequestID: result.RequestID, CorrelationID: result.CorrelationID,
		RunRef: result.RunRef, CausalVerdict: result.CausalVerdict, CausalReasonCode: result.CausalReasonCode,
		QueueRef: result.QueueRef, QueueHealth: result.QueueHealth,
		Run: result.Run, GoalProgressPolicy: result.GoalProgressPolicy,
		EfficiencySummary: result.EfficiencySummary, IdleSelfImprovementBudget: result.IdleSelfImprovementBudget,
		EvidenceRefs: limitMCPAutoprogrammingStatusStringsV0(result.EvidenceRefs, 8),
		Diagnostics:  limitMCPAutoprogrammingStatusItemsV0(result.Diagnostics, 4),
		Errores:      limitMCPAutoprogrammingStatusItemsV0(result.Errores, 4),
	}
	if result.Queue != nil {
		queue := *result.Queue
		queue.Ranked = nil
		queue.Terminal = nil
		queue.Updated = nil
		minimal.Queue = &queue
	}
	return minimal
}

func compactMCPAutoprogrammingStatusQueueV0(
	queue *MCPRunQueuePriorityToolResultV0,
) *MCPRunQueuePriorityToolResultV0 {
	if queue == nil {
		return nil
	}
	compact := *queue
	compact.Ranked = compactMCPAutoprogrammingStatusCandidatesV0(queue.Ranked)
	compact.Terminal = compactMCPAutoprogrammingStatusCandidatesV0(queue.Terminal)
	return &compact
}

func compactMCPAutoprogrammingStatusCandidatesV0(
	items []MCPRunQueueRankedCandidateCompactV0,
) []MCPRunQueueRankedCandidateCompactV0 {
	items = limitMCPAutoprogrammingStatusItemsV0(items, mcpAutoprogrammingStatusTransportListLimitV0)
	for index := range items {
		items[index].WriteSetRefs = limitMCPAutoprogrammingStatusStringsV0(items[index].WriteSetRefs, 2)
		items[index].EvidenceRefs = limitMCPAutoprogrammingStatusStringsV0(items[index].EvidenceRefs, 2)
	}
	return items
}

func compactMCPAutoprogrammingStatusRunV0(
	run *MCPDirectorStatsToolResultV0,
) *MCPDirectorStatsToolResultV0 {
	if run == nil {
		return nil
	}
	compact := *run
	compact.Stats = nil
	compact.DecisionContext = nil
	compact.OpsSnapshot = nil
	compact.ExternalJob = nil
	if run.Goal != nil {
		goal := *run.Goal
		goal.ArtifactRefs = limitMCPAutoprogrammingStatusStringsV0(goal.ArtifactRefs, mcpAutoprogrammingStatusTransportRefsLimitV0)
		goal.DomainReceiptRefs = limitMCPAutoprogrammingStatusStringsV0(goal.DomainReceiptRefs, mcpAutoprogrammingStatusTransportRefsLimitV0)
		goal.ExpectedReceiptRefs = limitMCPAutoprogrammingStatusStringsV0(goal.ExpectedReceiptRefs, mcpAutoprogrammingStatusTransportRefsLimitV0)
		goal.IssueCodes = limitMCPAutoprogrammingStatusStringsV0(goal.IssueCodes, mcpAutoprogrammingStatusTransportRefsLimitV0)
		goal.EvidenceRefs = limitMCPAutoprogrammingStatusStringsV0(goal.EvidenceRefs, mcpAutoprogrammingStatusTransportRefsLimitV0)
		compact.Goal = &goal
	}
	return &compact
}

func compactMCPAutoprogrammingStatusActionableRunsV0(
	items []MCPAutoprogrammingActionableRunV0,
) []MCPAutoprogrammingActionableRunV0 {
	items = limitMCPAutoprogrammingStatusItemsV0(items, mcpAutoprogrammingStatusTransportListLimitV0)
	for index := range items {
		items[index].ProcessRefs = limitMCPAutoprogrammingStatusStringsV0(items[index].ProcessRefs, 2)
		items[index].ArtifactRefs = limitMCPAutoprogrammingStatusStringsV0(items[index].ArtifactRefs, 2)
		items[index].DomainReceiptRefs = limitMCPAutoprogrammingStatusStringsV0(items[index].DomainReceiptRefs, 2)
		items[index].ExpectedReceiptRefs = limitMCPAutoprogrammingStatusStringsV0(items[index].ExpectedReceiptRefs, 2)
		items[index].EvidenceRefs = limitMCPAutoprogrammingStatusStringsV0(items[index].EvidenceRefs, mcpAutoprogrammingStatusTransportRefsLimitV0)
	}
	return items
}

func compactMCPAutoprogrammingStatusProjectsV0(items []MCPAutoprogrammingProjectV0) []MCPAutoprogrammingProjectV0 {
	items = limitMCPAutoprogrammingStatusItemsV0(items, 12)
	for index := range items {
		items[index].RunRefs = limitMCPAutoprogrammingStatusStringsV0(items[index].RunRefs, mcpAutoprogrammingStatusTransportRefsLimitV0)
	}
	return items
}

func compactMCPAutoprogrammingStatusTasksV0(items []MCPAutoprogrammingTaskV0) []MCPAutoprogrammingTaskV0 {
	items = limitMCPAutoprogrammingStatusItemsV0(items, 12)
	for index := range items {
		items[index].EvidenceRefs = limitMCPAutoprogrammingStatusStringsV0(items[index].EvidenceRefs, 2)
	}
	return items
}

func compactMCPAutoprogrammingStatusAgentsV0(items []MCPAutoprogrammingAgentV0) []MCPAutoprogrammingAgentV0 {
	items = limitMCPAutoprogrammingStatusItemsV0(items, 12)
	for index := range items {
		items[index].EvidenceRefs = limitMCPAutoprogrammingStatusStringsV0(items[index].EvidenceRefs, 2)
	}
	return items
}

func compactMCPAutoprogrammingStatusDiagnosticsV0(items []MCPAutoprogrammingDiagnosticV0) []MCPAutoprogrammingDiagnosticV0 {
	items = limitMCPAutoprogrammingStatusItemsV0(items, 10)
	for index := range items {
		items[index].EvidenceRefs = limitMCPAutoprogrammingStatusStringsV0(items[index].EvidenceRefs, 2)
	}
	return items
}

func compactMCPAutoprogrammingStatusOperatorV0(operator *MCPAutoprogrammingOperatorV0) *MCPAutoprogrammingOperatorV0 {
	if operator == nil {
		return nil
	}
	compact := *operator
	compact.ActiveRuns = limitMCPAutoprogrammingStatusItemsV0(operator.ActiveRuns, mcpAutoprogrammingStatusTransportListLimitV0)
	compact.ClosureBlockers = limitMCPAutoprogrammingStatusItemsV0(operator.ClosureBlockers, mcpAutoprogrammingStatusTransportListLimitV0)
	for index := range compact.ClosureBlockers {
		compact.ClosureBlockers[index].Evidence = limitMCPAutoprogrammingStatusStringsV0(compact.ClosureBlockers[index].Evidence, 2)
	}
	compact.AgentsInFlight = limitMCPAutoprogrammingStatusItemsV0(operator.AgentsInFlight, mcpAutoprogrammingStatusTransportListLimitV0)
	compact.SupervisorErrors = limitMCPAutoprogrammingStatusItemsV0(operator.SupervisorErrors, mcpAutoprogrammingStatusTransportListLimitV0)
	compact.SafeActions = limitMCPAutoprogrammingStatusItemsV0(operator.SafeActions, mcpAutoprogrammingStatusTransportListLimitV0)
	return &compact
}

func compactMCPAutoprogrammingAdviceV0(items []MCPAutoprogrammingOperatorAdviceV0) []MCPAutoprogrammingOperatorAdviceV0 {
	items = limitMCPAutoprogrammingStatusItemsV0(items, mcpAutoprogrammingStatusTransportListLimitV0)
	for index := range items {
		items[index].AdviceRef = compactMCPAutoprogrammingStatusScalarV0(items[index].AdviceRef)
		items[index].OperatorRef = compactMCPAutoprogrammingStatusScalarV0(items[index].OperatorRef)
		items[index].TargetRef = compactMCPAutoprogrammingStatusScalarV0(items[index].TargetRef)
		items[index].SubjectRef = compactMCPAutoprogrammingStatusScalarV0(items[index].SubjectRef)
		items[index].Ref = compactMCPAutoprogrammingStatusScalarV0(items[index].Ref)
		items[index].Target = compactMCPAutoprogrammingStatusScalarV0(items[index].Target)
		items[index].RunRef = compactMCPAutoprogrammingStatusScalarV0(items[index].RunRef)
		items[index].Run = compactMCPAutoprogrammingStatusScalarV0(items[index].Run)
		items[index].TaskRef = compactMCPAutoprogrammingStatusScalarV0(items[index].TaskRef)
		items[index].Task = compactMCPAutoprogrammingStatusScalarV0(items[index].Task)
		items[index].Action = compactMCPAutoprogrammingStatusScalarV0(items[index].Action)
		items[index].Kind = compactMCPAutoprogrammingStatusScalarV0(items[index].Kind)
		items[index].Message = compactMCPAutoprogrammingStatusScalarV0(items[index].Message)
		items[index].Advice = compactMCPAutoprogrammingStatusScalarV0(items[index].Advice)
		items[index].Text = compactMCPAutoprogrammingStatusScalarV0(items[index].Text)
		items[index].EvidenceRefs = limitMCPAutoprogrammingStatusStringsV0(items[index].EvidenceRefs, 4)
	}
	return items
}

func limitMCPAutoprogrammingStatusStringsV0(items []string, limit int) []string {
	return limitMCPAutoprogrammingStatusItemsV0(compactStringsMCPV0(items), limit)
}

func limitMCPAutoprogrammingStatusItemsV0[T any](items []T, limit int) []T {
	if len(items) == 0 || limit <= 0 {
		return nil
	}
	if len(items) < limit {
		limit = len(items)
	}
	return append([]T(nil), items[:limit]...)
}
