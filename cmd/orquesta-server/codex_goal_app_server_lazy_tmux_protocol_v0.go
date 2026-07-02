package main

import "context"

type serverCodexAppServerLazyTmuxProtocolV0 struct {
	Backend   serverCodexAppServerTmuxBackendV0
	Inner     serverCodexAppServerWebSocketProtocolV0
	Preflight serverCodexAppServerWebSocketProtocolV0
}

func (protocol serverCodexAppServerLazyTmuxProtocolV0) ProbeV0(ctx context.Context) error {
	if err := protocol.ensureV0(ctx); err != nil {
		return err
	}
	return protocol.Inner.ProbeV0(ctx)
}

func (protocol serverCodexAppServerLazyTmuxProtocolV0) StartThreadV0(
	ctx context.Context,
	params serverCodexAppServerThreadStartParamsV0,
) (serverCodexAppServerThreadV0, error) {
	if err := protocol.ensureV0(ctx); err != nil {
		return serverCodexAppServerThreadV0{}, err
	}
	return protocol.Inner.StartThreadV0(ctx, params)
}

func (protocol serverCodexAppServerLazyTmuxProtocolV0) SetGoalV0(
	ctx context.Context,
	params serverCodexAppServerThreadGoalSetParamsV0,
) (serverCodexAppServerThreadGoalV0, error) {
	if err := protocol.ensureV0(ctx); err != nil {
		return serverCodexAppServerThreadGoalV0{}, err
	}
	return protocol.Inner.SetGoalV0(ctx, params)
}

func (protocol serverCodexAppServerLazyTmuxProtocolV0) StartTurnV0(
	ctx context.Context,
	params serverCodexAppServerTurnStartParamsV0,
) (serverCodexAppServerTurnV0, error) {
	if err := protocol.ensureV0(ctx); err != nil {
		return serverCodexAppServerTurnV0{}, err
	}
	return protocol.Inner.StartTurnV0(ctx, params)
}

func (protocol serverCodexAppServerLazyTmuxProtocolV0) GetGoalV0(
	ctx context.Context,
	threadID string,
) (*serverCodexAppServerThreadGoalV0, error) {
	if err := protocol.ensureV0(ctx); err != nil {
		return nil, err
	}
	return protocol.Inner.GetGoalV0(ctx, threadID)
}

func (protocol serverCodexAppServerLazyTmuxProtocolV0) ReadThreadV0(
	ctx context.Context,
	threadID string,
	includeEvents bool,
) (serverCodexAppServerThreadReadV0, error) {
	if err := protocol.ensureV0(ctx); err != nil {
		return serverCodexAppServerThreadReadV0{}, err
	}
	return protocol.Inner.ReadThreadV0(ctx, threadID, includeEvents)
}

func (protocol serverCodexAppServerLazyTmuxProtocolV0) ensureV0(ctx context.Context) error {
	return protocol.Backend.EnsureV0(ctx, protocol.Preflight)
}
