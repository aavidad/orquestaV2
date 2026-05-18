package orquestacionnucleoapp

import (
	"context"
	"fmt"

	orquestadirectorcycle "orquesta/modulos/orquesta-director-cycle"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestadirectorsupervisedburst "orquesta/modulos/orquesta-director-supervised-burst"
)

type cycleStepInputBuilderV0 struct {
	RunStore          RunStorePortV0
	CandidateProvider CandidateProviderPortV0
	OutboxLedger      orquestadirectorcycleoutbox.DirectorCycleOutboxLedgerPortV0
	Workflow          storedWorkflowPortV0
	Request           SupervisedBurstRequestV0
	MaxCommands       int
	MaxOutboxPerCycle int
}

func (builder cycleStepInputBuilderV0) BuildDirectorCycleStepInputV0(
	ctx context.Context,
	request orquestadirectorsupervisedburst.DirectorSupervisedBurstStepRequestV0,
) (orquestadirectorcycle.DirectorCycleStepInputV0, error) {
	run, err := builder.RunStore.LoadRunV0(ctx, request.RunRef)
	if err != nil {
		return orquestadirectorcycle.DirectorCycleStepInputV0{}, err
	}
	candidates, err := builder.CandidateProvider.BuildSchedulerCandidatesV0(ctx, SchedulerCandidateRequestV0{
		Run:              run,
		StepNumber:       request.StepNumber,
		MaxSteps:         request.MaxSteps,
		OccurredAt:       builder.Request.OccurredAt,
		PreviousStep:     request.PreviousStepResult,
		CorrelationID:    request.CorrelationID,
		EvidenceRefs:     request.EvidenceRefs,
		PreviousDecision: previousDecisionActionV0(request),
		WaitAgentRefs:    builder.Request.WaitAgentRefs,
	})
	if err != nil {
		return orquestadirectorcycle.DirectorCycleStepInputV0{}, err
	}
	return orquestadirectorcycle.DirectorCycleStepInputV0{
		Scheduler:                     orquestadirectorrunner.DirectDirectorSchedulerPortV0{},
		Workflow:                      builder.Workflow,
		OutboxLedger:                  builder.OutboxLedger,
		CycleRef:                      stepRefV0(request.RunRef, "cycle", request.StepNumber),
		TickRef:                       stepRefV0(request.RunRef, "tick", request.StepNumber),
		RunRef:                        request.RunRef,
		OccurredAt:                    builder.Request.OccurredAt,
		Run:                           run,
		LeaseActionCandidates:         candidates.LeaseActionCandidates,
		PhaseArtifactCandidates:       candidates.PhaseArtifactCandidates,
		DeliveryCandidates:            candidates.DeliveryCandidates,
		ReviewGateCandidates:          candidates.ReviewGateCandidates,
		ProgressSupervisionCandidates: candidates.ProgressSupervisionCandidates,
		ReplanFollowupCandidates:      candidates.ReplanFollowupCandidates,
		WorkClaims:                    candidates.WorkClaims,
		WorkCandidates:                candidates.WorkCandidates,
		MaxCommands:                   builder.MaxCommands,
		MaxOutbox:                     builder.MaxOutboxPerCycle,
		CorrelationID:                 request.CorrelationID,
		EvidenceRefs:                  mergedEvidenceRefsV0(request.EvidenceRefs, candidates.EvidenceRefs),
	}, nil
}

func previousDecisionActionV0(
	request orquestadirectorsupervisedburst.DirectorSupervisedBurstStepRequestV0,
) string {
	if request.PreviousDecision == nil {
		return ""
	}
	return string(request.PreviousDecision.Action)
}

func stepRefV0(runRef string, kind string, stepNumber int) string {
	return fmt.Sprintf("%s-%s-%03d", runRef, kind, stepNumber)
}

func mergedEvidenceRefsV0(left []string, right []string) []string {
	merged := make([]string, 0, len(left)+len(right))
	merged = append(merged, left...)
	merged = append(merged, right...)
	return compactStringsV0(merged)
}
