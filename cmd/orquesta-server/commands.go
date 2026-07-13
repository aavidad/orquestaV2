package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func runMain(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "comando requerido")
		return 2
	}
	command := args[0]
	commandArgs := args[1:]
	if serverCommandRequiresExactIdentityV0(command) {
		if code := validateServerBroadLaunchIdentityV0(commandArgs); code != "" {
			_, _ = fmt.Fprintf(stderr, "orquesta-server %s: reason_code=%s\n", command, code)
			return 1
		}
	}
	switch command {
	case "run":
		return runServerCommandV0(commandArgs, stdout, stderr)
	case "start":
		return startServerCommandV0(commandArgs, stdout, stderr)
	case "status":
		return statusServerCommandV0(commandArgs, stdout, stderr)
	case "run-status":
		return runStatusCommandV0(commandArgs, stdout, stderr)
	case "stop":
		return stopServerCommandV0(commandArgs, stdout, stderr)
	case "opes-drain-once":
		return opesDrainOnceCommandV0(stdout, stderr)
	case "opes-temario-cycle":
		return opesTemarioCycleCommandV0(stdout, stderr)
	case "mcp-real-smoke":
		return mcpRealSmokeCommandV0(stdout, stderr)
	case "mcp-stdio":
		return mcpStdioCommandV0(commandArgs, os.Stdin, stdout, stderr)
	case "codex-launch-wave":
		return codexLaunchWaveCommandV0(commandArgs, stdout, stderr)
	case "codex-launch-director-wave":
		return codexLaunchDirectorWaveCommandV0(commandArgs, stdout, stderr)
	case "codex-wave-status":
		return codexWaveStatusCommandV0(commandArgs, stdout, stderr)
	case "codex-wave-stop":
		return codexWaveStopCommandV0(commandArgs, stdout, stderr)
	case "codex-wave-tail":
		return codexWaveTailCommandV0(commandArgs, stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "comando no soportado: %s\n", command)
		return 2
	}
}

func serverCommandRequiresExactIdentityV0(command string) bool {
	switch strings.TrimSpace(command) {
	case "codex-launch-wave", "codex-launch-director-wave":
		return true
	default:
		return false
	}
}

func validateServerBroadLaunchIdentityV0(args []string) string {
	preflight, err := serverIdentityPreflightFromEnvV0("")
	if err != nil || preflight.Issue != nil {
		return serverWorkLaunchDegradedIdentityV0
	}
	if projectDir := serverCommandProjectDirArgV0(args); projectDir != "" {
		canonicalProjectDir, issue := canonicalServerWorkdirV0(projectDir)
		if issue != nil || canonicalProjectDir != preflight.ProjectWorkDir {
			return serverWorkLaunchDegradedIdentityV0
		}
	}
	return ""
}

func serverCommandProjectDirArgV0(args []string) string {
	for index, arg := range args {
		if strings.HasPrefix(arg, "--project-dir=") {
			return strings.TrimSpace(strings.TrimPrefix(arg, "--project-dir="))
		}
		if arg == "--project-dir" && index+1 < len(args) {
			return strings.TrimSpace(args[index+1])
		}
	}
	return ""
}

type serverCommandConfigOptionsV0 struct {
	ConfigPath string
}

func parseServerCommandConfigOptionsV0(command string, args []string) (serverCommandConfigOptionsV0, error) {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "ruta a orquesta.config.json canonico")
	if err := flags.Parse(args); err != nil {
		return serverCommandConfigOptionsV0{}, fmt.Errorf("%s_args_invalid", command)
	}
	if flags.NArg() != 0 {
		return serverCommandConfigOptionsV0{}, fmt.Errorf("%s_args_invalid", command)
	}
	return serverCommandConfigOptionsV0{ConfigPath: strings.TrimSpace(*configPath)}, nil
}

