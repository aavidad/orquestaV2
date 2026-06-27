package orquestaruntimecodexdelivery

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

type staticCodexReceiptStoreV0 struct {
	Descriptors []CodexReceiptDescriptorV0
	LastRequest CodexReceiptDescriptorRequestV0
}

type staticCodexReceiptWorktreeEvidenceVerifierV0 struct {
	Refs []string
}

func (verifier staticCodexReceiptWorktreeEvidenceVerifierV0) VerifyCodexReceiptWorktreeV0(
	context.Context,
	CodexReceiptWorktreeVerificationRequestV0,
) error {
	return nil
}

func (verifier staticCodexReceiptWorktreeEvidenceVerifierV0) VerifyCodexReceiptWorktreeEvidenceRefsV0(
	context.Context,
	CodexReceiptWorktreeVerificationRequestV0,
) ([]string, error) {
	return append([]string(nil), verifier.Refs...), nil
}

func (store *staticCodexReceiptStoreV0) ListCodexReceiptDescriptorsV0(
	_ context.Context,
	request CodexReceiptDescriptorRequestV0,
) ([]CodexReceiptDescriptorV0, error) {
	store.LastRequest = request
	return store.Descriptors, nil
}

func codexDeliveryRequestForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	deliveries []string,
) orquestacionnucleoapp.AgentDeliveryObservationRequestV0 {
	return orquestacionnucleoapp.AgentDeliveryObservationRequestV0{
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID:         "run-ref-001",
			Agents:        []string{spec.RequestID},
			StartedAgents: []string{spec.RequestID},
			Deliveries:    deliveries,
		},
		CorrelationID: "corr-receipt-source-001",
	}
}

func codexDeliverySpecForTestV0() orquestaruntime.ExternalAgentLaunchSpecV0 {
	return orquestaruntime.ExternalAgentLaunchSpecV0{
		RequestID:     "agent-ref-001",
		CorrelationID: "corr-agent-001",
		AgentPacket: orquestaruntime.AgentStartPacketV0{
			RequestID:     "agent-ref-001",
			CorrelationID: "corr-agent-001",
			TargetModule:  "agenda-app",
			Phase:         string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
			Task: orquestaruntime.AgentStartTaskV0{
				TaskRef:       "task-ref-001",
				WriteSet:      []string{"README.md"},
				RequiredTests: []string{"go test ./..."},
			},
			Context: orquestacontext.ContextMaterializedBundleV0{
				SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
				BundleRef:     "bundle-ref-001",
				WorkOrderRef:  "task-ref-001",
				TargetModule:  "agenda-app",
			},
			DeliveryRefs: orquestaruntime.AgentStartDeliveryRefsV0{
				MailboxRef:   "mailbox-ref-001",
				AckRef:       "ack-ref-001",
				ReadinessRef: "readiness-ref-001",
			},
		},
	}
}

func codexDeliverySpecWithRefsForTestV0(
	agentRef string,
	taskRef string,
	ackRef string,
) orquestaruntime.ExternalAgentLaunchSpecV0 {
	spec := codexDeliverySpecForTestV0()
	spec.RequestID = agentRef
	spec.CorrelationID = "corr-" + agentRef
	spec.AgentPacket.RequestID = agentRef
	spec.AgentPacket.CorrelationID = spec.CorrelationID
	spec.AgentPacket.Task.TaskRef = taskRef
	spec.AgentPacket.Context.WorkOrderRef = taskRef
	spec.AgentPacket.DeliveryRefs.AckRef = ackRef
	return spec
}

func codexDeliveryAckForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) orquestaruntimecodex.CodexAgentAckV0 {
	exitCode := 0
	outputRedacted := true
	return orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     spec.RequestID,
		CorrelationID: spec.CorrelationID,
		AckRef:        spec.AgentPacket.DeliveryRefs.AckRef,
		TargetModule:  spec.AgentPacket.TargetModule,
		TaskRef:       spec.AgentPacket.Task.TaskRef,
		Status:        "completed",
		Files:         orquestaruntimecodex.EvidenceListV0{"README.md"},
		Tests:         orquestaruntimecodex.EvidenceListV0{"go test ./..."},
		TestReceipts: []orquestaruntimecodex.CodexRequiredTestReceiptV0{{
			SchemaVersion:  orquestaruntimecodex.CodexRequiredTestReceiptSchemaVersionV0,
			Command:        "go test ./...",
			Status:         "passed",
			ExitCode:       &exitCode,
			EvidenceRefs:   []string{"required-test-receipt-ref-codex-delivery-test"},
			OccurredAt:     "2026-05-24T10:00:00Z",
			Sequence:       1,
			OutputRedacted: &outputRedacted,
		}},
		Notes: orquestaruntimecodex.EvidenceListV0{"contexto_ref_only_resuelto: fixture local sin contexto externo"},
	}
}

func writeCodexDeliveryAckForTestV0(
	t *testing.T,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
	ack orquestaruntimecodex.CodexAgentAckV0,
) string {
	t.Helper()
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	path := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write ack for %s: %v", spec.RequestID, err)
	}
	return path
}

func observationLeaksCodexDeliveryPathV0(
	observation orquestacionnucleoapp.AgentDeliveryObservationV0,
	path string,
) bool {
	values := append([]string{
		observation.CandidateRef,
		observation.DeliveryRef,
		observation.PhaseID,
		observation.TaskID,
		observation.AgentRef,
		observation.Summary,
	}, observation.EvidenceRefs...)
	for _, value := range values {
		if strings.Contains(value, path) {
			return true
		}
	}
	return false
}

func codexDeliveryObservationRefsForTestV0(
	observations []orquestacionnucleoapp.AgentDeliveryObservationV0,
	agentRef string,
) bool {
	for _, observation := range observations {
		if observation.AgentRef == agentRef {
			return true
		}
	}
	return false
}

func codexDeliveryObservationByAgentForTestV0(
	observations []orquestacionnucleoapp.AgentDeliveryObservationV0,
	agentRef string,
) orquestacionnucleoapp.AgentDeliveryObservationV0 {
	for _, observation := range observations {
		if observation.AgentRef == agentRef {
			return observation
		}
	}
	return orquestacionnucleoapp.AgentDeliveryObservationV0{}
}

func manyCodexDeliveryEvidenceRefsForTestV0() []string {
	refs := make([]string, 0, 40)
	for index := 0; index < 40; index++ {
		refs = append(refs, strings.Repeat("evidence-ref-worktree-soft-issue-", 12)+string(rune('a'+index%20)))
	}
	return refs
}

func codexDeliveryEvidenceContainsForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
