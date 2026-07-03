package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	codexGoalBackendAppServerProxyV0      = "app_server_proxy"
	codexGoalBackendAppServerTmuxV0       = "app_server_tmux"
	codexAppServerGoalObjectiveMaxRunesV0 = 4000
)

type serverCodexGoalBackendV0 struct {
	Starter      orquestaruntimecodexgoal.CodexGoalStarterPortV0
	Observer     orquestaruntimecodexgoal.CodexGoalObserverPortV0
	ShutdownHook orquestaserver.RuntimeShutdownHookPortV0
}

type serverGoalSupervisorV0 struct {
	serverStackSupervisorV0
	launcher orquestaserver.IdleSelfImprovementGoalLauncherPortV0
	observer orquestaserver.IdleSelfImprovementGoalObserverPortV0
}

func serverSupervisorWithCodexGoalBackendV0(
	base serverStackSupervisorV0,
	backend serverCodexGoalBackendV0,
) orquestaserver.SupervisorPortV0 {
	if backend.Starter == nil || backend.Observer == nil {
		return base
	}
	return serverGoalSupervisorV0{
		serverStackSupervisorV0: base,
		launcher: orquestaruntimecodexgoal.CodexGoalLauncherV0{
			Starter: backend.Starter,
		},
		observer: orquestaruntimecodexgoal.CodexGoalObserverV0{
			Observer: backend.Observer,
		},
	}
}

func serverGoalWorkLauncherFromBackendV0(
	backend serverCodexGoalBackendV0,
) orquestagoal.GoalWorkLauncherPortV0 {
	if backend.Starter == nil {
		return nil
	}
	launcher := orquestaruntimecodexgoal.CodexGoalLauncherV0{
		Starter: backend.Starter,
	}
	active := serverGoalActiveShutdownWorkReaderFromBackendV0(backend)
	cleaner := serverGoalActiveShutdownWorkCleanerFromBackendV0(backend)
	if active != nil || cleaner != nil {
		return serverGoalWorkLauncherWithActiveShutdownWorkV0{
			Inner:         launcher,
			ActiveWork:    active,
			ActiveCleaner: cleaner,
		}
	}
	return launcher
}

func serverGoalWorkObserverFromBackendV0(
	backend serverCodexGoalBackendV0,
) orquestagoal.GoalWorkObservationPortV0 {
	if backend.Observer == nil {
		return nil
	}
	observer := orquestaruntimecodexgoal.CodexGoalObserverV0{
		Observer: backend.Observer,
	}
	active := serverGoalActiveShutdownWorkReaderFromBackendV0(backend)
	cleaner := serverGoalActiveShutdownWorkCleanerFromBackendV0(backend)
	if active != nil || cleaner != nil {
		return serverGoalWorkObserverWithActiveShutdownWorkV0{
			Inner:         observer,
			ActiveWork:    active,
			ActiveCleaner: cleaner,
		}
	}
	return observer
}

func serverGoalObservationFingerprintFromBackendV0(
	backend serverCodexGoalBackendV0,
	enabled bool,
) orquestaserver.GoalObservationFingerprintPortV0 {
	if !enabled {
		return nil
	}
	if backend.Observer == nil {
		return nil
	}
	fingerprint, ok := backend.Observer.(orquestaserver.GoalObservationFingerprintPortV0)
	if !ok {
		return nil
	}
	return fingerprint
}

func (supervisor serverGoalSupervisorV0) LaunchGoalWorkV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	if supervisor.launcher == nil {
		return orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:        orquestagoal.GoalStatusInvalidV0,
			GoalRef:       strings.TrimSpace(spec.GoalRef),
			Issues: []orquestagoal.GoalWorkIssueV0{{
				Code: orquestaruntimecodexgoal.ErrCodexGoalStarterMissingV0,
			}},
		}, errors.New(orquestaruntimecodexgoal.ErrCodexGoalStarterMissingV0)
	}
	return supervisor.launcher.LaunchGoalWorkV0(ctx, spec)
}

