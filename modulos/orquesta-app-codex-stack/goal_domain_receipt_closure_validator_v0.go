package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	goalDomainReceiptLedgerAcceptedEvidenceRefV0            = "evidence-ref-goal-domain-receipt-ledger-accepted"
	goalDomainReceiptLedgerMissingEvidenceRefV0             = "evidence-ref-goal-domain-receipt-ledger-missing"
	goalDomainReceiptLedgerUnavailableIssueV0               = "domain_work_receipt_ledger_unavailable"
	goalDomainReceiptLedgerAcceptedMissingIssueV0           = "domain_work_receipt_not_accepted"
	goalDomainReceiptLedgerIncompleteArtifactV0             = "domain_work_receipt_artifact_incomplete"
	goalDomainReceiptLedgerRequiredRunRefIssueV0            = "domain_work_receipt_run_ref_required"
	goalDomainReceiptLedgerRequiredReceiptIssueV0           = "domain_work_receipt_ref_required"
	goalDomainReceiptLedgerRequiredContractIssueV0          = "domain_work_receipt_contract_required"
	goalDomainReceiptLedgerRequiredArtifactIssueV0          = "domain_work_receipt_artifact_required"
	goalDomainReceiptStructuredArtifactIssueCodeV0          = "domain_work_receipt_artifact_structured_non_terminal"
	goalDomainReceiptOPESSubrolesEvidenceMissingIssueV0     = "domain_work_opes_subroles_evidence_missing"
	goalDomainReceiptOPESFinalPackageEvidenceIssueCodeV0    = codexStackOPESFinalPackageEvidenceIncompleteIssueV0
	goalDomainReceiptOPESFinalPackageExtensionQAIssueV0     = codexStackOPESFinalPackageExtensionQAMissingIssueV0
	goalDomainReceiptOPESFinalPackageOfficialTextIssueV0    = codexStackOPESFinalPackageOfficialTextQAMissingIssueV0
	goalDomainReceiptOPESFinalPackageStrictEditorialIssueV0 = codexStackOPESFinalPackageStrictEditorialQAMissingIssueV0
	goalDomainReceiptOPESVisualFinalIssueCodeV0             = "domain_work_opes_visual_final_not_professional"
	goalDomainReceiptOPESVisualReuseIssueCodeV0             = "domain_work_opes_visual_reuse_missing"
	goalDomainReceiptOPESHTMLShellIssueCodeV0               = "domain_work_opes_html_shell_incomplete"
	goalDomainReceiptOPESPracticalCasesIssueCodeV0          = "domain_work_opes_practical_cases_contract_incomplete"
	goalDomainReceiptRequiredTestEvidenceMissingIssueV0     = "domain_work_required_test_evidence_missing"
	goalDomainReceiptLedgerRequiredArtifactFieldV0          = "domain_receipt_refs.artifact_contracts"
	goalDomainReceiptStructuredArtifactFieldV0              = "domain_receipt_refs.artifact_payload"
	goalDomainReceiptLedgerUnavailableIssueFieldV0          = "domain_receipt_refs.ledger"
	goalDomainReceiptLedgerMissingIssueFieldV0              = "domain_receipt_refs"
	goalDomainReceiptLedgerRequiredRunRefFieldV0            = "domain_receipt_refs.run_ref"
	goalDomainReceiptLedgerRequiredReceiptFieldV0           = "domain_receipt_refs.receipt_ref"
	goalDomainReceiptOPESVisualFinalFieldV0                 = "domain_receipt_refs.opes_visual_final"
	goalDomainReceiptOPESVisualReuseFieldV0                 = "domain_receipt_refs.opes_visual_reuse"
	goalDomainReceiptOPESHTMLShellFieldV0                   = "domain_receipt_refs.opes_html_shell"
	goalDomainReceiptOPESPracticalCasesFieldV0              = "domain_receipt_refs.opes_practical_cases"
	goalDomainReceiptRequiredTestEvidenceMissingFieldV0     = "required_test_results.evidence_refs"
	goalDomainReceiptOPESSubrolesEvidenceFieldV0            = "domain_receipt_refs.opes_subroles"
	goalDomainReceiptOPESSubrolesAcceptedEvidenceRefV0      = "evidence-ref-goal-domain-receipt-opes-subroles-accepted"
	goalDomainReceiptOPESSubrolesEvidencePrefixV0           = "domain-work-opes-subrole-"
	goalDomainReceiptOPESSubrolesRequiredCountV0            = 6
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
	if spec.ClosurePolicy.RequireDomainReceipt {
		result = validator.goalResultWithAcceptedDomainReceiptRefsV0(ctx, spec, result)
		result = validator.goalResultWithAcceptedDomainRequiredTestResultsV0(ctx, spec, result)
	}
	base := validator.Base
	if base == nil {
		base = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	}
	closure, err := base.ValidateGoalWorkClosureV0(ctx, spec, result)
	if err != nil || !spec.ClosurePolicy.RequireDomainReceipt {
		return closure, err
	}
	if !closure.Accepted {
		if goalDomainReceiptClosureOnlyBlockedByArtifactPathsV0(closure) {
			_, receiptIssue := validator.validateAcceptedDomainReceiptsV0(ctx, spec, result)
			if receiptIssue.Code != "" {
				return goalDomainReceiptBlockedClosureV0(closure, receiptIssue), nil
			}
		}
		return closure, nil
	}
	receiptEvidence, receiptIssue := validator.validateAcceptedDomainReceiptsV0(ctx, spec, result)
	if receiptIssue.Code != "" {
		return goalDomainReceiptBlockedClosureV0(closure, receiptIssue), nil
	}
	closure.EvidenceRefs = compactStringsV0(append(closure.EvidenceRefs, receiptEvidence...))
	return closure, nil
}

