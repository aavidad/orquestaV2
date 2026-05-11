package orquestamcp

import (
	"errors"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	MCPCoreWorkflowCommandToolNameV0    = "orquesta.core_workflow.handle_command.v0"
	MCPCoreWorkflowCommandToolVersionV0 = "v0"
	MCPCoreWorkflowCommandToolURIV0     = "orquesta://core-workflow/tools/handle-command/v0"
	MCPCoreWorkflowCommandEstadoOKV0    = "ok"
	MCPCoreWorkflowCommandEstadoErrorV0 = "error"
)

type MCPCoreWorkflowCommandToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Guardrails  []string `json:"guardrails"`
}

type MCPCoreWorkflowCommandToolInputV0 struct {
	RequestID     string                                      `json:"request_id,omitempty"`
	CorrelationID string                                      `json:"correlation_id,omitempty"`
	IncludeState  bool                                        `json:"include_state,omitempty"`
	Current       orquestacoreworkflow.OrchestrationRunV0     `json:"current"`
	Command       orquestacoreworkflow.OrchestrationCommandV0 `json:"command"`
}

type MCPCoreWorkflowCommandToolResultV0 struct {
	Estado        string                                   `json:"estado"`
	RequestID     string                                   `json:"request_id,omitempty"`
	CorrelationID string                                   `json:"correlation_id,omitempty"`
	Command       MCPCoreWorkflowCommandCompactV0          `json:"command,omitempty"`
	Events        []string                                 `json:"events,omitempty"`
	Outbox        []MCPCoreWorkflowOutboxCompactV0         `json:"outbox,omitempty"`
	StateAfter    MCPCoreWorkflowStateCompactV0            `json:"state_after,omitempty"`
	StatePublic   *orquestacoreworkflow.OrchestrationRunV0 `json:"state_public,omitempty"`
	Idempotent    bool                                     `json:"idempotent,omitempty"`
	NoopReason    string                                   `json:"noop_reason,omitempty"`
	Errores       []MCPCoreWorkflowCommandErrorMCPV0       `json:"errores_publicos,omitempty"`
}

type MCPCoreWorkflowCommandCompactV0 struct {
	CommandID   string `json:"command_id,omitempty"`
	CommandType string `json:"command_type,omitempty"`
	RunID       string `json:"run_id,omitempty"`
}

type MCPCoreWorkflowOutboxCompactV0 struct {
	MessageID   string `json:"message_id,omitempty"`
	MessageType string `json:"message_type,omitempty"`
	TargetPort  string `json:"target_port,omitempty"`
}

type MCPCoreWorkflowStateCompactV0 struct {
	RunID        string `json:"run_id,omitempty"`
	Status       string `json:"status,omitempty"`
	CurrentPhase string `json:"current_phase,omitempty"`
	LastSequence int64  `json:"last_sequence,omitempty"`
	Tasks        int    `json:"tasks"`
	Agents       int    `json:"agents"`
	Deliveries   int    `json:"deliveries"`
	Reviews      int    `json:"reviews"`
	QualityGates int    `json:"quality_gates"`
	Blockers     int    `json:"blockers"`
	Closures     int    `json:"closures"`
}

