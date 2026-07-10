package orquestaruntimecodexappserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	codexAppServerTmuxOwnerSchemaGenerationV0 = "orquesta_codex_app_server_tmux_owner_generation.v0"
	codexAppServerTmuxGenerationConflictV0    = "codex_app_server_tmux_generation_conflict"
	codexAppServerTmuxLeaseConflictV0         = "codex_app_server_tmux_owner_lease_conflict"
)

type codexAppServerTmuxLeaseGuardV0 struct{ file *os.File }

var codexAppServerTmuxMarkerCASLocksV0 sync.Map

func codexAppServerTmuxMarkerCASLockV0(path string) *sync.Mutex {
	lock, _ := codexAppServerTmuxMarkerCASLocksV0.LoadOrStore(filepath.Clean(path), &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func (guard *codexAppServerTmuxLeaseGuardV0) releaseV0() {
	if guard == nil || guard.file == nil {
		return
	}
	_ = syscall.Flock(int(guard.file.Fd()), syscall.LOCK_UN)
	_ = guard.file.Close()
}

func (backend serverCodexAppServerTmuxBackendV0) acquireTmuxLeaseGuardV0(ctx context.Context) (*codexAppServerTmuxLeaseGuardV0, error) {
	path := backend.tmuxLeasePathV0()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, codexAppServerCallErrorV0{Code: "codex_app_server_tmux_owner_lease_unavailable", Err: err}
	}
	if info, statErr := file.Stat(); statErr != nil || !info.Mode().IsRegular() {
		_ = file.Close()
		if statErr == nil {
			statErr = errors.New("codex_app_server_tmux_owner_lease_not_regular")
		}
		return nil, codexAppServerCallErrorV0{Code: "codex_app_server_tmux_owner_lease_unavailable", Err: statErr}
	}
	for {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return &codexAppServerTmuxLeaseGuardV0{file: file}, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			_ = file.Close()
			return nil, codexAppServerCallErrorV0{Code: "codex_app_server_tmux_owner_lease_unavailable", Err: err}
		}
		select {
		case <-ctx.Done():
			_ = file.Close()
			return nil, codexAppServerCallErrorV0{Code: codexAppServerTmuxLeaseConflictV0, Err: ctx.Err()}
		case <-time.After(codexAppServerTmuxSocketPollEveryV0):
		}
	}
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxLeasePathV0() string {
	socketPath := strings.TrimSpace(backend.SocketPath)
	if socketPath == "" {
		return ""
	}
	return socketPath + ".owner.lease"
}

func newCodexAppServerTmuxGenerationRefV0() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "generation-ref-" + hex.EncodeToString(raw), nil
}

func codexAppServerTmuxProcessStartRefV0(pid int) string {
	startRef, _ := codexAppServerTmuxProcessStatV0(pid)
	return startRef
}

func codexAppServerTmuxProcessStatV0(pid int) (string, string) {
	if pid <= 0 {
		return "", ""
	}
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return "", ""
	}
	text := string(raw)
	end := strings.LastIndex(text, ")")
	if end < 0 {
		return "", ""
	}
	fields := strings.Fields(text[end+1:])
	// /proc/<pid>/stat field 22 is starttime; fields starts at field 3.
	if len(fields) <= 19 {
		return "", ""
	}
	return fields[19], fields[0]
}

func codexAppServerTmuxProcessIdentityAliveV0(pid int, startRef string) bool {
	startRef = strings.TrimSpace(startRef)
	if pid <= 0 || startRef == "" || !codexAppServerTmuxPIDAliveV0(strconv.Itoa(pid)) {
		return false
	}
	observedStartRef, state := codexAppServerTmuxProcessStatV0(pid)
	return state != "Z" && observedStartRef == startRef
}

func currentCodexAppServerTmuxLeaseIdentityV0() (int, string) {
	pid := os.Getpid()
	return pid, codexAppServerTmuxProcessStartRefV0(pid)
}

func codexAppServerTmuxOwnedProcessGroupV0(pid int) int {
	if pid <= 0 {
		return 0
	}
	pgid, err := syscall.Getpgid(pid)
	if err != nil || pgid != pid {
		return 0
	}
	return pgid
}

type codexAppServerTmuxSessionIdentityV0 struct {
	SessionID      string
	SessionCreated string
	PanePID        int
	PaneStartRef   string
}

