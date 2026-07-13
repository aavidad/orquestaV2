package orquestaruntimecodexappserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	codexAppServerTmuxDirV0             = "goal-srv"
	codexAppServerTmuxMarkerFileV0      = "owner.json"
	codexAppServerTmuxOwnerSchemaV0     = "orquesta_codex_app_server_tmux_owner.v0"
	codexAppServerTmuxEvidenceOwnedV0   = "orquesta-codex-goal-app-server-tmux-v0"
	codexAppServerTmuxDefaultTimeoutV0  = 3 * time.Second
	codexAppServerTmuxCleanupTimeoutV0  = 10 * time.Second
	codexAppServerTmuxForcedStopV0      = 2 * time.Second
	codexAppServerTmuxCooperativeStopV0 = 750 * time.Millisecond
	codexAppServerTmuxMinStartupV0      = 60 * time.Second
	codexAppServerTmuxMaxSocketPathV0   = 107
	codexAppServerTmuxSocketPollEveryV0 = 50 * time.Millisecond
)

type serverCodexAppServerTmuxBackendV0 struct {
	CommandPath              string
	PathEnv                  string
	SocketPath               string
	SessionName              string
	HomeDir                  string
	CodeHomeDir              string
	RuntimeWorkDir           string
	ProjectWorkDir           string
	SourceCodeHomeDir        string
	Timeout                  time.Duration
	ShutdownCleanupTimeout   time.Duration
	beforeSocketQuarantineV0 func()
	beforeSessionKillV0      func()
	beforePathQuarantineV0   func(string)
	signalProcessIdentityV0  func(int, string, syscall.Signal) error
}

type codexAppServerTmuxOwnerMarkerV0 struct {
	SchemaVersion           string `json:"schema_version"`
	OwnerRef                string `json:"owner_ref"`
	SessionName             string `json:"session_name"`
	SocketRef               string `json:"socket_ref"`
	SocketPath              string `json:"socket_path,omitempty"`
	GenerationRef           string `json:"generation_ref,omitempty"`
	LeaseOwnerPID           int    `json:"lease_owner_pid,omitempty"`
	LeaseOwnerStartRef      string `json:"lease_owner_start_ref,omitempty"`
	AppServerPID            int    `json:"app_server_pid,omitempty"`
	AppServerStartRef       string `json:"app_server_start_ref,omitempty"`
	AppServerProcessGroupID int    `json:"app_server_process_group_id,omitempty"`
	SocketOwnerPID          int    `json:"socket_owner_pid,omitempty"`
	SocketOwnerStartRef     string `json:"socket_owner_start_ref,omitempty"`
	TmuxSessionID           string `json:"tmux_session_id,omitempty"`
	TmuxSessionCreated      string `json:"tmux_session_created,omitempty"`
	TmuxPanePID             int    `json:"tmux_pane_pid,omitempty"`
	TmuxPaneStartRef        string `json:"tmux_pane_start_ref,omitempty"`
}

