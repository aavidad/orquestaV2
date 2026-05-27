package orquestaruntime

import (
	"context"
	"os"
	"time"
)

const (
	defaultProcessRuntimeLaunchTimeoutV0 = 10 * time.Second
	defaultProcessRuntimeStopTimeoutV0   = 10 * time.Second
	defaultProcessRuntimeKillWaitV0      = 2 * time.Second
)

type ProcessRuntimeEffectPolicyV0 struct {
	LaunchTimeout time.Duration
	StopTimeout   time.Duration
	KillWait      time.Duration
	StopSignal    os.Signal
}

func NewProcessRuntimeConnectorWithPolicyV0(
	policy ProcessRuntimeEffectPolicyV0,
) *ProcessRuntimeConnectorV0 {
	connector := NewProcessRuntimeConnectorV0()
	connector.effectPolicy = normalizeProcessRuntimeEffectPolicyV0(policy)
	return connector
}

func normalizeProcessRuntimeEffectPolicyV0(
	policy ProcessRuntimeEffectPolicyV0,
) ProcessRuntimeEffectPolicyV0 {
	if policy.LaunchTimeout <= 0 {
		policy.LaunchTimeout = defaultProcessRuntimeLaunchTimeoutV0
	}
	if policy.StopTimeout <= 0 {
		policy.StopTimeout = defaultProcessRuntimeStopTimeoutV0
	}
	if policy.KillWait <= 0 {
		policy.KillWait = defaultProcessRuntimeKillWaitV0
	}
	if policy.StopSignal == nil {
		policy.StopSignal = os.Interrupt
	}
	return policy
}

func (c *ProcessRuntimeConnectorV0) normalizedEffectPolicyV0() ProcessRuntimeEffectPolicyV0 {
	if c == nil {
		return normalizeProcessRuntimeEffectPolicyV0(ProcessRuntimeEffectPolicyV0{})
	}
	return normalizeProcessRuntimeEffectPolicyV0(c.effectPolicy)
}

func processRuntimeEffectContextV0(
	ctx context.Context,
	timeout time.Duration,
) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}
