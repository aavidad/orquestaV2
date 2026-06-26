package orquestamcp

type mcpTransportToolDescriptorsV0 struct {
	nueva                    MCPNuevaAppToolDescriptorV0
	director                 MCPArrancarDirectorAppToolDescriptorV0
	observeDirectorGoal      MCPObserveAppDirectorGoalToolDescriptorV0
	change                   MCPRequestAppChangeToolDescriptorV0
	decision                 MCPDirectorAgentDecisionToolDescriptorV0
	supervisorBriefing       MCPDirectorSupervisorBriefingToolDescriptorV0
	stats                    MCPDirectorStatsToolDescriptorV0
	preparar                 MCPPrepararOrquestacionAppToolDescriptorV0
	ejecutar                 MCPEjecutarOrquestacionAppToolDescriptorV0
	autoprogramming          MCPAutoprogrammingValidateRequestToolDescriptorV0
	humanDirectorWork        MCPHumanDirectorWorkReviewPlanToolDescriptorV0
	selfImprovement          MCPAutoprogrammingSelfImprovementToolDescriptorV0
	autoprogrammingPrepare   MCPAutoprogrammingPrepareRunToolDescriptorV0
	autoprogrammingGoal      MCPAutoprogrammingObserveGoalToolDescriptorV0
	autoprogrammingStatus    MCPAutoprogrammingStatusToolDescriptorV0
	autoprogrammingSupervise MCPAutoprogrammingSuperviseToolDescriptorV0
	bootstrap                MCPBootstrapToolDescriptorV0
	workflow                 MCPCoreWorkflowCommandToolDescriptorV0
	runControl               MCPRunControlToolDescriptorV0
	runtimeModels            MCPRuntimeModelsToolDescriptorV0
	runQueue                 MCPRunQueuePriorityToolDescriptorV0
	runSupervisor            MCPRunSupervisorToolDescriptorV0
	workspaceTimeline        MCPWorkspaceTimelineToolDescriptorV0
	serverShutdown           MCPServerShutdownToolDescriptorV0
	domainWork               MCPDomainWorkToolDescriptorV0
	externalWorkRun          MCPExternalWorkRunToolDescriptorV0
	appVCS                   MCPAppVCSToolDescriptorV0
}

func newMCPTransportToolDescriptorsV0() mcpTransportToolDescriptorsV0 {
	return mcpTransportToolDescriptorsV0{
		nueva:                    MCPNuevaAppDescriptorV0(),
		director:                 MCPArrancarDirectorAppDescriptorV0(),
		observeDirectorGoal:      MCPObserveAppDirectorGoalDescriptorV0(),
		change:                   MCPRequestAppChangeDescriptorV0(),
		decision:                 MCPDirectorAgentDecisionDescriptorV0(),
		supervisorBriefing:       MCPDirectorSupervisorBriefingDescriptorV0(),
		stats:                    MCPDirectorStatsDescriptorV0(),
		preparar:                 MCPPrepararOrquestacionAppDescriptorV0(),
		ejecutar:                 MCPEjecutarOrquestacionAppDescriptorV0(),
		autoprogramming:          MCPAutoprogrammingValidateRequestDescriptorV0(),
		humanDirectorWork:        MCPHumanDirectorWorkReviewPlanDescriptorV0(),
		selfImprovement:          MCPAutoprogrammingSelfImprovementDescriptorV0(),
		autoprogrammingPrepare:   MCPAutoprogrammingPrepareRunDescriptorV0(),
		autoprogrammingGoal:      MCPAutoprogrammingObserveGoalDescriptorV0(),
		autoprogrammingStatus:    MCPAutoprogrammingStatusDescriptorV0(),
		autoprogrammingSupervise: MCPAutoprogrammingSuperviseDescriptorV0(),
		bootstrap:                MCPBootstrapToolDescriptorV0Value(),
		workflow:                 MCPCoreWorkflowCommandToolDescriptorV0Value(),
		runControl:               MCPRunControlDescriptorV0(),
		runtimeModels:            MCPRuntimeModelsDescriptorV0(),
		runQueue:                 MCPRunQueuePriorityDescriptorV0(),
		runSupervisor:            MCPRunSupervisorDescriptorV0(),
		workspaceTimeline:        MCPWorkspaceTimelineToolDescriptorV0Value(),
		serverShutdown:           MCPServerShutdownDescriptorV0(),
		domainWork:               MCPDomainWorkDescriptorV0(),
		externalWorkRun:          MCPExternalWorkRunDescriptorV0(),
		appVCS:                   MCPAppVCSDescriptorV0(),
	}
}
