package main

import "time"

const (
	externalBridgeStatusRetryScheduledV0       = "retry_scheduled"
	externalBridgeStatusRetryBudgetExhaustedV0 = "retry_budget_exhausted"
	externalBridgeStatusRateLimitedV0          = "rate_limited"
)

type externalBridgeRetryPolicyV0 struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Jitter      func(int, time.Duration) time.Duration
}

type externalBridgeRetryOutcomeV0 struct {
	Event externalBridgeLoopEventV0
	Delay time.Duration
}

func externalBridgeRetryEventV0(
	policy externalBridgeRetryPolicyV0,
	consecutiveErrors int,
	fallbackDelay time.Duration,
	errorCode string,
) externalBridgeRetryOutcomeV0 {
	policy = normalizeExternalBridgeRetryPolicyV0(policy, fallbackDelay)
	if policy.MaxAttempts > 0 && consecutiveErrors >= policy.MaxAttempts {
		return externalBridgeRetryOutcomeV0{
			Event: externalBridgeLoopEventV0{
				Status:     externalBridgeStatusRetryBudgetExhaustedV0,
				ErrorCode:  "retry_budget_exhausted",
				StopReason: errorCode,
			},
			Delay: fallbackDelay,
		}
	}
	delay := externalBridgeRetryDelayV0(policy, consecutiveErrors)
	status := externalBridgeStatusRetryScheduledV0
	if errorCode == "rate_limited" {
		status = externalBridgeStatusRateLimitedV0
	}
	return externalBridgeRetryOutcomeV0{
		Event: externalBridgeLoopEventV0{
			Status:    status,
			ErrorCode: firstExternalBridgeValueV0(errorCode, "retry_scheduled"),
		},
		Delay: delay,
	}
}

func normalizeExternalBridgeRetryPolicyV0(
	policy externalBridgeRetryPolicyV0,
	fallbackDelay time.Duration,
) externalBridgeRetryPolicyV0 {
	if policy.BaseDelay <= 0 {
		policy.BaseDelay = fallbackDelay
	}
	if policy.BaseDelay <= 0 {
		policy.BaseDelay = time.Second
	}
	if policy.MaxDelay > 0 && policy.BaseDelay > policy.MaxDelay {
		policy.BaseDelay = policy.MaxDelay
	}
	return policy
}

func externalBridgeRetryDelayV0(
	policy externalBridgeRetryPolicyV0,
	consecutiveErrors int,
) time.Duration {
	if consecutiveErrors < 1 {
		consecutiveErrors = 1
	}
	delay := policy.BaseDelay
	for i := 1; i < consecutiveErrors; i++ {
		delay *= 2
		if policy.MaxDelay > 0 && delay >= policy.MaxDelay {
			delay = policy.MaxDelay
			break
		}
	}
	if policy.Jitter != nil {
		delay = policy.Jitter(consecutiveErrors, delay)
		if delay < 0 {
			delay = 0
		}
	}
	return delay
}
