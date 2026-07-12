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
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	codexAppServerTmuxOwnerSchemaGenerationV0 = "orquesta_codex_app_server_tmux_owner_generation.v0"
	codexAppServerTmuxGenerationConflictV0    = "codex_app_server_tmux_generation_conflict"
	codexAppServerTmuxObservationTransientV0  = "codex_app_server_tmux_generation_observation_transient"
	codexAppServerTmuxLeaseConflictV0         = "codex_app_server_tmux_owner_lease_conflict"
	codexAppServerTmuxGenerationEnvironmentV0 = "ORQUESTA_CODEX_APP_SERVER_GENERATION_REF"
)

var (
	errCodexAppServerTmuxSocketInodeNotObservedV0 = errors.New("codex_app_server_tmux_socket_inode_not_observed")
	errCodexAppServerTmuxSocketOwnerOutsidePaneV0 = errors.New("codex_app_server_tmux_socket_owner_outside_pane")
	errCodexAppServerTmuxSocketOwnerAmbiguousV0   = errors.New("codex_app_server_tmux_socket_owner_ambiguous")
)

type codexAppServerTmuxGenerationObservationV0 uint8

const (
	codexAppServerTmuxGenerationTransientV0 codexAppServerTmuxGenerationObservationV0 = iota
	codexAppServerTmuxGenerationVerifiedV0
	codexAppServerTmuxGenerationContradictedV0
)

type codexAppServerTmuxLeaseGuardV0 struct {
	file *os.File
	path string
}

func (guard *codexAppServerTmuxLeaseGuardV0) releaseV0() {
	if guard == nil || guard.file == nil {
		return
	}
	_ = syscall.Flock(int(guard.file.Fd()), syscall.LOCK_UN)
	_ = guard.file.Close()
	guard.file = nil
	guard.path = ""
}

