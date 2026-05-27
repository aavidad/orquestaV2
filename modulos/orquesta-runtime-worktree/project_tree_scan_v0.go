package orquestaruntimeworktree

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

const (
	ProjectTreeScanModeTargetV0 = "target"
	ProjectTreeScanModeGlobV0   = "glob"
	ProjectTreeScanModeDirV0    = "dir"

	ProjectTreeScanReasonFoundV0           = "found"
	ProjectTreeScanReasonNotFoundV0        = "not_found"
	ProjectTreeScanReasonInvalidRequestV0  = "invalid_request"
	ProjectTreeScanReasonBudgetExhaustedV0 = "scan_budget_exhausted"
	ProjectTreeScanReasonCancelledV0       = "scan_cancelled"
	ProjectTreeScanReasonFilesystemV0      = "filesystem_error"
)

const (
	defaultProjectTreeScanMaxEntriesV0   = 2000
	defaultProjectTreeScanMaxDepthV0     = 24
	defaultProjectTreeScanMaxFileBytesV0 = 10 * 1024 * 1024
)

type ProjectTreeScanRequestV0 struct {
	ProjectRoot    string   `json:"project_root"`
	Target         string   `json:"target"`
	Mode           string   `json:"mode,omitempty"`
	MaxEntries     int      `json:"max_entries,omitempty"`
	MaxDepth       int      `json:"max_depth,omitempty"`
	MaxFileBytes   int64    `json:"max_file_bytes,omitempty"`
	MaxResults     int      `json:"max_results,omitempty"`
	IgnorePrefixes []string `json:"ignore_prefixes,omitempty"`
}

