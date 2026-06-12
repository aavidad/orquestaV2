package orquestaruntimecodexdelivery

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func TestCodexReceiptRecordingSpecResolverV0RegistraDescriptor(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	ackPath := filepath.Join(t.TempDir(), "agent_ack.json")
	store := NewInMemoryCodexReceiptDescriptorStoreV0()
	resolver := CodexReceiptRecordingSpecResolverV0{
		Inner:           staticExternalAgentSpecResolverV0{Spec: spec},
		Recorder:        store,
		AckPathResolver: StaticCodexReceiptAckPathResolverV0{AckPath: ackPath},
	}

	_, err := resolver.ResolveExternalAgentLaunchSpecV0(context.Background(), codexReceiptInboundForTestV0(spec))
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}
	descriptors, err := store.ListCodexReceiptDescriptorsV0(context.Background(), CodexReceiptDescriptorRequestV0{
		RunID:         "run-ref-001",
		StartedAgents: []string{spec.RequestID},
	})
	if err != nil {
		t.Fatalf("ListCodexReceiptDescriptorsV0: %v", err)
	}
	if len(descriptors) != 1 {
		t.Fatalf("descriptors=%+v", descriptors)
	}
	got := descriptors[0]
	if got.AckPath != ackPath || got.AgentRef != spec.RequestID || got.RunID != "run-ref-001" {
		t.Fatalf("descriptor=%+v", got)
	}
}

func TestCodexReceiptRecordingSpecResolverV0PermiteSourcePosterior(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	path := filepath.Join(t.TempDir(), "agent_ack.json")
	writeCodexDeliveryAckAtPathForTestV0(t, path, codexDeliveryAckForTestV0(spec))
	store := NewInMemoryCodexReceiptDescriptorStoreV0()
	resolver := CodexReceiptRecordingSpecResolverV0{
		Inner:           staticExternalAgentSpecResolverV0{Spec: spec},
		Recorder:        store,
		AckPathResolver: StaticCodexReceiptAckPathResolverV0{AckPath: path},
	}
	if _, err := resolver.ResolveExternalAgentLaunchSpecV0(context.Background(), codexReceiptInboundForTestV0(spec)); err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}

	observations, err := (CodexDeliveryObservationSourceV0{Store: store}).
		BuildAgentDeliveryObservationsV0(context.Background(), codexDeliveryRequestForTestV0(spec, nil))
	if err != nil {
		t.Fatalf("BuildAgentDeliveryObservationsV0: %v", err)
	}
	if len(observations) != 1 || observations[0].DeliveryRef != spec.AgentPacket.DeliveryRefs.AckRef {
		t.Fatalf("observations=%+v", observations)
	}
}

func TestCodexReceiptRecordingSpecResolverV0RegistraBaselineWorktree(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("base"), 0o600); err != nil {
		t.Fatalf("write base: %v", err)
	}
	ackPath := filepath.Join(t.TempDir(), "agent_ack.json")
	receiptStore := NewInMemoryCodexReceiptDescriptorStoreV0()
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0()
	resolver := CodexReceiptRecordingSpecResolverV0{
		Inner:    staticExternalAgentSpecResolverV0{Spec: spec},
		Recorder: receiptStore,
		AckPathResolver: StaticCodexReceiptAckPathResolverV0{
			AckPath:        ackPath,
			ProjectWorkDir: projectDir,
		},
		WorktreeBaselineRecorder: CodexReceiptWorktreeBaselineRecorderV0{
			SnapshotStore: snapshotStore,
		},
	}

	_, err := resolver.ResolveExternalAgentLaunchSpecV0(context.Background(), codexReceiptInboundForTestV0(spec))
	if err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0: %v", err)
	}
	descriptors, err := receiptStore.ListCodexReceiptDescriptorsV0(context.Background(), CodexReceiptDescriptorRequestV0{
		RunID:         "run-ref-001",
		StartedAgents: []string{spec.RequestID},
	})
	if err != nil || len(descriptors) != 1 {
		t.Fatalf("descriptors=%+v err=%v", descriptors, err)
	}
	if descriptors[0].WorktreeBaselineRef == "" || descriptors[0].ProjectWorkDir != projectDir {
		t.Fatalf("descriptor sin baseline: %+v", descriptors[0])
	}
	snapshot, err := snapshotStore.LoadWorktreeSnapshotV0(context.Background(), descriptors[0].WorktreeBaselineRef)
	if err != nil || len(snapshot.Files) != 1 || snapshot.Files[0].Path != "README.md" {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
}

