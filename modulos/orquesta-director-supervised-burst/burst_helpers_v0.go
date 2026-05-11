package orquestadirectorsupervisedburst

import (
	"errors"
	"strings"

	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func normalizeDirectorSupervisedBurstInputV0(
	input DirectorSupervisedBurstInputV0,
) DirectorSupervisedBurstInputV0 {
	input.RunRef = strings.TrimSpace(input.RunRef)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.EvidenceRefs = compactBurstStringsV0(input.EvidenceRefs)
	return input
}

func newDirectorSupervisedBurstResultV0(
	input DirectorSupervisedBurstInputV0,
) DirectorSupervisedBurstResultV0 {
	return DirectorSupervisedBurstResultV0{
		RunRef:       input.RunRef,
		MaxSteps:     input.MaxSteps,
		EvidenceRefs: compactBurstStringsV0(input.EvidenceRefs),
	}
}

func burstStepRequestV0(
	input DirectorSupervisedBurstInputV0,
	stepNumber int,
	previousStep *orquestadirectorcycle.DirectorCycleStepResultV0,
	previousDecision *orquestadirectorsupervisor.DirectorSupervisorDecisionV0,
) DirectorSupervisedBurstStepRequestV0 {
	return DirectorSupervisedBurstStepRequestV0{
		RunRef:             input.RunRef,
		StepNumber:         stepNumber,
		MaxSteps:           input.MaxSteps,
		PreviousStepResult: previousStep,
		PreviousDecision:   previousDecision,
		CorrelationID:      input.CorrelationID,
		EvidenceRefs:       compactBurstStringsV0(input.EvidenceRefs),
	}
}

func burstSupervisorInputV0(
	input DirectorSupervisedBurstInputV0,
	stepNumber int,
	stepResult orquestadirectorcycle.DirectorCycleStepResultV0,
	errorCode string,
) orquestadirectorsupervisor.DirectorSupervisorDecisionInputV0 {
	return orquestadirectorsupervisor.DirectorSupervisorDecisionInputV0{
		RunRef:         input.RunRef,
		StepNumber:     stepNumber,
		MaxSteps:       input.MaxSteps,
		LastStepResult: stepResult,
		LastErrorCode:  errorCode,
		CorrelationID:  input.CorrelationID,
		EvidenceRefs:   compactBurstStringsV0(input.EvidenceRefs),
	}
}

func burstStepResultV0(
	stepNumber int,
	stepResult orquestadirectorcycle.DirectorCycleStepResultV0,
	decision orquestadirectorsupervisor.DirectorSupervisorDecisionV0,
	errorCode string,
) DirectorSupervisedBurstStepResultV0 {
	return DirectorSupervisedBurstStepResultV0{
		StepNumber:   stepNumber,
		CycleRef:     stepResult.CycleRef,
		TickRef:      stepResult.TickRef,
		CycleStatus:  string(stepResult.Status),
		Action:       decision.Action,
		ErrorCode:    errorCode,
		ShouldRepeat: decision.ShouldContinue,
	}
}

func publicCycleStepErrorCodeV0(err error) string {
	if err == nil {
		return ""
	}
	var publicErr orquestadirectorcycle.DirectorCycleStepErrorV0
	if errors.As(err, &publicErr) {
		return publicErr.Code
	}
	return ErrDirectorSupervisedBurstStepV0
}

func compactBurstStringsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
