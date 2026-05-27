package orquestadomainworkfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	domainWorkFileSnapshotMaxBytesV0   int64 = 16 * 1024 * 1024
	domainWorkFileSnapshotMaxRecordsV0       = 10000
)

type domainWorkFileJobSnapshotV0 struct {
	SchemaVersion string                          `json:"schema_version"`
	Records       []domainWorkFileJobRecordJSONV0 `json:"records"`
}

type domainWorkFileJobRecordJSONV0 struct {
	DomainRef      string                                    `json:"domain_ref"`
	IdempotencyKey string                                    `json:"idempotency_key"`
	Fingerprint    string                                    `json:"fingerprint"`
	Request        orquestadomainwork.DomainWorkJobRequestV0 `json:"request"`
	Job            orquestadomainwork.DomainWorkJobV0        `json:"job"`
}

func loadDomainWorkFileRecordsV0(
	path string,
) (map[domainWorkFileJobKeyV0]domainWorkFileJobRecordV0, error) {
	var snapshot domainWorkFileJobSnapshotV0
	ok, err := readDomainWorkFileSnapshotV0(path, &snapshot)
	if err != nil {
		return nil, err
	}
	if !ok {
		return map[domainWorkFileJobKeyV0]domainWorkFileJobRecordV0{}, nil
	}
	if snapshot.SchemaVersion != DomainWorkFileJobCreatorSnapshotSchemaV0 {
		return nil, fmt.Errorf("orquesta_domain_work_file: schema_invalid")
	}
	if len(snapshot.Records) > domainWorkFileSnapshotMaxRecordsV0 {
		return nil, fmt.Errorf("orquesta_domain_work_file: records_limit_exceeded")
	}
	records := make(map[domainWorkFileJobKeyV0]domainWorkFileJobRecordV0, len(snapshot.Records))
	jobsByRef := map[string]struct{}{}
	for _, item := range snapshot.Records {
		key := domainWorkFileJobKeyV0{
			DomainRef:      item.DomainRef,
			IdempotencyKey: item.IdempotencyKey,
		}
		if key.DomainRef == "" ||
			key.IdempotencyKey == "" ||
			item.Fingerprint == "" ||
			item.Job.JobRef == "" {
			return nil, fmt.Errorf("orquesta_domain_work_file: record_invalid")
		}
		if _, ok := records[key]; ok {
			return nil, fmt.Errorf("orquesta_domain_work_file: duplicate_key")
		}
		if _, ok := jobsByRef[item.Job.JobRef]; ok {
			return nil, fmt.Errorf("orquesta_domain_work_file: duplicate_job_ref")
		}
		jobsByRef[item.Job.JobRef] = struct{}{}
		records[key] = domainWorkFileJobRecordV0{
			Request:     cloneDomainWorkFileJobRequestV0(item.Request),
			Job:         cloneDomainWorkFileJobV0(item.Job),
			Fingerprint: item.Fingerprint,
		}
	}
	return records, nil
}

func writeDomainWorkFileSnapshotV0(
	path string,
	records map[domainWorkFileJobKeyV0]domainWorkFileJobRecordV0,
) error {
	snapshot := domainWorkFileJobSnapshotV0{
		SchemaVersion: DomainWorkFileJobCreatorSnapshotSchemaV0,
		Records:       domainWorkFileRecordsJSONV0(records),
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("orquesta_domain_work_file: json_failed")
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("orquesta_domain_work_file: temp_failed")
	}
	tmpName := tmp.Name()
	keepTemp := false
	defer removeDomainWorkFileTempV0(tmpName, &keepTemp)
	if err := writeDomainWorkFileTempV0(tmp, data); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("orquesta_domain_work_file: rename_failed")
	}
	keepTemp = true
	return syncDomainWorkFileDirV0(filepath.Dir(path))
}

func readDomainWorkFileSnapshotV0(path string, target any) (bool, error) {
	data, err := readDomainWorkFileSnapshotBytesV0(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return true, fmt.Errorf("orquesta_domain_work_file: json_invalid")
	}
	return true, nil
}

func readDomainWorkFileSnapshotBytesV0(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("orquesta_domain_work_file: read_failed")
	}
	if info.Size() > domainWorkFileSnapshotMaxBytesV0 {
		return nil, fmt.Errorf("orquesta_domain_work_file: size_limit_exceeded")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("orquesta_domain_work_file: read_failed")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, domainWorkFileSnapshotMaxBytesV0+1))
	if err != nil {
		return nil, fmt.Errorf("orquesta_domain_work_file: read_failed")
	}
	if int64(len(data)) > domainWorkFileSnapshotMaxBytesV0 {
		return nil, fmt.Errorf("orquesta_domain_work_file: size_limit_exceeded")
	}
	return data, nil
}

func domainWorkFileRecordsJSONV0(
	records map[domainWorkFileJobKeyV0]domainWorkFileJobRecordV0,
) []domainWorkFileJobRecordJSONV0 {
	keys := make([]domainWorkFileJobKeyV0, 0, len(records))
	for key := range records {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i int, j int) bool {
		left := records[keys[i]].Job.JobRef
		right := records[keys[j]].Job.JobRef
		if left == right {
			if keys[i].DomainRef == keys[j].DomainRef {
				return keys[i].IdempotencyKey < keys[j].IdempotencyKey
			}
			return keys[i].DomainRef < keys[j].DomainRef
		}
		return left < right
	})
	out := make([]domainWorkFileJobRecordJSONV0, 0, len(keys))
	for _, key := range keys {
		record := records[key]
		out = append(out, domainWorkFileJobRecordJSONV0{
			DomainRef:      key.DomainRef,
			IdempotencyKey: key.IdempotencyKey,
			Fingerprint:    record.Fingerprint,
			Request:        cloneDomainWorkFileJobRequestV0(record.Request),
			Job:            cloneDomainWorkFileJobV0(record.Job),
		})
	}
	return out
}

func writeDomainWorkFileTempV0(tmp *os.File, data []byte) error {
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("orquesta_domain_work_file: chmod_failed")
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("orquesta_domain_work_file: write_failed")
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("orquesta_domain_work_file: sync_failed")
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("orquesta_domain_work_file: close_failed")
	}
	return nil
}

func removeDomainWorkFileTempV0(path string, keep *bool) {
	if keep != nil && *keep {
		return
	}
	_ = os.Remove(path)
}

func syncDomainWorkFileDirV0(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("orquesta_domain_work_file: dir_sync_failed")
	}
	defer handle.Close()
	if err := handle.Sync(); err != nil {
		return fmt.Errorf("orquesta_domain_work_file: dir_sync_failed")
	}
	return nil
}
