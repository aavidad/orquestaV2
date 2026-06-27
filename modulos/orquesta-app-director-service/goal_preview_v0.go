package orquestaappdirectorservice

import (
	"context"
	"strings"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

func BuildStartAppDirectorGoalWorkPreviewV0(
	_ context.Context,
	request StartAppDirectorRequestV0,
) (StartAppDirectorGoalPreviewV0, error) {
	request = normalizeStartAppDirectorRequestV0(request)
	request = startAppDirectorRequestWithOperationalPlanRunRefV0(request)
	if err := validateStartAppDirectorRequestV0(request); err != nil {
		return StartAppDirectorGoalPreviewV0{}, err
	}
	now, err := startAppDirectorNowV0(request.OccurredAt)
	if err != nil {
		return StartAppDirectorGoalPreviewV0{}, err
	}
	if request.OccurredAt == "" {
		request.OccurredAt = now.Format("2006-01-02T15:04:05Z07:00")
	}
	request.DirectorExecutionMode = AppDirectorExecutionModeGoalFirstV0
	spec, issues := orquestafactory.SolicitarNuevaAppV0(request.AppSpecRequest, now)
	if len(issues) > 0 {
		return invalidStartAppDirectorGoalPreviewV0(request, issues), nil
	}
	prepared, err := orquestaappdirectorintake.PrepareAppDirectorIntakeV0(
		prepareDirectorIntakeRequestV0(request, spec),
	)
	if err != nil {
		return StartAppDirectorGoalPreviewV0{}, err
	}
	goalSpec := buildStartAppDirectorGoalWorkSpecV0(request, spec, prepared)
	goalIssues := orquestagoal.ValidateGoalWorkSpecV0(goalSpec)
	status := StartAppDirectorStatusPreviewReadyV0
	if len(goalIssues) > 0 {
		status = StartAppDirectorStatusInvalidV0
	}
	return StartAppDirectorGoalPreviewV0{
		SchemaVersion:         StartAppDirectorPreviewSchemaV0,
		Status:                status,
		DirectorExecutionMode: AppDirectorExecutionModeGoalFirstV0,
		CorrelationID:         request.CorrelationID,
		AppSpec:               spec,
		Run:                   prepared.Run,
		GoalSpec:              goalSpec,
		WriteSet:              appDirectorGoalPreviewWriteSetV0(goalSpec),
		RequiredTests:         appDirectorGoalPreviewRequiredTestsV0(goalSpec),
		Estimate:              appDirectorGoalPreviewEstimateV0(goalSpec),
		GoalSpecIssues:        append([]orquestagoal.GoalWorkIssueV0(nil), goalIssues...),
		EvidenceRefs: compactStartAppDirectorStringsV0(append(
			[]string{"evidence-ref-app-director-goal-preview-v0"},
			prepared.EvidenceRefs...,
		)),
	}, nil
}

func invalidStartAppDirectorGoalPreviewV0(
	request StartAppDirectorRequestV0,
	issues []orquestafactory.ValidationIssue,
) StartAppDirectorGoalPreviewV0 {
	return StartAppDirectorGoalPreviewV0{
		SchemaVersion:         StartAppDirectorPreviewSchemaV0,
		Status:                StartAppDirectorStatusInvalidV0,
		DirectorExecutionMode: AppDirectorExecutionModeGoalFirstV0,
		CorrelationID:         request.CorrelationID,
		ValidationIssues:      append([]orquestafactory.ValidationIssue(nil), issues...),
		EvidenceRefs:          []string{"evidence-ref-app-director-goal-preview-invalid-v0"},
	}
}

func appDirectorGoalPreviewWriteSetV0(spec orquestagoal.GoalWorkSpecV0) []string {
	out := make([]string, 0, len(spec.WriteSet))
	for _, scope := range spec.WriteSet {
		if path := strings.TrimSpace(scope.Path); path != "" {
			out = append(out, path)
		}
	}
	return compactStartAppDirectorStringsV0(out)
}

func appDirectorGoalPreviewRequiredTestsV0(spec orquestagoal.GoalWorkSpecV0) []string {
	out := make([]string, 0, len(spec.RequiredTests))
	for _, test := range spec.RequiredTests {
		value := strings.TrimSpace(test.Command)
		if value == "" {
			value = strings.TrimSpace(test.TestRef)
		}
		if value != "" {
			out = append(out, value)
		}
	}
	return compactStartAppDirectorStringsV0(out)
}

func appDirectorGoalPreviewEstimateV0(spec orquestagoal.GoalWorkSpecV0) AppDirectorGoalPreviewEstimateV0 {
	maxSubgoals := spec.Budget.MaxSubgoals
	if maxSubgoals <= 0 {
		maxSubgoals = 1
	}
	maxRuntimeSeconds := spec.Budget.MaxRuntimeSeconds
	if maxRuntimeSeconds <= 0 {
		maxRuntimeSeconds = 1800 + maxSubgoals*600
	}
	tokenBudget := spec.Budget.TokenBudget
	if tokenBudget <= 0 {
		tokenBudget = 12000 +
			len(spec.ContextRefs)*200 +
			len(spec.RuleRefs)*250 +
			len(spec.WriteSet)*1000 +
			len(spec.RequiredTests)*2000 +
			len(spec.ArtifactContracts)*800 +
			len(spec.AcceptanceCriteria)*450 +
			maxSubgoals*2500
	}
	return AppDirectorGoalPreviewEstimateV0{
		TokenBudget:       tokenBudget,
		MaxRuntimeSeconds: maxRuntimeSeconds,
		MaxSubgoals:       maxSubgoals,
		MaxReworkGoals:    spec.Budget.MaxReworkGoals,
		WriteSetItems:     len(spec.WriteSet),
		RequiredTests:     len(spec.RequiredTests),
		ArtifactContracts: len(spec.ArtifactContracts),
		CostTier:          appDirectorGoalPreviewCostTierV0(tokenBudget),
	}
}

func appDirectorGoalPreviewCostTierV0(tokenBudget int) string {
	switch {
	case tokenBudget <= 22000:
		return "low"
	case tokenBudget <= 52000:
		return "medium"
	default:
		return "high"
	}
}
