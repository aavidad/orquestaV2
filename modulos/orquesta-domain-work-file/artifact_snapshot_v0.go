package orquestadomainworkfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

type domainWorkFileArtifactSnapshotV0 struct {
	SchemaVersion string                               `json:"schema_version"`
	Records       []domainWorkFileArtifactRecordJSONV0 `json:"records"`
}

type domainWorkFileArtifactRecordJSONV0 struct {
	DomainRef      string                                            `json:"domain_ref"`
	IdempotencyKey string                                            `json:"idempotency_key"`
	Fingerprint    string                                            `json:"fingerprint"`
	Submission     orquestadomainwork.DomainWorkArtifactSubmissionV0 `json:"submission"`
	Receipt        orquestadomainwork.DomainWorkArtifactReceiptV0    `json:"receipt"`
}

func loadDomainWorkFileArtifactRecordsV0(
	path string,
) (map[domainWorkFileArtifactKeyV0]domainWorkFileArtifactRecordV0, error) {
	var snapshot domainWorkFileArtifactSnapshotV0
	ok, err := readDomainWorkFileSnapshotV0(path, &snapshot)
	if err != nil {
		return nil, err
	}
	if !ok {
		return map[domainWorkFileArtifactKeyV0]domainWorkFileArtifactRecordV0{}, nil
	}
	if snapshot.SchemaVersion != DomainWorkFileArtifactSnapshotSchemaV0 {
		return nil, fmt.Errorf("orquesta_domain_work_file_artifact: schema_invalid")
	}
	if len(snapshot.Records) > domainWorkFileSnapshotMaxRecordsV0 {
		return nil, fmt.Errorf("orquesta_domain_work_file_artifact: records_limit_exceeded")
	}
	records := make(map[domainWorkFileArtifactKeyV0]domainWorkFileArtifactRecordV0, len(snapshot.Records))
	for _, item := range snapshot.Records {
		key := domainWorkFileArtifactKeyV0{
			DomainRef:      item.DomainRef,
			IdempotencyKey: item.IdempotencyKey,
		}
		if key.DomainRef == "" ||
			key.IdempotencyKey == "" ||
			item.Fingerprint == "" ||
			item.Receipt.ReceiptRef == "" {
			return nil, fmt.Errorf("orquesta_domain_work_file_artifact: record_invalid")
		}
		if _, ok := records[key]; ok {
			return nil, fmt.Errorf("orquesta_domain_work_file_artifact: duplicate_key")
		}
		records[key] = domainWorkFileArtifactRecordV0{
			Submission:  cloneDomainWorkFileArtifactSubmissionV0(item.Submission),
			Receipt:     cloneDomainWorkFileArtifactReceiptV0(item.Receipt),
			Fingerprint: item.Fingerprint,
		}
	}
	return records, nil
}

func writeDomainWorkFileArtifactSnapshotV0(
	path string,
	records map[domainWorkFileArtifactKeyV0]domainWorkFileArtifactRecordV0,
) error {
	snapshot := domainWorkFileArtifactSnapshotV0{
		SchemaVersion: DomainWorkFileArtifactSnapshotSchemaV0,
		Records:       domainWorkFileArtifactRecordsJSONV0(records),
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("orquesta_domain_work_file_artifact: json_failed")
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("orquesta_domain_work_file_artifact: temp_failed")
	}
	tmpName := tmp.Name()
	keepTemp := false
	defer removeDomainWorkFileTempV0(tmpName, &keepTemp)
	if err := writeDomainWorkFileTempV0(tmp, data); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("orquesta_domain_work_file_artifact: rename_failed")
	}
	keepTemp = true
	return syncDomainWorkFileDirV0(filepath.Dir(path))
}

func domainWorkFileArtifactRecordsJSONV0(
	records map[domainWorkFileArtifactKeyV0]domainWorkFileArtifactRecordV0,
) []domainWorkFileArtifactRecordJSONV0 {
	keys := make([]domainWorkFileArtifactKeyV0, 0, len(records))
	for key := range records {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i int, j int) bool {
		left := records[keys[i]].Receipt.ReceiptRef
		right := records[keys[j]].Receipt.ReceiptRef
		if left == right {
			if keys[i].DomainRef == keys[j].DomainRef {
				return keys[i].IdempotencyKey < keys[j].IdempotencyKey
			}
			return keys[i].DomainRef < keys[j].DomainRef
		}
		return left < right
	})
	out := make([]domainWorkFileArtifactRecordJSONV0, 0, len(keys))
	for _, key := range keys {
		record := records[key]
		out = append(out, domainWorkFileArtifactRecordJSONV0{
			DomainRef:      key.DomainRef,
			IdempotencyKey: key.IdempotencyKey,
			Fingerprint:    record.Fingerprint,
			Submission:     cloneDomainWorkFileArtifactSubmissionV0(record.Submission),
			Receipt:        cloneDomainWorkFileArtifactReceiptV0(record.Receipt),
		})
	}
	return out
}
