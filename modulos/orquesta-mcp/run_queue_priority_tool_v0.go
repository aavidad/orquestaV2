package orquestamcp

import (
	"context"
	"strings"
	"time"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

const (
	MCPRunQueuePriorityToolNameV0    = "orquesta.run_queue.priority.v0"
	MCPRunQueuePriorityToolVersionV0 = "v0"
	MCPRunQueuePriorityResourceURIV0 = "orquesta://contracts/run-queue-priority/v0"
	MCPRunQueuePriorityEstadoOKV0    = "ok"
	MCPRunQueuePriorityEstadoErrorV0 = "error"
	MCPRunQueuePriorityActionRankV0  = "rank"
	MCPRunQueuePriorityActionSetV0   = "set_priority"
)

type MCPRunQueuePriorityToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPRunQueuePriorityToolInputV0 struct {
	RequestID      string   `json:"request_id,omitempty"`
	CorrelationID  string   `json:"correlation_id,omitempty"`
	Action         string   `json:"action"`
	QueueRef       string   `json:"queue_ref,omitempty"`
	AppRefs        []string `json:"app_refs,omitempty"`
	RunRef         string   `json:"run_ref,omitempty"`
	AppRef         string   `json:"app_ref,omitempty"`
	Status         string   `json:"status,omitempty"`
	PriorityScore  int      `json:"priority_score,omitempty"`
	RequestedBy    string   `json:"requested_by,omitempty"`
	Reason         string   `json:"reason,omitempty"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
	Limit          int      `json:"limit,omitempty"`
	OccurredAt     string   `json:"occurred_at,omitempty"`
}

type MCPRunQueuePriorityToolResultV0 struct {
	Estado        string                                `json:"estado"`
	RequestID     string                                `json:"request_id,omitempty"`
	CorrelationID string                                `json:"correlation_id,omitempty"`
	Action        string                                `json:"action,omitempty"`
	QueueRef      string                                `json:"queue_ref,omitempty"`
	Count         int                                   `json:"count"`
	Ranked        []MCPRunQueueRankedCandidateCompactV0 `json:"ranked,omitempty"`
	Updated       *MCPRunQueueRankedCandidateCompactV0  `json:"updated,omitempty"`
	Errores       []MCPValidationIssueV0                `json:"errores_publicos,omitempty"`
}

type MCPRunQueueRankedCandidateCompactV0 struct {
	Rank             int      `json:"rank"`
	RunRef           string   `json:"run_ref"`
	AppRef           string   `json:"app_ref"`
	Status           string   `json:"status,omitempty"`
	PriorityScore    int      `json:"priority_score"`
	AgingBoost       int      `json:"aging_boost,omitempty"`
	UpdatedAt        string   `json:"updated_at,omitempty"`
	FairnessGroupRef string   `json:"fairness_group_ref,omitempty"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
}

type MCPRunQueuePriorityToolExecutorV0 struct {
	Reader orquestarunqueue.RunQueueReaderPortV0
	Writer orquestarunqueue.RunQueuePriorityWriterPortV0
}

func MCPRunQueuePriorityDescriptorV0() MCPRunQueuePriorityToolDescriptorV0 {
	return MCPRunQueuePriorityToolDescriptorV0{
		Name:        MCPRunQueuePriorityToolNameV0,
		Version:     MCPRunQueuePriorityToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,action:rank|set_priority,queue_ref?,app_refs?,run_ref?,app_ref?,status?,priority_score?,limit?,occurred_at?}",
		Output:      "ok:{action,queue_ref,count,ranked?,updated?}|error:{errores_publicos}",
		ResourceURI: MCPRunQueuePriorityResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"consulta candidatos por RunQueueReaderPortV0 inyectado",
			"muta prioridad solo por RunQueuePriorityWriterPortV0 inyectado",
			"rank usa orquesta-run-queue sin duplicar reglas",
			"sin DB runtime filesystem ni scheduler interno",
		},
	}
}

func (executor MCPRunQueuePriorityToolExecutorV0) Execute(
	ctx context.Context,
	input MCPRunQueuePriorityToolInputV0,
) (MCPRunQueuePriorityToolResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	action := strings.ToLower(strings.TrimSpace(input.Action))
	if action == "" {
		action = MCPRunQueuePriorityActionRankV0
	}
	switch action {
	case MCPRunQueuePriorityActionRankV0:
		return executor.executeRankV0(ctx, input)
	case MCPRunQueuePriorityActionSetV0:
		return executor.executeSetPriorityV0(ctx, input)
	default:
		return newMCPRunQueuePriorityErrorV0(input, "action_no_soportada", "action", "action debe ser rank o set_priority"), nil
	}
}

