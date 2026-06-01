package orquestadirectorcycle

import (
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
)

func normalizeDirectorCycleStepsInputV0(input DirectorCycleStepsInputV0) DirectorCycleStepsInputV0 {
	if directorCycleStepInputEmptyV0(input.InitialStep) && !directorCycleStepInputEmptyV0(input.StepTemplate) {
		input.InitialStep = input.StepTemplate
	}
	input.InitialStep = normalizeDirectorCycleStepInputV0(input.InitialStep)
	if directorCycleStepInputEmptyV0(input.StepTemplate) {
		input.StepTemplate = input.InitialStep
	} else {
		input.StepTemplate = normalizeDirectorCycleStepInputV0(input.StepTemplate)
	}
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	if input.CorrelationID == "" {
		input.CorrelationID = input.InitialStep.CorrelationID
	}
	input.EvidenceRefs = compactDirectorCycleStepStringsV0(input.EvidenceRefs)
	if len(input.EvidenceRefs) == 0 {
		input.EvidenceRefs = compactDirectorCycleStepStringsV0(input.InitialStep.EvidenceRefs)
	}
	if input.MaxSteps == 0 {
		input.MaxSteps = 1
	}
	return input
}

func newDirectorCycleStepsResultV0(input DirectorCycleStepsInputV0) DirectorCycleStepsResultV0 {
	return DirectorCycleStepsResultV0{
		RunRef:        input.InitialStep.RunRef,
		MaxSteps:      input.MaxSteps,
		CorrelationID: input.CorrelationID,
		EvidenceRefs:  append([]string(nil), input.EvidenceRefs...),
	}
}

func appendCycleStepsStepResultV0(
	result DirectorCycleStepsResultV0,
	stepResult DirectorCycleStepResultV0,
) DirectorCycleStepsResultV0 {
	result.ExecutedSteps++
	result.StepsExecuted = result.ExecutedSteps
	result.Status = stepResult.Status
	result.LastStepResult = stepResult
	result.StepResults = append(result.StepResults, stepResult)
	result.PendingOutboxRefs = compactDirectorCycleStepStringsV0(append(
		append([]string{}, stepResult.PendingOutboxBeforeRefs...),
		stepResult.PendingOutboxAfterRefs...,
	))
	result.WaitingReasons = append([]orquestadirectorscheduler.SchedulerWaitingReasonV0(nil), stepResult.WaitingReasons...)
	result.BlockedRefs = compactDirectorCycleStepStringsV0(stepResult.BlockedRefs)
	return result
}

func directorCycleStepInputEmptyV0(input DirectorCycleStepInputV0) bool {
	return strings.TrimSpace(input.CycleRef) == "" &&
		strings.TrimSpace(input.TickRef) == "" &&
		strings.TrimSpace(input.RunRef) == "" &&
		strings.TrimSpace(input.OccurredAt) == "" &&
		strings.TrimSpace(input.Run.RunID) == "" &&
		isNilCycleStepPortV0(input.Scheduler) &&
		isNilCycleStepPortV0(input.Workflow) &&
		isNilCycleStepPortV0(input.OutboxLedger)
}

func cycleStepsStopReasonForStepV0(
	stepNumber int,
	maxSteps int,
	stepResult DirectorCycleStepResultV0,
) string {
	if cycleStepsStepHasPendingOutboxV0(stepResult) {
		return DirectorCycleStepsStopWaitOutboxV0
	}
	switch stepResult.Status {
	case orquestadirectorrunner.DirectorCycleStatusOutboxPendingV0:
		return DirectorCycleStepsStopWaitOutboxV0
	case orquestadirectorrunner.DirectorCycleStatusWaitingV0:
		if cycleStepsHasWaitingReasonV0(stepResult, orquestadirectorscheduler.SchedulerWaitingOutboxPendingV0) {
			return DirectorCycleStepsStopWaitOutboxV0
		}
		return DirectorCycleStepsStopWaitExternalV0
	case orquestadirectorrunner.DirectorCycleStatusBlockedV0:
		return DirectorCycleStepsStopBlockedV0
	case orquestadirectorrunner.DirectorCycleStatusNeedsDirectorV0:
		return DirectorCycleStepsStopNeedsDirectorV0
	case orquestadirectorrunner.DirectorCycleStatusQuiescentV0:
		return DirectorCycleStepsStopQuiescentV0
	case orquestadirectorrunner.DirectorCycleStatusCommandsAppliedV0:
		if stepNumber >= maxSteps {
			return DirectorCycleStepsStopMaxStepsV0
		}
		return ""
	default:
		return DirectorCycleStepsStopErrorV0
	}
}

