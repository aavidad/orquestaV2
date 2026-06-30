package main

import (
	"context"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config orquestaserver.ConfigV0, pid int) {
	if pid > 0 && processAliveV0(pid) {
		return
	}
	cleanupCodexAppServerTmuxAfterStartupFailureV0(config)
}

func cleanupCodexAppServerTmuxAfterStartupFailureV0(config orquestaserver.ConfigV0) {
	if codexGoalBackendFromEnvV0() != codexGoalBackendAppServerTmuxV0 {
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
	timeout := time.Duration(codexGoalPreflightTimeoutMSFromEnvV0()) * time.Millisecond
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
		SourceCodeHomeDir: runtimeConfig.CodeHomeDir,
		Timeout:           timeout,
	}
	_ = backend.ShutdownV0(context.Background())
}
