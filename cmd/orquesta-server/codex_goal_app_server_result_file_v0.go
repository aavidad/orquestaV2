package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

const codexAppServerGoalResultFileMaxBytesV0 = 128 * 1024

var errCodexAppServerGoalResultWalkDoneV0 = errors.New("codex_app_server_goal_result_walk_done")

type codexAppServerGoalResultContractErrorV0 struct {
	Code string
}

func (err codexAppServerGoalResultContractErrorV0) Error() string {
	return strings.TrimSpace(err.Code)
}

func (backend serverCodexAppServerGoalBackendV0) observeCodexAppServerTerminalGoalResultV0(
	ctx context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
	receipt orquestaruntimecodexgoal.CodexGoalObservationReceiptV0,
) (orquestaruntimecodexgoal.CodexGoalObservationReceiptV0, error) {
	var marker codexAppServerGoalResultMarkerV0
	markerFound := false
	markerIssue := ""
	thread, readErr := backend.Protocol.ReadThreadV0(ctx, receipt.ExternalGoalRef, true)
	if readErr == nil {
		marked, found, err := codexAppServerGoalResultFromThreadV0(thread)
		if err != nil {
			markerIssue = codexAppServerGoalResultErrorIssueCodeV0(
				err,
				"codex_app_server_goal_result_marker_invalid",
			)
		} else if found {
			if codexAppServerGoalResultMarkerGoalRefMismatchV0(marked, request.GoalRef) {
				markerIssue = "codex_app_server_goal_result_marker_goal_ref_mismatch"
			} else if codexAppServerGoalResultMarkerExternalGoalRefMismatchV0(marked, request.ExternalGoalRef) {
				markerIssue = "codex_app_server_goal_result_marker_external_goal_ref_mismatch"
			} else {
				marker = marked
				markerFound = true
			}
		}
	}
	fileMarked, fileFound, err := codexAppServerGoalResultFromWorkspaceV0(backend.CWD, request.GoalRef, request.ExternalGoalRef)
	if err != nil && !markerFound {
		if strings.TrimSpace(receipt.IssueCode) == "" {
			receipt.IssueCode = codexAppServerGoalResultErrorIssueCodeV0(
				err,
				"codex_app_server_goal_result_file_invalid",
			)
		}
		return receipt, nil
	}
	resultFound := false
	if fileFound {
		mergeCodexAppServerGoalResultV0(
			&receipt,
			fileMarked,
			"evidence-ref-codex-app-server-goal-result-file",
		)
		resultFound = true
	} else if markerFound {
		mergeCodexAppServerGoalResultV0(
			&receipt,
			marker,
			"evidence-ref-codex-app-server-goal-result-marker",
		)
		resultFound = true
	}
	if markerIssue != "" && !resultFound {
		if strings.TrimSpace(receipt.IssueCode) == "" {
			receipt.IssueCode = markerIssue
		}
		return receipt, nil
	}
	if readErr != nil && !resultFound {
		if strings.TrimSpace(receipt.IssueCode) == "" {
			receipt.IssueCode = "codex_app_server_thread_read_failed"
		}
		return receipt, nil
	}
	return receipt, nil
}

func (backend serverCodexAppServerGoalBackendV0) observeCodexAppServerActiveGoalResultV0(
	ctx context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
	receipt orquestaruntimecodexgoal.CodexGoalObservationReceiptV0,
) (orquestaruntimecodexgoal.CodexGoalObservationReceiptV0, bool) {
	if strings.TrimSpace(receipt.Status) != "running" {
		return receipt, false
	}
	observed := receipt
	if threadID := strings.TrimSpace(observed.ExternalGoalRef); threadID != "" && backend.Protocol != nil {
		if thread, err := backend.Protocol.ReadThreadV0(ctx, threadID, true); err == nil {
			marked, found, markerErr := codexAppServerGoalResultFromThreadV0(thread)
			if markerErr != nil && found && !codexAppServerGoalResultMarkerGoalRefMismatchV0(marked, request.GoalRef) {
				observed.IssueCode = codexAppServerGoalResultErrorIssueCodeV0(
					markerErr,
					"codex_app_server_goal_result_marker_invalid",
				)
				return observed, true
			}
			if markerErr == nil && found &&
				!codexAppServerGoalResultMarkerGoalRefMismatchV0(marked, request.GoalRef) &&
				!codexAppServerGoalResultMarkerExternalGoalRefMismatchV0(marked, request.ExternalGoalRef) &&
				codexAppServerGoalResultReadyForActiveCompletionV0(marked) {
				observed.Status = "complete"
				observed.Summary = "codex_app_server_goal_result_marker"
				mergeCodexAppServerGoalResultV0(
					&observed,
					marked,
					"evidence-ref-codex-app-server-goal-result-marker",
				)
				return observed, true
			}
		}
	}
	fileMarked, fileFound, fileErr := codexAppServerGoalResultFromWorkspaceV0(
		backend.CWD,
		request.GoalRef,
		request.ExternalGoalRef,
	)
	if fileErr != nil {
		observed.IssueCode = codexAppServerGoalResultErrorIssueCodeV0(
			fileErr,
			"codex_app_server_goal_result_file_invalid",
		)
		return observed, true
	}
	if fileErr == nil && fileFound && codexAppServerGoalResultReadyForActiveCompletionV0(fileMarked) {
		observed.Status = "complete"
		observed.Summary = "codex_app_server_goal_result_file"
		mergeCodexAppServerGoalResultV0(
			&observed,
			fileMarked,
			"evidence-ref-codex-app-server-goal-result-file",
		)
		return observed, true
	}
	return receipt, false
}

