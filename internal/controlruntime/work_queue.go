package controlruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type WorkQueueEntry struct {
	MailboxID       int64  `json:"mailbox_id,omitempty"`
	RuntimeOrderID  int64  `json:"runtime_order_id,omitempty"`
	Kind            string `json:"kind,omitempty"`
	Action          string `json:"action,omitempty"`
	TaskID          int64  `json:"task_id,omitempty"`
	VerificationKey string `json:"verification_key,omitempty"`
	State           string `json:"state,omitempty"`
	Title           string `json:"title,omitempty"`
	Reason          string `json:"reason,omitempty"`
	RecordedAt      string `json:"recorded_at"`
}

type workQueueFile struct {
	Version   int              `json:"version"`
	UpdatedAt string           `json:"updated_at"`
	Current   *WorkQueueEntry  `json:"current,omitempty"`
	Recent    []WorkQueueEntry `json:"recent,omitempty"`
}

type WorkQueueRecordInput struct {
	MailboxID       int64
	RuntimeOrderID  int64
	Kind            string
	Action          string
	TaskID          int64
	VerificationKey string
	State           string
	Title           string
	Reason          string
	RecordedAt      time.Time
}

type WorkQueueMatch struct {
	Kind            string
	Action          string
	VerificationKey string
	TaskID          int64
	Within          time.Duration
}

func WorkQueuePathFromMetadataJSON(raw string) string {
	meta := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &meta); err != nil {
		return ""
	}
	candidates := []string{
		stringValueFromMetadata(meta, "trace_dir"),
		dirIfFile(stringValueFromMetadata(meta, "worker_manifest_path")),
		dirIfFile(stringValueFromMetadata(meta, "worker_status_path")),
		dirIfFile(stringValueFromMetadata(meta, "worker_heartbeat_path")),
		dirIfFile(stringValueFromMetadata(meta, "trace_manifest")),
		dirIfFile(stringValueFromMetadata(meta, "log_path")),
	}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		return filepath.Join(candidate, "work-queue.json")
	}
	return ""
}

func RecordWorkQueueFromMetadataJSON(raw string, input WorkQueueRecordInput) error {
	path := WorkQueuePathFromMetadataJSON(raw)
	if strings.TrimSpace(path) == "" {
		return nil
	}
	file, err := loadWorkQueueFile(path)
	if err != nil {
		return err
	}
	recordedAt := input.RecordedAt.UTC()
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}
	entry := WorkQueueEntry{
		MailboxID:       input.MailboxID,
		RuntimeOrderID:  input.RuntimeOrderID,
		Kind:            strings.TrimSpace(input.Kind),
		Action:          strings.TrimSpace(input.Action),
		TaskID:          input.TaskID,
		VerificationKey: strings.TrimSpace(input.VerificationKey),
		State:           strings.TrimSpace(input.State),
		Title:           strings.TrimSpace(input.Title),
		Reason:          strings.TrimSpace(input.Reason),
		RecordedAt:      recordedAt.Format(time.RFC3339Nano),
	}
	file.Version = 1
	file.UpdatedAt = entry.RecordedAt
	file.Current = &entry
	file.Recent = append(trimWorkQueueEntries(file.Recent), entry)
	return writeJSONAtomic(path, file)
}

func MarkWorkQueueStateFromMetadataJSON(raw string, mailboxID int64, state, reason string, recordedAt time.Time) error {
	path := WorkQueuePathFromMetadataJSON(raw)
	if strings.TrimSpace(path) == "" || mailboxID <= 0 {
		return nil
	}
	file, err := loadWorkQueueFile(path)
	if err != nil {
		return err
	}
	recordedAt = recordedAt.UTC()
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}
	state = strings.TrimSpace(state)
	reason = strings.TrimSpace(reason)
	update := func(entry *WorkQueueEntry) {
		if entry == nil || entry.MailboxID != mailboxID {
			return
		}
		if state != "" {
			entry.State = state
		}
		if reason != "" {
			entry.Reason = reason
		}
		entry.RecordedAt = recordedAt.Format(time.RFC3339Nano)
	}
	update(file.Current)
	for i := range file.Recent {
		if file.Recent[i].MailboxID == mailboxID {
			update(&file.Recent[i])
		}
	}
	file.Version = 1
	file.UpdatedAt = recordedAt.Format(time.RFC3339Nano)
	return writeJSONAtomic(path, file)
}

func HasPendingWorkQueueEntryFromMetadataJSON(raw string, match WorkQueueMatch) (bool, error) {
	entries, err := readWorkQueueEntriesFromMetadataJSON(raw)
	if err != nil || len(entries) == 0 {
		return false, err
	}
	for _, entry := range entries {
		if !workQueueEntryMatches(entry, match) {
			continue
		}
		if workQueueStatePending(entry.State) {
			return true, nil
		}
	}
	return false, nil
}

