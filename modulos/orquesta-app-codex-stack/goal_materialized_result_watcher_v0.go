package orquestaappcodexstack

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	defaultGoalMaterializedResultWatcherIntervalV0   = 250 * time.Millisecond
	defaultGoalMaterializedResultWatcherMaxGoalsV0   = 70
	defaultGoalMaterializedResultWatcherMaxDirsV0    = 256
	goalMaterializedResultWatcherCauseDetectedV0     = "goal_materialized_result_detected"
	goalMaterializedResultWatcherEvidenceDetectedV0  = "evidence-ref-goal-materialized-result-watcher-detected"
	goalMaterializedResultWatcherEvidenceNoActiveV0  = "evidence-ref-goal-materialized-result-watcher-no-active-goals"
	goalMaterializedResultWatcherEvidenceWakeupV0    = "evidence-ref-goal-materialized-result-watcher-wakeup"
	goalMaterializedResultWatcherDirSkipNameGitV0    = ".git"
	goalMaterializedResultWatcherDirSkipNameVendorV0 = "vendor"
	goalMaterializedResultWatcherDirSkipNameNodeV0   = "node_modules"
)

type GoalMaterializedResultWatcherConfigV0 struct {
	ProjectWorkDir      string
	ProjectRootResolver GoalMaterializedResultProjectRootResolverPortV0
	StateStore          orquestagoal.GoalWorkStateStorePortV0
	Interval            time.Duration
	MaxGoals            int
	MaxDirsPerGoal      int
	Wakeup              func(context.Context, GoalMaterializedResultWakeupV0) bool
	BeforeWakeup        func(context.Context, GoalMaterializedResultWakeupV0)
}

type GoalMaterializedResultProjectRootResolverPortV0 interface {
	ResolveGoalMaterializedResultProjectRootV0(context.Context, orquestagoal.GoalWorkStateV0) (string, error)
}

type GoalMaterializedResultWakeupV0 struct {
	RunRef          string
	GoalRef         string
	ExternalGoalRef string
	Cause           string
	EvidenceRefs    []string
}

type GoalMaterializedResultWatcherStatsV0 struct {
	ActiveRefreshes int
	DirectoryPolls  int
	ResultChecks    int
	Wakeups         int
	NoActiveSleeps  int
}

type GoalMaterializedResultWatcherV0 struct {
	config  GoalMaterializedResultWatcherConfigV0
	signals chan struct{}
	mu      sync.Mutex
	stats   GoalMaterializedResultWatcherStatsV0
	emitted map[string]string
}

type goalMaterializedResultWatchV0 struct {
	State       orquestagoal.GoalWorkStateV0
	ProjectRoot string
	Dirs        map[string]goalMaterializedResultDirFingerprintV0
}

type goalMaterializedResultDirFingerprintV0 struct {
	ModTime     int64
	Size        int64
	ResultFiles map[string]goalMaterializedResultFileFingerprintV0
}

type goalMaterializedResultFileFingerprintV0 struct {
	ModTime int64
	Size    int64
}

func NewGoalMaterializedResultWatcherV0(
	config GoalMaterializedResultWatcherConfigV0,
) *GoalMaterializedResultWatcherV0 {
	config = normalizeGoalMaterializedResultWatcherConfigV0(config)
	return &GoalMaterializedResultWatcherV0{
		config:  config,
		signals: make(chan struct{}, 1),
		emitted: map[string]string{},
	}
}

func (watcher *GoalMaterializedResultWatcherV0) NotifyActiveGoalsChangedV0() bool {
	if watcher == nil || watcher.signals == nil {
		return false
	}
	select {
	case watcher.signals <- struct{}{}:
		return true
	default:
		return false
	}
}

