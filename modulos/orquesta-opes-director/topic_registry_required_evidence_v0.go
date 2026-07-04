package orquestaopesdirector

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func topicRegistryRequiredEvidencePendingRefsForRecordV0(record OPESCausalArtifactRecordV0) []string {
	sourceWorkKind := strings.TrimSpace(fieldStringV0(record.PayloadFields, "source_work_kind", "work_kind"))
	artifactType := topicRegistryEffectiveArtifactTypeForWorkKindV0(sourceWorkKind, record.ArtifactType)
	if artifactType == orquestadomainwork.DomainWorkArtifactTypeTopicRegistryUpdateV0 {
		return nil
	}
	if sourceWorkKind == "" && !topicRegistryArtifactRequiresEvidenceGateV0(artifactType) {
		return nil
	}
	refs := topicRegistryRequiredEvidenceRefsForRecordV0(record)
	var pending []string
	for _, requirement := range topicRegistryRequiredEvidenceRequirementsV0(sourceWorkKind, artifactType) {
		if !topicRegistryEvidenceContainsAnyV0(refs, requirement.AcceptedRefs...) {
			pending = append(pending, requirement.PendingRef)
		}
	}
	return compactStringsV0(pending)
}

type topicRegistryRequiredEvidenceRequirementV0 struct {
	PendingRef   string
	AcceptedRefs []string
}

func topicRegistryRequiredEvidenceRequirementsV0(
	sourceWorkKind string,
	artifactType string,
) []topicRegistryRequiredEvidenceRequirementV0 {
	sourceWorkKind = strings.TrimSpace(sourceWorkKind)
	artifactType = strings.TrimSpace(artifactType)
	switch artifactType {
	case orquestadomainwork.DomainDocumentPlanArtifactTypeV0:
		return []topicRegistryRequiredEvidenceRequirementV0{{
			PendingRef: "required-evidence-document-plan-contract",
			AcceptedRefs: []string{
				"opes-final-evidence:document_plan_contract",
				"document_plan_contract",
				"required_plan_parts",
			},
		}}
	case orquestadomainwork.DomainWorkArtifactTypeContentBlockV0,
		orquestadomainwork.DomainWorkArtifactTypeTopicSummaryV0,
		orquestadomainwork.DomainWorkArtifactTypeTopicExpansionPackageV0,
		orquestadomainwork.DomainWorkArtifactTypeAssembledTopicV0:
		if quality, ok := topicRegistryRequiredEvidenceTextQualityAliasesV0(sourceWorkKind); ok {
			return quality
		}
	case orquestadomainwork.DomainWorkArtifactTypeVisualAssetV0:
		return []topicRegistryRequiredEvidenceRequirementV0{{
			PendingRef: "required-evidence-didactic-visual-publicable",
			AcceptedRefs: []string{
				"opes-final-evidence:visual_didactic_publicable",
				"didactic_visual_report",
				"visual_anchor_manifest",
				"didactic_visual",
			},
		}}
	case orquestadomainwork.DomainWorkArtifactTypeBlockRevisionV0,
		orquestadomainwork.DomainWorkArtifactTypeAgentReviewReportV0,
		orquestadomainwork.DomainWorkArtifactTypeAgentPairReviewReportV0,
		orquestadomainwork.DomainWorkArtifactTypeDirectorReviewMatrixV0:
		return []topicRegistryRequiredEvidenceRequirementV0{{
			PendingRef: "required-evidence-review-report-actionable",
			AcceptedRefs: []string{
				"opes-final-evidence:review_report_actionable",
				"review_report",
				"issue_refs",
				"evidence_refs",
			},
		}}
	case orquestadomainwork.DomainWorkArtifactTypeExamResearchReportV0,
		orquestadomainwork.DomainWorkArtifactTypeSourceV0:
		return []topicRegistryRequiredEvidenceRequirementV0{{
			PendingRef: "required-evidence-source-research-traceable",
			AcceptedRefs: []string{
				"opes-final-evidence:source_research_traceable",
				"source_research_report",
				"source_refs",
			},
		}}
	case orquestadomainwork.DomainWorkArtifactTypePracticalCasesV0:
		return []topicRegistryRequiredEvidenceRequirementV0{{
			PendingRef: "required-evidence-practical-cases-publicable",
			AcceptedRefs: []string{
				"opes-final-evidence:practical_cases_publicable",
				"practical_cases_schema_report",
				"practical_cases_coverage_report",
			},
		}}
	case orquestadomainwork.DomainWorkArtifactTypeLocalHTMLSiteV0:
		return []topicRegistryRequiredEvidenceRequirementV0{{
			PendingRef: "required-evidence-html-site-publicable",
			AcceptedRefs: []string{
				"opes-final-evidence:html_site_publicable",
				"html_validation_report",
				"html_topic_pages_manifest",
			},
		}}
	case orquestadomainwork.DomainWorkArtifactTypeQuestionBankV0:
		return []topicRegistryRequiredEvidenceRequirementV0{{
			PendingRef: "required-evidence-question-bank-publicable",
			AcceptedRefs: []string{
				"opes-final-evidence:question_bank_publicable",
				"question_bank_structural_report",
				"question_bank_three_model_review",
			},
		}}
	case orquestadomainwork.DomainWorkArtifactTypeAudioAssetV0:
		return []topicRegistryRequiredEvidenceRequirementV0{{
			PendingRef: "required-evidence-audio-tts-resumable",
			AcceptedRefs: []string{
				"opes-final-evidence:audio_tts_resumable",
				"audio_tts_operational_manifest",
				"resume_without_duplicate_valid_mp3",
			},
		}}
	case orquestadomainwork.DomainWorkArtifactTypeTutorBotPackageV0:
		return []topicRegistryRequiredEvidenceRequirementV0{{
			PendingRef: "required-evidence-tutor-assets-publicable",
			AcceptedRefs: []string{
				"opes-final-evidence:tutor_assets_publicable",
				"tutor_qa_report",
				"rag_corpus_manifest",
			},
		}}
	case orquestadomainwork.DomainWorkArtifactTypeInteractivePracticeV0, "learning_games_package":
		return []topicRegistryRequiredEvidenceRequirementV0{{
			PendingRef: "required-evidence-interactive-practice-publicable",
			AcceptedRefs: []string{
				"opes-final-evidence:interactive_practice_publicable",
				"interactive_practice_manifest",
				"interactive_practice_qa_report",
			},
		}}
	case orquestadomainwork.DomainWorkArtifactTypeHelpPackageV0, "help_manual_package":
		return []topicRegistryRequiredEvidenceRequirementV0{{
			PendingRef: "required-evidence-help-manual-publicable",
			AcceptedRefs: []string{
				"opes-final-evidence:help_manual_publicable",
				"help_manual_manifest",
				"help_manual_qa_report",
			},
		}}
	case "visual_reuse_manifest":
		return []topicRegistryRequiredEvidenceRequirementV0{{
			PendingRef: "required-evidence-visual-reuse-manifest",
			AcceptedRefs: []string{
				"opes-final-evidence:visual_reuse",
				"visual_reuse_manifest",
			},
		}}
	}
	return nil
}