func (supervisor serverGoalSupervisorV0) ObserveGoalWorkV0(
	ctx context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	if supervisor.observer == nil {
		return orquestagoal.GoalWorkResultV0{
			SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
			Status:          orquestagoal.GoalStatusInvalidV0,
			GoalRef:         strings.TrimSpace(request.GoalRef),
			ExternalGoalRef: strings.TrimSpace(request.ExternalGoalRef),
			Issues: []orquestagoal.GoalWorkIssueV0{{
				Code: orquestaruntimecodexgoal.ErrCodexGoalObserverMissingV0,
			}},
		}, errors.New(orquestaruntimecodexgoal.ErrCodexGoalObserverMissingV0)
	}
	return supervisor.observer.ObserveGoalWorkV0(ctx, request)
}

func codexGoalTimeoutMSFromEnvV0() int {
	return intEnvOrDefaultV0(envCodexGoalTimeoutMSV0, defaultCodexGoalTimeoutMSV0)
}

func codexGoalPreflightTimeoutMSFromEnvV0() int {
	return intEnvOrDefaultV0(envCodexGoalPreflightTimeoutMSV0, defaultCodexGoalPreflightTimeoutMSV0)
}

type serverCodexUnavailableGoalBackendV0 struct {
	IssueCode string
}

func (backend serverCodexUnavailableGoalBackendV0) StartCodexGoalV0(
	_ context.Context,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) (orquestaruntimecodexgoal.CodexGoalStartReceiptV0, error) {
	code := strings.TrimSpace(backend.IssueCode)
	if code == "" {
		code = "codex_app_server_unavailable"
	}
	return codexAppServerStartReceiptV0(packet, "", code), errors.New(code)
}

func (backend serverCodexUnavailableGoalBackendV0) ObserveCodexGoalV0(
	_ context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
) (orquestaruntimecodexgoal.CodexGoalObservationReceiptV0, error) {
	code := strings.TrimSpace(backend.IssueCode)
	if code == "" {
		code = "codex_app_server_unavailable"
	}
	return codexAppServerObservationReceiptV0(request, orquestagoal.GoalStatusInvalidV0, code), errors.New(code)
}

type serverCodexAppServerGoalBackendV0 struct {
	Protocol          serverCodexAppServerProtocolPortV0
	CWD               string
	DiagnosticLogPath string
	AuthIssueCode     string
	Model             string
	ReasoningEffort   string
	Sandbox           string
	ApprovalPolicy    string
	ServiceTier       string
	Timeout           time.Duration
	Runtime           *serverCodexAppServerGoalRuntimeV0
	Now               func() time.Time
}

type serverCodexAppServerProtocolPortV0 interface {
	StartThreadV0(context.Context, serverCodexAppServerThreadStartParamsV0) (serverCodexAppServerThreadV0, error)
	SetGoalV0(context.Context, serverCodexAppServerThreadGoalSetParamsV0) (serverCodexAppServerThreadGoalV0, error)
	StartTurnV0(context.Context, serverCodexAppServerTurnStartParamsV0) (serverCodexAppServerTurnV0, error)
	GetGoalV0(context.Context, string) (*serverCodexAppServerThreadGoalV0, error)
	ReadThreadV0(context.Context, string, bool) (serverCodexAppServerThreadReadV0, error)
}

