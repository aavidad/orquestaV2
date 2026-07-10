package orquestadocumentextraction

import (
	"context"
	"fmt"
)

func ExtractDocumentV0(
	ctx context.Context,
	request DocumentExtractionRequestV0,
	ports DocumentExtractionPortsV0,
) (DocumentExtractionResultV0, error) {
	if err := validateDocumentExtractionRequestV0(request); err != nil {
		return DocumentExtractionResultV0{}, err
	}
	if err := validateDocumentExtractionPortsV0(ports); err != nil {
		return DocumentExtractionResultV0{}, err
	}
	request.Policy = normalizedDocumentExtractionPolicyV0(request.Policy)
	source, err := ports.Source.ResolveDocumentV0(ctx, request.DocumentRef)
	if err != nil {
		return DocumentExtractionResultV0{}, err
	}
	if source.DocumentRef != request.DocumentRef || blankDocumentValueV0(source.ContentHash) {
		return DocumentExtractionResultV0{}, fmt.Errorf("document_source_material_invalid")
	}
	normalized, err := ports.Normalizer.NormalizeDocumentV0(ctx, source, request.Policy)
	if err != nil {
		return DocumentExtractionResultV0{}, err
	}
	document, err := ports.Parser.ParseDocumentV0(ctx, normalized)
	if err != nil {
		return DocumentExtractionResultV0{}, err
	}
	if err := validateParsedDocumentV0(document, source); err != nil {
		return DocumentExtractionResultV0{}, err
	}
	locations, err := ports.Localizer.LocateDocumentSchemaV0(ctx, document, request.Schema)
	if err != nil {
		return DocumentExtractionResultV0{}, err
	}
	slices, err := BuildDocumentSchemaSlicesV0(document, request.Schema, locations, request.DocumentLanguage, request.NormalizationLocale, request.MaxFieldsPerSlice)
	if err != nil {
		return DocumentExtractionResultV0{}, err
	}
	result := DocumentExtractionResultV0{Document: document, SchemaSlices: slices}
	for _, slice := range slices {
		candidates, extractErr := ports.SchemaExtractor.ExtractDocumentSchemaSliceV0(ctx, document, slice)
		if extractErr != nil {
			return DocumentExtractionResultV0{}, extractErr
		}
		for _, candidate := range candidates {
			field, known := documentSchemaFieldByRefV0(request.Schema, candidate.FieldRef)
			if !known {
				result.Candidates = append(result.Candidates, candidate)
				result.Issues = append(result.Issues, DocumentExtractionIssueV0{Code: "document_candidate_field_unknown", CandidateRef: candidate.CandidateRef})
				continue
			}
			evidence, evidenceErr := ports.EvidenceLocator.LocateDocumentEvidenceV0(ctx, document, candidate)
			if evidenceErr != nil {
				return DocumentExtractionResultV0{}, evidenceErr
			}
			if !hasLocatedDocumentEvidenceV0(document.DocumentRef, evidence) {
				result.Candidates = append(result.Candidates, candidate)
				result.Issues = append(result.Issues, DocumentExtractionIssueV0{Code: "document_candidate_evidence_required", CandidateRef: candidate.CandidateRef})
				continue
			}
			candidate.EvidenceRefs = appendDocumentEvidenceRefsV0(candidate.EvidenceRefs, evidence)
			validation, validationErr := ports.Validator.ValidateDocumentFieldV0(ctx, field, candidate, request.NormalizationLocale)
			if validationErr != nil {
				return DocumentExtractionResultV0{}, validationErr
			}
			candidate = validation.Candidate
			candidate.ValidationRefs = append(candidate.ValidationRefs, validation.ValidationRefs...)
			accepted := validation.Accepted
			if request.RequireHumanReview || validation.RequiresHumanReview {
				if ports.HumanReview == nil {
					result.Candidates = append(result.Candidates, candidate)
					result.Issues = append(result.Issues, DocumentExtractionIssueV0{Code: "document_human_review_port_required", CandidateRef: candidate.CandidateRef})
					continue
				}
				review, reviewErr := ports.HumanReview.ReviewDocumentFieldV0(ctx, field, candidate)
				if reviewErr != nil {
					return DocumentExtractionResultV0{}, reviewErr
				}
				candidate = review.Candidate
				accepted = review.Accepted
				candidate.ValidationRefs = append(candidate.ValidationRefs, review.DecisionRef)
			}
			result.Candidates = append(result.Candidates, candidate)
			result.Evidence = append(result.Evidence, evidence...)
			if accepted {
				result.AcceptedFields = append(result.AcceptedFields, candidate)
			}
		}
	}
	for _, exporter := range ports.Exporters {
		if len(result.AcceptedFields) == 0 {
			break
		}
		artifact, exportErr := exporter.ExportDocumentFieldsV0(ctx, DocumentExportRequestV0{
			DocumentRef: document.DocumentRef, SchemaRef: request.Schema.SchemaRef, OutputLocale: request.OutputLocale,
			Fields: append([]DocumentFieldCandidateV0(nil), result.AcceptedFields...), Evidence: append([]DocumentEvidenceV0(nil), result.Evidence...),
		})
		if exportErr != nil {
			return DocumentExtractionResultV0{}, exportErr
		}
		result.Artifacts = append(result.Artifacts, artifact)
	}
	receipt := buildDocumentExtractionReceiptV0(request, ports, source, normalized, slices, result)
	receiptRef, receiptErr := ports.ReceiptStore.StoreDocumentExtractionReceiptV0(ctx, receipt)
	if receiptErr != nil {
		return DocumentExtractionResultV0{}, receiptErr
	}
	receipt.ReceiptRef = receiptRef
	result.Receipt = receipt
	return result, nil
}

