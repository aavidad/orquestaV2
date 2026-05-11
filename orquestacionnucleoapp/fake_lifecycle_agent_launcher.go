package orquestacionnucleoapp

import (
	"context"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type FakeLifecycleAgentLauncherV0 struct {
	Runtime *orquestaruntime.RuntimeFakeLifecycleV0
}

func NewFakeLifecycleAgentLauncherV0() *FakeLifecycleAgentLauncherV0 {
	return &FakeLifecycleAgentLauncherV0{
		Runtime: orquestaruntime.NewRuntimeFakeLifecycleV0(),
	}
}

func (launcher *FakeLifecycleAgentLauncherV0) LaunchAgentV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (AgentLaunchResultV0, error) {
	if err := ctx.Err(); err != nil {
		return AgentLaunchResultV0{}, err
	}
	launcher.ensureRuntimeV0()
	snapshot, err := launcher.Runtime.LaunchAgentV0(inbound)
	if err != nil {
		return AgentLaunchResultV0{}, err
	}
	return AgentLaunchResultV0{
		AgentRequestID: snapshot.AgentRequestID,
		LaunchRef:      snapshot.LaunchRef,
		AckRef:         "ack-ref-" + snapshot.AgentRequestID,
		ReadinessRef:   "readiness-ref-" + snapshot.AgentRequestID,
		EvidenceRefs:   []string{"evidence-ref-launch-" + snapshot.AgentRequestID},
	}, nil
}

func (launcher *FakeLifecycleAgentLauncherV0) ensureRuntimeV0() {
	if launcher.Runtime == nil {
		launcher.Runtime = orquestaruntime.NewRuntimeFakeLifecycleV0()
	}
}