func (backend serverCodexAppServerGoalBackendV0) StartCodexGoalV0(
	ctx context.Context,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) (orquestaruntimecodexgoal.CodexGoalStartReceiptV0, error) {
	if backend.Protocol == nil {
		return codexAppServerStartReceiptV0(packet, "", "codex_app_server_protocol_missing"), errors.New("codex_app_server_protocol_missing")
	}
	if code := codexAppServerGoalSandboxIssueForPacketV0(backend.Sandbox, packet); code != "" {
		return codexAppServerStartReceiptV0(packet, "", code), errors.New(code)
	}
	if err := backend.prepareCodexGoalWriteSetV0(packet); err != nil {
		code := codexAppServerIssueCodeForErrorV0(err, "codex_app_server_write_set_prepare_failed")
		return codexAppServerStartReceiptV0(packet, "", code), err
	}
	thread, err := backend.Protocol.StartThreadV0(ctx, backend.threadStartParamsV0(packet))
	if err != nil {
		code := codexAppServerIssueCodeForErrorV0(err, "codex_app_server_thread_start_failed")
		return codexAppServerStartReceiptV0(packet, "", code), err
	}
	threadID := strings.TrimSpace(thread.ID)
	if threadID == "" {
		return codexAppServerStartReceiptV0(packet, "", "codex_app_server_thread_id_missing"), errors.New("codex_app_server_thread_id_missing")
	}
	tokenBudget := 0
	if packet.Budget.TokenBudget > 0 {
		tokenBudget = packet.Budget.TokenBudget
	}
	goalSetEvidence := "evidence-ref-codex-app-server-goal-set"
	if _, err := backend.Protocol.SetGoalV0(ctx, serverCodexAppServerThreadGoalSetParamsV0{
		ThreadID:    threadID,
		Objective:   codexAppServerGoalObjectiveV0(packet.Objective),
		Status:      "active",
		TokenBudget: tokenBudget,
	}); err != nil {
		if codexAppServerGoalRPCUnsupportedV0(err) {
			goalSetEvidence = "evidence-ref-codex-app-server-goal-set-unsupported"
		} else {
			code := codexAppServerIssueCodeForErrorV0(err, "codex_app_server_goal_set_failed")
			return codexAppServerStartReceiptV0(packet, threadID, code), err
		}
	}
	if _, err := backend.Protocol.StartTurnV0(ctx, backend.turnStartParamsV0(threadID, packet)); err != nil {
		code := codexAppServerIssueCodeForErrorV0(err, "codex_app_server_turn_start_failed")
		return codexAppServerStartReceiptV0(packet, threadID, code), err
	}
	if receipt, limited := backend.codexAppServerStartImmediateLimitedReceiptV0(ctx, packet, threadID, goalSetEvidence); limited {
		return receipt, errors.New(receipt.IssueCode)
	}
	backend.recordCodexAppServerGoalRuntimeV0(
		threadID,
		backend.nowCodexAppServerGoalV0(),
		backend.codexAppServerGoalTimeoutForPacketV0(packet),
	)
	return orquestaruntimecodexgoal.CodexGoalStartReceiptV0{
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         packet.GoalRef,
		ExternalGoalRef: threadID,
		EvidenceRefs: []string{
			"evidence-ref-codex-app-server-thread-started",
			goalSetEvidence,
			"evidence-ref-codex-app-server-turn-started",
		},
	}, nil
}

func (backend serverCodexAppServerGoalBackendV0) prepareCodexGoalWriteSetV0(
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) error {
	cwd := strings.TrimSpace(backend.CWD)
	if cwd == "" || len(packet.WriteSet) == 0 {
		return nil
	}
	root, err := filepath.Abs(cwd)
	if err != nil {
		return fmt.Errorf("codex_app_server_write_set_prepare_failed: %w", err)
	}
	for _, scope := range packet.WriteSet {
		rel, ok := codexAppServerWriteSetDirectoryRelV0(scope.Path)
		if !ok {
			continue
		}
		target := filepath.Join(root, filepath.FromSlash(rel))
		if !codexAppServerPathInsideRootV0(root, target) {
			continue
		}
		if err := os.MkdirAll(target, 0o700); err != nil {
			return fmt.Errorf("codex_app_server_write_set_prepare_failed: %w", err)
		}
	}
	return nil
}

func codexAppServerWriteSetDirectoryRelV0(path string) (string, bool) {
	raw := strings.TrimSpace(path)
	if raw == "" || filepath.IsAbs(raw) {
		return "", false
	}
	trimmed := strings.Trim(raw, "/")
	if trimmed == "" || strings.HasPrefix(trimmed, ".git/") || trimmed == ".git" {
		return "", false
	}
	clean := filepath.ToSlash(filepath.Clean(trimmed))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean == ".git" || strings.HasPrefix(clean, ".git/") || filepath.IsAbs(clean) {
		return "", false
	}
	if codexAppServerWriteSetLooksLikeFileV0(clean) {
		return "", false
	}
	return clean, true
}

func codexAppServerWriteSetLooksLikeFileV0(path string) bool {
	lower := strings.ToLower(strings.TrimSpace(path))
	return strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown")
}

