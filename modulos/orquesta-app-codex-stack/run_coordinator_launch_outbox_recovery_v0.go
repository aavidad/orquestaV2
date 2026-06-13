package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

const codexStackLaunchOutboxRecoveryRequestedByV0 = "orquesta-app-codex-stack-launch-outbox-recovery"

func (stack StackV0) recoverMissingCapacityOutboxForPendingCapacityRequestsV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, error) {
	if stack.Stores.RunStore == nil || stack.Stores.OutboxLedger == nil {
		return false, nil
	}
	reader := stack.eventReaderForLaunchOutboxRecoveryV0()
	if reader == nil {
		return false, nil
	}
	missingRefs, err := stack.missingCapacityOutboxRefsV0(ctx, run)
	if err != nil || len(missingRefs) == 0 {
		return false, err
	}
	events, err := reader.LoadRunEventsV0(ctx, run.RunID)
	if err != nil {
		return false, err
	}
	eventsByCapacity := capacityRequestedEventsByCapacityRefV0(run.RunID, events)
	recovered := false
	for _, capacityRef := range missingRefs {
		event, ok := eventsByCapacity[capacityRef]
		if !ok {
			continue
		}
		command, err := requestCapacityCommandFromRequestedEventV0(event)
		if err != nil {
			return false, err
		}
		result, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, stack.Stores.RunStore, stack.Ports.EventSink, command)
		if err != nil {
			return false, err
		}
		if len(result.Outbox) == 0 {
			continue
		}
		record, err := orquestadirectorcycleoutbox.RecordDirectorCycleOutboxV0(ctx, orquestadirectorcycleoutbox.DirectorCycleOutboxRecordInputV0{
			Ledger:        stack.Stores.OutboxLedger,
			RunRef:        run.RunID,
			TargetPort:    orquestacoreworkflow.OutboxTargetCapacityV0,
			Messages:      result.Outbox,
			CorrelationID: strings.TrimSpace(command.CorrelationID),
		})
		if err != nil {
			return false, err
		}
		recovered = recovered || record.SavedCount > 0
	}
	return recovered, nil
}

func (stack StackV0) recoverMissingLaunchOutboxForRequestedAgentsV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, error) {
	if stack.Stores.RunStore == nil || stack.Stores.OutboxLedger == nil {
		return false, nil
	}
	reader := stack.eventReaderForLaunchOutboxRecoveryV0()
	if reader == nil {
		return false, nil
	}
	missingRefs, err := stack.missingLaunchOutboxAgentRefsV0(ctx, run)
	if err != nil || len(missingRefs) == 0 {
		return false, err
	}
	events, err := reader.LoadRunEventsV0(ctx, run.RunID)
	if err != nil {
		return false, err
	}
	eventsByAgent := agentRequestedEventsByAgentRefV0(run.RunID, events)
	recovered := false
	for _, agentRef := range missingRefs {
		event, ok := eventsByAgent[agentRef]
		if !ok {
			continue
		}
		command, err := requestAgentCommandFromRequestedEventV0(event)
		if err != nil {
			return false, err
		}
		result, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, stack.Stores.RunStore, stack.Ports.EventSink, command)
		if err != nil {
			return false, err
		}
		if len(result.Outbox) == 0 {
			continue
		}
		record, err := orquestadirectorcycleoutbox.RecordDirectorCycleOutboxV0(ctx, orquestadirectorcycleoutbox.DirectorCycleOutboxRecordInputV0{
			Ledger:        stack.Stores.OutboxLedger,
			RunRef:        run.RunID,
			TargetPort:    orquestacoreworkflow.OutboxTargetAgentLauncherV0,
			Messages:      result.Outbox,
			CorrelationID: strings.TrimSpace(command.CorrelationID),
		})
		if err != nil {
			return false, err
		}
		recovered = recovered || record.SavedCount > 0
	}
	return recovered, nil
}

