package orquestaruncontrol

import "strings"

func NormalizeRecordRunCheckpointCommandV0(
	command RecordRunCheckpointCommandV0,
) RecordRunCheckpointCommandV0 {
	command.RunRef = strings.TrimSpace(command.RunRef)
	command.RequestedBy = strings.TrimSpace(command.RequestedBy)
	command.Reason = strings.TrimSpace(command.Reason)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.EvidenceRefs = compactRunControlStringsV0(command.EvidenceRefs)
	return command
}
