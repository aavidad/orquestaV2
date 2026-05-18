package orquestadomainworksql

import orquestadomainwork "orquesta/modulos/orquesta-domain-work"

func cloneDomainWorkSQLPublicRecordV0(
	record sqlDomainWorkJobRecordV0,
) orquestadomainwork.DomainWorkJobRecordV0 {
	return orquestadomainwork.DomainWorkJobRecordV0{
		Request: cloneDomainWorkSQLJobRequestV0(record.Request),
		Job:     cloneDomainWorkSQLJobV0(record.Job),
	}
}

func cloneDomainWorkSQLJobRequestV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
) orquestadomainwork.DomainWorkJobRequestV0 {
	request.InterfaceRefs = append([]string(nil), request.InterfaceRefs...)
	request.WorkRefs = append([]string(nil), request.WorkRefs...)
	request.InputFields = cloneDomainWorkSQLFieldsV0(request.InputFields)
	request.InputRefs = append([]string(nil), request.InputRefs...)
	request.Constraints = append([]string(nil), request.Constraints...)
	request.AcceptanceCriteria = append([]string(nil), request.AcceptanceCriteria...)
	request.ExternalRefs = cloneDomainWorkSQLExternalRefsV0(request.ExternalRefs)
	request.EvidenceRefs = append([]string(nil), request.EvidenceRefs...)
	return request
}

func cloneDomainWorkSQLJobV0(
	job orquestadomainwork.DomainWorkJobV0,
) orquestadomainwork.DomainWorkJobV0 {
	job.ExternalRefs = cloneDomainWorkSQLExternalRefsV0(job.ExternalRefs)
	job.EvidenceRefs = append([]string(nil), job.EvidenceRefs...)
	job.Issues = cloneDomainWorkSQLIssuesV0(job.Issues)
	return job
}

func cloneDomainWorkSQLFieldsV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	out := make([]orquestadomainwork.DomainWorkFieldV0, len(fields))
	for i, field := range fields {
		out[i] = field
		out[i].Values = append([]string(nil), field.Values...)
		out[i].ValueJSON = append([]byte(nil), field.ValueJSON...)
	}
	if out == nil {
		return []orquestadomainwork.DomainWorkFieldV0{}
	}
	return out
}

func cloneDomainWorkSQLExternalRefsV0(
	refs []orquestadomainwork.DomainWorkExternalRefV0,
) []orquestadomainwork.DomainWorkExternalRefV0 {
	if refs == nil {
		return []orquestadomainwork.DomainWorkExternalRefV0{}
	}
	return append([]orquestadomainwork.DomainWorkExternalRefV0(nil), refs...)
}

func cloneDomainWorkSQLIssuesV0(
	issues []orquestadomainwork.DomainWorkIssueV0,
) []orquestadomainwork.DomainWorkIssueV0 {
	if issues == nil {
		return []orquestadomainwork.DomainWorkIssueV0{}
	}
	return append([]orquestadomainwork.DomainWorkIssueV0(nil), issues...)
}
