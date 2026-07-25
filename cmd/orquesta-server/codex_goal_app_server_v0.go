package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacapacity "orquesta/modulos/orquesta-capacity"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexappserver "orquesta/modulos/orquesta-runtime-codex-appserver"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

const (
	codexGoalBackendAppServerProxyV0      = orquestaruntimecodexappserver.CodexGoalBackendAppServerProxyV0
	codexGoalBackendAppServerTmuxV0       = orquestaruntimecodexappserver.CodexGoalBackendAppServerTmuxV0
	codexAppServerGoalObjectiveMaxRunesV0 = 4000
	codexAppServerTmuxDirV0               = orquestaruntimecodexappserver.CodexAppServerTmuxDirV0
	codexAppServerTmuxMarkerFileV0        = orquestaruntimecodexappserver.CodexAppServerTmuxMarkerFileV0
	codexAppServerTmuxDefaultTimeoutV0    = orquestaruntimecodexappserver.CodexAppServerTmuxDefaultTimeoutV0
	codexAppServerTmuxMaxSocketPathV0     = orquestaruntimecodexappserver.CodexAppServerTmuxMaxSocketPathV0
)

type serverCodexGoalBackendV0 struct {
	GoalLauncher    orquestagoal.GoalWorkLauncherPortV0
	GoalObserver    orquestagoal.GoalWorkObservationPortV0
	Starter         orquestaruntimecodexgoal.CodexGoalStarterPortV0
	Observer        orquestaruntimecodexgoal.CodexGoalObserverPortV0
	Controller      serverCodexGoalControllerV0
	ClaudeControl   serverClaudeGoalControllerV0
	GeminiControl   serverGeminiGoalControllerV0
	WorkspaceLookup serverGoalWorkspaceBindingLookupV0
	ShutdownHook    orquestaserver.RuntimeShutdownHookPortV0
}

type serverCodexAppServerGoalBackendV0 = orquestaruntimecodexappserver.GoalBackendV0
type serverCodexUnavailableGoalBackendV0 = orquestaruntimecodexappserver.UnavailableGoalBackendV0
type serverCodexAppServerGoalRuntimeV0 = orquestaruntimecodexappserver.GoalRuntimeV0
type serverCodexAppServerCommandProtocolV0 = orquestaruntimecodexappserver.CommandProtocolV0
type serverCodexAppServerLazyTmuxProtocolV0 = orquestaruntimecodexappserver.LazyTmuxProtocolV0
type serverCodexAppServerProbePortV0 = orquestaruntimecodexappserver.ProbePortV0
type serverCodexAppServerProtocolPortV0 = orquestaruntimecodexappserver.ProtocolPortV0
type serverCodexAppServerTmuxBackendV0 = orquestaruntimecodexappserver.TmuxBackendV0
type serverCodexAppServerWebSocketProtocolV0 = orquestaruntimecodexappserver.WebSocketProtocolV0
type serverCodexAppServerRPCResponseV0 = orquestaruntimecodexappserver.RPCResponseV0
type serverCodexAppServerReadItemV0 = orquestaruntimecodexappserver.ReadItemV0
type serverCodexAppServerReadTurnV0 = orquestaruntimecodexappserver.ReadTurnV0
type serverCodexAppServerThreadGoalGetResponseV0 = orquestaruntimecodexappserver.ThreadGoalGetResponseV0
type serverCodexAppServerThreadGoalSetParamsV0 = orquestaruntimecodexappserver.ThreadGoalSetParamsV0
type serverCodexAppServerThreadGoalV0 = orquestaruntimecodexappserver.ThreadGoalV0
type serverCodexAppServerThreadSettingsUpdateParamsV0 = orquestaruntimecodexappserver.ThreadSettingsUpdateParamsV0
type serverCodexAppServerThreadReadResponseV0 = orquestaruntimecodexappserver.ThreadReadResponseV0
type serverCodexAppServerThreadReadV0 = orquestaruntimecodexappserver.ThreadReadV0
type serverCodexAppServerThreadStartParamsV0 = orquestaruntimecodexappserver.ThreadStartParamsV0
type serverCodexAppServerThreadStartResponseV0 = orquestaruntimecodexappserver.ThreadStartResponseV0
type serverCodexAppServerThreadStatusV0 = orquestaruntimecodexappserver.ThreadStatusV0
type serverCodexAppServerThreadV0 = orquestaruntimecodexappserver.ThreadV0
type serverCodexAppServerTimestampV0 = orquestaruntimecodexappserver.TimestampV0
type serverCodexAppServerTurnStartParamsV0 = orquestaruntimecodexappserver.TurnStartParamsV0
type serverCodexAppServerTurnStartResponseV0 = orquestaruntimecodexappserver.TurnStartResponseV0
type serverCodexAppServerTurnV0 = orquestaruntimecodexappserver.TurnV0