func codexAppServerPathInsideRootV0(root string, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != "" && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func (backend serverCodexAppServerGoalBackendV0) ObserveCodexGoalV0(
	ctx context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
) (orquestaruntimecodexgoal.CodexGoalObservationReceiptV0, error) {
	if backend.Protocol == nil {
		return codexAppServerObservationReceiptV0(request, orquestagoal.GoalStatusInvalidV0, "codex_app_server_protocol_missing"), errors.New("codex_app_server_protocol_missing")
	}
	threadID := strings.TrimSpace(request.ExternalGoalRef)
	if threadID == "" {
		return codexAppServerObservationReceiptV0(request, orquestagoal.GoalStatusInvalidV0, "codex_app_server_thread_ref_required"), errors.New("codex_app_server_thread_ref_required")
	}
	goal, err := backend.Protocol.GetGoalV0(ctx, threadID)
	if err != nil {
		if codexAppServerGoalRPCUnsupportedV0(err) {
			return backend.observeCodexAppServerWithoutGoalRPCV0(ctx, request)
		}
		code := codexAppServerIssueCodeForErrorV0(err, "codex_app_server_goal_get_failed")
		return codexAppServerObservationReceiptV0(request, orquestagoal.GoalStatusInvalidV0, code), err
	}
	if goal == nil {
		return codexAppServerObservationReceiptV0(request, orquestagoal.GoalStatusInvalidV0, "codex_app_server_goal_missing"), errors.New("codex_app_server_goal_missing")
	}
	status := codexAppServerGoalStatusToGoalWorkStatusV0(goal.Status)
	receipt := codexAppServerObservationReceiptV0(request, status, "codex_app_server_goal_status_"+strings.TrimSpace(goal.Status))
	receipt = codexAppServerObservationReceiptWithGoalStatusCauseV0(receipt, goal.Status)
	if strings.TrimSpace(goal.ThreadID) != "" {
		receipt.ExternalGoalRef = strings.TrimSpace(goal.ThreadID)
	}
	receipt = codexAppServerObservationReceiptWithGoalUsageV0(receipt, goal)
	var activeFound bool
	receipt, activeFound = backend.observeCodexAppServerActiveGoalResultV0(ctx, request, receipt)
	if activeFound {
		return receipt, nil
	}
	if timedOut, timeoutReceipt := backend.codexAppServerActiveGoalTimeoutV0(request, goal, status, receipt); timedOut {
		return timeoutReceipt, nil
	}
	if codexGoalWorkStatusIsTerminalV0(status) {
		return backend.observeCodexAppServerTerminalGoalResultV0(ctx, request, receipt)
	}
	return receipt, nil
}

func (backend serverCodexAppServerGoalBackendV0) codexAppServerActiveGoalTimeoutV0(
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
	goal *serverCodexAppServerThreadGoalV0,
	status string,
	receipt orquestaruntimecodexgoal.CodexGoalObservationReceiptV0,
) (bool, orquestaruntimecodexgoal.CodexGoalObservationReceiptV0) {
	if goal == nil || status != orquestagoal.GoalStatusRunningV0 {
		return false, receipt
	}
	threadID := strings.TrimSpace(firstNonEmptyServerStackV0(goal.ThreadID, receipt.ExternalGoalRef, request.ExternalGoalRef))
	timeout := backend.codexAppServerGoalTimeoutForThreadV0(threadID)
	elapsed, ok := backend.codexAppServerActiveGoalElapsedV0(goal)
	if timeout <= 0 || !ok {
		return false, receipt
	}
	if elapsed < timeout {
		return false, receipt
	}
	receipt.Status = orquestagoal.GoalStatusBlockedV0
	receipt.Summary = "codex_app_server_goal_active_timeout"
	receipt.IssueCode = "codex_app_server_goal_active_timeout"
	receipt.EvidenceRefs = compactServerStackStringsV0(append(
		receipt.EvidenceRefs,
		"evidence-ref-codex-app-server-goal-active-timeout",
	))
	receipt.GoalRef = strings.TrimSpace(firstNonEmptyServerStackV0(receipt.GoalRef, request.GoalRef))
	receipt.ExternalGoalRef = strings.TrimSpace(firstNonEmptyServerStackV0(receipt.ExternalGoalRef, request.ExternalGoalRef))
	return true, receipt
}

func (backend serverCodexAppServerGoalBackendV0) FingerprintGoalObservationV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
) (orquestagoal.GoalObservationFingerprintV0, bool, error) {
	if backend.Protocol == nil {
		return orquestagoal.GoalObservationFingerprintV0{}, false, nil
	}
	state = orquestagoal.NormalizeGoalWorkStateV0(state)
	threadID := firstNonEmptyServerStackV0(
		state.ExternalGoalRef,
		state.LaunchReceipt.ExternalGoalRef,
	)
	if strings.TrimSpace(threadID) == "" && state.LastResult != nil {
		threadID = strings.TrimSpace(state.LastResult.ExternalGoalRef)
	}
	if strings.TrimSpace(threadID) == "" {
		return orquestagoal.GoalObservationFingerprintV0{}, false, nil
	}
	goal, err := backend.Protocol.GetGoalV0(ctx, threadID)
	if err != nil {
		if codexAppServerGoalRPCUnsupportedV0(err) {
			return backend.fingerprintCodexAppServerThreadReadV0(ctx, state, threadID)
		}
		return orquestagoal.GoalObservationFingerprintV0{}, true, err
	}
	if goal == nil {
		return orquestagoal.GoalObservationFingerprintV0{}, true, errors.New("codex_app_server_goal_missing")
	}
	status := codexAppServerGoalStatusToGoalWorkStatusV0(goal.Status)
	return orquestagoal.NormalizeGoalObservationFingerprintV0(orquestagoal.GoalObservationFingerprintV0{
		RunRef:       state.RunRef,
		GoalRef:      state.GoalRef,
		LastStatus:   status,
		EvidenceHash: codexAppServerGoalFingerprintHashV0(*goal),
	}), true, nil
}

