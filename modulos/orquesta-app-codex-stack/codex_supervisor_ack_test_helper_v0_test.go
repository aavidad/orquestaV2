package orquestaappcodexstack

import (
	"encoding/json"
	"os"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func writeCodexStackCompletedAckForDescriptorV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) error {
	packet := descriptor.Spec.AgentPacket
	files := compactStringsV0(packet.Task.WriteSet)
	if len(files) == 0 {
		files = []string{"README.md"}
	}
	data, err := json.Marshal(orquestaruntimecodex.CodexAgentAckV0{
		SchemaVersion: orquestaruntimecodex.CodexAgentAckSchemaVersionV0,
		RequestID:     descriptor.Spec.RequestID,
		CorrelationID: descriptor.Spec.CorrelationID,
		AckRef:        packet.DeliveryRefs.AckRef,
		TargetModule:  packet.TargetModule,
		TaskRef:       packet.Task.TaskRef,
		Status:        "completed",
		Files:         orquestaruntimecodex.EvidenceListV0{files[0]},
		Tests:         orquestaruntimecodex.EvidenceListV0(packet.Task.RequiredTests),
		TestReceipts:  codexStackRequiredTestReceiptsV0(packet.Task.RequiredTests),
		Notes:         orquestaruntimecodex.EvidenceListV0{"contexto_ref_only_resuelto: contexto de test validado"},
	})
	if err != nil {
		return err
	}
	return os.WriteFile(descriptor.AckPath, data, 0o600)
}
