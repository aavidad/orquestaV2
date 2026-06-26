package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

const codexAppServerGoalResultFileMaxBytesV0 = 128 * 1024

var errCodexAppServerGoalResultWalkDoneV0 = errors.New("codex_app_server_goal_result_walk_done")

func codexAppServerGoalResultFromWorkspaceV0(
	root string,
	goalRef string,
) (codexAppServerGoalResultMarkerV0, bool, error) {
	root = strings.TrimSpace(root)
	goalRef = strings.TrimSpace(goalRef)
	if root == "" || goalRef == "" {
		return codexAppServerGoalResultMarkerV0{}, false, nil
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return codexAppServerGoalResultMarkerV0{}, false, err
	}
	var found codexAppServerGoalResultMarkerV0
	foundOK := false
	err = filepath.WalkDir(rootAbs, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry == nil {
			return nil
		}
		if entry.IsDir() {
			if path != rootAbs && codexAppServerGoalResultSkipDirV0(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 ||
			entry.Name() != orquestaruntimecodexgoal.CodexGoalResultFileNameV0 {
			return nil
		}
		marked, ok, err := codexAppServerGoalResultFromFileV0(path, goalRef)
		if err != nil {
			return nil
		}
		if !ok {
			return nil
		}
		found = marked
		foundOK = true
		return errCodexAppServerGoalResultWalkDoneV0
	})
	if errors.Is(err, errCodexAppServerGoalResultWalkDoneV0) {
		return found, foundOK, nil
	}
	if err != nil {
		return codexAppServerGoalResultMarkerV0{}, false, err
	}
	return found, foundOK, nil
}

func codexAppServerGoalResultSkipDirV0(name string) bool {
	switch strings.TrimSpace(name) {
	case ".git", ".codex", "node_modules", "vendor":
		return true
	default:
		return false
	}
}

func codexAppServerGoalResultFromFileV0(
	path string,
	goalRef string,
) (codexAppServerGoalResultMarkerV0, bool, error) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() <= 0 || info.Size() > codexAppServerGoalResultFileMaxBytesV0 {
		return codexAppServerGoalResultMarkerV0{}, false, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return codexAppServerGoalResultMarkerV0{}, false, err
	}
	var marked codexAppServerGoalResultMarkerV0
	if err := json.Unmarshal(raw, &marked); err != nil {
		return codexAppServerGoalResultMarkerV0{}, true, err
	}
	marked = normalizeCodexAppServerGoalResultMarkerV0(marked)
	if marked.GoalRef != strings.TrimSpace(goalRef) {
		return codexAppServerGoalResultMarkerV0{}, false, nil
	}
	return marked, true, nil
}

func normalizeCodexAppServerGoalResultMarkerV0(
	marked codexAppServerGoalResultMarkerV0,
) codexAppServerGoalResultMarkerV0 {
	marked.GoalRef = strings.TrimSpace(marked.GoalRef)
	marked.Summary = strings.TrimSpace(marked.Summary)
	marked.ArtifactRefs = compactServerStackStringsV0(marked.ArtifactRefs)
	marked.DomainReceiptRefs = compactServerStackStringsV0(marked.DomainReceiptRefs)
	marked.EvidenceRefs = compactServerStackStringsV0(marked.EvidenceRefs)
	for index := range marked.RequiredTestResults {
		marked.RequiredTestResults[index].TestRef = strings.TrimSpace(marked.RequiredTestResults[index].TestRef)
		marked.RequiredTestResults[index].Status = strings.TrimSpace(marked.RequiredTestResults[index].Status)
		marked.RequiredTestResults[index].EvidenceRefs = compactServerStackStringsV0(
			marked.RequiredTestResults[index].EvidenceRefs,
		)
	}
	return marked
}
