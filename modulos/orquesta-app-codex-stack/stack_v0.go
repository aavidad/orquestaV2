package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaappgateway "orquesta/modulos/orquesta-app-gateway"
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	channel "orquesta/modulos/orquesta-operator-director-channel"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
	orquestaweb "orquesta/modulos/orquesta-web"
)

type StackV0 struct {
	Handler                               http.Handler
	MCPTransportBindings                  orquestamcp.MCPTransportBindingsV0
	Ports                                 orquestaappdirectorservice.StartAppDirectorPortsV0
	Stores                                StoresV0
	RunQueue                              RunQueueConfigV0
	RunSupervisor                         RunSupervisorConfigV0
	DirectorLimits                        orquestaweb.WebArrancarDirectorAppLimitsV0
	Clock                                 orquestafactoryhttp.AppSpecHTTPClockV0
	AutoprogrammingPromotion              AutoprogrammingPromotionConfigV0
	DomainWork                            orquestamcp.MCPDomainWorkExecutorPortV0
	ConfigProjectionSettings              []orquestamcp.MCPConfigProjectionSettingV0
	CodeContext                           orquestacontext.CodeContextQueryPortV0
	CodeContextToolLeases                 orquestacontext.CodeContextToolLeaseListPortV0
	RuntimeModels                         orquestaruntime.RuntimeModelManagerPortV0
	DecisionCouncil                       DecisionCouncilConfigV0
	DomainDelivery                        DomainWorkDeliveryBridgeConfigV0
	ProviderRuntimes                      []RuntimeProviderConfigV0
	EgressSanitizer                       EgressSanitizerConfigV0
	Codex                                 CodexRuntimeConfigV0
	CodexRuntimeWorkDir                   string
	CodexSnapshotSource                   orquestaruntimecodexdelivery.CodexProcessSnapshotSourcePortV0
	PromoteMaterializedArtifactWithoutAck bool
	AllowLegacyAutoprogrammingRun         bool
	AllowLegacyExternalWorkRun            bool
	GoalMaterializedResultWatcher         *GoalMaterializedResultWatcherV0
}

func BuildStackV0(config ConfigV0) (StackV0, error) {
	config = configWithCanonicalEgressSanitizerV0(config)
	if err := validateConfigV0(config); err != nil {
		return StackV0{}, err
	}
	if config.DomainDelivery.JobRecords == nil {
		config.DomainDelivery.JobRecords = domainWorkJobRecordSourceFromExecutorV0(config.DomainWork)
	}
	config.DomainDelivery = normalizeDomainWorkDeliveryBridgeConfigV0(config.DomainDelivery)
	decisionCouncil := codexStackDecisionCouncilConfigWithDefaultsV0(config)
	ports := buildDirectorPortsV0(config)
	queueConfig := normalizeRunQueueConfigV0(config.RunQueue)
	supervisorConfig := normalizeRunSupervisorConfigV0(config.RunSupervisor)
	stack := StackV0{
		Ports:                                 ports,
		Stores:                                config.Stores,
		RunQueue:                              queueConfig,
		RunSupervisor:                         supervisorConfig,
		DirectorLimits:                        config.DirectorLimits,
		Clock:                                 config.Clock,
		AutoprogrammingPromotion:              config.AutoprogrammingPromotion,
		DomainWork:                            config.DomainWork,
		ConfigProjectionSettings:              configProjectionSettingsV0(config.ConfigProjectionSettings),
		CodeContext:                           config.CodeContext,
		CodeContextToolLeases:                 config.CodeContextToolLeases,
		RuntimeModels:                         config.RuntimeModels,
		DecisionCouncil:                       decisionCouncil,
		DomainDelivery:                        config.DomainDelivery,
		ProviderRuntimes:                      CanonicalRuntimeProviderConfigsV0(config),
		EgressSanitizer:                       NormalizeEgressSanitizerConfigV0(config.EgressSanitizer),
		Codex:                                 config.Codex,
		CodexRuntimeWorkDir:                   config.Codex.RuntimeWorkDir,
		CodexSnapshotSource:                   config.Codex.SnapshotSource,
		PromoteMaterializedArtifactWithoutAck: config.PromoteMaterializedArtifactWithoutAck,
		AllowLegacyAutoprogrammingRun:         config.AllowLegacyAutoprogrammingRun,
		AllowLegacyExternalWorkRun:            config.AllowLegacyExternalWorkRun,
	}
	stack.MCPTransportBindings = buildStackMCPTransportBindingsV0(config, ports, queueConfig, &stack)
	stack.Handler = buildStackHTTPHandlerV0(config, stack.MCPTransportBindings)
	return stack, nil
}

