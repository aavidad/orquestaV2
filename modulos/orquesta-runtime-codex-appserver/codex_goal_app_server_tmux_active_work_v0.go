package orquestaruntimecodexappserver

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

func (backend serverCodexAppServerTmuxBackendV0) ReadActiveShutdownWorkV0(
	ctx context.Context,
	request orquestaservershutdown.ActiveShutdownWorkRequestV0,
) (orquestaservershutdown.ActiveShutdownWorkResultV0, error) {
	residues := backend.detectTmuxResiduesV0(ctx)
	if len(residues) == 0 {
		return orquestaservershutdown.ActiveShutdownWorkResultV0{}, nil
	}
	works := make([]orquestaservershutdown.ActiveShutdownWorkV0, 0, len(residues))
	evidenceRefs := append([]string(nil), request.EvidenceRefs...)
	for _, residue := range residues {
		if !residue.Active {
			continue
		}
		workRef := strings.TrimSpace(residue.WorkRef)
		if workRef == "" {
			workRef = strings.TrimSpace(backend.SessionName)
		}
		if workRef == "" {
			workRef = "codex-goal-app-server-tmux"
		}
		evidenceRefs = append(evidenceRefs, residue.EvidenceRefs...)
		works = append(works, orquestaservershutdown.ActiveShutdownWorkV0{
			Kind:            "goal_backend",
			WorkRef:         workRef,
			ExternalWorkRef: "codex-goal-app-server-tmux",
			Status:          orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0,
			EvidenceRefs:    residue.EvidenceRefs,
		})
	}
	evidenceRefs = compactStringsV0(evidenceRefs)
	return orquestaservershutdown.ActiveShutdownWorkResultV0{
		ActiveWorks:  works,
		EvidenceRefs: evidenceRefs,
	}, nil
}

type codexAppServerTmuxResidueV0 struct {
	Active       bool
	WorkRef      string
	EvidenceRefs []string
}

func (backend serverCodexAppServerTmuxBackendV0) detectTmuxResiduesV0(ctx context.Context) []codexAppServerTmuxResidueV0 {
	residues := make([]codexAppServerTmuxResidueV0, 0, 2)
	seen := map[string]struct{}{}
	configured := backend.detectTmuxResidueV0(ctx)
	if configured.Active {
		configured.WorkRef = strings.TrimSpace(backend.SessionName)
		residues = append(residues, configured)
		if configured.WorkRef != "" {
			seen[configured.WorkRef] = struct{}{}
		}
	}
	for _, residue := range backend.detectTmuxOwnerMarkerResiduesV0(ctx) {
		if !residue.Active {
			continue
		}
		key := strings.TrimSpace(residue.WorkRef)
		if key != "" {
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
		}
		residues = append(residues, residue)
	}
	if len(residues) == 0 {
		return nil
	}
	return residues
}

func (backend serverCodexAppServerTmuxBackendV0) detectTmuxResidueV0(ctx context.Context) codexAppServerTmuxResidueV0 {
	if !backend.tmuxResidueConfigLooksOwnV0() {
		return codexAppServerTmuxResidueV0{}
	}
	observed, _ := backend.recolectarObservacionBackendV0(ctx, solicitudObservacionBackendV0{
		Actual:         BackendInexistenteV0,
		CheckProcesses: true,
	})
	residue := codexAppServerTmuxResidueV0{
		Active:       codexAppServerTmuxObservationHasLiveResidueV0(observed.Observacion),
		EvidenceRefs: observed.EvidenceRefs,
	}
	if residue.Active {
		residue.EvidenceRefs = compactStringsV0(append(
			[]string{"evidence-ref-codex-app-server-tmux-residue"},
			residue.EvidenceRefs...,
		))
	}
	return residue
}

func codexAppServerTmuxObservationHasLiveResidueV0(
	obs ObservacionBackendV0,
) bool {
	return obs.OwnerMarkerObserved ||
		obs.OwnerMarkerValid ||
		obs.SessionObserved ||
		obs.SocketObserved ||
		obs.PanePIDLive ||
		obs.ProcessBySocketObserved ||
		obs.ProcessByRuntimeObserved
}

