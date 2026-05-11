package orquestamcp

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
)

const (
	MCPRequestAppChangeToolNameV0    = "orquesta.apps.request_change.v0"
	MCPRequestAppChangeToolVersionV0 = "v0"
	MCPRequestAppChangeResourceURIV0 = "orquesta://contracts/request-app-change/v0"
	MCPRequestAppChangeEstadoOKV0    = "ok"
	MCPRequestAppChangeEstadoErrorV0 = "error"
)

type MCPRequestAppChangeToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPRequestAppChangeToolInputV0 struct {
	RequestID        string                               `json:"request_id,omitempty"`
	CorrelationID    string                               `json:"correlation_id,omitempty"`
	Respuesta        string                               `json:"respuesta,omitempty"`
	AppChangeRequest orquestaappchange.AppChangeRequestV0 `json:"app_change_request"`
}

type MCPRequestAppChangeToolResultV0 struct {
	Estado              string                `json:"estado"`
	RequestID           string                `json:"request_id,omitempty"`
	CorrelationID       string                `json:"correlation_id,omitempty"`
	RunRef              string                `json:"run_ref,omitempty"`
	AppRef              string                `json:"app_ref,omitempty"`
	ChangeRef           string                `json:"change_ref,omitempty"`
	DirectorQuestionRef string                `json:"director_question_ref,omitempty"`
	EvidenceRefs        []string              `json:"evidence_refs,omitempty"`
	Errores             []MCPAppChangeIssueV0 `json:"errores_publicos,omitempty"`
}

type MCPAppChangeIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func MCPRequestAppChangeDescriptorV0() MCPRequestAppChangeToolDescriptorV0 {
	return MCPRequestAppChangeToolDescriptorV0{
		Name:        MCPRequestAppChangeToolNameV0,
		Version:     MCPRequestAppChangeToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,app_change_request:AppChangeRequestV0}",
		Output:      "ok:{run_ref,change_ref,director_question_ref}|error:{errores_publicos}",
		ResourceURI: MCPRequestAppChangeResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"no elige proveedor modelo DB runtime ni filesystem",
			"delegacion en orquesta-app-change",
		},
	}
}

func ToAppChangeRequestFromMCPV0(
	input MCPRequestAppChangeToolInputV0,
) orquestaappchange.AppChangeRequestV0 {
	request := input.AppChangeRequest
	if strings.TrimSpace(request.RequestID) == "" {
		request.RequestID = strings.TrimSpace(input.RequestID)
	}
	if strings.TrimSpace(request.CorrelationID) == "" {
		request.CorrelationID = strings.TrimSpace(input.CorrelationID)
	}
	return request
}

func NewMCPRequestAppChangeResultV0(
	result orquestaappchange.AppChangeResultV0,
) MCPRequestAppChangeToolResultV0 {
	estado := MCPRequestAppChangeEstadoOKV0
	if result.Status != orquestaappchange.AppChangeStatusAcceptedV0 {
		estado = MCPRequestAppChangeEstadoErrorV0
	}
	return MCPRequestAppChangeToolResultV0{
		Estado:              estado,
		RequestID:           strings.TrimSpace(result.RequestID),
		CorrelationID:       strings.TrimSpace(result.CorrelationID),
		RunRef:              strings.TrimSpace(result.RunRef),
		AppRef:              strings.TrimSpace(result.AppRef),
		ChangeRef:           strings.TrimSpace(result.ChangeRef),
		DirectorQuestionRef: strings.TrimSpace(result.DirectorQuestionRef),
		EvidenceRefs:        compactStringsMCPV0(result.EvidenceRefs),
		Errores:             appChangeIssuesMCPV0(result.Issues),
	}
}

func appChangeIssuesMCPV0(issues []orquestaappchange.AppChangeIssueV0) []MCPAppChangeIssueV0 {
	out := make([]MCPAppChangeIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, MCPAppChangeIssueV0{
			Code:  strings.TrimSpace(issue.Code),
			Field: strings.TrimSpace(issue.Field),
		})
	}
	if out == nil {
		return nil
	}
	return out
}
