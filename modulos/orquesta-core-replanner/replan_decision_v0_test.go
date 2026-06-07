package orquestacorereplanner

import (
	"reflect"
	"testing"
)

func TestNewReplanDecisionV0AcceptsOpaqueRefsAndClonesSlices(t *testing.T) {
	decision := validReplanDecisionV0()
	decision.ReplanRef = " replan-001 "
	decision.FollowupRefs = []string{" rework-001 ", "agent-request-002"}
	decision.EvidenceRefs = []string{" proposal-001 "}

	got, err := NewReplanDecisionV0(decision)
	if err != nil {
		t.Fatalf("NewReplanDecisionV0: %v", err)
	}
	decision.FollowupRefs[0] = "changed"
	decision.EvidenceRefs[0] = "changed"

	if got.ReplanRef != "replan-001" {
		t.Fatalf("replan_ref=%q", got.ReplanRef)
	}
	if got.FollowupRefs[0] != "rework-001" || got.EvidenceRefs[0] != "proposal-001" {
		t.Fatalf("slices were not normalized and cloned: %#v", got)
	}
}

func TestValidateReplanDecisionV0SupportsCatalogActions(t *testing.T) {
	for _, action := range []ReplanRecommendedActionV0{
		ReplanActionSplitTaskV0,
		ReplanActionRetryTaskV0,
		ReplanActionReplaceAgentV0,
		ReplanActionEscalateCapacityV0,
		ReplanActionAskDirectorV0,
		ReplanActionAbortTaskV0,
	} {
		t.Run(string(action), func(t *testing.T) {
			decision := validReplanDecisionV0()
			decision.AcceptedAction = action

			if _, err := NewReplanDecisionV0(decision); err != nil {
				t.Fatalf("NewReplanDecisionV0: %v", err)
			}
		})
	}
}

func TestValidateReplanDecisionV0RejectsRequiredRefsAndShape(t *testing.T) {
	cases := map[string]func(*ReplanDecisionV0){
		"replan_ref":      func(decision *ReplanDecisionV0) { decision.ReplanRef = " " },
		"run_ref":         func(decision *ReplanDecisionV0) { decision.RunRef = " " },
		"task_ref":        func(decision *ReplanDecisionV0) { decision.TaskRef = " " },
		"source_ref":      func(decision *ReplanDecisionV0) { decision.SourceRef = " " },
		"accepted_action": func(decision *ReplanDecisionV0) { decision.AcceptedAction = " " },
		"summary":         func(decision *ReplanDecisionV0) { decision.Summary = " " },
		"followup_refs":   func(decision *ReplanDecisionV0) { decision.FollowupRefs = nil },
		"empty_followup":  func(decision *ReplanDecisionV0) { decision.FollowupRefs = []string{"followup-001", " "} },
		"empty_evidence":  func(decision *ReplanDecisionV0) { decision.EvidenceRefs = []string{"evidence-001", " "} },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			decision := validReplanDecisionV0()
			mutate(&decision)

			err := ValidateReplanDecisionV0(NormalizeReplanDecisionV0(decision))
			field := name
			if name == "empty_followup" {
				field = "followup_refs"
			}
			if name == "empty_evidence" {
				field = "evidence_refs"
			}
			assertReplanDecisionErrorV0(t, err, ErrReplanDecisionInvalidaV0, field)
		})
	}
}

func TestValidateReplanDecisionV0RejectsUnknownAction(t *testing.T) {
	decision := validReplanDecisionV0()
	decision.AcceptedAction = ReplanRecommendedActionV0("restart_runtime")

	err := ValidateReplanDecisionV0(NormalizeReplanDecisionV0(decision))
	assertReplanDecisionErrorV0(t, err, ErrReplanDecisionActionInvalidaV0, "accepted_action")
}

func TestValidateReplanDecisionV0AllowsOperationalLabels(t *testing.T) {
	cases := map[string]func(*ReplanDecisionV0){
		"DB":          func(decision *ReplanDecisionV0) { decision.ReplanRef = "db-decision" },
		"runtime":     func(decision *ReplanDecisionV0) { decision.SourceRef = "runtime-assessment" },
		"provider":    func(decision *ReplanDecisionV0) { decision.Summary = "depende de provider externo" },
		"HOME":        func(decision *ReplanDecisionV0) { decision.FollowupRefs = []string{"$HOME/rework"} },
		"OAuth":       func(decision *ReplanDecisionV0) { decision.SourceRef = "oauth-source" },
		"modelo":      func(decision *ReplanDecisionV0) { decision.Summary = "cambiar modelo por ref de politica" },
		"secretos":    func(decision *ReplanDecisionV0) { decision.EvidenceRefs = []string{"secretos-policy-ref"} },
		"prompts":     func(decision *ReplanDecisionV0) { decision.Summary = "prompt policy ref sin contenido crudo" },
		"transcripts": func(decision *ReplanDecisionV0) { decision.EvidenceRefs = []string{"transcripts/policy-ref"} },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			decision := validReplanDecisionV0()
			mutate(&decision)

			err := ValidateReplanDecisionV0(NormalizeReplanDecisionV0(decision))
			if err != nil {
				t.Fatalf("ValidateReplanDecisionV0: %v", err)
			}
		})
	}
}

func TestValidateReplanDecisionV0RejectsSensitiveDetails(t *testing.T) {
	cases := map[string]func(*ReplanDecisionV0){
		"api_key":       func(decision *ReplanDecisionV0) { decision.Summary = "depende de api_key=valor" },
		"authorization": func(decision *ReplanDecisionV0) { decision.EvidenceRefs = []string{"authorization: bearer valor"} },
		"dsn":           func(decision *ReplanDecisionV0) { decision.SourceRef = "postgres://user:pass@host/db" },
		"raw_completion": func(decision *ReplanDecisionV0) {
			decision.Summary = "completion=raw payload"
		},
		"real_path": func(decision *ReplanDecisionV0) { decision.FollowupRefs = []string{"/home/operator/rework"} },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			decision := validReplanDecisionV0()
			mutate(&decision)

			err := ValidateReplanDecisionV0(NormalizeReplanDecisionV0(decision))
			assertReplanDecisionErrorCodeV0(t, err, ErrDetalleProhibidoV0)
		})
	}
}

func TestReplanDecisionV0IsDeterministicAndIdempotent(t *testing.T) {
	payload := validReplanDecisionV0()
	payload.ReplanRef = " replan-001 "
	payload.FollowupRefs = []string{" rework-001 ", " agent-request-002 "}

	first := mustReplanDecisionV0(t, payload)
	second := mustReplanDecisionV0(t, payload)
	third := NormalizeReplanDecisionV0(first)

	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(first, third) {
		t.Fatalf("decision is not deterministic/idempotent: first=%#v second=%#v third=%#v", first, second, third)
	}
}

func TestReplanDecisionV0JSONHasNoForbiddenDetails(t *testing.T) {
	decision := mustReplanDecisionV0(t, validReplanDecisionV0())

	assertReplanJSONHasNoForbiddenDetailsV0(t, "replan decision", decision)
}