func codexAppServerGoalRPCUnsupportedV0(err error) bool {
	var callErr codexAppServerCallErrorV0
	if !errors.As(err, &callErr) {
		return false
	}
	switch strings.TrimSpace(callErr.Code) {
	case "codex_app_server_rpc_method_not_found", "codex_app_server_rpc_invalid_request":
		return true
	default:
		return false
	}
}

func (backend serverCodexAppServerGoalBackendV0) observeCodexAppServerWithoutGoalRPCV0(
	ctx context.Context,
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
) (orquestaruntimecodexgoal.CodexGoalObservationReceiptV0, error) {
	thread, err := backend.Protocol.ReadThreadV0(ctx, request.ExternalGoalRef, true)
	if err != nil {
		code := codexAppServerIssueCodeForErrorV0(err, "codex_app_server_thread_read_failed")
		return codexAppServerObservationReceiptV0(request, orquestagoal.GoalStatusInvalidV0, code), err
	}
	thread, sanitized := sanitizeCodexAppServerThreadReadV0(thread)
	status := codexAppServerThreadStatusToGoalWorkStatusV0(thread.Status)
	receipt := codexAppServerObservationReceiptV0(
		request,
		status,
		"codex_app_server_thread_status_"+strings.TrimSpace(string(thread.Status)),
	)
	receipt.EvidenceRefs = compactServerStackStringsV0(append(
		receipt.EvidenceRefs,
		"evidence-ref-codex-app-server-goal-rpc-unsupported",
	))
	if sanitized {
		receipt.EvidenceRefs = compactServerStackStringsV0(append(
			receipt.EvidenceRefs,
			codexAppServerThreadOutputSanitizedEvidenceRefV0,
		))
	}
	marked, found, markerErr := codexAppServerGoalResultFromThreadV0(thread)
	if markerErr != nil {
		receipt.IssueCode = codexAppServerGoalResultErrorIssueCodeV0(
			markerErr,
			"codex_app_server_goal_result_marker_invalid",
		)
		return receipt, nil
	}
	if found &&
		!codexAppServerGoalResultMarkerGoalRefMismatchV0(marked, request.GoalRef) &&
		!codexAppServerGoalResultMarkerExternalGoalRefMismatchV0(marked, request.ExternalGoalRef) &&
		(status != orquestagoal.GoalStatusRunningV0 || codexAppServerGoalResultReadyForActiveCompletionV0(marked)) {
		receipt.Status = orquestagoal.GoalStatusCompleteV0
		receipt.Summary = "codex_app_server_goal_result_marker"
		mergeCodexAppServerGoalResultV0(
			&receipt,
			marked,
			"evidence-ref-codex-app-server-goal-result-marker",
		)
		return receipt, nil
	}
	fileMarked, fileFound, fileErr := codexAppServerGoalResultFromWorkspaceV0(
		backend.CWD,
		request.GoalRef,
		request.ExternalGoalRef,
	)
	if fileErr != nil {
		receipt.IssueCode = codexAppServerGoalResultErrorIssueCodeV0(
			fileErr,
			"codex_app_server_goal_result_file_invalid",
		)
		return receipt, nil
	}
	if fileFound &&
		(status != orquestagoal.GoalStatusRunningV0 || codexAppServerGoalResultReadyForActiveCompletionV0(fileMarked)) {
		receipt.Status = orquestagoal.GoalStatusCompleteV0
		receipt.Summary = "codex_app_server_goal_result_file"
		mergeCodexAppServerGoalResultV0(
			&receipt,
			fileMarked,
			"evidence-ref-codex-app-server-goal-result-file",
		)
		return receipt, nil
	}
	if issueCode := backend.codexAppServerThreadStatusIssueCodeV0(thread.Status); issueCode != "" {
		receipt.IssueCode = issueCode
		receipt.EvidenceRefs = compactServerStackStringsV0(append(
			receipt.EvidenceRefs,
			"evidence-ref-codex-app-server-thread-system-error",
		))
		if issueEvidence := codexAppServerIssueEvidenceRefV0(issueCode); issueEvidence != "" {
			receipt.EvidenceRefs = compactServerStackStringsV0(append(receipt.EvidenceRefs, issueEvidence))
		}
		return receipt, nil
	}
	if timedOut, timeoutReceipt := backend.codexAppServerThreadReadGoalResultTimeoutV0(request, status, thread, receipt); timedOut {
		return timeoutReceipt, nil
	}
	return receipt, nil
}

