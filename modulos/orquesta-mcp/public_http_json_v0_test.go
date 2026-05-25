package orquestamcp

import (
	"errors"
	"strings"
	"testing"
)

func TestPublicMCPExecutorErrorMessageFromErrorV0ConservaCausaSinFiltrarSecretos(t *testing.T) {
	message := publicMCPExecutorErrorMessageFromErrorV0(
		"run_supervisor_error",
		errors.New("codex_worktree_verification: ack_files_mismatch path=/home/alberto/Trabajo/orquesta/x task=agent-ref-task-autoprogramming-001 token=sk-1234567890abcdef"),
	)

	if !strings.Contains(message, "run_supervisor_error") ||
		!strings.Contains(message, "codex_worktree_verification: ack_files_mismatch") {
		t.Fatalf("mensaje sin causa util: %q", message)
	}
	if !strings.Contains(message, "agent-ref-task-autoprogramming-001") {
		t.Fatalf("redactor rompio ref operativa: %q", message)
	}
	if strings.Contains(message, "/home/alberto") ||
		strings.Contains(message, "sk-1234567890abcdef") ||
		!strings.Contains(message, "<path>") ||
		!strings.Contains(message, "<redacted>") {
		t.Fatalf("mensaje sin sanitizar: %q", message)
	}
}
