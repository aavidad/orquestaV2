package orquestamcp

import "testing"

func TestEnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0QAFailedPublicTextPideRework(t *testing.T) {
	result := EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0(
		MCPObserveAppDirectorGoalToolResultV0{
			GoalRef:    "goal-ref-observe-qa-failed-public-text-001",
			GoalStatus: "blocked",
		},
		MCPDirectorGoalMaterializedRefsV0{
			ArtifactRefs: []string{"artifact-ref-materialized-qa-failed-public-text-001"},
			EvidenceRefs: []string{"evidence-ref-goal-materialized-qa-failed-public-text-001"},
			IssueCodes:   []string{MCPGoalFirstQAFailedPublicTextV0},
		},
	)

	if result.RecommendedAction != MCPGoalFirstReworkPublicTextActionV0 ||
		len(result.ClosureIssues) != 1 ||
		result.ClosureIssues[0].Code != MCPGoalFirstQAFailedPublicTextV0 ||
		result.ClosureIssues[0].Field != "goal_first.qa_public_text" ||
		!containsStringMCPV0(result.ArtifactRefs, "artifact-ref-materialized-qa-failed-public-text-001") {
		t.Fatalf("result=%+v", result)
	}
}
