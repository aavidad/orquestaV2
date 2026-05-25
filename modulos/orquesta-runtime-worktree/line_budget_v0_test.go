package orquestaruntimeworktree

import (
	"context"
	"strings"
	"testing"
)

func TestVerifyWorktreeWriteSetV0PresupuestoLineasGoEstricto(t *testing.T) {
	t.Run("bloquea fichero Go nuevo sobre limite", func(t *testing.T) {
		root := t.TempDir()
		baseline := captureWorktreeSnapshotForTestV0(t, root, nil)
		writeWorktreeFileForTestV0(t, root, "internal/app/app.go", strings.Repeat("x\n", 301))

		result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
			Baseline:           baseline,
			ProjectWorkDir:     root,
			WriteSet:           []string{"internal/app"},
			StrictGoLineBudget: true,
		})

		if len(issues) != 1 || issues[0].Code != WorktreeIssueGoLineBudgetV0 {
			t.Fatalf("result=%+v issues=%+v", result, issues)
		}
		if len(result.GoLineBudgetViolations) != 1 ||
			result.GoLineBudgetViolations[0].Reason != "file_over_limit" {
			t.Fatalf("violations=%+v", result.GoLineBudgetViolations)
		}
	})

	t.Run("bloquea crecimiento de deuda historica", func(t *testing.T) {
		root := t.TempDir()
		writeWorktreeFileForTestV0(t, root, "internal/app/app.go", strings.Repeat("x\n", 320))
		baseline := captureWorktreeSnapshotForTestV0(t, root, nil)
		writeWorktreeFileForTestV0(t, root, "internal/app/app.go", strings.Repeat("x\n", 321))

		result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
			Baseline:           baseline,
			ProjectWorkDir:     root,
			WriteSet:           []string{"internal/app"},
			StrictGoLineBudget: true,
		})

		if len(issues) != 1 || issues[0].Code != WorktreeIssueGoLineBudgetV0 {
			t.Fatalf("result=%+v issues=%+v", result, issues)
		}
		if result.GoLineBudgetViolations[0].Reason != "baseline_growth" {
			t.Fatalf("violations=%+v", result.GoLineBudgetViolations)
		}
	})

	t.Run("acepta deuda historica sin crecimiento", func(t *testing.T) {
		root := t.TempDir()
		writeWorktreeFileForTestV0(t, root, "internal/app/app.go", strings.Repeat("x\n", 320))
		baseline := captureWorktreeSnapshotForTestV0(t, root, nil)
		writeWorktreeFileForTestV0(t, root, "internal/app/app.go", strings.Repeat("y\n", 320))

		result, issues := VerifyWorktreeWriteSetV0(context.Background(), WorktreeVerifyRequestV0{
			Baseline:           baseline,
			ProjectWorkDir:     root,
			WriteSet:           []string{"internal/app"},
			StrictGoLineBudget: true,
		})

		if len(issues) > 0 || !result.OK {
			t.Fatalf("result=%+v issues=%+v", result, issues)
		}
	})
}
