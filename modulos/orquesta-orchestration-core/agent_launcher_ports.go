package orquestacionnucleoapp

import (
	"context"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type AgentLauncherPortV0 interface {
	LaunchAgentV0(
		ctx context.Context,
		inbound orquestaruntime.AgentLauncherInboundV0,
	) (AgentLaunchResultV0, error)
}

type AgentLaunchResultV0 struct {
	AgentRequestID string
	LaunchRef      string
	AckRef         string
	ReadinessRef   string
	EvidenceRefs   []string
}
