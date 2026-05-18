package orquestadocumentplanexpander

import (
	"context"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	DomainDocumentPlanDerivedJobsCreationSchemaVersionV0  = "domain_document_plan_derived_jobs_creation.v0"
	DomainDocumentPlanDerivedJobsCreationStatusAcceptedV0 = "accepted"
	DomainDocumentPlanDerivedJobsCreationStatusInvalidV0  = "invalid"

	ErrDomainDocumentPlanJobCreatorRequiredV0    = "domain_document_plan_job_creator_required"
	ErrDomainDocumentPlanDerivedJobRejectedV0    = "domain_document_plan_derived_job_rejected"
	ErrDomainDocumentPlanDerivedJobRefRequiredV0 = "domain_document_plan_derived_job_ref_required"
)

type DomainDocumentPlanDerivedJobsCreationPortsV0 struct {
	JobCreator orquestadomainwork.DomainWorkJobCreatorPortV0
}

type DomainDocumentPlanDerivedJobsCreationResultV0 struct {
	SchemaVersion string                                      `json:"schema_version"`
	Status        string                                      `json:"status"`
	CorrelationID string                                      `json:"correlation_id,omitempty"`
	RequestedJobs []orquestadomainwork.DomainWorkJobRequestV0 `json:"requested_jobs,omitempty"`
	CreatedJobs   []orquestadomainwork.DomainWorkJobV0        `json:"created_jobs,omitempty"`
	Issues        []orquestadomainwork.DomainWorkIssueV0      `json:"issues,omitempty"`
}

func CreateDomainDocumentPlanDerivedJobsV0(
	ctx context.Context,
	request DomainDocumentPlanExpansionRequestV0,
	ports DomainDocumentPlanDerivedJobsCreationPortsV0,
) (DomainDocumentPlanDerivedJobsCreationResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result := DomainDocumentPlanDerivedJobsCreationResultV0{
		SchemaVersion: DomainDocumentPlanDerivedJobsCreationSchemaVersionV0,
		Status:        DomainDocumentPlanDerivedJobsCreationStatusInvalidV0,
		CorrelationID: documentPlanCorrelationIDV0(request, orquestadomainwork.NormalizeDomainDocumentPlanV0(request.Plan)),
	}
	if ports.JobCreator == nil {
		result.Issues = []orquestadomainwork.DomainWorkIssueV0{{
			Code:  ErrDomainDocumentPlanJobCreatorRequiredV0,
			Field: "job_creator",
		}}
		return result, nil
	}
	expansion := ExpandDomainDocumentPlanV0(request)
	if len(expansion.Issues) > 0 {
		result.Issues = expansion.Issues
		return result, nil
	}
	result.RequestedJobs = append([]orquestadomainwork.DomainWorkJobRequestV0(nil), expansion.Jobs...)
	result.CreatedJobs = make([]orquestadomainwork.DomainWorkJobV0, 0, len(expansion.Jobs))
	for _, jobRequest := range expansion.Jobs {
		job, err := ports.JobCreator.CreateDomainWorkJobV0(ctx, jobRequest)
		if err != nil {
			return result, err
		}
		if issues := validateCreatedDomainDocumentPlanJobV0(job); len(issues) > 0 {
			result.Issues = issues
			return result, nil
		}
		result.CreatedJobs = append(result.CreatedJobs, job)
	}
	result.Status = DomainDocumentPlanDerivedJobsCreationStatusAcceptedV0
	return result, nil
}

func validateCreatedDomainDocumentPlanJobV0(
	job orquestadomainwork.DomainWorkJobV0,
) []orquestadomainwork.DomainWorkIssueV0 {
	if job.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 {
		issues := append([]orquestadomainwork.DomainWorkIssueV0(nil), job.Issues...)
		issues = append(issues, orquestadomainwork.DomainWorkIssueV0{
			Code:  ErrDomainDocumentPlanDerivedJobRejectedV0,
			Field: "created_jobs.status",
		})
		return issues
	}
	if job.JobRef == "" {
		return []orquestadomainwork.DomainWorkIssueV0{{
			Code:  ErrDomainDocumentPlanDerivedJobRefRequiredV0,
			Field: "created_jobs.job_ref",
		}}
	}
	return nil
}
