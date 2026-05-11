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
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestCodexDeliveryObservationSourceV0ConstruyeObservacionNeutral(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	path := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-001",
			Spec:          spec,
			AckPath:       path,
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations=%d", len(observations))
	}
	got := observations[0]
	if got.DeliveryRef != spec.AgentPacket.DeliveryRefs.AckRef ||
		got.ArtifactRef != spec.AgentPacket.DeliveryRefs.AckRef ||
		got.AgentRef != spec.RequestID ||
		got.TaskID != spec.AgentPacket.Task.TaskRef {
		t.Fatalf("observacion no correlada: %+v", got)
	}
	if observationLeaksCodexDeliveryPathV0(got, path) {
		t.Fatalf("observacion filtra path: %+v", got)
	}
	if store.LastRequest.RunID != "run-ref-001" ||
		len(store.LastRequest.StartedAgents) != 1 {
		t.Fatalf("request store=%+v", store.LastRequest)
	}
}

func TestCodexDeliveryObservationSourceV0OmiteDeliveryYaRegistrada(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	path := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-001",
			Spec:          spec,
			AckPath:       path,
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(
			context.Background(),
			codexDeliveryRequestForTestV0(spec, []string{spec.AgentPacket.DeliveryRefs.AckRef}),
		)
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 0 {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestCodexDeliveryObservationSourceV0OmiteAgenteCerrado(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	path := writeCodexDeliveryAckForTestV0(t, spec, codexDeliveryAckForTestV0(spec))
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-001",
			Spec:          spec,
			AckPath:       path,
		}},
	}
	request := codexDeliveryRequestForTestV0(spec, nil)
	request.Run.StoppedAgents = []string{spec.RequestID}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 0 {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestCodexDeliveryObservationSourceV0OmiteACKNoListoSinError(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	path := filepath.Join(t.TempDir(), orquestaruntimecodex.CodexAgentAckFileNameV0)
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-001",
			Spec:          spec,
			AckPath:       path,
		}},
	}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 0 {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestCodexDeliveryObservationSourceV0PropagaACKInvalidoSinPath(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	dir := t.TempDir()
	path := filepath.Join(dir, orquestaruntimecodex.CodexAgentAckFileNameV0)
	if err := os.WriteFile(path, []byte(`{"schema_version":"codex_agent_ack.v0"}`), 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}
	store := &staticCodexReceiptStoreV0{
		Descriptors: []CodexReceiptDescriptorV0{{
			DescriptorRef: "receipt-ref-001",
			Spec:          spec,
			AckPath:       path,
		}},
	}

	_, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))
	if err == nil {
		t.Fatalf("esperaba error")
	}
	if strings.Contains(err.Error(), dir) || strings.Contains(err.Error(), path) {
		t.Fatalf("error filtra path: %v", err)
	}
}

type staticCodexReceiptStoreV0 struct {
	Descriptors []CodexReceiptDescriptorV0
	LastRequest CodexReceiptDescriptorRequestV0
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

func codexDeliveryAckForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) orquestaruntimecodex.CodexAgentAckV0 {
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
