package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestIdleSelfImprovementBacklogPlannerV0NoSaltaACKAmbiguoV0(t *testing.T) {
	cases := []struct {
		name      string
		writeAck  func(t *testing.T, dir string, runRef string, tests []string)
		evidence  string
		firstArea string
	}{
		{
			name: "sin packet",
			writeAck: func(t *testing.T, dir string, _ string, _ []string) {
				t.Helper()
				mustWriteFileForBacklogAckTestV0(t, filepath.Join(dir, "agent_ack.json"), `{"status":"completed"}`)
			},
			evidence:  idleSelfImprovementAckAmbiguousEvidenceRefV0,
			firstArea: "t08-runtime-neutral-e2e",
		},
		{
			name: "ack minimo hidratable",
			writeAck: func(t *testing.T, dir string, runRef string, tests []string) {
				t.Helper()
				writeBacklogPlannerCorrelatedACKForTestV0(t, dir, runRef, tests, `{"schema_version":"codex_agent_ack.v0","status":"completed"}`)
			},
			evidence:  idleSelfImprovementAckAmbiguousEvidenceRefV0,
			firstArea: "t08-runtime-neutral-e2e",
		},
		{
			name: "ack completo correlado",
			writeAck: func(t *testing.T, dir string, runRef string, tests []string) {
				t.Helper()
				writeBacklogPlannerCorrelatedACKForTestV0(t, dir, runRef, tests, "")
			},
			evidence:  idleSelfImprovementAckStructuredEvidenceRefV0,
			firstArea: "t09-sanitizer",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			projectDir, runRef, ackDir := writeBacklogPlannerDocAndRuntimeDirForTestV0(t)
			requiredTests := []string{"go test -count=1 ./cmd/orquesta-server"}
			tc.writeAck(t, ackDir, runRef, requiredTests)

			result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
				orquestaserverBacklogBasePlanForTestV0(requiredTests),
			)
			if err != nil {
				t.Fatalf("PlanV0: %v", err)
			}
			if len(result.Requests) == 0 || result.Requests[0].SuggestedArea != tc.firstArea {
				t.Fatalf("requests=%+v", result.Requests)
			}
			if !containsStringForTestV0(result.EvidenceRefs, tc.evidence) {
				t.Fatalf("evidence_refs=%+v want %s", result.EvidenceRefs, tc.evidence)
			}
		})
	}
}

func TestIdleSelfImprovementBacklogPlannerV0NoCierraACKConFotoDocumentalObsoletaV0(t *testing.T) {
	projectDir, runRef, ackDir := writeBacklogPlannerDocAndRuntimeDirForTestV0(t)
	requiredTests := []string{"go test -count=1 ./cmd/orquesta-server"}
	oldDocs := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).
		backlogScanDocumentsV0("t08-runtime-neutral-e2e", 1, []string{idleSelfImprovementBacklogDocRelV0})
	doneCriteria := []string{backlogScanDocTokenV0(oldDocs[0])}
	writeBacklogPlannerCorrelatedACKWithCriteriaForTestV0(t, ackDir, runRef, requiredTests, doneCriteria)

	changed := "## T08 runtime-neutral-e2e\n\nObjetivo: uno cambiado.\n\n## T09 sanitizer\n\nObjetivo: dos.\n"
	mustWriteFileForBacklogAckTestV0(t, filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), changed)

	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserverBacklogBasePlanForTestV0(requiredTests),
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	if len(result.Requests) == 0 || result.Requests[0].SuggestedArea != "t08-runtime-neutral-e2e" {
		t.Fatalf("requests=%+v", result.Requests)
	}
	if !containsStringForTestV0(result.EvidenceRefs, "evidence-ref-autoprogramming-backlog-doc-merge-pending") {
		t.Fatalf("evidence_refs=%+v", result.EvidenceRefs)
	}
	if len(result.Collisions) == 0 || result.Collisions[0].Code != "backlog_docs_changed_after_plan" {
		t.Fatalf("collisions=%+v", result.Collisions)
	}
}

