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
	if next.calls != 0 {
		t.Fatalf("next ejecutado con workspace incorrecto: %d", next.calls)
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