type serverCodexGoalCostRoutingStarterV0 struct {
	Backend      serverCodexAppServerGoalBackendV0
	ModelRouting orquestaappcodexstack.CodexModelRoutingConfigV0
}

func (starter serverCodexGoalCostRoutingStarterV0) StartCodexGoalV0(
	ctx context.Context,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) (orquestaruntimecodexgoal.CodexGoalStartReceiptV0, error) {
	decision, model, err := serverCodexGoalModelRouteForPacketV0(starter.ModelRouting, packet)
	if err != nil {
		return orquestaruntimecodexgoal.CodexGoalStartReceiptV0{
			Status:    orquestagoal.GoalStatusInvalidV0,
			GoalRef:   strings.TrimSpace(packet.GoalRef),
			IssueCode: "codex_goal_model_routing_rejected",
		}, err
	}
	backend := starter.Backend
	backend.Model = model
	backend.ReasoningEffort = decision.ReasoningEffort
	return backend.StartCodexGoalV0(ctx, packet)
}

func serverCodexGoalModelRouteForPacketV0(
	routing orquestaappcodexstack.CodexModelRoutingConfigV0,
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) (orquestacapacity.ModelRoutingDecisionV0, string, error) {
	taskRef := strings.TrimSpace(packet.GoalRef)
	reasoningEffort := strings.TrimSpace(packet.ReasoningEffort)
	if reasoningEffort != "" && !orquestagoal.ValidGoalReasoningEffortV0(reasoningEffort) {
		return orquestacapacity.ModelRoutingDecisionV0{
			TaskRef:      taskRef,
			Rejected:     true,
			RejectionRef: "model-routing-reasoning-effort-invalid",
		}, "", fmt.Errorf("model_routing_rejected:model-routing-reasoning-effort-invalid")
	}
	request := orquestacapacity.ModelRoutingRequestV0{
		TaskRef: taskRef,
		Level:   orquestacapacity.ModelRoutingLevelNormalV0,
		Trivial: serverCodexGoalTaskCostClassForPacketV0(packet) == orquestaruntimecodexgoal.CodexGoalTaskCostClassDocV0,
	}
	// An explicit GoalRef route keeps the critical/Sol path available with the
	// policy's existing causal authorization requirements.
	if declared, ok := routing.TaskRoutes[taskRef]; ok {
		request = declared
		request.TaskRef = taskRef
	}
	decision := orquestacapacity.ResolveModelRoutingV0(routing.Policy, request)
	if decision.Rejected {
		return decision, "", fmt.Errorf("model_routing_rejected:%s", decision.RejectionRef)
	}
	// GoalWorkSpecV0 carries the effort already authorized for the WorkItem and
	// its EffectIntent. Routing selects the model but must not re-decide it.
	if reasoningEffort != "" {
		decision.ReasoningEffort = reasoningEffort
	}
	model := strings.TrimSpace(routing.ModelAlias[decision.SelectedModelRef])
	if model == "" {
		return decision, "", fmt.Errorf("model_routing_alias_missing:%s", decision.SelectedModelRef)
	}
	return decision, model, nil
}

func serverCodexGoalTaskCostClassForPacketV0(
	packet orquestaruntimecodexgoal.CodexGoalStartPacketV0,
) string {
	derived := orquestaruntimecodexgoal.CodexGoalTaskCostClassForWriteSetV0(packet.WriteSet)
	declared := strings.TrimSpace(packet.TaskCostClass)
	if declared == "" || declared != derived {
		return derived
	}
	return declared
}

type serverGoalSupervisorV0 struct {
	serverStackSupervisorV0
	launcher orquestaserver.IdleSelfImprovementGoalLauncherPortV0
	observer orquestaserver.IdleSelfImprovementGoalObserverPortV0
}