func domainWorkJobRecordSourceFromExecutorV0(
	executor orquestamcp.MCPDomainWorkExecutorPortV0,
) orquestadomainwork.DomainWorkJobRecordSourcePortV0 {
	source, _ := executor.(orquestadomainwork.DomainWorkJobRecordSourcePortV0)
	return source
}

func buildStackMCPTransportBindingsV0(
	config ConfigV0,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
	queueConfig RunQueueConfigV0,
	stack *StackV0,
) orquestamcp.MCPTransportBindingsV0 {
	arrancar := NewQueuedArrancarDirectorExecutorV0(QueuedArrancarDirectorConfigV0{
		Inner:                    orquestamcp.NewMCPArrancarDirectorAppToolExecutorV0(startOnlyPortsV0(ports)),
		Writer:                   config.Stores.RunQueue,
		Queue:                    queueConfig,
		Clock:                    config.Clock,
		Source:                   "orquesta-app-codex-stack",
		Reason:                   "run creado desde director app",
		ConfigProjectionSettings: config.ConfigProjectionSettings,
	})
	estadoVivoSource := estadoVivoSourceV0(config)
	runControlPort := goalFirstRunControlPortFromConfigV0(config)
	directorStats := orquestamcp.MCPDirectorStatsToolExecutorV0{
		RunStore:          config.Stores.RunStore,
		RunControl:        config.Stores.RunControl,
		ProcessRegistry:   config.Stores.ProcessRegistry,
		ProcessSnapshot:   config.Codex.SnapshotSource,
		ProgressSource:    statsProgressSourceV0(config),
		AgentUsageSource:  agentUsageSourceV0(config),
		EstadoVivoSource:  estadoVivoSource,
		ExternalJobSource: externalJobStatsSourceV0(config),
		GoalStateSource:   config.Stores.AppGoalStateStore,
		GoalMarkerSource:  appGoalFirstRunMarkerStoreV0(config),
		GoalMaterializedRefsSource: stackGoalMaterializedRefsSourceV0{
			Config:                       config,
			GoalStateStore:               config.Stores.AppGoalStateStore,
			GoalClosureValidator:         ports.GoalClosureValidator,
			RepairMissingTerminalReceipt: true,
		},
	}
	bindings := orquestamcp.MCPTransportBindingsV0{
		NuevaAppWizard:   NewCodexStackNuevaAppWizardExecutorV0(config.AppIntakeAssistant),
		ArrancarDirector: arrancar,
		PreviewDirector:  orquestamcp.NewMCPPreviewDirectorAppToolExecutorV0(),
		ObserveDirectorGoal: NewCodexStackObserveAppDirectorGoalExecutorV0(
			stack,
		),
		RequestAppChange: orquestamcp.NewMCPRequestAppChangeToolExecutorV0(appChangePortsV0(config)),
		DirectorStats:    directorStats,
		RunControl: orquestamcp.MCPRunControlToolExecutorV0{
			Port:               runControlPort,
			ExternalJobSource:  externalJobStatsSourceV0(config),
			GoalBackendState:   directorStats,
			GoalStateStore:     config.Stores.AppGoalStateStore,
			GoalProgressPolicy: config.AutoprogrammingGoalProgressPolicy,
		},
		RuntimeModels: config.RuntimeModels,
		RunQueuePriority: orquestamcp.MCPRunQueuePriorityToolExecutorV0{
			Reader: config.Stores.RunQueue,
			Writer: config.Stores.RunQueue,
		},
		RunSupervisor: NewCodexStackRunSupervisorExecutorV0(stack),
		AppVCS:        NewCodexStackAppVCSExecutorV0(config.Codex.ProjectWorkDir),
		AutoprogrammingPrepareRun: NewCodexStackAutoprogrammingPrepareRunExecutorV0(
			stack,
			config.Capacity.OccurredAt,
			config.Capacity.RequestedBy,
			config.Stores.RunQueue,
			queueConfig,
			config.Clock,
			config.Codex.RuntimeWorkDir,
		),
		AutoprogrammingObserveGoal: NewCodexStackAutoprogrammingObserveGoalExecutorV0(
			stack,
		),
		AutoprogrammingObserveActiveGoals: orquestamcp.NewMCPAutoprogrammingObserveActiveGoalsToolExecutorV0(
			config.Stores.AppGoalStateStore,
			NewCodexStackAutoprogrammingObserveGoalExecutorV0(stack),
		),
		AutoprogrammingGoalStates:                      config.Stores.AppGoalStateStore,
		AutoprogrammingEstadoVivoSource:                estadoVivoSource,
		AutoprogrammingIdleSelfImprovementBudgetSource: config.AutoprogrammingIdleSelfImprovementBudgetSource,
		AutoprogrammingStatusDiagnostics:               config.AutoprogrammingStatusDiagnostics,
		AutoprogrammingGoalProgressPolicy:              config.AutoprogrammingGoalProgressPolicy,
		AllowLegacyAutoprogrammingSupervisorActions:    config.AllowLegacyAutoprogrammingRun,
		ServerShutdown:                                 serverShutdownExecutorV0(config, stack),
		DomainWork:                                     config.DomainWork,
		CodebaseQuery:                                  orquestamcp.MCPCodebaseQueryToolExecutorV0{Broker: config.CodeContext},
		CodebaseStatus: orquestamcp.MCPCodebaseStatusToolExecutorV0{
			Leases: config.CodeContextToolLeases,
			Clock: func() time.Time {
				return stackNowV0(config.Clock)
			},
		},
		ExternalWorkDryRun: externalWorkDryRunExecutorV0(config, queueConfig),
		ExternalWorkRun:    externalWorkRunGuardedExecutorV0(config, queueConfig),
	}
	bindings.OperatorDirectorMessage = channel.OperatorDirectorChannelServiceV0{
		Dispatcher: stackOperatorDirectorDispatcherV0{Bindings: bindings, Queue: queueConfig},
		Store:      newStackOperatorDirectorMemoryStoreV0(),
	}
	return bindings
}

