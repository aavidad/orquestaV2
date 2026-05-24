package orquestacionnucleoapp

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type ProgressSupervisionCandidateProviderV0 struct {
	Base           CandidateProviderPortV0
	ProgressSource AgentProgressObservationProviderPortV0
	RequestedBy    string
}

var _ CandidateProviderPortV0 = ProgressSupervisionCandidateProviderV0{}

func (provider ProgressSupervisionCandidateProviderV0) BuildSchedulerCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	candidates, err := provider.baseCandidatesV0(ctx, request)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	if provider.ProgressSource == nil {
		candidates.ProgressSupervisionCandidates = progressSupervisionCandidatesForRunV0(
			candidates.ProgressSupervisionCandidates,
			request.Run.RunID,
		)
		return candidates, nil
	}
	observations, err := provider.ProgressSource.BuildAgentProgressObservationsV0(
		ctx,
		agentProgressObservationRequestV0(request),
	)
	if err != nil {
		return SchedulerCandidateSetV0{}, err
	}
	candidates.ProgressSupervisionCandidates = progressSupervisionCandidatesForRunV0(
		candidates.ProgressSupervisionCandidates,
		request.Run.RunID,
	)
	for _, observation := range observations {
		observation = normalizeProgressObservationV0(request, observation)
		if progressObservationFromDifferentRunV0(request, observation) {
			continue
		}
		if err := validateProgressObservationV0(request, observation); err != nil {
			return SchedulerCandidateSetV0{}, err
		}
		if !progressObservationRequiresSchedulerDecisionV0(observation) {
			continue
		}
		candidate, err := provider.progressCandidateV0(request, observation)
		if err != nil {
			return SchedulerCandidateSetV0{}, err
		}
		candidates.ProgressSupervisionCandidates = append(candidates.ProgressSupervisionCandidates, candidate)
		candidates.EvidenceRefs = compactStringsV0(append(candidates.EvidenceRefs, candidate.EvidenceRefs...))
		return candidates, nil
	}
	return candidates, nil
}

func progressObservationFromDifferentRunV0(
	request SchedulerCandidateRequestV0,
	observation AgentProgressObservationV0,
) bool {
	runRef := strings.TrimSpace(request.Run.RunID)
	reportRunRef := strings.TrimSpace(observation.Report.RunID)
	return runRef != "" && reportRunRef != "" && reportRunRef != runRef
}

func progressSupervisionCandidatesForRunV0(
	candidates []orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0,
	runRef string,
) []orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0 {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" || len(candidates) == 0 {
		return candidates
	}
	filtered := make([]orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0, 0, len(candidates))
	for _, candidate := range candidates {
		candidateRunRef := strings.TrimSpace(candidate.SupervisionInput.Report.RunID)
		commandRunRef := strings.TrimSpace(candidate.SupervisionInput.CommandMeta.RunID)
		if commandRunRef != runRef || candidateRunRef != runRef {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}

func progressObservationRequiresSchedulerDecisionV0(
	observation AgentProgressObservationV0,
) bool {
	if observation.DecisionRequired || observation.Report.DecisionRequired {
		return true
	}
	return observation.Report.Status == orquestaruntime.AgentLoopDetectedV0 ||
		observation.Report.Status == orquestaruntime.AgentStoppedV0
}

func (provider ProgressSupervisionCandidateProviderV0) baseCandidatesV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
) (SchedulerCandidateSetV0, error) {
	if provider.Base == nil {
		return SchedulerCandidateSetV0{}, nil
	}
	return provider.Base.BuildSchedulerCandidatesV0(ctx, request)
}

func agentProgressObservationRequestV0(
	request SchedulerCandidateRequestV0,
) AgentProgressObservationRequestV0 {
	return AgentProgressObservationRequestV0{
		Run:              request.Run,
		StepNumber:       request.StepNumber,
		MaxSteps:         request.MaxSteps,
		OccurredAt:       request.OccurredAt,
		PreviousStep:     request.PreviousStep,
		CorrelationID:    request.CorrelationID,
		EvidenceRefs:     request.EvidenceRefs,
		PreviousDecision: request.PreviousDecision,
	}
}

func (provider ProgressSupervisionCandidateProviderV0) progressCandidateV0(
	request SchedulerCandidateRequestV0,
	observation AgentProgressObservationV0,
) (orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0, error) {
	observation = normalizeProgressObservationV0(request, observation)
	stopAllowed := protectedDirectionStopAllowedV0(request.Run, observation)
	if stopAllowed != nil && !*stopAllowed && observation.QuestionID == "" {
		observation.QuestionID = "question-ref-" + observation.Report.ReportID
	}
	if err := validateProgressObservationV0(request, observation); err != nil {
		return orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0{}, err
	}
	report := observation.Report
	meta := orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      "cmd-progress-supervision-" + report.ReportID,
		RunID:          report.RunID,
		IdempotencyKey: "idem-progress-supervision-" + report.ReportID,
		CorrelationID:  strings.TrimSpace(request.CorrelationID),
		RequestedBy:    strings.TrimSpace(provider.RequestedBy),
		OccurredAt:     strings.TrimSpace(request.OccurredAt),
	}
	return orquestadirectorscheduler.SchedulableProgressSupervisionCandidateV0{
		CandidateRef: observation.CandidateRef,
		SupervisionInput: orquestadirector.AgentProgressSupervisionInputV0{
			CommandMeta:   meta,
			Report:        report,
			PhaseID:       observation.PhaseID,
			TaskRef:       observation.TaskRef,
			DeliveryRef:   observation.DeliveryRef,
			AssessmentRef: observation.AssessmentRef,
			QuestionID:    observation.QuestionID,
			StopAllowed:   stopAllowed,
		},
		EvidenceRefs: compactStringsV0(append(request.EvidenceRefs, observation.EvidenceRefs...)),
	}, nil
}

