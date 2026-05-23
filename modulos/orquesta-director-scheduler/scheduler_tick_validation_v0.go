package orquestadirectorscheduler

import (
	"encoding/json"
	"strings"

	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
)

const (
	maxSchedulerTickPayloadBytesV0 = 262144
	maxSchedulerTickRefsV0         = 1024
)

var forbiddenSchedulerFragmentsV0 = []string{
	"db", "database", "sql", "dsn", "runtime",
	"provider", "proveedor", "model", "modelo",
	"home", "oauth", "prompt", "prompts",
	"transcript", "transcripts", "secret", "secrets",
	"token", "password", "credential", "credencial",
	"api_key", "filesystem", "docker", "tmux",
}

func ValidateDirectorSchedulerTickInputV0(input DirectorSchedulerTickInputV0) error {
	if err := validateSchedulerRequiredV0(input); err != nil {
		return err
	}
	if err := validateSchedulerRefsV0(input); err != nil {
		return err
	}
	if err := validateSchedulerCandidatesV0(input); err != nil {
		return err
	}
	if err := validateSchedulerLeaseCandidatesV0(input); err != nil {
		return err
	}
	if err := validateSchedulerPhaseArtifactCandidatesV0(input); err != nil {
		return err
	}
	if err := validateSchedulerDeliveryCandidatesV0(input); err != nil {
		return err
	}
	if err := validateSchedulerReviewGateCandidatesV0(input); err != nil {
		return err
	}
	if err := validateSchedulerProgressSupervisionCandidatesV0(input); err != nil {
		return err
	}
	if err := validateSchedulerReplanFollowupCandidatesV0(input); err != nil {
		return err
	}
	if schedulerContainsForbiddenDetailsV0(schedulerOperationalDetailFieldsV0(input)) {
		return schedulerTickErrorV0("payload")
	}
	return validateSchedulerCompactPayloadV0(input)
}

func validateSchedulerRequiredV0(input DirectorSchedulerTickInputV0) error {
	fields := map[string]string{
		"tick_ref":                  input.TickRef,
		"run_ref":                   input.RunRef,
		"occurred_at":               input.OccurredAt,
		"snapshot.run_ref":          input.Snapshot.RunRef,
		"snapshot.current_phase_id": input.Snapshot.CurrentPhaseID,
	}
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return schedulerTickErrorV0(field)
		}
	}
	if input.Snapshot.RunRef != input.RunRef {
		return schedulerTickErrorV0("snapshot.run_ref")
	}
	return nil
}

func validateSchedulerRefsV0(input DirectorSchedulerTickInputV0) error {
	// El scheduler no debe cortar por paths/evidencias opacas de agentes. Las
	// relaciones causales se validan en los candidatos que emiten comandos.
	return nil
}

func validateSchedulerCandidatesV0(input DirectorSchedulerTickInputV0) error {
	if err := validateSchedulerWorkClaimsV0(input); err != nil {
		return err
	}
	for index, candidate := range input.WorkCandidates {
		if strings.TrimSpace(candidate.CandidateRef) == "" {
			return schedulerTickErrorV0("work_candidates.candidate_ref")
		}
		if len(candidate.SubjectClaimRefs) == 0 {
			return schedulerTickErrorV0("work_candidates.claims")
		}
		if len(candidate.Claims) == 0 && len(input.WorkClaims) == 0 {
			return schedulerTickErrorV0("work_candidates.claims")
		}
		for _, claim := range candidate.Claims {
			if issues := orquestacoreconcurrency.ValidateWorksetClaimV0(claim); len(issues) > 0 {
				return schedulerTickErrorV0("work_candidates.claims")
			}
		}
		if len(input.WorkClaims) > 0 && !schedulerSubjectsInClaimsV0(candidate.SubjectClaimRefs, input.WorkClaims) {
			return schedulerTickErrorV0("work_candidates.subject_claim_refs")
		}
		if index > maxSchedulerTickRefsV0 {
			return schedulerTickErrorV0("work_candidates")
		}
		if schedulerRefsInvalidV0(candidate.SubjectClaimRefs) ||
			schedulerRefsInvalidV0(candidate.GateEvidenceRefs) ||
			schedulerRefsInvalidV0(candidate.EvidenceRefs) {
			return schedulerTickErrorV0("work_candidates.refs")
		}
		if err := validateSchedulerCandidateEvidenceRefsV0(candidate); err != nil {
			return err
		}
		if err := validateSchedulerCandidateRunRefsV0(input.RunRef, candidate); err != nil {
			return err
		}
		if err := validateSchedulerLiveWorkPolicyV0(input.RunRef, candidate.LiveWorkPolicy); err != nil {
			return err
		}
	}
	return nil
}

