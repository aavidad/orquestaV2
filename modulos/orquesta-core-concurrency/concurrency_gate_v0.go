package orquestacoreconcurrency

import (
	"strconv"
	"strings"
)

const ConcurrencyGateSchemaVersionV0 = "concurrency_gate.v0"

type ConcurrencyGateDecisionV0 string

const (
	ConcurrencyGateDecisionAllowRequestAgentV0 ConcurrencyGateDecisionV0 = "allow_request_agent"
	ConcurrencyGateDecisionBlockRequestAgentV0 ConcurrencyGateDecisionV0 = "block_request_agent"
	ConcurrencyGateDecisionAskDirectorV0       ConcurrencyGateDecisionV0 = "ask_director"
)

type ConcurrencyGateEvaluationV0 struct {
	SchemaVersion          string                    `json:"schema_version"`
	GateRef                string                    `json:"gate_ref"`
	RunRef                 string                    `json:"run_ref,omitempty"`
	PlanRef                string                    `json:"plan_ref"`
	SubjectClaimRefs       []string                  `json:"subject_claim_refs"`
	ReadyClaimRefs         []string                  `json:"ready_claim_refs,omitempty"`
	BlockedClaimRefs       []string                  `json:"blocked_claim_refs,omitempty"`
	HardBlockedClaimRefs   []string                  `json:"hard_blocked_claim_refs,omitempty"`
	ConflictRefs           []string                  `json:"conflict_refs,omitempty"`
	RepairableConflictRefs []string                  `json:"repairable_conflict_refs,omitempty"`
	SequenceClaimRefs      []string                  `json:"sequence_claim_refs,omitempty"`
	EvidenceRefs           []string                  `json:"evidence_refs,omitempty"`
	Decision               ConcurrencyGateDecisionV0 `json:"decision"`
	Summary                string                    `json:"summary"`
}

func EvaluateConcurrencyGateV0(claims []WorksetClaimV0, subjectClaimRefs []string) ConcurrencyGateEvaluationV0 {
	plan := EvaluateParallelGroupsV0(claims)
	subjects := normalizeOpaqueRefsV0(subjectClaimRefs)
	decision := concurrencyGateDecisionV0(plan, subjects)

	return ConcurrencyGateEvaluationV0{
		SchemaVersion:          ConcurrencyGateSchemaVersionV0,
		GateRef:                concurrencyGateRefV0(plan.PlanRef, subjects),
		RunRef:                 plan.RunRef,
		PlanRef:                plan.PlanRef,
		SubjectClaimRefs:       subjects,
		ReadyClaimRefs:         plan.ReadyClaimRefs,
		BlockedClaimRefs:       plan.BlockedClaimRefs,
		HardBlockedClaimRefs:   plan.HardBlockedClaimRefs,
		ConflictRefs:           plan.ConflictRefs,
		RepairableConflictRefs: plan.RepairableConflictRefs,
		SequenceClaimRefs:      plan.SequenceClaimRefs,
		EvidenceRefs:           plan.EvidenceRefs,
		Decision:               decision,
		Summary:                concurrencyGateSummaryV0(decision, subjects, plan),
	}
}

func concurrencyGateDecisionV0(plan ParallelGroupPlanV0, subjects []string) ConcurrencyGateDecisionV0 {
	if len(subjects) == 0 {
		return ConcurrencyGateDecisionAskDirectorV0
	}

	ready := stringSetV0(plan.ReadyClaimRefs)
	blocked := stringSetV0(plan.BlockedClaimRefs)
	for _, subject := range subjects {
		if blocked[subject] {
			return ConcurrencyGateDecisionBlockRequestAgentV0
		}
		if !ready[subject] {
			return ConcurrencyGateDecisionAskDirectorV0
		}
	}
	return ConcurrencyGateDecisionAllowRequestAgentV0
}

func concurrencyGateRefV0(planRef string, subjects []string) string {
	subjectKey := strings.Join(subjects, "+")
	if subjectKey == "" {
		subjectKey = "claims:none"
	}
	return "concurrency_gate:" + planRef + ":" + shortConcurrencyDigestV0(subjectKey)
}

func concurrencyGateSummaryV0(decision ConcurrencyGateDecisionV0, subjects []string, plan ParallelGroupPlanV0) string {
	return "concurrency_gate decision=" + string(decision) +
		" subjects=" + strconv.Itoa(len(subjects)) +
		" ready=" + strconv.Itoa(len(plan.ReadyClaimRefs)) +
		" blocked=" + strconv.Itoa(len(plan.BlockedClaimRefs)) +
		" conflicts=" + strconv.Itoa(len(plan.ConflictRefs)) +
		" repairable_conflicts=" + strconv.Itoa(len(plan.RepairableConflictRefs)) +
		" hard_blocked=" + strconv.Itoa(len(plan.HardBlockedClaimRefs))
}

func stringSetV0(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}