func (executor MCPRunQueuePriorityToolExecutorV0) executeRankV0(
	ctx context.Context,
	input MCPRunQueuePriorityToolInputV0,
) (MCPRunQueuePriorityToolResultV0, error) {
	if executor.Reader == nil {
		return newMCPRunQueuePriorityErrorV0(input, "run_queue_reader_no_disponible", "reader", "reader requerido"), nil
	}
	request := orquestarunqueue.RunQueueReadRequestV0{
		QueueRef: strings.TrimSpace(input.QueueRef),
		AppRefs:  compactStringsMCPV0(input.AppRefs),
		Limit:    input.Limit,
	}
	candidates, err := executor.Reader.ListRunSchedulingCandidatesV0(ctx, request)
	if err != nil {
		return newMCPRunQueuePriorityErrorV0(input, "run_queue_no_disponible", "reader", "cola no disponible"), nil
	}
	policy := orquestarunqueue.DefaultRunQueueRankingPolicyV0(parseMCPRunQueueOccurredAtV0(input.OccurredAt))
	ranked := orquestarunqueue.RankRunCandidatesV0(candidates, policy)
	ranked = limitMCPRunQueueRankedV0(ranked, input.Limit)
	return MCPRunQueuePriorityToolResultV0{
		Estado:        MCPRunQueuePriorityEstadoOKV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Action:        MCPRunQueuePriorityActionRankV0,
		QueueRef:      request.QueueRef,
		Count:         len(ranked),
		Ranked:        compactRunQueueRankedCandidatesMCPV0(ranked),
		Errores:       []MCPValidationIssueV0{},
	}, nil
}

func (executor MCPRunQueuePriorityToolExecutorV0) executeSetPriorityV0(
	ctx context.Context,
	input MCPRunQueuePriorityToolInputV0,
) (MCPRunQueuePriorityToolResultV0, error) {
	if executor.Writer == nil {
		return newMCPRunQueuePriorityErrorV0(input, "run_queue_writer_no_disponible", "writer", "writer requerido"), nil
	}
	command := orquestarunqueue.NormalizeRunQueuePriorityCommandV0(orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:         input.RunRef,
		QueueRef:       input.QueueRef,
		AppRef:         input.AppRef,
		Status:         input.Status,
		PriorityScore:  input.PriorityScore,
		UpdatedAt:      parseMCPRunQueueOccurredAtV0(input.OccurredAt),
		RequestedBy:    input.RequestedBy,
		Reason:         input.Reason,
		IdempotencyKey: input.IdempotencyKey,
		EvidenceRefs:   input.EvidenceRefs,
	})
	if issues := orquestarunqueue.ValidateRunQueuePriorityCommandV0(command); len(issues) > 0 {
		return newMCPRunQueuePriorityErrorV0(input, issues[0].Code, issues[0].Field, issues[0].Code), nil
	}
	updated, err := executor.Writer.SetRunPriorityV0(ctx, command)
	if err != nil {
		return newMCPRunQueuePriorityErrorV0(input, "run_queue_priority_no_disponible", "writer", "prioridad no disponible"), nil
	}
	compact := compactRunQueueRankedCandidatesMCPV0([]orquestarunqueue.RankedRunCandidateV0{{
		RunSchedulingCandidateV0: updated,
	}})
	return MCPRunQueuePriorityToolResultV0{
		Estado:        MCPRunQueuePriorityEstadoOKV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Action:        MCPRunQueuePriorityActionSetV0,
		QueueRef:      strings.TrimSpace(input.QueueRef),
		Count:         1,
		Updated:       &compact[0],
		Errores:       []MCPValidationIssueV0{},
	}, nil
}

func parseMCPRunQueueOccurredAtV0(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func limitMCPRunQueueRankedV0(
	values []orquestarunqueue.RankedRunCandidateV0,
	limit int,
) []orquestarunqueue.RankedRunCandidateV0 {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func compactRunQueueRankedCandidatesMCPV0(
	values []orquestarunqueue.RankedRunCandidateV0,
) []MCPRunQueueRankedCandidateCompactV0 {
	out := make([]MCPRunQueueRankedCandidateCompactV0, 0, len(values))
	for _, value := range values {
		out = append(out, MCPRunQueueRankedCandidateCompactV0{
			Rank:             value.Rank,
			RunRef:           strings.TrimSpace(value.RunRef),
			AppRef:           strings.TrimSpace(value.AppRef),
			Status:           strings.TrimSpace(value.Status),
			PriorityScore:    value.PriorityScore,
			AgingBoost:       value.AgingBoost,
			UpdatedAt:        formatMCPRunQueueTimeV0(value.UpdatedAt),
			FairnessGroupRef: strings.TrimSpace(value.FairnessGroupRef),
			EvidenceRefs:     compactStringsMCPV0(value.EvidenceRefs),
		})
	}
	if out == nil {
		return []MCPRunQueueRankedCandidateCompactV0{}
	}
	return out
}

func formatMCPRunQueueTimeV0(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func newMCPRunQueuePriorityErrorV0(
	input MCPRunQueuePriorityToolInputV0,
	code string,
	field string,
	message string,
) MCPRunQueuePriorityToolResultV0 {
	return MCPRunQueuePriorityToolResultV0{
		Estado:        MCPRunQueuePriorityEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Action:        strings.ToLower(strings.TrimSpace(input.Action)),
		QueueRef:      strings.TrimSpace(input.QueueRef),
		Errores: []MCPValidationIssueV0{{
			Code:    strings.TrimSpace(code),
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(message),
		}},
	}
}
