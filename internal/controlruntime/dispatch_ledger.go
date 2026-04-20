package controlruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DispatchLedgerEntry struct {
	RuntimeOrderID           int64  `json:"runtime_order_id,omitempty"`
	MailboxID                int64  `json:"mailbox_id,omitempty"`
	HandleID                 int64  `json:"handle_id,omitempty"`
	DeliveryAttemptSignature string `json:"delivery_attempt_signature,omitempty"`
	ExternalSessionID        string `json:"external_session_id,omitempty"`
	DispatchState            string `json:"dispatch_state,omitempty"`
	DeliveryState            string `json:"delivery_state,omitempty"`
	ReceiptSource            string `json:"receipt_source,omitempty"`
	Reason                   string `json:"reason,omitempty"`
	RecordedAt               string `json:"recorded_at"`
}

type dispatchLedgerFile struct {
	Version   int                   `json:"version"`
	UpdatedAt string                `json:"updated_at"`
	Entries   []DispatchLedgerEntry `json:"entries"`
}

type DispatchLedgerRecordInput struct {
	RuntimeOrderID           int64
	MailboxID                int64
	HandleID                 int64
	DeliveryAttemptSignature string
	ExternalSessionID        string
	DispatchState            string
	DeliveryState            string
	ReceiptSource            string
	Reason                   string
	RecordedAt               time.Time
}

func DispatchLedgerPathFromMetadataJSON(raw string) string {
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
		return filepath.Join(candidate, "dispatch-ledger.json")
	}
	return ""
}

func ReadDispatchLedgerFromMetadataJSON(raw string) ([]DispatchLedgerEntry, error) {
	path := DispatchLedgerPathFromMetadataJSON(raw)
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	return readDispatchLedger(path)
}

func RecordDispatchLedgerFromMetadataJSON(raw string, input DispatchLedgerRecordInput) error {
	path := DispatchLedgerPathFromMetadataJSON(raw)
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return recordDispatchLedger(path, input)
}

func FindDispatchLedgerEntryFromMetadataJSON(raw string, mailboxID int64, deliveryAttemptSignature, externalSessionID string) (*DispatchLedgerEntry, error) {
	entries, err := ReadDispatchLedgerFromMetadataJSON(raw)
	if err != nil || len(entries) == 0 {
		return nil, err
	}
	deliveryAttemptSignature = strings.TrimSpace(deliveryAttemptSignature)
	externalSessionID = strings.TrimSpace(externalSessionID)
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if mailboxID > 0 && entry.MailboxID != mailboxID {
			continue
		}
		if deliveryAttemptSignature != "" && strings.TrimSpace(entry.DeliveryAttemptSignature) != deliveryAttemptSignature {
			continue
		}
		if deliveryAttemptSignature == "" && externalSessionID != "" {
			entrySessionID := strings.TrimSpace(entry.ExternalSessionID)
			if entrySessionID != "" && entrySessionID != externalSessionID {
				continue
			}
		}
		cp := entry
		return &cp, nil
	}
	return nil, nil
}

func recordDispatchLedger(path string, input DispatchLedgerRecordInput) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	recordedAt := input.RecordedAt.UTC()
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}
	file, err := loadDispatchLedgerFile(path)
	if err != nil {
		return err
	}
	entry := DispatchLedgerEntry{
		RuntimeOrderID:           input.RuntimeOrderID,
		MailboxID:                input.MailboxID,
		HandleID:                 input.HandleID,
		DeliveryAttemptSignature: strings.TrimSpace(input.DeliveryAttemptSignature),
		ExternalSessionID:        strings.TrimSpace(input.ExternalSessionID),
		DispatchState:            strings.TrimSpace(input.DispatchState),
		DeliveryState:            strings.TrimSpace(input.DeliveryState),
		ReceiptSource:            strings.TrimSpace(input.ReceiptSource),
		Reason:                   strings.TrimSpace(input.Reason),
		RecordedAt:               recordedAt.Format(time.RFC3339Nano),
	}
	file.Version = 1
	file.UpdatedAt = recordedAt.Format(time.RFC3339Nano)
	file.Entries = append(trimDispatchLedgerEntries(file.Entries), entry)
	return writeJSONAtomic(path, file)
}

func readDispatchLedger(path string) ([]DispatchLedgerEntry, error) {
	file, err := loadDispatchLedgerFile(path)
	if err != nil {
		return nil, err
	}
	if len(file.Entries) == 0 {
		return nil, nil
	}
	out := make([]DispatchLedgerEntry, len(file.Entries))
	copy(out, file.Entries)
	return out, nil
}

func loadDispatchLedgerFile(path string) (*dispatchLedgerFile, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return &dispatchLedgerFile{Version: 1}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &dispatchLedgerFile{Version: 1}, nil
		}
		return nil, err
	}
	var file dispatchLedgerFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	if file.Version <= 0 {
		file.Version = 1
	}
	file.Entries = trimDispatchLedgerEntries(file.Entries)
	return &file, nil
}

func trimDispatchLedgerEntries(entries []DispatchLedgerEntry) []DispatchLedgerEntry {
	const maxEntries = 256
	if len(entries) <= maxEntries {
		return append([]DispatchLedgerEntry(nil), entries...)
	}
	return append([]DispatchLedgerEntry(nil), entries[len(entries)-maxEntries:]...)
}

func dirIfFile(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Dir(path)
}
