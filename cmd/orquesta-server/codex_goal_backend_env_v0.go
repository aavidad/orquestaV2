package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

type serverCodexGoalBackendsV0 struct {
	AppGoal  serverCodexGoalBackendV0
	IdleGoal serverCodexGoalBackendV0
}

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
		websocketProtocol := serverCodexAppServerWebSocketProtocolV0{
			SocketPath: socketPath,
			Timeout:    time.Duration(codexGoalTimeoutMSFromEnvV0()) * time.Millisecond,
		}
		runtimeProtocol = websocketProtocol
		preflightProtocol = serverCodexAppServerWebSocketProtocolV0{
			SocketPath: socketPath,
			Timeout:    time.Duration(codexGoalPreflightTimeoutMSFromEnvV0()) * time.Millisecond,
		}
		tmuxBackend := serverCodexAppServerTmuxBackendV0{
			CommandPath:       runtimeConfig.CommandPath,
			PathEnv:           runtimeConfig.PathEnv,
			SocketPath:        socketPath,
			SessionName:       codexAppServerTmuxSessionNameV0(config),
			HomeDir:           runtimeConfig.HomeDir,
			CodeHomeDir:       tmuxCodeHomePath,
			SourceCodeHomeDir: runtimeConfig.CodeHomeDir,
			Timeout:           codexAppServerTmuxStartupTimeoutV0(time.Duration(codexGoalPreflightTimeoutMSFromEnvV0()) * time.Millisecond),
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
	client := serverCodexAppServerGoalBackendV0{
		Protocol:        runtimeProtocol,
		CWD:             firstNonEmptyServerStackV0(workDir, config.ProjectWorkDir),
		Model:           runtimeConfig.Model,
		ReasoningEffort: runtimeConfig.ReasoningEffort,
		Sandbox:         runtimeConfig.Sandbox,
		ApprovalPolicy:  runtimeConfig.ApprovalPolicy,
		Timeout:         time.Duration(codexGoalTimeoutMSFromEnvV0()) * time.Millisecond,
		Runtime:         &serverCodexAppServerGoalRuntimeV0{},
	}
	return serverCodexGoalBackendV0{Starter: client, Observer: client}, nil
}

func codexGoalBackendFromEnvV0() string {
	return strings.TrimSpace(os.Getenv(envCodexGoalBackendV0))
}

func codexGoalBackendOperationalFromEnvV0() bool {
	backend := codexGoalBackendFromEnvV0()
	if backend == codexGoalBackendAppServerTmuxV0 {
		return true
	}
	return backend == codexGoalBackendAppServerProxyV0 && codexGoalBackendProxyDiagnosticAllowedV0()
}

func codexGoalBackendProxyDiagnosticAllowedV0() bool {
	return boolEnvOrDefaultV0(envAllowAppServerProxyDiagnosticV0, false)
}

func codexGoalBackendArgsV0(backend string) []string {
	switch strings.TrimSpace(backend) {
	default:
		return []string{"app-server", "proxy"}
	}
}

func codexGoalBackendProxyArgsForSocketV0(socketPath string) []string {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return []string{"app-server", "proxy"}
	}
	return []string{"app-server", "proxy", "--sock", socketPath}
}
