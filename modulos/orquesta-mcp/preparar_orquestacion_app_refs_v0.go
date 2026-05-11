package orquestamcp

func neutralPrepareOrchestrationRunRefMCPV0(
	input MCPPrepararOrquestacionAppToolInputV0,
) string {
	return neutralAppDirectorRequestRefV0(
		"run",
		prepareOrchestrationAppNameMCPV0(input),
		firstNonEmptyMCPV0(input.RunRef, input.RequestID, input.CorrelationID, input.AppSpec.RequestID, input.AppSpec.SpecID),
	)
}

func neutralPrepareOrchestrationProjectRefMCPV0(
	input MCPPrepararOrquestacionAppToolInputV0,
) string {
	return neutralAppDirectorRequestRefV0(
		"project",
		prepareOrchestrationAppNameMCPV0(input),
		firstNonEmptyMCPV0(input.ProjectRef, input.RequestID, input.CorrelationID, input.AppSpec.RequestID, input.AppSpec.SpecID),
	)
}

func neutralPrepareOrchestrationCorrelationMCPV0(
	input MCPPrepararOrquestacionAppToolInputV0,
) string {
	return neutralAppDirectorRequestRefV0(
		"corr",
		prepareOrchestrationAppNameMCPV0(input),
		firstNonEmptyMCPV0(input.CorrelationID, input.RequestID, input.RunRef, input.AppSpec.RequestID, input.AppSpec.SpecID),
	)
}

func prepareOrchestrationAppNameMCPV0(
	input MCPPrepararOrquestacionAppToolInputV0,
) string {
	return firstNonEmptyMCPV0(
		input.AppSpec.App.Nombre,
		input.AppSpec.App.Slug,
		input.AppSpec.SpecID,
		"app",
	)
}
