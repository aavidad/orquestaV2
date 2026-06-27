package orquestaappdirectorservice

import (
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func NewAppDirectorGoalFirstRunMarkerV0(
	marker AppDirectorGoalFirstRunMarkerV0,
) (AppDirectorGoalFirstRunMarkerV0, error) {
	marker.SchemaVersion = AppDirectorGoalFirstRunMarkerSchemaV0
	marker.RunRef = strings.TrimSpace(marker.RunRef)
	marker.GoalRef = strings.TrimSpace(marker.GoalRef)
	marker.ExternalGoalRef = strings.TrimSpace(marker.ExternalGoalRef)
	marker.DirectorKind = strings.TrimSpace(marker.DirectorKind)
	if marker.DirectorKind == "" {
		marker.DirectorKind = orquestagoal.GoalDirectorKindCodexGoalV0
	}
	marker.Status = strings.TrimSpace(marker.Status)
	marker.EvidenceRefs = compactStartAppDirectorStringsV0(marker.EvidenceRefs)
	if marker.RunRef == "" {
		return AppDirectorGoalFirstRunMarkerV0{}, AppDirectorServiceIssueV0{Field: "goal_first_run_marker.run_ref"}
	}
	if marker.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 &&
		marker.DirectorKind != orquestagoal.GoalDirectorKindRuntimeGoalV0 {
		return AppDirectorGoalFirstRunMarkerV0{}, AppDirectorServiceIssueV0{Field: "goal_first_run_marker.director_kind"}
	}
	return marker, nil
}

func appDirectorGoalFirstRunMarkerFromLaunchV0(
	runRef string,
	receipt orquestagoal.GoalLaunchReceiptV0,
	evidenceRefs []string,
) AppDirectorGoalFirstRunMarkerV0 {
	return AppDirectorGoalFirstRunMarkerV0{
		SchemaVersion:   AppDirectorGoalFirstRunMarkerSchemaV0,
		RunRef:          strings.TrimSpace(runRef),
		GoalRef:         strings.TrimSpace(receipt.GoalRef),
		ExternalGoalRef: strings.TrimSpace(receipt.ExternalGoalRef),
		DirectorKind:    orquestagoal.GoalDirectorKindCodexGoalV0,
		Status:          strings.TrimSpace(receipt.Status),
		EvidenceRefs: compactStartAppDirectorStringsV0(append(
			[]string{"evidence-ref-app-director-goal-first-run-marker-v0"},
			evidenceRefs...,
		)),
	}
}