type stackOperatorDirectorDispatcherV0 struct {
	Bindings orquestamcp.MCPTransportBindingsV0
	Queue    RunQueueConfigV0
}

func (dispatcher stackOperatorDirectorDispatcherV0) DispatchOperatorMessageV0(
	ctx context.Context,
	message channel.OperatorMessageV0,
) (channel.OperatorDirectorResponseV0, error) {
	operation := stackOperatorDirectorOperationV0(message)
	switch operation {
	case "status":
		return dispatcher.dispatchOperatorDirectorStatusV0(ctx, message), nil
	case "queue":
		return dispatcher.dispatchOperatorDirectorQueueV0(ctx, message), nil
	case "observe":
		return dispatcher.dispatchOperatorDirectorObserveV0(ctx, message), nil
	case "launch", "handoff":
		return dispatcher.dispatchOperatorDirectorLaunchHandoffV0(message, operation), nil
	case "stop", "control":
		return dispatcher.dispatchOperatorDirectorControlV0(ctx, message, operation), nil
	default:
		return stackOperatorDirectorResponseV0(
			message,
			"blocked",
			"operator_director_operation_not_supported",
			"use status, queue, observe, launch, handoff, stop or control",
			[]string{"evidence-ref-operator-director-operation-unsupported"},
		), nil
	}
}

func (dispatcher stackOperatorDirectorDispatcherV0) dispatchOperatorDirectorStatusV0(
	ctx context.Context,
	message channel.OperatorMessageV0,
) channel.OperatorDirectorResponseV0 {
	if stackOperatorDirectorRunRefV0(message) == "" {
		if dispatcher.Bindings.RunQueuePriority == nil {
			return stackOperatorDirectorPortUnavailableResponseV0(message, "status")
		}
		result, err := dispatcher.Bindings.RunQueuePriority.Execute(ctx, orquestamcp.MCPRunQueuePriorityToolInputV0{
			RequestID:     message.RequestRef,
			CorrelationID: message.ConversationRef,
			Action:        orquestamcp.MCPRunQueuePriorityActionRankV0,
			QueueRef:      normalizeRunQueueConfigV0(dispatcher.Queue).QueueRef,
			Limit:         10,
		})
		if err != nil || result.Estado != orquestamcp.MCPRunQueuePriorityEstadoOKV0 {
			return stackOperatorDirectorBlockedResponseV0(message, "status", "operator_director_status_unavailable")
		}
		return stackOperatorDirectorResponseV0(
			message,
			"accepted",
			"operator_director_status_ready",
			stackOperatorDirectorJSONSummaryV0(result),
			[]string{"evidence-ref-operator-director-status"},
		)
	}
	if dispatcher.Bindings.DirectorStats == nil {
		return stackOperatorDirectorPortUnavailableResponseV0(message, "status")
	}
	result, err := dispatcher.Bindings.DirectorStats.Execute(ctx, orquestamcp.MCPDirectorStatsToolInputV0{
		RequestID:     message.RequestRef,
		CorrelationID: message.ConversationRef,
		RunRef:        stackOperatorDirectorRunRefV0(message),
	})
	if err != nil || result.Estado != orquestamcp.MCPDirectorStatsEstadoOKV0 {
		return stackOperatorDirectorBlockedResponseV0(message, "status", "operator_director_status_unavailable")
	}
	return stackOperatorDirectorResponseV0(
		message,
		"accepted",
		"operator_director_status_ready",
		stackOperatorDirectorJSONSummaryV0(result),
		[]string{"evidence-ref-operator-director-status"},
	)
}

