package orquestadirectoragentfilesource

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

func TestDirectorAgentDecisionFileSourceV0PasaProyeccionesRunAlProvider(t *testing.T) {
	data := mustDecisionFileJSONForTestV0(t, []orquestadirectoragent.DirectorAgentDecisionV0{
		validOpenVoteDecisionForTestV0("run-ref-001"),
	})
	provider := &recordingDecisionFileDescriptorProviderForTestV0{
		Descriptors: []DirectorAgentDecisionFileDescriptorV0{{
			DescriptorRef: "descriptor-ref-001",
			RunID:         "run-ref-001",
			Path:          "artifact-ref-001.json",
		}},
	}
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: provider,
		Reader: memoryDecisionFileReaderForTestV0{
			Files: map[string][]byte{"artifact-ref-001.json": data},
		},
	}
	request := decisionSourceRequestForTestV0("run-ref-001")
	request.Run.PhaseArtifacts = []string{" artifact-ref-001#phase:brainstorming_arquitectura ", "artifact-ref-001#phase:brainstorming_arquitectura"}
	request.Run.Deliveries = []string{" delivery-ref-001 ", "delivery-ref-001"}

	if _, err := source.ListDirectorAgentDecisionsV0(context.Background(), request); err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if !reflect.DeepEqual(provider.LastRequest.PhaseArtifacts, []string{"artifact-ref-001#phase:brainstorming_arquitectura"}) {
		t.Fatalf("phase_artifacts=%v", provider.LastRequest.PhaseArtifacts)
	}
	if !reflect.DeepEqual(provider.LastRequest.Deliveries, []string{"delivery-ref-001"}) {
		t.Fatalf("deliveries=%v", provider.LastRequest.Deliveries)
	}
}

func TestDirectorAgentDecisionFileSourceV0NormalizaSchemasCompactos(t *testing.T) {
	decision := validMicrotaskDecisionForTestV0("run-ref-001")
	decision.SchemaVersion = ""
	decision.CreateMicrotask.Task.SchemaVersion = ""
	data := mustDecisionFileJSONForTestV0(t, []orquestadirectoragent.DirectorAgentDecisionV0{decision})
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: decisionFileDescriptorProviderForTestV0{
			Descriptors: []DirectorAgentDecisionFileDescriptorV0{{
				RunID: "run-ref-001",
				Path:  "compact.json",
			}},
		},
		Reader: memoryDecisionFileReaderForTestV0{
			Files: map[string][]byte{"compact.json": data},
		},
	}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		decisionSourceRequestForTestV0("run-ref-001"),
	)
	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	got := decisions[0]
	if got.SchemaVersion != orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0 ||
		got.CreateMicrotask.Task.SchemaVersion != orquestadirectoragent.DirectorAgentMicrotaskSchemaVersionV0 {
		t.Fatalf("schemas no normalizados: %+v", got)
	}
}

func TestDirectorAgentDecisionFileSourceV0NormalizaFaseDesdePayload(t *testing.T) {
	decision := validVoteDecisionForTestV0("run-ref-001")
	decision.PhaseID = string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0)
	data := mustDecisionFileJSONForTestV0(t, []orquestadirectoragent.DirectorAgentDecisionV0{decision})
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: decisionFileDescriptorProviderForTestV0{
			Descriptors: []DirectorAgentDecisionFileDescriptorV0{{
				RunID: "run-ref-001",
				Path:  "phase.json",
			}},
		},
		Reader: memoryDecisionFileReaderForTestV0{
			Files: map[string][]byte{"phase.json": data},
		},
	}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		decisionSourceRequestForTestV0("run-ref-001"),
	)
	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if decisions[0].PhaseID != string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0) {
		t.Fatalf("phase_id no normalizado: %+v", decisions[0])
	}
}

func TestDirectorAgentDecisionFileSourceV0NormalizaCreateMicrotaskPhaseObjetivo(t *testing.T) {
	decision := validMicrotaskDecisionForTestV0("run-ref-001")
	decision.PhaseID = decision.CreateMicrotask.Task.PhaseID
	data := mustDecisionFileJSONForTestV0(t, []orquestadirectoragent.DirectorAgentDecisionV0{decision})
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: decisionFileDescriptorProviderForTestV0{
			Descriptors: []DirectorAgentDecisionFileDescriptorV0{{
				RunID: "run-ref-001",
				Path:  "microtask-phase.json",
			}},
		},
		Reader: memoryDecisionFileReaderForTestV0{
			Files: map[string][]byte{"microtask-phase.json": data},
		},
	}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		decisionSourceRequestForTestV0("run-ref-001"),
	)
	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if decisions[0].PhaseID != orquestadirectoragent.DirectorAgentPlanningPhaseIDV0 {
		t.Fatalf("phase_id create_microtask no normalizado: %+v", decisions[0])
	}
}