func serverSupervisorWithCodexGoalBackendV0(
	base serverStackSupervisorV0,
	backend serverCodexGoalBackendV0,
) orquestaserver.SupervisorPortV0 {
	if backend.GoalLauncher != nil && backend.GoalObserver != nil {
		return serverGoalSupervisorV0{
			serverStackSupervisorV0: base,
			launcher:                backend.GoalLauncher,
			observer:                backend.GoalObserver,
		}
	}
	if backend.Starter == nil || backend.Observer == nil {
		return base
	}
	return serverGoalSupervisorV0{
		serverStackSupervisorV0: base,
		launcher: orquestaruntimecodexgoal.CodexGoalLauncherV0{
			Starter: backend.Starter,
		},
		observer: orquestaruntimecodexgoal.CodexGoalObserverV0{
			Observer: backend.Observer,
		},
	}
}

func serverGoalWorkLauncherFromBackendV0(
	backend serverCodexGoalBackendV0,
) orquestagoal.GoalWorkLauncherPortV0 {
	if backend.GoalLauncher != nil {
		return backend.GoalLauncher
	}
	if backend.Starter == nil {
		return nil
	}
	launcher := orquestaruntimecodexgoal.CodexGoalLauncherV0{
		Starter: backend.Starter,
	}
	active := serverGoalActiveShutdownWorkReaderFromBackendV0(backend)
	cleaner := serverGoalActiveShutdownWorkCleanerFromBackendV0(backend)
	if active != nil || cleaner != nil {
		return serverGoalWorkLauncherWithActiveShutdownWorkV0{
			Inner:         launcher,
			ActiveWork:    active,
			ActiveCleaner: cleaner,
		}
	}
	return launcher
}

// serverAutoprogrammingGoalWorkspaceLauncherV0 preserves the normal app Goal
// backend while routing only autoprogramming work to its physical workspace.
// The stack's context and required-test wrappers remain outside this selector.
type serverAutoprogrammingGoalWorkspaceLauncherV0 struct {
	AppGoal             orquestagoal.GoalWorkLauncherPortV0
	AutoprogrammingGoal orquestagoal.GoalWorkLauncherPortV0
}

func serverGoalWorkLauncherForAppAndAutoprogrammingV0(
	appGoal serverCodexGoalBackendV0,
	autoprogrammingGoal serverCodexGoalBackendV0,
) orquestagoal.GoalWorkLauncherPortV0 {
	appLauncher := serverGoalWorkLauncherFromBackendV0(appGoal)
	autoprogrammingLauncher := serverGoalWorkLauncherFromBackendV0(autoprogrammingGoal)
	if appLauncher == nil && autoprogrammingLauncher == nil {
		return nil
	}
	return serverAutoprogrammingGoalWorkspaceLauncherV0{
		AppGoal:             appLauncher,
		AutoprogrammingGoal: autoprogrammingLauncher,
	}
}

func (launcher serverAutoprogrammingGoalWorkspaceLauncherV0) LaunchGoalWorkV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	delegate := launcher.AppGoal
	if strings.TrimSpace(spec.WorkKind) == orquestaautoprogramming.AutoprogrammingGoalWorkKindV0 {
		delegate = launcher.AutoprogrammingGoal
	}
	if delegate == nil {
		return orquestagoal.GoalLaunchReceiptV0{}, errors.New("autoprogramming_goal_workspace_launcher_unavailable")
	}
	return delegate.LaunchGoalWorkV0(ctx, spec)
}

func (launcher serverAutoprogrammingGoalWorkspaceLauncherV0) ReadActiveShutdownWorkV0(
	ctx context.Context,
	request orquestaservershutdown.ActiveShutdownWorkRequestV0,
) (orquestaservershutdown.ActiveShutdownWorkResultV0, error) {
	return serverAutoprogrammingGoalReadActiveShutdownWorkV0(ctx, request, launcher.AppGoal, launcher.AutoprogrammingGoal)
}

func (launcher serverAutoprogrammingGoalWorkspaceLauncherV0) CleanupActiveShutdownWorkV0(
	ctx context.Context,
	command orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0,
) (orquestaservershutdown.ActiveShutdownWorkCleanupResultV0, error) {
	return serverAutoprogrammingGoalCleanupActiveShutdownWorkV0(ctx, command, launcher.AppGoal, launcher.AutoprogrammingGoal)
}

func (launcher serverAutoprogrammingGoalWorkspaceLauncherV0) ActiveShutdownWorkIdentityV0() string {
	return serverAutoprogrammingGoalActiveShutdownWorkIdentityV0(launcher.AppGoal, launcher.AutoprogrammingGoal)
}

