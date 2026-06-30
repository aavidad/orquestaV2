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
	if !codexStackOPESFinalPackageHasManifestV0(fields, evidenceRefs, payloadRefs, externalRefs) {
		return false
	}
	tokens := codexStackOPESFinalPackageEvidenceTokensV0(fields, evidenceRefs, payloadRefs, externalRefs)
	required := map[string][]string{
		"html":  {"opes-final-evidence:html", "html_evidence_ref", "local_html_site", "html/index.html"},
		"rag":   {"opes-final-evidence:rag", "rag_evidence_ref", "rag/manifest.json", "tutor_rag_manifest"},
		"audio": {"opes-final-evidence:audio", "audio_evidence_ref", "audio/guion_audio.md", "audio_manifest"},
		"tests": {"opes-final-evidence:tests", "tests_evidence_ref", "question_bank", "tests.json"},
		"visual": {
			"opes-final-evidence:visual",
			"visual_evidence_ref",
			"visuales_plan.md",
			"visual_validation_report_ref",
		},
		"qa": {"opes-final-evidence:qa", "qa_evidence_ref", "qa_final.md", "review_matrix_ref", "director_review_matrix"},
	}
	for _, needles := range required {
		if !codexStackOperationalClosureEvidenceContainsAnyV0(tokens, needles...) {
			return false
		}
	}
	return true
}

func codexStackOPESFinalPackageHasManifestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	evidenceRefs []string,
	payloadRefs []string,
	externalRefs []orquestadomainwork.DomainWorkExternalRefV0,
) bool {
	for _, field := range fields {
		if domainWorkDeliveryCanonicalPayloadFieldNameV0(orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0, field.Name) != "manifest_cierre" {
			continue
		}
		if strings.TrimSpace(field.Value) != "" || len(compactCodexStackStringsV0(field.Values)) > 0 {
			return true
		}
		if len(field.ValueJSON) > 0 && json.Valid(field.ValueJSON) {
			return true
		}
	}
	tokens := codexStackOPESFinalPackageEvidenceTokensV0(nil, evidenceRefs, payloadRefs, externalRefs)
	return codexStackOperationalClosureEvidenceContainsAnyV0(tokens,
		"manifest_cierre",
		"manifest_cierre.json",
		"completed_syllabus_package_manifest",
		"opes-expected-evidence-manifest-cierre",
		"opes-rule-final-package-manifest",
	)
}

func codexStackOPESFinalPackageEvidenceTokensV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	evidenceRefs []string,
	payloadRefs []string,
	externalRefs []orquestadomainwork.DomainWorkExternalRefV0,
) []string {
	tokens := append([]string(nil), evidenceRefs...)
	tokens = append(tokens, payloadRefs...)
	for _, field := range fields {
		name := strings.TrimSpace(field.Name)
		if strings.TrimSpace(field.Value) != "" {
			tokens = append(tokens, name, name+":"+field.Value, field.Value)
		}
		for _, value := range field.Values {
			if strings.TrimSpace(value) != "" {
				tokens = append(tokens, name, name+":"+value, value)
			}
		}
		if len(field.ValueJSON) > 0 {
			tokens = append(tokens, name, string(field.ValueJSON))
		}
	}
	for _, ref := range externalRefs {
		tokens = append(tokens, ref.Kind, ref.Ref, strings.TrimSpace(ref.Kind)+":"+strings.TrimSpace(ref.Ref))
	}
	return compactCodexStackStringsV0(tokens)
}
