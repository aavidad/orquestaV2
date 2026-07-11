package orquestaruntimecodexappserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
		if backend.mergeCodexAppServerGoalResultGuardedV0(ctx, request, &receipt, fileMarked, "evidence-ref-codex-app-server-goal-result-file") {
			return receipt, nil
		}
		resultFound = true
	} else if markerFound {
		if backend.mergeCodexAppServerGoalResultGuardedV0(ctx, request, &receipt, marker, "evidence-ref-codex-app-server-goal-result-marker") {
			return receipt, nil
		}
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
	threadReadIssueCode := ""
	threadIssueCode := ""
	threadStatus := ""
	if threadID := strings.TrimSpace(observed.ExternalGoalRef); threadID != "" && backend.Protocol != nil {
		if thread, err := backend.Protocol.ReadThreadV0(ctx, threadID, true); err == nil {
			var sanitized bool
			thread, sanitized = sanitizeCodexAppServerThreadReadV0(thread)
			if sanitized {
				observed.EvidenceRefs = compactServerStackStringsV0(append(
					observed.EvidenceRefs,
					codexAppServerThreadOutputSanitizedEvidenceRefV0,
				))
			}
			threadStatus = strings.TrimSpace(string(thread.Status))
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
				backend.mergeCodexAppServerGoalResultGuardedV0(ctx, request, &observed, marked, "evidence-ref-codex-app-server-goal-result-marker")
				return observed, true
			}
			threadIssueCode = backend.codexAppServerThreadIssueCodeV0(thread)
		} else {
			threadReadIssueCode = codexAppServerIssueCodeForErrorV0(err, "codex_app_server_thread_read_failed")
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
		backend.mergeCodexAppServerGoalResultGuardedV0(ctx, request, &observed, fileMarked, "evidence-ref-codex-app-server-goal-result-file")
		return observed, true
	}
	if codexAppServerActiveThreadReadFailureBlocksV0(threadReadIssueCode) {
		observed.Status = orquestagoal.GoalStatusBlockedV0
		observed.Summary = threadReadIssueCode
		observed.IssueCode = threadReadIssueCode
		observed.EvidenceRefs = compactServerStackStringsV0(append(
			observed.EvidenceRefs,
			"evidence-ref-codex-app-server-active-goal-thread-read-failed",
		))
		if issueEvidence := codexAppServerIssueEvidenceRefV0(threadReadIssueCode); issueEvidence != "" {
			observed.EvidenceRefs = compactServerStackStringsV0(append(observed.EvidenceRefs, issueEvidence))
		}
		return observed, true
	}
	if threadIssueCode != "" {
		observed.Status = orquestagoal.GoalStatusBlockedV0
		observed.Summary = "codex_app_server_thread_status_" + threadStatus
		observed.IssueCode = threadIssueCode
		observed.EvidenceRefs = compactServerStackStringsV0(append(
			observed.EvidenceRefs,
			"evidence-ref-codex-app-server-active-goal-thread-status",
			"evidence-ref-codex-app-server-thread-system-error",
		))
		if issueEvidence := codexAppServerIssueEvidenceRefV0(threadIssueCode); issueEvidence != "" {
			observed.EvidenceRefs = compactServerStackStringsV0(append(observed.EvidenceRefs, issueEvidence))
		}
		return observed, true
	}
	if checkpointRef, ok := codexAppServerCheckpointStartedFromWorkspaceV0(
		backend.CWD,
		request.GoalRef,
		request.ExternalGoalRef,
	); ok {
		observed.Summary = firstNonEmptyServerStackV0(observed.Summary, codexAppServerGoalResultCheckpointReasonCodeV0)
		if strings.TrimSpace(observed.IssueCode) == "" {
			observed.IssueCode = codexAppServerGoalResultCheckpointReasonCodeV0
		}
		observed.ArtifactPaths = compactServerStackStringsV0(append(observed.ArtifactPaths, checkpointRef))
		observed.EvidenceRefs = compactServerStackStringsV0(append(
			observed.EvidenceRefs,
			codexAppServerGoalResultCheckpointEvidenceRefV0,
			"evidence-ref-codex-app-server-early-checkpoint-materialized:"+checkpointRef,
		))
		return observed, true
	}
	return observed, false
}

