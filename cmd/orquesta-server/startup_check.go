package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const (
	startupCleanupModeOffV0        = "off"
	startupCleanupModeDiagnoseV0   = "diagnose"
	startupCleanupModeForcedStopV0 = "forced_stop"
)

type serverStartupCheckV0 struct {
	Stack        orquestaappcodexstack.StackV0
	ServerConfig orquestaserver.ConfigV0
	Mode         string
	QueueLimit   int
}

func startupCheckFromEnvV0(
	stack orquestaappcodexstack.StackV0,
	serverConfig orquestaserver.ConfigV0,
) orquestaserver.StartupCheckPortV0 {
	mode := strings.TrimSpace(os.Getenv("ORQUESTA_STARTUP_CLEANUP_MODE"))
	if mode == "" {
		mode = startupCleanupModeDiagnoseV0
	}
	mode = strings.ToLower(mode)
	if mode == startupCleanupModeOffV0 || mode == "disabled" || mode == "0" {
		return nil
	}
	return serverStartupCheckV0{
		Stack:        stack,
		ServerConfig: serverConfig,
		Mode:         mode,
		QueueLimit:   intEnvOrDefaultV0("ORQUESTA_STARTUP_QUEUE_LIMIT", 0),
	}
}

func (check serverStartupCheckV0) PrepareStartupV0(
	ctx context.Context,
	command orquestaserver.StartupCheckCommandV0,
) (orquestaserver.StartupCheckResultV0, error) {
	mode := strings.TrimSpace(check.Mode)
	if mode == "" {
		mode = startupCleanupModeDiagnoseV0
	}
	switch mode {
	case startupCleanupModeDiagnoseV0:
		return check.diagnoseStartupV0(ctx)
	case startupCleanupModeForcedStopV0, "purge", "purge_forced":
		return check.forceStopStartupStateV0(ctx, command)
	default:
		return orquestaserver.StartupCheckResultV0{}, fmt.Errorf("startup_cleanup_mode_invalid")
	}
}

func (check serverStartupCheckV0) diagnoseStartupV0(
	ctx context.Context,
) (orquestaserver.StartupCheckResultV0, error) {
	cleanup, err := check.startupCleanupCandidatesV0(ctx)
	if err != nil {
		return orquestaserver.StartupCheckResultV0{}, err
	}
	if len(cleanup) == 0 {
		return orquestaserver.StartupCheckResultV0{
			Status:       orquestaserver.StartupCheckStatusReadyV0,
			Ready:        true,
			Message:      "director: orquesta preparada; autodiagnostico sin runs transitorios ni cola sucia",
			EvidenceRefs: []string{"evidence-ref-orquesta-startup-diagnose-ready"},
		}, nil
	}
	active, queueDirty := countStartupCleanupCandidatesV0(cleanup)
	return orquestaserver.StartupCheckResultV0{
		Status:       "startup_dirty_runs_detected",
		Ready:        false,
		Message:      fmt.Sprintf("runs transitorios activos=%d cola_desincronizada=%d; usar ORQUESTA_STARTUP_CLEANUP_MODE=forced_stop para purga logica", active, queueDirty),
		EvidenceRefs: []string{"evidence-ref-orquesta-startup-dirty-runs"},
	}, nil
}

