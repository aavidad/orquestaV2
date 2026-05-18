package orquestadomainworksql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	DomainWorkSQLDefaultTableV0 = "domain_work_jobs_v0"

	ErrDomainWorkSQLIdempotencyConflictV0 = "domain_work_sql_idempotency_conflict"
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
		if existing.Fingerprint == fingerprint {
			return cloneDomainWorkSQLJobV0(existing.Job), nil
		}
		return invalidDomainWorkSQLJobV0(request, []orquestadomainwork.DomainWorkIssueV0{{
			Code:  ErrDomainWorkSQLIdempotencyConflictV0,
			Field: "idempotency_key",
		}}), nil
	}
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
			return store.jobAfterUniqueViolationV0(
				ctx,
				request,
				fingerprint,
			)
		}
		return orquestadomainwork.DomainWorkJobV0{}, err
	}
	return cloneDomainWorkSQLJobV0(job), nil
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

func (store *SQLDomainWorkJobRecordStoreV0) jobAfterUniqueViolationV0(
	ctx context.Context,
	request orquestadomainwork.DomainWorkJobRequestV0,
	fingerprint string,
) (orquestadomainwork.DomainWorkJobV0, error) {
	existing, found, err := store.findByKeyV0(ctx, request.DomainRef, request.IdempotencyKey)
	if err != nil {
		return orquestadomainwork.DomainWorkJobV0{}, err
	}
	if found {
		if existing.Fingerprint == fingerprint {
			return cloneDomainWorkSQLJobV0(existing.Job), nil
		}
		return invalidDomainWorkSQLJobV0(request, []orquestadomainwork.DomainWorkIssueV0{{
			Code:  ErrDomainWorkSQLIdempotencyConflictV0,
			Field: "idempotency_key",
		}}), nil
	}
	return orquestadomainwork.DomainWorkJobV0{}, fmt.Errorf("orquesta_domain_work_sql: insert_unique_violation_unresolved")
}

func (store *SQLDomainWorkJobRecordStoreV0) uniqueViolationV0(err error) bool {
	return err != nil && store.isUniqueViolation != nil && store.isUniqueViolation(err)
}

func (store *SQLDomainWorkJobRecordStoreV0) nextJobRefV0(
	ctx context.Context,
	domainRef string,
	idempotencyKey string,
) (string, error) {
	base := "domain-work-job-" + domainWorkSQLHashV0(domainRef+"\x00"+idempotencyKey)
	jobRef := base
	for index := 2; ; index++ {
		exists, err := store.jobRefExistsV0(ctx, jobRef)
		if err != nil {
			return "", err
		}
		if !exists {
			return jobRef, nil
		}
		jobRef = base + "-" + strconv.Itoa(index)
	}
}

func acceptedDomainWorkSQLJobV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
	jobRef string,
) orquestadomainwork.DomainWorkJobV0 {
	return orquestadomainwork.DomainWorkJobV0{
		SchemaVersion:  orquestadomainwork.DomainWorkJobSchemaV0,
		Status:         orquestadomainwork.DomainWorkStatusAcceptedV0,
		JobRef:         jobRef,
		DomainRef:      request.DomainRef,
		WorkKind:       request.WorkKind,
		CorrelationID:  request.CorrelationID,
		IdempotencyKey: request.IdempotencyKey,
		ExternalRefs:   cloneDomainWorkSQLExternalRefsV0(request.ExternalRefs),
		EvidenceRefs:   append([]string(nil), request.EvidenceRefs...),
	}
}

func invalidDomainWorkSQLJobV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
	issues []orquestadomainwork.DomainWorkIssueV0,
) orquestadomainwork.DomainWorkJobV0 {
	return orquestadomainwork.DomainWorkJobV0{
		SchemaVersion:  orquestadomainwork.DomainWorkJobSchemaV0,
		Status:         orquestadomainwork.DomainWorkStatusInvalidV0,
		DomainRef:      request.DomainRef,
		WorkKind:       request.WorkKind,
		CorrelationID:  request.CorrelationID,
		IdempotencyKey: request.IdempotencyKey,
		ExternalRefs:   cloneDomainWorkSQLExternalRefsV0(request.ExternalRefs),
		EvidenceRefs:   append([]string(nil), request.EvidenceRefs...),
		Issues:         cloneDomainWorkSQLIssuesV0(issues),
	}
}

func decodeDomainWorkSQLRecordV0(
	requestJSON []byte,
	jobJSON []byte,
) (sqlDomainWorkJobRecordV0, error) {
	var request orquestadomainwork.DomainWorkJobRequestV0
	if err := json.Unmarshal(requestJSON, &request); err != nil {
		return sqlDomainWorkJobRecordV0{}, fmt.Errorf("orquesta_domain_work_sql: request_json_invalid")
	}
	var job orquestadomainwork.DomainWorkJobV0
	if err := json.Unmarshal(jobJSON, &job); err != nil {
		return sqlDomainWorkJobRecordV0{}, fmt.Errorf("orquesta_domain_work_sql: job_json_invalid")
	}
	return sqlDomainWorkJobRecordV0{
		Request: cloneDomainWorkSQLJobRequestV0(request),
		Job:     cloneDomainWorkSQLJobV0(job),
	}, nil
}

