package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentfilesource "orquesta/modulos/orquesta-director-agent-file-source"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestCodexStackV0ConsumeDecisionFileYArrancaProgramacion(t *testing.T) {
	runtime := newDecisionWritingFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)

	director := postDirectorAPIV0(t, stack)
	if runtime.launchCountV0() != 4 {
		t.Fatalf("launches iniciales=%d want=4", runtime.launchCountV0())
	}
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if run.CurrentPhase == orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("decision consumida durante arranque inicial: phase=%s", run.CurrentPhase)
	}
	if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-stack-decision-drain-001",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     4,
	}); err != nil {
		t.Fatalf("DrainRunV0: %v", err)
	}
	run, err = stack.Stores.RunStore.LoadRunV0(context.Background(), director.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0 tras drain: %v", err)
	}

	taskRef := "task-ref-stack-agenda-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%s", run.CurrentPhase)
	}
	if !codexStackStringInSetForTestV0(run.Tasks, taskRef) {
		t.Fatalf("tasks=%v missing=%s", run.Tasks, taskRef)
	}
	if !codexStackStringInSetForTestV0(run.StartedAgents, agentRef) {
		t.Fatalf("loop=%s started_agents=%v missing=%s", director.LoopStatus, run.StartedAgents, agentRef)
	}
	if runtime.launchCountV0() < 5 {
		t.Fatalf("launches=%d want>=5", runtime.launchCountV0())
	}
}

type decisionWritingFakeCodexStackRuntimeV0 struct {
	*fakeCodexStackRuntimeV0
}

func newDecisionWritingFakeCodexStackRuntimeV0() *decisionWritingFakeCodexStackRuntimeV0 {
	return &decisionWritingFakeCodexStackRuntimeV0{
		fakeCodexStackRuntimeV0: newFakeCodexStackRuntimeV0(),
	}
}

func (runtime *decisionWritingFakeCodexStackRuntimeV0) LaunchV0(
	ctx context.Context,
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) (orquestaruntime.ProcessRuntimeSnapshotV0, error) {
	snapshot, err := runtime.fakeCodexStackRuntimeV0.LaunchV0(ctx, req)
	if err != nil {
		return snapshot, err
	}
	return snapshot, runtime.writeDirectorDecisionsV0(req)
}

func (runtime *decisionWritingFakeCodexStackRuntimeV0) writeDirectorDecisionsV0(
	req orquestaruntime.ProcessRuntimeLaunchRequestV0,
) error {
	runtimeDir := filepath.Dir(req.CommandPath)
	packet, err := codexStackPacketFromRuntimeDirForTestV0(runtimeDir)
	if err != nil {
		return err
	}
	if packet.TargetModule != "orquesta-app-stack-director" {
		return nil
	}
	runRef := codexStackObjectiveValueForTestV0(packet.Task.Objective, "RunID para decisiones:")
	brainstormRef := codexStackObjectiveValueForTestV0(packet.Task.Objective, "BrainstormRef inicial:")
	if runRef == "" || brainstormRef == "" {
		return fmt.Errorf("contrato de decisiones incompleto")
	}
	data, err := json.Marshal(orquestadirectoragentfilesource.DirectorAgentDecisionFileEnvelopeV0{
		SchemaVersion: orquestadirectoragentfilesource.DirectorAgentDecisionFileSchemaVersionV0,
		Decisions:     codexStackDirectorDecisionsForTestV0(runRef, brainstormRef),
	})
	if err != nil {
		return err
	}
	return os.WriteFile(
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexDirectorDecisionsFileNameV0),
		data,
		0o600,
	)
}

func codexStackPacketFromRuntimeDirForTestV0(
	runtimeDir string,
) (orquestaruntime.AgentStartPacketV0, error) {
	data, err := os.ReadFile(filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentPacketFileNameV0))
	if err != nil {
		return orquestaruntime.AgentStartPacketV0{}, err
	}
	var packet orquestaruntime.AgentStartPacketV0
	if err := json.Unmarshal(data, &packet); err != nil {
		return orquestaruntime.AgentStartPacketV0{}, err
	}
	return packet, nil
}

func codexStackObjectiveValueForTestV0(objective string, prefix string) string {
	for _, line := range strings.Split(objective, "\n") {
		value, ok := strings.CutPrefix(strings.TrimSpace(line), prefix)
		if !ok {
			continue
		}
		return strings.Trim(strings.TrimSpace(value), ".")
	}
	return ""
}

func codexStackDirectorDecisionsForTestV0(
	runRef string,
	brainstormRef string,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	return []orquestadirectoragent.DirectorAgentDecisionV0{
		codexStackOpenPhaseDecisionForTestV0(
			runRef,
			"director-decision-stack-open-vote-001",
			"command-ref-stack-open-vote-001",
			orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
			orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
		),
		codexStackVoteDecisionForTestV0(runRef, brainstormRef),
		codexStackAcceptDecisionForTestV0(runRef),
		codexStackOpenPhaseDecisionForTestV0(
			runRef,
			"director-decision-stack-open-plan-001",
			"command-ref-stack-open-plan-001",
			orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
			orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		),
		codexStackContractDecisionForTestV0(runRef),
		codexStackMicrotaskDecisionForTestV0(runRef),
		codexStackOpenPhaseDecisionForTestV0(
			runRef,
			"director-decision-stack-open-program-001",
			"command-ref-stack-open-program-001",
			orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
			orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		),
	}
}