func (identity codexAppServerTmuxSessionIdentityV0) completeV0() bool {
	return strings.TrimSpace(identity.SessionID) != "" &&
		strings.TrimSpace(identity.SessionCreated) != "" && identity.PanePID > 0 &&
		strings.TrimSpace(identity.PaneStartRef) != ""
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxExactSessionTargetV0() string {
	return "=" + strings.TrimSpace(backend.SessionName)
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxSessionIdentityV0(ctx context.Context, tmuxPath string) (codexAppServerTmuxSessionIdentityV0, error) {
	output, err := backend.runTmuxCommandV0(ctx, tmuxPath, "display-message", "-p", "-t", backend.tmuxExactSessionTargetV0(), "#{session_id}\t#{session_created}\t#{pane_pid}")
	if err != nil {
		return codexAppServerTmuxSessionIdentityV0{}, codexAppServerTmuxCommandErrorV0("codex_app_server_tmux_identity_failed", output, err)
	}
	parts := strings.Split(strings.TrimSpace(output), "\t")
	if len(parts) != 3 {
		return codexAppServerTmuxSessionIdentityV0{}, codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil || pid <= 0 {
		return codexAppServerTmuxSessionIdentityV0{}, codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	identity := codexAppServerTmuxSessionIdentityV0{
		SessionID: strings.TrimSpace(parts[0]), SessionCreated: strings.TrimSpace(parts[1]),
		PanePID: pid, PaneStartRef: codexAppServerTmuxProcessStartRefV0(pid),
	}
	if !identity.completeV0() {
		return codexAppServerTmuxSessionIdentityV0{}, codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	return identity, nil
}

func (marker codexAppServerTmuxOwnerMarkerV0) tmuxIdentityV0() codexAppServerTmuxSessionIdentityV0 {
	return codexAppServerTmuxSessionIdentityV0{
		SessionID: marker.TmuxSessionID, SessionCreated: marker.TmuxSessionCreated,
		PanePID: marker.TmuxPanePID, PaneStartRef: marker.TmuxPaneStartRef,
	}
}

func (marker codexAppServerTmuxOwnerMarkerV0) generationMarkerV0() bool {
	return strings.TrimSpace(marker.SchemaVersion) == codexAppServerTmuxOwnerSchemaGenerationV0 &&
		strings.TrimSpace(marker.GenerationRef) != ""
}

func (marker codexAppServerTmuxOwnerMarkerV0) leaseOwnedByCurrentProcessV0() bool {
	pid, startRef := currentCodexAppServerTmuxLeaseIdentityV0()
	return marker.LeaseOwnerPID == pid && strings.TrimSpace(marker.LeaseOwnerStartRef) == startRef && startRef != ""
}

func (marker codexAppServerTmuxOwnerMarkerV0) leaseOwnerAliveV0() bool {
	return codexAppServerTmuxProcessIdentityAliveV0(marker.LeaseOwnerPID, marker.LeaseOwnerStartRef)
}

func (marker codexAppServerTmuxOwnerMarkerV0) appServerAliveV0(socketPath string) bool {
	return codexAppServerTmuxProcessIdentityAliveV0(marker.AppServerPID, marker.AppServerStartRef) &&
		codexAppServerTmuxSocketListenerAliveV0(socketPath)
}

func codexAppServerTmuxConflictErrorV0(code string) error {
	return codexAppServerCallErrorV0{Code: code, Err: errors.New(code)}
}

func codexAppServerTmuxSocketListenerAliveV0(socketPath string) bool {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return false
	}
	info, err := os.Lstat(socketPath)
	if err != nil || info.Mode()&os.ModeSocket == 0 || info.Mode()&os.ModeSymlink != 0 {
		return false
	}
	connection, err := net.DialTimeout("unix", socketPath, 100*time.Millisecond)
	if err != nil {
		return false
	}
	_ = connection.Close()
	return true
}

// adoptOrFenceExistingGenerationV0 runs while owner.lease is held. It returns
// true when Ensure can reuse/adopt a responding generation without starting one.
func (backend serverCodexAppServerTmuxBackendV0) adoptOrFenceExistingGenerationV0(
	ctx context.Context,
	preflight serverCodexAppServerProbePortV0,
) (bool, error) {
	marker, markerOK := backend.readTmuxOwnerMarkerV0()
	socketPath := strings.TrimSpace(backend.SocketPath)
	listenerAlive := codexAppServerTmuxSocketListenerAliveV0(socketPath)
	responding := listenerAlive && preflight != nil && preflight.ProbeV0(ctx) == nil
	tmuxPath, tmuxErr := codexAppServerTmuxCommandPathV0(backend.PathEnv)
	hasSession := false
	if tmuxErr == nil {
		hasSession, _ = backend.tmuxHasSessionV0(ctx, tmuxPath)
	}

	if markerOK && marker.generationMarkerV0() {
		appAlive := marker.appServerAliveV0(socketPath)
		if responding && appAlive {
			if marker.leaseOwnedByCurrentProcessV0() {
				return true, nil
			}
			if marker.leaseOwnerAliveV0() {
				return false, codexAppServerTmuxConflictErrorV0(codexAppServerTmuxLeaseConflictV0)
			}
			if err := backend.claimTmuxOwnerLeaseV0(marker); err != nil {
				return false, err
			}
			return true, nil
		}
		if appAlive || listenerAlive || responding || (marker.leaseOwnerAliveV0() && !marker.leaseOwnedByCurrentProcessV0()) {
			return false, codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
		}
		if hasSession {
			identity, err := backend.tmuxSessionIdentityV0(ctx, tmuxPath)
			if err != nil || !reflect.DeepEqual(identity, marker.tmuxIdentityV0()) {
				return false, codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
			}
		}
		if err := backend.cleanupTmuxGenerationV0(ctx, marker, hasSession); err != nil {
			return false, err
		}
		return false, nil
	}

	if markerOK { // Legacy migration needs one exact tmux session and one socket.
		if responding && listenerAlive && hasSession {
			identity, identityErr := backend.tmuxSessionIdentityV0(ctx, tmuxPath)
			if identityErr != nil {
				return false, identityErr
			}
			upgraded, err := backend.newTmuxOwnerMarkerV0("", identity.PanePID)
			if err != nil {
				return false, err
			}
			upgraded.TmuxSessionID = identity.SessionID
			upgraded.TmuxSessionCreated = identity.SessionCreated
			upgraded.TmuxPanePID = identity.PanePID
			upgraded.TmuxPaneStartRef = identity.PaneStartRef
			if err := backend.replaceTmuxOwnerMarkerV0(marker, upgraded); err != nil {
				return false, err
			}
			return true, nil
		}
		return false, codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}

	if backend.tmuxOwnerMarkerObservedV0() || backend.tmuxLegacyOwnerMarkerObservedV0() || hasSession || listenerAlive || responding {
		return false, codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	if codexAppServerTmuxSocketPresentV0(socketPath) {
		if err := backend.removeStaleTmuxSocketV0(ctx, codexAppServerTmuxOwnerMarkerV0{}); err != nil {
			return false, err
		}
	}
	if len(codexAppServerTmuxSocketProcessPIDsV0(ctx, socketPath)) > 0 {
		return false, codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	return false, nil
}

func (backend serverCodexAppServerTmuxBackendV0) newTmuxOwnerMarkerV0(generationRef string, appPID int) (codexAppServerTmuxOwnerMarkerV0, error) {
	if strings.TrimSpace(generationRef) == "" {
		var err error
		generationRef, err = newCodexAppServerTmuxGenerationRefV0()
		if err != nil {
			return codexAppServerTmuxOwnerMarkerV0{}, codexAppServerCallErrorV0{Code: "codex_app_server_tmux_generation_unavailable", Err: err}
		}
	}
	leasePID, leaseStartRef := currentCodexAppServerTmuxLeaseIdentityV0()
	return codexAppServerTmuxOwnerMarkerV0{
		SchemaVersion:           codexAppServerTmuxOwnerSchemaGenerationV0,
		OwnerRef:                codexAppServerTmuxEvidenceOwnedV0,
		SessionName:             strings.TrimSpace(backend.SessionName),
		SocketRef:               "socket-ref-codex-goal-app-server-tmux",
		SocketPath:              strings.TrimSpace(backend.SocketPath),
		GenerationRef:           strings.TrimSpace(generationRef),
		LeaseOwnerPID:           leasePID,
		LeaseOwnerStartRef:      leaseStartRef,
		AppServerPID:            appPID,
		AppServerStartRef:       codexAppServerTmuxProcessStartRefV0(appPID),
		AppServerProcessGroupID: codexAppServerTmuxOwnedProcessGroupV0(appPID),
	}, nil
}

func (backend serverCodexAppServerTmuxBackendV0) claimTmuxOwnerLeaseV0(marker codexAppServerTmuxOwnerMarkerV0) error {
	claimed := marker
	claimed.LeaseOwnerPID, claimed.LeaseOwnerStartRef = currentCodexAppServerTmuxLeaseIdentityV0()
	return backend.replaceTmuxOwnerMarkerV0(marker, claimed)
}

func (backend serverCodexAppServerTmuxBackendV0) writeTmuxOwnerMarkerAtomicV0(marker codexAppServerTmuxOwnerMarkerV0) error {
	path := backend.tmuxOwnerMarkerPathV0()
	if path == "" {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_config_missing", Err: errors.New("codex_app_server_tmux_config_missing")}
	}
	raw, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".owner-*.tmp")
	if err != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_marker_unavailable", Err: err}
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(raw)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmpPath, path)
	}
	if err != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_marker_unavailable", Err: err}
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) replaceTmuxOwnerMarkerV0(
	expected codexAppServerTmuxOwnerMarkerV0,
	next codexAppServerTmuxOwnerMarkerV0,
) error {
	lock := codexAppServerTmuxMarkerCASLockV0(backend.tmuxOwnerMarkerPathV0())
	lock.Lock()
	defer lock.Unlock()
	path := backend.tmuxOwnerMarkerPathV0()
	if !expected.generationMarkerV0() {
		path = backend.tmuxLegacyOwnerMarkerPathV0()
	}
	current, ok := readCodexAppServerTmuxOwnerMarkerPathV0(path)
	if !ok || !reflect.DeepEqual(current, expected) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	if expected.generationMarkerV0() {
		return backend.writeTmuxOwnerMarkerAtomicV0(next)
	}
	if _, err := os.Lstat(backend.tmuxOwnerMarkerPathV0()); err == nil || !errors.Is(err, os.ErrNotExist) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	if err := backend.writeTmuxOwnerMarkerAtomicV0(next); err != nil {
		return err
	}
	current, ok = readCodexAppServerTmuxOwnerMarkerPathV0(path)
	if !ok || !reflect.DeepEqual(current, expected) {
		_ = os.Remove(backend.tmuxOwnerMarkerPathV0())
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	return os.Remove(path)
}

func (backend serverCodexAppServerTmuxBackendV0) removeTmuxOwnerMarkerExpectedV0(expected codexAppServerTmuxOwnerMarkerV0) error {
	lock := codexAppServerTmuxMarkerCASLockV0(backend.tmuxOwnerMarkerPathV0())
	lock.Lock()
	defer lock.Unlock()
	path := backend.tmuxOwnerMarkerPathV0()
	if !expected.generationMarkerV0() {
		path = backend.tmuxLegacyOwnerMarkerPathV0()
	}
	current, ok := readCodexAppServerTmuxOwnerMarkerPathV0(path)
	if !ok || !reflect.DeepEqual(current, expected) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	return os.Remove(path)
}

func (backend serverCodexAppServerTmuxBackendV0) cleanupTmuxGenerationV0(
	ctx context.Context,
	marker codexAppServerTmuxOwnerMarkerV0,
	killSession bool,
) error {
	current, ok := backend.readTmuxOwnerMarkerV0()
	if !ok || !reflect.DeepEqual(current, marker) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	tmuxPath, tmuxErr := codexAppServerTmuxCommandPathV0(backend.PathEnv)
	if tmuxErr == nil {
		hasSession, err := backend.tmuxHasSessionV0(ctx, tmuxPath)
		if err != nil {
			return err
		}
		if hasSession {
			identity, identityErr := backend.tmuxSessionIdentityV0(ctx, tmuxPath)
			if identityErr != nil || !marker.tmuxIdentityV0().completeV0() || !reflect.DeepEqual(identity, marker.tmuxIdentityV0()) {
				return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
			}
			if !killSession {
				return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
			}
			if err := backend.tmuxKillSessionV0(ctx, tmuxPath); err != nil {
				return err
			}
			if stillPresent, _ := backend.tmuxHasSessionV0(ctx, tmuxPath); stillPresent {
				return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
			}
			if err := waitCodexAppServerTmuxProcessIdentityGoneV0(ctx, marker.AppServerPID, marker.AppServerStartRef); err != nil {
				return err
			}
		}
	}
	// Without a matching tmux session there is no race-free primitive available
	// here to signal a PID. Preserve any live process and report a conflict.
	if codexAppServerTmuxProcessIdentityAliveV0(marker.AppServerPID, marker.AppServerStartRef) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	if err := backend.removeStaleTmuxSocketV0(ctx, marker); err != nil {
		return err
	}
	current, ok = backend.readTmuxOwnerMarkerV0()
	if !ok || !reflect.DeepEqual(current, marker) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	_ = os.Remove(backend.tmuxStdinPathV0())
	return backend.removeTmuxOwnerMarkerExpectedV0(marker)
}

func waitCodexAppServerTmuxProcessIdentityGoneV0(ctx context.Context, pid int, startRef string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(codexAppServerTmuxSocketPollEveryV0)
	defer ticker.Stop()
	for codexAppServerTmuxProcessIdentityAliveV0(pid, startRef) {
		select {
		case <-ctx.Done():
			return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_pane_exit_timeout", Err: ctx.Err()}
		case <-ticker.C:
		}
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) removeStaleTmuxSocketV0(ctx context.Context, expected codexAppServerTmuxOwnerMarkerV0) error {
	socketPath := strings.TrimSpace(backend.SocketPath)
	info, err := os.Lstat(socketPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || info.Mode()&os.ModeSocket == 0 || info.Mode()&os.ModeSymlink != 0 {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	validateOwner := func() bool {
		if expected.generationMarkerV0() {
			current, ok := backend.readTmuxOwnerMarkerV0()
			return ok && reflect.DeepEqual(current, expected) &&
				!codexAppServerTmuxProcessIdentityAliveV0(expected.AppServerPID, expected.AppServerStartRef)
		}
		return !backend.tmuxOwnerMarkerObservedV0() && !backend.tmuxLegacyOwnerMarkerObservedV0()
	}
	if !validateOwner() || codexAppServerTmuxSocketListenerAliveV0(socketPath) || len(codexAppServerTmuxSocketProcessPIDsV0(ctx, socketPath)) != 0 {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	revalidated, err := os.Lstat(socketPath)
	if err != nil || !os.SameFile(info, revalidated) || !validateOwner() ||
		codexAppServerTmuxSocketListenerAliveV0(socketPath) || len(codexAppServerTmuxSocketProcessPIDsV0(ctx, socketPath)) != 0 {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	if err := os.Remove(socketPath); err != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_socket_cleanup_failed", Err: err}
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) cleanupStartedTmuxGenerationV0(marker codexAppServerTmuxOwnerMarkerV0) {
	ctx, cancel := context.WithTimeout(context.Background(), codexAppServerTmuxDefaultTimeoutV0)
	defer cancel()
	_ = backend.cleanupTmuxGenerationV0(ctx, marker, true)
}

func (backend serverCodexAppServerTmuxBackendV0) shutdownTmuxGenerationLeaseV0(
	ctx context.Context,
	marker codexAppServerTmuxOwnerMarkerV0,
	allowTakeover bool,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
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
	current, ok := backend.readTmuxOwnerMarkerV0()
	if !ok || !current.generationMarkerV0() || !reflect.DeepEqual(current, marker) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	if !current.leaseOwnedByCurrentProcessV0() {
		if current.leaseOwnerAliveV0() {
			return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxLeaseConflictV0)
		}
		if !allowTakeover && current.appServerAliveV0(backend.SocketPath) {
			// Normal shutdown may safely adopt an abandoned lease only when it is
			// the same configured app-server generation.
			allowTakeover = true
		}
		if !allowTakeover {
			return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxLeaseConflictV0)
		}
		if err := backend.claimTmuxOwnerLeaseV0(current); err != nil {
			return err
		}
		current, _ = backend.readTmuxOwnerMarkerV0()
	}
	return backend.cleanupTmuxGenerationV0(runCtx, current, true)
}
