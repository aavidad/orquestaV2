package orquestastatefile

import (
	"context"
	"reflect"
	"strings"
	"testing"

	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func TestStoreV0WorktreeSnapshotStoreV0RecuperaReplayYRechazaConflictoTrasReinicio(t *testing.T) {
	ctx := context.Background()
	rootDir := t.TempDir()
	snapshot := validWorktreeSnapshotV0()
	store := mustStoreV0(t, rootDir)
	if err := store.RecordWorktreeSnapshotV0(ctx, snapshot); err != nil {
		t.Fatalf("record snapshot: %v", err)
	}
	if err := store.RecordWorktreeSnapshotV0(ctx, snapshot); err != nil {
		t.Fatalf("replay snapshot: %v", err)
	}

	recreated := mustStoreV0(t, rootDir)
	loaded, err := recreated.LoadWorktreeSnapshotV0(ctx, snapshot.SnapshotRef)
	if err != nil {
		t.Fatalf("load snapshot after restart: %v", err)
	}
	if !reflect.DeepEqual(loaded, snapshot) {
		t.Fatalf("loaded snapshot = %#v, want %#v", loaded, snapshot)
	}

	conflicting := snapshot
	conflicting.Files = append([]orquestaruntimeworktree.WorktreeSnapshotFileV0(nil), snapshot.Files...)
	conflicting.Files[0].Digest = "different-digest"
	if err := recreated.RecordWorktreeSnapshotV0(ctx, conflicting); err == nil || !strings.Contains(err.Error(), "contenido divergente") {
		t.Fatalf("conflicting record error = %v, want divergent-content error", err)
	}
}

func validWorktreeSnapshotV0() orquestaruntimeworktree.WorktreeSnapshotV0 {
	return orquestaruntimeworktree.WorktreeSnapshotV0{
		SchemaVersion: orquestaruntimeworktree.WorktreeSnapshotSchemaVersionV0,
		SnapshotRef:   "worktree-snapshot-state-file-001",
		Files: []orquestaruntimeworktree.WorktreeSnapshotFileV0{{
			Path:      "modulos/orquesta-state-file/worktree_snapshot_store_v0.go",
			Digest:    "digest-state-file-001",
			Size:      42,
			LineCount: 3,
		}},
	}
}
