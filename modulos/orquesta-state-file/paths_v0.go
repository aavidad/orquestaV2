package orquestastatefile

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

const (
	runsDirV0                         = "runs"
	eventsDirV0                       = "events"
	eventIndexesDirV0                 = "event_indexes"
	eventRecordsDirV0                 = "event_records"
	workflowTasksDirV0                = "workflow_tasks"
	workflowTaskParentIndexesDirV0    = "workflow_task_parent_indexes"
	workflowWaitsDirV0                = "workflow_waits"
	requiredTestEvidenceDirV0         = "required_test_evidence"
	goalRequiredTestAttestationsDirV0 = "goal_required_test_attestations"
	operationalPlanStatesDirV0        = "operational_director_plan_states"
	appDirectorGoalStatesDirV0        = "app_director_goal_states"
	appDirectorGoalMarkersDirV0       = "app_director_goal_markers"
	agentProcessesDirV0               = "agent_processes"
	autonomyProgramsDirV0             = "autonomy_programs"
	worktreeSnapshotsDirV0            = "worktree_snapshots"
)

func ensureStoreDirsV0(rootDir string) error {
	for _, dir := range []string{runsDirV0, eventsDirV0, eventIndexesDirV0, eventRecordsDirV0, workflowTasksDirV0, workflowTaskParentIndexesDirV0, workflowWaitsDirV0, requiredTestEvidenceDirV0, goalRequiredTestAttestationsDirV0, operationalPlanStatesDirV0, appDirectorGoalStatesDirV0, appDirectorGoalMarkersDirV0, agentProcessesDirV0, autonomyProgramsDirV0, worktreeSnapshotsDirV0} {
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

func (store *StoreV0) eventIndexPathV0(runRef string) string {
	return filepath.Join(store.rootDir, eventIndexesDirV0, hashRefsV0(runRef)+".json")
}

func (store *StoreV0) eventRecordPathV0(runRef string, recordRef string) string {
	return filepath.Join(store.rootDir, eventRecordsDirV0, hashRefsV0(runRef), hashRefsV0(recordRef)+".json")
}

func (store *StoreV0) workflowTaskPathV0(runRef string, taskRef string) string {
	return filepath.Join(store.rootDir, workflowTasksDirV0, hashRefsV0(runRef), hashRefsV0(taskRef)+".json")
}

func (store *StoreV0) workflowTaskParentIndexPathV0(runRef string) string {
	return filepath.Join(store.rootDir, workflowTaskParentIndexesDirV0, hashRefsV0(runRef)+".json")
}

func (store *StoreV0) workflowWaitPathV0(runRef string, waitRef string) string {
	return filepath.Join(store.rootDir, workflowWaitsDirV0, hashRefsV0(runRef), hashRefsV0(waitRef)+".json")
}

func (store *StoreV0) requiredTestEvidencePathV0(runRef string, evidenceRef string) string {
	return filepath.Join(store.rootDir, requiredTestEvidenceDirV0, hashRefsV0(runRef), hashRefsV0(evidenceRef)+".json")
}

func (store *StoreV0) operationalDirectorPlanStatePathV0(runRef string, planRef string) string {
	return filepath.Join(store.rootDir, operationalPlanStatesDirV0, hashRefsV0(runRef), hashRefsV0(planRef)+".json")
}

func (store *StoreV0) appDirectorGoalStatePathV0(runRef string) string {
	return filepath.Join(store.rootDir, appDirectorGoalStatesDirV0, hashRefsV0(runRef)+".json")
}

func (store *StoreV0) appDirectorGoalMarkerPathV0(runRef string) string {
	return filepath.Join(store.rootDir, appDirectorGoalMarkersDirV0, hashRefsV0(runRef)+".json")
}

func (store *StoreV0) agentProcessPathV0(runRef string, agentRequestRef string) string {
	return filepath.Join(store.rootDir, agentProcessesDirV0, hashRefsV0(runRef), hashRefsV0(agentRequestRef)+".json")
}

func (store *StoreV0) autonomyProgramPathV0(projectRef string, rootRef string, programRef string) string {
	return filepath.Join(store.rootDir, autonomyProgramsDirV0, hashRefsV0(projectRef), hashRefsV0(rootRef), hashRefsV0(programRef)+".json")
}

func (store *StoreV0) autonomyProgramLockPathV0(projectRef string, rootRef string, programRef string) string {
	return store.autonomyProgramPathV0(projectRef, rootRef, programRef) + ".lock"
}

func (store *StoreV0) worktreeSnapshotPathV0(snapshotRef string) string {
	return filepath.Join(store.rootDir, worktreeSnapshotsDirV0, hashRefsV0(snapshotRef)+".json")
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
