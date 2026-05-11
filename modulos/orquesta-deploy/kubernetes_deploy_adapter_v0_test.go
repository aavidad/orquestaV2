package orquestadeploy

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestKubernetesDeployAdapterV0PrepareDryRun(t *testing.T) {
	plan := mustDecodeDeploymentPlanFixtureV0(t, "kubernetes_valido.json")

	result, err := PrepareKubernetesDeployDryRunV0(plan)
	if err != nil {
		t.Fatalf("prepare dry-run: %v", err)
	}
	if result.SchemaVersion != KubernetesDeployResultSchemaV0 {
		t.Fatalf("schema_version=%q, want %q", result.SchemaVersion, KubernetesDeployResultSchemaV0)
	}
	if result.Adapter != KubernetesDeployAdapterNameV0 {
		t.Fatalf("adapter=%q, want %q", result.Adapter, KubernetesDeployAdapterNameV0)
	}
	if result.Mode != KubernetesDeployModeDryRunV0 || result.Status != KubernetesDeployStatusPreparadoV0 {
		t.Fatalf("mode/status=%q/%q, want dry-run/preparado", result.Mode, result.Status)
	}
	if result.PlanID != plan.PlanID || result.TargetResuelto != "kubernetes" {
		t.Fatalf("unexpected plan/target: %+v", result)
	}
	if !kubernetesDeployOSExactosV0(result.OSPreparados) {
		t.Fatalf("expected explicit prepared OS matrix: %+v", result.OSPreparados)
	}
	if !kubernetesMatrixHasClientRowsV0(result.OSPreparados) {
		t.Fatalf("expected darwin/windows client rows: %+v", result.OSPreparados)
	}
	if !algunaAccionTipoV0(result.AccionesPrevistas, "publicar_declarativo") {
		t.Fatalf("expected at least one publicar_declarativo action: %+v", result.AccionesPrevistas)
	}
	if _, err := json.Marshal(result); err != nil {
		t.Fatalf("result should be JSON serializable: %v", err)
	}

	plan.Healthcheck[0].ID = "mutado"
	if result.Healthcheck[0].ID == "mutado" {
		t.Fatalf("result should not alias plan slices")
	}
}

func TestKubernetesDeployAdapterV0RejectsInvalidPlans(t *testing.T) {
	kubernetes := mustDecodeDeploymentPlanFixtureV0(t, "kubernetes_valido.json")
	local := mustDecodeDeploymentPlanFixtureV0(t, "local_valido.json")

	tests := []struct {
		name       string
		plan       DeploymentPlanV0
		code       string
		deployment bool
	}{
		{
			name: "deployment plan invalido",
			plan: mutateDeploymentPlanV0(kubernetes, func(plan *DeploymentPlanV0) {
				plan.DeployTarget = " "
			}),
			code:       ErrDeployTargetRequeridoV0,
			deployment: true,
		},
		{
			name: "target no kubernetes",
			plan: local,
			code: ErrKubernetesDeployTargetInvalidoV0,
		},
		{
			name: "sin publicacion declarativa",
			plan: mutateDeploymentPlanV0(kubernetes, func(plan *DeploymentPlanV0) {
				for index := range plan.AccionesPrevistas {
					plan.AccionesPrevistas[index].Tipo = "validar_entorno"
				}
			}),
			code: ErrKubernetesDeploySinPublicacionV0,
		},
		{
			name: "rollback no reversible",
			plan: mutateDeploymentPlanV0(kubernetes, func(plan *DeploymentPlanV0) {
				plan.Rollback.Reversible = false
			}),
			code: ErrKubernetesDeployRollbackNoReversibleV0,
		},
		{
			name: "sin filas cliente",
			plan: mutateDeploymentPlanV0(kubernetes, func(plan *DeploymentPlanV0) {
				plan.MatrizOS[1].Motivo = "Nodo generico sin distincion."
				plan.MatrizOS[2].Motivo = "Nodo generico sin distincion."
			}),
			code: ErrKubernetesDeployMatrizClienteV0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := KubernetesDeployAdapterV0{}.PrepareDryRun(test.plan)
			if test.deployment {
				assertDeploymentPlanIssueV0(t, err, test.code)
				return
			}
			assertKubernetesDeployIssueV0(t, err, test.code)
		})
	}
}

func TestValidateKubernetesDeployResultV0(t *testing.T) {
	result, err := PrepareKubernetesDeployDryRunV0(mustDecodeDeploymentPlanFixtureV0(t, "kubernetes_valido.json"))
	if err != nil {
		t.Fatalf("prepare dry-run: %v", err)
	}
	if err := ValidateKubernetesDeployResultV0(result); err != nil {
		t.Fatalf("result should validate: %v", err)
	}

	result.TargetResuelto = "contenedor"
	assertKubernetesDeployIssueV0(t, ValidateKubernetesDeployResultV0(result), ErrKubernetesDeployTargetInvalidoV0)
}

func assertKubernetesDeployIssueV0(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q", want)
	}
	var kubernetesErr KubernetesDeployAdapterV0Error
	if !errors.As(err, &kubernetesErr) {
		t.Fatalf("error type=%T, want KubernetesDeployAdapterV0Error", err)
	}
	for _, issue := range kubernetesErr.Issues {
		if issue.Code == want {
			return
		}
	}
	t.Fatalf("missing issue %q in %+v", want, kubernetesErr.Issues)
}