func runServerCommandV0(args []string, _ io.Writer, stderr io.Writer) int {
	options, err := parseServerCommandConfigOptionsV0("run", args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server run: %v\n", err)
		return 2
	}
	preflight, err := serverIdentityPreflightFromEnvV0(options.ConfigPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "orquesta-server: reason_code=identity_preflight_failed")
		return 1
	}
	if preflight.Issue != nil {
		return runDegradedIdentityServerV0(orquestaserver.ConfigV0{
			Addr:            degradedIdentityServerAddrFromEnvV0(),
			ProjectWorkDir:  preflight.ProjectWorkDir,
			RuntimeIdentity: preflight.RuntimeIdentity,
		}, preflight.Issue.Code, stderr)
	}
	if err := validateCodexCommandAvailableV0(); err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server: %v\n", err)
		return 1
	}
	serverConfig, err := serverConfigFromEnvWithProjectConfigPathV0(options.ConfigPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server: %v\n", err)
		return 1
	}
	defer releaseServerProjectConfigSnapshotV0(serverConfig)
	if err := applyServerDetailRailsRuntimeDefaultsV0(); err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server: detail_rails_env_default_failed: %v\n", err)
		return 1
	}
	serverConfig.RuntimeIdentity = serverRuntimeIdentityWithWorktreeEvidenceV0(
		preflight.RuntimeIdentity,
		serverConfig.ProjectWorkDir,
	)
	runtime, err := buildRuntimeFromConfigV0(serverConfig)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server: %v\n", err)
		return 1
	}
	signalController := newServerSignalControllerV0(context.Background(), runtime)
	defer signalController.stopNotificationsV0()
	ctx := signalController.contextV0()
	bridgeConfig, err := opesBridgeLoopConfigFromServerConfigV0(serverConfig, "http://"+serverConfig.Addr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server opes-bridge blocked: %v\n", err)
		markExternalBridgeConfigBlockedV0(ctx, runtime, err)
	} else {
		bridgeConfig.Loop.Observer = externalBridgeRuntimeObserverV0(runtime)
	}
	registryFinalPkgConfig, err := opesRegistryFinalPkgLoopConfigFromEnvV0(serverConfig, "http://"+serverConfig.Addr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server opes-registry-finalpkg blocked: %v\n", err)
		markExternalBridgeConfigBlockedV0(ctx, runtime, err)
	} else {
		registryFinalPkgConfig.Loop.Observer = externalBridgeRuntimeObserverV0(runtime)
	}
	codeContextWatchdogConfig, err := serverCodeContextToolWatchdogLoopConfigFromServerConfigV0(serverConfig)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server code-context-watchdog blocked: %v\n", err)
	} else {
		codeContextWatchdogConfig.Loop.Observer = externalBridgeRuntimeObserverV0(runtime)
	}
	bridgeDone := runOPESBridgeLoopAsyncV0(ctx, bridgeConfig, stderr, runOPESDrainOnceV0)
	registryFinalPkgDone := runOPESRegistryFinalPkgLoopAsyncV0(ctx, registryFinalPkgConfig, stderr, runOPESRegistryFinalPkgOnceV0)
	codeContextWatchdogDone := runServerCodeContextToolWatchdogLoopAsyncV0(ctx, codeContextWatchdogConfig, stderr)
	runErr := runtime.RunWithShutdownCauseV0(ctx, signalController.shutdownCauseV0)
	signalController.stopNotificationsV0()
	bridgeOK := waitOPESBridgeLoopDoneV0(context.Background(), bridgeConfig, bridgeDone)
	registryFinalPkgOK := waitOPESRegistryFinalPkgLoopDoneV0(context.Background(), registryFinalPkgConfig, registryFinalPkgDone)
	codeContextWatchdogOK := waitServerCodeContextToolWatchdogLoopDoneV0(context.Background(), codeContextWatchdogConfig, codeContextWatchdogDone)
	if runErr != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server: %v\n", runErr)
		return 1
	}
	if !bridgeOK {
		_, _ = fmt.Fprintln(stderr, "orquesta-server opes-bridge: shutdown_timeout")
		return 1
	}
	if !registryFinalPkgOK {
		_, _ = fmt.Fprintln(stderr, "orquesta-server opes-registry-finalpkg: shutdown_timeout")
		return 1
	}
	if !codeContextWatchdogOK {
		_, _ = fmt.Fprintln(stderr, "orquesta-server code-context-watchdog: shutdown_timeout")
		return 1
	}
	return 0
}

