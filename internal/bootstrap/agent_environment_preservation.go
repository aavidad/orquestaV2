package bootstrap

import "orquesta/internal/application"

func composeAgentEnvironmentPreservationBuilder(
	configured application.AgentEnvironmentPreservationBuilder,
	artifacts application.ArtifactStore,
) (application.AgentEnvironmentPreservationBuilder, error) {
	if configured != nil {
		return configured, nil
	}
	return application.NewAgentEnvironmentPreservationBuilder(artifacts)
}
