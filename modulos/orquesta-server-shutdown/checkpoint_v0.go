package orquestaservershutdown

import (
	"context"
	"strings"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func prepareShutdownCheckpointV0(
	ctx context.Context,
	deps ServerShutdownDepsV0,
	command ServerShutdownCommandV0,
	candidate orquestarunqueue.RunSchedulingCandidateV0,
	state orquestaruncontrol.RunControlStateV0,
) (orquestaruncontrol.RunControlStateV0, PrepareAgentShutdownResultV0, bool, error) {
	if deps.CheckpointPreparer == nil || deps.RunCheckpointWriter == nil {
		return state, PrepareAgentShutdownResultV0{}, false, nil
	}
	prepared, err := deps.CheckpointPreparer.PrepareAgentShutdownV0(
		ctx,
		PrepareAgentShutdownCommandV0{
			RunRef:        strings.TrimSpace(candidate.RunRef),
			AppRef:        strings.TrimSpace(candidate.AppRef),
			RequestedBy:   command.RequestedBy,
			Reason:        command.Reason,
			CorrelationID: command.CorrelationID,
			EvidenceRefs:  command.EvidenceRefs,
			OccurredAt:    command.OccurredAt,
		},
	)
	if err != nil {
		return state, PrepareAgentShutdownResultV0{}, false, err
	}
	checkpointRef := strings.TrimSpace(prepared.CheckpointRef)
	if !prepared.CheckpointRecorded {
		prepared.CheckpointRef = checkpointRef
		prepared.PendingAgentRefs = compactServerShutdownStringsV0(prepared.PendingAgentRefs)
		prepared.EvidenceRefs = compactServerShutdownStringsV0(prepared.EvidenceRefs)
		return state, prepared, false, nil
	}
	refs := append([]string(nil), command.EvidenceRefs...)
	refs = append(refs, prepared.EvidenceRefs...)
	if checkpointRef != "" {
		refs = append(refs, checkpointRef)
	}
	recorded, err := deps.RunCheckpointWriter.RecordRunCheckpointV0(
		ctx,
		orquestaruncontrol.RecordRunCheckpointCommandV0{
			RunRef:         strings.TrimSpace(candidate.RunRef),
			RequestedBy:    command.RequestedBy,
			Reason:         "shutdown checkpoint recorded",
			IdempotencyKey: "idem-shutdown-checkpoint-" + safeServerShutdownRefPartV0(candidate.RunRef),
			EvidenceRefs:   compactServerShutdownStringsV0(refs),
		},
	)
	if err != nil {
		return state, PrepareAgentShutdownResultV0{}, false, err
	}
	prepared.CheckpointRef = checkpointRef
	prepared.EvidenceRefs = compactServerShutdownStringsV0(prepared.EvidenceRefs)
	return recorded, prepared, true, nil
}

func safeServerShutdownRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return "unknown"
	}
	return value
}
