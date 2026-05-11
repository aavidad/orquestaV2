package orquestadeploy

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestLocalDeployAdapterV0PrepareDryRun(t *testing.T) {
	plan := mustDecodeDeploymentPlanFixtureV0(t, "local_valido.json")

	result, err := PrepareLocalDeployDryRunV0(plan)
	if err != nil {
		t.Fatalf("prepare dry-run: %v", err)
	}
	if result.SchemaVersion != LocalDeployResultSchemaV0 {
		t.Fatalf("schema_version=%q, want %q", result.SchemaVersion, LocalDeployResultSchemaV0)
	}
	if result.Adapter != LocalDeployAdapterNameV0 {
		t.Fatalf("adapter=%q, want %q", result.Adapter, LocalDeployAdapterNameV0)
	}
	if result.Mode != LocalDeployModeDryRunV0 || result.Status != LocalDeployStatusPreparadoV0 {
		t.Fatalf("mode/status=%q/%q, want dry-run/preparado", result.Mode, result.Status)
	}
	if result.PlanID != plan.PlanID || result.TargetResuelto != "local" {
		t.Fatalf("unexpected plan/target: %+v", result)
	}
	if !localDeployOSExactosV0(result.OSPreparados) {
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
	if algunaAccionRequiereContenedorV0(result.AccionesPrevistas) || algunaAccionRequiereContenedorV0(result.Rollback.PasosPrevistos) {
		t.Fatalf("local dry-run result should not require container: %+v", result.AccionesPrevistas)
	}
	if _, err := json.Marshal(result); err != nil {
		t.Fatalf("result should be JSON serializable: %v", err)
	}

	plan.ValidacionEntorno[0].ID = "mutado"
	if result.ValidacionEntorno[0].ID == "mutado" {
		t.Fatalf("result should not alias plan slices")
	}
}

func TestLocalDeployAdapterV0RejectsInvalidPlans(t *testing.T) {
	local := mustDecodeDeploymentPlanFixtureV0(t, "local_valido.json")
	container := mustDecodeDeploymentPlanFixtureV0(t, "contenedor_valido.json")

	tests := []struct {
		name       string
		plan       DeploymentPlanV0
		code       string
		deployment bool
	}{
		{
			name: "deployment plan invalido",
			plan: mutateDeploymentPlanV0(local, func(plan *DeploymentPlanV0) {
				plan.DeployTarget = " "
			}),
			code:       ErrDeployTargetRequeridoV0,
			deployment: true,
		},
		{
			name: "target no local",
			plan: container,
			code: ErrLocalDeployTargetNoLocalV0,
		},
		{
			name: "os no aplicable",
			plan: mutateDeploymentPlanV0(local, func(plan *DeploymentPlanV0) {
				plan.MatrizOS[0].Aplica = false
			}),
			code: ErrMatrizOSIncompletaV0,
		},
		{
			name: "rollback no reversible",
			plan: mutateDeploymentPlanV0(local, func(plan *DeploymentPlanV0) {
				plan.Rollback.Reversible = false
			}),
			code: ErrLocalDeployRollbackNoReversibleV0,
		},
		{
			name: "contenedor en local puro",
			plan: mutateDeploymentPlanV0(local, func(plan *DeploymentPlanV0) {
				plan.AceptacionContenedor = true
				plan.AccionesPrevistas[0].RequiereContenedor = true
			}),
			code: ErrLocalDeployContenedorNoPuroV0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LocalDeployAdapterV0{}.PrepareDryRun(test.plan)
			if test.deployment {
				assertDeploymentPlanIssueV0(t, err, test.code)
				return
			}
			assertLocalDeployIssueV0(t, err, test.code)
		})
	}
}

func TestValidateLocalDeployResultV0(t *testing.T) {
	result, err := PrepareLocalDeployDryRunV0(mustDecodeDeploymentPlanFixtureV0(t, "local_valido.json"))
	if err != nil {
		t.Fatalf("prepare dry-run: %v", err)
	}
	if err := ValidateLocalDeployResultV0(result); err != nil {
		t.Fatalf("result should validate: %v", err)
	}

	result.TargetResuelto = "contenedor"
	assertLocalDeployIssueV0(t, ValidateLocalDeployResultV0(result), ErrLocalDeployTargetNoLocalV0)
}

func assertLocalDeployIssueV0(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q", want)
	}
	var localErr LocalDeployAdapterV0Error
	if !errors.As(err, &localErr) {
		t.Fatalf("error type=%T, want LocalDeployAdapterV0Error", err)
	}
	for _, issue := range localErr.Issues {
		if issue.Code == want {
			return
		}
	}
	t.Fatalf("missing issue %q in %+v", want, localErr.Issues)
}
