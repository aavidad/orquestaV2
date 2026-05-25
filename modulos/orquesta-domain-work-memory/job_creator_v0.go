package orquestadomainworkmemory

import (
	"context"
	"sort"
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