func runDegradedIdentityServerV0(config orquestaserver.ConfigV0, reason string, stderr io.Writer) int {
	listener, err := net.Listen("tcp", config.Addr)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "orquesta-server: reason_code=degraded_identity_listen_failed")
		return 1
	}
	defer listener.Close()
	server := &http.Server{Handler: newDegradedIdentityHTTPHandlerV0(config, reason)}
	ctx, stop := signal.NotifyContext(context.Background(), serverShutdownSignalsV0()...)
	defer stop()
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
		}
	case err := <-serveDone:
		if err != nil && err != http.ErrServerClosed {
			_, _ = fmt.Fprintln(stderr, "orquesta-server: reason_code=degraded_identity_server_failed")
			return 1
		}
		return 0
	}
	if err := <-serveDone; err != nil && err != http.ErrServerClosed {
		_, _ = fmt.Fprintln(stderr, "orquesta-server: reason_code=degraded_identity_server_failed")
		return 1
	}
	return 0
}

func degradedIdentityServerAddrFromEnvV0() string {
	if addr := strings.TrimSpace(os.Getenv(envServerAddrV0)); addr != "" {
		return addr
	}
	return orquestaserver.DefaultAddrV0
}

func statusServerCommandV0(args []string, stdout io.Writer, stderr io.Writer) int {
	options, err := parseServerCommandConfigOptionsV0("status", args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server status: %v\n", err)
		return 2
	}
	config, err := serverConfigFromEnvWithProjectConfigPathV0(options.ConfigPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server status: %v\n", err)
		return 1
	}
	defer releaseServerProjectConfigSnapshotV0(config)
	state, err := loadStateV0(config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server status: %v\n", err)
		return 1
	}
	if body, err := getStatusBodyV0(state.Addr); err == nil {
		var status orquestaserver.ServerPublicStatusV0
		if err := json.Unmarshal(body, &status); err != nil {
			_, _ = fmt.Fprintf(stderr, "orquesta-server status: %v\n", err)
			return 1
		}
		if err := writeCommandPublicOutputV0(
			stdout,
			"status",
			status.Status,
			commandPublicFreshnessLiveV0,
			"use_server_status_endpoint_for_canonical_public_status",
			commandPublicServerStatusPayloadV0(status),
		); err != nil {
			return reportCommandStdioWriteFailureV0(stderr, "status", "stdout", "json_encode", err)
		}
		return 0
	}
	state, reconciled := reconcileStatefileSnapshotLivenessV0(config, state)
	if !reconciled {
		state, reconciled = reconcileStoppedStatefileSnapshotV0(config, state)
	}
	status := "degraded"
	diagnosticsMode := "statefile_snapshot_public_projection"
	stateStatus := strings.TrimSpace(state.Status)
	if reconciled || stateStatus == "stale" || stateStatus == "stopped" {
		status = strings.TrimSpace(state.Status)
		if status == "" {
			status = "degraded"
		}
		if reconciled {
			diagnosticsMode = "statefile_snapshot_reconciled"
		}
		if stateStatus == "stale" {
			diagnosticsMode = "statefile_snapshot_reconciled_process_not_alive"
		}
	}
	if err := writeCommandPublicOutputV0(
		stdout,
		"status",
		status,
		commandPublicFreshnessSnapshotV0,
		diagnosticsMode,
		commandPublicServerStatusPayloadV0(orquestaserver.NewServerPublicStatusV0(state)),
	); err != nil {
		return reportCommandStdioWriteFailureV0(stderr, "status", "stdout", "json_encode", err)
	}
	return 1
}

func reconcileStatefileSnapshotLivenessV0(
	config orquestaserver.ConfigV0,
	state orquestaserver.StateV0,
) (orquestaserver.StateV0, bool) {
	if !serverStateStatusNeedsProcessReconciliationV0(state.Status) {
		return state, false
	}
	if state.PID > 0 && processAliveV0(state.PID) {
		return state, false
	}
	reconciled := orquestaserver.MarkServerProcessStaleStateV0(state, time.Now().UTC())
	store, err := orquestaserver.NewFileStateStoreV0(orquestaserver.StatePathV0(config))
	if err == nil {
		_ = store.SaveServerStateV0(context.Background(), reconciled)
	}
	return reconciled, true
}