func (dispatcher stackOperatorDirectorDispatcherV0) dispatchOperatorDirectorQueueV0(
	ctx context.Context,
	message channel.OperatorMessageV0,
) channel.OperatorDirectorResponseV0 {
	if dispatcher.Bindings.RunQueuePriority == nil {
		return stackOperatorDirectorPortUnavailableResponseV0(message, "queue")
	}
	result, err := dispatcher.Bindings.RunQueuePriority.Execute(ctx, orquestamcp.MCPRunQueuePriorityToolInputV0{
		RequestID:     message.RequestRef,
		CorrelationID: message.ConversationRef,
		Action:        orquestamcp.MCPRunQueuePriorityActionRankV0,
		QueueRef:      normalizeRunQueueConfigV0(dispatcher.Queue).QueueRef,
		Limit:         10,
	})
	if err != nil || result.Estado != orquestamcp.MCPRunQueuePriorityEstadoOKV0 {
		return stackOperatorDirectorBlockedResponseV0(message, "queue", "operator_director_queue_unavailable")
	}
	return stackOperatorDirectorResponseV0(
		message,
		"accepted",
		"operator_director_queue_ready",
		stackOperatorDirectorJSONSummaryV0(result),
		[]string{"evidence-ref-operator-director-queue"},
	)
}

func (dispatcher stackOperatorDirectorDispatcherV0) dispatchOperatorDirectorObserveV0(
	ctx context.Context,
	message channel.OperatorMessageV0,
) channel.OperatorDirectorResponseV0 {
	if dispatcher.Bindings.ObserveDirectorGoal == nil {
		return stackOperatorDirectorPortUnavailableResponseV0(message, "observe")
	}
	runRef := stackOperatorDirectorRunRefV0(message)
	if runRef == "" {
		return stackOperatorDirectorBlockedResponseV0(message, "observe", "operator_director_observe_run_ref_required")
	}
	result, err := dispatcher.Bindings.ObserveDirectorGoal.Execute(ctx, orquestamcp.MCPObserveAppDirectorGoalToolInputV0{
		RequestID:     message.RequestRef,
		CorrelationID: message.ConversationRef,
		RunRef:        runRef,
		RequestedBy:   message.SenderRef,
	})
	if err != nil || result.Estado != orquestamcp.MCPObserveAppDirectorGoalEstadoOKV0 {
		return stackOperatorDirectorBlockedResponseV0(message, "observe", "operator_director_observe_unavailable")
	}
	return stackOperatorDirectorResponseV0(
		message,
		"accepted",
		"operator_director_observe_ready",
		stackOperatorDirectorJSONSummaryV0(result),
		[]string{"evidence-ref-operator-director-observe"},
	)
}

func (dispatcher stackOperatorDirectorDispatcherV0) dispatchOperatorDirectorLaunchHandoffV0(
	message channel.OperatorMessageV0,
	operation string,
) channel.OperatorDirectorResponseV0 {
	if dispatcher.Bindings.AutoprogrammingPrepareRun == nil && dispatcher.Bindings.ArrancarDirector == nil {
		return stackOperatorDirectorPortUnavailableResponseV0(message, operation)
	}
	return stackOperatorDirectorResponseV0(
		message,
		"blocked",
		"operator_director_"+operation+"_requires_structured_payload",
		"launch/handoff requiere payload estructurado por orquesta.autoprogramming.prepare_run.v0 u orquesta.apps.arrancar_director.v0",
		[]string{"evidence-ref-operator-director-" + operation + "-safe-blocked"},
	)
}

