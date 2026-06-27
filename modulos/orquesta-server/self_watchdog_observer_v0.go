package orquestaserver

import (
	"context"
	"math"
	"runtime"
	"strings"
	"sync"
	"time"
)

type SelfWatchdogCPUSampleV0 struct {
	ProcessTicks uint64
	TotalTicks   uint64
}

type SelfWatchdogCPUSamplerV0 interface {
	SampleSelfWatchdogCPUV0(context.Context) (SelfWatchdogCPUSampleV0, error)
}

type ProcessSelfWatchdogObserverV0 struct {
	mu               sync.Mutex
	sampler          SelfWatchdogCPUSamplerV0
	lastSample       SelfWatchdogCPUSampleV0
	lastSampleOK     bool
	highCPUSince     time.Time
	lastProgress     selfWatchdogProgressCountersV0
	lastProgressAt   time.Time
	progressRestored bool
}

func NewProcessSelfWatchdogObserverV0(
	sampler SelfWatchdogCPUSamplerV0,
) *ProcessSelfWatchdogObserverV0 {
	if sampler == nil {
		sampler = NewProcSelfCPUSamplerV0()
	}
	return &ProcessSelfWatchdogObserverV0{sampler: sampler}
}

func (observer *ProcessSelfWatchdogObserverV0) ObserveSelfWatchdogV0(
	ctx context.Context,
	request SelfWatchdogObservationRequestV0,
) (SelfWatchdogObservationV0, error) {
	if observer == nil {
		observer = NewProcessSelfWatchdogObserverV0(nil)
	}
	config := NormalizeSelfWatchdogConfigV0(request.Config)
	now := request.ObservedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	observer.mu.Lock()
	defer observer.mu.Unlock()
	cpuPercent, cpuOK, err := observer.cpuPercentLockedV0(ctx)
	if err != nil {
		return SelfWatchdogObservationV0{}, err
	}
	if cpuPercent >= config.CPUHighPercent {
		if observer.highCPUSince.IsZero() {
			observer.highCPUSince = parseServerTimeV0(request.State.SelfWatchdogHighCPUSince)
			if observer.highCPUSince.IsZero() {
				observer.highCPUSince = now
			}
		}
	} else {
		observer.highCPUSince = time.Time{}
	}
	counters := selfWatchdogProgressCountersFromStateV0(request.State)
	observer.restoreProgressFromStateLockedV0(request.State, counters)
	if observer.lastProgress.progressedV0(counters) {
		observer.lastProgressAt = now
	}
	observer.lastProgress = counters
	observation := SelfWatchdogObservationV0{
		ObservedAt:                 now,
		CPUPercent:                 cpuPercent,
		HighCPUSince:               observer.highCPUSince,
		LastProgressAt:             observer.lastProgressAt,
		ActiveRuns:                 nonNegativeServerIntV0(request.State.LastSupervisorQueueSize),
		ActiveAgents:               nonNegativeServerIntV0(request.State.ShutdownAgentsInFlight),
		SupervisorTickActive:       request.State.SupervisorTickActive,
		ResidentDirectorTickActive: request.State.ResidentDirectorTickActive,
		GoalObserverTickActive:     request.State.GoalObserverTickActive,
		ExternalBridgeTickActive:   request.State.ExternalBridgeTickActive,
		AsyncWorkActive:            nonNegativeServerIntV0(request.State.ShutdownAsyncWorkActive),
		ShutdownInProgress:         request.State.ShutdownInProgress,
		EvidenceRefs:               selfWatchdogObservationEvidenceRefsV0(request.State, cpuOK),
	}
	return observation, nil
}

func (observer *ProcessSelfWatchdogObserverV0) cpuPercentLockedV0(
	ctx context.Context,
) (int, bool, error) {
	if observer.sampler == nil {
		observer.sampler = NewProcSelfCPUSamplerV0()
	}
	sample, err := observer.sampler.SampleSelfWatchdogCPUV0(ctx)
	if err != nil {
		observer.lastSampleOK = false
		return 0, false, err
	}
	if !observer.lastSampleOK ||
		sample.TotalTicks <= observer.lastSample.TotalTicks ||
		sample.ProcessTicks < observer.lastSample.ProcessTicks {
		observer.lastSample = sample
		observer.lastSampleOK = true
		return 0, true, nil
	}
	deltaProcess := sample.ProcessTicks - observer.lastSample.ProcessTicks
	deltaTotal := sample.TotalTicks - observer.lastSample.TotalTicks
	observer.lastSample = sample
	if deltaTotal == 0 {
		return 0, true, nil
	}
	cpu := math.Round(float64(deltaProcess) * float64(runtime.NumCPU()) * 100 / float64(deltaTotal))
	return nonNegativeServerIntV0(int(cpu)), true, nil
}

