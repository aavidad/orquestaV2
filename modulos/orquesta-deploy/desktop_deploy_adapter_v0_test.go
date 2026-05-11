package orquestadeploy

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestDesktopDeployAdapterV0PrepareDryRun(t *testing.T) {
	plan := mustDecodeDeploymentPlanFixtureV0(t, "desktop_valido.json")

	result, err := PrepareDesktopDeployDryRunV0(plan)
	if err != nil {
		t.Fatalf("prepare dry-run: %v", err)
	}
	if result.SchemaVersion != DesktopDeployResultSchemaV0 {
		t.Fatalf("schema_version=%q, want %q", result.SchemaVersion, DesktopDeployResultSchemaV0)
	}
	if result.Adapter != DesktopDeployAdapterNameV0 {
		t.Fatalf("adapter=%q, want %q", result.Adapter, DesktopDeployAdapterNameV0)
	}
	if result.Mode != DesktopDeployModeDryRunV0 || result.Status != DesktopDeployStatusPreparadoV0 {
		t.Fatalf("mode/status=%q/%q, want dry-run/preparado", result.Mode, result.Status)
	}
	if result.PlanID != plan.PlanID || result.TargetResuelto != "desktop" {
		t.Fatalf("unexpected plan/target: %+v", result)
	}
	if !desktopDeployOSExactosV0(result.OSPreparados) {
		t.Fatalf("expected explicit prepared OS matrix: %+v", result.OSPreparados)
	}
	if !desktopHasPackageArtifactV0(result.ArtefactosPrevistos) {
		t.Fatalf("expected desktop package artifact: %+v", result.ArtefactosPrevistos)
	}
	if _, err := json.Marshal(result); err != nil {
		t.Fatalf("result should be JSON serializable: %v", err)
	}

	plan.ArtefactosPrevistos[0].ID = "mutado"
	if result.ArtefactosPrevistos[0].ID == "mutado" {
		t.Fatalf("result should not alias plan slices")
	}
}

func TestDesktopDeployAdapterV0RejectsInvalidPlans(t *testing.T) {
	desktop := mustDecodeDeploymentPlanFixtureV0(t, "desktop_valido.json")
	local := mustDecodeDeploymentPlanFixtureV0(t, "local_valido.json")

	tests := []struct {
		name       string
		plan       DeploymentPlanV0
		code       string
		deployment bool
	}{
		{
			name: "deployment plan invalido",
			plan: mutateDeploymentPlanV0(desktop, func(plan *DeploymentPlanV0) {
				plan.DeployTarget = " "
			}),
			code:       ErrDeployTargetRequeridoV0,
			deployment: true,
		},
		{
			name: "target no desktop",
			plan: local,
			code: ErrDesktopDeployTargetInvalidoV0,
		},
		{
			name: "os no aplicable",
			plan: mutateDeploymentPlanV0(desktop, func(plan *DeploymentPlanV0) {
				plan.MatrizOS[0].Aplica = false
			}),
			code: ErrMatrizOSIncompletaV0,
		},
		{
			name: "rollback no reversible",
			plan: mutateDeploymentPlanV0(desktop, func(plan *DeploymentPlanV0) {
				plan.Rollback.Reversible = false
			}),
			code: ErrDesktopDeployRollbackNoReversibleV0,
		},
		{
			name: "sin paquete desktop",
			plan: mutateDeploymentPlanV0(desktop, func(plan *DeploymentPlanV0) {
				plan.ArtefactosPrevistos[0].Tipo = "metadata"
			}),
			code: ErrDesktopDeploySinPaqueteV0,
		},
		{
			name: "contenedor no puro",
			plan: mutateDeploymentPlanV0(desktop, func(plan *DeploymentPlanV0) {
				plan.AccionesPrevistas[0].RequiereContenedor = true
			}),
			code: ErrDesktopDeployContenedorNoPuroV0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := DesktopDeployAdapterV0{}.PrepareDryRun(test.plan)
			if test.deployment {
				assertDeploymentPlanIssueV0(t, err, test.code)
				return
			}
			assertDesktopDeployIssueV0(t, err, test.code)
		})
	}
}

func TestValidateDesktopDeployResultV0(t *testing.T) {
	result, err := PrepareDesktopDeployDryRunV0(mustDecodeDeploymentPlanFixtureV0(t, "desktop_valido.json"))
	if err != nil {
		t.Fatalf("prepare dry-run: %v", err)
	}
	if err := ValidateDesktopDeployResultV0(result); err != nil {
		t.Fatalf("result should validate: %v", err)
	}

	result.TargetResuelto = "mobile_store"
	assertDesktopDeployIssueV0(t, ValidateDesktopDeployResultV0(result), ErrDesktopDeployTargetInvalidoV0)
}

func assertDesktopDeployIssueV0(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q", want)
	}
	var desktopErr DesktopDeployAdapterV0Error
	if !errors.As(err, &desktopErr) {
		t.Fatalf("error type=%T, want DesktopDeployAdapterV0Error", err)
	}
	for _, issue := range desktopErr.Issues {
		if issue.Code == want {
			return
		}
	}
	t.Fatalf("missing issue %q in %+v", want, desktopErr.Issues)
}
