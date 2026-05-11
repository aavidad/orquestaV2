package orquestaruntime

import (
	"context"
	"testing"
)

func TestExternalAgentProcessAdapterV0LanzaProcesoOptInConResolverInyectado(t *testing.T) {
	spec := externalAgentLaunchSpecValidaV0(t)
	req := processRuntimeLaunchRequestForTestV0(t, "exit")
	connector := NewProcessRuntimeConnectorV0()

	result := LaunchExternalAgentProcessV0(
		context.Background(),
		spec,
		fakeExternalAgentProcessResolverV0{req: req},
		connector,
	)

	if result.Status != ExternalAgentProcessLaunchStartedV0 {
		t.Fatalf("status=%q issues=%+v", result.Status, result.Issues)
	}
	if result.Snapshot.ProcessRef == "" || result.Snapshot.LaunchRef == "" {
		t.Fatalf("snapshot sin refs publicas: %+v", result.Snapshot)
	}
	assertProcessRuntimeSnapshotDoesNotLeakV0(t, result.Snapshot, req)
	waitForProcessRuntimeStatusV0(t, connector, result.Snapshot.ProcessRef, ProcessRuntimeStoppedV0)
}

func TestExternalAgentProcessAdapterV0BloqueaSpecInvalidaAntesDeResolver(t *testing.T) {
	spec := externalAgentLaunchSpecValidaV0(t)
	spec.Security.OptIn = false

	result := LaunchExternalAgentProcessV0(
		context.Background(),
		spec,
		fakeExternalAgentProcessResolverV0{},
		NewProcessRuntimeConnectorV0(),
	)

	requireExternalAgentProcessStatusV0(t, result, ExternalAgentProcessLaunchBlockedV0)
	requireExternalAgentResultCodeV0(t, result, ExternalAgentSecurityInvalidaV0)
}

func TestExternalAgentProcessAdapterV0ExigePuertosInyectados(t *testing.T) {
	spec := externalAgentLaunchSpecValidaV0(t)

	result := LaunchExternalAgentProcessV0(context.Background(), spec, nil, NewProcessRuntimeConnectorV0())
	requireExternalAgentProcessStatusV0(t, result, ExternalAgentProcessLaunchBlockedV0)
	requireExternalAgentResultCodeV0(t, result, ExternalAgentResolverUnavailableV0)

	result = LaunchExternalAgentProcessV0(context.Background(), spec, fakeExternalAgentProcessResolverV0{}, nil)
	requireExternalAgentProcessStatusV0(t, result, ExternalAgentProcessLaunchBlockedV0)
	requireExternalAgentResultCodeV0(t, result, ExternalAgentRuntimeUnavailableV0)
}

func TestExternalAgentProcessAdapterV0BloqueaIssuesDelResolver(t *testing.T) {
	spec := externalAgentLaunchSpecValidaV0(t)
	result := LaunchExternalAgentProcessV0(
		context.Background(),
		spec,
		fakeExternalAgentProcessResolverV0{
			issues: []ExternalAgentConnectorErrorV0{
				externalAgentProcessErrorV0(
					ExternalAgentCommandResolutionInvalidaV0,
					"command_ref",
					"",
					false,
				),
			},
		},
		NewProcessRuntimeConnectorV0(),
	)

	requireExternalAgentProcessStatusV0(t, result, ExternalAgentProcessLaunchBlockedV0)
	requireExternalAgentResultCodeV0(t, result, ExternalAgentCommandResolutionInvalidaV0)
	if result.Issues[0].CorrelationID != spec.CorrelationID {
		t.Fatalf("correlation_id no normalizada: %+v", result.Issues[0])
	}
}

func TestExternalAgentProcessAdapterV0BloqueaRequestOperacionalInvalida(t *testing.T) {
	spec := externalAgentLaunchSpecValidaV0(t)
	req := processRuntimeLaunchRequestForTestV0(t, "exit")
	req.Env = nil

	result := LaunchExternalAgentProcessV0(
		context.Background(),
		spec,
		fakeExternalAgentProcessResolverV0{req: req},
		NewProcessRuntimeConnectorV0(),
	)

	requireExternalAgentProcessStatusV0(t, result, ExternalAgentProcessLaunchBlockedV0)
	requireExternalAgentResultCodeV0(t, result, ExternalAgentCommandResolutionInvalidaV0)
}

func TestExternalAgentProcessAdapterV0MapeaFalloRuntime(t *testing.T) {
	spec := externalAgentLaunchSpecValidaV0(t)
	req := processRuntimeLaunchRequestForTestV0(t, "exit")

	result := LaunchExternalAgentProcessV0(
		context.Background(),
		spec,
		fakeExternalAgentProcessResolverV0{req: req},
		fakeExternalAgentProcessRuntimeV0{
			err: processRuntimeErrorV0(ProcessRuntimeLaunchFallidoV0, "command_path"),
		},
	)

	requireExternalAgentProcessStatusV0(t, result, ExternalAgentProcessLaunchBlockedV0)
	requireExternalAgentResultCodeV0(t, result, ExternalAgentRuntimeLaunchFailedV0)
	if len(result.Issues[0].Evidence) == 0 {
		t.Fatalf("falta evidencia runtime: %+v", result.Issues[0])
	}
}

type fakeExternalAgentProcessResolverV0 struct {
	req    ProcessRuntimeLaunchRequestV0
	issues []ExternalAgentConnectorErrorV0
}

func (f fakeExternalAgentProcessResolverV0) ResolveExternalAgentProcessCommandV0(
	context.Context,
	ExternalAgentLaunchSpecV0,
) (ProcessRuntimeLaunchRequestV0, []ExternalAgentConnectorErrorV0) {
	return f.req, f.issues
}

type fakeExternalAgentProcessRuntimeV0 struct {
	snapshot ProcessRuntimeSnapshotV0
	err      error
}

func (f fakeExternalAgentProcessRuntimeV0) LaunchV0(
	context.Context,
	ProcessRuntimeLaunchRequestV0,
) (ProcessRuntimeSnapshotV0, error) {
	return f.snapshot, f.err
}

func externalAgentLaunchSpecValidaV0(t *testing.T) ExternalAgentLaunchSpecV0 {
	t.Helper()
	request := runtimeLaunchRequestValidaV0()
	materialized := runtimeMaterializedContextValidoV0(t, *request.ContextBundle)
	packet := BuildAgentStartPacketV0(request, materialized)
	spec := BuildExternalAgentLaunchSpecV0(request, packet, externalAgentConnectorProfileValidoV0())
	if !spec.Valid() {
		t.Fatalf("spec invalida: %+v", spec.Issues)
	}
	return spec
}

func requireExternalAgentProcessStatusV0(
	t *testing.T,
	result ExternalAgentProcessLaunchResultV0,
	status ExternalAgentProcessLaunchStatusV0,
) {
	t.Helper()
	if result.Status != status {
		t.Fatalf("status=%q, want %q result=%+v", result.Status, status, result)
	}
}

func requireExternalAgentResultCodeV0(
	t *testing.T,
	result ExternalAgentProcessLaunchResultV0,
	code ExternalAgentConnectorErrorCodeV0,
) {
	t.Helper()
	requireExternalAgentCodeV0(t, result.Issues, code)
}
