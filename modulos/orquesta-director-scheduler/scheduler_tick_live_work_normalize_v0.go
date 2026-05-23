package orquestadirectorscheduler

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func normalizeSchedulerLiveWorkPolicyV0(
	policy *SchedulerLiveWorkSequencePolicyV0,
) *SchedulerLiveWorkSequencePolicyV0 {
	if policy == nil {
		return nil
	}
	return &SchedulerLiveWorkSequencePolicyV0{
		OnOverlap: SchedulerWorkSequenceActionV0(strings.TrimSpace(string(policy.OnOverlap))),
		ReviewTaskCandidate: normalizeSchedulerCreateMicrotaskCandidateV0(
			policy.ReviewTaskCandidate,
		),
		StudyTaskCandidate: normalizeSchedulerCreateMicrotaskCandidateV0(
			policy.StudyTaskCandidate,
		),
		EvidenceRefs: compactSchedulerStringsV0(policy.EvidenceRefs),
	}
}

func normalizeSchedulerCreateMicrotaskCandidateV0(
	candidate *SchedulerCreateMicrotaskCandidateV0,
) *SchedulerCreateMicrotaskCandidateV0 {
	if candidate == nil {
		return nil
	}
	return &SchedulerCreateMicrotaskCandidateV0{
		CommandMeta: normalizeSchedulerCommandMetaV0(candidate.CommandMeta),
		Payload: orquestacoreworkflow.CreateMicrotaskCommandPayloadV0{
			Task: candidate.Payload.Task,
		},
	}
}
