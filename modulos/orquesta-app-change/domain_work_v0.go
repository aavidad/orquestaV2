package orquestaappchange

import orquestadomainwork "orquesta/modulos/orquesta-domain-work"

func DomainWorkJobRequestFromAppChangeV0(
	request AppChangeRequestV0,
) (orquestadomainwork.DomainWorkJobRequestV0, bool) {
	request = normalizeAppChangeRequestV0(request)
	if request.ExternalWork == nil {
		return orquestadomainwork.DomainWorkJobRequestV0{}, false
	}
	work := request.ExternalWork
	return orquestadomainwork.NormalizeDomainWorkJobRequestV0(
		orquestadomainwork.DomainWorkJobRequestV0{
			RequestID:          request.RequestID,
			CorrelationID:      request.CorrelationID,
			IdempotencyKey:     request.ChangeRef,
			RequestedBy:        AppChangeDefaultRequestedByV0,
			DomainRef:          work.ProjectRef,
			InterfaceRefs:      append([]string(nil), work.InterfaceRefs...),
			WorkKind:           work.WorkKind,
			WorkRefs:           append([]string(nil), work.WorkRefs...),
			Objective:          request.UserIntent,
			InputRefs:          append([]string(nil), request.CurrentStateRefs...),
			Constraints:        append([]string(nil), request.Constraints...),
			AcceptanceCriteria: append([]string(nil), request.AcceptanceCriteria...),
			ExternalRefs:       domainWorkExternalRefsFromAppChangeV0(request),
			EvidenceRefs:       append([]string(nil), request.MetadataRefs...),
		},
	), true
}

func domainWorkExternalRefsFromAppChangeV0(
	request AppChangeRequestV0,
) []orquestadomainwork.DomainWorkExternalRefV0 {
	refs := make([]orquestadomainwork.DomainWorkExternalRefV0, 0, 4)
	refs = appendDomainWorkExternalRefV0(refs, "run_ref", request.RunRef)
	refs = appendDomainWorkExternalRefV0(refs, "app_ref", request.AppRef)
	refs = appendDomainWorkExternalRefV0(refs, "change_ref", request.ChangeRef)
	refs = appendDomainWorkExternalRefV0(refs, "actor_ref", request.ActorRef)
	return refs
}

func appendDomainWorkExternalRefV0(
	refs []orquestadomainwork.DomainWorkExternalRefV0,
	kind string,
	ref string,
) []orquestadomainwork.DomainWorkExternalRefV0 {
	if ref == "" {
		return refs
	}
	return append(refs, orquestadomainwork.DomainWorkExternalRefV0{
		Kind: kind,
		Ref:  ref,
	})
}
