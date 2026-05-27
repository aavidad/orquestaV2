package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestBacklogTaskIDAllocationLeaseV0ReservaIDsEnOwnersYNoCuentaEscaneoV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogTaskIDDocV0(t, projectDir, `# Backlog

## T001 primera

Objetivo: una.

## Escaneo backlog 2026-05-27

Evidencia revisada.

## T003 tercera

Objetivo: tres.
`)
	result := planBacklogTaskIDLeaseForTestV0(t, projectDir, "capacity_free", 3)
	if len(result.Requests) != 2 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	for _, request := range result.Requests {
		if request.FailureKind == "backlog_scan" {
			t.Fatalf("scanner usado como relleno=%+v", result.Requests)
		}
		if !containsStringPrefixForTaskIDLeaseTestV0(request.ReservationRefs, "reservation-ref-backlog-task-id-") ||
			!containsStringForTaskIDLeaseTestV0(request.AcceptanceCriteria, "backlog_task_id_allocation_gap") ||
			!containsStringForTestV0(request.EvidenceRefs, "evidence-ref-autoprogramming-backlog-task-id-allocation-lease") {
			t.Fatalf("request=%+v", request)
		}
	}
	if !containsStringForTestV0(result.Requests[0].ContextRefs, "backlog_task_id_range:T001") ||
		!containsStringForTestV0(result.Requests[1].ContextRefs, "backlog_task_id_range:T003") {
		t.Fatalf("requests=%+v", result.Requests)
	}
}

func TestBacklogTaskIDAllocationLeaseV0ReservaIDEnScannerFallbackV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogTaskIDDocV0(t, projectDir, `# Backlog

## Escaneo backlog 2026-05-27

Evidencia revisada sin tarea ejecutable.
`)
	result := planBacklogTaskIDLeaseForTestV0(t, projectDir, "", 1)
	if len(result.Requests) != 1 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	request := result.Requests[0]
	if request.SuggestedArea != "backlog-scan" ||
		!containsStringPrefixForTaskIDLeaseTestV0(request.ContextRefs, "backlog_task_id_ref:task-id-ref-backlog-") ||
		!containsStringForTestV0(request.ContextRefs, "backlog_task_id_range:T001") ||
		!containsStringForTaskIDLeaseTestV0(request.AcceptanceCriteria, "backlog_task_number_reservation_required") {
		t.Fatalf("request=%+v", request)
	}
}

func TestBacklogTaskIDAllocationLeaseV0DetectaColisionDeIDHumanoV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogTaskIDDocV0(t, projectDir, `# Backlog

## T250 primera-frontera

Objetivo: una.

## Escaneo backlog 2026-05-27

## T250 segunda-frontera

	Objetivo: dos.
`)
	result := planBacklogTaskIDLeaseForTestV0(t, projectDir, "", 1)
	collision := backlogCollisionForTestV0(result.Collisions, "backlog_task_number_collision")
	if collision.Code == "" ||
		!strings.Contains(collision.Message, "task_id=T250") ||
		!strings.Contains(collision.Message, "reason=backlog_duplicate_task_id_ambiguous") ||
		!strings.Contains(collision.Message, "T250#01=task-instance-ref-backlog-") ||
		!strings.Contains(collision.Message, "T250#02=task-instance-ref-backlog-") ||
		!strings.Contains(collision.Message, "task-instance-ref-backlog-") ||
		!strings.Contains(collision.Message, ":hash:") ||
		strings.Contains(collision.Message, "Escaneo backlog") {
		t.Fatalf("collisions=%+v", result.Collisions)
	}
}

func TestBacklogTaskIDAliasIndexV0PropagaAliasYReasonEnRequestsV0(t *testing.T) {
	projectDir := t.TempDir()
	mustWriteBacklogTaskIDDocV0(t, projectDir, `# Backlog

## T250 primera-frontera

Objetivo: una.

Alcance:

- `+"`cmd/orquesta-server`"+`

## T250 segunda-frontera

Objetivo: dos.

Alcance:

- `+"`modulos/orquesta-server`"+`
`)
	result := planBacklogTaskIDLeaseForTestV0(t, projectDir, "", 2)
	if len(result.Requests) != 2 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	for _, request := range result.Requests {
		if !containsStringPrefixForTaskIDLeaseTestV0(request.ContextRefs, "backlog_task_entry_ref:task-instance-ref-backlog-") ||
			!containsStringForTaskIDLeaseTestV0(request.ContextRefs, "backlog_task_id_collision_reason:backlog_duplicate_task_id_ambiguous") ||
			!containsStringForTaskIDLeaseTestV0(request.AcceptanceCriteria, "backlog_duplicate_task_id_ambiguous") ||
			!containsStringForTestV0(request.EvidenceRefs, "evidence-ref-autoprogramming-backlog-task-id-alias-index") {
			t.Fatalf("request=%+v", request)
		}
	}
	if !containsStringForTestV0(result.Requests[0].ContextRefs, "backlog_task_alias:T250#01") ||
		!containsStringForTestV0(result.Requests[1].ContextRefs, "backlog_task_alias:T250#02") {
		t.Fatalf("requests=%+v", result.Requests)
	}
}

func mustWriteBacklogTaskIDDocV0(t *testing.T, projectDir string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
}

func planBacklogTaskIDLeaseForTestV0(
	t *testing.T,
	projectDir string,
	trigger string,
	maxRequests int,
) orquestaserver.IdleSelfImprovementPlanResultV0 {
	t.Helper()
	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: maxRequests,
			Trigger:     trigger,
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

func containsStringPrefixForTaskIDLeaseTestV0(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func containsStringForTaskIDLeaseTestV0(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
