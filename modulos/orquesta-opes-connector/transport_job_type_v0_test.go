package orquestaopesconnector

import "testing"

func TestOPESTransportJobTypeForWorkKindV0(t *testing.T) {
	for _, tc := range []struct {
		workKind string
		want     string
	}{
		{workKind: "update_topic_registry", want: "review_textual"},
		{workKind: "review_codex", want: "review_textual"},
		{workKind: "review_pair_codex_gemini", want: "review_textual"},
		{workKind: "review_director_consolidation", want: "review_textual"},
		{workKind: "generate_learning_games", want: "generate_tutor_assets"},
		{workKind: "finalize_temario_package", want: "generate_help_manual_assets"},
		{workKind: "research_exam_precedents", want: "research_exam_precedents"},
	} {
		t.Run(tc.workKind, func(t *testing.T) {
			if got := opesTransportJobTypeForWorkKindV0(tc.workKind); got != tc.want {
				t.Fatalf("transport=%s want=%s", got, tc.want)
			}
		})
	}
}