func domainWorkSQLJobRecordMatchesFilterV0(
	record sqlDomainWorkJobRecordV0,
	filter orquestadomainwork.DomainWorkJobRecordFilterV0,
) bool {
	job := record.Job
	if filter.DomainRef != "" && job.DomainRef != filter.DomainRef {
		return false
	}
	if filter.IdempotencyKey != "" && job.IdempotencyKey != filter.IdempotencyKey {
		return false
	}
	if filter.WorkKind != "" && job.WorkKind != filter.WorkKind {
		return false
	}
	if filter.JobRef != "" && job.JobRef != filter.JobRef {
		return false
	}
	if filter.CorrelationID != "" && job.CorrelationID != filter.CorrelationID {
		return false
	}
	if filter.Status != "" && job.Status != filter.Status {
		return false
	}
	if !domainWorkSQLExternalRefsContainAllV0(job.ExternalRefs, filter.ExternalRefs) {
		return false
	}
	return true
}

func domainWorkSQLExternalRefsContainAllV0(
	values []orquestadomainwork.DomainWorkExternalRefV0,
	required []orquestadomainwork.DomainWorkExternalRefV0,
) bool {
	if len(required) == 0 {
		return true
	}
	seen := map[orquestadomainwork.DomainWorkExternalRefV0]struct{}{}
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range required {
		if _, ok := seen[value]; !ok {
			return false
		}
	}
	return true
}

func (store *SQLDomainWorkJobRecordStoreV0) placeholderV0(index int) string {
	switch store.placeholderStyle {
	case SQLDomainWorkPlaceholderDollarV0:
		return "$" + strconv.Itoa(index)
	default:
		return "?"
	}
}

func validDomainWorkSQLPlaceholderStyleV0(style SQLDomainWorkPlaceholderStyleV0) bool {
	switch style {
	case SQLDomainWorkPlaceholderQuestionV0, SQLDomainWorkPlaceholderDollarV0:
		return true
	default:
		return false
	}
}

func domainWorkSQLRequestFingerprintV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
) (string, error) {
	value := domainWorkSQLRequestFingerprintInputV0{
		SchemaVersion:      request.SchemaVersion,
		CorrelationID:      request.CorrelationID,
		IdempotencyKey:     request.IdempotencyKey,
		RequestedBy:        request.RequestedBy,
		DomainRef:          request.DomainRef,
		InterfaceRefs:      append([]string(nil), request.InterfaceRefs...),
		WorkKind:           request.WorkKind,
		WorkRefs:           append([]string(nil), request.WorkRefs...),
		Objective:          request.Objective,
		InputFields:        cloneDomainWorkSQLFieldsV0(request.InputFields),
		InputRefs:          append([]string(nil), request.InputRefs...),
		Constraints:        append([]string(nil), request.Constraints...),
		AcceptanceCriteria: append([]string(nil), request.AcceptanceCriteria...),
		ExternalRefs:       cloneDomainWorkSQLExternalRefsV0(request.ExternalRefs),
		EvidenceRefs:       append([]string(nil), request.EvidenceRefs...),
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type domainWorkSQLRequestFingerprintInputV0 struct {
	SchemaVersion      string                                       `json:"schema_version"`
	CorrelationID      string                                       `json:"correlation_id"`
	IdempotencyKey     string                                       `json:"idempotency_key"`
	RequestedBy        string                                       `json:"requested_by"`
	DomainRef          string                                       `json:"domain_ref"`
	InterfaceRefs      []string                                     `json:"interface_refs"`
	WorkKind           string                                       `json:"work_kind"`
	WorkRefs           []string                                     `json:"work_refs"`
	Objective          string                                       `json:"objective"`
	InputFields        []orquestadomainwork.DomainWorkFieldV0       `json:"input_fields"`
	InputRefs          []string                                     `json:"input_refs"`
	Constraints        []string                                     `json:"constraints"`
	AcceptanceCriteria []string                                     `json:"acceptance_criteria"`
	ExternalRefs       []orquestadomainwork.DomainWorkExternalRefV0 `json:"external_refs"`
	EvidenceRefs       []string                                     `json:"evidence_refs"`
}

func validDomainWorkSQLIdentifierV0(value string) bool {
	if value == "" {
		return false
	}
	for index, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || index > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func domainWorkSQLContextV0(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func domainWorkSQLHashV0(value string) string {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(value))
	return strconv.FormatUint(hash.Sum64(), 36)
}