func (stack StackV0) eventReaderForLaunchOutboxRecoveryV0() orquestacionnucleoapp.RunEventReaderPortV0 {
	if stack.Ports.EventReader != nil {
		return stack.Ports.EventReader
	}
	reader, _ := stack.Stores.EventSink.(orquestacionnucleoapp.RunEventReaderPortV0)
	return reader
}

func (stack StackV0) missingLaunchOutboxAgentRefsV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) ([]string, error) {
	requested := requestedAgentsWithoutLaunchTerminalV0(run)
	if len(requested) == 0 {
		return nil, nil
	}
	pending, issues := stack.Stores.OutboxLedger.ListPendingOutboxV0(orquestaoutboxdispatch.PendingOutboxFilterV0{
		RunID:       run.RunID,
		TargetPort:  orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MessageType: orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0,
	})
	if len(issues) > 0 {
		return nil, fmt.Errorf("launch_outbox_recovery: list_pending: %s", strings.TrimSpace(issues[0].Code))
	}
	pendingRefs := launchOutboxAgentRefSetV0(pending)
	missing := make([]string, 0, len(requested))
	for _, agentRef := range requested {
		if pendingRefs[agentRef] {
			continue
		}
		missing = append(missing, agentRef)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return missing, nil
}

func (stack StackV0) missingCapacityOutboxRefsV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) ([]string, error) {
	requested := capacityRequestsWithoutDecisionV0(run)
	if len(requested) == 0 {
		return nil, nil
	}
	pending, issues := stack.Stores.OutboxLedger.ListPendingOutboxV0(orquestaoutboxdispatch.PendingOutboxFilterV0{
		RunID:       run.RunID,
		TargetPort:  orquestacoreworkflow.OutboxTargetCapacityV0,
		MessageType: orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0,
	})
	if len(issues) > 0 {
		return nil, fmt.Errorf("capacity_outbox_recovery: list_pending: %s", strings.TrimSpace(issues[0].Code))
	}
	pendingRefs := capacityOutboxRequestRefSetV0(pending)
	missing := make([]string, 0, len(requested))
	for _, capacityRef := range requested {
		if pendingRefs[capacityRef] {
			continue
		}
		missing = append(missing, capacityRef)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return missing, nil
}

func capacityRequestsWithoutDecisionV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	decided := map[string]bool{}
	for _, decisionRef := range compactStringsV0(run.CapacityDecisions) {
		capacityRef, _, _ := strings.Cut(decisionRef, "#capacity_decision:")
		capacityRef = strings.TrimSpace(capacityRef)
		if capacityRef != "" {
			decided[capacityRef] = true
		}
	}
	refs := make([]string, 0, len(run.CapacityRequests))
	for _, capacityRef := range compactStringsV0(run.CapacityRequests) {
		if decided[capacityRef] {
			continue
		}
		refs = append(refs, capacityRef)
	}
	return refs
}

func capacityOutboxRequestRefSetV0(
	pending []orquestaoutboxdispatch.OutboxPendingEntryV0,
) map[string]bool {
	refs := map[string]bool{}
	for _, entry := range pending {
		var payload orquestacoreworkflow.CapacityDecisionRequestV0
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			continue
		}
		capacityRef := strings.TrimSpace(payload.CapacityRequestID)
		if capacityRef != "" {
			refs[capacityRef] = true
		}
	}
	return refs
}

func requestedAgentsWithoutLaunchTerminalV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	started := launchOutboxRecoveryStringSetV0(run.StartedAgents)
	terminal := stackTerminalAgentRefSetV0(run)
	refs := make([]string, 0, len(run.Agents))
	for _, agentRef := range compactStringsV0(run.Agents) {
		if started[agentRef] || terminal[agentRef] {
			continue
		}
		refs = append(refs, agentRef)
	}
	return refs
}

func launchOutboxRecoveryStringSetV0(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range compactStringsV0(values) {
		set[value] = true
	}
	return set
}

