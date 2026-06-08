package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcandidates "orquesta/modulos/orquesta-director-candidates"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	codexStackResidentActionKindMaterializeDecisionCouncilV0 = "materialize_decision_council"
	codexStackResidentCouncilReasonV0                        = "resident_decision_council_ready"
	codexStackResidentCouncilTaskPrefixV0                    = "task-council-"
)

func (source codexStackResidentBriefingSourceV0) shouldMaterializeDecisionCouncilV0(
	ctx context.Context,
	request orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0,
) bool {
	if source.RunStore == nil || source.TaskStore == nil {
		return false
	}
	run, err := source.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return false
	}
	return codexStackResidentCouncilReadyV0(ctx, run, source.TaskStore)
}

func codexStackResidentCouncilReadyV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	taskStore orquestacionnucleoapp.WorkflowTaskStorePortV0,
) bool {
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		!codexStackResidentCouncilCanCreateMicrotasksV0(run.CurrentPhase) ||
		len(compactStringsV0(run.Brainstorms)) == 0 ||
		len(compactStringsV0(run.Votes)) == 0 ||
		len(compactStringsV0(run.FunctionContracts)) == 0 {
		return false
	}
	return !codexStackResidentCouncilAlreadyMaterializedV0(ctx, run, taskStore)
}

func codexStackResidentCouncilCanCreateMicrotasksV0(
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
) bool {
	return phase == orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0 ||
		phase == orquestacoreworkflow.OrchestrationPhaseProgramacionV0
}

func codexStackResidentCouncilAlreadyMaterializedV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	taskStore orquestacionnucleoapp.WorkflowTaskStorePortV0,
) bool {
	for _, taskRef := range compactStringsV0(run.Tasks) {
		if strings.HasPrefix(taskRef, codexStackResidentCouncilTaskPrefixV0) {
			return true
		}
	}
	if taskStore == nil || len(compactStringsV0(run.Tasks)) == 0 {
		return false
	}
	tasks, err := taskStore.LoadWorkflowTasksV0(ctx, run.RunID, run.Tasks)
	if err != nil {
		return false
	}
	for _, task := range tasks {
		if strings.HasPrefix(strings.TrimSpace(task.TaskID), codexStackResidentCouncilTaskPrefixV0) {
			return true
		}
	}
	return false
}

func (source codexStackResidentBriefingSourceV0) decisionCouncilBriefingV0(
	request orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0,
) orquestadirectorsupervisor.DirectorSupervisorBriefingV0 {
	targets := codexStackResidentCouncilTargetRefsV0(request.RunRef, nil)
	evidence := compactStringsV0(append(
		[]string{"evidence-ref-codex-stack-resident-council-ready"},
		request.EvidenceRefs...,
	))
	action := orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{
		ActionRef:        fmt.Sprintf("director-action:%s:%03d:%s", request.RunRef, request.StepNumber, codexStackResidentActionKindMaterializeDecisionCouncilV0),
		Kind:             codexStackResidentActionKindMaterializeDecisionCouncilV0,
		RunRef:           request.RunRef,
		ReasonCode:       codexStackResidentCouncilReasonV0,
		Priority:         15,
		SafeToApply:      true,
		RequiresDirector: false,
		SourceAction:     orquestadirectorsupervisor.DirectorSupervisorActionContinueV0,
		TargetRefs:       targets,
		EvidenceRefs:     evidence,
	}
	return orquestadirectorsupervisor.DirectorSupervisorBriefingV0{
		SchemaVersion:            orquestadirectorsupervisor.DirectorSupervisorBriefingSchemaV0,
		RunRef:                   request.RunRef,
		ObjectiveRef:             firstNonEmptyQueuedSourceV0(source.ObjectiveRef, request.ObjectiveRef),
		DecisionAction:           orquestadirectorsupervisor.DirectorSupervisorActionContinueV0,
		AutonomousRecommendation: orquestadirectorsupervisor.DirectorSupervisorAutonomousContinueV0,
		ReasonCode:               codexStackResidentCouncilReasonV0,
		NextAction:               &action,
		ActionQueue:              []orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{action},
		Timeline: []orquestadirectorsupervisor.DirectorSupervisorTimelineEventV0{{
			EventRef:     action.ActionRef + ":event",
			Kind:         action.Kind,
			RunRef:       request.RunRef,
			StepNumber:   request.StepNumber,
			ReasonCode:   action.ReasonCode,
			SourceAction: action.SourceAction,
			TargetRefs:   append([]string(nil), action.TargetRefs...),
			EvidenceRefs: append([]string(nil), action.EvidenceRefs...),
		}},
		ContextRefs:  compactStringsV0(append(source.ContextRefs, request.ContextRefs...)),
		EvidenceRefs: evidence,
	}
}

type codexStackResidentExternalActionHandlerV0 struct {
	Request orquestaappdirectorservice.ContinueAppDirectorRequestV0
	Ports   orquestaappdirectorservice.StartAppDirectorPortsV0
}

func (handler codexStackResidentExternalActionHandlerV0) ExecuteDirectorBriefingExternalActionV0(
	ctx context.Context,
	request orquestacionnucleoapp.DirectorBriefingExternalActionRequestV0,
) (orquestacionnucleoapp.DirectorBriefingExternalActionResultV0, error) {
	if request.Action.Kind == codexStackResidentActionKindMaterializeDecisionCouncilV0 {
		return handler.materializeDecisionCouncilV0(ctx, request)
	}
	return (codexStackResidentCloseHandlerV0{
		Request: handler.Request,
		Ports:   handler.Ports,
	}).ExecuteDirectorBriefingExternalActionV0(ctx, request)
}

