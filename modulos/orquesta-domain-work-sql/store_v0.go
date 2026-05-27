package orquestadomainworksql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	DomainWorkSQLDefaultTableV0 = "domain_work_jobs_v0"

	ErrDomainWorkSQLIdempotencyConflictV0 = "domain_work_sql_idempotency_conflict"

	domainWorkSQLJobRefCollisionRetriesV0 = 16
)

type SQLDomainWorkPlaceholderStyleV0 string

const (
	SQLDomainWorkPlaceholderQuestionV0 SQLDomainWorkPlaceholderStyleV0 = "question"
	SQLDomainWorkPlaceholderDollarV0   SQLDomainWorkPlaceholderStyleV0 = "dollar"
)

var _ orquestadomainwork.DomainWorkJobRecordStorePortV0 = (*SQLDomainWorkJobRecordStoreV0)(nil)

type SQLDomainWorkJobRecordStoreConfigV0 struct {
	TableName         string
	PlaceholderStyle  SQLDomainWorkPlaceholderStyleV0
	IsUniqueViolation func(error) bool
}

type SQLDomainWorkJobRecordStoreV0 struct {
	db                *sql.DB
	table             string
	placeholderStyle  SQLDomainWorkPlaceholderStyleV0
	isUniqueViolation func(error) bool
}

type sqlDomainWorkJobRecordV0 struct {
	Request     orquestadomainwork.DomainWorkJobRequestV0
	Job         orquestadomainwork.DomainWorkJobV0
	Fingerprint string
}

func NewSQLDomainWorkJobRecordStoreV0(
	db *sql.DB,
	config SQLDomainWorkJobRecordStoreConfigV0,
) (*SQLDomainWorkJobRecordStoreV0, error) {
	if db == nil {
		return nil, fmt.Errorf("orquesta_domain_work_sql: db_required")
	}
	table := strings.TrimSpace(config.TableName)
	if table == "" {
		table = DomainWorkSQLDefaultTableV0
	}
	if !validDomainWorkSQLIdentifierV0(table) {
		return nil, fmt.Errorf("orquesta_domain_work_sql: table_invalid")
	}
	placeholderStyle := config.PlaceholderStyle
	if placeholderStyle == "" {
		placeholderStyle = SQLDomainWorkPlaceholderQuestionV0
	}
	if !validDomainWorkSQLPlaceholderStyleV0(placeholderStyle) {
		return nil, fmt.Errorf("orquesta_domain_work_sql: placeholder_style_invalid")
	}
	return &SQLDomainWorkJobRecordStoreV0{
		db:                db,
		table:             table,
		placeholderStyle:  placeholderStyle,
		isUniqueViolation: config.IsUniqueViolation,
	}, nil
}

func (store *SQLDomainWorkJobRecordStoreV0) CreateDomainWorkJobV0(
	ctx context.Context,
	request orquestadomainwork.DomainWorkJobRequestV0,
) (orquestadomainwork.DomainWorkJobV0, error) {
	ctx = domainWorkSQLContextV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestadomainwork.DomainWorkJobV0{}, err
	}
	request = orquestadomainwork.NormalizeDomainWorkJobRequestV0(request)
	if issues := orquestadomainwork.ValidateDomainWorkJobRequestV0(request); len(issues) > 0 {
		return invalidDomainWorkSQLJobV0(request, issues), nil
	}
	fingerprint, err := domainWorkSQLRequestFingerprintV0(request)
	if err != nil {
		return orquestadomainwork.DomainWorkJobV0{}, err
	}
	existing, found, err := store.findByKeyV0(ctx, request.DomainRef, request.IdempotencyKey)
	if err != nil {
		return orquestadomainwork.DomainWorkJobV0{}, err
	}
	if found {
		equivalent, err := orquestadomainwork.EquivalentDomainWorkJobRequestsV0(
			existing.Request,
			request,
		)
		if err != nil {
			return orquestadomainwork.DomainWorkJobV0{}, err
		}
		if equivalent {
			return cloneDomainWorkSQLJobV0(existing.Job), nil
		}
		return invalidDomainWorkSQLJobV0(request, []orquestadomainwork.DomainWorkIssueV0{{
			Code:  ErrDomainWorkSQLIdempotencyConflictV0,
			Field: "idempotency_key",
		}}), nil
	}
	for attempt := 0; attempt < domainWorkSQLJobRefCollisionRetriesV0; attempt++ {
		jobRef, err := store.nextJobRefV0(ctx, request.DomainRef, request.IdempotencyKey)
		if err != nil {
			return orquestadomainwork.DomainWorkJobV0{}, err
		}
		job := acceptedDomainWorkSQLJobV0(request, jobRef)
		if err := ctx.Err(); err != nil {
			return orquestadomainwork.DomainWorkJobV0{}, err
		}
		if err := store.insertRecordV0(ctx, sqlDomainWorkJobRecordV0{
			Request:     cloneDomainWorkSQLJobRequestV0(request),
			Job:         cloneDomainWorkSQLJobV0(job),
			Fingerprint: fingerprint,
		}); err != nil {
			if store.uniqueViolationV0(err) {
				replayed, resolved, err := store.jobAfterUniqueViolationV0(ctx, request)
				if err != nil {
					return orquestadomainwork.DomainWorkJobV0{}, err
				}
				if resolved {
					return replayed, nil
				}
				continue
			}
			return orquestadomainwork.DomainWorkJobV0{}, err
		}
		return cloneDomainWorkSQLJobV0(job), nil
	}
	return orquestadomainwork.DomainWorkJobV0{}, fmt.Errorf("orquesta_domain_work_sql: job_ref_collision_unresolved")
}

func (store *SQLDomainWorkJobRecordStoreV0) ListDomainWorkJobRecordsV0(
	ctx context.Context,
	filter orquestadomainwork.DomainWorkJobRecordFilterV0,
) ([]orquestadomainwork.DomainWorkJobRecordV0, error) {
	ctx = domainWorkSQLContextV0(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	filter = orquestadomainwork.NormalizeDomainWorkJobRecordFilterV0(filter)
	rows, err := store.db.QueryContext(
		ctx,
		fmt.Sprintf("SELECT request_json, job_json FROM %s ORDER BY job_ref ASC", store.table),
	)
	if err != nil {
		return nil, fmt.Errorf("orquesta_domain_work_sql: query_failed")
	}
	defer rows.Close()
	records := []orquestadomainwork.DomainWorkJobRecordV0{}
	for rows.Next() {
		var requestJSON []byte
		var jobJSON []byte
		if err := rows.Scan(&requestJSON, &jobJSON); err != nil {
			return nil, fmt.Errorf("orquesta_domain_work_sql: scan_failed")
		}
		record, err := decodeDomainWorkSQLRecordV0(requestJSON, jobJSON)
		if err != nil {
			return nil, err
		}
		if !domainWorkSQLJobRecordMatchesFilterV0(record, filter) {
			continue
		}
		records = append(records, cloneDomainWorkSQLPublicRecordV0(record))
		if filter.Limit > 0 && len(records) >= filter.Limit {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("orquesta_domain_work_sql: rows_failed")
	}
	return records, nil
}
