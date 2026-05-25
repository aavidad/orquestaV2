package orquestadomainworksql

import (
	"encoding/json"
	"fmt"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

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
	return domainWorkSQLExternalRefsContainAllV0(job.ExternalRefs, filter.ExternalRefs)
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
