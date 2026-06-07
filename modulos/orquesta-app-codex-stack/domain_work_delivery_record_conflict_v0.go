package orquestaappcodexstack

import (
	"fmt"
	"reflect"
)

func mergeDomainWorkArtifactSubmissionRecordV0(
	existing DomainWorkArtifactSubmissionRecordV0,
	incoming DomainWorkArtifactSubmissionRecordV0,
) (DomainWorkArtifactSubmissionRecordV0, error) {
	existing = normalizeDomainWorkArtifactSubmissionRecordV0(existing)
	incoming = normalizeDomainWorkArtifactSubmissionRecordV0(incoming)
	if existing.IdempotencyKey == "" || incoming.IdempotencyKey == "" ||
		existing.IdempotencyKey != incoming.IdempotencyKey ||
		!domainWorkSubmissionSameCausalSurfaceV0(existing, incoming) {
		return DomainWorkArtifactSubmissionRecordV0{}, fmt.Errorf("domain_work_submit_conflict")
	}
	if reflect.DeepEqual(existing, incoming) {
		return existing, nil
	}
	if domainWorkSubmissionTerminalStatusV0(existing.Status) {
		if domainWorkSubmissionSameTerminalRecordV0(existing, incoming) {
			return existing, nil
		}
		return DomainWorkArtifactSubmissionRecordV0{}, fmt.Errorf("domain_work_submit_conflict")
	}
	if !domainWorkSubmissionStatusTransitionAllowedV0(existing.Status, incoming.Status) {
		return DomainWorkArtifactSubmissionRecordV0{}, fmt.Errorf("domain_work_submit_conflict")
	}
	return incoming, nil
}

func domainWorkSubmissionSameCausalSurfaceV0(
	left DomainWorkArtifactSubmissionRecordV0,
	right DomainWorkArtifactSubmissionRecordV0,
) bool {
	return domainWorkRecordFieldCompatibleV0(left.RunRef, right.RunRef) &&
		domainWorkRecordFieldCompatibleV0(left.TaskRef, right.TaskRef) &&
		domainWorkRecordFieldCompatibleV0(left.DeliveryRef, right.DeliveryRef) &&
		domainWorkRecordFieldCompatibleV0(left.DomainRef, right.DomainRef) &&
		domainWorkRecordFieldCompatibleV0(left.JobRef, right.JobRef) &&
		domainWorkRecordFieldCompatibleV0(left.ArtifactRef, right.ArtifactRef) &&
		domainWorkRecordFieldCompatibleV0(left.ArtifactType, right.ArtifactType)
}

func domainWorkRecordFieldCompatibleV0(left string, right string) bool {
	return left == "" || right == "" || left == right
}

func domainWorkSubmissionTerminalStatusV0(status string) bool {
	return status == DomainWorkArtifactSubmissionStatusAcceptedV0
}

func domainWorkSubmissionSameTerminalRecordV0(
	existing DomainWorkArtifactSubmissionRecordV0,
	incoming DomainWorkArtifactSubmissionRecordV0,
) bool {
	if existing.Status != incoming.Status {
		return false
	}
	if existing.Status == DomainWorkArtifactSubmissionStatusAcceptedV0 {
		return existing.ReceiptRef == incoming.ReceiptRef
	}
	if existing.Status == DomainWorkArtifactSubmissionStatusRejectedV0 {
		return reflect.DeepEqual(existing.IssueRefs, incoming.IssueRefs)
	}
	return false
}

func domainWorkSubmissionStatusTransitionAllowedV0(existing string, incoming string) bool {
	if existing == "" || existing == DomainWorkArtifactSubmissionStatusClaimedV0 {
		return incoming == DomainWorkArtifactSubmissionStatusClaimedV0 ||
			incoming == DomainWorkArtifactSubmissionStatusSubmittingV0 ||
			domainWorkSubmissionTerminalStatusV0(incoming)
	}
	if existing == DomainWorkArtifactSubmissionStatusSubmittingV0 {
		return incoming == DomainWorkArtifactSubmissionStatusClaimedV0 ||
			incoming == DomainWorkArtifactSubmissionStatusSubmittingV0 ||
			incoming == DomainWorkArtifactSubmissionStatusRejectedV0 ||
			domainWorkSubmissionTerminalStatusV0(incoming)
	}
	if existing == DomainWorkArtifactSubmissionStatusRejectedV0 {
		return incoming == DomainWorkArtifactSubmissionStatusClaimedV0 ||
			incoming == DomainWorkArtifactSubmissionStatusSubmittingV0 ||
			incoming == DomainWorkArtifactSubmissionStatusRejectedV0 ||
			domainWorkSubmissionTerminalStatusV0(incoming)
	}
	return false
}
