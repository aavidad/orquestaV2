package main

import (
	"context"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

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
	RevisionDir      string `json:"-"`
	RevisionRef      string `json:"revision_ref,omitempty"`
	RetentionDays    int    `json:"retention_days,omitempty"`
	MaxBytes         int64  `json:"max_bytes,omitempty"`
	Artifacts        int    `json:"artifacts,omitempty"`
	QueueRemoved     int    `json:"queue_removed"`
	QueueKept        int    `json:"queue_kept"`
	ControlRemoved   int    `json:"control_removed"`
	ControlKept      int    `json:"control_kept"`
	RuntimeArchived  int    `json:"runtime_archived"`
	CompactionNeeded bool   `json:"compaction_needed"`
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
	Candidate               orquestarunqueue.RunSchedulingCandidateV0
	Active                  bool
	QueueDirty              bool
	QueueStatus             string
	DomainSessionSuppressed bool
	CompleteControlStatus   orquestaruncontrol.RunControlStatusV0
	CompleteControlReason   string
	CompleteControlPending  bool
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
