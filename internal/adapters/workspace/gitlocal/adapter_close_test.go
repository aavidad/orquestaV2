package gitlocal

import (
	"context"
	"testing"

	"orquesta/internal/ports"
)

func TestAdapterCloseMakesEveryNewOperationUnavailable(t *testing.T) {
	adapter := pinnedTestAdapter(t, testGitExecutable(t))
	gitTestNoError(t, adapter.Close())
	ctx := context.Background()
	operations := map[string]func() error{
		"prepare": func() error { _, err := adapter.Prepare(ctx, ports.WorkspacePrepareRequest{}); return err },
		"inspect": func() error { _, err := adapter.Inspect(ctx, ports.WorkspaceInspectRequest{}); return err },
		"resolve_execution": func() error {
			_, err := adapter.ResolveExecutionWorkspace(ctx, ports.ExecutionWorkspaceRef{})
			return err
		},
		"commit":    func() error { _, err := adapter.Commit(ctx, ports.CommitRequest{}); return err },
		"release":   func() error { _, err := adapter.Release(ctx, ports.WorkspaceReleaseRequest{}); return err },
		"preview":   func() error { _, err := adapter.PreviewIntegration(ctx, ports.IntegrationPreviewRequest{}); return err },
		"integrate": func() error { _, err := adapter.Integrate(ctx, ports.IntegrationRequest{}); return err },
		"stream": func() error {
			_, err := adapter.OpenSnapshotStream(ctx, ports.SnapshotVerificationRequest{})
			return err
		},
	}
	for name, operation := range operations {
		if err := operation(); ErrorCodeOf(err) != CodeUnavailable {
			t.Errorf("%s after Close err=%v", name, err)
		}
	}
}
