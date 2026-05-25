package orquestadirectoragentfilesource

import (
	"context"
	"testing"

	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func TestDirectorAgentDecisionFileSourceV0ValidaYConsumeSidecarReceipt(t *testing.T) {
	data := mustDecisionFileJSONForTestV0(t, []orquestadirectoragent.DirectorAgentDecisionV0{
		validOpenVoteDecisionForTestV0("run-ref-001"),
	})
	recorder := &decisionFileConsumptionRecorderForTestV0{}
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: decisionFileDescriptorProviderForTestV0{
			Descriptors: []DirectorAgentDecisionFileDescriptorV0{{
				RunID: "run-ref-001",
				Path:  "sidecar.json",
				SidecarReceipt: &DirectorAgentDecisionSidecarReceiptV0{
					SchemaVersion: DirectorAgentDecisionSidecarReceiptSchemaV0,
					ReceiptRef:    "decision-sidecar-receipt-ref-001",
					SHA256:        directorAgentDecisionFileSHA256V0(data),
					SizeBytes:     int64(len(data)),
					Status:        "pending",
				},
			}},
		},
		Reader: memoryDecisionFileReaderForTestV0{
			Files: map[string][]byte{"sidecar.json": data},
		},
		ConsumptionRecorder: recorder,
	}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		decisionSourceRequestForTestV0("run-ref-001"),
	)
	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("decisions=%+v", decisions)
	}
	if len(recorder.Receipts) != 1 || recorder.Receipts[0].Status != "consumed" {
		t.Fatalf("receipts=%+v", recorder.Receipts)
	}
}

func TestDirectorAgentDecisionFileSourceV0RechazaSidecarHashCambiante(t *testing.T) {
	data := mustDecisionFileJSONForTestV0(t, []orquestadirectoragent.DirectorAgentDecisionV0{
		validOpenVoteDecisionForTestV0("run-ref-001"),
	})
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: decisionFileDescriptorProviderForTestV0{
			Descriptors: []DirectorAgentDecisionFileDescriptorV0{{
				RunID: "run-ref-001",
				Path:  "sidecar.json",
				SidecarReceipt: &DirectorAgentDecisionSidecarReceiptV0{
					SchemaVersion: DirectorAgentDecisionSidecarReceiptSchemaV0,
					ReceiptRef:    "decision-sidecar-receipt-ref-001",
					SHA256:        "sha256-obsoleto",
					SizeBytes:     int64(len(data)),
					Status:        "pending",
				},
			}},
		},
		Reader: memoryDecisionFileReaderForTestV0{
			Files: map[string][]byte{"sidecar.json": data},
		},
	}

	_, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		decisionSourceRequestForTestV0("run-ref-001"),
	)
	if err == nil {
		t.Fatalf("esperaba error de hash")
	}
}
