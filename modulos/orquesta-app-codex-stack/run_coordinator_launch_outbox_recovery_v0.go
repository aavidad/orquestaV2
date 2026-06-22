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

func (stack StackV0) reconcileOrphanCapacityOutboxForRunV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	occurredAt string,
) (bool, error) {
	if stack.Stores.RunStore == nil || stack.Stores.OutboxLedger == nil {
		return false, nil
	}
	pending, err := stack.pendingCapacityOutboxEntriesForReconcileV0(ctx, run.RunID)
	if err != nil {
		return false, err
	}
	known := launchOutboxRecoveryStringSetV0(run.CapacityRequests)
	decided := capacityOutboxDecisionSetV0(run)
	reconciled := false
	for _, entry := range pending {
		var payload orquestacoreworkflow.CapacityDecisionRequestV0
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			continue
		}
		capacityRef := strings.TrimSpace(payload.CapacityRequestID)
		if capacityRef == "" {
			continue
		}
		if known[capacityRef] && decided[capacityRef] {
			if err := stack.ackSupersededCapacityOutboxEntryV0(entry, payload); err != nil {
				return false, err
			}
			reconciled = true
			continue
		}
		if known[capacityRef] {
			continue
		}
		command, err := requestCapacityCommandFromOutboxEntryV0(entry, payload, occurredAt)
		if err != nil {
			return false, err
		}
		if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, stack.Stores.RunStore, stack.Ports.EventSink, command); err != nil {
			return false, err
		}
		stack.releaseOutboxDispatchClaimV0(entry)
		known[capacityRef] = true
		reconciled = true
	}
	return reconciled, nil
}

func capacityOutboxDecisionSetV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) map[string]bool {
	decided := map[string]bool{}
	for _, decisionRef := range compactStringsV0(run.CapacityDecisions) {
		capacityRef, _, _ := strings.Cut(decisionRef, "#capacity_decision:")
		capacityRef = strings.TrimSpace(capacityRef)
		if capacityRef != "" {
			decided[capacityRef] = true
		}
	}
	return decided
}

func (stack StackV0) reconcileClaimedLaunchOutboxForProcessRegistryV0(
	ctx context.Context,
	request DrainRunRequestV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) (bool, error) {
	if stack.Stores.RunStore == nil || stack.Stores.OutboxLedger == nil ||
		stack.Stores.ProcessRegistry == nil || stack.Ports.EventSink == nil {
		return false, nil
	}
	pending, err := stack.pendingLaunchOutboxEntriesForReconcileV0(ctx, run.RunID)
	if err != nil {
		return false, err
	}
	if len(pending) == 0 {
		return false, nil
	}
	reader := stack.eventReaderForLaunchOutboxRecoveryV0()
	if reader == nil {
		return false, nil
	}
	events, err := reader.LoadRunEventsV0(ctx, run.RunID)
	if err != nil {
		return false, err
	}
	agentRequestedEvents := agentRequestedEventsByAgentRefV0(run.RunID, events)
	agentStartedEvents := agentStartedEventsByAgentRefV0(run.RunID, events)
	current := run
	reconciled := false
	for _, entry := range pending {
		var payload orquestacoreworkflow.LaunchRuntimeAgentRequestV0
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			continue
		}
		agentRef := strings.TrimSpace(payload.AgentRequestID)
		if agentRef == "" {
			continue
		}
		if codexStackStringInSetV0(current.StartedAgents, agentRef) {
			if err := stack.ackSupersededLaunchOutboxEntryV0(entry, payload, nil); err != nil {
				return false, err
			}
			reconciled = true
			continue
		}
		var projected bool
		var projectErr error
		current, projected, projectErr = stack.projectDurableAgentRequestedForLaunchRecoveryV0(ctx, current, agentRef, agentRequestedEvents)
		if projectErr != nil {
			return false, projectErr
		}
		if projected {
			reconciled = true
		}
		if !codexStackStringInSetV0(current.Agents, agentRef) {
			continue
		}
		if event, ok := agentStartedEvents[agentRef]; ok {
			next, applied, err := stack.projectDurableAgentStartedForLaunchRecoveryV0(ctx, current, event)
			if err != nil {
				return false, err
			}
			current = next
			reconciled = reconciled || applied
			if err := stack.ackSupersededLaunchOutboxEntryV0(entry, payload, nil); err != nil {
				return false, err
			}
			reconciled = true
			continue
		}
		record, err := stack.Stores.ProcessRegistry.ResolveAgentProcessV0(ctx, strings.TrimSpace(run.RunID), agentRef)
		if err != nil {
			continue
		}
		command, err := registerAgentStartedCommandFromProcessRecordV0(entry, payload, record, request)
		if err != nil {
			return false, err
		}
		if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, stack.Stores.RunStore, stack.Ports.EventSink, command); err != nil {
			return false, err
		}
		if err := stack.ackSupersededLaunchOutboxEntryV0(entry, payload, record.EvidenceRefs); err != nil {
			return false, err
		}
		current, err = stack.Stores.RunStore.LoadRunV0(ctx, run.RunID)
		if err != nil {
			return false, err
		}
		reconciled = true
	}
	return reconciled, nil
}

