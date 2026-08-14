package application

import (
	"context"
	"errors"
	"testing"

	"orquesta/internal/ports"
)

type stopRecoveryControllerStub struct{ unsupportedAgentController }

func (*stopRecoveryControllerStub) ReconcileStop(
	context.Context,
	ports.AgentStopRequest,
) (ports.AgentStopReceipt, error) {
	return ports.AgentStopReceipt{}, nil
}

func TestAgentStopReconcilerFromRequiresExplicitReadOnlyCapability(t *testing.T) {
	capable := &stopRecoveryControllerStub{}
	reconciler, err := AgentStopReconcilerFrom(capable)
	if err != nil || reconciler != capable {
		t.Fatalf("reconciler=%T err=%v", reconciler, err)
	}
	if reconciler, err = AgentStopReconcilerFrom(unsupportedAgentController{}); reconciler != nil ||
		!errors.Is(err, ErrAgentStopRecoveryUnsupported) {
		t.Fatalf("ordinary controller reconciler=%T err=%v", reconciler, err)
	}
	var typedNil *stopRecoveryControllerStub
	if reconciler, err = AgentStopReconcilerFrom(typedNil); reconciler != nil ||
		!errors.Is(err, ErrAgentStopRecoveryUnsupported) {
		t.Fatalf("typed nil reconciler=%T err=%v", reconciler, err)
	}
	var absent AgentController
	if reconciler, err = AgentStopReconcilerFrom(absent); reconciler != nil ||
		!errors.Is(err, ErrAgentStopRecoveryUnsupported) {
		t.Fatalf("absent controller reconciler=%T err=%v", reconciler, err)
	}
}
