package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

func (check serverStartupCheckV0) compactStartupStateFilesV0(
	command orquestaserver.StartupCheckCommandV0,
) (startupStateCompactionV0, error) {
	stateDir := strings.TrimSpace(firstNonEmptyStartupStringV0(command.StateDir, check.ServerConfig.StateDir))
	runtimeDir := strings.TrimSpace(firstNonEmptyStartupStringV0(command.RuntimeWorkDir, check.ServerConfig.RuntimeWorkDir))
	if stateDir == "" {
		return startupStateCompactionV0{}, nil
	}
	queuePath := filepath.Join(stateDir, "run-state", "queue_v0.json")
	controlPath := filepath.Join(stateDir, "run-state", "control_v0.json")
	queue, queueBytes, err := readStartupQueueSnapshotV0(queuePath)
	if err != nil {
		return startupStateCompactionV0{}, err
	}
	control, controlBytes, err := readStartupControlSnapshotV0(controlPath)
	if err != nil {
		return startupStateCompactionV0{}, err
	}

	removedQueue, keptQueue := splitStartupQueueRecordsV0(queue.Records, stateDir, runtimeDir)
	removedControl, keptControl := splitStartupControlRecordsV0(control.Records)
	runRefs := startupRemovedRunRefsV0(removedQueue, removedControl)
	if len(removedQueue) == 0 && len(removedControl) == 0 && len(runRefs) == 0 {
		return startupStateCompactionV0{}, nil
	}

	revisionDir := filepath.Join(stateDir, "revision", "purga_startup_"+startupRevisionTimestampV0(command.OccurredAt))
	if err := os.MkdirAll(revisionDir, 0o755); err != nil {
		return startupStateCompactionV0{}, err
	}
	if len(queueBytes) > 0 {
		if err := os.WriteFile(filepath.Join(revisionDir, "queue_v0.before.json"), queueBytes, 0o644); err != nil {
			return startupStateCompactionV0{}, err
		}
	}
	if len(controlBytes) > 0 {
		if err := os.WriteFile(filepath.Join(revisionDir, "control_v0.before.json"), controlBytes, 0o644); err != nil {
			return startupStateCompactionV0{}, err
		}
	}

	queue.Records = keptQueue
	control.Records = keptControl
	if queueBytes != nil {
		if err := writeStartupJSONFileV0(queuePath, queue); err != nil {
			return startupStateCompactionV0{}, err
		}
	}
	if controlBytes != nil {
		if err := writeStartupJSONFileV0(controlPath, control); err != nil {
			return startupStateCompactionV0{}, err
		}
	}
	if err := writeStartupJSONFileV0(filepath.Join(revisionDir, "queue_removed.json"), startupQueueSnapshotV0{
		SchemaVersion: queue.SchemaVersion,
		Records:       removedQueue,
	}); err != nil {
		return startupStateCompactionV0{}, err
	}
	if err := writeStartupJSONFileV0(filepath.Join(revisionDir, "control_removed.json"), startupControlSnapshotV0{
		SchemaVersion: control.SchemaVersion,
		Records:       removedControl,
	}); err != nil {
		return startupStateCompactionV0{}, err
	}

	archivedRuntime, err := archiveStartupRuntimeDirsV0(runtimeDir, revisionDir, runRefs)
	if err != nil {
		return startupStateCompactionV0{}, err
	}
	compaction := startupStateCompactionV0{
		RevisionDir:      revisionDir,
		QueueRemoved:     len(removedQueue),
		QueueKept:        len(keptQueue),
		ControlRemoved:   len(removedControl),
		ControlKept:      len(keptControl),
		RuntimeArchived:  archivedRuntime,
		CompactionNeeded: len(removedQueue) > 0 || len(removedControl) > 0 || archivedRuntime > 0,
	}
	if err := writeStartupJSONFileV0(filepath.Join(revisionDir, "manifest.json"), compaction); err != nil {
		return startupStateCompactionV0{}, err
	}
	if compaction.CompactionNeeded {
		if err := check.reloadStartupStateStoresAfterCompactionV0(); err != nil {
			return startupStateCompactionV0{}, err
		}
	}
	return compaction, nil
}

func (check serverStartupCheckV0) reloadStartupStateStoresAfterCompactionV0() error {
	for _, store := range []any{check.Stack.Stores.RunQueue, check.Stack.Stores.RunControl} {
		reloader, ok := store.(startupStateStoreReloaderV0)
		if !ok || reloader == nil {
			continue
		}
		if err := reloader.ReloadFromDiskV0(); err != nil {
			return err
		}
	}
	return nil
}