func (check serverStartupCheckV0) forceStopStartupStateV0(
	ctx context.Context,
	command orquestaserver.StartupCheckCommandV0,
) (orquestaserver.StartupCheckResultV0, error) {
	cleanup, err := check.startupCleanupCandidatesV0(ctx)
	if err != nil {
		return orquestaserver.StartupCheckResultV0{}, err
	}
	if len(cleanup) == 0 {
		compaction, err := check.compactStartupStateFilesV0(command)
		if err != nil {
			return orquestaserver.StartupCheckResultV0{}, err
		}
		return orquestaserver.StartupCheckResultV0{
			Status:       orquestaserver.StartupCheckStatusReadyV0,
			Ready:        true,
			Message:      startupReadyMessageV0("director: orquesta preparada; no habia runs transitorios activos", compaction),
			EvidenceRefs: startupReadyEvidenceRefsV0([]string{"evidence-ref-orquesta-startup-cleanup-empty"}, compaction),
		}, nil
	}
	completed := 0
	queueSynced := 0
	blocked := 0
	for _, entry := range cleanup {
		if entry.QueueDirty && !entry.Active {
			if err := check.markStartupQueueCandidateStoppedV0(ctx, command, entry.Candidate); err != nil {
				return orquestaserver.StartupCheckResultV0{}, err
			}
			queueSynced++
			continue
		}
		ready, err := check.forceStopStartupCandidateV0(ctx, command, entry.Candidate)
		if err != nil {
			return orquestaserver.StartupCheckResultV0{}, err
		}
		if ready {
			completed++
			continue
		}
		blocked++
	}
	if blocked == 0 {
		compaction, err := check.compactStartupStateFilesV0(command)
		if err != nil {
			return orquestaserver.StartupCheckResultV0{}, err
		}
		return orquestaserver.StartupCheckResultV0{
			Status: orquestaserver.StartupCheckStatusReadyV0,
			Ready:  true,
			Message: startupReadyMessageV0(fmt.Sprintf(
				"director: orquesta preparada; purga logica completada runs=%d cola=%d",
				completed,
				queueSynced,
			), compaction),
			EvidenceRefs: startupReadyEvidenceRefsV0([]string{"evidence-ref-orquesta-startup-ready-after-cleanup"}, compaction),
		}, nil
	}
	return orquestaserver.StartupCheckResultV0{
		Status: "startup_waiting_drain",
		Ready:  false,
		Message: fmt.Sprintf(
			"startup no listo: runs_completados=%d cola_sincronizada=%d runs_bloqueados=%d",
			completed,
			queueSynced,
			blocked,
		),
		EvidenceRefs: []string{"evidence-ref-orquesta-startup-cleanup-pending"},
	}, nil
}

func (check serverStartupCheckV0) forceStopStartupCandidateV0(
	ctx context.Context,
	command orquestaserver.StartupCheckCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) (bool, error) {
	runRef := strings.TrimSpace(candidate.RunRef)
	if runRef == "" {
		return true, nil
	}
	_, err := check.Stack.Stores.RunControl.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef:         runRef,
		RequestedBy:    "orquesta-server-startup",
		Reason:         "purga logica de arranque antes de activar supervisor",
		Forced:         true,
		IdempotencyKey: "idem-orquesta-server-startup-stop-" + startupSafeRefPartV0(runRef),
		EvidenceRefs:   []string{"evidence-ref-orquesta-startup-forced-stop"},
	})
	if err != nil {
		return false, err
	}
	ready, err := check.startupCandidateCanBeCompletedByStartupV0(ctx, runRef)
	if err != nil {
		return false, err
	}
	if !ready {
		return false, nil
	}
	_, err = check.Stack.Stores.RunControl.CompleteRunControlV0(
		ctx,
		orquestaruncontrol.CompleteRunControlCommandV0{
			RunRef:         runRef,
			TargetStatus:   orquestaruncontrol.RunControlStatusStoppedV0,
			RequestedBy:    "orquesta-server-startup",
			Reason:         "purga logica completada sin agentes en vuelo",
			IdempotencyKey: "idem-orquesta-server-startup-complete-" + startupSafeRefPartV0(runRef),
			EvidenceRefs: []string{
				"evidence-ref-orquesta-startup-control-stopped",
			},
		},
	)
	if err != nil {
		return false, err
	}
	if err := check.markStartupQueueCandidateStoppedV0(ctx, command, candidate); err != nil {
		return false, err
	}
	return true, nil
}

func (check serverStartupCheckV0) markStartupQueueCandidateStoppedV0(
	ctx context.Context,
	command orquestaserver.StartupCheckCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) error {
	writer, ok := any(check.Stack.Stores.RunQueue).(startupQueueCandidateWriterV0)
	if !ok || writer == nil {
		return nil
	}
	_, err := writer.UpsertRunSchedulingCandidateV0(
		ctx,
		check.Stack.RunQueue.QueueRef,
		stoppedStartupQueueCandidateV0(candidate, command),
	)
	return err
}

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

	removedQueue, keptQueue := splitStartupQueueRecordsV0(queue.Records, runtimeDir)
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

func (check serverStartupCheckV0) startupCandidateCanBeCompletedByStartupV0(
	ctx context.Context,
	runRef string,
) (bool, error) {
	if strings.TrimSpace(check.Mode) == startupCleanupModeForcedStopV0 {
		return true, nil
	}
	run, err := check.Stack.Stores.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		var notFound orquestacionnucleoapp.RunNotFoundErrorV0
		if errors.As(err, &notFound) {
			return true, nil
		}
		return false, err
	}
	stats := orquestacionnucleoapp.BuildDirectorRunStatsV0(run)
	return stats.Counts.AgentsInFlight == 0, nil
}