func (watcher *GoalMaterializedResultWatcherV0) RunBackgroundV0(ctx context.Context) {
	if watcher == nil || !goalMaterializedResultWatcherRunnableV0(watcher.config) {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	watcher.runV0(ctx)
}

func (watcher *GoalMaterializedResultWatcherV0) StatsV0() GoalMaterializedResultWatcherStatsV0 {
	if watcher == nil {
		return GoalMaterializedResultWatcherStatsV0{}
	}
	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	return watcher.stats
}

func (watcher *GoalMaterializedResultWatcherV0) runV0(ctx context.Context) {
	watches := watcher.refreshActiveWatchesV0(ctx)
	for {
		if len(watches) == 0 {
			watcher.noteNoActiveSleepV0()
			select {
			case <-ctx.Done():
				return
			case <-watcher.signals:
				watches = watcher.refreshActiveWatchesV0(ctx)
				continue
			}
		}
		ticker := time.NewTicker(watcher.config.Interval)
		for len(watches) > 0 {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-watcher.signals:
				watches = watcher.refreshActiveWatchesV0(ctx)
			case <-ticker.C:
				watches = watcher.pollActiveWatchesV0(ctx, watches)
			}
		}
		ticker.Stop()
	}
}

func (watcher *GoalMaterializedResultWatcherV0) refreshActiveWatchesV0(
	ctx context.Context,
) []goalMaterializedResultWatchV0 {
	watcher.noteActiveRefreshV0()
	states := watcher.activeGoalStatesV0(ctx)
	if len(states) == 0 {
		return nil
	}
	watches := make([]goalMaterializedResultWatchV0, 0, len(states))
	for _, state := range states {
		projectRoot, ok := watcher.projectRootForStateV0(ctx, state)
		if !ok {
			continue
		}
		if watcher.detectMaterializedResultV0(ctx, projectRoot, state) {
			continue
		}
		dirs := goalMaterializedResultWatchDirsForStateV0(
			projectRoot,
			state,
			watcher.config.MaxDirsPerGoal,
		)
		if len(dirs) == 0 {
			continue
		}
		watches = append(watches, goalMaterializedResultWatchV0{
			State:       state,
			ProjectRoot: projectRoot,
			Dirs:        dirs,
		})
	}
	return watches
}

func (watcher *GoalMaterializedResultWatcherV0) pollActiveWatchesV0(
	ctx context.Context,
	watches []goalMaterializedResultWatchV0,
) []goalMaterializedResultWatchV0 {
	watcher.noteDirectoryPollV0()
	states := watcher.activeGoalStatesV0(ctx)
	if len(states) == 0 {
		return nil
	}
	stateByRun := map[string]orquestagoal.GoalWorkStateV0{}
	for _, state := range states {
		stateByRun[strings.TrimSpace(state.RunRef)] = state
	}
	next := make([]goalMaterializedResultWatchV0, 0, len(states))
	for _, watch := range watches {
		runRef := strings.TrimSpace(watch.State.RunRef)
		state, ok := stateByRun[runRef]
		if !ok {
			continue
		}
		projectRoot, ok := watcher.projectRootForStateV0(ctx, state)
		if !ok || projectRoot != watch.ProjectRoot {
			continue
		}
		current := goalMaterializedResultRefreshKnownDirsV0(watch.Dirs)
		if len(current) == 0 {
			current = goalMaterializedResultWatchDirsForStateV0(
				projectRoot,
				state,
				watcher.config.MaxDirsPerGoal,
			)
		}
		if goalMaterializedResultDirsChangedV0(watch.Dirs, current) {
			current = goalMaterializedResultWatchDirsForStateV0(
				projectRoot,
				state,
				watcher.config.MaxDirsPerGoal,
			)
			_ = watcher.detectMaterializedResultV0(ctx, projectRoot, state)
		}
		if len(current) == 0 {
			continue
		}
		next = append(next, goalMaterializedResultWatchV0{
			State:       state,
			ProjectRoot: projectRoot,
			Dirs:        current,
		})
	}
	seen := map[string]bool{}
	for _, watch := range next {
		seen[strings.TrimSpace(watch.State.RunRef)] = true
	}
	for _, state := range states {
		runRef := strings.TrimSpace(state.RunRef)
		if seen[runRef] {
			continue
		}
		projectRoot, ok := watcher.projectRootForStateV0(ctx, state)
		if !ok {
			continue
		}
		if watcher.detectMaterializedResultV0(ctx, projectRoot, state) {
			continue
		}
		dirs := goalMaterializedResultWatchDirsForStateV0(
			projectRoot,
			state,
			watcher.config.MaxDirsPerGoal,
		)
		if len(dirs) == 0 {
			continue
		}
		next = append(next, goalMaterializedResultWatchV0{
			State:       state,
			ProjectRoot: projectRoot,
			Dirs:        dirs,
		})
	}
	return next
}