func (backend serverCodexAppServerGoalBackendV0) codexAppServerThreadReadGoalResultTimeoutV0(
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
	status string,
	thread serverCodexAppServerThreadReadV0,
	receipt orquestaruntimecodexgoal.CodexGoalObservationReceiptV0,
) (bool, orquestaruntimecodexgoal.CodexGoalObservationReceiptV0) {
	if status != orquestagoal.GoalStatusRunningV0 {
		return false, receipt
	}
	if codexAppServerThreadReadHasActiveTurnV0(thread) {
		return false, receipt
	}
	threadID := strings.TrimSpace(firstNonEmptyServerStackV0(receipt.ExternalGoalRef, request.ExternalGoalRef))
	timeout := backend.codexAppServerGoalTimeoutForThreadV0(threadID)
	if timeout <= 0 || threadID == "" || backend.Runtime == nil {
		return false, receipt
	}
	startedAt := backend.Runtime.ensureStartedAtV0(threadID, backend.nowCodexAppServerGoalV0())
	if startedAt.IsZero() || backend.nowCodexAppServerGoalV0().Sub(startedAt) < timeout {
		return false, receipt
	}
	receipt.Status = orquestagoal.GoalStatusBlockedV0
	receipt.Summary = "codex_app_server_goal_result_missing_after_timeout"
	receipt.IssueCode = "codex_app_server_goal_result_missing_after_timeout"
	receipt.EvidenceRefs = compactServerStackStringsV0(append(
		receipt.EvidenceRefs,
		"evidence-ref-codex-app-server-goal-result-missing-after-timeout",
	))
	receipt.GoalRef = strings.TrimSpace(firstNonEmptyServerStackV0(receipt.GoalRef, request.GoalRef))
	receipt.ExternalGoalRef = threadID
	return true, receipt
}

func codexAppServerThreadReadHasActiveTurnV0(thread serverCodexAppServerThreadReadV0) bool {
	for _, turn := range thread.Turns {
		switch strings.TrimSpace(turn.Status) {
		case "active", "inProgress", "in_progress", "pending", "queued", "running":
			return true
		}
	}
	return false
}

func (backend serverCodexAppServerGoalBackendV0) fingerprintCodexAppServerThreadReadV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
	threadID string,
) (orquestagoal.GoalObservationFingerprintV0, bool, error) {
	thread, err := backend.Protocol.ReadThreadV0(ctx, threadID, false)
	if err != nil {
		return orquestagoal.GoalObservationFingerprintV0{}, true, err
	}
	status := codexAppServerThreadStatusToGoalWorkStatusV0(thread.Status)
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(thread.ID),
		strings.TrimSpace(string(thread.Status)),
	}, "\x00")))
	return orquestagoal.NormalizeGoalObservationFingerprintV0(orquestagoal.GoalObservationFingerprintV0{
		RunRef:       state.RunRef,
		GoalRef:      state.GoalRef,
		LastStatus:   status,
		EvidenceHash: fmt.Sprintf("%x", sum[:]),
	}), true, nil
}

