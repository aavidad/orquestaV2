package orquestadirectorscheduler

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func validateSchedulerLiveWorkPolicyV0(runRef string, policy *SchedulerLiveWorkSequencePolicyV0) error {
	if policy == nil {
		return nil
	}
	switch policy.OnOverlap {
	case "",
		SchedulerWorkSequenceActionWaitLiveWorkV0,
		SchedulerWorkSequenceActionQueueAfterLiveWorkV0:
	case SchedulerWorkSequenceActionCreateReviewTaskV0:
		if policy.ReviewTaskCandidate == nil {
			return schedulerTickErrorV0("work_candidates.live_work_policy.review_task_candidate")
		}
	case SchedulerWorkSequenceActionCreateStudyTaskV0:
		if policy.StudyTaskCandidate == nil {
			return schedulerTickErrorV0("work_candidates.live_work_policy.study_task_candidate")
		}
	default:
		return schedulerTickErrorV0("work_candidates.live_work_policy.on_overlap")
	}
	if err := validateSchedulerCreateMicrotaskCandidateV0(runRef, policy.ReviewTaskCandidate); err != nil {
		return err
	}
	if err := validateSchedulerCreateMicrotaskCandidateV0(runRef, policy.StudyTaskCandidate); err != nil {
		return err
	}
	if schedulerRefsInvalidV0(policy.EvidenceRefs) {
		return schedulerTickErrorV0("work_candidates.live_work_policy.evidence_refs")
	}
	return nil
}

func validateSchedulerCreateMicrotaskCandidateV0(
	runRef string,
	candidate *SchedulerCreateMicrotaskCandidateV0,
) error {
	if candidate == nil {
		return nil
	}
	if strings.TrimSpace(candidate.CommandMeta.RunID) != runRef {
		return schedulerTickErrorV0("work_candidates.live_work_policy.command_meta.run_id")
	}
	if _, err := orquestacoreworkflow.NewCreateMicrotaskCommandV0(
		candidate.CommandMeta,
		candidate.Payload,
	); err != nil {
		return schedulerTickErrorV0("work_candidates.live_work_policy.create_microtask")
	}
	return nil
}
