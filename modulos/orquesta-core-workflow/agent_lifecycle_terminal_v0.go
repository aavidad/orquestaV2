package orquestacoreworkflow

func agentStartHasTerminalConflictV0(run OrchestrationRunV0, agentRequestID string) bool {
	return agentFailedAlreadyReflectedV0(run, agentRequestID) ||
		agentStopAlreadyReflectedV0(run, agentRequestID)
}

func agentFailureHasTerminalConflictV0(run OrchestrationRunV0, agentRequestID string) bool {
	return agentStartedAlreadyReflectedV0(run, agentRequestID) ||
		agentStopAlreadyReflectedV0(run, agentRequestID)
}
