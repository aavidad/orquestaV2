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
	"os/exec"
	"strings"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	codexGoalBackendAppServerProxyV0 = "app_server_proxy"
	codexGoalBackendAppServerStdioV0 = "app_server_stdio"
)

type serverCodexGoalBackendV0 struct {
	Starter  orquestaruntimecodexgoal.CodexGoalStarterPortV0
	Observer orquestaruntimecodexgoal.CodexGoalObserverPortV0
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
	return orquestaruntimecodexgoal.CodexGoalLauncherV0{
		Starter: backend.Starter,
	}
}

func serverGoalWorkObserverFromBackendV0(
	backend serverCodexGoalBackendV0,
) orquestagoal.GoalWorkObservationPortV0 {
	if backend.Observer == nil {
		return nil
	}
	return orquestaruntimecodexgoal.CodexGoalObserverV0{
		Observer: backend.Observer,
	}
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
	Protocol        serverCodexAppServerProtocolPortV0
	CWD             string
	Model           string
	ReasoningEffort string
	Sandbox         string
	ApprovalPolicy  string
	ServiceTier     string
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
	thread, err := backend.Protocol.StartThreadV0(ctx, backend.threadStartParamsV0())
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
	if _, err := backend.Protocol.SetGoalV0(ctx, serverCodexAppServerThreadGoalSetParamsV0{
		ThreadID:    threadID,
		Objective:   strings.TrimSpace(packet.Objective),
		Status:      "active",
		TokenBudget: tokenBudget,
	}); err != nil {
		code := codexAppServerIssueCodeForErrorV0(err, "codex_app_server_goal_set_failed")
		return codexAppServerStartReceiptV0(packet, threadID, code), err
	}
	if _, err := backend.Protocol.StartTurnV0(ctx, backend.turnStartParamsV0(threadID, packet)); err != nil {
		code := codexAppServerIssueCodeForErrorV0(err, "codex_app_server_turn_start_failed")
		return codexAppServerStartReceiptV0(packet, threadID, code), err
	}
	return orquestaruntimecodexgoal.CodexGoalStartReceiptV0{
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         packet.GoalRef,
		ExternalGoalRef: threadID,
		EvidenceRefs: []string{
			"evidence-ref-codex-app-server-thread-started",
			"evidence-ref-codex-app-server-goal-set",
			"evidence-ref-codex-app-server-turn-started",
		},
	}, nil
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
		code := codexAppServerIssueCodeForErrorV0(err, "codex_app_server_goal_get_failed")
		return codexAppServerObservationReceiptV0(request, orquestagoal.GoalStatusInvalidV0, code), err
	}
	if goal == nil {
		return codexAppServerObservationReceiptV0(request, orquestagoal.GoalStatusInvalidV0, "codex_app_server_goal_missing"), errors.New("codex_app_server_goal_missing")
	}
	status := codexAppServerGoalStatusToGoalWorkStatusV0(goal.Status)
	receipt := codexAppServerObservationReceiptV0(request, status, "codex_app_server_goal_status_"+strings.TrimSpace(goal.Status))
	if strings.TrimSpace(goal.ThreadID) != "" {
		receipt.ExternalGoalRef = strings.TrimSpace(goal.ThreadID)
	}
	if codexGoalWorkStatusIsTerminalV0(status) {
		var marker codexAppServerGoalResultMarkerV0
		markerFound := false
		markerIssue := ""
		thread, readErr := backend.Protocol.ReadThreadV0(ctx, receipt.ExternalGoalRef, true)
		if readErr == nil {
			marked, found, err := codexAppServerGoalResultFromThreadV0(thread)
			if err != nil {
				markerIssue = "codex_app_server_goal_result_marker_invalid"
			} else if found {
				if codexAppServerGoalResultMarkerGoalRefMismatchV0(marked, request.GoalRef) {
					markerIssue = "codex_app_server_goal_result_marker_goal_ref_mismatch"
				} else {
					marker = marked
					markerFound = true
				}
			}
		}
		fileMarked, fileFound, err := codexAppServerGoalResultFromWorkspaceV0(backend.CWD, request.GoalRef)
		if err != nil && !markerFound {
			receipt.IssueCode = "codex_app_server_goal_result_file_invalid"
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
			receipt.IssueCode = markerIssue
			return receipt, nil
		}
		if readErr != nil && !resultFound {
			receipt.IssueCode = "codex_app_server_thread_read_failed"
			return receipt, nil
		}
	}
	return receipt, nil
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

