package orquestaruntimeworktree

import "strconv"

const (
	WorktreeDestructiveRemovedV0            = "removed"
	WorktreeDestructiveTruncatedV0          = "truncated"
	WorktreeDestructiveRenamedOrMovedV0     = "renamed_or_moved"
	WorktreeDestructiveReplacedLargeDeltaV0 = "replaced_large_delta"
)

// Umbrales de clasificacion de cambios destructivos. Son ADVISORY: ya no
// bloquean entregas, solo etiquetan el cambio como evidencia (`gate-issue:*`)
// para review del Director. Se documentan aqui como punto unico para afinar
// sensibilidad sin buscarlos por el codigo. Ver
// docs/estado_actual_2026-05-17.md:39-64 (no cortar trabajo recuperable).
const (
	// Tamano/lineas minimos para considerar un fichero "sustancial".
	worktreeTruncateMinBytesV0 int64 = 64
	worktreeTruncateMinLinesV0       = 10
	// Un fichero se marca truncado si su tamano/lineas cae a la mitad o menos
	// (current*Factor <= base). Untyped para servir a Size (int64) y LineCount (int).
	worktreeTruncateShrinkFactorV0 = 2
	// Tamano minimo para evaluar delta grande (ruido por debajo no cuenta).
	worktreeLargeDeltaMinBytesV0 int64 = 4096
	// Delta grande si el menor es <= 60% del mayor (min*5 <= max*3).
	worktreeLargeDeltaRatioNumV0 int64 = 3
	worktreeLargeDeltaRatioDenV0 int64 = 5
	// O si la diferencia de lineas supera este umbral en ficheros grandes.
	worktreeLargeDeltaLineFloorV0 = 120
	worktreeLargeDeltaLineDeltaV0 = 120
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
	if baseFile.Size >= worktreeTruncateMinBytesV0 &&
		currentFile.Size*worktreeTruncateShrinkFactorV0 <= baseFile.Size {
		return true
	}
	return baseFile.LineCount >= worktreeTruncateMinLinesV0 &&
		currentFile.LineCount > 0 &&
		currentFile.LineCount*worktreeTruncateShrinkFactorV0 <= baseFile.LineCount
}

func worktreeLargeDeltaReplacementV0(baseFile, currentFile WorktreeSnapshotFileV0) bool {
	if baseFile.Size < worktreeLargeDeltaMinBytesV0 || currentFile.Size < worktreeLargeDeltaMinBytesV0 {
		return false
	}
	maxSize := maxWorktreeInt64V0(baseFile.Size, currentFile.Size)
	minSize := minWorktreeInt64V0(baseFile.Size, currentFile.Size)
	if minSize*worktreeLargeDeltaRatioDenV0 <= maxSize*worktreeLargeDeltaRatioNumV0 {
		return true
	}
	if baseFile.LineCount >= worktreeLargeDeltaLineFloorV0 && currentFile.LineCount >= worktreeLargeDeltaLineFloorV0 {
		delta := baseFile.LineCount - currentFile.LineCount
		if delta < 0 {
			delta = -delta
		}
		return delta >= worktreeLargeDeltaLineDeltaV0
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
