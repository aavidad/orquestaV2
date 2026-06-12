package orquestadomainworkfile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

const (
	DomainWorkFileJobCreatorSnapshotSchemaV0 = "domain_work_file_job_creator_snapshot.v0"
	DomainWorkFileArtifactSnapshotSchemaV0   = "domain_work_file_artifact_snapshot.v0"

	ErrDomainWorkFileIdempotencyConflictV0         = "domain_work_file_idempotency_conflict"
	ErrDomainWorkFileArtifactIdempotencyConflictV0 = "domain_work_file_artifact_idempotency_conflict"

	domainWorkFileJobsNameV0      = "domain_work_jobs_v0.json"
	domainWorkFileArtifactsNameV0 = "domain_work_artifacts_v0.json"
)

var _ orquestadomainwork.DomainWorkJobRecordStorePortV0 = (*FileDomainWorkJobCreatorV0)(nil)
var _ orquestadomainwork.DomainWorkArtifactSubmitterPortV0 = (*FileDomainWorkJobCreatorV0)(nil)

type FileDomainWorkJobCreatorV0 struct {
	mu              sync.Mutex
	path            string
	artifactPath    string
	records         map[domainWorkFileJobKeyV0]domainWorkFileJobRecordV0
	artifactRecords map[domainWorkFileArtifactKeyV0]domainWorkFileArtifactRecordV0
	jobsByRef       map[string]domainWorkFileJobKeyV0
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
		path:         filepath.Join(dir, domainWorkFileJobsNameV0),
		artifactPath: filepath.Join(dir, domainWorkFileArtifactsNameV0),
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
		equivalent, err := orquestadomainwork.EquivalentDomainWorkJobRequestsV0(
			existing.Request,
			request,
		)
		if err != nil {
			return orquestadomainwork.DomainWorkJobV0{}, err
		}
		if equivalent {
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

func (creator *FileDomainWorkJobCreatorV0) SubmitDomainWorkArtifactV0(
	ctx context.Context,
	submission orquestadomainwork.DomainWorkArtifactSubmissionV0,
) (orquestadomainwork.DomainWorkArtifactReceiptV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orquestadomainwork.DomainWorkArtifactReceiptV0{}, err
	}
	submission = orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(submission)
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(submission); len(issues) > 0 {
		return invalidDomainWorkFileArtifactReceiptV0(submission, issues), nil
	}
	fingerprint, err := domainWorkFileArtifactSubmissionFingerprintV0(submission)
	if err != nil {
		return orquestadomainwork.DomainWorkArtifactReceiptV0{}, err
	}
	key := domainWorkFileArtifactKeyFromSubmissionV0(submission)

	creator.mu.Lock()
	defer creator.mu.Unlock()
	creator.ensureLockedV0()
	if existing, ok := creator.artifactRecords[key]; ok {
		if existing.Fingerprint == fingerprint {
			return cloneDomainWorkFileArtifactReceiptV0(existing.Receipt), nil
		}
		return invalidDomainWorkFileArtifactReceiptV0(submission, []orquestadomainwork.DomainWorkIssueV0{{
			Code:  ErrDomainWorkFileArtifactIdempotencyConflictV0,
			Field: "idempotency_key",
		}}), nil
	}

	receipt := acceptedDomainWorkFileArtifactReceiptV0(submission)
	next := cloneDomainWorkFileArtifactRecordsMapV0(creator.artifactRecords)
	next[key] = domainWorkFileArtifactRecordV0{
		Submission:  cloneDomainWorkFileArtifactSubmissionV0(submission),
		Receipt:     cloneDomainWorkFileArtifactReceiptV0(receipt),
		Fingerprint: fingerprint,
	}
	if err := ctx.Err(); err != nil {
		return orquestadomainwork.DomainWorkArtifactReceiptV0{}, err
	}
	if err := writeDomainWorkFileArtifactSnapshotV0(creator.artifactPath, next); err != nil {
		return orquestadomainwork.DomainWorkArtifactReceiptV0{}, err
	}
	creator.artifactRecords = next
	return cloneDomainWorkFileArtifactReceiptV0(receipt), nil
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
	artifactRecords, err := loadDomainWorkFileArtifactRecordsV0(creator.artifactPath)
	if err != nil {
		return err
	}
	creator.records = records
	creator.artifactRecords = artifactRecords
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
	if creator.artifactRecords == nil {
		creator.artifactRecords = map[domainWorkFileArtifactKeyV0]domainWorkFileArtifactRecordV0{}
	}
}
