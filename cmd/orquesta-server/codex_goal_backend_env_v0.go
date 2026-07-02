package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

type serverCodexGoalBackendsV0 struct {
	AppGoal  serverCodexGoalBackendV0
	IdleGoal serverCodexGoalBackendV0
}

const serverCodexGoalBackendDegradedDiagnosticCodeV0 = "codex_goal_backend_degraded"

func serverCodexGoalBackendFromEnvV0(
	config orquestaserver.ConfigV0,
) (serverCodexGoalBackendV0, error) {
	return serverCodexGoalBackendFromEnvForWorkDirV0(
		config,
		firstNonEmptyServerStackV0(config.IdleSelfImprovementProjectWorkDir, config.ProjectWorkDir),
	)
}

func serverCodexGoalBackendsFromEnvV0(
	config orquestaserver.ConfigV0,
) (serverCodexGoalBackendsV0, error) {
	appGoal, err := serverCodexGoalBackendFromEnvForWorkDirV0(config, config.ProjectWorkDir)
	if err != nil {
		return serverCodexGoalBackendsV0{}, err
	}
	idleGoal, err := serverCodexGoalBackendFromEnvForWorkDirV0(
		config,
		firstNonEmptyServerStackV0(config.IdleSelfImprovementProjectWorkDir, config.ProjectWorkDir),
	)
	if err != nil {
		return serverCodexGoalBackendsV0{}, err
	}
	return serverCodexGoalBackendsV0{
		AppGoal:  appGoal,
		IdleGoal: idleGoal,
	}, nil
}

