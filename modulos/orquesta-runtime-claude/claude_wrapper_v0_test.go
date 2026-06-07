package orquestaruntimeclaude

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildClaudeWrapperScriptV0UsaPrintYPromptPorStdin(t *testing.T) {
	root := t.TempDir()
	script := BuildClaudeWrapperScriptV0(ClaudeConnectorProfileV0{
		CommandPath:    filepath.Join(root, "claude"),
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
		Model:          "sonnet",
		PermissionMode: "dontAsk",
		OutputFormat:   "text",
		Effort:         "medium",
	})
	for _, want := range []string{
		"-p",
		"--model 'sonnet'",
		"--permission-mode 'dontAsk'",
		"--output-format 'text'",
		"--effort 'medium'",
		"agent_prompt.txt",
		"claude_stdout.log",
		"claude_stderr.log",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script no contiene %q:\n%s", want, script)
		}
	}
}