func codexAppServerGoalFingerprintHashV0(goal serverCodexAppServerThreadGoalV0) string {
	var tokenBudget int
	if goal.TokenBudget != nil {
		tokenBudget = *goal.TokenBudget
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(goal.ThreadID),
		strings.TrimSpace(goal.Status),
		fmt.Sprintf("%d", goal.TokensUsed),
		fmt.Sprintf("%d", goal.TimeUsedSeconds),
		fmt.Sprintf("%d", tokenBudget),
	}, "\x00")))
	return fmt.Sprintf("%x", sum[:])
}

func codexAppServerGoalObjectiveV0(objective string) string {
	objective = strings.TrimSpace(objective)
	if len([]rune(objective)) <= codexAppServerGoalObjectiveMaxRunesV0 {
		return objective
	}
	sum := sha256.Sum256([]byte(objective))
	suffix := fmt.Sprintf(
		"\n\n[objective_compacted original_sha256=%x original_bytes=%d prompt_contains_full_objective]",
		sum[:],
		len([]byte(objective)),
	)
	limit := codexAppServerGoalObjectiveMaxRunesV0 - len([]rune(suffix))
	if limit < 1 {
		return strings.TrimSpace(string([]rune(suffix)[:codexAppServerGoalObjectiveMaxRunesV0]))
	}
	return strings.TrimSpace(string([]rune(objective)[:limit])) + suffix
}

func (backend serverCodexAppServerGoalBackendV0) threadStartParamsV0(
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) serverCodexAppServerThreadStartParamsV0 {
	return serverCodexAppServerThreadStartParamsV0{
		CWD:            strings.TrimSpace(backend.CWD),
		Ephemeral:      false,
		Model:          strings.TrimSpace(backend.Model),
		Sandbox:        codexAppServerGoalSandboxForPacketV0(backend.Sandbox, packet),
		ApprovalPolicy: strings.TrimSpace(backend.ApprovalPolicy),
		ServiceTier:    strings.TrimSpace(backend.ServiceTier),
	}
}

func codexAppServerGoalSandboxForPacketV0(
	configured string,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) string {
	configured = strings.TrimSpace(configured)
	if strings.TrimSpace(packet.DirectionContract.WriteSetEnforcement) != orquestaruntimecodexgoal.CodexGoalWriteSetEnforcementV0 {
		return configured
	}
	minimum := strings.TrimSpace(packet.DirectionContract.MinimumSandbox)
	if minimum == "" {
		minimum = orquestaruntimecodexgoal.CodexGoalMinimumSandboxV0
	}
	switch strings.TrimSpace(configured) {
	case "", "danger-full-access":
		return minimum
	default:
		return configured
	}
}

func codexAppServerGoalSandboxIssueForPacketV0(
	configured string,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) string {
	if strings.TrimSpace(packet.DirectionContract.WriteSetEnforcement) != orquestaruntimecodexgoal.CodexGoalWriteSetEnforcementV0 {
		return ""
	}
	if strings.TrimSpace(configured) == "read-only" {
		return "codex_app_server_write_set_requires_workspace_write"
	}
	return ""
}

func (backend serverCodexAppServerGoalBackendV0) turnStartParamsV0(
	threadID string,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) serverCodexAppServerTurnStartParamsV0 {
	return serverCodexAppServerTurnStartParamsV0{
		ThreadID:        threadID,
		CWD:             strings.TrimSpace(backend.CWD),
		InputText:       packet.Prompt,
		ClientMessageID: packet.GoalRef,
		Model:           strings.TrimSpace(backend.Model),
		Effort:          strings.TrimSpace(backend.ReasoningEffort),
		ApprovalPolicy:  strings.TrimSpace(backend.ApprovalPolicy),
		ServiceTier:     strings.TrimSpace(backend.ServiceTier),
	}
}

func codexAppServerStartReceiptV0(
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
	externalGoalRef string,
	code string,
) orquestaruntimecodexgoal.CodexGoalStartReceiptV0 {
	return orquestaruntimecodexgoal.CodexGoalStartReceiptV0{
		Status:          orquestagoal.GoalStatusInvalidV0,
		GoalRef:         strings.TrimSpace(packet.GoalRef),
		ExternalGoalRef: strings.TrimSpace(externalGoalRef),
		IssueCode:       code,
		EvidenceRefs:    codexAppServerStartIssueEvidenceRefsV0(code),
	}
}

