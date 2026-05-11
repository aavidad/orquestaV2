package orquestaruntime

import "context"

type ExternalAgentProcessLaunchStatusV0 string

const (
	ExternalAgentProcessLaunchBlockedV0 ExternalAgentProcessLaunchStatusV0 = "blocked"
	ExternalAgentProcessLaunchStartedV0 ExternalAgentProcessLaunchStatusV0 = "started"
)

const (
	ExternalAgentCommandResolutionInvalidaV0 ExternalAgentConnectorErrorCodeV0 = "external_agent_command_resolution_invalida"
	ExternalAgentResolverUnavailableV0       ExternalAgentConnectorErrorCodeV0 = "external_agent_command_resolver_unavailable"
	ExternalAgentRuntimeUnavailableV0        ExternalAgentConnectorErrorCodeV0 = "external_agent_runtime_unavailable"
	ExternalAgentRuntimeLaunchFailedV0       ExternalAgentConnectorErrorCodeV0 = "external_agent_runtime_launch_failed"
)

type ExternalAgentProcessCommandResolverV0 interface {
	ResolveExternalAgentProcessCommandV0(
		context.Context,
		ExternalAgentLaunchSpecV0,
	) (ProcessRuntimeLaunchRequestV0, []ExternalAgentConnectorErrorV0)
}

type ExternalAgentProcessRuntimePortV0 interface {
	LaunchV0(context.Context, ProcessRuntimeLaunchRequestV0) (ProcessRuntimeSnapshotV0, error)
}

type ExternalAgentProcessLaunchResultV0 struct {
	Status   ExternalAgentProcessLaunchStatusV0 `json:"status"`
	Snapshot ProcessRuntimeSnapshotV0           `json:"snapshot,omitempty"`
	Issues   []ExternalAgentConnectorErrorV0    `json:"issues,omitempty"`
}
