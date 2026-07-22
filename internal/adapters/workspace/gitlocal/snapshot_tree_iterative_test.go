package gitlocal

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"orquesta/internal/ports"
)

func TestSnapshotTreeTraversalIsIterative(t *testing.T) {
	adapter, prepare := testAdapterAndPrepare(t)
	prepare.WriteSet = []string{"deep"}
	prepare.WriteSetDigest = ports.WorkspaceWriteSetDigest(prepare.WriteSet)
	prepared, err := adapter.Prepare(context.Background(), prepare)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(append([]string{adapter.workspacePath(prepare.WorkspaceRef), "deep"},
		append(slices.Repeat([]string{"d"}, 256), "leaf.txt")...)...)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("deep\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	change, err := adapter.Commit(context.Background(), testCommit(t, prepare, prepared))
	if err != nil {
		t.Fatal(err)
	}
	if err := drainSnapshot(adapter, gitSnapshotRequest(prepare, prepared, change)); err != nil {
		t.Fatalf("deep iterative snapshot: %v", err)
	}
}