func goalDomainReceiptClosureOnlyBlockedByArtifactPathsV0(
	closure orquestagoal.GoalClosureValidationV0,
) bool {
	issues := closure.Issues
	if len(issues) == 0 {
		return false
	}
	for _, issue := range issues {
		if strings.TrimSpace(issue.Field) != "artifact_paths" {
			return false
		}
	}
	return true
}

func (validator domainWorkGoalReceiptClosureValidatorV0) goalResultWithAcceptedDomainReceiptRefsV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
) orquestagoal.GoalWorkResultV0 {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	hasReceiptRefs := len(compactStringsV0(result.DomainReceiptRefs)) > 0
	hasArtifactPaths := len(compactStringsV0(result.ArtifactPaths)) > 0
	if (hasReceiptRefs && (!spec.ClosurePolicy.RequireArtifactPaths || hasArtifactPaths)) ||
		strings.TrimSpace(spec.RunRef) == "" ||
		validator.Ledger == nil {
		return result
	}
	required := goalDomainReceiptRequiredArtifactContractsV0(spec)
	if len(required) == 0 {
		return result
	}
	records, err := validator.Ledger.ListDomainWorkArtifactSubmissionsV0(
		ctx,
		DomainWorkArtifactSubmissionRecordFilterV0{
			RunRef: spec.RunRef,
			Status: DomainWorkArtifactSubmissionStatusAcceptedV0,
		},
	)
	if err != nil {
		return result
	}
	complete := goalDomainReceiptCompleteRecordsV0(spec, records)
	if !goalDomainReceiptAllContractsCoveredV0(required, complete) {
		return result
	}
	for _, record := range complete {
		record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
		if strings.TrimSpace(record.ReceiptRef) == "" {
			continue
		}
		for _, contract := range required {
			if goalDomainReceiptRecordMatchesContractV0(record, contract) {
				if !hasReceiptRefs {
					result.DomainReceiptRefs = append(result.DomainReceiptRefs, record.ReceiptRef)
				}
				result.ArtifactRefs = append(result.ArtifactRefs, contract.ArtifactRef)
				if fileRef := goalDomainReceiptArtifactPathFromRecordV0(record); fileRef != "" {
					result.ArtifactPaths = append(result.ArtifactPaths, fileRef)
				}
				result.EvidenceRefs = append(result.EvidenceRefs, "domain-work-goal-receipt-derived-"+safeDomainWorkEvidenceRefV0(record.ReceiptRef))
				break
			}
		}
	}
	return orquestagoal.NormalizeGoalWorkResultV0(result)
}

func goalDomainReceiptArtifactPathFromRecordV0(
	record DomainWorkArtifactSubmissionRecordV0,
) string {
	record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
	for _, field := range record.PayloadFields {
		if goalMaterializedCanonicalQAKeyV0(field.Name) != "file_ref" {
			continue
		}
		value := filepath.ToSlash(filepath.Clean(strings.TrimSpace(field.Value)))
		if value == "" || value == "." || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "../") || value == ".." {
			continue
		}
		return value
	}
	return ""
}

func (validator domainWorkGoalReceiptClosureValidatorV0) goalResultWithAcceptedDomainRequiredTestResultsV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
) orquestagoal.GoalWorkResultV0 {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	if !spec.ClosurePolicy.RequireRequiredTests ||
		strings.TrimSpace(spec.RunRef) == "" ||
		validator.Ledger == nil {
		return result
	}
	required := goalDomainReceiptRequiredArtifactContractsV0(spec)
	if len(required) == 0 {
		return result
	}
	records, err := validator.Ledger.ListDomainWorkArtifactSubmissionsV0(
		ctx,
		DomainWorkArtifactSubmissionRecordFilterV0{
			RunRef: spec.RunRef,
			Status: DomainWorkArtifactSubmissionStatusAcceptedV0,
		},
	)
	if err != nil {
		return result
	}
	complete := goalDomainReceiptCompleteRecordsV0(spec, records)
	if !goalDomainReceiptAllContractsCoveredV0(required, complete) {
		return result
	}
	evidenceRefs := goalDomainReceiptEvidenceRefsV0(complete)
	for i := range result.RequiredTestResults {
		testRef := strings.TrimSpace(result.RequiredTestResults[i].TestRef)
		status := strings.TrimSpace(result.RequiredTestResults[i].Status)
		if testRef == "" ||
			(status != "passed" && status != orquestagoal.GoalStatusAcceptedV0) ||
			len(compactStringsV0(result.RequiredTestResults[i].EvidenceRefs)) > 0 {
			continue
		}
		if !goalDomainReceiptDomainOnlyRequiredTestRefInSpecV0(spec.RequiredTests, testRef) {
			continue
		}
		result.RequiredTestResults[i].EvidenceRefs = compactStringsV0(append(append([]string(nil), evidenceRefs...), goalDomainReceiptRequiredTestEvidenceRefsV0(spec.RequiredTests, testRef)...))
	}
	for _, test := range spec.RequiredTests {
		testRef := strings.TrimSpace(test.TestRef)
		if testRef == "" ||
			strings.TrimSpace(test.Command) != "" ||
			strings.TrimSpace(test.CommandRef) != "" ||
			goalDomainReceiptRequiredTestPassedV0(result.RequiredTestResults, testRef) {
			continue
		}
		result.RequiredTestResults = append(result.RequiredTestResults, orquestagoal.GoalRequiredTestResultV0{
			TestRef:      testRef,
			Status:       "passed",
			EvidenceRefs: compactStringsV0(append(append([]string(nil), evidenceRefs...), test.EvidenceRefs...)),
		})
	}
	return orquestagoal.NormalizeGoalWorkResultV0(result)
}

