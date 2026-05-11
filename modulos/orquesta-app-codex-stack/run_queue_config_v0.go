package orquestaappcodexstack

import "strings"

const (
	DefaultRunQueueRefV0            = "global"
	DefaultRunQueuePriorityScoreV0  = 50
	DefaultRunQueueMaxRunsPerTickV0 = 1
)

func normalizeRunQueueConfigV0(config RunQueueConfigV0) RunQueueConfigV0 {
	config.QueueRef = strings.TrimSpace(config.QueueRef)
	if config.QueueRef == "" {
		config.QueueRef = DefaultRunQueueRefV0
	}
	if config.DefaultPriorityScore <= 0 {
		config.DefaultPriorityScore = DefaultRunQueuePriorityScoreV0
	}
	if config.QueueLimit < 0 {
		config.QueueLimit = 0
	}
	if config.MaxRunsPerTick <= 0 {
		config.MaxRunsPerTick = DefaultRunQueueMaxRunsPerTickV0
	}
	return config
}
