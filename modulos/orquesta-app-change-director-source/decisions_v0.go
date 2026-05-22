package orquestaappchangedirectorsource

import (
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
		decisionRef, ok := appChangeBasisDecisionRefV0(run, refs)
		if ok {
			return []orquestadirectoragent.DirectorAgentDecisionV0{
				appChangeAnswerDecisionV0(runRef, current, refs),
				appChangeContractDecisionV0(runRef, current, decisionRef, request, refs),
				appChangeMicrotaskDecisionV0(runRef, current, request, refs),
			}
		}
	}

	return []orquestadirectoragent.DirectorAgentDecisionV0{
		appChangeAnswerDecisionV0(runRef, current, refs),
		appChangeOpenPhaseV0(runRef, current, orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0, "open-vote", refs),
		appChangeVoteDecisionV0(runRef, refs),
		appChangeAcceptDecisionV0(runRef, refs),
		appChangeOpenPhaseV0(runRef, orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0, orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0, "open-plan", refs),
		appChangeContractDecisionV0(runRef, orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0, refs.DecisionRef, request, refs),
		appChangeMicrotaskDecisionV0(runRef, orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0, request, refs),
		appChangeOpenPhaseV0(runRef, orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0, orquestacoreworkflow.OrchestrationPhaseProgramacionV0, "open-program", refs),
	}
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

func appChangeMicrotaskDecisionV0(
	runRef string,
	phase orquestacoreworkflow.OrchestrationPhaseIDV0,
	request orquestaappchange.AppChangeRequestV0,
	refs appChangeRefSetV0,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	decision := appChangeDecisionV0(runRef, phase, "task", orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0, refs)
	decision.CreateMicrotask = &orquestadirectoragent.DirectorAgentCreateMicrotaskCommandV0{
		Task: orquestadirectoragent.DirectorAgentMicrotaskV0{
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
		},
	}
	return decision
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
	return refs.DecisionRef, false
}