func goalDomainReceiptDomainOnlyRequiredTestRefInSpecV0(
	required []orquestagoal.GoalRequiredTestV0,
	testRef string,
) bool {
	testRef = strings.TrimSpace(testRef)
	if testRef == "" {
		return false
	}
	for _, test := range required {
		if strings.TrimSpace(test.TestRef) == testRef &&
			strings.TrimSpace(test.Command) == "" &&
			strings.TrimSpace(test.CommandRef) == "" {
			return true
		}
	}
	return false
}

func goalDomainReceiptRequiredTestEvidenceRefsV0(
	required []orquestagoal.GoalRequiredTestV0,
	testRef string,
) []string {
	testRef = strings.TrimSpace(testRef)
	for _, test := range required {
		if strings.TrimSpace(test.TestRef) == testRef {
			return compactStringsV0(test.EvidenceRefs)
		}
	}
	return nil
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
	if issue := goalDomainReceiptStructuredArtifactIssueV0(matching); issue.Code != "" {
		return nil, issue
	}
	if issue := goalDomainReceiptOPESFinalPackageEvidenceIssueV0(spec, matching); issue.Code != "" {
		return nil, issue
	}
	if issue := goalDomainReceiptOPESPracticalCasesIssueV0(spec, matching); issue.Code != "" {
		return nil, issue
	}
	if issue := goalDomainReceiptOPESVisualFinalIssueV0(spec, matching); issue.Code != "" {
		return nil, issue
	}
	if issue := goalDomainReceiptOPESVisualReuseIssueV0(spec, matching); issue.Code != "" {
		return nil, issue
	}
	if issue := goalDomainReceiptOPESHTMLShellIssueV0(spec, matching); issue.Code != "" {
		return nil, issue
	}
	if issue := goalDomainReceiptRequiredTestEvidenceIssueV0(spec, result); issue.Code != "" {
		return nil, issue
	}
	completeMatching := goalDomainReceiptCompleteRecordsV0(spec, matching)
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

func goalDomainReceiptRequiredTestEvidenceIssueV0(
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
) orquestagoal.GoalWorkIssueV0 {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	if !spec.ClosurePolicy.RequireRequiredTests {
		return orquestagoal.GoalWorkIssueV0{}
	}
	for _, test := range spec.RequiredTests {
		testRef := strings.TrimSpace(test.TestRef)
		if testRef == "" {
			continue
		}
		for _, observed := range result.RequiredTestResults {
			if strings.TrimSpace(observed.TestRef) != testRef {
				continue
			}
			status := strings.TrimSpace(observed.Status)
			if (status == orquestagoal.GoalStatusAcceptedV0 || status == "passed") &&
				len(compactStringsV0(observed.EvidenceRefs)) == 0 {
				return orquestagoal.GoalWorkIssueV0{
					Code:  goalDomainReceiptRequiredTestEvidenceMissingIssueV0,
					Field: goalDomainReceiptRequiredTestEvidenceMissingFieldV0,
				}
			}
		}
	}
	return orquestagoal.GoalWorkIssueV0{}
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
	spec orquestagoal.GoalWorkSpecV0,
	records []DomainWorkArtifactSubmissionRecordV0,
) []DomainWorkArtifactSubmissionRecordV0 {
	out := make([]DomainWorkArtifactSubmissionRecordV0, 0, len(records))
	for _, record := range records {
		record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
		if record.CompleteJob &&
			domainWorkStructuredNonTerminalArtifactIssueRefV0(record.PayloadFields) == "" &&
			!goalDomainReceiptOPESFinalPackageEvidenceMissingV0(spec, record) &&
			!goalDomainReceiptOPESPracticalCasesInvalidV0(spec, record) &&
			!goalDomainReceiptOPESVisualFinalInvalidV0(spec, record) &&
			!goalDomainReceiptOPESVisualReuseMissingV0(spec, record) &&
			!goalDomainReceiptOPESHTMLShellIncompleteV0(spec, record) {
			out = append(out, record)
		}
	}
	return out
}

func goalDomainReceiptStructuredArtifactIssueV0(
	records []DomainWorkArtifactSubmissionRecordV0,
) orquestagoal.GoalWorkIssueV0 {
	for _, record := range records {
		record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
		if domainWorkStructuredNonTerminalArtifactIssueRefV0(record.PayloadFields) != "" {
			return orquestagoal.GoalWorkIssueV0{
				Code:  goalDomainReceiptStructuredArtifactIssueCodeV0,
				Field: goalDomainReceiptStructuredArtifactFieldV0,
			}
		}
	}
	return orquestagoal.GoalWorkIssueV0{}
}

func goalDomainReceiptOPESPracticalCasesIssueV0(
	spec orquestagoal.GoalWorkSpecV0,
	records []DomainWorkArtifactSubmissionRecordV0,
) orquestagoal.GoalWorkIssueV0 {
	for _, record := range records {
		if goalDomainReceiptOPESPracticalCasesInvalidV0(spec, record) {
			return orquestagoal.GoalWorkIssueV0{
				Code:  goalDomainReceiptOPESPracticalCasesIssueCodeV0,
				Field: goalDomainReceiptOPESPracticalCasesFieldV0,
			}
		}
	}
	return orquestagoal.GoalWorkIssueV0{}
}

func goalDomainReceiptOPESVisualFinalIssueV0(
	spec orquestagoal.GoalWorkSpecV0,
	records []DomainWorkArtifactSubmissionRecordV0,
) orquestagoal.GoalWorkIssueV0 {
	for _, record := range records {
		if goalDomainReceiptOPESVisualFinalInvalidV0(spec, record) {
			return orquestagoal.GoalWorkIssueV0{
				Code:  goalDomainReceiptOPESVisualFinalIssueCodeV0,
				Field: goalDomainReceiptOPESVisualFinalFieldV0,
			}
		}
	}
	return orquestagoal.GoalWorkIssueV0{}
}

func goalDomainReceiptOPESVisualReuseIssueV0(
	spec orquestagoal.GoalWorkSpecV0,
	records []DomainWorkArtifactSubmissionRecordV0,
) orquestagoal.GoalWorkIssueV0 {
	for _, record := range records {
		if goalDomainReceiptOPESVisualReuseMissingV0(spec, record) {
			return orquestagoal.GoalWorkIssueV0{
				Code:  goalDomainReceiptOPESVisualReuseIssueCodeV0,
				Field: goalDomainReceiptOPESVisualReuseFieldV0,
			}
		}
	}
	return orquestagoal.GoalWorkIssueV0{}
}

func goalDomainReceiptOPESHTMLShellIssueV0(
	spec orquestagoal.GoalWorkSpecV0,
	records []DomainWorkArtifactSubmissionRecordV0,
) orquestagoal.GoalWorkIssueV0 {
	for _, record := range records {
		if goalDomainReceiptOPESHTMLShellIncompleteV0(spec, record) {
			return orquestagoal.GoalWorkIssueV0{
				Code:  goalDomainReceiptOPESHTMLShellIssueCodeV0,
				Field: goalDomainReceiptOPESHTMLShellFieldV0,
			}
		}
	}
	return orquestagoal.GoalWorkIssueV0{}
}

func goalDomainReceiptOPESFinalPackageEvidenceIssueV0(
	spec orquestagoal.GoalWorkSpecV0,
	records []DomainWorkArtifactSubmissionRecordV0,
) orquestagoal.GoalWorkIssueV0 {
	for _, record := range records {
		if issueRef := goalDomainReceiptOPESFinalPackageEvidenceIssueRefV0(spec, record); issueRef != "" {
			field := goalDomainReceiptOPESFinalPackageEvidenceFieldV0
			switch issueRef {
			case goalDomainReceiptOPESFinalPackageExtensionQAIssueV0,
				goalDomainReceiptOPESFinalPackageOfficialTextIssueV0,
				goalDomainReceiptOPESFinalPackageStrictEditorialIssueV0:
				field = goalDomainReceiptOPESFinalPackageQAPassesFieldV0
			case codexStackOPESFinalPackageTopicQualityMissingIssueV0:
				field = goalDomainReceiptOPESFinalPackageTopicQualityFieldV0
			}
			return orquestagoal.GoalWorkIssueV0{
				Code:  issueRef,
				Field: field,
			}
		}
	}
	return orquestagoal.GoalWorkIssueV0{}
}

func goalDomainReceiptOPESFinalPackageEvidenceMissingV0(
	spec orquestagoal.GoalWorkSpecV0,
	record DomainWorkArtifactSubmissionRecordV0,
) bool {
	record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
	domainRef := firstNonEmptyQueuedSourceV0(record.DomainRef, spec.DomainRef, spec.ProjectRef)
	workKind := firstNonEmptyQueuedSourceV0(domainWorkFieldStringValueV0(record.PayloadFields, "source_work_kind"), spec.WorkKind)
	if !codexStackDomainWorkIsOPESFinalPackageV0(domainRef, workKind, record.ArtifactType) {
		return false
	}
	return codexStackOPESFinalPackageEvidenceIssueRefV0(
		record.PayloadFields,
		record.EvidenceRefs,
		record.PayloadRefs,
		record.ExternalRefs,
	) != ""
}

func goalDomainReceiptOPESFinalPackageEvidenceIssueRefV0(
	spec orquestagoal.GoalWorkSpecV0,
	record DomainWorkArtifactSubmissionRecordV0,
) string {
	record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
	domainRef := firstNonEmptyQueuedSourceV0(record.DomainRef, spec.DomainRef, spec.ProjectRef)
	workKind := firstNonEmptyQueuedSourceV0(domainWorkFieldStringValueV0(record.PayloadFields, "source_work_kind"), spec.WorkKind)
	if !codexStackDomainWorkIsOPESFinalPackageV0(domainRef, workKind, record.ArtifactType) {
		return ""
	}
	return codexStackOPESFinalPackageEvidenceIssueRefV0(
		record.PayloadFields,
		record.EvidenceRefs,
		record.PayloadRefs,
		record.ExternalRefs,
	)
}

func goalDomainReceiptOPESPracticalCasesInvalidV0(
	spec orquestagoal.GoalWorkSpecV0,
	record DomainWorkArtifactSubmissionRecordV0,
) bool {
	record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
	if !goalDomainReceiptIsOPESPracticalCasesV0(spec, record) {
		return false
	}
	if goalDomainReceiptPracticalCasesHasBadStatusV0(record) {
		return true
	}
	if goalDomainReceiptFieldHasAnyValueV0(record, "missing_artifacts", "missing_files", "missing_folders", "missing_topics", "faltantes", "schema_errors", "validation_errors") {
		return true
	}
	if goalDomainReceiptDeliveredLessThanExpectedV0(record, "expected_artifacts", "delivered_artifacts") ||
		goalDomainReceiptDeliveredLessThanExpectedV0(record, "expected_folders", "delivered_folders") ||
		goalDomainReceiptDeliveredLessThanExpectedV0(record, "expected_topics", "delivered_topics") ||
		goalDomainReceiptDeliveredLessThanExpectedV0(record, "expected_topics", "covered_topics_count") {
		return true
	}
	if goalDomainReceiptFieldHasAnyValueV0(record, "tasks") &&
		!goalDomainReceiptFieldHasAnyValueV0(record, "questions") {
		return true
	}
	for _, value := range goalDomainReceiptFieldStringValuesV0(record, "kind", "question_kind", "question_kinds", "kinds") {
		if strings.EqualFold(strings.TrimSpace(value), "development_task") {
			return true
		}
	}
	if goalDomainReceiptFieldValueJSONContainsV0(record, "questions", `"kind":"development_task"`) ||
		goalDomainReceiptFieldValueJSONContainsV0(record, "body", `"kind":"development_task"`) {
		return true
	}
	return false
}

func goalDomainReceiptIsOPESPracticalCasesV0(
	spec orquestagoal.GoalWorkSpecV0,
	record DomainWorkArtifactSubmissionRecordV0,
) bool {
	domainRef := firstNonEmptyQueuedSourceV0(record.DomainRef, spec.DomainRef, spec.ProjectRef)
	if !codexStackOperationalClosureRefIsOPESV0(domainRef) {
		return false
	}
	values := []string{
		record.ArtifactType,
		spec.WorkKind,
		domainWorkFieldStringValueV0(record.PayloadFields, "source_work_kind"),
		domainWorkFieldStringValueV0(record.PayloadFields, "expected_artifact_type"),
		domainWorkFieldStringValueV0(record.PayloadFields, "schema_version"),
	}
	for _, value := range values {
		if goalDomainReceiptLooksLikePracticalCasesV0(value) {
			return true
		}
	}
	return false
}

func goalDomainReceiptLooksLikePracticalCasesV0(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	for _, marker := range []string{
		"supuesto",
		"practical_case",
		"practical_cases",
		"case_bank",
		"opes_practical_case",
	} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func goalDomainReceiptPracticalCasesHasBadStatusV0(
	record DomainWorkArtifactSubmissionRecordV0,
) bool {
	if goalDomainReceiptAnyFieldValueInSetV0(record, []string{
		"delivery_status",
		"coverage_status",
		"schema_status",
		"validation_status",
		"status",
		"editorial_decision",
	}, map[string]bool{
		"partial":           true,
		"incomplete":        true,
		"missing":           true,
		"schema_invalid":    true,
		"schema_repairable": true,
		"invalid":           true,
		"no_apto":           true,
		"needs_rework":      true,
	}) {
		return true
	}
	if goalDomainReceiptAnyFieldValueInSetV0(record, []string{"process_status"}, map[string]bool{
		"stopped":  true,
		"stop":     true,
		"finished": true,
	}) && !goalDomainReceiptPracticalCasesDeliveryCompleteV0(record) {
		return true
	}
	return false
}

func goalDomainReceiptPracticalCasesDeliveryCompleteV0(
	record DomainWorkArtifactSubmissionRecordV0,
) bool {
	return goalDomainReceiptAnyFieldValueInSetV0(record, []string{
		"delivery_status",
		"validation_status",
		"status",
	}, map[string]bool{
		"complete":               true,
		"completed":              true,
		"valid":                  true,
		"validated":              true,
		"apto":                   true,
		"apto_importacion_local": true,
		"ok":                     true,
	})
}

func goalDomainReceiptOPESVisualFinalInvalidV0(
	spec orquestagoal.GoalWorkSpecV0,
	record DomainWorkArtifactSubmissionRecordV0,
) bool {
	record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
	domainRef := firstNonEmptyQueuedSourceV0(record.DomainRef, spec.DomainRef, spec.ProjectRef)
	if !codexStackOperationalClosureRefIsOPESV0(domainRef) {
		return false
	}
	artifactType := domainWorkDeliveryCanonicalArtifactTypeV0(record.ArtifactType)
	workKind := normalizeDomainWorkDeliveryAliasV0(firstNonEmptyQueuedSourceV0(
		domainWorkFieldStringValueV0(record.PayloadFields, "source_work_kind"),
		spec.WorkKind,
	))
	if artifactType != "visual_asset" && workKind != "generate_visual_asset" {
		return false
	}
	return goalDomainReceiptVisualPayloadIsSVGV0(record)
}

func goalDomainReceiptOPESVisualReuseMissingV0(
	spec orquestagoal.GoalWorkSpecV0,
	record DomainWorkArtifactSubmissionRecordV0,
) bool {
	record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
	domainRef := firstNonEmptyQueuedSourceV0(record.DomainRef, spec.DomainRef, spec.ProjectRef)
	if !codexStackOperationalClosureRefIsOPESV0(domainRef) ||
		!goalDomainReceiptOPESVisualReuseGateAppliesV0(spec, record) ||
		goalDomainReceiptOPESVisualZeroJustifiedV0(record) {
		return false
	}
	visualCount, ok := domainWorkFieldIntValueV0(record.PayloadFields, "visual_count")
	if !ok || visualCount != 0 {
		return false
	}
	return goalDomainReceiptOPESVisualReuseRequiredV0(record) ||
		goalDomainReceiptOPESVisualReadyWithoutAssetsV0(record)
}

func goalDomainReceiptOPESVisualReuseGateAppliesV0(
	spec orquestagoal.GoalWorkSpecV0,
	record DomainWorkArtifactSubmissionRecordV0,
) bool {
	artifactType := domainWorkDeliveryCanonicalArtifactTypeV0(record.ArtifactType)
	workKind := normalizeDomainWorkDeliveryAliasV0(firstNonEmptyQueuedSourceV0(
		domainWorkFieldStringValueV0(record.PayloadFields, "source_work_kind"),
		spec.WorkKind,
	))
	switch artifactType {
	case "completed_syllabus_package", "html_site", "html_package", "html_export", "assembled_topic":
		return true
	}
	switch workKind {
	case "generate_html_site",
		"assemble_topic",
		"finalize_topic_package",
		"finalize_temario_package",
		"close_temario_package",
		"finalize_syllabus_package",
		"close_syllabus_package":
		return true
	default:
		return false
	}
}

func goalDomainReceiptOPESHTMLShellIncompleteV0(
	spec orquestagoal.GoalWorkSpecV0,
	record DomainWorkArtifactSubmissionRecordV0,
) bool {
	record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
	domainRef := firstNonEmptyQueuedSourceV0(record.DomainRef, spec.DomainRef, spec.ProjectRef)
	if !codexStackOperationalClosureRefIsOPESV0(domainRef) ||
		!goalDomainReceiptOPESVisualReuseGateAppliesV0(spec, record) ||
		!goalDomainReceiptOPESVisualReadyWithoutAssetsV0(record) {
		return false
	}
	if goalDomainReceiptAnyTruthyFieldV0(record,
		"topnav_present",
		"html_topnav_present",
		"legacy_topnav_present",
		"old_shell_present",
		"ui_label_documentacion_oficial_present",
		"public_label_documentacion_oficial_present",
		"broken_local_links",
		"local_links_broken",
	) {
		return true
	}
	if goalDomainReceiptAnyPositiveIntFieldV0(record,
		"topnav_count",
		"legacy_topnav_count",
		"missing_required_index_count",
		"missing_variant_index_count",
		"broken_local_link_count",
	) {
		return true
	}
	if goalDomainReceiptAnyFieldValueInSetV0(record, []string{
		"tcae_shell_status",
		"html_shell_status",
		"course_shell_status",
		"required_indexes_status",
		"variant_indexes_status",
		"local_links_status",
	}, map[string]bool{
		"missing":       true,
		"incomplete":    true,
		"legacy":        true,
		"old":           true,
		"broken":        true,
		"invalid":       true,
		"needs_rework":  true,
		"not_generated": true,
	}) {
		return true
	}
	return goalDomainReceiptFieldHasAnyValueV0(record,
		"missing_required_indexes",
		"missing_variant_indexes",
		"missing_index_refs",
		"broken_local_link_refs",
		"html_shell_issue_refs",
	)
}

func goalDomainReceiptOPESVisualReuseRequiredV0(
	record DomainWorkArtifactSubmissionRecordV0,
) bool {
	if goalDomainReceiptAnyPositiveIntFieldV0(record,
		"reusable_visual_count",
		"common_visual_count",
		"common_visual_asset_count",
		"visual_reuse_expected_count",
		"visual_assets_to_import_count",
		"visual_assets_to_copy_count",
		"visual_assets_to_insert_count",
		"missing_visual_asset_count",
		"pending_visual_asset_count",
	) {
		return true
	}
	if goalDomainReceiptAnyFieldValueInSetV0(record, []string{
		"visual_reuse_status",
		"common_visual_assets_status",
		"visual_assets_import_status",
		"visual_assets_copy_status",
		"visual_assets_insert_status",
		"visual_professional_status",
	}, map[string]bool{
		"pending":        true,
		"missing":        true,
		"not_imported":   true,
		"not_copied":     true,
		"not_inserted":   true,
		"needs_rework":   true,
		"requires_asset": true,
	}) {
		return true
	}
	return goalDomainReceiptFieldHasAnyValueV0(record,
		"common_visual_asset_refs",
		"reusable_visual_asset_refs",
		"visual_assets_to_import",
		"visual_assets_to_copy",
		"visual_assets_to_insert",
		"missing_visual_asset_refs",
		"pending_professional_visual_refs",
	)
}

func goalDomainReceiptOPESVisualReadyWithoutAssetsV0(
	record DomainWorkArtifactSubmissionRecordV0,
) bool {
	return goalDomainReceiptAnyFieldValueInSetV0(record, []string{
		"html_status",
		"ready_status",
		"publication_status",
		"delivery_status",
		"validation_status",
		"status",
	}, map[string]bool{
		"ready":                       true,
		"html_validado":               true,
		"ready_profesional":           true,
		"ready_candidate_html":        true,
		"apto_para_subida":            true,
		"apto_para_subida_controlada": true,
		"validated":                   true,
		"validado":                    true,
	})
}

func goalDomainReceiptOPESVisualZeroJustifiedV0(
	record DomainWorkArtifactSubmissionRecordV0,
) bool {
	if goalDomainReceiptAnyFieldValueInSetV0(record, []string{
		"visual_reuse_status",
		"common_visual_assets_status",
		"visual_policy",
		"visual_requirement_status",
	}, map[string]bool{
		"not_applicable":      true,
		"no_aplica":           true,
		"not_required":        true,
		"no_visual_required":  true,
		"no_visuals_required": true,
	}) {
		return true
	}
	return goalDomainReceiptFieldHasAnyValueV0(record,
		"visual_zero_justification_ref",
		"visual_absence_reason",
		"visual_not_required_reason",
		"no_visuals_reason",
	)
}

func goalDomainReceiptAnyPositiveIntFieldV0(
	record DomainWorkArtifactSubmissionRecordV0,
	names ...string,
) bool {
	for _, name := range names {
		value, ok := domainWorkFieldIntValueV0(record.PayloadFields, name)
		if ok && value > 0 {
			return true
		}
	}
	return false
}

func goalDomainReceiptAnyTruthyFieldV0(
	record DomainWorkArtifactSubmissionRecordV0,
	names ...string,
) bool {
	for _, value := range goalDomainReceiptFieldStringValuesV0(record, names...) {
		value = strings.ToLower(strings.TrimSpace(value))
		switch value {
		case "1", "true", "yes", "si", "sí", "present", "presentes", "detected", "found":
			return true
		}
	}
	for _, name := range names {
		value, ok := domainWorkFieldIntValueV0(record.PayloadFields, name)
		if ok && value > 0 {
			return true
		}
	}
	return false
}

func goalDomainReceiptVisualPayloadIsSVGV0(record DomainWorkArtifactSubmissionRecordV0) bool {
	for _, name := range []string{"format", "mime_type", "content_type", "asset_format"} {
		value := strings.ToLower(strings.TrimSpace(domainWorkFieldStringValueV0(record.PayloadFields, name)))
		if value == "svg" || value == "image/svg+xml" || strings.Contains(value, "svg_") || strings.Contains(value, "_svg") {
			return true
		}
	}
	if strings.TrimSpace(domainWorkFieldStringValueV0(record.PayloadFields, "svg")) != "" {
		return true
	}
	body := strings.ToLower(strings.TrimSpace(domainWorkFieldStringValueV0(record.PayloadFields, "body")))
	if strings.Contains(body, "<svg") {
		return true
	}
	for _, ref := range append(append([]string(nil), record.PayloadRefs...), record.EvidenceRefs...) {
		if strings.HasSuffix(strings.ToLower(strings.TrimSpace(ref)), ".svg") {
			return true
		}
	}
	for _, ref := range record.ExternalRefs {
		if strings.HasSuffix(strings.ToLower(strings.TrimSpace(ref.Ref)), ".svg") {
			return true
		}
	}
	return false
}

func goalDomainReceiptAnyFieldValueInSetV0(
	record DomainWorkArtifactSubmissionRecordV0,
	names []string,
	accepted map[string]bool,
) bool {
	for _, value := range goalDomainReceiptFieldStringValuesV0(record, names...) {
		value = strings.ToLower(strings.TrimSpace(value))
		if accepted[value] {
			return true
		}
	}
	return false
}

func goalDomainReceiptDeliveredLessThanExpectedV0(
	record DomainWorkArtifactSubmissionRecordV0,
	expectedName string,
	deliveredName string,
) bool {
	expected, okExpected := domainWorkFieldIntValueV0(record.PayloadFields, expectedName)
	delivered, okDelivered := domainWorkFieldIntValueV0(record.PayloadFields, deliveredName)
	return okExpected && okDelivered && expected > 0 && delivered >= 0 && delivered < expected
}

func goalDomainReceiptFieldStringValuesV0(
	record DomainWorkArtifactSubmissionRecordV0,
	names ...string,
) []string {
	out := []string{}
	for _, field := range record.PayloadFields {
		if !goalDomainReceiptFieldNameInSetV0(field.Name, names...) {
			continue
		}
		if strings.TrimSpace(field.Value) != "" {
			out = append(out, field.Value)
		}
		out = append(out, field.Values...)
		if len(field.ValueJSON) > 0 {
			out = append(out, goalDomainReceiptStringValuesFromJSONV0(field.ValueJSON)...)
		}
	}
	return compactStringsV0(out)
}

func goalDomainReceiptStringValuesFromJSONV0(raw json.RawMessage) []string {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}
	return goalDomainReceiptStringValuesV0(decoded)
}

func goalDomainReceiptFieldHasAnyValueV0(
	record DomainWorkArtifactSubmissionRecordV0,
	names ...string,
) bool {
	for _, field := range record.PayloadFields {
		if !goalDomainReceiptFieldNameInSetV0(field.Name, names...) {
			continue
		}
		if strings.TrimSpace(field.Value) != "" || len(compactStringsV0(field.Values)) > 0 {
			return true
		}
		if len(field.ValueJSON) > 0 && goalDomainReceiptJSONHasAnyValueV0(field.ValueJSON) {
			return true
		}
	}
	return false
}

func goalDomainReceiptJSONHasAnyValueV0(raw json.RawMessage) bool {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return len(strings.TrimSpace(string(raw))) > 0
	}
	switch typed := decoded.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return true
	}
}

