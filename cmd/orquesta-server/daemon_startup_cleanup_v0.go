package main

import (
	"context"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config orquestaserver.ConfigV0, pid int) {
	if pid > 0 && processAliveV0(pid) {
		return
	}
	cleanupCodexAppServerTmuxAfterStartupFailureV0(config)
}

func cleanupCodexAppServerTmuxAfterStartupFailureV0(config orquestaserver.ConfigV0) {
	projectConfig := projectConfigFromServerConfigBestEffortV0(config)
	if codexGoalBackendFromProjectConfigFileV0(projectConfig) != codexGoalBackendAppServerTmuxV0 {
		return
	}
	socketPath, err := codexAppServerTmuxSocketPathV0(config)
	if err != nil {
		return
	}
	codeHomePath, err := codexAppServerTmuxCodeHomePathV0(config)
	if err != nil {
		return
	}
	runtimeConfig := codexRuntimeEnvConfigFromEnvV0()
	timeout := time.Duration(codexGoalPreflightTimeoutMSFromProjectConfigFileV0(projectConfig)) * time.Millisecond
	if timeout <= 0 {
		timeout = codexAppServerTmuxDefaultTimeoutV0
	}
	backend := serverCodexAppServerTmuxBackendV0{
		CommandPath:       runtimeConfig.CommandPath,
		PathEnv:           runtimeConfig.PathEnv,
		SocketPath:        socketPath,
		SessionName:       codexAppServerTmuxSessionNameV0(config),
		HomeDir:           runtimeConfig.HomeDir,
		CodeHomeDir:       codeHomePath,
		RuntimeWorkDir:    config.RuntimeWorkDir,
		SourceCodeHomeDir: runtimeConfig.CodeHomeDir,
		Timeout:           timeout,
	}
	_, _ = backend.CleanupActiveShutdownWorkV0(context.Background(), orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0{
		CleanupGoalBackends: true,
		EvidenceRefs:        []string{"evidence-ref-orquesta-startup-goal-backend-cleanup"},
	})
}
