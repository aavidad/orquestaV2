package orquestaruntimecodexdelivery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

const DefaultDirectorAgentDecisionFileNameV0 = "director_decisions.json"

type CodexReceiptDirectorDecisionFileDescriptorProviderV0 struct {
	Store    CodexReceiptDescriptorStorePortV0
	FileName string
}

var _ orquestadirectoragentfilesource.DirectorAgentDecisionFileDescriptorProviderPortV0 = CodexReceiptDirectorDecisionFileDescriptorProviderV0{}

type codexDirectorDecisionSidecarReceiptRecorderV0 interface {
	RecordDirectorAgentDecisionFileConsumptionV0(
		context.Context,
		orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0,
	) error
}

func (provider CodexReceiptDirectorDecisionFileDescriptorProviderV0) ListDirectorAgentDecisionFilesV0(
	ctx context.Context,
	request orquestadirectoragentfilesource.DirectorAgentDecisionFileListRequestV0,
) ([]orquestadirectoragentfilesource.DirectorAgentDecisionFileDescriptorV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if provider.Store == nil {
		return nil, fmt.Errorf("director_decision_file_descriptors: store requerido")
	}
	fileName := directorDecisionFileNameV0(provider.FileName)
	if fileName == "" {
		return nil, fmt.Errorf("director_decision_file_descriptors: file_name invalido")
	}
	runID := strings.TrimSpace(request.RunID)
	receipts, err := provider.Store.ListCodexReceiptDescriptorsV0(
		ctx,
		CodexReceiptDescriptorRequestV0{
			RunID:         runID,
			CorrelationID: strings.TrimSpace(request.CorrelationID),
		},
	)
	if err != nil {
		return nil, err
	}
	return directorDecisionDescriptorsFromReceiptsV0(ctx, provider.Store, receipts, runID, fileName, request)
}

func directorDecisionDescriptorsFromReceiptsV0(
	ctx context.Context,
	store CodexReceiptDescriptorStorePortV0,
	receipts []CodexReceiptDescriptorV0,
	runID string,
	fileName string,
	request orquestadirectoragentfilesource.DirectorAgentDecisionFileListRequestV0,
) ([]orquestadirectoragentfilesource.DirectorAgentDecisionFileDescriptorV0, error) {
	descriptors := make(
		[]orquestadirectoragentfilesource.DirectorAgentDecisionFileDescriptorV0,
		0,
		len(receipts),
	)
	for _, receipt := range receipts {
		receiptRunID := strings.TrimSpace(receipt.RunID)
		if runID != "" && receiptRunID != runID {
			continue
		}
		if !directorDecisionReceiptReflectedV0(receipt, request) {
			continue
		}
		ackPath := strings.TrimSpace(receipt.AckPath)
		if ackPath == "" {
			return nil, fmt.Errorf("director_decision_file_descriptors: ack_path requerido")
		}
		decisionPath := filepath.Join(filepath.Dir(ackPath), fileName)
		sidecar, exists, err := directorDecisionSidecarReceiptForFileV0(receipt, decisionPath, fileName, request)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		if sidecar.Status == "consumed" {
			continue
		}
		if err := recordDirectorDecisionSidecarPendingReceiptV0(ctx, store, sidecar); err != nil {
			return nil, err
		}
		descriptors = append(descriptors, orquestadirectoragentfilesource.DirectorAgentDecisionFileDescriptorV0{
			DescriptorRef:  directorDecisionDescriptorRefV0(receipt),
			RunID:          receiptRunID,
			Path:           decisionPath,
			SidecarReceipt: &sidecar,
		})
	}
	return descriptors, nil
}

func recordDirectorDecisionSidecarPendingReceiptV0(
	ctx context.Context,
	store CodexReceiptDescriptorStorePortV0,
	receipt orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0,
) error {
	recorder, ok := store.(codexDirectorDecisionSidecarReceiptRecorderV0)
	if !ok || recorder == nil {
		return nil
	}
	return recorder.RecordDirectorAgentDecisionFileConsumptionV0(ctx, receipt)
}