func goalDomainReceiptFieldValueJSONContainsV0(
	record DomainWorkArtifactSubmissionRecordV0,
	name string,
	fragment string,
) bool {
	fragment = strings.TrimSpace(fragment)
	if fragment == "" {
		return false
	}
	for _, field := range record.PayloadFields {
		if !goalDomainReceiptFieldNameInSetV0(field.Name, name) || len(field.ValueJSON) == 0 {
			continue
		}
		compact := strings.ReplaceAll(string(field.ValueJSON), " ", "")
		compact = strings.ReplaceAll(compact, "\n", "")
		compact = strings.ReplaceAll(compact, "\t", "")
		if strings.Contains(compact, fragment) {
			return true
		}
	}
	return false
}

func goalDomainReceiptFieldNameInSetV0(fieldName string, names ...string) bool {
	fieldName = normalizedExternalWorkFieldNameV0(fieldName)
	for _, name := range names {
		if fieldName == normalizedExternalWorkFieldNameV0(name) {
			return true
		}
	}
	return false
}

func goalDomainReceiptRequiredTestPassedV0(
	results []orquestagoal.GoalRequiredTestResultV0,
	testRef string,
) bool {
	testRef = strings.TrimSpace(testRef)
	if testRef == "" {
		return false
	}
	for _, result := range results {
		if strings.TrimSpace(result.TestRef) != testRef {
			continue
		}
		status := strings.TrimSpace(result.Status)
		if (status == orquestagoal.GoalStatusAcceptedV0 || status == "passed") &&
			len(compactStringsV0(result.EvidenceRefs)) > 0 {
			return true
		}
	}
	return false
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
	expectedTaskRefs := goalDomainReceiptOPESSubroleExpectedTaskRefsV0(spec)
	if len(expectedTaskRefs) >= goalDomainReceiptOPESSubrolesRequiredCountV0 {
		if goalDomainReceiptOPESSubroleExpectedEvidenceCountV0(evidenceRefs, expectedTaskRefs) >= len(expectedTaskRefs) {
			return orquestagoal.GoalWorkIssueV0{}
		}
		return orquestagoal.GoalWorkIssueV0{
			Code:  goalDomainReceiptOPESSubrolesEvidenceMissingIssueV0,
			Field: goalDomainReceiptOPESSubrolesEvidenceFieldV0,
		}
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
	for _, ref := range spec.ContextRefs {
		kind := strings.TrimSpace(ref.Kind)
		switch kind {
		case "input_field":
			if strings.TrimSpace(ref.Ref) == "input-field-subroles_required" {
				continue
			}
		case "input_field_value":
			if goalDomainReceiptInputFieldRequiresOPESSubrolesV0(ref.Purpose) {
				return true
			}
		}
	}
	return false
}

func goalDomainReceiptInputFieldRequiresOPESSubrolesV0(
	purpose string,
) bool {
	purpose = strings.TrimSpace(purpose)
	start := strings.Index(purpose, "{")
	if start < 0 {
		return strings.Contains(purpose, `"name":"subroles_required"`) &&
			(strings.Contains(purpose, `"value":"6"`) ||
				strings.Contains(purpose, `"value_json":6`) ||
				strings.Contains(purpose, `"value_json":true`))
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(purpose[start:]), &payload); err != nil {
		return false
	}
	if normalizedExternalWorkFieldNameV0(goalDomainReceiptStringValueV0(payload["name"])) != "subroles_required" {
		return false
	}
	for _, key := range []string{"value", "value_json", "values"} {
		if goalDomainReceiptOPESSubrolesRequiredCountValueV0(payload[key]) >= goalDomainReceiptOPESSubrolesRequiredCountV0 {
			return true
		}
	}
	return false
}

func goalDomainReceiptOPESSubroleExpectedTaskRefsV0(
	spec orquestagoal.GoalWorkSpecV0,
) []string {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	out := []string{}
	for _, ref := range spec.ContextRefs {
		switch strings.TrimSpace(ref.Kind) {
		case "opes_subrole_task_ref":
			out = append(out, ref.Ref)
			continue
		case "input_field_value":
		default:
			continue
		}
		values := goalDomainReceiptInputFieldStringValuesV0(ref.Purpose, "opes_subrole_task_refs")
		out = append(out, values...)
	}
	return compactStringsV0(out)
}

func goalDomainReceiptInputFieldStringValuesV0(
	purpose string,
	name string,
) []string {
	purpose = strings.TrimSpace(purpose)
	start := strings.Index(purpose, "{")
	if start < 0 {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(purpose[start:]), &payload); err != nil {
		return nil
	}
	if normalizedExternalWorkFieldNameV0(goalDomainReceiptStringValueV0(payload["name"])) != normalizedExternalWorkFieldNameV0(name) {
		return nil
	}
	values := []string{}
	if value := goalDomainReceiptStringValueV0(payload["value"]); value != "" {
		values = append(values, value)
	}
	values = append(values, goalDomainReceiptStringValuesV0(payload["values"])...)
	values = append(values, goalDomainReceiptStringValuesV0(payload["value_json"])...)
	return compactStringsV0(values)
}

func goalDomainReceiptStringValuesV0(value any) []string {
	switch typed := value.(type) {
	case string:
		return []string{strings.TrimSpace(typed)}
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, goalDomainReceiptStringValuesV0(item)...)
		}
		return compactStringsV0(out)
	case []string:
		return compactStringsV0(typed)
	default:
		return nil
	}
}

