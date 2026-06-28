package orquestaappcodexstack

import (
	"context"
	"fmt"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func (stack StackV0) domainWorkSubmissionAlreadyRecordedV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	task orquestacoreworkflow.WorkflowTaskV0,
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
) (bool, error) {
	reader, ok := stack.DomainDelivery.Ledger.(DomainWorkArtifactSubmissionRecordReaderPortV0)
	if !ok || reader == nil {
		return stack.DomainDelivery.Ledger.HasDomainWorkArtifactSubmissionV0(ctx, submission.IdempotencyKey)
	}
	records, err := reader.ListDomainWorkArtifactSubmissionsV0(ctx, DomainWorkArtifactSubmissionRecordFilterV0{
		IdempotencyKey: submission.IdempotencyKey,
	})
	if err != nil || len(records) == 0 {
		return false, err
	}
	record := normalizeDomainWorkArtifactSubmissionRecordV0(records[0])
	current := domainWorkSubmissionRecordFromSubmissionV0(run, task, observation, submission, "")
	if !domainWorkSubmissionSameCausalSurfaceV0(record, current) {
		return true, fmt.Errorf("domain_work_submit_conflict")
	}
	if record.Status == DomainWorkArtifactSubmissionStatusAcceptedV0 {
		return true, nil
	}
	if domainWorkSubmissionSubmittedWithReceiptV0(record) {
		if err := stack.promoteDomainWorkSubmittedReceiptV0(ctx, record); err != nil {
			return false, fmt.Errorf("domain_work_submit_recovery_required")
		}
		return true, nil
	}
	if domainWorkSubmissionLiveStatusV0(record.Status) {
		return false, fmt.Errorf("domain_work_submit_recovery_required")
	}
	if record.Status == DomainWorkArtifactSubmissionStatusRejectedV0 {
		if domainWorkRejectedSubmissionRecoverableV0(record.IssueRefs) {
			return false, nil
		}
		return true, nil
	}
	return false, nil
}

func domainWorkSubmissionSubmittedWithReceiptV0(record DomainWorkArtifactSubmissionRecordV0) bool {
	return record.Status == DomainWorkArtifactSubmissionStatusSubmittedV0 && record.ReceiptRef != ""
}

func (stack StackV0) promoteDomainWorkSubmittedReceiptV0(
	ctx context.Context,
	record DomainWorkArtifactSubmissionRecordV0,
) error {
	accepted := record
	accepted.Status = DomainWorkArtifactSubmissionStatusAcceptedV0
	accepted.IssueRefs = nil
	accepted.EvidenceRefs = compactStringsV0(append(
		append([]string(nil), record.EvidenceRefs...),
		"domain-work-submit-ledger-recovered",
	))
	return stack.DomainDelivery.Ledger.RecordDomainWorkArtifactSubmissionV0(ctx, accepted)
}

func domainWorkSubmissionLiveStatusV0(status string) bool {
	switch status {
	case DomainWorkArtifactSubmissionStatusClaimedV0,
		DomainWorkArtifactSubmissionStatusSubmittingV0,
		DomainWorkArtifactSubmissionStatusSubmittedV0:
		return true
	default:
		return false
	}
}

func domainWorkRejectedSubmissionRecoverableV0(issueRefs []string) bool {
	for _, issue := range issueRefs {
		switch issue {
		case "domain-work-submit-execute-error",
			"domain_work_port_no_disponible",
			"opes_http_request_failed",
			"opes_http_timeout",
			"opes_http_cancelled",
			"retry_budget_exhausted":
			return true
		case "opes_http_status_400",
			"opes_http_status_429",
			"opes_http_status_502",
			"opes_http_status_503",
			"opes_http_status_504":
			return true
		}
	}
	return false
}
