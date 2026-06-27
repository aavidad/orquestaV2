package orquestaruntimeworktree

import (
	"path"
	"sort"
	"strings"
)

type worktreeLocalArtifactMatchV0 struct {
	Category   string
	ReasonCode string
}

func IsWorktreeLocalArtifactPathV0(value string) bool {
	_, ok := worktreeLocalArtifactPolicyMatchV0(value)
	return ok
}

func worktreeLocalArtifactPolicyMatchV0(value string) (worktreeLocalArtifactMatchV0, bool) {
	pathValue, ok := normalizeWorktreeRelPathV0(value, false)
	if !ok {
		return worktreeLocalArtifactMatchV0{
			Category:   "invalid_local_path",
			ReasonCode: "local_artifact_path_invalid",
		}, true
	}
	segment, _, _ := strings.Cut(pathValue, "/")
	base := path.Base(pathValue)
	lowerSegment := strings.ToLower(segment)
	lowerBase := strings.ToLower(base)
	if lowerSegment == ".orquesta-server" {
		return worktreeLocalArtifactMatchV0{"server_state_dir", "local_server_state_artifact"}, true
	}
	if lowerSegment == ".git" {
		return worktreeLocalArtifactMatchV0{"vcs_state_dir", "local_vcs_state_artifact"}, true
	}
	if lowerSegment == ".orquesta-smoke-work" {
		return worktreeLocalArtifactMatchV0{"smoke_work_dir", "local_smoke_work_artifact"}, true
	}
	if lowerSegment == "orquesta.env" {
		return worktreeLocalArtifactMatchV0{"local_operator_config", "local_operator_config_artifact"}, true
	}
	if lowerSegment == "orquesta.db" || strings.HasPrefix(lowerSegment, "orquesta.db-") {
		return worktreeLocalArtifactMatchV0{"local_database_state", "local_database_state_artifact"}, true
	}
	if lowerSegment == ".orquesta-inbox.md" {
		return worktreeLocalArtifactMatchV0{"local_operator_notes", "local_operator_notes_artifact"}, true
	}
	if lowerSegment == "certs" || lowerBase == ".ssl-key.log" {
		return worktreeLocalArtifactMatchV0{"local_secret_diagnostics", "local_secret_diagnostic_artifact"}, true
	}
	if lowerSegment == "logs" || lowerSegment == ".orquesta-logs" || strings.HasSuffix(lowerBase, ".log") {
		return worktreeLocalArtifactMatchV0{"local_logs", "local_log_artifact"}, true
	}
	if strings.HasSuffix(lowerBase, ".test") {
		return worktreeLocalArtifactMatchV0{"local_build_artifact", "local_test_binary_artifact"}, true
	}
	if worktreeLocalScratchDirV0(lowerSegment) {
		return worktreeLocalArtifactMatchV0{"local_scratch_dir", "local_scratch_artifact"}, true
	}
	if worktreeRuntimeControlDirV0(lowerSegment) {
		return worktreeLocalArtifactMatchV0{"runtime_control_dir", "local_runtime_control_artifact"}, true
	}
	if worktreeDefaultControlFileNamesV0[base] {
		return worktreeLocalArtifactMatchV0{"control_file", "local_control_file_artifact"}, true
	}
	return worktreeLocalArtifactMatchV0{}, false
}

func worktreeRuntimeControlDirV0(segment string) bool {
	for _, prefix := range worktreeDefaultControlPrefixesV0 {
		if segment == prefix || strings.HasPrefix(segment, prefix+"-") {
			return true
		}
	}
	return false
}

func worktreeLocalScratchDirV0(segment string) bool {
	switch segment {
	case ".orquesta-runs", ".orquesta-worktrees", ".agents", ".codex",
		".codex-docker-home", ".codex-sandbox-workspace", "tmp", ".cache",
		"backups":
		return true
	default:
		return strings.HasPrefix(segment, ".orquesta-local-")
	}
}

func worktreeLocalArtifactReceiptsV0(paths []string) []WorktreeLocalArtifactExclusionReceiptV0 {
	counts := map[worktreeLocalArtifactMatchV0]int{}
	for _, item := range paths {
		if match, ok := worktreeLocalArtifactPolicyMatchV0(item); ok {
			counts[match]++
		}
	}
	receipts := make([]WorktreeLocalArtifactExclusionReceiptV0, 0, len(counts))
	for match, count := range counts {
		receipts = append(receipts, WorktreeLocalArtifactExclusionReceiptV0{
			Category:   match.Category,
			ReasonCode: match.ReasonCode,
			Count:      count,
		})
	}
	sort.Slice(receipts, func(i, j int) bool {
		if receipts[i].Category == receipts[j].Category {
			return receipts[i].ReasonCode < receipts[j].ReasonCode
		}
		return receipts[i].Category < receipts[j].Category
	})
	return receipts
}

func mergeWorktreeLocalArtifactReceiptsV0(
	values ...[]WorktreeLocalArtifactExclusionReceiptV0,
) []WorktreeLocalArtifactExclusionReceiptV0 {
	counts := map[worktreeLocalArtifactMatchV0]int{}
	for _, receipts := range values {
		for _, receipt := range receipts {
			match := worktreeLocalArtifactMatchV0{
				Category:   strings.TrimSpace(receipt.Category),
				ReasonCode: strings.TrimSpace(receipt.ReasonCode),
			}
			if match.Category == "" || match.ReasonCode == "" || receipt.Count <= 0 {
				continue
			}
			counts[match] += receipt.Count
		}
	}
	receipts := make([]WorktreeLocalArtifactExclusionReceiptV0, 0, len(counts))
	for match, count := range counts {
		receipts = append(receipts, WorktreeLocalArtifactExclusionReceiptV0{
			Category:   match.Category,
			ReasonCode: match.ReasonCode,
			Count:      count,
		})
	}
	sort.Slice(receipts, func(i, j int) bool {
		if receipts[i].Category == receipts[j].Category {
			return receipts[i].ReasonCode < receipts[j].ReasonCode
		}
		return receipts[i].Category < receipts[j].Category
	})
	return receipts
}
