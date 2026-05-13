package orquestaservershutdown

import "strings"

const serverShutdownCheckpointDeadlineEvidenceV0 = "evidence-ref-shutdown-checkpoint-deadline-expired"

func checkpointDeadlineExpiredV0(command ServerShutdownCommandV0) bool {
	if command.Forced || command.CheckpointDeadlineAt.IsZero() || command.OccurredAt.IsZero() {
		return false
	}
	return !command.OccurredAt.Before(command.CheckpointDeadlineAt)
}

func forcedShutdownCommandAfterCheckpointDeadlineV0(
	command ServerShutdownCommandV0,
) ServerShutdownCommandV0 {
	command.Forced = true
	command.EvidenceRefs = compactServerShutdownStringsV0(append(
		command.EvidenceRefs,
		serverShutdownCheckpointDeadlineEvidenceV0,
	))
	reason := strings.TrimSpace(command.Reason)
	if reason == "" {
		command.Reason = "shutdown checkpoint deadline expired"
		return command
	}
	command.Reason = reason + "; shutdown checkpoint deadline expired"
	return command
}
