package orquestadirectoragentfilesource

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
)

func TestDirectorAgentDecisionFileSourceV0ReadsEnvelope(t *testing.T) {
	data := mustDecisionFileJSONForTestV0(t, []orquestadirectoragent.DirectorAgentDecisionV0{
		validOpenVoteDecisionForTestV0("run-ref-001"),
	})
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: decisionFileDescriptorProviderForTestV0{
			Descriptors: []DirectorAgentDecisionFileDescriptorV0{{
				DescriptorRef: "descriptor-ref-001",
				RunID:         "run-ref-001",
				Path:          "artifact-ref-001.json",
			}},
		},
		Reader: memoryDecisionFileReaderForTestV0{
			Files: map[string][]byte{"artifact-ref-001.json": data},
		},
	}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		decisionSourceRequestForTestV0("run-ref-001"),
	)
	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 1 ||
		decisions[0].CommandType != orquestadirectoragent.DirectorAgentCommandOpenPhaseV0 {
		t.Fatalf("decisions=%+v", decisions)
	}
}

func TestDirectorAgentDecisionFileSourceV0FiltersForeignRun(t *testing.T) {
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: decisionFileDescriptorProviderForTestV0{
			Descriptors: []DirectorAgentDecisionFileDescriptorV0{{
				DescriptorRef: "descriptor-ref-foreign",
				RunID:         "run-ref-foreign",
				Path:          "missing.json",
			}},
		},
		Reader: memoryDecisionFileReaderForTestV0{Files: map[string][]byte{}},
	}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		decisionSourceRequestForTestV0("run-ref-001"),
	)
	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 0 {
		t.Fatalf("decisions=%+v", decisions)
	}
}

func TestDirectorAgentDecisionFileSourceV0RejectsInvalidDecision(t *testing.T) {
	data, err := json.Marshal(validOpenVoteDecisionForTestV0("run-ref-001"))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	delete(raw, "open_phase")
	data, err = json.Marshal(raw)
	if err != nil {
		t.Fatalf("Marshal invalid: %v", err)
	}
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: decisionFileDescriptorProviderForTestV0{
			Descriptors: []DirectorAgentDecisionFileDescriptorV0{{
				RunID: "run-ref-001",
				Path:  "invalid.json",
			}},
		},
		Reader: memoryDecisionFileReaderForTestV0{
			Files: map[string][]byte{"invalid.json": data},
		},
	}

	_, err = source.ListDirectorAgentDecisionsV0(
		context.Background(),
		decisionSourceRequestForTestV0("run-ref-001"),
	)
	if err == nil {
		t.Fatalf("esperaba error")
	}
}

func TestOSDirectorAgentDecisionFileReaderV0RejectsOversizedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "decision.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"x"}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := (OSDirectorAgentDecisionFileReaderV0{}).ReadDirectorAgentDecisionFileV0(
		context.Background(),
		path,
		4,
	)
	if err == nil {
		t.Fatalf("esperaba error de tamano")
	}
}

type decisionFileDescriptorProviderForTestV0 struct {
	Descriptors []DirectorAgentDecisionFileDescriptorV0
}

func (provider decisionFileDescriptorProviderForTestV0) ListDirectorAgentDecisionFilesV0(
	ctx context.Context,
	_ DirectorAgentDecisionFileListRequestV0,
) ([]DirectorAgentDecisionFileDescriptorV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]DirectorAgentDecisionFileDescriptorV0(nil), provider.Descriptors...), nil
}

type memoryDecisionFileReaderForTestV0 struct {
	Files map[string][]byte
}

func (reader memoryDecisionFileReaderForTestV0) ReadDirectorAgentDecisionFileV0(
	ctx context.Context,
	path string,
	_ int,
) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, ok := reader.Files[path]
	if !ok {
		return nil, DirectorAgentFileSourceIssueV0{Field: "path"}
	}
	return append([]byte(nil), data...), nil
}

func decisionSourceRequestForTestV0(
	runRef string,
) orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0 {
	return orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{
		Run: orquestacoreworkflow.OrchestrationRunV0{
			RunID: runRef,
		},
		OccurredAt:    "2026-05-09T23:00:00Z",
		CorrelationID: "corr-decision-file-source-001",
		RequestedBy:   "orquesta-director-agent-file-source-test",
	}
}

func mustDecisionFileJSONForTestV0(
	t *testing.T,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []byte {
	t.Helper()
	data, err := json.Marshal(DirectorAgentDecisionFileEnvelopeV0{
		SchemaVersion: DirectorAgentDecisionFileSchemaVersionV0,
		Decisions:     decisions,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return data
}

func validOpenVoteDecisionForTestV0(
	runRef string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-file-open-vote-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandOpenPhaseV0,
		CommandRef:    "command-ref-file-open-vote-001",
		Summary:       "Abrir fase de decision.",
		EvidenceRefs:  []string{"evidence-ref-file-open-vote-001"},
		OpenPhase: &orquestadirectoragent.DirectorAgentOpenPhaseCommandV0{
			PhaseID: string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			Reason:  "Preparar decision.",
		},
	}
}
