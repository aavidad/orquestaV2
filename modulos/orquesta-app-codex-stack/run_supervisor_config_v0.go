package orquestaappcodexstack

const (
	DefaultRunSupervisorMaxTicksV0      = 4
	DefaultRunSupervisorMaxExecutionsV0 = 4
)

func normalizeRunSupervisorConfigV0(config RunSupervisorConfigV0) RunSupervisorConfigV0 {
	if config.MaxTicks <= 0 {
		config.MaxTicks = DefaultRunSupervisorMaxTicksV0
	}
	if config.MaxExecutions < 0 {
		config.MaxExecutions = 0
	}
	if config.MaxExecutions == 0 {
		config.MaxExecutions = DefaultRunSupervisorMaxExecutionsV0
	}
	config.StopOnNoExecution = true
	return config
}
