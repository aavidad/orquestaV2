package orquestaappplanner

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildGoAPIWebMicrotaskPlanV0ParaAppGrandeDividePorContratos(t *testing.T) {
	plan, err := BuildGoAPIWebMicrotaskPlanV0(AppPlanRequestV0{
		RunRef:  "run-ref-large-001",
		AppRef:  "erp",
		AppName: "ERP",
		API:     true,
		Web:     true,
		Scale:   AppPlanScaleLargeV0,
	})
	if err != nil {
		t.Fatalf("BuildGoAPIWebMicrotaskPlanV0 large: %v", err)
	}
	if len(plan.Units) != 11 {
		t.Fatalf("units=%d", len(plan.Units))
	}
	assertAppUnitForTestV0(t, plan, "architecture", []string{"ack-erp-bootstrap"}, []string{
		"docs/arquitectura.md",
		"docs/contratos.md",
		"docs/decisiones.md",
	})
	assertAppUnitForTestV0(t, plan, "persistence-port", []string{"ack-erp-architecture"}, []string{
		"internal/persistence",
		"internal/testadapters",
	})
	assertAppUnitForTestV0(t, plan, "api", []string{"ack-erp-domain", "ack-erp-persistence-port"}, []string{
		"internal/api",
		"cmd/server",
	})
	assertAppUnitForTestV0(t, plan, "integration", []string{
		"ack-erp-domain",
		"ack-erp-persistence-port",
		"ack-erp-api",
		"ack-erp-web",
		"ack-erp-i18n",
	}, []string{"internal/app", "internal/integration"})

	review := appUnitByKeyForTestV0(t, plan, "review")
	if review.Capacity != orquestacoreworkflow.OrchestrationCapacityXHighV0 {
		t.Fatalf("review capacity=%s", review.Capacity)
	}
	assertLargePlanHasNoDBProviderWriteSetV0(t, plan)
}

func TestAppPlanRequestFromAppSpecV0EscalaConPersistenciaOCalidadAlta(t *testing.T) {
	spec := validFactoryAppSpecForPlannerTestV0(t)
	spec.Data.PersistenceRequired = true

	request, err := AppPlanRequestFromAppSpecV0("run-ref-large-factory-001", spec)
	if err != nil {
		t.Fatalf("AppPlanRequestFromAppSpecV0: %v", err)
	}
	if request.Scale != AppPlanScaleLargeV0 {
		t.Fatalf("scale=%s", request.Scale)
	}
}

func assertLargePlanHasNoDBProviderWriteSetV0(t *testing.T, plan AppMicrotaskPlanV0) {
	t.Helper()
	for _, unit := range plan.Units {
		for _, path := range unit.WriteSet {
			if path == "sqlite" || path == "postgres" || path == "mysql" {
				t.Fatalf("write-set provider concreto en %s: %s", unit.TaskRef, path)
			}
		}
	}
}
