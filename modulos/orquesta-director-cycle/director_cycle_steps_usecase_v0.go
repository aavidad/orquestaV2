package orquestadirectorcycle

import "context"

func ExecuteDirectorCycleStepsV0(
	ctx context.Context,
	input DirectorCycleStepsInputV0,
) (DirectorCycleStepsResultV0, error) {
	input = normalizeDirectorCycleStepsInputV0(input)
	result := newDirectorCycleStepsResultV0(input)
	preparedInput, err := prepareDirectorCycleStepsInitialInputV0(ctx, input)
	if err != nil {
		result.StopReason = DirectorCycleStepsStopErrorV0
		return resultWithCycleStepsErrorV0(result, err)
	}
	input = preparedInput
	result = newDirectorCycleStepsResultV0(input)
	if err := validateDirectorCycleStepsInputV0(ctx, input); err != nil {
		result.StopReason = DirectorCycleStepsStopErrorV0
		return resultWithCycleStepsErrorV0(result, err)
	}
	stepInput := input.InitialStep
	for stepNumber := 1; stepNumber <= input.MaxSteps; stepNumber++ {
		if stepNumber > 1 {
			nextInput, err := loadNextDirectorCycleStepInputV0(ctx, input, result.LastStepResult, stepNumber)
			if err != nil {
				result.StopReason = DirectorCycleStepsStopErrorV0
				return resultWithCycleStepsErrorV0(result, err)
			}
			stepInput = nextInput
		}
		stepResult, err := ExecuteDirectorCycleStepV0(ctx, stepInput)
		result = appendCycleStepsStepResultV0(result, stepResult)
		if err != nil {
			result.StopReason = DirectorCycleStepsStopErrorV0
			return resultWithCycleStepsErrorV0(result, err)
		}
		if stopReason := cycleStepsStopReasonForStepV0(stepNumber, input.MaxSteps, stepResult); stopReason != "" {
			result.StopReason = stopReason
			result.MaxStepsReached = stopReason == DirectorCycleStepsStopMaxStepsV0
			return result, nil
		}
	}
	result.StopReason = DirectorCycleStepsStopMaxStepsV0
	result.MaxStepsReached = true
	return result, nil
}

func prepareDirectorCycleStepsInitialInputV0(
	ctx context.Context,
	input DirectorCycleStepsInputV0,
) (DirectorCycleStepsInputV0, error) {
	if !isNilCycleStepPortV0(input.SnapshotPort) || isNilCycleStepPortV0(input.RunSnapshot) {
		return input, nil
	}
	run, err := input.RunSnapshot.LoadDirectorCycleRunSnapshotV0(ctx, input.InitialStep.RunRef)
	if err != nil {
		return input, cycleStepsErrorV0(
			input,
			ErrDirectorCycleStepsSnapshotV0,
			"run_snapshot fallo",
			"run_snapshot",
			true,
			nil,
		)
	}
	input.InitialStep.Run = run
	return normalizeDirectorCycleStepsInputV0(input), nil
}

func loadNextDirectorCycleStepInputV0(
	ctx context.Context,
	input DirectorCycleStepsInputV0,
	previous DirectorCycleStepResultV0,
	stepNumber int,
) (DirectorCycleStepInputV0, error) {
	if isNilCycleStepPortV0(input.SnapshotPort) {
		if !isNilCycleStepPortV0(input.RunSnapshot) {
			return loadNextDirectorCycleStepFromRunSnapshotV0(ctx, input, stepNumber)
		}
		return DirectorCycleStepInputV0{}, cycleStepsErrorV0(
			input,
			ErrDirectorCycleStepsSnapshotV0,
			"snapshot_port requerido para continuar",
			"snapshot_port",
			true,
			nil,
		)
	}
	snapshot, err := input.SnapshotPort.LoadDirectorCycleStepSnapshotV0(ctx, DirectorCycleStepSnapshotRequestV0{
		RunRef:             input.InitialStep.RunRef,
		StepNumber:         stepNumber,
		MaxSteps:           input.MaxSteps,
		PreviousStepResult: previous,
		CorrelationID:      input.CorrelationID,
		EvidenceRefs:       append([]string(nil), input.EvidenceRefs...),
	})
	if err != nil {
		return DirectorCycleStepInputV0{}, cycleStepsErrorV0(
			input,
			ErrDirectorCycleStepsSnapshotV0,
			"snapshot_port fallo",
			"snapshot_port",
			true,
			nil,
		)
	}
	if err := validateDirectorCycleStepSnapshotV0(input, stepNumber, snapshot); err != nil {
		return DirectorCycleStepInputV0{}, err
	}
	return cycleStepsStepInputFromSnapshotV0(input.InitialStep, snapshot), nil
}

func loadNextDirectorCycleStepFromRunSnapshotV0(
	ctx context.Context,
	input DirectorCycleStepsInputV0,
	stepNumber int,
) (DirectorCycleStepInputV0, error) {
	run, err := input.RunSnapshot.LoadDirectorCycleRunSnapshotV0(ctx, input.InitialStep.RunRef)
	if err != nil {
		return DirectorCycleStepInputV0{}, cycleStepsErrorV0(
			input,
			ErrDirectorCycleStepsSnapshotV0,
			"run_snapshot fallo",
			"run_snapshot",
			true,
			nil,
		)
	}
	snapshot := cycleStepsSnapshotFromRunSnapshotV0(input, run, stepNumber)
	if err := validateDirectorCycleStepSnapshotV0(input, stepNumber, snapshot); err != nil {
		return DirectorCycleStepInputV0{}, err
	}
	return cycleStepsStepInputFromSnapshotV0(input.InitialStep, snapshot), nil
}
