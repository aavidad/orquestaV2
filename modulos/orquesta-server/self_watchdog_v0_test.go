package orquestaserver

import (
	"context"
	"testing"
	"time"
)

func TestEvaluateSelfWatchdogV0NoParaSiHayProgresoReciente(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	decision := EvaluateSelfWatchdogV0(SelfWatchdogConfigV0{}, SelfWatchdogObservationV0{
		ObservedAt:     now,
		CPUPercent:     95,
		HighCPUSince:   now.Add(-10 * time.Minute),
		LastProgressAt: now.Add(-20 * time.Second),
		EvidenceRefs:   []string{"evidence-ref-cpu-progress"},
	})

	if decision.ShouldRequestShutdown ||
		decision.Status != SelfWatchdogStatusHighCPUWithCauseV0 ||
		decision.ReasonCode != SelfWatchdogReasonRecentProgressV0 {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestEvaluateSelfWatchdogV0NoParaSiHayTrabajoCausal(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 5, 0, 0, time.UTC)
	decision := EvaluateSelfWatchdogV0(SelfWatchdogConfigV0{}, SelfWatchdogObservationV0{
		ObservedAt:          now,
		CPUPercent:          95,
		HighCPUSince:        now.Add(-10 * time.Minute),
		ActiveRuns:          1,
		PendingOutbox:       2,
		ActiveAgents:        3,
		RegisteredProcesses: 4,
	})

	if decision.ShouldRequestShutdown ||
		decision.Status != SelfWatchdogStatusHighCPUWithCauseV0 ||
		decision.ReasonCode != SelfWatchdogReasonOperationalCauseV0 {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestEvaluateSelfWatchdogV0NoParaSiDirectorResidenteActivo(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 7, 0, 0, time.UTC)
	decision := EvaluateSelfWatchdogV0(SelfWatchdogConfigV0{}, SelfWatchdogObservationV0{
		ObservedAt:                 now,
		CPUPercent:                 95,
		HighCPUSince:               now.Add(-10 * time.Minute),
		ResidentDirectorTickActive: true,
		EvidenceRefs:               []string{"evidence-ref-resident-director-active"},
	})

	if decision.ShouldRequestShutdown ||
		decision.Status != SelfWatchdogStatusHighCPUWithCauseV0 ||
		decision.ReasonCode != SelfWatchdogReasonOperationalCauseV0 {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestEvaluateSelfWatchdogV0PideParadaSiCPUSostenidaSinCausa(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 10, 0, 0, time.UTC)
	decision := EvaluateSelfWatchdogV0(SelfWatchdogConfigV0{}, SelfWatchdogObservationV0{
		ObservedAt:     now,
		CPUPercent:     95,
		HighCPUSince:   now.Add(-10 * time.Minute),
		LastProgressAt: now.Add(-10 * time.Minute),
		EvidenceRefs:   []string{"evidence-ref-cpu-no-progress"},
	})

	if !decision.ShouldRequestShutdown ||
		decision.Status != SelfWatchdogStatusShutdownRequestedV0 ||
		decision.ReasonCode != SelfWatchdogReasonNoProgressV0 {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestStatusTrackerSelfWatchdogV0ProyectaUnhealthyYReadiness(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 15, 0, 0, time.UTC)
	tracker := NewStatusTrackerV0(ConfigV0{}, now)
	state := tracker.MarkServingV0("127.0.0.1:8787", now)
	state.StartupReady = true
	state.StartupStatus = StartupCheckStatusReadyV0
	tracker = &StatusTrackerV0{state: state}

	decision := EvaluateSelfWatchdogV0(SelfWatchdogConfigV0{}, SelfWatchdogObservationV0{
		ObservedAt:   now,
		CPUPercent:   99,
		HighCPUSince: now.Add(-5 * time.Minute),
		EvidenceRefs: []string{"evidence-ref-self-watchdog"},
	})
	state = tracker.MarkSelfWatchdogV0(decision, now)

	status := NewServerPublicStatusV0(state)
	readiness := NewServerReadinessV0(state)
	health := residentOperationalHealthV0(state)
	if status.Status != "unhealthy" || !status.SelfWatchdogShutdownRequested {
		t.Fatalf("status=%+v", status)
	}
	if readiness.Ready {
		t.Fatalf("readiness=%+v", readiness)
	}
	foundHealth := false
	for _, check := range health {
		if check.I18nKey == "server.health.self_watchdog" &&
			check.Severity == "error" &&
			check.Estado == "blocked" {
			foundHealth = true
		}
	}
	if !foundHealth {
		t.Fatalf("health=%+v", health)
	}
}

func TestRuntimeSelfWatchdogV0ApagaServidorSinTrabajo(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 20, 0, 0, time.UTC)
	store := &threadSafeStateStoreV0{}
	observer := &fakeSelfWatchdogObserverV0{observation: SelfWatchdogObservationV0{
		ObservedAt:   now,
		CPUPercent:   99,
		HighCPUSince: now.Add(-10 * time.Minute),
		EvidenceRefs: []string{"evidence-ref-runtime-watchdog"},
	}}
	runtime, err := NewRuntimeV0(ConfigV0{
		Addr:                "127.0.0.1:0",
		StateDir:            t.TempDir(),
		AuditDisabled:       true,
		TickInterval:        10 * time.Millisecond,
		ShutdownGracePeriod: time.Second,
		SelfWatchdog: SelfWatchdogConfigV0{
			SustainedFor:  time.Nanosecond,
			NoProgressFor: time.Nanosecond,
		},
	}, RuntimeDepsV0{
		StateStore:   store,
		Clock:        fixedClockV0{now: now},
		SelfWatchdog: observer,
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- runtime.RunV0(context.Background()) }()
	if err := waitRuntimeDoneV0(t, done); err != nil {
		t.Fatalf("RunV0: %v", err)
	}
	state := store.LastV0()
	if observer.calls == 0 {
		t.Fatalf("watchdog no observo")
	}
	if state.Status != "stopped" ||
		state.ShutdownSignalName != "self_watchdog" ||
		!state.SelfWatchdogShutdownRequested {
		t.Fatalf("state=%+v", state)
	}
}

func TestProcessSelfWatchdogObserverV0DerivaCPUYVentanaAlta(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 25, 0, 0, time.UTC)
	observer := NewProcessSelfWatchdogObserverV0(&fakeSelfWatchdogCPUSamplerV0{
		samples: []SelfWatchdogCPUSampleV0{
			{ProcessTicks: 100, TotalTicks: 1000},
			{ProcessTicks: 300, TotalTicks: 1100},
		},
	})
	config := SelfWatchdogConfigV0{CPUHighPercent: 50, SustainedFor: time.Minute, NoProgressFor: time.Minute}
	first, err := observer.ObserveSelfWatchdogV0(context.Background(), SelfWatchdogObservationRequestV0{
		Config:     config,
		ObservedAt: now,
	})
	if err != nil {
		t.Fatalf("Observe first: %v", err)
	}
	second, err := observer.ObserveSelfWatchdogV0(context.Background(), SelfWatchdogObservationRequestV0{
		Config:     config,
		ObservedAt: now.Add(5 * time.Second),
	})
	if err != nil {
		t.Fatalf("Observe second: %v", err)
	}
	if first.CPUPercent != 0 {
		t.Fatalf("first cpu=%d want 0 sin delta previa", first.CPUPercent)
	}
	if second.CPUPercent < config.CPUHighPercent || second.HighCPUSince.IsZero() {
		t.Fatalf("second=%+v", second)
	}
	decision := EvaluateSelfWatchdogV0(config, second)
	if decision.ShouldRequestShutdown || decision.ReasonCode != SelfWatchdogReasonSustainedWindowOpenV0 {
		t.Fatalf("decision=%+v", decision)
	}
}

func TestProcessSelfWatchdogObserverV0ReconoceProgresoPorEjecuciones(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 30, 0, 0, time.UTC)
	observer := NewProcessSelfWatchdogObserverV0(&fakeSelfWatchdogCPUSamplerV0{
		samples: []SelfWatchdogCPUSampleV0{
			{ProcessTicks: 100, TotalTicks: 1000},
			{ProcessTicks: 300, TotalTicks: 1100},
		},
	})
	config := SelfWatchdogConfigV0{CPUHighPercent: 50, SustainedFor: time.Second, NoProgressFor: time.Minute}
	_, err := observer.ObserveSelfWatchdogV0(context.Background(), SelfWatchdogObservationRequestV0{
		State:      StateV0{SupervisorExecutions: 1},
		Config:     config,
		ObservedAt: now,
	})
	if err != nil {
		t.Fatalf("Observe first: %v", err)
	}
	second, err := observer.ObserveSelfWatchdogV0(context.Background(), SelfWatchdogObservationRequestV0{
		State:      StateV0{SupervisorExecutions: 2},
		Config:     config,
		ObservedAt: now.Add(5 * time.Second),
	})
	if err != nil {
		t.Fatalf("Observe second: %v", err)
	}
	if second.LastProgressAt.IsZero() {
		t.Fatalf("sin progreso observable: %+v", second)
	}
	decision := EvaluateSelfWatchdogV0(config, second)
	if decision.ShouldRequestShutdown || decision.ReasonCode != SelfWatchdogReasonRecentProgressV0 {
		t.Fatalf("decision=%+v observation=%+v", decision, second)
	}
}

func TestProcessSelfWatchdogObserverV0TransportaDirectorResidenteActivo(t *testing.T) {
	now := time.Date(2026, 6, 8, 12, 35, 0, 0, time.UTC)
	observer := NewProcessSelfWatchdogObserverV0(&fakeSelfWatchdogCPUSamplerV0{
		samples: []SelfWatchdogCPUSampleV0{
			{ProcessTicks: 100, TotalTicks: 1000},
			{ProcessTicks: 300, TotalTicks: 1100},
		},
	})
	config := SelfWatchdogConfigV0{CPUHighPercent: 50, SustainedFor: time.Second, NoProgressFor: time.Second}
	_, err := observer.ObserveSelfWatchdogV0(context.Background(), SelfWatchdogObservationRequestV0{
		State:      StateV0{ResidentDirectorTickActive: true},
		Config:     config,
		ObservedAt: now,
	})
	if err != nil {
		t.Fatalf("Observe first: %v", err)
	}
	second, err := observer.ObserveSelfWatchdogV0(context.Background(), SelfWatchdogObservationRequestV0{
		State:      StateV0{ResidentDirectorTickActive: true},
		Config:     config,
		ObservedAt: now.Add(5 * time.Second),
	})
	if err != nil {
		t.Fatalf("Observe second: %v", err)
	}
	if !second.ResidentDirectorTickActive ||
		!containsStringForTestV0(second.EvidenceRefs, "evidence-ref-self-watchdog-resident-director-active") {
		t.Fatalf("observation=%+v", second)
	}
	decision := EvaluateSelfWatchdogV0(config, second)
	if decision.ShouldRequestShutdown || decision.ReasonCode != SelfWatchdogReasonOperationalCauseV0 {
		t.Fatalf("decision=%+v observation=%+v", decision, second)
	}
}

func TestProcSelfWatchdogSamplerV0ParseaProcStatConNombreConEspacios(t *testing.T) {
	processTicks, err := parseProcSelfStatTicksV0(
		"123 (orquesta server) S 1 2 3 4 5 6 7 8 9 10 21 34 17",
	)
	if err != nil {
		t.Fatalf("parseProcSelfStatTicksV0: %v", err)
	}
	totalTicks, err := parseProcStatTotalTicksV0("cpu  10 20 30 40 5\ncpu0 1 2 3 4\n")
	if err != nil {
		t.Fatalf("parseProcStatTotalTicksV0: %v", err)
	}
	if processTicks != 55 || totalTicks != 105 {
		t.Fatalf("ticks process=%d total=%d", processTicks, totalTicks)
	}
}

type fakeSelfWatchdogObserverV0 struct {
	observation SelfWatchdogObservationV0
	calls       int
}

func (observer *fakeSelfWatchdogObserverV0) ObserveSelfWatchdogV0(
	_ context.Context,
	request SelfWatchdogObservationRequestV0,
) (SelfWatchdogObservationV0, error) {
	observer.calls++
	observation := observer.observation
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = request.ObservedAt
	}
	return observation, nil
}

type fakeSelfWatchdogCPUSamplerV0 struct {
	samples []SelfWatchdogCPUSampleV0
	index   int
}

func (sampler *fakeSelfWatchdogCPUSamplerV0) SampleSelfWatchdogCPUV0(
	context.Context,
) (SelfWatchdogCPUSampleV0, error) {
	if sampler.index >= len(sampler.samples) {
		return sampler.samples[len(sampler.samples)-1], nil
	}
	out := sampler.samples[sampler.index]
	sampler.index++
	return out, nil
}
