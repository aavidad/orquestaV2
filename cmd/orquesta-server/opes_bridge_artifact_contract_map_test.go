package main

import (
	"strings"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaopesbridge "orquesta/modulos/orquesta-opes-bridge"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

func TestOPESDrainArtifactContractMapConsumeOwnerNeutralV0(t *testing.T) {
	for _, workKind := range []string{
		"draft_content_block",
		"generate_visual_asset",
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
		"select_agent_candidate_director",
		"review_director_consolidation",
		"validate_topic",
		"expand_topic_from_summary",
		"assemble_topic",
		"generate_audio_asset",
		"generate_tutor_assets",
		"generate_learning_games",
		"generate_html_site",
		"generate_help_manual_assets",
		"finalize_topic_package",
		"finalize_temario_package",
		"unknown_work_kind",
	} {
		expectedArtifact := expectedOPESArtifactForServerMapTestV0(workKind)
		req, ok := orquestaopesbridge.BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
			ID:          "job-ref-" + strings.ReplaceAll(workKind, "_", "-") + "-001",
			Type:        workKind,
			PayloadJSON: `{"topic_id":"topic-ref-server-artifact-map-001"}`,
		}, orquestaopesbridge.JobRunConfigV0{})
		if !ok || req.AppChangeRequest.ExternalWork == nil {
			t.Fatalf("%s request no construida", workKind)
		}
		if !domainWorkFieldHasValueForServerArtifactMapTestV0(
			req.AppChangeRequest.ExternalWork.InputFields,
			"expected_artifact_type",
			expectedArtifact,
		) {
			t.Fatalf("%s input_fields=%+v", workKind, req.AppChangeRequest.ExternalWork.InputFields)
		}
	}
}

func expectedOPESArtifactForServerMapTestV0(workKind string) string {
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
		return "learning_games_package"
	case "generate_help_manual_assets":
		return "help_manual_package"
	case "review_director_consolidation":
		return orquestadomainwork.DomainWorkArtifactTypeDirectorReviewMatrixV0
	case "finalize_temario_package":
		return "completed_syllabus_package"
	case "finalize_topic_package":
		return orquestadomainwork.DomainWorkArtifactTypeFinalDomainPackageV0
	default:
		return orquestadomainwork.ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind)
	}
}

func domainWorkFieldHasValueForServerArtifactMapTestV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
	value string,
) bool {
	for _, field := range fields {
		if field.Name == name && field.Value == value {
			return true
		}
	}
	return false
}
