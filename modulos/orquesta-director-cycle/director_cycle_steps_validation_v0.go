package orquestadirectorcycle

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func validateDirectorCycleStepsInputV0(
	ctx context.Context,
	input DirectorCycleStepsInputV0,
) error {
	if ctx == nil {
		return cycleStepsErrorV0(input, ErrDirectorCycleStepsInvalidoV0, "context requerido", "context", false, nil)
	}
	if input.MaxSteps < 1 {
		return cycleStepsErrorV0(input, ErrDirectorCycleStepsInvalidoV0, "max_steps debe ser mayor que cero", "max_steps", false, nil)
	}
	if input.MaxSteps > DirectorCycleStepsMaxStepsLimitV0 {
		return cycleStepsErrorV0(input, ErrDirectorCycleStepsInvalidoV0, "max_steps supera el limite", "max_steps", false, nil)
	}
	if err := validateDirectorCycleStepInputV0(ctx, input.InitialStep); err != nil {
		return cycleStepsErrorV0(input, ErrDirectorCycleStepsInvalidoV0, "initial_step invalido", "initial_step", false, cycleStepsIssuesFromStepErrorV0(err))
	}
	return nil
}

func validateDirectorCycleStepSnapshotV0(
	input DirectorCycleStepsInputV0,
	stepNumber int,
	snapshot DirectorCycleStepSnapshotV0,
) error {
	fields := map[string]string{
		"snapshot.cycle_ref":   snapshot.CycleRef,
		"snapshot.tick_ref":    snapshot.TickRef,
		"snapshot.occurred_at": snapshot.OccurredAt,
		"snapshot.run.run_id":  snapshot.Run.RunID,
	}
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			return cycleStepsErrorV0(input, ErrDirectorCycleStepsSnapshotV0, "snapshot incompleto", field, false, nil)
		}
	}
	if strings.TrimSpace(snapshot.Run.RunID) != input.InitialStep.RunRef {
		return cycleStepsErrorV0(input, ErrDirectorCycleStepsSnapshotV0, "snapshot de otro run", "snapshot.run.run_id", false, nil)
	}
	if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(snapshot.Run); len(issues) > 0 {
		return cycleStepsErrorV0(input, ErrDirectorCycleStepsSnapshotV0, "snapshot run invalido", "snapshot.run."+issues[0].Field, false, nil)
	}
	if stepNumber < 2 {
		return cycleStepsErrorV0(input, ErrDirectorCycleStepsSnapshotV0, "step_number invalido", "step_number", false, nil)
	}
	return nil
}

func cycleStepsIssuesFromStepErrorV0(err error) []DirectorCycleStepsIssueV0 {
	stepErr, ok := err.(DirectorCycleStepErrorV0)
	if !ok {
		return nil
	}
	return cycleStepsIssuesFromStepV0(stepErr.Issues)
}
