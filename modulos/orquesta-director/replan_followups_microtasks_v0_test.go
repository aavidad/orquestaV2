package orquestadirector

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestBuildReplanFollowupsV0ReviewReworkSplitConstruyeMicrotareas(t *testing.T) {
	input := validReplanFollowupsInputV0(orquestacoreworkflow.ReplanDecisionActionSplitTaskV0, "review-split-tasks")
	input.SourceKind = ReplanFollowupSourceReviewReworkV0
	input.OpenPhaseCandidate = validReplanOpenPhaseCandidateV0("review-split-tasks", input.DecisionCommandMeta.RunID)
	input.MicrotaskCandidates = []ReplanMicrotaskCandidateV0{
		validReplanMicrotaskCandidateV0("review-split-a", input.DecisionCommandMeta.RunID),
		validReplanMicrotaskCandidateV0("review-split-b", input.DecisionCommandMeta.RunID),
	}

	result, err := BuildReplanFollowupsV0(input)
	if err != nil {
		t.Fatalf("BuildReplanFollowupsV0 split microtasks: %v", err)
	}
	if result.FollowupStatus != ReplanFollowupStatusMicrotasksRequestedV0 {
		t.Fatalf("followup_status=%q", result.FollowupStatus)
	}
	assertOptionalCommandTypeV0(t, result.OpenPhaseCommand, orquestacoreworkflow.OrchestrationCommandOpenPhaseV0)
	if len(result.CreateMicrotaskCommands) != 2 {
		t.Fatalf("create_microtask_commands=%d", len(result.CreateMicrotaskCommands))
	}
	assertCommandTypeV0(t, result.CreateMicrotaskCommands[0], orquestacoreworkflow.OrchestrationCommandCreateMicrotaskV0)
	if result.AskDirectorCommand != nil || result.RequestCapacityCommand != nil || result.RequestAgentCommand != nil {
		t.Fatalf("comandos secundarios inesperados: %+v", result)
	}
}

func validReplanMicrotaskCandidateV0(suffix string, runRef string) ReplanMicrotaskCandidateV0 {
	contractRef := "contract:function:rework-split:v0"
	return ReplanMicrotaskCandidateV0{
		CommandMeta: validReplanCommandMetaV0("create-microtask-"+suffix, runRef),
		Payload: orquestacoreworkflow.CreateMicrotaskCommandPayloadV0{
			Task: orquestacoreworkflow.WorkflowTaskV0{
				SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
				TaskID:        "task-ref-" + suffix,
				RunID:         runRef,
				PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
				Title:         "Microtarea de retrabajo " + suffix,
				Summary:       "Trabajo compacto tras revision no aceptada.",
				WriteSet:      []string{"app/" + suffix + ".go"},
				AcceptanceCriteria: []string{
					"Solo se modifica el write-set declarado.",
					"La entrega queda lista para una nueva revision.",
				},
				FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{
					{ContractRef: contractRef, FunctionName: "NewWorkflowTaskV0"},
				},
			},
		},
	}
}
