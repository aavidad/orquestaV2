package orquestaruntimecodexappserver

import (
	"strings"
	"time"
)

const (
	defaultCodexGoalTimeoutMSV0        = 90000
	CodexGoalBackendAppServerProxyV0   = codexGoalBackendAppServerProxyV0
	CodexGoalBackendAppServerTmuxV0    = codexGoalBackendAppServerTmuxV0
	CodexAppServerTmuxDirV0            = codexAppServerTmuxDirV0
	CodexAppServerTmuxMarkerFileV0     = codexAppServerTmuxMarkerFileV0
	CodexAppServerTmuxDefaultTimeoutV0 = codexAppServerTmuxDefaultTimeoutV0
	CodexAppServerTmuxMaxSocketPathV0  = codexAppServerTmuxMaxSocketPathV0
)

type CommandProtocolV0 = serverCodexAppServerCommandProtocolV0
type GoalBackendV0 = serverCodexAppServerGoalBackendV0
type GoalRuntimeV0 = serverCodexAppServerGoalRuntimeV0
type LazyTmuxProtocolV0 = serverCodexAppServerLazyTmuxProtocolV0
type ProbePortV0 = serverCodexAppServerProbePortV0
type ProtocolPortV0 = serverCodexAppServerProtocolPortV0
type TmuxBackendV0 = serverCodexAppServerTmuxBackendV0
type UnavailableGoalBackendV0 = serverCodexUnavailableGoalBackendV0
type WebSocketProtocolV0 = serverCodexAppServerWebSocketProtocolV0
type RPCResponseV0 = serverCodexAppServerRPCResponseV0
type ReadItemV0 = serverCodexAppServerReadItemV0
type ReadTurnV0 = serverCodexAppServerReadTurnV0
type ThreadGoalGetResponseV0 = serverCodexAppServerThreadGoalGetResponseV0
type ThreadGoalSetParamsV0 = serverCodexAppServerThreadGoalSetParamsV0
type ThreadGoalV0 = serverCodexAppServerThreadGoalV0
type ThreadReadResponseV0 = serverCodexAppServerThreadReadResponseV0
type ThreadReadV0 = serverCodexAppServerThreadReadV0
type ThreadStartParamsV0 = serverCodexAppServerThreadStartParamsV0
type ThreadStartResponseV0 = serverCodexAppServerThreadStartResponseV0
type ThreadStatusV0 = serverCodexAppServerThreadStatusV0
type ThreadV0 = serverCodexAppServerThreadV0
type TimestampV0 = serverCodexAppServerTimestampV0
type TurnStartParamsV0 = serverCodexAppServerTurnStartParamsV0
type TurnStartResponseV0 = serverCodexAppServerTurnStartResponseV0
type TurnV0 = serverCodexAppServerTurnV0

type ConfigV0 struct {
	RuntimeWorkDir string
	ProjectWorkDir string
}

func CodexAppServerIssueCodeForErrorV0(err error, fallback string) string {
	return codexAppServerIssueCodeForErrorV0(err, fallback)
}

func CodexAppServerIssueCodeFromCommandFailureV0(stderr string, err error) string {
	return codexAppServerIssueCodeFromCommandFailureV0(stderr, err)
}

func CodexAppServerIssueCodeFromLogFileV0(path string) string {
	return codexAppServerIssueCodeFromLogFileV0(path)
}

func CodexAppServerTmuxCodeHomePathV0(config ConfigV0) (string, error) {
	return codexAppServerTmuxCodeHomePathV0(config)
}

func CodexAppServerTmuxSessionNameV0(config ConfigV0) string {
	return codexAppServerTmuxSessionNameV0(config)
}

func CodexAppServerTmuxSocketPathV0(config ConfigV0) (string, error) {
	return codexAppServerTmuxSocketPathV0(config)
}

func CodexAppServerTmuxStartupTimeoutV0(timeout time.Duration) time.Duration {
	return codexAppServerTmuxStartupTimeoutV0(timeout)
}

func CodexAppServerTmuxLogPathV0(backend TmuxBackendV0) string {
	return backend.tmuxLogPathV0()
}

func CodexAppServerTmuxOwnerMarkerPathV0(backend TmuxBackendV0) string {
	return backend.tmuxOwnerMarkerPathV0()
}

func DiagnosticLogPathForProtocolV0(protocol ProtocolPortV0) string {
	if websocket, ok := protocol.(serverCodexAppServerWebSocketProtocolV0); ok {
		return strings.TrimSpace(websocket.DiagnosticLogPath)
	}
	if lazyTmux, ok := protocol.(serverCodexAppServerLazyTmuxProtocolV0); ok {
		return strings.TrimSpace(lazyTmux.Inner.DiagnosticLogPath)
	}
	return ""
}

func compactServerStackStringsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func compactStringsV0(values []string) []string {
	return compactServerStackStringsV0(values)
}

func firstNonEmptyServerStackV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func codexCommandPathV0() string {
	return "codex"
}