func serverGoalWorkObserverFromBackendV0(
	backend serverCodexGoalBackendV0,
) orquestagoal.GoalWorkObservationPortV0 {
	if backend.GoalObserver != nil {
		return backend.GoalObserver
	}
	if backend.Observer == nil {
		return nil
	}
	observer := orquestaruntimecodexgoal.CodexGoalObserverV0{
		Observer: backend.Observer,
	}
	active := serverGoalActiveShutdownWorkReaderFromBackendV0(backend)
	cleaner := serverGoalActiveShutdownWorkCleanerFromBackendV0(backend)
	if active != nil || cleaner != nil {
		return serverGoalWorkObserverWithActiveShutdownWorkV0{
			Inner:         observer,
			ActiveWork:    active,
			ActiveCleaner: cleaner,
		}
	}
	return observer
}

type serverAutoprogrammingGoalWorkspaceObserverV0 struct {
	AppGoal             orquestagoal.GoalWorkObservationPortV0
	AutoprogrammingGoal orquestagoal.GoalWorkObservationPortV0
	WorkspaceLookup     serverGoalWorkspaceBindingLookupV0
}

func serverGoalWorkObserverForAppAndAutoprogrammingV0(
	appGoal serverCodexGoalBackendV0,
	autoprogrammingGoal serverCodexGoalBackendV0,
) orquestagoal.GoalWorkObservationPortV0 {
	appObserver := serverGoalWorkObserverFromBackendV0(appGoal)
	autoprogrammingObserver := serverGoalWorkObserverFromBackendV0(autoprogrammingGoal)
	lookup := serverCodexGoalWorkspaceLookupFromBackendV0(autoprogrammingGoal)
	if lookup == nil {
		return appObserver
	}
	return serverAutoprogrammingGoalWorkspaceObserverV0{
		AppGoal:             appObserver,
		AutoprogrammingGoal: autoprogrammingObserver,
		WorkspaceLookup:     lookup,
	}
}

func (observer serverAutoprogrammingGoalWorkspaceObserverV0) ObserveGoalWorkV0(
	ctx context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	delegate, err := observer.delegateForGoalV0(ctx, request.GoalRef)
	if err != nil {
		return orquestagoal.GoalWorkResultV0{}, err
	}
	if delegate == nil {
		return orquestagoal.GoalWorkResultV0{}, errors.New("autoprogramming_goal_workspace_observer_unavailable")
	}
	return delegate.ObserveGoalWorkV0(ctx, request)
}

func (observer serverAutoprogrammingGoalWorkspaceObserverV0) ReadActiveShutdownWorkV0(
	ctx context.Context,
	request orquestaservershutdown.ActiveShutdownWorkRequestV0,
) (orquestaservershutdown.ActiveShutdownWorkResultV0, error) {
	return serverAutoprogrammingGoalReadActiveShutdownWorkV0(ctx, request, observer.AppGoal, observer.AutoprogrammingGoal)
}

func (observer serverAutoprogrammingGoalWorkspaceObserverV0) CleanupActiveShutdownWorkV0(
	ctx context.Context,
	command orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0,
) (orquestaservershutdown.ActiveShutdownWorkCleanupResultV0, error) {
	return serverAutoprogrammingGoalCleanupActiveShutdownWorkV0(ctx, command, observer.AppGoal, observer.AutoprogrammingGoal)
}

func (observer serverAutoprogrammingGoalWorkspaceObserverV0) ActiveShutdownWorkIdentityV0() string {
	return serverAutoprogrammingGoalActiveShutdownWorkIdentityV0(observer.AppGoal, observer.AutoprogrammingGoal)
}

func (observer serverAutoprogrammingGoalWorkspaceObserverV0) delegateForGoalV0(
	ctx context.Context,
	goalRef string,
) (orquestagoal.GoalWorkObservationPortV0, error) {
	if observer.WorkspaceLookup == nil {
		return observer.AppGoal, nil
	}
	found, err := observer.WorkspaceLookup.HasGoalWorkspaceBindingV0(ctx, goalRef)
	if err != nil {
		return nil, err
	}
	if found {
		return observer.AutoprogrammingGoal, nil
	}
	return observer.AppGoal, nil
}

