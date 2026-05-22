package orquestadomainwork

import (
	"bytes"
	"encoding/json"
	"strings"
)

func NormalizeDomainWorkJobRequestV0(
	request DomainWorkJobRequestV0,
) DomainWorkJobRequestV0 {
	request.SchemaVersion = defaultDomainWorkSchemaV0(
		request.SchemaVersion,
		DomainWorkJobRequestSchemaV0,
	)
	request.RequestID = strings.TrimSpace(request.RequestID)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	request.RequestedBy = defaultDomainWorkRequestedByV0(request.RequestedBy)
	request.DomainRef = strings.TrimSpace(request.DomainRef)
	request.InterfaceRefs = compactDomainWorkStringsV0(request.InterfaceRefs)
	request.WorkKind = strings.TrimSpace(request.WorkKind)
	request.WorkRefs = compactDomainWorkStringsV0(request.WorkRefs)
	request.Objective = strings.TrimSpace(request.Objective)
	request.InputFields = compactDomainWorkFieldsV0(request.InputFields)
	request.InputRefs = compactDomainWorkStringsV0(request.InputRefs)
	request.Constraints = compactDomainWorkStringsV0(request.Constraints)
	request.AcceptanceCriteria = compactDomainWorkStringsV0(request.AcceptanceCriteria)
	request.RequiredTests = compactDomainWorkRequiredTestsV0(request.RequiredTests)
	request.ExternalRefs = compactDomainWorkExternalRefsV0(request.ExternalRefs)
	request.EvidenceRefs = compactDomainWorkStringsV0(request.EvidenceRefs)
	if request.RequestID == "" {
		request.RequestID = request.IdempotencyKey
	}
	if request.CorrelationID == "" {
		request.CorrelationID = request.RequestID
	}
	if request.IdempotencyKey == "" {
		request.IdempotencyKey = request.RequestID
	}
	return request
}

func NormalizeDomainWorkArtifactSubmissionV0(
	submission DomainWorkArtifactSubmissionV0,
) DomainWorkArtifactSubmissionV0 {
	submission.SchemaVersion = defaultDomainWorkSchemaV0(
		submission.SchemaVersion,
		DomainWorkArtifactSubmissionSchemaV0,
	)
	submission.RequestID = strings.TrimSpace(submission.RequestID)
	submission.CorrelationID = strings.TrimSpace(submission.CorrelationID)
	submission.IdempotencyKey = strings.TrimSpace(submission.IdempotencyKey)
	submission.RequestedBy = defaultDomainWorkRequestedByV0(submission.RequestedBy)
	submission.DomainRef = strings.TrimSpace(submission.DomainRef)
	submission.JobRef = strings.TrimSpace(submission.JobRef)
	submission.ArtifactRef = strings.TrimSpace(submission.ArtifactRef)
	submission.ArtifactType = strings.TrimSpace(submission.ArtifactType)
	submission.Summary = strings.TrimSpace(submission.Summary)
	submission.PayloadFields = compactDomainWorkFieldsV0(submission.PayloadFields)
	submission.PayloadRefs = compactDomainWorkStringsV0(submission.PayloadRefs)
	submission.ExternalRefs = compactDomainWorkExternalRefsV0(submission.ExternalRefs)
	submission.EvidenceRefs = compactDomainWorkStringsV0(submission.EvidenceRefs)
	if submission.RequestID == "" {
		submission.RequestID = submission.IdempotencyKey
	}
	if submission.CorrelationID == "" {
		submission.CorrelationID = submission.RequestID
	}
	if submission.IdempotencyKey == "" {
		submission.IdempotencyKey = submission.RequestID
	}
	return submission
}

func NormalizeDomainWorkJobRecordFilterV0(
	filter DomainWorkJobRecordFilterV0,
) DomainWorkJobRecordFilterV0 {
	filter.DomainRef = strings.TrimSpace(filter.DomainRef)
	filter.WorkKind = strings.TrimSpace(filter.WorkKind)
	filter.JobRef = strings.TrimSpace(filter.JobRef)
	filter.CorrelationID = strings.TrimSpace(filter.CorrelationID)
	filter.IdempotencyKey = strings.TrimSpace(filter.IdempotencyKey)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.ExternalRefs = compactDomainWorkExternalRefsV0(filter.ExternalRefs)
	if filter.Limit < 0 {
		filter.Limit = 0
	}
	return filter
}

