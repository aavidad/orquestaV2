package orquestaappplanner

import (
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestBuildGoAPIWebMicrotaskPlanFromAppSpecV0UsaContratoFactoryV0(t *testing.T) {
	spec := validFactoryAppSpecForPlannerTestV0(t)

	plan, err := BuildGoAPIWebMicrotaskPlanFromAppSpecV0("run-ref-factory-plan-001", spec)
	if err != nil {
		t.Fatalf("BuildGoAPIWebMicrotaskPlanFromAppSpecV0: %v", err)
	}
	if plan.AppRef != spec.App.Slug || plan.RunRef != "run-ref-factory-plan-001" {
		t.Fatalf("plan refs=%+v spec=%+v", plan, spec.App)
	}
	assertAppUnitForTestV0(t, plan, "bootstrap", orquestacoreworkflow.OrchestrationPhaseProgramacionV0, nil, []string{
		"go.mod",
		"AGENTS.md",
		"README.md",
		"docs/contratos.md",
		"docs/tareas.md",
		"docs/pruebas.md",
		"docs/decisiones.md",
	})
}

func TestAppPlanRequestFromAppSpecV0RechazaSpecNoValidada(t *testing.T) {
	spec := validFactoryAppSpecForPlannerTestV0(t)
	spec.Validation.Estado = "provisional"

	if _, err := AppPlanRequestFromAppSpecV0("run-ref-invalid-001", spec); err == nil {
		t.Fatalf("esperaba error")
	}
}

func TestPrepareDeploymentPlanDryRunFromAppSpecV0InvocaPuertoDeploy(t *testing.T) {
	spec := validFactoryAppSpecForPlannerTestV0(t)
	spec.Deploy.Target = "kubernetes"

	result, err := PrepareDeploymentPlanDryRunFromAppSpecV0("run-ref-deploy-plan-001", spec, nil)
	if err != nil {
		t.Fatalf("PrepareDeploymentPlanDryRunFromAppSpecV0: %v", err)
	}
	if result.Plan.DeployTarget != "kubernetes" ||
		result.Receipt.PlanRef == "" ||
		result.Receipt.EvidenceRef == "" {
		t.Fatalf("dry-run incompleto: %+v", result)
	}

	plan, err := BuildGoAPIWebMicrotaskPlanFromAppSpecV0("run-ref-deploy-plan-001", spec)
	if err != nil {
		t.Fatalf("BuildGoAPIWebMicrotaskPlanFromAppSpecV0: %v", err)
	}
	assertAppUnitForTestV0(t, plan, "deploy", orquestacoreworkflow.OrchestrationPhaseIntegracionV0, []string{
		"ack-agenda-architecture",
	}, []string{
		"contracts/deployment_plan_v0.json",
		"docs/deploy.md",
		"tests/deployment_plan_dry_run_test.go",
	})
}

func validFactoryAppSpecForPlannerTestV0(t *testing.T) orquestafactory.AppSpecV0 {
	t.Helper()
	observability := true
	spec, issues := orquestafactory.SolicitarNuevaAppV0(orquestafactory.AppSpecRequestV0{
		SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
		RequestID:     "request-ref-factory-plan-001",
		Source:        "orquesta-web",
		Locale:        "es-ES",
		Nombre:        "Agenda",
		Objetivo:      "Gestionar contactos y citas desde una API y una web.",
		TipoApp:       "mixed",
		PreferenciasTecnicas: orquestafactory.PreferenciasTecnicasV0{
			Lenguaje:     "go",
			Arquitectura: "hexagonal",
		},
		Calidad: orquestafactory.CalidadRequestV0{
			Pruebas:        "media",
			Accesibilidad:  "basica",
			Observabilidad: &observability,
		},
	}, time.Date(2026, 5, 9, 20, 30, 0, 0, time.UTC))
	if len(issues) != 0 {
		t.Fatalf("SolicitarNuevaAppV0 issues: %+v", issues)
	}
	return spec
}