func (backend serverCodexAppServerTmuxBackendV0) acquireTmuxLeaseGuardV0(ctx context.Context) (*codexAppServerTmuxLeaseGuardV0, error) {
	path := backend.tmuxLeasePathV0()
	if path == "" || !codexAppServerTmuxPrivateParentV0(path) {
		return nil, codexAppServerCallErrorV0{Code: "codex_app_server_tmux_owner_lease_unavailable", Err: errors.New("codex_app_server_tmux_owner_lease_parent_not_private")}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, codexAppServerCallErrorV0{Code: "codex_app_server_tmux_owner_lease_unavailable", Err: err}
	}
	info, statErr := file.Stat()
	pathInfo, pathErr := os.Lstat(path)
	var stat *syscall.Stat_t
	statOK := false
	if statErr == nil {
		stat, statOK = info.Sys().(*syscall.Stat_t)
	}
	if statErr != nil || pathErr != nil || !info.Mode().IsRegular() || pathInfo.Mode()&os.ModeSymlink != 0 ||
		!os.SameFile(info, pathInfo) || info.Mode().Perm() != 0o600 || !statOK || stat.Uid != uint32(os.Getuid()) {
		_ = file.Close()
		if statErr == nil {
			statErr = errors.New("codex_app_server_tmux_owner_lease_identity_invalid")
		}
		return nil, codexAppServerCallErrorV0{Code: "codex_app_server_tmux_owner_lease_unavailable", Err: statErr}
	}
	for {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return &codexAppServerTmuxLeaseGuardV0{file: file, path: filepath.Clean(path)}, nil
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
	startRef, _ := codexAppServerTmuxProcessStatAtV0("/proc", pid)
	return startRef
}

func codexAppServerTmuxProcessStatV0(pid int) (string, string) {
	return codexAppServerTmuxProcessStatAtV0("/proc", pid)
}

func codexAppServerTmuxProcessStatAtV0(procRoot string, pid int) (string, string) {
	if pid <= 0 {
		return "", ""
	}
	raw, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "stat"))
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
	// display-message expects a target-pane. Keep the exact session selector,
	// then select its current window/pane explicitly. tmux 3.6 accepts
	// "=session" but expands pane/session formats to empty values.
	return "=" + strings.TrimSpace(backend.SessionName) + ":"
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxSessionIdentityV0(ctx context.Context, tmuxPath string) (codexAppServerTmuxSessionIdentityV0, error) {
	return backend.tmuxSessionIdentityTargetV0(ctx, tmuxPath, backend.tmuxExactSessionTargetV0())
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxSessionIdentityTargetV0(ctx context.Context, tmuxPath, target string) (codexAppServerTmuxSessionIdentityV0, error) {
	output, err := backend.runTmuxCommandV0(ctx, tmuxPath, "display-message", "-p", "-t", strings.TrimSpace(target), "#{session_id}\t#{session_created}\t#{pane_pid}")
	if err != nil {
		return codexAppServerTmuxSessionIdentityV0{}, codexAppServerTmuxCommandErrorV0("codex_app_server_tmux_identity_failed", output, err)
	}
	return codexAppServerTmuxSessionIdentityFromOutputV0(output)
}

func codexAppServerTmuxSessionIdentityFromOutputV0(output string) (codexAppServerTmuxSessionIdentityV0, error) {
	parts := strings.Split(strings.TrimSpace(output), "\t")
	// tmux 3.3a does not expand `\t` in new-session -P -F; newer versions do.
	// The pipe form is portable across both families, while accepting tabs
	// keeps compatibility with already deployed wrappers and test fixtures.
	if len(parts) != 3 {
		parts = strings.Split(strings.TrimSpace(output), "|")
	}
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

func (backend serverCodexAppServerTmuxBackendV0) verifyTmuxGenerationTokenV0(ctx context.Context, tmuxPath, sessionID, generationRef string) error {
	observation := backend.observeTmuxGenerationTokenV0(ctx, tmuxPath, sessionID, generationRef)
	if observation == codexAppServerTmuxGenerationVerifiedV0 {
		return nil
	}
	if observation == codexAppServerTmuxGenerationContradictedV0 {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxObservationTransientV0)
}

func (backend serverCodexAppServerTmuxBackendV0) observeTmuxGenerationTokenV0(ctx context.Context, tmuxPath, sessionID, generationRef string) codexAppServerTmuxGenerationObservationV0 {
	output, err := backend.runTmuxCommandV0(ctx, tmuxPath, "show-environment", "-t", strings.TrimSpace(sessionID), codexAppServerTmuxGenerationEnvironmentV0)
	if err != nil {
		return codexAppServerTmuxGenerationTransientV0
	}
	want := codexAppServerTmuxGenerationEnvironmentV0 + "=" + strings.TrimSpace(generationRef)
	observed := strings.TrimSpace(output)
	if observed == want {
		return codexAppServerTmuxGenerationVerifiedV0
	}
	if observed == "" || observed == "-"+codexAppServerTmuxGenerationEnvironmentV0 {
		return codexAppServerTmuxGenerationTransientV0
	}
	return codexAppServerTmuxGenerationContradictedV0
}

func (backend serverCodexAppServerTmuxBackendV0) observeRecordedTmuxGenerationV0(
	ctx context.Context,
	tmuxPath string,
	marker codexAppServerTmuxOwnerMarkerV0,
) codexAppServerTmuxGenerationObservationV0 {
	identity, err := backend.tmuxSessionIdentityTargetV0(ctx, tmuxPath, marker.TmuxSessionID)
	if err != nil || !identity.completeV0() {
		return codexAppServerTmuxGenerationTransientV0
	}
	if !reflect.DeepEqual(identity, marker.tmuxIdentityV0()) {
		return codexAppServerTmuxGenerationContradictedV0
	}
	return backend.observeTmuxGenerationTokenV0(ctx, tmuxPath, marker.TmuxSessionID, marker.GenerationRef)
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
	return marker.observeAppServerV0(socketPath) == codexAppServerTmuxGenerationVerifiedV0
}

func (marker codexAppServerTmuxOwnerMarkerV0) observeAppServerV0(socketPath string) codexAppServerTmuxGenerationObservationV0 {
	if marker.SocketOwnerPID <= 0 || marker.SocketOwnerStartRef == "" ||
		marker.AppServerPID != marker.SocketOwnerPID || marker.AppServerStartRef != marker.SocketOwnerStartRef ||
		!codexAppServerTmuxProcessIdentityAliveV0(marker.TmuxPanePID, marker.TmuxPaneStartRef) ||
		!codexAppServerTmuxProcessIdentityAliveV0(marker.SocketOwnerPID, marker.SocketOwnerStartRef) ||
		!codexAppServerTmuxProcessDescendsFromV0(marker.SocketOwnerPID, marker.TmuxPanePID) {
		return codexAppServerTmuxGenerationTransientV0
	}
	owner, err := codexAppServerTmuxSocketOwnerDescendantV0(socketPath, marker.TmuxPanePID)
	if errors.Is(err, errCodexAppServerTmuxSocketOwnerOutsidePaneV0) {
		return codexAppServerTmuxGenerationContradictedV0
	}
	if err != nil {
		return codexAppServerTmuxGenerationTransientV0
	}
	if owner.PID != marker.SocketOwnerPID || owner.StartRef != marker.SocketOwnerStartRef {
		return codexAppServerTmuxGenerationContradictedV0
	}
	return codexAppServerTmuxGenerationVerifiedV0
}

type codexAppServerTmuxProcessIdentityV0 struct {
	PID      int
	StartRef string
}

func codexAppServerTmuxProcessParentPIDAtV0(procRoot string, pid int) int {
	if pid <= 0 {
		return 0
	}
	raw, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0
	}
	text := string(raw)
	end := strings.LastIndex(text, ")")
	if end < 0 {
		return 0
	}
	fields := strings.Fields(text[end+1:])
	if len(fields) < 2 {
		return 0
	}
	parent, _ := strconv.Atoi(fields[1])
	return parent
}

func codexAppServerTmuxProcessDescendsFromV0(pid, ancestor int) bool {
	return codexAppServerTmuxProcessDescendsFromAtV0("/proc", pid, ancestor)
}

func codexAppServerTmuxProcessDescendsFromAtV0(procRoot string, pid, ancestor int) bool {
	if pid <= 0 || ancestor <= 0 {
		return false
	}
	seen := map[int]struct{}{}
	for current := pid; current > 0; current = codexAppServerTmuxProcessParentPIDAtV0(procRoot, current) {
		if current == ancestor {
			return true
		}
		if _, exists := seen[current]; exists {
			return false
		}
		seen[current] = struct{}{}
	}
	return false
}

func codexAppServerTmuxSocketInodeAtV0(procRoot, socketPath string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(procRoot, "net", "unix"))
	if err != nil {
		return "", err
	}
	want := strings.TrimSpace(socketPath)
	inodes := map[string]struct{}{}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 8 || strings.Join(fields[7:], " ") != want {
			continue
		}
		flags, flagsErr := strconv.ParseUint(fields[3], 16, 32)
		socketType, typeErr := strconv.ParseUint(fields[4], 16, 32)
		// /proc/net/unix also reports connected peers with the listener's
		// pathname. Only SO_ACCEPTCON stream entries identify the inode bound
		// to the live filesystem listener.
		if flagsErr != nil || typeErr != nil || flags&0x00010000 == 0 || socketType != syscall.SOCK_STREAM {
			continue
		}
		if inode := strings.TrimSpace(fields[6]); inode != "" {
			inodes[inode] = struct{}{}
		}
	}
	if len(inodes) == 0 {
		return "", errCodexAppServerTmuxSocketInodeNotObservedV0
	}
	if len(inodes) != 1 {
		return "", errors.New("codex_app_server_tmux_socket_inode_ambiguous")
	}
	for inode := range inodes {
		return inode, nil
	}
	return "", errCodexAppServerTmuxSocketInodeNotObservedV0
}

