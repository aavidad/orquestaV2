package orquestadeploy

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestPaaSDeployAdapterV0PrepareDryRun(t *testing.T) {
	plan := mustDecodeDeploymentPlanFixtureV0(t, "paas_valido.json")

	result, err := PreparePaaSDeployDryRunV0(plan)
	if err != nil {
		t.Fatalf("prepare dry-run: %v", err)
	}
	if result.SchemaVersion != PaaSDeployResultSchemaV0 {
		t.Fatalf("schema_version=%q, want %q", result.SchemaVersion, PaaSDeployResultSchemaV0)
	}
	if result.Adapter != PaaSDeployAdapterNameV0 {
		t.Fatalf("adapter=%q, want %q", result.Adapter, PaaSDeployAdapterNameV0)
	}
	if result.Mode != PaaSDeployModeDryRunV0 || result.Status != PaaSDeployStatusPreparadoV0 {
		t.Fatalf("mode/status=%q/%q, want dry-run/preparado", result.Mode, result.Status)
	}
	if result.PlanID != plan.PlanID || result.TargetResuelto != "paas" {
		t.Fatalf("unexpected plan/target: %+v", result)
	}
	if !paasDeployOSExactosV0(result.OSPreparados) || !paasMatrixHasClientRowsV0(result.OSPreparados) {
		t.Fatalf("expected explicit prepared OS client matrix: %+v", result.OSPreparados)
	}
	if !algunaAccionTipoV0(result.AccionesPrevistas, "publicar_declarativo") {
		t.Fatalf("expected declarative publish action: %+v", result.AccionesPrevistas)
	}
	if _, err := json.Marshal(result); err != nil {
		t.Fatalf("result should be JSON serializable: %v", err)
	}

	plan.ArtefactosPrevistos[0].ID = "mutado"
	if result.ArtefactosPrevistos[0].ID == "mutado" {
		t.Fatalf("result should not alias plan slices")
	}
}

func TestPaaSDeployAdapterV0RejectsInvalidPlans(t *testing.T) {
	paas := mustDecodeDeploymentPlanFixtureV0(t, "paas_valido.json")
	local := mustDecodeDeploymentPlanFixtureV0(t, "local_valido.json")

	tests := []struct {
		name       string
		plan       DeploymentPlanV0
		code       string
		deployment bool
	}{
		{
			name: "deployment plan invalido",
			plan: mutateDeploymentPlanV0(paas, func(plan *DeploymentPlanV0) {
				plan.DeployTarget = " "
			}),
			code:       ErrDeployTargetRequeridoV0,
			deployment: true,
		},
		{
			name: "target no paas",
			plan: local,
			code: ErrPaaSDeployTargetInvalidoV0,
		},
		{
			name: "matriz sin filas cliente",
			plan: mutateDeploymentPlanV0(paas, func(plan *DeploymentPlanV0) {
				plan.MatrizOS[1].Motivo = "Preparacion generica sin clasificacion declarada."
				plan.MatrizOS[2].Motivo = "Preparacion generica sin clasificacion declarada."
			}),
			code: ErrPaaSDeployMatrizClienteV0,
		},
		{
			name: "rollback no reversible",
			plan: mutateDeploymentPlanV0(paas, func(plan *DeploymentPlanV0) {
				plan.Rollback.Reversible = false
			}),
			code: ErrPaaSDeployRollbackNoReversibleV0,
		},
		{
			name: "proveedor no declarado",
			plan: mutateDeploymentPlanV0(paas, func(plan *DeploymentPlanV0) {
				plan.Restricciones = nil
			}),
			code: ErrPaaSDeployProveedorNoDeclaradoV0,
		},
		{
			name: "sin publicacion declarativa",
			plan: mutateDeploymentPlanV0(paas, func(plan *DeploymentPlanV0) {
				plan.AccionesPrevistas[2].Tipo = "preparar_artefactos"
			}),
			code: ErrPaaSDeploySinPublicacionV0,
		},
		{
			name: "contenedor no puro",
			plan: mutateDeploymentPlanV0(paas, func(plan *DeploymentPlanV0) {
				plan.AccionesPrevistas[0].RequiereContenedor = true
			}),
			code: ErrPaaSDeployContenedorNoPuroV0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := PaaSDeployAdapterV0{}.PrepareDryRun(test.plan)
			if test.deployment {
				assertDeploymentPlanIssueV0(t, err, test.code)
				return
			}
			assertPaaSDeployIssueV0(t, err, test.code)
		})
	}
}

func TestValidatePaaSDeployResultV0(t *testing.T) {
	result, err := PreparePaaSDeployDryRunV0(mustDecodeDeploymentPlanFixtureV0(t, "paas_valido.json"))
	if err != nil {
		t.Fatalf("prepare dry-run: %v", err)
	}
	if err := ValidatePaaSDeployResultV0(result); err != nil {
		t.Fatalf("result should validate: %v", err)
	}

	result.TargetResuelto = "serverless"
	assertPaaSDeployIssueV0(t, ValidatePaaSDeployResultV0(result), ErrPaaSDeployTargetInvalidoV0)
}

func assertPaaSDeployIssueV0(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q", want)
	}
	var paasErr PaaSDeployAdapterV0Error
	if !errors.As(err, &paasErr) {
		t.Fatalf("error type=%T, want PaaSDeployAdapterV0Error", err)
	}
	for _, issue := range paasErr.Issues {
		if issue.Code == want {
			return
		}
	}
	t.Fatalf("missing issue %q in %+v", want, paasErr.Issues)
}
