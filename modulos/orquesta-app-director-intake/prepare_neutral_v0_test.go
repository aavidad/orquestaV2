package orquestaappdirectorintake

import (
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestPrepareAppDirectorInputV0UsesNeutralSpecWithoutFactory(t *testing.T) {
	prepared, err := PrepareAppDirectorInputV0(PrepareAppDirectorInputRequestV0{
		AppSpec: AppDirectorInputSpecV0{
			SchemaVersion: AppDirectorInputSpecSchemaVersionV0,
			SpecID:        "app-spec-neutral-director-001",
			CreatedAt:     "2026-05-23T11:00:00Z",
			App: AppDirectorInputAppV0{
				Nombre:      "Portal interno",
				Objetivo:    "Coordinar solicitudes internas.",
				Descripcion: "API y web pequenas con puertos claros.",
				TipoApp:     "mixed",
				Slug:        "portal-interno",
			},
			Platforms: []string{"web", "api"},
			Data: AppDirectorInputDataV0{
				Needs:               []string{"Persistir solicitudes con un puerto inyectado."},
				PersistenceRequired: true,
			},
			I18N:             AppDirectorInputI18NV0{Enabled: true},
			Quality:          AppDirectorInputQualityV0{Tests: "alta"},
			AgentPreferences: AppDirectorInputAgentPreferencesV0{Autonomy: "alta"},
			RequestKind:      AppDirectorRequestKindDocumentarAppV0,
			ExecutionMode:    AppDirectorExecutionModeDebugV0,
			Validation:       AppDirectorInputValidationV0{Estado: "valida"},
		},
	})
	if err != nil {
		t.Fatalf("PrepareAppDirectorInputV0: %v", err)
	}
	if prepared.Run.AppSpecRef != "app-spec-neutral-director-001" {
		t.Fatalf("app_spec_ref=%q", prepared.Run.AppSpecRef)
	}
	if prepared.DirectorTask.Capacity != orquestacoreworkflow.OrchestrationCapacityHighV0 {
		t.Fatalf("capacity=%q", prepared.DirectorTask.Capacity)
	}
	if len(prepared.DirectorTasks) != 4 {
		t.Fatalf("director_tasks=%d %+v", len(prepared.DirectorTasks), prepared.DirectorTasks)
	}
	for _, want := range []string{
		"request_kind=documentar_app",
		"execution_mode=debug",
		"app=Portal interno",
		"plataformas=web,api",
		"datos=Persistir solicitudes con un puerto inyectado.",
	} {
		if !strings.Contains(prepared.DirectorTask.Summary, want) {
			t.Fatalf("summary no contiene %q:\n%s", want, prepared.DirectorTask.Summary)
		}
	}
}
