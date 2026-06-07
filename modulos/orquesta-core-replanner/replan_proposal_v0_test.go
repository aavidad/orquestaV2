package orquestacorereplanner

import (
	"strings"
	"testing"
)

func TestNewReplanProposalV0AcceptsOpaqueRefs(t *testing.T) {
	proposal := validReplanProposalV0()
	proposal.ReplanRef = " replan-001 "
	proposal.EvidenceRefs = []string{" review-result-001 ", "assessment-compact-001"}

	got, err := NewReplanProposalV0(proposal)
	if err != nil {
		t.Fatalf("NewReplanProposalV0: %v", err)
	}
	if got.ReplanRef != "replan-001" {
		t.Fatalf("replan_ref=%q", got.ReplanRef)
	}
	if got.EvidenceRefs[0] != "review-result-001" {
		t.Fatalf("evidence ref was not normalized: %#v", got.EvidenceRefs)
	}
}

func TestValidateReplanProposalV0SupportsContractActions(t *testing.T) {
	for _, action := range []ReplanRecommendedActionV0{
		ReplanActionSplitTaskV0,
		ReplanActionRetryTaskV0,
		ReplanActionReplaceAgentV0,
		ReplanActionEscalateCapacityV0,
		ReplanActionAskDirectorV0,
		ReplanActionAbortTaskV0,
	} {
		t.Run(string(action), func(t *testing.T) {
			proposal := validReplanProposalV0()
			proposal.RecommendedAction = action
			if action == ReplanActionReplaceAgentV0 {
				proposal.ReplacementRole = "reviewer"
			}
			if action == ReplanActionEscalateCapacityV0 {
				proposal.CapacityRequestRef = "capacity-request-001"
			}

			if _, err := NewReplanProposalV0(proposal); err != nil {
				t.Fatalf("NewReplanProposalV0: %v", err)
			}
		})
	}
}

func TestValidateReplanProposalV0AllowsOperationalLabels(t *testing.T) {
	cases := map[string]func(*ReplanProposalV0){
		"DB":          func(proposal *ReplanProposalV0) { proposal.ReplanRef = "db-replan" },
		"provider":    func(proposal *ReplanProposalV0) { proposal.Summary = "depende de provider externo" },
		"model":       func(proposal *ReplanProposalV0) { proposal.ReasonCode = "model_limit" },
		"HOME":        func(proposal *ReplanProposalV0) { proposal.EvidenceRefs = []string{"$HOME/replan.txt"} },
		"OAuth":       func(proposal *ReplanProposalV0) { proposal.SourceRef = "oauth-source" },
		"runtime":     func(proposal *ReplanProposalV0) { proposal.Summary = "runtime log ref incluido" },
		"prompts":     func(proposal *ReplanProposalV0) { proposal.Summary = "prompt policy ref sin contenido crudo" },
		"transcripts": func(proposal *ReplanProposalV0) { proposal.EvidenceRefs = []string{"transcripts/policy-ref"} },
		"diffs":       func(proposal *ReplanProposalV0) { proposal.Summary = "diff ref compacto" },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			proposal := validReplanProposalV0()
			mutate(&proposal)

			err := ValidateReplanProposalV0(NormalizeReplanProposalV0(proposal))
			if err != nil {
				t.Fatalf("ValidateReplanProposalV0: %v", err)
			}
		})
	}
}

func TestValidateReplanProposalV0RejectsSensitiveDetails(t *testing.T) {
	cases := map[string]func(*ReplanProposalV0){
		"api_key":       func(proposal *ReplanProposalV0) { proposal.Summary = "depende de api_key=valor" },
		"authorization": func(proposal *ReplanProposalV0) { proposal.EvidenceRefs = []string{"authorization: bearer valor"} },
		"dsn":           func(proposal *ReplanProposalV0) { proposal.SourceRef = "postgres://user:pass@host/db" },
		"raw_prompt":    func(proposal *ReplanProposalV0) { proposal.Summary = "prompt=raw payload" },
		"raw_transcript": func(proposal *ReplanProposalV0) {
			proposal.EvidenceRefs = []string{"transcript=raw text"}
		},
		"real_path": func(proposal *ReplanProposalV0) { proposal.EvidenceRefs = []string{"/home/operator/.config/orquesta"} },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			proposal := validReplanProposalV0()
			mutate(&proposal)

			err := ValidateReplanProposalV0(NormalizeReplanProposalV0(proposal))
			assertReplanProposalErrorCodeV0(t, err, ErrDetalleProhibidoV0)
		})
	}
}

func TestValidateReplanProposalV0RejectsInvalidShape(t *testing.T) {
	cases := map[string]func(*ReplanProposalV0){
		"replan_ref":    func(proposal *ReplanProposalV0) { proposal.ReplanRef = " " },
		"run_ref":       func(proposal *ReplanProposalV0) { proposal.RunRef = " " },
		"task_ref":      func(proposal *ReplanProposalV0) { proposal.TaskRef = " " },
		"source_ref":    func(proposal *ReplanProposalV0) { proposal.SourceRef = " " },
		"reason_code":   func(proposal *ReplanProposalV0) { proposal.ReasonCode = " " },
		"summary":       func(proposal *ReplanProposalV0) { proposal.Summary = " " },
		"evidence_refs": func(proposal *ReplanProposalV0) { proposal.EvidenceRefs = []string{"evidence-001", " "} },
		"capacity_request_ref": func(proposal *ReplanProposalV0) {
			proposal.RecommendedAction = ReplanActionEscalateCapacityV0
			proposal.CapacityRequestRef = " "
		},
		"replacement_role": func(proposal *ReplanProposalV0) {
			proposal.RecommendedAction = ReplanActionReplaceAgentV0
			proposal.ReplacementRole = " "
		},
	}

	for field, mutate := range cases {
		t.Run(field, func(t *testing.T) {
			proposal := validReplanProposalV0()
			mutate(&proposal)

			err := ValidateReplanProposalV0(NormalizeReplanProposalV0(proposal))
			assertReplanProposalErrorV0(t, err, ErrReplanProposalInvalidoV0, field)
		})
	}
}

func TestValidateReplanProposalV0RejectsUnknownAction(t *testing.T) {
	proposal := validReplanProposalV0()
	proposal.RecommendedAction = ReplanRecommendedActionV0("restart_runtime")

	err := ValidateReplanProposalV0(NormalizeReplanProposalV0(proposal))
	assertReplanProposalErrorV0(t, err, ErrReplanActionNoSoportadaV0, "recommended_action")
}

func TestValidateReplanProposalV0RejectsMassivePayload(t *testing.T) {
	proposal := validReplanProposalV0()
	proposal.EvidenceRefs = nil
	for index := 0; index < maxReplanProposalEvidenceRefsV0; index++ {
		proposal.EvidenceRefs = append(proposal.EvidenceRefs, "evidence-"+strings.Repeat("x", 160))
	}

	err := ValidateReplanProposalV0(NormalizeReplanProposalV0(proposal))
	assertReplanProposalErrorV0(t, err, ErrReplanProposalPayloadInvalidoV0, "payload")
}

func TestReplanProposalV0JSONHasNoForbiddenDetails(t *testing.T) {
	proposal := mustReplanProposalV0(t, validReplanProposalV0())

	assertReplanJSONHasNoForbiddenDetailsV0(t, "replan proposal", proposal)
}
