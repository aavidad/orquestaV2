package orquestaservershutdown

import "strings"

const (
	defaultServerShutdownQueueLimitV0     = 50
	defaultServerShutdownMaxTicksV0       = 8
	defaultServerShutdownMaxRunsPerTickV0 = 4
)

func NormalizeServerShutdownCommandV0(
	command ServerShutdownCommandV0,
) ServerShutdownCommandV0 {
	command.RequestID = strings.TrimSpace(command.RequestID)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.QueueRef = strings.TrimSpace(command.QueueRef)
	command.AppRefs = compactServerShutdownStringsV0(command.AppRefs)
	command.RequestedBy = strings.TrimSpace(command.RequestedBy)
	if command.RequestedBy == "" {
		command.RequestedBy = "orquesta-director"
	}
	command.Reason = strings.TrimSpace(command.Reason)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.EvidenceRefs = compactServerShutdownStringsV0(command.EvidenceRefs)
	if command.QueueLimit <= 0 {
		command.QueueLimit = defaultServerShutdownQueueLimitV0
	}
	if command.MaxTicks <= 0 {
		command.MaxTicks = defaultServerShutdownMaxTicksV0
	}
	if command.MaxRunsPerTick <= 0 {
		command.MaxRunsPerTick = defaultServerShutdownMaxRunsPerTickV0
	}
	if command.MaxExecutions < 0 {
		command.MaxExecutions = 0
	}
	return command
}
