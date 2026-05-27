package orquestamcp

import (
	"errors"
	"testing"
)

func TestRegisterMCPTransportV0PropagaErrorDeRegistroResource(t *testing.T) {
	want := errors.New("mcp_duplicate_resource name=orquesta.contracts.shared.v0")
	transport := &failingMCPTransportRegistrationV0{resourceErr: want}

	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{})
	if !errors.Is(err, want) {
		t.Fatalf("err=%v want=%v", err, want)
	}
	if transport.resources != 1 || transport.tools != 0 {
		t.Fatalf("registro continuo tras error: resources=%d tools=%d", transport.resources, transport.tools)
	}
}

func TestRegisterMCPTransportV0PropagaErrorDeRegistroTool(t *testing.T) {
	want := errors.New("mcp_duplicate_tool name=orquesta.status.v0")
	transport := &failingMCPTransportRegistrationV0{toolErr: want}

	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{})
	if !errors.Is(err, want) {
		t.Fatalf("err=%v want=%v", err, want)
	}
	if transport.resources != len(MCPTransportResourcesV0()) || transport.tools != 1 {
		t.Fatalf("registro inesperado: resources=%d tools=%d", transport.resources, transport.tools)
	}
}

type failingMCPTransportRegistrationV0 struct {
	resources   int
	tools       int
	resourceErr error
	toolErr     error
}

func (f *failingMCPTransportRegistrationV0) RegisterResourceV0(MCPTransportResourceEnvelopeV0) error {
	f.resources++
	return f.resourceErr
}

func (f *failingMCPTransportRegistrationV0) RegisterToolV0(MCPTransportToolEnvelopeV0) error {
	f.tools++
	return f.toolErr
}
