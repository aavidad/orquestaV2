package orquestacionnucleoapp

import (
	"context"
	"strings"
	"time"

	orquestacoreleases "orquesta/modulos/orquesta-core-leases"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

var _ AgentProgressObservationProviderPortV0 = (*AgentProgressLeaseBridgeV0)(nil)
var _ AgentLeaseAssessmentProviderPortV0 = (*AgentProgressLeaseBridgeV0)(nil)

func (provider StaticAgentProgressLeasePolicyProviderV0) BuildAgentProgressLeasePolicyV0(
	_ context.Context,
	_ AgentProgressLeasePolicyRequestV0,
) (AgentProgressLeasePolicyV0, bool, error) {
	policy := AgentProgressLeasePolicyV0{
		LeaseRef:         strings.TrimSpace(provider.LeaseRef),
		LaunchObservedAt: progressLeaseUTCInstantV0(provider.LaunchObservedAt),
		Policy:           provider.Policy,
		EvidenceRefs:     compactStringsV0(provider.EvidenceRefs),
	}
	return policy, true, nil
}

func (bridge *AgentProgressLeaseBridgeV0) BuildAgentProgressObservationsV0(
	ctx context.Context,
	request AgentProgressObservationRequestV0,
) ([]AgentProgressObservationV0, error) {
	if bridge == nil || bridge.ProgressSource == nil {
		return nil, nil
	}
	observations, err := bridge.ProgressSource.BuildAgentProgressObservationsV0(ctx, request)
	if err != nil {
		return nil, err
	}
	bridge.storeProgressLeaseObservationsV0(progressLeaseBridgeKeyFromProgressV0(request), observations)
	return observations, nil
}

func (bridge *AgentProgressLeaseBridgeV0) BuildAgentLeaseAssessmentsV0(
	ctx context.Context,
	request AgentLeaseAssessmentRequestV0,
) ([]orquestacoreleases.AgentTimeoutAssessmentV0, error) {
	if bridge == nil || bridge.ProgressSource == nil || bridge.PolicySource == nil {
		return nil, nil
	}
	observations, err := bridge.progressObservationsForLeaseV0(ctx, request)
	if err != nil {
		return nil, err
	}
	assessments := make([]orquestacoreleases.AgentTimeoutAssessmentV0, 0, len(observations))
	for _, observation := range observations {
		observation = normalizeProgressObservationV0(progressLeaseSchedulerRequestV0(request), observation)
		if progressObservationFromDifferentRunV0(progressLeaseSchedulerRequestV0(request), observation) {
			continue
		}
		leasePolicy, ok, err := bridge.PolicySource.BuildAgentProgressLeasePolicyV0(
			ctx,
			progressLeasePolicyRequestV0(request, observation),
		)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		assessment, ready, err := BuildAgentLeaseAssessmentFromProgressV0(
			progressLeaseAssessmentInputV0(request, observation, leasePolicy),
		)
		if err != nil {
			return nil, err
		}
		if ready {
			assessments = append(assessments, assessment)
		}
	}
	return assessments, nil
}

func BuildAgentLeaseAssessmentFromProgressV0(
	input AgentProgressLeaseAssessmentInputV0,
) (orquestacoreleases.AgentTimeoutAssessmentV0, bool, error) {
	report := normalizeProgressObservationV0(SchedulerCandidateRequestV0{}, input.Report).Report
	if issues := orquestaruntime.ValidateAgentProgressReportV0(report); len(issues) > 0 {
		return orquestacoreleases.AgentTimeoutAssessmentV0{}, false,
			errorV0(ErrNucleoOrquestacionInvalidoV0, "agent_progress_report."+issues[0].Field, string(issues[0].Code))
	}
	observedAt := progressLeaseUTCInstantV0(input.ObservedAt)
	heartbeatObservedAt := progressLeaseHeartbeatObservedAtV0(report, observedAt)
	leaseRef := progressLeaseRefV0(input.LeaseRef, report.AgentRequestID)
	evaluation := orquestacoreleases.AgentLeaseEvaluationInputV0{
		AssessmentRef:    progressLeaseAssessmentRefV0(input.AssessmentRef, report.ReportID),
		RunRef:           report.RunID,
		AgentRequestID:   report.AgentRequestID,
		LeaseRef:         leaseRef,
		LaunchObservedAt: progressLeaseLaunchObservedAtV0(input.LaunchObservedAt, report, heartbeatObservedAt),
		NowObservedAt:    observedAt,
		Policy:           input.Policy,
		LastHeartbeat: &orquestacoreleases.AgentHeartbeatReportV0{
			HeartbeatRef:      progressLeaseHeartbeatRefV0(report.ReportID),
			RunRef:            report.RunID,
			AgentRequestID:    report.AgentRequestID,
			LeaseRef:          leaseRef,
			ObservedAt:        heartbeatObservedAt,
			Status:            progressLeaseHeartbeatStatusV0(report),
			ProgressReportRef: report.ReportID,
			EvidenceRefs:      compactStringsV0(append(input.EvidenceRefs, report.EvidenceRefs...)),
		},
		EvidenceRefs: compactStringsV0(input.EvidenceRefs),
	}
	assessment, err := orquestacoreleases.EvaluateAgentLeaseV0(evaluation)
	if err != nil {
		return orquestacoreleases.AgentTimeoutAssessmentV0{}, false, err
	}
	return assessment, true, nil
}

func (bridge *AgentProgressLeaseBridgeV0) progressObservationsForLeaseV0(
	ctx context.Context,
	request AgentLeaseAssessmentRequestV0,
) ([]AgentProgressObservationV0, error) {
	key := progressLeaseBridgeKeyFromLeaseV0(request)
	if observations, ok := bridge.cachedProgressLeaseObservationsV0(key); ok {
		return observations, nil
	}
	observations, err := bridge.ProgressSource.BuildAgentProgressObservationsV0(
		ctx,
		progressObservationRequestFromLeaseV0(request),
	)
	if err != nil {
		return nil, err
	}
	bridge.storeProgressLeaseObservationsV0(key, observations)
	return observations, nil
}

func progressLeaseAssessmentInputV0(
	request AgentLeaseAssessmentRequestV0,
	observation AgentProgressObservationV0,
	leasePolicy AgentProgressLeasePolicyV0,
) AgentProgressLeaseAssessmentInputV0 {
	return AgentProgressLeaseAssessmentInputV0{
		LeaseRef:         leasePolicy.LeaseRef,
		ObservedAt:       request.OccurredAt,
		LaunchObservedAt: leasePolicy.LaunchObservedAt,
		Report:           observation,
		Policy:           leasePolicy.Policy,
		EvidenceRefs:     compactStringsV0(append(request.EvidenceRefs, leasePolicy.EvidenceRefs...)),
	}
}

func progressLeasePolicyRequestV0(
	request AgentLeaseAssessmentRequestV0,
	observation AgentProgressObservationV0,
) AgentProgressLeasePolicyRequestV0 {
	return AgentProgressLeasePolicyRequestV0{
		Run:           request.Run,
		Observation:   observation,
		OccurredAt:    request.OccurredAt,
		CorrelationID: request.CorrelationID,
		EvidenceRefs:  compactStringsV0(request.EvidenceRefs),
	}
}

func progressObservationRequestFromLeaseV0(
	request AgentLeaseAssessmentRequestV0,
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

func progressLeaseSchedulerRequestV0(request AgentLeaseAssessmentRequestV0) SchedulerCandidateRequestV0 {
	return SchedulerCandidateRequestV0{
		Run:           request.Run,
		OccurredAt:    request.OccurredAt,
		CorrelationID: request.CorrelationID,
		EvidenceRefs:  request.EvidenceRefs,
	}
}

func progressLeaseHeartbeatStatusV0(
	report orquestaruntime.AgentProgressReportV0,
) orquestacoreleases.AgentHeartbeatStatusV0 {
	if report.Status == orquestaruntime.AgentStoppedV0 {
		return orquestacoreleases.AgentHeartbeatStoppedV0
	}
	if report.Status == orquestaruntime.AgentStalledV0 ||
		report.Status == orquestaruntime.AgentLoopDetectedV0 ||
		report.BudgetStatus == orquestaruntime.AgentProgressBudgetOverBudgetNoActivityV0 {
		return orquestacoreleases.AgentHeartbeatStalledV0
	}
	return orquestacoreleases.AgentHeartbeatProgressingV0
}

func progressLeaseHeartbeatObservedAtV0(
	report orquestaruntime.AgentProgressReportV0,
	fallback string,
) string {
	fallback = progressLeaseUTCInstantV0(fallback)
	if value, ok := progressLeaseValidUTCInstantV0(report.LastActivityAt); ok {
		return progressLeaseClampFutureInstantV0(value, fallback)
	}
	return fallback
}

func progressLeaseLaunchObservedAtV0(
	configured string,
	report orquestaruntime.AgentProgressReportV0,
	fallback string,
) string {
	fallback = progressLeaseUTCInstantV0(fallback)
	if value, ok := progressLeaseValidUTCInstantV0(configured); ok {
		return progressLeaseClampFutureInstantV0(value, fallback)
	}
	if value, ok := progressLeaseValidUTCInstantV0(report.StartedAt); ok {
		return progressLeaseClampFutureInstantV0(value, fallback)
	}
	return fallback
}

func progressLeaseUTCInstantV0(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return strings.TrimSpace(value)
	}
	return parsed.UTC().Format("2006-01-02T15:04:05Z")
}

func progressLeaseValidUTCInstantV0(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return "", false
	}
	return parsed.UTC().Format("2006-01-02T15:04:05Z"), true
}