func (stack StackV0) projectDurableAgentRequestedForLaunchRecoveryV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRef string,
	events map[string]orquestacoreworkflow.OrchestrationEventV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	if codexStackStringInSetV0(run.Agents, agentRef) {
		return run, false, nil
	}
	event, ok := events[strings.TrimSpace(agentRef)]
	if !ok {
		return run, false, nil
	}
	command, err := requestAgentCommandFromRequestedEventV0(event)
	if err != nil {
		return run, false, err
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(ctx, stack.Stores.RunStore, stack.Ports.EventSink, command); err != nil {
		return run, false, err
	}
	next, err := stack.Stores.RunStore.LoadRunV0(ctx, run.RunID)
	return next, true, err
}

func (stack StackV0) projectDurableAgentStartedForLaunchRecoveryV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	event orquestacoreworkflow.OrchestrationEventV0,
) (orquestacoreworkflow.OrchestrationRunV0, bool, error) {
	var payload orquestacoreworkflow.AgentStartedPayloadV0
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return run, false, err
	}
	agentRef := strings.TrimSpace(payload.AgentRequestID)
	if agentRef == "" || codexStackStringInSetV0(run.StartedAgents, agentRef) {
		return run, false, nil
	}
	lastEventID := run.LastEventID
	lastSequence := run.LastSequence
	projected, err := orquestacoreworkflow.ApplyEventV0(run, event)
	if err != nil {
		return run, false, err
	}
	projected.LastEventID = lastEventID
	projected.LastSequence = lastSequence
	if err := stack.Stores.RunStore.SaveRunV0(ctx, projected); err != nil {
		return run, false, err
	}
	return projected, true, nil
}

func registerAgentStartedCommandFromProcessRecordV0(
	entry orquestaoutboxdispatch.OutboxPendingEntryV0,
	payload orquestacoreworkflow.LaunchRuntimeAgentRequestV0,
	record orquestacionnucleoapp.AgentProcessRegistryRecordV0,
	request DrainRunRequestV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	agentRef := firstNonEmptyQueuedSourceV0(
		strings.TrimSpace(record.AgentRequestID),
		strings.TrimSpace(payload.AgentRequestID),
	)
	return orquestacoreworkflow.NewRegisterAgentStartedCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-reconcile-agent-started-" + strings.TrimSpace(entry.MessageID),
			RunID:          strings.TrimSpace(entry.RunID),
			IdempotencyKey: "idem-reconcile-agent-started-" + strings.TrimSpace(entry.MessageID),
			CorrelationID:  firstNonEmptyQueuedSourceV0(strings.TrimSpace(request.CorrelationID), strings.TrimSpace(entry.CorrelationID)),
			RequestedBy:    codexStackLaunchOutboxRecoveryRequestedByV0,
			OccurredAt:     firstNonEmptyQueuedSourceV0(strings.TrimSpace(request.OccurredAt), "2026-05-10T12:00:00Z"),
		},
		orquestacoreworkflow.RegisterAgentStartedCommandPayloadV0{
			AgentRequestID: agentRef,
			LaunchRef:      strings.TrimSpace(record.LaunchRef),
			AckRef:         ackRefFromAgentProcessRecordV0(record),
			ReadinessRef:   strings.TrimSpace(record.ReadinessRef),
			EvidenceRefs: compactStringsV0(append(
				record.EvidenceRefs,
				payload.EvidenceRefs...,
			)),
		},
	)
}

func ackRefFromAgentProcessRecordV0(
	record orquestacionnucleoapp.AgentProcessRegistryRecordV0,
) string {
	for _, ref := range compactStringsV0(record.EvidenceRefs) {
		if strings.HasPrefix(ref, "ack-ref-") {
			return ref
		}
	}
	return "ack-ref-" + strings.TrimSpace(record.AgentRequestID)
}

