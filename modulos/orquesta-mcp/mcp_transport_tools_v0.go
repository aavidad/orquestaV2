package orquestamcp

import operator "orquesta/modulos/orquesta-operator-mcp"

func MCPTransportToolsV0(bindings MCPTransportBindingsV0) []MCPTransportToolEnvelopeV0 {
	descriptors := newMCPTransportToolDescriptorsV0()
	tools := []MCPTransportToolEnvelopeV0{
		mcpTransportToolEnvelopeV0(descriptors.nueva.Name, descriptors.nueva.Version, descriptors.nueva.ResourceURI, descriptors.nueva.InputSchema, descriptors.nueva.Output, mcpNuevaAppTransportHandlerV0(bindings.NuevaApp)),
		mcpTransportToolEnvelopeV0(descriptors.director.Name, descriptors.director.Version, descriptors.director.ResourceURI, descriptors.director.InputSchema, descriptors.director.Output, mcpArrancarDirectorAppTransportHandlerV0(bindings.ArrancarDirector)),
		mcpTransportToolEnvelopeV0(descriptors.observeDirectorGoal.Name, descriptors.observeDirectorGoal.Version, descriptors.observeDirectorGoal.ResourceURI, descriptors.observeDirectorGoal.InputSchema, descriptors.observeDirectorGoal.Output, mcpObserveAppDirectorGoalTransportHandlerV0(bindings.ObserveDirectorGoal)),
		mcpTransportToolEnvelopeV0(descriptors.change.Name, descriptors.change.Version, descriptors.change.ResourceURI, descriptors.change.InputSchema, descriptors.change.Output, mcpRequestAppChangeTransportHandlerV0(bindings.RequestAppChange)),
		mcpTransportToolEnvelopeV0(descriptors.decision.Name, descriptors.decision.Version, descriptors.decision.ResourceURI, descriptors.decision.InputSchema, descriptors.decision.Output, mcpDirectorAgentDecisionTransportHandlerV0(bindings.DirectorDecision)),
		mcpTransportToolEnvelopeV0(descriptors.supervisorBriefing.Name, descriptors.supervisorBriefing.Version, descriptors.supervisorBriefing.ResourceURI, descriptors.supervisorBriefing.InputSchema, descriptors.supervisorBriefing.Output, mcpDirectorSupervisorBriefingTransportHandlerV0(bindings.DirectorSupervisorBriefing)),
		mcpTransportToolEnvelopeV0(descriptors.stats.Name, descriptors.stats.Version, descriptors.stats.ResourceURI, descriptors.stats.InputSchema, descriptors.stats.Output, mcpDirectorStatsTransportHandlerV0(bindings.DirectorStats)),
		mcpTransportToolEnvelopeV0(descriptors.preparar.Name, descriptors.preparar.Version, descriptors.preparar.ResourceURI, descriptors.preparar.InputSchema, descriptors.preparar.Output, mcpPrepararOrquestacionAppTransportHandlerV0),
		mcpTransportToolEnvelopeV0(descriptors.ejecutar.Name, descriptors.ejecutar.Version, descriptors.ejecutar.ResourceURI, descriptors.ejecutar.InputSchema, descriptors.ejecutar.Output, mcpEjecutarOrquestacionAppTransportHandlerV0(bindings.EjecutarOrquestacion)),
		mcpTransportToolEnvelopeV0(descriptors.autoprogramming.Name, descriptors.autoprogramming.Version, descriptors.autoprogramming.ResourceURI, descriptors.autoprogramming.InputSchema, descriptors.autoprogramming.Output, mcpAutoprogrammingValidateRequestTransportHandlerV0(MCPAutoprogrammingValidateRequestToolExecutorV0{})),
		mcpTransportToolEnvelopeV0(descriptors.humanDirectorWork.Name, descriptors.humanDirectorWork.Version, descriptors.humanDirectorWork.ResourceURI, descriptors.humanDirectorWork.InputSchema, descriptors.humanDirectorWork.Output, mcpHumanDirectorWorkReviewPlanTransportHandlerV0(MCPHumanDirectorWorkReviewPlanToolExecutorV0{OperatorQuery: mcpOperatorQueryBindingV0(bindings)})),
		mcpTransportToolEnvelopeV0(descriptors.selfImprovement.Name, descriptors.selfImprovement.Version, descriptors.selfImprovement.ResourceURI, descriptors.selfImprovement.InputSchema, descriptors.selfImprovement.Output, mcpAutoprogrammingSelfImprovementTransportHandlerV0(NewMCPAutoprogrammingSelfImprovementToolExecutorV0(bindings.AutoprogrammingPrepareRun))),
		mcpTransportToolEnvelopeV0(descriptors.autoprogrammingPrepare.Name, descriptors.autoprogrammingPrepare.Version, descriptors.autoprogrammingPrepare.ResourceURI, descriptors.autoprogrammingPrepare.InputSchema, descriptors.autoprogrammingPrepare.Output, mcpAutoprogrammingPrepareRunTransportHandlerV0(bindings.AutoprogrammingPrepareRun)),
		mcpTransportToolEnvelopeV0(descriptors.autoprogrammingGoal.Name, descriptors.autoprogrammingGoal.Version, descriptors.autoprogrammingGoal.ResourceURI, descriptors.autoprogrammingGoal.InputSchema, descriptors.autoprogrammingGoal.Output, mcpAutoprogrammingObserveGoalTransportHandlerV0(bindings.AutoprogrammingObserveGoal)),
		mcpTransportToolEnvelopeV0(descriptors.autoprogrammingGoals.Name, descriptors.autoprogrammingGoals.Version, descriptors.autoprogrammingGoals.ResourceURI, descriptors.autoprogrammingGoals.InputSchema, descriptors.autoprogrammingGoals.Output, mcpAutoprogrammingObserveActiveGoalsTransportHandlerV0(bindings.AutoprogrammingObserveActiveGoals)),
		mcpTransportToolEnvelopeV0(descriptors.autoprogrammingStatus.Name, descriptors.autoprogrammingStatus.Version, descriptors.autoprogrammingStatus.ResourceURI, descriptors.autoprogrammingStatus.InputSchema, descriptors.autoprogrammingStatus.Output, mcpAutoprogrammingStatusTransportHandlerV0(autoprogrammingStatusExecutorFromBindingsV0(bindings))),
		mcpTransportToolEnvelopeV0(descriptors.autoprogrammingSupervise.Name, descriptors.autoprogrammingSupervise.Version, descriptors.autoprogrammingSupervise.ResourceURI, descriptors.autoprogrammingSupervise.InputSchema, descriptors.autoprogrammingSupervise.Output, mcpAutoprogrammingSuperviseTransportHandlerV0(bindings.RunSupervisor)),
		mcpTransportToolEnvelopeV0(descriptors.bootstrap.Name, descriptors.bootstrap.Version, descriptors.bootstrap.ResourceURI, descriptors.bootstrap.InputSchema, descriptors.bootstrap.Output, mcpBootstrapTransportHandlerV0),
		mcpTransportToolEnvelopeV0(descriptors.workflow.Name, descriptors.workflow.Version, descriptors.workflow.ResourceURI, descriptors.workflow.InputSchema, descriptors.workflow.Output, mcpCoreWorkflowTransportHandlerV0),
		mcpTransportToolEnvelopeV0(descriptors.runControl.Name, descriptors.runControl.Version, descriptors.runControl.ResourceURI, descriptors.runControl.InputSchema, descriptors.runControl.Output, mcpRunControlTransportHandlerV0(bindings.RunControl)),
		mcpTransportToolEnvelopeV0(descriptors.runtimeModels.Name, descriptors.runtimeModels.Version, descriptors.runtimeModels.ResourceURI, descriptors.runtimeModels.InputSchema, descriptors.runtimeModels.Output, mcpRuntimeModelsTransportHandlerV0(bindings.RuntimeModels)),
		mcpTransportToolEnvelopeV0(descriptors.runQueue.Name, descriptors.runQueue.Version, descriptors.runQueue.ResourceURI, descriptors.runQueue.InputSchema, descriptors.runQueue.Output, mcpRunQueuePriorityTransportHandlerV0(bindings.RunQueuePriority)),
		mcpTransportToolEnvelopeV0(descriptors.runSupervisor.Name, descriptors.runSupervisor.Version, descriptors.runSupervisor.ResourceURI, descriptors.runSupervisor.InputSchema, descriptors.runSupervisor.Output, mcpRunSupervisorTransportHandlerV0(bindings.RunSupervisor)),
		mcpTransportToolEnvelopeV0(descriptors.workspaceTimeline.Name, descriptors.workspaceTimeline.Version, descriptors.workspaceTimeline.ResourceURI, descriptors.workspaceTimeline.InputSchema, descriptors.workspaceTimeline.Output, mcpWorkspaceTimelineTransportHandlerV0(bindings.WorkspaceTimeline)),
		mcpTransportToolEnvelopeV0(descriptors.serverShutdown.Name, descriptors.serverShutdown.Version, descriptors.serverShutdown.ResourceURI, descriptors.serverShutdown.InputSchema, descriptors.serverShutdown.Output, mcpServerShutdownTransportHandlerV0(bindings.ServerShutdown)),
		mcpTransportToolEnvelopeV0(descriptors.domainWork.Name, descriptors.domainWork.Version, descriptors.domainWork.ResourceURI, descriptors.domainWork.InputSchema, descriptors.domainWork.Output, mcpDomainWorkTransportHandlerV0(bindings.DomainWork)),
		mcpTransportToolEnvelopeV0(descriptors.externalWorkDryRun.Name, descriptors.externalWorkDryRun.Version, descriptors.externalWorkDryRun.ResourceURI, descriptors.externalWorkDryRun.InputSchema, descriptors.externalWorkDryRun.Output, mcpExternalWorkDryRunTransportHandlerV0(bindings.ExternalWorkDryRun)),
		mcpTransportToolEnvelopeV0(descriptors.externalWorkRun.Name, descriptors.externalWorkRun.Version, descriptors.externalWorkRun.ResourceURI, descriptors.externalWorkRun.InputSchema, descriptors.externalWorkRun.Output, mcpExternalWorkRunTransportHandlerV0(bindings.ExternalWorkRun)),
		mcpTransportToolEnvelopeV0(descriptors.appVCS.Name, descriptors.appVCS.Version, descriptors.appVCS.ResourceURI, descriptors.appVCS.InputSchema, descriptors.appVCS.Output, mcpAppVCSTransportHandlerV0(bindings.AppVCS)),
		mcpOperatorTransportToolV0(operator.OperatorMCPStatusToolNameV0, mcpOperatorStatusBindingV0(bindings)),
		mcpOperatorTransportToolV0(operator.OperatorMCPBurstToolNameV0, mcpOperatorBurstBindingV0(bindings)),
		mcpOperatorTransportToolV0(operator.OperatorMCPOutboxToolNameV0, mcpOperatorOutboxBindingV0(bindings)),
		mcpOperatorTransportToolV0(operator.OperatorMCPDirectedQueryToolV0, mcpOperatorQueryBindingV0(bindings)),
	}
	tools = append(tools, mcpOperatorFriendlyTransportToolsV0(bindings)...)
	return applyMCPTransportExecutionProfilesV0(tools)
}

