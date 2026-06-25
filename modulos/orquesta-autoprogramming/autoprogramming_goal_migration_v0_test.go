package orquestaautoprogramming

import (
	"reflect"
	"testing"
)

func TestClassifyAutoprogrammingGoalMigrationV0DefaultKeepsLegacyLoop(t *testing.T) {
	classification := ClassifyAutoprogrammingGoalMigrationV0(validAutoprogrammingRequestV0(nil))

	if classification.Status != AutoprogrammingGoalMigrationLegacyCompatibleV0 ||
		classification.RecommendedAction != AutoprogrammingGoalMigrationActionKeepLegacyLoopV0 ||
		classification.GoalFirstCandidate ||
		classification.LegacyLoopRequired ||
		classification.CoveredByGoalFirst {
		t.Fatalf("classification inesperada: %+v", classification)
	}
}

func TestClassifyAutoprogrammingGoalMigrationV0BloqueaGoalSinCapacidades(t *testing.T) {
	classification := ClassifyAutoprogrammingGoalMigrationV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef:     "task-ref-goal-first-001",
			Area:        "Autoprogramming",
			ContextRefs: []string{"goal_migration:goal-first", "goal_capability:starter"},
		}}
	}))

	if classification.Status != AutoprogrammingGoalMigrationBlockedByGoalCapabilityV0 ||
		classification.RecommendedAction != AutoprogrammingGoalMigrationActionWaitCapabilitiesV0 ||
		!classification.GoalFirstCandidate {
		t.Fatalf("classification inesperada: %+v", classification)
	}
	if !reflect.DeepEqual(classification.TaskRefs, []string{"task-ref-goal-first-001"}) {
		t.Fatalf("task_refs=%v", classification.TaskRefs)
	}
	wantMissing := []string{"goal_capability:observer", "goal_capability:closure-validator"}
	if !reflect.DeepEqual(classification.MissingCapabilityRefs, wantMissing) {
		t.Fatalf("missing=%v want=%v", classification.MissingCapabilityRefs, wantMissing)
	}
}

func TestClassifyAutoprogrammingGoalMigrationV0MarcaGoalListo(t *testing.T) {
	classification := ClassifyAutoprogrammingGoalMigrationV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef: "task-ref-goal-ready-001",
			Area:    "Autoprogramming",
			ContextRefs: []string{
				"goal_migration:goal-first",
				"goal_capability:starter",
				"goal_capability:observer",
				"goal_capability:closure-validator",
			},
		}}
	}))

	if classification.Status != AutoprogrammingGoalMigrationGoalReadyV0 ||
		classification.RecommendedAction != AutoprogrammingGoalMigrationActionLaunchGoalV0 ||
		!classification.GoalFirstCandidate ||
		len(classification.MissingCapabilityRefs) != 0 {
		t.Fatalf("classification inesperada: %+v", classification)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0ConservaClasificacionGoalMigration(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef:     "task-ref-covered-by-goal-001",
			Area:        "Autoprogramming",
			ContextRefs: []string{"goal_migration:covered"},
		}}
		request.WriteSet = []string{"modulos/orquesta-autoprogramming/autoprogramming_goal_migration_v0.go"}
	}))

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	classification := result.Work.GoalMigration
	if classification.Status != AutoprogrammingGoalMigrationCoveredByGoalFirstV0 ||
		classification.RecommendedAction != AutoprogrammingGoalMigrationActionDoNotScheduleLegacyV0 ||
		!classification.CoveredByGoalFirst {
		t.Fatalf("classification inesperada: %+v", classification)
	}
	if len(result.Work.Tasks) != 1 {
		t.Fatalf("tasks=%d", len(result.Work.Tasks))
	}
}

func TestClassifyAutoprogrammingGoalMigrationV0CoveredNoRelanzaGoal(t *testing.T) {
	classification := ClassifyAutoprogrammingGoalMigrationV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef: "task-ref-covered-goal-first-001",
			Area:    "Autoprogramming",
			ContextRefs: []string{
				"goal_migration:covered",
				"goal_migration:goal-first",
				"goal_capability:starter",
				"goal_capability:observer",
				"goal_capability:closure-validator",
			},
		}}
	}))

	if classification.Status != AutoprogrammingGoalMigrationCoveredByGoalFirstV0 ||
		classification.RecommendedAction != AutoprogrammingGoalMigrationActionDoNotScheduleLegacyV0 ||
		!classification.CoveredByGoalFirst ||
		!classification.GoalFirstCandidate {
		t.Fatalf("classification inesperada: %+v", classification)
	}
	if len(classification.MissingCapabilityRefs) != 0 {
		t.Fatalf("missing=%v", classification.MissingCapabilityRefs)
	}
}

func TestClassifyAutoprogrammingGoalMigrationV0LegacyRequiredGana(t *testing.T) {
	classification := ClassifyAutoprogrammingGoalMigrationV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef: "task-ref-legacy-required-001",
			Area:    "Autoprogramming",
			ContextRefs: []string{
				"goal_migration:goal-first",
				"goal_migration:legacy-required",
				"goal_capability:starter",
				"goal_capability:observer",
				"goal_capability:closure-validator",
			},
		}}
	}))

	if classification.Status != AutoprogrammingGoalMigrationLegacyLoopRequiredV0 ||
		classification.RecommendedAction != AutoprogrammingGoalMigrationActionKeepLegacyLoopV0 ||
		!classification.LegacyLoopRequired {
		t.Fatalf("classification inesperada: %+v", classification)
	}
}
