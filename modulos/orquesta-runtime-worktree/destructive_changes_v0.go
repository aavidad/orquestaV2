package orquestaruntimeworktree

import "strconv"

const (
	WorktreeDestructiveRemovedV0            = "removed"
	WorktreeDestructiveTruncatedV0          = "truncated"
	WorktreeDestructiveRenamedOrMovedV0     = "renamed_or_moved"
	WorktreeDestructiveReplacedLargeDeltaV0 = "replaced_large_delta"
)

func classifyWorktreeDestructiveChangesV0(
	base map[string]WorktreeSnapshotFileV0,
	now map[string]WorktreeSnapshotFileV0,
) []WorktreeDestructiveChangeV0 {
	var changes []WorktreeDestructiveChangeV0
	renamed := worktreeRenameCandidatesV0(base, now)
	for oldPath, newPath := range renamed {
		changes = append(changes, WorktreeDestructiveChangeV0{
			Kind:         WorktreeDestructiveRenamedOrMovedV0,
			PreviousPath: oldPath,
			CurrentPath:  newPath,
		})
	}
	for path, baseFile := range base {
		currentFile, exists := now[path]
		if !exists {
			if _, ok := renamed[path]; ok {
				continue
			}
			changes = append(changes, WorktreeDestructiveChangeV0{
				Kind:          WorktreeDestructiveRemovedV0,
				Path:          path,
				BaselineSize:  baseFile.Size,
				BaselineLines: baseFile.LineCount,
			})
			continue
		}
		if baseFile.Digest == currentFile.Digest && baseFile.Size == currentFile.Size {
			continue
		}
		if worktreeStrongTruncateV0(baseFile, currentFile) {
			changes = append(changes, WorktreeDestructiveChangeV0{
				Kind:          WorktreeDestructiveTruncatedV0,
				Path:          path,
				BaselineSize:  baseFile.Size,
				CurrentSize:   currentFile.Size,
				BaselineLines: baseFile.LineCount,
				CurrentLines:  currentFile.LineCount,
			})
			continue
		}
		if worktreeLargeDeltaReplacementV0(baseFile, currentFile) {
			changes = append(changes, WorktreeDestructiveChangeV0{
				Kind:          WorktreeDestructiveReplacedLargeDeltaV0,
				Path:          path,
				BaselineSize:  baseFile.Size,
				CurrentSize:   currentFile.Size,
				BaselineLines: baseFile.LineCount,
				CurrentLines:  currentFile.LineCount,
			})
		}
	}
	return changes
}

func worktreeRenameCandidatesV0(
	base map[string]WorktreeSnapshotFileV0,
	now map[string]WorktreeSnapshotFileV0,
) map[string]string {
	addedByDigest := map[string]string{}
	for path, file := range now {
		if _, existed := base[path]; !existed {
			addedByDigest[file.Digest+"|"+strconv.FormatInt(file.Size, 10)] = path
		}
	}
	out := map[string]string{}
	for path, file := range base {
		if _, exists := now[path]; exists {
			continue
		}
		if added := addedByDigest[file.Digest+"|"+strconv.FormatInt(file.Size, 10)]; added != "" {
			out[path] = added
		}
	}
	return out
}

func worktreeStrongTruncateV0(baseFile, currentFile WorktreeSnapshotFileV0) bool {
	if baseFile.Size >= 64 && currentFile.Size*2 <= baseFile.Size {
		return true
	}
	return baseFile.LineCount >= 10 &&
		currentFile.LineCount > 0 &&
		currentFile.LineCount*2 <= baseFile.LineCount
}

func worktreeLargeDeltaReplacementV0(baseFile, currentFile WorktreeSnapshotFileV0) bool {
	if baseFile.Size < 4096 || currentFile.Size < 4096 {
		return false
	}
	maxSize := maxWorktreeInt64V0(baseFile.Size, currentFile.Size)
	minSize := minWorktreeInt64V0(baseFile.Size, currentFile.Size)
	if minSize*5 <= maxSize*3 {
		return true
	}
	if baseFile.LineCount >= 120 && currentFile.LineCount >= 120 {
		delta := baseFile.LineCount - currentFile.LineCount
		if delta < 0 {
			delta = -delta
		}
		return delta >= 120
	}
	return false
}

func worktreeDestructivePathsByKindV0(
	changes []WorktreeDestructiveChangeV0,
	kind string,
) []string {
	var out []string
	for _, change := range changes {
		if change.Kind != kind {
			continue
		}
		if change.Path != "" {
			out = append(out, change.Path)
			continue
		}
		if change.PreviousPath != "" && change.CurrentPath != "" {
			out = append(out, change.PreviousPath+" -> "+change.CurrentPath)
		}
	}
	return compactWorktreeStringsV0(out)
}

func minWorktreeInt64V0(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxWorktreeInt64V0(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
