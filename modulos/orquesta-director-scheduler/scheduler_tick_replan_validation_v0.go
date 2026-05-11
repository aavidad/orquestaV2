package orquestadirectorscheduler

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func validateSchedulerReplanFollowupCandidatesV0(input DirectorSchedulerTickInputV0) error {
	if len(input.ReplanFollowupCandidates) > maxSchedulerTickRefsV0 {
		return schedulerTickErrorV0("replan_followup_candidates")
	}
	for _, candidate := range input.ReplanFollowupCandidates {
		if strings.TrimSpace(candidate.CandidateRef) == "" {
			return schedulerTickErrorV0("replan_followup_candidates.candidate_ref")
		}
		if schedulerRefsInvalidV0(candidate.EvidenceRefs) {
			return schedulerTickErrorV0("replan_followup_candidates.evidence_refs")
		}
		if schedulerReplanFollowupPayloadRefsInvalidV0(candidate.ReplanFollowupsInput) {
			return schedulerTickErrorV0("replan_followup_candidates.evidence_refs")
		}
		if err := validateSchedulerReplanFollowupRunRefsV0(input.RunRef, candidate); err != nil {
			return err
		}
	}
	return nil
}

func validateSchedulerReplanFollowupRunRefsV0(
	runRef string,
	candidate SchedulableReplanFollowupCandidateV0,
) error {
	input := candidate.ReplanFollowupsInput
	if strings.TrimSpace(input.DecisionCommandMeta.RunID) != runRef {
		return schedulerTickErrorV0("replan_followup_candidates.decision_command_meta.run_id")
	}
	if strings.TrimSpace(input.DecisionPayload.RunRef) != runRef {
		return schedulerTickErrorV0("replan_followup_candidates.decision_payload.run_ref")
	}
	if input.OpenPhaseCandidate != nil &&
		strings.TrimSpace(input.OpenPhaseCandidate.CommandMeta.RunID) != runRef {
		return schedulerTickErrorV0("replan_followup_candidates.open_phase_candidate.command_meta.run_id")
	}
	for _, microtask := range input.MicrotaskCandidates {
		if strings.TrimSpace(microtask.CommandMeta.RunID) != runRef {
			return schedulerTickErrorV0("replan_followup_candidates.microtask_candidates.command_meta.run_id")
		}
		if strings.TrimSpace(microtask.Payload.Task.RunID) != runRef {
			return schedulerTickErrorV0("replan_followup_candidates.microtask_candidates.payload.task.run_id")
		}
		if _, err := orquestacoreworkflow.NewCreateMicrotaskCommandV0(
			microtask.CommandMeta,
			microtask.Payload,
		); err != nil {
			return schedulerTickErrorV0("replan_followup_candidates.microtask_candidates.payload.task")
		}
	}
	if input.CapacityCandidate != nil &&
		strings.TrimSpace(input.CapacityCandidate.CommandMeta.RunID) != runRef {
		return schedulerTickErrorV0("replan_followup_candidates.capacity_candidate.command_meta.run_id")
	}
	if input.AgentCandidate != nil &&
		strings.TrimSpace(input.AgentCandidate.CommandMeta.RunID) != runRef {
		return schedulerTickErrorV0("replan_followup_candidates.agent_candidate.command_meta.run_id")
	}
	if input.AskDirectorCandidate != nil &&
		strings.TrimSpace(input.AskDirectorCandidate.CommandMeta.RunID) != runRef {
		return schedulerTickErrorV0("replan_followup_candidates.ask_director_candidate.command_meta.run_id")
	}
	return nil
}

func schedulerReplanFollowupPayloadRefsInvalidV0(input orquestadirector.ReplanFollowupsInputV0) bool {
	if schedulerRefsInvalidV0(input.DecisionPayload.EvidenceRefs) {
		return true
	}
	if input.CapacityCandidate != nil &&
		schedulerRefsInvalidV0(input.CapacityCandidate.Payload.EvidenceRefs) {
		return true
	}
	if input.AgentCandidate != nil &&
		schedulerRefsInvalidV0(input.AgentCandidate.Payload.EvidenceRefs) {
		return true
	}
	if input.AskDirectorCandidate != nil &&
		schedulerRefsInvalidV0(input.AskDirectorCandidate.Payload.EvidenceRefs) {
		return true
	}
	return false
}

func schedulerReplanFollowupCandidatesTextFieldsV0(
	candidates []SchedulableReplanFollowupCandidateV0,
) []string {
	var values []string
	for _, candidate := range candidates {
		input := candidate.ReplanFollowupsInput
		payload := input.DecisionPayload
		values = append(values,
			string(payload.AcceptedAction),
			payload.Summary,
		)
		values = append(values, schedulerReplanCandidateCommandTextFieldsV0(input)...)
	}
	return values
}

func schedulerReplanCandidateCommandTextFieldsV0(
	input orquestadirector.ReplanFollowupsInputV0,
) []string {
	var values []string
	if input.OpenPhaseCandidate != nil {
		values = append(values, input.OpenPhaseCandidate.Payload.Reason)
	}
	for _, microtask := range input.MicrotaskCandidates {
		values = append(values, schedulerReplanMicrotaskTextFieldsV0(microtask)...)
	}
	if input.CapacityCandidate != nil {
		values = append(values,
			input.CapacityCandidate.Payload.ReasonCode,
			input.CapacityCandidate.Payload.Summary,
		)
	}
	if input.AgentCandidate != nil {
		values = append(values,
			input.AgentCandidate.Payload.Role,
			input.AgentCandidate.Payload.Summary,
		)
	}
	if input.AskDirectorCandidate != nil {
		values = append(values,
			input.AskDirectorCandidate.Payload.SourceGroup,
			input.AskDirectorCandidate.Payload.TargetGroup,
			input.AskDirectorCandidate.Payload.Summary,
		)
		values = append(values, input.AskDirectorCandidate.Payload.Options...)
	}
	return values
}

func schedulerReplanMicrotaskTextFieldsV0(
	candidate orquestadirector.ReplanMicrotaskCandidateV0,
) []string {
	task := candidate.Payload.Task
	values := []string{
		task.Title,
		task.Summary,
	}
	values = append(values, task.WriteSet...)
	values = append(values, task.AcceptanceCriteria...)
	for _, ref := range task.FunctionContractRefs {
		values = append(values, ref.ContractRef, ref.FunctionName)
	}
	return values
}