func applyMCPTransportExecutionProfilesV0(
	tools []MCPTransportToolEnvelopeV0,
) []MCPTransportToolEnvelopeV0 {
	for idx := range tools {
		profile := MCPTransportExecutionProfileControlPlaneMutationV0
		switch tools[idx].Name {
		case MCPAutoprogrammingPrepareRunToolNameV0,
			MCPAutoprogrammingObserveGoalToolNameV0,
			MCPAutoprogrammingObserveActiveGoalsToolNameV0,
			MCPAutoprogrammingSelfImprovementToolNameV0,
			MCPAutoprogrammingSuperviseToolNameV0,
			MCPRuntimeModelsToolNameV0,
			MCPDomainWorkToolNameV0,
			MCPExternalWorkRunToolNameV0:
			profile = MCPTransportExecutionProfileAutoprogrammingLongV0
		case MCPWorkspaceTimelineToolNameV0,
			MCPDirectorStatsToolNameV0,
			MCPDirectorSupervisorBriefingToolNameV0,
			MCPAutoprogrammingStatusToolNameV0:
			profile = MCPTransportExecutionProfileDefaultToolV0
		case MCPExternalWorkDryRunToolNameV0:
			profile = MCPTransportExecutionProfileDefaultToolV0
		}
		tools[idx].ExecutionBudget = MCPTransportToolExecutionBudgetV0(profile)
	}
	return tools
}