func (handler codexStackResidentExternalActionHandlerV0) materializeDecisionCouncilV0(
	ctx context.Context,
	request orquestacionnucleoapp.DirectorBriefingExternalActionRequestV0,
) (orquestacionnucleoapp.DirectorBriefingExternalActionResultV0, error) {
	runRef := firstNonEmptyQueuedSourceV0(request.Briefing.RunRef, request.Action.RunRef)
	result := orquestacionnucleoapp.DirectorBriefingExternalActionResultV0{
		RunRef:       runRef,
		ActionRef:    request.Action.ActionRef,
		EvidenceRefs: compactStringsV0(request.EvidenceRefs),
	}
	if handler.Ports.RunStore == nil || handler.Ports.EventSink == nil || handler.Ports.DirectorTaskStore == nil {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-ports-missing"))
		return result, nil
	}
	run, err := handler.Ports.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		return result, err
	}
	if codexStackResidentCouncilAlreadyMaterializedV0(ctx, run, handler.Ports.DirectorTaskStore) {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-already-materialized"))
		return result, nil
	}
	functionRefs := codexStackResidentCouncilFunctionContractRefsV0(run)
	if len(functionRefs) == 0 {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-function-contract-missing"))
		return result, nil
	}
	plan, err := orquestadirectorcandidates.BuildDecisionCouncilTeamPlanFromComplexityV0(
		orquestadirectorcandidates.TeamPlanFromComplexityInputV0{
			RunRef:               run.RunID,
			DecisionTopicRef:     codexStackResidentCouncilTopicRefV0(run.RunID),
			BrainstormRequestRef: codexStackResidentCouncilBrainstormRefV0(run),
			VoteRequestRef:       codexStackResidentCouncilVoteRefV0(run),
			Complexity:           codexStackResidentCouncilComplexityV0(run),
			EvidenceRefs: compactStringsV0(append(
				[]string{"evidence-ref-codex-stack-resident-council-plan"},
				request.EvidenceRefs...,
			)),
		},
	)
	if err != nil {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-plan-pending"))
		return result, nil
	}
	materialized, err := (orquestacionnucleoapp.DecisionCouncilPlanMaterializerV0{
		RunStore:    handler.Ports.RunStore,
		EventSink:   handler.Ports.EventSink,
		TaskWriter:  handler.Ports.DirectorTaskStore,
		RequestedBy: "orquesta-codex-stack-resident-council",
	}).MaterializeDecisionCouncilPlanV0(ctx, orquestacionnucleoapp.DecisionCouncilPlanMaterializeRequestV0{
		Plan:                 plan,
		FunctionContractRefs: functionRefs,
		OccurredAt:           request.OccurredAt,
		CorrelationID:        request.CorrelationID,
	})
	if err != nil {
		return result, err
	}
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, codexStackResidentCouncilTaskRefsV0(materialized.Tasks)...))
	if len(materialized.Issues) > 0 {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-materializer-issues"))
		return result, nil
	}
	result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-materialized"))
	return result, nil
}

func codexStackResidentCouncilTargetRefsV0(
	runRef string,
	run *orquestacoreworkflow.OrchestrationRunV0,
) []string {
	if run == nil {
		return []string{
			codexStackResidentCouncilTopicRefV0(runRef),
			"brainstorm-ref-resident-council-" + codexStackOperationalClosureSafeRefV0(runRef),
			"vote-ref-resident-council-" + codexStackOperationalClosureSafeRefV0(runRef),
		}
	}
	return []string{
		codexStackResidentCouncilTopicRefV0(run.RunID),
		codexStackResidentCouncilBrainstormRefV0(*run),
		codexStackResidentCouncilVoteRefV0(*run),
	}
}

func codexStackResidentCouncilTopicRefV0(runRef string) string {
	return "topic-ref-resident-council-" + codexStackOperationalClosureSafeRefV0(runRef)
}

func codexStackResidentCouncilBrainstormRefV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) string {
	if ref := lastCompactStringV0(run.Brainstorms); ref != "" {
		return ref
	}
	return "brainstorm-ref-resident-council-" + codexStackOperationalClosureSafeRefV0(run.RunID)
}

func codexStackResidentCouncilVoteRefV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) string {
	if ref := lastCompactStringV0(run.Votes); ref != "" {
		return ref
	}
	return "vote-ref-resident-council-" + codexStackOperationalClosureSafeRefV0(run.RunID)
}

func codexStackResidentCouncilFunctionContractRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []orquestacoreworkflow.WorkflowFunctionContractRefV0 {
	contracts := compactStringsV0(run.FunctionContracts)
	out := make([]orquestacoreworkflow.WorkflowFunctionContractRefV0, 0, len(contracts))
	for _, contract := range contracts {
		out = append(out, orquestacoreworkflow.WorkflowFunctionContractRefV0{
			ContractRef:  contract,
			FunctionName: "DecisionCouncilRound",
		})
	}
	return out
}

func codexStackResidentCouncilComplexityV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) string {
	for _, phase := range run.Phases {
		if phase.ID == run.CurrentPhase && strings.TrimSpace(string(phase.RecommendedCapacity)) != "" {
			return string(phase.RecommendedCapacity)
		}
	}
	return string(orquestacoreworkflow.OrchestrationCapacityHighV0)
}

func codexStackResidentCouncilTaskRefsV0(
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) []string {
	out := make([]string, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, task.TaskID)
	}
	return compactStringsV0(out)
}

func lastCompactStringV0(values []string) string {
	compact := compactStringsV0(values)
	if len(compact) == 0 {
		return ""
	}
	return compact[len(compact)-1]
}
