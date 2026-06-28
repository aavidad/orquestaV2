package orquestaopesconnector

import "strings"

func opesTransportJobTypeForWorkKindV0(workKind string) string {
	workKind = strings.TrimSpace(workKind)
	switch workKind {
	case "update_topic_registry", "claim_topic_registry", "release_topic_registry",
		"review_codex", "review_gemini", "review_claude",
		"review_pair_codex_gemini", "review_pair_codex_claude", "review_pair_gemini_claude",
		"review_director_consolidation", "review_director_final", "review_consensus_director",
		"generate_agent_candidate_codex", "generate_agent_candidate_gemini", "generate_agent_candidate_claude",
		"generate_provider_candidate",
		"vote_agent_candidates_codex", "vote_agent_candidates_gemini", "vote_agent_candidates_claude",
		"vote_provider_candidates",
		"select_agent_candidate_director", "select_provider_candidate_director":
		return "review_textual"
	case "generate_learning_games", "create_learning_games", "generate_course_games":
		return "generate_tutor_assets"
	case "finalize_temario_package", "close_temario_package",
		"finalize_topic_package", "finalize_domain_package", "finalize_syllabus_package":
		return "generate_help_manual_assets"
	default:
		return workKind
	}
}
