package orquestaserver

import "time"

const (
	SelfWatchdogStatusDisabledV0            = "disabled"
	SelfWatchdogStatusOKV0                  = "ok"
	SelfWatchdogStatusHighCPUWithCauseV0    = "high_cpu_with_cause"
	SelfWatchdogStatusObservingHighCPUV0    = "observing_high_cpu"
	SelfWatchdogStatusShutdownRequestedV0   = "shutdown_requested"
	SelfWatchdogReasonDisabledV0            = "self_watchdog_disabled"
	SelfWatchdogReasonCPUWithinLimitV0      = "cpu_within_limit"
	SelfWatchdogReasonOperationalCauseV0    = "operational_cause_present"
	SelfWatchdogReasonRecentProgressV0      = "recent_progress_observed"
	SelfWatchdogReasonSustainedWindowOpenV0 = "sustained_window_open"
	SelfWatchdogReasonNoProgressV0          = "high_cpu_without_progress"
)

type SelfWatchdogConfigV0 struct {
	Disabled       bool
	CPUHighPercent int
	SustainedFor   time.Duration
	NoProgressFor  time.Duration
}

type SelfWatchdogObservationV0 struct {
	ObservedAt                 time.Time
	CPUPercent                 int
	HighCPUSince               time.Time
	LastProgressAt             time.Time
	ActiveRuns                 int
	PendingOutbox              int
	ActiveAgents               int
	RegisteredProcesses        int
	SupervisorTickActive       bool
	ResidentDirectorTickActive bool
	ExternalBridgeTickActive   bool
	AsyncWorkActive            int
	ShutdownInProgress         bool
	EvidenceRefs               []string
}

type SelfWatchdogDecisionV0 struct {
	Status                string
	ReasonCode            string
	ObservedAt            string
	HighCPUSince          string
	CPUPercent            int
	ShouldRequestShutdown bool
	EvidenceRefs          []string
}