func HasRecentWorkQueueEntryFromMetadataJSON(raw string, match WorkQueueMatch) (bool, error) {
	entries, err := readWorkQueueEntriesFromMetadataJSON(raw)
	if err != nil || len(entries) == 0 {
		return false, err
	}
	cutoff := time.Time{}
	if match.Within > 0 {
		cutoff = time.Now().UTC().Add(-match.Within)
	}
	for _, entry := range entries {
		if !workQueueEntryMatches(entry, match) {
			continue
		}
		moment := workQueueEntryMoment(entry)
		if !cutoff.IsZero() && moment.Before(cutoff) {
			continue
		}
		if workQueueStateRecentEligible(entry.State) {
			return true, nil
		}
	}
	return false, nil
}

func HasStartedWorkQueueEntryFromMetadataJSON(raw string, match WorkQueueMatch) (bool, error) {
	entries, err := readWorkQueueEntriesFromMetadataJSON(raw)
	if err != nil || len(entries) == 0 {
		return false, err
	}
	cutoff := time.Time{}
	if match.Within > 0 {
		cutoff = time.Now().UTC().Add(-match.Within)
	}
	for _, entry := range entries {
		if !workQueueEntryMatches(entry, match) {
			continue
		}
		moment := workQueueEntryMoment(entry)
		if !cutoff.IsZero() && moment.Before(cutoff) {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(entry.State)) {
		case "working", "running":
			return true, nil
		}
	}
	return false, nil
}

func loadWorkQueueFile(path string) (*workQueueFile, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return &workQueueFile{Version: 1}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &workQueueFile{Version: 1}, nil
		}
		return nil, err
	}
	var file workQueueFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	if file.Version <= 0 {
		file.Version = 1
	}
	file.Recent = trimWorkQueueEntries(file.Recent)
	return &file, nil
}

func readWorkQueueEntriesFromMetadataJSON(raw string) ([]WorkQueueEntry, error) {
	path := WorkQueuePathFromMetadataJSON(raw)
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	file, err := loadWorkQueueFile(path)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, nil
	}
	return enumerateWorkQueueEntries(file), nil
}

func enumerateWorkQueueEntries(file *workQueueFile) []WorkQueueEntry {
	if file == nil {
		return nil
	}
	out := make([]WorkQueueEntry, 0, len(file.Recent)+1)
	if file.Current != nil {
		out = append(out, *file.Current)
	}
	seen := map[int64]bool{}
	if file.Current != nil && file.Current.MailboxID > 0 {
		seen[file.Current.MailboxID] = true
	}
	for i := len(file.Recent) - 1; i >= 0; i-- {
		entry := file.Recent[i]
		if entry.MailboxID > 0 && seen[entry.MailboxID] {
			continue
		}
		if entry.MailboxID > 0 {
			seen[entry.MailboxID] = true
		}
		out = append(out, entry)
	}
	return out
}

func workQueueEntryMatches(entry WorkQueueEntry, match WorkQueueMatch) bool {
	if kind := strings.TrimSpace(match.Kind); kind != "" && !strings.EqualFold(strings.TrimSpace(entry.Kind), kind) {
		return false
	}
	if action := strings.TrimSpace(match.Action); action != "" && !strings.EqualFold(strings.TrimSpace(entry.Action), action) {
		return false
	}
	if verificationKey := strings.TrimSpace(match.VerificationKey); verificationKey != "" &&
		!strings.EqualFold(strings.TrimSpace(entry.VerificationKey), verificationKey) {
		return false
	}
	if match.TaskID > 0 && entry.TaskID != match.TaskID {
		return false
	}
	return true
}

func workQueueStatePending(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "", "pending", "notified", "delivered", "claimed", "working", "running":
		return true
	default:
		return false
	}
}

func workQueueStateRecentEligible(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "superseded", "cancelled", "canceled":
		return false
	default:
		return true
	}
}

func workQueueEntryMoment(entry WorkQueueEntry) time.Time {
	if ts, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(entry.RecordedAt)); err == nil {
		return ts.UTC()
	}
	return time.Time{}
}

func trimWorkQueueEntries(entries []WorkQueueEntry) []WorkQueueEntry {
	const maxEntries = 128
	if len(entries) <= maxEntries {
		return append([]WorkQueueEntry(nil), entries...)
	}
	return append([]WorkQueueEntry(nil), entries[len(entries)-maxEntries:]...)
}
