package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	goalDomainReceiptLedgerAcceptedEvidenceRefV0         = "evidence-ref-goal-domain-receipt-ledger-accepted"
	goalDomainReceiptLedgerMissingEvidenceRefV0          = "evidence-ref-goal-domain-receipt-ledger-missing"
	goalDomainReceiptLedgerUnavailableIssueV0            = "domain_work_receipt_ledger_unavailable"
	goalDomainReceiptLedgerAcceptedMissingIssueV0        = "domain_work_receipt_not_accepted"
	goalDomainReceiptLedgerIncompleteArtifactV0          = "domain_work_receipt_artifact_incomplete"
	goalDomainReceiptLedgerRequiredRunRefIssueV0         = "domain_work_receipt_run_ref_required"
	goalDomainReceiptLedgerRequiredReceiptIssueV0        = "domain_work_receipt_ref_required"
	goalDomainReceiptLedgerRequiredContractIssueV0       = "domain_work_receipt_contract_required"
	goalDomainReceiptLedgerRequiredArtifactIssueV0       = "domain_work_receipt_artifact_required"
	goalDomainReceiptStructuredArtifactIssueCodeV0       = "domain_work_receipt_artifact_structured_non_terminal"
	goalDomainReceiptOPESSubrolesEvidenceMissingIssueV0  = "domain_work_opes_subroles_evidence_missing"
	goalDomainReceiptOPESFinalPackageEvidenceIssueCodeV0 = codexStackOPESFinalPackageEvidenceIncompleteIssueV0
	goalDomainReceiptOPESVisualFinalIssueCodeV0          = "domain_work_opes_visual_final_not_professional"
	goalDomainReceiptLedgerRequiredArtifactFieldV0       = "domain_receipt_refs.artifact_contracts"
	goalDomainReceiptStructuredArtifactFieldV0           = "domain_receipt_refs.artifact_payload"
	goalDomainReceiptLedgerUnavailableIssueFieldV0       = "domain_receipt_refs.ledger"
	goalDomainReceiptLedgerMissingIssueFieldV0           = "domain_receipt_refs"
	goalDomainReceiptLedgerRequiredRunRefFieldV0         = "domain_receipt_refs.run_ref"
	goalDomainReceiptLedgerRequiredReceiptFieldV0        = "domain_receipt_refs.receipt_ref"
	goalDomainReceiptOPESVisualFinalFieldV0              = "domain_receipt_refs.opes_visual_final"
	goalDomainReceiptOPESSubrolesEvidenceFieldV0         = "domain_receipt_refs.opes_subroles"
	goalDomainReceiptOPESSubrolesAcceptedEvidenceRefV0   = "evidence-ref-goal-domain-receipt-opes-subroles-accepted"
	goalDomainReceiptOPESSubrolesEvidencePrefixV0        = "domain-work-opes-subrole-"
	goalDomainReceiptOPESSubrolesRequiredCountV0         = 6
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

func (validator domainWorkGoalReceiptClosureValidatorV0) goalResultWithAcceptedDomainReceiptRefsV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
	result orquestagoal.GoalWorkResultV0,
) orquestagoal.GoalWorkResultV0 {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	if len(compactStringsV0(result.DomainReceiptRefs)) > 0 ||
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
				result.DomainReceiptRefs = append(result.DomainReceiptRefs, record.ReceiptRef)
				result.ArtifactRefs = append(result.ArtifactRefs, contract.ArtifactRef)
				result.EvidenceRefs = append(result.EvidenceRefs, "domain-work-goal-receipt-derived-"+safeDomainWorkEvidenceRefV0(record.ReceiptRef))
				break
			}
		}
	}
	return orquestagoal.NormalizeGoalWorkResultV0(result)
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
	if issue := goalDomainReceiptOPESVisualFinalIssueV0(spec, matching); issue.Code != "" {
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
			!goalDomainReceiptOPESVisualFinalInvalidV0(spec, record) {
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

func goalDomainReceiptOPESFinalPackageEvidenceIssueV0(
	spec orquestagoal.GoalWorkSpecV0,
	records []DomainWorkArtifactSubmissionRecordV0,
) orquestagoal.GoalWorkIssueV0 {
	for _, record := range records {
		if goalDomainReceiptOPESFinalPackageEvidenceMissingV0(spec, record) {
			return orquestagoal.GoalWorkIssueV0{
				Code:  goalDomainReceiptOPESFinalPackageEvidenceIssueCodeV0,
				Field: goalDomainReceiptOPESFinalPackageEvidenceFieldV0,
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
	return !codexStackOPESFinalPackageSubmissionEvidenceCompleteV0(record)
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
		if status == orquestagoal.GoalStatusAcceptedV0 || status == "passed" {
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
