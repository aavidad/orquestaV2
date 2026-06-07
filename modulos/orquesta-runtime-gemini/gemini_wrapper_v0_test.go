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
