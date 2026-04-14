package coordinacion

import "testing"

func TestBuildBranchNameMantieneBaseSinTaskID(t *testing.T) {
	got := buildBranchName("orquestador", "Gemini1", nil)
	if got != "orq-orquestador-gemini1" {
		t.Fatalf("branch base inesperada: %q", got)
	}
}

func TestBuildBranchNameUsaSufijoPlanoParaTaskID(t *testing.T) {
	taskID := int64(526)
	got := buildBranchName("orquestador", "Gemini1", &taskID)
	if got != "orq-orquestador-gemini1-t526" {
		t.Fatalf("branch task-specific inesperada: %q", got)
	}
	if containsSlashAfterBase(got) {
		t.Fatalf("la branch task-specific no deberia usar jerarquia git: %q", got)
	}
}

func containsSlashAfterBase(branch string) bool {
	for _, r := range branch {
		if r == '/' {
			return true
		}
	}
	return false
}