func writeBacklogPlannerDocAndRuntimeDirForTestV0(t *testing.T) (string, string, string) {
	t.Helper()
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "## T08 runtime-neutral-e2e\n\nObjetivo: uno.\n\n## T09 sanitizer\n\nObjetivo: dos.\n"
	mustWriteFileForBacklogAckTestV0(t, filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), content)
	runRef := backlogRequestRefForTestV0(content, 0) + "-retry-deadbeef"
	ackDir := filepath.Join(projectDir, ".orquesta-runtime", runRef, "agent-ref-t08")
	if err := os.MkdirAll(ackDir, 0o700); err != nil {
		t.Fatalf("mkdir ack: %v", err)
	}
	return projectDir, runRef, ackDir
}

func writeBacklogPlannerCorrelatedACKForTestV0(
	t *testing.T,
	ackDir string,
	runRef string,
	requiredTests []string,
	ackOverride string,
) {
	t.Helper()
	writeBacklogPlannerCorrelatedACKWithCriteriaForTestV0(t, ackDir, runRef, requiredTests, nil)
	if ackOverride == "" {
		return
	}
	mustWriteFileForBacklogAckTestV0(t, filepath.Join(ackDir, "agent_ack.json"), ackOverride)
}

func writeBacklogPlannerCorrelatedACKWithCriteriaForTestV0(
	t *testing.T,
	ackDir string,
	runRef string,
	requiredTests []string,
	doneCriteria []string,
) {
	t.Helper()
	normalizedRunRef := idleSelfImprovementNormalizeQueuedRequestRefV0(runRef)
	packet := fmt.Sprintf(`{"schema_version":"agent_start_packet.v0","request_id":"agent-ref-t08","correlation_id":"corr-%s-burst-001","target_module":"orquesta-app-stack-programacion","task":{"task_ref":"task-t08","write_set":["cmd/orquesta-server"],"required_tests":%s,"done_criteria":%s},"delivery_refs":{"ack_ref":"ack-ref-t08"}}`, normalizedRunRef, mustJSONArrayForBacklogAckTestV0(t, requiredTests), mustJSONArrayForBacklogAckTestV0(t, doneCriteria))
	mustWriteFileForBacklogAckTestV0(t, filepath.Join(ackDir, "agent_packet.json"), packet)
	ack := fmt.Sprintf(`{"schema_version":"codex_agent_ack.v0","request_id":"agent-ref-t08","correlation_id":"corr-%s-burst-001","ack_ref":"ack-ref-t08","target_module":"orquesta-app-stack-programacion","task_ref":"task-t08","status":"completed","files":["cmd/orquesta-server/idle_self_improvement_backlog_planner_v0.go"],"tests":%s,"test_receipts":%s}`, normalizedRunRef, mustJSONArrayForBacklogAckTestV0(t, requiredTests), mustTestReceiptsJSONForBacklogAckTestV0(t, requiredTests))
	mustWriteFileForBacklogAckTestV0(t, filepath.Join(ackDir, "agent_ack.json"), ack)
}

func orquestaserverBacklogBasePlanForTestV0(
	requiredTests []string,
) orquestaserver.IdleSelfImprovementPlanRequestV0 {
	return orquestaserver.IdleSelfImprovementPlanRequestV0{
		MaxRequests: 2,
		BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
			RequestRef:    "request-ref-base",
			CorrelationID: "corr-request-ref-base",
			ProjectRef:    "project-ref-orquesta",
			WriteSet:      []string{"cmd/orquesta-server"},
			RequiredTests: append([]string(nil), requiredTests...),
		},
	}
}

func mustJSONArrayForBacklogAckTestV0(t *testing.T, values []string) string {
	t.Helper()
	out := "["
	for index, value := range values {
		if index > 0 {
			out += ","
		}
		out += fmt.Sprintf("%q", value)
	}
	return out + "]"
}

func mustTestReceiptsJSONForBacklogAckTestV0(t *testing.T, values []string) string {
	t.Helper()
	out := "["
	for index, value := range values {
		if index > 0 {
			out += ","
		}
		out += fmt.Sprintf(`{"schema_version":"codex_required_test_receipt.v0","command":%q,"status":"passed","exit_code":0,"evidence_refs":["required-test-receipt-ref-%d"],"occurred_at":"2026-05-24T10:00:00Z","sequence":%d,"output_redacted":true}`, value, index+1, index+1)
	}
	return out + "]"
}

func mustWriteFileForBacklogAckTestV0(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
