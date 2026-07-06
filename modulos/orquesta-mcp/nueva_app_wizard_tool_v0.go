package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
)

const (
	MCPNuevaAppWizardToolNameV0    = "orquesta.nueva_app.wizard.v0"
	MCPNuevaAppWizardToolVersionV0 = "v0"
	MCPNuevaAppWizardResourceURIV0 = "orquesta://contracts/nueva-app-wizard/v0"
	MCPNuevaAppWizardEstadoOKV0    = "ok"
	MCPNuevaAppWizardEstadoErrorV0 = "error"
)

type MCPNuevaAppWizardToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPNuevaAppWizardToolInputV0 struct {
	RequestID        string                      `json:"request_id,omitempty"`
	CorrelationID    string                      `json:"correlation_id,omitempty"`
	SessionID        string                      `json:"session_id,omitempty"`
	Locale           string                      `json:"locale,omitempty"`
	Nombre           string                      `json:"nombre,omitempty"`
	Idea             string                      `json:"idea,omitempty"`
	Need             string                      `json:"need,omitempty"`
	ActionID         string                      `json:"action_id,omitempty"`
	ActionIDs        []string                    `json:"action_ids,omitempty"`
	AnswerField      string                      `json:"answer_field,omitempty"`
	Answer           string                      `json:"answer,omitempty"`
	GlossaryExpanded bool                        `json:"glossary_expanded,omitempty"`
	WizardAnswers    []MCPNuevaAppWizardAnswerV0 `json:"wizard_answers,omitempty"`
	Session          json.RawMessage             `json:"session,omitempty"`
	IdempotencyKey   string                      `json:"idempotency_key,omitempty"`
}

type MCPNuevaAppWizardAnswerV0 struct {
	QuestionRef        string `json:"question_ref"`
	UserChoice         string `json:"user_choice"`
	FreeText           bool   `json:"free_text,omitempty"`
	ComprehensionQuery string `json:"comprehension_query,omitempty"`
	Justification      string `json:"justification,omitempty"`
}

type MCPNuevaAppWizardToolResultV0 struct {
	Estado        string                       `json:"estado"`
	RequestID     string                       `json:"request_id,omitempty"`
	CorrelationID string                       `json:"correlation_id,omitempty"`
	SchemaVersion string                       `json:"schema_version,omitempty"`
	Assistant     string                       `json:"assistant_status,omitempty"`
	Warnings      []MCPNuevaAppWizardWarningV0 `json:"warnings,omitempty"`
	Turn          json.RawMessage              `json:"turn,omitempty"`
	Session       json.RawMessage              `json:"session,omitempty"`
	Wizard        json.RawMessage              `json:"wizard,omitempty"`
	Errores       []MCPValidationIssueV0       `json:"errores_publicos,omitempty"`
}

type MCPNuevaAppWizardWarningV0 struct {
	Code    string `json:"code"`
	Scope   string `json:"scope,omitempty"`
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

type MCPNuevaAppWizardExecutorPortV0 interface {
	Execute(context.Context, MCPNuevaAppWizardToolInputV0) (MCPNuevaAppWizardToolResultV0, error)
}

type MCPNuevaAppWizardToolExecutorV0 struct {
	Port MCPNuevaAppWizardExecutorPortV0
}

func NewMCPNuevaAppWizardToolExecutorV0(port MCPNuevaAppWizardExecutorPortV0) MCPNuevaAppWizardToolExecutorV0 {
	return MCPNuevaAppWizardToolExecutorV0{Port: port}
}

func MCPNuevaAppWizardDescriptorV0() MCPNuevaAppWizardToolDescriptorV0 {
	return MCPNuevaAppWizardToolDescriptorV0{
		Name:        MCPNuevaAppWizardToolNameV0,
		Version:     MCPNuevaAppWizardToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,session_id?,locale?,nombre?,idea?,need?,action_id?,action_ids?,answer_field?,answer?,glossary_expanded?,wizard_answers?[question_ref,user_choice,free_text?,comprehension_query?,justification?],session?,idempotency_key?}",
		Output:      "ok:{turn,session,wizard}|error:{errores_publicos}",
		ResourceURI: MCPNuevaAppWizardResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"delega en el intake guiado canonico de nueva app",
			"no lanza agentes ni director",
			"conserva recomendaciones y contrastes del wizard",
		},
	}
}

func (executor MCPNuevaAppWizardToolExecutorV0) Execute(
	ctx context.Context,
	input MCPNuevaAppWizardToolInputV0,
) (MCPNuevaAppWizardToolResultV0, error) {
	input = NormalizeMCPNuevaAppWizardInputV0(input)
	if executor.Port == nil {
		return NewMCPNuevaAppWizardErrorResultV0(input, MCPTransportToolUnboundV0, "executor"), nil
	}
	result, err := executor.Port.Execute(ctx, input)
	if err != nil {
		return result, err
	}
	return NormalizeMCPNuevaAppWizardResultV0(input, result), nil
}

