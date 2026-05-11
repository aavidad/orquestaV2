package orquestaapprunner

const (
	defaultRunPreparedMaxBurstsV0            = 120
	defaultRunPreparedMaxStepsPerBurstV0     = 8
	defaultRunPreparedMaxDispatchesPerWaitV0 = 8
	defaultRunPreparedMaxCommandsV0          = 12
	defaultRunPreparedMaxOutboxPerCycleV0    = 8
	defaultRunPreparedMaxExternalWaitsV0     = 120
	maxRunPreparedMaxBurstsV0                = 1000
	maxRunPreparedMaxStepsPerBurstV0         = 50
	maxRunPreparedMaxDispatchesPerWaitV0     = 50
	maxRunPreparedMaxCommandsV0              = 20
	maxRunPreparedMaxOutboxPerCycleV0        = 20
	maxRunPreparedMaxExternalWaitsV0         = 1000
)

func normalizeRunPreparedLimitsV0(
	request RunPreparedAppOrchestrationRequestV0,
) RunPreparedAppOrchestrationRequestV0 {
	request.MaxBursts = normalizeRunPreparedLimitV0(
		request.MaxBursts,
		defaultRunPreparedMaxBurstsV0,
		maxRunPreparedMaxBurstsV0,
	)
	request.MaxStepsPerBurst = normalizeRunPreparedLimitV0(
		request.MaxStepsPerBurst,
		defaultRunPreparedMaxStepsPerBurstV0,
		maxRunPreparedMaxStepsPerBurstV0,
	)
	request.MaxDispatchesPerWait = normalizeRunPreparedLimitV0(
		request.MaxDispatchesPerWait,
		defaultRunPreparedMaxDispatchesPerWaitV0,
		maxRunPreparedMaxDispatchesPerWaitV0,
	)
	request.MaxCommands = normalizeRunPreparedLimitV0(
		request.MaxCommands,
		defaultRunPreparedMaxCommandsV0,
		maxRunPreparedMaxCommandsV0,
	)
	request.MaxOutboxPerCycle = normalizeRunPreparedLimitV0(
		request.MaxOutboxPerCycle,
		defaultRunPreparedMaxOutboxPerCycleV0,
		maxRunPreparedMaxOutboxPerCycleV0,
	)
	request.MaxExternalWaits = normalizeRunPreparedLimitV0(
		request.MaxExternalWaits,
		defaultRunPreparedMaxExternalWaitsV0,
		maxRunPreparedMaxExternalWaitsV0,
	)
	return request
}

func normalizeRunPreparedLimitV0(value int, fallback int, max int) int {
	if value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}