func launchOutboxAgentRefSetV0(
	pending []orquestaoutboxdispatch.OutboxPendingEntryV0,
) map[string]bool {
	refs := map[string]bool{}
	for _, entry := range pending {
		var payload orquestacoreworkflow.LaunchRuntimeAgentRequestV0
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			continue
		}
		agentRef := strings.TrimSpace(payload.AgentRequestID)
		if agentRef != "" {
			refs[agentRef] = true
		}
	}
	return refs
}

func capacityRequestedEventsByCapacityRefV0(
	runRef string,
	events []orquestacoreworkflow.OrchestrationEventV0,
) map[string]orquestacoreworkflow.OrchestrationEventV0 {
	out := map[string]orquestacoreworkflow.OrchestrationEventV0{}
	for _, event := range events {
		if strings.TrimSpace(event.RunID) != strings.TrimSpace(runRef) ||
			event.EventType != orquestacoreworkflow.OrchestrationEventCapacityRequestedV0 {
			continue
		}
		var payload orquestacoreworkflow.CapacityRequestedPayloadV0
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		capacityRef := strings.TrimSpace(payload.CapacityRequestID)
		if capacityRef != "" {
			out[capacityRef] = event
		}
	}
	return out
}

func agentRequestedEventsByAgentRefV0(
	runRef string,
	events []orquestacoreworkflow.OrchestrationEventV0,
) map[string]orquestacoreworkflow.OrchestrationEventV0 {
	out := map[string]orquestacoreworkflow.OrchestrationEventV0{}
	for _, event := range events {
		if strings.TrimSpace(event.RunID) != strings.TrimSpace(runRef) ||
			event.EventType != orquestacoreworkflow.OrchestrationEventAgentRequestedV0 {
			continue
		}
		var payload orquestacoreworkflow.AgentRequestedPayloadV0
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		agentRef := strings.TrimSpace(payload.AgentRequestID)
		if agentRef != "" {
			out[agentRef] = event
		}
	}
	return out
}

func requestCapacityCommandFromRequestedEventV0(
	event orquestacoreworkflow.OrchestrationEventV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	var payload orquestacoreworkflow.CapacityRequestedPayloadV0
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return orquestacoreworkflow.OrchestrationCommandV0{}, err
	}
	if strings.TrimSpace(event.CausationID) == "" {
		return orquestacoreworkflow.OrchestrationCommandV0{}, fmt.Errorf("capacity_outbox_recovery: causation_id requerido")
	}
	return orquestacoreworkflow.NewRequestCapacityCommandV0(orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      strings.TrimSpace(event.CausationID),
		RunID:          strings.TrimSpace(event.RunID),
		IdempotencyKey: strings.TrimSpace(event.IdempotencyKey),
		CorrelationID:  strings.TrimSpace(event.CorrelationID),
		RequestedBy:    codexStackLaunchOutboxRecoveryRequestedByV0,
		OccurredAt:     strings.TrimSpace(event.OccurredAt),
	}, orquestacoreworkflow.RequestCapacityCommandPayloadV0(payload))
}

func requestAgentCommandFromRequestedEventV0(
	event orquestacoreworkflow.OrchestrationEventV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	var payload orquestacoreworkflow.AgentRequestedPayloadV0
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return orquestacoreworkflow.OrchestrationCommandV0{}, err
	}
	if strings.TrimSpace(event.CausationID) == "" {
		return orquestacoreworkflow.OrchestrationCommandV0{}, fmt.Errorf("launch_outbox_recovery: causation_id requerido")
	}
	return orquestacoreworkflow.NewRequestAgentCommandV0(orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      strings.TrimSpace(event.CausationID),
		RunID:          strings.TrimSpace(event.RunID),
		IdempotencyKey: strings.TrimSpace(event.IdempotencyKey),
		CorrelationID:  strings.TrimSpace(event.CorrelationID),
		RequestedBy:    codexStackLaunchOutboxRecoveryRequestedByV0,
		OccurredAt:     strings.TrimSpace(event.OccurredAt),
	}, orquestacoreworkflow.RequestAgentCommandPayloadV0(payload))
}
