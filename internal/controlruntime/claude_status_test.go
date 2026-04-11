package controlruntime

import "testing"

func TestParseClaudeStatusLiveExtraeCampos(t *testing.T) {
	raw := `status: turns=12 model=sonnet permission-mode="workspace-write" output-format=Text last-usage=1234 in/567 out config=/tmp/claude.json`
	got, err := ParseClaudeStatusLive(raw)
	if err != nil {
		t.Fatalf("ParseClaudeStatusLive: %v", err)
	}
	if got == nil {
		t.Fatalf("status nil")
	}
	if got.Model != "sonnet" {
		t.Fatalf("model inesperado: %+v", got)
	}
	if got.LastUsageIn == nil || *got.LastUsageIn != 1234 {
		t.Fatalf("input inesperado: %+v", got)
	}
	if got.LastUsageOut == nil || *got.LastUsageOut != 567 {
		t.Fatalf("output inesperado: %+v", got)
	}
}
