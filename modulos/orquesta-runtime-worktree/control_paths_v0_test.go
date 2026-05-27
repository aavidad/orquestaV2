package orquestaruntimeworktree

import (
	"context"
	"path/filepath"
	"testing"
)

func TestCaptureWorktreeSnapshotV0ExcluyeControlFilesPorDefectoV0(t *testing.T) {
	root := t.TempDir()
	writeWorktreeFileForTestV0(t, root, "README.md", "v1")
	writeWorktreeFileForTestV0(t, root, ".orquesta-runtime/run/agent_packet.json", "{}")
	writeWorktreeFileForTestV0(t, root, ".orquesta-codex-runtime/run/codex_stderr.log", "log")
	writeWorktreeFileForTestV0(t, root, ".orquesta-local-runtime-20260525/run/debug.log", "log")
	writeWorktreeFileForTestV0(t, root, ".orquesta-runtime-20260525/run/debug.log", "log")
	writeWorktreeFileForTestV0(t, root, ".orquesta-server/state.json", "{}")
	writeWorktreeFileForTestV0(t, root, ".orquesta-smoke-work/run.log", "log")
	writeWorktreeFileForTestV0(t, root, ".orquesta-logs/daemon.log", "log")
	writeWorktreeFileForTestV0(t, root, "logs/server.log", "log")
	writeWorktreeFileForTestV0(t, root, ".ssl-key.log", "secret")
	writeWorktreeFileForTestV0(t, root, "orquesta.env", "SECRET=value")
	writeWorktreeFileForTestV0(t, root, "orquesta.db", "db")
	writeWorktreeFileForTestV0(t, root, ".orquesta-inbox.md", "local")
	writeWorktreeFileForTestV0(t, root, "agent_ack.json", "{}")

	snapshot, issues := CaptureWorktreeSnapshotV0(context.Background(), WorktreeSnapshotRequestV0{
		SnapshotRef:    "snapshot-ref-control-defaults",
		ProjectWorkDir: root,
	})
	if len(issues) > 0 {
		t.Fatalf("issues=%+v", issues)
	}
	if len(snapshot.Files) != 1 || snapshot.Files[0].Path != "README.md" {
		t.Fatalf("files=%+v", snapshot.Files)
	}
	for _, want := range []string{
		"runtime_control_dir",
		"server_state_dir",
		"smoke_work_dir",
		"local_logs",
		"local_secret_diagnostics",
		"local_operator_config",
		"local_database_state",
		"local_operator_notes",
	} {
		if !worktreeReceiptCategoryForTestV0(snapshot.ExclusionReceipts, want) {
			t.Fatalf("receipts sin %s: %+v", want, snapshot.ExclusionReceipts)
		}
	}
}

func TestVerifyWorktreeWriteSetV0IgnoraControlConWriteSetRaizYRechazaAckControlV0(t *testing.T) {
	root := t.TempDir()
	baseline := captureWorktreeSnapshotForTestV0(t, root, nil)

	writeWorktreeFileForTestV0(t, root, "README.md", "v1")
	writeWorktreeFileForTestV0(t, root, ".orquesta-runtime/run/agent_ack.json", "{}")
	writeWorktreeFileForTestV0(t, root, ".orquesta-local-runtime-20260525/run/agent_ack.json", "{}")
	result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
		Baseline:       baseline,
		ProjectWorkDir: root,
		WriteSet:       []string{"."},
		AckFiles:       []string{"README.md"},
	})
	if len(issues) > 0 || !result.OK || len(result.ChangedPaths) != 1 {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if !worktreeReceiptCategoryForTestV0(result.ExclusionReceipts, "runtime_control_dir") {
		t.Fatalf("receipts=%+v", result.ExclusionReceipts)
	}

	_, issues = VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
		Baseline:       baseline,
		ProjectWorkDir: root,
		WriteSet:       []string{"."},
		AckFiles:       []string{filepath.ToSlash(".orquesta-local-runtime-20260525/run/agent_ack.json")},
	})
	if len(issues) != 1 || issues[0].Code != WorktreeIssueControlPathV0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func worktreeReceiptCategoryForTestV0(
	receipts []WorktreeLocalArtifactExclusionReceiptV0,
	category string,
) bool {
	for _, receipt := range receipts {
		if receipt.Category == category && receipt.Count > 0 && receipt.ReasonCode != "" {
			return true
		}
	}
	return false
}