func topicRegistryRequiredEvidenceTextQualityAliasesV0(sourceWorkKind string) ([]topicRegistryRequiredEvidenceRequirementV0, bool) {
	switch strings.TrimSpace(sourceWorkKind) {
	case "draft_content_block",
		"summarize_topic",
		"expand_topic_from_summary",
		"assemble_topic",
		"validate_topic",
		"review_director_consolidation":
		return []topicRegistryRequiredEvidenceRequirementV0{{
			PendingRef: "required-evidence-topic-text-publicable",
			AcceptedRefs: []string{
				"opes-final-evidence:topic_quality_contract_pass",
				"topic_quality_contract_result",
				"public_text_qa_report",
				"topic_quality_status:complete",
			},
		}}, true
	default:
		return nil, false
	}
}

func topicRegistryRequiredEvidenceRefsForRecordV0(record OPESCausalArtifactRecordV0) []string {
	refs := append([]string(nil), record.EvidenceRefs...)
	refs = append(refs, record.PayloadRefs...)
	refs = append(refs, fieldStringsV0(
		record.PayloadFields,
		"evidence_refs",
		"validation_refs",
		"required_test_evidence_refs",
		"qa_report_refs",
		"topic_quality_evidence_refs",
	)...)
	for _, field := range record.PayloadFields {
		name := strings.ToLower(strings.TrimSpace(field.Name))
		value := strings.ToLower(strings.TrimSpace(field.Value))
		if name == "" || value == "" {
			continue
		}
		switch name {
		case "topic_quality_status":
			if value == OPESTopicQualityStatusCompleteV0 || value == "passed" || value == "pass" {
				refs = append(refs, "topic_quality_status:complete")
			}
		case "settlement_status":
			refs = append(refs, "settlement_status:"+value)
		case "artifact_type", "expected_artifact_type", "source_work_kind", "work_kind":
			refs = append(refs, value)
		}
	}
	return compactStringsV0(refs)
}

func topicRegistryArtifactRequiresEvidenceGateV0(artifactType string) bool {
	return len(topicRegistryRequiredEvidenceRequirementsV0("", artifactType)) > 0
}

func topicRegistryEffectiveArtifactTypeForWorkKindV0(sourceWorkKind string, artifactType string) string {
	sourceWorkKind = strings.TrimSpace(sourceWorkKind)
	artifactType = strings.TrimSpace(artifactType)
	if sourceWorkKind == "" {
		return artifactType
	}
	expected := strings.TrimSpace(orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(sourceWorkKind))
	if expected == "" || expected == orquestadomainwork.DomainWorkArtifactTypeGenericWorkDeliveryV0 {
		return artifactType
	}
	switch artifactType {
	case "", orquestadomainwork.DomainWorkArtifactTypeGenericWorkDeliveryV0:
		return expected
	default:
		return artifactType
	}
}
