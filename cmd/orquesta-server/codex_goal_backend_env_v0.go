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
	if backend != codexGoalBackendAppServerProxyV0 && backend != codexGoalBackendAppServerStdioV0 {
		return serverCodexGoalBackendV0{}, fmt.Errorf("codex_goal_backend_no_soportado:%s", backend)
	}
	runtimeConfig := codexRuntimeEnvConfigFromEnvV0()
	protocol := serverCodexAppServerCommandProtocolV0{
		CommandPath: runtimeConfig.CommandPath,
		Args:        codexGoalBackendArgsV0(backend),
		PathEnv:     runtimeConfig.PathEnv,
		Timeout:     time.Duration(codexGoalTimeoutMSFromEnvV0()) * time.Millisecond,
	}
	preflightProtocol := protocol
	preflightProtocol.Timeout = time.Duration(codexGoalPreflightTimeoutMSFromEnvV0()) * time.Millisecond
	if err := preflightProtocol.ProbeV0(context.Background()); err != nil {
		degraded := serverCodexUnavailableGoalBackendV0{
			IssueCode: codexAppServerIssueCodeForErrorV0(err, "codex_app_server_unavailable"),
		}
		return serverCodexGoalBackendV0{Starter: degraded, Observer: degraded}, nil
	}
	var runtimeProtocol serverCodexAppServerProtocolPortV0 = protocol
	if backend == codexGoalBackendAppServerStdioV0 {
		runtimeProtocol = &serverCodexAppServerPersistentCommandProtocolV0{
			CommandPath: runtimeConfig.CommandPath,
			Args:        codexGoalBackendArgsV0(backend),
			PathEnv:     runtimeConfig.PathEnv,
			Timeout:     time.Duration(codexGoalTimeoutMSFromEnvV0()) * time.Millisecond,
		}
	}
	client := serverCodexAppServerGoalBackendV0{
		Protocol:        runtimeProtocol,
		CWD:             firstNonEmptyServerStackV0(workDir, config.ProjectWorkDir),
		Model:           runtimeConfig.Model,
		ReasoningEffort: runtimeConfig.ReasoningEffort,
		Sandbox:         runtimeConfig.Sandbox,
		ApprovalPolicy:  runtimeConfig.ApprovalPolicy,
	}
	return serverCodexGoalBackendV0{Starter: client, Observer: client}, nil
}

func codexGoalBackendFromEnvV0() string {
	return strings.TrimSpace(os.Getenv(envCodexGoalBackendV0))
}

func codexGoalBackendArgsV0(backend string) []string {
	switch strings.TrimSpace(backend) {
	case codexGoalBackendAppServerStdioV0:
		return []string{"app-server", "--stdio"}
	default:
		return []string{"app-server", "proxy"}
	}
}
