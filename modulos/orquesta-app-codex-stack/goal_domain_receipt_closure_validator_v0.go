package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	goalDomainReceiptLedgerAcceptedEvidenceRefV0        = "evidence-ref-goal-domain-receipt-ledger-accepted"
	goalDomainReceiptLedgerMissingEvidenceRefV0         = "evidence-ref-goal-domain-receipt-ledger-missing"
	goalDomainReceiptLedgerUnavailableIssueV0           = "domain_work_receipt_ledger_unavailable"
	goalDomainReceiptLedgerAcceptedMissingIssueV0       = "domain_work_receipt_not_accepted"
	goalDomainReceiptLedgerIncompleteArtifactV0         = "domain_work_receipt_artifact_incomplete"
	goalDomainReceiptLedgerRequiredRunRefIssueV0        = "domain_work_receipt_run_ref_required"
	goalDomainReceiptLedgerRequiredReceiptIssueV0       = "domain_work_receipt_ref_required"
	goalDomainReceiptLedgerRequiredContractIssueV0      = "domain_work_receipt_contract_required"
	goalDomainReceiptLedgerRequiredArtifactIssueV0      = "domain_work_receipt_artifact_required"
	goalDomainReceiptOPESSubrolesEvidenceMissingIssueV0 = "domain_work_opes_subroles_evidence_missing"
	goalDomainReceiptLedgerRequiredArtifactFieldV0      = "domain_receipt_refs.artifact_contracts"
	goalDomainReceiptLedgerUnavailableIssueFieldV0      = "domain_receipt_refs.ledger"
	goalDomainReceiptLedgerMissingIssueFieldV0          = "domain_receipt_refs"
	goalDomainReceiptLedgerRequiredRunRefFieldV0        = "domain_receipt_refs.run_ref"
	goalDomainReceiptLedgerRequiredReceiptFieldV0       = "domain_receipt_refs.receipt_ref"
	goalDomainReceiptOPESSubrolesEvidenceFieldV0        = "domain_receipt_refs.opes_subroles"
	goalDomainReceiptOPESSubrolesAcceptedEvidenceRefV0  = "evidence-ref-goal-domain-receipt-opes-subroles-accepted"
	goalDomainReceiptOPESSubrolesEvidencePrefixV0       = "domain-work-opes-subrole-"
	goalDomainReceiptOPESSubrolesRequiredCountV0        = 6
)

type domainWorkGoalReceiptClosureValidatorV0 struct {
	Base   orquestagoal.GoalWorkClosureValidatorPortV0
	Ledger DomainWorkArtifactSubmissionRecordReaderPortV0
}

func (validator domainWorkGoalReceiptClosureValidatorV0) ValidateGoalWorkClosureV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
) (orquestagoal.GoalClosureValidationV0, error) {
	base := validator.Base
	if base == nil {
		base = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	}
	closure, err := base.ValidateGoalWorkClosureV0(ctx, spec, result)
	if err != nil || !closure.Accepted || !spec.ClosurePolicy.RequireDomainReceipt {
		return closure, err
	}
	receiptEvidence, receiptIssue := validator.validateAcceptedDomainReceiptsV0(ctx, spec, result)
	if receiptIssue.Code != "" {
		return goalDomainReceiptBlockedClosureV0(closure, receiptIssue), nil
	}
	closure.EvidenceRefs = compactStringsV0(append(closure.EvidenceRefs, receiptEvidence...))
	return closure, nil
}

