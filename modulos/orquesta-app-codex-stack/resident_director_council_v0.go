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
	codexStackResidentActionKindOpenCouncilBrainstormV0      = "open_decision_council_brainstorm_phase"
	codexStackResidentActionKindOpenCouncilVoteV0            = "open_decision_council_vote_phase"
	codexStackResidentActionKindAcceptCouncilDecisionV0      = "accept_decision_council_result"
	codexStackResidentCouncilReasonV0                        = "resident_decision_council_ready"
	codexStackResidentCouncilOpenBrainstormReasonV0          = "resident_decision_council_open_brainstorm"
	codexStackResidentCouncilOpenVoteReasonV0                = "resident_decision_council_open_vote"
	codexStackResidentCouncilAcceptReasonV0                  = "resident_decision_council_accept_result"
	codexStackResidentCouncilTaskPrefixV0                    = "task-council-"
	codexStackResidentCouncilProposalTaskPrefixV0            = "task-council-p-"
	codexStackResidentCouncilCritiqueTaskPrefixV0            = "task-council-c-"
	codexStackResidentCouncilVoteTaskPrefixV0                = "task-council-v-"
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

func (source codexStackResidentBriefingSourceV0) shouldOpenDecisionCouncilBrainstormPhaseV0(
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
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		run.CurrentPhase == orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0 ||
		!codexStackResidentCouncilCanCreateMicrotasksV0(run.CurrentPhase) {
		return false
	}
	tasks, ok := codexStackResidentCouncilTasksV0(ctx, run, source.TaskStore)
	if !ok {
		return false
	}
	return codexStackResidentCouncilHasPendingBrainstormWorkV0(run, tasks)
}

func (source codexStackResidentBriefingSourceV0) shouldOpenDecisionCouncilVotePhaseV0(
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
	if run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0 {
		return false
	}
	tasks, ok := codexStackResidentCouncilTasksV0(ctx, run, source.TaskStore)
	if !ok {
		return false
	}
	return codexStackResidentCouncilReadyForVotePhaseV0(run, tasks)
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

func (source codexStackResidentBriefingSourceV0) decisionCouncilOpenPhaseBriefingV0(
	request orquestacionnucleoapp.ResidentDirectorBriefingBuildRequestV0,
	kind string,
	reason string,
	priority int,
) orquestadirectorsupervisor.DirectorSupervisorBriefingV0 {
	targets := codexStackResidentCouncilTargetRefsV0(request.RunRef, nil)
	evidence := compactStringsV0(append(
		[]string{"evidence-ref-codex-stack-resident-council-phase-ready"},
		request.EvidenceRefs...,
	))
	action := orquestadirectorsupervisor.DirectorSupervisorRecommendedActionV0{
		ActionRef:        fmt.Sprintf("director-action:%s:%03d:%s", request.RunRef, request.StepNumber, kind),
		Kind:             kind,
		RunRef:           request.RunRef,
		ReasonCode:       reason,
		Priority:         priority,
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
		ReasonCode:               reason,
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
	Request         orquestaappdirectorservice.ContinueAppDirectorRequestV0
	Ports           orquestaappdirectorservice.StartAppDirectorPortsV0
	DecisionCouncil DecisionCouncilConfigV0
}

func (handler codexStackResidentExternalActionHandlerV0) ExecuteDirectorBriefingExternalActionV0(
	ctx context.Context,
	request orquestacionnucleoapp.DirectorBriefingExternalActionRequestV0,
) (orquestacionnucleoapp.DirectorBriefingExternalActionResultV0, error) {
	switch request.Action.Kind {
	case codexStackResidentActionKindMaterializeDecisionCouncilV0:
		return handler.materializeDecisionCouncilV0(ctx, request)
	case codexStackResidentActionKindOpenCouncilBrainstormV0:
		return handler.openDecisionCouncilPhaseV0(
			ctx,
			request,
			orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
			codexStackResidentCouncilOpenBrainstormReasonV0,
		)
	case codexStackResidentActionKindOpenCouncilVoteV0:
		return handler.openDecisionCouncilPhaseV0(
			ctx,
			request,
			orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
			codexStackResidentCouncilOpenVoteReasonV0,
		)
	case codexStackResidentActionKindAcceptCouncilDecisionV0:
		return handler.acceptDecisionCouncilResultV0(ctx, request)
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
	phaseResult, err := handler.openDecisionCouncilPhaseV0(
		ctx,
		request,
		orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
		codexStackResidentCouncilOpenBrainstormReasonV0,
	)
	if err != nil {
		return result, err
	}
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, phaseResult.EvidenceRefs...))
	if phaseResult.Status != orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0 {
		result.Status = phaseResult.Status
		return result, nil
	}
	result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-materialized"))
	return result, nil
}

func (handler codexStackResidentExternalActionHandlerV0) openDecisionCouncilPhaseV0(
	ctx context.Context,
	request orquestacionnucleoapp.DirectorBriefingExternalActionRequestV0,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
	reason string,
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
	tasks, ok := codexStackResidentCouncilTasksV0(ctx, run, handler.Ports.DirectorTaskStore)
	if !ok {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-tasks-missing"))
		return result, nil
	}
	if phase == orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0 &&
		!codexStackResidentCouncilReadyForVotePhaseV0(run, tasks) &&
		run.CurrentPhase != phase {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-vote-not-ready"))
		return result, nil
	}
	if phase == orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0 &&
		!codexStackResidentCouncilHasPendingBrainstormWorkV0(run, tasks) &&
		run.CurrentPhase != phase {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalPendingV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-brainstorm-not-ready"))
		return result, nil
	}
	if run.CurrentPhase == phase {
		result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0
		result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-phase-already-open"))
		return result, nil
	}
	command, err := codexStackResidentCouncilOpenPhaseCommandV0(
		runRef,
		phase,
		reason,
		request.OccurredAt,
		request.CorrelationID,
	)
	if err != nil {
		return result, err
	}
	if _, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(
		ctx,
		handler.Ports.RunStore,
		handler.Ports.EventSink,
		command,
	); err != nil {
		return result, err
	}
	result.Status = orquestacionnucleoapp.DirectorBriefingExecutionStatusExternalAppliedV0
	result.EvidenceRefs = compactStringsV0(append(result.EvidenceRefs, "evidence-ref-codex-stack-resident-council-phase-opened-"+string(phase)))
	return result, nil
}

func codexStackResidentCouncilOpenPhaseCommandV0(
	runRef string,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
	reason string,
	occurredAt string,
	correlationID string,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	suffix := codexStackOperationalClosureSafeRefV0(runRef + "-" + string(phase))
	return orquestacoreworkflow.NewOpenPhaseCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-resident-council-open-" + suffix,
			RunID:          runRef,
			IdempotencyKey: "idem-resident-council-open-" + suffix,
			CorrelationID:  correlationID,
			RequestedBy:    "orquesta-codex-stack-resident-council",
			OccurredAt:     occurredAt,
		},
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{
			PhaseID: string(phase),
			Reason:  reason,
		},
	)
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

