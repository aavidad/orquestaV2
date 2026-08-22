package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"orquesta/internal/ports"
)

type historicalPreserverProbe struct {
	calls     int
	authority ports.AgentHistoricalRuntimeAuthority
	request   ports.AgentPreserveRequest
}

func (probe *historicalPreserverProbe) PreserveWithAuthority(
	_ context.Context,
	authority ports.AgentHistoricalRuntimeAuthority,
	request ports.AgentPreserveRequest,
) (ports.AgentPreserveReceipt, error) {
	probe.calls++
	probe.authority = authority
	probe.request = request
	return ports.AgentPreserveReceipt{}, errors.New("physical.preserve_marker")
}

func TestPreserveAgentEnvironmentResolvesExactHistoricalAuthorityBeforePhysicalCall(t *testing.T) {
	expected := applicationHistoricalRuntimeAuthorityFixture(t, "a", 11)
	request := applicationHistoricalPreserveRequest(t, expected)
	probe := &historicalPreserverProbe{}

	_, err := PreserveAgentEnvironmentWithHistoricalAuthority(
		context.Background(),
		&historicalRuntimeAuthorityResolverStub{authority: expected},
		expected.Key,
		request,
		probe,
	)
	if err == nil || err.Error() != "physical.preserve_marker" || probe.calls != 1 ||
		probe.authority != expected || probe.request != request {
		t.Fatalf("calls=%d authority=%+v request=%+v err=%v", probe.calls, probe.authority, probe.request, err)
	}
}

func TestPreserveAgentEnvironmentRejectsCrossedAuthorityWithoutPhysicalCall(t *testing.T) {
	expected := applicationHistoricalRuntimeAuthorityFixture(t, "a", 11)
	request := applicationHistoricalPreserveRequest(t, expected)
	mutations := map[string]func(*ports.AgentHistoricalRuntimeAuthority){
		"subject": func(value *ports.AgentHistoricalRuntimeAuthority) {
			value.Subject.AgentRef = "agent:crossed"
		},
		"key": func(value *ports.AgentHistoricalRuntimeAuthority) {
			value.Key.ActionFence++
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			crossed := mutateHistoricalAuthority(expected, mutate)
			probe := &historicalPreserverProbe{}
			receipt, err := PreserveAgentEnvironmentWithHistoricalAuthority(
				context.Background(), &historicalRuntimeAuthorityResolverStub{authority: crossed}, expected.Key, request, probe,
			)
			if err == nil ||
				!reflect.DeepEqual(receipt, ports.AgentPreserveReceipt{}) || probe.calls != 0 {
				t.Fatalf("receipt=%+v calls=%d err=%v", receipt, probe.calls, err)
			}
		})
	}
}

func applicationHistoricalPreserveRequest(
	t *testing.T,
	authority ports.AgentHistoricalRuntimeAuthority,
) ports.AgentPreserveRequest {
	t.Helper()
	physical, err := ports.NewAgentPhysicalToken(authority.Subject.ExternalRef)
	if err != nil {
		t.Fatal(err)
	}
	revision, err := ports.NewAgentPhysicalRevision("revision:b12-preserve")
	if err != nil {
		t.Fatal(err)
	}
	fence, err := ports.NewAgentPhysicalFence("fence:b12-preserve")
	if err != nil {
		t.Fatal(err)
	}
	return ports.AgentPreserveRequest{
		Subject: authority.Subject,
		ExpectedToken: ports.AgentEnvironmentLifecycleToken{
			PhysicalToken: physical,
			Revision:      revision,
			Fence:         fence,
			State:         ports.AgentEnvironmentQuiesced,
		},
		IdempotencyKey: "preserve:b12:exact",
	}
}