func (watcher *GoalMaterializedResultWatcherV0) projectRootForStateV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
) (string, bool) {
	root := watcher.config.ProjectWorkDir
	if watcher.config.ProjectRootResolver != nil {
		resolved, err := watcher.config.ProjectRootResolver.ResolveGoalMaterializedResultProjectRootV0(ctx, state)
		if err != nil {
			return "", false
		}
		root = resolved
	}
	return goalMaterializedResultWatcherProjectRootV0(root)
}

func (watcher *GoalMaterializedResultWatcherV0) activeGoalStatesV0(
	ctx context.Context,
) []orquestagoal.GoalWorkStateV0 {
	lister, ok := watcher.config.StateStore.(orquestagoal.GoalWorkStateListPortV0)
	if !ok || lister == nil {
		return nil
	}
	states, err := lister.ListGoalWorkStatesV0(ctx, orquestagoal.GoalWorkStateListRequestV0{
		ActiveOnly: true,
		MaxItems:   watcher.config.MaxGoals,
	})
	if err != nil {
		return nil
	}
	out := make([]orquestagoal.GoalWorkStateV0, 0, len(states))
	for _, state := range states {
		if !orquestagoal.GoalWorkStatePendingObservationV0(state) {
			continue
		}
		out = append(out, state)
	}
	return out
}

func (watcher *GoalMaterializedResultWatcherV0) detectMaterializedResultV0(
	ctx context.Context,
	projectRoot string,
	state orquestagoal.GoalWorkStateV0,
) bool {
	watcher.noteResultCheckV0()
	source := stackGoalMaterializedRefsSourceV0{
		Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectRoot}},
	}
	result, ok, err := source.LoadTerminalGoalMaterializedResultV0(ctx, state)
	if err != nil || !ok {
		return false
	}
	key := goalMaterializedResultWatcherResultKeyV0(result)
	runRef := strings.TrimSpace(state.RunRef)
	if runRef == "" || key == "" || watcher.resultAlreadyEmittedV0(runRef, key) {
		return true
	}
	wakeup := GoalMaterializedResultWakeupV0{
		RunRef:          runRef,
		GoalRef:         strings.TrimSpace(state.GoalRef),
		ExternalGoalRef: strings.TrimSpace(state.ExternalGoalRef),
		Cause:           goalMaterializedResultWatcherCauseDetectedV0,
		EvidenceRefs: []string{
			goalMaterializedResultWatcherEvidenceDetectedV0,
			goalMaterializedResultWatcherEvidenceWakeupV0,
		},
	}
	if watcher.config.BeforeWakeup != nil {
		watcher.config.BeforeWakeup(ctx, wakeup)
	}
	if watcher.config.Wakeup == nil || watcher.config.Wakeup(ctx, wakeup) {
		watcher.markResultEmittedV0(runRef, key)
		watcher.noteWakeupV0()
	}
	return true
}

func goalMaterializedResultWatchDirsForStateV0(
	projectRoot string,
	state orquestagoal.GoalWorkStateV0,
	maxDirs int,
) map[string]goalMaterializedResultDirFingerprintV0 {
	if maxDirs <= 0 {
		maxDirs = defaultGoalMaterializedResultWatcherMaxDirsV0
	}
	out := map[string]goalMaterializedResultDirFingerprintV0{}
	for _, scope := range state.Spec.WriteSet {
		if len(out) >= maxDirs {
			break
		}
		root, ok := goalMaterializedResultScopeRootV0(projectRoot, scope)
		if !ok {
			continue
		}
		for _, dir := range goalMaterializedResultDirsUnderRootV0(projectRoot, root, maxDirs-len(out)) {
			if len(out) >= maxDirs {
				break
			}
			if fingerprint, ok := goalMaterializedResultInitialDirFingerprintV0(dir); ok {
				out[filepath.Clean(dir)] = fingerprint
			}
		}
	}
	return out
}

