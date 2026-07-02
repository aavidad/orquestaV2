package orquestaopesbridge

import (
	"strings"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

func TestOPESBridgeArtifactContractMapConsumeOwnerNeutralV0(t *testing.T) {
	for _, workKind := range []string{
		"update_topic_registry",
		"research_exam_precedents",
		"draft_content_block",
		"generate_visual_asset",
		"generate_question_bank",
		"generate_practical_cases",
		"generar_supuestos_practicos",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"review_codex",
		"review_gemini",
		"review_claude",
		"review_pair_codex_gemini",
		"review_pair_codex_claude",
		"review_pair_gemini_claude",
		"generate_agent_candidate_codex",
		"generate_agent_candidate_gemini",
		"generate_agent_candidate_claude",
		"vote_agent_candidates_codex",
		"vote_agent_candidates_gemini",
		"vote_agent_candidates_claude",
		"review_director_consolidation",
		"audit_existing_syllabus_quality",
		"validate_topic",
		"expand_topic_from_summary",
		"assemble_topic",
		"visual_asset_reuse",
		"generate_html_site",
		"generate_audio_asset",
		"generate_topic_audio",
		"generate_tutor_assets",
		"generate_learning_games",
		"generate_help_manual_assets",
		"finalize_temario_package",
		"configure_temario_bots",
		"unknown_work_kind",
	} {
		t.Run(workKind, func(t *testing.T) {
			expectedArtifact := expectedOPESArtifactForBridgeMapTestV0(workKind)
			if got := expectedArtifactTypeV0(workKind); got != expectedArtifact {
				t.Fatalf("artifact=%s want=%s", got, expectedArtifact)
			}
			req, ok := BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
				ID:          "job-ref-" + strings.ReplaceAll(workKind, "_", "-") + "-001",
				Type:        workKind,
				PayloadJSON: `{"topic_id":"topic-ref-artifact-map-001"}`,
			}, JobRunConfigV0{})
			if !ok || req.AppChangeRequest.ExternalWork == nil {
				t.Fatalf("request no construida")
			}
			if !fieldValueForTestV0(
				req.AppChangeRequest.ExternalWork.InputFields,
				"expected_artifact_type",
				expectedArtifact,
			) {
				t.Fatalf("input_fields=%+v", req.AppChangeRequest.ExternalWork.InputFields)
			}
			expectedContract := expectedOPESArtifactContractForBridgeMapTestV0(expectedArtifact)
			fields := req.AppChangeRequest.ExternalWork.InputFields
			if !fieldValueForTestV0(fields, "artifact_source_kind", expectedContract.SourceKind) ||
				!fieldValueForTestV0(fields, "artifact_canonicality", expectedContract.Canonicality) ||
				!fieldValueForTestV0(fields, "artifact_stage", expectedContract.Stage) ||
				!fieldValueForTestV0(fields, "artifact_materialization_target", expectedContract.MaterializationTarget) {
				t.Fatalf("artifact contract fields=%+v expected=%+v", fields, expectedContract)
			}
		})
	}
}

func expectedOPESArtifactForBridgeMapTestV0(workKind string) string {
	switch workKind {
	case "review_codex", "review_gemini", "review_claude":
		return orquestadomainwork.DomainWorkArtifactTypeAgentReviewReportV0
	case "review_pair_codex_gemini", "review_pair_codex_claude", "review_pair_gemini_claude":
		return orquestadomainwork.DomainWorkArtifactTypeAgentPairReviewReportV0
	case "generate_agent_candidate_codex", "generate_agent_candidate_gemini", "generate_agent_candidate_claude":
		return orquestadomainwork.DomainWorkArtifactTypeAgentCandidateV0
	case "vote_agent_candidates_codex", "vote_agent_candidates_gemini", "vote_agent_candidates_claude":
		return orquestadomainwork.DomainWorkArtifactTypeAgentCandidateVoteV0
	case "generate_learning_games":
		return opesArtifactTypeLearningGamesPackageV0
	case "generate_help_manual_assets":
		return opesArtifactTypeHelpManualPackageV0
	case "visual_asset_reuse":
		return opesArtifactTypeVisualReuseManifestV0
	case "generar_supuestos_practicos":
		return orquestadomainwork.DomainWorkArtifactTypePracticalCasesV0
	case "review_director_consolidation":
		return orquestadomainwork.DomainWorkArtifactTypeDirectorReviewMatrixV0
	case "audit_existing_syllabus_quality":
		return opesArtifactTypeQualityAuditReportV0
	case "finalize_temario_package":
		return opesArtifactTypeCompletedSyllabusPackageV0
	default:
		return orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind)
	}
}

func expectedOPESArtifactContractForBridgeMapTestV0(
	artifactType string,
) orquestadomainwork.DomainWorkArtifactContractV0 {
	contract := orquestadomainwork.ExpectedDomainWorkArtifactContractForArtifactTypeV0(artifactType)
	switch artifactType {
	case opesArtifactTypeCompletedSyllabusPackageV0:
		contract.SourceKind = orquestadomainwork.DomainWorkArtifactSourceKindMaterializableDomainV0
		contract.Canonicality = orquestadomainwork.DomainWorkArtifactCanonicalityCanonicalV0
		contract.Stage = orquestadomainwork.DomainWorkArtifactStageFinalV0
		contract.MaterializationTarget = orquestadomainwork.DomainWorkArtifactMaterializationTargetDomainV0
	case opesArtifactTypeQualityAuditReportV0:
		contract.SourceKind = orquestadomainwork.DomainWorkArtifactSourceKindEvidenceOnlyV0
		contract.Canonicality = orquestadomainwork.DomainWorkArtifactCanonicalityEvidenceOnlyV0
		contract.Stage = orquestadomainwork.DomainWorkArtifactStageReviewV0
		contract.MaterializationTarget = orquestadomainwork.DomainWorkArtifactMaterializationTargetEvidenceV0
	case opesArtifactTypeLearningGamesPackageV0, opesArtifactTypeHelpManualPackageV0,
		opesArtifactTypeVisualReuseManifestV0:
		contract.SourceKind = orquestadomainwork.DomainWorkArtifactSourceKindDerivedRegenerableV0
		contract.Canonicality = orquestadomainwork.DomainWorkArtifactCanonicalityDerivedRegenerableV0
		contract.Stage = orquestadomainwork.DomainWorkArtifactStageDerivedV0
		contract.MaterializationTarget = orquestadomainwork.DomainWorkArtifactMaterializationTargetRegenerateV0
	}
	return contract
}
