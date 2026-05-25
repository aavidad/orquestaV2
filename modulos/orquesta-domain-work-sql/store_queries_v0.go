package orquestadomainworksql

import (
	"context"
	"encoding/json"
	"fmt"
)

func (store *SQLDomainWorkJobRecordStoreV0) findByKeyV0(
	ctx context.Context,
	domainRef string,
	idempotencyKey string,
) (sqlDomainWorkJobRecordV0, bool, error) {
	rows, err := store.db.QueryContext(
		ctx,
		fmt.Sprintf(
			"SELECT fingerprint, request_json, job_json FROM %s WHERE domain_ref = %s AND idempotency_key = %s",
			store.table,
			store.placeholderV0(1),
			store.placeholderV0(2),
		),
		domainRef,
		idempotencyKey,
	)
	if err != nil {
		return sqlDomainWorkJobRecordV0{}, false, fmt.Errorf("orquesta_domain_work_sql: query_failed")
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return sqlDomainWorkJobRecordV0{}, false, fmt.Errorf("orquesta_domain_work_sql: rows_failed")
		}
		return sqlDomainWorkJobRecordV0{}, false, nil
	}
	var fingerprint string
	var requestJSON []byte
	var jobJSON []byte
	if err := rows.Scan(&fingerprint, &requestJSON, &jobJSON); err != nil {
		return sqlDomainWorkJobRecordV0{}, false, fmt.Errorf("orquesta_domain_work_sql: scan_failed")
	}
	record, err := decodeDomainWorkSQLRecordV0(requestJSON, jobJSON)
	if err != nil {
		return sqlDomainWorkJobRecordV0{}, false, err
	}
	record.Fingerprint = fingerprint
	return record, true, nil
}

func (store *SQLDomainWorkJobRecordStoreV0) jobRefExistsV0(
	ctx context.Context,
	jobRef string,
) (bool, error) {
	rows, err := store.db.QueryContext(
		ctx,
		fmt.Sprintf(
			"SELECT job_ref FROM %s WHERE job_ref = %s LIMIT 1",
			store.table,
			store.placeholderV0(1),
		),
		jobRef,
	)
	if err != nil {
		return false, fmt.Errorf("orquesta_domain_work_sql: query_failed")
	}
	defer rows.Close()
	if rows.Next() {
		return true, nil
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("orquesta_domain_work_sql: rows_failed")
	}
	return false, nil
}

func (store *SQLDomainWorkJobRecordStoreV0) insertRecordV0(
	ctx context.Context,
	record sqlDomainWorkJobRecordV0,
) error {
	requestJSON, err := json.Marshal(cloneDomainWorkSQLJobRequestV0(record.Request))
	if err != nil {
		return fmt.Errorf("orquesta_domain_work_sql: json_failed")
	}
	jobJSON, err := json.Marshal(cloneDomainWorkSQLJobV0(record.Job))
	if err != nil {
		return fmt.Errorf("orquesta_domain_work_sql: json_failed")
	}
	_, err = store.db.ExecContext(
		ctx,
		fmt.Sprintf(
			"INSERT INTO %s (domain_ref, idempotency_key, job_ref, correlation_id, work_kind, status, request_json, job_json, fingerprint) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)",
			store.table,
			store.placeholderV0(1),
			store.placeholderV0(2),
			store.placeholderV0(3),
			store.placeholderV0(4),
			store.placeholderV0(5),
			store.placeholderV0(6),
			store.placeholderV0(7),
			store.placeholderV0(8),
			store.placeholderV0(9),
		),
		record.Job.DomainRef,
		record.Job.IdempotencyKey,
		record.Job.JobRef,
		record.Job.CorrelationID,
		record.Job.WorkKind,
		record.Job.Status,
		requestJSON,
		jobJSON,
		record.Fingerprint,
	)
	if err != nil {
		return fmt.Errorf("orquesta_domain_work_sql: insert_failed: %w", err)
	}
	return nil
}
