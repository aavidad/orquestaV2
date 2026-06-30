package orquestamcp

import (
	"context"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

type MCPDomainWorkToolExecutorV0 struct {
	JobCreator        orquestadomainwork.DomainWorkJobCreatorPortV0
	ArtifactSubmitter orquestadomainwork.DomainWorkArtifactSubmitterPortV0
	JobRecordSource   orquestadomainwork.DomainWorkJobRecordSourcePortV0
	CapabilitySource  orquestadomainwork.DomainWorkExternalCapabilitySourcePortV0
}

func NewMCPDomainWorkToolExecutorV0(
	jobCreator orquestadomainwork.DomainWorkJobCreatorPortV0,
	artifactSubmitter orquestadomainwork.DomainWorkArtifactSubmitterPortV0,
) MCPDomainWorkToolExecutorV0 {
	source, _ := jobCreator.(orquestadomainwork.DomainWorkJobRecordSourcePortV0)
	capabilitySource, _ := jobCreator.(orquestadomainwork.DomainWorkExternalCapabilitySourcePortV0)
	if capabilitySource == nil {
		capabilitySource, _ = artifactSubmitter.(orquestadomainwork.DomainWorkExternalCapabilitySourcePortV0)
	}
	return MCPDomainWorkToolExecutorV0{
		JobCreator:        jobCreator,
		ArtifactSubmitter: artifactSubmitter,
		JobRecordSource:   source,
		CapabilitySource:  capabilitySource,
	}
}

func (executor MCPDomainWorkToolExecutorV0) ListDomainWorkJobRecordsV0(
	ctx context.Context,
	filter orquestadomainwork.DomainWorkJobRecordFilterV0,
) ([]orquestadomainwork.DomainWorkJobRecordV0, error) {
	if executor.JobRecordSource == nil {
		return nil, nil
	}
	return executor.JobRecordSource.ListDomainWorkJobRecordsV0(ctx, filter)
}

func (executor MCPDomainWorkToolExecutorV0) Execute(
	ctx context.Context,
	input MCPDomainWorkToolInputV0,
) (MCPDomainWorkToolResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	switch strings.ToLower(strings.TrimSpace(input.Action)) {
	case MCPDomainWorkActionCreateJobV0:
		return executor.executeCreateJobV0(ctx, input)
	case MCPDomainWorkActionSubmitArtifactV0:
		return executor.executeSubmitArtifactV0(ctx, input)
	case MCPDomainWorkActionEvaluateCapabilitiesV0:
		return executor.executeEvaluateExternalCapabilitiesV0(ctx, input)
	default:
		return newMCPDomainWorkErrorV0(
			input,
			MCPDomainWorkActionUnsupportedV0,
			"action",
			"action debe ser create_job, submit_artifact o evaluate_external_capabilities",
		), nil
	}
}

func (executor MCPDomainWorkToolExecutorV0) executeCreateJobV0(
	ctx context.Context,
	input MCPDomainWorkToolInputV0,
) (MCPDomainWorkToolResultV0, error) {
	if executor.JobCreator == nil {
		return newMCPDomainWorkErrorV0(
			input,
			MCPDomainWorkCreatorUnavailableV0,
			"job_creator",
			"job_creator requerido",
		), nil
	}
	job, err := executor.JobCreator.CreateDomainWorkJobV0(
		ctx,
		domainWorkJobRequestFromMCPV0(input),
	)
	if err != nil {
		return newMCPDomainWorkErrorV0(
			input,
			publicMCPDomainWorkPortErrorCodeV0(err, MCPDomainWorkPortUnavailableV0),
			"job_creator",
			"domain work creator no disponible",
		), nil
	}
	return newMCPDomainWorkJobResultV0(input, job), nil
}

func (executor MCPDomainWorkToolExecutorV0) executeSubmitArtifactV0(
	ctx context.Context,
	input MCPDomainWorkToolInputV0,
) (MCPDomainWorkToolResultV0, error) {
	if executor.ArtifactSubmitter == nil {
		return newMCPDomainWorkErrorV0(
			input,
			MCPDomainWorkSubmitterUnavailableV0,
			"artifact_submitter",
			"artifact_submitter requerido",
		), nil
	}
	receipt, err := executor.ArtifactSubmitter.SubmitDomainWorkArtifactV0(
		ctx,
		domainWorkArtifactSubmissionFromMCPV0(input),
	)
	if err != nil {
		return newMCPDomainWorkErrorV0(
			input,
			publicMCPDomainWorkPortErrorCodeV0(err, MCPDomainWorkPortUnavailableV0),
			"artifact_submitter",
			"domain work submitter no disponible",
		), nil
	}
	return newMCPDomainWorkReceiptResultV0(input, receipt), nil
}

func (executor MCPDomainWorkToolExecutorV0) executeEvaluateExternalCapabilitiesV0(
	ctx context.Context,
	input MCPDomainWorkToolInputV0,
) (MCPDomainWorkToolResultV0, error) {
	request := domainWorkJobRequestFromMCPV0(input)
	capabilities := append([]orquestadomainwork.DomainWorkExternalCapabilityV0(nil), input.ExternalCapabilities...)
	if executor.CapabilitySource != nil {
		sourceCapabilities, err := executor.CapabilitySource.ListDomainWorkExternalCapabilitiesV0(
			ctx,
			orquestadomainwork.BuildDomainWorkExternalCapabilityQueryV0(request),
		)
		if err != nil {
			return newMCPDomainWorkErrorV0(
				input,
				publicMCPDomainWorkPortErrorCodeV0(err, MCPDomainWorkPortUnavailableV0),
				"external_capabilities",
				"domain work external capability source no disponible",
			), nil
		}
		capabilities = append(capabilities, sourceCapabilities...)
	}
	evaluation := orquestadomainwork.EvaluateDomainWorkExternalCapabilitiesV0(request, capabilities)
	return newMCPDomainWorkExternalCapabilityResultV0(input, evaluation), nil
}

type mcpPublicCodeErrorV0 interface {
	PublicCodeV0() string
}

func publicMCPDomainWorkPortErrorCodeV0(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	if publicErr, ok := err.(mcpPublicCodeErrorV0); ok {
		if code := strings.TrimSpace(publicErr.PublicCodeV0()); code != "" {
			return code
		}
	}
	return fallback
}