func (validator domainWorkGoalReceiptClosureValidatorV0) validateAcceptedDomainReceiptsV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
) ([]string, orquestagoal.GoalWorkIssueV0) {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	if strings.TrimSpace(spec.RunRef) == "" {
		return nil, orquestagoal.GoalWorkIssueV0{
			Code:  goalDomainReceiptLedgerRequiredRunRefIssueV0,
			Field: goalDomainReceiptLedgerRequiredRunRefFieldV0,
		}
	}
	receiptRefs := compactStringsV0(result.DomainReceiptRefs)
	if len(receiptRefs) == 0 {
		return nil, orquestagoal.GoalWorkIssueV0{
			Code:  goalDomainReceiptLedgerRequiredReceiptIssueV0,
			Field: goalDomainReceiptLedgerRequiredReceiptFieldV0,
		}
	}
	if validator.Ledger == nil {
		return nil, orquestagoal.GoalWorkIssueV0{
			Code:  goalDomainReceiptLedgerUnavailableIssueV0,
			Field: goalDomainReceiptLedgerUnavailableIssueFieldV0,
		}
	}
	records, err := validator.Ledger.ListDomainWorkArtifactSubmissionsV0(
		ctx,
		DomainWorkArtifactSubmissionRecordFilterV0{
			RunRef: spec.RunRef,
			Status: DomainWorkArtifactSubmissionStatusAcceptedV0,
		},
	)
	if err != nil {
		return nil, orquestagoal.GoalWorkIssueV0{
			Code:  goalDomainReceiptLedgerUnavailableIssueV0,
			Field: goalDomainReceiptLedgerUnavailableIssueFieldV0,
		}
	}
	if !goalDomainReceiptHasAnyAcceptedRecordV0(records, receiptRefs) {
		return nil, orquestagoal.GoalWorkIssueV0{
			Code:  goalDomainReceiptLedgerAcceptedMissingIssueV0,
			Field: goalDomainReceiptLedgerMissingIssueFieldV0,
		}
	}
	required := goalDomainReceiptRequiredArtifactContractsV0(spec)
	if len(required) == 0 {
		return nil, orquestagoal.GoalWorkIssueV0{
			Code:  goalDomainReceiptLedgerRequiredContractIssueV0,
			Field: goalDomainReceiptLedgerRequiredArtifactFieldV0,
		}
	}
	matching := goalDomainReceiptMatchingRecordsV0(records, receiptRefs, required)
	if !goalDomainReceiptAllContractsCoveredV0(required, matching) {
		return nil, orquestagoal.GoalWorkIssueV0{
			Code:  goalDomainReceiptLedgerRequiredArtifactIssueV0,
			Field: goalDomainReceiptLedgerRequiredArtifactFieldV0,
		}
	}
	completeMatching := goalDomainReceiptCompleteRecordsV0(matching)
	if !goalDomainReceiptAllContractsCoveredV0(required, completeMatching) {
		return nil, orquestagoal.GoalWorkIssueV0{
			Code:  goalDomainReceiptLedgerIncompleteArtifactV0,
			Field: goalDomainReceiptLedgerRequiredArtifactFieldV0,
		}
	}
	if issue := goalDomainReceiptOPESSubrolesIssueV0(spec, result, completeMatching); issue.Code != "" {
		return nil, issue
	}
	evidenceRefs := goalDomainReceiptEvidenceRefsV0(completeMatching)
	if goalDomainReceiptRequiresOPESSubrolesV0(spec) {
		evidenceRefs = append(evidenceRefs, goalDomainReceiptOPESSubrolesAcceptedEvidenceRefV0)
	}
	return evidenceRefs, orquestagoal.GoalWorkIssueV0{}
}

func goalDomainReceiptBlockedClosureV0(
	closure orquestagoal.GoalClosureValidationV0,
	issue orquestagoal.GoalWorkIssueV0,
) orquestagoal.GoalClosureValidationV0 {
	closure.Status = orquestagoal.GoalStatusBlockedV0
	closure.Accepted = false
	closure.NeedsRework = true
	closure.Issues = append(closure.Issues, issue)
	closure.EvidenceRefs = compactStringsV0(append(
		closure.EvidenceRefs,
		goalDomainReceiptLedgerMissingEvidenceRefV0,
	))
	return closure
}

func goalDomainReceiptRequiredArtifactContractsV0(
	spec orquestagoal.GoalWorkSpecV0,
) []orquestagoal.GoalArtifactContractV0 {
	out := make([]orquestagoal.GoalArtifactContractV0, 0, len(spec.ArtifactContracts))
	for _, contract := range spec.ArtifactContracts {
		contract.ArtifactRef = strings.TrimSpace(contract.ArtifactRef)
		contract.ArtifactType = strings.TrimSpace(contract.ArtifactType)
		if !contract.Required || contract.ArtifactRef == "" {
			continue
		}
		out = append(out, contract)
	}
	return out
}

func goalDomainReceiptHasAnyAcceptedRecordV0(
	records []DomainWorkArtifactSubmissionRecordV0,
	receiptRefs []string,
) bool {
	for _, record := range records {
		record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
		if record.Status != DomainWorkArtifactSubmissionStatusAcceptedV0 ||
			record.ReceiptRef == "" {
			continue
		}
		if codexStackStringInSetV0(receiptRefs, record.ReceiptRef) {
			return true
		}
	}
	return false
}

func goalDomainReceiptMatchingRecordsV0(
	records []DomainWorkArtifactSubmissionRecordV0,
	receiptRefs []string,
	required []orquestagoal.GoalArtifactContractV0,
) []DomainWorkArtifactSubmissionRecordV0 {
	out := make([]DomainWorkArtifactSubmissionRecordV0, 0, len(records))
	for _, record := range records {
		record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
		if record.Status != DomainWorkArtifactSubmissionStatusAcceptedV0 ||
			record.ReceiptRef == "" ||
			!codexStackStringInSetV0(receiptRefs, record.ReceiptRef) {
			continue
		}
		for _, contract := range required {
			if goalDomainReceiptRecordMatchesContractV0(record, contract) {
				out = append(out, record)
				break
			}
		}
	}
	return out
}

