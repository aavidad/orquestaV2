package orquestaruntimecodexappserver

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

func (protocol serverCodexAppServerLazyTmuxProtocolV0) UpdateThreadSettingsV0(
	ctx context.Context,
	params serverCodexAppServerThreadSettingsUpdateParamsV0,
) error {
	if err := protocol.ensureV0(ctx); err != nil {
		return err
	}
	return protocol.Inner.UpdateThreadSettingsV0(ctx, params)
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

// withVerifiedGenerationV0 scopes all RPCs in call to one exact generation.
// The postverify runs even after a RPC failure, so a caller never retains an
// ID that crossed a runtime rotation.
func (protocol serverCodexAppServerLazyTmuxProtocolV0) withVerifiedGenerationV0(
	ctx context.Context,
	expected string,
	call func(serverCodexAppServerProtocolPortV0, string) error,
) (string, error) {
	if err := protocol.ensureV0(ctx); err != nil {
		return "", err
	}
	lease, err := protocol.Backend.acquireTmuxLeaseGuardV0(ctx)
	if err != nil {
		return "", err
	}
	defer lease.releaseV0()
	marker, err := protocol.Backend.verifiedTmuxGenerationWithLeaseV0(ctx, lease, expected)
	if err != nil {
		return "", err
	}
	callErr := call(protocol.Inner, marker.GenerationRef)
	if _, verifyErr := protocol.Backend.verifiedTmuxGenerationWithLeaseV0(ctx, lease, marker.GenerationRef); verifyErr != nil {
		return marker.GenerationRef, verifyErr
	}
	return marker.GenerationRef, callErr
}