func goalDomainReceiptStringValueV0(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func goalDomainReceiptOPESSubrolesRequiredCountValueV0(value any) int {
	switch typed := value.(type) {
	case string:
		text := strings.TrimSpace(typed)
		if count, err := strconv.Atoi(text); err == nil {
			return count
		}
		if strings.EqualFold(text, "true") {
			return goalDomainReceiptOPESSubrolesRequiredCountV0
		}
	case float64:
		return int(typed)
	case bool:
		if typed {
			return goalDomainReceiptOPESSubrolesRequiredCountV0
		}
	case []any:
		return len(typed)
	case []string:
		return len(compactStringsV0(typed))
	}
	return 0
}

func goalDomainReceiptOPESSubroleExpectedEvidenceCountV0(
	refs []string,
	expectedTaskRefs []string,
) int {
	seen := map[string]struct{}{}
	expected := map[string]struct{}{}
	for _, taskRef := range expectedTaskRefs {
		taskRef = strings.TrimSpace(taskRef)
		if taskRef == "" {
			continue
		}
		expected[taskRef] = struct{}{}
	}
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if !strings.HasPrefix(ref, goalDomainReceiptOPESSubrolesEvidencePrefixV0) {
			continue
		}
		suffix := strings.TrimPrefix(ref, goalDomainReceiptOPESSubrolesEvidencePrefixV0)
		if _, ok := expected[suffix]; ok {
			seen[suffix] = struct{}{}
		}
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
