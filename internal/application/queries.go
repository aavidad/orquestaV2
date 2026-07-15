package application

import (
	"context"
	"errors"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func (orchestrator *Orchestrator) GetGoal(ctx context.Context, access Access, goalRef goal.GoalRef) (GoalRecord, error) {
	if orchestrator == nil {
		return GoalRecord{}, errors.New("application.unavailable")
	}
	_, projectRef, err := access.values()
	if err != nil || goalRef.String() == "" {
		return GoalRecord{}, errors.New("application.query_invalid")
	}
	if _, err = orchestrator.authorizeRead(ctx, access, identity.PermissionGoalsGet, goalRef.String()); err != nil {
		return GoalRecord{}, err
	}
	record, err := orchestrator.state.GetGoal(ctx, goalRef)
	if err != nil {
		return GoalRecord{}, err
	}
	if record.Goal.Project() != projectRef {
		return GoalRecord{}, &StateError{Code: StateNotFound}
	}
	return record, nil
}

func (orchestrator *Orchestrator) ListGoals(
	ctx context.Context,
	access Access,
	limit int,
) ([]GoalSummary, error) {
	if orchestrator == nil {
		return nil, errors.New("application.unavailable")
	}
	_, projectRef, err := access.values()
	if err != nil || limit <= 0 {
		return nil, errors.New("application.query_invalid")
	}
	if _, err = orchestrator.authorizeRead(ctx, access, identity.PermissionGoalsList, projectRef.String()); err != nil {
		return nil, err
	}
	summaries, err := orchestrator.state.ListGoals(ctx, projectRef, limit)
	if err != nil {
		return nil, err
	}
	for _, summary := range summaries {
		if summary.ProjectRef != projectRef {
			return nil, &StateError{Code: StateConflict}
		}
	}
	return summaries, nil
}

func (orchestrator *Orchestrator) Status(ctx context.Context, access Access) (RepositoryStatus, error) {
	if orchestrator == nil {
		return RepositoryStatus{}, errors.New("application.unavailable")
	}
	_, projectRef, err := access.values()
	if err != nil {
		return RepositoryStatus{}, err
	}
	if _, err = orchestrator.authorizeRead(ctx, access, identity.PermissionProjectStatus, projectRef.String()); err != nil {
		return RepositoryStatus{}, err
	}
	return orchestrator.state.Status(ctx, projectRef)
}

func (orchestrator *Orchestrator) GetArtifact(
	ctx context.Context,
	access Access,
	goalRef goal.GoalRef,
	artifactRef goal.ArtifactRef,
) (ports.ArtifactContent, error) {
	if orchestrator == nil {
		return ports.ArtifactContent{}, errors.New("application.unavailable")
	}
	_, projectRef, err := access.values()
	if err != nil || goalRef.String() == "" || artifactRef.String() == "" {
		return ports.ArtifactContent{}, errors.New("application.query_invalid")
	}
	if _, err = orchestrator.authorizeRead(ctx, access, identity.PermissionArtifactsRead, artifactRef.String()); err != nil {
		return ports.ArtifactContent{}, err
	}
	record, err := orchestrator.state.GetGoal(ctx, goalRef)
	if err != nil {
		return ports.ArtifactContent{}, err
	}
	if record.Goal.Project() != projectRef {
		return ports.ArtifactContent{}, &StateError{Code: StateNotFound}
	}
	var stored ports.StoredArtifact
	for _, artifact := range record.Artifacts {
		if artifact.Stored.Ref == artifactRef {
			stored = artifact.Stored
			break
		}
	}
	if stored.Ref.String() == "" {
		return ports.ArtifactContent{}, &StateError{Code: StateNotFound}
	}
	content, err := orchestrator.artifacts.Get(ctx, artifactRef, stored.Size)
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