func (check serverStartupCheckV0) startupCleanupCandidatesV0(
	ctx context.Context,
) ([]startupCandidateCleanupV0, error) {
	candidates, err := check.Stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{
			QueueRef: check.Stack.RunQueue.QueueRef,
			Limit:    check.QueueLimit,
		},
	)
	if err != nil {
		return nil, err
	}
	cleanup := make([]startupCandidateCleanupV0, 0, len(candidates))
	for _, candidate := range candidates {
		if startupQueueCandidateTerminalV0(candidate) {
			continue
		}
		state, err := check.readStartupRunControlV0(ctx, candidate.RunRef)
		if err != nil {
			return nil, err
		}
		if orquestaruncontrol.IsTerminalRunControlStatusV0(state.Status) {
			cleanup = append(cleanup, startupCandidateCleanupV0{
				Candidate:  candidate,
				QueueDirty: true,
			})
			continue
		}
		cleanup = append(cleanup, startupCandidateCleanupV0{
			Candidate: candidate,
			Active:    true,
		})
	}
	return cleanup, nil
}

func (check serverStartupCheckV0) readStartupRunControlV0(
	ctx context.Context,
	runRef string,
) (orquestaruncontrol.RunControlStateV0, error) {
	state, err := check.Stack.Stores.RunControl.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef},
	)
	if err == nil {
		return state, nil
	}
	var notFound orquestaruncontrol.RunControlStateNotFoundErrorV0
	if errors.As(err, &notFound) {
		return orquestaruncontrol.DefaultRunControlStateV0(runRef), nil
	}
	return orquestaruncontrol.RunControlStateV0{}, err
}

func startupQueueCandidateTerminalV0(
	candidate orquestarunqueue.RunSchedulingCandidateV0,
) bool {
	switch strings.TrimSpace(candidate.Status) {
	case orquestarunqueue.RunStatusCanceledV0,
		orquestarunqueue.RunStatusStoppedV0,
		orquestarunqueue.RunStatusClosedV0:
		return true
	default:
		return false
	}
}

type startupQueueCandidateWriterV0 interface {
	UpsertRunSchedulingCandidateV0(
		context.Context,
		string,
		orquestarunqueue.RunSchedulingCandidateV0,
	) (orquestarunqueue.RunSchedulingCandidateV0, error)
}

type startupStateStoreReloaderV0 interface {
	ReloadFromDiskV0() error
}

type startupStateCompactionV0 struct {
	RevisionDir      string `json:"revision_dir,omitempty"`
	QueueRemoved     int    `json:"queue_removed"`
	QueueKept        int    `json:"queue_kept"`
	ControlRemoved   int    `json:"control_removed"`
	ControlKept      int    `json:"control_kept"`
	RuntimeArchived  int    `json:"runtime_archived"`
	CompactionNeeded bool   `json:"compaction_needed"`
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

type startupQueueSnapshotV0 struct {
	SchemaVersion string                 `json:"schema_version,omitempty"`
	Records       []startupQueueRecordV0 `json:"records"`
}

type startupQueueRecordV0 struct {
	RunRef      string                                    `json:"run_ref"`
	QueueRef    string                                    `json:"queue_ref,omitempty"`
	Candidate   orquestarunqueue.RunSchedulingCandidateV0 `json:"candidate"`
	PurgeReason string                                    `json:"purge_reason,omitempty"`
}

type startupControlSnapshotV0 struct {
	SchemaVersion string                   `json:"schema_version,omitempty"`
	Records       []startupControlRecordV0 `json:"records"`
}

type startupControlRecordV0 struct {
	RunRef      string                               `json:"run_ref"`
	State       orquestaruncontrol.RunControlStateV0 `json:"state"`
	PurgeReason string                               `json:"purge_reason,omitempty"`
}

type startupCandidateCleanupV0 struct {
	Candidate  orquestarunqueue.RunSchedulingCandidateV0
	Active     bool
	QueueDirty bool
}

func countStartupCleanupCandidatesV0(
	candidates []startupCandidateCleanupV0,
) (active int, queueDirty int) {
	for _, candidate := range candidates {
		if candidate.Active {
			active++
		}
		if candidate.QueueDirty {
			queueDirty++
		}
	}
	return active, queueDirty
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
	runtimeDir string,
) ([]startupQueueRecordV0, []startupQueueRecordV0) {
	removed := make([]startupQueueRecordV0, 0, len(records))
	kept := make([]startupQueueRecordV0, 0, len(records))
	for _, record := range records {
		reason := startupQueueRecordPurgeReasonV0(record, runtimeDir)
		if reason == "" {
			kept = append(kept, record)
			continue
		}
		record.PurgeReason = reason
		removed = append(removed, record)
	}
	return removed, kept
}

func startupQueueRecordPurgeReasonV0(record startupQueueRecordV0, runtimeDir string) string {
	status := strings.TrimSpace(record.Candidate.Status)
	if !orquestarunqueue.IsExecutableRunStatusV0(status) {
		return "queue_status_no_ejecutable"
	}
	runRef := startupQueueRecordRunRefV0(record)
	if runRef != "" && startupRunHasCompletedAckV0(runtimeDir, runRef) {
		return "ready_huerfano_con_ack_completed"
	}
	return ""
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
	if strings.TrimSpace(runtimeDir) == "" || strings.TrimSpace(runRef) == "" {
		return false
	}
	root := filepath.Join(runtimeDir, runRef)
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, entry.Name(), "codex_last_message.txt"))
		if err != nil {
			continue
		}
		text := strings.ToLower(string(data))
		if strings.Contains(text, "ack") && strings.Contains(text, "completed") {
			return true
		}
	}
	return false
}