func validateSchedulerWorkClaimsV0(input DirectorSchedulerTickInputV0) error {
	if len(input.WorkClaims) > maxSchedulerTickRefsV0 {
		return schedulerTickErrorV0("work_claims")
	}
	for _, claim := range input.WorkClaims {
		if issues := orquestacoreconcurrency.ValidateWorksetClaimV0(claim); len(issues) > 0 {
			return schedulerTickErrorV0("work_claims")
		}
		if strings.TrimSpace(claim.RunRef) != input.RunRef {
			return schedulerTickErrorV0("work_claims.run_ref")
		}
	}
	return nil
}

func schedulerSubjectsInClaimsV0(
	subjects []string,
	claims []orquestacoreconcurrency.WorksetClaimV0,
) bool {
	claimRefs := map[string]bool{}
	for _, claim := range claims {
		claimRefs[strings.TrimSpace(claim.ClaimRef)] = true
	}
	for _, subject := range subjects {
		if !claimRefs[strings.TrimSpace(subject)] {
			return false
		}
	}
	return true
}

func validateSchedulerCandidateRunRefsV0(runRef string, candidate SchedulableWorkCandidateV0) error {
	for _, claim := range candidate.Claims {
		if strings.TrimSpace(claim.RunRef) != runRef {
			return schedulerTickErrorV0("work_candidates.claims.run_ref")
		}
	}
	if candidate.CapacityCandidate != nil && candidate.CapacityCandidate.CommandMeta.RunID != runRef {
		return schedulerTickErrorV0("work_candidates.capacity_candidate.command_meta.run_id")
	}
	if candidate.AgentCandidate != nil && candidate.AgentCandidate.CommandMeta.RunID != runRef {
		return schedulerTickErrorV0("work_candidates.agent_candidate.command_meta.run_id")
	}
	if candidate.GateCommandMeta.RunID != runRef {
		return schedulerTickErrorV0("work_candidates.gate_command_meta.run_id")
	}
	return nil
}

func validateSchedulerCandidateEvidenceRefsV0(candidate SchedulableWorkCandidateV0) error {
	if candidate.CapacityCandidate != nil &&
		schedulerRefsInvalidV0(candidate.CapacityCandidate.Payload.EvidenceRefs) {
		return schedulerTickErrorV0("work_candidates.capacity_candidate.evidence_refs")
	}
	if candidate.AgentCandidate != nil &&
		schedulerRefsInvalidV0(candidate.AgentCandidate.Payload.EvidenceRefs) {
		return schedulerTickErrorV0("work_candidates.agent_candidate.evidence_refs")
	}
	return nil
}

func schedulerRefsInvalidV0(refs []string) bool {
	// Modo permisivo por defecto: refs ambiguas siguen al director/rework.
	return false
}

func schedulerOperationalDetailFieldsV0(input DirectorSchedulerTickInputV0) []string {
	var values []string
	for _, candidate := range input.WorkCandidates {
		if candidate.CapacityCandidate != nil {
			values = append(values,
				candidate.CapacityCandidate.Payload.ReasonCode,
				candidate.CapacityCandidate.Payload.Summary,
			)
		}
		if candidate.AgentCandidate != nil {
			values = append(values,
				candidate.AgentCandidate.Payload.Role,
				candidate.AgentCandidate.Payload.Summary,
			)
		}
	}
	values = append(values, schedulerLeaseCandidatesTextFieldsV0(input.LeaseActionCandidates)...)
	values = append(values, schedulerPhaseArtifactCandidatesTextFieldsV0(input.PhaseArtifactCandidates)...)
	values = append(values, schedulerDeliveryCandidatesTextFieldsV0(input.DeliveryCandidates)...)
	values = append(values, schedulerReviewGateCandidatesTextFieldsV0(input.ReviewGateCandidates)...)
	values = append(values, schedulerProgressSupervisionCandidatesTextFieldsV0(input.ProgressSupervisionCandidates)...)
	values = append(values, schedulerReplanFollowupCandidatesTextFieldsV0(input.ReplanFollowupCandidates)...)
	return values
}

func validateSchedulerCompactPayloadV0(input DirectorSchedulerTickInputV0) error {
	data, err := json.Marshal(input)
	if err != nil {
		return schedulerTickErrorV0("payload")
	}
	if len(data) == 0 || len(data) > maxSchedulerTickPayloadBytesV0 {
		return schedulerTickErrorV0("payload")
	}
	return nil
}

func schedulerTickErrorV0(field string) DirectorSchedulerTickErrorV0 {
	return DirectorSchedulerTickErrorV0{Code: ErrDirectorSchedulerTickInvalidoV0, Field: field}
}
