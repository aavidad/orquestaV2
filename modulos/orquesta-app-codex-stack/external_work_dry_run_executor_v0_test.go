package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestCodexStackExternalWorkDryRunExecutorV0UsaModeloPorDefectoYConfigV0(t *testing.T) {
	executor := NewCodexStackExternalWorkDryRunExecutorV0(
		orquestamcp.MCPExternalWorkDryRunToolExecutorV0{
			Config: orquestaexternalworkrun.StartExternalWorkRunConfigV0{
				OccurredAt:  "2026-06-27T10:00:00Z",
				RequestedBy: "test",
			},
		},
		"codex-model-from-capacity",
	)

	result, err := executor.Execute(context.Background(), validExternalWorkDryRunInputCodexStackTestV0())

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 ||
		result.EstModel != "codex-model-from-capacity" ||
		result.Spec.GoalRef == "" ||
		result.Spec.DirectorKind != "codex_goal" {
		t.Fatalf("result=%+v", result)
	}
}

func TestExternalWorkDryRunProjectWorkDirGuardBloqueaComoRunV0(t *testing.T) {
	next := &fakeExternalWorkDryRunGuardNextV0{}
	executor := NewExternalWorkDryRunProjectWorkDirGuardExecutorV0(next, ExternalWorkRunProjectWorkDirGuardConfigV0{
		ProjectWorkDir: "/home/alberto/Trabajo/orquesta",
		Rules: []ExternalWorkRunProjectWorkDirGuardRuleV0{{
			ProjectRef:             "opes",
			RequiredProjectWorkDir: "/home/alberto/Trabajo/OPES",
			EvidenceRef:            "evidence-ref-opes-project-workdir",
		}},
	})

	result, err := executor.Execute(context.Background(), validExternalWorkDryRunInputCodexStackTestV0())

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != ExternalWorkRunProjectWorkDirMismatchV0 {
		t.Fatalf("result=%+v", result)
	}
	if next.calls != 0 {
		t.Fatalf("next ejecutado con workspace incorrecto: %d", next.calls)
	}
}

func validExternalWorkDryRunInputCodexStackTestV0() orquestamcp.MCPExternalWorkDryRunToolInputV0 {
	return orquestamcp.MCPExternalWorkDryRunToolInputV0{
		MCPExternalWorkRunToolInputV0: orquestamcp.MCPExternalWorkRunToolInputV0{
			RequestID:     "req-codex-stack-dry-run-001",
			CorrelationID: "corr-codex-stack-dry-run-001",
			AppChangeRequest: orquestaappchange.AppChangeRequestV0{
				ChangeRef:       "change-ref-codex-stack-dry-run-001",
				AppRef:          "opes",
				UserIntent:      "Preparar entrega OPES sin lanzar agentes.",
				AllowedWriteSet: []string{"deliveries/opes/job-ref-codex-stack-dry-run"},
				RequiredTests:   []string{"validar contrato externo de dominio"},
				ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
					ProjectRef: "opes",
					JobRef:     "job-ref-codex-stack-dry-run",
					WorkKind:   "draft_content_block",
					InputFields: []orquestadomainwork.DomainWorkFieldV0{{
						Name:  "topic_ref",
						Value: "topic-ref-001",
					}},
				},
			},
		},
	}
}

type fakeExternalWorkDryRunGuardNextV0 struct {
	calls int
}

func (fake *fakeExternalWorkDryRunGuardNextV0) Execute(
	context.Context,
	orquestamcp.MCPExternalWorkDryRunToolInputV0,
) (orquestamcp.MCPExternalWorkDryRunToolResultV0, error) {
	fake.calls++
	return orquestamcp.MCPExternalWorkDryRunToolResultV0{Estado: orquestamcp.MCPExternalWorkRunEstadoOKV0}, nil
}