func codexStackOpenPhaseDecisionForTestV0(
	runRef string,
	decisionRef string,
	commandRef string,
	currentPhase orquestacoreworkflow.OrchestrationPhaseIDV0,
	nextPhase orquestacoreworkflow.OrchestrationPhaseIDV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   decisionRef,
		RunID:         runRef,
		PhaseID:       string(currentPhase),
		CommandType:   orquestadirectoragent.DirectorAgentCommandOpenPhaseV0,
		CommandRef:    commandRef,
		Summary:       "Avanzar fase del flujo.",
		EvidenceRefs:  []string{"evidence-ref-" + decisionRef},
		OpenPhase: &orquestadirectoragent.DirectorAgentOpenPhaseCommandV0{
			PhaseID: string(nextPhase),
			Reason:  "Salida previa suficiente.",
		},
	}
}

func codexStackVoteDecisionForTestV0(
	runRef string,
	brainstormRef string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-stack-vote-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandRequestVoteV0,
		CommandRef:    "command-ref-stack-vote-001",
		Summary:       "Solicitar voto tecnico.",
		EvidenceRefs:  []string{"evidence-ref-stack-vote-001"},
		RequestVote: &orquestadirectoragent.DirectorAgentVoteCommandV0{
			VoteRequestID:              "vote-ref-stack-001",
			PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			DecisionTopicRef:           "topic-ref-stack-agenda-001",
			BrainstormRef:              brainstormRef,
			Summary:                    "Elegir arquitectura compacta.",
			MinimumRecommendedCapacity: orquestadirectoragent.DirectorAgentCapacityHighV0,
			EvidenceRefs:               []string{"evidence-ref-stack-vote-001"},
		},
	}
}

func codexStackAcceptDecisionForTestV0(runRef string) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-stack-accept-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandAcceptDecisionV0,
		CommandRef:    "command-ref-stack-accept-001",
		Summary:       "Aceptar opcion compacta.",
		EvidenceRefs:  []string{"evidence-ref-stack-accept-001"},
		AcceptDecision: &orquestadirectoragent.DirectorAgentAcceptDecisionCommandV0{
			DecisionRef:       "decision-ref-stack-001",
			PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
			VoteRef:           "vote-ref-stack-001",
			AcceptedOptionRef: "option-ref-stack-001",
			Summary:           "Arquitectura con puertos y textos externos.",
			EvidenceRefs:      []string{"evidence-ref-stack-accept-001"},
		},
	}
}

func codexStackContractDecisionForTestV0(runRef string) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-stack-contract-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandPublishContractV0,
		CommandRef:    "command-ref-stack-contract-001",
		Summary:       "Publicar contrato funcional.",
		EvidenceRefs:  []string{"evidence-ref-stack-contract-001"},
		PublishContract: &orquestadirectoragent.DirectorAgentPublishContractCommandV0{
			ContractRef:   "contract:function:stack-agenda:v0",
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
			DecisionRef:   "decision-ref-stack-001",
			Summary:       "Contrato para agenda compacta.",
			FunctionNames: []string{"AgendaUseCases"},
			EvidenceRefs:  []string{"evidence-ref-stack-contract-001"},
		},
	}
}

func codexStackMicrotaskDecisionForTestV0(runRef string) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-stack-task-001",
		RunID:         runRef,
		PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
		CommandType:   orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0,
		CommandRef:    "command-ref-stack-task-001",
		Summary:       "Crear microtarea de agenda.",
		EvidenceRefs:  []string{"evidence-ref-stack-task-001"},
		CreateMicrotask: &orquestadirectoragent.DirectorAgentCreateMicrotaskCommandV0{
			Task: orquestadirectoragent.DirectorAgentMicrotaskV0{
				SchemaVersion: orquestadirectoragent.DirectorAgentMicrotaskSchemaVersionV0,
				TaskID:        "task-ref-stack-agenda-001",
				RunID:         runRef,
				PhaseID:       string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
				Title:         "Crear agenda minima",
				Summary:       "Implementar casos de uso de agenda.",
				WriteSet:      []string{"go.mod", "cmd/server", "internal/agenda", "README.md"},
				AcceptanceCriteria: []string{
					"Compila con go test ./....",
					"Usa imports de modulo desde go.mod y no imports relativos.",
					"Expone contrato funcional.",
				},
				RequiredTests: []string{"go test ./..."},
				FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{{
					ContractRef:  "contract:function:stack-agenda:v0",
					FunctionName: "AgendaUseCases",
				}},
			},
		},
	}
}