func TestDirectorAgentDecisionFileSourceV0NormalizaCreateMicrotaskCommandTypeVersionado(t *testing.T) {
	decision := validMicrotaskDecisionForTestV0("run-ref-001")
	decision.CommandType = "create_microtask.v0"
	data := mustDecisionFileJSONForTestV0(t, []orquestadirectoragent.DirectorAgentDecisionV0{decision})
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: decisionFileDescriptorProviderForTestV0{
			Descriptors: []DirectorAgentDecisionFileDescriptorV0{{
				RunID: "run-ref-001",
				Path:  "microtask-command-type.json",
			}},
		},
		Reader: memoryDecisionFileReaderForTestV0{
			Files: map[string][]byte{"microtask-command-type.json": data},
		},
	}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		decisionSourceRequestForTestV0("run-ref-001"),
	)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if decisions[0].CommandType != orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0 {
		t.Fatalf("command_type create_microtask no normalizado: %+v", decisions[0])
	}
}

func TestDirectorAgentDecisionFileSourceV0ToleraComasFinalesDeAgente(t *testing.T) {
	data := []byte(`{
		"schema_version": "director_agent_decisions_file.v0",
		"decisions": [
			{
				"schema_version": "director_agent_decision.v0",
				"decision_ref": "director-decision-file-open-vote-001",
				"run_id": "run-ref-001",
				"phase_id": "brainstorming_arquitectura",
				"command_type": "open_phase",
				"command_ref": "command-ref-file-open-vote-001",
				"summary": "Abrir fase de decision.",
				"evidence_refs": ["evidence-ref-file-open-vote-001",],
				"open_phase": {
					"phase_id": "votacion_y_decision",
					"reason": "Preparar decision.",
				},
			},
		],
	}`)
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: decisionFileDescriptorProviderForTestV0{
			Descriptors: []DirectorAgentDecisionFileDescriptorV0{{
				RunID: "run-ref-001",
				Path:  "trailing-commas.json",
			}},
		},
		Reader: memoryDecisionFileReaderForTestV0{
			Files: map[string][]byte{"trailing-commas.json": data},
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

func TestDirectorAgentDecisionFileSourceV0NormalizaPayloadsPlanosDeAgente(t *testing.T) {
	data := []byte(`{
		"schema_version": "director_agent_decisions_file.v0",
		"decisions": [
			{
				"schema_version": "director_agent_decision.v0",
				"decision_ref": "decision-request-vote-architecture-v1",
				"run_id": "run-ref-001",
				"phase_id": "votacion_y_decision",
				"command_type": "request_vote",
				"command_ref": "command-request-vote-architecture-v1",
				"summary": "Solicitar voto tecnico.",
				"evidence_refs": ["evidence-ref-vote-v1"],
				"vote_request_id": "vote-ref-architecture-v1",
				"decision_topic_ref": "topic-architecture-v1",
				"brainstorm_ref": "brainstorm-ref-v1",
				"minimum_recommended_capacity": "high"
			},
			{
				"schema_version": "director_agent_decision.v0",
				"decision_ref": "decision-accept-architecture-v1",
				"run_id": "run-ref-001",
				"phase_id": "votacion_y_decision",
				"command_type": "accept_decision",
				"command_ref": "command-accept-architecture-v1",
				"summary": "Aceptar opcion tecnica.",
				"evidence_refs": ["evidence-ref-accept-v1"],
				"vote_ref": "vote-ref-architecture-v1",
				"accepted_option_ref": "option-hexagonal-i18n-v1"
			},
			{
				"schema_version": "director_agent_decision.v0",
				"decision_ref": "decision-publish-contract-v1",
				"run_id": "run-ref-001",
				"phase_id": "planificacion_microtareas",
				"command_type": "publish_function_contract",
				"command_ref": "command-publish-contract-v1",
				"summary": "Publicar contrato funcional.",
				"evidence_refs": ["evidence-ref-contract-v1"],
				"contract_ref": "contract-ref-agenda-v1",
				"function_names": ["CreateContact", "CreateAppointment"]
			}
		]
	}`)
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: decisionFileDescriptorProviderForTestV0{
			Descriptors: []DirectorAgentDecisionFileDescriptorV0{{
				RunID: "run-ref-001",
				Path:  "flat-payloads.json",
			}},
		},
		Reader: memoryDecisionFileReaderForTestV0{
			Files: map[string][]byte{"flat-payloads.json": data},
		},
	}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		decisionSourceRequestForTestV0("run-ref-001"),
	)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) != 3 ||
		decisions[0].RequestVote == nil ||
		decisions[1].AcceptDecision == nil ||
		decisions[2].PublishContract == nil {
		t.Fatalf("payloads no normalizados: %+v", decisions)
	}
}

