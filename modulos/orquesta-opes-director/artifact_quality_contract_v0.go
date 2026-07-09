package orquestaopesdirector

import (
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	OPESArtifactQualityContractSchemaV0 = "opes_artifact_quality_contract.v0"

	OPESArtifactQualityStatusCompleteV0    = "complete"
	OPESArtifactQualityStatusNeedsReworkV0 = "needs_rework"

	ErrOPESArtifactQualityVisualDidacticFunctionRequiredV0 = "opes_artifact_visual_didactic_function_required"
	ErrOPESArtifactQualityVisualPlacementRequiredV0        = "opes_artifact_visual_placement_required"
	ErrOPESArtifactQualityVisualAltTextRequiredV0          = "opes_artifact_visual_alt_text_required"
	ErrOPESArtifactQualityHTMLTopicPagesRequiredV0         = "opes_artifact_html_topic_pages_required"
	ErrOPESArtifactQualityHTMLValidationRequiredV0         = "opes_artifact_html_validation_required"
	ErrOPESArtifactQualityAudioManifestRequiredV0          = "opes_artifact_audio_manifest_required"
	ErrOPESArtifactQualityAudioRefsRequiredV0              = "opes_artifact_audio_refs_required"
	ErrOPESArtifactQualityTutorManifestRequiredV0          = "opes_artifact_tutor_manifest_required"
	ErrOPESArtifactQualityTutorQARequiredV0                = "opes_artifact_tutor_qa_required"
	ErrOPESArtifactQualitySourceRefsRequiredV0             = "opes_artifact_source_refs_required"
	ErrOPESArtifactQualitySourceReportRequiredV0           = "opes_artifact_source_report_required"
	ErrOPESArtifactQualityReviewFindingsRequiredV0         = "opes_artifact_review_findings_required"
	ErrOPESArtifactQualityReviewEvidenceRequiredV0         = "opes_artifact_review_evidence_required"
	ErrOPESArtifactQualityCasesSchemaRequiredV0            = "opes_artifact_cases_schema_required"
	ErrOPESArtifactQualityCasesCoverageRequiredV0          = "opes_artifact_cases_coverage_required"
	ErrOPESArtifactQualityInteractiveManifestRequiredV0    = "opes_artifact_interactive_manifest_required"
	ErrOPESArtifactQualityInteractiveQARequiredV0          = "opes_artifact_interactive_qa_required"
	ErrOPESArtifactQualityHelpManifestRequiredV0           = "opes_artifact_help_manifest_required"
	ErrOPESArtifactQualityHelpQARequiredV0                 = "opes_artifact_help_qa_required"
	ErrOPESArtifactQualityVisualReuseManifestRequiredV0    = "opes_artifact_visual_reuse_manifest_required"
	ErrOPESArtifactQualityVisualReuseDecisionRequiredV0    = "opes_artifact_visual_reuse_decision_required"
	ErrOPESArtifactQualityAuditDecisionRequiredV0          = "opes_artifact_audit_decision_required"
	ErrOPESArtifactQualityAuditEvidenceRequiredV0          = "opes_artifact_audit_evidence_required"
)

type OPESArtifactQualityContractRequestV0 struct {
	WorkKind     string
	ArtifactType string
	TopicRef     string
	Fields       []orquestadomainwork.DomainWorkFieldV0
	EvidenceRefs []string
}

type OPESArtifactQualityContractResultV0 struct {
	SchemaVersion string                       `json:"schema_version"`
	Status        string                       `json:"status"`
	WorkKind      string                       `json:"work_kind,omitempty"`
	ArtifactType  string                       `json:"artifact_type,omitempty"`
	TopicRef      string                       `json:"topic_ref,omitempty"`
	EvidenceRefs  []string                     `json:"evidence_refs,omitempty"`
	Issues        []OPESArtifactQualityIssueV0 `json:"issues,omitempty"`
}

