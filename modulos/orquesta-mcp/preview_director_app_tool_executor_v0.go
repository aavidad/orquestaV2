package orquestamcp

import (
	"context"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
)

type MCPPreviewDirectorAppToolExecutorV0 struct{}

func NewMCPPreviewDirectorAppToolExecutorV0() MCPPreviewDirectorAppToolExecutorV0 {
	return MCPPreviewDirectorAppToolExecutorV0{}
}

func (executor MCPPreviewDirectorAppToolExecutorV0) Execute(
	ctx context.Context,
	input MCPArrancarDirectorAppToolInputV0,
) (MCPPreviewDirectorAppToolResultV0, error) {
	preview, err := orquestaappdirectorservice.BuildStartAppDirectorGoalWorkPreviewV0(
		ctx,
		ToStartAppDirectorRequestV0(input),
	)
	if err != nil {
		return MCPPreviewDirectorAppToolResultV0{}, err
	}
	return NewMCPPreviewDirectorAppResultV0(preview), nil
}
