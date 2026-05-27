package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

const (
	startupRevisionArchiveSchemaV0  = "orquesta_startup_revision_archive.v0"
	startupRevisionRetentionDaysV0  = 14
	startupRevisionMaxSnapshotBytes = 128 * 1024
)

type startupRevisionManifestV0 struct {
	SchemaVersion       string                      `json:"schema_version"`
	RevisionRef         string                      `json:"revision_ref"`
	CreatedAt           string                      `json:"created_at,omitempty"`
	ExpiresAt           string                      `json:"expires_at,omitempty"`
	RetentionDays       int                         `json:"retention_days"`
	MaxSnapshotBytes    int64                       `json:"max_snapshot_bytes"`
	RawMaterialIncluded bool                        `json:"raw_material_included"`
	ArchiveMode         string                      `json:"archive_mode"`
	QueueRemoved        int                         `json:"queue_removed"`
	QueueKept           int                         `json:"queue_kept"`
	ControlRemoved      int                         `json:"control_removed"`
	ControlKept         int                         `json:"control_kept"`
	RuntimeArchived     int                         `json:"runtime_archived"`
	Artifacts           []startupRevisionArtifactV0 `json:"artifacts,omitempty"`
}

type startupRevisionArtifactV0 struct {
	Name             string `json:"name"`
	Kind             string `json:"kind"`
	Records          int    `json:"records,omitempty"`
	Bytes            int64  `json:"bytes,omitempty"`
	RedactionApplied bool   `json:"redaction_applied"`
}

type startupRuntimeArchiveSummaryV0 struct {
	SchemaVersion       string `json:"schema_version"`
	RunRef              string `json:"run_ref"`
	RuntimeRef          string `json:"runtime_ref"`
	Directories         int    `json:"directories"`
	Files               int    `json:"files"`
	Bytes               int64  `json:"bytes"`
	Truncated           bool   `json:"truncated"`
	RawMaterialIncluded bool   `json:"raw_material_included"`
	RedactionApplied    bool   `json:"redaction_applied"`
}

func newStartupRevisionManifestV0(revisionRef string, occurredAt time.Time) startupRevisionManifestV0 {
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	return startupRevisionManifestV0{
		SchemaVersion:       startupRevisionArchiveSchemaV0,
		RevisionRef:         strings.TrimSpace(revisionRef),
		CreatedAt:           occurredAt.UTC().Format(time.RFC3339),
		ExpiresAt:           occurredAt.UTC().AddDate(0, 0, startupRevisionRetentionDaysV0).Format(time.RFC3339),
		RetentionDays:       startupRevisionRetentionDaysV0,
		MaxSnapshotBytes:    startupRevisionMaxSnapshotBytes,
		RawMaterialIncluded: false,
		ArchiveMode:         "redacted_metadata_only",
	}
}

func writeStartupRevisionQueueSnapshotV0(
	revisionDir string,
	name string,
	snapshot startupQueueSnapshotV0,
	manifest *startupRevisionManifestV0,
) error {
	sanitized := startupQueueSnapshotV0{
		SchemaVersion: snapshot.SchemaVersion,
		Records:       startupSanitizedQueueRecordsV0(snapshot.Records),
	}
	return writeStartupRevisionArtifactV0(revisionDir, name, "queue_snapshot", sanitized, len(sanitized.Records), manifest)
}

func writeStartupRevisionControlSnapshotV0(
	revisionDir string,
	name string,
	snapshot startupControlSnapshotV0,
	manifest *startupRevisionManifestV0,
) error {
	sanitized := startupControlSnapshotV0{
		SchemaVersion: snapshot.SchemaVersion,
		Records:       startupSanitizedControlRecordsV0(snapshot.Records),
	}
	return writeStartupRevisionArtifactV0(revisionDir, name, "control_snapshot", sanitized, len(sanitized.Records), manifest)
}

func writeStartupRevisionArtifactV0(
	revisionDir string,
	name string,
	kind string,
	value any,
	records int,
	manifest *startupRevisionManifestV0,
) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if int64(len(data)) > startupRevisionMaxSnapshotBytes {
		value = map[string]any{
			"schema_version":        startupRevisionArchiveSchemaV0,
			"truncated":             true,
			"records":               records,
			"redaction_applied":     true,
			"raw_material_included": false,
		}
		data, err = json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
	}
	data = append(data, '\n')
	path := filepath.Join(revisionDir, name)
	if err := writeStartupRevisionBytesV0(path, data); err != nil {
		return err
	}
	if manifest != nil {
		manifest.Artifacts = append(manifest.Artifacts, startupRevisionArtifactV0{
			Name:             name,
			Kind:             kind,
			Records:          records,
			Bytes:            int64(len(data)),
			RedactionApplied: true,
		})
	}
	return nil
}

