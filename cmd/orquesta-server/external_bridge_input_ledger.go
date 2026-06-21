package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type externalBridgeInputLedgerV0 interface {
	LookupExternalBridgeInputV0(
		ctx context.Context,
		key string,
	) (externalBridgeInputLedgerEntryV0, bool, error)
	ClaimExternalBridgeInputV0(
		ctx context.Context,
		entry externalBridgeInputLedgerEntryV0,
	) (externalBridgeInputLedgerEntryV0, bool, error)
	RecordExternalBridgeInputSubmittedV0(
		ctx context.Context,
		entry externalBridgeInputLedgerEntryV0,
	) error
	UpsertExternalBridgeInputV0(
		ctx context.Context,
		entry externalBridgeInputLedgerEntryV0,
	) error
}

type externalBridgeInputLedgerEntryV0 struct {
	Key            string    `json:"key"`
	ExternalSystem string    `json:"external_system"`
	ExternalJobRef string    `json:"external_job_ref"`
	Status         string    `json:"status"`
	ClaimRef       string    `json:"claim_ref,omitempty"`
	CorrelationID  string    `json:"correlation_id,omitempty"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
	RunRef         string    `json:"run_ref,omitempty"`
	ChangeRef      string    `json:"change_ref,omitempty"`
	LastError      string    `json:"last_error,omitempty"`
	Attempts       int       `json:"attempts,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

const (
	externalBridgeInputStatusClaimedV0      = "claimed"
	externalBridgeInputStatusSubmitFailedV0 = "submit_failed"
	externalBridgeInputStatusSubmittedV0    = "submitted"

	externalBridgeInputLedgerMaxBytesV0   int64 = 8 * 1024 * 1024
	externalBridgeInputLedgerMaxRecordsV0       = 10000
)

type fileExternalBridgeInputLedgerV0 struct {
	path string
	mu   sync.Mutex
}

type externalBridgeInputLedgerSnapshotV0 struct {
	Entries []externalBridgeInputLedgerEntryV0 `json:"entries"`
}

func newFileExternalBridgeInputLedgerV0(path string) (*fileExternalBridgeInputLedgerV0, error) {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, fmt.Errorf("external_bridge_input_ledger_path_required")
	}
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("external_bridge_input_ledger_path_invalid")
	}
	return &fileExternalBridgeInputLedgerV0{path: absPath}, nil
}

func externalBridgeInputLedgerKeyV0(externalSystem string, externalJobRef string) string {
	cleanExternalSystem := strings.TrimSpace(externalSystem)
	cleanExternalJobRef := strings.TrimSpace(externalJobRef)
	if cleanExternalSystem == "" || cleanExternalJobRef == "" {
		return ""
	}
	return cleanExternalSystem + ":" + cleanExternalJobRef
}

func externalBridgeSubmittedInputLedgerEntryV0(
	ctx context.Context,
	ledger externalBridgeInputLedgerV0,
	externalSystem string,
	externalJobRef string,
) (externalBridgeInputLedgerEntryV0, bool, error) {
	if ledger == nil {
		return externalBridgeInputLedgerEntryV0{}, false, nil
	}
	entry, ok, err := ledger.LookupExternalBridgeInputV0(
		ctx,
		externalBridgeInputLedgerKeyV0(externalSystem, externalJobRef),
	)
	if err != nil || !ok {
		return externalBridgeInputLedgerEntryV0{}, false, err
	}
	if entry.Status == externalBridgeInputStatusSubmittedV0 && strings.TrimSpace(entry.RunRef) != "" {
		return entry, true, nil
	}
	return externalBridgeInputLedgerEntryV0{}, false, nil
}

func externalBridgeRecordSubmittedInputV0(
	ctx context.Context,
	ledger externalBridgeInputLedgerV0,
	externalSystem string,
	externalJobRef string,
	runRef string,
	changeRef string,
) error {
	if ledger == nil {
		return nil
	}
	cleanExternalSystem := strings.TrimSpace(externalSystem)
	cleanExternalJobRef := strings.TrimSpace(externalJobRef)
	return ledger.RecordExternalBridgeInputSubmittedV0(ctx, externalBridgeInputLedgerEntryV0{
		Key:            externalBridgeInputLedgerKeyV0(cleanExternalSystem, cleanExternalJobRef),
		ExternalSystem: cleanExternalSystem,
		ExternalJobRef: cleanExternalJobRef,
		Status:         externalBridgeInputStatusSubmittedV0,
		RunRef:         strings.TrimSpace(runRef),
		ChangeRef:      strings.TrimSpace(changeRef),
	})
}