func (dispatcher stackOperatorDirectorDispatcherV0) dispatchOperatorDirectorControlV0(
	ctx context.Context,
	message channel.OperatorMessageV0,
	operation string,
) channel.OperatorDirectorResponseV0 {
	if dispatcher.Bindings.RunControl == nil {
		return stackOperatorDirectorPortUnavailableResponseV0(message, operation)
	}
	if !stackOperatorDirectorControlConfirmedV0(message) {
		return stackOperatorDirectorResponseV0(
			message,
			"blocked",
			"operator_director_control_confirmation_required",
			"stop/control requiere confirmacion explicita y run_ref opaca antes de tocar runtime",
			[]string{"evidence-ref-operator-director-control-confirmation-required"},
		)
	}
	runRef := stackOperatorDirectorRunRefV0(message)
	if runRef == "" {
		return stackOperatorDirectorBlockedResponseV0(message, operation, "operator_director_control_run_ref_required")
	}
	result, err := dispatcher.Bindings.RunControl.Execute(ctx, orquestamcp.MCPRunControlToolInputV0{
		RequestID:     message.RequestRef,
		CorrelationID: message.ConversationRef,
		Action:        "stop",
		RunRef:        runRef,
		RequestedBy:   message.SenderRef,
		Reason:        "operator_director_confirmed_control",
		Forced:        strings.Contains(strings.ToLower(message.Body), "forced"),
		EvidenceRefs:  message.EvidenceRefs,
	})
	if err != nil || result.Estado != orquestamcp.MCPRunControlEstadoOKV0 {
		return stackOperatorDirectorBlockedResponseV0(message, operation, "operator_director_control_unavailable")
	}
	return stackOperatorDirectorResponseV0(
		message,
		"accepted",
		"operator_director_control_applied",
		stackOperatorDirectorJSONSummaryV0(result),
		[]string{"evidence-ref-operator-director-control"},
	)
}

type stackOperatorDirectorMemoryStoreV0 struct {
	exchanges []channel.OperatorDirectorExchangeV0
}

func newStackOperatorDirectorMemoryStoreV0() *stackOperatorDirectorMemoryStoreV0 {
	return &stackOperatorDirectorMemoryStoreV0{}
}

func (store *stackOperatorDirectorMemoryStoreV0) SaveOperatorDirectorExchangeV0(
	_ context.Context,
	exchange channel.OperatorDirectorExchangeV0,
) error {
	store.exchanges = append(store.exchanges, exchange)
	return nil
}

func stackOperatorDirectorOperationV0(message channel.OperatorMessageV0) string {
	body := strings.ToLower(message.Body)
	switch message.Intent {
	case channel.OperatorMessageIntentStatusV0:
		return "status"
	case channel.OperatorMessageIntentObserveRunV0:
		return "observe"
	case channel.OperatorMessageIntentInstructionV0:
		if strings.Contains(body, "stop") || strings.Contains(body, "control") {
			return "control"
		}
		if strings.Contains(body, "launch") || strings.Contains(body, "arranca") {
			return "launch"
		}
		if strings.Contains(body, "handoff") || strings.Contains(body, "deriva") {
			return "handoff"
		}
	}
	for _, item := range []struct {
		op      string
		markers []string
	}{
		{"queue", []string{"queue", "cola"}},
		{"observe", []string{"observe", "observa"}},
		{"launch", []string{"launch", "arranca", "lanza"}},
		{"handoff", []string{"handoff", "deriva"}},
		{"control", []string{"control", "stop", "deten"}},
		{"status", []string{"status", "estado"}},
	} {
		for _, marker := range item.markers {
			if strings.Contains(body, marker) {
				return item.op
			}
		}
	}
	return "general"
}

func stackOperatorDirectorRunRefV0(message channel.OperatorMessageV0) string {
	for _, value := range append([]string{message.TargetRef}, strings.Fields(message.Body)...) {
		value = strings.Trim(value, ".,;:()[]{}")
		if strings.HasPrefix(value, "run-ref-") {
			return value
		}
	}
	return ""
}

func stackOperatorDirectorControlConfirmedV0(message channel.OperatorMessageV0) bool {
	body := strings.ToLower(message.Body)
	if strings.Contains(body, "confirmacion") ||
		strings.Contains(body, "confirmation") ||
		strings.Contains(body, "confirmed") ||
		strings.Contains(body, "confirmado") {
		return true
	}
	for _, ref := range message.EvidenceRefs {
		if strings.Contains(strings.ToLower(ref), "confirm") {
			return true
		}
	}
	return false
}

