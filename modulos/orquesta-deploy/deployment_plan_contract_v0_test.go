package orquestadeploy

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeDeploymentPlanV0FixturesValidos(t *testing.T) {
	for _, name := range []string{"contenedor_valido.json", "desktop_valido.json", "kubernetes_valido.json", "local_valido.json", "mobile_store_valido.json", "paas_valido.json"} {
		t.Run(name, func(t *testing.T) {
			plan, err := DecodeDeploymentPlanV0(readDeploymentPlanFixtureV0(t, name))
			if err != nil {
				t.Fatalf("fixture should validate: %v", err)
			}
			if plan.SchemaVersion != DeploymentPlanSchemaV0 {
				t.Fatalf("schema_version=%q, want %q", plan.SchemaVersion, DeploymentPlanSchemaV0)
			}
			if !osExactosV0(plan.SistemasOperativos) || !matrizOSExactaV0(plan.MatrizOS) {
				t.Fatalf("expected explicit linux/darwin/windows matrix: %+v %+v", plan.SistemasOperativos, plan.MatrizOS)
			}
		})
	}
}

func TestValidateDeploymentPlanV0ErroresRelacionales(t *testing.T) {
	base := mustDecodeDeploymentPlanFixtureV0(t, "local_valido.json")
	contenedor := mustDecodeDeploymentPlanFixtureV0(t, "contenedor_valido.json")

	tests := []struct {
		name string
		plan DeploymentPlanV0
		code string
	}{
		{
			name: "sistemas_operativos incompletos",
			plan: mutateDeploymentPlanV0(base, func(plan *DeploymentPlanV0) {
				plan.SistemasOperativos = []string{"linux", "darwin"}
			}),
			code: ErrMatrizOSIncompletaV0,
		},
		{
			name: "matriz_os incompleta",
			plan: mutateDeploymentPlanV0(base, func(plan *DeploymentPlanV0) {
				plan.MatrizOS[2].OS = "linux"
			}),
			code: ErrMatrizOSIncompletaV0,
		},
		{
			name: "target resuelto distinto",
			plan: mutateDeploymentPlanV0(base, func(plan *DeploymentPlanV0) {
				plan.TargetResuelto = "contenedor"
			}),
			code: ErrDeployTargetNoSoportadoV0,
		},
		{
			name: "contenedor no aceptado",
			plan: mutateDeploymentPlanV0(contenedor, func(plan *DeploymentPlanV0) {
				plan.AceptacionContenedor = false
			}),
			code: ErrContenedorNoAceptadoV0,
		},
		{
			name: "contenedor sin accion que lo requiera",
			plan: mutateDeploymentPlanV0(contenedor, func(plan *DeploymentPlanV0) {
				for index := range plan.AccionesPrevistas {
					plan.AccionesPrevistas[index].RequiereContenedor = false
				}
			}),
			code: ErrContenedorNoAceptadoV0,
		},
		{
			name: "local false con accion que requiere contenedor",
			plan: mutateDeploymentPlanV0(base, func(plan *DeploymentPlanV0) {
				plan.AccionesPrevistas[0].RequiereContenedor = true
			}),
			code: ErrContenedorNoAceptadoV0,
		},
		{
			name: "accion ejecutable no declarativa",
			plan: mutateDeploymentPlanV0(base, func(plan *DeploymentPlanV0) {
				plan.AccionesPrevistas[0].Tipo = "ejecutar_script"
			}),
			code: ErrScriptSinContratoV0,
		},
		{
			name: "validacion entorno vacia",
			plan: mutateDeploymentPlanV0(base, func(plan *DeploymentPlanV0) {
				plan.ValidacionEntorno = nil
			}),
			code: ErrValidacionEntornoIncompletaV0,
		},
		{
			name: "healthcheck vacio",
			plan: mutateDeploymentPlanV0(base, func(plan *DeploymentPlanV0) {
				plan.Healthcheck = nil
			}),
			code: ErrHealthcheckIncompletoV0,
		},
		{
			name: "rollback vacio",
			plan: mutateDeploymentPlanV0(base, func(plan *DeploymentPlanV0) {
				plan.Rollback.PasosPrevistos = nil
			}),
			code: ErrRollbackIncompletoV0,
		},
		{
			name: "artefactos previstos vacios",
			plan: mutateDeploymentPlanV0(base, func(plan *DeploymentPlanV0) {
				plan.ArtefactosPrevistos = nil
			}),
			code: ErrArtefactosPrevistosIncompletosV0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateDeploymentPlanV0(test.plan)
			assertDeploymentPlanIssueV0(t, err, test.code)
		})
	}
}

func TestDecodeDeploymentPlanV0ErroresInline(t *testing.T) {
	base := mustDecodeDeploymentPlanFixtureV0(t, "local_valido.json")

	t.Run("target requerido", func(t *testing.T) {
		plan := mutateDeploymentPlanV0(base, func(plan *DeploymentPlanV0) {
			plan.DeployTarget = " "
		})
		_, err := DecodeDeploymentPlanV0(mustMarshalDeploymentPlanV0(t, plan))
		assertDeploymentPlanIssueV0(t, err, ErrDeployTargetRequeridoV0)
	})

	t.Run("target no soportado", func(t *testing.T) {
		plan := mutateDeploymentPlanV0(base, func(plan *DeploymentPlanV0) {
			plan.DeployTarget = "maquina_inventada"
		})
		_, err := DecodeDeploymentPlanV0(mustMarshalDeploymentPlanV0(t, plan))
		assertDeploymentPlanIssueV0(t, err, ErrDeployTargetNoSoportadoV0)
	})
}

func readDeploymentPlanFixtureV0(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("docs", "fixtures", "deployment_plan_v0", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func mustDecodeDeploymentPlanFixtureV0(t *testing.T, name string) DeploymentPlanV0 {
	t.Helper()
	plan, err := DecodeDeploymentPlanV0(readDeploymentPlanFixtureV0(t, name))
	if err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
	return plan
}

func mutateDeploymentPlanV0(plan DeploymentPlanV0, mutate func(*DeploymentPlanV0)) DeploymentPlanV0 {
	data, err := json.Marshal(plan)
	if err != nil {
		panic(err)
	}
	var clone DeploymentPlanV0
	if err := json.Unmarshal(data, &clone); err != nil {
		panic(err)
	}
	mutate(&clone)
	return clone
}

func mustMarshalDeploymentPlanV0(t *testing.T, plan DeploymentPlanV0) []byte {
	t.Helper()
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}
	return data
}

func assertDeploymentPlanIssueV0(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q", want)
	}
	var contractErr DeploymentPlanContractV0Error
	if !errors.As(err, &contractErr) {
		t.Fatalf("error type=%T, want DeploymentPlanContractV0Error", err)
	}
	for _, issue := range contractErr.Issues {
		if issue.Code == want {
			return
		}
	}
	t.Fatalf("missing issue %q in %+v", want, contractErr.Issues)
}
