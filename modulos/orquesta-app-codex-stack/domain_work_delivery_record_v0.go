package orquestaappcodexstack

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func domainWorkRejectedSubmissionRecordV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
	result orquestamcp.MCPDomainWorkToolResultV0,
	occurredAt string,
) DomainWorkArtifactSubmissionRecordV0 {
	record := domainWorkSubmissionRecordFromSubmissionV0(run, task, observation, submission, occurredAt)
	record.Status = DomainWorkArtifactSubmissionStatusRejectedV0
	record.EvidenceRefs = submission.EvidenceRefs
	record.IssueRefs = domainWorkSubmitIssueRefsV0(result)
	return record
}

func domainWorkRejectedSubmissionRecordWithIssueRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
	issueRefs []string,
	occurredAt string,
) DomainWorkArtifactSubmissionRecordV0 {
	record := domainWorkSubmissionRecordFromSubmissionV0(run, task, observation, submission, occurredAt)
	record.Status = DomainWorkArtifactSubmissionStatusRejectedV0
	record.EvidenceRefs = submission.EvidenceRefs
	record.IssueRefs = compactStringsV0(append([]string{
		"domain-work-submit-artifact-rejected",
	}, issueRefs...))
	return record
}

func domainWorkClaimedSubmissionRecordV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
	occurredAt string,
) DomainWorkArtifactSubmissionRecordV0 {
	record := domainWorkSubmissionRecordFromSubmissionV0(run, task, observation, submission, occurredAt)
	record.Status = DomainWorkArtifactSubmissionStatusClaimedV0
	record.EvidenceRefs = compactStringsV0(append([]string{
		"domain-work-submit-claim",
	}, submission.EvidenceRefs...))
	return record
}

func domainWorkSubmittingSubmissionRecordV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
	occurredAt string,
) DomainWorkArtifactSubmissionRecordV0 {
	record := domainWorkSubmissionRecordFromSubmissionV0(run, task, observation, submission, occurredAt)
	record.Status = DomainWorkArtifactSubmissionStatusSubmittingV0
	record.EvidenceRefs = compactStringsV0(append([]string{
		"domain-work-submit-submitting",
	}, submission.EvidenceRefs...))
	return record
}

func domainWorkSubmittedSubmissionRecordV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
	result orquestamcp.MCPDomainWorkToolResultV0,
	occurredAt string,
) DomainWorkArtifactSubmissionRecordV0 {
	record := domainWorkSubmissionRecordFromSubmissionV0(run, task, observation, submission, occurredAt)
	record.Status = DomainWorkArtifactSubmissionStatusSubmittedV0
	if result.Receipt != nil {
		record.ReceiptRef = result.Receipt.ReceiptRef
		record.EvidenceRefs = compactStringsV0(append(
			append([]string{"domain-work-submit-receipt-recorded"}, submission.EvidenceRefs...),
			result.Receipt.EvidenceRefs...,
		))
	}
	return record
}

func domainWorkAcceptedSubmissionRecordV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
	result orquestamcp.MCPDomainWorkToolResultV0,
	occurredAt string,
) DomainWorkArtifactSubmissionRecordV0 {
	record := domainWorkSubmissionRecordFromSubmissionV0(run, task, observation, submission, occurredAt)
	record.Status = DomainWorkArtifactSubmissionStatusAcceptedV0
	record.ReceiptRef = result.Receipt.ReceiptRef
	record.EvidenceRefs = compactStringsV0(append(
		append(domainWorkAcceptedSubmissionEvidenceRefsV0(submission), submission.EvidenceRefs...),
		result.Receipt.EvidenceRefs...,
	))
	return record
}

func domainWorkSubmissionRecordFromSubmissionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
	occurredAt string,
) DomainWorkArtifactSubmissionRecordV0 {
	return DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: submission.IdempotencyKey,
		RunRef:         run.RunID,
		TaskRef:        task.TaskID,
		DeliveryRef:    observation.DeliveryRef,
		CorrelationID:  submission.CorrelationID,
		DomainRef:      submission.DomainRef,
		JobRef:         submission.JobRef,
		ArtifactRef:    submission.ArtifactRef,
		ArtifactType:   submission.ArtifactType,
		Summary:        submission.Summary,
		PayloadFields:  submission.PayloadFields,
		PayloadRefs:    submission.PayloadRefs,
		ExternalRefs:   submission.ExternalRefs,
		CompleteJob:    submission.CompleteJob,
		RecordedAt:     occurredAt,
	}
}

func domainWorkAcceptedSubmissionEvidenceRefsV0(
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
) []string {
	return []string{
		"domain-work-domain-" + safeDomainWorkEvidenceRefV0(submission.DomainRef),
		"domain-work-job-" + safeDomainWorkEvidenceRefV0(submission.JobRef),
		"domain-work-artifact-" + safeDomainWorkEvidenceRefV0(submission.ArtifactType),
		"domain-work-idempotency-" + safeDomainWorkEvidenceRefV0(submission.IdempotencyKey),
	}
}

func safeDomainWorkEvidenceRefV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "empty"
	}
	return codexStackOperationalClosureSafeRefV0(value)
}
