package orquestamcp

import (
	"context"
	"strings"

	orquestadocumentplanexpander "orquesta/modulos/orquesta-document-plan-expander"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	MCPDocumentPlanExpandToolNameV0    = "orquesta.document_plan.expand.v0"
	MCPDocumentPlanExpandToolVersionV0 = "v0"

	MCPDocumentPlanExpandActionPreviewV0    = "preview"
	MCPDocumentPlanExpandActionCreateJobsV0 = "create_jobs"

	MCPDocumentPlanExpandEstadoOKV0    = "ok"
	MCPDocumentPlanExpandEstadoErrorV0 = "error"

	MCPDocumentPlanExpandMaxDerivedJobsV0 = 40

	MCPDocumentPlanExpandCreatorUnavailableV0 = "document_plan_job_creator_no_disponible"
	MCPDocumentPlanExpandActionUnsupportedV0  = "document_plan_expand_action_no_soportada"
	MCPDocumentPlanExpandJobLimitExceededV0   = "document_plan_derived_jobs_limit_exceeded"
)

type MCPDocumentPlanExpandToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	Invariantes []string `json:"invariantes"`
}

type MCPDocumentPlanExpandToolInputV0 struct {
	RequestID        string                                                            `json:"request_id,omitempty"`
	CorrelationID    string                                                            `json:"correlation_id,omitempty"`
	Action           string                                                            `json:"action"`
	ExpansionRequest orquestadocumentplanexpander.DomainDocumentPlanExpansionRequestV0 `json:"expansion_request"`
}

type MCPDocumentPlanExpandToolResultV0 struct {
	Estado        string                                                                      `json:"estado"`
	RequestID     string                                                                      `json:"request_id,omitempty"`
	CorrelationID string                                                                      `json:"correlation_id,omitempty"`
	Action        string                                                                      `json:"action,omitempty"`
	Expansion     *orquestadocumentplanexpander.DomainDocumentPlanExpansionResultV0           `json:"expansion,omitempty"`
	Creation      *orquestadocumentplanexpander.DomainDocumentPlanDerivedJobsCreationResultV0 `json:"creation,omitempty"`
	Errores       []MCPValidationIssueV0                                                      `json:"errores_publicos,omitempty"`
}

type MCPDocumentPlanExpandToolExecutorV0 struct {
	JobCreator orquestadomainwork.DomainWorkJobCreatorPortV0
}

func NewMCPDocumentPlanExpandToolExecutorV0(
	jobCreator orquestadomainwork.DomainWorkJobCreatorPortV0,
) MCPDocumentPlanExpandToolExecutorV0 {
	return MCPDocumentPlanExpandToolExecutorV0{JobCreator: jobCreator}
}

func MCPDocumentPlanExpandDescriptorV0() MCPDocumentPlanExpandToolDescriptorV0 {
	return MCPDocumentPlanExpandToolDescriptorV0{
		Name:        MCPDocumentPlanExpandToolNameV0,
		Version:     MCPDocumentPlanExpandToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,action:preview|create_jobs,expansion_request:DomainDocumentPlanExpansionRequestV0}",
		Output:      "ok:{action,expansion?{jobs[]},creation?{requested_jobs[],created_jobs[]}}|error:{errores_publicos,expansion?,creation?}",
		Invariantes: []string{
			"preview delega en ExpandDomainDocumentPlanV0",
			"create_jobs delega en CreateDomainDocumentPlanDerivedJobsV0 por DomainWorkJobCreatorPortV0 inyectado",
			"create_jobs detecta job_creator ausente antes de validar el plan",
			"el plan y la respuesta no superan 40 jobs derivados",
			"sin registro, wiring, OPES, DB, runtime, filesystem ni proveedor",
		},
	}
}

func (executor MCPDocumentPlanExpandToolExecutorV0) Execute(
	ctx context.Context,
	input MCPDocumentPlanExpandToolInputV0,
) (MCPDocumentPlanExpandToolResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	action := strings.ToLower(strings.TrimSpace(input.Action))
	switch action {
	case MCPDocumentPlanExpandActionPreviewV0:
		return executor.previewV0(input), nil
	case MCPDocumentPlanExpandActionCreateJobsV0:
		// This must precede plan expansion: an unavailable effect port is a
		// composition error, independent from whether the supplied plan is valid.
		if executor.JobCreator == nil {
			return newMCPDocumentPlanExpandErrorV0(
				input,
				MCPDocumentPlanExpandCreatorUnavailableV0,
				"job_creator",
				"job_creator requerido",
			), nil
		}
		return executor.createJobsV0(ctx, input), nil
	default:
		return newMCPDocumentPlanExpandErrorV0(
			input,
			MCPDocumentPlanExpandActionUnsupportedV0,
			"action",
			"action debe ser preview o create_jobs",
		), nil
	}
}

