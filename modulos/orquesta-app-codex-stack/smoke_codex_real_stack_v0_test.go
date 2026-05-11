package orquestaappcodexstack

import (
	"net/url"
	"testing"
	"time"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaweb "orquesta/modulos/orquesta-web"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type codexStackRealSmokeStoresV0 struct {
	RunStore        *orquestacionnucleoapp.InMemoryRunStoreV0
	EventSink       *orquestacionnucleoapp.InMemoryEventSinkV0
	OutboxLedger    *orquestacionnucleoapp.InMemoryOutboxLedgerV0
	TaskStore       *orquestacionnucleoapp.InMemoryWorkflowTaskStoreV0
	AppChangeStore  *orquestaappchange.InMemoryAppChangeStoreV0
	ReceiptStore    *orquestaruntimecodexdelivery.InMemoryCodexReceiptDescriptorStoreV0
	ProgressState   *orquestaruntimecodexdelivery.InMemoryCodexProgressStateStoreV0
	ProcessRegistry *orquestacionnucleoapp.InMemoryAgentProcessRegistryV0
	RunMemory       *orquestarunmemory.RunMemoryStoreV0
}

func codexStackRealSmokeBuildStackV0(
	t *testing.T,
	cfg codexStackRealSmokeConfigV0,
	stores codexStackRealSmokeStoresV0,
	processRuntime *orquestaruntime.ProcessRuntimeConnectorV0,
) StackV0 {
	t.Helper()
	stack, err := BuildStackV0(ConfigV0{
		Enabled: true,
		Timeout: cfg.Timeout,
		DirectorLimits: orquestaweb.WebArrancarDirectorAppLimitsV0{
			MaxBursts:            16,
			MaxStepsPerBurst:     12,
			MaxDispatchesPerWait: 8,
			MaxCommands:          20,
			MaxOutboxPerCycle:    8,
			MaxExternalWaits:     codexStackRealSmokeMaxExternalWaitsV0(cfg.Timeout, 2*time.Second),
		},
		Stores: StoresV0{
			RunStore:        stores.RunStore,
			EventSink:       stores.EventSink,
			OutboxLedger:    stores.OutboxLedger,
			TaskStore:       stores.TaskStore,
			AppChangeStore:  stores.AppChangeStore,
			ReceiptStore:    stores.ReceiptStore,
			ProgressState:   stores.ProgressState,
			ProcessRegistry: stores.ProcessRegistry,
			RunControl:      stores.RunMemory,
			RunQueue:        stores.RunMemory,
		},
		Codex: CodexRuntimeConfigV0{
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
				"Smoke real de Orquesta desde /nueva-app: entrega documentos breves y accionables.",
				"Prioriza terminar con ACK valido antes que ampliar alcance.",
			},
			Runtime:        processRuntime,
			ProcessStopper: processRuntime,
			SnapshotSource: processRuntime,
			MaxBatchReady:  cfg.MaxBatchReady,
			MaxConcurrency: cfg.MaxConcurrency,
			WaitInterval:   2 * time.Second,
			ProgressPolicy: orquestaruntime.AgentProgressHeartbeatPolicyV0{
				StalledAfterNoProgressTicks: 45,
				LoopAfterRepeatedActions:    180,
			},
		},
		Capacity: CapacityConfigV0{
			Tier:            orquestacoreworkflow.OrchestrationCapacityXHighV0,
			ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityXHighV0,
			OccurredAt:      "2026-05-10T12:00:00Z",
			RequestedBy:     "orquesta-app-stack-smoke",
			Summary:         "Capacidad real opt-in para director de app.",
			EvidenceRefs:    []string{"evidence-ref-app-stack-real-smoke"},
		},
		ReviewGate: ReviewGateConfigV0{
			FileEvidence: orquestaruntimecodexdelivery.CodexReviewGateProjectFileEvidenceV0{},
		},
	})
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	return stack
}

func newCodexStackRealSmokeStoresV0() codexStackRealSmokeStoresV0 {
	return codexStackRealSmokeStoresV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(),
		EventSink:       orquestacionnucleoapp.NewInMemoryEventSinkV0(),
		OutboxLedger:    orquestacionnucleoapp.NewInMemoryOutboxLedgerV0(),
		TaskStore:       orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(),
		AppChangeStore:  orquestaappchange.NewInMemoryAppChangeStoreV0(),
		ReceiptStore:    orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(),
		ProgressState:   orquestaruntimecodexdelivery.NewInMemoryCodexProgressStateStoreV0(),
		ProcessRegistry: orquestacionnucleoapp.NewInMemoryAgentProcessRegistryV0(),
		RunMemory:       orquestarunmemory.NewRunMemoryStoreV0(),
	}
}

func codexStackRealSmokeFormValuesV0() url.Values {
	values := url.Values{}
	values.Set("request_id", "request-ref-app-stack-real-smoke-001")
	values.Set("locale", "es-ES")
	values.Set("request_kind", "crear_app_completa")
	values.Set("execution_mode", "normal")
	values.Set("nombre", "Agenda API Web")
	values.Set("objetivo", "Gestionar contactos y citas con API REST en Go y una web de administracion.")
	values.Set("descripcion", "Smoke real desde /nueva-app para validar director, ACK y documentacion.")
	values.Set("tipo_app", "mixed")
	values.Set("plataformas", "server,web")
	values.Set("preferencias_tecnicas.lenguaje", "go")
	values.Set("preferencias_tecnicas.arquitectura", "hexagonal")
	values.Set("preferencias_tecnicas.restricciones", "funciones pequenas,persistencia por puerto")
	values.Set("datos.db_required", "true")
	values.Set("datos.necesidad_funcional", "Persistir contactos y citas por puerto.")
	values.Set("calidad.pruebas", "alta")
	values.Set("calidad.accesibilidad", "basica")
	values.Set("calidad.observabilidad", "true")
	values.Set("documentacion.desarrollo", "true")
	values.Set("documentacion.sistemas", "true")
	values.Set("i18n.enabled", "true")
	values.Set("i18n.default_locale", "es-ES")
	values.Set("i18n.locales", "es-ES,en-US")
	values.Set("agentes.autonomia", "media")
	return values
}

func codexStackRealSmokeMultiagentFormValuesV0() url.Values {
	values := codexStackRealSmokeFormValuesV0()
	values.Set("request_id", "request-ref-app-stack-real-multi-001")
	values.Set("descripcion", "Smoke real multiagente desde /nueva-app con write-sets separados.")
	values.Set("agentes.autonomia", "alta")
	return values
}