func normalizeProgressObservationV0(
	request SchedulerCandidateRequestV0,
	observation AgentProgressObservationV0,
) AgentProgressObservationV0 {
	observation.CandidateRef = strings.TrimSpace(observation.CandidateRef)
	observation.PhaseID = strings.TrimSpace(observation.PhaseID)
	observation.TaskRef = strings.TrimSpace(observation.TaskRef)
	observation.DeliveryRef = strings.TrimSpace(observation.DeliveryRef)
	observation.AssessmentRef = strings.TrimSpace(observation.AssessmentRef)
	observation.QuestionID = strings.TrimSpace(observation.QuestionID)
	observation.EvidenceRefs = compactStringsV0(observation.EvidenceRefs)
	observation.Report.ReportID = strings.TrimSpace(observation.Report.ReportID)
	observation.Report.RunID = strings.TrimSpace(observation.Report.RunID)
	observation.Report.AgentRequestID = strings.TrimSpace(observation.Report.AgentRequestID)
	observation.Report.Summary = strings.TrimSpace(observation.Report.Summary)
	observation.Report.EvidenceRefs = compactStringsV0(observation.Report.EvidenceRefs)
	if observation.CandidateRef == "" {
		observation.CandidateRef = "progress-supervision-candidate-ref-" + observation.Report.ReportID
	}
	if observation.PhaseID == "" {
		observation.PhaseID = string(request.Run.CurrentPhase)
	}
	if observation.AssessmentRef == "" {
		observation.AssessmentRef = progressObservationAssessmentRefV0(observation)
	}
	if observation.QuestionID == "" && progressObservationRequiresSchedulerDecisionV0(observation) {
		observation.QuestionID = progressObservationQuestionRefV0(observation)
	}
	return observation
}

func progressObservationAssessmentRefV0(observation AgentProgressObservationV0) string {
	if progressObservationNeedsStableAdvisoryRefV0(observation) {
		return "assessment-ref-" + progressObservationStableAdvisoryHashV0(observation)
	}
	return "assessment-ref-" + observation.Report.ReportID
}

func progressObservationQuestionRefV0(observation AgentProgressObservationV0) string {
	if progressObservationNeedsStableAdvisoryRefV0(observation) {
		return "question-ref-" + progressObservationStableAdvisoryHashV0(observation)
	}
	return "question-ref-" + observation.Report.ReportID
}

func progressObservationNeedsStableAdvisoryRefV0(
	observation AgentProgressObservationV0,
) bool {
	if observation.Report.Status == orquestaruntime.AgentStoppedV0 ||
		observation.Report.Status == orquestaruntime.AgentLoopDetectedV0 {
		return false
	}
	return observation.DecisionRequired ||
		observation.Report.DecisionRequired ||
		observation.Report.Status == orquestaruntime.AgentStalledV0 ||
		observation.Report.BudgetStatus == orquestaruntime.AgentProgressBudgetOverBudgetButActiveV0 ||
		observation.Report.BudgetStatus == orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0
}

func progressObservationStableAdvisoryHashV0(
	observation AgentProgressObservationV0,
) string {
	parts := []string{
		observation.Report.RunID,
		observation.Report.AgentRequestID,
		observation.TaskRef,
		string(observation.Report.Status),
		string(observation.Report.BudgetStatus),
	}
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return "agent-progress-" + hex.EncodeToString(sum[:])[:16]
}

func validateProgressObservationV0(
	request SchedulerCandidateRequestV0,
	observation AgentProgressObservationV0,
) error {
	if strings.TrimSpace(request.OccurredAt) == "" {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "occurred_at", "occurred_at requerido")
	}
	if issues := orquestaruntime.ValidateAgentProgressReportV0(observation.Report); len(issues) > 0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "agent_progress_report."+issues[0].Field, string(issues[0].Code))
	}
	if observation.Report.RunID != request.Run.RunID {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "agent_progress_report.run_id", "run_id no coincide")
	}
	if strings.TrimSpace(observation.PhaseID) == "" {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "phase_id", "phase_id requerido")
	}
	if strings.TrimSpace(observation.AssessmentRef) == "" {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "assessment_ref", "assessment_ref requerido")
	}
	if strings.TrimSpace(observation.CandidateRef) == "" {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "candidate_ref", "candidate_ref requerido")
	}
	return nil
}
