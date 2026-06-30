package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestExternalWorkRunProjectWorkDirGuardBloqueaOPESConWorkspaceIncorrectoV0(t *testing.T) {
	next := &fakeExternalWorkRunGuardNextV0{}
	executor := NewExternalWorkRunProjectWorkDirGuardExecutorV0(next, ExternalWorkRunProjectWorkDirGuardConfigV0{
		ProjectWorkDir: "/home/alberto/Trabajo/orquesta",
		Rules: []ExternalWorkRunProjectWorkDirGuardRuleV0{{
			ProjectRef:             "opes",
			RequiredProjectWorkDir: "/home/alberto/Trabajo/OPES",
			EvidenceRef:            "evidence-ref-opes-project-workdir",
		}},
	})

	result, err := executor.Execute(context.Background(), orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID: "req-opes-001",
		ExternalWorkRunRequest: orquestaexternalworkrun.StartExternalWorkRunRequestV0{
			ProjectRef: "opes",
			AppChangeRequest: orquestaappchange.AppChangeRequestV0{
				AppRef: "opes",
			},
		},
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoErrorV0 || len(result.Errores) != 1 {
		t.Fatalf("result=%+v", result)
	}
	if got := result.Errores[0].Code; got != ExternalWorkRunProjectWorkDirMismatchV0 {
		t.Fatalf("code=%q", got)
	}
	if result.Errores[0].Message == "" ||
		result.Errores[0].Field != "project_work_dir:evidence-ref-opes-project-workdir:action_restart_with_required_project_workdir_or_use_local_external_opes_write_set" {
		t.Fatalf("error poco accionable: %+v", result.Errores[0])
	}
	if next.calls != 0 {
		t.Fatalf("next ejecutado con workspace incorrecto: %d", next.calls)
	}
}

func TestExternalWorkRunProjectWorkDirGuardPermiteOPESControlLocalExternalV0(t *testing.T) {
	next := &fakeExternalWorkRunGuardNextV0{
		result: orquestamcp.MCPExternalWorkRunToolResultV0{Estado: orquestamcp.MCPExternalWorkRunEstadoOKV0},
	}
	executor := NewExternalWorkRunProjectWorkDirGuardExecutorV0(next, ExternalWorkRunProjectWorkDirGuardConfigV0{
		ProjectWorkDir: "/home/alberto/Trabajo/orquesta",
		Rules: []ExternalWorkRunProjectWorkDirGuardRuleV0{{
			ProjectRef:                   "opes",
			RequiredProjectWorkDir:       "/home/alberto/Trabajo/OPES",
			EvidenceRef:                  "evidence-ref-opes-project-workdir",
			AllowedLocalWriteSetPrefixes: []string{"external/opes"},
		}},
	})

	result, err := executor.Execute(context.Background(), orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID: "req-opes-control-001",
		AppChangeRequest: orquestaappchange.AppChangeRequestV0{
			AppRef: "opes",
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef: "opes",
				WorkKind:   "generate_audio_asset",
			},
			AllowedWriteSet: []string{
				"external/opes/control_audio_psicologo_orquesta_20260630/informe_control_concurrencia_audio.md",
			},
		},
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 || next.calls != 1 {
		t.Fatalf("result=%+v calls=%d", result, next.calls)
	}
}

func TestExternalWorkRunProjectWorkDirGuardNoPermiteOPESLocalFueraDeExternalV0(t *testing.T) {
	next := &fakeExternalWorkRunGuardNextV0{}
	executor := NewExternalWorkRunProjectWorkDirGuardExecutorV0(next, ExternalWorkRunProjectWorkDirGuardConfigV0{
		ProjectWorkDir: "/home/alberto/Trabajo/orquesta",
		Rules: []ExternalWorkRunProjectWorkDirGuardRuleV0{{
			ProjectRef:                   "opes",
			RequiredProjectWorkDir:       "/home/alberto/Trabajo/OPES",
			AllowedLocalWriteSetPrefixes: []string{"external/opes"},
		}},
	})

	result, err := executor.Execute(context.Background(), orquestamcp.MCPExternalWorkRunToolInputV0{
		AppChangeRequest: orquestaappchange.AppChangeRequestV0{
			AppRef:          "opes",
			AllowedWriteSet: []string{"external/opes/control/informe.md", "docs/fuera.md"},
		},
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != ExternalWorkRunProjectWorkDirMismatchV0 ||
		next.calls != 0 {
		t.Fatalf("result=%+v calls=%d", result, next.calls)
	}
}

func TestExternalWorkRunProjectWorkDirGuardPermiteOPESConWorkspaceCorrectoV0(t *testing.T) {
	next := &fakeExternalWorkRunGuardNextV0{
		result: orquestamcp.MCPExternalWorkRunToolResultV0{Estado: orquestamcp.MCPExternalWorkRunEstadoOKV0},
	}
	executor := NewExternalWorkRunProjectWorkDirGuardExecutorV0(next, ExternalWorkRunProjectWorkDirGuardConfigV0{
		ProjectWorkDir: "/home/alberto/Trabajo/OPES",
		Rules: []ExternalWorkRunProjectWorkDirGuardRuleV0{{
			ProjectRef:             "opes",
			RequiredProjectWorkDir: "/home/alberto/Trabajo/OPES",
		}},
	})

	result, err := executor.Execute(context.Background(), orquestamcp.MCPExternalWorkRunToolInputV0{
		ExternalWorkRunRequest: orquestaexternalworkrun.StartExternalWorkRunRequestV0{
			ProjectRef: "opes",
		},
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 || next.calls != 1 {
		t.Fatalf("result=%+v calls=%d", result, next.calls)
	}
}

func TestExternalWorkRunProjectWorkDirGuardIgnoraOtrosProyectosV0(t *testing.T) {
	next := &fakeExternalWorkRunGuardNextV0{
		result: orquestamcp.MCPExternalWorkRunToolResultV0{Estado: orquestamcp.MCPExternalWorkRunEstadoOKV0},
	}
	executor := NewExternalWorkRunProjectWorkDirGuardExecutorV0(next, ExternalWorkRunProjectWorkDirGuardConfigV0{
		ProjectWorkDir: "/home/alberto/Trabajo/orquesta",
		Rules: []ExternalWorkRunProjectWorkDirGuardRuleV0{{
			ProjectRef:             "opes",
			RequiredProjectWorkDir: "/home/alberto/Trabajo/OPES",
		}},
	})

	result, err := executor.Execute(context.Background(), orquestamcp.MCPExternalWorkRunToolInputV0{
		ExternalWorkRunRequest: orquestaexternalworkrun.StartExternalWorkRunRequestV0{
			ProjectRef: "project-ref-other",
		},
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 || next.calls != 1 {
		t.Fatalf("result=%+v calls=%d", result, next.calls)
	}
}

type fakeExternalWorkRunGuardNextV0 struct {
	calls  int
	result orquestamcp.MCPExternalWorkRunToolResultV0
}

func (fake *fakeExternalWorkRunGuardNextV0) Execute(
	context.Context,
	orquestamcp.MCPExternalWorkRunToolInputV0,
) (orquestamcp.MCPExternalWorkRunToolResultV0, error) {
	fake.calls++
	return fake.result, nil
}