func goalMaterializedResultScopeRootV0(
	projectRoot string,
	scope orquestagoal.GoalWriteScopeV0,
) (string, bool) {
	relScope := filepath.Clean(filepath.FromSlash(strings.TrimSpace(scope.Path)))
	if relScope == "." || relScope == "" || filepath.IsAbs(relScope) ||
		strings.HasPrefix(relScope, ".."+string(filepath.Separator)) || relScope == ".." {
		return "", false
	}
	target := filepath.Join(projectRoot, relScope)
	if !pathWithinRootV0(projectRoot, target) {
		return "", false
	}
	info, err := os.Stat(target)
	if err == nil {
		if info.IsDir() {
			return filepath.Clean(target), true
		}
		parent := filepath.Dir(target)
		return parent, pathWithinRootV0(projectRoot, parent)
	}
	parent := filepath.Dir(target)
	for pathWithinRootV0(projectRoot, parent) {
		if info, statErr := os.Stat(parent); statErr == nil && info.IsDir() {
			return filepath.Clean(parent), true
		}
		next := filepath.Dir(parent)
		if next == parent {
			break
		}
		parent = next
	}
	return "", false
}

func goalMaterializedResultDirsUnderRootV0(
	projectRoot string,
	root string,
	limit int,
) []string {
	if limit <= 0 {
		return nil
	}
	root = filepath.Clean(root)
	if !pathWithinRootV0(projectRoot, root) {
		return nil
	}
	out := make([]string, 0, limit)
	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry == nil || !entry.IsDir() {
			return nil
		}
		if !pathWithinRootV0(projectRoot, path) {
			return fs.SkipDir
		}
		name := strings.ToLower(strings.TrimSpace(entry.Name()))
		if path != root && (name == goalMaterializedResultWatcherDirSkipNameGitV0 ||
			name == goalMaterializedResultWatcherDirSkipNameVendorV0 ||
			name == goalMaterializedResultWatcherDirSkipNameNodeV0) {
			return fs.SkipDir
		}
		out = append(out, filepath.Clean(path))
		if len(out) >= limit {
			return errGoalMaterializedRefsScanDoneV0
		}
		return nil
	})
	if walkErr != nil && walkErr != errGoalMaterializedRefsScanDoneV0 {
		return out
	}
	return out
}

func goalMaterializedResultDirsChangedV0(
	previous map[string]goalMaterializedResultDirFingerprintV0,
	current map[string]goalMaterializedResultDirFingerprintV0,
) bool {
	if len(previous) != len(current) {
		return true
	}
	for path, now := range current {
		then, ok := previous[path]
		if !ok || then.ModTime != now.ModTime || then.Size != now.Size ||
			goalMaterializedResultFilesChangedV0(then.ResultFiles, now.ResultFiles) {
			return true
		}
	}
	return false
}

func goalMaterializedResultRefreshKnownDirsV0(
	previous map[string]goalMaterializedResultDirFingerprintV0,
) map[string]goalMaterializedResultDirFingerprintV0 {
	if len(previous) == 0 {
		return nil
	}
	current := make(map[string]goalMaterializedResultDirFingerprintV0, len(previous))
	for path, previousFingerprint := range previous {
		fingerprint, ok := goalMaterializedResultKnownDirFingerprintV0(path, previousFingerprint)
		if !ok {
			continue
		}
		current[filepath.Clean(path)] = fingerprint
	}
	return current
}

func goalMaterializedResultInitialDirFingerprintV0(
	dir string,
) (goalMaterializedResultDirFingerprintV0, bool) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return goalMaterializedResultDirFingerprintV0{}, false
	}
	return goalMaterializedResultDirFingerprintV0{
		ModTime:     info.ModTime().UnixNano(),
		Size:        info.Size(),
		ResultFiles: goalMaterializedResultFilesInDirV0(dir),
	}, true
}

func goalMaterializedResultKnownDirFingerprintV0(
	dir string,
	previous goalMaterializedResultDirFingerprintV0,
) (goalMaterializedResultDirFingerprintV0, bool) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return goalMaterializedResultDirFingerprintV0{}, false
	}
	return goalMaterializedResultDirFingerprintV0{
		ModTime:     info.ModTime().UnixNano(),
		Size:        info.Size(),
		ResultFiles: goalMaterializedResultKnownFilesV0(previous.ResultFiles),
	}, true
}