func (backend serverCodexAppServerTmuxBackendV0) detectTmuxOwnerMarkerResiduesV0(
	ctx context.Context,
) []codexAppServerTmuxResidueV0 {
	paths := backend.tmuxOwnerMarkerScanPathsV0()
	if len(paths) == 0 {
		return nil
	}
	residues := make([]codexAppServerTmuxResidueV0, 0, len(paths))
	seen := map[string]struct{}{}
	currentMarker := filepath.Clean(backend.tmuxOwnerMarkerPathV0())
	for _, path := range paths {
		path = filepath.Clean(strings.TrimSpace(path))
		if path == "" || path == currentMarker {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		marker, ok := readCodexAppServerTmuxOwnerMarkerPathV0(path)
		if !ok {
			continue
		}
		sessionName := strings.TrimSpace(marker.SessionName)
		if sessionName == "" {
			continue
		}
		residue := codexAppServerTmuxResidueV0{
			Active:  true,
			WorkRef: sessionName,
			EvidenceRefs: []string{
				"evidence-ref-codex-app-server-tmux-residue",
				"evidence-ref-codex-app-server-tmux-owner-marker",
				"evidence-ref-codex-app-server-tmux-owner-scan",
			},
		}
		if session, process := backend.detectTmuxSessionResidueByNameV0(ctx, sessionName); session {
			residue.EvidenceRefs = append(residue.EvidenceRefs, "evidence-ref-codex-app-server-tmux-session")
			if process {
				residue.EvidenceRefs = append(residue.EvidenceRefs, "evidence-ref-codex-app-server-tmux-process")
			}
		}
		residue.EvidenceRefs = compactStringsV0(residue.EvidenceRefs)
		residues = append(residues, residue)
	}
	if len(residues) == 0 {
		return nil
	}
	return residues
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxOwnerMarkerScanPathsV0() []string {
	paths := []string{}
	if runtimeDir := strings.TrimSpace(backend.RuntimeWorkDir); runtimeDir != "" {
		paths = append(paths, filepath.Join(runtimeDir, codexAppServerTmuxDirV0, codexAppServerTmuxMarkerFileV0))
	}
	if socketPath := strings.TrimSpace(backend.SocketPath); socketPath != "" {
		socketDir := filepath.Dir(socketPath)
		paths = append(paths, filepath.Join(socketDir, codexAppServerTmuxMarkerFileV0))
		if strings.HasPrefix(filepath.Base(socketDir), "oq-gsrv-") {
			for _, dir := range codexAppServerTmuxFallbackDirsV0() {
				paths = append(paths, filepath.Join(dir, codexAppServerTmuxMarkerFileV0))
			}
		}
	}
	return paths
}

func codexAppServerTmuxFallbackDirsV0() []string {
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "oq-gsrv-*"))
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		info, err := os.Lstat(match)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		out = append(out, match)
	}
	return out
}

func (backend serverCodexAppServerTmuxBackendV0) tmuxResidueConfigLooksOwnV0() bool {
	return backend.tmuxOwnerMarkerExistsV0() || backend.tmuxConfiguredOrphanCleanupAllowedV0()
}

func (backend serverCodexAppServerTmuxBackendV0) detectTmuxSessionResidueV0(ctx context.Context) (bool, bool) {
	return backend.detectTmuxSessionResidueByNameV0(ctx, strings.TrimSpace(backend.SessionName))
}

func (backend serverCodexAppServerTmuxBackendV0) detectTmuxSessionResidueByNameV0(
	ctx context.Context,
	sessionName string,
) (bool, bool) {
	sessionName = strings.TrimSpace(sessionName)
	if sessionName == "" {
		return false, false
	}
	tmuxPath, err := codexAppServerTmuxCommandPathV0(backend.PathEnv)
	if err != nil {
		return false, false
	}
	timeout := backend.Timeout
	if timeout <= 0 {
		timeout = codexAppServerTmuxDefaultTimeoutV0
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	probe := backend
	probe.SessionName = sessionName
	observed, err := probe.recolectarObservacionBackendV0(runCtx, solicitudObservacionBackendV0{
		Actual:      BackendPreparandoV0,
		TmuxPath:    tmuxPath,
		RequireTmux: true,
	})
	if err != nil || !observed.Observacion.SessionObserved {
		return false, false
	}
	return true, observed.Observacion.PanePIDLive
}

func (backend serverCodexAppServerTmuxBackendV0) detectCodexAppServerConfiguredSocketProcessV0(ctx context.Context) bool {
	socketPath := strings.TrimSpace(backend.SocketPath)
	if socketPath == "" || !backend.tmuxConfiguredOrphanCleanupAllowedV0() {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return false
	}
	output, err := exec.CommandContext(ctx, "ps", "-eo", "args=").Output()
	if err != nil {
		return false
	}
	return codexAppServerTmuxCommandsContainSocketV0(string(output), socketPath)
}

func codexAppServerTmuxCommandsContainSocketV0(raw string, socketPath string) bool {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return false
	}
	for _, line := range strings.Split(raw, "\n") {
		if codexAppServerTmuxCommandMatchesSocketV0(line, socketPath) {
			return true
		}
	}
	return false
}

func codexAppServerTmuxCommandMatchesSocketV0(command string, socketPath string) bool {
	command = strings.TrimSpace(command)
	socketPath = strings.TrimSpace(socketPath)
	if command == "" || socketPath == "" {
		return false
	}
	if !strings.Contains(command, "app-server") || !strings.Contains(command, "--listen") {
		return false
	}
	if !strings.Contains(command, "codex") {
		return false
	}
	return strings.Contains(command, "unix://"+socketPath) || strings.Contains(command, socketPath)
}

func (backend serverCodexAppServerTmuxBackendV0) detectCodexAppServerRuntimeWorkdirProcessV0(ctx context.Context) bool {
	runtimeDir := strings.TrimSpace(backend.RuntimeWorkDir)
	if runtimeDir == "" || !backend.tmuxConfiguredOrphanCleanupAllowedV0() {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return false
	}
	output, err := exec.CommandContext(ctx, "ps", "-eo", "args=").Output()
	if err != nil {
		return false
	}
	return codexAppServerTmuxCommandsContainRuntimeWorkdirSocketV0(string(output), runtimeDir)
}

