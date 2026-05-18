package orquestadomainworkmemory

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"sort"
	"strconv"
	"sync"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	ErrDomainWorkMemoryIdempotencyConflictV0 = "domain_work_memory_idempotency_conflict"
)

var _ orquestadomainwork.DomainWorkJobRecordStorePortV0 = (*InMemoryDomainWorkJobCreatorV0)(nil)

type InMemoryDomainWorkJobCreatorV0 struct {
	mu        sync.Mutex
	records   map[domainWorkMemoryJobKeyV0]domainWorkMemoryJobRecordV0
	jobsByRef map[string]domainWorkMemoryJobKeyV0
}

func NewInMemoryDomainWorkJobCreatorV0() *InMemoryDomainWorkJobCreatorV0 {
	return &InMemoryDomainWorkJobCreatorV0{
		records:   map[domainWorkMemoryJobKeyV0]domainWorkMemoryJobRecordV0{},
		jobsByRef: map[string]domainWorkMemoryJobKeyV0{},
	}
}

func (creator *InMemoryDomainWorkJobCreatorV0) CreateDomainWorkJobV0(
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
		return invalidDomainWorkMemoryJobV0(request, issues), nil
	}
	fingerprint, err := domainWorkMemoryRequestFingerprintV0(request)
	if err != nil {
		return orquestadomainwork.DomainWorkJobV0{}, err
	}
	key := domainWorkMemoryJobKeyFromRequestV0(request)

	creator.mu.Lock()
	defer creator.mu.Unlock()
	creator.ensureLockedV0()
	if existing, ok := creator.records[key]; ok {
		if existing.Fingerprint == fingerprint {
			return cloneDomainWorkMemoryJobV0(existing.Job), nil
		}
		return invalidDomainWorkMemoryJobV0(request, []orquestadomainwork.DomainWorkIssueV0{{
			Code:  ErrDomainWorkMemoryIdempotencyConflictV0,
			Field: "idempotency_key",
		}}), nil
	}

	job := acceptedDomainWorkMemoryJobV0(request, creator.nextJobRefLockedV0(key))
	creator.records[key] = domainWorkMemoryJobRecordV0{
		Request:     cloneDomainWorkMemoryJobRequestV0(request),
		Job:         cloneDomainWorkMemoryJobV0(job),
		Fingerprint: fingerprint,
	}
	creator.jobsByRef[job.JobRef] = key
	return cloneDomainWorkMemoryJobV0(job), nil
}

func (creator *InMemoryDomainWorkJobCreatorV0) ListDomainWorkJobsV0(
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
		jobs = append(jobs, cloneDomainWorkMemoryJobV0(creator.records[key].Job))
	}
	return jobs, nil
}

func (creator *InMemoryDomainWorkJobCreatorV0) ListDomainWorkJobRecordsV0(
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
		if !domainWorkMemoryJobRecordMatchesFilterV0(key, record, filter) {
			continue
		}
		records = append(records, cloneDomainWorkMemoryJobRecordV0(record))
		if filter.Limit > 0 && len(records) >= filter.Limit {
			break
		}
	}
	return records, nil
}

func (creator *InMemoryDomainWorkJobCreatorV0) ensureLockedV0() {
	if creator.records == nil {
		creator.records = map[domainWorkMemoryJobKeyV0]domainWorkMemoryJobRecordV0{}
	}
	if creator.jobsByRef == nil {
		creator.jobsByRef = map[string]domainWorkMemoryJobKeyV0{}
		for key, record := range creator.records {
			if record.Job.JobRef != "" {
				creator.jobsByRef[record.Job.JobRef] = key
			}
		}
	}
}

func domainWorkMemoryJobRecordMatchesFilterV0(
	key domainWorkMemoryJobKeyV0,
	record domainWorkMemoryJobRecordV0,
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
	if !domainWorkMemoryExternalRefsContainAllV0(record.Job.ExternalRefs, filter.ExternalRefs) {
		return false
	}
	return true
}

func domainWorkMemoryExternalRefsContainAllV0(
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

func (creator *InMemoryDomainWorkJobCreatorV0) nextJobRefLockedV0(
	key domainWorkMemoryJobKeyV0,
) string {
	base := "domain-work-job-" + domainWorkMemoryHashV0(
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

type domainWorkMemoryJobKeyV0 struct {
	DomainRef      string
	IdempotencyKey string
}

type domainWorkMemoryJobRecordV0 struct {
	Request     orquestadomainwork.DomainWorkJobRequestV0
	Job         orquestadomainwork.DomainWorkJobV0
	Fingerprint string
}

func domainWorkMemoryJobKeyFromRequestV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
) domainWorkMemoryJobKeyV0 {
	return domainWorkMemoryJobKeyV0{
		DomainRef:      request.DomainRef,
		IdempotencyKey: request.IdempotencyKey,
	}
}

func acceptedDomainWorkMemoryJobV0(
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
		ExternalRefs:   cloneDomainWorkMemoryExternalRefsV0(request.ExternalRefs),
		EvidenceRefs:   append([]string(nil), request.EvidenceRefs...),
	}
}

func invalidDomainWorkMemoryJobV0(
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
		ExternalRefs:   cloneDomainWorkMemoryExternalRefsV0(request.ExternalRefs),
		EvidenceRefs:   append([]string(nil), request.EvidenceRefs...),
		Issues:         cloneDomainWorkMemoryIssuesV0(issues),
	}
}

