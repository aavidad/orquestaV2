package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestSelfAuditBacklogSectionsV0StaticcheckFindingEmiteSeccionEstableV0(t *testing.T) {
	restore := replaceSelfAuditRunnerForTestV0(func(_ context.Context, _ string, command selfAuditCommandV0) selfAuditCommandResultV0 {
		if command.ToolRef != "staticcheck" {
			return selfAuditCommandResultV0{}
		}
		return selfAuditCommandResultV0{
			Output: "modulos/orquesta-server/foo_v0.go:10:2: this value of err is never used (SA4006)\n",
		}
	})
	defer restore()

	first := selfAuditBacklogSectionsV0(context.Background(), t.TempDir())
	second := selfAuditBacklogSectionsV0(context.Background(), t.TempDir())

	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("sections first=%+v second=%+v", first, second)
	}
	section := first[0]
	if section.Ref == "" ||
		section.Ref != second[0].Ref ||
		section.SourceKind != "self_audit" ||
		section.Scope[0] != "modulos/orquesta-server/foo_v0.go" ||
		section.Tests[0] != "staticcheck ./..." ||
		!strings.Contains(section.Criteria[0], "staticcheck no reporta el hallazgo SA4006") ||
		!containsStringForTestV0(section.Inputs, "self_audit_tool:staticcheck") {
		t.Fatalf("section=%+v second=%+v", section, second[0])
	}
}

func TestIdleSelfImprovementBacklogPlannerV0IncluyeSelfAuditSoloConFlagV0(t *testing.T) {
	restore := replaceSelfAuditRunnerForTestV0(func(_ context.Context, _ string, command selfAuditCommandV0) selfAuditCommandResultV0 {
		if command.ToolRef != "staticcheck" {
			return selfAuditCommandResultV0{}
		}
		return selfAuditCommandResultV0{
			Output: "cmd/orquesta-server/self_audit_fixture_v0.go:12:3: should replace loop with copy (S1011)\n",
		}
	})
	defer restore()

	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0),
		[]byte("# Backlog\n\nSin secciones Txx.\n"),
		0o600,
	); err != nil {
		t.Fatalf("write backlog: %v", err)
	}

	disabled, err := (idleSelfImprovementBacklogPlannerV0{
		ProjectWorkDir: projectDir,
	}).loadBacklogSectionsV0()
	if err != nil {
		t.Fatalf("loadBacklogSectionsV0 disabled: %v", err)
	}
	if len(disabled) != 0 {
		t.Fatalf("self-audit no debe cargarse sin flag: %+v", disabled)
	}

	result, err := (idleSelfImprovementBacklogPlannerV0{
		ProjectWorkDir:          projectDir,
		SelfAuditBacklogEnabled: true,
	}).PlanV0(orquestaserver.IdleSelfImprovementPlanRequestV0{
		MaxRequests: 1,
		BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
			RequestRef:    "request-ref-base",
			CorrelationID: "corr-request-ref-base",
			ProjectRef:    "project-ref-orquesta",
		},
	})
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	if len(result.Requests) != 1 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	request := result.Requests[0]
	if request.FailureKind != "backlog_autoprogramming" ||
		!strings.HasPrefix(request.SuggestedArea, "self-audit-") ||
		!containsStringForTestV0(request.WriteSet, "cmd/orquesta-server/self_audit_fixture_v0.go") ||
		!containsStringForTestV0(request.RequiredTests, "staticcheck ./...") ||
		!containsStringForTestV0(request.ContextRefs, "backlog_doc:self_audit://staticcheck") {
		t.Fatalf("request=%+v", request)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0DeduplicaSelfAuditConKnownRefV0(t *testing.T) {
	restore := replaceSelfAuditRunnerForTestV0(func(_ context.Context, _ string, command selfAuditCommandV0) selfAuditCommandResultV0 {
		if command.ToolRef != "staticcheck" {
			return selfAuditCommandResultV0{}
		}
		return selfAuditCommandResultV0{
			Output: "cmd/orquesta-server/self_audit_fixture_v0.go:12:3: should replace loop with copy (S1011)\n",
		}
	})
	defer restore()

	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0),
		[]byte("# Backlog\n\nSin secciones Txx.\n"),
		0o600,
	); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	planner := idleSelfImprovementBacklogPlannerV0{
		ProjectWorkDir:          projectDir,
		SelfAuditBacklogEnabled: true,
	}
	base := orquestaserver.IdleSelfImprovementRequestV0{
		RequestRef: "request-ref-base",
		ProjectRef: "project-ref-orquesta",
	}
	first, err := planner.PlanV0(orquestaserver.IdleSelfImprovementPlanRequestV0{
		MaxRequests: 1,
		BaseRequest: base,
	})
	if err != nil || len(first.Requests) != 1 {
		t.Fatalf("first result=%+v err=%v", first, err)
	}
	second, err := planner.PlanV0(orquestaserver.IdleSelfImprovementPlanRequestV0{
		MaxRequests:      1,
		BaseRequest:      base,
		KnownRequestRefs: []string{first.Requests[0].RequestRef},
		KnownRunRefs:     []string{first.Requests[0].RequestRef + "-retry-001"},
	})
	if err != nil {
		t.Fatalf("second PlanV0: %v", err)
	}
	if len(second.Requests) != 0 || second.Message != "backlog_tareas_ya_visibles_en_cola" {
		t.Fatalf("second=%+v", second)
	}
}

func replaceSelfAuditRunnerForTestV0(
	runner func(context.Context, string, selfAuditCommandV0) selfAuditCommandResultV0,
) func() {
	previous := runSelfAuditCommandV0
	runSelfAuditCommandV0 = runner
	return func() { runSelfAuditCommandV0 = previous }
}