func codexAppServerTmuxCommandsContainRuntimeWorkdirSocketV0(raw string, runtimeDir string) bool {
	runtimeDir = strings.TrimSpace(runtimeDir)
	if runtimeDir == "" {
		return false
	}
	socketRoot := filepath.ToSlash(filepath.Join(filepath.Clean(runtimeDir), codexAppServerTmuxDirV0))
	for _, line := range strings.Split(raw, "\n") {
		if codexAppServerTmuxCommandMatchesRuntimeWorkdirSocketV0(line, socketRoot) {
			return true
		}
	}
	return false
}

func codexAppServerTmuxCommandMatchesRuntimeWorkdirSocketV0(command string, socketRoot string) bool {
	command = filepath.ToSlash(strings.TrimSpace(command))
	socketRoot = filepath.ToSlash(strings.TrimSpace(socketRoot))
	if command == "" || socketRoot == "" {
		return false
	}
	if !strings.Contains(command, "app-server") || !strings.Contains(command, "--listen") || !strings.Contains(command, "codex") {
		return false
	}
	return strings.Contains(command, "unix://"+socketRoot+"/")
}

func (backend serverCodexAppServerTmuxBackendV0) detectCodexAppServerRuntimeOwnedProcessV0(ctx context.Context) bool {
	runtimeDir := strings.TrimSpace(backend.RuntimeWorkDir)
	if runtimeDir == "" || !backend.tmuxConfiguredOrphanCleanupAllowedV0() {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return false
	}
	return codexAppServerTmuxRuntimeOwnedProcessExistsV0(ctx, filepath.Join(filepath.Clean(runtimeDir), codexAppServerTmuxDirV0))
}

func codexAppServerTmuxRuntimeOwnedProcessExistsV0(ctx context.Context, runtimeGoalDir string) bool {
	return len(codexAppServerTmuxRuntimeOwnedProcessPIDsV0(ctx, runtimeGoalDir)) > 0
}

func codexAppServerTmuxRuntimeOwnedProcessPIDsV0(ctx context.Context, runtimeGoalDir string) []int {
	runtimeGoalDir = strings.TrimSpace(runtimeGoalDir)
	if runtimeGoalDir == "" {
		return nil
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	out := []int{}
	currentPID := os.Getpid()
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil
		}
		if !entry.IsDir() || !codexAppServerTmuxProcNameLooksPIDV0(entry.Name()) {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 || pid == currentPID {
			continue
		}
		procDir := filepath.Join("/proc", entry.Name())
		command, ok := codexAppServerTmuxProcCommandV0(procDir)
		if !ok || !codexAppServerTmuxCommandLooksAppServerV0(command) {
			continue
		}
		if codexAppServerTmuxProcRuntimeOwnedV0(procDir, runtimeGoalDir) {
			out = append(out, pid)
		}
	}
	return out
}

func codexAppServerTmuxProcNameLooksPIDV0(name string) bool {
	if name == "" {
		return false
	}
	for _, char := range name {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func codexAppServerTmuxProcCommandV0(procDir string) (string, bool) {
	raw, err := os.ReadFile(filepath.Join(procDir, "cmdline"))
	if err != nil || len(raw) == 0 {
		return "", false
	}
	return strings.TrimSpace(strings.ReplaceAll(string(raw), "\x00", " ")), true
}

func codexAppServerTmuxCommandLooksAppServerV0(command string) bool {
	command = strings.TrimSpace(command)
	if command == "" {
		return false
	}
	return strings.Contains(command, "codex") && strings.Contains(command, "app-server")
}

func codexAppServerTmuxProcRuntimeOwnedV0(procDir string, runtimeGoalDir string) bool {
	if cwd, err := os.Readlink(filepath.Join(procDir, "cwd")); err == nil &&
		codexAppServerTmuxPathUnderRuntimeGoalDirV0(cwd, runtimeGoalDir) {
		return true
	}
	raw, err := os.ReadFile(filepath.Join(procDir, "environ"))
	if err != nil || len(raw) == 0 {
		return false
	}
	for _, entry := range strings.Split(string(raw), "\x00") {
		value, ok := strings.CutPrefix(entry, "CODEX_HOME=")
		if ok && codexAppServerTmuxPathUnderRuntimeGoalDirV0(value, runtimeGoalDir) {
			return true
		}
	}
	return false
}

func codexAppServerTmuxPathUnderRuntimeGoalDirV0(path string, runtimeGoalDir string) bool {
	path = strings.TrimSpace(path)
	runtimeGoalDir = strings.TrimSpace(runtimeGoalDir)
	if path == "" || runtimeGoalDir == "" {
		return false
	}
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(runtimeGoalDir)
	if cleanPath == cleanRoot {
		return true
	}
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil || rel == "." || rel == "" {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != ".."
}
