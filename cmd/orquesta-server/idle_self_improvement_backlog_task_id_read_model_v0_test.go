package main

import (
	"strings"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestBacklogTaskIDReadModelV0BloqueaSolicitudAmbiguaPorNumeroHumanoV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogTaskIDDocV0(t, projectDir, `# Backlog

## T250 primera-frontera

Objetivo: una.

## T250 segunda-frontera

Objetivo: dos.
`)
	result := planBacklogTaskIDReadModelForTestV0(t, projectDir, []string{
		"request-ref-autoprogramming-backlog-t250",
	})
	if len(result.Requests) != 0 || result.Message != "backlog_duplicate_task_id_ambiguous" {
		t.Fatalf("result=%+v", result)
	}
	collision := backlogCollisionForTestV0(result.Collisions, "backlog_duplicate_task_id_ambiguous")
	if collision.TaskID != "T250" || len(collision.InstanceRefs) != 2 ||
		!strings.HasPrefix(collision.InstanceRefs[0], "task-instance-ref-backlog-") {
		t.Fatalf("collision=%+v", collision)
	}
}

func TestBacklogTaskIDReadModelV0ExponeInstanciaEnRequestConAliasV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogTaskIDDocV0(t, projectDir, `# Backlog

## T250 primera-frontera

Objetivo: una.

## T250 segunda-frontera

Objetivo: dos.
`)
	result := planBacklogTaskIDReadModelForTestV0(t, projectDir, []string{
		"request-ref-autoprogramming-backlog-t250-primera-frontera-live",
	})
	if len(result.Requests) == 0 {
		t.Fatalf("result=%+v", result)
	}
	if !containsStringPrefixForTaskIDReadModelTestV0(result.Requests[0].ContextRefs, "backlog_task_instance_ref:") ||
		!containsStringForTaskIDReadModelTestV0(result.Requests[0].AcceptanceCriteria, "backlog_duplicate_task_id_ambiguous") {
		t.Fatalf("request=%+v", result.Requests[0])
	}
}

func planBacklogTaskIDReadModelForTestV0(
	t *testing.T,
	projectDir string,
	knownRequestRefs []string,
) orquestaserver.IdleSelfImprovementPlanResultV0 {
	t.Helper()
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests:      3,
			KnownRequestRefs: knownRequestRefs,
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef:    "request-ref-base",
				CorrelationID: "corr-request-ref-base",
				ProjectRef:    "project-ref-orquesta",
				WriteSet:      []string{"cmd/orquesta-server"},
				RequiredTests: []string{"go test -count=1 ./cmd/orquesta-server"},
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	return result
}

func containsStringPrefixForTaskIDReadModelTestV0(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func containsStringForTaskIDReadModelTestV0(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