func TestDirectorAgentDecisionFileSourceV0RechazaCreateMicrotaskVersionadoSinPayload(t *testing.T) {
	decision := validMicrotaskDecisionForTestV0("run-ref-001")
	decision.CommandType = "create_microtask.v0"
	decision.CreateMicrotask = nil
	data := mustDecisionFileJSONForTestV0(t, []orquestadirectoragent.DirectorAgentDecisionV0{decision})
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: decisionFileDescriptorProviderForTestV0{
			Descriptors: []DirectorAgentDecisionFileDescriptorV0{{
				RunID: "run-ref-001",
				Path:  "microtask-command-type-no-payload.json",
			}},
		},
		Reader: memoryDecisionFileReaderForTestV0{
			Files: map[string][]byte{"microtask-command-type-no-payload.json": data},
		},
	}

	_, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		decisionSourceRequestForTestV0("run-ref-001"),
	)

	if err == nil {
		t.Fatalf("esperaba error")
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

func TestDirectorAgentDecisionFileSourceV0FiltraDecisionRunExtranjeroSinRunEnDescriptor(t *testing.T) {
	data := mustDecisionFileJSONForTestV0(t, []orquestadirectoragent.DirectorAgentDecisionV0{
		validOpenVoteDecisionForTestV0("run-ref-foreign"),
	})
	source := DirectorAgentDecisionFileSourceV0{
		DescriptorProvider: decisionFileDescriptorProviderForTestV0{
			Descriptors: []DirectorAgentDecisionFileDescriptorV0{{
				Path: "foreign-decision.json",
			}},
		},
		Reader: memoryDecisionFileReaderForTestV0{
			Files: map[string][]byte{"foreign-decision.json": data},
		},
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

type recordingDecisionFileDescriptorProviderForTestV0 struct {
	Descriptors []DirectorAgentDecisionFileDescriptorV0
	LastRequest DirectorAgentDecisionFileListRequestV0
}

func (provider *recordingDecisionFileDescriptorProviderForTestV0) ListDirectorAgentDecisionFilesV0(
	ctx context.Context,
	request DirectorAgentDecisionFileListRequestV0,
) ([]DirectorAgentDecisionFileDescriptorV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	provider.LastRequest = request
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

func validMicrotaskDecisionForTestV0(
	runRef string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-file-task-001",
		RunID:         runRef,
		PhaseID:       "planificacion_microtareas",
		CommandType:   orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0,
		CommandRef:    "command-ref-file-task-001",
		Summary:       "Crear microtarea.",
		EvidenceRefs:  []string{"evidence-ref-file-task-001"},
		CreateMicrotask: &orquestadirectoragent.DirectorAgentCreateMicrotaskCommandV0{
			Task: orquestadirectoragent.DirectorAgentMicrotaskV0{
				SchemaVersion:      orquestadirectoragent.DirectorAgentMicrotaskSchemaVersionV0,
				TaskID:             "task-file-source-001",
				RunID:              runRef,
				PhaseID:            "programacion",
				Title:              "Implementar pieza pequena",
				Summary:            "Trabajo acotado.",
				WriteSet:           []string{"internal/app/app.go"},
				AcceptanceCriteria: []string{"Pieza creada."},
				RequiredTests:      []string{"go test ./..."},
				FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{{
					ContractRef:  "contract-file-source-001",
					FunctionName: "Build",
				}},
			},
		},
	}
}

func validVoteDecisionForTestV0(
	runRef string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-file-vote-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandRequestVoteV0,
		CommandRef:    "command-ref-file-vote-001",
		Summary:       "Solicitar voto tecnico.",
		EvidenceRefs:  []string{"evidence-ref-file-vote-001"},
		RequestVote: &orquestadirectoragent.DirectorAgentVoteCommandV0{
			VoteRequestID:              "vote-request-file-001",
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			DecisionTopicRef:           "decision-topic-file-001",
			BrainstormRef:              "brainstorm-ref-file-001",
			Summary:                    "Elegir arquitectura.",
			MinimumRecommendedCapacity: orquestadirectoragent.DirectorAgentCapacityHighV0,
			EvidenceRefs:               []string{"evidence-ref-file-vote-001"},
		},
	}
}