func codexAppServerTmuxSocketOwnerIdentitiesV0(socketPath string) ([]codexAppServerTmuxProcessIdentityV0, error) {
	return codexAppServerTmuxSocketOwnerIdentitiesAtV0("/proc", socketPath)
}

func codexAppServerTmuxSocketOwnerIdentitiesAtV0(procRoot, socketPath string) ([]codexAppServerTmuxProcessIdentityV0, error) {
	inode, err := codexAppServerTmuxSocketInodeAtV0(procRoot, socketPath)
	if err != nil {
		return nil, err
	}
	want := "socket:[" + inode + "]"
	procEntries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil, err
	}
	owners := []codexAppServerTmuxProcessIdentityV0{}
	for _, entry := range procEntries {
		pid, parseErr := strconv.Atoi(entry.Name())
		if parseErr != nil || pid <= 0 {
			continue
		}
		fds, readErr := os.ReadDir(filepath.Join(procRoot, entry.Name(), "fd"))
		if readErr != nil {
			continue
		}
		matched := false
		for _, fd := range fds {
			target, linkErr := os.Readlink(filepath.Join(procRoot, entry.Name(), "fd", fd.Name()))
			if linkErr == nil && target == want {
				matched = true
				break
			}
		}
		if matched {
			if startRef, _ := codexAppServerTmuxProcessStatAtV0(procRoot, pid); startRef != "" {
				owners = append(owners, codexAppServerTmuxProcessIdentityV0{PID: pid, StartRef: startRef})
			}
		}
	}
	sort.Slice(owners, func(i, j int) bool { return owners[i].PID < owners[j].PID })
	return owners, nil
}

