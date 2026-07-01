package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexAppServerIssueCodeForErrorV0ClasificaResetStdioV0(t *testing.T) {
	message := `Node.js[2796823]: void node::ResetStdio() at ../src/node.cc:751
Assertion failed: !(err != 0) || (err == -1 && (*__errno_location ()) == 1)`

	if got := codexAppServerIssueCodeForErrorV0(errors.New(message), "fallback"); got != "codex_app_server_wrapper_stdio_failed" {
		t.Fatalf("code=%q", got)
	}
}

func TestCodexAppServerIssueCodeFromLogFileV0ClasificaResetStdioV0(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "orquesta-goal.log")
	raw := strings.Repeat("x", codexAppServerDiagnosticLogMaxBytesV0+64) +
		"\nvoid node::ResetStdio() at ../src/node.cc:751\n"
	if err := os.WriteFile(logPath, []byte(raw), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}

	if got := codexAppServerIssueCodeFromLogFileV0(logPath); got != "codex_app_server_wrapper_stdio_failed" {
		t.Fatalf("code=%q", got)
	}
}

func TestCodexAppServerIssueCodeFromLogFileV0ClasificaUnauthorizedProviderV0(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "orquesta-goal.log")
	raw := "failed to connect to websocket: HTTP error: 401 Unauthorized, url: wss://api.openai.com/v1/responses\n"
	if err := os.WriteFile(logPath, []byte(raw), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}

	if got := codexAppServerIssueCodeFromLogFileV0(logPath); got != "codex_app_server_provider_unauthorized" {
		t.Fatalf("code=%q", got)
	}
}

func TestCodexAppServerWebSocketProtocolV0UsaLogResetStdioV0(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "orquesta-goal.log")
	if err := os.WriteFile(logPath, []byte("Node.js: void node::ResetStdio() at ../src/node.cc:751\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	protocol := serverCodexAppServerWebSocketProtocolV0{DiagnosticLogPath: logPath}

	err := protocol.wrapWebSocketErrorV0(errors.New("EOF"))

	var callErr codexAppServerCallErrorV0
	if !errors.As(err, &callErr) || callErr.Code != "codex_app_server_wrapper_stdio_failed" {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestCodexAppServerTmuxStartupFailureV0UsaLogResetStdioV0(t *testing.T) {
	root := t.TempDir()
	socketPath := filepath.Join(root, "runtime", codexAppServerTmuxDirV0, "goal.sock")
	backend := serverCodexAppServerTmuxBackendV0{
		SocketPath:  socketPath,
		SessionName: "orquesta-goal-resetstdio",
	}
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o700); err != nil {
		t.Fatalf("mkdir runtime: %v", err)
	}
	if err := os.WriteFile(backend.tmuxLogPathV0(), []byte("void node::ResetStdio() at ../src/node.cc:751\n"), 0o600); err != nil {
		t.Fatalf("write tmux log: %v", err)
	}

	err := backend.tmuxStartupFailureV0("codex_app_server_tmux_session_exited", errors.New("session exited"))

	var callErr codexAppServerCallErrorV0
	if !errors.As(err, &callErr) || callErr.Code != "codex_app_server_wrapper_stdio_failed" {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestServerCodexGoalBackendDiagnosticMessageV0RecomiendaBinarioNativo(t *testing.T) {
	message := serverCodexGoalBackendDiagnosticMessageV0("codex_app_server_wrapper_stdio_failed")

	if !strings.Contains(message, "ORQUESTA_CODEX_COMMAND") ||
		!strings.Contains(message, "binario nativo") {
		t.Fatalf("message=%q", message)
	}
}

func TestServerCodexGoalBackendDiagnosticMessageV0RecomiendaAuthAisladaV0(t *testing.T) {
	message := serverCodexGoalBackendDiagnosticMessageV0("codex_app_server_provider_unauthorized")

	if !strings.Contains(message, "CODEX_HOME") ||
		!strings.Contains(message, "autenticacion") {
		t.Fatalf("message=%q", message)
	}
}
