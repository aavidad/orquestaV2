package orquestaruntimecodexdelivery

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type InMemoryCodexReceiptDescriptorStoreV0 struct {
	mu          sync.Mutex
	descriptors []CodexReceiptDescriptorV0
}

func NewInMemoryCodexReceiptDescriptorStoreV0(
	descriptors ...CodexReceiptDescriptorV0,
) *InMemoryCodexReceiptDescriptorStoreV0 {
	store := &InMemoryCodexReceiptDescriptorStoreV0{}
	for _, descriptor := range descriptors {
		_ = store.RecordCodexReceiptDescriptorV0(context.Background(), descriptor)
	}
	return store
}

func (store *InMemoryCodexReceiptDescriptorStoreV0) RecordCodexReceiptDescriptorV0(
	_ context.Context,
	descriptor CodexReceiptDescriptorV0,
) error {
	descriptor = normalizeCodexReceiptDescriptorV0(descriptor)
	if err := validateCodexReceiptDescriptorV0(descriptor); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	for index := range store.descriptors {
		if store.descriptors[index].DescriptorRef == descriptor.DescriptorRef {
			store.descriptors[index] = descriptor
			return nil
		}
	}
	store.descriptors = append(store.descriptors, descriptor)
	return nil
}

func (store *InMemoryCodexReceiptDescriptorStoreV0) ListCodexReceiptDescriptorsV0(
	_ context.Context,
	request CodexReceiptDescriptorRequestV0,
) ([]CodexReceiptDescriptorV0, error) {
	request = normalizeCodexReceiptDescriptorRequestV0(request)
	store.mu.Lock()
	defer store.mu.Unlock()
	result := make([]CodexReceiptDescriptorV0, 0, len(store.descriptors))
	for _, descriptor := range store.descriptors {
		if !codexReceiptDescriptorMatchesRequestV0(descriptor, request) {
			continue
		}
		result = append(result, descriptor)
	}
	return result, nil
}

func normalizeCodexReceiptDescriptorV0(
	descriptor CodexReceiptDescriptorV0,
) CodexReceiptDescriptorV0 {
	descriptor.DescriptorRef = strings.TrimSpace(descriptor.DescriptorRef)
	descriptor.RunID = strings.TrimSpace(descriptor.RunID)
	descriptor.AgentRef = strings.TrimSpace(descriptor.AgentRef)
	descriptor.AckPath = strings.TrimSpace(descriptor.AckPath)
	if descriptor.AgentRef == "" {
		descriptor.AgentRef = strings.TrimSpace(descriptor.Spec.RequestID)
	}
	if descriptor.DescriptorRef == "" {
		descriptor.DescriptorRef = "codex-receipt-ref-" + descriptor.AgentRef
	}
	return descriptor
}

func normalizeCodexReceiptDescriptorRequestV0(
	request CodexReceiptDescriptorRequestV0,
) CodexReceiptDescriptorRequestV0 {
	request.RunID = strings.TrimSpace(request.RunID)
	request.StartedAgents = compactCodexDeliveryRefsV0(request.StartedAgents)
	request.Deliveries = compactCodexDeliveryRefsV0(request.Deliveries)
	request.PhaseArtifacts = compactCodexDeliveryRefsV0(request.PhaseArtifacts)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.EvidenceRefs = compactCodexDeliveryRefsV0(request.EvidenceRefs)
	return request
}

func validateCodexReceiptDescriptorV0(descriptor CodexReceiptDescriptorV0) error {
	switch {
	case descriptor.DescriptorRef == "":
		return fmt.Errorf("codex_receipt_descriptor: descriptor_ref requerido")
	case descriptor.AgentRef == "":
		return fmt.Errorf("codex_receipt_descriptor: agent_ref requerido")
	case descriptor.AckPath == "":
		return fmt.Errorf("codex_receipt_descriptor: ack_path requerido")
	case strings.TrimSpace(descriptor.Spec.RequestID) == "":
		return fmt.Errorf("codex_receipt_descriptor: spec.request_id requerido")
	default:
		return nil
	}
}

func codexReceiptDescriptorMatchesRequestV0(
	descriptor CodexReceiptDescriptorV0,
	request CodexReceiptDescriptorRequestV0,
) bool {
	if request.RunID != "" && descriptor.RunID != "" && descriptor.RunID != request.RunID {
		return false
	}
	agentRef := strings.TrimSpace(descriptor.AgentRef)
	if len(request.StartedAgents) > 0 && !stringInCodexDeliverySetV0(request.StartedAgents, agentRef) {
		return false
	}
	ackRef := strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef)
	if stringInCodexDeliverySetV0(request.Deliveries, ackRef) {
		return false
	}
	return !codexReceiptArtifactRefRegisteredV0(request.PhaseArtifacts, ackRef)
}