func serverCodexGoalBackendFromEnvForWorkDirV0(
	config orquestaserver.ConfigV0,
	workDir string,
) (serverCodexGoalBackendV0, error) {
	backend := codexGoalBackendFromEnvV0()
	if backend == "" {
		return serverCodexGoalBackendV0{}, nil
	}
	if backend != codexGoalBackendAppServerProxyV0 &&
		backend != codexGoalBackendAppServerTmuxV0 {
		return serverCodexGoalBackendV0{}, fmt.Errorf("codex_goal_backend_no_soportado:%s", backend)
	}
	if backend == codexGoalBackendAppServerProxyV0 && !codexGoalBackendProxyDiagnosticAllowedV0() {
		return serverCodexGoalBackendV0{}, fmt.Errorf("codex_goal_backend_proxy_diagnostic_opt_in_required:%s", envAllowAppServerProxyDiagnosticV0)
	}
	if backend == codexGoalBackendAppServerProxyV0 {
		return serverCodexGoalBackendV0{}, fmt.Errorf("codex_goal_backend_proxy_diagnostic_not_operational")
	}
	runtimeConfig := codexRuntimeEnvConfigFromEnvV0()
	commandProtocol := serverCodexAppServerCommandProtocolV0{
		CommandPath: runtimeConfig.CommandPath,
		Args:        codexGoalBackendArgsV0(backend),
		PathEnv:     runtimeConfig.PathEnv,
		Timeout:     time.Duration(codexGoalTimeoutMSFromEnvV0()) * time.Millisecond,
	}
	var runtimeProtocol serverCodexAppServerProtocolPortV0 = commandProtocol
	commandPreflightProtocol := commandProtocol
	commandPreflightProtocol.Timeout = time.Duration(codexGoalPreflightTimeoutMSFromEnvV0()) * time.Millisecond
	var preflightProtocol serverCodexAppServerProbePortV0 = commandPreflightProtocol
	var shutdownHook orquestaserver.RuntimeShutdownHookPortV0
	codeHomePath := ""
	authIssueCode := ""
	if backend == codexGoalBackendAppServerTmuxV0 {
		socketPath, err := codexAppServerTmuxSocketPathV0(config)
		if err != nil {
			degraded := serverCodexUnavailableGoalBackendV0{
				IssueCode: codexAppServerIssueCodeForErrorV0(err, "codex_app_server_tmux_unavailable"),
			}
			return serverCodexGoalBackendV0{Starter: degraded, Observer: degraded}, nil
		}
		tmuxCodeHomePath, err := codexAppServerTmuxCodeHomePathV0(config)
		if err != nil {
			degraded := serverCodexUnavailableGoalBackendV0{
				IssueCode: codexAppServerIssueCodeForErrorV0(err, "codex_app_server_tmux_unavailable"),
			}
			return serverCodexGoalBackendV0{Starter: degraded, Observer: degraded}, nil
		}
		codeHomePath = tmuxCodeHomePath
		tmuxBackend := serverCodexAppServerTmuxBackendV0{
			CommandPath:       runtimeConfig.CommandPath,
			PathEnv:           runtimeConfig.PathEnv,
			SocketPath:        socketPath,
			SessionName:       codexAppServerTmuxSessionNameV0(config),
			HomeDir:           runtimeConfig.HomeDir,
			CodeHomeDir:       tmuxCodeHomePath,
			RuntimeWorkDir:    config.RuntimeWorkDir,
			ProjectWorkDir:    firstNonEmptyServerStackV0(workDir, config.ProjectWorkDir),
			SourceCodeHomeDir: runtimeConfig.CodeHomeDir,
			Timeout:           codexAppServerTmuxStartupTimeoutV0(time.Duration(codexGoalPreflightTimeoutMSFromEnvV0()) * time.Millisecond),
		}
		shutdownTmuxBackend := tmuxBackend
		shutdownTmuxBackend.Timeout = time.Duration(codexGoalPreflightTimeoutMSFromEnvV0()) * time.Millisecond
		shutdownHook = shutdownTmuxBackend
		authIssueCode = codexAppServerAuthIssueCodeFromDirsV0(runtimeConfig.CodeHomeDir, tmuxCodeHomePath)
		if authIssueCode != "" {
			degraded := serverCodexUnavailableGoalBackendV0{IssueCode: authIssueCode}
			return serverCodexGoalBackendV0{Starter: degraded, Observer: degraded, ShutdownHook: shutdownHook}, nil
		}
		websocketProtocol := serverCodexAppServerWebSocketProtocolV0{
			SocketPath:        socketPath,
			Timeout:           time.Duration(codexGoalTimeoutMSFromEnvV0()) * time.Millisecond,
			DiagnosticLogPath: tmuxBackend.tmuxLogPathV0(),
		}
		runtimeProtocol = websocketProtocol
		preflightProtocol = serverCodexAppServerWebSocketProtocolV0{
			SocketPath:        socketPath,
			Timeout:           time.Duration(codexGoalPreflightTimeoutMSFromEnvV0()) * time.Millisecond,
			DiagnosticLogPath: tmuxBackend.tmuxLogPathV0(),
		}
		if err := tmuxBackend.EnsureV0(context.Background(), preflightProtocol); err != nil {
			degraded := serverCodexUnavailableGoalBackendV0{
				IssueCode: codexAppServerIssueCodeForErrorV0(err, "codex_app_server_tmux_unavailable"),
			}
			return serverCodexGoalBackendV0{Starter: degraded, Observer: degraded}, nil
		}
	}
	if err := preflightProtocol.ProbeV0(context.Background()); err != nil {
		degraded := serverCodexUnavailableGoalBackendV0{
			IssueCode: codexAppServerIssueCodeForErrorV0(err, "codex_app_server_unavailable"),
		}
		return serverCodexGoalBackendV0{Starter: degraded, Observer: degraded}, nil
	}
	if authIssueCode = codexAppServerAuthIssueCodeV0(codeHomePath); authIssueCode != "" {
		degraded := serverCodexUnavailableGoalBackendV0{IssueCode: authIssueCode}
		return serverCodexGoalBackendV0{Starter: degraded, Observer: degraded, ShutdownHook: shutdownHook}, nil
	}
	client := serverCodexAppServerGoalBackendV0{
		Protocol:          runtimeProtocol,
		CWD:               firstNonEmptyServerStackV0(workDir, config.ProjectWorkDir),
		DiagnosticLogPath: diagnosticLogPathForCodexAppServerProtocolV0(runtimeProtocol),
		AuthIssueCode:     authIssueCode,
		Model:             runtimeConfig.Model,
		ReasoningEffort:   runtimeConfig.ReasoningEffort,
		Sandbox:           runtimeConfig.Sandbox,
		ApprovalPolicy:    runtimeConfig.ApprovalPolicy,
		Timeout:           time.Duration(codexGoalTimeoutMSFromEnvV0()) * time.Millisecond,
		Runtime:           &serverCodexAppServerGoalRuntimeV0{},
	}
	return serverCodexGoalBackendV0{Starter: client, Observer: client, ShutdownHook: shutdownHook}, nil
}

func diagnosticLogPathForCodexAppServerProtocolV0(protocol serverCodexAppServerProtocolPortV0) string {
	if websocket, ok := protocol.(serverCodexAppServerWebSocketProtocolV0); ok {
		return strings.TrimSpace(websocket.DiagnosticLogPath)
	}
	return ""
}

func codexAppServerAuthIssueCodeV0(codeHomeDir string) string {
	return codexAppServerAuthIssueCodeFromDirsV0(codeHomeDir)
}