type MCPCoreWorkflowCommandErrorMCPV0 struct {
	Code          string `json:"code"`
	Field         string `json:"field,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

func MCPCoreWorkflowCommandToolDescriptorV0Value() MCPCoreWorkflowCommandToolDescriptorV0 {
	return MCPCoreWorkflowCommandToolDescriptorV0{
		Name:        MCPCoreWorkflowCommandToolNameV0,
		Version:     MCPCoreWorkflowCommandToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,current:OrchestrationRunV0,command:OrchestrationCommandV0}",
		Output:      "ok:{command compacto,events,outbox,state_after}|error:{errores_publicos}",
		ResourceURI: MCPCoreWorkflowContractsResourceURIV0,
		Guardrails: []string{
			"tool_puro_sin_persistencia",
			"no_ejecuta_outbox",
			"no_arranca_runtime",
			"no_expone_payloads",
		},
	}
}

func ExecuteMCPCoreWorkflowCommandToolV0(input MCPCoreWorkflowCommandToolInputV0) MCPCoreWorkflowCommandToolResultV0 {
	result, err := orquestacoreworkflow.HandleCommandV0(input.Current, input.Command)
	if err != nil {
		return newMCPCoreWorkflowCommandErrorResultV0(input, err)
	}
	stateAfter, err := applyCoreWorkflowEventsForMCPV0(input.Current, result.Events)
	if err != nil {
		return newMCPCoreWorkflowCommandErrorResultV0(input, err)
	}
	return newMCPCoreWorkflowCommandOKResultV0(input, result, stateAfter)
}

func applyCoreWorkflowEventsForMCPV0(
	current orquestacoreworkflow.OrchestrationRunV0,
	events []orquestacoreworkflow.OrchestrationEventV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	next := current
	var err error
	for _, event := range events {
		next, err = orquestacoreworkflow.ApplyEventV0(next, event)
		if err != nil {
			return current, err
		}
	}
	return next, nil
}

func newMCPCoreWorkflowCommandOKResultV0(
	input MCPCoreWorkflowCommandToolInputV0,
	result orquestacoreworkflow.OrchestrationCommandResultV0,
	stateAfter orquestacoreworkflow.OrchestrationRunV0,
) MCPCoreWorkflowCommandToolResultV0 {
	return MCPCoreWorkflowCommandToolResultV0{
		Estado:        MCPCoreWorkflowCommandEstadoOKV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.Command.CorrelationID),
		Command:       compactCoreWorkflowCommandMCPV0(input.Command),
		Events:        compactCoreWorkflowEventsMCPV0(result.Events),
		Outbox:        compactCoreWorkflowOutboxMCPV0(result.Outbox),
		StateAfter:    compactCoreWorkflowStateMCPV0(stateAfter),
		StatePublic:   publicStateIfRequestedMCPV0(input.IncludeState, stateAfter),
		Idempotent:    result.Idempotent,
		NoopReason:    strings.TrimSpace(result.NoopReason),
	}
}

func publicStateIfRequestedMCPV0(
	include bool,
	state orquestacoreworkflow.OrchestrationRunV0,
) *orquestacoreworkflow.OrchestrationRunV0 {
	if !include {
		return nil
	}
	publicState := state
	return &publicState
}

func newMCPCoreWorkflowCommandErrorResultV0(
	input MCPCoreWorkflowCommandToolInputV0,
	err error,
) MCPCoreWorkflowCommandToolResultV0 {
	return MCPCoreWorkflowCommandToolResultV0{
		Estado:        MCPCoreWorkflowCommandEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.Command.CorrelationID),
		Command:       compactCoreWorkflowCommandMCPV0(input.Command),
		Errores:       []MCPCoreWorkflowCommandErrorMCPV0{coreWorkflowPublicErrorMCPV0(err, input)},
	}
}

func compactCoreWorkflowCommandMCPV0(command orquestacoreworkflow.OrchestrationCommandV0) MCPCoreWorkflowCommandCompactV0 {
	return MCPCoreWorkflowCommandCompactV0{
		CommandID:   strings.TrimSpace(command.CommandID),
		CommandType: strings.TrimSpace(command.CommandType),
		RunID:       strings.TrimSpace(command.RunID),
	}
}

func compactCoreWorkflowEventsMCPV0(events []orquestacoreworkflow.OrchestrationEventV0) []string {
	values := make([]string, 0, len(events))
	for _, event := range events {
		values = append(values, event.EventType)
	}
	return compactStringsMCPV0(values)
}

func compactCoreWorkflowOutboxMCPV0(outbox []orquestacoreworkflow.OutboxMessageV0) []MCPCoreWorkflowOutboxCompactV0 {
	result := make([]MCPCoreWorkflowOutboxCompactV0, 0, len(outbox))
	for _, item := range outbox {
		result = append(result, MCPCoreWorkflowOutboxCompactV0{
			MessageID:   strings.TrimSpace(item.MessageID),
			MessageType: strings.TrimSpace(item.MessageType),
			TargetPort:  strings.TrimSpace(item.TargetPort),
		})
	}
	return result
}

func compactCoreWorkflowStateMCPV0(state orquestacoreworkflow.OrchestrationRunV0) MCPCoreWorkflowStateCompactV0 {
	return MCPCoreWorkflowStateCompactV0{
		RunID:        strings.TrimSpace(state.RunID),
		Status:       string(state.Status),
		CurrentPhase: string(state.CurrentPhase),
		LastSequence: state.LastSequence,
		Tasks:        len(state.Tasks),
		Agents:       len(state.Agents),
		Deliveries:   len(state.Deliveries),
		Reviews:      len(state.Reviews),
		QualityGates: len(state.QualityGates),
		Blockers:     len(state.Blockers),
		Closures:     len(state.Closures),
	}
}

func coreWorkflowPublicErrorMCPV0(
	err error,
	input MCPCoreWorkflowCommandToolInputV0,
) MCPCoreWorkflowCommandErrorMCPV0 {
	code, field := coreWorkflowErrorCodeAndFieldMCPV0(err)
	return MCPCoreWorkflowCommandErrorMCPV0{
		Code:          code,
		Field:         field,
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.Command.CorrelationID),
	}
}

func coreWorkflowErrorCodeAndFieldMCPV0(err error) (string, string) {
	var commandErr orquestacoreworkflow.OrchestrationCommandErrorV0
	if errors.As(err, &commandErr) {
		return strings.TrimSpace(commandErr.Code), strings.TrimSpace(commandErr.Field)
	}
	var eventErr orquestacoreworkflow.OrchestrationEventErrorV0
	if errors.As(err, &eventErr) {
		return strings.TrimSpace(eventErr.Code), strings.TrimSpace(eventErr.Field)
	}
	var issue orquestacoreworkflow.OrchestrationValidationIssueV0
	if errors.As(err, &issue) {
		return string(issue.Code), strings.TrimSpace(issue.Field)
	}
	return strings.TrimSpace(err.Error()), ""
}
