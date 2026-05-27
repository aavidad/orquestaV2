package orquestadomainworkmemory

import (
	"strconv"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

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
	return domainWorkMemoryExternalRefsContainAllV0(record.Job.ExternalRefs, filter.ExternalRefs)
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
	identity, err := orquestadomainwork.BuildDomainWorkJobIdentityV0(
		orquestadomainwork.DomainWorkJobRequestV0{
			DomainRef:      key.DomainRef,
			IdempotencyKey: key.IdempotencyKey,
		},
	)
	if err != nil {
		return ""
	}
	base := identity.JobRefBase
	jobRef := base
	for index := 2; ; index++ {
		existing, ok := creator.jobsByRef[jobRef]
		if !ok || existing == key {
			return jobRef
		}
		jobRef = domainWorkMemoryCollisionJobRefV0(base, index)
	}
}

func domainWorkMemoryCollisionJobRefV0(base string, index int) string {
	return base + "-collision-" + strconv.Itoa(index)
}