func stackOperatorDirectorPortUnavailableResponseV0(
	message channel.OperatorMessageV0,
	operation string,
) channel.OperatorDirectorResponseV0 {
	return stackOperatorDirectorResponseV0(
		message,
		"blocked",
		"operator_director_"+operation+"_port_unavailable",
		"puerto real no configurado para "+operation,
		[]string{"evidence-ref-operator-director-" + operation + "-port-unavailable"},
	)
}

func stackOperatorDirectorBlockedResponseV0(
	message channel.OperatorMessageV0,
	operation string,
	reason string,
) channel.OperatorDirectorResponseV0 {
	return stackOperatorDirectorResponseV0(
		message,
		"blocked",
		reason,
		operation+" blocked: "+reason,
		[]string{"evidence-ref-" + reason},
	)
}

func stackOperatorDirectorResponseV0(
	message channel.OperatorMessageV0,
	status string,
	summary string,
	text string,
	evidenceRefs []string,
) channel.OperatorDirectorResponseV0 {
	return channel.OperatorDirectorResponseV0{
		MessageRef:   message.MessageRef,
		Status:       status,
		Summary:      summary,
		ResponseRef:  "operator-director-response-ref-" + message.MessageRef,
		ResponseText: text,
		EvidenceRefs: evidenceRefs,
	}
}

func stackOperatorDirectorJSONSummaryV0(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "operator_director_response_ready"
	}
	return string(data)
}

func externalWorkRunGuardedExecutorV0(
	config ConfigV0,
	queueConfig RunQueueConfigV0,
) orquestamcp.MCPTransportExternalWorkRunExecutorV0 {
	executor := orquestamcp.MCPTransportExternalWorkRunExecutorV0(
		externalWorkRunExecutorV0(config, queueConfig),
	)
	executor = NewExternalWorkRunRuntimeCompatibilityGuardExecutorV0(executor, config.ExternalWorkRunRuntimeGuard)
	guard := config.ExternalWorkRunGuard
	if len(guard.Rules) == 0 {
		return executor
	}
	if strings.TrimSpace(guard.ProjectWorkDir) == "" {
		guard.ProjectWorkDir = strings.TrimSpace(config.Codex.ProjectWorkDir)
	}
	return NewExternalWorkRunProjectWorkDirGuardExecutorV0(executor, guard)
}

func buildStackHTTPHandlerV0(
	config ConfigV0,
	bindings orquestamcp.MCPTransportBindingsV0,
) http.Handler {
	handler := orquestaappgateway.NewHTTPHandlerV0(orquestaappgateway.ConfigV0{
		Clock:                             config.Clock,
		AppIntakeAssistant:                config.AppIntakeAssistant,
		ArrancarDirector:                  bindings.ArrancarDirector,
		PreviewDirector:                   bindings.PreviewDirector,
		ObserveDirectorGoal:               bindings.ObserveDirectorGoal,
		RequestAppChange:                  bindings.RequestAppChange,
		DirectorStats:                     bindings.DirectorStats,
		RunControl:                        bindings.RunControl,
		RuntimeModels:                     bindings.RuntimeModels,
		RunQueuePriority:                  bindings.RunQueuePriority,
		RunSupervisor:                     bindings.RunSupervisor,
		AppVCS:                            bindings.AppVCS,
		AutoprogrammingPrepareRun:         bindings.AutoprogrammingPrepareRun,
		AutoprogrammingObserveGoal:        bindings.AutoprogrammingObserveGoal,
		AutoprogrammingObserveActiveGoals: bindings.AutoprogrammingObserveActiveGoals,
		AutoprogrammingGoalStates:         bindings.AutoprogrammingGoalStates,
		AutoprogrammingEstadoVivoSource:   bindings.AutoprogrammingEstadoVivoSource,
		AutoprogrammingIdleSelfImprovementBudgetSource: bindings.AutoprogrammingIdleSelfImprovementBudgetSource,
		AutoprogrammingStatusDiagnostics:               bindings.AutoprogrammingStatusDiagnostics,
		AutoprogrammingGoalProgressPolicy:              bindings.AutoprogrammingGoalProgressPolicy,
		AllowLegacyAutoprogrammingSupervisorActions:    bindings.AllowLegacyAutoprogrammingSupervisorActions,
		ServerShutdown:           bindings.ServerShutdown,
		DomainWork:               bindings.DomainWork,
		DomainWorkRecords:        config.DomainDelivery.JobRecords,
		ExternalWorkDryRun:       bindings.ExternalWorkDryRun,
		ExternalWorkDryRunConfig: externalWorkRunStartConfigV0(config, normalizeRunQueueConfigV0(config.RunQueue)),
		ExternalWorkRun:          bindings.ExternalWorkRun,
		CodebaseQuery:            bindings.CodebaseQuery,
		CodebaseStatus:           bindings.CodebaseStatus,
		OpsAgentRuntimeDetail: NewCodexStackAgentRuntimeDetailHTTPHandlerV0(CodexStackAgentRuntimeDetailConfigV0{
			ReceiptStore:   config.Stores.ReceiptStore,
			RuntimeWorkDir: config.Codex.RuntimeWorkDir,
		}),
		Timeout:        config.Timeout,
		DirectorLimits: config.DirectorLimits,
	})
	return handler
}

