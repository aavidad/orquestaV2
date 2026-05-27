package orquestaruntimecodexdelivery

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func codexReceiptDescriptorRequestFromProgressV0(
	request orquestacionnucleoapp.AgentProgressObservationRequestV0,
) CodexReceiptDescriptorRequestV0 {
	return CodexReceiptDescriptorRequestV0{
		RunID:          strings.TrimSpace(request.Run.RunID),
		StartedAgents:  compactCodexDeliveryRefsV0(request.Run.StartedAgents),
		Deliveries:     compactCodexDeliveryRefsV0(request.Run.Deliveries),
		PhaseArtifacts: compactCodexDeliveryRefsV0(request.Run.PhaseArtifacts),
		CorrelationID:  strings.TrimSpace(request.CorrelationID),
		EvidenceRefs:   compactCodexDeliveryRefsV0(request.EvidenceRefs),
	}
}

func codexProgressAckReadyV0(
	descriptor CodexReceiptDescriptorV0,
) (bool, error) {
	_, ready, err := (CodexDeliveryObservationSourceV0{}).observationFromDescriptorV0(context.Background(), descriptor)
	return ready, err
}

func codexProgressSignatureV0(
	descriptor CodexReceiptDescriptorV0,
) string {
	dir := filepath.Dir(strings.TrimSpace(descriptor.AckPath))
	files := []string{
		orquestaruntimecodex.CodexStdoutFileNameV0,
		orquestaruntimecodex.CodexStderrFileNameV0,
		orquestaruntimecodex.CodexLastMessageFileNameV0,
		orquestaruntimecodex.CodexAgentAckFileNameV0,
	}
	var builder strings.Builder
	for _, name := range files {
		path := filepath.Join(dir, name)
		info, ok := codexProgressSafeLogInfoV0(path)
		if !ok {
			builder.WriteString(name)
			builder.WriteString(":missing;")
			continue
		}
		builder.WriteString(name)
		builder.WriteString(":")
		builder.WriteString(strconv.FormatInt(info.Size(), 10))
		builder.WriteString(":")
		builder.WriteString(strconv.FormatInt(info.ModTime().UnixNano(), 10))
		builder.WriteString(";")
	}
	return builder.String()
}
