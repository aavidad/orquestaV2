package orquestaruntimecodexappserver

import (
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

func TestMergeCodexAppServerGoalResultV0ProyectaReasonCodePlaceholderSinTextoLibreV0(t *testing.T) {
	for _, summary := range []string{
		"started",
		"checkpoint_started; implementacion pendiente",
		"in_progress_checkpoint_materialized",
		"cualquier redaccion libre distinta",
	} {
		receipt := orquestaruntimecodexgoal.CodexGoalObservationReceiptV0{}
		mergeCodexAppServerGoalResultV0(&receipt, codexAppServerGoalResultMarkerV0{
			SchemaVersion: "orquesta_goal_result.v0",
			Status:        orquestagoal.GoalStatusBlockedV0,
			GoalRef:       "goal-ref-placeholder-001",
			Summary:       summary,
		}, "evidence-ref-source-test")
		if receipt.IssueCode != codexAppServerGoalResultPlaceholderReasonCodeV0 {
			t.Fatalf("summary %q: issue_code=%q, want %q (el placeholder se discrimina por forma, no por texto)",
				summary, receipt.IssueCode, codexAppServerGoalResultPlaceholderReasonCodeV0)
		}
		if !stackStringsContainForPlaceholderTestV0(receipt.EvidenceRefs, "evidence-ref-goal-result-placeholder-in-progress") {
			t.Fatalf("summary %q: falta evidencia de placeholder: %v", summary, receipt.EvidenceRefs)
		}
	}
}

func TestMergeCodexAppServerGoalResultV0BlockedRealNoEsPlaceholderV0(t *testing.T) {
	cases := []codexAppServerGoalResultMarkerV0{
		{
			SchemaVersion: "orquesta_goal_result.v0",
			Status:        orquestagoal.GoalStatusBlockedV0,
			GoalRef:       "goal-ref-blocked-real-001",
			Summary:       "blocked por missing_required_settings",
			MissingRefs:   []string{"missing-ref-opes-base-url"},
		},
		{
			SchemaVersion: "orquesta_goal_result.v0",
			Status:        orquestagoal.GoalStatusBlockedV0,
			GoalRef:       "goal-ref-blocked-real-002",
			Summary:       "blocked con trabajo evidenciado",
			Checklist: orquestagoal.GoalWorkChecklistV0{
				CompletedRefs: []string{"criterion:parte-hecha"},
			},
		},
		{
			SchemaVersion: "orquesta_goal_result.v0",
			Status:        orquestagoal.GoalStatusCompleteV0,
			GoalRef:       "goal-ref-complete-001",
			Summary:       "started",
		},
	}
	for _, marked := range cases {
		receipt := orquestaruntimecodexgoal.CodexGoalObservationReceiptV0{}
		mergeCodexAppServerGoalResultV0(&receipt, marked, "evidence-ref-source-test")
		if receipt.IssueCode == codexAppServerGoalResultPlaceholderReasonCodeV0 {
			t.Fatalf("goal %s no debe clasificarse placeholder: %+v", marked.GoalRef, receipt)
		}
	}
}

func TestMergeCodexAppServerGoalResultV0ProyectaReasonCodeExplicitoV0(t *testing.T) {
	receipt := orquestaruntimecodexgoal.CodexGoalObservationReceiptV0{}
	mergeCodexAppServerGoalResultV0(&receipt, codexAppServerGoalResultMarkerV0{
		SchemaVersion: "orquesta_goal_result.v0",
		Status:        orquestagoal.GoalStatusBlockedV0,
		GoalRef:       "goal-ref-reason-code-001",
		ReasonCode:    codexAppServerGoalResultCheckpointReasonCodeV0,
		Summary:       "started",
		Checklist: orquestagoal.GoalWorkChecklistV0{
			MissingRefs: []string{"implementation"},
		},
	}, "evidence-ref-source-test")
	if receipt.IssueCode != codexAppServerGoalResultCheckpointReasonCodeV0 {
		t.Fatalf("issue_code=%q", receipt.IssueCode)
	}
}

func stackStringsContainForPlaceholderTestV0(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