func codexAppServerActiveThreadReadFailureBlocksV0(issueCode string) bool {
	switch strings.TrimSpace(issueCode) {
	case codexAppServerThreadReadFrameTooLargeIssueCodeV0:
		return true
	default:
		return false
	}
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
	// Runtime receipts are authoritative for new launches. The project tree is
	// retained below only as a compatibility fallback while S13 is migrated.
	runtimePath := filepath.Join(
		rootAbs,
		filepath.FromSlash(orquestaruntimecodexgoal.CodexGoalRuntimeReceiptRelativeDirV0(goalRef)),
		orquestaruntimecodexgoal.CodexGoalResultFileNameForGoalRefV0(goalRef),
	)
	if info, statErr := os.Stat(runtimePath); statErr == nil && !info.IsDir() {
		marked, ok, readErr := codexAppServerGoalResultFromFileV0(runtimePath, goalRef, externalGoalRef)
		if readErr != nil {
			return codexAppServerGoalResultMarkerV0{}, false, readErr
		}
		if ok {
			return marked, true, nil
		}
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return codexAppServerGoalResultMarkerV0{}, false, statErr
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
			!orquestaruntimecodexgoal.CodexGoalResultFileNameLooksValidV0(entry.Name()) {
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

func codexAppServerCheckpointStartedFromWorkspaceV0(
	root string,
	goalRef string,
	externalGoalRef string,
) (string, bool) {
	root = strings.TrimSpace(root)
	goalRef = strings.TrimSpace(goalRef)
	if root == "" || goalRef == "" {
		return "", false
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	runtimeRel := filepath.ToSlash(filepath.Join(
		orquestaruntimecodexgoal.CodexGoalRuntimeReceiptRelativeDirV0(goalRef),
		"checkpoint_started.txt",
	))
	runtimePath := filepath.Join(rootAbs, filepath.FromSlash(runtimeRel))
	if raw, readErr := os.ReadFile(runtimePath); readErr == nil {
		if codexAppServerCheckpointStartedMatchesV0(string(raw), goalRef, externalGoalRef) {
			return runtimeRel, true
		}
	} else if !os.IsNotExist(readErr) {
		return "", false
	}
	foundRef := ""
	_ = filepath.WalkDir(rootAbs, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil {
			return nil
		}
		if entry.IsDir() {
			if path != rootAbs && codexAppServerGoalResultSkipDirV0(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 || entry.Name() != "checkpoint_started.txt" {
			return nil
		}
		info, statErr := entry.Info()
		if statErr != nil || info.Size() <= 0 || info.Size() > 16*1024 {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil || !codexAppServerCheckpointStartedMatchesV0(string(raw), goalRef, externalGoalRef) {
			return nil
		}
		rel, relErr := filepath.Rel(rootAbs, path)
		if relErr != nil || rel == "" {
			return nil
		}
		foundRef = filepath.ToSlash(rel)
		return errCodexAppServerGoalResultWalkDoneV0
	})
	return foundRef, foundRef != ""
}

func codexAppServerCheckpointStartedMatchesV0(
	body string,
	goalRef string,
	externalGoalRef string,
) bool {
	values := map[string]string{}
	for _, line := range strings.Split(body, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if values["schema_version"] != "orquesta.codex_app_server.early_checkpoint.v0" {
		return false
	}
	if values["goal_ref"] != strings.TrimSpace(goalRef) {
		return false
	}
	expectedExternal := strings.TrimSpace(externalGoalRef)
	if expectedExternal == "" {
		return true
	}
	return values["external_goal_ref"] == expectedExternal
}

func codexAppServerGoalResultSkipDirV0(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case ".git", ".codex", ".gocache", ".gocache-local", "node_modules", "vendor":
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
	decoded, err := orquestagoal.DecodeGoalWorkResultJSONV0(raw)
	if err != nil {
		return codexAppServerGoalResultMarkerV0{}, true, err
	}
	if decoded.Disposition == orquestagoal.GoalWorkResultJSONDispositionIrrecoverableV0 {
		return codexAppServerGoalResultMarkerV0{}, true, errors.New("codex_app_server_goal_result_irrecoverable")
	}
	marked := codexAppServerGoalResultMarkerFromNeutralV0(decoded.Result)
	marked = normalizeCodexAppServerGoalResultMarkerV0(marked)
	if codexAppServerGoalResultMarkerGoalRefMismatchV0(marked, goalRef) {
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
	marked.ReasonCode = strings.TrimSpace(marked.ReasonCode)
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
	for index := range marked.MaterializedArtifacts {
		if artifactRef, ok := sanitizeCodexAppServerGoalResultRefV0(marked.MaterializedArtifacts[index].ArtifactRef); ok {
			marked.MaterializedArtifacts[index].ArtifactRef = artifactRef
			sanitized = true
		} else {
			marked.MaterializedArtifacts[index].ArtifactRef = strings.TrimSpace(marked.MaterializedArtifacts[index].ArtifactRef)
		}
		marked.MaterializedArtifacts[index].Path = filepath.ToSlash(strings.TrimSpace(marked.MaterializedArtifacts[index].Path))
		marked.MaterializedArtifacts[index].ArtifactType = strings.TrimSpace(marked.MaterializedArtifacts[index].ArtifactType)
		marked.MaterializedArtifacts[index].Scope = strings.TrimSpace(marked.MaterializedArtifacts[index].Scope)
		marked.MaterializedArtifacts[index].Status = strings.TrimSpace(marked.MaterializedArtifacts[index].Status)
		if refs, ok := sanitizeCodexAppServerGoalResultRefsV0(marked.MaterializedArtifacts[index].EvidenceRefs); ok {
			marked.MaterializedArtifacts[index].EvidenceRefs = refs
			sanitized = true
		} else {
			marked.MaterializedArtifacts[index].EvidenceRefs = refs
		}
		for issueIndex := range marked.MaterializedArtifacts[index].Issues {
			if code, ok := sanitizeCodexAppServerGoalResultRefV0(marked.MaterializedArtifacts[index].Issues[issueIndex].Code); ok {
				marked.MaterializedArtifacts[index].Issues[issueIndex].Code = code
				sanitized = true
			} else {
				marked.MaterializedArtifacts[index].Issues[issueIndex].Code = strings.TrimSpace(marked.MaterializedArtifacts[index].Issues[issueIndex].Code)
			}
			marked.MaterializedArtifacts[index].Issues[issueIndex].Field = strings.TrimSpace(marked.MaterializedArtifacts[index].Issues[issueIndex].Field)
		}
	}
	if checklist, ok := normalizeCodexAppServerGoalResultChecklistV0(marked.Checklist); ok {
		marked.Checklist = checklist
		sanitized = true
	} else {
		marked.Checklist = checklist
	}
	if refs, ok := sanitizeCodexAppServerGoalResultRefsV0(marked.MissingRefs); ok {
		marked.Checklist.MissingRefs = compactServerStackStringsV0(append(marked.Checklist.MissingRefs, refs...))
		marked.MissingRefs = nil
		sanitized = true
	} else {
		marked.Checklist.MissingRefs = compactServerStackStringsV0(append(marked.Checklist.MissingRefs, refs...))
		marked.MissingRefs = nil
	}
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
	if refs, ok := sanitizeCodexAppServerGoalResultRefsV0(marked.ReworkPlanRefs); ok {
		marked.ReworkPlanRefs = refs
		sanitized = true
	} else {
		marked.ReworkPlanRefs = refs
	}
	if sanitized {
		marked.EvidenceRefs = compactServerStackStringsV0(append(
			marked.EvidenceRefs,
			codexAppServerGoalResultSanitizedEvidenceRefV0,
		))
	}
	return marked
}

func normalizeCodexAppServerGoalResultChecklistV0(
	checklist orquestagoal.GoalWorkChecklistV0,
) (orquestagoal.GoalWorkChecklistV0, bool) {
	sanitized := false
	if refs, ok := sanitizeCodexAppServerGoalResultRefsV0(checklist.ExpectedRefs); ok {
		checklist.ExpectedRefs = refs
		sanitized = true
	} else {
		checklist.ExpectedRefs = refs
	}
	if refs, ok := sanitizeCodexAppServerGoalResultRefsV0(checklist.CompletedRefs); ok {
		checklist.CompletedRefs = refs
		sanitized = true
	} else {
		checklist.CompletedRefs = refs
	}
	if refs, ok := sanitizeCodexAppServerGoalResultRefsV0(checklist.MissingRefs); ok {
		checklist.MissingRefs = refs
		sanitized = true
	} else {
		checklist.MissingRefs = refs
	}
	if refs, ok := sanitizeCodexAppServerGoalResultRefsV0(checklist.EvidenceRefs); ok {
		checklist.EvidenceRefs = refs
		sanitized = true
	} else {
		checklist.EvidenceRefs = refs
	}
	return checklist, sanitized
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
