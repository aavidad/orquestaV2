package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestCodexStackAgentRuntimeDetailHTTPHandlerV0DevuelveEnvelopeRedactado(t *testing.T) {
	runRef := "run-ref-ops-runtime-detail-001"
	agentRef := "agent-ref-ops-runtime-detail-001"
	runtimeDir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentPromptFileNameV0),
		[]byte("Usa caveman lite y protocolo compacto de tokens. authorization: Bearer secreto"),
		0o600,
	); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		[]byte(`{"schema_version":"codex_agent_ack.v0","request_id":"agent-ref-ops-runtime-detail-001","ack_ref":"delivery-ref-ops-runtime-detail-001","task_ref":"task-ref-ops-runtime-detail-001","status":"completed","tests":["go test ./modulos/orquesta-web"]}`),
		0o600,
	); err != nil {
		t.Fatalf("write ack: %v", err)
	}
	packet := orquestaruntime.AgentStartPacketV0{
		SchemaVersion: orquestaruntime.AgentStartPacketSchemaVersionV0,
		RequestID:     agentRef,
		WorkOrderRef:  "work-order-ref-ops-runtime-detail-001",
		Phase:         "programacion",
		CapacityLevel: "high",
		Task: orquestaruntime.AgentStartTaskV0{
			TaskRef:        "task-ref-ops-runtime-detail-001",
			Title:          "Panel ops detalle runtime",
			Objective:      "Mostrar prompt, packet y ACK del agente.",
			RequiredTests:  []string{"go test ./modulos/orquesta-web"},
			MaxChildAgents: 6,
		},
		Policies: []string{"write_set_closed", "ack_required"},
	}
	store := fakeCodexOpsReceiptStoreV0{descriptors: []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{{
		DescriptorRef: "descriptor-ref-ops-runtime-detail-001",
		RunID:         runRef,
		AgentRef:      agentRef,
		AckPath:       filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		Spec: orquestaruntime.ExternalAgentLaunchSpecV0{
			RequestID:   agentRef,
			AgentPacket: packet,
		},
	}}}
	handler := NewCodexStackAgentRuntimeDetailHTTPHandlerV0(CodexStackAgentRuntimeDetailConfigV0{
		ReceiptStore:   store,
		RuntimeWorkDir: filepath.Dir(runtimeDir),
	})
	body, _ := json.Marshal(CodexStackAgentRuntimeDetailRequestV0{
		RequestID:   "ops-runtime-detail-test",
		RunRef:      runRef,
		AgentRef:    agentRef,
		IncludeLogs: true,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/ops/agent-runtime-detail", bytes.NewReader(body))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got CodexStackAgentRuntimeDetailResponseV0
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(got.Agents) != 1 {
		t.Fatalf("agents=%d body=%s", len(got.Agents), rec.Body.String())
	}
	agent := got.Agents[0]
	if !agent.Skills.CavemanRequested || !agent.Skills.CompactProtocol {
		t.Fatalf("skills=%+v", agent.Skills)
	}
	if agent.Task.ObjectiveSummary != "Mostrar prompt, packet y ACK del agente." ||
		agent.Task.RequiredTestsCount != 1 {
		t.Fatalf("agent detail incompleto: %+v", agent)
	}
	bodyText := rec.Body.String()
	for _, forbidden := range []string{"authorization: Bearer", runtimeDir, "prompt_text", `"agent_packet":`, `"ack":`} {
		if strings.Contains(bodyText, forbidden) {
			t.Fatalf("detalle runtime filtro contenido prohibido %q en %s", forbidden, bodyText)
		}
	}
	promptFile := codexStackAgentRuntimeDetailFileByNameForTestV0(
		agent.Files,
		orquestaruntimecodex.CodexAgentPromptFileNameV0,
	)
	if promptFile.RedactionLevel != "metadata_extracts" ||
		!codexStackOpsFileHasReasonForTestV0(promptFile, "runtime_detail_redacted") ||
		promptFile.ContentRef == "" ||
		promptFile.Preview != "" {
		t.Fatalf("prompt envelope inesperado: %+v", promptFile)
	}
	ackFile := codexStackAgentRuntimeDetailFileByNameForTestV0(
		agent.Files,
		orquestaruntimecodex.CodexAgentAckFileNameV0,
	)
	if ackFile.Extracts["status"] != "completed" || ackFile.Extracts["tests_count"] != float64(1) {
		t.Fatalf("ack extracts inesperados: %+v", ackFile.Extracts)
	}
}

func TestCodexStackAgentRuntimeDetailHTTPHandlerV0LeeLogsPorTailAcotado(t *testing.T) {
	runRef := "run-ref-ops-runtime-tail-001"
	agentRef := "agent-ref-ops-runtime-tail-001"
	runtimeDir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexStdoutFileNameV0),
		[]byte("prefijo-crudo\n"+strings.Repeat("padding\n", 64)+"tail-visible"),
		0o600,
	); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	store := fakeCodexOpsReceiptStoreV0{descriptors: []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{{
		RunID:    runRef,
		AgentRef: agentRef,
		AckPath:  filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentAckFileNameV0),
	}}}
	handler := NewCodexStackAgentRuntimeDetailHTTPHandlerV0(CodexStackAgentRuntimeDetailConfigV0{
		ReceiptStore: store,
	})
	body, _ := json.Marshal(CodexStackAgentRuntimeDetailRequestV0{
		RequestID:    "ops-runtime-detail-tail-test",
		RunRef:       runRef,
		AgentRef:     agentRef,
		IncludeLogs:  true,
		MaxFileBytes: 32,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/ops/agent-runtime-detail", bytes.NewReader(body))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got CodexStackAgentRuntimeDetailResponseV0
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("json: %v", err)
	}
	file := codexStackAgentRuntimeDetailFileByNameForTestV0(
		got.Agents[0].Files,
		orquestaruntimecodex.CodexStdoutFileNameV0,
	)
	if !strings.Contains(file.Preview, "tail-visible") || strings.Contains(file.Preview, "prefijo-crudo") {
		t.Fatalf("stdout tail inesperado: %+v", file)
	}
	if file.RedactionLevel != "redacted_tail" ||
		!codexStackOpsFileHasReasonForTestV0(file, "runtime_detail_redacted") {
		t.Fatalf("stdout envelope inesperado: %+v", file)
	}
}

func codexStackAgentRuntimeDetailFileByNameForTestV0(
	files []CodexStackAgentRuntimeDetailFileV0,
	name string,
) CodexStackAgentRuntimeDetailFileV0 {
	for _, file := range files {
		if file.Name == name {
			return file
		}
	}
	return CodexStackAgentRuntimeDetailFileV0{}
}

func codexStackOpsFileHasReasonForTestV0(file CodexStackAgentRuntimeDetailFileV0, reason string) bool {
	for _, got := range file.ReasonCodes {
		if got == reason {
			return true
		}
	}
	return false
}

type fakeCodexOpsReceiptStoreV0 struct {
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0
}

func (store fakeCodexOpsReceiptStoreV0) ListCodexReceiptDescriptorsV0(
	ctx context.Context,
	request orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0,
) ([]orquestaruntimecodexdelivery.CodexReceiptDescriptorV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0{}
	for _, descriptor := range store.descriptors {
		if request.RunID == "" || descriptor.RunID == request.RunID {
			out = append(out, descriptor)
		}
	}
	return out, nil
}
