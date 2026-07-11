package orquestastatefile

import (
	"context"
	"reflect"

	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

type worktreeSnapshotDocumentV0 struct {
	SchemaVersion string                                     `json:"schema_version"`
	SnapshotRef   string                                     `json:"snapshot_ref"`
	Snapshot      orquestaruntimeworktree.WorktreeSnapshotV0 `json:"snapshot"`
}

func (store *StoreV0) RecordWorktreeSnapshotV0(
	ctx context.Context,
	snapshot orquestaruntimeworktree.WorktreeSnapshotV0,
) error {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateWorktreeSnapshotV0(snapshot); err != nil {
		return err
	}
	snapshot.SnapshotRef = normalizeRefV0(snapshot.SnapshotRef)
	store.mu.Lock()
	defer store.mu.Unlock()
	path := store.worktreeSnapshotPathV0(snapshot.SnapshotRef)
	return withProcessFileLockV0(ctx, path+".lock", func() error {
		existing, ok, err := readJSONFileV0[worktreeSnapshotDocumentV0](path)
		if err != nil {
			return err
		}
		if ok {
			loaded, err := validateWorktreeSnapshotDocumentV0(existing, snapshot.SnapshotRef)
			if err != nil {
				return err
			}
			if worktreeSnapshotsSemanticallyEqualV0(loaded, snapshot) {
				return nil
			}
			return storeErrorV0("worktree_snapshot", "snapshot_ref existente con contenido divergente")
		}
		return writeJSONAtomicV0(path, worktreeSnapshotDocumentV0{
			SchemaVersion: worktreeSnapshotDocumentSchemaV0,
			SnapshotRef:   snapshot.SnapshotRef,
			Snapshot:      snapshot,
		})
	})
}

func worktreeSnapshotsSemanticallyEqualV0(
	left orquestaruntimeworktree.WorktreeSnapshotV0,
	right orquestaruntimeworktree.WorktreeSnapshotV0,
) bool {
	if len(left.OmittedPaths) == 0 {
		left.OmittedPaths = nil
	}
	if len(right.OmittedPaths) == 0 {
		right.OmittedPaths = nil
	}
	if len(left.ExclusionReceipts) == 0 {
		left.ExclusionReceipts = nil
	}
	if len(right.ExclusionReceipts) == 0 {
		right.ExclusionReceipts = nil
	}
	return reflect.DeepEqual(left, right)
}

func (store *StoreV0) LoadWorktreeSnapshotV0(
	ctx context.Context,
	snapshotRef string,
) (orquestaruntimeworktree.WorktreeSnapshotV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestaruntimeworktree.WorktreeSnapshotV0{}, err
	}
	snapshotRef = normalizeRefV0(snapshotRef)
	if snapshotRef == "" {
		return orquestaruntimeworktree.WorktreeSnapshotV0{}, invalidErrorV0("worktree_snapshot.snapshot_ref", "snapshot_ref requerido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	path := store.worktreeSnapshotPathV0(snapshotRef)
	var document worktreeSnapshotDocumentV0
	var ok bool
	err := withProcessFileLockV0(ctx, path+".lock", func() error {
		var readErr error
		document, ok, readErr = readJSONFileV0[worktreeSnapshotDocumentV0](path)
		return readErr
	})
	if err != nil {
		return orquestaruntimeworktree.WorktreeSnapshotV0{}, err
	}
	if !ok {
		return orquestaruntimeworktree.WorktreeSnapshotV0{}, storeErrorV0("worktree_snapshot", "snapshot no encontrado")
	}
	return validateWorktreeSnapshotDocumentV0(document, snapshotRef)
}

func validateWorktreeSnapshotDocumentV0(
	document worktreeSnapshotDocumentV0,
	expectedSnapshotRef string,
) (orquestaruntimeworktree.WorktreeSnapshotV0, error) {
	if document.SchemaVersion != worktreeSnapshotDocumentSchemaV0 {
		return orquestaruntimeworktree.WorktreeSnapshotV0{}, storeErrorV0("worktree_snapshot.schema_version", "schema_version invalida")
	}
	if document.SnapshotRef != expectedSnapshotRef {
		return orquestaruntimeworktree.WorktreeSnapshotV0{}, storeErrorV0("worktree_snapshot.ref", "snapshot_ref inconsistente")
	}
	if err := validateWorktreeSnapshotV0(document.Snapshot); err != nil {
		return orquestaruntimeworktree.WorktreeSnapshotV0{}, err
	}
	document.Snapshot.SnapshotRef = normalizeRefV0(document.Snapshot.SnapshotRef)
	if document.Snapshot.SnapshotRef != expectedSnapshotRef {
		return orquestaruntimeworktree.WorktreeSnapshotV0{}, storeErrorV0("worktree_snapshot.snapshot_ref", "ref interna inconsistente")
	}
	return document.Snapshot, nil
}

func validateWorktreeSnapshotV0(snapshot orquestaruntimeworktree.WorktreeSnapshotV0) error {
	if snapshot.SchemaVersion != orquestaruntimeworktree.WorktreeSnapshotSchemaVersionV0 {
		return invalidErrorV0("worktree_snapshot.schema_version", "schema_version invalida")
	}
	if normalizeRefV0(snapshot.SnapshotRef) == "" {
		return invalidErrorV0("worktree_snapshot.snapshot_ref", "snapshot_ref requerido")
	}
	return nil
}
