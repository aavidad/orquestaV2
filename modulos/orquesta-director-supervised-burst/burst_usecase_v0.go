package orquestadirectorsupervisedburst

import (
	"context"
	"errors"
	"fmt"
	"strings"

	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func RunDirectorSupervisedBurstV0(
	ctx context.Context,
	input DirectorSupervisedBurstInputV0,
) (DirectorSupervisedBurstResultV0, error) {
	input = normalizeDirectorSupervisedBurstInputV0(input)
	result := newDirectorSupervisedBurstResultV0(input)
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateDirectorSupervisedBurstInputV0(input); err != nil {
		var publicErr DirectorSupervisedBurstErrorV0
		if errors.As(err, &publicErr) {
			return resultWithBurstErrorV0(result, publicErr)
		}
		return result, err
	}
	return runDirectorSupervisedBurstLoopV0(ctx, input, result)
}

func runDirectorSupervisedBurstLoopV0(
	ctx context.Context,
	input DirectorSupervisedBurstInputV0,
	result DirectorSupervisedBurstResultV0,
) (DirectorSupervisedBurstResultV0, error) {
	var previousStep *orquestadirectorcycle.DirectorCycleStepResultV0
	var previousDecision *orquestadirectorsupervisor.DirectorSupervisorDecisionV0
	for stepNumber := 1; stepNumber <= input.MaxSteps; stepNumber++ {
		stepResult, decision, stepErr, err := runDirectorSupervisedBurstStepV0(
			ctx,
			input,
			stepNumber,
			previousStep,
			previousDecision,
		)
		if err.Code != "" {
			return resultWithBurstErrorV0(result, err)
		}
		decision = capBurstDecisionAtMaxStepsV0(input, stepNumber, decision)
		briefing, briefingErr := buildBurstBriefingV0(input, decision)
		if briefingErr.Code != "" {
			return resultWithBurstErrorV0(result, briefingErr)
		}
		errorCode := publicCycleStepErrorCodeV0(stepErr)
		result.Steps = append(result.Steps, burstStepResultV0(stepNumber, stepResult, decision, briefing, errorCode))
		result.ExecutedSteps = stepNumber
		result.FinalAction = decision.Action
		result.FinalBriefing = &briefing
		result.StopProjection = burstStopProjectionV0(result, decision)
		previousStep = &stepResult
		previousDecision = &decision
		if stepErr != nil {
			return resultWithBurstErrorV0(result, burstStepErrorV0(input, stepResult, stepErr))
		}
		if !decision.ShouldContinue {
			return result, nil
		}
	}
	return result, nil
}

func runDirectorSupervisedBurstStepV0(
	ctx context.Context,
	input DirectorSupervisedBurstInputV0,
	stepNumber int,
	previousStep *orquestadirectorcycle.DirectorCycleStepResultV0,
	previousDecision *orquestadirectorsupervisor.DirectorSupervisorDecisionV0,
) (
	orquestadirectorcycle.DirectorCycleStepResultV0,
	orquestadirectorsupervisor.DirectorSupervisorDecisionV0,
	error,
	DirectorSupervisedBurstErrorV0,
) {
	stepInput, err := input.StepInputBuilder.BuildDirectorCycleStepInputV0(
		ctx,
		burstStepRequestV0(input, stepNumber, previousStep, previousDecision),
	)
	if err != nil {
		return orquestadirectorcycle.DirectorCycleStepResultV0{},
			orquestadirectorsupervisor.DirectorSupervisorDecisionV0{},
			nil,
			burstErrorV0(input, ErrDirectorSupervisedBurstStepInputV0, fmt.Sprintf("%T: %v", err, err), "step_input_builder", true)
	}
	if err := validateBurstStepInputForRunV0(input, stepInput); err.Code != "" {
		return orquestadirectorcycle.DirectorCycleStepResultV0{},
			orquestadirectorsupervisor.DirectorSupervisorDecisionV0{},
			nil,
			err
	}
	stepResult, stepErr := input.StepExecutor.ExecuteDirectorCycleStepV0(ctx, stepInput)
	if err := validateBurstStepResultForRunV0(input, stepResult); err.Code != "" {
		return stepResult,
			orquestadirectorsupervisor.DirectorSupervisorDecisionV0{},
			stepErr,
			err
	}
	decision, decisionErr := input.Supervisor.DecideDirectorSupervisorNextActionV0(
		burstSupervisorInputV0(input, stepNumber, stepResult, publicCycleStepErrorCodeV0(stepErr)),
	)
	if decisionErr != nil {
		return stepResult,
			decision,
			stepErr,
			burstErrorV0(input, ErrDirectorSupervisedBurstSupervisorV0, "supervisor fallo", "supervisor", false)
	}
	return stepResult, decision, stepErr, DirectorSupervisedBurstErrorV0{}
}

func validateBurstStepInputForRunV0(
	input DirectorSupervisedBurstInputV0,
	stepInput orquestadirectorcycle.DirectorCycleStepInputV0,
) DirectorSupervisedBurstErrorV0 {
	if strings.TrimSpace(stepInput.RunRef) == input.RunRef {
		return DirectorSupervisedBurstErrorV0{}
	}
	return burstErrorV0(
		input,
		ErrDirectorSupervisedBurstStepInputV0,
		"step_input.run_ref no coincide con run_ref de la rafaga",
		"step_input.run_ref",
		true,
	)
}

func validateBurstStepResultForRunV0(
	input DirectorSupervisedBurstInputV0,
	stepResult orquestadirectorcycle.DirectorCycleStepResultV0,
) DirectorSupervisedBurstErrorV0 {
	resultRunRef := strings.TrimSpace(stepResult.RunRef)
	if resultRunRef == "" || resultRunRef == input.RunRef {
		return DirectorSupervisedBurstErrorV0{}
	}
	return burstErrorV0(
		input,
		ErrDirectorSupervisedBurstStepV0,
		"step.run_ref no coincide con run_ref de la rafaga",
		"step.run_ref",
		true,
	)
}

func capBurstDecisionAtMaxStepsV0(
	input DirectorSupervisedBurstInputV0,
	stepNumber int,
	decision orquestadirectorsupervisor.DirectorSupervisorDecisionV0,
) orquestadirectorsupervisor.DirectorSupervisorDecisionV0 {
	if stepNumber < input.MaxSteps || !decision.ShouldContinue {
		return decision
	}
	decision.Action = orquestadirectorsupervisor.DirectorSupervisorActionStopMaxStepsV0
	decision.ReasonCode = orquestadirectorsupervisor.DirectorSupervisorReasonMaxStepsV0
	decision.ShouldContinue = false
	decision.AutonomousRecommendation = orquestadirectorsupervisor.DirectorSupervisorAutonomousStopV0
	return decision
}
