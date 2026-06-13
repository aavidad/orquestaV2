package orquestadomainwork

import "strings"

const (
	DomainWorkArtifactTypeContentBlockV0          = "content_block"
	DomainWorkArtifactTypeVisualAssetV0           = "visual_asset"
	DomainWorkArtifactTypeBlockRevisionV0         = "block_revision"
	DomainWorkArtifactTypeSourceV0                = "source"
	DomainWorkArtifactTypeTopicStructureV0        = "topic_structure"
	DomainWorkArtifactTypeTopicOutlineV0          = "topic_outline"
	DomainWorkArtifactTypeTopicSummaryV0          = "topic_summary"
	DomainWorkArtifactTypeTopicExpansionPackageV0 = "topic_expansion_package"
	DomainWorkArtifactTypeAssembledTopicV0        = "assembled_topic"
	DomainWorkArtifactTypeAudioAssetV0            = "audio_asset"
	DomainWorkArtifactTypeExamResearchReportV0    = "exam_research_report"
	DomainWorkArtifactTypeQuestionBankV0          = "question_bank"
	DomainWorkArtifactTypeLocalHTMLSiteV0         = "local_html_site"
	DomainWorkArtifactTypeTutorBotPackageV0       = "tutor_bot_package"
	DomainWorkArtifactTypeInteractivePracticeV0   = "interactive_practice_package"
	DomainWorkArtifactTypeHelpPackageV0           = "help_package"
	DomainWorkArtifactTypeAgentReviewReportV0     = "agent_review_report"
	DomainWorkArtifactTypeAgentPairReviewReportV0 = "agent_pair_review_report"
	DomainWorkArtifactTypeAgentCandidateV0        = "agent_candidate_artifact"
	DomainWorkArtifactTypeAgentCandidateVoteV0    = "agent_candidate_vote_report"
	DomainWorkArtifactTypeAgentCandidateSelectV0  = "agent_candidate_selection_matrix"
	DomainWorkArtifactTypeDirectorReviewMatrixV0  = "director_review_matrix"
	DomainWorkArtifactTypeFinalDomainPackageV0    = "final_domain_package"
	DomainWorkArtifactTypeGenericWorkDeliveryV0   = "work_delivery"
)

func ExpectedDomainWorkArtifactTypeForWorkKindV0(workKind string) string {
	switch strings.TrimSpace(workKind) {
	case "draft_content_block", "generate_block", "generate_program_topic_draft":
		return DomainWorkArtifactTypeContentBlockV0
	case "generate_visual_asset":
		return DomainWorkArtifactTypeVisualAssetV0
	case "review_legal", "review_pedagogical", "review_quality", "validate_topic":
		return DomainWorkArtifactTypeBlockRevisionV0
	case "research_sources", "download_source", "verify_sources":
		return DomainWorkArtifactTypeSourceV0
	case "research_exam_precedents", "research_exam_results",
		"research_related_administration_exams", "buscar_examenes_ope",
		"investigar_examenes_administraciones":
		return DomainWorkArtifactTypeExamResearchReportV0
	case "split_syllabus_topic":
		return DomainWorkArtifactTypeTopicStructureV0
	case "draft_topic_outline", "create_exam_outline":
		return DomainWorkArtifactTypeTopicOutlineV0
	case "summarize_block", "summarize_chapter", "summarize_topic":
		return DomainWorkArtifactTypeTopicSummaryV0
	case "expand_topic_from_summary":
		return DomainWorkArtifactTypeTopicExpansionPackageV0
	case "plan_documento", "plan_tema", "plan_temario":
		return DomainDocumentPlanArtifactTypeV0
	case "assemble_topic":
		return DomainWorkArtifactTypeAssembledTopicV0
	case "generate_question_bank", "generate_topic_tests", "create_topic_tests",
		"crear_tests_tema", "banco_preguntas_tema":
		return DomainWorkArtifactTypeQuestionBankV0
	case "generate_html_site", "generate_local_html_site",
		"assemble_local_html_site", "crear_html_temario", "html_temario_local":
		return DomainWorkArtifactTypeLocalHTMLSiteV0
	case "generate_audio_asset", "generate_topic_audio", "create_topic_audio",
		"create_audio_asset", "synthesize_topic_audio", "narrate_topic", "tts_topic",
		"generacion_audio", "generar_audio", "crear_audio_tema", "audio_tema",
		"narracion_tema", "sintesis_voz_tema", "topic_audio":
		return DomainWorkArtifactTypeAudioAssetV0
	case "generate_tutor_assets", "configure_temario_tutor", "configure_temario_bots",
		"create_tutor_bot_package", "crear_tutor_temario", "bots_temario",
		"generate_rag_assets", "generate_rag_tutor_assets", "create_rag_corpus":
		return DomainWorkArtifactTypeTutorBotPackageV0
	case "generate_interactive_practice", "create_interactive_practice",
		"generate_practice_package", "generate_learning_games", "create_learning_games",
		"generate_learning_game_package":
		return DomainWorkArtifactTypeInteractivePracticeV0
	case "generate_help_package", "create_help_package", "help_package",
		"generate_help_manual_assets", "generate_help_manuals":
		return DomainWorkArtifactTypeHelpPackageV0
	case "review_agent_independent", "review_independent_agent",
		"review_codex", "review_gemini", "review_claude":
		return DomainWorkArtifactTypeAgentReviewReportV0
	case "review_agent_pair", "review_peer_pair", "review_pair",
		"review_pair_codex_gemini", "review_pair_codex_claude",
		"review_pair_gemini_claude":
		return DomainWorkArtifactTypeAgentPairReviewReportV0
	case "generate_agent_candidate":
		return DomainWorkArtifactTypeAgentCandidateV0
	case "vote_agent_candidates":
		return DomainWorkArtifactTypeAgentCandidateVoteV0
	case "select_agent_candidate_director":
		return DomainWorkArtifactTypeAgentCandidateSelectV0
	case "review_director_consolidation", "review_consensus_director":
		return DomainWorkArtifactTypeDirectorReviewMatrixV0
	case "finalize_domain_package", "close_domain_package",
		"finalize_topic_package", "finalize_temario_package", "finalize_syllabus_package",
		"completed_syllabus_package":
		return DomainWorkArtifactTypeFinalDomainPackageV0
	default:
		return DomainWorkArtifactTypeGenericWorkDeliveryV0
	}
}
