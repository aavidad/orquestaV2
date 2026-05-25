package orquestadomainworksql

import (
	"context"
	"fmt"
	"strconv"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

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
