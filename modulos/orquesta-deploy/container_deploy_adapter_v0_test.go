package orquestadeploy

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestContainerDeployAdapterV0PrepareDryRun(t *testing.T) {
	plan := mustDecodeDeploymentPlanFixtureV0(t, "contenedor_valido.json")

	result, err := PrepareContainerDeployDryRunV0(plan)
	if err != nil {
		t.Fatalf("prepare dry-run: %v", err)
	}
	if result.SchemaVersion != ContainerDeployResultSchemaV0 {
		t.Fatalf("schema_version=%q, want %q", result.SchemaVersion, ContainerDeployResultSchemaV0)
	}
	if result.Adapter != ContainerDeployAdapterNameV0 {
		t.Fatalf("adapter=%q, want %q", result.Adapter, ContainerDeployAdapterNameV0)
	}
	if result.Mode != ContainerDeployModeDryRunV0 || result.Status != ContainerDeployStatusPreparadoV0 {
		t.Fatalf("mode/status=%q/%q, want dry-run/preparado", result.Mode, result.Status)
	}
	if result.PlanID != plan.PlanID || result.TargetResuelto != "contenedor" {
		t.Fatalf("unexpected plan/target: %+v", result)
	}
	if !containerDeployOSExactosV0(result.OSPreparados) {
		t.Fatalf("expected explicit prepared OS matrix: %+v", result.OSPreparados)
	}
	if len(result.ValidacionEntorno) != len(plan.ValidacionEntorno) {
		t.Fatalf("validacion_entorno len=%d, want %d", len(result.ValidacionEntorno), len(plan.ValidacionEntorno))
	}
	if len(result.ArtefactosPrevistos) != len(plan.ArtefactosPrevistos) {
		t.Fatalf("artefactos_previstos len=%d, want %d", len(result.ArtefactosPrevistos), len(plan.ArtefactosPrevistos))
	}
	if !result.Rollback.Reversible {
		t.Fatalf("rollback should be reversible in dry-run result")
	}
	if !algunaAccionRequiereContenedorV0(result.AccionesPrevistas) {
		t.Fatalf("container dry-run result should require at least one container action: %+v", result.AccionesPrevistas)
	}
	if _, err := json.Marshal(result); err != nil {
		t.Fatalf("result should be JSON serializable: %v", err)
	}

	plan.ValidacionEntorno[0].ID = "mutado"
	if result.ValidacionEntorno[0].ID == "mutado" {
		t.Fatalf("result should not alias plan slices")
	}
}

func TestContainerDeployAdapterV0RejectsInvalidPlans(t *testing.T) {
	container := mustDecodeDeploymentPlanFixtureV0(t, "contenedor_valido.json")
	local := mustDecodeDeploymentPlanFixtureV0(t, "local_valido.json")

	tests := []struct {
		name       string
		plan       DeploymentPlanV0
		code       string
		deployment bool
	}{
		{
			name: "deployment plan invalido",
			plan: mutateDeploymentPlanV0(container, func(plan *DeploymentPlanV0) {
				plan.DeployTarget = " "
			}),
			code:       ErrDeployTargetRequeridoV0,
			deployment: true,
		},
		{
			name: "target no contenedor",
			plan: local,
			code: ErrContainerDeployTargetInvalidoV0,
		},
		{
			name: "contenedor no aceptado",
			plan: mutateDeploymentPlanV0(container, func(plan *DeploymentPlanV0) {
				plan.AceptacionContenedor = false
			}),
			code:       ErrContenedorNoAceptadoV0,
			deployment: true,
		},
		{
			name: "rollback no reversible",
			plan: mutateDeploymentPlanV0(container, func(plan *DeploymentPlanV0) {
				plan.Rollback.Reversible = false
			}),
			code: ErrContainerDeployRollbackNoReversibleV0,
		},
		{
			name: "sin accion de contenedor",
			plan: mutateDeploymentPlanV0(container, func(plan *DeploymentPlanV0) {
				for index := range plan.AccionesPrevistas {
					plan.AccionesPrevistas[index].RequiereContenedor = false
				}
			}),
			code:       ErrContenedorNoAceptadoV0,
			deployment: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ContainerDeployAdapterV0{}.PrepareDryRun(test.plan)
			if test.deployment {
				assertDeploymentPlanIssueV0(t, err, test.code)
				return
			}
			assertContainerDeployIssueV0(t, err, test.code)
		})
	}
}

func TestValidateContainerDeployResultV0(t *testing.T) {
	result, err := PrepareContainerDeployDryRunV0(mustDecodeDeploymentPlanFixtureV0(t, "contenedor_valido.json"))
	if err != nil {
		t.Fatalf("prepare dry-run: %v", err)
	}
	if err := ValidateContainerDeployResultV0(result); err != nil {
		t.Fatalf("result should validate: %v", err)
	}

	result.TargetResuelto = "local"
	assertContainerDeployIssueV0(t, ValidateContainerDeployResultV0(result), ErrContainerDeployTargetInvalidoV0)
}

func assertContainerDeployIssueV0(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q", want)
	}
	var containerErr ContainerDeployAdapterV0Error
	if !errors.As(err, &containerErr) {
		t.Fatalf("error type=%T, want ContainerDeployAdapterV0Error", err)
	}
	for _, issue := range containerErr.Issues {
		if issue.Code == want {
			return
		}
	}
	t.Fatalf("missing issue %q in %+v", want, containerErr.Issues)
}
