package orquestamcp

import (
	"context"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	MCPDomainWorkToolNameV0                  = "orquesta.domain_work.v0"
	MCPDomainWorkToolVersionV0               = "v0"
	MCPDomainWorkResourceURIV0               = "orquesta://contracts/domain-work/v0"
	MCPDomainWorkEstadoOKV0                  = "ok"
	MCPDomainWorkEstadoErrorV0               = "error"
	MCPDomainWorkActionCreateJobV0           = "create_job"
	MCPDomainWorkActionSubmitArtifactV0      = "submit_artifact"
	MCPDomainWorkDefaultErrorMessageV0       = "domain_work_error"
	MCPDomainWorkCreatorUnavailableV0        = "domain_work_creator_no_disponible"
	MCPDomainWorkSubmitterUnavailableV0      = "domain_work_submitter_no_disponible"
	MCPDomainWorkActionUnsupportedV0         = "domain_work_action_no_soportada"
	MCPDomainWorkPortUnavailableV0           = "domain_work_port_no_disponible"
	MCPDomainWorkHTTPPathV0                  = "/api/v0/domain-work"
	MCPDomainWorkHTTPErrorCodeV0             = "domain_work_http_error"
	MCPDomainWorkHTTPNotConfiguredCodeV0     = "domain_work_no_configurado"
	MCPDomainWorkHTTPExecutorErrorCodeV0     = "domain_work_error"
	MCPDomainWorkHTTPInvalidBodyCodeV0       = "request_body_invalido"
	MCPDomainWorkHTTPUnsupportedPathCodeV0   = "ruta_no_soportada"
	MCPDomainWorkHTTPUnsupportedMethodCodeV0 = "metodo_no_permitido"
)

type MCPDomainWorkToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPDomainWorkToolInputV0 struct {
	RequestID          string                                            `json:"request_id,omitempty"`
	CorrelationID      string                                            `json:"correlation_id,omitempty"`
	Action             string                                            `json:"action"`
	JobRequest         orquestadomainwork.DomainWorkJobRequestV0         `json:"job_request,omitempty"`
	ArtifactSubmission orquestadomainwork.DomainWorkArtifactSubmissionV0 `json:"artifact_submission,omitempty"`
}

type MCPDomainWorkToolResultV0 struct {
	Estado        string                                          `json:"estado"`
	RequestID     string                                          `json:"request_id,omitempty"`
	CorrelationID string                                          `json:"correlation_id,omitempty"`
	Action        string                                          `json:"action,omitempty"`
	Job           *orquestadomainwork.DomainWorkJobV0             `json:"job,omitempty"`
	Receipt       *orquestadomainwork.DomainWorkArtifactReceiptV0 `json:"receipt,omitempty"`
	Errores       []MCPValidationIssueV0                          `json:"errores_publicos,omitempty"`
}

type MCPDomainWorkExecutorPortV0 interface {
	Execute(context.Context, MCPDomainWorkToolInputV0) (MCPDomainWorkToolResultV0, error)
}

func MCPDomainWorkDescriptorV0() MCPDomainWorkToolDescriptorV0 {
	return MCPDomainWorkToolDescriptorV0{
		Name:        MCPDomainWorkToolNameV0,
		Version:     MCPDomainWorkToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,action:create_job|submit_artifact,job_request?:DomainWorkJobRequestV0,artifact_submission?:DomainWorkArtifactSubmissionV0}",
		Output:      "ok:{action,job?|receipt?}|error:{errores_publicos}",
		ResourceURI: MCPDomainWorkResourceURIV0,
		Invariantes: []string{
			"adaptador inbound fino",
			"create_job delega solo en DomainWorkJobCreatorPortV0 inyectado",
			"submit_artifact delega solo en DomainWorkArtifactSubmitterPortV0 inyectado",
			"sin OPES, conector REST, DB, runtime, filesystem ni proveedor",
		},
	}
}

