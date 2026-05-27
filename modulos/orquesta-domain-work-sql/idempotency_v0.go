package orquestadomainworksql

import (
	"context"
	"strconv"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func (store *SQLDomainWorkJobRecordStoreV0) jobAfterUniqueViolationV0(
	ctx context.Context,
	request orquestadomainwork.DomainWorkJobRequestV0,
) (orquestadomainwork.DomainWorkJobV0, bool, error) {
	existing, found, err := store.findByKeyV0(ctx, request.DomainRef, request.IdempotencyKey)
	if err != nil {
		return orquestadomainwork.DomainWorkJobV0{}, false, err
	}
	if found {
		equivalent, err := orquestadomainwork.EquivalentDomainWorkJobRequestsV0(
			existing.Request,
			request,
		)
		if err != nil {
			return orquestadomainwork.DomainWorkJobV0{}, false, err
		}
		if equivalent {
			return cloneDomainWorkSQLJobV0(existing.Job), true, nil
		}
		return invalidDomainWorkSQLJobV0(request, []orquestadomainwork.DomainWorkIssueV0{{
			Code:  ErrDomainWorkSQLIdempotencyConflictV0,
			Field: "idempotency_key",
		}}), true, nil
	}
	return orquestadomainwork.DomainWorkJobV0{}, false, nil
}

func (store *SQLDomainWorkJobRecordStoreV0) uniqueViolationV0(err error) bool {
	return err != nil && store.isUniqueViolation != nil && store.isUniqueViolation(err)
}

func (store *SQLDomainWorkJobRecordStoreV0) nextJobRefV0(
	ctx context.Context,
	domainRef string,
	idempotencyKey string,
) (string, error) {
	identity, err := orquestadomainwork.BuildDomainWorkJobIdentityV0(
		orquestadomainwork.DomainWorkJobRequestV0{
			DomainRef:      domainRef,
			IdempotencyKey: idempotencyKey,
		},
	)
	if err != nil {
		return "", err
	}
	base := identity.JobRefBase
	jobRef := base
	for index := 2; ; index++ {
		exists, err := store.jobRefExistsV0(ctx, jobRef)
		if err != nil {
			return "", err
		}
		if !exists {
			return jobRef, nil
		}
		jobRef = domainWorkSQLCollisionJobRefV0(base, index)
	}
}

func domainWorkSQLCollisionJobRefV0(base string, index int) string {
	return base + "-collision-" + strconv.Itoa(index)
}
