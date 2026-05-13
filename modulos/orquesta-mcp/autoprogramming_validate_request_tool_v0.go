package orquestamcp

import (
	"context"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

const (
	MCPAutoprogrammingValidateRequestToolNameV0    = "orquesta.autoprogramming.validate_request.v0"
	MCPAutoprogrammingValidateRequestToolVersionV0 = "v0"
	MCPAutoprogrammingValidateRequestResourceURIV0 = "orquesta://contracts/autoprogramming-request/v0"
	MCPAutoprogrammingValidateRequestEstadoOKV0    = "ok"
	MCPAutoprogrammingValidateRequestEstadoErrorV0 = "error"
)

type MCPAutoprogrammingValidateRequestToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPAutoprogrammingValidateRequestToolInputV0 struct {
	RequestID              string                                           `json:"request_id,omitempty"`
	CorrelationID          string                                           `json:"correlation_id,omitempty"`
	AutoprogrammingRequest orquestaautoprogramming.AutoprogrammingRequestV0 `json:"autoprogramming_request"`
}

type MCPAutoprogrammingValidateRequestToolResultV0 struct {
	Estado        string                                               `json:"estado"`
	RequestID     string                                               `json:"request_id,omitempty"`
	CorrelationID string                                               `json:"correlation_id,omitempty"`
	Accepted      bool                                                 `json:"accepted"`
	Groups        []orquestaautoprogramming.AutoprogrammingTaskGroupV0 `json:"groups,omitempty"`
	WriteSet      []string                                             `json:"write_set,omitempty"`
	RequiredTests []string                                             `json:"required_tests,omitempty"`
	Errores       []MCPValidationIssueV0                               `json:"errores_publicos,omitempty"`
}

type MCPAutoprogrammingValidateRequestToolExecutorV0 struct{}

func MCPAutoprogrammingValidateRequestDescriptorV0() MCPAutoprogrammingValidateRequestToolDescriptorV0 {
	return MCPAutoprogrammingValidateRequestToolDescriptorV0{
		Name:        MCPAutoprogrammingValidateRequestToolNameV0,
		Version:     MCPAutoprogrammingValidateRequestToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,autoprogramming_request:AutoprogrammingRequestV0}",
		Output:      "ok:{accepted,groups,write_set,required_tests}|error:{accepted:false,errores_publicos}",
		ResourceURI: MCPAutoprogrammingValidateRequestResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"delega la validacion en orquesta-autoprogramming",
			"no ejecuta agentes pruebas VCS comandos ni filesystem",
			"no elige DB runtime ni implementacion de cambio",
		},
	}
}

func (executor MCPAutoprogrammingValidateRequestToolExecutorV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingValidateRequestToolInputV0,
) (MCPAutoprogrammingValidateRequestToolResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request := autoprogrammingRequestFromMCPV0(input)
	result := orquestaautoprogramming.ValidateAutoprogrammingRequestV0(request)
	return newMCPAutoprogrammingValidateRequestResultV0(input, request, result), nil
}

func autoprogrammingRequestFromMCPV0(
	input MCPAutoprogrammingValidateRequestToolInputV0,
) orquestaautoprogramming.AutoprogrammingRequestV0 {
	request := input.AutoprogrammingRequest
	if strings.TrimSpace(request.RequestRef) == "" {
		request.RequestRef = firstNonEmptyMCPV0(input.RequestID, input.CorrelationID)
	}
	return request
}

func newMCPAutoprogrammingValidateRequestResultV0(
	input MCPAutoprogrammingValidateRequestToolInputV0,
	request orquestaautoprogramming.AutoprogrammingRequestV0,
	validation orquestaautoprogramming.AutoprogrammingRequestValidationResultV0,
) MCPAutoprogrammingValidateRequestToolResultV0 {
	estado := MCPAutoprogrammingValidateRequestEstadoOKV0
	if !validation.Accepted {
		estado = MCPAutoprogrammingValidateRequestEstadoErrorV0
	}
	return MCPAutoprogrammingValidateRequestToolResultV0{
		Estado:        estado,
		RequestID:     strings.TrimSpace(request.RequestRef),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID, request.RequestRef),
		Accepted:      validation.Accepted,
		Groups:        append([]orquestaautoprogramming.AutoprogrammingTaskGroupV0(nil), validation.Groups...),
		WriteSet:      compactStringsMCPV0(validation.WriteSet),
		RequiredTests: compactStringsMCPV0(validation.RequiredTests),
		Errores:       autoprogrammingRequestIssuesMCPV0(validation.Issues),
	}
}

func autoprogrammingRequestIssuesMCPV0(
	issues []orquestaautoprogramming.AutoprogrammingRequestIssueV0,
) []MCPValidationIssueV0 {
	out := make([]MCPValidationIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, MCPValidationIssueV0{
			Code:    strings.TrimSpace(issue.Code),
			Field:   strings.TrimSpace(issue.Field),
			Message: strings.TrimSpace(issue.Message),
		})
	}
	if out == nil {
		return []MCPValidationIssueV0{}
	}
	return out
}
