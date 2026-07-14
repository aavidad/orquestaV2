package application

import (
	"context"
	"errors"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type GoalQuery struct {
	ActorRef   goal.ActorRef
	ProjectRef goal.ProjectRef
	GoalRef    goal.GoalRef
}

func (orchestrator *Orchestrator) GetGoal(ctx context.Context, query GoalQuery) (GoalRecord, error) {
	if orchestrator == nil {
		return GoalRecord{}, errors.New("application.unavailable")
	}
	record, err := orchestrator.state.GetGoal(ctx, query.GoalRef)
	if err != nil {
		return GoalRecord{}, err
	}
	if record.Goal.Actor() != query.ActorRef || record.Goal.Project() != query.ProjectRef {
		return GoalRecord{}, &StateError{Code: StateNotFound}
	}
	return record, nil
}

func (orchestrator *Orchestrator) ListGoals(
	ctx context.Context,
	actorRef goal.ActorRef,
	projectRef goal.ProjectRef,
	limit int,
) ([]GoalSummary, error) {
	if orchestrator == nil {
		return nil, errors.New("application.unavailable")
	}
	if actorRef.String() == "" || projectRef.String() == "" || limit <= 0 {
		return nil, errors.New("application.query_invalid")
	}
	summaries, err := orchestrator.state.ListGoals(ctx, actorRef, projectRef, limit)
	if err != nil {
		return nil, err
	}
	return summaries, nil
}

func (orchestrator *Orchestrator) Status(ctx context.Context) (RepositoryStatus, error) {
	if orchestrator == nil {
		return RepositoryStatus{}, errors.New("application.unavailable")
	}
	return orchestrator.state.Status(ctx)
}

type ArtifactQuery struct {
	ActorRef    goal.ActorRef
	ProjectRef  goal.ProjectRef
	GoalRef     goal.GoalRef
	ArtifactRef goal.ArtifactRef
}

func (orchestrator *Orchestrator) GetArtifact(ctx context.Context, query ArtifactQuery) (ports.ArtifactContent, error) {
	record, err := orchestrator.GetGoal(ctx, GoalQuery{
		ActorRef: query.ActorRef, ProjectRef: query.ProjectRef, GoalRef: query.GoalRef,
	})
	if err != nil {
		return ports.ArtifactContent{}, err
	}
	var stored ports.StoredArtifact
	for _, artifact := range record.Artifacts {
		if artifact.Stored.Ref == query.ArtifactRef {
			stored = artifact.Stored
			break
		}
	}
	if stored.Ref.String() == "" {
		return ports.ArtifactContent{}, &StateError{Code: StateNotFound}
	}
	content, err := orchestrator.artifacts.Get(ctx, query.ArtifactRef, stored.Size)
	if err != nil {
		return ports.ArtifactContent{}, err
	}
	if err := ports.ValidateArtifactContent(content); err != nil || content.Ref != stored.Ref ||
		content.Digest != stored.Digest || content.Size != stored.Size {
		return ports.ArtifactContent{}, errors.New("artifact.content_contract_invalid")
	}
	content.MediaType = stored.MediaType
	return content, nil
}
