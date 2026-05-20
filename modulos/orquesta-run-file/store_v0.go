package orquestarunfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

const (
	runFileControlNameV0   = "control_v0.json"
	runFileQueueNameV0     = "queue_v0.json"
	runFileAppChangeNameV0 = "app_change_v0.json"
)

type RunFileStoreV0 struct {
	mu               sync.Mutex
	controlPath      string
	queuePath        string
	appChangePath    string
	control          map[string]orquestaruncontrol.RunControlStateV0
	queueRuns        map[string]runFileQueueEntryV0
	appChangeRecords []orquestaappchange.AppChangeRecordV0
}

type runFileQueueEntryV0 struct {
	queueRef  string
	candidate orquestarunqueue.RunSchedulingCandidateV0
}

var _ orquestaruncontrol.RunControlPortV0 = (*RunFileStoreV0)(nil)
var _ orquestarunqueue.RunQueuePortV0 = (*RunFileStoreV0)(nil)
var _ orquestaappchange.AppChangeRecordStorePortV0 = (*RunFileStoreV0)(nil)

func NewRunFileStoreV0(dir string) (*RunFileStoreV0, error) {
	dir, err := normalizeRunFileDirV0(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("orquesta_run_file: dir_unavailable")
	}
	store := &RunFileStoreV0{
		controlPath:   filepath.Join(dir, runFileControlNameV0),
		queuePath:     filepath.Join(dir, runFileQueueNameV0),
		appChangePath: filepath.Join(dir, runFileAppChangeNameV0),
	}
	if err := store.loadV0(); err != nil {
		return nil, err
	}
	return store, nil
}

func normalizeRunFileDirV0(dir string) (string, error) {
	dir = filepath.Clean(strings.TrimSpace(dir))
	if dir == "." || dir == "" || !filepath.IsAbs(dir) {
		return "", fmt.Errorf("orquesta_run_file: dir_invalid")
	}
	return dir, nil
}

func (store *RunFileStoreV0) loadV0() error {
	control, err := loadRunFileControlV0(store.controlPath)
	if err != nil {
		return err
	}
	queueRuns, err := loadRunFileQueueV0(store.queuePath)
	if err != nil {
		return err
	}
	appChangeRecords, err := loadRunFileAppChangeV0(store.appChangePath)
	if err != nil {
		return err
	}
	store.control = control
	store.queueRuns = queueRuns
	store.appChangeRecords = appChangeRecords
	return nil
}

func (store *RunFileStoreV0) ReloadFromDiskV0() error {
	control, err := loadRunFileControlV0(store.controlPath)
	if err != nil {
		return err
	}
	queueRuns, err := loadRunFileQueueV0(store.queuePath)
	if err != nil {
		return err
	}
	appChangeRecords, err := loadRunFileAppChangeV0(store.appChangePath)
	if err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.control = control
	store.queueRuns = queueRuns
	store.appChangeRecords = appChangeRecords
	return nil
}

func (store *RunFileStoreV0) ensureLockedV0() {
	if store.control == nil {
		store.control = map[string]orquestaruncontrol.RunControlStateV0{}
	}
	if store.queueRuns == nil {
		store.queueRuns = map[string]runFileQueueEntryV0{}
	}
	if store.appChangeRecords == nil {
		store.appChangeRecords = []orquestaappchange.AppChangeRecordV0{}
	}
}