func codexAppServerGoalResultReadyForActiveCompletionV0(marked codexAppServerGoalResultMarkerV0) bool {
	if !codexAppServerGoalResultStatusIsCompletionV0(codexAppServerGoalResultExplicitStatusV0(marked)) {
		return false
	}
	for _, result := range marked.RequiredTestResults {
		switch strings.TrimSpace(result.Status) {
		case "passed", orquestagoal.GoalStatusAcceptedV0:
			continue
		default:
			return false
		}
	}
	return true
}

func codexAppServerGoalResultFromWorkspaceV0(
	root string,
	goalRef string,
	externalGoalRef string,
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
	var contractErr error
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
		marked, ok, err := codexAppServerGoalResultFromFileV0(path, goalRef, externalGoalRef)
		if err != nil {
			if codexAppServerGoalResultErrorIssueCodeV0(err, "") != "" {
				contractErr = err
			}
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
	if contractErr != nil {
		return codexAppServerGoalResultMarkerV0{}, false, contractErr
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
	externalGoalRef string,
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
	if codexAppServerGoalResultMarkerExternalGoalRefMismatchV0(marked, externalGoalRef) {
		return codexAppServerGoalResultMarkerV0{}, false, nil
	}
	if issue := codexAppServerGoalResultContractIssueCodeV0(marked); issue != "" {
		return marked, true, codexAppServerGoalResultContractErrorV0{Code: issue}
	}
	return marked, true, nil
}

func normalizeCodexAppServerGoalResultMarkerV0(
	marked codexAppServerGoalResultMarkerV0,
) codexAppServerGoalResultMarkerV0 {
	marked.SchemaVersion = strings.TrimSpace(marked.SchemaVersion)
	marked.Status = strings.TrimSpace(marked.Status)
	marked.Estado = strings.TrimSpace(marked.Estado)
	marked.GoalRef = strings.TrimSpace(marked.GoalRef)
	marked.ExternalGoalRef = strings.TrimSpace(marked.ExternalGoalRef)
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

func codexAppServerGoalResultContractIssueCodeV0(marked codexAppServerGoalResultMarkerV0) string {
	if strings.TrimSpace(marked.SchemaVersion) == "" {
		return "codex_app_server_goal_result_schema_version_required"
	}
	status := codexAppServerGoalResultExplicitStatusV0(marked)
	if status == "" {
		return "codex_app_server_goal_result_status_required"
	}
	if !codexAppServerGoalResultStatusIsTerminalV0(status) {
		return "codex_app_server_goal_result_status_not_terminal"
	}
	return ""
}

func codexAppServerGoalResultExplicitStatusV0(marked codexAppServerGoalResultMarkerV0) string {
	return strings.TrimSpace(firstNonEmptyServerStackV0(marked.Status, marked.Estado))
}

func codexAppServerGoalResultStatusIsTerminalV0(status string) bool {
	switch strings.TrimSpace(status) {
	case orquestagoal.GoalStatusCompleteV0, orquestagoal.GoalStatusBlockedV0, orquestagoal.GoalStatusInvalidV0:
		return true
	default:
		return false
	}
}

func codexAppServerGoalResultStatusIsCompletionV0(status string) bool {
	return strings.TrimSpace(status) == orquestagoal.GoalStatusCompleteV0
}

func codexAppServerGoalResultErrorIssueCodeV0(err error, fallback string) string {
	var contractErr codexAppServerGoalResultContractErrorV0
	if errors.As(err, &contractErr) && strings.TrimSpace(contractErr.Code) != "" {
		return strings.TrimSpace(contractErr.Code)
	}
	return strings.TrimSpace(fallback)
}