func NormalizeMCPNuevaAppWizardInputV0(input MCPNuevaAppWizardToolInputV0) MCPNuevaAppWizardToolInputV0 {
	input.RequestID = strings.TrimSpace(input.RequestID)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.SessionID = strings.TrimSpace(input.SessionID)
	input.Locale = strings.TrimSpace(input.Locale)
	input.Nombre = strings.TrimSpace(input.Nombre)
	input.Idea = strings.TrimSpace(input.Idea)
	input.Need = strings.TrimSpace(input.Need)
	input.ActionID = strings.TrimSpace(input.ActionID)
	input.ActionIDs = compactStringsMCPV0(input.ActionIDs)
	input.AnswerField = strings.TrimSpace(input.AnswerField)
	input.Answer = strings.TrimSpace(input.Answer)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	for idx := range input.WizardAnswers {
		input.WizardAnswers[idx].QuestionRef = strings.TrimSpace(input.WizardAnswers[idx].QuestionRef)
		input.WizardAnswers[idx].UserChoice = strings.TrimSpace(input.WizardAnswers[idx].UserChoice)
		input.WizardAnswers[idx].ComprehensionQuery = strings.TrimSpace(input.WizardAnswers[idx].ComprehensionQuery)
		input.WizardAnswers[idx].Justification = strings.TrimSpace(input.WizardAnswers[idx].Justification)
	}
	if input.SessionID == "" {
		input.SessionID = firstNonEmptyMCPV0(input.RequestID, input.CorrelationID)
	}
	if input.RequestID == "" {
		input.RequestID = firstNonEmptyMCPV0(input.SessionID, input.CorrelationID, generatedMCPNuevaAppRequestIDV0())
	}
	if input.CorrelationID == "" {
		input.CorrelationID = input.RequestID
	}
	return input
}

func NewMCPNuevaAppWizardResultFromJSONV0(
	input MCPNuevaAppWizardToolInputV0,
	raw []byte,
) (MCPNuevaAppWizardToolResultV0, error) {
	input = NormalizeMCPNuevaAppWizardInputV0(input)
	var response struct {
		SchemaVersion   string                       `json:"schema_version"`
		AssistantStatus string                       `json:"assistant_status,omitempty"`
		Warnings        []MCPNuevaAppWizardWarningV0 `json:"warnings,omitempty"`
		Turn            json.RawMessage              `json:"turn,omitempty"`
		Session         json.RawMessage              `json:"session,omitempty"`
		Wizard          json.RawMessage              `json:"wizard,omitempty"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return MCPNuevaAppWizardToolResultV0{}, err
	}
	if len(response.Wizard) == 0 || string(response.Wizard) == "null" {
		return NewMCPNuevaAppWizardErrorResultV0(input, "nueva_app_wizard_response_incompleta", "wizard"), nil
	}
	return NormalizeMCPNuevaAppWizardResultV0(input, MCPNuevaAppWizardToolResultV0{
		Estado:        MCPNuevaAppWizardEstadoOKV0,
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		SchemaVersion: strings.TrimSpace(response.SchemaVersion),
		Assistant:     strings.TrimSpace(response.AssistantStatus),
		Warnings:      response.Warnings,
		Turn:          response.Turn,
		Session:       response.Session,
		Wizard:        response.Wizard,
	}), nil
}

func NormalizeMCPNuevaAppWizardResultV0(
	input MCPNuevaAppWizardToolInputV0,
	result MCPNuevaAppWizardToolResultV0,
) MCPNuevaAppWizardToolResultV0 {
	input = NormalizeMCPNuevaAppWizardInputV0(input)
	if result.Estado == "" {
		result.Estado = MCPNuevaAppWizardEstadoOKV0
	}
	result.RequestID = firstNonEmptyMCPV0(result.RequestID, input.RequestID)
	result.CorrelationID = firstNonEmptyMCPV0(result.CorrelationID, input.CorrelationID, result.RequestID)
	result.SchemaVersion = strings.TrimSpace(result.SchemaVersion)
	result.Assistant = strings.TrimSpace(result.Assistant)
	if result.Warnings == nil {
		result.Warnings = []MCPNuevaAppWizardWarningV0{}
	}
	if result.Errores == nil {
		result.Errores = []MCPValidationIssueV0{}
	}
	return result
}

func NewMCPNuevaAppWizardErrorResultV0(
	input MCPNuevaAppWizardToolInputV0,
	code string,
	field string,
) MCPNuevaAppWizardToolResultV0 {
	input = NormalizeMCPNuevaAppWizardInputV0(input)
	return MCPNuevaAppWizardToolResultV0{
		Estado:        MCPNuevaAppWizardEstadoErrorV0,
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		Warnings:      []MCPNuevaAppWizardWarningV0{},
		Errores: []MCPValidationIssueV0{{
			Code:    strings.TrimSpace(code),
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(code),
		}},
	}
}
