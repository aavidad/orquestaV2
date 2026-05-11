package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func outboxRefsV0(
	messages []orquestacoreworkflow.OutboxMessageV0,
) []string {
	refs := make([]string, 0, len(messages))
	for _, message := range messages {
		refs = append(refs, strings.TrimSpace(message.MessageID))
	}
	return compactStringsV0(refs)
}
