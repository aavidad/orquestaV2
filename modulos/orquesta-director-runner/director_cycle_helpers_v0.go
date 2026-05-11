package orquestadirectorrunner

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

const (
	defaultDirectorCycleMaxCommandsV0 = 8
	defaultDirectorCycleMaxOutboxV0   = 1
)

func normalizeDirectorCycleInputV0(input DirectorCycleInputV0) DirectorCycleInputV0 {
	input.CycleRef = strings.TrimSpace(input.CycleRef)
	input.RunRef = strings.TrimSpace(input.RunRef)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.EvidenceRefs = compactDirectorCycleStringsV0(input.EvidenceRefs)
	if input.MaxCommands == 0 {
		input.MaxCommands = defaultDirectorCycleMaxCommandsV0
	}
	if input.MaxOutbox == 0 {
		input.MaxOutbox = defaultDirectorCycleMaxOutboxV0
	}
	return input
}

func newDirectorCycleResultV0(input DirectorCycleInputV0) DirectorCycleResultV0 {
	return DirectorCycleResultV0{
		CycleRef:     input.CycleRef,
		RunRef:       input.RunRef,
		EvidenceRefs: cloneDirectorCycleStringsV0(input.EvidenceRefs),
	}
}

func copySchedulerPlanToResultV0(
	result *DirectorCycleResultV0,
	plan orquestadirectorscheduler.DirectorSchedulerTickPlanV0,
) {
	result.SchedulerStatus = plan.Status
	result.SchedulerSummary = strings.TrimSpace(plan.Summary)
	result.WaitingReasons = cloneDirectorCycleWaitingReasonsV0(plan.WaitingReasons)
	result.BlockedRefs = compactDirectorCycleStringsV0(plan.BlockedRefs)
	result.EvidenceRefs = compactDirectorCycleStringsV0(append(result.EvidenceRefs, plan.EvidenceRefs...))
}

func appliedCommandFromResultV0(
	command orquestacoreworkflow.OrchestrationCommandV0,
	result orquestacoreworkflow.OrchestrationCommandResultV0,
) DirectorCycleAppliedCommandV0 {
	return DirectorCycleAppliedCommandV0{
		CommandID:   strings.TrimSpace(command.CommandID),
		CommandType: strings.TrimSpace(command.CommandType),
		EventCount:  len(result.Events),
		OutboxCount: len(result.Outbox),
	}
}

func cloneDirectorCycleOutboxV0(messages []orquestacoreworkflow.OutboxMessageV0) []orquestacoreworkflow.OutboxMessageV0 {
	if len(messages) == 0 {
		return nil
	}
	cloned := make([]orquestacoreworkflow.OutboxMessageV0, 0, len(messages))
	for _, message := range messages {
		message.Payload = append(message.Payload[:0:0], message.Payload...)
		cloned = append(cloned, message)
	}
	return cloned
}

func cloneDirectorCycleStringsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func compactDirectorCycleStringsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	compacted := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		compacted = append(compacted, trimmed)
	}
	return compacted
}

func cloneDirectorCycleWaitingReasonsV0(
	reasons []orquestadirectorscheduler.SchedulerWaitingReasonV0,
) []orquestadirectorscheduler.SchedulerWaitingReasonV0 {
	if len(reasons) == 0 {
		return nil
	}
	cloned := make([]orquestadirectorscheduler.SchedulerWaitingReasonV0, len(reasons))
	copy(cloned, reasons)
	return cloned
}