func (backend serverCodexAppServerGoalBackendV0) threadStartParamsV0() serverCodexAppServerThreadStartParamsV0 {
	return serverCodexAppServerThreadStartParamsV0{
		CWD:            strings.TrimSpace(backend.CWD),
		Ephemeral:      false,
		Model:          strings.TrimSpace(backend.Model),
		Sandbox:        strings.TrimSpace(backend.Sandbox),
		ApprovalPolicy: strings.TrimSpace(backend.ApprovalPolicy),
		ServiceTier:    strings.TrimSpace(backend.ServiceTier),
	}
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

func codexAppServerGoalStatusToGoalWorkStatusV0(status string) string {
	switch strings.TrimSpace(status) {
	case "complete":
		return orquestagoal.GoalStatusCompleteV0
	case "blocked", "usageLimited", "budgetLimited":
		return orquestagoal.GoalStatusBlockedV0
	case "active", "paused":
		return orquestagoal.GoalStatusRunningV0
	default:
		return orquestagoal.GoalStatusInvalidV0
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

type codexAppServerGoalResultMarkerV0 struct {
	GoalRef             string                                  `json:"goal_ref,omitempty"`
	Summary             string                                  `json:"summary,omitempty"`
	ArtifactRefs        []string                                `json:"artifact_refs,omitempty"`
	RequiredTestResults []orquestagoal.GoalRequiredTestResultV0 `json:"required_test_results,omitempty"`
	DomainReceiptRefs   []string                                `json:"domain_receipt_refs,omitempty"`
	EvidenceRefs        []string                                `json:"evidence_refs,omitempty"`
}

func codexAppServerGoalResultFromThreadV0(
	thread serverCodexAppServerThreadReadV0,
) (codexAppServerGoalResultMarkerV0, bool, error) {
	if text := codexAppServerFinalMarkerTextV0(thread, true); text != "" {
		return parseCodexAppServerGoalResultMarkerV0(text)
	}
	if text := codexAppServerFinalMarkerTextV0(thread, false); text != "" {
		return parseCodexAppServerGoalResultMarkerV0(text)
	}
	return codexAppServerGoalResultMarkerV0{}, false, nil
}

func codexAppServerFinalMarkerTextV0(thread serverCodexAppServerThreadReadV0, finalOnly bool) string {
	for turnIndex := len(thread.Turns) - 1; turnIndex >= 0; turnIndex-- {
		items := thread.Turns[turnIndex].Items
		for itemIndex := len(items) - 1; itemIndex >= 0; itemIndex-- {
			item := items[itemIndex]
			if strings.TrimSpace(item.Type) != "agentMessage" {
				continue
			}
			if finalOnly && strings.TrimSpace(item.Phase) != "final_answer" {
				continue
			}
			text := strings.TrimSpace(item.Text)
			if strings.Contains(text, orquestaruntimecodexgoal.CodexGoalResultMarkerV0) {
				return text
			}
		}
	}
	return ""
}

func parseCodexAppServerGoalResultMarkerV0(text string) (codexAppServerGoalResultMarkerV0, bool, error) {
	if !strings.Contains(text, orquestaruntimecodexgoal.CodexGoalResultMarkerV0) {
		return codexAppServerGoalResultMarkerV0{}, false, nil
	}
	payload, ok := codexAppServerGoalResultMarkerJSONV0(text)
	if !ok {
		return codexAppServerGoalResultMarkerV0{}, true, errors.New("codex_app_server_goal_result_marker_json_missing")
	}
	var marked codexAppServerGoalResultMarkerV0
	if err := json.Unmarshal([]byte(payload), &marked); err != nil {
		return codexAppServerGoalResultMarkerV0{}, true, err
	}
	return normalizeCodexAppServerGoalResultMarkerV0(marked), true, nil
}

func codexAppServerGoalResultMarkerJSONV0(text string) (string, bool) {
	markerIndex := strings.LastIndex(text, orquestaruntimecodexgoal.CodexGoalResultMarkerV0)
	if markerIndex < 0 {
		return "", false
	}
	afterMarker := text[markerIndex+len(orquestaruntimecodexgoal.CodexGoalResultMarkerV0):]
	start := strings.Index(afterMarker, "{")
	if start < 0 {
		return "", false
	}
	depth := 0
	inString := false
	escaped := false
	for index := start; index < len(afterMarker); index++ {
		ch := afterMarker[index]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return afterMarker[start : index+1], true
			}
		}
	}
	return "", false
}

func codexAppServerGoalResultMarkerGoalRefMismatchV0(
	marked codexAppServerGoalResultMarkerV0,
	goalRef string,
) bool {
	markedGoalRef := strings.TrimSpace(marked.GoalRef)
	return markedGoalRef != "" && markedGoalRef != strings.TrimSpace(goalRef)
}

func mergeCodexAppServerGoalResultV0(
	receipt *orquestaruntimecodexgoal.CodexGoalObservationReceiptV0,
	marked codexAppServerGoalResultMarkerV0,
	sourceEvidenceRef string,
) {
	if strings.TrimSpace(marked.Summary) != "" {
		receipt.Summary = strings.TrimSpace(marked.Summary)
	}
	receipt.ArtifactRefs = compactServerStackStringsV0(append(receipt.ArtifactRefs, marked.ArtifactRefs...))
	receipt.RequiredTestResults = append(receipt.RequiredTestResults, marked.RequiredTestResults...)
	receipt.DomainReceiptRefs = compactServerStackStringsV0(append(receipt.DomainReceiptRefs, marked.DomainReceiptRefs...))
	receipt.EvidenceRefs = compactServerStackStringsV0(append(
		append(receipt.EvidenceRefs, sourceEvidenceRef),
		marked.EvidenceRefs...,
	))
}

type serverCodexAppServerCommandProtocolV0 struct {
	CommandPath string
	Args        []string
	PathEnv     string
	Timeout     time.Duration
}

func (protocol serverCodexAppServerCommandProtocolV0) ProbeV0(ctx context.Context) error {
	var response serverCodexAppServerThreadLoadedListResponseV0
	return protocol.callV0(ctx, "thread/loaded/list", map[string]interface{}{}, &response)
}

func (protocol serverCodexAppServerCommandProtocolV0) StartThreadV0(
	ctx context.Context,
	params serverCodexAppServerThreadStartParamsV0,
) (serverCodexAppServerThreadV0, error) {
	var response serverCodexAppServerThreadStartResponseV0
	err := protocol.callV0(ctx, "thread/start", params.toJSONV0(), &response)
	return response.Thread, err
}

func (protocol serverCodexAppServerCommandProtocolV0) SetGoalV0(
	ctx context.Context,
	params serverCodexAppServerThreadGoalSetParamsV0,
) (serverCodexAppServerThreadGoalV0, error) {
	var response serverCodexAppServerThreadGoalSetResponseV0
	err := protocol.callV0(ctx, "thread/goal/set", params.toJSONV0(), &response)
	return response.Goal, err
}

func (protocol serverCodexAppServerCommandProtocolV0) StartTurnV0(
	ctx context.Context,
	params serverCodexAppServerTurnStartParamsV0,
) (serverCodexAppServerTurnV0, error) {
	var response serverCodexAppServerTurnStartResponseV0
	err := protocol.callV0(ctx, "turn/start", params.toJSONV0(), &response)
	return response.Turn, err
}

func (protocol serverCodexAppServerCommandProtocolV0) GetGoalV0(
	ctx context.Context,
	threadID string,
) (*serverCodexAppServerThreadGoalV0, error) {
	var response serverCodexAppServerThreadGoalGetResponseV0
	err := protocol.callV0(ctx, "thread/goal/get", map[string]interface{}{"threadId": strings.TrimSpace(threadID)}, &response)
	if err != nil {
		return nil, err
	}
	return response.Goal, nil
}

func (protocol serverCodexAppServerCommandProtocolV0) ReadThreadV0(
	ctx context.Context,
	threadID string,
	includeTurns bool,
) (serverCodexAppServerThreadReadV0, error) {
	var response serverCodexAppServerThreadReadResponseV0
	err := protocol.callV0(ctx, "thread/read", map[string]interface{}{
		"threadId":     strings.TrimSpace(threadID),
		"includeTurns": includeTurns,
	}, &response)
	return response.Thread, err
}

func (protocol serverCodexAppServerCommandProtocolV0) callV0(
	ctx context.Context,
	method string,
	params interface{},
	out interface{},
) error {
	commandPath := strings.TrimSpace(protocol.CommandPath)
	if commandPath == "" {
		commandPath = codexCommandPathV0()
	}
	timeout := protocol.Timeout
	if timeout <= 0 {
		timeout = time.Duration(defaultCodexGoalTimeoutMSV0) * time.Millisecond
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := append([]string(nil), protocol.Args...)
	if len(args) == 0 {
		args = []string{"app-server", "proxy"}
	}
	cmd := exec.CommandContext(callCtx, commandPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if strings.TrimSpace(protocol.PathEnv) != "" {
		cmd.Env = append(os.Environ(), "PATH="+protocol.PathEnv)
	}
	if err := cmd.Start(); err != nil {
		return codexAppServerCallErrorV0{
			Code: codexAppServerIssueCodeFromCommandFailureV0(stderr.String(), err),
			Err:  err,
		}
	}
	writeLine := func(line string) error {
		if _, err := io.WriteString(stdin, line); err != nil {
			_ = stdin.Close()
			waitErr := cmd.Wait()
			if waitErr != nil {
				return codexAppServerCallErrorV0{
					Code: codexAppServerIssueCodeFromCommandFailureV0(stderr.String(), waitErr),
					Err:  waitErr,
				}
			}
			return err
		}
		return nil
	}
	initLine, err := codexAppServerRPCMessageLineV0(codexAppServerInitializeRequestV0())
	if err != nil {
		return err
	}
	if err := writeLine(initLine); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var initResponse map[string]interface{}
	if decodeErr := decodeCodexAppServerRPCResponseScannerV0(scanner, 1, &initResponse); decodeErr != nil {
		_ = stdin.Close()
		err = cmd.Wait()
		if callCtx.Err() != nil {
			return codexAppServerCallErrorV0{Code: "codex_app_server_timeout", Err: callCtx.Err()}
		}
		if err != nil {
			return codexAppServerCallErrorV0{
				Code: codexAppServerIssueCodeFromCommandFailureV0(stderr.String(), err),
				Err:  err,
			}
		}
		return decodeErr
	}
	initializedLine, err := codexAppServerRPCMessageLineV0(codexAppServerInitializedNotificationV0())
	if err != nil {
		return err
	}
	if err := writeLine(initializedLine); err != nil {
		return err
	}
	callLine, err := codexAppServerRPCMessageLineV0(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return err
	}
	if err := writeLine(callLine); err != nil {
		return err
	}
	decodeErr := decodeCodexAppServerRPCResponseScannerV0(scanner, 2, out)
	_ = stdin.Close()
	err = cmd.Wait()
	if callCtx.Err() != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_timeout", Err: callCtx.Err()}
	}
	if err != nil {
		return codexAppServerCallErrorV0{
			Code: codexAppServerIssueCodeFromCommandFailureV0(stderr.String(), err),
			Err:  err,
		}
	}
	return decodeErr
}

type codexAppServerCallErrorV0 struct {
	Code string
	Err  error
}

func (err codexAppServerCallErrorV0) Error() string {
	code := strings.TrimSpace(err.Code)
	if code == "" {
		code = "codex_app_server_call_failed"
	}
	return code
}

func (err codexAppServerCallErrorV0) Unwrap() error {
	return err.Err
}

func codexAppServerIssueCodeForErrorV0(err error, fallback string) string {
	if err == nil {
		return strings.TrimSpace(fallback)
	}
	var callErr codexAppServerCallErrorV0
	if errors.As(err, &callErr) && strings.TrimSpace(callErr.Code) != "" {
		return strings.TrimSpace(callErr.Code)
	}
	message := err.Error()
	return codexAppServerIssueCodeFromStderrV0(message, fallback)
}

func codexAppServerIssueCodeFromCommandFailureV0(stderr string, err error) string {
	if code := codexAppServerIssueCodeFromMessageV0(stderr); code != "" {
		return code
	}
	if err != nil {
		if code := codexAppServerIssueCodeFromMessageV0(err.Error()); code != "" {
			return code
		}
	}
	return "codex_app_server_call_failed"
}

func codexAppServerIssueCodeFromStderrV0(message string, fallback string) string {
	if code := codexAppServerIssueCodeFromMessageV0(message); code != "" {
		return code
	}
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		return "codex_app_server_call_failed"
	}
	return fallback
}

func codexAppServerIssueCodeFromMessageV0(message string) string {
	normalized := strings.ToLower(strings.TrimSpace(message))
	switch {
	case strings.Contains(normalized, "managed standalone codex install not found"):
		return "codex_app_server_standalone_missing"
	case strings.Contains(normalized, "failed to connect to socket") ||
		strings.Contains(normalized, "app-server-control.sock"):
		return "codex_app_server_control_socket_missing"
	case strings.Contains(normalized, "executable file not found"):
		return "codex_app_server_command_missing"
	case strings.Contains(normalized, "permission denied"):
		return "codex_app_server_permission_denied"
	}
	return ""
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
			return fmt.Errorf("codex_app_server_rpc_error:%s", response.Error.Message)
		}
		if out == nil {
			return nil
		}
		if len(response.Result) == 0 {
			return errors.New("codex_app_server_empty_result")
		}
		return json.Unmarshal(response.Result, out)
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
		Message string `json:"message"`
	} `json:"error,omitempty"`
}
