package orquestadomainworkfile

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	DomainWorkFileJobCreatorSnapshotSchemaV0 = "domain_work_file_job_creator_snapshot.v0"

	ErrDomainWorkFileIdempotencyConflictV0 = "domain_work_file_idempotency_conflict"

	domainWorkFileJobsNameV0 = "domain_work_jobs_v0.json"
)

var _ orquestadomainwork.DomainWorkJobRecordStorePortV0 = (*FileDomainWorkJobCreatorV0)(nil)

type FileDomainWorkJobCreatorV0 struct {
	mu        sync.Mutex
	path      string
	records   map[domainWorkFileJobKeyV0]domainWorkFileJobRecordV0
	jobsByRef map[string]domainWorkFileJobKeyV0
}

func NewFileDomainWorkJobCreatorV0(dir string) (*FileDomainWorkJobCreatorV0, error) {
	dir, err := normalizeDomainWorkFileDirV0(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("orquesta_domain_work_file: dir_unavailable")
	}
	creator := &FileDomainWorkJobCreatorV0{
		path: filepath.Join(dir, domainWorkFileJobsNameV0),
	}
	if err := creator.loadV0(); err != nil {
		return nil, err
	}
	return creator, nil
}

func (creator *FileDomainWorkJobCreatorV0) CreateDomainWorkJobV0(
	ctx context.Context,
	request orquestadomainwork.DomainWorkJobRequestV0,
) (orquestadomainwork.DomainWorkJobV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orquestadomainwork.DomainWorkJobV0{}, err
	}
	request = orquestadomainwork.NormalizeDomainWorkJobRequestV0(request)
	if issues := orquestadomainwork.ValidateDomainWorkJobRequestV0(request); len(issues) > 0 {
		return invalidDomainWorkFileJobV0(request, issues), nil
	}
	fingerprint, err := domainWorkFileRequestFingerprintV0(request)
	if err != nil {
		return orquestadomainwork.DomainWorkJobV0{}, err
	}
	key := domainWorkFileJobKeyFromRequestV0(request)

	creator.mu.Lock()
	defer creator.mu.Unlock()
	creator.ensureLockedV0()
	if existing, ok := creator.records[key]; ok {
		if existing.Fingerprint == fingerprint {
			return cloneDomainWorkFileJobV0(existing.Job), nil
		}
		return invalidDomainWorkFileJobV0(request, []orquestadomainwork.DomainWorkIssueV0{{
			Code:  ErrDomainWorkFileIdempotencyConflictV0,
			Field: "idempotency_key",
		}}), nil
	}

	job := acceptedDomainWorkFileJobV0(request, creator.nextJobRefLockedV0(key))
	nextRecords := cloneDomainWorkFileRecordsMapV0(creator.records)
	nextJobsByRef := cloneDomainWorkFileJobsByRefV0(creator.jobsByRef)
	nextRecords[key] = domainWorkFileJobRecordV0{
		Request:     cloneDomainWorkFileJobRequestV0(request),
		Job:         cloneDomainWorkFileJobV0(job),
		Fingerprint: fingerprint,
	}
	nextJobsByRef[job.JobRef] = key
	if err := ctx.Err(); err != nil {
		return orquestadomainwork.DomainWorkJobV0{}, err
	}
	if err := writeDomainWorkFileSnapshotV0(creator.path, nextRecords); err != nil {
		return orquestadomainwork.DomainWorkJobV0{}, err
	}
	creator.records = nextRecords
	creator.jobsByRef = nextJobsByRef
	return cloneDomainWorkFileJobV0(job), nil
}