func readStartupQueueSnapshotV0(path string) (startupQueueSnapshotV0, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return startupQueueSnapshotV0{}, nil, nil
		}
		return startupQueueSnapshotV0{}, nil, err
	}
	var snapshot startupQueueSnapshotV0
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return startupQueueSnapshotV0{}, nil, err
	}
	return snapshot, data, nil
}

func readStartupControlSnapshotV0(path string) (startupControlSnapshotV0, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return startupControlSnapshotV0{}, nil, nil
		}
		return startupControlSnapshotV0{}, nil, err
	}
	var snapshot startupControlSnapshotV0
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return startupControlSnapshotV0{}, nil, err
	}
	return snapshot, data, nil
}

func splitStartupQueueRecordsV0(
	records []startupQueueRecordV0,
	stateDir string,
	runtimeDir string,
) ([]startupQueueRecordV0, []startupQueueRecordV0) {
	removed := make([]startupQueueRecordV0, 0, len(records))
	kept := make([]startupQueueRecordV0, 0, len(records))
	for _, record := range records {
		reason := startupQueueRecordPurgeReasonV0(record, stateDir, runtimeDir)
		if reason == "" {
			kept = append(kept, record)
			continue
		}
		record.PurgeReason = reason
		removed = append(removed, record)
	}
	return removed, kept
}

func startupQueueRecordPurgeReasonV0(record startupQueueRecordV0, stateDir string, runtimeDir string) string {
	status := strings.TrimSpace(record.Candidate.Status)
	if !orquestarunqueue.IsExecutableRunStatusV0(status) {
		return "queue_status_no_ejecutable"
	}
	runRef := startupQueueRecordRunRefV0(record)
	if runRef != "" && startupRunNeedsFreshAutoprogrammingAttemptV0(stateDir, runtimeDir, runRef) {
		return "ready_run_autoprogramming_obsoleto"
	}
	if runRef != "" && startupRunHasCompletedAckV0(runtimeDir, runRef) {
		return "ready_huerfano_con_ack_completed"
	}
	return ""
}

func startupRunNeedsFreshAutoprogrammingAttemptV0(stateDir string, runtimeDir string, runRef string) bool {
	stateDir = strings.TrimSpace(stateDir)
	runRef = strings.TrimSpace(runRef)
	if stateDir == "" || runRef == "" {
		return false
	}
	rootDir := filepath.Join(stateDir, "orchestration-state")
	if _, err := os.Stat(rootDir); err != nil {
		return false
	}
	store, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{
		RootDir: rootDir,
	})
	if err != nil {
		return false
	}
	run, err := store.LoadRunV0(context.Background(), runRef)
	if err != nil {
		return false
	}
	return orquestaappcodexstack.AutoprogrammingRunNeedsFreshAttemptWithRuntimeV0(run, runtimeDir)
}

func splitStartupControlRecordsV0(
	records []startupControlRecordV0,
) ([]startupControlRecordV0, []startupControlRecordV0) {
	removed := make([]startupControlRecordV0, 0, len(records))
	kept := make([]startupControlRecordV0, 0, len(records))
	for _, record := range records {
		if orquestaruncontrol.IsTerminalRunControlStatusV0(record.State.Status) {
			record.PurgeReason = "control_terminal"
			removed = append(removed, record)
			continue
		}
		kept = append(kept, record)
	}
	return removed, kept
}

func startupRemovedRunRefsV0(
	queue []startupQueueRecordV0,
	control []startupControlRecordV0,
) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(queue)+len(control))
	for _, record := range queue {
		out = appendStartupRunRefV0(out, seen, startupQueueRecordRunRefV0(record))
	}
	for _, record := range control {
		out = appendStartupRunRefV0(out, seen, firstNonEmptyStartupStringV0(record.State.RunRef, record.RunRef))
	}
	return out
}

func appendStartupRunRefV0(out []string, seen map[string]struct{}, runRef string) []string {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return out
	}
	if _, ok := seen[runRef]; ok {
		return out
	}
	seen[runRef] = struct{}{}
	return append(out, runRef)
}

func startupQueueRecordRunRefV0(record startupQueueRecordV0) string {
	return firstNonEmptyStartupStringV0(record.Candidate.RunRef, record.RunRef)
}

func startupRunHasCompletedAckV0(runtimeDir string, runRef string) bool {
	return startupRunHasStrictCompletedACKV0(runtimeDir, runRef)
}

func writeStartupJSONFileV0(path string, value any) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