func reconcileStoppedStatefileSnapshotV0(
	config orquestaserver.ConfigV0,
	state orquestaserver.StateV0,
) (orquestaserver.StateV0, bool) {
	reconciled, ok := orquestaserver.NormalizeStoppedServerSnapshotV0(state)
	if !ok {
		return state, false
	}
	store, err := orquestaserver.NewFileStateStoreV0(orquestaserver.StatePathV0(config))
	if err == nil {
		_ = store.SaveServerStateV0(context.Background(), reconciled)
	}
	return reconciled, true
}

func serverStateStatusNeedsProcessReconciliationV0(status string) bool {
	switch strings.TrimSpace(status) {
	case "running", "starting", "startup_blocked":
		return true
	default:
		return false
	}
}

func stopServerCommandV0(args []string, stdout io.Writer, stderr io.Writer) int {
	options, err := parseStopOptionsV0(args, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server stop: %v\n", err)
		return 2
	}
	config, err := serverConfigFromEnvWithProjectConfigPathV0(options.ConfigPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server stop: %v\n", err)
		return 1
	}
	defer releaseServerProjectConfigSnapshotV0(config)
	state, err := loadStateV0(config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server stop: %v\n", err)
		return 1
	}
	live, err := verifyLiveDaemonIdentityV0(state)
	if err != nil {
		if handled, exitCode := stopServerCommandHandleUnavailableDaemonV0(config, state, options, stdout, stderr, err); handled {
			return exitCode
		}
		_, _ = fmt.Fprintf(stderr, "orquesta-server stop: %v\n", err)
		return 1
	}
	if err := requestServerShutdownV0(state.Addr, options); err != nil {
		liveAfter := live
		if shutdownRequestErrorMayStillNeedSignalV0(err) {
			if refreshed, refreshErr := verifyLiveDaemonIdentityV0(state); refreshErr == nil {
				liveAfter = refreshed
			}
		}
		if !shutdownRequestErrorAllowsSignalV0(err, liveAfter, options.Forced) {
			_, _ = fmt.Fprintf(stderr, "orquesta-server stop: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(stderr, "orquesta-server stop: %v; signal cooperativa permitida por estado publico\n", err)
	}
	if err := signalProcessV0(state.PID); err != nil {
		_, _ = fmt.Fprintf(stderr, "orquesta-server stop: %v\n", err)
		return 1
	}
	waitUntilDownV0(state.Addr, 5*time.Second)
	processWait := stopProcessWaitTimeoutV0(config)
	if !waitUntilProcessDownV0(state.PID, processWait) {
		if !options.Forced {
			_, _ = fmt.Fprintf(stderr, "orquesta-server stop: process_still_running_after_grace pid=%d timeout=%s\n", state.PID, processWait)
			return 1
		}
		if err := signalProcessV0(state.PID); err != nil {
			_, _ = fmt.Fprintf(stderr, "orquesta-server stop: force_escalation_failed pid=%d: %v\n", state.PID, err)
			return 1
		}
		if !waitUntilProcessDownV0(state.PID, 5*time.Second) {
			_, _ = fmt.Fprintf(stderr, "orquesta-server stop: process_still_running_after_force pid=%d\n", state.PID)
			return 1
		}
	}
	if err := writeCommandTextOutputV0(stdout, fmt.Sprintf("stop solicitado process_ref=%s\n", live.ProcessRef)); err != nil {
		return reportCommandStdioWriteFailureV0(stderr, "stop", "stdout", "text_write", err)
	}
	return 0
}

func stopServerCommandHandleUnavailableDaemonV0(
	config orquestaserver.ConfigV0,
	state orquestaserver.StateV0,
	options serverShutdownClientOptionsV0,
	stdout io.Writer,
	stderr io.Writer,
	identityErr error,
) (bool, int) {
	if !options.Forced ||
		!stopServerSnapshotProcessNotAliveV0(state) ||
		!stopServerSnapshotHTTPUnavailableV0(state, identityErr) {
		return false, 0
	}
	reconciled, _ := reconcileStatefileSnapshotLivenessV0(config, state)
	if cleanupErr := cleanupCodexGoalBackendAfterStartupFailureIfDaemonGoneV0(
		config,
		stoppedServerDaemonIdentityV0(state.PID),
	); cleanupErr != nil {
		_, _ = fmt.Fprintln(stderr, "orquesta-server stop: reason_code=forced_cleanup_failed")
		return true, 1
	}
	processRef := strings.TrimSpace(reconciled.ProcessRef)
	if processRef == "" {
		processRef = "process-ref-unavailable"
	}
	if err := writeCommandTextOutputV0(stdout, fmt.Sprintf("stop forced stale process_ref=%s\n", processRef)); err != nil {
		return true, reportCommandStdioWriteFailureV0(stderr, "stop", "stdout", "text_write", err)
	}
	return true, 0
}

func stopServerSnapshotProcessNotAliveV0(state orquestaserver.StateV0) bool {
	return state.PID <= 0 || !processAliveV0(state.PID)
}

func stopServerSnapshotHTTPUnavailableV0(state orquestaserver.StateV0, identityErr error) bool {
	if identityErr == nil || strings.TrimSpace(identityErr.Error()) != "daemon_identity_unavailable" {
		return false
	}
	return !stopServerStatusEndpointReachableV0(state)
}

func stopServerStatusEndpointReachableV0(state orquestaserver.StateV0) bool {
	addr := strings.TrimSpace(state.Addr)
	if addr == "" {
		return false
	}
	baseURL, err := commandRESTBaseURLFromAddrV0(addr)
	if err != nil {
		return false
	}
	target, err := commandRESTEndpointURLV0(baseURL, orquestaserver.ServerStatusEndpointV0)
	if err != nil {
		return false
	}
	client := commandHTTPClientWithRedirectPolicyV0(2*time.Second, baseURL)
	response, err := client.Get(target)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return true
}

func parseStopOptionsV0(args []string, stderr io.Writer) (serverShutdownClientOptionsV0, error) {
	flags := flag.NewFlagSet("stop", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	forced := flags.Bool("force", false, "forzar shutdown tras opt-in explicito")
	reason := flags.String("reason", "", "causa publica compacta")
	configPath := flags.String("config", "", "ruta a orquesta.config.json canonico")
	if err := flags.Parse(args); err != nil {
		return serverShutdownClientOptionsV0{}, fmt.Errorf("stop_args_invalid")
	}
	if flags.NArg() != 0 {
		return serverShutdownClientOptionsV0{}, fmt.Errorf("stop_args_invalid")
	}
	trimmedReason := strings.TrimSpace(*reason)
	if *forced && trimmedReason == "" {
		return serverShutdownClientOptionsV0{}, fmt.Errorf("forced_reason_required")
	}
	if trimmedReason == "" {
		trimmedReason = "apagado cooperativo solicitado por CLI al Director"
	}
	return serverShutdownClientOptionsV0{
		Forced:     *forced,
		Reason:     trimmedReason,
		ConfigPath: strings.TrimSpace(*configPath),
	}, nil
}

func loadStateV0(config orquestaserver.ConfigV0) (orquestaserver.StateV0, error) {
	store, err := orquestaserver.NewFileStateStoreV0(orquestaserver.StatePathV0(config))
	if err != nil {
		return orquestaserver.StateV0{}, err
	}
	return store.LoadServerStateV0(context.Background())
}

func getStatusBodyV0(addr string) ([]byte, error) {
	baseURL, err := commandRESTBaseURLFromAddrV0(addr)
	if err != nil {
		return nil, err
	}
	target, err := commandRESTEndpointURLV0(baseURL, orquestaserver.ServerStatusEndpointV0)
	if err != nil {
		return nil, err
	}
	client := commandHTTPClientWithRedirectPolicyV0(2*time.Second, baseURL)
	response, err := client.Get(target)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return readCommandHTTPResponseBodyV0(response, "status")
}

func waitUntilDownV0(addr string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := getStatusBodyV0(addr); err != nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func stopProcessWaitTimeoutV0(config orquestaserver.ConfigV0) time.Duration {
	timeout := config.ShutdownGracePeriod
	if timeout <= 0 {
		timeout = orquestaserver.DefaultShutdownGracePeriodV0
	}
	return timeout + 2*time.Second
}

func waitUntilProcessDownV0(pid int, timeout time.Duration) bool {
	if pid <= 0 {
		return true
	}
	if timeout <= 0 {
		timeout = time.Second
	}
	deadline := time.Now().Add(timeout)
	for {
		if !processAliveV0(pid) {
			return true
		}
		if !time.Now().Before(deadline) {
			return !processAliveV0(pid)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