func codexAppServerTmuxSocketOwnerDescendantV0(socketPath string, panePID int) (codexAppServerTmuxProcessIdentityV0, error) {
	return codexAppServerTmuxSocketOwnerDescendantAtV0("/proc", socketPath, panePID)
}

func codexAppServerTmuxSocketOwnerDescendantAtV0(procRoot, socketPath string, panePID int) (codexAppServerTmuxProcessIdentityV0, error) {
	owners, err := codexAppServerTmuxSocketOwnerIdentitiesAtV0(procRoot, socketPath)
	if err != nil {
		return codexAppServerTmuxProcessIdentityV0{}, err
	}
	for _, owner := range owners {
		if !codexAppServerTmuxProcessDescendsFromAtV0(procRoot, owner.PID, panePID) {
			return codexAppServerTmuxProcessIdentityV0{}, errCodexAppServerTmuxSocketOwnerOutsidePaneV0
		}
	}
	// Fork/exec wrappers can briefly retain the same listener FD as their
	// descendant. The unique deepest holder is the causal app-server owner;
	// unrelated leaves remain ambiguous.
	leaves := []codexAppServerTmuxProcessIdentityV0{}
	for _, candidate := range owners {
		ancestorOfAnotherOwner := false
		for _, other := range owners {
			if candidate.PID != other.PID && codexAppServerTmuxProcessDescendsFromAtV0(procRoot, other.PID, candidate.PID) {
				ancestorOfAnotherOwner = true
				break
			}
		}
		if !ancestorOfAnotherOwner {
			leaves = append(leaves, candidate)
		}
	}
	if len(leaves) != 1 {
		return codexAppServerTmuxProcessIdentityV0{}, errCodexAppServerTmuxSocketOwnerAmbiguousV0
	}
	return leaves[0], nil
}

func codexAppServerTmuxConflictErrorV0(code string) error {
	return codexAppServerCallErrorV0{Code: code, Err: errors.New(code)}
}

func codexAppServerTmuxIsGenerationConflictV0(err error) bool {
	var callErr codexAppServerCallErrorV0
	return errors.As(err, &callErr) && callErr.Code == codexAppServerTmuxGenerationConflictV0
}

func codexAppServerTmuxIsObservationTransientV0(err error) bool {
	var callErr codexAppServerCallErrorV0
	return errors.As(err, &callErr) && callErr.Code == codexAppServerTmuxObservationTransientV0
}

