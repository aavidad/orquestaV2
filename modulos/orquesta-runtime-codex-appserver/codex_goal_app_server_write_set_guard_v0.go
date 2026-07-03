package orquestaruntimecodexappserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
	"unicode"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

const (
	codexAppServerRuntimeWriteSetViolationV0         = "codex_app_server_runtime_write_set_violation"
	codexAppServerRuntimeWriteSetGuardSnapshotV0     = "codex_app_server_runtime_write_set_guard_snapshot_failed"
	codexAppServerRuntimeWriteSetGuardEvidenceV0     = "evidence-ref-codex-app-server-runtime-write-set-guard"
	codexAppServerRuntimeWriteSetViolationEvidenceV0 = "evidence-ref-codex-app-server-runtime-write-set-violation"
)

func (backend serverCodexAppServerGoalBackendV0) captureCodexAppServerRuntimeWriteSetBaselineV0(
	ctx context.Context,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) (codexAppServerRuntimeWriteSetBaselineV0, bool, string) {
	if backend.Runtime == nil ||
		strings.TrimSpace(packet.DirectionContract.WriteSetEnforcement) != orquestaruntimecodexgoal.CodexGoalWriteSetEnforcementV0 {
		return codexAppServerRuntimeWriteSetBaselineV0{}, false, ""
	}
	root := strings.TrimSpace(backend.CWD)
	if root == "" {
		return codexAppServerRuntimeWriteSetBaselineV0{}, false, ""
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return codexAppServerRuntimeWriteSetBaselineV0{}, false, codexAppServerRuntimeWriteSetGuardSnapshotV0
	}
	writeSet := codexAppServerGoalWriteSetPolicyPathsV0(packet.DirectionContract.AllowedWriteSet)
	if len(writeSet) == 0 {
		return codexAppServerRuntimeWriteSetBaselineV0{}, false, ""
	}
	snapshot, issues := orquestaruntimeworktree.CaptureWorktreeSnapshotV0(ctx, orquestaruntimeworktree.WorktreeSnapshotRequestV0{
		SnapshotRef:    "codex-appserver-write-set-baseline-" + codexAppServerRuntimeWriteSetSafeRefPartV0(packet.GoalRef),
		ProjectWorkDir: rootAbs,
		AllowPartial:   true,
	})
	if len(issues) > 0 {
		return codexAppServerRuntimeWriteSetBaselineV0{}, false, codexAppServerRuntimeWriteSetGuardSnapshotV0
	}
	return codexAppServerRuntimeWriteSetBaselineV0{
		ProjectWorkDir: rootAbs,
		WriteSet:       writeSet,
		Snapshot:       snapshot,
	}, true, ""
}

func (backend serverCodexAppServerGoalBackendV0) mergeCodexAppServerGoalResultGuardedV0(
	ctx context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
	receipt *orquestaruntimecodexgoal.CodexGoalObservationReceiptV0,
	marked codexAppServerGoalResultMarkerV0,
	sourceEvidenceRef string,
) bool {
	if !backend.codexAppServerRuntimeWriteSetGuardBlocksResultV0(ctx, request, receipt, marked) {
		mergeCodexAppServerGoalResultV0(receipt, marked, sourceEvidenceRef)
		return false
	}
	return true
}

func (backend serverCodexAppServerGoalBackendV0) codexAppServerRuntimeWriteSetGuardBlocksResultV0(
	ctx context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
	receipt *orquestaruntimecodexgoal.CodexGoalObservationReceiptV0,
	marked codexAppServerGoalResultMarkerV0,
) bool {
	if receipt == nil ||
		!codexAppServerGoalResultStatusIsCompletionV0(codexAppServerGoalResultExplicitStatusV0(marked)) ||
		backend.Runtime == nil {
		return false
	}
	threadID := strings.TrimSpace(firstNonEmptyServerStackV0(receipt.ExternalGoalRef, request.ExternalGoalRef, marked.ExternalGoalRef))
	baseline, ok := backend.Runtime.writeSetBaselineForThreadV0(threadID)
	if !ok {
		return false
	}
	result, issues := orquestaruntimeworktree.VerifyWorktreeWriteSetV0(ctx, orquestaruntimeworktree.WorktreeVerifyRequestV0{
		Baseline:             baseline.Snapshot,
		ProjectWorkDir:       baseline.ProjectWorkDir,
		WriteSet:             baseline.WriteSet,
		AllowPartialSnapshot: true,
	})
	if len(issues) == 0 && result.OK {
		receipt.EvidenceRefs = compactServerStackStringsV0(append(
			receipt.EvidenceRefs,
			codexAppServerRuntimeWriteSetGuardEvidenceV0,
		))
		return false
	}
	receipt.Status = orquestagoal.GoalStatusBlockedV0
	receipt.Summary = codexAppServerRuntimeWriteSetViolationV0
	receipt.IssueCode = codexAppServerRuntimeWriteSetViolationV0
	receipt.DomainReceiptRefs = nil
	receipt.ReworkPlanRefs = nil
	receipt.EvidenceRefs = compactServerStackStringsV0(append(
		append(receipt.EvidenceRefs, codexAppServerRuntimeWriteSetViolationEvidenceV0),
		codexAppServerRuntimeWriteSetIssueEvidenceRefsV0(result, issues)...,
	))
	receipt.ArtifactPaths = compactServerStackStringsV0(append(receipt.ArtifactPaths, result.ChangedPaths...))
	return true
}

func codexAppServerRuntimeWriteSetIssueEvidenceRefsV0(
	result orquestaruntimeworktree.WorktreeVerifyResultV0,
	issues []orquestaruntimeworktree.WorktreeIssueV0,
) []string {
	refs := []string{}
	for _, issue := range issues {
		if strings.TrimSpace(string(issue.Code)) != "" {
			refs = append(refs, "evidence-ref-codex-app-server-runtime-write-set-issue-"+codexAppServerRuntimeWriteSetSafeRefPartV0(string(issue.Code)))
		}
	}
	for _, path := range result.OutsideWriteSet {
		refs = append(refs, "evidence-ref-codex-app-server-runtime-write-set-outside-"+codexAppServerRuntimeWriteSetSafeRefPartV0(path))
	}
	return compactServerStackStringsV0(refs)
}

func codexAppServerRuntimeWriteSetSafeRefPartV0(value string) string {
	value = filepath.ToSlash(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 80 {
		sum := sha256.Sum256([]byte(value))
		out = strings.Trim(out[:64], "-") + "-" + hex.EncodeToString(sum[:])[:12]
	}
	if out == "" {
		return "unknown"
	}
	return out
}
