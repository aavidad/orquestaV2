package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type ExternalAgentLaunchSpecResolverV0 struct {
	Dependencies        AgentLauncherDependenciesResolverPortV0
	ContextMaterializer RuntimeContextMaterializerPortV0
	Connector           ExternalAgentConnectorResolverPortV0
	Options             orquestaruntime.AgentLauncherRuntimeLaunchOptionsV0
}

var _ ExternalAgentLaunchSpecResolverPortV0 = ExternalAgentLaunchSpecResolverV0{}

func (resolver ExternalAgentLaunchSpecResolverV0) ResolveExternalAgentLaunchSpecV0(
	ctx context.Context,
	inbound orquestaruntime.AgentLauncherInboundV0,
) (ExternalAgentLaunchSpecResolutionV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ExternalAgentLaunchSpecResolutionV0{}, err
	}
	if err := resolver.validatePortsV0(); err != nil {
		return ExternalAgentLaunchSpecResolutionV0{}, err
	}
	dependencies, err := resolver.Dependencies.ResolveAgentLauncherDependenciesV0(ctx, inbound)
	if err != nil {
		return ExternalAgentLaunchSpecResolutionV0{}, wrapSpecResolverErrorV0("agent_launcher_dependencies", err)
	}
	request, issues := orquestaruntime.LaunchRuntimeAgentToRuntimeLaunchRequestV0(
		inbound,
		dependencies,
		resolver.Options,
	)
	if len(issues) > 0 {
		return ExternalAgentLaunchSpecResolutionV0{}, errorFromAgentLauncherIssueV0(issues[0])
	}
	materialized, err := resolver.ContextMaterializer.MaterializeRuntimeContextV0(ctx, request)
	if err != nil {
		return ExternalAgentLaunchSpecResolutionV0{}, wrapSpecResolverErrorV0("context_materializer", err)
	}
	packet := orquestaruntime.BuildAgentStartPacketV0(request, materialized)
	if !packet.Valid() {
		return ExternalAgentLaunchSpecResolutionV0{}, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"agent_start_packet",
			"agent_start_packet_invalido",
		)
	}
	connector, err := resolver.Connector.ResolveExternalAgentConnectorV0(ctx, request)
	if err != nil {
		return ExternalAgentLaunchSpecResolutionV0{}, wrapSpecResolverErrorV0("external_agent_connector", err)
	}
	if connector.CommandResolver == nil {
		return ExternalAgentLaunchSpecResolutionV0{}, errorV0(
			ErrNucleoOrquestacionInvalidoV0,
			"external_agent_command_resolver",
			"external_agent_command_resolver requerido",
		)
	}
	spec := orquestaruntime.BuildExternalAgentLaunchSpecV0(request, packet, connector.Profile)
	if !spec.Valid() {
		return ExternalAgentLaunchSpecResolutionV0{}, errorFromExternalAgentIssuesV0(spec.Issues)
	}
	return ExternalAgentLaunchSpecResolutionV0{
		Spec:            spec,
		CommandResolver: orquestaruntime.NewExternalAgentProcessCommandResolverWithReceiptV0(connector.CommandResolver),
	}, nil
}

func (resolver ExternalAgentLaunchSpecResolverV0) validatePortsV0() error {
	switch {
	case resolver.Dependencies == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "agent_launcher_dependencies", "agent_launcher_dependencies requerido")
	case resolver.ContextMaterializer == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "context_materializer", "context_materializer requerido")
	case resolver.Connector == nil:
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "external_agent_connector", "external_agent_connector requerido")
	default:
		return nil
	}
}

func wrapSpecResolverErrorV0(field string, err error) error {
	if err == nil {
		return nil
	}
	return errorV0(ErrNucleoOrquestacionInvalidoV0, field, err.Error())
}

func errorFromAgentLauncherIssueV0(issue orquestaruntime.AgentLauncherInboundErrorV0) error {
	field := strings.TrimSpace(issue.Field)
	if field == "" {
		field = "agent_launcher"
	}
	return errorV0(ErrNucleoOrquestacionInvalidoV0, field, string(issue.Code))
}

func errorFromExternalAgentIssuesV0(issues []orquestaruntime.ExternalAgentConnectorErrorV0) error {
	if len(issues) == 0 {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "external_agent_launch_spec", "external_agent_launch_spec_invalido")
	}
	issue := issues[0]
	field := strings.TrimSpace(issue.Field)
	if field == "" {
		field = "external_agent_launch_spec"
	}
	return errorV0(ErrNucleoOrquestacionInvalidoV0, field, string(issue.Code))
}
