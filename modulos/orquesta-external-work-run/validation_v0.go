package orquestaexternalworkrun

import (
	"strings"
	"time"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
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

func externalWorkRunIssueV0(code string, field string) ExternalWorkRunIssueV0 {
	return ExternalWorkRunIssueV0{Code: strings.TrimSpace(code), Field: strings.TrimSpace(field)}
}