func goalMaterializedResultFilesInDirV0(
	dir string,
) map[string]goalMaterializedResultFileFingerprintV0 {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := map[string]goalMaterializedResultFileFingerprintV0{}
	for _, entry := range entries {
		if entry == nil || entry.IsDir() || !goalMaterializedFileIsGoalResultV0(entry.Name()) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if fingerprint, ok := goalMaterializedResultFileFingerprintForPathV0(path); ok {
			out[filepath.Clean(path)] = fingerprint
		}
	}
	return out
}

func goalMaterializedResultKnownFilesV0(
	previous map[string]goalMaterializedResultFileFingerprintV0,
) map[string]goalMaterializedResultFileFingerprintV0 {
	if len(previous) == 0 {
		return nil
	}
	out := map[string]goalMaterializedResultFileFingerprintV0{}
	for path := range previous {
		if fingerprint, ok := goalMaterializedResultFileFingerprintForPathV0(path); ok {
			out[filepath.Clean(path)] = fingerprint
		}
	}
	return out
}

func goalMaterializedResultFileFingerprintForPathV0(
	path string,
) (goalMaterializedResultFileFingerprintV0, bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return goalMaterializedResultFileFingerprintV0{}, false
	}
	return goalMaterializedResultFileFingerprintV0{
		ModTime: info.ModTime().UnixNano(),
		Size:    info.Size(),
	}, true
}

func goalMaterializedResultFilesChangedV0(
	previous map[string]goalMaterializedResultFileFingerprintV0,
	current map[string]goalMaterializedResultFileFingerprintV0,
) bool {
	if len(previous) != len(current) {
		return true
	}
	for path, now := range current {
		then, ok := previous[path]
		if !ok || then.ModTime != now.ModTime || then.Size != now.Size {
			return true
		}
	}
	return false
}

func goalMaterializedResultWatcherProjectRootV0(projectWorkDir string) (string, bool) {
	projectRoot := strings.TrimSpace(projectWorkDir)
	if projectRoot == "" {
		return "", false
	}
	abs, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", false
	}
	return filepath.Clean(abs), true
}

func goalMaterializedResultWatcherResultKeyV0(result orquestagoal.GoalWorkResultV0) string {
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	raw, err := json.Marshal(result)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return string(sum[:])
}

func normalizeGoalMaterializedResultWatcherConfigV0(
	config GoalMaterializedResultWatcherConfigV0,
) GoalMaterializedResultWatcherConfigV0 {
	config.ProjectWorkDir = strings.TrimSpace(config.ProjectWorkDir)
	if config.Interval <= 0 {
		config.Interval = defaultGoalMaterializedResultWatcherIntervalV0
	}
	if config.MaxGoals <= 0 {
		config.MaxGoals = defaultGoalMaterializedResultWatcherMaxGoalsV0
	}
	if config.MaxDirsPerGoal <= 0 {
		config.MaxDirsPerGoal = defaultGoalMaterializedResultWatcherMaxDirsV0
	}
	return config
}

func goalMaterializedResultWatcherRunnableV0(
	config GoalMaterializedResultWatcherConfigV0,
) bool {
	if strings.TrimSpace(config.ProjectWorkDir) == "" || config.StateStore == nil {
		return false
	}
	_, ok := config.StateStore.(orquestagoal.GoalWorkStateListPortV0)
	return ok
}

func (watcher *GoalMaterializedResultWatcherV0) resultAlreadyEmittedV0(runRef string, key string) bool {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	return watcher.emitted[runRef] == key
}

func (watcher *GoalMaterializedResultWatcherV0) markResultEmittedV0(runRef string, key string) {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	watcher.emitted[runRef] = key
}

func (watcher *GoalMaterializedResultWatcherV0) noteActiveRefreshV0() {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	watcher.stats.ActiveRefreshes++
}

func (watcher *GoalMaterializedResultWatcherV0) noteDirectoryPollV0() {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	watcher.stats.DirectoryPolls++
}

func (watcher *GoalMaterializedResultWatcherV0) noteResultCheckV0() {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	watcher.stats.ResultChecks++
}

func (watcher *GoalMaterializedResultWatcherV0) noteWakeupV0() {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	watcher.stats.Wakeups++
}

func (watcher *GoalMaterializedResultWatcherV0) noteNoActiveSleepV0() {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()
	watcher.stats.NoActiveSleeps++
}