func TestCodexReceiptRecordingSpecResolverV0NoBloqueaSiBaselineNoCabe(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "binario-local"), []byte("base"), 0o600); err != nil {
		t.Fatalf("write base: %v", err)
	}
	ackPath := filepath.Join(t.TempDir(), "agent_ack.json")
	receiptStore := NewInMemoryCodexReceiptDescriptorStoreV0()
	snapshotStore := orquestaruntimeworktree.NewInMemoryWorktreeSnapshotStoreV0()
	resolver := CodexReceiptRecordingSpecResolverV0{
		Inner:    staticExternalAgentSpecResolverV0{Spec: spec},
		Recorder: receiptStore,
		AckPathResolver: StaticCodexReceiptAckPathResolverV0{
			AckPath:        ackPath,
			ProjectWorkDir: projectDir,
		},
		WorktreeBaselineRecorder: CodexReceiptWorktreeBaselineRecorderV0{
			SnapshotStore: snapshotStore,
			SnapshotReadBudget: orquestaruntimeworktree.WorktreeSnapshotReadBudgetV0{
				MaxFiles:      10,
				MaxFileBytes:  1,
				MaxTotalBytes: 1024,
			},
		},
	}

	if _, err := resolver.ResolveExternalAgentLaunchSpecV0(context.Background(), codexReceiptInboundForTestV0(spec)); err != nil {
		t.Fatalf("ResolveExternalAgentLaunchSpecV0 bloqueo por baseline: %v", err)
	}
	descriptors, err := receiptStore.ListCodexReceiptDescriptorsV0(context.Background(), CodexReceiptDescriptorRequestV0{
		RunID:         "run-ref-001",
		StartedAgents: []string{spec.RequestID},
	})
	if err != nil || len(descriptors) != 1 {
		t.Fatalf("descriptors=%+v err=%v", descriptors, err)
	}
	if descriptors[0].WorktreeBaselineRef == "" {
		t.Fatalf("baseline parcial no registrada: %+v", descriptors[0])
	}
	snapshot, err := snapshotStore.LoadWorktreeSnapshotV0(context.Background(), descriptors[0].WorktreeBaselineRef)
	if err != nil {
		t.Fatalf("LoadWorktreeSnapshotV0: %v", err)
	}
	if len(snapshot.Files) != 0 {
		t.Fatalf("snapshot=%+v", snapshot)
	}
	requireCodexDeliveryReceiptReasonV0(t, snapshot.ExclusionReceipts, orquestaruntimeworktree.WorktreeIssueSnapshotFileTooLargeV0)
}

func TestInMemoryCodexReceiptDescriptorStoreV0FiltraRunAgenteYDelivery(t *testing.T) {
	spec := codexDeliverySpecForTestV0()
	other := codexDeliverySpecForTestV0()
	other.RequestID = "agent-ref-002"
	other.AgentPacket.RequestID = other.RequestID
	other.AgentPacket.DeliveryRefs.AckRef = "ack-ref-002"
	store := NewInMemoryCodexReceiptDescriptorStoreV0(
		CodexReceiptDescriptorV0{
			DescriptorRef: "receipt-ref-001",
			RunID:         "run-ref-001",
			AgentRef:      spec.RequestID,
			Spec:          spec,
			AckPath:       "/tmp/ack-001.json",
		},
		CodexReceiptDescriptorV0{
			DescriptorRef: "receipt-ref-002",
			RunID:         "run-ref-001",
			AgentRef:      other.RequestID,
			Spec:          other,
			AckPath:       "/tmp/ack-002.json",
		},
	)

	descriptors, err := store.ListCodexReceiptDescriptorsV0(context.Background(), CodexReceiptDescriptorRequestV0{
		RunID:         "run-ref-001",
		StartedAgents: []string{spec.RequestID, other.RequestID},
		Deliveries:    []string{spec.AgentPacket.DeliveryRefs.AckRef},
	})
	if err != nil {
		t.Fatalf("ListCodexReceiptDescriptorsV0: %v", err)
	}
	if len(descriptors) != 1 || descriptors[0].AgentRef != other.RequestID {
		t.Fatalf("descriptors=%+v", descriptors)
	}
}

type staticExternalAgentSpecResolverV0 struct {
	Spec            orquestaruntime.ExternalAgentLaunchSpecV0
	CommandResolver orquestaruntime.ExternalAgentProcessCommandResolverV0
}

func (resolver staticExternalAgentSpecResolverV0) ResolveExternalAgentLaunchSpecV0(
	context.Context,
	orquestaruntime.AgentLauncherInboundV0,
) (orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0, error) {
	return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{
		Spec:            resolver.Spec,
		CommandResolver: resolver.CommandResolver,
	}, nil
}

func codexReceiptInboundForTestV0(
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) orquestaruntime.AgentLauncherInboundV0 {
	return orquestaruntime.AgentLauncherInboundV0{
		CorrelationID: "corr-receipt-recording-001",
		Payload: &orquestaruntime.LaunchRuntimeAgentRequestV0{
			RunID:          "run-ref-001",
			AgentRequestID: spec.RequestID,
		},
	}
}

func writeCodexDeliveryAckAtPathForTestV0(
	t *testing.T,
	path string,
	ack any,
) {
	t.Helper()
	data, err := json.Marshal(ack)
	if err != nil {
		t.Fatalf("marshal ack: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write ack: %v", err)
	}
}
