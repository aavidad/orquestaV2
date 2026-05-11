package orquestacionnucleoapp

import (
	"context"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type AgentStopperPortV0 interface {
	StopAgentV0(
		ctx context.Context,
		inbound orquestaruntime.AgentStopperInboundV0,
	) (AgentStopResultV0, error)
}

type AgentStopResultV0 struct {
	AgentRequestID  string
	ConfirmationRef string
	EvidenceRefs    []string
}

type FakeLifecycleAgentStopperV0 struct {
	Runtime *orquestaruntime.RuntimeFakeLifecycleV0
}

func NewFakeLifecycleAgentStopperV0() *FakeLifecycleAgentStopperV0 {
	return &FakeLifecycleAgentStopperV0{
		Runtime: orquestaruntime.NewRuntimeFakeLifecycleV0(),
	}
}

func (stopper *FakeLifecycleAgentStopperV0) StopAgentV0(
	ctx context.Context,
	inbound orquestaruntime.AgentStopperInboundV0,
) (AgentStopResultV0, error) {
	if err := ctx.Err(); err != nil {
		return AgentStopResultV0{}, err
	}
	stopper.ensureRuntimeV0()
	snapshot, err := stopper.Runtime.StopAgentV0(inbound)
	if err != nil {
		return AgentStopResultV0{}, err
	}
	return AgentStopResultV0{
		AgentRequestID:  snapshot.AgentRequestID,
		ConfirmationRef: snapshot.StopRef,
		EvidenceRefs: []string{
			"evidence-ref-stop-" + snapshot.AgentRequestID,
			snapshot.StopRef,
		},
	}, nil
}

func (stopper *FakeLifecycleAgentStopperV0) ensureRuntimeV0() {
	if stopper.Runtime == nil {
		stopper.Runtime = orquestaruntime.NewRuntimeFakeLifecycleV0()
	}
}