func progressLeaseClampFutureInstantV0(value string, ceiling string) string {
	if strings.TrimSpace(ceiling) == "" {
		return value
	}
	valueAt, err := time.Parse("2006-01-02T15:04:05Z", value)
	if err != nil {
		return ceiling
	}
	ceilingAt, err := time.Parse("2006-01-02T15:04:05Z", ceiling)
	if err != nil {
		return value
	}
	if valueAt.After(ceilingAt) {
		return ceiling
	}
	return value
}

func progressLeaseAssessmentRefV0(configured string, reportRef string) string {
	if trimmed := strings.TrimSpace(configured); trimmed != "" {
		return trimmed
	}
	return "lease-assessment-ref-" + progressLeaseSafeRefPartV0(reportRef)
}

func progressLeaseHeartbeatRefV0(reportRef string) string {
	return "heartbeat-ref-" + progressLeaseSafeRefPartV0(reportRef)
}

func progressLeaseRefV0(configured string, agentRef string) string {
	if trimmed := strings.TrimSpace(configured); trimmed != "" {
		return trimmed
	}
	return "lease-ref-progress-" + progressLeaseSafeRefPartV0(agentRef)
}

func progressLeaseSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer("\\", "-", "/", "-", " ", "-").Replace(value)
	if value != "" {
		return value
	}
	return deterministicRefDigestPrefixV0("progress_lease_empty_ref", 32, "empty-progress-lease-ref")
}
