package orquestaruntime

import "testing"

func TestRuntimeFakeLifecycleV0CicloCompleto(t *testing.T) {
	runtime := NewRuntimeFakeLifecycleV0()

	launched, err := runtime.LaunchAgentV0(agentLauncherInboundValidoV0())
	requireNoRuntimeFakeLifecycleErrorV0(t, err)
	if launched.Status != RuntimeFakeLifecycleLaunchedV0 {
		t.Fatalf("launch status = %q", launched.Status)
	}
	if launched.LaunchRef != "corr-agent-launcher-001" {
		t.Fatalf("launch_ref = %q", launched.LaunchRef)
	}

	progress := agentProgressReportValidoV0()
	progress.Status = AgentProgressingV0
	progress.ReportID = "agent-progress-report-ref-001"
	reported, err := runtime.ReportProgressV0(progress)
	requireNoRuntimeFakeLifecycleErrorV0(t, err)
	if reported.Status != RuntimeFakeLifecycleProgressingV0 {
		t.Fatalf("progress status = %q", reported.Status)
	}
	if reported.ProgressReports != 1 {
		t.Fatalf("progress_reports = %d", reported.ProgressReports)
	}

	loop := agentProgressReportValidoV0()
	loop.Status = AgentLoopDetectedV0
	loop.ReportID = "agent-progress-report-ref-loop-001"
	loop.NoProgressTicks = 4
	looped, err := runtime.ReportProgressV0(loop)
	requireNoRuntimeFakeLifecycleErrorV0(t, err)
	if looped.Status != RuntimeFakeLifecycleLoopDetectedV0 {
		t.Fatalf("loop status = %q", looped.Status)
	}
	if looped.LastReportRef != loop.ReportID {
		t.Fatalf("last_report_ref = %q", looped.LastReportRef)
	}
	if looped.ProgressReports != 2 {
		t.Fatalf("progress_reports = %d", looped.ProgressReports)
	}

	stopped, err := runtime.StopAgentV0(agentStopperInboundValidoV0())
	requireNoRuntimeFakeLifecycleErrorV0(t, err)
	if stopped.Status != RuntimeFakeLifecycleStoppedV0 {
		t.Fatalf("stop status = %q", stopped.Status)
	}
	if stopped.StopRef != "corr-agent-stopper-001" {
		t.Fatalf("stop_ref = %q", stopped.StopRef)
	}
}

func TestRuntimeFakeLifecycleV0RechazaProgressDeAgenteInexistente(t *testing.T) {
	runtime := NewRuntimeFakeLifecycleV0()

	_, err := runtime.ReportProgressV0(agentProgressReportValidoV0())

	requireRuntimeFakeLifecycleCodeV0(t, err, RuntimeFakeLifecycleAgentMissingV0)
}

func TestRuntimeFakeLifecycleV0RechazaStopDeAgenteInexistente(t *testing.T) {
	runtime := NewRuntimeFakeLifecycleV0()

	_, err := runtime.StopAgentV0(agentStopperInboundValidoV0())

	requireRuntimeFakeLifecycleCodeV0(t, err, RuntimeFakeLifecycleAgentMissingV0)
}

func TestRuntimeFakeLifecycleV0RechazaRunMismatchEnProgress(t *testing.T) {
	runtime := NewRuntimeFakeLifecycleV0()
	_, err := runtime.LaunchAgentV0(agentLauncherInboundValidoV0())
	requireNoRuntimeFakeLifecycleErrorV0(t, err)

	report := agentProgressReportValidoV0()
	report.RunID = "run-ref-distinto"
	_, err = runtime.ReportProgressV0(report)

	requireRuntimeFakeLifecycleCodeV0(t, err, RuntimeFakeLifecycleRunMismatchV0)
}

func TestRuntimeFakeLifecycleV0RechazaRunMismatchEnStop(t *testing.T) {
	runtime := NewRuntimeFakeLifecycleV0()
	_, err := runtime.LaunchAgentV0(agentLauncherInboundValidoV0())
	requireNoRuntimeFakeLifecycleErrorV0(t, err)

	stop := agentStopperInboundValidoV0()
	stop.Payload.RunID = "run-ref-distinto"
	_, err = runtime.StopAgentV0(stop)

	requireRuntimeFakeLifecycleCodeV0(t, err, RuntimeFakeLifecycleRunMismatchV0)
}

