package orquestadirectorcandidates

type CompactBacklogPlanCandidatesInputV0 struct {
	RunRef    string                         `json:"run_ref"`
	PhaseID   string                         `json:"phase_id"`
	WorkItems []CompactBacklogPlanWorkItemV0 `json:"work_items"`
}

type CompactBacklogPlanWorkItemV0 struct {
	CandidateRef     string                       `json:"candidate_ref"`
	TaskRef          string                       `json:"task_ref"`
	SubjectClaimRefs []string                     `json:"subject_claim_refs"`
	ScopeClaims      []WorkCandidateScopeClaimV0  `json:"scope_claims"`
	Commands         WorkCandidateCommandsV0      `json:"commands"`
	Capacity         WorkCandidateCapacityInputV0 `json:"capacity"`
	Agent            WorkCandidateAgentInputV0    `json:"agent"`
	GateEvidenceRefs []string                     `json:"gate_evidence_refs,omitempty"`
	EvidenceRefs     []string                     `json:"evidence_refs,omitempty"`
}
