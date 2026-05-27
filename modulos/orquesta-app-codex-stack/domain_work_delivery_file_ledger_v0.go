package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	fileDomainWorkArtifactSubmissionLedgerSchemaV0           = "domain_work_artifact_submission_ledger.v0"
	fileDomainWorkArtifactSubmissionLedgerMaxBytesV0   int64 = 8 * 1024 * 1024
	fileDomainWorkArtifactSubmissionLedgerMaxRecordsV0       = 10000
)

type FileDomainWorkArtifactSubmissionLedgerV0 struct {
	path    string
	mu      sync.Mutex
	loaded  bool
	records map[string]DomainWorkArtifactSubmissionRecordV0
}

type fileDomainWorkArtifactSubmissionLedgerSnapshotV0 struct {
	SchemaVersion string                                 `json:"schema_version"`
	Records       []DomainWorkArtifactSubmissionRecordV0 `json:"records"`
}

func NewFileDomainWorkArtifactSubmissionLedgerV0(
	path string,
) *FileDomainWorkArtifactSubmissionLedgerV0 {
	return &FileDomainWorkArtifactSubmissionLedgerV0{
		path:    strings.TrimSpace(path),
		records: map[string]DomainWorkArtifactSubmissionRecordV0{},
	}
}

func (ledger *FileDomainWorkArtifactSubmissionLedgerV0) HasDomainWorkArtifactSubmissionV0(
	ctx context.Context,
	idempotencyKey string,
) (bool, error) {
	if ledger == nil {
		return false, nil
	}
	if err := contextErrDomainWorkFileLedgerV0(ctx); err != nil {
		return false, err
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	if err := ledger.loadLockedV0(); err != nil {
		return false, err
	}
	_, ok := ledger.records[strings.TrimSpace(idempotencyKey)]
	return ok, nil
}

func (ledger *FileDomainWorkArtifactSubmissionLedgerV0) RecordDomainWorkArtifactSubmissionV0(
	ctx context.Context,
	record DomainWorkArtifactSubmissionRecordV0,
) error {
	if ledger == nil {
		return fmt.Errorf("domain_work_artifact_file_ledger_requerido")
	}
	if err := contextErrDomainWorkFileLedgerV0(ctx); err != nil {
		return err
	}
	record.IdempotencyKey = strings.TrimSpace(record.IdempotencyKey)
	if record.IdempotencyKey == "" {
		return fmt.Errorf("domain_work_artifact_idempotency_key_requerida")
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	if err := ledger.loadLockedV0(); err != nil {
		return err
	}
	record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
	if existing, ok := ledger.records[record.IdempotencyKey]; ok {
		merged, err := mergeDomainWorkArtifactSubmissionRecordV0(existing, record)
		if err != nil {
			return err
		}
		ledger.records[record.IdempotencyKey] = merged
	} else {
		ledger.records[record.IdempotencyKey] = record
	}
	return ledger.persistLockedV0()
}

func (ledger *FileDomainWorkArtifactSubmissionLedgerV0) ListDomainWorkArtifactSubmissionsV0(
	ctx context.Context,
	filter DomainWorkArtifactSubmissionRecordFilterV0,
) ([]DomainWorkArtifactSubmissionRecordV0, error) {
	if ledger == nil {
		return nil, nil
	}
	if err := contextErrDomainWorkFileLedgerV0(ctx); err != nil {
		return nil, err
	}
	filter = normalizeDomainWorkArtifactSubmissionRecordFilterV0(filter)
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	if err := ledger.loadLockedV0(); err != nil {
		return nil, err
	}
	out := make([]DomainWorkArtifactSubmissionRecordV0, 0, len(ledger.records))
	for _, record := range ledger.records {
		record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
		if !domainWorkArtifactSubmissionRecordMatchesFilterV0(record, filter) {
			continue
		}
		out = append(out, record)
	}
	return out, nil
}

func (ledger *FileDomainWorkArtifactSubmissionLedgerV0) loadLockedV0() error {
	if ledger.loaded {
		return nil
	}
	if strings.TrimSpace(ledger.path) == "" {
		return fmt.Errorf("domain_work_artifact_file_ledger_path_requerido")
	}
	data, err := readFileDomainWorkArtifactSubmissionLedgerBytesV0(ledger.path)
	if err != nil {
		if os.IsNotExist(err) {
			ledger.loaded = true
			return nil
		}
		return err
	}
	var snapshot fileDomainWorkArtifactSubmissionLedgerSnapshotV0
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("domain_work_artifact_file_ledger_json_invalido")
	}
	if snapshot.SchemaVersion != fileDomainWorkArtifactSubmissionLedgerSchemaV0 {
		return fmt.Errorf("domain_work_artifact_file_ledger_schema_invalido")
	}
	if len(snapshot.Records) > fileDomainWorkArtifactSubmissionLedgerMaxRecordsV0 {
		return fmt.Errorf("domain_work_artifact_file_ledger_records_limit_exceeded")
	}
	ledger.records = map[string]DomainWorkArtifactSubmissionRecordV0{}
	for _, record := range snapshot.Records {
		record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
		if record.IdempotencyKey == "" {
			continue
		}
		ledger.records[record.IdempotencyKey] = record
	}
	ledger.loaded = true
	return nil
}

func readFileDomainWorkArtifactSubmissionLedgerBytesV0(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("domain_work_artifact_file_ledger_read_error")
	}
	if info.Size() > fileDomainWorkArtifactSubmissionLedgerMaxBytesV0 {
		return nil, fmt.Errorf("domain_work_artifact_file_ledger_size_limit_exceeded")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("domain_work_artifact_file_ledger_read_error")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, fileDomainWorkArtifactSubmissionLedgerMaxBytesV0+1))
	if err != nil {
		return nil, fmt.Errorf("domain_work_artifact_file_ledger_read_error")
	}
	if int64(len(data)) > fileDomainWorkArtifactSubmissionLedgerMaxBytesV0 {
		return nil, fmt.Errorf("domain_work_artifact_file_ledger_size_limit_exceeded")
	}
	return data, nil
}

func (ledger *FileDomainWorkArtifactSubmissionLedgerV0) persistLockedV0() error {
	records := make([]DomainWorkArtifactSubmissionRecordV0, 0, len(ledger.records))
	for _, record := range ledger.records {
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].IdempotencyKey < records[j].IdempotencyKey
	})
	snapshot := fileDomainWorkArtifactSubmissionLedgerSnapshotV0{
		SchemaVersion: fileDomainWorkArtifactSubmissionLedgerSchemaV0,
		Records:       records,
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(ledger.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("domain_work_artifact_file_ledger: dir_unavailable")
	}
	data = append(data, '\n')
	return writeAppStackDurableFileV0(
		ledger.path,
		data,
		"domain_work_artifact_file_ledger",
	)
}

func contextErrDomainWorkFileLedgerV0(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
