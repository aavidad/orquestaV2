package orquestacionnucleoapp

import (
	"context"

	orquestacontext "orquesta/modulos/orquesta-context"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type AgentLauncherDependenciesResolverPortV0 interface {
	ResolveAgentLauncherDependenciesV0(
		context.Context,
		orquestaruntime.AgentLauncherInboundV0,
	) (orquestaruntime.AgentLauncherResolvedDependenciesV0, error)
}

type RuntimeContextMaterializerPortV0 interface {
	MaterializeRuntimeContextV0(
		context.Context,
		orquestaruntime.RuntimeLaunchRequestV0,
	) (orquestacontext.ContextMaterializedBundleV0, error)
}

type ExternalAgentConnectorResolverPortV0 interface {
	ResolveExternalAgentConnectorV0(
		context.Context,
		orquestaruntime.RuntimeLaunchRequestV0,
	) (ExternalAgentConnectorResolutionV0, error)
}

type ExternalAgentConnectorResolutionV0 struct {
	Profile         orquestaruntime.ExternalAgentConnectorProfileV0
	CommandResolver orquestaruntime.ExternalAgentProcessCommandResolverV0
}