func buildDirectorPortsV0(
	config ConfigV0,
) orquestaappdirectorservice.StartAppDirectorPortsV0 {
	progressSource, leaseSource := progressAndLeaseSourcesV0(config)
	return orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:                    config.Stores.RunStore,
		EventSink:                   ackRuntimeCleanupSinkV0(config),
		EventReader:                 eventReaderV0(config),
		OutboxLedger:                config.Stores.OutboxLedger,
		DeliverySource:              deliverySourceV0(config),
		ReviewGateSource:            reviewGateSourceV0(config),
		ReviewReworkReplanSource:    reviewReworkReplanSourceV0(config),
		AssessmentReplanSource:      assessmentReplanSourceV0(config),
		ProgressSource:              progressSource,
		RunControl:                  config.Stores.RunControl,
		RunControlTerminal:          config.Stores.RunControl,
		LeaseSource:                 leaseSource,
		DirectorDecisionSource:      directorDecisionSourceV0(config),
		DirectorTaskStore:           config.Stores.TaskStore,
		WorkflowTaskDefaultCapacity: config.Capacity.Tier,
		WaitStateWriter:             waitStateWriterV0(config),
		WaitStateStore:              waitStateStoreV0(config),
		OperationalPlanStateWriter:  operationalPlanStateWriterV0(config),
		OperationalPlanStateStore:   operationalPlanStateStoreV0(config),
		RequiredTestEvidenceStore:   requiredTestEvidenceStoreV0(config),
		RequiredTestRunner:          requiredTestRunnerV0(config),
		AutonomousDirectorPolicy:    config.AutonomousDirectorPolicy,
		GoalLauncher:                goalLauncherWithCodeContextPrepareFromConfigV0(config.AppGoalLauncher, config.CodeContext),
		GoalReworkLauncher:          goalLauncherWithCodeContextPrepareFromConfigV0(config.AppGoalReworkLauncher, config.CodeContext),
		GoalObserver:                goalFirstReconciledObserverFromConfigV0(config),
		GoalClosureValidator:        appGoalClosureValidatorV0(config),
		GoalStateStore:              config.Stores.AppGoalStateStore,
		GoalFirstRunMarkerStore:     appGoalFirstRunMarkerStoreV0(config),
		ExternalWaiter:              ackWaiterV0(config),
		OperationalClosureSource:    operationalClosureSourceV0(config),
		Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
			capacityDispatcherV0(config),
			stopperDispatcherV0(config),
			directorQuestionDispatcherV0(config),
		},
		BatchDispatchers: []orquestacionnucleoapp.OutboxBatchDispatcherBindingV0{
			agentBatchDispatcherV0(config),
		},
	}
}

func appGoalFirstRunMarkerStoreV0(
	config ConfigV0,
) orquestaappdirectorservice.AppDirectorGoalFirstRunMarkerStorePortV0 {
	store, ok := config.Stores.AppGoalStateStore.(orquestaappdirectorservice.AppDirectorGoalFirstRunMarkerStorePortV0)
	if !ok {
		return nil
	}
	return store
}

