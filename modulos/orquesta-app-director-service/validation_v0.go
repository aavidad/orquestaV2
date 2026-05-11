package orquestaappdirectorservice

import "strings"

func validateStartAppDirectorRequestV0(
	request StartAppDirectorRequestV0,
) error {
	if strings.TrimSpace(request.AppSpecRequest.SchemaVersion) == "" {
		return AppDirectorServiceIssueV0{Field: "app_spec_request.schema_version"}
	}
	if strings.TrimSpace(request.AppSpecRequest.RequestID) == "" {
		return AppDirectorServiceIssueV0{Field: "app_spec_request.request_id"}
	}
	return nil
}

func validateStartAppDirectorPortsV0(
	ports StartAppDirectorPortsV0,
) error {
	if ports.RunStore == nil {
		return AppDirectorServiceIssueV0{Field: "ports.run_store"}
	}
	if ports.EventSink == nil {
		return AppDirectorServiceIssueV0{Field: "ports.event_sink"}
	}
	if ports.OutboxLedger == nil {
		return AppDirectorServiceIssueV0{Field: "ports.outbox_ledger"}
	}
	if len(ports.Dispatchers) == 0 && len(ports.BatchDispatchers) == 0 {
		return AppDirectorServiceIssueV0{Field: "ports.dispatchers"}
	}
	return nil
}
