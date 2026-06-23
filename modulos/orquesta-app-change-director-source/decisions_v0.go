package orquestaappchangedirectorsource

import (
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func buildAppChangeDecisionsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	record orquestaappchange.AppChangeRecordV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	request := record.Request
	refs := appChangeRefsV0(request.ChangeRef)
	runRef := run.RunID
	current := run.CurrentPhase
	if current == "" {
		current = orquestacoreworkflow.OrchestrationPhaseProgramacionV0
	}
	if current == orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		decisionRef, _ := appChangeBasisDecisionRefV0(run, refs)
		decisions := []orquestadirectoragent.DirectorAgentDecisionV0{
			appChangeAnswerDecisionV0(runRef, current, refs),
			appChangeContractDecisionV0(runRef, current, decisionRef, request, refs),
		}
		decisions = append(decisions, appChangeMicrotaskDecisionsV0(runRef, current, request, refs, run)...)
		return decisions
	}
	if current == orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0 {
		decisions := []orquestadirectoragent.DirectorAgentDecisionV0{
			appChangeAnswerDecisionV0(runRef, current, refs),
		}
		decisionRef, hasDecision := appChangeBasisDecisionRefV0(run, refs)
		if !hasDecision {
			decisionRef = refs.AnswerRef
			hasDecision = true
		}
		contractReady := stringInSetV0(run.FunctionContracts, refs.ContractRef)
		if !contractReady && hasDecision {
			decisions = append(decisions, appChangeContractDecisionV0(
				runRef,
				orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
				decisionRef,
				request,
				refs,
			))
			contractReady = true
		}
		taskRefs := appChangeMicrotaskRefsV0(request, refs)
		taskReady := appChangeAllTasksReadyV0(run, taskRefs)
		if !taskReady && contractReady {
			decisions = append(decisions, appChangeMicrotaskDecisionsV0(
				runRef,
				orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
				request,
				refs,
				run,
			)...)
			taskReady = appChangeAllTasksPlannedByDecisionsV0(run, taskRefs, decisions)
		}
		if taskReady {
			decisions = append(decisions, appChangeOpenPhaseV0(
				runRef,
				orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
				orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
				"open-program",
				refs,
			))
		}
		return decisions
	}

	decisions := []orquestadirectoragent.DirectorAgentDecisionV0{
		appChangeAnswerDecisionV0(runRef, current, refs),
		appChangeOpenPhaseV0(runRef, current, orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0, "open-vote", refs),
		appChangeVoteDecisionV0(runRef, refs),
		appChangeAcceptDecisionV0(runRef, refs),
		appChangeOpenPhaseV0(runRef, orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0, orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0, "open-plan", refs),
		appChangeContractDecisionV0(runRef, orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0, refs.DecisionRef, request, refs),
	}
	decisions = append(decisions, appChangeMicrotaskDecisionsV0(
		runRef,
		orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		request,
		refs,
		run,
	)...)
	decisions = append(decisions,
		appChangeOpenPhaseV0(runRef, orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0, orquestacoreworkflow.OrchestrationPhaseProgramacionV0, "open-program", refs),
	)
	return decisions
}

func appChangeAnswerDecisionV0(
	runRef string,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
	refs appChangeRefSetV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	decision := appChangeDecisionV0(runRef, phase, "answer", orquestadirectoragent.DirectorAgentCommandAnswerQuestionV0, refs)
	decision.AnswerQuestion = &orquestadirectoragent.DirectorAgentAnswerQuestionCommandV0{
		AnswerID:     refs.AnswerRef,
		QuestionID:   refs.QuestionRef,
		Decision:     orquestadirectoragent.DirectorAgentAnswerReplanV0,
		Summary:      "Aceptar replanificacion compacta.",
		EvidenceRefs: []string{refs.EvidenceRef},
	}
	return decision
}

func appChangeVoteDecisionV0(runRef string, refs appChangeRefSetV0) orquestadirectoragent.DirectorAgentDecisionV0 {
	decision := appChangeDecisionV0(runRef, orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0, "vote", orquestadirectoragent.DirectorAgentCommandRequestVoteV0, refs)
	decision.RequestVote = &orquestadirectoragent.DirectorAgentVoteCommandV0{
		VoteRequestID:              refs.VoteRef,
		PhaseID:                    string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
		DecisionTopicRef:           refs.TopicRef,
		BrainstormRef:              refs.BrainRef,
		Summary:                    "Elegir replanificacion compacta.",
		MinimumRecommendedCapacity: orquestadirectoragent.DirectorAgentCapacityHighV0,
		EvidenceRefs:               []string{refs.EvidenceRef},
	}
	return decision
}