func (executor MCPDocumentPlanExpandToolExecutorV0) previewV0(
	input MCPDocumentPlanExpandToolInputV0,
) MCPDocumentPlanExpandToolResultV0 {
	expansion := orquestadocumentplanexpander.ExpandDomainDocumentPlanV0(documentPlanExpansionRequestFromMCPV0(input))
	if limitIssue := documentPlanDerivedJobsLimitIssueMCPV0(expansion.Jobs); limitIssue != nil {
		return newMCPDocumentPlanExpandErrorV0(input, limitIssue.Code, limitIssue.Field, limitIssue.Message)
	}
	result := MCPDocumentPlanExpandToolResultV0{
		Estado:        MCPDocumentPlanExpandEstadoOKV0,
		RequestID:     documentPlanRequestIDMCPV0(input),
		CorrelationID: documentPlanCorrelationIDMCPV0(input),
		Action:        MCPDocumentPlanExpandActionPreviewV0,
		Expansion:     &expansion,
		Errores:       domainWorkIssuesMCPV0(expansion.Issues),
	}
	if len(expansion.Issues) > 0 {
		result.Estado = MCPDocumentPlanExpandEstadoErrorV0
	}
	return result
}

func (executor MCPDocumentPlanExpandToolExecutorV0) createJobsV0(
	ctx context.Context,
	input MCPDocumentPlanExpandToolInputV0,
) MCPDocumentPlanExpandToolResultV0 {
	request := documentPlanExpansionRequestFromMCPV0(input)
	preview := orquestadocumentplanexpander.ExpandDomainDocumentPlanV0(request)
	if limitIssue := documentPlanDerivedJobsLimitIssueMCPV0(preview.Jobs); limitIssue != nil {
		return newMCPDocumentPlanExpandErrorV0(input, limitIssue.Code, limitIssue.Field, limitIssue.Message)
	}
	creation, err := orquestadocumentplanexpander.CreateDomainDocumentPlanDerivedJobsV0(
		ctx,
		request,
		orquestadocumentplanexpander.DomainDocumentPlanDerivedJobsCreationPortsV0{JobCreator: executor.JobCreator},
	)
	result := MCPDocumentPlanExpandToolResultV0{
		Estado:        MCPDocumentPlanExpandEstadoOKV0,
		RequestID:     documentPlanRequestIDMCPV0(input),
		CorrelationID: documentPlanCorrelationIDMCPV0(input),
		Action:        MCPDocumentPlanExpandActionCreateJobsV0,
		Creation:      &creation,
		Errores:       domainWorkIssuesMCPV0(creation.Issues),
	}
	if err != nil {
		result.Estado = MCPDocumentPlanExpandEstadoErrorV0
		result.Errores = append(result.Errores, MCPValidationIssueV0{
			Code:    publicMCPDomainWorkPortErrorCodeV0(err, MCPDocumentPlanExpandCreatorUnavailableV0),
			Field:   "job_creator",
			Message: "document plan job creator no disponible",
		})
	} else if creation.Status != orquestadocumentplanexpander.DomainDocumentPlanDerivedJobsCreationStatusAcceptedV0 {
		result.Estado = MCPDocumentPlanExpandEstadoErrorV0
	}
	return result
}

func documentPlanExpansionRequestFromMCPV0(
	input MCPDocumentPlanExpandToolInputV0,
) orquestadocumentplanexpander.DomainDocumentPlanExpansionRequestV0 {
	request := input.ExpansionRequest
	if strings.TrimSpace(request.CorrelationID) == "" {
		request.CorrelationID = strings.TrimSpace(input.CorrelationID)
	}
	return request
}

func documentPlanRequestIDMCPV0(input MCPDocumentPlanExpandToolInputV0) string {
	return firstNonEmptyMCPV0(input.RequestID, input.ExpansionRequest.Plan.PlanRef)
}

func documentPlanCorrelationIDMCPV0(input MCPDocumentPlanExpandToolInputV0) string {
	return firstNonEmptyMCPV0(input.ExpansionRequest.CorrelationID, input.CorrelationID, input.RequestID)
}

func documentPlanDerivedJobsLimitIssueMCPV0(
	jobs []orquestadomainwork.DomainWorkJobRequestV0,
) *MCPValidationIssueV0 {
	if len(jobs) <= MCPDocumentPlanExpandMaxDerivedJobsV0 {
		return nil
	}
	return &MCPValidationIssueV0{
		Code:    MCPDocumentPlanExpandJobLimitExceededV0,
		Field:   "expansion_request.plan",
		Message: "document plan no puede derivar mas de 40 jobs",
	}
}

func newMCPDocumentPlanExpandErrorV0(
	input MCPDocumentPlanExpandToolInputV0,
	code string,
	field string,
	message string,
) MCPDocumentPlanExpandToolResultV0 {
	return MCPDocumentPlanExpandToolResultV0{
		Estado:        MCPDocumentPlanExpandEstadoErrorV0,
		RequestID:     documentPlanRequestIDMCPV0(input),
		CorrelationID: documentPlanCorrelationIDMCPV0(input),
		Action:        strings.ToLower(strings.TrimSpace(input.Action)),
		Errores: []MCPValidationIssueV0{{
			Code:    code,
			Field:   field,
			Message: message,
		}},
	}
}