func codexAppServerStartIssueEvidenceRefsV0(code string) []string {
	switch strings.TrimSpace(code) {
	case "codex_app_server_write_set_requires_workspace_write":
		return []string{"evidence-ref-codex-app-server-write-set-requires-workspace-write"}
	default:
		return nil
	}
}

func codexAppServerObservationReceiptV0(
	request orquestaruntimecodexgoal.CodexGoalObservationRequestV0,
	status string,
	summary string,
) orquestaruntimecodexgoal.CodexGoalObservationReceiptV0 {
	return orquestaruntimecodexgoal.CodexGoalObservationReceiptV0{
		Status:          status,
		GoalRef:         strings.TrimSpace(request.GoalRef),
		ExternalGoalRef: strings.TrimSpace(request.ExternalGoalRef),
		Summary:         summary,
		EvidenceRefs:    []string{"evidence-ref-codex-app-server-goal-observed"},
	}
}

func codexGoalWorkStatusIsTerminalV0(status string) bool {
	switch strings.TrimSpace(status) {
	case orquestagoal.GoalStatusCompleteV0, orquestagoal.GoalStatusBlockedV0:
		return true
	default:
		return false
	}
}

func codexAppServerRPCPayloadV0(method string, params interface{}) (string, error) {
	var b strings.Builder
	for _, request := range []map[string]interface{}{
		codexAppServerInitializeRequestV0(),
		codexAppServerInitializedNotificationV0(),
		{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  method,
			"params":  params,
		},
	} {
		line, err := codexAppServerRPCMessageLineV0(request)
		if err != nil {
			return "", err
		}
		b.WriteString(line)
	}
	return b.String(), nil
}

func codexAppServerInitializeRequestV0() map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"clientInfo": map[string]string{
				"name":    "orquesta-server",
				"version": "0",
			},
			"capabilities": map[string]interface{}{
				"experimentalApi": true,
			},
		},
	}
}

func codexAppServerInitializedNotificationV0() map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialized",
		"params":  map[string]interface{}{},
	}
}

func codexAppServerRPCMessageLineV0(request map[string]interface{}) (string, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

func decodeCodexAppServerRPCResponseV0(stdout []byte, responseID int, out interface{}) error {
	return decodeCodexAppServerRPCResponseReaderV0(bytes.NewReader(stdout), responseID, out)
}

func decodeCodexAppServerRPCResponseReaderV0(stdout io.Reader, responseID int, out interface{}) error {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return decodeCodexAppServerRPCResponseScannerV0(scanner, responseID, out)
}

func decodeCodexAppServerRPCResponseScannerV0(scanner *bufio.Scanner, responseID int, out interface{}) error {
	var lastErr error
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var response serverCodexAppServerRPCResponseV0
		if err := json.Unmarshal(line, &response); err != nil {
			lastErr = err
			continue
		}
		if response.ID != responseID {
			continue
		}
		if response.Error != nil {
			return codexAppServerRPCErrorV0(response.Error.Code, response.Error.Message)
		}
		if out == nil {
			return nil
		}
		if len(response.Result) == 0 {
			return errors.New("codex_app_server_empty_result")
		}
		if err := json.Unmarshal(response.Result, out); err != nil {
			return err
		}
		sanitizeCodexAppServerRPCDecodedOutV0(out)
		return nil
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if lastErr != nil {
		return lastErr
	}
	return errors.New("codex_app_server_response_missing")
}

type serverCodexAppServerRPCResponseV0 struct {
	ID     int             `json:"id,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *struct {
		Code    int    `json:"code,omitempty"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func codexAppServerRPCErrorV0(code int, message string) error {
	return codexAppServerCallErrorV0{
		Code: codexAppServerRPCIssueCodeV0(code),
		Err:  fmt.Errorf("codex_app_server_rpc_error: code=%d message=%s", code, strings.TrimSpace(message)),
	}
}

func codexAppServerRPCIssueCodeV0(code int) string {
	switch code {
	case -32700:
		return "codex_app_server_rpc_parse_error"
	case -32600:
		return "codex_app_server_rpc_invalid_request"
	case -32601:
		return "codex_app_server_rpc_method_not_found"
	case -32602:
		return "codex_app_server_rpc_invalid_params"
	case -32603:
		return "codex_app_server_rpc_internal_error"
	default:
		return "codex_app_server_rpc_error"
	}
}
