package orquestaruntimegemini

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildGeminiWrapperScriptV0UsaPromptPorStdin(t *testing.T) {
	root := t.TempDir()
	script := BuildGeminiWrapperScriptV0(GeminiConnectorProfileV0{
		CommandPath:    filepath.Join(root, "gemini"),
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
		Model:          "gemini-2.5-pro",
		ApprovalMode:   "auto_edit",
		OutputFormat:   "json",
	})
	for _, want := range []string{
		"--model 'gemini-2.5-pro'",
		"--approval-mode 'auto_edit'",
		"--output-format 'json'",
		"--include-directories '" + filepath.Join(root, "runtime") + "'",
		"--prompt ''",
		"< '" + filepath.Join(root, "runtime", GeminiAgentPromptFileNameV0) + "'",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script no contiene %q: %s", want, script)
		}
	}
}

func TestBuildGeminiWrapperScriptV0DiagnosticaIneligibleTier(t *testing.T) {
	root := t.TempDir()
	script := BuildGeminiWrapperScriptV0(GeminiConnectorProfileV0{
		CommandPath:    filepath.Join(root, "gemini"),
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
	})
	for _, want := range []string{
		"IneligibleTierError",
		GeminiProviderDiagnosticFileNameV0,
		"orquesta_provider_diagnostic.v0",
		"provider_auth_or_tier_blocked",
		"update_gemini_cli",
		"migrate_to_supported_provider",
		"configure_valid_credentials_or_tier",
		"evidence-ref-gemini-ineligible-tier",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("wrapper sin diagnostico Gemini tier/auth: falta %q en %s", want, script)
		}
	}
}