func codexStackResidentCouncilTasksV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
	taskStore orquestacionnucleoapp.WorkflowTaskStorePortV0,
) ([]orquestacoreworkflow.WorkflowTaskV0, bool) {
	if taskStore == nil {
		return nil, false
	}
	taskRefs := codexStackResidentCouncilTaskRefsFromRunV0(run)
	if len(taskRefs) == 0 {
		return nil, false
	}
	tasks, err := taskStore.LoadWorkflowTasksV0(ctx, run.RunID, taskRefs)
	if err != nil || len(tasks) == 0 {
		return nil, false
	}
	return tasks, true
}

func codexStackResidentCouncilTaskRefsFromRunV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []string {
	refs := []string{}
	for _, taskRef := range compactStringsV0(run.Tasks) {
		if strings.HasPrefix(taskRef, codexStackResidentCouncilTaskPrefixV0) {
			refs = append(refs, taskRef)
		}
	}
	return refs
}

func codexStackResidentCouncilHasPendingBrainstormWorkV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) bool {
	for _, task := range tasks {
		if codexStackResidentCouncilTaskRoleV0(task) == "v" {
			continue
		}
		if !codexStackResidentCouncilTaskDoneV0(run, task.TaskID) {
			return true
		}
	}
	return false
}

func codexStackResidentCouncilReadyForVotePhaseV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) bool {
	var proposals int
	var critiques int
	var votes int
	for _, task := range tasks {
		switch codexStackResidentCouncilTaskRoleV0(task) {
		case "p":
			proposals++
			if !codexStackResidentCouncilTaskDoneV0(run, task.TaskID) {
				return false
			}
		case "c":
			critiques++
			if !codexStackResidentCouncilTaskDoneV0(run, task.TaskID) {
				return false
			}
		case "v":
			votes++
		}
	}
	return proposals > 0 && critiques > 0 && votes > 0
}

func codexStackResidentCouncilTaskDoneV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
) bool {
	taskRef = strings.TrimSpace(taskRef)
	return stringInSetV0(run.DeliveredTasks, taskRef) ||
		stringInSetV0(run.ClosedTasks, taskRef)
}

func codexStackResidentCouncilTaskRoleV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) string {
	taskRef := strings.TrimSpace(task.TaskID)
	switch {
	case strings.HasPrefix(taskRef, codexStackResidentCouncilProposalTaskPrefixV0):
		return "p"
	case strings.HasPrefix(taskRef, codexStackResidentCouncilCritiqueTaskPrefixV0):
		return "c"
	case strings.HasPrefix(taskRef, codexStackResidentCouncilVoteTaskPrefixV0):
		return "v"
	}
	if task.PhaseID == orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0 {
		return "v"
	}
	return ""
}

func lastCompactStringV0(values []string) string {
	compact := compactStringsV0(values)
	if len(compact) == 0 {
		return ""
	}
	return compact[len(compact)-1]
}
