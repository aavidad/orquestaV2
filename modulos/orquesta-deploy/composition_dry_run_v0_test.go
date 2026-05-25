package orquestadeploy

import "testing"

func TestBuildDeploymentPlanForCompositionDryRunV0PreparaPuertoDryRun(t *testing.T) {
	plan, err := BuildDeploymentPlanForCompositionDryRunV0(DeploymentPlanCompositionRequestV0{
		RequestID:      "request-ref-deploy-001",
		AppSpecID:      "app-spec-ref-agenda",
		AppSpecVersion: AppSpecSchemaV0,
		DeployTarget:   "kubernetes",
		Restrictions:   []string{"sin secretos"},
	})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	receipt, err := PrepareDeploymentPlanDryRunThroughPortV0(plan, nil)
	if err != nil {
		t.Fatalf("prepare dry-run: %v", err)
	}
	if receipt.SchemaVersion != DeploymentPlanDryRunReceiptSchemaV0 ||
		receipt.Mode != DeploymentPlanDryRunModeV0 ||
		receipt.Status != DeploymentPlanDryRunStatusReadyV0 {
		t.Fatalf("receipt inesperado: %+v", receipt)
	}
	if receipt.DeployTarget != "kubernetes" ||
		receipt.PlanRef == "" ||
		len(receipt.EvidenceRefs) == 0 {
		t.Fatalf("refs/evidencia incompletas: %+v", receipt)
	}
}

func TestBuildDeploymentPlanForCompositionDryRunV0CubreTargetPaaSConProveedorOpaco(t *testing.T) {
	plan, err := BuildDeploymentPlanForCompositionDryRunV0(DeploymentPlanCompositionRequestV0{
		RequestID:      "request-ref-deploy-paas-001",
		AppSpecID:      "app-spec-ref-paas",
		DeployTarget:   "paas",
		Restrictions:   []string{"region se decide por composicion"},
		RequestedBy:    "orquesta-app-planner",
		CompositionRef: "composition-ref-test",
	})
	if err != nil {
		t.Fatalf("build paas plan: %v", err)
	}

	if _, err := PrepareDeploymentPlanDryRunThroughPortV0(plan, DefaultDeploymentPlanDryRunPortV0{}); err != nil {
		t.Fatalf("paas dry-run: %v", err)
	}
	if !paasHasProviderRestrictionV0(plan.Restricciones) {
		t.Fatalf("paas debe conservar proveedor opaco pendiente: %+v", plan.Restricciones)
	}
}
