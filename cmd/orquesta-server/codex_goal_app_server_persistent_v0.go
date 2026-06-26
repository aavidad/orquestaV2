package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type serverCodexAppServerPersistentCommandProtocolV0 struct {
	CommandPath string
	Args        []string
	PathEnv     string
	Timeout     time.Duration

	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	stderr  bytes.Buffer
	nextID  int
}

func (protocol *serverCodexAppServerPersistentCommandProtocolV0) StartThreadV0(
	ctx context.Context,
	params serverCodexAppServerThreadStartParamsV0,
) (serverCodexAppServerThreadV0, error) {
	var response serverCodexAppServerThreadStartResponseV0
	err := protocol.callV0(ctx, "thread/start", params.toJSONV0(), &response)
	return response.Thread, err
}

func (protocol *serverCodexAppServerPersistentCommandProtocolV0) SetGoalV0(
	ctx context.Context,
	params serverCodexAppServerThreadGoalSetParamsV0,
) (serverCodexAppServerThreadGoalV0, error) {
	var response serverCodexAppServerThreadGoalSetResponseV0
	err := protocol.callV0(ctx, "thread/goal/set", params.toJSONV0(), &response)
	return response.Goal, err
}

func (protocol *serverCodexAppServerPersistentCommandProtocolV0) StartTurnV0(
	ctx context.Context,
	params serverCodexAppServerTurnStartParamsV0,
) (serverCodexAppServerTurnV0, error) {
	var response serverCodexAppServerTurnStartResponseV0
	err := protocol.callV0(ctx, "turn/start", params.toJSONV0(), &response)
	return response.Turn, err
}

func (protocol *serverCodexAppServerPersistentCommandProtocolV0) GetGoalV0(
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

func (protocol *serverCodexAppServerPersistentCommandProtocolV0) ReadThreadV0(
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

func (protocol *serverCodexAppServerPersistentCommandProtocolV0) callV0(
	ctx context.Context,
	method string,
	params interface{},
	out interface{},
) error {
	if protocol == nil {
		return errors.New("codex_app_server_protocol_missing")
	}
	timeout := protocol.Timeout
	if timeout <= 0 {
		timeout = time.Duration(defaultCodexGoalTimeoutMSV0) * time.Millisecond
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	protocol.mu.Lock()
	defer protocol.mu.Unlock()

	if err := protocol.ensureStartedLockedV0(callCtx); err != nil {
		return err
	}
	requestID := protocol.nextID
	if requestID <= 1 {
		requestID = 2
	}
	protocol.nextID = requestID + 1
	line, err := codexAppServerRPCMessageLineV0(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      requestID,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return err
	}
	if _, err := io.WriteString(protocol.stdin, line); err != nil {
		waitErr := protocol.stopLockedV0()
		if waitErr != nil {
			return codexAppServerCallErrorV0{
				Code: codexAppServerIssueCodeFromCommandFailureV0(protocol.stderr.String(), waitErr),
				Err:  waitErr,
			}
		}
		return err
	}
	return protocol.readResponseLockedV0(callCtx, requestID, out)
}

func (protocol *serverCodexAppServerPersistentCommandProtocolV0) ensureStartedLockedV0(ctx context.Context) error {
	if protocol.cmd != nil && protocol.stdin != nil && protocol.scanner != nil {
		return nil
	}
	commandPath := strings.TrimSpace(protocol.CommandPath)
	if commandPath == "" {
		commandPath = codexCommandPathV0()
	}
	args := append([]string(nil), protocol.Args...)
	if len(args) == 0 {
		args = []string{"app-server", "--stdio"}
	}
	cmd := exec.Command(commandPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	protocol.stderr.Reset()
	cmd.Stderr = &protocol.stderr
	if strings.TrimSpace(protocol.PathEnv) != "" {
		cmd.Env = append(os.Environ(), "PATH="+protocol.PathEnv)
	}
	if err := cmd.Start(); err != nil {
		return codexAppServerCallErrorV0{
			Code: codexAppServerIssueCodeFromCommandFailureV0(protocol.stderr.String(), err),
			Err:  err,
		}
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	protocol.cmd = cmd
	protocol.stdin = stdin
	protocol.scanner = scanner
	protocol.nextID = 2

	initLine, err := codexAppServerRPCMessageLineV0(codexAppServerInitializeRequestV0())
	if err != nil {
		_ = protocol.stopLockedV0()
		return err
	}
	if _, err := io.WriteString(protocol.stdin, initLine); err != nil {
		waitErr := protocol.stopLockedV0()
		if waitErr != nil {
			return codexAppServerCallErrorV0{
				Code: codexAppServerIssueCodeFromCommandFailureV0(protocol.stderr.String(), waitErr),
				Err:  waitErr,
			}
		}
		return err
	}
	var initResponse map[string]interface{}
	if err := protocol.readResponseLockedV0(ctx, 1, &initResponse); err != nil {
		_ = protocol.stopLockedV0()
		return err
	}
	initializedLine, err := codexAppServerRPCMessageLineV0(codexAppServerInitializedNotificationV0())
	if err != nil {
		_ = protocol.stopLockedV0()
		return err
	}
	if _, err := io.WriteString(protocol.stdin, initializedLine); err != nil {
		waitErr := protocol.stopLockedV0()
		if waitErr != nil {
			return codexAppServerCallErrorV0{
				Code: codexAppServerIssueCodeFromCommandFailureV0(protocol.stderr.String(), waitErr),
				Err:  waitErr,
			}
		}
		return err
	}
	return nil
}

func (protocol *serverCodexAppServerPersistentCommandProtocolV0) readResponseLockedV0(
	ctx context.Context,
	responseID int,
	out interface{},
) error {
	resultCh := make(chan error, 1)
	scanner := protocol.scanner
	go func() {
		resultCh <- decodeCodexAppServerRPCResponseScannerV0(scanner, responseID, out)
	}()
	select {
	case err := <-resultCh:
		return err
	case <-ctx.Done():
		_ = protocol.stopLockedV0()
		return codexAppServerCallErrorV0{Code: "codex_app_server_timeout", Err: ctx.Err()}
	}
}

func (protocol *serverCodexAppServerPersistentCommandProtocolV0) stopLockedV0() error {
	if protocol.stdin != nil {
		_ = protocol.stdin.Close()
	}
	var waitErr error
	if protocol.cmd != nil {
		if protocol.cmd.Process != nil {
			_ = protocol.cmd.Process.Kill()
		}
		waitErr = protocol.cmd.Wait()
	}
	protocol.cmd = nil
	protocol.stdin = nil
	protocol.scanner = nil
	protocol.nextID = 0
	return waitErr
}