func appChangeAcceptDecisionV0(runRef string, refs appChangeRefSetV0) orquestadirectoragent.DirectorAgentDecisionV0 {
	decision := appChangeDecisionV0(runRef, orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0, "accept", orquestadirectoragent.DirectorAgentCommandAcceptDecisionV0, refs)
	decision.AcceptDecision = &orquestadirectoragent.DirectorAgentAcceptDecisionCommandV0{
		DecisionRef:       refs.DecisionRef,
		PhaseID:           string(orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0),
		VoteRef:           refs.VoteRef,
		AcceptedOptionRef: refs.OptionRef,
		Summary:           "Aplicar cambio como microtarea independiente.",
		EvidenceRefs:      []string{refs.EvidenceRef},
	}
	return decision
}

func appChangeContractDecisionV0(
	runRef string,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
	decisionRef string,
	request orquestaappchange.AppChangeRequestV0,
	refs appChangeRefSetV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	decision := appChangeDecisionV0(runRef, phase, "contract", orquestadirectoragent.DirectorAgentCommandPublishContractV0, refs)
	decision.PublishContract = &orquestadirectoragent.DirectorAgentPublishContractCommandV0{
		ContractRef:   refs.ContractRef,
		PhaseID:       string(phase),
		DecisionRef:   decisionRef,
		Summary:       appChangeContractSummaryV0(request),
		FunctionNames: appChangeFunctionNamesV0(request),
		EvidenceRefs:  []string{refs.EvidenceRef},
	}
	return decision
}

func appChangeMicrotaskDecisionsV0(
	runRef string,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
	request orquestaappchange.AppChangeRequestV0,
	refs appChangeRefSetV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	tasks := appChangeMicrotasksV0(runRef, request, refs)
	decisions := make([]orquestadirectoragent.DirectorAgentDecisionV0, 0, len(tasks))
	for _, task := range tasks {
		if stringInSetV0(run.Tasks, task.TaskID) {
			continue
		}
		decision := appChangeDecisionV0(runRef, phase, appChangeMicrotaskDecisionActionV0(task.TaskID, refs), orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0, refs)
		decision.CreateMicrotask = &orquestadirectoragent.DirectorAgentCreateMicrotaskCommandV0{
			Task: task,
		}
		decisions = append(decisions, decision)
	}
	return decisions
}

func appChangeMicrotasksV0(
	runRef string,
	request orquestaappchange.AppChangeRequestV0,
	refs appChangeRefSetV0,
) []orquestadirectoragent.DirectorAgentMicrotaskV0 {
	parent := appChangeParentMicrotaskV0(runRef, request, refs)
	if !appChangeRequiresOPESParentSixSubrolesV0(request) {
		return []orquestadirectoragent.DirectorAgentMicrotaskV0{parent}
	}
	subroles := appChangeOPESSubrolesV0()
	childRefs := make([]string, 0, len(subroles))
	for _, subrole := range subroles {
		childRefs = append(childRefs, appChangeSubroleTaskRefV0(refs, subrole.Ref))
	}
	parent.ChildTaskRefs = childRefs
	parent.MaxChildAgents = len(childRefs)
	parent.ContextRefs = compactAppChangeSourceRefsV0(append(parent.ContextRefs,
		"opes-padre-tema-6-subroles-v1",
		"opes-parent-task",
	))
	parent.CohortRef = appChangeOPESCohortRefV0(refs)
	parent.WaveRef = appChangeOPESWaveRefV0(refs)

	tasks := []orquestadirectoragent.DirectorAgentMicrotaskV0{parent}
	for _, subrole := range subroles {
		tasks = append(tasks, appChangeOPESSubroleMicrotaskV0(parent, request, refs, subrole))
	}
	return tasks
}