func NormalizeDomainWorkRequiredTestPlanV0(
	plan DomainWorkRequiredTestPlanV0,
) DomainWorkRequiredTestPlanV0 {
	plan.SchemaVersion = defaultDomainWorkSchemaV0(
		plan.SchemaVersion,
		DomainWorkRequiredTestPlanSchemaV0,
	)
	plan.DomainRef = strings.TrimSpace(plan.DomainRef)
	plan.WorkKind = strings.TrimSpace(plan.WorkKind)
	plan.JobRef = strings.TrimSpace(plan.JobRef)
	plan.AcceptanceCriteria = compactDomainWorkStringsV0(plan.AcceptanceCriteria)
	plan.RequiredTests = compactDomainWorkRequiredTestsV0(plan.RequiredTests)
	plan.ExternalRefs = compactDomainWorkExternalRefsV0(plan.ExternalRefs)
	plan.EvidenceRefs = compactDomainWorkStringsV0(plan.EvidenceRefs)
	return plan
}

func defaultDomainWorkSchemaV0(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func defaultDomainWorkRequestedByV0(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return DomainWorkDefaultRequestedByV0
	}
	return trimmed
}

func compactDomainWorkStringsV0(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if out == nil {
		return []string{}
	}
	return out
}

func compactDomainWorkExternalRefsV0(
	values []DomainWorkExternalRefV0,
) []DomainWorkExternalRefV0 {
	seen := map[string]struct{}{}
	out := make([]DomainWorkExternalRefV0, 0, len(values))
	for _, value := range values {
		ref := DomainWorkExternalRefV0{
			Kind: strings.TrimSpace(value.Kind),
			Ref:  strings.TrimSpace(value.Ref),
		}
		if ref.Kind == "" || ref.Ref == "" {
			continue
		}
		key := ref.Kind + "\x00" + ref.Ref
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ref)
	}
	if out == nil {
		return []DomainWorkExternalRefV0{}
	}
	return out
}

func compactDomainWorkRequiredTestsV0(
	values []DomainWorkRequiredTestV0,
) []DomainWorkRequiredTestV0 {
	seen := map[string]struct{}{}
	out := make([]DomainWorkRequiredTestV0, 0, len(values))
	for _, value := range values {
		test := DomainWorkRequiredTestV0{
			TestRef:                strings.TrimSpace(value.TestRef),
			AcceptanceCriteria:     compactDomainWorkStringsV0(value.AcceptanceCriteria),
			AcceptanceCriteriaRefs: compactDomainWorkStringsV0(value.AcceptanceCriteriaRefs),
			InputRefs:              compactDomainWorkStringsV0(value.InputRefs),
			ExternalRefs:           compactDomainWorkExternalRefsV0(value.ExternalRefs),
			EvidenceRefs:           compactDomainWorkStringsV0(value.EvidenceRefs),
		}
		if test.TestRef == "" {
			continue
		}
		key := test.TestRef + "\x00" +
			strings.Join(test.AcceptanceCriteria, "\x00") + "\x00" +
			strings.Join(test.AcceptanceCriteriaRefs, "\x00") + "\x00" +
			strings.Join(test.InputRefs, "\x00") + "\x00" +
			joinDomainWorkExternalRefsV0(test.ExternalRefs) + "\x00" +
			strings.Join(test.EvidenceRefs, "\x00")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, test)
	}
	if out == nil {
		return []DomainWorkRequiredTestV0{}
	}
	return out
}

func joinDomainWorkExternalRefsV0(values []DomainWorkExternalRefV0) string {
	parts := make([]string, 0, len(values)*2)
	for _, value := range values {
		parts = append(parts, value.Kind, value.Ref)
	}
	return strings.Join(parts, "\x00")
}

func compactDomainWorkFieldsV0(values []DomainWorkFieldV0) []DomainWorkFieldV0 {
	seen := map[string]struct{}{}
	out := make([]DomainWorkFieldV0, 0, len(values))
	for _, value := range values {
		field := DomainWorkFieldV0{
			Name:      strings.TrimSpace(value.Name),
			Value:     strings.TrimSpace(value.Value),
			Values:    compactDomainWorkStringsV0(value.Values),
			ValueJSON: normalizeDomainWorkFieldJSONV0(value.ValueJSON),
		}
		if field.Name == "" ||
			(field.Value == "" && len(field.Values) == 0 && len(field.ValueJSON) == 0) {
			continue
		}
		key := field.Name + "\x00" + field.Value + "\x00" +
			strings.Join(field.Values, "\x00") + "\x00" + string(field.ValueJSON)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, field)
	}
	if out == nil {
		return []DomainWorkFieldV0{}
	}
	return out
}

func normalizeDomainWorkFieldJSONV0(value json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	var raw any
	if err := json.Unmarshal(trimmed, &raw); err != nil {
		return append(json.RawMessage(nil), trimmed...)
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return append(json.RawMessage(nil), trimmed...)
	}
	return append(json.RawMessage(nil), data...)
}