func (backend serverCodexAppServerTmuxBackendV0) EnsureV0(
	ctx context.Context,
	preflight serverCodexAppServerProbePortV0,
) error {
	timeout := backend.Timeout
	if timeout <= 0 {
		timeout = codexAppServerTmuxDefaultTimeoutV0
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	socketPath := strings.TrimSpace(backend.SocketPath)
	sessionName := strings.TrimSpace(backend.SessionName)
	if socketPath == "" || sessionName == "" {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_config_missing", Err: errors.New("codex_app_server_tmux_config_missing")}
	}
	if err := ensureCodexAppServerTmuxRuntimeDirV0(socketPath); err != nil {
		return err
	}
	leaseGuard, err := backend.acquireTmuxLeaseGuardV0(runCtx)
	if err != nil {
		return err
	}
	defer leaseGuard.releaseV0()
	adopted, err := backend.adoptOrFenceExistingGenerationV0(runCtx, preflight, leaseGuard)
	if err != nil {
		return err
	}
	if adopted {
		return nil
	}
	tmuxPath, err := codexAppServerTmuxCommandPathV0(backend.PathEnv)
	if err != nil {
		return err
	}
	observed, err := backend.recolectarObservacionBackendV0(runCtx, solicitudObservacionBackendV0{
		Actual:                BackendPreparandoV0,
		ActionRequested:       AccionBackendEnsureV0,
		TmuxPath:              tmuxPath,
		Preflight:             preflight,
		CheckPreflight:        true,
		RequireTmux:           true,
		SessionUnownedIsIssue: true,
	})
	if err != nil {
		return err
	}
	if observed.Estado == BackendListoV0 {
		return nil
	}
	if observed.Observacion.SessionObserved {
		if observed.Observacion.IssueCode == "codex_app_server_tmux_session_unowned" {
			return codexAppServerCallErrorV0{
				Code: "codex_app_server_tmux_session_unowned",
				Err:  errors.New("codex_app_server_tmux_session_unowned"),
			}
		}
		if observed.Estado == BackendListoV0 {
			return nil
		}
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	if err := backend.removeStaleTmuxSocketWithLeaseV0(runCtx, leaseGuard, codexAppServerTmuxOwnerMarkerV0{}); err != nil {
		return err
	}
	if err := backend.prepareTmuxCodeHomeV0(); err != nil {
		return err
	}
	marker, err := backend.newTmuxOwnerMarkerV0("", 0)
	if err != nil {
		return err
	}
	if err := backend.writeTmuxOwnerMarkerAtomicWithLeaseV0(leaseGuard, marker); err != nil {
		return err
	}
	identity, err := backend.tmuxStartSessionV0(runCtx, tmuxPath, marker.GenerationRef)
	if err != nil {
		backend.cleanupStartedTmuxGenerationWithLeaseV0(leaseGuard, marker)
		return err
	}
	if !identity.completeV0() {
		backend.cleanupStartedTmuxGenerationWithLeaseV0(leaseGuard, marker)
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	updated := marker
	updated.AppServerPID = identity.PanePID
	updated.AppServerStartRef = identity.PaneStartRef
	updated.AppServerProcessGroupID = codexAppServerTmuxOwnedProcessGroupV0(identity.PanePID)
	updated.TmuxSessionID = identity.SessionID
	updated.TmuxSessionCreated = identity.SessionCreated
	updated.TmuxPanePID = identity.PanePID
	updated.TmuxPaneStartRef = identity.PaneStartRef
	if err := backend.replaceTmuxOwnerMarkerWithLeaseV0(leaseGuard, marker, updated); err != nil {
		backend.cleanupStartedTmuxGenerationWithLeaseV0(leaseGuard, marker)
		return err
	}
	marker = updated
	if err := backend.waitForTmuxSocketV0(runCtx, tmuxPath, preflight, marker, leaseGuard); err != nil {
		backend.cleanupStartedTmuxGenerationWithLeaseV0(leaseGuard, marker)
		return err
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxKillSessionV0(
	ctx context.Context,
	tmuxPath string,
	expected codexAppServerTmuxSessionIdentityV0,
) error {
	observed, err := backend.tmuxSessionIdentityTargetV0(ctx, tmuxPath, expected.SessionID)
	if err != nil || !reflect.DeepEqual(observed, expected) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	if backend.beforeSessionKillV0 != nil {
		backend.beforeSessionKillV0()
	}
	output, err := backend.runTmuxCommandV0(ctx, tmuxPath, "kill-session", "-t", expected.SessionID)
	if err != nil {
		return codexAppServerTmuxCommandErrorV0("codex_app_server_tmux_kill_failed", output, err)
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) ShutdownV0(ctx context.Context) error {
	var err error
	if marker, ok := backend.readTmuxOwnerMarkerV0(); ok && marker.generationMarkerV0() {
		err = backend.shutdownTmuxGenerationLeaseV0(ctx, marker, false)
	} else {
		err = backend.shutdownTmuxSessionV0(ctx, false)
	}
	// Missing/ambiguous re-observation is residual evidence, not permission to
	// kill or unlink. A live contradictory identity remains a hard conflict.
	if codexAppServerTmuxIsObservationTransientV0(err) {
		return nil
	}
	return err
}

func (backend serverCodexAppServerTmuxBackendV0) ShutdownForcedStopV0(ctx context.Context) error {
	if marker, ok := backend.readTmuxOwnerMarkerV0(); ok && marker.generationMarkerV0() {
		return backend.shutdownTmuxGenerationLeaseV0(ctx, marker, true)
	}
	cleanup := backend
	cleanup.Timeout = codexAppServerTmuxForcedStopV0
	return cleanup.shutdownTmuxSessionWithOptionsV0(ctx, true, true)
}

// shutdownForcedStopGenerationV0 refuses to stop a runtime that rotated after
// Goal selected its immutable generation. shutdownTmuxGenerationLeaseV0
// rechecks this exact marker while holding the owner lease before kill/unlink.
func (backend serverCodexAppServerTmuxBackendV0) shutdownForcedStopGenerationV0(ctx context.Context, expectedGenerationRef string) error {
	expectedGenerationRef = strings.TrimSpace(expectedGenerationRef)
	marker, ok := backend.readTmuxOwnerMarkerV0()
	if !ok || !marker.generationMarkerV0() || strings.TrimSpace(marker.GenerationRef) != expectedGenerationRef {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	return backend.shutdownTmuxGenerationLeaseV0(ctx, marker, true)
}

func (backend serverCodexAppServerTmuxBackendV0) ShutdownConfiguredSessionAfterStartupFailureV0(ctx context.Context) error {
	if marker, ok := backend.readTmuxOwnerMarkerV0(); ok && marker.generationMarkerV0() {
		return backend.shutdownTmuxGenerationLeaseV0(ctx, marker, true)
	}
	return backend.shutdownTmuxSessionV0(ctx, true)
}

func codexAppServerTmuxSocketPresentV0(socketPath string) bool {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return false
	}
	info, err := os.Lstat(socketPath)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func codexAppServerTmuxPIDAliveV0(panePID string) bool {
	pid, err := strconv.Atoi(strings.TrimSpace(panePID))
	if err != nil || pid <= 0 {
		return false
	}
	err = syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func (backend serverCodexAppServerTmuxBackendV0) shutdownTmuxSessionV0(ctx context.Context, allowConfiguredOrphan bool) error {
	return backend.shutdownTmuxSessionWithOptionsV0(ctx, allowConfiguredOrphan, false)
}

func (backend serverCodexAppServerTmuxBackendV0) shutdownTmuxSessionWithOptionsV0(
	ctx context.Context,
	_ bool,
	_ bool,
) error {
	timeout := backend.Timeout
	if timeout <= 0 {
		timeout = codexAppServerTmuxDefaultTimeoutV0
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	guard, err := backend.acquireTmuxLeaseGuardV0(runCtx)
	if err != nil {
		return err
	}
	defer guard.releaseV0()
	if marker, ok := backend.readTmuxOwnerMarkerV0(); ok && marker.generationMarkerV0() {
		// The caller classified a legacy marker before taking the lease, but a
		// newer generation won the race. Never fall through to broad cleanup.
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}

	if backend.tmuxOwnerMarkerObservedV0() || backend.tmuxLegacyOwnerMarkerObservedV0() {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	tmuxPath, err := codexAppServerTmuxCommandPathV0(backend.PathEnv)
	if err != nil {
		if codexAppServerTmuxSocketPresentV0(backend.SocketPath) {
			return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
		}
		return nil
	}
	hasSession, err := backend.tmuxHasSessionV0(runCtx, tmuxPath)
	if err != nil {
		return err
	}
	if hasSession || codexAppServerTmuxSocketPresentV0(backend.SocketPath) ||
		len(codexAppServerTmuxSocketProcessPIDsV0(runCtx, backend.SocketPath)) > 0 {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) shutdownCleanupTimeoutV0() time.Duration {
	if backend.ShutdownCleanupTimeout > 0 {
		return backend.ShutdownCleanupTimeout
	}
	return codexAppServerTmuxCleanupTimeoutV0
}

func (backend serverCodexAppServerTmuxBackendV0) shutdownTmuxSessionForCleanupV0(ctx context.Context) error {
	cleanup := backend
	cleanup.Timeout = backend.shutdownCleanupTimeoutV0()
	if marker, ok := cleanup.readTmuxOwnerMarkerV0(); ok && marker.generationMarkerV0() {
		return cleanup.shutdownTmuxGenerationLeaseV0(ctx, marker, true)
	}
	return cleanup.shutdownTmuxSessionWithOptionsV0(ctx, true, true)
}

func codexAppServerTmuxSocketProcessPIDsV0(ctx context.Context, socketPath string) []int {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil
	}
	output, err := exec.CommandContext(ctx, "ps", "-eo", "pid=,args=").Output()
	if err != nil {
		return nil
	}
	out := []int{}
	currentPID := os.Getpid()
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid <= 0 || pid == currentPID {
			continue
		}
		command := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), fields[0]))
		if codexAppServerTmuxCommandMatchesSocketV0(command, socketPath) {
			out = append(out, pid)
		}
	}
	return out
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxConfiguredOrphanCleanupAllowedV0() bool {
	sessionName := strings.TrimSpace(backend.SessionName)
	if !strings.HasPrefix(sessionName, "orquesta-goal-") ||
		len(strings.TrimPrefix(sessionName, "orquesta-goal-")) < 16 {
		return false
	}
	socketPath := strings.TrimSpace(backend.SocketPath)
	if socketPath == "" || filepath.Base(socketPath) == "." || filepath.Dir(socketPath) == "." {
		return false
	}
	if filepath.Ext(socketPath) != ".sock" && filepath.Base(socketPath) != "s.sock" {
		return false
	}
	return true
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxStartSessionV0(
	ctx context.Context,
	tmuxPath string,
	generationRef string,
) (codexAppServerTmuxSessionIdentityV0, error) {
	commandPath := strings.TrimSpace(backend.CommandPath)
	if commandPath == "" {
		commandPath = codexCommandPathV0()
	}
	socketURL := "unix://" + strings.TrimSpace(backend.SocketPath)
	pathEnv := strings.TrimSpace(backend.PathEnv)
	if pathEnv == "" {
		pathEnv = os.Getenv("PATH")
	}
	envAssignments := []string{
		"PATH=" + shellQuoteCodexAppServerTmuxV0(pathEnv),
	}
	if homeDir := strings.TrimSpace(backend.HomeDir); homeDir != "" {
		envAssignments = append(envAssignments, "HOME="+shellQuoteCodexAppServerTmuxV0(homeDir))
	}
	if codeHomeDir := strings.TrimSpace(backend.CodeHomeDir); codeHomeDir != "" {
		envAssignments = append(envAssignments, "CODEX_HOME="+shellQuoteCodexAppServerTmuxV0(codeHomeDir))
	}
	for _, key := range []string{
		"CODEX_MANAGED_BY_NPM",
		"CODEX_MANAGED_PACKAGE_ROOT",
		"CODEX_CI",
		"XDG_RUNTIME_DIR",
	} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			envAssignments = append(envAssignments, key+"="+shellQuoteCodexAppServerTmuxV0(value))
		}
	}
	logPath := backend.tmuxLogPathV0()
	stdinPath := backend.tmuxStdinPathV0()
	_ = os.Remove(logPath)
	_ = os.Remove(stdinPath)
	// app-server necesita stdin abierto; la FIFO evita heredar el PTY de tmux.
	stdinSetup := "rm -f " + shellQuoteCodexAppServerTmuxV0(stdinPath) +
		" && mkfifo " + shellQuoteCodexAppServerTmuxV0(stdinPath) +
		" || exit 1; (while :; do sleep 3600; done > " +
		shellQuoteCodexAppServerTmuxV0(stdinPath) + ") & "
	shellCommand := stdinSetup + strings.Join(envAssignments, " ") +
		" exec " + shellQuoteCodexAppServerTmuxV0(commandPath) +
		" app-server --listen " + shellQuoteCodexAppServerTmuxV0(socketURL) +
		" < " + shellQuoteCodexAppServerTmuxV0(stdinPath) +
		" >> " + shellQuoteCodexAppServerTmuxV0(logPath) + " 2>&1"
	output, err := backend.runTmuxCommandV0(
		ctx,
		tmuxPath,
		"new-session",
		"-d",
		"-P",
		"-F",
		"#{session_id}|#{session_created}|#{pane_pid}",
		"-e",
		codexAppServerTmuxGenerationEnvironmentV0+"="+strings.TrimSpace(generationRef),
		"-s",
		strings.TrimSpace(backend.SessionName),
		shellCommand,
	)
	if err != nil {
		return codexAppServerTmuxSessionIdentityV0{}, codexAppServerTmuxCommandErrorV0("codex_app_server_tmux_start_failed", output, err)
	}
	identity, err := codexAppServerTmuxSessionIdentityFromOutputV0(output)
	if err != nil {
		return codexAppServerTmuxSessionIdentityV0{}, err
	}
	if err := backend.verifyTmuxGenerationTokenV0(ctx, tmuxPath, identity.SessionID, generationRef); err != nil {
		return codexAppServerTmuxSessionIdentityV0{}, err
	}
	return identity, nil
}

func (backend serverCodexAppServerTmuxBackendV0) runTmuxCommandV0(
	ctx context.Context,
	tmuxPath string,
	args ...string,
) (string, error) {
	cmd := exec.CommandContext(ctx, tmuxPath, args...)
	if strings.TrimSpace(backend.PathEnv) != "" {
		cmd.Env = append(os.Environ(), "PATH="+strings.TrimSpace(backend.PathEnv))
	}
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if ctx.Err() != nil {
		return output.String(), codexAppServerCallErrorV0{Code: "codex_app_server_tmux_timeout", Err: ctx.Err()}
	}
	return output.String(), err
}

func (backend serverCodexAppServerTmuxBackendV0) waitForTmuxSocketV0(
	ctx context.Context,
	tmuxPath string,
	preflight serverCodexAppServerProbePortV0,
	marker codexAppServerTmuxOwnerMarkerV0,
	lease *codexAppServerTmuxLeaseGuardV0,
) error {
	socketPath := strings.TrimSpace(backend.SocketPath)
	nextSessionCheck := time.Now()
	var lastPreflightErr error
	lastObservationIssue := ""
	for {
		if ctx.Err() != nil {
			if lastPreflightErr != nil {
				return lastPreflightErr
			}
			if lastObservationIssue != "" {
				return codexAppServerCallErrorV0{Code: lastObservationIssue, Err: ctx.Err()}
			}
			return backend.tmuxStartupFailureV0("codex_app_server_tmux_socket_timeout", ctx.Err())
		}
		owner, ownerErr := codexAppServerTmuxSocketOwnerDescendantV0(socketPath, marker.TmuxPanePID)
		if errors.Is(ownerErr, errCodexAppServerTmuxSocketOwnerOutsidePaneV0) {
			return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
		}
		if marker.SocketOwnerPID == 0 && ownerErr != nil {
			switch {
			case errors.Is(ownerErr, errCodexAppServerTmuxSocketInodeNotObservedV0):
				lastObservationIssue = "codex_app_server_tmux_socket_inode_not_observed"
			case errors.Is(ownerErr, errCodexAppServerTmuxSocketOwnerAmbiguousV0):
				lastObservationIssue = "codex_app_server_tmux_socket_owner_ambiguous"
			default:
				lastObservationIssue = "codex_app_server_tmux_socket_owner_unavailable"
			}
		}
		if marker.SocketOwnerPID > 0 {
			current, currentOK := backend.readTmuxOwnerMarkerV0()
			if !currentOK {
				lastObservationIssue = "codex_app_server_tmux_marker_observation_transient"
			}
			if currentOK && !reflect.DeepEqual(current, marker) {
				return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
			}
			tmuxObservation := backend.observeRecordedTmuxGenerationV0(ctx, tmuxPath, marker)
			appObservation := marker.observeAppServerV0(socketPath)
			if tmuxObservation == codexAppServerTmuxGenerationTransientV0 {
				lastObservationIssue = codexAppServerTmuxObservationTransientV0
			} else if appObservation == codexAppServerTmuxGenerationTransientV0 {
				lastObservationIssue = "codex_app_server_tmux_socket_owner_observation_transient"
			} else if currentOK {
				lastObservationIssue = ""
			}
			if tmuxObservation == codexAppServerTmuxGenerationContradictedV0 || appObservation == codexAppServerTmuxGenerationContradictedV0 {
				return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
			}
			if currentOK && tmuxObservation == codexAppServerTmuxGenerationVerifiedV0 &&
				appObservation == codexAppServerTmuxGenerationVerifiedV0 &&
				backend.ensureTmuxSocketPrivateV0() == nil && preflight != nil {
				if probeErr := preflight.ProbeV0(ctx); probeErr == nil {
					return nil
				} else {
					lastPreflightErr = probeErr
				}
			}
		} else if ownerErr == nil && backend.ensureTmuxSocketPrivateV0() == nil && preflight != nil {
			probeErr := preflight.ProbeV0(ctx)
			if probeErr != nil {
				lastPreflightErr = probeErr
				goto waitForNextObservation
			}
			tmuxObservation := backend.observeRecordedTmuxGenerationV0(ctx, tmuxPath, marker)
			if tmuxObservation == codexAppServerTmuxGenerationContradictedV0 {
				return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
			}
			if tmuxObservation != codexAppServerTmuxGenerationVerifiedV0 {
				goto waitForNextObservation
			}
			updated := marker
			updated.SocketOwnerPID = owner.PID
			updated.SocketOwnerStartRef = owner.StartRef
			updated.AppServerPID = owner.PID
			updated.AppServerStartRef = owner.StartRef
			updated.AppServerProcessGroupID = codexAppServerTmuxOwnedProcessGroupV0(owner.PID)
			if err := backend.replaceTmuxOwnerMarkerWithLeaseV0(lease, marker, updated); err != nil {
				return err
			}
			marker = updated
			continue
		}
		if !nextSessionCheck.After(time.Now()) {
			observed, err := backend.recolectarObservacionBackendV0(ctx, solicitudObservacionBackendV0{
				Actual:          BackendSocketPendienteV0,
				ActionRequested: AccionBackendEnsureV0,
				TmuxPath:        tmuxPath,
				RequireTmux:     true,
			})
			if ctx.Err() != nil {
				return backend.tmuxStartupFailureV0("codex_app_server_tmux_socket_timeout", ctx.Err())
			}
			if err != nil {
				return err
			}
			if !observed.Observacion.SessionObserved {
				nextSessionCheck = time.Now().Add(250 * time.Millisecond)
				goto waitForNextObservation
			}
			nextSessionCheck = time.Now().Add(250 * time.Millisecond)
		}
	waitForNextObservation:
		select {
		case <-ctx.Done():
			if lastPreflightErr != nil {
				return lastPreflightErr
			}
			if lastObservationIssue != "" {
				return codexAppServerCallErrorV0{Code: lastObservationIssue, Err: ctx.Err()}
			}
			return backend.tmuxStartupFailureV0("codex_app_server_tmux_socket_timeout", ctx.Err())
		case <-time.After(codexAppServerTmuxSocketPollEveryV0):
		}
	}
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxStartupFailureV0(
	fallback string,
	err error,
) error {
	if code := codexAppServerIssueCodeFromLogFileV0(backend.tmuxLogPathV0()); code != "" {
		return codexAppServerCallErrorV0{Code: code, Err: err}
	}
	code := strings.TrimSpace(fallback)
	if code == "" {
		code = "codex_app_server_tmux_failed"
	}
	return codexAppServerCallErrorV0{Code: code, Err: err}
}

func ensureCodexAppServerTmuxRuntimeDirV0(socketPath string) error {
	dir := filepath.Dir(strings.TrimSpace(socketPath))
	if dir == "." || dir == "" {
		return codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_runtime_dir_unavailable",
			Err:  errors.New("codex_app_server_tmux_runtime_dir_unavailable"),
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_runtime_dir_unavailable", Err: err}
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_runtime_dir_unavailable", Err: err}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_runtime_dir_unavailable", Err: errors.New("codex_app_server_tmux_runtime_dir_symlink")}
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_runtime_dir_unavailable", Err: err}
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) ensureTmuxSocketPrivateV0() error {
	socketPath := strings.TrimSpace(backend.SocketPath)
	if socketPath == "" {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_config_missing", Err: errors.New("codex_app_server_tmux_config_missing")}
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_socket_permissions_unavailable", Err: err}
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxOwnerMarkerPathV0() string {
	socketPath := strings.TrimSpace(backend.SocketPath)
	if socketPath == "" {
		return ""
	}
	return socketPath + ".owner.json"
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxLegacyOwnerMarkerPathV0() string {
	socketPath := strings.TrimSpace(backend.SocketPath)
	if socketPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(socketPath), codexAppServerTmuxMarkerFileV0)
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxStdinPathV0() string {
	socketPath := strings.TrimSpace(backend.SocketPath)
	if socketPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(socketPath), "stdin.pipe")
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxLogPathV0() string {
	socketPath := strings.TrimSpace(backend.SocketPath)
	if socketPath == "" {
		return ""
	}
	sessionName := strings.TrimSpace(backend.SessionName)
	if sessionName == "" {
		sessionName = "codex-app-server"
	}
	return filepath.Join(filepath.Dir(socketPath), sessionName+".log")
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxOwnerMarkerExistsV0() bool {
	_, ok := backend.readTmuxOwnerMarkerV0()
	return ok
}

func (backend serverCodexAppServerTmuxBackendV0) readTmuxOwnerMarkerV0() (codexAppServerTmuxOwnerMarkerV0, bool) {
	path := backend.tmuxOwnerMarkerPathV0()
	if path == "" {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	marker, ok := readCodexAppServerTmuxOwnerMarkerPathV0(path)
	if !ok {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	if !codexAppServerTmuxOwnerSchemaSupportedV0(marker.SchemaVersion) ||
		strings.TrimSpace(marker.OwnerRef) != codexAppServerTmuxEvidenceOwnedV0 ||
		strings.TrimSpace(marker.SessionName) != strings.TrimSpace(backend.SessionName) {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	if marker.generationMarkerV0() && filepath.Clean(strings.TrimSpace(marker.SocketPath)) != filepath.Clean(strings.TrimSpace(backend.SocketPath)) {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	return marker, true
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxLegacyOwnerMarkerObservedV0() bool {
	path := backend.tmuxLegacyOwnerMarkerPathV0()
	if path == "" {
		return false
	}
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func readCodexAppServerTmuxOwnerMarkerPathV0(path string) (codexAppServerTmuxOwnerMarkerV0, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	info, err := os.Lstat(path)
	if err != nil || info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	var marker codexAppServerTmuxOwnerMarkerV0
	if err := json.Unmarshal(raw, &marker); err != nil {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	if !codexAppServerTmuxOwnerSchemaSupportedV0(marker.SchemaVersion) ||
		strings.TrimSpace(marker.OwnerRef) != codexAppServerTmuxEvidenceOwnedV0 {
		return codexAppServerTmuxOwnerMarkerV0{}, false
	}
	return marker, true
}

func codexAppServerTmuxOwnerSchemaSupportedV0(schema string) bool {
	switch strings.TrimSpace(schema) {
	case codexAppServerTmuxOwnerSchemaV0, codexAppServerTmuxOwnerSchemaGenerationV0:
		return true
	default:
		return false
	}
}

func shellQuoteCodexAppServerTmuxV0(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func codexAppServerTmuxCommandPathV0(pathEnv string) (string, error) {
	if strings.TrimSpace(pathEnv) != "" {
		for _, dir := range filepath.SplitList(pathEnv) {
			candidate := filepath.Join(dir, "tmux")
			info, err := os.Stat(candidate)
			if err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
				return candidate, nil
			}
		}
	}
	path, err := exec.LookPath("tmux")
	if err != nil {
		return "", codexAppServerCallErrorV0{Code: "codex_app_server_tmux_command_missing", Err: err}
	}
	return path, nil
}

func codexAppServerTmuxCommandErrorV0(fallback string, output string, err error) error {
	if callErr, ok := err.(codexAppServerCallErrorV0); ok {
		return callErr
	}
	code := codexAppServerIssueCodeFromCommandFailureV0(output, err)
	if code == "codex_app_server_call_failed" {
		code = strings.TrimSpace(fallback)
	}
	if code == "" {
		code = "codex_app_server_tmux_failed"
	}
	return codexAppServerCallErrorV0{Code: code, Err: err}
}

func codexAppServerTmuxSocketPathV0(config ConfigV0) (string, error) {
	runtimeDir := strings.TrimSpace(config.RuntimeWorkDir)
	if runtimeDir == "" {
		return "", codexAppServerCallErrorV0{Code: "codex_app_server_tmux_runtime_workdir_required", Err: errors.New("codex_app_server_tmux_runtime_workdir_required")}
	}
	socketPath := filepath.Join(runtimeDir, codexAppServerTmuxDirV0, codexAppServerTmuxSocketFileNameV0(config))
	if len(socketPath) <= codexAppServerTmuxMaxSocketPathV0 {
		return socketPath, nil
	}
	shortPath := codexAppServerTmuxShortSocketPathV0(config)
	if len(shortPath) > codexAppServerTmuxMaxSocketPathV0 {
		return "", codexAppServerCallErrorV0{
			Code: "codex_app_server_tmux_socket_path_too_long",
			Err:  fmt.Errorf("codex_app_server_tmux_socket_path_too_long:%d", len(shortPath)),
		}
	}
	return shortPath, nil
}

func codexAppServerTmuxShortSocketPathV0(config ConfigV0) string {
	tempDir := strings.TrimSpace(os.TempDir())
	if tempDir == "" {
		tempDir = "/tmp"
	}
	hashPart := codexAppServerTmuxSocketHashPartV0(config)
	shortPath := filepath.Join(tempDir, codexAppServerTmuxShortSocketDirNameV0(hashPart), "s.sock")
	if len(shortPath) <= codexAppServerTmuxMaxSocketPathV0 {
		return shortPath
	}
	return filepath.Join("/tmp", codexAppServerTmuxShortSocketDirNameV0(hashPart), "s.sock")
}

func codexAppServerTmuxShortSocketDirNameV0(hashPart string) string {
	return fmt.Sprintf("oq-gsrv-%d-%s", os.Getuid(), hashPart)
}

func codexAppServerTmuxCodeHomePathV0(config ConfigV0) (string, error) {
	runtimeDir := strings.TrimSpace(config.RuntimeWorkDir)
	if runtimeDir == "" {
		return "", codexAppServerCallErrorV0{Code: "codex_app_server_tmux_runtime_workdir_required", Err: errors.New("codex_app_server_tmux_runtime_workdir_required")}
	}
	return filepath.Join(runtimeDir, codexAppServerTmuxDirV0, "codex-home"), nil
}

func codexAppServerTmuxSessionNameV0(config ConfigV0) string {
	runtimeDir := filepath.Clean(strings.TrimSpace(config.RuntimeWorkDir))
	projectDir := filepath.Clean(strings.TrimSpace(config.ProjectWorkDir))
	sum := sha256.Sum256([]byte(strings.Join([]string{runtimeDir, projectDir}, "\x00")))
	return fmt.Sprintf("orquesta-goal-%x", sum[:8])
}

func codexAppServerTmuxSocketFileNameV0(config ConfigV0) string {
	return "g-" + codexAppServerTmuxSocketHashPartV0(config) + ".sock"
}

func codexAppServerTmuxSocketHashPartV0(config ConfigV0) string {
	sessionName := codexAppServerTmuxSessionNameV0(config)
	hashPart := strings.TrimPrefix(sessionName, "orquesta-goal-")
	if hashPart == "" || hashPart == sessionName {
		sum := sha256.Sum256([]byte(sessionName))
		hashPart = fmt.Sprintf("%x", sum[:8])
	}
	return hashPart
}

func codexAppServerTmuxStartupTimeoutV0(preflightTimeout time.Duration) time.Duration {
	if preflightTimeout < codexAppServerTmuxMinStartupV0 {
		return codexAppServerTmuxMinStartupV0
	}
	return preflightTimeout
}
