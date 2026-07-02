package orquestaserver

import (
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestCopyGoalWorkResultForServerStateV0ConservaContratoDeArtefactosParciales(t *testing.T) {
	source := orquestagoal.GoalWorkResultV0{
		Status:        orquestagoal.GoalStatusBlockedV0,
		GoalRef:       "goal-ref-server-copy-001",
		ArtifactRefs:  []string{"artifact-ref-copy-001"},
		ArtifactPaths: []string{"trabajo/tema_001.md"},
		MaterializedArtifacts: []orquestagoal.GoalMaterializedArtifactV0{{
			ArtifactRef:  "artifact-ref-copy-001",
			Path:         "trabajo/tema_001.md",
			ArtifactType: "markdown",
			Status:       orquestagoal.GoalMaterializedArtifactStatusPartialV0,
			EvidenceRefs: []string{"evidence-ref-artifact-copy-001"},
			Issues:       []orquestagoal.GoalWorkIssueV0{{Code: "qa_failed_public_text", Field: "public_text"}},
		}},
		Checklist: orquestagoal.GoalWorkChecklistV0{
			ExpectedRefs:  []string{"check-ref-publicable"},
			CompletedRefs: []string{"check-ref-fuentes"},
			MissingRefs:   []string{"check-ref-publicable"},
			EvidenceRefs:  []string{"evidence-ref-checklist-copy-001"},
		},
		DomainReceiptRefs: []string{"domain-receipt-ref-copy-001"},
		ReworkPlanRefs:    []string{"rework-plan-ref-copy-001"},
		EvidenceRefs:      []string{"evidence-ref-goal-copy-001"},
		Issues:            []orquestagoal.GoalWorkIssueV0{{Code: "goal_blocked", Field: "status"}},
	}

	copied := copyGoalWorkResultForServerStateV0(source)
	source.MaterializedArtifacts[0].EvidenceRefs[0] = "mutated"
	source.MaterializedArtifacts[0].Issues[0].Code = "mutated"
	source.Checklist.MissingRefs[0] = "mutated"
	source.ReworkPlanRefs[0] = "mutated"

	if len(copied.ArtifactPaths) != 1 ||
		copied.ArtifactPaths[0] != "trabajo/tema_001.md" ||
		len(copied.MaterializedArtifacts) != 1 ||
		copied.MaterializedArtifacts[0].Status != orquestagoal.GoalMaterializedArtifactStatusPartialV0 ||
		copied.MaterializedArtifacts[0].EvidenceRefs[0] != "evidence-ref-artifact-copy-001" ||
		copied.MaterializedArtifacts[0].Issues[0].Code != "qa_failed_public_text" ||
		copied.Checklist.MissingRefs[0] != "check-ref-publicable" ||
		copied.ReworkPlanRefs[0] != "rework-plan-ref-copy-001" {
		t.Fatalf("copied=%+v", copied)
	}
}