func domainWorkMemoryRequestFingerprintV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
) (string, error) {
	value := domainWorkMemoryRequestFingerprintInputV0{
		SchemaVersion:      request.SchemaVersion,
		CorrelationID:      request.CorrelationID,
		IdempotencyKey:     request.IdempotencyKey,
		RequestedBy:        request.RequestedBy,
		DomainRef:          request.DomainRef,
		InterfaceRefs:      append([]string(nil), request.InterfaceRefs...),
		WorkKind:           request.WorkKind,
		WorkRefs:           append([]string(nil), request.WorkRefs...),
		Objective:          request.Objective,
		InputFields:        cloneDomainWorkMemoryFieldsV0(request.InputFields),
		InputRefs:          append([]string(nil), request.InputRefs...),
		Constraints:        append([]string(nil), request.Constraints...),
		AcceptanceCriteria: append([]string(nil), request.AcceptanceCriteria...),
		ExternalRefs:       cloneDomainWorkMemoryExternalRefsV0(request.ExternalRefs),
		EvidenceRefs:       append([]string(nil), request.EvidenceRefs...),
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type domainWorkMemoryRequestFingerprintInputV0 struct {
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

func domainWorkMemoryHashV0(value string) string {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(value))
	return strconv.FormatUint(hash.Sum64(), 36)
}

func cloneDomainWorkMemoryJobRecordV0(
	record domainWorkMemoryJobRecordV0,
) orquestadomainwork.DomainWorkJobRecordV0 {
	return orquestadomainwork.DomainWorkJobRecordV0{
		Request: cloneDomainWorkMemoryJobRequestV0(record.Request),
		Job:     cloneDomainWorkMemoryJobV0(record.Job),
	}
}

func cloneDomainWorkMemoryJobRequestV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
) orquestadomainwork.DomainWorkJobRequestV0 {
	request.InterfaceRefs = append([]string(nil), request.InterfaceRefs...)
	request.WorkRefs = append([]string(nil), request.WorkRefs...)
	request.InputFields = cloneDomainWorkMemoryFieldsV0(request.InputFields)
	request.InputRefs = append([]string(nil), request.InputRefs...)
	request.Constraints = append([]string(nil), request.Constraints...)
	request.AcceptanceCriteria = append([]string(nil), request.AcceptanceCriteria...)
	request.ExternalRefs = cloneDomainWorkMemoryExternalRefsV0(request.ExternalRefs)
	request.EvidenceRefs = append([]string(nil), request.EvidenceRefs...)
	return request
}

func cloneDomainWorkMemoryJobV0(
	job orquestadomainwork.DomainWorkJobV0,
) orquestadomainwork.DomainWorkJobV0 {
	job.ExternalRefs = cloneDomainWorkMemoryExternalRefsV0(job.ExternalRefs)
	job.EvidenceRefs = append([]string(nil), job.EvidenceRefs...)
	job.Issues = cloneDomainWorkMemoryIssuesV0(job.Issues)
	return job
}

func cloneDomainWorkMemoryFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	out := make([]orquestadomainwork.DomainWorkFieldV0, len(fields))
	for i, field := range fields {
		out[i] = field
		out[i].Values = append([]string(nil), field.Values...)
		out[i].ValueJSON = append([]byte(nil), field.ValueJSON...)
	}
	if out == nil {
		return []orquestadomainwork.DomainWorkFieldV0{}
	}
	return out
}

func cloneDomainWorkMemoryExternalRefsV0(
	refs []orquestadomainwork.DomainWorkExternalRefV0,
) []orquestadomainwork.DomainWorkExternalRefV0 {
	if refs == nil {
		return []orquestadomainwork.DomainWorkExternalRefV0{}
	}
	return append([]orquestadomainwork.DomainWorkExternalRefV0(nil), refs...)
}

func cloneDomainWorkMemoryIssuesV0(
	issues []orquestadomainwork.DomainWorkIssueV0,
) []orquestadomainwork.DomainWorkIssueV0 {
	if issues == nil {
		return []orquestadomainwork.DomainWorkIssueV0{}
	}
	return append([]orquestadomainwork.DomainWorkIssueV0(nil), issues...)
}
