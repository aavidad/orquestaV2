package orquestadeploy

import "strings"

const (
	DeploymentPlanDryRunReceiptSchemaV0 = "DeploymentPlanDryRunReceiptV0"
	DeploymentPlanDryRunModeV0          = "dry_run"
	DeploymentPlanDryRunStatusReadyV0   = "prepared"
)

type DeploymentPlanCompositionRequestV0 struct {
	RequestID       string   `json:"request_id"`
	AppSpecID       string   `json:"app_spec_id"`
	AppSpecVersion  string   `json:"app_spec_version"`
	DeployTarget    string   `json:"deploy_target"`
	Restrictions    []string `json:"restrictions,omitempty"`
	CorrelationRef  string   `json:"correlation_ref,omitempty"`
	CompositionRef  string   `json:"composition_ref,omitempty"`
	RequestedBy     string   `json:"requested_by,omitempty"`
	TargetResultRef string   `json:"target_result_ref,omitempty"`
}

type DeploymentPlanDryRunReceiptV0 struct {
	SchemaVersion  string   `json:"schema_version"`
	PlanRef        string   `json:"plan_ref"`
	EvidenceRef    string   `json:"evidence_ref"`
	ResultRef      string   `json:"result_ref"`
	DeployTarget   string   `json:"deploy_target"`
	TargetResolved string   `json:"target_resolved"`
	Mode           string   `json:"mode"`
	Status         string   `json:"status"`
	AdapterRef     string   `json:"adapter_ref"`
	EvidenceRefs   []string `json:"evidence_refs"`
}

type DeploymentPlanDryRunPortV0 interface {
	PrepareDeploymentPlanDryRunV0(plan DeploymentPlanV0) (DeploymentPlanDryRunReceiptV0, error)
}

type DefaultDeploymentPlanDryRunPortV0 struct{}

func BuildDeploymentPlanForCompositionDryRunV0(request DeploymentPlanCompositionRequestV0) (DeploymentPlanV0, error) {
	target := firstDeploymentValueV0(request.DeployTarget, "local")
	plan := DeploymentPlanV0{
		SchemaVersion:        DeploymentPlanSchemaV0,
		RequestID:            strings.TrimSpace(request.RequestID),
		AppSpecID:            strings.TrimSpace(request.AppSpecID),
		AppSpecVersion:       firstDeploymentValueV0(request.AppSpecVersion, AppSpecSchemaV0),
		DeployTarget:         target,
		SistemasOperativos:   []string{"linux", "darwin", "windows"},
		AceptacionContenedor: target == "contenedor",
		Restricciones:        deploymentPlanRestrictionsV0(target, request.Restrictions),
		PlanID:               deploymentPlanRefV0(request.AppSpecID, target),
		TargetResuelto:       target,
		MatrizOS:             deploymentPlanOSMatrixV0(target),
		ValidacionEntorno:    deploymentPlanChecksV0("validacion_entorno"),
		ArtefactosPrevistos:  deploymentPlanArtifactsV0(target),
		Healthcheck:          deploymentPlanChecksV0("healthcheck"),
		Rollback:             deploymentPlanRollbackV0(target),
		AccionesPrevistas:    deploymentPlanActionsV0(target),
		Advertencias: []string{
			"dry-run: plan declarativo sin ejecutar Docker, Kubernetes, cloud, filesystem productivo ni secretos.",
		},
	}
	if err := ValidateDeploymentPlanV0(plan); err != nil {
		return DeploymentPlanV0{}, err
	}
	return plan, nil
}

func PrepareDeploymentPlanDryRunThroughPortV0(
	plan DeploymentPlanV0,
	port DeploymentPlanDryRunPortV0,
) (DeploymentPlanDryRunReceiptV0, error) {
	if port == nil {
		port = DefaultDeploymentPlanDryRunPortV0{}
	}
	return port.PrepareDeploymentPlanDryRunV0(plan)
}

func (DefaultDeploymentPlanDryRunPortV0) PrepareDeploymentPlanDryRunV0(
	plan DeploymentPlanV0,
) (DeploymentPlanDryRunReceiptV0, error) {
	if err := prepareDeploymentTargetDryRunV0(plan); err != nil {
		return DeploymentPlanDryRunReceiptV0{}, err
	}
	target := strings.TrimSpace(plan.TargetResuelto)
	return DeploymentPlanDryRunReceiptV0{
		SchemaVersion:  DeploymentPlanDryRunReceiptSchemaV0,
		PlanRef:        "deployment-plan-ref-" + strings.TrimSpace(plan.PlanID),
		EvidenceRef:    "deployment-dry-run-evidence-ref-" + strings.TrimSpace(plan.PlanID),
		ResultRef:      "deployment-dry-run-result-ref-" + strings.TrimSpace(plan.PlanID),
		DeployTarget:   strings.TrimSpace(plan.DeployTarget),
		TargetResolved: target,
		Mode:           DeploymentPlanDryRunModeV0,
		Status:         DeploymentPlanDryRunStatusReadyV0,
		AdapterRef:     "orquesta-deploy-" + target + "-dry-run",
		EvidenceRefs: []string{
			"deployment-plan-validated",
			"deployment-target-dry-run-prepared",
			"deployment-no-external-effects",
		},
	}, nil
}

func prepareDeploymentTargetDryRunV0(plan DeploymentPlanV0) error {
	switch strings.TrimSpace(plan.TargetResuelto) {
	case "local":
		_, err := PrepareLocalDeployDryRunV0(plan)
		return err
	case "contenedor":
		_, err := PrepareContainerDeployDryRunV0(plan)
		return err
	case "kubernetes":
		_, err := PrepareKubernetesDeployDryRunV0(plan)
		return err
	case "paas":
		_, err := PreparePaaSDeployDryRunV0(plan)
		return err
	case "desktop":
		_, err := PrepareDesktopDeployDryRunV0(plan)
		return err
	case "mobile_store":
		_, err := PrepareMobileStoreDeployDryRunV0(plan)
		return err
	default:
		return ValidateDeploymentPlanV0(plan)
	}
}
