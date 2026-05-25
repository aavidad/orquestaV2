package orquestadomainworkfile

import (
	"strconv"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

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
	return domainWorkFileExternalRefsContainAllV0(record.Job.ExternalRefs, filter.ExternalRefs)
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
