package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaweb "orquesta/modulos/orquesta-web"
	orquestacionnucleoapp "orquesta/orquestacionnucleoapp"
)

func TestMCPFormularioDirectorTeamCodexRealOptInV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_MCP_FORM_TEAM_SMOKE")) != "1" {
		t.Skip("set ORQUESTA_CODEX_MCP_FORM_TEAM_SMOKE=1 para lanzar equipo Codex real")
	}
	cfg := codexRealSmokeConfigForTestV0(t)
	if cfg.Timeout < 360*time.Second {
		cfg.Timeout = 360 * time.Second
	}
	form := mcpFormDirectorTeamSmokeFormV0()
	receiptStore := NewInMemoryCodexReceiptDescriptorStoreV0()
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	processRegistry := orquestacionnucleoapp.NewInMemoryAgentProcessRegistryV0()
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	executor := orquestamcp.NewMCPArrancarDirectorAppToolExecutorV0(
		orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				mcpFormDirectorCapacityDispatcherV0(store, sink, ledger),
			},
			BatchDispatchers: mcpFormDirectorTeamSmokeBatchDispatchersV0(
				store,
				sink,
				ledger,
				receiptStore,
				processRegistry,
				processRuntime,
				cfg,
				form,
			),
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	result, err := executor.Execute(ctx, orquestamcp.MCPArrancarDirectorAppToolInputV0{
		RequestID:            form.RequestID,
		CorrelationID:        "corr-form-director-team-001",
		AppSpecRequest:       form.ToAppSpecRequestV0(),
		MaxBursts:            8,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 8,
	})
	if err != nil {
		t.Fatalf("mcp director team launch: %#v\n%s", err, codexRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	if result.Estado != orquestamcp.MCPArrancarDirectorAppEstadoOKV0 ||
		len(result.DirectorTasks) != 4 ||
		len(result.StartedAgents) != 4 {
		t.Fatalf("mcp result=%+v", result)
	}
	defer mcpFormDirectorTeamSmokeStopProcessesV0(t, processRuntime, processRegistry, result.RunRef, result.StartedAgents)

	descriptors := mcpFormDirectorTeamSmokeDescriptorsV0(t, receiptStore, result.RunRef, result.StartedAgents)
	for _, descriptor := range descriptors {
		if err := codexRealSmokeWaitForAckPathV0(ctx, descriptor.AckPath, descriptor.Spec); err != nil {
			t.Fatalf("ack team no validado %s: %v\n%s", descriptor.AgentRef, err, codexRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
		}
	}
	artifactResult := mcpFormDirectorTeamSmokeRegisterArtifactsV0(t, ctx, store, sink, ledger, receiptStore, result.RunRef)
	for _, descriptor := range descriptors {
		ackRef := descriptor.Spec.AgentPacket.DeliveryRefs.AckRef
		if !mcpFormDirectorSmokeContainsProjectionPartV0(artifactResult.Run.PhaseArtifacts, ackRef) {
			t.Fatalf("phase_artifacts=%v missing=%s", artifactResult.Run.PhaseArtifacts, ackRef)
		}
	}
	mcpFormDirectorTeamSmokeVerifyDocsV0(t, cfg.ProjectWorkDir)
}

func mcpFormDirectorTeamSmokeFormV0() orquestaweb.WebNuevaAppFormV0 {
	form := mcpFormDirectorSmokeFormV0()
	form.RequestID = "req-form-director-team-real-001"
	form.Agentes.Autonomia = "alta"
	form.Descripcion = "Prueba real multiagente: Orquesta debe arrancar un equipo de directores Codex."
	return form
}

func mcpFormDirectorTeamSmokeBatchDispatchersV0(
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	receiptStore *InMemoryCodexReceiptDescriptorStoreV0,
	processRegistry orquestacionnucleoapp.AgentProcessRegistryPortV0,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
	cfg codexRealSmokeConfigV0,
	form orquestaweb.WebNuevaAppFormV0,
) []orquestacionnucleoapp.OutboxBatchDispatcherBindingV0 {
	specResolver := CodexReceiptRecordingSpecResolverV0{
		Inner: mcpFormDirectorTeamCodexSpecResolverV0{
			Config: cfg,
			Form:   form,
		},
		Recorder: receiptStore,
		AckPathResolver: AgentScopedCodexReceiptAckPathResolverV0{
			BaseDir: cfg.RuntimeWorkDir,
		},
	}
	return []orquestacionnucleoapp.OutboxBatchDispatcherBindingV0{{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		MaxReady:   2,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.ExternalProcessAgentBatchExecutorV0{
			RunStore:        store,
			EventSink:       sink,
			SpecResolver:    specResolver,
			Runtime:         processRuntime,
			ProcessStopper:  processRuntime,
			ProcessRegistry: processRegistry,
			MaxConcurrency:  2,
			OccurredAt:      "2026-05-09T23:45:00Z",
			RequestedBy:     "orquesta-form-director-team",
			EvidenceRefs:    []string{"evidence-ref-form-director-team-agent"},
		},
		Acker: ledger,
	}}
}
