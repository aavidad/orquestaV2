package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaagentprocessregistrymemory "orquesta/modulos/orquesta-agent-process-registry-memory"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaweb "orquesta/modulos/orquesta-web"
)

func TestMCPFormularioDirectorCodexRealOptInV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_CODEX_MCP_FORM_SMOKE")) != "1" {
		t.Skip("set ORQUESTA_CODEX_MCP_FORM_SMOKE=1 para lanzar Codex real desde formulario MCP")
	}
	cfg := codexRealSmokeConfigForTestV0(t)
	if cfg.Timeout < 180*time.Second {
		cfg.Timeout = 180 * time.Second
	}
	form := mcpFormDirectorSmokeFormV0()
	receiptStore := NewInMemoryCodexReceiptDescriptorStoreV0()
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	processRegistry := orquestaagentprocessregistrymemory.NewInMemoryAgentProcessRegistryV0()
	processRuntime := orquestaruntime.NewProcessRuntimeConnectorV0()
	executor := orquestamcp.NewMCPArrancarDirectorAppToolExecutorV0(
		orquestaappdirectorservice.StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			Dispatchers: mcpFormDirectorSmokeDispatchersV0(
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
		RequestID:             form.RequestID,
		CorrelationID:         "corr-form-director-001",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		AppSpecRequest:        form.ToAppSpecRequestV0(),
		MaxBursts:             4,
		MaxStepsPerBurst:      4,
		MaxDispatchesPerWait:  2,
	})
	if err != nil {
		t.Fatalf("mcp director launch: %#v\n%s", err, codexRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	if result.Estado != orquestamcp.MCPArrancarDirectorAppEstadoOKV0 || len(result.StartedAgents) != 1 {
		t.Fatalf("mcp result=%+v", result)
	}
	defer codexRealSmokeStopProcessForTestV0(t, processRuntime, processRegistry, result.RunRef, result.DirectorTask.AgentRequestID)

	descriptor := mcpFormDirectorSmokeDescriptorV0(t, receiptStore, result.RunRef, result.StartedAgents)
	if err := codexRealSmokeWaitForAckV0(ctx, cfg.RuntimeWorkDir, descriptor.Spec); err != nil {
		t.Fatalf("ack director no validado: %v\n%s", err, codexRealSmokeDiagnosticsV0(cfg.RuntimeWorkDir))
	}
	observation, issues := orquestaruntimecodex.ReadCodexDeliveryObservationFileV0(descriptor.AckPath, descriptor.Spec)
	if len(issues) > 0 {
		t.Fatalf("observacion director: %+v", issues)
	}
	if observation.PhaseID != string(orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0) ||
		observation.AgentRef != result.DirectorTask.AgentRequestID {
		t.Fatalf("observacion=%+v result=%+v", observation, result)
	}
	artifactResult := mcpFormDirectorSmokeRegisterArtifactV0(t, ctx, store, sink, ledger, receiptStore, result.RunRef)
	if !mcpFormDirectorSmokeContainsProjectionPartV0(artifactResult.Run.PhaseArtifacts, observation.DeliveryRef) {
		t.Fatalf("phase_artifacts=%v observation=%+v", artifactResult.Run.PhaseArtifacts, observation)
	}
	if !codexDeliveryLoopHasEventV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventPhaseArtifactRegisteredV0) {
		t.Fatalf("sink no contiene PhaseArtifactRegistered: %+v", sink.EventsV0())
	}
	if codexDeliveryLoopHasEventV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventDeliveryRegisteredV0) {
		t.Fatalf("sink no debe contener DeliveryRegistered para director: %+v", sink.EventsV0())
	}
	mcpFormDirectorSmokeVerifyDocsV0(t, cfg.ProjectWorkDir)
}

func mcpFormDirectorSmokeRegisterArtifactV0(
	t *testing.T,
	ctx context.Context,
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	receiptStore *InMemoryCodexReceiptDescriptorStoreV0,
	runRef string,
) orquestacionnucleoapp.ProgressiveLoopResultV0 {
	t.Helper()
	service := orquestacionnucleoapp.ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: orquestacionnucleoapp.DeliveryCandidateProviderV0{
			DeliverySource: CodexDeliveryObservationSourceV0{Store: receiptStore},
			RequestedBy:    "orquesta-form-director",
		},
		OutboxLedger:      ledger,
		MaxCommands:       4,
		MaxOutboxPerCycle: 2,
	}
	result, err := service.RunProgressiveLoopV0(ctx, orquestacionnucleoapp.ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-09T23:31:00Z",
		MaxBursts:            3,
		MaxStepsPerBurst:     3,
		MaxDispatchesPerWait: 1,
		CorrelationID:        "corr-form-director-artifact",
		EvidenceRefs:         []string{"evidence-ref-form-director-artifact"},
	})
	if err != nil {
		t.Fatalf("register director artifact: %v", err)
	}
	if result.Status != orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 {
		t.Fatalf("artifact loop status=%s result=%+v", result.Status, result)
	}
	if result.PendingOutboxCount != 0 {
		t.Fatalf("artifact loop pending=%d refs=%v", result.PendingOutboxCount, result.PendingOutboxRefs)
	}
	return result
}

