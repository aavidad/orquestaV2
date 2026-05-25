package orquestadomainworkfile

import orquestadomainwork "orquesta/modulos/orquesta-domain-work"

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
