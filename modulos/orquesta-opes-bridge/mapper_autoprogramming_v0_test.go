package orquestaopesbridge

import (
	"strings"
	"testing"

	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

func TestBuildExternalWorkRunRequestV0PreservaRefsOpacasAutoprogramacion(t *testing.T) {
	worktreeRef := "worktree-ref-run-autoprog2-t08-opes-e2e-consumidor"
	branchRef := "branch-ref-run-autoprog2-t08-opes-e2e-consumidor"
	req, ok := BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
		ID:   "job-ref-autoprog-opes-001",
		Type: "assemble_topic",
		PayloadJSON: `{
			"topic_id":"topic-ref-operadores-001",
			"document_plan_artifact_id":"artifact-plan-operadores-001"
		}`,
		ExternalRefs: map[string]string{
			"worktree_ref": worktreeRef,
			"branch_ref":   branchRef,
		},
	}, JobRunConfigV0{})

	if !ok {
		t.Fatalf("request no construida")
	}
	work := req.AppChangeRequest.ExternalWork
	if work == nil ||
		!fieldValueForTestV0(work.InputFields, "worktree_ref", worktreeRef) ||
		!fieldValueForTestV0(work.InputFields, "branch_ref", branchRef) ||
		!containsStringForTestV0(work.WorkRefs, worktreeRef) ||
		!containsStringForTestV0(work.WorkRefs, branchRef) {
		t.Fatalf("work=%+v", work)
	}
	for _, value := range []string{
		req.AppChangeRequest.AllowedWriteSet[0],
		req.AppChangeRequest.ChangeRef,
		req.AppChangeRequest.RequestID,
	} {
		if strings.Contains(value, worktreeRef) || strings.Contains(value, branchRef) {
			t.Fatalf("ref opaca usada como ruta/nombre derivado: %q", value)
		}
	}
}

func TestBuildExternalWorkRunRequestV0RechazaRefsOpacasAutoprogramacionComoRuta(t *testing.T) {
	_, ok := BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
		ID:          "job-ref-autoprog-invalid-001",
		Type:        "assemble_topic",
		PayloadJSON: `{}`,
		ExternalRefs: map[string]string{
			"worktree_ref": "tmp/worktree-real",
			"branch_ref":   "branch-ref-valid",
		},
	}, JobRunConfigV0{})

	if ok {
		t.Fatalf("request construida con worktree_ref tipo ruta")
	}
}
