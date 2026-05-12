package orquestastatefileoutbox

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func normalizePendingFilterV0(
	filter orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0,
) (orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0) {
	filter.RunRef = trimV0(filter.RunRef)
	filter.TargetPort = trimV0(filter.TargetPort)
	if filter.TargetPort != "" && !targetPortSupportedV0(filter.TargetPort) {
		return filter, []orquestadirectorcycleoutbox.DirectorCycleOutboxIssueV0{
			directorIssueV0(errPayloadInvalidV0, "target_port", "target_port no soportado"),
		}
	}
	return filter, nil
}

func targetPortSupportedV0(targetPort string) bool {
	switch trimV0(targetPort) {
	case orquestacoreworkflow.OutboxTargetPersistenceV0,
		orquestacoreworkflow.OutboxTargetObservabilityV0,
		orquestacoreworkflow.OutboxTargetCapacityV0,
		orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		orquestacoreworkflow.OutboxTargetDeployPlannerV0,
		orquestacoreworkflow.OutboxTargetDirectorV0:
		return true
	default:
		return false
	}
}

func recordMatchesPendingFilterV0(
	record *outboxLedgerRecordV0,
	filter orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0,
) bool {
	if record == nil || record.Ack != nil {
		return false
	}
	if filter.RunRef != "" && record.Message.RunID != filter.RunRef {
		return false
	}
	return filter.TargetPort == "" || record.Message.TargetPort == filter.TargetPort
}

func recordMatchesDispatchFilterV0(
	record *outboxLedgerRecordV0,
	filter orquestaoutboxdispatch.PendingOutboxFilterV0,
) bool {
	if record == nil || record.Ack != nil {
		return false
	}
	message := record.Message
	if filter.RunID != "" && message.RunID != filter.RunID {
		return false
	}
	if filter.TargetPort != "" && message.TargetPort != filter.TargetPort {
		return false
	}
	return filter.MessageType == "" || message.MessageType == filter.MessageType
}

func validateClaimV0(
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	if claim.MessageID == "" {
		return []orquestaoutboxdispatch.DispatchIssueV0{
			dispatchIssueV0(errPayloadInvalidV0, "message_id", "message_id requerido"),
		}
	}
	if claim.TargetPort != "" && !targetPortSupportedV0(claim.TargetPort) {
		return []orquestaoutboxdispatch.DispatchIssueV0{
			dispatchIssueV0(errPayloadInvalidV0, "target_port", "target_port no soportado"),
		}
	}
	return nil
}

func validateAckV0(
	ack orquestaoutboxdispatch.OutboxDispatchAckV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	if ack.MessageID == "" {
		return []orquestaoutboxdispatch.DispatchIssueV0{
			dispatchIssueV0(errPayloadInvalidV0, "message_id", "message_id requerido"),
		}
	}
	if ack.TargetPort != "" && !targetPortSupportedV0(ack.TargetPort) {
		return []orquestaoutboxdispatch.DispatchIssueV0{
			dispatchIssueV0(errPayloadInvalidV0, "target_port", "target_port no soportado"),
		}
	}
	return nil
}

func claimConflictsV0(
	record *outboxLedgerRecordV0,
	claim orquestaoutboxdispatch.OutboxDispatchClaimV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	if claim.RunID != "" && claim.RunID != record.Message.RunID {
		return []orquestaoutboxdispatch.DispatchIssueV0{
			dispatchIssueV0(errIdempotencyConflictV0, "run_id", "claim no corresponde al mensaje"),
		}
	}
	if claim.TargetPort != "" && claim.TargetPort != record.Message.TargetPort {
		return []orquestaoutboxdispatch.DispatchIssueV0{
			dispatchIssueV0(errIdempotencyConflictV0, "target_port", "claim no corresponde al mensaje"),
		}
	}
	if claim.IdempotencyKey != "" && claim.IdempotencyKey != record.Message.IdempotencyKey {
		return []orquestaoutboxdispatch.DispatchIssueV0{
			dispatchIssueV0(errIdempotencyConflictV0, "idempotency_key", "claim no corresponde al mensaje"),
		}
	}
	return nil
}

func ackConflictsV0(
	record *outboxLedgerRecordV0,
	ack orquestaoutboxdispatch.OutboxDispatchAckV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	if ack.RunID != "" && ack.RunID != record.Message.RunID {
		return []orquestaoutboxdispatch.DispatchIssueV0{
			dispatchIssueV0(errIdempotencyConflictV0, "run_id", "ack no corresponde al mensaje"),
		}
	}
	if ack.TargetPort != "" && ack.TargetPort != record.Message.TargetPort {
		return []orquestaoutboxdispatch.DispatchIssueV0{
			dispatchIssueV0(errIdempotencyConflictV0, "target_port", "ack no corresponde al mensaje"),
		}
	}
	return nil
}
