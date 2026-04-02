package cmd

import (
	"reflect"
	"testing"
)

func TestParseGitStatusPorcelainSummaryIncluyeMuestraYOverflow(t *testing.T) {
	raw := " M cmd/api.go\nA  cmd/serve.go\nR  old.go -> new.go\n?? docs/tmp.md\n?? scratch.txt\n?? extra.log\n"
	dirty, summary, files, overflow := parseGitStatusPorcelainSummary(raw)
	if !dirty {
		t.Fatalf("debería marcar dirty")
	}
	if summary != "3 tracked · 3 untracked" {
		t.Fatalf("summary inesperado: %q", summary)
	}
	wantFiles := []string{"cmd/api.go", "cmd/serve.go", "old.go -> new.go", "docs/tmp.md", "scratch.txt"}
	if !reflect.DeepEqual(files, wantFiles) {
		t.Fatalf("files inesperados: %#v", files)
	}
	if overflow != 1 {
		t.Fatalf("overflow inesperado: %d", overflow)
	}
}

func TestParseGitStatusPorcelainSummaryLimpio(t *testing.T) {
	dirty, summary, files, overflow := parseGitStatusPorcelainSummary("")
	if dirty || summary != "" || len(files) != 0 || overflow != 0 {
		t.Fatalf("resultado limpio inesperado: dirty=%v summary=%q files=%v overflow=%d", dirty, summary, files, overflow)
	}
}
