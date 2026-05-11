package orquestadirectorcycleoutbox

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func normalizeDirectorCycleOutboxInputV0(
	input DirectorCycleOutboxRecordInputV0,
) DirectorCycleOutboxRecordInputV0 {
	input.RunRef = strings.TrimSpace(input.RunRef)
	input.TargetPort = strings.TrimSpace(input.TargetPort)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.Messages = cloneCycleOutboxMessagesV0(input.Messages)
	return input
}

func newDirectorCycleOutboxResultV0(
	input DirectorCycleOutboxRecordInputV0,
) DirectorCycleOutboxRecordResultV0 {
	return DirectorCycleOutboxRecordResultV0{
		RunRef:     input.RunRef,
		TargetPort: input.TargetPort,
	}
}

func cycleOutboxMessageRefsV0(messages []orquestacoreworkflow.OutboxMessageV0) []string {
	if len(messages) == 0 {
		return nil
	}
	refs := make([]string, 0, len(messages))
	for _, message := range messages {
		refs = append(refs, message.MessageID)
	}
	return compactCycleOutboxStringsV0(refs)
}

func compactCycleOutboxStringsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
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
	return out
}

func cloneCycleOutboxMessagesV0(
	messages []orquestacoreworkflow.OutboxMessageV0,
) []orquestacoreworkflow.OutboxMessageV0 {
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
