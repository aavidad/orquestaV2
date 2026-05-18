package orquestaexternalworkrun

import (
	"encoding/json"
	"strings"
	"time"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func validateStartExternalWorkRunRequestV0(
	request StartExternalWorkRunRequestV0,
	ports StartExternalWorkRunPortsV0,
) []ExternalWorkRunIssueV0 {
	var issues []ExternalWorkRunIssueV0
	issues = append(issues, validateStartExternalWorkRunPortsV0(ports)...)
	issues = append(issues, validateStartExternalWorkRunRefsV0(request)...)
	if request.SchemaVersion != StartExternalWorkRunRequestSchemaV0 {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunSchemaVersionV0, "schema_version"))
	}
	if strings.TrimSpace(request.OccurredAt) == "" {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunOccurredAtRequiredV0, "occurred_at"))
	} else if _, err := time.Parse(time.RFC3339, request.OccurredAt); err != nil {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunOccurredAtInvalidV0, "occurred_at"))
	}
	if !hasExternalWorkV0(request.AppChangeRequest) {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunExternalWorkRequiredV0, "app_change_request.external_work"))
	}
	issues = append(issues, validateExternalWorkRunContextV0(request.AppChangeRequest.ExternalWork)...)
	for _, issue := range orquestaappchange.ValidateAppChangeRequestV0(request.AppChangeRequest) {
		issues = append(issues, externalWorkRunIssueV0(
			ErrExternalWorkRunAppChangeInvalidV0+":"+issue.Code,
			"app_change_request."+issue.Field,
		))
	}
	return issues
}

func validateStartExternalWorkRunPortsV0(
	ports StartExternalWorkRunPortsV0,
) []ExternalWorkRunIssueV0 {
	var issues []ExternalWorkRunIssueV0
	if ports.RunStore == nil {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunRunStoreRequiredV0, "run_store"))
	}
	if ports.EventSink == nil {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunEventSinkRequiredV0, "event_sink"))
	}
	if ports.RunQueue == nil {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunQueueWriterRequiredV0, "run_queue"))
	}
	if ports.AppChange.Store == nil {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunAppChangeStoreV0, "app_change.store"))
	}
	if ports.AppChange.DirectorNotifier == nil {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunAppChangeNotifierV0, "app_change.director_notifier"))
	}
	return issues
}

func validateStartExternalWorkRunRefsV0(
	request StartExternalWorkRunRequestV0,
) []ExternalWorkRunIssueV0 {
	var issues []ExternalWorkRunIssueV0
	if strings.TrimSpace(request.ProjectRef) == "" {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunProjectRefRequiredV0, "project_ref"))
	}
	if strings.TrimSpace(request.AppSpecRef) == "" {
		issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunAppSpecRefRequiredV0, "app_spec_ref"))
	}
	for field, value := range map[string]string{
		"run_ref":      request.RunRef,
		"project_ref":  request.ProjectRef,
		"app_spec_ref": request.AppSpecRef,
		"queue_ref":    request.QueueRef,
	} {
		if strings.ContainsAny(strings.TrimSpace(value), " /\\\t\n\r") {
			issues = append(issues, externalWorkRunIssueV0(ErrExternalWorkRunRefInvalidV0, field))
		}
	}
	return issues
}

func hasExternalWorkV0(request orquestaappchange.AppChangeRequestV0) bool {
	return request.ExternalWork != nil &&
		(request.ExternalWork.ProjectRef != "" ||
			request.ExternalWork.JobRef != "" ||
			request.ExternalWork.WorkKind != "" ||
			len(request.ExternalWork.InterfaceRefs) > 0 ||
			len(request.ExternalWork.WorkRefs) > 0 ||
			len(request.ExternalWork.InputFields) > 0)
}

func validateExternalWorkRunContextV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
) []ExternalWorkRunIssueV0 {
	if work == nil {
		return nil
	}
	var issues []ExternalWorkRunIssueV0
	if len(externalWorkRunFieldValuesV0(work.InputFields, "missing_context")) > 0 {
		issues = append(issues, externalWorkRunIssueV0(
			ErrExternalWorkRunMissingContextV0,
			"app_change_request.external_work.input_fields.missing_context",
		))
	}
	for _, required := range externalWorkRunFieldValuesV0(work.InputFields, "required_input_fields") {
		if !externalWorkRunHasInputFieldValueV0(work.InputFields, required) {
			issues = append(issues, externalWorkRunIssueV0(
				ErrExternalWorkRunRequiredInputMissingV0,
				"app_change_request.external_work.input_fields."+required,
			))
		}
	}
	return issues
}

func externalWorkRunFieldValuesV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
) []string {
	name = strings.TrimSpace(name)
	values := make([]string, 0)
	for _, field := range fields {
		if strings.TrimSpace(field.Name) != name {
			continue
		}
		values = append(values, compactExternalWorkRunStringsV0(field.Values)...)
		if value := strings.TrimSpace(field.Value); value != "" {
			values = append(values, value)
		}
		values = append(values, externalWorkRunJSONFieldValuesV0(field.ValueJSON)...)
	}
	return compactExternalWorkRunStringsV0(values)
}

func externalWorkRunJSONFieldValuesV0(raw json.RawMessage) []string {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "null" {
		return nil
	}
	var many []string
	if err := json.Unmarshal(raw, &many); err == nil {
		return compactExternalWorkRunStringsV0(many)
	}
	var one string
	if err := json.Unmarshal(raw, &one); err == nil {
		return compactExternalWorkRunStringsV0([]string{one})
	}
	return []string{strings.TrimSpace(string(raw))}
}

func externalWorkRunHasInputFieldValueV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return true
	}
	for _, field := range fields {
		if strings.TrimSpace(field.Name) != name {
			continue
		}
		if strings.TrimSpace(field.Value) != "" ||
			len(compactExternalWorkRunStringsV0(field.Values)) > 0 ||
			len(externalWorkRunJSONFieldValuesV0(field.ValueJSON)) > 0 {
			return true
		}
	}
	return false
}

func externalWorkRunIssueV0(code string, field string) ExternalWorkRunIssueV0 {
	return ExternalWorkRunIssueV0{Code: strings.TrimSpace(code), Field: strings.TrimSpace(field)}
}
