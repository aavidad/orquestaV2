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
	assertAppUnitForTestV0(t, plan, "architecture", orquestacoreworkflow.OrchestrationPhaseProgramacionV0, []string{"ack-erp-bootstrap"}, []string{
		"docs/arquitectura.md",
		"docs/contratos.md",
		"docs/decisiones.md",
	})
	assertAppUnitForTestV0(t, plan, "persistence-port", orquestacoreworkflow.OrchestrationPhaseProgramacionV0, []string{"ack-erp-architecture"}, []string{
		"internal/adapters/persistence",
		"internal/testadapters",
	})
	assertAppUnitForTestV0(t, plan, "api", orquestacoreworkflow.OrchestrationPhaseProgramacionV0, []string{"ack-erp-domain", "ack-erp-persistence-port"}, []string{
		"internal/adapters/http",
		"internal/app/bootstrap",
		"cmd/server",
	})
	assertAppUnitForTestV0(t, plan, "integration", orquestacoreworkflow.OrchestrationPhaseIntegracionV0, []string{
		"ack-erp-domain",
		"ack-erp-persistence-port",
		"ack-erp-api",
		"ack-erp-web",
		"ack-erp-i18n",
	}, []string{"internal/app", "internal/integration"})
	assertAppUnitForTestV0(t, plan, "deploy", orquestacoreworkflow.OrchestrationPhaseIntegracionV0, []string{"ack-erp-architecture"}, []string{
		"contracts/deployment_plan_v0.json",
		"docs/deploy.md",
		"tests/deployment_plan_dry_run_test.go",
	})
	assertAppUnitForTestV0(t, plan, "docs", orquestacoreworkflow.OrchestrationPhaseDocumentacionV0, []string{"ack-erp-integration", "ack-erp-deploy"}, []string{
		"README.md",
		"docs/operacion.md",
		"docs/pruebas.md",
	})

	review := appUnitByKeyForTestV0(t, plan, "review")
	if review.PhaseID != orquestacoreworkflow.OrchestrationPhaseRevisionV0 {
		t.Fatalf("review phase=%s", review.PhaseID)
	}
	if review.Capacity != orquestacoreworkflow.OrchestrationCapacityXHighV0 {
		t.Fatalf("review capacity=%s", review.Capacity)
	}
	assertLargePlanHasNoDBProviderWriteSetV0(t, plan)
}

func TestBuildGoAPIWebMicrotaskPlanV0ParaAppGrandeExigeHexagonalidadEstructural(t *testing.T) {
	plan, err := BuildGoAPIWebMicrotaskPlanV0(AppPlanRequestV0{
		RunRef:  "run-ref-large-hex-001",
		AppRef:  "erp",
		AppName: "ERP",
		API:     true,
		Web:     true,
		Scale:   AppPlanScaleLargeV0,
	})
	if err != nil {
		t.Fatalf("BuildGoAPIWebMicrotaskPlanV0 large: %v", err)
	}

	architecture := appUnitByKeyForTestV0(t, plan, "architecture")
	assertAppUnitCriteriaContainsForTestV0(t, architecture, "Fronteras domain, application, ports, adapters y bootstrap definidas.")
	assertAppUnitCriteriaContainsForTestV0(t, architecture, "Handlers/adaptadores declarados como finos y sin composicion global.")

	domain := appUnitByKeyForTestV0(t, plan, "domain")
	if !sameStringSetForTestV0(domain.WriteSet, []string{"internal/domain", "internal/application", "internal/ports"}) {
		t.Fatalf("domain write_set=%v", domain.WriteSet)
	}
	assertAppUnitCriteriaContainsForTestV0(t, domain, "Application/casos de uso dependen de puertos/interfaces, no de adaptadores concretos.")

	api := appUnitByKeyForTestV0(t, plan, "api")
	assertAppUnitCriteriaContainsForTestV0(t, api, "Handlers no construyen repositorios, autenticacion, fixtures ni reglas de negocio.")

	integration := appUnitByKeyForTestV0(t, plan, "integration")
	assertAppUnitCriteriaContainsForTestV0(t, integration, "Composicion de repositorios, adaptadores, autenticacion y configuracion centralizada en bootstrap/cmd.")

	review := appUnitByKeyForTestV0(t, plan, "review")
	assertAppUnitCriteriaContainsForTestV0(t, review, "No se acepta si domain/application importan adapters, HTTP, DB, filesystem, runtime o UI.")
	assertAppUnitCriteriaContainsForTestV0(t, review, "No se acepta si handlers contienen composicion de repositorios, autenticacion, fixtures o reglas de negocio.")
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