func archiveStartupRuntimeDirsV0(runtimeDir string, revisionDir string, runRefs []string) (int, error) {
	if strings.TrimSpace(runtimeDir) == "" || strings.TrimSpace(revisionDir) == "" {
		return 0, nil
	}
	targetRoot := filepath.Join(revisionDir, "runtime_archived")
	archived := 0
	for _, runRef := range runRefs {
		runRef = strings.TrimSpace(runRef)
		if runRef == "" {
			continue
		}
		source := filepath.Join(runtimeDir, runRef)
		if _, err := os.Stat(source); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return archived, err
		}
		if err := os.MkdirAll(targetRoot, 0o755); err != nil {
			return archived, err
		}
		target := filepath.Join(targetRoot, runRef)
		if err := os.Rename(source, target); err != nil {
			return archived, err
		}
		archived++
	}
	return archived, nil
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

func startupRevisionTimestampV0(value time.Time) string {
	if value.IsZero() {
		value = time.Now().UTC()
	}
	return value.UTC().Format("20060102T150405Z")
}

func startupReadyMessageV0(base string, compaction startupStateCompactionV0) string {
	base = strings.TrimSpace(base)
	if !compaction.CompactionNeeded {
		return base
	}
	return fmt.Sprintf(
		"%s; compactacion_activa cola=%d control=%d runtime=%d revision=%s",
		base,
		compaction.QueueRemoved,
		compaction.ControlRemoved,
		compaction.RuntimeArchived,
		compaction.RevisionDir,
	)
}

func startupReadyEvidenceRefsV0(base []string, compaction startupStateCompactionV0) []string {
	refs := append([]string(nil), base...)
	if compaction.CompactionNeeded {
		refs = append(refs, "evidence-ref-orquesta-startup-state-compacted")
	}
	return appendStartupEvidenceRefV0(refs, "")
}

func firstNonEmptyStartupStringV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func stoppedStartupQueueCandidateV0(
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	command orquestaserver.StartupCheckCommandV0,
) orquestarunqueue.RunSchedulingCandidateV0 {
	candidate.Status = orquestarunqueue.RunStatusStoppedV0
	if !command.OccurredAt.IsZero() {
		candidate.UpdatedAt = command.OccurredAt
	}
	candidate.EvidenceRefs = appendStartupEvidenceRefV0(
		candidate.EvidenceRefs,
		"evidence-ref-orquesta-startup-queue-stopped",
	)
	return candidate
}

func appendStartupEvidenceRefV0(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return append([]string(nil), values...)
	}
	out := make([]string, 0, len(values)+1)
	seen := map[string]struct{}{}
	for _, existing := range values {
		existing = strings.TrimSpace(existing)
		if existing == "" {
			continue
		}
		if _, ok := seen[existing]; ok {
			continue
		}
		seen[existing] = struct{}{}
		out = append(out, existing)
	}
	if _, ok := seen[value]; !ok {
		out = append(out, value)
	}
	return out
}

func startupSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return "unknown"
	}
	return value
}
