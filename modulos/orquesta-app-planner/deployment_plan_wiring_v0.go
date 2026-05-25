package orquestaappplanner

import (
	"strings"

	orquestadeploy "orquesta/modulos/orquesta-deploy"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

type AppDeploymentPlanDryRunV0 struct {
	Plan    orquestadeploy.DeploymentPlanV0              `json:"plan"`
	Receipt orquestadeploy.DeploymentPlanDryRunReceiptV0 `json:"receipt"`
}

func PrepareDeploymentPlanDryRunFromAppSpecV0(
	runRef string,
	spec orquestafactory.AppSpecV0,
	port orquestadeploy.DeploymentPlanDryRunPortV0,
) (AppDeploymentPlanDryRunV0, error) {
	if err := validateFactoryAppSpecForPlanV0(spec); err != nil {
		return AppDeploymentPlanDryRunV0{}, err
	}
	if !appPlanDeployNeedsPlanV0(spec.Deploy.Target) {
		return AppDeploymentPlanDryRunV0{}, AppPlannerIssueV0{Field: "app_spec.deploy.target"}
	}
	plan, err := orquestadeploy.BuildDeploymentPlanForCompositionDryRunV0(
		orquestadeploy.DeploymentPlanCompositionRequestV0{
			RequestID:      firstAppPlanValueV0(runRef, spec.RequestID),
			AppSpecID:      spec.SpecID,
			AppSpecVersion: spec.SchemaVersion,
			DeployTarget:   spec.Deploy.Target,
			Restrictions:   spec.Deploy.Restrictions,
			CorrelationRef: "run-ref-" + strings.TrimSpace(runRef),
			CompositionRef: "orquesta-app-planner",
			RequestedBy:    "orquesta-app-planner",
		},
	)
	if err != nil {
		return AppDeploymentPlanDryRunV0{}, err
	}
	receipt, err := orquestadeploy.PrepareDeploymentPlanDryRunThroughPortV0(plan, port)
	if err != nil {
		return AppDeploymentPlanDryRunV0{}, err
	}
	return AppDeploymentPlanDryRunV0{Plan: plan, Receipt: receipt}, nil
}

func deploymentPlanEvidenceRefV0(request AppPlanRequestV0) string {
	target := firstAppPlanValueV0(request.DeployTarget, "deploy")
	return "evidence-ref-" + request.AppRef + "-deployment-plan-" + target + "-dry-run"
}
