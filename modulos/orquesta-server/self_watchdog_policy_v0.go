package orquestaserver

import "time"

func NormalizeSelfWatchdogConfigV0(config SelfWatchdogConfigV0) SelfWatchdogConfigV0 {
	if config.CPUHighPercent <= 0 {
		config.CPUHighPercent = DefaultSelfWatchdogCPUHighPercentV0
	}
	if config.SustainedFor <= 0 {
		config.SustainedFor = DefaultSelfWatchdogSustainedForV0
	}
	if config.NoProgressFor <= 0 {
		config.NoProgressFor = DefaultSelfWatchdogNoProgressForV0
	}
	return config
}

func EvaluateSelfWatchdogV0(
	config SelfWatchdogConfigV0,
	observation SelfWatchdogObservationV0,
) SelfWatchdogDecisionV0 {
	config = NormalizeSelfWatchdogConfigV0(config)
	observation = normalizeSelfWatchdogObservationV0(observation)
	if config.Disabled {
		return selfWatchdogDecisionV0(SelfWatchdogStatusDisabledV0, SelfWatchdogReasonDisabledV0, config, observation, false)
	}
	if observation.CPUPercent < config.CPUHighPercent {
		return selfWatchdogDecisionV0(SelfWatchdogStatusOKV0, SelfWatchdogReasonCPUWithinLimitV0, config, observation, false)
	}
	if selfWatchdogHasOperationalCauseV0(observation) {
		return selfWatchdogDecisionV0(SelfWatchdogStatusHighCPUWithCauseV0, SelfWatchdogReasonOperationalCauseV0, config, observation, false)
	}
	if selfWatchdogHasRecentProgressV0(config, observation) {
		return selfWatchdogDecisionV0(SelfWatchdogStatusHighCPUWithCauseV0, SelfWatchdogReasonRecentProgressV0, config, observation, false)
	}
	if observation.HighCPUSince.IsZero() ||
		observation.ObservedAt.Sub(observation.HighCPUSince) < config.SustainedFor {
		return selfWatchdogDecisionV0(SelfWatchdogStatusObservingHighCPUV0, SelfWatchdogReasonSustainedWindowOpenV0, config, observation, false)
	}
	return selfWatchdogDecisionV0(SelfWatchdogStatusShutdownRequestedV0, SelfWatchdogReasonNoProgressV0, config, observation, true)
}

func normalizeSelfWatchdogObservationV0(observation SelfWatchdogObservationV0) SelfWatchdogObservationV0 {
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = time.Now().UTC()
	}
	observation.ObservedAt = observation.ObservedAt.UTC()
	observation.HighCPUSince = observation.HighCPUSince.UTC()
	observation.LastProgressAt = observation.LastProgressAt.UTC()
	observation.CPUPercent = nonNegativeServerIntV0(observation.CPUPercent)
	observation.ActiveRuns = nonNegativeServerIntV0(observation.ActiveRuns)
	observation.PendingOutbox = nonNegativeServerIntV0(observation.PendingOutbox)
	observation.ActiveAgents = nonNegativeServerIntV0(observation.ActiveAgents)
	observation.RegisteredProcesses = nonNegativeServerIntV0(observation.RegisteredProcesses)
	observation.AsyncWorkActive = nonNegativeServerIntV0(observation.AsyncWorkActive)
	observation.EvidenceRefs = compactServerStringsV0(observation.EvidenceRefs)
	return observation
}

func selfWatchdogHasOperationalCauseV0(observation SelfWatchdogObservationV0) bool {
	return observation.ActiveRuns > 0 ||
		observation.PendingOutbox > 0 ||
		observation.ActiveAgents > 0 ||
		observation.RegisteredProcesses > 0 ||
		observation.SupervisorTickActive ||
		observation.AsyncWorkActive > 0 ||
		observation.ShutdownInProgress
}

func selfWatchdogHasRecentProgressV0(
	config SelfWatchdogConfigV0,
	observation SelfWatchdogObservationV0,
) bool {
	return !observation.LastProgressAt.IsZero() &&
		!observation.LastProgressAt.After(observation.ObservedAt) &&
		observation.ObservedAt.Sub(observation.LastProgressAt) < config.NoProgressFor
}

func selfWatchdogDecisionV0(
	status string,
	reason string,
	_ SelfWatchdogConfigV0,
	observation SelfWatchdogObservationV0,
	shouldShutdown bool,
) SelfWatchdogDecisionV0 {
	return SelfWatchdogDecisionV0{
		Status:                status,
		ReasonCode:            reason,
		ObservedAt:            formatTimeV0(observation.ObservedAt),
		HighCPUSince:          formatTimeV0(observation.HighCPUSince),
		CPUPercent:            observation.CPUPercent,
		ShouldRequestShutdown: shouldShutdown,
		EvidenceRefs:          append([]string(nil), observation.EvidenceRefs...),
	}
}
