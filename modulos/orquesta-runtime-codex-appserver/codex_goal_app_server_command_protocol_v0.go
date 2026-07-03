package orquestaruntimecodexappserver

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type serverCodexAppServerCommandProtocolV0 struct {
	CommandPath string
	Args        []string
	PathEnv     string
	Timeout     time.Duration
}

const codexAppServerDiagnosticLogMaxBytesV0 = 8192

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
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_command_args_required",
			Err:  errors.New("codex_app_server_command_args_required"),
		}
	}
	cmd := exec.CommandContext(callCtx, commandPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
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
	if decodeErr := decodeCodexAppServerRPCResponseScannerWithContextV0(callCtx, scanner, cmd, stdin, 1, &initResponse); decodeErr != nil {
		var callErr codexAppServerCallErrorV0
		if errors.As(decodeErr, &callErr) {
			return callErr
		}
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
	decodeErr := decodeCodexAppServerRPCResponseScannerWithContextV0(callCtx, scanner, cmd, stdin, 2, out)
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

func decodeCodexAppServerRPCResponseScannerWithContextV0(
	ctx context.Context,
	scanner *bufio.Scanner,
	cmd *exec.Cmd,
	stdin io.Closer,
	responseID int,
	out interface{},
) error {
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- decodeCodexAppServerRPCResponseScannerV0(scanner, responseID, out)
	}()
	select {
	case err := <-resultCh:
		return err
	case <-ctx.Done():
		if stdin != nil {
			_ = stdin.Close()
		}
		codexAppServerKillProcessGroupV0(cmd)
		if cmd != nil {
			_ = cmd.Wait()
		}
		return codexAppServerCallErrorV0{Code: "codex_app_server_timeout", Err: ctx.Err()}
	}
}

func codexAppServerKillProcessGroupV0(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	if pid <= 0 {
		return
	}
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		_ = cmd.Process.Kill()
	}
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

func codexAppServerIssueCodeFromLogFileV0(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if len(raw) > codexAppServerDiagnosticLogMaxBytesV0 {
		raw = raw[len(raw)-codexAppServerDiagnosticLogMaxBytesV0:]
	}
	return codexAppServerIssueCodeFromMessageV0(string(raw))
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
	case strings.Contains(normalized, "resetstdio") ||
		strings.Contains(normalized, "node.cc:751"):
		return "codex_app_server_wrapper_stdio_failed"
	case strings.Contains(normalized, "401 unauthorized") ||
		(strings.Contains(normalized, "unauthorized") &&
			(strings.Contains(normalized, "api.openai.com/v1/responses") ||
				strings.Contains(normalized, "responses_websocket"))):
		return "codex_app_server_provider_unauthorized"
	case strings.Contains(normalized, "budgetlimited") ||
		strings.Contains(normalized, "budget limited") ||
		strings.Contains(normalized, "budget limit"):
		return "codex_app_server_goal_budget_limited"
	case strings.Contains(normalized, "policylimited") ||
		strings.Contains(normalized, "policy limited") ||
		strings.Contains(normalized, "policy limit"):
		return "codex_app_server_goal_policy_limited"
	case strings.Contains(normalized, "usagelimited") ||
		strings.Contains(normalized, "usage limited") ||
		strings.Contains(normalized, "usage limit") ||
		strings.Contains(normalized, "quota limited") ||
		strings.Contains(normalized, "quota exhausted") ||
		strings.Contains(normalized, "provider limited") ||
		strings.Contains(normalized, `"has_credits":false`) ||
		strings.Contains(normalized, `"has_credits": false`):
		return "codex_app_server_goal_provider_limited"
	case strings.Contains(normalized, "codex_app_server_storage_quota_exceeded") ||
		strings.Contains(normalized, "quota exceeded (os error 122)") ||
		strings.Contains(normalized, "disk quota exceeded") ||
		strings.Contains(normalized, "no space left on device") ||
		strings.Contains(normalized, "enospc"):
		return "codex_app_server_storage_quota_exceeded"
	case strings.Contains(normalized, "managed standalone codex install not found"):
		return "codex_app_server_standalone_missing"
	case strings.Contains(normalized, "failed to connect to socket") ||
		strings.Contains(normalized, "app-server-control.sock"):
		return "codex_app_server_control_socket_missing"
	case strings.Contains(normalized, "executable file not found"):
		return "codex_app_server_command_missing"
	case strings.Contains(normalized, "permission denied"):
		return "codex_app_server_permission_denied"
	case strings.Contains(normalized, "operation not permitted"):
		return "codex_app_server_operation_not_permitted"
	}
	return ""
}