func writeStartupRevisionBytesV0(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return writeCommandDurableFileV0(path, data, "startup_revision_archive")
}

func startupSanitizedQueueRecordsV0(records []startupQueueRecordV0) []startupQueueRecordV0 {
	out := make([]startupQueueRecordV0, 0, len(records))
	for _, record := range records {
		record.RunRef = startupRevisionRefOrHashV0(record.RunRef)
		record.QueueRef = startupRevisionRefOrHashV0(record.QueueRef)
		record.PurgeReason = strings.TrimSpace(record.PurgeReason)
		record.Candidate = startupSanitizedQueueCandidateV0(record.Candidate)
		out = append(out, record)
	}
	return out
}

func startupSanitizedQueueCandidateV0(candidate orquestarunqueue.RunSchedulingCandidateV0) orquestarunqueue.RunSchedulingCandidateV0 {
	candidate.RunRef = startupRevisionRefOrHashV0(candidate.RunRef)
	candidate.AppRef = startupRevisionRefOrHashV0(candidate.AppRef)
	candidate.FairnessGroupRef = startupRevisionRefOrHashV0(candidate.FairnessGroupRef)
	candidate.EvidenceRefs = startupRevisionRefsOrHashesV0(candidate.EvidenceRefs)
	return candidate
}

func startupSanitizedControlRecordsV0(records []startupControlRecordV0) []startupControlRecordV0 {
	out := make([]startupControlRecordV0, 0, len(records))
	for _, record := range records {
		record.RunRef = startupRevisionRefOrHashV0(record.RunRef)
		record.PurgeReason = strings.TrimSpace(record.PurgeReason)
		record.State = startupSanitizedControlStateV0(record.State)
		out = append(out, record)
	}
	return out
}

func startupSanitizedControlStateV0(state orquestaruncontrol.RunControlStateV0) orquestaruncontrol.RunControlStateV0 {
	state.RunRef = startupRevisionRefOrHashV0(state.RunRef)
	state.EvidenceRefs = startupRevisionRefsOrHashesV0(state.EvidenceRefs)
	state.Meta.RequestedBy = startupRevisionRefOrHashV0(state.Meta.RequestedBy)
	state.Meta.Reason = startupRevisionMessageV0(state.Meta.Reason)
	state.Meta.IdempotencyKey = startupRevisionRefOrHashV0(state.Meta.IdempotencyKey)
	return state
}

func startupRevisionRefsOrHashesV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if redacted := startupRevisionRefOrHashV0(value); redacted != "" {
			out = append(out, redacted)
		}
	}
	return out
}

func startupRevisionMessageV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if startupRevisionStringLooksRawV0(value) {
		return "redacted-text-ref-" + startupRevisionHashV0(value)
	}
	return value
}

func startupRevisionRefOrHashV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if startupRevisionStringLooksRawV0(value) {
		return "redacted-ref-" + startupRevisionHashV0(value)
	}
	return value
}

func startupRevisionStringLooksRawV0(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, fragment := range []string{"/", "\\", "home", "token", "secret", "authorization", "bearer ", "password", "prompt", "transcript", ".codex", "stdout", "stderr"} {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

func startupRevisionHashV0(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:16]
}

func summarizeStartupRuntimeDirV0(source string, runRef string) (startupRuntimeArchiveSummaryV0, error) {
	summary := startupRuntimeArchiveSummaryV0{
		SchemaVersion:       startupRevisionArchiveSchemaV0,
		RunRef:              startupRevisionRefOrHashV0(runRef),
		RuntimeRef:          "runtime-dir-ref-" + startupRevisionHashV0(source),
		RawMaterialIncluded: false,
		RedactionApplied:    true,
	}
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == source {
			return nil
		}
		if entry.IsDir() {
			summary.Directories++
			return nil
		}
		summary.Files++
		if info, statErr := entry.Info(); statErr == nil {
			summary.Bytes += info.Size()
		}
		if summary.Files > 128 || summary.Bytes > startupRevisionMaxSnapshotBytes {
			summary.Truncated = true
			return filepath.SkipAll
		}
		return nil
	})
	if errors.Is(err, filepath.SkipAll) {
		return summary, nil
	}
	return summary, err
}