func (stack StackV0) ackSupersededLaunchOutboxEntryV0(
	entry orquestaoutboxdispatch.OutboxPendingEntryV0,
	payload orquestacoreworkflow.LaunchRuntimeAgentRequestV0,
	evidenceRefs []string,
) error {
	if stack.Stores.OutboxLedger == nil {
		return nil
	}
	issues := stack.Stores.OutboxLedger.AckOutboxDispatchV0(orquestaoutboxdispatch.OutboxDispatchAckV0{
		MessageID:   strings.TrimSpace(entry.MessageID),
		RunID:       strings.TrimSpace(entry.RunID),
		TargetPort:  strings.TrimSpace(entry.TargetPort),
		DispatchRef: "dispatch-ref-reconciled-" + strings.TrimSpace(entry.MessageID),
		EvidenceRefs: compactStringsV0(append(append(
			payload.EvidenceRefs,
			evidenceRefs...,
		),
			"evidence-ref-launch-outbox-reconciled-from-process-registry",
			"evidence-ref-agent-request-"+strings.TrimSpace(payload.AgentRequestID),
		)),
	})
	if len(issues) > 0 {
		return fmt.Errorf("launch_outbox_reconcile: ack_superseded: %s", strings.TrimSpace(issues[0].Code))
	}
	return nil
}

func (stack StackV0) ackSupersededCapacityOutboxEntryV0(
	entry orquestaoutboxdispatch.OutboxPendingEntryV0,
	payload orquestacoreworkflow.CapacityDecisionRequestV0,
) error {
	if stack.Stores.OutboxLedger == nil {
		return nil
	}
	issues := stack.Stores.OutboxLedger.AckOutboxDispatchV0(orquestaoutboxdispatch.OutboxDispatchAckV0{
		MessageID:   strings.TrimSpace(entry.MessageID),
		RunID:       strings.TrimSpace(entry.RunID),
		TargetPort:  strings.TrimSpace(entry.TargetPort),
		DispatchRef: "dispatch-ref-superseded-" + strings.TrimSpace(entry.MessageID),
		EvidenceRefs: compactStringsV0(append(
			payload.EvidenceRefs,
			"evidence-ref-capacity-outbox-superseded-by-existing-decision",
			"evidence-ref-capacity-request-"+strings.TrimSpace(payload.CapacityRequestID),
		)),
	})
	if len(issues) > 0 {
		return fmt.Errorf("capacity_outbox_reconcile: ack_superseded: %s", strings.TrimSpace(issues[0].Code))
	}
	return nil
}

func (stack StackV0) pendingLaunchOutboxEntriesForReconcileV0(
	ctx context.Context,
	runID string,
) ([]orquestaoutboxdispatch.OutboxPendingEntryV0, error) {
	pending, issues := stack.Stores.OutboxLedger.ListPending(ctx, orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
		RunRef:     strings.TrimSpace(runID),
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
	})
	if len(issues) > 0 {
		return nil, fmt.Errorf("launch_outbox_reconcile: list_pending: %s", strings.TrimSpace(issues[0].Code))
	}
	entries := make([]orquestaoutboxdispatch.OutboxPendingEntryV0, 0, len(pending))
	for _, message := range pending {
		if strings.TrimSpace(message.MessageType) != orquestacoreworkflow.OutboxMessageLaunchRuntimeAgentV0 {
			continue
		}
		entries = append(entries, orquestaoutboxdispatch.OutboxPendingEntryV0{
			MessageID:      strings.TrimSpace(message.MessageID),
			RunID:          strings.TrimSpace(message.RunID),
			TargetPort:     strings.TrimSpace(message.TargetPort),
			MessageType:    strings.TrimSpace(message.MessageType),
			IdempotencyKey: strings.TrimSpace(message.IdempotencyKey),
			CorrelationID:  strings.TrimSpace(message.CorrelationID),
			PayloadVersion: strings.TrimSpace(message.PayloadVersion),
			Payload:        append(message.Payload[:0:0], message.Payload...),
		})
	}
	return entries, nil
}

