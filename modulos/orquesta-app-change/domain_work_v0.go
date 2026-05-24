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
			WorkRefs:           domainWorkRefsFromExternalWorkV0(work),
			Objective:          request.UserIntent,
			InputFields:        copyDomainWorkFieldsFromAppChangeV0(work.InputFields),
			InputRefs:          append([]string(nil), request.CurrentStateRefs...),
			Constraints:        append([]string(nil), request.Constraints...),
			AcceptanceCriteria: append([]string(nil), request.AcceptanceCriteria...),
			RequiredTests:      copyDomainWorkRequiredTestsFromAppChangeV0(work.RequiredTests),
			ExternalRefs:       domainWorkExternalRefsFromAppChangeV0(request),
			EvidenceRefs:       append([]string(nil), request.MetadataRefs...),
		},
	), true
}

func domainWorkRefsFromExternalWorkV0(
	work *AppChangeExternalWorkV0,
) []string {
	if work == nil {
		return []string{}
	}
	refs := make([]string, 0, len(work.WorkRefs)+1)
	if work.JobRef != "" {
		refs = append(refs, work.JobRef)
	}
	refs = append(refs, work.WorkRefs...)
	return compactAppChangeStringsV0(refs)
}

func copyDomainWorkFieldsFromAppChangeV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	out := make([]orquestadomainwork.DomainWorkFieldV0, 0, len(fields))
	for _, field := range fields {
		out = append(out, orquestadomainwork.DomainWorkFieldV0{
			Name:      field.Name,
			Value:     field.Value,
			Values:    append([]string(nil), field.Values...),
			ValueJSON: append([]byte(nil), field.ValueJSON...),
		})
	}
	if out == nil {
		return []orquestadomainwork.DomainWorkFieldV0{}
	}
	return out
}

func copyDomainWorkRequiredTestsFromAppChangeV0(
	tests []orquestadomainwork.DomainWorkRequiredTestV0,
) []orquestadomainwork.DomainWorkRequiredTestV0 {
	out := make([]orquestadomainwork.DomainWorkRequiredTestV0, 0, len(tests))
	for _, test := range tests {
		out = append(out, orquestadomainwork.DomainWorkRequiredTestV0{
			TestRef:                test.TestRef,
			AcceptanceCriteria:     append([]string(nil), test.AcceptanceCriteria...),
			AcceptanceCriteriaRefs: append([]string(nil), test.AcceptanceCriteriaRefs...),
			InputRefs:              append([]string(nil), test.InputRefs...),
			ExternalRefs:           append([]orquestadomainwork.DomainWorkExternalRefV0(nil), test.ExternalRefs...),
			EvidenceRefs:           append([]string(nil), test.EvidenceRefs...),
		})
	}
	if out == nil {
		return []orquestadomainwork.DomainWorkRequiredTestV0{}
	}
	return out
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
