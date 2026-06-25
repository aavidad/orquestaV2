package orquestaautoprogramming

import "strings"

const (
	AutoprogrammingGoalMigrationLegacyCompatibleV0          = "legacy_loop_compatible"
	AutoprogrammingGoalMigrationGoalReadyV0                 = "goal_ready"
	AutoprogrammingGoalMigrationBlockedByGoalCapabilityV0   = "blocked_by_goal_capability"
	AutoprogrammingGoalMigrationCoveredByGoalFirstV0        = "covered_by_goal_first"
	AutoprogrammingGoalMigrationLegacyLoopRequiredV0        = "legacy_loop_required"
	AutoprogrammingGoalMigrationActionKeepLegacyLoopV0      = "keep_legacy_loop"
	AutoprogrammingGoalMigrationActionLaunchGoalV0          = "launch_goal_when_composition_ready"
	AutoprogrammingGoalMigrationActionWaitCapabilitiesV0    = "wait_for_goal_capabilities"
	AutoprogrammingGoalMigrationActionDoNotScheduleLegacyV0 = "do_not_schedule_legacy_loop"
)

type AutoprogrammingGoalMigrationClassificationV0 struct {
	Status                string   `json:"status"`
	RecommendedAction     string   `json:"recommended_action"`
	GoalFirstCandidate    bool     `json:"goal_first_candidate,omitempty"`
	LegacyLoopRequired    bool     `json:"legacy_loop_required,omitempty"`
	CoveredByGoalFirst    bool     `json:"covered_by_goal_first,omitempty"`
	TaskRefs              []string `json:"task_refs,omitempty"`
	MissingCapabilityRefs []string `json:"missing_capability_refs,omitempty"`
	EvidenceRefs          []string `json:"evidence_refs,omitempty"`
}

func ClassifyAutoprogrammingGoalMigrationV0(
	request AutoprogrammingRequestV0,
) AutoprogrammingGoalMigrationClassificationV0 {
	markers := collectAutoprogrammingGoalMigrationMarkersV0(request.Tasks)
	taskRefs := compactStringsV0(markers.taskRefs)
	evidenceRefs := compactStringsV0(markers.evidenceRefs)

	if markers.legacyRequired {
		return AutoprogrammingGoalMigrationClassificationV0{
			Status:             AutoprogrammingGoalMigrationLegacyLoopRequiredV0,
			RecommendedAction:  AutoprogrammingGoalMigrationActionKeepLegacyLoopV0,
			LegacyLoopRequired: true,
			TaskRefs:           taskRefs,
			EvidenceRefs:       evidenceRefs,
		}
	}
	if markers.covered {
		return AutoprogrammingGoalMigrationClassificationV0{
			Status:             AutoprogrammingGoalMigrationCoveredByGoalFirstV0,
			RecommendedAction:  AutoprogrammingGoalMigrationActionDoNotScheduleLegacyV0,
			GoalFirstCandidate: markers.goalFirst,
			CoveredByGoalFirst: true,
			TaskRefs:           taskRefs,
			EvidenceRefs:       evidenceRefs,
		}
	}
	if markers.goalFirst {
		missing := autoprogrammingGoalMigrationMissingCapabilitiesV0(markers.capabilities)
		if len(missing) > 0 {
			return AutoprogrammingGoalMigrationClassificationV0{
				Status:                AutoprogrammingGoalMigrationBlockedByGoalCapabilityV0,
				RecommendedAction:     AutoprogrammingGoalMigrationActionWaitCapabilitiesV0,
				GoalFirstCandidate:    true,
				TaskRefs:              taskRefs,
				MissingCapabilityRefs: missing,
				EvidenceRefs:          evidenceRefs,
			}
		}
		return AutoprogrammingGoalMigrationClassificationV0{
			Status:             AutoprogrammingGoalMigrationGoalReadyV0,
			RecommendedAction:  AutoprogrammingGoalMigrationActionLaunchGoalV0,
			GoalFirstCandidate: true,
			TaskRefs:           taskRefs,
			EvidenceRefs:       evidenceRefs,
		}
	}
	return AutoprogrammingGoalMigrationClassificationV0{
		Status:            AutoprogrammingGoalMigrationLegacyCompatibleV0,
		RecommendedAction: AutoprogrammingGoalMigrationActionKeepLegacyLoopV0,
	}
}

type autoprogrammingGoalMigrationMarkersV0 struct {
	goalFirst      bool
	covered        bool
	legacyRequired bool
	capabilities   map[string]struct{}
	taskRefs       []string
	evidenceRefs   []string
}

func collectAutoprogrammingGoalMigrationMarkersV0(
	tasks []AutoprogrammingTaskGroupCandidateV0,
) autoprogrammingGoalMigrationMarkersV0 {
	markers := autoprogrammingGoalMigrationMarkersV0{
		capabilities: map[string]struct{}{},
	}
	for _, task := range tasks {
		taskRef := strings.TrimSpace(task.TaskRef)
		for _, ref := range task.ContextRefs {
			marker := strings.TrimSpace(ref)
			switch marker {
			case "goal_migration:goal-first":
				markers.goalFirst = true
				markers.taskRefs = append(markers.taskRefs, taskRef)
			case "goal_migration:covered":
				markers.covered = true
				markers.taskRefs = append(markers.taskRefs, taskRef)
			case "goal_migration:legacy-required":
				markers.legacyRequired = true
				markers.taskRefs = append(markers.taskRefs, taskRef)
			case "goal_capability:starter", "goal_capability:observer", "goal_capability:closure-validator":
				markers.capabilities[marker] = struct{}{}
				markers.evidenceRefs = append(markers.evidenceRefs, marker)
			}
		}
	}
	return markers
}

func autoprogrammingGoalMigrationMissingCapabilitiesV0(
	capabilities map[string]struct{},
) []string {
	required := []string{
		"goal_capability:starter",
		"goal_capability:observer",
		"goal_capability:closure-validator",
	}
	var missing []string
	for _, ref := range required {
		if _, ok := capabilities[ref]; !ok {
			missing = append(missing, ref)
		}
	}
	return missing
}
