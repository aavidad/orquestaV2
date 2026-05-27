package orquestaserver

import (
	"os"
	"time"
)

func NewStatusTrackerV0(config ConfigV0, now time.Time) *StatusTrackerV0 {
	config = NormalizeConfigV0(config)
	identity := NewDaemonIdentityV0(os.Getpid(), now)
	return &StatusTrackerV0{state: StateV0{
		SchemaVersion:             StateSchemaVersionV0,
		Status:                    "starting",
		PID:                       os.Getpid(),
		Addr:                      config.Addr,
		ProcessRef:                identity.ProcessRef,
		DaemonEpochRef:            identity.DaemonEpochRef,
		ProjectWorkDir:            config.ProjectWorkDir,
		RuntimeWorkDir:            config.RuntimeWorkDir,
		DaemonLogPolicy:           config.DaemonLogPolicy,
		ShutdownSignalPolicy:      config.ShutdownSignalPolicy,
		EffectiveConfig:           config.EffectiveConfig,
		StartedAt:                 formatTimeV0(now),
		LastHeartbeatAt:           formatTimeV0(now),
		IdleSelfImprovementAfter:  config.IdleSelfImprovementAfter.String(),
		IdleSelfImprovementTarget: config.IdleSelfImprovementTargetQueue,
	}}
}