func TestRuntimeFakeLifecycleV0StopRepetidoEsIdempotente(t *testing.T) {
	runtime := NewRuntimeFakeLifecycleV0()
	_, err := runtime.LaunchAgentV0(agentLauncherInboundValidoV0())
	requireNoRuntimeFakeLifecycleErrorV0(t, err)

	first, err := runtime.StopAgentV0(agentStopperInboundValidoV0())
	requireNoRuntimeFakeLifecycleErrorV0(t, err)
	secondStop := agentStopperInboundValidoV0()
	secondStop.CorrelationID = "corr-agent-stopper-002"
	second, err := runtime.StopAgentV0(secondStop)
	requireNoRuntimeFakeLifecycleErrorV0(t, err)

	if second.Status != RuntimeFakeLifecycleStoppedV0 {
		t.Fatalf("status = %q", second.Status)
	}
	if second.StopRef != first.StopRef {
		t.Fatalf("stop repetido cambio stop_ref: first=%q second=%q", first.StopRef, second.StopRef)
	}
}

func TestRuntimeFakeLifecycleV0LaunchRepetidoNoReabreAgenteParado(t *testing.T) {
	runtime := NewRuntimeFakeLifecycleV0()
	_, err := runtime.LaunchAgentV0(agentLauncherInboundValidoV0())
	requireNoRuntimeFakeLifecycleErrorV0(t, err)
	stopped, err := runtime.StopAgentV0(agentStopperInboundValidoV0())
	requireNoRuntimeFakeLifecycleErrorV0(t, err)

	repeated, err := runtime.LaunchAgentV0(agentLauncherInboundValidoV0())
	requireNoRuntimeFakeLifecycleErrorV0(t, err)

	if repeated.Status != RuntimeFakeLifecycleStoppedV0 {
		t.Fatalf("launch repetido reabrio agente: %+v", repeated)
	}
	if repeated.StopRef != stopped.StopRef {
		t.Fatalf("launch repetido cambio stop_ref: stopped=%q repeated=%q", stopped.StopRef, repeated.StopRef)
	}
}

func TestRuntimeFakeLifecycleV0ProgressTardioNoReabreAgenteParado(t *testing.T) {
	runtime := NewRuntimeFakeLifecycleV0()
	_, err := runtime.LaunchAgentV0(agentLauncherInboundValidoV0())
	requireNoRuntimeFakeLifecycleErrorV0(t, err)
	stopped, err := runtime.StopAgentV0(agentStopperInboundValidoV0())
	requireNoRuntimeFakeLifecycleErrorV0(t, err)

	progress := agentProgressReportValidoV0()
	progress.ReportID = "agent-progress-report-ref-late-001"
	reported, err := runtime.ReportProgressV0(progress)
	requireNoRuntimeFakeLifecycleErrorV0(t, err)

	if reported.Status != RuntimeFakeLifecycleStoppedV0 {
		t.Fatalf("progress tardio reabrio agente: %+v", reported)
	}
	if reported.StopRef != stopped.StopRef {
		t.Fatalf("progress tardio cambio stop_ref: stopped=%q reported=%q", stopped.StopRef, reported.StopRef)
	}
}

func TestRuntimeFakeLifecycleV0NoAceptaInboundInvalido(t *testing.T) {
	runtime := NewRuntimeFakeLifecycleV0()
	launch := agentLauncherInboundValidoV0()
	launch.Payload = nil
	_, err := runtime.LaunchAgentV0(launch)
	requireRuntimeFakeLifecycleCodeV0(t, err, RuntimeFakeLifecycleInboundInvalidoV0)

	_, err = runtime.LaunchAgentV0(agentLauncherInboundValidoV0())
	requireNoRuntimeFakeLifecycleErrorV0(t, err)
	stop := agentStopperInboundValidoV0()
	stop.Payload = nil
	_, err = runtime.StopAgentV0(stop)
	requireRuntimeFakeLifecycleCodeV0(t, err, RuntimeFakeLifecycleInboundInvalidoV0)

	progress := agentProgressReportValidoV0()
	progress.Status = "esperando"
	_, err = runtime.ReportProgressV0(progress)
	requireRuntimeFakeLifecycleCodeV0(t, err, RuntimeFakeLifecycleProgressInvalidV0)
}

func requireNoRuntimeFakeLifecycleErrorV0(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
}

func requireRuntimeFakeLifecycleCodeV0(
	t *testing.T,
	err error,
	code RuntimeFakeLifecycleErrorCodeV0,
) {
	t.Helper()
	if err == nil {
		t.Fatalf("error nil, se esperaba codigo %q", code)
	}
	runtimeErr, ok := err.(RuntimeFakeLifecycleErrorV0)
	if !ok {
		t.Fatalf("tipo de error = %T, se esperaba RuntimeFakeLifecycleErrorV0", err)
	}
	if runtimeErr.Code != code {
		t.Fatalf("codigo = %q, se esperaba %q", runtimeErr.Code, code)
	}
}