func goalDomainReceiptAllContractsCoveredV0(
	required []orquestagoal.GoalArtifactContractV0,
	records []DomainWorkArtifactSubmissionRecordV0,
) bool {
	for _, contract := range required {
		covered := false
		for _, record := range records {
			if goalDomainReceiptRecordMatchesContractV0(record, contract) {
				covered = true
				break
			}
		}
		if !covered {
			return false
		}
	}
	return true
}

func goalDomainReceiptCompleteRecordsV0(
	records []DomainWorkArtifactSubmissionRecordV0,
) []DomainWorkArtifactSubmissionRecordV0 {
	out := make([]DomainWorkArtifactSubmissionRecordV0, 0, len(records))
	for _, record := range records {
		record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
		if record.CompleteJob {
			out = append(out, record)
		}
	}
	return out
}

func goalDomainReceiptRecordMatchesContractV0(
	record DomainWorkArtifactSubmissionRecordV0,
	contract orquestagoal.GoalArtifactContractV0,
) bool {
	artifactRef := strings.TrimSpace(contract.ArtifactRef)
	artifactType := strings.TrimSpace(contract.ArtifactType)
	return (artifactRef != "" && strings.TrimSpace(record.ArtifactRef) == artifactRef) ||
		(artifactType != "" && strings.TrimSpace(record.ArtifactType) == artifactType)
}

func goalDomainReceiptOPESSubrolesIssueV0(
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
	records []DomainWorkArtifactSubmissionRecordV0,
) orquestagoal.GoalWorkIssueV0 {
	if !goalDomainReceiptRequiresOPESSubrolesV0(spec) {
		return orquestagoal.GoalWorkIssueV0{}
	}
	evidenceRefs := append([]string(nil), result.EvidenceRefs...)
	for _, record := range records {
		record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
		evidenceRefs = append(evidenceRefs, record.EvidenceRefs...)
	}
	if goalDomainReceiptOPESSubroleEvidenceCountV0(evidenceRefs) >= goalDomainReceiptOPESSubrolesRequiredCountV0 {
		return orquestagoal.GoalWorkIssueV0{}
	}
	return orquestagoal.GoalWorkIssueV0{
		Code:  goalDomainReceiptOPESSubrolesEvidenceMissingIssueV0,
		Field: goalDomainReceiptOPESSubrolesEvidenceFieldV0,
	}
}

func goalDomainReceiptRequiresOPESSubrolesV0(
	spec orquestagoal.GoalWorkSpecV0,
) bool {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	for _, ref := range spec.ContextRefs {
		if strings.TrimSpace(ref.Kind) == "domain_interface" &&
			strings.TrimSpace(ref.Ref) == "opes.padre-tema-6-subroles.v1" {
			return true
		}
	}
	return goalDomainReceiptHasOPESSubrolesRequiredFieldV0(spec)
}

func goalDomainReceiptHasOPESSubrolesRequiredFieldV0(
	spec orquestagoal.GoalWorkSpecV0,
) bool {
	hasField := false
	hasValue := false
	for _, ref := range spec.ContextRefs {
		kind := strings.TrimSpace(ref.Kind)
		switch kind {
		case "input_field":
			if strings.TrimSpace(ref.Ref) == "input-field-subroles_required" {
				hasField = true
			}
		case "input_field_value":
			purpose := strings.TrimSpace(ref.Purpose)
			if strings.Contains(purpose, `"name":"subroles_required"`) &&
				strings.Contains(purpose, `"value":"6"`) {
				hasValue = true
			}
		}
	}
	return hasField && hasValue
}

func goalDomainReceiptOPESSubroleEvidenceCountV0(
	refs []string,
) int {
	seen := map[string]struct{}{}
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if !strings.HasPrefix(ref, goalDomainReceiptOPESSubrolesEvidencePrefixV0) {
			continue
		}
		suffix := strings.TrimPrefix(ref, goalDomainReceiptOPESSubrolesEvidencePrefixV0)
		if suffix == "" {
			continue
		}
		seen[suffix] = struct{}{}
	}
	return len(seen)
}

func goalDomainReceiptEvidenceRefsV0(
	records []DomainWorkArtifactSubmissionRecordV0,
) []string {
	refs := []string{goalDomainReceiptLedgerAcceptedEvidenceRefV0}
	for _, record := range records {
		record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
		refs = append(refs, record.ReceiptRef)
		refs = append(refs, record.EvidenceRefs...)
	}
	return compactStringsV0(refs)
}
