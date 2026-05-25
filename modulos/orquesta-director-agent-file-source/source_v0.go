package orquestadirectoragentfilesource

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
)

var _ orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0 = DirectorAgentDecisionFileSourceV0{}

func (source DirectorAgentDecisionFileSourceV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := source.validateV0(); err != nil {
		return nil, err
	}
	runRef := strings.TrimSpace(request.Run.RunID)
	descriptors, err := source.DescriptorProvider.ListDirectorAgentDecisionFilesV0(
		ctx,
		DirectorAgentDecisionFileListRequestV0{
			RunID:          runRef,
			PhaseArtifacts: compactDirectorAgentFileSourceRefsV0(request.Run.PhaseArtifacts),
			Deliveries:     compactDirectorAgentFileSourceRefsV0(request.Run.Deliveries),
			CorrelationID:  strings.TrimSpace(request.CorrelationID),
			RequestedBy:    strings.TrimSpace(request.RequestedBy),
		},
	)
	if err != nil {
		return nil, err
	}
	return source.decisionsFromDescriptorsV0(ctx, runRef, descriptors)
}

func (source DirectorAgentDecisionFileSourceV0) validateV0() error {
	if source.DescriptorProvider == nil {
		return DirectorAgentFileSourceIssueV0{Field: "descriptor_provider"}
	}
	if source.Reader == nil {
		return DirectorAgentFileSourceIssueV0{Field: "reader"}
	}
	return nil
}

func (source DirectorAgentDecisionFileSourceV0) decisionsFromDescriptorsV0(
	ctx context.Context,
	runRef string,
	descriptors []DirectorAgentDecisionFileDescriptorV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	out := make([]orquestadirectoragent.DirectorAgentDecisionV0, 0, len(descriptors))
	for _, descriptor := range normalizeDirectorAgentDecisionFileDescriptorsV0(descriptors) {
		if descriptor.RunID != "" && descriptor.RunID != runRef {
			continue
		}
		decisions, err := source.decisionsFromDescriptorV0(ctx, descriptor)
		if err != nil {
			if source.IgnoreInvalidFiles && directorAgentDecisionFileSourceInvalidFileV0(err) {
				continue
			}
			return nil, err
		}
		out = append(out, filterDirectorAgentDecisionFileRunV0(runRef, decisions)...)
	}
	return out, nil
}

func directorAgentDecisionFileSourceInvalidFileV0(err error) bool {
	var issue DirectorAgentFileSourceIssueV0
	if !errors.As(err, &issue) {
		return false
	}
	switch strings.TrimSpace(issue.Field) {
	case "schema_version", "file", "file_size", "decisions", "decision":
		return true
	default:
		field := strings.TrimSpace(issue.Field)
		return strings.HasPrefix(field, "decision.") ||
			strings.HasPrefix(field, "sidecar_receipt.")
	}
}

func (source DirectorAgentDecisionFileSourceV0) decisionsFromDescriptorV0(
	ctx context.Context,
	descriptor DirectorAgentDecisionFileDescriptorV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if descriptor.Path == "" {
		return nil, DirectorAgentFileSourceIssueV0{Field: "descriptor.path"}
	}
	data, err := source.Reader.ReadDirectorAgentDecisionFileV0(
		ctx,
		descriptor.Path,
		source.maxBytesV0(),
	)
	if err != nil {
		return nil, err
	}
	if err := validateDirectorAgentDecisionSidecarReceiptV0(descriptor.SidecarReceipt, data); err != nil {
		return nil, err
	}
	decisions, err := DecodeDirectorAgentDecisionFileV0(data)
	if err != nil {
		return nil, err
	}
	decisions, err = validateDirectorAgentDecisionFileDecisionsV0(decisions)
	if err != nil {
		return nil, err
	}
	if err := source.recordDirectorAgentDecisionSidecarConsumedV0(ctx, descriptor.SidecarReceipt); err != nil {
		return nil, err
	}
	return decisions, nil
}

func validateDirectorAgentDecisionSidecarReceiptV0(
	receipt *DirectorAgentDecisionSidecarReceiptV0,
	data []byte,
) error {
	if receipt == nil {
		return nil
	}
	if strings.TrimSpace(receipt.SchemaVersion) != DirectorAgentDecisionSidecarReceiptSchemaV0 {
		return DirectorAgentFileSourceIssueV0{Field: "sidecar_receipt.schema_version"}
	}
	if strings.TrimSpace(receipt.ReceiptRef) == "" {
		return DirectorAgentFileSourceIssueV0{Field: "sidecar_receipt.receipt_ref"}
	}
	if strings.TrimSpace(receipt.SHA256) == "" {
		return DirectorAgentFileSourceIssueV0{Field: "sidecar_receipt.sha256"}
	}
	if strings.TrimSpace(receipt.SHA256) != directorAgentDecisionFileSHA256V0(data) {
		return DirectorAgentFileSourceIssueV0{Field: "sidecar_receipt.sha256_mismatch"}
	}
	if receipt.SizeBytes > 0 && receipt.SizeBytes != int64(len(data)) {
		return DirectorAgentFileSourceIssueV0{Field: "sidecar_receipt.size_mismatch"}
	}
	return nil
}

func (source DirectorAgentDecisionFileSourceV0) recordDirectorAgentDecisionSidecarConsumedV0(
	ctx context.Context,
	receipt *DirectorAgentDecisionSidecarReceiptV0,
) error {
	if source.ConsumptionRecorder == nil || receipt == nil {
		return nil
	}
	consumed := *receipt
	consumed.Status = "consumed"
	return source.ConsumptionRecorder.RecordDirectorAgentDecisionFileConsumptionV0(ctx, consumed)
}

func directorAgentDecisionFileSHA256V0(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func filterDirectorAgentDecisionFileRunV0(
	runRef string,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return decisions
	}
	out := make([]orquestadirectoragent.DirectorAgentDecisionV0, 0, len(decisions))
	for _, decision := range decisions {
		if strings.TrimSpace(decision.RunID) == runRef {
			out = append(out, decision)
		}
	}
	return out
}

func (source DirectorAgentDecisionFileSourceV0) maxBytesV0() int {
	if source.MaxBytes > 0 {
		return source.MaxBytes
	}
	return DefaultDirectorAgentDecisionFileMaxBytesV0
}

func compactDirectorAgentFileSourceRefsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