func serverAutoprogrammingGoalReadActiveShutdownWorkV0(
	ctx context.Context,
	request orquestaservershutdown.ActiveShutdownWorkRequestV0,
	ports ...interface{},
) (orquestaservershutdown.ActiveShutdownWorkResultV0, error) {
	out := orquestaservershutdown.ActiveShutdownWorkResultV0{}
	seen := map[string]struct{}{}
	for _, port := range ports {
		reader, ok := port.(orquestaservershutdown.ActiveShutdownWorkReaderPortV0)
		if !ok || reader == nil || serverAutoprogrammingGoalActiveShutdownWorkPortSeenV0(seen, reader) {
			continue
		}
		active, err := reader.ReadActiveShutdownWorkV0(ctx, request)
		if err != nil {
			return orquestaservershutdown.ActiveShutdownWorkResultV0{}, err
		}
		out.ActiveWorks = append(out.ActiveWorks, active.ActiveWorks...)
		out.EvidenceRefs = append(out.EvidenceRefs, active.EvidenceRefs...)
	}
	return out, nil
}

func serverAutoprogrammingGoalCleanupActiveShutdownWorkV0(
	ctx context.Context,
	command orquestaservershutdown.ActiveShutdownWorkCleanupCommandV0,
	ports ...interface{},
) (orquestaservershutdown.ActiveShutdownWorkCleanupResultV0, error) {
	out := orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{}
	seen := map[string]struct{}{}
	for _, port := range ports {
		cleaner, ok := port.(orquestaservershutdown.ActiveShutdownWorkCleanerPortV0)
		if !ok || cleaner == nil || serverAutoprogrammingGoalActiveShutdownWorkPortSeenV0(seen, cleaner) {
			continue
		}
		cleaned, err := cleaner.CleanupActiveShutdownWorkV0(ctx, command)
		if err != nil {
			return orquestaservershutdown.ActiveShutdownWorkCleanupResultV0{}, err
		}
		out.CleanedWorkCount += cleaned.CleanedWorkCount
		out.EvidenceRefs = append(out.EvidenceRefs, cleaned.EvidenceRefs...)
	}
	return out, nil
}

func serverAutoprogrammingGoalActiveShutdownWorkIdentityV0(ports ...interface{}) string {
	seen := map[string]struct{}{}
	identities := make([]string, 0, len(ports))
	for _, port := range ports {
		identity, ok := port.(orquestaservershutdown.ActiveShutdownWorkIdentityPortV0)
		if !ok || identity == nil {
			continue
		}
		value := strings.TrimSpace(identity.ActiveShutdownWorkIdentityV0())
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		identities = append(identities, fmt.Sprintf("%d:%s", len(value), value))
	}
	return strings.Join(identities, "|")
}

func serverAutoprogrammingGoalActiveShutdownWorkPortSeenV0(seen map[string]struct{}, port interface{}) bool {
	identity, ok := port.(orquestaservershutdown.ActiveShutdownWorkIdentityPortV0)
	if !ok || identity == nil {
		return false
	}
	value := strings.TrimSpace(identity.ActiveShutdownWorkIdentityV0())
	if value == "" {
		return false
	}
	if _, exists := seen[value]; exists {
		return true
	}
	seen[value] = struct{}{}
	return false
}

func serverCodexGoalWorkspaceLookupFromBackendV0(
	backend serverCodexGoalBackendV0,
) serverGoalWorkspaceBindingLookupV0 {
	if backend.WorkspaceLookup != nil {
		return backend.WorkspaceLookup
	}
	client, ok := backend.Observer.(serverCodexAppServerGoalBackendV0)
	if !ok || client.WorkspaceRouter == nil {
		return nil
	}
	lookup, _ := client.WorkspaceRouter.(serverGoalWorkspaceBindingLookupV0)
	return lookup
}

func serverGoalObservationFingerprintFromBackendV0(
	backend serverCodexGoalBackendV0,
	enabled bool,
) orquestaserver.GoalObservationFingerprintPortV0 {
	if !enabled {
		return nil
	}
	if fingerprint, ok := backend.GoalObserver.(orquestaserver.GoalObservationFingerprintPortV0); ok {
		return fingerprint
	}
	fingerprint, _ := backend.Observer.(orquestaserver.GoalObservationFingerprintPortV0)
	return fingerprint
}

type serverAutoprogrammingGoalWorkspaceFingerprintV0 struct {
	AppGoal             orquestaserver.GoalObservationFingerprintPortV0
	AutoprogrammingGoal orquestaserver.GoalObservationFingerprintPortV0
	WorkspaceLookup     serverGoalWorkspaceBindingLookupV0
}

