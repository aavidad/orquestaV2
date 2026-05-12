package orquestastatefile

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

const (
	runsDirV0           = "runs"
	eventsDirV0         = "events"
	workflowTasksDirV0  = "workflow_tasks"
	agentProcessesDirV0 = "agent_processes"
)

func ensureStoreDirsV0(rootDir string) error {
	for _, dir := range []string{runsDirV0, eventsDirV0, workflowTasksDirV0, agentProcessesDirV0} {
		if err := os.MkdirAll(filepath.Join(rootDir, dir), 0o700); err != nil {
			return err
		}
	}
	return nil
}

func (store *StoreV0) runPathV0(runRef string) string {
	return filepath.Join(store.rootDir, runsDirV0, hashRefsV0(runRef)+".json")
}

func (store *StoreV0) eventsPathV0(runRef string) string {
	return filepath.Join(store.rootDir, eventsDirV0, hashRefsV0(runRef)+".json")
}

func (store *StoreV0) workflowTaskPathV0(runRef string, taskRef string) string {
	return filepath.Join(store.rootDir, workflowTasksDirV0, hashRefsV0(runRef), hashRefsV0(taskRef)+".json")
}

func (store *StoreV0) agentProcessPathV0(runRef string, agentRequestRef string) string {
	return filepath.Join(store.rootDir, agentProcessesDirV0, hashRefsV0(runRef), hashRefsV0(agentRequestRef)+".json")
}

func hashRefsV0(refs ...string) string {
	hash := sha256.New()
	for _, ref := range refs {
		hash.Write([]byte{0})
		hash.Write([]byte(normalizeRefV0(ref)))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func normalizeRefV0(value string) string {
	return strings.TrimSpace(value)
}