func cycleStepsStepHasPendingOutboxV0(stepResult DirectorCycleStepResultV0) bool {
	return stepResult.OutboxPendingAfterCount > 0 ||
		len(stepResult.PendingOutboxBeforeRefs) > 0 ||
		len(stepResult.PendingOutboxAfterRefs) > 0
}

func cycleStepsHasWaitingReasonV0(
	stepResult DirectorCycleStepResultV0,
	reason orquestadirectorscheduler.SchedulerWaitingReasonV0,
) bool {
	for _, existing := range stepResult.WaitingReasons {
		if existing == reason {
			return true
		}
	}
	return false
}

func cycleStepsSnapshotFromRunSnapshotV0(
	input DirectorCycleStepsInputV0,
	runSnapshot orquestacoreworkflow.OrchestrationRunV0,
	stepNumber int,
) DirectorCycleStepSnapshotV0 {
	return DirectorCycleStepSnapshotV0{
		CycleRef:                      cycleStepsRefForStepV0(input.InitialStep.CycleRef, stepNumber),
		TickRef:                       cycleStepsRefForStepV0(input.InitialStep.TickRef, stepNumber),
		OccurredAt:                    input.InitialStep.OccurredAt,
		Run:                           runSnapshot,
		LeaseActionCandidates:         nil,
		PhaseArtifactCandidates:       nil,
		DeliveryCandidates:            nil,
		ReviewGateCandidates:          nil,
		ProgressSupervisionCandidates: nil,
		ReplanFollowupCandidates:      nil,
		WorkClaims:                    nil,
		WorkCandidates:                nil,
		EvidenceRefs:                  input.EvidenceRefs,
	}
}

func cycleStepsRefForStepV0(ref string, stepNumber int) string {
	ref = strings.TrimSpace(ref)
	if stepNumber <= 1 || ref == "" {
		return ref
	}
	return fmt.Sprintf("%s-step-%d", ref, stepNumber)
}

func cycleStepsStepInputFromSnapshotV0(
	base DirectorCycleStepInputV0,
	snapshot DirectorCycleStepSnapshotV0,
) DirectorCycleStepInputV0 {
	next := base
	next.CycleRef = strings.TrimSpace(snapshot.CycleRef)
	next.TickRef = strings.TrimSpace(snapshot.TickRef)
	next.OccurredAt = strings.TrimSpace(snapshot.OccurredAt)
	next.Run = snapshot.Run
	next.RunRef = strings.TrimSpace(snapshot.Run.RunID)
	next.LeaseActionCandidates = snapshot.LeaseActionCandidates
	next.PhaseArtifactCandidates = snapshot.PhaseArtifactCandidates
	next.DeliveryCandidates = snapshot.DeliveryCandidates
	next.ReviewGateCandidates = snapshot.ReviewGateCandidates
	next.ProgressSupervisionCandidates = snapshot.ProgressSupervisionCandidates
	next.ReplanFollowupCandidates = snapshot.ReplanFollowupCandidates
	next.WorkClaims = snapshot.WorkClaims
	next.WorkCandidates = snapshot.WorkCandidates
	if len(snapshot.EvidenceRefs) > 0 {
		next.EvidenceRefs = compactDirectorCycleStepStringsV0(snapshot.EvidenceRefs)
	}
	return normalizeDirectorCycleStepInputV0(next)
}
