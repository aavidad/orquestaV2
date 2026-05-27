package orquestaautoprogramming

import "strings"

const AutoprogrammingRequestSourceSchemaVersionV0 = "autoprogramming_request_source.v0"

type AutoprogrammingRequestSourceV0 struct {
	SchemaVersion          string                   `json:"schema_version,omitempty"`
	SourceRef              string                   `json:"source_ref,omitempty"`
	SourceSurface          string                   `json:"source_surface"`
	Transport              string                   `json:"transport"`
	Endpoint               string                   `json:"endpoint,omitempty"`
	ToolName               string                   `json:"tool_name,omitempty"`
	ResourceURI            string                   `json:"resource_uri,omitempty"`
	RequestID              string                   `json:"request_id"`
	CorrelationID          string                   `json:"correlation_id"`
	RequestedBy            string                   `json:"requested_by"`
	PriorityScore          int                      `json:"priority_score,omitempty"`
	AutoprogrammingRequest AutoprogrammingRequestV0 `json:"autoprogramming_request"`
}

type AutoprogrammingRequestSourceValidationResultV0 struct {
	Accepted          bool     `json:"accepted"`
	SourceRefs        []string `json:"source_refs,omitempty"`
	RequestValidation AutoprogrammingRequestValidationResultV0
	Issues            []AutoprogrammingRequestIssueV0 `json:"issues,omitempty"`
}

func ValidateAutoprogrammingRequestSourceV0(
	source AutoprogrammingRequestSourceV0,
) AutoprogrammingRequestSourceValidationResultV0 {
	source = normalizeAutoprogrammingRequestSourceV0(source)
	issues := autoprogrammingRequestSourceEnvelopeIssuesV0(source)
	sourceRefs := AutoprogrammingRequestSourceRefsV0(source)
	issues = append(issues, autoprogrammingRequestSourceContextIssuesV0(source, sourceRefs)...)

	requestValidation := ValidateAutoprogrammingRequestV0(source.AutoprogrammingRequest)
	if !requestValidation.Accepted {
		issues = append(issues, requestValidation.Issues...)
	}
	return AutoprogrammingRequestSourceValidationResultV0{
		Accepted:          len(issues) == 0,
		SourceRefs:        sourceRefs,
		RequestValidation: requestValidation,
		Issues:            issues,
	}
}

func AutoprogrammingRequestSourceRefsV0(source AutoprogrammingRequestSourceV0) []string {
	source = normalizeAutoprogrammingRequestSourceV0(source)
	return compactStringsV0([]string{
		"source_ref:" + source.SourceRef,
		"source_surface:" + source.SourceSurface,
		"source_transport:" + source.Transport,
		"source_request_id:" + source.RequestID,
		"source_correlation_id:" + source.CorrelationID,
		"source_requested_by:" + source.RequestedBy,
	})
}

func normalizeAutoprogrammingRequestSourceV0(
	source AutoprogrammingRequestSourceV0,
) AutoprogrammingRequestSourceV0 {
	source.SchemaVersion = strings.TrimSpace(source.SchemaVersion)
	source.SourceRef = strings.TrimSpace(source.SourceRef)
	source.SourceSurface = normalizeAutoprogrammingTaskAreaV0(source.SourceSurface)
	source.Transport = normalizeAutoprogrammingTaskAreaV0(source.Transport)
	source.Endpoint = strings.TrimSpace(source.Endpoint)
	source.ToolName = strings.TrimSpace(source.ToolName)
	source.ResourceURI = strings.TrimSpace(source.ResourceURI)
	source.RequestID = strings.TrimSpace(source.RequestID)
	source.CorrelationID = strings.TrimSpace(source.CorrelationID)
	source.RequestedBy = strings.TrimSpace(source.RequestedBy)
	return source
}

func autoprogrammingRequestSourceEnvelopeIssuesV0(
	source AutoprogrammingRequestSourceV0,
) []AutoprogrammingRequestIssueV0 {
	var issues []AutoprogrammingRequestIssueV0
	if source.SchemaVersion != AutoprogrammingRequestSourceSchemaVersionV0 {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"schema_version_invalid",
			"schema_version",
			"schema_version autoprogramming_request_source.v0 requerida",
		))
	}
	for _, item := range []struct {
		field string
		value string
	}{
		{"source_ref", source.SourceRef},
		{"source_surface", source.SourceSurface},
		{"transport", source.Transport},
		{"request_id", source.RequestID},
		{"correlation_id", source.CorrelationID},
		{"requested_by", source.RequestedBy},
	} {
		if item.value == "" {
			issues = append(issues, autoprogrammingRequestIssueV0(
				item.field+"_missing",
				item.field,
				item.field+" requerido para solicitud fuente",
			))
		}
	}
	if source.PriorityScore <= 0 {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"priority_score_invalid",
			"priority_score",
			"prioridad positiva requerida para solicitud fuente",
		))
	}
	return issues
}

func autoprogrammingRequestSourceContextIssuesV0(
	source AutoprogrammingRequestSourceV0,
	sourceRefs []string,
) []AutoprogrammingRequestIssueV0 {
	var issues []AutoprogrammingRequestIssueV0
	for _, task := range source.AutoprogrammingRequest.Tasks {
		taskRefs := compactStringsV0(task.ContextRefs)
		for _, ref := range sourceRefs {
			if autoprogrammingRequestSourceHasContextRefV0(taskRefs, ref) {
				continue
			}
			issues = append(issues, autoprogrammingRequestIssueV0(
				"source_context_ref_missing",
				"tasks.context_refs",
				"context_ref fuente requerido: "+ref,
			))
		}
	}
	return issues
}

func autoprogrammingRequestSourceHasContextRefV0(refs []string, want string) bool {
	for _, ref := range refs {
		if ref == want {
			return true
		}
	}
	return false
}
