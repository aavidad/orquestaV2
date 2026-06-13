package orquestaopesbridge

import (
	"strings"
	"testing"
)

func TestOPESFullTemarioJobTypeSequenceV0IncluyeCierreCompletoV0(t *testing.T) {
	sequence := OPESFullTemarioJobTypeSequenceV0()
	joined := strings.Join(sequence, ",")
	for _, want := range []string{
		"research_exam_precedents",
		"plan_temario",
		"update_topic_registry",
		"generate_question_bank",
		"review_codex",
		"review_gemini",
		"review_claude",
		"review_pair_codex_gemini",
		"review_pair_codex_claude",
		"review_pair_gemini_claude",
		"generate_audio_asset",
		"generate_tutor_assets",
		"generate_learning_games",
		"generate_html_site",
		"generate_help_manual_assets",
		"finalize_temario_package",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("secuencia no contiene %s: %s", want, joined)
		}
	}
	if got := OPESFinalTemarioJobTypeV0(); got != "finalize_temario_package" {
		t.Fatalf("final=%s", got)
	}
	if sequence[0] != "plan_temario" ||
		indexOfOPESSequenceForTestV0(sequence, "update_topic_registry") <
			indexOfOPESSequenceForTestV0(sequence, "plan_temario") ||
		sequence[len(sequence)-1] != "finalize_temario_package" {
		t.Fatalf("sequence=%v", sequence)
	}
	for _, optionalRework := range []string{
		"generate_agent_candidate_codex",
		"generate_agent_candidate_gemini",
		"generate_agent_candidate_claude",
		"vote_agent_candidates_codex",
		"vote_agent_candidates_gemini",
		"vote_agent_candidates_claude",
		"select_agent_candidate_director",
	} {
		if strings.Contains(joined, optionalRework) {
			t.Fatalf("rework opcional no debe ir en la secuencia principal %s: %v", optionalRework, sequence)
		}
	}
	if indexOfOPESSequenceForTestV0(sequence, "generate_tutor_assets") <
		indexOfOPESSequenceForTestV0(sequence, "generate_audio_asset") {
		t.Fatalf("audio debe generarse antes del tutor para integrar manifest accesible: %v", sequence)
	}
	if indexOfOPESSequenceForTestV0(sequence, "generate_html_site") <
		indexOfOPESSequenceForTestV0(sequence, "generate_learning_games") {
		t.Fatalf("html debe ir despues de tutor para integrar bots y juegos: %v", sequence)
	}
}

func indexOfOPESSequenceForTestV0(sequence []string, value string) int {
	for i, item := range sequence {
		if item == value {
			return i
		}
	}
	return -1
}
