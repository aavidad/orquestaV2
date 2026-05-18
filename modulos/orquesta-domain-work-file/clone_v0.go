package orquestadomainworkfile

import orquestadomainwork "orquesta/modulos/orquesta-domain-work"

func cloneDomainWorkFileRecordsMapV0(
	records map[domainWorkFileJobKeyV0]domainWorkFileJobRecordV0,
) map[domainWorkFileJobKeyV0]domainWorkFileJobRecordV0 {
	out := make(map[domainWorkFileJobKeyV0]domainWorkFileJobRecordV0, len(records))
	for key, record := range records {
		out[key] = domainWorkFileJobRecordV0{
			Request:     cloneDomainWorkFileJobRequestV0(record.Request),
			Job:         cloneDomainWorkFileJobV0(record.Job),
			Fingerprint: record.Fingerprint,
		}
	}
	return out
}

func cloneDomainWorkFileJobsByRefV0(
	jobsByRef map[string]domainWorkFileJobKeyV0,
) map[string]domainWorkFileJobKeyV0 {
	out := make(map[string]domainWorkFileJobKeyV0, len(jobsByRef))
	for ref, key := range jobsByRef {
		out[ref] = key
	}
	return out
}

func cloneDomainWorkFileJobRecordV0(
	record domainWorkFileJobRecordV0,
) orquestadomainwork.DomainWorkJobRecordV0 {
	return orquestadomainwork.DomainWorkJobRecordV0{
		Request: cloneDomainWorkFileJobRequestV0(record.Request),
		Job:     cloneDomainWorkFileJobV0(record.Job),
	}
}

func domainWorkFileJobsByRefFromRecordsV0(
	records map[domainWorkFileJobKeyV0]domainWorkFileJobRecordV0,
) map[string]domainWorkFileJobKeyV0 {
	out := make(map[string]domainWorkFileJobKeyV0, len(records))
	for key, record := range records {
		if record.Job.JobRef != "" {
			out[record.Job.JobRef] = key
		}
	}
	return out
}

func cloneDomainWorkFileJobRequestV0(
	request orquestadomainwork.DomainWorkJobRequestV0,
) orquestadomainwork.DomainWorkJobRequestV0 {
	request.InterfaceRefs = append([]string(nil), request.InterfaceRefs...)
	request.WorkRefs = append([]string(nil), request.WorkRefs...)
	request.InputFields = cloneDomainWorkFileFieldsV0(request.InputFields)
	request.InputRefs = append([]string(nil), request.InputRefs...)
	request.Constraints = append([]string(nil), request.Constraints...)
	request.AcceptanceCriteria = append([]string(nil), request.AcceptanceCriteria...)
	request.ExternalRefs = cloneDomainWorkFileExternalRefsV0(request.ExternalRefs)
	request.EvidenceRefs = append([]string(nil), request.EvidenceRefs...)
	return request
}

func cloneDomainWorkFileJobV0(
	job orquestadomainwork.DomainWorkJobV0,
) orquestadomainwork.DomainWorkJobV0 {
	job.ExternalRefs = cloneDomainWorkFileExternalRefsV0(job.ExternalRefs)
	job.EvidenceRefs = append([]string(nil), job.EvidenceRefs...)
	job.Issues = cloneDomainWorkFileIssuesV0(job.Issues)
	return job
}

func cloneDomainWorkFileFieldsV0(
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

func cloneDomainWorkFileExternalRefsV0(
	refs []orquestadomainwork.DomainWorkExternalRefV0,
) []orquestadomainwork.DomainWorkExternalRefV0 {
	if refs == nil {
		return []orquestadomainwork.DomainWorkExternalRefV0{}
	}
	return append([]orquestadomainwork.DomainWorkExternalRefV0(nil), refs...)
}

func cloneDomainWorkFileIssuesV0(
	issues []orquestadomainwork.DomainWorkIssueV0,
) []orquestadomainwork.DomainWorkIssueV0 {
	if issues == nil {
		return []orquestadomainwork.DomainWorkIssueV0{}
	}
	return append([]orquestadomainwork.DomainWorkIssueV0(nil), issues...)
}
