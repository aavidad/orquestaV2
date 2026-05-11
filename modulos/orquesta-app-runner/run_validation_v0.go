package orquestaapprunner

import "strings"

func validateRunPreparedAppOrchestrationRequestV0(
	request RunPreparedAppOrchestrationRequestV0,
) error {
	if request.Prepared.SchemaVersion != AppOrchestrationPreparedSchemaVersionV0 {
		return AppRunnerIssueV0{Field: "prepared.schema_version"}
	}
	if strings.TrimSpace(request.Prepared.Run.RunID) == "" {
		return AppRunnerIssueV0{Field: "prepared.run.run_id"}
	}
	if len(request.Prepared.Plan.Units) == 0 {
		return AppRunnerIssueV0{Field: "prepared.plan.units"}
	}
	if strings.TrimSpace(request.OccurredAt) == "" {
		return AppRunnerIssueV0{Field: "occurred_at"}
	}
	return nil
}

func validateRunPreparedAppOrchestrationPortsV0(
	ports RunPreparedAppOrchestrationPortsV0,
) error {
	if ports.RunStore == nil {
		return AppRunnerIssueV0{Field: "ports.run_store"}
	}
	if ports.EventSink == nil {
		return AppRunnerIssueV0{Field: "ports.event_sink"}
	}
	if ports.OutboxLedger == nil {
		return AppRunnerIssueV0{Field: "ports.outbox_ledger"}
	}
	if len(ports.Dispatchers) == 0 && len(ports.BatchDispatchers) == 0 {
		return AppRunnerIssueV0{Field: "ports.dispatchers"}
	}
	return nil
}