func codexAppServerTmuxObservationErrorV0() error {
	return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxObservationTransientV0)
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
	lease *codexAppServerTmuxLeaseGuardV0,
) (bool, error) {
	marker, markerOK := backend.readTmuxOwnerMarkerV0()
	socketPath := strings.TrimSpace(backend.SocketPath)
	listenerAlive := codexAppServerTmuxSocketListenerAliveV0(socketPath)
	responding := listenerAlive && preflight != nil && preflight.ProbeV0(ctx) == nil
	tmuxPath, tmuxErr := codexAppServerTmuxCommandPathV0(backend.PathEnv)
	hasSession := false
	var sessionObservationErr error
	if tmuxErr == nil {
		hasSession, sessionObservationErr = backend.tmuxHasSessionV0(ctx, tmuxPath)
	}

	if markerOK && marker.generationMarkerV0() {
		if tmuxErr != nil {
			return false, tmuxErr
		}
		if sessionObservationErr != nil {
			return false, sessionObservationErr
		}
		if !marker.tmuxIdentityV0().completeV0() && hasSession && tmuxErr == nil {
			identity, identityErr := backend.tmuxSessionIdentityV0(ctx, tmuxPath)
			if identityErr != nil || backend.verifyTmuxGenerationTokenV0(ctx, tmuxPath, identity.SessionID, marker.GenerationRef) != nil {
				return false, codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
			}
			recovered := marker
			recovered.TmuxSessionID = identity.SessionID
			recovered.TmuxSessionCreated = identity.SessionCreated
			recovered.TmuxPanePID = identity.PanePID
			recovered.TmuxPaneStartRef = identity.PaneStartRef
			if owner, ownerErr := codexAppServerTmuxSocketOwnerDescendantV0(socketPath, identity.PanePID); ownerErr == nil {
				recovered.SocketOwnerPID = owner.PID
				recovered.SocketOwnerStartRef = owner.StartRef
				recovered.AppServerPID = owner.PID
				recovered.AppServerStartRef = owner.StartRef
				recovered.AppServerProcessGroupID = codexAppServerTmuxOwnedProcessGroupV0(owner.PID)
			}
			if err := backend.replaceTmuxOwnerMarkerWithLeaseV0(lease, marker, recovered); err != nil {
				return false, err
			}
			marker = recovered
			listenerAlive = codexAppServerTmuxSocketListenerAliveV0(socketPath)
			responding = listenerAlive && preflight != nil && preflight.ProbeV0(ctx) == nil
		}
		if hasSession {
			identity, identityErr := backend.tmuxSessionIdentityV0(ctx, tmuxPath)
			if identityErr != nil || !reflect.DeepEqual(identity, marker.tmuxIdentityV0()) ||
				backend.verifyTmuxGenerationTokenV0(ctx, tmuxPath, marker.TmuxSessionID, marker.GenerationRef) != nil {
				return false, codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
			}
		}
		appAlive := marker.appServerAliveV0(socketPath)
		if responding && appAlive {
			if !hasSession && !marker.tmuxIdentityV0().completeV0() {
				return false, codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
			}
			if marker.leaseOwnedByCurrentProcessV0() {
				return true, nil
			}
			if marker.leaseOwnerAliveV0() {
				return false, codexAppServerTmuxConflictErrorV0(codexAppServerTmuxLeaseConflictV0)
			}
			if err := backend.claimTmuxOwnerLeaseWithLeaseV0(lease, marker); err != nil {
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
		if err := backend.cleanupTmuxGenerationWithLeaseV0(ctx, lease, marker, hasSession); err != nil {
			return false, err
		}
		return false, nil
	}

	if markerOK { // Legacy markers are read-only evidence; never mutate by path.
		return false, codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}

	if backend.tmuxOwnerMarkerObservedV0() || backend.tmuxLegacyOwnerMarkerObservedV0() || hasSession || listenerAlive || responding {
		return false, codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	if codexAppServerTmuxSocketPresentV0(socketPath) {
		if err := backend.removeStaleTmuxSocketWithLeaseV0(ctx, lease, codexAppServerTmuxOwnerMarkerV0{}); err != nil {
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

func (backend serverCodexAppServerTmuxBackendV0) claimTmuxOwnerLeaseWithLeaseV0(guard *codexAppServerTmuxLeaseGuardV0, marker codexAppServerTmuxOwnerMarkerV0) error {
	claimed := marker
	claimed.LeaseOwnerPID, claimed.LeaseOwnerStartRef = currentCodexAppServerTmuxLeaseIdentityV0()
	return backend.replaceTmuxOwnerMarkerWithLeaseV0(guard, marker, claimed)
}

func (backend serverCodexAppServerTmuxBackendV0) requireTmuxLeaseV0(guard *codexAppServerTmuxLeaseGuardV0) error {
	if guard == nil || guard.file == nil || filepath.Clean(guard.path) != filepath.Clean(backend.tmuxLeasePathV0()) {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_owner_lease_required", Err: errors.New("codex_app_server_tmux_owner_lease_required")}
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) writeTmuxOwnerMarkerAtomicWithLeaseV0(guard *codexAppServerTmuxLeaseGuardV0, marker codexAppServerTmuxOwnerMarkerV0) error {
	if err := backend.requireTmuxLeaseV0(guard); err != nil {
		return err
	}
	path := backend.tmuxOwnerMarkerPathV0()
	if path == "" {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_config_missing", Err: errors.New("codex_app_server_tmux_config_missing")}
	}
	if !backend.tmuxGenerationMarkerMatchesBackendV0(marker) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
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
	ctx, cancel := context.WithTimeout(context.Background(), codexAppServerTmuxDefaultTimeoutV0)
	defer cancel()
	guard, err := backend.acquireTmuxLeaseGuardV0(ctx)
	if err != nil {
		return err
	}
	defer guard.releaseV0()
	return backend.replaceTmuxOwnerMarkerWithLeaseV0(guard, expected, next)
}

func (backend serverCodexAppServerTmuxBackendV0) replaceTmuxOwnerMarkerWithLeaseV0(
	guard *codexAppServerTmuxLeaseGuardV0,
	expected codexAppServerTmuxOwnerMarkerV0,
	next codexAppServerTmuxOwnerMarkerV0,
) error {
	if err := backend.requireTmuxLeaseV0(guard); err != nil {
		return err
	}
	path := backend.tmuxOwnerMarkerPathV0()
	if !backend.tmuxGenerationMarkerMatchesBackendV0(expected) || !backend.tmuxGenerationMarkerMatchesBackendV0(next) ||
		strings.TrimSpace(next.GenerationRef) != strings.TrimSpace(expected.GenerationRef) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	current, ok := readCodexAppServerTmuxOwnerMarkerPathV0(path)
	if !ok || !reflect.DeepEqual(current, expected) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	return backend.writeTmuxOwnerMarkerAtomicWithLeaseV0(guard, next)
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxGenerationMarkerMatchesBackendV0(marker codexAppServerTmuxOwnerMarkerV0) bool {
	return marker.generationMarkerV0() &&
		strings.TrimSpace(marker.OwnerRef) == codexAppServerTmuxEvidenceOwnedV0 &&
		strings.TrimSpace(marker.SessionName) == strings.TrimSpace(backend.SessionName) &&
		filepath.Clean(strings.TrimSpace(marker.SocketPath)) == filepath.Clean(strings.TrimSpace(backend.SocketPath))
}

func (backend serverCodexAppServerTmuxBackendV0) removeTmuxOwnerMarkerExpectedWithLeaseV0(guard *codexAppServerTmuxLeaseGuardV0, expected codexAppServerTmuxOwnerMarkerV0) error {
	if err := backend.requireTmuxLeaseV0(guard); err != nil {
		return err
	}
	path := backend.tmuxOwnerMarkerPathV0()
	if !expected.generationMarkerV0() || filepath.Clean(strings.TrimSpace(expected.SocketPath)) != filepath.Clean(strings.TrimSpace(backend.SocketPath)) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	current, ok := readCodexAppServerTmuxOwnerMarkerPathV0(path)
	if !ok || !reflect.DeepEqual(current, expected) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	identity, err := readCodexAppServerTmuxPathIdentityV0(path)
	if err != nil {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	return backend.quarantineExactPathWithLeaseV0(guard, path, identity, func(candidate string) bool {
		current, ok := readCodexAppServerTmuxOwnerMarkerPathV0(candidate)
		return ok && reflect.DeepEqual(current, expected)
	})
}

type codexAppServerTmuxPathIdentityV0 struct {
	Device uint64
	Inode  uint64
	Mode   os.FileMode
}

func readCodexAppServerTmuxPathIdentityV0(path string) (codexAppServerTmuxPathIdentityV0, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return codexAppServerTmuxPathIdentityV0{}, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return codexAppServerTmuxPathIdentityV0{}, errors.New("codex_app_server_tmux_path_identity_unavailable")
	}
	return codexAppServerTmuxPathIdentityV0{Device: uint64(stat.Dev), Inode: stat.Ino, Mode: info.Mode()}, nil
}

func codexAppServerTmuxPrivateParentV0(path string) bool {
	parent := filepath.Dir(filepath.Clean(path))
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o700 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Getuid())
}

func (backend serverCodexAppServerTmuxBackendV0) quarantineExactPathWithLeaseV0(
	guard *codexAppServerTmuxLeaseGuardV0,
	path string,
	expected codexAppServerTmuxPathIdentityV0,
	validate func(string) bool,
) error {
	if err := backend.requireTmuxLeaseV0(guard); err != nil {
		return err
	}
	if filepath.Clean(filepath.Dir(path)) != filepath.Clean(filepath.Dir(backend.SocketPath)) ||
		!codexAppServerTmuxPrivateParentV0(path) || validate == nil || !validate(path) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	if backend.beforePathQuarantineV0 != nil {
		backend.beforePathQuarantineV0(path)
	}
	current, err := readCodexAppServerTmuxPathIdentityV0(path)
	if err != nil || current != expected {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	ref, err := newCodexAppServerTmuxGenerationRefV0()
	if err != nil {
		return err
	}
	quarantine := filepath.Join(filepath.Dir(path), ".orquesta-quarantine-"+filepath.Base(path)+"-"+strings.TrimPrefix(ref, "generation-ref-"))
	if _, err := os.Lstat(quarantine); !errors.Is(err, os.ErrNotExist) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	if err := os.Rename(path, quarantine); err != nil {
		return codexAppServerCallErrorV0{Code: "codex_app_server_tmux_quarantine_failed", Err: err}
	}
	quarantined, identityErr := readCodexAppServerTmuxPathIdentityV0(quarantine)
	valid := identityErr == nil && quarantined == expected && validate(quarantine)
	if !valid {
		if _, sourceErr := os.Lstat(path); errors.Is(sourceErr, os.ErrNotExist) {
			_ = os.Rename(quarantine, path)
		}
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	// Keep the exact inode quarantined. unlinkat by pathname cannot prove that a
	// concurrent replacement was not substituted; a private residual is safer.
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) cleanupTmuxGenerationV0(
	ctx context.Context,
	marker codexAppServerTmuxOwnerMarkerV0,
	killSession bool,
) error {
	guard, err := backend.acquireTmuxLeaseGuardV0(ctx)
	if err != nil {
		return err
	}
	defer guard.releaseV0()
	return backend.cleanupTmuxGenerationWithLeaseV0(ctx, guard, marker, killSession)
}

func (backend serverCodexAppServerTmuxBackendV0) cleanupTmuxGenerationWithLeaseV0(
	ctx context.Context,
	guard *codexAppServerTmuxLeaseGuardV0,
	marker codexAppServerTmuxOwnerMarkerV0,
	killSession bool,
) error {
	if err := backend.requireTmuxLeaseV0(guard); err != nil {
		return err
	}
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
			if !marker.tmuxIdentityV0().completeV0() {
				return codexAppServerTmuxObservationErrorV0()
			}
			observation := backend.observeRecordedTmuxGenerationV0(ctx, tmuxPath, marker)
			if observation == codexAppServerTmuxGenerationContradictedV0 {
				return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
			}
			if !killSession {
				return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
			}
			if backend.observeTmuxGenerationTokenV0(ctx, tmuxPath, marker.TmuxSessionID, marker.GenerationRef) != codexAppServerTmuxGenerationVerifiedV0 {
				return codexAppServerTmuxObservationErrorV0()
			}
			if err := backend.tmuxKillRecordedGenerationV0(ctx, tmuxPath, marker); err != nil {
				return err
			}
			if stillPresent, _ := backend.tmuxHasSessionTargetV0(ctx, tmuxPath, marker.TmuxSessionID); stillPresent {
				return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
			}
			if err := backend.stopExactTmuxGenerationProcessV0(ctx, guard, marker); err != nil {
				return err
			}
		}
	}
	// Without a matching tmux session there is no race-free primitive available
	// here to signal a PID. Preserve any live process and report a conflict.
	if codexAppServerTmuxProcessIdentityAliveV0(marker.AppServerPID, marker.AppServerStartRef) {
		return codexAppServerTmuxObservationErrorV0()
	}
	if err := backend.removeStaleTmuxSocketWithLeaseV0(ctx, guard, marker); err != nil {
		return err
	}
	current, ok = backend.readTmuxOwnerMarkerV0()
	if !ok || !reflect.DeepEqual(current, marker) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	return backend.removeTmuxOwnerMarkerExpectedWithLeaseV0(guard, marker)
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxKillRecordedGenerationV0(
	ctx context.Context,
	tmuxPath string,
	marker codexAppServerTmuxOwnerMarkerV0,
) error {
	if !marker.tmuxIdentityV0().completeV0() ||
		backend.observeTmuxGenerationTokenV0(ctx, tmuxPath, marker.TmuxSessionID, marker.GenerationRef) != codexAppServerTmuxGenerationVerifiedV0 {
		return codexAppServerTmuxObservationErrorV0()
	}
	if backend.beforeSessionKillV0 != nil {
		backend.beforeSessionKillV0()
	}
	if backend.observeTmuxGenerationTokenV0(ctx, tmuxPath, marker.TmuxSessionID, marker.GenerationRef) != codexAppServerTmuxGenerationVerifiedV0 {
		return codexAppServerTmuxObservationErrorV0()
	}
	output, err := backend.runTmuxCommandV0(ctx, tmuxPath, "kill-session", "-t", marker.TmuxSessionID)
	if err != nil {
		return codexAppServerTmuxCommandErrorV0("codex_app_server_tmux_kill_failed", output, err)
	}
	return nil
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
	guard, err := backend.acquireTmuxLeaseGuardV0(ctx)
	if err != nil {
		return err
	}
	defer guard.releaseV0()
	return backend.removeStaleTmuxSocketWithLeaseV0(ctx, guard, expected)
}

func (backend serverCodexAppServerTmuxBackendV0) removeStaleTmuxSocketWithLeaseV0(ctx context.Context, guard *codexAppServerTmuxLeaseGuardV0, expected codexAppServerTmuxOwnerMarkerV0) error {
	if err := backend.requireTmuxLeaseV0(guard); err != nil {
		return err
	}
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
				!codexAppServerTmuxProcessIdentityAliveV0(expected.SocketOwnerPID, expected.SocketOwnerStartRef)
		}
		return !backend.tmuxOwnerMarkerObservedV0() && !backend.tmuxLegacyOwnerMarkerObservedV0()
	}
	owners, ownerErr := codexAppServerTmuxSocketOwnerIdentitiesV0(socketPath)
	if !validateOwner() || codexAppServerTmuxSocketListenerAliveV0(socketPath) ||
		(ownerErr != nil && !errors.Is(ownerErr, errCodexAppServerTmuxSocketInodeNotObservedV0)) ||
		(ownerErr == nil && len(owners) != 0) {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	identity, err := readCodexAppServerTmuxPathIdentityV0(socketPath)
	if err != nil {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	if backend.beforeSocketQuarantineV0 != nil {
		backend.beforeSocketQuarantineV0()
	}
	if err := backend.quarantineExactPathWithLeaseV0(guard, socketPath, identity, func(candidate string) bool {
		candidateInfo, statErr := os.Lstat(candidate)
		return statErr == nil && candidateInfo.Mode()&os.ModeSocket != 0 && candidateInfo.Mode()&os.ModeSymlink == 0
	}); err != nil {
		return err
	}
	if !validateOwner() {
		return codexAppServerTmuxConflictErrorV0(codexAppServerTmuxGenerationConflictV0)
	}
	return nil
}

func (backend serverCodexAppServerTmuxBackendV0) cleanupStartedTmuxGenerationWithLeaseV0(guard *codexAppServerTmuxLeaseGuardV0, marker codexAppServerTmuxOwnerMarkerV0) {
	ctx, cancel := context.WithTimeout(context.Background(), codexAppServerTmuxDefaultTimeoutV0)
	defer cancel()
	_ = backend.cleanupTmuxGenerationWithLeaseV0(ctx, guard, marker, true)
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
		if err := backend.claimTmuxOwnerLeaseWithLeaseV0(guard, current); err != nil {
			return err
		}
		current, _ = backend.readTmuxOwnerMarkerV0()
	}
	return backend.cleanupTmuxGenerationWithLeaseV0(runCtx, guard, current, true)
}
