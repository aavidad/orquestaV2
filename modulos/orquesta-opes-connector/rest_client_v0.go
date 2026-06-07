package orquestaopesconnector

import (
	"context"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

var _ orquestadomainwork.DomainWorkJobCreatorPortV0 = RESTClientV0{}
var _ orquestadomainwork.DomainWorkArtifactSubmitterPortV0 = RESTClientV0{}

func (client RESTClientV0) CreateDomainWorkJobV0(
	ctx context.Context,
	request orquestadomainwork.DomainWorkJobRequestV0,
) (orquestadomainwork.DomainWorkJobV0, error) {
	request = orquestadomainwork.NormalizeDomainWorkJobRequestV0(request)
	if issues := orquestadomainwork.ValidateDomainWorkJobRequestV0(request); len(issues) > 0 {
		return invalidDomainWorkJobV0(request, issues), nil
	}
	if client.baseURL == "" {
		return orquestadomainwork.DomainWorkJobV0{}, connectorErrorV0{code: ErrOPESBaseURLRequiredV0}
	}
	var response opesJobResponseV0
	if err := client.postJSONV0(ctx, DefaultOPESCreateJobPathV0, opesCreateJobPayloadV0(request, client.defaultMaxAttempts), &response); err != nil {
		return orquestadomainwork.DomainWorkJobV0{}, err
	}
	return domainWorkJobFromOPESV0(request, response), nil
}

func (client RESTClientV0) SubmitDomainWorkArtifactV0(
	ctx context.Context,
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
) (orquestadomainwork.DomainWorkArtifactReceiptV0, error) {
	submission = orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(submission)
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) > 0 {
		return invalidDomainWorkArtifactReceiptV0(submission, issues), nil
	}
	if client.baseURL == "" {
		return orquestadomainwork.DomainWorkArtifactReceiptV0{}, connectorErrorV0{code: ErrOPESBaseURLRequiredV0}
	}
	var response opesArtifactResponseV0
	if err := client.postJSONV0(ctx, opesArtifactPathV0(submission.JobRef), opesArtifactPayloadV0(submission), &response); err != nil {
		if code, ok := connectorErrorCodeV0(err); ok && opesHTTPStatusErrorCodeIs4xxV0(code) {
			return invalidDomainWorkArtifactReceiptV0(submission, []orquestadomainwork.DomainWorkIssueV0{{
				Code:  code,
				Field: "artifact_submitter",
			}}), nil
		}
		return orquestadomainwork.DomainWorkArtifactReceiptV0{}, err
	}
	return domainWorkArtifactReceiptFromOPESV0(submission, response), nil
}