func estadoVivoSourceV0(
	config ConfigV0,
) orquestaestadovivo.FuenteEvidenciaEstadoPortV0 {
	fuentes := make([]orquestaestadovivo.FuenteEvidenciaEstadoPortV0, 0, 5)
	if config.Stores.RunStore != nil {
		fuentes = append(fuentes, EvidenciaEstadoRunStoreV0{Store: config.Stores.RunStore})
	}
	if config.Stores.AppGoalStateStore != nil {
		fuentes = append(fuentes, EvidenciaEstadoGoalStateV0{Store: config.Stores.AppGoalStateStore})
	}
	if markerStore := appGoalFirstRunMarkerStoreV0(config); markerStore != nil {
		fuentes = append(fuentes, EvidenciaEstadoMarkerV0{Store: markerStore})
	}
	if registry, ok := config.Stores.ProcessRegistry.(orquestacionnucleoapp.AgentProcessRegistryListPortV0); ok && registry != nil {
		fuentes = append(fuentes, EvidenciaEstadoProcesosV0{
			Registry:       registry,
			SnapshotSource: config.Codex.SnapshotSource,
		})
	}
	if config.Stores.AppGoalStateStore != nil && config.Stores.ReceiptStore != nil {
		fuentes = append(fuentes, EvidenciaEstadoReceiptsV0{
			GoalStateStore: config.Stores.AppGoalStateStore,
			ReceiptStore:   config.Stores.ReceiptStore,
		})
	}
	if len(fuentes) == 0 {
		return nil
	}
	return EvidenciaEstadoAgregadorV0{Fuentes: fuentes}
}

func appGoalClosureValidatorV0(
	config ConfigV0,
) orquestagoal.GoalWorkClosureValidatorPortV0 {
	base := config.AppGoalClosureValidator
	if base == nil {
		base = orquestagoal.DefaultGoalWorkClosureValidatorV0{}
	}
	domainValidator := domainWorkGoalReceiptClosureValidatorV0{
		Base:   base,
		Ledger: domainWorkSubmissionRecordReaderV0(config.DomainDelivery.Ledger),
	}
	return frozenRequiredTestsClosureValidatorV0{
		Base:           domainValidator,
		ProjectWorkDir: strings.TrimSpace(config.Codex.ProjectWorkDir),
	}
}

func requiredTestEvidenceStoreV0(
	config ConfigV0,
) orquestacionnucleoapp.RequiredTestEvidenceStorePortV0 {
	if config.Stores.RequiredTestEvidenceStore != nil {
		return config.Stores.RequiredTestEvidenceStore
	}
	store, _ := config.Stores.TaskStore.(orquestacionnucleoapp.RequiredTestEvidenceStorePortV0)
	return store
}

func waitStateWriterV0(
	config ConfigV0,
) orquestacionnucleoapp.WorkflowTaskWaitStateWriterPortV0 {
	writer, _ := config.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWaitStateWriterPortV0)
	return writer
}

func waitStateStoreV0(
	config ConfigV0,
) orquestacionnucleoapp.WorkflowTaskWaitStateStorePortV0 {
	if config.Stores.WaitStateStore != nil {
		return config.Stores.WaitStateStore
	}
	store, _ := config.Stores.TaskStore.(orquestacionnucleoapp.WorkflowTaskWaitStateStorePortV0)
	return store
}

func operationalPlanStateWriterV0(
	config ConfigV0,
) orquestacionnucleoapp.OperationalDirectorPlanStateWriterPortV0 {
	if config.Stores.OperationalPlanStateWriter != nil {
		return config.Stores.OperationalPlanStateWriter
	}
	writer, _ := config.Stores.TaskStore.(orquestacionnucleoapp.OperationalDirectorPlanStateWriterPortV0)
	return writer
}

func operationalPlanStateStoreV0(
	config ConfigV0,
) orquestacionnucleoapp.OperationalDirectorPlanStateStorePortV0 {
	if config.Stores.OperationalPlanStateStore != nil {
		return config.Stores.OperationalPlanStateStore
	}
	store, _ := config.Stores.OperationalPlanStateWriter.(orquestacionnucleoapp.OperationalDirectorPlanStateStorePortV0)
	if store != nil {
		return store
	}
	store, _ = config.Stores.TaskStore.(orquestacionnucleoapp.OperationalDirectorPlanStateStorePortV0)
	return store
}

func ackRuntimeCleanupSinkV0(config ConfigV0) orquestacionnucleoapp.EventSinkPortV0 {
	return ackRuntimeCleanupEventSinkV0{
		Inner: config.Stores.EventSink,
		Stopper: orquestacionnucleoapp.ProcessAgentStopperV0{
			Registry: config.Stores.ProcessRegistry,
			Runtime:  config.Codex.ProcessStopper,
		},
	}
}

func startOnlyPortsV0(
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
) orquestaappdirectorservice.StartAppDirectorPortsV0 {
	ports.DirectorDecisionSource = nil
	ports.ExternalWaiter = nil
	return ports
}