func serverGoalObservationFingerprintForAppAndAutoprogrammingV0(
	appGoal serverCodexGoalBackendV0,
	autoprogrammingGoal serverCodexGoalBackendV0,
	enabled bool,
) orquestaserver.GoalObservationFingerprintPortV0 {
	appFingerprint := serverGoalObservationFingerprintFromBackendV0(appGoal, enabled)
	autoprogrammingFingerprint := serverGoalObservationFingerprintFromBackendV0(autoprogrammingGoal, enabled)
	lookup := serverCodexGoalWorkspaceLookupFromBackendV0(autoprogrammingGoal)
	if lookup == nil || (appFingerprint == nil && autoprogrammingFingerprint == nil) {
		return appFingerprint
	}
	return serverAutoprogrammingGoalWorkspaceFingerprintV0{
		AppGoal:             appFingerprint,
		AutoprogrammingGoal: autoprogrammingFingerprint,
		WorkspaceLookup:     lookup,
	}
}

func (fingerprint serverAutoprogrammingGoalWorkspaceFingerprintV0) FingerprintGoalObservationV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
) (orquestagoal.GoalObservationFingerprintV0, bool, error) {
	found, err := fingerprint.WorkspaceLookup.HasGoalWorkspaceBindingV0(ctx, state.GoalRef)
	if err != nil {
		return orquestagoal.GoalObservationFingerprintV0{}, true, err
	}
	delegate := fingerprint.AppGoal
	if found {
		delegate = fingerprint.AutoprogrammingGoal
	}
	if delegate == nil {
		return orquestagoal.GoalObservationFingerprintV0{}, false, nil
	}
	return delegate.FingerprintGoalObservationV0(ctx, state)
}

func (supervisor serverGoalSupervisorV0) LaunchGoalWorkV0(
	ctx context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	if supervisor.launcher == nil {
		return orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:        orquestagoal.GoalStatusInvalidV0,
			GoalRef:       strings.TrimSpace(spec.GoalRef),
			Issues: []orquestagoal.GoalWorkIssueV0{{
				Code: orquestaruntimecodexgoal.ErrCodexGoalStarterMissingV0,
			}},
		}, errors.New(orquestaruntimecodexgoal.ErrCodexGoalStarterMissingV0)
	}
	return supervisor.launcher.LaunchGoalWorkV0(ctx, spec)
}

func (supervisor serverGoalSupervisorV0) ObserveGoalWorkV0(
	ctx context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	if supervisor.observer == nil {
		return orquestagoal.GoalWorkResultV0{
			SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
			Status:          orquestagoal.GoalStatusInvalidV0,
			GoalRef:         strings.TrimSpace(request.GoalRef),
			ExternalGoalRef: strings.TrimSpace(request.ExternalGoalRef),
			Issues: []orquestagoal.GoalWorkIssueV0{{
				Code: orquestaruntimecodexgoal.ErrCodexGoalObserverMissingV0,
			}},
		}, errors.New(orquestaruntimecodexgoal.ErrCodexGoalObserverMissingV0)
	}
	return supervisor.observer.ObserveGoalWorkV0(ctx, request)
}

func codexAppServerIssueCodeForErrorV0(err error, fallback string) string {
	return orquestaruntimecodexappserver.CodexAppServerIssueCodeForErrorV0(err, fallback)
}

func codexAppServerIssueCodeFromCommandFailureV0(stderr string, err error) string {
	return orquestaruntimecodexappserver.CodexAppServerIssueCodeFromCommandFailureV0(stderr, err)
}

func codexAppServerIssueCodeFromLogFileV0(path string) string {
	return orquestaruntimecodexappserver.CodexAppServerIssueCodeFromLogFileV0(path)
}

func codexAppServerTmuxCodeHomePathV0(config orquestaserver.ConfigV0) (string, error) {
	return orquestaruntimecodexappserver.CodexAppServerTmuxCodeHomePathV0(codexAppServerConfigFromServerConfigV0(config))
}

func codexAppServerTmuxSessionNameV0(config orquestaserver.ConfigV0) string {
	return orquestaruntimecodexappserver.CodexAppServerTmuxSessionNameV0(codexAppServerConfigFromServerConfigV0(config))
}

func codexAppServerTmuxSocketPathV0(config orquestaserver.ConfigV0) (string, error) {
	return orquestaruntimecodexappserver.CodexAppServerTmuxSocketPathV0(codexAppServerConfigFromServerConfigV0(config))
}

func codexAppServerTmuxStartupTimeoutV0(timeout time.Duration) time.Duration {
	return orquestaruntimecodexappserver.CodexAppServerTmuxStartupTimeoutV0(timeout)
}