func normalizedDocumentExtractionPolicyV0(policy DocumentExtractionPolicyV0) DocumentExtractionPolicyV0 {
	if policy.DataHandlingMode == "" {
		policy.DataHandlingMode = DocumentDataHandlingLocalV0
	}
	return policy
}

func validateParsedDocumentV0(document DocumentV0, source DocumentSourceMaterialV0) error {
	if document.IRVersion != DocumentExtractionIRSchemaVersionV0 || document.DocumentRef != source.DocumentRef || document.ContentHash != source.ContentHash || document.PageCount != len(document.Pages) {
		return fmt.Errorf("document_parser_output_invalid")
	}
	return nil
}

func appendDocumentEvidenceRefsV0(existing []string, evidence []DocumentEvidenceV0) []string {
	refs := append([]string(nil), existing...)
	for _, item := range evidence {
		if !blankDocumentValueV0(item.EvidenceRef) {
			refs = append(refs, item.EvidenceRef)
		}
	}
	return refs
}

func hasLocatedDocumentEvidenceV0(documentRef string, evidence []DocumentEvidenceV0) bool {
	for _, item := range evidence {
		if item.EvidenceRef == "" || item.DocumentRef != documentRef || item.PageRef == "" {
			continue
		}
		if item.BlockRef != "" || len(item.SpanRefs) > 0 || item.BoundingBox != nil || len(item.Polygon) > 0 {
			return true
		}
	}
	return false
}

func buildDocumentExtractionReceiptV0(request DocumentExtractionRequestV0, ports DocumentExtractionPortsV0, source DocumentSourceMaterialV0, normalized DocumentNormalizedMaterialV0, slices []DocumentSchemaSliceV0, result DocumentExtractionResultV0) DocumentExtractionReceiptV0 {
	adapters := []DocumentAdapterIdentityV0{ports.Source.AdapterIdentityV0(), ports.Normalizer.AdapterIdentityV0(), ports.Parser.AdapterIdentityV0(), ports.Localizer.AdapterIdentityV0(), ports.SchemaExtractor.AdapterIdentityV0(), ports.EvidenceLocator.AdapterIdentityV0(), ports.Validator.AdapterIdentityV0(), ports.ReceiptStore.AdapterIdentityV0()}
	if ports.HumanReview != nil {
		adapters = append(adapters, ports.HumanReview.AdapterIdentityV0())
	}
	for _, exporter := range ports.Exporters {
		adapters = append(adapters, exporter.AdapterIdentityV0())
	}
	receipt := DocumentExtractionReceiptV0{
		SchemaVersion: DocumentExtractionReceiptSchemaVersionV0, ContractVersion: DocumentExtractionContractSchemaVersionV0, IRVersion: DocumentExtractionIRSchemaVersionV0,
		DocumentRef: source.DocumentRef, ContentHash: source.ContentHash, SchemaRefs: []string{request.Schema.SchemaRef}, PolicyRef: request.Policy.PolicyRef,
		ConfigurationRef: request.ConfigurationRef, ConfigurationHash: request.ConfigurationHash, DocumentLanguage: request.DocumentLanguage,
		NormalizationLocale: request.NormalizationLocale, OutputLocale: request.OutputLocale, Adapters: adapters, Status: "completed",
	}
	for _, slice := range slices {
		receipt.SchemaSliceRefs = append(receipt.SchemaSliceRefs, slice.SliceRef)
	}
	for _, transform := range normalized.Transforms {
		receipt.TransformRefs = append(receipt.TransformRefs, transform.TransformRef)
	}
	for _, evidence := range result.Evidence {
		receipt.EvidenceRefs = append(receipt.EvidenceRefs, evidence.EvidenceRef)
	}
	for _, artifact := range result.Artifacts {
		receipt.ArtifactRefs = append(receipt.ArtifactRefs, artifact.ArtifactRef)
	}
	if len(result.Issues) > 0 {
		receipt.Status = "completed_with_issues"
	}
	return receipt
}
