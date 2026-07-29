package config

import (
	"errors"
	"time"
)

const ErrorSelfWatchdogInvalid ErrorCode = "config_self_watchdog_invalid"

var (
	errSelfWatchdogObservationInterval = errors.New("self_watchdog.observation_interval")
	errSelfWatchdogCPUThreshold        = errors.New("self_watchdog.cpu_high_percent")
	errSelfWatchdogSustainedWindow     = errors.New("self_watchdog.sustained_for")
	errSelfWatchdogNoProgressWindow    = errors.New("self_watchdog.no_progress_for")
)

// SelfWatchdogPolicy is the typed, bounded policy consumed by bootstrap. It is
// not a configuration input authority and deliberately owns neither keys,
// environment aliases, nor defaults; registry projection belongs to the
// product wiring write-set.
type SelfWatchdogPolicy struct {
	Enabled             bool
	ObservationInterval time.Duration
	CPUHighPercent      float64
	SustainedFor        time.Duration
	NoProgressFor       time.Duration
}

// Validate rejects ambiguous active policies. A disabled policy accepts an
// empty projection so disabled compositions do not require unrelated values.
func (policy SelfWatchdogPolicy) Validate() error {
	if !policy.Enabled {
		return nil
	}
	switch {
	case policy.ObservationInterval <= 0:
		return selfWatchdogConfigError(errSelfWatchdogObservationInterval)
	case policy.CPUHighPercent <= 0 || policy.CPUHighPercent > 100:
		return selfWatchdogConfigError(errSelfWatchdogCPUThreshold)
	case policy.SustainedFor < policy.ObservationInterval:
		return selfWatchdogConfigError(errSelfWatchdogSustainedWindow)
	case policy.NoProgressFor < policy.ObservationInterval:
		return selfWatchdogConfigError(errSelfWatchdogNoProgressWindow)
	default:
		return nil
	}
}

func selfWatchdogConfigError(cause error) error {
	return &Error{Code: ErrorSelfWatchdogInvalid, Cause: cause}
}
