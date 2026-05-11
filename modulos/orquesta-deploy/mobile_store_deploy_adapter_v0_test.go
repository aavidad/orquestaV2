package orquestadeploy

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestMobileStoreDeployAdapterV0PrepareDryRun(t *testing.T) {
	plan := mustDecodeDeploymentPlanFixtureV0(t, "mobile_store_valido.json")

	result, err := PrepareMobileStoreDeployDryRunV0(plan)
	if err != nil {
		t.Fatalf("prepare dry-run: %v", err)
	}
	if result.SchemaVersion != MobileStoreDeployResultSchemaV0 {
		t.Fatalf("schema_version=%q, want %q", result.SchemaVersion, MobileStoreDeployResultSchemaV0)
	}
	if result.Adapter != MobileStoreDeployAdapterNameV0 {
		t.Fatalf("adapter=%q, want %q", result.Adapter, MobileStoreDeployAdapterNameV0)
	}
	if result.Mode != MobileStoreDeployModeDryRunV0 || result.Status != MobileStoreDeployStatusPreparadoV0 {
		t.Fatalf("mode/status=%q/%q, want dry-run/preparado", result.Mode, result.Status)
	}
	if result.PlanID != plan.PlanID || result.TargetResuelto != "mobile_store" {
		t.Fatalf("unexpected plan/target: %+v", result)
	}
	if !mobileStoreDeployOSExactosV0(result.OSPreparados) || !mobileStoreHasDarwinApplicableV0(result.OSPreparados) || !mobileStoreHasClientRowsV0(result.OSPreparados) {
		t.Fatalf("expected valid mobile_store OS matrix: %+v", result.OSPreparados)
	}
	if !mobileStoreHasMetadataArtifactV0(result.ArtefactosPrevistos) {
		t.Fatalf("expected publication metadata artifact: %+v", result.ArtefactosPrevistos)
	}
	if !algunaAccionTipoV0(result.AccionesPrevistas, "solicitar_decision") {
		t.Fatalf("expected at least one solicitar_decision action: %+v", result.AccionesPrevistas)
	}
	if _, err := json.Marshal(result); err != nil {
		t.Fatalf("result should be JSON serializable: %v", err)
	}

	plan.Healthcheck[0].ID = "mutado"
	if result.Healthcheck[0].ID == "mutado" {
		t.Fatalf("result should not alias plan slices")
	}
}

func TestMobileStoreDeployAdapterV0RejectsInvalidPlans(t *testing.T) {
	mobile := mustDecodeDeploymentPlanFixtureV0(t, "mobile_store_valido.json")
	local := mustDecodeDeploymentPlanFixtureV0(t, "local_valido.json")

	tests := []struct {
		name       string
		plan       DeploymentPlanV0
		code       string
		deployment bool
	}{
		{
			name: "deployment plan invalido",
			plan: mutateDeploymentPlanV0(mobile, func(plan *DeploymentPlanV0) {
				plan.DeployTarget = " "
			}),
			code:       ErrDeployTargetRequeridoV0,
			deployment: true,
		},
		{
			name: "target no mobile_store",
			plan: local,
			code: ErrMobileStoreDeployTargetInvalidoV0,
		},
		{
			name: "sin metadata de publicacion",
			plan: mutateDeploymentPlanV0(mobile, func(plan *DeploymentPlanV0) {
				plan.ArtefactosPrevistos[0].Tipo = "paquete_mobile"
			}),
			code: ErrMobileStoreDeploySinMetadataV0,
		},
		{
			name: "sin decision manual",
			plan: mutateDeploymentPlanV0(mobile, func(plan *DeploymentPlanV0) {
				for index := range plan.AccionesPrevistas {
					plan.AccionesPrevistas[index].Tipo = "preparar_artefactos"
				}
			}),
			code: ErrMobileStoreDeploySinDecisionV0,
		},
		{
			name: "rollback no reversible",
			plan: mutateDeploymentPlanV0(mobile, func(plan *DeploymentPlanV0) {
				plan.Rollback.Reversible = false
			}),
			code: ErrMobileStoreDeployRollbackNoReversibleV0,
		},
		{
			name: "matriz sin cliente en linux y windows",
			plan: mutateDeploymentPlanV0(mobile, func(plan *DeploymentPlanV0) {
				plan.MatrizOS[0].Motivo = "No aplica."
				plan.MatrizOS[2].Motivo = "No aplica."
			}),
			code: ErrMobileStoreDeployMatrizInvalidaV0,
		},
		{
			name: "contenedor no puro",
			plan: mutateDeploymentPlanV0(mobile, func(plan *DeploymentPlanV0) {
				plan.AccionesPrevistas[0].RequiereContenedor = true
			}),
			code: ErrMobileStoreDeployContenedorNoPuroV0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := MobileStoreDeployAdapterV0{}.PrepareDryRun(test.plan)
			if test.deployment {
				assertDeploymentPlanIssueV0(t, err, test.code)
				return
			}
			assertMobileStoreDeployIssueV0(t, err, test.code)
		})
	}
}

func TestValidateMobileStoreDeployResultV0(t *testing.T) {
	result, err := PrepareMobileStoreDeployDryRunV0(mustDecodeDeploymentPlanFixtureV0(t, "mobile_store_valido.json"))
	if err != nil {
		t.Fatalf("prepare dry-run: %v", err)
	}
	if err := ValidateMobileStoreDeployResultV0(result); err != nil {
		t.Fatalf("result should validate: %v", err)
	}

	result.TargetResuelto = "desktop"
	assertMobileStoreDeployIssueV0(t, ValidateMobileStoreDeployResultV0(result), ErrMobileStoreDeployTargetInvalidoV0)
}

func assertMobileStoreDeployIssueV0(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q", want)
	}
	var mobileErr MobileStoreDeployAdapterV0Error
	if !errors.As(err, &mobileErr) {
		t.Fatalf("error type=%T, want MobileStoreDeployAdapterV0Error", err)
	}
	for _, issue := range mobileErr.Issues {
		if issue.Code == want {
			return
		}
	}
	t.Fatalf("missing issue %q in %+v", want, mobileErr.Issues)
}
