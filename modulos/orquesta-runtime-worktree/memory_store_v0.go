package orquestaruntimeworktree

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type WorktreeSnapshotStorePortV0 interface {
	RecordWorktreeSnapshotV0(context.Context, WorktreeSnapshotV0) error
	LoadWorktreeSnapshotV0(context.Context, string) (WorktreeSnapshotV0, error)
}

type InMemoryWorktreeSnapshotStoreV0 struct {
	mu        sync.Mutex
	snapshots map[string]WorktreeSnapshotV0
}

func NewInMemoryWorktreeSnapshotStoreV0(
	snapshots ...WorktreeSnapshotV0,
) *InMemoryWorktreeSnapshotStoreV0 {
	store := &InMemoryWorktreeSnapshotStoreV0{snapshots: map[string]WorktreeSnapshotV0{}}
	for _, snapshot := range snapshots {
		_ = store.RecordWorktreeSnapshotV0(context.Background(), snapshot)
	}
	return store
}

func (store *InMemoryWorktreeSnapshotStoreV0) RecordWorktreeSnapshotV0(
	ctx context.Context,
	snapshot WorktreeSnapshotV0,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateWorktreeSnapshotForStoreV0(snapshot); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.snapshots[strings.TrimSpace(snapshot.SnapshotRef)] = cloneWorktreeSnapshotV0(snapshot)
	return nil
}

func (store *InMemoryWorktreeSnapshotStoreV0) LoadWorktreeSnapshotV0(
	ctx context.Context,
	snapshotRef string,
) (WorktreeSnapshotV0, error) {
	if err := ctx.Err(); err != nil {
		return WorktreeSnapshotV0{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	snapshot, ok := store.snapshots[strings.TrimSpace(snapshotRef)]
	if !ok {
		return WorktreeSnapshotV0{}, fmt.Errorf("worktree_snapshot_not_found")
	}
	return cloneWorktreeSnapshotV0(snapshot), nil
}

func validateWorktreeSnapshotForStoreV0(snapshot WorktreeSnapshotV0) error {
	if snapshot.SchemaVersion != WorktreeSnapshotSchemaVersionV0 {
		return fmt.Errorf("worktree_snapshot_schema_invalid")
	}
	if strings.TrimSpace(snapshot.SnapshotRef) == "" {
		return fmt.Errorf("worktree_snapshot_ref_required")
	}
	return nil
}

func cloneWorktreeSnapshotV0(snapshot WorktreeSnapshotV0) WorktreeSnapshotV0 {
	snapshot.Files = append([]WorktreeSnapshotFileV0(nil), snapshot.Files...)
	return snapshot
}
