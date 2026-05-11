package orquestaruncontrol

import "strings"

func NormalizeCompleteRunControlCommandV0(
	command CompleteRunControlCommandV0,
) CompleteRunControlCommandV0 {
	command.RunRef = strings.TrimSpace(command.RunRef)
	command.TargetStatus = NormalizeRunControlStatusV0(command.TargetStatus)
	command.RequestedBy = strings.TrimSpace(command.RequestedBy)
	command.Reason = strings.TrimSpace(command.Reason)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.EvidenceRefs = compactRunControlStringsV0(command.EvidenceRefs)
	return command
}

func IsCompleteRunControlTargetV0(status RunControlStatusV0) bool {
	switch NormalizeRunControlStatusV0(status) {
	case RunControlStatusStoppedV0, RunControlStatusCanceledV0:
		return true
	default:
		return false
	}
}
