package main

import (
	"context"
	"fmt"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

type serverDaemonProcessIdentityV0 struct {
	PID       int
	StartRef  string
	GroupID   int
	SessionID int
}

var (
	serverStartupCleanupCooperativeTimeoutV0 = 500 * time.Millisecond
	serverStartupCleanupTerminateTimeoutV0   = 2 * time.Second
	serverStartupCleanupSignalProcessV0      = signalServerDaemonProcessIdentityV0
	serverStartupCleanupTerminateGroupV0     = terminateServerDaemonProcessGroupIdentityV0
	serverStartupCleanupKillGroupV0          = killServerDaemonProcessGroupIdentityV0
)

func cleanupDetachedDaemonAfterStartupFailureV0(
	config orquestaserver.ConfigV0,
	identity serverDaemonProcessIdentityV0,
) error {
	if identity.PID <= 0 || identity.GroupID <= 0 || identity.SessionID <= 0 || identity.StartRef == "" {
		return fmt.Errorf("daemon_process_identity_invalid")
	}
	active, err := serverDaemonProcessGroupActiveV0(identity)
	if err != nil {
		return err
	}
	if active {
		leaderAlive, identityErr := serverDaemonProcessIdentityAliveV0(identity)
		if identityErr != nil {
			return identityErr
		}
		if leaderAlive {
			if err := serverStartupCleanupSignalProcessV0(identity); err != nil {
				return err
			}
			if inactive, waitErr := waitForServerDaemonProcessGroupInactiveV0(identity, serverStartupCleanupCooperativeTimeoutV0); waitErr != nil {
				return waitErr
			} else if inactive {
				return cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config, identity)
			}
		}
		if err := serverStartupCleanupTerminateGroupV0(identity); err != nil {
			return err
		}
		if inactive, waitErr := waitForServerDaemonProcessGroupInactiveV0(identity, serverStartupCleanupTerminateTimeoutV0); waitErr != nil {
			return waitErr
		} else if !inactive {
			if err := serverStartupCleanupKillGroupV0(identity); err != nil {
				return err
			}
			if inactive, waitErr = waitForServerDaemonProcessGroupInactiveV0(identity, serverStartupCleanupTerminateTimeoutV0); waitErr != nil {
				return waitErr
			} else if !inactive {
				return fmt.Errorf("daemon_process_group_still_active")
			}
		}
	}
	return cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(config, identity)
}

func waitForServerDaemonProcessGroupInactiveV0(
	identity serverDaemonProcessIdentityV0,
	timeout time.Duration,
) (bool, error) {
	if timeout <= 0 {
		timeout = time.Second
	}
	deadline := time.Now().Add(timeout)
	for {
		active, err := serverDaemonProcessGroupActiveV0(identity)
		if err != nil || !active {
			return !active, err
		}
		if !time.Now().Before(deadline) {
			return false, nil
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func stoppedServerDaemonIdentityV0(pid int) serverDaemonProcessIdentityV0 {
	return serverDaemonProcessIdentityV0{PID: pid, GroupID: pid, SessionID: pid}
}

func cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(
	config orquestaserver.ConfigV0,
	identity serverDaemonProcessIdentityV0,
) error {
	if identity.PID <= 0 || identity.GroupID <= 0 || identity.SessionID <= 0 {
		return fmt.Errorf("daemon_process_identity_invalid")
	}
	active, err := serverDaemonProcessGroupActiveV0(identity)
	if err != nil {
		return err
	}
	if active {
		return fmt.Errorf("daemon_process_group_still_active")
	}
	return cleanupCodexAppServerTmuxAfterStartupFailureV0(config)
}

func cleanupCodexAppServerTmuxAfterStartupFailureV0(config orquestaserver.ConfigV0) error {
	projectConfig := projectConfigFromServerConfigBestEffortV0(config)
	if codexGoalBackendFromProjectConfigFileV0(projectConfig) != codexGoalBackendAppServerTmuxV0 {
		return nil
	}
	socketPath, err := codexAppServerTmuxSocketPathV0(config)
	if err != nil {
		return err
	}
	codeHomePath, err := codexAppServerTmuxCodeHomePathV0(config)
	if err != nil {
		return err
	}
	runtimeConfig := codexRuntimeEnvConfigFromEnvV0()
	timeout := time.Duration(codexGoalPreflightTimeoutMSFromProjectConfigFileV0(projectConfig)) * time.Millisecond
	if timeout <= 0 {
		timeout = codexAppServerTmuxDefaultTimeoutV0
	}
	if timeout > serverStartupCleanupTerminateTimeoutV0 {
		timeout = serverStartupCleanupTerminateTimeoutV0
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
	cleanupCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return backend.CleanupOwnedGenerationAfterStartupFailureV0(cleanupCtx)
}
