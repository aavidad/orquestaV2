package orquestaapprunner

import (
	orquestaappplanner "orquesta/modulos/orquesta-app-planner"
)

func PrepareAppOrchestrationV0(
	request PrepareAppOrchestrationRequestV0,
) (AppOrchestrationPreparedV0, error) {
	request = normalizePrepareAppOrchestrationRequestV0(request)
	if err := validatePrepareAppOrchestrationRequestV0(request); err != nil {
		return AppOrchestrationPreparedV0{}, err
	}
	plan, err := orquestaappplanner.BuildGoAPIWebMicrotaskPlanFromAppSpecV0(
		request.RunRef,
		request.AppSpec,
	)
	if err != nil {
		return AppOrchestrationPreparedV0{}, err
	}
	run, err := buildInitialAppRunV0(request, plan)
	if err != nil {
		return AppOrchestrationPreparedV0{}, err
	}
	progress, err := orquestaappplanner.EvaluateAppPlanProgressV0(plan, run.Deliveries)
	if err != nil {
		return AppOrchestrationPreparedV0{}, err
	}
	provider := orquestaappplanner.AppPlanCandidateProviderV0{
		Plan:        plan,
		RequestedBy: request.RequestedBy,
	}
	return AppOrchestrationPreparedV0{
		SchemaVersion:     AppOrchestrationPreparedSchemaVersionV0,
		Run:               run,
		Plan:              plan,
		InitialProgress:   progress,
		CandidateProvider: provider,
		RoutePolicy:       AppRunnerPreviewRoutePolicyV0(AppRunnerLegacyEntrypointPrepareV0),
		EvidenceRefs: compactAppRunnerRefsV0([]string{
			"evidence-ref-app-runner-prepared-v0",
			"evidence-ref-" + safeAppRunnerRefPartV0(request.RunRef),
			AppRunnerPreviewCompatibilityEvidenceV0,
		}),
	}, nil
}
