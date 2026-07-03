package main

import (
	"context"
	"errors"
	"strings"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimecodexappserver "orquesta/modulos/orquesta-runtime-codex-appserver"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
	orquestaserver "orquesta/modulos/orquesta-server"
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
	Starter      orquestaruntimecodexgoal.CodexGoalStarterPortV0
	Observer     orquestaruntimecodexgoal.CodexGoalObserverPortV0
	ShutdownHook orquestaserver.RuntimeShutdownHookPortV0
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

type serverGoalSupervisorV0 struct {
	serverStackSupervisorV0
	launcher orquestaserver.IdleSelfImprovementGoalLauncherPortV0
	observer orquestaserver.IdleSelfImprovementGoalObserverPortV0
}

func serverSupervisorWithCodexGoalBackendV0(
	base serverStackSupervisorV0,
	backend serverCodexGoalBackendV0,
) orquestaserver.SupervisorPortV0 {
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

func serverGoalWorkObserverFromBackendV0(
	backend serverCodexGoalBackendV0,
) orquestagoal.GoalWorkObservationPortV0 {
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

func serverGoalObservationFingerprintFromBackendV0(
	backend serverCodexGoalBackendV0,
	enabled bool,
) orquestaserver.GoalObservationFingerprintPortV0 {
	if !enabled || backend.Observer == nil {
		return nil
	}
	fingerprint, ok := backend.Observer.(orquestaserver.GoalObservationFingerprintPortV0)
	if !ok {
		return nil
	}
	return fingerprint
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

func codexGoalTimeoutMSFromEnvV0() int {
	return intEnvOrDefaultV0(envCodexGoalTimeoutMSV0, defaultCodexGoalTimeoutMSV0)
}

func codexGoalPreflightTimeoutMSFromEnvV0() int {
	return intEnvOrDefaultV0(envCodexGoalPreflightTimeoutMSV0, defaultCodexGoalPreflightTimeoutMSV0)
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