func mcpFormDirectorSmokeContainsProjectionPartV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

type mcpFormDirectorCodexSpecResolverV0 struct {
	CommandResolver orquestaruntime.ExternalAgentProcessCommandResolverV0
	Form            orquestaweb.WebNuevaAppFormV0
}

func (resolver mcpFormDirectorCodexSpecResolverV0) ResolveExternalAgentLaunchSpecV0(
	_ context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0, error) {
	agentRef := strings.TrimSpace(inbound.Payload.AgentRequestID)
	taskRef := strings.TrimSpace(inbound.Payload.TaskRef)
	if taskRef == "" {
		taskRef = "task-agenda-director"
	}
	spec := codexDeliveryLoopSpecForTestV0(agentRef, taskRef)
	spec.CorrelationID = strings.TrimSpace(inbound.CorrelationID)
	spec.AgentPacket.CorrelationID = spec.CorrelationID
	spec.AgentPacket.WorkOrderRef = taskRef
	spec.AgentPacket.TargetModule = "agenda-director"
	spec.AgentPacket.Phase = strings.TrimSpace(inbound.Payload.PhaseID)
	spec.AgentPacket.CapacityLevel = "high"
	spec.AgentPacket.Task = orquestaruntime.AgentStartTaskV0{
		TaskRef:      taskRef,
		Priority:     "alta",
		Title:        "Dirigir arquitectura inicial de Agenda",
		Objective:    mcpFormDirectorSmokeObjectiveV0(resolver.Form),
		TargetSymbol: "AgendaDirector",
		WriteSet: []string{
			"docs/arquitectura.md",
			"docs/plan_microtareas.md",
		},
		DoneCriteria: []string{
			"docs/arquitectura.md contiene decisiones de arquitectura, hexagonal e i18n.",
			"docs/plan_microtareas.md contiene cortes pequenos para agentes posteriores.",
			"agent_ack.json escrito con status completed.",
		},
	}
	spec.AgentPacket.Context = orquestacontext.ContextMaterializedBundleV0{
		SchemaVersion: orquestacontext.ContextMaterializedBundleSchemaVersionV0,
		BundleRef:     "bundle-ref-agenda-director",
		WorkOrderRef:  taskRef,
		TargetModule:  "agenda-director",
		Entries: []orquestacontext.ContextMaterializedEntryV0{{
			EntryRef:  "entry-ref-agenda-form",
			Layer:     orquestacontext.ContextLayerTaskContextV0,
			Kind:      orquestacontext.ContextEntryDocRefV0,
			SourceRef: "source-ref-agenda-form",
			Mode:      orquestacontext.ContextMaterializationModeRefOnlyV0,
			Required:  true,
		}},
	}
	spec.AgentPacket.DeliveryRefs = orquestaruntime.AgentStartDeliveryRefsV0{
		MailboxRef:   "mailbox-ref-agenda-director",
		AckRef:       "ack-ref-agenda-director",
		ReadinessRef: "readiness-ref-agenda-director",
	}
	return orquestacionnucleoapp.ExternalAgentLaunchSpecResolutionV0{
		Spec:            spec,
		CommandResolver: resolver.CommandResolver,
	}, nil
}

func mcpFormDirectorSmokeDispatchersV0(
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
	receiptStore *InMemoryCodexReceiptDescriptorStoreV0,
	processRegistry orquestacionnucleoapp.AgentProcessRegistryPortV0,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
	cfg codexRealSmokeConfigV0,
	form orquestaweb.WebNuevaAppFormV0,
) []orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	profile := codexRealSmokeProfileV0(cfg)
	specResolver := CodexReceiptRecordingSpecResolverV0{
		Inner: mcpFormDirectorCodexSpecResolverV0{
			CommandResolver: orquestaruntimecodex.NewCodexExecResolverV0(profile),
			Form:            form,
		},
		Recorder: receiptStore,
		AckPathResolver: StaticCodexReceiptAckPathResolverV0{
			AckPath: filepath.Join(cfg.RuntimeWorkDir, orquestaruntimecodex.CodexAgentAckFileNameV0),
		},
	}
	return []orquestacionnucleoapp.OutboxDispatcherBindingV0{
		mcpFormDirectorCapacityDispatcherV0(store, sink, ledger),
		{
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
				OccurredAt:     "2026-05-09T23:30:00Z",
				RequestedBy:    "orquesta-form-director",
				EvidenceRefs:   []string{"evidence-ref-form-director-agent"},
			},
			Acker: ledger,
		},
	}
}

func mcpFormDirectorCapacityDispatcherV0(
	store orquestacionnucleoapp.RunStorePortV0,
	sink orquestacionnucleoapp.EventSinkPortV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.CapacityDecisionExecutorV0{
			RunStore:        store,
			EventSink:       sink,
			Tier:            orquestacoreworkflow.OrchestrationCapacityHighV0,
			ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityHighV0,
			OccurredAt:      "2026-05-09T23:29:00Z",
			CorrelationID:   "corr-form-director-capacity",
			RequestedBy:     "orquesta-form-director",
		},
		Acker: ledger,
	}
}
