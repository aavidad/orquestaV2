package orquestaruntime

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	orquestaworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func TestRuntimeNeutralE2EV0LaunchProgressStopSinProviderConcreto(t *testing.T) {
	spec := runtimeNeutralExternalAgentSpecFromInboundForTestV0(t)
	req := processRuntimeLaunchRequestForTestV0(t, "wait")
	connector := NewProcessRuntimeConnectorV0()

	launch := LaunchExternalAgentProcessV0(
		context.Background(),
		spec,
		fakeExternalAgentProcessResolverV0{req: req},
		connector,
	)
	requireExternalAgentProcessStatusV0(t, launch, ExternalAgentProcessLaunchStartedV0)
	if launch.Snapshot.Status != ProcessRuntimeRunningV0 {
		t.Fatalf("launch status=%q", launch.Snapshot.Status)
	}

	progress := runtimeNeutralHeartbeatReportForTestV0(
		t,
		"agent-progress-report-ref-t08-progress",
		launch.Snapshot,
		nil,
		AgentProgressHeartbeatV0{
			HeartbeatRef:    "heartbeat-ref-t08-progress",
			RunID:           "run-ref-t08-runtime-neutral",
			AgentRequestID:  spec.RequestID,
			ProcessRef:      launch.Snapshot.ProcessRef,
			TickCounter:     1,
			ProgressCounter: 1,
			EvidenceRefs:    []string{"runtime-neutral-evidence-ref-progress"},
		},
	)
	if progress.Status != AgentProgressingV0 {
		t.Fatalf("progress status=%q", progress.Status)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	stopped, err := connector.StopV0(ctx, launch.Snapshot.ProcessRef)
	requireNoProcessRuntimeErrorV0(t, err)
	if stopped.Status != ProcessRuntimeStoppedV0 || stopped.StopRef == "" {
		t.Fatalf("stop snapshot invalido: %+v", stopped)
	}

	stopReport := runtimeNeutralHeartbeatReportForTestV0(
		t,
		"agent-progress-report-ref-t08-stop",
		stopped,
		&AgentProgressHeartbeatV0{
			HeartbeatRef:    "heartbeat-ref-t08-progress",
			RunID:           progress.RunID,
			AgentRequestID:  progress.AgentRequestID,
			ProcessRef:      launch.Snapshot.ProcessRef,
			TickCounter:     1,
			ProgressCounter: 1,
		},
		AgentProgressHeartbeatV0{
			HeartbeatRef:    "heartbeat-ref-t08-stop",
			RunID:           progress.RunID,
			AgentRequestID:  progress.AgentRequestID,
			ProcessRef:      stopped.ProcessRef,
			TickCounter:     2,
			ProgressCounter: 1,
			EvidenceRefs:    []string{"runtime-neutral-evidence-ref-stop"},
		},
	)
	if stopReport.Status != AgentStoppedV0 {
		t.Fatalf("stop report status=%q", stopReport.Status)
	}
	if !runtimeNeutralRefsContainV0(stopReport.EvidenceRefs, stopped.StopRef) {
		t.Fatalf("stop_ref no viaja como evidencia: report=%+v stop=%+v", stopReport, stopped)
	}

	assertRuntimeNeutralE2EWorktreeWriteSetV0(t, spec)
	assertRuntimeNeutralE2ENoOperationalLeakV0(t, spec, launch, progress, stopReport, req)
}

func runtimeNeutralExternalAgentSpecFromInboundForTestV0(t *testing.T) ExternalAgentLaunchSpecV0 {
	t.Helper()
	base := runtimeLaunchRequestValidaV0()
	inbound := agentLauncherInboundValidoV0()
	request, issues := LaunchRuntimeAgentToRuntimeLaunchRequestV0(
		inbound,
		AgentLauncherResolvedDependenciesV0{
			FunctionContract: base.FunctionContract,
			CapacityDecision: base.CapacityDecision,
			RuntimeBinding:   base.RuntimeBinding,
			EvidenceRefs:     base.EvidenceRefs,
			ContextBundle:    base.ContextBundle,
		},
		AgentLauncherRuntimeLaunchOptionsV0{
			Locale:                  "es-ES",
			AdapterRef:              "adapter-ref-runtime-neutral-e2e",
			RequestedAt:             "2026-05-24T10:00:00Z",
			ReadinessTimeoutSeconds: 30,
			MaxStartupSeconds:       60,
		},
	)
	if len(issues) != 0 {
		t.Fatalf("runtime launch request issues: %#v", issues)
	}
	packet := BuildAgentStartPacketV0(
		request,
		runtimeMaterializedContextValidoV0(t, *request.ContextBundle),
	)
	profile := BuildClosedExternalAgentConnectorProfileV0(ExternalAgentConnectorProfileRefsV0{
		ProfileRef:    "external-profile-ref-runtime-neutral-e2e",
		ConnectorRef:  request.RuntimeBinding.ConnectorRef,
		RuntimeKind:   request.RuntimeBinding.RuntimeKind,
		CommandRef:    "command-ref-runtime-neutral-e2e",
		ExecutableRef: "executable-ref-runtime-neutral-e2e",
		WorkingDirRef: "working-dir-ref-runtime-neutral-e2e",
		ArgRefs:       []string{"arg-ref-runtime-neutral-e2e"},
		EnvRefs:       []string{"env-ref-runtime-neutral-e2e"},
	})
	spec := BuildExternalAgentLaunchSpecV0(request, packet, profile)
	if !spec.Valid() {
		t.Fatalf("spec runtime-neutral invalida: %+v", spec.Issues)
	}
	return spec
}

func runtimeNeutralHeartbeatReportForTestV0(
	t *testing.T,
	reportRef string,
	snapshot ProcessRuntimeSnapshotV0,
	previous *AgentProgressHeartbeatV0,
	current AgentProgressHeartbeatV0,
) AgentProgressReportV0 {
	t.Helper()
	report, issues := BuildAgentProgressReportFromHeartbeatV0(
		reportRef,
		snapshot,
		previous,
		current,
		AgentProgressHeartbeatPolicyV0{
			StalledAfterNoProgressTicks: 2,
			LoopAfterRepeatedActions:    3,
		},
	)
	if len(issues) != 0 {
		t.Fatalf("progress issues inesperados: %#v", issues)
	}
	return report
}

func assertRuntimeNeutralE2ENoOperationalLeakV0(
	t *testing.T,
	spec ExternalAgentLaunchSpecV0,
	launch ExternalAgentProcessLaunchResultV0,
	progress AgentProgressReportV0,
	stopReport AgentProgressReportV0,
	req ProcessRuntimeLaunchRequestV0,
) {
	t.Helper()
	raw, err := json.Marshal(struct {
		Spec       ExternalAgentLaunchSpecV0          `json:"spec"`
		Launch     ExternalAgentProcessLaunchResultV0 `json:"launch"`
		Progress   AgentProgressReportV0              `json:"progress"`
		StopReport AgentProgressReportV0              `json:"stop_report"`
	}{spec, launch, progress, stopReport})
	if err != nil {
		t.Fatalf("marshal e2e neutral: %v", err)
	}
	text := strings.ToLower(string(raw))
	for _, forbidden := range []string{
		"codex",
		"provider-ref",
		"model-ref",
		"home-ref",
		"credential-ref",
		strings.ToLower(req.CommandPath),
		strings.ToLower(req.WorkingDir),
		"pid",
		"oauth",
		"token",
	} {
		if forbidden != "" && strings.Contains(text, forbidden) {
			t.Fatalf("e2e filtra detalle operacional %q: %s", forbidden, raw)
		}
	}
}

func runtimeNeutralRefsContainV0(refs []string, want string) bool {
	for _, ref := range refs {
		if ref == want {
			return true
		}
	}
	return false
}

func assertRuntimeNeutralE2EWorktreeWriteSetV0(
	t *testing.T,
	spec ExternalAgentLaunchSpecV0,
) {
	t.Helper()
	root := t.TempDir()
	baseline, issues := orquestaworktree.CaptureWorktreeSnapshotV0(
		context.Background(),
		orquestaworktree.WorktreeSnapshotRequestV0{
			SnapshotRef:    "snapshot-ref-runtime-neutral-e2e-base",
			ProjectWorkDir: root,
		},
	)
	if len(issues) > 0 {
		t.Fatalf("baseline worktree issues: %+v", issues)
	}

	req := processRuntimeLaunchRequestForTestV0(t, "envdump")
	req.WorkingDir = root
	connector := NewProcessRuntimeConnectorV0()
	launch := LaunchExternalAgentProcessV0(
		context.Background(),
		spec,
		fakeExternalAgentProcessResolverV0{req: req},
		connector,
	)
	requireExternalAgentProcessStatusV0(t, launch, ExternalAgentProcessLaunchStartedV0)
	waitForProcessRuntimeStatusV0(t, connector, launch.Snapshot.ProcessRef, ProcessRuntimeStoppedV0)

	result, issues := orquestaworktree.VerifyWorktreeWriteSetV0(
		context.Background(),
		orquestaworktree.WorktreeVerifyRequestV0{
			Baseline:       baseline,
			ProjectWorkDir: root,
			WriteSet:       []string{"env.txt"},
		},
	)
	if len(issues) > 0 || !result.OK {
		t.Fatalf("worktree verification result=%+v issues=%+v", result, issues)
	}
	if len(result.ChangedPaths) != 1 || result.ChangedPaths[0] != "env.txt" {
		t.Fatalf("worktree changed_paths=%v", result.ChangedPaths)
	}

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal worktree result: %v", err)
	}
	if strings.Contains(string(raw), root) {
		t.Fatalf("worktree result filtra path absoluto: %s", raw)
	}
}