func domainWorkJobRequestFromMCPV0(
	input MCPDomainWorkToolInputV0,
) orquestadomainwork.DomainWorkJobRequestV0 {
	request := input.JobRequest
	if strings.TrimSpace(request.RequestID) == "" {
		request.RequestID = strings.TrimSpace(input.RequestID)
	}
	if strings.TrimSpace(request.CorrelationID) == "" {
		request.CorrelationID = strings.TrimSpace(input.CorrelationID)
	}
	return request
}

func domainWorkArtifactSubmissionFromMCPV0(
	input MCPDomainWorkToolInputV0,
) orquestadomainwork.DomainWorkArtifactSubmissionV0 {
	submission := input.ArtifactSubmission
	if strings.TrimSpace(submission.RequestID) == "" {
		submission.RequestID = strings.TrimSpace(input.RequestID)
	}
	if strings.TrimSpace(submission.CorrelationID) == "" {
		submission.CorrelationID = strings.TrimSpace(input.CorrelationID)
	}
	return submission
}

func newMCPDomainWorkJobResultV0(
	input MCPDomainWorkToolInputV0,
	job orquestadomainwork.DomainWorkJobV0,
) MCPDomainWorkToolResultV0 {
	estado := MCPDomainWorkEstadoOKV0
	if job.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 {
		estado = MCPDomainWorkEstadoErrorV0
	}
	return MCPDomainWorkToolResultV0{
		Estado:        estado,
		RequestID:     firstNonEmptyMCPV0(input.RequestID, job.JobRef),
		CorrelationID: firstNonEmptyMCPV0(job.CorrelationID, input.CorrelationID, input.RequestID),
		Action:        MCPDomainWorkActionCreateJobV0,
		Job:           &job,
		Errores:       domainWorkIssuesMCPV0(job.Issues),
	}
}

func newMCPDomainWorkReceiptResultV0(
	input MCPDomainWorkToolInputV0,
	receipt orquestadomainwork.DomainWorkArtifactReceiptV0,
) MCPDomainWorkToolResultV0 {
	estado := MCPDomainWorkEstadoOKV0
	if receipt.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 {
		estado = MCPDomainWorkEstadoErrorV0
	}
	return MCPDomainWorkToolResultV0{
		Estado:        estado,
		RequestID:     firstNonEmptyMCPV0(input.RequestID, receipt.ReceiptRef, receipt.ArtifactRef),
		CorrelationID: firstNonEmptyMCPV0(receipt.CorrelationID, input.CorrelationID, input.RequestID),
		Action:        MCPDomainWorkActionSubmitArtifactV0,
		Receipt:       &receipt,
		Errores:       domainWorkIssuesMCPV0(receipt.Issues),
	}
}

func newMCPDomainWorkErrorV0(
	input MCPDomainWorkToolInputV0,
	code string,
	field string,
	message string,
) MCPDomainWorkToolResultV0 {
	return MCPDomainWorkToolResultV0{
		Estado:        MCPDomainWorkEstadoErrorV0,
		RequestID:     strings.TrimSpace(input.RequestID),
		CorrelationID: firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		Action:        strings.ToLower(strings.TrimSpace(input.Action)),
		Errores: []MCPValidationIssueV0{{
			Code:    strings.TrimSpace(code),
			Field:   strings.TrimSpace(field),
			Message: firstNonEmptyMCPV0(message, MCPDomainWorkDefaultErrorMessageV0),
		}},
	}
}

func domainWorkIssuesMCPV0(
	issues []orquestadomainwork.DomainWorkIssueV0,
) []MCPValidationIssueV0 {
	out := make([]MCPValidationIssueV0, 0, len(issues))
	for _, issue := range issues {
		code := strings.TrimSpace(issue.Code)
		if code == "" {
			code = MCPDomainWorkDefaultErrorMessageV0
		}
		out = append(out, MCPValidationIssueV0{
			Code:    code,
			Field:   strings.TrimSpace(issue.Field),
			Message: code,
		})
	}
	if out == nil {
		return []MCPValidationIssueV0{}
	}
	return out
}
