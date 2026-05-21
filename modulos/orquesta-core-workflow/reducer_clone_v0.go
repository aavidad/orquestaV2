package orquestacoreworkflow

func cloneRunForReducerV0(run OrchestrationRunV0) OrchestrationRunV0 {
	next := run
	next.Phases = clonePhasesForReducerV0(run.Phases)
	next.Tasks = cloneStringsV0(run.Tasks)
	next.Brainstorms = cloneStringsV0(run.Brainstorms)
	next.Votes = cloneStringsV0(run.Votes)
	next.FunctionContracts = cloneStringsV0(run.FunctionContracts)
	next.Decisions = cloneStringsV0(run.Decisions)
	next.CapacityRequests = cloneStringsV0(run.CapacityRequests)
	next.CapacityDecisions = cloneStringsV0(run.CapacityDecisions)
	next.Agents = cloneStringsV0(run.Agents)
	next.AgentPhaseRefs = cloneStringsV0(run.AgentPhaseRefs)
	next.StartedAgents = cloneStringsV0(run.StartedAgents)
	next.FailedAgents = cloneStringsV0(run.FailedAgents)
	next.LostAgents = cloneStringsV0(run.LostAgents)
	next.StoppedAgents = cloneStringsV0(run.StoppedAgents)
	next.AgentStopRequests = cloneStringsV0(run.AgentStopRequests)
	next.ConfirmedStoppedAgents = cloneStringsV0(run.ConfirmedStoppedAgents)
	next.AgentAssessments = cloneStringsV0(run.AgentAssessments)
	next.AgentLeaseExpirations = cloneStringsV0(run.AgentLeaseExpirations)
	next.ConcurrencyGates = cloneStringsV0(run.ConcurrencyGates)
	next.QualityGates = cloneStringsV0(run.QualityGates)
	next.PhaseArtifacts = cloneStringsV0(run.PhaseArtifacts)
	next.Deliveries = cloneStringsV0(run.Deliveries)
	next.DeliveredTasks = cloneStringsV0(run.DeliveredTasks)
	next.DeliveredAgents = cloneStringsV0(run.DeliveredAgents)
	next.Reviews = cloneStringsV0(run.Reviews)
	next.ReviewResults = cloneStringsV0(run.ReviewResults)
	next.ReworkRequests = cloneStringsV0(run.ReworkRequests)
	next.ReplanDecisions = cloneStringsV0(run.ReplanDecisions)
	next.AcceptedReviews = cloneStringsV0(run.AcceptedReviews)
	next.ClosedTasks = cloneStringsV0(run.ClosedTasks)
	next.Validations = cloneStringsV0(run.Validations)
	next.Closures = cloneStringsV0(run.Closures)
	next.DirectorQuestions = cloneStringsV0(run.DirectorQuestions)
	next.DirectorAnswers = cloneStringsV0(run.DirectorAnswers)
	next.DirectorAnsweredQuestions = cloneStringsV0(run.DirectorAnsweredQuestions)
	next.Blockers = cloneStringsV0(run.Blockers)
	next.CommandEffects = cloneStringsV0(run.CommandEffects)
	return next
}

func clonePhasesForReducerV0(phases []OrchestrationPhaseV0) []OrchestrationPhaseV0 {
	if phases == nil {
		return nil
	}
	result := make([]OrchestrationPhaseV0, len(phases))
	for index, phase := range phases {
		result[index] = cloneOrchestrationPhaseV0(phase)
	}
	return result
}