func (creator *FileDomainWorkJobCreatorV0) ListDomainWorkJobsV0(
	ctx context.Context,
) ([]orquestadomainwork.DomainWorkJobV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	creator.mu.Lock()
	defer creator.mu.Unlock()
	creator.ensureLockedV0()
	refs := make([]string, 0, len(creator.jobsByRef))
	for ref := range creator.jobsByRef {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	jobs := make([]orquestadomainwork.DomainWorkJobV0, 0, len(refs))
	for _, ref := range refs {
		key := creator.jobsByRef[ref]
		jobs = append(jobs, cloneDomainWorkFileJobV0(creator.records[key].Job))
	}
	return jobs, nil
}

func (creator *FileDomainWorkJobCreatorV0) ListDomainWorkJobRecordsV0(
	ctx context.Context,
	filter orquestadomainwork.DomainWorkJobRecordFilterV0,
) ([]orquestadomainwork.DomainWorkJobRecordV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	filter = orquestadomainwork.NormalizeDomainWorkJobRecordFilterV0(filter)
	creator.mu.Lock()
	defer creator.mu.Unlock()
	creator.ensureLockedV0()
	refs := make([]string, 0, len(creator.jobsByRef))
	for ref := range creator.jobsByRef {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	records := make([]orquestadomainwork.DomainWorkJobRecordV0, 0, len(refs))
	for _, ref := range refs {
		key := creator.jobsByRef[ref]
		record := creator.records[key]
		if !domainWorkFileJobRecordMatchesFilterV0(key, record, filter) {
			continue
		}
		records = append(records, cloneDomainWorkFileJobRecordV0(record))
		if filter.Limit > 0 && len(records) >= filter.Limit {
			break
		}
	}
	return records, nil
}

func (creator *FileDomainWorkJobCreatorV0) SnapshotPathV0() string {
	if creator == nil {
		return ""
	}
	return creator.path
}

func (creator *FileDomainWorkJobCreatorV0) loadV0() error {
	records, err := loadDomainWorkFileRecordsV0(creator.path)
	if err != nil {
		return err
	}
	creator.records = records
	creator.jobsByRef = domainWorkFileJobsByRefFromRecordsV0(records)
	return nil
}

func (creator *FileDomainWorkJobCreatorV0) ensureLockedV0() {
	if creator.records == nil {
		creator.records = map[domainWorkFileJobKeyV0]domainWorkFileJobRecordV0{}
	}
	if creator.jobsByRef == nil {
		creator.jobsByRef = domainWorkFileJobsByRefFromRecordsV0(creator.records)
	}
}

func domainWorkFileJobRecordMatchesFilterV0(
	key domainWorkFileJobKeyV0,
	record domainWorkFileJobRecordV0,
	filter orquestadomainwork.DomainWorkJobRecordFilterV0,
) bool {
	if filter.DomainRef != "" && key.DomainRef != filter.DomainRef {
		return false
	}
	if filter.IdempotencyKey != "" && key.IdempotencyKey != filter.IdempotencyKey {
		return false
	}
	if filter.WorkKind != "" && record.Job.WorkKind != filter.WorkKind {
		return false
	}
	if filter.JobRef != "" && record.Job.JobRef != filter.JobRef {
		return false
	}
	if filter.CorrelationID != "" && record.Job.CorrelationID != filter.CorrelationID {
		return false
	}
	if filter.Status != "" && record.Job.Status != filter.Status {
		return false
	}
	if !domainWorkFileExternalRefsContainAllV0(record.Job.ExternalRefs, filter.ExternalRefs) {
		return false
	}
	return true
}

func domainWorkFileExternalRefsContainAllV0(
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

func (creator *FileDomainWorkJobCreatorV0) nextJobRefLockedV0(
	key domainWorkFileJobKeyV0,
) string {
	base := "domain-work-job-" + domainWorkFileHashV0(
		key.DomainRef+"\x00"+key.IdempotencyKey,
	)
	jobRef := base
	for index := 2; ; index++ {
		existing, ok := creator.jobsByRef[jobRef]
		if !ok || existing == key {
			return jobRef
		}
		jobRef = base + "-" + strconv.Itoa(index)
	}
}

func normalizeDomainWorkFileDirV0(dir string) (string, error) {
	dir = filepath.Clean(strings.TrimSpace(dir))
	if dir == "." || dir == "" || !filepath.IsAbs(dir) {
		return "", fmt.Errorf("orquesta_domain_work_file: dir_invalid")
	}
	return dir, nil
}

type domainWorkFileJobKeyV0 struct {
	DomainRef      string
	IdempotencyKey string
}

type domainWorkFileJobRecordV0 struct {
	Request     orquestadomainwork.DomainWorkJobRequestV0
	Job         orquestadomainwork.DomainWorkJobV0
	Fingerprint string
}

func domainWorkFileJobKeyFromRequestV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
) domainWorkFileJobKeyV0 {
	return domainWorkFileJobKeyV0{
		DomainRef:      request.DomainRef,
		IdempotencyKey: request.IdempotencyKey,
	}
}

func acceptedDomainWorkFileJobV0(
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
		ExternalRefs:   cloneDomainWorkFileExternalRefsV0(request.ExternalRefs),
		EvidenceRefs:   append([]string(nil), request.EvidenceRefs...),
	}
}

func invalidDomainWorkFileJobV0(
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
		ExternalRefs:   cloneDomainWorkFileExternalRefsV0(request.ExternalRefs),
		EvidenceRefs:   append([]string(nil), request.EvidenceRefs...),
		Issues:         cloneDomainWorkFileIssuesV0(issues),
	}
}

func domainWorkFileRequestFingerprintV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
) (string, error) {
	value := domainWorkFileRequestFingerprintInputV0{
		SchemaVersion:      request.SchemaVersion,
		CorrelationID:      request.CorrelationID,
		IdempotencyKey:     request.IdempotencyKey,
		RequestedBy:        request.RequestedBy,
		DomainRef:          request.DomainRef,
		InterfaceRefs:      append([]string(nil), request.InterfaceRefs...),
		WorkKind:           request.WorkKind,
		WorkRefs:           append([]string(nil), request.WorkRefs...),
		Objective:          request.Objective,
		InputFields:        cloneDomainWorkFileFieldsV0(request.InputFields),
		InputRefs:          append([]string(nil), request.InputRefs...),
		Constraints:        append([]string(nil), request.Constraints...),
		AcceptanceCriteria: append([]string(nil), request.AcceptanceCriteria...),
		ExternalRefs:       cloneDomainWorkFileExternalRefsV0(request.ExternalRefs),
		EvidenceRefs:       append([]string(nil), request.EvidenceRefs...),
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type domainWorkFileRequestFingerprintInputV0 struct {
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

func domainWorkFileHashV0(value string) string {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(value))
	return strconv.FormatUint(hash.Sum64(), 36)
}
