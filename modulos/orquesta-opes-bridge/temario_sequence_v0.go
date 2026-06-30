package orquestaopesbridge

func OPESFullTemarioJobTypeSequenceV0() []string {
	return []string{
		"plan_temario",
		"update_topic_registry",
		"research_exam_precedents",
		"draft_content_block",
		"generate_visual_asset",
		"generate_question_bank",
		"review_legal",
		"review_pedagogical",
		"review_quality",
		"review_codex",
		"review_gemini",
		"review_claude",
		"review_pair_codex_gemini",
		"review_pair_codex_claude",
		"review_pair_gemini_claude",
		"review_director_consolidation",
		"validate_topic",
		"assemble_topic",
		"generate_audio_asset",
		"generate_tutor_assets",
		"generate_learning_games",
		"visual_asset_reuse",
		"generate_html_site",
		"generate_help_manual_assets",
		"finalize_temario_package",
	}
}

func OPESFinalTemarioJobTypeV0() string {
	sequence := OPESFullTemarioJobTypeSequenceV0()
	if len(sequence) == 0 {
		return ""
	}
	return sequence[len(sequence)-1]
}
