package orquestadirectorscheduler

import (
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func TestBuildDirectorSchedulerTickV0ReplanSplitCreatesMicrotasksAfterOpenPhase(t *testing.T) {
	input := validSchedulerTickInputWithReplanV0(orquestacoreworkflow.ReplanDecisionActionSplitTaskV0)
	input.Snapshot.CurrentPhaseID = string(orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	replan := &input.ReplanFollowupCandidates[0].ReplanFollowupsInput
	replan.SourceKind = orquestadirector.ReplanFollowupSourceReviewReworkV0
	replan.OpenPhaseCandidate = validReplanOpenPhaseCandidateForSchedulerV0()
	replan.AskDirectorCandidate = nil
	replan.MicrotaskCandidates = []orquestadirector.ReplanMicrotaskCandidateV0{
		validSchedulerReplanMicrotaskV0("split-a"),
		validSchedulerReplanMicrotaskV0("split-b"),
	}

	plan := mustSchedulerTickPlanV0(t, input)

	assertSchedulerPlanV0(t, plan, SchedulerTickStatusCommandsReadyV0, 4)
	assertSchedulerCommandTypesV0(t, plan,
		orquestacoreworkflow.OrchestrationCommandRecordReplanDecisionV0,
		orquestacoreworkflow.OrchestrationCommandOpenPhaseV0,
		orquestacoreworkflow.OrchestrationCommandCreateMicrotaskV0,
		orquestacoreworkflow.OrchestrationCommandCreateMicrotaskV0,
	)
}

func validSchedulerReplanMicrotaskV0(suffix string) orquestadirector.ReplanMicrotaskCandidateV0 {
	runRef := "run-scheduler-001"
	return orquestadirector.ReplanMicrotaskCandidateV0{
		CommandMeta: schedulerMetaV0("cmd-create-microtask-"+suffix, "create-microtask-"+suffix),
		Payload: orquestacoreworkflow.CreateMicrotaskCommandPayloadV0{
			Task: orquestacoreworkflow.WorkflowTaskV0{
				SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
				TaskID:        "task-ref-replan-" + suffix,
				RunID:         runRef,
				PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
				Title:         "Microtarea replan " + suffix,
				Summary:       "Trabajo compacto tras revision no aceptada.",
				WriteSet:      []string{"app/" + suffix + ".go"},
				AcceptanceCriteria: []string{
					"Solo se modifica el write-set declarado.",
					"La entrega queda lista para revision.",
				},
				FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{
					{ContractRef: "contract:function:rework-split:v0", FunctionName: "NewWorkflowTaskV0"},
				},
			},
		},
	}
}
