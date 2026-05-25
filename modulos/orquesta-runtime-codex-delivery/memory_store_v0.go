package orquestaruntimecodexdelivery

import (
	"context"
	"fmt"
	"strings"
	"sync"

	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
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

func (store *InMemoryCodexReceiptDescriptorStoreV0) RecordDirectorAgentDecisionFileConsumptionV0(
	_ context.Context,
	receipt orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0,
) error {
	receipt = normalizeDirectorDecisionSidecarReceiptV0(receipt)
	if err := validateDirectorDecisionSidecarReceiptForStoreV0(receipt); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	for index := range store.descriptors {
		if !codexReceiptDescriptorMatchesDecisionSidecarV0(store.descriptors[index], receipt) {
			continue
		}
		store.descriptors[index].DirectorDecisionSidecarReceipt = &receipt
		return nil
	}
	return fmt.Errorf("codex_receipt_descriptor: director_decision_sidecar_unknown")
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

func normalizeDirectorDecisionSidecarReceiptV0(
	receipt orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0,
) orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0 {
	receipt.SchemaVersion = strings.TrimSpace(receipt.SchemaVersion)
	receipt.ReceiptRef = strings.TrimSpace(receipt.ReceiptRef)
	receipt.ProducerDescriptorRef = strings.TrimSpace(receipt.ProducerDescriptorRef)
	receipt.ProducerAckRef = strings.TrimSpace(receipt.ProducerAckRef)
	receipt.RunID = strings.TrimSpace(receipt.RunID)
	receipt.AgentRef = strings.TrimSpace(receipt.AgentRef)
	receipt.CorrelationID = strings.TrimSpace(receipt.CorrelationID)
	receipt.SHA256 = strings.TrimSpace(receipt.SHA256)
	receipt.Status = strings.TrimSpace(receipt.Status)
	if receipt.Status == "" {
		receipt.Status = "pending"
	}
	return receipt
}

func validateDirectorDecisionSidecarReceiptForStoreV0(
	receipt orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0,
) error {
	switch {
	case receipt.SchemaVersion != orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptSchemaV0:
		return fmt.Errorf("codex_receipt_descriptor: director_decision_sidecar_schema_invalid")
	case receipt.ReceiptRef == "":
		return fmt.Errorf("codex_receipt_descriptor: director_decision_sidecar_receipt_ref_required")
	case receipt.SHA256 == "":
		return fmt.Errorf("codex_receipt_descriptor: director_decision_sidecar_hash_required")
	case receipt.Status != "pending" && receipt.Status != "consumed":
		return fmt.Errorf("codex_receipt_descriptor: director_decision_sidecar_status_invalid")
	default:
		return nil
	}
}

func codexReceiptDescriptorMatchesDecisionSidecarV0(
	descriptor CodexReceiptDescriptorV0,
	receipt orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0,
) bool {
	if receipt.ProducerDescriptorRef != "" &&
		strings.TrimSpace(descriptor.DescriptorRef) == receipt.ProducerDescriptorRef {
		return true
	}
	if descriptor.DirectorDecisionSidecarReceipt == nil {
		return false
	}
	return strings.TrimSpace(descriptor.DirectorDecisionSidecarReceipt.ReceiptRef) == receipt.ReceiptRef
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