func appChangeParentMicrotaskV0(
	runRef string,
	request orquestaappchange.AppChangeRequestV0,
	refs appChangeRefSetV0,
) orquestadirectoragent.DirectorAgentMicrotaskV0 {
	return orquestadirectoragent.DirectorAgentMicrotaskV0{
		SchemaVersion:      orquestadirectoragent.DirectorAgentMicrotaskSchemaVersionV0,
		TaskID:             refs.TaskRef,
		RunID:              runRef,
		PhaseID:            string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0),
		WorkProfileKind:    appChangeTaskWorkProfileKindV0(request),
		Title:              appChangeTaskTitleV0(request),
		Summary:            appChangeTaskSummaryV0(request),
		WriteSet:           appChangeTaskWriteSetV0(request),
		AcceptanceCriteria: appChangeTaskCriteriaV0(request),
		RequiredTests:      appChangeTaskRequiredTestsV0(request),
		ContextRefs:        appChangeTaskContextRefsV0(request),
		FunctionContractRefs: []orquestadirectoragent.DirectorAgentFunctionContractRefV0{{
			ContractRef:  refs.ContractRef,
			FunctionName: appChangeFunctionNamesV0(request)[0],
		}},
	}
}

func appChangeOPESSubroleMicrotaskV0(
	parent orquestadirectoragent.DirectorAgentMicrotaskV0,
	request orquestaappchange.AppChangeRequestV0,
	refs appChangeRefSetV0,
	subrole appChangeOPESSubroleV0,
) orquestadirectoragent.DirectorAgentMicrotaskV0 {
	child := parent
	child.TaskID = appChangeSubroleTaskRefV0(refs, subrole.Ref)
	child.Title = subrole.Title + ": " + appChangeTaskTitleV0(request)
	child.Summary = subrole.Summary
	child.WriteSet = appChangeSubroleWriteSetV0(parent.WriteSet, subrole.Ref)
	child.AcceptanceCriteria = compactAppChangeTaskCriteriaV0(append(
		[]string{subrole.Criteria},
		appChangeExternalWorkCriteriaV0(request)...,
	))
	child.RequiredTests = compactAppChangeTaskRequiredTestsV0(append(
		[]string{subrole.RequiredTest},
		appChangeTaskRequiredTestsV0(request)...,
	))
	child.ContextRefs = compactAppChangeSourceRefsV0(append(
		parent.ContextRefs,
		"opes-subrole-"+subrole.Ref,
		subrole.SkillRef,
	))
	child.ParentTaskRef = parent.TaskID
	child.CohortRef = appChangeOPESCohortRefV0(refs)
	child.WaveRef = appChangeOPESWaveRefV0(refs)
	child.DelegationDepth = parent.DelegationDepth + 1
	child.MaxChildAgents = 0
	child.ChildTaskRefs = nil
	return child
}

func appChangeMicrotaskRefsV0(
	request orquestaappchange.AppChangeRequestV0,
	refs appChangeRefSetV0,
) []string {
	if !appChangeRequiresOPESParentSixSubrolesV0(request) {
		return []string{refs.TaskRef}
	}
	taskRefs := []string{refs.TaskRef}
	for _, subrole := range appChangeOPESSubrolesV0() {
		taskRefs = append(taskRefs, appChangeSubroleTaskRefV0(refs, subrole.Ref))
	}
	return taskRefs
}

func appChangeAllTasksReadyV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRefs []string,
) bool {
	for _, taskRef := range taskRefs {
		if !stringInSetV0(run.Tasks, taskRef) {
			return false
		}
	}
	return len(taskRefs) > 0
}

func appChangeAllTasksPlannedByDecisionsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRefs []string,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) bool {
	planned := map[string]bool{}
	for _, taskRef := range run.Tasks {
		planned[taskRef] = true
	}
	for _, decision := range decisions {
		if decision.CreateMicrotask == nil {
			continue
		}
		planned[decision.CreateMicrotask.Task.TaskID] = true
	}
	for _, taskRef := range taskRefs {
		if !planned[taskRef] {
			return false
		}
	}
	return len(taskRefs) > 0
}

func appChangeMicrotaskDecisionActionV0(
	taskRef string,
	refs appChangeRefSetV0,
) string {
	if taskRef == refs.TaskRef {
		return "task"
	}
	return "task-" + strings.TrimPrefix(taskRef, refs.TaskRef+"-")
}

func appChangeTaskContextRefsV0(
	request orquestaappchange.AppChangeRequestV0,
) []string {
	return compactAppChangeSourceRefsV0(request.MetadataRefs)
}

func appChangeBasisDecisionRefV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	refs appChangeRefSetV0,
) (string, bool) {
	for i := len(run.Decisions) - 1; i >= 0; i-- {
		if run.Decisions[i] != "" {
			return run.Decisions[i], true
		}
	}
	for i := len(run.DirectorAnswers) - 1; i >= 0; i-- {
		if run.DirectorAnswers[i] != "" {
			return run.DirectorAnswers[i], true
		}
	}
	return refs.AnswerRef, false
}
