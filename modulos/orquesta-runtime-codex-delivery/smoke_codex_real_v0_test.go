package orquestaruntimecodexdelivery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorscheduler "orquesta/modulos/orquesta-director-scheduler"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestCodexReceiptDeliveryLoopV0SmokeCodexRealOptIn(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_SMOKE")) != "1" {
		t.Skip("set ORQUESTA_CODEX_SMOKE=1 para lanzar Codex real")
	}
	cfg := codexRealSmokeConfigForTestV0(t)
	spec := codexRealSmokeSpecForTestV0("agent-ref-real-smoke-001", "task-ref-real-smoke-001")
	if err := os.WriteFile(filepath.Join(cfg.ProjectWorkDir, "README.md"), []byte("# Agenda smoke\n"), 0o600); err != nil {
		t.Fatalf("write README.md: %v", err)
	}

	runRef := "run-ref-real-smoke-001"
	receiptStore := NewInMemoryCodexReceiptDescriptorStoreV0()
	run := codexDeliveryLoopRunForTestV0(t, runRef, spec.AgentPacket.Task.TaskRef)
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	processRegistry := orquestacionnucleoapp.NewInMemoryAgentProcessRegistryV0()
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	service := codexRealSmokeServiceForTestV0(
		store,
		sink,
		ledger,
		receiptStore,
		runRef,
		spec,
	)
	dispatchers := codexRealSmokeDispatchersForTestV0(
		store,
		sink,
		ledger,
		receiptStore,
		processRegistry,
		processRuntime,
		cfg,
		spec,
	)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	result, err := codexRealSmokeRunLoopForTestV0(ctx, service, runRef, dispatchers)
	if err != nil {
		t.Fatalf("launch loop: %v\n%s", err, codexRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	if !codexDeliveryLoopContainsRefV0(result.Run.StartedAgents, spec.RequestID) {
		t.Fatalf("started_agents=%v", result.Run.StartedAgents)
	}
	defer codexRealSmokeStopProcessForTestV0(t, processRuntime, processRegistry, runRef, spec.RequestID)

	if err := codexRealSmokeWaitForAckV0(ctx, cfg.RuntimeWorkDir, spec); err != nil {
		t.Fatalf("ack real no validado: %v\n%s", err, codexRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	result, err = codexRealSmokeRunLoopForTestV0(ctx, service, runRef, dispatchers)
	if err != nil {
		t.Fatalf("delivery loop: %v\n%s", err, codexRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	if !codexDeliveryLoopContainsRefV0(result.Run.Deliveries, spec.AgentPacket.DeliveryRefs.AckRef) {
		t.Fatalf("deliveries=%v\n%s", result.Run.Deliveries, codexRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
}

func codexRealSmokeSpecForTestV0(agentRef string, taskRef string) orquestaruntime.ExternalAgentLaunchSpecV0 {
	spec := codexDeliveryLoopSpecForTestV0(agentRef, taskRef)
	spec.AgentPacket.Task.Title = "Actualizar README de agenda smoke"
	spec.AgentPacket.Task.Objective = "Actualiza README.md con una descripcion minima de una agenda con API REST y web."
	spec.AgentPacket.Task.RequiredTests = nil
	spec.AgentPacket.Task.DoneCriteria = []string{
		"README.md actualizado dentro del write-set.",
		"agent_ack.json escrito con status completed.",
	}
	spec.AgentPacket.Context.Entries[0].SourceRef = "source-ref-real-smoke-001"
	return spec
}

func codexRealSmokeServiceForTestV0(
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	receiptStore *InMemoryCodexReceiptDescriptorStoreV0,
	runRef string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) orquestacionnucleoapp.ServiceV0 {
	return orquestacionnucleoapp.ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: orquestacionnucleoapp.DeliveryCandidateProviderV0{
			Base: orquestacionnucleoapp.StaticCandidateProviderV0{
				Candidates: orquestacionnucleoapp.SchedulerCandidateSetV0{
					WorkCandidates: []orquestadirectorscheduler.SchedulableWorkCandidateV0{
						codexDeliveryLoopWorkCandidateV0(runRef, spec.RequestID, spec.AgentPacket.Task.TaskRef),
					},
				},
			},
			DeliverySource: CodexDeliveryObservationSourceV0{Store: receiptStore},
			RequestedBy:    "orquesta-codex-real-smoke",
		},
		OutboxLedger: ledger,
		MaxCommands:  8,
	}
}

func codexRealSmokeDispatchersForTestV0(
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	receiptStore *InMemoryCodexReceiptDescriptorStoreV0,
	processRegistry orquestacionnucleoapp.AgentProcessRegistryPortV0,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
	cfg codexRealSmokeConfigV0,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) []orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	profile := codexRealSmokeProfileV0(cfg)
	specResolver := CodexReceiptRecordingSpecResolverV0{
		Inner: staticExternalAgentSpecResolverV0{
			Spec:            spec,
			CommandResolver: orquestaruntimecodex.NewCodexExecResolverV0(profile),
		},
		Recorder: receiptStore,
		AckPathResolver: StaticCodexReceiptAckPathResolverV0{
			AckPath: filepath.Join(cfg.RuntimeWorkDir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		},
	}
	agentDispatcher := orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
			RunStore:  store,
			EventSink: sink,
			Launcher: orquestacionnucleoapp.ExternalProcessAgentLauncherV0{
				SpecResolver:    specResolver,
				Runtime:         processRuntime,
				ProcessStopper:  processRuntime,
				ProcessRegistry: processRegistry,
			},
			FailureStopper: orquestacionnucleoapp.ProcessAgentStopperV0{Registry: processRegistry, Runtime: processRuntime},
			OccurredAt:     "2026-05-09T19:01:00Z",
			RequestedBy:    "orquesta-codex-real-smoke",
			EvidenceRefs:   []string{"evidence-ref-real-smoke-agent-001"},
		},
		Acker: ledger,
	}
	return []orquestacionnucleoapp.OutboxDispatcherBindingV0{
		codexDeliveryCapacityDispatcherForTestV0(store, sink, ledger),
		agentDispatcher,
	}
}

func codexRealSmokeProfileV0(cfg codexRealSmokeConfigV0) orquestaruntimecodex.CodexConnectorProfileV0 {
	return orquestaruntimecodex.CodexConnectorProfileV0{
		SchemaVersion:  orquestaruntimecodex.CodexConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    cfg.CommandPath,
		ProjectWorkDir: cfg.ProjectWorkDir,
		RuntimeWorkDir: cfg.RuntimeWorkDir,
		CodeHomeDir:    cfg.CodeHomeDir,
		HomeDir:        cfg.HomeDir,
		PathEnv:        cfg.PathEnv,
		Model:          cfg.Model,
		Profile:        cfg.Profile,
		Sandbox:        cfg.Sandbox,
		ApprovalPolicy: cfg.ApprovalPolicy,
		ExtraArgs:      cfg.ExtraArgs,
		PromptHints: []string{
			"Smoke real opt-in: trabaja solo dentro del write-set del paquete.",
			"Escribe el ACK exactamente en el path indicado al terminar.",
		},
	}
}

func codexRealSmokeRunLoopForTestV0(
	ctx context.Context,
	service orquestacionnucleoapp.ServiceV0,
	runRef string,
	dispatchers []orquestacionnucleoapp.OutboxDispatcherBindingV0,
) (orquestacionnucleoapp.ProgressiveLoopResultV0, error) {
	return service.RunProgressiveLoopV0(ctx, orquestacionnucleoapp.ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-09T19:00:00Z",
		MaxBursts:            4,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 2,
		CorrelationID:        "corr-real-smoke-001",
		EvidenceRefs:         []string{"evidence-ref-real-smoke-loop-001"},
		Dispatchers:          dispatchers,
	})
}

func codexRealSmokeWaitForAckV0(
	ctx context.Context,
	runtimeDir string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) error {
	return codexRealSmokeWaitForAckPathV0(
		ctx,
		filepath.Join(runtimeDir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		spec,
	)
}

func codexRealSmokeWaitForAckPathV0(
	ctx context.Context,
	ackPath string,
	spec orquestaruntime.ExternalAgentLaunchSpecV0,
) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		_, issues := orquestaruntimecodex.ReadCodexDeliveryObservationFileV0(ackPath, spec)
		if len(issues) == 0 {
			return nil
		}
		if !codexReceiptIssueMeansAckNotReadyV0(issues[0]) {
			return fmt.Errorf("%s:%s:%s", issues[0].Code, issues[0].Field, strings.Join(issues[0].Evidence, ","))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func codexRealSmokeStopProcessForTestV0(
	t *testing.T,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
	registry orquestacionnucleoapp.AgentProcessRegistryPortV0,
	runRef string,
	agentRef string,
) {
	t.Helper()
	record, err := registry.ResolveAgentProcessV0(context.Background(), runRef, agentRef)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = processRuntime.StopV0(ctx, record.ProcessRef)
}

func codexRealSmokeDiagnosticsV0(runtimeDir string) string {
	parts := []string{}
	for _, name := range []string{
		orquestaruntimecodex.CodexAgentAckFileNameV0,
		orquestaruntimecodex.CodexLastMessageFileNameV0,
		orquestaruntimecodex.CodexStdoutFileNameV0,
		orquestaruntimecodex.CodexStderrFileNameV0,
	} {
		data, err := os.ReadFile(filepath.Join(runtimeDir, name))
		if err != nil {
			continue
		}
		parts = append(parts, name+": "+codexRealSmokeTruncateV0(string(data)))
	}
	return strings.Join(parts, "\n")
}

func codexRealSmokeTruncateV0(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 1600 {
		return value
	}
	return value[:1600] + "\n[truncated]"
}
