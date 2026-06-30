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
	if marker.Spec != nil {
		normalized := orquestagoal.NormalizeGoalWorkSpecV0(*marker.Spec)
		marker.Spec = &normalized
	}
	if marker.LaunchReceipt != nil {
		normalized := orquestagoal.NormalizeGoalLaunchReceiptV0(*marker.LaunchReceipt)
		marker.LaunchReceipt = &normalized
	}
	marker.EvidenceRefs = compactStartAppDirectorStringsV0(marker.EvidenceRefs)
	if marker.RunRef == "" {
		return AppDirectorGoalFirstRunMarkerV0{}, AppDirectorServiceIssueV0{Field: "goal_first_run_marker.run_ref"}
	}
	if marker.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 &&
		marker.DirectorKind != orquestagoal.GoalDirectorKindRuntimeGoalV0 {
		return AppDirectorGoalFirstRunMarkerV0{}, AppDirectorServiceIssueV0{Field: "goal_first_run_marker.director_kind"}
	}
	normalized, err := orquestagoal.NewGoalWorkRunMarkerV0(marker)
	if err != nil {
		return AppDirectorGoalFirstRunMarkerV0{}, err
	}
	return normalized, nil
}

func appDirectorGoalFirstRunMarkerFromLaunchV0(
	runRef string,
	spec orquestagoal.GoalWorkSpecV0,
	receipt orquestagoal.GoalLaunchReceiptV0,
	evidenceRefs []string,
) AppDirectorGoalFirstRunMarkerV0 {
	spec = orquestagoal.NormalizeGoalWorkSpecV0(spec)
	receipt = orquestagoal.NormalizeGoalLaunchReceiptV0(receipt)
	return AppDirectorGoalFirstRunMarkerV0{
		SchemaVersion:   AppDirectorGoalFirstRunMarkerSchemaV0,
		RunRef:          strings.TrimSpace(runRef),
		GoalRef:         strings.TrimSpace(receipt.GoalRef),
		ExternalGoalRef: strings.TrimSpace(receipt.ExternalGoalRef),
		DirectorKind:    orquestagoal.GoalDirectorKindCodexGoalV0,
		Status:          strings.TrimSpace(receipt.Status),
		Spec:            &spec,
		LaunchReceipt:   &receipt,
		EvidenceRefs: compactStartAppDirectorStringsV0(append(
			[]string{"evidence-ref-app-director-goal-first-run-marker-v0"},
			evidenceRefs...,
		)),
	}
}