func mcpOperatorStatusBindingV0(bindings MCPTransportBindingsV0) operator.OperatorMCPStatusPortV0 {
	if bindings.OperatorStatus != nil {
		return bindings.OperatorStatus
	}
	return bindings.OperatorConnector
}

func mcpOperatorBurstBindingV0(bindings MCPTransportBindingsV0) operator.OperatorMCPBurstPortV0 {
	if bindings.OperatorBurst != nil {
		return bindings.OperatorBurst
	}
	return bindings.OperatorConnector
}

func mcpOperatorOutboxBindingV0(bindings MCPTransportBindingsV0) operator.OperatorMCPOutboxPortV0 {
	if bindings.OperatorOutbox != nil {
		return bindings.OperatorOutbox
	}
	return bindings.OperatorConnector
}

func mcpOperatorQueryBindingV0(bindings MCPTransportBindingsV0) operator.OperatorMCPDirectedQueryPortV0 {
	if bindings.OperatorQuery != nil {
		return bindings.OperatorQuery
	}
	return bindings.OperatorConnector
}

func autoprogrammingStatusExecutorFromBindingsV0(
	bindings MCPTransportBindingsV0,
) MCPTransportAutoprogrammingStatusExecutorV0 {
	if bindings.RunQueuePriority == nil && bindings.DirectorStats == nil {
		return nil
	}
	return MCPAutoprogrammingStatusToolExecutorV0{
		Queue:                        bindings.RunQueuePriority,
		Stats:                        bindings.DirectorStats,
		GoalStateStore:               bindings.AutoprogrammingGoalStates,
		AllowLegacySupervisorActions: bindings.AllowLegacyAutoprogrammingSupervisorActions,
	}
}
