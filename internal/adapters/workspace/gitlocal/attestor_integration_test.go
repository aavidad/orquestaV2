//go:build linux

package gitlocal

import (
	"context"
	"os"
	"testing"
)

func TestRestartedGitSnapshotStreamRejectsOriginalWorktreeSwapWithoutSubjectMutation(t *testing.T) {
	adapter, request, workspace := committedSnapshot(t)
	fresh, err := newTestAdapter(Config{
		Root: adapter.root, GitCommand: testGitExecutable(t), Locator: adapter.loc, Now: adapter.now,
		MaxSnapshotBytes: 1024 * 1024, MaxSnapshotEntries: 100,
	})
	gitTestNoError(t, err)
	t.Cleanup(func() { _ = fresh.Close() })
	displaced := workspace + ".displaced"
	gitTestNoError(t, os.Rename(workspace, displaced))
	gitTestNoError(t, os.Mkdir(workspace, 0o777))
	gitTestNoError(t, os.WriteFile(workspace+"/allowed.txt", []byte("attacker bytes\n"), 0o666))
	if _, err := fresh.OpenSnapshotStream(context.Background(), request); ErrorCodeOf(err) != CodeWorkspaceUnsafe {
		t.Fatalf("swapped binding stream err=%v", err)
	}
}