type ProjectTreeScanResultV0 struct {
	Found           bool     `json:"found"`
	ReasonCode      string   `json:"reason_code"`
	EntriesVisited  int      `json:"entries_visited,omitempty"`
	SkippedEntries  int      `json:"skipped_entries,omitempty"`
	BudgetExhausted bool     `json:"budget_exhausted,omitempty"`
	MatchedPath     string   `json:"matched_path,omitempty"`
	MatchedPaths    []string `json:"matched_paths,omitempty"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
}

func ProjectTreeScanHasFileV0(ctx context.Context, request ProjectTreeScanRequestV0) ProjectTreeScanResultV0 {
	if request.MaxResults <= 0 {
		request.MaxResults = 1
	}
	return projectTreeScanV0(ctx, request)
}

func ProjectTreeScanFilesV0(ctx context.Context, request ProjectTreeScanRequestV0) ProjectTreeScanResultV0 {
	if request.MaxResults <= 0 {
		request.MaxResults = 256
	}
	return projectTreeScanV0(ctx, request)
}

func projectTreeScanV0(ctx context.Context, request ProjectTreeScanRequestV0) ProjectTreeScanResultV0 {
	policy := projectTreeScanPolicyFromRequestV0(request)
	target, ok := projectTreeScanTargetV0(request.Target)
	if !ok || policy.root == "" {
		return projectTreeScanResultV0(ProjectTreeScanReasonInvalidRequestV0)
	}
	if projectTreeScanTargetIgnoredV0(target, policy.ignorePrefixes) {
		return projectTreeScanResultV0(ProjectTreeScanReasonNotFoundV0)
	}
	mode := strings.TrimSpace(request.Mode)
	if mode == "" {
		mode = ProjectTreeScanModeTargetV0
	}
	switch mode {
	case ProjectTreeScanModeGlobV0:
		return projectTreeScanWalkV0(ctx, policy, ".", func(rel string) bool {
			return worktreePathMatchesWriteSetEntryV0(rel, target)
		})
	case ProjectTreeScanModeDirV0:
		return projectTreeScanWalkV0(ctx, policy, target, func(rel string) bool {
			return rel == target || strings.HasPrefix(rel, target+"/")
		})
	default:
		return projectTreeScanTargetExistsV0(ctx, policy, target)
	}
}

type projectTreeScanPolicyV0 struct {
	root           string
	maxEntries     int
	maxDepth       int
	maxFileBytes   int64
	maxResults     int
	ignorePrefixes []string
}

func projectTreeScanPolicyFromRequestV0(request ProjectTreeScanRequestV0) projectTreeScanPolicyV0 {
	maxEntries := request.MaxEntries
	if maxEntries <= 0 {
		maxEntries = defaultProjectTreeScanMaxEntriesV0
	}
	maxDepth := request.MaxDepth
	if maxDepth <= 0 {
		maxDepth = defaultProjectTreeScanMaxDepthV0
	}
	maxFileBytes := request.MaxFileBytes
	if maxFileBytes <= 0 {
		maxFileBytes = defaultProjectTreeScanMaxFileBytesV0
	}
	return projectTreeScanPolicyV0{
		root:           strings.TrimSpace(request.ProjectRoot),
		maxEntries:     maxEntries,
		maxDepth:       maxDepth,
		maxFileBytes:   maxFileBytes,
		maxResults:     request.MaxResults,
		ignorePrefixes: normalizeWorktreeIgnorePrefixesV0(request.IgnorePrefixes),
	}
}

func projectTreeScanTargetExistsV0(ctx context.Context, policy projectTreeScanPolicyV0, target string) ProjectTreeScanResultV0 {
	if strings.ContainsAny(target, "*?[") {
		return projectTreeScanWalkV0(ctx, policy, ".", func(rel string) bool {
			return worktreePathMatchesWriteSetEntryV0(rel, target)
		})
	}
	fullPath := filepath.Join(policy.root, filepath.FromSlash(target))
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return projectTreeScanResultV0(ProjectTreeScanReasonNotFoundV0)
		}
		return projectTreeScanResultV0(ProjectTreeScanReasonFilesystemV0)
	}
	if !info.IsDir() {
		if info.Size() > 0 && info.Size() <= policy.maxFileBytes {
			return projectTreeScanFoundV0(target, 1)
		}
		return projectTreeScanResultV0(ProjectTreeScanReasonNotFoundV0)
	}
	return projectTreeScanWalkV0(ctx, policy, target, func(rel string) bool {
		return rel == target || strings.HasPrefix(rel, target+"/")
	})
}

func projectTreeScanWalkV0(
	ctx context.Context,
	policy projectTreeScanPolicyV0,
	startRel string,
	match func(string) bool,
) ProjectTreeScanResultV0 {
	start, ok := projectTreeScanTargetV0(startRel)
	if !ok {
		return projectTreeScanResultV0(ProjectTreeScanReasonInvalidRequestV0)
	}
	startPath := policy.root
	if start != "." {
		startPath = filepath.Join(policy.root, filepath.FromSlash(start))
	}
	result := projectTreeScanResultV0(ProjectTreeScanReasonNotFoundV0)
	err := filepath.WalkDir(startPath, func(path string, entry os.DirEntry, walkErr error) error {
		if ctx != nil && ctx.Err() != nil {
			result = projectTreeScanResultV0(ProjectTreeScanReasonCancelledV0)
			return filepath.SkipAll
		}
		if walkErr != nil || entry == nil {
			result.SkippedEntries++
			return nil
		}
		result.EntriesVisited++
		if result.EntriesVisited > policy.maxEntries {
			result.ReasonCode = ProjectTreeScanReasonBudgetExhaustedV0
			result.BudgetExhausted = true
			return filepath.SkipAll
		}
		rel, err := filepath.Rel(policy.root, path)
		if err != nil {
			result.SkippedEntries++
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == "" {
			rel = "."
		}
		if entry.IsDir() {
			if rel != "." && projectTreeScanTargetIgnoredV0(rel, policy.ignorePrefixes) {
				result.SkippedEntries++
				return filepath.SkipDir
			}
			if projectTreeScanDepthV0(rel) > policy.maxDepth {
				result.SkippedEntries++
				return filepath.SkipDir
			}
			return nil
		}
		if projectTreeScanTargetIgnoredV0(rel, policy.ignorePrefixes) {
			result.SkippedEntries++
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.Size() <= 0 || info.Size() > policy.maxFileBytes {
			result.SkippedEntries++
			return nil
		}
		if match(rel) {
			result.Found = true
			result.ReasonCode = ProjectTreeScanReasonFoundV0
			result.MatchedPath = rel
			result.MatchedPaths = append(result.MatchedPaths, rel)
			result.EvidenceRefs = []string{"project-tree-scan-ref-found"}
			if policy.maxResults <= 1 || len(result.MatchedPaths) >= policy.maxResults {
				return filepath.SkipAll
			}
		}
		return nil
	})
	if err != nil && !result.Found && !result.BudgetExhausted && result.ReasonCode != ProjectTreeScanReasonCancelledV0 {
		result.ReasonCode = ProjectTreeScanReasonFilesystemV0
	}
	if result.BudgetExhausted {
		result.EvidenceRefs = []string{"project-tree-scan-ref-budget-exhausted"}
	}
	return result
}

func projectTreeScanTargetV0(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "://") || strings.HasPrefix(value, "~") ||
		strings.Contains(value, "$HOME") || strings.ContainsAny(value, "\x00\r\n") ||
		filepath.IsAbs(value) {
		return "", false
	}
	value = filepath.ToSlash(filepath.Clean(value))
	if value == ".." || strings.HasPrefix(value, "../") {
		return "", false
	}
	if value == "" {
		value = "."
	}
	return value, true
}

func projectTreeScanTargetIgnoredV0(target string, ignorePrefixes []string) bool {
	if target == "." {
		return false
	}
	if strings.ContainsAny(target, "*?[") {
		return worktreePathIgnoredV0(target, ignorePrefixes)
	}
	return worktreePathIgnoredV0(target, ignorePrefixes) || IsWorktreeLocalArtifactPathV0(target)
}

func projectTreeScanDepthV0(rel string) int {
	rel = strings.Trim(rel, "/")
	if rel == "" || rel == "." {
		return 0
	}
	return strings.Count(rel, "/") + 1
}

func projectTreeScanFoundV0(path string, entries int) ProjectTreeScanResultV0 {
	return ProjectTreeScanResultV0{
		Found:          true,
		ReasonCode:     ProjectTreeScanReasonFoundV0,
		EntriesVisited: entries,
		MatchedPath:    path,
		MatchedPaths:   []string{path},
		EvidenceRefs:   []string{"project-tree-scan-ref-found"},
	}
}

func projectTreeScanResultV0(reason string) ProjectTreeScanResultV0 {
	return ProjectTreeScanResultV0{ReasonCode: reason}
}