func (stack StackV0) pendingCapacityOutboxEntriesForReconcileV0(
	ctx context.Context,
	runID string,
) ([]orquestaoutboxdispatch.OutboxPendingEntryV0, error) {
	pending, issues := stack.Stores.OutboxLedger.ListPending(ctx, orquestadirectorcycleoutbox.DirectorCycleOutboxPendingFilterV0{
		RunRef:     strings.TrimSpace(runID),
		TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
	})
	if len(issues) > 0 {
		return nil, fmt.Errorf("capacity_outbox_reconcile: list_pending: %s", strings.TrimSpace(issues[0].Code))
	}
	entries := make([]orquestaoutboxdispatch.OutboxPendingEntryV0, 0, len(pending))
	for _, message := range pending {
		if strings.TrimSpace(message.MessageType) != orquestacoreworkflow.OutboxMessageRequestCapacityDecisionV0 {
			continue
		}
		entries = append(entries, orquestaoutboxdispatch.OutboxPendingEntryV0{
			MessageID:      strings.TrimSpace(message.MessageID),
			RunID:          strings.TrimSpace(message.RunID),
			TargetPort:     strings.TrimSpace(message.TargetPort),
			MessageType:    strings.TrimSpace(message.MessageType),
			IdempotencyKey: strings.TrimSpace(message.IdempotencyKey),
			CorrelationID:  strings.TrimSpace(message.CorrelationID),
			PayloadVersion: strings.TrimSpace(message.PayloadVersion),
			Payload:        append(message.Payload[:0:0], message.Payload...),
		})
	}
	return entries, nil
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

func requestCapacityCommandFromOutboxEntryV0(
	entry orquestaoutboxdispatch.OutboxPendingEntryV0,
	payload orquestacoreworkflow.CapacityDecisionRequestV0,
	occurredAt string,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	originalIdempotencyKey := strings.TrimSpace(entry.IdempotencyKey)
	idempotencyKey := "idem-reconcile-capacity-outbox-" + strings.TrimSpace(entry.MessageID)
	commandID := "cmd-reconcile-capacity-outbox-" + strings.TrimSpace(entry.MessageID)
	return orquestacoreworkflow.NewRequestCapacityCommandV0(orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          strings.TrimSpace(entry.RunID),
		IdempotencyKey: idempotencyKey,
		CorrelationID:  strings.TrimSpace(entry.CorrelationID),
		RequestedBy:    codexStackLaunchOutboxRecoveryRequestedByV0,
		OccurredAt:     strings.TrimSpace(occurredAt),
	}, orquestacoreworkflow.RequestCapacityCommandPayloadV0{
		CapacityRequestID:          strings.TrimSpace(payload.CapacityRequestID),
		PhaseID:                    strings.TrimSpace(payload.PhaseID),
		TaskRef:                    strings.TrimSpace(payload.TaskRef),
		ReasonCode:                 strings.TrimSpace(payload.ReasonCode),
		Summary:                    strings.TrimSpace(payload.Summary),
		MinimumRecommendedCapacity: payload.MinimumRecommendedCapacity,
		EvidenceRefs: compactStringsV0(append(
			payload.EvidenceRefs,
			"evidence-ref-capacity-outbox-orphan-reconciled",
			"evidence-ref-original-idempotency-"+originalIdempotencyKey,
		)),
	})
}

func (stack StackV0) releaseOutboxDispatchClaimV0(
	entry orquestaoutboxdispatch.OutboxPendingEntryV0,
) []orquestaoutboxdispatch.DispatchIssueV0 {
	releaser, ok := stack.Stores.OutboxLedger.(interface {
		ReleaseOutboxDispatchClaimV0(orquestaoutboxdispatch.OutboxDispatchClaimV0) []orquestaoutboxdispatch.DispatchIssueV0
	})
	if !ok || releaser == nil {
		return nil
	}
	return releaser.ReleaseOutboxDispatchClaimV0(orquestaoutboxdispatch.OutboxDispatchClaimV0{
		MessageID:      strings.TrimSpace(entry.MessageID),
		RunID:          strings.TrimSpace(entry.RunID),
		TargetPort:     strings.TrimSpace(entry.TargetPort),
		IdempotencyKey: strings.TrimSpace(entry.IdempotencyKey),
	})
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

func agentStartedEventsByAgentRefV0(
	runRef string,
	events []orquestacoreworkflow.OrchestrationEventV0,
) map[string]orquestacoreworkflow.OrchestrationEventV0 {
	out := map[string]orquestacoreworkflow.OrchestrationEventV0{}
	for _, event := range events {
		if strings.TrimSpace(event.RunID) != strings.TrimSpace(runRef) ||
			event.EventType != orquestacoreworkflow.OrchestrationEventAgentStartedV0 {
			continue
		}
		var payload orquestacoreworkflow.AgentStartedPayloadV0
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