func codexAppServerAuthIssueCodeFromDirsV0(codeHomeDirs ...string) string {
	if codexAppServerAuthEnvPresentV0() {
		return ""
	}
	checked := false
	for _, codeHomeDir := range codeHomeDirs {
		codeHomeDir = strings.TrimSpace(codeHomeDir)
		if codeHomeDir == "" {
			continue
		}
		checked = true
		info, err := os.Stat(filepath.Join(codeHomeDir, "auth.json"))
		if err == nil && !info.IsDir() && info.Size() > 0 {
			return ""
		}
	}
	if !checked {
		return ""
	}
	return "codex_app_server_auth_missing"
}

func codexAppServerAuthEnvPresentV0() bool {
	for _, name := range []string{"OPENAI_API_KEY", "CODEX_API_KEY"} {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			return true
		}
	}
	return false
}

func codexGoalBackendFromEnvV0() string {
	return strings.TrimSpace(os.Getenv(envCodexGoalBackendV0))
}

func codexGoalBackendOperationalFromEnvV0() bool {
	return codexGoalBackendFromEnvV0() == codexGoalBackendAppServerTmuxV0
}

func serverConfigWithCodexGoalBackendDiagnosticsV0(
	config orquestaserver.ConfigV0,
	backends serverCodexGoalBackendsV0,
) orquestaserver.ConfigV0 {
	diagnostics := append([]orquestaserver.ServerDiagnosticV0(nil), config.EffectiveConfig.Diagnostics...)
	diagnostics = append(diagnostics, serverCodexGoalBackendDiagnosticV0("app_goal", backends.AppGoal)...)
	diagnostics = append(diagnostics, serverCodexGoalBackendDiagnosticV0("idle_goal", backends.IdleGoal)...)
	config.EffectiveConfig.Diagnostics = diagnostics
	config.EffectiveConfig = orquestaserver.NormalizeServerEffectiveConfigV0(config.EffectiveConfig)
	return config
}

func serverCodexGoalBackendDiagnosticV0(
	scope string,
	backend serverCodexGoalBackendV0,
) []orquestaserver.ServerDiagnosticV0 {
	issueCode := serverCodexGoalBackendUnavailableIssueCodeV0(backend)
	if issueCode == "" {
		return nil
	}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		scope = "codex_goal"
	}
	return []orquestaserver.ServerDiagnosticV0{{
		Code:    serverCodexGoalBackendDegradedDiagnosticCodeV0,
		Scope:   scope,
		Message: serverCodexGoalBackendDiagnosticMessageV0(issueCode),
		EvidenceRefs: []string{
			"evidence-ref-server-codex-goal-backend-degraded-" + scope,
		},
	}}
}

func serverCodexGoalBackendUnavailableIssueCodeV0(backend serverCodexGoalBackendV0) string {
	if unavailable, ok := backend.Starter.(serverCodexUnavailableGoalBackendV0); ok {
		return strings.TrimSpace(unavailable.IssueCode)
	}
	if unavailable, ok := backend.Observer.(serverCodexUnavailableGoalBackendV0); ok {
		return strings.TrimSpace(unavailable.IssueCode)
	}
	return ""
}

func serverCodexGoalBackendDiagnosticMessageV0(issueCode string) string {
	issueCode = strings.TrimSpace(issueCode)
	if issueCode == "codex_app_server_wrapper_stdio_failed" {
		return "codex goal backend degradado: codex_app_server_wrapper_stdio_failed; accion: configura ORQUESTA_CODEX_COMMAND con el binario nativo de Codex y reinicia"
	}
	if issueCode == "codex_app_server_provider_unauthorized" {
		return "codex goal backend degradado: codex_app_server_provider_unauthorized; accion: revisa autenticacion de Codex/OpenAI en el CODEX_HOME aislado y reinicia"
	}
	if issueCode == "codex_app_server_auth_missing" {
		return "codex goal backend degradado: codex_app_server_auth_missing; accion: ejecuta login de Codex o proyecta auth.json al CODEX_HOME aislado y reinicia"
	}
	if issueCode == "" {
		issueCode = "codex_app_server_unavailable"
	}
	return "codex goal backend degradado: " + issueCode
}

func codexGoalBackendProxyDiagnosticAllowedV0() bool {
	return boolEnvOrDefaultV0(envAllowAppServerProxyDiagnosticV0, false)
}

func codexGoalBackendArgsV0(backend string) []string {
	switch strings.TrimSpace(backend) {
	case codexGoalBackendAppServerTmuxV0:
		return nil
	case codexGoalBackendAppServerProxyV0:
		return []string{"app-server", "proxy"}
	default:
		return nil
	}
}

func codexGoalBackendProxyArgsForSocketV0(socketPath string) []string {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return []string{"app-server", "proxy"}
	}
	return []string{"app-server", "proxy", "--sock", socketPath}
}