func (observer *ProcessSelfWatchdogObserverV0) restoreProgressFromStateLockedV0(
	state StateV0,
	counters selfWatchdogProgressCountersV0,
) {
	if observer.progressRestored {
		return
	}
	observer.progressRestored = true
	observer.lastProgress = counters
	if counters.SupervisorExecutions > 0 && state.LastSupervisorExecutions > 0 {
		observer.lastProgressAt = parseServerTimeV0(state.LastSupervisorAt)
	}
	if parsed := parseServerTimeV0(state.ExternalBridgeLastSuccess); parsed.After(observer.lastProgressAt) {
		observer.lastProgressAt = parsed
	}
	if parsed := parseServerTimeV0(state.ResidentDirectorLastSuccessAt); parsed.After(observer.lastProgressAt) {
		observer.lastProgressAt = parsed
	}
	if parsed := parseServerTimeV0(state.GoalObserverLastSuccessAt); parsed.After(observer.lastProgressAt) {
		observer.lastProgressAt = parsed
	}
}

type selfWatchdogProgressCountersV0 struct {
	SupervisorExecutions     int
	IdleSelfImprovementRuns  int
	IdleSelfImprovementOK    int
	ExternalBridgeTicks      int
	ExternalBridgeErrorTicks int
	ResidentDirectorTicks    int
	ResidentDirectorErrors   int
	ResidentDirectorActions  int
	GoalObserverTicks        int
	GoalObserverErrors       int
}

func selfWatchdogProgressCountersFromStateV0(state StateV0) selfWatchdogProgressCountersV0 {
	return selfWatchdogProgressCountersV0{
		SupervisorExecutions:     nonNegativeServerIntV0(state.SupervisorExecutions),
		IdleSelfImprovementRuns:  nonNegativeServerIntV0(state.IdleSelfImprovementRuns),
		IdleSelfImprovementOK:    nonNegativeServerIntV0(state.IdleSelfImprovementOK),
		ExternalBridgeTicks:      nonNegativeServerIntV0(state.ExternalBridgeTicks),
		ExternalBridgeErrorTicks: nonNegativeServerIntV0(state.ExternalBridgeErrorTicks),
		ResidentDirectorTicks:    nonNegativeServerIntV0(state.ResidentDirectorTicks),
		ResidentDirectorErrors:   nonNegativeServerIntV0(state.ResidentDirectorErrorTicks),
		ResidentDirectorActions:  nonNegativeServerIntV0(state.ResidentDirectorExecutedActions),
		GoalObserverTicks:        nonNegativeServerIntV0(state.GoalObserverTicks),
		GoalObserverErrors:       nonNegativeServerIntV0(state.GoalObserverErrorTicks),
	}
}

func (counters selfWatchdogProgressCountersV0) progressedV0(next selfWatchdogProgressCountersV0) bool {
	return next.SupervisorExecutions > counters.SupervisorExecutions ||
		next.IdleSelfImprovementRuns > counters.IdleSelfImprovementRuns ||
		next.IdleSelfImprovementOK > counters.IdleSelfImprovementOK ||
		next.ExternalBridgeTicks > counters.ExternalBridgeTicks ||
		next.ExternalBridgeErrorTicks > counters.ExternalBridgeErrorTicks ||
		next.ResidentDirectorTicks > counters.ResidentDirectorTicks ||
		next.ResidentDirectorErrors > counters.ResidentDirectorErrors ||
		next.ResidentDirectorActions > counters.ResidentDirectorActions ||
		next.GoalObserverTicks > counters.GoalObserverTicks ||
		next.GoalObserverErrors > counters.GoalObserverErrors
}

func selfWatchdogObservationEvidenceRefsV0(state StateV0, cpuOK bool) []string {
	refs := []string{"evidence-ref-self-watchdog-state"}
	if cpuOK {
		refs = append(refs, "evidence-ref-self-watchdog-proc-cpu")
	} else {
		refs = append(refs, "evidence-ref-self-watchdog-cpu-unavailable")
	}
	if state.LastSupervisorQueueSize > 0 {
		refs = append(refs, "evidence-ref-self-watchdog-active-queue")
	}
	if state.SupervisorTickActive {
		refs = append(refs, "evidence-ref-self-watchdog-supervisor-active")
	}
	if state.ResidentDirectorTickActive {
		refs = append(refs, "evidence-ref-self-watchdog-resident-director-active")
	}
	if state.GoalObserverTickActive {
		refs = append(refs, "evidence-ref-self-watchdog-goal-observer-active")
	}
	if state.ExternalBridgeTickActive {
		refs = append(refs, "evidence-ref-self-watchdog-external-bridge-active")
	}
	if state.ShutdownAsyncWorkActive > 0 {
		refs = append(refs, "evidence-ref-self-watchdog-async-work-active")
	}
	if state.ShutdownInProgress {
		refs = append(refs, "evidence-ref-self-watchdog-shutdown-active")
	}
	return compactServerStringsV0(refs)
}

func parseServerTimeV0(value string) time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}