type OPESArtifactQualityIssueV0 struct {
	Code         string   `json:"code"`
	Field        string   `json:"field,omitempty"`
	Message      string   `json:"message,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type opesArtifactQualityRequirementV0 struct {
	Code         string
	Field        string
	Message      string
	FieldNames   []string
	EvidenceRefs []string
}

func ValidateOPESArtifactQualityContractV0(
	request OPESArtifactQualityContractRequestV0,
) OPESArtifactQualityContractResultV0 {
	request = normalizeOPESArtifactQualityContractRequestV0(request)
	result := OPESArtifactQualityContractResultV0{
		SchemaVersion: OPESArtifactQualityContractSchemaV0,
		Status:        OPESArtifactQualityStatusCompleteV0,
		WorkKind:      request.WorkKind,
		ArtifactType:  request.ArtifactType,
		TopicRef:      request.TopicRef,
		EvidenceRefs:  compactStringsV0(request.EvidenceRefs),
	}
	for _, requirement := range opesArtifactQualityRequirementsV0(request.ArtifactType) {
		if opesArtifactQualityRequirementSatisfiedV0(request, requirement) {
			continue
		}
		result.Issues = append(result.Issues, OPESArtifactQualityIssueV0{
			Code:         requirement.Code,
			Field:        requirement.Field,
			Message:      requirement.Message,
			EvidenceRefs: compactStringsV0(requirement.EvidenceRefs),
		})
	}
	if len(result.Issues) > 0 {
		result.Status = OPESArtifactQualityStatusNeedsReworkV0
	}
	return result
}

func normalizeOPESArtifactQualityContractRequestV0(
	request OPESArtifactQualityContractRequestV0,
) OPESArtifactQualityContractRequestV0 {
	request.WorkKind = strings.TrimSpace(request.WorkKind)
	request.ArtifactType = normalizeOPESDirectorArtifactTypeV0(request.ArtifactType)
	if request.ArtifactType == "" && request.WorkKind != "" {
		request.ArtifactType = normalizeOPESDirectorArtifactTypeV0(
			orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(request.WorkKind),
		)
	}
	request.TopicRef = strings.TrimSpace(request.TopicRef)
	request.EvidenceRefs = compactStringsV0(request.EvidenceRefs)
	request.Fields = cloneFieldsV0(request.Fields)
	return request
}

func opesArtifactQualityRequirementsV0(artifactType string) []opesArtifactQualityRequirementV0 {
	switch strings.TrimSpace(artifactType) {
	case orquestadomainwork.DomainWorkArtifactTypeVisualAssetV0:
		return []opesArtifactQualityRequirementV0{
			{
				Code:       ErrOPESArtifactQualityVisualDidacticFunctionRequiredV0,
				Field:      "didactic_function",
				Message:    "visual final requires structured didactic function",
				FieldNames: []string{"didactic_function", "visual_didactic_function", "objective", "purpose"},
			},
			{
				Code:       ErrOPESArtifactQualityVisualPlacementRequiredV0,
				Field:      "placement_ref",
				Message:    "visual final requires anchor or placement reference",
				FieldNames: []string{"placement_ref", "anchor_ref", "topic_anchor_ref", "section_ref"},
			},
			{
				Code:       ErrOPESArtifactQualityVisualAltTextRequiredV0,
				Field:      "alt_text",
				Message:    "visual final requires alt text or accessibility note",
				FieldNames: []string{"alt_text", "accessibility_note", "visual_alt_text"},
			},
		}
	case orquestadomainwork.DomainWorkArtifactTypeLocalHTMLSiteV0:
		return []opesArtifactQualityRequirementV0{
			{
				Code:       ErrOPESArtifactQualityHTMLTopicPagesRequiredV0,
				Field:      "html_topic_pages_manifest",
				Message:    "HTML package requires topic pages manifest, not just index shell",
				FieldNames: []string{"html_topic_pages_manifest", "topic_pages_manifest", "html_pages_manifest", "topic_pages_count"},
			},
			{
				Code:         ErrOPESArtifactQualityHTMLValidationRequiredV0,
				Field:        "html_validation_report",
				Message:      "HTML package requires structured validation report",
				FieldNames:   []string{"html_validation_report", "html_link_report", "html_validation_status"},
				EvidenceRefs: []string{"html_validation_report", "html_topic_pages_manifest"},
			},
		}
	case orquestadomainwork.DomainWorkArtifactTypeAudioAssetV0:
		return []opesArtifactQualityRequirementV0{
			{
				Code:         ErrOPESArtifactQualityAudioManifestRequiredV0,
				Field:        "audio_manifest",
				Message:      "audio asset requires resumable manifest",
				FieldNames:   []string{"audio_manifest", "audio_manifest_ref", "audio_tts_operational_manifest"},
				EvidenceRefs: []string{"audio_tts_operational_manifest"},
			},
			{
				Code:         ErrOPESArtifactQualityAudioRefsRequiredV0,
				Field:        "audio_refs",
				Message:      "audio asset requires generated or reused MP3 refs",
				FieldNames:   []string{"audio_refs", "generated_mp3_refs", "skipped_valid_mp3_refs", "section_audio_refs"},
				EvidenceRefs: []string{"resume_without_duplicate_valid_mp3"},
			},
		}
	case orquestadomainwork.DomainWorkArtifactTypeTutorBotPackageV0:
		return []opesArtifactQualityRequirementV0{
			{
				Code:         ErrOPESArtifactQualityTutorManifestRequiredV0,
				Field:        "rag_corpus_manifest",
				Message:      "tutor/RAG package requires corpus manifest",
				FieldNames:   []string{"rag_corpus_manifest", "rag_manifest_ref", "tutor_bot_package_ref", "tutor_manifest"},
				EvidenceRefs: []string{"rag_corpus_manifest", "tutor_bot_package"},
			},
			{
				Code:         ErrOPESArtifactQualityTutorQARequiredV0,
				Field:        "tutor_qa_report",
				Message:      "tutor/RAG package requires structured QA report",
				FieldNames:   []string{"tutor_qa_report", "tutor_scope_guard_report", "tutor_qa_status"},
				EvidenceRefs: []string{"tutor_qa_report"},
			},
		}
	case orquestadomainwork.DomainWorkArtifactTypeExamResearchReportV0,
		orquestadomainwork.DomainWorkArtifactTypeSourceV0:
		return []opesArtifactQualityRequirementV0{
			{
				Code:         ErrOPESArtifactQualitySourceRefsRequiredV0,
				Field:        "source_refs",
				Message:      "source research requires source refs",
				FieldNames:   []string{"source_refs", "official_source_refs", "exam_source_refs", "url_refs"},
				EvidenceRefs: []string{"source_refs"},
			},
			{
				Code:         ErrOPESArtifactQualitySourceReportRequiredV0,
				Field:        "source_research_report",
				Message:      "source research requires usage/relevance report",
				FieldNames:   []string{"source_research_report", "research_report_ref", "source_coverage_report"},
				EvidenceRefs: []string{"source_research_report"},
			},
		}
	case orquestadomainwork.DomainWorkArtifactTypeBlockRevisionV0,
		orquestadomainwork.DomainWorkArtifactTypeAgentReviewReportV0,
		orquestadomainwork.DomainWorkArtifactTypeAgentPairReviewReportV0,
		orquestadomainwork.DomainWorkArtifactTypeDirectorReviewMatrixV0:
		return []opesArtifactQualityRequirementV0{
			{
				Code:       ErrOPESArtifactQualityReviewFindingsRequiredV0,
				Field:      "issue_refs",
				Message:    "review report requires structured findings or decisions",
				FieldNames: []string{"issue_refs", "findings", "accepted_refs", "rework_refs", "decision_matrix"},
			},
			{
				Code:         ErrOPESArtifactQualityReviewEvidenceRequiredV0,
				Field:        "evidence_refs",
				Message:      "review report requires causal evidence refs",
				FieldNames:   []string{"evidence_refs", "review_evidence_refs", "artifact_refs"},
				EvidenceRefs: []string{"review_report", "evidence_refs"},
			},
		}
	case orquestadomainwork.DomainWorkArtifactTypePracticalCasesV0:
		return []opesArtifactQualityRequirementV0{
			{
				Code:         ErrOPESArtifactQualityCasesSchemaRequiredV0,
				Field:        "practical_cases_schema_report",
				Message:      "practical cases require schema validation",
				FieldNames:   []string{"practical_cases_schema_report", "practical_cases_manifest", "cases_schema_status"},
				EvidenceRefs: []string{"practical_cases_schema_report"},
			},
			{
				Code:         ErrOPESArtifactQualityCasesCoverageRequiredV0,
				Field:        "practical_cases_coverage_report",
				Message:      "practical cases require topic coverage report",
				FieldNames:   []string{"practical_cases_coverage_report", "cases_coverage_report", "case_count"},
				EvidenceRefs: []string{"practical_cases_coverage_report"},
			},
		}
	case orquestadomainwork.DomainWorkArtifactTypeInteractivePracticeV0:
		return []opesArtifactQualityRequirementV0{
			{
				Code:         ErrOPESArtifactQualityInteractiveManifestRequiredV0,
				Field:        "interactive_practice_manifest",
				Message:      "interactive practice requires manifest",
				FieldNames:   []string{"interactive_practice_manifest", "learning_games_manifest", "practice_manifest"},
				EvidenceRefs: []string{"interactive_practice_manifest"},
			},
			{
				Code:         ErrOPESArtifactQualityInteractiveQARequiredV0,
				Field:        "interactive_practice_qa_report",
				Message:      "interactive practice requires QA or build/test report",
				FieldNames:   []string{"interactive_practice_qa_report", "learning_games_qa_report", "build_test_report"},
				EvidenceRefs: []string{"interactive_practice_qa_report"},
			},
		}
	case orquestadomainwork.DomainWorkArtifactTypeHelpPackageV0:
		return []opesArtifactQualityRequirementV0{
			{
				Code:         ErrOPESArtifactQualityHelpManifestRequiredV0,
				Field:        "help_manual_manifest",
				Message:      "help package requires manual manifest",
				FieldNames:   []string{"help_manual_manifest", "help_package_manifest", "manual_manifest"},
				EvidenceRefs: []string{"help_manual_manifest"},
			},
			{
				Code:         ErrOPESArtifactQualityHelpQARequiredV0,
				Field:        "help_manual_qa_report",
				Message:      "help package requires QA or link/capture report",
				FieldNames:   []string{"help_manual_qa_report", "help_link_report", "capture_manifest"},
				EvidenceRefs: []string{"help_manual_qa_report"},
			},
		}
	case "visual_reuse_manifest":
		return []opesArtifactQualityRequirementV0{
			{
				Code:         ErrOPESArtifactQualityVisualReuseManifestRequiredV0,
				Field:        "visual_reuse_manifest",
				Message:      "visual reuse requires structured manifest",
				FieldNames:   []string{"visual_reuse_manifest", "visual_reuse_manifest_ref"},
				EvidenceRefs: []string{"visual_reuse_manifest"},
			},
			{
				Code:       ErrOPESArtifactQualityVisualReuseDecisionRequiredV0,
				Field:      "visual_reuse_decision",
				Message:    "visual reuse requires copied/inserted counters or not-applicable justification",
				FieldNames: []string{"copied_visual_count", "inserted_visual_count", "visual_requirement_status", "visual_zero_justification_ref"},
			},
		}
	case opesDirectorArtifactTypeQualityAuditReportV0:
		return []opesArtifactQualityRequirementV0{
			{
				Code:       ErrOPESArtifactQualityAuditDecisionRequiredV0,
				Field:      "decision_global",
				Message:    "existing syllabus quality audit requires structured global decision",
				FieldNames: []string{"decision_global", "audit_decision", "quality_audit_decision"},
			},
			{
				Code:         ErrOPESArtifactQualityAuditEvidenceRequiredV0,
				Field:        "audit_evidence_refs",
				Message:      "existing syllabus quality audit requires findings, evidence refs or rework task requests",
				FieldNames:   []string{"audit_evidence_refs", "evidence_refs", "findings", "topic_refs", "rework_task_requests"},
				EvidenceRefs: []string{"opes_quality_audit_report", "existing_syllabus_quality_audit"},
			},
		}
	default:
		return nil
	}
}

func opesArtifactQualityRequirementSatisfiedV0(
	request OPESArtifactQualityContractRequestV0,
	requirement opesArtifactQualityRequirementV0,
) bool {
	if len(requirement.FieldNames) > 0 && opesArtifactQualityHasAnyFieldValueV0(request.Fields, requirement.FieldNames...) {
		return true
	}
	if len(requirement.EvidenceRefs) > 0 && topicRegistryEvidenceContainsAnyV0(request.EvidenceRefs, requirement.EvidenceRefs...) {
		return true
	}
	return false
}

func opesArtifactQualityHasAnyFieldValueV0(fields []orquestadomainwork.DomainWorkFieldV0, names ...string) bool {
	for _, name := range names {
		for _, field := range fields {
			if strings.TrimSpace(field.Name) != strings.TrimSpace(name) {
				continue
			}
			if strings.TrimSpace(field.Value) != "" || len(compactStringsV0(field.Values)) > 0 || len(field.ValueJSON) > 0 {
				return true
			}
		}
	}
	return false
}