func (ledger *fileExternalBridgeInputLedgerV0) LookupExternalBridgeInputV0(
	ctx context.Context,
	key string,
) (externalBridgeInputLedgerEntryV0, bool, error) {
	if err := ctx.Err(); err != nil {
		return externalBridgeInputLedgerEntryV0{}, false, err
	}
	cleanKey := strings.TrimSpace(key)
	if cleanKey == "" {
		return externalBridgeInputLedgerEntryV0{}, false, fmt.Errorf("external_bridge_input_key_required")
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	entries, err := ledger.loadV0()
	if err != nil {
		return externalBridgeInputLedgerEntryV0{}, false, err
	}
	entry, ok := entries[cleanKey]
	return entry, ok, nil
}

func (ledger *fileExternalBridgeInputLedgerV0) UpsertExternalBridgeInputV0(
	ctx context.Context,
	entry externalBridgeInputLedgerEntryV0,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	entry.Key = strings.TrimSpace(entry.Key)
	if entry.Key == "" {
		return fmt.Errorf("external_bridge_input_key_required")
	}
	entry.UpdatedAt = time.Now().UTC()
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	entries, err := ledger.loadV0()
	if err != nil {
		return err
	}
	if previous, ok := entries[entry.Key]; ok && entry.Attempts <= 0 {
		if previous.Status == externalBridgeInputStatusSubmittedV0 &&
			entry.Status == externalBridgeInputStatusSubmittedV0 &&
			strings.TrimSpace(previous.RunRef) != strings.TrimSpace(entry.RunRef) {
			return fmt.Errorf("external_bridge_submitted_conflict")
		}
		entry.Attempts = previous.Attempts + 1
	}
	if entry.Attempts <= 0 {
		entry.Attempts = 1
	}
	entries[entry.Key] = entry
	return ledger.saveV0(entries)
}

func (ledger *fileExternalBridgeInputLedgerV0) loadV0() (map[string]externalBridgeInputLedgerEntryV0, error) {
	data, err := readExternalBridgeInputLedgerBytesV0(ledger.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]externalBridgeInputLedgerEntryV0{}, nil
		}
		return nil, err
	}
	var snapshot externalBridgeInputLedgerSnapshotV0
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("external_bridge_input_ledger_decode_error")
	}
	if len(snapshot.Entries) > externalBridgeInputLedgerMaxRecordsV0 {
		return nil, fmt.Errorf("external_bridge_input_ledger_records_limit_exceeded")
	}
	entries := make(map[string]externalBridgeInputLedgerEntryV0, len(snapshot.Entries))
	for _, entry := range snapshot.Entries {
		if key := strings.TrimSpace(entry.Key); key != "" {
			entry.Key = key
			entries[key] = entry
		}
	}
	return entries, nil
}

func (ledger *fileExternalBridgeInputLedgerV0) saveV0(
	entries map[string]externalBridgeInputLedgerEntryV0,
) error {
	snapshot := externalBridgeInputLedgerSnapshotV0{
		Entries: make([]externalBridgeInputLedgerEntryV0, 0, len(entries)),
	}
	for _, entry := range entries {
		snapshot.Entries = append(snapshot.Entries, entry)
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("external_bridge_input_ledger_encode_error")
	}
	if err := os.MkdirAll(filepath.Dir(ledger.path), 0o700); err != nil {
		return fmt.Errorf("external_bridge_input_ledger_dir_error")
	}
	data = append(data, '\n')
	return writeCommandDurableFileV0(
		ledger.path,
		data,
		"external_bridge_input_ledger",
	)
}

func readExternalBridgeInputLedgerBytesV0(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("external_bridge_input_ledger_read_error")
	}
	if info.Size() > externalBridgeInputLedgerMaxBytesV0 {
		return nil, fmt.Errorf("external_bridge_input_ledger_size_limit_exceeded")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("external_bridge_input_ledger_read_error")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, externalBridgeInputLedgerMaxBytesV0+1))
	if err != nil {
		return nil, fmt.Errorf("external_bridge_input_ledger_read_error")
	}
	if int64(len(data)) > externalBridgeInputLedgerMaxBytesV0 {
		return nil, fmt.Errorf("external_bridge_input_ledger_size_limit_exceeded")
	}
	return data, nil
}