func directorDecisionReceiptReflectedV0(
	receipt CodexReceiptDescriptorV0,
	request orquestadirectoragentfilesource.DirectorAgentDecisionFileListRequestV0,
) bool {
	ackRef := strings.TrimSpace(receipt.Spec.AgentPacket.DeliveryRefs.AckRef)
	if ackRef == "" {
		return true
	}
	return stringInCodexDeliverySetV0(request.Deliveries, ackRef) ||
		codexReceiptArtifactRefRegisteredV0(request.PhaseArtifacts, ackRef)
}

func directorDecisionSidecarReceiptForFileV0(
	receipt CodexReceiptDescriptorV0,
	path string,
	fileName string,
	request orquestadirectoragentfilesource.DirectorAgentDecisionFileListRequestV0,
) (orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0, bool, error) {
	data, err := orquestaruntimecodex.ReadCodexControlFileBytesV0(path, fileName)
	if err != nil {
		reason, _ := orquestaruntimecodex.CodexControlFileReadErrorReasonV0(err)
		if reason == "not_found" {
			return orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0{}, false, nil
		}
		return orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0{}, false,
			fmt.Errorf("director_decision_file_descriptors: read decision_file: %s", reason)
	}
	hash := directorDecisionFileHashV0(data)
	sidecar := directorDecisionSidecarReceiptV0(receipt, request, hash, int64(len(data)))
	if existing := receipt.DirectorDecisionSidecarReceipt; existing != nil {
		if err := validateDirectorDecisionExistingSidecarReceiptV0(*existing, sidecar); err != nil {
			return orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0{}, false, err
		}
		sidecar = *existing
		sidecar.Status = strings.TrimSpace(sidecar.Status)
		if sidecar.Status == "" {
			sidecar.Status = "pending"
		}
	}
	return sidecar, true, nil
}

func directorDecisionFileHashV0(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func directorDecisionSidecarReceiptV0(
	receipt CodexReceiptDescriptorV0,
	request orquestadirectoragentfilesource.DirectorAgentDecisionFileListRequestV0,
	hash string,
	size int64,
) orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0 {
	receiptRef := directorDecisionDescriptorRefV0(receipt) + "-" + shortDirectorDecisionHashV0(hash)
	return orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0{
		SchemaVersion:         orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptSchemaV0,
		ReceiptRef:            receiptRef,
		ProducerDescriptorRef: strings.TrimSpace(receipt.DescriptorRef),
		ProducerAckRef:        strings.TrimSpace(receipt.Spec.AgentPacket.DeliveryRefs.AckRef),
		RunID:                 strings.TrimSpace(receipt.RunID),
		AgentRef:              codexReceiptDescriptorAgentRefV0(receipt),
		CorrelationID:         strings.TrimSpace(request.CorrelationID),
		SHA256:                hash,
		SizeBytes:             size,
		Status:                "pending",
	}
}

func validateDirectorDecisionExistingSidecarReceiptV0(
	existing orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0,
	current orquestadirectoragentfilesource.DirectorAgentDecisionSidecarReceiptV0,
) error {
	if strings.TrimSpace(existing.SchemaVersion) != current.SchemaVersion ||
		strings.TrimSpace(existing.ReceiptRef) != current.ReceiptRef ||
		strings.TrimSpace(existing.SHA256) != current.SHA256 ||
		existing.SizeBytes != current.SizeBytes ||
		strings.TrimSpace(existing.ProducerAckRef) != current.ProducerAckRef ||
		strings.TrimSpace(existing.RunID) != current.RunID ||
		strings.TrimSpace(existing.AgentRef) != current.AgentRef {
		return fmt.Errorf("director_decision_file_descriptors: sidecar_receipt_conflict")
	}
	return nil
}

func shortDirectorDecisionHashV0(hash string) string {
	hash = strings.TrimSpace(hash)
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}

func directorDecisionFileNameV0(fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		fileName = DefaultDirectorAgentDecisionFileNameV0
	}
	if fileName == "." || fileName == ".." {
		return ""
	}
	if filepath.Base(fileName) != fileName || strings.Contains(fileName, `\`) {
		return ""
	}
	return fileName
}

func directorDecisionDescriptorRefV0(receipt CodexReceiptDescriptorV0) string {
	ref := strings.TrimSpace(receipt.DescriptorRef)
	if ref == "" {
		ref = codexReceiptDescriptorRefV0(receipt.RunID, receipt.AgentRef)
	}
	return ref + "-director-decisions"
}
