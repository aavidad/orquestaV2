package orquestaappcodexstack

import (
	"encoding/json"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	codexStackOPESFinalPackageEvidenceIncompleteIssueV0 = "opes_final_package_manifest_evidence_incomplete"
	goalDomainReceiptOPESFinalPackageEvidenceFieldV0    = "domain_receipt_refs.opes_final_package"
)

func codexStackDomainWorkIsOPESFinalPackageV0(domainRef string, workKind string, artifactType string) bool {
	finalWorkKind := codexStackDomainWorkFinalPackageWorkKindV0(workKind)
	finalArtifact := codexStackDomainWorkFinalPackageArtifactTypeV0(artifactType)
	return finalWorkKind || (codexStackOperationalClosureRefIsOPESV0(domainRef) && finalArtifact)
}

func codexStackDomainWorkFinalPackageWorkKindV0(workKind string) bool {
	switch normalizeDomainWorkDeliveryAliasV0(workKind) {
	case "finalize_topic_package",
		"finalize_temario_package",
		"close_temario_package",
		"finalize_syllabus_package",
		"close_syllabus_package":
		return true
	default:
		return false
	}
}

func codexStackDomainWorkFinalPackageArtifactTypeV0(artifactType string) bool {
	switch domainWorkDeliveryCanonicalArtifactTypeV0(artifactType) {
	case "completed_syllabus_package", orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0:
		return true
	default:
		return false
	}
}

func codexStackOPESFinalPackageSubmissionEvidenceCompleteV0(
	record DomainWorkArtifactSubmissionRecordV0,
) bool {
	record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
	return codexStackOPESFinalPackageEvidenceCompleteV0(
		record.PayloadFields,
		record.EvidenceRefs,
		record.PayloadRefs,
		record.ExternalRefs,
	)
}

func codexStackOPESFinalPackageEvidenceCompleteV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	evidenceRefs []string,
	payloadRefs []string,
	externalRefs []orquestadomainwork.DomainWorkExternalRefV0,
) bool {
	manifest, ok := codexStackOPESFinalPackageStructuredManifestV0(fields)
	if !ok {
		return false
	}
	for _, category := range codexStackOPESFinalPackageRequiredEvidenceCategoriesV0() {
		if len(manifest.RequiredEvidenceRefs[category]) == 0 {
			return false
		}
	}
	return true
}

type codexStackOPESFinalPackageManifestV0 struct {
	PackageRef           string
	ManifestRef          string
	ChecksumRefs         []string
	ValidationReportRef  string
	ReviewMatrixRef      string
	RequiredEvidenceRefs map[string][]string
}

func codexStackOPESFinalPackageRequiredEvidenceCategoriesV0() []string {
	return []string{"html", "rag", "audio", "tests", "visual", "qa"}
}

func codexStackOPESFinalPackageStructuredManifestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) (codexStackOPESFinalPackageManifestV0, bool) {
	for _, field := range fields {
		if domainWorkDeliveryCanonicalPayloadFieldNameV0(orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0, field.Name) != "manifest_cierre" {
			continue
		}
		manifest, ok := codexStackOPESFinalPackageManifestFromJSONV0(field.ValueJSON)
		if ok {
			return manifest, true
		}
	}
	return codexStackOPESFinalPackageManifestV0{}, false
}

func codexStackOPESFinalPackageManifestFromJSONV0(
	raw json.RawMessage,
) (codexStackOPESFinalPackageManifestV0, bool) {
	if len(raw) == 0 || !json.Valid(raw) {
		return codexStackOPESFinalPackageManifestV0{}, false
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return codexStackOPESFinalPackageManifestV0{}, false
	}
	if codexStackJSONRawStringV0(values["schema_version"]) != "opes_final_package_evidence_manifest.v0" {
		return codexStackOPESFinalPackageManifestV0{}, false
	}
	manifest := codexStackOPESFinalPackageManifestV0{
		PackageRef:           codexStackJSONRawStringV0(values["package_ref"]),
		ManifestRef:          codexStackJSONRawStringV0(values["manifest_ref"]),
		ChecksumRefs:         domainWorkDeliveryRawRefsV0(values["checksum_refs"]),
		ValidationReportRef:  codexStackJSONRawStringV0(values["validation_report_ref"]),
		ReviewMatrixRef:      codexStackJSONRawStringV0(values["review_matrix_ref"]),
		RequiredEvidenceRefs: codexStackOPESFinalPackageRequiredEvidenceRefsV0(values["required_evidence_refs"]),
	}
	if manifest.PackageRef == "" ||
		manifest.ManifestRef == "" ||
		len(manifest.ChecksumRefs) == 0 ||
		manifest.ValidationReportRef == "" ||
		manifest.ReviewMatrixRef == "" {
		return codexStackOPESFinalPackageManifestV0{}, false
	}
	return manifest, true
}

func codexStackOPESFinalPackageRequiredEvidenceRefsV0(
	raw json.RawMessage,
) map[string][]string {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil
	}
	var byCategory map[string]json.RawMessage
	if err := json.Unmarshal(raw, &byCategory); err != nil {
		return nil
	}
	out := map[string][]string{}
	for _, category := range codexStackOPESFinalPackageRequiredEvidenceCategoriesV0() {
		out[category] = compactCodexStackStringsV0(domainWorkDeliveryRawRefsV0(byCategory[category]))
	}
	return out
}

func codexStackJSONRawStringV0(raw json.RawMessage) string {
	if len(raw) == 0 || !json.Valid(raw) {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return ""
	}
	return strings.TrimSpace(text)
}
