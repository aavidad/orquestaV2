package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

const codexAppServerGoalResultFileMaxBytesV0 = 128 * 1024
const codexAppServerGoalResultSummaryMaxBytesV0 = 8 * 1024
const codexAppServerGoalResultRefMaxBytesV0 = 2 * 1024
const codexAppServerGoalResultSanitizedEvidenceRefV0 = "evidence-ref-codex-app-server-goal-result-output-sanitized"

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
		if rel, relErr := filepath.Rel(rootAbs, path); relErr == nil && rel != "" {
			marked.ArtifactPaths = compactServerStackStringsV0(append(
				marked.ArtifactPaths,
				filepath.ToSlash(rel),
			))
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
	sanitized := false
	if summary, ok := sanitizeCodexAppServerGoalResultSummaryV0(marked.Summary); ok {
		marked.Summary = summary
		sanitized = true
	} else {
		marked.Summary = strings.TrimSpace(marked.Summary)
	}
	if refs, ok := sanitizeCodexAppServerGoalResultRefsV0(marked.ArtifactRefs); ok {
		marked.ArtifactRefs = refs
		sanitized = true
	} else {
		marked.ArtifactRefs = refs
	}
	marked.ArtifactPaths = normalizeCodexAppServerGoalResultPathsV0(marked.ArtifactPaths)
	if refs, ok := sanitizeCodexAppServerGoalResultRefsV0(marked.DomainReceiptRefs); ok {
		marked.DomainReceiptRefs = refs
		sanitized = true
	} else {
		marked.DomainReceiptRefs = refs
	}
	if refs, ok := sanitizeCodexAppServerGoalResultRefsV0(marked.EvidenceRefs); ok {
		marked.EvidenceRefs = refs
		sanitized = true
	} else {
		marked.EvidenceRefs = refs
	}
	for index := range marked.RequiredTestResults {
		if testRef, ok := sanitizeCodexAppServerGoalResultRefV0(marked.RequiredTestResults[index].TestRef); ok {
			marked.RequiredTestResults[index].TestRef = testRef
			sanitized = true
		} else {
			marked.RequiredTestResults[index].TestRef = strings.TrimSpace(marked.RequiredTestResults[index].TestRef)
		}
		marked.RequiredTestResults[index].Status = strings.TrimSpace(marked.RequiredTestResults[index].Status)
		if refs, ok := sanitizeCodexAppServerGoalResultRefsV0(marked.RequiredTestResults[index].EvidenceRefs); ok {
			marked.RequiredTestResults[index].EvidenceRefs = refs
			sanitized = true
		} else {
			marked.RequiredTestResults[index].EvidenceRefs = refs
		}
	}
	if sanitized {
		marked.EvidenceRefs = compactServerStackStringsV0(append(
			marked.EvidenceRefs,
			codexAppServerGoalResultSanitizedEvidenceRefV0,
		))
	}
	return marked
}

func sanitizeCodexAppServerGoalResultSummaryV0(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if !codexAppServerGoalResultValueNeedsSanitizingV0(value, codexAppServerGoalResultSummaryMaxBytesV0) {
		return value, false
	}
	return "codex_app_server_goal_result_sanitized " +
		codexAppServerGoalResultSanitizedRefV0("goal-result-summary-ref", value), true
}

func sanitizeCodexAppServerGoalResultRefsV0(values []string) ([]string, bool) {
	out := []string{}
	sanitized := false
	for _, value := range values {
		ref, ok := sanitizeCodexAppServerGoalResultRefV0(value)
		if ok {
			sanitized = true
		}
		out = append(out, ref)
	}
	return compactServerStackStringsV0(out), sanitized
}

func normalizeCodexAppServerGoalResultPathsV0(values []string) []string {
	out := []string{}
	for _, value := range values {
		value = filepath.ToSlash(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return compactServerStackStringsV0(out)
}

func sanitizeCodexAppServerGoalResultRefV0(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if !codexAppServerGoalResultValueNeedsSanitizingV0(value, codexAppServerGoalResultRefMaxBytesV0) {
		return value, false
	}
	prefix := "large-goal-output-ref"
	if codexAppServerGoalResultContainsBase64DataURIV0(value) {
		prefix = "binary-data-ref"
	}
	return codexAppServerGoalResultSanitizedRefV0(prefix, value), true
}

func codexAppServerGoalResultValueNeedsSanitizingV0(value string, maxBytes int) bool {
	value = strings.TrimSpace(value)
	return value != "" && (len(value) > maxBytes || codexAppServerGoalResultContainsBase64DataURIV0(value))
}

func codexAppServerGoalResultContainsBase64DataURIV0(value string) bool {
	lower := strings.ToLower(value)
	index := strings.Index(lower, "data:")
	return index >= 0 && strings.Contains(lower[index:], ";base64,")
}

func codexAppServerGoalResultSanitizedRefV0(prefix string, value string) string {
	sum := sha256.Sum256([]byte(value))
	return strings.TrimSpace(prefix) + "-" + hex.EncodeToString(sum[:])[:12] + "-bytes-" + strconv.Itoa(len(value))
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
